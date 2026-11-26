package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"slices"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The Table Read view (tableread.go has what it reads): on a desktop's
// wide window a window of its own, to put on a second screen (a TV, a
// projector); in the browser and on a phone a middle tab (middle.go),
// which full screen fills the page with, so Chrome's own Cast tab casts it.
// Large type on dark; one scene at a time, Next and Previous through the
// scenes picked; the panel picks them.

// trTints are the actors' tints, light enough to read on dark.
var trTints = []color.NRGBA{
	{0xff, 0xa9, 0x4d, 0xff}, {0x74, 0xc0, 0xfc, 0xff}, {0x8c, 0xe9, 0x9a, 0xff}, {0xe5, 0x99, 0xf7, 0xff},
	{0xff, 0xd4, 0x3b, 0xff}, {0x66, 0xd9, 0xe8, 0xff}, {0xff, 0x87, 0x87, 0xff}, {0xb1, 0x97, 0xfc, 0xff},
}

// The table read's own colour names.
const (
	trNarratorColor fyne.ThemeColorName = "tableread-narrator"
	trTintColor     fyne.ThemeColorName = "tableread-tint-" // and its number
)

// trNarratorInk is the narrator's colour: a quiet grey-blue.
var trNarratorInk = color.NRGBA{0xa5, 0xb4, 0xc8, 0xff}

// trTheme is the app's theme in its dark variant, its text scale times
// as large, with the tints by name.
type trTheme struct {
	fyne.Theme
	scale float32
}

func (t trTheme) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	if _, ok := t.Theme.(*accoladeTheme); ok { // the theme's own dark (or System's)
		t.Theme = inVariant(t.Theme, theme.VariantDark)
	}
	if n == trNarratorColor {
		return trNarratorInk
	}
	if s, ok := strings.CutPrefix(string(n), string(trTintColor)); ok {
		var i int
		fmt.Sscan(s, &i)
		return trTints[i%len(trTints)]
	}
	return t.Theme.Color(n, theme.VariantDark)
}

func (t trTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNameText, theme.SizeNameHeadingText, theme.SizeNameSubHeadingText, theme.SizeNameCaptionText:
		return t.Theme.Size(n) * t.scale
	}
	return t.Theme.Size(n)
}

func tintName(i int) fyne.ThemeColorName { return trTintColor + fyne.ThemeColorName(fmt.Sprint(i)) }

// tableReadView is the view: what it read, the pick, the scene shown.
type tableReadView struct {
	w      *MainWindow
	text   string // the script as read last
	script *trScript
	tags   *trTags
	photos map[string]string // each speaker's cast photo
	pick   trPick
	order  []int // the scenes picked
	scene  int   // the scene shown (-1: none)
	// the line shown as the one being read (-1: none): a follower's
	// leader's line, or one tapped
	line                   int
	narrator, boards, tint bool
	beside                 bool // the frames beside the lines (else between)
	panelShown             bool
	// follow: the replica followed (tableread_follow.go), its name
	follow, followName string
	followLine         int      // their line as last followed
	followIDs          []string // the panel's Follow list's replicas

	box fyne.CanvasObject // the whole view, dark
	// the theme overrides, refreshed when what is in them is made anew
	// (they give it their theme then): the whole, the reading area
	dark, reading *container.ThemeOverride
	area          *trArea // the reading area (the keys, the focus)
	rows          *fyne.Container
	scroll        *container.Scroll
	title         *widget.Label
	estimate      *widget.Label
	panel         *fyne.Container
	from, to      *widget.Select
	tagSel        *widget.Select
	castRow       *fyne.Container
	follower      *widget.Select // who to follow (tableread_follow.go)
	checks        struct{ narrator, boards, beside, tint *widget.Check }
	marked        map[int]*canvas.Rectangle // each line's background, by script line
	// win is its window on a desktop (nil: a middle tab)
	win      fyne.Window
	building bool // the panel being filled (its widgets' callbacks wait)
}

// tableReadView is the window's view, made at the first need.
func (w *MainWindow) tableReadView() *tableReadView {
	if w.tableRead == nil {
		w.tableRead = newTableReadView(w)
	}
	return w.tableRead
}

// tableReadOpen reports whether the Table Read shows: its window, or its
// middle tab.
func (w *MainWindow) tableReadOpen() bool {
	return w.tableRead != nil && (w.tableRead.win != nil || w.mid.has("tableread"))
}

// tableReadInFront reports whether the Table Read is the middle tab in front.
func (w *MainWindow) tableReadInFront() bool { return w.mid.shown == "tableread" }

// trInWindow: the Table Read opens in a window of its own (a desktop's
// wide window), not as a middle tab.
func (w *MainWindow) trInWindow() bool { return !inBrowser && !w.narrow }

// openTableRead shows the Table Read: its window (brought to the front),
// or its middle tab.
func (w *MainWindow) openTableRead() {
	if w.markdown {
		w.setStatus("The Table Read reads a screenplay")
		return
	}
	fresh := !w.tableReadOpen() // it opens where the cursor is
	v := w.tableReadView()
	v.update()
	if fresh {
		v.startAt(w.scriptCursorLineNow())
	}
	if w.trInWindow() {
		v.openWindow()
		return
	}
	w.openMiddle("tableread")
}

// closeTableRead closes it, wherever it is.
func (w *MainWindow) closeTableRead() {
	v := w.tableRead
	if v == nil {
		return
	}
	if v.win != nil {
		win := v.win
		v.win = nil
		win.Close()
	}
	if w.mid.has("tableread") {
		w.closeMiddle("tableread")
	}
	w.ribbonRefresh()
}

// scriptCursorLineNow is the script editor's cursor's line.
func (w *MainWindow) scriptCursorLineNow() int {
	if w.textEditor == nil {
		return 0
	}
	line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
	return line
}

func newTableReadView(w *MainWindow) *tableReadView {
	v := &tableReadView{w: w, scene: -1, line: -1, narrator: true, tint: true, pick: trPick{To: -1}}
	scale := float32(1.9)
	if w.narrow {
		scale = 1.35
	}
	base := fyne.CurrentApp().Settings().Theme()
	v.rows = container.NewVBox()
	v.scroll = container.NewVScroll(v.rows)
	v.reading = container.NewThemeOverride(container.NewPadded(v.scroll), trTheme{base, scale})
	v.area = newTRArea(v, v.reading)

	prev := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), v.previous)
	next := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), v.next)
	prev.Importance, next.Importance = widget.LowImportance, widget.LowImportance
	v.title = widget.NewLabel("")
	v.title.TextStyle.Bold = true
	v.title.Truncation = fyne.TextTruncateEllipsis
	v.estimate = widget.NewLabel("")
	pickBtn := widget.NewButtonWithIcon("Pick", theme.MenuIcon(), v.togglePanel)
	full := widget.NewButtonWithIcon("", theme.ViewFullScreenIcon(), v.fullScreen)
	closeBtn := widget.NewButtonWithIcon("", theme.CancelIcon(), w.closeTableRead)
	for _, b := range []*widget.Button{pickBtn, full, closeBtn} {
		b.Importance = widget.LowImportance
	}
	top := container.NewVBox(
		container.NewBorder(nil, nil, prev, next, v.title),
		container.NewBorder(nil, nil, nil, container.NewHBox(pickBtn, full, closeBtn), v.estimate))

	v.from = widget.NewSelect(nil, func(string) { v.picked() })
	v.to = widget.NewSelect(nil, func(string) { v.picked() })
	v.tagSel = widget.NewSelect(nil, func(string) { v.picked() })
	v.follower = widget.NewSelect(nil, func(s string) { v.followPicked(s) })
	v.castRow = container.NewHBox()
	v.checks.narrator = widget.NewCheck("Narrator", func(on bool) { v.setNarrator(on) })
	v.checks.boards = widget.NewCheck("Storyboard", func(on bool) { v.setBoards(on) })
	v.checks.beside = widget.NewCheck("Beside", func(on bool) { v.setBeside(on) })
	v.checks.tint = widget.NewCheck("Actor tints", func(on bool) { v.setTint(on) })
	v.checks.narrator.SetChecked(v.narrator)
	v.checks.tint.SetChecked(v.tint)
	// a row each on a phone, two to a row on a wider window
	labelled := func(label string, o fyne.CanvasObject) fyne.CanvasObject {
		l := widget.NewLabel(label)
		return container.NewBorder(nil, nil, container.New(fixedWidth(64), l), nil, o)
	}
	cols := 2
	if w.narrow {
		cols = 1
	}
	checks := container.NewGridWithColumns(4, v.checks.narrator, v.checks.boards, v.checks.beside, v.checks.tint)
	if w.narrow {
		checks = container.NewGridWithColumns(2, v.checks.narrator, v.checks.tint, v.checks.boards, v.checks.beside)
	}
	v.panel = container.NewVBox(
		container.NewGridWithColumns(cols, labelled("From", v.from), labelled("To", v.to)),
		container.NewGridWithColumns(cols, labelled("Tag", v.tagSel), labelled("Follow", v.follower)),
		container.NewHScroll(v.castRow), checks, widget.NewSeparator())
	v.panel.Hide()

	bg := canvas.NewRectangle(trTheme{base, 1}.Color(theme.ColorNameBackground, theme.VariantDark))
	v.dark = container.NewThemeOverride(container.NewStack(bg,
		container.NewBorder(container.NewVBox(top, v.panel), nil, nil, nil, v.area)), trTheme{base, 1})
	v.box = v.dark
	return v
}

// openWindow shows the view in a window of its own.
func (v *tableReadView) openWindow() {
	if v.win != nil {
		v.win.Show()
		v.win.RequestFocus()
		return
	}
	win := fyne.CurrentApp().NewWindow("Table Read — " + v.w.pdfTitle())
	win.SetContent(v.box)
	win.Resize(fyne.NewSize(1100, 760))
	win.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) { v.key(e) })
	win.SetOnClosed(func() {
		if v.win == win {
			v.win = nil
			v.w.ribbonRefresh()
		}
	})
	v.win = win
	win.Show()
	v.show()
	win.Canvas().Focus(v.area)
	v.w.ribbonRefresh()
}

// pdfTitle is the script's name, for the window's title.
func (w *MainWindow) pdfTitle() string { return strings.TrimSuffix(w.pdfName(), ".pdf") }

// key is a key typed on the view: the arrows, Page Up/Down and Space go
// through the scenes, Escape closes the window, F11 makes it full screen.
func (v *tableReadView) key(e *fyne.KeyEvent) {
	switch e.Name {
	case fyne.KeyRight, fyne.KeyPageDown, fyne.KeySpace, fyne.KeyN:
		v.next()
	case fyne.KeyLeft, fyne.KeyPageUp, fyne.KeyBackspace, fyne.KeyP:
		v.previous()
	case fyne.KeyDown:
		v.scroll.Offset.Y += 60
		v.scroll.Refresh()
	case fyne.KeyUp:
		v.scroll.Offset.Y = max(0, v.scroll.Offset.Y-60)
		v.scroll.Refresh()
	case fyne.KeyF11, fyne.KeyF:
		v.fullScreen()
	case fyne.KeyEscape:
		if v.win != nil {
			if v.win.FullScreen() {
				v.win.SetFullScreen(false)
				return
			}
			v.w.closeTableRead()
		}
	}
}

// fullScreen: the window full screen (a TV's), or the page's.
func (v *tableReadView) fullScreen() {
	if v.win != nil {
		v.win.SetFullScreen(!v.win.FullScreen())
		return
	}
	v.w.toggleFullscreen()
}

// update reads the script again if it changed, and shows the scene.
func (v *tableReadView) update() {
	w := v.w
	text := w.textEditor.Text()
	if v.script != nil && sameString(text, v.text) {
		return
	}
	v.text = text
	v.script = readTableWith(text, w.sceneStarts(), w.dialect(), w.storyboarder(), w.frameFileSecs)
	v.tags = w.tableReadTags(v.script)
	v.photos = map[string]string{}
	list := charListOf(text)
	for _, k := range v.script.Cast {
		if e := list.find(k); e >= 0 {
			if src := strings.TrimSpace(list.entries[e].Fields[fieldPhoto]); src != "" {
				v.photos[k] = src
			}
		}
	}
	if v.pick.To >= len(v.script.Scenes) {
		v.pick.To = -1
	}
	v.order = v.script.picked(v.pick, v.tags)
	if v.scene >= len(v.script.Scenes) {
		v.scene = len(v.script.Scenes) - 1
	}
	if v.scene < 0 && len(v.order) > 0 {
		v.scene = v.order[0]
	}
	v.fillPanel()
	v.show()
}

// tableReadTags are the tags to pick by: the breakdown's elements, the
// schedule's days, the scenes' colours.
func (w *MainWindow) tableReadTags(s *trScript) *trTags {
	tags := &trTags{}
	byLine := map[int]int{}
	for i, sc := range s.Scenes {
		if sc.Heading != "" {
			byLine[sc.Line] = i
		}
	}
	if !w.markdown && w.textEditor != nil {
		f, bt, ps := w.breakdownNowFound()
		sceneOf := func(bdScene int) (int, bool) {
			if bdScene < 0 || bdScene >= len(bt.scenes) {
				return 0, false
			}
			i, ok := byLine[bt.scenes[bdScene].line]
			return i, ok
		}
		if sf := w.scheduleNow(); len(sf.Days) > 0 {
			p := sf.place(bt)
			for d := range p.days {
				for _, sc := range p.scenesOf(d) {
					if i, ok := sceneOf(sc); ok {
						tags.add(fmt.Sprintf("day:%d", d), fmt.Sprintf("Shoot day %d", d+1), i)
					}
				}
			}
		}
		_ = f
		var els []placed
		for _, p := range ps {
			if !p.lost {
				els = append(els, p)
			}
		}
		slices.SortStableFunc(els, func(a, b placed) int {
			return strings.Compare(a.element.Category+"\x00"+a.element.Name, b.element.Category+"\x00"+b.element.Name)
		})
		for _, p := range els {
			if i, ok := sceneOf(p.scene); ok {
				tags.add("el:"+p.element.ID, p.element.Category+": "+p.element.Name, i)
			}
		}
	}
	for i, sc := range s.Scenes {
		if sc.Color != "" {
			tags.add("color:"+strings.ToLower(sc.Color), "Colour: "+colorName(sc.Color), i)
		}
	}
	return tags
}

// fillPanel fills the panel's lists for the script as read.
func (v *tableReadView) fillPanel() {
	v.building = true
	defer func() { v.building = false }()
	s := v.script
	var scenes []string
	for _, sc := range s.Scenes {
		scenes = append(scenes, trSceneLabel(sc))
	}
	v.from.Options, v.to.Options = scenes, scenes
	if len(scenes) > 0 {
		v.from.SetSelectedIndex(min(max(0, v.pick.From), len(scenes)-1))
		to := v.pick.To
		if to < 0 || to >= len(scenes) {
			to = len(scenes) - 1
		}
		v.to.SetSelectedIndex(to)
	}
	v.tagSel.Options = []string{"Every scene"}
	sel := 0
	for i, k := range v.tags.keys {
		v.tagSel.Options = append(v.tagSel.Options, v.tags.labels[k])
		if k == v.pick.Tag {
			sel = i + 1
		}
	}
	v.tagSel.SetSelectedIndex(sel)
	v.fillCast()
	v.fillFollow()
}

// trSceneLabel is a scene as the panel names it: "3 INT. KITCHEN - NIGHT".
func trSceneLabel(sc trScene) string {
	if sc.Heading == "" {
		return "Opening"
	}
	return sc.Number + "  " + sc.Heading
}

// fillCast makes the panel's cast: Everyone, then each one who speaks,
// with their photo and their tint.
func (v *tableReadView) fillCast() {
	var objs []fyne.CanvasObject
	every := widget.NewButton("Everyone", func() { v.pickCharacter("") })
	if v.pick.Character == "" {
		every.Importance = widget.HighImportance
	}
	objs = append(objs, every)
	for _, k := range v.script.Cast {
		b := widget.NewButton(v.script.Names[k], func() { v.pickCharacter(k) })
		if img := v.photo(k, 48); img != nil {
			b.SetIcon(imageResource("cast-"+k, img))
		}
		if v.pick.Character == k {
			b.Importance = widget.HighImportance
		}
		bar := canvas.NewRectangle(trTints[v.script.tintOf(k)])
		bar.SetMinSize(fyne.NewSize(0, 3))
		objs = append(objs, container.NewBorder(nil, bar, nil, nil, b))
	}
	v.castRow.Objects = objs
	v.dark.Refresh() // the buttons take its theme
}

// photo is speaker's cast photo, size pixels square (nil: none).
func (v *tableReadView) photo(speaker string, size int) image.Image {
	src := v.photos[speaker]
	if src == "" {
		return nil
	}
	return v.w.pictures().Thumbnail(src, size, size)
}

// imageResource is an image as a resource (a button's icon).
func imageResource(name string, img image.Image) fyne.Resource {
	var b bytes.Buffer
	if png.Encode(&b, img) != nil {
		return nil
	}
	return fyne.NewStaticResource(name+".png", b.Bytes())
}

// picked takes the panel's choice of scenes and tag.
func (v *tableReadView) picked() {
	if v.building || v.script == nil {
		return
	}
	v.pick.From = max(0, v.from.SelectedIndex())
	v.pick.To = v.to.SelectedIndex()
	if v.pick.To < v.pick.From {
		v.pick.To = v.pick.From
		v.building = true
		v.to.SetSelectedIndex(v.pick.To)
		v.building = false
	}
	v.pick.Tag = ""
	if i := v.tagSel.SelectedIndex(); i > 0 && i-1 < len(v.tags.keys) {
		v.pick.Tag = v.tags.keys[i-1]
	}
	v.repick()
}

// pickCharacter shows only speaker's scenes ("": everyone's).
func (v *tableReadView) pickCharacter(speaker string) {
	v.pick.Character = speaker
	v.fillCast()
	v.repick()
}

// setPick picks p (a test's, the palette's way).
func (v *tableReadView) setPick(p trPick) {
	v.update()
	v.pick = p
	v.fillPanel()
	v.repick()
}

// repick follows a new pick: the scene shown stays if it is picked, else
// the first picked.
func (v *tableReadView) repick() {
	v.order = v.script.picked(v.pick, v.tags)
	if !slices.Contains(v.order, v.scene) {
		v.scene = -1
		if len(v.order) > 0 {
			v.scene = v.order[0]
		}
		v.line = -1
	}
	v.show()
}

// startAt shows the scene line is in (the cursor's, when it opens).
func (v *tableReadView) startAt(line int) {
	if v.script == nil {
		return
	}
	if sc := v.script.sceneAtLine(line); sc >= 0 {
		v.scene = sc
	}
	v.show()
}

// next and previous go to the next and the previous scene picked.
func (v *tableReadView) next() { v.step(1) }

func (v *tableReadView) previous() { v.step(-1) }

func (v *tableReadView) step(by int) {
	if v.script == nil || len(v.order) == 0 {
		return
	}
	at := -1
	for i, sc := range v.order {
		if by > 0 && sc > v.scene || by < 0 && sc < v.scene {
			if by > 0 {
				at = i
				break
			}
			at = i
		}
	}
	if at < 0 {
		if by > 0 {
			v.w.setStatus("The last scene picked")
		} else {
			v.w.setStatus("The first scene picked")
		}
		return
	}
	v.scene, v.line = v.order[at], -1
	v.show()
	v.scroll.ScrollToTop()
	v.moved()
}

// moved: the scene or the line read changed (the others following this
// one see it: tableread_follow.go).
func (v *tableReadView) moved() { v.w.sayHereNow() }

// lineNow is the script line the view is at: the line read, else the
// scene's heading (-1: none).
func (v *tableReadView) lineNow() int {
	if v.line >= 0 {
		return v.line
	}
	if v.script != nil && v.scene >= 0 && v.scene < len(v.script.Scenes) {
		return v.script.Scenes[v.scene].Line
	}
	return -1
}

// setNarrator, setBoards, setTint are the panel's checks (and the tab's).
func (v *tableReadView) setNarrator(on bool) {
	if v.narrator != on {
		v.narrator = on
		v.checks.narrator.SetChecked(on)
		v.show()
	}
}

func (v *tableReadView) setBoards(on bool) {
	if v.boards != on {
		v.boards = on
		v.checks.boards.SetChecked(on)
		v.show()
	}
}

// setBeside: the frames beside the lines they follow, or between them
// (beside shows them: the storyboard on).
func (v *tableReadView) setBeside(on bool) {
	if v.beside != on || on && !v.boards {
		v.beside = on
		v.checks.beside.SetChecked(on)
		if on && !v.boards {
			v.setBoards(true) // shows
			return
		}
		v.show()
	}
}

func (v *tableReadView) setTint(on bool) {
	if v.tint != on {
		v.tint = on
		v.checks.tint.SetChecked(on)
		v.show()
	}
}

func (v *tableReadView) togglePanel() {
	v.panelShown = !v.panelShown
	setVisible(v.panel, v.panelShown)
	v.dark.Refresh() // laid out again, with the panel or without
	v.w.ribbonRefresh()
}

// position is where the view is, as its title says: "Scene 3 · 2 of 5 picked".
func (v *tableReadView) position() string {
	if v.script == nil || v.scene < 0 {
		return "Nothing picked"
	}
	sc := v.script.Scenes[v.scene]
	where := "Opening"
	if sc.Heading != "" {
		where = "Scene " + sc.Number + " · " + sc.Heading
	}
	if i := slices.Index(v.order, v.scene); i >= 0 {
		return fmt.Sprintf("%s  (%d of %d)", where, i+1, len(v.order))
	}
	return where + "  (not in the pick)"
}

// estimateText is the scene's and the pick's length.
func (v *tableReadView) estimateText() string {
	if v.script == nil || v.scene < 0 {
		return ""
	}
	sc := v.script.Scenes[v.scene]
	secs, byBoards := sc.secs()
	s := "This scene " + trEstimate(sc.Rows, secs)
	if byBoards {
		s += " by its boards"
	}
	if len(v.order) > 1 {
		s += " · picked " + plural(len(v.order), "scene") + ", " + trEstimate(v.script.rowsOf(v.order))
	}
	if v.follow != "" {
		s += " · following " + v.followName
	}
	return s
}

// show shows the scene: its lines, the title, the estimate.
func (v *tableReadView) show() {
	v.title.SetText(v.position())
	v.estimate.SetText(v.estimateText())
	v.marked = map[int]*canvas.Rectangle{}
	var objs []fyne.CanvasObject
	if v.script != nil && v.scene >= 0 && v.scene < len(v.script.Scenes) {
		lines := v.script.Scenes[v.scene].Lines
		side := v.besideFrames(lines)
		for i, l := range lines {
			if l.Kind == trFrame && side != nil {
				continue
			}
			// a gap before each speech, and before the narrator takes over
			if len(objs) > 0 && (l.Kind == trCue || l.NarratorStart && v.narrator || l.Kind == trHeading) {
				gap := canvas.NewRectangle(color.Transparent)
				gap.SetMinSize(fyne.NewSize(0, 14))
				objs = append(objs, gap)
			}
			if o := v.row(l, side[i]); o != nil {
				objs = append(objs, o)
			}
		}
	} else {
		objs = append(objs, widget.NewLabel("No scene is picked: Pick chooses which are read."))
	}
	v.rows.Objects = objs
	v.reading.Refresh() // the rows take its theme,
	v.rows.Refresh()    // and are laid out at its sizes
	v.markLine()
}

// besideFrames are the frames beside each line, by its index in lines
// (nil: the frames between the lines, or none shown): a frame beside the
// line it follows, as [[board:]] notes illustrate the line above them
// (Storyboarder's, under the heading, beside the heading); one before any
// line beside the first.
func (v *tableReadView) besideFrames(lines []trLine) map[int][]trLine {
	if !v.boards || !v.beside {
		return nil
	}
	side := map[int][]trLine{}
	at, first := -1, -1
	var early []trLine
	for i, l := range lines {
		switch {
		case l.Kind != trFrame:
			at = i
			if first < 0 {
				first = i
			}
		case at < 0:
			early = append(early, l)
		default:
			side[at] = append(side[at], l)
		}
	}
	if first >= 0 && len(early) > 0 {
		side[first] = append(early, side[first]...)
	}
	return side
}

// frameObject is a frame's picture at w×h (its time under it, when
// timed), or a placeholder when it is missing.
func (v *tableReadView) frameObject(l trLine, w, h float32, timed bool) fyne.CanvasObject {
	img := canvas.NewImageFromImage(v.w.pictures().Thumbnail(l.Frame, 640, 360))
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(w, h))
	if img.Image == nil {
		miss := widget.NewLabel("▨ " + l.Frame)
		miss.Importance = widget.WarningImportance
		miss.Wrapping = fyne.TextWrapBreak
		return miss
	}
	if !timed || l.Secs <= 0 {
		return img
	}
	t := widget.NewLabel(strconv.FormatFloat(l.Secs, 'f', -1, 64) + " s")
	t.SizeName = theme.SizeNameCaptionText
	t.Importance = widget.LowImportance
	return container.NewVBox(img, t)
}

// row is a line's row: its text in its tint, the narrator's mark, a
// photo by a first cue, a frame's picture, the frames beside it (side);
// a background that shows when it is the line read.
func (v *tableReadView) row(l trLine, side []trLine) fyne.CanvasObject {
	s := v.script
	seg := &widget.TextSegment{Text: l.Text, Style: widget.RichTextStyleParagraph}
	seg.Style.Inline = false
	tint := fyne.ThemeColorName("")
	if v.tint && l.Speaker != "" {
		if i := s.tintOf(l.Speaker); i >= 0 {
			tint = tintName(i)
		}
	}
	indent := float32(0)
	var left fyne.CanvasObject
	var segs []widget.RichTextSegment
	switch l.Kind {
	case trFrame:
		if !v.boards {
			return nil
		}
		if v.w.narrow {
			return container.NewHBox(v.frameObject(l, 280, 158, false))
		}
		return container.NewHBox(v.frameObject(l, 400, 225, false))
	case trHeading:
		seg.Style = widget.RichTextStyleSubHeading
		seg.Style.Inline = false
	case trCue:
		seg.Style.TextStyle.Bold = true
		seg.Style.ColorName = tint
		indent = 2
		if l.First {
			if img := v.photo(l.Speaker, 160); img != nil {
				ph := canvas.NewImageFromImage(img)
				ph.FillMode = canvas.ImageFillContain
				ph.SetMinSize(fyne.NewSize(72, 72))
				left = ph
			}
		}
	case trParen:
		seg.Style.TextStyle.Italic = true
		seg.Style.ColorName = tint
		indent = 3
	case trSpeech, trLyric:
		seg.Style.ColorName = tint
		seg.Style.TextStyle.Italic = l.Kind == trLyric
		indent = 2
	case trTransition:
		seg.Style.Alignment = fyne.TextAlignTrailing
	}
	if l.Narrator && v.narrator {
		if seg.Style.ColorName == "" {
			seg.Style.ColorName = trNarratorColor
		}
		if l.NarratorStart {
			segs = append(segs, &widget.TextSegment{Text: "NARRATOR", Style: widget.RichTextStyle{
				ColorName: trNarratorColor, SizeName: theme.SizeNameCaptionText, TextStyle: fyne.TextStyle{Bold: true}}})
		}
	}
	segs = append(segs, seg)
	text := widget.NewRichText(segs...)
	text.Wrapping = fyne.TextWrapWord
	var body fyne.CanvasObject = text
	if left != nil {
		body = container.NewBorder(nil, nil, left, nil, text)
	}
	if indent > 0 {
		pad := canvas.NewRectangle(color.Transparent)
		pad.SetMinSize(fyne.NewSize(indent*24, 0))
		var bar fyne.CanvasObject = pad
		if tint != "" {
			r := canvas.NewRectangle(trTints[s.tintOf(l.Speaker)])
			r.SetMinSize(fyne.NewSize(5, 0))
			bar = container.NewBorder(nil, nil, nil, r, pad)
		}
		body = container.NewBorder(nil, nil, bar, nil, body)
	}
	if len(side) > 0 {
		fw, fh := float32(300), float32(169)
		if v.w.narrow {
			fw, fh = 128, 72
		}
		var pics []fyne.CanvasObject
		for _, f := range side {
			pics = append(pics, v.frameObject(f, fw, fh, true))
		}
		col := container.New(fixedWidth(fw), container.NewVBox(pics...))
		body = container.NewBorder(nil, nil, nil, col, body)
	}
	bg := canvas.NewRectangle(color.Transparent)
	if l.Kind != trFrame {
		if _, ok := v.marked[l.Line]; !ok {
			v.marked[l.Line] = bg
		}
	}
	return container.NewStack(bg, newTRLine(v, l.Line, body))
}

// markLine shows the line read (v.line) and scrolls to it.
func (v *tableReadView) markLine() {
	for line, bg := range v.marked {
		c := color.Color(color.Transparent)
		if line == v.line {
			c = color.NRGBA{0xff, 0xff, 0xff, 0x22}
		}
		bg.FillColor = c
		bg.Refresh()
	}
	if bg := v.marked[v.line]; bg != nil && v.line >= 0 {
		v.scrollTo(bg)
	}
}

// scrollTo scrolls the row with background bg into sight (once laid out).
func (v *tableReadView) scrollTo(bg *canvas.Rectangle) {
	for _, o := range v.rows.Objects {
		st, ok := o.(*fyne.Container)
		if !ok || len(st.Objects) == 0 || st.Objects[0] != bg {
			continue
		}
		y := st.Position().Y
		h := v.scroll.Size().Height
		if y < v.scroll.Offset.Y || y+st.Size().Height > v.scroll.Offset.Y+h {
			v.scroll.Offset.Y = max(0, y-h/4)
			v.scroll.Refresh()
		}
	}
}

// goLine shows line (its scene, picked or not), as the line read.
func (v *tableReadView) goLine(line int) {
	v.update()
	if v.script == nil || len(v.script.Scenes) == 0 {
		return
	}
	n := strings.Count(v.text, "\n")
	line = max(0, min(line, n))
	sc := v.script.sceneAtLine(line)
	if sc < 0 {
		sc = 0
	}
	if sc != v.scene {
		v.scene, v.line = sc, line
		v.show()
		v.scroll.ScrollToTop()
		v.markLine()
		return
	}
	v.line = line
	v.title.SetText(v.position())
	v.markLine()
}

// tapLine: a line tapped is the line read (the others following see it).
func (v *tableReadView) tapLine(line int) {
	v.line = line
	v.markLine()
	v.moved()
}

// trLine is a line's row, tapped to make it the line read.
type trLineRow struct {
	widget.BaseWidget
	v    *tableReadView
	line int
	body fyne.CanvasObject
}

func newTRLine(v *tableReadView, line int, body fyne.CanvasObject) *trLineRow {
	r := &trLineRow{v: v, line: line, body: body}
	r.ExtendBaseWidget(r)
	return r
}

func (r *trLineRow) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(r.body) }

func (r *trLineRow) Tapped(*fyne.PointEvent) {
	r.v.tapLine(r.line)
	if c := fyne.CurrentApp().Driver().CanvasForObject(r); c != nil {
		c.Focus(r.v.area)
	}
}

// trArea is the reading area: it takes the keys (and the focus a tap gives).
type trArea struct {
	widget.BaseWidget
	v       *tableReadView
	content fyne.CanvasObject
	focused bool
	ring    *canvas.Rectangle // the focus seen: a line round the area (keyboard audit)
}

func newTRArea(v *tableReadView, content fyne.CanvasObject) *trArea {
	a := &trArea{v: v, content: content}
	a.ring = canvas.NewRectangle(color.Transparent)
	a.ring.StrokeWidth = 2
	a.ring.Hide()
	a.ExtendBaseWidget(a)
	return a
}

func (a *trArea) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(a.content, a.ring))
}
func (a *trArea) FocusGained() { a.focused = true; a.showRing(true) }
func (a *trArea) FocusLost()   { a.focused = false; a.showRing(false) }

// showRing shows or hides the focus line.
func (a *trArea) showRing(on bool) {
	a.ring.StrokeColor = theme.Color(theme.ColorNamePrimary) // the focus colour is too faint for a line round a large area
	a.ring.Hidden = !on
	a.ring.Refresh()
}
func (a *trArea) TypedRune(rune)            {}
func (a *trArea) TypedKey(e *fyne.KeyEvent) { a.v.key(e) }
func (a *trArea) Tapped(*fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(a); c != nil {
		c.Focus(a)
	}
}

// tableReadEdited: the script changed; the view reads it again once
// typing pauses.
func (w *MainWindow) tableReadEdited() {
	if !w.tableReadOpen() {
		return
	}
	w.debounce(&w.tableReadTimer, 300*time.Millisecond, func() {
		if w.tableReadOpen() {
			w.tableRead.update()
		}
	})
}

// tableReadMidView is the Table Read's middle tab (middle.go).
func tableReadMidView() *midView {
	return &midView{title: "Table Read", why: "The Table Read reads a screenplay",
		applies: func(w *MainWindow) bool { return !w.markdown },
		box:     func(w *MainWindow) fyne.CanvasObject { return w.tableReadView().box },
		focus:   func(w *MainWindow) fyne.Focusable { return w.tableReadView().area },
		shown:   func(w *MainWindow) { w.tableReadView().update() }}
}
