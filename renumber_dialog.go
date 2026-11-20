package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/lex"
)

// The Renumber Scenes… dialog: the strategy at the left (one under the
// other on a phone), the preview of old → new at the right (below), made
// by the same plan Apply writes (renumber_strategy.go).

// The presets.
const (
	presetClean = "Clean 1…n (before anything went out)"
	presetSent  = "After an issue: keep sent scenes"
	presetLast  = "Last used"
)

// The radio groups' options, in the order of their constants.
var (
	scopeOptions   = []string{"All scenes", "Selected scenes", "From scene … to …", "From the cursor on"}
	listOptions    = []string{"No list", "Keep these numbers", "Only these numbers"}
	letterOptions  = []string{"Fold A-scenes into whole numbers", "Keep the letters"}
	omittedOptions = []string{"Drop them, close the gaps", "Keep them with their numbers", "Merge runs: OMITTED 12-14"}
)

// renumberDlg is the dialog and its controls.
type renumberDlg struct {
	w        *MainWindow
	d        dialog.Dialog
	heads    []sceneHead
	ctx      renumberContext
	sent     string // the numbers that went out, as a keep list
	issued   bool
	preset   *widget.Select
	scope    *widget.RadioGroup
	from, to *widget.Select
	list     *widget.RadioGroup
	numbers  *widget.Entry
	bad      *widget.Label
	letters  *widget.RadioGroup
	omitted  *widget.RadioGroup
	summary  *widget.Label
	preview  *widget.List
	rows     []previewRow
	apply    *widget.Button
	draft    *widget.Button // after an issue: Save a Draft, Then Renumber
	filling  bool           // a preset is setting the controls
}

// sceneLabel is heading i as the range's pickers show it.
func sceneLabel(i int, h sceneHead) string {
	heading, _ := lex.SceneNumber(strings.TrimSpace(h.text))
	heading = strings.TrimPrefix(strings.TrimSpace(heading), ".")
	n := h.num
	if n == "" {
		n = fmt.Sprintf("(%d)", i+1)
	}
	return n + "  " + heading
}

// newRenumberDialog is the dialog for the script now (not shown).
func (w *MainWindow) newRenumberDialog() *renumberDlg {
	text := w.textEditor.Text()
	r := &renumberDlg{w: w, heads: sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())}
	r.ctx = w.renumberContextNow(text, r.heads)
	var what string
	r.issued, what = w.firstIssue()
	if chain, err := w.revisionChainNow(); err == nil {
		r.sent = sentNumbers(chain, w.sceneStarts(), r.heads)
	}

	presets := []string{presetClean, presetSent}
	if _, ok := w.lastChoice(); ok {
		presets = append(presets, presetLast)
	}
	r.preset = widget.NewSelect(presets, r.choosePreset)
	r.preset.PlaceHolder = "Your own"
	labels := make([]string, len(r.heads))
	for i, h := range r.heads {
		labels[i] = sceneLabel(i, h)
	}
	r.from = widget.NewSelect(labels, func(string) { r.changed() })
	r.to = widget.NewSelect(labels, func(string) { r.changed() })
	r.scope = widget.NewRadioGroup(scopeOptions, func(string) { r.changed() })
	r.list = widget.NewRadioGroup(listOptions, func(string) { r.changed() })
	r.numbers = widget.NewEntry()
	r.numbers.SetPlaceHolder("5, 12A, 20-24")
	r.numbers.OnChanged = func(string) { r.changed() }
	r.bad = widget.NewLabel("")
	r.bad.Wrapping = fyne.TextWrapWord
	r.bad.Importance = widget.WarningImportance
	r.bad.Hide()
	r.letters = widget.NewRadioGroup(letterOptions, func(string) { r.changed() })
	r.omitted = widget.NewRadioGroup(omittedOptions, func(string) { r.changed() })
	for _, g := range []*widget.RadioGroup{r.scope, r.list, r.letters, r.omitted} {
		g.Required = true
	}
	r.summary = widget.NewLabel("")
	r.summary.Wrapping = fyne.TextWrapWord
	r.preview = widget.NewList(func() int { return len(r.rows) },
		func() fyne.CanvasObject {
			mono := fyne.TextStyle{Monospace: true}
			col := func(chars string) fyne.Size { // a column as wide as chars, padded
				m := fyne.MeasureText(chars, theme.TextSize(), mono)
				return fyne.NewSize(m.Width+2*theme.InnerPadding(), m.Height+2*theme.InnerPadding())
			}
			old := widget.NewLabelWithStyle("", fyne.TextAlignTrailing, mono)
			old.Truncation = fyne.TextTruncateEllipsis
			nw := widget.NewLabelWithStyle("", fyne.TextAlignLeading, mono)
			nw.Truncation = fyne.TextTruncateEllipsis
			heading := widget.NewLabel("")
			heading.Truncation = fyne.TextTruncateEllipsis
			nums := container.NewHBox(
				container.New(layout.NewGridWrapLayout(col("12A-14B")), old),
				container.New(layout.NewGridWrapLayout(col("→ into 12-14")), nw))
			return container.NewBorder(nil, nil, nums, nil, heading)
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			c := o.(*fyne.Container)
			heading, nums := c.Objects[0].(*widget.Label), c.Objects[1].(*fyne.Container)
			old := nums.Objects[0].(*fyne.Container).Objects[0].(*widget.Label)
			nw := nums.Objects[1].(*fyne.Container).Objects[0].(*widget.Label)
			row := r.rows[id]
			imp := widget.MediumImportance
			if !row.changed {
				imp = widget.LowImportance // stays: greyed
			}
			old.Importance, nw.Importance = imp, imp
			old.SetText(row.old)
			nw.SetText("→ " + row.new)
			heading.SetText(row.heading)
		})

	section := func(s string) fyne.CanvasObject {
		return widget.NewLabelWithStyle(s, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}
	var top []fyne.CanvasObject
	if r.issued {
		warn := widget.NewLabel("The script was issued: " + what + ". Sides, schedules and printed pages that went out use the old numbers. " +
			"Accolade's breakdown, schedule and locked pages follow the scenes, but nothing printed or sent does: " +
			"save a draft first, so the script as numbered now can be found again (File > Drafts…).")
		warn.Wrapping = fyne.TextWrapWord
		top = append(top, warn)
	}
	controls := container.NewVBox(append(top,
		section("Preset"), r.preset,
		section("Scenes"), r.scope, container.NewGridWithColumns(2, r.from, r.to),
		section("Numbers"), r.list, r.numbers, r.bad,
		section("A-scenes"), r.letters,
		section("OMITTED"), r.omitted,
	)...)
	preview := container.NewBorder(r.summary, nil, nil, nil, r.preview)

	cancel := widget.NewButton("Cancel", func() { r.d.Hide() })
	r.apply = widget.NewButton("Renumber", func() {
		if r.fresh() {
			r.d.Hide()
			r.applyNow()
		}
	})
	buttons := []*widget.Button{r.apply}
	if r.issued {
		r.draft = widget.NewButton("Save a Draft, Then Renumber", func() {
			if r.fresh() {
				r.d.Hide()
				r.draftThenApply()
			}
		})
		r.draft.Importance = widget.HighImportance
		buttons = append(buttons, r.draft)
	} else {
		r.apply.Importance = widget.HighImportance
	}
	var body *container.Split
	if c := w.fyneWindow.Canvas().Size(); c.Width > 0 && c.Width < 600 {
		body = container.NewVSplit(container.NewVScroll(controls), preview)
	} else {
		body = container.NewHSplit(container.NewVScroll(controls), preview)
	}
	body.Offset = 0.55
	r.d = dialog.NewCustomWithoutButtons("Renumber Scenes", container.NewBorder(nil, w.dialogButtons(cancel, buttons...), nil, nil, body), w.fyneWindow)
	r.d.Resize(fitDialog(w.fyneWindow, 900, 640))

	if r.issued {
		r.preset.SetSelected(presetSent)
	} else {
		r.preset.SetSelected(presetClean)
	}
	return r
}

// choosePreset sets the controls to a preset.
func (r *renumberDlg) choosePreset(name string) {
	var c renumberChoice
	switch name {
	case presetClean:
		c = cleanChoice()
	case presetSent:
		c = sentChoice(r.sent)
	case presetLast:
		last, ok := r.w.lastChoice()
		if !ok {
			return
		}
		c = last
	default:
		return // "Your own": the controls as they are
	}
	r.set(c)
}

// set puts a choice into the controls.
func (r *renumberDlg) set(c renumberChoice) {
	r.filling = true
	defer func() { r.filling = false; r.refresh() }()
	if c.Scope == scopeRange && (c.From >= len(r.heads) || c.To >= len(r.heads)) {
		c.Scope = scopeAll // a range of another script
	}
	r.scope.SetSelected(scopeOptions[c.Scope])
	from, to := c.From, c.To
	if c.Scope != scopeRange {
		from, to = r.ctx.selFrom, r.ctx.selTo
	}
	if len(r.heads) > 0 {
		r.from.SetSelectedIndex(min(max(from, 0), len(r.heads)-1))
		r.to.SetSelectedIndex(min(max(to, 0), len(r.heads)-1))
	}
	r.list.SetSelected(listOptions[c.List])
	r.numbers.SetText(c.Numbers)
	r.letters.SetSelected(letterOptions[map[bool]int{false: 0, true: 1}[c.KeepLetters]])
	r.omitted.SetSelected(omittedOptions[c.Omitted])
}

// indexIn is s's place in options (0 if none).
func indexIn(options []string, s string) int {
	for i, o := range options {
		if o == s {
			return i
		}
	}
	return 0
}

// choice is the controls' choice.
func (r *renumberDlg) choice() renumberChoice {
	return renumberChoice{
		Scope:       renumberScope(indexIn(scopeOptions, r.scope.Selected)),
		From:        max(r.from.SelectedIndex(), 0),
		To:          max(r.to.SelectedIndex(), 0),
		List:        renumberList(indexIn(listOptions, r.list.Selected)),
		Numbers:     r.numbers.Text,
		KeepLetters: r.letters.Selected == letterOptions[1],
		Omitted:     lex.OmittedMode(indexIn(omittedOptions, r.omitted.Selected)),
	}
}

// changed is a control changed by hand: the choice is the writer's own.
func (r *renumberDlg) changed() {
	if r.filling || r.preset == nil {
		return
	}
	if r.preset.Selected != "" {
		r.filling = true
		r.preset.ClearSelected()
		r.filling = false
	}
	r.refresh()
}

// plan is the plan the controls make now.
func (r *renumberDlg) plan() numberPlan { return strategyPlan(r.heads, r.choice(), r.ctx) }

// refresh shows the controls' plan in the preview.
func (r *renumberDlg) refresh() {
	if r.preview == nil {
		return
	}
	c := r.choice()
	if c.Scope == scopeRange {
		r.from.Enable()
		r.to.Enable()
	} else {
		r.from.Disable()
		r.to.Disable()
	}
	if c.List == listNone {
		r.numbers.Disable()
	} else {
		r.numbers.Enable()
	}
	_, bad := numbersOf(c, r.heads)
	if c.List != listNone && len(bad) > 0 {
		r.bad.SetText("Not a scene number: " + strings.Join(bad, ", ") + " (kept only as written)")
		r.bad.Show()
	} else {
		r.bad.Hide()
	}
	var n int
	r.rows, n = previewRows(r.heads, r.plan())
	switch {
	case n == 0:
		r.summary.SetText("No number changes.")
	default:
		r.summary.SetText(fmt.Sprintf("%s of %d change:", plural(n, "heading"), len(r.heads)))
	}
	for _, b := range []*widget.Button{r.apply, r.draft} {
		if b == nil {
			continue
		}
		if n == 0 {
			b.Disable()
		} else {
			b.Enable()
		}
	}
	r.preview.Refresh()
}

// fresh tells whether the headings are still those the preview shows;
// if not (the lock numbered a new scene at a pause, a peer typed), the
// preview is made again from them and the dialog stays, saying so.
func (r *renumberDlg) fresh() bool {
	text := r.w.textEditor.Text()
	heads := sceneHeads(text[:bodyEndByte(text)], r.w.sceneStarts())
	same := len(heads) == len(r.heads)
	for i := 0; same && i < len(heads); i++ {
		same = heads[i].text == r.heads[i].text && heads[i].num == r.heads[i].num
	}
	if same {
		return true
	}
	from, to := r.from.SelectedIndex(), r.to.SelectedIndex()
	r.heads = heads
	r.ctx = r.w.renumberContextNow(text, heads)
	labels := make([]string, len(heads))
	for i, h := range heads {
		labels[i] = sceneLabel(i, h)
	}
	r.filling = true
	r.from.SetOptions(labels)
	r.to.SetOptions(labels)
	if len(heads) > 0 {
		r.from.SetSelectedIndex(min(max(from, 0), len(heads)-1))
		r.to.SetSelectedIndex(min(max(to, 0), len(heads)-1))
	}
	r.filling = false
	r.refresh()
	r.summary.SetText("The script changed meanwhile: this is the preview now. " + r.summary.Text)
	return false
}

// applyNow renumbers as the controls say, and keeps the choice as the
// last used.
func (r *renumberDlg) applyNow() string {
	c := r.choice()
	r.w.rememberChoice(c)
	return r.w.renumberWith(c, r.ctx)
}

// draftThenApply saves a draft, then renumbers.
func (r *renumberDlg) draftThenApply() {
	w := r.w
	w.addDraft("Before renumbering", false, func(sd savedDraft, _ bool, err error) {
		if err != nil {
			dialog.ShowError(fmt.Errorf("could not save the draft, so the scenes are not renumbered: %v", err), w.fyneWindow)
			return
		}
		w.setStatus("Draft “" + sd.Name + "” saved; " + r.applyNow())
	})
}
