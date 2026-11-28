// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	lexreport "github.com/LaPingvino/lexington/report"
)

// reportScript is a locked script with an A-scene, an OMITTED scene, a
// scene written since the lock (no number yet), a cue under another name
// of its character, and cues in boneyard: the omitted scene's, the
// source's and the adaptation block's.
const reportScript = `Title: The Shed

INT. KOVAČ HOUSE - KITCHEN - DAY #1#

Anna makes tea.

ANNA
Is it still raining?

TOM (O.S.)
Like the end of the world.

EXT. GARDEN - NIGHT #1A#

TOM
Wait for me!

ANNIE (CONT'D)
I always do.

.OMITTED #2#

/* Omitted 2
INT. SHED - NIGHT #2#

CARL
Nobody sees me.
*/

INT. KOVAČ HOUSE - KITCHEN - NIGHT #3#

ANNA
Tom?

EXT. GARDEN - DAY

The garden in the sun.

/* Production
scene numbers: locked, 10A
*/

/* Adaptation
characters: ANNA = Annie
*/

/* Source
BORIS
Not in the script.
*/
`

func TestScriptReport(t *testing.T) {
	r, err := scriptReport(reportScript, exportJob{})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, s := range r.Scenes {
		ids = append(ids, s.ID())
	}
	// the scene after 3 gets its number as Export would print it
	if got := strings.Join(ids, " "); got != "1 1A 2 3 4" {
		t.Errorf("scenes %s", got)
	}
	if !r.Scenes[2].Omitted || r.Scenes[2].Eighths != 0 {
		t.Errorf("omitted scene %+v", r.Scenes[2])
	}
	var names []string
	for _, c := range r.Characters {
		names = append(names, c.Name)
	}
	if got := strings.Join(names, " "); got != "ANNA TOM" { // ANNIE is ANNA; CARL and BORIS are boneyard
		t.Errorf("characters %s", got)
	}
	if r.Characters[0].Speeches != 3 || strings.Join(r.Characters[0].Scenes, " ") != "1 1A 3" {
		t.Errorf("ANNA %+v", r.Characters[0])
	}
	if len(r.Locations) != 2 || r.Locations[0].Name != "KOVAČ HOUSE - KITCHEN" {
		t.Errorf("locations %+v", r.Locations)
	}
	// an unlocked script: its unnumbered scenes counted on, as Export's
	unlocked := strings.Replace(reportScript, "scene numbers: locked, 10A\n", "", 1)
	unlocked = strings.Replace(unlocked, " #1A#", "", 1)
	r, err = scriptReport(unlocked, exportJob{})
	if err != nil {
		t.Fatal(err)
	}
	ids = nil
	for _, s := range r.Scenes {
		ids = append(ids, s.ID())
	}
	if got := strings.Join(ids, " "); got != "1 2 2 3 4" { // count: on from the last whole number
		t.Errorf("unlocked scenes %s", got)
	}
}

// Sorting by a column: by its value (a scene's number by its place in the
// script, so 10 after 9 and 1A after 1), ties in the report's order.
func TestSortedTable(t *testing.T) {
	r := lexreport.Report{Scenes: []lexreport.Scene{
		{Order: 1, Number: "9", Eighths: 3},
		{Order: 2, Number: "9A", Eighths: 10},
		{Order: 3, Number: "10", Eighths: 3},
	}}
	tab, _ := r.Table(lexreport.KindScenes)
	col := func(t lexreport.Table, i int) string {
		var out []string
		for _, row := range t.Rows {
			out = append(out, row[i].String())
		}
		return strings.Join(out, " ")
	}
	if got := col(sortedTable(tab, "Scene", true), 1); got != "10 9A 9" {
		t.Errorf("by scene, reversed: %s", got)
	}
	if got := col(sortedTable(tab, "Length", false), 1); got != "9 10 9A" {
		t.Errorf("by length: %s", got)
	}
	if got := col(sortedTable(tab, "Script order", false), 1); got != "9 9A 10" {
		t.Errorf("own order: %s", got)
	}
}

// reportsWindow is a window with the report script, the reports dialog
// open at the window's size.
func reportsWindow(t *testing.T, size fyne.Size) (*MainWindow, *reportsDialog) {
	t.Helper()
	w, _ := scriptWindow(t, reportScript)
	w.fyneWindow.Resize(size)
	r, err := scriptReport(reportScript, w.layoutJob())
	if err != nil {
		t.Fatal(err)
	}
	rd := newReportsDialog(w, r)
	rd.dialog.Show()
	return w, rd
}

func TestReportsDialog(t *testing.T) {
	w, rd := reportsWindow(t, fyne.NewSize(1100, 650))
	if rd.narrow {
		t.Fatal("narrow on a desktop")
	}
	tab, ok := rd.body.Objects[0].(*widget.Table)
	if !ok {
		t.Fatalf("no table: %T", rd.body.Objects[0])
	}
	if rows, cols := tab.Length(); rows != 5 || cols != 7 {
		t.Errorf("scenes table %d×%d", rows, cols)
	}
	rd.kind.SetSelected("Cast")
	if got := rd.table.Rows[0][1].String(); got != "ANNA" {
		t.Errorf("cast 1 is %s", got)
	}
	if !strings.Contains(rd.summary.Text, "2 speaking characters") {
		t.Errorf("summary %q", rd.summary.Text)
	}
	rd.sortBy.SetSelected("Character")
	rd.reverse.OnTapped()
	if got := rd.table.Rows[0][1].String(); got != "TOM" {
		t.Errorf("by name reversed, first %s", got)
	}
	page, err := rd.reportPage()
	if err != nil || !strings.Contains(string(page), "<td>TOM</td>") {
		t.Errorf("page %v:\n%s", err, page)
	}
	if w.settings.GetString("report-kind") != lexreport.KindCast {
		t.Error("the report shown is not remembered")
	}
	// in the menu
	found := false
	for _, it := range w.fyneWindow.MainMenu().Items {
		for _, m := range it.Items {
			found = found || (it.Label == "Production" && m.Label == "Reports…")
		}
	}
	if !found {
		t.Error("no Production > Reports…")
	}
}

// On a phone the rows stack, and the dialog fits the screen.
func TestReportsDialogPhone(t *testing.T) {
	w, rd := reportsWindow(t, fyne.NewSize(380, 800))
	if !rd.narrow {
		t.Fatal("not narrow on a phone")
	}
	if _, ok := rd.body.Objects[0].(*container.Scroll); !ok {
		t.Fatalf("no stacked rows: %T", rd.body.Objects[0])
	}
	if top := w.fyneWindow.Canvas().Overlays().Top(); top == nil || top.MinSize().Width > 380 {
		t.Errorf("dialog %v wide", top.MinSize().Width)
	}
}

// TestReportsScreenshots renders each report at a desktop's size and a
// phone's (SHOT_DIR as in screenshot_test.go).
func TestReportsScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"wide", fyne.NewSize(1100, 650)}, {"phone", fyne.NewSize(380, 800)}} {
		w, rd := reportsWindow(t, size.size)
		for _, k := range reportKinds {
			rd.kind.SetSelected(k.label)
			w.fyneWindow.Content().Refresh()
			f, err := os.Create(filepath.Join(dir, "reports-"+k.kind+"-"+size.name+".png"))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
				t.Fatal(err)
			}
			f.Close()
		}
	}
}

// A script the PDF writer cannot print as written (Hebrew; Russian scene
// words): the reports are made all the same, or fail as an error.
func TestScriptReportOtherScripts(t *testing.T) {
	for _, c := range []struct{ lang, script string }{
		{"ru", "ИНТ. КУХНЯ - ДЕНЬ\n\nАННА\nПривет.\n\nНАТ. САД - НОЧЬ\n\nБОРИС\nПока.\n"},
		{"en", "INT. מטבח - DAY\n\n@שרה\nשלום, מה שלומך?\n\nEXT. גן - NIGHT\n\n@漢字\n你好。\n"},
	} {
		r, err := scriptReport(c.script, exportJob{scenes: sceneHeaders(c.lang)})
		if err != nil {
			t.Errorf("%s: %v", c.lang, err)
			continue
		}
		if len(r.Scenes) != 2 || len(r.Characters) != 2 || r.Scenes[0].Eighths == 0 {
			t.Errorf("%s: %+v", c.lang, r)
		}
	}
}
