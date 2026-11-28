// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/docs/help"
	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The ex command line (ex.go) at the window's foot: ":" in vim and helix
// keys opens it, Enter runs the command, Escape closes it.
//
// As it is typed, a list above it shows the commands it may be
// (ex_table.go), each with a line saying what it does: the line
// commands, the window's own (:scene, :help, the moves) and every
// command of the registry by its name. Tab completes the chosen one
// and goes on to the next (Shift+Tab back); the arrow keys choose
// without writing it in; Enter runs the line as typed, or the one
// chosen; a command wanting more (:scene, a folder) shows what it takes
// instead. On an empty line the list has the most useful commands, and
// Up goes back through the commands run before (Down into the list).
// A tap on a row (a phone's way) completes it, and runs it if it wants
// nothing more.

// exLine is the command line's entry.
type exLine struct {
	widget.Entry
	w    *MainWindow
	back int // how far back in the history (0: what is typed)

	ctx    exCtx
	stem   string // what was typed (Tab writes choices over its word)
	comp   exCompletion
	chosen int  // the choice chosen (-1 none)
	moved  bool // chosen by the arrows, not written in yet
	quiet  bool // the text set by completion, not typed
	shift  bool
}

func newExLine(w *MainWindow) *exLine {
	l := &exLine{w: w, chosen: -1}
	l.ExtendBaseWidget(l)
	l.OnSubmitted = func(string) { l.submit() }
	l.OnChanged = func(text string) {
		if l.quiet {
			return
		}
		l.back = 0
		l.typed(text)
	}
	fyne.SetAccessibleLabel(l, "Command")
	return l
}

// AcceptsTab keeps Tab in the line: it completes.
func (l *exLine) AcceptsTab() bool { return true }

func (l *exLine) KeyDown(k *fyne.KeyEvent) {
	if k.Name == desktop.KeyShiftLeft || k.Name == desktop.KeyShiftRight {
		l.shift = true
	}
	l.Entry.KeyDown(k)
}

func (l *exLine) KeyUp(k *fyne.KeyEvent) {
	if k.Name == desktop.KeyShiftLeft || k.Name == desktop.KeyShiftRight {
		l.shift = false
	}
	l.Entry.KeyUp(k)
}

// completing reports whether the line completes (not a prompt's).
func (l *exLine) completing() bool { return l.w.exPrompt == nil || len(l.w.exPromptNames) > 0 }

func (l *exLine) TypedKey(k *fyne.KeyEvent) {
	switch k.Name {
	case fyne.KeyEscape:
		l.w.hideEx()
	case fyne.KeyTab:
		if l.completing() {
			if l.shift {
				l.cycle(-1)
			} else {
				l.cycle(1)
			}
		}
	case fyne.KeyUp, fyne.KeyDown:
		up := k.Name == fyne.KeyUp
		// the list's while it is chosen in; else the history's (vim's
		// :<Up>), Down on an empty line going into the list
		if l.completing() && l.back == 0 && len(l.comp.choices) > 0 && (l.chosen >= 0 || l.Text != "" || !up) {
			l.choose(map[bool]int{true: -1, false: 1}[up])
			return
		}
		l.history(up)
	case fyne.KeyPageUp, fyne.KeyPageDown:
		if l.completing() && len(l.comp.choices) > 0 {
			l.choose(map[bool]int{true: -8, false: 8}[k.Name == fyne.KeyPageUp])
		}
	default:
		l.Entry.TypedKey(k)
	}
}

// history goes up or down through the commands run before.
func (l *exLine) history(up bool) {
	h := l.w.exState.history
	if up {
		l.back = min(l.back+1, len(h))
	} else {
		l.back = max(l.back-1, 0)
	}
	text := ""
	if l.back > 0 {
		text = h[len(h)-l.back]
	}
	l.set(text)
	if l.back > 0 { // the history's line, not a list's: Tab and Enter leave it be
		l.stem, l.comp, l.chosen, l.moved = text, exCompletion{}, -1, false
		l.w.exPopup().hide()
	} else {
		l.typed(text)
	}
}

// set writes text in the line as completion does, the cursor at its end.
func (l *exLine) set(text string) {
	l.quiet = true
	l.SetText(text)
	l.CursorColumn = len([]rune(text))
	l.quiet = false
	l.Refresh()
}

// typed lists what completes text, nothing chosen.
func (l *exLine) typed(text string) {
	l.stem, l.chosen, l.moved = text, -1, false
	if !l.completing() {
		return
	}
	if l.w.exPrompt != nil { // a prompt's names (helix s @cue): once it starts with @
		l.comp = promptCompletion(l.w.exPromptNames, text)
		if len(l.comp.choices) == 0 {
			l.w.exPopup().hide()
			return
		}
		l.w.exPopup().show(l)
		return
	}
	l.comp = exCommands().complete(l.ctx, text)
	l.w.exPopup().show(l)
}

// choiceText is the line with choice i written in.
func (l *exLine) choiceText(i int) string {
	return l.stem[:min(l.comp.start, len(l.stem))] + l.comp.choices[i].text
}

// cycle writes the next choice in (step -1: the one before); a choice
// alone is taken, and what it takes listed.
func (l *exLine) cycle(step int) {
	n := len(l.comp.choices)
	if n == 0 {
		return
	}
	if n == 1 || l.moved {
		if l.chosen < 0 {
			l.chosen = 0
		}
		l.take(l.chosen)
		return
	}
	switch {
	case l.chosen < 0 && step < 0:
		l.chosen = n - 1
	case l.chosen < 0:
		l.chosen = 0
	default:
		l.chosen = (l.chosen + step + n) % n
	}
	l.set(l.choiceText(l.chosen))
	l.w.exPopup().show(l)
}

// take writes choice i in and goes on from it: its argument's list.
func (l *exLine) take(i int) {
	text := l.choiceText(i)
	l.set(text)
	l.typed(text)
}

// choose moves the choice by step without writing it in.
func (l *exLine) choose(step int) {
	n := len(l.comp.choices)
	if n == 0 {
		return
	}
	switch {
	case l.chosen < 0 && step < 0:
		l.chosen = n - 1
	case l.chosen < 0:
		l.chosen = 0
	default:
		l.chosen = min(max(l.chosen+step, 0), n-1)
		if step == 1 || step == -1 {
			l.chosen = (l.chosen + n) % n
		}
	}
	l.moved = true
	l.w.exPopup().show(l)
}

// submit runs the line: as typed, or with the choice the arrows chose; a
// command wanting more lists what it takes instead.
func (l *exLine) submit() {
	w := l.w
	if run := w.exPrompt; run != nil { // a prompt's, not a command
		text := l.Text
		if l.moved && l.chosen >= 0 && l.chosen < len(l.comp.choices) {
			text = l.choiceText(l.chosen)
		}
		w.hideEx()
		w.exPrompt, w.exPromptNames = nil, nil
		run(text)
		return
	}
	text := l.Text
	if !l.moved && !l.comp.arg && len(l.comp.choices) > 0 && exUnresolved(text) {
		l.chosen, l.moved = 0, true // :nsp⏎ is the list's first, as it shows
	}
	if l.moved && l.chosen >= 0 && l.chosen < len(l.comp.choices) {
		ch := l.comp.choices[l.chosen]
		text = l.choiceText(l.chosen)
		if ch.more {
			l.take(l.chosen)
			return
		}
	}
	if exWantsMore(text) {
		l.set(strings.TrimRight(text, " ") + " ")
		l.typed(l.Text)
		if len(l.comp.choices) > 0 {
			return
		}
	}
	w.hideEx()
	w.runEx(text)
}

// tapped is a row tapped: completed, and run if it wants nothing more.
func (l *exLine) tapped(i int) {
	if i < 0 || i >= len(l.comp.choices) {
		return
	}
	l.w.fyneWindow.Canvas().Focus(l) // the tap took it away
	l.chosen, l.moved = i, false     // this row's, not the arrows'
	text := l.choiceText(i)
	if l.comp.choices[i].more || exWantsMore(text) {
		l.take(i)
		return
	}
	l.set(text)
	l.submit()
}

// exUnresolved reports whether line's name names no command (none, or
// more than one): Enter takes the list's first instead.
func exUnresolved(line string) bool {
	end, _, ok := exSkipRange(line)
	if !ok {
		return false
	}
	name, nameEnd := exNameOf(line, end)
	if name == "" || nameEnd != len(strings.TrimRight(line, " ")) {
		return false
	}
	c, _ := exCommands().lookup(name)
	return c == nil
}

// exWantsMore reports whether line names a command that does nothing
// without its argument and has none.
func exWantsMore(line string) bool {
	end, _, ok := exSkipRange(line)
	if !ok {
		return false
	}
	name, nameEnd := exNameOf(line, end)
	c, _ := exCommands().lookup(name)
	return c != nil && c.argReq && strings.TrimSpace(line[nameEnd:]) == ""
}

// exPopup is the list above the line, made the first time it is asked
// for: a layer of the window (setContent), not a pop-up, which would
// take the keyboard from the line.
func (w *MainWindow) exPopup() *exPop {
	if w.exPop == nil {
		w.exPop = newExPop(w)
	}
	return w.exPop
}

// exPop is the completion list.
type exPop struct {
	w      *MainWindow
	layer  *fyne.Container // the window's size, the panel placed in it
	panel  *fyne.Container
	rows   *fyne.Container
	hint   *widget.Label
	shown  []*exRow
	narrow bool    // the rows made for a phone
	col    float32 // the names' column: the widest name shown
	bg     *canvas.Rectangle
}

const exRowsShown = 8

func newExPop(w *MainWindow) *exPop {
	p := &exPop{w: w}
	p.rows = container.NewVBox()
	p.hint = widget.NewLabel("")
	p.hint.SizeName = theme.SizeNameCaptionText
	p.hint.Truncation = fyne.TextTruncateEllipsis
	p.bg = canvas.NewRectangle(theme.Color(theme.ColorNameMenuBackground))
	p.bg.StrokeWidth = 1
	p.bg.CornerRadius = theme.InputRadiusSize()
	p.panel = container.NewStack(p.bg, container.NewBorder(nil, p.hint, nil, nil, p.rows))
	p.layer = container.New(&exPopLayout{p: p}, p.panel)
	p.layer.Hide()
	return p
}

// show lists l's choices, the chosen one in sight.
func (p *exPop) show(l *exLine) {
	cs := l.comp.choices
	if len(cs) == 0 || !l.completing() {
		p.hide()
		return
	}
	rows := exRowsShown
	if p.w.narrow {
		rows = 6
	}
	first := 0
	if l.chosen >= rows {
		first = l.chosen - rows + 1
	}
	last := min(first+rows, len(cs))
	if p.narrow != p.w.narrow { // the window changed: rows of the other kind
		p.shown, p.narrow = nil, p.w.narrow
	}
	for len(p.shown) < last-first {
		r := newExRow(p.w.narrow, &p.col)
		p.shown = append(p.shown, r)
	}
	p.col = 0
	for k := first; k < last; k++ {
		p.col = max(p.col, fyne.MeasureText(cs[k].label, theme.TextSize(), fyne.TextStyle{Monospace: true, Bold: true}).Width+2*theme.InnerPadding())
	}
	objs := make([]fyne.CanvasObject, 0, last-first)
	for k := first; k < last; k++ {
		r := p.shown[k-first]
		i := k
		r.set(cs[k], k == l.chosen, func() { l.tapped(i) })
		objs = append(objs, r)
	}
	p.rows.Objects = objs
	hint := fmt.Sprintf("%d of %d", min(max(l.chosen+1, 1), len(cs)), len(cs))
	if l.chosen < 0 {
		hint = strconv.Itoa(len(cs)) + " commands"
		if l.comp.arg {
			hint = strconv.Itoa(len(cs)) + " to choose from"
		}
	}
	if p.w.narrow {
		hint += " · tap one"
	} else {
		hint += " · Tab completes · ↑↓ choose · Enter runs · Esc closes · :help lists them all"
	}
	p.hint.SetText(hint)
	p.bg.FillColor = theme.Color(theme.ColorNameMenuBackground) // the theme now (:theme light)
	p.bg.StrokeColor = theme.Color(theme.ColorNameSeparator)
	p.bg.Refresh()
	p.rows.Refresh()
	p.layer.Show()
	p.layer.Refresh()
}

func (p *exPop) hide() {
	if p.layer.Visible() {
		p.layer.Hide()
	}
}

// exPopLayout puts the list above the ex line, at the window's left:
// the line is the window's last row (relayout's bottom).
type exPopLayout struct{ p *exPop }

func (e *exPopLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.Size{} }

func (e *exPopLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	w, pad := e.p.w, theme.Padding()
	width := min(size.Width-2*pad, 900)
	if w.narrow {
		width = size.Width - 2*pad
	}
	h := e.p.panel.MinSize().Height
	bar := float32(0)
	if w.exBar != nil {
		bar = w.exBar.MinSize().Height
	}
	e.p.panel.Resize(fyne.NewSize(width, h))
	e.p.panel.Move(fyne.NewPos(pad, max(size.Height-bar-h-pad, 0)))
}

// exRow is one row of the list: tapped, it is completed (never focused:
// the line keeps the keyboard).
type exRow struct {
	widget.BaseWidget
	bg         *canvas.Rectangle
	name, desc *widget.Label
	box        fyne.CanvasObject
	tap        func()
}

func newExRow(narrow bool, col *float32) *exRow {
	r := &exRow{bg: canvas.NewRectangle(theme.Color(theme.ColorNameSelection))}
	r.name = widget.NewLabel("")
	r.name.TextStyle = fyne.TextStyle{Monospace: true}
	r.name.Truncation = fyne.TextTruncateEllipsis
	r.desc = widget.NewLabel("")
	r.desc.Truncation = fyne.TextTruncateEllipsis
	r.desc.SizeName = theme.SizeNameCaptionText
	if narrow { // two lines: the name, what it does
		r.box = container.New(layout.NewCustomPaddedVBoxLayout(-2*theme.Padding()), r.name, r.desc)
	} else {
		r.box = container.New(&exRowLayout{col: col}, r.name, r.desc)
	}
	r.ExtendBaseWidget(r)
	return r
}

func (r *exRow) set(c exChoice, chosen bool, tap func()) {
	r.name.SetText(c.label)
	r.desc.SetText(c.desc)
	r.name.Importance = widget.MediumImportance
	if c.off {
		r.name.Importance = widget.LowImportance
	}
	r.name.TextStyle.Bold = chosen
	r.name.Refresh()
	r.tap = tap
	r.bg.FillColor = theme.Color(theme.ColorNameSelection)
	if chosen {
		r.bg.Show()
	} else {
		r.bg.Hide()
	}
	fyne.SetAccessibleLabel(r, c.label+": "+c.desc)
}

func (r *exRow) Tapped(*fyne.PointEvent) {
	if r.tap != nil {
		r.tap()
	}
}

func (r *exRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(r.bg, r.box))
}

// exRowLayout is a desktop row: the name in a column of its own (as
// wide as the widest name shown, up to half the width), what it does
// after it.
type exRowLayout struct{ col *float32 }

func (*exRowLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	h := max(objs[0].MinSize().Height, objs[1].MinSize().Height)
	return fyne.NewSize(200, h)
}

func (r *exRowLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	want := *r.col // the column's, the same in every row
	nameW := min(max(want, size.Width*0.25), size.Width*0.5)
	if d, ok := objs[1].(*widget.Label); ok && d.Text == "" {
		nameW = size.Width // a scene's heading, a file: the row is the name's
	}
	h := objs[0].MinSize().Height
	objs[0].Move(fyne.NewPos(0, 0))
	objs[0].Resize(fyne.NewSize(nameW, h))
	dh := objs[1].MinSize().Height
	objs[1].Move(fyne.NewPos(nameW, (h-dh)/2))
	objs[1].Resize(fyne.NewSize(size.Width-nameW, dh))
}

// showPrompt asks for a line in the command line's place (helix's s and
// S), done gets it.
func (w *MainWindow) showPrompt(label string, done func(string)) { w.showPromptNames(label, nil, done) }

// showPromptNames asks as showPrompt, completing names once the line
// starts with their first character (helix s: @cue, @note …).
func (w *MainWindow) showPromptNames(label string, names []string, done func(string)) {
	w.showEx([2]int{-1, -1})
	w.exLabel.SetText(label)
	w.exPrompt, w.exPromptNames = done, names
	w.exPopup().hide()
}

// promptCompletion are the names text begins, once it starts as they do
// (@@ is no name's: a literal @).
func promptCompletion(names []string, text string) exCompletion {
	var c exCompletion
	if text == "" || len(names) == 0 || !strings.HasPrefix(names[0], text[:1]) || strings.HasPrefix(text, "@@") {
		return c
	}
	c.arg = true
	for _, n := range names {
		if strings.HasPrefix(n, strings.ToLower(text)) && n != text {
			c.choices = append(c.choices, exChoice{text: n, label: n, desc: kindDesc(n)})
		}
	}
	return c
}

// showEx opens the command line, with '<,'> for the lines given.
func (w *MainWindow) showEx(visual [2]int) { w.showExWith(visual, "") }

// showExWith opens the command line with '<,'> for the lines given and
// more typed after it (vim's mas: :'<,'>move ).
func (w *MainWindow) showExWith(visual [2]int, more string) {
	if w.exBar == nil {
		w.exEntry = newExLine(w)
		w.exLabel = widget.NewLabel(":")
		w.exLabel.TextStyle = fyne.TextStyle{Monospace: true}
		w.exBar = container.NewBorder(nil, nil, w.exLabel, nil, w.exEntry)
		w.statusBar.Add(w.exBar)
	}
	w.exLabel.SetText(":")
	w.exPrompt, w.exPromptNames = nil, nil
	w.exVisual = visual
	l := w.exEntry
	l.back = 0
	l.ctx = w.exCtx()
	text := ""
	if visual[0] >= 0 {
		text = "'<,'>"
	}
	text += more
	l.set(text)
	setVisible(w.statusBar, true)
	w.exBar.Show()
	w.fyneWindow.Canvas().Focus(l)
	l.typed(text)
}

// exCtx is what the line completes from, gathered as it opens.
func (w *MainWindow) exCtx() exCtx {
	ctx := exCtx{env: w.cmdEnv(), files: !inBrowser}
	if w.textEditor != nil && w.scriptInMain() {
		ctx.headings = w.outlineItems(w.textEditor.Text())
	}
	ctx.dir = "."
	if w.currentFile != "" {
		ctx.dir = filepath.Dir(w.currentFile)
	} else if wd, err := os.Getwd(); err == nil {
		ctx.dir = wd
	}
	ctx.refs = w.exRefChoices()
	ctx.categories = w.tagCategoryNames()
	ctx.items = func(id string) []string {
		var out []string
		for _, it := range w.exItems(id) {
			out = append(out, it.Label)
		}
		return out
	}
	return ctx
}

// exItems are a submenu command's items that can be chosen.
func (w *MainWindow) exItems(id string) []*fyne.MenuItem {
	c := commands().byID[id]
	if c == nil || c.submenu == nil {
		return nil
	}
	var out []*fyne.MenuItem
	for _, it := range flatItems(c.submenu(w).Items) {
		if !it.IsSeparator && it.Action != nil {
			out = append(out, it)
		}
	}
	return out
}

func (w *MainWindow) hideEx() {
	if w.exBar != nil {
		w.exBar.Hide()
	}
	if w.exPop != nil {
		w.exPop.hide()
	}
	setVisible(w.statusBar, !w.isFullscreen && w.settings.GetBoolean("statusbar-visible"))
	w.focusMain()
}

// runEx runs an ex command: a line command on what the main editor
// holds (ex.go), else the window's or the registry's (runExCommand).
func (w *MainWindow) runEx(cmd string) {
	if strings.TrimSpace(cmd) == "" {
		return
	}
	st := &w.exState
	if h := st.history; len(h) == 0 || h[len(h)-1] != cmd {
		st.history = append(h, cmd)
	}
	e := w.writeEditor()
	text := e.Text()
	lines := strings.Split(text, "\n")
	cur, _ := e.Buffer().Position(e.CursorOffset())
	res, err := st.run(cmd, lines, cur, w.exVisual)
	var other *exNotCommand
	if errors.As(err, &other) && other.structural {
		if err := w.exMoveUnits(other.rest, other.from, other.to, other.given); err != nil {
			w.setStatus(err.Error())
		}
		return
	}
	if errors.As(err, &other) {
		err = w.runExCommand(other.name, strings.TrimSpace(other.rest))
		if err != nil {
			w.setStatus(err.Error())
		}
		return
	}
	if err != nil {
		w.setStatus(err.Error())
		return
	}
	if res.undo {
		w.undo()
		return
	}
	if nw := strings.Join(res.lines, "\n"); nw != text {
		replaceTextIn(e, text, nw) // one undo step, only what changed
	}
	e.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(min(res.line, b.LineCount()-1)), false) })
	if v := w.mainVim(); res.yank != "" && v != nil {
		v.reg, v.regLine = res.yank, true
	}
	if res.msg != "" {
		w.setStatus(res.msg)
	}
	if res.write {
		if res.file != "" {
			p := res.file
			if !filepath.IsAbs(p) && w.currentFile != "" {
				p = filepath.Join(filepath.Dir(w.currentFile), p)
			}
			w.currentFile = p
			if l := languageOf(p); l != "" && bundleKindOf(p) == notBundle {
				w.setLanguage(l) // a bundle's is its own
			}
		}
		if err := w.SaveFile(); err != nil {
			w.setStatus(err.Error())
			return
		}
		w.setStatus("Written: " + filepath.Base(w.currentFile))
	}
	if res.quit {
		if res.keep {
			w.hasChanges = false // :q! closes without asking
		}
		w.flushBundleWrites() // the write just asked for, before the window goes
		w.fyneWindow.Close()
	}
}

// runExCommand runs the window's or the registry's command name with arg.
func (w *MainWindow) runExCommand(name, arg string) error {
	t := exCommands()
	c, amb := t.lookup(name)
	if c == nil {
		if len(amb) > 0 {
			if len(amb) > 6 {
				amb = append(amb[:6], "…")
			}
			return fmt.Errorf("E464: :%s is more than one command: %s (Tab lists them)", name, strings.Join(amb, ", "))
		}
		return fmt.Errorf("E492: not an editor command: %s (:help lists them)", name)
	}
	if c.argReq && arg == "" {
		return fmt.Errorf(":%s wants %s (Tab lists them)", c.name, c.usage)
	}
	if c.run != nil {
		return c.run(w, arg)
	}
	rc := commands().byID[c.id]
	if rc == nil || c.line() {
		return fmt.Errorf("E492: not an editor command: %s", name)
	}
	env := w.cmdEnv()
	if !rc.appliesForRole(env) {
		return fmt.Errorf(":%s is not for this document (%s)", c.name, strings.TrimSuffix(rc.label, "…"))
	}
	away := !rc.applies(env)
	switch {
	case c.arg == argItems:
		it := w.exItem(c.id, arg)
		if it == nil {
			return fmt.Errorf(":%s has no %q (Tab lists them)", c.name, arg)
		}
		w.runByName(rc, it.Action, away)
	case c.arg == argExport && arg != "":
		return w.exExport(arg)
	default:
		w.runByName(rc, nil, away)
	}
	return nil
}

// exItem is the submenu item of id that arg names: its label, or the
// best that arg begins or matches.
func (w *MainWindow) exItem(id, arg string) *fyne.MenuItem {
	items := w.exItems(id)
	var choices []exChoice
	for _, it := range items {
		if strings.EqualFold(it.Label, arg) {
			return it
		}
		choices = append(choices, exChoice{text: it.Label, label: it.Label})
	}
	if m := exFilter(choices, arg); len(m) > 0 {
		for _, it := range items {
			if it.Label == m[0].text {
				return it
			}
		}
	}
	return nil
}

// exOrg is a move or jump of the outline's (org.go) as a command.
func exOrg(key string) func(w *MainWindow, arg string) error {
	return func(w *MainWindow, _ string) error {
		if !w.scriptInMain() {
			return errors.New("The script's tab is not in front: its headings move there")
		}
		w.org(key)
		return nil
	}
}

// exScene goes to the scene or section arg names: its number, its
// heading, or the best heading it matches.
func (w *MainWindow) exScene(arg string) error {
	if !w.scriptInMain() {
		return errors.New("The script's tab is not in front")
	}
	e := w.textEditor
	items := w.outlineItems(e.Text())
	line := -1
	num, _, _ := strings.Cut(arg, " ") // "2" or "2 EXT. FIELD": the second scene
	if n, err := strconv.Atoi(num); err == nil {
		k := 0
		for _, it := range items {
			if it.Scene {
				if k++; k == n {
					line = it.Line
					break
				}
			}
		}
		if line < 0 {
			return fmt.Errorf("There is no scene %d", n)
		}
	} else {
		var choices []exChoice
		for i, it := range items {
			if strings.EqualFold(it.Text, arg) {
				line = it.Line
				break
			}
			choices = append(choices, exChoice{text: it.Text, label: strconv.Itoa(i)})
		}
		if line < 0 {
			m := exFilter(choices, arg)
			if len(m) == 0 {
				return fmt.Errorf("No scene or section like %q", arg)
			}
			i, _ := strconv.Atoi(m[0].label)
			line = items[i].Line
		}
	}
	e.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(line), false) })
	return nil
}

// exElement makes the cursor's line the element arg names.
func (w *MainWindow) exElement(arg string) error {
	var choices []exChoice
	for _, el := range toolbarElements {
		choices = append(choices, exChoice{text: exElementName(el), label: el})
	}
	m := exFilter(choices, arg)
	if len(m) == 0 {
		return fmt.Errorf("No element %q (Tab lists them)", arg)
	}
	c := cmd(elementCommandID(m[0].label))
	if !c.applies(w.cmdEnv()) {
		return errors.New("Elements are a screenplay's, in the script")
	}
	w.runByName(c, nil, false)
	return nil
}

// exHeading makes the line a Markdown heading of level arg, or opens the
// picker.
func (w *MainWindow) exHeading(arg string) error {
	id := "md.headings"
	if arg != "" {
		if n, err := strconv.Atoi(arg); err != nil || n < 0 || n > 6 {
			return errors.New(":heading takes a level, 0 to 6")
		}
		id = "md.heading" + arg
	}
	c := cmd(id)
	if !c.applies(w.cmdEnv()) {
		return errors.New("Headings are a Markdown document's")
	}
	w.runByName(c, nil, false)
	return nil
}

// exEdit opens path (beside the file open, ~ the home), or the Open
// dialog.
func (w *MainWindow) exEdit(path string) error {
	if path == "" || inBrowser {
		w.OpenFile()
		return nil
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[2:])
		}
	}
	if !filepath.IsAbs(path) && w.currentFile != "" {
		path = filepath.Join(filepath.Dir(w.currentFile), path)
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("No file %s", path)
	}
	w.openPath(path)
	return nil
}

// exHelp opens the ex commands' help, at the group of the command arg
// names, or the help page arg is.
func (w *MainWindow) exHelp(arg string) error {
	if arg == "" {
		w.showHelp("ex", "")
		return nil
	}
	if _, ok := help.Get(arg); ok {
		w.showHelp(arg, "")
		return nil
	}
	c, amb := exCommands().lookup(strings.TrimPrefix(arg, ":"))
	if c == nil {
		if len(amb) > 0 {
			w.showHelp("ex", "")
			return nil
		}
		return fmt.Errorf("No help for %q", arg)
	}
	w.showHelp("ex", help.Anchor(exHelpSection(c)))
	w.setStatus(":" + c.name + " — " + c.desc)
	return nil
}

// exExport opens the Export dialog with the format id chosen.
func (w *MainWindow) exExport(id string) error {
	id = strings.ToLower(id)
	if w.markdown {
		for _, f := range markdownExports() {
			if f.id == id {
				ed := newMarkdownExportDialog(w)
				ed.format.SetSelected(f.label)
				ed.Show()
				return nil
			}
		}
	} else {
		for _, f := range exportFormatList() {
			if f.id == id {
				ed := NewExportDialog(w)
				ed.formatSelect.SetSelected(f.label)
				ed.Show()
				return nil
			}
		}
	}
	return fmt.Errorf("No export format %q (Tab lists them)", id)
}

// replaceText changes the script's text from old to nw as one edit: the
// part between what they start and end with in common.
func (w *MainWindow) replaceText(old, nw string) { replaceTextIn(w.textEditor, old, nw) }

// replaceTextIn is replaceText in editor e.
func replaceTextIn(e *editor.ScriptEditor, old, nw string) {
	o, n := []rune(old), []rune(nw)
	start := 0
	for start < len(o) && start < len(n) && o[start] == n[start] {
		start++
	}
	end := 0
	for end < len(o)-start && end < len(n)-start && o[len(o)-1-end] == n[len(n)-1-end] {
		end++
	}
	e.Edit(func(b *buffer.Buffer) {
		b.Group(func() { b.Replace(start, len(o)-end, string(n[start:len(n)-end])) })
	})
}
