package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func menuItemByLabel(t *testing.T, w *MainWindow, label string) *fyne.MenuItem {
	t.Helper()
	for _, m := range w.fyneWindow.MainMenu().Items {
		for _, it := range m.Items {
			if it.Label == label {
				return it
			}
		}
	}
	t.Fatalf("no menu item %q", label)
	return nil
}

func TestMenuShortcutsAreUniqueAndLeaveEditingKeysAlone(t *testing.T) {
	w := newTestWindow(t, "")
	if w.fyneWindow.MainMenu() == nil {
		t.Fatal("window has no main menu")
	}

	// the editor handles these itself; a menu shortcut would steal them
	reserved := map[fyne.KeyName]bool{fyne.KeyZ: true, fyne.KeyY: true, fyne.KeyC: true, fyne.KeyV: true, fyne.KeyX: true, fyne.KeyA: true}
	seen := map[string]string{}
	for _, m := range w.fyneWindow.MainMenu().Items {
		for _, it := range m.Items {
			if it.Shortcut == nil {
				continue
			}
			name := it.Shortcut.ShortcutName()
			if prev, dup := seen[name]; dup {
				t.Errorf("%s and %s share shortcut %s", prev, it.Label, name)
			}
			seen[name] = it.Label
			if ks, ok := it.Shortcut.(fyne.KeyboardShortcut); ok && reserved[ks.Key()] && ks.Mod() == fyne.KeyModifierShortcutDefault {
				t.Errorf("%s takes editor shortcut %s", it.Label, name)
			}
		}
	}
	if len(seen) < 10 {
		t.Errorf("only %d shortcuts registered", len(seen))
	}
}

func TestFindMenuOpensSearchAndEscapeCloses(t *testing.T) {
	w := newTestWindow(t, "INT. HOUSE - DAY")
	menuItemByLabel(t, w, "Find…").Action()
	if !w.searchBar.IsVisible() || w.searchBar.replaceMode {
		t.Fatal("Find should show the search bar without the replace row")
	}
	if w.fyneWindow.Canvas().Focused() != w.searchBar.searchEntry {
		t.Error("Find should focus the search field")
	}

	test.Type(w.searchBar.searchEntry, "house")
	menuItemByLabel(t, w, "Find Next").Action()
	if got := w.textEditor.SelectedText(); got != "HOUSE" {
		t.Errorf("Find Next selected %q", got)
	}

	w.searchBar.searchEntry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if w.searchBar.IsVisible() {
		t.Error("Escape should close the search bar")
	}
	if w.fyneWindow.Canvas().Focused() != w.textEditor {
		t.Error("closing the search bar should focus the editor")
	}

	menuItemByLabel(t, w, "Replace…").Action()
	if !w.searchBar.IsVisible() || !w.searchBar.replaceMode {
		t.Error("Replace should show the search bar with the replace row")
	}
}
