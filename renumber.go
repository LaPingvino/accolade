package main

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/lexington/lex"
)

// Taking a lock back, and numbering again (docs/PRODUCTION.md §15).
//
// Lock Scene Numbers notes which headings had a number before it, on a
// line of its own in the production block:
//
//	/* Production
//	scene numbers: locked, 10A
//	numbered before the lock: 3, 7A
//	*/
//
// ("none" when no heading had one), so Unlock Scene Numbers… can offer to
// take out exactly what the lock wrote: the numbers it gave and the
// letters new scenes got while locked; the ones the writer typed stay. A
// lock made before this line existed has no record: then Remove takes
// every number out, and the dialog says so.
//
// Production > Renumber Scenes… asks how (renumber_dialog.go,
// renumber_strategy.go; Lexington's lex.Renumber does the numbering):
// which scenes, which numbers stay or only which go, A-scenes folded or
// kept, OMITTED dropped or kept (a run printed as one "12-14 OMITTED" or
// not: omitted_runs.go), with
// presets (Clean 1…n; after an issue, keep sent scenes; last used) and a
// preview of old → new made by the plan Apply writes. Clean 1…n numbers
// the scenes 1…n again, in script order: A-numbers and gaps go. An
// OMITTED scene loses its number too: OMITTED keeps a number that went
// out with the script from being given again (Wikipedia, Shooting
// script), and renumbering is what closes those gaps (StudioBinder, Scene
// numbering). Its text stays where it was, as boneyard (/* Omitted scene
// … */), as Remove leaves an OMITTED scene whose number goes. After an
// issue the dialog warns that what went out uses the old numbers, and
// offers to save a draft first. Each is one undo step, the page lock's
// scene numbers following in the same step; the breakdown and the
// schedule, kept beside the script, follow by the scenes' places
// (renumber_follow.go).

// handNumbersKey is the key of the line naming the numbers there before
// the lock.
const handNumbersKey = "numbered before the lock"

// handNumbersOf are the numbers the script had before its lock; known is
// false for a lock without the line (made before it existed).
func handNumbersOf(text string) (nums map[string]bool, known bool) {
	inside, ok := productionBlock(text)
	if !ok {
		return nil, false
	}
	for _, line := range strings.Split(inside, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), handNumbersKey) {
			continue
		}
		nums = map[string]bool{}
		for _, n := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' }) {
			if !strings.EqualFold(n, "none") {
				nums[n] = true
			}
		}
		return nums, true
	}
	return nil, false
}

// withoutBlockLine is the block's inside without the line of key.
func withoutBlockLine(inside, key string) string {
	var lines []string
	for _, line := range strings.Split(inside, "\n") {
		if k, _, ok := strings.Cut(line, ":"); ok && strings.EqualFold(strings.TrimSpace(k), key) {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Trim(strings.Join(lines, "\n"), "\n")
}

// handNumbersLine is the line's value for the headings' numbers now.
func handNumbersLine(heads []sceneHead) string {
	var nums []string
	for _, h := range heads {
		if h.num != "" {
			if _, omitted := lex.OmittedNumber(h.text); !omitted {
				nums = append(nums, h.num)
			}
		}
	}
	if len(nums) == 0 {
		return "none"
	}
	return strings.Join(nums, ", ")
}

// numberPlan is what Remove or Renumber does to each heading: its new
// number ("" none), or, for an OMITTED one, drop: its number goes, and
// with it the heading; its scene stays as boneyard. free are the
// headings Renumber gave a new number (nil for Remove).
type numberPlan struct {
	nums []string
	drop []bool
	free []bool
}

// isOmitted tells whether a heading is an OMITTED one.
func isOmitted(h sceneHead) bool {
	_, omitted := lex.OmittedNumber(h.text)
	return omitted
}

// removePlan takes out every number but keep's, and the labels out of
// Lexington's scheme (I-1-A, PROLOGUE), which no lock writes.
func removePlan(heads []sceneHead, keep map[string]bool) numberPlan {
	p := numberPlan{nums: make([]string, len(heads)), drop: make([]bool, len(heads))}
	for i, h := range heads {
		if h.num != "" && (keep[h.num] || !lex.InNumberScheme(h.num)) {
			p.nums[i] = h.num
			continue
		}
		p.drop[i] = isOmitted(h)
	}
	return p
}

// omittedOpenLine is how an OMITTED scene's text is kept once its number
// is gone.
const omittedOpenLine = "/* Omitted scene"

// planEdits are the edits of a plan, in order through the text: a
// heading with another number written anew; an OMITTED heading dropped,
// its "/* Omitted 12" block becoming "/* Omitted scene", the old heading
// in it without its number (no OMITTED heading, no number: nothing to
// restore it as).
func planEdits(b *buffer.Buffer, lines []string, heads []sceneHead, p numberPlan) []lineEdit {
	var out []lineEdit
	for i, h := range heads {
		if p.drop[i] {
			out = append(out, dropOmitted(b, lines, h))
			continue
		}
		if p.nums[i] == h.num && !h.stray {
			continue
		}
		out = append(out, lineEdit{b.LineStart(h.line), b.LineEnd(h.line), lex.WithSceneNumber(h.text, p.nums[i])})
	}
	return out
}

// dropOmitted is the edit that takes an OMITTED heading out, its block
// kept as plain boneyard.
func dropOmitted(b *buffer.Buffer, lines []string, h sceneHead) lineEdit {
	// block finds the "/* Omitted …" block from line at (blank lines
	// skipped): its first and closing lines, or -1
	block := func(at int) (open, closing int) {
		for at < len(lines) && strings.TrimSpace(lines[at]) == "" {
			at++
		}
		first := ""
		if at < len(lines) {
			first = strings.TrimSpace(lines[at])
		}
		if !strings.HasPrefix(first, "/* Omitted") || first == omittedOpenLine {
			return at, -1
		}
		for k := at + 1; k < len(lines); k++ {
			if strings.TrimSpace(lines[k]) == "*/" {
				return at, k
			}
		}
		return at, -1
	}
	open, closing := block(h.line + 1)
	if closing < 0 { // no text under it: the heading goes, its line with it
		end := b.LineEnd(h.line)
		if h.line+1 < len(lines) {
			end = b.LineStart(h.line + 1)
		}
		return lineEdit{b.LineStart(h.line), end, ""}
	}
	var text strings.Builder
	inner := lines[open+1 : closing]
	kept := make([]string, len(inner))
	copy(kept, inner)
	if len(kept) > 0 { // the old heading, without its number
		_, n := lex.SceneNumber(strings.TrimSpace(kept[0]))
		if n != "" {
			kept[0] = lex.WithSceneNumber(kept[0], "")
		}
	}
	text.WriteString(omittedOpenLine + "\n" + strings.Join(kept, "\n"))
	if len(kept) > 0 {
		text.WriteString("\n")
	}
	text.WriteString("*/")
	return lineEdit{b.LineStart(h.line), b.LineEnd(closing), text.String()}
}

// pageLockFollowing is the block's inside with the page lock's scenes
// renamed: old number → new ("" for a scene whose number went).
func pageLockFollowing(text, inside string, heads []sceneHead, p numberPlan) string {
	l := pageLockOf(text)
	if l == nil {
		return inside
	}
	to := map[string]string{}
	for i, h := range heads {
		if h.num != "" {
			to[h.num] = p.nums[i]
		}
	}
	pages := append([]lex.LockedPage(nil), l.lock.Pages...)
	changed := false
	for i, pg := range pages {
		if n, ok := to[pg.Scene]; ok && pg.Scene != "" && n != pg.Scene {
			pages[i].Scene = n
			if n == "" {
				pages[i].Nth = 0 // its words anywhere after the page before
			}
			changed = true
		}
	}
	if !changed {
		return inside
	}
	lock := *l.lock
	lock.Pages = pages
	nl := *l
	nl.lock = &lock
	return withPageLock(inside, &nl)
}

// applyNumberPlan makes a plan and the block's new inside one undo step,
// the cursor kept on its line.
func (w *MainWindow) applyNumberPlan(heads []sceneHead, p numberPlan, inside string) {
	e := w.textEditor
	text := e.Text()
	lines := strings.Split(text, "\n")
	_, _, pb, _ := allTails(text)
	w.followNumbers(true) // the extras in line with the text before
	start, end, block := withProductionBlock(text, inside)
	if strings.TrimSpace(inside) == "" { // the block goes
		block = ""
		if end == utf8.RuneCountInString(text) {
			block = "\n"
		}
	}
	e.Edit(func(b *buffer.Buffer) {
		line, col := b.Position(b.Cursor())
		edits := planEdits(b, lines, heads, p)
		b.Group(func() {
			if pb.ok || strings.TrimSpace(inside) != "" {
				b.Replace(start, end, block) // after the script: the headings' places hold
			}
			applyBackwards(edits, b.Replace)
			b.SetCursor(b.Offset(line, col), false)
		})
	})
	w.followSceneNumbers()
}

// firstIssue tells whether the script has an issue: its numbers went
// out with it. The name and date of the first, for the dialogs.
func (w *MainWindow) firstIssue() (bool, string) {
	chain, err := w.revisionChainNow()
	if err != nil || chain == nil || len(chain.issues) == 0 {
		return false, ""
	}
	first := chain.issues[0].set
	what := first.Name
	if what == "" {
		what = "White"
	}
	if !first.Date.IsZero() {
		what += " (" + first.Date.Format("2 Jan 2006") + ")"
	}
	return true, what
}

// unlockSceneNumbers is Production > Unlock Scene Numbers…: Keep the
// numbers, or Remove the numbers the lock gave.
func (w *MainWindow) unlockSceneNumbers() {
	if w.markdown || w.textEditor == nil {
		return
	}
	if !sceneLockOf(w.textEditor.Text()).locked {
		w.setStatus("The scene numbers are not locked")
		return
	}
	w.splitOldOmittedRuns() // v1.13.0's OMITTED #12-14#: one heading a scene first
	w.afterBundleWrites(func() { w.unlockDialog().Show() })
}

// unlockDialog is the Unlock dialog (not shown).
func (w *MainWindow) unlockDialog() dialog.Dialog {
	text := w.textEditor.Text()
	heads := sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())
	keep, known := handNumbersOf(text)
	p := removePlan(heads, keep)
	gone, kept, omitted := 0, []string{}, 0
	for i, h := range heads {
		switch {
		case h.num == "":
		case p.nums[i] != "":
			kept = append(kept, h.num)
		default:
			gone++
			if p.drop[i] {
				omitted++
			}
		}
	}
	issued, what := w.firstIssue()

	var msg strings.Builder
	if issued {
		fmt.Fprintf(&msg, "The script was issued: %s. Its numbers have gone out with it, on sides, schedules and printed pages, so keeping them is safest.", what)
	} else {
		msg.WriteString("Nothing has been issued since the lock: the numbers have not gone out yet.")
	}
	msg.WriteString("\n\nKeep the numbers: every number stays as it is.\nRemove the numbers: ")
	switch {
	case !known:
		fmt.Fprintf(&msg, "every number goes (%s): this lock was made before Accolade noted which numbers were typed before it.", plural(gone, "number"))
	case len(kept) > 0:
		fmt.Fprintf(&msg, "the %s the lock gave go; the ones you typed before locking stay (%s).", plural(gone, "number"), strings.Join(kept, ", "))
	default:
		fmt.Fprintf(&msg, "the %s the lock gave go, and the script is as it was before the lock.", plural(gone, "number"))
	}
	if omitted > 0 {
		fmt.Fprintf(&msg, " %s lose their number too; their text stays as boneyard.",
			map[bool]string{true: "The OMITTED scene", false: fmt.Sprintf("The %d OMITTED scenes", omitted)}[omitted == 1])
	}
	msg.WriteString("\n\nEither way, new scenes get no number until you lock again, and Undo takes it back.")

	label := widget.NewLabel(msg.String())
	label.Wrapping = fyne.TextWrapWord
	var d dialog.Dialog
	keepBtn := widget.NewButton("Keep the Numbers", func() { d.Hide(); w.unlockNow(false) })
	removeBtn := widget.NewButton("Remove the Numbers", func() { d.Hide(); w.unlockNow(true) })
	cancel := widget.NewButton("Cancel", func() { d.Hide() })
	if issued {
		keepBtn.Importance = widget.HighImportance
	} else {
		removeBtn.Importance = widget.HighImportance
	}
	if gone == 0 {
		removeBtn.Disable()
	}
	d = dialog.NewCustomWithoutButtons("Unlock Scene Numbers", container.NewBorder(nil, w.dialogButtons(cancel, keepBtn, removeBtn), nil, nil, label), w.fyneWindow)
	d.Resize(fitDialog(w.fyneWindow, 520, 330))
	return d
}

// dialogButtons are a dialog's buttons: in a row at the right, Cancel
// first; one under the other on a phone, the default first.
func (w *MainWindow) dialogButtons(cancel *widget.Button, bs ...*widget.Button) fyne.CanvasObject {
	if c := w.fyneWindow.Canvas().Size(); c.Width > 0 && c.Width < 600 {
		var col []fyne.CanvasObject
		for _, b := range bs {
			if b.Importance == widget.HighImportance {
				col = append([]fyne.CanvasObject{b}, col...)
			} else {
				col = append(col, b)
			}
		}
		return container.NewVBox(append(col, cancel)...)
	}
	row := []fyne.CanvasObject{layout.NewSpacer(), cancel}
	for _, b := range bs {
		row = append(row, b)
	}
	return container.NewHBox(row...)
}

// unlockNow takes the lock's lines out of the production block (the
// block too if nothing else is in it), and with remove the numbers the
// lock wrote: one undo step.
func (w *MainWindow) unlockNow(remove bool) {
	e := w.textEditor
	text := e.Text()
	inside, _ := productionBlock(text)
	inside = withoutBlockLine(withSceneLock(inside, nil), handNumbersKey)
	heads := sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())
	p := numberPlan{nums: make([]string, len(heads)), drop: make([]bool, len(heads))}
	for i, h := range heads {
		p.nums[i] = h.num
	}
	if remove {
		keep, _ := handNumbersOf(text)
		p = removePlan(heads, keep)
		inside = pageLockFollowing(text, inside, heads, p)
	}
	w.applyNumberPlan(heads, p, inside)
	if !remove {
		w.setStatus("Scene numbers unlocked; the numbers stay in the script")
		return
	}
	n := 0
	for i, h := range heads {
		if h.num != "" && p.nums[i] == "" {
			n++
		}
	}
	w.setStatus("Scene numbers unlocked: " + plural(n, "number") + " taken out (Undo puts them back)")
}

// renumberScenes is Production > Renumber Scenes…: the dialog of
// strategies (renumber_dialog.go).
func (w *MainWindow) renumberScenes() {
	if w.markdown || w.textEditor == nil {
		return
	}
	note := w.splitOldOmittedRuns() // v1.13.0's OMITTED #12-14#: one heading a scene first
	text := w.textEditor.Text()
	heads := sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())
	numbered := false
	for _, h := range heads {
		numbered = numbered || h.num != ""
	}
	if !numbered {
		w.setStatus("No scene numbers to renumber: Lock Scene Numbers numbers the scenes")
		return
	}
	w.afterBundleWrites(func() {
		w.renumbering = w.newRenumberDialog()
		w.renumbering.note = note
		w.renumbering.d.Show()
	})
}

// renumberNow numbers the scenes 1…n again (the Clean preset): one undo
// step. It says so in the status bar, and returns what it said.
func (w *MainWindow) renumberNow() string {
	text := w.textEditor.Text()
	heads := sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())
	return w.renumberWith(cleanChoice(), w.renumberContextNow(text, heads))
}

// renumberWith numbers the scenes as c says, read against ctx: one undo
// step, the lock's "numbered before" line, the locked pages and (after)
// the breakdown and the schedule following. It says what it did in the
// status bar, and returns that.
func (w *MainWindow) renumberWith(c renumberChoice, ctx renumberContext) string {
	w.splitOldOmittedRuns() // v1.13.0's OMITTED #12-14#: one heading a scene first
	text := w.textEditor.Text()
	heads := sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())
	if bad := c.problems(heads); len(bad) > 0 {
		msg := "Not renumbered: " + strings.Join(bad, "; ")
		w.setStatus(msg)
		return msg
	}
	p := strategyPlan(heads, c, ctx)
	inside, _ := productionBlock(text)
	if sceneLockOf(text).locked {
		inside = withBlockLine(inside, handNumbersKey, handLineAfter(heads, p, ctx))
	}
	printRunsWas := omittedRunsPrinted(text)
	if c.Omitted != omitDrop { // drop leaves the setting as it is
		inside = withOmittedRuns(inside, c.Omitted == omitPrintRuns)
	}
	inside = pageLockFollowing(text, inside, heads, p)
	w.applyNumberPlan(heads, p, inside)
	_, changed := previewRows(heads, p, nil)
	dropped := 0
	for i := range heads {
		if p.drop[i] {
			dropped++
		}
	}
	msg := "Scenes renumbered: " + plural(changed, "heading") + " changed"
	if dropped > 0 {
		msg += "; " + plural(dropped, "OMITTED scene") + " kept as boneyard"
	}
	if now := omittedRunsPrinted(w.textEditor.Text()); now != printRunsWas {
		msg += map[bool]string{true: "; OMITTED runs now print as one (12-14 OMITTED)", false: "; OMITTED runs now print one heading a scene"}[now]
	}
	msg += " (Undo takes it back)"
	w.setStatus(msg)
	return msg
}
