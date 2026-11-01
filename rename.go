package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// A scriptTitle is the script's name over the editor; clicked, it asks
// for a new one.
type scriptTitle struct {
	widget.Label
	onTap func()
}

func newScriptTitle(text string, onTap func()) *scriptTitle {
	t := &scriptTitle{onTap: onTap}
	t.Text = text
	t.ExtendBaseWidget(t)
	fyne.SetAccessibleLabel(t, "Script name: click to rename")
	return t
}

func (t *scriptTitle) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *scriptTitle) Cursor() desktop.Cursor { return desktop.PointerCursor }

// scriptName is the script's name as the title has it: its file's, the
// one it will be saved as, or "" for neither.
func (w *MainWindow) scriptName() string {
	if w.currentFile != "" {
		return filepath.Base(w.currentFile)
	}
	return w.suggestedName
}

// renamedName is a name typed for a script: trimmed, a bare name with the
// old one's extension (draft and draft.fountain alike), "" if it is no
// name for a file.
func renamedName(typed, old string) string {
	name := strings.TrimSpace(typed)
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return ""
	}
	if filepath.Ext(name) == "" {
		ext := filepath.Ext(old)
		if ext == "" {
			ext = ".fountain"
		}
		name += ext
	}
	return name
}

// renameScript asks for the script's name and gives it: a saved script's
// file is renamed, an unsaved one will be saved by it. A name that says a
// language (scene.nl.fountain) makes the script that language's, as
// saving does.
func (w *MainWindow) renameScript() {
	old := w.scriptName()
	entry := widget.NewEntry()
	entry.SetText(old)
	if old == "" {
		entry.SetPlaceHolder("untitled.fountain")
	}
	where := "It is not saved yet: this is the name it will be saved by."
	if w.currentFile != "" {
		where = "The file is renamed in " + filepath.Dir(w.currentFile) + "."
	}
	note := widget.NewLabel(where + " A language in the name (scene.eo.fountain) is the script's.")
	note.Wrapping = fyne.TextWrapWord
	form := widget.NewForm(widget.NewFormItem("Name", entry), widget.NewFormItem("", note))
	d := dialog.NewCustomConfirm("Rename Script", "Rename", "Cancel", form, func(ok bool) {
		if !ok {
			return
		}
		if err := w.renameTo(renamedName(entry.Text, old)); err != nil {
			dialog.ShowError(err, w.fyneWindow)
		}
	}, w.fyneWindow)
	entry.OnSubmitted = func(string) { d.Confirm() }
	d.Resize(fyne.NewSize(420, 220))
	d.Show()
	w.fyneWindow.Canvas().Focus(entry)
}

// renameTo gives the script the name name.
func (w *MainWindow) renameTo(name string) error {
	if name == "" {
		return fmt.Errorf("that is no name for a file")
	}
	if w.currentFile == "" {
		w.suggestedName = name
	} else if to := filepath.Join(filepath.Dir(w.currentFile), name); to != w.currentFile {
		if _, err := os.Stat(to); err == nil && !strings.EqualFold(to, w.currentFile) {
			return fmt.Errorf("%s already exists", name)
		}
		if err := os.Rename(w.currentFile, to); err != nil {
			return fmt.Errorf("could not rename the file: %v", err)
		}
		w.currentFile = to
		w.rememberRecent(to)
	}
	if l := languageOf(name); l != "" && l != w.language {
		w.setLanguage(l)
	}
	w.updateTitle()
	return nil
}
