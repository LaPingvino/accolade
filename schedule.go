// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"cmp"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"time"

	"github.com/LaPingvino/lexington/lex"
	lexreport "github.com/LaPingvino/lexington/report"
)

// The shooting schedule (docs/PRODUCTION.md §14, step 8): the shoot days
// in order, each with its date, call time and location notes and its
// scenes in shooting order; the scenes on no day are the unscheduled
// pool. Kept in assets/production/schedule.json, in the script's extras
// as the breakdown is (the bundle's, else the kept copy, else
// Name.production/ beside the script), never in the text:
//
//	{"version": 1,
//	 "days": [{"id": "5f1c…", "date": "2026-12-01", "call": "07:00",
//	           "notes": "Kovač house, the kitchen",
//	           "strips": [{"scene": "12", "heading": "int kitchen night",
//	                       "start": "anna makes tea rain on the window",
//	                       "sum": "1c9e…", "set": "Blue"}]}],
//	 "pool": "location"}
//
// A strip is its scene, named as a breakdown tag's scene is (its written
// number, its heading's words and which of the scenes with that heading
// it is) and by its first words, so a scene keeps its strip when a scene
// is put before it, when its number is locked after scheduling, and when
// its heading changes. Its sum is the scene's words when it was put on
// the day (or checked): a scene changed since is flagged on its strip,
// with the revision it changed in. A strip whose scene is gone stays in
// the file, shown as lost.

const scheduleName = productionDir + "schedule.json"

type scheduleFile struct {
	Version int        `json:"version"`
	Days    []schedDay `json:"days"`
	// Pool is how the unscheduled strips are sorted: "" script order,
	// "location", "cast".
	Pool string `json:"pool,omitempty"`
}

// schedDay is a shoot day.
type schedDay struct {
	ID     string       `json:"id"`
	Date   string       `json:"date,omitempty"` // 2006-01-02
	Call   string       `json:"call,omitempty"` // 07:00
	Notes  string       `json:"notes,omitempty"`
	Strips []schedStrip `json:"strips"`
}

// schedStrip is a scene on a day.
type schedStrip struct {
	bdSceneKey
	Start string `json:"start,omitempty"` // its first words after the heading, normalised
	Sum   string `json:"sum,omitempty"`   // its words when put on the day or checked
	Set   string `json:"set,omitempty"`   // the revision then (the open set, or the last issued)
}

// date is the day's date (zero for none).
func (d schedDay) date() time.Time {
	t, err := time.Parse("2006-01-02", d.Date)
	if err != nil {
		return time.Time{}
	}
	return t
}

// readSchedule reads the extras' schedule (an empty one if none).
func readSchedule(r extrasReader) (scheduleFile, error) {
	f := scheduleFile{Version: 1}
	data, err := r.readExtra(scheduleName)
	if errors.Is(err, fs.ErrNotExist) || (err == nil && len(data) == 0) {
		return f, nil
	}
	if err != nil {
		return f, err
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return scheduleFile{Version: 1}, fmt.Errorf("%s: %v", scheduleName, err)
	}
	return f, nil
}

// startWords is how many of a scene's first words a strip keeps.
const startWords = 8

// sceneBody are the words of scene sc after its heading's line, [i, j).
func (t *bdText) sceneBody(sc int) (int, int) {
	lo, hi := t.sceneWords(sc)
	next := len(t.runes)
	if l := t.scenes[sc].line + 1; l < len(t.lineStart) {
		next = t.lineStart[l]
	}
	for lo < hi && t.words[lo].start < next {
		lo++
	}
	return lo, hi
}

// sceneStart is a scene's first words, normalised.
func (t *bdText) sceneStart(sc int) string {
	lo, hi := t.sceneBody(sc)
	var ws []string
	for i := lo; i < hi && len(ws) < startWords; i++ {
		ws = append(ws, t.words[i].norm)
	}
	return strings.Join(ws, " ")
}

// sceneSum is a scene's words, heading included, as a short hash: the
// same words (whatever the spaces and line breaks), the same sum.
func (t *bdText) sceneSum(sc int) string {
	lo, hi := t.sceneWords(sc)
	return wordsSum(t.scenes[sc].key.Heading, t.words[lo:hi])
}

// oldSceneSum is sceneSum as it was before the words left out notes and
// source links: a strip summed then is not changed if that matches.
func (t *bdText) oldSceneSum(sc int) string {
	s := t.scenes[sc]
	return wordsSum(s.key.Heading, wordsIn(t.runes, s.start, min(s.end, len(t.runes))))
}

func wordsSum(heading string, ws []bdWord) string {
	h := sha1.New()
	h.Write([]byte(heading))
	for _, w := range ws {
		h.Write([]byte{0})
		h.Write([]byte(w.norm))
	}
	return fmt.Sprintf("%x", h.Sum(nil)[:8])
}

// stripOf is scene sc's strip as put on a day now, in revision set.
func (t *bdText) stripOf(sc int, set string) schedStrip {
	return schedStrip{bdSceneKey: t.scenes[sc].key, Start: t.sceneStart(sc), Sum: t.sceneSum(sc), Set: set}
}

// schedPlaced is a schedule found in a text: each day's strips' scenes
// (-1: lost), and each scene's day (-1: the pool).
type schedPlaced struct {
	days  [][]int
	dayOf []int
	at    []int // each scene's place in its day
	lost  []schedStrip
	dups  int // strips of a scene already on a day (left out)
}

// find finds a strip's scene among those not taken: by its number, else
// among the scenes with its heading (the one with its first words, else
// which of those it was), else the only one with its first words.
func (t *bdText) find(s schedStrip, taken []bool) int {
	free := func(i int) bool { return !taken[i] }
	if s.Scene != "" {
		for i, sc := range t.scenes {
			if sc.key.Scene == s.Scene && free(i) {
				return i
			}
		}
	}
	var same []int
	for i, sc := range t.scenes {
		if sc.key.Heading == s.Heading && free(i) && (s.Scene == "" || sc.key.Scene == "" || sc.key.Scene == s.Scene) {
			same = append(same, i)
		}
	}
	starts := func(list []int) []int {
		var out []int
		for _, i := range list {
			if s.Start != "" && t.sceneStart(i) == s.Start {
				out = append(out, i)
			}
		}
		return out
	}
	if m := starts(same); len(m) == 1 {
		return m[0]
	} else if len(m) > 1 {
		same = m
	} else if i := t.onlyStart(s, taken); i >= 0 {
		return i // its heading changed, and another has its old one
	}
	for _, i := range same {
		if t.scenes[i].key.Nth == s.Nth {
			return i
		}
	}
	if len(same) == 1 {
		return same[0]
	}
	// its heading changed: its first words, if no other scene starts so
	return t.onlyStart(s, taken)
}

// onlyStart is the only scene not taken that starts with the strip's
// first words (three at least), -1 none or several.
func (t *bdText) onlyStart(s schedStrip, taken []bool) int {
	if len(strings.Fields(s.Start)) < 3 {
		return -1
	}
	found := -1
	for i := range t.scenes {
		if !taken[i] && t.sceneStart(i) == s.Start {
			if found >= 0 {
				return -1
			}
			found = i
		}
	}
	return found
}

// place finds the schedule's strips in the text.
func (f *scheduleFile) place(t *bdText) schedPlaced {
	p := schedPlaced{days: make([][]int, len(f.Days)), dayOf: make([]int, len(t.scenes)), at: make([]int, len(t.scenes))}
	for i := range p.dayOf {
		p.dayOf[i] = -1
	}
	taken := make([]bool, len(t.scenes))
	// numbered strips first: a number is surer than a heading
	type ref struct{ d, k int }
	var order []ref
	for d, day := range f.Days {
		p.days[d] = make([]int, len(day.Strips))
		for k, s := range day.Strips {
			if s.Scene != "" {
				order = append(order, ref{d, k})
			}
		}
	}
	for d, day := range f.Days {
		for k, s := range day.Strips {
			if s.Scene == "" {
				order = append(order, ref{d, k})
			}
		}
	}
	for _, r := range order {
		s := f.Days[r.d].Strips[r.k]
		sc := t.find(s, taken)
		if sc < 0 {
			// a scene on a day already: the strip twice; else lost
			if t.find(s, make([]bool, len(t.scenes))) >= 0 {
				p.dups++
			} else {
				p.lost = append(p.lost, s)
			}
			p.days[r.d][r.k] = -1
			continue
		}
		taken[sc] = true
		p.days[r.d][r.k] = sc
	}
	for d, day := range p.days {
		n := 0
		for _, sc := range day {
			if sc >= 0 {
				p.dayOf[sc], p.at[sc] = d, n
				n++
			}
		}
	}
	return p
}

// scenesOf are a day's scenes found, in shooting order.
func (p schedPlaced) scenesOf(d int) []int {
	var out []int
	for _, sc := range p.days[d] {
		if sc >= 0 {
			out = append(out, sc)
		}
	}
	return out
}

// schedCount is how far the schedule is: the scenes on a day of those to
// shoot (OMITTED ones aside), and the days.
type schedCount struct {
	scheduled, total, days int
}

func (p schedPlaced) count(t *bdText) schedCount {
	c := schedCount{days: len(p.days)}
	for i, s := range t.scenes {
		if s.omitted {
			continue
		}
		c.total++
		if p.dayOf[i] >= 0 {
			c.scheduled++
		}
	}
	return c
}

// pool are the scenes on no day (OMITTED ones aside), in script order or
// sorted as the pool's sort says (strips: the report's, by scene; nil:
// script order).
func (f *scheduleFile) pool(t *bdText, p schedPlaced, strips []*lexreport.Strip) []int {
	var out []int
	for i, s := range t.scenes {
		if !s.omitted && p.dayOf[i] < 0 {
			out = append(out, i)
		}
	}
	sortStrips(out, f.Pool, strips)
	return out
}

// sortStrips sorts scenes (stable) by location (then INT/EXT, D/N) or by
// cast (the lowest cast numbers first: the lead's scenes together, then
// by location): the stripboard's one-click suggestions.
func sortStrips(scenes []int, by string, strips []*lexreport.Strip) {
	if by == "" || strips == nil {
		return
	}
	get := func(i int) *lexreport.Strip {
		if i < len(strips) && strips[i] != nil {
			return strips[i]
		}
		return &lexreport.Strip{}
	}
	first := map[string]int{} // a location's first place, so groups keep the script's order
	for k, i := range scenes {
		if _, ok := first[locKey(get(i).Location)]; !ok {
			first[locKey(get(i).Location)] = k
		}
	}
	byLoc := func(a, b *lexreport.Strip) int {
		return cmp.Or(cmp.Compare(first[locKey(a.Location)], first[locKey(b.Location)]),
			strings.Compare(a.IE(), b.IE()), strings.Compare(a.DN(), b.DN()))
	}
	slices.SortStableFunc(scenes, func(x, y int) int {
		a, b := get(x), get(y)
		if by == "cast" {
			if c := slices.Compare(castKey(a), castKey(b)); c != 0 {
				return c
			}
		}
		return byLoc(a, b)
	})
}

// castKey orders a strip by its cast: its numbers, none last.
func castKey(s *lexreport.Strip) []int {
	if len(s.CastNumbers) == 0 {
		return []int{1 << 30}
	}
	return s.CastNumbers
}

func locKey(s string) string { return strings.ToUpper(strings.Join(strings.Fields(s), " ")) }

// The schedule's changes. Each works on the file and the strips as found
// (p, of text t), and leaves the file to be written.

// newDay adds a day at the end: the day after the last dated one, when
// there is one.
func (f *scheduleFile) newDay() int {
	d := schedDay{ID: newID()[:8], Strips: []schedStrip{}}
	for i := len(f.Days) - 1; i >= 0; i-- {
		if t := f.Days[i].date(); !t.IsZero() {
			d.Date = t.AddDate(0, 0, 1).Format("2006-01-02")
			break
		}
	}
	f.Days = append(f.Days, d)
	return len(f.Days) - 1
}

// removeDay takes a day away; its strips go back to the pool.
func (f *scheduleFile) removeDay(d int) {
	if d >= 0 && d < len(f.Days) {
		f.Days = slices.Delete(f.Days, d, d+1)
	}
}

// take takes scene sc's strip off its day (as found); it gives the strip
// it had (ok false: it was in the pool).
func (f *scheduleFile) take(p schedPlaced, sc int) (schedStrip, bool) {
	d := p.dayOf[sc]
	if d < 0 {
		return schedStrip{}, false
	}
	for k, x := range p.days[d] {
		if x == sc {
			s := f.Days[d].Strips[k]
			f.Days[d].Strips = slices.Delete(f.Days[d].Strips, k, k+1)
			return s, true
		}
	}
	return schedStrip{}, false
}

// move puts scene sc on day d (-1: the pool) before its at-th found strip
// (past the end: last). A strip moved keeps its sum and set (a change
// stays flagged); one from the pool gets the scene's as it is now.
func (f *scheduleFile) move(t *bdText, p schedPlaced, sc, d, at int, set string) {
	if sc < 0 || sc >= len(t.scenes) || d >= len(f.Days) {
		return
	}
	s, had := f.take(p, sc)
	if !had {
		s = t.stripOf(sc, set)
	} else { // its scene as it is named now (a number locked since): its sum and set kept
		s.bdSceneKey, s.Start = t.scenes[sc].key, t.sceneStart(sc)
	}
	if d < 0 {
		return
	}
	// at counts the found strips: its index among all (the lost kept)
	strips := f.Days[d].Strips
	k, n := 0, 0
	day := p.days[d]
	if p.dayOf[sc] == d { // taken out of this day: its place gone
		day = slices.DeleteFunc(slices.Clone(day), func(x int) bool { return x == sc })
	}
	for k = 0; k < len(day) && n < at; k++ {
		if day[k] >= 0 {
			n++
		}
	}
	k = min(k, len(strips))
	f.Days[d].Strips = slices.Insert(strips, k, s)
}

// shift moves scene sc's strip one up (-1) or down (+1) in the board's
// order (the days one after the other, then the pool): over the strip
// next to it, or over a day break into the day before or after (from the
// pool's top up: to the last day's end; a day's last down into the next
// day's start, the last day's into the pool). It tells whether it moved.
func (f *scheduleFile) shift(t *bdText, p schedPlaced, sc, dir int, set string, pool []int) bool {
	d := p.dayOf[sc]
	if d < 0 { // in the pool: up into the last day
		if dir < 0 && len(f.Days) > 0 && len(pool) > 0 && pool[0] == sc {
			last := len(f.Days) - 1
			f.move(t, p, sc, last, len(p.scenesOf(last)), set)
			return true
		}
		return false
	}
	day := p.scenesOf(d)
	at := p.at[sc]
	switch {
	case dir < 0 && at > 0:
		f.move(t, p, sc, d, at-1, set)
	case dir > 0 && at < len(day)-1:
		f.move(t, p, sc, d, at+1, set)
	case dir < 0 && d > 0:
		f.move(t, p, sc, d-1, len(p.scenesOf(d-1)), set)
	case dir > 0 && d < len(f.Days)-1:
		f.move(t, p, sc, d+1, 0, set)
	case dir > 0: // the last day's last: back to the pool
		f.move(t, p, sc, -1, 0, set)
	default:
		return false
	}
	return true
}

// sortDay sorts a day's strips as sortStrips does (the lost kept last).
func (f *scheduleFile) sortDay(p schedPlaced, d int, by string, strips []*lexreport.Strip) {
	if d < 0 || d >= len(f.Days) {
		return
	}
	scenes := p.scenesOf(d)
	sortStrips(scenes, by, strips)
	byScene := map[int]schedStrip{}
	var lost []schedStrip
	for k, sc := range p.days[d] {
		if sc >= 0 {
			byScene[sc] = f.Days[d].Strips[k]
		} else {
			lost = append(lost, f.Days[d].Strips[k])
		}
	}
	out := make([]schedStrip, 0, len(f.Days[d].Strips))
	for _, sc := range scenes {
		out = append(out, byScene[sc])
	}
	f.Days[d].Strips = append(out, lost...)
}

// check takes a scene's change flag away: its strip gets the scene's
// words as they are now.
func (f *scheduleFile) check(t *bdText, p schedPlaced, sc int, set string) {
	d := p.dayOf[sc]
	if d < 0 {
		return
	}
	for k, x := range p.days[d] {
		if x == sc {
			f.Days[d].Strips[k] = t.stripOf(sc, set)
		}
	}
}

// prune drops the strips of a scene on a day already (the second of
// two), keeping the lost.
func (f *scheduleFile) prune(t *bdText) {
	p := f.place(t)
	for d := range f.Days {
		var keep []schedStrip
		for k, s := range f.Days[d].Strips {
			if p.days[d][k] >= 0 || slices.Contains(p.lost, s) {
				keep = append(keep, s)
			}
		}
		if keep == nil {
			keep = []schedStrip{}
		}
		f.Days[d].Strips = keep
	}
}

// changedScene is a scheduled scene whose words changed since its strip
// was put on its day (or checked): the revision it changed in, the
// latest mark in it newer than the strip's set ("" none: changed before
// any, or in no issue's marks).
func (f *scheduleFile) changed(t *bdText, p schedPlaced, sc int) (bool, string) {
	d := p.dayOf[sc]
	if d < 0 {
		return false, ""
	}
	for k, x := range p.days[d] {
		if x == sc {
			s := f.Days[d].Strips[k]
			return s.Sum != "" && s.Sum != t.sceneSum(sc) && s.Sum != t.oldSceneSum(sc), s.Set
		}
	}
	return false, ""
}

// sceneChangedIn is the latest revision set of a scene's lines (marks: all
// issues' marks, a line each) newer than since ("" none).
func sceneChangedIn(t *bdText, sc int, marks []string, since string) string {
	if len(marks) == 0 {
		return ""
	}
	from := t.scenes[sc].line
	to := len(t.lineStart)
	if sc+1 < len(t.scenes) {
		to = t.scenes[sc+1].line
	}
	best, bestN := "", lex.RevisionIndex(since)
	for l := from; l < to && l < len(marks); l++ {
		if m := marks[l]; m != "" {
			if n := lex.RevisionIndex(m); n > bestN || (n < 0 && best == "") {
				best, bestN = m, max(n, bestN)
			}
		}
	}
	return best
}

// dayLength is a day's length: its scenes' eighths (eighths: by scene,
// -1 not measured yet).
func dayLength(scenes []int, eighths []lex.Eighths) lex.Eighths {
	var n lex.Eighths
	for _, sc := range scenes {
		if sc < len(eighths) && eighths[sc] > 0 {
			n += eighths[sc]
		}
	}
	return n
}

// lexSchedule is the schedule as Lexington's reports take it: each day's
// scenes by their report Order (m: the text's scenes to the report's).
func (f *scheduleFile) lexSchedule(p schedPlaced, m []int) lexreport.Schedule {
	var s lexreport.Schedule
	for d, day := range f.Days {
		ld := lexreport.Day{ID: day.ID, Date: day.date(), Call: day.Call, Notes: day.Notes}
		for _, sc := range p.scenesOf(d) {
			if sc < len(m) && m[sc] >= 0 {
				ld.Scenes = append(ld.Scenes, m[sc]+1)
			}
		}
		s.Days = append(s.Days, ld)
	}
	return s
}
