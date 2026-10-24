package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/tooltip"
)

func TestOverflowRow(t *testing.T) {
	test.NewTempApp(t)
	tapped := ""
	var items []overflowItem
	var buttons []*tooltip.Button
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		b := tooltip.NewButton(theme.HomeIcon(), name+" (Ctrl+X)", func() { tapped = name }, nil)
		buttons = append(buttons, b)
		items = append(items, buttonItem(b))
	}
	title := widget.NewLabel("x")
	r := newOverflowRow(nil, items, title, 2)
	w := buttons[0].MinSize().Width + theme.Padding()

	resize := func(width float32) { // laid out again: an earlier test's theme may still be settling
		r.Container.Resize(fyne.NewSize(width, 40))
		r.Layout(r.Container.Objects, r.Container.Size())
	}
	resize(20 * w) // room for all, and the title
	if r.more.Visible() || !title.Visible() || r.first != 5 {
		t.Fatalf("wide: more %v, title %v, first %d", r.more.Visible(), title.Visible(), r.first)
	}
	if title.Position().X != 2*w {
		t.Errorf("the title is at %v, want after two buttons (%v)", title.Position().X, 2*w)
	}

	resize(3.5 * w) // two buttons and "more"
	if !r.more.Visible() || title.Visible() || r.first != 2 {
		t.Fatalf("narrow: more %v, title %v, first %d (button %v, more %v, padding %v, row %v)", r.more.Visible(), title.Visible(), r.first, buttons[0].MinSize(), r.more.MinSize(), theme.Padding(), r.Container.Size())
	}
	for i, b := range buttons {
		if b.Visible() != (i < 2) {
			t.Errorf("narrow: button %d visible %v", i, b.Visible())
		}
	}
	if m := items[3].menu(r.more); m[0].Label != "d" {
		t.Errorf("the menu calls d %q, want its tip without the shortcut", m[0].Label)
	} else if m[0].Action(); tapped != "d" {
		t.Errorf("the menu item tapped %q", tapped)
	}
	if got := r.MinSize(r.Container.Objects).Width; got != r.more.MinSize().Width {
		t.Errorf("min width %v, want the more button's %v", got, r.more.MinSize().Width)
	}
}
