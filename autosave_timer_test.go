//go:build !race

// The test driver runs fyne.Do callbacks on the timer goroutine instead of
// a UI thread, which the race detector rightly flags; the real GLFW
// driver serialises them. So this one end-to-end timer test is skipped
// under -race.

package main

import (
	"os"
	"testing"
	"time"
)

func TestAutoSaveTimerFires(t *testing.T) {
	w, path := autoSaveWindow(t, true)
	w.textEditor.SetText("Rain on the window.")

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if data, _ := os.ReadFile(path); string(data) == "Rain on the window." {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("auto-save timer never wrote the file")
}
