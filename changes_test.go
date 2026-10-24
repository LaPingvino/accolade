package main

import (
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"

	"github.com/LaPingvino/accolade/internal/collab"
)

// The history of a shared script: b's typing shows in a's Changes as b's,
// by the name b gave, at its line; close edits make one change; a's own
// edits are not in it.
func TestChangesHistory(t *testing.T) {
	was := collab.Trackers
	collab.Trackers = nil
	t.Cleanup(func() { collab.Trackers = was })
	w0, app := newDialogTestWindow(t)
	prev := w0.settings.GetString("your-name")
	t.Cleanup(func() { w0.settings.SetString("your-name", prev) })
	a, b := app.newWindow(), app.newWindow()
	a.textEditor.SetText("INT. BARN - DAY\n\nRain falls.\n")
	key, _ := collab.NewKey()
	w0.settings.SetString("your-name", "Anna")
	a.startSharing(key, false)
	w0.settings.SetString("your-name", "Bram") // the windows share the settings
	b.startSharing(key, true)
	t.Cleanup(func() { a.stopSharing(); b.stopSharing() })
	sa, sb := a.shared.session, b.shared.session
	sa.AddPeer("b", func(m []byte) error { return sb.Receive("a", m) })
	sb.AddPeer("a", func(m []byte) error { return sa.Receive("b", m) })
	if len(a.changes)+len(b.changes) != 0 {
		t.Fatalf("joining made changes: %v %v", a.changes, b.changes)
	}

	b.fyneWindow.Canvas().Focus(b.textEditor)
	b.textEditor.SetCursorOffset(len("INT. BARN - DAY\n\nRain"))
	test.Type(b.textEditor, " hard")
	if len(a.changes) != 1 {
		t.Fatalf("a's history: %+v", a.changes)
	}
	c := a.changes[0]
	if c.Who != "Bram" || c.Line != 2 || c.Added != " hard" || c.Removed != "" {
		t.Errorf("the change: %+v", c)
	}
	if len(b.changes) != 0 {
		t.Errorf("b's own edit is in b's history: %+v", b.changes)
	}

	a.toggleChanges()
	if !a.changesVisible || a.changesPanel.list.Length() != 1 {
		t.Errorf("the panel lists %d", a.changesPanel.list.Length())
	}
	a.changesPanel.list.Select(0)
	if line, _ := a.textEditor.Buffer().Position(a.textEditor.CursorOffset()); line != 2 {
		t.Errorf("tapped, the change shows line %d", line)
	}
	if got := ago(time.Now().Add(-5*time.Minute), time.Now()); !strings.HasPrefix(got, "5 min") {
		t.Errorf("ago: %q", got)
	}
}
