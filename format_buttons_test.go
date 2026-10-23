package main

import (
	"testing"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// A format's buttons insert a paragraph of their own, numbered.
func TestInsertButton(t *testing.T) {
	comic := defaultFormatButtons["comic"]
	page, panel := comic[0], comic[1]
	b := buffer.New("")
	insertButton(b, page)
	insertButton(b, panel)
	b.Insert("Rain.")
	insertButton(b, panel)
	insertButton(b, page)
	insertButton(b, panel)
	want := ".PAGE 1\n\nPanel 1. Rain.\n\nPanel 2. \n\n===\n\n.PAGE 2\n\nPanel 1. "
	if b.Text() != want {
		t.Errorf("got %q\nwant %q", b.Text(), want)
	}
}
