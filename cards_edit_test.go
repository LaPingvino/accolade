// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import "testing"

func TestSetSynopsis(t *testing.T) {
	starts := []string{"INT.", "EXT."}
	text := "INT. BARN - DAY\n= Old one\n= and more\n\nRain.\n\nEXT. FIELD - NIGHT\n\nWind.\n"
	got := setSynopsis(text, starts, 0, "Jo  hides\nthe letter")
	if want := "INT. BARN - DAY\n= Jo hides the letter\n\nRain.\n\nEXT. FIELD - NIGHT\n\nWind.\n"; got != want {
		t.Errorf("replaced:\n%q\nwant\n%q", got, want)
	}
	got = setSynopsis(got, starts, 1, "Dark")
	if want := "INT. BARN - DAY\n= Jo hides the letter\n\nRain.\n\nEXT. FIELD - NIGHT\n= Dark\n\nWind.\n"; got != want {
		t.Errorf("added:\n%q\nwant\n%q", got, want)
	}
	if got = setSynopsis(got, starts, 0, ""); outlineOf(got, starts)[0].Synopsis != "" {
		t.Errorf("not removed: %q", got)
	}
	if items := outlineOf(got, starts); items[1].Synopsis != "Dark" {
		t.Errorf("the other card's synopsis: %q", items[1].Synopsis)
	}
}

func TestAddScene(t *testing.T) {
	starts := []string{"INT.", "EXT."}
	text := "# Act One\n\nINT. BARN - DAY\n\nRain.\n\n# Act Two\n\nEXT. FIELD - NIGHT\n"
	// at the end of Act One: before the item "Act Two"
	got, line := addScene(text, starts, 2, "EXT. ROAD - DAY", "Jo runs")
	items := outlineOf(got, starts)
	if len(items) != 5 || items[2].Text != "EXT. ROAD - DAY" || items[2].Synopsis != "Jo runs" || items[3].Text != "Act Two" {
		t.Fatalf("in the middle: %q", got)
	}
	if items[2].Line != line {
		t.Errorf("the heading is on line %d, said %d", items[2].Line, line)
	}
	// at the end, with a heading in no language: forced
	got, _ = addScene(text, starts, 99, "The chase", "")
	items = outlineOf(got, starts)
	if last := items[len(items)-1]; last.Text != "THE CHASE" || !last.Scene {
		t.Errorf("at the end: %q", got)
	}
	// into an empty script
	if got, _ = addScene("", starts, 0, "INT. HALL - DAY", ""); got != "INT. HALL - DAY\n" {
		t.Errorf("empty: %q", got)
	}
}

// A heading's synopsis is the first synopsis under it (blank lines may
// stand between); the synopses after it, after other lines or a blank
// line, are loose ones of their own (a beat, a planned moment): each
// its own, in the item's Loose, never joined into the heading's.
func TestLooseSynopses(t *testing.T) {
	starts := []string{"INT.", "EXT."}
	text := "# Act One\n\n= What the act does\n= in two lines\n\n= Mara finds the letter.\n\n= She stays.\n\nINT. BARN - DAY\n\n= The barn.\n\nRain.\n\n= Later: the roof goes.\n"
	items := outlineOf(text, starts)
	if len(items) != 2 {
		t.Fatalf("items: %+v", items)
	}
	if items[0].Synopsis != "What the act does in two lines" || items[1].Synopsis != "The barn." {
		t.Errorf("own synopses: %q, %q", items[0].Synopsis, items[1].Synopsis)
	}
	want := []looseSynopsis{{Line: 5, Text: "Mara finds the letter."}, {Line: 7, Text: "She stays."}}
	if len(items[0].Loose) != 2 || items[0].Loose[0] != want[0] || items[0].Loose[1] != want[1] {
		t.Errorf("the act's loose ones: %+v", items[0].Loose)
	}
	if len(items[1].Loose) != 1 || items[1].Loose[0] != (looseSynopsis{Line: 15, Text: "Later: the roof goes."}) {
		t.Errorf("the scene's loose ones: %+v", items[1].Loose)
	}
}

// Writing a card's or a section's synopsis replaces its own, never the
// loose synopses after it (the beats under an act were joined into one).
func TestSetSynopsisKeepsLoose(t *testing.T) {
	starts := []string{"INT.", "EXT."}
	text := "# Act One\n\n= Mara finds the letter.\n\n= She stays.\n\n= The storm.\n"
	got := setSynopsis(text, starts, 0, "Mara finds the letter!")
	if want := "# Act One\n\n= Mara finds the letter!\n\n= She stays.\n\n= The storm.\n"; got != want {
		t.Errorf("got\n%q\nwant\n%q", got, want)
	}
}
