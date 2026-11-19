package main

import (
	"image/color"
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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
	block                *regexp.Regexp
}

func newNotesMarks(open, close, escaped string) notesMarks {
	return notesMarks{open, close, escaped,
		regexp.MustCompile(`(?s)^` + regexp.QuoteMeta(open) + `\n(.*?)\n?` + regexp.QuoteMeta(close) + `\s*$`)}
}

var (
	fountainNotes = newNotesMarks("/* Notes", "*/", "* /")
	markdownNotes = newNotesMarks("<!-- Notes", "-->", "-- >")
)

// at finds the notes block: the last "/* Notes" in the script, if
// it closes at the very end (an earlier boneyard of the same name is
// the script's, not the notes); its byte offset (with the blank lines
// before it) and the notes, or -1.
func (m notesMarks) at(text string) (int, string) {
	i := strings.LastIndex(text, m.open+"\n")
	if i < 0 || (i > 0 && text[i-1] != '\n') {
		return -1, ""
	}
	found := m.block.FindStringSubmatch(text[i:])
	if found == nil || strings.Contains(found[1], m.close) {
		return -1, ""
	}
	for i > 0 && text[i-1] == '\n' {
		i--
	}
	return i, found[1]
}

// of are the notes at the end of a script.
func (m notesMarks) of(text string) string {
	_, notes := m.at(text)
	return notes
}

// start is where the notes block starts (the text's length if there is
// none), in runes.
func (m notesMarks) start(text string) int {
	if i, _ := m.at(text); i >= 0 {
		return len([]rune(text[:i]))
	}
	return len([]rune(text))
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

// notesPanel is the notepad beside the editor.
type notesPanel struct {
	w        *MainWindow
	entry    *widget.Entry
	box      fyne.CanvasObject
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
	width := canvas.NewRectangle(color.Transparent) // a nil colour crashes the painter
	width.SetMinSize(fyne.NewSize(240, 0))
	heading := widget.NewLabelWithStyle("Notes", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	n.box = container.NewStack(width, container.NewBorder(heading, nil, nil, nil, n.entry))
	return n
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
func (n *notesPanel) save(notes string) {
	e := n.w.textEditor
	text := e.Text()
	from := n.w.notesMarks().start(text)
	block := n.w.notesMarks().text(notes)
	if from == len([]rune(text)) && block != "" {
		// no block yet: after the script, without its trailing new lines
		from = len([]rune(strings.TrimRight(text, "\n")))
	}
	if block == "" && from < len([]rune(text)) {
		block = "\n" // the script ends with a new line as before
	}
	e.Edit(func(b *buffer.Buffer) {
		cursor := b.Cursor()
		b.Replace(from, b.Len(), block)
		b.SetCursor(min(cursor, from), false)
	})
}
