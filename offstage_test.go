// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// inSightEditors are the editors drawn in the window now: those in its
// tree of objects, none of their parents hidden.
func inSightEditors(w *MainWindow) []*editor.ScriptEditor {
	var out []*editor.ScriptEditor
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if o == nil || !o.Visible() {
			return
		}
		switch c := o.(type) {
		case *editor.ScriptEditor:
			out = append(out, c)
		case *fyne.Container:
			for _, child := range c.Objects {
				walk(child)
			}
		case fyne.Widget:
			for _, child := range test.WidgetRenderer(c).Objects() {
				walk(child)
			}
		}
	}
	walk(w.fyneWindow.Content())
	return out
}

// TestDocsOffstage: a document's editor out of sight is offstage (a key
// in it, its own or a co-writer's, does not wrap the whole text again),
// one in sight never is, and one shown again shows its text as it is now.
func TestDocsOffstage(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	// check: no editor in sight is offstage, and the document's editor is
	// offstage exactly when it is out of sight; it reports whether in sight
	check := func(step string, d *document) bool {
		t.Helper()
		seen := false
		for _, e := range inSightEditors(w) {
			if e.Offstage() {
				t.Errorf("%s: an editor in sight is offstage (%s)", step, e.AccessibleName)
			}
			seen = seen || d != nil && e == d.ed
		}
		if d != nil && !seen && !d.ed.Offstage() {
			t.Errorf("%s: the %s editor out of sight is not offstage", step, d.kind)
		}
		return seen
	}
	must := func(step string, d *document, shown bool) {
		t.Helper()
		if check(step, d) != shown {
			t.Fatalf("%s: the %s editor in sight is %v, want %v", step, d.kind, !shown, shown)
		}
	}
	type1 := func(d *document, s string) {
		d.ed.Edit(func(b *buffer.Buffer) { b.SetCursor(0, false); b.Insert(s) })
	}
	rowHas := func(step string, d *document, s string) {
		t.Helper()
		if row := d.ed.GridRow(0); !strings.Contains(rowText(row), s) {
			t.Errorf("%s: the first row %q has not %q", step, rowText(row), s)
		}
	}

	d := w.sourceDoc() // opened to follow (as a shared source's Doc): out of sight
	must("opened", d, false)
	type1(d, "Hidden ")

	s := w.sourcePane()
	s.toggleEdit() // the pane's Edit: in sight
	must("pane Edit", d, true)
	rowHas("pane Edit", d, "Hidden ")

	// the other panes in and out, in front of the Source's place or not
	for _, id := range []string{"notes", "outline", "characters", "notes", "source", "source", "outline", "characters"} {
		w.togglePane(id)
		if !check("pane "+id, d) {
			type1(d, "Behind ")
			w.dock.pick("source")
			if !w.dock.shown("source") {
				w.togglePane("source")
			}
			must("source back after "+id, d, true)
			rowHas("source back after "+id, d, "Behind ")
		}
	}
	w.togglePane("source")
	if w.dock.shown("source") {
		w.togglePane("source")
	}
	must("pane away", d, false)
	w.togglePane("source")
	must("pane back", d, true)

	s.toggleEdit() // Done: the document let go
	check("Done", nil)

	w.openMiddle("source")
	d = w.midDoc()
	must("middle", d, true)
	w.showMiddle("")
	must("the script in front", d, false)
	if w.textEditor.Offstage() {
		t.Error("the script in front is offstage")
	}
	type1(d, "Away ")
	w.showMiddle("source")
	must("middle again", d, true)
	rowHas("middle again", d, "Away ")
	if !w.textEditor.Offstage() {
		t.Error("the script behind the source is not offstage")
	}
	w.backToSide("source")
	check("back to the side", nil)

	// a phone: the pane under the script, folded down under the board
	w.fyneWindow.Resize(fyne.NewSize(400, 760))
	w.onWindowWidth(400)
	if !w.narrow {
		t.Fatal("not narrow")
	}
	if !w.dock.shown("source") {
		w.togglePane("source")
	}
	s.toggleEdit()
	d = w.srcDoc
	must("phone Edit", d, true)
	w.sheetDown = true
	w.relayout()
	must("phone sheet down", d, false)
	type1(d, "Folded ")
	if w.raiseSheet() {
		w.relayout()
	}
	must("phone sheet up", d, true)
	rowHas("phone sheet up", d, "Folded ")
	s.toggleEdit()
}

// rowText is a grid row's characters.
func rowText(r widget.TextGridRow) string {
	var b strings.Builder
	for _, c := range r.Cells {
		b.WriteRune(c.Rune)
	}
	return b.String()
}
