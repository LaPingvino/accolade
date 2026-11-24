package main

// What a peer's edit costs the window it reaches (docs/COLLAB.md, What is
// left 8): two windows sharing the PERF-SOURCE bench script with the
// novel beside it, linked in-process, the receiving window's source pane
// shown with its passages read. A peer types a key; what is timed is the
// receiving window taking its message in: the session's apply, the
// source document following it, the pane's passages, the redraws.
// Background work runs at once (counted: in the browser it shares the
// one thread); what waits for a pause is cancelled. Run them with
//
//	go test -run '^$' -bench PeerKey -benchmem -benchtime 10x

import (
	"os"
	"testing"

	"github.com/LaPingvino/accolade/internal/collab"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// cancelPending stops a window's timers now waiting (the UI turn held,
// none has begun), so nothing put off runs inside a measurement.
func cancelPending(w *MainWindow) {
	ts := &w.timers
	ts.mu.Lock()
	set := ts.set
	ts.set = nil
	ts.mu.Unlock()
	for t := range set {
		t.stopped.Store(true)
		t.t.Stop()
	}
}

// peerPair is a (the bench script, the novel beside it, the pane shown)
// sharing with b, who joined and has the source. Messages go by queue:
// deliver takes in those waiting, a's first, until none are.
func peerPair(b *testing.B) (a, p *MainWindow, deliver func(toA bool)) {
	was := collab.Trackers
	collab.Trackers = nil
	b.Cleanup(func() { collab.Trackers = was })
	a, _, _ = benchWindow(b, false)
	p = a.app.newWindow()
	key, _ := collab.NewKey()
	a.startSharing(key, false)
	_ = os.Remove(keptSharedPath(key)) // p starts from nothing
	_ = os.Remove(ownMarkPath(key))
	p.startSharing(key, true)
	b.Cleanup(func() { a.stopSharing(); p.stopSharing() })
	sa, sp := a.shared.session, p.shared.session
	at := func(f func()) { f() }
	sa.Run, sp.Run = at, at
	var toA, toP [][]byte
	sa.AddPeer("p", func(m []byte) error { toP = append(toP, append([]byte(nil), m...)); return nil })
	sp.AddPeer("a", func(m []byte) error { toA = append(toA, append([]byte(nil), m...)); return nil })
	deliver = func(onlyA bool) {
		for len(toA) > 0 || !onlyA && len(toP) > 0 {
			if len(toA) > 0 {
				m := toA[0]
				toA = toA[1:]
				_ = sa.Receive("p", m)
				continue
			}
			m := toP[0]
			toP = toP[1:]
			_ = sp.Receive("a", m)
		}
	}
	deliver(false)
	if p.srcDoc == nil || p.srcDoc.Text() != a.srcDoc.Text() || a.source.src != a.srcDoc.Text() {
		b.Fatal("p has not the source")
	}
	return a, p, deliver
}

// type1 types a key in a document at off, or takes it away again.
func type1(ed interface {
	Edit(func(*buffer.Buffer))
}, off, i int) {
	ed.Edit(func(e *buffer.Buffer) {
		if i%2 == 0 {
			e.SetCursor(off, false)
			e.Insert("x")
		} else {
			e.Replace(off, off+1, "")
		}
	})
}

func BenchmarkPeerKey(b *testing.B) {
	const off = 5000 // in the novel's first chapter
	setup := func(b *testing.B) (a, p *MainWindow, deliver func(bool)) {
		a, p, deliver = peerPair(b)
		// timers as the app has them: what waits for a pause (the follow,
		// the kept state, the rows' measuring after the first stretch) is
		// cancelled before it runs; the background's work runs at once
		debounceAtOnce = false
		uiTurn.Lock() // a timer's function waits for the turn, then sees it was stopped
		b.Cleanup(func() { cancelPending(a); cancelPending(p); uiTurn.Unlock(); debounceAtOnce = true })
		a.fyneWindow.Canvas().Focus(a.srcDoc.ed)
		return a, p, deliver
	}
	// a's own key in its source: what runs at once (the Doc's SetText, the
	// message out); the pane follows after a pause
	b.Run("local-source", func(b *testing.B) {
		a, _, _ := setup(b)
		i := 0
		for b.Loop() {
			type1(a.srcDoc.ed, off, i)
			b.StopTimer()
			cancelPending(a)
			b.StartTimer()
			i++
		}
	})
	// p's key in the source, taken in by a
	b.Run("peer-source", func(b *testing.B) {
		a, p, deliver := setup(b)
		i := 0
		for b.Loop() {
			b.StopTimer()
			type1(p.srcDoc.ed, off, i)
			cancelPending(p)
			b.StartTimer()
			deliver(true)
			b.StopTimer()
			cancelPending(a)
			b.StartTimer()
			i++
		}
		if a.srcDoc.Text() != p.srcDoc.Text() {
			b.Fatal("a's source did not follow")
		}
	})
	// the pane reading the source again once p pauses (once a burst)
	b.Run("peer-pause", func(b *testing.B) {
		a, p, deliver := setup(b)
		i := 0
		for b.Loop() {
			b.StopTimer()
			type1(p.srcDoc.ed, off, i)
			cancelPending(p)
			deliver(true)
			if a.source.src == a.srcDoc.Text() {
				b.Fatal("the pane read the source at once")
			}
			cancelPending(a)
			b.StartTimer()
			a.source.loadSource(true) // what the pause's timer does
			b.StopTimer()
			if a.source.src != a.srcDoc.Text() {
				b.Fatal("the pane did not read the source again")
			}
			cancelPending(a)
			b.StartTimer()
			i++
		}
	})
	// the same two with the source in front of a's middle (Open in the
	// Middle), so in sight: laid out at each key, where the rows above
	// (the source open only to follow its Doc) are out of sight and wait
	shown := func(b *testing.B) (a, p *MainWindow, deliver func(bool)) {
		a, p, deliver = setup(b)
		a.openMiddle("source")
		if a.midDoc() != a.srcDoc || a.srcDoc.ed.Offstage() {
			b.Fatal("the source is not in front")
		}
		cancelPending(a)
		a.fyneWindow.Canvas().Focus(a.srcDoc.ed)
		return a, p, deliver
	}
	b.Run("local-source-shown", func(b *testing.B) {
		a, _, _ := shown(b)
		i := 0
		for b.Loop() {
			type1(a.srcDoc.ed, off, i)
			b.StopTimer()
			cancelPending(a)
			b.StartTimer()
			i++
		}
	})
	b.Run("peer-source-shown", func(b *testing.B) {
		a, p, deliver := shown(b)
		i := 0
		for b.Loop() {
			b.StopTimer()
			type1(p.srcDoc.ed, off, i)
			cancelPending(p)
			b.StartTimer()
			deliver(true)
			b.StopTimer()
			cancelPending(a)
			b.StartTimer()
			i++
		}
		if a.srcDoc.Text() != p.srcDoc.Text() {
			b.Fatal("a's source did not follow")
		}
	})
	// a's own key in the script (the pane shown), against p's below
	b.Run("local-script", func(b *testing.B) {
		a, _, _ := setup(b)
		at := a.textEditor.Buffer().LineStart(40)
		i := 0
		for b.Loop() {
			type1(a.textEditor, at, i)
			b.StopTimer()
			cancelPending(a)
			b.StartTimer()
			i++
		}
	})
	// p's key in the script, taken in by a (the pane shown)
	b.Run("peer-script", func(b *testing.B) {
		a, p, deliver := setup(b)
		at := p.textEditor.Buffer().LineStart(40)
		i := 0
		for b.Loop() {
			b.StopTimer()
			type1(p.textEditor, at, i)
			cancelPending(p)
			b.StartTimer()
			deliver(true)
			b.StopTimer()
			cancelPending(a)
			b.StartTimer()
			i++
		}
		if a.textEditor.Text() != p.textEditor.Text() {
			b.Fatal("a did not follow")
		}
	})
}
