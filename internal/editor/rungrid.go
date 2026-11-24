package editor

import (
	"image"
	"image/color"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/image/font/sfnt"

	"github.com/LaPingvino/lexington/lekhan"
)

// runGrid draws rows of styled cells as runs: a text for each stretch of
// cells in one style, a rectangle for each stretch of one background. A
// TextGrid makes objects for every cell, thousands for a screen of
// script: too many to scroll smoothly in the browser. Characters wider
// than a cell (CJK, emoji) are drawn one by one, so each stays in its cell.
type runGrid struct {
	widget.BaseWidget
	Rows []widget.TextGridRow
	// Bidi is how each row's text is ordered (right-to-left text, and
	// left-to-right runs in it)
	Bidi []rowBidi
	// Cols are the columns each row's cells take (rowCells)
	Cols [][]cellCol
	e    *ScriptEditor // its theme and cell size
}

// cellCol is what a grid cell's rune does not tell by itself: the columns
// the cell takes, 0 or 1 (a mark none; a Devanagari cluster, by Lekhan
// monospace, its first rune one and a conjunct's second consonant one,
// its other runes none; a wide character one, its wideFill the other),
// and whether it is drawn in one piece with the cell before it (a rune
// in the cluster of the one before, a wideFill).
type cellCol struct {
	W    int8
	Join bool
}

// rowBuilder makes a grid row: its cells and their columns.
type rowBuilder struct {
	cells []widget.TextGridCell
	cols  []cellCol
}

// add puts a cell for a rune taking w columns (2: a wide character, its
// second column a wideFill), join: in the cluster of the cell before.
func (b *rowBuilder) add(c widget.TextGridCell, w int, join bool) {
	b.cells = append(b.cells, c)
	b.cols = append(b.cols, cellCol{W: int8(min(w, 1)), Join: join})
	if w == 2 {
		b.cells = append(b.cells, widget.TextGridCell{Rune: wideFill, Style: c.Style})
		b.cols = append(b.cols, cellCol{W: 1, Join: true})
	}
}

// plain puts a cell of one column, apart (a gutter's, a space).
func (b *rowBuilder) plain(r rune, style widget.TextGridStyle) {
	b.add(widget.TextGridCell{Rune: r, Style: style}, 1, false)
}

// label puts a label's runes, each in its cluster's cells (lekhan.Cells),
// in the style styleAt gives each.
func (b *rowBuilder) label(rs []rune, styleAt func(k int) widget.TextGridStyle) {
	for k, w := range lekhan.Cells(rs) {
		b.add(widget.TextGridCell{Rune: rs[k], Style: styleAt(k)}, w, k > 0 && !lekhan.Boundary(rs, k))
	}
}

// clusterCells is where the cells of the cluster that starts at cell i
// end (the cells after it drawn in one piece with it).
func clusterCells(cols []cellCol, i int) int {
	j := i + 1
	for j < len(cols) && cols[j].Join {
		j++
	}
	return j
}

func newRunGrid(e *ScriptEditor) *runGrid {
	g := &runGrid{e: e}
	g.ExtendBaseWidget(g)
	return g
}

func (g *runGrid) CreateRenderer() fyne.WidgetRenderer {
	r := &runRenderer{g: g}
	r.Refresh()
	return r
}

type runRenderer struct {
	g       *runGrid
	texts   []*canvas.Text
	rects   []*canvas.Rectangle
	images  []*canvas.Image
	objects []fyne.CanvasObject
}

func (r *runRenderer) Layout(fyne.Size) {}
func (r *runRenderer) Destroy()         {}

func (r *runRenderer) MinSize() fyne.Size {
	cols := 0
	for _, row := range r.g.Rows {
		cols = max(cols, len(row.Cells))
	}
	return fyne.NewSize(float32(cols)*r.g.e.cellWidth(), float32(len(r.g.Rows))*r.g.e.cellHeight())
}

func (r *runRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *runRenderer) Refresh() {
	th := r.g.e.Theme()
	v := theme.VariantLight
	if app := fyne.CurrentApp(); app != nil {
		v = app.Settings().ThemeVariant()
	}
	fg := th.Color(theme.ColorNameForeground, v)
	size := th.Size(theme.SizeNameText)
	cw, ch := r.g.e.cellWidth(), r.g.e.cellHeight()
	mono := th.Font(fyne.TextStyle{Monospace: true})
	nt, nr, ni := 0, 0, 0
	for y, row := range r.g.Rows {
		cells, cc := row.Cells, r.g.Cols[y]
		top := float32(y) * ch
		var bd rowBidi
		if y < len(r.g.Bidi) {
			bd = r.g.Bidi[y]
		}
		cols := drawnCols(cc, bd) // where each cell is drawn
		end := min(bd.From+len(bd.Levels), len(cells))
		left := func(i int) float32 { return float32(cols[i]) * cw }
		width := func(i int) float32 { return float32(cc[i].W) * cw }
		span := func(i, j int) (float32, float32) { // where cells [i, j) are drawn
			l, rt := left(i), left(i)+width(i)
			for k := i + 1; k < j; k++ {
				l, rt = min(l, left(k)), max(rt, left(k)+width(k))
			}
			return l, rt - l
		}
		// runs do not cross into or out of the reordered text, nor from one
		// level to another in it (drawn apart)
		boundary := func(j int) bool {
			return bd.reordered() && (j == bd.From || j == end || bd.level(j) != bd.level(j-1))
		}
		for i := 0; i < len(cells); { // backgrounds
			bg := backgroundOf(cells[i].Style)
			j := i + 1
			for j < len(cells) && !boundary(j) && sameColor(backgroundOf(cells[j].Style), bg) {
				j++
			}
			if bg != nil {
				rect := r.rect(nr)
				nr++
				rect.FillColor = bg
				at, w := span(i, j)
				rect.Move(fyne.NewPos(at, top))
				rect.Resize(fyne.NewSize(w, ch))
			}
			i = j
		}
		var joins []bool // in the reordered text: a cell drawn in one piece with the one before
		if bd.reordered() {
			rs := make([]rune, end-bd.From)
			for k := range rs {
				rs[k] = cells[bd.From+k].Rune
			}
			joins = make([]bool, len(cells))
			copy(joins[bd.From:], bidiJoins(rs, bd.Levels[:len(rs)]))
		}
		for i := 0; i < len(cells); { // texts
			style, col := styleOf(cells[i].Style), colorOf(cells[i].Style, fg)
			var b strings.Builder
			j := i
			inText := bd.reordered() && i >= bd.From && i < end
			// in the reordered text a piece starts with a letter of its way
			// (one cut off from its letters by a style or a wide character
			// stands alone)
			alone := inText && bd.level(i) > 0 && !leads(cells[i].Rune, int8(bd.level(i)))
			for j < len(cells) && narrow(mono, cells[j].Rune, cc[j].W) && cells[j].Rune != wideFill &&
				(j > i && cc[j].Join || // a cluster is drawn whole (क्ष, कि), in its first cell's style
					(j == i || !boundary(j) && !alone) && (!inText || j == i || joins[j]) &&
						styleOf(cells[j].Style) == style && sameColor(colorOf(cells[j].Style, fg), col)) {
				b.WriteRune(cells[j].Rune)
				j++
			}
			if j == i { // a wide character, alone in its cells
				if cells[i].Rune != wideFill {
					b.WriteRune(cells[i].Rune)
				}
				j++
				// with its cluster: its marks (a variation selector), what a
				// joiner joins to it (👩‍💻), a script's signs (in a font that
				// lacks the script), and a flag's second letter: one picture
				for j < len(cells) {
					c := cells[j].Rune
					flag := j == i+1 && isFlagLetter(cells[i].Rune) && isFlagLetter(c)
					if !cc[j].Join && cc[j].W != 0 && !flag {
						break
					}
					if c != wideFill {
						b.WriteRune(c)
					}
					j++
				}
				if wideGlyph != nil && b.Len() > 0 {
					img := r.image(ni)
					ni++
					at, w := span(i, j)
					img.Image, img.FillMode = wideGlyph(b.String(), col, style.Bold, size, w, ch), canvas.ImageFillStretch
					img.Move(fyne.NewPos(at, top))
					img.Resize(fyne.NewSize(w, ch))
					img.Refresh()
					i = j
					continue
				}
			}
			if text := b.String(); strings.TrimSpace(text) != "" {
				if bd.level(i)%2 == 1 && j == i+1 {
					text = mirrorBracket(text) // "(" drawn right to left is ")"
				}
				t := r.text(nt)
				nt++
				t.Text, t.Color, t.TextSize = text, col, size
				t.TextStyle = style
				t.FontSource = th.Font(style)
				at, w := span(i, j)
				t.Move(fyne.NewPos(at, top))
				t.Resize(fyne.NewSize(w, ch))
			}
			i = j
		}
	}
	for y, row := range r.g.Rows { // the pictures over their cells (a chip's thumbnail)
		top := float32(y) * ch
		x := float32(0)
		for i := 0; i < len(row.Cells); i++ {
			pc, ok := row.Cells[i].Style.(*pictureCell)
			if !ok || !pc.lead {
				x += float32(r.g.Cols[y][i].W) * cw
				continue
			}
			j, w := i, float32(0)
			for ; j < len(row.Cells); j++ {
				if q, ok := row.Cells[j].Style.(*pictureCell); !ok || q.img != pc.img || (j > i && q.lead) {
					break
				}
				w += cw
			}
			img := r.image(ni)
			ni++
			img.Image, img.FillMode = pc.img, canvas.ImageFillContain
			img.Move(fyne.NewPos(x+1, top+1))
			img.Resize(fyne.NewSize(w-2, ch-2))
			img.Refresh()
			x += w
			i = j - 1
		}
	}
	r.objects = r.objects[:0]
	for _, o := range r.rects[:nr] {
		r.objects = append(r.objects, o)
	}
	for _, o := range r.texts[:nt] {
		r.objects = append(r.objects, o)
	}
	for _, o := range r.images[:ni] {
		r.objects = append(r.objects, o)
	}
}

// wideGlyph, when set, draws a wide character as a picture w by h of
// size, in color: in the browser, whose fonts have the CJK the bundled
// one lacks (glyph_js.go). The desktop's fonts fall back by themselves.
var wideGlyph func(s string, c color.Color, bold bool, size, w, h float32) image.Image

// mirrorBracket is a bracket as drawn right to left: its mirror.
func mirrorBracket(s string) string {
	pairs := map[string]string{"(": ")", ")": "(", "[": "]", "]": "[", "{": "}", "}": "{",
		"<": ">", ">": "<", "«": "»", "»": "«", "‹": "›", "›": "‹"}
	if m, ok := pairs[s]; ok {
		return m
	}
	return s
}

// isFlagLetter: a regional indicator, two of which make a flag (🇳🇱).
func isFlagLetter(r rune) bool { return r >= 0x1F1E6 && r <= 0x1F1FF }

func (r *runRenderer) image(i int) *canvas.Image {
	if i == len(r.images) {
		img := canvas.NewImageFromImage(nil)
		img.FillMode = canvas.ImageFillStretch
		img.ScaleMode = canvas.ImageScaleSmooth
		r.images = append(r.images, img)
	}
	return r.images[i]
}

// text and rect are the i-th text and rectangle, made when there are not
// that many yet (they are kept for the next refresh).
func (r *runRenderer) text(i int) *canvas.Text {
	if i == len(r.texts) {
		r.texts = append(r.texts, canvas.NewText("", nil))
	}
	return r.texts[i]
}

func (r *runRenderer) rect(i int) *canvas.Rectangle {
	if i == len(r.rects) {
		r.rects = append(r.rects, canvas.NewRectangle(color.Transparent))
	}
	return r.rects[i]
}

// narrow: a character as wide as a cell, or one taking no column of its
// own (w, its cellCol's: a mark, a cluster's sign), that the editor's font
// has (wider ones, from the CJK ranges on, and those the font lacks, drawn
// in another, are drawn one by one, each in its cells). font is the one the
// row is drawn in (the editor theme's monospace one).
func narrow(font fyne.Resource, r rune, w int8) bool {
	return w == 0 || (r < 0x1100 && (r < 0x250 || inFont(font, r)))
}

var (
	fontRunes   = map[rune]bool{}
	fontParsed  *sfnt.Font
	fontOf      fyne.Resource
	fontRunesMu sync.Mutex
)

// inFont reports whether a font (the editor theme's monospace one, not
// the app theme's: that is Fyne's, without Devanagari, and a cluster of a
// word drawn apart from the next lost the font's joins, the headline
// between them) has a character.
func inFont(res fyne.Resource, r rune) bool {
	fontRunesMu.Lock()
	defer fontRunesMu.Unlock()
	if res != fontOf {
		fontOf, fontParsed = res, nil
		clear(fontRunes)
		if res != nil {
			fontParsed, _ = sfnt.Parse(res.Content())
		}
	}
	has, ok := fontRunes[r]
	if !ok {
		has = true // unknown: as the font draws it
		if fontParsed != nil {
			var buf sfnt.Buffer
			if g, err := fontParsed.GlyphIndex(&buf, r); err == nil {
				has = g != 0
			}
		}
		fontRunes[r] = has
	}
	return has
}

func styleOf(s widget.TextGridStyle) fyne.TextStyle {
	st := fyne.TextStyle{Monospace: true}
	if s != nil {
		st = s.Style()
		st.Monospace = true
	}
	return st
}

func colorOf(s widget.TextGridStyle, fg color.Color) color.Color {
	if s != nil && s.TextColor() != nil {
		return s.TextColor()
	}
	return fg
}

func backgroundOf(s widget.TextGridStyle) color.Color {
	if s == nil {
		return nil
	}
	return s.BackgroundColor()
}

func sameColor(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	r1, g1, b1, a1 := a.RGBA()
	r2, g2, b2, a2 := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}

// RowText is row i's text, as TextGrid has it (tests read what is shown).
func (g *runGrid) RowText(i int) string {
	var b strings.Builder
	for _, c := range g.Rows[i].Cells {
		if c.Rune != wideFill {
			b.WriteRune(c.Rune)
		}
	}
	return b.String()
}
