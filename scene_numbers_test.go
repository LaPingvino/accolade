package main

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/lexington/rules"
)

// numberedScripts are a script per pattern language with its numbers in
// its headings' words, and English with #n#, its numbers in the headings
// turned on: three scenes; place is the field New Scene's place goes in.
var numberedScripts = map[string]string{
	"zh":      "场景1 厨房 日 内\n\n他做饭。\n\n场景2 街道 夜 外\n\n雨。\n\n场景3 公园 日 外\n\n鸟。\n",
	"he":      "1. פנים. משרד – יום.\n\nהוא כותב.\n\n2. חוץ. רחוב – לילה.\n\nגשם.\n\n3. חוץ. פארק – יום.\n\nציפורים.\n",
	"hi-Deva": "1. रसोई/दिन/अंदर\n\nवह खाना बनाता है।\n\n2. सड़क/रात/बाहर\n\nबारिश।\n\n3. पार्क/दिन/बाहर\n\nपक्षी।\n",
	"en":      "INT. KITCHEN - DAY #1#\n\nHe cooks.\n\nEXT. STREET - NIGHT #2#\n\nRain.\n\nEXT. PARK - DAY #3#\n\nBirds.\n\n/* Production\nnumbers in headings: on\n*/\n",
}

// numbersOfHeads are the scenes' numbers, in order.
func numbersOfHeads(w *MainWindow) []string {
	var out []string
	for _, h := range sceneHeads(w.textEditor.Text()[:bodyEndByte(w.textEditor.Text())], w.sceneStarts()) {
		out = append(out, h.num)
	}
	return out
}

// numberedWindow is a window with lang's numbered script, edited once
// (a script just opened is not numbered by itself).
func numberedWindow(t *testing.T, lang string) *MainWindow {
	t.Helper()
	w := newTestWindow(t, "")
	w.setLanguage(lang)
	w.textEditor.SetText(numberedScripts[lang])
	w.textEditor.SetCursorOffset(0)
	if !w.numbersInHeadings(w.textEditor.Text()) {
		t.Fatalf("%s: the numbers are not in the headings", lang)
	}
	return w
}

// undoTo undoes once and wants text back.
func undoTo(t *testing.T, w *MainWindow, what, text string) {
	t.Helper()
	w.undo()
	if got := w.textEditor.Text(); got != text {
		t.Errorf("%s: one Undo leaves\n%s\nnot\n%s", what, got, text)
	}
}

// In each pattern language, and English with #n#, a scene put in, moved
// or deleted renumbers the scenes 1…n in their new order, in the same
// undo step, the status saying so.
func TestNumbersKeptInStep(t *testing.T) {
	for _, lang := range []string{"zh", "he", "hi-Deva", "en"} {
		w := numberedWindow(t, lang)
		start := w.textEditor.Text()
		heads := headsOf(w)

		// New Scene after the first: it is 2, the ones after it 3 and 4
		w.textEditor.SetCursorOffset(2)
		d, at := w.newSceneDraft()
		if d.suggested != "2" {
			t.Errorf("%s: suggested %q", lang, d.suggested)
		}
		if slices.Contains(rules.PatternSlots(d.form), rules.SlotPlace) {
			d.values[rules.SlotPlace] = samplePlaceFor(lang)
		} else {
			d.values[rules.SlotAny] = samplePlaceFor(lang)
		}
		w.insertScene(at, d.heading())
		if got := numbersOfHeads(w); !slices.Equal(got, []string{"1", "2", "3", "4"}) {
			t.Errorf("%s: after New Scene %q:\n%s", lang, got, w.textEditor.Text())
		}
		if hs := headsOf(w); !strings.Contains(hs[2], strings.TrimSuffix(strings.Fields(heads[1])[1], ".")) {
			t.Errorf("%s: the second scene is not 3: %q", lang, hs)
		}
		if !strings.Contains(w.statusLabel.Text, "Scenes renumbered (2 changed)") {
			t.Errorf("%s: status %q", lang, w.statusLabel.Text)
		}
		undoTo(t, w, lang+" New Scene", start)

		// the last scene moved to the top
		w.moveOutlineItem(2, 0)
		hs := headsOf(w)
		if got := numbersOfHeads(w); !slices.Equal(got, []string{"1", "2", "3"}) {
			t.Errorf("%s: after the move %q", lang, hs)
		}
		if bare := withoutWordsNumber(hs[0], w.sceneStarts()); !strings.Contains(heads[2], strings.Fields(strings.TrimSpace(bare))[0]) {
			t.Errorf("%s: moved: %q", lang, hs)
		}
		if !strings.Contains(w.statusLabel.Text, "Scenes renumbered (3 changed)") {
			t.Errorf("%s: status %q", lang, w.statusLabel.Text)
		}
		undoTo(t, w, lang+" move", start)
		w.redo()
		if got := numbersOfHeads(w); !slices.Equal(got, []string{"1", "2", "3"}) || headsOf(w)[0] != hs[0] {
			t.Errorf("%s: redone: %q", lang, headsOf(w))
		}
		undoTo(t, w, lang+" move again", start)

		// the second scene deleted, as the writer does
		text := w.textEditor.Text()
		from := len([]rune(text[:strings.Index(text, heads[1])]))
		to := len([]rune(text[:strings.Index(text, heads[2])]))
		w.textEditor.Edit(func(b *buffer.Buffer) {
			b.Group(func() { b.Replace(from, to, ""); b.SetCursor(0, false) })
		})
		w.numberNewScenes() // typing paused
		if got := numbersOfHeads(w); !slices.Equal(got, []string{"1", "2"}) {
			t.Errorf("%s: after the delete %q", lang, headsOf(w))
		}
		undoTo(t, w, lang+" delete", start)
	}
}

// A heading typed is numbered once the cursor leaves it, not while it
// is typed.
func TestNumberTypedHeading(t *testing.T) {
	w := numberedWindow(t, "zh")
	text := w.textEditor.Text()
	at := len([]rune(text[:strings.Index(text, "场景2")]))
	w.textEditor.Edit(func(b *buffer.Buffer) { b.Replace(at, at, "车站 夜 外\n\n等。\n\n"); b.SetCursor(at+3, false) })
	w.numberNewScenes()
	if got := headsOf(w)[1]; got != "车站 夜 外" {
		t.Errorf("numbered while typed: %q", got)
	}
	w.textEditor.SetCursorOffset(0)
	w.numberNewScenes()
	if got := strings.Join(headsOf(w), "|"); got != "场景1 厨房 日 内|场景2 车站 夜 外|场景3 街道 夜 外|场景4 公园 日 外" {
		t.Errorf("headings %q", got)
	}
}

// Locked, the numbers stay with their scenes: a new one gets its letter
// in its words (12A), a move changes no number.
func TestLockedNumbersInWords(t *testing.T) {
	for _, lang := range []string{"zh", "he", "hi-Deva", "en"} {
		w := numberedWindow(t, lang)
		w.lockSceneNumbers()
		locked := w.textEditor.Text()
		w.textEditor.SetCursorOffset(2)
		d, at := w.newSceneDraft()
		if d.suggested != "1A" {
			t.Errorf("%s: suggested %q", lang, d.suggested)
		}
		d.values[rules.SlotPlace], d.values[rules.SlotAny] = samplePlaceFor(lang), samplePlaceFor(lang)
		w.insertScene(at, d.heading())
		if got := numbersOfHeads(w); !slices.Equal(got, []string{"1", "1A", "2", "3"}) {
			t.Errorf("%s: locked New Scene %q", lang, headsOf(w))
		}
		if h := headsOf(w)[1]; lang != "en" && (!strings.Contains(h, "1A") || strings.Contains(h, "#")) {
			t.Errorf("%s: the letter not in the words: %q", lang, h)
		}
		undoTo(t, w, lang+" locked New Scene", locked)
		w.moveOutlineItem(2, 0)
		if got := numbersOfHeads(w); !slices.Equal(got, []string{"3", "1", "2"}) {
			t.Errorf("%s: locked move %q", lang, headsOf(w))
		}
		if strings.Contains(w.statusLabel.Text, "renumbered") {
			t.Errorf("%s: status %q", lang, w.statusLabel.Text)
		}
	}
}

// Format › Number Scenes in Headings: off takes the numbers out of the
// words, on puts them back; each one undo step; locked, the numbers
// move to #n# and back.
func TestNumbersInHeadingsToggle(t *testing.T) {
	for _, lang := range []string{"zh", "he", "hi-Deva", "en"} {
		w := numberedWindow(t, lang)
		start := w.textEditor.Text()
		w.toggleNumbersInHeadings()
		off := w.textEditor.Text()
		if w.numbersInHeadings(off) {
			t.Errorf("%s: still on", lang)
		}
		if got := numbersOfHeads(w); !slices.Equal(got, []string{"", "", ""}) {
			t.Errorf("%s: off leaves %q", lang, headsOf(w))
		}
		for _, h := range headsOf(w) {
			if !readsAsHeading(w.lex().Language(lang), h) {
				t.Errorf("%s: %q is no heading", lang, h)
			}
		}
		// off: nothing numbered by itself, Renumber on demand
		w.moveOutlineItem(2, 0)
		if got := numbersOfHeads(w); !slices.Equal(got, []string{"", "", ""}) {
			t.Errorf("%s: numbered while off: %q", lang, headsOf(w))
		}
		undoTo(t, w, lang+" move while off", off)
		w.toggleNumbersInHeadings()
		if got := numbersOfHeads(w); !slices.Equal(got, []string{"1", "2", "3"}) {
			t.Errorf("%s: on again %q", lang, headsOf(w))
		}
		if lang != "en" && strings.Contains(strings.Join(headsOf(w), ""), "#") {
			t.Errorf("%s: on writes #n#: %q", lang, headsOf(w))
		}
		undoTo(t, w, lang+" on", off)
		undoTo(t, w, lang+" off", start)

		// locked: off moves the numbers to #n#, on back into the words
		w.lockSceneNumbers()
		w.toggleNumbersInHeadings()
		if got := numbersOfHeads(w); !slices.Equal(got, []string{"1", "2", "3"}) {
			t.Errorf("%s: locked off %q", lang, headsOf(w))
		}
		if h := headsOf(w)[1]; !strings.HasSuffix(h, "#2#") {
			t.Errorf("%s: locked off: %q", lang, h)
		}
		w.toggleNumbersInHeadings()
		if h := headsOf(w)[1]; lang != "en" && strings.Contains(h, "#") {
			t.Errorf("%s: locked on: %q", lang, h)
		}
		if got := numbersOfHeads(w); !slices.Equal(got, []string{"1", "2", "3"}) {
			t.Errorf("%s: locked on %q", lang, headsOf(w))
		}
	}
}

// The choice's default: on for a language writing its headings by a
// pattern with a number, but not for a script written without numbers;
// off for English and Hindi written as English.
func TestNumbersInHeadingsDefault(t *testing.T) {
	for _, tc := range []struct {
		lang, text string
		on         bool
	}{
		{"zh", "", true},
		{"zh", "场景1 厨房 日 内\n\n他做饭。\n", true},
		{"zh", "厨房 日 内\n\n他做饭。\n", false},
		{"hi-Deva", "", true},
		{"he", "פנים. משרד – יום.\n\nהוא כותב.\n", false},
		{"en", "", false},
		{"en", "INT. KITCHEN - DAY #1#\n\nHe cooks.\n", false},
		{"hi", "", false},
		{"zh", "场景1 厨房 日 内\n\n他做饭。\n\n/* Production\nnumbers in headings: off\n*/\n", false},
	} {
		w := newTestWindow(t, "")
		w.setLanguage(tc.lang)
		if got := w.numbersInHeadings(tc.text); got != tc.on {
			t.Errorf("%s %q: %v", tc.lang, tc.text, got)
		}
	}
}

// Off, Renumber Scenes numbers on demand in the words, and a heading
// with a number in its words and another #n# is written with both equal.
func TestRenumberWithNumbersInWords(t *testing.T) {
	w := newTestWindow(t, "")
	w.setLanguage("zh")
	w.textEditor.SetText("场景4 厨房 日 内\n\n他做饭。\n\n场景9 街道 夜 外 #7#\n\n雨。\n\n/* Production\nnumbers in headings: off\n*/\n")
	if got := numbersOfHeads(w); !slices.Equal(got, []string{"4", "7"}) {
		t.Fatalf("read %q", got)
	}
	w.renumberNow()
	if got := strings.Join(headsOf(w), "|"); got != "场景1 厨房 日 内|场景2 街道 夜 外 #2#" {
		t.Errorf("renumbered %q", got)
	}
}

// A keystroke in a long Chinese script with its numbers in its headings
// kept in step, and with them off: the per-key cost of the choice (the
// renumbering itself waits for the pause). Run with
//
//	go test -run '^$' -bench 'BenchmarkNumbersKey' -benchmem -count 6
func BenchmarkNumbersKey(b *testing.B) {
	var sb strings.Builder
	for i := 1; i <= 120; i++ {
		sb.WriteString("场景" + strconv.Itoa(i) + " 厨房 日 内\n\n他做饭。雨一直下。\n\n")
	}
	for _, c := range []struct{ name, block string }{{"on", ""}, {"off", "/* Production\nnumbers in headings: off\n*/\n"}} {
		b.Run(c.name, func(b *testing.B) {
			w, _ := newBenchWindow(b)
			w.setLanguage("zh")
			w.textEditor.SetText(sb.String() + c.block)
			realTimers(b)
			e := w.textEditor
			w.fyneWindow.Canvas().Focus(e)
			e.SetCursorOffset(strings.Index(e.Text(), "雨一直下") / 3)
			for b.Loop() {
				e.TypedRune('x')
			}
		})
	}
}
