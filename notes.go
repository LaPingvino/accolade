package main

import (
	"strings"
	"unicode/utf8"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/lexington/fountain"
)

// The notes block keeps the unsorted notes (the Notes pane's, for notes
// not tied to a place: notes_pane.go) with the script: at its end, in a
// Fountain boneyard block, which no Fountain app prints (in Markdown, an
// HTML comment: <!-- Notes ... -->); the other blocks at the end come
// before it, and the marks here are theirs too (endMarks):
//
//	/* Notes
//	...
//	*/

// notesMarks are how a kind of file keeps its notes block: Fountain's
// boneyard, Markdown's HTML comment (neither shows in what is printed).
type notesMarks struct {
	open, close, escaped string // escaped: close, when it is in the notes
}

func newNotesMarks(open, close, escaped string) notesMarks {
	return notesMarks{open, close, escaped}
}

var (
	fountainNotes = endMarks(fountain.EndNotes)
	markdownNotes = newNotesMarks("<!-- Notes", "-->", "-- >")
)

// at finds the block at the end of a text (Lexington's
// fountain.BlockMarks.At, which says what Accolade's end blocks are): its
// byte offset (with the blank lines before it) and what is inside it as
// written, or -1.
func (m notesMarks) at(text string) (int, string) {
	return fountain.BlockMarks{Open: m.open, Close: m.close, Escaped: m.escaped}.At(text)
}

// endMarks are the marks of Accolade's end block of that name
// (fountain.EndBlockMarks).
func endMarks(name string) notesMarks {
	b := fountain.EndBlockMarks(name)
	return notesMarks{b.Open, b.Close, b.Escaped}
}

// of are the notes at the end of a script.
func (m notesMarks) of(text string) string {
	_, notes := m.at(text)
	return notes
}

// startByte is where the notes block starts (the text's length if there
// is none), in bytes.
func (m notesMarks) startByte(text string) int {
	if i, _ := m.at(text); i >= 0 {
		return i
	}
	return len(text)
}

// start is where the notes block starts (the text's length if there is
// none), in runes.
func (m notesMarks) start(text string) int {
	return utf8.RuneCountInString(text[:m.startByte(text)])
}

// text is the block for notes ("" for none); a "*/" (Markdown: "-->")
// in them would end it, so it becomes "* /" ("-- >").
func (m notesMarks) text(notes string) string {
	notes = strings.TrimRight(notes, "\n")
	if strings.TrimSpace(notes) == "" {
		return ""
	}
	return "\n\n" + m.open + "\n" + strings.ReplaceAll(notes, m.close, m.escaped) + "\n" + m.close + "\n"
}

// A Fountain script's notes.
func notesAt(text string) (int, string) { return fountainNotes.at(text) }
func notesOf(text string) string        { return fountainNotes.of(text) }
func notesStart(text string) int        { return fountainNotes.start(text) }
func notesText(notes string) string     { return fountainNotes.text(notes) }

// notesMarks are the window's document's kind's.
func (w *MainWindow) notesMarks() notesMarks {
	if w.markdown {
		return markdownNotes
	}
	return fountainNotes
}

// The dock's panes, each made at its first need.

func (w *MainWindow) outlinePane() *outlinePanel {
	if w.outline == nil {
		w.outline = newOutlinePanel(w)
	}
	return w.outline
}

func (w *MainWindow) productionPane() *productionPane {
	if w.production == nil {
		w.production = newProductionPane(w)
	}
	return w.production
}

func (w *MainWindow) helpPane() *helpPane {
	if w.help == nil {
		w.help = newHelpPane(w)
	}
	return w.help
}

func (w *MainWindow) changesPane() *changesPanel {
	if w.changesPanel == nil {
		w.changesPanel = newChangesPanel(w)
	}
	return w.changesPanel
}

// edit is the edit that writes notes into text's notes block: from
// where (runes) to the end, what.
func (m notesMarks) edit(text, notes string) (from int, block string) {
	from = m.start(text)
	block = m.text(notes)
	if n := utf8.RuneCountInString(text); from == n && block != "" {
		// no block yet: after the script, without its trailing new lines
		from = utf8.RuneCountInString(strings.TrimRight(text, "\n"))
	} else if block == "" && from < n {
		block = "\n" // the script ends with a new line as before
	}
	return from, block
}

// writeNotes writes the script's whole notes block (the unsorted notes:
// notes_pane.go), "" taking it out.
func (w *MainWindow) writeNotes(notes string) { w.writeNotesAt(notes) }

// writeNotesAt is writeNotes, saying where the block went: from, what.
func (w *MainWindow) writeNotesAt(notes string) (from int, block string) {
	e := w.textEditor
	from, block = w.notesMarks().edit(e.Text(), notes)
	e.Edit(func(b *buffer.Buffer) {
		cursor := b.Cursor()
		replaceLeast(b, from, b.Len(), block) // only what changed (document.go)
		b.SetCursor(min(cursor, from), false)
	})
	return from, block
}
