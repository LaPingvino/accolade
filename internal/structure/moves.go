package structure

import (
	"fmt"
	"slices"
	"strings"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
)

// Op is a structural move.
type Op int

const (
	MoveUp    Op = iota // the unit at the cursor before the one above it
	MoveDown            // after the one below it
	Promote             // out: a section a level up, a scene out of its section, a line of a speech out into action
	Demote              // in: a section a level down, a scene into the next section, action into the speech beside it
	IntoPrev            // into the unit before its parent: a speech into the scene before
	IntoNext            // into the next one: a speech into the next scene
	Barf                // the scene's last speech or paragraph out into the next scene
	Slurp               // the next scene's first speech or paragraph into this one
	Transpose           // two speakers' lines: the cues of this speech and the one before change places
)

// run is units side by side: kids i to j of parent.
type run struct {
	parent *Node
	i, j   int
}

func (r run) first() *Node { return r.parent.Kids[r.i] }
func (r run) last() *Node  { return r.parent.Kids[r.j] }
func (r run) kind() Kind   { return r.first().Kind }
func (r run) lines() (int, int) {
	return r.first().First, r.last().Last
}
func (r run) nodes() []*Node { return slices.Clone(r.parent.Kids[r.i : r.j+1]) }

func one(n *Node) run { return run{n.Parent, index(n), index(n)} }

// unitAt is the unit at line i: a heading's section or scene, else the
// block it is in (a blank line: the one above).
func (t *Tree) unitAt(i int) *Node {
	for i > t.Front && t.sep(i) {
		i--
	}
	p := t.path(i)
	n := p[len(p)-1]
	if n.Kind == Line {
		n = n.Parent
	}
	if n.Kind == Body {
		return nil
	}
	return n
}

// selected is the units lines a to b select: the one at the cursor, or
// those side by side that the lines are in (a speech's lines when they
// are in one speech).
func (t *Tree) selected(a, b int) (run, error) {
	if a > b {
		a, b = b, a
	}
	if a == b {
		n := t.unitAt(a)
		if n == nil {
			return run{}, Refusal("Nothing to move here: the title page or an empty script")
		}
		return one(n), nil
	}
	pa, pb := t.path(a), t.path(b)
	k := 0
	for k+1 < len(pa) && k+1 < len(pb) && pa[k+1] == pb[k+1] {
		k++
	}
	p := pa[k]
	if p.Kind == Line {
		p = p.Parent
		return one(p), nil
	}
	if p.Kind != Body && (p.inHead(a) || p.inHead(b) || p.Kind == Block && len(p.Kids) == 0) {
		return one(p), nil
	}
	var ca, cb *Node
	if k+1 < len(pa) {
		ca = pa[k+1]
	} else if i := slices.IndexFunc(p.Kids, func(n *Node) bool { return n.First > a }); i >= 0 {
		ca = p.Kids[i]
	}
	if k+1 < len(pb) {
		cb = pb[k+1]
	} else {
		for _, n := range p.Kids {
			if n.Last < b {
				cb = n
			}
		}
	}
	if ca == nil || cb == nil || index(ca) > index(cb) {
		if p.Kind == Body {
			return run{}, Refusal("Nothing to move in the selection")
		}
		return one(p), nil
	}
	if ca.Kind != cb.Kind {
		return run{}, Refusal(fmt.Sprintf("The selection holds a %s and a %s: select units of one kind", ca.Kind, cb.Kind))
	}
	return run{p, index(ca), index(cb)}, nil
}

// Do makes the move op on the unit at lines a to b (the cursor's line
// twice, or a selection's first and last), checked with read.
func (t *Tree) Do(op Op, a, b int, read Reader) (Result, error) {
	if len(t.S.Lines) == 0 {
		return Result{}, Refusal("Nothing to move")
	}
	a, b = clamp(a, len(t.S.Lines)), clamp(b, len(t.S.Lines))
	if a > b {
		a, b = b, a
	}
	if a < t.Front || a >= t.S.End {
		return Result{}, Refusal("Moves work in the script's body: not on the title page or in the blocks at its end")
	}
	var res Result
	var err error
	switch op {
	case Barf, Slurp:
		res, err = t.barfSlurp(op, a)
	case Transpose:
		res, err = t.transpose(a)
	default:
		r, e := t.selected(a, b)
		if e != nil {
			return Result{}, e
		}
		switch op {
		case MoveUp, MoveDown:
			res, err = t.step(r, op == MoveDown)
		case Promote:
			res, err = t.promote(r, a, b)
		case Demote:
			res, err = t.demote(r, a, b)
		case IntoPrev, IntoNext:
			res, err = t.into(r, op == IntoNext)
		}
	}
	if err != nil {
		return Result{}, err
	}
	if err := t.check(res, read); err != nil {
		return Result{}, err
	}
	return res, nil
}

func clamp(i, n int) int { return min(max(i, 0), n-1) }

// sibs are the units of r's kind beside it: the kids of its parent of
// that kind (they stand together: blocks, then scenes, then sections).
func (r run) sibs() (lo, hi int) {
	k := r.kind()
	lo, hi = r.i, r.j
	for lo-1 >= 0 && r.parent.Kids[lo-1].Kind == k {
		lo--
	}
	for hi+1 < len(r.parent.Kids) && r.parent.Kids[hi+1].Kind == k {
		hi++
	}
	return lo, hi
}

// movedKeys are units that move, for the check.
func (t *Tree) movedKeys(ns ...*Node) map[*Node]bool {
	out := map[*Node]bool{}
	for _, n := range ns {
		out[n] = true
	}
	return out
}

// step moves r past the unit of its kind before it (down: after it).
func (t *Tree) step(r run, down bool) (Result, error) {
	lo, hi := r.sibs()
	sep := r.kind() != Line
	what := r.kind().String()
	if !down {
		if r.i == lo {
			return Result{}, Refusal(fmt.Sprintf("Nothing to move it past: the first %s here%s", what, t.crossHint(r, false)))
		}
		p := r.parent.Kids[r.i-1]
		a0, a1 := r.lines()
		lines, at, _, where := t.swap(p.First, p.Last, a0, a1, sep)
		res := t.result(lines, at, where, fmt.Sprintf("Moved up, before %s", t.Name(p)))
		res.moved, res.kept = t.movedKeys(r.nodes()...), true
		return res, nil
	}
	if r.j == hi {
		return Result{}, Refusal(fmt.Sprintf("Nothing to move it past: the last %s here%s", what, t.crossHint(r, true)))
	}
	nx := r.parent.Kids[r.j+1]
	a0, a1 := r.lines()
	lines, at, where, _ := t.swap(a0, a1, nx.First, nx.Last, sep)
	res := t.result(lines, at, where, fmt.Sprintf("Moved down, after %s", t.Name(nx)))
	res.moved, res.kept = t.movedKeys(r.nodes()...), true
	return res, nil
}

// crossHint says how a block goes on into the scene beside (a
// section's own block: the section beside).
func (t *Tree) crossHint(r run, down bool) string {
	k := r.kind()
	plan := k == Block && r.parent.Kind == Section && t.planning(r)
	switch {
	case plan && down:
		return " (Alt+Shift+Down takes it into the next section)"
	case plan:
		return " (Alt+Shift+Up takes it into the section before)"
	case k == Block && down:
		return " (Alt+Shift+Down takes it into the next scene)"
	case k == Block:
		return " (Alt+Shift+Up takes it into the scene before)"
	case k == Scene && down:
		return " (Alt+Right takes it into the next section)"
	case k == Scene:
		return " (Alt+Left takes it out of its section)"
	}
	return ""
}

// InPlace: a move that would leave the text as it is.
const InPlace = Refusal("It is there already")

// moveTo moves r before line to (after: after it) as a move of units.
func (t *Tree) moveTo(r run, to int, after bool, msg string) (Result, error) {
	a0, a1 := r.lines()
	lines, at, where := t.move(a0, a1, to, after, r.kind() != Line)
	if slices.Equal(lines, t.S.Lines) {
		return Result{}, InPlace
	}
	res := t.result(lines, at, where, msg)
	res.moved, res.kept = t.movedKeys(r.nodes()...), true
	return res, nil
}

// promote: out of what holds it.
func (t *Tree) promote(r run, a, b int) (Result, error) {
	switch r.kind() {
	case Section:
		p := r.parent
		if p.Kind == Body {
			lvl := r.first().Level
			for _, n := range r.nodes() {
				lvl = min(lvl, n.Level)
			}
			if lvl <= 1 {
				return Result{}, Refusal("A section at the top already (#)")
			}
			return t.relevel(r, -1, fmt.Sprintf("%s a level up", t.Name(r.first())))
		}
		// the next section after its parent, at its parent's level
		delta := p.Level - r.first().Level
		lines := t.relevelLines(r, delta)
		sub := &Tree{S: t.S, Root: t.Root, Front: t.Front, bone: t.bone}
		sub.S.Lines = lines
		a0, a1 := r.lines()
		nl, at, where := sub.move(a0, a1, p.Last, true, true)
		res := t.result(nl, at, where, fmt.Sprintf("%s out of %s, after it", t.Name(r.first()), t.Name(p)))
		res.moved = t.movedKeys(r.nodes()...)
		return res, nil
	case Scene:
		p := r.parent
		if p.Kind != Section {
			return Result{}, Refusal("This scene is in no section to come out of")
		}
		return t.moveTo(r, p.First, false, fmt.Sprintf("Out of %s, before it", t.Name(p)))
	case Block:
		if r.i == r.j && r.first().Speech && !r.first().Dual {
			bl := r.first()
			lo, hi := max(a, bl.HeadEnd+1), min(b, bl.Last)
			if a == b || lo > hi {
				if a <= bl.HeadEnd {
					return Result{}, Refusal("Put the cursor on the line of the speech to make action")
				}
			}
			return t.raise(bl, lo, hi)
		}
		if t.planning(r) {
			return Result{}, Refusal(planHint)
		}
		return Result{}, Refusal("Action already: Alt+Right makes it a line of the speech beside it")
	case Line:
		bl := r.parent
		if !bl.Speech {
			return Result{}, Refusal("Action already: Alt+Right makes it a line of the speech beside it")
		}
		a0, a1 := r.lines()
		return t.raise(bl, a0, a1)
	}
	return Result{}, Refusal("Nothing to bring out here")
}

// demote: into the unit before or after it.
func (t *Tree) demote(r run, a, b int) (Result, error) {
	switch r.kind() {
	case Section:
		if r.i == 0 || r.parent.Kids[r.i-1].Kind != Section { // none before it to go into: a level down, where it is
			return t.relevel(r, 1, fmt.Sprintf("%s a level down", t.Name(r.first())))
		}
		prev := r.parent.Kids[r.i-1]
		return t.relevel(r, prev.Level+1-r.first().Level, fmt.Sprintf("%s into %s", t.Name(r.first()), t.Name(prev)))
	case Scene:
		_, last := r.lines()
		var next *Node
		for _, c := range t.containers() {
			if c.Kind == Section && c.First > last {
				next = c
				break
			}
		}
		if next == nil {
			return Result{}, Refusal("No section after it to go into")
		}
		for _, k := range next.Kids {
			if k.container() {
				return t.moveTo(r, k.First, false, fmt.Sprintf("Into %s", t.Name(next)))
			}
		}
		return t.moveTo(r, next.Last, true, fmt.Sprintf("Into %s", t.Name(next)))
	case Block, Line:
		bl := r.first()
		if r.kind() == Line {
			bl = r.parent
		}
		if bl.Speech {
			return Result{}, Refusal("A speech already: Alt+Left makes its line action")
		}
		return t.lower(one(bl))
	}
	return Result{}, Refusal("Nothing to take in here")
}

// relevelLines is the lines with r's sections (and the sections in them)
// delta levels deeper.
func (t *Tree) relevelLines(r run, delta int) []string {
	change := map[int]string{}
	var walk func(n *Node)
	walk = func(n *Node) {
		if n.Kind == Section {
			l := t.S.Lines[n.First]
			indent := l[:len(l)-len(strings.TrimLeft(l, " \t"))]
			rest := strings.TrimLeft(l, " \t")
			rest = rest[len(rest)-len(strings.TrimLeft(rest, "#")):]
			change[n.First] = indent + strings.Repeat("#", max(n.Level+delta, 1)) + rest
		}
		for _, k := range n.Kids {
			walk(k)
		}
	}
	for _, n := range r.nodes() {
		walk(n)
	}
	return t.replaced(change)
}

// relevel changes r's sections' levels where they are.
func (t *Tree) relevel(r run, delta int, msg string) (Result, error) {
	if delta == 0 {
		return Result{}, Refusal("At that level already")
	}
	lines := t.relevelLines(r, delta)
	at := make([]int, len(lines))
	for i := range at {
		at[i] = i
	}
	a0, a1 := r.lines()
	res := t.result(lines, at, [2]int{a0, a1}, msg)
	res.moved = t.movedKeys(r.nodes()...)
	return res, nil
}

// scenes are the scenes in the text's order.
func (t *Tree) scenes() []*Node {
	var out []*Node
	for _, c := range t.containers() {
		if c.Kind == Scene {
			out = append(out, c)
		}
	}
	return out
}

// into takes r into the unit before its parent (next: after it): blocks
// into the scene before or after, scenes out of or into a section.
func (t *Tree) into(r run, next bool) (Result, error) {
	switch r.kind() {
	case Scene:
		if next {
			return t.demote(r, 0, 0)
		}
		return t.promote(r, 0, 0)
	case Block:
	case Line:
		return Result{}, Refusal("A line goes out of its speech with Alt+Left")
	default:
		return Result{}, Refusal("A section moves with Alt+Up and Alt+Down, a level with Alt+Left and Alt+Right")
	}
	a0, a1 := r.lines()
	own := r.parent
	if own.Kind == Section && t.planning(r) {
		return t.intoSection(r, next)
	}
	if next {
		for _, s := range t.scenes() {
			if s.First > a1 && s != own {
				return t.toSceneStart(r, s)
			}
		}
		return Result{}, Refusal("No scene after it to go into")
	}
	var prev *Node
	for _, s := range t.scenes() {
		if s.Last < a0 && s != own {
			prev = s
		}
	}
	if prev == nil {
		return Result{}, Refusal("No scene before it to go into")
	}
	return t.moveTo(r, prev.Last, true, fmt.Sprintf("Into %s, at its end", t.Name(prev)))
}

// intoSection takes a section's own synopses and notes into the section before it in the text (next: after
// it), among that one's own blocks: at their end, or their start (after
// its own synopsis, which stays the first). A
// section's blocks stay a section's, as a scene's go into scenes.
func (t *Tree) intoSection(r run, next bool) (Result, error) {
	own := r.parent
	var to *Node
	for _, n := range t.containers() {
		if n.Kind != Section || n == own {
			continue
		}
		if next && n.First > own.First {
			to = n
			break
		}
		if !next && n.First < own.First {
			to = n
		}
	}
	if to == nil {
		if next {
			return Result{}, Refusal("No section after it to go into")
		}
		return Result{}, Refusal("No section before it to go into")
	}
	var lastOwn, firstOwn *Node // its own blocks, before what it holds
	for _, k := range to.Kids {
		if k.Kind != Block {
			break
		}
		if firstOwn == nil {
			firstOwn = k
		}
		lastOwn = k
	}
	if next {
		msg := fmt.Sprintf("Into %s, at its start", t.Name(to))
		if firstOwn != nil && t.synopsis(firstOwn) { // its own synopsis stays first
			return t.moveTo(r, firstOwn.Last, true, msg)
		}
		if firstOwn != nil {
			return t.moveTo(r, firstOwn.First, false, msg)
		}
		if len(to.Kids) > 0 {
			return t.moveTo(r, to.Kids[0].First, false, msg)
		}
		return t.moveTo(r, to.HeadEnd, true, msg)
	}
	msg := fmt.Sprintf("Into %s, after its own", t.Name(to))
	switch {
	case lastOwn != nil:
		return t.moveTo(r, lastOwn.Last, true, msg)
	case len(to.Kids) > 0:
		return t.moveTo(r, to.Kids[0].First, false, msg)
	}
	return t.moveTo(r, to.HeadEnd, true, msg)
}

// planHint is what Alt+Left and Alt+Right say on a synopsis or a note.
const planHint = "A synopsis or a note moves with Alt+Up and Alt+Down, and into the section or scene beside with Shift+Alt+Up and Shift+Alt+Down"

// planning tells whether blocks r are all a section's plan: synopses
// and notes, which go from section to section (the rest, as FADE IN:,
// goes into scenes).
func (t *Tree) planning(r run) bool {
	for _, n := range r.nodes() {
		l := strings.TrimSpace(t.S.Lines[n.First])
		if !t.synopsis(n) && !(strings.HasPrefix(l, "[[") && strings.HasSuffix(strings.TrimSpace(t.S.Lines[n.Last]), "]]")) {
			return false
		}
	}
	return true
}

// synopsis tells whether block n is a synopsis paragraph ("= …").
func (t *Tree) synopsis(n *Node) bool {
	l := strings.TrimSpace(t.S.Lines[n.First])
	return strings.HasPrefix(l, "=") && !strings.HasPrefix(l, "==")
}

// toSceneStart moves blocks r to the start of scene s, under its heading.
func (t *Tree) toSceneStart(r run, s *Node) (Result, error) {
	msg := fmt.Sprintf("Into %s, at its start", t.Name(s))
	if len(s.Kids) > 0 {
		return t.moveTo(r, s.Kids[0].First, false, msg)
	}
	return t.moveTo(r, s.HeadEnd, true, msg)
}

// barfSlurp: the last block of the scene at line a out into the next
// scene (Barf), or the next scene's first block into this one (Slurp).
func (t *Tree) barfSlurp(op Op, a int) (Result, error) {
	var sc *Node
	for _, n := range t.path(a) {
		if n.Kind == Scene {
			sc = n
		}
	}
	if sc == nil {
		return Result{}, Refusal("Put the cursor in a scene")
	}
	var next *Node
	for _, s := range t.scenes() {
		if s.First > sc.First {
			next = s
			break
		}
	}
	if next == nil {
		return Result{}, Refusal("No scene after this one")
	}
	if op == Barf {
		if len(sc.Kids) == 0 {
			return Result{}, Refusal("Nothing in this scene to pass on")
		}
		res, err := t.toSceneStart(one(sc.Kids[len(sc.Kids)-1]), next)
		if err == nil {
			res.Msg = fmt.Sprintf("Its last %s passed on into %s", blockWord(sc.Kids[len(sc.Kids)-1]), t.Name(next))
		}
		return res, err
	}
	if len(next.Kids) == 0 {
		return Result{}, Refusal("Nothing in the next scene to take in")
	}
	res, err := t.moveTo(one(next.Kids[0]), sc.Last, true, "")
	if err == nil {
		res.Msg = fmt.Sprintf("Took in the first %s of %s", blockWord(next.Kids[0]), t.Name(next))
	}
	return res, err
}

func blockWord(n *Node) string {
	if n.Speech {
		return "speech"
	}
	return "paragraph"
}

// cueParts is a cue's line in three: what comes before the name (its
// indent, an @), the name, and what comes after (extensions, a ^).
func cueParts(l string) (pre, name, post string) {
	i := len(l) - len(strings.TrimLeft(l, " \t‎‏"))
	if strings.HasPrefix(l[i:], "@") {
		i++
	}
	pre, rest := l[:i], l[i:]
	j := len(rest)
	if k := strings.IndexAny(rest, "(^"); k >= 0 {
		j = k
	}
	name = strings.TrimRight(rest[:j], " \t")
	return pre, name, rest[len(name):]
}

// transpose gives this speech's lines to the speaker before and theirs
// to this one: the two cues' names change places (side by side ones:
// the pair's).
func (t *Tree) transpose(a int) (Result, error) {
	b := t.unitAt(a)
	if b == nil || b.Kind != Block || !b.Speech {
		return Result{}, Refusal("Put the cursor in a speech")
	}
	var x, y int // the two cues' lines
	if b.Dual {
		x = b.First
		for l := b.First + 1; l <= b.Last; l++ {
			if t.S.Kinds[l] == syntax.Character {
				y = l
			}
		}
	} else {
		i := index(b)
		if i == 0 || !b.Parent.Kids[i-1].Speech {
			return Result{}, Refusal("No speech right before this one to change lines with")
		}
		p := b.Parent.Kids[i-1]
		x, y = p.First, b.First
		if p.Dual {
			for l := p.First + 1; l <= p.Last; l++ {
				if t.S.Kinds[l] == syntax.Character {
					x = l
				}
			}
		}
	}
	xp, xn, xs := cueParts(t.S.Lines[x])
	yp, yn, ys := cueParts(t.S.Lines[y])
	if strings.EqualFold(xn, yn) {
		return Result{}, Refusal("Both lines are " + xn + "'s")
	}
	lines := t.replaced(map[int]string{x: xp + yn + xs, y: yp + xn + ys})
	at := make([]int, len(lines))
	for i := range at {
		at[i] = i
	}
	res := t.result(lines, at, [2]int{min(x, y), b.Last}, fmt.Sprintf("%s's and %s's lines changed places", xn, yn))
	return res, nil
}

// contd is a cue for a speech going on after action: its name with
// (CONT'D), without a ^.
func contd(cue string) string {
	c := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(cue), "^"))
	if strings.Contains(strings.ToUpper(c), "CONT'D") {
		return c
	}
	return c + " (CONT'D)"
}

// sameSpeaker tells whether two cues are one character's.
func sameSpeaker(a, b string) bool {
	_, an, _ := cueParts(a)
	_, bn, _ := cueParts(b)
	return an != "" && strings.EqualFold(an, bn)
}

// raise makes lines lo to hi of speech bl action: after the speech (its
// last lines), before it (its first), or between it and the rest of it
// under the cue again (CONT'D). A parenthetical right before them goes
// with them; the speech keeps at least one line.
func (t *Tree) raise(bl *Node, lo, hi int) (Result, error) {
	L, K := t.S.Lines, t.S.Kinds
	for lo-1 > bl.HeadEnd && K[lo-1] == syntax.Parenthetical {
		lo--
	}
	for hi+1 <= bl.Last && K[hi+1] == syntax.Parenthetical && hi+1 == bl.Last {
		hi++ // a parenthetical left last in the speech: it goes too
	}
	before := bl.HeadEnd + 1
	if lo == before && hi == bl.Last {
		return Result{}, Refusal("Those are all the speech's lines: a cue needs its line (select fewer, or change the cue's element)")
	}
	only := true
	for l := lo; l <= hi; l++ {
		only = only && K[l] == syntax.Parenthetical
	}
	if only {
		return Result{}, Refusal("A parenthetical goes with its line: raise the line")
	}
	var act []string
	for l := lo; l <= hi; l++ {
		act = append(act, t.asAction(L[l]))
	}
	n := len(L)
	var pieces []piece
	switch {
	case hi == bl.Last:
		pieces = []piece{span(0, lo-1), lit(""), lit(act...), span(hi+1, n-1)}
	case lo == before:
		pieces = []piece{span(0, bl.First-1), lit(act...), lit(""), span(bl.First, bl.First), span(hi+1, n-1)}
	default:
		pieces = []piece{span(0, lo-1), lit(""), lit(act...), lit("", contd(L[bl.First])), span(hi+1, n-1)}
	}
	lines, at, st := t.assemble(pieces, false)
	actAt := st[2]
	if lo == before && hi != bl.Last {
		actAt = st[1]
	}
	res := t.result(lines, at, [2]int{actAt, actAt + len(act) - 1}, "Made action, out of the speech")
	return res, nil
}

// asAction is a line of dialogue written as action: with a ! where it
// would read as something else (capitals: a cue).
func (t *Tree) asAction(l string) string {
	s := strings.TrimSpace(l)
	if s != "" && (s == strings.ToUpper(s) && strings.ToUpper(s) != strings.ToLower(s) ||
		strings.HasPrefix(s, "#") || strings.HasPrefix(s, "=") || strings.HasPrefix(s, ">") ||
		strings.HasPrefix(s, ".") || strings.HasPrefix(s, "@") || strings.HasPrefix(s, "~") || strings.HasPrefix(s, "(")) {
		return "!" + s
	}
	return s
}

// lower makes the action r a speech's lines: the last of the speech
// before it (and the speech after, if it is the same speaker's going on,
// joined to it), else the first of the speech after.
func (t *Tree) lower(r run) (Result, error) {
	bl := r.first()
	if t.bone[bl.First] {
		return Result{}, Refusal("A boneyard (/* … */) is no speech's")
	}
	if t.planning(r) {
		return Result{}, Refusal(planHint)
	}
	for l := bl.First; l <= bl.Last; l++ {
		if k := t.S.Kinds[l]; k != syntax.Action && k != syntax.Note {
			return Result{}, Refusal("Only action becomes a speech's lines")
		}
	}
	var lines []string
	for l := bl.First; l <= bl.Last; l++ {
		lines = append(lines, strings.TrimPrefix(strings.TrimSpace(t.S.Lines[l]), "!"))
	}
	kids := r.parent.Kids
	i := index(bl)
	var prev, next *Node
	if i > 0 && kids[i-1].Speech {
		prev = kids[i-1]
	}
	if i+1 < len(kids) && kids[i+1].Speech {
		next = kids[i+1]
	}
	n := len(t.S.Lines)
	var pieces []piece
	at0 := 0
	switch {
	case prev != nil && !prev.Dual:
		end := n - 1
		rest := bl.Last + 1
		msg := "Now the end of " + t.Name(prev) + "'s speech"
		if next != nil && !next.Dual && sameSpeaker(t.S.Lines[prev.First], t.S.Lines[next.First]) {
			rest = next.HeadEnd + 1 // the speech going on: one again
			msg = "Back in " + t.Name(prev) + "'s speech, one again"
		}
		pieces = []piece{span(0, prev.Last), lit(lines...), span(rest, end)}
		ls, at, st := t.assemble(pieces, false)
		at0 = st[1]
		res := t.result(ls, at, [2]int{at0, at0 + len(lines) - 1}, msg)
		return res, nil
	case next != nil && !next.Dual:
		pieces = []piece{span(0, bl.First-1), span(next.First, next.HeadEnd), lit(lines...), span(next.HeadEnd+1, n-1)}
		ls, at, st := t.assemble(pieces, false)
		at0 = st[2]
		res := t.result(ls, at, [2]int{at0, at0 + len(lines) - 1}, "Now the start of "+t.Name(next)+"'s speech")
		return res, nil
	}
	return Result{}, Refusal("No speech right beside it to take it in")
}
