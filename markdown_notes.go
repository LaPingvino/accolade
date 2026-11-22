package main

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/accolade/internal/prose"
)

// A Markdown document's footnotes in the editor: Go to Note or Reference
// goes from a mark [^1] to its note ([^1]: …) and from the note back to
// its first mark; Renumber Notes numbers them 1, 2, 3… in the order the
// text refers to them (named ones too), their notes put in that order
// when they are together at the end; the pointer over a mark shows its
// note's text. Marks in code are not marks.

// mdNoteMark is a footnote's mark in the text: its label, where it is
// (in runes), and whether it starts its note ([^1]: …, at a line's start).
type mdNoteMark struct {
	label      string
	start, end int
	def        bool
}

var mdNoteMarkRe = regexp.MustCompile(`\[\^([^\]\s]+)\](:)?`)

// mdNoteMarks are the footnote marks of a Markdown text, in order: not in
// code blocks, code spans, front matter or comments.
func mdNoteMarks(text string) []mdNoteMark {
	lines := strings.Split(text, "\n")
	kinds := syntax.ClassifyMarkdown(lines)
	var out []mdNoteMark
	at := 0 // the line's start, in runes
	for i, l := range lines {
		switch kinds[i] {
		case syntax.MdCode, syntax.MdCodeFence, syntax.MdFrontMatter, syntax.MdComment:
		default:
			code := codeSpans(l)
			for _, m := range mdNoteMarkRe.FindAllStringSubmatchIndex(l, -1) {
				if slices.ContainsFunc(code, func(c [2]int) bool { return m[0] >= c[0] && m[0] < c[1] }) {
					continue
				}
				def := m[4] >= 0 && strings.TrimLeft(l[:m[0]], " ") == "" && m[0] <= 3
				end := m[1]
				if m[4] >= 0 && !def { // a mark followed by a colon in the text
					end = m[3] + 1
				}
				out = append(out, mdNoteMark{label: l[m[2]:m[3]], def: def,
					start: at + utf8.RuneCountInString(l[:m[0]]), end: at + utf8.RuneCountInString(l[:end])})
			}
		}
		at += utf8.RuneCountInString(l) + 1
	}
	return out
}

// codeSpans are a line's code spans (byte ranges): from a run of
// backticks to the next run as long.
func codeSpans(l string) [][2]int {
	var out [][2]int
	for i := 0; i < len(l); {
		if l[i] != '`' {
			i++
			continue
		}
		j := i
		for j < len(l) && l[j] == '`' {
			j++
		}
		run := l[i:j]
		k := strings.Index(l[j:], run)
		if k < 0 {
			break
		}
		out = append(out, [2]int{i, j + k + len(run)})
		i = j + k + len(run)
	}
	return out
}

// noteMarkAt is the mark at offset (or touching it), else the one on its
// line nearest to it; ok false if its line has none.
func noteMarkAt(text string, marks []mdNoteMark, offset int) (mdNoteMark, bool) {
	for _, m := range marks {
		if offset >= m.start && offset <= m.end {
			return m, true
		}
	}
	runes := []rune(text)
	ls := offset
	for ls > 0 && runes[ls-1] != '\n' {
		ls--
	}
	le := offset
	for le < len(runes) && runes[le] != '\n' {
		le++
	}
	best, dist := mdNoteMark{}, -1
	for _, m := range marks {
		if m.start >= ls && m.end <= le {
			d := max(m.start-offset, offset-m.end)
			if dist < 0 || d < dist {
				best, dist = m, d
			}
		}
	}
	return best, dist >= 0
}

// jumpNote goes from the mark at the cursor to its note, or from a note
// to its first mark, and says where it went.
func jumpNote(b *buffer.Buffer) string {
	text := b.Text()
	marks := mdNoteMarks(text)
	m, ok := noteMarkAt(text, marks, b.Cursor())
	if !ok {
		return "No note mark here: put the cursor on a [^1] or its note."
	}
	for _, o := range marks {
		if o.label != m.label || o.def == m.def {
			continue
		}
		at := o.end
		if o.def && at < b.Len() && b.Slice(at, at+1) == " " {
			at++ // the note's text
		}
		b.SetCursor(at, false)
		if o.def {
			return "Note " + m.label + "."
		}
		return "Note " + m.label + "'s mark."
	}
	if m.def {
		return "Nothing refers to note " + m.label + "."
	}
	return "Note " + m.label + " has no text yet: write [^" + m.label + "]: at the end."
}

// noteText is note label's text as it reads (its first lines, short),
// "" if the text has no such note.
func noteText(text, label string) string {
	lines := strings.Split(text, "\n")
	head := "[^" + label + "]:"
	for i, l := range lines {
		t := strings.TrimLeft(l, " ")
		if !strings.HasPrefix(t, head) || len(l)-len(t) > 3 {
			continue
		}
		body := []string{strings.TrimSpace(t[len(head):])}
		for _, c := range lines[i+1:] { // its lines after, indented
			if strings.TrimSpace(c) == "" || !strings.HasPrefix(c, "    ") && !strings.HasPrefix(c, "\t") {
				break
			}
			body = append(body, strings.TrimSpace(c))
		}
		s := strings.Join(strings.Fields(prose.FromMarkdown(strings.Join(body, "\n")).Plain()), " ")
		if r := []rune(s); len(r) > 100 { // a tip is a line
			s = string(r[:99]) + "…"
		}
		return s
	}
	return ""
}

// renumberNotesIn numbers the notes 1, 2, 3… in the order the text refers
// to them (a note nothing refers to after them), their marks and notes
// alike; the notes, when they are all together at the end, put in that
// order. One undo step; it says what it did.
func renumberNotesIn(b *buffer.Buffer) string {
	text := b.Text()
	marks := mdNoteMarks(text)
	if len(marks) == 0 {
		return "No notes to renumber."
	}
	num := map[string]string{}
	var order []string
	for _, defs := range []bool{false, true} { // referred to first, then the rest
		for _, m := range marks {
			if m.def == defs {
				if _, ok := num[m.label]; !ok {
					order = append(order, m.label)
					num[m.label] = strconv.Itoa(len(order))
				}
			}
		}
	}
	cursor := b.Cursor()
	changed := false
	b.Group(func() {
		for i := len(marks) - 1; i >= 0; i-- {
			m := marks[i]
			if n := num[m.label]; n != m.label {
				b.Replace(m.start+2, m.start+2+utf8.RuneCountInString(m.label), n)
				changed = true
			}
		}
		if sortNotes(b) {
			changed = true
		}
		b.SetCursor(min(cursor, b.Len()), false)
	})
	if !changed {
		return "The notes are in order."
	}
	return "Notes renumbered 1 to " + strconv.Itoa(len(order)) + "."
}

// sortNotes puts the notes in their numbers' order when they are all
// together at the text's end (each note's lines, blank lines between
// them), and reports whether it moved any.
func sortNotes(b *buffer.Buffer) bool {
	text := b.Text()
	var defs []mdNoteMark
	for _, m := range mdNoteMarks(text) {
		if m.def {
			defs = append(defs, m)
		}
	}
	if len(defs) < 2 {
		return false
	}
	runes := []rune(text)
	start := defs[0].start
	region := strings.TrimRight(string(runes[start:]), "\n")
	// the notes' blocks: each from its mark to the next's; nothing but
	// notes, their indented lines and blank lines
	var blocks []string
	var labels []string
	blank := false
	for i, d := range defs {
		end := len(runes)
		if i+1 < len(defs) {
			end = defs[i+1].start
		}
		block := strings.TrimRight(string(runes[d.start:end]), "\n")
		if i+1 < len(defs) && strings.Contains(string(runes[d.start:end]), "\n\n") {
			blank = true
		}
		for _, l := range strings.Split(block, "\n")[1:] {
			if strings.TrimSpace(l) != "" && !strings.HasPrefix(l, "    ") && !strings.HasPrefix(l, "\t") {
				return false // text between the notes: left as it is
			}
		}
		blocks = append(blocks, block)
		labels = append(labels, d.label)
	}
	idx := make([]int, len(blocks))
	for i := range idx {
		idx[i] = i
	}
	slices.SortStableFunc(idx, func(a, c int) int {
		x, _ := strconv.Atoi(labels[a])
		y, _ := strconv.Atoi(labels[c])
		return x - y
	})
	if slices.IsSorted(idx) {
		return false
	}
	sep := "\n"
	if blank {
		sep = "\n\n"
	}
	var sorted []string
	for _, i := range idx {
		sorted = append(sorted, blocks[i])
	}
	b.Replace(start, start+utf8.RuneCountInString(region), strings.Join(sorted, sep))
	return true
}

// noteTip is the tip for the pointer at offset of a Markdown text: the
// note's text over its mark ("" if not on a mark, or the note has none).
func noteTip(text string, offset int) string {
	for _, m := range mdNoteMarks(text) {
		if !m.def && offset >= m.start && offset < m.end {
			if t := noteText(text, m.label); t != "" {
				return m.label + ". " + t
			}
			return "Note " + m.label + " has no text yet."
		}
	}
	return ""
}

// noteHover shows the note's text under its mark while the pointer is
// on it in the main editor (a Markdown document's).
func (w *MainWindow) noteHover(offset int, at fyne.Position, cell fyne.Size) {
	if !w.markdown {
		w.chipHover(offset, at, cell) // a screenplay's: its links' chips (link_chips.go)
		return
	}
	if w.tooltips == nil {
		return
	}
	tip := ""
	if offset >= 0 {
		tip = noteTip(w.textEditor.Text(), offset)
	}
	if tip == w.noteTipShown {
		return
	}
	if w.noteTipShown != "" && w.tooltips.Text() == w.noteTipShown {
		w.tooltips.HideTip()
	}
	w.noteTipShown = tip
	if tip != "" {
		w.tooltips.ShowTipUnder(tip, w.textEditor, at, cell)
	}
}
