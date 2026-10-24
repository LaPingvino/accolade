package editor

import (
	"image"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

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
		// x[i] is cell i's column: marks take none
		x := make([]float32, len(cells)+1)
		for i, c := range cells {
			x[i+1] = x[i] + float32(cellWidth(c.Rune))*cw
		}
		for i := 0; i < len(cells); { // backgrounds
			bg := backgroundOf(cells[i].Style)
			j := i + 1
			for j < len(cells) && sameColor(backgroundOf(cells[j].Style), bg) {
				j++
			}
			if bg != nil {
				rect := r.rect(nr)
				nr++
				rect.FillColor = bg
				rect.Move(fyne.NewPos(x[i], top))
				rect.Resize(fyne.NewSize(x[j]-x[i], ch))
			}
			i = j
		}
		for i := 0; i < len(cells); { // texts
			style, col := styleOf(cells[i].Style), colorOf(cells[i].Style, fg)
			var b strings.Builder
			j := i
			for j < len(cells) && narrow(cells[j].Rune) && cells[j].Rune != wideFill && styleOf(cells[j].Style) == style && sameColor(colorOf(cells[j].Style, fg), col) {
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
					img.Image = wideGlyph(b.String(), col, style.Bold, size, x[j]-x[i], ch)
					img.Move(fyne.NewPos(x[i], top))
					img.Resize(fyne.NewSize(x[j]-x[i], ch))
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
				t.Move(fyne.NewPos(x[i], top))
				t.Resize(fyne.NewSize(x[j]-x[i], ch))
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

// narrow: a character as wide as a cell, or a mark on one (wider ones,
// from the CJK ranges on, are drawn one by one).
func narrow(r rune) bool { return r < 0x1100 || wrap.Width(r) == 0 }

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
