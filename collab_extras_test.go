// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"slices"
	"strings"
	"testing"
)

// The breakdown edited together: a's tags reach b, who joined with none;
// b's Untag reaches a; tagging different words apart (not connected), and
// the same new element at once, both are kept and end alike, one element;
// each side's file holds it.
func TestBreakdownTogether(t *testing.T) {
	script := "INT. BARN - DAY\n\nAnna lights the lamp and takes the rope.\n\nINT. LOFT - NIGHT\n\nThe lamp swings. A hat on a nail.\n"
	noTrackers(t)
	a, b, _ := sharedPair(t, script)
	tagNow(t, a, "lamp", 0, "Props")
	cut := connect(t, a, b)
	textIs(t, b, "b", script)
	waitFor(t, "b has a's tag", func() bool { return tagsAre(b, "lamp") })
	tagNow(t, b, "rope", 0, "Props")
	waitFor(t, "a has b's tag", func() bool { return tagsAre(a, "lamp rope") })
	selectWords(t, b, "lamp", 0)
	b.untagSelection()
	waitFor(t, "a without the lamp", func() bool { return tagsAre(a, "rope") })
	cut()
	tagNow(t, a, "hat", 0, "Wardrobe")
	tagNow(t, b, "nail", 0, "Set Dressing")
	tagNow(t, a, "lamp", 1, "Props") // a new lamp element on both sides, each its own id
	tagNow(t, b, "lamp", 0, "Props")
	connect(t, a, b)
	for _, w := range []*MainWindow{a, b} {
		waitFor(t, "all kept", func() bool { return tagsAre(w, "hat lamp lamp nail rope") })
	}
	if !slices.Equal(found(a), found(b)) {
		t.Errorf("not alike:\n%v\n%v", found(a), found(b))
	}
	for _, w := range []*MainWindow{a, b} {
		n := 0
		for _, e := range w.breakdownNow().Elements {
			if e.Name == "lamp" {
				n++
			}
		}
		if n != 1 {
			t.Errorf("%d lamp elements", n)
		}
		f, err := readBreakdown(w.extrasHome())
		if err != nil || len(f.Tags) < 5 {
			t.Errorf("kept: %d tags, %v", len(f.Tags), err)
		}
	}
}

// tagsAre reports whether w's tags found are on these words (sorted, one
// for each tag).
func tagsAre(w *MainWindow, words string) bool {
	var got []string
	for _, f := range found(w) {
		parts := strings.Split(f, "@")
		got = append(got, parts[1])
	}
	slices.Sort(got)
	return strings.Join(got, " ") == words
}

// The schedule edited together: a's days and strips reach b; apart, a
// moves one strip to another day and b puts another on a day: both moves
// kept, on both sides alike; a day taken away on one side sends its
// strips to the pool on both.
func TestScheduleTogether(t *testing.T) {
	noTrackers(t)
	a, b, _ := sharedPair(t, schedScript)
	sa := board(t, a)
	a.addShootDay()
	a.addShootDay()
	sa.selectScene(0, false)
	sa.moveSelected(0, 0) // kitchen on day 1
	sa.selectScene(3, false)
	sa.moveSelected(0, 1) // garden after it
	cut := connect(t, a, b)
	textIs(t, b, "b", schedScript)
	waitFor(t, "b has a's days", func() bool { return strings.Join(days(b), "|") == "kitchen,garden|" })
	sb := board(t, b)
	cut()
	sa.selectScene(3, false)
	sa.moveSelected(1, 0) // a: garden to day 2
	sb.selectScene(2, false)
	sb.moveSelected(0, 0) // b: the shed before the kitchen on day 1
	connect(t, a, b)
	for _, w := range []*MainWindow{a, b} {
		waitFor(t, "both moves on both", func() bool { return strings.Join(days(w), "|") == "shed,kitchen|garden" })
	}
	// a day taken away: its strips to the pool on both
	a.changeSchedule(func(f *scheduleFile, _ *bdText, _ schedPlaced) bool { f.removeDay(1); return true })
	waitFor(t, "b without day 2", func() bool { return strings.Join(days(b), "|") == "shed,kitchen" })
}

// Places between: kept where they hold, new ones between their
// neighbours; a strip moved changes its place only.
func TestPlacesBetween(t *testing.T) {
	got := placesBetween([]float64{1, 2, 3, 0}, []bool{true, true, true, false})
	if got[0] != 1 || got[1] != 2 || got[2] != 3 || got[3] != 4 {
		t.Errorf("appended: %v", got)
	}
	got = placesBetween([]float64{3, 1, 2}, []bool{true, true, true}) // the last moved to the front
	if got[1] != 1 && got[2] != 2 || !(got[0] < got[1] && got[1] < got[2]) {
		t.Errorf("moved to the front: %v", got)
	}
	got = placesBetween([]float64{0, 1, 2}, []bool{false, true, true}) // a new one first
	if got[0] <= 0 || got[0] >= 1 || got[1] != 1 || got[2] != 2 {
		t.Errorf("new first: %v", got)
	}
}
