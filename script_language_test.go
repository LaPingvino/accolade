package main

import "testing"

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
		{"EST. CITY - DAY", "en", "nl", translateHeadings, ".EST. CITY - DAY"}, // no word: dotted
		{"EST. GIARDINO", "it", "en", translateHeadings, "EXT. GIARDINO"},
		{"EN. KUIRO - TAGO", "eo", "en", fountainForm, ".EN. KUIRO - TAGO"},
		{".EN. KUIRO - TAGO", "en", "eo", fountainForm, "EN. KUIRO - TAGO"},
		{"INT. KITCHEN", "en", "de", fountainForm, "INT. KITCHEN"}, // German knows it
		{"EN. KUIRO", "eo", "en", markLanguage, "EN. KUIRO"},
		{"...and so on", "en", "eo", fountainForm, "...and so on"},
		{"He goes in.", "en", "nl", translateHeadings, "He goes in."},
	} {
		if got, _, _ := changedLine(c.line, c.from, c.to, c.how); got != c.want {
			t.Errorf("%q %s->%s (%d): got %q, want %q", c.line, c.from, c.to, c.how, got, c.want)
		}
	}
}

func TestHeadingChanges(t *testing.T) {
	text := "Title: X\n\nINT. ROOM - DAY\n\nHe waits.\n\nEXT. ROAD - NIGHT\n\nEST. CITY - DAY\n"
	if changed, kept := headingChanges(text, "en", "nl", translateHeadings); changed != 3 || kept != 1 {
		t.Errorf("got %d changed, %d kept; want 3 (one dotted), 1", changed, kept)
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
