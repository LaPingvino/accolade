package main

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// placeWords are a language's scene heading words for inside, outside and
// both (INT., EXT., INT./EXT.), as the statistics pair them
// (insideOutside): the halves of its first word with a slash. They end in
// a dot when the language's words have dots.
func placeWords(scenes []string) (in, out, both string, ok bool) {
	dot := ""
	for _, s := range scenes {
		if strings.Contains(s, ".") {
			dot = "."
		}
	}
	for _, s := range scenes {
		if i, o, found := strings.Cut(s, "/"); found {
			clean := func(s string) string { return strings.Trim(s, ". ") }
			return clean(i) + dot, clean(o) + dot, strings.TrimRight(s, ". ") + dot, true
		}
	}
	return "", "", "", false
}

// withPlace is a line made a scene heading starting with word: its own
// heading word (scenes) replaced, or word put before it.
func withPlace(line, word string, scenes []string) string {
	bare := strings.TrimLeft(line, directionMarks)
	marks := line[:len(line)-len(bare)]
	bare = strings.TrimSpace(bare)
	if tag := headingTag(bare, scenes); tag != "" && len(tag) <= len(bare) {
		bare = bare[len(tag):]
	} else {
		bare = strings.TrimPrefix(bare, ".") // forced: the word makes it a heading
	}
	bare = strings.TrimLeft(bare, ". ")
	return marks + word + " " + bare
}

// pickPlace offers inside, outside and both in the script's language, to
// start the cursor's line, a scene heading, with.
func (w *MainWindow) pickPlace(at fyne.CanvasObject) {
	scenes := sceneHeaders(w.language)
	in, out, both, ok := placeWords(scenes)
	if !ok {
		w.setStatus("This language has no words for inside and outside")
		return
	}
	var items []*fyne.MenuItem
	for _, word := range []string{in, out, both} {
		items = append(items, fyne.NewMenuItem(word, func() { w.setPlace(word) }))
	}
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(at).Add(fyne.NewPos(0, at.Size().Height))
	widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", items...), w.fyneWindow.Canvas(), pos)
}

// setPlace starts the cursor's line with word, the cursor at its end.
func (w *MainWindow) setPlace(word string) {
	scenes := sceneHeaders(w.language)
	w.textEditor.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			line, _ := b.Position(b.Cursor())
			start, end := b.LineStart(line), b.LineEnd(line)
			to := withPlace(b.Slice(start, end), word, scenes)
			b.Replace(start, end, to)
			b.SetCursor(start+len([]rune(to)), false)
		})
	})
	w.fyneWindow.Canvas().Focus(w.textEditor)
}
