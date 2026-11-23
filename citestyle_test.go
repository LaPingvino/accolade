package main

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"github.com/LaPingvino/lexington/pdfin"

	"github.com/LaPingvino/accolade/internal/cite"
)

// citeStyleText cites in each of Pandoc's forms, lists one work with
// nocite and names one the library lacks.
const citeStyleText = `---
nocite: |
  @goethe08
---

# On Incompleteness

As Gödel showed [@goedel31, pp. 173-175], some truths cannot be proved.
The point is old [see @goedel31; also @knuth84, ch. 1], and @knuth84 [p. 2] set
it in type. A year alone: [-@goedel31]. Not in the library: [@nobody].
`

// citeStyleWant is what each rendering holds of it.
var citeStyleWant = []string{
	"(Gödel 1931, 173–175)", "(see Gödel 1931; also Knuth 1984, ch. 1)", "Knuth (1984, 2)", "(1931)", "(nobody?)",
	"References", "Gödel, Kurt. 1931.", "Monatshefte für Mathematik and Physik 38: 173–198.", "Goethe, Johann Wolfgang von. 1808. Faust. Cotta.", "Knuth, Donald E. 1984. The TeXbook.",
}

// noSpaces is s without its spaces and line breaks (a PDF's text is
// read back in runs and lines).
func noSpaces(s string) string {
	return strings.NewReplacer(" ", "", "\n", "", "\u00a0", "").Replace(s)
}

func hasAll(t *testing.T, what, got string, want []string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(noSpaces(got), noSpaces(w)) {
			t.Errorf("%s has no %q:\n%s", what, w, got)
		}
	}
	for _, bad := range []string{"[@", "@goedel31", cite.EntryMark} {
		if strings.Contains(got, bad) {
			t.Errorf("%s still has %q", what, bad)
		}
	}
}

// Accolade's own renderings cite in its author-date style: the Live
// Preview's text, its PDF (the page view's), Word, ODT, the web page.
func TestCiteStyleRenderings(t *testing.T) {
	w, dir := citeWindow(t, citeStyleText+"|")
	lib := w.library() // read at once in the tests
	if lib == nil {
		t.Fatal("no library")
	}
	page := w.markdownPageSetup()
	if page.lib != lib {
		t.Fatal("the page setup has not got the library")
	}
	// the Live Preview's text
	w.preview()
	var b strings.Builder
	for _, s := range w.previewPane.text.Segments {
		b.WriteString(s.Textual() + "\n")
	}
	hasAll(t, "the Live Preview", b.String(), citeStyleWant)

	// the PDF, read back
	out := filepath.Join(dir, "essay.pdf")
	if err := exportMarkdownWith(citeStyleText, "Essay", out, "pdf", page, nil); err != nil {
		t.Fatal(err)
	}
	pages, err := pdfin.Pages(out)
	if err != nil {
		t.Fatal(err)
	}
	var pb strings.Builder
	for _, p := range pages {
		for _, l := range p {
			pb.WriteString(l.Text + "\n")
		}
	}
	hasAll(t, "the PDF", pb.String(), citeStyleWant)

	// the web page: entries with a hanging indent
	html, err := markdownHTMLWith(citeStyleText, "Essay", page, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(html), `<p class="csl-entry">`); n != 3 {
		t.Errorf("the web page has %d entries, want 3", n)
	}
	hasAll(t, "the web page", string(html), []string{"(Gödel 1931, 173–175)", "<em>Faust</em>"})

	// Word and ODT, with the library read as the export dialog does
	// (the window's not read yet); read back by pandoc if it is here
	for _, id := range []string{"docx", "odt"} {
		p := page
		p.lib = libraryNow(citeStyleText, w.citeInputsFor(citeStyleText))
		f := filepath.Join(dir, "essay."+id)
		if err := exportMarkdownWith(citeStyleText, "Essay", f, id, p, nil); err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(f)
		x := zipText(t, data, map[string]string{"docx": "word/document.xml", "odt": "content.xml"}[id])
		if !strings.Contains(x, "(Gödel 1931, 173–175)") || !strings.Contains(x, map[string]string{"docx": `w:hanging="720"`, "odt": "P_hanging"}[id]) {
			t.Errorf("%s: no citation or no hanging indent", id)
		}
		if _, err := exec.LookPath("pandoc"); err != nil {
			continue
		}
		plain, err := exec.Command("pandoc", "-t", "plain", "--wrap=none", f).Output()
		if err != nil {
			t.Fatalf("pandoc reading %s: %v", id, err)
		}
		hasAll(t, id+" (read by pandoc)", string(plain), citeStyleWant)
	}

	// without a library: as written
	if got := citedBody("Text [@goedel31].", nil); got != "Text [@goedel31]." {
		t.Errorf("no library: %q", got)
	}
}

// zipText is a file's text in a zip.
func zipText(t *testing.T, data []byte, name string) string {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range z.File {
		if f.Name == name {
			r, _ := f.Open()
			b, _ := io.ReadAll(r)
			r.Close()
			return string(b)
		}
	}
	t.Fatalf("no %s", name)
	return ""
}

// The library read after the preview was shown (as the app reads it, in
// the background): the preview cites from it then.
func TestCiteStyleLibraryReadLater(t *testing.T) {
	holdUI(t) // the test is the UI goroutine: the library's reading comes back in its turn
	w, _ := citeWindow(t, citeStyleText+"|")
	old := backgroundAtOnce
	backgroundAtOnce = false
	t.Cleanup(func() { backgroundAtOnce = old })
	if !w.previewVisible { // preview() toggles: a window may open with it shown
		w.preview()
	} else {
		w.updateLivePreview()
	}
	if !w.previewVisible {
		t.Fatal("no preview")
	}
	shown := func() string {
		var b strings.Builder
		for _, s := range w.previewPane.text.Segments {
			b.WriteString(s.Textual())
		}
		return b.String()
	}
	if strings.Contains(shown(), "(Gödel 1931") {
		t.Skip("the library was read before the preview") // nothing to wait for
	}
	waitFor(t, "the preview citing once the library is read", func() bool {
		return strings.Contains(shown(), "(Gödel 1931, 173–175)")
	})
}

// TestCiteStyleScreenshots renders the Live Preview's text and pages with
// citations and the References, desktop and phone (SHOT_DIR).
func TestCiteStyleScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(dir, 0o755)
	for _, c := range []struct {
		name string
		size fyne.Size
	}{{"desktop", fyne.NewSize(1200, 800)}, {"phone", fyne.NewSize(380, 800)}} {
		w, _ := newDialogTestWindow(t)
		docs := t.TempDir()
		os.WriteFile(filepath.Join(docs, "refs.bib"), []byte(citeBib), 0o644)
		w.setMarkdown(true)
		w.currentFile = filepath.Join(docs, "essay.md")
		w.textEditor.SetText(citeStyleText)
		w.fyneWindow.Resize(c.size)
		if w.library() == nil {
			t.Fatal("no library")
		}
		w.preview()
		if c.name == "phone" { // the preview in the script's place
			w.showMiddle("preview")
		}
		w.fyneWindow.Content().Refresh()
		capture(t, w.fyneWindow.Canvas(), "citestyle-text-"+c.name+".png")
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
		painted(t, v, 0)
		time.Sleep(100 * time.Millisecond)
		fyne.DoAndWait(func() {})
		capture(t, w.fyneWindow.Canvas(), "citestyle-pages-"+c.name+".png")
		pagesShownHook.Store(old)
		w.previewPane.mode.SetSelected(previewText)
		w.settings.SetBoolean(previewPagesPref, false)
	}
}
