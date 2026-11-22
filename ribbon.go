package main

import (
	"image/color"
	"runtime"
	"strings"
	"time"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/tooltip"
)

// The ribbon (Preferences > General > Look: Ribbon) is the header bar's
// other look: a row of tabs and, under it, the chosen tab's groups of
// buttons, icons with their labels, large for the common actions. All of
// it comes from the command registry (commands.go: ribbonLayout), so a
// command is defined once and its rules (when it applies, when it can be
// given) grey or hide its button here as they do elsewhere. The menu bar
// is taken away where Fyne draws it in the window (Linux, Windows, the
// browser: menuless), so there is one bar, not two; its keys stay (as in
// full screen) and the ☰ button at the tab row's end opens the same menus.
// A Mac's menu bar is the system's and stays. The header's two rows are
// hidden while the ribbon shows.
//
// The Production tab follows the script's stage (the Production pane's
// strip: Lock, Revise, Break down, Schedule): its first group is that
// stage's actions. The buttons follow the script's state (a source, a
// lock, the document's kind) when typing pauses and when the menu is made
// again.
//
// At a phone's width the tabs are one row and the groups one row that
// scrolls sideways, every button at least 44 px high.
//
// Keyboard: F10, or Alt pressed and let go with no key between (where the
// ribbon took the menu bar's place; seen with the focus in the script or
// on the ribbon), shows key tips: a
// letter on each tab, then (a tab chosen by its letter) on each button
// of its groups; the letter runs it. Esc goes back a level, then leaves.
// Tab and the arrow keys move between the tabs and buttons; Enter or
// Space presses one. Screen readers hear each button's label and keys.

// lookPref is the setting of the header's look: "menus" or "ribbon".
const lookPref = "look"

// defaultLook is a new user's look: the ribbon (Joop, 2026-11-21). The
// tests start in Menus (TestMain), as they were written for it.
var defaultLook = "ribbon"

// ribbonOn reports whether the window shows the ribbon.
func (w *MainWindow) ribbonOn() bool {
	return w.settings != nil && w.settings.GetString(lookPref) == "ribbon"
}

// applyLook shows the header bar's rows or the ribbon, as set.
func (w *MainWindow) applyLook() {
	if w.headerArea == nil || w.headerBar == nil {
		return
	}
	var top fyne.CanvasObject = w.headerBar.container
	if w.ribbonOn() {
		if w.ribbon == nil {
			w.ribbon = newRibbon(w)
		}
		top = w.ribbon.box
		w.ribbon.refresh()
	} else if w.ribbon != nil {
		w.ribbon.leaveKeyTips(false)
	}
	if w.headerArea.Objects[0] != top {
		w.headerArea.Objects[0] = top
		w.headerArea.Refresh()
	}
	for _, v := range []*pageView{w.pages, w.mdPagesView()} { // the bar over the pages: when no ribbon is
		if v != nil {
			v.followLook()
		}
	}
	if w.menuless() != w.menuGone { // the menu bar away, or back
		w.refreshMenu()
	}
}

// nativeMenus: the menu bar is the system's (a Mac's), not in the window.
var nativeMenus = runtime.GOOS == "darwin" && !inBrowser

// menuless reports whether the ribbon replaces the in-window menu bar:
// it shows, and the menu bar is Fyne's own.
func (w *MainWindow) menuless() bool { return !nativeMenus && w.ribbonShown() }

// showMenusAt is the ☰ button: the menu bar's menus, as the role and the
// document have them, in a menu by anchor.
func (w *MainWindow) showMenusAt(anchor fyne.CanvasObject) {
	showMenuBy(w, anchor, w.menusMenu())
}

// menusMenu is the menu bar as one menu, a menu's items under its title;
// Quit last, as Fyne puts it in the menu bar's first menu.
func (w *MainWindow) menusMenu() *fyne.Menu {
	var items []*fyne.MenuItem
	for i, m := range w.buildMainMenu().Items {
		sub := fyne.NewMenu(m.Label, m.Items...)
		if i == 0 && !inBrowser {
			quit := fyne.NewMenuItem("Quit", func() { fyne.CurrentApp().Quit() })
			quit.IsQuit = true
			sub = fyne.NewMenu(m.Label, append(append([]*fyne.MenuItem{}, m.Items...), fyne.NewMenuItemSeparator(), quit)...)
		}
		item := fyne.NewMenuItem(m.Label, nil)
		item.ChildMenu = sub
		items = append(items, item)
	}
	return fyne.NewMenu("Menus", items...)
}

// ribbonShown reports whether the ribbon is in sight.
func (w *MainWindow) ribbonShown() bool {
	return w.ribbon != nil && w.ribbonOn() && w.headerArea != nil && w.headerArea.Visible() &&
		len(w.headerArea.Objects) > 0 && w.headerArea.Objects[0] == w.ribbon.box
}

// ribbonFollow refreshes the ribbon soon (the text changed: a lock, a
// source block), once typing pauses.
func (w *MainWindow) ribbonFollow() {
	if !w.ribbonShown() {
		return
	}
	w.debounce(&w.ribbonTimer, 300*time.Millisecond, func() { w.ribbon.refresh() })
}

// ribbonRefresh refreshes the ribbon at once, if it shows.
func (w *MainWindow) ribbonRefresh() {
	if w.ribbonShown() {
		w.ribbon.refresh()
	}
}

// ribbonBar is the window's ribbon.
type ribbonBar struct {
	w   *MainWindow
	box *fyne.Container

	tabBox   *fyne.Container // the tabs
	tabs     []*ribbonTabButton
	title    *scriptTitle
	palette  *ribbonButton // the command palette's, at the tab row's end
	menus    *ribbonButton // ☰: the menus, beside it
	tabRow   *fyne.Container
	groupBox *fyne.Container // the groups of the tab chosen
	groups   *container.Scroll

	selected string // the tab chosen
	// previewing: the Preview tab is there (it comes forward when it
	// comes); beforePreview: the tab chosen before it, chosen again when
	// it goes
	beforePreview string
	previewing    bool
	pageCount     *widget.Label // the Preview tab's "Page n of N"
	env           cmdEnv
	stage         int    // the Production stage shown (stageLock …)
	builtFor      string // what the groups were made for
	phone         bool

	buttons  []*ribbonButton            // the groups' buttons, in order
	elements map[string]*ribbonButton   // the element buttons, by element
	format   *widget.Select             // the script format, on the Write tab
	showing  bool                       // the format select shows, not chooses
	byID     map[string][]*ribbonButton // the buttons by command id

	tipsOn   bool
	tipLevel int    // 1: the tabs', 2: the buttons'
	typed    string // the key tip's letters typed so far
	tipsFrom fyne.Focusable
	altAlone bool // Alt went down with no other key yet
}

func newRibbon(w *MainWindow) *ribbonBar {
	r := &ribbonBar{w: w, selected: "Write", stage: -1, byID: map[string][]*ribbonButton{}}
	r.tabBox = container.NewHBox()
	for _, t := range commands().ribbon {
		title := t.title
		tb := newRibbonTabButton(r, title, func() { r.selectTab(title) })
		r.tabs = append(r.tabs, tb)
		r.tabBox.Add(tb)
	}
	r.title = newScriptTitle("Untitled", w.renameScript)
	r.title.Alignment = fyne.TextAlignCenter
	r.title.TextStyle = fyne.TextStyle{Bold: true}
	r.title.Truncation = fyne.TextTruncateEllipsis
	if w.headerBar != nil {
		r.title.SetText(w.headerBar.titleLabel.Text)
	}
	end := commands().ribbonEnd // the palette's, then ☰
	r.palette = newRibbonButton(r, cmd(end[0].id), false)
	r.palette.iconOnly = true
	r.menus = newRibbonButton(r, cmd(end[1].id), false)
	r.menus.iconOnly = true
	r.menus.homeLabel = "Menus"
	r.format = newScriptFormatSelect(w.lex, nil, func(key string) {
		if !r.showing {
			w.setScriptFormat(key)
			_ = w.settings.Save()
			w.app.applySettingsToWindows()
		}
	})
	fyne.SetAccessibleLabel(r.format, "Script format")
	r.groupBox = container.NewHBox()
	r.groups = container.NewHScroll(r.groupBox)
	r.tabRow = container.New(&tabRowLayout{tabs: r.tabBox}, container.NewHScroll(r.tabBox), r.title, container.NewHBox(r.palette, r.menus))
	r.box = container.NewVBox(r.tabRow, widget.NewSeparator(), r.groups)
	return r
}

// tabRowLayout puts the tabs at the left (scrolling sideways where the
// row is too narrow for them: a phone), the palette's button at the
// right, the script's name between them if there is room.
type tabRowLayout struct{ tabs *fyne.Container }

func (l *tabRowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	h := max(objects[0].MinSize().Height, l.tabs.MinSize().Height, objects[2].MinSize().Height)
	return fyne.NewSize(objects[2].MinSize().Width+2*ribbonPhoneHeight, h)
}

func (l *tabRowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	tabs, title, pal := objects[0], objects[1], objects[2]
	pw := pal.MinSize().Width
	pal.Move(fyne.NewPos(size.Width-pw, 0))
	pal.Resize(fyne.NewSize(pw, size.Height))
	tw := min(l.tabs.MinSize().Width, size.Width-pw)
	tabs.Move(fyne.NewPos(0, 0))
	tabs.Resize(fyne.NewSize(tw, size.Height))
	room := size.Width - pw - tw
	setVisible(title, room >= 80)
	title.Move(fyne.NewPos(tw, (size.Height-title.MinSize().Height)/2))
	title.Resize(fyne.NewSize(max(room, 0), title.MinSize().Height))
}

// tabShown reports whether the tab titled t applies in env.
func tabShown(t ribbonTab, env cmdEnv) bool { return (t.when == nil || t.when(env)) && t.role.in(env) }

// selectTab shows a tab's groups.
func (r *ribbonBar) selectTab(title string) {
	r.selected = title
	r.refresh()
}

// tabOf is the registry's tab titled title.
func tabOf(title string) (ribbonTab, bool) {
	for _, t := range commands().ribbon {
		if t.title == title {
			return t, true
		}
	}
	return ribbonTab{}, false
}

// productionStage is the stage the Production pane shows (the one looked
// at, while it is in sight), else the script's own.
func (w *MainWindow) productionStage() int {
	if w.markdown || w.textEditor == nil {
		return stageLock
	}
	if w.production != nil && w.production.shown() {
		return w.production.view()
	}
	stage, _, _ := w.scriptStage()
	return stage
}

// refresh shows the ribbon for the window as it is now: the tabs that
// apply, the chosen tab's groups (made again when what they hold
// changed), each button enabled as its command's rules say.
func (r *ribbonBar) refresh() {
	w := r.w
	env := w.cmdEnv()
	r.env = env
	phone := w.narrow
	for i, t := range commands().ribbon {
		setVisible(r.tabs[i], tabShown(t, env))
	}
	r.menus.follow(env)
	if env.preview != r.previewing { // a preview opened or closed
		r.previewing = env.preview
		if env.preview {
			r.beforePreview, r.selected = r.selected, "Preview"
		} else if r.selected == "Preview" {
			r.selected = r.beforePreview
		}
	}
	if t, ok := tabOf(r.selected); !ok || !tabShown(t, env) {
		r.selected = "Write"
	}
	for _, tb := range r.tabs {
		tb.setSelected(tb.title == r.selected)
	}
	r.tabInSight()
	stage := -1
	if r.selected == "Production" && !env.markdown {
		stage = w.productionStage()
	}
	r.stage = stage
	key := strings.Join([]string{r.selected, boolKey(env.markdown), boolKey(env.source), boolKey(env.browser), boolKey(env.pandoc), boolKey(env.writer),
		boolKey(env.away), boolKey(env.swappedDoc), boolKey(phone), boolKey(env.preview), boolKey(env.livePreview), string(rune('0' + stage + 1)), w.scriptFormatKey(), w.language, w.newScriptLanguage()}, "\x00")
	if key != r.builtFor {
		r.builtFor = key
		r.phone = phone
		r.build()
	}
	for _, b := range r.buttons {
		b.follow(env)
	}
	r.palette.follow(env)
	r.tabRow.Refresh()
	if r.tipsOn && r.tipLevel == 2 {
		r.assignTips(2) // other buttons now
	}
	r.followPreview()
}

// tabInSight scrolls the tab row (a phone's, too narrow for them all) to
// the tab chosen: the Preview tab, last, comes forward out of sight.
func (r *ribbonBar) tabInSight() {
	sc, ok := r.tabRow.Objects[0].(*container.Scroll)
	if !ok || sc.Size().Width <= 0 {
		return
	}
	x := float32(0)
	for _, tb := range r.tabs {
		if !tb.Visible() {
			continue
		}
		w := tb.MinSize().Width
		if tb.title == r.selected {
			switch {
			case x < sc.Offset.X:
				sc.Offset.X = x
			case x+w > sc.Offset.X+sc.Size().Width:
				sc.Offset.X = x + w - sc.Size().Width
			default:
				return
			}
			sc.Refresh()
			return
		}
		x += w + theme.Padding()
	}
}

func boolKey(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// build makes the chosen tab's groups.
func (r *ribbonBar) build() {
	w := r.w
	for _, b := range r.buttons { // going: a tip of theirs goes with them
		b.hover.Out(w.tooltips, b.hoverTip())
	}
	r.buttons, r.byID, r.elements = nil, map[string][]*ribbonButton{}, map[string]*ribbonButton{}
	r.groupBox.Objects = nil
	t, _ := tabOf(r.selected)
	// every tab the same height (three small buttons and a title), so the
	// script does not jump when another is chosen
	title := widget.NewLabel("X")
	title.SizeName = theme.SizeNameCaptionText
	rows := ribbonPhoneHeight
	if !r.phone {
		rows = 3 * ribbonSmallHeight
	}
	cell := canvas.NewRectangle(color.Transparent)
	cell.SetMinSize(fyne.NewSize(1, float32(rows)))
	h := container.NewBorder(nil, title, nil, nil, container.NewVBox(cell)).MinSize().Height
	if !r.phone {
		h += 2 * theme.Padding() // the VBox's gaps between three
	}
	strut := canvas.NewRectangle(color.Transparent)
	strut.SetMinSize(fyne.NewSize(0, h))
	r.groupBox.Add(strut)
	first := true
	for _, g := range t.groups {
		content := r.groupContent(g)
		if content == nil {
			continue
		}
		if !first {
			r.groupBox.Add(widget.NewSeparator())
		}
		first = false
		caption := g.title
		if r.env.markdown && g.mdTitle != "" {
			caption = g.mdTitle
		}
		title := widget.NewLabel(caption)
		title.SizeName = theme.SizeNameCaptionText
		title.Importance = widget.LowImportance
		title.Alignment = fyne.TextAlignCenter
		fyne.SetAccessibleLabel(title, caption+" group")
		r.groupBox.Add(container.NewBorder(nil, title, nil, nil, content))
	}
	r.groupBox.Refresh()
	r.groups.Refresh()
	r.showElement(w.currentElement)
}

// groupContent is a group's buttons: large ones a column each, small
// ones three to a column (all one row on a phone); nil if none apply.
func (r *ribbonBar) groupContent(g ribbonGroup) fyne.CanvasObject {
	env := r.env
	if g.when != nil && !g.when(env) {
		return nil
	}
	if g.stage != 0 && g.stage != r.stage+1 {
		return nil
	}
	var cols []fyne.CanvasObject
	var smalls []fyne.CanvasObject
	var row []fyne.CanvasObject // compact buttons, side by side
	flushRow := func() {
		if len(row) > 0 {
			smalls = append(smalls, container.New(layout.NewCustomPaddedHBoxLayout(2), row...))
			row = nil
		}
	}
	flush := func() {
		flushRow()
		for len(smalls) > 0 {
			n := min(3, len(smalls))
			cols = append(cols, container.New(ribbonRows{}, smalls[:n]...))
			smalls = smalls[n:]
		}
	}
	addSmall := func(o fyne.CanvasObject) {
		if r.phone {
			cols = append(cols, o)
			return
		}
		smalls = append(smalls, o)
	}
	for _, h := range g.homes {
		if h.sep { // a row of compact buttons ends
			flushRow()
			continue
		}
		if h.keys && r.phone || h.phoneOnly && !r.phone {
			continue
		}
		switch h.id {
		case "elements":
			if !scriptEditing(env) { // the main editor holds Markdown, or no editor (swap.go)
				continue
			}
			for _, el := range toolbarElements {
				c := cmd(elementCommandID(el))
				b := newRibbonButton(r, c, false)
				b.label, b.mark = el, elementMarks[el]
				b.keys = c.keysShort() // typed all the time: the key shows
				r.elements[el] = b
				r.add(b)
				addSmall(b)
			}
			continue
		case "scriptFormat":
			if env.markdown {
				continue
			}
			r.showing = true
			conf := r.w.lex()
			r.format.SetOptions(conf.formatNames())
			conf.selectFormat(r.format, r.w.scriptFormatKey())
			r.showing = false
			addSmall(r.format)
			continue
		case "pageCount": // the Preview tab's "Page n of N"
			if r.pageCount == nil {
				r.pageCount = widget.NewLabel("")
				r.pageCount.Alignment = fyne.TextAlignCenter
				fyne.SetAccessibleLabel(r.pageCount, "Page")
			}
			r.pageCount.SetText("")
			if v := r.w.previewView(); v != nil {
				r.pageCount.SetText(v.count.Text)
			}
			addSmall(r.pageCount)
			continue
		case "formatButtons":
			if !scriptEditing(env) { // they write in the script, in the main editor
				continue
			}
			for _, fb := range r.w.lex().formatButtons(r.w.scriptFormatKey()) {
				fb := fb
				b := newRibbonButton(r, nil, false)
				b.label, b.icon = fb.Label, insertBlockIcon
				b.onTap = func() {
					if r.w.scriptAway(fb.Label) { // the script is in the side panes (swap.go)
						return
					}
					r.w.textEditor.Edit(func(b *buffer.Buffer) { insertButton(b, fb) })
					r.w.fyneWindow.Canvas().Focus(r.w.textEditor)
				}
				r.add(b)
				addSmall(b)
			}
			continue
		}
		c := cmd(h.id)
		if !c.applies(env) {
			continue
		}
		b := newRibbonButton(r, c, h.large)
		b.compact = h.compact
		if h.keys {
			b.keys = c.keysShort()
		}
		if h.label != "" { // this place's own (Source Pane)
			b.label, b.homeLabel = h.label, h.label
			b.name = h.label + c.keysTip()
		}
		if h.short != "" {
			b.label = h.short
		}
		r.add(b)
		switch {
		case h.large || r.phone:
			flush()
			cols = append(cols, b)
		case h.compact:
			row = append(row, b)
		default:
			flushRow()
			addSmall(b)
		}
	}
	flush()
	if len(cols) == 0 {
		return nil
	}
	return container.NewHBox(cols...)
}

// ribbonRows lays a column of small buttons out in the group's three
// rows, each row a third of the height, each button centred in its row:
// the rows line up across columns and groups, whatever is in them (the
// script format's list is taller than a button).
type ribbonRows struct{}

func (ribbonRows) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var w float32
	for _, o := range objects {
		w = max(w, o.MinSize().Width)
	}
	return fyne.NewSize(w, 3*ribbonSmallHeight+2*theme.Padding())
}

func (ribbonRows) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	row := size.Height / 3
	for i, o := range objects {
		h := min(max(o.MinSize().Height, ribbonSmallHeight), row)
		if _, ok := o.(*widget.Select); ok {
			h = row - 2 // as tall as the row lets it
		}
		o.Move(fyne.NewPos(0, float32(i)*row+(row-h)/2))
		o.Resize(fyne.NewSize(size.Width, h))
	}
}

// add keeps b among the groups' buttons.
func (r *ribbonBar) add(b *ribbonButton) {
	r.buttons = append(r.buttons, b)
	if b.c != nil {
		r.byID[b.c.id] = append(r.byID[b.c.id], b)
	}
}

// showElement shows the cursor line's element among the element buttons.
func (r *ribbonBar) showElement(el string) {
	for e, b := range r.elements {
		b.setSelected(e == el)
	}
}

// showFormats shows the formats again (one added or chosen).
func (r *ribbonBar) showFormats() {
	r.builtFor = "" // the format's own buttons made again
	r.refresh()
}

// shownTabs are the tabs that apply.
func (r *ribbonBar) shownTabs() []*ribbonTabButton {
	var out []*ribbonTabButton
	for _, t := range r.tabs {
		if t.Visible() {
			out = append(out, t)
		}
	}
	return out
}

// move gives the focus to the tab or button step from o in its row.
func (r *ribbonBar) move(o fyne.Focusable, step int) {
	c := r.w.fyneWindow.Canvas()
	if tb, ok := o.(*ribbonTabButton); ok {
		tabs := r.shownTabs()
		for i, t := range tabs {
			if t == tb {
				c.Focus(tabs[(i+step+len(tabs))%len(tabs)])
				return
			}
		}
		return
	}
	var row []*ribbonButton
	for _, b := range r.buttons {
		if !b.Disabled() {
			row = append(row, b)
		}
	}
	for i, b := range row {
		if fyne.Focusable(b) == o {
			c.Focus(row[(i+step+len(row))%len(row)])
			return
		}
	}
}

// down and up go between the tab row and the group row.
func (r *ribbonBar) down() {
	for _, b := range r.buttons {
		if !b.Disabled() {
			r.w.fyneWindow.Canvas().Focus(b)
			return
		}
	}
}

func (r *ribbonBar) up() {
	for _, t := range r.shownTabs() {
		if t.title == r.selected {
			r.w.fyneWindow.Canvas().Focus(t)
		}
	}
}

// Key tips.

// keyTipsFor are the key tips of labels, one each, unique and none the
// start of another: a word's first letter if free, else another letter
// of the label, else a digit, else any letter; past those (35), two
// keys starting with 0 (36 more).
func keyTipsFor(labels []string) []string {
	used := map[string]bool{}
	out := make([]string, len(labels))
	try := func(i int, s string) bool {
		if s == "" || used[s] {
			return false
		}
		used[s] = true
		out[i] = s
		return true
	}
	for i, label := range labels {
		var cands []string
		for _, word := range strings.FieldsFunc(label, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
			cands = append(cands, keyTipRune([]rune(word)[0]))
		}
		for _, ch := range label {
			cands = append(cands, keyTipRune(ch))
		}
		done := false
		for _, s := range cands {
			if s != "0" && try(i, s) {
				done = true
				break
			}
		}
		for d := '1'; !done && d <= '9'; d++ {
			done = try(i, string(d))
		}
		for ch := 'A'; !done && ch <= 'Z'; ch++ { // any letter free
			done = try(i, string(ch))
		}
		for ch := 'A'; !done && ch <= 'Z'; ch++ {
			done = try(i, "0"+string(ch))
		}
		for d := '0'; !done && d <= '9'; d++ {
			done = try(i, "0"+string(d))
		}
	}
	return out
}

// keyTipRune is a label's character as a key tip ("" for none: not a
// letter or digit of A–Z, 0–9).
func keyTipRune(ch rune) string {
	ch = unicode.ToUpper(ch)
	if ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' {
		return string(ch)
	}
	return ""
}

// showKeyTips shows the tabs' key tips, the focus on the chosen tab (the
// letters typed go to the ribbon, not the script).
func (r *ribbonBar) showKeyTips() {
	if !r.w.ribbonShown() {
		return
	}
	c := r.w.fyneWindow.Canvas()
	if !r.tipsOn {
		r.tipsFrom = c.Focused()
	}
	r.tipsOn = true
	r.assignTips(1)
	r.up()
	r.w.setStatus("Key tips: a letter chooses; Esc goes back")
}

// assignTips puts the key tips of a level on its tabs or buttons (and
// takes the other level's away).
func (r *ribbonBar) assignTips(level int) {
	r.tipLevel, r.typed = level, ""
	for _, t := range r.tabs {
		t.setTip("")
	}
	r.palette.setTip("")
	r.menus.setTip("")
	for _, b := range r.buttons {
		b.setTip("")
	}
	if level == 1 {
		tabs := r.shownTabs()
		labels := make([]string, 0, len(tabs)+2)
		for _, t := range tabs {
			labels = append(labels, t.title)
		}
		labels = append(labels, r.palette.label, "Menus")
		tips := keyTipsFor(labels)
		for i, t := range tabs {
			t.setTip(tips[i])
		}
		r.palette.setTip(tips[len(tabs)])
		r.menus.setTip(tips[len(tabs)+1])
		return
	}
	var bs []*ribbonButton
	var labels []string
	for _, b := range r.buttons {
		if !b.Disabled() {
			bs = append(bs, b)
			labels = append(labels, b.label)
		}
	}
	tips := keyTipsFor(labels)
	for i, b := range bs {
		b.setTip(tips[i])
	}
}

// focusLeft takes the key tips away once the focus is out of the ribbon
// (a click in the script, Tab out of it): looked at after the focus
// moved, as the widget losing it does not know where it goes.
func (r *ribbonBar) focusLeft() {
	if !r.tipsOn {
		return
	}
	r.w.after(0, func() {
		switch r.w.fyneWindow.Canvas().Focused().(type) {
		case *ribbonTabButton, *ribbonButton:
		default:
			r.leaveKeyTips(false)
		}
	})
}

// keyTipTyped takes a letter typed while key tips show; it reports
// whether it was one.
func (r *ribbonBar) keyTipTyped(ch rune) bool {
	if !r.tipsOn {
		return false
	}
	k := keyTipRune(ch)
	if k == "" {
		return true // nothing, but not typed in the script either
	}
	typed := r.typed + k
	var match func()
	prefix := false
	if r.tipLevel == 1 {
		for _, t := range r.shownTabs() {
			if t.tip == typed {
				title := t.title
				match = func() {
					r.selectTab(title)
					r.assignTips(2)
					r.down()
				}
			} else if strings.HasPrefix(t.tip, typed) {
				prefix = true
			}
		}
		if r.palette.tip == typed {
			match = func() {
				r.leaveKeyTips(true)
				r.w.showPalette()
			}
		}
		if r.menus.tip == typed {
			match = func() {
				r.leaveKeyTips(true)
				r.w.showMenusAt(r.menus)
			}
		}
	} else {
		for _, b := range r.buttons {
			if b.tip == "" {
				continue
			}
			if b.tip == typed {
				b := b
				match = func() {
					r.leaveKeyTips(true)
					b.press()
				}
			} else if strings.HasPrefix(b.tip, typed) {
				prefix = true
			}
		}
	}
	switch {
	case match != nil:
		match()
	case prefix:
		r.typed = typed
	default:
		r.typed = ""
	}
	return true
}

// keyTipEscape is Esc while key tips show: back to the tabs' tips, or
// out of them.
func (r *ribbonBar) keyTipEscape() bool {
	if !r.tipsOn {
		return false
	}
	if r.tipLevel == 2 {
		r.assignTips(1)
		r.up()
		return true
	}
	r.leaveKeyTips(true)
	return true
}

// leaveKeyTips takes the key tips away, the focus back where it was
// (restore).
func (r *ribbonBar) leaveKeyTips(restore bool) {
	if !r.tipsOn {
		return
	}
	r.tipsOn, r.typed = false, ""
	for _, t := range r.tabs {
		t.setTip("")
	}
	r.palette.setTip("")
	r.menus.setTip("")
	for _, b := range r.buttons {
		b.setTip("")
	}
	if restore {
		c := r.w.fyneWindow.Canvas()
		if r.tipsFrom != nil {
			c.Focus(r.tipsFrom)
		} else {
			r.w.focusMain()
		}
	}
	r.tipsFrom = nil
}

// altKey sees each key go down (keyPreview): Alt going down alone, where
// the ribbon took the menu bar's place, may be Alt alone (altUp); any
// other key between makes it Alt with a key (Alt+J, Alt+Enter).
func (w *MainWindow) altKey(name fyne.KeyName) {
	r := w.ribbon
	if r == nil {
		return
	}
	if name != desktop.KeyAltLeft && name != desktop.KeyAltRight || !w.menuGone || !w.ribbonShown() {
		r.altAlone = false
		return
	}
	r.altAlone = true // held, it repeats: still alone
}

// altUp sees a key let go: Alt let go alone shows the key tips, or takes
// them away.
func (w *MainWindow) altUp(name fyne.KeyName) {
	r := w.ribbon
	if r == nil || !r.altAlone || name != desktop.KeyAltLeft && name != desktop.KeyAltRight {
		return
	}
	r.altAlone = false
	if !w.menuGone || !w.ribbonShown() {
		return
	}
	if r.tipsOn {
		r.leaveKeyTips(true)
		return
	}
	r.showKeyTips()
}

// altCancel: the keyboard went elsewhere while Alt was down (Alt+Tab to
// another window, which sends Alt's let-go after): no key tips for it.
func (w *MainWindow) altCancel() {
	if w.ribbon != nil {
		w.ribbon.altAlone = false
	}
}

// The tabs and buttons.

// ribbonTabButton is a tab of the ribbon.
type ribbonTabButton struct {
	widget.BaseWidget
	r                          *ribbonBar
	title                      string
	onTap                      func()
	selected, hovered, focused bool
	tip                        string
}

func newRibbonTabButton(r *ribbonBar, title string, tap func()) *ribbonTabButton {
	t := &ribbonTabButton{r: r, title: title, onTap: tap}
	t.ExtendBaseWidget(t)
	return t
}

func (t *ribbonTabButton) setSelected(on bool) {
	if t.selected != on {
		t.selected = on
		t.Refresh()
	}
}

func (t *ribbonTabButton) setTip(tip string) {
	if t.tip != tip {
		t.tip = tip
		t.Refresh()
	}
}

func (t *ribbonTabButton) Tapped(*fyne.PointEvent) { t.onTap() }
func (t *ribbonTabButton) MouseIn(*desktop.MouseEvent) {
	t.hovered = true
	t.Refresh()
}
func (t *ribbonTabButton) MouseMoved(*desktop.MouseEvent) {}
func (t *ribbonTabButton) MouseOut() {
	t.hovered = false
	t.Refresh()
}
func (t *ribbonTabButton) FocusGained() {
	t.focused = true
	t.Refresh()
}
func (t *ribbonTabButton) FocusLost() {
	t.focused = false
	t.Refresh()
	t.r.focusLeft()
	t.r.w.altCancel()
}

// KeyDown and KeyUp: Alt let go alone takes the key tips away (altUp).
func (t *ribbonTabButton) KeyDown(*fyne.KeyEvent) {}
func (t *ribbonTabButton) KeyUp(k *fyne.KeyEvent) { t.r.w.altUp(k.Name) }
func (b *ribbonButton) KeyDown(*fyne.KeyEvent)    {}
func (b *ribbonButton) KeyUp(k *fyne.KeyEvent)    { b.r.w.altUp(k.Name) }
func (b *ribbonButton) onTabRow() bool            { return b == b.r.palette || b == b.r.menus }
func (t *ribbonTabButton) TypedRune(ch rune) {
	if ch == ' ' && !t.r.tipsOn {
		t.onTap()
		return
	}
	t.r.keyTipTyped(ch)
}
func (t *ribbonTabButton) TypedKey(k *fyne.KeyEvent) {
	switch k.Name {
	case fyne.KeyLeft:
		t.r.move(t, -1)
	case fyne.KeyRight:
		t.r.move(t, 1)
	case fyne.KeyDown:
		t.onTap()
		t.r.down()
	case fyne.KeyReturn, fyne.KeyEnter:
		t.onTap()
		t.r.down()
	case fyne.KeyEscape:
		if !t.r.keyTipEscape() {
			t.r.w.focusMain()
		}
	}
}

func (t *ribbonTabButton) AccessibilityLabel() string             { return t.title + " tab" }
func (t *ribbonTabButton) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleTab }
func (t *ribbonTabButton) AccessibilityStates() []fyne.AccessibleState {
	if t.selected {
		return []fyne.AccessibleState{fyne.AccessibleStateSelected}
	}
	return nil
}
func (t *ribbonTabButton) AccessibilityActions() []fyne.AccessibleAction {
	return []fyne.AccessibleAction{fyne.AccessibleActionSelect}
}
func (t *ribbonTabButton) AccessibilityPerformAction(fyne.AccessibleAction) bool {
	t.onTap()
	return true
}

func (t *ribbonTabButton) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.Transparent)
	bg.CornerRadius = theme.InputRadiusSize()
	line := canvas.NewRectangle(color.Transparent)
	text := canvas.NewText(t.title, theme.Color(theme.ColorNameForeground))
	text.TextSize = theme.TextSize()
	badge := newKeyTipBadge()
	r := &ribbonRenderer{objects: []fyne.CanvasObject{bg, line, text, badge.bg, badge.text}}
	r.layout = func(size fyne.Size) {
		bg.Resize(size)
		line.Move(fyne.NewPos(0, size.Height-3))
		line.Resize(fyne.NewSize(size.Width, 3))
		ts := text.MinSize()
		text.Move(fyne.NewPos((size.Width-ts.Width)/2, (size.Height-ts.Height)/2))
		badge.place(size, true)
	}
	r.min = func() fyne.Size {
		ts := fyne.MeasureText(t.title, theme.TextSize(), fyne.TextStyle{Bold: true})
		h := ts.Height + 2*theme.InnerPadding()
		if t.r.phone {
			h = max(h, 44)
		}
		return fyne.NewSize(ts.Width+3*theme.InnerPadding(), h)
	}
	r.refresh = func() {
		text.Text = t.title
		text.TextStyle.Bold = t.selected
		text.Color = theme.Color(theme.ColorNameForeground)
		switch {
		case t.focused:
			bg.FillColor = theme.Color(theme.ColorNameFocus)
		case t.hovered:
			bg.FillColor = theme.Color(theme.ColorNameHover)
		default:
			bg.FillColor = color.Transparent
		}
		line.FillColor = color.Transparent
		if t.selected {
			line.FillColor = theme.Color(theme.ColorNamePrimary)
		}
		badge.show(t.tip)
		bg.Refresh()
		line.Refresh()
		text.Refresh()
	}
	r.refresh()
	return r
}

// ribbonButton is a button of the ribbon: a command's (c), or a script
// format's own (onTap); large is an icon over its label, small an icon
// before it; on a phone every one is at least 44 px high. mark, if set,
// is shown in the icon's place (an element's Fountain mark).
type ribbonButton struct {
	widget.BaseWidget
	r     *ribbonBar
	c     *command
	label string
	icon  fyne.Resource
	mark  string
	// keys, if set, shows dim after the label (an element's Ctrl+1:
	// elements are given by their keys all the time; other buttons leave
	// their keys to the tooltip, as Word's ribbon does)
	keys string
	// homeLabel is its place's own label, if any (Source Pane)
	homeLabel string
	iconOnly  bool // no label on a phone (the palette's: the tabs have the row)
	compact   bool // an icon alone, always (B I U)
	large     bool
	onTap     func()

	disabled, selected, checked, hovered, focused bool
	dot                                           bool   // a dot by the icon (Refresh: the pages are stale)
	tip                                           string // the key tip
	name                                          string // the screen reader's
	hover                                         tooltip.Hover
}

// hoverTip is b's tooltip: its full name and keys, as a screen reader
// says it (the label may be a short one).
func (b *ribbonButton) hoverTip() string { return b.AccessibilityLabel() }

func newRibbonButton(r *ribbonBar, c *command, large bool) *ribbonButton {
	b := &ribbonButton{r: r, c: c, large: large}
	if c != nil {
		b.label = c.ribbonLabelIn(r.env)
		if c.ribbonLabelFor != nil {
			b.label = c.ribbonLabelFor(r.w)
		}
		if c.id == "write.place" { // in the script's language: BIN./BUI. in Dutch
			if _, _, both, ok := placeWords(r.w.sceneHeaders()); ok {
				b.label = both
			}
		}
		if c.icon != nil {
			b.icon = c.icon()
		}
		b.name = c.labelIn(r.env) + c.keysTip()
		if c.nameFor != nil {
			b.name = c.nameFor(r.w) + c.keysTip()
		}
		b.dot = c.dot != nil && c.dot(r.w)
	}
	b.ExtendBaseWidget(b)
	return b
}

// setDot shows the dot by b's icon, or not.
func (b *ribbonButton) setDot(on bool) {
	if b.dot != on {
		b.dot = on
		b.Refresh()
	}
}

// follow enables b and shows it checked as its command's rules say.
func (b *ribbonButton) follow(env cmdEnv) {
	if b.c == nil {
		return
	}
	dis := !b.c.canRun(env)
	checked := b.c.checked != nil && b.c.checked(b.r.w)
	name := b.c.labelIn(env) + b.c.keysTip() // its menu item's words
	if b.homeLabel != "" {
		name = b.homeLabel + b.c.keysTip()
	}
	if b.c.nameFor != nil {
		name = b.c.nameFor(b.r.w) + b.c.keysTip()
	}
	label := b.label
	if b.c.ribbonLabelFor != nil {
		label = b.c.ribbonLabelFor(b.r.w)
	}
	dot := b.c.dot != nil && b.c.dot(b.r.w)
	if dis != b.disabled || checked != b.checked || name != b.name || label != b.label || dot != b.dot {
		b.disabled, b.checked, b.name, b.label, b.dot = dis, checked, name, label, dot
		b.Refresh()
	}
}

func (b *ribbonButton) setSelected(on bool) {
	if b.selected != on {
		b.selected = on
		b.Refresh()
	}
}

func (b *ribbonButton) setTip(tip string) {
	if b.tip != tip {
		b.tip = tip
		b.Refresh()
	}
}

// press does what b does: its command as a button does it (a picker
// opening by it).
func (b *ribbonButton) press() {
	if b.disabled {
		return
	}
	b.hover.Out(b.r.w.tooltips, b.hoverTip()) // by a key too: the tip goes with the press
	if b.onTap != nil {
		b.onTap()
		return
	}
	c := b.c
	if c.submenu != nil { // Open Recent, Document Type: their menu by it
		showMenuBy(b.r.w, b, c.submenu(b.r.w))
		return
	}
	b.r.w.tapped(c, b)()
	b.r.w.ribbonRefresh() // a state it changed (a lock, a pane)
}

func (b *ribbonButton) Disabled() bool { return b.disabled }
func (b *ribbonButton) Enable() {
	b.disabled = false
	b.Refresh()
}
func (b *ribbonButton) Disable() {
	b.disabled = true
	b.Refresh()
}

func (b *ribbonButton) Tapped(*fyne.PointEvent) {
	b.hover.Out(b.r.w.tooltips, b.hoverTip())
	b.press()
}
func (b *ribbonButton) MouseIn(*desktop.MouseEvent) {
	b.hovered = true
	b.Refresh()
	b.hover.In(b.r.w.tooltips, b, b.hoverTip)
}
func (b *ribbonButton) MouseMoved(*desktop.MouseEvent) {}
func (b *ribbonButton) MouseOut() {
	b.hovered = false
	b.Refresh()
	b.hover.Out(b.r.w.tooltips, b.hoverTip())
}
func (b *ribbonButton) FocusGained() {
	b.focused = true
	b.Refresh()
	if !b.r.tipsOn { // the key tips say enough
		b.hover.Now(b.r.w.tooltips, b, b.hoverTip())
	}
}
func (b *ribbonButton) FocusLost() {
	b.focused = false
	b.Refresh()
	b.hover.Out(b.r.w.tooltips, b.hoverTip())
	b.r.focusLeft()
	b.r.w.altCancel()
}
func (b *ribbonButton) TypedRune(ch rune) {
	if ch == ' ' && !b.r.tipsOn {
		b.press()
		return
	}
	b.r.keyTipTyped(ch)
}
func (b *ribbonButton) TypedKey(k *fyne.KeyEvent) {
	switch k.Name {
	case fyne.KeyLeft:
		b.r.move(b, -1)
	case fyne.KeyRight:
		b.r.move(b, 1)
	case fyne.KeyUp:
		if !b.onTabRow() {
			b.r.up()
		}
	case fyne.KeyDown:
		if b.onTabRow() {
			b.r.down()
		}
	case fyne.KeyReturn, fyne.KeyEnter:
		b.r.leaveKeyTips(false)
		b.press()
	case fyne.KeyEscape:
		if !b.r.keyTipEscape() {
			b.r.w.focusMain()
		}
	}
}

func (b *ribbonButton) AccessibilityLabel() string {
	if b.name != "" {
		return b.name
	}
	return b.label
}
func (b *ribbonButton) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleButton }
func (b *ribbonButton) AccessibilityStates() []fyne.AccessibleState {
	var out []fyne.AccessibleState
	if b.disabled {
		out = append(out, fyne.AccessibleStateDisabled)
	}
	if b.checked {
		out = append(out, fyne.AccessibleStateChecked)
	}
	if b.selected {
		out = append(out, fyne.AccessibleStateSelected)
	}
	return out
}
func (b *ribbonButton) AccessibilityActions() []fyne.AccessibleAction {
	return []fyne.AccessibleAction{fyne.AccessibleActionPress}
}
func (b *ribbonButton) AccessibilityPerformAction(fyne.AccessibleAction) bool {
	b.press()
	return true
}

// largeLines is a large button's label on two lines, split at the space
// nearest its middle (one line if short).
func largeLines(label string) []string {
	if len([]rune(label)) <= 10 || !strings.Contains(label, " ") {
		return []string{label}
	}
	mid, best := len(label)/2, -1
	for i, ch := range label {
		if ch == ' ' && (best < 0 || abs(i-mid) < abs(best-mid)) {
			best = i
		}
	}
	return []string{label[:best], label[best+1:]}
}

// keysShort is c's first key as the platform writes it ("Ctrl+1", "⌘1"),
// "" if none.
func (c *command) keysShort() string {
	return strings.TrimSuffix(strings.TrimPrefix(c.keysTip(), " ("), ")")
}

// The sizes of the buttons.
const (
	ribbonSmallHeight = 28
	ribbonPhoneHeight = 48 // a finger's (44 at least)
	ribbonIconSmall   = 16
	ribbonIconLarge   = 28
)

func (b *ribbonButton) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.Transparent)
	bg.CornerRadius = theme.InputRadiusSize()
	img := canvas.NewImageFromResource(nil)
	img.FillMode = canvas.ImageFillContain
	mark := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	mark.TextStyle.Monospace = true
	mark.TextStyle.Bold = true
	keyT := canvas.NewText("", theme.Color(theme.ColorNamePlaceHolder))
	var lines []*canvas.Text
	badge := newKeyTipBadge()
	dot := canvas.NewCircle(theme.Color(theme.ColorNamePrimary))
	objects := []fyne.CanvasObject{bg, img, mark, keyT, dot}
	for i := 0; i < 2; i++ {
		t := canvas.NewText("", theme.Color(theme.ColorNameForeground))
		lines = append(lines, t)
		objects = append(objects, t)
	}
	objects = append(objects, badge.bg, badge.text)
	r := &ribbonRenderer{objects: objects}
	texts := func() []string {
		if b.iconOnly && b.r.phone || b == b.r.menus || b.compact {
			return nil // the palette's on a phone's tab row; ☰ always
		}
		if b.large || b.r.phone {
			if b.r.phone {
				return []string{b.label}
			}
			return largeLines(b.label)
		}
		return []string{b.label}
	}
	textSize := func() float32 {
		if b.large || b.r.phone {
			return theme.CaptionTextSize()
		}
		return theme.TextSize()
	}
	iconSize := func() float32 {
		switch {
		case b.r.phone:
			return 20
		case b.large:
			return ribbonIconLarge
		}
		return ribbonIconSmall
	}
	pad := theme.InnerPadding() / 2
	// keyShown: the keys after a small button's label (not on a phone:
	// no keyboard to type them on)
	keyShown := func() string {
		if b.large || b.compact || b.r.phone {
			return ""
		}
		return b.keys
	}
	keySize := func() float32 { return theme.CaptionTextSize() }
	keyGap := theme.Padding()
	r.min = func() fyne.Size {
		ts := textSize()
		var tw, th float32
		for _, l := range texts() {
			s := fyne.MeasureText(l, ts, fyne.TextStyle{})
			tw = max(tw, s.Width)
			th += s.Height
		}
		is := iconSize()
		if b.compact {
			if b.r.phone {
				return fyne.NewSquareSize(ribbonPhoneHeight)
			}
			return fyne.NewSquareSize(ribbonSmallHeight)
		}
		if b.large || b.r.phone {
			w := max(tw, is) + 2*pad + 4
			h := is + th + 3*pad
			if b.r.phone {
				return fyne.NewSize(max(w, ribbonPhoneHeight), max(h, ribbonPhoneHeight))
			}
			return fyne.NewSize(max(w, 52), max(h, 3*ribbonSmallHeight))
		}
		w := is + tw + 3*pad + 2
		if k := keyShown(); k != "" {
			w += keyGap + fyne.MeasureText(k, keySize(), fyne.TextStyle{}).Width
		}
		return fyne.NewSize(w, ribbonSmallHeight)
	}
	r.layout = func(size fyne.Size) {
		bg.Resize(size)
		is := iconSize()
		ts := textSize()
		ls := texts()
		if b.compact {
			img.Move(fyne.NewPos((size.Width-is)/2, (size.Height-is)/2))
			img.Resize(fyne.NewSize(is, is))
		} else if b.large || b.r.phone {
			// the icon and the first line at the same height in every
			// large button, one line or two: laid out as for two
			lh := fyne.MeasureText("Mg", ts, fyne.TextStyle{}).Height
			th := lh * float32(max(len(ls), 1))
			if !b.r.phone {
				th = 2 * lh
			}
			top := (size.Height - is - th - pad) / 2
			img.Move(fyne.NewPos((size.Width-is)/2, top))
			img.Resize(fyne.NewSize(is, is))
			y := top + is + pad
			for i, t := range lines {
				if i < len(ls) {
					s := fyne.MeasureText(ls[i], ts, fyne.TextStyle{})
					t.Move(fyne.NewPos((size.Width-s.Width)/2, y))
					y += s.Height
				}
			}
		} else {
			img.Move(fyne.NewPos(pad, (size.Height-is)/2))
			img.Resize(fyne.NewSize(is, is))
			if len(ls) > 0 {
				s := fyne.MeasureText(ls[0], ts, fyne.TextStyle{})
				lines[0].Move(fyne.NewPos(pad*2+is, (size.Height-s.Height)/2))
			}
			if k := keyShown(); k != "" { // right-aligned, on the label's line
				ks := fyne.MeasureText(k, keySize(), fyne.TextStyle{})
				ls := fyne.MeasureText("Mg", ts, fyne.TextStyle{})
				// the baselines close: the key's text sits low in the label's
				keyT.Move(fyne.NewPos(size.Width-pad-2-ks.Width, (size.Height-ls.Height)/2+(ls.Height-ks.Height)*0.6))
			}
		}
		ms := mark.MinSize()
		mark.Move(fyne.NewPos(img.Position().X+(is-ms.Width)/2, img.Position().Y+(is-ms.Height)/2))
		ds := max(is/3, 6) // the dot: over the icon's top right corner
		dot.Move(img.Position().AddXY(is-ds*0.6, -ds*0.4))
		dot.Resize(fyne.NewSquareSize(ds))
		badge.place(size, b.large || b.r.phone)
	}
	r.refresh = func() {
		fg := theme.Color(theme.ColorNameForeground)
		if b.disabled {
			fg = theme.Color(theme.ColorNameDisabled)
		}
		switch {
		case b.focused:
			bg.FillColor = theme.Color(theme.ColorNameFocus)
		case b.hovered && !b.disabled:
			bg.FillColor = theme.Color(theme.ColorNameHover)
		case b.selected || b.checked:
			bg.FillColor = theme.Color(theme.ColorNameSelection)
		default:
			bg.FillColor = color.Transparent
		}
		ic := b.icon
		if ic != nil && b.disabled {
			ic = theme.NewDisabledResource(ic)
		}
		img.Resource = ic
		img.Hidden = ic == nil || b.mark != ""
		mark.Text = b.mark
		mark.Color = fg
		keyT.Text = keyShown()
		keyT.TextSize = keySize()
		keyT.Color = theme.Color(theme.ColorNamePlaceHolder)
		if b.disabled {
			keyT.Color = theme.Color(theme.ColorNameDisabled)
		}
		keyT.Refresh()
		mark.TextSize = iconSize() * 0.8
		ls := texts()
		for i, t := range lines {
			t.Text = ""
			if i < len(ls) {
				t.Text = ls[i]
			}
			t.TextSize = textSize()
			t.Color = fg
			t.Refresh()
		}
		badge.show(b.tip)
		dot.FillColor = theme.Color(theme.ColorNamePrimary)
		dot.Hidden = !b.dot
		dot.Refresh()
		bg.Refresh()
		img.Refresh()
		mark.Refresh()
	}
	r.refresh()
	return r
}

// keyTipBadge is a key tip shown on a tab or button.
type keyTipBadge struct {
	bg   *canvas.Rectangle
	text *canvas.Text
}

func newKeyTipBadge() *keyTipBadge {
	k := &keyTipBadge{bg: canvas.NewRectangle(color.Black), text: canvas.NewText("", color.White)}
	k.bg.CornerRadius = 3
	k.text.TextStyle.Bold = true
	k.text.TextSize = theme.CaptionTextSize()
	k.bg.Hide()
	k.text.Hide()
	return k
}

// place puts the badge at the bottom middle (below: of a tab or a large
// button) or at the left (a small button's, over its icon).
func (k *keyTipBadge) place(size fyne.Size, below bool) {
	ts := fyne.MeasureText(k.text.Text+"W", k.text.TextSize, k.text.TextStyle)
	bw, bh := ts.Width, ts.Height
	pos := fyne.NewPos((size.Width-bw)/2, size.Height-bh)
	if !below {
		pos = fyne.NewPos(1, (size.Height-bh)/2)
	}
	k.bg.Move(pos)
	k.bg.Resize(fyne.NewSize(bw, bh))
	tw := fyne.MeasureText(k.text.Text, k.text.TextSize, k.text.TextStyle)
	k.text.Move(pos.AddXY((bw-tw.Width)/2, (bh-tw.Height)/2))
}

func (k *keyTipBadge) show(tip string) {
	k.text.Text = tip
	k.bg.FillColor = theme.Color(theme.ColorNameForeground)
	k.text.Color = theme.Color(theme.ColorNameBackground)
	k.bg.Hidden, k.text.Hidden = tip == "", tip == ""
	k.bg.Refresh()
	k.text.Refresh()
}

// ribbonRenderer is the tabs' and buttons' renderer.
type ribbonRenderer struct {
	objects []fyne.CanvasObject
	layout  func(fyne.Size)
	min     func() fyne.Size
	refresh func()
	size    fyne.Size
}

func (r *ribbonRenderer) Layout(s fyne.Size) {
	r.size = s
	r.layout(s)
}
func (r *ribbonRenderer) MinSize() fyne.Size           { return r.min() }
func (r *ribbonRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *ribbonRenderer) Destroy()                     {}
func (r *ribbonRenderer) Refresh() {
	r.refresh()
	r.layout(r.size)
}
