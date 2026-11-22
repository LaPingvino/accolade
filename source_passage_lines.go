package main

import (
	"strings"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Where the source's passages are in its Markdown, and what goes with the
// main cursor: the Cards pane lights the scenes linking to the passage
// under the source's cursor (the source in the middle, middle.go), and the
// link offer asks when the cursor leaves a passage whose opening changed
// (source_links_offer.go).

// litScenes are the items to light for the main cursor: with the source
// in the main editor, the scenes linking to the passage there (and what
// is said of it); else the script cursor's scene. ok is false while the
// source is being read.
func (w *MainWindow) litScenes(items []outlineItem, seen *linesSeen) (lit map[int]bool, info string, ok bool) {
	lit = map[int]bool{}
	if d := w.midDoc(); d != nil && d.kind == docSource {
		s := w.sourcePane()
		s.loadSource(true)
		if s.loading != nil {
			return nil, "Reading the source…", false
		}
		m := s.current()
		p := passageAt(d.Text(), m.passages, d.ed.Buffer(), seen)
		if p < 0 {
			info = "Before the source's first passage"
		} else {
			lines := m.linked(p)
			for i, it := range items {
				for _, l := range lines {
					if it.Line == l {
						lit[i] = true
					}
				}
			}
			q := "“" + shortQuote(m.passages[p].Plain) + "”"
			switch st := statusOf(m, p); {
			case len(lines) > 0:
				info = q + ": " + plural(len(lines), "scene") + " from it"
			case st == statusCut:
				info = q + ": cut on purpose"
			case st == statusNone:
				info = q
			default:
				info = q + ": not adapted yet"
			}
		}
	} else {
		line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
		at := -1
		for i, it := range items {
			if it.Line <= line {
				at = i
			}
		}
		if at >= 0 {
			lit[at] = true
			info = "The cursor's scene: " + items[at].Text
		}
	}
	return lit, info, true
}

// statusOf is passage p's status in the model.
func statusOf(m *sourceModel, p int) passageStatus {
	if p >= 0 && p < len(m.status) {
		return m.status[p]
	}
	return statusNone
}

// passageLines are the lines of a source's Markdown each passage starts
// on (-1: not found). passagesOf keeps no places, so each passage is
// looked for by its first words, after the one before it: on the first
// line from there whose words hold them (a list's mark or number, a
// heading's #, a quote's > before them).
func passageLines(src string, ps []passage) []int {
	lines := strings.Split(src, "\n")
	body := markdownBody(src)
	from := len(lines) - len(strings.Split(body, "\n")) // the front matter's lines
	end := strings.Count(src[:markdownNotes.startByte(src)], "\n") + 1
	out := make([]int, len(ps))
	at := max(0, from)
	for i, p := range ps {
		out[i] = -1
		if len(p.Words) == 0 {
			continue
		}
		want := p.Words[:min(3, len(p.Words))]
		for l := at; l < min(end, len(lines), at+passageReach); l++ {
			if opensWith(normWords(lines[l]), want) {
				out[i], at = l, l+1
				break
			}
		}
	}
	return out
}

// passageReach is how many lines on a passage is looked for: one not
// found there (its words changed) is left out, the next looked for from
// where the one before it was.
const passageReach = 400

// opensWith reports whether want comes in ws, at its start or after a
// mark or two (a list's number).
func opensWith(ws, want []string) bool {
	for skip := 0; skip <= 2 && skip+len(want) <= len(ws); skip++ {
		ok := true
		for j, x := range want {
			if ws[skip+j] != x {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// linesSeen are the passages' lines found last (the cursor moves often,
// the text not).
type linesSeen struct {
	rev   uint64
	buf   *buffer.Buffer
	ps    []passage
	lines []int
}

// passageAt is the passage the cursor of a source's editor is in: the
// last one starting on its line or before (-1: before the first).
func passageAt(src string, ps []passage, b *buffer.Buffer, c *linesSeen) int {
	line, _ := b.Position(b.Cursor())
	at := -1
	if c.rev != b.Rev() || c.buf != b || !samePassages(c.ps, ps) {
		c.rev, c.buf, c.ps, c.lines = b.Rev(), b, ps, passageLines(src, ps)
	}
	for i, l := range c.lines {
		if l >= 0 && l <= line {
			at = i
		}
	}
	return at
}
