package main

import (
	"runtime"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"sort"
	"strings"
	"unicode/utf8"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/accolade/internal/structure"
)

// Structural moves in the window (internal/structure has the engine,
// LOOP.md the why): the keys, the menu's Structure group, vim's and
// helix's keys, the outline's and the cards' dragging and their Swap… and
// Move… all come here. Each is one undo step; the cursor goes with what
// moved; the breakdown's tags and the schedule's strips follow the
// scenes (and back with Undo, as with a rename); source links and scene
// numbers are in the lines and go with them, so locked numbers stay with
// their scenes (the status says so).
//
//	Alt+Up / Alt+Down          the unit at the cursor (or the selected ones) past the one before / after
//	(on a Mac Control+Option+Left/Right and with Shift: Option+Left/Right
//	move by word there)
//	Alt+Left / Alt+Right       promote / demote: a section's level; a scene out of its section / into the
//	                           next; a speech's line out into action / action into the speech beside it
//	Alt+Shift+Up / Down        a speech or paragraph into the scene before / after
//	Alt+Shift+Left / Right     barf / slurp: the scene's last speech out into the next scene / the next
//	                           scene's first speech in
//	Alt+T                      the two speakers' lines (this speech's and the one before) change places
//	Alt+O / Alt+I              expand / shrink the selection by unit (helix's keys)

// onMac: the desktop on a Mac, where some keys are others (macKeys).
var onMac = runtime.GOOS == "darwin"

// structReader reads a text for the engine as the window does: the
// screenplay's elements in its language, its outline's headings, its
// body up to the blocks at its end; a Markdown document's paragraphs and
// headings.
func (w *MainWindow) structReader() structure.Reader {
	starts, md := w.sceneStarts(), w.markdown
	var d = w.dialect()
	return func(lines []string) structure.Script {
		text := strings.Join(lines, "\n")
		s := structure.Script{Lines: lines, End: len(lines)}
		var items []outlineItem
		if md {
			s.Kinds = make([]syntax.Kind, len(lines))
			for i, l := range lines {
				if strings.TrimSpace(l) == "" {
					s.Kinds[i] = syntax.Empty
				}
			}
			items = markdownOutline(text)
		} else {
			s.Kinds = syntax.ClassifyIn(lines, starts, d)
			items = outlineOf(text, starts)
			s.End = strings.Count(text[:bodyEndByte(text)], "\n")
			if be := bodyEndByte(text); be < len(text) && be > 0 && text[be-1] != '\n' {
				s.End++ // the block starts after a line's end
			}
		}
		for _, it := range items {
			s.Heads = append(s.Heads, structure.Head{Line: it.Line, Level: it.Level, Scene: it.Scene})
		}
		return s
	}
}

// structTree is the script's text read as its units.
func (w *MainWindow) structTree(text string) (*structure.Tree, structure.Reader) {
	read := w.structReader()
	return structure.Parse(read(structure.Lines(text))), read
}

// structEditor is the script's editor when moves can work in it: the
// script in front, a screenplay (a status says why not).
func (w *MainWindow) structEditor() *editor.ScriptEditor {
	if w.textEditor == nil {
		return nil
	}
	if w.markdown || !w.scriptInMain() {
		w.setStatus("Structure moves work in a screenplay's editor")
		return nil
	}
	return w.textEditor
}

// selLines are the lines of the editor's selection (a selection ending at
// a line's start leaves that line out), or the cursor's line twice; sel
// says whether there is a selection, col is the cursor's column.
func selLines(e *editor.ScriptEditor) (a, b int, sel bool, col int) {
	bf := e.Buffer()
	s, t := bf.Selection()
	a, _ = bf.Position(s)
	b, cb := bf.Position(t)
	if t > s && b > a && cb == 0 {
		b--
	}
	_, col = bf.Position(e.CursorOffset())
	cl, _ := bf.Position(e.CursorOffset())
	if s == t {
		a, b = cl, cl
	}
	return a, b, s != t, col
}

// structDo makes the move op at the cursor or on the selection.
func (w *MainWindow) structDo(op structure.Op) {
	e := w.structEditor()
	if e == nil {
		return
	}
	text := e.Text()
	tr, read := w.structTree(text)
	a, b, sel, col := selLines(e)
	cur, _ := e.Buffer().Position(e.CursorOffset())
	res, err := tr.Do(op, a, b, read)
	if err != nil {
		w.setStatus(err.Error())
		return
	}
	w.applyStruct(text, res, sel, cur, col)
}

// applyStruct puts a move's text in as one undo step, the extras
// following; what moved is selected again if it was, else the cursor
// stays where it was in it.
func (w *MainWindow) applyStruct(text string, res structure.Result, sel bool, cur, col int) {
	e := w.textEditor
	after := structure.Text(res.Lines)
	w.followNumbers(true) // the extras in line with the text before
	r := &followed{before: text, after: after}
	r.bd[0], r.sched[0] = w.extrasKept()
	w.followedLast = nil // (what follows the text's changes leaves it be meanwhile)
	w.noteNumbersOf(after)
	w.replaceText(text, after)
	fwd, back := moveEdit(text, after, res), moveEdit(after, text, invertResult(res))
	r.refollow = func(toBefore, tags, strips bool) {
		if toBefore {
			w.followEdit(r.after, r.before, back, tags, strips)
		} else {
			w.followEdit(r.before, r.after, fwd, tags, strips)
		}
	}
	w.followEdit(text, after, fwd, true, true)
	r.bd[1], r.sched[1] = w.extrasKept()
	r.atAfter = true
	w.followedLast = r
	w.noteNumbers() // scenes that moved, not numbers that changed
	e.Navigate(func(b *buffer.Buffer) {
		if sel {
			b.Select(b.LineStart(res.First), b.LineEnd(res.Last))
			return
		}
		line := res.First
		if cur >= 0 && cur < len(res.Pos) && res.Pos[cur] >= 0 {
			line = res.Pos[cur]
		}
		b.SetCursor(b.LineStart(line)+min(col, b.LineEnd(line)-b.LineStart(line)), false)
	})
	w.structSel = nil
	w.setStatus(res.Msg + lockedNote(text, after, w.sceneStarts()))
}

// lockedNote says, when the scene numbers are locked and their order
// changed, that they stayed with their scenes, and how they read now.
func lockedNote(before, after string, starts []string) string {
	if !sceneLockOf(before).locked {
		return ""
	}
	_, was, _ := sceneSignature(before, starts)
	_, now, _ := sceneSignature(after, starts)
	if slices.Equal(was, now) {
		return ""
	}
	var shown []string
	for _, n := range now {
		if n != "" {
			shown = append(shown, n)
		}
	}
	if len(shown) > 12 {
		shown = append(shown[:12], "…")
	}
	return " · the locked numbers stay with their scenes: " + strings.Join(shown, ", ")
}

// invertResult is the way back: each new line's old one.
func invertResult(res structure.Result) structure.Result {
	inv := func(m []int, n int) []int {
		out := make([]int, n)
		for i := range out {
			out[i] = -1
		}
		for i, j := range m {
			if j >= 0 && j < n {
				out[j] = i
			}
		}
		return out
	}
	return structure.Result{Map: inv(res.Map, len(res.Lines)), Pos: inv(res.Pos, len(res.Lines))}
}

// moveEdit is how the extras follow a move from before to after: an
// offset goes with its line when the line's text stayed (else -1), a
// scene with its heading.
func moveEdit(before, after string, res structure.Result) extrasEdit {
	oldStarts, newStarts := runeLineStarts(before), runeLineStarts(after)
	off := func(o int, end bool) int {
		l := sort.Search(len(oldStarts), func(i int) bool { return oldStarts[i] > o }) - 1
		if l < 0 || l >= len(res.Map) || res.Map[l] < 0 || res.Map[l] >= len(newStarts) {
			return -1
		}
		return newStarts[res.Map[l]] + o - oldStarts[l]
	}
	var oldLines, newLines []int // the scenes' heading lines
	var oldBt *bdText
	scene := func(i int) int {
		if oldBt == nil {
			return -1
		}
		if i < 0 || i >= len(oldLines) || oldLines[i] >= len(res.Pos) || res.Pos[oldLines[i]] < 0 {
			return -1
		}
		return slices.Index(newLines, res.Pos[oldLines[i]])
	}
	ed := extrasEdit{off: off, scene: scene}
	ed.prepare = func(o, n *bdText) {
		oldBt = o
		oldLines, newLines = nil, nil
		for _, s := range o.scenes {
			oldLines = append(oldLines, s.line)
		}
		for _, s := range n.scenes {
			newLines = append(newLines, s.line)
		}
	}
	return ed
}

// runeLineStarts are the runes where each line of text starts.
func runeLineStarts(text string) []int {
	out := []int{0}
	n := 0
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		i += size
		n++
		if r == '\n' {
			out = append(out, n)
		}
	}
	return out
}

// moveOutlineItem moves an outline item (with what it holds) in the
// window's script before the item to (len: to the end), through the
// engine: as one undo step, refused (the status says why) where it would
// put a unit in another's section.
func (w *MainWindow) moveOutlineItem(from, to int) {
	e := w.textEditor
	if e == nil {
		return
	}
	text := e.Text()
	items := w.outlineItems(text)
	if from < 0 || from >= len(items) || to < 0 || to > len(items) {
		return
	}
	tr, read := w.structTree(text)
	line := items[from].Line
	var res structure.Result
	var err error
	if to < len(items) {
		res, err = tr.Move(structure.Range{First: line, Last: line}, items[to].Line, false, read)
	} else { // the end: after the last at the top, with all it holds
		res, err = tr.Move(structure.Range{First: line, Last: line}, tr.LastTop(), true, read)
	}
	if err == structure.InPlace {
		return
	}
	if err != nil {
		w.setStatus(err.Error())
		return
	}
	cur, col := -1, 0
	if c := e.CursorOffset(); c >= 0 {
		cur, col = e.Buffer().Position(c)
	}
	w.applyStruct(text, res, false, cur, col)
}

// expandSelection selects the unit around the selection (the line, its
// speech, its scene, its section, …); shrinkSelection goes back a step.
func (w *MainWindow) expandSelection() {
	e := w.structEditor()
	if e == nil {
		return
	}
	bf := e.Buffer()
	s, t := bf.Selection()
	tr, _ := w.structTree(e.Text())
	a, b, sel, _ := selLines(e)
	first, last := a, b // the lines first, whole
	if sel && s == bf.LineStart(a) && t == bf.LineEnd(b) {
		f, l, ok := tr.Expand(a, b)
		if !ok {
			w.setStatus("All of the script's body is selected")
			return
		}
		first, last = f, l
	}
	if w.structSel == nil || w.structSel.rev != bf.Rev() {
		w.structSel = &structSelection{rev: bf.Rev()}
	}
	w.structSel.stack = append(w.structSel.stack, [2]int{s, t})
	e.Navigate(func(b *buffer.Buffer) { b.Select(b.LineStart(first), b.LineEnd(last)) })
}

func (w *MainWindow) shrinkSelection() {
	e := w.structEditor()
	if e == nil {
		return
	}
	bf := e.Buffer()
	st := w.structSel
	if st == nil || st.rev != bf.Rev() || len(st.stack) == 0 {
		c := e.CursorOffset()
		e.Navigate(func(b *buffer.Buffer) { b.SetCursor(c, false) })
		return
	}
	prev := st.stack[len(st.stack)-1]
	st.stack = st.stack[:len(st.stack)-1]
	e.Navigate(func(b *buffer.Buffer) {
		if prev[0] == prev[1] {
			b.SetCursor(prev[0], false)
		} else {
			b.Select(prev[0], prev[1])
		}
	})
}

// structSelection is the selections Expand Selection grew from, for
// Shrink Selection, while the text is as it was.
type structSelection struct {
	rev   uint64
	stack [][2]int
}

// structureCommands are the structural moves as commands: the keys, the
// menu (View > Structure), the ribbon's Edit > Structure, the palette and
// ex (:move-up, :promote, :slurp, :swap-ranges …).
func structureCommands() []*command {
	mv := func(op structure.Op) func(w *MainWindow) { return func(w *MainWindow) { w.structDo(op) } }
	// on a Mac Option+←/→ (and with Shift) move and select by word: there
	// the sideways moves take Control+Option
	mac := func(k fyne.KeyName, shift bool) []fyne.Shortcut {
		mod := fyne.KeyModifierControl | fyne.KeyModifierAlt
		if shift {
			mod |= fyne.KeyModifierShift
		}
		return []fyne.Shortcut{plainKey(k, mod)}
	}
	alt := func(k fyne.KeyName, shift bool) []fyne.Shortcut {
		mod := fyne.KeyModifierAlt
		if shift {
			mod |= fyne.KeyModifierShift
		}
		return []fyne.Shortcut{plainKey(k, mod)}
	}
	step := func(dir int, op structure.Op) func(w *MainWindow) {
		return func(w *MainWindow) { // a strip with the focus moves instead (stripboard.go); a Markdown document's lines
			if w.cardsShown() { // the current card's (cards_tab.go)
				w.cardsMoveBy(dir)
				return
			}
			w.orStrip(dir, w.do(&command{editing: true, run: func(w *MainWindow) {
				if w.editorMarkdown() {
					moveLines(w.writeEditor(), dir)
					return
				}
				w.structDo(op)
			}}))()
		}
	}
	return []*command{
		{id: "structure.up", label: "Move Up", ex: "move-up", keys: alt(fyne.KeyUp, false), icon: theme.MoveUpIcon, run: step(-1, structure.MoveUp),
			tip: "Move the section, scene, speech or paragraph at the cursor (or the selected ones) up past the one before"},
		{id: "structure.down", label: "Move Down", ex: "move-down", keys: alt(fyne.KeyDown, false), icon: theme.MoveDownIcon, run: step(1, structure.MoveDown),
			tip: "Move the section, scene, speech or paragraph at the cursor (or the selected ones) down past the next"},
		{id: "structure.promote", label: "Promote", ex: "promote", keys: alt(fyne.KeyLeft, false), macKeys: mac(fyne.KeyLeft, false), icon: theme.NavigateBackIcon, editing: true, when: screenplayOnly, run: mv(structure.Promote),
			onCards: cardsSideways(structure.Promote, -1),
			tip:     "Out: a section a level up, a scene out of its section, a speech's line out into action"},
		{id: "structure.demote", label: "Demote", ex: "demote", keys: alt(fyne.KeyRight, false), macKeys: mac(fyne.KeyRight, false), icon: theme.NavigateNextIcon, editing: true, when: screenplayOnly, run: mv(structure.Demote),
			onCards: cardsSideways(structure.Demote, 1),
			tip:     "In: a section a level down, a scene into the next section, action into the speech beside it"},
		{id: "structure.intoPrev", label: "Into the Scene Before", ex: "into-previous", keys: alt(fyne.KeyUp, true), icon: theme.UploadIcon, editing: true, when: screenplayOnly, run: mv(structure.IntoPrev),
			tip: "The speech or paragraph at the cursor into the end of the scene before"},
		{id: "structure.intoNext", label: "Into the Next Scene", ex: "into-next", keys: alt(fyne.KeyDown, true), icon: theme.DownloadIcon, editing: true, when: screenplayOnly, run: mv(structure.IntoNext),
			tip: "The speech or paragraph at the cursor into the start of the next scene"},
		{id: "structure.barf", label: "Pass the Last Speech On", ex: "barf", keys: alt(fyne.KeyLeft, true), macKeys: mac(fyne.KeyLeft, true), icon: theme.ContentRemoveIcon, editing: true, when: screenplayOnly, run: mv(structure.Barf),
			tip: "Barf: this scene's last speech or paragraph out into the next scene"},
		{id: "structure.slurp", label: "Take In the Next Speech", ex: "slurp", keys: alt(fyne.KeyRight, true), macKeys: mac(fyne.KeyRight, true), icon: theme.ContentAddIcon, editing: true, when: screenplayOnly, run: mv(structure.Slurp),
			tip: "Slurp: the next scene's first speech or paragraph into the end of this one"},
		{id: "structure.transpose", label: "Swap Speakers' Lines", ex: "transpose-speakers", keys: alt(fyne.KeyT, false), icon: theme.AccountIcon, editing: true, when: screenplayOnly, run: mv(structure.Transpose),
			tip: "This speech's lines to the speaker before, theirs to this one (the cues change places)"},
		{id: "structure.raise", label: "Line to Action", ex: "raise", icon: theme.MenuDropUpIcon, editing: true, when: screenplayOnly, run: func(w *MainWindow) { w.raiseLower(true) },
			tip: "The speech's line at the cursor (or the selected ones) out into action (as Promote on a line)"},
		{id: "structure.lower", label: "Action into the Speech", ex: "lower", icon: theme.MenuDropDownIcon, editing: true, when: screenplayOnly, run: func(w *MainWindow) { w.raiseLower(false) },
			tip: "The action at the cursor into the speech beside it, as its lines (as Demote on action)"},
		{id: "structure.expand", label: "Expand Selection", ex: "expand-selection", keys: alt(fyne.KeyO, false), icon: theme.ZoomInIcon, editing: true, when: screenplayOnly, run: (*MainWindow).expandSelection,
			tip: "Select the unit around the selection: the line, its speech, its scene, its section"},
		{id: "structure.shrink", label: "Shrink Selection", ex: "shrink-selection", keys: alt(fyne.KeyI, false), icon: theme.ZoomOutIcon, editing: true, when: screenplayOnly, run: (*MainWindow).shrinkSelection,
			tip: "Back to the selection before Expand Selection"},
		{id: "structure.swap", label: "Swap…", ex: "swap-ranges", icon: theme.ViewRefreshIcon, when: screenplayOnly, run: func(w *MainWindow) { w.showStructPicker(false, w.cardsAtLine()) },
			tip: "Swap scenes, runs of scenes or sections with others"},
		{id: "structure.move", label: "Move…", ex: "move-range", icon: theme.ContentPasteIcon, when: screenplayOnly, run: func(w *MainWindow) { w.showStructPicker(true, w.cardsAtLine()) },
			tip: "Move scenes, a run of scenes or a section before or after another"},
	}
}

// structureMenu is View > Structure.
func structureMenu() []home {
	return []home{
		at("structure.up"), at("structure.down"), at("structure.promote"), at("structure.demote"),
		sep(), at("structure.intoPrev"), at("structure.intoNext"), at("structure.barf"), at("structure.slurp"),
		sep(), at("structure.transpose"), at("structure.raise"), at("structure.lower"),
		sep(), at("structure.expand"), at("structure.shrink"),
		sep(), at("structure.swap"), at("structure.move"),
	}
}

// raiseLower is Promote on a speech's line (raise), Demote on action
// (lower), saying so where the cursor is on something else.
func (w *MainWindow) raiseLower(raise bool) {
	e := w.structEditor()
	if e == nil {
		return
	}
	tr, _ := w.structTree(e.Text())
	a, _, _, _ := selLines(e)
	speech, ok := tr.InSpeech(a)
	switch {
	case raise && !(ok && speech):
		w.setStatus("Put the cursor on a line of a speech to make it action")
	case !raise && (!ok || speech):
		w.setStatus("Put the cursor in action to make it a speech's lines")
	case raise:
		w.structDo(structure.Promote)
	default:
		w.structDo(structure.Demote)
	}
}
