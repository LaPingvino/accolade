package collab

import (
	"encoding/json"
	"fmt"
	"maps"
	"sync"
)

// message is what peers send each other, sealed with the key: a hello
// with the sender's version (the peer answers with what it lacks), or
// ops.
type message struct {
	Hello   bool              `json:"hello,omitempty"`
	Proto   int               `json:"p,omitempty"` // with a hello: Protocol
	Version map[string]uint64 `json:"v,omitempty"` // an empty one is omitted: Hello says it is one
	Runs    []run             `json:"r,omitempty"`
	Meta    map[string]Value  `json:"m,omitempty"` // with a hello, or when one changed
	// By is the replica whose edit the runs are, passed on as it is when
	// relayed ("": a catch-up of several peers' edits)
	By string `json:"by,omitempty"`
}

// Protocol is the version of what peers send: peers of another one are
// told apart (OnMismatch), not half understood.
const Protocol = 3

// Value is one of the script's settings shared with it (its language,
// its format): the last one set wins, by clock, then by replica.
type Value struct {
	Value   string `json:"v"`
	Clock   uint64 `json:"c"`
	Replica string `json:"r"`
}

func (a Value) newer(b Value) bool {
	if a.Clock != b.Clock {
		return a.Clock > b.Clock
	}
	return a.Replica > b.Replica
}

// Session shares a Doc with peers over a Key. Transports (WebRTC, or
// anything carrying bytes) add peers with a send function and pass what
// arrives to Receive.
type Session struct {
	mu    sync.Mutex
	key   Key
	doc   *Doc
	peers map[string]func([]byte) error
	other map[string]bool // peers of another Protocol, not listened to
	// OnRemote is called (outside the lock) after ops from peers changed
	// the text: by is the replica whose edit it is ("": not known, a
	// catch-up of several peers' edits).
	OnRemote func(by string)
	meta     map[string]Value
	// OnMeta is called (outside the lock) after a peer changed a setting.
	OnMeta func(key, value string)
	// OnMismatch is called when a peer speaks another Protocol (0: one
	// from before there were versions).
	OnMismatch func(theirs int)
}

// NewSession shares doc with the peers that have key.
func NewSession(key Key, doc *Doc) *Session {
	return &Session{key: key, doc: doc, peers: map[string]func([]byte) error{}, other: map[string]bool{}, meta: map[string]Value{}}
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

// sendOps sends ops, the edit of replica by, as runs, in messages small
// enough for any data channel.
func (s *Session) sendOps(to func([]byte) error, ops []Op, by string) error {
	m := message{By: by}
	size := 0
	for _, r := range runsOf(ops) {
		if size += len(r.Text) + 64; size > batch && len(m.Runs) > 0 {
			if err := s.send(to, m); err != nil {
				return err
			}
			m, size = message{By: by}, len(r.Text)+64
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
	hello := message{Hello: true, Proto: Protocol, Version: s.doc.Version(), Meta: maps.Clone(s.meta)}
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
	other := s.other[from] || m.Hello && m.Proto != Protocol
	if other {
		s.other[from] = true
		delete(s.peers, from) // nothing more to it, nor from it
	}
	s.mu.Unlock()
	if other && !m.Hello {
		return nil
	}
	if other {
		if s.OnMismatch != nil {
			s.OnMismatch(m.Proto)
		}
		return fmt.Errorf("a peer speaks protocol %d, not %d", m.Proto, Protocol)
	}
	s.mu.Lock()
	var reply []Op
	if m.Hello {
		reply = s.doc.Missing(m.Version)
	}
	var changed map[string]Value
	for k, v := range m.Meta {
		if v.newer(s.meta[k]) {
			s.meta[k] = v
			if changed == nil {
				changed = map[string]Value{}
			}
			changed[k] = v
		}
	}
	var fresh []Op
	for _, r := range m.Runs {
		for _, op := range r.ops() {
			if s.doc.Apply(op) {
				fresh = append(fresh, op)
			}
		}
	}
	var newer map[string]Value // with a hello: the settings it lacks
	for k, v := range s.meta {
		if m.Hello && v.newer(m.Meta[k]) {
			if newer == nil {
				newer = map[string]Value{}
			}
			newer[k] = v
		}
	}
	to := s.peers[from]
	others := s.othersThan(from)
	s.mu.Unlock()

	if newer != nil && to != nil {
		_ = s.send(to, message{Meta: newer})
	}
	if changed != nil {
		for _, p := range others {
			_ = s.send(p, message{Meta: changed})
		}
		if s.OnMeta != nil {
			for k, v := range changed {
				s.OnMeta(k, v.Value)
			}
		}
	}
	if len(reply) > 0 && to != nil {
		if err := s.sendOps(to, reply, ""); err != nil {
			return err
		}
	}
	if len(fresh) > 0 {
		for _, p := range others {
			_ = s.sendOps(p, fresh, m.By) // a peer gone is removed by its transport
		}
		if s.OnRemote != nil {
			s.OnRemote(m.By)
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

// Meta is a shared setting's value ("" if none).
func (s *Session) Meta(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.meta[key].Value
}

// SetMeta sets a shared setting here and sends it to every peer.
func (s *Session) SetMeta(key, value string) {
	s.mu.Lock()
	if s.meta[key].Value == value && s.meta[key].Clock > 0 {
		s.mu.Unlock()
		return
	}
	var clock uint64
	for _, v := range s.meta {
		clock = max(clock, v.Clock)
	}
	v := Value{Value: value, Clock: clock + 1, Replica: s.doc.replica}
	s.meta[key] = v
	peers := s.othersThan("")
	s.mu.Unlock()
	for _, p := range peers {
		_ = s.send(p, message{Meta: map[string]Value{key: v}})
	}
}

// Local sends ops made here to every peer.
func (s *Session) Local(ops []Op) {
	if len(ops) == 0 {
		return
	}
	s.mu.Lock()
	peers := s.othersThan("")
	by := s.doc.replica
	s.mu.Unlock()
	for _, p := range peers {
		_ = s.sendOps(p, ops, by)
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
