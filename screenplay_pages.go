package main

import (
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/LaPingvino/lexington/pdfin"

	"github.com/LaPingvino/accolade/internal/scriptdiff"
)

// A screenplay source's pages and the writer's decisions on it (LOOP.md,
// SCREENPLAY SOURCES):
//
//   - Keep Mine is remembered with the document's dismissed findings
//     (dismissed_place.go: beside it or in its notes block, as that
//     setting says), a line a difference, so it holds across sessions and
//     deleting the line brings the difference back.
//   - A source read from a PDF keeps its pages' text as printed beside it
//     (a kept copy's own folder, or the bundle): the Source pane shows the
//     page of the cursor's scene in its layout (each line where the page
//     has it, in a fixed-width font), and "Page 12 Looks Right" ticks a
//     page, kept as Keep Mine is, so its differences fold away. Text, not
//     an image: Accolade's rasterizer paints its own print (pageview.go),
//     not any PDF, and the PDF's text is what is compared anyway.

// sourcePagesName is the pages' file in a kept source copy's own folder
// (keepSourceCopy), which goes with it (dropKeptCopy).
const sourcePagesName = "pages.txt"

// sourcePagesExtra is the pages' file in a bundle.
const sourcePagesExtra = productionDir + "source-pages.txt"

// layoutPages are a PDF's pages as text in their layout: a line at its
// indent (ten characters an inch, as a script's Courier), blank lines
// where the page leaves room, pages apart by a form feed line.
func layoutPages(ps [][]pdfin.PageLine) string {
	left := math.Inf(1)
	for _, p := range ps {
		for _, l := range p {
			left = math.Min(left, l.X)
		}
	}
	var b strings.Builder
	for i, p := range ps {
		if i > 0 {
			b.WriteString("\f\n")
		}
		for j, l := range p {
			if j > 0 {
				gap := int(math.Round((l.Y-p[j-1].Y)*6)) - 1 // six lines an inch
				b.WriteString(strings.Repeat("\n", min(max(gap, 0), 3)))
			}
			b.WriteString(strings.Repeat(" ", max(int(math.Round((l.X-left)*10)), 0)))
			b.WriteString(strings.TrimRightFunc(l.Text, unicode.IsSpace))
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// pdfPagesText is the pages of the PDF at path as layoutPages has them,
// "" if it has none to read.
func pdfPagesText(path string) string {
	ps, err := pdfin.Pages(path)
	if err != nil {
		return ""
	}
	for _, p := range ps {
		if len(p) > 0 {
			return layoutPages(ps)
		}
	}
	return ""
}

// pageIndex is a source's pages, and their words in order (lower case,
// a word broken over a line joined) with the page each is on.
type pageIndex struct {
	pages []string
	words []string
	page  []int
}

// newPageIndex reads layoutPages' text (nil: no pages).
func newPageIndex(s string) *pageIndex {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	p := &pageIndex{pages: strings.Split(strings.TrimSuffix(s, "\n"), "\n\f\n")}
	for i, pg := range p.pages {
		for _, w := range pageWords(hyphenRe.ReplaceAllString(pg, "$1$2")) {
			p.words = append(p.words, w)
			p.page = append(p.page, i)
		}
	}
	return p
}

// pageWords are a text's words as pages are searched for them.
func pageWords(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// find is the page (from 0) of text's first words, looked for from the
// word at onwards and then from the start, and where they are; -1 if
// they are not.
func (p *pageIndex) find(text string, at int) (page, where int) {
	ws := pageWords(text)
	if p == nil || len(ws) == 0 {
		return -1, at
	}
	for _, n := range []int{min(len(ws), 8), min(len(ws), 3)} {
		for _, from := range []int{at, 0} {
			if i := p.index(ws[:n], from); i >= 0 {
				return p.page[i], i
			}
		}
	}
	return -1, at
}

// index is where the words ws are, from the word from on (-1: not).
func (p *pageIndex) index(ws []string, from int) int {
	for i := max(from, 0); i+len(ws) <= len(p.words); i++ {
		if p.words[i] != ws[0] {
			continue
		}
		ok := true
		for j := 1; j < len(ws) && ok; j++ {
			ok = p.words[i+j] == ws[j]
		}
		if ok {
			return i
		}
	}
	return -1
}

// pagesOf are each change's page in the source (-1: not known): where
// the source's words of it are, else the script's, else its neighbour's
// before it, looked for in the script's order.
func pagesOf(p *pageIndex, m *scriptdiff.Merge, at []int) []int {
	out := make([]int, len(m.Changes))
	for i := range out {
		out[i] = -1
	}
	if p == nil {
		return out
	}
	order := make([]int, len(m.Changes))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return at[a] - at[b] })
	last, pos := -1, 0
	for _, id := range order {
		c := m.Changes[id]
		pg, w := p.find(c.Theirs, pos)
		if pg < 0 {
			pg, w = p.find(c.Ours, pos)
		}
		if pg < 0 {
			pg = last
		} else {
			pos = w
		}
		out[id], last = pg, pg
	}
	return out
}

// The decisions, as lines among the dismissed findings.
const (
	keptMineHead  = "Source, kept mine: "
	pageRightHead = "Source page "
)

// sourceDecision reports whether a dismissed line is the Source pane's.
func sourceDecision(k string) bool {
	return strings.HasPrefix(k, keptMineHead) || strings.HasPrefix(k, pageRightHead)
}

// shortHash is a text's hash, short: what tells two lines apart.
func shortHash(s string) string {
	h := fnv.New32a()
	h.Write([]byte(s))
	return fmt.Sprintf("%08x", h.Sum32())
}

// keptLine is Keep Mine's line for a difference: a few of its words to
// read, and its hash.
func keptLine(c scriptdiff.Change) string {
	about := strings.Join(strings.Fields(c.Ours+" "+c.Theirs), " ")
	if r := []rune(about); len(r) > 50 {
		about = string(r[:50]) + "…"
	}
	return keptMineHead + about + " · #" + shortHash(changeKey(c))
}

// rightLine is the tick's line for page i (from 0): its number and its
// text's hash, so another print's page 12 is not ticked.
func (p *pageIndex) rightLine(i int) string {
	return fmt.Sprintf("%s%d looks right · #%s", pageRightHead, i+1, shortHash(p.pages[i]))
}

// readSourcePages are the pages kept beside the source r ("" none).
func (w *MainWindow) readSourcePages(r sourceRef) string {
	switch r.Origin {
	case sourceBundle:
		if f := w.freshPages; f != nil && w.bundle != nil && f.bundle == w.bundle.Path {
			return f.pages // being written into it
		}
		if w.bundle != nil {
			if data, err := w.bundle.readExtra(sourcePagesExtra); err == nil {
				return string(data)
			}
		}
	case sourceKept:
		if p := keptPagesFile(r.Path); p != "" {
			if data, err := os.ReadFile(p); err == nil {
				return string(data)
			}
		}
	}
	return ""
}

// keptPagesFile is the pages' file of a source copy Accolade keeps in a
// folder of its own ("" for any other file).
func keptPagesFile(source string) string {
	dir := filepath.Dir(source)
	if filepath.Dir(dir) != filepath.Clean(keptSourcesDir()) {
		return ""
	}
	return filepath.Join(dir, sourcePagesName)
}

// keepSourcePages keeps a newly chosen source's pages beside it (""
// none: a bundle's old ones go).
func (w *MainWindow) keepSourcePages(pages string) {
	if w.source != nil && w.source.sp != nil {
		defer func() { w.source.sp.pagesIn = false }() // the pane reads them again
	}
	if w.bundle != nil {
		if _, err := w.bundle.readExtra(sourcePagesExtra); pages == "" && err != nil && w.freshPages == nil {
			return // none there
		}
		w.freshPages = &freshPages{w.bundle.Path, pages}
		files, gone := []bundleFile{{sourcePagesExtra, []byte(pages)}}, []string(nil)
		if pages == "" {
			files, gone = nil, []string{sourcePagesExtra}
		}
		w.queueBundle(bundleJob{run: func(b *textBundle) (*textBundle, error) {
			if b == nil {
				return nil, fmt.Errorf("the script's bundle is gone")
			}
			return saveBundleFilesWithout(b, files, gone, false)
		}})
		return
	}
	if pages == "" {
		return // a new copy's folder is new
	}
	if r := w.sourceRef(); r.Origin == sourceKept {
		if p := keptPagesFile(r.Path); p != "" {
			_ = writeFileAtomic(p, []byte(pages))
		}
	}
}

// freshPages are a bundle's source pages until its queue has written
// them.
type freshPages struct{ bundle, pages string }

// storeSourceDecisions keeps the Source pane's decisions among the
// document's dismissed findings: add added, the pane's lines keep says
// no more let go, the findings' kept as they are.
func (w *MainWindow) storeSourceDecisions(add []string, keep func(string) bool) error {
	set := map[string]bool{}
	for k := range w.allDismissed() {
		if !sourceDecision(k) || keep == nil || keep(k) {
			set[k] = true
		}
	}
	for _, k := range add {
		set[k] = true
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	return w.storeDismissed(keys)
}
