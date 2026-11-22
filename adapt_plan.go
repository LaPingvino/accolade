package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/LaPingvino/lexington/lex"
)

// Where adapting goes on, and what Adapt… writes (docs/help/adapting.md,
// "Next and Adapt"): pure functions of the script, its source and where
// the writer is, so they are tested without a window.
//
// Next and Previous only move: from the anchor (what the writer is
// looking at) to the next or previous passage still to do, in the
// source's order. Writing goes through Adapt…, which offers how and
// where with the likely choice first and a preview of what goes in.

// anchorKind is what the anchor was taken from.
type anchorKind int

const (
	anchorStart  anchorKind = iota // nothing to go by: the source's start
	anchorPicked                   // the passage picked in the pane since the cursor last moved
	anchorScene                    // the last passage the cursor's scene links to
	anchorBefore                   // the last passage of the nearest linked scene before the cursor
	anchorView                     // the passage at the top of the pane's view
)

// anchor is where Next and Previous go on from: a passage and why.
type anchor struct {
	passage int // -1: before the first
	kind    anchorKind
	span    int // the scene's first line (anchorScene, anchorBefore)
}

// inclusive reports whether the anchor's own passage is a candidate:
// the top of the view is where the writer is, not something done.
func (a anchor) inclusive() bool { return a.kind == anchorView }

// anchorAt is the anchor at the cursor's line: the passage picked (if
// picked since the cursor last moved; -1 none); else the passages the
// cursor's scene links to, the last of them in the source's order; else
// the nearest scene before the cursor that links any, its last; else
// the passage at the top of the pane's view (-1: not known); else the
// source's start.
func (m *sourceModel) anchorAt(line, picked, viewTop int) anchor {
	if picked >= 0 && picked < len(m.passages) {
		return anchor{passage: picked, kind: anchorPicked}
	}
	cur := spanAt(m.spans, line)
	k := slices.IndexFunc(m.spans, func(o sceneSpan) bool { return o.Line == cur.Line })
	for ; k >= 0; k-- {
		at := -1
		for i, l := range m.links {
			if mt := m.matches[i]; mt.found() && l.Line >= m.spans[k].Line && l.Line < m.spans[k].End {
				at = max(at, mt.last())
			}
		}
		if at >= 0 {
			kind := anchorScene
			if m.spans[k].Line != cur.Line {
				kind = anchorBefore
			}
			return anchor{passage: at, kind: kind, span: m.spans[k].Line}
		}
	}
	if viewTop >= 0 && viewTop < len(m.passages) {
		return anchor{passage: viewTop, kind: anchorView}
	}
	return anchor{passage: -1, kind: anchorStart}
}

// step is the passage still to do after the anchor (dir 1) or before it
// (dir -1), in the source's order; -1 if none is left that way.
func (m *sourceModel) step(a anchor, dir int) int {
	from := a.passage
	if dir > 0 {
		if a.inclusive() {
			from--
		}
		for i := from + 1; i < len(m.status); i++ {
			if m.status[i] == statusToDo {
				return i
			}
		}
		return -1
	}
	if a.kind == anchorStart {
		return -1
	}
	if a.inclusive() {
		from++
	}
	for i := min(from, len(m.status)) - 1; i >= 0; i-- {
		if m.status[i] == statusToDo {
			return i
		}
	}
	return -1
}

// wrapTo is the first passage still to do (dir 1), or the last (dir -1);
// -1 if none is.
func (m *sourceModel) wrapTo(dir int) int {
	if dir > 0 {
		return m.step(anchor{passage: -1, kind: anchorStart}, 1)
	}
	return m.step(anchor{passage: len(m.status), kind: anchorPicked}, -1)
}

// spanLabels are the script's spans' names as the pane shows them, by
// their first line: a scene by its number ("Scene 4", the locked label
// if the numbers are locked, else counted), a section by its heading,
// the part before any heading as such.
func spanLabels(text string, starts []string) map[int]string {
	body := text[:bodyEndByte(text)]
	heads := sceneHeads(body, starts)
	o := lex.NumberOptions{}
	if l := sceneLockOf(text); l.locked {
		o = lex.NumberOptions{Locked: true, Style: l.style}
	}
	nums := sceneNumbersFor(heads, o)
	out := map[int]string{0: "the title page"}
	for i, h := range heads {
		n := nums[i]
		if n == "" {
			n = fmt.Sprint(i + 1)
		}
		out[h.line] = "Scene " + n
	}
	for _, it := range outlineOf(body, starts) {
		if !it.Scene {
			out[it.Line] = strings.Repeat("#", max(it.Level, 1)) + " " + it.Text
		}
	}
	return out
}

// adaptHow is how Adapt… adapts a passage.
type adaptHow int

const (
	howNewScene adaptHow = iota // a scene of its own: heading, link
	howDraft                    // a scene drafted from it: heading, link, its lines as cues and dialogue, the rest as action
	howAddHere                  // the link at the cursor, in the cursor's scene
	howCut                      // marked as cut
	howSkip                     // left to do; on to the next
)

// adaptWhere is where a new scene goes.
type adaptWhere int

const (
	whereStory  adaptWhere = iota // in the story's order: after the scene adapting the passage before it
	whereCursor                   // at the cursor (after its paragraph)
	whereAfter                    // after the cursor's scene
	whereEnd                      // at the end of the script
)

// placeOption is a place a new scene can go: before line Line (the
// script's line count: at its end), told as Label, why in Reason.
type placeOption struct {
	Where  adaptWhere
	Line   int
	Label  string
	Note   string // which: "after Scene 4 · INT. HOUSE"
	Reason string // why it is likely ("" for the others)
}

// howOption is a way to adapt, and why it is likely ("" not more than
// another).
type howOption struct {
	How    adaptHow
	Label  string
	Note   string
	Off    bool // can't be done here (Add to the current scene outside a scene)
	Reason string
}

// adaptPlan is what Adapt… offers for passage P: the ways and places,
// the likely one of each first chosen, the headings the source suggests
// (the first the likely), the passage drafted.
type adaptPlan struct {
	P        int
	Hows     []howOption
	How      int // the chosen, in Hows
	Wheres   []placeOption
	Where    int // the chosen, in Wheres
	Headings []string
	Draft    string
}

// adaptCtx is what a plan is made from.
type adaptCtx struct {
	text     string
	starts   []string
	ps       []passage
	m        *sourceModel
	cursor   int      // runes
	labels   map[int]string
	headings []string // the headings the source suggests for the passage's scene, the likely first
	draft    string   // the passage drafted ("" none: not read yet)
	opens    string   // why the story starts a scene at the passage ("" it does not)
}

// runeLineOf is the line a rune offset is on.
func runeLineOf(text string, at int) int {
	r := []rune(text)
	return strings.Count(string(r[:max(0, min(at, len(r)))]), "\n")
}

// scenesLinking are the spans linking passage p, in the script's order.
func (m *sourceModel) scenesLinking(p int) []sceneSpan {
	var out []sceneSpan
	for _, l := range m.linked(p) {
		out = append(out, spanAt(m.spans, l))
	}
	return out
}

// isScene reports whether a span is a scene (its heading a scene
// heading), not a section or what comes before any heading.
func isScene(c adaptCtx, s sceneSpan) bool {
	return strings.HasPrefix(c.labels[s.Line], "Scene ")
}

// planAdapt is what Adapt… offers for passage p.
func planAdapt(c adaptCtx, p int) adaptPlan {
	pl := adaptPlan{P: p, Draft: c.draft, Headings: c.headings}
	if len(pl.Headings) == 0 {
		start, words := headingPlaceholder(c.starts)
		pl.Headings = []string{start + " " + words}
	}
	m := c.m
	line := runeLineOf(c.text, c.cursor)
	cur := spanAt(m.spans, line)
	inScene := isScene(c, cur) && line < scriptBodyLines(c.text)
	curLinks := map[int]bool{}
	for _, i := range m.inSpan(line) {
		if mt := m.matches[i]; mt.found() {
			for q := mt.Passages[0]; q <= mt.last(); q++ {
				curLinks[q] = true
			}
		}
	}
	prev := prevCounted(c.ps, p)

	// how: in the cursor's scene when the story's scene goes on from what
	// it adapts, or it adapts nothing yet; else a scene of its own
	add := howOption{How: howAddHere, Label: "Add to the current scene", Note: "the link at the cursor"}
	newScene := howOption{How: howNewScene, Label: "Start a new scene", Note: "a heading and the link"}
	draft := howOption{How: howDraft, Label: "Draft a scene from the passage", Note: "its lines as cues and dialogue, the rest as action, to edit"}
	if inScene {
		add.Note = "the link at the cursor, in " + c.labels[cur.Line]
	} else {
		add.Off, add.Note = true, "the cursor is not in a scene"
	}
	if c.draft == "" {
		draft.Off, draft.Note = true, "the story is still being read"
	}
	likelyAdd := false
	switch {
	case !inScene:
	case prev >= 0 && curLinks[prev] && c.opens == "":
		likelyAdd, add.Reason = true, "the story's scene goes on from “"+shortQuote(c.ps[prev].Plain)+"”, which this scene adapts"
	case len(curLinks) == 0:
		likelyAdd, add.Reason = true, c.labels[cur.Line]+" adapts no passage yet"
	}
	switch {
	case likelyAdd:
	case c.opens != "":
		newScene.Reason = "a scene starts here in the story: " + c.opens
	case inScene:
		newScene.Reason = c.labels[cur.Line] + " adapts other passages"
	default:
		newScene.Reason = "the cursor is not in a scene"
	}
	pl.Hows = []howOption{newScene, draft, add,
		{How: howCut, Label: "Mark as cut", Note: "left out of the script on purpose"},
		{How: howSkip, Label: "Skip for now", Note: "it stays to do; on to the next"}}
	if likelyAdd {
		pl.How = 2
	}

	// where: in the story's order first
	lines := scriptBodyLines(c.text)
	pl.Wheres = append(pl.Wheres, storyPlace(c, p))
	if line < lines && cur.Heading != "" { // not in the title page
		pl.Wheres = append(pl.Wheres, placeOption{Where: whereCursor, Line: afterParagraph(c.text, line), Label: "At the cursor",
			Note: "after the cursor's paragraph, line " + fmt.Sprint(line+1)})
		if cur.End < lines {
			pl.Wheres = append(pl.Wheres, placeOption{Where: whereAfter, Line: cur.End, Label: "After the cursor's scene",
				Note: c.labels[cur.Line] + spanHeading(cur)})
		}
	}
	pl.Wheres = append(pl.Wheres, placeOption{Where: whereEnd, Line: lines, Label: "At the end", Note: "after the last scene"})
	// one option a place: the first that names it
	var seen []int
	pl.Wheres = slices.DeleteFunc(pl.Wheres, func(o placeOption) bool {
		if slices.Contains(seen, o.Line) {
			return true
		}
		seen = append(seen, o.Line)
		return false
	})
	return pl
}

// spanHeading is a span's heading after its label: " · INT. HOUSE".
func spanHeading(s sceneSpan) string {
	if s.Heading == "" {
		return ""
	}
	return " · " + s.Heading
}

// scriptBodyLines is how many lines the script has before its blocks (the
// Adaptation, Source and notes blocks at its end): a scene put "at the
// end" goes before them.
func scriptBodyLines(text string) int {
	body := []rune(text)[:bodyEnd(text)]
	return strings.Count(strings.TrimRight(string(body), "\n"), "\n") + 1
}

// prevCounted is the passage to adapt just before p in the source (a
// heading between counts: the story's scene does not go on over it); -1
// none or a heading.
func prevCounted(ps []passage, p int) int {
	for i := p - 1; i >= 0; i-- {
		switch {
		case !ps[i].linkable():
			continue // a break
		case !ps[i].counted():
			return -1 // a heading
		}
		return i
	}
	return -1
}

// storyPlace is where a scene for passage p goes in the story's order:
// after the (last) scene adapting the nearest passage before it that a
// scene adapts; without one, before the first scene adapting a later
// one; and within its chapter's section, if the script has it. Without
// any, at the end.
func storyPlace(c adaptCtx, p int) placeOption {
	m := c.m
	lines := scriptBodyLines(c.text)
	o := placeOption{Where: whereStory, Line: lines, Label: "In the story's order"}
	found := false
	for q := p - 1; q >= 0 && !found; q-- {
		if sc := m.scenesLinking(q); len(sc) > 0 {
			last := sc[len(sc)-1]
			o.Line, found = min(last.End, lines), true
			o.Note = "after " + c.labels[last.Line] + spanHeading(last)
			o.Reason = "it adapts “" + shortQuote(c.ps[q].Plain) + "”, the passage before this one in the story"
			if len(sc) > 1 {
				o.Reason += " (the last of the " + fmt.Sprint(len(sc)) + " scenes that do)"
			}
		}
	}
	for q := p + 1; q < len(c.ps) && !found; q++ {
		if sc := m.scenesLinking(q); len(sc) > 0 {
			first := sc[0]
			o.Line, found = first.Line, true
			o.Note = "before " + c.labels[first.Line] + spanHeading(first)
			o.Reason = "it adapts “" + shortQuote(c.ps[q].Plain) + "”, later in the story"
		}
	}
	// in its chapter's section, if the script has it: at its start when
	// the scene to go by is before it, at its end when after it or none
	if ch := chapterOf(c.ps, p); ch >= 0 {
		if sec, end, ok := scriptSectionOf(c.text, c.starts, c.ps[ch]); ok && (!found || o.Line <= sec || o.Line > end) {
			before := found && o.Line <= sec
			o.Line = end
			if before {
				o.Line = spanAt(m.spans, sec).End // after the section's heading (and its synopsis)
			}
			o.Note = "in " + c.labels[sec]
			o.Reason = "the passage is in the story's " + c.ps[ch].Plain
		}
	}
	if !found && o.Note == "" {
		o.Note = "at the end: no scene adapts a passage near it"
	}
	return o
}

// scriptSectionOf finds the script's section for a source heading: its line,
// and the line its chapter ends at (the next section, or the end).
func scriptSectionOf(text string, starts []string, h passage) (line, end int, ok bool) {
	key := strings.Join(h.Words, " ")
	outline := outlineOf(text[:bodyEndByte(text)], starts)
	sec := slices.IndexFunc(outline, func(it outlineItem) bool {
		return !it.Scene && strings.Join(normWords(it.Text), " ") == key
	})
	if sec < 0 {
		return 0, 0, false
	}
	end = scriptBodyLines(text)
	for _, it := range outline[sec+1:] {
		if !it.Scene {
			end = it.Line
			break
		}
	}
	return outline[sec].Line, end, true
}

// afterParagraph is the line after the paragraph at line: the next blank
// line's (the line itself if blank).
func afterParagraph(text string, line int) int {
	ls := strings.Split(text, "\n")
	for i := line; i < len(ls); i++ {
		if strings.TrimSpace(ls[i]) == "" {
			return i
		}
	}
	return len(ls)
}

// sceneBlock is the new scene's text for passage p before line: its
// chapter's section first if the story enters a chapter there the script
// has no section for (another than that of the scene before it), the
// heading, the link, the drafted lines.
func sceneBlock(c adaptCtx, p, line int, heading, body string) string {
	var parts []string
	if ch := chapterOf(c.ps, p); ch >= 0 {
		if _, _, ok := scriptSectionOf(c.text, c.starts, c.ps[ch]); !ok {
			from := -1
			if line > 0 {
				from = c.m.position(line - 1)
			}
			if from < 0 || chapterOf(c.ps, from) != ch {
				parts = append(parts, strings.Repeat("#", max(c.ps[ch].Level, 1))+" "+c.ps[ch].Plain)
			}
		}
	}
	note, _ := linkNoteFor(matcherFor(c.ps), p, p)
	parts = append(parts, strings.TrimSpace(heading), note)
	if b := strings.TrimSpace(body); b != "" {
		parts = append(parts, b)
	}
	return strings.Join(parts, "\n\n")
}

// insertBlock is the edit putting block in before line (at the end of
// the script's body, before its blocks, for a line past it), with a
// blank line before and after it: an insertion only, nothing replaced
// (a shared script merges it as it is: docs/COLLAB.md).
func insertBlock(text string, line int, block string) sceneEdit {
	r := []rune(text)
	body := bodyEnd(text)
	lines := scriptBodyLines(text)
	if line < lines {
		at := 0 // line's start
		for i, l := 0, 0; i < len(r) && l < line; i++ {
			if r[i] == '\n' {
				l++
				at = i + 1
			}
		}
		ed := sceneEdit{At: at, Replace: at, Text: block + "\n\n"}
		if at >= 2 && r[at-2] != '\n' { // the line before is not blank
			ed.Text = "\n" + ed.Text
		}
		if at < len(r) && r[at] == '\n' { // the line there is blank already
			ed.Text = strings.TrimSuffix(ed.Text, "\n")
		}
		return ed
	}
	at := len([]rune(strings.TrimRight(string(r[:body]), "\n")))
	lead := "\n\n"
	if at == 0 {
		lead = ""
	}
	trailing := 0 // the newlines after the body, kept after the block
	for i := at; i < len(r) && r[i] == '\n'; i++ {
		trailing++
	}
	need := 2             // a blank line before the blocks
	if body == len(r) {
		need = 1 // the file's last line ended
	}
	ed := sceneEdit{At: at, Replace: at, Text: lead + block}
	if trailing < need {
		ed.Text += strings.Repeat("\n", need-trailing)
	}
	return ed
}
