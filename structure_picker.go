package main

import (
	"fmt"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/structure"
)

// Swap… and Move… (LOOP.md, Joop 2026-11-20): parts of the script
// swapped or moved as clear actions of their own, not hidden in a drag.
// A range is a scene, a run of scenes side by side (from one heading to
// another) or a section with all it holds; Swap puts range A where range
// B is and B where A was, Move puts A before or after another scene or
// section. The preview says what it makes (the outline after, what moved
// marked) or why it will not; with locked scene numbers, that they stay
// with their scenes and how they read then. One undo step, through the
// engine as every move.

// structPicker is the open Swap… or Move… dialog.
type structPicker struct {
	w     *MainWindow
	d     *dialog.CustomDialog
	units []structure.Unit
	names []string // the units as the lists show them

	mode               *widget.RadioGroup
	aFrom, aTo         *widget.Select
	bFrom, bTo, target *widget.Select
	where              *widget.Select // before, after
	withRow, toRow     *fyne.Container
	preview            *widget.Label
	after              *fyne.Container // the outline after
	ok                 *widget.Button
}

const (
	pickSwap = "Swap"
	pickMove = "Move"
)

// showStructPicker opens Swap… (move: Move…) with range A at the scene
// or section whose heading is on line from (-1: the one at the cursor).
func (w *MainWindow) showStructPicker(move bool, from int) {
	if w.textEditor == nil || w.markdown {
		w.setStatus("Swap… and Move… are for a screenplay's scenes and sections")
		return
	}
	p := w.newStructPicker()
	if len(p.units) < 2 {
		w.setStatus("Swap… and Move… need two scenes or sections at least")
		return
	}
	if from < 0 {
		tr, _ := w.structTree(w.textEditor.Text())
		line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
		from = tr.At(line)
	}
	a := slices.IndexFunc(p.units, func(u structure.Unit) bool { return u.Line == from })
	if a < 0 {
		a = 0
	}
	p.choose(a)
	mode := pickSwap
	if move {
		mode = pickMove
	}
	p.mode.SetSelected(mode) // updates
	p.d.Resize(fitDialog(w.fyneWindow, 560+dialogMargin, 600+dialogMargin))
	p.d.Show()
	w.structPick = p
}

// newStructPicker makes the dialog for the script as it is.
func (w *MainWindow) newStructPicker() *structPicker {
	p := &structPicker{w: w}
	tr, _ := w.structTree(w.textEditor.Text())
	p.units = tr.Outline()
	scene := 0
	seen := map[string]int{}
	for _, u := range p.units {
		name := strings.Repeat("§", u.Depth+1) + " " + u.Name // a section's: a § for each level
		if u.Scene {
			scene++
			name = fmt.Sprintf("%d. %s", scene, u.Name)
		}
		if seen[name]++; seen[name] > 1 {
			name += fmt.Sprintf(" (%d)", seen[name])
		}
		p.names = append(p.names, name)
	}
	upd := func(string) { p.update() }
	p.aFrom = widget.NewSelect(p.names, func(s string) { p.choose(slices.Index(p.names, s)) })
	p.aTo, p.bFrom, p.bTo, p.target = widget.NewSelect(nil, upd), widget.NewSelect(p.names, func(s string) { p.chooseB(slices.Index(p.names, s)) }),
		widget.NewSelect(nil, upd), widget.NewSelect(p.names, upd)
	p.where = widget.NewSelect([]string{"before", "after"}, upd)
	p.where.SetSelected("after")
	p.mode = widget.NewRadioGroup([]string{pickSwap, pickMove}, upd)
	p.mode.Horizontal = true
	p.preview = widget.NewLabel("")
	p.preview.Wrapping = fyne.TextWrapWord
	p.after = container.NewVBox()
	label := func(s string) *widget.Label {
		return widget.NewLabelWithStyle(s, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}
	p.withRow = container.NewVBox(label("with"), p.bFrom, container.NewBorder(nil, nil, widget.NewLabel("to"), nil, p.bTo))
	p.toRow = container.NewVBox(container.NewBorder(nil, nil, nil, nil, p.where), p.target)
	form := container.NewVBox(p.mode, label("Range"), p.aFrom, container.NewBorder(nil, nil, widget.NewLabel("to"), nil, p.aTo),
		p.withRow, p.toRow, widget.NewSeparator(), p.preview)
	scroll := container.NewVScroll(p.after)
	scroll.SetMinSize(fyne.NewSize(0, 120))
	content := container.NewBorder(form, nil, nil, nil, scroll)
	p.ok = widget.NewButton(pickSwap, p.apply)
	p.ok.Importance = widget.HighImportance
	cancel := widget.NewButton("Cancel", func() { p.d.Hide() })
	p.d = dialog.NewCustomWithoutButtons("Swap or Move", content, w.fyneWindow)
	p.d.SetButtons([]fyne.CanvasObject{cancel, p.ok})
	p.d.SetOnClosed(func() {
		if w.structPick == p {
			w.structPick = nil
		}
	})
	return p
}

// choose makes unit a the start of range A, its end offered among the
// units side by side after it, and B or the target the next one.
func (p *structPicker) choose(a int) {
	if a < 0 {
		return
	}
	tr, _ := p.w.structTree(p.w.textEditor.Text())
	p.aFrom.Selected = p.names[a]
	p.aFrom.Refresh()
	ends := p.namesOf(tr.Kin(p.units[a].Line))
	p.aTo.Options = ends
	p.aTo.Selected = p.names[a]
	p.aTo.Refresh()
	// B: the next unit of its kind side by side, else the next of its kind
	kin := tr.Kin(p.units[a].Line)
	b := -1
	if len(kin) > 1 {
		b = p.index(kin[1])
	}
	for i := a + 1; b < 0 && i < len(p.units); i++ {
		if p.units[i].Scene == p.units[a].Scene {
			b = i
		}
	}
	if b < 0 {
		b = max(a-1, 0)
	}
	p.chooseB(b)
	p.target.Selected = p.names[b] // Move: after it, a step down
	p.target.Refresh()
	p.update()
}

// chooseB makes unit b the start of range B.
func (p *structPicker) chooseB(b int) {
	if b < 0 {
		return
	}
	tr, _ := p.w.structTree(p.w.textEditor.Text())
	p.bFrom.Selected = p.names[b]
	p.bFrom.Refresh()
	p.bTo.Options = p.namesOf(tr.Kin(p.units[b].Line))
	p.bTo.Selected = p.names[b]
	p.bTo.Refresh()
	p.update()
}

// namesOf are the units' names whose headings are on lines.
func (p *structPicker) namesOf(lines []int) []string {
	var out []string
	for _, l := range lines {
		if i := p.index(l); i >= 0 {
			out = append(out, p.names[i])
		}
	}
	return out
}

// index is the unit whose heading is on line (-1: none).
func (p *structPicker) index(line int) int {
	return slices.IndexFunc(p.units, func(u structure.Unit) bool { return u.Line == line })
}

// line is the heading line of the unit named s (-1: none).
func (p *structPicker) line(s string) int {
	if i := slices.Index(p.names, s); i >= 0 {
		return p.units[i].Line
	}
	return -1
}

// result is what the dialog's choice makes of the text, or why not.
func (p *structPicker) result(text string) (structure.Result, error) {
	tr, read := p.w.structTree(text)
	a := structure.Range{First: p.line(p.aFrom.Selected), Last: p.line(p.aTo.Selected)}
	if p.mode.Selected == pickMove {
		return tr.Move(a, p.line(p.target.Selected), p.where.Selected == "after", read)
	}
	return tr.Swap(a, structure.Range{First: p.line(p.bFrom.Selected), Last: p.line(p.bTo.Selected)}, read)
}

// update shows the rows of the mode and the preview.
func (p *structPicker) update() {
	if p.mode == nil || p.ok == nil {
		return
	}
	move := p.mode.Selected == pickMove
	p.withRow.Hidden, p.toRow.Hidden = move, !move
	p.ok.SetText(map[bool]string{false: pickSwap, true: pickMove}[move])
	text := p.w.textEditor.Text()
	res, err := p.result(text)
	p.after.RemoveAll()
	if err != nil {
		p.preview.SetText("Not possible: " + err.Error())
		p.ok.Disable()
		p.after.Refresh()
		return
	}
	after := structure.Text(res.Lines)
	p.preview.SetText(res.Msg + "." + lockedNote(text, after, p.w.sceneStarts()))
	p.ok.Enable()
	// the outline after, what moved marked
	moved := map[int]bool{}
	tr, _ := p.w.structTree(text)
	ranges := [][2]int{{p.line(p.aFrom.Selected), p.line(p.aTo.Selected)}}
	if !move {
		ranges = append(ranges, [2]int{p.line(p.bFrom.Selected), p.line(p.bTo.Selected)})
	}
	for _, r := range ranges {
		for _, u := range tr.Outline() {
			if u.Line >= r[0] && u.Line <= r[1] && u.Line < len(res.Pos) && res.Pos[u.Line] >= 0 {
				moved[res.Pos[u.Line]] = true
			}
		}
	}
	nt, _ := p.w.structTree(after)
	for _, u := range nt.Outline() {
		name := strings.Repeat("    ", u.Depth) + u.Name
		l := widget.NewLabel(name)
		l.Truncation = fyne.TextTruncateEllipsis
		if moved[u.Line] {
			l.SetText(strings.Repeat("    ", u.Depth) + "→ " + u.Name)
			l.TextStyle.Bold = true
		}
		p.after.Add(l)
	}
	p.after.Refresh()
}

// apply makes the move, as one undo step.
func (p *structPicker) apply() {
	w := p.w
	text := w.textEditor.Text()
	res, err := p.result(text)
	if err != nil {
		w.setStatus(err.Error())
		return
	}
	p.d.Hide()
	w.applyStruct(text, res, false, -1, 0)
}

// itemMenu is a scene's or section's menu in the outline and on the cards:
// its color, and Swap… and Move… with it as the range.
func (w *MainWindow) itemMenu(it outlineItem, pos fyne.Position) {
	if w.markdown {
		w.pickColor(it, pos, nil)
		return
	}
	items := w.colorMenu(it, nil).Items
	items = append(items, fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Swap…", func() { w.showStructPicker(false, it.Line) }),
		fyne.NewMenuItem("Move…", func() { w.showStructPicker(true, it.Line) }))
	m := widget.NewPopUpMenu(fyne.NewMenu("", items...), w.fyneWindow.Canvas())
	fyne.SetAccessibleLabel(m, it.Text)
	m.ShowAtPosition(pos)
}
