//go:build !js

package collab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/pion/logging"
	"github.com/pion/transport/v5/vnet"
	"github.com/pion/webrtc/v4"
)

// fakeTracker relays offers and answers between the peers of an info
// hash, as WebTorrent trackers do.
func fakeTracker(t *testing.T) string {
	var mu sync.Mutex
	peers := map[string]*websocket.Conn{} // info hash + peer id -> conn
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		ctx := context.Background()
		for {
			_, b, err := c.Read(ctx)
			if err != nil {
				return
			}
			var m trackerMsg
			if json.Unmarshal(b, &m) != nil {
				continue
			}
			mu.Lock()
			peers[m.InfoHash+m.PeerID] = c
			var out []func()
			relay := func(to *websocket.Conn, msg trackerMsg) {
				b, _ := json.Marshal(msg)
				out = append(out, func() { _ = to.Write(ctx, websocket.MessageText, b) })
			}
			if m.Answer != nil {
				if to := peers[m.InfoHash+m.ToPeerID]; to != nil {
					relay(to, trackerMsg{Action: "announce", InfoHash: m.InfoHash, PeerID: m.PeerID, Answer: m.Answer, OfferID: m.OfferID})
				}
			}
			i := 0
			for key, to := range peers {
				if i >= len(m.Offers) || !strings.HasPrefix(key, m.InfoHash) || to == c {
					continue
				}
				o := m.Offers[i]
				i++
				relay(to, trackerMsg{Action: "announce", InfoHash: m.InfoHash, PeerID: m.PeerID, Offer: &o.Offer, OfferID: o.OfferID})
			}
			mu.Unlock()
			for _, f := range out {
				f()
			}
		}
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

// Two peers find each other through a tracker, connect over WebRTC and
// edit together; a third joins later and catches up.
func TestNetwork(t *testing.T) {
	// a virtual network (pion's vnet): the same on every system
	was, wasNew := ICEServers, newPeerConnection
	ICEServers = nil
	t.Cleanup(func() { ICEServers, newPeerConnection = was, wasNew })
	router, err := vnet.NewRouter(&vnet.RouterConfig{CIDR: "10.0.0.0/24", LoggerFactory: logging.NewDefaultLoggerFactory()})
	if err != nil {
		t.Fatal(err)
	}
	on := func(ip string) func(webrtc.Configuration) (*webrtc.PeerConnection, error) {
		n, err := vnet.NewNet(&vnet.NetConfig{StaticIPs: []string{ip}})
		if err != nil {
			t.Fatal(err)
		}
		if err := router.AddNet(n); err != nil {
			t.Fatal(err)
		}
		var se webrtc.SettingEngine
		se.SetNet(n)
		return webrtc.NewAPI(webrtc.WithSettingEngine(se)).NewPeerConnection
	}
	peerA, peerB, peerC := on("10.0.0.1"), on("10.0.0.2"), on("10.0.0.3")
	if err := router.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = router.Stop() })
	tracker := fakeTracker(t)
	key, _ := NewKey()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a, b := NewSession(key, New("a")), NewSession(key, New("b"))
	a.Do(func(d *Doc) { d.SetText("INT. BARN - DAY\n") })
	newPeerConnection = peerA
	na := Join(ctx, a, key, []string{tracker}, nil)
	time.Sleep(time.Second) // a's offers wait at the tracker for b
	newPeerConnection = peerB
	nb := Join(ctx, b, key, []string{tracker}, nil)

	text := func(s *Session) (t string) { s.Do(func(d *Doc) { t = d.Text() }); return }
	waitFor := func(what string, ok func() bool) {
		for deadline := time.Now().Add(30 * time.Second); !ok(); time.Sleep(100 * time.Millisecond) {
			if time.Now().After(deadline) {
				t.Fatalf("%s: a %q (%d peers), b %q (%d peers)", what, text(a), na.Peers(), text(b), nb.Peers())
			}
		}
	}
	waitFor("b gets a's text", func() bool { return text(b) == "INT. BARN - DAY\n" })
	var ops []Op
	b.Do(func(d *Doc) { ops = d.SetText("INT. BARN - NIGHT\n") })
	b.Local(ops)
	waitFor("a gets b's edit", func() bool { return text(a) == "INT. BARN - NIGHT\n" })

	c := NewSession(key, New("c"))
	newPeerConnection = peerC
	Join(ctx, c, key, []string{tracker}, nil)
	waitFor("c catches up", func() bool { return text(c) == "INT. BARN - NIGHT\n" })
}

// The same through the public trackers (COLLAB_PUBLIC=1: it needs the
// internet and the trackers to be up).
func TestPublicTrackers(t *testing.T) {
	if os.Getenv("COLLAB_PUBLIC") == "" {
		t.Skip("set COLLAB_PUBLIC=1 to try the public trackers")
	}
	for _, tr := range Trackers {
		t.Run(tr, func(t *testing.T) {
			key, _ := NewKey()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			a, b := NewSession(key, New("a")), NewSession(key, New("b"))
			a.Do(func(d *Doc) { d.SetText("FADE IN:\n") })
			Join(ctx, a, key, []string{tr}, nil)
			time.Sleep(2 * time.Second)
			nb := Join(ctx, b, key, []string{tr}, nil)
			text := func(s *Session) (t string) { s.Do(func(d *Doc) { t = d.Text() }); return }
			for deadline := time.Now().Add(45 * time.Second); text(b) != "FADE IN:\n"; time.Sleep(200 * time.Millisecond) {
				if time.Now().After(deadline) {
					t.Fatalf("no sync (%d peers)", nb.Peers())
				}
			}
		})
	}
}
