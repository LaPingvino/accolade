package main

import (
	"errors"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The ex command line (ex.go) at the window's foot: ":" in vim and helix
// keys opens it, Enter runs the command, Escape closes it, Up and Down go
// through the commands run before.

// exLine is the command line's entry.
type exLine struct {
	widget.Entry
	w    *MainWindow
	back int // how far back in the history (0: what is typed)
}

func newExLine(w *MainWindow) *exLine {
	l := &exLine{w: w}
	l.ExtendBaseWidget(l)
	l.OnSubmitted = func(cmd string) {
		w.hideEx()
		w.runEx(cmd)
	}
	fyne.SetAccessibleLabel(l, "Command")
	return l
}

func (l *exLine) TypedKey(k *fyne.KeyEvent) {
	h := l.w.exState.history
	switch k.Name {
	case fyne.KeyEscape:
		l.w.hideEx()
	case fyne.KeyUp, fyne.KeyDown:
		if k.Name == fyne.KeyUp {
			l.back = min(l.back+1, len(h))
		} else {
			l.back = max(l.back-1, 0)
		}
		if l.back > 0 {
			l.SetText(h[len(h)-l.back])
		} else {
			l.SetText("")
		}
		l.CursorColumn = len([]rune(l.Text))
		l.Refresh()
	default:
		l.Entry.TypedKey(k)
	}
}

// showEx opens the command line, with '<,'> for the lines given.
func (w *MainWindow) showEx(visual [2]int) {
	if w.exBar == nil {
		w.exEntry = newExLine(w)
		prompt := widget.NewLabel(":")
		prompt.TextStyle = fyne.TextStyle{Monospace: true}
		w.exBar = container.NewBorder(nil, nil, prompt, nil, w.exEntry)
		w.statusBar.Add(w.exBar)
	}
	w.exVisual = visual
	w.exEntry.back = 0
	w.exEntry.SetText("")
	if visual[0] >= 0 {
		w.exEntry.SetText("'<,'>")
		w.exEntry.CursorColumn = 5
	}
	setVisible(w.statusBar, true)
	w.exBar.Show()
	w.fyneWindow.Canvas().Focus(w.exEntry)
}

func (w *MainWindow) hideEx() {
	if w.exBar != nil {
		w.exBar.Hide()
	}
	setVisible(w.statusBar, !w.isFullscreen && w.settings.GetBoolean("statusbar-visible"))
	w.fyneWindow.Canvas().Focus(w.textEditor)
}

// runEx runs an ex command on the script.
func (w *MainWindow) runEx(cmd string) {
	if strings.TrimSpace(cmd) == "" {
		return
	}
	st := &w.exState
	if h := st.history; len(h) == 0 || h[len(h)-1] != cmd {
		st.history = append(h, cmd)
	}
	text := w.textEditor.Text()
	lines := strings.Split(text, "\n")
	cur, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
	res, err := st.run(cmd, lines, cur, w.exVisual)
	if err != nil {
		if errors.Is(err, errNotCommand) {
			w.setStatus("E492: " + err.Error())
		} else {
			w.setStatus(err.Error())
		}
		return
	}
	if res.undo {
		w.undo()
		return
	}
	if nw := strings.Join(res.lines, "\n"); nw != text {
		w.replaceText(text, nw) // one undo step, only what changed
	}
	w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(min(res.line, b.LineCount()-1)), false) })
	if res.yank != "" && w.vim != nil {
		w.vim.reg, w.vim.regLine = res.yank, true
	}
	if res.msg != "" {
		w.setStatus(res.msg)
	}
	if res.write {
		if res.file != "" {
			p := res.file
			if !filepath.IsAbs(p) && w.currentFile != "" {
				p = filepath.Join(filepath.Dir(w.currentFile), p)
			}
			w.currentFile = p
			w.setLanguage(languageOf(p))
		}
		if err := w.SaveFile(); err != nil {
			w.setStatus(err.Error())
			return
		}
		w.setStatus("Written: " + filepath.Base(w.currentFile))
	}
	if res.quit {
		if res.keep {
			w.hasChanges = false // :q! closes without asking
		}
		w.fyneWindow.Close()
	}
}

// replaceText changes the editor's text from old to nw as one edit: the
// part between what they start and end with in common.
func (w *MainWindow) replaceText(old, nw string) {
	o, n := []rune(old), []rune(nw)
	start := 0
	for start < len(o) && start < len(n) && o[start] == n[start] {
		start++
	}
	end := 0
	for end < len(o)-start && end < len(n)-start && o[len(o)-1-end] == n[len(n)-1-end] {
		end++
	}
	w.textEditor.Edit(func(b *buffer.Buffer) {
		b.Group(func() { b.Replace(start, len(o)-end, string(n[start:len(n)-end])) })
	})
}
