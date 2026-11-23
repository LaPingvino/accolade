package main

import (
	"archive/zip"
	"bytes"
	"html"
	"io"
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode"
)

// The grammar: what is written reads back the same, encoded or plain.
func TestTextDirectiveRoundTrip(t *testing.T) {
	for _, d := range []textDirective{
		{Start: "She opened the door"},
		{Start: "She opened the door", End: "stepped out"},
		{Prefix: "said Anna.", Start: "Yes.", Suffix: "She left"},
		{Prefix: "a", Start: "b", End: "c", Suffix: "d"},
		{Start: "well-known, 100% & more", End: "the end -- or not"},
		{Start: "[[nested]] notes", End: "مِصر"},
	} {
		for _, href := range []bool{false, true} {
			s := fragmentString("", []textDirective{d}, href)
			_, dirs, bad := parseFragment(s)
			if bad != "" || len(dirs) != 1 || dirs[0] != d {
				t.Errorf("%+v (href %v) as %q read back %+v %q", d, href, s, dirs, bad)
			}
			if !href && (strings.Contains(s, "%20") || strings.Contains(s, "]]")) {
				t.Errorf("not readable: %q", s)
			}
		}
	}
	// readable: only , & % [ ] encoded, and a dash at a term's edges (where
	// it would read as the prefix's or suffix's mark), not inside a word
	if got := encodeTerm("a-b, c & 5% [x]"); got != "a-b%2C c %26 5%25 %5Bx%5D" {
		t.Errorf("encoded %q", got)
	}
	for in, want := range map[string]string{
		"the timber-yard": "the timber-yard", "-yard": "%2Dyard", "timber-": "timber%2D",
		" - a dash - ": " %2D a dash %2D ", "-": "%2D", "a--": "a-%2D",
	} {
		if got := encodeTerm(in); got != want {
			t.Errorf("encoded %q as %q, want %q", in, got, want)
		}
	}
	// the edges' dashes read back as text; an old link's %2D inside a word too
	for _, d := range []textDirective{{Prefix: "timber-", Start: "-yard", End: "a - b-", Suffix: "-x"}} {
		s := fragmentString("", []textDirective{d}, false)
		if _, dirs, bad := parseFragment(s); bad != "" || len(dirs) != 1 || dirs[0] != d {
			t.Errorf("%+v as %q read back %+v %q", d, s, dirs, bad)
		}
	}
	if _, dirs, _ := parseFragment("#:~:text=the timber%2Dyard"); len(dirs) != 1 || dirs[0].Start != "the timber-yard" {
		t.Errorf("old %%2D: %+v", dirs)
	}
	if got := encodeTermHref("a b,é"); got != "a%20b%2C%C3%A9" {
		t.Errorf("href %q", got)
	}
	// the spec's own: an Arabic word, percent-encoded as UTF-8
	if got := decodeTerm("%D9%85%D8%B5%D8%B1"); got != "مصر" {
		t.Errorf("decoded %q", got)
	}
	// a stray % stays; bytes that are no UTF-8 too
	if decodeTerm("100% sure") != "100% sure" || decodeTerm("%FF%FE") != "%FF%FE" || decodeTerm("%4") != "%4" {
		t.Error("a stray % changed")
	}
	// several text= joined by &, a file in front
	file, dirs, bad := parseFragment("story.md#:~:text=one,two&text=three-,four&text=five,-six")
	if file != "story.md" || bad != "" || !slices.Equal(dirs, []textDirective{{Start: "one", End: "two"},
		{Prefix: "three", Start: "four"}, {Start: "five", Suffix: "six"}}) {
		t.Errorf("several: %q %+v %q", file, dirs, bad)
	}
	if s := fragmentString("story.md", dirs, false); s != "story.md#:~:text=one,two&text=three-,four&text=five,-six" {
		t.Errorf("written %q", s)
	}
	// plain: spaces kept or encoded, a & in the text, a dash inside a word
	_, dirs, _ = parseFragment("#:~:text=Tom & Jerry ran,away")
	if len(dirs) != 1 || dirs[0] != (textDirective{Start: "Tom & Jerry ran", End: "away"}) {
		t.Errorf("a plain &: %+v", dirs)
	}
	_, dirs, _ = parseFragment("#:~:text=a%20well-known%20tale")
	if len(dirs) != 1 || dirs[0].Start != "a well-known tale" {
		t.Errorf("a plain dash: %+v", dirs)
	}
}

// The spec's "parse a text directive", its cases.
func TestParseTextDirective(t *testing.T) {
	for v, want := range map[string]textDirective{
		"prefix-,foo,bar": {Prefix: "prefix", Start: "foo", End: "bar"},
		"foo,-bar":        {Start: "foo", Suffix: "bar"},
		"an%20example":    {Start: "an example"},
		"a-,b,c,-d":       {Prefix: "a", Start: "b", End: "c", Suffix: "d"},
		"this%20is-,an%20example,-text%20fragment": {Prefix: "this is", Start: "an example", Suffix: "text fragment"},
	} {
		if d, why := parseTextDirective(v); why != "" || d != want {
			t.Errorf("%q: %+v %q", v, d, why)
		}
	}
	for _, v := range []string{"", "-foo,bar", "foo,", "a-,-b", "a,b,c", "a-,b,c,d,-e", "a-", "-a", "a,-"} {
		if d, why := parseTextDirective(v); why == "" {
			t.Errorf("%q read as %+v", v, d)
		}
	}
	// other directives: ignored; none at all: a problem shown
	for _, f := range []string{"#:~:other=x&text=foo", "#:~:text=foo&other=x"} {
		if _, dirs, bad := parseFragment(f); bad != "" || len(dirs) != 1 || dirs[0].Start != "foo" {
			t.Errorf("unknown directive in %q: %+v %q", f, dirs, bad)
		}
	}
	links := parseLinks("INT. A\n\n[[from: #:~:text=-oops]] [[from: #:~:nothing]]\n")
	if len(links) != 2 || links[0].Ref.Bad == "" || links[1].Ref.Bad == "" {
		t.Fatalf("bad links: %+v", links)
	}
	if mt := newLinkMatcher(nil).matchRef(links[0].Ref); mt.Kind != linkMissing {
		t.Errorf("a bad link matched: %+v", mt)
	}
}

// fragMatch is a directive matched on a source's passages.
func fragMatch(src string, d textDirective) linkMatch {
	return newLinkMatcher(passagesOf(src)).matchRef(linkRef{Dir: d})
}

// The spec's matching examples, on passages (one block each).
func TestFragmentSpecExamples(t *testing.T) {
	ctx := textDirective{Prefix: "this is", Start: "an example", Suffix: "text fragment"}
	if mt := fragMatch("this is an example text fragment\n", ctx); mt.Kind != linkFound {
		t.Errorf("with context: %+v", mt)
	}
	if mt := fragMatch("here is an example text\n", ctx); mt.found() && mt.Kind == linkFound {
		t.Errorf("other context found: %+v", mt)
	}
	// word bounded
	if mt := fragMatch("An impressive mountain range\n", textDirective{Start: "mountain range"}); mt.Kind != linkFound {
		t.Errorf("mountain range: %+v", mt)
	}
	if mt := fragMatch("An impressive mountain ranger\n", textDirective{Start: "mountain range"}); mt.Kind != linkMissing {
		t.Errorf("mountain ranger: %+v", mt)
	}
	for _, s := range []string{"color orange\n", "forest ranger\n"} {
		if mt := fragMatch(s, textDirective{Start: "range"}); mt.Kind != linkMissing {
			t.Errorf("range in %q: %+v", s, mt)
		}
	}
	// a term within one block: "The quick" apart in two blocks is not found
	if mt := fragMatch("# The\n\nquick brown fox jumps over the lazy dog\n", textDirective{Start: "The quick", End: "lazy dog"}); mt.Kind != linkMissing {
		t.Errorf("across blocks: %+v", mt)
	}
	if mt := fragMatch("The quick brown fox jumps over the lazy dog\n", textDirective{Start: "The quick", End: "lazy dog"}); mt.Kind != linkFound {
		t.Errorf("in one block: %+v", mt)
	}
	// textStart anywhere in a passage, not only at its start
	if mt := fragMatch("One.\n\nAnna opened the door and went out.\n", textDirective{Start: "the door"}); mt.Kind != linkFound || mt.Passages[0] != 1 {
		t.Errorf("mid-passage: %+v", mt)
	}
	// case and punctuation aside (the spec: case and accents; here quotes and punctuation too)
	if mt := fragMatch("“Don’t,” she said.\n", textDirective{Start: `"don't" SHE said`}); mt.Kind != linkFound {
		t.Errorf("normalised: %+v", mt)
	}
}

// Prefix and suffix tell passages that read alike apart; in the passage
// before or after too, across a break.
func TestFragmentDisambiguation(t *testing.T) {
	src := "Anna asked again.\n\nYes.\n\nTom shrugged.\n\n* * *\n\nYes.\n\nAnd so it was.\n"
	ps := passagesOf(src)
	lm := newLinkMatcher(ps)
	if mt := lm.matchRef(linkRef{Dir: textDirective{Start: "Yes."}}); mt.Kind != linkAmbiguous || !slices.Equal(mt.Passages, []int{1, 4}) {
		t.Errorf("twice: %+v", mt)
	}
	if mt := lm.matchRef(linkRef{Dir: textDirective{Prefix: "again.", Start: "Yes."}}); mt.Kind != linkFound || mt.Passages[0] != 1 {
		t.Errorf("by prefix: %+v", mt)
	}
	if mt := lm.matchRef(linkRef{Dir: textDirective{Prefix: "shrugged.", Start: "Yes."}}); mt.Kind != linkFound || mt.Passages[0] != 4 {
		t.Errorf("by prefix across the break: %+v", mt)
	}
	if mt := lm.matchRef(linkRef{Dir: textDirective{Start: "Yes.", Suffix: "And so"}}); mt.Kind != linkFound || mt.Passages[0] != 4 {
		t.Errorf("by suffix: %+v", mt)
	}
	// a prefix that is not right before it: none of the two
	if mt := lm.matchRef(linkRef{Dir: textDirective{Prefix: "Anna", Start: "Yes."}}); mt.Kind == linkFound {
		t.Errorf("a prefix apart: %+v", mt)
	}
	// written: the prefix only when needed, and the first passage by its suffix
	if d, u := fragmentFor(lm, 4, 4); !u || d != (textDirective{Prefix: "shrugged.", Start: "Yes."}) {
		t.Errorf("written for the second: %+v %v", d, u)
	}
	src2 := "Yes.\n\nOne.\n\nYes.\n\nTwo.\n"
	lm2 := newLinkMatcher(passagesOf(src2))
	if d, u := fragmentFor(lm2, 0, 0); !u || d != (textDirective{Start: "Yes.", Suffix: "One."}) {
		t.Errorf("the first passage: %+v %v", d, u)
	}
	// twins whose neighbours are alike too: no link tells them apart
	src3 := "Rain.\n\nYes.\n\nRain.\n\nRain.\n\nYes.\n\nRain.\n"
	lm3 := newLinkMatcher(passagesOf(src3))
	d, u := fragmentFor(lm3, 1, 1)
	if u {
		t.Errorf("twins told apart: %+v", d)
	}
	if mt := lm3.matchRef(linkRef{Dir: d}); mt.Kind != linkAmbiguous {
		t.Errorf("the twins' link: %+v", mt)
	}
	// cut, they are told apart in the old form's way, "(2)", still read
	ps3 := passagesOf(src3)
	for _, i := range []int{1, 4} {
		if c := cutFor(ps3, i); !c.Ref.Old || resolveCut(c, ps3) != (cutSpan{i, i}) {
			t.Errorf("twin %d cut as %q", i, c.String())
		}
	}
	if c := cutFor(ps3, 4); c.String() != `"Yes." (2)` {
		t.Errorf("the second twin's entry: %q", c.String())
	}
}

// A link spanning passages: from textStart's to textEnd's, the first end
// after the start; all of them adapted.
func TestFragmentRange(t *testing.T) {
	src := "# One\n\nShe opened the door to the garden.\n\nThe wind came in.\n\nShe stepped out into the rain.\n\nLater, the door shut.\n"
	ps := passagesOf(src)
	lm := newLinkMatcher(ps)
	mt := lm.matchRef(linkRef{Dir: textDirective{Start: "She opened the door", End: "stepped out"}})
	if mt.Kind != linkFound || mt.Passages[0] != 1 || mt.Last != 3 || !mt.covers(2) || mt.covers(4) {
		t.Fatalf("range: %+v", mt)
	}
	// the end: the first after the start, though it is in the start's passage
	if mt := lm.matchRef(linkRef{Dir: textDirective{Start: "She opened", End: "door"}}); mt.Kind != linkFound || mt.Last != 1 {
		t.Errorf("end in the start's passage: %+v", mt)
	}
	// with a suffix: the first end the suffix follows
	if mt := lm.matchRef(linkRef{Dir: textDirective{Start: "She opened", End: "the door", Suffix: "shut"}}); mt.Kind != linkFound || mt.Last != 4 {
		t.Errorf("end by its suffix: %+v", mt)
	}
	script := "INT. GARDEN - DAY\n\n[[from: #:~:text=She opened the door,stepped out]]\n"
	m := newSourceModel(ps, script, nil)
	if !slices.Equal(m.status, []passageStatus{statusNone, statusAdapted, statusAdapted, statusAdapted, statusToDo}) {
		t.Errorf("statuses %v", m.status)
	}
	if !slices.Equal(m.linked(2), []int{0}) || m.position(2) != 3 {
		t.Errorf("linked %v, position %d", m.linked(2), m.position(2))
	}
	// written: the first passage's words to the last's
	d, u := fragmentFor(lm, 1, 3)
	if !u || d.Start != "She opened" || d.End != "rain." { // "She" alone opens a passage before the last: a range of its own
		t.Errorf("written range %+v %v", d, u)
	}
	// suggestions of neighbours are one range (a heading between is next to)
	if runs := runsOf(ps, []int{3, 1, 2, 4}); !slices.Equal(runs, [][2]int{{1, 4}}) {
		t.Errorf("runs %v", runs)
	}
	if runs := runsOf(passagesOf("# A\n\nOne.\n\n# B\n\nTwo.\n\nThree.\n\nFour.\n"), []int{1, 3, 5}); !slices.Equal(runs, [][2]int{{1, 3}, {5, 5}}) {
		t.Errorf("runs with a gap %v", runs)
	}
	// several text= in one note: one link each, at the note's place
	links := parseLinks("INT. A\n\n[[from: #:~:text=She opened&text=Later,shut]]\n")
	if len(links) != 2 || links[0].Start != links[1].Start || links[1].Ref.Dir.End != "shut" {
		t.Errorf("two directives: %+v", links)
	}
}

// Near: a few words edited in textStart (as the old form's), textEnd
// edited, or gone (the start's passage alone); flagged near.
func TestFragmentNear(t *testing.T) {
	src := "She opened the door to the garden and the wind came in.\n\nThen she stepped out into the cold and the rain.\n\nNothing else.\n"
	lm := newLinkMatcher(passagesOf(src))
	for name, c := range map[string]struct {
		d     textDirective
		first int
		last  int
	}{
		"start edited": {textDirective{Start: "She opened a door to the garden and", End: "cold and the rain."}, 0, 1},
		"end edited":   {textDirective{Start: "She opened the door", End: "into the cold and rain"}, 0, 1},
		"end gone":     {textDirective{Start: "She opened the door", End: "zzz yyy"}, 0, 0},
		"context gone": {textDirective{Prefix: "Before all", Start: "She opened the door"}, 0, 0},
	} {
		if mt := lm.matchRef(linkRef{Dir: c.d}); mt.Kind != linkNear || mt.Passages[0] != c.first || mt.last() != c.last {
			t.Errorf("%s: %+v", name, mt)
		}
	}
	if mt := lm.matchRef(linkRef{Dir: textDirective{Start: "Quite another thing altogether"}}); mt.Kind != linkMissing {
		t.Errorf("missing: %+v", mt)
	}
}

// The old form reads as before, beside the new.
func TestOldFormStillRead(t *testing.T) {
	ps := passagesOf(sourceStory)
	script := "INT. A - DAY\n\n[[from: \"She opened the door to the garden and…\"]]\n\n[[from: #:~:text=She opened the door to the garden and,out into the rain.]]\n"
	m := newSourceModel(ps, script, nil)
	if len(m.links) != 2 || !m.links[0].Ref.Old || m.links[1].Ref.Old {
		t.Fatalf("links %+v", m.links)
	}
	if m.matches[0].Kind != linkFound || m.matches[1].Kind != linkFound || m.matches[0].Passages[0] != m.matches[1].Passages[0] {
		t.Errorf("matches %+v", m.matches)
	}
	if m.links[0].Key == m.links[1].Key {
		t.Error("the forms share a key")
	}
	// the old form only opens a passage: words from its middle are not it
	if mt := newLinkMatcher(ps).matchRef(linkRef{Old: true, Quote: "to the garden and"}); mt.Kind != linkMissing {
		t.Errorf("old form mid-passage: %+v", mt)
	}
}

// The block's entries in the new form: cut:, skipped links:, read and
// written; the old kept.
func TestFragmentBlockEntries(t *testing.T) {
	inside := "cut:\n#:~:text=The rain kept falling.\n  #:~:text=-bad\n  story.md#:~:text=open.-,The rain kept falling.\n  \"Old words\" (2)\n" +
		"skipped links:\n  INT. KITCHEN - NIGHT = #:~:text=a=b,c\n  EXT. YARD→#:~:text=Anna%2C at last\n  INT. HALL = \"Old quote\"\nother: kept"
	cuts := parseCuts(inside)
	if len(cuts) != 4 || cuts[0].Ref.Dir.Start != "The rain kept falling." || cuts[1].Ref.Bad == "" || cuts[2].Ref.File != "story.md" ||
		cuts[2].Ref.Dir.Prefix != "open." || !cuts[3].Ref.Old || cuts[3].Nth != 2 {
		t.Fatalf("cuts %+v", cuts)
	}
	if got := withCuts(inside, cuts); !strings.Contains(got, "cut:\n  #:~:text=The rain kept falling.\n  #:~:text=-bad\n  story.md#:~:text=open.-,The rain kept falling.\n  \"Old words\" (2)\n") ||
		!strings.Contains(got, "other: kept") {
		t.Errorf("written:\n%s", got)
	}
	if cs := parseCuts("cut: #:~:text=one; #:~:text=two; three\n"); len(cs) != 2 || cs[1].Ref.Dir.Start != "two; three" {
		t.Errorf("on the key's line: %+v", cs)
	}
	l := parseLearned(inside)
	if len(l.Skipped) != 3 || l.Skipped[0].Heading != "INT. KITCHEN - NIGHT" || l.Skipped[0].Link.Ref.Dir != (textDirective{Start: "a=b", End: "c"}) ||
		l.Skipped[1].Heading != "EXT. YARD" || l.Skipped[1].Link.Ref.Dir.Start != "Anna, at last" || !l.Skipped[2].Link.Ref.Old {
		t.Fatalf("skipped %+v", l.Skipped)
	}
	got := withLearned(inside, l)
	if !strings.Contains(got, "  INT. KITCHEN - NIGHT = #:~:text=a=b,c\n  EXT. YARD = #:~:text=Anna%2C at last\n  INT. HALL = \"Old quote\"") {
		t.Errorf("skipped written:\n%s", got)
	}
	// resolved: a range cut cuts each
	ps := passagesOf("One two three.\n\nFour five.\n\nSix.\n")
	m := newSourceModel(ps, "INT. A\n\n/* Adaptation\ncut:\n  #:~:text=One two,five\n*/\n", nil)
	if !slices.Equal(m.status, []passageStatus{statusCut, statusCut, statusToDo}) {
		t.Errorf("range cut %v", m.status)
	}
}

// Adaptation > Update Source Links: old links and entries in the new form,
// for the same passages, one undo step, a near one re-anchored; the
// ambiguous and missing left.
func TestUpdateSourceLinks(t *testing.T) {
	story := "# The Garden\n\nShe opened the door to the garden and the wind came in from the sea.\n\nThe rain kept falling.\n\nIt was late.\n\nThe rain kept falling.\n\n“Who's there?” Anna called, but nobody answered.\n"
	script := "INT. HOUSE - NIGHT\n\nAnna. [[from: \"She opened the door to the garden and…\"]]\n\n" +
		"[[from: \"The rain kept falling.\"]]\n\n[[from: \"Who’s there? Anna called, but someone answered.\"]]\n\n[[from: \"Not in the story at all\"]]\n\n" +
		"/* Adaptation\nskipped links:\n  INT. HOUSE - NIGHT = \"It was late.\"\ncut:\n  \"The rain kept falling.\" (2)\n*/\n"
	w, _ := sourceWindow(t, script, story)
	s := w.sourcePane()
	before := s.current()
	w.updateSourceLinks()
	text := w.textEditor.Text()
	for _, want := range []string{
		"Anna. [[from: #:~:text=She,sea.]]",
		"[[from: \"The rain kept falling.\"]]", // ambiguous: as it was
		"[[from: #:~:text=“Who's,answered.]]",  // near: re-anchored to its passage's words
		"[[from: \"Not in the story at all\"]]",
		"skipped links:\n  INT. HOUSE - NIGHT = #:~:text=It,late.",
		"cut:\n  #:~:text=late.-,The,falling.",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("no %q in\n%s", want, text)
		}
	}
	if !strings.Contains(w.statusLabel.Text, "Updated 4 source links") || !strings.Contains(w.statusLabel.Text, "2 left") {
		t.Errorf("status %q", w.statusLabel.Text)
	}
	after := s.current()
	if len(after.matches) != len(before.matches) {
		t.Fatalf("links %d, were %d", len(after.matches), len(before.matches))
	}
	for i := range after.matches {
		if a, b := after.matches[i], before.matches[i]; a.Kind != b.Kind && !(b.Kind == linkNear && a.Kind == linkFound) || !slices.Equal(a.Passages, b.Passages) {
			t.Errorf("link %d: %+v, was %+v", i, a, b)
		}
	}
	if !slices.Equal(after.status, before.status) || !slices.Equal(after.cutAt, before.cutAt) {
		t.Errorf("statuses %v, were %v; cuts %v, were %v", after.status, before.status, after.cutAt, before.cutAt)
	}
	// once more: nothing to do (the ambiguous and missing still left)
	w.updateSourceLinks()
	if w.textEditor.Text() != text {
		t.Error("updated twice")
	}
	w.textEditor.Undo()
	if w.textEditor.Text() != script {
		t.Errorf("one undo:\n%s", w.textEditor.Text())
	}
}

// A link written opens at its passage in the source exported as a web
// page: a stand-in for the browser's find (the spec's algorithm on the
// page's blocks' text, literally: case aside, whitespace collapsed).
func TestFragmentOpensInExportedSource(t *testing.T) {
	src := sourceStory + "\n\nShort.\n\nIt was “late”, and well-known.\n\nShort.\n\n- a list item, of its own\n- Short.\n\n> Short.\n\n" +
		"> Dear Anna, I hope this finds you well and the garden too.\n>\n> I am writing to say the house is sold, and the rain came in at last.\n\n" +
		"She *opened* the [door](http://example.com) to `the` garden, **and** walked on into the night air.\n\n" +
		"- a loose item with words enough to have an end\n\n  and a second paragraph of it, which ends here\n- nested: the outer words\n  - the inner item, with an end of its own\n"
	page, err := markdownHTML(src, "Story")
	if err != nil {
		t.Fatal(err)
	}
	blocks := htmlBlocks(string(page))
	ps := passagesOf(src)
	lm := newLinkMatcher(ps)
	checked, withContext, multi := 0, 0, 0
	for i, p := range ps {
		if !p.linkable() {
			continue
		}
		d, unique := fragmentFor(lm, i, i)
		if !unique {
			t.Errorf("passage %d %q: not unique", i, p.Plain)
			continue
		}
		// what the script reads back is what the address has
		_, dirs, _ := parseFragment(fragmentString("", []textDirective{d}, true))
		b, e := browserFind(blocks, dirs[0])
		if b < 0 {
			t.Errorf("passage %d %q: %+v not found on the page", i, p.Plain, d)
			continue
		}
		plain := collapse(strings.Join(slices.Concat(blockFields(p.shown())...), " "))
		if !strings.Contains(plain, collapse(blocks[b])) || !strings.Contains(plain, collapse(blocks[e])) ||
			(len(blockFields(p.shown())) == 1 && b != e) {
			t.Errorf("passage %d %q: %+v opens blocks %d..%d %q", i, p.Plain, d, b, e, blocks[b:e+1])
		}
		if len(blockFields(p.shown())) > 1 {
			multi++
		}
		if d.Prefix != "" || d.Suffix != "" {
			withContext++
		}
		checked++
	}
	if checked < 12 || withContext < 3 || multi < 3 {
		t.Errorf("%d passages checked, %d with a prefix or suffix, %d of several blocks", checked, withContext, multi)
	}
}

var anyTag = regexp.MustCompile(`<(/?)([a-zA-Z0-9]+)[^>]*>`)

// htmlBlocks are a page's blocks' text, in order, as the spec collects
// it: each block element (opening or closing) ends the text before it;
// the head's left out.
func htmlBlocks(page string) []string {
	if i := strings.Index(page, "<body>"); i >= 0 {
		page = page[i+6:]
	}
	block := map[string]bool{"p": true, "li": true, "ul": true, "ol": true, "blockquote": true, "pre": true, "div": true, "hr": true,
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true, "body": true, "table": true, "tr": true, "td": true, "th": true}
	var out []string
	var cur strings.Builder
	flush := func() {
		if t := strings.TrimSpace(html.UnescapeString(cur.String())); t != "" {
			out = append(out, t)
		}
		cur.Reset()
	}
	at := 0
	for _, m := range anyTag.FindAllStringSubmatchIndex(page, -1) {
		cur.WriteString(page[at:m[0]])
		at = m[1]
		if block[strings.ToLower(page[m[4]:m[5]])] {
			flush()
		}
	}
	cur.WriteString(page[at:])
	flush()
	return out
}

func collapse(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

// browserFind is the spec's "find a range from a text directive" on
// blocks of text, as a browser does it: the first match; the blocks the
// start and the end are in (-1: none).
func browserFind(blocks []string, d textDirective) (int, int) {
	text := make([][]rune, len(blocks))
	for i, b := range blocks {
		text[i] = []rune(collapse(b))
	}
	type pos struct{ b, i int }
	boundary := func(r []rune, i int) bool {
		word := func(c rune) bool { return unicode.IsLetter(c) || unicode.IsDigit(c) }
		return i <= 0 || i >= len(r) || !word(r[i-1]) || !word(r[i])
	}
	find := func(q string, from pos, startB, endB bool) (pos, pos, bool) {
		qr := []rune(collapse(q))
		for b := from.b; b < len(text); b++ {
			i0 := 0
			if b == from.b {
				i0 = from.i
			}
			for i := i0; i+len(qr) <= len(text[b]); i++ {
				if string(text[b][i:i+len(qr)]) == string(qr) && (!startB || boundary(text[b], i)) && (!endB || boundary(text[b], i+len(qr))) {
					return pos{b, i}, pos{b, i + len(qr)}, true
				}
			}
		}
		return pos{}, pos{}, false
	}
	// next non-whitespace: past spaces, into the next block
	skip := func(p pos) pos {
		for p.b < len(text) {
			for p.i < len(text[p.b]) && text[p.b][p.i] == ' ' {
				p.i++
			}
			if p.i < len(text[p.b]) {
				return p
			}
			p = pos{p.b + 1, 0}
		}
		return p
	}
	search := pos{0, 0}
	for search.b < len(text) {
		var ms, me pos
		if d.Prefix != "" {
			ps, pe, ok := find(d.Prefix, search, true, false)
			if !ok {
				return -1, -1
			}
			search = pos{ps.b, ps.i + 1}
			at := skip(pe)
			s, e, ok := find(d.Start, at, false, d.End != "" || d.Suffix == "")
			if !ok {
				return -1, -1
			}
			if s != at {
				continue
			}
			ms, me = s, e
		} else {
			s, e, ok := find(d.Start, search, true, d.End != "" || d.Suffix == "")
			if !ok {
				return -1, -1
			}
			search = pos{s.b, s.i + 1}
			ms, me = s, e
		}
		endFrom := me
		for {
			end := me
			if d.End != "" {
				_, ee, ok := find(d.End, endFrom, true, d.Suffix == "")
				if !ok {
					return -1, -1
				}
				end = ee
			}
			if d.Suffix == "" {
				return ms.b, end.b
			}
			at := skip(end)
			s, _, ok := find(d.Suffix, at, false, true)
			if !ok {
				return -1, -1
			}
			if s == at {
				return ms.b, end.b
			}
			if d.End == "" {
				break
			}
			endFrom = end
		}
	}
	return -1, -1
}

// A link's file: the source's name in any of its layers (its extension
// aside) is it; another name is read against it, and said; moving the
// source into the script keeps the named links found.
func TestFragmentFileNames(t *testing.T) {
	script := "INT. HOUSE - NIGHT\n\n[[from: story.md#:~:text=She opened the door]]\n\nINT. YARD - DAY\n\n[[from: other.md#:~:text=She opened the door]]\n"
	w, _ := sourceWindow(t, script, sourceStory)
	s := w.sourcePane()
	follow := func(line int) string {
		w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(line))
		s.follow()
		return s.info.Text
	}
	if info := follow(2); strings.Contains(info, "names") || !strings.Contains(info, "1 passage") {
		t.Errorf("story.md: %q", info)
	}
	if info := follow(6); !strings.Contains(info, "names other.md, not this script's source") {
		t.Errorf("other.md: %q", info)
	}
	if err := w.keepSourceInside(true); err != nil {
		t.Fatal(err)
	}
	m := s.current()
	if len(m.matches) != 2 || m.matches[0].Kind != linkFound {
		t.Errorf("kept inside: %+v", m.matches)
	}
	if info := follow(2); strings.Contains(info, "names") {
		t.Errorf("story.md, kept inside: %q", info)
	}
	if !w.sourceNames(w.sourceRef())[sourceNameKey("Story.HTML")] {
		t.Error("story.html is not the source's name")
	}
}

// New links are the shortest that find their passage alone: on the
// sample's story and on the bench novel, each written link finds its
// passage and only it, and one word less of its start or its end does
// not.
func TestShortLinksUnique(t *testing.T) {
	zr, err := zip.NewReader(bytes.NewReader(sampleAccolade), int64(len(sampleAccolade)))
	if err != nil {
		t.Fatal(err)
	}
	var story string
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, ".textbundle/assets/source.md") {
			r, _ := f.Open()
			b, _ := io.ReadAll(r)
			r.Close()
			story = string(b)
		}
	}
	novel, _ := benchFixture()
	for _, c := range []struct{ name, md string }{{"sample", story}, {"novel", novel}} {
		ps := passagesOf(c.md)
		lm := newLinkMatcher(ps)
		linkable, words, alike := 0, 0, 0
		finds := func(d textDirective, i int) bool {
			mt := lm.matchRef(linkRef{Dir: d})
			return mt.Kind == linkFound && mt.Passages[0] == i && mt.last() == i
		}
		less := func(term string, fromEnd bool) (string, bool) { // a word less: its first or last field with words
			fs := strings.Fields(term)
			if len(normWords(term)) < 2 {
				return "", false
			}
			n := wordFields(fs, len(fs), fromEnd) // all
			for k := 1; k < n; k++ {
				var cut []string
				if fromEnd {
					cut = fs[k:]
				} else {
					cut = fs[:len(fs)-k]
				}
				if w := normWords(strings.Join(cut, " ")); len(w) < len(normWords(term)) && len(w) > 0 {
					return strings.Join(cut, " "), true
				}
			}
			return "", false
		}
		for i, p := range ps {
			if !p.linkable() {
				continue
			}
			linkable++
			d, unique := fragmentFor(lm, i, i)
			if !unique {
				alike++ // a passage that reads as another one does, word for word
				continue
			}
			if !finds(d, i) {
				t.Errorf("%s: passage %d's link %+v finds %+v", c.name, i, d, lm.matchRef(linkRef{Dir: d}))
			}
			if d.End != "" && normWords(d.End) == nil {
				t.Errorf("%s: passage %d's link ends in no words: %+v", c.name, i, d)
			}
			words += len(normWords(d.Start)) + len(normWords(d.End))
			if d.End == "" || d.Prefix != "" || d.Suffix != "" {
				continue // quoted whole, or with context
			}
			if s, ok := less(d.Start, false); ok && finds(textDirective{Start: s, End: d.End}, i) {
				t.Errorf("%s: passage %d's link %+v finds it with the start %q", c.name, i, d, s)
			}
			if e, ok := less(d.End, true); ok && finds(textDirective{Start: d.Start, End: e}, i) {
				t.Errorf("%s: passage %d's link %+v finds it with the end %q", c.name, i, d, e)
			}
		}
		t.Logf("%s: %d passages linkable, %.2f words a link, %d alike (not unique)", c.name, linkable, float64(words)/float64(max(1, linkable-alike)), alike)
		if c.name == "sample" && linkable < 25 {
			t.Errorf("the sample's story has %d passages", linkable)
		}
	}
}
