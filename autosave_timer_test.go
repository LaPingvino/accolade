// The auto-save timer as the app has it; the test holds the UI turn
// (holdUI, waitfor_test.go), so it runs under -race too.

package main

import (
	"os"
	"testing"
	"time"
)

func TestAutoSaveTimerFires(t *testing.T) {
	holdUI(t)
	w, path := autoSaveWindow(t, true)
	w.textEditor.SetText("Rain on the window.")

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if data, _ := os.ReadFile(path); string(data) == "Rain on the window." {
			return
		}
		uiSleep(10 * time.Millisecond)
	}
	t.Fatal("auto-save timer never wrote the file")
}
