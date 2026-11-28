// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fmt"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// Paging through a source with Next never asks for more height than the
// window has (Joop: "paging through the adaptation at some point nudges the
// ribbon up and makes a part unreachable"; on the desktop a window whose
// content needs more grows, its top edge off the screen).
func TestPagingKeepsWindowHeight(t *testing.T) {
	// a half-screen tile: side by side (960 wide) or top and bottom (520 high)
	for _, size := range []fyne.Size{{Width: 1200, Height: 800}, {Width: 960, Height: 1040}, {Width: 640, Height: 800}, {Width: 1920, Height: 520}, {Width: 960, Height: 520}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) { pagingKeepsHeight(t, size) })
	}
}

func pagingKeepsHeight(t *testing.T, size fyne.Size) {
	var b strings.Builder
	b.WriteString("# The Story\n\n")
	for i := 0; i < 24; i++ {
		switch i % 3 {
		case 0:
			fmt.Fprintf(&b, "Short line %d.\n\n", i)
		case 1:
			fmt.Fprintf(&b, "%s\n\n", strings.Repeat(fmt.Sprintf("A long, winding passage number %d with many words and commas, ", i), 12))
		default:
			fmt.Fprintf(&b, "\"Who is there?\" asked Anna %d, and nobody answered for a long while; the rain kept falling and falling.\n\n", i)
		}
	}
	w, _ := sourceWindow(t, sourceScript, b.String())
	w.fyneWindow.Resize(size)
	content := w.fyneWindow.Content()
	if size.Height == 520 && size.Width == 960 {
		defer func() { capture(t, w.fyneWindow.Canvas(), "paging-half-tile.png") }()
	}
	worst, at := float32(0), ""
	worstW, atW := float32(0), ""
	report := func(step string) {
		m := content.MinSize()
		if m.Height > worst {
			worst, at = m.Height, step
			if m.Height > size.Height {
				t.Logf("over at %s:", step)
				dumpTallest(t, content, 60)
			}
		}
		if m.Width > worstW {
			worstW, atW = m.Width, step
		}
	}
	report("start")
	for i := 0; i < 30; i++ {
		w.nextSourcePassage()
		report(fmt.Sprintf("Next %d (status %q)", i+1, w.statusText()))
		if i%2 == 0 { // adapt every other one: Adapt… accepted with its defaults
			w.adaptSourcePassage()
			if d := w.sourcePane().adapting; d != nil {
				report(fmt.Sprintf("Adapt dialog %d", i+1))
				d.accept()
				report(fmt.Sprintf("Adapted %d (status %q)", i+1, w.statusText()))
			}
		}
	}
	if worstW > size.Width {
		t.Errorf("the content asks for %.0f px wide of %.0f, at %s", worstW, size.Width, atW)
	}
	if worst > size.Height {
		t.Errorf("the content asks for %.0f px of %.0f, at %s", worst, size.Height, at)
		dumpTallest(t, content, size.Height)
	}
}

// dumpTallest lists the objects whose MinSize height is over limit, the
// deepest last, to name the one that grows.
func dumpTallest(t *testing.T, o fyne.CanvasObject, limit float32) {
	var walk func(o fyne.CanvasObject, depth int)
	walk = func(o fyne.CanvasObject, depth int) {
		if o == nil || !o.Visible() {
			return
		}
		if h := o.MinSize().Height; h > limit/3 {
			t.Logf("%s%T min %.0f", strings.Repeat("  ", depth), o, h)
		}
		var kids []fyne.CanvasObject
		switch c := o.(type) {
		case *fyne.Container:
			kids = c.Objects
		case fyne.Widget:
			kids = test.WidgetRenderer(c).Objects()
		}
		for _, k := range kids {
			walk(k, depth+1)
		}
	}
	walk(o, 0)
}
