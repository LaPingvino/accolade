// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/docs/help"
)

// The editor of a .toml file (toml_doc.go): a window of its own on the
// desktop, a dialog over the window in the browser, as Preferences is.
// Not a middle tab (middle.go): the file is no part of a script. It is
// opened from Preferences (itself a window), from the command line with
// no script at all (accolade lexington.toml), and saving it changes every
// window; a tab would tie it to one script's window, its preview, menus
// and ribbon, and close with it.
//
// Its tabs are the form's pages, then TOML (the text itself, the same the
// pages edit) and Help. The text is the only state: a page is drawn from
// it, each control is one edit of it, the TOML tab shows it. Text that is
// no TOML keeps the pages off until it is mended.

// tomlEditor is one file's editor.
type tomlEditor struct {
	app    *Application
	doc    *tomlDoc
	host   fyne.Window   // what its dialogs show over
	win    fyne.Window   // its window (the desktop)
	dlg    dialog.Dialog // its dialog (the browser)
	tabs   *container.AppTabs
	pages  []*editorPage
	raw    *widget.Entry
	rawTab *container.TabItem
	status *widget.Label
	// sel are the pages' choices (the language shown, the element), by
	// page, so a page drawn again shows the same
	sel map[string]string
	// sample is the element sample of the page in front (nil: none)
	sample *formatSample
}

// editorPage is a page of the form, drawn from the text by build.
type editorPage struct {
	item  *container.TabItem
	build func() fyne.CanvasObject
	stale bool
}

// tomlEditors are the editors open, so closing the last script window
// does not quit under one, and one file opens once.
func (app *Application) tomlEditorOf(path string) *tomlEditor {
	for _, e := range app.tomlEditors {
		if absPath(e.doc.path) == absPath(path) {
			return e
		}
	}
	return nil
}

// newTOMLEditor makes the editor of doc; pages are added, then show.
func newTOMLEditor(app *Application, doc *tomlDoc) *tomlEditor {
	e := &tomlEditor{app: app, doc: doc, sel: map[string]string{}}
	e.status = widget.NewLabel("")
	e.status.Wrapping = fyne.TextWrapWord
	e.raw = widget.NewMultiLineEntry()
	e.raw.TextStyle = fyne.TextStyle{Monospace: true}
	e.raw.SetText(doc.text)
	e.raw.OnChanged = e.rawChanged
	return e
}

// page adds a page of the form.
func (e *tomlEditor) page(title string, build func() fyne.CanvasObject) {
	e.pages = append(e.pages, &editorPage{item: container.NewTabItem(title, widget.NewLabel("")), build: build, stale: true})
}

// kindName is what the file is, for its title.
func (e *tomlEditor) kindName() string {
	switch e.doc.kind {
	case tomlSettings:
		return "Lexington Settings"
	case tomlFormat:
		return "Format"
	}
	return "TOML"
}

// title is the window's title: what, which file, • if changed.
func (e *tomlEditor) title() string {
	t := e.kindName() + ": " + displayPath(e.doc.path)
	if e.doc.kind == tomlSettings {
		if absPath(e.doc.path) == absPath(lexingtonConfig()) {
			t += " (Accolade's; the lexington command reads it with -config)"
		} else {
			t += " (the lexington command reads it)"
		}
	}
	if e.doc.dirty() {
		t = "• " + t
	}
	return t
}

// displayPath is a path with the home folder as ~.
func displayPath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(p, home+string(filepath.Separator)) {
		return "~" + p[len(home):]
	}
	return p
}

// about is the line above the tabs: who reads this file.
func (e *tomlEditor) about() string {
	p := e.doc.path
	switch e.doc.kind {
	case tomlSettings:
		if absPath(p) == absPath(lexingtonConfig()) {
			return "Accolade's Lexington settings: its exports and its scene headings read them. The lexington command reads them too with -config " + displayPath(p) + "."
		}
		return "The lexington command reads this file when run in " + displayPath(filepath.Dir(p)) + " (or with -config " + displayPath(p) +
			"). Accolade reads its own: Preferences > Script > Edit Lexington Settings…."
	case tomlFormat:
		switch {
		case absPath(filepath.Dir(p)) == absPath(builtinFormatDir()):
			return "One of Accolade's formats. Saving keeps a changed copy in your formats folder (" + displayPath(userFormatDir()) + "), over it."
		case inFormatFolder(p):
			return "A format in your formats folder: every script can use it once saved. The lexington command reads it with -formats " + displayPath(filepath.Dir(p)) + "."
		}
		return "A format file. The lexington command reads it with -formats " + displayPath(filepath.Dir(p)) + "; Add to My Formats lets Accolade's scripts use it."
	}
	return "This file is no Lexington settings or format file Accolade knows: it is edited as text."
}

// content is the editor: the line saying what it is, the tabs, the
// status and the buttons.
func (e *tomlEditor) content() fyne.CanvasObject {
	var items []*container.TabItem
	for _, p := range e.pages {
		items = append(items, p.item)
	}
	e.rawTab = container.NewTabItem("TOML", e.raw)
	items = append(items, e.rawTab)
	if e.doc.kind != tomlPlain {
		items = append(items, container.NewTabItem("Help", e.helpPage()))
	}
	e.tabs = container.NewAppTabs(items...)
	e.tabs.OnSelected = func(*container.TabItem) { e.shown() }
	about := widget.NewLabel(e.about())
	about.Wrapping = fyne.TextWrapWord
	saveBtn := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), e.saveNow)
	saveBtn.Importance = widget.HighImportance
	buttons := container.NewHBox(layout.NewSpacer(), widget.NewButton("Close", e.close), saveBtn)
	if e.doc.kind == tomlFormat && !inFormatFolder(e.doc.path) {
		buttons.Objects = append([]fyne.CanvasObject{widget.NewButton("Add to My Formats", e.addToMyFormats)}, buttons.Objects...)
	}
	err := e.doc.check(e.doc.text)
	if err != nil { // to be mended in the TOML tab first
		e.tabs.Select(e.rawTab)
		for i := range e.pages {
			e.tabs.DisableIndex(i)
		}
	}
	e.changed(err)
	return container.NewBorder(about, container.NewVBox(e.status, buttons), nil, nil, e.tabs)
}

// show opens the editor: a window, or in the browser a dialog over
// parent.
func (e *tomlEditor) show(parent fyne.Window) {
	content := e.content()
	if e.app != nil {
		e.app.tomlEditors = append(e.app.tomlEditors, e)
	}
	if inBrowser && parent != nil {
		e.host = parent
		e.dlg = dialog.NewCustomWithoutButtons(e.title(), content, parent)
		e.dlg.Resize(fitDialog(parent, 860, 640))
		e.dlg.Show()
		e.shown()
		return
	}
	a := fyne.CurrentApp()
	if e.app != nil && e.app.fyneApp != nil {
		a = e.app.fyneApp
	}
	e.win = a.NewWindow(e.title())
	e.host = e.win
	e.win.SetContent(content)
	e.win.Resize(fyne.NewSize(860, 640))
	e.win.SetCloseIntercept(e.close)
	e.win.SetOnClosed(e.closed)
	e.win.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierShortcutDefault}, func(fyne.Shortcut) { e.saveNow() })
	e.win.CenterOnScreen()
	e.win.Show()
	e.shown()
}

// shown draws the page in front if the text changed since.
func (e *tomlEditor) shown() {
	if e.tabs == nil {
		return
	}
	cur := e.tabs.Selected()
	if cur == e.rawTab {
		if e.raw.Text != e.doc.text {
			e.raw.SetText(e.doc.text)
		}
		return
	}
	for _, p := range e.pages {
		if p.item == cur && p.stale {
			e.draw(p)
		}
	}
}

// draw draws a page from the text.
func (e *tomlEditor) draw(p *editorPage) {
	e.sample.stop()
	e.sample = nil
	p.item.Content = p.build()
	p.stale = false
	e.tabs.Refresh()
}

// redraw draws the page in front again (a list grew, an override was
// made) and marks the others to be drawn when they come to the front.
func (e *tomlEditor) redraw() {
	for _, p := range e.pages {
		p.stale = true
	}
	e.shown()
}

// do makes edits from the form; the page stays as it is (the control
// that made them shows them already).
func (e *tomlEditor) do(ops ...tomlOp) bool {
	err := e.doc.edit(ops...)
	e.changed(err)
	return err == nil
}

// doAndRedraw makes edits that change what the page shows.
func (e *tomlEditor) doAndRedraw(ops ...tomlOp) bool {
	ok := e.do(ops...)
	e.redraw()
	return ok
}

// changed shows the state after an edit: its error, else what is to
// know about the text; the title's •; the other pages to be drawn again.
func (e *tomlEditor) changed(err error) {
	for _, p := range e.pages {
		if e.tabs == nil || p.item != e.tabs.Selected() {
			p.stale = true
		}
	}
	switch {
	case err != nil:
		e.status.Importance = widget.DangerImportance
		e.status.SetText(err.Error())
	default:
		e.status.Importance = widget.MediumImportance
		notes := e.doc.notes()
		if e.doc.dirty() {
			notes = append([]string{"Changed, not saved yet."}, notes...)
		}
		e.status.SetText(strings.Join(notes, "\n"))
	}
	if e.win != nil {
		e.win.SetTitle(e.title())
	}
	if e.sample != nil {
		e.sample.update()
	}
}

// rawChanged follows the TOML tab: the text is the file; text that is
// not right keeps the pages off and says why (with its line).
func (e *tomlEditor) rawChanged(text string) {
	if text == e.doc.text {
		return
	}
	e.doc.text = text
	err := e.doc.check(text)
	for i := range e.pages {
		if e.tabs == nil {
			break
		}
		if err != nil {
			e.tabs.DisableIndex(i)
		} else {
			e.tabs.EnableIndex(i)
		}
	}
	e.changed(err)
}

// saveNow keeps the text if it is right.
func (e *tomlEditor) saveNow() {
	if err := e.doc.check(e.doc.text); err != nil {
		e.changed(err)
		dialog.ShowError(fmt.Errorf("not saved: %w", err), e.host)
		return
	}
	if err := e.writeFile(e.doc.text); err != nil {
		dialog.ShowError(err, e.host)
		return
	}
	e.doc.saved = e.doc.text
	e.afterSave()
	e.changed(nil)
	e.status.SetText("Saved.")
}

// writeFile writes the text to the file, and where the file is one
// Accolade reads, reads it again; a format of Accolade's goes into one's
// own folder; in the browser a file opened from the computer is offered
// back as a download.
func (e *tomlEditor) writeFile(text string) error {
	p := e.doc.path
	if e.doc.kind == tomlFormat && absPath(filepath.Dir(p)) == absPath(builtinFormatDir()) {
		if err := saveFormat(e.doc.key, text); err != nil {
			return err
		}
		e.doc.path = filepath.Join(userFormatDir(), e.doc.key+".toml")
		return nil
	}
	if absPath(p) == absPath(lexingtonConfig()) {
		return saveLexingtonSettings(text)
	}
	if e.doc.kind == tomlFormat && inFormatFolder(p) {
		return saveFormat(e.doc.key, text)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if err := writeFileAtomic(p, []byte(text)); err != nil {
		return err
	}
	if inBrowser && !strings.HasPrefix(absPath(p), absPath(getConfigDir())) {
		offerDownload(p)
	}
	return nil
}

// afterSave tells the rest of Accolade: its settings read again, its
// formats listed again.
func (e *tomlEditor) afterSave() {
	if absPath(e.doc.path) == absPath(lexingtonConfig()) || e.doc.kind == tomlFormat && inFormatFolder(e.doc.path) {
		if e.app != nil {
			e.app.scriptFormatsChanged()
			if e.app.prefs != nil {
				e.app.prefs.settingsFilesChanged()
			}
		}
	}
}

// addToMyFormats saves the format into one's own formats folder, under
// its file's name.
func (e *tomlEditor) addToMyFormats() {
	if err := e.doc.check(e.doc.text); err != nil {
		dialog.ShowError(err, e.host)
		return
	}
	key := formatKey(e.doc.key)
	add := func() {
		if err := saveFormat(key, e.doc.text); err != nil {
			dialog.ShowError(err, e.host)
			return
		}
		if e.app != nil {
			e.app.scriptFormatsChanged()
			if e.app.prefs != nil {
				e.app.prefs.settingsFilesChanged()
			}
		}
		e.status.SetText("Added to your formats as " + key + ".toml: every script can use it now.")
	}
	if userFormat(key) {
		dialog.ShowConfirm("Replace Your Format?", "You have a format "+key+" already. Replace it with this file?", func(ok bool) {
			if ok {
				add()
			}
		}, e.host)
		return
	}
	add()
}

// close closes the editor, asking first if there are changes.
func (e *tomlEditor) close() {
	shut := func() {
		if e.win != nil {
			e.win.Close()
			return
		}
		if e.dlg != nil {
			e.dlg.Hide()
		}
		e.closed()
	}
	if !e.doc.dirty() {
		shut()
		return
	}
	dialog.ShowConfirm("Close Without Saving?", "The changes to "+filepath.Base(e.doc.path)+" are not saved. Close anyway?", func(ok bool) {
		if ok {
			shut()
		}
	}, e.host)
}

// closed: the editor is gone; with no window left, Accolade ends.
func (e *tomlEditor) closed() {
	e.sample.stop()
	if e.app == nil {
		return
	}
	for i, x := range e.app.tomlEditors {
		if x == e {
			e.app.tomlEditors = append(e.app.tomlEditors[:i], e.app.tomlEditors[i+1:]...)
			break
		}
	}
	if len(e.app.windows) == 0 && len(e.app.tomlEditors) == 0 && e.app.prefs == nil && !inBrowser && e.app.fyneApp != nil {
		e.app.fyneApp.Quit()
	}
}

// helpPage is the Help tab: what each setting does, for Accolade and for
// the lexington command (docs/help settings.md, made from Lexington's
// descriptions).
func (e *tomlEditor) helpPage() fyne.CanvasObject {
	p, _ := help.Get("settings")
	text := widget.NewRichText(richMarkdown(p.Markdown, nil)...)
	text.Wrapping = fyne.TextWrapWord
	return container.NewVScroll(container.NewPadded(text))
}
