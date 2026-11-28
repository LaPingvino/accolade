// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"context"
	"image"
	"image/draw"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/pdf"
	"github.com/LaPingvino/lexington/rules"
)

// The live sample of the format and settings editors: a short script laid
// out by Lexington itself (cli.Write, as an export draws its PDF) with
// the element set as the form has it now, its first page's top shown. It
// is drawn again a moment after a change, off the UI thread.

// sampleScript has one of each of the usual elements.
const sampleScript = `Title: The Sample
Credit: written by
Author: Anna Writer

# Act One

INT. KITCHEN - NIGHT

Rain on the window. ANNA (40s) stirs a pot.

ANNA
(quietly)
Did you hear that?

BEN (O.S.)
Only the wind.

ANNA ^
Listen.

~Somewhere over the rain

CUT TO:

>THE END<
`

// formatSample is a sample image and how to get the set it shows.
type formatSample struct {
	img  *canvas.Image
	note *widget.Label
	set  func() (rules.Set, bool)
	page string
	// titlePage: the title page is shown, not the first page after it
	titlePage bool

	mu    sync.Mutex
	timer *time.Timer
	gen   int
}

// newFormatSample is a sample of the set that set gives (false: none to
// show), on paper page ("" for Letter).
func newFormatSample(set func() (rules.Set, bool), page string) *formatSample {
	s := &formatSample{set: set, page: page}
	s.img = canvas.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 1, 1)))
	s.img.FillMode = canvas.ImageFillContain
	s.img.ScaleMode = canvas.ImageScaleSmooth
	s.img.SetMinSize(fyne.NewSize(120, 120))
	s.note = widget.NewLabel("Drawing the sample…")
	s.note.Importance = widget.LowImportance
	s.update()
	return s
}

// object is the sample with its caption.
func (s *formatSample) object() fyne.CanvasObject {
	return container.NewBorder(nil, s.note, nil, nil, s.img)
}

// update draws the sample again soon.
func (s *formatSample) update() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gen++
	gen := s.gen
	if s.timer != nil {
		s.timer.Stop()
	}
	set, ok := s.set()
	if !ok {
		s.note.SetText("No sample: the file has an error.")
		return
	}
	title := s.titlePage
	s.timer = time.AfterFunc(250*time.Millisecond, func() {
		img, err := drawSample(set, s.page, title)
		uiDo(func() {
			s.mu.Lock()
			current := gen == s.gen
			s.mu.Unlock()
			if !current {
				return
			}
			if err != nil {
				s.note.SetText("No sample: " + err.Error())
				return
			}
			s.img.Image = img
			s.img.Refresh()
			s.note.SetText("As Lexington prints it (the top of a page)")
		})
	})
}

var sampleDrawing sync.Mutex

// stop: the sample is no longer shown (its page drawn again, its editor
// closed); a drawing under way is dropped.
func (s *formatSample) stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gen++
	if s.timer != nil {
		s.timer.Stop()
	}
}

// drawSample is the top of the sample script's first page after its
// title page (or of the title page), laid out with set.
func drawSample(set rules.Set, page string, titlePage bool) (image.Image, error) {
	sampleDrawing.Lock() // one at a time: they share Lexington's fonts
	defer sampleDrawing.Unlock()
	conf := rules.DefaultConf()
	conf.Elements = map[string]rules.Set{"sample": set}
	dir, err := os.MkdirTemp("", "accolade-sample-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	var d pdf.Drawing
	o := cli.Options{Conf: &conf, Elements: "sample", To: "pdf", Output: filepath.Join(dir, "sample.pdf"), Record: &d, Page: page}
	script := fountain.Parse(rules.DefaultConf().Scenes("en"), strings.NewReader(sampleScript))
	if err := cli.Write(context.Background(), script, o); err != nil {
		return nil, err
	}
	if len(d.Pages) == 0 {
		return nil, os.ErrNotExist
	}
	i := 0
	if len(d.Pages) > 1 && !titlePage {
		i = 1 // the first page is the title page
	}
	full := d.Image(i, 1.2)
	b := full.Bounds()
	// the page's whole width (the margins are the point), down to a
	// little under its last line
	bottom := lastInked(full) + b.Dy()/20
	if bottom > b.Dy() || bottom <= b.Dy()/20 {
		bottom = b.Dy() / 2
	}
	top := image.Rect(0, 0, b.Dx(), bottom)
	out := image.NewRGBA(top)
	draw.Draw(out, top, full, b.Min, draw.Src)
	return out, nil
}

// lastInked is the last row of img with something on it (not white).
func lastInked(img *image.RGBA) int {
	b := img.Bounds()
	for y := b.Max.Y - 1; y >= b.Min.Y; y-- {
		for x := b.Min.X; x < b.Max.X; x++ {
			if c := img.RGBAAt(x, y); c.R < 200 || c.G < 200 || c.B < 200 {
				return y - b.Min.Y
			}
		}
	}
	return 0
}
