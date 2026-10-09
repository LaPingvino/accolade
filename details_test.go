package main

import (
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
