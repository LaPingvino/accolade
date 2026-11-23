package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/LaPingvino/accolade/internal/prose"
)

// The adaptation benchmark: public-domain pairs of a story and a period
// stage version of it (testdata/adapt-bench/<pair>/: source.md,
// script.fountain, and mapping.tsv, the passages each scene adapts,
// checked by hand). For each pair it runs the matching Accolade has and
// reports how many scenes it places:
//
//   - scenes: alignScenes (scene_align.go), the scene aligner the
//     script's "↗ link?" guesses and the source pane's sections come
//     from: the scenes placed in order along the source, each silent
//     unless sure. Its guess for a scene is a range of passages.
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
// Before it (2026-11-23), suggestLinks guessed for every scene: Christmas
// Carol 12 right, the invented one guessed; Jane Eyre 0 right of 11 once
// its sums no longer ran in a map's order (0–3 before, from run to run),
// both invented ones guessed.
//
// The numbers are the same from run to run (TestAlignScenesSameEachTime).
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
	// scenes: the scene aligner (scene_align.go), silent when unsure
	guesses := alignScenes(scenes, ps, nil)
	var sa benchScore
	for k, r := range rows {
		g := guesses[k]
		var guess [][2]int
		if g.guessed() {
			guess = [][2]int{{g.First, g.Last}}
		}
		t.Logf("scenes  %-60s %-28s guess %d–%d (%d wide, z %.1f, sure %.1f): %s", r.heading, rangesString(r.ranges),
			g.First, g.Last, g.Last-g.First+1, g.Z, g.Sure, sa.add(r, guess))
	}
	// cuts: the sections the pane would offer as cut; one the mapping
	// says a scene adapts is wrong
	allToDo := newSourceModel(ps, script, starts).status
	var cutRight, cutWrong int
	for _, sec := range sectionsOf(ps, allToDo, guesses) {
		if !sec.skipped {
			continue
		}
		wrong := false
		for _, r := range rows {
			for _, rg := range r.ranges {
				wrong = wrong || rg[0] <= sec.last && rg[1] >= sec.first
			}
		}
		if wrong {
			cutWrong++
		} else {
			cutRight++
		}
		t.Logf("cut?    %-60s %d–%d: wrong %v", sec.name, sec.first, sec.last, wrong)
	}
	t.Logf("%s cuts offered: %d right, %d wrong (adapted by the mapping)", filepath.Base(dir), cutRight, cutWrong)
	start := time.Now()
	for range 5 {
		alignScenes(scenes, ps, nil)
	}
	t.Logf("%s alignScenes: %v a look (%d scenes, %d passages)", filepath.Base(dir), time.Since(start)/5, len(scenes), len(ps))
	report := func(name string, s benchScore) {
		t.Logf("%s %s: %d scenes mapped: %d right, %d wrong, %d silent; %d invented: %d silent (right), %d guessed (wrong)",
			filepath.Base(dir), name, s.right+s.wrong+s.silent, s.right, s.wrong, s.silent,
			s.noneSilent+s.noneGuessed, s.noneSilent, s.noneGuessed)
	}
	report("align  ", al)
	report("scenes ", sa)
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

// TestAlignScenesSameEachTime: the same script and source give the same
// guesses each time (sums over a map's words run in their order).
func TestAlignScenesSameEachTime(t *testing.T) {
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
	scenes := scenesOf(string(script), []string{"INT.", "EXT.", "INT./EXT.", "I/E"})
	first := fmt.Sprint(alignScenes(scenes, ps, nil))
	for range 4 {
		if again := fmt.Sprint(alignScenes(scenes, ps, nil)); again != first {
			t.Fatalf("another answer for the same input:\n%s\n%s", first, again)
		}
	}
}

// TestAdaptBenchPrivate measures on a script and source kept outside the
// repository (ACCOLADE_PRIVATE_BENCH, default
// ~/.local/share/accolade-bench/kredu/*.textbundle: text*.fountain and
// assets/source.md), counts only: nothing of the text is printed. The
// scenes linked there are the truth: their links taken out (in memory),
// are they found again? Then with them in as anchors: how many of the
// rest are guessed (whether rightly is not known).
func TestAdaptBenchPrivate(t *testing.T) {
	glob := os.Getenv("ACCOLADE_PRIVATE_BENCH")
	if glob == "" {
		home, _ := os.UserHomeDir()
		glob = filepath.Join(home, ".local", "share", "accolade-bench", "kredu", "*.textbundle")
	}
	dirs, _ := filepath.Glob(glob)
	if len(dirs) == 0 {
		t.Skip("no private bench here")
	}
	for k, dir := range dirs {
		texts, _ := filepath.Glob(filepath.Join(dir, "text.*.fountain"))
		if len(texts) == 0 {
			texts, _ = filepath.Glob(filepath.Join(dir, "text.fountain"))
		}
		src, err1 := os.ReadFile(filepath.Join(dir, "assets", "source.md"))
		if len(texts) == 0 || err1 != nil {
			continue
		}
		b, _ := os.ReadFile(texts[0])
		script := string(b)
		lang := ""
		if parts := strings.Split(filepath.Base(texts[0]), "."); len(parts) == 3 {
			lang = parts[1]
		}
		starts := sceneStarts(lang)
		ps := passagesOf(string(src))
		m := newSourceModel(ps, script, starts)
		scenes := scenesOf(script, starts)
		truth := sceneAnchors(scenes, m)
		// the links out
		rs := []rune(script)
		links := parseLinks(script)
		for i := len(links) - 1; i >= 0; i-- {
			rs = append(rs[:links[i].Start:links[i].Start], rs[links[i].End:]...)
		}
		bare := string(rs)
		bareScenes := scenesOf(bare, starts)
		if len(bareScenes) != len(scenes) {
			t.Fatalf("pair %d: %d scenes without the links, %d with", k, len(bareScenes), len(scenes))
		}
		var right, wrong, silent, others, guessed int
		gs := alignScenes(bareScenes, ps, nil)
		for s, g := range gs {
			tr, ok := truth[s]
			switch {
			case !ok:
				others++
				if g.guessed() {
					guessed++
				}
			case !g.guessed():
				silent++
			case g.First <= tr[1] && g.Last >= tr[0]:
				right++
			default:
				wrong++
			}
		}
		t.Logf("private %d (%d scenes, %d passages, %d linked): links out: linked ones %d right, %d wrong, %d silent; the other %d: %d guessed",
			k, len(scenes), len(ps), len(truth), right, wrong, silent, others, guessed)
		found, holding := 0, map[int]bool{}
		for i, l := range m.links {
			if m.matches[i].found() {
				found++
			}
			for s, sc := range scenes {
				if l.Line >= sc.Line && (s+1 == len(scenes) || l.Line < scenes[s+1].Line) {
					holding[s] = true
				}
			}
		}
		t.Logf("private %d: %d links, %d found in the source, in %d scenes", k, len(m.links), found, len(holding))
		gs = alignScenes(scenes, ps, truth)
		guessed, gap, last, order := 0, 0, -1, true
		at := map[int]int{}
		for _, g := range gs {
			if g.guessed() {
				guessed++
				at[g.First]++
				if g.Gap {
					gap++
				}
			}
			if g.First >= 0 {
				order = order && g.First >= last
				last = g.First
			}
		}
		shared := 0
		for _, c := range at {
			if c > 1 {
				shared += c
			}
		}
		t.Logf("private %d: links in as anchors: %d of the other %d guessed (%d by the order, %d in a gap between; %d share a first passage); in the source's order: %v",
			k, guessed, others, guessed-gap, gap, shared, order)
	}
}

// An invented scene between two the aligner is sure of, close together
// in the source: still no guess (the gap's second look needs evidence of
// its own), and the others as they were.
func TestAlignInventedBetween(t *testing.T) {
	dir := filepath.Join("testdata", "adapt-bench", "christmas-carol")
	src, _ := os.ReadFile(filepath.Join(dir, "source.md"))
	b, _ := os.ReadFile(filepath.Join(dir, "script.fountain"))
	script := string(b)
	invented := ".ACT II SCENE IVB - A TAVERN\n\nA darts match, loud and long. Nobody here has read the story.\n\nPUBLICAN\nAnother round for the table by the window, and mind the dog.\n\n"
	i := strings.Index(script, ".ACT II SCENE V - A CHAMBER")
	if i < 0 {
		t.Fatal("no ACT II SCENE V")
	}
	ps := passagesOf(string(src))
	starts := []string{"INT.", "EXT.", "INT./EXT.", "I/E"}
	before := alignScenes(scenesOf(script, starts), ps, nil)
	after := alignScenes(scenesOf(script[:i]+invented+script[i:], starts), ps, nil)
	k := slices.IndexFunc(after, func(g sceneGuess) bool { return strings.Contains(g.Heading, "IVB") })
	if k < 0 || after[k].guessed() {
		t.Fatalf("the invented scene: %+v", after[max(k, 0)])
	}
	for j, g := range before {
		h := after[j]
		if j >= k {
			h = after[j+1]
		}
		if g.First != h.First || g.Last != h.Last {
			t.Errorf("%s: %d–%d, was %d–%d", g.Heading, h.First, h.Last, g.First, g.Last)
		}
	}
}
