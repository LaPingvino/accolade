package collab

import (
	"encoding/json"
	"errors"
	"maps"
)

// A session's state can be kept (a browser page that reloads, a window
// that joins the same script again) and taken up again: the document's
// ops and the shared settings. A peer taking it up has every op it had,
// so the hello's catch-up brings only what is new, and nothing is doubled.

// stateVersion is the kept state's format: a file of another is not read.
const stateVersion = 1

type state struct {
	V    int              `json:"v"`
	Runs []Op             `json:"runs"`
	Meta map[string]Value `json:"meta,omitempty"`
}

// State is the session's document and settings, to keep.
func (s *Session) State() ([]byte, error) {
	s.mu.Lock()
	st := state{V: stateVersion, Runs: runsOf(s.doc.Ops()), Meta: maps.Clone(s.meta)}
	s.mu.Unlock()
	return json.Marshal(st)
}

// Restore is a session over key with the document and settings of a kept
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
	maps.Copy(s.meta, st.Meta)
	return s, nil
}
