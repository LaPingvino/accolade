package main

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/rules"
)

func newDialogTestWindow(t *testing.T) (*MainWindow, *Application) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := test.NewApp()
	t.Cleanup(a.Quit)
	app := &Application{fyneApp: a}
	w := NewMainWindow(app)
	app.windows = append(app.windows, w)
	w.fyneWindow.Resize(fyne.NewSize(1100, 650))
	return w, app
}

func capture(t *testing.T, c fyne.Canvas, name string) {
	if dir := os.Getenv("SHOT_DIR"); dir != "" {
		f, _ := os.Create(filepath.Join(dir, name))
		png.Encode(f, c.Capture())
		f.Close()
	}
}

// The preview shows the screenplay (it was empty: HTML went into a
// Markdown parser).
func TestPreviewShowsTheScreenplay(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.showPreview()
	separators := 0
	for _, s := range w.previewArea.Segments {
		if _, ok := s.(*widget.SeparatorSegment); ok {
			separators++
		}
	}
	got := previewText(w.previewArea.Segments)
	for _, want := range []string{"UNTITLED SCREENPLAY", "INT. LIVING ROOM - DAY", "JOHN", "FADE OUT."} {
		if !strings.Contains(got, want) {
			t.Errorf("preview lacks %q", want)
		}
	}
	if separators != 1 || !strings.Contains(got, strings.Repeat(" ", 20)+"UNTITLED SCREENPLAY") {
		t.Errorf("title page: %d page breaks, title not centred:\n%s", separators, got)
	}
	for _, line := range strings.Split(got, "\n") {
		if len([]rune(line)) > 60 {
			t.Errorf("line wider than the page: %q", line)
		}
	}
	capture(t, w.fyneWindow.Canvas(), "preview.png")
}

func TestPreviewDualDialogueSideBySide(t *testing.T) {
	segs := previewSegments("INT. ROOM - DAY\n\nANNA\nHello there.\n\nBRAM ^\nGoodbye now.\n", []string{"INT", "EXT"}, nil)
	text := previewText(segs)
	found := false
	for _, line := range strings.Split(text, "\n") {
		found = found || (strings.Contains(line, "ANNA") && strings.Contains(line, "BRAM"))
	}
	if !found {
		t.Errorf("dual dialogue not side by side:\n%s", text)
	}
}

// No dialog has an empty dismiss button any more, and Preferences is a
// window of its own.
func TestDialogsHaveNoEmptyButton(t *testing.T) {
	w, app := newDialogTestWindow(t)
	empty := func(o fyne.CanvasObject) int {
		n := 0
		var walk func(fyne.CanvasObject)
		walk = func(o fyne.CanvasObject) {
			if b, ok := o.(*widget.Button); ok && b.Text == "" && b.Icon == nil && b.Visible() {
				n++
			}
			if c, ok := o.(*fyne.Container); ok {
				for _, x := range c.Objects {
					walk(x)
				}
			}
			if wr, ok := o.(fyne.Widget); ok {
				r := test.WidgetRenderer(wr)
				for _, x := range r.Objects() {
					walk(x)
				}
			}
		}
		walk(o)
		return n
	}
	tp := NewTitlePageDialog(w)
	tp.Show()
	if n := empty(w.fyneWindow.Canvas().Overlays().Top()); n != 0 {
		t.Errorf("title page dialog: %d empty buttons", n)
	}
	capture(t, w.fyneWindow.Canvas(), "titlepage.png")
	tp.dialog.Hide()
	ed := NewExportDialog(w)
	ed.Show()
	if n := empty(w.fyneWindow.Canvas().Overlays().Top()); n != 0 {
		t.Errorf("export dialog: %d empty buttons", n)
	}
	capture(t, w.fyneWindow.Canvas(), "export.png")
	ed.dialog.Hide()
	pd := NewPreferencesDialog(app, w.fyneWindow)
	pd.Show()
	if pd.win == w.fyneWindow || w.fyneWindow.Canvas().Overlays().Top() != nil {
		t.Error("preferences is not a window of its own")
	}
	pd.win.Resize(fyne.NewSize(640, 420)) // smaller than its content: it scrolls, the buttons stay
	capture(t, pd.win.Canvas(), "preferences.png")
	pd.cancel()
}

// The preview follows the script format chosen in the preferences.
func TestPreviewFollowsTheScriptFormat(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText("INT. ROOM - DAY\n\nANNA\nHello there.\n")
	text := func() string {
		w.showPreview()
		return previewText(w.previewArea.Segments)
	}
	if got := text(); !strings.Contains(got, strings.Repeat(" ", 22)+"ANNA\n") {
		t.Errorf("screenplay:\n%s", got)
	}
	w.settings.SetString("script-format", "transcript")
	t.Cleanup(func() { w.settings.SetString("script-format", "default") })
	w.app.applySettingsToWindows()
	if got := text(); !strings.Contains(got, "\nANNA:\n") {
		t.Errorf("transcript:\n%s", got)
	}
}

// Every script format previews (radio's names left of the action margin
// once made the layout panic).
func TestPreviewInEveryScriptFormat(t *testing.T) {
	script := "INT. ROOM - DAY #1#\n\n# ACT ONE\n\nANNA\n(quietly)\nHello there, this is a long line of dialogue that wraps.\n\nBRAM ^\nGOTCHA!\n\n~A LYRIC\n\n> CUT TO:\n"
	for _, p := range rules.Presets() {
		if segs := previewSegments(script, []string{"INT", "EXT"}, p.Elements); len(segs) == 0 {
			t.Errorf("%s: empty preview", p.Key)
		}
	}
}

// previewText is the preview's text as shown: inline segments joined,
// one line per paragraph.
func previewText(segs []widget.RichTextSegment) string {
	var b strings.Builder
	for _, s := range segs {
		if s, ok := s.(*widget.TextSegment); ok {
			b.WriteString(s.Text)
			if !s.Style.Inline {
				b.WriteString("\n")
			}
		}
	}
	return b.String()
}

// The indent is not underlined with the text (radio's directions).
func TestPreviewIndentUnstyled(t *testing.T) {
	p, _ := rules.GetPreset("radio")
	for _, s := range previewSegments("INT. ROOM - DAY\n\nA clock ticks.\n", []string{"INT", "EXT"}, p.Elements) {
		if s, ok := s.(*widget.TextSegment); ok && strings.TrimSpace(s.Text) == "" && s.Style.TextStyle.Underline {
			t.Errorf("underlined indent %q", s.Text)
		}
	}
}
