package collab

import "unicode/utf8"

// On the wire ops go as runs: text typed or pasted in one go is runes
// with consecutive IDs, each after the one before, and deleting it
// deletes consecutive IDs. An Op is such a run; ops typed key by key are
// joined into one, and a long one is cut to fit a message.

// maxRun keeps a run's text well inside one message.
const maxRun = 2048

// runeLen is how many IDs an op takes.
func (o Op) runeLen() int {
	if o.Del > 0 {
		return o.Del
	}
	return utf8.RuneCountInString(o.Text)
}

// runsOf joins ops where one goes on from the one before, and cuts them
// to at most maxRun runes, in their order.
func runsOf(ops []Op) []Op {
	var out []Op
	n := 0 // runes in the last
	for _, op := range ops {
		k := op.runeLen()
		if k == 0 {
			continue
		}
		if last := len(out) - 1; last >= 0 && n+k <= maxRun {
			l := &out[last]
			if l.Del > 0 && op.Del > 0 && l.ID.plus(l.Del) == op.ID {
				l.Del += op.Del
				n += k
				continue
			}
			if l.Del == 0 && op.Del == 0 && l.ID.plus(n) == op.ID && op.After == l.ID.plus(n-1) {
				l.Text += op.Text
				n += k
				continue
			}
		}
		if k <= maxRun {
			out = append(out, op)
			n = k
			continue
		}
		if op.Del > 0 {
			for i := 0; i < op.Del; i += maxRun {
				out = append(out, Op{ID: op.ID.plus(i), Del: min(maxRun, op.Del-i)})
			}
		} else {
			rs := []rune(op.Text)
			for i := 0; i < len(rs); i += maxRun {
				after := op.After
				if i > 0 {
					after = op.ID.plus(i - 1)
				}
				out = append(out, Op{ID: op.ID.plus(i), After: after, Text: string(rs[i:min(len(rs), i+maxRun)])})
			}
		}
		n = out[len(out)-1].runeLen()
	}
	return out
}
