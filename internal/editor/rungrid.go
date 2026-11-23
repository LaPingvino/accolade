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

	"github.com/LaPingvino/accolade/internal/editor/wrap"
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
	e    *ScriptEditor // its theme and cell size
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
	nt, nr, ni := 0, 0, 0
	for y, row := range r.g.Rows {
		cells := row.Cells
		top := float32(y) * ch
		var bd rowBidi
		if y < len(r.g.Bidi) {
			bd = r.g.Bidi[y]
		}
		cols := drawnCols(cells, bd) // where each cell is drawn
		end := min(bd.From+len(bd.Levels), len(cells))
		left := func(i int) float32 { return float32(cols[i]) * cw }
		width := func(i int) float32 { return float32(cellWidth(cells[i].Rune)) * cw }
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
			for j < len(cells) && narrow(cells[j].Rune) && cells[j].Rune != wideFill &&
				(j > i && inCluster(cells, j) || // a cluster is drawn whole (क्ष, कि), in its first cell's style
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
				// with its marks (a variation selector), what a joiner joins
				// to it (👩‍💻) and a flag's second letter: one picture
				prev := cells[i].Rune
				for j < len(cells) {
					c := cells[j].Rune
					joined := prev == 0x200D || (isFlagLetter(prev) && isFlagLetter(c) && j == i+1) ||
						wrap.JoinsBefore(prev, c) // a cluster (in a font that lacks its script)
					if c != wideFill && cellWidth(c) != 0 && !joined {
						break
					}
					if c != wideFill {
						b.WriteRune(c)
						prev = c
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
				x += float32(cellWidth(row.Cells[i].Rune)) * cw
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

// inCluster tells whether cell j belongs to the cluster of the cell
// before it (wrap.JoinsBefore): a text run goes on over it, whatever its
// style, as the shaper draws a cluster only whole (Devanagari's
// conjuncts, the i-matra before its consonant, the reph over the next).
func inCluster(cells []widget.TextGridCell, j int) bool {
	return j > 0 && wrap.JoinsBefore(cells[j-1].Rune, cells[j].Rune)
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

// narrow: a character as wide as a cell, or a mark on one, that the
// editor's font has (wider ones, from the CJK ranges on, and those the
// font lacks, drawn in another, are drawn one by one, each in its cells).
func narrow(r rune) bool {
	return wrap.Width(r) == 0 || (r < 0x1100 && (r < 0x250 || inFont(r)))
}

var (
	fontRunes   = map[rune]bool{}
	fontParsed  *sfnt.Font
	fontOf      fyne.Resource
	fontRunesMu sync.Mutex
)

// inFont reports whether the editor's font (the theme's monospace one)
// has a character.
func inFont(r rune) bool {
	fontRunesMu.Lock()
	defer fontRunesMu.Unlock()
	res := theme.TextMonospaceFont()
	if app := fyne.CurrentApp(); app != nil {
		res = app.Settings().Theme().Font(fyne.TextStyle{Monospace: true})
	}
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
