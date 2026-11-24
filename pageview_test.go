package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/pdf"
	"github.com/LaPingvino/lexington/pdfin"
	lexreport "github.com/LaPingvino/lexington/report"
)

// onPagesShown sets the pagesShown hook to f for this test.
func onPagesShown(t *testing.T, f func(*pageView)) {
	t.Helper()
	old := pagesShownHook.Swap(&f)
	t.Cleanup(func() { pagesShownHook.Store(old) })
}

// previewAndWait runs Preview PDF and waits for its pages.
func previewAndWait(t *testing.T, w *MainWindow) *pageView {
	t.Helper()
	shown := make(chan *pageView, 4)
	onPagesShown(t, func(v *pageView) { shown <- v })
	w.previewPDF()
	return awaitPages(t, shown)
}

// awaitPages waits for the pages a background Preview shows, giving up
// the UI turn meanwhile when the test holds it (holdUI): the preview's
// callback needs it to show them.
func awaitPages(t *testing.T, shown chan *pageView) *pageView {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for {
		select {
		case v := <-shown:
			return v
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("no pages shown")
		}
		uiSleep(5 * time.Millisecond)
	}
}

// painted waits until page i of the view is painted.
func painted(t *testing.T, v *pageView, i int) image.Image {
	t.Helper()
	for try := 0; try < 400; try++ {
		var img image.Image
		fyne.DoAndWait(func() { img = v.image(i) })
		if img != nil {
			return img
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("page %d never painted", i+1)
	return nil
}

// firstText is the first run of text drawn on a page, top to bottom.
func firstText(p pdf.DrawnPage) string {
	best, y := "", 1e9
	for _, m := range p.Marks {
		if t, ok := m.(pdf.Text); ok && strings.TrimSpace(t.Text) != "" && t.Y < y {
			best, y = t.Text, t.Y
		}
	}
	return best
}

const pageViewScript = "Title: The Barn\nAuthor: Someone\n\nINT. BARN - DAY\n\nRain falls on the roof, steady, for a long while.\n\nANNA\nHello there. Is anyone in?\n\nEXT. FIELD - NIGHT\n\nThe moon.\n"

const cjkPagesScript = "Title: パリ症候群\nAuthor: Someone\n\nINT. CAFÉ - DAY\n\nRain on the glass. 東京 is far.\n\nYUKI\nパリは遠い。\n"

// A title in Japanese, which Courier Badi has none of, shows in the page
// view: its mark on the title page paints dark pixels in its box (the
// glyphs from a font on the computer, as the PDF prints them).
func TestPageViewDrawsCJK(t *testing.T) {
	if m, _ := pdf.Unsupported(fountain.Parse(nil, strings.NewReader(cjkPagesScript))); len(m) > 0 {
		t.Skipf("no font on this computer has %s", string(m))
	}
	w := newTestWindow(t, cjkPagesScript)
	w.fyneWindow.Resize(fyne.NewSize(1100, 760))
	v := previewAndWait(t, w)
	var title pdf.Text
	found := false
	for _, m := range v.d.Pages[0].Marks {
		if tx, ok := m.(pdf.Text); ok && strings.Contains(tx.Text, "パリ症候群") {
			title, found = tx, true
		}
	}
	if !found {
		t.Fatalf("no mark of the title on the title page: %q", firstText(v.d.Pages[0]))
	}
	if title.Width < 9*7.2 {
		t.Errorf("the title is %.1fpt wide: not two cells a character", title.Width)
	}
	img := painted(t, v, 0)
	scale := float64(img.Bounds().Dx()) / v.d.Pages[0].Width
	ink := 0
	for y := int((title.Y - 11) * scale); y < int((title.Y+2)*scale); y++ {
		for x := int(title.X * scale); x < int((title.X+title.Width)*scale); x++ {
			if r, _, _, _ := img.At(x, y).RGBA(); r < 0x8000 {
				ink++
			}
		}
	}
	if ink < 20 {
		t.Errorf("the title paints %d dark pixels in its box: not drawn", ink)
	}
	if v.note.Visible() {
		t.Errorf("a note over pages the PDF prints as written: %q", v.note.Text)
	}
}

// A script with letters the PDF cannot shape (Syriac) shows Lexington's
// pages with a note naming the script, and the ways to the browser's
// shaping: Open as HTML, and with a browser on the computer, Make the PDF
// with the Browser.
func TestPageViewShapingNote(t *testing.T) {
	w := newTestWindow(t, "INT. ROOM - DAY\n\nShe writes: ܫܠܡܐ ܥܠܡܐ\n")
	w.fyneWindow.Resize(fyne.NewSize(1100, 760))
	v := previewAndWait(t, w)
	var text string
	var labels []string
	fyne.DoAndWait(func() {
		if v.note.Visible() {
			text = v.note.Text
		}
		if v.noteActs.Visible() {
			for _, o := range v.noteActs.Objects {
				labels = append(labels, o.(*widget.Button).Text)
			}
		}
	})
	if !strings.Contains(text, "Shaping for Syriac is missing in the PDF") {
		t.Errorf("the note: %q", text)
	}
	want := []string{"Open as HTML"}
	if localBrowser() != "" {
		want = append([]string{"Make the PDF with the Browser"}, want...)
	}
	if !slices.Equal(labels, want) {
		t.Errorf("the note's actions %q, want %q", labels, want)
	}
	if dir := os.Getenv("SHOT_DIR"); dir != "" {
		painted(t, v, 0)
		capture(t, w.fyneWindow.Canvas(), "pageview-shaping-note.png")
	}
}

// The page view shows the PDF's pages: as many as the PDF has, the first
// lines the PDF's, the pages painted in the window.
func TestPageViewMatchesThePDF(t *testing.T) {
	w := newTestWindow(t, pageViewScript)
	w.fyneWindow.Resize(fyne.NewSize(1100, 760))
	v := previewAndWait(t, w)
	ps, err := pdfin.Pages(v.pdfPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != len(v.d.Pages) || len(ps) != 2 {
		t.Fatalf("the PDF has %d pages, the page view %d (want 2)", len(ps), len(v.d.Pages))
	}
	for i, p := range ps {
		got := firstText(v.d.Pages[i])
		if len(p) == 0 || !strings.Contains(p[0].Text, strings.TrimSpace(got)) {
			t.Errorf("page %d: the view's first line %q, the PDF's %+v", i+1, got, p[:min(1, len(p))])
		}
	}
	if !strings.Contains(v.count.Text, "of 2") {
		t.Errorf("the count: %q", v.count.Text)
	}
	img := painted(t, v, 1)
	if img.Bounds().Dx() < 100 {
		t.Errorf("page 2 painted %v", img.Bounds())
	}
	if !w.pagesVisible || w.pages != v {
		t.Error("the page view is not in the window")
	}
	v.closeB.OnTapped()
	if w.pagesVisible {
		t.Error("Close left the page view")
	}
}

// The keys move through the pages and zoom; the zoom buttons too.
func TestPageViewKeys(t *testing.T) {
	text := pageViewScript
	for i := 0; i < 6; i++ {
		text += "\nINT. ROOM - DAY\n\nA long paragraph of action that goes on, and on, over the line and the next one too.\n\nBOB\nWords, many words, said slowly over the lines of the page.\n"
	}
	w := newTestWindow(t, text)
	w.fyneWindow.Resize(fyne.NewSize(1100, 700))
	v := previewAndWait(t, w)
	a := v.area
	fit := v.scale
	a.TypedKey(&fyne.KeyEvent{Name: fyne.KeyPageDown})
	if v.scroll.Offset.Y <= 0 {
		t.Error("Page Down did not scroll")
	}
	a.TypedKey(&fyne.KeyEvent{Name: fyne.KeyHome})
	a.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	tops, _ := v.pageTops()
	if got := v.scroll.Offset.Y; got < tops[1]-pageGap-1 || got > tops[1] {
		t.Errorf("Right: at %v, page 2 at %v", got, tops[1])
	}
	a.TypedRune('1')
	if v.fit || v.scale != pointUnits {
		t.Errorf("1: scale %v, want %v", v.scale, pointUnits)
	}
	a.TypedRune('+')
	if v.scale <= pointUnits {
		t.Error("+ did not zoom in")
	}
	a.TypedRune('0')
	if !v.fit || v.scale != fit {
		t.Errorf("0: scale %v, the fit's %v", v.scale, fit)
	}
	v.pinch(v.scale, 2)
	if v.fit || v.scale < fit*1.9 {
		t.Errorf("a pinch to twice: %v from %v", v.scale, fit)
	}
	widths := func() (px []int) {
		for _, p := range v.pages {
			px = append(px, p.px)
		}
		return px
	}
	before := widths()
	v.pinch(v.scale/2, 2.5) // still pinching: the pages painted stretch
	for i, p := range v.pages {
		if before[i] != 0 && p.px != before[i] {
			t.Errorf("page %d painted again while pinching", i+1)
		}
	}
	v.pinchEnd()
	if slices.Equal(widths(), before) {
		t.Error("not painted again when the fingers lifted")
	}
	a.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if w.pagesVisible {
		t.Error("Escape left the page view")
	}
}

// A phone shows the pages in the script's place, fitting its width.
func TestPageViewOnAPhone(t *testing.T) {
	w := newTestWindow(t, pageViewScript)
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	v := previewAndWait(t, w)
	if !v.narrow {
		t.Error("not laid out for a phone")
	}
	if w.editorView.Visible() && fyne.CurrentApp().Driver().AbsolutePositionForObject(w.editorView).X >= 0 &&
		w.editorView.Size().Width > 0 && isShown(w, w.editorView) {
		t.Error("the script is beside the pages on a phone")
	}
	if pw := float32(v.d.Pages[0].Width) * v.scale; pw > 380 {
		t.Errorf("the page is %v wide on a phone 380 wide", pw)
	}
}

// isShown tells whether o is in the window's content.
func isShown(w *MainWindow, o fyne.CanvasObject) bool {
	found := false
	var walk func(fyne.CanvasObject)
	walk = func(x fyne.CanvasObject) {
		if x == o {
			found = true
		}
		switch c := x.(type) {
		case *fyne.Container:
			for _, y := range c.Objects {
				walk(y)
			}
		case fyne.Widget:
			if r := test.WidgetRenderer(c); r != nil {
				for _, y := range r.Objects() {
					walk(y)
				}
			}
		}
	}
	walk(w.fyneWindow.Content())
	return found
}

// Everything Tab reaches over the pages shows that it has the focus, the
// pages too (a frame), on the desktop and on a phone.
func TestPageViewFocusVisible(t *testing.T) {
	if raceDetector {
		t.Skip("pixel capture of focus: checked by the runs without -race (as TestFocusIsVisible)")
	}
	for _, size := range []fyne.Size{fyne.NewSize(1100, 700), fyne.NewSize(380, 800)} {
		w, _ := newDialogTestWindow(t)
		w.textEditor.SetText(pageViewScript)
		w.fyneWindow.Resize(size)
		v := previewAndWait(t, w)
		painted(t, v, 0)
		var u []string
		fyne.DoAndWait(func() { u = invisibleFocus(w.fyneWindow.Canvas()) })
		if len(u) > 0 {
			t.Errorf("%v: the focus is not seen on\n  %s", size, strings.Join(u, "\n  "))
		}
	}
}

// The reports' PDFs (the breakdown sheets) show in the page view.
func TestReportPDFInPageView(t *testing.T) {
	w, rd := reportsWindow(t, fyne.NewSize(1100, 760))
	rd.showReportPDF(lexreport.KindSheets)
	if w.fyneWindow.Canvas().Overlays().Top() != nil {
		t.Error("the dialog still covers the pages")
	}
	if !w.pagesVisible || w.pages == nil || len(w.pages.d.Pages) == 0 {
		t.Fatal("the sheets are not shown")
	}
	ps, err := pdfin.Pages(w.pages.pdfPath)
	if err != nil || len(ps) != len(w.pages.d.Pages) {
		t.Errorf("the PDF has %d pages (%v), the view %d", len(ps), err, len(w.pages.d.Pages))
	}
}

// Screenshots of the page view (SHOT_DIR): the desktop and a phone, a
// screenplay and a Hebrew one.
func TestPageViewScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("SHOT_DIR not set")
	}
	hebrew := "Title: חנות\n\nפנים. חנות ספרים - יום\n\nדני (40) עומד ליד הקופה. He looks at the sign: \"פתוח\" and smiles.\n\n@מִרְיָם\n(בשקט)\nשָׁלוֹם, דָּנִי! מה שלומך היום?\n\n> חתך ל:\n"
	for _, c := range []struct {
		name, text string
		size       fyne.Size
	}{
		{"desktop", pageViewScript, fyne.NewSize(1200, 800)},
		{"phone", pageViewScript, fyne.NewSize(380, 800)},
		{"hebrew-desktop", hebrew, fyne.NewSize(1200, 800)},
		{"hebrew-phone", hebrew, fyne.NewSize(380, 800)},
		{"cjk-desktop", cjkPagesScript, fyne.NewSize(1200, 800)},
		{"cjk-phone", cjkPagesScript, fyne.NewSize(380, 800)},
	} {
		w, _ := newDialogTestWindow(t)
		w.textEditor.SetText(c.text)
		if strings.HasPrefix(c.name, "hebrew") {
			w.language = "he"
		}
		w.fyneWindow.Resize(c.size)
		v := previewAndWait(t, w)
		painted(t, v, 0)
		if len(v.d.Pages) > 1 && !strings.HasPrefix(c.name, "cjk") { // (CJK's: its title page)
			v.area.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
			painted(t, v, 1)
		}
		time.Sleep(100 * time.Millisecond)
		fyne.DoAndWait(func() {})
		f, err := os.Create(filepath.Join(dir, "pageview-"+c.name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, w.fyneWindow.Canvas().Capture())
		f.Close()
	}
}

const mdPagesDoc = "---\ntitle: The Lighthouse Keeper\nauthor: A. Writer\n---\n\n# The Lighthouse Keeper\n\n## One\n\n" +
	"The wind came in from the *west*, and the rain followed it **hard and grey**. Nobody went out that night.[^1]\n\n" +
	"- a brass compass\n- a letter from his *sister*\n- a key\n\n> The sea gives back what it takes,\n> said his mother.\n\n" +
	"| Room | Things |\n| --- | --- |\n| Kitchen | a kettle |\n\n```\nlight()\n```\n\n[^1]: Not even the fishermen.\n"

// A Markdown document's PDF is Accolade's own (no browser): its text in
// the PDF, Lexington's producer, the page setup's paper.
func TestMarkdownPDFNative(t *testing.T) {
	out := filepath.Join(t.TempDir(), "doc.pdf")
	if m := markdownPDFMissing(mdPagesDoc, nil); m != "" {
		t.Fatalf("missing: %s", m)
	}
	if err := exportMarkdownWith(mdPagesDoc, "The Lighthouse Keeper", out, "pdf", mdPageSetup{paper: "letter", titlePage: true},
		markdownTitlePage(mdPagesDoc, "x.md")); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "/Producer") || strings.Contains(string(data), "Skia") {
		t.Error("not Accolade's PDF")
	}
	pages, err := pdfin.Pages(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 2 || !strings.Contains(pages[0][0].Text, "The Lighthouse Keeper") || !strings.Contains(pages[1][0].Text, "The Lighthouse Keeper") {
		t.Errorf("%d pages: %+v", len(pages), pages)
	}
	if m := markdownPDFMissing("# שלום\n\nעולם\n", nil); m != "" {
		t.Errorf("Hebrew prose: %q", m)
	}
	if m := markdownPDFMissing("# 日本語\n", nil); m == "" {
		t.Error("Japanese prose prints")
	}
}

// The Live Preview's Pages show the document's PDF, and follow the text.
func TestMarkdownLivePreviewPages(t *testing.T) {
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"desktop", fyne.NewSize(1200, 800)}, {"phone", fyne.NewSize(380, 800)}} {
		w, _ := newDialogTestWindow(t)
		w.setMarkdown(true)
		w.settings.SetBoolean(mdTitlePagePref, true)
		w.textEditor.SetText(mdPagesDoc)
		w.fyneWindow.Resize(size.size)
		w.preview()
		shown := make(chan *pageView, 4)
		f := func(v *pageView) { shown <- v }
		old := pagesShownHook.Swap(&f)
		w.previewPane.mode.SetSelected(previewPages)
		var v *pageView
		select {
		case v = <-shown:
		case <-time.After(30 * time.Second):
			t.Fatal("no pages")
		}
		if len(v.d.Pages) != 2 || firstText(v.d.Pages[1]) != "The" {
			t.Errorf("%s: %d pages, the second's first line %q", size.name, len(v.d.Pages), firstText(v.d.Pages[1]))
		}
		painted(t, v, 0)
		if size.name == "desktop" {
			v.area.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
			painted(t, v, 1)
		}
		capture(t, w.fyneWindow.Canvas(), "pageview-markdown-"+size.name+".png")
		w.textEditor.SetText(mdPagesDoc + "\n## Two\n\nMore.\n")
		select {
		case v = <-shown:
		case <-time.After(30 * time.Second):
			t.Fatal("the pages did not follow the text")
		}
		pagesShownHook.Store(old)
		w.previewPane.mode.SetSelected(previewText)
		w.settings.SetBoolean(previewPagesPref, false)
	}
}

// A setting of the page changed anywhere (the ribbon's too, not only Page
// Setup's OK) makes the preview's pages again.
func TestPageViewFollowsSettings(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(pageViewScript)
	t.Cleanup(func() { w.settings.SetBoolean("include-title-page", true) })
	w.settings.SetBoolean("include-title-page", true)
	v := previewAndWait(t, w)
	with := len(v.d.Pages)
	shown := make(chan *pageView, 4)
	onPagesShown(t, func(v *pageView) { shown <- v })
	w.changeSetting("include-title-page", func(s *Settings) { s.SetBoolean("include-title-page", false) })
	select {
	case v = <-shown:
	case <-time.After(30 * time.Second):
		t.Fatal("the pages were not made again")
	}
	if len(v.d.Pages) != with-1 {
		t.Errorf("%d pages, %d with the title page", len(v.d.Pages), with)
	}
	w.changeSetting("font-size", func(s *Settings) { s.SetFloat("font-size", 17) })
	select {
	case <-shown:
		t.Error("the editor's font size made the pages again")
	case <-time.After(300 * time.Millisecond):
	}
}

// The Live Preview's Pages print Hebrew, Arabic and Hindi prose
// themselves (Accolade's PDF, no browser), with no note that they cannot.
func TestMarkdownPagesRightToLeft(t *testing.T) {
	for _, c := range []struct{ name, text string }{
		{"hebrew", "# בְּרֵאשִׁית\n\nבְּרֵאשִׁית בָּרָא אֱלֹהִים אֵת הַשָּׁמַיִם וְאֵת הָאָרֶץ. הספר **הזה** נכתב בשנת 2024 עם Accolade.\n\n1. ראשון\n2. שני\n"},
		{"arabic", "# بِسْمِ اللَّهِ\n\nبِسْمِ اللَّهِ الرَّحْمَٰنِ الرَّحِيمِ. هذا **النص** مكتوب في سنة 2024 مع Accolade.\n\n* می‌خواهم بروم\n"},
		{"hindi", "# हिन्दी में कहानी\n\nक्षत्रिय किसान ने **धर्म** और श्रद्धा से ज्ञान पाया।\n"},
	} {
		w, _ := newDialogTestWindow(t)
		w.setMarkdown(true)
		w.textEditor.SetText(c.text)
		w.fyneWindow.Resize(fyne.NewSize(1200, 800))
		w.preview()
		shown := make(chan *pageView, 4)
		f := func(v *pageView) { shown <- v }
		old := pagesShownHook.Swap(&f)
		w.previewPane.mode.SetSelected(previewPages)
		var v *pageView
		select {
		case v = <-shown:
		case <-time.After(30 * time.Second):
			t.Fatal("no pages")
		}
		if v.note.Visible() {
			t.Errorf("%s: a note: %q", c.name, v.note.Text)
		}
		painted(t, v, 0)
		capture(t, w.fyneWindow.Canvas(), "pageview-markdown-"+c.name+".png")
		pagesShownHook.Store(old)
		w.previewPane.mode.SetSelected(previewText)
		w.settings.SetBoolean(previewPagesPref, false)
	}
}
