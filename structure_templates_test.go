// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/lex"
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
		"Freytag's Pyramid":     {8, 0},
		"Seven Key Steps":       {7, 0},
		"Six Stages":            {11, 0},
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
// structure; Insert Beats / Structure (:structure) in a script with no sections
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

// Insert Beats / Structure goes in at the cursor's section boundary: after the
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

// Every section of every structure has its aim: a purpose in one line
// and an intended tension of 1 to 5; no aim names a section the
// structure does not have, and every range one it has. Each structure's
// own text is recognised as it (structureOf), so its ranges and aims
// apply to a script started from it.
func TestStructureAims(t *testing.T) {
	for _, st := range structureTemplates {
		secs := templateSections(st)
		for k := range secs {
			a, ok := st.aims[k]
			switch {
			case !ok:
				t.Errorf("%s: %q has no aim", st.name, k)
			case strings.TrimSpace(a.purpose) == "" || strings.ContainsAny(a.purpose, "\n") || len([]rune(a.purpose)) > 80:
				t.Errorf("%s: %q's purpose is not one short line: %q", st.name, k, a.purpose)
			case a.tension < 1 || a.tension > 5:
				t.Errorf("%s: %q's tension %d, not 1 to 5", st.name, k, a.tension)
			}
		}
		if len(st.aims) != len(structureAims[st.name]) {
			t.Errorf("%s: two aims for one section", st.name)
		}
		for k := range st.aims {
			if !secs[k] {
				t.Errorf("%s: an aim for %q, not one of its sections", st.name, k)
			}
		}
		for k := range st.ranges {
			if !secs[k] {
				t.Errorf("%s: a range for %q, not one of its sections", st.name, k)
			}
		}
		var names []string
		for _, it := range outlineOf(st.body, sceneStarts("en")) {
			names = append(names, it.Text)
		}
		if got, ok := structureOf(names); st.name != "Beats First" && (!ok || got.name != st.name) {
			t.Errorf("%s: its own sections read as %q", st.name, got.name)
		}
	}
	if len(structureAims) != len(structureTemplates) {
		t.Errorf("aims for %d structures, %d structures", len(structureAims), len(structureTemplates))
	}
}

// A section of a known structure carries its aim to where the beats
// land: the tooltip and the list say its purpose and tension.
func TestBeatPlaceAim(t *testing.T) {
	st, _ := templateNamed("Six Stages")
	b := beatPlace{name: "Point of No Return", aim: st.aims[sectionKey("Point of No Return")]}
	if got := b.aimNote(); got != "Burn the hero's bridges · tension 4 of 5" {
		t.Errorf("aim: %q", got)
	}
	if (beatPlace{name: "Mine"}).aimNote() != "" {
		t.Error("a section of the writer's own has an aim")
	}
}

// structureAimsHelp is Help > Structure's list of what each beat is for,
// made from structureAims: its section, to the next heading.
func structureAimsHelp() string {
	var b strings.Builder
	b.WriteString("## What each beat is for\n\n")
	b.WriteString("Each structure's sections come with what they are for and the tension they are meant to carry, from 1 (calm) to 5 (the most): Accolade's own reading, not the method's words. Where the Beats Land (Revision tab) says the cursor's in its tooltip and lists them all with their tension, so reading down the list gives the intended rise and fall; the actual script is yours to compare.\n")
	for _, st := range structureTemplates {
		b.WriteString("\n**" + st.name + "**\n\n")
		for _, l := range strings.Split(st.body, "\n") {
			if !strings.HasPrefix(l, "#") {
				continue
			}
			name := lex.SectionText(l)
			a := st.aims[sectionKey(name)]
			indent := strings.Repeat("  ", strings.Count(strings.Fields(l)[0], "#")-1)
			fmt.Fprintf(&b, "%s- %s (%d): %s.\n", indent, name, a.tension, a.purpose)
		}
	}
	return b.String()
}

// Help > Structure lists every beat's aim as the table has it
// (UPDATE_HELP=1 writes it in).
func TestStructureAimsHelp(t *testing.T) {
	path := filepath.Join("docs", "help", "structure.md")
	page, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const head = "## What each beat is for\n"
	text := string(page)
	i := strings.Index(text, head)
	if i < 0 {
		t.Fatalf("%s has no %q", path, head)
	}
	j := strings.Index(text[i+len(head):], "\n## ")
	if j < 0 {
		t.Fatalf("%s: nothing after %q", path, head)
	}
	j += i + len(head) + 1
	want := structureAimsHelp() + "\n"
	if os.Getenv("UPDATE_HELP") != "" {
		if err := os.WriteFile(path, []byte(text[:i]+want+text[j:]), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if text[i:j] != want {
		t.Errorf("%s's beats are not what the table makes: UPDATE_HELP=1 go test -run TestStructureAimsHelp .", path)
	}
}

// A structure's beats are beats wherever beats are read (beatsOf): each
// structure put into an empty script has every beat listed under the
// Notes pane's Beats filter: each section marked [[beat]] (by its own
// synopsis) and each synopsis standing on its own (the notes form,
// Beats First's); an unmarked section (an act) is not one, and a marked
// one is not once a scene is in it.
func TestStructureBeatsInNotes(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.togglePane("notes")
	p := w.notesPane()
	beatsListed := func() []string {
		var out []string
		for _, r := range p.rows {
			if r.item >= 0 && p.all[r.item].Kind == openBeat {
				out = append(out, p.all[r.item].Text)
			}
		}
		return out
	}
	for _, st := range structureTemplates {
		w.textEditor.SetText("")
		w.insertStructure(st)
		p.read()
		// the beats the body has: a marked section's own synopsis, and
		// every synopsis after a section's own
		var want []string
		marked, own := false, false
		for _, l := range strings.Split(st.body, "\n") {
			switch {
			case strings.HasPrefix(l, "#"):
				marked, own = markedBeat(l), true
			case strings.HasPrefix(l, "="):
				if own && marked || !own {
					want = append(want, strings.TrimSpace(strings.TrimPrefix(l, "=")))
				}
				own = false
			}
		}
		if got := beatsListed(); strings.Join(got, "|") != strings.Join(want, "|") || len(want) == 0 {
			t.Errorf("%s: beats\n%q\nwant\n%q", st.name, got, want)
		}
	}
	for _, c := range []struct{ text, want string }{
		// an unmarked section is not a beat; a synopsis on its own is
		{"# Act One\n\n= What it does.\n\n= A beat.\n", "A beat."},
		// a marked section is, by its synopsis or else its name
		{"# Act One\n\n## Midpoint [[beat]]\n= It turns.\n\n## Climax [[ Beat ]]\n", "It turns.|Climax"},
		// with a scene in it, it has grown; the next stays
		{"## Part [[beat]]\n= Its beat.\n\nINT. A - DAY\n\nA.\n\n## Next [[beat]]\n= Next.\n", "Next."},
		// a [[beat]] note elsewhere marks nothing
		{"# Act One\n= Own.\n\nINT. A - DAY\n\nA. [[beat]]\n", ""},
	} {
		w.textEditor.SetText(c.text)
		p.read()
		if got := strings.Join(beatsListed(), "|"); got != c.want {
			t.Errorf("%q: beats %q, want %q", c.text, got, c.want)
		}
	}
}

// The marker is a note on the heading: the outline, the cards and the
// ranges read the section's name without it; Fountain keeps it, and an
// outline (OPML) out and in gives it back (TestOutlineRoundTrip); the
// PDF prints no section, so no marker.
func TestBeatMarkerKept(t *testing.T) {
	st, _ := templateNamed("Six Stages")
	text := structureText(st)
	items := outlineOf(text, sceneStarts("en"))
	if items[0].Text != "Setup" {
		t.Errorf("the section's name: %q", items[0].Text)
	}
	data, err := outlineOPML(text, sceneStarts("en"))
	if err != nil {
		t.Fatal(err)
	}
	back, err := scriptOfOutline(data, sceneStarts("en"), true)
	if err != nil || !strings.Contains(back, "# Setup [[beat]]\n") {
		t.Errorf("out and in: %v\n%s", err, back)
	}
}

// Every Section Is a Beat (the Beats group's setting, the script's own):
// on, every section with no scene in it is a beat, marked or not; off,
// only the marked ones and the synopses on their own. It is kept in the
// script's .accolade and read back from it.
func TestEverySectionBeat(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.togglePane("notes")
	p := w.notesPane()
	w.textEditor.SetText("# Act One\n= What it does.\n\n## Midpoint [[beat]]\n= It turns.\n\n## Plain\n\n# Act Two\n\nINT. A - DAY\n\nA.\n")
	beats := func() string {
		p.read()
		var out []string
		for _, r := range p.rows {
			if r.item >= 0 && p.all[r.item].Kind == openBeat {
				out = append(out, p.all[r.item].Text)
			}
		}
		return strings.Join(out, "|")
	}
	if got := beats(); got != "It turns." {
		t.Errorf("off: %q", got)
	}
	cmd("set.everySectionBeat").run(w)
	if got := beats(); got != "What it does.|It turns.|Plain" || !w.hasChanges {
		t.Errorf("on: %q (changed %v)", got, w.hasChanges)
	}
	path := filepath.Join(t.TempDir(), "Beats.accolade")
	if err := w.saveToFile(path); err != nil {
		t.Fatal(err)
	}
	w.flushBundleWrites()
	w2, _ := newDialogTestWindow(t)
	if err := w2.LoadFile(path); err != nil || !w2.everySectionBeat {
		t.Fatalf("reopened: %v, every section a beat %v", err, w2.everySectionBeat)
	}
	cmd("set.everySectionBeat").run(w)
	if got := beats(); got != "It turns." {
		t.Errorf("off again: %q", got)
	}
}
