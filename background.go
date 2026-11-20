package main

import (
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/accolade/internal/pace"
)

// Timers and background work, for a window.
//
// A timer's function runs on the UI goroutine (fyne.Do), and never after
// the timer is stopped or its window closed: a window's timers stop with
// it (an auto-save due in a closed window wrote into a window no longer
// there, and in the tests into the next test's).
//
// Background work runs in a goroutine and hands its result to the UI
// goroutine; long loops in it call pace.Yield so that the web version's
// page is not frozen meanwhile (internal/pace).

// uiTimer is a timer of a window's.
type uiTimer struct {
	t       *time.Timer
	stopped atomic.Bool
	owner   *uiTimers
}

// Stop stops the timer; its function does not run after.
func (t *uiTimer) Stop() {
	if t == nil {
		return
	}
	t.stopped.Store(true)
	t.t.Stop()
	t.owner.drop(t)
}

// uiTimers are a window's timers.
type uiTimers struct {
	mu     sync.Mutex
	set    map[*uiTimer]bool
	closed bool
}

func (ts *uiTimers) drop(t *uiTimer) {
	ts.mu.Lock()
	delete(ts.set, t)
	ts.mu.Unlock()
}

// after runs f on the UI goroutine after d, unless the timer is stopped
// first (or the window closed).
func (w *MainWindow) after(d time.Duration, f func()) *uiTimer {
	ts := &w.timers
	t := &uiTimer{owner: ts}
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.closed {
		t.stopped.Store(true)
		t.t = time.NewTimer(time.Hour)
		t.t.Stop()
		return t
	}
	if ts.set == nil {
		ts.set = map[*uiTimer]bool{}
	}
	ts.set[t] = true
	t.t = time.AfterFunc(d, func() {
		if t.stopped.Load() {
			return
		}
		fyne.Do(func() {
			if t.stopped.Load() {
				return
			}
			ts.drop(t)
			f()
		})
	})
	return t
}

// stopTimers stops the window's timers, for good: it is closed.
func (w *MainWindow) stopTimers() {
	ts := &w.timers
	ts.mu.Lock()
	ts.closed = true
	set := ts.set
	ts.set = nil
	ts.mu.Unlock()
	for t := range set {
		t.stopped.Store(true)
		t.t.Stop()
	}
}

// debounceAtOnce makes debounce run its function at once, as it was
// before there was a debounce: the tests' (a timer's function runs on
// the timer's goroutine under the test driver, beside the test's).
var debounceAtOnce = false

// debounce runs f after d without another call for the same slot: the
// last of a burst (typing, a drag) runs, once.
func (w *MainWindow) debounce(slot **uiTimer, d time.Duration, f func()) {
	(*slot).Stop()
	*slot = nil
	if debounceAtOnce {
		f()
		return
	}
	var t *uiTimer
	t = w.after(d, func() {
		if *slot == t {
			*slot = nil
		}
		f()
	})
	*slot = t
}

// cancel stops what is pending in the slot.
func cancel(slot **uiTimer) {
	(*slot).Stop()
	*slot = nil
}

// backgroundAtOnce makes inBackground work on the calling goroutine, the
// result given at once: the tests' (see debounceAtOnce).
var backgroundAtOnce = false

// inBackground runs work in a goroutine, then done on the UI goroutine
// with its result. Work that loops long calls pace.Yield.
func inBackground[T any](work func() T, done func(T)) {
	if backgroundAtOnce {
		done(work())
		return
	}
	go func() {
		r := work()
		pace.Yield()
		fyne.Do(func() { done(r) })
	}()
}

// stretchGap is the pause between the stretches of work repeat does on
// the UI goroutine: a frame, for the page to draw and take input.
const stretchGap = 16 * time.Millisecond

// repeat runs step on the UI goroutine again and again, a pause between,
// until it reports there is nothing left; the slot holds the next run
// (a new repeat in it, or cancel, stops it).
func (w *MainWindow) repeat(slot **uiTimer, step func() bool) {
	cancel(slot)
	if debounceAtOnce {
		for step() {
		}
		return
	}
	var run func()
	run = func() { // the slot is empty only once it is all done
		if step() {
			*slot = w.after(stretchGap, run)
		} else {
			*slot = nil
		}
	}
	*slot = w.after(0, run)
}
