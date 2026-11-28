// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

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
	var back, want []oldOp
	for _, r := range runs {
		back = append(back, runes(r)...)
	}
	for _, op := range ops {
		want = append(want, runes(op)...)
	}
	if !reflect.DeepEqual(back, want) {
		t.Fatalf("got %v\nwant %v", back, want)
	}
	if len(runs) > 8 {
		t.Errorf("%d runs for %d ops", len(runs), len(ops))
	}
}
