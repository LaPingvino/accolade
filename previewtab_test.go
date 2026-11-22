package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"github.com/LaPingvino/lexington/pdf"
)

// featureScript is a made-up feature-length screenplay (some 110 pages):
// scenes of action over several lines, speeches long enough to break
// with (MORE), parentheticals, transitions, acts.
func featureScript(seed uint64, scenes int) string {
	r := rand.New(rand.NewPCG(seed, 3))
	words := strings.Fields(`the a old new dark quiet long small door window light rain city night car table glass hand
		eyes street room voice letter train morning shadow river bridge coat key phone silence moment wall floor looks
		turns waits walks opens closes holds reaches smiles listens stops runs falls watches remembers finds slowly again
		never always almost nothing everything somewhere outside inside behind across under`)
	sentence := func(n int) string {
		w := make([]string, n)
		for i := range w {
			w[i] = words[r.IntN(len(words))]
		}
		w[0] = strings.ToUpper(w[0][:1]) + w[0][1:]
		return strings.Join(w, " ") + [...]string{".", ".", "?", "!"}[r.IntN(4)]
	}
	sentences := func(n int) string {
		s := make([]string, n)
		for i := range s {
			s[i] = sentence(4 + r.IntN(9))
		}
		return strings.Join(s, " ")
	}
	names := []string{"ANNA", "BORIS", "CARLA", "DMITRI", "ELENA"}
	places := []string{"KITCHEN", "STATION", "ROOFTOP", "OFFICE", "RIVERBANK", "HALLWAY"}
	var b strings.Builder
	b.WriteString("Title: A Feature\nAuthor: Nobody\n\n")
	for sc := 1; sc <= scenes; sc++ {
		if sc%25 == 1 {
			fmt.Fprintf(&b, "# ACT %d\n\n", sc/25+1)
		}
		fmt.Fprintf(&b, "%s. %s - %s\n\n", [...]string{"INT", "EXT"}[r.IntN(2)], places[r.IntN(len(places))], [...]string{"DAY", "NIGHT"}[r.IntN(2)])
		for range 1 + r.IntN(3) {
			for range 1 + r.IntN(3) {
				b.WriteString(sentences(1+r.IntN(3)) + "\n")
			}
			b.WriteString("\n")
		}
		for range 2 + r.IntN(5) {
			b.WriteString(names[r.IntN(len(names))] + "\n")
			if r.IntN(4) == 0 {
				b.WriteString("(" + strings.ToLower(strings.TrimSuffix(sentence(2+r.IntN(3)), ".")) + ")\n")
			}
			n := 1 + r.IntN(3)
			if r.IntN(6) == 0 {
				n = 6 + r.IntN(6)
			}
			b.WriteString(sentences(n) + "\n\n")
		}
		if r.IntN(10) == 0 {
			b.WriteString("CUT TO:\n\n")
		}
	}
	return b.String()
}

// typeAt types s at the end of line n (from 1) of the script, a key at
// a time, as the writer does.
func typeAt(w *MainWindow, n int, s string) { putAt(w, n, s, false) }

// pasteAt pastes s there, at once.
func pasteAt(w *MainWindow, n int, s string) { putAt(w, n, s, true) }

func putAt(w *MainWindow, n int, s string, paste bool) {
	lines := strings.SplitAfter(w.textEditor.Text(), "\n")
	at := 0
	for _, l := range lines[:n-1] {
		at += len([]rune(l))
	}
	at += len([]rune(strings.TrimRight(lines[n-1], "\n"))) // at the line's end
	if paste {
		w.textEditor.Replace(at, at, s)
		return
	}
	for _, r := range s {
		w.textEditor.Replace(at, at, string(r))
		at++
	}
}

// wholeDrawing is the script's pages as Preview PDF makes them whole.
func wholeDrawing(t *testing.T, w *MainWindow, text string) pdf.Drawing {
	t.Helper()
	d, err := drawer(text, w.pages.follow.job(text))(nil)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// samePages fails unless the view shows want's pages.
func samePages(t *testing.T, v *pageView, want pdf.Drawing) {
	t.Helper()
	if len(v.d.Pages) != len(want.Pages) {
		t.Fatalf("%d pages, want %d", len(v.d.Pages), len(want.Pages))
	}
	for i := range want.Pages {
		if !v.d.Pages[i].Equal(want.Pages[i]) {
			t.Fatalf("page %d is not as made whole", i+1)
		}
	}
}

// featureLine is a line of dialogue near the middle of the script (a
// line of the page n of it, from 0).
func featureLine(w *MainWindow, page int) int {
	p := w.pages.d.Pages[page]
	lines := strings.Split(w.textEditor.Text(), "\n")
	for n := p.First + 2; n < len(lines); n++ {
		if l := lines[n-1]; len(l) > 40 && l != strings.ToUpper(l) {
			return n
		}
	}
	return p.First
}

// A key typed mid-script, Follow Edits on: its page is laid out again at
// once and painted, the pages after it dimmed; then the pages are made
// again from the page before to where they start as before — the pages
// shown are the whole PDF's of the text as typed, only the pages that
// changed painted again, the place and the zoom kept. The measures are
// logged (a feature's 110 pages).
func TestPreviewFollowsATypedKey(t *testing.T) {
	w := newTestWindow(t, featureScript(1, 150))
	w.fyneWindow.Resize(fyne.NewSize(1200, 800))
	v := previewAndWait(t, w)
	whole := time.Now()
	wholeDrawing(t, w, w.textEditor.Text())
	wholeTook := time.Since(whole)
	mid := len(v.d.Pages) / 2
	v.area.toPage(0)
	for range mid {
		v.area.toPage(1)
	}
	v.setZoom(1.25)
	page, frac := v.anchor()
	line := featureLine(w, mid)
	if got := v.d.PageOf(line); got != mid {
		t.Fatalf("line %d is on page %d, want %d", line, got+1, mid+1)
	}
	f := v.follow
	typeAt(w, line, " x")
	st := f.stats
	t.Logf("a feature of %d pages (made whole in %v): a key typed on page %d: %d pages laid out at once (%v to the page shown); "+
		"%d laid out again, the rest kept: %v (%v to all made again, after the pause of %v); %d painted again",
		len(v.d.Pages), wholeTook, mid+1, st.redrawn, st.shown, st.reflowed, st.kept, st.settled, pagesFollowPause, st.painted)
	if st.redrawn < 1 || st.redrawn > 2 || st.reflowed > 4 || !st.kept {
		t.Errorf("laid out %d at once, %d after, kept %v", st.redrawn, st.reflowed, st.kept)
	}
	if st.painted > 2 {
		t.Errorf("%d pages painted again", st.painted)
	}
	samePages(t, v, wholeDrawing(t, w, w.textEditor.Text()))
	if p, fr := v.anchor(); p != page || fr-frac > 0.01 || frac-fr > 0.01 || v.fit || v.zoom != 1.25 {
		t.Errorf("at page %d (%.2f) zoom %v, was page %d (%.2f) 1.25", p, fr, v.zoom, page, frac)
	}
	if v.stale || v.dimFrom != -1 || !v.pdfStale {
		t.Errorf("stale %v, dimmed from %d, the PDF stale %v", v.stale, v.dimFrom, v.pdfStale)
	}
}

// A paragraph typed in moves the page breaks after it: the pages are
// made again until they start as before, or to the end.
func TestPreviewReflowsMovedBreaks(t *testing.T) {
	w := newTestWindow(t, featureScript(2, 60))
	w.fyneWindow.Resize(fyne.NewSize(1200, 800))
	v := previewAndWait(t, w)
	before := len(v.d.Pages)
	line := featureLine(w, 10)
	pasteAt(w, line, "\n\n"+strings.Repeat("More and more action fills the page, line after line. ", 30))
	st := v.follow.stats
	t.Logf("a paragraph on page 11 of %d: %d laid out at once, %d after (kept %v), %d painted", before, st.redrawn, st.reflowed, st.kept, st.painted)
	samePages(t, v, wholeDrawing(t, w, w.textEditor.Text()))
	if st.reflowed < 2 {
		t.Errorf("%d pages laid out again", st.reflowed)
	}
}

// Typing again cancels the pages being made in the background: the
// older edit's never shown.
func TestPreviewTypingCancelsTheReflow(t *testing.T) {
	w := newTestWindow(t, featureScript(3, 60))
	w.fyneWindow.Resize(fyne.NewSize(1200, 800))
	v := previewAndWait(t, w)
	f := v.follow
	line := featureLine(w, 5)
	pages, typed := 0, false
	hook := func() {
		if pages++; pages == 3 && !typed { // a page made: the writer types again
			typed = true
			typeAt(w, line+2, "y")
		}
	}
	followPageHook.Store(&hook)
	t.Cleanup(func() { followPageHook.Store(nil) })
	typeAt(w, line, "x")
	if !typed {
		t.Fatal("never typed again")
	}
	if f.baseText != w.textEditor.Text() {
		t.Error("the pages are not of the text as typed last")
	}
	samePages(t, v, wholeDrawing(t, w, w.textEditor.Text()))
}

// Follow Edits off: an edit leaves the pages, a dot on Refresh says they
// are stale; Refresh makes them again, the place and the zoom kept.
func TestFollowEditsOffAndRefresh(t *testing.T) {
	w := newTestWindow(t, featureScript(4, 30))
	w.fyneWindow.Resize(fyne.NewSize(1200, 800))
	r := ribbonOnFor(t, w)
	v := previewAndWait(t, w)
	w.setFollowEdits(false)
	t.Cleanup(func() { w.setFollowEdits(true) })
	if v.followC.Checked {
		t.Error("the bar's Follow Edits still checked")
	}
	shown := v.d
	v.area.toPage(1)
	v.area.toPage(1)
	v.zoomBy(1)
	page, zoom := func() int { p, _ := v.anchor(); return p }(), v.zoom
	typeAt(w, featureLine(w, 2), " x")
	if v.d != shown || !v.stale {
		t.Fatalf("the pages followed (stale %v)", v.stale)
	}
	dot := false
	for _, b := range r.byID["preview.refresh"] {
		dot = dot || b.dot
	}
	if !dot {
		t.Error("no dot on Refresh")
	}
	shownCh := make(chan *pageView, 2)
	onPagesShown(t, func(v *pageView) { shownCh <- v })
	w.do(cmd("preview.refresh"))()
	select {
	case <-shownCh:
	case <-time.After(60 * time.Second):
		t.Fatal("Refresh made no pages")
	}
	if v.stale || v.d == shown {
		t.Error("still stale after Refresh")
	}
	if p, _ := v.anchor(); p != page || v.zoom != zoom || v.fit {
		t.Errorf("Refresh: at page %d zoom %v, was %d %v", p, v.zoom, page, zoom)
	}
	samePages(t, v, wholeDrawing(t, w, w.textEditor.Text()))
}

// The Preview tab comes forward when a preview opens, and goes when it
// closes (the tab before it chosen again); its buttons act on the pages;
// its key tips are its own; the bar over the pages is the Menus look's.
func TestPreviewTab(t *testing.T) {
	holdUI(t)
	w := newTestWindow(t, featureScript(5, 12))
	w.fyneWindow.Resize(fyne.NewSize(1200, 800))
	r := ribbonOnFor(t, w)
	r.selectTab("Edit")
	v := previewAndWait(t, w)
	w.fyneWindow.Canvas().Capture()
	if r.selected != "Preview" {
		t.Fatalf("the tab chosen: %q", r.selected)
	}
	if v.tools.Visible() || v.head.Visible() {
		t.Error("the bar over the pages shows in the Ribbon look")
	}
	if r.pageCount == nil || !strings.HasPrefix(r.pageCount.Text, "Page 1 of ") {
		t.Errorf("the page count: %v", r.pageCount)
	}
	press := func(id string) {
		t.Helper()
		bs := r.byID[id]
		if len(bs) == 0 {
			t.Fatalf("no %s button", id)
		}
		bs[0].press()
	}
	press("preview.nextPage")
	if !strings.HasPrefix(r.pageCount.Text, "Page 2 of ") {
		t.Errorf("Next: %q", r.pageCount.Text)
	}
	press("preview.zoomIn")
	if v.fit {
		t.Error("Zoom In left the fit")
	}
	press("preview.fit")
	if !v.fit {
		t.Error("Fit did not fit")
	}
	if b := r.byID["preview.pageSetup"]; len(b) == 0 || b[0].label == "" || !strings.Contains(b[0].name, "title page") {
		t.Errorf("Setup: %+v", b)
	}
	r.showKeyTips()
	r.assignTips(2)
	seen := map[string]string{}
	for _, b := range r.buttons {
		if b.tip == "" {
			continue
		}
		if o, ok := seen[b.tip]; ok {
			t.Errorf("key tip %s for %s and %s", b.tip, o, b.label)
		}
		seen[b.tip] = b.label
	}
	r.leaveKeyTips(false)
	press("preview.close")
	if w.pagesVisible || r.selected != "Edit" {
		t.Errorf("closed: pages %v, the tab %q", w.pagesVisible, r.selected)
	}
	for _, tb := range r.shownTabs() {
		if tb.title == "Preview" {
			t.Error("the Preview tab is there with no preview")
		}
	}
	w.settings.SetString(lookPref, "menus")
	w.applyEditorSettings()
	v = previewAndWait(t, w)
	if !v.tools.Visible() || !v.head.Visible() {
		t.Error("no bar over the pages in the Menus look")
	}
}

// A Markdown document's Live Preview has the Preview tab too: Text or
// Pages.
func TestPreviewTabMarkdown(t *testing.T) {
	holdUI(t)
	w, _ := newDialogTestWindow(t)
	w.setMarkdown(true)
	w.textEditor.SetText(mdPagesDoc)
	w.fyneWindow.Resize(fyne.NewSize(1200, 800))
	r := ribbonOnFor(t, w)
	t.Cleanup(func() { w.settings.SetBoolean(previewPagesPref, false) })
	w.preview()
	if r.selected != "Preview" || len(r.byID["preview.pages"]) == 0 {
		t.Fatalf("the tab %q, Pages %v", r.selected, r.byID["preview.pages"])
	}
	shown := make(chan *pageView, 4)
	onPagesShown(t, func(v *pageView) { shown <- v })
	r.byID["preview.pages"][0].press()
	select {
	case <-shown:
	case <-time.After(30 * time.Second):
		t.Fatal("no pages")
	}
	if !w.previewPane.showsPages() || w.previewView() == nil {
		t.Fatal("Pages did not show the pages")
	}
	if b := r.byID["preview.pages"]; len(b) == 0 || !b[0].checked {
		t.Error("Pages not checked")
	}
	r.byID["preview.close"][0].press()
	if w.previewVisible || r.selected == "Preview" {
		t.Error("Close Preview left the Live Preview")
	}
}

// Screenshots of the Preview tab (SHOT_DIR): a screenplay's pages, a key
// just typed (its page laid out again, the pages after it dimmed until
// they are made again) and after; a Markdown document's Pages; on the
// desktop and a phone.
func TestPreviewTabScreenshots(t *testing.T) {
	if testing.Short() || os.Getenv("SHOT_DIR") == "" {
		t.Skip("SHOT_DIR not set")
	}
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"desktop", fyne.NewSize(1200, 800)}, {"phone", fyne.NewSize(380, 800)}} {
		w := newTestWindow(t, featureScript(6, 12))
		w.fyneWindow.Resize(size.size)
		ribbonOnFor(t, w)
		v := previewAndWait(t, w)
		painted(t, v, 0)
		capture(t, w.fyneWindow.Canvas(), "previewtab-screenplay-"+size.name+".png")
		// Follow Edits off: typed, the dot on Refresh
		w.setFollowEdits(false)
		before := w.textEditor.Text()
		line := featureLine(w, 1)
		pasteAt(w, line, " And then the rain stops, and the street is quiet again for a long while.")
		capture(t, w.fyneWindow.Canvas(), "previewtab-stale-"+size.name+".png")
		// the page laid out at once, the pages after it dimmed (as the
		// pause before the rest is made)
		f, text := v.follow, w.textEditor.Text()
		first, last := changedLines(before, text)
		d, from, n, err := pdf.Redraw(f.base, first, last, drawer(text, f.job(text)))
		if err != nil {
			t.Fatal(err)
		}
		v.setDrawing(d, from+n)
		if size.name == "desktop" {
			v.setZoom(0.33)
		}
		v.area.toPage(1) // the page typed on, the next dimmed under it
		painted(t, v, 1)
		painted(t, v, 2)
		capture(t, w.fyneWindow.Canvas(), "previewtab-typed-"+size.name+".png")
		w.setFollowEdits(true)

		m, _ := newDialogTestWindow(t)
		m.setMarkdown(true)
		m.textEditor.SetText(mdPagesDoc)
		m.fyneWindow.Resize(size.size)
		ribbonOnFor(t, m)
		m.settings.SetBoolean(previewPagesPref, true)
		shown := make(chan *pageView, 4)
		onPagesShown(t, func(v *pageView) { shown <- v })
		m.preview()
		select {
		case v = <-shown:
		case <-time.After(30 * time.Second):
			t.Fatal("no pages")
		}
		painted(t, v, 0)
		capture(t, m.fyneWindow.Canvas(), "previewtab-markdown-"+size.name+".png")
		m.settings.SetBoolean(previewPagesPref, false)
	}
}
