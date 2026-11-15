package main

import (
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/lex"
)

func TestWithColor(t *testing.T) {
	for _, tc := range []struct{ line, color, want string }{
		{"INT. BARN - DAY", "red", "INT. BARN - DAY [[red]]"},
		{"INT. BARN - DAY  ", "Blue", "INT. BARN - DAY [[blue]]"},
		{"INT. BARN - DAY #12#", "red", "INT. BARN - DAY #12# [[red]]"},
		{"INT. BARN - DAY #12# [[red]]", "green", "INT. BARN - DAY #12# [[green]]"},
		{"INT. BARN - DAY [[COLOR BLUE]] #12#", "teal", "INT. BARN - DAY #12# [[teal]]"},
		{"INT. BARN - DAY [[#ff8800]] [[storyline A]] #3A#", "pink", "INT. BARN - DAY [[storyline A]] #3A# [[pink]]"},
		{"INT. BARN - DAY [[a note]] [[red]]", "none", "INT. BARN - DAY [[a note]]"},
		{"INT. BARN - DAY #12# [[red]]", "", "INT. BARN - DAY #12#"},
		{"INT. BARN - DAY [[red]] [[blue]]", "gray", "INT. BARN - DAY [[gray]]"},
		{"INT. BARN - DAY [[color]]", "red", "INT. BARN - DAY [[color]] [[red]]"}, // "color" alone is no color
		{"[[red]] .FLASHBACK", "none", ".FLASHBACK"},
		{"# ACT ONE [[red]]", "brown", "# ACT ONE [[brown]]"},
		{"## The garden [[note]]", "yellow", "## The garden [[note]] [[yellow]]"},
		{"INT. BARN - DAY [[open", "red", "INT. BARN - DAY [[red]] [[open"},
		{"INT. BARN - DAY [[blue]] [[open", "none", "INT. BARN - DAY [[open"},
		{"INT. BARN - DAY", "none", "INT. BARN - DAY"},
	} {
		got := withColor(tc.line, tc.color)
		if got != tc.want {
			t.Errorf("withColor(%q, %q) = %q, want %q", tc.line, tc.color, got, tc.want)
		}
		if want := tc.color; want != "" && want != "none" && !lexColorIs(got, want) {
			t.Errorf("%q: lex.NoteColor is %q", got, lex.NoteColor(got))
		}
	}
}

func lexColorIs(line, c string) bool { return lex.NoteColor(line) == strings.ToLower(c) }

func TestSetItemColor(t *testing.T) {
	text := "# ACT ONE\n\nINT. BARN - DAY #1# [[red]]\n= Anna finds the tin.\n\nRain.\n"
	got := setItemColor(text, nil, 1, "blue")
	if want := "# ACT ONE\n\nINT. BARN - DAY #1# [[blue]]\n= Anna finds the tin.\n\nRain.\n"; got != want {
		t.Errorf("scene: %q", got)
	}
	got = setItemColor(got, nil, 0, "orange")
	if want := "# ACT ONE [[orange]]\n\nINT. BARN - DAY #1# [[blue]]\n= Anna finds the tin.\n\nRain.\n"; got != want {
		t.Errorf("section: %q", got)
	}
	if got := setItemColor(text, nil, 5, "red"); got != text {
		t.Errorf("no such item: %q", got)
	}
}

func TestNoteRGB(t *testing.T) {
	for name, ok := range map[string]bool{
		"red": true, "grey": true, "brown": true, "#ff8800": true, "#FF8800": true,
		"#zzzzzz": false, "#ff88": false, "chartreuse": false, "": false, "#+12345": false,
	} {
		if _, got := noteRGB(name); got != ok {
			t.Errorf("noteRGB(%q): %v", name, got)
		}
	}
	if c, _ := noteRGB("#ff8800"); c.R != 0xff || c.G != 0x88 || c.B != 0 {
		t.Errorf("#ff8800 is %v", c)
	}
	for _, n := range beatColorNames {
		if _, ok := noteRGB(n); !ok {
			t.Errorf("no RGB for the picker's %q", n)
		}
	}
}
