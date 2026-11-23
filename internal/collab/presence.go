package collab

import (
	"slices"
	"strings"
	"time"
)

// Presence is who is in the session now, apart from the script: each
// peer says it is here (its name, the line its cursor is on) when it
// comes, when its cursor goes to another line, and every HereEvery; it is
// counted here until HereFor has gone by without a word from it, until it
// says it leaves (Gone), or until the peer it came through goes away.
// None of it is kept (State): it is about now. A peer that does not know
// it ignores the message's "h" (and a presence's "a" and "f"): it added
// no protocol.
//
// At and From are where the cursor is and where its selection starts
// (nil: no selection), in the script's Doc, as the rune each comes after
// (the zero ID: the start; Doc.Anchor): a co-writer's edit before them
// leaves them where they were, so a window draws them right however its
// text has moved since they were said (Doc.Place). Line stays for a peer
// whose ops have not come yet, and for the peers before At.
type Presence struct {
	Replica string `json:"r"`
	Name    string `json:"n,omitempty"`
	Line    int    `json:"l"`
	At      *ID    `json:"a,omitempty"`
	From    *ID    `json:"f,omitempty"`
	Seq     uint64 `json:"s"`
	Gone    bool   `json:"g,omitempty"`
	// View: a viewer that follows only (a Table Read on a TV): it edits
	// nothing and is not to be followed. Older peers ignore it.
	View bool `json:"v,omitempty"`
}

// HereEvery is how often a window says it is here, HereFor how long a
// peer is counted here after the last word from it.
const (
	HereEvery = 30 * time.Second
	HereFor   = 3 * HereEvery
)

// heard is a peer's presence as last heard: through which connected peer
// (its own connection, or one passing it on), and when.
type heard struct {
	p   Presence
	via string
	at  time.Time
}

// SetHere says this side is here, as name, its cursor on line; again with
// nothing changed, it says it is still here.
func (s *Session) SetHere(name string, line int) { s.SetHereAt(name, line, -1, -1) }

// SetHereAt says this side is here, as name, its cursor on line at rune
// offset cursor in the script, its selection from rune offset from
// (from == cursor: none; cursor < 0: no place but the line).
func (s *Session) SetHereAt(name string, line, cursor, from int) {
	s.mu.Lock()
	p := Presence{Replica: s.replica, Name: name, Line: line, Seq: s.me.Seq + 1, View: s.Viewer}
	if d := s.docs[Script]; d != nil && cursor >= 0 {
		at := d.Anchor(cursor)
		p.At = &at
		if from >= 0 && from != cursor {
			f := d.Anchor(from)
			p.From = &f
		}
	}
	s.me = p
	me := s.me
	peers := s.othersThan("")
	s.mu.Unlock()
	for _, p := range peers {
		_ = s.send(p, message{Here: &me})
	}
}

// Leave says this side leaves (the others take it off their list at once).
func (s *Session) Leave() {
	s.mu.Lock()
	if s.me.Seq == 0 {
		s.mu.Unlock()
		return
	}
	s.me = Presence{Replica: s.replica, Name: s.me.Name, Seq: s.me.Seq + 1, Gone: true, View: s.Viewer}
	me := s.me
	peers := s.othersThan("")
	s.mu.Unlock()
	for _, p := range peers {
		_ = s.send(p, message{Here: &me})
	}
}

// Here is who else is here now, by name.
func (s *Session) Here() []Presence {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.clock()
	var out []Presence
	for r, h := range s.here {
		if r != s.replica && !h.p.Gone && now.Sub(h.at) < HereFor {
			out = append(out, h.p)
		}
	}
	slices.SortFunc(out, func(a, b Presence) int {
		if c := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); c != 0 {
			return c
		}
		return strings.Compare(a.Replica, b.Replica)
	})
	return out
}

// clock is the time now (a test's own, if set; locked).
func (s *Session) clock() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// heardHere takes p, from peer from, if it is newer than what was heard
// of its replica (locked): true if it was, to be passed on.
func (s *Session) heardHere(from string, p Presence) bool {
	if p.Replica == "" || p.Replica == s.replica {
		return false
	}
	if h, ok := s.here[p.Replica]; ok && h.p.Seq >= p.Seq {
		return false
	}
	s.here[p.Replica] = heard{p: p, via: from, at: s.clock()}
	return true
}

// heres is what to tell a peer just connected of who is here (locked):
// this side, and the others heard from lately.
func (s *Session) heres() []Presence {
	var out []Presence
	if s.me.Seq > 0 {
		out = append(out, s.me)
	}
	now := s.clock()
	for _, h := range s.here {
		if !h.p.Gone && now.Sub(h.at) < HereFor {
			out = append(out, h.p)
		}
	}
	return out
}

// dropVia forgets the presences heard through peer id, gone (locked):
// true if there were any.
func (s *Session) dropVia(id string) bool {
	dropped := false
	for r, h := range s.here {
		if h.via == id {
			delete(s.here, r)
			dropped = true
		}
	}
	return dropped
}

// hereChanged calls OnHere, in Run, on a goroutine of its own: a peer
// goes away (RemovePeer) on its transport's, which may be the one Run
// waits for (OnHere looks at who is here as it is then, so the order of
// two calls does not matter).
func (s *Session) hereChanged() {
	if s.OnHere != nil {
		go s.run(s.OnHere)
	}
}
