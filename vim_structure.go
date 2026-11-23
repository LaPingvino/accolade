package main

import (
	"strings"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/structure"
)

// The screenplay's structure in vim's and helix's keys (LOOP.md, the
// keyboard pass): scenes and sections as motions and text objects, read
// by the structure engine (internal/structure), as the window reads the
// script (its language's scene headings, its blocks at the end left out).
//
//	]] [[        the next / previous scene heading (a count: that many)
//	]s [s        the next / previous section heading
//	as is        a scene (with the blank lines after it) / its inside
//	aS iS        a section with all it holds / its inside
//	d c y        with an object: das deletes the scene, yis copies its inside
//	m            with an object: mas opens :'<,'>move … to move it by the engine
//	v            vas, viS: select it (line-wise)
//	p P          a scene or section in the register goes after or before
//	             the cursor's scene or section, never inside it
//
// helix: ]] [[ ]s [s select the next / previous unit; mas mis maS miS
// select one; then d c y p act on it, and : (with '<,'>) moves it.
//
// The text is read only when one of these keys comes, never per key.

// structureKey takes a key of the structure's (a pending ] or [, an
// object after i or a, helix's m); it reports whether it did.
func (v *vim) structureKey(r rune) bool {
	if v.g || v.z || v.replace {
		return false
	}
	switch {
	case v.bracket != 0:
		b := v.bracket
		v.bracket = 0
		v.bracketKey(b, r)
		return true
	case v.obj != 0:
		ia, f := v.obj, v.objFor
		v.obj = 0
		v.objectKey(ia, f, r)
		return true
	case v.match:
		v.match = false
		if r == 'i' || r == 'a' {
			v.obj, v.objFor = r, 'h'
		}
		return true
	}
	helix := v.helix != nil && v.helix()
	switch {
	case (r == ']' || r == '[') && v.op == 0:
		v.bracket = r
	case helix && r == 'm':
		v.match = true
	case !helix && v.op != 0 && (r == 'i' || r == 'a'):
		v.obj, v.objFor = r, v.op
		v.op = 0
	case !helix && v.visual() && (r == 'i' || r == 'a'):
		v.obj, v.objFor = r, 'v'
	default:
		return false
	}
	return true
}

// unitKind is the kind an object's or motion's letter names.
func unitKindOf(r rune) (structure.Kind, bool) {
	switch r {
	case 's':
		return structure.Scene, true
	case 'S':
		return structure.Section, true
	}
	return 0, false
}

// unitTree is the text read as its units.
func (v *vim) unitTree() *structure.Tree {
	if v.tree != nil {
		if t := v.tree(); t != nil {
			return t
		}
	}
	text := v.e.Text()
	s := structure.Plain(structure.Lines(text))
	if be := bodyEndByte(text); be < len(text) { // the blocks at the end are no scene's (as structReader)
		s.End = strings.Count(text[:be], "\n")
		if be > 0 && text[be-1] != '\n' {
			s.End++
		}
	}
	return structure.Parse(s)
}

// say shows s on the status line.
func (v *vim) say(s string) {
	if v.status != nil {
		v.status(s)
	}
}

// curLine is the line the keys act from: the visual cursor's in vim's
// visual mode, else the cursor's.
func (v *vim) curLine() int {
	at := v.e.CursorOffset()
	switch {
	case v.helix != nil && v.helix(): // the selection's start: its end may be past its last line
		at, _ = v.e.Buffer().Selection()
	case v.visual():
		at = v.vcur
	}
	l, _ := v.e.Buffer().Position(at)
	return l
}

// unitWord names a kind for a message.
func unitWord(k structure.Kind) string {
	if k == structure.Section {
		return "section"
	}
	return "scene"
}

// objectKey completes a text object: ia is i or a, f what it is for, r
// its letter.
func (v *vim) objectKey(ia, f, r rune) {
	n := v.times()
	k, ok := unitKindOf(r)
	if !ok {
		v.say("No text object " + string(ia) + string(r) + ": the screenplay's are as, is, aS and iS")
		return
	}
	tr := v.unitTree()
	first, last, ok := tr.SpanOf(v.curLine(), k, n)
	if !ok {
		v.say("Not in a " + unitWord(k))
		return
	}
	head := first
	if ia == 'a' {
		last = tr.Trail(last)
	} else {
		first++
		for first <= last && strings.TrimSpace(tr.S.Lines[first]) == "" {
			first++
		}
	}
	b := v.e.Buffer()
	t := []rune(v.e.Text())
	if first > last { // an empty inside
		if f == 'c' {
			v.edit(func(b *buffer.Buffer, _ []rune, _ int) {
				at := b.LineEnd(head)
				b.Replace(at, at, "\n\n")
				b.SetCursor(at+2, false)
			})
			v.setMode(vimInsert)
			return
		}
		v.say("This " + unitWord(k) + " has nothing inside")
		return
	}
	from, to := b.LineStart(first), b.LineEnd(last)
	switch f {
	case 'd', 'c', 'y':
		v.apply(f, t, from, to, true)
		if ia == 'a' && f != 'c' {
			v.unitKind, v.unitText = r, v.reg
		}
	case 'm':
		if v.exWith == nil {
			v.say("Moving needs the window's command line")
			return
		}
		v.setMode(vimNormal)
		v.exWith([2]int{first, last}, "move ")
	case 'v':
		v.anchor, v.vcur = from, b.LineStart(last)
		v.mode = vimVisualLine
		v.select_()
		v.say(v.mode.String())
	case 'h':
		end := min(to+1, len(t))
		v.e.Navigate(func(bf *buffer.Buffer) { bf.Select(from, end) })
		if ia == 'a' {
			v.helixUnit = [3]int{int(r), from, end}
		}
	}
}

// bracketKey completes ]] [[ ]s [s: b is the first key, r the second.
func (v *vim) bracketKey(b, r rune) {
	n := v.times()
	k := structure.Section
	switch {
	case r == b:
		k = structure.Scene
	case r == 's':
	default:
		return
	}
	tr := v.unitTree()
	back := b == '['
	to, ok := tr.HeadAfter(v.curLine(), k, n, back)
	if !ok {
		where := "after this"
		if back {
			where = "before this"
		}
		v.say("No " + unitWord(k) + " heading " + where)
		return
	}
	bf := v.e.Buffer()
	switch {
	case v.helix != nil && v.helix():
		first, last, _ := tr.Span(to, k)
		from, end := bf.LineStart(first), min(bf.LineEnd(last)+1, bf.Len())
		if v.mode == vimVisual { // select mode: the selection grows to it
			s, e := bf.Selection()
			from, end = min(from, s), max(end, e)
		}
		v.e.Navigate(func(b *buffer.Buffer) { b.Select(from, end) })
		v.helixUnit = [3]int{0, 0, 0}
	case v.visual():
		v.vcur = bf.LineStart(to)
		v.select_()
	default:
		v.e.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(to), false) })
	}
}

// putUnit puts a scene or section from the register before or after the
// cursor's (a heading pasted inside a scene would be none, no blank line
// above it), n times; false if the register holds no unit, or the
// cursor is in none of its kind (a plain put then).
func (v *vim) putUnit(before bool, n int) bool {
	if v.unitKind == 0 || v.reg != v.unitText {
		return false
	}
	k, _ := unitKindOf(v.unitKind)
	tr := v.unitTree()
	first, last, ok := tr.Span(v.curLine(), k)
	if !ok {
		return false
	}
	text := strings.Repeat(strings.TrimRight(v.reg, "\n")+"\n\n", max(n, 1))
	v.edit(func(b *buffer.Buffer, t []rune, _ int) {
		at := b.LineStart(first)
		if !before {
			if tl := tr.Trail(last); tl+1 < b.LineCount() {
				at = b.LineStart(tl + 1)
			} else {
				at = b.Len()
			}
		}
		pre := ""
		switch {
		case at > 0 && t[at-1] != '\n':
			pre = "\n\n"
		case at > 1 && t[at-2] != '\n':
			pre = "\n"
		}
		b.Replace(at, at, pre+text)
		b.SetCursor(at+len([]rune(pre)), false)
	})
	return true
}
