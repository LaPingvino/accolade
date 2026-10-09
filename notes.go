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
// a Fountain boneyard block, which no Fountain app prints:
//
//	/* Notes
//	...
//	*/

var notesBlock = regexp.MustCompile(`(?s)^/\* Notes\n(.*?)\n?\*/\s*$`)

// notesAt finds the notes block: the last "/* Notes" in the script, if
// it closes at the very end (an earlier boneyard of the same name is
// the script's, not the notes); its byte offset (with the blank lines
// before it) and the notes, or -1.
func notesAt(text string) (int, string) {
	i := strings.LastIndex(text, "/* Notes\n")
	if i < 0 || (i > 0 && text[i-1] != '\n') {
		return -1, ""
	}
	m := notesBlock.FindStringSubmatch(text[i:])
	if m == nil || strings.Contains(m[1], "*/") {
		return -1, ""
	}
	for i > 0 && text[i-1] == '\n' {
		i--
	}
	return i, m[1]
}

// notesOf are the notes at the end of a script.
func notesOf(text string) string {
	_, notes := notesAt(text)
	return notes
}

// notesStart is where the notes block starts (the text's length if
// there is none), in runes.
func notesStart(text string) int {
	if i, _ := notesAt(text); i >= 0 {
		return len([]rune(text[:i]))
	}
	return len([]rune(text))
}

// notesText is the block for notes ("" for none); a "*/" in them would
// end it, so it becomes "* /".
func notesText(notes string) string {
	notes = strings.TrimRight(notes, "\n")
	if strings.TrimSpace(notes) == "" {
		return ""
	}
	return "\n\n/* Notes\n" + strings.ReplaceAll(notes, "*/", "* /") + "\n*/\n"
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
	notes := notesOf(text)
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
	from := notesStart(text)
	block := notesText(notes)
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
