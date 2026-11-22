package main

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/lexington/lex"
)

// The breakdown in the window (breakdown.go has the file and the
// anchors): tagging the words selected (Production > Tag Selection…, the
// editor's right click or long press, the Production pane's Tag button),
// Untag, the tagged words drawn in their category's colour and
// underlined (Production > Show Tagged Words), found again 400 ms after
// typing pauses (as the revision marks are: never per key).

// breakdownState is the script's breakdown and its tags as found in a
// text.
type breakdownState struct {
	key  string // what the file was read for (breakdownKey)
	file breakdownFile
	err  error

	text   string // the text the tags were found in
	bt     *bdText
	placed []placed
	inks   map[int][]editor.Ink // in the categories' own colours

	shown   map[int][]editor.Ink // as drawn: mixed with the theme's text colour
	shownFg color.Color          // the text colour they are mixed with
}

// breakdownHiddenPref hides the tagged words' colours in the editor.
const breakdownHiddenPref = "breakdown-tags-hidden"

// breakdownKey is what the file is read again for: another script, or an
// import (breakdownGen).
func (w *MainWindow) breakdownKey() string {
	return w.revisionKey() + fmt.Sprintf("|%d", w.breakdownGen)
}

// breakdownNow is the script's breakdown, read again if need be.
func (w *MainWindow) breakdownNow() *breakdownFile {
	if key := w.breakdownKey(); w.breakdown.key != key {
		f, err := readBreakdown(w.extrasHome())
		w.breakdown = breakdownState{key: key, file: f, err: err}
	}
	return &w.breakdown.file
}

// scheduleBreakdown finds the tags again once typing pauses.
func (w *MainWindow) scheduleBreakdown() {
	if w.markdown || w.textEditor == nil {
		return
	}
	w.debounce(&w.breakdownTimer, revisionPause, w.updateBreakdown)
}

// updateBreakdown finds the tags in the text (in the background) and
// shows them.
func (w *MainWindow) updateBreakdown() {
	if w.markdown || w.textEditor == nil {
		return
	}
	w.followSceneNumbers() // an Undo of Renumber, say
	f := w.breakdownNow()
	text, starts, key := w.textEditor.Text(), w.sceneStarts(), w.breakdown.key
	if len(f.Tags) == 0 && len(f.Done) == 0 && !w.productionShowsBreakdown() {
		if w.breakdown.inks != nil {
			w.breakdown.inks = nil
			w.textEditor.RedrawMarks()
		}
		w.breakdown.text, w.breakdown.bt, w.breakdown.placed = text, nil, nil
		return
	}
	file := *f
	file.Tags = append([]bdTag(nil), f.Tags...)
	w.breakdownRun++
	run := w.breakdownRun
	type found struct {
		bt     *bdText
		placed []placed
		inks   map[int][]editor.Ink
	}
	inBackground(func() found {
		bt := readBdText(text, starts)
		ps := bt.place(&file)
		return found{bt, ps, inksOf(bt, &file, ps, func(c lex.TagCategory) color.Color {
			if r, g, b, ok := parseHexRGB(c.Color); ok {
				return color.NRGBA{R: r, G: g, B: b, A: 0xff}
			}
			return nil
		})}
	}, func(r found) {
		if run != w.breakdownRun || key != w.breakdown.key {
			return // a later text's are coming
		}
		w.breakdown.text, w.breakdown.bt, w.breakdown.placed, w.breakdown.inks = text, r.bt, r.placed, r.inks
		w.breakdown.shown = nil
		w.textEditor.RedrawMarks()
		w.productionFollow()
	})
}

// breakdownNowFound is the breakdown found in the text as it is now
// (found at once if the last finding is of another text).
func (w *MainWindow) breakdownNowFound() (*breakdownFile, *bdText, []placed) {
	text := w.textEditor.Text()
	if w.breakdown.bt == nil || !sameString(w.breakdown.text, text) {
		w.followSceneNumbers()
	}
	f := w.breakdownNow()
	if w.breakdown.bt == nil || !sameString(w.breakdown.text, text) {
		bt := readBdText(text, w.sceneStarts())
		w.breakdown.text, w.breakdown.bt, w.breakdown.placed = text, bt, bt.place(f)
	}
	return f, w.breakdown.bt, w.breakdown.placed
}

// breakdownInksShown are the editor's inks (its Inks).
func (w *MainWindow) breakdownInksShown() map[int][]editor.Ink {
	// production's colours, put away in Writer (role.go); the tags are kept
	if w.settings != nil && w.settings.GetBoolean(breakdownHiddenPref) || w.breakdown.inks == nil || w.writerView() {
		return nil
	}
	fg := w.textEditor.Theme().Color(theme.ColorNameForeground, fyne.CurrentApp().Settings().ThemeVariant())
	if w.breakdown.shown == nil || w.breakdown.shownFg != fg { // mixed again only for another theme
		shown := make(map[int][]editor.Ink, len(w.breakdown.inks))
		for line, ks := range w.breakdown.inks {
			out := make([]editor.Ink, len(ks))
			for i, k := range ks {
				out[i] = k
				if k.Color == nil {
					out[i].Color = fg
				} else {
					out[i].Color = mixInk(k.Color, fg)
				}
			}
			shown[line] = out
		}
		w.breakdown.shown, w.breakdown.shownFg = shown, fg
	}
	return w.breakdown.shown
}

// toggleBreakdownTags is Production > Show Tagged Words.
func (w *MainWindow) toggleBreakdownTags() {
	if w.settings == nil {
		return
	}
	w.settings.SetBoolean(breakdownHiddenPref, !w.settings.GetBoolean(breakdownHiddenPref))
	_ = w.settings.Save()
	w.updateBreakdown()
	w.textEditor.RedrawMarks()
	w.refreshMenu()
}

// saveBreakdown writes the breakdown where the script's extras are, and
// shows it again.
func (w *MainWindow) saveBreakdown() {
	f := w.breakdownNow()
	if err := w.breakdown.err; err != nil { // never written over what could not be read
		dialog.ShowError(fmt.Errorf("the breakdown could not be read, so it is not written over (%v): "+
			"mend or move %s first", err, breakdownName), w.fyneWindow)
		w.breakdown.key = "" // read again next time
		return
	}
	f.Version = 1
	if err := w.writeExtra(breakdownName, f); err != nil {
		dialog.ShowError(fmt.Errorf("could not keep the breakdown: %v", err), w.fyneWindow)
	}
	w.breakdown.key = w.breakdownKey() // a kept copy made for it: the same breakdown
	w.breakdown.bt = nil
	w.schedule.measuredFor = "" // the strips' cast measured again (a silent one tagged)
	w.updateBreakdown()
	w.productionFollow()
}

// tagSelectionAs tags the words selected (or at the cursor) as an
// element of category.
func (w *MainWindow) tagSelectionAs(category string) {
	if w.markdown || w.textEditor == nil {
		return
	}
	f, bt, _ := w.breakdownNowFound()
	start, end := w.textEditor.Buffer().Selection()
	i, j := bt.wordRange(start, end)
	if i == j && bt.inNote(start, end) {
		w.setStatus("Tags go on the script's words, not a note's or a source link's")
		return
	}
	if i == j {
		w.setStatus("Select the words to tag (or put the cursor in a word)")
		return
	}
	if bt.sceneOf[i] < 0 {
		w.setStatus("Tags go in a scene: these words are before the first")
		return
	}
	if l1, _ := bt.lineCol(bt.words[i].start); func() bool { l2, _ := bt.lineCol(bt.words[j-1].start); return l1 != l2 }() {
		w.setStatus("Tag words on one line")
		return
	}
	e, added := f.tagWords(bt, i, j, category)
	w.lastTagCategory = e.Category
	w.saveBreakdown()
	scene := bt.scenes[bt.sceneOf[i]].label
	if added {
		w.setStatus(fmt.Sprintf("Tagged %q as %s in scene %s", e.Name, e.Category, scene))
	} else {
		w.setStatus(fmt.Sprintf("%q is tagged as %s here already", e.Name, e.Category))
	}
}

// untagSelection takes away the tags on the words selected (or at the
// cursor).
func (w *MainWindow) untagSelection() {
	if w.scriptAway("Untag") { // the script is in the side panes (swap.go)
		return
	}
	if w.markdown || w.textEditor == nil {
		return
	}
	f, bt, _ := w.breakdownNowFound()
	start, end := w.textEditor.Buffer().Selection()
	if n := f.untag(bt, start, end); n > 0 {
		w.saveBreakdown()
		w.setStatus("Untagged " + plural(n, "tag"))
		return
	}
	w.setStatus("No tag here")
}

// tagsAt are the tags placed over the cursor or selection.
func (w *MainWindow) tagsAt() []placed {
	_, _, ps := w.breakdownNowFound()
	start, end := w.textEditor.Buffer().Selection()
	var out []placed
	for _, p := range ps {
		if !p.lost && ((start == end && p.start <= start && start <= p.end) || (start != end && p.end > start && p.start < end)) {
			out = append(out, p)
		}
	}
	return out
}

// showTagMenu is the editor's right click (a long press): tag as the
// category last used, tag as another, untag.
func (w *MainWindow) showTagMenu(at fyne.Position) {
	if w.markdown {
		return
	}
	items := w.menuItems(commands().context, w.cmdEnv())
	widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", items...), w.fyneWindow.Canvas(), at)
}

// tagAsLastItems are the context menu's Tag as (the category tagged
// last), if one was.
func (w *MainWindow) tagAsLastItems() []*fyne.MenuItem {
	c := w.lastTagCategory
	if c == "" {
		return nil
	}
	return []*fyne.MenuItem{fyne.NewMenuItem("Tag as "+c, func() { w.tagSelectionAs(c) })}
}

// untagHereItems are the context menu's Untag, one for each tag at the
// cursor or in the selection.
func (w *MainWindow) untagHereItems() []*fyne.MenuItem {
	var items []*fyne.MenuItem
	for _, p := range w.tagsAt() {
		items = append(items, fyne.NewMenuItem(fmt.Sprintf("Untag %q (%s)", p.element.Name, p.element.Category), func() {
			f := w.breakdownNow()
			if f.untag(w.breakdown.bt, p.start, p.end) > 0 {
				w.saveBreakdown()
			}
		}))
	}
	return items
}

// clipboard is the app's clipboard.
func (w *MainWindow) clipboard() fyne.Clipboard {
	if a := fyne.CurrentApp(); a != nil {
		return a.Clipboard()
	}
	return nil
}

// ownCategoryColours are the colours new categories get, in turn.
var ownCategoryColours = []string{"#C71585", "#2F4F4F", "#FF8C00", "#4B0082", "#556B2F", "#8B008B", "#00688B"}

// tagPicker is the category picker of Tag Selection…: a filter (or a new
// category's name) and the categories, the last used first. A tap tags.
type tagPicker struct {
	w      *MainWindow
	d      dialog.Dialog
	filter *widget.Entry
	list   *fyne.Container
	words  string
}

// showTagPicker is Production > Tag Selection….
func (w *MainWindow) showTagPicker() {
	if w.scriptAway("Tag Selection") { // the script is in the side panes (swap.go)
		return
	}
	if w.markdown || w.textEditor == nil {
		return
	}
	_, bt, _ := w.breakdownNowFound()
	start, end := w.textEditor.Buffer().Selection()
	i, j := bt.wordRange(start, end)
	if i == j {
		w.setStatus("Select the words to tag (or put the cursor in a word)")
		return
	}
	tp := &tagPicker{w: w, words: string(bt.runes[bt.words[i].start:bt.words[j-1].end])}
	tp.filter = widget.NewEntry()
	tp.filter.SetPlaceHolder("Filter, or a new category's name")
	tp.filter.OnChanged = func(string) { tp.fill() }
	tp.filter.OnSubmitted = func(string) { tp.pickFirst() }
	tp.list = container.NewVBox()
	tp.fill()
	head := widget.NewLabel(fmt.Sprintf("Tag %q as", strings.Join(strings.Fields(tp.words), " ")))
	head.Wrapping = fyne.TextWrapWord
	content := container.NewBorder(container.NewVBox(head, tp.filter), nil, nil, nil, container.NewVScroll(tp.list))
	tp.d = dialog.NewCustom("Tag Selection", "Cancel", content, w.fyneWindow)
	tp.d.Resize(fitDialog(w.fyneWindow, 380+dialogMargin, 560+dialogMargin))
	tp.d.Show()
	w.lastPicker = tp
	if !w.narrow { // on a phone the keyboard would cover the list
		w.fyneWindow.Canvas().Focus(tp.filter)
	}
}

// fill lists the categories that match the filter (and the new one it
// names).
func (tp *tagPicker) fill() {
	f := tp.w.breakdownNow()
	q := strings.ToLower(strings.TrimSpace(tp.filter.Text))
	var objs []fyne.CanvasObject
	cats := f.categories()
	if last := tp.w.lastTagCategory; last != "" { // the last used first
		for k, c := range cats {
			if c.Name == last {
				cats = append([]lex.TagCategory{c}, append(cats[:k:k], cats[k+1:]...)...)
				break
			}
		}
	}
	exact := false
	for _, c := range cats {
		if q != "" && !strings.Contains(strings.ToLower(c.Name), q) {
			continue
		}
		exact = exact || strings.EqualFold(c.Name, q)
		objs = append(objs, tp.button(c.Name, c.Color, func() { tp.pick(c.Name) }))
	}
	if q != "" && !exact {
		name := strings.TrimSpace(tp.filter.Text)
		objs = append(objs, tp.button("New category “"+name+"”", "", func() { tp.pick(name) })) // made by tagWords
	}
	tp.list.Objects = objs
	tp.list.Refresh()
}

// button is a category's row: its colour and its name.
func (tp *tagPicker) button(name, col string, tap func()) fyne.CanvasObject {
	b := widget.NewButton(name, tap)
	b.Alignment = widget.ButtonAlignLeading
	sw := canvas.NewRectangle(color.Transparent)
	if r, g, bl, ok := parseHexRGB(col); ok {
		sw.FillColor = color.NRGBA{R: r, G: g, B: bl, A: 0xff}
	}
	sw.SetMinSize(fyne.NewSize(6, 0))
	return container.NewBorder(nil, nil, sw, nil, b)
}

// pickFirst is Enter: the first category listed.
func (tp *tagPicker) pickFirst() {
	if len(tp.list.Objects) == 0 {
		return
	}
	if b, ok := tp.list.Objects[0].(*fyne.Container).Objects[0].(*widget.Button); ok {
		b.OnTapped()
	}
}

func (tp *tagPicker) pick(category string) {
	tp.d.Hide()
	tp.w.tagSelectionAs(category)
}
