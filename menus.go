package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// shortcut builds a Ctrl+key (Cmd+key on macOS) shortcut; extra
// modifiers such as Shift are added on top.
func shortcut(key fyne.KeyName, extra fyne.KeyModifier) fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: key, Modifier: fyne.KeyModifierShortcutDefault | extra}
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

// buildMainMenu is the menu bar, from the command registry (commands.go).
func (w *MainWindow) buildMainMenu() *fyne.MainMenu { return w.menuBar(w.cmdEnv()) }

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

func (w *MainWindow) undo() { w.followSceneNumbers(); w.editTarget().Undo(); w.followSceneNumbers() }
func (w *MainWindow) redo() { w.followSceneNumbers(); w.editTarget().Redo(); w.followSceneNumbers() }

// editShortcut gives a clipboard or selection shortcut (Cut, Copy, Paste,
// Select All) to the text field with the focus, else the editor.
func (w *MainWindow) editShortcut(s fyne.Shortcut) {
	if f, ok := w.fyneWindow.Canvas().Focused().(fyne.Shortcutable); ok {
		f.TypedShortcut(s)
		return
	}
	w.textEditor.TypedShortcut(s)
}
