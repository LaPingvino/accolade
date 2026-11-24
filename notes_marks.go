package main

import (
	"fmt"
	"image/color"
	"maps"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The calls to action of what is open (open_items.go; Joop: "We need
// statistics that come with a call to action"): the status bar's
// "4 open" (the to-dos and the beats without a scene), which opens the
// Notes pane; a section's "3 beats without a scene" in the outline and
// on the cards, which steps through them; and a quiet mark in the
// editor's margin where a note, a cut or a to-do is. All read from the
// text once typing pauses.

// openPause is how long after typing the count and the marks follow.
const openPause = 400 * time.Millisecond

// linkLabel is a small label that does something when tapped: the
// status bar's count, a header's beats.
type linkLabel struct {
	widget.Label
	onTap func()
}

func newLinkLabel(onTap func()) *linkLabel {
	l := &linkLabel{onTap: onTap}
	l.SizeName = theme.SizeNameCaptionText
	l.ExtendBaseWidget(l)
	return l
}

func (l *linkLabel) Tapped(*fyne.PointEvent) {
	if l.onTap != nil {
		l.onTap()
	}
}

func (l *linkLabel) Cursor() desktop.Cursor { return desktop.PointerCursor }

// scheduleOpen counts what is open again once typing pauses.
func (w *MainWindow) scheduleOpen() {
	if w.textEditor == nil || w.openLabel == nil {
		return
	}
	w.debounce(&w.openTimer, openPause, w.updateOpen)
}

// updateOpen counts what is open and marks the margin.
func (w *MainWindow) updateOpen() {
	if w.textEditor == nil || w.openLabel == nil {
		return
	}
	items := openItemsOf(w.textEditor.Text(), w.sceneStarts(), w.markdown)
	todos, beats := 0, 0
	marks := map[int]color.Color{}
	for _, it := range items {
		switch it.Kind {
		case openTodo:
			todos++
		case openBeat:
			beats++
		}
		if it.Kind != openBeat && it.Kind != openUnsorted && !it.Kept { // an omitted scene's block folds: its fold mark there
			marks[it.Line] = marginColor(it.Kind)
		}
	}
	if n := todos + beats; n > 0 {
		w.openLabel.SetText(fmt.Sprintf("%d open", n))
		var parts []string
		if todos > 0 {
			parts = append(parts, plural(todos, "to-do"))
		}
		if beats > 0 {
			parts = append(parts, plural(beats, "beat")+" without a scene")
		}
		fyne.SetAccessibleLabel(w.openLabel, strings.Join(parts, ", ")+": the Notes pane")
		w.openLabel.Show()
	} else {
		w.openLabel.Hide()
	}
	if len(marks) == 0 {
		marks = nil
	}
	if !maps.EqualFunc(marks, w.noteMarks, func(a, b color.Color) bool { return a == b }) {
		w.noteMarks = marks
		w.textEditor.RedrawMarks()
	}
}

// marginColor is a kind's mark in the margin: its colour faded, not to
// be taken for a revision's.
func marginColor(k openKind) color.Color {
	r, g, b, _ := kindColor(k).RGBA()
	return color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 0x60}
}

// marksShown are the editor's margin marks: the revisions' colours
// (revisions.go), and where there is none, the notes' quiet ones.
func (w *MainWindow) marksShown() map[int]color.Color {
	ink := w.revisionMarksShown()
	if len(w.noteMarks) == 0 {
		return ink
	}
	if len(ink) == 0 {
		return w.noteMarks
	}
	out := maps.Clone(w.noteMarks)
	maps.Copy(out, ink)
	return out
}

// sectionBeats are the beats no scene grew from under each section, by
// the section's line (-1: before the first): the outline's and the
// cards' to-do marks.
func sectionBeats(text string, starts []string, items []outlineItem) map[int][]int {
	lines := strings.Split(text, "\n")
	return beatsBySection(beatsOf(lines, items, titlePageEnd(lines), endLine(text, fountainBodyEnd(text))))
}

// beatsMark is a header's mark for n beats ("" none).
func beatsMark(n int) string {
	if n == 0 {
		return ""
	}
	return "⚑ " + plural(n, "beat") + " without a scene"
}

// stepToBeat goes to the next beat without a scene of those (lines)
// after the cursor, round to the first, and says which it is.
func (w *MainWindow) stepToBeat(lines []int) {
	if len(lines) == 0 {
		return
	}
	at := strings.Count(string([]rune(w.textEditor.Text())[:w.textEditor.CursorOffset()]), "\n")
	k := 0
	for i, l := range lines {
		if l > at {
			k = i
			break
		}
	}
	w.goToScript(lines[k])
	w.setStatus(fmt.Sprintf("Beat %d of %d without a scene: New Scene grows it into one", k+1, len(lines)))
}
