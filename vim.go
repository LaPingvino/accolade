package main

import (
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// A vim mode for the editor (Preferences > Editor): Escape leaves typing
// for normal mode, where letters move and edit. What is there:
//
//	motions     h j k l  w b e  0 ^ $  gg G   (with a count: 3w)
//	insert      i a I A o O
//	edit        x X dd D cc C yy Y p P J r<c> u Ctrl+R
//	operators   d c y with a motion (dw, c$, y3j ...)
//	visual      v V, then a motion, then d x y c
//	search      / (the search bar)
//
// Each command is one undo step.

type vimMode int

const (
	vimInsert vimMode = iota
	vimNormal
	vimVisual
	vimVisualLine
)

func (m vimMode) String() string {
	switch m {
	case vimNormal:
		return "-- NORMAL --"
	case vimVisual:
		return "-- VISUAL --"
	case vimVisualLine:
		return "-- VISUAL LINE --"
	}
	return "-- INSERT --"
}

type vim struct {
	e       *editor.ScriptEditor
	on      func() bool  // whether vim mode is on (the setting)
	status  func(string) // shows the mode
	search  func()       // "/": the search bar
	mode    vimMode
	count   int  // the count typed so far
	op      rune // a pending operator: d, c or y
	opCount int  // the count before the operator
	g       bool // a pending g (gg)
	replace bool // a pending r
	reg     string
	regLine bool        // the register holds whole lines
	helix   func() bool // helix keys instead of vim's
	anchor  int         // where the visual selection started
	vcur    int         // the visual cursor (the buffer's is at the selection's end)
}

func newVim(e *editor.ScriptEditor) *vim {
	v := &vim{e: e, mode: vimInsert}
	return v
}

func (v *vim) active() bool { return v.on != nil && v.on() }

func (v *vim) setMode(m vimMode) {
	v.mode = m
	v.count, v.op, v.g, v.replace = 0, 0, false, false
	if v.helix != nil && v.helix() && m != vimInsert {
		if v.status != nil {
			v.status("-- HELIX NORMAL --")
		}
		return // helix keeps its selection
	}
	if m == vimNormal { // no selection outside visual mode
		v.e.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.Cursor(), false) })
	}
	if v.status != nil {
		v.status(m.String())
	}
}

// key handles keys that are not characters: Escape, and in normal mode
// Enter and Backspace move. It reports whether it handled the key.
func (v *vim) key(k *fyne.KeyEvent) bool {
	if !v.active() {
		return false
	}
	if k.Name == fyne.KeyEscape {
		if v.mode == vimInsert {
			// as vim: back onto the character typed last
			v.e.Navigate(func(b *buffer.Buffer) {
				t := []rune(b.Text())
				if c := b.Cursor(); c > 0 && t[c-1] != '\n' {
					b.SetCursor(c-1, false)
				}
			})
		}
		v.setMode(vimNormal)
		return true
	}
	if v.mode == vimInsert {
		return false
	}
	switch k.Name {
	case fyne.KeyReturn, fyne.KeyEnter:
		v.e.Key(fyne.KeyDown, v.visual())
		v.e.Key(fyne.KeyHome, v.visual())
		return true
	case fyne.KeyBackspace:
		v.e.Key(fyne.KeyLeft, v.visual())
		return true
	case fyne.KeyTab:
		return true
	}
	return false // arrows and the like do what they do
}

// shortcut handles Ctrl+R (redo) in normal mode.
func (v *vim) shortcut(s fyne.Shortcut) bool {
	if !v.active() || v.mode == vimInsert {
		return false
	}
	if c, ok := s.(*desktop.CustomShortcut); ok && c.KeyName == fyne.KeyR && c.Modifier&fyne.KeyModifierControl != 0 {
		v.e.Redo()
		return true
	}
	return false
}

func (v *vim) visual() bool { return v.mode == vimVisual || v.mode == vimVisualLine }

// times is the count typed, 1 without one.
func (v *vim) times() int {
	n := max(v.count, 1) * max(v.opCount, 1)
	v.count, v.opCount = 0, 0
	return n
}

// rune handles a typed character; false in insert mode (it is typed).
func (v *vim) rune(r rune) bool {
	if !v.active() || v.mode == vimInsert {
		return false
	}
	if v.replace {
		v.replace = false
		v.edit(func(b *buffer.Buffer, t []rune, c int) {
			if c < len(t) && t[c] != '\n' {
				b.Replace(c, c+1, string(r))
				b.SetCursor(c, false)
			}
		})
		return true
	}
	if r >= '1' && r <= '9' || r == '0' && v.count > 0 {
		v.count = v.count*10 + int(r-'0')
		return true
	}
	if v.g {
		v.g = false
		if r == 'g' {
			v.motion('g')
		}
		return true
	}
	if v.helix != nil && v.helix() {
		v.helixKey(r)
		return true
	}
	if v.visual() {
		v.visualKey(r)
		return true
	}
	if v.op != 0 {
		v.operatorKey(r)
		return true
	}
	v.normalKey(r)
	return true
}

func (v *vim) normalKey(r rune) {
	switch r {
	case 'i':
		v.setMode(vimInsert)
	case 'a':
		v.e.Navigate(func(b *buffer.Buffer) {
			t := []rune(b.Text())
			if c := b.Cursor(); c < len(t) && t[c] != '\n' {
				b.SetCursor(c+1, false)
			}
		})
		v.setMode(vimInsert)
	case 'I':
		v.motion('^')
		v.setMode(vimInsert)
	case 'A':
		v.motion('$')
		v.setMode(vimInsert)
	case 'o', 'O':
		v.edit(func(b *buffer.Buffer, t []rune, c int) {
			s, e := lineBounds(t, c)
			if r == 'o' {
				b.Replace(e, e, "\n")
				b.SetCursor(e+1, false)
			} else {
				b.Replace(s, s, "\n")
				b.SetCursor(s, false)
			}
		})
		v.setMode(vimInsert)
	case 'x', 'X':
		n := v.times()
		v.edit(func(b *buffer.Buffer, t []rune, c int) {
			s, e := lineBounds(t, c)
			from, to := c, min(c+n, e)
			if r == 'X' {
				from, to = max(c-n, s), c
			}
			if from < to {
				v.reg, v.regLine = string(t[from:to]), false
				b.Replace(from, to, "")
				b.SetCursor(from, false)
			}
		})
	case 'D', 'C':
		v.op = map[rune]rune{'D': 'd', 'C': 'c'}[r]
		v.operatorKey('$')
	case 'Y':
		v.op = 'y'
		v.operatorKey('y')
	case 'd', 'c', 'y':
		v.op, v.opCount, v.count = r, v.count, 0
	case 'p', 'P':
		v.put(r == 'P', v.times())
	case 'J':
		n := v.times()
		v.edit(func(b *buffer.Buffer, t []rune, c int) {
			for i := 0; i < max(n, 1); i++ {
				t = []rune(b.Text())
				_, e := lineBounds(t, b.Cursor())
				if e >= len(t) {
					return
				}
				next := e + 1
				for next < len(t) && (t[next] == ' ' || t[next] == '\t') {
					next++
				}
				b.Replace(e, next, " ")
				b.SetCursor(e, false)
			}
		})
	case 'r':
		v.replace = true
	case 'u':
		for n := v.times(); n > 0; n-- {
			v.e.Undo()
		}
	case 'v':
		v.anchor = v.e.CursorOffset()
		v.vcur = v.anchor
		v.mode = vimVisual
		v.select_()
		v.status(v.mode.String())
	case 'V':
		v.anchor = v.e.CursorOffset()
		v.vcur = v.anchor
		v.mode = vimVisualLine
		v.select_()
		v.status(v.mode.String())
	case '/':
		if v.search != nil {
			v.search()
		}
	case 'g':
		v.g = true
	default:
		v.motion(r)
	}
}

// motion moves the cursor (count times); false if r is no motion.
func (v *vim) motion(r rune) bool {
	n := v.times()
	if v.visual() { // from the visual cursor, not the selection's end
		v.e.Navigate(func(b *buffer.Buffer) { b.SetCursor(v.vcur, false) })
		defer func() {
			v.vcur = v.e.CursorOffset()
			v.select_()
		}()
	}
	switch r {
	case 'j', 'k':
		key := fyne.KeyDown
		if r == 'k' {
			key = fyne.KeyUp
		}
		for i := 0; i < n; i++ {
			v.e.Key(key, false)
		}
		return true
	}
	t := []rune(v.e.Text())
	to, ok := target(t, v.e.CursorOffset(), r, n)
	if !ok {
		return false
	}
	v.e.Navigate(func(b *buffer.Buffer) { b.SetCursor(to, false) })
	return true
}

// target is where a motion from c goes in t.
func target(t []rune, c int, r rune, n int) (int, bool) {
	s, e := lineBounds(t, c)
	switch r {
	case 'h':
		return max(c-n, s), true
	case 'l':
		return min(c+n, e), true
	case '0':
		return s, true
	case '^':
		for s < e && unicode.IsSpace(t[s]) {
			s++
		}
		return s, true
	case '$':
		return e, true
	case 'w':
		for ; n > 0; n-- {
			c = nextWord(t, c)
		}
		return c, true
	case 'b':
		for ; n > 0; n-- {
			c = prevWord(t, c)
		}
		return c, true
	case 'e':
		for ; n > 0; n-- {
			c = wordEnd(t, c)
		}
		return c, true
	case 'G':
		ls, _ := lineBounds(t, len(t))
		return ls, true
	case 'g': // gg
		return 0, true
	}
	return c, false
}

// class is a character's kind for word motions: 0 space, 1 word, 2
// punctuation.
func class(r rune) int {
	switch {
	case unicode.IsSpace(r):
		return 0
	case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '\'':
		return 1
	}
	return 2
}

func nextWord(t []rune, c int) int {
	if c >= len(t) {
		return c
	}
	k := class(t[c])
	for c < len(t) && k != 0 && class(t[c]) == k {
		c++
	}
	for c < len(t) && class(t[c]) == 0 {
		c++
	}
	return c
}

func prevWord(t []rune, c int) int {
	c--
	for c > 0 && class(t[c]) == 0 {
		c--
	}
	if c <= 0 {
		return 0
	}
	k := class(t[c])
	for c > 0 && class(t[c-1]) == k {
		c--
	}
	return c
}

func wordEnd(t []rune, c int) int {
	c++
	for c < len(t) && class(t[c]) == 0 {
		c++
	}
	if c >= len(t) {
		return max(len(t)-1, 0)
	}
	k := class(t[c])
	for c+1 < len(t) && class(t[c+1]) == k {
		c++
	}
	return c
}

// operatorKey completes a pending d, c or y: dd/cc/yy take lines, other
// keys are motions.
func (v *vim) operatorKey(r rune) {
	op := v.op
	v.op = 0
	t := []rune(v.e.Text())
	c := v.e.CursorOffset()
	n := v.times()
	var from, to int
	lines := false
	switch {
	case r == op || (op == 'y' && r == 'y'): // dd cc yy: n lines
		lines = true
		from, _ = lineBounds(t, c)
		to = c
		for i := 0; i < n; i++ {
			_, to = lineBounds(t, to)
			if i < n-1 && to < len(t) {
				to++
			}
		}
	case r == 'j' || r == 'k':
		lines = true
		from, to = c, c
		for i := 0; i < n; i++ {
			if r == 'j' {
				_, e := lineBounds(t, to)
				if e < len(t) {
					to = e + 1
				}
			} else {
				s, _ := lineBounds(t, from)
				if s > 0 {
					from = s - 1
				}
			}
		}
		from, _ = lineBounds(t, from)
		_, to = lineBounds(t, to)
	default:
		end, ok := target(t, c, r, n)
		if !ok {
			return
		}
		if r == 'e' && end < len(t) {
			end++ // e takes the word's last character
		}
		if op == 'c' && r == 'w' { // cw changes to the word's end, as ce
			end = min(wordEnd(t, c)+1, len(t))
		}
		from, to = min(c, end), max(c, end)
	}
	v.apply(op, t, from, to, lines)
}

// apply does an operator on [from, to); lines: whole lines (with their
// line break).
func (v *vim) apply(op rune, t []rune, from, to int, lines bool) {
	if lines && to < len(t) && t[to] == '\n' && op != 'c' {
		to++
	}
	v.reg, v.regLine = string(t[from:to]), lines
	if lines && !strings.HasSuffix(v.reg, "\n") {
		v.reg += "\n"
	}
	switch op {
	case 'y':
		v.e.Navigate(func(b *buffer.Buffer) { b.SetCursor(from, false) })
		v.setMode(vimNormal)
	case 'd', 'c':
		v.edit(func(b *buffer.Buffer, _ []rune, _ int) {
			if lines && op == 'd' && to == len(t) && from > 0 && t[from-1] == '\n' {
				from-- // the last line: take the line break before it
			}
			b.Replace(from, to, "")
			b.SetCursor(min(from, b.Len()), false)
		})
		if op == 'c' {
			v.setMode(vimInsert)
		} else {
			v.setMode(vimNormal)
		}
	}
}

// put pastes the register after (or before) the cursor, n times.
func (v *vim) put(before bool, n int) {
	if v.reg == "" {
		return
	}
	text := strings.Repeat(v.reg, n)
	v.edit(func(b *buffer.Buffer, t []rune, c int) {
		if v.regLine {
			s, e := lineBounds(t, c)
			at := s
			if !before {
				at = e + 1
				if e >= len(t) { // after the last line
					text = "\n" + strings.TrimSuffix(text, "\n")
					at = e
				}
			}
			b.Replace(at, at, text)
			b.SetCursor(min(at+boolInt(at == e && !before), b.Len()), false)
			return
		}
		at := c
		if !before && c < len(t) && t[c] != '\n' {
			at++
		}
		b.Replace(at, at, text)
		b.SetCursor(at+len([]rune(text))-1, false)
	})
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// visualKey is a key in visual mode: a motion extends the selection, an
// operator acts on it.
func (v *vim) visualKey(r rune) {
	switch r {
	case 'd', 'x', 'y', 'c':
		t := []rune(v.e.Text())
		from, to := v.selection(t)
		op := r
		if op == 'x' {
			op = 'd'
		}
		v.apply(op, t, from, to, v.mode == vimVisualLine)
	case 'v', 'V':
		v.setMode(vimNormal)
	case 'g':
		v.g = true
	default:
		v.motion(r)
	}
}

// selection is the visual selection: from the anchor to the cursor, the
// character under the cursor included (whole lines in V).
func (v *vim) selection(t []rune) (int, int) {
	c := v.vcur
	from, to := min(v.anchor, c), max(v.anchor, c)
	if v.mode == vimVisualLine {
		from, _ = lineBounds(t, from)
		_, to = lineBounds(t, to)
		return from, to
	}
	return from, min(to+1, len(t))
}

// select_ shows the visual selection.
func (v *vim) select_() {
	if !v.visual() {
		return
	}
	c := v.vcur
	from, to := v.selection([]rune(v.e.Text()))
	v.e.Navigate(func(b *buffer.Buffer) {
		if c < v.anchor { // the cursor stays at the moving end
			b.Select(to, from)
		} else {
			b.Select(from, to)
		}
	})
}

// edit changes the text as one undo step.
func (v *vim) edit(f func(b *buffer.Buffer, t []rune, c int)) {
	v.e.Edit(func(b *buffer.Buffer) {
		b.Group(func() { f(b, []rune(b.Text()), b.Cursor()) })
	})
}

// Helix keys (Preferences > Editor > Keys: helix): selection first.
// Motions select what they pass over, commands act on the selection:
//
//	select      w b e (v: extend)  x (the line; again: the next)  % ;
//	move        h j k l  gg G
//	act         d c y  p P  u U  r<c>
//	insert      i a I A o O
func (v *vim) helixKey(r rune) {
	t := []rune(v.e.Text())
	from, to := v.e.Buffer().Selection()
	c := v.e.CursorOffset()
	extend := v.mode == vimVisual
	sel := func(a, b int) {
		v.e.Navigate(func(bf *buffer.Buffer) { bf.Select(a, b) })
	}
	switch r {
	case 'w', 'b', 'e':
		n := v.times()
		start := c
		if extend {
			start = v.anchor
		}
		end, _ := target(t, c, r, n)
		if r == 'e' && end < len(t) {
			end++
		}
		if !extend {
			// a word: from where the motion starts, past spaces
			if r == 'w' || r == 'e' {
				for start < len(t) && start < end && class(t[start]) == 0 {
					start++
				}
			}
		} else {
			v.anchor = start
		}
		sel(start, end)
	case 'x':
		s, _ := lineBounds(t, from)
		_, e := lineBounds(t, max(to-1, from))
		if v.e.Buffer().HasSelection() && from == s && to == e+1 && e+1 < len(t) {
			_, e = lineBounds(t, e+1) // pressed again: the next line too
		}
		sel(s, min(e+1, len(t)))
	case '%':
		sel(0, len(t))
	case ';':
		v.e.Navigate(func(bf *buffer.Buffer) { bf.SetCursor(bf.Cursor(), false) })
	case 'v':
		if v.mode == vimVisual {
			v.mode = vimNormal
		} else {
			v.mode, v.anchor = vimVisual, from
		}
		v.status(map[bool]string{true: "-- HELIX SELECT --", false: "-- HELIX NORMAL --"}[v.mode == vimVisual])
	case 'd', 'c', 'y':
		if from == to { // no selection: the character under the cursor
			to = min(from+1, len(t))
		}
		lines := to > from && t[to-1] == '\n'
		v.reg, v.regLine = string(t[from:to]), lines
		if r == 'y' {
			break
		}
		v.edit(func(b *buffer.Buffer, _ []rune, _ int) {
			b.Replace(from, to, "")
			b.SetCursor(from, false)
		})
		if r == 'c' {
			v.mode = vimInsert
			v.status(vimInsert.String())
		}
	case 'p', 'P':
		at := to
		if r == 'P' {
			at = from
		}
		if v.regLine { // lines go after (or before) the selection's lines
			if r == 'P' {
				at, _ = lineBounds(t, from)
			} else if _, e := lineBounds(t, max(to-1, from)); e < len(t) {
				at = e + 1
			}
		}
		text := v.reg
		v.edit(func(b *buffer.Buffer, _ []rune, _ int) {
			b.Replace(at, at, text)
			b.Select(at, at+len([]rune(text)))
		})
	case 'u':
		v.e.Undo()
	case 'U':
		v.e.Redo()
	case 'i', 'a':
		at := from
		if r == 'a' {
			at = to
		}
		v.e.Navigate(func(bf *buffer.Buffer) { bf.SetCursor(at, false) })
		v.mode = vimInsert
		v.status(vimInsert.String())
	case 'h', 'j', 'k', 'l', '0', '^', '$', 'G', 'I', 'A', 'o', 'O', 'r':
		if r == 'h' || r == 'l' || r == 'j' || r == 'k' {
			// moving drops the selection
			v.e.Navigate(func(bf *buffer.Buffer) { bf.SetCursor(bf.Cursor(), false) })
		}
		v.normalKey(r)
	case 'g':
		v.g = true
	}
}
