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
