package main

import (
	"fmt"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/scriptdiff"
)

// Collaboration > Save Draft… and Collaboration > Drafts… (drafts.go has the store): the
// list of drafts, Compare (two drafts, or one with the script now, by
// scene and element: side by side on a desktop, one under the other on a
// phone; read-only, taking changes over is a later step) and Restore.

// showSaveDraft is Collaboration > Save Draft…: a name for the draft.
func (w *MainWindow) showSaveDraft() {
	w.afterBundleWrites(func() {
		ds, _ := readDrafts(w.extrasHome())
		entry := widget.NewEntry()
		entry.SetPlaceHolder(fmt.Sprintf("Draft %d", len(ds)+1))
		where := widget.NewLabel(draftsWhere(w))
		where.Wrapping = fyne.TextWrapWord
		where.Importance = widget.LowImportance
		var d *dialog.FormDialog
		d = dialog.NewForm("Save Draft", "Save", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Name", entry),
			widget.NewFormItem("", where),
		}, func(ok bool) {
			if !ok {
				return
			}
			w.addDraft(entry.Text, false, func(d savedDraft, _ bool, err error) {
				if err != nil {
					dialog.ShowError(fmt.Errorf("could not save the draft: %v", err), w.fyneWindow)
					return
				}
				w.setStatus("Draft “" + d.Name + "” saved")
			})
		}, w.fyneWindow)
		entry.OnSubmitted = func(string) { d.Submit() }
		d.Resize(fitDialog(w.fyneWindow, 460, 220))
		d.Show()
		w.fyneWindow.Canvas().Focus(entry)
	})
}

// draftsWhere says where the drafts are kept.
func draftsWhere(w *MainWindow) string {
	h := w.extrasHome()
	switch h.kind {
	case extrasNone:
		if w.currentFile == "" {
			return "Kept by Accolade until the script is saved."
		}
		return "Kept by Accolade for this file (saved as .accolade, they go inside it)."
	case extrasKept:
		return "Kept by Accolade for this file (saved as .accolade, they go inside it)."
	}
	return "Kept " + h.describe() + "."
}

// draftsDialog is Collaboration > Drafts….
type draftsDialog struct {
	w        *MainWindow
	drafts   []savedDraft // newest first
	list     *widget.List
	selected int
	dialog   *dialog.CustomDialog
	compare  *widget.Button
	restore  *widget.Button
}

// showDrafts is Collaboration > Drafts….
func (w *MainWindow) showDrafts() {
	w.afterBundleWrites(func() {
		dd, err := newDraftsDialog(w)
		if err != nil {
			dialog.ShowError(err, w.fyneWindow)
			return
		}
		dd.dialog.Show()
	})
}

func newDraftsDialog(w *MainWindow) (*draftsDialog, error) {
	ds, err := readDrafts(w.extrasHome())
	if err != nil {
		return nil, err
	}
	slices.Reverse(ds)
	dd := &draftsDialog{w: w, drafts: ds, selected: -1}
	size := fitDialog(w.fyneWindow, 640+dialogMargin, 520+dialogMargin).SubtractWidthHeight(dialogMargin, dialogMargin)
	narrow := size.Width < 600

	where := widget.NewLabel(draftsWhere(w))
	where.Wrapping = fyne.TextWrapWord
	where.Importance = widget.LowImportance
	dd.list = widget.NewList(func() int { return len(dd.drafts) },
		func() fyne.CanvasObject {
			name := widget.NewLabel("")
			name.TextStyle.Bold = true
			name.Truncation = fyne.TextTruncateEllipsis
			info := widget.NewLabel("")
			info.Truncation = fyne.TextTruncateEllipsis
			return container.NewVBox(name, info)
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			box := o.(*fyne.Container)
			d := dd.drafts[id]
			box.Objects[0].(*widget.Label).SetText(d.Name)
			box.Objects[1].(*widget.Label).SetText(dd.describe(d))
		})
	dd.list.OnSelected = func(id widget.ListItemID) {
		dd.selected = id
		dd.compare.Enable()
		dd.restore.Enable()
	}
	dd.list.OnUnselected = func(widget.ListItemID) {
		dd.selected = -1
		dd.compare.Disable()
		dd.restore.Disable()
	}
	dd.compare = widget.NewButtonWithIcon("Compare", theme.ViewRestoreIcon(), func() {
		if dd.selected >= 0 {
			showCompare(w, dd.drafts, dd.drafts[dd.selected].ID, "")
		}
	})
	dd.restore = widget.NewButtonWithIcon("Restore…", theme.HistoryIcon(), dd.askRestore)
	dd.compare.Disable()
	dd.restore.Disable()
	saveDraft := widget.NewButtonWithIcon("Save Draft…", theme.DocumentSaveIcon(), func() {
		dd.dialog.Hide()
		w.showSaveDraft()
	})
	closeButton := widget.NewButtonWithIcon("Close", theme.CancelIcon(), func() { dd.dialog.Hide() })
	closeButton.Importance = widget.HighImportance

	var body fyne.CanvasObject = dd.list
	if len(ds) == 0 {
		none := widget.NewLabel("No drafts yet. Save Draft… keeps the script as it is now, under a name, to come back to or compare with later.")
		none.Wrapping = fyne.TextWrapWord
		body = container.NewVBox(none)
	}
	var buttons fyne.CanvasObject
	if narrow {
		buttons = container.NewGridWithColumns(2, dd.compare, dd.restore, saveDraft, closeButton)
	} else {
		buttons = container.NewHBox(dd.compare, dd.restore, saveDraft, layout.NewSpacer(), closeButton)
	}
	content := container.NewBorder(where, container.NewVBox(widget.NewSeparator(), buttons), nil, nil, body)
	dd.dialog = dialog.NewCustomWithoutButtons("Drafts", content, w.fyneWindow)
	dd.dialog.SetOnEscape(func() { dd.dialog.Hide() })
	dd.dialog.Resize(size.AddWidthHeight(dialogMargin, dialogMargin))
	if len(ds) > 0 {
		dd.list.Select(0)
	}
	return dd, nil
}

// describe is a draft's second line: when, and what it shares.
func (dd *draftsDialog) describe(d savedDraft) string {
	s := d.Date.Local().Format("2 Jan 2006, 15:04")
	if d.Kind == kindIssue {
		s += " · issued"
		if d.Colour != "" {
			s += " (" + d.Colour + ")"
		}
	}
	for _, o := range dd.drafts { // the same text as an older one
		if o.ID != d.ID && o.Text == d.Text && o.Date.Before(d.Date) && !strings.HasPrefix(d.Text, draftsDir+d.ID+"/") {
			s += " · same text as “" + o.Name + "”"
			break
		}
	}
	if d.Source != "" && strings.HasPrefix(d.Source, draftsDir+d.ID+"/") {
		s += " · source changed"
	}
	return s
}

// askRestore asks, then restores the selected draft.
func (dd *draftsDialog) askRestore() {
	if dd.selected < 0 {
		return
	}
	d := dd.drafts[dd.selected]
	w := dd.w
	dialog.ShowConfirm("Restore “"+d.Name+"”?",
		"The script becomes this draft's text again (Undo takes it back). The script as it is now is kept as a draft first, so nothing is lost.",
		func(ok bool) {
			if !ok {
				return
			}
			w.restoreDraft(d, func(err error) {
				if err != nil {
					dialog.ShowError(err, w.fyneWindow)
					return
				}
				dd.dialog.Hide()
				w.setStatus("Restored “" + d.Name + "”, kept as a new draft")
			})
		}, w.fyneWindow)
}

// The compare view.

// nowID stands for the script as it is now among the drafts to compare.
const nowID = ""

// compareView is Compare: two versions, by scene and element.
type compareView struct {
	w          *MainWindow
	drafts     []savedDraft // newest first
	from, to   *widget.Select
	what       *widget.Select // Script or Source, if any draft has one
	sideBySide *widget.Check
	narrow     bool
	summary    *widget.Label
	body       *fyne.Container
	dialog     *dialog.CustomDialog
	labels     []string // the selects' options: drafts, then the script now
	ids        []string
	gen        int
	rows       []scriptdiff.Row // as shown last (tests)
}

const nowLabel = "The script now"

// showCompare compares draft fromID with toID (nowID: the script now).
func showCompare(w *MainWindow, drafts []savedDraft, fromID, toID string) *compareView {
	cv := &compareView{w: w, drafts: drafts}
	size := fitDialog(w.fyneWindow, 1000+dialogMargin, 680+dialogMargin).SubtractWidthHeight(dialogMargin, dialogMargin)
	cv.narrow = size.Width < 600
	cv.labels, cv.ids = []string{nowLabel}, []string{nowID}
	hasSource := w.draftSource() != nil
	for _, d := range drafts {
		cv.labels = append(cv.labels, draftLabel(d))
		cv.ids = append(cv.ids, d.ID)
		hasSource = hasSource || d.Source != ""
	}
	cv.from = widget.NewSelect(cv.labels, func(string) { cv.show() })
	cv.to = widget.NewSelect(cv.labels, func(string) { cv.show() })
	cv.what = widget.NewSelect([]string{"Script", "Source"}, func(string) { cv.show() })
	cv.what.SetSelected("Script")
	cv.sideBySide = widget.NewCheck("Side by side", func(on bool) {
		cv.w.settings.SetBoolean("drafts-side-by-side", on)
		_ = cv.w.settings.Save()
		cv.show()
	})
	cv.sideBySide.SetChecked(!w.settings.HasKey("drafts-side-by-side") || w.settings.GetBoolean("drafts-side-by-side"))
	cv.summary = widget.NewLabel("")
	cv.summary.Wrapping = fyne.TextWrapWord
	cv.summary.Importance = widget.LowImportance
	cv.body = container.NewVBox()
	closeButton := widget.NewButtonWithIcon("Close", theme.CancelIcon(), func() { cv.dialog.Hide() })
	closeButton.Importance = widget.HighImportance

	var top fyne.CanvasObject
	fromRow, toRow := labelled("From:", cv.from), labelled("To:", cv.to)
	extras := []fyne.CanvasObject{}
	if hasSource {
		extras = append(extras, labelled("Compare:", cv.what))
	}
	if cv.narrow {
		top = container.NewVBox(append([]fyne.CanvasObject{fromRow, toRow}, extras...)...)
	} else {
		extras = append(extras, cv.sideBySide)
		top = container.NewVBox(container.NewGridWithColumns(2, fromRow, toRow), container.NewHBox(extras...))
	}
	note := widget.NewLabel("Read only: restoring a whole draft is in Drafts…")
	note.Importance = widget.LowImportance
	buttons := container.NewHBox(note, layout.NewSpacer(), closeButton)
	if cv.narrow {
		buttons = container.NewHBox(layout.NewSpacer(), closeButton)
	}
	content := container.NewBorder(container.NewVBox(top, cv.summary),
		container.NewVBox(widget.NewSeparator(), buttons), nil, nil, container.NewVScroll(cv.body))
	cv.dialog = dialog.NewCustomWithoutButtons("Compare", content, w.fyneWindow)
	cv.dialog.SetOnEscape(func() { cv.dialog.Hide() })
	cv.dialog.Resize(size.AddWidthHeight(dialogMargin, dialogMargin))
	cv.from.SetSelected(cv.labelOf(fromID))
	cv.to.SetSelected(cv.labelOf(toID))
	cv.dialog.Show()
	return cv
}

func (cv *compareView) labelOf(id string) string {
	for i, x := range cv.ids {
		if x == id {
			return cv.labels[i]
		}
	}
	return nowLabel
}

func (cv *compareView) idOf(label string) string {
	for i, l := range cv.labels {
		if l == label {
			return cv.ids[i]
		}
	}
	return nowID
}

// version is a version's text: the script's or its source's.
func (cv *compareView) version(id string, source bool) (string, error) {
	w := cv.w
	if id == nowID {
		if source {
			return string(w.draftSource()), nil
		}
		return w.textEditor.Text(), nil
	}
	for _, d := range cv.drafts {
		if d.ID != id {
			continue
		}
		ref := d.Text
		if source {
			if ref = d.Source; ref == "" {
				return "", nil
			}
		}
		data, err := w.extrasHome().readExtra(ref)
		if err != nil {
			return "", err
		}
		text, _ := decodeScript(data)
		return text, nil
	}
	return "", fmt.Errorf("no such draft")
}

// show compares the two chosen, in the background, and shows it.
func (cv *compareView) show() {
	if cv.from.Selected == "" || cv.to.Selected == "" {
		return
	}
	source := cv.what.Selected == "Source"
	a, err := cv.version(cv.idOf(cv.from.Selected), source)
	if err == nil {
		var b string
		b, err = cv.version(cv.idOf(cv.to.Selected), source)
		if err == nil {
			cv.gen++
			gen := cv.gen
			heading := cv.headingTest(source)
			cv.summary.SetText("Comparing...")
			inBackground(func() []scriptdiff.Row {
				return scriptdiff.Diff(scriptdiff.Split(a, heading), scriptdiff.Split(b, heading))
			}, func(rows []scriptdiff.Row) {
				if gen == cv.gen {
					cv.render(rows)
				}
			})
			return
		}
	}
	cv.summary.SetText("Could not read it: " + err.Error())
	cv.body.RemoveAll()
}

// headingTest tells a scene heading's line (a Markdown heading's for a
// source or a Markdown document).
func (cv *compareView) headingTest(source bool) func(string) bool {
	if source || cv.w.markdown {
		return func(l string) bool { return strings.HasPrefix(l, "#") }
	}
	starts := cv.w.sceneStarts()
	return func(l string) bool { return isSceneHeadingIn(l, starts) }
}

// maxCompareRows is as many rows as Compare shows; past them, a note.
const maxCompareRows = 400

// render shows the rows by scene: a changed scene's heading, its changed
// elements (the unchanged counted), and the unchanged scenes counted
// between.
func (cv *compareView) render(rows []scriptdiff.Row) {
	cv.rows = rows
	cv.body.RemoveAll()
	removed, added, changed := scriptdiff.Count(rows)
	scenes := scriptdiff.Group(rows)
	nChanged := 0
	for _, s := range scenes {
		if s.Changed {
			nChanged++
		}
	}
	if removed+added+changed == 0 {
		cv.summary.SetText("The same, word for word.")
		return
	}
	cv.summary.SetText(fmt.Sprintf("%s changed: %s cut, %s added, %s changed.",
		plural(nChanged, "scene"), plural(removed, "element"), plural(added, "element"), plural(changed, "element")))
	side := cv.sideBySide.Checked && !cv.narrow
	shown, same := 0, 0
	flushSame := func() {
		if same > 0 {
			cv.body.Add(muted("⋯ " + plural(same, "scene") + " the same"))
			same = 0
		}
	}
	for _, s := range scenes {
		if !s.Changed {
			same++
			continue
		}
		flushSame()
		if shown >= maxCompareRows {
			cv.body.Add(muted("⋯ more changes: compare fewer drafts apart to see them all"))
			break
		}
		title := s.Heading
		rows := s.Rows
		if len(rows) > 0 && rows[0].Heading && rows[0].Kind != scriptdiff.Same && rows[0].Kind != scriptdiff.Changed {
			// a scene cut or added whole: said in its title
			title += map[scriptdiff.Kind]string{scriptdiff.Removed: "  (cut)", scriptdiff.Added: "  (new)"}[rows[0].Kind]
			rows = rows[1:]
		}
		if title == "" {
			title = "Before the first scene"
		}
		h := widget.NewLabel(title)
		h.TextStyle.Bold = true
		h.Wrapping = fyne.TextWrapWord
		cv.body.Add(container.NewVBox(widget.NewSeparator(), h))
		run := 0
		for i := 0; i < len(rows); i++ {
			r := rows[i]
			if r.Kind == scriptdiff.Same {
				if !r.Heading {
					run++
				}
				continue
			}
			if run > 0 {
				cv.body.Add(muted("⋯ " + plural(run, "element") + " the same"))
				run = 0
			}
			if side && r.Kind == scriptdiff.Removed && i+1 < len(rows) && rows[i+1].Kind == scriptdiff.Added {
				// cut on the left, what came instead on the right
				cv.body.Add(container.NewGridWithColumns(2, diffRow(r, false), diffRow(rows[i+1], false)))
				i++
			} else {
				cv.body.Add(diffRow(r, side))
			}
			shown++
		}
		if run > 0 {
			cv.body.Add(muted("⋯ " + plural(run, "element") + " the same"))
		}
	}
	flushSame()
	cv.body.Refresh()
}

// diffRow is one compared element: side by side (the old left, the new
// right) or one under the other.
func diffRow(r scriptdiff.Row, side bool) fyne.CanvasObject {
	cut := widget.RichTextStyle{ColorName: theme.ColorNameError, Inline: true, TextStyle: fyne.TextStyle{Strikethrough: true}}
	add := widget.RichTextStyle{ColorName: theme.ColorNameSuccess, Inline: true, TextStyle: fyne.TextStyle{Underline: true}}
	plain := widget.RichTextStyle{Inline: true}
	text := func(segs ...widget.RichTextSegment) *widget.RichText {
		rt := widget.NewRichText(segs...)
		rt.Wrapping = fyne.TextWrapWord
		return rt
	}
	seg := func(s string, st widget.RichTextStyle) widget.RichTextSegment {
		return &widget.TextSegment{Text: s, Style: st}
	}
	var old, new []widget.RichTextSegment
	switch r.Kind {
	case scriptdiff.Removed:
		old = []widget.RichTextSegment{seg(r.Old, cut)}
	case scriptdiff.Added:
		new = []widget.RichTextSegment{seg(r.New, add)}
	case scriptdiff.Changed:
		for _, wd := range r.Words {
			switch wd.Kind {
			case scriptdiff.Same:
				old = append(old, seg(wd.Text, plain))
				new = append(new, seg(wd.Text, plain))
			case scriptdiff.Removed:
				old = append(old, seg(wd.Text, cut))
			case scriptdiff.Added:
				new = append(new, seg(wd.Text, add))
			}
		}
	}
	if side {
		left, right := fyne.CanvasObject(layout.NewSpacer()), fyne.CanvasObject(layout.NewSpacer())
		if old != nil {
			left = text(old...)
		}
		if new != nil {
			right = text(new...)
		}
		return container.NewGridWithColumns(2, left, right)
	}
	if r.Kind == scriptdiff.Changed { // the words in one text, cut then added
		var all []widget.RichTextSegment
		for _, wd := range r.Words {
			switch wd.Kind {
			case scriptdiff.Same:
				all = append(all, seg(wd.Text, plain))
			case scriptdiff.Removed:
				all = append(all, seg(wd.Text, cut))
			case scriptdiff.Added:
				all = append(all, seg(wd.Text, add))
			}
		}
		return text(all...)
	}
	return text(append(old, new...)...)
}

func muted(s string) *widget.Label {
	l := widget.NewLabel(s)
	l.Importance = widget.LowImportance
	return l
}

// compareDrafts is Collaboration > Compare…: the newest draft with the
// script now, in Drafts' compare (the Drafts window, saying there are
// none yet, without one).
func (w *MainWindow) compareDrafts() {
	w.afterBundleWrites(func() {
		ds, err := readDrafts(w.extrasHome())
		if err != nil {
			dialog.ShowError(err, w.fyneWindow)
			return
		}
		if len(ds) == 0 {
			w.showDrafts()
			return
		}
		showCompare(w, ds, ds[len(ds)-1].ID, nowID)
	})
}
