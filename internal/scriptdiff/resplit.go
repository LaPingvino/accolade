package scriptdiff

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Re-splitting (MergeOptions.Resplit). A source read from a print joins
// what the script has as several elements into one (a cue, its speech and
// the action after it, with no blank line between), or the other way
// round: the same words, cut elsewhere. Element against element, every
// such cut is a difference, and the pieces do not sit side by side. So
// when a scene's two sides have different numbers of elements, theirs is
// first cut at ours' breaks: the scene's words are compared as one run
// (spaces and line breaks aside), and each of ours' elements gets the
// words of theirs that match its own, with theirs' words between them in
// the element they came with. What then differs is words, each in the
// one element of ours it belongs to (written in ours' lines, a speech
// kept a speech), so taking it never writes a word twice. A whole
// element of theirs that shares little with ours stays one of its own,
// and so does a heading of theirs that ours has inside another element
// (a conversion that ran a heading into the action after it: taking it
// puts the heading back). Such a heading starts a scene only where ours
// has a scene start too (a scene the script lacks).

// rtok is a word or punctuation mark of an element, as compared.
type rtok struct {
	key      string // as compared (the element's key's token)
	el       int    // its element
	from, to int    // its bytes in the element's Text (from < 0: the key's own)
}

// rside are a side's tokens, and each element's text tokens with the
// token (index in the tokens) each one is (-1: not compared: spaces, or
// what the key leaves out).
func rside(es []melem) (ts []rtok, text [][]string, of [][]int) {
	text, of = make([][]string, len(es)), make([][]int, len(es))
	for i, e := range es {
		if e.key == "" {
			continue
		}
		var kt []string
		for _, x := range tokens(e.key) {
			if strings.TrimSpace(x) != "" && !rMark[x] {
				kt = append(kt, x)
			}
		}
		var tt []string
		var at []int // tt's index in text[i]
		text[i] = tokens(e.Text)
		of[i] = make([]int, len(text[i]))
		var spans [][2]int
		pos := 0
		for x, s := range text[i] {
			of[i][x] = -1
			if strings.TrimSpace(s) != "" && !rMark[s] {
				tt, at = append(tt, strings.ToLower(s)), append(at, x)
				spans = append(spans, [2]int{pos, pos + len(s)})
			}
			pos += len(s)
		}
		mk := matches(kt, tt)
		for q, k := range kt {
			t := rtok{key: k, el: i, from: -1}
			if x := mk[q]; x >= 0 {
				t.from, t.to = spans[x][0], spans[x][1]
				of[i][at[x]] = len(ts)
			}
			ts = append(ts, t)
		}
	}
	return
}

var (
	// Fountain's centring marks say how a line is laid out, not what it
	// says; a print read back centres lines a conversion does not. Not
	// compared: ours' stay as they are. (A forced cue's or action's @ or !
	// the key leaves out already.)
	rMark    = map[string]bool{">": true, "<": true}
	rSpaces  = regexp.MustCompile(`[ \t]{2,}`)
	rBefore  = regexp.MustCompile(` +([,.;:!?)])`)
	rTrailSp = regexp.MustCompile(`(?m)[ \t]+$`)
)

// resplit is theirs' elements cut at ours' breaks (see above).
func resplit(o, t []melem) []melem {
	ot, otext, oof := rside(o)
	tt, _, _ := rside(t)
	if len(ot) == 0 || len(tt) == 0 {
		return t
	}
	ok, tk := make([]string, len(ot)), make([]string, len(tt))
	for i, x := range ot {
		ok[i] = x.key
	}
	for i, x := range tt {
		tk[i] = x.key
	}
	om := make([]int, len(ot)) // ours' token → theirs' (-1 none)
	tm := make([]int, len(tt))
	at := make([]int, len(tt)) // theirs' token: ours' next token there
	byEl := map[int][]int{}    // theirs' tokens by element
	for x, tok := range tt {
		byEl[tok.el] = append(byEl[tok.el], x)
	}
	own := map[int]bool{} // theirs' elements sharing little: their own
	align := func() {     // the words compared as one run, theirs' own left out
		var keys []string
		var idx []int
		for x, k := range tk {
			tm[x], at[x] = -1, -1
			if !own[tt[x].el] {
				keys, idx = append(keys, k), append(idx, x)
			}
		}
		i, j := 0, 0
		for _, op := range diff(ok, keys) {
			switch op {
			case Same:
				om[i], tm[idx[j]], at[idx[j]] = idx[j], i, i
				i, j = i+1, j+1
			case Removed:
				om[i] = -1
				i++
			case Added:
				at[idx[j]] = i
				j++
			}
		}
	}
	align()
	n := 0
	for _, m := range om {
		if m >= 0 {
			n++
		}
	}
	if 2*n < len(ot) { // too unlike (or past the diff's reach): element by element
		return t
	}
	for el, xs := range byEl {
		n := 0
		for _, x := range xs {
			if tm[x] >= 0 && (!t[el].Heading || o[ot[tm[x]].el].Heading) {
				n++
			}
		}
		if 2*n < len(xs) {
			own[el] = true
		}
	}
	if len(own) > 0 {
		align() // without them: what they matched by chance matched anew
	}
	// each of theirs' tokens to one of ours' elements: a matched one to
	// its match's, the others to their element's matched neighbour's
	// (the one before, else after)
	owner := make([]int, len(tt))
	for x := range tt {
		owner[x] = -1
		if tm[x] >= 0 {
			owner[x] = ot[tm[x]].el
		}
	}
	for _, xs := range byEl {
		if own[tt[xs[0]].el] {
			continue
		}
		last := -1
		for _, x := range xs {
			if owner[x] >= 0 {
				last = owner[x]
			} else if last >= 0 {
				owner[x] = last
			}
		}
		last = -1
		for k := len(xs) - 1; k >= 0; k-- {
			if x := xs[k]; owner[x] >= 0 {
				last = owner[x]
			} else {
				owner[x] = last
			}
		}
	}
	// the insertions into each of ours' elements, before which of ours'
	// tokens (at)
	ins := map[int][]int{}
	changed := map[int]bool{}
	for x := range tt {
		if tm[x] < 0 && owner[x] >= 0 {
			ins[owner[x]] = append(ins[owner[x]], x)
			changed[owner[x]] = true
		}
	}
	got := map[int]bool{} // ours' elements that have some of theirs
	for x := range tt {
		if owner[x] >= 0 {
			got[owner[x]] = true
		}
	}
	for q, m := range om {
		if m < 0 {
			changed[ot[q].el] = true
		}
	}
	tokText := func(x int) string {
		tok := tt[x]
		if tok.from < 0 {
			return tok.key
		}
		return t[tok.el].Text[tok.from:tok.to]
	}
	isWord := func(s string, last bool) bool {
		r, _ := utf8.DecodeRuneInString(s)
		if last {
			r, _ = utf8.DecodeLastRuneInString(s)
		}
		return unicode.IsLetter(r) || unicode.IsDigit(r)
	}
	rebuild := func(el int) string {
		var b strings.Builder
		pend := ins[el]
		put := func(s string) {
			if cur := b.String(); cur != "" && isWord(cur, true) && isWord(s, false) {
				b.WriteByte(' ')
			}
			b.WriteString(s)
		}
		flushTo := func(q int) { // theirs' insertions before ours' token q
			prev := -1
			for len(pend) > 0 && at[pend[0]] <= q {
				x := pend[0]
				pend = pend[1:]
				s := tokText(x)
				if prev >= 0 && tt[prev].el == tt[x].el && tt[prev].from >= 0 && tt[x].from >= 0 {
					s = t[tt[x].el].Text[tt[prev].to:tt[x].from] + s // theirs' spaces between them
					s = strings.ReplaceAll(s, "\n", " ")
					b.WriteString(s)
				} else {
					cur := b.String()
					if cur != "" && !strings.HasSuffix(cur, " ") && !strings.HasSuffix(cur, "\n") && !strings.ContainsAny(s[:1], ",.;:!?)") {
						b.WriteByte(' ')
					}
					b.WriteString(s)
				}
				prev = x
			}
		}
		lastGone := false // ours' last word here is not theirs
		for x, s := range otext[el] {
			q := oof[el][x]
			if q < 0 {
				if strings.Contains(s, "\n") { // a line break inside an edit goes with it ("mo-\nney")
					next := len(ot)
					for y := x + 1; y < len(otext[el]); y++ {
						if oof[el][y] >= 0 {
							next = oof[el][y]
							break
						}
					}
					nextGone, ins := next < len(ot) && om[next] < 0, len(pend) > 0 && at[pend[0]] <= next
					if lastGone && nextGone || ins && (lastGone || nextGone) { // the edit spans it
						s = " "
					}
				}
				b.WriteString(s)
				continue
			}
			lastGone = om[q] < 0
			flushTo(q)
			if om[q] >= 0 {
				put(s)
			}
		}
		flushTo(len(ot))
		out := rSpaces.ReplaceAllString(b.String(), " ")
		out = rBefore.ReplaceAllString(out, "$1")
		return strings.TrimSpace(rTrailSp.ReplaceAllString(out, ""))
	}
	// ours' breaks that are none of theirs: inside one line of theirs
	// (a speech the script cut in two with a blank line) is a difference,
	// theirs' element kept whole there
	glue := map[int]string{} // ours' element → what joins it to the next
	for q := 1; q < len(ot); q++ {
		a, b := ot[q-1], ot[q]
		if a.el == b.el || om[q-1] < 0 || om[q] < 0 || o[a.el].Heading || o[b.el].Heading {
			continue
		}
		if x, y := tt[om[q-1]], tt[om[q]]; x.el == y.el && x.from >= 0 && y.from >= 0 && x.to <= y.from {
			if sep := t[x.el].Text[x.to:y.from]; !strings.Contains(sep, "\n") {
				glue[a.el] = " "
				if strings.TrimSpace(sep) == "" && sep == "" {
					glue[a.el] = ""
				}
			}
		}
	}
	// theirs anew: theirs' own elements where they were, ours' with
	// theirs' words
	var out []melem
	ownDone := map[int]bool{}
	// a heading run into ours' action goes with the element it was run
	// into, one change (taken whole: the heading once, the action without
	// it); it starts no scene there, so that the scenes still pair
	var lead *melem
	flushLead := func() {
		if lead != nil {
			lead.inScene = true
			out = append(out, *lead)
			lead = nil
		}
	}
	emitOwn := func(upTo int) { // theirs' own elements before theirs' token upTo
		for x := 0; x < upTo; x++ {
			el := tt[x].el
			if !own[el] || ownDone[el] {
				continue
			}
			ownDone[el] = true
			e := t[el]
			q := 0 // where it goes in ours: after the last of theirs' that ours has
			for y := byEl[el][0] - 1; y >= 0; y-- {
				if tm[y] >= 0 {
					q = tm[y] + 1
					break
				}
			}
			flushLead()
			if e.Heading && q < len(ot) && (!o[ot[q].el].Heading || q > 0 && ot[q-1].el == ot[q].el) {
				lead = &e // inside one of ours' scenes: with the element after it
				continue
			}
			out = append(out, e)
		}
	}
	firstOf := map[int]int{} // ours' element → theirs' first token it has
	for x := len(tt) - 1; x >= 0; x-- {
		if owner[x] >= 0 {
			firstOf[owner[x]] = x
		}
	}
	joined := -1 // the element glued to the next, last emitted
	for el := range o {
		if o[el].key == "" || !got[el] {
			continue
		}
		e := o[el]
		if changed[el] || joined >= 0 {
			text := rebuild(el)
			e.Text, e.key = text, o[el].key+"\x00"+text
		}
		if joined >= 0 {
			p := &out[len(out)-1]
			p.Text += glue[joined] + e.Text
			p.key += "\x00" + e.key
		} else {
			emitOwn(firstOf[el])
			if _, ok := glue[el]; ok {
				e.Text, e.key = rebuild(el), o[el].key+"\x00"
			}
			if lead != nil {
				e.Text, e.key = lead.Text+"\n\n"+e.Text, o[el].key+"\x00"+lead.Text+"\n\n"+e.Text
				lead = nil
			}
			out = append(out, e)
		}
		joined = -1
		if _, ok := glue[el]; ok {
			joined = el
		}
	}
	emitOwn(len(tt))
	flushLead()
	return out
}
