package main

import (
	"testing"
	"time"
)

// waitFor waits (up to 5 s) until ok holds; shared by the background tests
// (built without -race) and others that run under -race too.
func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	// The test driver runs fyne.Do's callback on the background goroutine
	// itself, under uiTurn: checking under uiTurn too means a condition is
	// seen only after the callback that made it true has finished, so the
	// test's next UI call doesn't run beside it (CI, macOS, 2026-11-20: a
	// label refreshed from both at once panicked).
	check := func() bool {
		uiTurn.Lock()
		defer uiTurn.Unlock()
		return ok()
	}
	for !check() {
		if time.Now().After(deadline) {
			t.Fatalf("waited in vain for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
