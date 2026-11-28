// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"math"
	"slices"
	"sync"

	"github.com/LaPingvino/accolade/internal/pace"
)

// Scene alignment: where in the source each of the script's scenes
// comes from, read off the words alone (a script written before it had
// links, or a free adaptation). The scenes are placed in order along the
// source, so one with little of its own to go on is held by its
// neighbours, and a scene is only guessed for when it has evidence of
// its own: a guess must be confident, an invented scene gets none.
//
// A scene against a window of the source (its passages from u, L of
// them): the rarer words they share, each once (its weight log N/df),
// against what a window of that length shares with the scene by chance
// (a word in df of the N passages is in L of them with p = 1-(1-df/N)^L):
// a z-score. Each scene takes the window length that suits it best
// (from a passage to a chapter's worth), and its z at each start u.
//
// Order: the scenes' starts go along the source (a scene may start where
// the one before does: two scenes from one passage), any scene may be
// left out (an invented one), and the best placing of them all is the
// one with the most z above a floor (floorFor), found by dynamic
// programming. How sure a scene's place is: the best total with it
// there, against the best with it left out or placed elsewhere (beyond
// its window): that margin is its confidence; below alignSure it is
// silent. Then a scene left out between two placed ones whose gap is
// short is looked for again inside the gap alone, with a floor for the
// fewer places there are to look (the chance best of fewer is lower).
//
// Scenes linked already are fixed where their links are: the others
// fall in order around them.

// sceneGuess is where a scene seems to come from.
type sceneGuess struct {
	Scene       int // which of the script's scenes (scenesOf)
	Heading     string
	Line        int
	First, Last int     // the passages (-1: no guess)
	Z           float64 // its own evidence
	Sure        float64 // its margin over the next best placing
	Linked      bool    // placed by its links, not guessed
	Gap         bool    // placed in the gap between two placed scenes (the second look)
	Unordered   bool    // placed out of the source's order (the third look: a flash-forward, a scene moved)
}

func (g sceneGuess) guessed() bool { return g.First >= 0 && !g.Linked }

// alignFloor is what a scene's window must reach to be placed, in z
// above the best that chance gives among the places looked at (about
// √(2 ln places) for that many windows): over the whole source of a
// novel, about 5.5; between two scenes placed near each other, less.
const alignFloor = 1.0

// floorFor is the z a scene must reach among places starts.
func floorFor(places int) float64 {
	return alignFloor + math.Sqrt(2*math.Log(float64(max(places*len(alignWindows), 2))))
}

// alignSure is the margin (in z) a placing must win by to be shown.
const alignSure = 2.0

// alignUnorderedFloor: a scene placed out of the source's order needs
// this many times the floor, and as many times alignSure over any other
// place (the order holds it nowhere: only its own words do).
const alignUnorderedFloor = 2.0

// alignOutSlack: against leaving a scene out, its placing needs less
// margin than against another place (it is above the floor already:
// evidence of its own).
const alignOutSlack = 1.5

// alignReach is how far (in passages, at least; a fortieth of a long
// source) another placing must be to count as another place: one nearer
// is the same guess, roughly.
const alignReach = 20

// alignWindows are the window lengths tried, in passages.
var alignWindows = []int{1, 3, 8, 20, 50}

// sceneScores is one scene against the source.
type sceneScores struct {
	z   []float64 // the best z of a window starting at each counted passage
	len []int     // that window's length
}

// alignScenes places the scenes along the source: a guess for each
// scene, in the scenes' order (First -1: none). anchors are the scenes
// linked already, by scene: the passages their links find.
func alignScenes(scenes []sceneBody, ps []passage, anchors map[int][2]int) []sceneGuess {
	out := make([]sceneGuess, len(scenes))
	for k, sc := range scenes {
		out[k] = sceneGuess{Scene: k, Heading: sc.Heading, Line: sc.Line, First: -1, Last: -1}
	}
	src := alignSourceFor(ps)
	at, df, where, cum, place := src.at, src.df, src.where, src.cum, src.place
	n := len(at)
	if n == 0 || len(scenes) == 0 {
		return out
	}
	common := func(w string) bool { return n >= commonFrom && df[w] > n/4 }
	scores := make([]sceneScores, len(scenes))
	for k, sc := range scenes {
		var ws []string
		for _, w := range sortedKeys(sc.Words) {
			if df[w] > 0 && !common(w) {
				ws = append(ws, w)
			}
		}
		scores[k] = scoreScene(ws, df, where, n, cum)
		pace.Yield()
	}
	// anchors: fixed
	fixed := map[int]int{}
	for k, a := range anchors {
		if k < len(scenes) && a[0] >= 0 && a[0] < len(ps) {
			fixed[k] = place[a[0]]
			out[k].First, out[k].Last, out[k].Linked = a[0], a[1], true
		}
	}
	floor := floorFor(n)
	gain := func(k, u int) float64 {
		if f, ok := fixed[k]; ok {
			if u == f {
				return 1e6 // only there
			}
			return math.Inf(-1)
		}
		return scores[k].z[u] - floor
	}
	starts, sure := alignOrder(len(scenes), n, gain, func(k, u int) int { return max(scores[k].len[u], alignReach, n/40) })
	for k := range scenes {
		if _, ok := fixed[k]; ok || starts[k] < 0 {
			continue
		}
		out[k].Sure, out[k].Z = sure[k], scores[k].z[starts[k]]
		if sure[k] < alignSure {
			continue
		}
		u := starts[k]
		setGuess(&out[k], scores[k], u, at, ps, scenes[k].Words)
	}
	// plain: a scene's best place in the whole source when its words
	// alone point there, well above the floor and well ahead of any other
	// place (the order holds it nowhere)
	plain := func(k int) (bu int, best, margin float64, ok bool) {
		best, bu = math.Inf(-1), -1
		for u := range n {
			if z := scores[k].z[u]; z > best {
				best, bu = z, u
			}
		}
		if bu < 0 || best < alignUnorderedFloor*floor {
			return -1, 0, 0, false
		}
		r := max(scores[k].len[bu], alignReach, n/40)
		second := 0.0
		for u := range n {
			if u < bu-r || u > bu+r {
				second = max(second, scores[k].z[u])
			}
		}
		return bu, best, best - second, best-second >= alignUnorderedFloor*alignSure
	}
	// left out, between two placed ones: looked for again in the gap
	for changed := true; changed; {
		changed = false
		for k := range scenes {
			if out[k].First >= 0 {
				continue
			}
			lo, hi := 0, n-1
			prev, next := -1, -1
			for j := k - 1; j >= 0; j-- {
				if out[j].First >= 0 {
					prev = j
					lo = place[out[j].First]
					break
				}
			}
			for j := k + 1; j < len(scenes); j++ {
				if out[j].First >= 0 {
					next = j
					hi = place[out[j].First]
					break
				}
			}
			if prev < 0 || next < 0 || hi < lo {
				continue // only between two
			}
			places := hi - lo + 1
			if places > n/4 && places > 2*alignWindows[len(alignWindows)-1] {
				continue // a long gap: as good as the whole source
			}
			best, second, bu := math.Inf(-1), math.Inf(-1), -1
			for u := lo; u <= hi; u++ {
				z := scores[k].z[u]
				switch {
				case z > best:
					if bu < 0 || u-bu > scores[k].len[bu] {
						second = best
					}
					best, bu = z, u
				case z > second && u-bu > scores[k].len[bu]:
					second = z
				}
			}
			floor := floorFor(places)
			if bu < 0 || best < floor || best-max(second, 0) < alignSure/2 {
				continue
			}
			// its words point plainly to one place outside the gap (a
			// scene moved, for the third look): not from here
			if u, _, _, ok := plain(k); ok && (u < lo || u > hi) {
				continue
			}
			setGuess(&out[k], scores[k], bu, at, ps, scenes[k].Words)
			out[k].Sure, out[k].Gap = best-max(second, 0), true
			changed = true
		}
	}
	// still left out, but plainly from one place, out of the order: a
	// scene moved (a flash-forward, a prologue from the end) is no
	// invented one
	for k := range scenes {
		if out[k].First >= 0 {
			continue
		}
		if u, z, margin, ok := plain(k); ok {
			setGuess(&out[k], scores[k], u, at, ps, scenes[k].Words)
			out[k].Z, out[k].Sure, out[k].Unordered = z, margin, true
		}
	}
	for k := range out {
		if out[k].guessed() && out[k].Sure == 0 {
			out[k].Sure = sure[k]
		}
	}
	return out
}

// alignSource is what the aligner reads off the source alone, kept
// while the passages stay (the script changes as it is typed, the
// source seldom).
type alignSource struct {
	ps    []passage
	at    []int            // the counted passages: the places a scene can start
	df    map[string]int   // how many of them each word is in
	where map[string][]int // and which (in at's terms)
	cum   []float64        // their sizes summed (words against the average)
	place []int            // where each passage is among the counted ones
}

var lastAlignSource struct {
	sync.Mutex
	src *alignSource
}

func alignSourceFor(ps []passage) *alignSource {
	lastAlignSource.Lock()
	defer lastAlignSource.Unlock()
	if s := lastAlignSource.src; s != nil && samePassages(s.ps, ps) {
		return s
	}
	s := &alignSource{ps: ps, df: map[string]int{}, where: map[string][]int{}}
	for i, p := range ps {
		if p.counted() {
			s.at = append(s.at, i)
		}
	}
	n := len(s.at)
	seen := map[string]int{}
	for u, i := range s.at {
		for _, w := range ps[i].Words {
			if len([]rune(w)) > 2 && seen[w] != u+1 {
				seen[w] = u + 1
				s.df[w]++
				s.where[w] = append(s.where[w], u)
			}
		}
	}
	s.cum = make([]float64, n+1)
	for u, i := range s.at {
		s.cum[u+1] = s.cum[u] + float64(len(ps[i].Words))
	}
	if n > 0 {
		avg := max(s.cum[n]/float64(n), 1)
		for u := range s.cum {
			s.cum[u] /= avg
		}
	}
	s.place = make([]int, len(ps))
	for u, i := range s.at {
		s.place[i] = u
	}
	for i := range ps { // a heading: the passage after it
		if !ps[i].counted() && n > 0 {
			s.place[i] = sort0(s.at, i)
		}
	}
	lastAlignSource.src = s
	return s
}

// sort0 is the first of at (sorted) at or after i, as an index into at
// (the last if none).
func sort0(at []int, i int) int {
	u, _ := slices.BinarySearch(at, i)
	return min(u, len(at)-1)
}

// setGuess writes a scene's guess: its window from u, kept to the part
// with its rarer words in (the first and last passage sharing one).
func setGuess(g *sceneGuess, s sceneScores, u int, at []int, ps []passage, words map[string]bool) {
	l := s.len[u]
	end := min(u+l, len(at))
	first, last := -1, -1
	best := 0
	hits := make([]int, end-u)
	for v := u; v < end; v++ {
		seen := map[string]bool{}
		for _, w := range ps[at[v]].Words {
			if words[w] && !seen[w] && len([]rune(w)) > 2 {
				seen[w] = true
				hits[v-u]++
			}
		}
		best = max(best, hits[v-u])
	}
	for v := u; v < end; v++ {
		if hits[v-u]*2 >= best && best > 0 {
			if first < 0 {
				first = v
			}
			last = v
		}
	}
	if first < 0 {
		first, last = u, u
	}
	g.First, g.Last, g.Z = at[first], at[last], s.z[u]
}

// scoreScene is a scene's z for a window from each start, the best of
// the lengths tried. cum is the passages' sizes summed (cum[u]: those
// before u), each passage's its count of words against the average: a
// long passage holds a word more likely than a short one, so the chance
// share of a window goes by its size, not by how many passages it has.
func scoreScene(ws []string, df map[string]int, where map[string][]int, n int, cum []float64) sceneScores {
	s := sceneScores{z: make([]float64, n), len: make([]int, n)}
	for u := range s.z {
		s.z[u], s.len[u] = math.Inf(-1), 1
	}
	type wordOdds struct{ idf, lq float64 } // lq: log of the chance a passage of average size misses it
	var odds []wordOdds
	for _, w := range ws {
		d := float64(df[w])
		if idf := math.Log(float64(n) / d); idf > 0 {
			odds = append(odds, wordOdds{idf, math.Log1p(-d / float64(n))})
		}
	}
	if len(odds) == 0 {
		for u := range s.z {
			s.z[u] = 0
		}
		return s
	}
	// the chance mean and spread of a window by its size, on a grid of
	// sizes (log steps), read between
	const steps = 8 // per doubling
	maxSize := cum[n] + 1
	grid := int(math.Ceil(math.Log2(maxSize/0.01)*steps)) + 2
	means, sds := make([]float64, grid), make([]float64, grid)
	for j := range grid {
		e := 0.01 * math.Exp2(float64(j)/steps)
		var mean, v float64
		for _, o := range odds {
			p := -math.Expm1(o.lq * e)
			mean += o.idf * p
			v += o.idf * o.idf * p * (1 - p)
		}
		means[j], sds[j] = mean, math.Sqrt(max(v, 1e-9))
	}
	chance := func(e float64) (float64, float64) {
		x := math.Log2(max(e, 0.01)/0.01) * steps
		j := min(int(x), grid-2)
		f := x - float64(j)
		return means[j]*(1-f) + means[j+1]*f, sds[j]*(1-f) + sds[j+1]*f
	}
	mass := make([]float64, n+1)
	for _, L := range alignWindows {
		if L > n && L != alignWindows[0] {
			break
		}
		L = min(L, n)
		clear(mass)
		for _, w := range ws {
			d := float64(df[w])
			idf := math.Log(float64(n) / d)
			if idf <= 0 {
				continue
			}
			// the starts whose window holds w: (j-L, j] for each place j,
			// as a union of runs
			runLo, runHi := -1, -1
			for _, j := range where[w] {
				a, b := max(j-L+1, 0), j
				if runLo >= 0 && a <= runHi+1 {
					runHi = b
					continue
				}
				if runLo >= 0 {
					mass[runLo] += idf
					mass[runHi+1] -= idf
				}
				runLo, runHi = a, b
			}
			if runLo >= 0 {
				mass[runLo] += idf
				mass[runHi+1] -= idf
			}
		}
		acc := 0.0
		for u := 0; u < n; u++ {
			acc += mass[u]
			mean, sd := chance(cum[min(u+L, n)] - cum[u])
			if z := (acc - mean) / sd; z > s.z[u] {
				s.z[u], s.len[u] = z, L
			}
		}
	}
	return s
}

// alignOrder places m scenes at starts 0..n-1 in order (a start no
// earlier than the one before), or leaves one out (-1), for the most
// gain(k, u) in all (a scene is placed only where its gain is above 0).
// sure is each placed scene's margin: the best total with it where it
// is, against the best with it left out or more than reach(k, u) away.
func alignOrder(m, n int, gain func(k, u int) float64, reach func(k, u int) int) (starts []int, sure []float64) {
	// F[k][u]: the best of scenes 0..k with the last placed at or
	// before u (u = -1 is index 0: none placed yet), a running max.
	inf := math.Inf(-1)
	F := make([][]float64, m+1)
	F[0] = make([]float64, n+1) // none: 0 everywhere
	for k := 0; k < m; k++ {
		row := make([]float64, n+1)
		row[0] = F[k][0] // k left out, none placed
		for u := 0; u < n; u++ {
			v := F[k][u+1] // k left out, the last placed at or before u
			if g := gain(k, u); g > 0 {
				v += g // k at u, the ones before at or before u
			}
			row[u+1] = max(v, row[u])
		}
		F[k+1] = row
		pace.Yield()
	}
	// B[k][u]: the best of scenes k..m-1 with all placed at or after u
	// (u = n: none can be), a running max from the right.
	B := make([][]float64, m+1)
	B[m] = make([]float64, n+1)
	for k := m - 1; k >= 0; k-- {
		row := make([]float64, n+1)
		row[n] = B[k+1][n]
		for u := n - 1; u >= 0; u-- {
			v := B[k+1][u] // k left out
			if g := gain(k, u); g > 0 {
				v += g
			}
			row[u] = max(v, row[u+1])
		}
		B[k] = row
	}
	// with k at u: the best before at or before u, k, the best after at
	// or after u; with k left out: the best split of the rest
	starts = make([]int, m)
	sure = make([]float64, m)
	through := make([]float64, n)
	for k := 0; k < m; k++ {
		best, bu := inf, -1
		for u := 0; u < n; u++ {
			g := gain(k, u)
			through[u] = inf
			if g > 0 {
				through[u] = F[k][u+1] + g + B[k+1][u]
				if through[u] > best {
					best, bu = through[u], u
				}
			}
		}
		out := inf // k left out
		for u := 0; u <= n; u++ {
			out = max(out, F[k][u]+B[k+1][u])
		}
		starts[k] = -1
		if bu < 0 || best <= out {
			continue
		}
		// the hill bu is on: the starts around it above the floor, with
		// gaps shorter than reach (a long scene's window fits all along
		// its stretch: that is the same place)
		r := reach(k, bu)
		lo, hi := bu, bu
		for u, gap := bu-1, 0; u >= 0 && gap <= r; u-- {
			if gain(k, u) > 0 {
				lo, gap = u, 0
			} else {
				gap++
			}
		}
		for u, gap := bu+1, 0; u < n && gap <= r; u++ {
			if gain(k, u) > 0 {
				hi, gap = u, 0
			} else {
				gap++
			}
		}
		far := out - alignOutSlack
		for u := 0; u < n; u++ {
			if u < lo-r || u > hi+r {
				far = max(far, through[u])
			}
		}
		starts[k], sure[k] = bu, best-far
	}
	return starts, sure
}

// guessScenes is alignScenes for a script read against its source (m):
// the scenes linked already are its anchors.
func guessScenes(text string, starts []string, m *sourceModel) []sceneGuess {
	scenes := scenesOf(text, starts)
	return alignScenes(scenes, m.passages, sceneAnchors(scenes, m))
}

// sceneAnchors are the passages each linked scene's links find: the
// first and the last of them.
func sceneAnchors(scenes []sceneBody, m *sourceModel) map[int][2]int {
	anchors := map[int][2]int{}
	for k, sc := range scenes {
		end := math.MaxInt
		if k+1 < len(scenes) {
			end = scenes[k+1].Line
		}
		lo, hi := -1, -1
		for i, l := range m.links {
			mt := m.matches[i]
			if l.Line < sc.Line || l.Line >= end || !mt.found() {
				continue
			}
			if lo < 0 || mt.Passages[0] < lo {
				lo = mt.Passages[0]
			}
			hi = max(hi, mt.last())
		}
		if lo >= 0 {
			anchors[k] = [2]int{lo, hi}
		}
	}
	return anchors
}
