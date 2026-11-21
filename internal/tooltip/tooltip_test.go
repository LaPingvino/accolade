package tooltip

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

// manualDelay has a tip's delay end only when the test fires it.
func manualDelay(t *testing.T) (fire func()) {
	var f func()
	afterFunc = func(_ time.Duration, g func()) *time.Timer {
		f = g
		return time.NewTimer(time.Hour) // stopped by Out, never fires
	}
	t.Cleanup(func() { afterFunc = time.AfterFunc })
	return func() { f() }
}

func TestTipShowsBelowTheButtonAndHides(t *testing.T) {
	fire := manualDelay(t)
	a := test.NewApp()
	t.Cleanup(a.Quit)
	layer := NewLayer()
	taps := 0
	b := NewButton(theme.DocumentSaveIcon(), "Save (Ctrl+S)", func() { taps++ }, layer)
	w := test.NewWindow(container.NewStack(container.NewVBox(container.NewHBox(b)), layer))
	t.Cleanup(w.Close)
	w.Resize(fyne.NewSize(300, 200))

	b.MouseIn(&desktop.MouseEvent{})
	fire()
	if layer.Text() != "Save (Ctrl+S)" {
		t.Fatalf("tip %q after the delay", layer.Text())
	}
	if p := layer.TipPosition(); p.Y < b.Size().Height || p.X < 0 {
		t.Errorf("tip at %v, want below the button (height %v)", p, b.Size().Height)
	}

	// the layer takes no input: tapping where the button is still taps it
	test.TapCanvas(w.Canvas(), b.Position().Add(fyne.NewPos(b.Size().Width/2, b.Size().Height/2)))
	if taps != 1 {
		t.Fatalf("tap with a tip showing reached the button %d times, want 1", taps)
	}
	if layer.Text() != "" {
		t.Error("tapping should hide the tip")
	}

	b.MouseIn(&desktop.MouseEvent{})
	fire()
	b.MouseOut()
	if layer.Text() != "" {
		t.Error("tip still shown after the pointer left")
	}

	// leaving before the delay: no tip at all
	b.MouseIn(&desktop.MouseEvent{})
	b.MouseOut()
	fire() // the stopped delay, had it run anyway
	if layer.Text() != "" {
		t.Error("tip shown although the pointer left before the delay")
	}
}

// A keyboard user reaching an icon button sees its tip.
func TestTipOnFocus(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	layer := NewLayer()
	b := NewButton(nil, "Save (Ctrl+S)", func() {}, layer)
	w := test.NewWindow(container.NewStack(container.NewVBox(b), layer))
	t.Cleanup(w.Close)
	w.Canvas().Focus(b)
	if layer.Text() != "Save (Ctrl+S)" {
		t.Errorf("focused: tip %q", layer.Text())
	}
	w.Canvas().Unfocus()
	if layer.Text() == "Save (Ctrl+S)" && layer.Visible() {
		t.Error("unfocused: the tip stays")
	}
}
