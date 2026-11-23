package main

import (
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/structure"
)

// Scenes and sections move with what they hold, before another or to the
// end, as the outline and the cards drag them; each one undo step.
func TestMoveOutline(t *testing.T) {
	const script = "# ACT ONE\n\nINT. BARN - DAY\n\nRain.\n\nEXT. FIELD - DAY\n\nWind.\n\n# ACT TWO\n\nINT. HOUSE - NIGHT\n\nFire.\n"
	w, _ := newDialogTestWindow(t)
	e := w.textEditor
	for _, c := range []struct {
		from, to int
		want     string
	}{
		// the field before the barn
		{2, 1, "# ACT ONE\n\nEXT. FIELD - DAY\n\nWind.\n\nINT. BARN - DAY\n\nRain.\n\n# ACT TWO\n\nINT. HOUSE - NIGHT\n\nFire.\n"},
		// act two first, with its scene
		{3, 0, "# ACT TWO\n\nINT. HOUSE - NIGHT\n\nFire.\n\n# ACT ONE\n\nINT. BARN - DAY\n\nRain.\n\nEXT. FIELD - DAY\n\nWind.\n"},
		// the barn to the end
		{1, 5, "# ACT ONE\n\nEXT. FIELD - DAY\n\nWind.\n\n# ACT TWO\n\nINT. HOUSE - NIGHT\n\nFire.\n\nINT. BARN - DAY\n\nRain.\n"},
		// the house into act one
		{4, 2, "# ACT ONE\n\nINT. BARN - DAY\n\nRain.\n\nINT. HOUSE - NIGHT\n\nFire.\n\nEXT. FIELD - DAY\n\nWind.\n\n# ACT TWO\n"},
		// an act into itself, a section before a scene (taking it in): nothing
		{0, 1, script},
		{3, 2, script},
	} {
		e.SetText(script)
		w.moveOutlineItem(c.from, c.to)
		if got := e.Text(); got != c.want {
			t.Errorf("%d before %d:\n%q\nwant\n%q", c.from, c.to, got, c.want)
		}
		e.Undo()
		if got := e.Text(); got != script {
			t.Errorf("%d before %d, one undo: %q", c.from, c.to, got)
		}
	}
}

// Dragging an outline row up one row moves its scene before that one.
func TestOutlineDrag(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText("INT. BARN - DAY\n\nRain.\n\nEXT. FIELD - DAY\n\nWind.\n")
	if !w.paneShown("outline") {
		w.togglePane("outline")
	}
	o := w.outline
	o.refresh()
	if o.dropIndex(1, -o.oneLine) != 0 || o.dropIndex(0, o.oneLine*1.5) != 2 || o.dropIndex(0, 2) != -1 {
		t.Fatalf("drop places: %d %d %d", o.dropIndex(1, -o.oneLine), o.dropIndex(0, o.oneLine*1.5), o.dropIndex(0, 2))
	}
	r := newDragRow(o, container.NewStack())
	r.id = 1
	r.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DY: -o.oneLine}})
	r.DragEnd()
	if got := w.textEditor.Text(); got != "EXT. FIELD - DAY\n\nWind.\n\nINT. BARN - DAY\n\nRain.\n" {
		t.Errorf("dragged: %q", got)
	}
	w.textEditor.Undo()
	if got := w.textEditor.Text(); got != "INT. BARN - DAY\n\nRain.\n\nEXT. FIELD - DAY\n\nWind.\n" {
		t.Errorf("one undo: %q", got)
	}
}

// pressKey is a key as the window gets it: a menu item's (in a submenu
// too) first, as Fyne looks, else keyPreview's; it reports whether
// something took it.
func pressKey(w *MainWindow, key fyne.KeyName, mod fyne.KeyModifier) bool {
	var find func(items []*fyne.MenuItem) *fyne.MenuItem
	find = func(items []*fyne.MenuItem) *fyne.MenuItem {
		for _, it := range items {
			if s, ok := it.Shortcut.(*desktop.CustomShortcut); ok && s.KeyName == key && s.Modifier == mod {
				return it
			}
			if it.ChildMenu != nil {
				if f := find(it.ChildMenu.Items); f != nil {
					return f
				}
			}
		}
		return nil
	}
	for _, m := range w.buildMainMenu().Items {
		if it := find(m.Items); it != nil {
			it.Action()
			return true
		}
	}
	return w.keyPreview(key, mod)
}

const structScript = "# ACT ONE\n\nINT. BARN - DAY\n\nRain.\n\nANNA\nIt leaks.\n\nBOB\nFix it.\n\nEXT. FIELD - DAY\n\nWind.\n"

// The keys, in every mode (standard, vim, helix): each move one undo
// step, the cursor with what moved; in vim and helix evil-org's Alt keys
// too.
func TestStructureKeys(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	t.Cleanup(func() { w.settings.SetString("editor-keys", "standard"); w.applyEditorSettings() })
	e := w.textEditor
	at := func(line string) {
		t.Helper()
		e.SetCursorOffset(len([]rune(e.Text()[:strings.Index(e.Text(), line)])))
	}
	for _, mode := range []string{"standard", "vim", "helix"} {
		w.settings.SetString("editor-keys", mode)
		w.applyEditorSettings()
		for _, c := range []struct {
			key   fyne.KeyName
			mod   fyne.KeyModifier
			at    string
			want  string
			stays string // the cursor's line after
		}{
			{fyne.KeyUp, fyne.KeyModifierAlt, "BOB", strings.Replace(structScript, "ANNA\nIt leaks.\n\nBOB\nFix it.", "BOB\nFix it.\n\nANNA\nIt leaks.", 1), "BOB"},
			{fyne.KeyDown, fyne.KeyModifierAlt, "INT. BARN", "# ACT ONE\n\nEXT. FIELD - DAY\n\nWind.\n\nINT. BARN - DAY\n\nRain.\n\nANNA\nIt leaks.\n\nBOB\nFix it.\n", "INT. BARN"},
			{fyne.KeyLeft, fyne.KeyModifierAlt, "# ACT", "", ""}, // at the top already: refused, nothing changes
			{fyne.KeyRight, fyne.KeyModifierAlt, "Rain.", strings.Replace(structScript, "Rain.\n\nANNA\n", "ANNA\nRain.\n", 1), "Rain."},
			{fyne.KeyDown, fyne.KeyModifierAlt | fyne.KeyModifierShift, "BOB", strings.Replace(structScript, "BOB\nFix it.\n\nEXT. FIELD - DAY\n\nWind.", "EXT. FIELD - DAY\n\nBOB\nFix it.\n\nWind.", 1), "BOB"},
			{fyne.KeyRight, fyne.KeyModifierAlt | fyne.KeyModifierShift, "Rain.", strings.Replace(structScript, "Fix it.\n\nEXT. FIELD - DAY\n\nWind.", "Fix it.\n\nWind.\n\nEXT. FIELD - DAY", 1), ""},
			{fyne.KeyT, fyne.KeyModifierAlt, "Fix it.", strings.Replace(strings.Replace(structScript, "ANNA\nIt", "BOB\nIt", 1), "BOB\nFix", "ANNA\nFix", 1), "Fix it."},
		} {
			e.SetText(structScript)
			if mode != "standard" {
				e.TypedKey(escape())
			}
			at(c.at)
			if !pressKey(w, c.key, c.mod) {
				t.Fatalf("%s: %s+%v went nowhere", mode, c.key, c.mod)
			}
			want := c.want
			if want == "" {
				want = structScript
			}
			if got := e.Text(); got != want {
				t.Errorf("%s: %v+%s at %s:\n%s\nwant\n%s", mode, c.mod, c.key, c.at, got, want)
				continue
			}
			if c.stays != "" {
				if l := strings.Split(e.Text(), "\n")[cursorLine(w)]; !strings.HasPrefix(l, c.stays) {
					t.Errorf("%s: %v+%s: the cursor on %q, not with %q", mode, c.mod, c.key, l, c.stays)
				}
			}
			if want != structScript {
				e.Undo()
				if e.Text() != structScript {
					t.Errorf("%s: %v+%s: not one undo step: %q", mode, c.mod, c.key, e.Text())
				}
			}
		}
		// expand and shrink the selection by unit
		e.SetText(structScript)
		if mode != "standard" {
			e.TypedKey(escape())
		}
		at("It leaks.")
		sel := func() string {
			a, b := e.Buffer().Selection()
			return string([]rune(e.Text())[a:b])
		}
		for _, want := range []string{"It leaks.", "ANNA\nIt leaks.", "INT. BARN - DAY\n\nRain.\n\nANNA\nIt leaks.\n\nBOB\nFix it."} {
			pressKey(w, fyne.KeyO, fyne.KeyModifierAlt)
			if got := sel(); got != want {
				t.Errorf("%s: expanded to %q, want %q", mode, got, want)
			}
		}
		pressKey(w, fyne.KeyI, fyne.KeyModifierAlt)
		if got := sel(); got != "ANNA\nIt leaks." {
			t.Errorf("%s: shrunk to %q", mode, got)
		}
		if mode == "standard" {
			continue
		}
		// evil-org's: Alt+j down, Alt+Shift+K into the scene before
		e.SetText(structScript)
		e.TypedKey(escape())
		at("ANNA")
		e.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyJ, Modifier: fyne.KeyModifierAlt})
		if got := e.Text(); got != strings.Replace(structScript, "ANNA\nIt leaks.\n\nBOB\nFix it.", "BOB\nFix it.\n\nANNA\nIt leaks.", 1) {
			t.Errorf("%s: Alt+j:\n%s", mode, got)
		}
		e.SetText(structScript)
		at("Wind.")
		e.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyK, Modifier: fyne.KeyModifierAlt | fyne.KeyModifierShift})
		if got := e.Text(); got != strings.Replace(structScript, "Fix it.\n\nEXT. FIELD - DAY\n\nWind.", "Fix it.\n\nWind.\n\nEXT. FIELD - DAY", 1) {
			t.Errorf("%s: Alt+Shift+K:\n%s", mode, got)
		}
	}
}

// The breakdown's tags, a scene marked done and the schedule's strips
// follow their scenes when a scene moves past another with the same
// heading (named by which of those it is), and back with Undo, on with
// Redo; locked numbers stay with their scenes and the status says so.
func TestStructureFollowsExtras(t *testing.T) {
	holdUI(t)
	const script = "INT. KITCHEN - NIGHT\n\nAnna loads the revolver.\n\nINT. KITCHEN - NIGHT\n\nBen hides the knife.\n\nEXT. STREET - DAY\n\nRain.\n"
	w, _ := scriptWindow(t, script)
	tagNow(t, w, "revolver", 0, "Props")
	tagNow(t, w, "knife", 0, "Props")
	w.markSceneDone(1) // the knife's kitchen
	sb := board(t, w)
	w.addShootDay()
	sb.selectScene(0, false)
	sb.moveSelected(0, 0)
	sb.selectScene(1, false)
	sb.moveSelected(0, 1)
	knife := func() int { // the knife's kitchen's index
		_, bt, _ := w.scheduleFound()
		for i, s := range bt.scenes {
			if strings.Contains(string(bt.runes[s.start:s.end]), "knife") {
				return i
			}
		}
		return -1
	}
	check := func(what string) {
		t.Helper()
		k := knife()
		if got, want := tagLine(w, "knife"), editorLineWith(t, w, "Ben hides the knife."); got != want {
			t.Errorf("%s: the knife's tag on line %d, not %d", what, got, want)
		}
		if got, want := tagLine(w, "revolver"), editorLineWith(t, w, "Anna loads the revolver."); got != want {
			t.Errorf("%s: the revolver's tag on line %d, not %d", what, got, want)
		}
		f, bt, p := w.scheduleFound()
		if got := p.scenesOf(0); !slices.Equal(got, []int{1 - k, k}) {
			t.Errorf("%s: day 1 %v (the knife is scene %d)", what, got, k)
		}
		for _, sc := range p.scenesOf(0) {
			if changed, _ := f.changed(bt, p, sc); changed {
				t.Errorf("%s: scene %d flagged as changed", what, sc)
			}
		}
		bf := w.breakdownNow()
		if !bf.isDone(bt.scenes[k]) || bf.isDone(bt.scenes[1-k]) {
			t.Errorf("%s: done %+v", what, bf.Done)
		}
	}
	check("before")
	w.showMiddle("") // the script in front, the editor with the keys
	w.fyneWindow.Canvas().Focus(w.textEditor)
	w.textEditor.SetCursorOffset(0)
	w.structDo(structure.MoveUp) // at the first: nothing
	pressKey(w, fyne.KeyDown, fyne.KeyModifierAlt)
	if !strings.HasPrefix(w.textEditor.Text(), "INT. KITCHEN - NIGHT\n\nBen hides") {
		t.Fatalf("not moved: %q", w.textEditor.Text())
	}
	check("moved")
	w.undo()
	if w.textEditor.Text() != script {
		t.Fatalf("undo: %q", w.textEditor.Text())
	}
	check("undo")
	w.redo()
	check("redo")

	// locked numbers stay with their scenes, and the status says so
	w.undo()
	check("undo again")
	w.lockSceneNumbers()
	check("locked, before")
	w.textEditor.SetCursorOffset(0)
	pressKey(w, fyne.KeyDown, fyne.KeyModifierAlt)
	if !strings.HasPrefix(w.textEditor.Text(), "INT. KITCHEN - NIGHT #2#\n\nBen hides") {
		t.Fatalf("locked, moved: %q", w.textEditor.Text()[:40])
	}
	if s := w.statusText(); !strings.Contains(s, "locked numbers stay with their scenes: 2, 1, 3") {
		t.Errorf("status %q", s)
	}
	check("locked")
}

// The Swap… and Move… picker: a range and another, or a place; its
// preview says what it makes or why not, and its button does it.
func TestStructPicker(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	e := w.textEditor
	const script = "# ACT ONE\n\nINT. BARN - DAY\n\nRain.\n\nEXT. FIELD - DAY\n\nWind.\n\n# ACT TWO\n\nINT. HOUSE - NIGHT\n\nFire.\n"
	e.SetText(script)
	e.SetCursorOffset(len("# ACT ONE\n\nINT. B"))
	w.showStructPicker(false, -1)
	p := w.structPick
	if p == nil {
		t.Fatal("no picker")
	}
	if p.aFrom.Selected != "1. INT. BARN - DAY" || p.bFrom.Selected != "2. EXT. FIELD - DAY" {
		t.Fatalf("picked %q with %q", p.aFrom.Selected, p.bFrom.Selected)
	}
	if !strings.HasPrefix(p.preview.Text, "INT. BARN - DAY and EXT. FIELD - DAY change places") || p.ok.Disabled() {
		t.Errorf("preview %q", p.preview.Text)
	}
	p.bFrom.SetSelected("3. INT. HOUSE - NIGHT")
	p.bTo.SetSelected("3. INT. HOUSE - NIGHT")
	if !strings.HasPrefix(p.preview.Text, "INT. BARN - DAY and INT. HOUSE - NIGHT change places") {
		t.Errorf("preview %q", p.preview.Text)
	}
	marked := 0
	for _, o := range p.after.Objects {
		if strings.Contains(o.(*widget.Label).Text, "→") {
			marked++
		}
	}
	if marked != 2 {
		t.Errorf("%d marked in the outline after", marked)
	}
	// a scene where a section is: refused, the button off
	p.bFrom.SetSelected("§ ACT TWO")
	if !strings.HasPrefix(p.preview.Text, "Not possible") || !p.ok.Disabled() {
		t.Errorf("preview %q", p.preview.Text)
	}
	// Move: the barn and the field after the house
	p.mode.SetSelected(pickMove)
	p.aTo.SetSelected("2. EXT. FIELD - DAY")
	p.where.SetSelected("after")
	p.target.SetSelected("3. INT. HOUSE - NIGHT")
	test.Tap(p.ok)
	if got := e.Text(); got != "# ACT ONE\n\n# ACT TWO\n\nINT. HOUSE - NIGHT\n\nFire.\n\nINT. BARN - DAY\n\nRain.\n\nEXT. FIELD - DAY\n\nWind.\n" {
		t.Errorf("moved:\n%s", got)
	}
	e.Undo()
	if e.Text() != script {
		t.Errorf("one undo: %q", e.Text())
	}
}

// Screenshots of Swap… (desktop and phone) and a before and after
// (SHOT_DIR).
func TestStructPickerScreenshots(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	w, _ := newDialogTestWindow(t)
	e := w.textEditor
	e.SetText(sampleStructScript)
	e.SetCursorOffset(len([]rune(sampleStructScript[:strings.Index(sampleStructScript, "INT. DINER")])))
	capture(t, w.fyneWindow.Canvas(), "structure-1-before.png")
	w.showStructPicker(false, -1)
	p := w.structPick
	p.bFrom.SetSelected(p.namesOf([]int{lineOfText(sampleStructScript, "EXT. PIER - NIGHT")})[0])
	capture(t, w.fyneWindow.Canvas(), "structure-2-swap.png")
	test.Tap(p.ok)
	capture(t, w.fyneWindow.Canvas(), "structure-3-after.png")
	e.Undo()
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.onWindowWidth(380)
	w.showStructPicker(true, -1)
	capture(t, w.fyneWindow.Canvas(), "structure-4-phone-move.png")
}

// lineOfText is the line of text that starts with s.
func lineOfText(text, s string) int {
	for i, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(l, s) {
			return i
		}
	}
	return -1
}

const sampleStructScript = `# ACT ONE

INT. DINER - NIGHT

Rain against the glass. MAYA (30s) stirs a coffee she will not drink.

MAYA
He said nine.

LEO
It's ten past.

EXT. PIER - NIGHT

Fog. A ferry horn, far off.

# ACT TWO

INT. FERRY - DAWN

Maya alone on a bench, the coffee cup still in her hand.
`

// On a Mac Option+Left/Right move by word: promote, demote, barf and
// slurp take Control+Option there, and the help pages say so.
func TestStructureMacKeys(t *testing.T) {
	for _, c := range structureCommands() {
		sideways := len(c.keys) > 0 && (c.keys[0].(*desktop.CustomShortcut).KeyName == fyne.KeyLeft || c.keys[0].(*desktop.CustomShortcut).KeyName == fyne.KeyRight)
		if !sideways {
			if c.macKeys != nil {
				t.Errorf("%s: other keys on a Mac", c.id)
			}
			continue
		}
		m, ok := c.macKeys[0].(*desktop.CustomShortcut)
		if !ok || m.Modifier&fyne.KeyModifierControl == 0 || m.Modifier&fyne.KeyModifierAlt == 0 {
			t.Errorf("%s: on a Mac %v", c.id, c.macKeys)
		}
		if n := macNote(c); !strings.HasPrefix(n, "on a Mac Control+") || !strings.Contains(n, "Option+") {
			t.Errorf("%s: the note %q", c.id, n)
		}
	}
}
