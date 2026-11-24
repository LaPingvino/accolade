package main

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/rules"
)

// newSceneLangs are the languages New Scene is checked in.
var newSceneLangs = []string{"en", "eo", "zh", "hi-Deva", "ar", "he", "ja"}

// In every language the form's heading, written into the script, reads
// back as a heading in it (Accolade's outline and Lexington's parser),
// with the place typed.
func TestNewSceneReadsBack(t *testing.T) {
	for _, lang := range newSceneLangs {
		w := newTestWindow(t, "")
		w.setLanguage(lang)
		first := rules.FillPattern(w.lex().Language(lang).HeadingForm(), sampleValues(lang, w.lex().Language(lang), w.lex().Language(lang).HeadingForm()))
		w.textEditor.SetText(first + "\n\nRain.\n")
		w.textEditor.SetCursorOffset(3)
		d, at := w.newSceneDraft()
		place := samplePlaceFor(lang) + "2"
		if slices := rules.PatternSlots(d.form); strings.Contains(strings.Join(slices, " "), "place") {
			d.values[rules.SlotPlace] = place
		} else {
			d.values[rules.SlotAny] = place
		}
		if !d.reads() {
			t.Errorf("%s: %q (by %q) does not read", lang, d.heading(), d.form)
			continue
		}
		w.insertScene(at, d.heading())
		text := w.textEditor.Text()
		var scenes []string
		for _, it := range outlineOf(text, w.sceneStarts()) {
			if it.Scene {
				scenes = append(scenes, it.Text)
			}
		}
		if len(scenes) != 2 || !strings.Contains(scenes[1], strings.ToUpper(place)) {
			t.Errorf("%s: scenes %q in\n%s", lang, scenes, text)
		}
		places, _ := scriptPlaces(text, w.lex().Language(lang), w.sceneStarts())
		if lang != "he" && (len(places) != 2 || places[1] != strings.ToUpper(place)) {
			t.Errorf("%s: places %q", lang, places)
		}
		// the cursor on the new scene's first action line, between blank lines
		lines := strings.Split(text, "\n")
		line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
		if lines[line] != "" || !strings.Contains(lines[line-2], strings.ToUpper(place)) && !strings.Contains(lines[line-2], place) {
			t.Errorf("%s: cursor on line %d of\n%s", lang, line, text)
		}
		w.undo()
		if w.textEditor.Text() != first+"\n\nRain.\n" {
			t.Errorf("%s: undo gives %q", lang, w.textEditor.Text())
		}
	}
}

// Where a new scene goes and the blank lines around it.
func TestNewSceneAt(t *testing.T) {
	starts := sceneStarts("en")
	script := "Title: X\nAuthor: Y\n\nINT. A - DAY\n\nOne.\n\n# Act\n\nINT. B - DAY\n\nTwo."
	r := func(s string) int { return len([]rune(s)) }
	for _, c := range []struct {
		text   string
		cursor int
		want   string // what is before the new scene
	}{
		{script, 2, "Title: X\nAuthor: Y\n\n"},                                                    // on the title page: after it
		{script, strings.Index(script, "One"), "Title: X\nAuthor: Y\n\nINT. A - DAY\n\nOne.\n\n"}, // in scene A: before the section
		{script, strings.Index(script, "Two"), script},                                            // the last scene, no newline at the end
		{"", 0, ""},
		{"# Act\n\nSomething.\n\n", 1, "# Act\n"}, // a section before any scene: after its paragraph
	} {
		at := newSceneAt(c.text, r(c.text[:c.cursor]), starts)
		if got := string([]rune(c.text)[:at]); got != c.want {
			t.Errorf("cursor %d: before %q, want %q", c.cursor, got, c.want)
		}
		ins, cur := newSceneText(c.text, at, "INT. NEW - NIGHT")
		out := string([]rune(c.text)[:at]) + ins + string([]rune(c.text)[at:])
		if !strings.Contains(out, "\n\nINT. NEW - NIGHT\n\n") && !strings.HasPrefix(out, "INT. NEW - NIGHT\n\n") {
			t.Errorf("%q: no blank lines around: %q", c.text, out)
		}
		if strings.Contains(out, "\n\n\n\n\n") || !strings.HasPrefix(string([]rune(ins)[cur:]), "") {
			t.Errorf("too many blank lines: %q", out)
		}
		if strings.Contains(c.text, "INT. B") && !strings.Contains(out, "\n\nINT. B") {
			t.Errorf("next heading's blank line lost: %q", out)
		}
	}
	// the blocks at the end stay after it
	withBlock := "INT. A - DAY\n\nOne.\n\n/* Production\nscene numbers: locked, 10A\n*/\n"
	at := newSceneAt(withBlock, r(withBlock)-2, starts)
	if at > strings.Index(withBlock, "/*") {
		t.Errorf("after the blocks: %d", at)
	}
}

// A locked script: the next free number, written with the heading; in a
// zh pattern the {n} gets it too, and the lock's A11 reads.
func TestNewSceneLocked(t *testing.T) {
	block := "\n\n/* Production\nscene numbers: locked, %s\n*/\n"
	for _, c := range []struct{ lang, style, want string }{{"en", "10A", "1A"}, {"zh", "A11", "A2"}} {
		w := newTestWindow(t, "")
		w.setLanguage(c.lang)
		l := w.lex().Language(c.lang)
		h1 := strings.Replace(rules.FillPattern(l.HeadingForm(), map[string]string{"n": "1", "io": l.IOWords(l.HeadingForm())[0], "place": "A", "time": "DAY"}), "DAY", "日", 1)
		h2 := strings.Replace(h1, "1", "2", 1)
		text := h1 + " #1#\n\nOne.\n\n" + h2 + " #2#\n\nTwo." + strings.Replace(block, "%s", c.style, 1)
		w.textEditor.SetText(text)
		w.textEditor.SetCursorOffset(len([]rune(h1)) + 4)
		d, at := w.newSceneDraft()
		d.values[rules.SlotPlace] = "B"
		if d.suggested != c.want || !strings.HasSuffix(d.heading(), "#"+c.want+"#") || !d.reads() {
			t.Errorf("%s: suggested %q, heading %q, reads %v", c.lang, d.suggested, d.heading(), d.reads())
		}
		w.insertScene(at, d.heading())
		if !strings.Contains(w.textEditor.Text(), "\n\n"+d.heading()+"\n\n") {
			t.Errorf("%s:\n%s", c.lang, w.textEditor.Text())
		}
	}
}

// The form itself: fields in the form's order, typing a place updates
// the preview, Insert inserts.
func TestNewSceneForm(t *testing.T) {
	w := newTestWindow(t, "INT. KITCHEN - DAY\n\nRain.\n")
	w.fyneWindow.Resize(fyne.NewSize(900, 700))
	top, _ := w.newSceneContent(func() {})
	var labels []string
	collect(top, func(l *widget.Label) {
		if l.TextStyle.Bold && !l.TextStyle.Monospace {
			labels = append(labels, l.Text)
		}
	})
	if strings.Join(labels, ",") != "Inside/Outside,Place,Time,Number,Heading" {
		t.Errorf("fields %v", labels)
	}
	place := find(top, func(e *widget.Entry) bool { return e.PlaceHolder == "KITCHEN" })
	if place == nil {
		t.Fatal("no place")
	}
	// the script's place to pick
	if find(top, func(b *widget.Button) bool { return b.Text == "KITCHEN" }) == nil {
		t.Error("the script's place is not offered")
	}
	test.Type(place, "HALL")
	preview := find(top, func(l *widget.Label) bool { return l.TextStyle.Monospace })
	if preview.Text != "INT. HALL - DAY" {
		t.Errorf("preview %q", preview.Text)
	}
	test.Tap(find(top, func(b *widget.Button) bool { return b.Text == "Insert" }))
	if got := w.textEditor.Text(); got != "INT. KITCHEN - DAY\n\nRain.\n\nINT. HALL - DAY\n\n" {
		t.Errorf("text %q", got)
	}
}

// A field left empty leaves its slot out with its own text, and the
// heading still reads (Joop: "keeps working when used partially"): no
// time, no place, no number, in every language; a sentence is no heading.
func TestNewScenePartial(t *testing.T) {
	for _, lang := range newSceneLangs {
		w := newTestWindow(t, "")
		w.setLanguage(lang)
		w.textEditor.SetText("Rain.\n")
		d, _ := w.newSceneDraft()
		full := sampleValues(lang, d.l, d.form)
		for _, drop := range []string{"", rules.SlotTime, rules.SlotPlace, rules.SlotNumber} {
			d.values = maps.Clone(full)
			if drop != "" {
				if !slices.Contains(rules.PatternSlots(d.form), drop) {
					continue
				}
				delete(d.values, drop)
			}
			h := d.heading()
			if !d.reads() {
				t.Errorf("%s without %q: %q does not read", lang, drop, h)
			}
			if strings.HasSuffix(h, "-") || strings.Contains(h, "（）") || strings.Contains(h, "//") || strings.HasSuffix(h, "/") {
				t.Errorf("%s without %q: %q has text of a slot left out", lang, drop, h)
			}
			if !isSceneHeadingIn(h, w.sceneStarts()) {
				t.Errorf("%s: the outline does not read %q", lang, h)
			}
		}
		for _, action := range []string{"Inside the house it was dark.", "1. Buy milk"} {
			if isSceneHeadingIn(action, w.sceneStarts()) {
				t.Errorf("%s: %q read as a heading", lang, action)
			}
		}
	}
}

// Completion when the writer stops early or skips a slot: a heading
// left after its place reads, and after "场景2 厨房 " both the time and
// the inside/outside are offered.
func TestCompletePartial(t *testing.T) {
	got := completionsIn("zh", "场景1 客厅 日 内\n\n下雨。\n\n", "场景2 厨房 ")
	for _, want := range []string{"日", "内"} {
		if !slices.Contains(got, want) {
			t.Errorf("zh skipping the time: %v lacks %q", got, want)
		}
	}
	w := flowWindow(t, "INT. KITCHEN - DAY\n\nRain.\n\n")
	typeKeys(t, w, "EXT. K\t\nSnow.")
	if e := elements(w.textEditor.Text()); e[len(e)-2] != "scene:EXT. KITCHEN" || e[len(e)-1] != "action:Snow." {
		t.Errorf("stopped after the place: %q", e)
	}
}

// New Scene on a loose synopsis (one standing on its own, not a
// heading's): the heading goes right above it, the synopsis becomes the
// new scene's, the cursor after it. In a section's beats and in a scene
// alike; a heading's own synopsis still gets its scene after it.
func TestNewSceneOnLooseSynopsis(t *testing.T) {
	starts := sceneStarts("en")
	r := func(s string) int { return len([]rune(s)) }
	for _, c := range []struct {
		text, at, want string
	}{
		{"# Act\n\n= Own.\n\n= Beat one.\n\n= Beat two.\n", "Beat one",
			"# Act\n\n= Own.\n\nINT. NEW - NIGHT\n\n= Beat one.\n\n= Beat two.\n"},
		{"# Act\n\n= Own.\n\nINT. A - DAY\n\nAction.\n\n= Then this.\n\nMore.\n", "Then this",
			"# Act\n\n= Own.\n\nINT. A - DAY\n\nAction.\n\nINT. NEW - NIGHT\n\n= Then this.\n\nMore.\n"},
	} {
		cursor := r(c.text[:strings.Index(c.text, c.at)])
		at := newSceneAt(c.text, cursor, starts)
		if !onLooseSynopsis(c.text, at, cursor) {
			t.Errorf("%q: not on a synopsis", c.at)
		}
		ins, cur := onSynopsisText(c.text, at, "INT. NEW - NIGHT")
		out := string([]rune(c.text)[:at]) + ins + string([]rune(c.text)[at:])
		if out != c.want {
			t.Errorf("got\n%q\nwant\n%q", out, c.want)
			continue
		}
		if got := string([]rune(out)[:at+cur]); !strings.HasSuffix(got, c.at+".") {
			t.Errorf("the cursor is after %q, not after the synopsis", got)
		}
	}
	// in a scene with a beat still to write after it: before that beat
	text0 := "# Act\n\nINT. A - DAY\n\n= Own.\n\nAction.\n\n= Next beat.\n"
	at0 := newSceneAt(text0, r(text0[:strings.Index(text0, "Action")]), starts)
	if got := string([]rune(text0)[:at0]); got != "# Act\n\nINT. A - DAY\n\n= Own.\n\nAction.\n\n" {
		t.Errorf("before the next beat: before %q", got)
	}
	// a synopsis with more of the scene after it is the scene's: after the scene
	text1 := "INT. A - DAY\n\nAction.\n\n= A moment in it.\n\nMore.\n\nINT. B - DAY\n\nB.\n"
	at1 := newSceneAt(text1, r(text1[:strings.Index(text1, "Action")]), starts)
	if got := string([]rune(text1)[:at1]); got != "INT. A - DAY\n\nAction.\n\n= A moment in it.\n\nMore.\n\n" {
		t.Errorf("past a mid-scene synopsis: before %q", got)
	}
	// a heading's own synopsis: after its paragraph, as before
	text := "# Act\n\n= Own.\n\n= Beat one.\n"
	at := newSceneAt(text, r(text[:strings.Index(text, "Own")]), starts)
	if got := string([]rune(text)[:at]); got != "# Act\n\n= Own.\n" {
		t.Errorf("on the act's own synopsis: before %q", got)
	}
}
