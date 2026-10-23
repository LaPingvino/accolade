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
// edit together.
func TestNetwork(t *testing.T) {
	was := ICEServers
	ICEServers = nil // this machine only
	t.Cleanup(func() { ICEServers = was })
	tracker := fakeTracker(t)
	key, _ := NewKey()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a, b := NewSession(key, New("a")), NewSession(key, New("b"))
	a.Do(func(d *Doc) { d.SetText("INT. BARN - DAY\n") })
	na := Join(ctx, a, key, []string{tracker}, nil)
	time.Sleep(time.Second) // a's offers wait at the tracker for b
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
