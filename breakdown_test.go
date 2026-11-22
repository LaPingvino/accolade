package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/lex"
	lexreport "github.com/LaPingvino/lexington/report"
)

const bdScript = `Title: The Shed

INT. KITCHEN - NIGHT

Anna puts the revolver on the table. A revolver again.

ANNA
Where is the taxi?

TOM
Outside.

EXT. STREET - NIGHT

The taxi waits. A dog barks at it.

INT. KITCHEN - NIGHT

The revolver is gone. Nobody speaks.
`

// selectWords selects the nth (from 0) words in the editor.
func selectWords(t *testing.T, w *MainWindow, words string, nth int) (int, int) {
	t.Helper()
	text := w.textEditor.Text()
	off := 0
	for k := 0; ; k++ {
		i := strings.Index(text[off:], words)
		if i < 0 {
			t.Fatalf("no %q #%d", words, nth)
		}
		if k == nth {
			start := len([]rune(text[:off+i]))
			end := start + len([]rune(words))
			w.textEditor.Select(start, end)
			return start, end
		}
		off += i + len(words)
	}
}

// tagNow tags the nth words as category.
func tagNow(t *testing.T, w *MainWindow, words string, nth int, category string) {
	t.Helper()
	selectWords(t, w, words, nth)
	w.tagSelectionAs(category)
}

// found are the tags as found in the text now: "words@line:col" (lost:
// "words@lost"), by element name.
func found(w *MainWindow) []string {
	w.updateBreakdown()
	_, bt, ps := w.breakdownNowFound()
	var out []string
	for _, p := range ps {
		if p.lost {
			out = append(out, p.element.Name+"@lost")
			continue
		}
		line, col := bt.lineCol(p.start)
		out = append(out, p.element.Name+"@"+string(bt.runes[p.start:p.end])+"@"+itoa(line)+":"+itoa(col))
	}
	return out
}

func itoa(n int) string { return fmt.Sprintf("%03d", n) }

// Tagging: the words and their anchor, one element for every place it
// recurs, the ink in the editor; tags stay on their words through edits
// elsewhere, in their scene and before it.
func TestTagAndAnchors(t *testing.T) {
	w, _ := scriptWindow(t, bdScript)
	tagNow(t, w, "revolver", 1, "Props") // the second in scene 1
	tagNow(t, w, "revolver", 2, "Prop")  // scene 3's; the same element
	tagNow(t, w, "taxi", 1, "Vehicles")
	f := w.breakdownNow()
	if len(f.Elements) != 2 || len(f.Tags) != 3 {
		t.Fatalf("elements %+v tags %+v", f.Elements, f.Tags)
	}
	if f.Tags[0].At != "text=the table. A-,revolver,-again. ANNA Where" || f.Tags[0].Heading != "int kitchen night" || f.Tags[0].Nth != 0 {
		t.Errorf("anchor %+v", f.Tags[0])
	}
	if f.Tags[1].Nth != 1 {
		t.Errorf("scene 3 is the second INT. KITCHEN - NIGHT: %+v", f.Tags[1])
	}
	want := "revolver@revolver@004:039 revolver@revolver@018:004 taxi@taxi@014:004"
	if got := strings.Join(found(w), " "); got != want {
		t.Errorf("found %s\nwant  %s", got, want)
	}
	inks := w.breakdownInksShown()
	if len(inks[4]) != 1 || inks[4][0].Start != 39 || inks[4][0].End != 47 {
		t.Errorf("inks %v", inks)
	}
	// tagging the same place again adds nothing
	tagNow(t, w, "revolver", 1, "Props")
	if len(w.breakdownNow().Tags) != 3 {
		t.Error("the same words tagged twice")
	}

	// edits: a word before the tag in its line, a scene before all, a
	// revolver more before it in its scene
	replaceText(t, w, "Anna puts", "Anna slowly puts")
	replaceText(t, w, "INT. KITCHEN - NIGHT", "EXT. YARD - DAY\n\nChickens.\n\nINT. KITCHEN - NIGHT")
	replaceText(t, w, "Anna slowly", "A revolver. Anna slowly")
	want = "revolver@revolver@008:058 revolver@revolver@022:004 taxi@taxi@018:004"
	if got := strings.Join(found(w), " "); got != want {
		t.Errorf("after edits %s\nwant         %s", got, want)
	}
	// locking the numbers after tagging: still found
	w.lockSceneNumbers()
	if !strings.Contains(w.textEditor.Text(), "INT. KITCHEN - NIGHT #2#") {
		t.Fatalf("not locked:\n%s", w.textEditor.Text())
	}
	if got := strings.Join(found(w), " "); got != want {
		t.Errorf("after the lock %s\nwant           %s", got, want)
	}
	// a tag made on a numbered heading's scene is kept by its number:
	// found when the heading changes
	tagNow(t, w, "dog", 0, "Animals")
	replaceText(t, w, "EXT. STREET - NIGHT", "EXT. ALLEY - NIGHT")
	if got := found(w); got[len(got)-1] != "dog@dog@018:018" {
		t.Errorf("after the heading changed: %v", got)
	}
}

// A tag whose words are gone is lost: listed, kept in the file, found
// again when they come back.
func TestLostTags(t *testing.T) {
	w, _ := scriptWindow(t, bdScript)
	tagNow(t, w, "dog", 0, "Animals")
	replaceText(t, w, "A dog barks at it.", "Silence.")
	if got := found(w); len(got) != 1 || got[0] != "dog@lost" {
		t.Fatalf("found %v", got)
	}
	if len(w.breakdownNow().Tags) != 1 {
		t.Error("the lost tag was dropped")
	}
	p := prodPane(t, w)
	p.pick(stageBreakdown)
	if !strings.Contains(p.count.Text, "1 lost tag") {
		t.Errorf("count %q", p.count.Text)
	}
	lost := false
	for _, l := range rowLabels(p) {
		lost = lost || strings.HasPrefix(l.Text, "Lost: “dog” (Animals)")
	}
	if !lost {
		t.Error("the lost tag is not listed")
	}
	replaceText(t, w, "Silence.", "A dog barks at it.")
	if got := found(w); len(got) != 1 || got[0] != "dog@dog@014:018" {
		t.Errorf("back: %v", got)
	}
}

// Tags are on the script's words: never in a source link's quote or a
// note, so a tag whose words went from the script is lost though a link
// above them still quotes them; tagging inside a link says why not.
func TestTagsSkipLinksAndNotes(t *testing.T) {
	script := `Title: The Square

EXT. MARKET SQUARE - DAY

[[from: "Not even a beggar near them."]] [[a beggar, cut?]]

The doors stand open. Not even a beggar near them.
`
	w, _ := scriptWindow(t, script)
	tagNow(t, w, "beggar", 0, "Extras") // the link's: nothing tagged
	if f := w.breakdownNow(); len(f.Tags) != 0 {
		t.Fatalf("tagged in a link: %+v", f.Tags)
	}
	if s := w.statusText(); !strings.Contains(s, "not a note's or a source link's") {
		t.Errorf("status %q", s)
	}
	tagNow(t, w, "beggar", 1, "Extras") // the note's
	if f := w.breakdownNow(); len(f.Tags) != 0 {
		t.Fatalf("tagged in a note: %+v", f.Tags)
	}
	tagNow(t, w, "beggar", 2, "Extras") // the action's
	if got := found(w); len(got) != 1 || got[0] != "beggar@beggar@006:033" {
		t.Fatalf("found %v", got)
	}
	if at := w.breakdownNow().Tags[0].At; at != "text=Not even a-,beggar,-near them" {
		t.Errorf("anchored %q", at)
	}
	replaceText(t, w, "Not even a beggar near them.\n", "Nobody.\n")
	if got := found(w); len(got) != 1 || got[0] != "beggar@lost" {
		t.Errorf("the words gone, the link still quoting them: %v", got)
	}
}

// A tag whose words left its scene is not moved to another place with
// the same word: it is lost. A scene with the same heading and the word
// put before its scene does not take it either.
func TestTagNotMoved(t *testing.T) {
	w, _ := scriptWindow(t, bdScript)
	tagNow(t, w, "revolver", 1, "Props")
	replaceText(t, w, "INT. KITCHEN - NIGHT", "INT. KITCHEN - NIGHT\n\nA revolver on the shelf.\n\nINT. KITCHEN - NIGHT")
	if got := found(w); len(got) != 1 || got[0] != "revolver@revolver@008:039" {
		t.Errorf("after a scene put before: %v", got)
	}
	replaceText(t, w, "Anna puts the revolver on the table. A revolver again.", "Anna waits.")
	if got := found(w); len(got) != 1 || got[0] != "revolver@lost" {
		t.Errorf("after its words went: %v (scene 3 and the new scene have the word)", got)
	}
}

// Untag takes the tag away (and its element, if it has no other).
func TestUntag(t *testing.T) {
	w, _ := scriptWindow(t, bdScript)
	tagNow(t, w, "taxi", 0, "Vehicles")
	tagNow(t, w, "taxi", 1, "Vehicles")
	start, _ := selectWords(t, w, "taxi", 0)
	w.textEditor.Select(start+1, start+1) // the cursor in the word
	w.untagSelection()
	f := w.breakdownNow()
	if len(f.Tags) != 1 || len(f.Elements) != 1 {
		t.Fatalf("after one untag %+v", f)
	}
	selectWords(t, w, "taxi", 1)
	w.untagSelection()
	if f := w.breakdownNow(); len(f.Tags) != 0 || len(f.Elements) != 0 {
		t.Errorf("after both %+v", f)
	}
}

// The file: assets/production/breakdown.json in the kept copy of a plain
// script, read again by another window.
func TestBreakdownStored(t *testing.T) {
	w, p := scriptWindow(t, bdScript)
	tagNow(t, w, "revolver", 0, "Props")
	h := w.extrasHome()
	if h.kind != extrasKept {
		t.Fatalf("home %v", h.kind)
	}
	data, err := h.readExtra(breakdownName)
	if err != nil {
		t.Fatal(err)
	}
	var f breakdownFile
	if err := json.Unmarshal(data, &f); err != nil || f.Version != 1 || len(f.Tags) != 1 || f.Elements[0].Category != "Props" {
		t.Fatalf("%v %s", err, data)
	}
	// another script: its own (none); this one again: the tag
	w.NewFile()
	if len(w.breakdownNow().Tags) != 0 {
		t.Error("a new script has the last one's tags")
	}
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if got := found(w); len(got) != 1 || !strings.HasPrefix(got[0], "revolver@revolver@") {
		t.Errorf("read again: %v", got)
	}
}

// A breakdown.json that cannot be read is never written over.
func TestBreakdownUnreadable(t *testing.T) {
	w, _ := scriptWindow(t, bdScript)
	tagNow(t, w, "dog", 0, "Animals") // the kept copy made
	h := w.extrasHome()
	if err := h.dir.write([]bundleFile{{breakdownName, []byte("{not json")}}); err != nil {
		t.Fatal(err)
	}
	w.breakdownGen++ // read again
	tagNow(t, w, "taxi", 0, "Vehicles")
	if data, _ := h.readExtra(breakdownName); string(data) != "{not json" {
		t.Errorf("written over: %s", data)
	}
}

// In an .accolade the file is in the bundle.
func TestBreakdownInBundle(t *testing.T) {
	w := draftsWindow(t)
	p := filepath.Join(t.TempDir(), "Script.accolade")
	os.WriteFile(p, makeZip(t, bundleItems("Script.textbundle/")), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	words := strings.Fields(w.textEditor.Text())
	if len(words) < 6 {
		t.Fatalf("text %q", w.textEditor.Text())
	}
	// a word of the bundle's script, in a scene
	_, bt, _ := w.breakdownNowFound()
	if len(bt.scenes) == 0 {
		t.Skip("the bundle's script has no scene")
	}
	wd := bt.words[len(bt.words)-1]
	w.textEditor.Select(wd.start, wd.end)
	w.tagSelectionAs("Props")
	b, _, err := readBundle(p)
	if err != nil {
		t.Fatal(err)
	}
	if data, err := b.readExtra(breakdownName); err != nil || !strings.Contains(string(data), `"Props"`) {
		t.Errorf("in the bundle: %v %s", err, data)
	}
}

// The Break down stage: the count, the scene's cast (from its cues, and
// tagged), its location, its elements; Next Scene and Mark Scene Done.
func TestBreakdownStage(t *testing.T) {
	w, _ := scriptWindow(t, bdScript)
	p := prodPane(t, w)
	test.Tap(p.stages[stageBreakdown])
	if p.view() != stageBreakdown {
		t.Fatal("not shown")
	}
	if got := p.count.Text; got != "0% broken down · 3 scenes to go" {
		t.Errorf("count %q", got)
	}
	// Next Scene: the first, with its cast and location
	test.Tap(p.bd.next)
	if line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset()); line != 2 {
		t.Errorf("Next went to line %d", line)
	}
	labels := func() string {
		var out []string
		for _, l := range rowLabels(p) {
			out = append(out, l.Text)
		}
		return strings.Join(out, " | ")
	}
	got := labels()
	for _, want := range []string{"INT. KITCHEN - NIGHT · ", "Location: KITCHEN (INT, NIGHT)", "1 ANNA (speaks)", "2 TOM (speaks)"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in %s", want, got)
		}
	}
	tagNow(t, w, "revolver", 0, "Props")
	tagNow(t, w, "Anna", 0, "Cast Members") // speaks: not twice
	w.textEditor.Select(0, 0)
	w.goToLine(4)
	p.refresh()
	got = labels()
	if !strings.Contains(got, "revolver") || strings.Count(got, "ANNA") != 1 {
		t.Errorf("after tagging: %s", got)
	}
	// Mark Scene Done: 33%, Next goes to the second
	test.Tap(p.bd.done)
	if got := p.count.Text; got != "33% broken down · 2 scenes to go" {
		t.Errorf("count %q", got)
	}
	if p.bd.done.Text != "Mark Scene Not Done" {
		t.Errorf("done %q", p.bd.done.Text)
	}
	test.Tap(p.bd.next)
	if line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset()); line != 12 {
		t.Errorf("Next went to line %d", line)
	}
	// scene 3 has a revolver tagged elsewhere: suggested; a tap tags it
	w.goToLine(18)
	p.refresh()
	var sugg *tapLabel
	for _, l := range rowLabels(p) {
		if strings.HasPrefix(l.Text, "Tag “revolver” here") {
			sugg = l
		}
	}
	if sugg == nil {
		t.Fatalf("no suggestion in %s", labels())
	}
	sugg.Tapped(nil)
	if n := len(w.breakdownNow().Tags); n != 3 {
		t.Errorf("%d tags after the suggestion", n)
	}
	// a silent cast member in scene 3
	tagNow(t, w, "Nobody", 0, "Cast Members")
	w.goToLine(18)
	p.refresh()
	if got := labels(); !strings.Contains(got, "3 NOBODY") {
		t.Errorf("silent cast: %s", got)
	}
	// all done
	for _, line := range []int{12, 18} {
		w.goToLine(line)
		p.refresh()
		test.Tap(p.bd.done)
	}
	if got := p.count.Text; got != "100% broken down · 3 scenes" || !p.bd.next.Disabled() {
		t.Errorf("all done: %q", got)
	}
}

// Breakdown sheets and the element report, as the Reports dialog has
// them; the sheets' PDF.
func TestBreakdownReports(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, bdScript)
	tagNow(t, w, "revolver", 0, "Props")
	tagNow(t, w, "taxi", 1, "Vehicles")
	tagNow(t, w, "dog", 0, "Animals")
	tagNow(t, w, "Nobody", 0, "Cast Members")
	t.Cleanup(func() { w.settings.SetString("report-kind", lexreport.KindScenes) })
	w.showReportsOf(lexreport.KindSheets)
	waitFor(t, "the reports", func() bool { return w.lastReports != nil })
	rd := w.lastReports
	if rd.kindOf() != lexreport.KindSheets {
		t.Fatalf("kind %s", rd.kindOf())
	}
	var b bytes.Buffer
	if err := rd.table.WriteCSV(&b); err != nil {
		t.Fatal(err)
	}
	csv := strings.ReplaceAll(b.String(), "\r\n", "\n")
	for _, want := range []string{
		"1,1,INT,KITCHEN,NIGHT,1,", "1 ANNA; 2 TOM,,,,revolver,",
		"2,2,EXT,STREET,NIGHT,1,", ",taxi,", ",dog,",
		"3,3,INT,KITCHEN,NIGHT,1,", ",3 NOBODY,",
	} {
		if !strings.Contains(csv, want) {
			t.Errorf("no %q in\n%s", want, csv)
		}
	}
	if !strings.Contains(rd.summary.Text, "Breakdown: 4 tags in 3 scenes") {
		t.Errorf("summary %q", rd.summary.Text)
	}
	rd.kind.SetSelected("Elements")
	b.Reset()
	rd.table.WriteCSV(&b)
	els := strings.ReplaceAll(b.String(), "\r\n", "\n")
	for _, want := range []string{"Cast Members,ANNA,1,1,1,", "Cast Members,NOBODY,3,1,3,", "Vehicles,taxi,,1,2,", "Props,revolver,,1,1,"} {
		if !strings.Contains(els, want) {
			t.Errorf("no %q in\n%s", want, els)
		}
	}
	rep, _ := scriptReport(w.textEditor.Text(), w.layoutJob())
	f, bt, ps := w.breakdownNowFound()
	b.Reset()
	if err := lexreport.WriteSheetsPDF(&b, "The Shed", rep.Sheets(reportBreakdownFor(f, ps, bt, &rep)), ""); err != nil || !bytes.HasPrefix(b.Bytes(), []byte("%PDF")) {
		t.Errorf("sheets PDF: %v", err)
	}
	if dir := os.Getenv("SHOT_DIR"); dir != "" {
		os.WriteFile(filepath.Join(dir, "breakdown-sheets.pdf"), b.Bytes(), 0o644)
	}
	rd.dialog.Hide()
}

// The text's scenes are the report's, with an OMITTED scene (its old
// heading in boneyard), a forced heading and a title page: each tag on
// its scene's sheet.
func TestBreakdownScenesAlign(t *testing.T) {
	script := "Title: X\n\nINT. HOUSE - DAY #1#\n\nA lamp.\n\n.OMITTED #2#\n\n/* Omitted 2\nINT. SHED - NIGHT #2#\n\nA rake.\n*/\n\n" +
		".THE VOID #3#\n\nA star.\n\nEXT. ROAD - NIGHT #4#\n\nA truck.\n\n/* Production\nscene numbers: locked, 10A\n*/\n"
	w, _ := scriptWindow(t, script)
	tagNow(t, w, "lamp", 0, "Props")
	tagNow(t, w, "star", 0, "Props")
	tagNow(t, w, "truck", 0, "Vehicles")
	rep, err := scriptReport(w.textEditor.Text(), w.layoutJob())
	if err != nil {
		t.Fatal(err)
	}
	f, bt, ps := w.breakdownNowFound()
	if len(bt.scenes) != len(rep.Scenes) {
		t.Errorf("%d scenes in the text, %d in the report", len(bt.scenes), len(rep.Scenes))
	}
	got := map[string]string{}
	for _, sh := range rep.Sheets(reportBreakdownFor(f, ps, bt, &rep)) {
		for _, g := range sh.Groups {
			got[sh.ID()] += strings.Join(g.Names, ",")
		}
	}
	if got["1"] != "lamp" || got["3"] != "star" || got["4"] != "truck" || got["2"] != "" {
		t.Errorf("sheets %v", got)
	}
	// and when they differ, by their numbers
	short := rep
	short.Scenes = append([]lexreport.Scene{}, rep.Scenes[:1]...)
	short.Scenes = append(short.Scenes, rep.Scenes[2:]...)
	if m := reportScenes(bt, &short); m[0] != 0 || m[1] != -1 || m[2] != 1 || m[3] != 2 {
		t.Errorf("by number %v", m)
	}
}

// FDX out: the scenes' lengths, the TagData with the tags and the cast
// from the cues; read back in, the same tags on the same words.
func TestBreakdownFDX(t *testing.T) {
	w, _ := scriptWindow(t, bdScript)
	tagNow(t, w, "revolver", 1, "Props")
	tagNow(t, w, "dog", 0, "Animals")
	tagNow(t, w, "dog", 0, "Puppets") // a category of one's own, the same words
	text := w.textEditor.Text()
	j := w.withBreakdown(w.layoutJob(), text)
	out := filepath.Join(t.TempDir(), "shed.fdx")
	if err := cli.Write(t.Context(), j.script(text), j.options("fdx", out)); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(out)
	fdxText := string(data)
	for _, want := range []string{"<TagData>", `Name="Props" Number="6"`, `Name="Puppets"`, `Label="revolver"`,
		`Label="ANNA"`, `Label="TOM"`, `<Text TagNumber=`, `<SceneProperties Length="`, `Page="1"`,
		// what Movie Magic reads of an FDX: the headings and who speaks
		`<Paragraph Type="Scene Heading">`, `<Text>INT. KITCHEN - NIGHT</Text>`, `<Paragraph Type="Character">`} {
		if !strings.Contains(fdxText, want) {
			t.Errorf("no %s in the FDX", want)
		}
	}
	// in again: the tags on their words (the cast from the cues too)
	text2, _, _, tags, err := importFDXTagged(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	bt := readBdText(text2, nil)
	got := map[string]string{}
	for _, p := range bt.place(&tags) {
		if p.lost {
			t.Errorf("lost %+v", p.tag)
			continue
		}
		line, _ := bt.lineCol(p.start)
		l := strings.TrimSpace(strings.Split(text2, "\n")[line])
		got[p.element.Category+":"+p.element.Name] += l[:min(len(l), 8)] + ";"
	}
	for k, v := range map[string]string{"Props:revolver": "Anna put;", "Animals:dog": "The taxi;", "Puppets:dog": "The taxi;",
		"Cast Members:ANNA": "ANNA;", "Cast Members:TOM": "TOM;"} {
		if got[k] != v {
			t.Errorf("%s: %q (all %v)", k, got[k], got)
		}
	}
	// imported into a window: kept
	p := filepath.Join(t.TempDir(), "shed.fdx")
	os.WriteFile(p, data, 0o644)
	w2, _ := scriptWindow(t, "")
	if err := w2.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if n := len(w2.breakdownNow().Tags); n < 3 {
		t.Errorf("%d tags imported", n)
	}
}

// Tags on the second line of a two-line action paragraph and of a
// two-line speech go out on their paragraph's runs and come back on their
// words.
func TestBreakdownFDXMultiLine(t *testing.T) {
	script := "INT. HOUSE - DAY\n\nThe taxi waits.\nA dog barks at it.\n\nANNA\nWhere is the taxi?\nAnd my umbrella?\n"
	w, _ := scriptWindow(t, script)
	tagNow(t, w, "dog", 0, "Animals")
	tagNow(t, w, "umbrella", 0, "Props")
	text := w.textEditor.Text()
	j := w.withBreakdown(w.layoutJob(), text)
	out := filepath.Join(t.TempDir(), "multi.fdx")
	if err := cli.Write(t.Context(), j.script(text), j.options("fdx", out)); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(out)
	for _, want := range []string{">dog</Text>", ">umbrella</Text>"} {
		if !strings.Contains(string(data), "TagNumber") || !regexpMatch(`<Text TagNumber="\d+"`+want, string(data)) {
			t.Errorf("no tagged %s in\n%s", want, data)
		}
	}
	text2, _, _, tags, err := importFDXTagged(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	bt := readBdText(text2, nil)
	var got []string
	for _, p := range bt.place(&tags) {
		if !p.lost && p.element.Category != lex.CastCategory {
			got = append(got, string(bt.runes[p.start:p.end]))
		}
	}
	slices.Sort(got)
	if strings.Join(got, ",") != "dog,umbrella" {
		t.Errorf("read back %v from\n%s", got, text2)
	}
}

func regexpMatch(re, s string) bool { return regexp.MustCompile(re).MatchString(s) }

// A bundle whose only extra is its breakdown, saved as a plain script,
// keeps it (the kept copy); saved as .accolade again, it goes back in.
func TestBreakdownFlattened(t *testing.T) {
	w := draftsWindow(t)
	p := filepath.Join(t.TempDir(), "Script.accolade")
	os.WriteFile(p, makeZip(t, bundleItems("Script.textbundle/")), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	_, bt, _ := w.breakdownNowFound()
	if len(bt.scenes) == 0 {
		t.Fatal("the bundle's script has no scene")
	}
	wd := bt.words[len(bt.words)-1]
	word := string(bt.runes[wd.start:wd.end])
	w.textEditor.Select(wd.start, wd.end)
	w.tagSelectionAs("Props")
	to := filepath.Join(filepath.Dir(p), "Script.fountain")
	saveAsPlain(t, w, to, flattenKept)
	w2 := againWindow(t, w)
	if err := w2.LoadFile(to); err != nil {
		t.Fatal(err)
	}
	if got := found(w2); len(got) != 1 || !strings.HasPrefix(got[0], word+"@") {
		t.Fatalf("flattened: %v", got)
	}
	back := filepath.Join(filepath.Dir(p), "Back.accolade")
	if err := w2.saveToFile(back); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(back)
	if got, _ := zipFile(t, data, "Back.textbundle/"+breakdownName); !strings.Contains(got, `"Props"`) {
		t.Errorf("back in: %q", got)
	}
}

// Beat's tags: ranges in UTF-16 units of the text before its block (an
// emoji before them counts two), read while its Text Length is the
// text's; read once.
func TestBeatTags(t *testing.T) {
	body := "INT. CAFÉ - DAY\n\nZoë 🙂 drops the cup.\n\nEXT. ROAD - NIGHT\n\nA truck.\n"
	units := func(s string) int {
		n := 0
		for _, r := range s {
			n++
			if r > 0xffff {
				n++
			}
		}
		return n
	}
	cup := units(body[:strings.Index(body, "cup")])
	truck := units(body[:strings.Index(body, "truck")])
	block := func(length int) string {
		return "\n" + beatMarker + ` {"Text Length":` + itoaPlain(length) + `,"Tags":[{"range":[` + itoaPlain(cup) + `,3],"type":"prop","definition":"d1"},` +
			`{"range":[` + itoaPlain(truck) + `,5],"type":"vehicle","definition":"d2"}],"TagDefinitions":[{"type":"prop","name":"cup","id":"d1"},` +
			`{"type":"vehicle","name":"truck","id":"d2"}]} ` + beatEnd + "\n"
	}
	f, n, why := beatBreakdown(body+block(units(body)), nil)
	if why != "" || n != 2 {
		t.Fatalf("%d tags, %q", n, why)
	}
	bt := readBdText(body, nil)
	var got []string
	for _, p := range bt.place(&f) {
		got = append(got, p.element.Category+":"+string(bt.runes[p.start:p.end]))
	}
	if strings.Join(got, ",") != "Props:cup,Vehicles:truck" {
		t.Errorf("found %v", got)
	}
	if _, _, why := beatBreakdown(body+"More.\n"+block(units(body)), nil); why == "" {
		t.Error("read though the text changed")
	}
	// opened: read, and once only
	w, p := scriptWindow(t, body+block(units(body)))
	if n := len(w.breakdownNow().Tags); n != 2 {
		t.Fatalf("%d tags", n)
	}
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if n := len(w.breakdownNow().Tags); n != 2 {
		t.Errorf("read twice: %d tags", n)
	}
}

func itoaPlain(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// The category picker: the categories (the last used first), a filter,
// a new category of one's own.
func TestTagPicker(t *testing.T) {
	w, _ := scriptWindow(t, bdScript)
	selectWords(t, w, "dog", 0)
	w.showTagPicker()
	tp := topTagPicker(t, w)
	if len(tp.list.Objects) < 20 {
		t.Errorf("%d categories", len(tp.list.Objects))
	}
	tp.filter.SetText("anim")
	if len(tp.list.Objects) != 3 { // Animals, Animal Wrangler, New category “anim”
		t.Errorf("%d for anim", len(tp.list.Objects))
	}
	tp.filter.SetText("Dog Trainer")
	tp.pickFirst() // the new category
	f := w.breakdownNow()
	if len(f.Categories) != 1 || f.Categories[0].Name != "Dog Trainer" || len(f.Tags) != 1 {
		t.Fatalf("%+v", f)
	}
	selectWords(t, w, "taxi", 0)
	w.showTagPicker()
	tp = topTagPicker(t, w)
	if b := tp.list.Objects[0].(*fyne.Container).Objects[0].(*widget.Button); b.Text != "Dog Trainer" {
		t.Errorf("the last used is not first: %s", b.Text)
	}
}

func topTagPicker(t *testing.T, w *MainWindow) *tagPicker {
	t.Helper()
	if w.lastPicker == nil {
		t.Fatal("no picker")
	}
	return w.lastPicker
}

// The FDX's categories are Final Draft's.
func TestCategoriesAreFinalDraftsAndOSFs(t *testing.T) {
	c, ok := lex.TagCategoryOf("Props")
	if !ok || c.ID != "05c556eb-6bc1-4a3a-b09f-f8b5ba1b6afa" || c.Color != "#FF0000" {
		t.Errorf("%+v", c)
	}
}

// SHOT_DIR renders a tagged script with the Break down stage, and the
// breakdown sheets in the Reports dialog, wide and on a phone.
func TestBreakdownScreenshots(t *testing.T) {
	holdUI(t)
	if os.Getenv("SHOT_DIR") == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	w, _ := scriptWindow(t, bdScript)
	tagNow(t, w, "revolver", 0, "Props")
	tagNow(t, w, "revolver", 1, "Props")
	tagNow(t, w, "table", 0, "Set Dressing")
	tagNow(t, w, "taxi", 1, "Vehicles")
	tagNow(t, w, "dog", 0, "Animals")
	tagNow(t, w, "Nobody", 0, "Cast Members")
	p := prodPane(t, w)
	p.pick(stageBreakdown)
	w.goToLine(13)
	p.refresh()
	test.Tap(p.bd.done) // scene 2 done
	w.goToLine(4)       // scene 1: cast, location, elements
	w.updateBreakdown()
	p.refresh()
	comboShot(t, w, "breakdown-stage")
	w.app.setColorScheme("light")
	t.Cleanup(func() { w.app.setColorScheme("dark") })
	w.updateBreakdown()
	p.refresh()
	comboShot(t, w, "breakdown-stage-light")

	t.Cleanup(func() { w.settings.SetString("report-kind", lexreport.KindScenes) })
	// the reports made for the window's size, as a phone opens them
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"wide", fyne.NewSize(1100, 650)}, {"phone", fyne.NewSize(380, 800)}} {
		w.fyneWindow.Resize(size.size)
		w.lastReports = nil
		w.showReportsOf(lexreport.KindSheets)
		waitFor(t, "the reports", func() bool { return w.lastReports != nil })
		f, err := os.Create(filepath.Join(os.Getenv("SHOT_DIR"), "breakdown-sheets-"+size.name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, w.fyneWindow.Canvas().Capture())
		f.Close()
		w.lastReports.dialog.Hide()
	}
}

// On a phone the Break down stage fits the pane: its buttons one under
// the other at the list's end.
func TestBreakdownStageNarrow(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, bdScript)
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	p := prodPane(t, w)
	p.pick(stageBreakdown)
	if p.box.MinSize().Width > sourceWidth {
		t.Errorf("the pane is at least %v wide", p.box.MinSize().Width)
	}
	if len(p.buttons.Objects) != 4 || !p.narrow {
		t.Errorf("%d buttons, narrow %v", len(p.buttons.Objects), p.narrow)
	}
	w.fyneWindow.Resize(fyne.NewSize(1100, 650))
	w.fyneWindow.Content().Refresh()
	p.refresh()
	if len(p.buttons.Objects) != 1 || p.buttons.Objects[0] != p.bd.grid {
		t.Error("beside the script the buttons are not in two columns")
	}
}
