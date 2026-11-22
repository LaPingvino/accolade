package main

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/lexington/lex"
)

// The Production pane's reading of the script (production_pane.go shows
// it): the check before the lock, and what changed in the revision.
// Pure functions of the text, so the pane computes them once typing
// pauses and the tests read them directly.

// prodCheck is one thing to look at before the lock: the line it is on
// (the place a tap goes to), what is wrong, and its kind; info is a
// note, not a problem (no numbers yet: the lock writes them).
type prodCheck struct {
	line int
	kind string
	text string
	info bool
}

// The kinds of check.
const (
	checkPlace    = "place"    // a heading without INT./EXT.
	checkTime     = "time"     // a heading without a time of day
	checkNumber   = "number"   // a duplicate, missing or stray scene number
	checkBoneyard = "boneyard" // /* without */, or */ without /*
	checkNote     = "note"     // [[ without ]], or ]] without [[
	checkCue      = "cue"      // one character under two cues
	checkLong     = "long"     // a very long scene
	checkTitle    = "title"    // no title page
)

// longScenePages is how long (estimated pages) a scene is before the check
// calls it very long: a day's shooting is some five pages, so a scene over
// three is worth a look before it is locked in.
const longScenePages = 3

// headLabel is how a check names a scene: its number and heading.
func headLabel(h sceneHead) string {
	t, _ := lex.SceneNumber(strings.TrimSpace(h.text))
	t = strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(t), ".")))
	if h.num != "" {
		return h.num + " " + t
	}
	return t
}

// headingParts are a heading's INT./EXT. (tag) and its time: "" when it
// has none.
func headingParts(heading string, starts []string) (tag, time string) {
	t, _ := lex.SceneNumber(strings.TrimSpace(heading))
	t = strings.ToUpper(strings.Join(strings.Fields(strings.TrimPrefix(strings.TrimSpace(t), ".")), " "))
	rest := t
	if tag = headingTag(t, starts); tag != "" {
		rest = strings.TrimLeft(rest[len(tag):], ". -")
	}
	cut, at := -1, ""
	for _, d := range []string{" - ", " – ", " — ", " -- "} {
		if i := strings.LastIndex(rest, d); i > cut {
			cut, at = i, d
		}
	}
	if cut >= 0 {
		time = strings.TrimSpace(rest[cut+len(at):])
	}
	return tag, time
}

// boneyardLines are the lines of the body in a boneyard (/* … */), as the
// outline reads them: from a line starting with /* to the one with */.
func boneyardLines(lines []string) []bool {
	in := make([]bool, len(lines))
	open := false
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if open || strings.HasPrefix(t, "/*") {
			in[i] = true
			open = !strings.Contains(strings.TrimPrefix(t, "/*"), "*/")
		}
	}
	return in
}

// preLockChecks are the checks of a script before its lock (text: the
// whole script; only its body is read, not the blocks at its end), in
// the order of their lines.
func preLockChecks(text string, starts []string) []prodCheck {
	body := text[:bodyEndByte(text)]
	lines := strings.Split(body, "\n")
	var out []prodCheck
	heads := sceneHeads(body, starts)

	// the title page
	if strings.TrimSpace(body) != "" {
		kinds := syntax.ClassifyIn(lines[:min(len(lines), 40)], starts, dialectForStarts(starts))
		if len(kinds) == 0 || kinds[0] != syntax.TitlePage {
			out = append(out, prodCheck{line: 0, kind: checkTitle, text: "No title page: Edit > Title Page… writes one (the issued sets are listed there later)"})
		}
	}

	// headings: INT./EXT. and a time
	for _, h := range heads {
		if _, omitted := lex.OmittedNumber(strings.TrimSpace(h.text)); omitted {
			continue
		}
		tag, time := headingParts(h.text, starts)
		switch {
		case tag == "" && time == "":
			out = append(out, prodCheck{line: h.line, kind: checkPlace, text: "No INT./EXT. and no time of day: " + headLabel(h)})
		case tag == "":
			out = append(out, prodCheck{line: h.line, kind: checkPlace, text: "No INT./EXT.: " + headLabel(h)})
		case time == "":
			out = append(out, prodCheck{line: h.line, kind: checkTime, text: "No time of day (DAY, NIGHT…): " + headLabel(h)})
		}
	}

	// scene numbers: twice, missing (some numbered, not all), inside the heading
	numbered := 0
	first := map[string]int{}
	for _, h := range heads {
		if h.num != "" {
			numbered++
		}
	}
	if numbered == 0 && len(heads) > 0 {
		out = append(out, prodCheck{line: heads[0].line, kind: checkNumber, info: true,
			text: "No scene numbers yet: Lock Scene Numbers writes them all"})
	}
	for _, h := range heads {
		switch {
		case h.num == "" && numbered > 0:
			out = append(out, prodCheck{line: h.line, kind: checkNumber, text: "No number (Lock Scene Numbers gives it one): " + headLabel(h)})
		case h.stray:
			out = append(out, prodCheck{line: h.line, kind: checkNumber, text: "Number " + h.num + " inside the heading, not at its end: " + headLabel(h)})
		}
		if h.num == "" {
			continue
		}
		key := strings.ToUpper(h.num)
		if at, ok := first[key]; ok {
			out = append(out, prodCheck{line: h.line, kind: checkNumber,
				text: fmt.Sprintf("Number %s twice (also line %d): %s", h.num, at+1, headLabel(h))})
		} else {
			first[key] = h.line
		}
	}

	out = append(out, balanceChecks(lines)...)
	out = append(out, cueChecks(text, lines, starts)...)
	out = append(out, longScenes(lines, heads, starts)...)

	sort.SliceStable(out, func(i, j int) bool { return out[i].line < out[j].line })
	return out
}

// balanceChecks find a boneyard or a note opened and not closed, or
// closed and not opened, in the body's lines: Fountain hides everything
// up to the close, so an open one hides the script after it.
func balanceChecks(lines []string) []prodCheck {
	var out []prodCheck
	bone, note := -1, -1 // the line of the open one
	for i, l := range lines {
		for j := 0; j+1 < len(l); j++ {
			two := l[j : j+2]
			switch {
			case bone >= 0:
				switch two {
				case "*/":
					bone = -1
					j++
				case "/*":
					out = append(out, prodCheck{line: bone, kind: checkBoneyard,
						text: fmt.Sprintf("/* not closed before the next /* (line %d): the script between is hidden", i+1)})
					bone = i
					j++
				}
			case two == "/*":
				bone = i
				j++
			case two == "*/":
				out = append(out, prodCheck{line: i, kind: checkBoneyard, text: "*/ without a /* before it"})
				j++
			case two == "[[":
				if note >= 0 {
					out = append(out, prodCheck{line: note, kind: checkNote,
						text: fmt.Sprintf("[[ not closed before the next [[ (line %d)", i+1)})
				}
				note = i
				j++
			case two == "]]":
				if note < 0 {
					out = append(out, prodCheck{line: i, kind: checkNote, text: "]] without a [[ before it"})
				}
				note = -1
				j++
			}
		}
		// a note ends at an empty line (one of two spaces is kept in it)
		if note >= 0 && note < i && strings.TrimRight(l, "\r") == "" {
			out = append(out, prodCheck{line: note, kind: checkNote, text: "[[ not closed: the note runs on into the script"})
			note = -1
		}
	}
	if bone >= 0 {
		out = append(out, prodCheck{line: bone, kind: checkBoneyard, text: "/* not closed: everything after it is hidden"})
	}
	if note >= 0 {
		out = append(out, prodCheck{line: note, kind: checkNote, text: "[[ not closed"})
	}
	return out
}

// cueInfo is a cue's name as the script uses it: how often, and its first line.
type cueInfo struct {
	name  string
	count int
	line  int
}

// cueChecks find one character under two cues: a cue that is another
// name the adaptation learned for a cue (characters: ANNA = Annie), or
// two cues alike but for case, dots and spaces (MRS. BENNET, MRS BENNET).
func cueChecks(text string, lines []string, starts []string) []prodCheck {
	in := boneyardLines(lines)
	masked := make([]string, len(lines))
	for i, l := range lines {
		if !in[i] {
			masked[i] = l
		}
	}
	cues := map[string]*cueInfo{}
	var order []string
	for i, k := range syntax.ClassifyIn(masked, starts, dialectForStarts(starts)) {
		if k != syntax.Character {
			continue
		}
		name := strings.ToUpper(cueName(masked[i]))
		if name == "" {
			continue
		}
		c := cues[name]
		if c == nil {
			c = &cueInfo{name: name, line: i}
			cues[name] = c
			order = append(order, name)
		}
		c.count++
	}
	var out []prodCheck
	flagged := map[string]bool{}
	l, _ := readLearned(text)
	for _, a := range l.Aliases {
		cue := strings.ToUpper(strings.TrimSpace(a.Cue))
		for _, n := range a.Names {
			other := strings.ToUpper(strings.TrimSpace(n))
			c := cues[other]
			if other == cue || c == nil || flagged[other] {
				continue
			}
			flagged[other] = true
			out = append(out, prodCheck{line: c.line, kind: checkCue,
				text: fmt.Sprintf("%s (%s) is %s's other name, as learned: one cue for both?", other, plural(c.count, "cue"), cue)})
		}
	}
	byKey := map[string][]*cueInfo{}
	var keys []string
	for _, n := range order {
		k := wordsKey(n)
		if k == "" {
			continue
		}
		if byKey[k] == nil {
			keys = append(keys, k)
		}
		byKey[k] = append(byKey[k], cues[n])
	}
	for _, k := range keys {
		cs := byKey[k]
		if len(cs) < 2 {
			continue
		}
		// the most used is the one meant; the others are flagged
		slices.SortStableFunc(cs, func(a, b *cueInfo) int { return b.count - a.count })
		for _, c := range cs[1:] {
			if flagged[c.name] {
				continue
			}
			flagged[c.name] = true
			out = append(out, prodCheck{line: c.line, kind: checkCue,
				text: fmt.Sprintf("%s (%s) and %s (%s): one character?", c.name, plural(c.count, "cue"), cs[0].name, plural(cs[0].count, "cue"))})
		}
	}
	return out
}

// sceneRows estimates the printed lines of lines (the status bar's page
// estimate: action 60 columns, dialogue 35, parentheticals 25).
func sceneRows(lines []string, kinds []syntax.Kind) int {
	rows := 0
	for i, k := range kinds {
		if k == syntax.TitlePage {
			continue
		}
		width := actionColumns
		switch k {
		case syntax.Dialogue:
			width = dialogColumns
		case syntax.Parenthetical:
			width = parenColumns
		}
		rows += max(1, (utf8.RuneCountInString(strings.TrimSpace(lines[i]))+width-1)/width)
	}
	return rows
}

// longScenes are the scenes estimated longer than longScenePages.
func longScenes(lines []string, heads []sceneHead, starts []string) []prodCheck {
	in := boneyardLines(lines)
	masked := make([]string, len(lines))
	for i, l := range lines {
		if !in[i] {
			masked[i] = l
		}
	}
	kinds := syntax.ClassifyIn(masked, starts, dialectForStarts(starts))
	var out []prodCheck
	for i, h := range heads {
		end := len(lines)
		if i+1 < len(heads) {
			end = heads[i+1].line
		}
		rows := sceneRows(masked[h.line:end], kinds[h.line:end])
		if rows > longScenePages*linesPerPage {
			e := lex.EighthsOf(float64(rows) / linesPerPage)
			out = append(out, prodCheck{line: h.line, kind: checkLong,
				text: "Very long, about " + e.String() + " pages: " + headLabel(h)})
		}
	}
	return out
}

// problems is how many of the checks are problems (not notes).
func problems(cs []prodCheck) int {
	n := 0
	for _, c := range cs {
		if !c.info {
			n++
		}
	}
	return n
}

// changedScene is a scene with a line marked in the current revision.
type changedScene struct {
	line    int    // its heading's (0: the title page, before any scene)
	label   string // its number, else its heading
	heading string
}

// changedScenes are the scenes with a line marked set (marks: one per
// line of text, as the revision chain gives them), in order; changes
// before the first scene are the title page's.
func changedScenes(text string, starts []string, marks []string, set string) []changedScene {
	body := text[:bodyEndByte(text)]
	heads := sceneHeads(body, starts)
	var out []changedScene
	seen := map[int]bool{}
	for i, m := range marks {
		if m == "" || m != set {
			continue
		}
		at := -1 // the scene: the last heading at or before line i
		for k, h := range heads {
			if h.line > i {
				break
			}
			at = k
		}
		if seen[at] {
			continue
		}
		seen[at] = true
		if at < 0 {
			out = append(out, changedScene{line: 0, label: "Title page", heading: "before the first scene"})
			continue
		}
		h := heads[at]
		label := h.num
		t, _ := lex.SceneNumber(strings.TrimSpace(h.text))
		t = strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(t), ".")))
		if label == "" {
			label = t
		}
		out = append(out, changedScene{line: h.line, label: label, heading: t})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].line < out[j].line })
	return out
}

// sceneList is the changed scenes' labels as the count line has them:
// "scenes 4, 12A, 17", the first few and "and n more".
func sceneList(cs []changedScene) string {
	if len(cs) == 0 {
		return "no scene changed"
	}
	const shown = 8
	var labels []string
	for i, c := range cs {
		if i == shown {
			labels = append(labels, fmt.Sprintf("and %d more", len(cs)-shown))
			break
		}
		labels = append(labels, c.label)
	}
	word := "scenes "
	if len(cs) == 1 {
		word = "scene "
	}
	if cs[0].label == "Title page" {
		word = ""
	}
	return word + strings.Join(labels, ", ")
}
