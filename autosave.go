package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
)

// Auto-save writes the open file a fixed delay after the first unsaved
// change. The delay is not pushed back by further typing, so during a
// long writing streak the file is still saved every interval.

const defaultAutoSaveInterval = 30 * time.Second

func (w *MainWindow) autoSaveInterval() time.Duration {
	if w.autoSaveDelay > 0 {
		return w.autoSaveDelay
	}
	if secs := w.settings.GetInt("auto-save-interval"); secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return defaultAutoSaveInterval
}

// scheduleAutoSave arms the auto-save timer unless it is already pending.
func (w *MainWindow) scheduleAutoSave() {
	if w.currentFile == "" || !w.settings.GetBoolean("auto-save") || w.autoSaveTimer != nil {
		return
	}
	w.autoSaveTimer = time.AfterFunc(w.autoSaveInterval(), func() {
		fyne.Do(w.autoSaveNow)
	})
}

func (w *MainWindow) autoSaveNow() {
	w.autoSaveTimer = nil
	if !w.hasChanges || w.currentFile == "" || !w.settings.GetBoolean("auto-save") {
		return
	}
	if err := w.saveToFile(w.currentFile); err != nil {
		log.Printf("Auto-save failed: %v", err)
		w.setStatus("Auto-save failed: " + err.Error())
		return
	}
	w.setStatus("Auto-saved " + time.Now().Format("15:04"))
}

func (w *MainWindow) cancelAutoSave() {
	if w.autoSaveTimer != nil {
		w.autoSaveTimer.Stop()
		w.autoSaveTimer = nil
	}
}

// writeFileAtomic writes data to a temporary file next to path and renames
// it into place, so a crash mid-save never leaves a truncated screenplay.
func writeFileAtomic(path string, data []byte) error {
	mode := os.FileMode(0644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op once renamed

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
