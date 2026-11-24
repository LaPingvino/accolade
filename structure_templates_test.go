package main

import (
	"fmt"
	"strings"
	"testing"
)

// Each structure reads as its parts: sections only (no scenes yet),
// each with its own synopsis; Beats First's acts each with three loose
// ones. Each opens with its note, naming whose method it follows.
func TestStructureTemplates(t *testing.T) {
	want := map[string]struct{ sections, loose int }{
		"Three Acts":            {9, 0},
		"Sequences":             {11, 0},
		"Hero's Journey":        {15, 0},
		"Story Circle":          {10, 0},
		"Kishōtenketsu":         {4, 0},
		"Beat Sheet (15 Beats)": {18, 0},
		"Beats First":           {3, 9},
	}
	if len(structureTemplates) != len(want) {
		t.Errorf("%d structures, want %d", len(structureTemplates), len(want))
	}
	starts := sceneStarts("en")
	for _, st := range structureTemplates {
		w, ok := want[st.name]
		if !ok {
			t.Errorf("%s: not expected", st.name)
			continue
		}
		text := structureText(st)
		if !strings.HasPrefix(text, "[["+st.name) || !strings.Contains(strings.SplitN(text, "\n", 2)[0], "Help > Structure") {
			t.Errorf("%s: its note: %q", st.name, strings.SplitN(text, "\n", 2)[0])
		}
		if st.after != "" && !strings.Contains(text, "after "+st.after) {
			t.Errorf("%s: not attributed", st.name)
		}
		items := outlineOf(text, starts)
		loose := 0
		for _, it := range items {
			if it.Scene {
				t.Errorf("%s: a scene: %q", st.name, it.Text)
			}
			if it.Synopsis == "" {
				t.Errorf("%s: %q has no synopsis", st.name, it.Text)
			}
			loose += len(it.Loose)
		}
		if len(items) != w.sections || loose != w.loose {
			t.Errorf("%s: %d sections and %d loose synopses, want %d and %d", st.name, len(items), loose, w.sections, w.loose)
		}
	}
}

// New from Structure starts a script with a title page and the
// structure; Insert Structure (:structure) in a script with no sections
// puts one at the end of the body, before the blocks at the end, one undo
// step.
func TestStructureNewAndInsert(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	st := structureTemplates[0]
	w.newFromStructure(st)
	got := w.textEditor.Text()
	if !strings.HasPrefix(got, "Title: UNTITLED SCREENPLAY\n") || !strings.HasSuffix(got, st.body) || w.hasChanges {
		t.Fatalf("new:\n%s", got)
	}

	script := "Title: X\n\nINT. A - DAY\n\nOne.\n\n/* Notes\nkeep\n*/\n"
	w.textEditor.SetText(script)
	if err := w.runExCommand("structure", "kish"); err != nil {
		t.Fatal(err)
	}
	got = w.textEditor.Text()
	ki := structureText(structureTemplates[4])
	if want := "Title: X\n\nINT. A - DAY\n\nOne.\n\n" + ki + "\n/* Notes\nkeep\n*/\n"; got != want {
		t.Errorf("inserted:\n%q\nwant\n%q", got, want)
	}
	w.textEditor.Undo()
	if got := w.textEditor.Text(); got != script {
		t.Errorf("one undo:\n%q", got)
	}
}

// Insert Structure goes in at the cursor's section boundary: after the
// top-level section the cursor is in, before the next one (on such a
// heading, before it), else at the end of the body; nothing there before
// comes to be in another section.
func TestStructureInsertAtSection(t *testing.T) {
	const script = "Title: X\n\n# Act One\n= One.\n\nINT. A - DAY\n\nA.\n\n# Act Two\n\n## Part\n\nINT. B - DAY\n\nB.\n\n# Act Three\n\nINT. C - DAY\n\nC.\n"
	starts := sceneStarts("en")
	at := func(cursorAt string) string {
		at, _ := structureInsertAt(script, len([]rune(script[:strings.Index(script, cursorAt)])), starts)
		return string([]rune(script)[at:])
	}
	if got := at("B."); !strings.HasPrefix(got, "# Act Three") {
		t.Errorf("in Act Two's part: before %q", got)
	}
	if got := at("# Act Two"); !strings.HasPrefix(got, "# Act Two") {
		t.Errorf("on Act Two's heading: before %q", got)
	}
	if got := at("Title"); !strings.HasPrefix(got, "# Act One") {
		t.Errorf("on the title page: before %q", got)
	}
	if got := at("C."); got != "" {
		t.Errorf("in the last act: before %q", got)
	}
	// with ## sections only, nothing is followed by a top-level one: the end
	const parts = "## One\n\nINT. A - DAY\n\n## Two\n\nINT. B - DAY\n"
	if a, where := structureInsertAt(parts, 3, starts); a != len([]rune(parts)) || where != "at the end" {
		t.Errorf("## only: at %d, %s", a, where)
	}

	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(script)
	w.textEditor.SetCursorOffset(len([]rune(script[:strings.Index(script, "B.")])))
	before := outlineOf(script, starts)
	w.insertStructure(structureTemplates[4]) // Kishōtenketsu: four # sections
	after := outlineOf(w.textEditor.Text(), starts)
	if len(after) != len(before)+4 {
		t.Fatalf("%d items, then %d", len(before), len(after))
	}
	// the items there before keep their levels and order, the four in
	// between Act Two's and Act Three's
	var kept []string
	for _, it := range after {
		if !strings.Contains(structureTemplates[4].body, it.Text) {
			kept = append(kept, fmt.Sprint(it.Level, it.Text))
		}
	}
	var was []string
	for _, it := range before {
		was = append(was, fmt.Sprint(it.Level, it.Text))
	}
	if strings.Join(kept, "|") != strings.Join(was, "|") {
		t.Errorf("the script's items: %v, were %v", kept, was)
	}
	if got := w.textEditor.Text(); !strings.Contains(got, "B.\n\n"+structureText(structureTemplates[4])+"\n# Act Three") {
		t.Errorf("inserted:\n%s", got)
	}
}
