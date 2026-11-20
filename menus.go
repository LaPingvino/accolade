package main

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// Keyboard shortcuts live on the main menu: while the editor has focus,
// Fyne hands shortcuts to the focused widget and only main menu shortcuts
// get a look first, so shortcuts added to the canvas would never fire.

// shortcut builds a Ctrl+key (Cmd+key on macOS) shortcut; extra
// modifiers such as Shift are added on top.
func shortcut(key fyne.KeyName, extra fyne.KeyModifier) fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: key, Modifier: fyne.KeyModifierShortcutDefault | extra}
}

func menuItem(label string, sc fyne.Shortcut, action func()) *fyne.MenuItem {
	item := fyne.NewMenuItem(label, action)
	item.Shortcut = sc
	return item
}

// reportErr adapts an action that can fail into a menu/button callback
// that shows the error instead of dropping it.
func (w *MainWindow) reportErr(action func() error) func() {
	return func() {
		if err := action(); err != nil {
			dialog.ShowError(err, w.fyneWindow)
		}
	}
}

func (w *MainWindow) buildMainMenu() *fyne.MainMenu {
	file := fyne.NewMenu("File",
		menuItem("New", shortcut(fyne.KeyN, 0), w.NewFile),
		menuItem("New Empty", shortcut(fyne.KeyN, fyne.KeyModifierAlt), w.NewEmptyFile),
		fyne.NewMenuItem("New Markdown Document", w.NewMarkdownFile),
		menuItem(openLabel, shortcut(fyne.KeyO, 0), w.OpenFile),
		w.recentMenu(),
		fyne.NewMenuItem("Import as Markdown…", w.ImportDocument),
		fyne.NewMenuItem("Choose Source…", w.chooseSource),
		fyne.NewMenuItem(startScreenplayLabel, w.startScreenplayFromSource),
		fyne.NewMenuItem("Save Source As…", w.saveSourceAs),
		fyne.NewMenuItemSeparator(),
		menuItem(saveLabel, shortcut(fyne.KeyS, 0), w.reportErr(w.SaveFile)),
		menuItem("Save As…", shortcut(fyne.KeyS, fyne.KeyModifierShift), w.reportErr(w.SaveFileAs)),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Edit Together…", w.shareDialog),
		fyne.NewMenuItem("Join a Shared Script…", w.joinDialog),
		fyne.NewMenuItemSeparator(),
		menuItem("Title Page…", shortcut(fyne.KeyT, fyne.KeyModifierShift), w.showTitlePageDialog),
		menuItem("Export…", shortcut(fyne.KeyE, 0), w.showExportDialog),
		menuItem("Print…", shortcut(fyne.KeyP, 0), w.print),
		fyne.NewMenuItemSeparator(),
		menuItem("Preferences…", shortcut(fyne.KeyComma, 0), func() { w.app.showPreferences() }),
		// Fyne appends Quit (Ctrl+Q) to the first menu itself.
	)
	if !inBrowser { // the browser has one window
		newWindow := menuItem("New Window", shortcut(fyne.KeyN, fyne.KeyModifierShift), func() { w.app.newScriptWindow(false) })
		newEmptyWindow := menuItem("New Empty Window", shortcut(fyne.KeyN, fyne.KeyModifierShift|fyne.KeyModifierAlt), func() { w.app.newScriptWindow(true) })
		file.Items = append([]*fyne.MenuItem{file.Items[0], file.Items[1], newWindow, newEmptyWindow}, file.Items[2:]...)
	}
	if hasPandoc() { // more formats to import, with pandoc
		for i, it := range file.Items {
			if it.Label == "Import as Markdown…" {
				file.Items = slices.Insert(file.Items, i+1, fyne.NewMenuItem("Import with pandoc…", w.ImportWithPandoc))
				break
			}
		}
	}
	if w.markdown { // the title page and the source are a screenplay's
		file.Items = withoutItems(file.Items, "Title Page…", "Choose Source…", "Save Source As…")
	} else { // a Markdown document is a source to start from
		file.Items = withoutItems(file.Items, startScreenplayLabel)
	}

	edit := fyne.NewMenu("Edit",
		// Ctrl+Z / Ctrl+Y reach the focused text field directly; the menu adds
		// the Ctrl+Shift+Z redo that Fyne does not map itself.
		fyne.NewMenuItem("Undo", w.undo),
		menuItem("Redo", shortcut(fyne.KeyZ, fyne.KeyModifierShift), w.redo),
		fyne.NewMenuItemSeparator(),
		menuItem("Find…", shortcut(fyne.KeyF, 0), w.showFind),
		menuItem("Replace…", shortcut(fyne.KeyH, 0), w.showFindReplace),
		menuItem("Find Next", shortcut(fyne.KeyG, 0), w.findNext),
		menuItem("Find Previous", shortcut(fyne.KeyG, fyne.KeyModifierShift), w.findPrevious),
	)

	// the Format menu acts on the script only while it is being written
	// in: its shortcuts also fire with the focus in the notes or search
	inEditor := func(f func()) func() {
		return func() {
			if f := w.fyneWindow.Canvas().Focused(); f != nil && f != fyne.Focusable(w.textEditor) {
				return
			}
			f()
		}
	}
	el := func(element string) func() {
		return inEditor(func() { setLineElement(w.textEditor, element, w.sceneStarts()) })
	}
	alt := func(key fyne.KeyName) fyne.Shortcut {
		return &desktop.CustomShortcut{KeyName: key, Modifier: fyne.KeyModifierAlt}
	}
	format := fyne.NewMenu("Format",
		menuItem("Scene Heading", shortcut(fyne.Key1, 0), el(elementScene)),
		menuItem("Action", shortcut(fyne.Key2, 0), el(elementAction)),
		menuItem("Character", shortcut(fyne.Key3, 0), el(elementCharacter)),
		menuItem("Parenthetical", shortcut(fyne.Key4, 0), el(elementParenthetical)),
		menuItem("Dialogue", shortcut(fyne.Key5, 0), el(elementDialogue)),
		menuItem("Transition", shortcut(fyne.Key6, 0), el(elementTransition)),
		menuItem("Centered", shortcut(fyne.Key7, 0), el(elementCentered)),
		menuItem("Lyrics", shortcut(fyne.Key8, 0), el(elementLyrics)),
		menuItem("Right to Left Line", nil, inEditor(func() { toggleDirection(w.textEditor) })),
		menuItem("Script Language…", nil, w.pickLanguage),
		menuItem("Scene or Section Color…", shortcut(fyne.KeyL, fyne.KeyModifierShift), func() { w.pickColorHere(nil) }),
		fyne.NewMenuItemSeparator(),
		menuItem("Bold", shortcut(fyne.KeyB, 0), inEditor(func() { toggleEmphasis(w.textEditor, "**") })),
		menuItem("Italic", shortcut(fyne.KeyI, 0), inEditor(func() { toggleEmphasis(w.textEditor, "*") })),
		menuItem("Underline", shortcut(fyne.KeyU, 0), inEditor(func() { toggleEmphasis(w.textEditor, "_") })),
		fyne.NewMenuItemSeparator(),
		menuItem("Move Line Up", alt(fyne.KeyUp), inEditor(func() { moveLines(w.textEditor, -1) })),
		menuItem("Move Line Down", alt(fyne.KeyDown), inEditor(func() { moveLines(w.textEditor, 1) })),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Link to Source Passage", w.linkToSource),
		menuItem("Next Source Passage", shortcut(fyne.KeyJ, fyne.KeyModifierAlt), w.nextSourcePassage),
		fyne.NewMenuItem("Mark Source Passage as Cut", w.toggleCutHere),
		fyne.NewMenuItem("Source Headings as Sections", w.sectionsFromSourceHere),
		fyne.NewMenuItem("Suggest Links to the Source", w.suggestSourceLinks),
		w.sourceInsideItem(),
	)
	if w.markdown { // a screenplay's elements, language, colours and source
		format.Items = withoutItems(format.Items, "Link to Source Passage", "Next Source Passage", "Mark Source Passage as Cut", "Source Headings as Sections", "Suggest Links to the Source", "Keep Source in the Script", "Scene Heading", "Action", "Character", "Parenthetical",
			"Dialogue", "Transition", "Centered", "Lyrics", "Script Language…", "Scene or Section Color…", "Underline")
	}

	view := fyne.NewMenu("View",
		menuItem("Preview PDF", shortcut(fyne.KeyP, fyne.KeyModifierShift), w.preview),
		menuItem("Outline", shortcut(fyne.KeyO, fyne.KeyModifierShift), w.toggleOutline),
		menuItem("Cards", shortcut(fyne.KeyD, fyne.KeyModifierShift), w.toggleCards),
		menuItem("Fold or Unfold Here", shortcut(fyne.KeyPeriod, fyne.KeyModifierShift), func() { w.fold(false, true) }),
		menuItem("Fold All (Sections Only)", nil, func() { w.foldOverview(w.outlineItems(w.textEditor.Text())) }),
		menuItem("Headings Only", nil, w.foldHeadings),
		menuItem("Unfold All", nil, func() { w.foldAll(true) }),
		menuItem("Notes", shortcut(fyne.KeyK, fyne.KeyModifierShift), w.toggleNotes),
		fyne.NewMenuItem("Source", w.toggleSource),
		menuItem("Changes", shortcut(fyne.KeyH, fyne.KeyModifierShift), w.toggleChanges),
		fyne.NewMenuItem("Statistics…", w.showStatistics),
		menuItem("Fullscreen", shortcut(fyne.KeyF, fyne.KeyModifierShift), w.toggleFullscreen),
		fyne.NewMenuItemSeparator(),
		w.documentTypeMenu(),
	)
	if w.markdown {
		view.Items[0].Label = "Preview"
		view.Items = withoutItems(view.Items, "Cards", "Statistics…", "Source") // a screenplay's
	}

	// a shooting script's (docs/PRODUCTION.md; production.go)
	production := fyne.NewMenu("Production",
		fyne.NewMenuItem("Lock Scene Numbers", w.lockSceneNumbers),
		fyne.NewMenuItem("Unlock Scene Numbers…", w.unlockSceneNumbers),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Omit Scene", w.omitScene),
		fyne.NewMenuItem("Restore Scene", w.restoreScene),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Reports…", w.showReports),
	)

	help := fyne.NewMenu("Help",
		fyne.NewMenuItem("About Accolade", func() { w.app.showAbout() }),
	)

	if w.markdown {
		return fyne.NewMainMenu(file, edit, format, view, help)
	}
	return fyne.NewMainMenu(file, edit, format, production, view, help)
}

// withoutItems is a menu's items without those labelled so.
func withoutItems(items []*fyne.MenuItem, labels ...string) []*fyne.MenuItem {
	var out []*fyne.MenuItem
	for _, it := range items {
		if !slices.Contains(labels, it.Label) {
			out = append(out, it)
		}
	}
	return out
}

// print prints the document: a screenplay as Lexington lays it out, a
// Markdown document as a web page.
func (w *MainWindow) print() {
	if w.markdown {
		w.printMarkdown()
		return
	}
	w.printScript()
}

// escEntry is the entry used in the search bar: Escape closes the bar
// and gives focus back to the editor.
type escEntry struct {
	widget.Entry
	onEscape func()
}

func newEscEntry(onEscape func()) *escEntry {
	e := &escEntry{onEscape: onEscape}
	e.ExtendBaseWidget(e)
	return e
}

func (e *escEntry) TypedKey(key *fyne.KeyEvent) {
	if key.Name == fyne.KeyEscape && e.onEscape != nil {
		e.onEscape()
		return
	}
	e.Entry.TypedKey(key)
}

type undoer interface {
	Undo()
	Redo()
}

// editTarget is the text field undo/redo should act on: the focused one
// (the editor or a search field), falling back to the editor.
func (w *MainWindow) editTarget() undoer {
	if u, ok := w.fyneWindow.Canvas().Focused().(undoer); ok {
		return u
	}
	return w.textEditor
}

func (w *MainWindow) undo() { w.editTarget().Undo() }
func (w *MainWindow) redo() { w.editTarget().Redo() }
