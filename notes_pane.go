package main

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The Notes pane (View > Notes; LOOP.md, Joop 2026-11-24: "You kinda
// want to be able to add and see notes in context"): what the script
// holds besides the script, in its order, each under the scene or
// section it is in (open_items.go): the notes no other tool claims, the
// to-dos among them, the boneyard sections (a cut that reads as script,
// or a note of prose), the beats no scene grew from yet, and at the end
// the unsorted notes (the notes block at the script's end). A tap goes
// there; the item is edited here, and written into the text as one
// undo step; a boneyard is restored, a beat grows into a scene. The
// pane keeps nothing of its own: it reads the text again after each
// edit. Filters by kind; the status bar counts what is open (to-dos and
// beats) and opens the pane.

// notesRow is a row of the list: a heading (the scene or section the
// items under it are in), or an item (its index in all).
type notesRow struct {
	head string
	item int
}

type notesPanel struct {
	w     *MainWindow
	all   []openItem                       // the text's, as read last
	rows  []notesRow                       // shown: what the filters let through, under their headings
	kinds [len(noteFilters)]*widget.Button // the filters: To do, Cut, Note, Beats, Revision, Checks
	list  *widget.List
	empty *widget.Label
	timer *uiTimer

	// the item picked, edited below the list: its index (-1 none), the
	// item as read; adding: a new unsorted note is being written
	sel     int
	item    openItem
	adding  bool
	where   *widget.Label
	entry   *noteEntry
	save    *widget.Button
	keep    *widget.Button
	dismiss *widget.Button
	restore *widget.Button
	scene   *widget.Button
	remove  *widget.Button
	detail  *fyne.Container
	add     *widget.Button
	box     fyne.CanvasObject
}

// focusTarget is what takes the keyboard: the list, or with nothing in
// it, Add Note.
func (p *notesPanel) focusTarget() fyne.Focusable {
	if p.list.Visible() {
		return p.list
	}
	return p.add
}

// noteEntry is the item's text being edited: written into the script
// on Shift+Enter, Ctrl+Enter, Save, or when it loses the focus.
type noteEntry struct {
	widget.Entry
	p *notesPanel
}

func (e *noteEntry) FocusLost() {
	e.Entry.FocusLost()
	if e.p.dirty() {
		e.p.commit()
	}
}

// TypedShortcut takes Ctrl+Enter as Save; Escape is in TypedKey.
func (e *noteEntry) TypedShortcut(s fyne.Shortcut) {
	if c, ok := s.(*desktop.CustomShortcut); ok && (c.KeyName == fyne.KeyReturn || c.KeyName == fyne.KeyEnter) {
		e.p.commit()
		return
	}
	e.Entry.TypedShortcut(s)
}

// TypedKey: Escape leaves it as it was, back to the script.
func (e *noteEntry) TypedKey(k *fyne.KeyEvent) {
	if k.Name == fyne.KeyEscape {
		e.p.cancel()
		return
	}
	e.Entry.TypedKey(k)
}

func (w *MainWindow) notesPane() *notesPanel {
	if w.notes == nil {
		w.notes = newNotesPanel(w)
	}
	return w.notes
}

// The filters' kinds, in their order, and their names; the Revision
// tab's kind toggles are the same (revision_tab.go).
var noteFilters = [...][]openKind{{openTodo}, {openCut}, {openNote, openUnsorted}, {openBeat}, {openRevision}, {openCheck}}
var noteFilterNames = [len(noteFilters)]string{"To do", "Cut", "Note", "Beats", "Revision", "Checks"}

func newNotesPanel(w *MainWindow) *notesPanel {
	p := &notesPanel{w: w, sel: -1}
	for k := range p.kinds {
		b := widget.NewButton("", func() { p.toggle(k) })
		p.kinds[k] = b
	}
	p.list = widget.NewList(
		func() int { return len(p.rows) },
		func() fyne.CanvasObject {
			bar := canvas.NewRectangle(color.Transparent)
			bar.SetMinSize(fyne.NewSize(3, 1))
			what := widget.NewLabel("")
			what.SizeName = theme.SizeNameCaptionText
			what.Importance = widget.LowImportance
			what.Truncation = fyne.TextTruncateEllipsis
			text := widget.NewLabel("")
			text.Wrapping = fyne.TextWrapWord
			// a card over the pane (Slate's raised step), its edge the kind's colour
			return container.NewStack(newThemedFill(colorNameRaised), container.NewBorder(nil, nil, bar, nil, container.NewVBox(what, text)))
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			if id >= len(p.rows) {
				return
			}
			p.showRow(id, o.(*fyne.Container))
		},
	)
	p.list.OnSelected = func(id widget.ListItemID) {
		p.list.UnselectAll()
		if id < len(p.rows) && p.rows[id].item >= 0 {
			p.pick(p.rows[id].item, true)
		}
	}
	fyne.SetAccessibleLabel(p.list, "Notes, cuts, to-dos and beats")
	p.empty = widget.NewLabel("")
	p.empty.Wrapping = fyne.TextWrapWord
	p.empty.Importance = widget.LowImportance
	p.empty.SizeName = theme.SizeNameCaptionText

	add := widget.NewButtonWithIcon("Note", theme.ContentAddIcon(), w.addNoteAtCursor)
	p.add = add
	add.Importance = widget.LowImportance
	fyne.SetAccessibleLabel(add, "Add Note at the cursor")
	unsorted := widget.NewButtonWithIcon("Unsorted", theme.ContentAddIcon(), p.newUnsorted)
	unsorted.Importance = widget.LowImportance
	fyne.SetAccessibleLabel(unsorted, "New Unsorted Note")

	p.where = widget.NewLabel("")
	p.where.SizeName = theme.SizeNameCaptionText
	p.where.Truncation = fyne.TextTruncateEllipsis
	p.entry = &noteEntry{p: p}
	p.entry.MultiLine = true
	p.entry.Wrapping = fyne.TextWrapWord
	p.entry.SetMinRowsVisible(3)
	p.entry.ExtendBaseWidget(p.entry)
	p.entry.OnSubmitted = func(string) { p.commit() } // Shift+Enter
	fyne.SetAccessibleLabel(p.entry, "The note's text")
	p.save = widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), p.commit)
	p.save.Importance = widget.HighImportance
	p.keep = widget.NewButtonWithIcon("Keep", theme.DocumentSaveIcon(), func() { p.w.keepFinding(p.item) })
	p.keep.Importance = widget.HighImportance
	fyne.SetAccessibleLabel(p.keep, "Keep it as a revision note in the script")
	p.dismiss = widget.NewButtonWithIcon("Dismiss", theme.VisibilityOffIcon(), func() { p.w.dismissFinding(p.item) })
	fyne.SetAccessibleLabel(p.dismiss, "Dismiss it until the window closes")
	p.restore = widget.NewButtonWithIcon("Restore", theme.ContentUndoIcon(), p.restoreItem)
	p.scene = widget.NewButtonWithIcon("New Scene", theme.ContentAddIcon(), p.newScene)
	p.remove = widget.NewButtonWithIcon("", theme.DeleteIcon(), p.removeItem)
	fyne.SetAccessibleLabel(p.remove, "Remove it from the script")
	done := widget.NewButtonWithIcon("", theme.CancelIcon(), p.cancel)
	done.Importance = widget.LowImportance
	fyne.SetAccessibleLabel(done, "Close")
	p.detail = container.NewVBox(widget.NewSeparator(),
		container.NewBorder(nil, nil, nil, done, p.where), p.entry,
		container.NewHBox(p.save, p.keep, p.dismiss, p.restore, p.scene, layoutSpacer(), p.remove))
	p.detail.Hide()

	width := canvas.NewRectangle(color.Transparent)
	width.SetMinSize(fyne.NewSize(260, 0))
	filters := make([]fyne.CanvasObject, len(p.kinds))
	for i, b := range p.kinds {
		filters[i] = b
	}
	top := container.NewVBox(container.NewGridWithColumns(3, filters...),
		container.NewHBox(add, unsorted))
	// in the order Tab goes: the filters, the list, the editor
	body := container.New(layout.NewBorderLayout(top, p.detail, nil, nil), top, container.NewStack(p.list, container.NewVBox(p.empty)), p.detail)
	p.box = container.NewStack(width, body)
	return p
}

// layoutSpacer pushes what follows to the row's end.
func layoutSpacer() fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	return container.NewBorder(nil, nil, nil, nil, r)
}

// kindColor is the colour a kind is marked in: in the pane, the margin.
func kindColor(k openKind) color.Color {
	return kindColorOf(fyne.CurrentApp().Settings().Theme(), fyne.CurrentApp().Settings().ThemeVariant(), k)
}

// kindColorOf is kindColor in a theme: to-dos amber, cuts in the accent,
// beats green, notes the theme's notes' edge (Slate's blue).
func kindColorOf(th fyne.Theme, v fyne.ThemeVariant, k openKind) color.Color {
	switch k {
	case openTodo:
		return th.Color(theme.ColorNameWarning, v)
	case openCut:
		return th.Color(theme.ColorNamePrimary, v)
	case openBeat:
		return th.Color(theme.ColorNameSuccess, v)
	case openRevision: // its violet, standing out of the theme's background
		return fitInk(revisionNoteColor, th.Color(theme.ColorNameBackground, v))
	case openCheck:
		return th.Color(theme.ColorNameError, v)
	}
	return editor.Themed(th, colorNameNoteMark, v, th.Color(theme.ColorNameDisabled, v))
}

// revisionNoteColor is a kept finding's: a violet, neither a to-do's
// amber nor a finding's red.
var revisionNoteColor = color.NRGBA{0x8e, 0x5c, 0xd0, 0xff}

// showRow fills a row of the list.
func (p *notesPanel) showRow(id int, row *fyne.Container) {
	r := p.rows[id]
	card, inner := row.Objects[0].(*themedFill), row.Objects[1].(*fyne.Container)
	box, bar := inner.Objects[0].(*fyne.Container), inner.Objects[1].(*canvas.Rectangle)
	if r.item < 0 { // a heading: on the pane itself
		card.Hide()
	} else {
		card.Show()
	}
	what, text := box.Objects[0].(*widget.Label), box.Objects[1].(*widget.Label)
	if r.item < 0 { // a heading
		bar.FillColor = color.Transparent
		what.Hide()
		text.TextStyle = fyne.TextStyle{Bold: true}
		text.SizeName = theme.SizeNameCaptionText
		text.SetText(r.head)
	} else {
		it := p.all[r.item]
		bar.FillColor = kindColor(it.Kind)
		what.Show()
		label := it.Kind.String()
		switch {
		case it.Kind == openCut && it.Omitted != "":
			label = "Cut · scene " + it.Omitted + " omitted"
		case it.Kind == openNote && it.Shape == shapeBoneyard:
			label = "Note · boneyard"
		case it.Kind == openTodo && it.Done:
			label = "Done"
		}
		what.TextStyle.Bold = themeTraits().caps
		what.SetText(capsLabel(fmt.Sprintf("%s · line %d", label, it.Line+1)))
		text.TextStyle = fyne.TextStyle{}
		text.SizeName = theme.SizeNameText
		t := it.Text
		if it.Kind == openCut && it.Title != "" {
			t = it.Title
		}
		text.SetText(shortNote(t))
	}
	bar.Refresh()
	what.Refresh()
	text.Refresh()
	box.Refresh() // a recycled row: the kind's label shown or hidden, laid out again
	p.list.SetItemHeight(id, row.MinSize().Height)
}

// shortNote is a note's text as a row shows it: three lines at most.
func shortNote(t string) string {
	if t == "" {
		return "(empty)"
	}
	lines := strings.Split(t, "\n")
	if len(lines) > 3 {
		lines = append(lines[:3], "…")
	}
	t = strings.Join(lines, "\n")
	if r := []rune(t); len(r) > 240 {
		t = string(r[:240]) + "…"
	}
	return t
}

// read reads the text again and shows it.
func (p *notesPanel) read() {
	cancel(&p.timer)
	w := p.w
	if w.textEditor == nil {
		return
	}
	p.all = w.openItems()
	p.show()
	p.refind()
}

// edited follows the script's text once typing pauses.
func (p *notesPanel) edited() {
	p.w.debounce(&p.timer, 300*time.Millisecond, func() {
		if p.w.paneOn("notes") {
			p.read()
		}
	})
}

// show lists what the filters let through.
func (p *notesPanel) show() {
	var counts [openKinds]int
	for _, it := range p.all {
		counts[it.Kind]++
	}
	names := noteFilterNames
	for k, b := range p.kinds {
		n := 0
		for _, kind := range noteFilters[k] {
			n += counts[kind]
		}
		b.SetText(fmt.Sprintf("%s %d", names[k], n))
		if p.w.openHide[noteFilters[k][0]] {
			b.Importance = widget.LowImportance
		} else {
			b.Importance = widget.MediumImportance
		}
		b.Refresh()
	}
	p.rows = p.rows[:0]
	head := "\x00"
	for i, it := range p.all {
		if p.w.openHide[it.Kind] {
			continue
		}
		h := it.Ctx
		if h == "" {
			h = "Before the first scene"
			if it.Kind == openBeat {
				h = "Unplaced"
			}
		}
		if h != head {
			p.rows = append(p.rows, notesRow{head: h, item: -1})
			head = h
		}
		p.rows = append(p.rows, notesRow{item: i})
	}
	switch {
	case len(p.all) == 0:
		p.empty.SetText("Nothing here yet. Add Note (Ctrl+Shift+K) writes a note at the cursor, [[todo: …]] makes it a to-do; " +
			"/* … */ keeps a cut out of the print; = … plans a beat; Unsorted is for notes about the whole script.")
		p.empty.Show()
	case len(p.rows) == 0:
		p.empty.SetText("The filters hide all of them.")
		p.empty.Show()
	default:
		p.empty.Hide()
	}
	setVisible(p.list, len(p.rows) > 0) // an empty list: nothing for Tab to show its focus on
	p.list.Refresh()
}

// toggle shows or hides a filter's kinds (the Revision tab's too).
func (p *notesPanel) toggle(k int) { p.w.toggleOpenKind(k) }

// toggleOpenKind shows or hides filter k's kinds, in the Notes pane and
// the Revision tab's stepping alike.
func (w *MainWindow) toggleOpenKind(k int) {
	h := !w.openHide[noteFilters[k][0]]
	for _, kind := range noteFilters[k] {
		w.openHide[kind] = h
	}
	if w.notes != nil {
		w.notes.show()
	}
	w.updateOpen() // the margin's marks of the kinds shown
	w.ribbonRefresh()
}

// pick picks item i: the script's cursor goes there (go), the item into
// the editor below the list.
func (p *notesPanel) pick(i int, goThere bool) {
	if p.dirty() {
		p.commit()
		if i >= len(p.all) {
			return
		}
	}
	it := p.all[i]
	p.sel, p.item, p.adding = i, it, false
	where := it.Kind.String()
	if it.Ctx != "" {
		where += " · " + it.Ctx
	}
	p.where.SetText(fmt.Sprintf("%s · line %d", where, it.Line+1))
	p.entry.SetText(it.Text)
	p.entry.SetPlaceHolder("")
	finding := it.Kind == openCheck
	setVisible(p.save, !finding)
	setVisible(p.keep, finding)
	setVisible(p.dismiss, finding)
	if finding { // computed: nothing of it to edit
		p.entry.Disable()
	} else {
		p.entry.Enable()
	}
	setVisible(p.restore, it.restorable())
	if it.Kept {
		p.restore.SetText("Restore Scene " + it.Omitted)
	} else {
		p.restore.SetText("Restore")
	}
	setVisible(p.scene, it.Kind == openBeat && !p.w.markdown)
	setVisible(p.remove, !finding)
	p.detail.Show()
	if goThere {
		p.w.goToOffset(it.Start)
	}
}

// refind keeps the item picked after the text changed: the one with
// the same text nearest to where it was, else none.
func (p *notesPanel) refind() {
	if p.sel < 0 || p.adding {
		return
	}
	best, dist := -1, 0
	for i, it := range p.all {
		if it.Kind != p.item.Kind || it.Raw != p.item.Raw {
			continue
		}
		d := max(it.Start-p.item.Start, p.item.Start-it.Start)
		if best < 0 || d < dist {
			best, dist = i, d
		}
	}
	if best < 0 {
		if !p.entryFocused() {
			p.close()
		}
		return
	}
	p.sel, p.item = best, p.all[best]
}

// entryFocused tells whether the item's text is being typed.
func (p *notesPanel) entryFocused() bool {
	return p.w.fyneWindow.Canvas().Focused() == p.entry
}

// dirty tells whether the editor holds a change not written yet.
func (p *notesPanel) dirty() bool {
	if p.adding {
		return strings.TrimSpace(p.entry.Text) != ""
	}
	return p.sel >= 0 && p.detail.Visible() && p.entry.Text != p.item.Text
}

// close puts the editor away.
func (p *notesPanel) close() {
	p.sel, p.adding = -1, false
	p.detail.Hide()
}

// cancel leaves the item as it is, the editor away, the script focused.
func (p *notesPanel) cancel() {
	p.close()
	p.w.fyneWindow.Canvas().Focus(p.w.textEditor)
}

// apply makes an edit in the script, one undo step, the cursor at
// cursor (-1: the edit's start).
func (w *MainWindow) applyTextEdit(e textEdit, cursor int) {
	if cursor < 0 {
		cursor = e.Start
	}
	w.textEditor.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			b.Replace(e.Start, e.End, e.Text)
			b.SetCursor(min(cursor, b.Len()), false)
		})
	})
}

// current is the item picked as the text has it now; false (said in
// the status bar) if the text no longer has it as it was.
func (p *notesPanel) current() (openItem, string, bool) {
	text := p.w.textEditor.Text()
	if p.item.still(text) {
		return p.item, text, true
	}
	p.read() // the text moved on: find it again
	if p.sel >= 0 && p.item.still(text) {
		return p.item, text, true
	}
	p.w.setStatus("That note has changed in the script since: picked again, nothing written")
	return openItem{}, text, false
}

// commit writes the editor's text into the script.
func (p *notesPanel) commit() {
	w := p.w
	if p.adding {
		note := strings.TrimSpace(p.entry.Text)
		p.adding = false
		p.detail.Hide()
		if note == "" {
			return
		}
		text := w.textEditor.Text()
		w.applyTextEdit(newUnsorted(text, note, w.notesMarks()), w.textEditor.CursorOffset())
		w.setStatus("Unsorted note kept at the script's end")
		p.read()
		return
	}
	if !p.dirty() {
		return
	}
	it, text, ok := p.current()
	if !ok {
		return
	}
	nw := p.entry.Text
	var e textEdit
	if it.Kind == openUnsorted && strings.TrimSpace(nw) == "" {
		e = removeUnsorted(text, it, w.notesMarks())
	} else {
		e = it.editTo(nw, text, w.markdown)
	}
	kind := it.Kind
	p.sel = -1 // not found again by its old text: by where the edit put it
	w.applyTextEdit(e, keptCursor(w.textEditor.CursorOffset(), e, text))
	p.read()
	for i, n := range p.all {
		if n.Start == e.Start && n.Kind == kind {
			p.sel, p.item = i, n
			p.entry.SetText(n.Text)
			return
		}
	}
	p.close() // gone (emptied) or no longer of its kind
}

// keptCursor is where a cursor at c is after edit e of text: before it,
// as it was; after it, moved by what the edit took or added; in it, at
// its start.
func keptCursor(c int, e textEdit, text string) int {
	switch {
	case c <= e.Start:
		return c
	case c >= e.End:
		return c + len([]rune(e.Text)) - (e.End - e.Start)
	}
	return e.Start
}

// removeItem takes the item out of the script.
func (p *notesPanel) removeItem() {
	w := p.w
	if p.adding {
		p.close()
		return
	}
	it, text, ok := p.current()
	if !ok {
		return
	}
	e := it.removal(text)
	if it.Kind == openUnsorted {
		e = removeUnsorted(text, it, w.notesMarks())
	}
	w.applyTextEdit(e, -1)
	w.setStatus(it.Kind.String() + " taken out (Undo brings it back)")
	p.close()
	p.read()
}

// restoreItem takes a boneyard's marks away: its inside is script again.
func (p *notesPanel) restoreItem() {
	w := p.w
	it, text, ok := p.current()
	if !ok || !it.restorable() {
		return
	}
	if p.dirty() { // what was typed first, then restored
		p.commit()
		if it, text, ok = p.current(); !ok {
			return
		}
	}
	e, ok := it.restore(text, w.markdown)
	if !ok {
		w.setStatus("Scene " + it.Omitted + "'s text is not as Omit Scene kept it: Restore Scene cannot bring it back")
		return
	}
	w.applyTextEdit(e, -1)
	switch {
	case it.Kept:
		w.setStatus("Scene " + it.Omitted + " restored")
	case it.Title != "":
		w.setStatus("Restored: " + it.Title)
	default:
		w.setStatus("Restored into the script")
	}
	p.close()
	p.read()
	w.fyneWindow.Canvas().Focus(w.textEditor)
}

// newScene grows the beat into a scene: New Scene on it (new_scene.go).
func (p *notesPanel) newScene() {
	it, _, ok := p.current()
	if !ok {
		return
	}
	p.w.goToOffset(it.Start)
	p.w.showNewScene()
}

// newUnsorted opens the editor for a new unsorted note.
func (p *notesPanel) newUnsorted() {
	if p.dirty() {
		p.commit()
	}
	p.sel, p.adding = -1, true
	p.where.SetText("New unsorted note: kept at the script's end")
	p.entry.SetText("")
	p.entry.SetPlaceHolder("A note about the whole script")
	p.entry.Enable()
	p.save.Show()
	p.keep.Hide()
	p.dismiss.Hide()
	p.restore.Hide()
	p.scene.Hide()
	p.remove.Hide()
	p.detail.Show()
	p.w.fyneWindow.Canvas().Focus(p.entry)
}

// goToOffset puts the script's cursor at off (runes), the script in
// front and focused.
func (w *MainWindow) goToOffset(off int) {
	e := w.textEditor
	e.Navigate(func(b *buffer.Buffer) { b.SetCursor(max(0, min(off, b.Len())), false) })
	if !w.scriptInMain() {
		w.showMiddle("")
	}
	w.fyneWindow.Canvas().Focus(e)
}

// addNoteAtCursor writes an empty note at the cursor, the cursor in it:
// [[ ]] in a screenplay, <!-- --> in Markdown; one undo step.
func (w *MainWindow) addNoteAtCursor() {
	e := w.textEditor
	if e == nil || w.scriptAway("Add Note") {
		return
	}
	open, close := "[[", "]]"
	if w.markdown {
		open, close = "<!-- ", " -->"
	}
	at := e.CursorOffset()
	r := []rune(e.Text())
	ins := open + close
	if at > 0 && at <= len(r) && r[at-1] != ' ' && r[at-1] != '\n' && r[at-1] != '\t' {
		ins = " " + ins // after a word: a space between
	}
	cur := at + len([]rune(ins)) - len([]rune(close))
	w.applyTextEdit(textEdit{at, at, ins}, cur)
	w.fyneWindow.Canvas().Focus(e)
	w.setStatus("A note here: type it; the Notes pane lists it with the others")
}

// showNotes brings the Notes pane in (the status bar's count).
func (w *MainWindow) showNotes() {
	if !w.paneOn("notes") {
		w.togglePane("notes")
		return
	}
	w.showPane("notes", false)
}

// newUnsortedNote is the command: the pane in, its editor for a new one.
func (w *MainWindow) newUnsortedNote() {
	w.showNotes()
	w.notesPane().newUnsorted()
}

// notesCommands are the notes' own commands (the pane's toggle is the
// dock's, View > Notes).
func notesCommands() []*command {
	anyDoc := func(e cmdEnv) bool { return !e.away }
	return []*command{
		&command{id: "write.note", label: "Add Note", ex: "note", text: "Note", ribbonLabel: "Note", editing: true,
			tip:  "A note at the cursor, never printed: [[…]] (Markdown: <!-- … -->); [[todo: …]] makes it a to-do",
			keys: []fyne.Shortcut{shortcut(fyne.KeyK, fyne.KeyModifierShift)}, icon: theme.DocumentCreateIcon,
			when: anyDoc, run: (*MainWindow).addNoteAtCursor},
		&command{id: "write.unsortedNote", label: "New Unsorted Note", ex: "unsorted-note",
			tip:  "A note about the whole script, not tied to a place: kept at its end, listed in the Notes pane",
			icon: theme.ContentAddIcon, run: (*MainWindow).newUnsortedNote},
	}
}
