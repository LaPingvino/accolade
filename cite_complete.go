package main

import (
	"fmt"
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/cite"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/editor/syntax"
)

// The @ list: in a Markdown document with a library, a key being typed
// after @ ([@do, @do, [-@do) lists the entries that fit under the
// cursor, the ex line's list's rows (ex_window.go): the key, then
// author, year and title. ↑↓ choose, Tab or Enter puts the key in, Esc
// puts the list away (until another @), a tap takes one. It follows the
// text, not the @ key: vim's normal mode, an undo or a paste open it
// only where a key is being written in insert mode.

// citeListMax is how many entries the list holds (it scrolls through
// them as ↑↓ go).
const citeListMax = 50

// citePop is the list, a layer of the window as the ex line's is: a
// pop-up would take the keyboard from the editor.
type citePop struct {
	w      *MainWindow
	layer  *fyne.Container
	panel  *fyne.Container
	rows   *fyne.Container
	hint   *widget.Label
	bg     *canvas.Rectangle
	shown  []*exRow
	narrow bool
	col    float32

	start     int // the @'s offset in the text
	typed     string
	entries   []*cite.Entry
	chosen    int
	dismissed int // the @ Esc put the list away at (-1: none)
	// waiting is the @ typed while the library was being read (-1: none):
	// the list opens there once it is
	waiting int
}

func (w *MainWindow) citePopup() *citePop {
	if w.citePop == nil {
		p := &citePop{w: w, dismissed: -1, start: -1, waiting: -1}
		p.rows = container.NewVBox()
		p.hint = widget.NewLabel("")
		p.hint.SizeName = theme.SizeNameCaptionText
		p.hint.Truncation = fyne.TextTruncateEllipsis
		p.bg = canvas.NewRectangle(theme.Color(theme.ColorNameMenuBackground))
		p.bg.StrokeWidth = 1
		p.bg.CornerRadius = theme.InputRadiusSize()
		p.panel = container.NewStack(p.bg, container.NewBorder(nil, p.hint, nil, nil, p.rows))
		p.layer = container.New(&citePopLayout{p: p}, p.panel)
		p.layer.Hide()
		w.citePop = p
		w.citations().changed = append(w.citations().changed, func() {
			if p.open() {
				p.w.citeFollow(false) // the library read again: the list from it
			} else if p.waiting >= 0 {
				p.w.citeFollow(true) // read at last: the list for the @ typed meanwhile
			}
		})
	}
	return w.citePop
}

func (p *citePop) open() bool { return p != nil && p.layer.Visible() }

// citeTyping is the key being written at the cursor: the @'s offset and
// what follows it up to the cursor.
func (w *MainWindow) citeTyping() (start int, typed string, ok bool) {
	e := w.textEditor
	if !w.markdown || e == nil || e.Buffer().HasSelection() {
		return 0, "", false
	}
	if w.vim != nil && w.vim.active() && w.vim.mode != vimInsert {
		return 0, "", false
	}
	b := e.Buffer()
	cursor := b.Cursor()
	li, col := b.Position(cursor)
	line := b.Line(li)
	if !strings.ContainsRune(line, '@') {
		return 0, "", false
	}
	s, typed, ok := syntax.CiteTyped(line, col)
	if !ok {
		return 0, "", false
	}
	kinds, ok := e.MarkdownKinds() // as laid out; out of sight, not laid out since: read here
	if !ok {
		kinds = syntax.ClassifyMarkdown(strings.Split(e.Text(), "\n"))
	}
	if li < len(kinds) {
		switch kinds[li] {
		case syntax.MdCode, syntax.MdCodeFence, syntax.MdFrontMatter, syntax.MdComment:
			return 0, "", false
		}
	}
	return cursor - col + s, typed, true
}

// citeFollow shows, changes or hides the list as the text and the
// cursor are now. Only an edit opens it: the cursor put in a citation
// (a click, a move) does not, it changes or closes the list open.
func (w *MainWindow) citeFollow(edited bool) {
	if w.citePop == nil && !w.markdown {
		return
	}
	p := w.citePopup()
	if edited {
		w.citations().tipRev = w.textEditor.Buffer().Rev() // the cursor's moves were the edit's (citeTipAtCursor)
	}
	if !edited && !p.open() {
		return
	}
	start, typed, ok := w.citeTyping()
	if !ok || start != p.dismissed {
		p.dismissed = -1
	}
	if !ok || start == p.dismissed {
		p.hide()
		return
	}
	lib := w.library()
	p.waiting = -1
	if lib == nil {
		p.hide()
		if w.citations().loading && edited {
			p.waiting = start // the list once the library is read
		}
		if typed == "" {
			w.setStatus(w.libraryNote())
		}
		return
	}
	es := lib.Match(typed, citeListMax)
	if len(es) == 0 {
		p.hide()
		return
	}
	if start != p.start || typed != p.typed {
		p.chosen = 0
	}
	p.start, p.typed, p.entries = start, typed, es
	p.chosen = min(p.chosen, len(es)-1)
	p.show()
}

func (p *citePop) hide() {
	if p.layer.Visible() {
		p.layer.Hide()
	}
	p.start, p.typed, p.entries = -1, "", nil
}

// show lists the entries, the chosen one in sight.
func (p *citePop) show() {
	rows := exRowsShown
	if p.w.narrow {
		rows = 5
	}
	first := 0
	if p.chosen >= rows {
		first = p.chosen - rows + 1
	}
	last := min(first+rows, len(p.entries))
	if p.narrow != p.w.narrow {
		p.shown, p.narrow = nil, p.w.narrow
	}
	for len(p.shown) < last-first {
		p.shown = append(p.shown, newExRow(p.w.narrow, &p.col))
	}
	p.col = 0
	for k := first; k < last; k++ {
		p.col = max(p.col, fyne.MeasureText(p.entries[k].Key, theme.TextSize(), fyne.TextStyle{Monospace: true, Bold: true}).Width+2*theme.InnerPadding())
	}
	objs := make([]fyne.CanvasObject, 0, last-first)
	for k := first; k < last; k++ {
		r, i := p.shown[k-first], k
		e := p.entries[k]
		r.set(exChoice{label: e.Key, desc: e.Line()}, k == p.chosen, func() { p.take(i) })
		objs = append(objs, r)
	}
	p.rows.Objects = objs
	hint := fmt.Sprintf("%d of %d", p.chosen+1, len(p.entries))
	if p.w.narrow {
		hint += " · tap one"
	} else {
		hint += " · Tab or Enter puts it in · ↑↓ choose · Esc closes"
	}
	if c := p.entries[p.chosen]; c.Container != "" || c.Publisher != "" {
		hint = strings.TrimSpace(c.Container+" "+c.Publisher) + " — " + hint
	}
	p.hint.SetText(hint)
	p.bg.FillColor = theme.Color(theme.ColorNameMenuBackground)
	p.bg.StrokeColor = theme.Color(theme.ColorNameSeparator)
	p.bg.Refresh()
	p.rows.Refresh()
	p.layer.Show()
	p.layer.Refresh()
}

// citeKey takes the list's keys while it is open.
func (w *MainWindow) citeKey(k *fyne.KeyEvent) bool {
	p := w.citePop
	if !p.open() || len(p.entries) == 0 {
		return false
	}
	switch k.Name {
	case fyne.KeyDown:
		p.chosen = (p.chosen + 1) % len(p.entries)
	case fyne.KeyUp:
		p.chosen = (p.chosen - 1 + len(p.entries)) % len(p.entries)
	case fyne.KeyTab, fyne.KeyReturn, fyne.KeyEnter:
		p.take(p.chosen)
		return true
	case fyne.KeyEscape:
		p.dismissed = p.start
		p.hide()
		return w.vim == nil || !w.vim.active() // vim's insert mode ends too
	default:
		return false
	}
	p.show()
	return true
}

// take puts entry i's key in place of the one being written (the whole
// key at the @, the part after the cursor too).
func (p *citePop) take(i int) {
	if i < 0 || i >= len(p.entries) || p.start < 0 {
		return
	}
	e, w := p.entries[i], p.w
	key := e.Key
	if strings.IndexFunc(key, func(r rune) bool { return !citeKeyRune(r) }) >= 0 {
		key = "{" + key + "}" // @{a key with spaces}
	}
	b := w.textEditor.Buffer()
	li, col := b.Position(p.start)
	end := p.start - col + syntax.CiteKeyEnd(b.Line(li), col)
	start := p.start
	w.textEditor.Edit(func(b *buffer.Buffer) {
		b.Replace(start+1, end, key)
		b.SetCursor(start+1+len([]rune(key)), false)
	})
	p.dismissed = start // taken: not listed again for the key now there
	p.hide()
	w.fyneWindow.Canvas().Focus(w.textEditor)
}

// citeKeyRune: a rune a key may hold without braces.
func citeKeyRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("_:.#$%&-+?<>~/", r)
}

// citePopLayout puts the list under the cursor's line (over it when
// there is no room below), as wide as fits: the window's on a phone.
type citePopLayout struct{ p *citePop }

func (l *citePopLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.Size{} }

func (l *citePopLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	p, pad := l.p, theme.Padding()
	w := p.w
	width := min(size.Width-2*pad, 620)
	if w.narrow {
		width = size.Width - 2*pad
	}
	h := p.panel.MinSize().Height
	e := w.textEditor
	d := fyne.CurrentApp().Driver()
	at := d.AbsolutePositionForObject(e).Add(e.CursorPosition()).Subtract(d.AbsolutePositionForObject(p.layer))
	cell := e.CellSize()
	x := min(max(at.X-2*pad, pad), size.Width-width-pad)
	if w.narrow {
		x = pad
	}
	y := at.Y + cell.Height + pad
	if y+h > size.Height-pad && at.Y-h-pad >= 0 { // no room below: above the line
		y = at.Y - h - pad
	}
	p.panel.Resize(fyne.NewSize(width, h))
	p.panel.Move(fyne.NewPos(x, max(min(y, size.Height-h), 0)))
}
