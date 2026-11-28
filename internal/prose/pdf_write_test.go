// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package prose

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/pdf"
	"github.com/LaPingvino/lexington/pdfin"
)

// more is what the story has not got: code, a break, a table, a line
// break, struck and code text, a long word.
const more = "\n## Three\n\n```\nfunc light() {\n\treturn \"on\"\n}\n```\n\n* * *\n\n" +
	"| Room | Things |\n| --- | --- |\n| Kitchen | a kettle, two cups |\n| Lamp room | the **lens** |\n\n" +
	"A line,  \nbroken. Some ~~struck~~ and `code` text, and Llanfairpwllgwyngyllgogerychwyrndrobwllllantysiliogogogoch.\n"

// The PDF of a document: its text in order, a page for each of the
// drawing's, the title page first.
func TestWritePDF(t *testing.T) {
	d := FromMarkdown(story + more)
	d.TitlePage = []string{"The Lighthouse Keeper", "A. Writer"}
	d.Paper = "a5"
	var b bytes.Buffer
	drawing, err := d.WritePDF(&b, PDFFonts{})
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "doc.pdf")
	if err := os.WriteFile(file, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	pages, err := pdfin.Pages(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != len(drawing.Pages) || len(pages) < 3 {
		t.Fatalf("%d pages in the PDF, %d drawn", len(pages), len(drawing.Pages))
	}
	if w := drawing.Pages[0].Width; w < 415 || w > 422 {
		t.Errorf("A5's width %v", w)
	}
	var text strings.Builder
	for _, p := range pages {
		for _, l := range p {
			text.WriteString(l.Text + "\n")
		}
	}
	all := strings.Join(strings.Fields(text.String()), " ")
	for _, want := range []string{
		"The Lighthouse Keeper A. Writer", "The wind came in from the west, and the rain followed it", "Nobody went out that night.",
		"• a brass compass", "1. read once", "The sea gives back what it takes", "func light() {", `return "on"`,
		"Room Things", "Kitchen a kettle, two cups", "A line, broken.", "1. Not even the fishermen.",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("no %q in\n%s", want, all)
		}
	}
	if dir := os.Getenv("SHOT_DIR"); dir != "" {
		r := pdf.NewRasterizer(&drawing)
		for i := range drawing.Pages {
			f, _ := os.Create(filepath.Join(dir, "prose-page-"+string(rune('1'+i))+".png"))
			png.Encode(f, r.Page(i, 1.5))
			f.Close()
		}
	}
}

// What the PDF cannot print yet: characters none of its fonts has (text
// read right to left it prints).
func TestPDFMissing(t *testing.T) {
	if m := FromMarkdown(story).PDFMissing(PDFFonts{}); m != "" {
		t.Errorf("the story: %q", m)
	}
	for _, text := range []string{"שָׁלוֹם", "بِسْمِ اللَّهِ", "می‌خواهم", "हिन्दी।"} {
		if m := FromMarkdown(text).PDFMissing(PDFFonts{}); m != "" {
			t.Errorf("%s: %q", text, m)
		}
	}
	if m := FromMarkdown("日本語").PDFMissing(PDFFonts{}); !strings.Contains(m, "日") {
		t.Errorf("Japanese: %q", m)
	}
}

// written is the document's PDF and its pages as drawn.
func written(t *testing.T, d *Doc) ([]byte, pdf.Drawing) {
	t.Helper()
	var b bytes.Buffer
	drawing, err := d.WritePDF(&b, PDFFonts{})
	if err != nil {
		t.Fatal(err)
	}
	return b.Bytes(), drawing
}

// textAt is where text starting with s is drawn (its words and spaces
// runs of their own): its page's index and its first word's baseline (-1 if
// nowhere).
func textAt(d pdf.Drawing, s string) (int, float64) {
	for i, p := range d.Pages {
		var runs []pdf.Text
		for _, m := range p.Marks {
			if t, ok := m.(pdf.Text); ok {
				runs = append(runs, t)
			}
		}
		for j := range runs {
			var b strings.Builder
			for k := j; k < len(runs) && b.Len() < len(s); k++ {
				b.WriteString(runs[k].Text)
			}
			if strings.HasPrefix(b.String(), s) {
				return i, runs[j].Y
			}
		}
	}
	return -1, 0
}

// linksTo are the links of a drawing to url (by page index).
func linksTo(d pdf.Drawing, url string) map[int][]pdf.Link {
	out := map[int][]pdf.Link{}
	for i, p := range d.Pages {
		for _, l := range p.Links {
			if l.URL == url {
				out[i] = append(out[i], l)
			}
		}
	}
	return out
}

const linked = "# Book\n\n## Chapter Ōne\n\nSee [the harbour](https://example.org/harbour) at dawn, or go on to " +
	"[the second chapter](#chapter-two), or [nowhere](#no-such-heading). A long link wraps: " +
	"[a link whose words are many and run on and on past the end of the line, over two lines at least](https://example.org/long).\n\n" +
	"A claim.[^a]\n\n## Chapter Two\n\nThe end.\n\n[^a]: The note, with [its own link](https://example.org/note).\n"

// Links work in the PDF: an address over its text (a wrapped one over
// each line's part), a heading's anchor to the heading, a note's number
// to the note and the note's back to it; an anchor no heading has is not
// a link. The headings are the PDF's outline.
func TestPDFLinks(t *testing.T) {
	data, d := written(t, FromMarkdown(linked))
	harbour := linksTo(d, "https://example.org/harbour")
	if len(harbour[0]) != 1 {
		t.Fatalf("the harbour's links: %v", harbour)
	}
	pi, base := textAt(d, "harbour")
	if l := harbour[0][0]; pi != 0 || base < l.Y || base > l.Y+l.H || l.W < 50 {
		t.Errorf("the harbour's link %+v, its text on page %d at %v", l, pi, base)
	}
	if long := linksTo(d, "https://example.org/long"); len(long[0]) < 2 || long[0][0].Y >= long[0][1].Y {
		t.Errorf("the long link: %v", long)
	}
	var internal []pdf.Link
	for _, l := range d.Pages[0].Links {
		if l.URL == "" {
			internal = append(internal, l)
		}
	}
	// the second chapter's anchor, the note's number, the note back to it
	if len(internal) != 3 {
		t.Fatalf("internal links: %+v", internal)
	}
	_, two := textAt(d, "Chapter")
	page, twoY := 0, 0.0
	for i, p := range d.Pages {
		for _, m := range p.Marks {
			if t, ok := m.(pdf.Text); ok && t.Text == "Two" {
				page, twoY = i, t.Y
			}
		}
	}
	if l := internal[0]; l.Page != page+1 || l.DestY > twoY || l.DestY < twoY-30 {
		t.Errorf("the chapter's link goes to page %d at %v; it is on page %d at %v (%v)", l.Page, l.DestY, page+1, twoY, two)
	}
	_, noteY := textAt(d, "The note")
	_, claimY := textAt(d, "claim")
	if l := internal[1]; l.Page != 1 || l.DestY > noteY || l.DestY < noteY-20 {
		t.Errorf("the number's link: %+v (the note at %v)", l, noteY)
	}
	if l := internal[2]; l.Page != 1 || l.DestY > claimY || l.DestY < claimY-20 || l.Y < claimY {
		t.Errorf("the note's link back: %+v (the claim at %v)", l, claimY)
	}
	if n := linksTo(d, "https://example.org/note"); len(n[0]) != 1 || n[0][0].Y < claimY {
		t.Errorf("the note's own link: %v", n)
	}
	for _, want := range []string{"/Outlines", "/URI (https://example.org/harbour)"} {
		if !bytes.Contains(data, []byte(want)) {
			t.Errorf("no %s in the PDF", want)
		}
	}
	if FromMarkdown(linked).Blocks[1].ID != "chapter-ōne" {
		t.Errorf("the anchor: %q", FromMarkdown(linked).Blocks[1].ID)
	}
	if d := FromMarkdown("# Notes\n\n# Notes\n\n# ¿Qué?\n"); d.Blocks[1].ID != "notes-1" || d.Blocks[2].ID != "qué" {
		t.Errorf("the anchors: %q %q", d.Blocks[1].ID, d.Blocks[2].ID)
	}
}

// filler is n paragraphs of some lines each, numbered (P1…).
func filler(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		b.WriteString("P" + strconv.Itoa(i) + " The keeper climbed the stairs again, as he did every evening, and counted the steps " +
			"one by one until the lamp room, where the glass was cold and the wind sang in the frames.\n\n")
	}
	return b.String()
}

// longNote is a note of some words, end its last.
func longNote(words int, end string) string {
	return strings.Repeat("and on and on ", words/4) + end
}

// noteFoot checks the notes at the feet: each note's first line on the
// page of its reference, under all the page's text.
func notesUnderText(t *testing.T, d pdf.Drawing) {
	t.Helper()
	for i, p := range d.Pages {
		rule := -1.0
		for _, m := range p.Marks {
			if b, ok := m.(pdf.Box); ok && b.Fill && b.H == 0.5 {
				rule = b.Y
			}
		}
		if rule < 0 {
			continue
		}
		for _, m := range p.Marks {
			if tx, ok := m.(pdf.Text); ok && tx.Size == textSize && tx.Y > rule {
				t.Errorf("page %d: text %q under the notes' rule", i+1, tx.Text)
			}
		}
	}
}

// Notes at the page's foot: on the page that refers to them, smaller,
// under a short rule, the text over them; a reference near the page's
// end goes to the next page with its note; a long note goes on at the
// next page's foot; a note referred to twice keeps its number.
func TestPageFootnotes(t *testing.T) {
	src := "A first claim.[^1] Again.[^1]\n\n" + filler(9) + "Near the end, a claim with a long note.[^2]\n\n" + filler(8) +
		"The very long one.[^3]\n\n" + filler(12) +
		"[^1]: The first note.\n[^2]: The second, " + longNote(60, "SECOND-END.") + "\n[^3]: The third, " + longNote(1800, "THIRD-END.") + "\n"
	_, d := written(t, FromMarkdown(src))
	notesUnderText(t, d)
	for _, c := range []struct{ ref, note string }{{"A", "The first"}, {"Near", "The second"}, {"The very", "The third"}} {
		rp, _ := textAt(d, c.ref)
		np, ny := textAt(d, c.note)
		if rp < 0 || rp != np {
			t.Errorf("%q on page %d, its note %q on page %d", c.ref, rp+1, c.note, np+1)
		}
		if ny < d.Pages[np].Height*0.4 {
			t.Errorf("%q's note is not at the foot (%v)", c.ref, ny)
		}
	}
	np, _ := textAt(d, "The third")
	lp, ly := textAt(d, "THIRD-END.")
	for p := np; p <= lp; p++ { // carried on, page by page
		has := false
		for _, m := range d.Pages[p].Marks {
			if tx, ok := m.(pdf.Text); ok && tx.Size == noteSize && strings.HasPrefix(tx.Text, "and") {
				has = true
			}
		}
		if !has {
			t.Errorf("page %d has none of the long note", p+1)
		}
	}
	if lp <= np || ly < d.Pages[lp].Height*0.3 {
		t.Errorf("the long note ends on page %d (it starts on %d) at %v", lp+1, np+1, ly)
	}
	ones := 0
	for _, p := range d.Pages {
		for _, m := range p.Marks {
			if tx, ok := m.(pdf.Text); ok && tx.Text == "1" && tx.Size < noteSize {
				ones++
			}
		}
	}
	if ones != 2 {
		t.Errorf("the first note's number %d times in the text", ones)
	}
	if n, _ := textAt(d, "1."); n != 0 {
		t.Errorf("the first note on page %d", n+1)
	}
	if s, _ := textAt(d, "2."); s < 0 {
		t.Error("no second note")
	}
	if dir := os.Getenv("SHOT_DIR"); dir != "" {
		r := pdf.NewRasterizer(&d)
		for i := range d.Pages {
			f, _ := os.Create(filepath.Join(dir, "footnotes-page-"+strconv.Itoa(i+1)+".png"))
			png.Encode(f, r.Page(i, 1))
			f.Close()
		}
	}
}

// A reference on the page's last line whose note has no room under it
// goes to the next page, with the note.
func TestFootnoteMovesWithItsReference(t *testing.T) {
	for n := 1; n <= 14; n++ { // the reference at each place near a page's end
		src := filler(n) + "The claim.[^1]\n\n" + filler(3) + "[^1]: " + longNote(40, "End.") + "\n"
		_, d := written(t, FromMarkdown(src))
		rp, _ := textAt(d, "The claim")
		np, _ := textAt(d, "1.")
		if rp != np {
			t.Errorf("%d paragraphs before: the reference on page %d, its note on %d", n, rp+1, np+1)
		}
		notesUnderText(t, d)
	}
}

// After each chapter, after the text; numbered through, or again in each
// chapter (after the text, under the chapter's heading).
func TestEndnotes(t *testing.T) {
	src := "# Book\n\n## One\n\nFirst.[^a] Second.[^b]\n\n## Two\n\nThird.[^c]\n\n[^a]: Note a.\n[^b]: Note b.\n[^c]: Note c.\n"
	nums := func(d pdf.Drawing) string {
		var out []string
		for _, p := range d.Pages {
			for _, m := range p.Marks {
				if tx, ok := m.(pdf.Text); ok && tx.Size == noteSize && regexp.MustCompile(`^\d+\.$`).MatchString(tx.Text) {
					out = append(out, tx.Text)
				}
			}
		}
		return strings.Join(out, " ")
	}
	all := func(d pdf.Drawing) string {
		var at []string
		for _, p := range d.Pages {
			for _, m := range p.Marks {
				if tx, ok := m.(pdf.Text); ok {
					at = append(at, tx.Text)
				}
			}
		}
		return strings.Join(at, "")
	}
	order := func(d pdf.Drawing, words ...string) bool {
		all := all(d)
		last := -1
		for _, w := range words {
			i := strings.Index(all[last+1:], w)
			if i < 0 {
				return false
			}
			last += 1 + i
		}
		return true
	}
	doc := FromMarkdown(src)
	doc.NotesAt = NotesAtChapterEnd
	_, d := written(t, doc)
	if !order(d, "First.", "a.", "b.", "Two", "Third.", "c.") || nums(d) != "1. 2. 3." {
		t.Errorf("after each chapter: %s in %s", nums(d), all(d))
	}
	doc.NotesPerChapter = true
	if _, d = written(t, doc); nums(d) != "1. 2. 1." {
		t.Errorf("after each chapter, numbered again: %s", nums(d))
	}
	doc.NotesAt = NotesAtEnd
	_, d = written(t, doc)
	if !order(d, "First.", "Third.", "One", "a.", "b.", "Two", "c.") || nums(d) != "1. 2. 1." {
		t.Errorf("after the text, numbered again: %s", nums(d))
	}
	doc.NotesPerChapter = false
	if _, d = written(t, doc); !order(d, "Third.", "a.", "b.", "c.") || nums(d) != "1. 2. 3." {
		t.Errorf("after the text: %s", nums(d))
	}
	doc.NotesAt, doc.NotesPerChapter = NotesAtPageFoot, true
	if _, d = written(t, doc); nums(d) != "1. 2. 1." {
		t.Errorf("at the foot, numbered again: %s", nums(d))
	}
}

// A note in a table cell is numbered once; an empty note and a note of
// its own page's length end.
func TestFootnoteEdges(t *testing.T) {
	src := "| A | B |\n| --- | --- |\n| x[^1] | y[^2] |\n\nThen.[^3]\n\n[^1]: One.\n[^2]: Two.\n[^3]: " + longNote(3000, "HUGE-END.") + "\n"
	_, d := written(t, FromMarkdown(src))
	if _, y := textAt(d, "3."); y == 0 {
		t.Error("no third note")
	}
	if lp, _ := textAt(d, "HUGE-END."); lp < 2 {
		t.Errorf("the huge note ends on page %d", lp+1)
	}
	notesUnderText(t, d)
}
