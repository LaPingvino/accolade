package collab

import (
	"encoding/json"
	"errors"
	"maps"
	"slices"
)

// A session's state can be kept (a browser page that reloads, a window
// that joins the same script again) and taken up again: its Docs' ops and
// the shared settings. A peer taking it up has every op it had, so the
// hello's catch-up brings only what is new, and nothing is doubled.

// stateVersion is the kept state's format: a file of another is not read.
// Docs other than the script's ("docs") came with protocol 5; a reader of
// before ignores them and takes up the script, so the version stays.
const stateVersion = 1

type state struct {
	V    int              `json:"v"`
	Runs []Op             `json:"runs"`
	Docs map[string][]Op  `json:"docs,omitempty"`
	Meta map[string]Value `json:"meta,omitempty"`
}

// State is the session's Docs and settings, to keep.
func (s *Session) State() ([]byte, error) { return s.StateOf() }

// StateOf is State with only the Docs named (none named: all); the
// script's is always in it.
func (s *Session) StateOf(names ...string) ([]byte, error) {
	s.mu.Lock()
	st := state{V: stateVersion, Runs: runsOf(s.docs[Script].Ops()), Meta: maps.Clone(s.meta)}
	for name, d := range s.docs {
		if name != Script && (len(names) == 0 || slices.Contains(names, name)) {
			if st.Docs == nil {
				st.Docs = map[string][]Op{}
			}
			st.Docs[name] = runsOf(d.Ops())
		}
	}
	s.mu.Unlock()
	return json.Marshal(st)
}

// Restore is a session over key with the Docs and settings of a kept
// state, as replica (a new name: two pages that took up one state are two
// peers).
func Restore(key Key, replica string, kept []byte) (*Session, error) {
	var st state
	if err := json.Unmarshal(kept, &st); err != nil {
		return nil, err
	}
	if st.V != stateVersion {
		return nil, errors.New("a kept shared script of another version")
	}
	d := New(replica)
	for _, op := range st.Runs {
		d.Apply(op)
	}
	s := NewSession(key, d)
	for name, ops := range st.Docs {
		doc := s.doc(name)
		for _, op := range ops {
			doc.Apply(op)
		}
	}
	maps.Copy(s.meta, st.Meta)
	return s, nil
}
