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

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	for end := time.Now().Add(2 * time.Second); time.Now().Before(end); time.Sleep(5 * time.Millisecond) {
		if cond() {
			return
		}
	}
	t.Fatal("timed out")
}

func TestTipShowsBelowTheButtonAndHides(t *testing.T) {
	Delay = 10 * time.Millisecond
	a := test.NewApp()
	t.Cleanup(a.Quit)
	layer := NewLayer()
	taps := 0
	b := NewButton(theme.DocumentSaveIcon(), "Save (Ctrl+S)", func() { taps++ }, layer)
	w := test.NewWindow(container.NewStack(container.NewVBox(container.NewHBox(b)), layer))
	t.Cleanup(w.Close)
	w.Resize(fyne.NewSize(300, 200))

	b.MouseIn(&desktop.MouseEvent{})
	waitFor(t, func() bool { return layer.Text() == "Save (Ctrl+S)" })
	if layer.bg.Position().Y < b.Size().Height || layer.bg.Position().X < 0 {
		t.Errorf("tip at %v, want below the button (height %v)", layer.bg.Position(), b.Size().Height)
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
	waitFor(t, func() bool { return layer.Text() != "" })
	b.MouseOut()
	if layer.Text() != "" {
		t.Error("tip still shown after the pointer left")
	}

	// leaving before the delay: no tip at all
	b.MouseIn(&desktop.MouseEvent{})
	b.MouseOut()
	time.Sleep(3 * Delay)
	if layer.Text() != "" {
		t.Error("tip shown although the pointer left before the delay")
	}
}
