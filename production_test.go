package main

import (
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/LaPingvino/lexington/lex"
)

const lockScript = "Title: Rain\n\nINT. HOUSE - DAY\n\nRain.\n\nEXT. GARDEN - DAY [[red]]\n\nMore rain.\n\nINT. HOUSE - NIGHT\n\nDry.\n\n/* Notes\nkeep\n*/\n"

// headsOf are the script's headings' lines as written.
func headsOf(w *MainWindow) []string {
	var out []string
	for _, h := range sceneHeads(w.textEditor.Text(), w.sceneStarts()) {
		out = append(out, strings.TrimSpace(h.text))
	}
	return out
}

func TestLockAndUnlockSceneNumbers(t *testing.T) {
	w := newTestWindow(t, lockScript)
	w.textEditor.SetCursorOffset(strings.Index(lockScript, "More"))
	w.lockSceneNumbers()
	text := w.textEditor.Text()
	if got := strings.Join(headsOf(w), "|"); got != "INT. HOUSE - DAY #1#|EXT. GARDEN - DAY #2# [[red]]|INT. HOUSE - NIGHT #3#" {
		t.Errorf("headings %q", got)
	}
	if !strings.Contains(text, "Dry.\n\n/* Production\nscene numbers: locked, 10A\nnumbered before the lock: none\n*/\n\n/* Notes\nkeep\n*/\n") {
		t.Errorf("the block:\n%s", text)
	}
	if l := sceneLockOf(text); !l.locked || l.style != lex.NumberAfter {
		t.Errorf("lock %+v", l)
	}
	if got := w.textEditor.Text()[strings.Index(text, "More"):][:4]; got != "More" || w.textEditor.CursorOffset() != len([]rune(text[:strings.Index(text, "More")])) {
		t.Errorf("the cursor moved: at %d", w.textEditor.CursorOffset())
	}
	if !strings.Contains(notesOf(text), "keep") || bodyEnd(text) > strings.Index(text, "/* Production") {
		t.Error("the block is part of the script, or the notes lost")
	}
	// one undo step
	w.undo()
	if w.textEditor.Text() != lockScript {
		t.Errorf("undo:\n%s", w.textEditor.Text())
	}
	w.redo()
	// Unlock: the lock goes, the numbers stay
	w.unlockNow(false)
	text = w.textEditor.Text()
	if sceneLockOf(text).locked || strings.Contains(text, "Production") {
		t.Errorf("still locked:\n%s", text)
	}
	if !strings.Contains(text, "INT. HOUSE - NIGHT #3#\n\nDry.\n\n/* Notes\nkeep\n*/\n") {
		t.Errorf("unlocked:\n%s", text)
	}
}

// The other lines of the block stay, and the block is found among the
// others in any order.
func TestProductionBlockKeepsOtherLines(t *testing.T) {
	inside := "schedule: later\nscene numbers: locked, A11"
	text := "INT. A - DAY #1#\n\nHi.\n\n/* Source\nThe story.\n*/\n\n/* Production\n" + inside + "\n*/\n\n/* Adaptation\ncut:\n  \"x\"\n*/\n"
	if l := sceneLockOf(text); !l.locked || l.style != lex.NumberBefore {
		t.Errorf("lock %+v", l)
	}
	src, adapt := tailBlocks(text)
	if !src.ok || !adapt.ok || strings.Contains(src.inside, "Production") {
		t.Errorf("blocks %+v %+v", src, adapt)
	}
	if got := text[:bodyEndByte(text)]; got != "INT. A - DAY #1#\n\nHi." {
		t.Errorf("body %q", got)
	}
	if got := withSceneLock(inside, nil); got != "schedule: later" {
		t.Errorf("unlocked %q", got)
	}
	if got := withSceneLock("schedule: later", &sceneLock{locked: true}); got != "scene numbers: locked, 10A\nschedule: later" {
		t.Errorf("locked %q", got)
	}
	if sceneLockOf("INT. A\n\n/* Production\nscene numbers: unlocked\n*/\n").locked {
		t.Error("unlocked read as locked")
	}
}

// A scene written in a locked script gets its letter once the cursor
// has left its heading; Undo takes the heading and the number together.
func TestNewSceneGetsANumber(t *testing.T) {
	w := newTestWindow(t, lockScript)
	w.lockSceneNumbers()
	text := w.textEditor.Text()
	at := len([]rune(text[:strings.Index(text, "EXT. GARDEN")]))
	w.textEditor.SetCursorOffset(at)
	w.fyneWindow.Canvas().Focus(w.textEditor)
	before := w.textEditor.Text()
	test.Type(w.textEditor, "INT. SHED - DAY\n\nTools.\n\n")
	if got := strings.Join(headsOf(w), "|"); got != "INT. HOUSE - DAY #1#|INT. SHED - DAY #1A#|EXT. GARDEN - DAY #2# [[red]]|INT. HOUSE - NIGHT #3#" {
		t.Errorf("headings %q", got)
	}
	// typing a heading: not numbered while the cursor is on it; between
	// 1 and 1A there is only Wikipedia's 1aA
	w.textEditor.SetCursorOffset(at)
	test.Type(w.textEditor, "\n\n")
	w.textEditor.SetCursorOffset(at)
	test.Type(w.textEditor, "EXT. YARD")
	if got := headsOf(w)[1]; got != "EXT. YARD" {
		t.Errorf("numbered while typed: %q", got)
	}
	test.Type(w.textEditor, " - DAY")
	w.textEditor.SetCursorOffset(0) // the cursor leaves it
	if got := strings.Join(headsOf(w)[1:3], "|"); got != "EXT. YARD - DAY #1aA#|INT. SHED - DAY #1A#" {
		t.Errorf("before 1A: %q", got)
	}
	for w.textEditor.Buffer().CanUndo() && w.textEditor.Text() != before {
		w.undo()
	}
	if w.textEditor.Text() != before {
		t.Errorf("undone:\n%s", w.textEditor.Text())
	}
	if strings.Contains(w.textEditor.Text(), "#1A#") {
		t.Error("a number left behind by Undo")
	}
}

// Unlocked, nothing is numbered by itself.
func TestUnlockedScenesStayUnnumbered(t *testing.T) {
	w := newTestWindow(t, lockScript)
	w.fyneWindow.Canvas().Focus(w.textEditor)
	w.textEditor.SetCursorOffset(len([]rune(lockScript[:strings.Index(lockScript, "EXT. GARDEN")])))
	test.Type(w.textEditor, "INT. SHED - DAY\n\nTools.\n\n")
	if strings.Contains(w.textEditor.Text(), "#") {
		t.Errorf("numbered:\n%s", w.textEditor.Text())
	}
}

func TestOmitAndRestoreScene(t *testing.T) {
	w := newTestWindow(t, lockScript)
	w.lockSceneNumbers()
	locked := w.textEditor.Text()
	w.textEditor.SetCursorOffset(len([]rune(locked[:strings.Index(locked, "More rain")])))
	w.omitScene()
	text := w.textEditor.Text()
	want := "Rain.\n\n.OMITTED #2#\n\n/* Omitted 2\nEXT. GARDEN - DAY #2# [[red]]\n\nMore rain.\n*/\n\nINT. HOUSE - NIGHT #3#"
	if !strings.Contains(text, want) {
		t.Errorf("omitted:\n%s", text)
	}
	if got := strings.Join(headsOf(w), "|"); got != "INT. HOUSE - DAY #1#|.OMITTED #2#|INT. HOUSE - NIGHT #3#" {
		t.Errorf("headings %q", got)
	}
	// a scene written after it is 2A: 2 is kept by OMITTED
	if s := exportJobFor(text); !strings.Contains(s, "OMITTED #2#") {
		t.Errorf("export: %s", s)
	}
	w.textEditor.SetCursorOffset(len([]rune(text[:strings.Index(text, ".OMITTED")])) + 3)
	w.restoreScene()
	if w.textEditor.Text() != locked {
		t.Errorf("restored:\n%s", w.textEditor.Text())
	}
	w.undo()
	if w.textEditor.Text() != text {
		t.Errorf("undo of Restore:\n%s", w.textEditor.Text())
	}
	w.undo()
	if w.textEditor.Text() != locked {
		t.Errorf("undo of Omit:\n%s", w.textEditor.Text())
	}
	// the last scene, before the blocks at the end
	w.textEditor.SetCursorOffset(len([]rune(locked[:strings.Index(locked, "Dry.")])))
	w.omitScene()
	text = w.textEditor.Text()
	if !strings.Contains(text, ".OMITTED #3#\n\n/* Omitted 3\nINT. HOUSE - NIGHT #3#\n\nDry.\n*/\n\n/* Production\n") {
		t.Errorf("the last omitted:\n%s", text)
	}
	if sceneLockOf(text).locked != true || notesOf(text) != "keep" {
		t.Error("the blocks at the end lost")
	}
	// a scene without a number is not omitted
	w2 := newTestWindow(t, lockScript)
	w2.textEditor.SetCursorOffset(len([]rune(lockScript[:strings.Index(lockScript, "Rain.")])))
	w2.omitScene()
	if w2.textEditor.Text() != lockScript || !strings.Contains(w2.statusText(), "lock the scene numbers") {
		t.Errorf("omitted unnumbered: %q", w2.statusText())
	}
}

// exportJobFor is the script's scene headings as the export has them.
func exportJobFor(text string) string {
	var out []string
	for _, l := range (exportJob{}).script(text) {
		if l.Type == lex.TypeScene {
			out = append(out, l.Contents)
		}
	}
	return strings.Join(out, "|")
}

// A locked script's export numbers a scene that has none yet as it would
// be numbered; the A11 style as locked.
func TestExportOfLockedScript(t *testing.T) {
	text := "INT. A - DAY #10#\n\nHi.\n\nINT. B - DAY\n\nHo.\n\nINT. C - DAY #11#\n\n/* Production\nscene numbers: locked, A11\n*/\n"
	if got := exportJobFor(text); got != "INT. A - DAY #10#|INT. B - DAY #A11#|INT. C - DAY #11#" {
		t.Errorf("export %q", got)
	}
}

func TestProductionMenu(t *testing.T) {
	w := newTestWindow(t, "")
	for _, label := range []string{"Lock Scene Numbers", "Unlock Scene Numbers…", "Omit Scene", "Restore Scene"} {
		menuItemByLabel(t, w, label)
	}
	w.markdown = true
	for _, m := range w.buildMainMenu().Items {
		if m.Label == "Production" {
			t.Error("a Production menu in Markdown")
		}
	}
}

// The locked script with an A-scene and an omitted one, beside its
// outline, at a desktop's width and a phone's (SHOT_DIR=… go test -run
// TestProductionScreenshots).
func TestProductionScreenshots(t *testing.T) {
	if os.Getenv("SHOT_DIR") == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	w, _ := scriptWindow(t, lockScript)
	w.lockSceneNumbers()
	text := w.textEditor.Text()
	w.fyneWindow.Canvas().Focus(w.textEditor)
	w.textEditor.SetCursorOffset(len([]rune(text[:strings.Index(text, "EXT. GARDEN")])))
	test.Type(w.textEditor, "INT. SHED - DAY\n\nTools on the wall.\n\n")
	text = w.textEditor.Text()
	w.textEditor.SetCursorOffset(len([]rune(text[:strings.Index(text, "More rain")])))
	w.omitScene()
	if !strings.Contains(w.textEditor.Text(), "#1A#") || !strings.Contains(w.textEditor.Text(), ".OMITTED #2#") {
		t.Fatalf("not as shown:\n%s", w.textEditor.Text())
	}
	w.toggleOutline()
	comboShot(t, w, "production")
}

// Lock rewrites the headings with their numbers: the learner does not
// take a heading the script had when opened, now numbered, for one the
// writer gave its scene (adaptation_learn.go headingKey).
func TestLockTeachesNoPlace(t *testing.T) {
	script := strings.Replace(suggestScript, "INT. KITCHEN - NIGHT", "INT. BARN - NIGHT", 1)
	w, _ := sourceWindow(t, script, learnStory)
	moveTo(t, w, "[[from:")
	w.lockSceneNumbers()
	if !strings.Contains(w.textEditor.Text(), "INT. BARN - NIGHT #1#") {
		t.Fatalf("not locked:\n%s", w.textEditor.Text())
	}
	moveTo(t, w, "Title")
	moveTo(t, w, "[[from:")
	w.learnFromWriting()
	if b := learnedBlock(w); b != "" {
		t.Errorf("learned from Lock: %q", b)
	}
}

// Text typed after a heading's number: the number goes back to the end
// once the cursor has left, no second one given.
func TestStrayNumberMovesToTheEnd(t *testing.T) {
	w := newTestWindow(t, lockScript)
	w.lockSceneNumbers()
	text := w.textEditor.Text()
	at := strings.Index(text, "#2#") + len("#2#")
	w.fyneWindow.Canvas().Focus(w.textEditor)
	w.textEditor.SetCursorOffset(len([]rune(text[:at])))
	test.Type(w.textEditor, " - LATER")
	w.textEditor.SetCursorOffset(0)
	if got := headsOf(w)[1]; got != "EXT. GARDEN - DAY - LATER #2# [[red]]" {
		t.Errorf("heading %q", got)
	}
}
