// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"cmp"
	"crypto/rand"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io/fs"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/lexington/lex"
	lexreport "github.com/LaPingvino/lexington/report"
)

// The breakdown (docs/PRODUCTION.md §13, step 7): words of the script
// tagged as elements of a category (lex.TagCategories, Final Draft's, and
// one's own), each scene marked done when broken down.
//
// Kept in assets/production/breakdown.json, in the script's extras (the
// bundle's, else the copy Accolade keeps for a plain script, else
// Name.production/ beside it: extras.go), never in the text: a tag is
// the words it is on, found again by them (a Text Fragment directive, as
// the source links are: textfrag.go), in its scene (by its number, else
// its heading):
//
//	{"version": 1,
//	 "categories": [{"id": "…", "name": "Puppets", "color": "#123456"}],
//	 "elements": [{"id": "…", "category": "Props", "name": "revolver"}],
//	 "tags": [{"element": "…", "scene": "12", "heading": "int kitchen night",
//	           "at": "text=puts the-,revolver,-on the table"}],
//	 "done": [{"scene": "12", "heading": "int kitchen night"}]}
//
// So an edit elsewhere, in any program, does not move a tag, and a tag
// whose words are gone is shown as lost (the Production pane's list),
// never dropped by itself. The cast needs no tags: who speaks in a scene
// is its cast (the reports'); a tag in Cast Members adds a silent one.
// Undo does not cover the file: Untag takes a tag away.

const breakdownName = productionDir + "breakdown.json"

type breakdownFile struct {
	Version    int          `json:"version"`
	Categories []bdCategory `json:"categories,omitempty"`
	Elements   []bdElement  `json:"elements"`
	Tags       []bdTag      `json:"tags"`
	Done       []bdSceneKey `json:"done,omitempty"`
}

// bdCategory is a category of one's own.
type bdCategory struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// bdElement is a thing tagged, once for all the places it is.
type bdElement struct {
	ID       string `json:"id"`
	Category string `json:"category"` // the category's name
	Name     string `json:"name"`
}

// bdSceneKey names a scene: its number (when it has one written), its
// heading's words, and which of the scenes with that heading it is.
type bdSceneKey struct {
	Scene   string `json:"scene,omitempty"`
	Heading string `json:"heading"`
	Nth     int    `json:"nth,omitempty"`
}

// bdTag is an element's place: its scene and words.
type bdTag struct {
	Element string `json:"element"`
	bdSceneKey
	At string `json:"at"` // text=prefix-,words,-suffix
}

// newID is a new element's or category's id: a random UUID, lower case,
// as Final Draft's.
func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// nameID is an id made from a name (the same name, the same id): the cast
// written from the cues.
func nameID(name string) string {
	h := sha1.Sum([]byte("accolade cast " + strings.ToUpper(name)))
	h[6] = h[6]&0x0f | 0x50
	h[8] = h[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", h[0:4], h[4:6], h[6:8], h[8:10], h[10:16])
}

// bdCategoryOf is the category of a name: a standard one (by name or
// alias) or one's own.
func (f *breakdownFile) category(name string) (lex.TagCategory, bool) {
	if c, ok := lex.TagCategoryOf(name); ok {
		return c, true
	}
	for _, c := range f.Categories {
		if strings.EqualFold(c.Name, name) || c.ID == name {
			return lex.TagCategory{ID: c.ID, Name: c.Name, Color: c.Color}, true
		}
	}
	return lex.TagCategory{}, false
}

// categories are those one tags in: the standard ones (Synopsis and
// Location aside: the scene's own) then one's own.
func (f *breakdownFile) categories() []lex.TagCategory {
	var out []lex.TagCategory
	for _, c := range lex.TagCategories {
		if c.Name != "Synopsis" && c.Name != "Location" {
			out = append(out, c)
		}
	}
	for _, c := range f.Categories {
		out = append(out, lex.TagCategory{ID: c.ID, Name: c.Name, Color: c.Color})
	}
	return out
}

// element is the element with an id.
func (f *breakdownFile) element(id string) (bdElement, bool) {
	for _, e := range f.Elements {
		if e.ID == id {
			return e, true
		}
	}
	return bdElement{}, false
}

// elementFor is the element of a category with a name, made if new: one
// element, tagged wherever it recurs.
func (f *breakdownFile) elementFor(category, name string) bdElement {
	for _, e := range f.Elements {
		if strings.EqualFold(e.Category, category) && strings.EqualFold(e.Name, name) {
			return e
		}
	}
	e := bdElement{ID: newID(), Category: category, Name: name}
	f.Elements = append(f.Elements, e)
	return e
}

// readBreakdown reads the extras' breakdown (an empty one if none).
func readBreakdown(r extrasReader) (breakdownFile, error) {
	f := breakdownFile{Version: 1}
	data, err := r.readExtra(breakdownName)
	if errors.Is(err, fs.ErrNotExist) || (err == nil && len(data) == 0) {
		return f, nil
	}
	if err != nil {
		return f, err
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return breakdownFile{Version: 1}, fmt.Errorf("%s: %v", breakdownName, err)
	}
	return f, nil
}

// bdScene is a scene of the text, as the breakdown finds it.
type bdScene struct {
	index      int    // its place among the scenes (the report's Order - 1)
	line       int    // its heading's line
	label      string // its number as printed (a counted one if it has none written)
	key        bdSceneKey
	heading    string // as printed: without its number and notes
	start, end int    // its words' range, in runes: from its heading to the next
	omitted    bool
}

// bdWord is a word of the text: normalised (normWords), its runes.
type bdWord struct {
	norm       string
	start, end int
}

// bdText is a text read for the breakdown: its scenes and words.
type bdText struct {
	text      string
	runes     []rune
	lineStart []int // each line's first rune
	scenes    []bdScene
	printed   []rune   // runes with notes blanked (printedRunes)
	words     []bdWord // the body's, as printed
	sceneOf   []int    // each word's scene (-1: before the first)
}

// bdWordRune and isInWordMark are normWords' rules: letters, digits and
// marks make words; quotes and emphasis marks inside one are part of it.
func bdWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r)
}

func isInWordMark(r rune) bool {
	return strings.ContainsRune(`'’‘‛ʼ"“”„«»‹›*_`+"`", r)
}

// wordsIn are the words of runes [from, to).
func wordsIn(rs []rune, from, to int) []bdWord {
	var out []bdWord
	var b strings.Builder
	start, last := -1, -1
	flush := func() {
		if start >= 0 {
			out = append(out, bdWord{norm: b.String(), start: start, end: last + 1})
		}
		b.Reset()
		start = -1
	}
	for i := from; i < to; i++ {
		r := rs[i]
		switch {
		case bdWordRune(r):
			if start < 0 {
				start = i
			}
			b.WriteRune(unicode.ToLower(r))
			last = i
		case isInWordMark(r):
		default:
			flush()
		}
	}
	flush()
	return out
}

// bdHeadingKey is a heading's words, normalised, without its number and
// notes.
func bdHeadingKey(h string) string {
	h = lex.StripNotes(h)
	h, _ = lex.SceneNumber(strings.TrimSpace(h))
	return strings.Join(normWords(h), " ")
}

// readBdText reads a text's scenes and words (the body: not the blocks at
// its end).
func readBdText(text string, starts []string) *bdText {
	t := &bdText{text: text, runes: []rune(text)}
	t.lineStart = []int{0}
	for i, r := range t.runes {
		if r == '\n' {
			t.lineStart = append(t.lineStart, i+1)
		}
	}
	body := utf8.RuneCountInString(text[:bodyEndByte(text)])
	heads := sceneHeads(text[:bodyEndByte(text)], starts)
	o := lex.NumberOptions{}
	if l := sceneLockOf(text); l.locked {
		o = lex.NumberOptions{Locked: true, Style: l.style}
	}
	labels := sceneNumbersFor(heads, o)
	nth := map[string]int{}
	for i, h := range heads {
		k := bdHeadingKey(h.bare(starts))
		s := bdScene{index: i, line: h.line, label: labels[i], heading: printedHeading(h.text),
			key: bdSceneKey{Scene: h.num, Heading: k, Nth: nth[k]}, start: t.lineStart[h.line], end: body}
		s.omitted = strings.HasPrefix(strings.TrimSpace(strings.ToUpper(h.text)), ".OMITTED")
		nth[k]++
		if i > 0 {
			t.scenes[i-1].end = s.start
		}
		t.scenes = append(t.scenes, s)
	}
	t.printed = printedRunes(t.runes)
	t.words = wordsIn(t.printed, 0, min(body, len(t.runes)))
	t.sceneOf = make([]int, len(t.words))
	sc := -1
	for i, w := range t.words {
		for sc+1 < len(t.scenes) && w.start >= t.scenes[sc+1].start {
			sc++
		}
		t.sceneOf[i] = sc
	}
	return t
}

// printedRunes is a text with its notes ([[…]]: source links too)
// blanked, the same length: a tag is on the script's words, never on a
// link's quote, and is lost when those are gone. A note not closed within
// its paragraph is left as it is (being typed). The boneyard stays: an
// omitted scene's text kept there keeps its tags.
func printedRunes(rs []rune) []rune {
	out := rs
	copied := false
	blank := func(from, to int) {
		if !copied {
			out, copied = slices.Clone(rs), true
		}
		for i := from; i < to; i++ {
			if out[i] != '\n' {
				out[i] = ' '
			}
		}
	}
	at := func(i int, s string) bool {
		return i+1 < len(rs) && rs[i] == rune(s[0]) && rs[i+1] == rune(s[1])
	}
	for i := 0; i+1 < len(rs); i++ {
		if !at(i, "[[") {
			continue
		}
		for j := i + 2; j+1 < len(rs); j++ {
			if at(j, "\n\n") {
				break // a note ends with its paragraph
			}
			if at(j, "]]") {
				blank(i, j+2)
				i = j + 1
				break
			}
		}
	}
	return out
}

// printedHeading is a heading as printed: without its number, notes and
// a forcing dot.
func printedHeading(h string) string {
	h = strings.TrimSpace(lex.StripNotes(h))
	h, _ = lex.SceneNumber(h)
	return strings.TrimPrefix(strings.TrimSpace(h), ".")
}

// sceneAt is the scene a rune offset is in (-1: before the first).
func (t *bdText) sceneAt(off int) int {
	for i := len(t.scenes) - 1; i >= 0; i-- {
		if off >= t.scenes[i].start {
			return i
		}
	}
	return -1
}

// lineCol is a rune offset's line and column.
func (t *bdText) lineCol(off int) (line, col int) {
	line, _ = slices.BinarySearch(t.lineStart, off+1)
	line--
	return line, off - t.lineStart[line]
}

// sceneWords are the indexes of a scene's words [i, j).
func (t *bdText) sceneWords(sc int) (int, int) {
	i, _ := slices.BinarySearchFunc(t.words, t.scenes[sc].start, func(w bdWord, off int) int { return cmp.Compare(w.start, off) })
	j, _ := slices.BinarySearchFunc(t.words, t.scenes[sc].end, func(w bdWord, off int) int { return cmp.Compare(w.start, off) })
	return i, j
}

// contextWords is how many words before and after a tag's anchor keeps.
const contextWords = 3

// anchorOf is the anchor of words [i, j) of the text: their scene, the
// words as written, and the words around them in the scene.
func (t *bdText) anchorOf(i, j int) (bdSceneKey, string) {
	sc := t.sceneOf[i]
	var key bdSceneKey
	lo, hi := 0, len(t.words)
	if sc >= 0 {
		key = t.scenes[sc].key
		lo, hi = t.sceneWords(sc)
	}
	raw := func(a, b int) string { // words [a, b) as written, on one line
		if a >= b {
			return ""
		}
		return strings.Join(strings.Fields(string(t.printed[t.words[a].start:t.words[b-1].end])), " ")
	}
	d := textDirective{Start: raw(i, j)}
	if p := max(lo, i-contextWords); p < i {
		d.Prefix = raw(p, i)
	}
	if s := min(hi, j+contextWords); j < s {
		d.Suffix = raw(j, s)
	}
	return key, d.String(false)
}

// inNote reports whether runes [start, end) have a note's (a link's
// quote): no words to tag.
func (t *bdText) inNote(start, end int) bool {
	for i := max(start, 0); i < min(max(end, start+1), len(t.runes)); i++ {
		if t.printed[i] != t.runes[i] {
			return true
		}
	}
	return false
}

// placed is a tag found in the text: its words' runes (lost: not found).
type placed struct {
	tag        bdTag
	element    bdElement
	scene      int // the scene it is in (-1: lost, or before the first)
	start, end int
	lost       bool
	words      string // its words, as anchored
}

// locate finds a tag's words: in its scene (by number, else by heading
// and which of those it is, else any with that heading), else anywhere;
// of the places with the words, the one with the most words around it as
// anchored (the first of equals).
func (t *bdText) locate(tag bdTag) (start, end, scene int, ok bool) {
	d, bad := parseTextDirective(strings.TrimPrefix(tag.At, "text="))
	if bad != "" {
		return 0, 0, -1, false
	}
	want, pre, suf := normWords(d.Start), normWords(d.Prefix), normWords(d.Suffix)
	if len(want) == 0 {
		return 0, 0, -1, false
	}
	// its own scene: by its number (a number given later, by Lock, is
	// none: then the heading and which of those it is)
	numbered := false
	for _, s := range t.scenes {
		numbered = numbered || (tag.Scene != "" && s.key.Scene == tag.Scene)
	}
	own := func(s bdScene) bool {
		if numbered {
			return s.key.Scene == tag.Scene
		}
		return s.key.Heading == tag.Heading && s.key.Nth == tag.Nth
	}
	type cand struct {
		i, score int
		own      bool
	}
	// best is the place with the words with the most words around them
	// as anchored, of the scenes; of equals, one in its own scene
	best := func(scenes []int) (cand, bool) {
		c := cand{i: -1, score: -1}
		for _, sc := range scenes {
			lo, hi := t.sceneWords(sc)
			isOwn := own(t.scenes[sc])
			for i := lo; i+len(want) <= hi; i++ {
				if !t.wordsAt(i, want) {
					continue
				}
				score := 0
				for k := 1; k <= len(pre) && i-k >= lo && t.words[i-k].norm == pre[len(pre)-k]; k++ {
					score++
				}
				for k := 0; k < len(suf) && i+len(want)+k < hi && t.words[i+len(want)+k].norm == suf[k]; k++ {
					score++
				}
				if score > c.score || (score == c.score && isOwn && !c.own) {
					c = cand{i, score, isOwn}
				}
			}
		}
		return c, c.i >= 0
	}
	// told apart: in its own scene, or with two words around it as anchored,
	// or more than a word that stand once in the script (a single word
	// elsewhere is another place: the tag is lost, not moved)
	var ownScenes, sameHeading, all []int
	headings := 0
	for i, s := range t.scenes {
		all = append(all, i)
		if own(s) {
			ownScenes = append(ownScenes, i)
		}
		if s.key.Heading == tag.Heading {
			headings++
		}
		if s.key.Heading == tag.Heading || own(s) {
			sameHeading = append(sameHeading, i)
		}
	}
	// its own scene is sure by its number, or by a heading no other
	// scene has: which of several it is can shift (a scene put before)
	sure := numbered || headings == 1
	accept := func(c cand) bool {
		return (c.own && sure) || c.score >= 2 || (len(want) > 1 && t.count(want) == 1)
	}
	// its own scene when numbered (the heading may have changed); else
	// the scenes with its heading together (a scene with that heading put
	// before it shifts which it is); else anywhere (moved)
	groups := [][]int{sameHeading, all}
	if numbered {
		groups = [][]int{ownScenes, sameHeading, all}
	}
	for _, g := range groups {
		if c, ok := best(g); ok && accept(c) {
			return t.words[c.i].start, t.words[c.i+len(want)-1].end, t.sceneOf[c.i], true
		}
	}
	return 0, 0, -1, false
}

// wordsAt tells whether the words from i are want.
func (t *bdText) wordsAt(i int, want []string) bool {
	if i+len(want) > len(t.words) {
		return false
	}
	for k, w := range want {
		if t.words[i+k].norm != w {
			return false
		}
	}
	return true
}

// count is how often the words stand in the body.
func (t *bdText) count(want []string) int {
	n := 0
	for i := range t.words {
		if t.wordsAt(i, want) {
			n++
		}
	}
	return n
}

// place finds every tag of a breakdown in the text.
func (t *bdText) place(f *breakdownFile) []placed {
	out := make([]placed, 0, len(f.Tags))
	for _, tag := range f.Tags {
		e, _ := f.element(tag.Element)
		p := placed{tag: tag, element: e, scene: -1}
		if d, bad := parseTextDirective(strings.TrimPrefix(tag.At, "text=")); bad == "" {
			p.words = d.Start
		}
		if start, end, sc, ok := t.locate(tag); ok {
			p.start, p.end, p.scene = start, end, sc
		} else {
			p.lost = true
		}
		out = append(out, p)
	}
	return out
}

// wordRange is the words a selection [start, end) of runes touches,
// [i, j) (none: i == j): with nothing selected, the word at the cursor.
func (t *bdText) wordRange(start, end int) (int, int) {
	if start > end {
		start, end = end, start
	}
	i := -1
	j := -1
	for k, w := range t.words {
		touches := w.end > start && w.start < end
		if start == end {
			touches = w.start <= start && start <= w.end
		}
		if touches {
			if i < 0 {
				i = k
			}
			j = k + 1
		} else if i >= 0 {
			break
		}
	}
	if i < 0 {
		return 0, 0
	}
	return i, j
}

// tagWords tags words [i, j) of the text as an element of category: the
// element named by the words (made if new) and the tag, both added to f.
// It tells which element and whether the tag is new (the same element
// on the same words is not tagged twice).
func (f *breakdownFile) tagWords(t *bdText, i, j int, category string) (bdElement, bool) {
	name := strings.Join(strings.Fields(string(t.runes[t.words[i].start:t.words[j-1].end])), " ")
	name = strings.Trim(name, `*_"“”'’‘`)
	if c, ok := lex.TagCategoryOf(category); ok {
		category = c.Name
		if c.Name == lex.CastCategory {
			name = strings.ToUpper(name)
		}
	} else if c, ok := f.category(category); ok {
		category = c.Name
	} else { // a category of one's own, new
		f.Categories = append(f.Categories, bdCategory{ID: newID(), Name: category,
			Color: ownCategoryColours[len(f.Categories)%len(ownCategoryColours)]})
	}
	e := f.elementFor(category, name)
	key, at := t.anchorOf(i, j)
	for _, p := range t.place(f) {
		if p.tag.Element == e.ID && !p.lost && p.start == t.words[i].start && p.end == t.words[j-1].end {
			return e, false
		}
	}
	f.Tags = append(f.Tags, bdTag{Element: e.ID, bdSceneKey: key, At: at})
	return e, true
}

// untag takes away the tags placed over [start, end) of runes (with
// nothing selected: at the cursor); it tells how many. An element left
// with no tag goes too.
func (f *breakdownFile) untag(t *bdText, start, end int) int {
	if start > end {
		start, end = end, start
	}
	placedTags := t.place(f)
	var keep []bdTag
	n := 0
	for k, p := range placedTags {
		hit := !p.lost && p.end > start && p.start < end
		if start == end {
			hit = !p.lost && p.start <= start && start <= p.end
		}
		if hit {
			n++
			continue
		}
		keep = append(keep, f.Tags[k])
	}
	f.Tags = keep
	f.pruneElements()
	return n
}

// pruneElements drops the elements no tag is of.
func (f *breakdownFile) pruneElements() {
	used := map[string]bool{}
	for _, t := range f.Tags {
		used[t.Element] = true
	}
	f.Elements = slices.DeleteFunc(f.Elements, func(e bdElement) bool { return !used[e.ID] })
}

// isDone tells whether a scene is marked done.
func (f *breakdownFile) isDone(s bdScene) bool {
	for _, d := range f.Done {
		if d.Scene != "" && d.Scene == s.key.Scene {
			return true
		}
		if (d.Scene == "" || s.key.Scene == "") && d.Heading == s.key.Heading && d.Nth == s.key.Nth {
			return true
		}
	}
	return false
}

// setDone marks a scene done, or not.
func (f *breakdownFile) setDone(s bdScene, done bool) {
	f.Done = slices.DeleteFunc(f.Done, func(d bdSceneKey) bool {
		return (d.Scene != "" && d.Scene == s.key.Scene) ||
			((d.Scene == "" || s.key.Scene == "") && d.Heading == s.key.Heading && d.Nth == s.key.Nth)
	})
	if done {
		f.Done = append(f.Done, s.key)
	}
}

// bdProgress is how far the breakdown is: the scenes done of those to do
// (OMITTED ones aside), and the first not done.
type bdProgress struct {
	done, total int
	next        int // the first scene not done (-1: none)
}

func (f *breakdownFile) progress(t *bdText) bdProgress {
	p := bdProgress{next: -1}
	for i, s := range t.scenes {
		if s.omitted {
			continue
		}
		p.total++
		if f.isDone(s) {
			p.done++
		} else if p.next < 0 {
			p.next = i
		}
	}
	return p
}

// percent is the part done, rounded down (100 only when all are).
func (p bdProgress) percent() int {
	if p.total == 0 {
		return 0
	}
	return p.done * 100 / p.total
}

// mixInk is a category's colour as the editor draws words in it: mixed
// a little towards the text's colour, so that light ones (Camera's
// yellow-green) read on a light page and dark ones on a dark.
func mixInk(c, fg color.Color) color.Color {
	r, g, b, _ := c.RGBA()
	fr, fgc, fb, _ := fg.RGBA()
	mix := func(x, y uint32) uint8 { return uint8(float64(x>>8)*0.7 + float64(y>>8)*0.3) }
	return color.NRGBA{R: mix(r, fr), G: mix(g, fgc), B: mix(b, fb), A: 0xff}
}

// parseHexRGB reads #RRGGBB.
func parseHexRGB(s string) (r, g, b uint8, ok bool) {
	var v uint32
	if _, err := fmt.Sscanf(strings.TrimPrefix(s, "#"), "%06x", &v); err != nil || len(strings.TrimPrefix(s, "#")) != 6 {
		return 0, 0, 0, false
	}
	return uint8(v >> 16), uint8(v >> 8), uint8(v), true
}

// inksOf are the editor's inks of the tags placed: each tag's words, by
// line, in its category's colour.
func inksOf(t *bdText, f *breakdownFile, ps []placed, ink func(lex.TagCategory) color.Color) map[int][]editor.Ink {
	out := map[int][]editor.Ink{}
	for _, p := range ps {
		if p.lost {
			continue
		}
		c, _ := f.category(p.element.Category)
		col := ink(c)
		for off := p.start; off < p.end; {
			line, colStart := t.lineCol(off)
			lineEnd := len(t.runes)
			if line+1 < len(t.lineStart) {
				lineEnd = t.lineStart[line+1] - 1
			}
			end := min(p.end, lineEnd)
			out[line] = append(out[line], editor.Ink{Start: colStart, End: colStart + end - off, Color: col})
			off = end + 1
		}
	}
	return out
}

// reportScenes maps the text's scenes to the report's (-1: none): one
// to one when there are as many, else by their numbers, in order.
func reportScenes(t *bdText, rep *lexreport.Report) []int {
	out := make([]int, len(t.scenes))
	if rep == nil {
		for i := range out {
			out[i] = -1
		}
		return out
	}
	if len(rep.Scenes) == len(t.scenes) {
		for i := range out {
			out[i] = i
		}
		return out
	}
	j := 0
	for i, s := range t.scenes {
		out[i] = -1
		for k := j; k < len(rep.Scenes); k++ {
			if rep.Scenes[k].ID() == s.label {
				out[i], j = k, k+1
				break
			}
		}
	}
	return out
}

// reportBreakdown is the breakdown as Lexington's report takes it: each
// scene's elements by its Order (the text's scenes mapped to the
// report's by reportScenes).
func reportBreakdownFor(f *breakdownFile, ps []placed, t *bdText, rep *lexreport.Report) lexreport.Breakdown {
	m := reportScenes(t, rep)
	moved := make([]placed, 0, len(ps))
	for _, p := range ps {
		if p.scene >= 0 && p.scene < len(m) {
			p.scene = m[p.scene]
		}
		if p.scene >= 0 {
			moved = append(moved, p)
		}
	}
	return reportBreakdown(f, moved)
}

// reportBreakdown is the breakdown as Lexington's report takes it: each
// scene's elements by its Order, the tags' scenes being the report's.
func reportBreakdown(f *breakdownFile, ps []placed) lexreport.Breakdown {
	b := lexreport.Breakdown{Elements: map[int][]lexreport.Element{}}
	for _, c := range f.Categories {
		b.Categories = append(b.Categories, lex.TagCategory{ID: c.ID, Name: c.Name, Color: c.Color})
	}
	for _, p := range ps {
		if p.lost || p.scene < 0 {
			continue
		}
		c, ok := f.category(p.element.Category)
		name := p.element.Category
		if ok {
			name = c.Name
		}
		b.Elements[p.scene+1] = append(b.Elements[p.scene+1], lexreport.Element{Category: name, Name: p.element.Name})
	}
	return b
}
