// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

const cursorsScript = "INT. BARN - DAY\n\nRain.\n\nINT. LOFT - NIGHT\n\nHay in the loft.\n\nEXT. FIELD - DAY\n\nSun.\n"

// cursorsPair is a and b sharing cursorsScript, connected, Ann and Bob
// (one app's windows have one name: each says it as theirs, sayAs).
func cursorsPair(t *testing.T) (a, b *MainWindow) {
	t.Helper()
	a, b, _ = sharedPair(t, cursorsScript)
	sayAs(a, "Ann")
	sayAs(b, "Bob")
	connect(t, a, b)
	textIs(t, b, "b", a.textEditor.Text())
	return a, b
}

// sayAs says w's place as name.
func sayAs(w *MainWindow, name string) {
	w.settings.SetString("your-name", name)
	w.sayHere(true)
}

// Bob's place in Ann's editor: a bar on his line, moved with the text as
// Ann types above it (before he says anything again), his selection with
// it; a dot on his scene in the outline and the cards; gone when he
// leaves.
func TestCursorsTogether(t *testing.T) {
	a, b := cursorsPair(t)
	hay := len([]rune(cursorsScript[:strings.Index(cursorsScript, "Hay")]))
	b.textEditor.Select(hay, hay+3) // "Hay", the cursor after it
	b.onCursorChanged()
	waitFor(t, "Bob on line 6 at a", func() bool { return a.textEditor.PeerLines()[6] == "Bob" })
	waitFor(t, "Bob's selection at a", func() bool {
		p := a.textEditor.Peers()
		return len(p) == 1 && p[0].Cursor == hay+3 && p[0].From == hay
	})
	a.fyneWindow.Canvas().Focus(a.textEditor)
	a.textEditor.SetCursorOffset(0)
	test.Type(a.textEditor, "FADE IN:\n\n")
	if got := a.textEditor.PeerLines(); got[8] != "Bob" {
		t.Errorf("after Ann typed above: %v", got)
	}
	textIs(t, b, "b", a.textEditor.Text())
	// a's edit came to b, whose cursor moved with it and said so: still on Hay
	waitFor(t, "Bob still at Hay", func() bool {
		p := a.textEditor.Peers()
		return len(p) == 1 && p[0].Cursor == hay+13 && p[0].From == hay+10
	})
	// Bob's own edit above, his selection kept: placed from the anchors
	b.textEditor.Edit(func(buf *buffer.Buffer) {
		from, to := buf.Selection()
		buf.Replace(0, 0, "Title: Hay\n\n")
		buf.Select(from+12, to+12)
	})
	waitFor(t, "a has b's edit", func() bool { return strings.HasPrefix(a.textEditor.Text(), "Title: Hay") })
	waitFor(t, "Bob on line 10 at a", func() bool { return a.textEditor.PeerLines()[10] == "Bob" })
	// the outline's dot and the cards'
	a.togglePane("outline")
	a.outline.refresh()
	if d := a.outline.dots; len(d) != 1 {
		t.Errorf("outline dots %v", d)
	} else {
		for i := range d {
			if !strings.Contains(a.outline.items[i].Text, "LOFT") {
				t.Errorf("Bob's dot on %q", a.outline.items[i].Text)
			}
		}
	}
	a.cardsPane().update(a.textEditor.Text())
	if d := a.cardsDock.dots; len(d) != 1 {
		t.Errorf("cards dots %v", d)
	}
	// Bob types along his line: nothing said; Enter: his new line, exactly
	b.fyneWindow.Canvas().Focus(b.textEditor)
	b.textEditor.SetCursorOffset(b.textEditor.CursorOffset()) // the selection gone
	b.sayHere(false)
	seq := func() uint64 { return a.hereNow()[0].Seq }
	waitFor(t, "Bob's cursor alone", func() bool { p := a.textEditor.Peers(); return len(p) == 1 && p[0].From == p[0].Cursor })
	before := seq()
	test.Type(b.textEditor, "ab")
	textIs(t, a, "a", b.textEditor.Text())
	if seq() != before {
		t.Errorf("typing along a line said the place %d times", seq()-before)
	}
	end := len([]rune(b.textEditor.Text()[:strings.Index(b.textEditor.Text(), "loft.")+len("loft.")]))
	b.textEditor.SetCursorOffset(end)
	test.Type(b.textEditor, "\n")
	textIs(t, a, "a", b.textEditor.Text())
	waitFor(t, "Bob on his new line", func() bool {
		p := a.textEditor.Peers()
		return len(p) == 1 && p[0].Cursor == end+1 && a.textEditor.PeerLines()[11] == "Bob"
	})
	// the pointer over his bar: his name, kept (not taken away by the text's hover)
	at, cell, ok := a.textEditor.PeerBarAt(11)
	if !ok {
		t.Fatal("no bar on line 11")
	}
	a.textEditor.MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: at.AddXY(cell.Width/2, cell.Height/2)}})
	if a.tooltips == nil || a.noteTipShown != "Bob is here" {
		t.Errorf("the bar's tip: %q", a.noteTipShown)
	}
	b.leaveSession()
	waitFor(t, "Bob gone from a's editor", func() bool { return len(a.textEditor.PeerLines()) == 0 })
	if len(a.outline.dots) != 0 {
		t.Errorf("outline dots after Bob left: %v", a.outline.dots)
	}
	a.leaveSession()
	if len(a.textEditor.Peers()) != 0 {
		t.Error("peers left after leaving")
	}
}

// TestCursorsScreenshots: two windows sharing, each with the other's
// place shown (SHOT_DIR), desktop and phone.
func TestCursorsScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	a, b := cursorsPair(t)
	a.showPane("outline", false)
	b.showPane("cards", false)
	hay := len([]rune(cursorsScript[:strings.Index(cursorsScript, "Hay")]))
	b.textEditor.Select(hay, hay+len("Hay in the loft"))
	sayAs(b, "Bob")
	a.textEditor.SetCursorOffset(len([]rune(cursorsScript[:strings.Index(cursorsScript, "Sun")])))
	sayAs(a, "Ann")
	waitFor(t, "each sees the other", func() bool {
		return a.textEditor.PeerLines()[6] == "Bob" && b.textEditor.PeerLines()[10] == "Ann"
	})
	for _, size := range []struct {
		name string
		s    fyne.Size
	}{{"desktop", fyne.NewSize(1100, 650)}, {"phone", fyne.NewSize(380, 800)}} {
		for _, w := range []struct {
			n string
			w *MainWindow
		}{{"ann", a}, {"bob", b}} {
			w.w.fyneWindow.Resize(size.s)
			w.w.onWindowWidth(size.s.Width)
			w.w.showPeers()
			w.w.fyneWindow.Content().Refresh()
			f, err := os.Create(filepath.Join(dir, "collab-cursors-"+w.n+"-"+size.name+".png"))
			if err != nil {
				t.Fatal(err)
			}
			_ = png.Encode(f, w.w.fyneWindow.Canvas().Capture())
			f.Close()
		}
	}
}
