package collab

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func textIn(s *Session, name string) (t string) {
	s.DoIn(name, func(d *Doc) { t = d.Text() })
	return t
}

// A session's named Docs: a peer joining gets each, edits in one reach
// everyone in a chain (a-b-c) and say which Doc they are of; the script's
// stays apart.
func TestNamedDocs(t *testing.T) {
	key, _ := NewKey()
	s := map[string]*Session{}
	heard := map[string][]string{}
	for _, n := range []string{"a", "b", "c"} {
		s[n] = NewSession(key, New(n))
		s[n].OnRemote = func(doc, by string) { heard[n] = append(heard[n], doc+"/"+by) }
	}
	s["a"].Do(func(d *Doc) { d.SetText("INT. BARN - DAY\n") })
	s["a"].DoIn("source:1", func(d *Doc) { d.SetText("# One\n\nRain fell on the barn.\n") })
	link(s["a"], s["b"], "a", "b")
	link(s["b"], s["c"], "b", "c")
	for n, x := range s {
		if textIn(x, Script) != "INT. BARN - DAY\n" || textIn(x, "source:1") != "# One\n\nRain fell on the barn.\n" {
			t.Fatalf("%s joined with %q and %q", n, textIn(x, Script), textIn(x, "source:1"))
		}
	}
	heard = map[string][]string{}
	var ops []Op
	s["c"].DoIn("source:1", func(d *Doc) { ops = d.SetText("# One\n\nRain fell hard on the barn.\n") })
	s["c"].LocalIn("source:1", ops)
	for n, x := range s {
		if textIn(x, "source:1") != "# One\n\nRain fell hard on the barn.\n" || textIn(x, Script) != "INT. BARN - DAY\n" {
			t.Errorf("%s: %q, script %q", n, textIn(x, "source:1"), textIn(x, Script))
		}
	}
	if got := strings.Join(heard["a"], ","); got != "source:1/c" {
		t.Errorf("a heard %q", got)
	}
}

// A peer joining while another types in a long source: the catch-up and
// the keys typed meanwhile, in any order, end alike on both.
func TestJoinMidEdit(t *testing.T) {
	key, _ := NewKey()
	a, b := NewSession(key, New("a")), NewSession(key, New("b"))
	novel := strings.Repeat("Rain fell on the barn, and the snow was grey.\n\n", 3000)
	a.DoIn("source:1", func(d *Doc) { d.SetText(novel) })
	// b's messages queued, delivered between a's keys
	var toB [][]byte
	a.AddPeer("b", func(m []byte) error { toB = append(toB, m); return nil })
	b.AddPeer("a", func(m []byte) error { return a.Receive("b", m) })
	for i := 0; len(toB) > 0 || i < 50; i++ {
		if i < 50 {
			var ops []Op
			a.DoIn("source:1", func(d *Doc) { ops = d.Insert(i*97, "x") })
			a.LocalIn("source:1", ops)
		}
		if len(toB) > 0 {
			m := toB[0]
			toB = toB[1:]
			if err := b.Receive("a", m); err != nil {
				t.Fatal(err)
			}
		}
	}
	if ta, tb := textIn(a, "source:1"), textIn(b, "source:1"); ta != tb || strings.Count(tb, "x") != 50 {
		t.Errorf("b has %d runes, %d x; a %d", len(tb), strings.Count(tb, "x"), len(ta))
	}
}

// The kept state holds every Doc; StateOf only those named (the script
// always).
func TestStateDocs(t *testing.T) {
	key, _ := NewKey()
	s := NewSession(key, New("a"))
	s.Do(func(d *Doc) { d.SetText("INT. BARN\n") })
	s.DoIn("source:1", func(d *Doc) { d.SetText("Rain.\n") })
	s.SetMeta("source", "source:1")
	all, _ := s.State()
	r, err := Restore(key, "b", all)
	if err != nil || textIn(r, Script) != "INT. BARN\n" || textIn(r, "source:1") != "Rain.\n" || r.Meta("source") != "source:1" {
		t.Fatalf("restored %q %q %q %v", textIn(r, Script), textIn(r, "source:1"), r.Meta("source"), err)
	}
	only, _ := s.StateOf(Script)
	r, _ = Restore(key, "c", only)
	if textIn(r, Script) != "INT. BARN\n" || r.Has("source:1") {
		t.Errorf("StateOf(script) has the source")
	}
}

// v4 is a protocol 4 peer's message, as it unmarshals one.
type v4 struct {
	Hello   bool              `json:"hello,omitempty"`
	Proto   int               `json:"p,omitempty"`
	Version map[string]uint64 `json:"v,omitempty"`
	Counts  map[string]int    `json:"n,omitempty"`
	App     string            `json:"app,omitempty"`
	Runs    []struct {
		ID    ID     `json:"id"`
		After ID     `json:"a,omitempty"`
		Text  string `json:"t,omitempty"`
		Del   int    `json:"d,omitempty"`
	} `json:"r,omitempty"`
	Meta map[string]Value `json:"m,omitempty"`
	By   string           `json:"by,omitempty"`
}

// Peers of protocol 4 and 5 tell each other apart, and neither takes the
// other's text: a protocol 4 peer reads the hello (it unmarshals into its
// message) and sees p=5; a v4 hello here is a mismatch, older, and what
// follows it is not listened to nor passed on.
func TestMixedProtocols(t *testing.T) {
	key, _ := NewKey()
	s := NewSession(key, New("a"))
	s.App = "1.5"
	s.Do(func(d *Doc) { d.SetText("INT. BARN\n") })
	s.DoIn("source:1", func(d *Doc) { d.SetText("Rain.\n") })
	var sent [][]byte
	s.AddPeer("old", func(m []byte) error { sent = append(sent, m); return nil })
	if len(sent) != 1 {
		t.Fatalf("%d messages for a hello", len(sent))
	}
	b, _ := key.Open(sent[0])
	var m v4
	if err := json.Unmarshal(b, &m); err != nil || !m.Hello || m.Proto != 5 || m.App != "1.5" {
		t.Fatalf("a v4 peer reads %+v, %v", m, err)
	}
	// a v4 peer's hello and edits
	relayed := 0
	s.AddPeer("c", func([]byte) error { relayed++; return nil })
	relayed = 0
	theirs := -1
	s.OnMismatch = func(p int, app string) { theirs = p }
	hello, _ := key.Seal([]byte(`{"hello":true,"p":4,"app":"1.4","v":{"x":3},"n":{"x":3}}`))
	if s.Receive("old", hello) == nil || theirs != 4 {
		t.Fatalf("a v4 hello: mismatch %d", theirs)
	}
	ops, _ := key.Seal([]byte(`{"r":[{"id":{"c":1,"r":"x"},"t":"old"}]}`))
	_ = s.Receive("old", ops)
	if textIn(s, Script) != "INT. BARN\n" || textIn(s, "source:1") != "Rain.\n" || relayed != 0 {
		t.Errorf("listened to it: %q, %d relayed", textIn(s, Script), relayed)
	}
	// two sessions of either protocol (the old one's proto set to 4)
	o := NewSession(key, New("o"))
	o.proto = 4
	var ours, olds = -1, -1
	s.OnMismatch = func(p int, _ string) { ours = p }
	o.OnMismatch = func(p int, _ string) { olds = p }
	link(s, o, "a", "o")
	if ours != 4 || olds != 5 || textIn(o, Script) != "" {
		t.Errorf("mismatch seen as %d and %d; o has %q", ours, olds, textIn(o, Script))
	}
}

// A Doc let go is not taken again from a peer that still has it.
func TestDrop(t *testing.T) {
	key, _ := NewKey()
	a, b := NewSession(key, New("a")), NewSession(key, New("b"))
	a.DoIn("source:old", func(d *Doc) { d.SetText("Old.\n") })
	b.Drop("source:old")
	link(a, b, "a", "b")
	if b.Has("source:old") || slices.Contains(b.Names(), "source:old") {
		t.Errorf("b took the dropped Doc: %v", b.Names())
	}
}
