package collab

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"
)

// message is what peers send each other, sealed with the key: a hello
// with the sender's versions (the peer answers with what it lacks), or
// ops of one of the session's Docs.
//
// A peer of protocol 4 unmarshals a hello into its own message before it
// looks at p, so the fields it knows keep their JSON types: "hello", "p",
// "app", "v", "n", "r", "m", "by". What protocol 5 adds goes under keys of
// its own ("docs", "d"), which it ignores, and it then stops at p=5.
type message struct {
	Hello bool   `json:"hello,omitempty"`
	Proto int    `json:"p,omitempty"`   // with a hello: Protocol
	App   string `json:"app,omitempty"` // with a hello: the sender's Accolade version
	// Docs, with a hello: each Doc's version and counts (Doc.Missing),
	// by name; a Doc the sender has with nothing in it is there too
	Docs map[string]docVersion `json:"docs,omitempty"`
	// Doc is the Doc the runs are of ("": the script)
	Doc  string           `json:"d,omitempty"`
	Runs []Op             `json:"r,omitempty"`
	Meta map[string]Value `json:"m,omitempty"` // with a hello, or when one changed
	// By is the replica whose edit the runs are, passed on as it is when
	// relayed ("": a catch-up of several peers' edits)
	By string `json:"by,omitempty"`
	// Here is a peer saying it is here, or leaves (presence.go)
	Here *Presence `json:"h,omitempty"`
}

// docVersion is a Doc's state in a hello.
type docVersion struct {
	V map[string]uint64 `json:"v,omitempty"`
	N map[string]int    `json:"n,omitempty"`
}

// Protocol is the version of what peers send: peers of another one are
// told apart (OnMismatch), not half understood. Any change to what goes
// over the wire bumps it (a field old peers may ignore does not); see
// docs/COLLAB.md. 4: hellos carry counts (Doc.Missing). 5: named Docs (a
// message names its Doc, a hello has each Doc's version).
const Protocol = 5

// Script is the name of the session's first Doc, the script's.
const Script = "script"

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

// Session shares Docs with peers over a Key: the script's, and others
// by name (the script's source). Transports (WebRTC, or anything carrying
// bytes) add peers with a send function and pass what arrives to Receive.
type Session struct {
	mu      sync.Mutex
	key     Key
	replica string
	docs    map[string]*Doc // by name; Script is always there
	peers   map[string]func([]byte) error
	other   map[string]bool // peers of another Protocol, not listened to
	dropped map[string]bool // Docs let go (Drop): their ops are not taken again
	proto   int             // Protocol (another in the tests of mixed peers)
	// OnRemote is called (outside the lock, in Run) after ops from peers
	// changed Doc doc's text: by is the replica whose edit it is ("": not
	// known, a catch-up of several peers' edits).
	OnRemote func(doc, by string)
	meta     map[string]Value
	// OnMeta is called (outside the lock, in Run) after a peer changed a setting.
	OnMeta func(key, value string)
	// OnMismatch is called when a peer speaks another Protocol (0: one
	// from before there were versions), with its Accolade version if it
	// said ("" before protocol 4).
	OnMismatch func(theirs int, app string)
	// App is this Accolade's version, told to peers in the hello.
	App string
	// Run runs what a peer's message does to the documents and the
	// callbacks after it, in order (nil: at once, on the receiving
	// goroutine). A window passes fyne.Do: a document then changes only
	// on the UI goroutine, in the same step as its editor, so a key typed
	// meanwhile is never compared with a document that is ahead of it.
	Run func(func())
	// OnHere is called (outside the lock, in Run) when who is here
	// changed (Here), or what line they are on.
	OnHere func()
	// Now, if set, is the clock presence goes by (the tests').
	Now  func() time.Time
	me   Presence         // this side's, as last said (SetHere)
	here map[string]heard // the others', by replica
}

// NewSession shares doc, the script, with the peers that have key.
func NewSession(key Key, doc *Doc) *Session {
	return &Session{key: key, replica: doc.replica, docs: map[string]*Doc{Script: doc}, peers: map[string]func([]byte) error{},
		other: map[string]bool{}, dropped: map[string]bool{}, meta: map[string]Value{}, proto: Protocol, here: map[string]heard{}}
}

// Do is f with the script's Doc, the session locked.
func (s *Session) Do(f func(d *Doc)) { s.DoIn(Script, f) }

// DoIn is f with the Doc named name (made, empty, if there is none), the
// session locked.
func (s *Session) DoIn(name string, f func(d *Doc)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(s.doc(name))
}

// Has reports whether the session has a Doc named name.
func (s *Session) Has(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.docs[name] != nil
}

// Names are the session's Docs' names.
func (s *Session) Names() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Sorted(maps.Keys(s.docs))
}

// Drop lets the Doc named name go (not the script's): a source no longer
// shared. Its ops coming from a peer that has it still are not taken.
func (s *Session) Drop(name string) {
	if name == Script || name == "" {
		return
	}
	s.mu.Lock()
	delete(s.docs, name)
	s.dropped[name] = true
	s.mu.Unlock()
}

// doc is the Doc named name, made if there is none (locked).
func (s *Session) doc(name string) *Doc {
	if name == "" {
		name = Script
	}
	d := s.docs[name]
	if d == nil {
		d = New(s.replica)
		s.docs[name] = d
	}
	return d
}

// batch is about the most bytes of runs in one message: browsers drop
// data channel messages much over 16 KiB.
const batch = 8 << 10

// sendOps sends ops of Doc doc, the edit of replica by, as runs, in
// messages small enough for any data channel.
func (s *Session) sendOps(to func([]byte) error, doc string, ops []Op, by string) error {
	if doc == Script {
		doc = ""
	}
	m := message{Doc: doc, By: by}
	size := 0
	for _, r := range runsOf(ops) {
		if size += len(r.Text) + 64; size > batch && len(m.Runs) > 0 {
			if err := s.send(to, m); err != nil {
				return err
			}
			m, size = message{Doc: doc, By: by}, len(r.Text)+64
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

// hello is this side's hello (locked).
func (s *Session) hello() message {
	m := message{Hello: true, Proto: s.proto, App: s.App, Meta: maps.Clone(s.meta), Docs: map[string]docVersion{}}
	for name, d := range s.docs {
		m.Docs[name] = docVersion{V: d.Version(), N: d.Counts()}
	}
	return m
}

// AddPeer adds a connected peer and greets it.
func (s *Session) AddPeer(id string, send func([]byte) error) error {
	s.mu.Lock()
	s.peers[id] = send
	hello := s.hello()
	heres := s.heres()
	s.mu.Unlock()
	if err := s.send(send, hello); err != nil {
		return err
	}
	for _, p := range heres { // who is here, for it (presence.go)
		_ = s.send(send, message{Here: &p})
	}
	return nil
}

// RemovePeer forgets a peer that went away.
func (s *Session) RemovePeer(id string) {
	s.mu.Lock()
	delete(s.peers, id)
	dropped := s.dropVia(id)
	s.mu.Unlock()
	if dropped {
		s.hereChanged()
	}
}

// Receive handles a message from peer from: a hello gets the ops it
// lacks; new ops are applied and passed on to the other peers. What it
// does to the documents runs in Run.
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
	other := s.other[from] || m.Hello && m.Proto != s.proto
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
			s.run(func() { s.OnMismatch(m.Proto, m.App) })
		}
		return fmt.Errorf("a peer speaks protocol %d, not %d", m.Proto, s.proto)
	}
	s.run(func() { s.receive(from, m) })
	return nil
}

func (s *Session) run(f func()) {
	if s.Run != nil {
		s.Run(f)
		return
	}
	f()
}

// receive does what message m from peer from says (in Run).
func (s *Session) receive(from string, m message) {
	s.mu.Lock()
	type reply struct {
		doc string
		ops []Op
	}
	var replies []reply
	if m.Hello { // each Doc's ops the peer lacks; a Doc it has not, whole
		names := slices.Sorted(maps.Keys(s.docs))
		for _, name := range names {
			v, ok := m.Docs[name]
			var ops []Op
			if ok {
				ops = s.docs[name].Missing(v.V, v.N)
			} else {
				ops = s.docs[name].Missing(nil, nil)
			}
			if len(ops) > 0 {
				replies = append(replies, reply{name, ops})
			}
		}
		for name := range m.Docs { // the peer's Docs, known here too (its ops come with its reply)
			if !s.dropped[name] {
				s.doc(name)
			}
		}
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
	name := m.Doc
	if name == "" {
		name = Script
	}
	var fresh []Op
	if len(m.Runs) > 0 && !s.dropped[name] {
		d := s.doc(name)
		for _, r := range m.Runs {
			fresh = append(fresh, d.Apply(r)...)
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
	var here *Presence // a peer's presence, new: passed on
	if m.Here != nil && s.heardHere(from, *m.Here) {
		here = m.Here
	}
	to := s.peers[from]
	others := s.othersThan(from)
	s.mu.Unlock()

	if here != nil {
		for _, p := range others {
			_ = s.send(p, message{Here: here})
		}
		if s.OnHere != nil {
			s.OnHere()
		}
	}

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
	if to != nil {
		for _, r := range replies {
			_ = s.sendOps(to, r.doc, r.ops, "")
		}
	}
	if len(fresh) > 0 {
		for _, p := range others {
			_ = s.sendOps(p, name, fresh, m.By) // a peer gone is removed by its transport
		}
		if s.OnRemote != nil {
			s.OnRemote(name, m.By)
		}
	}
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
	v := Value{Value: value, Clock: clock + 1, Replica: s.replica}
	s.meta[key] = v
	peers := s.othersThan("")
	s.mu.Unlock()
	for _, p := range peers {
		_ = s.send(p, message{Meta: map[string]Value{key: v}})
	}
}

// Local sends ops made here in the script's Doc to every peer.
func (s *Session) Local(ops []Op) { s.LocalIn(Script, ops) }

// LocalIn sends ops made here in Doc doc to every peer.
func (s *Session) LocalIn(doc string, ops []Op) {
	if len(ops) == 0 {
		return
	}
	s.mu.Lock()
	peers := s.othersThan("")
	s.mu.Unlock()
	for _, p := range peers {
		_ = s.sendOps(p, doc, ops, s.replica)
	}
}
