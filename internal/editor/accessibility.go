package editor

import (
	"fyne.io/fyne/v2"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// What screen readers read of the editor (Fyne's accessibility
// interfaces): a multi-line text with its caret and selection, by the
// lines as they are shown (wrapped), in characters.

// AccessibilityLabel is the editor's name for a screen reader.
func (e *ScriptEditor) AccessibilityLabel() string {
	if e.AccessibleName != "" {
		return e.AccessibleName
	}
	return "Script"
}

// AccessibilityRole: text edited over several lines.
func (e *ScriptEditor) AccessibilityRole() fyne.AccessibleRole { return "textArea" }

// AccessibilityValue is the text (for platforms that read a value).
func (e *ScriptEditor) AccessibilityValue() string { return e.buf.Text() }

// AccessibilityText is the text, read by character, word and line.
func (e *ScriptEditor) AccessibilityText() string { return e.buf.Text() }

// AccessibilityCaret is where the caret is, while the editor has the focus.
func (e *ScriptEditor) AccessibilityCaret() int {
	if !e.focused {
		return -1
	}
	return e.buf.Cursor()
}

// AccessibilitySelection is the selected text.
func (e *ScriptEditor) AccessibilitySelection() (int, int) { return e.buf.Selection() }

// AccessibilityTextLines are where the shown lines start: a wrapped
// line is two lines to the reader too, as it is on the screen.
func (e *ScriptEditor) AccessibilityTextLines() []int {
	starts := make([]int, 0, len(e.layout.Rows))
	for _, r := range e.layout.Rows {
		if len(starts) == 0 || r.Start > starts[len(starts)-1] {
			starts = append(starts, r.Start)
		}
	}
	if len(starts) == 0 || starts[0] != 0 {
		starts = append([]int{0}, starts...)
	}
	return starts
}

// AccessibilitySetCaret moves the caret (a screen reader's request).
func (e *ScriptEditor) AccessibilitySetCaret(offset int) bool {
	e.Navigate(func(b *buffer.Buffer) { b.SetCursor(offset, false) })
	return true
}

// AccessibilitySetSelection selects [start, end).
func (e *ScriptEditor) AccessibilitySetSelection(start, end int) bool {
	e.Navigate(func(b *buffer.Buffer) {
		b.SetCursor(start, false)
		b.SetCursor(end, true)
	})
	return true
}

// AccessibilityChildren: assistive technologies see through the page to
// the editor in it.
func (p *page) AccessibilityChildren() []fyne.CanvasObject { return []fyne.CanvasObject{p.content} }
