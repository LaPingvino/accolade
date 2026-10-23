package collab

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestLocalEditsLandWhereTyped(t *testing.T) {
	d := New("a")
	d.Insert(0, "INT. BARN")
	d.Insert(4, " DAY")
	d.Delete(4, 4)
	d.Insert(9, " - DAY")
	if got := d.Text(); got != "INT. BARN - DAY" {
		t.Errorf("%q", got)
	}
}

// Concurrent inserts at one place keep each replica's run together, in
// the same order everywhere.
func TestConcurrentInsertsAtOnePlace(t *testing.T) {
	a, b := New("a"), New("b")
	base := a.Insert(0, "AB")
	for _, op := range base {
		b.Apply(op)
	}
	oa := a.Insert(1, "xx")
	ob := b.Insert(1, "yy")
	for _, op := range oa {
		b.Apply(op)
	}
	for _, op := range ob {
		a.Apply(op)
	}
	if a.Text() != b.Text() || (a.Text() != "AxxyyB" && a.Text() != "AyyxxB") {
		t.Errorf("a %q, b %q", a.Text(), b.Text())
	}
}

// Replicas editing at random, receiving each other's ops shuffled,
// repeated and out of order, end with the same text.
func TestConvergence(t *testing.T) {
	for seed := int64(1); seed <= 200; seed++ {
		rng := rand.New(rand.NewSource(seed))
		docs := make([]*Doc, 2+rng.Intn(3))
		for i := range docs {
			docs[i] = New(fmt.Sprintf("r%d", i))
		}
		inbox := make([][]Op, len(docs))
		send := func(from int, ops []Op) {
			for to := range docs {
				if to != from {
					inbox[to] = append(inbox[to], ops...)
				}
			}
		}
		deliver := func(to, n int) {
			rng.Shuffle(len(inbox[to]), func(i, j int) { inbox[to][i], inbox[to][j] = inbox[to][j], inbox[to][i] })
			for i := 0; i < n && len(inbox[to]) > 0; i++ {
				op := inbox[to][0]
				if rng.Intn(5) > 0 { // sometimes delivered twice
					inbox[to] = inbox[to][1:]
				}
				docs[to].Apply(op)
			}
		}
		for step := 0; step < 60; step++ {
			i := rng.Intn(len(docs))
			d := docs[i]
			n := len([]rune(d.Text()))
			switch {
			case n > 0 && rng.Intn(3) == 0:
				p := rng.Intn(n)
				send(i, d.Delete(p, 1+rng.Intn(min(3, n-p))))
			default:
				send(i, d.Insert(rng.Intn(n+1), strings.Repeat(string(rune('a'+i)), 1+rng.Intn(3))))
			}
			deliver(rng.Intn(len(docs)), rng.Intn(6))
		}
		for to := range docs { // everything arrives in the end
			for len(inbox[to]) > 0 {
				deliver(to, len(inbox[to]))
			}
		}
		for i := 1; i < len(docs); i++ {
			if docs[i].Text() != docs[0].Text() {
				t.Fatalf("seed %d: replica %d has %q, replica 0 %q", seed, i, docs[i].Text(), docs[0].Text())
			}
		}
	}
}

// A peer catches up from another's version.
func TestMissing(t *testing.T) {
	a, b := New("a"), New("b")
	a.Insert(0, "Rain falls.")
	for _, op := range a.Missing(b.Version()) {
		b.Apply(op)
	}
	a.Delete(0, 4)
	a.Insert(0, "Snow")
	for _, op := range a.Missing(b.Version()) {
		b.Apply(op)
	}
	if b.Text() != "Snow falls." {
		t.Errorf("%q", b.Text())
	}
}

// Typing a long script stays fast.
func TestLongScript(t *testing.T) {
	d := New("a")
	start := time.Now()
	line := "Rain hammers the roof of the barn.\n"
	for i := 0; i < 2000; i++ {
		d.Insert(len([]rune(d.Text())), line)
	}
	if took := time.Since(start); took > 10*time.Second {
		t.Errorf("%d runes in %v", len(d.Text()), took)
	}
	other := New("b")
	start = time.Now()
	for _, op := range d.Ops() {
		other.Apply(op)
	}
	if other.Text() != d.Text() || time.Since(start) > 2*time.Second {
		t.Errorf("replaying %d ops took %v", len(d.Ops()), time.Since(start))
	}
}
