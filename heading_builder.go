package main

import (
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/rules"
)

// The heading pattern builder (Joop, 2026-11-23: "a better way to
// describe this in the settings"): a language's Headings in the settings
// editor, each pattern as its slots (chips) with the text between them
// (small entries), an example in the language's own words under it and
// whether that reads as a heading. The TOML stays the source: every
// change is one edit of the Headings list (rules.JoinPattern puts a
// pattern together byte for byte as rules.ParsePattern took it apart),
// the rest of the file as it was. A language with scene words shows
// Fountain's own form (INT. KITCHEN - DAY) first, read always, not to be
// edited: it is no pattern of the list.

// slotNames are the slots as the builder and New Scene name them.
var slotNames = map[string]string{
	rules.SlotNumber: "Number", rules.SlotIO: "Inside/Outside", rules.SlotPlace: "Place",
	rules.SlotTime: "Time", rules.SlotAny: "Free text",
}

// patternModel is a pattern as the builder edits it: its slots and the
// text around and between them (len(seps) == len(slots)+1).
type patternModel struct {
	slots []string
	seps  []string
}

// modelOf takes a pattern apart.
func modelOf(p string) patternModel {
	m := patternModel{seps: []string{""}}
	for _, pt := range rules.ParsePattern(p) {
		if pt.Slot == "" {
			m.seps[len(m.seps)-1] += pt.Text
			continue
		}
		m.slots = append(m.slots, pt.Slot)
		m.seps = append(m.seps, "")
	}
	return m
}

// String is the pattern again.
func (m patternModel) String() string {
	var b strings.Builder
	for i, s := range m.slots {
		b.WriteString(m.seps[i] + "{" + s + "}")
	}
	b.WriteString(m.seps[len(m.seps)-1])
	return b.String()
}

// add is m with a slot added at the end, a space before it.
func (m patternModel) add(slot string) patternModel {
	m = m.clone()
	if len(m.slots) > 0 && m.seps[len(m.seps)-1] == "" {
		m.seps[len(m.seps)-1] = " "
	}
	m.slots = append(m.slots, slot)
	m.seps = append(m.seps, "")
	return m
}

// remove is m without its i'th slot and the text joining it to the
// slot before it (the first slot: to the one after it, the text before
// it kept; the last: the text after it too).
func (m patternModel) remove(i int) patternModel {
	m = m.clone()
	n := len(m.slots)
	m.slots = slices.Delete(m.slots, i, i+1)
	switch {
	case n == 1:
		m.seps = []string{m.seps[0] + m.seps[1]}
	case i == 0:
		m.seps = slices.Delete(m.seps, 1, 2)
	case i == n-1: // the text after it too, as FillPattern leaves an empty last slot out: ○{place}（{time}）
		m.seps = append(m.seps[:i], "")
	default:
		m.seps = slices.Delete(m.seps, i, i+1)
	}
	return m
}

// swap is m with slots i and j changed round, the text between kept.
func (m patternModel) swap(i, j int) patternModel {
	m = m.clone()
	m.slots[i], m.slots[j] = m.slots[j], m.slots[i]
	return m
}

func (m patternModel) clone() patternModel {
	return patternModel{slots: slices.Clone(m.slots), seps: slices.Clone(m.seps)}
}

// samplePlaces are a place, in the language, for the examples.
var samplePlaces = map[string]string{
	"en": "KITCHEN", "eo": "KUIREJO", "nl": "KEUKEN", "de": "KÜCHE", "fr": "CUISINE", "it": "CUCINA", "ru": "КУХНЯ",
	"tok": "TOMO", "zh": "客厅", "ja": "台所", "hi": "रसोई", "hi-Deva": "रसोई", "hi-Latn": "KITCHEN", "ar": "المطبخ", "he": "מטבח",
}

// sampleValues are a heading's slots filled for an example in lang: its
// first inside word and time, a place in it, scene 12.
func sampleValues(lang string, l rules.Language, form string) map[string]string {
	place := samplePlaces[lang]
	if place == "" {
		place = "KITCHEN"
	}
	v := map[string]string{rules.SlotNumber: "12", rules.SlotPlace: place, rules.SlotAny: place, rules.SlotTime: "DAY"}
	if io := l.IOWords(form); len(io) > 0 {
		v[rules.SlotIO] = io[0]
	}
	if len(l.Times) > 0 {
		v[rules.SlotTime] = l.Times[0]
	}
	return v
}

// readsAsHeading tells whether a line reads as a scene heading in l,
// after a blank line and not forced.
func readsAsHeading(l rules.Language, line string) bool {
	return !strings.HasPrefix(strings.TrimSpace(line), ".") && fountain.NewDialect(l).Heading(line)
}

// exampleOf is a pattern's example in lang and whether it reads.
func exampleOf(lang string, l rules.Language, p string) (string, bool) {
	h := rules.FillPattern(p, sampleValues(lang, l, p))
	return h, readsAsHeading(l, h)
}

// exampleText is an example as the builder shows it.
func exampleText(lang string, l rules.Language, p string) string {
	h, ok := exampleOf(lang, l, p)
	if ok {
		return "Example: " + h + "   ✓ reads as a scene heading"
	}
	return "Example: " + h + "   ✗ does not read as a scene heading"
}

// flowLayout lays its objects out in rows, as many as fit, each its
// minimum size: the chips of a pattern on a phone.
type flowLayout struct{}

func (flowLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	pad := theme.Padding()
	x, y, row := float32(0), float32(0), float32(0)
	for _, o := range objs {
		if !o.Visible() {
			continue
		}
		m := o.MinSize()
		if x > 0 && x+m.Width > size.Width {
			x, y, row = 0, y+row+pad, 0
		}
		o.Move(fyne.NewPos(x, y))
		o.Resize(m)
		x += m.Width + pad
		row = max(row, m.Height)
	}
}

func (flowLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	w, h := float32(0), float32(0)
	for _, o := range objs {
		if o.Visible() {
			m := o.MinSize()
			w, h = max(w, m.Width), max(h, m.Height)
		}
	}
	return fyne.NewSize(w, h)
}

// flowBox is a flowLayout whose height follows its width: its rows at
// the width it was last laid out at (the driver lays it out again when
// its minimum size changes).
type flowBox struct {
	widget.BaseWidget
	box   *fyne.Container
	width float32
}

func newFlowBox(objs ...fyne.CanvasObject) *flowBox {
	f := &flowBox{box: container.New(flowLayout{}, objs...)}
	f.ExtendBaseWidget(f)
	return f
}

func (f *flowBox) CreateRenderer() fyne.WidgetRenderer { return &flowRenderer{f: f} }

type flowRenderer struct{ f *flowBox }

func (r *flowRenderer) Layout(s fyne.Size) {
	r.f.box.Resize(s)
	if s.Width != r.f.width {
		r.f.width = s.Width
		r.f.Refresh()
	}
}
func (r *flowRenderer) Refresh()                     { r.f.box.Refresh() }
func (r *flowRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.f.box} }
func (r *flowRenderer) Destroy()                     {}

// MinSize is the rows' height at the last width (the widest object's
// before the first layout).
func (r *flowRenderer) MinSize() fyne.Size {
	pad := theme.Padding()
	least := flowLayout{}.MinSize(r.f.box.Objects)
	width := max(r.f.width, least.Width)
	x, y, row := float32(0), float32(0), float32(0)
	for _, o := range r.f.box.Objects {
		if !o.Visible() {
			continue
		}
		m := o.MinSize()
		if x > 0 && x+m.Width > width {
			x, y, row = 0, y+row+pad, 0
		}
		x += m.Width + pad
		row = max(row, m.Height)
	}
	return fyne.NewSize(least.Width, y+row)
}

// headingBuilder edits a language's heading patterns: list is the list
// as it is, change gets it after every edit, redraw is for an edit that
// changes the page (a slot added, a pattern moved). writeSo makes a
// pattern the one new headings are written by (NewHeading).
func headingBuilder(lang string, l rules.Language, list []string, change func([]string), redraw func(), writeSo func(string)) fyne.CanvasObject {
	list = slices.Clone(list)
	form := l.HeadingForm()
	box := container.NewVBox()
	marked := false // the form, marked once (a pattern may be in the list twice)
	isForm := func(p string) bool {
		if marked || p != form {
			return false
		}
		marked = true
		return true
	}
	if len(l.Scenes) > 0 { // Fountain's own, first: read by the scene words
		ex := widget.NewLabel(exampleText(lang, l, rules.SceneForm))
		ex.Wrapping = fyne.TextWrapWord
		items := []fyne.CanvasObject{chipRow(modelOf(rules.SceneForm), nil, nil), ex}
		items = append(items, writtenSoRow(isForm(rules.SceneForm), func() { writeSo(rules.SceneForm) }))
		box.Add(widget.NewCard("", "Fountain's own: a scene word first, always read", container.NewVBox(items...)))
	}
	for i := range list {
		i := i
		m := modelOf(list[i])
		ex := widget.NewLabel(exampleText(lang, l, list[i]))
		ex.Wrapping = fyne.TextWrapWord
		edit := func(n patternModel, structural bool) {
			m = n
			list[i] = n.String()
			change(list)
			if structural {
				redraw()
				return
			}
			ex.SetText(exampleText(lang, l, list[i]))
		}
		chips := chipRow(m, func() patternModel { return m }, edit)
		slotMenu := widget.NewButtonWithIcon("Slot", theme.ContentAddIcon(), nil)
		slotMenu.OnTapped = func() {
			var items []*fyne.MenuItem
			for _, s := range rules.Slots {
				items = append(items, fyne.NewMenuItem(slotNames[s], func() { edit(m.add(s), true) }))
			}
			showMenuUnder(slotMenu, items)
		}
		slotMenu.Importance = widget.LowImportance
		up := widget.NewButtonWithIcon("", theme.MoveUpIcon(), func() {
			list[i-1], list[i] = list[i], list[i-1]
			change(list)
			redraw()
		})
		fyne.SetAccessibleLabel(up, "Move the pattern up")
		down := widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() {
			list[i+1], list[i] = list[i], list[i+1]
			change(list)
			redraw()
		})
		fyne.SetAccessibleLabel(down, "Move the pattern down")
		rm := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
			list = slices.Delete(list, i, i+1)
			change(list)
			redraw()
		})
		fyne.SetAccessibleLabel(rm, "Remove the pattern")
		if i == 0 {
			up.Disable()
		}
		if i == len(list)-1 {
			down.Disable()
		}
		tools := container.NewHBox(slotMenu, up, down, rm)
		title := "Pattern " + strconv.Itoa(i+1)
		if i == 0 && len(l.Scenes) == 0 {
			title += ": the language's first"
		}
		box.Add(widget.NewCard("", title, container.NewVBox(chips, ex,
			writtenSoRow(isForm(list[i]), func() { writeSo(list[i]) }), tools)))
	}
	add := widget.NewButtonWithIcon("Add Pattern", theme.ContentAddIcon(), func() {
		list = append(list, "{n}. {io} {place} - {time}")
		change(list)
		redraw()
	})
	add.Importance = widget.LowImportance
	box.Add(container.NewHBox(add))
	return box
}

// writtenSoRow says a pattern is the one new headings are written by,
// or offers to make it so.
func writtenSoRow(is bool, make func()) fyne.CanvasObject {
	if is {
		l := widget.NewLabelWithStyle("✎ New scenes are written so (Insert > New Scene…)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		l.Importance = widget.SuccessImportance
		return l
	}
	b := widget.NewButtonWithIcon("Write New Scenes So", theme.DocumentCreateIcon(), make)
	b.Importance = widget.LowImportance
	return container.NewHBox(b)
}

// chipRow is a pattern's slots as chips with the text between them as
// small entries; cur and edit nil for one shown, not edited.
func chipRow(m patternModel, cur func() patternModel, edit func(patternModel, bool)) fyne.CanvasObject {
	var objs []fyne.CanvasObject
	sep := func(k int) {
		if edit == nil {
			if t := m.seps[k]; t != "" {
				l := widget.NewLabel(strings.ReplaceAll(t, " ", "␣"))
				l.Importance = widget.LowImportance
				objs = append(objs, l)
			}
			return
		}
		en := widget.NewEntry()
		en.SetText(m.seps[k])
		en.SetPlaceHolder("·")
		en.OnChanged = func(s string) {
			n := cur().clone()
			n.seps[k] = s
			edit(n, false)
		}
		fyne.SetAccessibleLabel(en, "Text between the slots")
		objs = append(objs, container.NewGridWrap(fyne.NewSize(64, en.MinSize().Height), en))
	}
	for i, s := range m.slots {
		sep(i)
		chip := widget.NewButton(slotNames[s], nil)
		chip.Importance = widget.HighImportance
		if edit == nil {
			chip.Importance = widget.MediumImportance
			chip.Disable()
		} else {
			i := i
			chip.OnTapped = func() {
				n := cur()
				var items []*fyne.MenuItem
				if i > 0 {
					items = append(items, fyne.NewMenuItem("Move Left", func() { edit(n.swap(i, i-1), true) }))
				}
				if i < len(n.slots)-1 {
					items = append(items, fyne.NewMenuItem("Move Right", func() { edit(n.swap(i, i+1), true) }))
				}
				for _, other := range rules.Slots {
					if other != n.slots[i] {
						items = append(items, fyne.NewMenuItem("Make It "+slotNames[other], func() {
							c := n.clone()
							c.slots[i] = other
							edit(c, true)
						}))
					}
				}
				items = append(items, fyne.NewMenuItemSeparator(), fyne.NewMenuItem("Remove", func() { edit(n.remove(i), true) }))
				showMenuUnder(chip, items)
			}
		}
		objs = append(objs, chip)
	}
	sep(len(m.slots))
	return newFlowBox(objs...)
}

// showMenuUnder pops a menu up under an object.
func showMenuUnder(at fyne.CanvasObject, items []*fyne.MenuItem) {
	c := fyne.CurrentApp().Driver().CanvasForObject(at)
	if c == nil {
		return
	}
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(at).Add(fyne.NewPos(0, at.Size().Height))
	widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", items...), c, pos)
}
