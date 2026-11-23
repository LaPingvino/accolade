package collab

import (
	"encoding/json"
	"math/rand"
	"testing"
)

// mapOf is s's map name as "k=v;" in key order.
func mapOf(s *Session, name string) string {
	b, _ := json.Marshal(s.MapOf(name))
	return string(b)
}

// meet connects two sessions as a network does: what each sends waits
// until both have the other as a peer (link's first hello has no answer).
func meet(x, y *Session, xn, yn string) {
	ready := false
	var queued []func()
	via := func(to *Session, from string) func([]byte) error {
		return func(b []byte) error {
			if !ready {
				queued = append(queued, func() { _ = to.Receive(from, b) })
				return nil
			}
			return to.Receive(from, b)
		}
	}
	x.AddPeer(yn, via(y, xn))
	y.AddPeer(xn, via(x, yn))
	ready = true
	for _, f := range queued {
		f()
	}
}

// Shared maps: set and taken away in a chain of three, each key's last
// writer winning on every peer; two writing one key at once end alike;
// different keys both kept; a peer joining catches up; a kept state
// carries them; entries delivered shuffled and twice end the same.
func TestSharedMaps(t *testing.T) {
	key, _ := NewKey()
	a, b, c := NewSession(key, New("a")), NewSession(key, New("b")), NewSession(key, New("c"))
	a.MapSet("bd", map[string]string{"t1": "lamp", "t2": "rope"}, nil)
	meet(a, b, "a", "b")
	meet(b, c, "b", "c")
	if got := mapOf(c, "bd"); got != `{"t1":"lamp","t2":"rope"}` {
		t.Fatalf("c joined with %s", got)
	}
	c.MapSet("bd", map[string]string{"t3": "hat"}, []string{"t1"})
	for _, s := range []*Session{a, b, c} {
		if got := mapOf(s, "bd"); got != `{"t2":"rope","t3":"hat"}` {
			t.Errorf("after c's change: %s", got)
		}
	}
	// a and d write one key apart, then meet: one value on both
	d := NewSession(key, New("d"))
	d.MapSet("bd", map[string]string{"t2": "chain", "t4": "cup"}, nil)
	a.MapSet("bd", map[string]string{"t2": "string"}, nil)
	meet(a, d, "a", "d")
	want := mapOf(a, "bd")
	for _, s := range []*Session{b, c, d} {
		if got := mapOf(s, "bd"); got != want {
			t.Errorf("not alike: %s and %s", got, want)
		}
	}
	if m := a.MapOf("bd"); m["t4"] != "cup" || m["t3"] != "hat" {
		t.Errorf("a key lost: %v", m)
	}
	// kept and taken up again
	st, _ := c.State()
	r, err := Restore(key, "r", st)
	if err != nil || mapOf(r, "bd") != mapOf(c, "bd") {
		t.Fatalf("restored %s, %v", mapOf(r, "bd"), err)
	}
	// shuffled and repeated: the same
	var all []Entry
	a.mu.Lock()
	all = a.maps["bd"].missing(nil)
	a.mu.Unlock()
	rng := rand.New(rand.NewSource(1))
	for range 20 {
		x := newSharedMap()
		es := append(append([]Entry(nil), all...), all...)
		rng.Shuffle(len(es), func(i, j int) { es[i], es[j] = es[j], es[i] })
		for _, e := range es {
			x.apply(e)
		}
		got, _ := json.Marshal(x.values())
		if string(got) != want {
			t.Fatalf("shuffled: %s, not %s", got, want)
		}
	}
	// OnMap is told
	told := ""
	b.OnMap = func(name string) { told = name }
	a.MapSet("bd", nil, []string{"t4"})
	if told != "bd" || b.MapOf("bd")["t4"] != "" {
		t.Errorf("OnMap %q; b has %v", told, b.MapOf("bd"))
	}
}
