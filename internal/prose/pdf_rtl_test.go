package prose

import (
	"math"
	"os/exec"
	"strings"
	"testing"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/LaPingvino/lexington/pdf"
)

// rtlProse is prose read right to left in each place a prose PDF prints
// text, with marks (Hebrew points, Arabic vowels, a shadda), Persian with
// its ZWNJ, and Latin words and numbers inside it.
var rtlProse = map[string]string{
	"Hebrew": "# בְּרֵאשִׁית\n\n" +
		"בְּרֵאשִׁית בָּרָא אֱלֹהִים אֵת הַשָּׁמַיִם וְאֵת הָאָרֶץ. הספר **הזה** נכתב *לאט* בשנת 2024 עם Accolade.[^1]\n\n" +
		"> ציטוט ארוך מספיק כדי לעבור לשורה הבאה בתוך הפסקה הזאת, ולהראות שהשורות מיושרות לימין כפי שצריך.\n\n" +
		"1. ראשון\n2. שני עם [קישור](https://example.org)\n\n" +
		"| שם | עבודה |\n| --- | --- |\n| דני | נגר |\n\n" +
		"[^1]: הערה בעברית.\n",
	"Arabic": "# بِسْمِ اللَّهِ\n\n" +
		"بِسْمِ اللَّهِ الرَّحْمَٰنِ الرَّحِيمِ. هذا **النص** مكتوب *ببطء* في سنة 2024 مع Accolade.[^1]\n\n" +
		"* أَعُودَ حَتَّى\n* می‌خواهم بروم\n\n" +
		"[^1]: ملاحظة بالعربية.\n",
	"Mixed": "# A story with שלום in it\n\n" +
		"The word שָׁלוֹם means peace, and سلام too; עולם is world. A longer line wraps where the Hebrew words בְּרֵאשִׁית בָּרָא אֱלֹהִים stand in the middle of the English text.\n\n" +
		"שלום world, 12 עולם.\n",
}

// rtlReadBack are what pdftotext must give back as written, compared as
// readable makes them: lines read right to left without Latin or numbers
// inside (pdftotext keeps a line's left-to-right runs where it sees
// them, not where they are read).
//
// readable is text as compared: normalised (a reader gives the
// presentation forms printed), without pdftotext's embedding controls,
// joiners and punctuation (pdftotext turns the letters around, not the
// signs between: "שלום. עולם" it reads "שלום .עולם", brackets the wrong
// way round), its spaces one.
func readable(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r >= 0x202A && r <= 0x202E || r == 0x200C || r == 0x200D:
			return -1
		case unicode.Is(unicode.Mn, r):
			// points and harakat: Windows' pdftotext repeats or moves some
			// (the renders check where they sit); the letters are compared
			return -1
		case unicode.IsPunct(r):
			return ' '
		}
		return r
	}, norm.NFKD.String(s))
	return strings.Join(strings.Fields(s), " ")
}

var rtlReadBack = map[string][]string{
	"Hebrew": {"בְּרֵאשִׁית", "בְּרֵאשִׁית בָּרָא אֱלֹהִים אֵת הַשָּׁמַיִם וְאֵת הָאָרֶץ. הספר הזה נכתב לאט בשנת", "ציטוט ארוך מספיק", "ראשון", "שני עם קישור", "דני", "נגר", "הערה בעברית."},
	"Arabic": {"بِسْمِ اللَّهِ", "بِسْمِ اللَّهِ الرَّحْمَٰنِ الرَّحِيمِ. هذا النص مكتوب ببطء في سنة", "أَعُودَ حَتَّى", "میخواهم بروم", "ملاحظة بالعربية."},
	"Mixed":  {"שָׁלוֹם", "سلام", "בְּרֵאשִׁית בָּרָא אֱלֹהִים"},
}

// A prose PDF prints text read right to left as Lexington's screenplay
// PDF does (pdf.Shaper): Arabic joined, marks placed, each line in
// visual order; a paragraph read right to left set from the right. Read
// back, it is as written.
func TestPDFRightToLeft(t *testing.T) {
	for _, name := range []string{"Hebrew", "Arabic", "Mixed"} {
		t.Run(name, func(t *testing.T) {
			text := rtlProse[name]
			if m := FromMarkdown(text).PDFMissing(PDFFonts{}); m != "" {
				t.Errorf("PDFMissing: %q", m)
			}
			file, _ := writeDevaProse(t, text, nil)
			_, drawing := written(t, FromMarkdown(text))
			// a paragraph read right to left ends at the right margin
			right := 0.0
			for _, m := range drawing.Pages[0].Marks {
				if tx, ok := m.(pdf.Text); ok && strings.IndexFunc(tx.Text, rtlLetter) >= 0 && tx.Size == textSize {
					right = math.Max(right, tx.X+tx.Width)
				}
			}
			pageW := 595.28 // A4
			if name != "Mixed" && math.Abs(right-(pageW-72)) > 1 {
				t.Errorf("right-to-left text ends at %.1f, not at the margin %.1f", right, pageW-72)
			}
			if _, err := exec.LookPath("pdftotext"); err != nil {
				return
			}
			out, err := exec.Command("pdftotext", "-enc", "UTF-8", file, "-").Output()
			if err != nil {
				t.Fatal(err)
			}
			read := readable(string(out))
			for _, want := range rtlReadBack[name] {
				if !strings.Contains(read, readable(want)) {
					t.Errorf("pdftotext reads without %q:\n%s", want, out)
				}
			}
		})
	}
}

// A paragraph's line in visual order: a Hebrew paragraph with a Latin
// word and a number shows them left to right inside it, its words from
// the right; a left-to-right one with a Hebrew phrase turns the phrase
// only.
func TestVisualOrder(t *testing.T) {
	p := &pdfWriter{doc: &Doc{}}
	rec := pdf.NewRecorder(pdf.NewFpdf("P", "pt", "A4"))
	p.d = rec
	p.f = PDFFonts{}.orGo()
	p.setup()
	p.d.AddPage()
	shown := func(text string) string {
		lines := p.lines(p.pieces([]Span{{Text: text}}, style{size: textSize}), 1000)
		var b strings.Builder
		for _, pc := range lines[0] {
			if pc.rtl {
				rs := []rune(pc.text)
				for i := len(rs) - 1; i >= 0; i-- {
					b.WriteRune(pdf.Mirror(rs[i]))
				}
				continue
			}
			b.WriteString(pc.text)
		}
		return b.String()
	}
	for _, c := range []struct{ text, seen string }{
		{"שלום world 12 עולם.", ".םלוע world 12 םולש"},
		{"Hello שלום עולם (x).", "Hello םלוע םולש (x)."},
		{"(שלום)", "(םולש)"},
		{"שלום. עולם", "םלוע .םולש"},
	} {
		if got := shown(c.text); got != c.seen {
			t.Errorf("%q shown %q, want %q", c.text, got, c.seen)
		}
	}
}
