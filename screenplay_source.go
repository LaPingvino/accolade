package main

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/accolade/internal/scriptdiff"
	"github.com/LaPingvino/accolade/internal/structure"
)

// Screenplay sources (LOOP.md, SCREENPLAY SOURCES, Joop 2026-11-23: "We
// might want to be able to recognize screenplay sources as well, for
// fix-up work"). A script converted from another (a PDF, an FDX, a
// Fountain from another tool) has the one it was made from as its
// source: Choose Source… reads it as a screenplay, and the Source pane,
// seeing a screenplay's shape (headings, cues, speeches), compares it
// with the script scene to scene and element to element instead of
// showing passages to adapt. The pane follows the cursor: the scene being
// fixed shows what the source has otherwise, each difference with Take
// the Source's (one undo step) and Keep Mine; Next and Previous go to the
// scenes that differ; a line sums it up ("18 scenes match, 3 differ, 1
// missing"). Nothing is written without the writer's choice.
//
// One engine with Merge Changes (merge.go, internal/scriptdiff): the
// script is the base and ours, the source theirs, so every difference is
// the source's. The source's print is read without what a page leaves in
// its text (page numbers, (MORE), (CONT'D), CONTINUED, words broken by a
// hyphen at a line's end); the script's own such debris is a difference,
// to fix. A speech or paragraph wrapped over lines is none, and the
// script's own outline, notes and links are left out, as Merge Changes
// does.

// screenplayShaped reports whether a source's text reads as a
// screenplay: scene headings and speeches, not a story's paragraphs.
func screenplayShaped(text string, starts []string) bool {
	if text == "" {
		return false
	}
	lines := strings.Split(text, "\n")
	if len(lines) > 1500 { // a long novel: its opening tells
		lines = lines[:1500]
	}
	var headings, cues, speech, other int
	for i, k := range syntax.ClassifyIn(lines, starts, dialectForStarts(starts)) {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		switch k {
		case syntax.SceneHeading:
			headings++
		case syntax.Character:
			cues++
		case syntax.Dialogue, syntax.Parenthetical:
			speech++
		case syntax.Action, syntax.Transition, syntax.Centered:
			other++
		}
	}
	total := headings + cues + speech + other
	return headings >= 2 && cues >= 3 && total > 0 && (headings+cues+speech)*4 >= total
}

// What a conversion leaves in the text that is no part of the script.
var (
	debrisLineRe = regexp.MustCompile(`(?i)^(\(?more\)?|\(?continued\)?:?|cont(inued|'d|’d)\.?|-?\s*\d+[a-z]?\s*-?\.?|page \d+( of \d+)?)$`)
	contdRe      = regexp.MustCompile(`(?i)\s*\((cont'd|cont’d|contd|cont\.|continued|continuing)\)`)
	hyphenRe     = regexp.MustCompile(`(\p{L})-\n(\p{Ll})`)
)

// withoutDebris is a source's text without what its pages left in it:
// page numbers, (MORE) and CONTINUED lines gone, (CONT'D) off a cue,
// words broken over a line joined.
func withoutDebris(s string) string {
	s = hyphenRe.ReplaceAllString(s, "$1$2")
	lines := strings.Split(s, "\n")
	out := lines[:0]
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if debrisLineRe.MatchString(t) {
			continue
		}
		if c := contdRe.ReplaceAllString(l, ""); c != l && strings.ToUpper(c) == c {
			l = c // a cue's
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}

// sourceKey is an element as the comparison sees it: as Merge Changes
// does (what the source cannot carry left out, spaces as one), a
// speech's or paragraph's wrapped lines as one, Fountain's forcing marks
// off, a heading without its scene number, and case as one: a print
// sets headings, action or a whole page in capitals where a conversion
// does not ("1930s", "1930S"), and what is looked for is the words.
func sourceKey(c mergeCarried, heading func(string) bool) func(string) string {
	base := mergeKey(c, false)
	return func(s string) string {
		k := base(s)
		if k == "" {
			return ""
		}
		lines := strings.Split(k, "\n")
		if heading != nil && heading(lines[0]) { // a printed source numbers its scenes; a conversion may not
			lines[0] = mergeNumberRe.ReplaceAllString(lines[0], "")
		}
		for i, l := range lines {
			if len(l) > 1 && (l[0] == '@' || l[0] == '!') {
				lines[i] = l[1:]
			} else if strings.HasPrefix(l, ">") && !strings.HasSuffix(l, "<") {
				lines[i] = strings.TrimSpace(l[1:])
			}
		}
		if len(lines) > 2 {
			lines = []string{lines[0], strings.Join(lines[1:], " ")} // the cue (or first line), then the rest as one
		}
		return strings.ToLower(strings.Join(lines, "\n"))
	}
}

// sourceOptions are how the script and its screenplay source are
// compared.
func (w *MainWindow) sourceOptions(src string) scriptdiff.MergeOptions {
	o := w.mergeOptions(src, false)
	h := o.IsHeading
	o.IsHeading = func(l string) bool { return h(l) || forcedHeading(l) } // a stage play's ".ACT I SCENE II" too
	c := carriedBy(src, false, o.IsHeading)
	o.Key = sourceKey(c, o.IsHeading) // Carry: Merge Changes' (the script's links, notes and locked number kept)
	return o
}

// forcedHeading: a line Fountain forces to be a scene heading (".BRIDGE",
// not "...").
func forcedHeading(l string) bool {
	return len(l) > 1 && l[0] == '.' && l[1] != '.' && l[1] != ' '
}

// screenplayCompare is the script compared with its screenplay source.
type screenplayCompare struct {
	text, src string // what was compared
	m         *scriptdiff.Merge
	at        []int // each change's line in the script: where taking it writes
	heading   func(string) bool
	scenes    int // the script's scenes
}

// compareWithSource compares text with the screenplay src, read without
// its pages' debris.
func compareWithSource(text, src string, o scriptdiff.MergeOptions) *screenplayCompare {
	m := scriptdiff.NewMerge(text, text, withoutDebris(src), o)
	sc := &screenplayCompare{text: text, src: src, m: m, heading: o.IsHeading}
	lines := strings.Split(text, "\n")
	for _, c := range m.Changes {
		after, _, _ := m.Apply(map[int]bool{c.ID: true}, nil)
		sc.at = append(sc.at, firstDifferentLine(lines, after))
	}
	for _, e := range scriptdiff.Split(text, o.IsHeading) {
		if e.Heading && !strings.HasPrefix(e.Text, "#") {
			sc.scenes++
		}
	}
	return sc
}

// firstDifferentLine is the first line where after differs from lines.
func firstDifferentLine(lines []string, after string) int {
	i := 0
	for after != "" && i < len(lines) {
		l, rest, more := strings.Cut(after, "\n")
		if l != lines[i] {
			break
		}
		i++
		if !more {
			break
		}
		after = rest
	}
	return min(i, max(len(lines)-1, 0))
}

// changeKey is a change as Keep Mine remembers it: by what it says, so the
// choice holds while the script changes elsewhere.
func changeKey(c scriptdiff.Change) string {
	return fmt.Sprint(c.Kind, "\x00", c.Row, "\x00", c.Ours, "\x00", c.Theirs)
}

// open are the changes not kept as the writer's, in the script's order.
func (sc *screenplayCompare) open(kept map[string]bool) []int {
	var ids []int
	for _, c := range sc.m.Changes {
		if !kept[changeKey(c)] {
			ids = append(ids, c.ID)
		}
	}
	slices.SortStableFunc(ids, func(a, b int) int { return sc.at[a] - sc.at[b] })
	return ids
}

// sceneSpan is the cursor's scene's lines: [from, to), and its heading.
func sceneSpanOf(lines []string, line int, heading func(string) bool) (from, to int, head string) {
	from = 0
	for i := min(line, len(lines)-1); i >= 0; i-- {
		if heading(lines[i]) {
			from, head = i, lines[i]
			break
		}
	}
	to = len(lines)
	for i := max(line, from) + 1; i < len(lines); i++ {
		if heading(lines[i]) {
			to = i
			break
		}
	}
	return from, to, head
}

// summary is the comparison in a line.
func (sc *screenplayCompare) summary(kept map[string]bool) string {
	differ := map[int]bool{}
	sceneOnly := func(l string) bool { return sc.heading(l) && !strings.HasPrefix(l, "#") }
	var missing, extra, moved, keptN int
	lines := strings.Split(sc.text, "\n")
	for _, c := range sc.m.Changes {
		if kept[changeKey(c)] {
			keptN++
			continue
		}
		switch c.Kind {
		case scriptdiff.SceneAdded:
			missing++
		case scriptdiff.SceneCut:
			extra++
		case scriptdiff.SceneMoved:
			moved++
		default: // counted by its scene (not a section, nor the title page)
			if from, _, head := sceneSpanOf(lines, sc.at[c.ID], sceneOnly); head != "" {
				differ[from] = true
			}
		}
	}
	match := max(sc.scenes-len(differ)-extra-moved, 0)
	verb := func(n int, one, many string) string {
		if n == 1 {
			return one
		}
		return many
	}
	parts := []string{plural(match, "scene") + verb(match, " matches", " match")}
	if len(differ) > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", len(differ), verb(len(differ), "differs", "differ")))
	}
	if missing > 0 {
		parts = append(parts, fmt.Sprintf("%d missing", missing))
	}
	if extra > 0 {
		parts = append(parts, fmt.Sprintf("%d only in yours", extra))
	}
	if moved > 0 {
		parts = append(parts, fmt.Sprintf("%d moved", moved))
	}
	if keptN > 0 {
		parts = append(parts, plural(keptN, "difference")+" kept as yours")
	}
	return strings.Join(parts, " · ")
}

// sourceWhat says what a difference is, the source's way round.
func sourceWhat(c scriptdiff.Change) string {
	switch c.Kind {
	case scriptdiff.SceneCut:
		return "This scene is not in the source:"
	case scriptdiff.SceneAdded:
		return "The source has a scene here that yours has not:"
	case scriptdiff.SceneMoved:
		return "The source has this scene elsewhere"
	}
	switch c.Row {
	case scriptdiff.Removed:
		return "Only in yours:"
	case scriptdiff.Added:
		return "The source has, yours not:"
	}
	return "The source reads:"
}

// The pane's side of it (source_pane.go shows this body for a screenplay).

// screenplayView is the Source pane's body for a screenplay source.
type screenplayView struct {
	s       *sourcePane
	cmp     *screenplayCompare
	working bool
	again   bool
	kept    map[string]bool // Keep Mine, for this source while the window is open
	keptFor string          // the source they are for
	shown   string          // what the rows show (their key)
	rows    *fyne.Container
	box     fyne.CanvasObject
}

func newScreenplayView(s *sourcePane) *screenplayView {
	v := &screenplayView{s: s, kept: map[string]bool{}}
	v.rows = container.NewVBox()
	v.box = container.NewVScroll(v.rows)
	return v
}

// follow compares again if the script or the source changed (in the
// background: a feature's worth of scenes takes a moment), then shows
// the cursor's scene.
func (v *screenplayView) follow() {
	s, w := v.s, v.s.w
	if v.keptFor != s.src {
		v.kept, v.keptFor = map[string]bool{}, s.src
	}
	text := w.textEditor.Text()
	if v.cmp == nil || !sameString(v.cmp.text, text) || !sameString(v.cmp.src, s.src) {
		if v.working {
			v.again = true
			return
		}
		v.working = true
		src, o := s.src, w.sourceOptions(s.src)
		inBackground(func() *screenplayCompare { return compareWithSource(text, src, o) }, func(sc *screenplayCompare) {
			v.working = false
			v.cmp = sc
			if v.again {
				v.again = false
				v.follow()
				return
			}
			v.show()
		})
		if v.cmp == nil || v.working {
			return
		}
	}
	v.show()
}

// current is the comparison if it is of the script as it is now.
func (v *screenplayView) current() *screenplayCompare {
	if v.cmp == nil || !sameString(v.cmp.text, v.s.w.textEditor.Text()) || !sameString(v.cmp.src, v.s.src) {
		return nil
	}
	return v.cmp
}

// inScene are the open differences in the scene of line, its span and
// heading.
func (v *screenplayView) inScene(sc *screenplayCompare, line int) (ids []int, head string) {
	lines := strings.Split(sc.text, "\n")
	from, to, head := sceneSpanOf(lines, line, sc.heading)
	for _, id := range sc.open(v.kept) {
		if at := sc.at[id]; at >= from && (at < to || to == len(lines) && at == to-1) {
			ids = append(ids, id)
		}
	}
	return ids, head
}

// show puts the cursor's scene's differences in the pane.
func (v *screenplayView) show() {
	s, w := v.s, v.s.w
	sc := v.current()
	if sc == nil {
		return
	}
	s.count.SetText(sc.summary(v.kept))
	line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
	ids, head := v.inScene(sc, line)
	where := strings.TrimSpace(head)
	if where == "" {
		where = "Before the first scene"
	}
	open := len(sc.open(v.kept))
	switch {
	case open == 0:
		s.setInfo("The script matches its source, scene for scene.")
	case len(ids) == 0:
		s.setInfo(where + ": matches the source. Next goes to the next scene that differs.")
	default:
		s.setInfo(where + ": " + plural(len(sc.hunks(ids)), "difference") + " from the source")
	}
	key := fmt.Sprintf("%p %v %d", sc, ids, len(v.kept))
	if key == v.shown && len(v.rows.Objects) > 0 {
		return
	}
	v.shown = key
	hs := sc.hunks(ids)
	var objs []fyne.CanvasObject
	if len(hs) > 1 {
		all := widget.NewButtonWithIcon("Take All in This Scene", theme.ContentPasteIcon(), func() { v.take(ids...) })
		all.Importance = widget.LowImportance
		objs = append(objs, all)
	}
	for i, h := range hs {
		if i == maxSourceRows {
			objs = append(objs, muted(fmt.Sprintf("…and %s more here: take or keep these first, or Take All", plural(len(hs)-i, "difference"))))
			break
		}
		objs = append(objs, v.row(sc, h), widget.NewSeparator())
	}
	v.rows.Objects = objs
	v.rows.Refresh()
}

// maxSourceRows is how many differences a scene shows at once: a badly
// read source can differ everywhere, and a long list helps nobody.
const maxSourceRows = 12

// hunks groups ids (in the script's order) into the differences as
// shown: a paragraph or speech the script has in two pieces where the
// source has one (or the other way round) is one difference, shown and
// taken whole, never half (taking one piece alone would write its words
// twice). Pieces are told by touching (no element between them) and by
// words: the piece only one side has is in the other side's neighbour,
// word for word.
func (sc *screenplayCompare) hunks(ids []int) [][]int {
	var hs [][]int
	end := -1 // the line after the last difference's text in the script, and its blank line
	for _, id := range ids {
		c := sc.m.Changes[id]
		at := sc.at[id]
		if len(hs) > 0 && at <= end && sc.piece(hs[len(hs)-1], c) {
			hs[len(hs)-1] = append(hs[len(hs)-1], id)
		} else {
			hs = append(hs, []int{id})
		}
		end = at + linesOf(c.Ours) + 1
	}
	return hs
}

// piece reports whether c, touching the difference h, is a piece of it:
// c is only on one side (a piece cut off, or one the other side joined)
// and its words are in h's text on the other side, word for word; or h
// is such a piece of c.
func (sc *screenplayCompare) piece(h []int, c scriptdiff.Change) bool {
	if c.Kind != scriptdiff.ChangeEdit || sc.m.Changes[h[0]].Kind != scriptdiff.ChangeEdit {
		return false
	}
	norm := func(s string) string { return strings.Join(strings.Fields(strings.ToLower(s)), " ") }
	in := func(piece, whole string) bool {
		p := norm(piece)
		return p != "" && strings.Contains(norm(whole), p)
	}
	ours, theirs := sc.joined(h)
	switch {
	case c.Row == scriptdiff.Removed:
		return in(c.Ours, theirs) || ours == "" && in(theirs, c.Ours)
	case c.Row == scriptdiff.Added:
		return in(c.Theirs, ours) || theirs == "" && in(ours, c.Theirs)
	}
	// a changed one after pieces only one side has: theirs held them
	return theirs == "" && in(ours, c.Theirs) || ours == "" && in(theirs, c.Ours)
}

// linesOf is how many lines a text holds ("" none).
func linesOf(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// joined are a hunk's texts, the script's and the source's, elements
// with a blank line between.
func (sc *screenplayCompare) joined(h []int) (ours, theirs string) {
	var o, t []string
	for _, id := range h {
		c := sc.m.Changes[id]
		if c.Ours != "" {
			o = append(o, c.Ours)
		}
		if c.Theirs != "" {
			t = append(t, c.Theirs)
		}
	}
	return strings.Join(o, "\n\n"), strings.Join(t, "\n\n")
}

// row is a difference as shown (a hunk: one or more that touch): what
// it is, the words, the two choices.
func (v *screenplayView) row(sc *screenplayCompare, h []int) fyne.CanvasObject {
	c := sc.m.Changes[h[0]]
	what := sourceWhat(c)
	var text fyne.CanvasObject
	ours, theirs := sc.joined(h)
	switch {
	case len(h) > 1:
		what = "The source reads:"
		if strings.Count(ours, "\n\n") != strings.Count(theirs, "\n\n") {
			what = "The source has this in other pieces:"
		}
		text = diffRow(scriptdiff.Row{Kind: scriptdiff.Changed, Old: ours, New: theirs, Words: scriptdiff.Words(ours, theirs)}, false)
	case c.Kind == scriptdiff.SceneMoved:
	case c.Kind == scriptdiff.SceneCut || c.Row == scriptdiff.Removed:
		text = diffRow(scriptdiff.Row{Kind: scriptdiff.Removed, Old: c.Ours}, false)
	case c.Kind == scriptdiff.SceneAdded || c.Row == scriptdiff.Added:
		text = diffRow(scriptdiff.Row{Kind: scriptdiff.Added, New: c.Theirs}, false)
	default:
		text = diffRow(scriptdiff.Row{Kind: scriptdiff.Changed, Old: c.Ours, New: c.Theirs, Words: c.Words}, false)
	}
	take := widget.NewButtonWithIcon("Take the Source's", theme.ConfirmIcon(), func() { v.take(h...) })
	take.Importance = widget.HighImportance
	fyne.SetAccessibleLabel(take, "Take the source's: "+strings.TrimSuffix(what, ":"))
	keep := widget.NewButtonWithIcon("Keep Mine", theme.CancelIcon(), func() { v.keepMine(sc, h) })
	keep.Importance = widget.LowImportance
	box := container.NewVBox(muted(what))
	if text != nil {
		box.Add(text)
	}
	box.Add(container.NewHBox(take, keep))
	return box
}

// take writes the source's form of the differences ids into the script,
// one undo step.
func (v *screenplayView) take(ids ...int) {
	w := v.s.w
	sc := v.current()
	if sc == nil || len(ids) == 0 {
		w.setStatus("The script changed since it was compared: a moment, it is compared again")
		v.follow()
		return
	}
	pick := map[int]bool{}
	for _, id := range ids {
		pick[id] = true
	}
	after, lineMap, pos := sc.m.Apply(pick, nil)
	if after == sc.text {
		return
	}
	first := len(lineMap)
	for _, id := range ids {
		first = min(first, sc.at[id])
	}
	lines := strings.Split(after, "\n")
	first = max(min(first, len(lines)-1), 0)
	msg := "Took the source's"
	if len(ids) > 1 {
		msg = "Took the source's " + plural(len(ids), "difference")
	}
	cur, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
	w.applyStruct(sc.text, structure.Result{Lines: lines, Map: lineMap, Pos: pos, First: first, Last: first, Msg: msg}, false, cur, 0)
	v.follow()
}

// keepMine leaves a difference (its hunk) as the script has it: not
// shown again while the window has this source.
func (v *screenplayView) keepMine(sc *screenplayCompare, h []int) {
	for _, id := range h {
		v.kept[changeKey(sc.m.Changes[id])] = true
	}
	v.s.w.setStatus("Kept yours")
	v.show()
}

// step goes to the next (dir 1) or previous scene with a difference
// from the source, round from the end.
func (v *screenplayView) step(dir int) {
	w := v.s.w
	sc := v.current()
	if sc == nil {
		w.setStatus("Comparing with the source…")
		v.follow()
		return
	}
	open := sc.open(v.kept)
	if len(open) == 0 {
		w.setStatus("The script matches its source: nothing left to fix")
		return
	}
	lines := strings.Split(sc.text, "\n")
	line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
	from, to, _ := sceneSpanOf(lines, line, sc.heading)
	target := -1
	if dir > 0 {
		for _, id := range open {
			if sc.at[id] >= to {
				target = sc.at[id]
				break
			}
		}
		if target < 0 {
			target = sc.at[open[0]]
			w.setStatus("From the start: the first scene that differs from the source")
		}
	} else {
		for i := len(open) - 1; i >= 0; i-- {
			if sc.at[open[i]] < from {
				target = sc.at[open[i]]
				break
			}
		}
		if target < 0 {
			target = sc.at[open[len(open)-1]]
			w.setStatus("From the end: the last scene that differs from the source")
		}
	}
	target = min(target, len(lines)-1)
	w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(target), false) })
	v.follow()
}

// useScreenplay shows a source read that is a screenplay: no passages
// (nothing to adapt, link or guess), the comparison instead.
func (s *sourcePane) useScreenplay(r *sourceRead) {
	if r.ref != s.ref {
		s.up, s.seedAsked = linkUp{}, false
		s.base.session = sessionRead{}
	}
	s.prompt.hide()
	s.ref, s.modTime, s.size, s.src, s.err = r.ref, r.modTime, r.size, r.src, r.err
	s.screenplay = true
	s.passages = nil
	s.model, s.modelFor, s.matcher = nil, "", nil
	s.reading = nil
	s.selected, s.selWords, s.pickFresh = -1, nil, false
	s.offered, s.offeredWords = -1, nil
	s.linkedBy, s.statuses, s.full = nil, nil, nil
	s.lit, s.doubt = map[int]bool{}, map[int]bool{}
	s.showBody()
	s.showReading(false)
}

// followScreenplay is the pane following the cursor with a screenplay
// source: its tools say what Next does here, the comparison follows.
func (s *sourcePane) followScreenplay() {
	s.updateGuesses() // none: no passages (any shown before go)
	s.learned.box.Hide()
	s.learned.ok = false
	s.prompt.hide()
	s.anchorLine.SetText("Next and Previous go to the scenes that differ from the source")
	s.adapt.Hide()
	s.next.Enable()
	s.prev.Enable()
	fyne.SetAccessibleLabel(s.next, "Next scene that differs from the source")
	fyne.SetAccessibleLabel(s.prev, "Previous scene that differs from the source")
	s.sp.follow()
}
