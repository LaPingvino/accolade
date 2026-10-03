package main

import (
	"strings"
	"testing"
)

func TestComputeStats(t *testing.T) {
	script := "Title: The Barn\nAuthor: Jane\n\n" +
		"INT. BARN - DAY\n\nRain falls.\n\n" +
		"EXT. FIELD - NIGHT\n\nJOHN\nIt's coming.\n"
	if got := computeStats(script, 0); got.Scenes != 2 || got.Scene != 0 || got.Words != 11 || got.Pages != 1 {
		t.Errorf("at the title page: %+v", got)
	}
	inField := strings.Index(script, "JOHN")
	if got := computeStats(script, inField).Scene; got != 2 {
		t.Errorf("cursor in the second scene: scene %d", got)
	}
	if got := computeStats(script, strings.Index(script, "Rain")).Scene; got != 1 {
		t.Errorf("cursor in the first scene: scene %d", got)
	}
	long := strings.Repeat("INT. ROOM - DAY\n\n"+strings.Repeat("Words go here. ", 20)+"\n\n", 30)
	if got := computeStats(long, 0).Pages; got < 3 || got > 5 {
		t.Errorf("30 scenes with 5-line action: ~%d pages, want about 4", got)
	}
	if got := computeStats("  \n", 0); got != (scriptStats{}) {
		t.Errorf("empty script: %+v", got)
	}
}

func TestStatsText(t *testing.T) {
	for st, want := range map[scriptStats]string{
		{Scenes: 3, Scene: 2, Words: 412, Pages: 2}: "Scene 2 of 3 · 412 words · ~2 pages",
		{Scenes: 3, Words: 40, Pages: 1}:            "3 scenes · 40 words · ~1 page",
		{Words: 1, Pages: 1}:                        "1 word · ~1 page",
		{}:                                          "0 words",
	} {
		if got := st.String(); got != want {
			t.Errorf("%+v: %q, want %q", st, got, want)
		}
	}
}

func TestStatusBarShowsLiveStats(t *testing.T) {
	w := newTestWindow(t, "INT. BARN - DAY\n\nRain falls.")
	if got := w.statsLabel.Text; !strings.Contains(got, "5 words") || !strings.Contains(got, "Scene 1 of 1") {
		t.Errorf("after loading: %q", got)
	}
	w.textEditor.SetText("INT. BARN - DAY\n\nRain falls hard.\n\nEXT. FIELD - DAY")
	w.textEditor.SetCursorOffset(len("INT. BARN - DAY\n\nRain"))
	if got := w.statsLabel.Text; !strings.Contains(got, "Scene 1 of 2") || !strings.Contains(got, "9 words") {
		t.Errorf("after editing: %q", got)
	}
}
