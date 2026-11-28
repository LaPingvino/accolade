// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"

	"github.com/LaPingvino/lexington/rules"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The contextual ribbon tabs (LOOP.md, CONTEXTUAL RIBBON TABS), each
// like the Preview tab (previewtab.go): there only while its context is.
//
//   - Fix-Up, after a script is imported from a PDF or Final Draft, until
//     it is saved or Done Fixing: Fix Lines Like This…, the rules the
//     script keeps applied again, and for a PDF reading it again with OCR
//     and the garbled-text check. It comes to the front as the import
//     comes (that is the work then). A Word document imports as a
//     Markdown document (File > Import as Markdown…), where the fixer,
//     a screenplay's, does not apply: it has no Fix-Up tab.
//   - Title Page, while the cursor is in the title page: a key to add
//     (Title, Credit, Author, Source, Draft date, Contact, Notes,
//     Copyright), in the script's language's words where Lexington has
//     them (العنوان in Arabic), the Title Page… dialog, and To the Script.
//     It does not come to the front by itself (a new script's cursor
//     starts in its title page): it is there, beside the others.

// contextEnv fills in what the contextual tabs' rules look at.
func (w *MainWindow) contextEnv(e *cmdEnv) {
	e.fixup = w.fixupFrom != "" && !w.markdown
	e.fixupPDF = e.fixup && strings.EqualFold(filepath.Ext(w.fixupFrom), ".pdf")
	if e.fixup && w.textEditor != nil {
		e.fixupRules = len(fixRulesOf(w.textEditor.Text())) > 0
	}
	e.titlePage = w.inTitlePage && !w.markdown && !e.away
}

// The rules of the contextual tabs' commands.
func fixingUp(e cmdEnv) bool     { return e.fixup }
func fixingUpPDF(e cmdEnv) bool  { return e.fixupPDF }
func keptFixRules(e cmdEnv) bool { return e.fixupRules }
func inTitlePage(e cmdEnv) bool  { return e.titlePage }

// startFixUp: an import is in the window, from path (its Fix-Up tab).
func (w *MainWindow) startFixUp(path string) {
	w.fixupFrom = path
	w.ribbonRefresh()
}

// endFixUp: saved, or done fixing (the tab goes).
func (w *MainWindow) endFixUp() {
	if w.fixupFrom == "" {
		return
	}
	w.fixupFrom = ""
	w.ribbonRefresh()
}

// doneFixing is Done Fixing: the Fix-Up tab goes (its commands stay in
// Edit > Fix Lines Like This…).
func (w *MainWindow) doneFixing() {
	w.endFixUp()
	w.setStatus("Done fixing up: Fix Lines Like This… stays in the Edit menu and on the Edit tab")
}

// applyKeptRules applies the rules the script keeps again, asking first.
func (w *MainWindow) applyKeptRules() {
	rs := fixRulesOf(w.textEditor.Text())
	if len(rs) == 0 {
		w.setStatus("The script keeps no fix-up rules yet: Fix Lines Like This… makes one")
		return
	}
	w.askApplySaved(rs, nil)
}

// pdfOfImport is the PDF the script was imported from, if it is still
// there ("" and a status if not).
func (w *MainWindow) pdfOfImport() string {
	p := w.fixupFrom
	if p == "" || !strings.EqualFold(filepath.Ext(p), ".pdf") {
		w.setStatus("This script was not imported from a PDF")
		return ""
	}
	if _, err := os.Stat(p); err != nil {
		w.setStatus(filepath.Base(p) + " is not there any more: it cannot be read again")
		return ""
	}
	return p
}

// checkGarbled says how the imported PDF's pages read: scanned (no text
// of their own), with text that looks garbled (a poor OCR's), or clean.
func (w *MainWindow) checkGarbled() {
	p := w.pdfOfImport()
	if p == "" {
		return
	}
	w.setStatus(garbledReport(filepath.Base(p), pdfPagesOf(p)))
}

// garbledReport is the garbled-text check's words for n.
func garbledReport(name string, n pdfPages) string {
	if n.pages == 0 {
		return name + ": its pages could not be read"
	}
	if n.scanned+n.garbled == 0 {
		return fmt.Sprintf("%s: all %s have clean text of their own — nothing for OCR to read again", name, plural(n.pages, "page"))
	}
	var parts []string
	if n.scanned > 0 {
		parts = append(parts, plural(n.scanned, "scanned page")+" (no text: read with OCR)")
	}
	if n.garbled > 0 {
		parts = append(parts, plural(n.garbled, "page")+" whose text looks garbled")
	}
	return fmt.Sprintf("%s: of %s, %s — Read with OCR Again… reads them with OCR", name, plural(n.pages, "page"), strings.Join(parts, ", "))
}

// ocrAgain reads the imported PDF again with OCR, its scans and the pages
// whose text looks garbled, asking the language first; the fixes made
// since are replaced (asked first), the kept rules offered again.
func (w *MainWindow) ocrAgain() {
	p := w.pdfOfImport()
	if p == "" {
		return
	}
	n := pdfPagesOf(p)
	langs, tesseract := ocrLanguages()
	if n.scanned+n.garbled == 0 {
		w.setStatus(garbledReport(filepath.Base(p), n))
		return
	}
	if len(langs) == 0 {
		w.setStatus("There is no OCR here to read the scanned pages with")
		return
	}
	read := func() {
		w.askOCRLanguage(n, w.guessOCRLanguage(languageOf(p), langs), langs, tesseract,
			func(lang string, ocrText bool) { w.readPDF(p, lang, lang, ocrText || n.garbled > 0) },
			func() { w.readPDF(p, languageOf(p), "", false) })
	}
	if w.textEditor.Text() == w.filledWith {
		read()
		return
	}
	dialog.ShowConfirm("Read the PDF Again",
		"The script as changed since the import is replaced by the PDF read again. The rules kept in its Fix-Up block are offered again after it.",
		func(ok bool) {
			if ok {
				read()
			}
		}, w.fyneWindow)
}

// fixUpTab is the Fix-Up tab's layout.
func fixUpTab() ribbonTab {
	return ribbonTab{title: "Fix-Up", contextual: true, forward: true, when: fixingUp, groups: []ribbonGroup{
		{title: "Fix", homes: []home{big("edit.fixLines"), at("fixup.applyRules")}},
		{title: "PDF", when: fixingUpPDF, homes: []home{big("fixup.ocrAgain"), at("fixup.checkText")}},
		{title: "Done", homes: []home{big("file.save"), big("fixup.done")}},
	}}
}

// The title page.

// titleKeys are the keys the Title Page tab adds, the standard ones.
var titleKeys = []string{"Title", "Credit", "Author", "Source", "Draft date", "Contact", "Notes", "Copyright"}

// titleKeyMore: the key's value is lines of its own under it.
func titleKeyMore(std string) bool { return std == "Contact" || std == "Notes" }

// titleKeyIn is std's key in language l: std itself if l writes it as it
// is (hi: English first, not a mix of its scripts' words), else its own,
// the shortest, else std.
func titleKeyIn(l rules.Language, std string) string {
	if l.TitleKeys[std] == std {
		return std
	}
	var own []string
	for k, v := range l.TitleKeys {
		if v == std {
			own = append(own, k)
		}
	}
	if len(own) == 0 {
		return std
	}
	slices.SortFunc(own, func(a, b string) int {
		if d := len([]rune(a)) - len([]rune(b)); d != 0 {
			return d
		}
		return strings.Compare(a, b)
	})
	return own[0]
}

// titleKeyHere is std's key in the script's language.
func (w *MainWindow) titleKeyHere(std string) string {
	if w.settings == nil {
		return std
	}
	return titleKeyIn(w.lex().Language(w.language), std)
}

// titleKeyIs reports whether key is std (its own word or the language's).
func titleKeyIs(l rules.Language, key, std string) bool {
	key = strings.TrimSpace(key)
	return strings.EqualFold(key, std) || l.TitleKeys[key] == std
}

// titlePageLines is how many lines the title page takes in text: its
// fields to the first blank line (one just added, its value not typed yet,
// counted in), 0 if the script has none.
func titlePageLines(text string) int {
	head := text
	if i := strings.Index(text, "\n\n"); i >= 0 {
		head = text[:i]
	}
	lines := strings.Split(head, "\n")
	if len(lines) == 0 || !isTitleField(lines[0], lines, 0) {
		return 0
	}
	return len(lines)
}

// followTitlePage sees whether the cursor is in the title page (the
// Title Page tab), the ribbon made again only when that changes.
func (w *MainWindow) followTitlePage(line int) {
	in := w.currentElement == "Title Page"
	if !in && !w.markdown && w.textEditor != nil && line < 60 {
		in = line < titlePageLines(w.textEditor.Text())
	}
	if in != w.inTitlePage {
		w.inTitlePage = in
		w.ribbonFollow()
	}
}

// addTitleKey adds std's key to the title page, in the script's
// language, after the field the cursor is on; the cursor after it to
// type its value. One there already: the cursor goes to it.
func (w *MainWindow) addTitleKey(std string) {
	if w.scriptAway("Title Page") {
		return
	}
	e := w.textEditor
	l := w.lex().Language(w.language)
	key := w.titleKeyHere(std)
	text := e.Text()
	lines := strings.Split(text, "\n")
	end := max(titlePageLines(text), titlePageEnd(lines))
	offset := func(line int) int { // the rune offset of line's start
		n := 0
		for _, s := range lines[:line] {
			n += len([]rune(s)) + 1
		}
		return n
	}
	for i := 0; i < end && i < len(lines); i++ {
		if k, _, ok := strings.Cut(lines[i], ":"); ok && !strings.HasPrefix(lines[i], " ") && !strings.HasPrefix(lines[i], "\t") && titleKeyIs(l, k, std) {
			e.SetCursorOffset(offset(i) + len([]rune(lines[i])))
			w.focusMain()
			w.setStatus("The title page has " + key + " already: the cursor is on it")
			return
		}
	}
	line, _ := e.Buffer().Position(e.CursorOffset())
	at := min(line, end)
	if at < len(lines) && strings.TrimSpace(lines[at]) != "" {
		at++
		for at < len(lines) && at < end && (strings.HasPrefix(lines[at], "\t") || strings.HasPrefix(lines[at], "   ")) {
			at++ // the field's lines under it
		}
	}
	ins := key + ": "
	if titleKeyMore(std) {
		ins = key + ":\n    "
	}
	var pos int
	put := ins + "\n"
	switch {
	case end == 0: // no title page yet: one, a blank line before the script
		pos, put = 0, ins+"\n\n"
	case at >= len(lines): // the script is its title page alone
		pos, put = len([]rune(text)), "\n"+ins
		if strings.HasSuffix(text, "\n") {
			pos, put = pos-1, "\n"+ins
		}
	default:
		pos = offset(at)
	}
	cursor := pos + len([]rune(ins))
	if strings.HasPrefix(put, "\n") {
		cursor++
	}
	e.Edit(func(b *buffer.Buffer) { b.Replace(pos, pos, put) })
	e.SetCursorOffset(cursor)
	w.focusMain()
	w.setStatus(key + " added to the title page: type its value")
}

// toScript puts the cursor on the script's first line after the title
// page.
func (w *MainWindow) toScript() {
	if w.scriptAway("To the Script") {
		return
	}
	e := w.textEditor
	text := e.Text()
	lines := strings.Split(text, "\n")
	end := max(titlePageLines(text), titlePageEnd(lines))
	i := end
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	n := 0
	for _, s := range lines[:min(i, len(lines))] {
		n += len([]rune(s)) + 1
	}
	e.SetCursorOffset(min(n, len([]rune(text))))
	w.focusMain()
}

// titlePageTab is the Title Page tab's layout.
func titlePageTab() ribbonTab {
	var keys []home
	for _, k := range titleKeys {
		keys = append(keys, at(titleKeyID(k)))
	}
	keys = append(keys, at(titleTemplateKeysHome))
	return ribbonTab{title: "Title Page", contextual: true, when: inTitlePage, groups: []ribbonGroup{
		{title: "Template", homes: []home{at(titleTemplateHome), at("view.preview")}},
		{title: "Add", homes: keys},
		{title: "Page", homes: []home{big("file.titlePage"), at("titlepage.toScript")}},
	}}
}

// titleKeyID is the id of the command adding std's key.
func titleKeyID(std string) string {
	return "titlepage.add." + strings.ReplaceAll(strings.ToLower(std), " ", "")
}

// contextCommands are the Fix-Up and Title Page tabs' commands.
func contextCommands() []*command {
	cs := []*command{
		{id: "fixup.applyRules", label: "Apply the Kept Rules Again…", ribbonLabel: "Kept Rules", icon: res(theme.MediaReplayIcon()),
			tip:  "The rules the script keeps in its Fix-Up block applied again: how many places each changes first, one undo step",
			when: fixingUp, enabled: keptFixRules, run: (*MainWindow).applyKeptRules},
		{id: "fixup.ocrAgain", label: "Read with OCR Again…", ribbonLabel: "OCR Again", icon: res(theme.ViewRefreshIcon()),
			tip:  "The PDF read again, its scans and its garbled pages with OCR (the language asked first)",
			when: fixingUpPDF, run: (*MainWindow).ocrAgain},
		{id: "fixup.checkText", label: "Check for Garbled Text", ribbonLabel: "Check Text", icon: res(theme.QuestionIcon()),
			tip: "How the PDF's pages read: scanned, garbled (a poor OCR's text) or clean", when: fixingUpPDF, run: (*MainWindow).checkGarbled},
		{id: "fixup.done", label: "Done Fixing", ribbonLabel: "Done", icon: res(theme.ConfirmIcon()),
			tip: "The Fix-Up tab goes (Saving does it too); Fix Lines Like This… stays in Edit", when: fixingUp, run: (*MainWindow).doneFixing},
		{id: "titlepage.toScript", label: "To the Script", icon: res(theme.MoveDownIcon()),
			tip: "The cursor to the script's first line, after the title page", when: inTitlePage, run: (*MainWindow).toScript},
	}
	for _, k := range titleKeys {
		std := k
		label := "Add " + strings.ToUpper(std[:1]) + std[1:]
		if std == "Draft date" {
			label = "Add Draft Date"
		}
		cs = append(cs, &command{id: titleKeyID(std), label: label, ribbonLabel: std, icon: res(theme.ContentAddIcon()),
			tip:            "The " + std + " key added to the title page, in the script's language",
			ribbonLabelFor: func(w *MainWindow) string { return w.titleKeyHere(std) },
			nameFor:        func(w *MainWindow) string { return "Add " + w.titleKeyHere(std) + " to the title page" },
			when:           inTitlePage, run: func(w *MainWindow) { w.addTitleKey(std) }})
	}
	return cs
}
