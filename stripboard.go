package main

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	lexreport "github.com/LaPingvino/lexington/report"
)

// The stripboard (Production > Stripboard, the Schedule stage's Open
// Stripboard, View > Stripboard, the Schedule stage's Open Stripboard):
// the schedule as a middle tab (middle.go), as the cards' full view is,
// built on them (a strip is dragged as a card is): the script's tab and
// the others beside it, the side panes beside it, the ribbon's Production
// tab on its Schedule stage while it is in front. A strip a scene, in its
// colour by INT/EXT and day or night (white INT. DAY, yellow EXT. DAY,
// blue INT. NIGHT, green EXT. NIGHT: lexreport.StripColour), with its
// number, INT/EXT, location, D/N, length in eighths and cast numbers; the
// days one under the other, each with its head (date, call, notes) and
// its break ("End of Day 1 — 4 6/8 pages"); then the strips on no day.
//
// A strip is dragged to its place (on a desktop), or selected (a tap, or
// Tab) and moved with Move Up / Move Down (Alt+Up / Alt+Down) and Move to
// (a day, or back to the unscheduled): the way on a phone, where a drag
// scrolls the board. Group by Location / by Cast sort the selected
// strip's day (or the unscheduled) once, as a suggestion: nothing is
// sorted by itself. Undo takes the board's last change back.

// stripboardView is the board.
type stripboardView struct {
	w      *MainWindow
	box    *fyne.Container
	list   *fyne.Container
	scroll *container.Scroll
	count  *widget.Label

	actions                          *fyne.Container
	up, down, open, check, toPool    *widget.Button
	addDay, undo, byLocation, byCast *widget.Button
	moveTo                           *widget.Select
	moveChoices                      []int // moveTo's options' days (-1: the unscheduled)
	settingMove                      bool

	selected int // the scene selected (-1: none)
	strips   map[int]*stripWidget
	targets  []stripTarget
	shownKey string
	narrow   bool
	of       docKey // the script it shows (state.go)

	lastDayForm dialog.Dialog // the day's form last shown (tests)

	// found as last shown
	bt   *bdText
	p    schedPlaced
	pool []int
}

// stripTarget is where a strip dropped goes: before strip at of day (-1:
// the unscheduled), or after it on the lower half of a strip.
type stripTarget struct {
	obj    fyne.CanvasObject
	day    int
	at     int
	halves bool
}

func newStripboard(w *MainWindow) *stripboardView {
	sb := &stripboardView{w: w, selected: -1, list: container.NewVBox(), strips: map[int]*stripWidget{}}
	back := widget.NewButtonWithIcon("Back to the Script", theme.DocumentIcon(), func() { w.showMiddle("") })
	back.Importance = widget.LowImportance
	heading := widget.NewLabelWithStyle("Stripboard", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	sb.count = widget.NewLabel("")
	sb.count.Wrapping = fyne.TextWrapWord
	sb.count.SizeName = theme.SizeNameCaptionText
	button := func(label string, icon fyne.Resource, f func()) *widget.Button {
		return widget.NewButtonWithIcon(label, icon, f)
	}
	sb.up = button("Move Up", theme.MoveUpIcon(), func() { sb.shift(-1) })
	sb.down = button("Move Down", theme.MoveDownIcon(), func() { sb.shift(1) })
	sb.moveTo = widget.NewSelect(nil, func(string) { sb.moveToChosen() })
	sb.moveTo.PlaceHolder = "Move to…"
	fyne.SetAccessibleLabel(sb.moveTo, "Move the strip to a day")
	sb.toPool = button("Unschedule", theme.ContentRemoveIcon(), func() { sb.moveSelected(-1, 0) })
	sb.open = button("Open Scene", theme.DocumentIcon(), func() { sb.openScene(sb.selected) })
	sb.check = button("Checked", theme.ConfirmIcon(), sb.checkSelected)
	sb.addDay = button("Add Day", theme.ContentAddIcon(), w.addShootDay)
	sb.addDay.Importance = widget.HighImportance
	sb.byLocation = button("Group by Location", theme.ListIcon(), func() { sb.group("location") })
	sb.byCast = button("Group by Cast", theme.AccountIcon(), func() { sb.group("cast") })
	sb.undo = button("Undo", theme.ContentUndoIcon(), w.undoSchedule)
	sb.actions = container.NewVBox()
	sb.scroll = container.NewVScroll(sb.list)
	top := container.NewVBox(container.NewBorder(nil, nil, heading, back, sb.count), sb.actions)
	sb.box = container.NewBorder(top, nil, nil, nil, sb.scroll)
	return sb
}

// toggleStripboard is the Stripboard command: its middle tab opened, or
// to the front; in front, closed (as a pane's View command).
func (w *MainWindow) toggleStripboard() {
	switch {
	case w.stripsShown():
		w.closeMiddle("strips")
	case w.mid.has("strips"):
		w.showMiddle("strips")
	default:
		w.openMiddle("strips")
	}
}

// stripsShown reports whether the stripboard is the middle tab in front.
func (w *MainWindow) stripsShown() bool { return w.mid.shown == "strips" && w.stripboard != nil }

// stripsBoard is the stripboard, made at the first need.
func (w *MainWindow) stripsBoard() *stripboardView {
	if w.stripboard == nil {
		w.stripboard = newStripboard(w)
	}
	return w.stripboard
}

// showStripboard shows the board (scrolled to day d, -1 the top) with
// scene sc selected (-1 none).
func (w *MainWindow) showStripboard(d, sc int) {
	if w.markdown {
		return
	}
	if !w.stripsShown() {
		w.toggleStripboard()
	}
	sb := w.stripboard
	if sb == nil || !w.stripsShown() {
		return // not for this role
	}
	if sc >= 0 {
		sb.selected = sc
		sb.shownKey = ""
	}
	sb.update()
	if sc >= 0 {
		if s := sb.strips[sc]; s != nil {
			sb.scrollTo(s)
		}
	} else if d >= 0 {
		for _, t := range sb.targets {
			if t.day == d && !t.halves && t.at == 0 {
				sb.scrollTo(t.obj)
				break
			}
		}
	}
}

// addShootDay adds a day at the end.
func (w *MainWindow) addShootDay() {
	var n int
	w.changeSchedule(func(f *scheduleFile, _ *bdText, _ schedPlaced) bool {
		n = f.newDay()
		return true
	})
	w.setStatus(fmt.Sprintf("Day %d added: drag strips onto it, or select one and Move to Day %d", n+1, n+1))
}

// update shows the schedule as it is (rows made again only when what they
// show changed).
func (sb *stripboardView) update() {
	w := sb.w
	w.measureSchedule()
	f, bt, p := w.scheduleFound()
	sb.bt, sb.p = bt, p
	sb.of = w.scriptKey()
	sb.pool = f.pool(bt, p, sb.measuredStrips())
	if sb.selected >= len(bt.scenes) || (sb.selected >= 0 && bt.scenes[sb.selected].omitted && p.dayOf[sb.selected] < 0) {
		sb.selected = -1
	}
	flagged := 0
	var key strings.Builder
	fmt.Fprintf(&key, "%v|%v|%d|%s|", w.narrow, w.scheduleMeasured(), sb.selected, f.Pool)
	for d, day := range f.Days {
		fmt.Fprintf(&key, "D%s|%s|%s|%s|%v|", day.ID, day.Date, day.Call, day.Notes, p.days[d])
		for _, sc := range p.scenesOf(d) {
			if fl := w.flagOf(f, bt, p, sc); fl.changed {
				flagged++
				fmt.Fprintf(&key, "f%d%s|", sc, fl.in)
			}
		}
	}
	fmt.Fprintf(&key, "P%v|L%d|", sb.pool, len(p.lost))
	for i, s := range bt.scenes {
		fmt.Fprintf(&key, "%s %s %v|", s.label, s.heading, s.omitted)
		if st := w.stripOfScene(i); st != nil {
			fmt.Fprintf(&key, "%d %v %v %v|", st.Eighths, st.Exterior, st.Night, st.CastNumbers)
		}
	}
	sb.count.SetText(scheduleCountLine(p.count(bt), flagged, len(p.lost)))
	if k := key.String(); k != sb.shownKey {
		sb.shownKey = k
		sb.rebuild(f, bt, p)
	}
	sb.showActions(f, bt, p)
}

// measuredStrips are the strips by scene as measured (nil: not yet).
func (sb *stripboardView) measuredStrips() []*lexreport.Strip {
	if !sb.w.scheduleMeasured() {
		return nil
	}
	return sb.w.schedule.strips
}

// rebuild makes the board's rows again.
func (sb *stripboardView) rebuild(f *scheduleFile, bt *bdText, p schedPlaced) {
	w := sb.w
	sb.narrow = w.narrow
	focused := -1
	if s, ok := w.fyneWindow.Canvas().Focused().(*stripWidget); ok {
		focused = s.sc
	}
	sb.list.RemoveAll()
	sb.targets = nil
	sb.strips = map[int]*stripWidget{}
	eighths := w.sceneEighths()
	addStrip := func(sc, day, at int) {
		s := newStripWidget(sb, sc, w.stripOfScene(sc), w.flagOf(f, bt, p, sc))
		sb.strips[sc] = s
		sb.list.Add(s)
		sb.targets = append(sb.targets, stripTarget{obj: s, day: day, at: at, halves: true})
	}
	for d, day := range f.Days {
		h := newDayHeader(sb, d, day)
		sb.list.Add(h)
		sb.targets = append(sb.targets, stripTarget{obj: h, day: d, at: 0})
		scenes := p.scenesOf(d)
		if len(scenes) == 0 {
			empty := note(fmt.Sprintf("No scenes yet: drag strips here, or select one and Move to Day %d.", d+1))
			sb.list.Add(empty)
			sb.targets = append(sb.targets, stripTarget{obj: empty, day: d, at: 0})
		}
		for k, sc := range scenes {
			addStrip(sc, d, k)
		}
		total := "measuring…"
		if eighths != nil {
			total = lengthText(dayLength(scenes, eighths))
		}
		br := newDayBreak(fmt.Sprintf("End of Day %d — %s", d+1, total))
		sb.list.Add(br)
		sb.targets = append(sb.targets, stripTarget{obj: br, day: d, at: len(scenes)})
	}
	var poolHead fyne.CanvasObject = caption("Unscheduled · " + plural(len(sb.pool), "scene"))
	if f.Pool != "" { // grouped once: a tap puts it back in script order
		head := newTapLabel("Unscheduled · "+plural(len(sb.pool), "scene")+" · grouped by "+f.Pool+
			" (tap for script order)", sb.poolInScriptOrder)
		head.TextStyle.Bold = true
		poolHead = head
	}
	sb.list.Add(poolHead)
	sb.targets = append(sb.targets, stripTarget{obj: poolHead, day: -1})
	if len(f.Days) == 0 {
		sb.list.Add(note("No shoot days yet: Add Day makes the first. Each day lists its scenes in shooting order."))
	}
	for k, sc := range sb.pool {
		addStrip(sc, -1, k)
	}
	if len(sb.pool) == 0 && len(bt.scenes) > 0 {
		sb.list.Add(note("Every scene is on a day."))
	}
	for _, s := range p.lost {
		s := s
		where := s.Scene
		if where == "" {
			where = strings.ToUpper(s.Heading)
		}
		sb.list.Add(newTapLabel("Lost strip: scene "+where+" (its scene is gone). Tap to remove it.", func() { sb.removeLost(s) }))
	}
	sb.list.Refresh()
	if s := sb.strips[focused]; s != nil {
		w.fyneWindow.Canvas().Focus(s)
	}
}

// showActions shows the selected strip's actions and the board's.
func (sb *stripboardView) showActions(f *scheduleFile, bt *bdText, p schedPlaced) {
	w := sb.w
	sel := sb.selected
	day := -1
	if sel >= 0 {
		day = p.dayOf[sel]
	}
	// the Move to choices: the days and the unscheduled, not where it is
	var opts []string
	sb.moveChoices = nil
	for d := range f.Days {
		if d != day {
			opts = append(opts, fmt.Sprintf("Day %d", d+1))
			sb.moveChoices = append(sb.moveChoices, d)
		}
	}
	if day >= 0 {
		opts = append(opts, "Unscheduled")
		sb.moveChoices = append(sb.moveChoices, -1)
	}
	sb.settingMove = true
	sb.moveTo.Options = opts
	sb.moveTo.ClearSelected()
	sb.settingMove = false
	enable := func(b fyne.Disableable, on bool) {
		if on {
			b.Enable()
		} else {
			b.Disable()
		}
	}
	has := sel >= 0
	enable(sb.up, has && (day > 0 || (day == 0 && p.at[sel] > 0) || (day < 0 && len(f.Days) > 0 && len(sb.pool) > 0 && sb.pool[0] == sel)))
	enable(sb.down, has && day >= 0)
	enable(sb.moveTo, has && len(opts) > 0)
	enable(sb.toPool, has && day >= 0)
	enable(sb.open, has)
	flag := has && w.flagOf(f, bt, p, sel).changed
	enable(sb.check, flag)
	enable(sb.undo, w.schedule.undo != nil)
	scope := "the Unscheduled"
	if day >= 0 {
		scope = fmt.Sprintf("Day %d", day+1)
	}
	sb.byLocation.SetText("Group " + scope + " by Location")
	sb.byCast.SetText("Group " + scope + " by Cast")
	enable(sb.byLocation, w.scheduleMeasured())
	enable(sb.byCast, w.scheduleMeasured())
	strip := []fyne.CanvasObject{sb.up, sb.down, sb.moveTo, sb.toPool, sb.open, sb.check}
	board := []fyne.CanvasObject{sb.addDay, sb.byLocation, sb.byCast, sb.undo}
	var rows []fyne.CanvasObject
	if w.narrow { // a phone: two columns, one under the other
		rows = []fyne.CanvasObject{container.NewGridWithColumns(2, append(strip, sb.addDay, sb.undo)...), sb.byLocation, sb.byCast}
	} else {
		rows = []fyne.CanvasObject{container.NewHScroll(container.NewHBox(strip...)), container.NewHScroll(container.NewHBox(board...))}
	}
	if len(sb.actions.Objects) == 0 || sb.narrow != w.narrow || len(sb.actions.Objects) != len(rows) {
		sb.actions.Objects = rows
		sb.actions.Refresh()
	}
}

// selectScene selects a scene's strip (and gives it the focus).
func (sb *stripboardView) selectScene(sc int, focus bool) {
	if sc != sb.selected {
		old := sb.strips[sb.selected]
		sb.selected = sc
		if old != nil {
			old.showState()
		}
		if s := sb.strips[sc]; s != nil {
			s.showState()
		}
		f, bt, p := sb.w.scheduleFound()
		sb.showActions(f, bt, p)
	}
	if s := sb.strips[sc]; s != nil && focus {
		sb.w.fyneWindow.Canvas().Focus(s)
		sb.scrollTo(s)
	}
}

// order are the scenes in the board's order: the days', then the pool's.
func (sb *stripboardView) order() []int {
	var out []int
	for d := range sb.p.days {
		out = append(out, sb.p.scenesOf(d)...)
	}
	return append(out, sb.pool...)
}

// step selects the strip before (-1) or after (+1) the selected.
func (sb *stripboardView) step(dir int) {
	o := sb.order()
	for i, sc := range o {
		if sc == sb.selected && i+dir >= 0 && i+dir < len(o) {
			sb.selectScene(o[i+dir], true)
			return
		}
	}
}

// shift moves the selected strip up or down.
func (sb *stripboardView) shift(dir int) {
	sc := sb.selected
	if sc < 0 {
		return
	}
	w := sb.w
	moved := w.changeSchedule(func(f *scheduleFile, bt *bdText, p schedPlaced) bool {
		return f.shift(bt, p, sc, dir, w.currentSet(), f.pool(bt, p, sb.measuredStrips()))
	})
	if moved {
		sb.afterMove(sc)
	}
}

// moveSelected moves the selected strip to day d (-1 the unscheduled),
// before its at-th strip.
func (sb *stripboardView) moveSelected(d, at int) {
	sc := sb.selected
	if sc < 0 {
		return
	}
	sb.moveScene(sc, d, at)
}

func (sb *stripboardView) moveScene(sc, d, at int) {
	w := sb.w
	w.changeSchedule(func(f *scheduleFile, bt *bdText, p schedPlaced) bool {
		if p.dayOf[sc] == d && (d < 0 || p.at[sc] == at) {
			return false
		}
		f.move(bt, p, sc, d, at, w.currentSet())
		return true
	})
	sb.afterMove(sc)
}

// afterMove keeps the moved strip selected, in sight, with the focus.
func (sb *stripboardView) afterMove(sc int) {
	sb.shownKey = ""
	sb.update()
	sb.selected = -1
	sb.selectScene(sc, true)
	if d := sb.p.dayOf[sc]; d >= 0 {
		sb.w.setStatus(fmt.Sprintf("Scene %s on Day %d", sb.bt.scenes[sc].label, d+1))
	} else {
		sb.w.setStatus("Scene " + sb.bt.scenes[sc].label + " unscheduled")
	}
}

// moveToChosen is Move to's choice.
func (sb *stripboardView) moveToChosen() {
	if sb.settingMove {
		return
	}
	i := sb.moveTo.SelectedIndex()
	if i < 0 || i >= len(sb.moveChoices) {
		return
	}
	d := sb.moveChoices[i]
	at := 0
	if d >= 0 {
		at = len(sb.p.scenesOf(d)) // at the day's end
	}
	sb.moveSelected(d, at)
}

// group sorts the selected strip's day (or the unscheduled) by location
// or cast, once.
func (sb *stripboardView) group(by string) {
	w := sb.w
	day := -1
	if sb.selected >= 0 {
		day = sb.p.dayOf[sb.selected]
	}
	w.changeSchedule(func(f *scheduleFile, _ *bdText, p schedPlaced) bool {
		if day < 0 {
			f.Pool = by
			return true
		}
		f.sortDay(p, day, by, sb.measuredStrips())
		return true
	})
	what := "the unscheduled"
	if day >= 0 {
		what = fmt.Sprintf("Day %d", day+1)
	}
	w.setStatus(fmt.Sprintf("Grouped %s by %s (Undo takes it back)", what, by))
}

// poolInScriptOrder lists the unscheduled in script order again.
func (sb *stripboardView) poolInScriptOrder() {
	sb.w.changeSchedule(func(f *scheduleFile, _ *bdText, _ schedPlaced) bool {
		f.Pool = ""
		return true
	})
}

// checkSelected takes the selected strip's change flag away.
func (sb *stripboardView) checkSelected() {
	sc := sb.selected
	w := sb.w
	w.changeSchedule(func(f *scheduleFile, bt *bdText, p schedPlaced) bool {
		f.check(bt, p, sc, w.currentSet())
		return true
	})
}

// removeLost asks, then takes a lost strip away.
func (sb *stripboardView) removeLost(s schedStrip) {
	w := sb.w
	dialog.ShowConfirm("Remove the Lost Strip?", "Its scene is no longer in the script. Remove the strip?", func(ok bool) {
		if !ok {
			return
		}
		w.changeSchedule(func(f *scheduleFile, _ *bdText, _ schedPlaced) bool {
			for d := range f.Days {
				for k, x := range f.Days[d].Strips {
					if x == s {
						f.Days[d].Strips = append(f.Days[d].Strips[:k:k], f.Days[d].Strips[k+1:]...)
						return true
					}
				}
			}
			return false
		})
	}, w.fyneWindow)
}

// openScene shows a scene in the editor.
func (sb *stripboardView) openScene(sc int) {
	if sc < 0 || sb.bt == nil || sc >= len(sb.bt.scenes) {
		return
	}
	sb.w.goToLine(sb.bt.scenes[sc].line) // the script's tab to the front, the board's kept
}

// scrollTo scrolls the board so that obj is in sight.
func (sb *stripboardView) scrollTo(obj fyne.CanvasObject) {
	y, h, view := obj.Position().Y, obj.Size().Height, sb.scroll.Size().Height
	off := sb.scroll.Offset.Y
	switch {
	case y < off:
		off = y
	case y+h > off+view:
		off = y + h - view
	default:
		return
	}
	sb.scroll.ScrollToOffset(fyne.NewPos(0, off))
}

// dropAt is where a strip dragged (moving) goes, dropped at pos: its day
// (-1 the unscheduled) and place; ok false when on nothing.
func (sb *stripboardView) dropAt(moving fyne.CanvasObject, pos fyne.Position) (day, at int, ok bool) {
	d := fyne.CurrentApp().Driver()
	for _, t := range sb.targets {
		if t.obj == moving {
			continue
		}
		p, size := d.AbsolutePositionForObject(t.obj), t.obj.Size()
		if pos.Y < p.Y || pos.Y >= p.Y+size.Height {
			continue
		}
		if t.halves && pos.Y >= p.Y+size.Height/2 {
			return t.day, t.at + 1, true
		}
		return t.day, t.at, true
	}
	return 0, 0, false
}

// dragging says where the strip would go.
func (sb *stripboardView) dragging(_ int, self fyne.CanvasObject, pos fyne.Position) {
	day, _, ok := sb.dropAt(self, pos)
	switch {
	case !ok:
		sb.w.setStatus("")
	case day < 0:
		sb.w.setStatus("Unschedule")
	default:
		sb.w.setStatus(fmt.Sprintf("Move to Day %d", day+1))
	}
}

// dropped moves the strip of scene sc where it was dropped.
func (sb *stripboardView) dropped(sc int, self fyne.CanvasObject, pos fyne.Position) {
	day, at, ok := sb.dropAt(self, pos)
	if ok && sb.p.dayOf[sc] == day && day >= 0 && at > sb.p.at[sc] {
		at-- // below itself: its own place taken out first
	}
	if ok {
		sb.selected = sc
		sb.moveScene(sc, day, at)
	}
	sb.shownKey = ""
	sb.update() // back in its place if nothing moved
}

// stripWidget is a scene's strip.
type stripWidget struct {
	widget.BaseWidget
	dragged
	sb      *stripboardView
	sc      int
	frame   *canvas.Rectangle
	body    fyne.CanvasObject
	focused bool
	label   string
}

// paperTheme is the app's theme in its light variant: a strip is paper,
// its ink dark in a dark theme too.
type paperTheme struct{ fyne.Theme }

func (t paperTheme) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return t.Theme.Color(n, theme.VariantLight)
}

// fixedWidth lays one object out at a width of its own.
type fixedWidth float32

func (f fixedWidth) MinSize(objs []fyne.CanvasObject) fyne.Size {
	h := float32(0)
	for _, o := range objs {
		h = max(h, o.MinSize().Height)
	}
	return fyne.NewSize(float32(f), h)
}

func (f fixedWidth) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Resize(size)
		o.Move(fyne.NewPos(0, 0))
	}
}

func cell(width float32, o fyne.CanvasObject) fyne.CanvasObject {
	return container.New(fixedWidth(width), o)
}

func newStripWidget(sb *stripboardView, sc int, st *lexreport.Strip, flag stripFlag) *stripWidget {
	bt := sb.bt
	scene := bt.scenes[sc]
	s := &stripWidget{sb: sb, sc: sc}
	s.dragged = dragged{board: sb, i: sc, self: s}
	hex := "#FFFFFF"
	ie, dn, length, cast, loc, colour := "", "", "…", "", scene.heading, "white"
	if st != nil {
		colour, hex = st.Colour()
		ie, dn, length, loc = st.IE(), st.DN(), st.Eighths.String(), st.Location
		var ns []string
		for _, n := range st.CastNumbers {
			ns = append(ns, strconv.Itoa(n))
		}
		cast = strings.Join(ns, ", ")
		if st.DayNightFrom != "" {
			dn += "*" // from the scene before
		}
	}
	if scene.omitted {
		loc, length = "OMITTED: take it off the day", "0"
	}
	r, g, b, _ := parseHexRGB(hex)
	paper := canvas.NewRectangle(color.NRGBA{R: r, G: g, B: b, A: 0xff})
	paper.StrokeColor = color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}
	paper.StrokeWidth = 1
	s.frame = canvas.NewRectangle(color.Transparent)
	label := func(t string, bold bool) *widget.Label {
		l := widget.NewLabel(t)
		l.TextStyle.Bold = bold
		l.Truncation = fyne.TextTruncateEllipsis
		return l
	}
	num := label(scene.label, true)
	var content fyne.CanvasObject
	if sb.w.narrow { // a phone: two lines
		first := label(scene.label+" · "+ie+" · "+loc, true)
		second := label(strings.Join(nonEmpty(dn, length+" pg", castText(cast)), " · "), false)
		content = container.NewVBox(first, second)
	} else {
		left := container.NewHBox(cell(48, num), cell(40, label(ie, false)))
		right := container.NewHBox(cell(32, label(dn, false)), cell(56, label(length, false)), cell(110, label(castText(cast), false)))
		content = container.NewBorder(nil, nil, left, right, label(loc, true))
	}
	if flag.changed {
		f := widget.NewLabelWithStyle("⚠ "+flag.text(), fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
		f.Importance = widget.DangerImportance
		f.Truncation = fyne.TextTruncateEllipsis
		content = container.NewVBox(content, f)
	}
	s.body = container.NewStack(paper,
		container.NewThemeOverride(container.New(&tightPad{}, content), paperTheme{fyne.CurrentApp().Settings().Theme()}),
		s.frame)
	s.label = fmt.Sprintf("Scene %s, %s %s %s, %s, %s pages, cast %s, %s strip", scene.label, ie, loc, dn, scene.heading, length,
		cmpOr(cast, "none"), colour)
	if flag.changed {
		s.label += ", " + flag.text()
	}
	s.ExtendBaseWidget(s)
	fyne.SetAccessibleLabel(s, s.label)
	s.showState()
	return s
}

// tightPad pads a strip's content less than the theme's padding: a board
// shows many strips.
type tightPad struct{}

func (tightPad) MinSize(objs []fyne.CanvasObject) fyne.Size {
	s := fyne.NewSize(0, 0)
	for _, o := range objs {
		s = s.Max(o.MinSize())
	}
	return s.AddWidthHeight(8, 0)
}

func (tightPad) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Resize(size.SubtractWidthHeight(8, 0))
		o.Move(fyne.NewPos(4, 0))
	}
}

func castText(c string) string {
	if c == "" {
		return ""
	}
	return "Cast " + c
}

func nonEmpty(xs ...string) []string {
	var out []string
	for _, x := range xs {
		if x != "" {
			out = append(out, x)
		}
	}
	return out
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// showState draws the strip selected (an outline) and focused (a thicker
// one).
func (s *stripWidget) showState() {
	sel := s.sb.selected == s.sc
	s.frame.StrokeColor = theme.Color(theme.ColorNamePrimary)
	switch {
	case s.focused:
		s.frame.StrokeWidth = 4
	case sel:
		s.frame.StrokeWidth = 2
	default:
		s.frame.StrokeWidth = 0
	}
	s.frame.Refresh()
}

func (s *stripWidget) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(s.body) }

// Tapped selects the strip.
func (s *stripWidget) Tapped(*fyne.PointEvent) { s.sb.selectScene(s.sc, true) }

// DoubleTapped opens its scene.
func (s *stripWidget) DoubleTapped(*fyne.PointEvent) { s.sb.openScene(s.sc) }

// Dragged moves the strip on a desktop; on a phone a drag scrolls the
// board (Move Up, Move Down and Move to move strips there).
func (s *stripWidget) Dragged(e *fyne.DragEvent) {
	if s.sb.w.narrow {
		s.sb.scroll.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.Delta{DY: e.Dragged.DY}})
		return
	}
	s.dragged.Dragged(e)
}

func (s *stripWidget) DragEnd() {
	if s.sb.w.narrow {
		return
	}
	s.dragged.DragEnd()
}

func (s *stripWidget) FocusGained() {
	s.focused = true
	s.sb.selectScene(s.sc, false)
	s.showState()
}

func (s *stripWidget) FocusLost() {
	s.focused = false
	s.showState()
}

func (s *stripWidget) TypedRune(rune) {}

// TypedKey: Up and Down select the strip before or after, Enter opens the
// scene.
func (s *stripWidget) TypedKey(e *fyne.KeyEvent) {
	switch e.Name {
	case fyne.KeyUp:
		s.sb.step(-1)
	case fyne.KeyDown:
		s.sb.step(1)
	case fyne.KeyReturn, fyne.KeyEnter:
		s.sb.openScene(s.sc)
	}
}

// TypedShortcut: Alt+Up and Alt+Down move the strip (where no menu has
// the keys: on a desktop Edit > Move Line Up/Down has them, and gives
// them to a strip with the focus, orStrip).
func (s *stripWidget) TypedShortcut(sc fyne.Shortcut) {
	if c, ok := sc.(*desktop.CustomShortcut); ok && c.Modifier == fyne.KeyModifierAlt {
		switch c.KeyName {
		case fyne.KeyUp:
			s.moveBy(-1)
		case fyne.KeyDown:
			s.moveBy(1)
		}
	}
}

// moveBy moves the strip up (-1) or down (+1).
func (s *stripWidget) moveBy(dir int) {
	s.sb.selectScene(s.sc, false)
	s.sb.shift(dir)
}

// orStrip is a menu action whose keys move a strip that has the focus
// (Alt+Up / Alt+Down), else f.
func (w *MainWindow) orStrip(dir int, f func()) func() {
	return func() {
		if s, ok := w.fyneWindow.Canvas().Focused().(*stripWidget); ok {
			s.moveBy(dir)
			return
		}
		f()
	}
}

// Cursor is a pointer: the strip can be tapped and dragged.
func (s *stripWidget) Cursor() desktop.Cursor { return desktop.PointerCursor }

// newDayHeader is a day's head: its number, date, call and notes, and a
// button to edit them.
func newDayHeader(sb *stripboardView, d int, day schedDay) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(dayLabel(d, day), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	title.Truncation = fyne.TextTruncateEllipsis
	edit := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() { sb.editDay(d) })
	edit.Importance = widget.LowImportance
	fyne.SetAccessibleLabel(edit, fmt.Sprintf("Edit Day %d (date, call, notes)", d+1))
	rows := []fyne.CanvasObject{widget.NewSeparator(), container.NewBorder(nil, nil, nil, edit, title)}
	if day.Notes != "" {
		n := note(day.Notes)
		rows = append(rows, n)
	}
	return container.NewVBox(rows...)
}

// newDayBreak is a day's break: a black bar with the day's total, as the
// board's.
func newDayBreak(text string) fyne.CanvasObject {
	bar := canvas.NewRectangle(color.Black)
	t := canvas.NewText(text, color.White)
	t.TextStyle.Bold = true
	t.TextSize = theme.TextSize() * 0.9
	return container.NewStack(bar, container.NewPadded(t))
}

// editDay asks for a day's date, call and notes (or to remove it).
func (sb *stripboardView) editDay(d int) {
	w := sb.w
	f := w.scheduleNow()
	if d >= len(f.Days) {
		return
	}
	day := f.Days[d]
	date := widget.NewEntry()
	date.SetPlaceHolder("2026-12-01")
	date.SetText(day.Date)
	date.Validator = func(s string) error {
		if s = strings.TrimSpace(s); s == "" {
			return nil
		}
		if _, err := time.Parse("2006-01-02", s); err != nil {
			return fmt.Errorf("a date as 2026-12-01")
		}
		return nil
	}
	call := widget.NewEntry()
	call.SetPlaceHolder("07:00")
	call.SetText(day.Call)
	call.Validator = func(s string) error {
		if s = strings.TrimSpace(s); s == "" {
			return nil
		}
		if _, _, ok := lexreport.ParseCall(s); !ok {
			return fmt.Errorf("a time as 07:00")
		}
		return nil
	}
	notes := widget.NewMultiLineEntry()
	notes.Wrapping = fyne.TextWrapWord
	notes.SetPlaceHolder("Where the unit goes: address, parking, base camp")
	notes.SetText(day.Notes)
	remove := widget.NewCheck("Remove this day (its scenes go back to the unscheduled)", nil)
	form := dialog.NewForm(fmt.Sprintf("Day %d", d+1), "Done", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Date", date),
		widget.NewFormItem("Call", call),
		widget.NewFormItem("Notes", notes),
		widget.NewFormItem("", remove),
	}, func(ok bool) {
		if !ok {
			return
		}
		sb.setDay(d, strings.TrimSpace(date.Text), strings.TrimSpace(call.Text), strings.TrimSpace(notes.Text), remove.Checked)
	}, w.fyneWindow)
	form.Resize(fitDialog(w.fyneWindow, 460, 380))
	form.Show()
	sb.lastDayForm = form
}

// setDay sets a day's date, call and notes, or removes it.
func (sb *stripboardView) setDay(d int, date, call, notes string, remove bool) {
	sb.w.changeSchedule(func(f *scheduleFile, _ *bdText, _ schedPlaced) bool {
		if d >= len(f.Days) {
			return false
		}
		if remove {
			f.removeDay(d)
			return true
		}
		if h, m, ok := lexreport.ParseCall(call); ok {
			call = fmt.Sprintf("%02d:%02d", h, m)
		}
		f.Days[d].Date, f.Days[d].Call, f.Days[d].Notes = date, call, notes
		return true
	})
}
