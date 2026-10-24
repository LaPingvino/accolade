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
	"golang.org/x/text/unicode/bidi"

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
	// Mirror, for a right-to-left row, is the cells of its text [from, end)
	// drawn right to left; cells after them (the cursor at the line's end)
	// go left of them
	Mirror [][2]int
	e      *ScriptEditor // its theme and cell size
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
		// x[i] is cell i's column: marks take none
		x := make([]float32, len(cells)+1)
		for i, c := range cells {
			x[i+1] = x[i] + float32(cellWidth(c.Rune))*cw
		}
		var m [2]int
		if y < len(r.g.Mirror) {
			m = r.g.Mirror[y]
			m[1] = min(m[1], len(cells))
		}
		left := func(i int) float32 { // where cell i is drawn
			switch {
			case m[1] <= m[0] || i < m[0]:
				return x[i]
			case i < m[1]:
				return x[m[0]] + x[m[1]] - x[i+1]
			}
			return x[m[0]] - (x[i+1] - x[m[1]])
		}
		span := func(i, j int) (float32, float32) { // where cells [i, j) are drawn
			l, rt := left(i), left(i)+x[i+1]-x[i]
			for k := i + 1; k < j; k++ {
				l, rt = min(l, left(k)), max(rt, left(k)+x[k+1]-x[k])
			}
			return l, rt - l
		}
		// runs do not cross into or out of the mirrored cells
		boundary := func(j int) bool { return m[1] > m[0] && (j == m[0] || j == m[1]) }
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
		for i := 0; i < len(cells); { // texts
			style, col := styleOf(cells[i].Style), colorOf(cells[i].Style, fg)
			var b strings.Builder
			j := i
			inMirror := m[1] > m[0] && i >= m[0] && i < m[1]
			for j < len(cells) && (j == i || !boundary(j)) && (!inMirror || j == i || sameWay(cells[i].Rune, cells[j].Rune)) &&
				narrow(cells[j].Rune) && cells[j].Rune != wideFill && styleOf(cells[j].Style) == style && sameColor(colorOf(cells[j].Style, fg), col) {
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
					joined := prev == 0x200D || (isFlagLetter(prev) && isFlagLetter(c) && j == i+1)
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
					img.Image = wideGlyph(b.String(), col, style.Bold, size, w, ch)
					img.Move(fyne.NewPos(at, top))
					img.Resize(fyne.NewSize(w, ch))
					img.Refresh()
					i = j
					continue
				}
			}
			if text := b.String(); strings.TrimSpace(text) != "" {
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

// sameWay reports whether two characters of a right-to-left row are drawn
// in one piece: both written left to right (numbers, Latin) or both right
// to left (Arabic, joined); spaces and signs stand each in its own cell.
func sameWay(a, b rune) bool {
	way := func(r rune) int {
		p, _ := bidi.LookupRune(r)
		switch p.Class() {
		case bidi.L, bidi.EN, bidi.AN:
			return 1
		case bidi.R, bidi.AL:
			return 2
		case bidi.NSM:
			return 3 // a mark goes with what it is on
		}
		return 0
	}
	wa, wb := way(a), way(b)
	return wa != 0 && (wb == wa || wb == 3)
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
