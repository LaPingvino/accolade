package main

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// Save As, on the desktop and in the browser (docs/PRODUCTION.md §4,
// step 3):
//
//   - The name offered is the file's, else the one suggested, else
//     untitled; a screenplay with production data (drafts, a lock in its
//     production block) that would be saved plain is offered as
//     .accolade instead. Save (and auto-save) of a file that has a name
//     keep writing it as it is: a plain .fountain stays one, its drafts
//     where a plain script keeps them (extras.go).
//   - A bundle with drafts saved as a plain file asks where they go
//     (flatten.go).
//   - A .highland that holds Accolade's files (drafts, the source) says
//     once, after it is saved, that Highland may drop them: untested with
//     a real Highland (docs/HIGHLAND.md). The file remembers it was said
//     (info.json, Accolade's key), so it is said once per file.

// saveAsName is the name Save As offers: the file's (current), else the
// suggested one, else untitled; a screenplay with production data that
// would be plain is offered as .accolade.
func saveAsName(current, suggested, untitled string, production, markdown bool) string {
	name := untitled
	if current != "" {
		name = filepath.Base(current)
	} else if suggested != "" {
		name = suggested
	}
	if production && !markdown && bundleKindOf(name) == notBundle {
		name = strings.TrimSuffix(name, filepath.Ext(name)) + ".accolade"
	}
	return name
}

// offeredName is the name Save As offers for the window's script.
func (w *MainWindow) offeredName() string {
	return saveAsName(w.currentFile, w.suggestedName, w.untitledName(), w.hasProductionData(), w.markdown)
}

// askBrowserName is the browser's Save As: a name for the file kept here
// and downloaded.
func (w *MainWindow) askBrowserName() {
	entry := widget.NewEntry()
	entry.SetText(w.offeredName())
	d := dialog.NewForm("Save As", "Save", "Cancel", []*widget.FormItem{widget.NewFormItem("Name", entry)}, func(ok bool) {
		if !ok {
			return
		}
		p, err := browserSavePath(entry.Text)
		if err != nil {
			dialog.ShowError(err, w.fyneWindow)
			return
		}
		w.saveAs(p)
	}, w.fyneWindow)
	entry.OnSubmitted = func(string) { d.Submit() }
	d.Resize(fitDialog(w.fyneWindow, 440, 180))
	d.Show()
	w.fyneWindow.Canvas().Focus(entry)
}

// browserSavePath is where the browser keeps a file named name.
func browserSavePath(name string) (string, error) {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "", fmt.Errorf("that is no name for a file")
	}
	if bundleKindOf(name) == bundleFolder {
		return "", fmt.Errorf("a .textbundle is a folder, which the browser cannot save: use .accolade or .textpack")
	}
	if err := os.MkdirAll("/files", 0o755); err != nil {
		return "", err
	}
	return filepath.Join("/files", name), nil
}

// saveAs saves the script as filePath, once the writes under way are
// done; a bundle's drafts saved plain are asked about first.
func (w *MainWindow) saveAs(filePath string) {
	w.afterBundleWrites(func() {
		if w.bundle != nil && len(w.bundle.extras) > 0 && bundleKindOf(filePath) == notBundle {
			w.askFlatten(filePath, func(how flattenChoice) {
				w.flattenTo = how
				w.finishSaveAs(filePath)
			})
			return
		}
		w.finishSaveAs(filePath)
	})
}

// finishSaveAs writes the script to filePath and makes it the window's
// file.
func (w *MainWindow) finishSaveAs(filePath string) {
	flattening := w.bundle != nil && len(w.bundle.extras) > 0 && bundleKindOf(filePath) == notBundle
	how := w.flattenTo
	failures := w.saveFailures
	w.splitOldRunsOnSave()
	if err := w.saveToFile(filePath); err != nil {
		w.flattenTo = flattenKept
		dialog.ShowError(fmt.Errorf("failed to save file: %v", err), w.fyneWindow)
		return
	}
	oldPath, oldName := w.currentFile, cmp.Or(w.currentFile, w.suggestedName, w.untitledName())
	w.currentFile = filePath
	w.suggestedName = ""
	w.rememberRecent(filePath)
	w.typeAfterRename(oldPath, oldName, filePath, filePath)
	if l := languageOf(filePath); l != "" {
		w.setLanguage(l)
	}
	w.hasChanges = false
	w.updateTitle()
	if flattening {
		w.setStatus(map[flattenChoice]string{
			flattenKept:   "Saved; its drafts are kept by Accolade, found again when you open it here",
			flattenFolder: "Saved; its drafts are in " + filepath.Base(productionFolderOf(filePath)) + " beside it",
			flattenDrop:   "Saved, without its drafts",
		}[how])
	}
	if inBrowser {
		w.afterBundleWrites(func() { offerDownload(filePath) })
		return
	}
	w.afterBundleWrites(func() {
		if !flattening && w.saveFailures == failures { // a background save that failed said so
			dialog.ShowInformation("File Saved", fmt.Sprintf("File saved to %s", filepath.Base(filePath)), w.fyneWindow)
		}
	})
}

// flattenChoices are where a bundle's drafts can go as it is saved plain:
// the browser has no folder beside a file.
func flattenChoices(browser bool) []flattenChoice {
	if browser {
		return []flattenChoice{flattenKept, flattenDrop}
	}
	return []flattenChoice{flattenKept, flattenFolder, flattenDrop}
}

// flattenAsk is the question asked (for the tests: choose answers it, as
// the dialog's Save does; confirmDrop the "leave them out?" that follows).
type flattenAsk struct {
	choose      func(flattenChoice)
	confirmDrop func(bool)
}

// askFlatten asks where the bundle's drafts go as it is saved as the plain
// file p, then calls then with the answer (not if cancelled).
func (w *MainWindow) askFlatten(p string, then func(flattenChoice)) {
	bundle := filepath.Base(w.bundle.path)
	what := "drafts"
	if n, err := readDrafts(w.extrasHome()); err == nil && len(n) == 1 {
		what = "draft"
	} else if err == nil && len(n) > 1 {
		what = fmt.Sprintf("%d drafts", len(n))
	}
	choices := flattenChoices(inBrowser)
	labels := make([]string, len(choices))
	for i, c := range choices {
		switch c {
		case flattenKept:
			labels[i] = "Kept by Accolade (they come back when you open " + filepath.Base(p) + " here)"
		case flattenFolder:
			labels[i] = "In " + filepath.Base(productionFolderOf(p)) + " beside the file"
		case flattenDrop:
			labels[i] = "Left out (" + bundle + " keeps them)"
		}
	}
	radio := widget.NewRadioGroup(labels, nil)
	radio.Required = true
	radio.SetSelected(labels[0])
	text := widget.NewLabel(filepath.Base(p) + " is plain Fountain: it cannot hold the " + what + " kept in " + bundle + ". Where should they go?")
	text.Wrapping = fyne.TextWrapWord
	ask := &flattenAsk{}
	w.flattenAsked = ask
	ask.choose = func(how flattenChoice) {
		if how != flattenDrop {
			then(how)
			return
		}
		ask.confirmDrop = func(ok bool) {
			if ok {
				then(how)
			}
		}
		dialog.ShowConfirm("Leave the Drafts Out?",
			filepath.Base(p)+" will be saved without the "+what+". "+bundle+" keeps them, as it is on disk now.",
			ask.confirmDrop, w.fyneWindow)
	}
	d := dialog.NewCustomConfirm("Where Should the Drafts Go?", "Save", "Cancel", container.NewVBox(text, radio), func(ok bool) {
		if !ok {
			return
		}
		for i, l := range labels {
			if l == radio.Selected {
				ask.choose(choices[i])
			}
		}
	}, w.fyneWindow)
	d.Resize(fitDialog(w.fyneWindow, 520, 300))
	d.Show()
}

// highlandWarning says, once per file, that Highland may drop Accolade's
// files from a .highland.
func (w *MainWindow) highlandWarning(p string) {
	key := absPath(p)
	if w.highlandWarned[key] {
		return
	}
	if w.highlandWarned == nil {
		w.highlandWarned = map[string]bool{}
	}
	w.highlandWarned[key] = true
	if w.fyneWindow == nil {
		return
	}
	dialog.ShowInformation("Highland May Drop Accolade's Files",
		"Saved as "+filepath.Base(p)+". Accolade keeps the script's drafts (and its source) inside the file, "+
			"under assets/. Highland has not been tried with them yet: when Highland saves this file, it may leave them out. "+
			"To keep them safe, keep a copy as .accolade (File > Save As…). This is said once for this file.",
		w.fyneWindow)
}

// warnsHighland tells whether saving to p should give the Highland
// warning: a .highland holding Accolade's files (extra carried in, or
// the bundle's), not warned about before.
func (w *MainWindow) warnsHighland(p string, extra []bundleFile) bool {
	if !strings.EqualFold(filepath.Ext(p), ".highland") {
		return false
	}
	b := w.bundle
	ours := len(extra) > 0
	if b != nil {
		entry, _ := w.bundleSource()
		ours = ours || len(b.extras) > 0 || entry != ""
		if b.ours.HighlandWarned && strings.EqualFold(filepath.Ext(b.path), ".highland") {
			return false // said for this file (or the one it is a copy of)
		}
	}
	return ours && !w.highlandWarned[absPath(p)]
}
