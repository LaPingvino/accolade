package main

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The Characters pane (View > Characters, a tab of the side dock): the
// script's characters, the most present first. A row says how often they
// speak and where they start; tapped, the script goes to their next cue
// (again: the one after), and › opens them: their scenes (tapped, the
// scene), the note and fields of the character list (characters_block.go:
// written into the script as they are typed), the learned notes' other
// names, and the actions, Rename Everywhere…, Merge Into…, Add Alias…,
// Not a Character. As Production a row has the cast number (Lexington's
// cast list, as the reports number it) and the entry an actor.
//
// In a Markdown document the pane lists the names the document reads as
// someone's (storyread, as a story is read), each tapped to where it next
// stands; Rename Everywhere renames one. The list and its notes are a
// screenplay's.
//
// The pane reads the script once typing pauses, and only while in sight;
// the cast numbers, which lay the script out as printed, in the
// background after that.

type charactersPane struct {
	w      *MainWindow
	of     docKey // the script it reads (state.go)
	box    fyne.CanvasObject
	filter *widget.Entry
	list   *widget.List
	none   *widget.Label
	count  *widget.Label
	body   *fyne.Container // the list, or a character opened
	top    *fyne.Container // the filter and the count, over the list

	model   *charModel
	modelOf string // the text the model was read from
	cast    map[string]int
	castOf  string
	// castBusy: the cast numbers are being read; castAgain: read them again after
	castBusy, castAgain bool
	shown               []int // the rows: indexes into model.chars, filtered
	// rowH, aliasH are a row's height, and with a line of other names
	rowH, aliasH float32
	lit          string // the key of the character tapped last
	open         string // the key of the character opened ("": the list)
	timer        *uiTimer

	detail struct {
		box      *fyne.Container
		name     *widget.Label
		stats    *widget.Label
		aliases  *widget.Label
		actions  *fyne.Container
		note     *widget.Entry
		photo    *canvas.Image // the cast photo, larger
		fields   [fieldCount]*widget.Entry
		fieldRow [fieldCount]*widget.FormItem
		noteRow  *widget.FormItem
		form     *widget.Form
		scenes   *fyne.Container
		listed   *widget.Label
		updating bool
	}
}

func (w *MainWindow) charactersPane() *charactersPane {
	if w.charsPane == nil {
		w.charsPane = newCharactersPane(w)
	}
	return w.charsPane
}

func newCharactersPane(w *MainWindow) *charactersPane {
	p := &charactersPane{w: w}
	p.filter = widget.NewEntry()
	p.filter.SetPlaceHolder("Filter characters")
	fyne.SetAccessibleLabel(p.filter, "Filter characters")
	p.filter.OnChanged = func(string) { p.refilter() }
	p.count = widget.NewLabel("")
	p.count.SizeName = theme.SizeNameCaptionText
	p.count.Importance = widget.LowImportance
	p.count.Truncation = fyne.TextTruncateEllipsis
	p.none = widget.NewLabel("")
	p.none.Wrapping = fyne.TextWrapWord
	p.none.Importance = widget.LowImportance
	p.list = widget.NewList(
		func() int { return len(p.shown) },
		func() fyne.CanvasObject { return newCharRow(p) },
		func(id widget.ListItemID, o fyne.CanvasObject) {
			if id < len(p.shown) {
				o.(*charRow).show(p, p.shown[id])
			}
		},
	)
	p.list.OnSelected = func(id widget.ListItemID) {
		p.list.UnselectAll()
		if id < len(p.shown) {
			p.next(p.model.chars[p.shown[id]].Key, 1)
		}
	}
	p.makeDetail()
	width := canvas.NewRectangle(color.Transparent)
	width.SetMinSize(fyne.NewSize(260, 0))
	p.body = container.NewStack(p.list, p.none)
	p.top = container.NewVBox(p.filter, p.count)
	p.box = container.NewStack(width, container.NewBorder(p.top, nil, nil, nil, p.body))
	return p
}

// focusTarget is what takes the keyboard in the pane: the filter, or
// with a character opened their note.
func (p *charactersPane) focusTarget() fyne.Focusable {
	switch {
	case p.open == "":
		return p.filter
	case p.model != nil && !p.model.markdown:
		return p.detail.note
	}
	return nil // a name opened in Markdown: its buttons, by Tab
}

// inSight reports whether the pane is in sight.
func (p *charactersPane) inSight() bool { return p.w.paneInSight("characters") }

// update reads the script again once typing pauses (the dock's edited:
// each key only puts the reading off).
func (p *charactersPane) update(text string) {
	if p.modelOf != "" && sameString(p.modelOf, text) {
		return
	}
	p.w.debounce(&p.timer, followPause, func() {
		if p.inSight() {
			p.read()
		}
	})
}

// read reads the script (in the background: a Markdown document is read
// as a story) and shows it.
func (p *charactersPane) read() {
	w := p.w
	p.of = w.scriptKey()
	text := w.textEditor.Text()
	if p.model != nil && sameString(p.modelOf, text) && p.model.markdown == w.markdown {
		p.refresh()
		p.readCast(text)
		return
	}
	markdown, starts, lang := w.markdown, w.sceneStarts(), w.language
	inBackground(func() *charModel {
		if markdown {
			l, _ := readingLanguage(text, lang)
			return markdownChars(text, l)
		}
		return screenplayChars(text, starts)
	}, func(m *charModel) {
		if !sameString(w.textEditor.Text(), text) {
			p.update(w.textEditor.Text()) // typed since: read again after the pause
			return
		}
		if p.model == nil || p.model.markdown != m.markdown { // another document
			p.lit, p.cast, p.castOf = "", nil, ""
		}
		p.model, p.modelOf = m, text
		p.refresh()
		p.readCast(text)
	})
}

// readCast reads the cast numbers (Production's: the script laid out as
// printed, in the background) for the text, if they are not known.
func (p *charactersPane) readCast(text string) {
	w := p.w
	if w.markdown || w.writerView() || sameString(p.castOf, text) {
		return
	}
	if p.castBusy { // one layout at a time: the text as it is then, after it
		p.castAgain = true
		return
	}
	p.castBusy = true
	job := w.layoutJob()
	inBackground(func() map[string]int {
		r, err := scriptReport(text, job)
		if err != nil {
			return nil
		}
		out := map[string]int{}
		for _, c := range r.Cast() {
			out[charKey(c.Name)] = c.Number
		}
		return out
	}, func(cast map[string]int) {
		p.castBusy = false
		if p.castAgain {
			p.castAgain = false
			p.readCast(w.textEditor.Text())
		}
		if cast == nil || !sameString(w.textEditor.Text(), text) {
			return
		}
		p.cast, p.castOf = cast, text
		p.list.Refresh()
		if p.open != "" {
			p.showDetail()
		}
	})
}

// castOfKey is the character's cast number as Production sees it (0: none
// or not known, or as Writer).
func (p *charactersPane) castOfKey(key string) int {
	if p.w.writerView() || p.cast == nil || p.model == nil || p.model.markdown {
		return 0
	}
	return p.cast[key]
}

// refresh shows the model: the list filtered, or the character opened.
func (p *charactersPane) refresh() {
	if p.model == nil {
		return
	}
	p.refilter()
	if p.open != "" {
		if p.model.find(p.open) < 0 {
			p.closeDetail()
			return
		}
		p.showDetail()
	}
}

// refilter picks the rows the filter lets through, the not characters
// last.
func (p *charactersPane) refilter() {
	if p.model == nil {
		return
	}
	q := strings.ToLower(strings.TrimSpace(p.filter.Text))
	var rows, not []int
	for i, c := range p.model.chars {
		if q != "" && !strings.Contains(strings.ToLower(c.Name+"\n"+strings.Join(c.Aliases, "\n")+"\n"+p.noteOf(c)), q) {
			continue
		}
		if c.Not {
			not = append(not, i)
		} else {
			rows = append(rows, i)
		}
	}
	p.shown = append(rows, not...)
	// a row with other names is a line taller
	if p.rowH == 0 {
		r := newCharRow(p)
		p.rowH = r.MinSize().Height
		r.aliases.Show()
		p.aliasH = r.MinSize().Height
	}
	for id, i := range p.shown {
		h := p.rowH
		if len(p.model.chars[i].Aliases) > 0 {
			h = p.aliasH
		}
		p.list.SetItemHeight(id, h)
	}
	n, listed := 0, 0
	for _, c := range p.model.chars {
		if !c.Not {
			n++
			if c.Speeches == 0 && !p.model.markdown {
				listed++
			}
		}
	}
	switch {
	case p.model.markdown:
		p.count.SetText(plural(n, "name") + " in this document")
	case listed > 0:
		p.count.SetText(plural(n, "character") + " · " + fmt.Sprint(listed) + " in the list only")
	default:
		p.count.SetText(plural(n, "character"))
	}
	switch {
	case len(p.model.chars) == 0 && p.model.markdown:
		p.none.SetText("No names read as someone's yet.")
	case len(p.model.chars) == 0:
		p.none.SetText("No one speaks yet. A character cue (a name in capitals before dialogue) puts them here.")
	case len(p.shown) == 0:
		p.none.SetText("No character matches the filter.")
	}
	if len(p.shown) == 0 {
		p.none.Show()
		p.list.Hide()
	} else {
		p.none.Hide()
		p.list.Show()
	}
	p.list.Refresh()
}

// noteOf is the character's note in the list ("" none).
func (p *charactersPane) noteOf(c charInfo) string {
	if c.Entry < 0 || c.Entry >= len(p.model.list.entries) {
		return ""
	}
	e := p.model.list.entries[c.Entry]
	return strings.Join(append([]string{e.Note}, e.Fields[:]...), "\n")
}

// charStats is a character's line under its name.
func (p *charactersPane) charStats(c charInfo) string {
	var parts []string
	if p.model.markdown {
		parts = append(parts, plural(len(c.Mentions), "mention"))
		if c.Speeches > 0 {
			parts = append(parts, plural(c.Speeches, "line"))
		}
		return strings.Join(parts, " · ")
	}
	if c.Speeches == 0 {
		parts = append(parts, "in the list, no lines yet")
	} else {
		parts = append(parts, plural(len(c.Scenes), "scene"), nOf(c.Speeches, "speech", "speeches"), plural(c.Words, "word"))
		if len(c.Scenes) > 0 {
			parts = append(parts, "from scene "+p.model.scenes[c.Scenes[0]].label)
		}
	}
	if c.Not {
		parts = append([]string{"not a character"}, parts...)
	}
	return strings.Join(parts, " · ")
}

// next puts the script's cursor on the character's next place (step 1)
// or the one before (-1), the name selected.
func (p *charactersPane) next(key string, step int) {
	w := p.w
	m := p.model
	i := m.find(key)
	if i < 0 {
		return
	}
	c := m.chars[i]
	places := c.places()
	p.lit = key
	p.list.Refresh()
	if len(places) == 0 {
		w.setStatus(c.Name + " has no lines yet")
		return
	}
	e := w.textEditor
	cur := e.CursorOffset()
	if a, b := e.Buffer().Selection(); a != b {
		cur = min(a, b)
	}
	at := -1
	if step > 0 {
		at = nextPlace(places, cur)
	} else {
		for _, pl := range places {
			if pl < cur {
				at = pl
			}
		}
		if at < 0 {
			at = places[len(places)-1]
		}
	}
	n := len([]rune(c.Name))
	if len(c.Cues) > 0 {
		for _, s := range c.Cues {
			if s.start == at {
				n = s.end - s.start
			}
		}
	}
	idx := 0
	for k, pl := range places {
		if pl == at {
			idx = k + 1
		}
	}
	e.Navigate(func(b *buffer.Buffer) {
		end := min(at+n, b.Len())
		b.SetCursor(end, false)
		b.SetCursor(at, true)
	})
	if !w.scriptInMain() && w.mid.shown != "characters" { // the script's tab to the front, to see the cue (middle.go)
		w.showMiddle("")
	}
	what := "cue"
	if m.markdown {
		what = "mention"
	}
	w.setStatus(fmt.Sprintf("%s: %s %d of %d", c.Name, what, idx, len(places)))
}

// charRow is a row of the list: the name (and cast number), the line
// under it, and › to open the character.
type charRow struct {
	widget.BaseWidget
	name, stats *widget.Label
	aliases     *widget.Label // the learned notes' other names, under the cue
	texts       *fyne.Container
	cast        *widget.Label
	photo       *canvas.Image // the cast photo, small (hidden: none)
	bg          *canvas.Rectangle
	open        *widget.Button
	key         string
}

func newCharRow(p *charactersPane) *charRow {
	r := &charRow{}
	r.name = widget.NewLabel("")
	r.name.TextStyle = fyne.TextStyle{Bold: true}
	r.name.Truncation = fyne.TextTruncateEllipsis
	r.stats = widget.NewLabel("")
	r.stats.SizeName = theme.SizeNameCaptionText
	r.stats.Truncation = fyne.TextTruncateEllipsis
	r.aliases = widget.NewLabel("")
	r.aliases.SizeName = theme.SizeNameCaptionText
	r.aliases.Truncation = fyne.TextTruncateEllipsis
	r.aliases.Importance = widget.LowImportance
	r.aliases.Hide()
	r.cast = widget.NewLabel("")
	r.cast.SizeName = theme.SizeNameCaptionText
	r.cast.Importance = widget.LowImportance
	r.photo = canvas.NewImageFromImage(nil)
	r.photo.FillMode = canvas.ImageFillContain
	r.photo.SetMinSize(fyne.NewSize(40, 40))
	r.photo.Hide()
	r.bg = canvas.NewRectangle(color.Transparent)
	r.bg.CornerRadius = theme.InputRadiusSize()
	r.open = widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() { p.openChar(r.key) })
	r.open.Importance = widget.LowImportance
	r.ExtendBaseWidget(r)
	return r
}

func (r *charRow) show(p *charactersPane, i int) {
	c := p.model.chars[i]
	r.key = c.Key
	r.name.SetText(c.Name)
	r.stats.SetText(p.charStats(c))
	if aliases := len(c.Aliases) > 0; aliases != r.aliases.Visible() || aliases {
		r.aliases.SetText("also " + strings.Join(c.Aliases, ", "))
		showIf(r.aliases, aliases)
		if r.texts != nil {
			r.texts.Refresh() // laid out again: the line in or out
		}
	}
	if n := p.castOfKey(c.Key); n > 0 {
		r.cast.SetText(fmt.Sprintf("#%d", n))
		r.cast.Show()
	} else {
		r.cast.Hide()
	}
	if img := p.photoOf(c, 80); img != nil {
		r.photo.Image = img
		r.photo.Show()
		r.photo.Refresh()
	} else {
		r.photo.Hide()
	}
	if c.Not {
		r.name.Importance = widget.LowImportance
	} else {
		r.name.Importance = widget.MediumImportance
	}
	r.name.Refresh()
	r.bg.FillColor = color.Transparent
	if c.Key == p.lit {
		r.bg.FillColor = theme.Color(theme.ColorNameSelection)
	}
	r.bg.Refresh()
	fyne.SetAccessibleLabel(r.open, "Open "+c.Name)
}

func (r *charRow) CreateRenderer() fyne.WidgetRenderer {
	top := container.NewBorder(nil, nil, nil, r.cast, r.name)
	texts := container.New(layout.NewCustomPaddedVBoxLayout(-2*theme.Padding()), top, r.stats, r.aliases)
	r.texts = texts
	return widget.NewSimpleRenderer(container.NewStack(r.bg, container.NewBorder(nil, nil, r.photo, r.open, texts)))
}

// The character opened: back to the list, the next and previous cue, the
// actions, the scenes, the note and the fields.

func (p *charactersPane) makeDetail() {
	d := &p.detail
	back := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), p.closeDetail)
	back.Importance = widget.LowImportance
	fyne.SetAccessibleLabel(back, "Back to the characters")
	d.name = widget.NewLabel("")
	d.name.TextStyle = fyne.TextStyle{Bold: true}
	d.name.Truncation = fyne.TextTruncateEllipsis
	d.stats = widget.NewLabel("")
	d.stats.Wrapping = fyne.TextWrapWord
	d.stats.SizeName = theme.SizeNameCaptionText
	d.aliases = widget.NewLabel("")
	d.aliases.Wrapping = fyne.TextWrapWord
	d.aliases.SizeName = theme.SizeNameCaptionText
	prev := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() { p.next(p.open, -1) })
	nextB := widget.NewButtonWithIcon("Next Cue", theme.NavigateNextIcon(), func() { p.next(p.open, 1) })
	fyne.SetAccessibleLabel(prev, "Previous cue")
	d.actions = container.New(layout.NewRowWrapLayout())
	nav := container.NewHBox(prev, nextB)
	d.listed = widget.NewLabel("")
	d.listed.Wrapping = fyne.TextWrapWord
	d.listed.SizeName = theme.SizeNameCaptionText
	d.listed.Importance = widget.LowImportance
	d.note = widget.NewMultiLineEntry()
	d.note.Wrapping = fyne.TextWrapWord
	d.note.SetMinRowsVisible(3)
	d.note.SetPlaceHolder("Who they are, what they want")
	fyne.SetAccessibleLabel(d.note, "Note")
	d.note.OnChanged = func(s string) {
		if !d.updating {
			p.writeEntry(p.open, -1, func(e *charEntry) { e.Note = s })
		}
	}
	d.noteRow = widget.NewFormItem("Note", d.note)
	items := []*widget.FormItem{d.noteRow}
	for f := range d.fields {
		en := widget.NewEntry()
		fyne.SetAccessibleLabel(en, fieldLabels[f])
		en.OnChanged = func(s string) {
			if !d.updating {
				p.writeEntry(p.open, f, func(e *charEntry) { e.Fields[f] = s })
			}
		}
		d.fields[f] = en
		d.fieldRow[f] = widget.NewFormItem(fieldLabels[f], en)
		items = append(items, d.fieldRow[f])
	}
	d.fields[fieldLook].SetPlaceHolder("What we see")
	d.fields[fieldPhoto].SetPlaceHolder("A picture: Choose Photo…")
	d.photo = canvas.NewImageFromImage(nil)
	d.photo.FillMode = canvas.ImageFillContain
	d.photo.SetMinSize(fyne.NewSize(120, 120))
	d.photo.Hide()
	d.fields[fieldArc].SetPlaceHolder("How they change")
	d.form = widget.NewForm(items...)
	d.form.Orientation = widget.Adaptive
	scenesTitle := widget.NewLabel("Scenes")
	scenesTitle.TextStyle = fyne.TextStyle{Bold: true}
	d.scenes = container.NewVBox()
	d.box = container.NewVBox(container.NewBorder(nil, nil, back, nil, d.name), container.NewHBox(d.photo), d.stats, d.aliases, nav, d.actions, widget.NewSeparator(),
		d.form, d.listed, widget.NewSeparator(), scenesTitle, d.scenes)
}

// openChar opens the character with key.
func (p *charactersPane) openChar(key string) {
	p.open, p.lit = key, key
	p.showDetail()
	p.body.Objects = []fyne.CanvasObject{container.NewVScroll(p.detail.box)}
	p.body.Refresh()
	p.top.Hide() // the filter is the list's
}

// closeDetail goes back to the list.
func (p *charactersPane) closeDetail() {
	p.open = ""
	p.top.Show()
	p.body.Objects = []fyne.CanvasObject{p.list, p.none}
	p.body.Refresh()
	p.refilter()
	p.w.fyneWindow.Canvas().Focus(p.filter)
}

// showDetail shows the character opened as the model has it now; a field
// being typed in is left as it is.
func (p *charactersPane) showDetail() {
	m, d := p.model, &p.detail
	i := m.find(p.open)
	if i < 0 {
		return
	}
	c := m.chars[i]
	name := c.Name
	if n := p.castOfKey(c.Key); n > 0 {
		name = fmt.Sprintf("%s  #%d", c.Name, n)
	}
	d.name.SetText(name)
	if img := p.photoOf(c, 360); img != nil {
		d.photo.Image = img
		d.photo.Show()
		d.photo.Refresh()
	} else {
		d.photo.Hide()
	}
	d.stats.SetText(p.charStats(c))
	if len(c.Aliases) > 0 {
		d.aliases.SetText("Also called (the learned notes): " + strings.Join(c.Aliases, ", "))
		d.aliases.Show()
	} else {
		d.aliases.Hide()
	}
	d.actions.Objects = p.actionButtons(c)
	d.actions.Refresh()
	// the note and fields: a screenplay's list
	focused := p.w.fyneWindow.Canvas().Focused()
	var e charEntry
	if c.Entry >= 0 && c.Entry < len(m.list.entries) {
		e = m.list.entries[c.Entry]
	}
	d.updating = true
	if focused != d.note && d.note.Text != e.Note {
		d.note.SetText(e.Note)
	}
	for f, en := range d.fields {
		if focused != en && en.Text != e.Fields[f] {
			en.SetText(e.Fields[f])
		}
	}
	d.updating = false
	if m.markdown {
		d.form.Hide()
		d.listed.Hide()
	} else {
		d.form.Show()
		d.listed.Show()
		items := []*widget.FormItem{d.noteRow}
		for f, it := range d.fieldRow {
			if f != fieldActor || !p.w.writerView() { // the actor is Production's
				items = append(items, it)
			}
		}
		if len(items) != len(d.form.Items) {
			d.form.Items = items
			d.form.Refresh()
		}
		if c.Entry >= 0 {
			d.listed.SetText("Kept in the script's character list (/* Characters */ at its end)")
		} else {
			d.listed.SetText("Not in the character list yet: a note or a field puts them there")
		}
	}
	// the scenes
	var rows []fyne.CanvasObject
	for _, sc := range c.Scenes {
		s := m.scenes[sc]
		b := widget.NewButton(s.label+". "+s.heading, func() { p.w.goToScript(s.line) })
		b.Alignment = widget.ButtonAlignLeading
		b.Importance = widget.LowImportance
		rows = append(rows, b)
	}
	if len(rows) == 0 {
		none := widget.NewLabel("None yet")
		none.Importance = widget.LowImportance
		rows = append(rows, none)
	}
	d.scenes.Objects = rows
	d.scenes.Refresh()
	d.box.Refresh()
}

// actionButtons are the actions on the character c.
func (p *charactersPane) actionButtons(c charInfo) []fyne.CanvasObject {
	btn := func(label string, f func()) fyne.CanvasObject {
		b := widget.NewButton(label, f)
		return b
	}
	out := []fyne.CanvasObject{btn("Rename Everywhere…", func() { p.askRename(c.Key) })}
	if p.model.markdown {
		return out
	}
	out = append(out, btn("Merge Into…", func() { p.askMerge(c.Key) }), btn("Add Alias…", func() { p.askAlias(c.Key) }))
	if c.Not {
		out = append(out, btn("A Character After All", func() { p.setNot(c.Key, false) }))
	} else if c.Speeches > 0 && c.Entry < 0 {
		out = append(out, btn("Not a Character", func() { p.setNot(c.Key, true) }))
	}
	if c.Entry < 0 {
		out = append(out, btn("Add to the List", func() { p.addToList(c.Key) }))
	}
	out = append(out, btn("Choose Photo…", func() { p.choosePhoto(c.Key) }))
	return out
}

// photoOf is a character's cast photo, at most size pixels a side (nil:
// none, or it cannot be read).
func (p *charactersPane) photoOf(c charInfo, size int) image.Image {
	m := p.model
	if m == nil || m.markdown || c.Entry < 0 || c.Entry >= len(m.list.entries) {
		return nil
	}
	src := strings.TrimSpace(m.list.entries[c.Entry].Fields[fieldPhoto])
	if src == "" {
		return nil
	}
	return p.w.pictures().Thumbnail(src, size, size)
}

// choosePhoto picks a picture for a character's cast photo, keeps it with
// the script (pictures.go) and names it in their entry.
func (p *charactersPane) choosePhoto(key string) {
	w := p.w
	if w.bundle == nil && (w.currentFile == "" || inBrowser) {
		dialog.ShowError(errNoPictureHome, w.fyneWindow)
		return
	}
	chooseFileIn(w.fyneWindow, pictureExtensions, func(picked string) {
		w.keepPicture(picked, w.pictureFolder(""), func(rel string, err error) {
			if err != nil {
				dialog.ShowError(fmt.Errorf("could not keep the photo: %v", err), w.fyneWindow)
				return
			}
			p.writeEntry(key, fieldPhoto, func(e *charEntry) { e.Fields[fieldPhoto] = rel })
			p.showDetail()
			p.refresh()
		})
	})
}

// charOf is the character with key as the model has it.
func (p *charactersPane) charOf(key string) (charInfo, bool) {
	if p.model == nil {
		return charInfo{}, false
	}
	if i := p.model.find(key); i >= 0 {
		return p.model.chars[i], true
	}
	return charInfo{}, false
}

// writeEntry changes the character's entry in the list (made if it has
// none) and writes it into the script: only what changed, the typing in
// one field (field; -1 the note) one undo step of the script's.
// entryTag names the undo step of one field's typing (field -1: the note).
type entryTag struct {
	p     *charactersPane
	key   string
	field int
}

func (p *charactersPane) writeEntry(key string, field int, change func(*charEntry)) {
	w := p.w
	c, ok := p.charOf(key)
	if !ok || w.markdown {
		return
	}
	e := w.textEditor
	text := e.Text()
	list := charListOf(text)
	entries := append([]charEntry(nil), list.entries...)
	i := list.find(key)
	if i < 0 {
		entries = append(entries, charEntry{Name: c.Name, from: -1})
		i = len(entries) - 1
	}
	change(&entries[i])
	if entries[i].from < 0 && entries[i].empty() {
		return // nothing to keep yet
	}
	inside, _ := charactersBlock(text)
	nu := list.inside(entries)
	if nu == inside {
		return
	}
	start, end, block := withCharactersBlock(text, nu)
	tag := entryTag{p, key, field}
	ours := e.UndoTag() == tag
	n := e.UndoSteps()
	e.Edit(func(b *buffer.Buffer) { replaceLeast(b, start, end, block) })
	if e.UndoSteps() > n {
		if ours && n > 0 {
			e.JoinStepsSince(n - 1)
		}
		e.TagLastStep(tag)
	}
}

// addToList puts the character in the list, by the name shown.
func (p *charactersPane) addToList(key string) {
	c, ok := p.charOf(key)
	if !ok {
		return
	}
	e := p.w.textEditor
	start, end, block, ok := addedName(e.Text(), c.Name)
	if !ok {
		return
	}
	e.Edit(func(b *buffer.Buffer) { replaceLeast(b, start, end, block) })
	p.w.setStatus(c.Name + " is in the character list now")
}

// askRename asks for the new name and renames the character everywhere.
func (p *charactersPane) askRename(key string) {
	c, ok := p.charOf(key)
	if !ok {
		return
	}
	entry := widget.NewEntry()
	entry.SetText(c.Name)
	n := len(nameEdits(p.w.textEditor.Text(), p.w.sceneStarts(), key, c.Name, renameEverywhere, p.spellings(c)))
	what := fmt.Sprintf("%s stands in %s (cues, action, dialogue, headings): each is renamed in its own case, in one undo step. "+
		"The title page and the source links are left as they are.", c.Name, plural(n, "place"))
	if p.model.markdown {
		what = fmt.Sprintf("%s stands in %s: each is renamed in its own case, in one undo step.", c.Name, plural(n, "place"))
	}
	note := widget.NewLabel(what)
	note.Wrapping = fyne.TextWrapWord
	dialog.ShowForm("Rename "+c.Name, "Rename", "Cancel", []*widget.FormItem{
		widget.NewFormItem("New name", entry), widget.NewFormItem("", note),
	}, func(ok bool) {
		if ok {
			p.rename(key, entry.Text, renameEverywhere)
		}
	}, p.w.fyneWindow)
	p.w.fyneWindow.Canvas().Focus(entry)
}

// askMerge asks which character this one is, and merges it into them.
func (p *charactersPane) askMerge(key string) {
	c, ok := p.charOf(key)
	if !ok {
		return
	}
	var names []string
	byName := map[string]string{}
	for _, o := range p.model.chars {
		if o.Key != key && !o.Not {
			names = append(names, o.Name)
			byName[o.Name] = o.Name
		}
	}
	if len(names) == 0 {
		p.w.setStatus("No other character to merge " + c.Name + " into")
		return
	}
	pick := widget.NewSelect(names, nil)
	pick.SetSelected(names[0])
	note := widget.NewLabel(c.Name + "'s cues take the other name, and their notes go together. Action and dialogue stay as written.")
	note.Wrapping = fyne.TextWrapWord
	dialog.ShowForm("Merge "+c.Name+" Into…", "Merge", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Into", pick), widget.NewFormItem("", note),
	}, func(ok bool) {
		if ok && pick.Selected != "" {
			p.rename(key, byName[pick.Selected], renameCues)
		}
	}, p.w.fyneWindow)
}

// rename renames or merges the character (scope), the pane following.
func (p *charactersPane) rename(key, name string, scope renameScope) {
	c, ok := p.charOf(key)
	if !ok || strings.TrimSpace(name) == "" {
		return
	}
	if p.w.markdown {
		scope = renameEverywhere
	}
	if o, ok := p.charOf(charKey(name)); ok && scope == renameEverywhere && o.Key != key {
		p.w.setStatus(o.Name + " is a character already: Merge Into… makes one of the two")
		return
	}
	n := p.w.renameCharacter(key, name, scope, p.spellings(c))
	if n == 0 && charKey(name) == key {
		return
	}
	p.w.setStatus(renameStatus(n, c.Name, strings.TrimSpace(name), scope == renameCues))
	if p.open == key {
		p.open = charKey(name)
	}
	p.lit = charKey(name)
	p.read()
}

// spellings are the ways the character is written in the script: its
// name as listed and its cues (a rename takes them in any case).
func (p *charactersPane) spellings(c charInfo) []string {
	out := []string{c.Name}
	for _, s := range c.Cues {
		out = append(out, s.name)
	}
	if c.Entry >= 0 && c.Entry < len(p.model.list.entries) {
		out = append(out, p.model.list.entries[c.Entry].Name)
	}
	return out
}

// askAlias asks for another name of the character: the learned notes'
// (characters: CUE = name), which the source's reading and the reports
// take as theirs.
func (p *charactersPane) askAlias(key string) {
	c, ok := p.charOf(key)
	if !ok {
		return
	}
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Annie, Miss Kovač")
	note := widget.NewLabel("Another name the story or a cue has for " + c.Name + ": kept in the learned notes, read by the source pane and the reports.")
	note.Wrapping = fyne.TextWrapWord
	dialog.ShowForm("Add an Alias of "+c.Name, "Add", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Also called", entry), widget.NewFormItem("", note),
	}, func(ok bool) {
		if ok {
			p.addAlias(key, entry.Text)
		}
	}, p.w.fyneWindow)
	p.w.fyneWindow.Canvas().Focus(entry)
}

// addAlias makes each name (parted by commas) another name of the
// character, as one undo step.
func (p *charactersPane) addAlias(key, names string) {
	c, ok := p.charOf(key)
	if !ok {
		return
	}
	l := learnedOf(p.w.textEditor.Text())
	cue := c.Name
	for _, a := range l.Aliases {
		if charKey(a.Cue) == key {
			cue = a.Cue
		}
	}
	added := 0
	for _, n := range strings.Split(names, ",") {
		if n = strings.TrimSpace(n); n != "" && charKey(n) != key {
			l.addAlias(cue, n)
			added++
		}
	}
	if added > 0 && p.w.writeLearned(l, false) {
		p.w.setStatus(fmt.Sprintf("%s: %s added", c.Name, nOf(added, "alias", "aliases")))
		p.read()
	}
}

// setNot puts the name on the learned notes' not characters list (on:
// left out of the pane's characters and the source's reading) or takes
// it off.
func (p *charactersPane) setNot(key string, on bool) {
	c, ok := p.charOf(key)
	if !ok {
		return
	}
	l := learnedOf(p.w.textEditor.Text())
	if on {
		l.addNot(c.Name)
	} else {
		l.dropNot(c.Name)
	}
	if p.w.writeLearned(l, false) {
		if on {
			p.w.setStatus(c.Name + " is not a character now (the learned notes' not characters)")
		} else {
			p.w.setStatus(c.Name + " is a character again")
		}
		p.read()
	}
}

// nOf is n with the word for one or more.
func nOf(n int, one, more string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, more)
}
