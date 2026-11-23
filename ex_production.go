package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/structure"
)

// The ex commands for production and structure (LOOP.md, the keyboard
// pass), each the menus' own command with its choices typed:
//
//	:renumber [keep 5,12A] [only …] [from X to Y] [fold|letters] [omitted drop|keep|runs]
//	:lock   :unlock [keep|remove]   :omit [X..Y]   :restore [X..Y]
//	:move X..Y after|before Z   :'<,'>move after Z   :swap X..Y with Z..W
//	:issue   :tag <category>   :draft <name>
//
// Scenes are named by their labels, in the script's order: a scene's
// number as written (5, 12A, I-1-A), or in a script without numbers its
// place (1, 2, 3…; (3) in a numbered script names the third scene
// whatever its label); #Name a section by the start of its heading;
// . the cursor's scene, $ the last. A range X..Y runs from X to Y in
// the script's order (Y before X is an error, never swapped).

// exRefs are the script's scenes and sections as the commands name them.
type exRefs struct {
	items    []outlineItem // the body's headings, in order
	heads    []sceneHead   // its scenes
	numbered bool          // some scene has a number
	cursor   int           // the cursor's line
	lines    []string
}

// scriptRefs reads the script for the commands.
func (w *MainWindow) scriptRefs() (exRefs, error) {
	if w.markdown || w.textEditor == nil {
		return exRefs{}, errors.New("Scenes are a screenplay's")
	}
	if !w.scriptInMain() {
		return exRefs{}, errors.New("The script's tab is not in front")
	}
	text := w.textEditor.Text()
	body := text[:bodyEndByte(text)]
	r := exRefs{heads: sceneHeads(body, w.sceneStarts()), lines: strings.Split(text, "\n")}
	r.items = outlineOf(body, w.sceneStarts())
	for _, h := range r.heads {
		r.numbered = r.numbered || h.num != ""
	}
	r.cursor, _ = w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
	return r, nil
}

// label is scene i's name: its number, else (numbered script) its
// place in brackets, else its place.
func (r exRefs) label(i int) string {
	switch {
	case r.heads[i].num != "":
		return r.heads[i].num
	case r.numbered:
		return "(" + strconv.Itoa(i+1) + ")"
	}
	return strconv.Itoa(i + 1)
}

// scene is the index of the scene ref names (-1 and an error if none).
func (r exRefs) scene(ref string) (int, error) {
	ref = strings.TrimSpace(ref)
	switch ref {
	case "":
		return -1, errors.New("a scene is missing")
	case ".":
		at := -1
		for i, h := range r.heads {
			if h.line <= r.cursor {
				at = i
			}
		}
		if at < 0 {
			return -1, errors.New("The cursor is in no scene")
		}
		return at, nil
	case "$":
		if len(r.heads) == 0 {
			return -1, errors.New("The script has no scenes")
		}
		return len(r.heads) - 1, nil
	}
	for i := range r.heads {
		if strings.EqualFold(r.label(i), ref) {
			return i, nil
		}
	}
	if n, err := strconv.Atoi(strings.Trim(ref, "()")); err == nil && strings.HasPrefix(ref, "(") && n >= 1 && n <= len(r.heads) {
		return n - 1, nil
	}
	return -1, fmt.Errorf("No scene %s", ref)
}

// line is the heading line ref names: a scene's, or #Name a section's.
func (r exRefs) line(ref string) (int, error) {
	ref = strings.TrimSpace(ref)
	if name, ok := strings.CutPrefix(ref, "#"); ok {
		name = strings.ToLower(strings.TrimSpace(name))
		for _, it := range r.items {
			if !it.Scene && strings.HasPrefix(strings.ToLower(it.Text), name) {
				return it.Line, nil
			}
		}
		return -1, fmt.Errorf("No section %s", ref)
	}
	i, err := r.scene(ref)
	if err != nil {
		return -1, err
	}
	return r.heads[i].line, nil
}

// splitRange is X..Y as X and Y (X alone: X twice).
func splitRange(s string) (string, string) {
	for _, sep := range []string{"..", "…", "–"} {
		if a, b, ok := strings.Cut(s, sep); ok {
			return strings.TrimSpace(a), strings.TrimSpace(b)
		}
	}
	return strings.TrimSpace(s), strings.TrimSpace(s)
}

// lines are the heading lines of X..Y, the first before the last.
func (r exRefs) rangeLines(s string) (int, int, error) {
	a, b := splitRange(s)
	x, err := r.line(a)
	if err != nil {
		return 0, 0, err
	}
	y, err := r.line(b)
	if err != nil {
		return 0, 0, err
	}
	if y < x {
		return 0, 0, fmt.Errorf("%s comes before %s in the script: a range runs in the script's order", b, a)
	}
	return x, y, nil
}

// asProduction is a production command given by name: as a writer the
// script works as Production first (as the palette does, role.go).
func (w *MainWindow) asProduction(what string) {
	if w.writerView() {
		w.setOwnRole(roleProduction)
		w.setStatus("Working as Production for this script (" + what + " is production's): View > Working As goes back")
	}
}

// exRenumber is :renumber: with no choices the dialog, else the choices
// typed, applied as the dialog's Apply does.
func (w *MainWindow) exRenumber(arg string) error {
	r, err := w.scriptRefs()
	if err != nil {
		return err
	}
	w.asProduction("Renumber")
	if strings.TrimSpace(arg) == "" {
		w.renumberScenes()
		return nil
	}
	if !r.numbered {
		return errors.New("No scene numbers to renumber: :lock numbers the scenes")
	}
	c, err := parseRenumber(arg, r)
	if err != nil {
		return err
	}
	text := w.textEditor.Text()
	heads := sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())
	w.rememberChoice(c)
	msg := w.renumberWith(c, w.renumberContextNow(text, heads))
	if strings.HasPrefix(msg, "Not renumbered") {
		return errors.New(msg)
	}
	return nil
}

// renumberWords are :renumber's words.
var renumberWords = []string{"keep", "only", "from", "to", "fold", "letters", "omitted"}

// parseRenumber reads :renumber's choices.
func parseRenumber(arg string, r exRefs) (renumberChoice, error) {
	var c renumberChoice
	words := strings.Fields(arg)
	from, to := "", ""
	for i := 0; i < len(words); i++ {
		word := strings.ToLower(words[i])
		next := func() (string, error) {
			if i+1 >= len(words) {
				return "", fmt.Errorf(":renumber %s wants what follows it", word)
			}
			i++
			return words[i], nil
		}
		switch word {
		case "keep", "only":
			if c.List != listNone {
				return c, errors.New(":renumber takes keep or only, once")
			}
			c.List = listKeep
			if word == "only" {
				c.List = listOnly
			}
			var list []string
			for i+1 < len(words) && !isRenumberWord(words[i+1]) {
				i++
				list = append(list, words[i])
			}
			if len(list) == 0 {
				return c, fmt.Errorf(":renumber %s wants numbers: %s 5, 12A, 20-24", word, word)
			}
			c.Numbers = strings.Join(list, " ")
		case "from":
			s, err := next()
			if err != nil {
				return c, err
			}
			from = s
		case "to":
			s, err := next()
			if err != nil {
				return c, err
			}
			to = s
		case "fold":
			c.KeepLetters = false
		case "letters":
			c.KeepLetters = true
		case "omitted":
			s, err := next()
			if err != nil {
				return c, err
			}
			switch strings.ToLower(s) {
			case "drop":
				c.Omitted = omitDrop
			case "keep":
				c.Omitted = omitKeep
			case "runs":
				c.Omitted = omitPrintRuns
			default:
				return c, fmt.Errorf(":renumber omitted takes drop, keep or runs, not %s", s)
			}
		default:
			return c, fmt.Errorf(":renumber does not know %q (Tab lists its words)", words[i])
		}
	}
	if from != "" || to != "" {
		if from == "" {
			from = "(1)"
		}
		if to == "" {
			to = "$"
		}
		a, err := r.scene(from)
		if err != nil {
			return c, err
		}
		b, err := r.scene(to)
		if err != nil {
			return c, err
		}
		c.Scope, c.From, c.To = scopeRange, a, b // To before From: problems says so
	}
	return c, nil
}

func isRenumberWord(s string) bool {
	for _, w := range renumberWords {
		if strings.EqualFold(w, s) {
			return true
		}
	}
	return false
}

// exLock is :lock.
func (w *MainWindow) exLock(string) error {
	if _, err := w.scriptRefs(); err != nil {
		return err
	}
	w.asProduction("Lock Scene Numbers")
	w.lockSceneNumbers()
	return nil
}

// exUnlock is :unlock: keep the numbers, remove them, or (neither) the
// dialog that asks.
func (w *MainWindow) exUnlock(arg string) error {
	if _, err := w.scriptRefs(); err != nil {
		return err
	}
	if !sceneLockOf(w.textEditor.Text()).locked {
		return errors.New("The scene numbers are not locked")
	}
	w.asProduction("Unlock Scene Numbers")
	switch strings.ToLower(strings.TrimSpace(arg)) {
	case "":
		w.unlockSceneNumbers()
	case "keep":
		w.unlockNow(false)
	case "remove":
		w.unlockNow(true)
	default:
		return fmt.Errorf(":unlock takes keep or remove, not %s", arg)
	}
	return nil
}

// exOmit is :omit and :restore (restore): the scenes of the range, the
// cursor's without one, as one undo step.
func exOmit(restore bool) func(w *MainWindow, arg string) error {
	return func(w *MainWindow, arg string) error {
		r, err := w.scriptRefs()
		if err != nil {
			return err
		}
		what := "Omit Scene"
		if restore {
			what = "Restore Scene"
		}
		w.asProduction(what)
		if strings.TrimSpace(arg) == "" {
			arg = "."
		}
		x, y, err := r.rangeLines(arg)
		if err != nil {
			return err
		}
		e := w.textEditor
		text := e.Text()
		tr, _ := w.structTree(text)
		if _, last, ok := tr.Span(y, structure.Section); ok && !isSceneLine(r, y) {
			y = last // a section: all its scenes
		} else if _, last, ok := tr.Span(y, structure.Scene); ok {
			y = last
		}
		b := buffer.New(text)
		body := bodyEnd(text)
		type span struct{ start, end int }
		var spans []span
		var nums []string
		for i, it := range r.items {
			if !it.Scene || it.Line < x || it.Line > y {
				continue
			}
			end := body
			if i+1 < len(r.items) {
				end = b.LineStart(r.items[i+1].Line)
			}
			spans = append(spans, span{b.LineStart(it.Line), end})
		}
		if len(spans) == 0 {
			return errors.New("No scene there")
		}
		runes := []rune(text)
		skipped := 0
		for k := len(spans) - 1; k >= 0; k-- { // from the end: the earlier offsets hold
			s := spans[k]
			scene := string(runes[s.start:s.end])
			heading, _, _ := strings.Cut(scene, "\n")
			_, omitted := lex.OmittedNumber(heading)
			var out string
			var ok bool
			switch {
			case restore && omitted:
				out, ok = fountain.RestoreScene(scene)
			case !restore && !omitted:
				out, ok = fountain.OmitScene(scene)
				if !ok {
					return errors.New("An omitted scene keeps its number: lock the scene numbers first (:lock)")
				}
			}
			if !ok {
				skipped++
				continue
			}
			_, n := lex.SceneNumber(heading)
			if restore {
				n, _ = lex.OmittedNumber(heading)
			}
			nums = append([]string{n}, nums...)
			runes = append(append(append([]rune(nil), runes[:s.start]...), []rune(out)...), runes[s.end:]...)
		}
		if len(nums) == 0 {
			if restore {
				return errors.New("No omitted scene there to restore")
			}
			return errors.New("These scenes are omitted already")
		}
		replaceTextIn(e, text, string(runes))
		verb := "omitted"
		if restore {
			verb = "restored"
		}
		msg := plural(len(nums), "scene") + " " + verb + ": " + strings.Join(nums, ", ")
		if skipped > 0 {
			msg += fmt.Sprintf(" (%d left as they were)", skipped)
		}
		w.setStatus(msg + " (one undo step)")
		return nil
	}
}

// isSceneLine reports whether line is a scene's heading.
func isSceneLine(r exRefs, line int) bool {
	for _, h := range r.heads {
		if h.line == line {
			return true
		}
	}
	return false
}

// exStructural reports whether a :move's argument moves scenes or
// sections (after or before a unit), not lines (an address).
func exStructural(rest string) bool {
	for _, f := range strings.Fields(strings.ToLower(rest)) {
		if f == "after" || f == "before" {
			return true
		}
	}
	return false
}

// exMoveUnits is :move X..Y after|before Z, or with a range of lines
// ('<,'>move after Z) the units they are in: through the structure
// engine, one undo step.
func (w *MainWindow) exMoveUnits(arg string, from, to int, given bool) error {
	r, err := w.scriptRefs()
	if err != nil {
		return err
	}
	words := strings.Fields(arg)
	at := -1
	for i, f := range words {
		if l := strings.ToLower(f); l == "after" || l == "before" {
			at = i
		}
	}
	if at < 0 || at == len(words)-1 {
		return errors.New(":move wants where: :move 3..5 after 9")
	}
	after := strings.EqualFold(words[at], "after")
	text := w.textEditor.Text()
	tr, read := w.structTree(text)
	var x, y int
	if src := strings.Join(words[:at], " "); src != "" {
		if x, y, err = r.rangeLines(src); err != nil {
			return err
		}
	} else if given {
		x, y = tr.At(from), tr.At(to)
		if x < 0 || y < 0 {
			return errors.New("The lines are in no scene or section")
		}
	} else {
		x, y = tr.At(r.cursor), tr.At(r.cursor)
		if x < 0 {
			return errors.New("The cursor is in no scene or section")
		}
	}
	z, err := r.line(strings.Join(words[at+1:], " "))
	if err != nil {
		return err
	}
	res, err := tr.Move(structure.Range{First: x, Last: y}, z, after, read)
	if err != nil {
		return err
	}
	_, _, sel, col := selLines(w.textEditor)
	w.applyStruct(text, res, sel, r.cursor, col)
	return nil
}

// exSwap is :swap X..Y with Z..W.
func (w *MainWindow) exSwap(arg string) error {
	r, err := w.scriptRefs()
	if err != nil {
		return err
	}
	a, b, ok := cutWord(arg, "with")
	if !ok {
		return errors.New(":swap wants two: :swap 3..4 with 8")
	}
	x0, x1, err := r.rangeLines(a)
	if err != nil {
		return err
	}
	y0, y1, err := r.rangeLines(b)
	if err != nil {
		return err
	}
	text := w.textEditor.Text()
	tr, read := w.structTree(text)
	res, err := tr.Swap(structure.Range{First: x0, Last: x1}, structure.Range{First: y0, Last: y1}, read)
	if err != nil {
		return err
	}
	_, _, sel, col := selLines(w.textEditor)
	w.applyStruct(text, res, sel, r.cursor, col)
	return nil
}

// cutWord cuts s at the word sep (any case).
func cutWord(s, sep string) (string, string, bool) {
	words := strings.Fields(s)
	for i, f := range words {
		if strings.EqualFold(f, sep) {
			return strings.Join(words[:i], " "), strings.Join(words[i+1:], " "), true
		}
	}
	return s, "", false
}

// exIssue is :issue: the Issue Draft dialog.
func (w *MainWindow) exIssue(string) error {
	if _, err := w.scriptRefs(); err != nil {
		return err
	}
	w.asProduction("Issue Draft")
	w.showIssueDraft()
	return nil
}

// tagCategoryNames are the categories one tags in, for :tag.
func (w *MainWindow) tagCategoryNames() []string {
	if w.markdown || w.textEditor == nil {
		return nil
	}
	f, _, _ := w.breakdownNowFound()
	var out []string
	for _, c := range f.categories() {
		out = append(out, c.Name)
	}
	return out
}

// exTag is :tag <category>: the selection (or the word at the cursor)
// tagged in the category its name begins.
func (w *MainWindow) exTag(arg string) error {
	if _, err := w.scriptRefs(); err != nil {
		return err
	}
	var choices []exChoice
	for _, n := range w.tagCategoryNames() {
		if strings.EqualFold(n, arg) {
			choices = []exChoice{{text: n}}
			break
		}
		choices = append(choices, exChoice{text: n, label: n})
	}
	m := exFilter(choices, arg)
	if len(m) == 0 {
		return fmt.Errorf("No category %q (Tab lists them)", arg)
	}
	w.asProduction("Tag Selection")
	w.tagSelectionAs(m[0].text)
	return nil
}

// exDraft is :draft <name>: a draft saved now, by that name.
func (w *MainWindow) exDraft(arg string) error {
	name := strings.TrimSpace(arg)
	w.afterBundleWrites(func() {
		w.addDraft(name, false, func(d savedDraft, _ bool, err error) {
			if err != nil {
				w.setStatus("Could not save the draft: " + err.Error())
				return
			}
			w.setStatus("Draft “" + d.Name + "” saved")
		})
	})
	return nil
}

// exRefChoices are the scenes and sections to complete a reference with.
func (w *MainWindow) exRefChoices() []exChoice {
	r, err := w.scriptRefs()
	if err != nil {
		return nil
	}
	var out []exChoice
	scene := 0
	for _, it := range r.items {
		if it.Scene {
			l := r.label(scene)
			out = append(out, exChoice{text: l, label: l, desc: it.Text})
			scene++
			continue
		}
		name := "#" + strings.ReplaceAll(it.Text, " ", "")
		if f := strings.Fields(it.Text); len(f) > 0 {
			name = "#" + f[0]
		}
		out = append(out, exChoice{text: name, label: name, desc: strings.Repeat("#", max(it.Level, 1)) + " " + it.Text})
	}
	return out
}
