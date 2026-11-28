// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"strings"
	"testing"
)

const openSample = `Title: The Test

# Act One

= The act's own synopsis.

= A beat with no scene yet.

= Another beat.

INT. HOUSE - DAY

Anna waits. [[todo: give her a reason]]

[[Is this too slow?]]

[[red]]

[[from: "She opened the door"]]

/* INT. GARDEN - NIGHT

ANNA
Goodbye.
*/

/* a thought about the ending */

= A beat at the scene's end.

.OMITTED #12#

/* Omitted 12
INT. CELLAR - DAY #12#

Dark.
*/

# Act Two [[pace this]]

INT. ROAD - DAY

Bram drives.

/* Notes
Call Bram.

Check the lake * / scene.
*/
`

func TestOpenItems(t *testing.T) {
	items := openItemsOf(openSample, nil, false, false)
	type want struct {
		kind  openKind
		text  string
		title string
		ctx   string
	}
	wants := []want{
		{openBeat, "A beat with no scene yet.", "", "# Act One"},
		{openBeat, "Another beat.", "", "# Act One"},
		{openTodo, "give her a reason", "", "INT. HOUSE - DAY"},
		{openNote, "Is this too slow?", "", "INT. HOUSE - DAY"},
		{openCut, "INT. GARDEN - NIGHT\n\nANNA\nGoodbye.", "INT. GARDEN - NIGHT", "INT. HOUSE - DAY"},
		{openNote, "a thought about the ending", "", "INT. HOUSE - DAY"},
		{openBeat, "A beat at the scene's end.", "", "INT. HOUSE - DAY"},
		{openCut, "INT. CELLAR - DAY #12#\n\nDark.", "INT. CELLAR - DAY", "OMITTED"},
		{openUnsorted, "Call Bram.", "", "Unsorted notes"},
		{openUnsorted, "Check the lake */ scene.", "", "Unsorted notes"},
	}
	if len(items) != len(wants) {
		for _, it := range items {
			t.Logf("%v %q %q", it.Kind, it.Text, it.Ctx)
		}
		t.Fatalf("%d items, want %d", len(items), len(wants))
	}
	for i, w := range wants {
		it := items[i]
		if it.Kind != w.kind || it.Text != w.text || it.Ctx != w.ctx || w.title != "" && it.Title != w.title {
			t.Errorf("item %d: %v %q %q (%q), want %v %q %q (%q)", i, it.Kind, it.Text, it.Ctx, it.Title, w.kind, w.text, w.ctx, w.title)
		}
		if !it.still(openSample) {
			t.Errorf("item %d: its place %d-%d is not %q", i, it.Start, it.End, it.Raw)
		}
	}
	if !items[7].Kept || items[7].Omitted != "12" {
		t.Errorf("the omitted scene: kept %v, number %q", items[7].Kept, items[7].Omitted)
	}
}

func TestTodoNote(t *testing.T) {
	for in, want := range map[string]string{
		"todo: fix": "fix", "TODO fix it": "fix it", "To do: x": "x", "todo": "", " TODO:  y ": "y",
	} {
		_, rest, _, ok := todoNote(in)
		if !ok || rest != want {
			t.Errorf("%q: %q %v, want %q", in, rest, ok, want)
		}
	}
	for _, in := range []string{"todos are many", "a todo", "toddler"} {
		if _, _, _, ok := todoNote(in); ok {
			t.Errorf("%q read as a to-do", in)
		}
	}
}

func TestOpenEdits(t *testing.T) {
	items := openItemsOf(openSample, nil, false, false)
	edit := func(e textEdit) string { return e.apply(openSample) }
	// a to-do edited keeps its prefix
	got := edit(items[2].editTo("give her a gun", openSample, false))
	if !strings.Contains(got, "Anna waits. [[todo: give her a gun]]") {
		t.Errorf("to-do edited: %q", got)
	}
	// a to-do emptied goes, with the space before it
	got = edit(items[2].editTo("", openSample, false))
	if !strings.Contains(got, "Anna waits.\n") {
		t.Errorf("to-do taken out: %q", got)
	}
	// a note alone on its line goes with its line, one blank line kept
	got = edit(items[3].removal(openSample))
	if strings.Contains(got, "too slow") || strings.Contains(got, "\n\n\n") {
		t.Errorf("note taken out: %q", got)
	}
	// a cut restored: its inside in the script, blank lines around it
	e, ok := items[4].restore(openSample, false)
	got = edit(e)
	if !ok || !strings.Contains(got, "\n\nINT. GARDEN - NIGHT\n\nANNA\nGoodbye.\n\n") || strings.Contains(got, "/* INT. GARDEN") {
		t.Errorf("cut restored: %q", got)
	}
	// an omitted scene restored as Restore Scene brings it back
	e, ok = items[7].restore(openSample, false)
	got = edit(e)
	if !ok || strings.Contains(got, ".OMITTED #12#") || !strings.Contains(got, "INT. CELLAR - DAY #12#\n\nDark.") {
		t.Errorf("omitted scene restored: %q", got)
	}
	// a beat edited stays a synopsis
	got = edit(items[0].editTo("A new beat\nover two lines", openSample, false))
	if !strings.Contains(got, "= A new beat\n= over two lines\n") {
		t.Errorf("beat edited: %q", got)
	}
	// a boneyard note edited, on one line
	got = edit(items[5].editTo("a */ thought", openSample, false))
	if !strings.Contains(got, "/* a * / thought */") {
		t.Errorf("boneyard edited: %q", got)
	}
	// the unsorted notes: one edited, one taken out, one added
	got = edit(items[9].editTo("Check the */ lake", openSample, false))
	if !strings.Contains(got, "Call Bram.\n\nCheck the * / lake\n*/") {
		t.Errorf("unsorted edited: %q", got)
	}
	got = edit(removeUnsorted(openSample, items[8], fountainNotes))
	if strings.Contains(got, "Call Bram") || !strings.Contains(got, "/* Notes\nCheck the lake * / scene.\n*/") {
		t.Errorf("unsorted taken out: %q", got)
	}
	got = edit(newUnsorted(openSample, "Ring the lake's owner.", fountainNotes))
	if !strings.HasSuffix(got, "Check the lake * / scene.\n\nRing the lake's owner.\n*/\n") {
		t.Errorf("unsorted added: %q", got)
	}
	plain := "INT. HOUSE - DAY\n\nAnna waits.\n"
	got = newUnsorted(plain, "First.", fountainNotes).apply(plain)
	if got != "INT. HOUSE - DAY\n\nAnna waits.\n\n/* Notes\nFirst.\n*/\n" {
		t.Errorf("the first unsorted note: %q", got)
	}
	if its := openItemsOf(got, nil, false, false); len(its) != 1 || its[0].Kind != openUnsorted {
		t.Errorf("read back: %+v", its)
	}
}

func TestOpenItemsSkip(t *testing.T) {
	// the title page, Accolade's own end blocks, a heading's tags, a
	// storyboard frame: none of them items
	text := "Title: [[not a note]]\n\nINT. HOUSE - DAY [[tag]]\n\nAnna.\n\n[[board: a.png]]\n\n/* Characters\nANNA: the lead\n*/\n"
	if its := openItemsOf(text, nil, false, false); len(its) != 0 {
		t.Errorf("items: %+v", its)
	}
	// prose in the boneyard is a note; a beat before any heading is one
	text = "= Before it all.\n\nINT. HOUSE - DAY\n\n/*\nShe could be older.\n*/\n"
	its := openItemsOf(text, nil, false, false)
	if len(its) != 2 || its[0].Kind != openBeat || its[0].Ctx != "" || its[1].Kind != openNote || its[1].Text != "She could be older." {
		t.Errorf("items: %+v", its)
	}
}

func TestMarkdownOpenItems(t *testing.T) {
	text := "# One\n\nText.\n\n<!-- TODO: cite it -->\n\n<!--\nA thought.\n-->\n\n<!-- Notes\nGeneral.\n-->\n"
	its := openItemsOf(text, nil, true, false)
	if len(its) != 3 || its[0].Kind != openTodo || its[0].Text != "cite it" || its[1].Kind != openNote || its[1].Text != "A thought." || its[2].Kind != openUnsorted {
		t.Fatalf("items: %+v", its)
	}
	got := its[0].editTo("cite both", text, true).apply(text)
	if !strings.Contains(got, "<!-- TODO: cite both -->") {
		t.Errorf("edited: %q", got)
	}
}
