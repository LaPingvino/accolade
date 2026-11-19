package prose

import (
	"archive/zip"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// story is a short text of our own with what a story draft has.
const story = `# The Lighthouse Keeper

## One

The wind came in from the *west*, and the rain followed it **hard and grey**.
Nobody went out that night.[^1]

He kept three things:

- a brass compass
- a letter from his *sister*
  1. read once
  2. read twice
- a key

> The sea gives back what it takes,
> said his mother, *always*.

## Two

1. Climb the stairs.
2. Light the lamp.

See [the harbour](https://example.org/harbour) at ***dawn***.

[^1]: Not even the fishermen.
`

func TestDOCXRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := FromMarkdown(story).WriteDOCX(&buf); err != nil {
		t.Fatal(err)
	}
	d, err := ReadDOCX(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	checkRoundTrip(t, d.Markdown())
	if d.Title != "The Lighthouse Keeper" {
		t.Errorf("title %q", d.Title)
	}
}

func TestODTRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := FromMarkdown(story).WriteODT(&buf); err != nil {
		t.Fatal(err)
	}
	d, err := ReadODT(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	checkRoundTrip(t, d.Markdown())
}

// checkRoundTrip: the story came back with its headings, emphasis, lists,
// quote, link and footnote.
func checkRoundTrip(t *testing.T, md string) {
	t.Helper()
	for _, want := range []string{
		"# The Lighthouse Keeper\n\n## One\n\n",
		"The wind came in from the *west*, and the rain followed it **hard and grey**. Nobody went out that night.[^1]",
		"He kept three things:\n\n- a brass compass\n- a letter from his *sister*\n  1. read once\n  2. read twice\n- a key\n",
		"> The sea gives back what it takes, said his mother, *always*.",
		"## Two\n\n1. Climb the stairs.\n2. Light the lamp.\n",
		"See [the harbour](https://example.org/harbour) at ***dawn***.",
		"[^1]: Not even the fishermen.",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q in:\n%s", want, md)
		}
	}
}

func TestMarkdownEscapes(t *testing.T) {
	d := &Doc{Blocks: []*Block{
		{Kind: Para, Spans: []Span{{Text: "# not a heading, 2*3 = 6 and [this] "}, {Text: "bold ", Bold: true}, {Text: "end"}}},
		{Kind: Para, Spans: []Span{{Text: "1. not a list"}}},
		{Kind: Para, Spans: []Span{{Text: "a"}, {Text: "b", Italic: true}, {Text: "c", Italic: true, Bold: true}}},
	}}
	want := "\\# not a heading, 2\\*3 = 6 and \\[this\\] **bold** end\n\n1\\. not a list\n\na*b****c***\n"
	if got := d.Markdown(); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestPlain(t *testing.T) {
	p := FromMarkdown(story).Plain()
	for _, want := range []string{"The Lighthouse Keeper\n\nOne\n\n", "- a brass compass\n", "    The sea gives back", "[1] Not even the fishermen."} {
		if !strings.Contains(p, want) {
			t.Errorf("missing %q in:\n%s", want, p)
		}
	}
	if strings.ContainsAny(p, "*#") {
		t.Errorf("markup left in:\n%s", p)
	}
}

// Documents made by another program (pandoc, from our own text): its
// styles, numbering and footnotes as Word and LibreOffice write them.
func TestPandocFixtures(t *testing.T) {
	for _, name := range []string{"pandoc.docx", "pandoc.odt"} {
		b, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		var d *Doc
		if strings.HasSuffix(name, ".docx") {
			d, err = ReadDOCX(bytes.NewReader(b), int64(len(b)))
		} else {
			d, err = ReadODT(bytes.NewReader(b), int64(len(b)))
		}
		if err != nil {
			t.Fatal(name, err)
		}
		md := d.Markdown()
		for _, want := range []string{"# The Lighthouse Keeper", "## One", "*west*", "**hard and grey**", "- a brass compass", "1. Climb the stairs.", "> The sea gives back", "[^1]: Not even the fishermen."} {
			if !strings.Contains(md, want) {
				t.Errorf("%s: missing %q in:\n%s", name, want, md)
			}
		}
	}
}

// pandoc reads what we write as we meant it (when pandoc is there).
func TestPandocReadsOurs(t *testing.T) {
	pandoc, err := exec.LookPath("pandoc")
	if err != nil {
		t.Skip("no pandoc")
	}
	for _, f := range []string{"docx", "odt"} {
		var buf bytes.Buffer
		d := FromMarkdown(story)
		if f == "docx" {
			err = d.WriteDOCX(&buf)
		} else {
			err = d.WriteODT(&buf)
		}
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "story."+f)
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := exec.Command(pandoc, "-f", f, "-t", "gfm", "--wrap=none", path).CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v: %s", f, err, out)
		}
		md := string(out)
		for _, want := range []string{"# The Lighthouse Keeper", "## One", "*west*", "**hard and grey**", "- a brass compass", "1.  Climb the stairs.", "> The sea gives back", "[^1]", "Not even the fishermen."} {
			if !strings.Contains(strings.ReplaceAll(md, "1. Climb", "1.  Climb"), want) {
				t.Errorf("%s: pandoc read no %q in:\n%s", f, want, md)
			}
		}
	}
}

// zipOf is a zip of named parts.
func zipOf(t *testing.T, parts map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for name, body := range parts {
		f, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		f.Write([]byte(body))
	}
	z.Close()
	return buf.Bytes()
}

// What Word writes that we do not: a heading style in another language
// (found by its outline level), bold switched off, tracked changes, a
// content control, an image.
func TestDOCXWordsOwn(t *testing.T) {
	w := `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"`
	b := zipOf(t, map[string]string{
		"word/styles.xml": `<w:styles ` + w + `><w:style w:type="paragraph" w:styleId="berschrift1"><w:name w:val="Kopfzeile Eins"/>` +
			`<w:pPr><w:outlineLvl w:val="0"/></w:pPr><w:rPr><w:b/></w:rPr></w:style>` +
			`<w:style w:type="character" w:styleId="Fett"><w:name w:val="Strong"/></w:style></w:styles>`,
		"word/document.xml": `<w:document ` + w + `><w:body>` +
			`<w:p><w:pPr><w:pStyle w:val="berschrift1"/></w:pPr><w:r><w:t>Kapitel</w:t></w:r></w:p>` +
			`<w:p><w:r><w:rPr><w:b/></w:rPr><w:t xml:space="preserve">bold </w:t></w:r><w:r><w:rPr><w:b w:val="0"/></w:rPr><w:t>plain</w:t></w:r>` +
			`<w:del><w:r><w:delText>gone</w:delText></w:r></w:del><w:ins><w:r><w:t xml:space="preserve"> added</w:t></w:r></w:ins>` +
			`<w:r><w:rPr><w:rStyle w:val="Fett"/></w:rPr><w:t xml:space="preserve"> strong</w:t></w:r></w:p>` +
			`<w:sdt><w:sdtContent><w:p><w:r><w:t>In a control.</w:t></w:r><w:r><w:drawing><wp:inline><wp:docPr id="1" name="Bild 1" descr="a map"/></wp:inline></w:drawing></w:r></w:p></w:sdtContent></w:sdt>` +
			`</w:body></w:document>`,
	})
	d, err := ReadDOCX(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	want := "# Kapitel\n\n**bold** plain added **strong**\n\nIn a control.<!-- image left out: a map -->\n"
	if got := d.Markdown(); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	if d.Dropped != 1 {
		t.Errorf("%d dropped, want 1", d.Dropped)
	}
}

func TestReadHTML(t *testing.T) {
	page := `<!doctype html><html><head><title>The Keeper</title><style>p{}</style><script>var x = "<p>no</p>";</script></head>
<body><nav><a href="/">Home</a></nav>
<article>
  <h1>The Lighthouse Keeper</h1>
  <p>The wind came in from the <em>west</em>, and the rain followed it
     <strong>hard and grey</strong>.<br>Nobody went out.</p>
  <ul><li>a brass compass</li><li>a letter
    <ol><li>read once</li><li>read twice</li></ol></li></ul>
  <blockquote><p>The sea gives back what it takes.</p></blockquote>
  <p>See <a href="https://example.org/harbour">the harbour</a> and <img src="x.png" alt="a map">.</p>
  <pre><code>keep *this*
  as it is</code></pre>
  <table><tr><th>Day</th><th>Ships</th></tr><tr><td>Monday</td><td>3</td></tr></table>
  Loose text at the end.
</article></body></html>`
	d, err := ReadHTML(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	want := "# The Lighthouse Keeper\n\n" +
		"The wind came in from the *west*, and the rain followed it **hard and grey**.\\\nNobody went out.\n\n" +
		"- a brass compass\n- a letter\n  1. read once\n  2. read twice\n\n" +
		"> The sea gives back what it takes.\n\n" +
		"See [the harbour](https://example.org/harbour) and <!-- image left out: a map -->.\n\n" +
		"```\nkeep *this*\n  as it is\n```\n\n" +
		"| Day | Ships |\n| --- | --- |\n| Monday | 3 |\n\n" +
		"Loose text at the end.\n"
	if got := d.Markdown(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if d.Title != "The Keeper" || d.Dropped != 1 {
		t.Errorf("title %q, %d dropped", d.Title, d.Dropped)
	}
}

// A list item with a second paragraph stays one item, and a numbered list
// after it goes on counting; a page in Latin-1 is read as Latin-1.
func TestListsGoOn(t *testing.T) {
	src := "1. First.\n\n   More of the first.\n\n2. Second.\n"
	var buf bytes.Buffer
	if err := FromMarkdown(src).WriteDOCX(&buf); err != nil {
		t.Fatal(err)
	}
	d, err := ReadDOCX(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if got := d.Markdown(); got != src {
		t.Errorf("got %q, want %q", got, src)
	}
	page := "<html><head><meta charset=\"iso-8859-1\"></head><body><p>Caf\xe9</p></body></html>"
	h, err := ReadHTML(strings.NewReader(page))
	if err != nil || h.Markdown() != "Café\n" {
		t.Errorf("%q %v", h.Markdown(), err)
	}
}
