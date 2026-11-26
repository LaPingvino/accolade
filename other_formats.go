package main

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/fountain"
)

// Other programs' screenplays (Fade In, Trelby, KIT Scenarist, and what
// Lexington reads next) open, import and save through Lexington's own
// list of the formats it reads (cli.Inputs) and writes (cli.Formats):
// a format Lexington learns shows up here by itself. Fountain, Final
// Draft and PDF have ways in of their own (window.go's importFile), and
// Lexington's lex is no one's script.

// otherFormat is the other program's format a file is in, by its name.
func otherFormat(path string) (cli.InputFormat, bool) {
	f, ok := cli.InputFor(path)
	if !ok {
		return f, false
	}
	switch f.Key {
	case "fountain", "fdx", "pdf", "lex":
		return f, false
	}
	return f, true
}

// otherFormatExts are the other programs' formats' extensions.
func otherFormatExts() []string {
	var out []string
	for _, f := range cli.Inputs {
		if _, ok := otherFormat("x" + f.Exts[0]); ok {
			out = append(out, f.Exts...)
		}
	}
	return out
}

// otherSaveFormat is the other program's format Save As writes a file in,
// by its name: one Lexington reads and writes.
func otherSaveFormat(path string) (cli.Format, bool) {
	in, ok := otherFormat(path)
	if !ok {
		return cli.Format{}, false
	}
	return cli.FormatFor(in.Key)
}

// readOtherFormat reads another program's screenplay as Fountain text.
func readOtherFormat(path string) (string, error) {
	sp, err := cli.ReadScreenplay(context.Background(), path, cli.Options{ConfigFile: lexingtonConfig()})
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := (&fountain.FountainWriter{SceneConfig: englishScenes}).Write(&buf, sp); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// importOther opens another program's screenplay as a new, unsaved
// script, as a Final Draft file opens.
func (w *MainWindow) importOther(path string, f cli.InputFormat) error {
	text, err := readOtherFormat(path)
	if err != nil {
		return fmt.Errorf("could not import %s: %v", filepath.Base(path), err)
	}
	w.showImported(path, text, f.Name)
	return nil
}

// saveOtherCopy is Save As to another program's format: the script
// written there through Lexington, the window going on with its own file
// (that program's format holds less than Fountain: writing it on every
// save would lose what it has no place for).
func (w *MainWindow) saveOtherCopy(path string, f cli.Format) {
	content := w.textEditor.Text()
	if err := exportAs(content, path, f.Key, w.layoutJob()); err != nil {
		w.setStatus("Could not save " + filepath.Base(path) + ": " + err.Error())
		return
	}
	keeps := "this window"
	if w.currentFile != "" {
		keeps = filepath.Base(w.currentFile)
	}
	w.setStatus("Saved a copy as " + f.Name + ": " + filepath.Base(path) + "; " + keeps + " stays the script you write in")
}

// isOtherExt tells whether an extension is another program's format's.
func isOtherExt(ext string) bool {
	return slices.Contains(otherFormatExts(), strings.ToLower(ext))
}
