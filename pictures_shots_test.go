package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
)

// harbourPicture is a made-up picture to look at: sky, sea, a sun.
func harbourPicture(t testing.TB, w, h int, hue float64) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			fy := float64(y) / float64(h)
			c := color.NRGBA{uint8(250 - 80*fy), uint8(180 + 40*fy*hue), uint8(120 + 120*fy), 255}
			if fy > 0.6 {
				c = color.NRGBA{20, uint8(70 + 60*(fy-0.6)), uint8(120 + 100*(fy-0.6)), 255}
			}
			if dx, dy := float64(x)-float64(w)*0.7, float64(y)-float64(h)*0.45; math.Hypot(dx, dy) < float64(h)*0.12 {
				c = color.NRGBA{255, 230, 120, 255}
			}
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	return b.Bytes()
}

// TestPictureScreenshots: a Markdown document with a picture (its Live
// Preview's text and pages), a script with storyboard frame chips, the
// Characters tab with a cast photo; desktop and phone (SHOT_DIR).
func TestPictureScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(dir, 0o755)
	docs := t.TempDir()
	os.MkdirAll(filepath.Join(docs, "assets"), 0o755)
	os.WriteFile(filepath.Join(docs, "assets", "harbour.png"), harbourPicture(t, 900, 450, 0), 0o644)
	os.WriteFile(filepath.Join(docs, "assets", "boats.png"), harbourPicture(t, 640, 360, 1), 0o644)
	os.WriteFile(filepath.Join(docs, "assets", "quay.png"), harbourPicture(t, 640, 270, 0.5), 0o644)
	os.WriteFile(filepath.Join(docs, "assets", "anna.png"), harbourPicture(t, 300, 300, 0.8), 0o644)
	md := filepath.Join(docs, "harbour.md")
	os.WriteFile(md, []byte("# The Harbour\n\nThe boats came in at dawn, one by one, their lamps still lit.\n\n"+
		"![The harbour at dawn](assets/harbour.png \"The harbour, an hour before the market\")\n\n"+
		"Anna waited on the quay. ![a sun](assets/quay.png) She had waited all night.\n\n![A picture not there](assets/lost.png)\n\n"+
		"![On the web](https://example.com/far.png)\n"), 0o644)
	script := filepath.Join(docs, "harbour.fountain")
	os.WriteFile(script, []byte("Title: The Harbour\n\nEXT. HARBOUR - DAWN\n\nThe boats come in, one by one, lamps still lit.\n[[board: assets/boats.png, duration: 2.50]]\n\n"+
		"Anna waits on the quay.\n[[board: assets/quay.png]]\n\nANNA\nThere he is.\n[[board: assets/gone.png]]\n\n"+
		"/* Characters\nANNA: waits for her brother\n  photo: assets/anna.png\n*/\n"), 0o644)
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"desktop", fyne.NewSize(1200, 800)}, {"phone", fyne.NewSize(380, 800)}} {
		w, _ := newDialogTestWindow(t)
		w.fyneWindow.Resize(size.size)
		if err := w.LoadFile(md); err != nil {
			t.Fatal(err)
		}
		w.preview()
		w.fyneWindow.Content().Refresh()
		capture(t, w.fyneWindow.Canvas(), "pictures-md-text-"+size.name+".png")
		shown := make(chan *pageView, 4)
		f := func(v *pageView) { shown <- v }
		old := pagesShownHook.Swap(&f)
		w.previewPane.mode.SetSelected(previewPages)
		select {
		case v := <-shown:
			painted(t, v, 0)
		case <-time.After(30 * time.Second):
			t.Fatal("no pages")
		}
		pagesShownHook.Store(old)
		capture(t, w.fyneWindow.Canvas(), "pictures-md-pages-"+size.name+".png")
		w.previewPane.mode.SetSelected(previewText)
		w.settings.SetBoolean(previewPagesPref, false)

		w2, _ := newDialogTestWindow(t)
		w2.fyneWindow.Resize(size.size)
		if err := w2.LoadFile(script); err != nil {
			t.Fatal(err)
		}
		w2.fyneWindow.Content().Refresh()
		capture(t, w2.fyneWindow.Canvas(), "pictures-frames-"+size.name+".png")
		w2.togglePane("characters")
		p := w2.charactersPane()
		p.update(w2.textEditor.Text())
		p.read()
		p.refresh()
		w2.fyneWindow.Content().Refresh()
		capture(t, w2.fyneWindow.Canvas(), "pictures-characters-"+size.name+".png")
		p.openChar("ANNA")
		w2.fyneWindow.Content().Refresh()
		capture(t, w2.fyneWindow.Canvas(), "pictures-character-"+size.name+".png")
	}
}
