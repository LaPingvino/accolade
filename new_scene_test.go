package main

import (
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
		{script, 2, "Title: X\nAuthor: Y\n\n"},                           // on the title page: after it
		{script, strings.Index(script, "One"), "Title: X\nAuthor: Y\n\nINT. A - DAY\n\nOne.\n\n"}, // in scene A: before the section
		{script, strings.Index(script, "Two"), script},                    // the last scene, no newline at the end
		{"", 0, ""},
		{"# Act\n\nSomething.\n\n", 1, "# Act\n"},                          // a section before any scene: after its paragraph
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
