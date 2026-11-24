package main

import (
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
// structure; Insert Structure (:structure) puts one at the end of the
// body, before the blocks at the end, one undo step.
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
