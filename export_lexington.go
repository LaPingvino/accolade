package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/LaPingvino/lexington/fdx"
	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/html"
	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/pdf"
	"github.com/LaPingvino/lexington/rules"
)

// PDF and HTML export go through the lexington library, which parses
// the script the same way the lexington command line tool does.

var englishScenes = rules.DefaultConf().Scenes["en"]

func parseForExport(content string) lex.Screenplay {
	return fountain.Parse(englishScenes, strings.NewReader(content))
}

// exportPDF renders the script to a PDF at outputPath.
func exportPDF(content, outputPath string) (err error) {
	return writeViaTemp(outputPath, func(tmp string) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("PDF rendering failed: %v", r)
			}
		}()
		w := &pdf.PDFWriter{OutputFile: tmp, Elements: rules.Default}
		return w.Write(nil, parseForExport(content))
	})
}

// exportHTML renders the script to a standalone HTML page at outputPath.
func exportHTML(content, outputPath string) error {
	return writeViaTemp(outputPath, func(tmp string) error {
		f, err := os.Create(tmp)
		if err != nil {
			return err
		}
		w := &html.HTMLWriter{Elements: rules.Default}
		if err := w.Write(io.Writer(f), parseForExport(content)); err != nil {
			f.Close()
			return err
		}
		return f.Close()
	})
}

// exportFDX writes the script as a Final Draft document.
func exportFDX(content, outputPath string) error {
	return writeViaTemp(outputPath, func(tmp string) error {
		f, err := os.Create(tmp)
		if err != nil {
			return err
		}
		if err := (&fdx.FDXWriter{}).Write(f, parseForExport(content)); err != nil {
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
