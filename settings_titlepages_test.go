// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The Title Pages page: a preset overridden into the file, a key's field
// changed, keys moved and removed, the template removed; each an edit of
// the text that reads back as Lexington reads it.
func TestTitlePagesEditor(t *testing.T) {
	w := tomlTestApp(t)
	path := filepath.Join(t.TempDir(), "lexington.toml")
	if err := os.WriteFile(path, []byte("# mine\nStandard = \"en\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w.openPath(path)
	e := w.app.tomlEditorOf(path)
	selectTab(t, e, "Title Pages")
	e.sel["titlepage"] = "bold"
	e.redraw()
	test.Tap(button(t, e, "Override"))
	bold, ok := e.doc.conf().TitlePages["bold"]
	if !ok || len(bold.Keys) != 9 || bold.Keys[0].Key != "Title" || bold.Keys[0].Size != 26 || !bold.Keys[0].Caps {
		t.Fatalf("overridden: %+v\n%s", bold, e.doc.text)
	}
	// the first key's size typed anew
	size := find(e.tabs.Selected().Content, func(en *widget.Entry) bool { return en.Text == "26" })
	if size == nil {
		t.Fatal("no Size entry with 26")
	}
	size.SetText("30")
	if got := e.doc.conf().TitlePages["bold"].Keys[0].Size; got != 30 {
		t.Errorf("size %v\n%s", got, e.doc.text)
	}
	// the first key moved down, the last removed
	var downs, removes []*widget.Button
	collect(e.tabs.Selected().Content, func(b *widget.Button) {
		if b.Icon != nil && b.Text == "" && !b.Disabled() {
			switch b.Icon.Name() {
			case theme.MoveDownIcon().Name():
				downs = append(downs, b)
			case theme.DeleteIcon().Name():
				removes = append(removes, b)
			}
		}
	})
	if len(downs) == 0 || len(removes) == 0 {
		t.Fatalf("buttons: %d down, %d remove", len(downs), len(removes))
	}
	test.Tap(downs[0])
	if k := e.doc.conf().TitlePages["bold"].Keys; k[0].Key != "Logline" || k[1].Key != "Title" || k[1].Size != 30 {
		t.Errorf("moved: %v %v\n%s", k[0], k[1], e.doc.text)
	}
	removes = nil
	collect(e.tabs.Selected().Content, func(b *widget.Button) {
		if b.Icon != nil && b.Text == "" && b.Icon.Name() == theme.DeleteIcon().Name() {
			removes = append(removes, b)
		}
	})
	test.Tap(removes[len(removes)-1])
	if k := e.doc.conf().TitlePages["bold"].Keys; len(k) != 8 || k[len(k)-1].Key == "*" {
		t.Errorf("removed: %d keys\n%s", len(k), e.doc.text)
	}
	before := e.doc.text
	if err := e.removeTitleTemplate("bold"); err != nil {
		t.Fatalf("%v\n%s\nnow:\n%s", err, before, e.doc.text)
	}
	if _, ok := e.doc.conf().TitlePages["bold"]; ok || e.doc.has("TitlePages", "bold") {
		t.Errorf("the template stays:\n%s", e.doc.text)
	}
	// of two, one removed: the other stays whole
	for _, k := range []string{"bold", "poster"} {
		e.sel["titlepage"] = k
		e.redraw()
		test.Tap(button(t, e, "Override"))
	}
	if err := e.removeTitleTemplate("bold"); err != nil {
		t.Fatal(err)
	}
	if c := e.doc.conf().TitlePages; len(c) != 1 || len(c["poster"].Keys) != 10 {
		t.Errorf("left: %v\n%s", c, e.doc.text)
	}
	if !strings.Contains(e.doc.text, "# mine\nStandard = \"en\"") {
		t.Errorf("the rest lost:\n%s", e.doc.text)
	}
}
