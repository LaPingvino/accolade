package main

import (
	"bytes"
	"compress/zlib"
	"context"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/cli"
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

// Every style with and without its block and tint: only what goes with
// the style reaches Lexington (text with a block, a marker with a tint,
// which it refuses, never do), and each export works.
func TestExportColorScopeAndTint(t *testing.T) {
	script := "# Act One [[blue]]\n\nINT. BARN - DAY\n\nRain.\n"
	for _, s := range colorStyles {
		for _, block := range []bool{false, true} {
			for _, tint := range []bool{false, true} {
				job := exportJob{format: "default", colors: true, colorStyle: s.style, colorBlock: block, colorTint: tint}
				o := job.options("html", "x.html")
				wantScope := block && (s.style == lex.ColorMarker || s.style == lex.ColorBackground)
				wantTint := tint && s.style == lex.ColorLine
				if (o.ColorScope == "block") != wantScope || o.ColorTint != wantTint || (o.ColorScope != "" && o.ColorScope != "block") {
					t.Errorf("%s block %v tint %v: scope %q tint %v", s.style, block, tint, o.ColorScope, o.ColorTint)
				}
				if err := exportAs(script, filepath.Join(t.TempDir(), "s.html"), "html", job); err != nil {
					t.Errorf("%s block %v tint %v: %v", s.style, block, tint, err)
				}
				job.colors = false
				if o := job.options("html", "x.html"); o.ColorStyle != "" || o.ColorScope != "" || o.ColorTint {
					t.Errorf("colors off, %s: %+v", s.style, o)
				}
			}
		}
	}
	// the ones kept back would fail
	for _, o := range []cli.Options{{Colors: true, ColorStyle: "text", ColorScope: "block"}, {Colors: true, ColorStyle: "marker", ColorTint: true}} {
		o.Elements, o.To, o.Output, o.ConfigFile = "default", "html", filepath.Join(t.TempDir(), "x.html"), lexingtonConfig()
		if err := cli.Write(context.Background(), exportJob{}.script(script), o); err == nil {
			t.Errorf("Lexington takes %s with scope %q, tint %v", o.ColorStyle, o.ColorScope, o.ColorTint)
		}
	}
}

// The dialog: Whole block with a marker or background, Tint with the
// line, neither without Colors; saved for the preview and printing.
func TestExportDialogScopeAndTint(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	t.Cleanup(func() {
		w.settings.SetBoolean("export-colors", false)
		w.settings.SetString("export-color-style", "line")
		w.settings.SetBoolean("export-color-block", false)
		w.settings.SetBoolean("export-color-tint", false)
	})
	ed := NewExportDialog(w)
	ed.Show()
	if !ed.colorBlockCheck.Disabled() || !ed.colorTintCheck.Disabled() {
		t.Error("block or tint without colors")
	}
	ed.colorsCheck.SetChecked(true)
	for _, c := range []struct {
		style       string
		block, tint bool // enabled
	}{
		{"Line in the margin", false, true},
		{"Marker", true, false},
		{"Background", true, false},
		{"Text color", false, false},
	} {
		ed.colorStyleSelect.SetSelected(c.style)
		if ed.colorBlockCheck.Disabled() == c.block || ed.colorTintCheck.Disabled() == c.tint {
			t.Errorf("%s: block enabled %v, tint enabled %v", c.style, !ed.colorBlockCheck.Disabled(), !ed.colorTintCheck.Disabled())
		}
	}
	if u := unnamed(w.fyneWindow.Canvas()); len(u) > 0 {
		t.Errorf("no name for %v", u)
	}
	ed.colorStyleSelect.SetSelected("Background")
	ed.colorBlockCheck.SetChecked(true)
	ed.colorTintCheck.SetChecked(true) // (kept for the line, not passed)
	ed.scroll.ScrollToBottom()
	capture(t, w.fyneWindow.Canvas(), "export-color-scope.png")
	ed.saveSettings()
	if o := w.savedExportJob().options("pdf", "x.pdf"); o.ColorStyle != "background" || o.ColorScope != "block" || o.ColorTint {
		t.Errorf("preview and print: %+v", o)
	}
	ed.cancel()
	ed = NewExportDialog(w)
	if !ed.colorBlockCheck.Checked || !ed.colorTintCheck.Checked || ed.colorBlockCheck.Disabled() || !ed.colorTintCheck.Disabled() {
		t.Error("block and tint not kept")
	}
}
