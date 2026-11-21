package main

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/test"

	"github.com/LaPingvino/accolade/internal/collab"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The tests of editing together connect two windows' sessions here, over
// a loopback that delivers each side's messages in order on a goroutine
// of its own, as a data channel does: what a message does then waits for
// the UI turn (Session.Run is uiDo), as on the real UI goroutine. The
// tests hold the UI turn (holdUI) and give it up while they wait.

// noTrackers keeps the sessions off the internet.
func noTrackers(t *testing.T) {
	was := collab.Trackers
	collab.Trackers = nil
	t.Cleanup(func() { collab.Trackers = was })
}

// pipe delivers messages to a session, in order, on its own goroutine.
type pipe struct {
	mu     sync.Mutex
	cond   *sync.Cond
	q      [][]byte
	closed bool
}

func newPipe(to *collab.Session, from string) *pipe {
	p := &pipe{}
	p.cond = sync.NewCond(&p.mu)
	go func() {
		for {
			p.mu.Lock()
			for len(p.q) == 0 && !p.closed {
				p.cond.Wait()
			}
			if p.closed {
				p.mu.Unlock()
				return
			}
			m := p.q[0]
			p.q = p.q[1:]
			p.mu.Unlock()
			_ = to.Receive(from, m)
		}
	}()
	return p
}

func (p *pipe) send(m []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.closed {
		p.q = append(p.q, append([]byte(nil), m...))
		p.cond.Signal()
	}
	return nil
}

func (p *pipe) close() {
	p.mu.Lock()
	p.closed = true
	p.cond.Broadcast()
	p.mu.Unlock()
}

// connect joins two windows' sessions as peers a and b; the returned
// function takes the connection away again.
func connect(t *testing.T, a, b *MainWindow) func() {
	t.Helper()
	sa, sb := a.shared.session, b.shared.session
	toB, toA := newPipe(sb, "a"), newPipe(sa, "b")
	sa.AddPeer("b", toB.send)
	sb.AddPeer("a", toA.send)
	done := false
	cut := func() {
		if !done {
			done = true
			toA.close()
			toB.close()
			sa.RemovePeer("b")
			sb.RemovePeer("a")
		}
	}
	t.Cleanup(cut)
	return cut
}

// sharedPair is two windows of one app, a sharing text and b joining, in
// scratch folders, not yet connected.
func sharedPair(t *testing.T, text string) (a, b *MainWindow, key collab.Key) {
	t.Helper()
	noTrackers(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	_, app := newDialogTestWindow(t)
	holdUI(t)
	a, b = app.newWindow(), app.newWindow()
	a.textEditor.SetText(text)
	key, _ = collab.NewKey()
	a.startSharing(key, false)
	// one data folder for both: b starts from nothing, as on another computer
	_ = os.Remove(keptSharedPath(key))
	b.startSharing(key, true)
	t.Cleanup(func() { a.stopSharing(); b.stopSharing() })
	return a, b, key
}

func textIs(t *testing.T, w *MainWindow, name, want string) {
	t.Helper()
	waitFor(t, name+" has "+want, func() bool { return w.textEditor.Text() == want })
}

// Two windows editing one script: typing in one shows in the other, whose
// cursor stays with its text; the script's language goes with it.
func TestEditingTogether(t *testing.T) {
	a, b, _ := sharedPair(t, "")
	a.setLanguage("eo")
	b.setLanguage("en")
	a.textEditor.SetText("INT. BARN - DAY\n\nRain falls.\n")
	connect(t, a, b)
	textIs(t, b, "b", a.textEditor.Text())
	waitFor(t, "b in a's language", func() bool { return b.language == "eo" })
	if b.shared.session.Meta("format") != a.settings.GetString("script-format") {
		t.Errorf("b joined in format %q", b.shared.session.Meta("format"))
	}
	b.setLanguage("ru")
	waitFor(t, "a in b's language", func() bool { return a.language == "ru" })
	b.setLanguage("eo")
	waitFor(t, "a back in eo", func() bool { return a.language == "eo" })
	b.textEditor.SetCursorOffset(len([]rune("INT. BARN - DAY\n\nRain"))) // before " falls."
	a.fyneWindow.Canvas().Focus(a.textEditor)
	a.textEditor.SetCursorOffset(0)
	test.Type(a.textEditor, "FADE IN:\n\n")
	textIs(t, b, "b", "FADE IN:\n\nINT. BARN - DAY\n\nRain falls.\n")
	if got := b.textEditor.CursorOffset(); got != len([]rune("FADE IN:\n\nINT. BARN - DAY\n\nRain")) {
		t.Errorf("b's cursor at %d: it did not move with the text", got)
	}
	b.fyneWindow.Canvas().Focus(b.textEditor)
	test.Type(b.textEditor, " hard")
	textIs(t, a, "a", "FADE IN:\n\nINT. BARN - DAY\n\nRain hard falls.\n")
	if a.sharingStatus() == "" {
		t.Error("the status bar does not say it is shared")
	}
}

// Sharing pauses while away and comes back, the script still shared, and
// what was typed meanwhile goes to the peers once they meet again.
func TestPauseSharing(t *testing.T) {
	a, b, key := sharedPair(t, "INT. BARN - DAY\n")
	a.pauseSharing()
	if !strings.Contains(a.sharingStatus(), "paused") {
		t.Errorf("status %q", a.sharingStatus())
	}
	a.textEditor.SetText("INT. BARN - NIGHT\n") // typed while away
	a.resumeSharing()
	if a.shared.paused || a.shared.key != key {
		t.Fatal("not shared again")
	}
	connect(t, a, b) // they meet again
	textIs(t, b, "b", "INT. BARN - NIGHT\n")
}

// Keys typed while a peer's edits are on their way (the document has
// them, the editor not yet) never take those edits away: the document
// changes only in the UI turn, with the editor.
func TestTypingDuringCatchUp(t *testing.T) {
	long := strings.Repeat("INT. ROOM - DAY\n\nSomething happens here, at length.\n\n", 400)
	a, b, _ := sharedPair(t, long)
	connect(t, a, b)
	// the UI turn kept: a's script comes in meanwhile (it would reach b's
	// document, not its editor, if the document changed outside the turn)
	for deadline := time.Now().Add(300 * time.Millisecond); time.Now().Before(deadline); {
		var got string
		b.shared.session.Do(func(d *collab.Doc) { got = d.Text() })
		if got != "" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	b.fyneWindow.Canvas().Focus(b.textEditor)
	b.textEditor.SetCursorOffset(0)
	test.Type(b.textEditor, "x") // b types while it does
	// both have a's script and b's x (where the x goes among a's first
	// runes, typed at the same place, is the replicas' order)
	waitFor(t, "a and b alike, with the x", func() bool {
		ta, tb := a.textEditor.Text(), b.textEditor.Text()
		return ta == tb && strings.Count(tb, "x") == 1 && strings.Replace(tb, "x", "", 1) == long
	})
}

// Undo in a shared script undoes your own edits, never a co-writer's.
func TestUndoOnlyYourOwn(t *testing.T) {
	a, b, _ := sharedPair(t, "INT. BARN - DAY\n\nRain falls.\n")
	connect(t, a, b)
	textIs(t, b, "b", "INT. BARN - DAY\n\nRain falls.\n")
	a.fyneWindow.Canvas().Focus(a.textEditor)
	a.textEditor.SetCursorOffset(len("INT. BARN - DAY\n\nRain"))
	test.Type(a.textEditor, " hard")
	textIs(t, b, "b", "INT. BARN - DAY\n\nRain hard falls.\n")
	b.fyneWindow.Canvas().Focus(b.textEditor)
	b.textEditor.SetCursorOffset(0)
	test.Type(b.textEditor, "FADE IN:\n\n")
	textIs(t, a, "a", "FADE IN:\n\nINT. BARN - DAY\n\nRain hard falls.\n")
	for a.textEditor.Buffer().CanUndo() {
		a.textEditor.Undo()
	}
	textIs(t, b, "b", "FADE IN:\n\nINT. BARN - DAY\n\nRain falls.\n")
	textIs(t, a, "a", "FADE IN:\n\nINT. BARN - DAY\n\nRain falls.\n")
}

// A whole-text action (renumbering every scene) while the co-writer types
// in the middle: both stay.
func TestWholeTextActionWhileTyping(t *testing.T) {
	var sb strings.Builder
	for i := 1; i <= 12; i++ {
		sb.WriteString("INT. ROOM " + strings.Repeat("I", i%4+1) + " - DAY #" + string(rune('0'+i%10)) + "#\n\nAction.\n\n")
	}
	start := sb.String()
	a, b, _ := sharedPair(t, start)
	cut := connect(t, a, b)
	textIs(t, b, "b", start)
	cut() // both edit before they hear from each other
	renumbered := strings.NewReplacer("#1#", "#A#", "#5#", "#E#", "#9#", "#I#", "#2#", "#B#").Replace(start)
	a.textEditor.Edit(func(buf *buffer.Buffer) { buf.Replace(0, buf.Len(), renumbered) })
	mid := strings.Index(start, "#6#")
	b.textEditor.Edit(func(buf *buffer.Buffer) { buf.Replace(mid-1, mid-1, " (LATER)") })
	connect(t, a, b)
	want := renumbered[:mid-1] + " (LATER)" + renumbered[mid-1:]
	textIs(t, a, "a", want)
	textIs(t, b, "b", want)
}

// Another document in a shared window leaves the session: the peers keep
// the script as it was.
func TestNewDocumentLeavesSharing(t *testing.T) {
	a, b, _ := sharedPair(t, "INT. BARN - DAY\n")
	connect(t, a, b)
	textIs(t, b, "b", "INT. BARN - DAY\n")
	b.createNewFile(false) // File > New in the joined window
	if b.shared != nil {
		t.Fatal("b is still in the session")
	}
	uiSleep(50e6)
	if a.textEditor.Text() != "INT. BARN - DAY\n" {
		t.Errorf("a's script became %q", a.textEditor.Text())
	}
	b.NewMarkdownFile()
	if a.markdown {
		t.Error("a went Markdown with b")
	}
	// a shared window is never replaced by Open: it goes to another one
	if b.shared == nil {
		var target *MainWindow
		a.withTarget(func(w *MainWindow) { target = w })
		if target == a {
			t.Error("Open would replace the shared script")
		}
	}
}

// A window joining a room it was in before starts from what it kept, with
// nobody else there, and merges with the peers without doubling.
func TestRejoinFromKeptState(t *testing.T) {
	a, b, key := sharedPair(t, "INT. BARN - DAY\n\nRain.\n")
	cut := connect(t, a, b)
	textIs(t, b, "b", "INT. BARN - DAY\n\nRain.\n")
	waitFor(t, "b's state kept", func() bool {
		data, err := os.ReadFile(keptSharedPath(key))
		return err == nil && strings.Contains(string(data), "Rain")
	})
	cut()
	b.stopSharing() // a reload: the page goes, its kept state stays
	b.textEditor.SetText("")
	b.startSharing(key, true)
	if b.textEditor.Text() != "INT. BARN - DAY\n\nRain.\n" {
		t.Fatalf("b rejoined with %q", b.textEditor.Text())
	}
	a.textEditor.SetText("INT. BARN - NIGHT\n\nRain.\n") // before they meet again
	connect(t, a, b)
	textIs(t, b, "b", "INT. BARN - NIGHT\n\nRain.\n")
	textIs(t, a, "a", "INT. BARN - NIGHT\n\nRain.\n")
	b.leaveSharing()
	if _, err := os.Stat(keptSharedPath(key)); err == nil {
		t.Error("Stop sharing left the kept state")
	}
}

// Markdown documents on their own, edited together: the mode goes to the
// joiner, and a Markdown command is an edit like any other.
func TestMarkdownTogether(t *testing.T) {
	noTrackers(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	_, app := newDialogTestWindow(t)
	holdUI(t)
	a, b := app.newWindow(), app.newWindow()
	a.setMarkdown(true)
	a.textEditor.SetText("# Notes\n\nfirst line\n")
	key, _ := collab.NewKey()
	a.startSharing(key, false)
	b.startSharing(key, true)
	t.Cleanup(func() { a.stopSharing(); b.stopSharing() })
	connect(t, a, b)
	textIs(t, b, "b", "# Notes\n\nfirst line\n")
	waitFor(t, "b in Markdown", func() bool { return b.markdown })
	b.textEditor.SetCursorOffset(len("# Notes\n\nfirst"))
	cmd("md.heading2").run(b)
	textIs(t, a, "a", "# Notes\n\n## first line\n")
}

// The link a host gets is read back by Join, and joining the same script
// twice does nothing.
func TestShareLinkJoins(t *testing.T) {
	a, _, key := sharedPair(t, "INT. BARN\n")
	link := shareLink(key)
	if !strings.HasPrefix(link, webVersion+"#k=") {
		t.Errorf("link %q", link)
	}
	if k, err := collab.ParseKey(link + "."); err != nil || k != key {
		t.Errorf("the link with a full stop: %v", err)
	}
	s := a.shared
	a.join(link)
	if a.shared != s {
		t.Error("joining the script it shares started again")
	}
}

// TestCollabScreenshots renders the dialogs of editing together, on the
// desktop and at a phone's size (SHOT_DIR as in screenshot_test.go):
// Edit Together, and the browser's question before Open leaves a shared
// script.
func TestCollabScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	a, _, _ := sharedPair(t, "INT. BARN - DAY\n\nRain falls.\n")
	for _, size := range []struct {
		name string
		w, h float32
	}{{"desktop", 1100, 650}, {"phone", 380, 800}} {
		a.fyneWindow.Resize(fyne.NewSize(size.w, size.h))
		a.shareDialog()
		snapWindow(t, a, filepath.Join(dir, "collab-together-"+size.name+".png"))
		closeOverlays(a)
		title, msg := a.replaceQuestion()
		dialog.ShowConfirm(title, msg, func(bool) {}, a.fyneWindow)
		snapWindow(t, a, filepath.Join(dir, "collab-leave-"+size.name+".png"))
		closeOverlays(a)
	}
}

func snapWindow(t *testing.T, w *MainWindow, path string) {
	t.Helper()
	w.fyneWindow.Content().Refresh()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
		t.Fatal(err)
	}
}

func closeOverlays(w *MainWindow) {
	for o := w.fyneWindow.Canvas().Overlays().Top(); o != nil; o = w.fyneWindow.Canvas().Overlays().Top() {
		w.fyneWindow.Canvas().Overlays().Remove(o)
	}
}
