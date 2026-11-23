package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

// Opening a .toml: File > Open, a file dropped on the window, Open Recent,
// the command line (accolade path/x.toml) and the desktop's Open With all
// come here before a script window is picked, so the window's script stays
// as it is. The file is edited where it is.

// isTOMLFile reports whether path is a .toml file.
func isTOMLFile(path string) bool { return strings.EqualFold(filepath.Ext(path), ".toml") }

// openTOMLPath opens the .toml file at path in its editor.
func openTOMLPath(app *Application, parent fyne.Window, path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && strings.EqualFold(filepath.Base(path), "lexington.toml") {
			b = []byte(newLexingtonSettings) // a new one, written on Save
		} else {
			return err
		}
	}
	openTOMLFile(app, parent, path, string(b))
	return nil
}

// openTOMLFile opens the editor of the file at path with text (what it
// has, or what a new one starts with); a file open already comes to the
// front.
func openTOMLFile(app *Application, parent fyne.Window, path, text string) *tomlEditor {
	if app != nil {
		if e := app.tomlEditorOf(path); e != nil {
			if e.win != nil {
				e.win.RequestFocus()
			} else if e.dlg != nil {
				e.dlg.Show()
			}
			return e
		}
	}
	doc := newTOMLDoc(path, text)
	switch doc.kind {
	case tomlSettings:
		return openSettingsEditor(app, parent, doc)
	case tomlFormat:
		return openFormatEditor(app, parent, doc)
	}
	e := newTOMLEditor(app, doc)
	e.show(parent)
	return e
}

// openTOMLFrom opens a .toml from a window (File > Open and the like),
// reporting a failure over it.
func (w *MainWindow) openTOMLFrom(path string) {
	if err := openTOMLPath(w.app, w.fyneWindow, path); err != nil {
		dialog.ShowError(fmt.Errorf("failed to open %s: %v", filepath.Base(path), err), w.fyneWindow)
	}
}
