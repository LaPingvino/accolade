// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/lex"
)

// filterScript has tags (a color, a place) and three characters.
const filterScript = `Title: Sides

# ACT ONE [[color blue]]

INT. BARN - DAY [[London]]

EDWARD
Is it raining?

ANNA
It always rains here in the barn.

EDWARD
Then we stay.

EXT. FIELD - NIGHT #4# [[red]]

ANNA
The field is dry.

# ACT TWO

INT. KITCHEN - DAY [[london]] [[note to self]]

BOB (V.O.)
Dinner.
`

func TestScriptTagsAndSpeakers(t *testing.T) {
	if got, want := scriptTags(filterScript, nil), []string{"red", "blue", "London", "note to self"}; !reflect.DeepEqual(got, want) {
		t.Errorf("tags %q, want %q", got, want)
	}
	if got, want := scriptSpeakers(filterScript, nil), []string{"ANNA", "EDWARD", "BOB"}; !reflect.DeepEqual(got, want) {
		t.Errorf("speakers %q, want %q", got, want)
	}
	for _, c := range []struct {
		tags      []string
		character string
		want      string
	}{
		{nil, "", ""},
		{nil, "EDWARD", " - EDWARD sides"},
		{[]string{"London", "red"}, "", " - London, red"},
		{[]string{"a/b"}, "ANNA", " - ab - ANNA sides"},
	} {
		if got := filterSuffix(c.tags, c.character); got != c.want {
			t.Errorf("filterSuffix(%q, %q) = %q", c.tags, c.character, got)
		}
	}
}

// The dialog's choices are Lexington's options: colors (Page Setup's), tags, sides in
// the mode chosen (none chosen: no export), the cue's length.
func TestExportDialogFilters(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(filterScript)
	t.Cleanup(func() { w.settings.SetBoolean("export-colors", false) })
	ed := NewExportDialog(w)
	ed.Show()
	if w.pageSetup().colors {
		t.Error("colors on by default")
	}
	if o := ed.job().options("pdf", "x.pdf"); o.Colors || o.Tags != nil || o.Character != "" || o.Sides != "" {
		t.Errorf("filters by default: %+v", o)
	}
	if ed.sidesBox.Visible() {
		t.Error("the sides' modes without a character")
	}
	check := func(what string) {
		if u := unnamed(w.fyneWindow.Canvas()); len(u) > 0 {
			t.Errorf("%s: no name for\n  %s", what, strings.Join(u, "\n  "))
		}
	}
	check("export with filters")

	w.settings.SetBoolean("export-colors", true) // Page Setup's
	for _, c := range ed.tagChecks {
		if c.Text == "London" || c.Text == "red" {
			c.SetChecked(true)
		}
	}
	ed.characterSelect.SetSelected("EDWARD")
	if !ed.sidesBox.Visible() || ed.sidesRadio.Selected != "" {
		t.Errorf("sides: shown %v, mode %q preselected", ed.sidesBox.Visible(), ed.sidesRadio.Selected)
	}
	if err := ed.validateInputs(); err == nil || !strings.Contains(err.Error(), "full scenes, cue lines or lines only") {
		t.Errorf("no mode chosen: %v", err)
	}
	ed.sidesRadio.SetSelected("Cue lines")
	if !ed.cueBox.Visible() {
		t.Error("no cue length with cue lines")
	}
	check("export with sides")
	ed.cueWordsEntry.SetText("3")
	o := ed.job().options("pdf", "x.pdf")
	want := cli.Options{Colors: true, Tags: []string{"red", "London"}, Character: "EDWARD", Sides: "cue", CueWords: 3}
	if !o.Colors || !reflect.DeepEqual(o.Tags, want.Tags) || o.Character != want.Character || o.Sides != want.Sides || o.CueWords != want.CueWords {
		t.Errorf("options %+v", o)
	}
	if !strings.HasSuffix(ed.filenameEntry.Text, " - red, London - EDWARD sides."+exportFormatByLabel(ed.formatSelect.Selected).ext) {
		t.Errorf("file name %q", ed.filenameEntry.Text)
	}
	ed.scroll.ScrollToBottom()
	capture(t, w.fyneWindow.Canvas(), "export-filters.png")
	ed.cueRadio.SetSelected(cueLengths[0])
	if o := ed.job().options("pdf", "x.pdf"); o.CueWords != 0 {
		t.Errorf("whole speech: %d words", o.CueWords)
	}
	ed.cueWordsEntry.SetText("many")
	if err := ed.validateInputs(); err == nil {
		t.Error("a cue of \"many\" words")
	}
	ed.sidesRadio.SetSelected("Lines only")
	if ed.cueBox.Visible() || ed.job().sides != lex.SidesIsolated {
		t.Errorf("lines only: cue shown %v, %q", ed.cueBox.Visible(), ed.job().sides)
	}
	ed.characterSelect.SetSelected(noSides)
	if o := ed.job().options("pdf", "x.pdf"); o.Character != "" || o.Sides != "" {
		t.Errorf("sides left: %+v", o)
	}
	ed.cancel()
}

// One export for each option, through Lexington.
func TestExportFiltered(t *testing.T) {
	export := func(job exportJob) string {
		t.Helper()
		job.format = "default"
		out := filepath.Join(t.TempDir(), "s.html")
		if err := exportAs(filterScript, out, "html", job); err != nil {
			t.Fatal(err)
		}
		b, _ := os.ReadFile(out)
		return string(b)
	}
	has := func(what, page string, in, out []string) {
		t.Helper()
		for _, s := range in {
			if !strings.Contains(page, s) {
				t.Errorf("%s: no %q", what, s)
			}
		}
		for _, s := range out {
			if strings.Contains(page, s) {
				t.Errorf("%s: has %q", what, s)
			}
		}
	}
	has("plain", export(exportJob{}), []string{"INT. BARN", "EXT. FIELD", "Dinner."}, []string{`<div class="colorbar"`})
	red, _ := lex.ColorFor(lex.ColorLine, "red")
	has("colors", export(exportJob{colors: true}), []string{`<div class="colorbar"`, lex.HexOf(red)}, nil)
	has("tags", export(exportJob{tags: []string{"red"}}), []string{"EXT. FIELD"}, []string{"INT. BARN", "Dinner."})
	has("london", export(exportJob{tags: []string{"LONDON"}}), []string{"INT. BARN", "INT. KITCHEN"}, []string{"EXT. FIELD"})
	has("full", export(exportJob{character: "EDWARD", sides: lex.SidesFull}), []string{"INT. BARN", "It always rains"}, []string{"EXT. FIELD", "Dinner."})
	has("cue", export(exportJob{character: "EDWARD", sides: lex.SidesCue, cueWords: 2}), []string{"...the barn.", "Then we stay."}, []string{"It always rains", "EXT. FIELD"})
	has("isolated", export(exportJob{character: "EDWARD", sides: lex.SidesIsolated}), []string{"Is it raining?", "Then we stay."}, []string{"in the barn", "EXT. FIELD"})
	// the filters compose: London's scenes, then Anna's lines there
	has("both", export(exportJob{tags: []string{"London"}, character: "ANNA", sides: lex.SidesIsolated}), []string{"It always rains"}, []string{"The field is dry.", "Dinner.", "Is it raining?"})
}
