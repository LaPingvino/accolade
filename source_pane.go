package main

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/pace"
	"github.com/LaPingvino/accolade/internal/storyread"
)

// The source pane: the Notes pane's other tab (View > Source), the
// script's source as prose, one passage under the other. Tapping a
// passage picks it (for Adapt…, Mark as Cut, Adaptation > Link to Source
// Passage); a passage scenes link to shows them under it ("→ Scene 4",
// "→ 2 scenes"), and that goes to the scene, which flashes; the passages
// the cursor's scene links to are lit, and scrolled to. Next and
// Previous pick the next passage still to do from where the writer is
// (the caption under the count says from where: adapt_plan.go).
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
	modelFor string       // the script text the model was made from
	matcher  *linkMatcher // the passages' links found, kept while they stay

	selected   int            // the passage picked (-1 none)
	selWords   []string       // its words, to find it again when the source is read again
	pickFresh  bool           // picked (or gone to by Next) since the cursor last moved: the anchor
	pickCursor int            // the cursor then
	cycle      int            // which of its scenes Go to the Passage's Scene went to last
	cycleFor   int            // for which passage
	linkedBy   map[int][]int  // the spans (first lines) linking each passage, as the rows show them
	labels     map[int]string // the spans' names (spanLabels), for the script text labelsFor
	labelsFor  string
	acts       []bool       // the rows' heights were made with the line under the text (rowActs)
	adapting   *adaptDialog // Adapt… open
	lit        map[int]bool // linked from the cursor's scene
	doubt      map[int]bool // an ambiguous link of the scene's fits them
	jumping    bool         // the cursor is moving because of a tap here

	statuses     []passageStatus     // as the rows show them (adaptation.go)
	changed      map[int]bool        // adapted passages changed since they were linked
	linkedWords  map[string][]string // each link's passage's words as first seen linked (by the link's words)
	sugg         suggestions         // links offered (adaptation_suggest.go)
	seedAsked    bool                // an unwritten script offered to start from the source
	fresh        bool                // the source just read: the list not scrolled yet
	offered      int                 // the passage Next showed last (-1 none)
	offeredWords []string            // its words, to find it again

	reading    *storyread.Reading // what the source shows (source_reading.go)
	readingFor string
	view       *readingView
	learned    *learnedView // what Accolade learned (adaptation_learned_view.go)

	heights          []float32 // the rows' heights, estimated or measured
	exact            []bool    // measured
	width            float32   // the passages were measured at
	est              rowEstimate
	measuring        int                         // the measuring under way (a new one stops it)
	scanned          int                         // rows before it are measured, in the measuring's order
	measureT, widthT *uiTimer                    // the measuring's next stretch; a width change waited out
	shownMD          map[*widget.RichText]string // what each row shows (rowKey)
	full             map[int]bool                // rows shown in full (Show All)
	measureRow       *fyne.Container
	loading          *sourceRead // the source being read in the background
	hideRows         bool        // the list is told it has none (setHeights)

	title      *widget.Label
	info       *widget.Label
	list       *widget.List
	edit       *widget.Button
	choose     *widget.Button
	count      *widget.Label // "62% adapted · 5 cut · 12 to go"
	anchorLine *widget.Label // where Next and Previous go on from
	next, prev *widget.Button
	adapt      *widget.Button // Adapt… for the picked passage, beside Next: in sight when the row's is not (a phone's short list)
	prompt     *statusPrompt
	editing    bool              // the source open as text in the pane (the window's source document)
	empty      fyne.CanvasObject // no source
	blank      fyne.CanvasObject // a source with no passages yet
	tools      *fyne.Container   // the count, Previous and Next: hidden without a source
	body       *fyne.Container   // the list, the editor or the empty state
	box        fyne.CanvasObject
}

func newSourcePane(w *MainWindow) *sourcePane {
	s := &sourcePane{w: w, selected: -1, offered: -1, cycleFor: -1, lit: map[int]bool{}, doubt: map[int]bool{},
		ref: sourceRef{Origin: -1}} // none read yet: the first load shows what there is, the empty state too
	s.title = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	s.title.Truncation = fyne.TextTruncateEllipsis
	s.info = widget.NewLabel("")
	s.info.Wrapping = fyne.TextWrapWord
	s.info.SizeName = theme.SizeNameCaptionText
	s.info.Importance = widget.LowImportance
	s.list = widget.NewList(
		func() int {
			if s.hideRows {
				return 0
			}
			return len(s.passages)
		},
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
	s.count = widget.NewLabel("")
	s.count.SizeName = theme.SizeNameCaptionText
	s.count.TextStyle.Bold = true
	s.count.Wrapping = fyne.TextWrapWord // a phone's width: on two lines, not under the buttons
	s.anchorLine = widget.NewLabel("")
	s.anchorLine.SizeName = theme.SizeNameCaptionText
	s.anchorLine.Importance = widget.MediumImportance
	s.anchorLine.Truncation = fyne.TextTruncateEllipsis
	s.next = widget.NewButtonWithIcon("Next", theme.NavigateNextIcon(), w.nextSourcePassage)
	s.next.Importance = widget.LowImportance
	s.next.IconPlacement = widget.ButtonIconTrailingText
	fyne.SetAccessibleLabel(s.next, "Next passage to adapt (picks it; nothing is written)")
	s.prev = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), w.previousSourcePassage)
	s.prev.Importance = widget.LowImportance
	fyne.SetAccessibleLabel(s.prev, "Previous passage to adapt")
	s.adapt = widget.NewButtonWithIcon("Adapt…", adaptIcon, w.adaptSourcePassage)
	s.adapt.Importance = widget.HighImportance
	s.adapt.Disable()
	fyne.SetAccessibleLabel(s.adapt, "Adapt the picked passage: how and where it goes in the script")
	s.prompt = newStatusPrompt(s)
	none := widget.NewLabel(noSourceText())
	none.Wrapping = fyne.TextWrapWord
	s.empty = container.NewVBox(none, widget.NewButtonWithIcon("Choose Source…", theme.FolderOpenIcon(), w.chooseSource))
	blank := widget.NewLabel("The source has no passages yet. Edit writes it here: " +
		"each paragraph you add is a passage to adapt, and Next finds it.")
	blank.Wrapping = fyne.TextWrapWord
	s.blank = container.NewVBox(blank)
	s.body = container.NewStack(newWidthWatch(s.list, s.onWidth))
	head := container.NewBorder(nil, nil, nil, container.NewHBox(s.edit, s.choose), s.title)
	s.tools = container.NewVBox(container.NewBorder(nil, nil, nil, container.NewHBox(s.prev, s.next, s.adapt), s.count), s.anchorLine)
	s.view = newReadingView(s)
	s.learned = newLearnedView(s)
	top := container.NewVScroll(container.NewVBox(head, s.tools, s.prompt.box, s.info, s.learned.box))
	s.box = container.New(sourcePaneLayout{}, top, container.New(bodyReadingLayout{}, s.body, s.view.box))
	return s
}

// newPassageRow is a passage's row: its background (lit), a bar on its
// left (picked), its status mark (adaptation_pane.go), its text, and
// under it a line with Show All for a long one (source_rows.go), the
// scenes linking to it (→ Scene 4), and for the picked one Adapt… and
// Mark as Cut. Nothing in it scrolls, so a drag over it scrolls the list.
// The text and the line are stacked, so a button is never over the text,
// whatever height the list gives the row.
func newPassageRow() *fyne.Container {
	bg := canvas.NewRectangle(color.Transparent)
	bg.CornerRadius = 4
	bar := canvas.NewRectangle(color.Transparent)
	bar.SetMinSize(fyne.NewSize(4, 0))
	text := widget.NewRichText()
	text.Wrapping = fyne.TextWrapWord
	more := newMoreButton()
	more.Hide()
	scenes := newRowButton("", theme.NavigateNextIcon())
	adapt := newRowButton("Adapt…", theme.DocumentCreateIcon())
	adapt.Importance = widget.HighImportance
	cut := newRowButton("Mark as Cut", theme.ContentCutIcon())
	line := container.NewHBox(more, scenes, adapt, cut)
	line.Hide()
	left := container.NewHBox(bar, newStatusMark()) // clear of the list's scroll bar on the right
	return container.NewStack(bg, container.NewBorder(nil, nil, left, nil, container.NewVBox(text, line)))
}

// newRowButton is a small button of a row's line, hidden at first.
func newRowButton(label string, icon fyne.Resource) *widget.Button {
	b := widget.NewButtonWithIcon(label, icon, nil)
	b.Importance = widget.LowImportance
	b.Hide()
	return b
}

// passageRow is a row's parts.
type passageRow struct {
	bg, bar *canvas.Rectangle
	mark    *statusMark
	body    *fyne.Container // text over line
	text    *widget.RichText
	line    *fyne.Container // more, scenes, adapt, cut
	more    *widget.Button
	scenes  *widget.Button // → Scene 4
	adapt   *widget.Button
	cut     *widget.Button
}

func rowParts(row *fyne.Container) passageRow {
	inner := row.Objects[1].(*fyne.Container)
	body, left := inner.Objects[0].(*fyne.Container), inner.Objects[1].(*fyne.Container)
	line := body.Objects[1].(*fyne.Container)
	return passageRow{
		bg: row.Objects[0].(*canvas.Rectangle), bar: left.Objects[0].(*canvas.Rectangle),
		mark: left.Objects[1].(*statusMark), body: body,
		text: body.Objects[0].(*widget.RichText), line: line, more: line.Objects[0].(*widget.Button),
		scenes: line.Objects[1].(*widget.Button), adapt: line.Objects[2].(*widget.Button), cut: line.Objects[3].(*widget.Button),
	}
}

// rowActs is what a row's line shows besides Show All: the scenes
// linking to the passage, and for the picked one Adapt… and Mark as Cut
// (or Unmark Cut).
type rowActs struct {
	scenes     string // the scenes' button ("" none): → Scene 4, → 2 scenes
	scenesName string
	adapt, cut bool
	cutLabel   string
}

func (a rowActs) any() bool { return a.scenes != "" || a.adapt || a.cut }

// fill puts md in a row, shown in full or short, with its line's acts;
// it reports whether it was cut. The row is laid out again: the list may
// have sized it before (with the line hidden, or another passage's text
// in it), and showing or hiding a button does not move it.
func fill(r passageRow, md string, full bool, acts rowActs) bool {
	segs, cut := rowSegments(md, !full)
	r.text.Segments = segs
	r.text.Refresh()
	switch {
	case cut:
		r.more.SetText("Show All")
		r.more.Show()
	case full && md != "":
		if _, short := rowSegments(md, true); short {
			r.more.SetText("Show Less")
			r.more.Show()
		} else {
			r.more.Hide()
		}
	default:
		r.more.Hide()
	}
	setLabel := func(b *widget.Button, on bool, text, name string) {
		setVisible(b, on)
		if on && b.Text != text {
			b.SetText(text)
		}
		if on {
			fyne.SetAccessibleLabel(b, name)
		}
	}
	setLabel(r.scenes, acts.scenes != "", acts.scenes, acts.scenesName)
	setLabel(r.adapt, acts.adapt, "Adapt…", "Adapt the picked passage: how and where it goes in the script")
	cutName := "Mark the picked passage as cut on purpose"
	if acts.cutLabel == "Unmark Cut" {
		cutName = "Unmark the picked passage: not cut"
	}
	setLabel(r.cut, acts.cut, acts.cutLabel, cutName)
	if on := r.more.Visible() || acts.any(); on != r.line.Visible() {
		setVisible(r.line, on)
	} else if on {
		r.line.Layout.Layout(r.line.Objects, r.line.Size()) // a button came or went
	}
	r.body.Layout.Layout(r.body.Objects, r.body.Size())
	return cut
}

// rowKey is what a row shows, to tell whether it has to be filled again.
func rowKey(md string, full bool, acts rowActs) string {
	k := fmt.Sprintf("%v\x00%v\x00", full, acts)
	return k + md
}

// toggleFull is Show All / Show Less on row id.
func (s *sourcePane) toggleFull(id int) {
	if id < 0 || id >= len(s.passages) {
		return
	}
	if s.full == nil {
		s.full = map[int]bool{}
	}
	s.full[id] = !s.full[id]
	if id < len(s.heights) && s.width > 0 {
		s.heights[id] = s.measured(s.passages[id].Markdown, s.full[id], lineActs(s.hasActs(id)), s.width)
		s.exact[id] = true
		s.list.SetItemHeight(id, s.heights[id])
	}
	s.list.RefreshItem(id)
	if !s.full[id] {
		s.list.ScrollTo(id) // back to the row's start, not far below it
	}
}

// tint is a theme colour, see-through.
func tint(name fyne.ThemeColorName, alpha uint8) color.Color {
	r, g, b, _ := theme.Color(name).RGBA()
	return color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), alpha}
}

// rowActs are row id's line's acts as the pane is now.
func (s *sourcePane) rowActs(id int) rowActs {
	var a rowActs
	if sc := s.linkedBy[id]; len(sc) == 1 {
		label := spanLabelsFor(s)[sc[0]]
		a.scenes, a.scenesName = label, "Go to "+label+", which adapts this passage"
	} else if len(sc) > 1 {
		a.scenes, a.scenesName = plural(len(sc), "scene"), fmt.Sprintf("Choose one of the %d scenes adapting this passage", len(sc))
	}
	if id == s.selected {
		switch s.statusOf(id) {
		case statusToDo:
			a.adapt, a.cut, a.cutLabel = true, true, "Mark as Cut"
		case statusCut:
			a.adapt, a.cut, a.cutLabel = true, true, "Unmark Cut"
		}
	}
	return a
}

// show fills a row with passage id.
func (s *sourcePane) show(row *fyne.Container, id int) {
	p := s.passages[id]
	r := rowParts(row)
	bg, bar, text := r.bg, r.bar, r.text
	r.mark.setStatus(s.statusOf(id), s.changed[id])
	r.more.OnTapped = func() { s.toggleFull(id) }
	r.scenes.OnTapped = func() { s.scenesTapped(id, r.scenes) }
	r.adapt.OnTapped = func() {
		s.setPicked(id)
		s.w.adaptSourcePassage()
	}
	r.cut.OnTapped = func() {
		s.setPicked(id)
		s.toggleCut()
	}
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
	if s.shownMD == nil {
		s.shownMD = map[*widget.RichText]string{}
	}
	acts := s.rowActs(id)
	key := rowKey(p.Markdown, s.full[id], acts)
	if k, ok := s.shownMD[text]; !ok || k != key { // wrapping it again is what takes the time
		fill(r, p.Markdown, s.full[id], acts)
		s.shownMD[text] = key
	}
}

// The rows' heights. Measuring a row (wrapping its text) takes about a
// millisecond, all of a novel's rows seconds: so each row's height is
// estimated from its length first, and then measured a stretch at a time
// on the UI goroutine, the rows in view first, with the page given back
// between the stretches. A change of width is waited out (a split being
// dragged) before the rows are measured again.

// widthPause is how long the width has to stay before the rows are
// measured again for it.
const widthPause = 150 * time.Millisecond

// onWidth is the list's width changing.
func (s *sourcePane) onWidth(width float32) {
	if width <= 0 {
		return
	}
	if s.width == 0 || len(s.heights) != len(s.passages) { // the first time: at once
		cancel(&s.widthT)
		s.measure(width)
		return
	}
	s.w.debounce(&s.widthT, widthPause, func() { s.measure(width) })
}

// rowEstimate is how a row's height is estimated at a width: a line's
// height, the next lines', and how many characters fit on a line.
type rowEstimate struct {
	width, one, step, perLine float32
}

// measured is the height of a row showing md (in full or short, with
// its line's acts) at the width.
func (s *sourcePane) measured(md string, full bool, acts rowActs, width float32) float32 {
	if s.measureRow == nil {
		s.measureRow = newPassageRow()
	}
	fill(rowParts(s.measureRow), md, full, acts)
	s.measureRow.Resize(fyne.NewSize(width, 1))
	return s.measureRow.MinSize().Height
}

// estimateFor finds how rows are estimated at a width: one line's row
// and a two lines' row measured, and the characters of the source's
// opening that fit on a line.
func (s *sourcePane) estimateFor(width float32) rowEstimate {
	e := rowEstimate{width: width, one: s.measured("x", true, rowActs{}, width)}
	text := rowParts(s.measureRow).text
	pad := theme.InnerPadding()
	lineW := max(text.Size().Width-2*pad, 10)
	var sample strings.Builder
	for _, p := range s.passages {
		if sample.Len() > 2000 {
			break
		}
		sample.WriteString(p.Plain + " ")
	}
	if sample.Len() == 0 {
		sample.WriteString("The quick brown fox jumps over the lazy dog. ")
	}
	str := sample.String()
	sz := fyne.MeasureText(str, theme.TextSize(), fyne.TextStyle{})
	charW := max(sz.Width/float32(max(utf8.RuneCountInString(str), 1)), 1)
	e.perLine = max(lineW/charW*0.92, 1) // words wrap a little before the edge
	two := strings.Repeat("x ", int(e.perLine*1.5)/2+1)
	e.step = max(s.measured(two, true, rowActs{}, width)-e.one, 1)
	return e
}

// estimate is a passage's row's height, by its length (in full or
// short, Show All or the acts' line under it).
func (e rowEstimate) estimate(p passage, full, line bool) float32 {
	n := utf8.RuneCountInString(p.Plain)
	more := float32(0)
	if !full && n > rowBudget {
		n, more = shortRunes(p), e.one
	} else if line {
		more = e.one
	}
	lines := max(1, int(math.Ceil(float64(float32(n)/e.perLine))))
	return e.one + float32(lines-1)*e.step + more
}

// measure sets the rows' heights for the list's width: estimated now,
// measured over time (measureStretch).
func (s *sourcePane) measure(width float32) {
	if width <= 0 {
		return
	}
	s.width = width
	s.measuring++
	s.scanned = 0
	s.est = s.estimateFor(width)
	s.heights = make([]float32, len(s.passages))
	s.exact = make([]bool, len(s.passages))
	s.acts = make([]bool, len(s.passages))
	for id, p := range s.passages {
		s.acts[id] = s.hasActs(id)
		s.heights[id] = s.est.estimate(p, s.full[id], s.acts[id])
	}
	s.setHeights(0, len(s.passages))
	gen := s.measuring
	s.w.repeat(&s.measureT, func() bool { return s.measureStretch(gen) })
}

// setHeights gives the list the heights of rows [from, to). The list
// lays itself out again at each one, so with many it is told it has no
// rows meanwhile (its scroll offset kept).
func (s *sourcePane) setHeights(from, to int) {
	if to-from > 8 {
		off := s.list.GetScrollOffset()
		s.hideRows = true
		for id := from; id < to; id++ {
			s.list.SetItemHeight(id, s.heights[id])
		}
		s.hideRows = false
		s.list.Refresh()
		s.list.ScrollToOffset(off)
		return
	}
	for id := from; id < to; id++ {
		s.list.SetItemHeight(id, s.heights[id])
	}
}

// measureStretch measures rows for a while: those in view first, then on
// from the first not measured; it reports whether rows are left.
func (s *sourcePane) measureStretch(gen int) bool {
	if gen != s.measuring || len(s.exact) != len(s.passages) {
		return false
	}
	start := time.Now()
	one := func(id int) {
		if s.exact[id] {
			return
		}
		s.exact[id] = true
		if h := s.measured(s.passages[id].Markdown, s.full[id], lineActs(s.acts[id]), s.width); h != s.heights[id] {
			s.heights[id] = h
			s.list.SetItemHeight(id, h)
		}
	}
	first := s.firstInView()
	for id := first; id < min(first+40, len(s.passages)); id++ {
		one(id)
	}
	for ; s.scanned < len(s.passages); s.scanned++ {
		if time.Since(start) > pace.Work {
			return true
		}
		one(s.scanned)
	}
	return false
}

// firstInView is the first row in the list's view.
func (s *sourcePane) firstInView() int {
	top, pad := s.list.GetScrollOffset(), theme.Padding()
	y := float32(0)
	for id, h := range s.heights {
		if y+h > top {
			return id
		}
		y += h + pad
	}
	return 0
}

// load reads the window's script's source again if it is another one
// now, or it changed: its block in the script, or its file's time or
// size.
func (s *sourcePane) load() { s.loadSource(false) }

// sourceRead is a source read: where it is, its file's time and size,
// its Markdown.
type sourceRead struct {
	ref     sourceRef
	modTime time.Time
	size    int64
	src     string
	err     error
}

func (r *sourceRead) same(o *sourceRead) bool {
	return o != nil && r.ref == o.ref && r.modTime.Equal(o.modTime) && r.size == o.size &&
		sameString(r.src, o.src) && (r.err == nil) == (o.err == nil)
}

// backgroundSource is how long a source is (bytes) before it is parted
// into passages in the background: a short one is at once.
const backgroundSource = 64 << 10

// loadSource is load; in the background (the pane follows the cursor), a
// long source is parted into passages in a goroutine, the pane as it was
// meanwhile, and follows again when they are in.
func (s *sourcePane) loadSource(background bool) {
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
	if open := s.w.srcDoc; ref.Origin == sourceInside || ref.Origin == sourceBundle || open != nil && open.ref == ref { // in memory (an open document's text)
		src, _ = s.w.sourceText(ref)
		if ref == s.ref && sameString(src, s.src) {
			s.loading = nil
			return
		}
	} else {
		if ref == s.ref && mod.Equal(s.modTime) && size == s.size && s.err == nil {
			s.loading = nil
			return
		}
		src, err = s.w.sourceText(ref)
	}
	r := &sourceRead{ref: ref, modTime: mod, size: size, src: src, err: err}
	if !background || len(src) < backgroundSource {
		s.loading = nil
		s.use(r, passagesOf(src))
		return
	}
	if r.same(s.loading) {
		return // being read
	}
	s.loading = r
	inBackground(func() []passage { return passagesOf(r.src) }, func(ps []passage) {
		if s.loading != r {
			return // another one since
		}
		s.loading = nil
		s.use(r, ps)
		s.follow()
	})
}

// use shows a source read, parted into passages.
func (s *sourcePane) use(r *sourceRead, ps []passage) {
	if (r.ref.Origin == sourceInside) != (s.ref.Origin == sourceInside) {
		s.w.refreshMenu() // Keep Source in the Script (an undo can change it)
	}
	if r.ref != s.ref { // another source: its offers are new, the last one's question gone
		s.sugg, s.seedAsked = suggestions{}, false
		s.prompt.hide()
		s.linkedWords = nil
	}
	s.ref, s.modTime, s.size, s.src, s.err = r.ref, r.modTime, r.size, r.src, r.err
	s.fresh = true
	s.passages = ps
	s.model, s.modelFor, s.matcher = nil, "", nil
	s.reading = nil
	if s.selected >= 0 { // found again by its words, or none
		old := s.selected
		s.selected = -1
		for d := 0; d < len(s.passages) && s.selWords != nil; d++ { // the nearest of those with them
			if q := old - d; q >= 0 && q < len(s.passages) && slices.Equal(s.passages[q].Words, s.selWords) {
				s.selected = q
				break
			}
			if q := old + d; q < len(s.passages) && slices.Equal(s.passages[q].Words, s.selWords) {
				s.selected = q
				break
			}
		}
		if s.selected < 0 {
			s.selWords, s.pickFresh = nil, false
		}
	}
	s.linkedBy = nil
	s.lit, s.doubt = map[int]bool{}, map[int]bool{}
	s.full = nil
	s.statuses = nil
	s.offeredPassage() // found again by its words, or none
	if s.offered < 0 {
		s.offeredWords = nil
	}
	s.showBody()
	if s.w.swapped != "source" { // its rows are not in sight while the source is in the main editor (swap.go)
		s.measure(s.width)
	}
	s.showReading(false)
}

// noSourceText is what the pane says without a source: where one comes
// from (the browser has no folder beside the script).
func noSourceText() string {
	t := "No source yet. Choose the story this script adapts: Markdown, Word, OpenDocument, a web page or text."
	if !inBrowser {
		t += " A Markdown file beside the script with its name (story.md for story.fountain) is its source too."
	}
	return t
}

// showBody shows the list, the source's text being edited, or what to
// do without a source.
func (s *sourcePane) showBody() {
	var o fyne.CanvasObject
	s.title.SetText(s.ref.describe())
	s.tools.Show()
	switch {
	case s.ref.Origin == sourceNone:
		s.edit.Hide()
		s.tools.Hide() // nothing to count or go to
		o = s.empty
	case s.editing && s.w.srcDoc != nil:
		o = s.w.srcDoc.view
	case !slices.ContainsFunc(s.passages, passage.linkable) && s.err == nil:
		s.edit.Show()
		o = s.blank
	default:
		s.edit.Show()
		o = s.body.Objects[0]
		if wl, ok := o.(*widthWatch); !ok || wl.content != s.list {
			o = newWidthWatch(s.list, s.onWidth)
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
	if s.model == nil || !sameString(s.modelFor, text) {
		if s.matcher == nil || !samePassages(s.matcher.ps, s.passages) {
			s.matcher = matcherFor(s.passages)
		}
		s.model, s.modelFor = newSourceModelWith(s.matcher, text, s.w.sceneStarts()), text
	}
	return s.model
}

// changedIn reports whether a passage the link found is in changed.
func changedIn(changed map[int]bool, mt linkMatch) bool {
	for p := mt.Passages[0]; p <= mt.last(); p++ {
		if changed[p] {
			return true
		}
	}
	return false
}

// samePassages reports whether a and b are the same passages, not
// read again since.
func samePassages(a, b []passage) bool {
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0])
}

// shown reports whether the source is in sight.
func (s *sourcePane) shown() bool {
	return s.w.paneInSight("source")
}

// follow lights the passages the cursor's scene links to, scrolls to
// them if none is in sight, and says what of the scene's links was not
// found.
func (s *sourcePane) follow() {
	cancel(&s.w.followTimer) // done now
	s.w.learnFromWriting()
	if !s.shown() {
		return
	}
	s.loadSource(true)
	if s.loading != nil { // being read: the pane follows when it is in
		s.title.SetText(s.loading.ref.describe())
		s.setInfo("Reading the source…")
		return
	}
	if s.ref.Origin == sourceNone {
		s.setInfo("")
		s.count.SetText("")
		s.learned.box.Hide()
		s.learned.ok = false
		s.prompt.hide()
		s.next.Disable()
		s.prev.Disable()
		return
	}
	s.next.Enable()
	s.prev.Enable()
	m := s.current()
	s.showStatuses(m)
	s.showAnchor(m)
	s.learned.update(s)
	s.offerStart(m)
	line, _ := s.w.textEditor.Buffer().Position(s.w.textEditor.CursorOffset())
	lit, doubt := map[int]bool{}, map[int]bool{}
	var problems []string
	first := -1
	var names map[string]bool // the source's names, if a link names a file
	for _, i := range m.inSpan(line) {
		mt := m.matches[i]
		q := "“" + shortQuote(m.links[i].Quote) + "”"
		if f := m.links[i].Ref.File; f != "" {
			if names == nil {
				names = s.w.sourceNames(s.ref)
			}
			if !names[sourceNameKey(f)] {
				problems = append(problems, q+" names "+f+", not this script's source: read against it")
			}
		}
		switch mt.Kind {
		case linkFound, linkNear:
			for p := mt.Passages[0]; p <= mt.last(); p++ {
				lit[p] = true
			}
			if first < 0 || mt.Passages[0] < first {
				first = mt.Passages[0]
			}
			if mt.Kind == linkNear || changedIn(s.changed, mt) {
				problems = append(problems, q+": the source changed since it was linked; "+
					"check the scene, then Link to Source Passage again")
			}
		case linkAmbiguous:
			for _, p := range mt.Passages {
				doubt[p] = true
			}
			problems = append(problems, fmt.Sprintf("%s fits %d passages: quote more of it", q, len(mt.Passages)))
		case linkMissing:
			if bad := m.links[i].Ref.Bad; bad != "" {
				problems = append(problems, "A link cannot be read: "+bad)
			} else {
				problems = append(problems, q+" is not in the source")
			}
		}
	}
	span := spanAt(m.spans, line)
	where := span.Heading
	if where == "" {
		where = "Before the first scene"
	}
	info := where + ": " + plural(len(lit), "passage")
	if len(lit) == 0 {
		info = where + ": no passage linked"
	}
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
	if s.fresh && first < 0 && s.selected < 0 { // just read: where the adapting goes on
		if last, _ := m.lastLinked(); last >= 0 {
			if next, _ := m.nextToDo(last); next >= 0 && !s.inSight(map[int]bool{next: true}) {
				s.scrollToTop(max(next-1, 0)) // with the one before it, for where it follows on
			}
		}
	}
	if s.list.Size().Height > 0 {
		s.fresh = false
	}
}

// followPause is how long typing pauses before the source pane follows
// the text: the links are read again then, once.
const followPause = 200 * time.Millisecond

// followSource has the source pane follow the cursor: at once when it
// moves, once after a pause when the text changes (the cursor and the
// text both say they changed on a keystroke).
func (w *MainWindow) followSource() {
	if w.source == nil { // made once shown: it learns from the writing hidden too (follow)
		return
	}
	s := w.source
	rev := w.textEditor.Buffer().Rev()
	edited := rev != w.seenRev
	w.seenRev = rev
	if edited || w.followTimer != nil {
		w.debounce(&w.followTimer, followPause, s.follow)
		return
	}
	s.follow()
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

// scrollToTop scrolls the list so that row id is at its top.
func (s *sourcePane) scrollToTop(id int) {
	if len(s.heights) != len(s.passages) {
		s.list.ScrollTo(id)
		return
	}
	y := float32(0)
	for _, h := range s.heights[:min(id, len(s.heights))] {
		y += h + theme.Padding()
	}
	s.list.ScrollToOffset(y)
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

// pick is a tap on passage id: it is the picked one (for Adapt…, Mark
// as Cut, Link to Source Passage) and where Next goes on from; the
// scenes linking to it are the button under it.
func (s *sourcePane) pick(id int) {
	if id < 0 || id >= len(s.passages) {
		return
	}
	s.setPicked(id)
	switch sc := s.current().linked(id); {
	case len(sc) > 0:
		s.w.setStatus("Picked; " + spanLabelsFor(s)[sc[0]] + " adapts it: → under it goes there")
	case s.statusOf(id) == statusToDo || s.statusOf(id) == statusCut:
		s.w.setStatus("Picked: Adapt… under it writes it into the script; Next goes on after it")
	default:
		s.w.setStatus("Picked")
	}
}

// setPicked makes passage id the picked one, the anchor until the cursor
// moves.
func (s *sourcePane) setPicked(id int) {
	if id < 0 || id >= len(s.passages) {
		return
	}
	s.selected, s.selWords = id, s.passages[id].Words
	s.pickFresh, s.pickCursor = true, s.w.textEditor.CursorOffset()
	s.list.Refresh()
	s.syncActs()
	s.showReading(true)
	if s.ref.Origin != sourceNone {
		s.showAnchor(s.current())
	}
}

// scenesTapped is the → under passage id: to its scene, or a list of
// them to choose from.
func (s *sourcePane) scenesTapped(id int, from fyne.CanvasObject) {
	s.setPicked(id)
	scenes := s.current().linked(id)
	switch len(scenes) {
	case 0:
		return
	case 1:
		s.jumpTo(id, 0)
		return
	}
	labels := spanLabelsFor(s)
	m := s.current()
	var items []*fyne.MenuItem
	for k, l := range scenes {
		k := k
		items = append(items, fyne.NewMenuItem(labels[l]+spanHeading(spanAt(m.spans, l)), func() { s.jumpTo(id, k) }))
	}
	c := s.w.fyneWindow.Canvas()
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(from).Add(fyne.NewPos(0, from.Size().Height))
	widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", items...), c, pos)
}

// jumpTo goes to the k-th scene linking to passage id: the script's
// cursor on it (the Script tab lighting it, if the script is swapped
// out), the scene flashing; the passage stays picked, the anchor.
func (s *sourcePane) jumpTo(id, k int) {
	scenes := s.current().linked(id)
	if k < 0 || k >= len(scenes) {
		return
	}
	w := s.w
	line := scenes[k]
	s.jumping = true
	w.goToScript(line)
	s.jumping = false
	s.cycleFor, s.cycle = id, k
	s.selected, s.selWords = id, s.passages[id].Words
	s.pickFresh, s.pickCursor = true, w.textEditor.CursorOffset()
	w.flashScene(line)
	label := spanLabelsFor(s)[line]
	if len(scenes) > 1 {
		w.setStatus(fmt.Sprintf("%s, %d of %d from this passage (Go to the Passage's Scene again: the next)", label, k+1, len(scenes)))
	} else {
		w.setStatus(label + ", from this passage")
	}
	s.showAnchor(s.current())
}

// showAdapt enables Adapt… beside Next while the picked passage can be
// adapted.
func (s *sourcePane) showAdapt() {
	if st := s.statusOf(s.selected); st == statusToDo || st == statusCut {
		s.adapt.Enable()
	} else {
		s.adapt.Disable()
	}
}

// syncActs gives the rows whose line came or went (a link to them, the
// pick) their height again.
//
// Many at once (a source read again: its links found after it was
// measured without them) are estimated a line taller or shorter and
// measured again a stretch at a time, as the rows were at first.
func (s *sourcePane) syncActs() {
	s.showAdapt()
	if len(s.acts) != len(s.passages) || len(s.heights) != len(s.passages) {
		return
	}
	var flipped []int
	for id := range s.passages {
		if s.hasActs(id) != s.acts[id] {
			flipped = append(flipped, id)
		}
	}
	if len(flipped) > 8 && s.width > 0 { // a line more or less, measured later
		for _, id := range flipped {
			s.acts[id] = !s.acts[id]
			if s.acts[id] {
				s.heights[id] += s.est.one
			} else {
				s.heights[id] = max(s.heights[id]-s.est.one, s.est.one)
			}
			s.exact[id] = false
		}
		s.setHeights(0, len(s.passages))
		s.scanned = min(s.scanned, flipped[0])
		if s.measureT == nil {
			gen := s.measuring
			s.w.repeat(&s.measureT, func() bool { return s.measureStretch(gen) })
		}
		return
	}
	for _, id := range flipped {
		s.acts[id] = !s.acts[id]
		if s.width > 0 {
			s.heights[id], s.exact[id] = s.measured(s.passages[id].Markdown, s.full[id], lineActs(s.acts[id]), s.width), true
		} else {
			s.heights[id] = s.est.estimate(s.passages[id], s.full[id], s.acts[id])
		}
		s.list.SetItemHeight(id, s.heights[id])
	}
}

// hasActs reports whether row id has a line of acts (rowActs) under its
// text: the scenes linking to it, or the pick's Adapt… and Mark as Cut.
func (s *sourcePane) hasActs(id int) bool {
	if len(s.linkedBy[id]) > 0 {
		return true
	}
	return id == s.selected && (s.statusOf(id) == statusToDo || s.statusOf(id) == statusCut)
}

// lineActs are acts that make the line show, or none: for a row's height
// (the line's buttons are one row high, whatever they say).
func lineActs(on bool) rowActs {
	if on {
		return rowActs{scenes: "→"}
	}
	return rowActs{}
}

// spanLabelsFor are the script's spans' names (spanLabels), made again
// when the script changed.
func spanLabelsFor(s *sourcePane) map[int]string {
	text := s.w.textEditor.Text()
	if s.labels == nil || !sameString(s.labelsFor, text) {
		s.labels, s.labelsFor = spanLabels(text, s.w.sceneStarts()), text
	}
	return s.labels
}

// toggleEdit opens the source as text in the pane, or saves it and
// shows it as passages again. The text is the window's source document
// (document.go), the one Swap with Main puts in the main editor.
func (s *sourcePane) toggleEdit() {
	if s.ref.Origin == sourceNone {
		return
	}
	if !s.editing {
		d := s.w.sourceDoc()
		if d == nil {
			return
		}
		s.editing = true
		s.edit.SetText("Done")
		s.edit.SetIcon(theme.ConfirmIcon())
		fyne.SetAccessibleLabel(s.edit, "Done: save the source")
		s.showBody()
		s.showReading(false)
		s.w.fyneWindow.Canvas().Focus(d.ed)
		return
	}
	d := s.w.srcDoc
	s.w.offerLinkUpdate(d) // its passages' openings changed: the links quoting them
	if err := s.w.closeDoc(d); err != nil {
		dialog.ShowError(err, s.w.fyneWindow)
		return
	}
	if s.editing { // no document (another script since): the pane's state alone
		s.endEdit()
	}
}

// endEdit shows the source as passages again (its document saved and
// let go: closeDoc).
func (s *sourcePane) endEdit() {
	s.editing = false
	s.edit.SetText("Edit")
	s.edit.SetIcon(theme.DocumentCreateIcon())
	fyne.SetAccessibleLabel(s.edit, "Edit the source as text")
	s.modTime = time.Time{} // read again
	s.load()
	s.showBody()
	s.follow()
}

// focusTarget is what takes the keyboard in the pane.
func (s *sourcePane) focusTarget() fyne.Focusable {
	if s.editing && s.w.srcDoc != nil {
		return s.w.srcDoc.ed
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
	if w.source == nil {
		w.source = newSourcePane(w)
	}
	return w.source
}

// chooseSource is Adaptation > Choose Source…: a story to be the script's
// source, kept by Accolade (source_file.go) and remembered for it.
func (w *MainWindow) chooseSource() {
	exts := []string{".md", ".markdown", ".txt", ".docx", ".odt", ".html", ".htm", ".xhtml"}
	w.chooseFile(exts, func(path string) { w.chooseSourceFile(path) })
}

// chooseSourceFile makes path the script's source: converted in the
// background (a Word novel took seconds in the browser), then kept for
// the script it was chosen for, if the window still has that one.
func (w *MainWindow) chooseSourceFile(path string) {
	type read struct {
		md  string
		err error
	}
	w.sourceGen++
	gen, script, bundle := w.sourceGen, w.currentFile, w.bundle
	w.setStatus("Reading " + filepath.Base(path) + "…")
	inBackground(func() read {
		md, _, _, err := toMarkdown(path)
		return read{md, err}
	}, func(r read) {
		if w.takeChosenSource(gen, script, bundle, path, r.md, r.err) {
			w.showSource()
			if w.bundle != nil {
				w.setStatus("Source: " + filepath.Base(path) + ", put in " + filepath.Base(w.bundle.path))
			} else {
				w.setStatus("Source: " + filepath.Base(path) + ", kept by Accolade")
			}
		}
	})
}

// takeChosenSource makes what Choose Source read the script's source, if
// it is still for the script it was chosen for (gen: no later Choose
// Source; the same file and bundle open) and could be read.
func (w *MainWindow) takeChosenSource(gen int, script string, bundle *textBundle, path, md string, err error) bool {
	if gen != w.sourceGen {
		return false // another one chosen since
	}
	if w.currentFile != script || w.bundle != bundle {
		w.setStatus("Another script is open now: " + filepath.Base(path) + " was not made its source")
		return false
	}
	if err == nil {
		err = w.useSourceText(path, md)
	}
	if err != nil {
		dialog.ShowError(fmt.Errorf("could not use %s as the source: %v", filepath.Base(path), err), w.fyneWindow)
		return false
	}
	return true
}

// showSource shows the source pane, the source read again.
func (w *MainWindow) showSource() {
	s := w.sourcePane()
	s.ref = sourceRef{Origin: -1} // read again
	if !s.shown() {
		w.showPane("source", true)
	}
	s.follow()
}

// saveSourceAs is Adaptation > Save Source As…: the source, wherever it is
// kept, written out as Markdown (the browser downloads it).
func (w *MainWindow) saveSourceAs() {
	r := w.sourceRef()
	if r.Origin == sourceNone {
		w.setStatus("No source: Adaptation > Choose Source…")
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

// toggleSourceInside is Adaptation > Keep Source in the Script: the source
// moved into the script's end as a boneyard block, or out of it again.
func (w *MainWindow) toggleSourceInside() {
	in := w.sourceRef().Origin != sourceInside
	d := w.srcDoc
	if d != nil { // open: what is being written moves with it, its home moving (not going)
		if err := d.store(); err != nil {
			dialog.ShowError(err, w.fyneWindow)
			return
		}
		d.moving = true
	}
	err := w.keepSourceInside(in)
	if d != nil {
		d.moving = false
		if w.srcDoc == d {
			d.rehome() // kept where the source is now
		}
	}
	if err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return
	}
	if in {
		w.setStatus("The source is kept in the script now, at its end")
	} else {
		w.setStatus("The source is kept by Accolade now, out of the script")
	}
	if w.source != nil {
		w.source.follow()
	}
	w.shareSourceAnew() // shared: in the script now, or out of it in a Doc of its own (collab_source.go)
}

// linkToSource is Adaptation > Link to Source Passage: a link to the passage
// picked in the source pane, at the cursor.
func (w *MainWindow) linkToSource() {
	if w.scriptAway("Link to Source Passage") { // the script is in the side panes (swap.go)
		return
	}
	if w.markdown {
		return
	}
	s := w.sourcePane()
	if !s.shown() {
		w.showPane("source", true)
		w.setStatus("Pick the passage in the source, then Link to Source Passage again")
		return
	}
	s.load()
	if s.selected < 0 || s.selected >= len(s.passages) || !s.passages[s.selected].linkable() {
		w.setStatus("Pick the passage in the source first")
		return
	}
	s.linkPassage(s.selected)
}

// linkPassage puts a link to passage p at the script's cursor: one undo
// step, with its cut entry taken away if it was cut.
func (s *sourcePane) linkPassage(p int) {
	w := s.w
	d, unique := fragmentFor(matcherFor(s.passages), p, p)
	note, quote := fragmentNote("", []textDirective{d}), d.Start
	e := w.textEditor
	at, insert, after := insertLink(e.Text(), e.CursorOffset(), note, w.sceneStarts())
	m := s.current()
	uncut := unique && m.status[p] == statusCut // an ambiguous link does not adapt it
	e.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			if uncut { // linked now, so not cut: its entry goes (at the end, after the link's place)
				inside, _ := adaptationBlock(b.Text())
				start, end, block := withAdaptationBlock(b.Text(), withCuts(inside, s.cutsWithout(m, p)))
				if start >= at {
					b.Replace(start, end, block)
				}
			}
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

// sectionsFromSourceHere is Adaptation > Source Headings as Sections: the
// source's headings, those not in the script yet, as its sections.
func (w *MainWindow) sectionsFromSourceHere() {
	if w.scriptAway("Source Headings as Sections") { // the script is in the side panes (swap.go)
		return
	}
	if w.markdown {
		return
	}
	s := w.sourcePane()
	s.load()
	if s.ref.Origin == sourceNone {
		w.setStatus("No source: Adaptation > Choose Source…")
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

// showLearned is Adaptation > What Accolade Learned…: the source pane,
// with its list of what was learned open.
func (w *MainWindow) showLearned() {
	if w.markdown {
		return
	}
	s := w.sourcePane()
	if !s.shown() {
		w.showPane("source", true)
	}
	if w.sourceRef().Origin == sourceNone {
		w.setStatus("No source: Adaptation > Choose Source…")
		return
	}
	s.learned.update(s)
	if s.learned.n == 0 {
		w.setStatus("Accolade has learned nothing from your writing yet")
		return
	}
	s.learned.open = true
	s.learned.layout()
}
