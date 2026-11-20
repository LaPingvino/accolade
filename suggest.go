package main

import (
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/LaPingvino/lexington/lex"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/pace"
	"github.com/LaPingvino/accolade/internal/storyread"
)

// Suggestions while writing (docs/ADAPTATION.md, section 5): the
// passages linked in the cursor's scene (else the one Next showed) offer
// what comes next, in the status bar as the Tab completion's hint does;
// Tab takes it, Tab again goes to the next one, Esc or typing on puts it
// away.
//
//   - A character cue written: that character's next line not in the
//     scene yet (with its parenthetical), then the lines no one is named
//     for. The same on the empty line under the cue.
//   - A cue being typed completes from the passages' speakers first.
//   - An empty line: the passage's next piece not in the scene, a line
//     with its cue, or the narration as action, as the source has it.
//   - A scene heading being typed: the place and time the source reads.
//
// What the source offers is read once per source (storyread, in the
// background for a long one); the cursor's scene, its passages and what
// of them it has are found once per revision and line, from the scene's
// lines only. A keystroke looks them up.

// suggestPref turns the suggestions on or off (Preferences > Script >
// Adapting).
const suggestPref = "suggest-from-source"

// castPlaceholder is the cue of a line no one is named for, selected to
// write over.
const castPlaceholder = "CHARACTER"

// offerPiece is a piece of a passage the script can take: a line and who
// says it, or narration.
type offerPiece struct {
	Speech  bool
	Speaker string // as the source reads it ("" unknown)
	Manner  string // the parenthetical: whispering
	Text    string
	Unseen  bool   // narration a camera cannot see (a thought, a feeling)
	key     string // its words, normalised, between spaces: " a b c "
	words   int
}

// placeGuess is the heading the source suggests for a passage's scene.
type placeGuess struct {
	Setting     storyread.Setting
	Place, Time string
	Given       string // the writer's heading for the place (places:), without the time
}

// sourceOffers is what a source offers, read once.
type sourceOffers struct {
	ps     []passage      // the passages it was read from
	lang   string         // read in
	script string         // the script's language then
	pieces [][]offerPiece // each passage's, in order
	places []placeGuess   // each passage's scene's
	hints  string         // read with (learned.hintsKey)
}

// offersOf reads what passages offer, in lang.
func offersOf(ps []passage, lang, script string, r *storyread.Reading) *sourceOffers {
	o := &sourceOffers{ps: ps, lang: lang, script: script, pieces: make([][]offerPiece, len(ps)), places: make([]placeGuess, len(ps))}
	for i, pr := range r.Passages {
		if i >= len(ps) {
			break
		}
		if sc := pr.Scene; sc >= 0 && sc < len(r.Scenes) {
			o.places[i] = placeGuess{r.Scenes[sc].Setting, r.Scenes[sc].Place, r.Scenes[sc].Time, r.Scenes[sc].Given}
		}
		if !ps[i].counted() {
			continue
		}
		pace.Yield()
		done := map[int]bool{}
		for _, pc := range pr.Pieces {
			var op offerPiece
			switch {
			case pc.Speech && pc.Line >= 0 && pc.Line < len(pr.Lines):
				if done[pc.Line] {
					continue // the line's other pieces: it is offered whole
				}
				done[pc.Line] = true
				l := pr.Lines[pc.Line]
				op = offerPiece{Speech: true, Speaker: l.Speaker, Manner: l.Manner, Text: lineText(pr.Pieces, pc.Line, l.Text)}
			case pc.Speech || pc.Tag:
				continue // who says it: the cue says that
			default:
				op = offerPiece{Text: pc.Text}
				for _, f := range pr.Flags {
					if f.Start < pc.End && f.End > pc.Start {
						op.Unseen = true
					}
				}
			}
			ws := normWords(op.Text)
			if len(ws) == 0 {
				continue
			}
			op.key, op.words = " "+strings.Join(ws, " ")+" ", len(ws)
			o.pieces[i] = append(o.pieces[i], op)
		}
	}
	return o
}

// lineText is line li said again from its pieces: where the narration
// between two of them ended a sentence ("“No,” said Anna. “There is
// never tea.”"), the comma before it ends one too, as a comma at its
// end does.
func lineText(pieces []storyread.Piece, li int, as string) string {
	var b strings.Builder
	ended := false // the narration since the last piece ended a sentence
	for _, pc := range pieces {
		switch {
		case !pc.Speech:
			if b.Len() > 0 {
				t := strings.TrimRight(pc.Text, "\"'”’»« ")
				ended = strings.ContainsAny(t[max(len(t)-1, 0):], ".!?…")
			}
		case pc.Line == li:
			if b.Len() > 0 {
				so := strings.TrimSpace(b.String())
				if ended && strings.HasSuffix(so, ",") {
					so = strings.TrimSuffix(so, ",") + "."
				}
				b.Reset()
				b.WriteString(so + " ")
			}
			b.WriteString(pc.Text)
			ended = false
		}
	}
	if b.Len() == 0 {
		return as
	}
	// "“Then I will,” Tom said.": the line ends where the tag does
	t := strings.TrimSpace(b.String())
	if c, ok := strings.CutSuffix(t, ","); ok {
		t = c + "."
	}
	return t
}

// sceneCtx is the cursor's scene as the suggestions see it.
type sceneCtx struct {
	head, end int             // the heading's line, and the line after the scene
	passages  []int           // linked in it (else the one Next showed), in order
	linked    bool            // passages are the scene's links
	joined    string          // the scene's words, normalised, between spaces
	lineKeys  map[string]bool // each line's words so
}

// used reports whether the scene has the piece: its words in a row (a
// piece of a word or two only as a line of its own).
func (c *sceneCtx) used(p offerPiece) bool {
	if p.words < 3 {
		return c.lineKeys[p.key]
	}
	return strings.Contains(c.joined, p.key)
}

// ctxSlot is a scene found for a revision and line.
type ctxSlot struct {
	ok      bool
	rev     uint64
	line    int
	offered int // the passage Next showed then
	c       *sceneCtx
}

// suggestion is what Tab would put in at the cursor: [start, end) (runes)
// replaced by one of alts, each told in hints.
type suggestion struct {
	start, end int
	alts       []string
	hints      []string
	takes      []takenCue // a cue from the source in the alternative, to learn from (adaptation_learn.go)
}

// suggestCycle is a suggestion taken: Tab again puts in the next one,
// while nothing changed since.
type suggestCycle struct {
	s        *suggestion
	i        int
	rev      uint64
	cursor   int
	inserted string
}

// suggester keeps a window's suggestions: what its source offers, the
// cursor's scene, the cycle and a dismissal.
type suggester struct {
	w *MainWindow

	offers       *sourceOffers
	pending      []passage // being read in the background
	pendingHints string    // with these hints
	learn        learner   // what the writer does, learned from (adaptation_learn.go)

	rev     uint64 // the text's revision the lines and cues are of
	lines   []string
	posRev  uint64 // the cursor's line and column, for this revision and offset
	posAt   int
	posLine int
	posCol  int
	cues    map[string]int
	cueOf   map[string]string // cueFor's, this revision
	cuesOK  bool
	ctxs    [2]ctxSlot // the cursor's scene, as a line in it and as its heading being typed

	cycle     *suggestCycle
	dismissed struct {
		rev  uint64
		line int
		on   bool
	}
	loaded    bool     // the source was read for the suggestions once
	loadedFor string   // for this file
	seenRev   uint64   // the text's revision it was last looked at in
	loadTimer *uiTimer // reading it again after typing, the pane hidden
}

func (w *MainWindow) suggester() *suggester {
	if w.suggest == nil {
		w.suggest = &suggester{w: w}
	}
	return w.suggest
}

// on reports whether suggestions are wanted here: the preference, a
// screenplay, not vim's normal mode.
func (sg *suggester) on() bool {
	w := sg.w
	if w.markdown || (w.settings != nil && !w.settings.GetBoolean(suggestPref)) {
		return false
	}
	if w.vim != nil && w.vim.active() && w.vim.mode != vimInsert {
		return false
	}
	return true
}

// source is the source pane's state, read for the suggestions if the
// pane never was; nil without a source or passages yet.
func (sg *suggester) source() *sourcePane {
	w := sg.w
	rev := w.textEditor.Buffer().Rev()
	// the pane is made for a script with a source only
	load := func() {
		if w.notes != nil || w.sourceRef().Origin != sourceNone {
			w.sourcePane().loadSource(true)
		}
	}
	switch {
	case !sg.loaded || sg.loadedFor != w.currentFile: // another script
		sg.loaded, sg.loadedFor, sg.seenRev = true, w.currentFile, rev
		load()
		w.knowScript()
	case rev != sg.seenRev: // the pane reads it again as it follows; hidden, after a pause
		sg.seenRev = rev
		if w.notes == nil || !w.notes.source.shown() {
			w.debounce(&sg.loadTimer, time.Second, func() {
				load()
				w.learnFromWriting() // the pane hidden, it does not follow
			})
		}
	}
	if w.notes == nil {
		return nil
	}
	s := w.notes.source
	if s.ref.Origin == sourceNone || s.loading != nil || len(s.passages) == 0 {
		return nil
	}
	return s
}

// offersFor is what the pane's source offers, read again when its
// passages or language changed; nil while a long one is read.
func (sg *suggester) offersFor(s *sourcePane) *sourceOffers {
	_, hk := readLearned(sg.w.textEditor.Text())
	if o := sg.offers; o != nil && samePassages(o.ps, s.passages) && o.script == sg.w.language && o.hints == hk {
		return o // the language is the source's, or the script's: it holds while both do
	}
	lang, _ := readingLanguage(s.src, sg.w.language)
	ps := s.passages
	if samePassages(sg.pending, ps) && sg.pendingHints == hk {
		return nil // being read
	}
	key, hints := s.readingKey()
	if s.reading != nil && s.readingFor == key {
		sg.offers = offersOf(ps, lang, sg.w.language, s.reading)
		sg.offers.hints = hk
		return sg.offers
	}
	if len(s.src) < backgroundSource {
		sg.offers = offersOf(ps, lang, sg.w.language, s.currentReading())
		sg.offers.hints = hk
		return sg.offers
	}
	sg.pending, sg.pendingHints = ps, hk
	script := sg.w.language
	src := s.src
	type read struct {
		r *storyread.Reading
		o *sourceOffers
	}
	inBackground(func() read {
		r := storyread.Read(storyPassages(ps), lang, hints)
		o := offersOf(ps, lang, script, r)
		o.hints = hk
		return read{r, o}
	}, func(got read) {
		if !samePassages(sg.pending, ps) || sg.pendingHints != hk {
			return
		}
		sg.pending = nil
		sg.offers = got.o
		if s.src == src && s.reading == nil {
			s.reading, s.readingFor = got.r, key
		}
		sg.w.showSuggestionHint()
	})
	return nil
}

// refresh keeps the lines of the text's revision.
func (sg *suggester) refresh() {
	b := sg.w.textEditor.Buffer()
	if sg.lines != nil && b.Rev() == sg.rev {
		return
	}
	sg.rev = b.Rev()
	sg.lines = strings.Split(b.Text(), "\n")
	sg.cuesOK, sg.ctxs = false, [2]ctxSlot{}
	clear(sg.cueOf)
}

// position is the cursor's line and column, found once per revision and
// offset.
func (sg *suggester) position() (line, col int) {
	b := sg.w.textEditor.Buffer()
	if c := b.Cursor(); sg.posRev != b.Rev() || sg.posAt != c || sg.posLine < 0 {
		sg.posLine, sg.posCol = b.Position(c)
		sg.posRev, sg.posAt = b.Rev(), c
	}
	return sg.posLine, sg.posCol
}

// scriptCues are the script's character names, once per revision.
func (sg *suggester) scriptCues() map[string]int {
	if !sg.cuesOK {
		text := sg.w.textEditor.Text()
		sg.cues, sg.cuesOK = characterNames(text[:bodyEndByte(text)]), true
	}
	return sg.cues
}

// cueFor is the cue of a speaker as the source names them: the script's
// own cue when one is the name or part of it (ANNA for Anna Kovač), else
// the name in capitals, without dots (a cue cannot have one).
func (sg *suggester) cueFor(speaker string) string {
	if cue, ok := sg.cueOf[speaker]; ok {
		return cue
	}
	if sg.cueOf == nil {
		sg.cueOf = map[string]string{}
	}
	cue := sg.cueForNow(speaker)
	sg.cueOf[speaker] = cue
	return cue
}

func (sg *suggester) cueForNow(speaker string) string {
	if speaker == storyread.Narrator {
		speaker = "Narrator"
	}
	if l, _ := readLearned(sg.w.textEditor.Text()); len(l.Aliases) > 0 {
		if cue := l.aliasOf(speaker); cue != "" {
			return cue // characters: ANNA = Anna, Annie
		}
	}
	cue := strings.ToUpper(strings.Join(strings.FieldsFunc(speaker, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '\'' && r != '’' && r != '-'
	}), " "))
	cues := sg.scriptCues()
	if _, ok := cues[cue]; ok || cue == "" {
		return cue
	}
	words := strings.Fields(cue)
	best, bestN := "", 0
	for c, n := range cues {
		cw := strings.Fields(c)
		if len(cw) > 0 && (subset(cw, words) || subset(words, cw)) && (n > bestN || n == bestN && c < best) {
			best, bestN = c, n
		}
	}
	if best != "" {
		return best
	}
	return cue
}

// subset reports whether every word of a is in b.
func subset(a, b []string) bool {
	for _, w := range a {
		if !slices.Contains(b, w) {
			return false
		}
	}
	return true
}

// headingAt reports whether line i of lines is a scene heading.
func headingAt(lines []string, i int, starts []string) bool {
	t := strings.TrimSpace(lines[i])
	if i > 0 && strings.TrimSpace(lines[i-1]) != "" {
		return false
	}
	t, _ = lex.SceneNumber(t)
	return isSceneHeadingIn(t, starts) || strings.HasPrefix(t, ".") && !strings.HasPrefix(t, "..")
}

func sectionAt(lines []string, i int) bool {
	return strings.HasPrefix(strings.TrimSpace(lines[i]), "#")
}

// scene is the scene with line in it (nil: none, as in the title page),
// its passages and what it has of them; head, when line is the heading
// being typed.
func (sg *suggester) scene(s *sourcePane, line int, head bool) *sceneCtx {
	slot := &sg.ctxs[0]
	if head {
		slot = &sg.ctxs[1]
	}
	if !slot.ok || slot.rev != sg.rev || slot.line != line || slot.offered != s.offered {
		*slot = ctxSlot{ok: true, rev: sg.rev, line: line, offered: s.offered, c: sg.sceneOf(s, line, head)}
	}
	return slot.c
}

func (sg *suggester) sceneOf(s *sourcePane, line int, head bool) *sceneCtx {
	lines, starts := sg.lines, sg.w.sceneStarts()
	if line >= len(lines) {
		return nil
	}
	// up to the heading, past notes and other boneyard (/* … */); in
	// one, as in the source and adaptation blocks after the script, none
	h := -1
	skipping := false
	for i := line; i >= 0; i-- {
		t := strings.TrimSpace(lines[i])
		opens, closes := strings.HasPrefix(t, "/*"), strings.Contains(t, "*/")
		switch {
		case skipping:
			skipping = !opens
			continue
		case closes && !opens && i < line:
			skipping = true
			continue
		case opens && !(closes && i < line):
			return nil
		case opens:
			continue
		case i == line && !head:
			continue
		}
		if head && i == line || headingAt(lines, i, starts) {
			h = i
			break
		}
		if sectionAt(lines, i) {
			return nil
		}
	}
	if h < 0 {
		return nil // before the first scene
	}
	// down to the next, the boneyard left out
	var body []string
	end := h + 1
	for skipping = false; end < len(lines); end++ {
		t := strings.TrimSpace(lines[end])
		if skipping {
			skipping = !strings.Contains(t, "*/")
			continue
		}
		if strings.HasPrefix(t, "/*") {
			skipping = !strings.Contains(t[2:], "*/")
			continue
		}
		if headingAt(lines, end, starts) || sectionAt(lines, end) {
			break
		}
		body = append(body, lines[end])
	}
	c := &sceneCtx{head: h, end: end, lineKeys: map[string]bool{}}
	if s.matcher == nil || !samePassages(s.matcher.ps, s.passages) {
		s.matcher = matcherFor(s.passages)
	}
	for _, l := range parseLinks(strings.Join(body, "\n")) {
		if mt := s.matcher.matchRef(l.Ref); mt.found() {
			for p := mt.Passages[0]; p <= mt.last(); p++ {
				if !slices.Contains(c.passages, p) {
					c.passages = append(c.passages, p)
				}
			}
		}
	}
	slices.Sort(c.passages)
	c.linked = len(c.passages) > 0
	if len(c.passages) == 0 {
		if p := s.offeredPassage(); p >= 0 {
			c.passages = []int{p}
		} else if head {
			if p := sg.nextAfterLinks(s, h); p >= 0 {
				c.passages = []int{p}
			}
		}
	}
	var all []string
	for _, raw := range body {
		ws := normWords(stripNotes(raw))
		if len(ws) == 0 {
			continue
		}
		c.lineKeys[" "+strings.Join(ws, " ")+" "] = true
		all = append(all, ws...)
	}
	c.joined = " " + strings.Join(all, " ") + " "
	return c
}

// nextAfterLinks is the passage a new scene at line goes on with: the
// first to do after the last one linked above it (the first of the
// source without a link above).
func (sg *suggester) nextAfterLinks(s *sourcePane, line int) int {
	last := -1
	for i := line - 1; i >= 0 && last < 0; i-- {
		for _, l := range parseLinks(sg.lines[i]) {
			if mt := s.matcher.matchRef(l.Ref); mt.found() {
				last = max(last, mt.last())
			}
		}
	}
	for p := last + 1; p < len(s.passages); p++ {
		if !s.passages[p].counted() {
			continue
		}
		if len(s.statuses) == len(s.passages) && s.statuses[p] != statusToDo {
			continue
		}
		return p
	}
	return -1
}

// unused are the pieces of the scene's passages it has not got yet, in
// order.
func (sg *suggester) unused(o *sourceOffers, c *sceneCtx) []offerPiece {
	var out []offerPiece
	for _, p := range c.passages {
		if p >= len(o.pieces) {
			continue
		}
		for _, pc := range o.pieces[p] {
			if !c.used(pc) {
				out = append(out, pc)
			}
		}
	}
	return out
}

func isCueLine(lines []string, i int) bool {
	if i < 0 || i >= len(lines) {
		return false
	}
	t := strings.TrimSpace(lines[i])
	return t != "" && (i == 0 || strings.TrimSpace(lines[i-1]) == "") &&
		(isCharacterCue(t) || strings.HasPrefix(t, "@"))
}

func blankAt(lines []string, i int) bool {
	return i < 0 || i >= len(lines) || strings.TrimSpace(lines[i]) == ""
}

// at is what Tab would put in at the cursor from the source (nil:
// nothing): a character's line after their cue, the next piece on an
// empty line.
func (sg *suggester) at() *suggestion {
	if !sg.on() {
		return nil
	}
	s := sg.source()
	if s == nil {
		return nil
	}
	sg.refresh()
	e := sg.w.textEditor
	b := e.Buffer()
	if b.HasSelection() {
		return nil
	}
	cursor := b.Cursor()
	line, col := sg.position()
	if line >= len(sg.lines) {
		return nil
	}
	if d := sg.dismissed; d.on && d.rev == sg.rev && d.line == line {
		return nil
	}
	lines := sg.lines
	text := lines[line]
	if col != utf8.RuneCountInString(text) {
		return nil // only at a line's end
	}
	o := sg.offersFor(s)
	if o == nil {
		return nil
	}
	start := cursor - col
	switch {
	case isCueLine(lines, line) && blankAt(lines, line+1):
		c := sg.scene(s, line, false)
		if c == nil {
			return nil
		}
		sug, own := sg.linesFor(o, c, strings.ToUpper(cueName(text)), cursor, cursor, "\n", true)
		if sug != nil && !own && sg.startsSomething(text) {
			return nil // AN, EXT: a name or heading being typed, not a cue for a line no one is named for
		}
		return sug
	case strings.TrimSpace(text) != "":
		return nil
	case isCueLine(lines, line-1): // the cue's empty dialogue line
		c := sg.scene(s, line, false)
		if c == nil {
			return nil
		}
		sug, _ := sg.linesFor(o, c, strings.ToUpper(cueName(lines[line-1])), start, cursor, "", true)
		return sug
	case strings.HasPrefix(strings.TrimSpace(lines[max(line-1, 0)]), "(") && isCueLine(lines, line-2):
		c := sg.scene(s, line, false)
		if c == nil {
			return nil
		}
		sug, _ := sg.linesFor(o, c, strings.ToUpper(cueName(lines[line-2])), start, cursor, "", false)
		return sug
	}
	c := sg.scene(s, line, false)
	if c == nil {
		return nil
	}
	before, after := "", ""
	if !blankAt(lines, line-1) {
		before = "\n" // a new element: a blank line before it
	}
	if !blankAt(lines, line+1) {
		after = "\n"
	}
	sug := &suggestion{start: start, end: cursor}
	for _, pc := range sg.unused(o, c) {
		var body, hint string
		var take takenCue
		switch {
		case pc.Speech && pc.Speaker == "":
			body = castPlaceholder + "\n" + paren(pc.Manner) + pc.Text
			hint = quoteShort(pc.Text) + ", who says it? (the cue is yours)"
		case pc.Speech:
			cue := sg.cueFor(pc.Speaker)
			body = cue + "\n" + paren(pc.Manner) + pc.Text
			hint = cue + withManner(pc.Manner) + ": " + quoteShort(pc.Text)
			take = takenCue{Speaker: pc.Speaker, Cue: cue, Text: pc.Text}
		default:
			body = pc.Text
			hint = "action, as the source has it: " + quoteShort(pc.Text)
			if pc.Unseen {
				hint += " (not seen on screen)"
			}
		}
		sug.alts = append(sug.alts, before+body+after)
		sug.hints = append(sug.hints, hint)
		sug.takes = append(sug.takes, take)
	}
	if len(sug.alts) == 0 {
		return nil
	}
	return sug
}

// linesFor is the lines a character says that the scene has not got,
// then those no one is named for: put in at [start, end) after lead (a
// line break after the cue), with their parenthetical if withParen; own
// reports whether the character has lines of their own in it.
func (sg *suggester) linesFor(o *sourceOffers, c *sceneCtx, cue string, start, end int, lead string, withParen bool) (sug *suggestion, own bool) {
	sug = &suggestion{start: start, end: end}
	var nobody []offerPiece
	for _, pc := range sg.unused(o, c) {
		switch {
		case !pc.Speech:
		case pc.Speaker == "":
			nobody = append(nobody, pc)
		case sg.cueFor(pc.Speaker) == cue:
			sug.add(pc, lead, withParen, cue)
		}
	}
	own = len(sug.alts) > 0
	for _, pc := range nobody {
		sug.add(pc, lead, withParen, "")
	}
	if len(sug.alts) == 0 {
		return nil, false
	}
	return sug, own
}

// startsSomething reports whether a line being typed could be the start
// of a heading, or of a name Tab completes.
func (sg *suggester) startsSomething(line string) bool {
	starts := sg.w.sceneStarts()
	t := strings.TrimSpace(line)
	if headingStart(t, starts) || isSceneHeadingIn(t, starts) {
		return true
	}
	upper := strings.ToUpper(cueName(t))
	for name := range sg.scriptCues() {
		if strings.HasPrefix(name, upper) && name != upper {
			return true
		}
	}
	names, _ := sg.firsts(0)
	for _, name := range names {
		if strings.HasPrefix(name, upper) && name != upper {
			return true
		}
	}
	return false
}

func (sug *suggestion) add(pc offerPiece, lead string, withParen bool, cue string) {
	text := pc.Text
	if withParen {
		text = paren(pc.Manner) + text
	}
	sug.alts = append(sug.alts, lead+text)
	if cue == "" {
		sug.hints = append(sug.hints, quoteShort(pc.Text)+", no one named for it")
	} else {
		sug.hints = append(sug.hints, cue+"’s line"+withManner(pc.Manner)+": "+quoteShort(pc.Text))
	}
}

// paren is a manner as a parenthetical line ("" none).
func paren(manner string) string {
	if manner == "" {
		return ""
	}
	return "(" + manner + ")\n"
}

func withManner(manner string) string {
	if manner == "" {
		return ""
	}
	return " (" + manner + ")"
}

// quoteShort is a piece quoted for the status bar, cut to a few words:
// on a phone the bar is narrow.
func quoteShort(s string) string {
	words := strings.Fields(s)
	if len(words) > 8 {
		return "“" + strings.Join(words[:8], " ") + "…”"
	}
	return "“" + strings.Join(words, " ") + "”"
}

// hintSuffix tells where a suggestion comes from, and how many there are.
func hintSuffix(i, n int) string {
	h := " · from the source"
	if n > 1 {
		h += " · " + strconv.Itoa(i+1) + " of " + strconv.Itoa(n)
	}
	return h
}

// firsts are the cues and headings the source puts first in Tab's
// completions at the cursor: the scene's passages' speakers, in the
// order of their next lines; the heading the source reads for the
// scene being headed.
func (sg *suggester) firsts(cursor int) (names, heads []string) {
	if !sg.on() {
		return nil, nil
	}
	s := sg.source()
	if s == nil {
		return nil, nil
	}
	sg.refresh()
	line, _ := sg.position()
	if line >= len(sg.lines) {
		return nil, nil
	}
	o := sg.offersFor(s)
	if o == nil {
		return nil, nil
	}
	typed := strings.TrimSpace(sg.lines[line])
	if isSceneHeadingIn(typed, sg.w.sceneStarts()) || headingStart(typed, sg.w.sceneStarts()) {
		c := sg.scene(s, line, true)
		if c != nil && len(c.passages) > 0 && c.passages[0] < len(o.places) {
			heads = sg.headingsFor(o.places[c.passages[0]])
		}
		if isSceneHeadingIn(typed, sg.w.sceneStarts()) {
			return nil, heads // INT. … is no one's name
		}
	}
	c := sg.scene(s, line, false)
	if c == nil {
		return nil, nil
	}
	for _, pc := range sg.unused(o, c) {
		if pc.Speech && pc.Speaker != "" {
			if cue := sg.cueFor(pc.Speaker); !slices.Contains(names, cue) {
				names = append(names, cue)
			}
		}
	}
	return names, heads
}

// headingStart reports whether t is the start of a heading word being
// typed: "IN", "ext" (two letters at least).
func headingStart(t string, starts []string) bool {
	u := strings.ToUpper(t)
	if utf8.RuneCountInString(u) < 2 {
		return false
	}
	for _, s := range starts {
		if strings.HasPrefix(s, u) {
			return true
		}
	}
	return false
}

// headingsFor are the headings a guessed place makes, the setting read
// first: INT. KITCHEN - MORNING, EXT. KITCHEN - MORNING …; none without a
// place.
//
// A place the writer gave a heading (places:) has it first.
func (sg *suggester) headingsFor(g placeGuess) []string {
	if g.Place == "" {
		return nil
	}
	var hs []string
	if g.Given != "" {
		h := g.Given
		if g.Time != "" {
			h += " - " + g.Time
		}
		hs = append(hs, h)
	}
	for _, h := range sg.readHeadings(g) {
		if !slices.Contains(hs, h) {
			hs = append(hs, h)
		}
	}
	return hs
}

// readHeadings are the headings the place as read makes.
func (sg *suggester) readHeadings(g placeGuess) []string {
	in, out, either, ok := placeWords(sceneHeaders(sg.w.language))
	if !ok {
		in, out, either = "INT.", "EXT.", "INT./EXT."
	}
	rest := g.Place
	if g.Time != "" {
		rest += " - " + g.Time
	}
	order := []string{either, in, out}
	switch g.Setting {
	case storyread.Inside:
		order = []string{in, out, either}
	case storyread.Outside:
		order = []string{out, in, either}
	}
	var hs []string
	for _, w := range order {
		hs = append(hs, w+" "+rest)
	}
	return hs
}

// valid reports whether the cycle is still where Tab left it.
func (c *suggestCycle) valid(b *buffer.Buffer) bool {
	return c != nil && b.Rev() == c.rev && b.Cursor() == c.cursor
}

// tab takes the suggestion at the cursor, or the next one of those just
// taken; false if there is none (Tab completes or indents).
func (sg *suggester) tab() bool {
	e := sg.w.textEditor
	b := e.Buffer()
	if c := sg.cycle; c.valid(b) {
		if len(c.s.alts) < 2 {
			return false
		}
		c.i = (c.i + 1) % len(c.s.alts)
		sg.put(c.s.start, c.s.start+utf8.RuneCountInString(c.inserted), c.s, c.i)
		return true
	}
	sg.cycle = nil
	if sg.w.complete != nil && sg.w.complete.active(e) {
		return false // going through the completions
	}
	s := sg.at()
	if s == nil {
		return false
	}
	sg.put(s.start, s.end, s, 0)
	return true
}

// put puts in alternative i of s over [from, to), one undo step; a
// placeholder cue in it is selected to write over.
func (sg *suggester) put(from, to int, s *suggestion, i int) {
	e := sg.w.textEditor
	text := s.alts[i]
	e.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			b.Replace(from, to, text)
			end := from + utf8.RuneCountInString(text)
			if k := strings.Index(text, castPlaceholder+"\n"); k >= 0 {
				at := from + utf8.RuneCountInString(text[:k])
				b.Select(at, at+utf8.RuneCountInString(castPlaceholder))
			} else {
				b.SetCursor(end, false)
			}
		})
	})
	b := e.Buffer()
	sg.cycle = &suggestCycle{s: s, i: i, rev: b.Rev(), cursor: b.Cursor(), inserted: text}
	sg.learn.took(s, i, s.takes)
	sg.w.showSuggestionHint()
}

// dismiss puts the suggestion away (Esc) until the text or the line
// changes; it reports whether one was shown.
func (sg *suggester) dismiss() bool {
	w := sg.w
	if w.statusHint == "" {
		return false
	}
	line, _ := sg.position()
	sg.dismissed.rev, sg.dismissed.line, sg.dismissed.on = w.textEditor.Buffer().Rev(), line, true
	sg.cycle = nil
	w.setStatus("")
	return true
}

// hint is what the status bar says Tab would do at the cursor: the next
// of the suggestions just taken, a suggestion, or the completions.
func (sg *suggester) hint() string {
	e := sg.w.textEditor
	b := e.Buffer()
	if d := sg.dismissed; d.on {
		line, _ := sg.position()
		if d.rev == b.Rev() && d.line == line {
			return ""
		}
		sg.dismissed.on = false
	}
	if c := sg.cycle; c.valid(b) {
		if len(c.s.alts) < 2 {
			return ""
		}
		n := (c.i + 1) % len(c.s.alts)
		return "Tab: next, " + c.s.hints[n] + hintSuffix(n, len(c.s.alts))
	}
	if k := sg.w.complete; k != nil && k.active(e) {
		return k.cycleHint()
	}
	if s := sg.at(); s != nil {
		return "Tab: " + s.hints[0] + hintSuffix(0, len(s.alts))
	}
	names, heads := sg.firsts(b.Cursor())
	return completionHintWith(e.Text(), b.Cursor(), sg.w.sceneStarts(), names, heads)
}

// showSuggestionHint puts the hint for the cursor in the status bar, or
// takes an old one away.
func (w *MainWindow) showSuggestionHint() {
	if w.markdown || w.textEditor == nil {
		return
	}
	if h := w.suggester().hint(); h != "" {
		w.setStatus(h)
	} else if w.statusHint != "" {
		w.setStatus("")
	}
}
