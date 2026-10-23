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
	want := ".PAGE ONE\n\nPanel 1. Rain.\n\nPanel 2. \n\n===\n\n.PAGE TWO\n\nPanel 1. "
	if b.Text() != want {
		t.Errorf("got %q\nwant %q", b.Text(), want)
	}
}

// Series acts as Lexington's example has them; a song counts songs, not
// every transition.
func TestInsertButtonSeries(t *testing.T) {
	series := defaultFormatButtons["series"]
	b := buffer.New("")
	insertButton(b, series[2]) // Act
	insertButton(b, series[3]) // End of Act
	insertButton(b, series[2])
	if want := "# ACT ONE\n\n>ACT ONE<\n\n>END OF ACT ONE<\n\n===\n\n# ACT TWO\n\n>ACT TWO<"; b.Text() != want {
		t.Errorf("got %q", b.Text())
	}
	musical := defaultFormatButtons["musical"]
	b = buffer.New("> 1. OPENING <\n\n> BLACKOUT.")
	b.SetCursor(b.Len(), false)
	insertButton(b, musical[1])
	if want := "> 1. OPENING <\n\n> BLACKOUT.\n\n> 2.  <"; b.Text() != want {
		t.Errorf("got %q", b.Text())
	}
}

// Lyrics go on the next line, under the singer.
func TestInsertLyrics(t *testing.T) {
	lyrics := defaultFormatButtons["musical"][2]
	b := buffer.New("ANNA")
	b.SetCursor(b.Len(), false)
	insertButton(b, lyrics)
	b.Insert("Who took the biscuit?")
	insertButton(b, lyrics)
	if want := "ANNA\n~Who took the biscuit?\n~"; b.Text() != want {
		t.Errorf("got %q", b.Text())
	}
}
