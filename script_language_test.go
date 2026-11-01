package main

import (
	"testing"

	"github.com/LaPingvino/lexington/rules"
)

func TestChangedLine(t *testing.T) {
	for _, c := range []struct {
		line, from, to string
		how            languageChange
		want           string
	}{
		{"INT. KITCHEN - DAY", "en", "nl", translateHeadings, "BIN. KITCHEN - DAY"},
		{"I/E CAR - DAY", "en", "nl", translateHeadings, "BIN./BUI. CAR - DAY"},
		{"EKST. ĜARDENO - NOKTO", "eo", "en", translateHeadings, "EXT. ĜARDENO - NOKTO"},
		{"INT. КУХНЯ", "en", "ru", translateHeadings, "ИНТ. КУХНЯ"},
		{"EST. CITY - DAY", "en", "de", translateHeadings, "ETABL. CITY - DAY"},
		{"EST. CITY - DAY", "en", "nl", translateHeadings, "BUI. CITY - DAY"},  // no word for it: outside
		{"EST. CITY - DAY", "en", "ja", translateHeadings, ".EST. CITY - DAY"}, // no words at all: dotted
		{"EST. GIARDINO", "it", "en", translateHeadings, "EXT. GIARDINO"},
		{"EN. KUIRO - TAGO", "eo", "en", fountainForm, ".EN. KUIRO - TAGO"},
		{".EN. KUIRO - TAGO", "en", "eo", fountainForm, "EN. KUIRO - TAGO"},
		{"INT. KITCHEN", "en", "de", fountainForm, "INT. KITCHEN"}, // German knows it
		{"EN. KUIRO", "eo", "en", markLanguage, "EN. KUIRO"},
		{"...and so on", "en", "eo", fountainForm, "...and so on"},
		{"He goes in.", "en", "nl", translateHeadings, "He goes in."},
	} {
		var words map[string]string
		if c.how == translateHeadings {
			words = map[string]string{}
			for _, p := range proposedWords(c.line, c.from, c.to) {
				words[p.tag] = p.word
			}
		}
		if got, _, _ := changedLine(c.line, c.from, c.to, c.how, words); got != c.want {
			t.Errorf("%q %s->%s (%d): got %q, want %q", c.line, c.from, c.to, c.how, got, c.want)
		}
	}
}

func TestProposedWords(t *testing.T) {
	text := "Title: X\n\nINT. ROOM - DAY\n\nHe waits.\n\nEXT. ROAD - NIGHT\n\nINT. HALL\n\nEST. CITY - DAY\n"
	props := proposedWords(text, "en", "nl")
	want := []wordProposal{{"INT", 2, "BIN."}, {"EXT", 1, "BUI."}, {"EST", 1, "BUI."}}
	if len(props) != len(want) {
		t.Fatalf("got %v, want %v", props, want)
	}
	for i := range want {
		if props[i] != want[i] {
			t.Errorf("%d: got %v, want %v", i, props[i], want[i])
		}
	}
	// the writer's choice: EST. typed as OPEN., INT. cleared
	words := map[string]string{"INT": "", "EXT": "BUI.", "EST": "OPEN."}
	// something uncommon typed: it goes in with a dot, a heading anyway
	if got, _, _ := changedLine("EXT. GARDEN", "en", "nl", translateHeadings, map[string]string{"EXT": "TUIN."}); got != ".TUIN. GARDEN" {
		t.Errorf("uncommon word: got %q", got)
	}
	for line, want := range map[string]string{
		"INT. ROOM - DAY": ".INT. ROOM - DAY", // kept, dotted: no heading in Dutch
		"EST. CITY - DAY": "OPEN. CITY - DAY",
		"EXT. ROAD":       "BUI. ROAD",
	} {
		if got, _, _ := changedLine(line, "en", "nl", translateHeadings, words); got != want {
			t.Errorf("%q: got %q, want %q", line, got, want)
		}
	}
	if changed, kept := headingChanges(text, "en", "nl", translateHeadings, words); changed != 4 || kept != 2 {
		t.Errorf("got %d changed, %d kept; want 4, 2", changed, kept)
	}
}

func TestWithLanguageName(t *testing.T) {
	for _, c := range []struct{ path, lang, want string }{
		{"/s/scene.eo.fountain", "nl", "/s/scene.nl.fountain"},
		{"/s/scene.fountain", "eo", "/s/scene.eo.fountain"},
		{"/s/scene.eo.fountain", "en", "/s/scene.fountain"},
		{"/s/scene.fountain", "en", "/s/scene.fountain"},
		{"/s/my.script.fountain", "nl", "/s/my.script.nl.fountain"},
		{"scene", "eo", "scene.eo"},
	} {
		if got := withLanguageName(c.path, c.lang); got != c.want {
			t.Errorf("%s %s: got %s, want %s", c.path, c.lang, got, c.want)
		}
	}
}

func TestLanguageLabels(t *testing.T) {
	for l := range rules.DefaultConf().Scenes { // Lexington's own: tests add others
		if languageLabel(l) == l {
			t.Errorf("%s has no name", l)
		}
	}
	if got := languageLabel("tok"); got != "toki pona (tok)" {
		t.Errorf("tok: %q", got)
	}
}
