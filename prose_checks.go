package main

import (
	"bytes"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
)

// Prose checks (LOOP.md, REVISION TOOLS; Joop 2026-11-24: "I guess such
// an approach also works for the Markdown/source part?"): the findings of
// a Markdown document and of the source, as the screenplay's checks are a
// script's (revision_checks.go), the same items, the same Revision tab.
// Kept modest, and at most one of each check per paragraph: a word
// twice close together, a very long sentence, and in English text a
// paragraph leaning on filter words (felt, saw, noticed…) or on the
// passive. Each finding's place is what it is about (the sentence, the
// two words), its message ends with what to try. Keep writes the finding
// as a comment on its own line under the paragraph, <!-- revision: … -->
// (within the paragraph it would split it), which then stands for it.

// The prose checks' kinds.
const (
	checkRepeat   = "repeat"
	checkSentence = "sentence"
	checkFilter   = "filter"
	checkPassive  = "passive"
)

// proseLong is the words a sentence has before it is a long one.
const proseLong = 45

// proseNear is how many words apart a word comes again close together.
const proseNear = 6

var (
	proseWordRe = regexp.MustCompile(`[\p{L}\p{M}][\p{L}\p{M}\p{N}'’]*`)
	// masked before reading: a link's address, a footnote's mark, a comment
	proseMaskRe = regexp.MustCompile(`\]\([^)]*\)|\[\^[^\]]*\]|<!--.*?-->|<[^>]+>`)
)

// englishCommon are the words that tell English text; englishStop the
// short words a repetition of which is not one.
var (
	englishCommon = setOf("the", "and", "of", "to", "a", "in", "is", "was", "it", "that", "he", "she", "with", "for", "his", "her", "you", "i", "on", "at")
	englishStop   = setOf("that", "this", "with", "from", "have", "were", "would", "could", "should", "there", "their", "they", "them", "then", "than",
		"what", "when", "which", "while", "will", "your", "about", "into", "only", "some", "very", "just", "been", "said", "says", "like", "also",
		"more", "much", "here", "where", "after", "before", "over", "under", "again", "even", "upon", "such", "each", "other", "these", "those")
	filterWords = setOf("felt", "feel", "feels", "saw", "sees", "heard", "hears", "noticed", "notices", "realized", "realised", "realizes",
		"wondered", "wonders", "seemed", "seems", "watched", "knew", "decided", "thought")
	beWords     = setOf("am", "is", "are", "was", "were", "be", "been", "being")
	participles = setOf("taken", "given", "seen", "done", "made", "known", "written", "told", "found", "held", "left", "brought", "thrown",
		"shown", "broken", "chosen", "driven", "eaten", "forgotten", "hidden", "spoken", "stolen", "torn", "worn", "kept", "sent", "put", "set",
		"caught", "taught", "bought", "built", "paid", "sold", "struck", "hit", "cut", "shut", "beaten", "bitten", "drawn", "grown", "thrust")
)

func setOf(words ...string) map[string]bool {
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[w] = true
	}
	return m
}

// proseWord is a word of a paragraph: where it is (bytes, in the text)
// and its lower case.
type proseWord struct {
	start, end int
	low        string
}

// prosePara is a paragraph: its lines and bytes in the text.
type prosePara struct {
	first, last int // lines
	start, end  int // bytes
}

// proseParas are a Markdown text's paragraphs of prose (not headings,
// code, comments, lists, front matter), up to end (bytes).
func proseParas(text string, end int) []prosePara {
	lines := strings.Split(text, "\n")
	kinds := syntax.ClassifyMarkdown(lines)
	var out []prosePara
	at := 0
	open := false
	for i, l := range lines {
		if at >= end {
			break
		}
		prose := kinds[i] == syntax.Action || kinds[i] == syntax.MdQuote
		switch {
		case prose && open:
			p := &out[len(out)-1]
			p.last, p.end = i, at+len(l)
		case prose:
			out = append(out, prosePara{first: i, last: i, start: at, end: at + len(l)})
			open = true
		default:
			open = false
		}
		at += len(l) + 1
	}
	return out
}

// proseWords are a paragraph's words, code spans and marks left out.
func proseWords(text string, p prosePara) []proseWord {
	b := []byte(text[p.start:p.end])
	for _, m := range proseMaskRe.FindAllIndex(b, -1) {
		for k := m[0]; k < m[1]; k++ {
			b[k] = ' '
		}
	}
	for ls := 0; ls <= len(b); {
		le := bytes.IndexByte(b[ls:], '\n')
		if le < 0 {
			le = len(b)
		} else {
			le += ls
		}
		for _, c := range codeSpans(string(b[ls:le])) {
			for k := ls + c[0]; k < ls+c[1]; k++ {
				b[k] = ' '
			}
		}
		ls = le + 1
	}
	var out []proseWord
	for _, m := range proseWordRe.FindAllIndex(b, -1) {
		out = append(out, proseWord{p.start + m[0], p.start + m[1], strings.ToLower(string(b[m[0]:m[1]]))})
	}
	return out
}

// englishText tells whether words read as English: enough of its most
// common words among them.
func englishText(words []proseWord) bool {
	n := 0
	for _, w := range words {
		if englishCommon[w.low] {
			n++
		}
	}
	return len(words) >= 20 && n*100 >= len(words)*15
}

// proseFinding is a finding in bytes: from, to, the check, what it says,
// the paragraph it is in.
type proseFinding struct {
	start, end int
	check, msg string
	para       prosePara
}

// proseFindings are a Markdown text's findings, in their order.
func proseFindings(text string) []proseFinding {
	end := markdownEndByte(text)
	paras := proseParas(text, end)
	words := make([][]proseWord, len(paras))
	var all []proseWord
	for i, p := range paras {
		words[i] = proseWords(text, p)
		all = append(all, words[i]...)
	}
	english := englishText(all)
	var out []proseFinding
	for i, p := range paras {
		ws := words[i]
		var found []proseFinding
		add := func(s, e int, check, msg string) {
			found = append(found, proseFinding{s, e, check, msg, p})
		}
		// a word twice close together (the first such pair)
		for k := range ws {
			w := ws[k]
			if utf8.RuneCountInString(w.low) < 4 || (english && englishStop[w.low]) {
				continue
			}
			for j := k + 1; j < len(ws) && j-k <= proseNear; j++ {
				if ws[j].low == w.low && !sentenceBetween(text[w.end:ws[j].start]) {
					add(w.start, ws[j].end, checkRepeat, fmt.Sprintf("“%s” twice in %d words: another word, or one less?", w.low, j-k+1))
					break
				}
			}
			if len(found) > 0 {
				break
			}
		}
		// the longest sentence, if long
		longS, longE, longN := 0, 0, 0
		forSentences(text, ws, func(s, e, n int) {
			if n > longN {
				longS, longE, longN = s, e, n
			}
		})
		if longN > proseLong {
			add(longS, longE, checkSentence, fmt.Sprintf("A sentence of %d words: where could it break in two?", longN))
		}
		if english {
			var fl, pa []int // the words' indexes
			for k, w := range ws {
				if filterWords[w.low] {
					fl = append(fl, k)
				}
				if beWords[w.low] && k+1 < len(ws) {
					n := k + 1
					if strings.HasSuffix(ws[n].low, "ly") && n+1 < len(ws) {
						n++
					}
					if participle(ws[n].low) && !sentenceBetween(text[w.end:ws[n].start]) {
						pa = append(pa, k, n)
					}
				}
			}
			if len(fl) >= 3 {
				add(ws[fl[0]].start, ws[fl[len(fl)-1]].end, checkFilter,
					fmt.Sprintf("%d filter words (%s): what if the reader saw it straight, not through someone?", len(fl), wordList(text, ws, fl, 1)))
			}
			if len(pa) >= 6 {
				add(ws[pa[0]].start, ws[pa[len(pa)-1]].end, checkPassive,
					fmt.Sprintf("%d passives (%s): who is doing it?", len(pa)/2, wordList(text, ws, pa, 2)))
			}
		}
		slices.SortStableFunc(found, func(a, b proseFinding) int { return a.start - b.start })
		out = append(out, found...)
	}
	return out
}

// participle tells whether a word reads as a past participle.
func participle(w string) bool {
	return participles[w] || (strings.HasSuffix(w, "ed") && utf8.RuneCountInString(w) > 4)
}

// sentenceBetween tells whether a sentence ends in s (the text between two words).
func sentenceBetween(s string) bool {
	return strings.ContainsAny(s, ".!?;") || strings.Contains(s, "\n\n")
}

// forSentences calls f with each sentence of a paragraph's words: its
// bytes (first word's start to last's end) and its words.
func forSentences(text string, ws []proseWord, f func(s, e, n int)) {
	from := 0
	for k := range ws {
		if k+1 == len(ws) || strings.ContainsAny(text[ws[k].end:ws[k+1].start], ".!?") {
			f(ws[from].start, ws[k].end, k-from+1)
			from = k + 1
		}
	}
}

// wordList names the first words at idx (a group of per words each), up
// to three groups.
func wordList(text string, ws []proseWord, idx []int, per int) string {
	var parts []string
	for g := 0; g+per <= len(idx) && len(parts) < 3; g += per {
		parts = append(parts, text[ws[idx[g]].start:ws[idx[g+per-1]].end])
	}
	s := strings.Join(parts, ", ")
	if len(idx)/per > 3 {
		s += "…"
	}
	return s
}

// proseMemo is the last text's findings.
var proseMemo struct {
	text  string
	items []openItem
	ok    bool
}

// proseItemsOf are a Markdown text's findings as items: each its span
// (Start, End in runes; Raw the span), KeepAt the end of its paragraph;
// one whose paragraph has a revision note saying the same left out.
func proseItemsOf(text string) []openItem {
	checkMemo.Lock()
	defer checkMemo.Unlock()
	if proseMemo.ok && sameString(proseMemo.text, text) {
		return proseMemo.items
	}
	items := proseItems(text)
	proseMemo.text, proseMemo.items, proseMemo.ok = text, items, true
	return items
}

func proseItems(text string) []openItem {
	found := proseFindings(text)
	if len(found) == 0 {
		return nil
	}
	lines := strings.Split(text, "\n")
	kinds := syntax.ClassifyMarkdown(lines)
	lineAt := lineIndex(text)
	var items []openItem
	for _, f := range found {
		if paraKept(lines, kinds, f.para.last)[f.msg] {
			continue
		}
		items = append(items, openItem{Kind: openCheck, Shape: shapeFinding, Start: f.start, End: f.end, Line: lineAt(f.start),
			Text: f.msg, Check: f.check, Raw: text[f.start:f.end], KeepAt: f.para.end})
	}
	setContext(items, markdownOutline(text), lines)
	c, k := runeCounter{text: text}, runeCounter{text: text} // Start and KeepAt each in order
	for i := range items {
		it := &items[i]
		it.Start = c.runes(it.Start)
		it.End = it.Start + utf8.RuneCountInString(it.Raw)
		it.KeepAt = k.runes(it.KeepAt)
	}
	return items
}

// paraKept are the revision notes under a paragraph (whose last line is
// last): the one-line comments right after it, what they say.
func paraKept(lines []string, kinds []syntax.Kind, last int) map[string]bool {
	out := map[string]bool{}
	for i := last + 1; i < len(lines) && kinds[i] == syntax.MdComment; i++ {
		t := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(t, "<!--") || !strings.HasSuffix(t, "-->") {
			break
		}
		if _, rest, ok := revisionNote(strings.TrimSuffix(strings.TrimPrefix(t, "<!--"), "-->")); ok {
			out[strings.ReplaceAll(rest, "-- >", "-->")] = true
		}
	}
	return out
}
