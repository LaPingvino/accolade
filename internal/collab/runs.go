package collab

// On the wire ops go as runs: text typed or pasted in one go is runes
// with consecutive IDs, each after the one before, and deleting it
// deletes consecutive IDs. A run is one entry for all of them.

// run is ops with consecutive IDs from ID on: inserts of Text, each rune
// after the one before (the first after After), or Del deletes.
type run struct {
	ID    ID     `json:"id"`
	After ID     `json:"a,omitempty"`
	Text  string `json:"t,omitempty"`
	Del   int    `json:"d,omitempty"`
}

// maxRun keeps a run's text well inside one message.
const maxRun = 2048

// runsOf packs ops into runs, in their order.
func runsOf(ops []Op) []run {
	var out []run
	var last Op
	n := 0 // runes (or deletes) in the run being made
	for i, op := range ops {
		next := ID{last.ID.Counter + 1, last.ID.Replica}
		if i > 0 && n < maxRun && op.ID == next && op.Delete == last.Delete && (op.Delete || op.After == last.ID) {
			r := &out[len(out)-1]
			if op.Delete {
				r.Del++
			} else {
				r.Text += string(op.Rune)
			}
			n++
		} else if op.Delete {
			out = append(out, run{ID: op.ID, Del: 1})
			n = 1
		} else {
			out = append(out, run{ID: op.ID, After: op.After, Text: string(op.Rune)})
			n = 1
		}
		last = op
	}
	return out
}

// ops are a run's ops.
func (r run) ops() []Op {
	var out []Op
	id := r.ID
	for i := 0; i < r.Del; i++ {
		out = append(out, Op{ID: id, Delete: true})
		id.Counter++
	}
	after := r.After
	for _, c := range r.Text {
		out = append(out, Op{ID: id, After: after, Rune: c})
		after = id
		id.Counter++
	}
	return out
}
