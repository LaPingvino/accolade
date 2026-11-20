package main

import (
	"image/color"
	"strings"
	"unicode/utf8"

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

// notesPanel is the notepad beside the editor; for a screenplay, with a
// second tab, its source (source_pane.go), and a third, its production
// stages (production_pane.go); last, for either, the help (help_pane.go).
type notesPanel struct {
	w        *MainWindow
	entry    *widget.Entry
	updating bool // the entry is being set from the script

	tabs      *container.AppTabs
	notesTab  *container.TabItem
	sourceTab *container.TabItem
	tabbed    fyne.CanvasObject // notes, source, production and help
	source    *sourcePane

	productionTab *container.TabItem
	production    *productionPane

	helpTab *container.TabItem
	help    *helpPane         // help_pane.go
	mdBox   fyne.CanvasObject // a Markdown document's: notes and help
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
	n.source = newSourcePane(w)
	n.notesTab = container.NewTabItem("Notes", n.entry)
	n.sourceTab = container.NewTabItem("Source", n.source.box)
	n.production = newProductionPane(w)
	n.productionTab = container.NewTabItem("Production", n.production.box)
	n.help = newHelpPane(w)
	n.helpTab = container.NewTabItem("Help", n.help.box)
	n.tabs = container.NewAppTabs(n.notesTab, n.sourceTab, n.productionTab, n.helpTab)
	n.tabs.OnSelected = func(t *container.TabItem) {
		switch t {
		case n.sourceTab:
			n.source.follow()
		case n.productionTab:
			n.production.refresh()
		case n.helpTab:
			if n.help.page.Slug == "" {
				n.help.open("index", "")
			}
		}
	}
	wide := canvas.NewRectangle(color.Transparent)
	wide.SetMinSize(fyne.NewSize(sourceWidth, 0))
	n.tabbed = container.NewStack(wide, n.tabs)
	narrow := canvas.NewRectangle(color.Transparent) // a nil colour crashes the painter
	narrow.SetMinSize(fyne.NewSize(280, 0))
	n.mdBox = container.NewStack(narrow, n.tabs)
	return n
}

// box is the pane as the document has it: a Markdown document's notes,
// a screenplay's notes and source.
func (n *notesPanel) box() fyne.CanvasObject {
	if n.w.markdown {
		n.setTabs(n.notesTab, n.helpTab)
		return n.mdBox
	}
	n.setTabs(n.notesTab, n.sourceTab, n.productionTab, n.helpTab)
	return n.tabbed
}

// setTabs makes the tabs items (a Markdown document has no source or
// production), keeping the one selected if it stays.
func (n *notesPanel) setTabs(items ...*container.TabItem) {
	same := len(items) == len(n.tabs.Items)
	for i := 0; same && i < len(items); i++ {
		same = items[i] == n.tabs.Items[i]
	}
	if same {
		return
	}
	sel := n.tabs.Selected()
	n.tabs.SetItems(items)
	for _, it := range items {
		if it == sel {
			n.tabs.Select(it)
		}
	}
	n.help.fillPicker() // the cheat sheets in the mode's order
}

// focusTarget is what takes the keyboard in the pane: the notes, or the
// source if its tab is open.
func (n *notesPanel) focusTarget() fyne.Focusable {
	if n.tabs.Selected() == n.helpTab {
		return n.help.focusTarget()
	}
	if !n.w.markdown && n.tabs.Selected() == n.sourceTab {
		return n.source.focusTarget()
	}
	if !n.w.markdown && n.tabs.Selected() == n.productionTab {
		return n.production.focusTarget()
	}
	return n.entry
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
