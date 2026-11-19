package main

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The source pane: the Notes pane's other tab (View > Source), the
// script's source as prose, one passage under the other. Tapping a
// passage picks it (for Format > Link to Source Passage) and goes to the
// scene that links to it, the next one on each tap if several do; the
// passages the cursor's scene links to are lit, and scrolled to.
//
// It shows the source read-only; Edit opens it as text in the pane, and
// Done saves it to its file (the browser too: there is one window, so no
// window of its own). A source changed elsewhere (in another window, by
// another app) is read again when its file's time or size changes, which
// the pane looks at whenever it follows the cursor.

// sourceWidth is how wide the side panel is with the source in it.
const sourceWidth = 320

type sourcePane struct {
	w *MainWindow

	ref      sourceRef // where the source is
	modTime  time.Time
	size     int64
	src      string
	passages []passage
	err      error

	model    *sourceModel
	modelFor string // the script text the model was made from

	selected int          // the passage picked (-1 none)
	cycle    int          // which of its scenes the last tap went to
	lit      map[int]bool // linked from the cursor's scene
	doubt    map[int]bool // an ambiguous link of the scene's fits them
	jumping  bool         // the cursor is moving because of a tap here

	heights []float32
	width   float32 // the passages were measured at

	title   *widget.Label
	info    *widget.Label
	list    *widget.List
	edit    *widget.Button
	choose  *widget.Button
	editor  *widget.Entry
	editing bool
	empty   fyne.CanvasObject
	body    *fyne.Container // the list, the editor or the empty state
	box     fyne.CanvasObject
}

func newSourcePane(w *MainWindow) *sourcePane {
	s := &sourcePane{w: w, selected: -1, lit: map[int]bool{}, doubt: map[int]bool{}}
	s.title = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	s.title.Truncation = fyne.TextTruncateEllipsis
	s.info = widget.NewLabel("")
	s.info.Wrapping = fyne.TextWrapWord
	s.info.SizeName = theme.SizeNameCaptionText
	s.info.Importance = widget.LowImportance
	s.list = widget.NewList(
		func() int { return len(s.passages) },
		func() fyne.CanvasObject { return newPassageRow() },
		func(id widget.ListItemID, o fyne.CanvasObject) {
			if id < len(s.passages) {
				s.show(o.(*fyne.Container), id)
			}
		},
	)
	fyne.SetAccessibleLabel(s.list, "Source passages")
	s.list.OnSelected = func(id widget.ListItemID) {
		s.list.UnselectAll()
		s.pick(id)
	}
	s.edit = widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), s.toggleEdit)
	s.edit.Importance = widget.LowImportance
	fyne.SetAccessibleLabel(s.edit, "Edit the source as text")
	s.choose = widget.NewButtonWithIcon("", theme.FolderOpenIcon(), w.chooseSource)
	s.choose.Importance = widget.LowImportance
	fyne.SetAccessibleLabel(s.choose, "Choose the source…")
	s.editor = widget.NewMultiLineEntry()
	s.editor.Wrapping = fyne.TextWrapWord
	fyne.SetAccessibleLabel(s.editor, "Source text")
	none := widget.NewLabel("No source yet. A Markdown file with the script's name beside it " +
		"(story.md for story.fountain) is its source; or choose one.")
	none.Wrapping = fyne.TextWrapWord
	s.empty = container.NewVBox(none, widget.NewButtonWithIcon("Choose Source…", theme.FolderOpenIcon(), w.chooseSource))
	s.body = container.NewStack(newWidthWatch(s.list, s.measure))
	head := container.NewBorder(nil, nil, nil, container.NewHBox(s.edit, s.choose), s.title)
	s.box = container.NewBorder(container.NewVBox(head, s.info), nil, nil, nil, s.body)
	return s
}

// newPassageRow is a passage's row: its background (lit), a bar on its
// left (picked) and its text.
func newPassageRow() *fyne.Container {
	bg := canvas.NewRectangle(color.Transparent)
	bg.CornerRadius = 4
	bar := canvas.NewRectangle(color.Transparent)
	bar.SetMinSize(fyne.NewSize(4, 0))
	text := widget.NewRichText()
	text.Wrapping = fyne.TextWrapWord
	return container.NewStack(bg, container.NewBorder(nil, nil, bar, nil, text))
}

// tint is a theme colour, see-through.
func tint(name fyne.ThemeColorName, alpha uint8) color.Color {
	r, g, b, _ := theme.Color(name).RGBA()
	return color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), alpha}
}

// show fills a row with passage id.
func (s *sourcePane) show(row *fyne.Container, id int) {
	p := s.passages[id]
	bg := row.Objects[0].(*canvas.Rectangle)
	inner := row.Objects[1].(*fyne.Container)
	text, bar := inner.Objects[0].(*widget.RichText), inner.Objects[1].(*canvas.Rectangle)
	switch {
	case s.lit[id]:
		bg.FillColor = tint(theme.ColorNamePrimary, 0x38)
	case s.doubt[id]:
		bg.FillColor = tint(theme.ColorNameWarning, 0x38)
	default:
		bg.FillColor = color.Transparent
	}
	bar.FillColor = color.Transparent
	if id == s.selected {
		bar.FillColor = theme.Color(theme.ColorNamePrimary)
	}
	bg.Refresh()
	bar.Refresh()
	text.ParseMarkdown(p.Markdown)
}

// measure sets the rows' heights for the list's width.
func (s *sourcePane) measure(width float32) {
	if width <= 0 {
		return
	}
	s.width = width
	row := newPassageRow()
	s.heights = make([]float32, len(s.passages))
	for id := range s.passages {
		s.show(row, id)
		row.Resize(fyne.NewSize(width, 1))
		s.heights[id] = row.MinSize().Height
		s.list.SetItemHeight(id, s.heights[id])
	}
	s.list.Refresh()
}

// load reads the window's script's source again if it is another one
// now, or it changed: its block in the script, or its file's time or
// size.
func (s *sourcePane) load() {
	ref := s.w.sourceRef()
	var mod time.Time
	var size int64
	if ref.Path != "" {
		if st, err := os.Stat(ref.Path); err == nil {
			mod, size = st.ModTime(), st.Size()
		}
	}
	var src string
	var err error
	if ref.Origin == sourceInside || ref.Origin == sourceBundle { // in memory
		src, _ = s.w.sourceText(ref)
		if ref == s.ref && src == s.src {
			return
		}
	} else {
		if ref == s.ref && mod.Equal(s.modTime) && size == s.size && s.err == nil {
			return
		}
		src, err = s.w.sourceText(ref)
	}
	if (ref.Origin == sourceInside) != (s.ref.Origin == sourceInside) {
		s.w.refreshMenu() // Keep Source in the Script (an undo can change it)
	}
	s.ref, s.modTime, s.size, s.src, s.err = ref, mod, size, src, err
	s.passages = passagesOf(s.src)
	s.model, s.modelFor = nil, ""
	if s.selected >= len(s.passages) {
		s.selected = -1
	}
	s.lit, s.doubt = map[int]bool{}, map[int]bool{}
	s.showBody()
	s.measure(s.width)
}

// showBody shows the list, the source's text being edited, or what to
// do without a source.
func (s *sourcePane) showBody() {
	var o fyne.CanvasObject
	s.title.SetText(s.ref.describe())
	switch {
	case s.ref.Origin == sourceNone:
		s.edit.Hide()
		o = s.empty
	case s.editing:
		o = s.editor
	default:
		s.edit.Show()
		o = s.body.Objects[0]
		if wl, ok := o.(*widthWatch); !ok || wl.content != s.list {
			o = newWidthWatch(s.list, s.measure)
		}
	}
	if s.err != nil {
		s.setInfo("Could not read it: " + s.err.Error())
	}
	if len(s.body.Objects) == 0 || s.body.Objects[0] != o {
		s.body.Objects = []fyne.CanvasObject{o}
		s.body.Refresh()
	}
}

// current is the model of the script as it is now, made again if the
// script changed.
func (s *sourcePane) current() *sourceModel {
	text := s.w.textEditor.Text()
	if s.model == nil || s.modelFor != text {
		s.model, s.modelFor = newSourceModel(s.passages, text, s.w.sceneStarts()), text
	}
	return s.model
}

// shown reports whether the source is in sight.
func (s *sourcePane) shown() bool {
	n := s.w.notes
	return n != nil && s.w.notesVisible && !s.w.markdown && n.tabs.Selected() == n.sourceTab
}

// follow lights the passages the cursor's scene links to, scrolls to
// them if none is in sight, and says what of the scene's links was not
// found.
func (s *sourcePane) follow() {
	if !s.shown() {
		return
	}
	s.load()
	if s.ref.Origin == sourceNone {
		s.setInfo("")
		return
	}
	m := s.current()
	line, _ := s.w.textEditor.Buffer().Position(s.w.textEditor.CursorOffset())
	lit, doubt := map[int]bool{}, map[int]bool{}
	var problems []string
	first := -1
	for _, i := range m.inSpan(line) {
		mt := m.matches[i]
		q := "“" + shortQuote(m.links[i].Quote) + "”"
		switch mt.Kind {
		case linkFound, linkNear:
			lit[mt.Passages[0]] = true
			if first < 0 || mt.Passages[0] < first {
				first = mt.Passages[0]
			}
			if mt.Kind == linkNear {
				problems = append(problems, q+" found nearly: the passage changed?")
			}
		case linkAmbiguous:
			for _, p := range mt.Passages {
				doubt[p] = true
			}
			problems = append(problems, fmt.Sprintf("%s fits %d passages: quote more of it", q, len(mt.Passages)))
		case linkMissing:
			problems = append(problems, q+" is not in the source")
		}
	}
	span := spanAt(m.spans, line)
	where := span.Heading
	if where == "" {
		where = "Before the first scene"
	}
	info := where + ": " + plural(len(lit), "passage")
	if len(problems) > 0 {
		info += "\n" + strings.Join(problems, "\n")
	}
	if a, n := m.count(linkAmbiguous), m.count(linkMissing); a+n > 0 {
		info += fmt.Sprintf("\nIn the script: %d of %d links ambiguous, %d not found", a, len(m.links), n)
	}
	s.setInfo(info)
	if !sameSet(lit, s.lit) || !sameSet(doubt, s.doubt) {
		s.lit, s.doubt = lit, doubt
		s.list.Refresh()
		if first >= 0 && !s.jumping && !s.inSight(lit) {
			s.list.ScrollTo(first)
		}
	}
}

// setInfo says what the scene's links found; the pane is laid out again
// for the label's new height.
func (s *sourcePane) setInfo(text string) {
	if text == s.info.Text {
		return
	}
	s.info.SetText(text)
	if text == "" {
		s.info.Hide()
	} else {
		s.info.Show()
	}
	if c, ok := s.box.(*fyne.Container); ok {
		c.Refresh()
	}
}

func sameSet(a, b map[int]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// shortQuote is a link's words cut to a few, for a message.
func shortQuote(q string) string {
	words := strings.Fields(q)
	if len(words) > 6 {
		return strings.Join(words[:6], " ") + "…"
	}
	return q
}

// inSight reports whether one of the passages is in the list's view.
func (s *sourcePane) inSight(ps map[int]bool) bool {
	if len(s.heights) != len(s.passages) {
		return false
	}
	top, view := s.list.GetScrollOffset(), s.list.Size().Height
	pad := theme.Padding() // the list's separators
	y := float32(0)
	for id, h := range s.heights {
		if ps[id] && y+h > top && y < top+view {
			return true
		}
		y += h + pad
	}
	return false
}

// pick is a tap on passage id: it is the picked one, and the cursor goes
// to the scene linking to it (on another tap, the next such scene).
func (s *sourcePane) pick(id int) {
	if id < 0 || id >= len(s.passages) {
		return
	}
	if id == s.selected {
		s.cycle++
	} else {
		s.selected, s.cycle = id, 0
	}
	s.list.Refresh()
	scenes := s.current().linked(id)
	if len(scenes) == 0 {
		s.w.setStatus("Picked; no scene links to it yet: Format > Link to Source Passage puts the link at the cursor")
		return
	}
	line := scenes[s.cycle%len(scenes)]
	s.jumping = true
	s.w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(line), false) })
	s.jumping = false
	if len(scenes) > 1 {
		s.w.setStatus(fmt.Sprintf("Scene %d of %d from this passage: tap it again for the next", s.cycle%len(scenes)+1, len(scenes)))
	} else {
		s.w.setStatus("The scene from this passage")
	}
}

// toggleEdit opens the source as text in the pane, or saves it and
// shows it as passages again.
func (s *sourcePane) toggleEdit() {
	if s.ref.Origin == sourceNone {
		return
	}
	if !s.editing {
		s.editing = true
		s.editor.SetText(s.src)
		s.edit.SetText("Done")
		s.edit.SetIcon(theme.ConfirmIcon())
		fyne.SetAccessibleLabel(s.edit, "Done: save the source")
		s.showBody()
		s.w.fyneWindow.Canvas().Focus(s.editor)
		return
	}
	if s.editor.Text != s.src {
		if s.ref.Origin == sourceInside { // into the script's block
			e := s.w.textEditor
			start, end, block := withSourceBlock(e.Text(), s.editor.Text)
			e.Replace(start, end, block)
		} else if s.ref.Origin == sourceBundle {
			if err := s.w.saveBundleSource(s.editor.Text); err != nil {
				dialog.ShowError(err, s.w.fyneWindow)
				return
			}
		} else if err := writeFileAtomic(s.ref.Path, []byte(s.editor.Text)); err != nil {
			dialog.ShowError(fmt.Errorf("could not save the source: %v", err), s.w.fyneWindow)
			return
		}
		s.modTime = time.Time{} // read again
	}
	s.editing = false
	s.edit.SetText("Edit")
	s.edit.SetIcon(theme.DocumentCreateIcon())
	fyne.SetAccessibleLabel(s.edit, "Edit the source as text")
	s.load()
	s.showBody()
	s.follow()
}

// focusTarget is what takes the keyboard in the pane.
func (s *sourcePane) focusTarget() fyne.Focusable {
	if s.editing {
		return s.editor
	}
	return s.list
}

// widthWatch lays out its content to its size, and says when its width
// changes (the passages' wrapped heights depend on it).
type widthWatch struct {
	widget.BaseWidget
	content fyne.CanvasObject
	onWidth func(float32)
	width   float32
}

func newWidthWatch(content fyne.CanvasObject, onWidth func(float32)) *widthWatch {
	ww := &widthWatch{content: content, onWidth: onWidth}
	ww.ExtendBaseWidget(ww)
	return ww
}

func (ww *widthWatch) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(ww.content)
}

func (ww *widthWatch) Resize(size fyne.Size) {
	ww.BaseWidget.Resize(size)
	if size.Width != ww.width {
		ww.width = size.Width
		ww.onWidth(size.Width)
	}
}

// The window's side of it.

// sourcePane is the Notes pane's source tab, made when first wanted.
func (w *MainWindow) sourcePane() *sourcePane {
	if w.notes == nil {
		w.notes = newNotesPanel(w)
	}
	return w.notes.source
}

// toggleSource is View > Source: the Notes pane with the source in it,
// or (if that is shown) neither.
func (w *MainWindow) toggleSource() {
	s := w.sourcePane()
	if s.shown() {
		w.toggleNotes()
		return
	}
	w.notes.tabs.Select(w.notes.sourceTab)
	if !w.notesVisible {
		w.toggleNotes()
	}
	w.fyneWindow.Canvas().Focus(s.focusTarget())
	s.follow()
}

// chooseSource is File > Choose Source…: a story to be the script's
// source, kept by Accolade (source_file.go) and remembered for it.
func (w *MainWindow) chooseSource() {
	exts := []string{".md", ".markdown", ".txt", ".docx", ".odt", ".html", ".htm", ".xhtml"}
	w.chooseFile(exts, func(path string) {
		if err := w.useSource(path); err != nil {
			dialog.ShowError(fmt.Errorf("could not use %s as the source: %v", filepath.Base(path), err), w.fyneWindow)
			return
		}
		w.showSource()
		if w.bundle != nil {
			w.setStatus("Source: " + filepath.Base(path) + ", put in " + filepath.Base(w.bundle.path))
		} else {
			w.setStatus("Source: " + filepath.Base(path) + ", kept by Accolade")
		}
	})
}

// showSource shows the source pane, the source read again.
func (w *MainWindow) showSource() {
	s := w.sourcePane()
	s.ref = sourceRef{Origin: -1} // read again
	if !s.shown() {
		w.toggleSource()
	}
	s.follow()
}

// saveSourceAs is File > Save Source As…: the source, wherever it is
// kept, written out as Markdown (the browser downloads it).
func (w *MainWindow) saveSourceAs() {
	r := w.sourceRef()
	if r.Origin == sourceNone {
		w.setStatus("No source: File > Choose Source…")
		return
	}
	md, err := w.sourceText(r)
	if err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return
	}
	name := w.sourceName(r)
	if inBrowser {
		p := filepath.Join("/files", name)
		if err := os.MkdirAll("/files", 0o755); err == nil {
			err = os.WriteFile(p, []byte(md), 0o644)
		}
		if err != nil {
			dialog.ShowError(err, w.fyneWindow)
			return
		}
		offerDownload(p)
		return
	}
	d := dialog.NewFileSave(func(c fyne.URIWriteCloser, err error) {
		if err != nil || c == nil {
			return
		}
		p := c.URI().Path()
		c.Close()
		if err := writeFileAtomic(p, []byte(md)); err != nil {
			dialog.ShowError(fmt.Errorf("could not save the source: %v", err), w.fyneWindow)
			return
		}
		w.setStatus("Source saved as " + filepath.Base(p))
	}, w.fyneWindow)
	d.SetFileName(name)
	d.Show()
}

// toggleSourceInside is Format > Keep Source in the Script: the source
// moved into the script's end as a boneyard block, or out of it again.
func (w *MainWindow) toggleSourceInside() {
	in := w.sourceRef().Origin != sourceInside
	if err := w.keepSourceInside(in); err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return
	}
	if in {
		w.setStatus("The source is kept in the script now, at its end")
	} else {
		w.setStatus("The source is kept by Accolade now, out of the script")
	}
	if w.notes != nil {
		w.notes.source.follow()
	}
}

// linkToSource is Format > Link to Source Passage: a link to the passage
// picked in the source pane, at the cursor.
func (w *MainWindow) linkToSource() {
	if w.markdown {
		return
	}
	s := w.sourcePane()
	if !s.shown() {
		w.toggleSource()
		w.setStatus("Pick the passage in the source, then Link to Source Passage again")
		return
	}
	s.load()
	if s.selected < 0 || s.selected >= len(s.passages) || !s.passages[s.selected].linkable() {
		w.setStatus("Pick the passage in the source first")
		return
	}
	quote, unique := anchorFor(s.passages, s.selected)
	note := linkNote(quote)
	e := w.textEditor
	at, insert, after := insertLink(e.Text(), e.CursorOffset(), note, w.sceneStarts())
	e.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			b.Replace(at, at, insert)
			b.SetCursor(after, false)
		})
	})
	if unique {
		w.setStatus("Linked to “" + shortQuote(quote) + "”")
	} else {
		w.setStatus("Linked, but the source has this passage more than once: the link cannot tell which")
	}
	s.follow()
}

// sectionsFromSourceHere is Format > Source Headings as Sections: the
// source's headings, those not in the script yet, as its sections.
func (w *MainWindow) sectionsFromSourceHere() {
	if w.markdown {
		return
	}
	s := w.sourcePane()
	s.load()
	if s.ref.Origin == sourceNone {
		w.setStatus("No source: File > Choose Source…")
		return
	}
	e := w.textEditor
	edits := sectionsFromSource(e.Text(), w.sceneStarts(), sourceHeadings(s.passages), e.CursorOffset())
	if len(edits) == 0 {
		w.setStatus("The source's headings are all in the script already")
		return
	}
	n := 0
	for _, ed := range edits {
		n += strings.Count("\n"+ed.Text, "\n#")
	}
	e.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			cursor := b.Cursor()
			for _, ed := range edits { // the last first: the offsets before stay
				b.Replace(ed.At, ed.At, ed.Text)
				n += strings.Count(ed.Text, "# ") + strings.Count(ed.Text, "# ")*0
				if ed.At <= cursor {
					cursor += len([]rune(ed.Text))
				}
			}
			b.SetCursor(cursor, false)
		})
	})
	w.setStatus(plural(n, "section") + " from the source's headings")
	s.follow()
}

// sourceInsideItem is Format > Keep Source in the Script, checked while
// it is.
func (w *MainWindow) sourceInsideItem() *fyne.MenuItem {
	item := fyne.NewMenuItem("Keep Source in the Script", w.toggleSourceInside)
	item.Checked = w.textEditor != nil && w.sourceRef().Origin == sourceInside
	return item
}
