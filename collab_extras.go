package main

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
)

// The breakdown shared while editing together (docs/COLLAB.md, question
// 3, step 4): the breakdown file as a shared map (internal/collab
// maps.go), a key per category, element, tag and scene marked done,
// each key's last writer winning, a tag or element taken away kept as
// gone. Two people tagging different words keep both; Untag on one side
// and a tag on the other are both kept, each on its own words. Every
// change to the breakdown goes through saveBreakdown, which gives the map
// the keys that changed; a peer's change comes in as the whole map read
// back into the window's breakdown and written where its extras are, as
// a change of its own would be.
//
// Only the peer who shares the script puts its breakdown in the map; a
// peer joining (or taking up a kept state) takes the map as it is and
// never puts its own in, so nothing is doubled.

// breakdownMap is the shared map's name.
const breakdownMap = "breakdown"

// bdEntries are a breakdown's keys and values in the shared map.
func bdEntries(f *breakdownFile) map[string]string {
	out := map[string]string{}
	put := func(kind string, key, v any) {
		k, _ := json.Marshal(key)
		val, _ := json.Marshal(v)
		out[kind+" "+string(k)] = string(val)
	}
	for _, c := range f.Categories {
		put("c", c.ID, c)
	}
	for _, e := range f.Elements {
		put("e", e.ID, e)
	}
	for _, t := range f.Tags {
		put("t", t, true)
	}
	for _, d := range f.Done {
		put("d", d, true)
	}
	return out
}

// bdFromEntries is the breakdown the shared map holds, in its keys'
// order (the same on every peer).
func bdFromEntries(m map[string]string) breakdownFile {
	f := breakdownFile{Version: 1}
	for _, k := range slices.Sorted(maps.Keys(m)) {
		kind, key, ok := strings.Cut(k, " ")
		if !ok {
			continue
		}
		switch kind {
		case "c":
			var c bdCategory
			if json.Unmarshal([]byte(m[k]), &c) == nil {
				f.Categories = append(f.Categories, c)
			}
		case "e":
			var e bdElement
			if json.Unmarshal([]byte(m[k]), &e) == nil {
				f.Elements = append(f.Elements, e)
			}
		case "t":
			var t bdTag
			if json.Unmarshal([]byte(key), &t) == nil {
				f.Tags = append(f.Tags, t)
			}
		case "d":
			var d bdSceneKey
			if json.Unmarshal([]byte(key), &d) == nil {
				f.Done = append(f.Done, d)
			}
		}
	}
	// one element made by two at once (the same category and name, each
	// its own id): the first id is the element, the other's tags its own
	same := map[string]string{} // id → the id it is the same as
	first := map[string]string{}
	var elements []bdElement
	for _, e := range f.Elements {
		k := strings.ToLower(e.Category + "\x00" + e.Name)
		if id, ok := first[k]; ok {
			same[e.ID] = id
			continue
		}
		first[k] = e.ID
		elements = append(elements, e)
	}
	if len(same) > 0 {
		f.Elements = elements
		seen := map[string]bool{}
		var tags []bdTag
		for _, t := range f.Tags {
			if id, ok := same[t.Element]; ok {
				t.Element = id
			}
			if k, _ := json.Marshal(t); !seen[string(k)] {
				seen[string(k)] = true
				tags = append(tags, t)
			}
		}
		f.Tags = tags
	}
	return f
}

// shareBreakdownAnew puts the window's breakdown in the shared map: the
// one sharing the script, as it starts.
func (w *MainWindow) shareBreakdownAnew() {
	s := w.shared
	if s == nil || w.markdown {
		return
	}
	f := w.breakdownNow()
	if w.breakdown.err != nil {
		return // not read: nothing of it to share, nor to write over
	}
	if e := bdEntries(f); len(e) > 0 {
		s.session.MapSet(breakdownMap, e, nil)
	}
}

// joinExtras starts a joining window's breakdown and schedule from the
// shared maps, empty while they have nothing: what the window had of its
// own before is not the shared script's, and is never put in the maps.
func (w *MainWindow) joinExtras() {
	s := w.shared
	if s == nil || w.markdown {
		return
	}
	if !s.session.HasMap(breakdownMap) {
		if w.breakdownNow(); w.breakdown.err == nil {
			w.breakdown.file = breakdownFile{Version: 1}
		}
	}
	if !s.session.HasMap(scheduleMap) {
		if w.scheduleNow(); w.schedule.err == nil {
			w.schedule.file, w.schedule.undo = scheduleFile{Version: 1, Days: []schedDay{}}, nil
		}
	}
	w.takeSharedBreakdown()
	w.takeSharedSchedule()
}

// shareBreakdownChange gives the shared map what a change of this
// window's breakdown changed (saveBreakdown).
func (w *MainWindow) shareBreakdownChange(f *breakdownFile) {
	s := w.shared
	if s == nil || w.markdown {
		return
	}
	was := s.session.MapOf(breakdownMap)
	now := bdEntries(f)
	set := map[string]string{}
	for k, v := range now {
		if was[k] != v {
			set[k] = v
		}
	}
	var del []string
	for k := range was {
		if _, ok := now[k]; !ok {
			del = append(del, k)
		}
	}
	slices.Sort(del)
	if len(set) > 0 || len(del) > 0 {
		s.session.MapSet(breakdownMap, set, del)
	}
}

// takeSharedBreakdown makes the shared map the window's breakdown (a
// peer's change, or the map taken up on joining), written where its
// extras are.
func (w *MainWindow) takeSharedBreakdown() {
	s := w.shared
	if s == nil || w.markdown || !s.session.HasMap(breakdownMap) {
		return
	}
	f := bdFromEntries(s.session.MapOf(breakdownMap))
	cur := w.breakdownNow()
	if w.breakdown.err == nil && sameBreakdown(cur, &f) {
		return
	}
	w.breakdown.file = f // a file here that could not be read is not written over (storeBreakdown says so)
	w.storeBreakdown()
}

// sameBreakdown reports whether two breakdowns hold the same.
func sameBreakdown(a, b *breakdownFile) bool { return maps.Equal(bdEntries(a), bdEntries(b)) }

// The schedule shared (the same way): a key per shoot day (its date, call
// and notes, and its place among the days) and one per strip, by its
// scene (the day it is on and its place there), and the pool's order.
// A place is a number between its neighbours' (fractional), so moving
// one strip changes that strip's key only: two people moving strips at
// once each keep their move, a strip moved by both ends on the day of
// the last move, and none is lost or doubled. A strip whose day was
// taken away meanwhile goes back to the pool, as the day's own strips do.

// scheduleMap is the shared map's name.
const scheduleMap = "schedule"

// schedDayEntry and schedStripEntry are a day's and a strip's values.
type schedDayEntry struct {
	Date  string  `json:"date,omitempty"`
	Call  string  `json:"call,omitempty"`
	Notes string  `json:"notes,omitempty"`
	At    float64 `json:"at"`
}

type schedStripEntry struct {
	Day   string  `json:"day"`
	At    float64 `json:"at"`
	Start string  `json:"start,omitempty"`
	Sum   string  `json:"sum,omitempty"`
	Set   string  `json:"set,omitempty"`
}

// placesBetween gives each of n things in order a place: the one it had
// (had[i], ok[i]) if that is still after the one before it and before
// the next kept one; else one between its neighbours.
func placesBetween(had []float64, ok []bool) []float64 {
	n := len(had)
	out := make([]float64, n)
	prev := 0.0
	for i := 0; i < n; i++ {
		next := -1.0 // the next place kept as it was, after prev
		for j := i + 1; j < n; j++ {
			if ok[j] && had[j] > prev {
				next = had[j]
				break
			}
		}
		switch {
		case ok[i] && had[i] > prev && (next < 0 || had[i] < next):
			out[i] = had[i]
		case next < 0:
			out[i] = prev + 1
		default:
			out[i] = prev + (next-prev)/2
		}
		prev = out[i]
	}
	return out
}

// schedEntries are a schedule's keys and values in the shared map, its
// places kept from was (the map as it is) where they still hold.
func schedEntries(f *scheduleFile, was map[string]string) map[string]string {
	out := map[string]string{}
	if f.Pool != "" {
		out["p"] = f.Pool
	}
	had := func(key string, v any) (float64, bool) {
		s, ok := was[key]
		if !ok || json.Unmarshal([]byte(s), v) != nil {
			return 0, false
		}
		switch x := v.(type) {
		case *schedDayEntry:
			return x.At, true
		case *schedStripEntry:
			return x.At, true
		}
		return 0, false
	}
	dayKey := func(d schedDay) string { return "y " + d.ID }
	stripKey := func(s schedStrip) string { k, _ := json.Marshal(s.bdSceneKey); return "s " + string(k) }
	var dh []float64
	var dok []bool
	for _, d := range f.Days {
		at, ok := had(dayKey(d), &schedDayEntry{})
		dh, dok = append(dh, at), append(dok, ok)
	}
	for i, at := range placesBetween(dh, dok) {
		d := f.Days[i]
		v, _ := json.Marshal(schedDayEntry{Date: d.Date, Call: d.Call, Notes: d.Notes, At: at})
		out[dayKey(d)] = string(v)
		var sh []float64
		var sok []bool
		for _, s := range d.Strips {
			var e schedStripEntry
			at, ok := had(stripKey(s), &e)
			sh, sok = append(sh, at), append(sok, ok && e.Day == d.ID)
		}
		for k, at := range placesBetween(sh, sok) {
			s := d.Strips[k]
			v, _ := json.Marshal(schedStripEntry{Day: d.ID, At: at, Start: s.Start, Sum: s.Sum, Set: s.Set})
			out[stripKey(s)] = string(v)
		}
	}
	return out
}

// schedFromEntries is the schedule the shared map holds.
func schedFromEntries(m map[string]string) scheduleFile {
	f := scheduleFile{Version: 1, Days: []schedDay{}, Pool: m["p"]}
	type at struct {
		at  float64
		key string
	}
	less := func(a, b at) int {
		if a.at != b.at {
			if a.at < b.at {
				return -1
			}
			return 1
		}
		return strings.Compare(a.key, b.key)
	}
	var days []at
	dayAt := map[string]schedDayEntry{}
	strips := map[string][]at{}
	stripOf := map[string]schedStrip{}
	for k, v := range m {
		kind, key, _ := strings.Cut(k, " ")
		switch kind {
		case "y":
			var e schedDayEntry
			if json.Unmarshal([]byte(v), &e) == nil {
				dayAt[key] = e
				days = append(days, at{e.At, key})
			}
		case "s":
			var e schedStripEntry
			var sk bdSceneKey
			if json.Unmarshal([]byte(v), &e) == nil && json.Unmarshal([]byte(key), &sk) == nil {
				strips[e.Day] = append(strips[e.Day], at{e.At, key})
				stripOf[key] = schedStrip{bdSceneKey: sk, Start: e.Start, Sum: e.Sum, Set: e.Set}
			}
		}
	}
	slices.SortFunc(days, less)
	for _, d := range days {
		e := dayAt[d.key]
		day := schedDay{ID: d.key, Date: e.Date, Call: e.Call, Notes: e.Notes, Strips: []schedStrip{}}
		ss := strips[d.key]
		slices.SortFunc(ss, less)
		for _, s := range ss {
			day.Strips = append(day.Strips, stripOf[s.key])
		}
		f.Days = append(f.Days, day)
	}
	return f
}

// shareScheduleAnew puts the window's schedule in the shared map: the one
// sharing the script, as it starts.
func (w *MainWindow) shareScheduleAnew() {
	s := w.shared
	if s == nil || w.markdown {
		return
	}
	f := w.scheduleNow()
	if w.schedule.err != nil {
		return
	}
	if e := schedEntries(f, nil); len(e) > 0 {
		s.session.MapSet(scheduleMap, e, nil)
	}
}

// shareScheduleChange gives the shared map what a change of this window's
// schedule changed (writeSchedule).
func (w *MainWindow) shareScheduleChange(f *scheduleFile) {
	s := w.shared
	if s == nil || w.markdown {
		return
	}
	was := s.session.MapOf(scheduleMap)
	now := schedEntries(f, was)
	set := map[string]string{}
	for k, v := range now {
		if was[k] != v {
			set[k] = v
		}
	}
	var del []string
	for k := range was {
		if _, ok := now[k]; !ok {
			del = append(del, k)
		}
	}
	slices.Sort(del)
	if len(set) > 0 || len(del) > 0 {
		s.session.MapSet(scheduleMap, set, del)
	}
}

// takeSharedSchedule makes the shared map the window's schedule.
func (w *MainWindow) takeSharedSchedule() {
	s := w.shared
	if s == nil || w.markdown || !s.session.HasMap(scheduleMap) {
		return
	}
	f := schedFromEntries(s.session.MapOf(scheduleMap))
	cur := w.scheduleNow()
	if w.schedule.err != nil {
		return // never written over what could not be read
	}
	if maps.Equal(schedEntries(cur, nil), schedEntries(&f, nil)) {
		return
	}
	w.schedule.file, w.schedule.undo = f, nil // the board's Undo is of one's own change only
	w.storeSchedule()
}
