// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LaPingvino/accolade/internal/collab"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The script's source shared while editing together (docs/COLLAB.md, named
// Docs): a source kept outside the script travels in a Doc of its own,
// which the shared setting "source" names. Only its text is shared, not
// where it is: each peer keeps it where its window keeps a source (in the
// bundle, a kept copy, the file beside the script), and a peer with none
// gets a copy kept for the room. The source document (document.go) is the
// Doc's editor: a key typed in it, in the main editor or the Source pane,
// goes to the peers, and theirs come into it outside its undo history, as
// the script's do. A source kept in the script travels in the script's
// Doc, and the setting is "".
//
// Only the peer who shares the script, or who picks a source later
// (Choose Source, Keep Source in the Script turned off), makes a source
// Doc: a peer joining, or taking up a kept state, takes the Doc as it is
// and never sets it from its own copy, so nothing is doubled. Two peers
// picking a source at once: the setting's last writer wins, the other Doc
// is let go on every peer (the winner hears of it as a setting that lost:
// Session.OnLost), and the one whose pick lost is told so.

// sourceMeta is the shared setting naming the source's Doc.
const sourceMeta = "source"

// sharedSource is the Doc the window's source is shared in ("" when it is
// not shared, or travels in the script).
func (w *MainWindow) sharedSource() string {
	if w.shared == nil || w.markdown {
		return ""
	}
	name := w.shared.session.Meta(sourceMeta)
	if name == "" {
		return ""
	}
	if o := w.sourceRef().Origin; o == sourceNone || o == sourceInside {
		return ""
	}
	return name
}

// keptSharedSource is where a peer without a source of its own keeps the
// shared one: a folder of its own among the kept sources (dropKeptCopy
// removes it whole), one per room.
func keptSharedSource(k collab.Key) string {
	r := k.Room()
	return filepath.Join(keptSourcesDir(), "shared-"+hex.EncodeToString(r[:8]), "source.md")
}

// shareSourceAnew shares the window's source in a new Doc and names it:
// sharing the script, or another source picked while shared. A source
// kept in the script, or none, clears the setting.
func (w *MainWindow) shareSourceAnew() {
	s := w.shared
	if s == nil || !s.synced || w.markdown || s.followOnly { // a viewer shares nothing
		return
	}
	ref := w.sourceRef()
	if ref.Origin == sourceNone || ref.Origin == sourceInside {
		if s.session.Meta(sourceMeta) != "" {
			s.session.SetMeta(sourceMeta, "")
		}
		return
	}
	text, err := w.sourceText(ref)
	if err != nil || text == "" {
		return // an empty source is not shared: a Doc with nothing in it is one whose ops have not come (followShared)
	}
	name := "source:" + newReplica()
	var ops []collab.Op
	s.session.DoIn(name, func(d *collab.Doc) { ops = d.SetText(text) })
	s.session.LocalIn(name, ops)
	s.session.SetMeta(sourceMeta, name)
	s.madeSource = name
	w.dropOtherSources(name)
	w.bindSource()
	w.keepSharedState()
}

// adoptSource takes up the source a peer named (OnMeta, or a kept state):
// kept for the room if this window has none, then its document follows
// the Doc.
func (w *MainWindow) adoptSource(name string) {
	s := w.shared
	if s == nil || name == "" || w.markdown {
		w.updateStatusBar()
		return
	}
	w.dropOtherSources(name)
	if made := s.madeSource; made != "" && made != name {
		s.madeSource = ""
		w.lostSourcePick()
	}
	ref := w.sourceRef()
	if ref.Origin == sourceInside {
		return // the script holds one: that one is the script's
	}
	if ref.Origin == sourceNone {
		var text string
		s.session.DoIn(name, func(d *collab.Doc) { text = d.Text() })
		p := keptSharedSource(s.key)
		if _, err := os.Stat(p); err == nil && text == "" {
			// kept before, the Doc not here yet (a browser's state kept
			// without it): the copy until the peers send it
		} else if os.MkdirAll(filepath.Dir(p), 0o755) != nil || writeFileAtomic(p, []byte(text)) != nil {
			return
		}
		w.rememberSource(w.currentFile, p)
		if w.source != nil {
			w.source.modTime = time.Time{} // read again
		}
		w.ribbonRefresh()
	}
	w.bindSource()
}

// bindSource opens the source document of a shared source (it stays open
// while it is shared) and brings it up to the Doc. The pane reads its
// passages again once the peers' edits pause, as it follows this
// writer's own typing: reading a novel again is the whole pane anew (its
// passages, rows, links and guesses: ≈0.5 s on the novel), and a peer's
// message is one key (docs/COLLAB.md, step 8).
func (w *MainWindow) bindSource() {
	if w.sharedSource() == "" {
		return
	}
	d := w.sourceDoc()
	if d == nil {
		return
	}
	d.followShared()
	if p := w.source; p != nil && !p.editing {
		w.debounce(&p.peerRead, followPause, func() {
			if !p.editing {
				p.loadSource(true)
			}
		})
	}
	w.updateTitle()
	w.updateStatusBar()
}

// shareSourceEdit sends the source document's change to the peers.
func (w *MainWindow) shareSourceEdit(name, text string) {
	if w.shared.followOnly { // a viewer's edit stays its own (tableread_follow.go)
		return
	}
	var ops []collab.Op
	w.shared.session.DoIn(name, func(d *collab.Doc) {
		if !d.Empty() { // a Doc not come yet: the edit stays this peer's (followShared)
			ops = d.SetText(text)
		}
	})
	w.shared.session.LocalIn(name, ops)
	w.keepSharedState()
}

// followShared brings a source document shared in a Doc of its own up to
// the Doc: the peers' changes as edits outside its undo history, the
// cursor kept with its text. Its home has them once it is written (Save,
// the pane's Done), as for the writer's own.
func (d *document) followShared() {
	name := d.sharedAs()
	if name == "" || name == collab.Script {
		return
	}
	var text string
	empty := false
	d.w.shared.session.DoIn(name, func(x *collab.Doc) { text, empty = x.Text(), x.Empty() })
	if empty {
		return // not come from the peers yet
	}
	edits := collab.Edits(d.ed.Text(), text)
	if len(edits) == 0 {
		return
	}
	d.fromHome = true
	d.ed.Edit(func(b *buffer.Buffer) {
		for i := len(edits) - 1; i >= 0; i-- { // from the end: the positions before stay
			e := edits[i]
			b.Remote(e.Pos, e.Pos+e.Del, e.Text)
		}
	})
	d.fromHome = false
}

// dropOtherSources lets go of the source Docs the setting no longer names
// (another source picked, or two picked at once): not kept, nor sent to a
// peer joining.
func (w *MainWindow) dropOtherSources(name string) {
	for _, n := range w.shared.session.Names() {
		if n != name && strings.HasPrefix(n, "source:") {
			w.shared.session.Drop(n)
		}
	}
}

// keepSharedSource writes the source to the room's copy when editing
// pauses (keepSharedState), for a peer whose source is that copy: a
// reload, or the browser's state kept without the source, finds it as
// the Doc was. The sharer's own home is written on its Save, as ever.
func (w *MainWindow) keepSharedSource() {
	d := w.srcDoc
	if d == nil || w.shared == nil || d.ref.Path != keptSharedSource(w.shared.key) || d.sharedAs() == "" {
		return
	}
	if !d.dirty() {
		return
	}
	_ = d.store()
}

// lostSourcePick tells the writer that the source they shared or picked
// is not the shared one any more: another peer picked one at the same
// moment (or since), and the last pick wins. The window follows the
// shared one from now on; the file they picked is left as it is.
func (w *MainWindow) lostSourcePick() {
	who := "Another writer"
	if v := w.shared.session.MetaValue(sourceMeta); v.Replica != "" {
		if n := w.shared.session.Meta(nameKey(v.Replica)); n != "" {
			who = n
		}
	}
	w.setStatus(who + " picked a source too: theirs is the shared one now. Choose Source… again to share yours instead")
}
