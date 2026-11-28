// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"regexp"
	"strings"
	"time"

	"github.com/LaPingvino/accolade/internal/collab"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"
)

// Production, step 1 (docs/PRODUCTION.md): locked scene numbers.
//
// Production > Lock Scene Numbers writes a number (#12#) into every scene
// heading that has none and marks the script locked, in a boneyard block
// at its end, beside the adaptation's, so the lock travels with the file
// and other Fountain apps ignore it:
//
//	/* Production
//	scene numbers: locked, 10A
//	*/
//
// "10A" or "A11" is the lettering the script was locked with
// (lex.NumberStyle; Preferences > Script > Production gives a new lock
// its style). The block's other lines are kept as they are: later steps
// put more there. While locked, a scene written between two numbered ones
// gets its letter (10A, A11) once the cursor has left its heading and
// typing pauses, as part of the writer's last undo step (the learned
// notes' way, buffer.Amend). Unlock asks whether to keep the numbers or
// take out those the lock wrote, and Renumber numbers the scenes again
// (renumber.go); a new scene gets none until the next lock.
//
// Omit Scene makes the cursor's scene's heading OMITTED with its number
// and puts the scene in a boneyard under it (fountain.OmitScene), so
// Restore Scene brings it back whole; each is one undo step.

var productionMarks = endMarks(fountain.EndProduction)

// productionBlock is what is inside the script's production block.
func productionBlock(text string) (string, bool) {
	_, _, p, _ := allTails(text)
	return p.inside, p.ok
}

// sceneLock is the script's "scene numbers:" line.
type sceneLock struct {
	locked bool
	style  lex.NumberStyle
}

// sceneLockKey is the key of the lock's line.
const sceneLockKey = "scene numbers"

// sceneLockOf reads the lock from the production block: "scene numbers:
// locked", with the style after a comma or in brackets, any case.
func sceneLockOf(text string) sceneLock {
	return sceneLockIn(productionBlock(text))
}

// sceneLockIn is sceneLockOf for the production block's inside (ok:
// there is one).
func sceneLockIn(inside string, ok bool) sceneLock {
	if !ok {
		return sceneLock{}
	}
	for _, line := range strings.Split(inside, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), sceneLockKey) {
			continue
		}
		v := strings.ToLower(value)
		if !strings.Contains(v, "locked") || strings.Contains(v, "unlocked") {
			return sceneLock{}
		}
		l := sceneLock{locked: true}
		if strings.Contains(v, "a11") || strings.Contains(v, "a1 ") || strings.Contains(v, "before") {
			l.style = lex.NumberBefore
		}
		return l
	}
	return sceneLock{}
}

// String is the lock's line.
func (l sceneLock) String() string {
	return sceneLockKey + ": locked, " + l.style.String()
}

// withSceneLock is the block's inside with the lock's line set (nil:
// taken out), the other lines kept.
func withSceneLock(inside string, l *sceneLock) string {
	var lines []string
	set := false
	if inside != "" {
		for _, line := range strings.Split(inside, "\n") {
			if key, _, ok := strings.Cut(line, ":"); ok && strings.EqualFold(strings.TrimSpace(key), sceneLockKey) {
				if l != nil && !set {
					lines = append(lines, l.String())
					set = true
				}
				continue
			}
			lines = append(lines, line)
		}
	}
	if l != nil && !set {
		lines = append([]string{l.String()}, lines...)
	}
	return strings.Trim(strings.Join(lines, "\n"), "\n")
}

// withProductionBlock is the edit that makes the script's production
// block hold inside ("" takes it out): where (runes) and what. A new
// block goes after the script, before the other blocks at its end.
func withProductionBlock(text, inside string) (start, end int, block string) {
	_, _, p, _ := allTails(text)
	return withTailBlock(text, p, productionMarks, inside)
}

// sceneHead is a scene heading of the script: its line, the line as
// written, and its number ("" none); stray, a number left inside the
// heading by text typed after it ("INT. HOUSE #1A# - DAY"), which Fountain
// does not read as its number, or a number in its words other than its
// #n# (written anew, the two equal: scene_numbers.go); inWords, its number
// is in its words ("场景12 厨房 日 内": a pattern's {n}), Fountain's #12# or
// not.
type sceneHead struct {
	line    int
	text    string
	num     string
	stray   bool
	inWords bool
}

// bare is the heading without the number in its words (#12# kept): what
// it is called whatever its number (bdHeadingKey).
func (h sceneHead) bare(starts []string) string {
	if !h.inWords {
		return h.text
	}
	return withoutWordsNumber(h.text, starts)
}

// withoutWordsNumber is a heading line without the number in its words,
// a #12# at its end left as it is (lex.SceneNumber takes that).
func withoutWordsNumber(line string, starts []string) string {
	d := wordsDialect(starts)
	if d == nil {
		return line
	}
	_, hash := lex.SceneNumber(line)
	if out, ok := d.WithHeadingNumber(line, ""); ok {
		return lex.WithSceneNumber(out, hash)
	}
	return line
}

// strayNumber is a scene number inside a heading, not at its end.
var strayNumber = regexp.MustCompile(`\s#([^#\s]+)#\s`)

// sceneHeads are the script's scene headings, in order: not those in a
// boneyard (an omitted scene's old heading, the blocks at the end).
func sceneHeads(text string, starts []string) []sceneHead {
	lines := strings.Split(text, "\n")
	var out []sceneHead
	d := wordsDialect(starts)
	for _, it := range outlineOf(text, starts) {
		if it.Scene {
			h := sceneHead{line: it.Line, text: lines[it.Line]}
			_, h.num = lex.SceneNumber(strings.TrimSpace(h.text))
			if d != nil {
				if n, ok := d.HeadingNumber(h.text); ok {
					h.inWords = true
					switch {
					case h.num == "":
						h.num = n
					case n != h.num: // Fountain's is the one: the lock's, the breakdown's
						h.stray = true
					}
				}
			}
			if m := strayNumber.FindStringSubmatchIndex(h.text); h.num == "" && m != nil {
				// moved to the end: a locked number is not given twice
				h.num, h.stray = h.text[m[2]:m[3]], true
				h.text = h.text[:m[0]] + " " + h.text[m[1]:]
			}
			out = append(out, h)
		}
	}
	return out
}

// sceneNumbersFor are the headings' numbers with the missing ones filled in as
// a locked script (or, unlocked, by counting) does.
func sceneNumbersFor(heads []sceneHead, o lex.NumberOptions) []string {
	return lex.NumberLabels(sceneLabelsOf(heads), o)
}

// lineEdit is a line written anew: its range in runes and its text.
type lineEdit struct {
	start, end int
	text       string
}

// numberEdits are the edits that give the headings without a number
// theirs (skip: a line left as it is, the one being typed; -1 none).
func numberEdits(b *buffer.Buffer, heads []sceneHead, nums []string, skip int, nw numberWriter) []lineEdit {
	var out []lineEdit
	for i, h := range heads {
		if (h.num != "" && !h.stray) || h.line == skip {
			continue
		}
		start := b.LineStart(h.line)
		end := b.LineEnd(h.line)
		out = append(out, lineEdit{start, end, nw.write(h, nums[i])})
	}
	return out
}

// applyBackwards makes the edits (in order through the text) from the
// last one, so that the earlier ones' places hold, with replace.
func applyBackwards(edits []lineEdit, replace func(start, end int, text string)) {
	for i := len(edits) - 1; i >= 0; i-- {
		replace(edits[i].start, edits[i].end, edits[i].text)
	}
}

// sceneNumberStyle is the lettering a new lock gets (Preferences).
func (w *MainWindow) sceneNumberStyle() lex.NumberStyle {
	if w.settings == nil {
		return lex.NumberAfter
	}
	return lex.ParseNumberStyle(w.settings.GetString(sceneNumberStylePref))
}

// sceneNumberStylePref is the preference's key: "10A" or "A11".
const sceneNumberStylePref = "scene-number-style"

// lockSceneNumbers numbers every scene without a number and locks them:
// one undo step, the cursor kept where it was.
func (w *MainWindow) lockSceneNumbers() {
	if w.markdown || w.textEditor == nil {
		return
	}
	e := w.textEditor
	text := e.Text()
	if l := sceneLockOf(text); l.locked {
		w.setStatus("The scene numbers are locked already (" + l.style.String() + ")")
		return
	}
	heads := sceneHeads(text, w.sceneStarts())
	if len(heads) == 0 {
		w.setStatus("No scenes to number: lock the numbers once the script has scene headings")
		return
	}
	w.followSceneNumbers() // the extras in line with the text before
	lock := sceneLock{locked: true, style: w.sceneNumberStyle()}
	nums := sceneNumbersFor(heads, lex.NumberOptions{Locked: true, Style: lock.style})
	inside, _ := productionBlock(text)
	// which numbers were there before it: Unlock's Remove leaves them (renumber.go)
	hand := handNumbersLine(heads)
	start, end, block := withProductionBlock(text, withBlockLine(withSceneLock(inside, &lock), handNumbersKey, hand))
	nw := w.numberWriterFor(text)
	e.Edit(func(b *buffer.Buffer) {
		line, col := b.Position(b.Cursor())
		edits := numberEdits(b, heads, nums, -1, nw)
		b.Group(func() {
			b.Replace(start, end, block) // after the script: the headings' places hold
			applyBackwards(edits, b.Replace)
			b.SetCursor(b.Offset(line, col), false)
		})
	})
	n := 0
	for _, h := range heads {
		if h.num == "" {
			n++
		}
	}
	w.setStatus("Scene numbers locked: " + plural(len(heads), "scene") + ", " + plural(n, "number") + " written. A new scene gets a letter (" + lock.style.String() + ")")
	w.followSceneNumbers()
}

// sceneNumberPause is how long typing pauses before a new scene in a
// locked script is numbered.
const sceneNumberPause = 700 * time.Millisecond

// scheduleSceneNumbers numbers a locked script's new scenes, or keeps a
// script's numbers in its headings in step (scene_numbers.go), once
// typing pauses (onTextChanged, onCursorChanged): never per key.
func (w *MainWindow) scheduleSceneNumbers() {
	if w.markdown || w.textEditor == nil {
		return
	}
	inside, ok := productionBlock(w.textEditor.Text())
	if !sceneLockIn(inside, ok).locked && !w.mayNumberInHeadings(inside, ok) {
		return
	}
	w.debounce(&w.sceneNumberTimer, sceneNumberPause, w.numberNewScenes)
}

// numberNewScenes gives a locked script's scenes without a number theirs,
// all but the heading the cursor is on (being written), as part of the
// writer's last undo step. Not while Redo is possible or text is
// selected, and not before the writer has edited (a script opened as it
// is, written elsewhere, is numbered at the next edit or by Lock).
// Shared, a peer numbers only the headings it wrote (wroteLine): every
// peer sees a new scene, and the one writing it is the one who numbers
// it, never in a co-writer's half-typed heading.
//
// Two peers adding a scene at one place at the same moment, before
// either hears of the other's, each number theirs as the only new one:
// the same number twice. Of two headings with one number, the later in
// the text is numbered again, by its writer (the same answer on every
// peer, so only one of them does it).
func (w *MainWindow) numberNewScenes() {
	if w.markdown || w.textEditor == nil {
		return
	}
	e := w.textEditor
	b := e.Buffer()
	if b.HasSelection() {
		return
	}
	text := e.Text()
	lock := sceneLockOf(text)
	line, _ := b.Position(b.Cursor())
	fresh := !b.CanRedo() && b.CanUndo() // the writer has edited: new scenes are theirs to number
	if !lock.locked {
		// unlocked, the numbers in the headings kept in step; shared, by
		// the moves and New Scene of each peer's own (scene_numbers.go)
		if fresh && w.shared == nil {
			if n := w.keepInStep(line); n > 0 {
				w.setStatus(renumberedNote(n))
			}
		}
		return
	}
	heads := sceneHeads(text, w.sceneStarts())
	var mine []int           // the headings to number: this writer's (wroteLine)
	seen := map[string]int{} // a number's first heading's line
	for i, h := range heads {
		first, had := seen[h.num]
		twice := h.num != "" && had && w.numberedApart(first, h.line)
		if !had {
			seen[h.num] = h.line
		}
		switch {
		case h.line == line || !(twice || fresh && (h.num == "" || h.stray)):
		case w.wroteLine(h.line):
			mine = append(mine, i)
			if twice {
				heads[i].num, heads[i].stray = "", false // a new one, numbered as such
			}
		}
	}
	if len(mine) == 0 {
		return
	}
	nums := sceneNumbersFor(heads, lex.NumberOptions{Locked: true, Style: lock.style})
	nw := w.numberWriterFor(text)
	var edits []lineEdit
	for _, i := range mine {
		h := heads[i]
		edits = append(edits, lineEdit{b.LineStart(h.line), b.LineEnd(h.line), nw.write(h, nums[i])})
	}
	e.Edit(func(b *buffer.Buffer) { applyBackwards(edits, b.Amend) })
}

// numberedApart reports whether the headings on lines a and b were
// written by two peers of a shared script (their first runes' writers),
// each numbering theirs before hearing of the other's. A number written
// twice by one writer, or in a script not shared, is the writer's to
// mend (Renumber), not changed by itself.
func (w *MainWindow) numberedApart(a, b int) bool {
	if w.shared == nil {
		return false
	}
	buf := w.textEditor.Buffer()
	apart := false
	w.shared.session.Do(func(d *collab.Doc) {
		if d.Len() == buf.Len() {
			x, y := d.Writer(buf.LineStart(a)), d.Writer(buf.LineStart(b))
			apart = x != "" && y != "" && x != y
		}
	})
	return apart
}

// sceneAtCursor is the scene the cursor is in, in runes: from its
// heading's line to the next heading or section (the blank lines before
// it included), the last scene to the script's end before the blocks
// there; false on the title page, in a section before any scene, or in
// the blocks at the end.
func (w *MainWindow) sceneAtCursor() (start, end int, ok bool) {
	e := w.textEditor
	b := e.Buffer()
	text := e.Text()
	body := bodyEnd(text)
	if e.CursorOffset() > body {
		return 0, 0, false
	}
	line, _ := b.Position(e.CursorOffset())
	from, to := -1, -1
	for _, it := range outlineOf(text, w.sceneStarts()) {
		if b.LineStart(it.Line) >= body {
			break
		}
		if it.Line <= line {
			from = -1
			if it.Scene {
				from = it.Line
			}
			continue
		}
		to = it.Line
		break
	}
	if from < 0 {
		return 0, 0, false
	}
	start, end = b.LineStart(from), body
	if to >= 0 {
		end = b.LineStart(to)
	}
	return start, end, true
}

// omitScene makes the cursor's scene OMITTED, keeping its number, the
// scene in a boneyard under it: one undo step.
func (w *MainWindow) omitScene() {
	if w.scriptAway("Omit Scene") { // another middle tab in front (middle.go)
		return
	}
	if w.markdown || w.textEditor == nil {
		return
	}
	start, end, ok := w.sceneAtCursor()
	if !ok {
		w.setStatus("Put the cursor in the scene to omit")
		return
	}
	scene := string([]rune(w.textEditor.Text())[start:end])
	heading, _, _ := strings.Cut(scene, "\n")
	if _, omitted := lex.OmittedNumber(heading); omitted {
		w.setStatus("This scene is omitted already: Restore Scene brings it back")
		return
	}
	out, ok := fountain.OmitScene(scene)
	if !ok {
		w.setStatus("An omitted scene keeps its number: lock the scene numbers first (Production > Lock Scene Numbers)")
		return
	}
	w.textEditor.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			b.Replace(start, end, out)
			b.SetCursor(start, false)
		})
	})
	_, n := lex.SceneNumber(heading)
	w.setStatus("Scene " + n + " omitted: its number is kept, the scene below it in boneyard (Restore Scene brings it back)")
}

// restoreScene brings the cursor's omitted scene back as it was: one
// undo step.
func (w *MainWindow) restoreScene() {
	if w.scriptAway("Restore Scene") { // another middle tab in front (middle.go)
		return
	}
	if w.markdown || w.textEditor == nil {
		return
	}
	start, end, ok := w.sceneAtCursor()
	if !ok {
		w.setStatus("Put the cursor on the OMITTED heading to restore")
		return
	}
	scene := string([]rune(w.textEditor.Text())[start:end])
	heading, _, _ := strings.Cut(scene, "\n")
	n, omitted := lex.OmittedNumber(heading)
	if !omitted {
		w.setStatus("This scene is not omitted")
		return
	}
	out, ok := fountain.RestoreScene(scene)
	if !ok {
		w.setStatus("Scene " + n + "'s text is not under its heading (no /* Omitted " + n + " … */ block): write it again over OMITTED")
		return
	}
	w.textEditor.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			b.Replace(start, end, out)
			b.SetCursor(start, false)
		})
	})
	w.setStatus("Scene " + n + " restored")
}
