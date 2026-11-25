package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// spOriginal is a screenplay as its print reads (a page number and a
// (MORE) left in its text); spConverted the same script after a rough
// conversion: a cue with (CONT'D), a speech wrapped over lines with a
// word broken at a line's end, a page number left in, a line cut short,
// a word garbled, a speech broken by a blank line, the last scene lost.
const spOriginal = `Title: The Ferry

INT. FERRY CABIN - NIGHT

The engine hums. MARTA counts coins on the table.

MARTA
Three crossings left, then we are done with this boat.

JONAS
You said that
(MORE)
last winter.

2.

EXT. FERRY DECK - NIGHT

Rain on the rail. Jonas lights a cigarette that will not stay lit.

JONAS
Tell me the truth about the money.

MARTA
The truth is that there never was any money.

INT. HARBOUR OFFICE - DAY

A clerk stamps papers without looking up.

CLERK
Next.
`

const spConverted = `Title: The Ferry

INT. FERRY CABIN - NIGHT

The engine hums. MARTA counts coins on the table.

MARTA
Three crossings left, then we are done
with this boat.

JONAS
You said that last winter.

EXT. FERRY DECK - NIGHT

Rain on the rail. Jonas lights a cigarette.

12.

JONAS
Tell me the truth

about the money.

MARTA (CONT'D)
The truth is that there never was any mo-
ney at alll.
`

// A screenplay reads as one; a story, a stage of Markdown, does not.
func TestScreenplayShaped(t *testing.T) {
	if !screenplayShaped(spOriginal, nil) || !screenplayShaped(spConverted, nil) {
		t.Error("a screenplay not seen as one")
	}
	if screenplayShaped(comboStory, nil) {
		t.Error("a story seen as a screenplay")
	}
	if screenplayShaped("", nil) {
		t.Error("nothing seen as a screenplay")
	}
}

// What a source's pages leave in its text is read out of it; wrapping
// and forcing marks are no difference.
func TestSourceKeyDebris(t *testing.T) {
	clean := [][2]string{
		{"MARTA (CONT'D)\nThe truth is that there never was any mo-\nney.", "MARTA\nThe truth is that there never was any money."},
		{"JONAS\nYou said that\n(MORE)\nlast winter.", "JONAS\nYou said that\nlast winter."},
		{"Action.\n\n12.\n\n- 4 -\n\nCONTINUED:\n\nPage 3 of 9\n\nMore action (continued) here.", "Action.\n\n\n\n\n\nMore action (continued) here."},
	}
	for _, p := range clean {
		if got := withoutDebris(p[0]); got != p[1] {
			t.Errorf("%q read as %q, want %q", p[0], got, p[1])
		}
	}
	key := sourceKey(mergeCarried{}, forcedHeading)
	same := [][2]string{
		{"MARTA\nThree crossings left, then we are done\nwith this boat.", "MARTA\nThree crossings left, then we are done with this boat."},
		{"@McKAY\nHello.", "McKAY\nHello."},
	}
	for _, p := range same {
		if a, b := key(p[0]), key(p[1]); a != b {
			t.Errorf("%q and %q differ: %q, %q", p[0], p[1], a, b)
		}
	}
	if key("Rain on the rail.") == key("Rain on the roof.") || key("12.") == "" {
		t.Error("a changed word, or the script's own page number, is no difference")
	}
}

// spWindow is a window on the converted script with the original
// beside it as its source.
func spWindow(t *testing.T) *MainWindow {
	t.Helper()
	w, _ := sourceWindow(t, spConverted, spOriginal)
	s := w.sourcePane()
	s.follow()
	if !s.screenplay {
		t.Fatal("the source is not read as a screenplay")
	}
	return w
}

// spCursorTo puts the cursor on the script's line holding s, and the pane
// follows.
func spCursorTo(t *testing.T, w *MainWindow, s string) {
	t.Helper()
	line := lineWith(t, w, s)
	w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(line), false) })
	w.sourcePane().follow()
}

// A screenplay source is compared, never adapted: no passages, no
// guesses, the script byte for byte as it was; the summary counts the
// scenes, the debris counted as nothing.
func TestScreenplaySourceLeavesScript(t *testing.T) {
	w := spWindow(t)
	s := w.sourcePane()
	if len(s.passages) != 0 || len(w.guessTags()) != 0 {
		t.Errorf("%d passages, %d guesses", len(s.passages), len(w.guessTags()))
	}
	if w.textEditor.Text() != spConverted || w.textEditor.Buffer().CanUndo() || w.hasChanges {
		t.Errorf("the script changed (changes %v)", w.hasChanges)
	}
	if got, want := s.count.Text, "1 scene matches · 1 differs · 1 missing"; got != want {
		t.Errorf("summary %q, want %q", got, want)
	}
	spCursorTo(t, w, "INT. FERRY CABIN")
	if !strings.Contains(s.info.Text, "matches the source") || len(s.sp.rows.Objects) != 0 {
		t.Errorf("the first scene (only debris) shows %q, %d rows", s.info.Text, len(s.sp.rows.Objects))
	}
	comboShot(t, w, "spsource-match")
}

// Take the Source's writes the source's form, one undo step, the
// conversion's debris left out of it; Keep Mine only hides it.
func TestScreenplaySourceTake(t *testing.T) {
	w := spWindow(t)
	s := w.sourcePane()
	spCursorTo(t, w, "EXT. FERRY DECK")
	sc := s.sp.current()
	ids, _ := s.sp.inScene(sc, lineWith(t, w, "EXT. FERRY DECK"))
	if len(ids) != 6 {
		for _, id := range ids {
			t.Logf("%+v", sc.m.Changes[id])
		}
		t.Fatalf("%d differences in the deck scene, want 6 (the cigarette, the page number, the speech in two pieces, the money, the lost scene after it)", len(ids))
	}
	hs := sc.hunks(ids)
	if len(hs) != 5 || len(hs[2]) != 2 {
		t.Fatalf("hunks %v, want the speech's two pieces as one", hs)
	}
	comboShot(t, w, "spsource-differ")
	s.sp.take(ids[0])
	after := w.textEditor.Text()
	if !strings.Contains(after, "lights a cigarette that will not stay lit.") {
		t.Errorf("not taken:\n%s", after)
	}
	if strings.Count(after, "\n") != strings.Count(spConverted, "\n") || !strings.Contains(after, "\n12.\n") {
		t.Errorf("taking one line moved others:\n%s", after)
	}
	w.textEditor.Undo()
	if w.textEditor.Text() != spConverted {
		t.Errorf("undo:\n%s", w.textEditor.Text())
	}
	// the speech in two pieces: taken whole, its words once
	s.sp.take(hs[2]...)
	if after := w.textEditor.Text(); !strings.Contains(after, "JONAS\nTell me the truth about the money.\n") || strings.Count(after, "about the money") != 1 {
		t.Errorf("pieces taken as:\n%s", after)
	}
	w.textEditor.Undo()
	// the speech: the source's words, no "mo-ney", no (CONT'D)
	s.follow()
	sc = s.sp.current()
	ids, _ = s.sp.inScene(sc, lineWith(t, w, "EXT. FERRY DECK"))
	var speech int = -1
	for _, id := range ids {
		if strings.Contains(sc.m.Changes[id].Theirs, "never was any money") {
			speech = id
		}
	}
	if speech < 0 {
		t.Fatal("no difference for the garbled speech")
	}
	s.sp.take(speech)
	if after := w.textEditor.Text(); !strings.Contains(after, "MARTA\nThe truth is that there never was any money.") || strings.Contains(after, "alll") {
		t.Errorf("speech taken as:\n%s", after)
	}
	w.textEditor.Undo()
	// Keep Mine: gone from the pane and the count, nothing written
	s.follow()
	sc = s.sp.current()
	ids, _ = s.sp.inScene(sc, lineWith(t, w, "EXT. FERRY DECK"))
	s.sp.keepMine(sc, ids[:1])
	if w.textEditor.Text() != spConverted {
		t.Error("Keep Mine wrote")
	}
	if left, _ := s.sp.inScene(sc, lineWith(t, w, "EXT. FERRY DECK")); len(left) != len(ids)-1 {
		t.Errorf("%d left after Keep Mine, want %d", len(left), len(ids)-1)
	}
	if !strings.Contains(s.count.Text, "1 difference kept as yours") {
		t.Errorf("summary %q", s.count.Text)
	}
}

// Next and Previous go to the scenes that differ, round at the ends;
// the missing scene is offered where it goes, and taken whole.
func TestScreenplaySourceStep(t *testing.T) {
	w := spWindow(t)
	s := w.sourcePane()
	spCursorTo(t, w, "Title: The Ferry")
	w.nextSourcePassage()
	line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
	if deck := lineWith(t, w, "EXT. FERRY DECK"); line < deck {
		t.Errorf("Next went to line %d, before the deck scene (%d)", line, deck)
	}
	w.nextSourcePassage() // the end: the missing office scene goes after the last
	w.nextSourcePassage() // round to the start
	line, _ = w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
	if deck := lineWith(t, w, "EXT. FERRY DECK"); line < deck {
		t.Errorf("round: line %d", line)
	}
	w.previousSourcePassage()
	sc := s.sp.current()
	var added int = -1
	for _, c := range sc.m.Changes {
		if strings.Contains(c.Theirs, "HARBOUR OFFICE") {
			added = c.ID
		}
	}
	if added < 0 {
		t.Fatal("the missing scene is not offered")
	}
	s.sp.take(added)
	if !strings.Contains(w.textEditor.Text(), "INT. HARBOUR OFFICE - DAY\n\nA clerk stamps papers") {
		t.Errorf("the scene not taken:\n%s", w.textEditor.Text())
	}
}

// Choose Source… on a screenplay's PDF (Lexington's own print of it)
// reads it as a screenplay, kept as a Fountain.
func TestChooseScreenplaySourcePDF(t *testing.T) {
	w, _ := sourceWindow(t, spConverted, "")
	os.Remove(filepath.Join(filepath.Dir(w.currentFile), "story.md"))
	pdf := filepath.Join(t.TempDir(), "ferry.pdf")
	if err := exportAs(spOriginal, pdf, "pdf", exportJob{}); err != nil {
		t.Fatal(err)
	}
	holdUI(t)
	w.chooseSourceFile(pdf)
	waitForText(t, w)
	var ok bool
	var kept string
	fyne.DoAndWait(func() {
		s := w.sourcePane()
		s.follow()
		ok, kept = s.screenplay, s.ref.Path
	})
	if !ok {
		t.Fatal("the PDF is not read as a screenplay source")
	}
	if filepath.Ext(kept) != ".fountain" {
		t.Errorf("kept as %s", kept)
	}
}
