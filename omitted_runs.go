package main

import (
	"strings"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"
)

// Runs of OMITTED scenes (docs/PRODUCTION.md §15). The text keeps one
// ".OMITTED #n#" heading a scene, each with its old scene in boneyard
// under it, so each keeps its number and Restore Scene brings it back. A
// shooting script may print a run of them as one line, "12-14 OMITTED
// 12-14": a setting in the production block, so that it travels with the
// file:
//
//	/* Production
//	scene numbers: locked, 10A
//	omitted runs: print as one
//	*/
//
// Renumber's OMITTED choice "Keep them, print runs as one" sets it ("Keep
// them with their numbers" takes it out). PDF, HTML, DOCX, ODT and the
// print follow it (Lexington's MergeOmitted: the pages, their lock and
// the scenes' lengths are the merged print's); FDX and Fountain keep one
// heading a number.
//
// Lexington v1.13.0 (Accolade's first Renumber strategies) wrote a run
// into the text as one heading, ".OMITTED #12-14#". Accolade reads it as
// its scenes, and writes it as one heading a scene (fountain.SplitOmitted)
// on the next Renumber, Unlock or Save: one undo step, the setting
// switched on so that it prints as before, the page lock's "12-14" its
// first scene's, with a note in the status bar.

// omittedRunsKey is the setting's key, omittedRunsOne its value.
const (
	omittedRunsKey = "omitted runs"
	omittedRunsOne = "print as one"
)

// omittedRunsPrinted tells whether the script prints a run of OMITTED
// headings as one.
func omittedRunsPrinted(text string) bool {
	inside, ok := productionBlock(text)
	if !ok {
		return false
	}
	for _, line := range strings.Split(inside, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if ok && strings.EqualFold(strings.TrimSpace(key), omittedRunsKey) {
			return strings.Contains(strings.ToLower(value), "one")
		}
	}
	return false
}

// withOmittedRuns is the block's inside with the setting on or off (its
// line taken out), the other lines kept.
func withOmittedRuns(inside string, one bool) string {
	if one {
		return withBlockLine(inside, omittedRunsKey, omittedRunsOne)
	}
	return withoutBlockLine(inside, omittedRunsKey)
}

// oldOmittedRun tells whether a heading is v1.13.0's merged run.
func oldOmittedRun(h sceneHead) bool {
	_, _, ok := lex.OmittedRange(strings.TrimSpace(h.text))
	return ok
}

// splitOldOmittedRuns writes v1.13.0's merged OMITTED headings as one
// heading a scene: one undo step, the setting on, the page lock
// following; it says so in the status bar. false if there were none.
func (w *MainWindow) splitOldOmittedRuns() bool {
	if w.markdown || w.textEditor == nil {
		return false
	}
	e := w.textEditor
	text := e.Text()
	body := bodyEndByte(text)
	heads := sceneHeads(text[:body], w.sceneStarts())
	var runs []int
	for i, h := range heads {
		if oldOmittedRun(h) {
			runs = append(runs, i)
		}
	}
	if len(runs) == 0 {
		return false
	}
	// the page lock: a page that starts at the run now starts at its first
	p := numberPlan{nums: make([]string, len(heads)), drop: make([]bool, len(heads))}
	for i, h := range heads {
		p.nums[i] = h.num
		if oldOmittedRun(h) {
			p.nums[i], _, _ = lex.OmittedRange(strings.TrimSpace(h.text))
		}
	}
	inside, _ := productionBlock(text)
	inside = pageLockFollowing(text, withOmittedRuns(inside, true), heads, p)
	var names []string
	e.Edit(func(b *buffer.Buffer) {
		runes := []rune(text)
		bodyEnd := bodyEnd(text)
		var edits []lineEdit
		for _, i := range runs {
			start, end := b.LineStart(heads[i].line), bodyEnd
			if i+1 < len(heads) {
				end = b.LineStart(heads[i+1].line)
			}
			// up to its last text: the blank lines after it stay, where a
			// new production block may go
			end = start + len([]rune(strings.TrimRight(string(runes[start:end]), "\n")))
			out, ok := fountain.SplitOmitted(string(runes[start:end]))
			if !ok {
				continue
			}
			edits = append(edits, lineEdit{start, end, out})
			names = append(names, heads[i].num)
		}
		bs, be, block := withProductionBlock(text, inside)
		line, col := b.Position(b.Cursor())
		b.Group(func() {
			b.Replace(bs, be, block) // after the script: the headings' places hold
			applyBackwards(edits, b.Replace)
			b.SetCursor(b.Offset(line, col), false)
		})
	})
	w.followSceneNumbers()
	w.setStatus("OMITTED " + strings.Join(names, ", ") + " written as one OMITTED heading a scene, as the text keeps them now; " +
		"they still print as one (Production > Renumber Scenes…, OMITTED) — Undo takes it back")
	return true
}
