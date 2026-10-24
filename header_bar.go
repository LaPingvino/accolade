package main

import (
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

	newButton, openButton, saveButton *tooltip.Button
	undoButton, redoButton            *tooltip.Button
	findButton, previewButton         *tooltip.Button
	exportButton, preferencesButton   *tooltip.Button
	fullscreenButton                  *tooltip.Button
	titleLabel                        *widget.Label
	element, format                   *widget.Select
	elementBox                        *fyne.Container           // the list, or the buttons
	elementMarks                      *fyne.Container           // the buttons
	place                             *widget.Button            // inside, outside or both
	elementButtons                    map[string]*widget.Button // by element
	formatBox                         *fyne.Container           // the format's own buttons
	showing                           bool                      // the selects show, not choose
}

// The emphasis icons (Material Design, Apache 2.0).
func icon(name, path string) fyne.Resource {
	return theme.NewThemedResource(fyne.NewStaticResource(name+".svg",
		[]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="#000" d="`+path+`"/></svg>`)))
}

var (
	boldIcon      = icon("bold", "M15.6 10.79c.97-.67 1.65-1.77 1.65-2.79 0-2.26-1.75-4-4-4H7v14h7.04c2.09 0 3.71-1.7 3.71-3.79 0-1.52-.86-2.82-2.15-3.42zM10 6.5h3c.83 0 1.5.67 1.5 1.5s-.67 1.5-1.5 1.5h-3v-3zm3.5 9H10v-3h3.5c.83 0 1.5.67 1.5 1.5s-.67 1.5-1.5 1.5z")
	italicIcon    = icon("italic", "M10 4v3h2.21l-3.42 8H6v3h8v-3h-2.21l3.42-8H18V4z")
	underlineIcon = icon("underline", "M12 17c3.31 0 6-2.69 6-6V3h-2.5v8c0 1.93-1.57 3.5-3.5 3.5S8.5 12.93 8.5 11V3H6v8c0 3.31 2.69 6 6 6zm-7 2v2h14v-2H5z")
)

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
	hb := &HeaderBar{window: window}
	tips := window.tooltips
	button := func(icon fyne.Resource, tip string, action func()) *tooltip.Button {
		b := tooltip.NewButton(icon, tip, action, tips)
		b.Importance = widget.LowImportance
		return b
	}

	hb.newButton = button(theme.DocumentCreateIcon(), "New"+keys("N"), window.NewFile)
	hb.openButton = button(theme.FolderOpenIcon(), "Open"+keys("O"), window.OpenFile)
	hb.saveButton = button(theme.DocumentSaveIcon(), "Save"+keys("S"), window.reportErr(window.SaveFile))
	hb.undoButton = button(theme.ContentUndoIcon(), "Undo"+keys("Z"), window.undo)
	hb.redoButton = button(theme.ContentRedoIcon(), "Redo"+keys("Shift+Z"), window.redo)
	hb.findButton = button(theme.SearchIcon(), "Find and replace"+keys("F"), window.showFindReplace)
	hb.previewButton = button(theme.VisibilityIcon(), "Preview PDF"+keys("Shift+P"), window.previewPDF)
	hb.exportButton = button(theme.DownloadIcon(), "Export to PDF, HTML, FDX…"+keys("E"), window.showExportDialog)
	hb.fullscreenButton = button(theme.ViewFullScreenIcon(), "Full screen, without distractions"+keys("Shift+F"), window.toggleFullscreen)
	hb.preferencesButton = button(theme.SettingsIcon(), "Preferences"+keys(","), func() { window.app.showPreferences() })

	inEditor := func(f func()) func() {
		return func() {
			f()
			window.fyneWindow.Canvas().Focus(window.textEditor)
		}
	}
	emphasis := func(marker string) func() {
		return inEditor(func() { toggleEmphasis(window.textEditor, marker) })
	}
	bold := button(boldIcon, "Bold"+keys("B"), emphasis("**"))
	italic := button(italicIcon, "Italic"+keys("I"), emphasis("*"))
	underline := button(underlineIcon, "Underline"+keys("U"), emphasis("_"))
	hb.element = widget.NewSelect(toolbarElements, func(el string) {
		if !hb.showing && el != "" {
			inEditor(func() { setLineElement(window.textEditor, el, window.sceneStarts()) })()
		}
	})
	hb.element.PlaceHolder = "Element"
	fyne.SetAccessibleLabel(hb.element, "The line's element")
	hb.elementButtons = map[string]*widget.Button{}
	marks := container.NewHBox()
	for _, el := range toolbarElements {
		b := widget.NewButton(elementMarks[el], inEditor(func() { setLineElement(window.textEditor, el, window.sceneStarts()) }))
		b.Importance = widget.LowImportance
		fyne.SetAccessibleLabel(b, el)
		hb.elementButtons[el] = b
		marks.Add(b)
	}
	hb.elementMarks = marks
	hb.elementBox = container.NewStack()
	hb.showElementKind()
	toggle := button(theme.ViewRestoreIcon(), "Elements as a list or as buttons", func() {
		window.settings.SetBoolean("element-buttons", !window.settings.GetBoolean("element-buttons"))
		_ = window.settings.Save()
		hb.showElementKind()
	})
	hb.format = newScriptFormatSelect(nil, func(key string) {
		if !hb.showing {
			window.settings.SetString("script-format", key)
			_ = window.settings.Save()
			window.app.applySettingsToWindows()
		}
	})
	fyne.SetAccessibleLabel(hb.format, "Script format")
	hb.formatBox = container.NewHBox()
	hb.showFormats()

	hb.titleLabel = widget.NewLabel("Untitled")
	hb.titleLabel.Alignment = fyne.TextAlignCenter
	hb.titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	hb.titleLabel.Truncation = fyne.TextTruncateEllipsis

	// the editing row: what a line is and its emphasis, the format's own
	// buttons, and the format; both rows put what does not fit in a menu
	var character *widget.Button
	character = widget.NewButtonWithIcon("Character", theme.AccountIcon(), func() { window.pickCharacter(character) })
	character.Importance = widget.LowImportance
	var place *widget.Button
	place = widget.NewButtonWithIcon("INT./EXT.", theme.HomeIcon(), func() { window.pickPlace(place) })
	place.Importance = widget.LowImportance
	hb.place = place
	top := newOverflowRow(tips, []overflowItem{
		buttonItem(hb.newButton), buttonItem(hb.openButton), buttonItem(hb.saveButton), gapItem(),
		buttonItem(hb.undoButton), buttonItem(hb.redoButton),
		buttonItem(hb.findButton), buttonItem(hb.previewButton), buttonItem(hb.exportButton), gapItem(),
		buttonItem(hb.fullscreenButton), buttonItem(hb.preferencesButton),
	}, hb.titleLabel, 6)
	elements := overflowItem{obj: hb.elementBox, menu: func(fyne.CanvasObject) []*fyne.MenuItem {
		var items []*fyne.MenuItem
		for _, el := range toolbarElements {
			items = append(items, fyne.NewMenuItem(el, inEditor(func() { setLineElement(window.textEditor, el, window.sceneStarts()) })))
		}
		menu := fyne.NewMenuItem("Element", nil)
		menu.ChildMenu = fyne.NewMenu("", items...)
		return []*fyne.MenuItem{menu}
	}}
	picker := func(b *widget.Button, pick func(fyne.CanvasObject)) overflowItem {
		return overflowItem{obj: b, menu: func(anchor fyne.CanvasObject) []*fyne.MenuItem {
			item := fyne.NewMenuItem(b.Text, func() { pick(anchor) })
			item.Icon = b.Icon
			return []*fyne.MenuItem{item}
		}}
	}
	formatButtons := overflowItem{obj: hb.formatBox, menu: func(fyne.CanvasObject) []*fyne.MenuItem {
		var items []*fyne.MenuItem
		for _, o := range hb.formatBox.Objects {
			if b, ok := o.(*widget.Button); ok {
				items = append(items, fyne.NewMenuItem(b.Text, b.OnTapped))
			}
		}
		return items
	}}
	editing := newOverflowRow(tips, []overflowItem{
		buttonItem(toggle), elements, picker(place, window.pickPlace), picker(character, window.pickCharacter),
		buttonItem(bold), buttonItem(italic), buttonItem(underline), gapItem(), formatButtons,
	}, nil, 0)
	hb.container = container.NewVBox(top.Container, container.NewBorder(nil, nil, nil, hb.format, editing.Container))
	return hb
}

// gap is a little space between groups of buttons.
func gap() fyne.CanvasObject {
	return container.NewGridWrap(fyne.NewSize(theme.Padding(), 1))
}

// keys formats a shortcut for a tooltip: " (Ctrl+S)", or " (⌘S)" on macOS.
func keys(k string) string {
	if runtime.GOOS == "darwin" {
		return " (" + strings.ReplaceAll("⌘"+k, "Shift+", "⇧") + ")"
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
	key := hb.window.settings.GetString("script-format")
	hb.format.SetOptions(scriptFormatNames())
	selectScriptFormat(hb.format, key)
	w := hb.window
	hb.formatBox.RemoveAll()
	for _, fb := range formatButtons(key) {
		b := widget.NewButton(fb.Label, func() {
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
