package main

import (
	"bytes"
	"fmt"
	"maps"
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
// twice close together (a content word: a language's function words, je,
// die, the, are no repetition: Joop 2026-11-25, "as a dutch speaker with
// a fair bit of 'je je' in texts, that has kinda annoyed me growing up
// with Word correction"), a word doubled (the the), but not a language's
// valid doubles (je je, die die, had had), a very long sentence, and in English text a
// paragraph leaning on filter words (felt, saw, noticed…) or on the
// passive. Each finding's place is what it is about (the sentence, the
// two words), its message ends with what to try. Keep writes the finding
// as a comment on its own line under the paragraph, <!-- revision: … -->
// (within the paragraph it would split it), which then stands for it.

// The prose checks' kinds.
const (
	checkRepeat   = "repeat"
	checkDouble   = "double"
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

// englishStop are English's longer function words (proseLangs has the
// short ones), a repetition of which is not one.
var (
	englishStop = setOf("that", "this", "with", "from", "have", "were", "would", "could", "should", "there", "their", "they", "them", "then", "than",
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

// proseLang is what the prose checks know of a language: the common
// words that tell it, its function words (never a repetition), and the
// words that may stand twice in a row (Dutch "dat dat", English "had
// had": no typo).
type proseLang struct {
	common, stop, doubles map[string]bool
}

// proseLangs are the languages the checks tell, by code.
var proseLangs = map[string]proseLang{
	"en": {
		common:  setOf("the", "and", "of", "to", "a", "in", "is", "was", "it", "that", "he", "she", "with", "for", "his", "her", "you", "i", "on", "at"),
		stop:    englishStop,
		doubles: setOf("had", "that", "is"),
	},
	"nl": {
		common: setOf("de", "het", "een", "en", "van", "ik", "je", "dat", "die", "niet", "is", "op", "te", "er", "zijn", "met", "voor", "maar", "hij", "ze", "was", "naar", "ook", "wat"),
		stop: setOf("aan", "alle", "alles", "als", "altijd", "andere", "want", "toch", "toen", "tegen", "door", "over", "omdat", "zodat", "terwijl", "nadat", "voordat",
			"maar", "niet", "niets", "niemand", "iets", "iemand", "veel", "weer", "wel", "zich", "zelf", "zelfs", "zijn", "haar", "hun", "hen", "hem", "mijn", "jouw",
			"onze", "jullie", "deze", "dit", "dat", "die", "daar", "hier", "waar", "wat", "wie", "welke", "hoe", "naar", "voor", "achter", "onder", "boven", "tussen",
			"zonder", "binnen", "buiten", "bij", "uit", "met", "tot", "sinds", "heeft", "hebben", "had", "hadden", "was", "waren", "wordt", "worden", "werd", "werden",
			"zal", "zullen", "zou", "zouden", "kan", "kunnen", "kon", "konden", "moet", "moeten", "moest", "wil", "willen", "wilde", "mag", "mogen", "mocht",
			"geweest", "geen", "nog", "ook", "dan", "toe", "even", "heel", "erg", "meer", "meest", "zeer", "zo", "nu", "al", "pas", "eens", "ooit", "nooit"),
		doubles: setOf("je", "die", "dat", "er", "ze", "haar", "wat", "zij"),
	},
	"de": {
		common: setOf("der", "die", "und", "das", "ist", "nicht", "ich", "sie", "es", "ein", "eine", "zu", "den", "mit", "auf", "sich", "dem", "auch", "war", "er"),
		stop: setOf("aber", "alle", "alles", "als", "also", "auch", "auf", "aus", "bei", "bin", "bis", "dann", "darum", "dass", "dein", "deine", "denn", "dessen",
			"diese", "dieser", "dieses", "doch", "dort", "durch", "eine", "einem", "einen", "einer", "eines", "etwas", "gegen", "habe", "haben", "hatte", "hatten",
			"hier", "ihre", "ihren", "ihrem", "ihrer", "immer", "jetzt", "kann", "kein", "keine", "konnte", "können", "mein", "meine", "mich", "mir", "muss",
			"musste", "nach", "nicht", "nichts", "noch", "nun", "oder", "ohne", "schon", "sehr", "sein", "seine", "seinem", "seinen", "seiner", "sich", "sind",
			"soll", "sollte", "über", "unter", "viel", "vom", "von", "vor", "war", "waren", "warum", "was", "weil", "wenn", "werden", "wird", "wieder", "will",
			"wir", "wird", "wollte", "wurde", "wurden", "zum", "zur", "zwischen", "während", "einmal", "selbst", "euch", "uns", "unser", "unsere", "welche"),
		doubles: setOf("die", "das", "sie", "der"),
	},
	"fr": {
		common: setOf("le", "la", "les", "de", "et", "un", "une", "est", "que", "il", "elle", "je", "ne", "pas", "en", "du", "des", "dans", "qui", "pour"),
		stop: setOf("avec", "dans", "pour", "mais", "elle", "elles", "nous", "vous", "leur", "leurs", "cette", "cela", "comme", "tout", "tous", "toute", "toutes",
			"sans", "sous", "était", "étaient", "avait", "avaient", "sont", "être", "avoir", "fait", "plus", "très", "aussi", "encore", "quand", "alors", "donc",
			"puis", "même", "dont", "entre", "vers", "chez", "depuis", "pendant", "avant", "après", "rien", "jamais", "toujours", "quelque", "notre", "votre"),
		doubles: setOf("nous", "vous"),
	},
	"es": {
		common: setOf("el", "la", "los", "las", "de", "y", "que", "en", "un", "una", "es", "no", "se", "por", "con", "su", "para", "lo", "del", "al"),
		stop: setOf("como", "pero", "para", "porque", "cuando", "donde", "este", "esta", "esto", "estos", "estas", "ese", "esa", "eso", "todo", "toda", "todos",
			"todas", "muy", "más", "también", "sobre", "entre", "hasta", "desde", "sin", "había", "habían", "estaba", "estaban", "era", "eran", "fue", "ser",
			"estar", "tiene", "tenía", "ella", "ellos", "ellas", "nosotros", "nada", "nunca", "siempre", "ahora", "entonces", "después", "antes", "sus"),
		doubles: setOf(),
	},
	"eo": {
		common: setOf("la", "kaj", "estas", "estis", "de", "en", "al", "mi", "li", "ŝi", "ne", "ke", "kun", "por", "el", "sed", "tio", "ĉu", "ili", "ni"),
		stop: setOf("estas", "estis", "estos", "kiam", "ankaŭ", "ankoraŭ", "ĉar", "tiam", "kiu", "kiuj", "kio", "kiel", "kie", "tio", "tiu", "tiuj", "ĉiuj",
			"nur", "tre", "post", "antaŭ", "inter", "sur", "sub", "pri", "ŝin", "lin", "ilin", "ŝia", "ŝiaj", "lia", "liaj", "ilia", "iliaj", "nia", "via",
			"mia", "neniu", "ĉiu", "oni", "povas", "povis", "devas", "devis", "havas", "havis", "jam", "ankoraŭ", "eĉ", "ĉiam", "neniam", "nun", "poste"),
		doubles: setOf(),
	},
}

// proseLangOf is the language words read as (its code, "" for none told):
// the one whose common words are most among them, enough of them.
func proseLangOf(words []proseWord) string {
	if len(words) < 8 {
		return ""
	}
	best, bestN := "", 0
	for _, code := range slices.Sorted(maps.Keys(proseLangs)) {
		n := 0
		for _, w := range words {
			if proseLangs[code].common[w.low] {
				n++
			}
		}
		if n > bestN {
			best, bestN = code, n
		}
	}
	if bestN*100 < len(words)*15 {
		return ""
	}
	return best
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
	lang := proseLangOf(all)
	english := lang == "en" && len(all) >= 20
	known := proseLangs[lang] // none: no lists
	// the commonest words (more than 1 in 100, in a text long enough to
	// tell) are its function words too: not repetitions (in a language
	// without a list, all it knows of them)
	common := map[string]bool{}
	if !english && len(all) >= 300 {
		n := map[string]int{}
		for _, w := range all {
			n[w.low]++
		}
		for w, c := range n {
			if c >= 3 && c*100 > len(all) {
				common[w] = true
			}
		}
	}
	var out []proseFinding
	for i, p := range paras {
		ws := words[i]
		var found []proseFinding
		add := func(s, e int, check, msg string) {
			found = append(found, proseFinding{s, e, check, msg, p})
		}
		function := func(w string) bool { return known.stop[w] || known.common[w] || common[w] }
		// a word doubled, a typo (the first): not a valid double of the
		// language; in one without a list, a content word only
		for k := 0; k+1 < len(ws); k++ {
			w, n := ws[k], ws[k+1]
			if n.low != w.low || strings.TrimSpace(text[w.end:n.start]) != "" || known.doubles[w.low] ||
				(known.common == nil && (utf8.RuneCountInString(w.low) < 4 || function(w.low))) {
				continue
			}
			add(w.start, n.end, checkDouble, fmt.Sprintf("“%s %s”: the word twice in a row, one too many?", text[w.start:w.end], text[n.start:n.end]))
			break
		}
		// a content word twice close together (the first such pair; one
		// right after the other is the doubled word's)
		for k := range ws {
			w := ws[k]
			if utf8.RuneCountInString(w.low) < 4 || function(w.low) {
				continue
			}
			for j := k + 1; j < len(ws) && j-k <= proseNear; j++ {
				if j == k+1 && strings.TrimSpace(text[w.end:ws[j].start]) == "" {
					continue
				}
				if ws[j].low == w.low && !sentenceBetween(text[w.end:ws[j].start]) {
					add(w.start, ws[j].end, checkRepeat, fmt.Sprintf("“%s” twice in %d words: another word, or one less?", w.low, j-k+1))
					break
				}
			}
			if slices.ContainsFunc(found, func(f proseFinding) bool { return f.check == checkRepeat }) {
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

// sentenceEnds end a sentence: Latin, Devanagari's danda, CJK's, Arabic's
// and Urdu's.
const sentenceEnds = ".!?।॥。！？؟۔"

// sentenceBetween tells whether a sentence ends in s (the text between two words).
func sentenceBetween(s string) bool {
	return strings.ContainsAny(s, sentenceEnds+";") || strings.Contains(s, "\n\n")
}

// forSentences calls f with each sentence of a paragraph's words: its
// bytes (first word's start to last's end) and its words.
func forSentences(text string, ws []proseWord, f func(s, e, n int)) {
	from := 0
	for k := range ws {
		if k+1 == len(ws) || strings.ContainsAny(text[ws[k].end:ws[k+1].start], sentenceEnds) {
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
