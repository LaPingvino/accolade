package prose

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/pdf"
	"github.com/LaPingvino/lexington/pdfin"
)

// devaProse is Hindi prose with every part of the shaping (conjuncts,
// the i-matra before its consonant, reph and rakar, the eyelash ra with
// its joiner) in each place a prose PDF prints text: a bold heading, an
// italic quote, a link, a list, a table, a footnote, and a word too long
// for its line.
var devaProse = "# हिन्दी में कहानी\n\n" +
	"क्षत्रिय किसान ने **धर्म** और *श्रद्धा* से ज्ञान पाया। कृष्ण की ट्रेन आई, र्\u200dय भी।[^1]\n\n" +
	"> हिन्दी की *कविता* माँ के लिए।\n\n" +
	"* पहला [कृष्ण](https://example.org/krishna)\n* दूसरा शर्मा\n\n" +
	"| नाम | काम |\n| --- | --- |\n| शर्मा | ट्रेन चलाना |\n\n" +
	"लम्बाशब्द" + strings.Repeat("क्षत्रियकृष्णश्रद्धाज्ञान", 6) + "\n\n" +
	"[^1]: टिप्पणी: श्रद्धा और ज्ञान।\n"

// devaProseWords are what the PDF must read back as written.
var devaProseWords = []string{"हिन्दी में कहानी", "क्षत्रिय किसान ने धर्म और श्रद्धा से ज्ञान पाया।",
	"कृष्ण की ट्रेन आई, र्\u200dय भी।", "हिन्दी की कविता माँ के लिए।", "पहला कृष्ण", "दूसरा शर्मा", "शर्मा", "ट्रेन चलाना",
	"टिप्पणी: श्रद्धा और ज्ञान।", "हिन्दी कहानी", "लेखक"}

func writeDevaProse(t *testing.T, text string, titlePage []string) (string, []byte) {
	t.Helper()
	d := FromMarkdown(text)
	d.TitlePage = titlePage
	var b bytes.Buffer
	if _, err := d.WritePDF(&b, PDFFonts{}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if shots := os.Getenv("SHOT_DIR"); shots != "" {
		dir = shots
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	file := filepath.Join(dir, strings.ReplaceAll(t.Name(), "/", "-")+".pdf")
	if err := os.WriteFile(file, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("SHOT_DIR") != "" {
		if out, err := exec.Command("pdftoppm", "-png", "-r", "600", file, strings.TrimSuffix(file, ".pdf")).CombinedOutput(); err != nil {
			t.Errorf("pdftoppm: %v %s", err, out)
		}
	}
	return file, b.Bytes()
}

// A prose PDF prints Devanagari shaped (Lexington's Shaper): its glyphs
// as the font's features make them, the text read back as written, by
// Lexington's import and by pdftotext; a long word broken between its
// clusters only.
func TestPDFDevanagari(t *testing.T) {
	file, _ := writeDevaProse(t, devaProse, []string{"हिन्दी कहानी", "लेखक"})
	if missing := FromMarkdown(devaProse).PDFMissing(PDFFonts{}); missing != "" {
		t.Errorf("PDFMissing: %q", missing)
	}
	pages, err := pdfin.Pages(file)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, p := range pages {
		for _, l := range p {
			lines = append(lines, strings.TrimSpace(l.Text))
		}
	}
	all := strings.Join(strings.Fields(strings.Join(lines, "\n")), " ")
	for _, want := range devaProseWords {
		if !strings.Contains(all, want) {
			t.Errorf("read back without %q:\n%s", want, strings.Join(lines, "\n"))
		}
	}
	long := "लम्बाशब्द" + strings.Repeat("क्षत्रियकृष्णश्रद्धाज्ञान", 6)
	var parts []string
	for _, l := range lines {
		if strings.HasPrefix(l, "लम्बा") || len(parts) > 0 && len(strings.Join(parts, "")) < len(long) {
			parts = append(parts, l)
		}
	}
	if len(parts) < 2 || strings.Join(parts, "") != long {
		t.Errorf("the long word read back as %q", parts)
	}
	for i := 1; i < len(parts); i++ {
		a, b := []rune(parts[i-1]), []rune(parts[i])
		if pdf.JoinsBefore(a[len(a)-1], b[0]) {
			t.Errorf("the long word broken inside a cluster: %q | %q", parts[i-1], parts[i])
		}
	}
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return
	}
	out, err := exec.Command("pdftotext", file, "-").Output()
	if err != nil {
		t.Fatal(err)
	}
	read := strings.Join(strings.Fields(string(out)), " ")
	for _, want := range devaProseWords {
		if !strings.Contains(read, want) {
			t.Errorf("pdftotext reads without %q:\n%s", want, out)
		}
	}
}

// clusters keep a letter with its marks and a conjunct whole.
func TestClusters(t *testing.T) {
	got := strings.Join(clusters("क्षत्रिय र्\u200dय é"), "|")
	if want := "क्ष|त्रि|य| |र्\u200dय| |é"; got != want {
		t.Errorf("clusters: %q, want %q", got, want)
	}
}
