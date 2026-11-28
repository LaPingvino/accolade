// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/fountain"
)

// Lexington 1.19 reads Fountain as the spec has it where it used to read
// more loosely: a scene heading and a character cue need a blank line
// before them, a transition is in capitals with blank lines around it, a
// line of two spaces keeps a speech going. A script written against the
// old reading may mean something else now. On opening one (a plain file
// or a bundle, on the desktop or in the browser), Accolade asks once,
// above the status bar: Keep It As It Was (Lexington's fountain.KeepLoose:
// blank lines and force marks added so that it reads as it did; one undo
// step) or Read It the New Way. The answer is remembered per script
// (settings.json "loose-reading", "keep\t/path" or "new\t/path", moved
// with a rename), so it never asks twice; a script not saved yet is asked
// once while it is open.

// looseReadingPref is the setting that remembers the answers.
const looseReadingPref = "loose-reading"

// looseNotice is the notice row: what it says, and its two answers.
type looseNotice struct {
	box  *fyne.Container
	text *widget.Label
	keep *widget.Button
	new  *widget.Button
}

func (w *MainWindow) newLooseNotice() *fyne.Container {
	n := &looseNotice{text: widget.NewLabel("")}
	n.text.Wrapping = fyne.TextWrapWord
	n.keep = widget.NewButtonWithIcon("Keep It As It Was", theme.HistoryIcon(), w.keepLooseReading)
	n.keep.Importance = widget.HighImportance
	n.new = widget.NewButton("Read It the New Way", w.readTheNewWay)
	n.box = container.NewVBox(
		widget.NewSeparator(),
		n.text,
		container.NewHBox(layout.NewSpacer(), n.new, n.keep),
	)
	n.box.Hide()
	w.loose = n
	return n.box
}

// checkLooseReading shows the notice if the script in the window reads
// otherwise under the spec than it did, and its answer is not known.
func (w *MainWindow) checkLooseReading() {
	w.hideLooseNotice()
	if w.markdown || w.textEditor == nil || w.looseAnswer(w.currentFile) != "" {
		return
	}
	if _, places := fountain.KeepLoose(w.textEditor.Text(), w.sceneHeaders()); places > 0 {
		w.showLooseNotice(places)
	}
}

func (w *MainWindow) showLooseNotice(places int) {
	if w.loose == nil {
		return
	}
	w.loose.text.SetText("This script relied on Lexington's old, looser reading in " + placesText(places) + ".")
	w.loose.box.Show()
}

// placesText is "1 place", "2 places".
func placesText(n int) string {
	if n == 1 {
		return "1 place"
	}
	return fmt.Sprintf("%d places", n)
}

func (w *MainWindow) hideLooseNotice() {
	if w.loose != nil {
		w.loose.box.Hide()
	}
}

// keepLooseReading writes the script so that it reads as it did: one
// edit, one undo step.
func (w *MainWindow) keepLooseReading() {
	w.hideLooseNotice()
	old := w.textEditor.Text()
	kept, places := fountain.KeepLoose(old, w.sceneHeaders())
	w.rememberLooseAnswer(w.currentFile, "keep")
	if places == 0 {
		w.setStatus("The script reads the same either way: nothing to change")
		return
	}
	w.replaceText(old, kept)
	w.setStatus("Kept as it was: blank lines and force marks in " + placesText(places) + " (Undo takes them back)")
}

// readTheNewWay leaves the script as it is, read as the spec has it.
func (w *MainWindow) readTheNewWay() {
	w.hideLooseNotice()
	w.rememberLooseAnswer(w.currentFile, "new")
	w.setStatus("Read the new way, as the Fountain spec has it")
}

// looseAnswer is what was answered for a script: "keep", "new" or "".
// A script not saved yet: the window's answer, while it is open.
func (w *MainWindow) looseAnswer(script string) string {
	if script == "" {
		return w.looseUnsaved
	}
	if w.settings == nil {
		return ""
	}
	script = absPath(script)
	for _, e := range w.settings.GetStringSlice(looseReadingPref) {
		if how, path, ok := strings.Cut(e, "\t"); ok && path == script {
			return how
		}
	}
	return ""
}

// rememberLooseAnswer keeps the answer for a script ("" forgets it).
func (w *MainWindow) rememberLooseAnswer(script, how string) {
	if script == "" {
		w.looseUnsaved = how
		return
	}
	if w.settings == nil {
		return
	}
	script = absPath(script)
	var list []string
	if how != "" {
		list = append(list, how+"\t"+script)
	}
	old := w.settings.GetStringSlice(looseReadingPref)
	for _, e := range old {
		if _, path, _ := strings.Cut(e, "\t"); path != script && len(list) < maxDocumentTypes {
			list = append(list, e)
		}
	}
	if len(list) == 0 && len(old) == 0 {
		return
	}
	w.settings.SetStringSlice(looseReadingPref, list)
	_ = w.settings.Save()
}

// looseAfterRename carries a script's answer to its new name (a script
// saved for the first time: the window's answer).
func (w *MainWindow) looseAfterRename(from, to string) {
	how := w.looseAnswer(from)
	if how == "" || to == "" {
		return
	}
	w.rememberLooseAnswer(to, how)
	if from == "" {
		w.looseUnsaved = ""
	}
}
