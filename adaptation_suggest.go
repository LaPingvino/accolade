package main

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/pace"
	"github.com/LaPingvino/accolade/internal/storyread"
)

// Suggested links: a script written from its source before it had one
// (or before Accolade linked them) gets its links offered scene by
// scene, never put in by itself. A scene with no link reads like a
// passage when its dialogue says a line the story's reading
// (internal/storyread) finds in the passage, or when its words (heading,
// action, dialogue) hold most of the passage's, the rarer words counting
// more. The pane asks per scene: Link, Skip, or Not Now.

// suggestMin is how alike a scene and a passage are at least (0–1) for
// a link to be suggested.
const suggestMin = 0.5

// commonFrom is how many passages a source has before its commonest
// words are left out of the matching.
const commonFrom = 50

// suggestPerScene is how many passages a scene is offered at most.
const suggestPerScene = 3

// linkSuggestion is a scene of the script and the passages it reads like.
type linkSuggestion struct {
	Scene    int // which of the script's scenes (0: its first heading)
	Heading  string
	Passages []int
}

// sceneBody is a scene's words: what its people say, and the rest.
type sceneBody struct {
	Line     int
	Heading  string
	Dialogue [][]string // each speech, normalised words
	Words    map[string]bool
	Linked   bool // a link is in it already
}

// scenesOf reads the script's scenes for their words: the heading, the
// action and the dialogue, notes and boneyard left out.
func scenesOf(text string, starts []string) []sceneBody {
	body := text[:bodyEndByte(text)]
	lines := strings.Split(body, "\n")
	outline := outlineOf(body, starts)
	links := parseLinks(body)
	var out []sceneBody
	for k, it := range outline {
		if !it.Scene {
			continue
		}
		end := len(lines)
		if k+1 < len(outline) {
			end = outline[k+1].Line
		}
		sc := sceneBody{Line: it.Line, Heading: it.Text, Words: map[string]bool{}}
		for _, l := range links {
			if l.Line >= it.Line && l.Line < end {
				sc.Linked = true
			}
		}
		add := func(ws []string) {
			for _, w := range ws {
				sc.Words[w] = true
			}
		}
		add(normWords(it.Text))
		speaking, blank := false, true
		for _, raw := range lines[min(it.Line+1, end):end] {
			line := strings.TrimSpace(stripNotes(raw))
			switch {
			case line == "":
				speaking, blank = false, true
				continue
			case strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "*/") || strings.HasPrefix(line, ">") ||
				strings.HasSuffix(line, "TO:") || strings.HasPrefix(line, "="):
			case blank && isCharacterCue(line):
				speaking = true
			case speaking && strings.HasPrefix(line, "("): // a parenthetical
			case speaking:
				ws := normWords(line)
				sc.Dialogue = append(sc.Dialogue, ws)
				add(ws)
			default:
				add(normWords(line))
			}
			blank = false
		}
		out = append(out, sc)
	}
	return out
}

// stripNotes is a line without its [[notes]].
func stripNotes(line string) string {
	for {
		i := strings.Index(line, "[[")
		if i < 0 {
			return line
		}
		j := strings.Index(line[i:], "]]")
		if j < 0 {
			return line[:i]
		}
		line = line[:i] + " " + line[i+j+2:]
	}
}

// suggestLinks are the links a script could have to its source: for each
// scene with none, the passages still to do (status) it reads like, the
// best first, each passage offered to one scene only (the one it fits
// best); r is the source's reading (its passages' dialogue lines).
//
// skipped are the passages the writer skipped for a scene, by its
// heading (wordsKey): not offered to it again.
func suggestLinks(text string, starts []string, ps []passage, status []passageStatus, r *storyread.Reading, skipped map[string][]int) []linkSuggestion {
	scenes := scenesOf(text, starts)
	// the passages to do, their words, and how rare each word is
	df := map[string]int{}
	index := map[string][]int{}
	words := make([]map[string]bool, len(ps))
	n := 0
	for i, p := range ps {
		if i >= len(status) || status[i] != statusToDo {
			continue
		}
		n++
		words[i] = map[string]bool{}
		for _, w := range p.Words {
			if len([]rune(w)) > 2 && !words[i][w] {
				words[i][w] = true
				df[w]++
				index[w] = append(index[w], i)
			}
		}
	}
	if n == 0 || len(scenes) == 0 {
		return nil
	}
	if n >= commonFrom { // a long source: a word in many of its passages tells none apart
		for w, d := range df {
			if d > n/4 {
				for _, i := range index[w] {
					delete(words[i], w)
				}
				delete(index, w)
			}
		}
	}
	weight := func(w string) float64 { return math.Log(1 + float64(n)/float64(max(df[w], 1))) }
	total := make([]float64, len(ps))
	for i, ws := range words {
		for w := range ws {
			total[i] += weight(w)
		}
	}
	type fit struct {
		scene, passage int
		score          float64
	}
	var fits []fit
	shared := make([]float64, len(ps)) // a scene's words in each passage, weighed
	var touched []int
	for k, sc := range scenes {
		if sc.Linked {
			continue
		}
		skip := skipped[wordsKey(sc.Heading)]
		touched = touched[:0]
		for w := range sc.Words {
			wt := weight(w)
			for _, i := range index[w] {
				if shared[i] == 0 {
					touched = append(touched, i)
				}
				shared[i] += wt
			}
			pace.Yield() // the browser draws between
		}
		for _, i := range touched {
			sum := shared[i]
			shared[i] = 0
			pace.Yield()
			score := sum / max(total[i], 1e-9)
			if r != nil && i < len(r.Passages) { // a line of the passage's said in the scene
				for _, l := range r.Passages[i].Lines {
					lw := normWords(l.Text)
					if len(lw) < 2 {
						continue
					}
					for _, d := range sc.Dialogue {
						if o := overlap(lw, d); o >= 0.6 {
							score = max(score, 0.5+0.5*o)
						}
					}
				}
			}
			if score >= suggestMin && !slices.Contains(skip, i) {
				fits = append(fits, fit{k, i, score})
			}
		}
	}
	// each passage to the scene it fits best
	best := map[int]fit{}
	for _, f := range fits {
		if b, ok := best[f.passage]; !ok || f.score > b.score || (f.score == b.score && f.scene < b.scene) {
			best[f.passage] = f
		}
	}
	per := map[int][]fit{}
	for _, f := range best {
		per[f.scene] = append(per[f.scene], f)
	}
	var out []linkSuggestion
	for k, sc := range scenes {
		fs := per[k]
		if len(fs) == 0 {
			continue
		}
		slices.SortFunc(fs, func(a, b fit) int {
			if a.score != b.score {
				return cmpFloat(b.score, a.score)
			}
			return a.passage - b.passage
		})
		fs = fs[:min(len(fs), suggestPerScene)]
		sg := linkSuggestion{Scene: k, Heading: sc.Heading}
		for _, f := range fs {
			sg.Passages = append(sg.Passages, f.passage)
		}
		slices.Sort(sg.Passages) // in the story's order
		out = append(out, sg)
	}
	return out
}

func cmpFloat(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// overlap is how much of a's words b has (0–1).
func overlap(a, b []string) float64 {
	if len(a) == 0 {
		return 0
	}
	in := map[string]bool{}
	for _, w := range b {
		in[w] = true
	}
	hit := 0
	for _, w := range a {
		if in[w] {
			hit++
		}
	}
	return float64(hit) / float64(len(a))
}

// The pane's side.

// suggestions is what the pane offers: the list and the one asked about.
type suggestions struct {
	list  []linkSuggestion
	at    int
	tried bool // offered once for this source (auto)
}

// offerSuggestions reads the scenes against the source, in the
// background, and asks about the first suggestion; auto (the pane
// opening on a script with no links) says nothing when there is none.
func (s *sourcePane) offerSuggestions(auto bool) {
	s.load()
	if s.ref.Origin == sourceNone || len(s.passages) == 0 {
		if !auto {
			s.w.setStatus("No source: File > Choose Source…")
		}
		return
	}
	m := s.current()
	text, starts, ps, status := s.modelFor, s.w.sceneStarts(), s.passages, slices.Clone(m.status)
	reading := s.reading
	key, hints := s.readingKey()
	if s.readingFor != key {
		reading = nil
	}
	l := learnedOf(text)
	skipped := l.skippedFor(ps)
	lang, _ := readingLanguage(s.src, s.w.language)
	s.sugg.tried = true
	inBackground(func() []linkSuggestion {
		r := reading
		if r == nil {
			r = storyread.Read(storyPassages(ps), lang, hints)
		}
		return suggestLinks(text, starts, ps, status, r, skipped)
	}, func(list []linkSuggestion) {
		if !samePassages(ps, s.passages) || !sameString(text, s.w.textEditor.Text()) {
			return // changed meanwhile
		}
		s.sugg.list, s.sugg.at = list, 0
		if len(list) == 0 {
			if !auto {
				s.w.setStatus("No scene without a link reads like a passage of the source")
			}
			return
		}
		s.showSuggestion(!auto)
	})
}

// showSuggestion asks about the suggestion at sugg.at, its first passage
// scrolled to; asked for (go: the writer went on from one before), the
// cursor goes to the scene and the passage is picked. Offered by itself,
// the script and what is picked stay as they are.
func (s *sourcePane) showSuggestion(goTo bool) {
	if s.sugg.at >= len(s.sugg.list) {
		s.prompt.hide()
		s.w.setStatus("Suggested links: gone through them all")
		return
	}
	sg := s.sugg.list[s.sugg.at]
	var quotes []string
	for _, p := range sg.Passages {
		if p < len(s.passages) {
			quotes = append(quotes, "“"+shortQuote(s.passages[p].Plain)+"”")
		}
	}
	text := fmt.Sprintf("Suggested link %d of %d: %s reads like %s. Link it?", s.sugg.at+1, len(s.sugg.list),
		sg.Heading, strings.Join(quotes, " and "))
	s.prompt.askSkip(text, "Link", s.acceptSuggestion, "Skip", func() {
		s.skipSuggestion(sg)
		s.sugg.at++
		s.showSuggestion(true)
	})
	p := sg.Passages[0]
	if p >= len(s.passages) {
		return
	}
	s.list.ScrollTo(p)
	if !goTo {
		return
	}
	if line := s.sceneLine(sg.Scene, sg.Heading); line >= 0 {
		s.jumping = true
		s.w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(line), false) })
		s.jumping = false
	}
	s.selected, s.cycle = p, 0
	s.list.Refresh()
	s.showCutButton()
	s.follow()
}

// sceneLine is the line of the script's k-th scene heading, if it is
// still the heading suggested for; else the first scene with that
// heading (-1: none, the script changed).
func (s *sourcePane) sceneLine(k int, heading string) int {
	text := s.w.textEditor.Text()
	n, first := 0, -1
	for _, it := range outlineOf(text[:bodyEndByte(text)], s.w.sceneStarts()) {
		if !it.Scene {
			continue
		}
		if it.Text == heading {
			if n == k {
				return it.Line
			}
			if first < 0 {
				first = it.Line
			}
		}
		n++
	}
	return first
}

// acceptSuggestion links the scene asked about to its passages: the
// links on lines below its heading, one undo step; then the next.
func (s *sourcePane) acceptSuggestion() {
	if s.sugg.at >= len(s.sugg.list) {
		return
	}
	sg := s.sugg.list[s.sugg.at]
	s.sugg.at++
	line := s.sceneLine(sg.Scene, sg.Heading)
	if line < 0 {
		s.showSuggestion(true)
		return
	}
	var notes []string
	for _, p := range sg.Passages {
		if p < len(s.passages) {
			q, _ := anchorFor(s.passages, p)
			notes = append(notes, linkNote(q))
		}
	}
	e := s.w.textEditor
	b := e.Buffer()
	r := []rune(e.Text())
	_, end := lineBounds(r, b.LineStart(line))
	at, insert, _ := insertLink(e.Text(), end, strings.Join(notes, "\n"), s.w.sceneStarts())
	e.Edit(func(b *buffer.Buffer) {
		b.Group(func() { b.Replace(at, at, insert) })
	})
	s.w.setStatus("Linked " + sg.Heading + " to " + plural(len(notes), "passage"))
	s.showSuggestion(true)
}

// offerStart asks, once per source, what a script that has not begun
// with it could do first: an unwritten one, start from the source; one
// with scenes and no links, the links its scenes suggest.
func (s *sourcePane) offerStart(m *sourceModel) {
	if s.prompt.box.Visible() || !slices.ContainsFunc(s.passages, passage.counted) {
		return
	}
	if len(m.links) > 0 || (s.seedAsked && s.sugg.tried) {
		return // the cursor moves: nothing to read again
	}
	if !s.seedAsked && len(m.spans) == 1 && m.spans[0].Heading == "" { // no scene, no section
		s.seedAsked = true
		s.prompt.ask(-1, false, "Nothing written yet. Start from the source: its chapters as sections, and the first scene?",
			"Start from the Source", s.seedScript)
		return
	}
	if s.sugg.tried || !slices.ContainsFunc(outlineOf(s.modelFor, s.w.sceneStarts()), func(it outlineItem) bool { return it.Scene }) {
		return
	}
	if len(s.src) >= backgroundSource { // a long one is read for it only when asked
		s.sugg.tried = true
		s.prompt.ask(-1, false, "The script has scenes but no links to the source yet. Suggest them, scene by scene?",
			"Suggest Links", func() { s.offerSuggestions(false) })
		return
	}
	s.offerSuggestions(true)
}

// suggestSourceLinks is Format > Suggest Links to the Source.
func (w *MainWindow) suggestSourceLinks() {
	if w.markdown {
		return
	}
	s := w.sourcePane()
	if !s.shown() {
		w.toggleSource()
	}
	s.offerSuggestions(false)
}

// skipSuggestion remembers a suggestion skipped, in the adaptation block
// (skipped links:): its passages are not offered to that scene again.
func (s *sourcePane) skipSuggestion(sg linkSuggestion) {
	l := learnedOf(s.w.textEditor.Text())
	for _, p := range sg.Passages {
		if p < len(s.passages) {
			l.addSkip(sg.Heading, cutFor(s.passages, p))
		}
	}
	s.w.writeLearned(l, false)
}
