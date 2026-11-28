// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package structure

import (
	"strings"
	"testing"
)

// Reorder sorts the scenes inside each section, as one move: the
// sections and the blank lines stay, a scene never leaves its section.
func TestReorder(t *testing.T) {
	const s = `# ONE

INT. A - DAY

a.

INT. B - DAY

b.

INT. C - DAY

c.

# TWO

INT. D - DAY

d.

INT. E - DAY

e.
`
	tr := Parse(Plain(Lines(s)))
	rank := map[int]int{
		lineOf(t, s, "INT. C - DAY"): 0, lineOf(t, s, "INT. A - DAY"): 1, // B unranked: last
		lineOf(t, s, "INT. E - DAY"): 0, lineOf(t, s, "INT. D - DAY"): 1,
	}
	res, err := tr.Reorder(rank, Plain)
	if err != nil {
		t.Fatal(err)
	}
	got := Text(res.Lines)
	want := "# ONE\n\nINT. C - DAY\n\nc.\n\nINT. A - DAY\n\na.\n\nINT. B - DAY\n\nb.\n\n# TWO\n\nINT. E - DAY\n\ne.\n\nINT. D - DAY\n\nd.\n"
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	if !strings.Contains(res.Msg, "5 scenes") {
		t.Errorf("message %q", res.Msg)
	}
	// each old line went where its text is now
	for i, j := range res.Pos {
		if j < 0 || res.Lines[j] != tr.S.Lines[i] {
			t.Fatalf("line %d (%q) mapped to %d", i, tr.S.Lines[i], j)
		}
	}
	// in order already: in place
	if _, err := Parse(Plain(Lines(want))).Reorder(map[int]int{}, Plain); err != InPlace {
		t.Errorf("no ranks: %v, want InPlace", err)
	}
}
