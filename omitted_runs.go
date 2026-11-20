package main

import "strings"

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
// heading a number. ".OMITTED #12-14#" in the text is one scene labelled
// "12-14", a fixed label like any dashed one.

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
