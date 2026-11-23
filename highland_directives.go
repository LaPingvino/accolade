package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/lexington/bundle"
	"github.com/LaPingvino/lexington/lex"
)

// Highland's directives in Accolade (Joop 2026-11-23: "add the inclusion
// feature and match other features to Highland's codification"): what
// Accolade has an equivalent of reads and writes Highland's notation, so
// that a script moves between the two as it is. Lexington (1.47) prints
// them all, as Highland 2 does: the page view, Preview PDF, Print and
// Export have the included files, the header and footer, {{SERIES}},
// {{TIMESTAMP}}, {{TOC}} and {{PAGENUMBER}}.
//
//   - {{INCLUDE: file}} (relative to the script's file): the line is
//     drawn as a chip naming the file and what it holds; a tap on it, or
//     vim's zo (za) on its line, shows the included text, read only, with
//     Open to edit it. Insert › Include File… writes one.
//   - {{HEADER: …}} and {{FOOTER: …}}: Page Setup's Header and Footer
//     (pagesetup.go). The script's own wins: Page Setup shows it, and
//     what is changed there is changed in the script.
//   - Markers, {{%m: label}}: Highland's own unprinted places to come
//     back to. Accolade's nearest are its notes ([[…]]), which Highland
//     reads and keeps as notes too: both apps already share them, so
//     neither is turned into the other (a note made a marker would lose
//     its text in Accolade's notes; a marker made a note would vanish
//     from Highland's marker list). Accolade shows a marker as a chip,
//     "⚑ label", and goes from one to the next (Next Marker, Previous
//     Marker) or writes one (Marker…), as Highland's ⌃M and ⌃⌥M.

// includeLine is an {{INCLUDE}} line of the script.
type includeLine struct {
	line       int // from 0
	start, end int // the directive, in runes of the text
	path       string
}

// findIncludes are the script body's {{INCLUDE}} lines: a line of
// nothing but the directive (as Lexington and Highland take them).
func findIncludes(body string) []includeLine {
	if !strings.Contains(body, "{{") {
		return nil
	}
	var out []includeLine
	off := 0
	for i, line := range strings.Split(body, "\n") {
		n := utf8.RuneCountInString(line)
		if t := strings.TrimSpace(line); strings.HasPrefix(t, "{{") && strings.HasSuffix(t, "}}") && strings.Count(t, "{{") == 1 {
			if d := lex.ParseDirective(t); d.Kind == lex.DirectiveInclude && d.Path != "" {
				lead := utf8.RuneCountInString(line[:strings.Index(line, "{{")])
				out = append(out, includeLine{line: i, start: off + lead, end: off + lead + utf8.RuneCountInString(t), path: d.Path})
			}
		}
		off += n + 1
	}
	return out
}

// markerAt is a marker in the script: where it is, in runes, and its label.
type markerAt struct {
	start, end int
	label      string
}

// findMarkers are the markers ({{%m: label}}) in the script's body.
func findMarkers(body string) []markerAt {
	if !strings.Contains(body, "{{") {
		return nil
	}
	var out []markerAt
	for _, m := range directiveRe.FindAllStringIndex(body, -1) {
		d := lex.ParseDirective(body[m[0]:m[1]])
		if d.Kind != lex.DirectiveMarker {
			continue
		}
		start := utf8.RuneCountInString(body[:m[0]])
		out = append(out, markerAt{start: start, end: start + utf8.RuneCountInString(body[m[0]:m[1]]), label: d.Label})
	}
	return out
}

// includeFull is the file an include names, from the script's folder; ""
// when the script has none (not saved, or in the browser).
func (w *MainWindow) includeFull(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	if w.currentFile == "" || inBrowser {
		return ""
	}
	return filepath.Join(filepath.Dir(w.currentFile), filepath.FromSlash(p))
}

// readIncludeText is an included file's text: a bundle's (.highland…), or
// the file's own.
func readIncludeText(full string) (string, error) {
	st, err := os.Stat(full)
	switch {
	case err != nil:
		return "", err
	case !st.IsDir() && !st.Mode().IsRegular():
		return "", fmt.Errorf("%s is not a file", filepath.Base(full))
	case st.Size() > maxIncludeSize:
		return "", fmt.Errorf("%s is larger than %d MB", filepath.Base(full), maxIncludeSize>>20)
	}
	if bundle.Is(full) || st.IsDir() {
		b, err := bundle.Read(full)
		if err != nil {
			return "", err
		}
		text, err := b.ReadText()
		return string(text), err
	}
	data, err := os.ReadFile(full)
	return string(bytes.ToValidUTF8(data, []byte("�"))), err
}

// maxIncludeSize is the largest file an include reads.
const maxIncludeSize = 16 << 20

// includeInfo is a chip's words for an included file, as read last: read
// again only when the file's size or time changed.
type includeInfo struct {
	size       int64
	mod        int64
	label, tip string
	missing    bool
}

// includeChipText is the chip's words for the include of path (full on
// disk), from the cache while the file is as it was.
func (w *MainWindow) includeChipText(path, full string) (label, tip string, missing bool) {
	st, err := os.Stat(full)
	if err != nil {
		return includeSummary(path, "", err)
	}
	c := &w.hlChips
	if i, ok := c.files[full]; ok && i.size == st.Size() && i.mod == st.ModTime().UnixNano() {
		return i.label, i.tip, i.missing
	}
	t, err := readIncludeText(full)
	label, tip, missing = includeSummary(path, t, err)
	if c.files == nil {
		c.files = map[string]includeInfo{}
	}
	c.files[full] = includeInfo{size: st.Size(), mod: st.ModTime().UnixNano(), label: label, tip: tip, missing: missing}
	return label, tip, missing
}

// includeSummary is what a chip says of an included file: its name, and
// its scenes (or lines); missing when it cannot be read.
func includeSummary(path, text string, err error) (label, tip string, missing bool) {
	name := filepath.Base(filepath.FromSlash(path))
	if err != nil {
		return "⤵ " + name + " · not found", "Included file " + path + " cannot be read: " + err.Error(), true
	}
	scenes := 0
	for _, l := range strings.Split(text, "\n") {
		if isSceneHeadingLine(l) {
			scenes++
		}
	}
	what := fmt.Sprintf("%d lines", strings.Count(strings.TrimRight(text, "\n"), "\n")+1)
	if scenes > 0 {
		what = fmt.Sprintf("%d scenes", scenes)
		if scenes == 1 {
			what = "1 scene"
		}
	}
	first := strings.TrimSpace(text)
	if i := strings.Index(first, "\n"); i >= 0 {
		first = first[:i]
	}
	return "⤵ " + name + " · " + what, "Included: " + path + " — " + first + " (tap to see it)", false
}

// isSceneHeadingLine tells, roughly, whether a line is a scene heading
// (for a count only).
func isSceneHeadingLine(l string) bool {
	u := strings.ToUpper(strings.TrimSpace(l))
	for _, p := range []string{"INT.", "EXT.", "INT/EXT", "I/E", "EST.", "INT ", "EXT "} {
		if strings.HasPrefix(u, p) {
			return true
		}
	}
	return strings.HasPrefix(u, ".") && len(u) > 1 && u[1] != '.'
}

// highlandCache is the script's include and marker chips as found last.
type highlandCache struct {
	text  string
	cur   string // the script's file then
	chips []editor.Chip
	// merged with the links' (editorChips): the links' and these chips
	// they were merged from
	merged, links, of []editor.Chip
	files             map[string]includeInfo // the included files' words, by their paths
}

// directiveRe is one of Highland's directives, {{…}}.
var directiveRe = regexp.MustCompile(`\{\{[^{}]*\}\}`)

// highlandChips are the include lines and markers of the script as chips
// (sorted, the same slice for the same text).
func (w *MainWindow) highlandChips(text string) []editor.Chip {
	if w.markdown {
		return nil
	}
	c := &w.hlChips
	if c.chips != nil && sameString(c.text, text) && c.cur == w.currentFile {
		return c.chips
	}
	body := text[:bodyEndByte(text)]
	var out []editor.Chip
	for _, inc := range findIncludes(body) {
		full := w.includeFull(inc.path)
		var label, tip string
		var missing bool
		if full == "" {
			label, tip, missing = "⤵ "+filepath.Base(inc.path)+" · save the script to read it", "Included: "+inc.path+" (relative to the script's file)", true
		} else {
			label, tip, missing = w.includeChipText(inc.path, full)
		}
		out = append(out, editor.Chip{Start: inc.start, End: inc.end, Label: label, Tip: tip, Warn: missing, Tap: true})
	}
	for _, m := range findMarkers(body) {
		label := "⚑"
		if m.label != "" {
			label += " " + m.label
		}
		out = append(out, editor.Chip{Start: m.start, End: m.end, Label: label, Tip: "Highland's marker (not printed): " + m.label})
	}
	slices.SortFunc(out, func(a, b editor.Chip) int { return a.Start - b.Start })
	if out == nil {
		out = []editor.Chip{}
	}
	c.text, c.cur, c.chips = text, w.currentFile, out
	return out
}

// mergeChips is two sorted lists of chips as one, sorted, without those
// of b that overlap a's.
func mergeChips(a, b []editor.Chip) []editor.Chip {
	if len(b) == 0 {
		return a
	}
	if len(a) == 0 {
		return b
	}
	out := slices.Clone(a)
	for _, c := range b {
		if !slices.ContainsFunc(a, func(x editor.Chip) bool { return c.Start < x.End && x.Start < c.End }) {
			out = append(out, c)
		}
	}
	slices.SortFunc(out, func(x, y editor.Chip) int { return x.Start - y.Start })
	return out
}

// chipTapped is a tap on a chip that takes one: an include's shows the
// file, a frame's its picture.
func (w *MainWindow) chipTapped(c editor.Chip, at fyne.Position, cell fyne.Size) {
	text := w.textEditor.Text()
	for _, inc := range findIncludes(text[:bodyEndByte(text)]) {
		if inc.start == c.Start {
			w.showInclude(inc)
			return
		}
	}
	w.frameTapped(c, at, cell)
}

// includeOnLine is the include on a line of the script, if it is one.
func (w *MainWindow) includeOnLine(line int) (includeLine, bool) {
	if w.markdown {
		return includeLine{}, false
	}
	text := w.textEditor.Text()
	for _, inc := range findIncludes(text[:bodyEndByte(text)]) {
		if inc.line == line {
			return inc, true
		}
	}
	return includeLine{}, false
}

// showInclude shows an included file's text, read only, with Open to edit
// it (in a window of its own).
func (w *MainWindow) showInclude(inc includeLine) {
	full := w.includeFull(inc.path)
	var body fyne.CanvasObject
	var open *widget.Button
	if full == "" {
		body = widget.NewLabel("The script is not saved: " + inc.path + " is read from the script's folder once it is.")
	} else if text, err := readIncludeText(full); err != nil {
		l := widget.NewLabel(inc.path + " cannot be read: " + err.Error() + "\n\nIt is read from the script's folder, " + filepath.Dir(full) + ". " +
			"Printed, the script has a line saying it is not there.")
		l.Wrapping = fyne.TextWrapWord
		body = l
	} else {
		l := widget.NewLabel(text)
		l.TextStyle = fyne.TextStyle{Monospace: true}
		l.Wrapping = fyne.TextWrapWord
		body = container.NewVScroll(l)
		open = widget.NewButtonWithIcon("Open", theme.FolderOpenIcon(), nil)
	}
	var d *dialog.CustomDialog
	closeB := widget.NewButton("Close", func() { d.Hide() })
	buttons := []*widget.Button{}
	if open != nil {
		open.OnTapped = func() {
			d.Hide()
			w.openPath(full)
		}
		open.Importance = widget.HighImportance
		buttons = append(buttons, open)
	}
	note := widget.NewLabel("Included where the line {{INCLUDE: " + inc.path + "}} is: printed with the script, not part of its text. Open it to edit it.")
	note.Wrapping = fyne.TextWrapWord
	note.Importance = widget.LowImportance
	d = dialog.NewCustomWithoutButtons("Included: "+filepath.Base(inc.path),
		container.NewBorder(note, w.dialogButtons(closeB, buttons...), nil, nil, body), w.fyneWindow)
	d.Resize(fitDialog(w.fyneWindow, 620, 520))
	d.SetOnEscape(d.Hide)
	d.Show()
	w.includeShown = d
}

// askInclude is Insert › Include File…: a file chosen, its line
// {{INCLUDE: path}} written under the cursor's, the path relative to the
// script's folder (as Highland has it).
func (w *MainWindow) askInclude() {
	chooseFileIn(w.fyneWindow, []string{".fountain", ".highland", ".txt", ".spmd", ".fdx"}, func(picked string) {
		w.textEditor.Edit(func(b *buffer.Buffer) { insertInclude(b, w.includeRel(picked)) })
		w.setStatus("Included: " + filepath.Base(picked) + " (printed where its line is)")
		w.focusMain()
	})
}

// includeRel is the path an include of the file p is written with:
// relative to the script's folder, with /, else as it is.
func (w *MainWindow) includeRel(p string) string {
	if w.currentFile != "" {
		if rel, err := filepath.Rel(filepath.Dir(w.currentFile), p); err == nil {
			return filepath.ToSlash(rel)
		}
	}
	return p
}

// insertInclude writes the include of rel on a line of its own, after
// the cursor's line (on it, when it is empty), blank lines around it.
func insertInclude(b *buffer.Buffer, rel string) {
	insertOwnLine(b, "{{INCLUDE: "+rel+"}}")
}

// insertOwnLine writes line as a paragraph of its own after the cursor's
// line (on it, when it is empty), and puts the cursor at its end.
func insertOwnLine(b *buffer.Buffer, line string) {
	text := []rune(b.Text())
	c := b.Cursor()
	start, end := c, c
	for start > 0 && text[start-1] != '\n' {
		start--
	}
	for end < len(text) && text[end] != '\n' {
		end++
	}
	if strings.TrimSpace(string(text[start:end])) == "" {
		b.SetCursor(end, false)
		ins := line
		if start > 0 && start >= 2 && text[start-2] != '\n' {
			ins = "\n" + ins
		}
		if end < len(text)-1 && text[end+1] != '\n' {
			ins += "\n"
		}
		b.Insert(ins)
		return
	}
	b.SetCursor(end, false)
	ins := "\n\n" + line
	if end < len(text)-1 && text[end+1] != '\n' {
		ins += "\n"
	}
	b.Insert(ins)
	b.SetCursor(end+len([]rune("\n\n"+line)), false)
}

// askMarker is Marker…: Highland's marker, {{%m: label}}, at the end of
// the cursor's line.
func (w *MainWindow) askMarker() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("What to come back to")
	d := dialog.NewForm("Marker", "Insert", "Cancel", []*widget.FormItem{widget.NewFormItem("Label", entry)}, func(ok bool) {
		if !ok {
			return
		}
		label := strings.NewReplacer("{", "(", "}", ")").Replace(strings.TrimSpace(entry.Text))
		w.textEditor.Edit(func(b *buffer.Buffer) {
			text := []rune(b.Text())
			end := b.Cursor()
			for end < len(text) && text[end] != '\n' {
				end++
			}
			b.SetCursor(end, false)
			m := "{{%m: " + label + "}}"
			if label == "" {
				m = "{{%m}}"
			}
			if end > 0 && text[end-1] != '\n' && text[end-1] != ' ' {
				m = " " + m
			}
			b.Insert(m)
		})
		w.focusMain()
	}, w.fyneWindow)
	d.Show()
	w.fyneWindow.Canvas().Focus(entry)
}

// gotoMarker moves the cursor to the next marker after it (or with back,
// the one before), around the script's end.
func (w *MainWindow) gotoMarker(back bool) {
	text := w.textEditor.Text()
	ms := findMarkers(text[:bodyEndByte(text)])
	if len(ms) == 0 {
		w.setStatus("No markers ({{%m: …}}) in the script")
		return
	}
	c := w.textEditor.CursorOffset()
	to := -1
	if back {
		for i := len(ms) - 1; i >= 0; i-- {
			if ms[i].end < c {
				to = ms[i].start
				break
			}
		}
		if to < 0 {
			to = ms[len(ms)-1].start
		}
	} else {
		for _, m := range ms {
			if m.start > c {
				to = m.start
				break
			}
		}
		if to < 0 {
			to = ms[0].start
		}
	}
	w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(to, false) })
	w.focusMain()
}

// noIncludeFolder is where an unsaved script's includes are looked for:
// nowhere (they print as not found), not the working folder.
var noIncludeFolder = filepath.Join(string(filepath.Separator), "nonexistent-accolade-folder", "script")

// Page Setup's header and footer (pagesetup.go) and the script's own.

// scriptPageText is the script's own {{HEADER: …}} (or, footer, its
// {{FOOTER: …}}): its value, and where the directive is in runes of text;
// ok false for none.
func scriptPageText(text string, footer bool) (value string, start, end int, ok bool) {
	body := text[:bodyEndByte(text)]
	if !strings.Contains(body, "{{") {
		return "", 0, 0, false
	}
	want := lex.DirectiveHeader
	if footer {
		want = lex.DirectiveFooter
	}
	for _, m := range directiveRe.FindAllStringIndex(body, -1) {
		if d := lex.ParseDirective(body[m[0]:m[1]]); d.Kind == want {
			s := utf8.RuneCountInString(body[:m[0]])
			return d.Value, s, s + utf8.RuneCountInString(body[m[0]:m[1]]), true
		}
	}
	return "", 0, 0, false
}

// pageTexts are the header and footer the script prints with under p:
// its own, else p's.
func (w *MainWindow) pageTexts(p pageSetup) (header, footer string) {
	header, footer = strings.TrimSpace(p.header), strings.TrimSpace(p.footer)
	if w.textEditor == nil {
		return header, footer
	}
	text := w.textEditor.Text()
	if v, _, _, ok := scriptPageText(text, false); ok {
		header = v
	}
	if v, _, _, ok := scriptPageText(text, true); ok {
		footer = v
	}
	return header, footer
}

// pageTextWidgets are Page Setup's Header and Footer, with how to write
// them.
func (p *pageSetupDlg) pageTextWidgets() fyne.CanvasObject {
	p.pageTextNote = widget.NewLabel("")
	p.pageTextNote.Wrapping = fyne.TextWrapWord
	p.pageTextNote.Importance = widget.LowImportance
	return container.NewVBox(labelled("Header:", p.headerEntry), labelled("Footer:", p.footerEntry), p.pageTextNote)
}

// pageTextHelp is how a header is written, Highland's way.
const pageTextHelp = "As Highland writes them: Left | Centre | Right (one part: a header's right, a footer's centre); " +
	"%p the page, %pp the last page, %# the section, %none nothing. A header takes the page number's place."

// loadPageTexts shows the script's own header and footer, else s's.
func (p *pageSetupDlg) loadPageTexts(s pageSetup) {
	text := ""
	if p.w.textEditor != nil {
		text = p.w.textEditor.Text()
	}
	var own []string
	h, _, _, hOwn := scriptPageText(text, false)
	f, _, _, fOwn := scriptPageText(text, true)
	p.headerOwn, p.footerOwn = hOwn, fOwn
	if !hOwn {
		h = s.header
	} else {
		own = append(own, "{{HEADER}}")
	}
	if !fOwn {
		f = s.footer
	} else {
		own = append(own, "{{FOOTER}}")
	}
	p.headerEntry.SetText(h)
	p.footerEntry.SetText(f)
	note := pageTextHelp
	if len(own) > 0 {
		note = "The script's own " + strings.Join(own, " and ") + " (it wins over the setting): changed here, it is changed in the script. " + note
	}
	p.pageTextNote.SetText(note)
}

// settingText is a header's or footer's setting as the dialog leaves it:
// the entry's, or with the script's own shown, the setting as it was.
func (p *pageSetupDlg) settingText(e *widget.Entry, own bool, pref string) string {
	if own {
		return p.w.settings.GetString(pref)
	}
	return strings.TrimSpace(e.Text)
}

// savePageTexts writes the script's own header and footer, as changed in
// the dialog, into the script (emptied: taken out, with its line if it
// is alone on it).
func (p *pageSetupDlg) savePageTexts() {
	for _, f := range []bool{true, false} { // the footer first: it is after the header, usually
		e, own, name := p.headerEntry, p.headerOwn, "HEADER"
		if f {
			e, own, name = p.footerEntry, p.footerOwn, "FOOTER"
		}
		if !own {
			continue
		}
		v, start, end, ok := scriptPageText(p.w.textEditor.Text(), f)
		nv := strings.TrimSpace(e.Text)
		if !ok || nv == v {
			continue
		}
		p.w.textEditor.Edit(func(b *buffer.Buffer) { replacePageText(b, start, end, name, nv) })
	}
}

// replacePageText puts {{name: value}} in place of the directive at
// [start, end) (runes), or with value "" takes it out (its line too, if
// nothing else is on it, and a blank line after it).
func replacePageText(b *buffer.Buffer, start, end int, name, value string) {
	text := []rune(b.Text())
	if value != "" {
		b.Replace(start, end, "{{"+name+": "+value+"}}")
		return
	}
	ls, le := start, end
	for ls > 0 && text[ls-1] != '\n' {
		ls--
	}
	for le < len(text) && text[le] != '\n' {
		le++
	}
	if strings.TrimSpace(string(text[ls:start])+string(text[end:le])) == "" {
		start, end = ls, min(le+1, len(text))
		if end < len(text) && text[end] == '\n' && (ls == 0 || ls >= 2 && text[ls-2] == '\n') {
			end++ // the blank line after it
		}
	}
	b.Replace(start, end, "")
}
