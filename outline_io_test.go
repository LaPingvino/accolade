// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The outline round trip: Accolade's own OPML export, opened again,
// gives the same text, for every structure to start from (its note, its
// acts touching their synopses, the beats), the beats worked example
// with its title page, and a structure with scenes' headings, a forced
// one, a ## with no # above it, notes and a synopsis of two lines.
func TestOutlineRoundTrip(t *testing.T) {
	starts := sceneStarts("en")
	texts := map[string]string{
		"beats": structureBeats,
		"scenes": "Title: The Lighthouse\nCredit: Written by\nAuthor: Joop\n\n[[a note before it all]]\n\n## Prologue\n= Before the story.\n\n" +
			"# Act One\n= The keeper's daughter comes home.\n[[blue]]\n\n= Mara finds the letter,\n= and reads it twice.\n\n" +
			"INT. LAMP ROOM - NIGHT #1#\n= She reads it.\n\n= A beat still to write.\n\n[[ask Tomas about this]]\n\n.THE VOID\n\n# Act Two\n\n= The storm.\n",
	}
	for _, st := range structureTemplates {
		texts[st.name] = structureText(st)
	}
	for name, text := range texts {
		t.Run(name, func(t *testing.T) {
			data, err := outlineOPML(text, starts)
			if err != nil {
				t.Fatal(err)
			}
			back, err := scriptOfOutline(data, starts, true)
			if err != nil {
				t.Fatal(err)
			}
			if back != text {
				t.Errorf("back:\n%s\nwas:\n%s\nOPML:\n%s", back, text, data)
			}
		})
	}
}

// The scenes' own text is not in an outline: their headings and
// synopses are, the action and dialogue stay in the script; the blocks
// at the script's end and the boneyard are left out.
func TestOutlineLeavesTheScenesText(t *testing.T) {
	text := "# Act One\n= Home.\n\nINT. KITCHEN - DAY\n= Anna waits.\n\nRain on the window.\n\nANNA\nIt's late.\n\n= Then Tom comes.\n\n/* cut\n= not this\n*/\n\n/* Notes\nkeep\n*/\n"
	data, err := outlineOPML(text, sceneStarts("en"))
	if err != nil {
		t.Fatal(err)
	}
	back, _ := scriptOfOutline(data, sceneStarts("en"), true)
	want := "# Act One\n= Home.\n\nINT. KITCHEN - DAY\n= Anna waits.\n\n= Then Tom comes.\n"
	if back != want {
		t.Errorf("got:\n%q\nwant\n%q", back, want)
	}
	for _, not := range []string{"Rain", "ANNA", "not this", "keep"} {
		if strings.Contains(string(data), not) {
			t.Errorf("%q in the outline", not)
		}
	}
}

// File › Open opens an outline by its content too (Workflowy's export
// saved as .txt or .xml), as a new script not saved: Save never writes
// over the outline.
func TestOpenOutline(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	dir := t.TempDir()
	data, err := os.ReadFile(filepath.Join("internal", "opml", "testdata", "workflowy.opml"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "lighthouse.txt")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := w.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	got := w.textEditor.Text()
	if !strings.HasPrefix(got, "# The Lighthouse\n\n## Act One\n= The keeper's daughter comes home.") {
		t.Fatalf("opened:\n%s", got)
	}
	if w.currentFile != "" || !w.hasChanges || w.suggestedName != "lighthouse.fountain" {
		t.Errorf("file %q, changes %v, name %q", w.currentFile, w.hasChanges, w.suggestedName)
	}
	items := w.outlineItems(got)
	if len(items) != 6 || items[1].Synopsis != "The keeper's daughter comes home." || len(items[1].Loose) != 3 {
		t.Errorf("the outline: %+v", items)
	}
}

// A treatment: the title page, the sections as headings, the scenes'
// headings under them, synopses and beats as paragraphs, notes in
// italics; a colour's note, the scenes' text and Markdown's own marks
// left as text.
func TestTreatment(t *testing.T) {
	text := "Title: The Lighthouse\nCredit: Written by\nAuthor: Joop\n\n# Act One [[blue]]\n= The keeper's daughter comes home.\n\n= Mara finds the *letter* <and> [reads] it.\n\n[[ask Tomas]]\n\nINT. LAMP ROOM - NIGHT #1#\n= She reads it.\n\nRain.\n\n- not a list\n"
	md, title, tp := treatmentMarkdown(text, sceneStarts("en"))
	want := "# Act One\n\nThe keeper's daughter comes home.\n\nMara finds the *letter* \\<and\\> \\[reads\\] it.\n\n*ask Tomas*\n\n## INT. LAMP ROOM - NIGHT\n\nShe reads it.\n"
	if md != want {
		t.Errorf("got:\n%q\nwant\n%q", md, want)
	}
	if title != "The Lighthouse" || strings.Join(tp, "|") != "The Lighthouse||A treatment||Written by Joop" {
		t.Errorf("title %q, title page %q", title, tp)
	}
	dir := t.TempDir()
	for _, f := range []string{"pdf", "docx"} {
		out := filepath.Join(dir, "t."+f)
		if err := exportTreatment(text, sceneStarts("en"), out, f, ""); err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(out)
		if len(data) < 500 {
			t.Errorf("%s: %d bytes", f, len(data))
		}
	}
}

// Findable where one looks (Joop: "If you look for it, you need to be
// able to recognize it, don't hide it too well"): OPML named in the File
// menu's label, the Open dialog's files, the palette, :export opml, the
// Export dialog's formats and a help section titled OPML; Paste as
// Structure and the plain words (outline, treatment, Workflowy) find it
// too.
func TestOutlineFindable(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(structureBeats)
	if c := commands().byID["file.exportOutline"]; c == nil || !strings.Contains(c.label, "OPML") || !strings.Contains(c.label, "Workflowy") {
		t.Errorf("the File menu's label: %+v", c)
	}
	if !slices.Contains(openExtensions(), ".opml") {
		t.Error("Open does not offer .opml")
	}
	entries := w.paletteEntries(w.cmdEnv())
	for q, want := range map[string]string{
		"opml":               "Export Outline (OPML) — for Workflowy, OmniOutliner…",
		"workflowy":          "Export Outline (OPML) — for Workflowy, OmniOutliner…",
		"treatment":          "Export Treatment (PDF, Word)…",
		"paste as structure": "Paste as Structure",
	} {
		if m := paletteMatches(entries, q); len(m) == 0 || m[0].label != want {
			t.Errorf("the palette's %q: %v", q, m)
		}
	}
	if f := exportFormatByLabel("opml"); f.id != "opml" || !strings.Contains(f.label, "OPML") {
		t.Errorf("the Export dialog's format: %+v", f)
	}
	if err := w.exExport("opml"); err != nil {
		t.Errorf(":export opml: %v", err)
	}
	page, err := os.ReadFile(filepath.Join("docs", "help", "structure.md"))
	if err != nil || !strings.Contains(string(page), "\n## OPML") {
		t.Errorf("no help section titled OPML (%v)", err)
	}
}

// Edit › Paste as Structure: indented lines from an outliner go in at
// the cursor's section boundary as sections and beats, one undo step.
func TestPasteAsStructure(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	script := "Title: X\n\n# Act One\n= Home.\n\nINT. A - DAY\n\nA.\n\n# Act Three\n= The end.\n"
	w.textEditor.SetText(script)
	w.textEditor.SetCursorOffset(len([]rune(script[:strings.Index(script, "A.")])))
	w.clipboard().SetContent("- Act Two\n  \"The storm.\"\n  - The roof goes.\n  - Tomas is missing.\n")
	runCmd(w, "edit.pasteStructure")
	want := "Title: X\n\n# Act One\n= Home.\n\nINT. A - DAY\n\nA.\n\n# Act Two\n= The storm.\n\n= The roof goes.\n\n= Tomas is missing.\n\n# Act Three\n= The end.\n"
	if got := w.textEditor.Text(); got != want {
		t.Fatalf("pasted:\n%q\nwant\n%q", got, want)
	}
	w.textEditor.Undo()
	if got := w.textEditor.Text(); got != script {
		t.Errorf("one undo: %q", got)
	}
}

// The outline's ways in and out as seen (SHOT_DIR): the Export dialog on
// the outline format, an outline opened as a script with its outline and
// cards, Paste as Structure in the Edit menu's place, and the treatment's
// PDF (treatment.pdf, for a look at its pages).
func TestOutlineIOScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("SHOT_DIR not set")
	}
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(structureBeats)
	if err := w.exExport("opml"); err != nil {
		t.Fatal(err)
	}
	comboShot(t, w, "outline-export-opml")
	for _, o := range w.fyneWindow.Canvas().Overlays().List() {
		w.fyneWindow.Canvas().Overlays().Remove(o)
	}
	if err := w.LoadFile(filepath.Join("internal", "opml", "testdata", "omnioutliner.opml")); err != nil {
		t.Fatal(err)
	}
	w.togglePane("outline")
	comboShot(t, w, "outline-opened")
	runCmd(w, "view.cards")
	comboShot(t, w, "outline-opened-cards")
	if err := exportTreatment(structureBeats+"\nINT. LAMP ROOM - NIGHT\n= Mara reads the letter twice.\n\n[[ask: is the letter enough?]]\n", sceneStarts("en"), filepath.Join(dir, "treatment.pdf"), "pdf", ""); err != nil {
		t.Fatal(err)
	}
}

// Every beat the outline shows is in the outline exported, in a
// paragraph of its own or not: a beat with a note touching it, a note
// before a beat, a beat touching action.
func TestOutlineKeepsEveryBeat(t *testing.T) {
	starts := sceneStarts("en")
	text := "# Act One\n= Home.\n\n= A beat\n[[with a note]]\n\n[[a note]]\n= then a beat\n\nINT. A - DAY\n\nAction.\n= a beat touching action\n\n= Last.\n"
	data, err := outlineOPML(text, starts)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range outlineOf(text, starts) {
		for _, l := range it.Loose {
			if !strings.Contains(string(data), l.Text) {
				t.Errorf("beat %q not in the outline:\n%s", l.Text, data)
			}
		}
	}
	md, _, _ := treatmentMarkdown(text, starts)
	for _, want := range []string{"A beat\n\n*with a note*", "then a beat", "a beat touching action"} {
		if !strings.Contains(md, want) {
			t.Errorf("treatment without %q:\n%s", want, md)
		}
	}
}

// Opened again end to end: the Export dialog's job writes the outline,
// File > Open reads it back, the editor holds the same text; the
// treatment's PDF and Word through the same job.
func TestOutlineExportOpenEndToEnd(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(structureBeats)
	ed := NewExportDialog(w)
	dir := t.TempDir()
	out := filepath.Join(dir, "lighthouse.opml")
	if err := ed.performExportJob(structureBeats, out, "opml", ed.job()); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"treatment-pdf", "treatment-docx"} {
		p := filepath.Join(dir, f)
		if err := ed.performExportJob(structureBeats, p, f, ed.job()); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
	}
	w2, _ := newDialogTestWindow(t)
	if err := w2.LoadFile(out); err != nil {
		t.Fatal(err)
	}
	if got := w2.textEditor.Text(); got != structureBeats {
		t.Errorf("opened again:\n%q\nwas\n%q", got, structureBeats)
	}
}
