package main

import (
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// View > Document Type: a document is a screenplay or Markdown whatever
// its name. The extension says which by default (.md Markdown, the rest a
// screenplay); a choice made here wins, and is remembered for the file.
//
// Where the choice is kept: in the settings (settings.json), as
// "document-types", a list of "markdown:/path" or "screenplay:/path" like
// File > Open Recent's list. It is the lightest place that lasts and works
// in the browser too (its settings folder is kept: web/index.html's
// memfsPersist). Not in the file: other Fountain and Markdown apps would
// see a mark they do not know. Not beside it: a hidden file next to every
// script is clutter, and the browser has no folder of the user's to put it
// in. Only a choice the extension would not make is kept, the newest
// first and at most maxDocumentTypes, so the list stays short.

const maxDocumentTypes = 200

// documentTypeEntry is how a file's chosen type is kept in the list.
func documentTypeEntry(path string, markdown bool) string {
	if markdown {
		return "markdown:" + path
	}
	return "screenplay:" + path
}

// absPath is path made absolute, as the recent files are kept.
func absPath(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

// chosenType is the type chosen for a file, if one was.
func (w *MainWindow) chosenType(path string) (markdown, chosen bool) {
	if w.settings == nil || path == "" {
		return false, false
	}
	path = absPath(path)
	for _, e := range w.settings.GetStringSlice("document-types") {
		switch e {
		case documentTypeEntry(path, true):
			return true, true
		case documentTypeEntry(path, false):
			return false, true
		}
	}
	return false, false
}

// typeOf tells whether a file opens as Markdown: as chosen for it, else
// as its extension says.
func (w *MainWindow) typeOf(path string) bool {
	if md, ok := w.chosenType(path); ok {
		return md
	}
	return isMarkdownFile(path)
}

// rememberType keeps a file's type: kept if its extension would say
// otherwise, forgotten if not.
func (w *MainWindow) rememberType(path string, markdown bool) {
	if w.settings == nil || path == "" {
		return
	}
	path = absPath(path)
	var list []string
	if markdown != isMarkdownFile(path) {
		list = append(list, documentTypeEntry(path, markdown))
	}
	old := w.settings.GetStringSlice("document-types")
	for _, e := range old {
		if e != documentTypeEntry(path, true) && e != documentTypeEntry(path, false) && len(list) < maxDocumentTypes {
			list = append(list, e)
		}
	}
	if len(list) == 0 && len(old) == 0 {
		return // nothing kept, nothing to keep
	}
	w.settings.SetStringSlice("document-types", list)
	_ = w.settings.Save()
}

// decisiveExtension: naming a file so makes it that type (renaming
// story.txt to story.md makes it Markdown). .txt and other names say
// nothing on a rename: the document stays what it is.
func decisiveExtension(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".markdown", ".fountain", ".spmd":
		return true
	}
	return false
}

// typeAfterRename follows a document to its new name (a rename, Save As):
// a new extension that says a type makes it that type; otherwise it stays
// what it is, and the choice moves with it. oldName and newName are the
// names (an unsaved document's too), oldPath and newPath the files ("" for
// none).
func (w *MainWindow) typeAfterRename(oldPath, oldName, newPath, newName string) {
	w.sourceAfterRename(oldPath, newPath) // and its source with it (source.go)
	if oldPath != "" && absPath(oldPath) != absPath(newPath) {
		w.rememberType(oldPath, isMarkdownFile(oldPath)) // forgotten
	}
	if decisiveExtension(newName) && !strings.EqualFold(filepath.Ext(oldName), filepath.Ext(newName)) {
		w.modeFromName(newName)
	}
	w.rememberType(newPath, w.markdown)
}

// setDocumentType is View > Document Type: the document becomes a
// screenplay or Markdown, keeping its name and its text (no Fountain
// clean-up runs); a notes block in the other type's form is carried over
// to this one's. The choice is remembered for the file.
func (w *MainWindow) setDocumentType(markdown bool) {
	if w.markdown == markdown {
		return
	}
	from, to := fountainNotes, markdownNotes
	if !markdown {
		from, to = markdownNotes, fountainNotes
	}
	text := w.textEditor.Text()
	if start, end, block, ok := carriedNotes(text, from, to); ok {
		w.textEditor.Edit(func(b *buffer.Buffer) {
			cursor := b.Cursor()
			b.Replace(start, end, block)
			b.SetCursor(min(cursor, start), false)
		})
	}
	w.setMarkdown(markdown)
	w.rememberType(w.currentFile, markdown)
	if inBrowser { // an unsaved document comes back as this type after a reload
		keepDraft(kept{Text: w.textEditor.Text(), Language: w.language, File: w.currentFile, Name: w.suggestedName, Markdown: w.markdown})
	}
}

// carriedNotes is the notes block of text in from's form, rewritten in
// to's: where it is (in runes) and what replaces it. Only the marks
// change; the notes and the blank lines around them stay as they are.
// ok is false if there is no such block, or one in to's form already.
func carriedNotes(text string, from, to notesMarks) (start, end int, block string, ok bool) {
	if i, _ := from.at(text); i < 0 {
		return 0, 0, "", false
	}
	if i, _ := to.at(text); i >= 0 {
		return 0, 0, "", false
	}
	j := strings.LastIndex(text, from.open+"\n")
	region := text[j:]
	c := strings.LastIndex(region, from.close)
	notes := region[len(from.open)+1 : c]
	block = to.open + "\n" + strings.ReplaceAll(notes, to.close, to.escaped) + to.close + region[c+len(from.close):]
	return len([]rune(text[:j])), len([]rune(text)), block, true
}

// documentTypeMenu is View > Document Type: Screenplay or Markdown, the
// one the document is checked.
func (w *MainWindow) documentTypeMenu() *fyne.MenuItem {
	screenplay := fyne.NewMenuItem("Screenplay", func() { w.setDocumentType(false) })
	markdown := fyne.NewMenuItem("Markdown", func() { w.setDocumentType(true) })
	screenplay.Checked, markdown.Checked = !w.markdown, w.markdown
	item := fyne.NewMenuItem("Document Type", nil)
	item.ChildMenu = fyne.NewMenu("", screenplay, markdown)
	return item
}
