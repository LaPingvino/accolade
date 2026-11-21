package buffer

import (
	"math/rand"
	"strings"
	"testing"
)

// A peer's edit is no undo step here: Undo undoes this buffer's own edits,
// moved past the peer's, and leaves the peer's text alone.
func TestRemoteNotUndone(t *testing.T) {
	b := New("INT. BARN - DAY\n\nRain falls.\n")
	b.SetCursor(len("INT. BARN - DAY\n\nRain"), false)
	typeText(b, " hard")
	b.Remote(0, 0, "FADE IN:\n\n") // above it
	b.Remote(len(b.Text())-1, len(b.Text())-1, " Thunder.")
	if want := "FADE IN:\n\nINT. BARN - DAY\n\nRain hard falls. Thunder.\n"; b.Text() != want {
		t.Fatalf("text %q", b.Text())
	}
	if want := len("FADE IN:\n\nINT. BARN - DAY\n\nRain hard"); b.Cursor() != want {
		t.Errorf("cursor %d, want %d", b.Cursor(), want)
	}
	for b.Undo() {
	}
	if want := "FADE IN:\n\nINT. BARN - DAY\n\nRain falls. Thunder.\n"; b.Text() != want {
		t.Errorf("undone to %q", b.Text())
	}
	if want := len("FADE IN:\n\nINT. BARN - DAY\n\nRain"); b.Cursor() != want {
		t.Errorf("cursor after undo %d, want %d", b.Cursor(), want)
	}
	b.Redo() // the space
	b.Remote(0, len("FADE IN:\n\n"), "") // a peer takes theirs away again
	b.Redo() // "hard"
	if want := "INT. BARN - DAY\n\nRain hard falls. Thunder.\n"; b.Text() != want {
		t.Errorf("redone to %q", b.Text())
	}
	b.Undo()
	b.Undo()
	if want := "INT. BARN - DAY\n\nRain falls. Thunder.\n"; b.Text() != want {
		t.Errorf("undone again to %q", b.Text())
	}
}

// A peer's edit inside a step's own text: that step (and older ones)
// cannot be undone faithfully, and go; newer ones stay.
func TestRemoteOverlapDropsStep(t *testing.T) {
	b := New("")
	b.Insert("INT. BARN")
	b.SetCursor(len(b.Text()), false)
	b.Insert("\n\nRain.")
	b.Remote(2, 4, "XY") // in "INT. BARN"
	if !b.Undo() || b.Text() != "INXY BARN" {
		t.Fatalf("undid to %q", b.Text())
	}
	if b.Undo() {
		t.Errorf("the overlapped step was undone: %q", b.Text())
	}
}

// Redo steps move past a peer's edit too.
func TestRemoteRedo(t *testing.T) {
	b := New("one two three")
	b.Select(4, 7)
	b.Insert("2")
	b.Undo() // "one two three", redo: two -> 2
	b.Remote(0, 3, "ONE!")
	b.Remote(len(b.Text()), len(b.Text()), " four")
	if !b.Redo() || b.Text() != "ONE! 2 three four" {
		t.Errorf("redone to %q", b.Text())
	}
	b.Undo()
	b.Remote(5, 6, "T") // in the step's text: Redo goes
	if b.Redo() {
		t.Errorf("an overlapped redo was redone: %q", b.Text())
	}
}

// Random local edits and peers' edits: an Undo never takes away a peer's
// text (the peers type R, this buffer only x), and undoing every step
// left and redoing them all comes back to the same text.
func TestRemoteRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for round := 0; round < 500; round++ {
		b := New("abcdefghij")
		for i := 0; i < 14; i++ {
			l := len(b.text)
			p := rng.Intn(l + 1)
			q := min(l, p+rng.Intn(3))
			switch rng.Intn(3) {
			case 0:
				b.Select(p, q)
				b.Insert(strings.Repeat("x", rng.Intn(3)+1))
			case 1:
				b.Remote(p, p, strings.Repeat("R", rng.Intn(2)+1))
			default:
				b.Remote(p, q, "")
			}
		}
		was := b.Text()
		for {
			rs := strings.Count(b.Text(), "R")
			if !b.Undo() {
				break
			}
			if strings.Count(b.Text(), "R") < rs {
				t.Fatalf("round %d: an undo took a peer's text: %q", round, b.Text())
			}
		}
		for b.Redo() {
		}
		if b.Text() != was {
			t.Fatalf("round %d: undo all, redo all: %q, was %q", round, b.Text(), was)
		}
	}
}
