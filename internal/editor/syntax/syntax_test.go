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
