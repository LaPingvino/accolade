package main

import (
	"cmp"
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// A scriptTitle is the script's name over the editor; clicked, it opens
// the rename editor.
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

// planPanel is the list of the script's files and where each goes, with
// the choices that change it: where a bundle's drafts go when it becomes
// a plain file, and whether the source beside it follows the new name.
// The rename editor and Save As show the same one.
type planPanel struct {
	w        *MainWindow
	rename   bool
	rows     *fyne.Container
	how      *widget.Select
	hows     []flattenChoice
	keepName *widget.Check
	// copy: an .accolade in another language keeps text.fountain too
	copy     *widget.Check
	content  *fyne.Container
	onChange func()
}

func (w *MainWindow) newPlanPanel(rename bool, onChange func()) *planPanel {
	p := &planPanel{w: w, rename: rename, rows: container.NewVBox(), onChange: onChange}
	p.hows = flattenChoices(inBrowser)
	if rename { // the bundle goes: nothing keeps what is left out
		p.hows = p.hows[:len(p.hows)-1]
	}
	labels := make([]string, len(p.hows))
	for i, c := range p.hows {
		labels[i] = map[flattenChoice]string{
			flattenKept: "Kept by Accolade", flattenFolder: "In a folder beside it", flattenDrop: "Left out",
		}[c]
	}
	p.how = widget.NewSelect(labels, nil)
	p.how.SetSelectedIndex(0)
	p.how.OnChanged = func(string) { p.changed() }
	fyne.SetAccessibleLabel(p.how, "Where the drafts and production files go")
	p.keepName = widget.NewCheck("Keep the source's own name", func(bool) { p.changed() })
	p.copy = widget.NewCheck("Keep a plain Fountain copy inside (text.fountain)", nil)
	p.copy.Checked = !w.noFountainCopy
	p.copy.OnChanged = func(bool) { p.changed() }
	p.content = container.NewVBox(p.rows)
	return p
}

func (p *planPanel) changed() {
	if p.onChange != nil {
		p.onChange()
	}
}

// fill sets r's choices from the panel's.
func (p *planPanel) fill(r *relocation) {
	r.how = flattenKept
	if i := p.how.SelectedIndex(); i >= 0 && i < len(p.hows) {
		r.how = p.hows[i]
	}
	r.keepBesideName = p.keepName.Checked
	noCopy := !p.copy.Checked
	r.noCopy = &noCopy
}

// show shows r's plan, and the choices that matter for it.
func (p *planPanel) show(r relocation) {
	w := p.w
	p.rows.RemoveAll()
	for _, row := range w.planRows(r) {
		what := widget.NewLabel(row.what)
		what.TextStyle = fyne.TextStyle{Bold: true}
		if row.fixed {
			what.SetText(row.what + "  · name fixed by the format")
			what.TextStyle = fyne.TextStyle{Bold: true, Italic: true}
		}
		what.Wrapping = fyne.TextWrapWord
		where := widget.NewLabel(row.now + "  →  " + row.after)
		where.Wrapping = fyne.TextWrapWord
		if row.warn {
			where.Importance = widget.WarningImportance
		}
		p.rows.Add(container.NewVBox(what, where))
	}
	fromKind, toKind := w.kinds(r)
	var extra []fyne.CanvasObject
	if lang, _ := w.langAfter(r); !w.markdown && w.lex().dialect(lang) && strings.EqualFold(filepath.Ext(r.to), typeAccolade.ext) {
		extra = append(extra, p.copy)
	}
	if fromKind != notBundle && toKind == notBundle && w.bundle != nil && len(w.bundle.extras) > 0 {
		extra = append(extra, widget.NewLabel("Its drafts and production files go:"), p.how)
	}
	if !w.markdown && toKind == notBundle && r.from != "" {
		if ref := w.sourceRef(); ref.Origin == sourceBeside {
			p.keepName.Text = "Keep the source's own name (" + filepath.Base(ref.Path) + ")"
			p.keepName.Refresh()
			theirs := strings.TrimSuffix(r.to, filepath.Ext(r.to)) + filepath.Ext(ref.Path)
			if absPath(theirs) != absPath(ref.Path) {
				extra = append(extra, p.keepName)
			}
		}
	}
	p.content.Objects = append([]fyne.CanvasObject{p.rows}, extra...)
	p.content.Refresh()
}

// renameScript opens the rename editor: the base name, the script's
// language and type, the resulting name live, and every file that
// belongs to the script with where it goes. A full name typed with its
// extension is taken as it is. The language is the script's: a plain
// file's name says it as a dialect of Fountain (film.eo.fountain) if that
// type is picked; a bundle keeps it in info.json, its name plain.
func (w *MainWindow) renameScript() {
	old := cmp.Or(w.scriptName(), w.untitledName())
	parts := splitScriptName(old)
	current := typeOfName(old, w.markdown)
	if w.bundle == nil && w.isPlainBundle(w.currentFile) {
		current = typeFountain
	}

	base := widget.NewEntry()
	base.SetText(parts.base)
	fyne.SetAccessibleLabel(base, "Name")
	langs := w.lex().languages()
	langLabels := make([]string, len(langs))
	for i, l := range langs {
		langLabels[i] = languageLabel(l)
	}
	lang := widget.NewSelect(langLabels, nil)
	fyne.SetAccessibleLabel(lang, "Language")
	kind := widget.NewSelect(nil, nil)
	fyne.SetAccessibleLabel(kind, "Type")
	var types []fileType
	// the types for a language: its dialect among them; the type picked
	// stays (a dialect another language's)
	setTypes := func(l string, keep fileType) {
		types = scriptTypes(w.lex(), w.markdown, inBrowser, old, l)
		labels := make([]string, len(types))
		for i, t := range types {
			labels[i] = t.label + " (" + t.shownExt() + ")"
		}
		kind.Options = labels
		if keep.id == "dialect" {
			keep = dialectType(l)
			if !w.lex().dialect(l) && languageOf(old) != l {
				keep = typeFountain
			}
		}
		if i := indexOfType(types, keep); i >= 0 {
			kind.SetSelectedIndex(i)
		} else {
			kind.ClearSelected()
		}
		kind.Refresh()
	}
	keeps := widget.NewLabel("")
	keeps.Wrapping = fyne.TextWrapWord
	result := widget.NewLabel("")
	result.TextStyle = fyne.TextStyle{Bold: true}
	result.Wrapping = fyne.TextWrapBreak
	where := widget.NewLabel("")
	where.Wrapping = fyne.TextWrapWord
	if w.currentFile == "" {
		where.SetText("Not saved yet: this is the name it will be saved by.")
	} else {
		where.SetText("In " + filepath.Dir(w.currentFile) + ".")
	}

	picked := func() (string, fileType) {
		l, t := w.language, current
		if i := lang.SelectedIndex(); i >= 0 && !w.markdown {
			l = langs[i]
		}
		if i := kind.SelectedIndex(); i >= 0 && i < len(types) {
			t = types[i]
		}
		return l, t
	}
	var plan *planPanel
	var d *dialog.ConfirmDialog
	setting := false
	name := func() string {
		_, t := picked()
		return composedName(base.Text, t)
	}
	target := func() relocation {
		n := renamedName(name(), old)
		r := relocation{rename: true, from: w.currentFile, to: n}
		if l, t := picked(); l != w.language || t.id == "dialect" { // said only by a choice
			r.lang = l
		}
		if w.currentFile != "" && n != "" {
			r.to = filepath.Join(filepath.Dir(w.currentFile), n)
		}
		plan.fill(&r)
		return r
	}
	update := func() {
		if setting {
			return
		}
		typed := strings.TrimSpace(base.Text)
		if hasTypeExt(typed) { // a full name: the pickers follow it
			setting = true
			p := splitScriptName(typed)
			l, _ := picked()
			if p.lang != "" {
				l = p.lang
				lang.SetSelectedIndex(max(0, indexOf(langs, l)))
			}
			setTypes(l, typeOfName(typed, w.markdown))
			setting = false
		}
		l, t := picked()
		keeps.SetText(t.keeps(w.lex(), w.markdown, inBrowser, l))
		n := renamedName(name(), old)
		if n == "" {
			result.SetText("Becomes: (no name yet)")
			plan.rows.RemoveAll()
			return
		}
		r := target()
		becomes := "Becomes: " + n
		if lang, _ := w.langAfter(r); !w.markdown && !isMarkdownFile(n) && languageOf(n) == "" &&
			languageHome(w.lex(), n, lang, w.languageSaid || r.lang != "", bundleKindOf(n)) != "" {
			becomes += "  (" + languageName(lang) + " inside)" // the Language row says where
		}
		result.SetText(becomes)
		plan.show(r)
	}
	plan = w.newPlanPanel(true, update)
	base.OnChanged = func(string) { update() }
	lang.OnChanged = func(string) {
		if setting {
			return
		}
		setting = true
		l, t := picked()
		setTypes(l, t)
		setting = false
		update()
	}
	kind.OnChanged = func(string) { update() }

	form := widget.NewForm(widget.NewFormItem("Name", base))
	if !w.markdown { // a document has no script language
		form.Append("Language", lang)
	}
	form.Append("Type", kind)
	files := widget.NewLabel("Its files")
	files.TextStyle = fyne.TextStyle{Bold: true}
	content := container.NewVBox(form, keeps, result, where, widget.NewSeparator(), files, plan.content)
	scroll := container.NewVScroll(content)
	title := "Rename Script"
	if w.markdown {
		title = "Rename Document"
	}
	d = dialog.NewCustomConfirm(title, "Rename", "Cancel", scroll, func(ok bool) {
		if !ok {
			return
		}
		r := target()
		if r.to == "" {
			dialog.ShowError(fmt.Errorf("that is no name for a file"), w.fyneWindow)
			return
		}
		w.doRename(r)
	}, w.fyneWindow)
	setting = true
	lang.SetSelectedIndex(max(0, indexOf(langs, w.language)))
	setTypes(w.language, current)
	setting = false
	update()
	w.renameAsked = &renameAsk{base: base, lang: lang, kind: kind, plan: plan, confirm: d.Confirm, result: result, scroll: scroll}
	base.OnSubmitted = func(string) { d.Confirm() }
	d.Resize(fitDialog(w.fyneWindow, 560, 1000))
	d.Show()
	w.fyneWindow.Canvas().Focus(base)
}

// renameAsk is the rename editor last opened (for the tests).
type renameAsk struct {
	base       *widget.Entry
	lang, kind *widget.Select
	plan       *planPanel
	result     *widget.Label
	scroll     *container.Scroll
	confirm    func()
}

func indexOf(ss []string, s string) int {
	for i, x := range ss {
		if x == s {
			return i
		}
	}
	return -1
}

func indexOfType(ts []fileType, t fileType) int {
	for i, x := range ts {
		if x.id == t.id && (t.id != "asis" || x.ext == t.ext) && x.lang == t.lang {
			return i
		}
	}
	return -1
}

// doRename carries out a rename the editor asked for, saying how it went.
func (w *MainWindow) doRename(r relocation) {
	from := w.scriptName()
	err := w.relocate(r, func(err error) {
		if err != nil {
			dialog.ShowError(err, w.fyneWindow)
			return
		}
		switch {
		case from != "" && r.from != "" && filepath.Base(r.from) == filepath.Base(r.to):
			w.setStatus("Saved in " + languageName(w.language) + ": " + filepath.Base(r.to))
		case from != "" && r.from != "":
			w.setStatus("Renamed " + from + " to " + filepath.Base(r.to) + " — File > Undo Rename takes it back")
		}
	})
	if err != nil {
		dialog.ShowError(err, w.fyneWindow)
	}
}

// renameTo gives the script the name name: its file and every file
// belonging to it renamed together (relocate.go), converted to or from a
// bundle as the name says; an unsaved script will be saved by it.
func (w *MainWindow) renameTo(name string) error {
	if name == "" {
		return fmt.Errorf("that is no name for a file")
	}
	r := relocation{rename: true, from: w.currentFile, to: name, how: w.flattenTo}
	if w.currentFile != "" {
		r.to = filepath.Join(filepath.Dir(w.currentFile), name)
	}
	var later error
	err := w.relocate(r, func(err error) { later = err })
	return cmp.Or(err, later)
}
