package collab

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/pion/webrtc/v4"
)

// The network: peers of a script find each other through WebTorrent
// trackers (they pass WebRTC offers and answers between the peers that
// announce one info hash: the script's Room), then talk directly over
// WebRTC data channels. The offers and answers are sealed with the key,
// so a tracker sees the room's hash and opaque data, nothing else.

// Trackers are the public WebTorrent trackers tried.
var Trackers = []string{
	"wss://tracker.openwebtorrent.com",
	"wss://tracker.webtorrent.dev",
}

// ICEServers find a way through NAT (STUN only: they see addresses).
var ICEServers = []webrtc.ICEServer{{URLs: []string{"stun:stun.cloudflare.com:3478", "stun:stun.l.google.com:19302"}}}

// trackerMsg is a WebTorrent tracker message (the fields used here).
type trackerMsg struct {
	Action   string    `json:"action"`
	InfoHash string    `json:"info_hash"`
	PeerID   string    `json:"peer_id"`
	ToPeerID string    `json:"to_peer_id,omitempty"`
	NumWant  int       `json:"numwant,omitempty"`
	Offers   []offerIn `json:"offers,omitempty"`
	Offer    *sdpMsg   `json:"offer,omitempty"`
	Answer   *sdpMsg   `json:"answer,omitempty"`
	OfferID  string    `json:"offer_id,omitempty"`
	Interval int       `json:"interval,omitempty"`
}

type offerIn struct {
	Offer   sdpMsg `json:"offer"`
	OfferID string `json:"offer_id"`
}

type sdpMsg struct {
	Type string `json:"type"`
	SDP  string `json:"sdp"` // sealed with the key
}

// Network connects a Session to the script's other peers.
type Network struct {
	session *Session
	key     Key
	peerID  string // 20 bytes, as trackers want
	room    string

	mu      sync.Mutex
	offers  map[string]*webrtc.PeerConnection // our offers waiting for an answer
	peers   map[string]bool                   // peer IDs connected (or connecting)
	conns   []*webrtc.PeerConnection
	onPeers func(n int)
}

// binary20 is 20 random bytes as a tracker string (one char per byte).
func binary20() string {
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	r := make([]rune, 20)
	for i, c := range b {
		r[i] = rune(c)
	}
	return string(r)
}

// Join connects the session to its peers through the trackers until ctx
// ends; onPeers (may be nil) is told how many peers are connected.
func Join(ctx context.Context, s *Session, key Key, trackers []string, onPeers func(int)) *Network {
	room := key.Room()
	r := make([]rune, 20)
	for i, c := range room {
		r[i] = rune(c)
	}
	n := &Network{session: s, key: key, peerID: binary20(), room: string(r),
		offers: map[string]*webrtc.PeerConnection{}, peers: map[string]bool{}, onPeers: onPeers}
	for _, t := range trackers {
		go n.tracker(ctx, t)
	}
	go func() {
		<-ctx.Done()
		n.mu.Lock()
		defer n.mu.Unlock()
		for _, pc := range n.conns {
			_ = pc.Close()
		}
	}()
	return n
}

// Peers is how many peers are connected.
func (n *Network) Peers() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.peers)
}

// tracker keeps announcing to one tracker, reconnecting after failures.
func (n *Network) tracker(ctx context.Context, url string) {
	for ctx.Err() == nil {
		_ = n.announceLoop(ctx, url)
		select {
		case <-ctx.Done():
		case <-time.After(15 * time.Second):
		}
	}
}

func (n *Network) announceLoop(ctx context.Context, url string) error {
	c, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		return err
	}
	defer c.CloseNow()
	c.SetReadLimit(1 << 20)
	var wmu sync.Mutex
	send := func(m trackerMsg) error {
		b, _ := json.Marshal(m)
		wmu.Lock()
		defer wmu.Unlock()
		return c.Write(ctx, websocket.MessageText, b)
	}
	kick := make(chan struct{}, 1)
	go func() { // announce now and then, with fresh offers
		for ctx.Err() == nil {
			offers := n.makeOffers(3)
			if send(trackerMsg{Action: "announce", InfoHash: n.room, PeerID: n.peerID, NumWant: len(offers), Offers: offers}) != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-kick:
			case <-time.After(60 * time.Second):
			}
		}
	}()
	for {
		_, b, err := c.Read(ctx)
		if err != nil {
			return err
		}
		var m trackerMsg
		if json.Unmarshal(b, &m) != nil || m.InfoHash != n.room || m.PeerID == n.peerID {
			continue
		}
		switch {
		case m.Offer != nil && m.PeerID > n.peerID:
			// the other peer answers: offer it ours now (trackers pass
			// offers to the peers there at the time only)
			if !n.connected(m.PeerID) {
				select {
				case kick <- struct{}{}:
				default:
				}
			}
		case m.Offer != nil:
			go n.answer(m, send)
		case m.Answer != nil:
			n.accept(m)
		}
	}
}

// newConn is a peer connection whose data channel joins the session.
func (n *Network) newConn() (*webrtc.PeerConnection, error) {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{ICEServers: ICEServers})
	if err != nil {
		return nil, err
	}
	n.mu.Lock()
	n.conns = append(n.conns, pc)
	n.mu.Unlock()
	return pc, nil
}

// attach joins an open data channel to the session as peer id.
func (n *Network) attach(id string, dc *webrtc.DataChannel) {
	dc.OnOpen(func() {
		n.mu.Lock()
		count := len(n.peers)
		n.mu.Unlock()
		if n.onPeers != nil {
			n.onPeers(count)
		}
		_ = n.session.AddPeer(id, func(b []byte) error { return dc.Send(b) })
	})
	dc.OnMessage(func(m webrtc.DataChannelMessage) { _ = n.session.Receive(id, m.Data) })
	dc.OnClose(func() {
		n.session.RemovePeer(id)
		n.mu.Lock()
		delete(n.peers, id)
		count := len(n.peers)
		n.mu.Unlock()
		if n.onPeers != nil {
			n.onPeers(count)
		}
	})
}

// sealSDP and openSDP keep the connection details from the tracker.
func (n *Network) sealSDP(sdp string) (string, error) {
	b, err := n.key.Seal([]byte(sdp))
	return base64.StdEncoding.EncodeToString(b), err
}

func (n *Network) openSDP(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	sdp, err := n.key.Open(b)
	return string(sdp), err
}

// makeOffers are offers for peers that may answer through the tracker.
func (n *Network) makeOffers(count int) []offerIn {
	var out []offerIn
	for i := 0; i < count; i++ {
		pc, err := n.newConn()
		if err != nil {
			continue
		}
		dc, err := pc.CreateDataChannel("accolade", nil)
		if err != nil {
			_ = pc.Close()
			continue
		}
		id := binary20()
		n.mu.Lock()
		n.offers[id] = pc
		n.mu.Unlock()
		pc.OnDataChannel(func(*webrtc.DataChannel) {})
		pc.OnConnectionStateChange(func(st webrtc.PeerConnectionState) {
			if st == webrtc.PeerConnectionStateFailed {
				_ = pc.Close()
			}
		})
		n.pendingChannel(id, dc)
		sdp, err := localSDP(pc, true)
		if err != nil {
			_ = pc.Close()
			continue
		}
		sealed, err := n.sealSDP(sdp)
		if err != nil {
			continue
		}
		out = append(out, offerIn{Offer: sdpMsg{Type: "offer", SDP: sealed}, OfferID: id})
	}
	return out
}

// pendingChannel remembers an offer's data channel until its answer
// says which peer it is.
var channels sync.Map // offer id -> *webrtc.DataChannel

func (n *Network) pendingChannel(offerID string, dc *webrtc.DataChannel) { channels.Store(offerID, dc) }

// localSDP makes an offer or answer and waits for its ICE candidates (no
// trickling through a tracker).
func localSDP(pc *webrtc.PeerConnection, offer bool) (string, error) {
	var d webrtc.SessionDescription
	var err error
	if offer {
		d, err = pc.CreateOffer(nil)
	} else {
		d, err = pc.CreateAnswer(nil)
	}
	if err != nil {
		return "", err
	}
	done := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(d); err != nil {
		return "", err
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
	}
	if pc.LocalDescription() == nil {
		return "", errors.New("no local description")
	}
	return pc.LocalDescription().SDP, nil
}

// connected reports whether a peer is connected (or connecting).
func (n *Network) connected(peer string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.peers[peer]
}

// claim reserves a peer, false if it is connected (or connecting) already.
func (n *Network) claim(peer string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.peers[peer] {
		return false
	}
	n.peers[peer] = true
	return true
}

// answer answers another peer's offer (one connection per pair: the
// peer with the higher ID answers; both answering, each refused the
// other's answer as a second connection).
func (n *Network) answer(m trackerMsg, send func(trackerMsg) error) {
	sdp, err := n.openSDP(m.Offer.SDP)
	if err != nil || !n.claim(m.PeerID) { // not this script's, or known
		return
	}
	pc, err := n.newConn()
	if err != nil {
		n.release(m.PeerID)
		return
	}
	peer := m.PeerID
	pc.OnDataChannel(func(dc *webrtc.DataChannel) { n.attach(peer, dc) })
	if pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}) != nil {
		n.release(peer)
		return
	}
	answer, err := localSDP(pc, false)
	if err != nil {
		n.release(peer)
		return
	}
	sealed, err := n.sealSDP(answer)
	if err != nil {
		return
	}
	_ = send(trackerMsg{Action: "announce", InfoHash: n.room, PeerID: n.peerID, ToPeerID: peer,
		Answer: &sdpMsg{Type: "answer", SDP: sealed}, OfferID: m.OfferID})
}

// accept takes the answer to one of our offers.
func (n *Network) accept(m trackerMsg) {
	n.mu.Lock()
	pc := n.offers[m.OfferID]
	delete(n.offers, m.OfferID)
	n.mu.Unlock()
	if pc == nil {
		return
	}
	sdp, err := n.openSDP(m.Answer.SDP)
	if err != nil || !n.claim(m.PeerID) {
		_ = pc.Close()
		return
	}
	if dc, ok := channels.LoadAndDelete(m.OfferID); ok {
		n.attach(m.PeerID, dc.(*webrtc.DataChannel))
	}
	if pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: sdp}) != nil {
		n.release(m.PeerID)
		_ = pc.Close()
	}
}

func (n *Network) release(peer string) {
	n.mu.Lock()
	delete(n.peers, peer)
	n.mu.Unlock()
}
