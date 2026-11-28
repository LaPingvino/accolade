// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/lexington/bundle"
	"github.com/LaPingvino/lexington/pdf"
)

// lexingtonSamples is Lexington's folder of real Highland files
// (bundle/testdata/highland, highwater-too's, MIT), in the module cache.
func lexingtonSamples(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/LaPingvino/lexington").Output()
	if err != nil {
		t.Skip("no go list:", err)
	}
	dir := filepath.Join(strings.TrimSpace(string(out)), "bundle", "testdata", "highland")
	if _, err := os.Stat(filepath.Join(dir, "IncludedScene.fountain")); err != nil {
		t.Fatal(err)
	}
	return dir
}

// copyIn copies src to dst.
func copyIn(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// drawnTexts are the texts of a drawing's pages, a page a string.
func drawnTexts(d pdf.Drawing) []string {
	var out []string
	for _, p := range d.Pages {
		var b []string
		for _, m := range p.Marks {
			if tx, ok := m.(pdf.Text); ok {
				b = append(b, tx.Text)
			}
		}
		out = append(out, strings.Join(b, "|"))
	}
	return out
}

// A real Highland file (ScreenplayEdgeCases, with its {{HEADER}},
// {{FOOTER}} and {{INCLUDE}}) opens, shows its include as a chip that
// shows the file, prints with the included scene, the header and footer,
// and saves back byte for byte: the directives as Highland wrote them.
func TestHighlandRoundTrip(t *testing.T) {
	src := lexingtonSamples(t)
	dir := t.TempDir()
	p := filepath.Join(dir, "ScreenplayEdgeCases.highland")
	copyIn(t, filepath.Join(src, "ScreenplayEdgeCases.highland"), p)
	copyIn(t, filepath.Join(src, "IncludedScene.fountain"), filepath.Join(dir, "IncludedScene.fountain"))
	b, err := bundle.Read(p)
	if err != nil {
		t.Fatal(err)
	}
	orig, _ := b.ReadText()

	w := newTestWindow(t, "")
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	text := w.textEditor.Text()
	if text != string(orig) {
		t.Fatal("the text read is not the bundle's")
	}
	// the chip
	chips := w.highlandChips(text)
	var inc *includeLine
	for _, c := range chips {
		if c.Label == "⤵ IncludedScene.fountain · 1 scene" && c.Tap && !c.Warn {
			incs := findIncludes(text[:bodyEndByte(text)])
			inc = &incs[0]
			w.chipTapped(c, w.textEditor.Position(), w.textEditor.Size())
		}
	}
	if inc == nil {
		t.Fatalf("no include chip in %+v", chips)
	}
	if w.includeShown == nil {
		t.Fatal("the included file not shown on a tap")
	}
	w.includeShown.Hide()
	w.includeShown = nil
	w.textEditor.SetCursorOffset(inc.start)
	w.foldsOf(w.textEditor).fold(true, false) // zo
	if w.includeShown == nil {
		t.Error("zo did not show the included file")
	} else {
		w.includeShown.Hide()
	}
	// Page Setup shows the script's own header and footer
	ps := w.newPageSetup("pdf", nil)
	if ps.headerEntry.Text != "%# | %p" || ps.footerEntry.Text != "Left | Centre | Right" || !ps.headerOwn {
		t.Errorf("page setup: %q %q", ps.headerEntry.Text, ps.footerEntry.Text)
	}
	// printed: the included scene, the header (the section, the page), the footer
	d, err := drawScript(text, filepath.Join(dir, "out.pdf"), w.layoutJob())
	if err != nil {
		t.Fatal(err)
	}
	pages := drawnTexts(d)
	all := strings.Join(pages, "\n")
	for _, want := range []string{"INT. INCLUDED SCENE - DAY", "This scene comes from an included file.", "Left|Centre|Right", "Act One"} {
		if !strings.Contains(all, want) {
			t.Errorf("not printed: %q", want)
		}
	}
	if strings.Contains(all, "{{") {
		t.Error("a directive printed")
	}
	// saved back as it was
	if err := w.saveToFile(p); err != nil {
		t.Fatal(err)
	}
	b, _ = bundle.Read(p)
	if got, _ := b.ReadText(); string(got) != string(orig) {
		t.Error("the text changed in the round trip")
	}
}

// Page Setup's header and footer print with a script without its own;
// a script's own wins, and changed in Page Setup it is changed in the
// script (taken out when emptied).
func TestPageSetupHeaderFooter(t *testing.T) {
	w := newTestWindow(t, "INT. ROOM - DAY\n\nTake {{SERIES}}, {{SERIES}}.\n")
	p := w.pageSetup()
	p.header, p.footer = "Draft | %p", "Printed {{TIMESTAMP}}"
	w.setPageSetup(p)
	t.Cleanup(func() { // (the settings outlive the test)
		p.header, p.footer = "", ""
		w.setPageSetup(p)
	})
	d, err := drawScript(w.textEditor.Text(), filepath.Join(t.TempDir(), "a.pdf"), w.layoutJob())
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(drawnTexts(d), "\n")
	for _, want := range []string{"Draft", "|1|", "Take 1, 2."} {
		if !strings.Contains("|"+got+"|", want) {
			t.Errorf("no %q in %q", want, got)
		}
	}
	if !strings.Contains(w.pageSummary(w.pageSetup(), "pdf"), "header and footer") {
		t.Error("the summary does not say it")
	}

	w.textEditor.SetText("{{HEADER: Mine | %p}}\n\nINT. ROOM - DAY\n\nText.\n")
	ps := w.newPageSetup("pdf", nil)
	if ps.headerEntry.Text != "Mine | %p" || !ps.headerOwn || ps.footerOwn || ps.footerEntry.Text != "Printed {{TIMESTAMP}}" {
		t.Fatalf("shown: %q %v, %q", ps.headerEntry.Text, ps.headerOwn, ps.footerEntry.Text)
	}
	ps.headerEntry.SetText("Yours | %p")
	ps.ok()
	if got := w.textEditor.Text(); got != "{{HEADER: Yours | %p}}\n\nINT. ROOM - DAY\n\nText.\n" {
		t.Errorf("script: %q", got)
	}
	if w.settings.GetString(headerPref) != "Draft | %p" {
		t.Errorf("the setting changed: %q", w.settings.GetString(headerPref))
	}
	ps = w.newPageSetup("pdf", nil)
	ps.headerEntry.SetText("")
	ps.ok()
	if got := w.textEditor.Text(); got != "INT. ROOM - DAY\n\nText.\n" {
		t.Errorf("emptied: %q", got)
	}
}

// Insert › Include File… writes the line, relative to the script's
// folder; an include not there is a warning chip; markers are chips and
// Next Marker goes round them.
func TestIncludeInsertAndMarkers(t *testing.T) {
	cases := map[string]string{
		"Action.|":             "Action.\n\n{{INCLUDE: act2.fountain}}",
		"Action.|\nMore.":      "Action.\n\n{{INCLUDE: act2.fountain}}\n\nMore.",
		"Action.\n\n|\n\nMore": "Action.\n\n{{INCLUDE: act2.fountain}}\n\nMore",
	}
	for in, want := range cases {
		b := buffer.New(strings.Replace(in, "|", "", 1))
		b.SetCursor(len([]rune(in[:strings.Index(in, "|")])), false)
		insertInclude(b, "act2.fountain")
		if b.Text() != want {
			t.Errorf("%q: %q, want %q", in, b.Text(), want)
		}
	}
	w := newTestWindow(t, "")
	dir := t.TempDir()
	w.currentFile = filepath.Join(dir, "main.fountain")
	if got := w.includeRel(filepath.Join(dir, "acts", "two.fountain")); got != "acts/two.fountain" {
		t.Errorf("relative: %q", got)
	}
	w.textEditor.SetText("INT. A - DAY\n\n{{INCLUDE: gone.fountain}}\n\nOne {{%m: first}} and {{%m}}.\n")
	chips := w.highlandChips(w.textEditor.Text())
	var labels []string
	for _, c := range chips {
		labels = append(labels, c.Label)
	}
	if strings.Join(labels, "; ") != "⤵ gone.fountain · not found; ⚑ first; ⚑" || !chips[0].Warn {
		t.Errorf("chips: %q", labels)
	}
	w.textEditor.SetCursorOffset(0)
	var at []int
	for range 3 {
		w.gotoMarker(false)
		at = append(at, w.textEditor.CursorOffset())
	}
	if at[0] != chips[1].Start || at[1] != chips[2].Start || at[2] != chips[1].Start {
		t.Errorf("next marker: %v", at)
	}
	w.gotoMarker(true)
	if w.textEditor.CursorOffset() != chips[2].Start {
		t.Errorf("previous marker: %d", w.textEditor.CursorOffset())
	}
}

// TestHighlandScreenshots draws an include's chip folded and open, its
// file shown, and prints a PDF with Highland's header, footer and series
// (SHOT_DIR, as TestScreenshots).
func TestHighlandScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(dir, 0o755)
	w := newTestWindow(t, "")
	w.fyneWindow.Resize(fyne.NewSize(1000, 650))
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "act2.fountain"), []byte("INT. CELLAR - NIGHT\n\nThe second act, from a file of its own.\n\nBRAM\nDown here, {{SERIES}}.\n"), 0o644)
	text := "Title: Directives\n\n{{HEADER: Directives / %# | %p of %pp}}\n{{FOOTER: Draft of {{TIMESTAMP}}}}\n\n# Act {{SERIES}}\n\nINT. KITCHEN - DAY\n\nTake {{SERIES}}. {{%m: check the props}}\n\n{{INCLUDE: act2.fountain}}\n\n# Act {{SERIES}}\n\nEXT. GARDEN - NIGHT\n\nThe end.\n"
	w.currentFile = filepath.Join(src, "main.fountain")
	w.textEditor.SetText(text)
	shot := func(name string) {
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, w.fyneWindow.Canvas().Capture())
		f.Close()
	}
	w.textEditor.SetCursorOffset(0)
	shot("include-folded.png")
	inc := findIncludes(text)[0]
	w.textEditor.SetCursorOffset(inc.start + 3)
	shot("include-open.png")
	w.showInclude(inc)
	shot("include-shown.png")
	w.includeShown.Hide()
	if _, err := drawScript(text, filepath.Join(dir, "highland-directives.pdf"), w.layoutJob()); err != nil {
		t.Fatal(err)
	}
}
