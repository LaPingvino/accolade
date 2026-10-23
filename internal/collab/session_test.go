package collab

import (
	"strings"
	"testing"
)

func TestKey(t *testing.T) {
	k, err := NewKey()
	if err != nil {
		t.Fatal(err)
	}
	link := "https://lapingvino.github.io/accolade/#k=" + k.String()
	if back, err := ParseKey(link); err != nil || back != k {
		t.Fatalf("link round trip: %v", err)
	}
	sealed, _ := k.Seal([]byte("INT. BARN"))
	if strings.Contains(string(sealed), "BARN") {
		t.Error("the message is readable")
	}
	if msg, err := k.Open(sealed); err != nil || string(msg) != "INT. BARN" {
		t.Errorf("open: %q, %v", msg, err)
	}
	other, _ := NewKey()
	if _, err := other.Open(sealed); err == nil {
		t.Error("another key opened it")
	}
	if k.Room() == other.Room() {
		t.Error("two keys, one room")
	}
}

// Three peers in a chain (a-b-c): edits reach everyone, also a peer that
// joins late, and they end with one text.
func TestSessions(t *testing.T) {
	key, _ := NewKey()
	s := map[string]*Session{}
	for _, n := range []string{"a", "b", "c"} {
		s[n] = NewSession(key, New(n))
	}
	link := func(x, y string) {
		s[x].AddPeer(y, func(b []byte) error { return s[y].Receive(x, b) })
		s[y].AddPeer(x, func(b []byte) error { return s[x].Receive(y, b) })
	}
	var a []Op
	s["a"].Do(func(d *Doc) { a = d.SetText("INT. BARN - DAY\n\nRain.\n") })
	link("a", "b") // b catches up on joining
	link("b", "c")
	s["a"].Do(func(d *Doc) { a = d.SetText("INT. BARN - NIGHT\n\nRain.\n") })
	s["a"].Local(a)
	var c []Op
	s["c"].Do(func(d *Doc) { c = d.SetText("INT. BARN - NIGHT\n\nRain falls.\n") })
	s["c"].Local(c)
	want := "INT. BARN - NIGHT\n\nRain falls.\n"
	for n, x := range s {
		x.Do(func(d *Doc) {
			if d.Text() != want {
				t.Errorf("%s: %q", n, d.Text())
			}
		})
	}
}

// A script replaced, or caught up on, goes in messages a data channel takes.
func TestSessionsLargeEdits(t *testing.T) {
	key, _ := NewKey()
	a, b := NewSession(key, New("a")), NewSession(key, New("b"))
	small := func(to *Session, from string) func([]byte) error {
		return func(m []byte) error {
			if len(m) > 16<<10 {
				t.Fatalf("a message of %d bytes", len(m))
			}
			return to.Receive(from, m)
		}
	}
	text := strings.Repeat("INT. BARN - DAY\n\nRain.\n\n", 400)
	a.Do(func(d *Doc) { d.SetText(text) })
	_ = a.AddPeer("b", small(b, "a"))
	_ = b.AddPeer("a", small(a, "b"))
	var ops []Op
	text = strings.Repeat("EXT. FIELD - NIGHT\n\nWind.\n\n", 400)
	a.Do(func(d *Doc) { ops = d.SetText(text) })
	a.Local(ops)
	b.Do(func(d *Doc) {
		if d.Text() != text {
			t.Errorf("b has %d runes, want %d", len([]rune(d.Text())), len([]rune(text)))
		}
	})
}
