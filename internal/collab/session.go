package collab

import (
	"encoding/json"
	"sync"
)

// message is what peers send each other, sealed with the key: a hello
// with the sender's version (the peer answers with what it lacks), or
// ops.
type message struct {
	Hello   bool              `json:"hello,omitempty"`
	Version map[string]uint64 `json:"v,omitempty"` // an empty one is omitted: Hello says it is one
	Runs    []run             `json:"r,omitempty"`
}

// Session shares a Doc with peers over a Key. Transports (WebRTC, or
// anything carrying bytes) add peers with a send function and pass what
// arrives to Receive.
type Session struct {
	mu    sync.Mutex
	key   Key
	doc   *Doc
	peers map[string]func([]byte) error
	// OnRemote is called (outside the lock) after ops from peers changed
	// the text.
	OnRemote func()
}

// NewSession shares doc with the peers that have key.
func NewSession(key Key, doc *Doc) *Session {
	return &Session{key: key, doc: doc, peers: map[string]func([]byte) error{}}
}

// Doc is the shared document; lock with Do while using it.
func (s *Session) Do(f func(d *Doc)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(s.doc)
}

// batch is about the most bytes of runs in one message: browsers drop
// data channel messages much over 16 KiB.
const batch = 8 << 10

// sendOps sends ops as runs, in messages small enough for any data channel.
func (s *Session) sendOps(to func([]byte) error, ops []Op) error {
	var m message
	size := 0
	for _, r := range runsOf(ops) {
		if size += len(r.Text) + 64; size > batch && len(m.Runs) > 0 {
			if err := s.send(to, m); err != nil {
				return err
			}
			m, size = message{}, len(r.Text)+64
		}
		m.Runs = append(m.Runs, r)
	}
	if len(m.Runs) == 0 {
		return nil
	}
	return s.send(to, m)
}

func (s *Session) send(to func([]byte) error, m message) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	sealed, err := s.key.Seal(b)
	if err != nil {
		return err
	}
	return to(sealed)
}

// AddPeer adds a connected peer and greets it.
func (s *Session) AddPeer(id string, send func([]byte) error) error {
	s.mu.Lock()
	s.peers[id] = send
	hello := message{Hello: true, Version: s.doc.Version()}
	s.mu.Unlock()
	return s.send(send, hello)
}

// RemovePeer forgets a peer that went away.
func (s *Session) RemovePeer(id string) {
	s.mu.Lock()
	delete(s.peers, id)
	s.mu.Unlock()
}

// Receive handles a message from peer from: a hello gets the ops it
// lacks; new ops are applied and passed on to the other peers.
func (s *Session) Receive(from string, sealed []byte) error {
	b, err := s.key.Open(sealed)
	if err != nil {
		return err
	}
	var m message
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	s.mu.Lock()
	var reply []Op
	if m.Hello {
		reply = s.doc.Missing(m.Version)
	}
	var fresh []Op
	for _, r := range m.Runs {
		for _, op := range r.ops() {
			if s.doc.Apply(op) {
				fresh = append(fresh, op)
			}
		}
	}
	to := s.peers[from]
	others := s.othersThan(from)
	s.mu.Unlock()

	if len(reply) > 0 && to != nil {
		if err := s.sendOps(to, reply); err != nil {
			return err
		}
	}
	if len(fresh) > 0 {
		for _, p := range others {
			_ = s.sendOps(p, fresh) // a peer gone is removed by its transport
		}
		if s.OnRemote != nil {
			s.OnRemote()
		}
	}
	return nil
}

func (s *Session) othersThan(id string) []func([]byte) error {
	var out []func([]byte) error
	for p, send := range s.peers {
		if p != id {
			out = append(out, send)
		}
	}
	return out
}

// Local sends ops made here to every peer.
func (s *Session) Local(ops []Op) {
	if len(ops) == 0 {
		return
	}
	s.mu.Lock()
	peers := s.othersThan("")
	s.mu.Unlock()
	for _, p := range peers {
		_ = s.sendOps(p, ops)
	}
}

// SetText makes the document's text text, as the edit between the two
// (the common start and end kept), and returns the ops: an editor's
// change becomes ops this way.
func (d *Doc) SetText(text string) []Op {
	old, nw := []rune(d.Text()), []rune(text)
	start := 0
	for start < len(old) && start < len(nw) && old[start] == nw[start] {
		start++
	}
	end := 0
	for end < len(old)-start && end < len(nw)-start && old[len(old)-1-end] == nw[len(nw)-1-end] {
		end++
	}
	ops := d.Delete(start, len(old)-start-end)
	return append(ops, d.Insert(start, string(nw[start:len(nw)-end]))...)
}
