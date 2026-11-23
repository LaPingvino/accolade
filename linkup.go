package main

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/prose"
)

// Linking up a free adaptation: a script written from its source before
// it had links (LOOP.md, 2026-11-23). Rules: (R1) nothing is written into
// the script from a guess: guesses live beside it, a faint "↗ link?" tag
// after a scene's heading and the source pane's section labels, until
// the writer accepts one (one undo step); (R2) quiet: only confident
// guesses show (scene_align.go stays silent otherwise), one cut offered
// at a time, no dialog full of suggestions unless asked for (Link Up the
// Script…).
//
// What the pane shows: the source in sections (its chapters, else the
// runs the scenes come from), each labelled with the scenes adapting it,
// "not adapted yet" after the last of them, where Next goes on from; a
// section skipped before or between adapted ones is offered as cut, one
// at a time.

// linkUp is the pane's guesses and what they were made from.
type linkUp struct {
	guesses []sceneGuess
	forText string
	forPS   []passage
	working bool   // a look under way
	again   bool   // the script changed meanwhile: look again when it is done
	then    func() // to do once the look is in (Link Up the Script…)

	// by line of the script as it is now: the guess at each scene heading
	byLine  map[int]sceneGuess
	lineFor string

	sections []sourceSection
	labelAt  map[int]string  // the section labels, by passage (the row they show on)
	asked    map[string]bool // sections offered as cut this session, by their first words

	checks []*widget.Check // Link Up the Script…'s, as shown
	apply  func()          // its Link Up
}

// sourceSection is a stretch of the source: a chapter (from its heading
// to the next), or a run of passages; the scenes adapting it, linked or
// guessed (their numbers, from 1).
type sourceSection struct {
	head        int // the passage its label goes on: its heading, or its first passage
	first, last int // its counted passages (first -1: none)
	scenes      []int
	linked      bool // one of its scenes is linked to it, not only guessed
	notYet      bool // after the last adapted section
	skipped     bool // before or between adapted sections, with passages still to do
	name        string
	toDo        int // its passages still to do
}

// sectionsOf are the source's sections: by its headings at the level
// its chapters are (the highest level found more than once), what comes
// before the first a section too; with no such headings, runs: those the
// scenes come from, and those between them.
func sectionsOf(ps []passage, st []passageStatus, gs []sceneGuess) []sourceSection {
	count := map[int]int{}
	for _, p := range ps {
		if p.Kind == prose.Heading && p.linkable() {
			count[p.Level]++
		}
	}
	level := 0
	for l := 1; l <= 6; l++ {
		if count[l] > 1 {
			level = l
			break
		}
	}
	var out []sourceSection
	add := func(head, from, to int, name string) {
		sec := sourceSection{head: head, first: -1, last: -1, name: name}
		for i := from; i <= to && i < len(ps); i++ {
			if ps[i].counted() {
				if sec.first < 0 {
					sec.first = i
				}
				sec.last = i
				if i < len(st) && st[i] == statusToDo {
					sec.toDo++
				}
			}
		}
		if sec.first >= 0 {
			if sec.head < 0 {
				sec.head = sec.first
			}
			out = append(out, sec)
		}
	}
	if level > 0 {
		start, head := 0, -1
		for i, p := range ps {
			if p.Kind == prose.Heading && p.Level <= level && p.linkable() {
				if head >= 0 || i > start {
					add(head, start, i-1, sectionName(ps, head, start))
				}
				start, head = i+1, i
			}
		}
		add(head, start, len(ps)-1, sectionName(ps, head, start))
	} else {
		// runs: each scene's stretch, and the stretches between
		var cuts []int // where a run starts
		for _, g := range gs {
			if g.First >= 0 {
				cuts = append(cuts, g.First, g.Last+1)
			}
		}
		cuts = append(cuts, 0, len(ps))
		slices.Sort(cuts)
		cuts = slices.Compact(cuts)
		for k := 0; k+1 < len(cuts); k++ {
			add(-1, cuts[k], cuts[k+1]-1, sectionName(ps, -1, cuts[k]))
		}
	}
	// the scenes in each
	adapted := -1
	for k := range out {
		sec := &out[k]
		for _, g := range gs {
			if g.First >= 0 && g.First <= sec.last && g.Last >= sec.first {
				sec.scenes = append(sec.scenes, g.Scene+1)
				sec.linked = sec.linked || g.Linked
			}
		}
		if len(sec.scenes) > 0 {
			adapted = k
		}
	}
	for k := range out {
		sec := &out[k]
		switch {
		case k > adapted:
			sec.notYet = adapted >= 0
		case len(sec.scenes) == 0:
			sec.skipped = sec.toDo > 0
		}
	}
	return out
}

// sectionName is how a section is named: its heading, else its first
// words.
func sectionName(ps []passage, head, start int) string {
	if head >= 0 {
		return shortQuote(ps[head].Plain)
	}
	for i := start; i < len(ps); i++ {
		if ps[i].counted() {
			return "“" + shortQuote(ps[i].Plain) + "”"
		}
	}
	return ""
}

// label is what a section's row says (empty: nothing).
func (sec sourceSection) label() string {
	switch {
	case len(sec.scenes) > 0:
		mark := "adapted"
		if !sec.linked {
			mark = "adapted?" // guessed only
		}
		return mark + " (" + sceneNumbers(sec.scenes) + ")"
	case sec.notYet:
		return "not adapted yet"
	}
	return ""
}

// sceneNumbers is "scene 4", "scenes 3–9", or "scenes 2, 5–6".
func sceneNumbers(ns []int) string {
	ns = slices.Compact(slices.Sorted(slices.Values(ns)))
	var parts []string
	for i := 0; i < len(ns); {
		j := i
		for j+1 < len(ns) && ns[j+1] == ns[j]+1 {
			j++
		}
		if j > i {
			parts = append(parts, strconv.Itoa(ns[i])+"–"+strconv.Itoa(ns[j]))
		} else {
			parts = append(parts, strconv.Itoa(ns[i]))
		}
		i = j + 1
	}
	word := "scenes "
	if len(ns) == 1 {
		word = "scene "
	}
	return word + strings.Join(parts, ", ")
}

// updateGuesses looks at the script against the source again when
// either changed, in the background: the guesses, the sections, the
// tags. Nothing is written.
func (s *sourcePane) updateGuesses() {
	if s.loading != nil || s.ref.Origin == sourceNone || !slices.ContainsFunc(s.passages, passage.counted) || s.w.markdown {
		if s.up.guesses != nil || s.up.labelAt != nil {
			s.up = linkUp{asked: s.up.asked}
			s.w.textEditor.RedrawMarks()
			s.syncActs()
			s.list.Refresh()
		}
		return
	}
	text, ps := s.w.textEditor.Text(), s.passages
	if sameString(text, s.up.forText) && samePassages(ps, s.up.forPS) {
		return
	}
	if s.up.working {
		s.up.again = true
		return
	}
	m := s.current()
	starts := s.w.sceneStarts()
	l := learnedOf(text)
	skipped := l.skippedFor(ps)
	s.up.working = true
	inBackground(func() []sceneGuess {
		gs := guessScenes(text, starts, m)
		for k := range gs { // Not This: not that guess again
			g := &gs[k]
			if g.guessed() && slices.ContainsFunc(skipped[wordsKey(g.Heading)], func(p int) bool { return p >= g.First && p <= g.Last }) {
				g.First, g.Last = -1, -1
			}
		}
		return gs
	}, func(gs []sceneGuess) {
		s.up.working = false
		if !samePassages(ps, s.passages) {
			s.updateGuesses()
			return
		}
		s.up.guesses, s.up.forText, s.up.forPS = gs, text, ps
		s.up.lineFor = ""
		s.up.sections = sectionsOf(ps, m.status, gs)
		labels := map[int]string{}
		for _, sec := range s.up.sections {
			if l := sec.label(); l != "" {
				labels[sec.head] = l
			}
		}
		if !maps.Equal(labels, s.up.labelAt) {
			s.up.labelAt = labels
			s.syncActs()
			s.list.Refresh()
		}
		s.w.textEditor.RedrawMarks() // the tags
		s.model = nil                // its anchors (guessEnd) again
		if s.up.again {
			s.up.again = false
			s.updateGuesses()
			return
		}
		if then := s.up.then; then != nil {
			s.up.then = nil
			then()
		}
		if s.shown() {
			s.showAnchor(s.current())
			s.offerCut()
		}
	})
}

// guessesByLine are the guesses by the line of their scene's heading in
// the script as it is now: the k-th scene still under the heading it was
// guessed for, else the first scene with that heading not taken.
func (s *sourcePane) guessesByLine() map[int]sceneGuess {
	text := s.w.textEditor.Text()
	if s.up.guesses == nil {
		return nil
	}
	if s.up.byLine != nil && sameString(s.up.lineFor, text) {
		return s.up.byLine
	}
	var heads []outlineItem
	for _, it := range outlineOf(text[:bodyEndByte(text)], s.w.sceneStarts()) {
		if it.Scene {
			heads = append(heads, it)
		}
	}
	out := map[int]sceneGuess{}
	taken := map[int]bool{}
	for _, g := range s.up.guesses {
		if g.First < 0 {
			continue
		}
		at := -1
		if g.Scene < len(heads) && heads[g.Scene].Text == g.Heading {
			at = g.Scene
		} else {
			at = slices.IndexFunc(heads, func(it outlineItem) bool { return it.Text == g.Heading && !taken[it.Line] })
		}
		if at >= 0 && !taken[heads[at].Line] {
			taken[heads[at].Line] = true
			out[heads[at].Line] = g
		}
	}
	s.up.byLine, s.up.lineFor = out, text
	return out
}

// guessTags are the editor's tags: "↗ link?" after each scene heading
// with a guess and no link yet.
func (w *MainWindow) guessTags() map[int]editor.Tag {
	if w.source == nil || w.markdown {
		return nil
	}
	s := w.source
	var out map[int]editor.Tag
	for line, g := range s.guessesByLine() {
		if !g.guessed() || g.Last >= len(s.passages) {
			continue
		}
		if out == nil {
			out = map[int]editor.Tag{}
		}
		out[line] = editor.Tag{Label: "↗ link?", Short: "↗?", Tip: s.guessTip(g)}
	}
	return out
}

// guessTip says what a guess is: where, and that it is one.
func (s *sourcePane) guessTip(g sceneGuess) string {
	q := "“" + shortQuote(s.passages[g.First].Plain) + "”"
	if g.Last > g.First {
		q += " … “" + shortQuote(s.passages[g.Last].Plain) + "”"
	}
	where := ""
	if ch := chapterOf(s.passages, g.First); ch >= 0 {
		where = " (" + shortQuote(s.passages[ch].Plain) + ")"
	}
	return "Reads like " + q + where + ": a guess, nothing is linked yet. Tap: Accept, Adjust or Not This"
}

// guessTagTapped opens a tag's menu: Accept, Adjust, Not This.
func (w *MainWindow) guessTagTapped(line int, t editor.Tag, at fyne.Position, cell fyne.Size) {
	if w.source == nil {
		return
	}
	g, ok := w.source.guessesByLine()[line]
	if !ok || !g.guessed() {
		return
	}
	s := w.source
	menu := fyne.NewMenu("",
		fyne.NewMenuItem("Accept", func() { s.acceptGuess(line, g) }),
		fyne.NewMenuItem("Adjust…", func() { s.adjustGuess(line, g) }),
		fyne.NewMenuItem("Not This", func() { s.notThisGuess(g) }))
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(w.textEditor).Add(at).AddXY(0, cell.Height)
	widget.ShowPopUpMenuAtPosition(menu, w.fyneWindow.Canvas(), pos)
	w.showEditorTip(t.Tip, at, cell)
}

// linkGuessHere is Adaptation > Link Guess for This Scene… (and vim's
// gs): the tag's menu of the cursor's scene, from the keyboard.
func (w *MainWindow) linkGuessHere() {
	if w.markdown || w.source == nil {
		w.setStatus("No guess here: the source pane guesses once it has the source")
		return
	}
	e := w.textEditor
	text := e.Text()
	cur, _ := e.Buffer().Position(e.CursorOffset())
	line := -1
	for _, it := range outlineOf(text[:bodyEndByte(text)], w.sceneStarts()) {
		if it.Scene && it.Line <= cur {
			line = it.Line
		}
	}
	g, ok := w.source.guessesByLine()[line]
	if line < 0 || !ok || !g.guessed() {
		w.setStatus("No guess for this scene")
		return
	}
	at, cell, shown := e.TagRect(line)
	if !shown { // folded away, or no room: the menu at the heading's start
		at, cell = fyne.Position{}, fyne.NewSize(0, 0)
	}
	w.guessTagTapped(line, editor.Tag{Label: "↗ link?", Short: "↗?", Tip: w.source.guessTip(g)}, at, cell)
}

// acceptGuess links the scene at line to its guess: the link on a line
// below its heading, one undo step.
func (s *sourcePane) acceptGuess(line int, g sceneGuess) {
	if g.Last >= len(s.passages) || !samePassages(s.passages, s.up.forPS) {
		return
	}
	e := s.w.textEditor
	text := e.Text()
	note, _ := linkNoteFor(matcherFor(s.passages), g.First, g.Last)
	b := e.Buffer()
	_, end := lineBounds([]rune(text), b.LineStart(line))
	at, insert, _ := insertLink(text, end, note, s.w.sceneStarts())
	e.Edit(func(b *buffer.Buffer) {
		b.Group(func() { b.Replace(at, at, insert) })
	})
	s.w.setStatus("Linked " + g.Heading + " (Undo takes it back)")
	s.follow()
}

// adjustGuess shows the guess in the source pane, the cursor on its
// scene: the writer picks the passage it adapts and links it (Link to
// Source Passage).
func (s *sourcePane) adjustGuess(line int, g sceneGuess) {
	w := s.w
	if !s.shown() {
		w.showPane("source", false)
	}
	s.jumping = true
	w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(line), false) })
	s.jumping = false
	s.setPicked(g.First)
	s.scrollToTop(max(g.First-1, 0))
	s.follow()
	w.setStatus("The guess is picked in the source: pick the passage this scene adapts, then Link to Source Passage")
}

// notThisGuess remembers a guess turned down (the adaptation block's
// skipped links:): it is not offered for that scene again.
func (s *sourcePane) notThisGuess(g sceneGuess) {
	if g.Last >= len(s.passages) {
		return
	}
	d, _ := fragmentFor(matcherFor(s.passages), g.First, g.Last)
	l := learnedOf(s.w.textEditor.Text())
	l.addSkip(g.Heading, cutOf(linkRef{Dir: d}))
	s.w.writeLearned(l, false)
	s.w.setStatus("Not offered for " + g.Heading + " again (Undo takes it back)")
	s.follow()
}

// guessEnd is, by the script's span lines, the last passage guessed for
// the scene there: where Next goes on from after a scene with no link.
func (s *sourcePane) guessEnd() map[int]int {
	out := map[int]int{}
	for line, g := range s.guessesByLine() {
		if g.guessed() {
			out[line] = g.Last
		}
	}
	return out
}

// offerCut asks about the first section skipped (before the first
// adapted one, or between two) not asked about yet: one at a time, the
// question's own line in the pane; nothing is written unless the writer
// says so.
func (s *sourcePane) offerCut() {
	if s.prompt.box.Visible() || s.adapting != nil {
		return
	}
	for _, sec := range s.up.sections {
		if !sec.skipped {
			continue
		}
		key := wordsKey(s.passages[sec.head].Plain) + "\x00" + strconv.Itoa(sec.toDo)
		if s.up.asked[key] {
			continue
		}
		if s.up.asked == nil {
			s.up.asked = map[string]bool{}
		}
		s.up.asked[key] = true
		text := fmt.Sprintf("%s (%s) isn't in the script. Mark as Cut?", sec.name, plural(sec.toDo, "paragraph"))
		sec := sec
		s.prompt.askSkip(text, "Mark as Cut", func() { s.cutSection(sec) }, "Next", func() {
			s.prompt.hide()
			s.offerCut()
		})
		return
	}
}

// cutSection marks a section's passages still to do as cut: one entry
// for the run (or one per run between the adapted ones in it), one undo
// step.
func (s *sourcePane) cutSection(sec sourceSection) {
	m := s.current()
	var runs [][2]int
	for i := sec.first; i <= sec.last && i < len(m.status); i++ {
		if m.status[i] != statusToDo {
			continue
		}
		if n := len(runs); n > 0 && runs[n-1][1] == prevCounted(s.passages, i) {
			runs[n-1][1] = i
		} else {
			runs = append(runs, [2]int{i, i})
		}
	}
	if len(runs) == 0 {
		return
	}
	cuts := slices.Clone(m.cuts)
	lm := matcherFor(s.passages)
	for _, r := range runs {
		d, _ := fragmentFor(lm, r[0], r[1])
		cuts = append(cuts, cutOf(linkRef{Dir: d}))
	}
	e := s.w.textEditor
	inside, _ := adaptationBlock(e.Text())
	start, end, block := withAdaptationBlock(e.Text(), withCuts(inside, cuts))
	e.Edit(func(b *buffer.Buffer) {
		cursor := b.Cursor()
		b.Group(func() { b.Replace(start, end, block) })
		if cursor > start {
			cursor = min(cursor, start)
		}
		b.SetCursor(cursor, false)
	})
	s.w.setStatus(sec.name + ": cut on purpose (Undo takes it back)")
	s.follow()
}

// linkUpScript is Adaptation > Link Up the Script…: the guesses and the
// skipped sections in one list to go through, each with its box: the
// scenes' links checked, the cuts not; Link Up writes what is checked,
// one undo step.
func (w *MainWindow) linkUpScript() {
	if w.markdown {
		return
	}
	s := w.sourcePane()
	if !s.shown() {
		w.showPane("source", false)
	}
	s.load()
	if s.ref.Origin == sourceNone {
		w.setStatus("No source: Adaptation > Choose Source…")
		return
	}
	s.up.then = nil
	s.updateGuesses()
	if s.up.working {
		w.setStatus("Reading the script against the source…")
		s.up.then = s.showLinkUp
		return
	}
	s.showLinkUp()
}

// showLinkUp is the Link Up dialog for the guesses as they are.
func (s *sourcePane) showLinkUp() {
	w := s.w
	type item struct {
		check *widget.Check
		line  int
		g     sceneGuess
		sec   *sourceSection
	}
	var items []item
	rows := container.NewVBox()
	byLine := s.guessesByLine()
	lines := make([]int, 0, len(byLine))
	for l := range byLine {
		lines = append(lines, l)
	}
	slices.Sort(lines)
	labels := spanLabelsFor(s)
	for _, l := range lines {
		g := byLine[l]
		if !g.guessed() {
			continue
		}
		q := "“" + shortQuote(s.passages[g.First].Plain) + "”"
		c := widget.NewCheck(labels[l]+": "+g.Heading+" ↗ "+q, nil)
		c.SetChecked(true)
		items = append(items, item{check: c, line: l, g: g})
		rows.Add(c)
	}
	for i := range s.up.sections {
		sec := &s.up.sections[i]
		if !sec.skipped {
			continue
		}
		c := widget.NewCheck("Mark as cut: "+sec.name+" ("+plural(sec.toDo, "paragraph")+")", nil)
		items = append(items, item{check: c, sec: sec})
		rows.Add(c)
	}
	if len(items) == 0 {
		w.setStatus("Link Up: no scene without a link reads surely enough like a part of the source")
		return
	}
	intro := widget.NewLabel("Guesses from the words and the scenes' order: rough. Nothing is written until Link Up; Undo takes it all back at once.")
	intro.Wrapping = fyne.TextWrapWord
	content := container.NewBorder(intro, nil, nil, nil, container.NewVScroll(rows))
	apply := func(ok bool) {
		if !ok {
			return
		}
		var links []item
		var cuts []sourceSection
		for _, it := range items {
			if !it.check.Checked {
				continue
			}
			if it.sec != nil {
				cuts = append(cuts, *it.sec)
			} else {
				links = append(links, it)
			}
		}
		gs := make([]sceneGuess, len(links))
		ls := make([]int, len(links))
		for k, it := range links {
			gs[k], ls[k] = it.g, it.line
		}
		s.linkUpWrite(ls, gs, cuts)
	}
	d := dialog.NewCustomConfirm("Link Up the Script", "Link Up", "Cancel", content, apply, w.fyneWindow)
	s.linkUpDialog = d
	s.up.checks, s.up.apply = nil, func() { d.Hide(); apply(true) }
	for _, it := range items {
		s.up.checks = append(s.up.checks, it.check)
	}
	d.Resize(fitDialog(w.fyneWindow, 620, 700))
	d.Show()
}

// linkUpWrite writes the links of the scenes at lines (their guesses gs)
// and the cuts of sections: one undo step.
func (s *sourcePane) linkUpWrite(lines []int, gs []sceneGuess, cuts []sourceSection) {
	if len(lines)+len(cuts) == 0 {
		return
	}
	e := s.w.textEditor
	starts := s.w.sceneStarts()
	lm := matcherFor(s.passages)
	m := s.current()
	allCuts := slices.Clone(m.cuts)
	for _, sec := range cuts {
		for i := sec.first; i <= sec.last && i < len(m.status); {
			if m.status[i] != statusToDo {
				i++
				continue
			}
			j := i
			for k := i + 1; k <= sec.last && k < len(m.status) && m.status[k] != statusAdapted; k++ {
				if m.status[k] == statusToDo {
					j = k
				}
			}
			d, _ := fragmentFor(lm, i, j)
			allCuts = append(allCuts, cutOf(linkRef{Dir: d}))
			i = j + 1
		}
	}
	order := make([]int, len(lines)) // bottom up: the earlier offsets stay
	for k := range order {
		order[k] = k
	}
	slices.SortFunc(order, func(a, b int) int { return lines[b] - lines[a] })
	e.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			for _, k := range order {
				text := b.Text()
				note, _ := linkNoteFor(lm, gs[k].First, gs[k].Last)
				_, end := lineBounds([]rune(text), b.LineStart(lines[k]))
				at, insert, _ := insertLink(text, end, note, starts)
				b.Replace(at, at, insert)
			}
			if len(cuts) > 0 {
				text := b.Text()
				inside, _ := adaptationBlock(text)
				start, end, block := withAdaptationBlock(text, withCuts(inside, allCuts))
				b.Replace(start, end, block)
			}
		})
	})
	say := "Linked up: " + plural(len(lines), "scene")
	if len(cuts) > 0 {
		say += ", " + plural(len(cuts), "section") + " cut"
	}
	s.w.setStatus(say + " (Undo takes it back)")
	s.follow()
}
