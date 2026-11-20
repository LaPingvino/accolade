package main

import (
	"cmp"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Save As… first asks for the type, Accolade's own step: Fyne's save
// dialog has a filter but no type chooser, and the browser's download
// none at all. Each type says in a line what it keeps; the file's type now
// is chosen, the recommended one marked; below, the script's files and
// where each goes (the rename editor's list, planPanel). Then the system's
// save dialog, with the type's filter and extension (the browser: the
// name, here, and the download).

// saveAsAsk is the Save As chooser last opened (for the tests).
type saveAsAsk struct {
	types   []fileType
	choose  func(i int)
	chosen  func() fileType
	name    *widget.Entry
	plan    *planPanel
	radios  []*widget.RadioGroup
	scroll  *container.Scroll
	confirm func()
}

// showSaveAs is Save As…: the type chooser, then where.
func (w *MainWindow) showSaveAs() {
	offered := w.offeredName()
	now := cmp.Or(w.scriptName(), offered)
	types := scriptTypes(w.markdown, inBrowser, now)
	rec := recommendedType(w.markdown, w.hasProductionData())
	current := typeOfName(now, w.markdown)
	if w.currentFile == "" && w.suggestedName == "" { // nothing yet: the recommended one
		current = rec
	}
	if w.bundle == nil && w.isPlainBundle(w.currentFile) {
		current = typeFountain // a .highland of plain text is Fountain
	}
	base := splitScriptName(offered)
	base.ext = ""

	dir := ""
	if w.currentFile != "" {
		dir = filepath.Dir(w.currentFile)
	}
	if inBrowser {
		dir = "/files"
	}
	ask := &saveAsAsk{types: types}
	w.saveAsAsked = ask
	sel := max(0, indexOfType(types, current))
	keepsRow := func(t fileType) fyne.CanvasObject {
		l := widget.NewLabel(t.keeps(w.markdown, inBrowser))
		l.Wrapping = fyne.TextWrapWord
		return container.NewBorder(nil, nil, container.NewGridWrap(fyne.NewSize(theme.IconInlineSize()+theme.Padding()*2, 1)), nil, l)
	}
	name := widget.NewEntry()
	name.SetText(base.String())
	fyne.SetAccessibleLabel(name, "Name")
	ask.name = name
	result := widget.NewLabel("")
	result.TextStyle = fyne.TextStyle{Bold: true}
	result.Wrapping = fyne.TextWrapBreak

	target := func() relocation {
		t := types[sel]
		n := composedName(name.Text, "", t)
		if n == "" || strings.TrimSpace(name.Text) == "" {
			n = base.String() + t.ext
		}
		r := relocation{from: w.currentFile, to: filepath.Join(dir, filepath.Base(n))}
		if dir == "" {
			r.to = filepath.Base(n)
		}
		ask.plan.fill(&r)
		return r
	}
	update := func() {
		r := target()
		result.SetText("Saved as " + filepath.Base(r.to))
		ask.plan.show(r)
	}
	ask.plan = w.newPlanPanel(false, update)
	setting := false
	var list []fyne.CanvasObject
	for i, t := range types {
		label := t.label + " (" + t.ext + ")"
		var marks []string
		if t.id == current.id && (w.currentFile != "" || w.suggestedName != "") {
			marks = append(marks, "now")
		}
		if t.id == rec.id {
			marks = append(marks, "recommended")
		}
		if len(marks) > 0 {
			label += " — " + strings.Join(marks, ", ")
		}
		radio := widget.NewRadioGroup([]string{label}, nil)
		fyne.SetAccessibleLabel(radio, t.label+": "+t.keeps(w.markdown, inBrowser))
		radio.OnChanged = func(s string) {
			if setting {
				return
			}
			if s == "" && i == sel { // the chosen one stays chosen
				setting = true
				radio.SetSelected(label)
				setting = false
				return
			}
			if s != "" {
				ask.choose(i)
			}
		}
		ask.radios = append(ask.radios, radio)
		list = append(list, container.NewVBox(radio, keepsRow(t)))
	}
	ask.choose = func(i int) {
		setting = true
		sel = i
		for j, r := range ask.radios {
			if j == i {
				r.SetSelected(r.Options[0])
			} else {
				r.SetSelected("")
			}
		}
		setting = false
		update()
	}
	ask.chosen = func() fileType { return types[sel] }
	name.OnChanged = func(string) { update() }

	heading := widget.NewLabel("Save as type")
	heading.TextStyle = fyne.TextStyle{Bold: true}
	help := widget.NewButtonWithIcon("About the formats", theme.HelpIcon(), func() { w.showHelp("formats", "save-as-and-rename") })
	help.Importance = widget.LowImportance
	help.Alignment = widget.ButtonAlignLeading
	// a copy to hand on (PDF, Final Draft, Fountain as written) is an
	// export; Clean Fountain joins it there (LOOP.md)
	export := widget.NewButtonWithIcon("A copy for someone else: Export…", theme.DownloadIcon(), func() { w.showExportDialog() })
	export.Importance = widget.LowImportance
	export.Alignment = widget.ButtonAlignLeading
	files := widget.NewLabel("Its files")
	files.TextStyle = fyne.TextStyle{Bold: true}
	parts := []fyne.CanvasObject{heading}
	parts = append(parts, list...)
	parts = append(parts, help, export, widget.NewSeparator())
	if inBrowser { // no system dialog asks it
		parts = append(parts, widget.NewForm(widget.NewFormItem("Name", name)))
	}
	parts = append(parts, result, files, ask.plan.content)
	content := container.NewVScroll(container.NewVBox(parts...))
	ask.scroll = content

	confirm := "Choose Where…"
	if inBrowser {
		confirm = "Save"
	}
	d := dialog.NewCustomConfirm("Save As", confirm, "Cancel", content, func(ok bool) {
		if !ok {
			return
		}
		r := target()
		t := types[sel]
		if inBrowser {
			p, err := browserSavePath(filepath.Base(r.to))
			if err != nil {
				dialog.ShowError(err, w.fyneWindow)
				return
			}
			w.saveAsWith(p, &r)
			return
		}
		w.askWhere(t, filepath.Base(r.to), &r)
	}, w.fyneWindow)
	ask.confirm = d.Confirm
	name.OnSubmitted = func(string) { d.Confirm() }
	ask.choose(sel)
	d.Resize(fitDialog(w.fyneWindow, 600, 760))
	d.Show()
}

// askWhere is the system's save dialog for a type: its filter, the name
// with its extension; then the script is saved there with the chooser's
// choices (a name typed with another type's extension is that type, asked
// about as Save As always did).
func (w *MainWindow) askWhere(t fileType, name string, chosen *relocation) {
	fileDialog := dialog.NewFileSave(func(closer fyne.URIWriteCloser, err error) {
		if err != nil || closer == nil {
			return
		}
		p := closer.URI().Path()
		// written as Save does, safely: not through the dialog's file,
		// which truncates first
		closer.Close()
		if !hasTypeExt(p) { // a bare name: the type's extension, the empty file the dialog made gone
			if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() && st.Size() == 0 {
				os.Remove(p)
			}
			p += t.ext
		}
		if typeOfName(p, w.markdown).id != t.id {
			chosen = nil
		}
		w.saveAsWith(p, chosen)
	}, w.fyneWindow)
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{t.ext}))
	fileDialog.SetFileName(name)
	if w.currentFile != "" {
		if l, err := storage.ListerForURI(storage.NewFileURI(filepath.Dir(w.currentFile))); err == nil {
			fileDialog.SetLocation(l)
		}
	}
	fileDialog.Show()
}
