// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"strings"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/accolade/internal/collab"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Following another device (LOOP.md, TABLE READ / CAST, level 2): the
// Table Read follows a person here in the shared script (the session's
// presence, session_tab.go): the scene and the line they are at, their
// Table Read's if theirs is open, else their cursor's. So a phone or a
// laptop drives the TV.
//
// The follow link (the share link with &follow) joins as a follow-only
// viewer: it opens the Table Read, follows the first one here who is no
// viewer, and has no editing rights: nothing it does to the script, the
// source or the settings goes to the others, and an edit typed there is
// put back. Its presence says it is a viewer (collab.Presence.View), so
// nobody follows a TV.

// followFlag marks a share link as the follow-only viewer's.
const followFlag = "&follow"

// followLink is the follow-only viewer's link for key.
func followLink(k collab.Key) string { return shareLink(k) + followFlag }

// isFollowLink reports whether a link is a follow-only viewer's.
func isFollowLink(link string) bool { return strings.Contains(link, followFlag) }

// pageKey is what the page's address keeps of the session (#k=…), so a
// reload comes back to it the same way: a viewer's with its flag.
func (s *shared) pageKey() string {
	if s.followOnly {
		return s.key.String() + followFlag
	}
	return s.key.String()
}

// followsOnly reports whether the window is a follow-only viewer.
func (w *MainWindow) followsOnly() bool { return w.shared != nil && w.shared.followOnly }

// hereLine is the line this window is at for the others: its Table
// Read's, while it is open and follows nobody, else the cursor's.
func (w *MainWindow) hereLine() int {
	if v := w.tableRead; v != nil && w.tableReadOpen() && v.follow == "" {
		if l := v.lineNow(); l >= 0 {
			return l
		}
	}
	return w.scriptCursorLine()
}

// sayHereNow tells the others the line this window is at, if it moved.
func (w *MainWindow) sayHereNow() { w.sayHere(false) }

// copyFollowLink is Copy the Follow Link: the script shared first if it
// is not, the viewer's link copied.
func (w *MainWindow) copyFollowLink() {
	if w.followsOnly() {
		w.setStatus("This window follows only: the follow link is the leader's to give")
		return
	}
	key := w.sharedKey()
	if w.shared == nil {
		return
	}
	fyne.CurrentApp().Clipboard().SetContent(followLink(key))
	w.setStatus("Follow link copied: open it on the TV's browser (or in Accolade) — it shows your Table Read, and edits nothing")
}

// startFollowOnly makes a joined window the follow-only viewer: its
// presence a viewer's, its Table Read open.
func (w *MainWindow) startFollowOnly() {
	s := w.shared
	s.followOnly = true
	s.session.Viewer = true
	w.sayHere(true)
	w.openTableRead()
	w.tableReadFollowHere()
	w.setStatus("Following only: the Table Read follows whoever leads; nothing here is sent to the others")
}

// putBackFollowerEdit takes an edit typed in a follow-only window back:
// the editor made the shared script's again, outside the undo history.
func (w *MainWindow) putBackFollowerEdit() {
	s := w.shared
	w.after(0, func() {
		if w.shared != s || w.textEditor == nil {
			return
		}
		var text string
		s.session.Do(func(d *collab.Doc) { text = d.Text() })
		edits := collab.Edits(w.textEditor.Text(), text)
		if len(edits) == 0 {
			return
		}
		w.textEditor.Edit(func(b *buffer.Buffer) {
			for i := len(edits) - 1; i >= 0; i-- {
				e := edits[i]
				b.Remote(e.Pos, e.Pos+e.Del, e.Text)
			}
		})
		w.setStatus("Following only: this window does not edit the script")
	})
}

// tableReadFollowHere follows who is here in the Table Read: the one
// followed gone, it stops; their line moved, it goes there. A follow-only
// viewer following nobody takes the first one here who leads.
func (w *MainWindow) tableReadFollowHere() {
	v := w.tableRead
	if v == nil || w.shared == nil {
		if v != nil && v.follow != "" {
			v.stopFollow()
		}
		return
	}
	here := w.shared.session.Here()
	if v.follow == "" && w.followsOnly() {
		for _, p := range here {
			if !p.View {
				v.follow, v.followName = p.Replica, p.Name
				v.followLine = -1
				break
			}
		}
	}
	if v.follow != "" {
		var them *collab.Presence
		for _, p := range here {
			if p.Replica == v.follow {
				them = &p
			}
		}
		switch {
		case them == nil:
			w.setStatus(v.followName + " left: the Table Read follows nobody now")
			v.stopFollow()
			if w.followsOnly() {
				w.tableReadFollowHere() // another one leading, if any
			}
		case them.Line != v.followLine:
			v.followLine = them.Line
			v.goLine(them.Line)
		}
	}
	v.fillFollow()
}

// followReplica makes the Table Read follow replica ("": nobody).
func (v *tableReadView) followReplica(replica string) {
	w := v.w
	if replica == "" {
		v.stopFollow()
		return
	}
	if w.shared == nil {
		w.setStatus("The script is not shared: Collaboration > Edit Together… first")
		return
	}
	for _, p := range w.shared.session.Here() {
		if p.Replica == replica {
			v.follow, v.followName, v.followLine = p.Replica, p.Name, p.Line
			v.goLine(p.Line)
			v.fillFollow()
			w.setStatus("The Table Read follows " + p.Name)
			w.ribbonRefresh()
			return
		}
	}
	w.setStatus("They are not here any more")
}

// stopFollow: the Table Read follows nobody.
func (v *tableReadView) stopFollow() {
	v.follow, v.followName = "", ""
	v.fillFollow()
	v.estimate.SetText(v.estimateText())
	v.w.ribbonRefresh()
}

// fillFollow fills the panel's list of who to follow: Nobody, then each
// one here who leads.
func (v *tableReadView) fillFollow() {
	was := v.building
	v.building = true
	defer func() { v.building = was }()
	v.followIDs = []string{""}
	opts := []string{"Nobody"}
	sel := 0
	for _, p := range v.w.hereNow() {
		if p.View {
			continue
		}
		v.followIDs = append(v.followIDs, p.Replica)
		opts = append(opts, p.Name)
		if p.Replica == v.follow {
			sel = len(opts) - 1
		}
	}
	if v.follow != "" && sel == 0 { // not heard of yet
		v.followIDs = append(v.followIDs, v.follow)
		opts = append(opts, v.followName)
		sel = len(opts) - 1
	}
	v.follower.Options = opts
	v.follower.SetSelectedIndex(sel)
	if v.estimate != nil {
		v.estimate.SetText(v.estimateText())
	}
}

// followPicked follows the one picked in the panel.
func (v *tableReadView) followPicked(string) {
	if v.building {
		return
	}
	if i := v.follower.SelectedIndex(); i >= 0 && i < len(v.followIDs) && v.followIDs[i] != v.follow {
		v.followReplica(v.followIDs[i])
	}
}

// tableReadFollowMenu is the Table Read's Follow menu: Nobody, each one
// here who leads, the one followed checked.
func (w *MainWindow) tableReadFollowMenu() *fyne.Menu {
	v := w.tableReadView()
	nobody := fyne.NewMenuItem("Nobody", func() { v.followReplica("") })
	nobody.Checked = v.follow == ""
	items := []*fyne.MenuItem{nobody}
	for _, p := range w.hereNow() {
		if p.View {
			continue
		}
		item := fyne.NewMenuItem(p.Name, func() { v.followReplica(p.Replica) })
		item.Icon = peerDot(p.Replica)
		item.Checked = v.follow == p.Replica
		items = append(items, item)
	}
	return fyne.NewMenu("Follow", items...)
}
