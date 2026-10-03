package main

import (
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
		menuItem("New Window", shortcut(fyne.KeyN, fyne.KeyModifierShift), func() { w.app.newWindow() }),
		menuItem("Open…", shortcut(fyne.KeyO, 0), w.OpenFile),
		fyne.NewMenuItemSeparator(),
		menuItem("Save", shortcut(fyne.KeyS, 0), w.reportErr(w.SaveFile)),
		menuItem("Save As…", shortcut(fyne.KeyS, fyne.KeyModifierShift), w.reportErr(w.SaveFileAs)),
		fyne.NewMenuItemSeparator(),
		menuItem("Title Page…", shortcut(fyne.KeyT, fyne.KeyModifierShift), w.showTitlePageDialog),
		menuItem("Export…", shortcut(fyne.KeyE, 0), w.showExportDialog),
		fyne.NewMenuItemSeparator(),
		menuItem("Preferences…", shortcut(fyne.KeyComma, 0), func() { w.app.showPreferences() }),
		// Fyne appends Quit (Ctrl+Q) to the first menu itself.
	)

	edit := fyne.NewMenu("Edit",
		menuItem("Find…", shortcut(fyne.KeyF, 0), w.showFind),
		menuItem("Replace…", shortcut(fyne.KeyH, 0), w.showFindReplace),
		menuItem("Find Next", shortcut(fyne.KeyG, 0), w.findNext),
		menuItem("Find Previous", shortcut(fyne.KeyG, fyne.KeyModifierShift), w.findPrevious),
	)

	view := fyne.NewMenu("View",
		menuItem("Preview", shortcut(fyne.KeyP, fyne.KeyModifierShift), w.togglePreview),
		menuItem("Fullscreen", shortcut(fyne.KeyF, fyne.KeyModifierShift), w.toggleFullscreen),
	)

	help := fyne.NewMenu("Help",
		fyne.NewMenuItem("About Accolade", func() { w.app.showAbout() }),
	)

	return fyne.NewMainMenu(file, edit, view, help)
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
