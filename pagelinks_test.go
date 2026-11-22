package main

import (
	"encoding/json"
	"fmt"
	"image"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/LaPingvino/lexington/pdf"

	"github.com/LaPingvino/accolade/internal/prose"
)

// mdLinksDoc has links of every kind: an address, a script's (not
// opened), a heading's anchor on a later page, a note.
var mdLinksDoc = "# The Keeper\n\n## One\n\nSee [the harbour](https://example.org/harbour) at dawn, " +
	"or [this](javascript:alert(1)), or go on to [the second chapter](#two).\n\n" +
	"He counted the steps.[^1]\n\n" + strings.Repeat("The keeper climbed the stairs again, as he did every evening, and counted the steps one by one.\n\n", 30) +
	"## Two\n\nThe end.\n\n[^1]: There were a hundred and twelve, and one of them creaked.\n"

// mdPagesOf shows text's pages in the Live Preview of a window size big
// (set up by before first, if set).
func mdPagesOf(t *testing.T, text string, size fyne.Size, before ...func(*MainWindow)) (*MainWindow, *pageView, chan *pageView) {
	t.Helper()
	w, _ := newDialogTestWindow(t)
	w.setMarkdown(true)
	w.textEditor.SetText(text)
	w.fyneWindow.Resize(size)
	for _, f := range before {
		f(w)
	}
	w.preview()
	shown := make(chan *pageView, 8)
	onPagesShown(t, func(v *pageView) { shown <- v })
	w.previewPane.mode.SetSelected(previewText) // (Pages kept from before: shown again)
	w.previewPane.mode.SetSelected(previewPages)
	t.Cleanup(func() { w.settings.SetBoolean(previewPagesPref, false) })
	select {
	case v := <-shown:
		return w, v, shown
	case <-time.After(30 * time.Second):
		t.Fatal("no pages")
	}
	return nil, nil, nil
}

// linkPoint is the middle of a link of the view, in its area's place.
func linkPoint(v *pageView, i int) fyne.Position {
	pos, size := v.linkRect(v.links[i])
	return pos.Subtract(v.scroll.Offset).AddXY(size.Width/2, size.Height/2)
}

// linkIndex is the view's link to url (internal: the n-th going to a
// place), -1 if none.
func linkIndex(v *pageView, url string, n int) int {
	for i, l := range v.links {
		if l.URL == url {
			if n == 0 {
				return i
			}
			n--
		}
	}
	return -1
}

// The page view's links: a hand and the address in a tip over one; a
// click opens a web address (not a script's), goes to a heading, to a
// note and back; Tab goes through them from the page in sight, Enter
// follows; past the last, Tab leaves the pages.
func TestPageViewLinks(t *testing.T) {
	var opened []string
	old := openLink
	openLink = func(u *url.URL) { opened = append(opened, u.String()) }
	t.Cleanup(func() { openLink = old })
	w, v, _ := mdPagesOf(t, mdLinksDoc, fyne.NewSize(1200, 800))
	a := v.area
	harbour := linkIndex(v, "https://example.org/harbour", 0)
	if harbour < 0 || len(v.links) != 4 { // javascript: is no link (safeAddress)
		t.Fatalf("links %+v", v.links)
	}
	at := linkPoint(v, harbour)
	a.MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: at}})
	if v.hoverLink != harbour || a.Cursor() != desktop.PointerCursor || !v.marks.lit.Visible() {
		t.Errorf("over the link: lit %d, cursor %v", v.hoverLink, a.Cursor())
	}
	if tip := w.tooltips.Text(); tip != "https://example.org/harbour" {
		t.Errorf("the tip %q", tip)
	}
	a.MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(2, 2)}})
	if v.hoverLink != -1 || a.Cursor() != desktop.DefaultCursor || v.marks.lit.Visible() || w.tooltips.Text() != "" {
		t.Error("off the link, still lit")
	}
	a.Tapped(&fyne.PointEvent{Position: at})
	if !slices.Equal(opened, []string{"https://example.org/harbour"}) {
		t.Errorf("opened %v", opened)
	}
	if linkIndex(v, "javascript:alert(1)", 0) >= 0 || len(opened) != 1 {
		t.Errorf("a script's link became a link (opened %v)", opened)
	}
	// the chapter's anchor: page 2 in sight, the heading at the top
	chapter := linkIndex(v, "", 0)
	if l := v.links[chapter]; l.Page != 2 {
		t.Fatalf("the chapter's link goes to page %d", l.Page)
	}
	a.Tapped(&fyne.PointEvent{Position: linkPoint(v, chapter)})
	tops, _ := v.pageTops()
	if page, _ := v.anchor(); page != 1 || v.scroll.Offset.Y < tops[1] {
		t.Errorf("at page %d, offset %v (page 2 at %v)", page+1, v.scroll.Offset.Y, tops[1])
	}
	// the note's number to the note at page 1's foot, and back
	a.TypedKey(&fyne.KeyEvent{Name: fyne.KeyHome})
	number := linkIndex(v, "", 1)
	a.Tapped(&fyne.PointEvent{Position: linkPoint(v, number)})
	note := v.scroll.Offset.Y
	if dest := tops[0] + float32(v.links[number].DestY)*v.scale; note < dest-3*pageGap || note > dest {
		t.Errorf("at %v, the note at %v", note, dest)
	}
	back := linkIndex(v, "", 2)
	if v.links[back].Page != 1 || v.links[back].DestY >= v.links[number].DestY {
		t.Errorf("the note's link back %+v", v.links[back])
	}
	a.Tapped(&fyne.PointEvent{Position: linkPoint(v, back)})
	if v.scroll.Offset.Y >= note {
		t.Error("the note's number did not go back")
	}
	// the keyboard: the pages focused again at the top, Tab goes through
	// the links, Enter follows
	a.TypedKey(&fyne.KeyEvent{Name: fyne.KeyHome})
	w.fyneWindow.Canvas().Unfocus()
	w.fyneWindow.Canvas().Focus(a)
	var went []int
	for a.AcceptsTab() && len(went) < 10 {
		a.TypedKey(&fyne.KeyEvent{Name: fyne.KeyTab})
		went = append(went, v.focusLink)
		if !v.marks.ring.Visible() {
			t.Errorf("link %d not ringed", v.focusLink)
		}
	}
	if !slices.Equal(went, []int{0, 1, 2, 3}) {
		t.Errorf("Tab went %v", went)
	}
	a.KeyDown(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
	if !a.AcceptsTab() {
		t.Error("Shift+Tab has nowhere to go")
	}
	for range 3 {
		a.TypedKey(&fyne.KeyEvent{Name: fyne.KeyTab})
	}
	a.KeyUp(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
	if v.focusLink != 0 {
		t.Errorf("back to %d", v.focusLink)
	}
	a.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if len(opened) != 2 {
		t.Errorf("Enter opened %v", opened)
	}
	// a screenplay's pages have none: Tab leaves them
	s := newTestWindow(t, pageViewScript)
	if sv := previewAndWait(t, s); len(sv.links) != 0 || sv.area.AcceptsTab() {
		t.Errorf("a screenplay's links %v", sv.links)
	}
}

// fnDoc is a document with a note whose reference is near a page's end.
func fnDoc(before int) string {
	var b strings.Builder
	b.WriteString("# Notes\n\n")
	for i := 1; i <= before; i++ {
		fmt.Fprintf(&b, "P%d The keeper climbed the stairs again, as he did every evening, and counted the steps one by one.\n\n", i)
	}
	b.WriteString("The claim that needs its note.[^1]\n\n")
	for i := 1; i <= 30; i++ {
		fmt.Fprintf(&b, "Q%d The lamp turned, and the light went out over the water, and came back, and went out again.\n\n", i)
	}
	b.WriteString("[^1]: The note, long enough to take some lines at the foot: " + strings.Repeat("the steps, the lamp, the glass, ", 20) + "and the wind.\n")
	return b.String()
}

// pagesWith is the page (index) a drawing prints text starting with s on.
func pagesWith(d pdf.Drawing, s string) int {
	for i, p := range d.Pages {
		var all strings.Builder
		for _, m := range p.Marks {
			if t, ok := m.(pdf.Text); ok {
				all.WriteString(t.Text)
			}
		}
		if strings.Contains(all.String(), s) {
			return i
		}
	}
	return -1
}

// samePaint reports whether two paintings are the same.
func samePaint(a, b image.Image) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
			r1, g1, b1, _ := a.At(x, y).RGBA()
			r2, g2, b2, _ := b.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 {
				return false
			}
		}
	}
	return true
}

// An edit near a note's reference moves it, and its note, to the next
// page: the pages shown after it are the document's as edited, made
// whole (Markdown's pages are made again whole at each pause; only the
// pages that look otherwise are painted again), their paintings and
// links the new ones.
func TestFootnoteReflowInPreview(t *testing.T) {
	// a reference on page 1's last lines: a paragraph more before it
	// moves it to page 2
	claimPage := func(n int) int {
		_, d, err := markdownPDF(fnDoc(n), "Notes", mdPageSetup{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		return pagesWith(d, "claim")
	}
	n := 1
	for n < 40 && (claimPage(n) != 0 || claimPage(n+1) != 1) {
		n++
	}
	if n == 40 {
		t.Fatal("no place where one more paragraph moves the reference")
	}
	w, v, shown := mdPagesOf(t, fnDoc(n), fyne.NewSize(1200, 900))
	if p := pagesWith(*v.d, "claim"); p != 0 || pagesWith(*v.d, "The note") != 0 {
		t.Fatalf("before: the claim on page %d, its note on %d", p+1, pagesWith(*v.d, "The note")+1)
	}
	painted(t, v, 0)
	painted(t, v, 1) // (the pages in sight, and a screen's height round them)
	// a paragraph typed before the reference
	text := fnDoc(n + 1)
	w.textEditor.SetText(text)
	select {
	case v = <-shown:
	case <-time.After(30 * time.Second):
		t.Fatal("the pages did not follow the edit")
	}
	if p := pagesWith(*v.d, "claim"); p != 1 || pagesWith(*v.d, "The note") != 1 {
		t.Errorf("after: the claim on page %d, its note on %d", p+1, pagesWith(*v.d, "The note")+1)
	}
	_, fresh, err := markdownPDF(text, markdownTitle(text, w.scriptName()), w.markdownPageSetup(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(fresh.Pages) != len(v.d.Pages) {
		t.Fatalf("%d pages shown, %d made whole", len(v.d.Pages), len(fresh.Pages))
	}
	r := pdf.NewRasterizer(&fresh)
	for i, p := range fresh.Pages {
		if !p.Equal(v.d.Pages[i]) || !slices.Equal(p.Links, v.d.Pages[i].Links) {
			t.Errorf("page %d is not the document's as edited", i+1)
		}
		img := v.image(i)
		if i < 2 {
			img = painted(t, v, i)
		}
		if img == nil {
			continue // out of sight: painted when it comes in sight
		}
		if want := r.Paint(p, float64(img.Bounds().Dx())/p.Width); !samePaint(img, want) {
			t.Errorf("page %d's painting is not the page as edited", i+1)
		}
	}
	if !slices.Equal(v.links, drawingLinks(&fresh)) {
		t.Error("the view's links are not the new pages'")
	}
}

// Page Setup's notes: where they print and how they are numbered, kept,
// on the setup's line, and in the pages.
func TestMarkdownPageSetupNotes(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.markdown = true
	t.Cleanup(func() {
		w.settings.SetString(mdNotesPref, "")
		w.settings.SetBoolean(mdNotesPerChapterPref, false)
	})
	if s := w.markdownPageSetup(); s.notesAt != prose.NotesAtPageFoot || s.notesPerChapter {
		t.Errorf("the default %+v", s)
	}
	p := w.newMarkdownPageSetup(nil)
	if p.notes.Selected != mdNotesLabels[0] || p.numbering.Selected != mdNumberings[0] {
		t.Errorf("the dialog's defaults %q %q", p.notes.Selected, p.numbering.Selected)
	}
	p.notes.SetSelected("After each chapter")
	p.numbering.SetSelected("Again in each chapter")
	p.d.Show()
	capture(t, w.fyneWindow.Canvas(), "pagesetup-markdown-notes.png")
	p.ok.OnTapped()
	got := w.markdownPageSetup()
	if got.notesAt != prose.NotesAtChapterEnd || !got.notesPerChapter {
		t.Errorf("saved %+v", got)
	}
	if s := w.markdownPageSummary(); !strings.Contains(s, "notes after each chapter · notes numbered by chapter") {
		t.Errorf("the line %q", s)
	}
	doc := markdownDoc("# Book\n\n## One\n\nA.[^1]\n\n## Two\n\nB.[^2]\n\n[^1]: x\n[^2]: y\n", "Book", got, nil)
	if doc.NotesAt != prose.NotesAtChapterEnd || !doc.NotesPerChapter {
		t.Errorf("the document's %v %v", doc.NotesAt, doc.NotesPerChapter)
	}
	q := w.newMarkdownPageSetup(nil)
	if q.notes.Selected != "After each chapter" || q.numbering.Selected != "Again in each chapter" {
		t.Errorf("the dialog again %q %q", q.notes.Selected, q.numbering.Selected)
	}
}

// The PDF's links as another PDF reader sees them (qpdf's JSON, if qpdf
// is there): the addresses, the internal links' pages and places, the
// outline's titles and pages.
func TestPDFLinksInAReader(t *testing.T) {
	qpdf, err := exec.LookPath("qpdf")
	if err != nil {
		t.Skip("no qpdf")
	}
	data, d, err := markdownPDF(mdLinksDoc, "The Keeper", mdPageSetup{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "links.pdf")
	os.WriteFile(file, data, 0o644)
	out, err := exec.Command(qpdf, "--json=2", "--json-key=pages", "--json-key=outlines", "--json-key=qpdf", file).Output()
	if err != nil {
		t.Fatal(err)
	}
	var j struct {
		Pages []struct {
			Object string `json:"object"`
		} `json:"pages"`
		Outlines []outline         `json:"outlines"`
		QPDF     []json.RawMessage `json:"qpdf"`
	}
	if err := json.Unmarshal(out, &j); err != nil || len(j.QPDF) < 2 {
		t.Fatalf("qpdf's JSON: %v", err)
	}
	var objects map[string]struct {
		Value struct {
			Annots []struct {
				A *struct {
					URI string `json:"/URI"`
				} `json:"/A"`
				Dest []any `json:"/Dest"`
			} `json:"/Annots"`
		} `json:"value"`
	}
	if err := json.Unmarshal(j.QPDF[1], &objects); err != nil {
		t.Fatal(err)
	}
	pageOf := map[string]int{}
	for i, p := range j.Pages {
		pageOf[p.Object] = i + 1
	}
	var got, want []string
	for i, p := range j.Pages {
		for _, a := range objects["obj:"+p.Object].Value.Annots {
			if a.A != nil {
				got = append(got, fmt.Sprintf("%d %s", i+1, strings.TrimPrefix(a.A.URI, "u:")))
			} else if len(a.Dest) > 3 {
				got = append(got, fmt.Sprintf("%d to page %d at %.0f", i+1, pageOf[fmt.Sprint(a.Dest[0])], a.Dest[3]))
			}
		}
	}
	for i, p := range d.Pages {
		for _, l := range p.Links {
			if l.URL != "" {
				want = append(want, fmt.Sprintf("%d %s", i+1, l.URL))
			} else {
				want = append(want, fmt.Sprintf("%d to page %d at %.0f", i+1, l.Page, p.Height-l.DestY))
			}
		}
	}
	var titles func(os []outline, indent string)
	titles = func(os []outline, indent string) {
		for _, o := range os {
			got = append(got, fmt.Sprintf("%s%s, page %d", indent, o.Title, o.Page))
			titles(o.Kids, indent+"  ")
		}
	}
	titles(j.Outlines, "")
	want = append(want, "The Keeper, page 1", "  One, page 1", "  Two, page 2")
	if !slices.Equal(got, want) {
		t.Errorf("qpdf sees\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if dir := os.Getenv("SHOT_DIR"); dir != "" {
		os.WriteFile(filepath.Join(dir, "links.pdf"), data, 0o644)
		os.WriteFile(filepath.Join(dir, "links-qpdf.txt"), []byte(strings.Join(got, "\n")+"\n"), 0o644)
	}
}

// outline is an entry of qpdf's outline.
type outline struct {
	Title string    `json:"title"`
	Page  int       `json:"destpageposfrom1"`
	Kids  []outline `json:"kids"`
}

// Screenshots (SHOT_DIR): the Live Preview's pages with a link under the
// pointer (lit, its address in a tip), then the foot of the page with
// its note, the note's number ringed by Tab; on the desktop and a phone.
func TestPageLinksScreenshots(t *testing.T) {
	if testing.Short() || os.Getenv("SHOT_DIR") == "" {
		t.Skip("SHOT_DIR not set")
	}
	doc := strings.Replace(mdLinksDoc, strings.Repeat("The keeper climbed the stairs again, as he did every evening, and counted the steps one by one.\n\n", 30),
		strings.Repeat("The keeper climbed the stairs again, as he did every evening, and counted the steps one by one.\n\n", 8)+
			"The lamp was lit at dusk.[^2]\n\n"+strings.Repeat("The keeper climbed the stairs again, as he did every evening, and counted the steps one by one.\n\n", 22), 1) +
		"[^2]: It burned paraffin until 1924, when the lighthouse board sent the electric lamp, which the keeper distrusted for a year.\n"
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"desktop", fyne.NewSize(1200, 800)}, {"phone", fyne.NewSize(380, 800)}} {
		w, v, _ := mdPagesOf(t, doc, size.size, func(w *MainWindow) { ribbonOnFor(t, w) })
		painted(t, v, 0)
		harbour := linkIndex(v, "https://example.org/harbour", 0)
		v.area.MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: linkPoint(v, harbour)}})
		capture(t, w.fyneWindow.Canvas(), "pagelinks-hover-"+size.name+".png")
		v.hover(-1)
		// the foot of page 1: its notes; the second note's number ringed
		w.fyneWindow.Canvas().Focus(v.area)
		for v.area.AcceptsTab() {
			v.area.TypedKey(&fyne.KeyEvent{Name: fyne.KeyTab})
			if l := v.links[v.focusLink]; l.URL == "" && l.page == 0 && l.Y > v.d.Pages[0].Height/2 && l.DestY < v.d.Pages[0].Height/2 && l.DestY > 200 {
				break // the second note's number, back to its reference
			}
		}
		tops, _ := v.pageTops()
		foot := tops[0] + float32(v.d.Pages[0].Height)*v.scale - v.scroll.Size().Height + 2*pageGap
		v.scroll.Offset.Y = max(foot, 0)
		v.scroll.Refresh()
		v.showInSight()
		v.markLinks()
		painted(t, v, 0)
		capture(t, w.fyneWindow.Canvas(), "pagelinks-footnote-"+size.name+".png")
	}
}
