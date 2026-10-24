package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/LaPingvino/accolade/internal/collab"
)

// Two windows editing one script (their sessions connected here, no
// network): typing in one shows in the other, whose cursor stays with
// its text; the script's language goes with it.
func TestEditingTogether(t *testing.T) {
	was := collab.Trackers
	collab.Trackers = nil
	t.Cleanup(func() { collab.Trackers = was })
	_, app := newDialogTestWindow(t)
	a, b := app.newWindow(), app.newWindow()
	a.textEditor.SetText("INT. BARN - DAY\n\nRain falls.\n")
	a.setLanguage("eo")
	b.setLanguage("en")
	key, _ := collab.NewKey()
	a.startSharing(key, false)
	b.startSharing(key, true)
	t.Cleanup(func() { a.stopSharing(); b.stopSharing() })
	sa, sb := a.shared.session, b.shared.session
	sa.AddPeer("b", func(m []byte) error { return sb.Receive("a", m) })
	sb.AddPeer("a", func(m []byte) error { return sa.Receive("b", m) })
	if b.textEditor.Text() != a.textEditor.Text() {
		t.Fatalf("b joined with %q", b.textEditor.Text())
	}
	if b.language != "eo" || sb.Meta("format") != a.settings.GetString("script-format") {
		t.Errorf("b joined in %q, format %q", b.language, sb.Meta("format"))
	}
	b.setLanguage("ru")
	if a.language != "ru" {
		t.Errorf("a's language stayed %q", a.language)
	}
	b.setLanguage("eo")
	b.textEditor.SetCursorOffset(len([]rune("INT. BARN - DAY\n\nRain"))) // before " falls."
	a.fyneWindow.Canvas().Focus(a.textEditor)
	a.textEditor.SetCursorOffset(0)
	test.Type(a.textEditor, "FADE IN:\n\n")
	if want := "FADE IN:\n\nINT. BARN - DAY\n\nRain falls.\n"; b.textEditor.Text() != want {
		t.Fatalf("b has %q", b.textEditor.Text())
	}
	if got := b.textEditor.CursorOffset(); got != len([]rune("FADE IN:\n\nINT. BARN - DAY\n\nRain")) {
		t.Errorf("b's cursor at %d: it did not move with the text", got)
	}
	b.fyneWindow.Canvas().Focus(b.textEditor)
	test.Type(b.textEditor, " hard")
	if want := "FADE IN:\n\nINT. BARN - DAY\n\nRain hard falls.\n"; a.textEditor.Text() != want {
		t.Errorf("a has %q", a.textEditor.Text())
	}
	if a.sharingStatus() == "" {
		t.Error("the status bar does not say it is shared")
	}
}

// Sharing pauses while away and comes back, the script still shared, and
// what was typed meanwhile goes to the peers once they meet again.
func TestPauseSharing(t *testing.T) {
	was := collab.Trackers
	collab.Trackers = nil
	t.Cleanup(func() { collab.Trackers = was })
	_, app := newDialogTestWindow(t)
	a, b := app.newWindow(), app.newWindow()
	a.textEditor.SetText("INT. BARN - DAY\n")
	key, _ := collab.NewKey()
	a.startSharing(key, false)
	b.startSharing(key, true)
	t.Cleanup(func() { a.stopSharing(); b.stopSharing() })
	a.pauseSharing()
	if !strings.Contains(a.sharingStatus(), "paused") {
		t.Errorf("status %q", a.sharingStatus())
	}
	a.textEditor.SetText("INT. BARN - NIGHT\n") // typed while away
	a.resumeSharing()
	if a.shared.paused || a.shared.key != key {
		t.Fatal("not shared again")
	}
	sa, sb := a.shared.session, b.shared.session // they meet again
	sa.AddPeer("b", func(m []byte) error { return sb.Receive("a", m) })
	sb.AddPeer("a", func(m []byte) error { return sa.Receive("b", m) })
	if b.textEditor.Text() != "INT. BARN - NIGHT\n" {
		t.Errorf("b has %q", b.textEditor.Text())
	}
}
