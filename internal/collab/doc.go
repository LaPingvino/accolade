// Package collab is a text CRDT for editing a script together: an RGA
// (replicated growable array) of runes. Every rune has an ID (a Lamport
// counter and the replica that made it) and is inserted after the rune it
// was typed after; deleted runes stay as tombstones. Replicas that have
// applied the same operations, in any order and any number of times, have
// the same text. The same code runs in Accolade and its web version.
package collab

import "strings"

// ID names a rune: the replica that made it and its Lamport counter.
type ID struct {
	Counter uint64 `json:"c"`
	Replica string `json:"r"`
}

// less orders IDs: by counter, then replica.
func (a ID) less(b ID) bool {
	if a.Counter != b.Counter {
		return a.Counter < b.Counter
	}
	return a.Replica < b.Replica
}

// Op is an edit: inserting Rune with ID after After (the zero ID: at the
// start), or deleting the rune ID (Delete).
type Op struct {
	ID     ID   `json:"id"`
	After  ID   `json:"after,omitempty"`
	Rune   rune `json:"rune,omitempty"`
	Delete bool `json:"del,omitempty"`
}

type node struct {
	id      ID
	r       rune
	deleted bool
	next    *node
}

// Doc is a replica of a text.
type Doc struct {
	replica string
	clock   uint64
	head    node         // before the first rune
	nodes   map[ID]*node // every rune, by ID
	ops     []Op         // every op applied, for peers that lack them
	dels    map[ID]bool  // deletes applied
	pending []Op         // ops waiting for the rune they refer to
	version map[string]uint64
}

// New is an empty replica named replica (unique among the peers).
func New(replica string) *Doc {
	return &Doc{replica: replica, nodes: map[ID]*node{}, dels: map[ID]bool{}, version: map[string]uint64{}}
}

// Replica is the replica's name.
func (d *Doc) Replica() string { return d.replica }

// Text is the document's text.
func (d *Doc) Text() string {
	var b strings.Builder
	for n := d.head.next; n != nil; n = n.next {
		if !n.deleted {
			b.WriteRune(n.r)
		}
	}
	return b.String()
}

// visible is the node before the pos-th visible rune: inserting at pos
// goes after it (the head for 0).
func (d *Doc) visible(pos int) *node {
	before := &d.head
	for n := d.head.next; n != nil && pos > 0; n = n.next {
		if !n.deleted {
			pos--
			before = n
		}
	}
	return before
}

// Insert inserts s at rune offset pos and returns the ops to send.
func (d *Doc) Insert(pos int, s string) []Op {
	after := d.visible(pos).id
	var ops []Op
	for _, r := range s {
		d.clock++
		op := Op{ID: ID{d.clock, d.replica}, After: after, Rune: r}
		d.Apply(op)
		ops = append(ops, op)
		after = op.ID
	}
	return ops
}

// Delete deletes n runes at rune offset pos and returns the ops to send.
func (d *Doc) Delete(pos, n int) []Op {
	var ops []Op
	for x := d.visible(pos).next; x != nil && len(ops) < n; x = x.next {
		if !x.deleted {
			ops = append(ops, Op{ID: x.id, Delete: true})
		}
	}
	for _, op := range ops {
		d.Apply(op)
	}
	return ops
}

// Apply applies an op from any replica; again or out of order is fine.
// It reports whether the op was new.
func (d *Doc) Apply(op Op) bool {
	if op.Delete && d.dels[op.ID] || !op.Delete && d.nodes[op.ID] != nil {
		return false
	}
	if !d.ready(op) {
		d.pending = append(d.pending, op)
		return true
	}
	d.apply(op)
	for progress := true; progress; { // ops that waited for this one
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
	return true
}

// ready: the rune the op refers to is there.
func (d *Doc) ready(op Op) bool {
	ref := op.After
	if op.Delete {
		ref = op.ID
	}
	return ref == (ID{}) || d.nodes[ref] != nil
}

func (d *Doc) apply(op Op) {
	d.ops = append(d.ops, op)
	if op.ID.Counter > d.clock {
		d.clock = op.ID.Counter
	}
	if op.Delete {
		d.dels[op.ID] = true
		d.nodes[op.ID].deleted = true
		return
	}
	if op.ID.Counter > d.version[op.ID.Replica] {
		d.version[op.ID.Replica] = op.ID.Counter
	}
	prev := &d.head
	if op.After != (ID{}) {
		prev = d.nodes[op.After]
	}
	// RGA: after its origin, past the runes with greater IDs (inserted
	// there concurrently, or after those)
	for prev.next != nil && op.ID.less(prev.next.id) {
		prev = prev.next
	}
	n := &node{id: op.ID, r: op.Rune, next: prev.next}
	prev.next = n
	d.nodes[op.ID] = n
}

// Ops are every op applied, in an order any replica can apply them in.
func (d *Doc) Ops() []Op { return append([]Op(nil), d.ops...) }

// Version is how far each replica's inserts are known: a peer sends the
// ops it has beyond it (Missing).
func (d *Doc) Version() map[string]uint64 {
	v := make(map[string]uint64, len(d.version))
	for k, c := range d.version {
		v[k] = c
	}
	return v
}

// Missing are the ops a peer at version lacks: inserts beyond it, and
// every delete (cheap, and applying one twice does nothing).
func (d *Doc) Missing(version map[string]uint64) []Op {
	var out []Op
	for _, op := range d.ops {
		if op.Delete || op.ID.Counter > version[op.ID.Replica] {
			out = append(out, op)
		}
	}
	return out
}
