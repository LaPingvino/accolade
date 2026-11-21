package main

import (
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The notepad (View > Notes) keeps notes with the script: at its end, in
// a Fountain boneyard block, which no Fountain app prints (in Markdown, an
// HTML comment: <!-- Notes ... -->):
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
	fountainNotes = newNotesMarks("/* Notes", "*/", "* /")
	markdownNotes = newNotesMarks("<!-- Notes", "-->", "-- >")
)

// at finds the notes block: the last "/* Notes" in the script, if
// it closes at the very end (an earlier boneyard of the same name is
// the script's, not the notes); its byte offset (with the blank lines
// before it) and the notes, or -1.
//
// It is a plain scan of the text's end (a regexp over a source kept in
// the script took a tenth of a second, several times per keystroke): the
// block's first close mark must have only white space after it, and the
// notes are what is before that mark, less one new line.
func (m notesMarks) at(text string) (int, string) {
	// only the end is read: the text must close with the mark (white
	// space after it), and the block open before that with no other
	// close between (the notes cannot hold one)
	c := len(text)
	for c > 0 && blankTail(text[c-1:c]) {
		c--
	}
	if !strings.HasSuffix(text[:c], m.close) {
		return -1, ""
	}
	c -= len(m.close)
	// forward scans (strings.Index is fast, LastIndex is not): the last
	// close before it, and the last opening after that
	from := 0
	for off := 0; ; {
		j := strings.Index(text[off:c], m.close)
		if j < 0 {
			break
		}
		from = off + j + len(m.close)
		off = from
	}
	open := m.open + "\n"
	i := -1
	for off := from; ; {
		j := strings.Index(text[off:c], open)
		if j < 0 {
			break
		}
		i = off + j
		off = i + 1
	}
	if i < 0 || (i > 0 && text[i-1] != '\n') {
		return -1, ""
	}
	notes := strings.TrimSuffix(text[i+len(open):c], "\n")
	for i > 0 && text[i-1] == '\n' {
		i--
	}
	return i, notes
}

// blankTail reports whether s is only white space as a regexp's \s
// means it (ASCII: space, tab, new line, form feed, carriage return).
func blankTail(s string) bool {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ', '\t', '\n', '\f', '\r':
		default:
			return false
		}
	}
	return true
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

// notesPanel is the script's notepad, a pane of the side dock (dock.go).
type notesPanel struct {
	w        *MainWindow
	entry    *widget.Entry
	updating bool // the entry is being set from the script
}

func newNotesPanel(w *MainWindow) *notesPanel {
	n := &notesPanel{w: w}
	n.entry = widget.NewMultiLineEntry()
	fyne.SetAccessibleLabel(n.entry, "Notes")
	n.entry.Wrapping = fyne.TextWrapWord
	n.entry.SetPlaceHolder("Notes on this script: kept at its end, never printed.")
	n.entry.OnChanged = func(notes string) {
		if !n.updating {
			n.save(notes)
		}
	}
	return n
}

// The dock's panes, each made at its first need.

func (w *MainWindow) notesPane() *notesPanel {
	if w.notes == nil {
		w.notes = newNotesPanel(w)
	}
	return w.notes
}

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

// update shows the script's notes, unless they are being typed.
func (n *notesPanel) update(text string) {
	notes := n.w.notesMarks().of(text)
	if notes == n.entry.Text {
		return
	}
	n.updating = true
	n.entry.SetText(notes)
	n.updating = false
}

// save writes the notes into the script's notes block.
func (n *notesPanel) save(notes string) { n.w.writeNotes(notes) }

// writeNotes writes notes into the script's notes block (the Notes pane,
// the notes in the main editor: document.go).
func (w *MainWindow) writeNotes(notes string) {
	e := w.textEditor
	text := e.Text()
	from := w.notesMarks().start(text)
	block := w.notesMarks().text(notes)
	if from == len([]rune(text)) && block != "" {
		// no block yet: after the script, without its trailing new lines
		from = len([]rune(strings.TrimRight(text, "\n")))
	}
	if block == "" && from < len([]rune(text)) {
		block = "\n" // the script ends with a new line as before
	}
	e.Edit(func(b *buffer.Buffer) {
		cursor := b.Cursor()
		replaceLeast(b, from, b.Len(), block) // only what changed (document.go)
		b.SetCursor(min(cursor, from), false)
	})
}
