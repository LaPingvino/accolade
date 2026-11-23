package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/LaPingvino/accolade/internal/prose"
	"github.com/LaPingvino/accolade/internal/storyread"
)

// The adaptation benchmark: public-domain pairs of a story and a period
// stage version of it (testdata/adapt-bench/<pair>/: source.md,
// script.fountain, and mapping.tsv, the passages each scene adapts,
// checked by hand). For each pair it runs the matching Accolade has and
// reports how many scenes it places:
//
//   - suggest: suggestLinks, the per-scene guesser the source pane offers
//     links from (a script with no links, all passages still to do, the
//     story read by storyread). Its guess for a scene is the passages it
//     suggests (at most three), each on its own: right when one of them
//     is in the mapping.
//   - align: alignPassagesSure, the passage aligner relinking uses, run on
//     the whole script made into passages (each block of action or speech
//     one, cues and parentheticals left out) against the source. Its guess
//     for a scene is the range of the passages its blocks pair with.
//
// A scene with a mapping is right when the guess overlaps one of its
// ranges, wrong when it does not, silent when there is no guess. Invented
// scenes ("none") are counted apart: silent is right there, any guess
// wrong. It reports and does not fail on the numbers; it fails only when
// the files disagree with each other (a scene count, a heading, or a
// passage index whose words moved).
//
// The numbers are the same from run to run (TestSuggestLinksSameEachTime).
func TestAdaptBench(t *testing.T) {
	if testing.Short() {
		t.Skip("adaptation benchmark: not with -short")
	}
	dirs, _ := filepath.Glob(filepath.Join("testdata", "adapt-bench", "*", "mapping.tsv"))
	if len(dirs) == 0 {
		t.Fatal("no pairs in testdata/adapt-bench")
	}
	for _, m := range dirs {
		dir := filepath.Dir(m)
		t.Run(filepath.Base(dir), func(t *testing.T) { adaptBenchPair(t, dir) })
	}
}

// benchRow is a line of mapping.tsv.
type benchRow struct {
	heading     string
	ranges      [][2]int // none: empty
	confidence  string
	first, last string // the words the first and last passage start with
}

func readBenchMapping(t *testing.T, path string) []benchRow {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rows []benchRow
	for n, line := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 5 {
			t.Fatalf("%s:%d: %d fields, want at least 5", path, n+1, len(f))
		}
		r := benchRow{heading: f[0], confidence: f[2], first: f[3], last: f[4]}
		if f[1] != "none" {
			for _, rg := range strings.Split(f[1], ",") {
				a, z, ok := strings.Cut(rg, "-")
				i, err1 := strconv.Atoi(a)
				j, err2 := strconv.Atoi(z)
				if !ok || err1 != nil || err2 != nil || j < i {
					t.Fatalf("%s:%d: bad range %q", path, n+1, rg)
				}
				r.ranges = append(r.ranges, [2]int{i, j})
			}
		}
		rows = append(rows, r)
	}
	return rows
}

// benchScore is one method's tally on one pair.
type benchScore struct{ right, wrong, silent, noneSilent, noneGuessed int }

func (s *benchScore) add(r benchRow, guess [][2]int) string {
	guessed := len(guess) > 0
	switch {
	case len(r.ranges) == 0 && !guessed:
		s.noneSilent++
		return "silent (none: right)"
	case len(r.ranges) == 0:
		s.noneGuessed++
		return "guessed (none: wrong)"
	case !guessed:
		s.silent++
		return "silent"
	}
	for _, rg := range r.ranges {
		for _, g := range guess {
			if g[0] <= rg[1] && g[1] >= rg[0] {
				s.right++
				return "right"
			}
		}
	}
	s.wrong++
	return "wrong"
}

func adaptBenchPair(t *testing.T, dir string) {
	src, err := os.ReadFile(filepath.Join(dir, "source.md"))
	if err != nil {
		t.Fatal(err)
	}
	scriptB, err := os.ReadFile(filepath.Join(dir, "script.fountain"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(scriptB)
	rows := readBenchMapping(t, filepath.Join(dir, "mapping.tsv"))
	ps := passagesOf(string(src))
	starts := []string{"INT.", "EXT.", "INT./EXT.", "I/E"}

	// the files agree: a row per scene, by its heading, and the passages
	// still where the mapping says
	scenes := scenesOf(script, starts)
	if len(scenes) != len(rows) {
		t.Fatalf("%d scenes in the script, %d rows in the mapping", len(scenes), len(rows))
	}
	for k, r := range rows {
		if scenes[k].Heading != r.heading {
			t.Fatalf("scene %d is %q, the mapping has %q", k, scenes[k].Heading, r.heading)
		}
		if len(r.ranges) == 0 {
			continue
		}
		a, z := r.ranges[0][0], r.ranges[len(r.ranges)-1][1]
		if z >= len(ps) || !strings.HasPrefix(ps[a].Plain, r.first) || !strings.HasPrefix(ps[z].Plain, r.last) {
			t.Fatalf("%s: passages %d–%d are not the mapping's (%q … %q): source.md changed under it", r.heading, a, z, r.first, r.last)
		}
	}

	// suggest
	m := newSourceModel(ps, script, starts)
	reading := storyread.Read(storyPassages(ps), "en", storyread.Hints{})
	sugs := suggestLinks(script, starts, ps, m.status, reading, nil)
	byScene := map[int][]int{}
	for _, s := range sugs {
		byScene[s.Scene] = s.Passages
	}
	var sug benchScore
	for k, r := range rows {
		var guess [][2]int
		for _, p := range byScene[k] {
			guess = append(guess, [2]int{p, p})
		}
		t.Logf("suggest %-60s %-28s guess %v: %s", r.heading, rangesString(r.ranges), byScene[k], sug.add(r, guess))
	}

	// align
	sps, sceneOf := scriptPassages(script, starts)
	to, sure := alignPassagesSure(sps, ps)
	var al benchScore
	for k, r := range rows {
		lo, hi, n, nSure := -1, -1, 0, 0
		for i, j := range to {
			if sceneOf[i] != k || j < 0 {
				continue
			}
			if lo < 0 || j < lo {
				lo = j
			}
			hi = max(hi, j)
			n++
			if sure[i] {
				nSure++
			}
		}
		var guess [][2]int
		if n > 0 {
			guess = [][2]int{{lo, hi}}
		}
		t.Logf("align   %-60s %-28s guess %d–%d (%d blocks paired, %d sure): %s", r.heading, rangesString(r.ranges), lo, hi, n, nSure, al.add(r, guess))
	}
	report := func(name string, s benchScore) {
		t.Logf("%s %s: %d scenes mapped: %d right, %d wrong, %d silent; %d invented: %d silent (right), %d guessed (wrong)",
			filepath.Base(dir), name, s.right+s.wrong+s.silent, s.right, s.wrong, s.silent,
			s.noneSilent+s.noneGuessed, s.noneSilent, s.noneGuessed)
	}
	report("suggest", sug)
	report("align  ", al)
}

func rangesString(rs [][2]int) string {
	if len(rs) == 0 {
		return "none"
	}
	var parts []string
	for _, r := range rs {
		parts = append(parts, strconv.Itoa(r[0])+"-"+strconv.Itoa(r[1]))
	}
	return strings.Join(parts, ",")
}

// scriptPassages is the script's scenes as passages, for the aligner:
// each block of action or of speech one (its cue, parentheticals and the
// scene heading left out), and for each the scene (its index in scenesOf)
// it is in.
func scriptPassages(script string, starts []string) ([]passage, []int) {
	body := script[:bodyEndByte(script)]
	lines := strings.Split(body, "\n")
	var heads []int
	for _, it := range outlineOf(body, starts) {
		if it.Scene {
			heads = append(heads, it.Line)
		}
	}
	var ps []passage
	var sceneOf []int
	for k, h := range heads {
		end := len(lines)
		if k+1 < len(heads) {
			end = heads[k+1]
		}
		var cur []string
		flush := func() {
			if len(cur) == 0 {
				return
			}
			plain := strings.Join(cur, " ")
			cur = nil
			if ws := normWords(plain); len(ws) > 0 {
				ps = append(ps, passage{Index: len(ps), Kind: prose.Para, Plain: plain, Words: ws})
				sceneOf = append(sceneOf, k)
			}
		}
		blank := true
		for _, raw := range lines[h+1 : end] {
			l := strings.TrimSpace(stripNotes(raw))
			switch {
			case l == "":
				flush()
				blank = true
				continue
			case strings.HasPrefix(l, "#") || strings.HasPrefix(l, "="):
			case blank && isCharacterCue(l):
			case strings.HasPrefix(l, "("):
			default:
				cur = append(cur, strings.TrimPrefix(l, "!"))
			}
			blank = false
		}
		flush()
	}
	return ps, sceneOf
}

// TestSuggestLinksSameEachTime: the same script and source give the same
// guesses each time (the sums once ran in a map's order, so near-ties fell
// either way from one call to the next).
func TestSuggestLinksSameEachTime(t *testing.T) {
	dir := filepath.Join("testdata", "adapt-bench", "jane-eyre")
	src, err := os.ReadFile(filepath.Join(dir, "source.md"))
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile(filepath.Join(dir, "script.fountain"))
	if err != nil {
		t.Fatal(err)
	}
	ps := passagesOf(string(src))
	starts := []string{"INT.", "EXT.", "INT./EXT.", "I/E"}
	m := newSourceModel(ps, string(script), starts)
	first := fmt.Sprint(suggestLinks(string(script), starts, ps, m.status, nil, nil))
	for range 6 {
		if again := fmt.Sprint(suggestLinks(string(script), starts, ps, m.status, nil, nil)); again != first {
			t.Fatalf("another answer for the same input:\n%s\n%s", first, again)
		}
	}
}
