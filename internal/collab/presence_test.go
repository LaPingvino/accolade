package collab

import (
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"
)

// names is who s has here, by name.
func names(s *Session) []string {
	var out []string
	for _, p := range s.Here() {
		out = append(out, p.Name)
	}
	return out
}

// Who is here: told on joining (also of a peer two away, in a chain
// a-b-c), each cursor line, gone when they leave, when the peer they came
// through goes, or after HereFor without a word; a rejoin is listed once;
// nothing of it is kept in the state.
func TestPresence(t *testing.T) {
	key, _ := NewKey()
	now := time.Unix(1000, 0)
	s := map[string]*Session{}
	var changes atomic.Int32 // OnHere runs on a goroutine of its own (hereChanged)
	for _, n := range []string{"a", "b", "c"} {
		s[n] = NewSession(key, New(n))
		s[n].Now = func() time.Time { return now }
		s[n].OnHere = func() { changes.Add(1) }
	}
	link := func(x, y string) {
		s[x].AddPeer(y, func(b []byte) error { return s[y].Receive(x, b) })
		s[y].AddPeer(x, func(b []byte) error { return s[x].Receive(y, b) })
	}
	cut := func(x, y string) { s[x].RemovePeer(y); s[y].RemovePeer(x) }
	s["a"].SetHere("Ann", 3)
	s["b"].SetHere("Bob", 0)
	link("a", "b")
	link("b", "c")
	s["c"].SetHere("Cy", 7)
	if got := names(s["a"]); len(got) != 2 || got[0] != "Bob" || got[1] != "Cy" {
		t.Fatalf("a has %v", got)
	}
	if got := names(s["c"]); len(got) != 2 || got[0] != "Ann" {
		t.Fatalf("c has %v (a two away, told on joining)", got)
	}
	if changes.Load() == 0 {
		t.Error("OnHere not called")
	}
	s["a"].SetHere("Ann", 12)
	if p := s["c"].Here()[0]; p.Line != 12 {
		t.Errorf("c sees Ann on line %d", p.Line)
	}
	// c leaves: gone at once
	s["c"].Leave()
	if got := names(s["a"]); len(got) != 1 || got[0] != "Bob" {
		t.Errorf("after c left, a has %v", got)
	}
	cut("b", "c")
	// b cut off from a: gone from a's list, a from b's
	cut("a", "b")
	if got := names(s["a"]); len(got) != 0 {
		t.Errorf("cut off, a still has %v", got)
	}
	// b rejoins as another replica (a new window): listed once
	s["b"] = NewSession(key, New("b2"))
	s["b"].Now = func() time.Time { return now }
	s["b"].SetHere("Bob", 1)
	link("a", "b")
	if got := names(s["a"]); len(got) != 1 {
		t.Errorf("rejoined, a has %v", got)
	}
	// silence: gone after HereFor; a word again brings them back
	now = now.Add(HereFor)
	if got := names(s["a"]); len(got) != 0 {
		t.Errorf("after HereFor of silence, a has %v", got)
	}
	s["b"].SetHere("Bob", 1)
	if got := names(s["a"]); len(got) != 1 {
		t.Errorf("heard again, a has %v", got)
	}
	// none of it in the kept state
	b, err := s["a"].State()
	if err != nil {
		t.Fatal(err)
	}
	var st map[string]any
	_ = json.Unmarshal(b, &st)
	if meta, _ := st["meta"].(map[string]any); len(meta) != 0 {
		t.Errorf("kept: %v", meta)
	}
}

// A presence alone is nothing to a peer of protocol 5 before it (no
// mismatch, no change): the message's other fields are empty.
func TestPresenceOldPeer(t *testing.T) {
	key, _ := NewKey()
	s := NewSession(key, New("a"))
	mismatch := false
	s.OnMismatch = func(int, string) { mismatch = true }
	m, _ := key.Seal([]byte(`{"h":{"r":"b","n":"Bob","l":2,"s":1}}`))
	if err := s.Receive("b", m); err != nil || mismatch {
		t.Fatalf("a presence: %v, mismatch %v", err, mismatch)
	}
	if got := names(s); len(got) != 1 || got[0] != "Bob" {
		t.Errorf("here: %v", got)
	}
	var old message
	_ = json.Unmarshal([]byte(`{"h":{"r":"b"}}`), &old)
	if old.Hello || len(old.Runs) > 0 || len(old.Meta) > 0 {
		t.Error("a presence reads as more than itself")
	}
}

// A viewer says so (a Table Read following only): the others see it.
func TestPresenceViewer(t *testing.T) {
	key, _ := NewKey()
	a, b := NewSession(key, New("a")), NewSession(key, New("b"))
	b.Viewer = true
	a.AddPeer("b", func(m []byte) error { return b.Receive("a", m) })
	b.AddPeer("a", func(m []byte) error { return a.Receive("b", m) })
	a.SetHere("Ann", 4)
	b.SetHere("TV", 0)
	if h := a.Here(); len(h) != 1 || !h[0].View || h[0].Name != "TV" {
		t.Errorf("a sees %+v", h)
	}
	if h := b.Here(); len(h) != 1 || h[0].View || h[0].Line != 4 {
		t.Errorf("b sees %+v", h)
	}
}

// A cursor said as an anchor stays with its text: a peer's edit before
// it, a deletion of the rune it comes after, and the start; the selection
// comes with it, and a peer whose ops have not come yet has the line.
func TestPresenceAnchors(t *testing.T) {
	key, _ := NewKey()
	a, b := NewSession(key, New("a")), NewSession(key, New("b"))
	edit := func(f func(d *Doc) []Op) {
		var ops []Op
		a.Do(func(d *Doc) { ops = f(d) })
		a.Local(ops)
	}
	edit(func(d *Doc) []Op { return d.Insert(0, "INT. BARN - DAY\n\nRain falls.\n") })
	link(a, b, "a", "b")
	at := len([]rune("INT. BARN - DAY\n\nRain"))
	b.SetHereAt("Bob", 2, at, at-4) // "Rain" selected, the cursor after it
	place := func(s *Session) (cur, from int) {
		p := s.Here()[0]
		if p.At == nil || p.From == nil {
			t.Fatalf("no place said: %+v", p)
		}
		s.Do(func(d *Doc) {
			cur, _ = d.Place(*p.At)
			from, _ = d.Place(*p.From)
		})
		return cur, from
	}
	if cur, from := place(a); cur != at || from != at-4 {
		t.Fatalf("a sees Bob at %d from %d, not %d from %d", cur, from, at, at-4)
	}
	edit(func(d *Doc) []Op { return d.Insert(0, "FADE IN:\n\n") }) // before it: it moves with the text
	if cur, _ := place(a); cur != at+10 {
		t.Errorf("after an edit before it, at %d, not %d", cur, at+10)
	}
	edit(func(d *Doc) []Op { return d.Delete(at+10-1, 1) }) // the rune it comes after: where it was
	if cur, _ := place(a); cur != at+10-1 {
		t.Errorf("after its rune went, at %d, not %d", cur, at+9)
	}
	b.SetHereAt("Bob", 0, 0, 0)
	if p := a.Here()[0]; p.At == nil || *p.At != (ID{}) || p.From != nil {
		t.Errorf("at the start: %+v", p)
	}
	c := NewSession(key, New("c")) // nothing of the script here: the ID is not, the line is
	var id ID
	a.Do(func(d *Doc) { id = d.Anchor(5) })
	c.Do(func(d *Doc) {
		if _, ok := d.Place(id); ok {
			t.Error("placed an ID not here")
		}
	})
	if b, _ := json.Marshal(Presence{Replica: "x", Line: 2, Seq: 1}); string(b) != `{"r":"x","l":2,"s":1}` {
		t.Errorf("a presence without a place: %s", b)
	}
}
