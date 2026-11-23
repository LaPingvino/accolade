package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/scriptdiff"
)

// Merge Changes… in the Compare middle tab (merge.go has the rest): the
// base (From) and the copy (Changes from) compared three ways with the
// script now; each of their changes a row with Take and Leave (a
// conflict: both ways, and Edit), each scene Take All From This Scene;
// Merge Taken writes what was taken, and only that, as a new draft. The
// Compare tab has Take, Leave, Take Scene and Merge on the change in view.

// mergeState is a merge's: what was compared, and what is taken.
type mergeState struct {
	from   string // theirs' name, as the drafts say it
	baseID string
	ours   string // the script as compared
	m      *scriptdiff.Merge
	take   map[int]bool
	edits  map[int]string // a conflict's own text, edited
	// what was compared: the versions' ids and texts
	key          string
	base, theirs string
}

// Choices of a change.
const (
	choiceTake  = "Take"
	choiceLeave = "Leave"
	choiceEdit  = "Edit"
)

// outsideLabel is how the versions' lists name a copy from outside.
func outsideLabel(oc *outsideCopy) string { return oc.name + " (file)" }

// showMerge merges the copy oc into the script now, from the base
// baseID (nowID: the script now), in the Compare middle tab.
func showMerge(w *MainWindow, drafts []savedDraft, baseID string, oc *outsideCopy) *compareView {
	cv := showCompareWith(w, drafts, baseID, outsideID, oc, true)
	if baseID == nowID && len(drafts) == 0 {
		w.setStatus("No drafts to find its base in: compared with the script now, so every difference is theirs")
	}
	return cv
}

// chooseOutside is Compare's File…: a copy from outside to compare with
// (merging: whose changes to take), in To.
func (cv *compareView) chooseOutside() {
	w := cv.w
	w.chooseFile(mergeExtensions(), func(p string) {
		oc, err := w.readOutside(p)
		if err != nil {
			dialog.ShowError(err, w.fyneWindow)
			return
		}
		cv.outside = oc
		label := outsideLabel(oc)
		if i := indexOf(cv.ids, outsideID); i >= 0 {
			cv.labels[i] = label
		} else {
			cv.labels, cv.ids = append([]string{cv.labels[0], label}, cv.labels[1:]...), append([]string{cv.ids[0], outsideID}, cv.ids[1:]...)
		}
		from := cv.idOf(cv.from.Selected)
		cv.from.Options, cv.to.Options = cv.labels, cv.labels
		if cv.merge != nil {
			from = w.mergeBase(cv.drafts, oc.text, w.mergeOptions(oc.text, oc.lossy))
		}
		cv.from.Selected = cv.labelOf(from)
		cv.from.Refresh()
		cv.to.SetSelected(label)
	})
}

// newMergeBar is the Menus look's Take, Leave and Take Scene, beside
// Previous and Next (the ribbon's Compare tab has them otherwise).
func (cv *compareView) newMergeBar() *fyne.Container {
	take := widget.NewButtonWithIcon("Take", theme.ConfirmIcon(), cv.takeCurrent)
	leave := widget.NewButtonWithIcon("Leave", theme.ContentClearIcon(), cv.leaveCurrent)
	scene := widget.NewButtonWithIcon("Take Scene", theme.ContentAddIcon(), cv.takeCurrentScene)
	fyne.SetAccessibleLabel(scene, "Take all from this scene")
	return container.NewHBox(take, leave, scene)
}

// newMergeRow is what merging adds over the compare: what it does, and
// Merge Taken.
func (cv *compareView) newMergeRow() *fyne.Container {
	cv.mergeBtn = widget.NewButtonWithIcon("Merge Taken Changes", res(draftIcon)(), cv.doMerge)
	cv.mergeBtn.Importance = widget.HighImportance
	cv.mergeBtn.Disable()
	note := widget.NewLabel("Into the script now: nothing is written until you take a change and merge. " +
		"The script as it is is kept as a draft first; what is merged is a new draft.")
	note.Wrapping = fyne.TextWrapWord
	note.Importance = widget.LowImportance
	if cv.narrow {
		return container.NewVBox(note, cv.mergeBtn)
	}
	return container.NewBorder(nil, nil, nil, cv.mergeBtn, note)
}

// showMerged merges the base and the copy chosen with the script now, in
// the background, and shows their changes; what was taken stays taken
// while the three are the same (Side by side shows it again).
func (cv *compareView) showMerged() {
	w := cv.w
	baseID, theirsID := cv.idOf(cv.from.Selected), cv.idOf(cv.to.Selected)
	base, err := cv.version(baseID, false)
	var theirs string
	if err == nil {
		theirs, err = cv.version(theirsID, false)
	}
	if err != nil {
		cv.summary.SetText("Could not read it: " + err.Error())
		cv.body.RemoveAll()
		return
	}
	ours := w.textEditor.Text()
	lossy := theirsID == outsideID && cv.outside != nil && cv.outside.lossy
	opts := w.mergeOptions(theirs, lossy)
	key := baseID + "\x00" + theirsID
	ms := cv.merge
	if ms.key != key || ms.ours != ours || ms.base != base || ms.theirs != theirs {
		*ms = mergeState{key: key, ours: ours, base: base, theirs: theirs, baseID: baseID, take: map[int]bool{}, edits: map[int]string{}}
	}
	ms.from = cv.to.Selected
	if theirsID == outsideID && cv.outside != nil {
		ms.from = cv.outside.name
	}
	cv.gen++
	gen := cv.gen
	cv.summary.SetText("Comparing...")
	inBackground(func() *scriptdiff.Merge {
		return scriptdiff.NewMerge(base, ours, theirs, opts)
	}, func(m *scriptdiff.Merge) {
		if gen == cv.gen {
			ms.m = m
			cv.renderMerge()
		}
	})
}

// renderMerge shows the changes by scene, each with its choice.
func (cv *compareView) renderMerge() {
	ms := cv.merge
	m := ms.m
	cv.body.RemoveAll()
	cv.changes, cv.titles, cv.changeIDs, cv.at = nil, nil, nil, -1
	cv.choices = map[int]*widget.RadioGroup{}
	cv.scroll.Offset.Y = 0
	cv.rows = nil
	if len(m.Changes) == 0 {
		s := "Nothing of theirs to take: the copy has no changes of its own since the base."
		if m.Kept > 0 {
			s += " " + plural(m.Kept, "change") + " of yours since then stay as they are."
		}
		cv.summary.SetText(s)
		cv.updateMerge()
		return
	}
	side := cv.sideBySide.Checked && !cv.narrow
	shown := 0
	for si, sc := range m.Scenes {
		if shown >= maxCompareRows {
			cv.body.Add(muted("⋯ " + plural(len(m.Changes)-shown, "more change") + " not shown: merge these first, then Merge Changes… again"))
			break
		}
		title := sc.Heading
		if title == "" {
			title = "Before the first scene"
		}
		if sc.New {
			if len(sc.Changes) > 0 && m.Changes[sc.Changes[0]].Conflict {
				title += "  (you cut it)"
			} else {
				title += "  (new)"
			}
		}
		h := widget.NewLabel(title)
		h.TextStyle.Bold = true
		h.Wrapping = fyne.TextWrapWord
		all := widget.NewButtonWithIcon("Take All From This Scene", theme.ContentAddIcon(), func() { cv.takeScene(si) })
		var head fyne.CanvasObject = container.NewBorder(nil, nil, nil, all, h)
		if cv.narrow {
			head = container.NewVBox(h, all)
		}
		cv.body.Add(container.NewVBox(widget.NewSeparator(), head))
		for _, id := range sc.Changes {
			c := m.Changes[id]
			row := cv.changeRow(c, side)
			cv.changes = append(cv.changes, row)
			cv.titles = append(cv.titles, strings.TrimSuffix(changeWhat(c), ":")+" in "+title)
			cv.changeIDs = append(cv.changeIDs, id)
			cv.body.Add(row)
			shown++
		}
	}
	cv.updateMerge()
	cv.body.Refresh()
}

// changeWhat says what a change is.
func changeWhat(c scriptdiff.Change) string {
	switch c.Kind {
	case scriptdiff.SceneCut:
		if c.Conflict {
			return "They cut this scene; you changed it since:"
		}
		return "They cut this scene:"
	case scriptdiff.SceneAdded:
		if c.Conflict {
			return "You cut this scene; they changed it:"
		}
		return "They added this scene:"
	case scriptdiff.SceneMoved:
		return "They moved this scene"
	}
	if c.Conflict {
		switch c.Row {
		case scriptdiff.Removed:
			return "You changed this; they cut it:"
		case scriptdiff.Added:
			return "You cut this; they changed it:"
		}
		return "Both changed this: yours, then theirs:"
	}
	switch c.Row {
	case scriptdiff.Removed:
		return "They cut:"
	case scriptdiff.Added:
		return "They added:"
	}
	return "They changed:"
}

// changeRow is a change as shown: what it is, the text, Take or Leave.
func (cv *compareView) changeRow(c scriptdiff.Change, side bool) fyne.CanvasObject {
	ms := cv.merge
	what := muted(changeWhat(c))
	var text fyne.CanvasObject
	switch {
	case c.Kind == scriptdiff.SceneMoved:
	case c.Conflict && c.Kind == scriptdiff.ChangeEdit && c.Ours != "" && c.Theirs != "":
		text = bothWays(c, side)
	case c.Kind == scriptdiff.SceneCut || c.Row == scriptdiff.Removed:
		text = diffRow(scriptdiff.Row{Kind: scriptdiff.Removed, Old: c.Ours}, side)
	case c.Row == scriptdiff.Added:
		text = diffRow(scriptdiff.Row{Kind: scriptdiff.Added, New: c.Theirs}, side)
	default:
		text = diffRow(scriptdiff.Row{Kind: scriptdiff.Changed, Old: c.Ours, New: c.Theirs, Words: c.Words}, side)
	}
	options := []string{choiceTake, choiceLeave}
	editable := c.Conflict && c.Kind != scriptdiff.SceneCut && c.Row != scriptdiff.Removed
	if editable {
		options = append(options, choiceEdit)
	}
	var edit *widget.Entry
	if editable {
		edit = widget.NewMultiLineEntry()
		edit.Wrapping = fyne.TextWrapWord
		edit.SetText(c.Theirs)
		if e, ok := ms.edits[c.ID]; ok {
			edit.SetText(e)
		}
		edit.OnChanged = func(s string) {
			if _, ok := ms.edits[c.ID]; ok {
				ms.edits[c.ID] = s
			}
		}
		edit.SetMinRowsVisible(3)
	}
	radio := widget.NewRadioGroup(options, nil)
	radio.Horizontal = true
	radio.Required = true
	fyne.SetAccessibleLabel(radio, strings.TrimSuffix(changeWhat(c), ":"))
	switch _, edited := ms.edits[c.ID]; {
	case edited:
		radio.Selected = choiceEdit
	case ms.take[c.ID]:
		radio.Selected = choiceTake
	default:
		radio.Selected = choiceLeave
	}
	radio.OnChanged = func(s string) {
		delete(ms.edits, c.ID)
		ms.take[c.ID] = s == choiceTake
		if s == choiceEdit && edit != nil {
			ms.edits[c.ID] = edit.Text
		}
		if edit != nil {
			setVisible(edit, s == choiceEdit)
		}
		cv.updateMerge()
	}
	cv.choices[c.ID] = radio
	box := container.NewVBox(what)
	if text != nil {
		box.Add(text)
	}
	box.Add(radio)
	if edit != nil {
		setVisible(edit, radio.Selected == choiceEdit)
		box.Add(edit)
	}
	return box
}

// bothWays is a conflict's two texts: yours and theirs, side by side or
// one under the other, the words that differ marked.
func bothWays(c scriptdiff.Change, side bool) fyne.CanvasObject {
	cut := widget.RichTextStyle{ColorName: theme.ColorNameError, Inline: true}
	add := widget.RichTextStyle{ColorName: theme.ColorNameSuccess, Inline: true}
	plain := widget.RichTextStyle{Inline: true}
	var old, nu []widget.RichTextSegment
	if c.Words != nil {
		for _, wd := range c.Words {
			switch wd.Kind {
			case scriptdiff.Same:
				old = append(old, &widget.TextSegment{Text: wd.Text, Style: plain})
				nu = append(nu, &widget.TextSegment{Text: wd.Text, Style: plain})
			case scriptdiff.Removed:
				old = append(old, &widget.TextSegment{Text: wd.Text, Style: cut})
			case scriptdiff.Added:
				nu = append(nu, &widget.TextSegment{Text: wd.Text, Style: add})
			}
		}
	} else {
		old = []widget.RichTextSegment{&widget.TextSegment{Text: c.Ours, Style: plain}}
		nu = []widget.RichTextSegment{&widget.TextSegment{Text: c.Theirs, Style: plain}}
	}
	text := func(head string, segs []widget.RichTextSegment) fyne.CanvasObject {
		rt := widget.NewRichText(segs...)
		rt.Wrapping = fyne.TextWrapWord
		l := widget.NewLabel(head)
		l.TextStyle.Italic = true
		return container.NewVBox(l, rt)
	}
	yours, theirs := text("Yours", old), text("Theirs", nu)
	if side {
		return container.NewGridWithColumns(2, yours, theirs)
	}
	return container.NewVBox(yours, theirs)
}

// updateMerge says how many are taken, Merge Taken on or off.
func (cv *compareView) updateMerge() {
	ms := cv.merge
	if ms == nil || ms.m == nil {
		return
	}
	n := scriptdiff.Taken(ms.take, ms.edits)
	if n == 0 {
		cv.mergeBtn.SetText("Merge Taken Changes")
		cv.mergeBtn.Disable()
	} else {
		cv.mergeBtn.SetText("Merge " + plural(n, "Taken Change"))
		cv.mergeBtn.Enable()
	}
	m := ms.m
	if len(m.Changes) > 0 {
		conflicts := 0
		for _, c := range m.Changes {
			if c.Conflict {
				conflicts++
			}
		}
		s := fmt.Sprintf("%s from %s in %s", plural(len(m.Changes), "change"), ms.from, plural(len(m.Scenes), "scene"))
		if conflicts > 0 {
			s += fmt.Sprintf(", %s (you changed it too)", plural(conflicts, "conflict"))
		}
		if m.Kept > 0 {
			verb := "stay"
			if m.Kept == 1 {
				verb = "stays"
			}
			s += fmt.Sprintf("; %s of yours %s", plural(m.Kept, "change"), verb)
		}
		cv.summary.SetText(fmt.Sprintf("%s. %d taken.", s, n))
	}
	cv.w.ribbonRefresh()
}

// current is the change in view (the first if none was gone to yet).
func (cv *compareView) current() int {
	if cv.merge == nil || len(cv.changeIDs) == 0 {
		return -1
	}
	if cv.at < 0 {
		cv.at = 0
	}
	return cv.changeIDs[cv.at]
}

// choose sets a change's choice, as its row's buttons do.
func (cv *compareView) choose(id int, choice string) {
	if r := cv.choices[id]; r != nil {
		r.SetSelected(choice)
	}
}

// takeCurrent takes the change in view and goes to the next.
func (cv *compareView) takeCurrent() { cv.decideCurrent(choiceTake) }

// leaveCurrent leaves it and goes to the next.
func (cv *compareView) leaveCurrent() { cv.decideCurrent(choiceLeave) }

func (cv *compareView) decideCurrent(choice string) {
	id := cv.current()
	if id < 0 {
		cv.w.setStatus("No changes to take")
		return
	}
	cv.choose(id, choice)
	if cv.at < len(cv.changeIDs)-1 {
		cv.step(1)
	} else {
		cv.w.setStatus(fmt.Sprintf("%s the last change: Merge Taken writes what is taken", map[string]string{choiceTake: "Took", choiceLeave: "Left"}[choice]))
	}
}

// takeScene takes every change in scene si.
func (cv *compareView) takeScene(si int) {
	if cv.merge == nil || cv.merge.m == nil || si < 0 || si >= len(cv.merge.m.Scenes) {
		return
	}
	for _, id := range cv.merge.m.Scenes[si].Changes {
		if _, edited := cv.merge.edits[id]; !edited {
			cv.choose(id, choiceTake)
		}
	}
	cv.w.setStatus("Took all from " + cmpTitle(cv.merge.m.Scenes[si].Heading))
}

func cmpTitle(h string) string {
	if h == "" {
		return "before the first scene"
	}
	return h
}

// takeCurrentScene is Take Scene: the change in view's scene, all of it.
func (cv *compareView) takeCurrentScene() {
	if id := cv.current(); id >= 0 {
		cv.takeScene(cv.merge.m.Changes[id].Scene)
	}
}

// doMerge is Merge Taken: what is taken into the script, as a new draft;
// the rest compared again.
func (cv *compareView) doMerge() {
	w := cv.w
	if cv.merge == nil {
		return
	}
	w.applyMerge(cv.merge, func(err error) {
		if err != nil {
			dialog.ShowError(err, w.fyneWindow)
			return
		}
		if w.compare == cv {
			cv.show() // what was left, offered still
		}
	})
}
