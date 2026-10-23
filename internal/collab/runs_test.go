package collab

import (
	"reflect"
	"testing"
)

// Runs give back the ops they were made of, in order.
func TestRuns(t *testing.T) {
	d := New("a")
	ops := d.SetText("INT. BARN - DAY\n")
	ops = append(ops, d.SetText("INT. BARN - NIGHT\n")...)          // deletes and inserts
	ops = append(ops, d.SetText("EXT. FIELD - NIGHT\nWind ✓\n")...) // runes outside ASCII
	ops = append(ops, New("b").Insert(0, "x")...)                   // another replica
	runs := runsOf(ops)
	var back []Op
	for _, r := range runs {
		back = append(back, r.ops()...)
	}
	if !reflect.DeepEqual(back, ops) {
		t.Fatalf("got %v\nwant %v", back, ops)
	}
	if len(runs) > 8 {
		t.Errorf("%d runs for %d ops", len(runs), len(ops))
	}
}
