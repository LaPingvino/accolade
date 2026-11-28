package main

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The Revision tab (LOOP.md, REVISION TOOLS; Joop 2026-11-24: "Maybe
// provide those as different colored in-work comments you can skip
// through and save, pick up whenever you want and sort through the type
// etc, with possibly mass-appliable actions"): the open items the Notes
// pane lists (open_items.go) and the checks' findings (revision_checks.go),
// stepped through as Find steps through matches. The current item is the
// one the cursor is in: nothing is kept but the text and the cursor.
// Next and Previous (F8, Shift+F8; :next-item, :previous-item) go from
// it; the kinds shown are the Notes pane's filters, one for both; the
// item's own actions (Keep a finding as a revision note, Dismiss, Done,
// Restore a cut, New Scene on a beat, Edit it in the pane) and those on
// many at once (Remove Done To-dos, Restore Cuts in This Scene) are each
// one undo step. Where the beats land (beat_place.go) is its statistic.

// revEnv is what the Revision tab's commands can do now: the current
// item's kind and the bulk actions' counts.
type revEnv struct {
	on                                     bool // an item at the cursor
	kind                                   openKind
	finding, todo, restorable, beat, notes bool
	keepable, dismissable                  bool
	doneTodos, sceneCuts                   int
	any                                    bool // items to step through
	src                                    bool // in the source (revEditor)
}

// revisionEnv fills e's Revision part from the script and its cursor.
func (w *MainWindow) revisionEnv(e *cmdEnv) {
	ed, src := w.revEditor()
	if ed == nil || (e.away && !src) {
		return
	}
	items := w.shownOpenItems()
	r := revEnv{any: len(items) > 0, src: src}
	cur := ed.CursorOffset()
	if i := currentOpen(items, cur); i >= 0 {
		it := items[i]
		r.on, r.kind = true, it.Kind
		r.finding = it.Kind == openCheck
		mine := !src // the source's notes are its own text's: written there
		r.todo = it.Kind == openTodo && mine
		r.restorable = it.restorable() && mine
		r.beat = it.Kind == openBeat && !w.markdown && mine
		r.notes = !r.finding && mine
		r.keepable = it.keepable() && mine
		r.dismissable = r.finding || (it.noteLike() && mine)
	}
	if src {
		e.rev = r
		return
	}
	all := w.openItems()
	cuts := false
	for _, it := range all {
		if it.Kind == openTodo && it.Done {
			r.doneTodos++
		}
		cuts = cuts || it.Kind == openCut
	}
	if cuts {
		r.sceneCuts = len(sceneCuts(all, w.textEditor.Text(), cur, w.sceneStarts(), w.markdown))
	}
	e.rev = r
}

// shownOpenItems are the open items of the kinds shown.
func (w *MainWindow) shownOpenItems() []openItem {
	var out []openItem
	for _, it := range w.revItems() {
		if !w.openHide[it.Kind] && !(it.Kind == openUnsorted && strings.HasPrefix(it.Text, dismissedHead)) { // the dismissed: the pane's
			out = append(out, it)
		}
	}
	return out
}

// currentOpen is the item the cursor is in (the narrowest, a note in a
// finding's line before the line), -1 none.
func currentOpen(items []openItem, cur int) int {
	best := -1
	for i, it := range items {
		if it.Start <= cur && cur <= it.End && (best < 0 || it.End-it.Start < items[best].End-items[best].Start) {
			best = i
		}
	}
	return best
}

// stepOpen goes to the next (dir 1) or previous (-1) item shown from the
// cursor's, round at the ends, and says what it is.
func (w *MainWindow) stepOpen(dir int) {
	ed, src := w.revAway("Next Open Item")
	if ed == nil {
		return
	}
	items := w.shownOpenItems()
	if len(items) == 0 {
		w.setStatus("Nothing open of the kinds shown: no to-dos, cuts, notes, beats without a scene, revision notes or findings")
		return
	}
	cur := ed.CursorOffset()
	from := cur
	k, round := -1, false
	if i := currentOpen(items, cur); i >= 0 { // from it by its place in the list: one at the same place is not skipped
		k = i + dir
		round = k < 0 || k >= len(items)
		k = (k + len(items)) % len(items)
	} else if dir > 0 {
		for i, it := range items {
			if it.Start > from {
				k = i
				break
			}
		}
		if k < 0 {
			k, round = 0, true
		}
	} else {
		for i := len(items) - 1; i >= 0; i-- {
			if items[i].Start < from {
				k = i
				break
			}
		}
		if k < 0 {
			k, round = len(items)-1, true
		}
	}
	it := items[k]
	if src {
		ed.Navigate(func(b *buffer.Buffer) { b.SetCursor(max(0, min(it.Start, b.Len())), false) })
		w.fyneWindow.Canvas().Focus(ed)
	} else {
		w.goToOffset(it.Start)
	}
	if w.notes != nil && w.paneOn("notes") && !src { // the pane shows it too: one list, two places
		p := w.notes
		p.read()
		for i, a := range p.all {
			if a.Start == it.Start && a.Kind == it.Kind && a.Raw == it.Raw {
				p.pick(i, false)
				break
			}
		}
	}
	say := fmt.Sprintf("%s %d of %d: %s", it.Kind, k+1, len(items), firstLine(itemWords(it)))
	if round {
		say = "Round again · " + say
	}
	w.setStatus(say)
	w.ribbonRefresh()
}

// itemWords is what an item says, as the status bar and the readout
// name it: a cut's title, else its text.
func itemWords(it openItem) string {
	if it.Kind == openCut && it.Title != "" {
		return it.Title
	}
	if it.Text == "" {
		return "(empty)"
	}
	return it.Text
}

// firstLine is the first line of s, cut short.
func firstLine(s string) string {
	s, _, _ = strings.Cut(s, "\n")
	if r := []rune(s); len(r) > 90 {
		s = string(r[:90]) + "…"
	}
	return s
}

// currentItem is the item at the cursor, of the kinds shown; false (said
// in the status bar) if there is none.
func (w *MainWindow) currentItem(what string) (openItem, bool) {
	ed, _ := w.revAway(what)
	if ed == nil {
		return openItem{}, false
	}
	items := w.shownOpenItems()
	i := currentOpen(items, ed.CursorOffset())
	if i < 0 {
		w.setStatus(what + ": the cursor is on no open item (Next Open Item, F8, goes to one)")
		return openItem{}, false
	}
	return items[i], true
}

// noteLike tells whether Dismiss takes the item out of the text: a note,
// a to-do, a revision note (in Markdown a comment).
func (it openItem) noteLike() bool {
	return it.Shape == shapeNote || it.Kind == openTodo || it.Kind == openRevision
}

// revAway is the editor the Revision tab works in (revEditor), and
// whether it is the source's; nil (said in the status bar) when neither
// the script nor the source is in front.
func (w *MainWindow) revAway(what string) (*editor.ScriptEditor, bool) {
	ed, src := w.revEditor()
	if ed == nil || (!src && w.scriptAway(what)) {
		return nil, false
	}
	return ed, src
}

// inSource tells (in the status bar) that what works on the script's
// own notes, not the source's: true when the Revision tab is in the source.
func (w *MainWindow) inSource(what string) bool {
	if _, src := w.revEditor(); src {
		w.setStatus(what + " works on the script's notes: in the source, Next steps through its findings and Dismiss puts one away")
		return true
	}
	return false
}

// openChanged reads the items again where they show: the pane, the
// margin, the status bar's count, the ribbon.
func (w *MainWindow) openChanged() {
	if w.notes != nil && w.paneOn("notes") {
		w.notes.read()
	}
	w.updateOpen()
	w.ribbonRefresh()
}

// keepFinding writes a finding into the text as a revision note at the
// end of its line: kept with the script, listed as a revision note.
func (w *MainWindow) keepFinding(it openItem) {
	if it.Kind != openCheck {
		w.setStatus("Keep is for a check's finding: a note is in the script already")
		return
	}
	if !it.keepable() {
		w.setStatus("This one is fixed in the text, not kept: a note on its line would change it")
		return
	}
	if w.inSource("Keep") { // a note in the source would be read as its passages' text
		return
	}
	text := w.textEditor.Text()
	if !it.still(text) {
		w.setStatus("That line has changed since: nothing written")
		w.openChanged()
		return
	}
	e := it.keepEdit()
	w.applyTextEdit(e, e.Start+len([]rune(e.Text)))
	w.setStatus("Kept as a revision note: " + firstLine(it.Text))
	w.openChanged()
}

// dismissFinding puts a finding out of sight: a line beside the window's
// document or in its notes block, as the setting says (dismissed_place.go;
// the script's for its source's findings too), so it stays dismissed when
// the file is opened again (in the notes, one undo step). The dismissed
// lines no finding says any more are let go then.
func (w *MainWindow) dismissFinding(it openItem) {
	if it.Kind != openCheck || w.textEditor == nil {
		return
	}
	gone := w.allDismissed()
	now := map[string]bool{it.dismissKey(): true}
	for k := range gone {
		if sourceDecision(k) { // the Source pane's (screenplay_pages.go): no finding's
			now[k] = true
		}
	}
	if w.sourceRef().Origin != sourceNone && w.srcEditor() == nil { // the source's not read now: none let go
		maps.Copy(now, gone)
	}
	for _, ed := range []*editor.ScriptEditor{w.textEditor, w.srcEditor()} {
		if ed == nil {
			continue
		}
		md := w.markdown || ed != w.textEditor
		all := proseItemsOf
		if !md {
			all = func(t string) []openItem { return checkItemsOf(t, w.sceneStarts()) }
		}
		for _, f := range all(ed.Text()) {
			if k := f.dismissKey(); gone[k] {
				now[k] = true
			}
		}
	}
	keys := make([]string, 0, len(now))
	for k := range now {
		keys = append(keys, k)
	}
	switch err := w.storeDismissed(keys); {
	case err != nil:
		w.setStatus("Could not keep the dismissal beside the document: " + err.Error())
	case w.dismissInDocument():
		w.setStatus("Dismissed (kept in the notes at the end, where deleting its line brings it back): " + firstLine(it.Text))
	default:
		w.setStatus("Dismissed (kept beside the document, where deleting its line brings it back): " + firstLine(it.Text))
	}
	if w.notes != nil {
		w.notes.close()
	}
	w.openChanged()
}

// srcEditor is the source's editor, when the source is open as a
// document (nil: it is not).
func (w *MainWindow) srcEditor() *editor.ScriptEditor {
	if w.srcDoc != nil {
		return w.srcDoc.ed
	}
	return nil
}

// dismissCurrent dismisses the item at the cursor: a finding out of sight
// (nothing written), a note, a to-do or a revision note taken out of the
// text (one undo step: Undo brings it back).
func (w *MainWindow) dismissCurrent() {
	it, ok := w.currentItem("Dismiss")
	if !ok {
		return
	}
	if it.Kind == openCheck {
		w.dismissFinding(it)
		return
	}
	if w.inSource("Dismiss of a note") {
		return
	}
	if !it.noteLike() { // a cut has Restore, a beat New Scene: the Notes pane's bin takes them out
		w.setStatus("Dismiss takes out a [[note]], a to-do or a revision note; a cut or a beat goes with the Notes pane's bin")
		return
	}
	e := it.removal(w.textEditor.Text())
	w.applyTextEdit(e, -1)
	w.setStatus(it.Kind.String() + " taken out (Undo brings it back)")
	w.openChanged()
}

// doneEdit is the edit that marks a to-do done ([[done: …]]) or, done,
// to do again ([[todo: …]]).
func (it openItem) doneEdit(markdown bool) textEdit {
	n := it
	if it.Done {
		n.Prefix = "todo: "
		if strings.ToUpper(it.Prefix) == it.Prefix {
			n.Prefix = "TODO: "
		}
	} else {
		n.Prefix = "done: "
	}
	return textEdit{it.Start, it.End, n.written(n.Text, markdown)}
}

// markDone marks the to-do at the cursor done, or a done one to do again.
func (w *MainWindow) markDone() {
	if w.inSource("Done") {
		return
	}
	it, ok := w.currentItem("Done")
	if !ok {
		return
	}
	if it.Kind != openTodo {
		w.setStatus("Done is for a to-do: [[todo: …]]")
		return
	}
	e := it.doneEdit(w.markdown)
	w.applyTextEdit(e, keptCursor(w.textEditor.CursorOffset(), e, w.textEditor.Text()))
	if it.Done {
		w.setStatus("To do again: " + firstLine(it.Text))
	} else {
		w.setStatus("Done: " + firstLine(it.Text) + " (Remove Done To-dos takes the done ones out)")
	}
	w.openChanged()
}

// restoreCurrent restores the cut (any boneyard) at the cursor.
func (w *MainWindow) restoreCurrent() {
	if w.inSource("Restore") {
		return
	}
	it, ok := w.currentItem("Restore")
	if !ok {
		return
	}
	if !it.restorable() {
		w.setStatus("Restore is for a cut, or any /* … */ boneyard")
		return
	}
	e, ok := it.restore(w.textEditor.Text(), w.markdown)
	if !ok {
		w.setStatus("Scene " + it.Omitted + "'s text is not as Omit Scene kept it: Restore Scene cannot bring it back")
		return
	}
	w.applyTextEdit(e, -1)
	w.setStatus("Restored: " + firstLine(itemWords(it)))
	w.openChanged()
}

// sceneOfBeat grows the beat at the cursor into a scene: New Scene there.
func (w *MainWindow) sceneOfBeat() {
	if w.inSource("New Scene") {
		return
	}
	it, ok := w.currentItem("New Scene")
	if !ok {
		return
	}
	if it.Kind != openBeat {
		w.setStatus("New Scene from a beat: the cursor on a synopsis no scene grew from")
		return
	}
	w.goToOffset(it.Start)
	w.showNewScene()
}

// editCurrent opens the item at the cursor in the Notes pane, its text
// to type in.
func (w *MainWindow) editCurrent() {
	if w.inSource("Edit") {
		return
	}
	it, ok := w.currentItem("Edit")
	if !ok {
		return
	}
	if it.Kind == openCheck {
		w.setStatus("A finding is the checks', not the script's: Keep makes it a revision note to edit")
		return
	}
	w.showNotes()
	p := w.notesPane()
	p.read()
	for i, a := range p.all {
		if a.Start == it.Start && a.Kind == it.Kind {
			p.pick(i, false)
			w.fyneWindow.Canvas().Focus(p.entry)
			return
		}
	}
}

// applyTextEdits makes edits (not overlapping) in the script as one undo
// step, the last first so that each one's place holds; the cursor kept.
func (w *MainWindow) applyTextEdits(edits []textEdit) {
	slices.SortFunc(edits, func(a, b textEdit) int { return b.Start - a.Start })
	cur := w.textEditor.CursorOffset()
	text := w.textEditor.Text()
	var made []textEdit
	limit := len([]rune(text)) + 1
	for _, e := range edits {
		if e.End > limit { // overlaps the one after it: left
			continue
		}
		made = append(made, e)
		limit = e.Start
	}
	for _, e := range made {
		cur = keptCursor(cur, e, text)
	}
	w.textEditor.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			for _, e := range made {
				b.Replace(e.Start, e.End, e.Text)
			}
			b.SetCursor(max(0, min(cur, b.Len())), false)
		})
	})
}

// confirmCount asks before an action on n items, the count in the
// question; yes runs it.
func (w *MainWindow) confirmCount(title, question string, yes func()) {
	dialog.ShowConfirm(title, question, func(ok bool) {
		if ok {
			yes()
		}
	}, w.fyneWindow)
}

// removeDoneTodos takes every done to-do out of the script, one undo step.
func (w *MainWindow) removeDoneTodos() {
	if w.textEditor == nil || w.scriptAway("Remove Done To-dos") {
		return
	}
	var done []openItem
	for _, it := range w.openItems() {
		if it.Kind == openTodo && it.Done {
			done = append(done, it)
		}
	}
	if len(done) == 0 {
		w.setStatus("No done to-dos: Done marks one ([[done: …]])")
		return
	}
	w.confirmCount("Remove Done To-dos", fmt.Sprintf("Take the %s out of the script? (One undo step brings them back.)", plural(len(done), "done to-do")), func() {
		text := w.textEditor.Text()
		var edits []textEdit
		for _, it := range done {
			if it.still(text) {
				edits = append(edits, it.removal(text))
			}
		}
		w.applyTextEdits(edits)
		w.setStatus(plural(len(edits), "done to-do") + " taken out (Undo brings them back)")
		w.openChanged()
	})
}

// sceneCuts are the cuts in the scene (or section) the cursor is in: the
// boneyards that read as script under its heading.
func sceneCuts(items []openItem, text string, cur int, starts []string, markdown bool) []openItem {
	if markdown {
		return nil
	}
	ctx := -1
	line := strings.Count(string([]rune(text)[:min(cur, len([]rune(text)))]), "\n")
	for _, o := range outlineOf(text, starts) {
		if o.Line > line {
			break
		}
		ctx = o.Line
	}
	var out []openItem
	for _, it := range items {
		if it.Kind == openCut && it.CtxLine == ctx {
			out = append(out, it)
		}
	}
	return out
}

// restoreSceneCuts restores every cut in the cursor's scene, one undo step.
func (w *MainWindow) restoreSceneCuts() {
	if w.textEditor == nil || w.scriptAway("Restore Cuts in This Scene") {
		return
	}
	text := w.textEditor.Text()
	cuts := sceneCuts(w.openItems(), text, w.textEditor.CursorOffset(), w.sceneStarts(), w.markdown)
	if len(cuts) == 0 {
		w.setStatus("No cuts in this scene: a cut is a /* boneyard */ that reads as script")
		return
	}
	where := cuts[0].Ctx
	if where == "" {
		where = "before the first scene"
	}
	w.confirmCount("Restore Cuts", fmt.Sprintf("Restore the %s in %s? (One undo step takes them out again.)", plural(len(cuts), "cut"), where), func() {
		text := w.textEditor.Text()
		var edits []textEdit
		for _, it := range cuts {
			if !it.still(text) {
				continue
			}
			if e, ok := it.restore(text, w.markdown); ok {
				edits = append(edits, e)
			}
		}
		w.applyTextEdits(edits)
		w.setStatus(plural(len(edits), "cut") + " restored (Undo takes them out again)")
		w.openChanged()
	})
}

// revisionReadout is the Step group's readout: where the cursor is among
// the items shown ("Check 3 of 12"), else how many there are.
func (w *MainWindow) revisionReadout() string {
	ed, _ := w.revEditor()
	if ed == nil {
		return "Open Items"
	}
	items := w.shownOpenItems()
	if len(items) == 0 {
		return "None Open"
	}
	if i := currentOpen(items, ed.CursorOffset()); i >= 0 {
		return fmt.Sprintf("%s %d of %d", items[i].Kind, i+1, len(items))
	}
	return plural(len(items), "Item")
}

// revisionNamed is the readout's tooltip: what the item at the cursor says.
func (w *MainWindow) revisionNamed() string {
	ed, _ := w.revEditor()
	if ed == nil {
		return "Open items"
	}
	items := w.shownOpenItems()
	if i := currentOpen(items, ed.CursorOffset()); i >= 0 {
		return "Open Items: " + items[i].Kind.String() + ", " + firstLine(itemWords(items[i])) + " (the Notes pane lists them all)"
	}
	return "Open Items: " + plural(len(items), "item") + " of the kinds shown (the Notes pane lists them)"
}

// revisionCommands are the Revision tab's.
func revisionCommands() []*command {
	script := func(e cmdEnv) bool { return !e.away || e.rev.src }
	cmds := []*command{
		{id: "revision.next", label: "Next Open Item", ribbonLabel: "Next", ex: "next-item", icon: theme.NavigateNextIcon,
			tip:  "The next to-do, cut, note, beat, revision note or finding of the kinds shown, from the cursor (round at the end)",
			keys: []fyne.Shortcut{plainKey(fyne.KeyF8, 0)}, when: script, run: func(w *MainWindow) { w.stepOpen(1) }},
		{id: "revision.previous", label: "Previous Open Item", ribbonLabel: "Previous", ex: "previous-item", icon: theme.NavigateBackIcon,
			tip:  "The open item before the cursor's, of the kinds shown",
			keys: []fyne.Shortcut{plainKey(fyne.KeyF8, fyne.KeyModifierShift)}, when: script, run: func(w *MainWindow) { w.stepOpen(-1) }},
		{id: "revision.where", label: "Open Items", ex: "open-items", icon: theme.ListIcon,
			tip: "Where the cursor is among the open items: the Notes pane lists them", ribbonLabelFor: (*MainWindow).revisionReadout,
			nameFor: (*MainWindow).revisionNamed, run: (*MainWindow).showNotes},
		{id: "revision.keep", label: "Keep as Revision Note", ribbonLabel: "Keep", ex: "keep-finding", icon: theme.DocumentSaveIcon,
			tip:  "The finding at the cursor written into the text as a revision note ([[revision: …]] at its line's end; in Markdown <!-- revision: … --> under its paragraph): saved with it, picked up later",
			when: script, enabled: func(e cmdEnv) bool { return e.rev.keepable },
			run: func(w *MainWindow) {
				if it, ok := w.currentItem("Keep"); ok {
					w.keepFinding(it)
				}
			}},
		{id: "revision.dismiss", label: "Dismiss Item", ribbonLabel: "Dismiss", ex: "dismiss-item", icon: theme.VisibilityOffIcon,
			tip:  "A finding out of sight, a line beside the document or in its notes, as Dismissed in Doc says (opening the file again keeps it so; deleting the line brings it back); a note, to-do or revision note taken out of the script (Undo brings it back)",
			when: script, enabled: func(e cmdEnv) bool { return e.rev.dismissable }, run: (*MainWindow).dismissCurrent},
		{id: "revision.done", label: "Mark To-do Done", ribbonLabel: "Done", ex: "todo-done", icon: theme.ConfirmIcon,
			tip:  "The to-do at the cursor written as done, [[done: …]] (again: to do again)",
			when: script, enabled: func(e cmdEnv) bool { return e.rev.todo }, run: (*MainWindow).markDone},
		{id: "revision.restore", label: "Restore Cut", ribbonLabel: "Restore", ex: "restore-cut", icon: theme.ContentUndoIcon,
			tip:  "The boneyard at the cursor back in the script: its /* */ taken away (an omitted scene with its number)",
			when: script, enabled: func(e cmdEnv) bool { return e.rev.restorable }, run: (*MainWindow).restoreCurrent},
		{id: "revision.newScene", label: "New Scene from Beat", ribbonLabel: "New Scene", ex: "beat-scene", icon: theme.ContentAddIcon,
			tip:  "The beat at the cursor grown into a scene: New Scene's form, the beat its synopsis",
			when: func(e cmdEnv) bool { return !e.away && !e.markdown }, enabled: func(e cmdEnv) bool { return e.rev.beat }, run: (*MainWindow).sceneOfBeat},
		{id: "revision.edit", label: "Edit Item", ribbonLabel: "Edit", ex: "edit-item", icon: theme.DocumentCreateIcon,
			tip:  "The note, to-do or cut at the cursor in the Notes pane, its text to type in",
			when: script, enabled: func(e cmdEnv) bool { return e.rev.notes }, run: (*MainWindow).editCurrent},
		{id: "revision.removeDone", label: "Remove Done To-dos…", ribbonLabel: "Remove Done", ex: "remove-done", icon: theme.DeleteIcon,
			tip:  "Every done to-do taken out of the script, one undo step",
			when: script, enabled: func(e cmdEnv) bool { return e.rev.doneTodos > 0 }, run: (*MainWindow).removeDoneTodos},
		{id: "revision.restoreScene", label: "Restore Cuts in This Scene…", ribbonLabel: "Restore in Scene", ex: "restore-scene-cuts", icon: theme.ContentUndoIcon,
			tip:  "Every cut in the cursor's scene back in the script, one undo step",
			when: func(e cmdEnv) bool { return !e.away && !e.markdown }, enabled: func(e cmdEnv) bool { return e.rev.sceneCuts > 0 }, run: (*MainWindow).restoreSceneCuts},
	}
	cmds = append(cmds, dismissedPlaceCommand())
	for k := range noteFilters {
		name, what := noteFilterNames[k], noteFilterLabels[k]
		cmds = append(cmds, &command{id: "revision.show." + strings.ToLower(strings.ReplaceAll(name, " ", "")), label: "Show " + what,
			ribbonLabel: name, ex: "show-" + strings.ToLower(strings.ReplaceAll(what, " ", "-")),
			tip:     "Step through " + strings.ToLower(what) + " too: the Notes pane's filter, the same",
			checked: func(w *MainWindow) bool { return !w.openHide[noteFilters[k][0]] },
			run:     func(w *MainWindow) { w.toggleOpenKind(k) }})
	}
	return cmds
}

// noteFilterLabels are the filters' commands' words: Show To-dos …
var noteFilterLabels = [len(noteFilters)]string{"To-dos", "Cuts", "Notes", "Beats", "Revision Notes", "Findings"}

// revisionTab is the Revision tab: stepping through what is open, the
// kinds shown, the current item's actions, those on many, and where the
// beats land.
func revisionTab() ribbonTab {
	return ribbonTab{title: "Revision", groups: []ribbonGroup{
		{title: "Step", homes: []home{big("revision.next"), at("revision.previous"), at("revision.where")}},
		{title: "Show", homes: []home{at("revision.show.todo"), at("revision.show.cut"), at("revision.show.note"),
			at("revision.show.beats"), at("revision.show.revision"), at("revision.show.checks")}},
		{title: "Item", homes: []home{big("revision.keep"), at("revision.dismiss"), at("revision.done"),
			at("revision.restore"), at("revision.newScene"), at("revision.edit")}},
		{title: "All at Once", homes: []home{at("revision.removeDone"), at("revision.restoreScene"), at("revision.dismissInDoc")}},
		// a statistic with its call to action (beat_place.go)
		{title: "Beats", when: func(e cmdEnv) bool { return !e.markdown }, homes: []home{big("revision.beats"), at("revision.nextOff"), at("revision.previousOff"), at("write.insertStructure")},
			// its settings, behind its launcher
			more: []home{at("set.everySectionBeat")}},
	}}
}
