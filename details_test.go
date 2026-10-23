package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestDetails(t *testing.T) {
	script := "Title: T\n\nINT. KITCHEN - DAY\n\nAnna looks.\n\nANNA\nHello there, Bram.\n\nBRAM (V.O.)\nHi.\n\nEXT. GARDEN - NIGHT\n\nANNA\nGood night.\n\nI/E. CAR - DAY\n\nThey drive.\n"
	d := detailsOf(script, sceneHeaders("en"), nil)
	if d.Scenes != 3 || d.Interior != 2 || d.Exterior != 2 {
		t.Errorf("scenes %d, interior %d, exterior %d", d.Scenes, d.Interior, d.Exterior)
	}
	if len(d.Roles) != 2 || d.Roles[0].Name != "ANNA" || d.Roles[0].Speeches != 2 || d.Roles[0].Words != 5 ||
		d.Roles[1].Name != "BRAM" {
		t.Errorf("roles %+v", d.Roles)
	}
	if d.Pages != 1 || d.DialogueLines == 0 || d.ActionLines == 0 || d.Words == 0 {
		t.Errorf("%+v", d)
	}
	// a long script: more pages, never fewer than none
	long := script + strings.Repeat("\nA line of action that goes on.\n", 200)
	if d := detailsOf(long, sceneHeaders("en"), nil); d.Pages < 7 || d.Pages > 9 {
		t.Errorf("long script: %d pages", d.Pages)
	}
	if minutes(110) != "2:00" {
		t.Errorf("minutes %s", minutes(110))
	}
}

// Inside and outside in the script's language.
func TestInsideOutside(t *testing.T) {
	for _, c := range []struct {
		lang, heading string
		in, out       bool
	}{
		{"en", "INT. BARN - DAY", true, false},
		{"en", "EXT. FIELD - NIGHT", false, true},
		{"en", "INT./EXT. CAR - DAY", true, true},
		{"en", "I/E CAR - DAY", true, true},
		{"en", "EST. CITY - DAY", false, false},
		{"eo", "EN. DOMO - TAGO", true, false},
		{"eo", "ENE DOMO - TAGO", true, false},
		{"eo", "EKST. KAMPO - NOKTO", false, true},
		{"eo", "EK KAMPO - NOKTO", false, true},
		{"nl", "BUI. VELD - NACHT", false, true},
		{"nl", "BIN. HUIS - DAG", true, false},
		{"ru", "НАТ. ПОЛЕ - НОЧЬ", false, true},
		{"it", "EST. CAMPO - NOTTE", false, true},
	} {
		if in, out := insideOutside(c.heading, sceneHeaders(c.lang)); in != c.in || out != c.out {
			t.Errorf("%s %q: inside %v outside %v", c.lang, c.heading, in, out)
		}
	}
}

// Scenes tallied by their heading's start as written, and inside or
// outside, in the script's language.
func TestDetailsTags(t *testing.T) {
	d := detailsOf("EN. DOMO - TAGO\n\nPluvas.\n\nEKST. KAMPO - NOKTO\n\nVento.\n\nEN. KUIREJO - TAGO\n\nFajro.\n\nEN/EKST. AŬTO - TAGO\n\nVojo.\n", sceneHeaders("eo"), currentScriptFormat())
	if d.Interior != 3 || d.Exterior != 2 {
		t.Errorf("inside %d, outside %d", d.Interior, d.Exterior)
	}
	if fmt.Sprint(d.Tags) != "[{EN 2} {EKST 1} {EN/EKST 1}]" {
		t.Errorf("tags %v", d.Tags)
	}
}
