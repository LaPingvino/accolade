// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/rules"
)

// The format editor (LOOP.md, Joop 2026-11-22): a format file (a preset's
// file: its Name, the format it starts from, the elements that differ,
// Accolade's toolbar buttons) as a form. Format: its name, description,
// paper and Base. Elements: each element as the format has it, Base's
// greyed until changed here (an element is whole: changing one copies
// Base's to start from), with a sample Lexington draws. Buttons: the
// toolbar's buttons, in order, to move up and down. TOML and Help as for
// every .toml.

// openFormatEditor opens the editor of the format file doc.
func openFormatEditor(app *Application, parent fyne.Window, doc *tomlDoc) *tomlEditor {
	e := newTOMLEditor(app, doc)
	e.page("Format", e.formatPage)
	e.page("Elements", func() fyne.CanvasObject {
		s := e.doc.spec()
		return e.elementsPane("format", nil, nil, []string{"Elements"}, s.Elements, e.formatBase(), e.formatBaseName(), func() (rules.Set, bool) {
			p, ok := e.formatResolved()
			return p.Elements, ok
		}, e.formatPageSize())
	})
	e.page("Buttons", e.buttonsPage)
	e.show(parent)
	return e
}

// formatBaseKey is the format the file starts from.
func (e *tomlEditor) formatBaseKey() string {
	if b := e.doc.spec().Base; b != "" {
		return b
	}
	return rules.ScreenplayPreset
}

// baseFormat is the format the file starts from: the machine's; for a
// file of a format of its own name (a changed copy of a built-in one)
// the one under it: Accolade's under one's own, Lexington's under
// Accolade's.
func (e *tomlEditor) baseFormat() (rules.Preset, bool) {
	base := e.formatBaseKey()
	if base == e.doc.key {
		if absPath(filepath.Dir(e.doc.path)) == absPath(userFormatDir()) {
			list, _ := rules.PresetsIn(builtinFormatDir())
			for _, p := range list {
				if p.Key == base {
					return p, true
				}
			}
		}
		return rules.GetPreset(base)
	}
	for _, p := range scriptFormats() {
		if p.Key == base {
			return p, true
		}
	}
	return rules.GetPreset(base)
}

// formatBase is the elements the file starts from.
func (e *tomlEditor) formatBase() rules.Set {
	p, _ := e.baseFormat()
	if p.Elements == nil {
		return rules.Default
	}
	return p.Elements
}

func (e *tomlEditor) formatBaseName() string {
	p, ok := e.baseFormat()
	if !ok {
		return e.formatBaseKey()
	}
	return p.Name
}

// formatResolved is the format as Lexington reads the file now.
func (e *tomlEditor) formatResolved() (rules.Preset, bool) {
	base, ok := e.baseFormat()
	if !ok {
		return rules.Preset{}, false
	}
	list, err := rules.WithPresetFiles([]rules.Preset{base}, map[string][]byte{e.doc.key: []byte(e.doc.text)})
	if err != nil {
		return rules.Preset{}, false
	}
	for _, p := range list {
		if p.Key == e.doc.key {
			return p, true
		}
	}
	return rules.Preset{}, false
}

func (e *tomlEditor) formatPageSize() string {
	p, _ := e.formatResolved()
	return p.Page
}

// formatPage: the name, description, paper and base.
func (e *tomlEditor) formatPage() fyne.CanvasObject {
	s := e.doc.spec()
	base, _ := e.baseFormat()
	fields := rules.FormatFileFields()
	text := func(key, value, inherited string, multi bool) fyne.CanvasObject {
		en := widget.NewEntry()
		if multi {
			en = widget.NewMultiLineEntry()
			en.Wrapping = fyne.TextWrapWord
			en.SetMinRowsVisible(3)
		}
		en.SetText(value)
		en.SetPlaceHolder(inherited)
		en.OnChanged = func(v string) {
			if strings.TrimSpace(v) == "" {
				e.do(drop(key))
			} else {
				e.do(set(v, key))
			}
		}
		return fieldBox(key, docOf(fields, key), en)
	}
	pages := []string{"As the base", "letter", "a4", "a5"}
	page := widget.NewSelect(pages, nil)
	if s.Page == "" {
		page.SetSelected(pages[0])
	} else {
		page.SetSelected(s.Page)
	}
	page.OnChanged = func(v string) {
		if v == pages[0] {
			e.do(drop("Page"))
		} else {
			e.do(set(v, "Page"))
		}
	}
	var keys, names []string
	for _, p := range rules.Presets() {
		keys, names = append(keys, p.Key), append(names, p.Name+" ("+p.Key+")")
	}
	for _, p := range scriptFormats() {
		if !slices.Contains(keys, p.Key) && p.Key != e.doc.key {
			keys, names = append(keys, p.Key), append(names, p.Name+" ("+p.Key+")")
		}
	}
	baseSel := widget.NewSelect(names, nil)
	if i := slices.Index(keys, e.formatBaseKey()); i >= 0 {
		baseSel.SetSelected(names[i])
	} else {
		baseSel.PlaceHolder = e.formatBaseKey() + " (not found)"
	}
	baseSel.OnChanged = func(v string) {
		k := keys[slices.Index(names, v)]
		if k == e.formatBaseKey() {
			return
		}
		if k == rules.ScreenplayPreset && s.Base == "" {
			return
		}
		e.doAndRedraw(set(k, "Base"))
	}
	inheritedPage := base.Page
	if inheritedPage == "" {
		inheritedPage = "the user's paper, else Letter"
	}
	return pagePane(nil,
		text("Name", s.Name, base.Name, false),
		text("Description", s.Description, base.Description, true),
		fieldBox("Page", docOf(fields, "Page")+" Its base's: "+inheritedPage+".", page),
		fieldBox("Base", docOf(fields, "Base"), baseSel),
	)
}

// elementsPane edits the elements at prefix (a format's Elements, a
// set's): one element at a time, chosen at the top. own are those the
// file has; base those used where it has none (nil: an element not
// there is not printed, as in a set of lexington.toml); resolved the
// whole set for the sample.
func (e *tomlEditor) elementsPane(page string, top, intro fyne.CanvasObject, prefix []string, own, base rules.Set, baseName string, resolved func() (rules.Set, bool), paper string) fyne.CanvasObject {
	elements := rules.ElementKeys()
	var labels []string
	for _, f := range elements {
		l := f.Key
		switch _, mine := own[f.Key]; {
		case mine:
			l += " (changed here)"
		case base == nil:
			l += " (not printed)"
		}
		labels = append(labels, l)
	}
	cur := e.sel[page+"-element"]
	if cur == "" {
		cur = rules.KeyScene.String()
	}
	i := slices.IndexFunc(elements, func(f rules.Field) bool { return f.Key == cur })
	if i < 0 {
		i, cur = 0, elements[0].Key
	}
	pick := widget.NewSelect(labels, nil)
	pick.SetSelected(labels[i])
	pick.OnChanged = func(v string) {
		e.sel[page+"-element"] = elements[slices.Index(labels, v)].Key
		e.redraw()
	}
	fyne.SetAccessibleLabel(pick, "Element")
	path := append(slices.Clone(prefix), cur)
	doc := elements[i].Doc
	var body fyne.CanvasObject
	f, mine := own[cur]
	switch {
	case mine:
		undo := "Use the Base's"
		if base == nil {
			undo = "Leave It Out (not printed)"
		}
		b := widget.NewButtonWithIcon(undo, theme.ContentUndoIcon(), func() { e.doAndRedraw(drop(path...)) })
		b.Importance = widget.LowImportance
		body = container.NewVBox(e.formatControls(path, cur, f), container.NewHBox(b))
	case base != nil:
		bf, ok := base[cur]
		what := formatSummary(cur, bf)
		if !ok {
			what = "Not printed (" + baseName + " has no rule for it)."
		}
		b := widget.NewButtonWithIcon("Change It Here", theme.DocumentCreateIcon(), func() {
			e.overrideElement(path, bf)
		})
		body = container.NewVBox(widget.NewLabel("As "+baseName+" has it:"), greyed(what), container.NewHBox(b))
	default:
		b := widget.NewButtonWithIcon("Print It", theme.ContentAddIcon(), func() {
			from, ok := rules.Default[cur]
			if !ok {
				from = rules.Default[rules.KeyAction.String()]
			}
			e.overrideElement(path, from)
		})
		body = container.NewVBox(greyed("Not printed: the set has no rule for it."), container.NewHBox(b))
	}
	sample := newFormatSample(resolved, paper)
	sample.titlePage = cur == rules.KeyTitle.String() || cur == rules.KeyMeta.String()
	e.sample = sample
	var form fyne.CanvasObject = fieldBox(cur, doc, body)
	if intro != nil {
		form = container.NewVBox(intro, widget.NewSeparator(), form)
	}
	scroll := container.NewVScroll(container.NewPadded(form))
	head := pickRow("Element:", pick)
	if top != nil {
		head = container.NewVBox(top, head)
	}
	return container.NewBorder(container.NewPadded(head), nil, nil, nil,
		container.New(&besideOrUnder{side: 400, under: 280}, scroll, sample.object()))
}

// overrideElement writes an element into the file whole, as from has it.
func (e *tomlEditor) overrideElement(path []string, from rules.Format) {
	err := e.doc.addTable(path, formatMap(from), formatOrder)
	e.changed(err)
	e.redraw()
}

// formatOrder is the order an element's keys are written in.
var formatOrder = []string{"Left", "Right", "Font", "Size", "Style", "Align", "Hide", "Prefix", "Postfix"}

// formatMap is a format's fields that are set.
func formatMap(f rules.Format) map[string]any {
	m := map[string]any{}
	if f.Left != 0 {
		m["Left"] = f.Left
	}
	if f.Right != 0 {
		m["Right"] = f.Right
	}
	if f.Font != "" {
		m["Font"] = f.Font
	}
	if f.Size != 0 {
		m["Size"] = f.Size
	}
	if f.Style != "" {
		m["Style"] = f.Style
	}
	if f.Align != "" {
		m["Align"] = f.Align
	}
	if f.Hide {
		m["Hide"] = true
	}
	if f.Prefix != "" {
		m["Prefix"] = f.Prefix
	}
	if f.Postfix != "" {
		m["Postfix"] = f.Postfix
	}
	return m
}

// formatSummary is a format in a line.
func formatSummary(key string, f rules.Format) string {
	if rules.IsSwitch(key) {
		if f.Hide {
			return "Off"
		}
		return "On: " + cmp.Or(f.Style, "all")
	}
	var parts []string
	parts = append(parts, fmt.Sprintf("left %s in, right %s in", trimFloat(f.Left), trimFloat(f.Right)))
	if f.Font != "" {
		parts = append(parts, f.Font)
	}
	if f.Size != 0 {
		parts = append(parts, trimFloat(f.Size)+" pt")
	}
	if f.Style != "" {
		parts = append(parts, styleWords(f.Style))
	}
	switch f.Align {
	case "C":
		parts = append(parts, "centred")
	case "R":
		parts = append(parts, "right")
	}
	if f.Prefix != "" || f.Postfix != "" {
		parts = append(parts, fmt.Sprintf("%q…%q", f.Prefix, f.Postfix))
	}
	if f.Hide {
		parts = append(parts, "not printed")
	}
	return strings.Join(parts, ", ")
}

func trimFloat(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// styleWords is a style's letters in words.
func styleWords(s string) string {
	var w []string
	for _, c := range []struct{ l, word string }{{"b", "bold"}, {"i", "italic"}, {"u", "underlined"}} {
		if strings.Contains(strings.ToLower(s), c.l) {
			w = append(w, c.word)
		}
	}
	return strings.Join(w, ", ")
}

// fontChoicesLex are the fonts Lexington has built in by name.
var fontChoicesLex = []string{"CourierPrime", "Helvetica", "Times"}

// formatControls are an element's fields, each changing its key in the
// file as it is set (a field set back to nothing is taken out).
func (e *tomlEditor) formatControls(path []string, key string, f rules.Format) fyne.CanvasObject {
	fields := rules.FormatFields()
	at := func(k string) []string { return append(slices.Clone(path), k) }
	setNum := func(k string) func(float64) {
		return func(v float64) {
			if v == 0 {
				e.do(drop(at(k)...))
			} else {
				e.do(set(v, at(k)...))
			}
		}
	}
	setText := func(k string) func(string) {
		return func(v string) {
			if v == "" {
				e.do(drop(at(k)...))
			} else {
				e.do(set(v, at(k)...))
			}
		}
	}
	hideLabel := "Not printed"
	if rules.IsSwitch(key) {
		hideLabel = "Off"
	}
	hide := widget.NewCheck(hideLabel, func(on bool) {
		if on {
			e.do(set(true, at("Hide")...))
		} else {
			e.do(drop(at("Hide")...))
		}
	})
	hide.Checked = f.Hide
	if rules.IsSwitch(key) {
		return container.NewVBox(fieldBox("Style", docOf(fields, "Style"), e.switchStyle(key, f.Style, setText("Style"))), hide)
	}
	left := numberEntry(f.Left, 0, 8, "0", setNum("Left"))
	right := numberEntry(f.Right, 0, 8, "0", setNum("Right"))
	size := numberEntry(f.Size, 0, 72, "12", setNum("Size"))
	font := widget.NewSelectEntry(fontChoicesLex)
	font.SetText(f.Font)
	font.SetPlaceHolder("CourierPrime")
	font.OnChanged = setText("Font")
	style := map[string]*widget.Check{}
	styleBox := container.NewHBox()
	for _, c := range []struct{ l, word string }{{"b", "Bold"}, {"i", "Italic"}, {"u", "Underlined"}} {
		ch := widget.NewCheck(c.word, nil)
		ch.Checked = strings.Contains(strings.ToLower(f.Style), c.l)
		style[c.l] = ch
		styleBox.Add(ch)
	}
	for _, ch := range style {
		ch.OnChanged = func(bool) {
			s := ""
			for _, l := range []string{"b", "i", "u"} {
				if style[l].Checked {
					s += l
				}
			}
			setText("Style")(s)
		}
	}
	aligns := []string{"Left", "Centre", "Right"}
	align := widget.NewRadioGroup(aligns, nil)
	align.Horizontal = true
	align.Selected = map[string]string{"": "Left", "L": "Left", "C": "Centre", "R": "Right"}[f.Align]
	align.OnChanged = func(v string) {
		setText("Align")(map[string]string{"Left": "", "Centre": "C", "Right": "R"}[v])
	}
	prefix := widget.NewEntry()
	prefix.SetText(f.Prefix)
	prefix.OnChanged = setText("Prefix")
	postfix := widget.NewEntry()
	postfix.SetText(f.Postfix)
	postfix.OnChanged = setText("Postfix")
	labelled := func(name string, o fyne.CanvasObject) fyne.CanvasObject {
		l := widget.NewLabel(name)
		return container.NewBorder(nil, nil, l, nil, o)
	}
	return container.NewVBox(
		container.NewGridWithColumns(2, labelled("Left (in)", left), labelled("Right (in)", right)),
		container.NewGridWithColumns(2, labelled("Font", font), labelled("Size (pt)", size)),
		styleBox,
		align,
		container.NewGridWithColumns(2, labelled("Before", prefix), labelled("After", postfix)),
		hide,
	)
}

// switchStyle is the Style of notecolors (how colors show) or pagination
// (what is on).
func (e *tomlEditor) switchStyle(key, style string, change func(string)) fyne.CanvasObject {
	if key == rules.KeyPagination.String() {
		p, err := rules.ParsePagination(style)
		if err != nil {
			p = rules.AllPagination
		}
		keep := widget.NewCheck("Keep (headings and cues go over with what follows)", nil)
		widows := widget.NewCheck("Widows (two lines of a paragraph on each page)", nil)
		more := widget.NewCheck("More ((MORE) and CONT'D in a speech over two pages)", nil)
		keep.Checked, widows.Checked, more.Checked = p.Keep, p.Widows, p.More
		upd := func(bool) {
			q := rules.Pagination{Keep: keep.Checked, Widows: widows.Checked, More: more.Checked}
			s := q.String()
			if q == rules.AllPagination {
				s = ""
			}
			change(s)
		}
		keep.OnChanged, widows.OnChanged, more.OnChanged = upd, upd, upd
		return container.NewVBox(keep, widows, more)
	}
	sel := widget.NewSelectEntry([]string{"line", "line+tint", "line+patterns", "marker", "marker+block", "background", "background+block", "text", "text+block"})
	sel.SetText(style)
	sel.SetPlaceHolder("line")
	sel.Validator = func(s string) error {
		_, err := checkSwitchStyle(key, s)
		return err
	}
	sel.OnChanged = func(s string) {
		if _, err := checkSwitchStyle(key, s); err == nil {
			change(s)
		}
	}
	return sel
}

func checkSwitchStyle(key, s string) (string, error) {
	return s, checkFormat(key, rules.Format{Style: s})
}

// buttonsPage: the toolbar's buttons of the format, in order.
func (e *tomlEditor) buttonsPage() fyne.CanvasObject {
	buttons := e.doc.spec().Accolade.Buttons
	path := []string{"Accolade", "Buttons"}
	box := container.NewVBox()
	n := len(buttons)
	for i, b := range buttons {
		i := i
		setKeyIn := func(k string) func(string) {
			return func(v string) {
				value := ""
				if v != "" {
					value = tomlValue(v)
				}
				err := e.doc.editText(func(s string) (string, error) { return setInBlock(s, path, i, k, value) },
					func(m map[string]any) map[string]any { return withButton(m, i, k, v) })
				e.changed(err)
			}
		}
		label := widget.NewEntry()
		label.SetText(b.Label)
		label.SetPlaceHolder("Label")
		label.OnChanged = setKeyIn("Label")
		insert := widget.NewEntry()
		insert.SetText(escapeLines(b.Insert))
		insert.SetPlaceHolder(`Insert ({n} numbers it, \n a new line)`)
		insert.OnChanged = func(v string) { setKeyIn("Insert")(unescapeLines(v)) }
		restart := widget.NewEntry()
		restart.SetText(b.Restart)
		restart.SetPlaceHolder("Counts again after a line starting with…")
		restart.OnChanged = setKeyIn("Restart")
		line := widget.NewCheck("On the next line, not a paragraph of its own", func(on bool) {
			value := ""
			if on {
				value = "true"
			}
			err := e.doc.editText(func(s string) (string, error) { return setInBlock(s, path, i, "Line", value) },
				func(m map[string]any) map[string]any {
					if on {
						return withButton(m, i, "Line", true)
					}
					return withButton(m, i, "Line", "")
				})
			e.changed(err)
		})
		line.Checked = b.Line
		move := func(to int) func() {
			return func() {
				err := e.doc.editText(func(s string) (string, error) { return moveBlock(s, path, i, to) },
					func(m map[string]any) map[string]any { return withButtonMoved(m, i, to) })
				e.changed(err)
				e.redraw()
			}
		}
		up := widget.NewButtonWithIcon("", theme.MoveUpIcon(), move(i-1))
		fyne.SetAccessibleLabel(up, "Move up")
		down := widget.NewButtonWithIcon("", theme.MoveDownIcon(), move(i+1))
		fyne.SetAccessibleLabel(down, "Move down")
		if i == 0 {
			up.Disable()
		}
		if i == n-1 {
			down.Disable()
		}
		rm := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
			err := e.doc.editText(func(s string) (string, error) { return dropBlock(s, path, i) },
				func(m map[string]any) map[string]any { return withButtonDropped(m, i) })
			e.changed(err)
			e.redraw()
		})
		fyne.SetAccessibleLabel(rm, "Remove the button")
		head := container.NewBorder(nil, nil, widget.NewLabelWithStyle(strconv.Itoa(i+1)+".", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			container.NewHBox(up, down, rm), label)
		box.Add(widget.NewCard("", "", container.NewVBox(head, insert, restart, line)))
	}
	add := widget.NewButtonWithIcon("Add a Button", theme.ContentAddIcon(), func() {
		before, err := tomlMap(e.doc.text)
		if err != nil {
			e.changed(err)
			return
		}
		m := map[string]any{"Label": "New", "Insert": "NEW "}
		err = e.doc.keep(addBlock(e.doc.text, path, m, []string{"Label", "Insert", "Restart", "Line"}), withButtonAdded(before, m))
		e.changed(err)
		e.redraw()
	})
	about := widget.NewLabel("The toolbar's buttons for scripts in this format, each inserting what it says. {n} numbers it ({N} in words), counting again after a line starting with what Counts again says; \\n is a new line. Lexington skips them.")
	about.Wrapping = fyne.TextWrapWord
	about.TextStyle = fyne.TextStyle{Italic: true}
	return pagePane(nil, about, box, container.NewHBox(add))
}

// The buttons in the TOML read as maps: what the text is to read back as
// after a change to one.

func buttonsOf(m map[string]any) []map[string]any {
	a, _ := m["Accolade"].(map[string]any)
	b, _ := a["Buttons"].([]map[string]any)
	out := make([]map[string]any, len(b))
	for i, x := range b {
		c := map[string]any{}
		for k, v := range x {
			c[k] = v
		}
		out[i] = c
	}
	return out
}

func withButtons(m map[string]any, b []map[string]any) map[string]any {
	if len(b) == 0 {
		if _, ok := m["Accolade"].(map[string]any); !ok {
			return m
		}
		return withAt(m, []string{"Accolade", "Buttons"}, nil)
	}
	return withAt(m, []string{"Accolade", "Buttons"}, b)
}

// withButton is m with button i's key k set to v ("" takes it out).
func withButton(m map[string]any, i int, k string, v any) map[string]any {
	b := buttonsOf(m)
	if s, ok := v.(string); ok && s == "" {
		delete(b[i], k)
	} else {
		b[i][k] = v
	}
	return withButtons(m, b)
}

func withButtonMoved(m map[string]any, i, j int) map[string]any {
	b := buttonsOf(m)
	x := b[i]
	b = slices.Delete(b, i, i+1)
	return withButtons(m, slices.Insert(b, j, x))
}

func withButtonDropped(m map[string]any, i int) map[string]any {
	b := buttonsOf(m)
	return withButtons(m, slices.Delete(b, i, i+1))
}

func withButtonAdded(m map[string]any, x map[string]any) map[string]any {
	return withButtons(m, append(buttonsOf(m), x))
}

// escapeLines shows a button's new lines as \n, on one line.
func escapeLines(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), "\n", `\n`)
}

// unescapeLines is escapeLines the other way.
func unescapeLines(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
				i++
				continue
			case '\\':
				b.WriteByte('\\')
				i++
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// askName asks for a name, then does with it.
func askName(parent fyne.Window, title, label, placeholder string, then func(string)) {
	en := widget.NewEntry()
	en.SetPlaceHolder(placeholder)
	dialog.ShowForm(title, "OK", "Cancel", []*widget.FormItem{widget.NewFormItem(label, en)}, func(ok bool) {
		if ok && strings.TrimSpace(en.Text) != "" {
			then(strings.TrimSpace(en.Text))
		}
	}, parent)
}

// besideOrUnder lays out a form and its sample: the sample beside the
// form on a wide window, under it on a narrow one (a phone).
type besideOrUnder struct{ side, under float32 }

func (l *besideOrUnder) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(280, 320) }

func (l *besideOrUnder) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	if len(objs) < 2 {
		return
	}
	form, sample := objs[0], objs[1]
	pad := theme.Padding()
	if size.Width >= 2*l.side {
		form.Move(fyne.NewPos(0, 0))
		form.Resize(fyne.NewSize(size.Width-l.side-pad, size.Height))
		sample.Move(fyne.NewPos(size.Width-l.side, 0))
		sample.Resize(fyne.NewSize(l.side, size.Height))
		return
	}
	h := min(l.under, size.Height*2/5)
	form.Move(fyne.NewPos(0, 0))
	form.Resize(fyne.NewSize(size.Width, size.Height-h-pad))
	sample.Move(fyne.NewPos(0, size.Height-h))
	sample.Resize(fyne.NewSize(size.Width, h))
}
