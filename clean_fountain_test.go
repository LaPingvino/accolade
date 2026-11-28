// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/lexington/fountain"
)

// cleanScript has a bit of everything Export Clean Fountain takes out.
const cleanScript = `Title: Rain

# Act One

INT. HOUSE - DAY #1#

Anna waits. [[from: #:~:text=She waited]] [[slow]]

.OMITTED #2#

/* Omitted 2
EXT. GARDEN - DAY #2#

Wind.
*/

EXT. STREET - NIGHT #3#

She runs.

/* Characters
ANNA: the lead
*/

/* Notes
call Tom
*/
`

// cleanPrefsKept puts the boxes' settings back after the test.
func cleanPrefsKept(t *testing.T, w *MainWindow) {
	t.Helper()
	was := map[string]bool{}
	for _, k := range fountain.StripKeys {
		was[k.Key] = w.settings.GetBoolean(cleanPref(k.Key))
		w.settings.SetBoolean(cleanPref(k.Key), false)
	}
	t.Cleanup(func() {
		for k, v := range was {
			w.settings.SetBoolean(cleanPref(k), v)
		}
	})
}

// cleanWindow is a window with cleanScript saved as film.fountain, and
// unsaved changes in the editor.
func cleanWindow(t *testing.T) (*MainWindow, string) {
	t.Helper()
	w := draftsWindow(t)
	cleanPrefsKept(t, w)
	p := filepath.Join(t.TempDir(), "film.fountain")
	os.WriteFile(p, []byte(cleanScript), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	return w, p
}

// The dialog: the presets set the boxes, the boxes are remembered and
// name their preset; the copy is written as Lexington strips it, the
// script and its window left as they were.
func TestCleanFountainExport(t *testing.T) {
	w, p := cleanWindow(t)
	w.textEditor.SetText(strings.Replace(cleanScript, "She runs.\n", "She runs.\n\nShe stops.\n", 1)) // unsaved
	w.hasChanges = true
	c := w.newCleanFountain()
	if c.preset.Selected != "Clean" {
		t.Errorf("no boxes ticked: preset %q", c.preset.Selected)
	}
	c.preset.SetSelected("Writer's version")
	if o := c.options(); o != (fountain.StripOptions{SceneNumbers: true, Omitted: true}) {
		t.Errorf("writer's version ticks %+v", o)
	}
	c.checks[0].SetChecked(true) // notes too: no preset
	if c.preset.Selected != "" || !w.settings.GetBoolean(cleanPref("notes")) {
		t.Errorf("own choice: preset %q, notes remembered %v", c.preset.Selected, w.settings.GetBoolean(cleanPref("notes")))
	}
	if !strings.Contains(c.summary.Text, "of the script's") {
		t.Errorf("summary %q", c.summary.Text)
	}
	out := filepath.Join(t.TempDir(), w.cleanFileName())
	if filepath.Base(out) != "film - clean.fountain" {
		t.Errorf("named %s", filepath.Base(out))
	}
	if err := c.writeTo(out); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	want := "Title: Rain\n\n# Act One\n\nINT. HOUSE - DAY\n\nAnna waits.\n\nEXT. STREET - NIGHT\n\nShe runs.\n\nShe stops.\n"
	if string(got) != want {
		t.Errorf("copy:\n%s\nwant:\n%s", got, want)
	}
	if b, _ := os.ReadFile(p); string(b) != cleanScript || w.currentFile != p || !w.hasChanges {
		t.Errorf("the script changed: file %v, now %s, changes %v", string(b) != cleanScript, w.currentFile, w.hasChanges)
	}
	if err := c.writeTo(p); err == nil {
		t.Error("written over the script")
	}
	if b, _ := os.ReadFile(p); string(b) != cleanScript {
		t.Error("the script was overwritten")
	}
	if c.about.Text != "Your own choice of what goes." {
		t.Errorf("about %q", c.about.Text)
	}
	// with the notes in the middle, the copy is still the script's
	w.openMiddle("notes")
	if w.scriptInMain() {
		t.Fatal("the notes not in the middle")
	}
	defer w.closeMiddle("notes")
	if c3 := w.newCleanFountain(); c3.text != c.text {
		t.Errorf("the notes in front: the copy is of\n%s", c3.text)
	}
	// remembered: the next dialog has the same boxes
	if c2 := w.newCleanFountain(); c2.options() != c.options() {
		t.Errorf("next time %+v, was %+v", c2.options(), c.options())
	}
	// Clean only: nothing but Accolade's data goes
	c.preset.SetSelected("Clean")
	if err := c.writeTo(out); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(out)
	if s := string(got); strings.Contains(s, "/* Notes") || strings.Contains(s, "from:") || !strings.Contains(s, "[[slow]]") || !strings.Contains(s, "/* Omitted 2") {
		t.Errorf("clean:\n%s", s)
	}
}

// A script in Esperanto is written as standard Fountain: its headings
// forced, no language line.
func TestCleanFountainDialect(t *testing.T) {
	w := draftsWindow(t)
	cleanPrefsKept(t, w)
	p := filepath.Join(t.TempDir(), "film.eo.fountain")
	os.WriteFile(p, []byte(eoScript+"\n/* Notes\nnoto\n*/\n"), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	c := w.newCleanFountain()
	if w.cleanFileName() != "film - clean.fountain" {
		t.Errorf("named %s", w.cleanFileName())
	}
	got := w.cleanText(c.text, c.options())
	if want := standardFountain(eoScript, "eo"); got != want || !strings.HasPrefix(got, ".EN. DOMO") {
		t.Errorf("copy:\n%q\nwant:\n%q", got, want)
	}
}

// Export Clean Fountain is a screenplay's: in the File menu after
// Export…, not in a Markdown document's.
func TestCleanFountainCommand(t *testing.T) {
	w, _ := cleanWindow(t)
	c := cmd("file.exportClean")
	if c.when == nil || !c.when(w.cmdEnv()) {
		t.Fatal("no Export Clean Fountain for a screenplay")
	}
	w.markdown = true
	defer func() { w.markdown = false }()
	if c.when(w.cmdEnv()) {
		t.Error("Export Clean Fountain in a Markdown document")
	}
}

// Screenshots of the dialog on a desktop and a phone (SHOT_DIR=… go test
// -run TestCleanFountainScreenshots): clean-*.png.
func TestCleanFountainScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(dir, 0o755)
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"desktop", fyne.NewSize(1100, 760)}, {"phone", fyne.NewSize(380, 800)}} {
		w, _ := cleanWindow(t)
		w.fyneWindow.Resize(size.size)
		c := w.newCleanFountain()
		c.d.Show()
		capture(t, w.fyneWindow.Canvas(), "clean-"+size.name+".png")
		c.preset.SetSelected(presetLabel(fountain.StripPresets[2].Name))
		capture(t, w.fyneWindow.Canvas(), "clean-writer-"+size.name+".png")
		c.d.Hide()
	}
}
