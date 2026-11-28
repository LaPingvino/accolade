// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"runtime"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/tooltip"
)

// HeaderBar is the bar at the top of the window: the file actions and
// undo/redo, the document's name, then find, PDF preview, export and
// preferences, as icon buttons with tooltips; under them the line's
// element, emphasis, the script format's own buttons and the format.
type HeaderBar struct {
	window    *MainWindow
	container *fyne.Container

	buttons                    map[string]*tooltip.Button // the icon buttons, by command id
	titleLabel                 *scriptTitle               // the file's name; click to rename it
	element, format            *widget.Select
	elementBox                 *fyne.Container           // the list, or the buttons
	elementMarks               *fyne.Container           // the buttons
	place                      *widget.Button            // inside, outside or both
	elementButtons             map[string]*widget.Button // by element
	formatBox                  *fyne.Container           // the format's own buttons
	screenplayRow, markdownRow fyne.CanvasObject         // the editing row for each
	docRow                     fyne.CanvasObject         // Markdown's, for the source or notes in the main editor (swap.go)
	top                        *overflowRow              // the top row
	showing                    bool                      // the selects show, not choose
}

// elementMarks are the elements' marks in Fountain, as the buttons show
// them (dialogue has none: an opening quote).
var elementMarks = map[string]string{
	elementScene: ".", elementAction: "!", elementCharacter: "@", elementParenthetical: "()",
	elementDialogue: "“", elementTransition: ">", elementCentered: "><", elementLyrics: "~",
}

// showElementKind shows the elements as a list or as buttons (the
// element-buttons setting).
func (hb *HeaderBar) showElementKind() {
	var shown fyne.CanvasObject = hb.element
	if hb.window.settings.GetBoolean("element-buttons") {
		shown = hb.elementMarks
	}
	hb.elementBox.Objects = []fyne.CanvasObject{shown}
	hb.elementBox.Refresh()
	if hb.container != nil {
		hb.container.Refresh() // the row's widths again
	}
}

// toolbarElements are the elements the bar can make a line.
var toolbarElements = []string{elementScene, elementAction, elementCharacter, elementParenthetical, elementDialogue, elementTransition, elementCentered, elementLyrics}

func NewHeaderBar(window *MainWindow) *HeaderBar {
	hb := &HeaderBar{window: window, buttons: map[string]*tooltip.Button{}}
	tips := window.tooltips
	env := cmdEnv{browser: inBrowser}

	hb.element = widget.NewSelect(toolbarElements, func(el string) {
		if !hb.showing && el != "" {
			window.tapped(cmd(elementCommandID(el)), nil)()
		}
	})
	hb.element.PlaceHolder = "Element"
	fyne.SetAccessibleLabel(hb.element, "The line's element")
	hb.elementButtons = map[string]*widget.Button{}
	marks := container.NewHBox()
	for _, el := range toolbarElements {
		c := cmd(elementCommandID(el))
		b := widget.NewButton(c.text, window.tapped(c, nil))
		b.Importance = widget.LowImportance
		fyne.SetAccessibleLabel(b, c.label)
		hb.elementButtons[el] = b
		marks.Add(b)
	}
	hb.elementMarks = marks
	hb.elementBox = container.NewStack()
	hb.showElementKind()
	hb.format = newScriptFormatSelect(window.lex, nil, func(key string) {
		if !hb.showing {
			window.setScriptFormat(key)
			_ = window.settings.Save()
			window.app.applySettingsToWindows()
		}
	})
	fyne.SetAccessibleLabel(hb.format, "Script format")
	hb.formatBox = container.NewHBox()
	hb.showFormats()

	hb.titleLabel = newScriptTitle("Untitled", window.renameScript)
	hb.titleLabel.Alignment = fyne.TextAlignCenter
	hb.titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	hb.titleLabel.Truncation = fyne.TextTruncateEllipsis

	// the rows from the registry (commands.go); both put what does not fit
	// in a menu
	top := hb.newTop(env)
	editing := newOverflowRow(tips, hb.rowItems(commands().screenplayRow, env, false), nil, 0)
	hb.screenplayRow = container.NewBorder(nil, nil, nil, hb.format, editing.Container)
	hb.markdownRow = newOverflowRow(tips, hb.rowItems(commands().markdownRow, cmdEnv{markdown: true, browser: inBrowser}, false), nil, 0).Container
	// a screenplay's own source or notes: Markdown's row without Start a
	// Screenplay (the window has one)
	var docHomes []home
	for _, h := range commands().markdownRow {
		if h.id != "file.startScreenplay" {
			docHomes = append(docHomes, h)
		}
	}
	for len(docHomes) > 0 && docHomes[len(docHomes)-1].sep {
		docHomes = docHomes[:len(docHomes)-1]
	}
	hb.docRow = newOverflowRow(tips, hb.rowItems(docHomes, cmdEnv{markdown: true, browser: inBrowser}, false), nil, 0).Container
	hb.container = container.NewVBox(top.Container, hb.screenplayRow)
	return hb
}

// newTop makes the top row (toprow.go: as the window's changes have it),
// the name keeping its room before the less used icons.
func (hb *HeaderBar) newTop(env cmdEnv) *overflowRow {
	w := hb.window
	hb.buttons = map[string]*tooltip.Button{}
	homes := w.toolbarHomes()
	at := 6 // the name after the file's and undo's buttons
	if i := slices.IndexFunc(homes, func(h home) bool { return h.id == "edit.redo" }); i >= 0 {
		at = i + 1
	}
	items := hb.rowItems(homes, env, true)
	top := newOverflowRow(w.tooltips, items, hb.titleLabel, min(at, len(items)))
	top.keepFill = topTitleRoom
	top.fillMenu = func() []*fyne.MenuItem { return []*fyne.MenuItem{w.renameItem()} }
	ids := make([]string, len(items))
	for i, h := range homes {
		if i < len(ids) && !h.sep {
			ids[i] = h.id
		}
	}
	rankEnv := env
	if w.textEditor != nil {
		rankEnv = w.cmdEnv()
	}
	ranks := make([]float64, len(ids))
	for i, id := range ids {
		ranks[i] = w.topRank(id, rankEnv) + float64(len(ids)-i)/100 // the later first among equals
	}
	top.rank = func(i int) float64 { return ranks[i] }
	hb.top = top
	return top
}

// rebuildTop makes the top row again (its icons changed, or the window
// turned narrow or wide), as the document and the role have it.
func (hb *HeaderBar) rebuildTop() {
	w := hb.window
	env := cmdEnv{markdown: w.markdown, browser: inBrowser}
	top := hb.newTop(env)
	hb.followTop(env)
	if w.textEditor != nil {
		hb.showRole(w.cmdEnv())
	}
	if hb.container != nil {
		hb.container.Objects[0] = top.Container
		hb.container.Refresh()
	}
}

// rowItems are a header row's items for homes: a command with an icon
// and no text is an icon button with a tooltip (the source pane's stays
// however narrow the row), one with text a text button, which opens its
// picker next to it; "elements" and "formatButtons" are the element list
// and the script format's own buttons. The top row's buttons are kept
// (tracked) to follow the document's kind (setMarkdown).
func (hb *HeaderBar) rowItems(homes []home, env cmdEnv, tracked bool) []overflowItem {
	w := hb.window
	var items []overflowItem
	for _, h := range homes {
		switch {
		case h.sep:
			items = append(items, gapItem())
			continue
		case h.id == "elements":
			items = append(items, overflowItem{obj: hb.elementBox, menu: func(fyne.CanvasObject) []*fyne.MenuItem {
				var items []*fyne.MenuItem
				for _, el := range toolbarElements {
					items = append(items, fyne.NewMenuItem(el, w.tapped(cmd(elementCommandID(el)), nil)))
				}
				menu := fyne.NewMenuItem("Element", nil)
				menu.ChildMenu = fyne.NewMenu("", items...)
				return []*fyne.MenuItem{menu}
			}})
			continue
		case h.id == "formatButtons":
			items = append(items, overflowItem{obj: hb.formatBox, menu: func(fyne.CanvasObject) []*fyne.MenuItem {
				var items []*fyne.MenuItem
				for _, o := range hb.formatBox.Objects {
					if b, ok := o.(*widget.Button); ok {
						items = append(items, fyne.NewMenuItem(b.Text, b.OnTapped))
					}
				}
				return items
			}})
			continue
		}
		c := cmd(h.id)
		var icon fyne.Resource
		if c.icon != nil {
			icon = c.icon()
		}
		if c.text == "" || tracked { // an icon button (a mark's on the top row: an element added there)
			b := tooltip.NewButton(icon, c.tipIn(env), nil, w.tooltips)
			b.OnTapped = w.tapped(c, b)
			b.Importance = widget.LowImportance
			if c.text != "" {
				b.Text = c.text
			}
			if tracked {
				hb.buttons[c.id] = b
				id := c.id
				b.Secondary = func(pos fyne.Position) { w.showTopRowMenu(id, b, pos) }
				if c.id == "app.palette" { // the search field, wide (toprow.go)
					s := w.newTopSearch(b, w.narrow)
					it := buttonItem(b)
					it.obj = s.Container
					items = append(items, it)
					continue
				}
			}
			if c.id == "view.source" {
				items = append(items, keptItem(b))
			} else {
				items = append(items, buttonItem(b))
			}
			continue
		}
		b := widget.NewButtonWithIcon(c.text, icon, nil)
		b.OnTapped = w.tapped(c, b)
		b.Importance = widget.LowImportance
		if c.tip != "" { // its accessible name: the tip, else the label
			fyne.SetAccessibleLabel(b, c.tipIn(env))
		} else if c.labelIn(env) != c.text {
			fyne.SetAccessibleLabel(b, c.labelIn(env))
		}
		if c.id == "write.place" {
			hb.place = b
		}
		label := c.labelIn(env) // in the menu: a picker as its button says, opening by the menu
		if c.pick != nil {
			label = c.text
		}
		items = append(items, overflowItem{obj: b, menu: func(anchor fyne.CanvasObject) []*fyne.MenuItem {
			item := fyne.NewMenuItem(label, w.tapped(c, anchor))
			item.Icon = b.Icon
			return []*fyne.MenuItem{item}
		}})
	}
	return items
}

// button is the top row's button of the command with id (nil if none).
func (hb *HeaderBar) button(id string) *tooltip.Button { return hb.buttons[id] }

// setMarkdown shows the bar for a Markdown document or a screenplay: the
// editing row, the preview's tip; the export is a screenplay's.
func (hb *HeaderBar) setMarkdown(on bool) {
	hb.followTop(cmdEnv{markdown: on, browser: inBrowser})
	hb.showEditingRow(on, false)
}

// followTop enables the top row's buttons for the document's kind, with
// its tips.
func (hb *HeaderBar) followTop(env cmdEnv) {
	for id, b := range hb.buttons { // a screenplay's disabled, the tips its own
		c := cmd(id)
		if c.applies(env) {
			b.Enable()
		} else {
			b.Disable()
		}
		b.Tip = c.tipIn(env)
	}
}

// showEditingRow shows the editing row for what the main editor holds:
// Markdown's (a Markdown file, the source or notes in the middle: middle.go)
// or the screenplay's.
func (hb *HeaderBar) showEditingRow(markdown, doc bool) {
	row := hb.screenplayRow
	switch {
	case doc:
		row = hb.docRow
	case markdown:
		row = hb.markdownRow
	}
	if hb.container.Objects[1] != row {
		hb.container.Objects[1] = row
		hb.container.Refresh()
	}
}

// showRole puts the top row's buttons of commands the role puts away
// away (role.go: the Production pane's in Writer), or back.
func (hb *HeaderBar) showRole(env cmdEnv) {
	if hb.top == nil {
		return
	}
	for id, b := range hb.buttons {
		hb.top.setOff(b, !cmd(id).role.in(env))
	}
}

// gap is a little space between groups of buttons.
func gap() fyne.CanvasObject {
	return container.NewGridWrap(fyne.NewSize(theme.Padding(), 1))
}

// macKeyLabels: shortcuts are written the Mac's way (⌘S, not Ctrl+S);
// a variable so the tests can draw the Mac's labels anywhere.
var macKeyLabels = runtime.GOOS == "darwin"

// keys formats a shortcut for a tooltip: " (Ctrl+S)", or " (⌘S)" on macOS.
func keys(k string) string {
	if macKeyLabels {
		return " (" + strings.NewReplacer("Shift+", "⇧", "Alt+", "⌥").Replace("⌘"+k) + ")"
	}
	return " (Ctrl+" + k + ")"
}

// showElement shows the element of the cursor's line ("" for none the
// bar makes).
func (hb *HeaderBar) showElement(el string) {
	hb.showing = true
	defer func() { hb.showing = false }()
	for e, b := range hb.elementButtons { // the line's own stands out
		if e == el {
			b.Importance = widget.HighImportance
		} else {
			b.Importance = widget.LowImportance
		}
		b.Refresh()
	}
	for _, e := range toolbarElements {
		if e == el {
			hb.element.SetSelected(el)
			return
		}
	}
	hb.element.ClearSelected()
}

// showFormats shows the formats, with the one in the settings chosen.
func (hb *HeaderBar) showFormats() {
	hb.showing = true
	defer func() { hb.showing = false }()
	w := hb.window
	key, conf := w.scriptFormatKey(), w.lex()
	hb.format.SetOptions(conf.formatNames())
	conf.selectFormat(hb.format, key)
	hb.formatBox.RemoveAll()
	for _, fb := range conf.formatButtons(key) {
		b := widget.NewButton(fb.Label, func() {
			if w.scriptAway(fb.Label) { // another middle tab in front (middle.go)
				return
			}
			w.textEditor.Edit(func(b *buffer.Buffer) { insertButton(b, fb) })
			w.fyneWindow.Canvas().Focus(w.textEditor)
		})
		b.Importance = widget.LowImportance
		hb.formatBox.Add(b)
	}
}

// SetTitle shows the document's name in the middle of the bar.
func (hb *HeaderBar) SetTitle(title string) {
	hb.titleLabel.SetText(title)
}

// SetVisible hides or shows the bar (for focus mode).
func (hb *HeaderBar) SetVisible(visible bool) {
	if visible {
		hb.container.Show()
	} else {
		hb.container.Hide()
	}
}

// heldModifiers are the keys held down, as for an Alt+click (none
// without a desktop keyboard).
func heldModifiers() fyne.KeyModifier {
	if d, ok := fyne.CurrentApp().Driver().(desktop.Driver); ok {
		return d.CurrentKeyModifiers()
	}
	return 0
}
