package main

// The tests run debounced and background work at once, on the test's
// goroutine, as the window did before it had either: the test driver
// runs fyne.Do on the calling goroutine, so a timer or a goroutine would
// run UI code beside the test's. The real paths have tests of their own
// (background_real_test.go, not under -race).
func init() {
	debounceAtOnce = true
	backgroundAtOnce = true
}
