// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/lexington/rules"
)

// completionsIn are the labels the completion offers after typed, in a
// script in lang.
func completionsIn(lang, script, typed string) []string {
	text := script + typed
	l := rules.DefaultConf().Language(lang)
	c := completeIn(text, len([]rune(text)), completeCtx{starts: sceneStarts(lang), l: l})
	if c == nil {
		return nil
	}
	return c.labels
}

// A heading completes slot by slot in each language's own order: the
// place after the scene word or before it, the time, inside/outside,
// with the language's words and the script's places.
func TestCompleteSlots(t *testing.T) {
	for _, c := range []struct {
		lang, script, typed string
		want                []string // among the labels, in this order
	}{
		{"en", "INT. KITCHEN - DAY\n\nRain.\n\n", "EXT. K", []string{"KITCHEN"}},
		{"en", "INT. KITCHEN - DAY\n\nRain.\n\n", "EXT. KITCHEN - ", []string{"DAY"}},
		{"en", "INT. KITCHEN - DAY\n\nRain.\n\n", "in", []string{"INT.", "INT./EXT."}},
		{"eo", "EN. KUIREJO - TAGO\n\nPluvo.\n\n", "EKST. K", []string{"KUIREJO"}},
		{"eo", "EN. KUIREJO - TAGO\n\nPluvo.\n\n", "EKST. KUIREJO -", []string{"TAGO"}},
		{"zh", "场景1 客厅 日 内\n\n下雨。\n\n", "场景2 客", []string{"客厅"}},
		{"zh", "场景1 客厅 日 内\n\n下雨。\n\n", "场景2 客厅 ", []string{"日", "夜"}},
		{"zh", "场景1 客厅 日 内\n\n下雨。\n\n", "场景2 客厅 夜 ", []string{"内", "外"}},
		{"hi-Deva", "रसोई/दिन/अंदर\n\nबारिश।\n\n", "रस", []string{"रसोई"}},
		{"hi-Deva", "रसोई/दिन/अंदर\n\nबारिश।\n\n", "रसोई/", []string{"दिन", "रात"}},
		{"hi-Deva", "रसोई/दिन/अंदर\n\nबारिश।\n\n", "रसोई/रात/", []string{"अंदर", "बाहर"}},
		{"ar", "مشهد 1 المطبخ نهار/داخلي\n\nمطر.\n\n", "مشهد 2 الم", []string{"المطبخ"}},
		{"ar", "مشهد 1 المطبخ نهار/داخلي\n\nمطر.\n\n", "مشهد 2 المطبخ ", []string{"نهار", "ليل"}},
		{"he", "1. פנים מטבח\n\nגשם.\n\n", "2. ", []string{"פנים", "חוץ"}},
		{"ja", "○台所（朝）\n\n雨。\n\n", "○台", []string{"台所"}},
		{"ja", "○台所（朝）\n\n雨。\n\n", "○台所（", []string{"朝", "昼", "夕", "夜"}},
	} {
		got := completionsIn(c.lang, c.script, c.typed)
		at := -1
		for _, w := range c.want {
			i := slices.Index(got, w)
			if i <= at {
				t.Errorf("%s %q: %v, want %v in order", c.lang, c.typed, got, c.want)
				break
			}
			at = i
		}
	}
	// an action line is no heading: nothing offered
	if got := completionsIn("en", "INT. KITCHEN - DAY\n\n", "Rain falls"); got != nil {
		t.Errorf("action: %v", got)
	}
}

// A cue completes its name (the scene's speakers before the script's
// others, the one who spoke last after them), then the language's
// extensions after "NAME (", the closing bracket typed for it kept one.
func TestCompleteCueAndExtension(t *testing.T) {
	script := "INT. A - DAY\n\nBOB\nHi.\n\nBOB\nHi.\n\nBOB\nHi.\n\nINT. B - DAY\n\nBETTY\nHo.\n\nBEN\nHa.\n\n"
	if got := completionsIn("en", script, "B"); !slices.Equal(got, nil) {
		t.Logf("one letter: %v", got)
	}
	if got := completionsIn("en", script, "BE"); !slices.Equal(got, []string{"BETTY", "BEN"}) {
		t.Errorf("names: %v", got) // BEN spoke last: BETTY first
	}
	if got := completionsIn("en", script, "BEN (V"); !slices.Equal(got, []string{"V.O."}) {
		t.Errorf("extension: %v", got)
	}
	if got := completionsIn("zh", "张三\n你好。\n\n", "张三（画"); !slices.Contains(got, "画外音") {
		t.Errorf("zh extension: %v", got)
	}
	// the cursor before an automatic )
	text := script + "BEN ()"
	c := completeIn(text, len([]rune(text))-1, completeCtx{starts: sceneStarts("en"), l: rules.DefaultConf().Language("en")})
	if c == nil || c.candidates[0] != "BEN (V.O.)" || c.prefix != "BEN ()" {
		t.Fatalf("before ): %+v", c)
	}
}

// The list under the line: shown while typing a name, ↓ chooses, Enter
// takes the one chosen, Esc closes it, no list (and the keys the
// editor's) when nothing matches; vim's insert mode too.
func TestCompletionListKeys(t *testing.T) {
	w := flowWindow(t, "ANNA\nHi.\n\nANDREW\nHo.\n\nANNA\nBye.\n\n") // ANNA spoke last: ANDREW first
	typeKeys(t, w, "AN")
	p := w.compl
	if !p.open() || !slices.Equal(p.c.labels, []string{"ANDREW", "ANNA"}) || p.chosen != -1 {
		t.Fatalf("list: %v", p.c)
	}
	w.textEditor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	w.textEditor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	if p.chosen != 1 {
		t.Fatalf("chosen %d", p.chosen)
	}
	if !strings.HasSuffix(w.textEditor.Text(), "\n\nAN") {
		t.Fatalf("an arrow moved the cursor: %q", w.textEditor.Text())
	}
	typeKeys(t, w, "\n")
	if !strings.HasSuffix(w.textEditor.Text(), "\n\nANNA") {
		t.Fatalf("Enter: %q", w.textEditor.Text())
	}
	// Esc closes it; typing on opens it again
	w.textEditor.Undo()
	typeKeys(t, w, "D")
	if !p.open() {
		t.Fatal("no list after typing on")
	}
	typeKeys(t, w, "<esc>")
	if p.open() {
		t.Error("Esc left it open")
	}
	typeKeys(t, w, "\t") // closed: Tab is the flow's (or the completer's), not the list's
	// nothing matches: no list, Down moves the cursor
	w2 := flowWindow(t, "ANNA\nHi.\n\n")
	typeKeys(t, w2, "ZE")
	if w2.compl.open() || w2.compl.key(&fyne.KeyEvent{Name: fyne.KeyDown}) {
		t.Error("a list with nothing in it")
	}
	// a heading, slot by slot, with Tab
	w3 := flowWindow(t, "INT. KITCHEN - DAY\n\nRain.\n\n")
	typeKeys(t, w3, "EXT. K\t - N\t")
	if !strings.HasSuffix(w3.textEditor.Text(), "\n\nEXT. KITCHEN - NIGHT") {
		t.Errorf("Tab by slots: %q", w3.textEditor.Text())
	}
	// vim's insert mode
	w4 := flowWindow(t, "ANNA\nHi.\n\n")
	w4.settings.SetString("editor-keys", "vim")
	t.Cleanup(func() { w4.settings.SetString("editor-keys", "standard") })
	w4.vim.setMode(vimInsert)
	typeKeys(t, w4, "AN")
	if !w4.compl.open() {
		t.Error("no list in vim's insert mode")
	}
	w4.vim.setMode(vimNormal)
	w4.showSuggestionHint()
	if w4.compl.open() {
		t.Error("a list in vim's normal mode")
	}
}

// Typing a key in a long script, with and without the completion at the
// cursor (the per-key path: the list follows every key).
func BenchmarkTypingCompletion(b *testing.B) {
	script := strings.Repeat("INT. KITCHEN - DAY\n\nANNA\nHello there.\n\nEXT. GARDEN - NIGHT\n\nBOB\nHi.\n\n", 200)
	l := rules.DefaultConf().Language("en")
	cx := completeCtx{starts: sceneStarts("en"), l: l}
	b.Run("heading", func(b *testing.B) {
		text := script + "INT. KIT"
		for b.Loop() {
			completeIn(text, len([]rune(text)), cx)
		}
	})
	b.Run("action", func(b *testing.B) {
		text := script + "Rain falls on the"
		for b.Loop() {
			completeIn(text, len([]rune(text)), cx)
		}
	})
}
