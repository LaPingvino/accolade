package main

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Suggestions while writing (suggest.go), driven by keys as a writer
// types them.

const suggestStory = `# Chapter 1

Anna opened the kitchen door. “Where were you?” Anna whispered.

“Out,” said Tom. He shook the rain from his coat. “Is there tea?”

“No,” said Anna. “There is never tea.”

“Then I will make some,” Tom said.

* * *

The next morning the street was white with snow.
`

// suggestScript's one scene links the four kitchen passages.
const suggestScript = `Title: Snow

INT. KITCHEN - NIGHT

[[from: "Anna opened the kitchen door."]] [[from: "Out, said Tom. He shook"]] [[from: "No, said Anna. There is"]] [[from: "Then I will make some, Tom said."]]

`

func tabKey(w *MainWindow) { w.textEditor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyTab}) }
func escKey(w *MainWindow) { w.textEditor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape}) }

func typeText(w *MainWindow, s string) {
	for _, r := range s {
		if r == '\n' {
			w.textEditor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
			continue
		}
		w.textEditor.TypedRune(r)
	}
}

// toEnd puts the cursor at the end of the script.
func toEnd(w *MainWindow) {
	w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.Len(), false) })
}

// status is what the status bar says.
func status(w *MainWindow) string { return w.statusText() }

// tail is the script after the fixture's text.
func tail(w *MainWindow) string { return tailOf(w, suggestScript) }

// tailOf is the script after script.
func tailOf(w *MainWindow, script string) string {
	text := w.textEditor.Text()
	if !strings.HasPrefix(text, script) {
		return "(changed) " + text
	}
	return text[len(script):]
}

func suggestWindow(t *testing.T, script string) *MainWindow {
	t.Helper()
	w, _ := sourceWindow(t, script, suggestStory)
	toEnd(w)
	return w
}

// A cue offers that character's next line not in the scene, with its
// parenthetical; Tab takes it, Tab again goes to the next, round.
func TestSuggestCueLine(t *testing.T) {
	w := suggestWindow(t, suggestScript)
	typeText(w, "TOM")
	if h := status(w); !strings.HasPrefix(h, "Tab: TOM’s line: “Out. Is there tea?”") || !strings.Contains(h, "from the source · 1 of 2") {
		t.Fatalf("hint %q", h)
	}
	tabKey(w)
	if got := tail(w); got != "TOM\nOut. Is there tea?" {
		t.Fatalf("taken: %q", got)
	}
	if h := status(w); !strings.Contains(h, "Tab: next, TOM’s line: “Then I will make some.”") {
		t.Errorf("next hint %q", h)
	}
	tabKey(w)
	if got := tail(w); got != "TOM\nThen I will make some." {
		t.Errorf("Tab again: %q", got)
	}
	tabKey(w)
	if got := tail(w); got != "TOM\nOut. Is there tea?" {
		t.Errorf("round: %q", got)
	}
	// one undo step each: undone, the cue is as typed
	w.textEditor.Undo()
	w.textEditor.Undo()
	w.textEditor.Undo()
	if got := tail(w); got != "TOM" {
		t.Errorf("undone: %q", got)
	}
	// the parenthetical from the manner, on the cue's empty line too
	typeText(w, "\n\nANNA\n")
	if h := status(w); !strings.HasPrefix(h, "Tab: ANNA’s line (whispering): “Where were you?”") {
		t.Fatalf("dialogue line hint %q", h)
	}
	tabKey(w)
	if got := tail(w); !strings.HasSuffix(got, "ANNA\n(whispering)\nWhere were you?") {
		t.Errorf("under the cue: %q", got)
	}
}

// A cue being typed completes from the passages' speakers first, before
// the script's own (ANDREW speaks more), mapped to the script's cue when
// it has one (ANNA K for Anna).
func TestSuggestCueCompletion(t *testing.T) {
	script := suggestScript + "ANDREW\nHi.\n\nANDREW\nHi again.\n\n"
	w := suggestWindow(t, script)
	typeText(w, "AN")
	if h := status(w); !strings.HasPrefix(h, "Tab: ANNA · ANDREW") {
		t.Fatalf("hint %q", h)
	}
	tabKey(w)
	if !strings.HasSuffix(w.textEditor.Text(), "\nANNA") {
		t.Fatalf("completed: %q", w.textEditor.Text())
	}
	if h := status(w); h != "Tab: ANDREW" {
		t.Errorf("going through: %q", h)
	}
	tabKey(w) // the completions go on
	if !strings.HasSuffix(w.textEditor.Text(), "\nANDREW") {
		t.Errorf("Tab again: %q", w.textEditor.Text())
	}

	// the script's own cue for a speaker
	w2 := suggestWindow(t, suggestScript+"ANNA K\nHello.\n\n")
	typeText(w2, "ANNA K")
	if h := status(w2); !strings.HasPrefix(h, "Tab: ANNA K’s line (whispering): “Where were you?”") {
		t.Errorf("mapped cue: %q", h)
	}
}

// An empty line offers the passages' next piece not in the scene: the
// narration as action (as written), then a line with its cue.
func TestSuggestEmptyLine(t *testing.T) {
	w := suggestWindow(t, suggestScript)
	if h := status(w); h != "" && !strings.HasPrefix(h, "Tab: ") {
		t.Logf("status %q", h)
	}
	w.showSuggestionHint()
	if h := status(w); !strings.HasPrefix(h, "Tab: action, as the source has it: “Anna opened the kitchen door.”") || !strings.Contains(h, "1 of 6") {
		t.Fatalf("hint %q", h)
	}
	tabKey(w)
	if got := tail(w); got != "Anna opened the kitchen door." {
		t.Fatalf("taken: %q", got)
	}
	tabKey(w)
	if got := tail(w); got != "ANNA\n(whispering)\nWhere were you?" {
		t.Errorf("Tab again: %q", got)
	}
	tabKey(w)
	tabKey(w)
	if got := tail(w); got != "He shook the rain from his coat." {
		t.Errorf("fourth: %q", got)
	}
	// on the next empty line: what the scene has is not offered again
	for tail(w) != "" {
		w.textEditor.Undo()
	}
	tabKey(w) // the action
	typeText(w, "\n\n")
	if h := status(w); !strings.HasPrefix(h, "Tab: ANNA (whispering): “Where were you?”") || !strings.Contains(h, "1 of 5") {
		t.Errorf("after the action: %q", h)
	}
	// right under an element: a blank line goes before the piece
	w3 := suggestWindow(t, suggestScript+"Anna opened the kitchen door.")
	typeText(w3, "\n")
	tabKey(w3)
	if got := tail(w3); got != "Anna opened the kitchen door.\n\nANNA\n(whispering)\nWhere were you?" {
		t.Errorf("under action: %q", got)
	}
}

// A line the scene has (words matched, case and punctuation aside) is
// not offered again; with all of them in, nothing is.
func TestSuggestUsed(t *testing.T) {
	w := suggestWindow(t, suggestScript+"TOM\nout! Is there TEA\n\n")
	typeText(w, "TOM")
	if h := status(w); !strings.HasPrefix(h, "Tab: TOM’s line: “Then I will make some.”") || strings.Contains(h, " of ") {
		t.Fatalf("hint %q", h)
	}
	all := suggestScript + strings.Join([]string{
		"Anna opened the kitchen door.",
		"ANNA\n(whispering)\nWhere were you?",
		"TOM\nOut. Is there tea?",
		"He shook the rain from his coat.",
		"ANNA\nNo. There is never tea.",
		"TOM\nThen I will make some.",
	}, "\n\n") + "\n\n"
	w2 := suggestWindow(t, all)
	w2.showSuggestionHint()
	if h := status(w2); strings.HasPrefix(h, "Tab: ") {
		t.Errorf("all used, still: %q", h)
	}
	tabKey(w2)
	if got := w2.textEditor.Text(); got != all || !w2.cueHere() {
		t.Errorf("Tab with nothing to offer on an empty line starts a cue (keyflow.go): %q", got[len(all):])
	}
	typeText(w2, "\nTOM")
	if h := status(w2); strings.HasPrefix(h, "Tab: TOM’s") {
		t.Errorf("a used cue: %q", h)
	}
}

// Esc puts the suggestion away (Tab then does what it did before);
// typing on, the next one comes from where the cursor is.
func TestSuggestDismiss(t *testing.T) {
	w := suggestWindow(t, suggestScript)
	w.showSuggestionHint()
	if !strings.HasPrefix(status(w), "Tab: action") {
		t.Fatalf("hint %q", status(w))
	}
	escKey(w)
	if h := status(w); h != "" {
		t.Errorf("after Esc: %q", h)
	}
	w.onCursorChanged()
	if h := status(w); h != "" {
		t.Errorf("came back: %q", h)
	}
	tabKey(w)
	if got := tail(w); got != "" || !w.cueHere() {
		t.Errorf("Tab after Esc: %q, not a cue started (keyflow.go)", got)
	}
	w.textEditor.Undo()
	// typing on: action being written has nothing to take
	w2 := suggestWindow(t, suggestScript)
	w2.showSuggestionHint()
	typeText(w2, "Rain")
	if h := status(w2); strings.HasPrefix(h, "Tab: ") {
		t.Errorf("typing on: %q", h)
	}
	tabKey(w2)
	if got := tail(w2); got != "Rain    " {
		t.Errorf("Tab in action: %q", got)
	}
}

// Preferences > Script > Adapting turns them off: no hint, Tab as before.
func TestSuggestPreference(t *testing.T) {
	w := suggestWindow(t, suggestScript)
	w.settings.SetBoolean(suggestPref, false)
	t.Cleanup(func() { w.settings.SetBoolean(suggestPref, true) })
	w.showSuggestionHint()
	if h := status(w); strings.HasPrefix(h, "Tab: ") {
		t.Errorf("off, a hint: %q", h)
	}
	tabKey(w)
	if got := tail(w); got != "" || !w.cueHere() {
		t.Errorf("off, Tab: %q, not a cue started (keyflow.go)", got)
	}
	w.textEditor.Undo()
	typeText(w, "AN")
	if h := status(w); strings.HasPrefix(h, "Tab: ANNA") {
		t.Errorf("off, the source's names: %q", h)
	}
	// the dialog has it
	pd := NewPreferencesDialog(w.app, w.fyneWindow)
	pd.loadFrom(w.settings)
	if pd.suggestCheck == nil || pd.suggestCheck.Checked {
		t.Errorf("the dialog's check: %+v", pd.suggestCheck)
	}
}

// Without a source nothing is suggested; with the source pane never
// shown, it is read for the suggestions all the same.
func TestSuggestNoSource(t *testing.T) {
	w, _ := scriptWindow(t, suggestScript)
	toEnd(w)
	w.showSuggestionHint()
	if h := status(w); strings.HasPrefix(h, "Tab: ") {
		t.Errorf("no source, a hint: %q", h)
	}
	tabKey(w)
	if got := tail(w); got != "" || !w.cueHere() {
		t.Errorf("no source, Tab: %q, not a cue started (keyflow.go)", got)
	}

	w2, p := scriptWindow(t, suggestScript)
	os.WriteFile(filepath.Join(filepath.Dir(p), "story.md"), []byte(suggestStory), 0o644)
	if err := w2.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if w2.paneInSight("source") {
		t.Fatal("the pane is shown")
	}
	toEnd(w2)
	w2.showSuggestionHint()
	if h := status(w2); !strings.HasPrefix(h, "Tab: action") {
		t.Errorf("pane hidden: %q", h)
	}
}

// A new scene's heading is offered as the source reads the next
// passage's scene: EXT. for a street, its time.
func TestSuggestHeading(t *testing.T) {
	w := suggestWindow(t, suggestScript)
	typeText(w, "ext")
	if h := status(w); !strings.HasPrefix(h, "Tab: EXT. STREET - MORNING") {
		t.Fatalf("hint %q", h)
	}
	tabKey(w)
	if got := tail(w); got != "EXT. STREET - MORNING" {
		t.Errorf("taken: %q", got)
	}
	w.textEditor.Undo()
	typeText(w, "\n\nINT. ") // as the writer has it, the place read
	w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.Len(), false) })
	w.onCursorChanged()
	if h := status(w); !strings.HasPrefix(h, "Tab: INT. STREET - MORNING") {
		t.Errorf("INT.: %q", h)
	}
}

// A line no one is named for comes under any cue after the character's
// own, but a name or heading being typed is still completed (AN, EXT).
func TestSuggestUnattributed(t *testing.T) {
	story := strings.Replace(suggestStory, "Anna opened the kitchen door. “Where were you?” Anna whispered.", "The kitchen door opened. “Where were you?”", 1)
	script := strings.Replace(suggestScript, "Anna opened the kitchen door.", "The kitchen door opened.", 1)
	w, _ := sourceWindow(t, script, story)
	toEnd(w)
	typeText(w, "ANNA")
	if h := status(w); !strings.HasPrefix(h, "Tab: ANNA’s line: “No. There is never tea.”") || !strings.Contains(h, "1 of 2") {
		t.Fatalf("hint %q", h)
	}
	tabKey(w)
	tabKey(w)
	if got := tailOf(w, script); got != "ANNA\nWhere were you?" {
		t.Errorf("no one's line: %q", got)
	}
	for tailOf(w, script) != "" {
		w.textEditor.Undo()
	}
	typeText(w, "AN")
	tabKey(w)
	if got := tailOf(w, script); got != "ANNA" {
		t.Errorf("AN, Tab: %q", got)
	}
	for tailOf(w, script) != "" {
		w.textEditor.Undo()
	}
	typeText(w, "EXT")
	tabKey(w)
	if got := tailOf(w, script); got != "EXT. STREET - MORNING" {
		t.Errorf("EXT, Tab: %q", got)
	}
	for tailOf(w, script) != "" {
		w.textEditor.Undo()
	}
	// a cue of its own for it: the line, as the cue's
	typeText(w, "MAID")
	if h := status(w); !strings.HasPrefix(h, "Tab: “Where were you?”, no one named for it") {
		t.Errorf("a new cue: %q", h)
	}
}

// A name completed with Tab, the next Tab gives its line (the name is
// the only one to complete).
func TestSuggestCompleteThenLine(t *testing.T) {
	w := suggestWindow(t, suggestScript)
	typeText(w, "TO")
	tabKey(w)
	if got := tail(w); got != "TOM" {
		t.Fatalf("completed: %q", got)
	}
	if h := status(w); !strings.HasPrefix(h, "Tab: TOM’s line") {
		t.Errorf("hint %q", h)
	}
	tabKey(w)
	if got := tail(w); got != "TOM\nOut. Is there tea?" {
		t.Errorf("Tab again: %q", got)
	}
}

// A scene with no link yet, the passage Next picked is the one
// suggested from.
func TestSuggestNextShown(t *testing.T) {
	script := "Title: Snow\n\nINT. KITCHEN - NIGHT\n\n[[from: \"Anna opened the kitchen door.\"]]\n\nEXT. STREET - DAY\n\n"
	w, _ := sourceWindow(t, script, suggestStory)
	toEnd(w)
	w.showSuggestionHint()
	if h := status(w); strings.HasPrefix(h, "Tab: ") {
		t.Fatalf("no link, nothing shown, still: %q", h)
	}
	w.nextSourcePassage()
	if p := w.sourcePane().offeredPassage(); p < 0 || !strings.HasPrefix(w.sourcePane().passages[p].Plain, "“Out,”") {
		t.Fatalf("Next showed %d", p)
	}
	toEnd(w)
	w.showSuggestionHint()
	if h := status(w); !strings.HasPrefix(h, "Tab: TOM: “Out. Is there tea?”") {
		t.Errorf("from Next's: %q", h)
	}
}

// A phone has no Tab: a tap on the hint's chip does what Tab does,
// takes the suggestion (one undo step) and goes on to the next; it
// completes names too. The chip is there only with a hint, and the bar
// stays within a phone's width.
func TestSuggestTapChip(t *testing.T) {
	w := suggestWindow(t, suggestScript)
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	typeText(w, "TOM")
	if !w.tabChip.Visible() || !strings.HasPrefix(w.statusLabel.Text, "TOM’s line") {
		t.Fatalf("chip %v, label %q", w.tabChip.Visible(), w.statusLabel.Text)
	}
	w.fyneWindow.Content().Refresh()
	row := w.statusLabel
	if right := row.Position().X + row.Size().Width; right > 380 || row.Position().X < w.tabChip.Size().Width {
		t.Errorf("the hint at %v, %v wide, the chip %v", row.Position(), row.Size(), w.tabChip.Size())
	}
	test.Tap(w.tabChip)
	if got := tail(w); got != "TOM\nOut. Is there tea?" {
		t.Fatalf("tapped: %q", got)
	}
	test.Tap(w.tabChip)
	if got := tail(w); got != "TOM\nThen I will make some." {
		t.Errorf("tapped again: %q", got)
	}
	w.textEditor.Undo()
	w.textEditor.Undo()
	if got := tail(w); got != "TOM" {
		t.Errorf("undone: %q", got)
	}
	// a completion
	typeText(w, "\n\nANDREW\nHi.\n\nAN")
	if h := status(w); !strings.HasPrefix(h, "Tab: ANNA · ANDREW") || !w.tabChip.Visible() {
		t.Fatalf("hint %q", h)
	}
	test.Tap(w.tabChip)
	if !strings.HasSuffix(w.textEditor.Text(), "\nANNA") {
		t.Errorf("completed by a tap: %q", w.textEditor.Text())
	}
	test.Tap(w.tabChip)
	if !strings.HasSuffix(w.textEditor.Text(), "\nANDREW") {
		t.Errorf("the next by a tap: %q", w.textEditor.Text())
	}
	// no hint, no chip
	typeText(w, " is here")
	if w.tabChip.Visible() || status(w) != "" {
		t.Errorf("after typing on: chip %v, %q", w.tabChip.Visible(), status(w))
	}
}

// Screenshots of a cue's suggestion and a narration's, at a desktop's
// width and a phone's (SHOT_DIR).
func TestSuggestScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("SHOT_DIR not set")
	}
	os.MkdirAll(dir, 0o755)
	for _, c := range []struct {
		name  string
		typed string
	}{{"cue", "TOM"}, {"narration", ""}} {
		for _, size := range []struct {
			name string
			size fyne.Size
		}{{"wide", fyne.NewSize(1100, 650)}, {"phone", fyne.NewSize(380, 800)}} {
			w := suggestWindow(t, suggestScript)
			w.fyneWindow.Resize(size.size)
			typeText(w, c.typed)
			w.showSuggestionHint()
			w.fyneWindow.Canvas().Focus(w.textEditor)
			w.fyneWindow.Content().Refresh()
			f, err := os.Create(filepath.Join(dir, "suggest-"+c.name+"-"+size.name+".png"))
			if err != nil {
				t.Fatal(err)
			}
			png.Encode(f, w.fyneWindow.Canvas().Capture())
			f.Close()
		}
	}
}

// What a keystroke costs with suggestions on and off: the novel and
// its 200-scene script, a rune typed and taken away in a linked scene,
// the cursor on an empty line in it, where a suggestion is shown.
func BenchmarkSuggestKeystroke(b *testing.B) {
	for _, on := range []bool{true, false} {
		b.Run(map[bool]string{true: "on", false: "off"}[on], func(b *testing.B) {
			w, off := benchSuggestWindow(b, on)
			realTimers(b)
			e := w.textEditor
			i := 0
			for b.Loop() {
				e.Edit(func(bf *buffer.Buffer) { // the action under the empty line, the cursor kept
					if i%2 == 0 {
						bf.Replace(off+1, off+1, "x")
					} else {
						bf.Replace(off+1, off+2, "")
					}
					bf.SetCursor(off, false)
				})
				cancel(&w.followTimer)
				cancel(&w.suggester().loadTimer)
				i++
			}
			if h := status(w); on != strings.Contains(h, "from the source") {
				b.Errorf("hint %q", h)
			}
		})
	}
}

// benchSuggestWindow is benchWindow with an empty line put in scene 100,
// the cursor on it, what the source offers read; off: suggestions off.
func benchSuggestWindow(b *testing.B, on bool) (*MainWindow, int) {
	w, s, off := benchWindow(b, false)
	w.settings.SetBoolean(suggestPref, on)
	b.Cleanup(func() { w.settings.SetBoolean(suggestPref, true) })
	w.textEditor.Replace(off, off, "\n")
	w.textEditor.Navigate(func(bf *buffer.Buffer) { bf.SetCursor(off, false) })
	w.suggester().offersFor(s)
	w.showSuggestionHint()
	if h := status(w); on != strings.Contains(h, "from the source") {
		b.Fatalf("hint %q", h)
	}
	return w, off
}

// The hint's own share of a keystroke, on and off (off: the completion
// hint as before), where a suggestion is shown: the edit, and the text it
// makes, which every part of a keystroke shares, are outside the timing.
func BenchmarkSuggestHint(b *testing.B) {
	for _, on := range []bool{true, false} {
		b.Run(map[bool]string{true: "on", false: "off"}[on], func(b *testing.B) {
			w, off := benchSuggestWindow(b, on)
			sg := w.suggester()
			bf := w.textEditor.Buffer()
			i := 0
			h := ""
			for b.Loop() {
				b.StopTimer()
				if i%2 == 0 {
					bf.Replace(off+1, off+1, "x")
				} else {
					bf.Replace(off+1, off+2, "")
				}
				bf.SetCursor(off, false)
				bf.Text()
				i++
				b.StartTimer()
				h = sg.hint()
			}
			if on != strings.Contains(h, "from the source") {
				b.Errorf("hint %q", h)
			}
		})
	}
}
