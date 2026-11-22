package main

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// choiceList is a one-of-several choice that takes the keyboard as a
// whole (Adapt…, adapt_dialog.go): each choice a row with a note under
// it, the chosen marked; Up and Down choose, Enter does OnEnter (the
// dialog's default action), a tap chooses. One Tab stop; the chosen row
// has a ring while the list has the focus.
type choiceList struct {
	widget.BaseWidget
	name     string // for screen readers: "How", "Where"
	items    []choiceItem
	chosen   int
	focused  bool
	OnChange func(int)
	OnEnter  func()

	box  *fyne.Container
	rows []*choiceRow
}

// choiceItem is a choice: what it is, a note (which, or why it is
// likely), and whether it can be chosen.
type choiceItem struct {
	Label, Note string
	Off         bool
}

func newChoiceList(name string, items []choiceItem, chosen int) *choiceList {
	l := &choiceList{name: name, box: container.NewVBox()}
	l.ExtendBaseWidget(l)
	l.SetItems(items, chosen)
	return l
}

// SetItems puts the choices in, chosen the one chosen.
func (l *choiceList) SetItems(items []choiceItem, chosen int) {
	l.items = items
	l.chosen = max(0, min(chosen, len(items)-1))
	l.rows = l.rows[:0]
	l.box.Objects = nil
	for i := range items {
		r := newChoiceRow(l, i)
		l.rows = append(l.rows, r)
		l.box.Objects = append(l.box.Objects, r)
	}
	l.show()
}

// Chosen is the chosen choice's index.
func (l *choiceList) Chosen() int { return l.chosen }

// choose makes choice i the chosen one, if it can be.
func (l *choiceList) choose(i int) {
	if i < 0 || i >= len(l.items) || l.items[i].Off || i == l.chosen {
		return
	}
	l.chosen = i
	l.show()
	if l.OnChange != nil {
		l.OnChange(i)
	}
}

// move chooses the next choice that can be chosen, by dir.
func (l *choiceList) move(dir int) {
	for i := l.chosen + dir; i >= 0 && i < len(l.items); i += dir {
		if !l.items[i].Off {
			l.choose(i)
			return
		}
	}
}

func (l *choiceList) show() {
	for _, r := range l.rows {
		r.show()
	}
	l.box.Refresh()
}

func (l *choiceList) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(l.box)
}

func (l *choiceList) FocusGained()   { l.focused = true; l.show() }
func (l *choiceList) FocusLost()     { l.focused = false; l.show() }
func (l *choiceList) TypedRune(rune) {}

func (l *choiceList) TypedKey(ev *fyne.KeyEvent) {
	switch ev.Name {
	case fyne.KeyUp, fyne.KeyLeft:
		l.move(-1)
	case fyne.KeyDown, fyne.KeyRight:
		l.move(1)
	case fyne.KeyHome:
		l.chooseFirst(1)
	case fyne.KeyEnd:
		l.chooseFirst(-1)
	case fyne.KeyReturn, fyne.KeyEnter:
		if l.OnEnter != nil {
			l.OnEnter()
		}
	}
}

// chooseFirst chooses the first choice that can be (dir 1), or the last.
func (l *choiceList) chooseFirst(dir int) {
	i, end := 0, len(l.items)
	if dir < 0 {
		i, end = len(l.items)-1, -1
	}
	for ; i != end; i += dir {
		if !l.items[i].Off {
			l.choose(i)
			return
		}
	}
}

// Tapped on the list (between rows): the focus, nothing chosen.
func (l *choiceList) Tapped(*fyne.PointEvent) { l.focus() }

func (l *choiceList) focus() {
	if c := fyne.CurrentApp().Driver().CanvasForObject(l); c != nil {
		c.Focus(l)
	}
}

func (l *choiceList) AccessibilityLabel() string {
	if l.chosen >= 0 && l.chosen < len(l.items) {
		return l.name + ": " + l.items[l.chosen].Label
	}
	return l.name
}

func (l *choiceList) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleList }

func (l *choiceList) AccessibilityValue() string {
	if l.chosen >= 0 && l.chosen < len(l.items) {
		return l.items[l.chosen].Label
	}
	return ""
}

// choiceRow is a choice's row: its mark, label and note, tapped to
// choose it.
type choiceRow struct {
	widget.BaseWidget
	l     *choiceList
	i     int
	bg    *canvas.Rectangle
	mark  *widget.Icon
	label *widget.Label
	note  *widget.Label
}

func newChoiceRow(l *choiceList, i int) *choiceRow {
	r := &choiceRow{l: l, i: i}
	r.bg = canvas.NewRectangle(color.Transparent)
	r.bg.CornerRadius = 4
	r.mark = widget.NewIcon(theme.RadioButtonIcon())
	r.label = widget.NewLabel("")
	r.note = widget.NewLabel("")
	r.note.SizeName = theme.SizeNameCaptionText
	r.note.Wrapping = fyne.TextWrapWord
	r.label.Wrapping = fyne.TextWrapWord
	r.ExtendBaseWidget(r)
	return r
}

func (r *choiceRow) CreateRenderer() fyne.WidgetRenderer {
	texts := container.New(&tightVBox{}, r.label, r.note)
	return widget.NewSimpleRenderer(container.NewStack(r.bg,
		container.NewBorder(nil, nil, container.NewCenter(r.mark), nil, texts)))
}

func (r *choiceRow) show() {
	it := r.l.items[r.i]
	chosen := r.i == r.l.chosen
	r.label.SetText(it.Label)
	r.label.TextStyle.Bold = chosen
	r.label.Importance = widget.MediumImportance
	if it.Off {
		r.label.Importance = widget.LowImportance
	}
	r.label.Refresh()
	r.note.SetText(it.Note)
	if it.Note == "" {
		r.note.Hide()
	} else {
		r.note.Show()
	}
	r.note.Importance = widget.MediumImportance
	if it.Off {
		r.note.Importance = widget.LowImportance
	}
	if strings.HasPrefix(it.Note, "likely") {
		r.note.Importance = widget.SuccessImportance
	}
	r.note.Refresh()
	if chosen {
		r.mark.SetResource(theme.RadioButtonCheckedIcon())
		r.bg.FillColor = tint(theme.ColorNamePrimary, 0x26)
	} else {
		r.mark.SetResource(theme.RadioButtonIcon())
		r.bg.FillColor = color.Transparent
	}
	r.bg.StrokeWidth = 0
	if chosen && r.l.focused {
		r.bg.StrokeColor, r.bg.StrokeWidth = theme.Color(theme.ColorNameFocus), 2
	}
	r.bg.Refresh()
}

func (r *choiceRow) Tapped(*fyne.PointEvent) {
	r.l.choose(r.i)
	r.l.focus()
}

// tightVBox stacks its objects with no gap: a label and its note.
type tightVBox struct{}

func (tightVBox) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var s fyne.Size
	for _, o := range objs {
		if !o.Visible() {
			continue
		}
		m := o.MinSize()
		s.Width = max(s.Width, m.Width)
		s.Height += m.Height - theme.Padding()
	}
	return s
}

func (tightVBox) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	y := float32(0)
	for _, o := range objs {
		if !o.Visible() {
			continue
		}
		h := o.MinSize().Height - theme.Padding()
		o.Move(fyne.NewPos(0, y-theme.Padding()))
		o.Resize(fyne.NewSize(size.Width, h+theme.Padding()))
		y += h
	}
}
