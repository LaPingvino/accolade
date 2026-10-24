package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/fdx"
	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/pdfin"
	"github.com/LaPingvino/lexington/rules"
)

// PDF and HTML export go through the lexington library, which parses
// the script the same way the lexington command line tool does.

var englishScenes = rules.DefaultConf().Scenes["en"]

func parseForExport(content string) lex.Screenplay {
	return fountain.Parse(englishScenes, strings.NewReader(content))
}

// exportJob is how to export: the script format (a Lexington preset's
// key: its rules), the paper ("letter", "a4", "a5"; "" for the format's
// suggestion or Letter), a template for the formats that take one, and
// what to do with the script.
type exportJob struct {
	format        string
	page          string
	template      string
	omitTitlePage bool     // leave the title page out
	numberScenes  bool     // number the scenes that have no number
	scenes        []string // the script language's scene headings (nil: English)
}

// script is the content parsed and prepared for the job.
func (j exportJob) script(content string) lex.Screenplay {
	scenes := j.scenes
	if scenes == nil {
		scenes = englishScenes
	}
	s := fountain.Parse(scenes, strings.NewReader(content))
	if j.omitTitlePage {
		s = withoutTitlePage(s)
	}
	if j.numberScenes {
		s = numberScenes(s)
	}
	return s
}

// withoutTitlePage is the screenplay without its title page (up to and
// including the page break after it).
func withoutTitlePage(s lex.Screenplay) lex.Screenplay {
	if len(s) == 0 || s[0].Type != lex.TypeTitlePage {
		return s
	}
	for i, l := range s {
		if l.Type == lex.TypeNewPage {
			return s[i+1:]
		}
	}
	return nil
}

// numberScenes gives the scenes without a number (#12#) the next one:
// 1, 2, 3, or after a scene numbered 7, 8.
func numberScenes(s lex.Screenplay) lex.Screenplay {
	out := make(lex.Screenplay, len(s))
	copy(out, s)
	n := 0
	for i, l := range out {
		if l.Type != lex.TypeScene {
			continue
		}
		if _, num := lex.SceneNumber(l.Contents); num != "" {
			if v, err := strconv.Atoi(num); err == nil {
				n = v
			}
			continue
		}
		n++
		out[i].Contents = strings.TrimSpace(l.Contents) + " #" + strconv.Itoa(n) + "#"
	}
	return out
}

// exportAs writes the script as Lexington's format to (its -to), with
// Lexington's command line's options: what "accolade -to ..." makes.
func exportAs(content, outputPath, to string, job exportJob) error {
	if to == "pdf" {
		// a script the PDF writer cannot print as written (Hebrew, CJK,
		// right to left): a browser on the computer prints Lexington's
		// HTML (browser_pdf.go)
		if b := localBrowser(); b != "" && pdfWarning(content, job.scenes) != "" {
			dir, err := os.MkdirTemp("", "accolade-pdf-")
			if err != nil {
				return err
			}
			defer os.RemoveAll(dir)
			page := filepath.Join(dir, "script.html")
			if err := exportAs(content, page, "html", job); err != nil {
				return err
			}
			return browserPDF(b, page, outputPath)
		}
	}
	return writeViaTemp(outputPath, func(tmp string) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%s export failed: %v", to, r)
			}
		}()
		return cli.Write(context.Background(), job.script(content), cli.Options{
			ConfigFile:   lexingtonConfig(),
			FormatDir:    formatDirs(),
			Elements:     job.format,
			Page:         job.page,
			To:           to,
			Output:       tmp,
			TemplatePath: job.template,
		})
	})
}

// lexingtonConfig is Lexington's settings file in Accolade's settings
// folder: its element sets and scene headings, as for the lexington
// command (Lexington's defaults and presets without it).
func lexingtonConfig() string { return filepath.Join(getConfigDir(), "lexington.toml") }

// importFDX converts a Final Draft document to Fountain text.
func importFDX(r io.Reader) (string, error) {
	screenplay, err := fdx.ParseWithError(r)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := (&fountain.FountainWriter{SceneConfig: englishScenes}).Write(&buf, screenplay); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// importPDF reads a screenplay from a PDF as Fountain text. Scanned
// pages are read with OCR: tesseract if it is installed, else the
// built-in one (slower). progress (may be nil) is told the pages read;
// skipped are the pages that could not be read.
func importPDF(ctx context.Context, path string, scenes []string, progress func(done, pages int)) (text string, skipped []string, err error) {
	if scenes == nil {
		scenes = englishScenes
	}
	opts := pdfin.Options{Context: ctx, Progress: progress, Scenes: scenes, Skipped: func(page int, err error) {
		skipped = append(skipped, fmt.Sprintf("page %d: %v", page, err))
	}}
	if ocr, done := pdfOCR(); ocr != nil {
		defer done()
		opts.OCR = ocr
	}
	screenplay, err := pdfin.ReadFileWith(path, opts)
	if err != nil {
		return "", skipped, err
	}
	var buf bytes.Buffer
	if err := (&fountain.FountainWriter{SceneConfig: scenes}).Write(&buf, screenplay); err != nil {
		return "", skipped, err
	}
	return buf.String(), skipped, nil
}

// writeViaTemp lets write produce the file under a temporary name next to
// outputPath and moves it into place only if writing succeeded.
func writeViaTemp(outputPath string, write func(tmp string) error) error {
	tmp, err := os.CreateTemp(filepath.Dir(outputPath), "."+filepath.Base(outputPath)+".*.tmp")
	if err != nil {
		return err
	}
	tmp.Close()
	defer os.Remove(tmp.Name()) // no-op once renamed

	if err := write(tmp.Name()); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), outputPath)
}
