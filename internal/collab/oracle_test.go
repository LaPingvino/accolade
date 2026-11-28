// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package collab

// The Doc before runs (a node, a map entry and an op per rune), kept as
// the oracle: a Doc of runs given the same ops, in any cut, has the same
// text, version, counts and missing runes.

import (
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"testing"
)

type oldOp struct {
	ID     ID
	After  ID
	Rune   rune
	Delete bool
}

type oldNode struct {
	id      ID
	r       rune
	deleted bool
	next    *oldNode
}

type oldDoc struct {
	head    oldNode
	nodes   map[ID]*oldNode
	ops     []oldOp
	dels    map[ID]bool
	pending []oldOp
	version map[string]uint64
	counts  map[string]int
}

func newOld() *oldDoc {
	return &oldDoc{nodes: map[ID]*oldNode{}, dels: map[ID]bool{}, version: map[string]uint64{}, counts: map[string]int{}}
}

func (d *oldDoc) Text() string {
	var b strings.Builder
	for n := d.head.next; n != nil; n = n.next {
		if !n.deleted {
			b.WriteRune(n.r)
		}
	}
	return b.String()
}

func (d *oldDoc) Apply(op oldOp) {
	if op.Delete && d.dels[op.ID] || !op.Delete && d.nodes[op.ID] != nil {
		return
	}
	if !d.ready(op) {
		d.pending = append(d.pending, op)
		return
	}
	d.apply(op)
	for progress := true; progress; {
		progress = false
		for i := 0; i < len(d.pending); i++ {
			if p := d.pending[i]; d.ready(p) {
				d.pending = append(d.pending[:i], d.pending[i+1:]...)
				if p.Delete && !d.dels[p.ID] || !p.Delete && d.nodes[p.ID] == nil {
					d.apply(p)
				}
				progress = true
				i--
			}
		}
	}
}

func (d *oldDoc) ready(op oldOp) bool {
	ref := op.After
	if op.Delete {
		ref = op.ID
	}
	return ref == (ID{}) || d.nodes[ref] != nil
}

func (d *oldDoc) apply(op oldOp) {
	d.ops = append(d.ops, op)
	if op.Delete {
		d.dels[op.ID] = true
		d.nodes[op.ID].deleted = true
		return
	}
	if op.ID.Counter > d.version[op.ID.Replica] {
		d.version[op.ID.Replica] = op.ID.Counter
	}
	d.counts[op.ID.Replica]++
	prev := &d.head
	if op.After != (ID{}) {
		prev = d.nodes[op.After]
	}
	for prev.next != nil && op.ID.less(prev.next.id) {
		prev = prev.next
	}
	n := &oldNode{id: op.ID, r: op.Rune, next: prev.next}
	prev.next = n
	d.nodes[op.ID] = n
}

func (d *oldDoc) Missing(version map[string]uint64, counts map[string]int) []oldOp {
	gap := map[string]bool{}
	upTo := map[string]int{}
	for _, op := range d.ops {
		if !op.Delete && op.ID.Counter <= version[op.ID.Replica] {
			upTo[op.ID.Replica]++
		}
	}
	for r, n := range upTo {
		gap[r] = n > counts[r]
	}
	var out []oldOp
	for _, op := range d.ops {
		if op.Delete || op.ID.Counter > version[op.ID.Replica] || gap[op.ID.Replica] {
			out = append(out, op)
		}
	}
	return out
}

// runes are an op's ops, a rune each.
func runes(op Op) []oldOp {
	var out []oldOp
	for i := 0; i < op.Del; i++ {
		out = append(out, oldOp{ID: op.ID.plus(i), Delete: true})
	}
	after := op.After
	for i, r := range []rune(op.Text) {
		out = append(out, oldOp{ID: op.ID.plus(i), After: after, Rune: r})
		after = op.ID.plus(i)
	}
	return out
}

// cut is op in pieces (as another peer may have sent it).
func cut(rng *rand.Rand, op Op) []Op {
	k := op.runeLen()
	if k < 2 || rng.Intn(2) == 0 {
		return []Op{op}
	}
	at := 1 + rng.Intn(k-1)
	if op.Del > 0 {
		return []Op{{ID: op.ID, Del: at}, {ID: op.ID.plus(at), Del: k - at}}
	}
	rs := []rune(op.Text)
	return []Op{{ID: op.ID, After: op.After, Text: string(rs[:at])}, {ID: op.ID.plus(at), After: op.ID.plus(at - 1), Text: string(rs[at:])}}
}

func missingSet(ops []oldOp) []string {
	var out []string
	for _, o := range ops {
		out = append(out, fmt.Sprint(o))
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// Replicas of runs and their oracles get the same ops, cut, shuffled,
// repeated and out of order: each pair agrees at every step, and all
// end alike.
func TestRunsAgreeWithRunes(t *testing.T) {
	for seed := int64(1); seed <= 300; seed++ {
		rng := rand.New(rand.NewSource(seed))
		k := 2 + rng.Intn(3)
		docs, olds := make([]*Doc, k), make([]*oldDoc, k)
		for i := range docs {
			docs[i], olds[i] = New(fmt.Sprintf("r%d", i)), newOld()
		}
		inbox := make([][]Op, k)
		send := func(from int, ops []Op) {
			for _, op := range ops {
				for _, o := range runes(op) { // the maker's oracle
					olds[from].Apply(o)
				}
			}
			for to := range docs {
				if to != from {
					for _, op := range ops {
						inbox[to] = append(inbox[to], cut(rng, op)...)
					}
				}
			}
		}
		check := func(i int, when string) {
			if a, b := docs[i].Text(), olds[i].Text(); a != b {
				t.Fatalf("seed %d %s: replica %d has %q, its oracle %q", seed, when, i, a, b)
			}
			if !mapsU(docs[i].Version(), olds[i].version) || !mapsInt(docs[i].Counts(), olds[i].counts) {
				t.Fatalf("seed %d %s: replica %d version %v counts %v, oracle %v %v", seed, when, i, docs[i].Version(), docs[i].Counts(), olds[i].version, olds[i].counts)
			}
			if docs[i].Len() != len([]rune(olds[i].Text())) {
				t.Fatalf("seed %d: Len %d", seed, docs[i].Len())
			}
		}
		deliver := func(to, n int) {
			rng.Shuffle(len(inbox[to]), func(i, j int) { inbox[to][i], inbox[to][j] = inbox[to][j], inbox[to][i] })
			for i := 0; i < n && len(inbox[to]) > 0; i++ {
				op := inbox[to][0]
				if rng.Intn(5) > 0 {
					inbox[to] = inbox[to][1:]
				}
				before := docs[to].Text()
				docs[to].Apply(op)
				for _, o := range runes(op) {
					olds[to].Apply(o)
				}
				if docs[to].Text() != olds[to].Text() {
					t.Logf("before %q op %+v pending %+v", before, op, docs[to].pending)
				}
				check(to, "delivering")
			}
		}
		for step := 0; step < 80; step++ {
			i := rng.Intn(len(docs))
			d := docs[i]
			n := d.Len()
			switch {
			case n > 0 && rng.Intn(3) == 0:
				p := rng.Intn(n)
				send(i, d.Delete(p, 1+rng.Intn(min(8, n-p))))
			case rng.Intn(4) == 0: // a key at a time
				p := rng.Intn(n + 1)
				for j := 0; j < 1+rng.Intn(4); j++ {
					send(i, d.Insert(p+j, string(rune('a'+i))))
				}
			default:
				send(i, d.Insert(rng.Intn(n+1), strings.Repeat(string(rune('a'+i)), 1+rng.Intn(6))+"é"))
			}
			check(i, "editing")
			// a peer catching up from another's version
			if j := rng.Intn(len(docs)); j != i && rng.Intn(4) == 0 {
				v, c := docs[j].Version(), docs[j].Counts()
				got, want := docs[i].Missing(v, c), olds[i].Missing(olds[j].version, olds[j].counts)
				var gotRunes []oldOp
				for _, op := range got {
					gotRunes = append(gotRunes, runes(op)...)
				}
				if !slices.Equal(missingSet(gotRunes), missingSet(want)) {
					t.Fatalf("seed %d: Missing differs:\n%v\n%v", seed, missingSet(gotRunes), missingSet(want))
				}
			}
			deliver(rng.Intn(len(docs)), rng.Intn(6))
		}
		for to := range docs {
			for len(inbox[to]) > 0 {
				deliver(to, len(inbox[to]))
			}
		}
		for i := range docs {
			if docs[i].Text() != docs[0].Text() {
				t.Fatalf("seed %d: replica %d has %q, replica 0 %q", seed, i, docs[i].Text(), docs[0].Text())
			}
			// a replica rebuilt from the kept ops is the same
			r := New("x")
			for _, op := range docs[i].Ops() {
				r.Apply(op)
			}
			if r.Text() != docs[i].Text() {
				t.Fatalf("seed %d: replayed %q, want %q", seed, r.Text(), docs[i].Text())
			}
		}
	}
}

func mapsU(a, b map[string]uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func mapsInt(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
