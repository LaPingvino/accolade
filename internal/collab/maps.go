// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package collab

import (
	"cmp"
	"maps"
	"slices"
	"strings"
)

// Shared maps (docs/COLLAB.md, question 3): what lives beside the text
// and is not text — the breakdown's tags, the schedule's strips — kept as
// maps of keys to values, each key's last writer winning (by clock, then
// replica) and a key taken away kept as a tombstone, so two people
// changing different keys keep both, and a key changed by two at once
// ends the same on every peer. A map has messages of its own (batched
// like the text's runs), not the settings' meta: a hello carries only
// each map's version, never the map.

// Entry is a key of a shared map as last set: its value, or taken away
// (Gone), by Replica at Clock.
type Entry struct {
	Key     string `json:"k"`
	Value   string `json:"v,omitempty"`
	Gone    bool   `json:"g,omitempty"`
	Clock   uint64 `json:"c"`
	Replica string `json:"r"`
}

func (e Entry) newer(o Entry) bool {
	if e.Clock != o.Clock {
		return e.Clock > o.Clock
	}
	return e.Replica > o.Replica
}

// sharedMap is a map's entries, and the highest clock seen of each
// replica (its version, for a hello).
type sharedMap struct {
	entries map[string]Entry
	seen    map[string]uint64
	clock   uint64
}

func newSharedMap() *sharedMap {
	return &sharedMap{entries: map[string]Entry{}, seen: map[string]uint64{}}
}

// apply takes e if it is newer than what the map has of its key: true if
// it was (to pass on).
func (m *sharedMap) apply(e Entry) bool {
	m.clock = max(m.clock, e.Clock)
	m.seen[e.Replica] = max(m.seen[e.Replica], e.Clock)
	if cur, ok := m.entries[e.Key]; ok && !e.newer(cur) {
		return false
	}
	m.entries[e.Key] = e
	return true
}

// missing are the entries a peer that has seen version lacks, oldest
// first (a catch-up cut short leaves no gap in a replica's).
func (m *sharedMap) missing(version map[string]uint64) []Entry {
	var out []Entry
	for _, e := range m.entries {
		if e.Clock > version[e.Replica] {
			out = append(out, e)
		}
	}
	slices.SortFunc(out, func(a, b Entry) int {
		if a.Clock != b.Clock {
			return cmp.Compare(a.Clock, b.Clock)
		}
		return strings.Compare(a.Replica, b.Replica)
	})
	return out
}

// values are the keys not taken away, and their values.
func (m *sharedMap) values() map[string]string {
	out := make(map[string]string, len(m.entries))
	for k, e := range m.entries {
		if !e.Gone {
			out[k] = e.Value
		}
	}
	return out
}

// sharedMap is the map named name, made if there is none (locked).
func (s *Session) sharedMap(name string) *sharedMap {
	m := s.maps[name]
	if m == nil {
		m = newSharedMap()
		s.maps[name] = m
	}
	return m
}

// MapSet sets keys of the shared map name and takes others away (del),
// here and at every peer: a key already so is left as it is.
func (s *Session) MapSet(name string, set map[string]string, del []string) {
	s.mu.Lock()
	m := s.sharedMap(name)
	var out []Entry
	put := func(e Entry) {
		cur, ok := m.entries[e.Key]
		if ok && cur.Gone == e.Gone && cur.Value == e.Value || !ok && e.Gone {
			return
		}
		m.clock++
		e.Clock, e.Replica = m.clock, s.replica
		m.apply(e)
		out = append(out, e)
	}
	for _, k := range slices.Sorted(maps.Keys(set)) {
		put(Entry{Key: k, Value: set[k]})
	}
	for _, k := range del {
		put(Entry{Key: k, Gone: true})
	}
	peers := s.othersThan("")
	s.mu.Unlock()
	for _, p := range peers {
		_ = s.sendEntries(p, name, out)
	}
}

// MapOf is the shared map name as it is: its keys and values.
func (s *Session) MapOf(name string) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m := s.maps[name]; m != nil {
		return m.values()
	}
	return map[string]string{}
}

// HasMap reports whether the session has heard of the map name (set
// here, come from a peer, or kept).
func (s *Session) HasMap(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.maps[name] != nil
}

// sendEntries sends a map's entries in messages small enough for any
// data channel.
func (s *Session) sendEntries(to func([]byte) error, name string, es []Entry) error {
	m := message{Map: name}
	size := 0
	for _, e := range es {
		if size += len(e.Key) + len(e.Value) + 64; size > batch && len(m.Entries) > 0 {
			if err := s.send(to, m); err != nil {
				return err
			}
			m, size = message{Map: name}, len(e.Key)+len(e.Value)+64
		}
		m.Entries = append(m.Entries, e)
	}
	if len(m.Entries) == 0 {
		return nil
	}
	return s.send(to, m)
}
