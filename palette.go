// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

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
//
// Working as a writer (role.go), Production's commands are found too once
// something is typed (not in the list before: a writer opening the
// palette is not shown production), marked "(Production)"; run, the
// script switches to Production first and says so. A writer asking for
// Lock Scene Numbers by name means it, and its result (the lock, the pane
// with Unlock) is then where it can be seen and taken back, not done
// behind a view that hides how to undo it. Only this script switches;
// View > Working As goes back.

// paletteEntry is one line of the palette.
type paletteEntry struct {
	c      *command
	ex     string   // its ex name (:fix-lines), matched too
	words  []string // its command's keywords, matched too (less than its label)
	label  string   // with its menu's for a submenu's item
	home   string
	keys   string
	action func() // a submenu item's, else the command's
	off    bool   // it cannot be given now
	away   bool   // put away by the role: found once something is typed (role.go)
}

// paletteEntries are the commands of env, in the registry's order.
func (w *MainWindow) paletteEntries(env cmdEnv) []paletteEntry {
	var out []paletteEntry
	for _, c := range commands().commands {
		if !c.appliesForRole(env) || c.id == "app.palette" || (c.id == "view.keyTips" && !w.ribbonOn()) {
			continue
		}
		away := !c.applies(env)
		home := ""
		if hs := homesOf(c.id); len(hs) > 0 {
			home = hs[0]
		}
		if c.submenu != nil {
			for _, it := range flatItems(c.submenu(w).Items) {
				if it.IsSeparator || it.Action == nil {
					continue
				}
				out = append(out, paletteEntry{c: c, label: c.labelIn(env) + " > " + it.Label, home: home, action: it.Action, away: away, words: c.keywords})
			}
			continue
		}
		keys := strings.Join(commandKeys(c), ", ")
		label := c.labelIn(env)
		if away {
			label += " (Production)"
		}
		e := env
		e.writer = false
		ex := commandExName(c)
		out = append(out, paletteEntry{c: c, label: label, home: home, keys: keys, off: !c.canRun(e), away: away, ex: ex, words: c.keywords})
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
	typed := strings.TrimSpace(query) != ""
	for i, e := range entries {
		if e.away && !typed {
			continue // the role's put away: only asked for by name
		}
		s := fuzzyScore(query, e.label) * 2
		if h := fuzzyScore(query, e.home); h > s {
			s = h
		}
		if e.ex != "" { // its name on the ':' line, as the ex completion has it
			q := strings.TrimPrefix(strings.TrimSpace(query), ":")
			if x := fuzzyScore(q, e.ex); strings.Contains(q, "-") || strings.HasPrefix(strings.TrimSpace(query), ":") {
				s = max(s, x*2)
			} else if x/2 > s {
				s = x / 2
			}
		}
		for _, k := range e.words { // a keyword: below a label's match, so "beats" lists Where the Beats Land first
			if x := fuzzyScore(query, k) / 2; x > s {
				s = x
			}
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
	hint    *widget.Label
	// pick, if set, takes the entry chosen instead of running it (Add
	// to the Top Row…: toprow.go)
	pick func(c *command)
}

// showPalette opens the palette (or, open, gives it the keyboard).
func (w *MainWindow) showPalette() {
	if w.palette != nil && w.palette.pop.Visible() {
		w.fyneWindow.Canvas().Focus(w.palette.entry)
		return
	}
	w.openPalette(w.paletteEntries(w.cmdEnv()), "Type a command", "Command palette: type to find a command")
}

// openPalette opens the palette over entries: placeholder in its field,
// name the field's for a screen reader.
func (w *MainWindow) openPalette(entries []paletteEntry, placeholder, name string) {
	c := w.fyneWindow.Canvas()
	p := &palette{w: w, from: c.Focused()}
	if w.ribbon != nil {
		w.ribbon.leaveKeyTips(false)
	}
	p.all = entries
	p.entry = newPaletteField(p)
	p.entry.SetPlaceHolder(placeholder)
	fyne.SetAccessibleLabel(p.entry, name)
	p.list = widget.NewList(func() int { return len(p.shown) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Truncation = fyne.TextTruncateEllipsis
			home := widget.NewLabel("")
			home.SizeName = theme.SizeNameCaptionText // smaller, not fainter: it is read
			home.Truncation = fyne.TextTruncateEllipsis
			keys := widget.NewLabel("")
			keys.Importance = widget.LowImportance
			return newPaletteRow(p, container.NewBorder(nil, nil, nil, keys, container.NewVBox(label, home)))
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			e := p.shown[id]
			row := o.(*paletteRow)
			row.id = id
			box := row.box
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
			fyne.SetAccessibleLabel(row.box, name)
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
	hint := widget.NewLabel("Enter runs it · ↑↓ choose · Esc closes · right-click: the top row")
	p.hint = hint
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
		p.w.focusMain()
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
	if p.pick != nil {
		p.w.focusMain()
		p.pick(e.c)
		return
	}
	p.w.runByName(e.c, e.action, e.away)
}

// runByName runs c as asked for by name (the palette, the ':' line):
// action is a submenu item's, nil for c's own; away, c is put away by
// the role, and the script works as Production first and says so.
func (w *MainWindow) runByName(c *command, action func(), away bool) {
	w.focusMain() // the main editor: an editing command acts on what is written there
	said := ""
	if away { // asked for by name: this script works as Production now
		w.setOwnRole(roleProduction)
		said = "Working as Production for this script (" + c.label + " is production's): View > Working As goes back"
		w.setStatus(said)
	}
	if action != nil {
		action()
	} else {
		w.tapped(c, nil)()
	}
	if said != "" && w.statusLabel != nil && w.statusLabel.Text != said && w.statusLabel.Text != "" {
		w.setStatus(w.statusLabel.Text + " · working as Production for this script now")
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

// paletteRow is a line of the palette: a right-click (or a long press)
// on it puts its command on the top row, or takes it off (toprow.go).
type paletteRow struct {
	widget.BaseWidget
	p   *palette
	box *fyne.Container
	id  int
}

func newPaletteRow(p *palette, box *fyne.Container) *paletteRow {
	r := &paletteRow{p: p, box: box}
	r.ExtendBaseWidget(r)
	return r
}

func (r *paletteRow) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(r.box) }

func (r *paletteRow) TappedSecondary(e *fyne.PointEvent) {
	if r.id < 0 || r.id >= len(r.p.shown) || r.p.pick != nil {
		return
	}
	en := r.p.shown[r.id]
	if en.action != nil { // a submenu's item: its command is no button
		return
	}
	item := r.p.w.topRowItem(en.c.id)
	if item == nil {
		return
	}
	act := item.Action
	item.Action = func() { r.p.close(); act() }
	widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", item), r.p.w.fyneWindow.Canvas(), e.AbsolutePosition)
}

// commandExName is c's name on the ':' line ("" if none): the palette
// matches it as the ex completion does.
func commandExName(c *command) string {
	if by, ok := exByCommand(c); ok {
		return by
	}
	if ec := exRegistryCmd(c); ec != nil {
		return ec.name
	}
	return ""
}
