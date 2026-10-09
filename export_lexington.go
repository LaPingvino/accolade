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

	"github.com/LaPingvino/lexington/fdx"
	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/html"
	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/ocrwasm"
	"github.com/LaPingvino/lexington/office"
	"github.com/LaPingvino/lexington/pdf"
	"github.com/LaPingvino/lexington/pdfin"
	"github.com/LaPingvino/lexington/rules"
)

// PDF and HTML export go through the lexington library, which parses
// the script the same way the lexington command line tool does.

var englishScenes = rules.DefaultConf().Scenes["en"]

func parseForExport(content string) lex.Screenplay {
	return fountain.Parse(englishScenes, strings.NewReader(content))
}

// exportJob is how to export: the script format's rules, the paper
// ("letter", "a4", "a5"; "" for the format's suggestion or Letter), and
// what to do with the script.
type exportJob struct {
	elements      rules.Set
	page          string
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

// exportPDF renders the script to a PDF at outputPath.
func exportPDF(content, outputPath string, job exportJob) (err error) {
	return writeViaTemp(outputPath, func(tmp string) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("PDF rendering failed: %v", r)
			}
		}()
		w := &pdf.PDFWriter{OutputFile: tmp, Elements: job.elements, Page: job.paper()}
		return w.Write(nil, job.script(content))
	})
}

// exportHTML renders the script to a standalone HTML page at outputPath.
func exportHTML(content, outputPath string, job exportJob) error {
	return writeViaTemp(outputPath, func(tmp string) error {
		f, err := os.Create(tmp)
		if err != nil {
			return err
		}
		w := &html.HTMLWriter{Elements: job.elements}
		if err := w.Write(io.Writer(f), job.script(content)); err != nil {
			f.Close()
			return err
		}
		return f.Close()
	})
}

// exportDocument writes the script as a Word ("docx") or OpenDocument
// ("odt") document.
func exportDocument(content, outputPath, format string, job exportJob) error {
	return writeViaTemp(outputPath, func(tmp string) error {
		f, err := os.Create(tmp)
		if err != nil {
			return err
		}
		var w interface {
			Write(io.Writer, lex.Screenplay) error
		} = &office.DOCXWriter{Elements: job.elements, Page: job.paper()}
		if format == "odt" {
			w = &office.ODTWriter{Elements: job.elements, Page: job.paper()}
		}
		if err := w.Write(f, job.script(content)); err != nil {
			f.Close()
			return err
		}
		return f.Close()
	})
}

// paper is the job's paper for Lexington.
func (j exportJob) paper() string { return j.page }

// exportFDX writes the script as a Final Draft document.
func exportFDX(content, outputPath string, job exportJob) error {
	return writeViaTemp(outputPath, func(tmp string) error {
		f, err := os.Create(tmp)
		if err != nil {
			return err
		}
		if err := (&fdx.FDXWriter{}).Write(f, job.script(content)); err != nil {
			f.Close()
			return err
		}
		return f.Close()
	})
}

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
func importPDF(ctx context.Context, path string, progress func(done, pages int)) (text string, skipped []string, err error) {
	opts := pdfin.Options{Context: ctx, Progress: progress, Skipped: func(page int, err error) {
		skipped = append(skipped, fmt.Sprintf("page %d: %v", page, err))
	}}
	if t, ok := pdfin.InstalledTesseract(); ok {
		opts.OCR = t
	} else {
		o := ocrwasm.New()
		defer o.Close(context.Background())
		opts.OCR = o
	}
	screenplay, err := pdfin.ReadFileWith(path, opts)
	if err != nil {
		return "", skipped, err
	}
	var buf bytes.Buffer
	if err := (&fountain.FountainWriter{SceneConfig: englishScenes}).Write(&buf, screenplay); err != nil {
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
