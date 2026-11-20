package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"
)

// typeKeys types s into the window's editor as real key events: \t is Tab,
// \n is Enter, <esc> Escape, the rest characters.
func typeKeys(t *testing.T, w *MainWindow, s string) {
	t.Helper()
	w.fyneWindow.Canvas().Focus(w.textEditor)
	for s != "" {
		switch {
		case strings.HasPrefix(s, "<esc>"):
			w.textEditor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
			s = s[5:]
			continue
		case s[0] == '\t':
			w.textEditor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyTab})
		case s[0] == '\n':
			w.textEditor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
		default:
			r := []rune(s)[0]
			w.textEditor.TypedRune(r)
			s = s[len(string(r)):]
			continue
		}
		s = s[1:]
	}
}

// elements are the script's elements as Lexington reads them: "type:text"
// (empty lines left out).
func elements(text string) []string {
	var out []string
	for _, l := range fountain.Parse(englishScenes, strings.NewReader(text)) {
		if l.Type == lex.TypeEmpty {
			continue
		}
		out = append(out, l.Type+":"+strings.TrimSpace(l.Contents))
	}
	return out
}

func flowWindow(t *testing.T, text string) *MainWindow {
	t.Helper()
	w := newTestWindow(t, text)
	w.textEditor.SetCursorOffset(len([]rune(text)))
	return w
}

// Each step of the flow, typed with real keys, gives ordinary Fountain
// that Lexington reads as the elements meant.
func TestKeyFlowTransitions(t *testing.T) {
	for _, tc := range []struct {
		name, start, typed, want string
		elems                    []string
	}{
		{"heading Enter goes to action", "", "int. barn - day\nRain.",
			"INT. BARN - DAY\n\nRain.", []string{"scene:INT. BARN - DAY", "action:Rain."}},
		{"a second Enter after a heading adds nothing", "", "int. barn - day\n\nRain.",
			"INT. BARN - DAY\n\nRain.", []string{"scene:INT. BARN - DAY", "action:Rain."}},
		{"Tab on an empty action line starts a cue, typed in capitals", "Rain.\n\n", "\tanna\nHello.",
			"Rain.\n\nANNA\nHello.", []string{"action:Rain.", "speaker:ANNA", "dialog:Hello."}},
		{"Tab right under action puts the blank line in", "Rain.\n", "\tanna\nHello.",
			"Rain.\n\nANNA\nHello.", []string{"action:Rain.", "speaker:ANNA", "dialog:Hello."}},
		{"Tab after a typed name goes to its dialogue", "Rain.\n\n", "TOM\tHi.",
			"Rain.\n\nTOM\nHi.", []string{"action:Rain.", "speaker:TOM", "dialog:Hi."}},
		{"Tab after the cue Tab started goes to its dialogue", "Rain.\n\n", "\ttom\tHi.",
			"Rain.\n\nTOM\nHi.", []string{"action:Rain.", "speaker:TOM", "dialog:Hi."}},
		{"Tab in an empty dialogue line makes a parenthetical; Enter goes on in dialogue", "TOM\n", "\tbeat\nHi.",
			"TOM\n(beat)\nHi.", []string{"speaker:TOM", "paren:(beat)", "dialog:Hi."}},
		{"Tab at a dialogue line's end: a parenthetical under it", "TOM\nHi.", "\tpause\nThere.",
			"TOM\nHi.\n(pause)\nThere.", []string{"speaker:TOM", "dialog:Hi.", "paren:(pause)", "dialog:There."}},
		{"Tab in a parenthetical goes on in dialogue too", "TOM\n", "\tbeat\tHi.",
			"TOM\n(beat)\nHi.", []string{"speaker:TOM", "paren:(beat)", "dialog:Hi."}},
		{"Enter in an empty parenthetical leaves it for the dialogue", "TOM\n", "\t\nHi.",
			"TOM\nHi.", []string{"speaker:TOM", "dialog:Hi."}},
		{"Enter on an empty dialogue line goes to action", "TOM\nHi.", "\n\nHe leaves.",
			"TOM\nHi.\n\nHe leaves.", []string{"speaker:TOM", "dialog:Hi.", "action:He leaves."}},
		{"a whole exchange", "", "ext. pier - night\nWaves.\n\tmara\nWho's there?\n\n\tthe gull\t\tsquawks\nKraa.\n\nShe laughs.",
			"EXT. PIER - NIGHT\n\nWaves.\n\nMARA\nWho's there?\n\nTHE GULL\n(squawks)\nKraa.\n\nShe laughs.",
			[]string{"scene:EXT. PIER - NIGHT", "action:Waves.", "speaker:MARA", "dialog:Who's there?",
				"speaker:THE GULL", "paren:(squawks)", "dialog:Kraa.", "action:She laughs."}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := flowWindow(t, tc.start)
			w.settings.SetBoolean("auto-close-brackets", true)
			typeKeys(t, w, tc.typed)
			got := w.textEditor.Text()
			if got != tc.want {
				t.Fatalf("text %q, want %q", got, tc.want)
			}
			if e := elements(got); strings.Join(e, "|") != strings.Join(tc.elems, "|") {
				t.Errorf("Lexington reads %q, want %q", e, tc.elems)
			}
		})
	}
}

// The cue Tab started shows as Character; leaving the line ends it.
func TestKeyFlowCueState(t *testing.T) {
	w := flowWindow(t, "Rain.\n\n")
	typeKeys(t, w, "\t")
	if !w.cueHere() || w.currentElement != "Character" {
		t.Fatalf("after Tab: cue %v, element %q", w.cueHere(), w.currentElement)
	}
	w.textEditor.SetCursorOffset(0)
	w.onCursorChanged()
	typeKeys(t, w, "x")
	if w.cueHere() || !strings.HasPrefix(w.textEditor.Text(), "xRain.") {
		t.Errorf("the cue went on off its line: %q", w.textEditor.Text())
	}
	// in a script without capitals the cue gets Fountain's @
	if got := asCue("שרה"); got != "@שרה" {
		t.Errorf("asCue = %q", got)
	}
}

// Completion goes first while its hint is shown: Tab completes the name,
// a second Tab (one name to give) goes on to the dialogue; with several,
// Tab goes through them and Enter goes on.
func TestKeyFlowCompletionFirst(t *testing.T) {
	w := flowWindow(t, "ANNABEL\nHi.\n\n")
	typeKeys(t, w, "AN")
	if h := status(w); !strings.Contains(h, "Tab: ANNABEL") {
		t.Fatalf("hint %q", h)
	}
	typeKeys(t, w, "\t")
	if got := w.textEditor.Text(); got != "ANNABEL\nHi.\n\nANNABEL" {
		t.Fatalf("Tab did not complete: %q", got)
	}
	typeKeys(t, w, "\tBye.")
	if got := w.textEditor.Text(); got != "ANNABEL\nHi.\n\nANNABEL\nBye." {
		t.Errorf("the second Tab: %q", got)
	}

	w2 := flowWindow(t, "ANNA\nHi.\n\nANNABEL\nHo.\n\n")
	typeKeys(t, w2, "AN\t\t")
	if got := w2.textEditor.Text(); !strings.HasSuffix(got, "\n\nANNABEL") && !strings.HasSuffix(got, "\n\nANNA") {
		t.Fatalf("Tab through the names: %q", got)
	}
	typeKeys(t, w2, "\nYes.")
	if e := elements(w2.textEditor.Text()); e[len(e)-1] != "dialog:Yes." {
		t.Errorf("Enter after a completed name: %q", e)
	}
}

// The source's suggestion goes first while it is shown; after Esc, Tab
// is the flow's again.
func TestKeyFlowSuggestionFirst(t *testing.T) {
	w := suggestWindow(t, suggestScript)
	w.showSuggestionHint()
	if !strings.HasPrefix(status(w), "Tab: action") {
		t.Fatalf("hint %q", status(w))
	}
	typeKeys(t, w, "\t")
	if got := tail(w); got == "" || w.cueHere() {
		t.Errorf("the suggestion was not taken: %q", got)
	}
	w2 := suggestWindow(t, suggestScript)
	w2.showSuggestionHint()
	typeKeys(t, w2, "<esc>\t")
	if got := tail(w2); got != "" || !w2.cueHere() {
		t.Errorf("after Esc: %q, cue %v", got, w2.cueHere())
	}
}

// vim and helix: normal mode keeps Tab and Enter (Enter moves down,
// nothing is written); insert mode has the flow.
func TestKeyFlowVimHelix(t *testing.T) {
	for _, mode := range []string{"vim", "helix"} {
		t.Run(mode, func(t *testing.T) {
			text := "INT. BARN - DAY\n\nRain.\n\n"
			w := flowWindow(t, text)
			w.settings.SetString("editor-keys", mode)
			t.Cleanup(func() { w.settings.SetString("editor-keys", "standard") })
			w.vim.setMode(vimNormal)
			w.textEditor.SetCursorOffset(0)
			typeKeys(t, w, "\t\n")
			if got := w.textEditor.Text(); got != text {
				t.Fatalf("normal mode Tab/Enter wrote: %q", got)
			}
			if w.cueHere() {
				t.Errorf("normal mode Tab started a cue")
			}
			w.textEditor.SetCursorOffset(len([]rune(text)))
			w.vim.setMode(vimInsert)
			typeKeys(t, w, "\tanna\nHello.")
			if got := w.textEditor.Text(); got != text+"ANNA\nHello." {
				t.Errorf("insert mode: %q", got)
			}
		})
	}
}

// In a /* … */ block (the boneyard, the Notes and Source blocks) Tab and
// Enter are a text editor's.
func TestKeyFlowNotInBlocks(t *testing.T) {
	start := "INT. BARN - DAY\n\nRain.\n\n/* Notes\nkeep\n\n"
	w := flowWindow(t, start)
	typeKeys(t, w, "\t")
	if w.cueHere() {
		t.Errorf("Tab started a cue in the notes block")
	}
	typeKeys(t, w, "int. x\nab")
	if got := w.textEditor.Text(); !strings.HasSuffix(strings.ToUpper(got), "\nKEEP\n\nINT. X\nAB") || strings.Contains(got, "INT. X\n\n") {
		t.Errorf("in the notes block: %q", got)
	}
}

// Markdown documents, auto-indent off and a selection keep Tab as it was.
func TestKeyFlowOff(t *testing.T) {
	w := flowWindow(t, "Rain.\n\n")
	w.settings.SetBoolean("auto-indent", false)
	t.Cleanup(func() { w.settings.SetBoolean("auto-indent", true) })
	typeKeys(t, w, "\t")
	if got := w.textEditor.Text(); got != "Rain.\n\n    " {
		t.Errorf("auto-indent off: %q", got)
	}
	w2 := flowWindow(t, "INT. BARN - DAY\n\nRain.\n\n")
	w2.setMarkdown(true)
	w2.textEditor.SetCursorOffset(len([]rune(w2.textEditor.Text())))
	typeKeys(t, w2, "\t")
	if w2.cueHere() {
		t.Errorf("a cue in Markdown")
	}
}
