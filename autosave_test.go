package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func autoSaveWindow(t *testing.T, enabled bool) (*MainWindow, string) {
	t.Helper()
	w := newTestWindow(t, "")
	prev := w.settings.GetBoolean("auto-save")
	w.settings.SetBoolean("auto-save", enabled)
	t.Cleanup(func() { w.settings.SetBoolean("auto-save", prev); w.cancelAutoSave() })

	path := filepath.Join(t.TempDir(), "script.fountain")
	if err := os.WriteFile(path, []byte("FADE IN:"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := w.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	w.autoSaveDelay = 50 * time.Millisecond
	return w, path
}

// Fyne's test driver runs fyne.Do callbacks on the calling goroutine, so a
// real timer would touch the window off the test goroutine. Tests check
// that a save is pending and then fire it themselves; only
// TestAutoSaveTimerFires (autosave_timer_test.go) lets the timer run.

func fireAutoSave(t *testing.T, w *MainWindow) {
	t.Helper()
	if w.autoSaveTimer == nil {
		t.Fatal("no auto-save pending")
	}
	w.autoSaveTimer.Stop()
	w.autoSaveNow()
}

func TestAutoSaveWritesFile(t *testing.T) {
	w, path := autoSaveWindow(t, true)
	w.autoSaveDelay = time.Hour
	w.textEditor.SetText("A quiet room.\n\nJohn waits.")
	fireAutoSave(t, w)

	if w.hasChanges {
		t.Error("document still marked changed after auto-save")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "A quiet room.\n\nJohn waits." {
		t.Errorf("file = %q", data)
	}
	// Windows has no Unix permission bits to keep
	if info, _ := os.Stat(path); runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Errorf("mode changed to %v", info.Mode().Perm())
	}
	if !strings.HasPrefix(w.statusLabel.Text, "Auto-saved") {
		t.Errorf("status = %q", w.statusLabel.Text)
	}
}

func TestAutoSaveIsNotPostponedByTyping(t *testing.T) {
	w, _ := autoSaveWindow(t, true)
	w.autoSaveDelay = time.Hour
	w.textEditor.SetText("one")
	first := w.autoSaveTimer
	w.textEditor.SetText("one two")
	if first == nil || w.autoSaveTimer != first {
		t.Error("further edits should keep the pending auto-save, not restart it")
	}
}

func TestAutoSaveRespectsSettingAndUntitled(t *testing.T) {
	w, path := autoSaveWindow(t, false)
	w.textEditor.SetText("changed")
	if w.autoSaveTimer != nil {
		t.Error("auto-save disabled but a save was scheduled")
	}
	time.Sleep(100 * time.Millisecond)
	if data, _ := os.ReadFile(path); string(data) != "FADE IN:" {
		t.Errorf("file written with auto-save off: %q", data)
	}

	u := newTestWindow(t, "")
	u.textEditor.SetText("untitled draft")
	if u.autoSaveTimer != nil {
		t.Error("untitled document should not schedule an auto-save")
	}
}

func TestManualSaveCancelsPendingAutoSave(t *testing.T) {
	w, path := autoSaveWindow(t, true)
	w.autoSaveDelay = time.Hour
	w.textEditor.SetText("draft")
	if err := w.SaveFile(); err != nil {
		t.Fatal(err)
	}
	if w.autoSaveTimer != nil {
		t.Error("save should cancel the pending auto-save")
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("leftover files next to the script: %v", entries)
	}
}
