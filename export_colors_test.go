package main

import (
	"bytes"
	"compress/zlib"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/lex"
)

// The Export dialog's color style: chosen only with Colors on, passed to
// Lexington, saved for the preview and printing.
func TestExportColorStyle(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	t.Cleanup(func() {
		w.settings.SetBoolean("export-colors", false)
		w.settings.SetString("export-color-style", "line")
	})
	w.textEditor.SetText("# ACT ONE [[blue]]\n\nINT. BARN - DAY\n\nRain.\n")
	ed := NewExportDialog(w)
	ed.Show()
	if !ed.colorStyleSelect.Disabled() || ed.colorStyleSelect.Selected != "Line in the margin" {
		t.Errorf("without colors: disabled %v, %q", ed.colorStyleSelect.Disabled(), ed.colorStyleSelect.Selected)
	}
	if o := ed.job().options("pdf", "x.pdf"); o.ColorStyle != "" {
		t.Errorf("a style without colors: %q", o.ColorStyle)
	}
	ed.colorsCheck.SetChecked(true)
	if ed.colorStyleSelect.Disabled() {
		t.Error("no style with colors")
	}
	if u := unnamed(w.fyneWindow.Canvas()); len(u) > 0 {
		t.Errorf("no name for %v", u)
	}
	for _, s := range colorStyles {
		ed.colorStyleSelect.SetSelected(s.label)
		if o := ed.job().options("pdf", "x.pdf"); !o.Colors || o.ColorStyle != string(s.style) {
			t.Errorf("%s: colors %v, style %q", s.label, o.Colors, o.ColorStyle)
		}
		if _, err := lex.ParseColorStyle(string(s.style)); err != nil {
			t.Errorf("%s: %v", s.label, err)
		}
	}
	ed.colorStyleSelect.SetSelected("Background")
	capture(t, w.fyneWindow.Canvas(), "export-color-style.png")
	ed.saveSettings()
	if o := w.savedExportJob().options("pdf", "x.pdf"); !o.Colors || o.ColorStyle != "background" {
		t.Errorf("preview and print: colors %v, style %q", o.Colors, o.ColorStyle)
	}
	ed.cancel()
	if NewExportDialog(w).colorStyleSelect.Selected != "Background" {
		t.Error("the style is not kept")
	}
}

// Joop's script: colors only on sections, which the screenplay format
// does not print; their scenes get the bar in the margin, the others none.
func TestExportSectionColorBar(t *testing.T) {
	script := "# Act One [[blue]]\n\nINT. BARN - DAY\n\nRain.\n\n# Act Two\n\nINT. PLAIN - DAY\n\nNothing.\n"
	bars := func(job exportJob) []string {
		t.Helper()
		job.format = "default"
		out := filepath.Join(t.TempDir(), "s.pdf")
		if err := exportAs(script, out, "pdf", job); err != nil {
			t.Fatal(err)
		}
		b, _ := os.ReadFile(out)
		return regexp.MustCompile(`36\.00 [\d.]+ 2\.88 -[\d.]+ re f`).FindAllString(pdfStreams(b), -1)
	}
	if got := bars(exportJob{colors: true, colorStyle: lex.ColorLine}); len(got) != 1 {
		t.Errorf("colors on: bars %q, want the one of Act One's scene", got)
	}
	if got := bars(exportJob{colorStyle: lex.ColorLine}); len(got) != 0 {
		t.Errorf("colors off: bars %q", got)
	}
}

var pdfStream = regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\nendstream`)

// pdfStreams are a PDF's streams, inflated where they are compressed.
func pdfStreams(b []byte) string {
	var all strings.Builder
	for _, m := range pdfStream.FindAllSubmatch(b, -1) {
		if r, err := zlib.NewReader(bytes.NewReader(m[1])); err == nil {
			if d, err := io.ReadAll(r); err == nil {
				all.Write(d)
				continue
			}
		}
		all.Write(m[1])
	}
	return all.String()
}
