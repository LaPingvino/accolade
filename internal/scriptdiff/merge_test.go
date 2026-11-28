// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package scriptdiff

import (
	"regexp"
	"strings"
	"testing"
)

const mergeBase = `Title: Rain

INT. KITCHEN - DAY

Bram cooks.

BRAM
Dinner's ready.

EXT. GARDEN - NIGHT

Rain falls on the roses.

INT. SHED - DAY

Tom waits by the door.
`

func mergeHeading(l string) bool {
	return strings.HasPrefix(l, "INT.") || strings.HasPrefix(l, "EXT.") || strings.HasPrefix(l, "#")
}

var linkRe = regexp.MustCompile(` ?\[\[from:[^\]]*\]\]`)

// mergeOpts compare without the source links, which are carried over.
var mergeOpts = MergeOptions{
	IsHeading: mergeHeading,
	Key:       func(s string) string { return strings.TrimSpace(linkRe.ReplaceAllString(s, "")) },
	Carry: func(o, t string) string {
		for _, l := range linkRe.FindAllString(o, -1) {
			if !strings.Contains(t, strings.TrimSpace(l)) {
				t += l
			}
		}
		return t
	},
}

func TestMergeTable(t *testing.T) {
	for _, c := range []struct {
		name         string
		ours, theirs string
		kinds        string // the changes: kind/row, c for a conflict
		kept         int
		all          string // with every change taken (theirs where a conflict)
	}{
		{name: "no conflict",
			ours:   strings.Replace(mergeBase, "Tom waits by the door.", "Tom waits by the red door.", 1),
			theirs: strings.Replace(mergeBase, "Bram cooks.", "Bram cooks a stew.", 1),
			kinds:  "edit/changed", kept: 1,
			all: strings.Replace(strings.Replace(mergeBase, "Tom waits by the door.", "Tom waits by the red door.", 1), "Bram cooks.", "Bram cooks a stew.", 1)},
		{name: "both changed",
			ours:   strings.Replace(mergeBase, "Bram cooks.", "Bram cooks soup.", 1),
			theirs: strings.Replace(mergeBase, "Bram cooks.", "Bram cooks a stew.", 1),
			kinds:  "edit/changed c",
			all:    strings.Replace(mergeBase, "Bram cooks.", "Bram cooks a stew.", 1)},
		{name: "outside deleted a scene",
			ours:   mergeBase,
			theirs: strings.Replace(mergeBase, "EXT. GARDEN - NIGHT\n\nRain falls on the roses.\n\n", "", 1),
			kinds:  "cut/removed",
			all:    strings.Replace(mergeBase, "EXT. GARDEN - NIGHT\n\nRain falls on the roses.\n\n", "", 1)},
		{name: "outside deleted a scene we changed",
			ours:   strings.Replace(mergeBase, "Rain falls on the roses.", "Rain falls hard on the roses.", 1),
			theirs: strings.Replace(mergeBase, "EXT. GARDEN - NIGHT\n\nRain falls on the roses.\n\n", "", 1),
			kinds:  "cut/removed c",
			all:    strings.Replace(mergeBase, "EXT. GARDEN - NIGHT\n\nRain falls on the roses.\n\n", "", 1)},
		{name: "outside added one",
			ours:   mergeBase,
			theirs: strings.Replace(mergeBase, "INT. SHED - DAY", "EXT. ROAD - NIGHT\n\nA car passes.\n\nINT. SHED - DAY", 1),
			kinds:  "added/added",
			all:    strings.Replace(mergeBase, "INT. SHED - DAY", "EXT. ROAD - NIGHT\n\nA car passes.\n\nINT. SHED - DAY", 1)},
		{name: "outside added one at the end",
			ours:   mergeBase,
			theirs: mergeBase + "\nEXT. ROAD - NIGHT\n\nA car passes.\n",
			kinds:  "added/added",
			all:    mergeBase + "\nEXT. ROAD - NIGHT\n\nA car passes.\n"},
		{name: "reordered scenes",
			ours: mergeBase,
			theirs: `Title: Rain

INT. KITCHEN - DAY

Bram cooks.

BRAM
Dinner's ready.

INT. SHED - DAY

Tom waits by the door.

EXT. GARDEN - NIGHT

Rain falls on the roses.
`,
			kinds: "moved/same",
			all: `Title: Rain

INT. KITCHEN - DAY

Bram cooks.

BRAM
Dinner's ready.

INT. SHED - DAY

Tom waits by the door.

EXT. GARDEN - NIGHT

Rain falls on the roses.
`},
		{name: "a moved scene changed too, ours added one",
			ours: mergeBase + "\nINT. ATTIC - DAY\n\nDust.\n",
			theirs: `Title: Rain

EXT. GARDEN - NIGHT

Rain falls on the yellow roses.

INT. KITCHEN - DAY

Bram cooks.

BRAM
Dinner's ready.

INT. SHED - DAY

Tom waits by the door.
`,
			kinds: "moved/same edit/changed", kept: 0,
			all: `Title: Rain

EXT. GARDEN - NIGHT

Rain falls on the yellow roses.

INT. KITCHEN - DAY

Bram cooks.

BRAM
Dinner's ready.

INT. SHED - DAY

Tom waits by the door.

INT. ATTIC - DAY

Dust.
`},
		{name: "element added and cut in a scene",
			ours:   mergeBase,
			theirs: strings.Replace(strings.Replace(mergeBase, "BRAM\nDinner's ready.\n\n", "", 1), "Tom waits by the door.\n", "Tom waits by the door.\n\nTOM\nHello?\n", 1),
			kinds:  "edit/removed edit/added",
			all:    strings.Replace(strings.Replace(mergeBase, "BRAM\nDinner's ready.\n\n", "", 1), "Tom waits by the door.\n", "Tom waits by the door.\n\nTOM\nHello?\n", 1)},
		{name: "next to each other, each changed on one side",
			ours:   strings.Replace(mergeBase, "Dinner's ready.", "Dinner is ready.", 1),
			theirs: strings.Replace(mergeBase, "Bram cooks.", "Bram cooks a stew.", 1),
			kinds:  "edit/changed", kept: 1,
			all: strings.Replace(strings.Replace(mergeBase, "Dinner's ready.", "Dinner is ready.", 1), "Bram cooks.", "Bram cooks a stew.", 1)},
		{name: "the same change on both sides",
			ours:   strings.Replace(mergeBase, "Bram cooks.", "Bram cooks a stew.", 1),
			theirs: strings.Replace(mergeBase, "Bram cooks.", "Bram cooks a stew.", 1),
			kinds:  "",
			all:    strings.Replace(mergeBase, "Bram cooks.", "Bram cooks a stew.", 1)},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := NewMerge(mergeBase, c.ours, c.theirs, mergeOpts)
			var kinds []string
			for _, ch := range m.Changes {
				k := []string{"edit", "cut", "added", "moved"}[ch.Kind] + "/" + []string{"same", "removed", "added", "changed"}[ch.Row]
				if ch.Conflict {
					k += " c"
				}
				kinds = append(kinds, k)
			}
			if got := strings.Join(kinds, " "); got != c.kinds {
				t.Errorf("changes %q, want %q", got, c.kinds)
			}
			if m.Kept != c.kept {
				t.Errorf("kept %d, want %d", m.Kept, c.kept)
			}
			// nothing taken: ours, every line where it was
			got, lm, _ := m.Apply(nil, nil)
			if got != c.ours {
				t.Errorf("nothing taken changed it:\n%s", got)
			}
			for i := range lm {
				if lm[i] != i {
					t.Errorf("nothing taken: line %d → %d", i, lm[i])
					break
				}
			}
			take := map[int]bool{}
			for _, ch := range m.Changes {
				take[ch.ID] = true
			}
			got, lm, _ = m.Apply(take, nil)
			if got != c.all {
				t.Errorf("all taken:\n%s\nwant:\n%s", got, c.all)
			}
			// a line kept as it was is where the map says
			ol, nl := strings.Split(c.ours, "\n"), strings.Split(got, "\n")
			for i, n := range lm {
				if n >= 0 && (n >= len(nl) || nl[n] != ol[i]) {
					t.Errorf("line %d %q mapped to %d", i, ol[i], n)
				}
			}
		})
	}
}

// Take and Leave one by one: only what is taken is written.
func TestMergeTakeLeave(t *testing.T) {
	theirs := strings.Replace(strings.Replace(mergeBase, "Bram cooks.", "Bram cooks a stew.", 1), "Tom waits by the door.", "Tom waits.", 1)
	m := NewMerge(mergeBase, mergeBase, theirs, mergeOpts)
	if len(m.Changes) != 2 || len(m.Scenes) != 2 {
		t.Fatalf("%d changes in %d scenes", len(m.Changes), len(m.Scenes))
	}
	if m.Scenes[0].Heading != "INT. KITCHEN - DAY" || m.Scenes[1].Heading != "INT. SHED - DAY" {
		t.Errorf("scenes %+v", m.Scenes)
	}
	got, _, _ := m.Apply(map[int]bool{1: true}, nil)
	if want := strings.Replace(mergeBase, "Tom waits by the door.", "Tom waits.", 1); got != want {
		t.Errorf("one taken:\n%s", got)
	}
	got, _, _ = m.Apply(nil, map[int]string{0: "Bram burns the stew."})
	if want := strings.Replace(mergeBase, "Bram cooks.", "Bram burns the stew.", 1); got != want {
		t.Errorf("edited:\n%s", got)
	}
	if n := Taken(map[int]bool{0: true, 1: false}, map[int]string{1: "x"}); n != 2 {
		t.Errorf("taken %d", n)
	}
}

// A copy without the links: no change for them, and taking a change
// keeps ours' link; what Accolade keeps after the body stays.
func TestMergeCarriesLinks(t *testing.T) {
	ours := strings.Replace(mergeBase, "Rain falls on the roses.", `Rain falls on the roses. [[from: "It rained."]]`, 1) + "\n/* Notes\nkeep\n*/\n"
	base := ours
	theirs := strings.Replace(mergeBase, "Rain falls on the roses.", "Rain falls on the dead roses.", 1)
	opts := mergeOpts
	opts.Body = func(s string) int {
		if i := strings.Index(s, "/* Notes"); i >= 0 {
			return i
		}
		return len(s)
	}
	m := NewMerge(base, ours, theirs, opts)
	if len(m.Changes) != 1 {
		t.Fatalf("changes %+v", m.Changes)
	}
	if c := m.Changes[0]; !strings.HasSuffix(c.Theirs, `[[from: "It rained."]]`) {
		t.Errorf("theirs as taken %q", c.Theirs)
	}
	got, _, pos := m.Apply(map[int]bool{0: true}, nil)
	want := strings.Replace(ours, "on the roses.", "on the dead roses.", 1)
	if got != want {
		t.Errorf("taken:\n%s\nwant:\n%s", got, want)
	}
	if l := strings.Count(ours[:strings.Index(ours, "Rain falls")], "\n"); pos[l] != l {
		t.Errorf("the changed line went to %d", pos[l])
	}
}

// The base is the version most like theirs.
func TestResemblance(t *testing.T) {
	theirs := strings.Replace(mergeBase, "Bram cooks.", "Bram cooks a stew.", 1)
	later := mergeBase + "\nINT. ATTIC - DAY\n\nDust.\n"
	s1, o1 := Resemblance(mergeBase, theirs, mergeOpts)
	s2, o2 := Resemblance(later, theirs, mergeOpts)
	if s1 != s2 || o1 >= o2 {
		t.Errorf("base %d/%d, later %d/%d", s1, o1, s2, o2)
	}
}

func TestSplitAt(t *testing.T) {
	text := "A\nB\n\n  \nC\n"
	es := SplitAt(text, nil)
	if len(es) != 2 || es[0].Text != "A\nB" || es[0].Start != 0 || es[0].TextEnd != 3 || es[0].End != 8 || es[1].Start != 8 || es[1].End != len(text) {
		t.Errorf("%+v", es)
	}
}

// A source that joins the script's elements (a print read back) or runs a
// heading into the action: re-split at the script's breaks, the same
// words are no difference, a changed word is one, in its own element.
func TestMergeResplit(t *testing.T) {
	o := MergeOptions{IsHeading: func(l string) bool { return mergeHeading(strings.ToUpper(l)) }, Key: strings.ToLower, Resplit: true}
	script := "INT. A - DAY\n\nBram waits.\n\nBRAM\nHello there.\n\nHe sits.\n\nint. b - night Tom runs.\nHe looks back.\n\nTom falls.\n"
	joined := "INT. A - DAY\n\nBram waits.\nBRAM\nHello there.\nHe sits.\n\nINT. B - NIGHT\n\nTom runs.\nHe looks back.\nTom falls.\n"
	plain := o
	plain.Resplit = false
	if n := len(NewMerge(script, script, joined, plain).Changes); n == 0 {
		t.Fatal("element by element the joins should differ")
	}
	m := NewMerge(script, script, strings.Replace(joined, "Hello there.", "Hello there, friend.", 1), o)
	var rows []string
	for _, c := range m.Changes {
		rows = append(rows, c.Ours+" → "+c.Theirs)
	}
	// the heading run into the action: one change, taken whole
	want := []string{"BRAM\nHello there. → BRAM\nHello there, friend.", "int. b - night Tom runs.\nHe looks back. → INT. B - NIGHT\n\nTom runs.\nHe looks back."}
	if strings.Join(rows, " | ") != strings.Join(want, " | ") {
		t.Fatalf("changes %q,\nwant %q", rows, want)
	}
	all := map[int]bool{}
	for _, c := range m.Changes {
		all[c.ID] = true
	}
	got, _, _ := m.Apply(all, nil)
	if strings.Count(got, "Tom runs") != 1 || strings.Count(got, "He sits") != 1 || !strings.Contains(got, "BRAM\nHello there, friend.\n\nHe sits.") {
		t.Errorf("taken:\n%s", got)
	}
	if n := len(NewMerge(got, got, joined, o).Changes); n != 1 { // only the friend, now the script's
		t.Errorf("after taking: %d changes, want 1", n)
	}
	if n := len(NewMerge(script, script, joined, o).Changes); n != 1 { // the heading only
		t.Errorf("the same words: %d changes, want 1 (the heading)", n)
	}
	// punctuation is compared
	for _, p := range [][2]string{{"He sits.", "He sits!"}, {"Bram waits.", "Bram waits"}} {
		m := NewMerge(got, got, strings.Replace(joined, p[0], p[1], 1), o)
		found := false
		for _, c := range m.Changes {
			found = found || c.Theirs == p[1]
		}
		if len(m.Changes) != 2 || !found {
			var rows []string
			for _, c := range m.Changes {
				rows = append(rows, c.Ours+" → "+c.Theirs)
			}
			t.Errorf("%q for %q: %q", p[1], p[0], rows)
		}
	}
}
