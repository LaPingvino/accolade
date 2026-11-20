package main

import (
	"sort"
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The command palette (Help > Command Palette…, Ctrl+K or F1; not
// Ctrl+Shift+P, which is Preview): a search over every command of the
// registry that applies to the document open, each with its shortcut and
// where it lives ("Adaptation > Next Source Passage"). Typing narrows the
// list by a fuzzy match (the letters in order, word starts first); Up and
// Down choose, Enter runs, Esc closes. A command that cannot be given now
// is greyed and, run, says why on the status line, as from its menu. Open
// Recent and the like are listed by their items. The same in the browser,
// where the page keeps those keys from the browser as it does Ctrl+O.

// paletteEntry is one line of the palette.
type paletteEntry struct {
	c      *command
	label  string // with its menu's for a submenu's item
	home   string
	keys   string
	action func() // a submenu item's, else the command's
	off    bool   // it cannot be given now
}

// paletteEntries are the commands of env, in the registry's order.
func (w *MainWindow) paletteEntries(env cmdEnv) []paletteEntry {
	var out []paletteEntry
	for _, c := range commands().commands {
		if !c.applies(env) || c.id == "app.palette" || (c.id == "view.keyTips" && !w.ribbonOn()) {
			continue
		}
		home := ""
		if hs := homesOf(c.id); len(hs) > 0 {
			home = hs[0]
		}
		if c.submenu != nil {
			for _, it := range c.submenu(w).Items {
				if it.IsSeparator || it.Action == nil {
					continue
				}
				out = append(out, paletteEntry{c: c, label: c.labelIn(env) + " > " + it.Label, home: home, action: it.Action})
			}
			continue
		}
		keys := strings.Join(commandKeys(c), ", ")
		out = append(out, paletteEntry{c: c, label: c.labelIn(env), home: home, keys: keys, off: !c.canRun(env)})
	}
	return out
}

// fuzzyScore is how well query matches s (0: not at all): its letters in
// order, more for a word's start, a run of letters and an early start;
// a prefix of s most.
func fuzzyScore(query, s string) int {
	q := []rune(strings.ToLower(strings.TrimSpace(query)))
	if len(q) == 0 {
		return 1
	}
	t := []rune(strings.ToLower(s))
	if strings.HasPrefix(string(t), string(q)) {
		return 10000 - len(t)
	}
	if i := strings.Index(string(t), string(q)); i >= 0 && (i == 0 || !unicode.IsLetter(t[i-1])) {
		return 5000 - len(t)
	}
	score, qi, run, first, last, words := 0, 0, 0, -1, -1, 0
	for i := 0; i < len(t) && qi < len(q); i++ {
		if q[qi] == ' ' {
			qi++
			if qi == len(q) {
				break
			}
		}
		if t[i] != q[qi] {
			run = 0
			continue
		}
		score += 10
		if i == 0 || !unicode.IsLetter(t[i-1]) && !unicode.IsDigit(t[i-1]) {
			score += 30
			words++
		}
		if i == 0 {
			score += 20
		}
		if first < 0 {
			first = i
		}
		last = i
		run++
		score += 5 * run
		qi++
	}
	if qi < len(q) {
		return 0
	}
	// spread over a long stretch and many words: a worse match ("src"
	// is Source more than Sheets > Revision Colours)
	gaps := last - first + 1 - len(q)
	return score - 2*gaps - 25*max(0, words-1) + 100 - min(len(t), 100)
}

// paletteMatches are the entries matching query, best first (the
// registry's order among equals); the label counts more than the home.
func paletteMatches(entries []paletteEntry, query string) []paletteEntry {
	type scored struct {
		e     paletteEntry
		score int
		i     int
	}
	var ss []scored
	for i, e := range entries {
		s := fuzzyScore(query, e.label) * 2
		if h := fuzzyScore(query, e.home); h > s {
			s = h
		}
		if s > 0 {
			ss = append(ss, scored{e, s, i})
		}
	}
	sort.SliceStable(ss, func(a, b int) bool {
		if ss[a].score != ss[b].score {
			return ss[a].score > ss[b].score
		}
		return ss[a].i < ss[b].i
	})
	out := make([]paletteEntry, len(ss))
	for i, s := range ss {
		out[i] = s.e
	}
	return out
}

// palette is the palette while it is open.
type palette struct {
	w       *MainWindow
	pop     *widget.PopUp
	entry   *paletteField
	list    *widget.List
	all     []paletteEntry
	shown   []paletteEntry
	chosen  int
	from    fyne.Focusable // the focus before it opened
	running bool
}

// showPalette opens the palette (or, open, gives it the keyboard).
func (w *MainWindow) showPalette() {
	if w.palette != nil && w.palette.pop.Visible() {
		w.fyneWindow.Canvas().Focus(w.palette.entry)
		return
	}
	c := w.fyneWindow.Canvas()
	p := &palette{w: w, from: c.Focused()}
	if w.ribbon != nil {
		w.ribbon.leaveKeyTips(false)
	}
	p.all = w.paletteEntries(w.cmdEnv())
	p.entry = newPaletteField(p)
	p.entry.SetPlaceHolder("Type a command")
	fyne.SetAccessibleLabel(p.entry, "Command palette: type to find a command")
	p.list = widget.NewList(func() int { return len(p.shown) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Truncation = fyne.TextTruncateEllipsis
			home := widget.NewLabel("")
			home.SizeName = theme.SizeNameCaptionText // smaller, not fainter: it is read
			home.Truncation = fyne.TextTruncateEllipsis
			keys := widget.NewLabel("")
			keys.Importance = widget.LowImportance
			return container.NewBorder(nil, nil, nil, keys, container.NewVBox(label, home))
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			e := p.shown[id]
			box := o.(*fyne.Container)
			texts := box.Objects[0].(*fyne.Container)
			label, home := texts.Objects[0].(*widget.Label), texts.Objects[1].(*widget.Label)
			label.SetText(e.label)
			label.Importance = widget.MediumImportance
			if e.off {
				label.Importance = widget.LowImportance
			}
			label.Refresh()
			home.SetText(e.home)
			box.Objects[1].(*widget.Label).SetText(e.keys)
			box.Refresh() // the keys' width: laid out again
			name := e.label
			if e.keys != "" {
				name += ", " + e.keys
			}
			if e.home != "" {
				name += ", in " + e.home
			}
			fyne.SetAccessibleLabel(o, name)
		})
	p.list.OnSelected = func(id widget.ListItemID) {
		p.chosen = id
		if !p.running {
			p.run() // a click or a tap runs it
		}
	}
	p.entry.OnChanged = func(string) { p.filter() }
	p.entry.OnSubmitted = func(string) { p.run() }
	p.filter()
	size := c.Size()
	width := min(size.Width-2*theme.Padding(), 600)
	height := min(size.Height*0.6, 440)
	if w.narrow {
		width, height = size.Width-2*theme.Padding(), size.Height*0.55
	}
	hint := widget.NewLabel("Enter runs it · ↑↓ choose · Esc closes")
	hint.SizeName = theme.SizeNameCaptionText
	hint.Importance = widget.LowImportance
	content := container.NewBorder(p.entry, hint, nil, nil, p.list)
	p.pop = widget.NewModalPopUp(content, c)
	p.pop.OnEscape = func() bool { // Esc reaches the top overlay first
		p.close()
		return true
	}
	p.pop.Resize(fyne.NewSize(width, height))
	p.pop.Show()
	p.pop.Move(fyne.NewPos((size.Width-width)/2, min(60, size.Height/10)))
	w.palette = p
	c.Focus(p.entry)
}

// filter shows the entries matching what is typed, the best chosen.
func (p *palette) filter() {
	p.shown = paletteMatches(p.all, p.entry.Text)
	p.chosen = 0
	p.list.Refresh()
	p.mark()
}

// mark shows the chosen entry selected and in sight.
func (p *palette) mark() {
	if len(p.shown) == 0 {
		p.list.UnselectAll()
		return
	}
	p.running = true
	p.list.Select(p.chosen)
	p.list.ScrollTo(p.chosen)
	p.running = false
}

// step chooses the entry step from the chosen one.
func (p *palette) step(step int) {
	if len(p.shown) == 0 {
		return
	}
	p.chosen = (p.chosen + step + len(p.shown)) % len(p.shown)
	p.mark()
}

// close closes the palette, the focus back where it was.
func (p *palette) close() {
	p.pop.Hide()
	c := p.w.fyneWindow.Canvas()
	if p.from != nil {
		c.Focus(p.from)
	} else {
		c.Focus(p.w.textEditor)
	}
}

// run closes the palette and runs the chosen entry: with the editor's
// focus back first, so an editing command acts on the script.
func (p *palette) run() {
	if p.chosen < 0 || p.chosen >= len(p.shown) {
		return
	}
	e := p.shown[p.chosen]
	p.pop.Hide()
	w := p.w
	w.fyneWindow.Canvas().Focus(w.textEditor)
	if e.action != nil {
		e.action()
	} else {
		w.tapped(e.c, nil)()
	}
	w.ribbonRefresh()
}

// paletteField is the palette's search field: Up and Down choose, Esc
// closes.
type paletteField struct {
	widget.Entry
	p *palette
}

func newPaletteField(p *palette) *paletteField {
	f := &paletteField{p: p}
	f.ExtendBaseWidget(f)
	return f
}

func (f *paletteField) TypedKey(k *fyne.KeyEvent) {
	switch k.Name {
	case fyne.KeyEscape:
		f.p.close()
	case fyne.KeyDown:
		f.p.step(1)
	case fyne.KeyUp:
		f.p.step(-1)
	case fyne.KeyPageDown:
		f.p.step(8)
	case fyne.KeyPageUp:
		f.p.step(-8)
	default:
		f.Entry.TypedKey(k)
	}
}
