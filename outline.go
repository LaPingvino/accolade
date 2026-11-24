package main

import (
	"image/color"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/lex"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/prose"
)

// The outline (View > Outline) lists the structure of the document in
// front in the middle (Joop, 2026-11-23: "Outline might also need to
// refresh on context? (depending on Source or Screenplay active in the
// middle?)"): the script's sections and scenes with their synopses (a
// Markdown script's headings), the source's chapters and passages, the
// notes' headings; with a view in front (the cards), the script's.
// Clicking one goes there. Dragging a row moves a scene or section, and
// its menu colours it: the script's only.

// outlineItem is a section or scene of the script.
type outlineItem struct {
	Line     int    // where it is
	Text     string // the heading, without markers, notes or scene number
	Synopsis string // its own synopsis: the first "= ..." lines under it
	Color    string // the heading's color note, as lex.NoteColor reads it
	Level    int    // a section's level (# 1, ## 2); scenes are one deeper
	Scene    bool
	// Loose are the synopses in it after its own: each "= ..." paragraph
	// standing on its own (a beat, a moment planned), in the text's order
	Loose []looseSynopsis
}

// looseSynopsis is a synopsis paragraph standing on its own in a scene or
// section: its first line and its text (its lines joined).
type looseSynopsis struct {
	Line int
	Text string
}

// outlineOf the script's text.
func outlineOf(text string, starts []string) []outlineItem {
	var items []outlineItem
	section := 0
	lines := strings.Split(text, "\n")
	inBoneyard := false // notes and other /* ... */ are not the script
	// own: the item's own synopsis can still come (only blank lines and
	// notes since its heading); inSyn: the line before was a synopsis
	own, inSyn := false, false
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if inBoneyard || strings.HasPrefix(t, "/*") {
			inBoneyard = !strings.Contains(strings.TrimPrefix(t, "/*"), "*/")
			inSyn = false
			continue
		}
		prevBlank := i == 0 || strings.TrimSpace(lines[i-1]) == ""
		switch {
		case strings.HasPrefix(t, "#"):
			level := len(t) - len(strings.TrimLeft(t, "#"))
			section = level
			items = append(items, outlineItem{Line: i, Text: lex.SectionText(t), Level: level, Color: lex.NoteColor(t)})
			own, inSyn = true, false
		case prevBlank && (isSceneHeadingIn(t, starts) || (strings.HasPrefix(t, ".") && !strings.HasPrefix(t, ".."))):
			heading, _ := lex.SceneNumber(strings.TrimPrefix(t, "."))
			items = append(items, outlineItem{Line: i, Text: strings.ToUpper(strings.TrimSpace(heading)), Level: section + 1, Scene: true, Color: lex.NoteColor(t)})
			own, inSyn = true, false
		case isSynopsisLine(t) && len(items) > 0:
			it := &items[len(items)-1]
			s := strings.TrimSpace(t[1:])
			switch {
			case own:
				it.Synopsis = strings.TrimSpace(it.Synopsis + " " + s)
			case inSyn && len(it.Loose) > 0:
				lo := &it.Loose[len(it.Loose)-1]
				lo.Text = strings.TrimSpace(lo.Text + " " + s)
			default:
				it.Loose = append(it.Loose, looseSynopsis{Line: i, Text: s})
			}
			inSyn = true
		case t == "":
			if inSyn {
				own = false // its own synopsis has ended
			}
			inSyn = false
		case isNoteLine(t):
			inSyn = false
		default:
			own, inSyn = false, false
		}
	}
	return items
}

// looseText is item it's loose synopses, a line each (for finding them).
func looseText(it outlineItem) string {
	var b strings.Builder
	for _, l := range it.Loose {
		b.WriteString("\n" + l.Text)
	}
	return b.String()
}

// isSynopsisLine tells whether the trimmed line t is a synopsis ("= …",
// not a page break's "===").
func isSynopsisLine(t string) bool { return strings.HasPrefix(t, "=") && !strings.HasPrefix(t, "==") }

// isNoteLine tells whether the trimmed line t is a note alone: [[…]].
func isNoteLine(t string) bool { return strings.HasPrefix(t, "[[") && strings.HasSuffix(t, "]]") }

// filterOutline is the items whose heading or synopsis has query in it
// (in any case), each with the sections it is in, for where it is.
func filterOutline(items []outlineItem, query string) []outlineItem {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return items
	}
	var out []outlineItem
	var sections []int // the open sections, outermost first (indexes)
	shown := map[int]bool{}
	for i, it := range items {
		if !it.Scene {
			for len(sections) > 0 && items[sections[len(sections)-1]].Level >= it.Level {
				sections = sections[:len(sections)-1]
			}
		}
		if strings.Contains(strings.ToLower(it.Text+"\n"+it.Synopsis+looseText(it)), query) {
			for _, s := range sections {
				if !shown[s] {
					shown[s] = true
					out = append(out, items[s])
				}
			}
			shown[i] = true
			out = append(out, it)
		}
		if !it.Scene {
			sections = append(sections, i)
		}
	}
	return out
}

// foldKey names a section across edits (its line moves).
func foldKey(it outlineItem) string { return strconv.Itoa(it.Level) + " " + it.Text }

// hasInside reports whether the i-th item is a section with something in it.
func hasInside(items []outlineItem, i int) bool {
	return !items[i].Scene && i+1 < len(items) && (items[i+1].Scene || items[i+1].Level > items[i].Level)
}

// unfolded is the items without those inside a folded section.
func unfolded(items []outlineItem, folded map[string]bool) []outlineItem {
	var out []outlineItem
	hiding := 0 // inside a folded section of this level (0: none)
	for _, it := range items {
		if hiding > 0 && (it.Scene || it.Level > hiding) {
			continue
		}
		hiding = 0
		out = append(out, it)
		if !it.Scene && folded[foldKey(it)] {
			hiding = it.Level
		}
	}
	return out
}

// outlinePanel is the outline beside the editor.
type outlinePanel struct {
	w *MainWindow
	// of is the document it shows (state.go), doc that document
	of    docKey
	doc   *document
	timer *uiTimer // a document's edits, refreshed after a pause
	// srcOf and srcItems are a source's outline as made last
	srcOf    string
	srcItems []outlineItem
	all      []outlineItem // the document's
	items    []outlineItem // shown: those the filter lets through
	filter   *widget.Entry
	cards    *widget.Button
	folded   map[string]bool // sections shown without what is in them
	list     *widget.List
	box      fyne.CanvasObject
	dots     map[int][]color.Color // who is in which item, by index (collab_cursors.go)
	// beats are the beats no scene grew from by section line (the
	// script's: notes_marks.go)
	beats map[int][]int

	oneLine, twoLines float32 // row heights without and with a synopsis
}

func newOutlinePanel(w *MainWindow) *outlinePanel {
	o := &outlinePanel{w: w, folded: map[string]bool{}}
	o.list = widget.NewList(
		func() int { return len(o.items) },
		func() fyne.CanvasObject {
			title := widget.NewLabel("")
			title.Truncation = fyne.TextTruncateEllipsis
			syn := widget.NewLabel("") // its own synopsis, at the weight of the beats under it
			syn.SizeName = theme.SizeNameCaptionText
			syn.TextStyle = fyne.TextStyle{Italic: true}
			syn.Truncation = fyne.TextTruncateEllipsis
			fold := widget.NewButtonWithIcon("", theme.MenuDropDownIcon(), nil)
			fold.Importance = widget.LowImportance
			// the button's room stays when it is hidden (a scene), so the
			// rows' text does not move as rows are reused and redrawn
			room := canvas.NewRectangle(color.Transparent)
			room.SetMinSize(fold.MinSize())
			swatch := container.NewCenter(newSwatch())
			dots := container.NewCenter(newPeerDots()) // the co-writers in it (collab_cursors.go)
			dots.Hide()
			loose := container.NewVBox() // the loose synopses in it, a line each
			beats := newLinkLabel(nil)   // a section's beats without a scene: a tap steps through them (notes_marks.go)
			beats.Importance = widget.WarningImportance
			beats.Truncation = fyne.TextTruncateEllipsis
			beats.Hide()
			return newDragRow(o, container.NewBorder(nil, nil, container.NewStack(room, fold), container.NewHBox(dots, swatch), container.NewVBox(title, syn, loose, beats)))
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(o.items) {
				return
			}
			it := o.items[id]
			drag := obj.(*dragRow)
			drag.id = id
			row := drag.content
			box, fold := row.Objects[0].(*fyne.Container), row.Objects[1].(*fyne.Container).Objects[1].(*widget.Button)
			title, syn := box.Objects[0].(*widget.Label), box.Objects[1].(*widget.Label)
			o.showFold(fold, id)
			right := row.Objects[2].(*fyne.Container)
			showSwatch(right.Objects[1].(*fyne.Container).Objects[0].(*canvas.Rectangle), it.Color)
			showRowDots(right.Objects[0].(*fyne.Container), o.dots[id])
			title.SetText(strings.Repeat("  ", max(0, it.Level-1)) + it.Text)
			title.TextStyle = fyne.TextStyle{Bold: !it.Scene}
			title.Refresh()
			syn.SetText(strings.Repeat("  ", max(0, it.Level-1)) + it.Synopsis)
			if it.Synopsis == "" {
				syn.Hide()
			} else {
				syn.Show()
			}
			showLoose(box.Objects[2].(*fyne.Container), it)
			o.showBeats(box.Objects[3].(*linkLabel), it)
			box.Refresh() // laid out again: the rows are reused
		},
	)
	o.list.OnSelected = func(id widget.ListItemID) {
		if id < len(o.items) {
			o.goTo(o.items[id].Line)
		}
		o.list.UnselectAll()
	}
	width := canvas.NewRectangle(color.Transparent) // a nil colour crashes the painter
	width.SetMinSize(fyne.NewSize(240, 0))
	fyne.SetAccessibleLabel(o.list, "Outline")
	o.filter = widget.NewEntry()
	o.filter.SetPlaceHolder("Filter scenes and sections")
	fyne.SetAccessibleLabel(o.filter, "Filter the outline")
	o.filter.OnChanged = func(string) { o.show() }
	o.filter.OnSubmitted = func(string) { // Enter: to the first one shown
		if len(o.items) > 0 {
			o.goTo(o.items[0].Line)
		}
	}
	o.cards = widget.NewButtonWithIcon("", theme.GridIcon(), w.toggleCards)
	o.cards.Importance = widget.LowImportance
	fyne.SetAccessibleLabel(o.cards, "Cards: the outline as index cards")
	// its tab in the dock names it (dock.go)
	o.box = container.NewStack(width, container.NewBorder(container.NewBorder(nil, nil, nil, o.cards, o.filter), nil, nil, nil, o.list))
	return o
}

// refresh reads the outline again from the document in front.
func (o *outlinePanel) refresh() {
	cancel(&o.timer)
	w := o.w
	d := w.frontDoc()
	o.of, o.doc = w.frontKey(), d
	o.all = o.itemsOf(d)
	o.beats = nil
	if d.kind == docScript && !w.markdown {
		o.beats = sectionBeats(d.Text(), w.sceneStarts(), o.all)
	}
	switch {
	case d.kind == docSource:
		o.filter.SetPlaceHolder("Filter chapters and passages")
		o.cards.Hide()
	case w.markdown: // headings only: no scenes, no cards
		o.filter.SetPlaceHolder("Filter headings")
		o.cards.Hide()
	default:
		o.filter.SetPlaceHolder("Filter scenes and sections")
		o.cards.Show()
	}
	o.show()
}

// itemsOf is document d's outline: the script's (a screenplay's sections
// and scenes, a Markdown script's headings), a source's chapters and
// passages, the notes' headings.
func (o *outlinePanel) itemsOf(d *document) []outlineItem {
	switch d.kind {
	case docSource:
		return o.sourceItems(d.Text())
	}
	return o.w.outlineItems(d.Text())
}

// edited is a change in the script or the document in front: the
// script's outline at once (as it always was), a document's after a
// pause (a source can be a novel).
func (o *outlinePanel) edited() {
	if o.w.frontDoc().kind == docScript {
		o.refresh()
		return
	}
	o.w.debounce(&o.timer, followPause, func() {
		if o.w.outlineInSight() {
			o.refresh()
		}
	})
}

// sourceItems are a source's outline: its headings (the chapters) as
// sections, its passages under them by their first words.
func (o *outlinePanel) sourceItems(text string) []outlineItem {
	if o.srcItems != nil && sameString(o.srcOf, text) {
		return o.srcItems
	}
	ps := passagesOf(text)
	lines := passageLines(text, ps)
	var items []outlineItem
	section := 0
	for i, p := range ps {
		if lines[i] < 0 {
			continue // not found in the text as it is (being typed)
		}
		if p.Kind == prose.Heading {
			section = p.Level
			items = append(items, outlineItem{Line: lines[i], Text: p.Plain, Level: p.Level})
			continue
		}
		if !p.linkable() {
			continue
		}
		items = append(items, outlineItem{Line: lines[i], Text: shortQuote(p.Plain), Level: section + 1, Scene: true})
	}
	o.srcOf, o.srcItems = text, items
	return items
}

// outlineInSight reports whether the outline is where it can be seen: in
// the dock, or in front in the middle (dock.edited's test).
func (w *MainWindow) outlineInSight() bool {
	p := w.dock.pane("outline")
	return p != nil && (w.dock.shown("outline") || w.mid.boxShown(p))
}

// ofScript reports whether the outline shows the script (its rows move
// and colour scenes).
func (o *outlinePanel) ofScript() bool { return o.doc == nil || o.doc.kind == docScript }

// show shows the items the filter lets through.
func (o *outlinePanel) show() {
	if o.filter.Text != "" {
		o.items = filterOutline(o.all, o.filter.Text)
	} else {
		o.items = unfolded(o.all, o.folded)
	}
	o.dots = nil
	if o.ofScript() {
		o.dots = o.w.peersInScenes(o.items) // who is in which (collab_cursors.go)
	}
	// rows without a synopsis are one line high
	if o.oneLine == 0 {
		row := o.list.CreateItem().(*dragRow).content
		row.Objects[0].(*fyne.Container).Objects[2].Hide() // no loose synopses
		row.Objects[0].(*fyne.Container).Objects[3].Hide() // no beats' mark
		o.twoLines = row.MinSize().Height
		row.Objects[0].(*fyne.Container).Objects[1].Hide()
		o.oneLine = row.MinSize().Height
	}
	for id := range o.items {
		o.list.SetItemHeight(id, o.rowHeight(id))
	}
	o.list.Refresh()
}

// showLoose shows item it's loose synopses in box, a line each ("= …"),
// as its synopsis is shown.
func showLoose(box *fyne.Container, it outlineItem) {
	if len(it.Loose) == 0 {
		box.Hide()
		return
	}
	box.Show()
	for len(box.Objects) < len(it.Loose) {
		l := widget.NewLabel("")
		l.SizeName = theme.SizeNameCaptionText
		l.TextStyle = fyne.TextStyle{Italic: true}
		l.Truncation = fyne.TextTruncateEllipsis
		box.Add(l)
	}
	for k, o := range box.Objects {
		l := o.(*widget.Label)
		if k >= len(it.Loose) {
			l.Hide()
			continue
		}
		l.SetText(strings.Repeat("  ", max(0, it.Level)) + "= " + it.Loose[k].Text)
		l.Show()
	}
}

// showFold makes the item's fold button: shown for a section with
// something in it (not while filtering, which shows all it finds).
func (o *outlinePanel) showFold(b *widget.Button, id int) {
	it := o.items[id]
	i := slices.IndexFunc(o.all, func(a outlineItem) bool { return a.Line == it.Line })
	if o.filter.Text != "" || i < 0 || !hasInside(o.all, i) {
		b.Hide()
		return
	}
	b.Show()
	key := foldKey(it)
	if o.folded[key] {
		b.SetIcon(theme.MenuExpandIcon())
		fyne.SetAccessibleLabel(b, "Unfold "+it.Text)
	} else {
		b.SetIcon(theme.MenuDropDownIcon())
		fyne.SetAccessibleLabel(b, "Fold "+it.Text)
	}
	b.OnTapped = func() {
		o.folded[key] = !o.folded[key]
		o.show()
	}
}

// goTo puts the cursor at the start of a line of the document shown and
// the focus in its editor. For the script with another middle tab in
// front, the script's comes forward (goToScript).
func (o *outlinePanel) goTo(line int) {
	d := o.doc
	if o.ofScript() || o.w.main() != d {
		o.w.goToScript(line)
		return
	}
	d.ed.Navigate(func(b *buffer.Buffer) {
		n, _ := b.Position(b.Len())
		b.SetCursor(b.LineStart(max(0, min(line, n))), false)
	})
	o.w.fyneWindow.Canvas().Focus(d.ed)
}

// dragList is a list of the script's items whose rows can be dragged
// to move them (the outline, the Cards pane).
type dragList interface {
	rowItems() []outlineItem  // the items shown
	colours() bool            // its rows are the script's: a secondary tap colours one
	rowHeight(id int) float32 // a shown row's height
	tapRow(id int)            // a tap on a row
	moveRow(from, to int)     // a row dropped before row to (len: the end)
	listWindow() *MainWindow
}

// dragRow is a row that can be dragged to move its scene or section:
// dropped, it goes before the row it is dropped on.
type dragRow struct {
	widget.BaseWidget
	o       dragList
	content *fyne.Container
	id      int
	dy      float32
}

func newDragRow(o dragList, content *fyne.Container) *dragRow {
	r := &dragRow{o: o, content: content}
	r.ExtendBaseWidget(r)
	return r
}

func (r *dragRow) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(r.content) }

// Tapped selects the row, as the list's own item does: a row that takes
// a secondary tap is what Fyne gives the tap to.
func (r *dragRow) Tapped(*fyne.PointEvent) { r.o.tapRow(r.id) }

// TappedSecondary shows the row's scene or section's menu: its color,
// Swap…, Move….
func (r *dragRow) TappedSecondary(e *fyne.PointEvent) {
	if items := r.o.rowItems(); r.o.colours() && r.id >= 0 && r.id < len(items) {
		r.o.listWindow().itemMenu(items[r.id], e.AbsolutePosition)
	}
}

func (r *dragRow) Dragged(e *fyne.DragEvent) {
	r.dy += e.Dragged.DY
	items := r.o.rowItems()
	if to := dropIndex(r.o, r.id, r.dy); to >= 0 && to < len(items) {
		r.o.listWindow().setStatus("Move before " + items[to].Text)
	} else if to >= 0 {
		r.o.listWindow().setStatus("Move to the end")
	}
}

func (r *dragRow) DragEnd() {
	to := dropIndex(r.o, r.id, r.dy)
	r.dy = 0
	if to < 0 {
		return
	}
	r.o.moveRow(r.id, to)
}

// dropIndex is where a row dragged dy from row id goes: before the row
// shown there (len(items): the end), -1 if it stays.
func dropIndex(o dragList, id int, dy float32) int {
	items := o.rowItems()
	height := o.rowHeight
	if id < 0 || id >= len(items) {
		return -1
	}
	at := id
	if dy > 0 {
		y := dy - height(id)/2
		for at+1 < len(items) && y >= height(at+1)/2 {
			y -= height(at + 1)
			at++
		}
		if at == id {
			return -1
		}
		if at+1 >= len(items) && y > 0 {
			return len(items)
		}
		return at + 1
	}
	y := -dy - height(id)/2
	for at > 0 && y >= height(at-1)/2 {
		y -= height(at - 1)
		at--
	}
	if at == id {
		return -1
	}
	return at
}

// The outline as a dragList.
func (o *outlinePanel) rowItems() []outlineItem { return o.items }
func (o *outlinePanel) rowHeight(id int) float32 {
	it := o.items[id]
	h := o.oneLine + float32(len(it.Loose))*(o.twoLines-o.oneLine) // a line each
	if it.Synopsis != "" {
		h += o.twoLines - o.oneLine
	}
	if !it.Scene && len(o.beats[it.Line]) > 0 {
		h += o.twoLines - o.oneLine
	}
	return h
}

// showBeats shows a section's mark of its beats without a scene.
func (o *outlinePanel) showBeats(l *linkLabel, it outlineItem) {
	lines := o.beats[it.Line]
	if it.Scene || len(lines) == 0 {
		l.Hide()
		return
	}
	l.SetText(strings.Repeat("  ", max(0, it.Level-1)) + beatsMark(len(lines)))
	l.onTap = func() { o.w.stepToBeat(lines) }
	l.Show()
}
func (o *outlinePanel) tapRow(id int) { o.list.Select(id) }
func (o *outlinePanel) moveRow(from, to int) {
	if !o.ofScript() {
		o.w.setStatus("Rows move in the script's outline only")
		return
	}
	o.move(from, to)
}
func (o *outlinePanel) colours() bool           { return o.ofScript() }
func (o *outlinePanel) listWindow() *MainWindow { return o.w }

// dropIndex is dropIndex in the outline (its tests').
func (o *outlinePanel) dropIndex(id int, dy float32) int { return dropIndex(o, id, dy) }

// move moves the shown item from before the shown item to (len: the end)
// in the script.
func (o *outlinePanel) move(from, to int) {
	index := func(shown int) int { // in the script's whole outline
		if shown >= len(o.items) {
			return len(o.all)
		}
		for i, it := range o.all {
			if it.Line == o.items[shown].Line {
				return i
			}
		}
		return -1
	}
	if f, t := index(from), index(to); f >= 0 && t >= 0 {
		o.w.moveOutlineItem(f, t)
	}
}
