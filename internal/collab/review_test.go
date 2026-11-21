package collab

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// link connects two sessions directly, both ways.
func link(x, y *Session, xn, yn string) {
	x.AddPeer(yn, func(b []byte) error { return y.Receive(xn, b) })
	y.AddPeer(xn, func(b []byte) error { return x.Receive(yn, b) })
}

func text(s *Session) (t string) {
	s.Do(func(d *Doc) { t = d.Text() })
	return t
}

// Edits make old new, for random texts and changes.
func TestEditsRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	alpha := []rune("ab \nINT.é")
	randText := func(n int) string {
		r := make([]rune, n)
		for i := range r {
			r[i] = alpha[rng.Intn(len(alpha))]
		}
		return string(r)
	}
	for i := 0; i < 2000; i++ {
		old := []rune(randText(rng.Intn(40)))
		nw := append([]rune(nil), old...)
		for k := rng.Intn(4); k >= 0; k-- {
			p := rng.Intn(len(nw) + 1)
			q := min(len(nw), p+rng.Intn(5))
			nw = append(nw[:p:p], append([]rune(randText(rng.Intn(4))), nw[q:]...)...)
		}
		got := []rune(string(old))
		es := Edits(string(old), string(nw))
		last := -1
		for j := len(es) - 1; j >= 0; j-- {
			e := es[j]
			if e.Pos < last && j < len(es)-1 {
				t.Fatalf("edits out of order: %+v", es)
			}
			got = append(got[:e.Pos:e.Pos], append([]rune(e.Text), got[e.Pos+e.Del:]...)...)
		}
		for j := 1; j < len(es); j++ {
			if es[j].Pos < es[j-1].Pos+es[j-1].Del {
				t.Fatalf("edits overlap: %+v", es)
			}
		}
		if string(got) != string(nw) {
			t.Fatalf("%q -> %q: edits %+v made %q", string(old), string(nw), es, string(got))
		}
	}
}

// A script with numbered scenes, as Renumber leaves it.
func numbered(n int, num func(i int) string, mid string) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "INT. ROOM %d - DAY #%s#\n\nAction %d.\n\n", i, num(i), i)
		if i == n/2 {
			b.WriteString(mid)
		}
	}
	return b.String()
}

// A change all over the text (renumbering) made while a peer types in the
// middle: both stay, the peer's typing where it was typed.
func TestWholeTextChangeMerges(t *testing.T) {
	key, _ := NewKey()
	a, b := NewSession(key, New("a")), NewSession(key, New("b"))
	before := numbered(20, func(i int) string { return fmt.Sprint(i + 10) }, "")
	a.Do(func(d *Doc) { d.SetText(before) })
	link(a, b, "a", "b")
	// offline from each other for a moment: a renumbers, b types and cuts
	var opsA, opsB []Op
	a.Do(func(d *Doc) {
		opsA = d.SetText(numbered(20, func(i int) string { return fmt.Sprint(i) }, ""))
	})
	b.Do(func(d *Doc) {
		t := d.Text()
		t = strings.Replace(t, "Action 10.\n", "Action 10. She waits.\n", 1)
		t = strings.Replace(t, "Action 12.\n\n", "", 1)
		opsB = d.SetText(t)
	})
	if len(runsOf(opsA)) < 20 {
		t.Errorf("renumbering made %d runs: one edit over the text?", len(runsOf(opsA)))
	}
	a.Local(opsA)
	b.Local(opsB)
	want := strings.Replace(numbered(20, func(i int) string { return fmt.Sprint(i) }, ""), "Action 10.\n", "Action 10. She waits.\n", 1)
	want = strings.Replace(want, "Action 12.\n\n", "", 1)
	if text(a) != want || text(b) != want {
		t.Errorf("a: %q\nb: %q\nwant %q", text(a), text(b), want)
	}
}

// A peer that has a replica's later op and not an earlier one (applied out
// of order, then the link broke) gets the earlier one at the next hello.
func TestCatchUpGap(t *testing.T) {
	key, _ := NewKey()
	r := NewSession(key, New("r"))
	var first, second []Op
	r.Do(func(d *Doc) {
		first = d.Insert(0, "A")  // counter 1, at the start
		second = d.Insert(0, "B") // counter 2, at the start too: needs nothing
	})
	p := NewSession(key, New("p"))
	p.Do(func(d *Doc) { d.Apply(second[0]) }) // only the later one came
	_ = first
	link(r, p, "r", "p")
	if text(p) != text(r) {
		t.Errorf("p has %q, r %q", text(p), text(r))
	}
}

// With Run, a message changes the document only when Run runs it.
func TestSessionRun(t *testing.T) {
	key, _ := NewKey()
	a := NewSession(key, New("a"))
	b := NewSession(key, New("b"))
	var queue []func()
	b.Run = func(f func()) { queue = append(queue, f) }
	link(a, b, "a", "b")
	var ops []Op
	a.Do(func(d *Doc) { ops = d.SetText("INT. BARN\n") })
	a.Local(ops)
	if text(b) != "" {
		t.Fatalf("b changed outside Run: %q", text(b))
	}
	for len(queue) > 0 {
		f := queue[0]
		queue = queue[1:]
		f()
	}
	if text(b) != "INT. BARN\n" {
		t.Errorf("b has %q", text(b))
	}
}

// A kept state taken up again (a reload) merges with the peers without
// doubling, under a new replica name.
func TestStateRestore(t *testing.T) {
	key, _ := NewKey()
	a, b := NewSession(key, New("a")), NewSession(key, New("b"))
	a.Do(func(d *Doc) { d.SetText("INT. BARN - DAY\n\nRain.\n") })
	link(a, b, "a", "b")
	b.SetMeta("language", "eo")
	kept, err := b.State()
	if err != nil {
		t.Fatal(err)
	}
	a.RemovePeer("b")
	var ops []Op
	a.Do(func(d *Doc) { ops = d.SetText("INT. BARN - NIGHT\n\nRain.\n") }) // while b was away
	a.Local(ops)
	b2, err := Restore(key, "b2", kept)
	if err != nil {
		t.Fatal(err)
	}
	if text(b2) != "INT. BARN - DAY\n\nRain.\n" || b2.Meta("language") != "eo" {
		t.Fatalf("restored %q, %q", text(b2), b2.Meta("language"))
	}
	b2.Do(func(d *Doc) { ops = d.SetText("INT. BARN - DAY\n\nRain falls.\n") })
	link(a, b2, "a", "b2")
	b2.Local(ops)
	want := "INT. BARN - NIGHT\n\nRain falls.\n"
	if text(a) != want || text(b2) != want {
		t.Errorf("a %q, b2 %q", text(a), text(b2))
	}
	if _, err := Restore(key, "x", []byte(`{"v":99}`)); err == nil {
		t.Error("a state of another version was read")
	}
}

// Keys as chats and mails pass them on.
func TestParseKeyMangled(t *testing.T) {
	k, _ := NewKey()
	s := k.String()
	for _, in := range []string{
		s,
		"https://lapingvino.github.io/accolade/#k=" + s,
		"https://lapingvino.github.io/accolade/#k=" + s + ".",
		"(https://example.org/accolade/#k=" + s + ")",
		"\"" + s + "\"",
		"  " + s[:20] + "\n" + s[20:] + "  ",
		"https://lapingvino.github.io/accolade/%23k=" + s,
		"<https://lapingvino.github.io/accolade/#k=" + s + ">",
	} {
		if got, err := ParseKey(in); err != nil || got != k {
			t.Errorf("%q: %v", in, err)
		}
	}
	for _, in := range []string{"", "hello", s[:30], "#k=" + s[:42] + "!"} {
		if _, err := ParseKey(in); err == nil {
			t.Errorf("%q read as a key", in)
		}
	}
}
