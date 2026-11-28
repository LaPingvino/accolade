// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"os"
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
	noTrackers(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	w0, app := newDialogTestWindow(t)
	holdUI(t)
	prev := w0.settings.GetString("your-name")
	t.Cleanup(func() { w0.settings.SetString("your-name", prev) })
	a, b := app.newWindow(), app.newWindow()
	a.textEditor.SetText("INT. BARN - DAY\n\nRain falls.\n")
	key, _ := collab.NewKey()
	w0.settings.SetString("your-name", "Anna")
	a.startSharing(key, false)
	_ = os.Remove(keptSharedPath(key))         // b from nothing, as on another computer
	w0.settings.SetString("your-name", "Bram") // the windows share the settings
	b.startSharing(key, true)
	t.Cleanup(func() { a.stopSharing(); b.stopSharing() })
	connect(t, a, b)
	textIs(t, b, "b", "INT. BARN - DAY\n\nRain falls.\n")
	if len(a.changes)+len(b.changes) != 0 {
		t.Fatalf("joining made changes: %v %v", a.changes, b.changes)
	}

	b.fyneWindow.Canvas().Focus(b.textEditor)
	b.textEditor.SetCursorOffset(len("INT. BARN - DAY\n\nRain"))
	test.Type(b.textEditor, " hard")
	textIs(t, a, "a", "INT. BARN - DAY\n\nRain hard falls.\n")
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

	a.togglePane("changes")
	if !a.paneShown("changes") || a.changesPanel.list.Length() != 1 {
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
