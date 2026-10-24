package syntax

import (
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	script := []struct {
		line string
		kind Kind
	}{
		{"Title: The Barn", TitlePage},
		{"Contact:", TitlePage},
		{"    1 Writer's Lane", TitlePage},
		{"", Empty},
		{"FADE IN:", Action},
		{"", Empty},
		{"INT. BARN - DAY", SceneHeading},
		{"", Empty},
		{"Rain hammers the roof.", Action},
		{"", Empty},
		{"                                     JOHN (V.O.)", Character},
		{"                               (beat)", Parenthetical},
		{"                         It's coming.", Dialogue},
		{"", Empty},
		{"[[remember the dog]]", Note},
		{"# Act One", Section},
		{"= John finds out.", Synopsis},
		{"===", PageBreak},
		{"> THE END <", Centered},
		{"", Empty},
		{"CUT TO:", Transition},
		{"", Empty},
		{"> FADE OUT.", Transition},
		{"", Empty},
		{".FLASHBACK", SceneHeading},
		{"", Empty},
		{"BANG!", Action},
		{"", Empty},
		{"MARY", Action}, // no dialogue after it: action in all caps
	}
	var lines []string
	for _, l := range script {
		lines = append(lines, l.line)
	}
	got := Classify(lines)
	for i, l := range script {
		if got[i] != l.kind {
			t.Errorf("line %d %q: kind %d, want %d", i, l.line, got[i], l.kind)
		}
	}
}

func TestNoTitlePage(t *testing.T) {
	got := Classify(strings.Split("INT. LAB: NIGHT\n\nJOHN\nHi.", "\n"))
	want := []Kind{SceneHeading, Empty, Character, Dialogue}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: %d, want %d", i, got[i], want[i])
		}
	}
}

func TestClassifyTyping(t *testing.T) {
	lines := []string{"Anna waits.", "", "ANNA", "", "BRAM enters."}
	if k := ClassifyTyping(lines, nil, 3); k[2] != Character || k[3] != Dialogue {
		t.Errorf("after a name: %v", k)
	}
	if k := ClassifyTyping(lines, nil, 1); k[2] != Action {
		t.Errorf("elsewhere: %v", k)
	}
	lines = []string{"ANNA", "Hi.", ""}
	if k := ClassifyTyping(lines, nil, 2); k[2] != Empty {
		t.Errorf("after dialogue: %v", k)
	}
}

// Lyrics (~) are sung, under a name or on their own, and dialogue goes on
// after them.
func TestLyrics(t *testing.T) {
	kinds := ClassifyWith([]string{"ANNA", "~Who took the biscuit?", "~Who? Who?", "Not me.", "", "~La la la"}, nil)
	want := []Kind{Character, Lyrics, Lyrics, Dialogue, Empty, Lyrics}
	for i, k := range want {
		if kinds[i] != k {
			t.Errorf("line %d: %v, want %v", i, kinds[i], k)
		}
	}
}

// Scripts without capitals count as small letters: a cue in Hebrew or
// Arabic is marked with @.
func TestUncasedCues(t *testing.T) {
	kinds := ClassifyWith([]string{"سارة", "مرحبا", "", "@سارة", "مرحبا", "", "SARA (V.O.)", "Hi."}, nil)
	if kinds[0] == Character || kinds[3] != Character || kinds[6] != Character {
		t.Errorf("cues: %v", kinds)
	}
}
