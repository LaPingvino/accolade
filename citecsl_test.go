package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaPingvino/accolade/internal/cite"
)

// The References pane's style picker writes the front matter's csl:
// (making the front matter, changing the line, taking it out), the
// cursor where it was.
func TestCiteStylePicker(t *testing.T) {
	for _, c := range []struct{ text, value, want string }{
		{"# Essay\n", "ieee", "---\ncsl: ieee\n---\n\n# Essay\n"},
		{"---\ntitle: T\n---\nText\n", "apa", "---\ntitle: T\ncsl: apa\n---\nText\n"},
		{"---\ncsl: apa\ntitle: T\n---\nText\n", "vancouver", "---\ncsl: vancouver\ntitle: T\n---\nText\n"},
		{"---\ncsl: apa\ntitle: T\n---\nText\n", "", "---\ntitle: T\n---\nText\n"},
	} {
		e, ok := cslEdit(c.text, c.value)
		if got := e.apply(c.text); !ok || got != c.want {
			t.Errorf("%q with %q: %q, want %q", c.text, c.value, got, c.want)
		}
	}
	if _, ok := cslEdit("Text\n", ""); ok {
		t.Error("no csl: to take out")
	}
	w, _ := citeWindow(t, "As [@goedel31] showed.|\n")
	w.togglePane("references")
	p := w.referencesPane()
	p.refresh()
	if p.style.Selected != "Author-date (Accolade's)" {
		t.Errorf("selected %q", p.style.Selected)
	}
	p.style.SetSelected("IEEE (numbered)")
	text := w.textEditor.Text()
	if !strings.HasPrefix(text, "---\ncsl: ieee\n---\n\n") {
		t.Fatalf("text %q", text)
	}
	if got := w.textEditor.CursorOffset(); got != len([]rune(text))-1 {
		t.Errorf("cursor at %d, want %d", got, len([]rune(text))-1)
	}
	lib := libraryNow(text, w.citeInputsFor(text))
	body := citedBody(text, lib)
	if !strings.Contains(body, `As [1] showed.`) || !strings.Contains(body, `[1] K. Gödel, “Über formal unentscheidbare sätze,”`) { // BibTeX's sentence case: {S}ätze keeps its capital
		t.Errorf("in IEEE:\n%s", body)
	}
}

// A csl: file beside the document is the style (read with the library);
// Pandoc's export gets the same file, or Accolade's own for a name.
func TestCiteStyleFile(t *testing.T) {
	w, dir := citeWindow(t, "|")
	os.WriteFile(filepath.Join(dir, "mine.csl"), cite.BuiltinCSL("vancouver"), 0o644)
	text := "---\ncsl: mine.csl\n---\n\nAs [@goedel31; @knuth84] showed.\n"
	lib := libraryNow(text, w.citeInputsFor(text))
	if body := citedBody(text, lib); !strings.Contains(body, `As (1,2) showed.`) {
		t.Errorf("with mine.csl:\n%s", body)
	}
	os.WriteFile(filepath.Join(dir, "journal.csl"), []byte(`<style><info><link rel="independent-parent" href="http://www.zotero.org/styles/apa"/></info></style>`), 0o644)
	dep := strings.Replace(text, "mine.csl", "journal.csl", 1)
	if lib := libraryNow(dep, w.citeInputsFor(dep)); lib.Style != nil || !strings.Contains(lib.StyleNote, "use its parent, apa") {
		t.Errorf("a dependent style: %q", lib.StyleNote)
	}
	text = strings.Replace(text, "mine.csl", "apa", 1)
	c, err := citeExportFor(text, w.citeInputsFor(text))
	if err != nil || c.cslName != "style.csl" || string(c.csl) != string(cite.BuiltinCSL("apa")) {
		t.Errorf("export: %s %v", c.cslName, err)
	}
	lib = libraryNow(text, w.citeInputsFor(text))
	if body := citedBody(text, lib); !strings.Contains(body, `As (Gödel, 1931; Knuth, 1984) showed.`) {
		t.Errorf("in APA:\n%s", body)
	}
}
