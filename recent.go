package main

import (
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
)

// File > Open Recent: the scripts opened or saved last, newest first.

const maxRecent = 10

// rememberRecent puts path at the top of the recent files.
func (w *MainWindow) rememberRecent(path string) {
	if w.settings == nil || path == "" {
		return
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	list := []string{path}
	for _, p := range w.settings.GetStringSlice("recent-files") {
		if p != path && len(list) < maxRecent {
			list = append(list, p)
		}
	}
	w.settings.SetStringSlice("recent-files", list)
	w.settings.Save()
	if w.app != nil { // every window's menu shows the new list
		for _, win := range w.app.windows {
			win.fyneWindow.SetMainMenu(win.buildMainMenu())
		}
	}
}

// recentMenu is File > Open Recent: the recent files that still exist.
func (w *MainWindow) recentMenu() *fyne.MenuItem {
	item := fyne.NewMenuItem("Open Recent", nil)
	var items []*fyne.MenuItem
	if w.settings != nil {
		for _, p := range w.settings.GetStringSlice("recent-files") {
			if _, err := os.Stat(p); err != nil {
				continue
			}
			path := p
			label := filepath.Base(p)
			if n := len(items); n < 9 {
				label = fmt.Sprintf("%d  %s", n+1, label)
			}
			items = append(items, fyne.NewMenuItem(label, func() { w.openPath(path) }))
		}
	}
	if len(items) == 0 {
		none := fyne.NewMenuItem("No recent scripts", nil)
		none.Disabled = true
		items = append(items, none)
	} else {
		items = append(items, fyne.NewMenuItemSeparator(), fyne.NewMenuItem("Clear List", func() {
			w.settings.SetStringSlice("recent-files", nil)
			w.settings.Save()
			for _, win := range w.app.windows {
				win.fyneWindow.SetMainMenu(win.buildMainMenu())
			}
		}))
	}
	item.ChildMenu = fyne.NewMenu("", items...)
	return item
}
