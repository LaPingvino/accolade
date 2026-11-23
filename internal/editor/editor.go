// Package editor is Accolade's own script editor: a text model (buffer)
// wrapped into rows (wrap) and drawn in a Fyne TextGrid, so selection,
// highlights and single-step undo are under Accolade's control. See
// docs/EDITOR_WIDGET.md; this is step 3 (display and keyboard).
package editor

import (
	"github.com/LaPingvino/lexington/fountain"

	"fmt"
	"image/color"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/lex"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/accolade/internal/editor/wrap"
)

// ScriptEditor edits a screenplay in a monospace grid.
type ScriptEditor struct {
	widget.BaseWidget

	// OnChanged is called with the text after every edit.
	OnChanged func(text string)
	// OnEnter, when set, handles Enter instead of inserting a line break
	// (Accolade formats the completed line); it reports whether it did.
	OnEnter func() bool
	// OnRune, when set, handles a typed character instead of inserting it
	// (Accolade closes brackets); it reports whether it did.
	OnRune func(r rune) bool
	// OnTab, when set, handles Tab instead of indenting (Accolade
	// completes names); it reports whether it did.
	OnTab func() bool
	// OnKey, when set, sees every key first (a vim mode: Escape); it
	// reports whether it handled the key.
	OnKey func(k *fyne.KeyEvent) bool
	// OnKeyUp, if set, sees each key let go (Alt alone: the ribbon's key
	// tips); OnFocusLost the keyboard going elsewhere
	OnKeyUp     func(k *fyne.KeyEvent)
	OnFocusLost func()
	// OnTaggedStep, when set, is asked before a tagged step (TagLastStep)
	// is undone (redo false) or redone, by any way in (a menu, Ctrl+Z,
	// vim's u): it does what the step means beyond its text (Accolade's
	// Issue: the drafts store), or reports false to keep it, the text
	// untouched.
	OnTaggedStep func(tag any, redo bool) bool
	// OnShortcut, when set, sees shortcuts first (vim's Ctrl+R); it
	// reports whether it handled the shortcut.
	OnShortcut func(s fyne.Shortcut) bool
	// SceneStarts are what scene headings start with in the script's
	// language, for the colours (nil: English)
	SceneStarts []string
	// Dialect is the script's language's rules as Lexington reads them
	// (its heading patterns, cue and transition rules): nil reads by
	// SceneStarts as English Fountain.
	Dialect *fountain.Dialect
	casts   *fountain.CastCache // Dialect's, the text's cast kept between keys
	// LanguageRTL is the script's language's direction (Lexington's
	// [Directions]): a line with no direction mark and no strong letter
	// reads right to left, laid out mirrored. Every other line follows its
	// own letters.
	LanguageRTL bool
	// Format, when set, lays out each line as its element (its column,
	// width and alignment), instead of by its leading spaces.
	Format func(syntax.Kind) wrap.Format
	// Folds are the line ranges hidden (nil: none): the lines under a
	// folded heading, or a block folded to one line. A fold the cursor
	// gets into opens: its heading's line goes to OnUnfold.
	Folds    func() []Fold
	OnUnfold func(heading int)
	// FoldLabel is what a folded block's line shows instead of its text
	// (Fold.Block), asked only for the rows drawn.
	FoldLabel func(heading int) string
	// Chips, when set, are stretches of the text drawn as a short label
	// (Accolade's source links: "↗ source"), sorted and apart, unless the
	// cursor is inside one (Start < cursor < End): that one is drawn as
	// its text. Asked at each layout, with the editor's text: the same
	// slice back for the same text (a new one re-lays the text out). A
	// screenplay's only: not for Markdown, nor on a line read right to left.
	Chips func(text string) []Chip
	// OnChipPressed is told of a right click or a long press on a chip,
	// with its cell's place and size in the editor (to show its Tip).
	OnChipPressed func(c Chip, at fyne.Position, cell fyne.Size)
	// Headings, when set, are the lines that can fold: a gutter column
	// shows ▾ on them (▸ folded), and a tap there goes to OnFoldTapped.
	Headings     func() map[int]bool
	OnFoldTapped func(line int)
	// Marks, when set, are the lines with a mark in a colour of their
	// own: a bar beside them in the fold marks' column (production's
	// revision colours). Read at each layout, as Headings.
	Marks func() map[int]color.Color
	// Inks, when set, are words drawn in a colour of their own and
	// underlined (production's breakdown tags), by line: each a range of
	// the line's columns (runes). Read at each layout, as Marks.
	Inks func() map[int][]Ink
	// Tags, when set, are labels drawn faintly after a line's text (a
	// guess beside a scene heading: "↗ link?"), by line. Not text: the
	// cursor never goes into one and nothing of it is in the buffer; a
	// tap (or a right click, a long press) on it goes to OnTagTapped, with
	// its first cell's place and size. Read at each layout, as Marks. A
	// tag that does not fit after the line's last row is not drawn.
	Tags        func() map[int]Tag
	OnTagTapped func(line int, t Tag, at fyne.Position, cell fyne.Size)
	// OnSecondaryTapped, when set, is told of a right click (a long press
	// on a touch screen) at a position on the canvas: the cursor is put
	// there first, unless it is in the selection.
	OnSecondaryTapped func(at fyne.Position)
	// OnShiftTab handles Shift+Tab, and reports whether it did.
	OnShiftTab func() bool
	// PageColumns, when set, is the page's width (NewPageScroll's columns).
	PageColumns int
	// OnCursorChanged is called when the cursor or selection moves.
	OnCursorChanged func()
	// OnHover, when set, is told of the character under the pointer as it
	// moves over the text: its offset (-1: none, or the pointer gone), and
	// its cell's place and size in the editor.
	OnHover func(offset int, at fyne.Position, cell fyne.Size)
	// AccessibleName is what screen readers call the editor ("Script").
	AccessibleName string

	buf     *buffer.Buffer
	layout  wrap.Layout
	rtl     []bool // per line: it reads right to left (drawn mirrored)
	grid    *runGrid
	columns int
	focused bool
	shift   bool
	// preferred column for moving up and down
	goalCol int
	// dragging: selecting from dragFrom
	dragging bool
	dragFrom int

	// highlights are ranges shown with a highlight background (search
	// matches); [start, end) rune offsets
	highlights [][2]int
	// flash is a range shown lit for a moment (a scene gone to from the
	// source pane: SetFlash); [start, end) rune offsets, empty for none
	flash [2]int
	// Syntax colours the text by Fountain element
	Syntax bool
	// Markdown: the text is Markdown, coloured as such and wrapped as
	// prose (Format is not used)
	Markdown bool
	// lineNumbers shows each line's number in a gutter (SetLineNumbers)
	lineNumbers bool

	// onCursorMoved is told where the cursor is (see NewScroll)
	onCursorMoved func(cursor fyne.Position, height float32)

	// In a scroll (NewScroll) the grid holds only the rows in view, and
	// some around them: a whole long script's rows took seconds to lay
	// out at every resize and keystroke. viewTop and viewHeight are the
	// part of the editor in view; gridFirst is the grid's first row.
	viewed              bool
	onView              func() // the view again (after a resize)
	viewTop, viewHeight float32
	gridFirst, gridEnd  int
	draw                drawing                             // what relayout found, to draw rows with
	foldAt              map[int]bool                        // the folded headings' lines
	blockAt             map[int]bool                        // those of blocks, drawn as their labels
	foldRoom            []wrap.Row                          // room for the rows the folds leave
	headings            map[int]bool                        // the lines that can fold (Headings)
	marks               map[int]*widget.CustomTextGridStyle // the lines' marks (Marks), as drawn
	inks                map[int][]Ink                       // the lines' inked words (Inks)
	tags                map[int]Tag                         // the lines' tags (Tags)
	chips               []Chip                              // the text's (Chips), as asked last
	laid                laidOut                             // the last layout, for its key
	// offstage: out of sight (SetOffstage); stale: changed since laid out
	offstage, stale bool
}

// Chip is a stretch of the text, [Start, End) in runes, drawn as Label
// (Chips); Tip is the caller's (what a hover shows).
type Chip struct {
	Start, End int
	Label, Tip string
	Warn       bool // drawn in the warning colour (a link that cannot find its passage alone)
}

// Tag is a label after a line (Tags); Short is drawn instead where Label
// has no room (a phone's long heading); Tip is the caller's.
type Tag struct {
	Label, Short, Tip string
}

// Fold is a fold of lines: those in [From, To) hidden under the
// heading, line From-1. A Block's heading is its first line: the cursor
// on it is inside the fold too (it opens), and while folded the line is
// drawn as FoldLabel's summary, one row, instead of its text.
type Fold struct {
	From, To int
	Block    bool
}

// Ink is words of a line drawn in Color and underlined: the line's
// columns [Start, End), in runes.
type Ink struct {
	Start, End int
	Color      color.Color
}

// drawing is what rows are drawn with: the text, its kinds and styles.
type drawing struct {
	text                                     []rune
	kinds                                    []syntax.Kind
	inline                                   map[int][]syntax.Span // Markdown's emphasis, by line
	lineStart                                []int                 // each line's first offset
	th                                       fyne.Theme
	v                                        fyne.ThemeVariant
	selected, cursor, highlight, number      *widget.CustomTextGridStyle
	chip, chipWarn                           *widget.CustomTextGridStyle // a chip's label; a warning chip's
	flash                                    color.Color
	selStart, selEnd, curRow, curCol, gutter int
	others                                   []buffer.Span // more selections
	otherCursors                             map[int][]int // their cursors' columns by row
}

// viewMargin is how many rows beyond the view the grid holds, so that
// scrolling a little needs no redrawing.
const viewMargin = 60

// New creates an editor holding text.
func New(text string) *ScriptEditor {
	e := &ScriptEditor{buf: buffer.New(text), columns: 80, goalCol: -1}
	e.grid = newRunGrid(e)
	e.ExtendBaseWidget(e)
	e.relayout()
	return e
}

// Buffer gives access to the text model (selection, Replace, Group, undo).
func (e *ScriptEditor) Buffer() *buffer.Buffer { return e.buf }

// Text is the edited text.
func (e *ScriptEditor) Text() string { return e.buf.Text() }

// SetText replaces the text and forgets the undo history; OnChanged is
// called, as Fyne's Entry does.
func (e *ScriptEditor) SetText(text string) {
	e.buf.SetText(text)
	e.changed(true)
}

// SetLineNumbers shows or hides line numbers in a gutter on the left.
func (e *ScriptEditor) SetLineNumbers(on bool) {
	e.lineNumbers = on
	e.relayout()
}

// gutter is the number of columns the line numbers take (0 when hidden):
// the digits of the highest line number and a space.
func (e *ScriptEditor) gutter() int {
	return e.numbersWidth() + e.foldWidth()
}

// GutterWidth is how many columns come before the text: the line numbers
// and the fold marks.
func (e *ScriptEditor) GutterWidth() int { return e.gutter() }

// numbersWidth is the line numbers' part of the gutter.
func (e *ScriptEditor) numbersWidth() int {
	if !e.lineNumbers {
		return 0
	}
	return len(strconv.Itoa(e.buf.LineCount())) + 1
}

// foldWidth is the fold marks' part of the gutter, after the numbers.
func (e *ScriptEditor) foldWidth() int {
	if e.Headings == nil {
		return 0
	}
	return 2
}

// SetHighlights marks ranges of the text, such as all search matches, with
// a highlight background; nil clears them.
func (e *ScriptEditor) SetHighlights(ranges [][2]int) {
	e.highlights = ranges
	e.relayout()
}

// Highlights are the ranges SetHighlights marked.
func (e *ScriptEditor) Highlights() [][2]int { return e.highlights }

// SetFlash lights [start, end) (rune offsets) apart from the search's
// highlights, until it is set again (an empty range: none). The caller
// takes it away after a moment.
func (e *ScriptEditor) SetFlash(start, end int) {
	if e.flash == [2]int{start, end} {
		return
	}
	e.flash = [2]int{start, end}
	e.relayout()
}

// Flash is the range SetFlash lights.
func (e *ScriptEditor) Flash() (start, end int) { return e.flash[0], e.flash[1] }

// Edit changes the text through the buffer and reports the change: use
// Buffer().Group inside for one undo step.
func (e *ScriptEditor) Edit(f func(b *buffer.Buffer)) {
	f(e.buf)
	e.changed(true)
}

// Navigate changes the cursor or selection through the buffer.
func (e *ScriptEditor) Navigate(f func(b *buffer.Buffer)) {
	f(e.buf)
	e.changed(false)
}

// Select selects [start, end) (rune offsets), cursor at end.
func (e *ScriptEditor) Select(start, end int) {
	e.Navigate(func(b *buffer.Buffer) { b.Select(start, end) })
}

// Replace replaces [start, end) with text as one undo step.
func (e *ScriptEditor) Replace(start, end int, text string) {
	e.Edit(func(b *buffer.Buffer) { b.Replace(start, end, text) })
}

// GridRow is what the editor shows in a visual row (for tests and
// inspection).
func (e *ScriptEditor) GridRow(row int) widget.TextGridRow {
	e.fresh()
	if row < 0 || row >= len(e.layout.Rows) {
		return widget.TextGridRow{}
	}
	return e.row(row)
}

// SelectedText is the selected text.
func (e *ScriptEditor) SelectedText() string { return e.buf.SelectedText() }

// CursorOffset is the cursor's rune offset.
func (e *ScriptEditor) CursorOffset() int { return e.buf.Cursor() }

// SetCursorOffset moves the cursor, dropping the selection.
func (e *ScriptEditor) SetCursorOffset(off int) {
	e.Navigate(func(b *buffer.Buffer) { b.SetCursor(off, false) })
}

// Undo undoes the last edit step.
func (e *ScriptEditor) Undo() {
	if t := e.buf.UndoTag(); t != nil && e.OnTaggedStep != nil && !e.OnTaggedStep(t, false) {
		return
	}
	if e.buf.Undo() {
		e.changed(true)
	}
}

// Redo redoes the last undone step.
func (e *ScriptEditor) Redo() {
	if t := e.buf.RedoTag(); t != nil && e.OnTaggedStep != nil && !e.OnTaggedStep(t, true) {
		return
	}
	if e.buf.Redo() {
		e.changed(true)
	}
}

// UndoSteps is how many steps there are to undo: a mark for
// JoinStepsSince.
func (e *ScriptEditor) UndoSteps() int { return e.buf.Steps() }

// JoinStepsSince makes the steps after mark n one undo step (buffer's
// JoinSince).
func (e *ScriptEditor) JoinStepsSince(n int) bool { return e.buf.JoinSince(n) }

// TagLastStep tags the last undo step: OnTaggedStep is asked before it is
// undone or redone.
func (e *ScriptEditor) TagLastStep(tag any) bool { return e.buf.Tag(tag) }

// UndoTag and RedoTag are the tags of the steps Undo and Redo would take.
func (e *ScriptEditor) UndoTag() any { return e.buf.UndoTag() }
func (e *ScriptEditor) RedoTag() any { return e.buf.RedoTag() }

// Columns is the number of columns the text is wrapped to.
func (e *ScriptEditor) Columns() int { return e.columns }

// SetOffstage takes the editor out of sight (its document's tab behind
// another: Accolade's middle tabs) or back. Out of sight, an
// edit does not wrap and colour the whole text again: that waits until
// the layout is asked for (a cursor's place, a row, a key's motion) or the
// editor is back in sight, laid out once then. Its text, undo history and
// callbacks go on as always.
func (e *ScriptEditor) SetOffstage(off bool) {
	if e.offstage == off {
		return
	}
	e.offstage = off
	if !off && e.stale {
		e.relayout()
		if e.onCursorMoved != nil {
			e.onCursorMoved(e.CursorPosition(), e.cellHeight())
		}
	}
}

// fresh lays the editor out if it changed out of sight: what reads the
// layout reads the text as it is.
func (e *ScriptEditor) fresh() {
	if e.stale {
		off := e.offstage
		e.offstage = false
		e.relayout()
		e.offstage = off
	}
}

// CreateRenderer draws the grid, where its rows are.
func (e *ScriptEditor) CreateRenderer() fyne.WidgetRenderer {
	return &editorRenderer{e: e}
}

type editorRenderer struct{ e *ScriptEditor }

func (r *editorRenderer) Layout(size fyne.Size) {
	h := r.e.cellHeight()
	r.e.grid.Move(fyne.NewPos(0, float32(r.e.gridFirst)*h))
	r.e.grid.Resize(fyne.NewSize(size.Width, float32(r.e.gridEnd-r.e.gridFirst)*h))
}
func (r *editorRenderer) MinSize() fyne.Size           { return r.e.MinSize() }
func (r *editorRenderer) Refresh()                     { r.Layout(r.e.Size()); r.e.grid.Refresh() }
func (r *editorRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.e.grid} }
func (r *editorRenderer) Destroy()                     {}

// setView tells the editor which part of it is in view (in a scroll);
// the grid is redrawn when the view leaves the rows it holds.
func (e *ScriptEditor) setView(top, height float32) {
	e.viewed, e.viewTop, e.viewHeight = true, top, max(height, 1)
	first, end := e.viewRows()
	if first < e.gridFirst || end > e.gridEnd || e.gridEnd-e.gridFirst > end-first+4*viewMargin {
		e.drawRows()
	}
}

// viewRows are the rows in view (all of them without a view).
func (e *ScriptEditor) viewRows() (first, end int) {
	e.fresh()
	n := len(e.layout.Rows)
	if !e.viewed {
		return 0, n
	}
	h := e.cellHeight()
	first = min(max(int(e.viewTop/h), 0), n)
	end = min(max(int((e.viewTop+e.viewHeight)/h)+1, first), n)
	return first, end
}

// Resize re-wraps the text to the new width.
func (e *ScriptEditor) Resize(size fyne.Size) {
	e.BaseWidget.Resize(size)
	if cols := int(size.Width / e.cellWidth()); cols > 0 && cols != e.columns {
		e.columns = cols
		e.relayout()
	}
}

// MinSize is a few columns wide and as tall as the wrapped text.
func (e *ScriptEditor) MinSize() fyne.Size {
	e.fresh()
	return fyne.NewSize(e.cellWidth()*10, e.cellHeight()*float32(len(e.layout.Rows)))
}

func (e *ScriptEditor) cellSize() fyne.Size {
	// in the font the grid draws in (the editor's theme's: MeasureText
	// measures the app theme's, which is not the editor's font)
	th, style := e.Theme(), fyne.TextStyle{Monospace: true}
	size, _ := fyne.CurrentApp().Driver().RenderedTextSize("M", th.Size(theme.SizeNameText), style, th.Font(style))
	// the width as the font has it: a run of text is that wide a character
	return fyne.NewSize(size.Width, float32(math.Round(float64(size.Height))))
}

func (e *ScriptEditor) cellWidth() float32  { return max(e.cellSize().Width, 1) }
func (e *ScriptEditor) cellHeight() float32 { return max(e.cellSize().Height, 1) }

// relayout re-wraps the text and redraws the grid.
func (e *ScriptEditor) relayout() {
	if e.offstage {
		e.stale = true // laid out when asked for, or back in sight
		return
	}
	e.stale = false
	g := e.gutter()
	line := -1 // the kinds depend on the cursor's line only while typing a screenplay
	if e.Format != nil && !e.Markdown {
		line, _ = e.buf.Position(e.buf.Cursor())
	}
	key := layoutKey{rev: e.buf.Rev(), columns: e.columns, gutter: g, page: e.PageColumns, line: line,
		markdown: e.Markdown, syntax: e.Syntax, format: e.Format != nil, rtl: e.LanguageRTL, raw: -1}
	e.chips = nil
	if e.Chips != nil && !e.Markdown {
		e.chips = e.Chips(e.buf.Text())
		if len(e.chips) > 0 {
			key.chips, key.nchips = &e.chips[0], len(e.chips)
			key.raw = chipAround(e.chips, e.buf.Cursor())
		}
	}
	if !e.laid.ok || e.laid.key != key {
		e.laid = e.layOut(key)
	}
	text, kinds, inline, lineStart := e.laid.text, e.laid.kinds, e.laid.inline, e.laid.lineStart
	e.rtl, e.layout = e.laid.rtl, e.laid.layout
	e.foldAt = e.fold()
	e.headings = nil
	if e.Headings != nil {
		e.headings = e.Headings()
	}
	e.inks = nil
	if e.Inks != nil {
		e.inks = e.Inks()
	}
	e.tags = nil
	if e.Tags != nil && !e.Markdown {
		e.tags = e.Tags()
	}
	e.marks = nil
	if e.Marks != nil {
		for line, c := range e.Marks() {
			if e.marks == nil {
				e.marks = map[int]*widget.CustomTextGridStyle{}
			}
			e.marks[line] = &widget.CustomTextGridStyle{FGColor: c}
		}
	}

	th := e.Theme()
	v := theme.VariantLight
	if app := fyne.CurrentApp(); app != nil {
		v = app.Settings().ThemeVariant()
	}
	selected := &widget.CustomTextGridStyle{BGColor: th.Color(theme.ColorNameSelection, v)}
	cursor := &widget.CustomTextGridStyle{
		FGColor: th.Color(theme.ColorNameBackground, v),
		BGColor: th.Color(theme.ColorNameForeground, v),
	}
	highlight := &widget.CustomTextGridStyle{BGColor: th.Color(theme.ColorNameHover, v)}
	if c := th.Color(theme.ColorNameWarning, v); c != nil {
		r, g, b, _ := c.RGBA()
		highlight.BGColor = color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 0x60}
	}
	selStart, selEnd := e.buf.Selection()
	curRow, curCol := e.layout.RowCol(e.buf.Cursor())

	number := &widget.CustomTextGridStyle{FGColor: th.Color(theme.ColorNamePlaceHolder, v)}
	others := e.buf.Spans()[1:]
	var otherCursors map[int][]int
	for _, o := range others {
		if otherCursors == nil {
			otherCursors = map[int][]int{}
		}
		r, c := e.layout.RowCol(o.Cursor)
		otherCursors[r] = append(otherCursors[r], c)
	}
	fr, fg, fb, _ := th.Color(theme.ColorNamePrimary, v).RGBA()
	flash := color.NRGBA{R: uint8(fr >> 8), G: uint8(fg >> 8), B: uint8(fb >> 8), A: 0x50}
	chip := &widget.CustomTextGridStyle{FGColor: th.Color(theme.ColorNamePrimary, v),
		BGColor: color.NRGBA{R: uint8(fr >> 8), G: uint8(fg >> 8), B: uint8(fb >> 8), A: 0x28}}
	wr, wg, wb, _ := th.Color(theme.ColorNameWarning, v).RGBA()
	chipWarn := &widget.CustomTextGridStyle{FGColor: th.Color(theme.ColorNameWarning, v),
		BGColor: color.NRGBA{R: uint8(wr >> 8), G: uint8(wg >> 8), B: uint8(wb >> 8), A: 0x28}}
	e.draw = drawing{chip: chip, chipWarn: chipWarn, flash: flash, text: text, kinds: kinds, inline: inline, lineStart: lineStart, th: th, v: v, selected: selected, cursor: cursor,
		highlight: highlight, number: number, selStart: selStart, selEnd: selEnd,
		curRow: curRow, curCol: curCol, gutter: g, others: others, otherCursors: otherCursors}
	e.drawRows()
}

// layoutKey is what a layout was made for: when a cursor move leaves it
// the same, the text is not wrapped and classified again (that took
// tens of milliseconds a move in a long script).
type layoutKey struct {
	rev                      uint64
	columns, gutter, page    int
	line                     int
	markdown, syntax, format bool
	rtl                      bool // LanguageRTL
	// the chips (Chips: their slice) and the one drawn as text (-1: none)
	chips       *Chip
	nchips, raw int
}

// laidOut is the text laid out for a key.
type laidOut struct {
	key       layoutKey
	ok        bool
	text      []rune
	rtl       []bool
	layout    wrap.Layout
	kinds     []syntax.Kind
	inline    map[int][]syntax.Span
	lineStart []int
	chipAt    map[int]int // the chips drawn as labels, by their start: which of key's
}

// TypingKinds are the lines' kinds as the editor has them laid out now
// (syntax.ClassifyTyping at the cursor's line), when it has: ok is false
// for Markdown, without Format, or before the editor laid out this text.
func (e *ScriptEditor) TypingKinds() (kinds []syntax.Kind, ok bool) {
	k := e.laid.key
	if !e.laid.ok || !k.format || k.markdown || k.rev != e.buf.Rev() {
		return nil, false
	}
	if line, _ := e.buf.Position(e.buf.Cursor()); line != k.line {
		return nil, false
	}
	return e.laid.kinds, true
}

// MarkdownKinds are the lines' kinds as the editor laid out its
// Markdown just now (syntax.ClassifyMarkdown): ok is false for a
// screenplay, without Syntax, or before it laid out this text.
func (e *ScriptEditor) MarkdownKinds() (kinds []syntax.Kind, ok bool) {
	k := e.laid.key
	if !e.laid.ok || !k.markdown || !k.syntax || k.rev != e.buf.Rev() {
		return nil, false
	}
	return e.laid.kinds, true
}

// layOut wraps and classifies the text.
func (e *ScriptEditor) layOut(key layoutKey) laidOut {
	l := laidOut{key: key, ok: true, text: []rune(e.buf.Text())}
	text, g := l.text, key.gutter
	lines := strings.Split(e.buf.Text(), "\n")
	l.rtl = make([]bool, len(lines))
	for i, ln := range lines {
		l.rtl[i] = lex.LineRTL(ln, key.rtl)
	}
	spans := l.chipSpans(e.chips, key.raw, lines)
	var kinds []syntax.Kind
	if e.Format != nil && !e.Markdown {
		kinds = e.classifyTyping(lines, key.line)
		formats := make([]wrap.Format, len(kinds))
		page := e.PageColumns
		if page <= 0 {
			page = max(e.columns-g, 1)
		}
		for i, k := range kinds {
			formats[i] = e.Format(k)
			if l.rtl[i] {
				formats[i] = mirrored(formats[i], page)
			}
		}
		l.layout = wrap.WrapFormattedSpans(text, max(e.columns-g, 1), formats, spans)
	} else {
		l.layout = wrap.WrapSpans(text, max(e.columns-g, 1), spans)
	}
	switch {
	case e.Markdown && e.Syntax:
		kinds = syntax.ClassifyMarkdown(lines)
		l.inline, l.lineStart = map[int][]syntax.Span{}, make([]int, len(lines))
		off := 0
		for i, ln := range lines {
			l.lineStart[i] = off
			off += len([]rune(ln)) + 1
			switch kinds[i] {
			case syntax.Action, syntax.MdHeading, syntax.MdQuote, syntax.MdListItem:
				if s := syntax.MarkdownInline(ln); s != nil {
					l.inline[i] = s
				}
			}
		}
	case kinds == nil && e.Syntax:
		kinds = e.classifyTyping(lines, -1)
	}
	if l.lineStart == nil { // where each line starts, for Inks
		l.lineStart = make([]int, len(lines))
		off := 0
		for i, ln := range lines {
			l.lineStart[i] = off
			off += utf8.RuneCountInString(ln) + 1
		}
	}
	l.kinds = kinds
	return l
}

// chipSpans are the chips drawn as labels (all but raw, and none on a
// line read right to left), as spans to wrap in one piece; chipAt is
// filled in.
func (l *laidOut) chipSpans(chips []Chip, raw int, lines []string) []wrap.Span {
	if len(chips) == 0 {
		return nil
	}
	var spans []wrap.Span
	line, off := 0, 0 // the line chip i is on, and its start
	for i, c := range chips {
		if i == raw || c.End <= c.Start {
			continue
		}
		for line < len(lines)-1 && off+utf8.RuneCountInString(lines[line]) < c.Start {
			off += utf8.RuneCountInString(lines[line]) + 1
			line++
		}
		if line < len(l.rtl) && l.rtl[line] {
			continue
		}
		cols := 0
		for _, r := range c.Label {
			cols += wrap.Width(r)
		}
		if l.chipAt == nil {
			l.chipAt = map[int]int{}
		}
		l.chipAt[c.Start] = i
		spans = append(spans, wrap.Span{Start: c.Start, End: c.End, Cols: cols})
	}
	return spans
}

// chipAround is the chip the offset is inside (-1: none).
func chipAround(chips []Chip, off int) int {
	i := sort.Search(len(chips), func(i int) bool { return chips[i].End > off })
	if i < len(chips) && chips[i].Start < off && off < chips[i].End {
		return i
	}
	return -1
}

// ChipAt is the chip drawn as its label at offset off, if there is one.
func (e *ScriptEditor) ChipAt(off int) (Chip, bool) {
	e.fresh()
	if s, ok := e.layout.SpanAt(off); ok {
		if i, ok := e.laid.chipAt[s.Start]; ok && i < len(e.chips) {
			return e.chips[i], true
		}
	}
	return Chip{}, false
}

// chipUnder is the chip drawn under a position in the editor, and its
// first cell's place.
func (e *ScriptEditor) chipUnder(p fyne.Position) (Chip, fyne.Position, bool) {
	off, at := e.charAt(p)
	if off < 0 {
		return Chip{}, at, false
	}
	c, ok := e.ChipAt(off)
	return c, at, ok
}

// drawRows puts the rows in view (and viewMargin around them) in the grid.
func (e *ScriptEditor) drawRows() {
	first, end := e.viewRows()
	if e.viewed {
		first, end = max(first-viewMargin, 0), min(end+viewMargin, len(e.layout.Rows))
	}
	rows := make([]widget.TextGridRow, end-first)
	mirror := make([][2]int, end-first)
	for i := first; i < end; i++ {
		rows[i-first] = e.row(i)
		mirror[i-first] = e.mirrored(i)
	}
	e.gridFirst, e.gridEnd = first, end
	e.grid.Rows, e.grid.Mirror = rows, mirror
	e.BaseWidget.Refresh()
}

// row is the grid row of the wrapped row i.
func (e *ScriptEditor) row(i int) widget.TextGridRow {
	d := &e.draw
	text, kinds, th, v, g := d.text, d.kinds, d.th, d.v, d.gutter
	selStart, selEnd, curRow, curCol := d.selStart, d.selEnd, d.curRow, d.curCol
	selected, cursor, highlight, number := d.selected, d.cursor, d.highlight, d.number
	highlighted := func(off int) bool {
		for _, h := range e.highlights {
			if off >= h[0] && off < h[1] {
				return true
			}
		}
		return false
	}
	r := e.layout.Rows[i]
	cells := make([]widget.TextGridCell, 0, g+r.Indent+r.End-r.Start+1)
	first := i == 0 || e.layout.Rows[i-1].Line != r.Line // the line's first row
	if n := e.numbersWidth(); n > 0 {
		label := ""
		if first {
			label = strconv.Itoa(r.Line + 1)
		}
		label = fmt.Sprintf("%*s ", n-1, label)
		for _, ch := range label {
			cells = append(cells, widget.TextGridCell{Rune: ch, Style: number})
		}
	}
	if e.foldWidth() > 0 {
		mark := "  "
		switch {
		case first && e.foldAt[r.Line]:
			mark = "▸ "
		case first && e.headings[r.Line]:
			mark = "▾ "
		}
		for _, ch := range mark {
			cells = append(cells, widget.TextGridCell{Rune: ch, Style: number})
		}
		if m := e.marks[r.Line]; m != nil { // the mark's bar, on each of the line's rows
			cells[len(cells)-1] = widget.TextGridCell{Rune: markBar, Style: m}
		}
	}
	for c := 0; c < r.Indent; c++ {
		cells = append(cells, widget.TextGridCell{Rune: ' '})
	}
	var lineStyle widget.TextGridStyle
	if r.Line < len(kinds) {
		lineStyle = kindStyle(kinds[r.Line], th, v)
	}
	if first && e.blockAt[r.Line] { // a folded block: its summary, quiet
		return widget.TextGridRow{Cells: e.labelCells(cells, r, number)}
	}
	spans := d.inline[r.Line]
	inks := e.inks[r.Line]
	for off := r.Start; off < r.End; off++ {
		if i, ok := e.laid.chipAt[off]; ok && i < len(e.chips) { // a chip: its label
			c := e.chips[i]
			var style widget.TextGridStyle = d.chip
			if c.Warn {
				style = d.chipWarn
			}
			if off >= selStart && off < selEnd {
				style = withBackground(style, selected.BGColor)
			}
			for _, ch := range c.Label {
				cells = append(cells, widget.TextGridCell{Rune: ch, Style: style})
				if wrap.Width(ch) == 2 {
					cells = append(cells, widget.TextGridCell{Rune: wideFill, Style: style})
				}
			}
			off = min(c.End, r.End) - 1
			continue
		}
		cell := widget.TextGridCell{Rune: text[off], Style: lineStyle}
		for _, s := range spans {
			if at := off - d.lineStart[r.Line]; at >= s.Start && at < s.End {
				cell.Style = inlineStyle(lineStyle, s.Kind, th, v)
			}
		}
		for _, k := range inks {
			if at := off - d.lineStart[r.Line]; at >= k.Start && at < k.End {
				cell.Style = inked(cell.Style, k.Color)
			}
		}
		switch {
		case off >= selStart && off < selEnd, d.inOther(off):
			cell.Style = withBackground(cell.Style, selected.BGColor)
		case highlighted(off):
			cell.Style = withBackground(cell.Style, highlight.BGColor)
		case off >= e.flash[0] && off < e.flash[1]:
			cell.Style = withBackground(cell.Style, d.flash)
		}
		cells = append(cells, cell)
		if wrap.Width(cell.Rune) == 2 { // a wide character's second column
			cells = append(cells, widget.TextGridCell{Rune: wideFill, Style: cell.Style})
		}
	}
	if r.Last && e.foldAt[r.Line] { // a folded heading: what it holds is hidden
		for _, ch := range " …" {
			cells = append(cells, widget.TextGridCell{Rune: ch, Style: number})
		}
	}
	if _, label, ok := e.tagOf(i); ok { // a tag after the line: faint, not text
		cells = append(cells, widget.TextGridCell{Rune: ' '})
		for _, ch := range label {
			cells = append(cells, widget.TextGridCell{Rune: ch, Style: number})
			if wrap.Width(ch) == 2 {
				cells = append(cells, widget.TextGridCell{Rune: wideFill, Style: number})
			}
		}
	}
	if e.focused && i == curRow {
		setCursor(&cells, g+curCol, cursor)
	}
	for _, c := range d.otherCursors[i] {
		if e.focused {
			setCursor(&cells, g+c, cursor)
		}
	}
	return widget.TextGridRow{Cells: cells}
}

// tagOf is the tag drawn after row i, if one is, and the label drawn:
// on a line's last row, read left to right, not a folded block, with
// room for it (its Short if only that has room).
func (e *ScriptEditor) tagOf(i int) (Tag, string, bool) {
	if i < 0 || i >= len(e.layout.Rows) || len(e.tags) == 0 {
		return Tag{}, "", false
	}
	r := e.layout.Rows[i]
	t, ok := e.tags[r.Line]
	if !ok || !r.Last || e.blockAt[r.Line] || e.rowRTL(i) {
		return Tag{}, "", false
	}
	room := e.columns - 1 - e.tagStart(i) - 1 // the last column clear (a scroll bar over it)
	for _, l := range []string{t.Label, t.Short} {
		if l != "" && labelWidth(l) <= room {
			return t, l, true
		}
	}
	return Tag{}, "", false
}

// tagStart is the column (the gutter's counted) after row i's text,
// where its tag's space goes.
func (e *ScriptEditor) tagStart(i int) int {
	r := e.layout.Rows[i]
	c := e.gutter() + r.Indent + e.layout.RowWidth(i)
	if e.foldAt[r.Line] {
		c += 2 // " …"
	}
	return c
}

func labelWidth(s string) int {
	n := 0
	for _, r := range s {
		n += wrap.Width(r)
	}
	return n
}

// tagAt is the line whose tag is drawn at p, its tag, and its first
// cell's place.
func (e *ScriptEditor) tagAt(p fyne.Position) (int, Tag, fyne.Position, bool) {
	e.fresh()
	if e.OnTagTapped == nil || p.Y < 0 || len(e.layout.Rows) == 0 {
		return 0, Tag{}, fyne.Position{}, false
	}
	row, col := int(p.Y/e.cellHeight()), int(p.X/e.cellWidth())
	t, label, ok := e.tagOf(row)
	if !ok {
		return 0, Tag{}, fyne.Position{}, false
	}
	start := e.tagStart(row) + 1
	if col < start || col >= start+labelWidth(label) {
		return 0, Tag{}, fyne.Position{}, false
	}
	return e.layout.Rows[row].Line, t, fyne.NewPos(float32(start)*e.cellWidth(), float32(row)*e.cellHeight()), true
}

// TagRect is where line's tag is drawn in the editor (its first cell's
// place), if it is: for a menu opened on it from the keyboard.
func (e *ScriptEditor) TagRect(line int) (fyne.Position, fyne.Size, bool) {
	e.fresh()
	for i, r := range e.layout.Rows {
		if r.Line != line || !r.Last {
			continue
		}
		if _, _, ok := e.tagOf(i); !ok {
			return fyne.Position{}, fyne.Size{}, false
		}
		return fyne.NewPos(float32(e.tagStart(i)+1)*e.cellWidth(), float32(i)*e.cellHeight()), fyne.NewSize(e.cellWidth(), e.cellHeight()), true
	}
	return fyne.Position{}, fyne.Size{}, false
}

// labelCells are a folded block's row: its label (FoldLabel) after the
// gutter's cells, cut to the row's width.
func (e *ScriptEditor) labelCells(cells []widget.TextGridCell, r wrap.Row, style widget.TextGridStyle) []widget.TextGridCell {
	label := "…"
	if e.FoldLabel != nil {
		label = e.FoldLabel(r.Line)
	}
	room := max(e.columns-e.draw.gutter-r.Indent-1, 1) // the last column clear (a scroll bar over it)
	if k := e.draw.kinds; e.Format != nil && !e.Markdown && r.Line < len(k) {
		if f := e.Format(k[r.Line]); f.Width > 0 { // the line's element's width (a phone's narrower page)
			room = max(min(room, f.Width), 1)
		}
	}
	runes := []rune(label)
	if w := wrap.Width; len(runes) > 0 {
		used, n := 0, 0
		for n < len(runes) && used+w(runes[n]) <= room {
			used += w(runes[n])
			n++
		}
		if n < len(runes) {
			n = max(n-1, 0)
			runes = append(runes[:n:n], '…')
		}
	}
	quiet := &widget.CustomTextGridStyle{TextStyle: fyne.TextStyle{Italic: true}, FGColor: style.TextColor()}
	for c := 0; c < r.Indent; c++ {
		cells = append(cells, widget.TextGridCell{Rune: ' '})
	}
	for _, ch := range runes {
		cells = append(cells, widget.TextGridCell{Rune: ch, Style: quiet})
		if wrap.Width(ch) == 2 {
			cells = append(cells, widget.TextGridCell{Rune: wideFill, Style: quiet})
		}
	}
	return cells
}

// markBar is a marked line's bar (Marks).
const markBar = '┃'

// wideFill is the cell after a wide character, which it is drawn over.
const wideFill = rune(0)

// setCursor shows the cursor at column col (both columns of a wide
// character), past the row's end in a space.
func setCursor(cells *[]widget.TextGridCell, col int, cursor widget.TextGridStyle) {
	i := cellAt(*cells, col)
	for len(*cells) <= i {
		*cells = append(*cells, widget.TextGridCell{Rune: ' '})
	}
	c := *cells
	c[i].Style = cursor
	if i+1 < len(c) && c[i+1].Rune == wideFill {
		c[i+1].Style = cursor
	}
}

// cellAt is the cell shown at column col: marks drawn on the character
// before them (wrap.Width 0) take no column of their own.
func cellAt(cells []widget.TextGridCell, col int) int {
	i, c := 0, 0
	for ; i < len(cells); i++ {
		w := cellWidth(cells[i].Rune)
		if c+w > col && w > 0 {
			return i
		}
		c += w
	}
	return i + (col - c)
}

// cellWidth is how many columns a cell takes (the wide character's second
// column is its own cell).
func cellWidth(r rune) int {
	if r == wideFill {
		return 1
	}
	return min(wrap.Width(r), 1)
}

// fold leaves the folded lines out of the layout (one the cursor is in
// opens) and returns the folded headings' lines.
func (e *ScriptEditor) fold() map[int]bool {
	if e.Folds == nil {
		return nil
	}
	e.blockAt = nil
	ranges := e.Folds()
	if len(ranges) == 0 {
		return nil
	}
	line, _ := e.buf.Position(e.buf.Cursor())
	var shut [][2]int // the cursor's fold opens
	headings := map[int]bool{}
	for _, r := range ranges {
		from := r.From
		if r.Block {
			from-- // the block's first line is in it
		}
		if line >= from && line < r.To {
			if e.OnUnfold != nil {
				e.OnUnfold(r.From - 1)
			}
			continue
		}
		shut = append(shut, [2]int{r.From, r.To})
		headings[r.From-1] = true
		if r.Block {
			if e.blockAt == nil {
				e.blockAt = map[int]bool{}
			}
			e.blockAt[r.From-1] = true
		}
	}
	if len(shut) == 0 {
		return headings
	}
	hidden := func(l int) bool {
		for _, r := range shut {
			if l >= r[0] && l < r[1] {
				return true
			}
		}
		return false
	}
	// the rows left, in room kept from the last time (a cursor move lays
	// the rows out again: no new room each time)
	all := e.layout.Rows
	rows := e.foldRoom[:0]
	for i, r := range all {
		more := i > 0 && all[i-1].Line == r.Line // a block's line: its first row only
		if !hidden(r.Line) && !(more && e.blockAt[r.Line]) {
			rows = append(rows, r)
		}
	}
	e.foldRoom = rows
	e.layout.Rows = rows
	return headings
}

// inOther reports whether off is in one of the other selections.
func (d drawing) inOther(off int) bool {
	for _, o := range d.others {
		if off >= o.Start() && off < o.End() {
			return true
		}
	}
	return false
}

// Refresh redraws the editor with the current theme's colours: Fyne
// refreshes every widget when the theme changes, and the syntax colours
// are baked into the grid's cell styles.
// RedrawMarks draws the rows again with the Marks as they are now (the
// text's layout kept).
func (e *ScriptEditor) RedrawMarks() { e.relayout() }

func (e *ScriptEditor) Refresh() {
	e.laid.ok = false // its settings may have changed (Format, SceneStarts)
	e.relayout()
}

// kindStyle is how a Fountain element is shown (nil for plain action and
// dialogue). Scene headings, the landmarks of a script, are the only
// element in the accent colour; character names are bold; parentheticals,
// transitions, the title page and notes are dimmed.
func kindStyle(k syntax.Kind, th fyne.Theme, v fyne.ThemeVariant) widget.TextGridStyle {
	fg := th.Color(theme.ColorNameForeground, v)
	dim := mix(fg, th.Color(theme.ColorNameBackground, v), 0.45)
	switch k {
	case syntax.SceneHeading:
		return &widget.CustomTextGridStyle{TextStyle: fyne.TextStyle{Bold: true}, FGColor: th.Color(theme.ColorNamePrimary, v)}
	case syntax.Character:
		return &widget.CustomTextGridStyle{TextStyle: fyne.TextStyle{Bold: true}, FGColor: fg}
	case syntax.Parenthetical:
		return &widget.CustomTextGridStyle{TextStyle: fyne.TextStyle{Italic: true}, FGColor: dim}
	case syntax.Lyrics:
		return &widget.CustomTextGridStyle{TextStyle: fyne.TextStyle{Italic: true}, FGColor: fg}
	case syntax.Transition, syntax.TitlePage, syntax.PageBreak:
		return &widget.CustomTextGridStyle{FGColor: dim}
	case syntax.Note, syntax.Section, syntax.Synopsis:
		return &widget.CustomTextGridStyle{TextStyle: fyne.TextStyle{Italic: true}, FGColor: dim}
	// Markdown: headings are its landmarks, as scene headings are a script's
	case syntax.MdHeading:
		return &widget.CustomTextGridStyle{TextStyle: fyne.TextStyle{Bold: true}, FGColor: th.Color(theme.ColorNamePrimary, v)}
	case syntax.MdQuote:
		return &widget.CustomTextGridStyle{TextStyle: fyne.TextStyle{Italic: true}, FGColor: fg}
	case syntax.MdCode, syntax.MdCodeFence, syntax.MdRule, syntax.MdFrontMatter:
		return &widget.CustomTextGridStyle{FGColor: dim}
	case syntax.MdComment:
		return &widget.CustomTextGridStyle{TextStyle: fyne.TextStyle{Italic: true}, FGColor: dim}
	}
	return nil
}

// inlineStyle is a line's style with Markdown's emphasis on it: bold,
// italic, code dimmed, links in the accent colour.
func inlineStyle(line widget.TextGridStyle, k syntax.InlineKind, th fyne.Theme, v fyne.ThemeVariant) widget.TextGridStyle {
	out := &widget.CustomTextGridStyle{FGColor: th.Color(theme.ColorNameForeground, v)}
	if line != nil {
		out.TextStyle, out.FGColor = line.Style(), line.TextColor()
	}
	switch k {
	case syntax.Strong:
		out.TextStyle.Bold = true
	case syntax.Emphasis:
		out.TextStyle.Italic = true
	case syntax.CodeSpan:
		out.FGColor = mix(th.Color(theme.ColorNameForeground, v), th.Color(theme.ColorNameBackground, v), 0.45)
	case syntax.Link:
		out.FGColor = th.Color(theme.ColorNamePrimary, v)
	}
	return out
}

// mix blends a towards b by the fraction t.
func mix(a, b color.Color, t float64) color.Color {
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	f := func(x, y uint32) uint8 { return uint8((float64(x)*(1-t) + float64(y)*t) / 257) }
	return color.NRGBA{R: f(ar, br), G: f(ag, bg), B: f(ab, bb), A: 0xff}
}

// inked is a style in an ink's colour, underlined (the underline is
// drawn in the text's colour).
func inked(s widget.TextGridStyle, c color.Color) widget.TextGridStyle {
	out := &widget.CustomTextGridStyle{FGColor: c}
	if s != nil {
		out.TextStyle, out.BGColor = s.Style(), s.BackgroundColor()
	}
	out.TextStyle.Underline = true
	return out
}

// withBackground is a line style with a background colour added.
func withBackground(s widget.TextGridStyle, bg color.Color) widget.TextGridStyle {
	out := &widget.CustomTextGridStyle{BGColor: bg}
	if s != nil {
		out.TextStyle, out.FGColor = s.Style(), s.TextColor()
	}
	return out
}

// changed re-lays out after an edit or cursor move.
func (e *ScriptEditor) changed(edited bool) {
	e.relayout()
	if e.onCursorMoved != nil && !e.offstage {
		e.onCursorMoved(e.CursorPosition(), e.cellHeight())
	}
	if e.OnCursorChanged != nil {
		e.OnCursorChanged()
	}
	if edited && e.OnChanged != nil {
		e.OnChanged(e.buf.Text())
	}
}

// Focus handling: the window's focus can change inside a browser event,
// so the cursor is redrawn after it, not in it.

func (e *ScriptEditor) FocusGained() { e.focused = true; fyne.Do(e.relayout) }
func (e *ScriptEditor) FocusLost() {
	e.focused = false
	e.shift = false
	if e.OnFocusLost != nil {
		e.OnFocusLost()
	}
	fyne.Do(e.relayout)
}

// Tapped focuses the editor and places the cursor at the click (Shift
// extends the selection).
func (e *ScriptEditor) Tapped(ev *fyne.PointEvent) {
	if line, ok := e.foldMarkAt(ev.Position); ok {
		e.OnFoldTapped(line)
		return
	}
	if line, t, at, ok := e.tagAt(ev.Position); ok {
		e.OnTagTapped(line, t, at, fyne.NewSize(e.cellWidth(), e.cellHeight()))
		return
	}
	e.focus()
	off := e.offsetAt(ev.Position)
	if c, _, ok := e.chipUnder(ev.Position); ok && !e.shift { // a chip tapped: into it, its text shown
		off = min(c.Start+2, c.End-1)
	}
	e.buf.SetCursor(off, e.shift)
	e.goalCol = -1
	e.changed(false)
}

// TappedSecondary puts the cursor at a right click (unless it is in the
// selection) and tells OnSecondaryTapped.
func (e *ScriptEditor) TappedSecondary(ev *fyne.PointEvent) {
	if line, t, at, ok := e.tagAt(ev.Position); ok {
		e.OnTagTapped(line, t, at, fyne.NewSize(e.cellWidth(), e.cellHeight()))
		return
	}
	if c, at, ok := e.chipUnder(ev.Position); ok && e.OnChipPressed != nil {
		e.OnChipPressed(c, at, fyne.NewSize(e.cellWidth(), e.cellHeight()))
		return
	}
	if e.OnSecondaryTapped == nil {
		return
	}
	e.focus()
	off := e.offsetAt(ev.Position)
	if start, end := e.buf.Selection(); start == end || off < start || off > end {
		e.buf.SetCursor(off, false)
		e.goalCol = -1
		e.changed(false)
	}
	e.OnSecondaryTapped(ev.AbsolutePosition)
}

// foldMarkAt is the heading whose fold mark is at p, if p is on one.
func (e *ScriptEditor) foldMarkAt(p fyne.Position) (int, bool) {
	e.fresh()
	if e.OnFoldTapped == nil || e.foldWidth() == 0 || len(e.layout.Rows) == 0 {
		return 0, false
	}
	row, col := int(p.Y/e.cellHeight()), int(p.X/e.cellWidth())
	if row < 0 || row >= len(e.layout.Rows) || col < e.numbersWidth() || col >= e.gutter() {
		return 0, false
	}
	line := e.layout.Rows[row].Line
	if !e.headings[line] && !e.foldAt[line] {
		return 0, false
	}
	return line, true
}

// DoubleTapped selects the word under the pointer.
func (e *ScriptEditor) DoubleTapped(ev *fyne.PointEvent) {
	e.focus()
	start, end := e.wordAt(e.offsetAt(ev.Position))
	e.buf.Select(start, end)
	e.changed(false)
}

// Dragged selects from where the drag started to the pointer.
func (e *ScriptEditor) Dragged(ev *fyne.DragEvent) {
	if !e.dragging {
		e.focus()
		e.dragging = true
		e.dragFrom = e.offsetAt(ev.Position.Subtract(ev.Dragged))
	}
	e.buf.Select(e.dragFrom, e.offsetAt(ev.Position))
	e.changed(false)
}

// DragEnd ends a drag selection.
func (e *ScriptEditor) DragEnd() { e.dragging = false }

// Cursor shows the text cursor over the editor.
func (e *ScriptEditor) Cursor() desktop.Cursor { return desktop.TextCursor }

// MouseIn, MouseMoved and MouseOut tell OnHover what is under the pointer.
func (e *ScriptEditor) MouseIn(ev *desktop.MouseEvent) { e.MouseMoved(ev) }
func (e *ScriptEditor) MouseMoved(ev *desktop.MouseEvent) {
	if e.OnHover == nil {
		return
	}
	off, at := e.charAt(ev.Position)
	e.OnHover(off, at, fyne.NewSize(e.cellWidth(), e.cellHeight()))
}
func (e *ScriptEditor) MouseOut() {
	if e.OnHover != nil {
		e.OnHover(-1, fyne.Position{}, fyne.Size{})
	}
}

// charAt is the character whose cell is at a position in the editor, and
// the cell's place (-1 if there is none: the gutter, past the text, a
// line laid out right to left).
func (e *ScriptEditor) charAt(p fyne.Position) (int, fyne.Position) {
	e.fresh()
	row, col := int(p.Y/e.cellHeight()), int(p.X/e.cellWidth())-e.gutter()
	if p.Y < 0 || row >= len(e.layout.Rows) || col < 0 || e.rowRTL(row) {
		return -1, fyne.Position{}
	}
	r := e.layout.Rows[row]
	if col < r.Indent || col >= r.Indent+e.layout.RowWidth(row) {
		return -1, fyne.Position{}
	}
	return e.layout.Offset(row, col), fyne.NewPos(float32(e.gutter()+col)*e.cellWidth(), float32(row)*e.cellHeight())
}

func (e *ScriptEditor) focus() {
	if app := fyne.CurrentApp(); app != nil {
		if c := app.Driver().CanvasForObject(e); c != nil {
			c.Focus(e)
		}
	}
}

// offsetAt is the text offset nearest to a position in the editor (clicks
// land between characters).
func (e *ScriptEditor) offsetAt(p fyne.Position) int {
	e.fresh()
	if len(e.layout.Rows) == 0 {
		return 0
	}
	row := int(p.Y / e.cellHeight())
	col := int(math.Round(float64(p.X/e.cellWidth()))) - e.gutter()
	if row >= len(e.layout.Rows) {
		return e.buf.Len()
	}
	if e.rowRTL(row) { // between two cells, counted from the right
		r := e.layout.Rows[max(row, 0)]
		w := e.layout.RowWidth(max(row, 0))
		col = r.Indent + w - min(max(col-r.Indent, 0), w)
	}
	return e.layout.Offset(max(row, 0), max(col, 0))
}

// wordAt is the word (letters and digits) around an offset, or the single
// character there if it is not part of a word.
func (e *ScriptEditor) wordAt(off int) (start, end int) {
	text := []rune(e.buf.Text())
	isWord := func(i int) bool {
		return i >= 0 && i < len(text) && (unicode.IsLetter(text[i]) || unicode.IsDigit(text[i]))
	}
	if !isWord(off) && isWord(off-1) {
		off--
	}
	if !isWord(off) {
		return off, min(off+1, len(text))
	}
	start, end = off, off
	for isWord(start - 1) {
		start--
	}
	for isWord(end) {
		end++
	}
	return start, end
}

// CursorPosition is where the cursor cell is drawn, relative to the editor.
func (e *ScriptEditor) CursorPosition() fyne.Position {
	e.fresh()
	row, col := e.layout.RowCol(e.buf.Cursor())
	return fyne.NewPos(float32(e.gutter()+e.visualCol(row, col))*e.cellWidth(), float32(row)*e.cellHeight())
}

// rowRTL reports whether a visual row reads right to left: its cells are
// drawn mirrored, the start of its text at the right.
func (e *ScriptEditor) rowRTL(row int) bool {
	if row < 0 || row >= len(e.layout.Rows) {
		return false
	}
	l := e.layout.Rows[row].Line
	return l < len(e.rtl) && e.rtl[l]
}

// mirrored is the cells of a right-to-left row's text, to draw right to
// left (none for other rows).
func (e *ScriptEditor) mirrored(row int) [2]int {
	if !e.rowRTL(row) {
		return [2]int{}
	}
	r := e.layout.Rows[row]
	from := e.gutter() + r.Indent
	n := r.End - r.Start
	if text := e.draw.text; r.End <= len(text) {
		for _, ch := range text[r.Start:r.End] {
			if wrap.Width(ch) == 2 {
				n++ // its second cell
			}
		}
	}
	return [2]int{from, from + n}
}

// visualCol is where column col of a row is drawn: mirrored in a
// right-to-left row (its end left of its text).
func (e *ScriptEditor) visualCol(row, col int) int {
	if !e.rowRTL(row) {
		return col
	}
	r := e.layout.Rows[row]
	w := e.layout.RowWidth(row)
	if t := col - r.Indent; t >= 0 && t < w {
		return r.Indent + w - 1 - t
	} else if t >= w {
		return max(r.Indent-1-(t-w), 0)
	}
	return col
}

// ShiftHeld reports whether Shift is held (a vim mode's Tab and Shift+Tab).
func (e *ScriptEditor) ShiftHeld() bool { return e.shift }

// KeyDown and KeyUp track Shift for selecting with the arrow keys.
func (e *ScriptEditor) KeyDown(k *fyne.KeyEvent) {
	if k.Name == desktop.KeyShiftLeft || k.Name == desktop.KeyShiftRight {
		e.shift = true
	}
}

func (e *ScriptEditor) KeyUp(k *fyne.KeyEvent) {
	if k.Name == desktop.KeyShiftLeft || k.Name == desktop.KeyShiftRight {
		e.shift = false
	}
	if e.OnKeyUp != nil {
		e.OnKeyUp(k)
	}
}

// TypedRune inserts a typed character.
func (e *ScriptEditor) TypedRune(r rune) {
	if e.OnRune != nil && e.OnRune(r) {
		e.goalCol = -1
		return
	}
	e.insert(string(r))
	e.goalCol = -1
	e.changed(true)
}

// insert types text at the selection, or at every one there is.
func (e *ScriptEditor) insert(text string) {
	if !e.buf.HasOthers() {
		e.buf.Insert(text)
		return
	}
	e.buf.EditSpans(func(s buffer.Span) (int, int, string) { return s.Start(), s.End(), text })
}

// deleteAt deletes the selection, else the character before (or after)
// the cursor: at every selection there is.
func (e *ScriptEditor) deleteAt(before bool) {
	switch {
	case !e.buf.HasOthers() && before:
		e.buf.DeleteBackward()
	case !e.buf.HasOthers():
		e.buf.DeleteForward()
	default:
		e.buf.EditSpans(func(s buffer.Span) (int, int, string) {
			switch {
			case s.Start() != s.End():
				return s.Start(), s.End(), ""
			case before:
				return max(s.Start()-1, 0), s.End(), ""
			}
			return s.Start(), s.End() + 1, ""
		})
	}
}

// TypedKey handles editing and cursor keys.
func (e *ScriptEditor) TypedKey(k *fyne.KeyEvent) {
	if e.OnKey != nil && e.OnKey(k) {
		return
	}
	e.Key(k.Name, e.shift) // Shift as held (selecting)
}

// Key does what the key does, past OnKey (a vim mode's motions use the
// cursor keys); with extend, as with Shift held: the selection grows.
func (e *ScriptEditor) Key(name fyne.KeyName, extend bool) {
	shift := e.shift
	e.shift = extend
	defer func() { e.shift = shift }()
	k := &fyne.KeyEvent{Name: name}
	e.fresh()
	cur := e.buf.Cursor()
	row, col := e.layout.RowCol(cur)
	move := func(off int) {
		e.buf.SetCursor(off, e.shift)
		e.changed(false)
	}
	vertical := func(dRow int) {
		if e.goalCol < 0 {
			e.goalCol = col
		}
		target := row + dRow
		switch {
		case target < 0:
			move(0)
		case target >= len(e.layout.Rows):
			move(e.buf.Len())
		default:
			move(e.layout.Offset(target, e.goalCol))
		}
	}

	switch k.Name {
	case fyne.KeyUp:
		vertical(-1)
		return
	case fyne.KeyDown:
		vertical(1)
		return
	case fyne.KeyPageUp:
		vertical(-e.pageRows())
		return
	case fyne.KeyPageDown:
		vertical(e.pageRows())
		return
	}
	e.goalCol = -1

	key := k.Name
	if e.rowRTL(row) { // the arrows move the way the text is drawn
		switch key {
		case fyne.KeyLeft:
			key = fyne.KeyRight
		case fyne.KeyRight:
			key = fyne.KeyLeft
		}
	}
	switch key {
	case fyne.KeyLeft:
		if s, _ := e.buf.Selection(); e.buf.HasSelection() && !e.shift {
			move(s)
		} else {
			move(cur - 1)
		}
	case fyne.KeyRight:
		if _, end := e.buf.Selection(); e.buf.HasSelection() && !e.shift {
			move(end)
		} else {
			move(cur + 1)
		}
	case fyne.KeyHome:
		move(e.layout.Offset(row, 0))
	case fyne.KeyEnd:
		move(e.layout.Offset(row, math.MaxInt32))
	case fyne.KeyBackspace:
		e.deleteAt(true)
		e.changed(true)
	case fyne.KeyDelete:
		e.deleteAt(false)
		e.changed(true)
	case fyne.KeyReturn, fyne.KeyEnter:
		if !e.buf.HasOthers() && e.OnEnter != nil && e.OnEnter() {
			return
		}
		e.insert("\n")
		e.changed(true)
	case fyne.KeyTab:
		if e.shift && e.OnShiftTab != nil && e.OnShiftTab() {
			return
		}
		if e.OnTab != nil && e.OnTab() {
			return
		}
		e.buf.Insert("    ")
		e.changed(true)
	}
}

// AcceptsTab keeps Tab in the editor (element completion, indenting):
// without it Fyne moved the focus on, and Tab never reached the script.
func (e *ScriptEditor) AcceptsTab() bool { return true }

func (e *ScriptEditor) pageRows() int {
	return max(int(e.Size().Height/e.cellHeight())-1, 1)
}

// TypedShortcut handles clipboard, select all, undo and redo.
func (e *ScriptEditor) TypedShortcut(s fyne.Shortcut) {
	if e.OnShortcut != nil && e.OnShortcut(s) {
		return
	}
	// the editor keeps Tab (AcceptsTab): Ctrl+Tab and Ctrl+Shift+Tab move
	// the focus on, as in other editors that take Tab
	if cs, ok := s.(*desktop.CustomShortcut); ok && cs.KeyName == fyne.KeyTab {
		if c := fyne.CurrentApp().Driver().CanvasForObject(e); c != nil {
			switch cs.Modifier {
			case fyne.KeyModifierControl:
				c.FocusNext()
				return
			case fyne.KeyModifierControl | fyne.KeyModifierShift:
				c.FocusPrevious()
				return
			}
		}
	}
	switch sc := s.(type) {
	case *fyne.ShortcutCopy:
		if e.buf.HasSelection() {
			sc.Clipboard.SetContent(e.buf.SelectedText())
		}
	case *fyne.ShortcutCut:
		if e.buf.HasSelection() {
			sc.Clipboard.SetContent(e.buf.SelectedText())
			e.buf.DeleteBackward()
			e.changed(true)
		}
	case *fyne.ShortcutPaste:
		if text := sc.Clipboard.Content(); text != "" {
			s, end := e.buf.Selection()
			e.buf.Replace(s, end, text)
			e.changed(true)
		}
	case *fyne.ShortcutSelectAll:
		e.buf.SelectAll()
		e.changed(false)
	case *fyne.ShortcutUndo:
		e.Undo()
	case *fyne.ShortcutRedo:
		e.Redo()
	}
}

// NewScroll puts the editor in a vertical scroll container that keeps the
// cursor in view as it moves.
func NewScroll(e *ScriptEditor) *container.Scroll {
	return newScroll(e, e)
}

// NewPageScroll is NewScroll with the text in a centred column at most
// columns characters wide (plus the line number gutter), like a page;
// the margins scroll too.
func NewPageScroll(e *ScriptEditor, columns int) *container.Scroll {
	return newScroll(e, newPage(e, columns))
}

// page holds the editor's column; a click in the margins beside it goes
// to the editor at the nearest column (left of a line: its start).
type page struct {
	widget.BaseWidget
	e       *ScriptEditor
	content *fyne.Container
}

func newPage(e *ScriptEditor, columns int) *page {
	p := &page{e: e, content: container.New(&pageLayout{e: e, columns: columns}, e)}
	p.ExtendBaseWidget(p)
	return p
}

func (p *page) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(p.content) }

// Tapped handles clicks in the margins (clicks on the text reach the
// editor directly).
func (p *page) Tapped(ev *fyne.PointEvent) {
	at := ev.Position.Subtract(p.e.Position())
	at.X = min(max(at.X, 0), p.e.Size().Width-1)
	at.Y = min(max(at.Y, 0), p.e.Size().Height-1)
	p.e.Tapped(&fyne.PointEvent{Position: at, AbsolutePosition: ev.AbsolutePosition})
}

func newScroll(e *ScriptEditor, content fyne.CanvasObject) *container.Scroll {
	s := container.NewVScroll(content)
	// the part of the editor in view; a screen's height until the
	// scroll has a size of its own
	inView := func() {
		h := s.Size().Height
		if h < 1 {
			h = 2000
		}
		e.setView(s.Offset.Y-e.Position().Y, h)
	}
	e.viewed, e.viewHeight = true, 2000
	s.OnScrolled = func(fyne.Position) { inView() }
	e.onView = inView
	e.onCursorMoved = func(p fyne.Position, h float32) {
		view := s.Size().Height
		y := s.Offset.Y
		switch {
		case p.Y < y:
			y = p.Y
		case p.Y+h > y+view:
			y = p.Y + h - view
		default:
			return
		}
		s.ScrollToOffset(fyne.NewPos(s.Offset.X, max(y, 0)))
		inView()
	}
	return s
}

// pageLayout centres the editor in a column of a given number of
// characters, or the full width when the window is narrower.
type pageLayout struct {
	e       *ScriptEditor
	columns int
}

func (l *pageLayout) width(avail float32) float32 {
	columns := l.columns
	if l.e.PageColumns > 0 {
		columns = l.e.PageColumns
	}
	w := l.e.cellWidth()*float32(columns+l.e.gutter()) + 1
	return min(w, avail)
}

// top is the space above the first line.
func (l *pageLayout) top() float32 { return 3 * theme.Padding() }

func (l *pageLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	w := l.width(size.Width)
	l.e.Resize(fyne.NewSize(w, max(size.Height-l.top(), l.e.MinSize().Height)))
	l.e.Move(fyne.NewPos((size.Width-w)/2, l.top()))
	if l.e.onView != nil {
		l.e.onView() // the rows in view, at the new size
	}
}

func (l *pageLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	return l.e.MinSize().Add(fyne.NewSize(0, l.top()))
}

var (
	_ fyne.Focusable      = (*ScriptEditor)(nil)
	_ fyne.Draggable      = (*ScriptEditor)(nil)
	_ fyne.DoubleTappable = (*ScriptEditor)(nil)
	_ desktop.Cursorable  = (*ScriptEditor)(nil)
	_ desktop.Hoverable   = (*ScriptEditor)(nil)
	_ fyne.Shortcutable   = (*ScriptEditor)(nil)
	_ fyne.Tappable       = (*ScriptEditor)(nil)
	_ desktop.Keyable     = (*ScriptEditor)(nil)
)

// classifyTyping is syntax.ClassifyTypingIn by the editor's Dialect, its
// cast kept between keys (line -1: no cursor line).
func (e *ScriptEditor) classifyTyping(lines []string, line int) []syntax.Kind {
	if e.Dialect == nil {
		return syntax.ClassifyTyping(lines, e.SceneStarts, line)
	}
	if e.casts == nil || e.casts.Dialect() != e.Dialect {
		e.casts = e.Dialect.NewCastCache()
	}
	return syntax.ClassifyTypingCached(lines, e.SceneStarts, e.casts, line)
}
