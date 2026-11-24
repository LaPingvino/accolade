package main

import (
	"slices"
	"strings"
	"sync"

	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/linter"
)

// The checks as open items (LOOP.md, REVISION TOOLS; Joop 2026-11-24:
// "It's always good to always have tools available that help you figure
// out where your screenplay might be lacking"): the heuristics the
// Production pane runs before a lock (production_checks.go) that are
// about the writing, not the lock (a heading without INT./EXT. or a
// time, a boneyard or a note not closed, one character under two cues,
// a very long scene), and Lexington's linter (a parenthetical with no
// speech, a cue without a name, a dual dialogue gone wrong). Their
// findings are computed from the text after each pause, never stored:
// a finding is gone when the text no longer gives it. Keep writes one
// into the text as a revision note, [[revision: …]] at the end of its
// line, which is then listed (and the finding it was, which the note now
// stands for, is not); Dismiss puts one out of sight until the window
// closes, writing nothing.

// checkLint is a finding of Lexington's linter.
const checkLint = "lint"

// writingCheck tells whether a pre-lock check's kind is about the
// writing (the numbers and the title page are the lock's).
func writingCheck(c prodCheck) bool {
	return !c.info && c.kind != checkNumber && c.kind != checkTitle
}

// lintMessages are the linter's messages in Accolade's words.
var lintMessages = map[string]string{
	"Nested dual dialogue block detected.":                  "A dual dialogue inside another: Fountain has one at a time",
	"More than two speakers in a dual dialogue block.":      "More than two speakers side by side: a dual dialogue has two",
	"Parenthetical without a preceding speaker or dialogue": "A parenthetical with no cue or speech before it: it reads as action",
	"Empty speaker name detected.":                          "A cue without a name",
}

// lintMessage is a linter message in Accolade's words, else as it is.
func lintMessage(m string) string {
	for k, v := range lintMessages {
		if strings.HasPrefix(m, k) {
			return v
		}
	}
	return m
}

// withoutLineNotes is text with the notes that open and close on one
// line taken out (the lines stay where they are): a kept revision note
// on a heading does not change the heading the checks name.
func withoutLineNotes(text string) string {
	if !strings.Contains(text, "[[") {
		return text
	}
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		for {
			o := strings.Index(l, "[[")
			if o < 0 {
				break
			}
			c := strings.Index(l[o+2:], "]]")
			if c < 0 {
				break
			}
			l = strings.TrimRight(l[:o], " \t") + l[o+2+c+2:]
		}
		lines[i] = l
	}
	return strings.Join(lines, "\n")
}

// lintFindings are the linter's findings in text (a screenplay's body):
// each its line (from 0) and message. The linter counts the rows it
// reads, not the text's lines (a title page's rows not counted, and its
// own line 1 left out): its count is replayed here to find the row, and
// the row's Source (fountain.ParseLines) is the line.
func lintFindings(text string, starts []string) []prodCheck {
	rows := fountain.ParseLines(starts, strings.NewReader(text))
	l := linter.NewLinter()
	l.Lint(rows)
	if !l.HasErrors() {
		return nil
	}
	row := map[int]int{} // the linter's line number: the row's index
	n := 1
	for i, r := range rows {
		if r.Type != lex.TypeTitlePage && r.Type != lex.TypeMetaSection {
			n++
		}
		if _, ok := row[n]; !ok {
			row[n] = i
		}
	}
	var out []prodCheck
	for _, e := range l.Errors {
		i, ok := row[e.LineNum]
		if !ok || rows[i].Source <= 0 {
			continue
		}
		out = append(out, prodCheck{line: rows[i].Source - 1, kind: checkLint, text: lintMessage(e.Message)})
	}
	return out
}

// checkMemo is the last text's findings.
var checkMemo struct {
	sync.Mutex
	text   string
	starts []string
	items  []openItem
	ok     bool
}

// checkItemsOf are a screenplay's findings as items, in the order of
// their lines: each on its line (Start, End: the line's, in runes; Raw
// the line), under the heading it is in. A finding whose line has a
// revision note saying the same is left out: the note stands for it.
func checkItemsOf(text string, starts []string) []openItem {
	checkMemo.Lock()
	defer checkMemo.Unlock()
	if checkMemo.ok && slices.Equal(checkMemo.starts, starts) && sameString(checkMemo.text, text) {
		return checkMemo.items
	}
	items := findingItems(text, starts)
	checkMemo.text, checkMemo.starts, checkMemo.items, checkMemo.ok = text, slices.Clone(starts), items, true
	return items
}

func findingItems(text string, starts []string) []openItem {
	body := text[:fountainBodyEnd(text)]
	if strings.TrimSpace(body) == "" {
		return nil
	}
	clean := withoutLineNotes(body)
	var checks []prodCheck
	for _, c := range preLockChecks(clean, starts) {
		if writingCheck(c) {
			checks = append(checks, c)
		}
	}
	checks = append(checks, lintFindings(clean, starts)...)
	slices.SortStableFunc(checks, func(a, b prodCheck) int { return a.line - b.line })
	lines := strings.Split(text, "\n")
	kept := keptRevisions(body)
	var items []openItem
	seen := map[string]bool{}
	for _, c := range checks {
		if c.line < 0 || c.line >= len(lines) {
			continue
		}
		key := string(rune(c.line)) + c.text
		if seen[key] || kept[c.line][c.text] { // twice (a heading's two checks alike), or kept as a note
			continue
		}
		seen[key] = true
		items = append(items, openItem{Kind: openCheck, Shape: shapeFinding, Line: c.line, Text: c.text, Check: c.kind, Raw: lines[c.line]})
	}
	if len(items) == 0 {
		return nil
	}
	setContext(items, outlineOf(text, starts), lines)
	// the lines' places in runes, in order
	c := runeCounter{text: text}
	at, line := 0, 0
	for i := range items {
		it := &items[i]
		for line < it.Line {
			at += len(lines[line]) + 1
			line++
		}
		it.Start = c.runes(at)
		it.End = it.Start + len([]rune(it.Raw))
	}
	return items
}

// keptRevisions are the revision notes on each line of body: by line,
// what they say.
func keptRevisions(body string) map[int]map[string]bool {
	out := map[int]map[string]bool{}
	if !strings.Contains(body, "[[") {
		return out
	}
	for i, l := range strings.Split(body, "\n") {
		for {
			o := strings.Index(l, "[[")
			if o < 0 {
				break
			}
			c := strings.Index(l[o+2:], "]]")
			if c < 0 {
				break
			}
			if _, rest, ok := revisionNote(l[o+2 : o+2+c]); ok {
				if out[i] == nil {
					out[i] = map[string]bool{}
				}
				out[i][rest] = true
			}
			l = l[o+2+c+2:]
		}
	}
	return out
}

// keepEdit is the edit that keeps a finding as a revision note: at the
// end of its line, [[revision: …]].
func (it openItem) keepEdit() textEdit {
	note := " [[revision: " + strings.ReplaceAll(it.Text, "]]", "] ]") + "]]"
	if strings.TrimSpace(it.Raw) == "" {
		note = note[1:]
	}
	end := it.Start + len([]rune(strings.TrimRight(it.Raw, " \t\r")))
	return textEdit{end, it.End, note}
}

// dismissKey is what a dismissed finding is known by while the window
// is open: its check, what it says and its line's text (not its place,
// which moves with every edit before it).
func (it openItem) dismissKey() string {
	return it.Check + "\x00" + it.Text + "\x00" + strings.TrimSpace(it.Raw)
}

// openItems are the script's open items with its checks' findings among
// them (in their places, the unsorted notes still last), the findings
// dismissed in this window left out.
func (w *MainWindow) openItems() []openItem {
	if w.textEditor == nil {
		return nil
	}
	text := w.textEditor.Text()
	items := openItemsOf(text, w.sceneStarts(), w.markdown)
	if w.markdown {
		return items
	}
	var found []openItem
	for _, f := range checkItemsOf(text, w.sceneStarts()) {
		if !w.dismissed[f.dismissKey()] {
			found = append(found, f)
		}
	}
	return mergeFindings(items, found)
}

// mergeFindings puts findings among items by their place (a note on a
// line before its line's finding), the unsorted notes last.
func mergeFindings(items, found []openItem) []openItem {
	if len(found) == 0 {
		return items
	}
	out := make([]openItem, 0, len(items)+len(found))
	j := 0
	for _, it := range items {
		if it.Kind != openUnsorted {
			for j < len(found) && found[j].Start < it.Start {
				out = append(out, found[j])
				j++
			}
		} else {
			out = append(out, found[j:]...)
			j = len(found)
		}
		out = append(out, it)
	}
	return append(out, found[j:]...)
}
