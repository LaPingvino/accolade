package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRecentFiles(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.settings.SetStringSlice("recent-files", nil)
	t.Cleanup(func() { w.settings.SetStringSlice("recent-files", nil) })
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.fountain"), filepath.Join(dir, "b.fountain")
	os.WriteFile(a, []byte("INT. A - DAY\n"), 0o644)
	os.WriteFile(b, []byte("INT. B - DAY\n"), 0o644)
	w.LoadFile(a)
	w.LoadFile(b)
	w.LoadFile(a) // again: to the top, not twice
	got := w.settings.GetStringSlice("recent-files")
	if len(got) != 2 || got[0] != a || got[1] != b {
		t.Fatalf("recent %v", got)
	}
	labels := func() []string {
		var ls []string
		for _, it := range w.recentMenu().ChildMenu.Items {
			ls = append(ls, it.Label)
		}
		return ls
	}
	if ls := labels(); len(ls) != 4 || ls[0] != "1  a.fountain" || ls[1] != "2  b.fountain" || ls[3] != "Clear List" {
		t.Errorf("menu %q", ls)
	}
	os.Remove(b) // gone: not offered
	if ls := labels(); len(ls) != 3 || ls[0] != "1  a.fountain" {
		t.Errorf("menu after removing b %q", ls)
	}
}
