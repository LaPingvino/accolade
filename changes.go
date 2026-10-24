package main

import (
	"fmt"
	"image/color"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// A shared script's history (View > Changes): what the others changed,
// who and when, newest first; tapped, a change shows its line. Each peer
// tells its name (Preferences > General: Your name) as a setting of the
// shared script, under its replica's name.

// change is one peer's edit, or several of theirs close together.
type change struct {
	Who            string
	When           time.Time
	Line           int    // where it was made
	Added, Removed string // the text put in and taken out
}

// maxChanges is how many changes the history keeps.
const maxChanges = 200

// nameKey is the shared setting with replica's name.
func nameKey(replica string) string { return "name/" + replica }

// yourName is the name the others see: the preferences', else the
// login's, else Someone.
func (w *MainWindow) yourName() string {
	if n := strings.TrimSpace(w.settings.GetString("your-name")); n != "" {
		return n
	}
	if n := os.Getenv("USER"); n != "" {
		return n
	}
	return "Someone"
}

// recordChange puts replica by's edit at line in the history: removed
// replaced by added. Edits of one person close in time and place make
// one change; a catch-up of several people's edits (by "") is left out.
func (w *MainWindow) recordChange(by, removed, added string, line int) {
	if by == "" || removed == added {
		return
	}
	who := "Someone"
	if w.shared != nil {
		if n := w.shared.session.Meta(nameKey(by)); n != "" {
			who = n
		}
	}
	now := time.Now()
	if n := len(w.changes); n > 0 {
		last := &w.changes[n-1]
		if last.Who == who && now.Sub(last.When) < time.Minute && abs(last.Line-line) <= 2 {
			last.When, last.Line = now, line
			last.Added += added
			last.Removed += removed
			w.refreshChanges()
			return
		}
	}
	w.changes = append(w.changes, change{Who: who, When: now, Line: line, Added: added, Removed: removed})
	if len(w.changes) > maxChanges {
		w.changes = w.changes[len(w.changes)-maxChanges:]
	}
	w.setStatus(fmt.Sprintf("%s changed line %d", who, line+1))
	w.refreshChanges()
}

func abs(n int) int { return max(n, -n) }

// ago is how long ago t was, roughly.
func ago(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d h ago", int(d.Hours()))
	}
	return t.Format("Jan 2, 15:04")
}

// shortened is s on one line, at most n characters.
func shortened(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// changesPanel lists the history beside the editor.
type changesPanel struct {
	w    *MainWindow
	list *widget.List
	none *widget.Label
	box  fyne.CanvasObject
}

func newChangesPanel(w *MainWindow) *changesPanel {
	p := &changesPanel{w: w}
	p.none = widget.NewLabel("No changes from others yet. Share the script (File > Share) to write it together.")
	p.none.Wrapping = fyne.TextWrapWord
	p.none.Importance = widget.LowImportance
	p.list = widget.NewList(
		func() int { return len(w.changes) },
		func() fyne.CanvasObject {
			head := widget.NewLabel("")
			head.TextStyle = fyne.TextStyle{Bold: true}
			head.SizeName = theme.SizeNameCaptionText
			head.Truncation = fyne.TextTruncateEllipsis
			what := widget.NewLabel("")
			what.Truncation = fyne.TextTruncateEllipsis
			return container.NewVBox(head, what)
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			c := w.changes[len(w.changes)-1-id] // newest first
			box := o.(*fyne.Container)
			box.Objects[0].(*widget.Label).SetText(fmt.Sprintf("%s · %s · line %d", c.Who, ago(c.When, time.Now()), c.Line+1))
			var what []string
			if a := shortened(c.Added, 60); a != "" {
				what = append(what, "+ "+a)
			}
			if r := shortened(c.Removed, 60); r != "" {
				what = append(what, "− "+r)
			}
			box.Objects[1].(*widget.Label).SetText(strings.Join(what, "  "))
		},
	)
	p.list.OnSelected = func(id widget.ListItemID) {
		if id < len(w.changes) {
			line := w.changes[len(w.changes)-1-id].Line
			e := w.textEditor
			e.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(min(line, b.LineCount()-1)), false) })
			w.fyneWindow.Canvas().Focus(e)
		}
		p.list.UnselectAll()
	}
	title := widget.NewLabelWithStyle("Changes", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	width := canvas.NewRectangle(color.Transparent) // as wide as the outline and notes
	width.SetMinSize(fyne.NewSize(240, 0))
	p.box = container.NewStack(width, container.NewBorder(title, nil, nil, nil, container.NewStack(p.list, p.none)))
	p.refresh()
	return p
}

func (p *changesPanel) refresh() {
	if len(p.w.changes) == 0 {
		p.none.Show()
		p.list.Hide()
	} else {
		p.none.Hide()
		p.list.Show()
	}
	p.list.Refresh()
}

func (w *MainWindow) refreshChanges() {
	if w.changesVisible && w.changesPanel != nil {
		w.changesPanel.refresh()
	}
}

// toggleChanges shows or hides the history beside the editor.
func (w *MainWindow) toggleChanges() {
	w.changesVisible = !w.changesVisible
	if w.changesVisible && w.changesPanel == nil {
		w.changesPanel = newChangesPanel(w)
	}
	w.refreshChanges()
	w.relayout()
}
