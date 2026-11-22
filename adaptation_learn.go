package main

import (
	"slices"
	"strings"

	"github.com/LaPingvino/lexington/lex"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/storyread"
)

// Learning from what the writer does (docs/ADAPTATION.md, step 7): what
// is learned goes into the adaptation block (adaptation_learned.go). It
// is quiet (no question, no status), cheap (on the source pane's follow
// after typing pauses, the cursor's scene only), and never touches the
// writer's own text: only the block changes, as part of the writer's
// last undo step, so an Undo of what taught it takes the lesson too.
//
// It learns from actions, not from how the script stands: a scene's
// heading and its cue lines as they were when the script was opened are
// known already; a new one, or one changed, is looked at once, when the
// cursor has left it (a name or heading half typed teaches nothing). A
// lesson the writer takes away (the pane's What Accolade learned) is not
// learned again in the session.
//
//   - A cue Tab put in from the source, renamed: the source's name for
//     the speaker becomes another name of the new cue (characters:). The
//     same for a line the source reads as someone's, written under
//     another cue. Not when the old name's own cue still speaks elsewhere
//     in the script: then the reading was wrong, not the name.
//   - A cue Tab put in, its name deleted (the line left with no cue over
//     it), or renamed to another cue than the one learned: a strike; two,
//     and the name is no one (not characters). A line deleted whole or
//     reworded says nothing about the name.
//   - Shared live, a peer learns only from what it wrote: a heading or a
//     cue line whose first rune is its own (wroteLine, the single-writer
//     rule of docs/COLLAB.md). Every peer sees a co-writer's new heading;
//     only its writer learns from it, so the lesson is written once. A
//     cue Tab put in is watched only where Tab was pressed.
//   - A heading written for a scene linked to a passage (written over
//     the placeholder of a scene Adapt… started, or a link put under a heading written in this
//     session), other than any the source's place makes: the place maps
//     to it (places:), without the time. A link added under a heading
//     the script had when it was opened teaches nothing.
//   - Skip on a suggested link: written at once (adaptation_suggest.go).

// takenCue is a cue a suggestion put in: the source's name for the
// speaker, the cue, the line under it.
type takenCue struct {
	Speaker, Cue, Text string
	words              []string
	from               *suggestion // Tab again replaces it with another of the same
}

// maxTaken is how many cues put in are watched at most.
const maxTaken = 16

// learner is a window's learning state, for the session.
type learner struct {
	taken     []takenCue
	strikes   map[string]int  // a source name's cue thrown away, by wordsKey
	forgotten map[string]bool // lessons taken away by the writer
	known     map[string]bool // headings with their links, cues with their lines, seen already
	knownFor  string          // the file they are of
	knownOK   bool
	rev       uint64 // looked at last for this revision and line
	line      int
	ran, busy bool
}

// took watches the cue Tab put in with alternative i of s, if it has one
// from the source (Tab again: another alternative of s replaces it).
func (l *learner) took(s *suggestion, i int, takes []takenCue) {
	l.taken = slices.DeleteFunc(l.taken, func(o takenCue) bool { return o.from == s })
	if i >= len(takes) || takes[i].Speaker == "" {
		return
	}
	t := takes[i]
	t.from, t.words = s, normWords(t.Text)
	l.taken = append(l.taken, t)
	if len(l.taken) > maxTaken {
		l.taken = l.taken[len(l.taken)-maxTaken:]
	}
}

// forget keeps a lesson the writer took away from being learned again.
func (l *learner) forget(key string) {
	if l.forgotten == nil {
		l.forgotten = map[string]bool{}
	}
	l.forgotten[key] = true
}

func aliasLesson(cue, name string) string { return "alias\x00" + cue + "\x00" + wordsKey(name) }
func notLesson(name string) string        { return "not\x00" + wordsKey(name) }
func placeLesson(place, heading string) string {
	return "place\x00" + placeKey(place) + "\x00" + heading
}

// dialogueBlock is a cue and the lines under it.
type dialogueBlock struct {
	cueLine, last int    // lines
	cue           string // its name, in capitals
	key           string // its lines' words, normalised, between spaces
}

// blocksIn are the cue blocks of lines [from, to).
func blocksIn(lines []string, from, to int) []dialogueBlock {
	var out []dialogueBlock
	for i := max(from, 0); i < to && i < len(lines); i++ {
		if !isCueLine(lines, i) || blankAt(lines, i+1) {
			continue
		}
		b := dialogueBlock{cueLine: i, cue: strings.ToUpper(cueName(lines[i]))}
		var ws []string
		j := i + 1
		for ; j < len(lines) && !blankAt(lines, j); j++ {
			if t := strings.TrimSpace(lines[j]); !strings.HasPrefix(t, "(") {
				ws = append(ws, normWords(stripNotes(t))...)
			}
		}
		b.last = j - 1
		b.key = " " + strings.Join(ws, " ") + " "
		out = append(out, b)
		i = j
	}
	return out
}

// headingKey is a heading as known: without its scene number, which
// Lock Scene Numbers and a locked script's new scenes get (production.go)
// without the writer writing the heading anew.
func headingKey(heading string) string {
	return "head\x00" + strings.TrimSpace(lex.WithSceneNumber(heading, ""))
}

// knowScript takes in the script as it is: what is there is no action.
func (l *learner) knowScript(lines []string, starts []string) {
	l.known = map[string]bool{}
	for i := range lines {
		if headingAt(lines, i, starts) {
			l.known[headingKey(strings.TrimSpace(lines[i]))] = true
		}
	}
	for _, b := range blocksIn(lines, 0, len(lines)) {
		l.known["cue\x00"+b.cue+"\x00"+b.key] = true
	}
	for _, k := range parseLinks(strings.Join(lines, "\n")) {
		l.known[linkKey(k.Ref.key())] = true
	}
}

// linkKey is a link's key among the learner's known things.
func linkKey(k string) string { return "link\x00" + k }

// knowScript takes in a script just opened: what is in it, the writer
// did not do now.
func (w *MainWindow) knowScript() {
	sg := w.suggester()
	if l := &sg.learn; !l.knownOK || l.knownFor != w.currentFile {
		sg.refresh()
		l.knownOK, l.knownFor = true, w.currentFile
		l.knowScript(sg.lines, w.sceneStarts())
		l.taken = nil
	}
}

// learnFromWriting looks at what the writer did since it last looked,
// and writes what it learns to the adaptation block. The source pane's
// follow calls it (after typing pauses; the pane shown or not).
func (w *MainWindow) learnFromWriting() {
	if w.markdown || w.textEditor == nil {
		return
	}
	sg := w.suggester()
	l := &sg.learn
	starts := w.sceneStarts()
	if l.busy {
		return
	}
	w.knowScript()
	if w.source == nil {
		return
	}
	s := w.source
	if s.ref.Origin == sourceNone || s.loading != nil || len(s.passages) == 0 {
		return
	}
	e := w.textEditor
	b := e.Buffer()
	// not while the writer goes back and forth (Redo would be lost), takes
	// suggestions with Tab, or writes over what is selected
	if b.CanRedo() || sg.cycle.valid(b) || b.HasSelection() {
		return
	}
	line, _ := sg.position()
	if l.ran && l.rev == b.Rev() && l.line == line {
		return
	}
	o := sg.offersFor(s)
	if o == nil {
		return // being read
	}
	sg.refresh()
	prev, hadPrev := l.line, l.ran
	l.ran, l.rev, l.line = true, b.Rev(), line
	next := learnedOf(e.Text())
	changed := l.checkTaken(sg, &next, line)
	sceneAt := func(at int) *sceneCtx {
		if at < 0 || at >= len(sg.lines) {
			return nil
		}
		return sg.scene(s, at, headingAt(sg.lines, at, starts))
	}
	c := sceneAt(line)
	if c != nil && l.checkScene(sg, o, c, &next, line, starts, w.wroteLine) {
		changed = true
	}
	// the scene the cursor just left (a tap elsewhere): what was being
	// written there is done
	if p := sceneAt(prev); hadPrev && prev != line && p != nil && (c == nil || p.head != c.head) {
		if l.checkScene(sg, o, p, &next, line, starts, w.wroteLine) {
			changed = true
		}
	}
	if changed {
		w.writeLearned(next, true)
	}
}

// checkTaken looks at the cues Tab put in: renamed, deleted, or still
// as they were.
func (l *learner) checkTaken(sg *suggester, next *learned, cursor int) bool {
	changed := false
	var keep []takenCue
	for _, t := range l.taken {
		cue, cueLine, at := findTaken(sg.lines, t)
		switch {
		case at >= 0 && cursor >= cueLine && cursor <= at:
			keep = append(keep, t) // being written
		case at >= 0 && (cue == t.Cue || cue == castPlaceholder):
			keep = append(keep, t)
		case at >= 0 && cue != "": // renamed
			if sg.scriptCues()[t.Cue] > 0 {
				break // the old cue still speaks elsewhere: the line was read as the wrong one's
			}
			if next.renameCue(t.Speaker, cue) { // a cue learned before (the source's names read as it)
				changed = true
			} else if a := next.aliasOf(t.Speaker); a != "" && a != cue {
				changed = l.strike(next, t.Speaker) || changed
			} else if a == "" && !next.isNot(t.Speaker) && !l.forgotten[aliasLesson(cue, t.Speaker)] {
				next.addAlias(cue, t.Speaker)
				changed = true
			}
		case at >= 0: // the line there, no name over it any more
			changed = l.strike(next, t.Speaker) || changed
		default:
			// the line gone or reworded: nothing said about the name
		}
	}
	l.taken = keep
	return changed
}

// strike counts a source name thrown away; the second time, it is no one.
func (l *learner) strike(next *learned, name string) bool {
	if l.strikes == nil {
		l.strikes = map[string]int{}
	}
	k := wordsKey(name)
	l.strikes[k]++
	if l.strikes[k] < 2 || next.isNot(name) || l.forgotten[notLesson(name)] {
		return false
	}
	names := []string{name}
	if i := slices.IndexFunc(next.Aliases, func(a aliasEntry) bool { return a.Cue == name }); i >= 0 {
		names = slices.Clone(next.Aliases[i].Names) // a learned cue: the source's names for it
	}
	for _, n := range names {
		next.dropAlias(n)
		next.addNot(n)
	}
	return true
}

// findTaken is where a taken cue's line is now: the cue over it ("" no
// cue), the cue's line, the line's (-1: gone). The line as put in, else
// one under a cue that has most of its words.
func findTaken(lines []string, t takenCue) (cue string, cueLine, at int) {
	at = -1
	over := func(i int) (string, int) {
		c := i - 1
		if c >= 0 && strings.HasPrefix(strings.TrimSpace(lines[c]), "(") {
			c--
		}
		if isCueLine(lines, c) {
			return strings.ToUpper(cueName(lines[c])), c
		}
		return "", max(c, 0) // where the cue was: the cursor there, it is being written
	}
	for i, ln := range lines {
		if strings.TrimSpace(ln) == t.Text {
			c, cl := over(i)
			if at < 0 || c == t.Cue {
				cue, cueLine, at = c, cl, i
			}
		}
	}
	if at >= 0 || len(t.words) < 3 {
		return cue, cueLine, at
	}
	for i, ln := range lines {
		if strings.TrimSpace(ln) == "" || !strings.ContainsAny(ln, " ") {
			continue
		}
		if overlap(t.words, normWords(ln)) >= 0.6 {
			if c, cl := over(i); c != "" || cl < i {
				return c, cl, i
			}
		}
	}
	return "", -1, -1
}

// checkScene looks at the cursor's scene: its heading, if new or changed
// (or its links), and its cue blocks that are new, the one the cursor is
// in left for later. Only lines own says this writer wrote teach: a
// co-writer's are known without a lesson (their writer learns from them).
func (l *learner) checkScene(sg *suggester, o *sourceOffers, c *sceneCtx, next *learned, cursor int, starts []string, own func(line int) bool) bool {
	lines := sg.lines
	changed := false
	// a heading new since the script was opened, once its scene is linked
	// (Adapt…'s scene, or a link put under it): not one written before, a
	// link added to it later
	//
	// Its links must be new too: those another heading had are not
	// put under this one ("EXT. YARD" typed above a scene, its links
	// under it until the blank line after it is typed); seen under a
	// known heading, they are known.
	if c.head >= 0 && c.head < len(lines) && cursor != c.head && c.linked {
		heading := strings.TrimSpace(lines[c.head])
		k := headingKey(heading)
		first := -1
		for _, ln := range c.links {
			if l.known[k] {
				l.known[linkKey(ln.key)] = true
			} else if first < 0 && !l.known[linkKey(ln.key)] {
				first = ln.first
			}
		}
		if !l.known[k] && first >= 0 {
			l.known[k] = true
			for _, ln := range c.links {
				l.known[linkKey(ln.key)] = true
			}
			if own(c.head) && learnPlace(sg, o, first, heading, next, l.forgotten, starts) {
				changed = true
			}
		}
	}
	for _, b := range blocksIn(lines, c.head+1, c.end) {
		if cursor >= b.cueLine && cursor <= b.last {
			continue // being written
		}
		k := "cue\x00" + b.cue + "\x00" + b.key
		if l.known[k] {
			continue
		}
		l.known[k] = true
		if c.linked && own(b.cueLine) && learnCue(sg, o, c, b, next, l.forgotten) {
			changed = true
		}
	}
	return changed
}

// headingBase is a heading without its time and scene number: INT.
// KITCHEN for INT. KITCHEN - NIGHT #12#.
func headingBase(h string) string {
	h, _ = lex.SceneNumber(strings.TrimSpace(h))
	h = strings.TrimPrefix(h, ".")
	if i := strings.LastIndex(h, " - "); i > 0 {
		h = h[:i]
	}
	return strings.TrimSpace(h)
}

// learnPlace maps the place the source reads for passage p to the
// heading the writer gave its scene, if no heading the source made is it.
func learnPlace(sg *suggester, o *sourceOffers, p int, heading string, next *learned, forgotten map[string]bool, starts []string) bool {
	if p < 0 || p >= len(o.places) {
		return false
	}
	g := o.places[p]
	base := headingBase(heading)
	start, words := headingPlaceholder(starts)
	if g.Place == "" || len(strings.Fields(base)) < 2 || strings.Contains(heading, "…") ||
		heading == start+" "+words || base == g.Given {
		return false
	}
	for _, h := range sg.readHeadings(g) {
		if headingBase(h) == base {
			return false
		}
	}
	place := strings.ToLower(g.Place)
	if forgotten[placeLesson(place, base)] || next.placeFor(place) == base {
		return false
	}
	next.setPlace(place, base)
	return true
}

// learnCue makes the source's name for a line's speaker another name of
// the cue the writer put over it, when the line is one the source reads
// as theirs and nothing says the reading was wrong.
func learnCue(sg *suggester, o *sourceOffers, c *sceneCtx, b dialogueBlock, next *learned, forgotten map[string]bool) bool {
	if b.cue == "" || b.cue == castPlaceholder {
		return false
	}
	for _, p := range c.passages {
		if p >= len(o.pieces) {
			continue
		}
		for _, pc := range o.pieces[p] {
			if !pc.Speech || pc.Speaker == "" || pc.Speaker == storyread.Narrator || pc.words < 3 || !strings.Contains(b.key, pc.key) {
				continue
			}
			own := sg.cueFor(pc.Speaker)
			switch {
			case own == b.cue:
				return false // as the source has it
			case next.isNot(pc.Speaker) || next.aliasOf(pc.Speaker) != "" || forgotten[aliasLesson(b.cue, pc.Speaker)]:
				return false
			case sg.scriptCues()[own] > 0:
				return false // that one speaks in the script: the line was read as the wrong one's
			}
			next.addAlias(b.cue, pc.Speaker)
			return true
		}
	}
	return false
}

// writeLearned puts l in the script's adaptation block; quiet (learned
// from the last edit): part of its undo step, the cursor and selection
// kept; else (a button) an undo step of its own. It reports whether the
// block changed.
func (w *MainWindow) writeLearned(l learned, quiet bool) bool {
	e := w.textEditor
	text := e.Text()
	inside, _ := adaptationBlock(text)
	nu := withLearned(inside, l)
	if nu == inside {
		return false
	}
	start, end, block := withAdaptationBlock(text, nu)
	sg := w.suggester()
	b := e.Buffer()
	keep := -1 // the cursor on an empty line before the block: kept there, a blank line after it
	if c := b.Cursor(); quiet && c > start && c <= end {
		before := []rune(text)[start:c]
		if strings.Trim(string(before), "\n") != "" {
			return false // in the block: written by hand now
		}
		keep = c
		block = string(before) + "\n\n" + strings.TrimLeft(block, "\n")
	}
	dismissed := sg.dismissed.on && sg.dismissed.rev == b.Rev()
	sg.learn.busy = true
	e.Edit(func(b *buffer.Buffer) {
		if quiet {
			b.Amend(start, end, block)
			if keep >= 0 {
				b.SetCursor(keep, false)
			}
			return
		}
		cursor := b.Cursor()
		b.Replace(start, end, block)
		if cursor > start {
			cursor = min(cursor, start)
		}
		b.SetCursor(cursor, false)
	})
	sg.learn.busy = false
	if dismissed { // Esc put the hint away: the block changing does not bring it back
		sg.dismissed.rev = b.Rev()
	}
	if quiet {
		line, _ := sg.position()
		sg.learn.rev, sg.learn.line = b.Rev(), line
	}
	return true
}
