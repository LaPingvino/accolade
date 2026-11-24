package structure

import (
	"strings"
	"testing"
)

const script = `Title: Test
Author: Me

# ACT ONE

INT. BARN - DAY

Rain on the roof.

ANNA
(quietly)
It leaks.
Again.

BOB
Then fix it.

EXT. FIELD - DAY

Wind.

# ACT TWO

## Night

INT. HOUSE - NIGHT

Fire.

## Morning

EXT. FIELD - DAY

Dawn.
`

// lineOf is the first line of text that is key (trimmed).
func lineOf(t *testing.T, text, key string) int {
	t.Helper()
	for i, l := range Lines(text) {
		if strings.TrimSpace(l) == key {
			return i
		}
	}
	t.Fatalf("no line %q", key)
	return -1
}

// rep is s with old (once, it must be there) as nu.
func rep(t *testing.T, s, old, nu string) string {
	t.Helper()
	if strings.Count(s, old) != 1 {
		t.Fatalf("%q is in the script %d times", old, strings.Count(s, old))
	}
	return strings.Replace(s, old, nu, 1)
}

func do(text string, op Op, a, b int) (string, Result, error) {
	tr := Parse(Plain(Lines(text)))
	res, err := tr.Do(op, a, b, Plain)
	if err != nil {
		return text, res, err
	}
	return Text(res.Lines), res, nil
}

// Every move on scenes, sections, speeches and lines, by the cursor's
// line (and a selection's last): what it makes, or why it will not.
func TestMoves(t *testing.T) {
	barn := "Rain on the roof.\n\nANNA\n(quietly)\nIt leaks.\nAgain.\n\nBOB\nThen fix it.\n\n"
	for _, c := range []struct {
		name   string
		op     Op
		at, to string // the cursor's line, a selection's last
		want   func(t *testing.T) string
		refuse string
	}{
		{name: "speech up", op: MoveUp, at: "BOB", want: func(t *testing.T) string {
			return rep(t, script, barn, "Rain on the roof.\n\nBOB\nThen fix it.\n\nANNA\n(quietly)\nIt leaks.\nAgain.\n\n")
		}},
		{name: "speech down, from a line in it", op: MoveDown, at: "It leaks.", want: func(t *testing.T) string {
			return rep(t, script, barn, "Rain on the roof.\n\nBOB\nThen fix it.\n\nANNA\n(quietly)\nIt leaks.\nAgain.\n\n")
		}},
		{name: "first paragraph up", op: MoveUp, at: "Rain on the roof.", refuse: "first block"},
		{name: "last speech down", op: MoveDown, at: "Then fix it.", refuse: "into the next scene"},
		{name: "scene down", op: MoveDown, at: "INT. BARN - DAY", want: func(t *testing.T) string {
			return rep(t, script, "INT. BARN - DAY\n\n"+barn+"EXT. FIELD - DAY\n\nWind.\n\n# ACT TWO",
				"EXT. FIELD - DAY\n\nWind.\n\nINT. BARN - DAY\n\n"+barn+"# ACT TWO")
		}},
		{name: "last paragraph down", op: MoveDown, at: "Wind.", refuse: "last block"},
		{name: "section down with all it holds", op: MoveDown, at: "# ACT ONE", want: func(t *testing.T) string {
			one := script[strings.Index(script, "# ACT ONE"):strings.Index(script, "# ACT TWO")]
			two := script[strings.Index(script, "# ACT TWO"):]
			return script[:strings.Index(script, "# ACT ONE")] + two + "\n" + strings.TrimSuffix(one, "\n")
		}},
		{name: "subsection down", op: MoveDown, at: "## Night", want: func(t *testing.T) string {
			return rep(t, script, "## Night\n\nINT. HOUSE - NIGHT\n\nFire.\n\n## Morning\n\nEXT. FIELD - DAY\n\nDawn.\n",
				"## Morning\n\nEXT. FIELD - DAY\n\nDawn.\n\n## Night\n\nINT. HOUSE - NIGHT\n\nFire.\n")
		}},
		{name: "lines in a speech up", op: MoveUp, at: "It leaks.", to: "Again.", want: func(t *testing.T) string {
			return rep(t, script, "(quietly)\nIt leaks.\nAgain.", "It leaks.\nAgain.\n(quietly)")
		}},
		{name: "the cue does not move as a line", op: MoveUp, at: "(quietly)", to: "It leaks.", refuse: "first line"},
		{name: "section out of its parent", op: Promote, at: "## Night", want: func(t *testing.T) string {
			return rep(t, script, "## Night\n\nINT. HOUSE - NIGHT\n\nFire.\n\n## Morning\n\nEXT. FIELD - DAY\n\nDawn.\n",
				"## Morning\n\nEXT. FIELD - DAY\n\nDawn.\n\n# Night\n\nINT. HOUSE - NIGHT\n\nFire.\n")
		}},
		{name: "the last subsection out: a level up, where it is", op: Promote, at: "## Morning", want: func(t *testing.T) string {
			return rep(t, script, "## Morning", "# Morning")
		}},
		{name: "top section up", op: Promote, at: "# ACT ONE", refuse: "top already"},
		{name: "section into the one before", op: Demote, at: "## Morning", want: func(t *testing.T) string {
			return rep(t, script, "## Morning", "### Morning")
		}},
		{name: "section in, with its sections", op: Demote, at: "# ACT TWO", want: func(t *testing.T) string {
			return rep(t, rep(t, rep(t, script, "# ACT TWO", "## ACT TWO"), "## Night", "### Night"), "## Morning", "### Morning")
		}},
		{name: "first section in: a level down where it is", op: Demote, at: "# ACT ONE", want: func(t *testing.T) string {
			return rep(t, script, "# ACT ONE", "## ACT ONE")
		}},
		{name: "scene out of its section", op: Promote, at: "INT. BARN - DAY", want: func(t *testing.T) string {
			return rep(t, script, "# ACT ONE\n\nINT. BARN - DAY\n\n"+barn, "INT. BARN - DAY\n\n"+barn+"# ACT ONE\n\n")
		}},
		{name: "scene into the next section", op: Demote, at: "EXT. FIELD - DAY", want: func(t *testing.T) string {
			return rep(t, script, "EXT. FIELD - DAY\n\nWind.\n\n# ACT TWO\n\n", "# ACT TWO\n\nEXT. FIELD - DAY\n\nWind.\n\n")
		}},
		{name: "the same with Alt+Shift+Down", op: IntoNext, at: "EXT. FIELD - DAY", want: func(t *testing.T) string {
			return rep(t, script, "EXT. FIELD - DAY\n\nWind.\n\n# ACT TWO\n\n", "# ACT TWO\n\nEXT. FIELD - DAY\n\nWind.\n\n")
		}},
		{name: "a scene in no section", op: Promote, at: "INT. A - DAY", refuse: "no section"},
		{name: "speech into the next scene", op: IntoNext, at: "BOB", want: func(t *testing.T) string {
			return rep(t, script, "BOB\nThen fix it.\n\nEXT. FIELD - DAY\n\nWind.", "EXT. FIELD - DAY\n\nBOB\nThen fix it.\n\nWind.")
		}},
		{name: "paragraph into the scene before", op: IntoPrev, at: "Wind.", want: func(t *testing.T) string {
			return rep(t, script, "Then fix it.\n\nEXT. FIELD - DAY\n\nWind.\n", "Then fix it.\n\nWind.\n\nEXT. FIELD - DAY\n")
		}},
		{name: "speech into the scene before the first", op: IntoPrev, at: "Rain on the roof.", refuse: "No scene before"},
		{name: "barf: the last speech out into the next scene", op: Barf, at: "Rain on the roof.", want: func(t *testing.T) string {
			return rep(t, script, "BOB\nThen fix it.\n\nEXT. FIELD - DAY\n\nWind.", "EXT. FIELD - DAY\n\nBOB\nThen fix it.\n\nWind.")
		}},
		{name: "slurp: the next scene's first paragraph in", op: Slurp, at: "INT. BARN - DAY", want: func(t *testing.T) string {
			return rep(t, script, "Then fix it.\n\nEXT. FIELD - DAY\n\nWind.\n", "Then fix it.\n\nWind.\n\nEXT. FIELD - DAY\n")
		}},
		{name: "barf in the last scene", op: Barf, at: "Dawn.", refuse: "No scene after"},
		{name: "raise the last line", op: Promote, at: "Again.", want: func(t *testing.T) string {
			return rep(t, script, "It leaks.\nAgain.\n", "It leaks.\n\nAgain.\n")
		}},
		{name: "raise the first line: before the speech, its parenthetical with it", op: Promote, at: "It leaks.", want: func(t *testing.T) string {
			return rep(t, script, "ANNA\n(quietly)\nIt leaks.\nAgain.", "!(quietly)\nIt leaks.\n\nANNA\nAgain.")
		}},
		{name: "raise all a speech's lines", op: Promote, at: "(quietly)", to: "Again.", refuse: "all the speech's lines"},
		{name: "raise on the cue", op: Promote, at: "ANNA", refuse: "cursor on the line"},
		{name: "lower action into the speech after", op: Demote, at: "Rain on the roof.", want: func(t *testing.T) string {
			return rep(t, script, "Rain on the roof.\n\nANNA\n", "ANNA\nRain on the roof.\n")
		}},
		{name: "lower with no speech beside", op: Demote, at: "Dawn.", refuse: "No speech right beside"},
		{name: "transpose two speakers' lines", op: Transpose, at: "Then fix it.", want: func(t *testing.T) string {
			return rep(t, rep(t, script, "ANNA\n(quietly)", "BOB\n(quietly)"), "BOB\nThen fix it.", "ANNA\nThen fix it.")
		}},
		{name: "transpose with no speech before", op: Transpose, at: "ANNA", refuse: "No speech right before"},
		{name: "nothing on the title page", op: MoveDown, at: "Title: Test", refuse: "title page"},
	} {
		t.Run(c.name, func(t *testing.T) {
			text := script
			if strings.Contains(c.refuse, "no section") {
				text = "INT. A - DAY\n\nOne.\n"
			}
			a := lineOf(t, text, c.at)
			b := a
			if c.to != "" {
				b = lineOf(t, text, c.to)
			}
			got, _, err := do(text, c.op, a, b)
			if c.refuse != "" {
				if err == nil || !strings.Contains(err.Error(), c.refuse) {
					t.Fatalf("want refused (%q), got %v:\n%s", c.refuse, err, got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if want := c.want(t); got != want {
				t.Errorf("got\n%s\nwant\n%s", got, want)
			}
		})
	}
}

// Raising a line from the middle of a speech splits it, the rest under
// the cue again; lowering it joins them back: one is the other's undoing.
func TestRaiseLowerBack(t *testing.T) {
	const s = "INT. A - DAY\n\nANNA\nOne.\nTwo.\nThree.\n"
	up, _, err := do(s, Promote, 4, 4)
	if err != nil || up != "INT. A - DAY\n\nANNA\nOne.\n\nTwo.\n\nANNA (CONT'D)\nThree.\n" {
		t.Fatalf("raised (%v):\n%s", err, up)
	}
	down, _, err := do(up, Demote, lineOf(t, up, "Two."), lineOf(t, up, "Two."))
	if err != nil || down != s {
		t.Fatalf("lowered (%v):\n%s", err, down)
	}
}

// Two speeches side by side (the second cue's ^) are one: they move
// together, and change speakers with each other.
func TestDualDialogue(t *testing.T) {
	const s = "INT. A - DAY\n\nANNA\nYes.\n\nBOB ^\nNo.\n\nCARL\nMaybe.\n"
	got, _, err := do(s, MoveUp, 8, 8)
	if err != nil || got != "INT. A - DAY\n\nCARL\nMaybe.\n\nANNA\nYes.\n\nBOB ^\nNo.\n" {
		t.Fatalf("up (%v):\n%s", err, got)
	}
	got, _, err = do(s, Transpose, 5, 5)
	if err != nil || got != "INT. A - DAY\n\nBOB\nYes.\n\nANNA ^\nNo.\n\nCARL\nMaybe.\n" {
		t.Fatalf("transposed (%v):\n%s", err, got)
	}
	if _, _, err := do(s, Promote, 6, 6); err == nil {
		t.Error("a line out of side-by-side speeches")
	}
}

// tailed reads a text whose body ends at its /* Notes block.
func tailed(lines []string) Script {
	s := Plain(lines)
	for i, l := range lines {
		if strings.HasPrefix(l, "/* Notes") {
			s.End = i
			s.Heads = nil
			for _, h := range Plain(lines).Heads {
				if h.Line < i {
					s.Heads = append(s.Heads, h)
				}
			}
		}
	}
	return s
}

// The blocks at the script's end stay there, whatever moves.
func TestTailStays(t *testing.T) {
	const s = "INT. A - DAY\n\nOne.\n\nINT. B - DAY\n\nTwo.\n\n/* Notes\nkeep\n\nthis\n*/\n"
	tr := Parse(tailed(Lines(s)))
	if _, err := tr.Do(MoveDown, 4, 4, tailed); err == nil {
		t.Error("the last scene moved past the end")
	}
	if _, err := tr.Do(IntoNext, 6, 6, tailed); err == nil {
		t.Error("the last paragraph into the notes")
	}
	want := "INT. B - DAY\n\nTwo.\n\nINT. A - DAY\n\nOne.\n\n/* Notes\nkeep\n\nthis\n*/\n"
	for name, f := range map[string]func() (Result, error){
		"up":         func() (Result, error) { return tr.Do(MoveUp, 4, 4, tailed) },
		"after":      func() (Result, error) { return tr.Move(Range{0, 0}, 4, true, tailed) },
		"before":     func() (Result, error) { return tr.Move(Range{4, 4}, 0, false, tailed) },
		"swap":       func() (Result, error) { return tr.Swap(Range{0, 0}, Range{4, 4}, tailed) },
		"swap other": func() (Result, error) { return tr.Swap(Range{4, 4}, Range{0, 0}, tailed) },
	} {
		res, err := f()
		if err != nil || Text(res.Lines) != want {
			t.Errorf("%s (%v):\n%s", name, err, Text(res.Lines))
		}
	}
}

// The picker's moves: ranges of scenes, sections, swapped or moved
// before or after another; what would put a unit in another's section is
// refused.
func TestRanges(t *testing.T) {
	tr := Parse(Plain(Lines(script)))
	l := func(k string) int { return lineOf(t, script, k) }
	res, err := tr.Swap(Range{l("INT. BARN - DAY"), l("INT. BARN - DAY")}, Range{l("INT. HOUSE - NIGHT"), l("INT. HOUSE - NIGHT")}, Plain)
	if err != nil {
		t.Fatal(err)
	}
	barn := "INT. BARN - DAY\n\nRain on the roof.\n\nANNA\n(quietly)\nIt leaks.\nAgain.\n\nBOB\nThen fix it."
	house := "INT. HOUSE - NIGHT\n\nFire."
	if want := rep(t, rep(t, script, barn, "@@"), house, barn); Text(res.Lines) != strings.Replace(want, "@@", house, 1) {
		t.Errorf("swapped:\n%s", Text(res.Lines))
	}
	res, err = tr.Move(Range{l("INT. BARN - DAY"), l("EXT. FIELD - DAY")}, l("INT. HOUSE - NIGHT"), true, Plain)
	if err != nil {
		t.Fatal(err)
	}
	field := "EXT. FIELD - DAY\n\nWind."
	want := rep(t, rep(t, script, barn+"\n\n"+field+"\n\n", ""), house+"\n", house+"\n\n"+barn+"\n\n"+field+"\n")
	if Text(res.Lines) != want {
		t.Errorf("moved:\n%s\nwant\n%s", Text(res.Lines), want)
	}
	if _, err := tr.Move(Range{l("## Morning"), l("## Morning")}, l("INT. HOUSE - NIGHT"), false, Plain); err == nil || !strings.Contains(err.Error(), "another section") {
		t.Errorf("a section before a scene, taking it in: %v", err)
	}
	if _, err := tr.Swap(Range{l("INT. BARN - DAY"), l("INT. BARN - DAY")}, Range{l("## Morning"), l("## Morning")}, Plain); err == nil {
		t.Error("a scene swapped with a section")
	}
	if _, err := tr.Move(Range{l("# ACT ONE"), l("# ACT ONE")}, l("INT. BARN - DAY"), true, Plain); err == nil {
		t.Error("a section into itself")
	}
	if _, err := tr.Move(Range{l("INT. BARN - DAY"), l("INT. HOUSE - NIGHT")}, l("## Morning"), false, Plain); err == nil {
		t.Error("a range across sections")
	}
	res, err = tr.Move(Range{l("## Morning"), l("## Morning")}, l("## Night"), false, Plain)
	if err != nil {
		t.Fatal(err)
	}
	if want := rep(t, rep(t, script, "\n\n## Morning\n\nEXT. FIELD - DAY\n\nDawn.", ""), "## Night", "## Morning\n\nEXT. FIELD - DAY\n\nDawn.\n\n## Night"); Text(res.Lines) != want {
		t.Errorf("a section before another:\n%s", Text(res.Lines))
	}
}

// What rides with its lines: source links, scene numbers (locked ones
// too); every line that moved keeps its text, and Map says where it went.
func TestAnchorsRide(t *testing.T) {
	const s = "INT. A - DAY #12A#\n\nShe opens it. [[from: #:~:text=She opened the door]]\n\nINT. B - DAY #13#\n\nShut.\n"
	got, res, err := do(s, MoveDown, 0, 0)
	if err != nil || got != "INT. B - DAY #13#\n\nShut.\n\nINT. A - DAY #12A#\n\nShe opens it. [[from: #:~:text=She opened the door]]\n" {
		t.Fatalf("(%v):\n%s", err, got)
	}
	old := Lines(s)
	for i, j := range res.Map {
		if j < 0 && strings.TrimSpace(old[i]) != "" {
			t.Errorf("line %d (%q) lost", i, old[i])
		}
		if j >= 0 && res.Lines[j] != old[i] {
			t.Errorf("line %d went to %d: %q", i, j, res.Lines[j])
		}
	}
	if res.First != 4 || res.Last != 6 {
		t.Errorf("what moved: %d-%d", res.First, res.Last)
	}
}

// Expanding the selection by units: a line, its speech, its scene, its
// section, the body.
func TestExpand(t *testing.T) {
	tr := Parse(Plain(Lines(script)))
	l := func(k string) int { return lineOf(t, script, k) }
	a, b := l("It leaks."), l("It leaks.")
	var got [][2]int
	for {
		na, nb, ok := tr.Expand(a, b)
		if !ok {
			break
		}
		a, b = na, nb
		got = append(got, [2]int{a, b})
	}
	want := [][2]int{{l("ANNA"), l("Again.")}, {l("INT. BARN - DAY"), l("Then fix it.")}, {l("# ACT ONE"), l("Wind.")}, {l("# ACT ONE"), l("Dawn.")}}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("step %d: %v, want %v", i, got[i], want[i])
		}
	}
}

// A section's own blocks (synopses, notes, action written before its
// scenes) go into the section before or after with Alt+Shift+Up and
// Down: at the end of its own blocks, or the start of them; a scene's
// still go into scenes.
func TestIntoSectionBlocks(t *testing.T) {
	const s = "Title: T\n\n# One\n\n= a\n\n= b\n\nINT. X - DAY\n\nAction.\n\n# Two\n\n= c\n\n= d\n\n# Three\n\n= e\n"
	for _, c := range []struct {
		name   string
		op     Op
		at     string
		want   string
		refuse string
	}{
		{name: "the first of Two into One, after its own", op: IntoPrev, at: "= c",
			want: "Title: T\n\n# One\n\n= a\n\n= b\n\n= c\n\nINT. X - DAY\n\nAction.\n\n# Two\n\n= d\n\n# Three\n\n= e\n"},
		{name: "the last of Two into Three, after its own synopsis", op: IntoNext, at: "= d",
			want: "Title: T\n\n# One\n\n= a\n\n= b\n\nINT. X - DAY\n\nAction.\n\n# Two\n\n= c\n\n# Three\n\n= e\n\n= d\n"},
		{name: "One's into Two", op: IntoNext, at: "= b",
			want: "Title: T\n\n# One\n\n= a\n\nINT. X - DAY\n\nAction.\n\n# Two\n\n= c\n\n= b\n\n= d\n\n# Three\n\n= e\n"},
		{name: "a scene's paragraph: no scene after", op: IntoNext, at: "Action.", refuse: "No scene after"},
		{name: "none before One", op: IntoPrev, at: "= a", refuse: "No section before"},
		{name: "none after Three", op: IntoNext, at: "= e", refuse: "No section after"},
		{name: "a scene's paragraph still into scenes", op: IntoPrev, at: "Action.", refuse: "No scene before"},
	} {
		t.Run(c.name, func(t *testing.T) {
			l := lineOf(t, s, c.at)
			got, _, err := do(s, c.op, l, l)
			if c.refuse != "" {
				if err == nil || !strings.Contains(err.Error(), c.refuse) {
					t.Fatalf("want a refusal %q, got %v:\n%s", c.refuse, err, got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, c.want)
			}
		})
	}
	// the first block of a section: the hint names the section before
	l := lineOf(t, s, "= c")
	if _, _, err := do(s, MoveUp, l, l); err == nil || !strings.Contains(err.Error(), "into the section before") {
		t.Errorf("the hint: %v", err)
	}
}
