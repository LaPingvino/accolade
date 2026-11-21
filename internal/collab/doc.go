// Package collab is a text CRDT for editing a script together: an RGA
// (replicated growable array) of runes. Every rune has an ID (a Lamport
// counter and the replica that made it) and is inserted after the rune it
// was typed after; deleted runes stay as tombstones. Replicas that have
// applied the same operations, in any order and any number of times, have
// the same text. The same code runs in Accolade and its web version.
//
// Runes are kept in runs: text typed or pasted in one go has consecutive
// IDs, each rune after the one before, and is one node and one op however
// long it is. A node is split where someone types inside it or deletes part
// of it. A novel pasted in costs a few bytes a rune, not a node each.
package collab

import (
	"sort"
	"strings"
	"unicode/utf8"
)

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

// plus is the ID k runes on in the same replica's run.
func (a ID) plus(k int) ID { return ID{a.Counter + uint64(k), a.Replica} }

// Op is an edit of runes with consecutive IDs from ID on: inserting Text,
// its first rune after After (the zero ID: at the start) and each other
// after the one before it, or deleting Del runes (ID, ID+1, …). It is
// what goes over the wire and what is kept.
type Op struct {
	ID    ID     `json:"id"`
	After ID     `json:"a,omitempty"`
	Text  string `json:"t,omitempty"`
	Del   int    `json:"d,omitempty"`
}

// node is runes with consecutive IDs from id on, each after the one
// before, all deleted or none.
type node struct {
	id      ID
	text    []rune
	deleted bool
	next    *node
}

// entry is a kept op (Ops, Missing): text as bytes, so a run typed key by
// key grows in place.
type entry struct {
	id, after ID
	text      []byte
	n, del    int // runes inserted; or deleted
}

func (e *entry) op() Op {
	if e.del > 0 {
		return Op{ID: e.id, Del: e.del}
	}
	return Op{ID: e.id, After: e.after, Text: string(e.text)}
}

// Doc is a replica of a text.
type Doc struct {
	replica string
	clock   uint64
	head    node               // before the first rune
	index   map[string][]*node // each replica's nodes, by counter
	size    int                // runes not deleted
	log     []entry            // every op applied, for peers that lack them
	pending []Op               // ops waiting for the runes they refer to
	version map[string]uint64
	counts  map[string]int // inserts applied, by replica (Missing)
	// text is Text, while textOK: SetText knows the text it made, and the
	// next key is compared with it without the text made again
	text   string
	textOK bool
}

// New is an empty replica named replica (unique among the peers).
func New(replica string) *Doc {
	return &Doc{replica: replica, index: map[string][]*node{}, version: map[string]uint64{}, counts: map[string]int{}}
}

// Replica is the replica's name.
func (d *Doc) Replica() string { return d.replica }

// Empty reports whether the Doc has had no op at all (not even one
// deleted since): a Doc named by a peer whose ops have not come yet.
func (d *Doc) Empty() bool { return len(d.log) == 0 && len(d.pending) == 0 }

// Len is the text's length in runes.
func (d *Doc) Len() int { return d.size }

// Text is the document's text.
func (d *Doc) Text() string {
	if d.textOK {
		return d.text
	}
	var b strings.Builder
	b.Grow(d.size)
	for n := d.head.next; n != nil; n = n.next {
		if !n.deleted {
			for _, r := range n.text {
				b.WriteRune(r)
			}
		}
	}
	d.text, d.textOK = b.String(), true
	return d.text
}

// runeAt is the node and offset in it of the visible rune at pos (nil
// past the end).
func (d *Doc) runeAt(pos int) (*node, int) {
	for n := d.head.next; n != nil; n = n.next {
		if n.deleted {
			continue
		}
		if pos < len(n.text) {
			return n, pos
		}
		pos -= len(n.text)
	}
	return nil, 0
}

// Writer is the replica that wrote the rune at rune offset pos ("" past
// the end): who made a line, for the rule that only its writer makes the
// automatic writes that follow from it (docs/COLLAB.md).
func (d *Doc) Writer(pos int) string {
	if n, _ := d.runeAt(pos); n != nil {
		return n.id.Replica
	}
	return ""
}

// find is the node holding the rune id and its offset there (nil: not
// inserted here, yet).
func (d *Doc) find(id ID) (*node, int) {
	idx := d.index[id.Replica]
	i := sort.Search(len(idx), func(i int) bool { return idx[i].id.Counter > id.Counter }) - 1
	if i < 0 {
		return nil, 0
	}
	n := idx[i]
	if k := int(id.Counter - n.id.Counter); k < len(n.text) {
		return n, k
	}
	return nil, 0
}

// absent is how many of the count IDs from id on are not inserted here,
// from id (which is not).
func (d *Doc) absent(id ID, count int) int {
	idx := d.index[id.Replica]
	i := sort.Search(len(idx), func(i int) bool { return idx[i].id.Counter > id.Counter })
	if i < len(idx) {
		return min(count, int(idx[i].id.Counter-id.Counter))
	}
	return count
}

func (d *Doc) addIndex(n *node) {
	idx := d.index[n.id.Replica]
	if k := len(idx); k == 0 || idx[k-1].id.Counter < n.id.Counter {
		d.index[n.id.Replica] = append(idx, n)
		return
	}
	i := sort.Search(len(idx), func(i int) bool { return idx[i].id.Counter > n.id.Counter })
	idx = append(idx, nil)
	copy(idx[i+1:], idx[i:])
	idx[i] = n
	d.index[n.id.Replica] = idx
}

// split cuts n before its k-th rune (0 < k < len) and returns the rest.
func (d *Doc) split(n *node, k int) *node {
	m := &node{id: n.id.plus(k), text: n.text[k:], deleted: n.deleted, next: n.next}
	n.text = n.text[:k:k] // n growing (a run typed on) must not write over m
	n.next = m
	d.addIndex(m)
	return m
}

// Insert inserts s at rune offset pos and returns the ops to send.
func (d *Doc) Insert(pos int, s string) []Op {
	if s == "" {
		return nil
	}
	var after ID
	if pos = min(pos, d.size); pos > 0 {
		n, k := d.runeAt(pos - 1)
		after = n.id.plus(k)
	}
	// a copy: s is often a piece of the editor's whole text
	op := Op{ID: ID{d.clock + 1, d.replica}, After: after, Text: strings.Clone(s)}
	d.Apply(op)
	return []Op{op}
}

// Delete deletes n runes at rune offset pos and returns the ops to send.
func (d *Doc) Delete(pos, n int) []Op {
	var ops []Op
	x, k := d.runeAt(pos)
	for ; x != nil && n > 0; x, k = x.next, 0 {
		if x.deleted {
			continue
		}
		m := min(n, len(x.text)-k)
		id := x.id.plus(k)
		if last := len(ops) - 1; last >= 0 && ops[last].ID.plus(ops[last].Del) == id {
			ops[last].Del += m
		} else {
			ops = append(ops, Op{ID: id, Del: m})
		}
		n -= m
	}
	for _, op := range ops {
		d.Apply(op)
	}
	return ops
}

// Apply applies an op from any replica; again, in part again, or out of
// order is fine. It returns the part of it that was new here (applied, or
// waiting for the runes it refers to): what is passed on to other peers.
func (d *Doc) Apply(op Op) []Op {
	var fresh []Op
	inserted := false
	if op.Del > 0 {
		fresh = d.del(op)
	} else if op.Text != "" {
		fresh, inserted = d.insert(op)
	}
	if inserted && len(d.pending) > 0 {
		d.retry()
	}
	return fresh
}

// retry applies the ops that waited, as far as they can be now.
func (d *Doc) retry() {
	for progress := true; progress; {
		progress = false
		waiting := d.pending
		d.pending = nil
		for _, p := range waiting {
			ready := false
			if p.Del > 0 { // any of its runes here now
				ready = d.has(p.ID) || d.absent(p.ID, p.Del) < p.Del
			} else {
				ready = p.After == (ID{}) || d.has(p.After) || d.has(p.ID)
			}
			if !ready {
				d.pending = append(d.pending, p)
				continue
			}
			if p.Del > 0 {
				d.del(p)
			} else if _, ok := d.insert(p); ok {
				progress = true
			}
		}
	}
}

func (d *Doc) has(id ID) bool {
	n, _ := d.find(id)
	return n != nil
}

// insert applies an insert op: the part not here yet (its runes here are
// always its first ones: each rune waits for the one before it). It
// reports whether runes went in.
func (d *Doc) insert(op Op) ([]Op, bool) {
	rs := []rune(op.Text)
	p := 0
	for p < len(rs) {
		n, k := d.find(op.ID.plus(p))
		if n == nil {
			break
		}
		p += len(n.text) - k
	}
	if p >= len(rs) {
		return nil, false
	}
	if p > 0 {
		op = Op{ID: op.ID.plus(p), After: op.ID.plus(p - 1), Text: string(rs[p:])}
		rs = rs[p:]
	}
	if op.After != (ID{}) && !d.has(op.After) {
		d.pending = append(d.pending, op)
		return []Op{op}, false
	}
	d.integrate(op.ID, op.After, rs)
	return []Op{op}, true
}

// integrate puts runes rs with IDs from id on after the rune after.
func (d *Doc) integrate(id ID, after ID, rs []rune) {
	prev := &d.head
	if after != (ID{}) {
		n, k := d.find(after)
		prev = n
		// after a rune inside a run: the rest of the run goes after the new
		// runes, unless it is greater (and so skipped, below, with it)
		if k < len(n.text)-1 && !id.less(n.id.plus(k+1)) {
			d.split(n, k+1)
		}
	}
	// RGA: after its origin, past the runes with greater IDs (inserted
	// there concurrently, or after those); a node's runes have greater IDs
	// than its first, so a node is passed whole
	for prev.next != nil && id.less(prev.next.id) {
		prev = prev.next
	}
	if prev != &d.head && !prev.deleted && prev.id.Replica == id.Replica && after == prev.id.plus(len(prev.text)-1) && prev.id.plus(len(prev.text)) == id {
		prev.text = append(prev.text, rs...) // typed on: the run grows
	} else {
		n := &node{id: id, text: rs, next: prev.next}
		prev.next = n
		d.addIndex(n)
	}
	last := id.plus(len(rs) - 1)
	d.size += len(rs)
	d.textOK = false
	d.clock = max(d.clock, last.Counter)
	d.version[id.Replica] = max(d.version[id.Replica], last.Counter)
	d.counts[id.Replica] += len(rs)
	if k := len(d.log) - 1; k >= 0 {
		e := &d.log[k]
		if e.del == 0 && e.id.Replica == id.Replica && e.id.plus(e.n) == id && after == e.id.plus(e.n-1) {
			e.text = utf8.AppendRune(e.text, rs[0])
			for _, r := range rs[1:] {
				e.text = utf8.AppendRune(e.text, r)
			}
			e.n += len(rs)
			return
		}
	}
	var b []byte
	for _, r := range rs {
		b = utf8.AppendRune(b, r)
	}
	d.log = append(d.log, entry{id: id, after: after, text: b, n: len(rs)})
}

// del applies a delete op: its runes here are deleted, those not here yet
// wait for them.
func (d *Doc) del(op Op) []Op {
	var fresh []Op
	add := func(o Op) {
		if k := len(fresh) - 1; k >= 0 && fresh[k].ID.plus(fresh[k].Del) == o.ID {
			fresh[k].Del += o.Del
			return
		}
		fresh = append(fresh, o)
	}
	id, left := op.ID, op.Del
	for left > 0 {
		n, k := d.find(id)
		if n == nil {
			m := d.absent(id, left)
			d.pending = append(d.pending, Op{ID: id, Del: m})
			add(Op{ID: id, Del: m})
			id, left = id.plus(m), left-m
			continue
		}
		m := min(left, len(n.text)-k)
		if !n.deleted {
			if k > 0 {
				n = d.split(n, k)
			}
			if m < len(n.text) {
				d.split(n, m)
			}
			n.deleted = true
			d.size -= m
			d.textOK = false
			add(Op{ID: id, Del: m})
			if j := len(d.log) - 1; j >= 0 && d.log[j].del > 0 && d.log[j].id.plus(d.log[j].del) == id {
				d.log[j].del += m
			} else {
				d.log = append(d.log, entry{id: id, del: m})
			}
		}
		id, left = id.plus(m), left-m
	}
	return fresh
}

// Ops are every op applied, in an order any replica can apply them in.
func (d *Doc) Ops() []Op {
	out := make([]Op, len(d.log))
	for i := range d.log {
		out[i] = d.log[i].op()
	}
	return out
}

// Version is how far each replica's inserts are known: a peer sends the
// ops it has beyond it (Missing).
func (d *Doc) Version() map[string]uint64 {
	v := make(map[string]uint64, len(d.version))
	for k, c := range d.version {
		v[k] = c
	}
	return v
}

// Counts are how many inserts of each replica were applied: with
// Version, what a peer sends (Missing).
func (d *Doc) Counts() map[string]int {
	c := make(map[string]int, len(d.counts))
	for k, n := range d.counts {
		c[k] = n
	}
	return c
}

// Missing are the ops a peer at version (with counts, nil if not known)
// lacks: inserts beyond its version, and every delete (cheap, and applying
// one twice does nothing). Ops are applied out of a replica's order when
// they wait for the rune they go after, so a peer can have a replica's op
// 52 and not its op 40. Where it has fewer of a replica's inserts up to
// its version than we do, it gets all of that replica's inserts (applying
// one again does nothing).
func (d *Doc) Missing(version map[string]uint64, counts map[string]int) []Op {
	gap := map[string]bool{}
	if counts != nil {
		upTo := map[string]int{} // our inserts of each replica within its version
		for i := range d.log {
			e := &d.log[i]
			if v := version[e.id.Replica]; e.del == 0 && v >= e.id.Counter {
				upTo[e.id.Replica] += min(e.n, int(v-e.id.Counter+1))
			}
		}
		for r, n := range upTo {
			gap[r] = n > counts[r]
		}
	}
	var out []Op
	for i := range d.log {
		e := &d.log[i]
		v := version[e.id.Replica]
		switch {
		case e.del > 0 || gap[e.id.Replica] || v < e.id.Counter:
			out = append(out, e.op())
		case v < e.id.Counter+uint64(e.n)-1: // the run's runes after v
			j := int(v - e.id.Counter + 1)
			rest := e.text
			for range j {
				_, size := utf8.DecodeRune(rest)
				rest = rest[size:]
			}
			out = append(out, Op{ID: e.id.plus(j), After: e.id.plus(j - 1), Text: string(rest)})
		}
	}
	return out
}
