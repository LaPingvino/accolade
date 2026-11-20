package scriptdiff

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

func heading(l string) bool { return strings.HasPrefix(l, "INT.") || strings.HasPrefix(l, "EXT.") }

// apply checks the edit script turns a into b.
func apply(t *testing.T, a, b []string, ops []Kind) {
	t.Helper()
	var got []string
	i, j := 0, 0
	for _, op := range ops {
		switch op {
		case Same:
			if a[i] != b[j] {
				t.Fatalf("same %q %q", a[i], b[j])
			}
			got = append(got, a[i])
			i, j = i+1, j+1
		case Removed:
			i++
		case Added:
			got = append(got, b[j])
			j++
		}
	}
	if i != len(a) || j != len(b) || strings.Join(got, "|") != strings.Join(b, "|") {
		t.Fatalf("ops %v: a %d/%d b %d/%d", ops, i, len(a), j, len(b))
	}
}

func TestDiffRandom(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for n := 0; n < 300; n++ {
		var a, b []string
		for range r.Intn(30) {
			a = append(a, fmt.Sprint(r.Intn(6)))
		}
		for range r.Intn(30) {
			b = append(b, fmt.Sprint(r.Intn(6)))
		}
		ops := diff(a, b)
		apply(t, a, b, ops)
	}
}

func TestDiffMinimal(t *testing.T) {
	a := strings.Split("a b c a b b a", " ")
	b := strings.Split("c b a b a c", " ")
	edits := 0
	for _, op := range diff(a, b) {
		if op != Same {
			edits++
		}
	}
	if edits != 5 { // Myers' paper's example: D = 5
		t.Errorf("%d edits", edits)
	}
}

func TestDiffTooUnlike(t *testing.T) {
	var a, b []string
	for i := range 2500 {
		a = append(a, fmt.Sprint("a", i))
		b = append(b, fmt.Sprint("b", i))
	}
	ops := diff(a, b)
	apply(t, a, b, ops)
}

const oldScript = `Title: X

INT. HOUSE - DAY

Anna opens the door.

ANNA
Hello there.

EXT. GARDEN - NIGHT

Rain falls.

INT. SHED - DAY

Tom waits.
`

const newScript = `Title: X

INT. HOUSE - DAY

Anna opens the old door slowly.

ANNA
Hello there.

INT. SHED - DAY

Tom waits.

TOM
Who's there?
`

func TestDiffScript(t *testing.T) {
	rows := Diff(Split(oldScript, heading), Split(newScript, heading))
	removed, added, changed := Count(rows)
	if removed != 2 || added != 1 || changed != 1 {
		t.Fatalf("removed %d added %d changed %d: %+v", removed, added, changed, rows)
	}
	scenes := Group(rows)
	var got []string
	for _, s := range scenes {
		got = append(got, fmt.Sprintf("%s:%v", s.Heading, s.Changed))
	}
	if want := ":false INT. HOUSE - DAY:true EXT. GARDEN - NIGHT:true INT. SHED - DAY:true"; strings.Join(got, " ") != want {
		t.Errorf("scenes %q", strings.Join(got, " "))
	}
	for _, r := range rows {
		if r.Kind == Changed {
			var b strings.Builder
			for _, w := range r.Words {
				fmt.Fprintf(&b, "%d[%s]", w.Kind, w.Text)
			}
			if b.String() != "0[Anna opens the ]2[old ]0[door]2[ slowly]0[.]" {
				t.Errorf("words %s", b.String())
			}
		}
	}
}

func TestSplit(t *testing.T) {
	es := Split("A\r\nB  \n\n\n\nINT. X - DAY\n", heading)
	if len(es) != 2 || es[0].Text != "A\nB" || !es[1].Heading {
		t.Errorf("%+v", es)
	}
}
