// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/fountain"
)

// The pattern fixer's core on made-up look-alikes of imported scripts
// (never the benchmarks themselves: they are not ours to copy).

var fixEnvEN = fixEnv{scenes: []string{"INT", "EXT", "EST", "INT./EXT", "INT/EXT", "I/E"}}

// apply1 is text with one rule applied, and how many places it changed.
func apply1(t *testing.T, text string, r fixRule) (string, int) {
	t.Helper()
	f := splitForFix(text)
	cs, err := f.changes(r, fixEnvEN)
	if err != nil {
		t.Fatal(err)
	}
	return f.apply(cs).text(), len(cs)
}

// A typescript's numbered headings, "Scene N - …", one pattern for all.
func TestFixNumberedSceneHeadings(t *testing.T) {
	text := "Title: The Kite\n\nScene 3 - int. kitchen, morning\n\nPia waits.\n\nScene 4 - ext. & int.\n\nShe runs.\n\nScene 5 - on the roof & elsewhere -\nThe wind.\n"
	f := splitForFix(text)
	r := proposeRule("Scene 3 - int. kitchen, morning", f)
	if r.Action != fixHeading || r.Pattern != "Scene <number> - <text>" || r.Template != "<text> #<number>#" {
		t.Fatalf("proposed %+v", r)
	}
	ms, err := f.matches(r)
	if err != nil || len(ms) != 3 {
		t.Fatalf("matches %v %v", ms, err)
	}
	got, n := apply1(t, text, r)
	want := "Title: The Kite\n\nINT. KITCHEN, MORNING #3#\n\nPia waits.\n\nEXT. & INT. #4#\n\nShe runs.\n\n.ON THE ROOF & ELSEWHERE #5#\n\nThe wind.\n"
	if got != want || n != 3 {
		t.Errorf("got %d changes:\n%s\nwant:\n%s", n, got, want)
	}
	// applied again, nothing changes: the rule's lines are headings now
	if again, n := apply1(t, got, r); n != 0 || again != got {
		t.Errorf("applied twice: %d changes", n)
	}
}

// "3. EXT – STREET – DAY": a number, the scene word, the rest.
func TestFixNumberedIntExt(t *testing.T) {
	text := "3. EXT – MARKET SQUARE – DAY\n\nCROWDS.\n\n4. INT – BAKERY – NIGHT\n\nBREAD.\n"
	r := proposeRule("3. EXT – MARKET SQUARE – DAY", splitForFix(text))
	if r.Pattern != "<number>. <int/ext> – <text>" || r.Action != fixHeading {
		t.Fatalf("proposed %+v", r)
	}
	got, _ := apply1(t, text, r)
	if !strings.Contains(got, "EXT. MARKET SQUARE – DAY #3#\n") || !strings.Contains(got, "INT. BAKERY – NIGHT #4#\n") {
		t.Errorf("got\n%s", got)
	}
}

// A name glued to its parenthetical becomes a cue and a parenthetical.
func TestFixGluedCue(t *testing.T) {
	text := "Rain.\n\nTHE GUARD(quietly)\nWho goes there?\n\nTHE GUARD (V.O.)\nStop.\n"
	f := splitForFix(text)
	r := proposeRule("THE GUARD(quietly)", f)
	if r.Action != fixSplitCue || r.Pattern != "<NAME>(<text>)" {
		t.Fatalf("proposed %+v", r)
	}
	got, n := apply1(t, text, r)
	want := "Rain.\n\nTHE GUARD\n(quietly)\nWho goes there?\n\nTHE GUARD (V.O.)\nStop.\n"
	if got != want || n != 1 {
		t.Errorf("got %d:\n%q", n, got)
	}
}

// Centred names with a blank line before the speech (how a translated
// script set them): cues, the speech's centring taken off.
func TestFixCentredCues(t *testing.T) {
	text := "THE TWO WALK ON.\n\n>MIRA<\n\nWhere now?\n\n>TOMAS<\n\n>Home.<\n\n!(LAUGHS)\n\n>MIRA<\n!(SMILES)\n\nGood.\n\n>(IN SPANISH) WE GO HOME TOMORROW, ALL OF US.<\n"
	f := splitForFix(text)
	r := proposeRule(">MIRA<", f)
	if r.Action != fixCue || r.Pattern != "><NAME><" {
		t.Fatalf("proposed %+v", r)
	}
	got, n := apply1(t, text, r)
	want := "THE TWO WALK ON.\n\nMIRA\nWhere now?\n\nTOMAS\nHome.\n\n!(LAUGHS)\n\nMIRA\n(SMILES)\nGood.\n\n>(IN SPANISH) WE GO HOME TOMORROW, ALL OF US.<\n"
	if got != want || n != 3 {
		t.Errorf("got %d:\n%q\nwant\n%q", n, got, want)
	}
}

// A double-spaced typescript: each paragraph's lines joined again.
func TestFixJoinAcrossBlankLines(t *testing.T) {
	text := "INT. ROOM - DAY\n\nThe clock on the wall ticks\n\nslowly, and nobody looks\n\nat it.\n\nPIA\nHello.\n\nA door\n\nOPENS.\n"
	f := splitForFix(text)
	r := proposeRule("The clock on the wall ticks", f)
	if r.Action != fixJoin || r.Pattern != "<unfinished>" {
		t.Fatalf("proposed %+v", r)
	}
	got, n := apply1(t, text, r)
	want := "INT. ROOM - DAY\n\nThe clock on the wall ticks slowly, and nobody looks at it.\n\nPIA\nHello.\n\nA door OPENS.\n"
	if got != want || n != 2 {
		t.Errorf("got %d:\n%q", n, got)
	}
}

// Page numbers and (MORE) dropped, one blank line left between.
func TestFixDropDebris(t *testing.T) {
	text := "Pia waits.\n\n12.\n\nShe runs.\n\n13.\n\nShe stops.\n"
	f := splitForFix(text)
	r := proposeRule("12.", f)
	if r.Action != fixDrop || r.Pattern != "<number>." {
		t.Fatalf("proposed %+v", r)
	}
	got, n := apply1(t, text, r)
	if want := "Pia waits.\n\nShe runs.\n\nShe stops.\n"; got != want || n != 2 {
		t.Errorf("got %d:\n%q", n, got)
	}
	if r := proposeRule("(MORE)", f); r.Action != fixDrop || r.Pattern != "(MORE)" {
		t.Errorf("(MORE): %+v", r)
	}
}

// Words spread by a justified typescript's spaces, made one space apart.
func TestFixCollapseSpaces(t *testing.T) {
	text := "She   looks    out at\nthe  sea.\n"
	r := proposeRule("She   looks    out at", splitForFix(text))
	if r.Action != fixSpaces {
		t.Fatalf("proposed %+v", r)
	}
	if got, n := apply1(t, text, r); got != "She looks out at\nthe sea.\n" || n != 2 {
		t.Errorf("got %d %q", n, got)
	}
}

// Words broken at line ends, joined as the script's own words say.
func TestFixHyphens(t *testing.T) {
	text := "The sol- diers wait. More soldiers come.\n\nA self- portrait of the portrait painter, in a quarter-\nmile line.\n"
	r := fixRule{Action: fixHyphens, Pattern: "<text>"}
	got, n := apply1(t, text, r)
	want := "The soldiers wait. More soldiers come.\n\nA self-portrait of the portrait painter, in a quarter-mile line.\n"
	if got != want || n != 2 {
		t.Errorf("got %d:\n%q\nwant\n%q", n, got, want)
	}
}

// Narration: a quoted speech without a cue gets one.
func TestFixNarration(t *testing.T) {
	text := "Pia at the window.\n\n\"She was nine when the kite\nflew away.\"\n\nShe turns.\n"
	r := proposeRule("\"She was nine when the kite", splitForFix(text))
	if r.Action != fixNarration || r.Template != "NARRATOR (V.O.)" {
		t.Fatalf("proposed %+v", r)
	}
	got, _ := apply1(t, text, r)
	if want := "Pia at the window.\n\nNARRATOR (V.O.)\n\"She was nine when the kite\nflew away.\"\n\nShe turns.\n"; got != want {
		t.Errorf("got %q", got)
	}
}

// A repeated line in capitals ("VOICE OVER") as the narrator's cue.
func TestFixVoiceOverCue(t *testing.T) {
	text := ".VOICE OVER\n\n\"One.\"\n\nRain.\n\n.VOICE OVER\n\n\"Two.\"\n\n.VOICE OVER\n\"Three.\"\n"
	r := proposeRule(".VOICE OVER", splitForFix(text))
	if r.Action != fixCue || r.Pattern != "VOICE OVER" || r.Template != "NARRATOR (V.O.)" {
		t.Fatalf("proposed %+v", r)
	}
	got, n := apply1(t, text, r)
	if want := "NARRATOR (V.O.)\n\"One.\"\n\nRain.\n\nNARRATOR (V.O.)\n\"Two.\"\n\nNARRATOR (V.O.)\n\"Three.\"\n"; got != want || n != 3 {
		t.Errorf("got %d %q", n, got)
	}
}

// Shots in capitals, numbered or not, made headings (forced: no INT/EXT).
func TestFixShots(t *testing.T) {
	text := "A RED FLAG—\n\n—waves.\n\n17 A GLASS OF WATER—\n\n—full.\n"
	f := splitForFix(text)
	if r := proposeRule("A RED FLAG—", f); r.Pattern != "<CAPITALS>—" || r.Action != fixHeading {
		t.Errorf("shot: %+v", r)
	}
	r := proposeRule("17 A GLASS OF WATER—", f)
	if r.Pattern != "<number> <CAPITALS>—" {
		t.Fatalf("numbered shot: %+v", r)
	}
	got, _ := apply1(t, text, r)
	if !strings.Contains(got, "\n.A GLASS OF WATER #17#\n") {
		t.Errorf("got %q", got)
	}
}

// The title page and the blocks at the end are never touched.
func TestFixLeavesTitleAndBlocks(t *testing.T) {
	text := "Title: 12\n\n12\n\nPia.\n\n/* an old\n12\n*/\n\n/* Notes\n12\n*/\n"
	f := splitForFix(text)
	ms, _ := f.matches(fixRule{Action: fixDrop, Pattern: "<number>"})
	if len(ms) != 1 || ms[0].line != 2 {
		t.Errorf("matches %+v", ms)
	}
}

// A regular expression written by hand that does not compile says so.
func TestFixBadRegex(t *testing.T) {
	if _, err := splitForFix("x\n").matches(fixRule{Action: fixDrop, Regex: "(oops"}); err == nil {
		t.Error("no error")
	}
}

// The rules as the /* Fix-Up */ block keeps them, read back; a "*/" in a
// regular expression kept escaped; Clean Fountain drops the block.
func TestFixRulesBlock(t *testing.T) {
	rules := []fixRule{
		{Action: fixHeading, Pattern: "Scene <number> - <text>", Template: "<text> #<number>#"},
		{Action: fixDrop, Regex: `^\*/ \d+$`},
		{Action: fixJoin, Pattern: "<unfinished>"},
	}
	text := "Scene 1 - int. hall\n\nPia.\n\n/* Notes\nhi\n*/\n"
	for _, r := range rules {
		start, end, block := withFixRule(text, r)
		rs := []rune(text)
		text = string(rs[:start]) + block + string(rs[end:])
	}
	// the same rule again is not added twice
	start, end, block := withFixRule(text, rules[0])
	rs := []rune(text)
	text = string(rs[:start]) + block + string(rs[end:])
	got := fixRulesOf(text)
	if len(got) != len(rules) {
		t.Fatalf("read back %d rules from\n%s", len(got), text)
	}
	for i := range rules {
		if got[i] != rules[i] {
			t.Errorf("rule %d: %+v, want %+v", i, got[i], rules[i])
		}
	}
	if !strings.Contains(text, "/* Fix-Up\n") || !strings.HasSuffix(text, "/* Notes\nhi\n*/\n") {
		t.Errorf("the block:\n%s", text)
	}
	if strings.Contains(fountain.Strip(text, nil, fountain.StripOptions{}), "Fix-Up") {
		t.Error("Clean Fountain keeps the block")
	}
	// the rules applied again, after a re-import
	out, counts := applyRules("Scene 2 - ext. yard\n\nRain on the\n\nroof.\n", got, fixEnvEN)
	if out != "EXT. YARD #2#\n\nRain on the roof.\n" || counts[0] != 1 || counts[2] != 1 {
		t.Errorf("applied: %v %q", counts, out)
	}
}

func TestChangedSpan(t *testing.T) {
	s, e, mid := changedSpan("abcXYdef", "abcZdef")
	if s != 3 || e != 5 || mid != "Z" {
		t.Errorf("%d %d %q", s, e, mid)
	}
}

// fixScript is a made-up typescript as an import leaves it: numbered
// "Scene N -" headings, a glued cue, double-spaced paragraphs, page numbers.
const fixScript = "Title: The Kite\n\nScene 3 - int. kitchen, morning\n\nPia waits by the\n\nwindow.\n\n12.\n\nTHE GUARD(quietly)\nWho goes there?\n\nScene 4 - ext. yard\n\nRain.\n\n13.\n\nScene 5 - on the roof & elsewhere\n\nWind.\n"

// fixWindow is a window with fixScript, the cursor on the line that
// starts with at.
func fixWindow(t *testing.T, at string) *MainWindow {
	t.Helper()
	w := newTestWindow(t, fixScript)
	w.textEditor.SetCursorOffset(len([]rune(fixScript[:strings.Index(fixScript, at)])))
	return w
}

// The dialog proposes the cursor's line's pattern, counts and lights the
// matches, previews them; Apply changes them in one undo step and keeps
// the rule; the rule is offered again in the next dialog.
func TestPatternFixerDialog(t *testing.T) {
	w := fixWindow(t, "Scene 4")
	steps := w.textEditor.UndoSteps()
	f := w.newPatternFixer()
	if f.pattern.Text != "Scene <number> - <text>" || f.action.Selected != fixHeading.label() || f.template.Text != "<text> #<number>#" {
		t.Fatalf("proposed %q %q %q", f.pattern.Text, f.action.Selected, f.template.Text)
	}
	if !strings.HasPrefix(f.count.Text, "3 lines match") || len(w.textEditor.Highlights()) != 3 || len(f.preview.Objects) != 3 {
		t.Errorf("count %q, %d lit, %d previewed", f.count.Text, len(w.textEditor.Highlights()), len(f.preview.Objects))
	}
	if f.apply.Disabled() || f.apply.Text != "Apply to 3" {
		t.Errorf("apply button %q", f.apply.Text)
	}
	// a pattern of one's own: fewer
	f.pattern.SetText("Scene <number> - <int/ext>. <text>")
	if !strings.HasPrefix(f.count.Text, "2 lines match") {
		t.Errorf("own pattern: %q", f.count.Text)
	}
	f.pattern.SetText("Scene <number> - <text>")
	// a regular expression that does not read
	f.regex.SetText("(Scene")
	if f.count.Importance != widget.DangerImportance || !f.apply.Disabled() {
		t.Errorf("bad regex: %q", f.count.Text)
	}
	f.pattern.SetText("Scene <number>")
	f.pattern.SetText("Scene <number> - <text>")
	f.d.Show()
	f.applyNow()
	text := w.textEditor.Text()
	for _, want := range []string{"\nINT. KITCHEN, MORNING #3#\n", "\nEXT. YARD #4#\n", "\n.ON THE ROOF & ELSEWHERE #5#\n",
		"/* Fix-Up\nheading: Scene <number> - <text> → <text> #<number>#\n*/"} {
		if !strings.Contains(text, want) {
			t.Errorf("no %q in\n%s", want, text)
		}
	}
	if w.textEditor.UndoSteps() != steps+1 {
		t.Errorf("%d undo steps", w.textEditor.UndoSteps()-steps)
	}
	if len(w.textEditor.Highlights()) != 0 {
		t.Error("the matches still lit after closing")
	}
	w.undo()
	if w.textEditor.Text() != fixScript {
		t.Errorf("undo left\n%s", w.textEditor.Text())
	}
	w.redo()
	// the next dialog lists the rule; applying them all again changes nothing
	g := w.newPatternFixer()
	if len(g.saved.Objects) != 3 {
		t.Errorf("saved rules shown: %d objects", len(g.saved.Objects))
	}
	g.d.Hide()
}

// The other actions from the dialog: drop the page numbers, split the
// glued cue, join the double-spaced paragraph; each its own undo step.
func TestPatternFixerActions(t *testing.T) {
	w := fixWindow(t, "12.")
	f := w.newPatternFixer()
	if f.action.Selected != fixDrop.label() || !strings.HasPrefix(f.count.Text, "2 lines") {
		t.Fatalf("page numbers: %q %q", f.action.Selected, f.count.Text)
	}
	f.applyNow()
	w.textEditor.SetCursorOffset(strings.Index(w.textEditor.Text(), "THE GUARD"))
	f = w.newPatternFixer()
	if f.action.Selected != fixSplitCue.label() {
		t.Fatalf("glued cue: %q", f.action.Selected)
	}
	f.applyNow()
	w.textEditor.SetCursorOffset(strings.Index(w.textEditor.Text(), "Pia waits"))
	f = w.newPatternFixer()
	if f.action.Selected != fixJoin.label() {
		t.Fatalf("join: %q", f.action.Selected)
	}
	// the template row is for the actions that write a result
	if f.tmplRow.Visible() {
		t.Error("a result for joining")
	}
	f.action.SetSelected(fixHeading.label())
	if !f.tmplRow.Visible() {
		t.Error("no result for headings")
	}
	f.action.SetSelected(fixJoin.label())
	f.applyNow()
	text := w.textEditor.Text()
	for _, want := range []string{"Pia waits by the window.\n", "THE GUARD\n(quietly)\nWho goes there?", "Rain.\n\nScene 5"} {
		if !strings.Contains(text, want) {
			t.Errorf("no %q in\n%s", want, text)
		}
	}
	if strings.Contains(text, "12.") {
		t.Errorf("page number kept:\n%s", text)
	}
	if n := len(fixRulesOf(text)); n != 3 {
		t.Errorf("%d rules kept", n)
	}
}

// A re-import of the same file keeps the rules and offers them; another
// file's import does not.
func TestFixRulesCarriedOnReimport(t *testing.T) {
	w := fixWindow(t, "12.")
	w.suggestedName = "kite.fountain"
	f := w.newPatternFixer()
	f.applyNow()
	text, rules := w.carryFixRules("/x/kite.pdf", "Rain.\n\n7.\n\nWind.\n")
	if len(rules) != 1 || !strings.Contains(text, "/* Fix-Up\ndrop: <number>.\n*/") {
		t.Errorf("carried %d:\n%s", len(rules), text)
	}
	if _, rules := w.carryFixRules("/x/other.pdf", "Rain.\n"); rules != nil {
		t.Error("carried to another file's import")
	}
	w.showImported("/x/kite.pdf", "Rain.\n\n7.\n\nWind.\n", "a PDF")
	if !strings.Contains(w.textEditor.Text(), "/* Fix-Up") || strings.Contains(w.textEditor.Text(), "7.") == false {
		t.Errorf("after re-import (not applied by itself):\n%s", w.textEditor.Text())
	}
	w.applyFixRules(fixRulesOf(w.textEditor.Text()), false)
	if strings.Contains(w.textEditor.Text(), "7.") {
		t.Errorf("applied again:\n%s", w.textEditor.Text())
	}
}

// Screenshots of the fixer on a desktop and a phone (SHOT_DIR=… go test
// -run TestPatternFixerScreenshots): fixup-*.png.
func TestPatternFixerScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(dir, 0o755)
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"desktop", fyne.NewSize(1100, 760)}, {"phone", fyne.NewSize(380, 800)}} {
		w := fixWindow(t, "Scene 4")
		w.fyneWindow.Resize(size.size)
		f := w.newPatternFixer()
		f.d.Show()
		capture(t, w.fyneWindow.Canvas(), "fixup-heading-"+size.name+".png")
		f.d.Hide()
		w.textEditor.SetCursorOffset(strings.Index(w.textEditor.Text(), "THE GUARD"))
		f = w.newPatternFixer()
		f.d.Show()
		capture(t, w.fyneWindow.Canvas(), "fixup-cue-"+size.name+".png")
		f.d.Hide()
		w.textEditor.SetCursorOffset(strings.Index(w.textEditor.Text(), "12."))
		f = w.newPatternFixer()
		f.regexBox.Open(0)
		f.d.Show()
		capture(t, w.fyneWindow.Canvas(), "fixup-regex-"+size.name+".png")
		f.d.Hide()
	}
}
