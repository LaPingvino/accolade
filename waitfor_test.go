package main

import (
	"testing"
	"time"
)

// The tests of real timers and background work (realPaths, the locked
// pages') make the test's goroutine the UI goroutine: holdUI gives it the
// UI turn (uiTurn) for the whole test, and it gives the turn up only
// while it waits (waitFor, uiSleep). The test driver runs fyne.Do's
// callback on the calling goroutine; a timer's or a background result's
// (uiDo) thus waits for its turn and runs between the test's steps, one
// at a time, as the real drivers run them on the UI goroutine. So these
// tests run under -race too.

// uiHeld: the test's goroutine has the UI turn (holdUI). Only the test's
// goroutine reads and writes it.
var uiHeld bool

// holdUI gives the test's goroutine the UI turn until the test ends; call
// it first in the test, so the window's cleanup runs in the turn too.
func holdUI(t *testing.T) {
	t.Helper()
	if uiHeld {
		return
	}
	uiTurn.Lock()
	uiHeld = true
	t.Cleanup(func() {
		uiHeld = false
		uiTurn.Unlock()
	})
}

// uiSleep sleeps for d, the UI turn given up meanwhile when the test has
// it: what is due runs in the pause.
func uiSleep(d time.Duration) {
	if !uiHeld {
		time.Sleep(d)
		return
	}
	uiTurn.Unlock()
	time.Sleep(d)
	uiTurn.Lock()
}

// waitFor waits (up to 5 s) until ok holds; shared by the background tests
// and others.
func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	// The test driver runs fyne.Do's callback on the background goroutine
	// itself, under uiTurn: checking under uiTurn too means a condition is
	// seen only after the callback that made it true has finished, so the
	// test's next UI call doesn't run beside it (CI, macOS, 2026-11-20: a
	// label refreshed from both at once panicked). A test that holds the
	// turn (holdUI) checks in it and gives it up while it sleeps.
	check := func() bool {
		if uiHeld {
			return ok()
		}
		uiTurn.Lock()
		defer uiTurn.Unlock()
		return ok()
	}
	for !check() {
		if time.Now().After(deadline) {
			t.Fatalf("waited in vain for %s", what)
		}
		uiSleep(5 * time.Millisecond)
	}
}
