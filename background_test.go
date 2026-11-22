package main

import (
	"cmp"
	"slices"
	"time"
)

// The tests run debounced and background work at once, on the test's
// goroutine, as the window did before it had either: the test driver
// runs fyne.Do on the calling goroutine, so a timer or a goroutine would
// run UI code beside the test's. The real paths have tests of their own
// (background_real_test.go, not under -race).
func init() {
	debounceAtOnce = true
	backgroundAtOnce = true
}

// heldDebounce runs burst (typing) with the debounced work held, then
// runs each slot's last function once, shortest pause first: what the
// app does after a pause in typing, where the tests' debounce otherwise
// runs on every key (debounceAtOnce).
func heldDebounce(burst func()) {
	if debounceHeld != nil || !debounceAtOnce {
		burst()
		return
	}
	h := &heldDebounces{pause: map[**uiTimer]time.Duration{}, f: map[**uiTimer]func(){}}
	debounceHeld = h
	func() {
		defer func() { debounceHeld = nil }()
		burst()
	}()
	slices.SortStableFunc(h.order, func(a, b **uiTimer) int { return cmp.Compare(h.pause[a], h.pause[b]) })
	for _, slot := range h.order {
		if f, ok := h.f[slot]; ok {
			f()
		}
	}
}
