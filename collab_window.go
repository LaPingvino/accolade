package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/collab"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Editing together: a script shared with a key (collab), its peers
// found through WebTorrent trackers and connected over WebRTC. The
// editor's changes become the document's ops; the peers' ops change
// the editor's text where they apply, keeping the cursor, and outside
// this window's undo history. docs/COLLAB.md has the design.

// webVersion is the published web version, the link's page from the desktop.
const webVersion = "https://lapingvino.github.io/accolade/"

// shareLink is the link for a key: the web version's page (in the
// browser this page itself, wherever it is served); the desktop app
// takes it too.
func shareLink(k collab.Key) string {
	base := webVersion
	if p := pageBase(); p != "" {
		base = p
	}
	return base + "#k=" + k.String()
}

// keptSharedDir is where a shared script's state is kept, one file per
// room: beside the kept sources (the data folder; the browser's kept
// settings folder).
func keptSharedDir() string { return filepath.Join(filepath.Dir(keptSourcesDir()), "shared") }

func keptSharedPath(k collab.Key) string {
	r := k.Room()
	return filepath.Join(keptSharedDir(), hex.EncodeToString(r[:8])+".json")
}

// ownMarkPath marks a room whose script is this writer's own (they
// shared it): the browser keeps it as their script too (keepsOwnDraft),
// also after a reload, which comes back through joining.
func ownMarkPath(k collab.Key) string { return strings.TrimSuffix(keptSharedPath(k), ".json") + ".own" }

// keepsOwnDraft: the browser keeps the window's script as the writer's
// own (draft.json) when it is not shared, or shared by them. A script
// joined is kept only as the session's state, so joining never
// overwrites the script kept before.
func keepsOwnDraft(s *shared) bool { return s == nil || s.own }

// maxKeptState is the most the browser keeps of a shared script's state:
// its local storage is a few megabytes for everything (web/memfs.js).
const maxKeptState = 1 << 20

// sharedStatePause is how long editing pauses before the shared script's
// state is kept.
const sharedStatePause = time.Second

// shared is a window's script shared with peers.
type shared struct {
	session *collab.Session
	network *collab.Network
	key     collab.Key
	stop    context.CancelFunc
	peers   int
	other   bool // a peer runs another version of Accolade
	// otherApp, otherOlder: that peer's Accolade version ("" if it did
	// not say), and whether its protocol is older than ours
	otherApp   string
	otherOlder bool
	stateTimer *uiTimer // keeping the state (keepSharedState)
	own        bool     // shared by this writer: their script too (keepsOwnDraft)
	// synced: the script's language and format are the peers' (joining,
	// once they sent them), so changes here go to them
	synced bool
	paused bool // let go of the peers while the page is hidden
	// who is here (session_tab.go): this side's replica; the line said
	// last; the beat saying it is still here; the one followed ("":
	// nobody), their name and line as last followed
	replica            string
	hereLine           int
	saidHere           bool
	beat               *uiTimer
	follow, followName string
	followLine         int
}

func newReplica() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// startSharing shares the window's script with the peers that have key;
// joining, the script comes from them (the window's text is replaced), or
// from the state kept when this window or page was in it before.
func (w *MainWindow) startSharing(key collab.Key, joining bool) {
	// the views follow at the end (state.go)
	defer w.scriptChange()()
	if joining { // the script is replaced: its documents first (document.go)
		w.leaveDocs()
	}
	w.stopSharing()
	var session *collab.Session
	if joining {
		if b, err := os.ReadFile(keptSharedPath(key)); err == nil {
			session, _ = collab.Restore(key, newReplica(), b) // unreadable: from the peers
		}
	}
	restored := session != nil
	if session == nil {
		doc := collab.New(newReplica())
		if !joining {
			doc.SetText(w.textEditor.Text())
		}
		session = collab.NewSession(key, doc)
	}
	s := &shared{session: session, key: key, own: !joining}
	if joining {
		_, err := os.Stat(ownMarkPath(key))
		s.own = err == nil
	}
	if inBrowser { // room for one shared script's state: this one
		if es, err := os.ReadDir(keptSharedDir()); err == nil {
			keep := strings.TrimSuffix(filepath.Base(keptSharedPath(key)), ".json")
			for _, e := range es {
				if strings.TrimSuffix(strings.TrimSuffix(e.Name(), ".json"), ".own") != keep {
					_ = os.Remove(filepath.Join(keptSharedDir(), e.Name()))
				}
			}
		}
	}
	if s.own && os.MkdirAll(keptSharedDir(), 0o755) == nil {
		_ = os.WriteFile(ownMarkPath(key), nil, 0o644)
	}
	s.session.App = appVersion()
	s.session.Run = uiDo // the document changes on the UI goroutine only
	s.session.OnRemote = func(doc, by string) {
		if w.shared != s {
			return
		}
		s.synced = true
		switch doc {
		case collab.Script:
			w.applyRemote(by)
		case w.sharedSource(): // the source, in a Doc of its own (collab_source.go)
			w.bindSource()
			w.keepSharedState()
		}
	}
	var replica string
	s.session.Do(func(d *collab.Doc) { replica = d.Replica() })
	// who this is, for the others' history of changes (changes.go)
	s.session.SetMeta(nameKey(replica), w.yourName())
	s.replica = replica
	s.session.OnHere = func() {
		if w.shared == s {
			w.hereChanged()
		}
	}
	s.session.OnMeta = func(key, value string) {
		if w.shared == s {
			s.synced = true
			w.applyShared(key, value)
			w.keepSharedState()
		}
	}
	s.session.OnMismatch = func(theirs int, app string) {
		if w.shared == s {
			s.other, s.otherApp, s.otherOlder = true, app, theirs < collab.Protocol
			w.updateStatusBar()
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.stop = cancel
	w.shared = s
	if joining {
		var text string
		s.session.Do(func(d *collab.Doc) { text = d.Text() })
		w.textEditor.SetText(text)
		w.filledWith = ""
		w.hasChanges = text != ""
	}
	if restored { // the peers' settings, as they were
		s.synced = true
		for _, k := range []string{"markdown", "language", "format", sourceMeta} {
			if v := s.session.Meta(k); v != "" {
				w.applyShared(k, v)
			}
		}
	}
	if !joining {
		s.synced = true
		w.shareMeta("language", w.language)
		w.shareMeta("format", w.scriptFormatKey())
		w.shareMeta("markdown", strconv.FormatBool(w.markdown))
		w.shareSourceAnew() // the source, if kept outside the script (collab_source.go)
	}
	setPageKey(key.String()) // a reload of the page comes back to it
	w.keepSharedState()
	w.sayHere(true) // who is here (session_tab.go)
	w.hereBeat(s)
	w.connect(ctx, s)
	w.updateStatusBar()
	w.ribbonRefresh() // the Session tab
}

// connect finds the script's peers until ctx ends.
func (w *MainWindow) connect(ctx context.Context, s *shared) {
	s.network = collab.Join(ctx, s.session, s.key, collab.Trackers, func(n int) {
		uiDo(func() {
			s.peers = n
			w.updateStatusBar()
		})
	})
}

// pauseSharing lets go of the peers while nobody looks (a hidden page),
// keeping the script shared; resumeSharing finds them again, and each side
// gets what the other changed meanwhile (the hello's catch-up).
func (w *MainWindow) pauseSharing() {
	if s := w.shared; s != nil && !s.paused {
		s.stop()
		s.paused, s.peers = true, 0
		w.updateStatusBar()
	}
}

func (w *MainWindow) resumeSharing() {
	if s := w.shared; s != nil && s.paused {
		ctx, cancel := context.WithCancel(context.Background())
		s.stop, s.paused = cancel, false
		w.sayHere(true)
		w.connect(ctx, s)
		w.updateStatusBar()
	}
}

// stopSharing leaves the peers (the script stays as it is); the state
// kept for a reload is kept too, for a window that is closed.
func (w *MainWindow) stopSharing() {
	if s := w.shared; s != nil {
		s.session.Leave() // off the others' list at once (session_tab.go)
		s.stop()
		s.stateTimer.Stop()
		s.beat.Stop()
		w.shared = nil
		setPageKey("")
		w.updateStatusBar()
		w.ribbonRefresh() // the Session tab goes
	}
}

// leaveSharing leaves the peers for good (Stop sharing, or another
// document in the window): the kept state goes too, and the script
// stays with the others.
func (w *MainWindow) leaveSharing() {
	// the views follow at the end (state.go)
	defer w.scriptChange()()
	if s := w.shared; s != nil {
		w.stopSharing()
		_ = os.Remove(keptSharedPath(s.key))
		_ = os.Remove(ownMarkPath(s.key))
		if s.own {
			w.keepDraft() // the browser keeps it as the writer's script again, at once
		}
	}
}

// newDocument is called before the window takes another document (Open,
// New, an import, a restore): a shared script stays with the peers, not
// replaced for them by this one.
func (w *MainWindow) newDocument() {
	w.fixupFrom = "" // another document: no import to fix up (contextual.go)
	w.leaveDocs()    // the script back in the main editor, its documents written (document.go)
	w.closeCompare() // its drafts are the script's going
	if w.shared != nil {
		w.leaveSharing()
		w.setStatus("Left the shared script: it goes on with the others")
	}
}

// shareEdit sends the editor's change to the peers.
func (w *MainWindow) shareEdit(text string) {
	if w.shared == nil {
		return
	}
	var ops []collab.Op
	w.shared.session.Do(func(d *collab.Doc) { ops = d.SetText(text) })
	w.shared.session.Local(ops)
	if len(ops) > 0 { // this side's own edit (a peer's makes none here)
		w.typedWhileFollowing()
	}
	w.keepSharedState()
}

// keepSharedState keeps the shared script's state when editing pauses:
// a reload (or Join with the same link) starts from it, with nobody else
// there too, and nothing is doubled when the peers come.
func (w *MainWindow) keepSharedState() {
	s := w.shared
	if s == nil {
		return
	}
	w.debounce(&s.stateTimer, sharedStatePause, func() {
		if w.shared != s {
			return
		}
		w.keepSharedSource() // the room's copy of the source, as the Doc is (collab_source.go)
		b, err := s.session.State()
		if err == nil && inBrowser && len(b) > maxKeptState { // a novel for a source: the script's alone
			b, err = s.session.StateOf(collab.Script)
		}
		if err != nil || inBrowser && len(b) > maxKeptState {
			return
		}
		if os.MkdirAll(keptSharedDir(), 0o755) == nil {
			_ = os.WriteFile(keptSharedPath(s.key), b, 0o644)
		}
	})
}

// applyRemote puts the peers' changes in the editor: edit by edit
// (collab.Edits), outside the undo history, so the cursor and selection
// stay with their text and Undo undoes only this window's own edits.
// Each edit goes in the history of changes, as replica by's.
func (w *MainWindow) applyRemote(by string) {
	if w.shared == nil {
		return
	}
	var text string
	w.shared.session.Do(func(d *collab.Doc) { text = d.Text() })
	old := w.textEditor.Text()
	edits := collab.Edits(old, text)
	if len(edits) == 0 {
		return
	}
	w.textEditor.Edit(func(b *buffer.Buffer) {
		for i := len(edits) - 1; i >= 0; i-- { // from the end: the positions before stay
			e := edits[i]
			b.Remote(e.Pos, e.Pos+e.Del, e.Text)
		}
	})
	rs := []rune(old)
	line, at := 0, 0 // the line in the new text
	for _, e := range edits {
		line += strings.Count(string(rs[at:e.Pos]), "\n")
		at = e.Pos + e.Del
		removed := string(rs[e.Pos:at])
		w.recordChange(by, removed, e.Text, line)
		line += strings.Count(e.Text, "\n")
	}
}

// shareMeta sends a setting of the script (its language, its format) to
// the peers, if it is shared and has theirs.
func (w *MainWindow) shareMeta(key, value string) {
	if w.shared != nil && w.shared.synced {
		w.shared.session.SetMeta(key, value)
	}
}

// applyShared takes a setting of the script from a peer.
func (w *MainWindow) applyShared(key, value string) {
	switch key {
	case "language":
		if value != w.language {
			w.setLanguage(value)
			w.updateStatusBar()
		}
	case "markdown":
		if on := value == "true"; on != w.markdown {
			w.setMarkdown(on)
		}
	case sourceMeta:
		w.adoptSource(value)
	case "format":
		if value != w.scriptFormatKey() {
			w.setScriptFormat(value)
			_ = w.settings.Save()
			w.app.applySettingsToWindows()
		}
	}
}

// shareDialog shares the script: a new key, its link to send.
func (w *MainWindow) shareDialog() {
	key := w.sharedKey()
	link := widget.NewEntry()
	link.SetText(shareLink(key))
	copyBtn := widget.NewButton("Copy the link", func() {
		fyne.CurrentApp().Clipboard().SetContent(link.Text)
		w.setStatus("Link copied")
	})
	msg := widget.NewLabel("Whoever has this link can read and edit the script with you, in the web version or in Accolade (Collaboration > Join a Shared Script…).\nThe peers connect directly; trackers only help them find each other and see nothing of the script.")
	msg.Wrapping = fyne.TextWrapWord
	stop := widget.NewButton("Stop sharing", func() { w.leaveSharing() })
	if w.shared == nil {
		stop.Disable()
	}
	d := dialog.NewCustom("Edit Together", "Close", container.NewVBox(msg, link, container.NewHBox(copyBtn, stop)), w.fyneWindow)
	d.Resize(fyne.NewSize(560, 0))
	d.Show()
}

// sharedKey is the key the script is shared with, sharing it first.
func (w *MainWindow) sharedKey() collab.Key {
	if w.shared != nil {
		return w.shared.key
	}
	key, err := collab.NewKey()
	if err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return key
	}
	w.startSharing(key, false)
	return key
}

// joinDialog joins a shared script by its link or key.
func (w *MainWindow) joinDialog() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("https://lapingvino.github.io/accolade/#k=…")
	dialog.ShowForm("Join a Shared Script", "Join", "Cancel",
		[]*widget.FormItem{widget.NewFormItem("Link or key", entry)}, func(ok bool) {
			if ok {
				w.join(entry.Text)
			}
		}, w.fyneWindow)
}

// join joins a shared script: in this window if it holds nothing of its
// own (see openPath), else in a new one (in the browser, after asking).
func (w *MainWindow) join(linkOrKey string) {
	key, err := collab.ParseKey(strings.TrimSpace(linkOrKey))
	if err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return
	}
	if w.shared != nil && w.shared.key == key {
		w.setStatus("You are editing this script together already")
		return
	}
	if w.shared != nil { // the address shows this one's until the other is joined
		setPageKey(w.shared.key.String())
	}
	w.withTarget(func(target *MainWindow) {
		target.startSharing(key, true)
		target.setStatus("Joining the shared script…")
	})
}

// sharingStatus is the status bar's word on sharing ("" when not shared).
func (w *MainWindow) sharingStatus() string {
	if w.shared == nil {
		return ""
	}
	if w.shared.paused {
		return "Shared · paused while away"
	}
	if s := w.shared; s.other {
		who := "another peer's Accolade"
		if s.otherApp != "" {
			who = "another peer's Accolade (" + s.otherApp + ")"
		}
		if s.otherOlder {
			return "Shared · " + who + " is older than yours: it needs updating"
		}
		return "Shared · " + who + " is newer than yours: update yours"
	}
	// connected, or here through one connected (a chain: session_tab.go)
	switch n := max(w.shared.peers, len(w.shared.session.Here())); n {
	case 0:
		return "Shared · looking for others"
	case 1:
		return "Shared · 1 other"
	default:
		return fmt.Sprintf("Shared · %d others", n)
	}
}

// wroteLine reports whether the script's line i is this writer's to
// follow with an automatic write: a locked script's new scene numbered,
// a lesson learned from a heading or a cue (the single-writer rule,
// docs/COLLAB.md). Every peer sees what triggers such a write, and each
// would make it, so only the writer of the line does: the replica that
// wrote its first rune, which is the same on every peer. Not shared, the
// writer is always this window. A line whose writer left (or wrote it in
// an earlier run, under another replica) is nobody's: Lock Scene Numbers
// still numbers it.
func (w *MainWindow) wroteLine(i int) bool {
	if w.shared == nil {
		return true
	}
	b := w.textEditor.Buffer()
	at := b.LineStart(i)
	own := false
	w.shared.session.Do(func(d *collab.Doc) {
		own = d.Len() == b.Len() && d.Writer(at) == d.Replica()
	})
	return own
}
