// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

// The Stripboard tab: forward as the board comes to the front (the tab
// chosen before back as it goes), its buttons following the selected
// strip, its key tips each their own; Production still there, on its
// Schedule stage; on a phone too; in the Menus look the Production menu
// has the commands while the board is in front.
func TestStripboardTab(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, schedScript)
	w.fyneWindow.Resize(fyne.NewSize(1200, 800))
	r := ribbonOnFor(t, w)
	r.selectTab("Edit")
	sb := board(t, w)
	if r.selected != "Stripboard" {
		t.Fatalf("the tab chosen: %q", r.selected)
	}
	press := func(id string) {
		t.Helper()
		b := buttonOf(r, id)
		if b == nil {
			t.Fatalf("no %s button", id)
		}
		b.press()
	}
	enabled := func(id string) bool {
		t.Helper()
		b := buttonOf(r, id)
		if b == nil {
			t.Fatalf("no %s button", id)
		}
		return !b.disabled
	}
	if enabled("strips.up") || enabled("strips.moveTo") || enabled("strips.open") {
		t.Error("a strip's commands with none selected")
	}
	press("production.addDay")
	sb.selectScene(1, false)
	if !enabled("strips.open") || !enabled("strips.moveTo") || enabled("strips.down") || enabled("strips.unschedule") {
		t.Error("the buttons do not follow the selection")
	}
	press("strips.up") // greyed: the garden is not the pool's top
	if got := days(w)[0]; got != "" {
		t.Errorf("up from the pool's second: %q", got)
	}
	sb.selectScene(0, false)
	press("strips.up") // the pool's top: onto the last day's end
	if got := days(w)[0]; got != "kitchen" {
		t.Fatalf("up onto the day: %q", got)
	}
	if !enabled("strips.unschedule") || !enabled("strips.undo") {
		t.Error("Unschedule or Undo greyed for a strip on a day")
	}
	if b := buttonOf(r, "strips.byLocation"); b == nil || b.name != "Group Day 1 by Location" {
		t.Errorf("Group's name: %+v", b)
	}
	press("strips.undo")
	if got := days(w)[0]; got != "" {
		t.Errorf("undone: %q", got)
	}
	r.showKeyTips()
	r.assignTips(2)
	seen := map[string]string{}
	for _, b := range r.buttons {
		if b.tip == "" {
			continue
		}
		if o, ok := seen[b.tip]; ok {
			t.Errorf("key tip %s for %s and %s", b.tip, o, b.label)
		}
		seen[b.tip] = b.label
	}
	r.leaveKeyTips(false)
	r.selectTab("Production") // still there, on the board's stage
	if r.stage != stageSchedule {
		t.Errorf("Production's stage %d", r.stage)
	}
	r.selectTab("Stripboard")
	w.showMiddle("")
	if r.selected != "Edit" {
		t.Errorf("the board left: the tab %q", r.selected)
	}
	w.showMiddle("strips")
	if r.selected != "Stripboard" {
		t.Errorf("back: the tab %q", r.selected)
	}
	w.fyneWindow.Resize(fyne.NewSize(380, 800)) // a phone keeps it in front
	w.onWindowWidth(380)
	if r.selected != "Stripboard" {
		t.Errorf("on a phone: the tab %q", r.selected)
	}
	w.fyneWindow.Resize(fyne.NewSize(1200, 800))
	w.onWindowWidth(1200)
	press("strips.close")
	if w.mid.has("strips") || r.selected != "Edit" {
		t.Errorf("closed: %q, the tab %q", midTitles(w), r.selected)
	}
	for _, tb := range r.shownTabs() {
		if tb.title == "Stripboard" {
			t.Error("the Stripboard tab is there with no board")
		}
	}
}

// In the Menus look the board's commands are the Production menu's while
// it is in front (and not otherwise); with nothing selected they say so.
func TestStripboardMenus(t *testing.T) {
	w, _ := scriptWindow(t, schedScript)
	production := func() string {
		for _, m := range w.fyneWindow.MainMenu().Items {
			if m.Label == "Production" {
				var ls []string
				for _, it := range m.Items {
					ls = append(ls, it.Label)
				}
				return strings.Join(ls, "|")
			}
		}
		return ""
	}
	if strings.Contains(production(), "Move Strip Up") {
		t.Error("the board's commands with no board")
	}
	sb := board(t, w)
	if !strings.Contains(production(), "Move Strip Up") || !strings.Contains(production(), "Close Stripboard") {
		t.Errorf("Production: %s", production())
	}
	menuItemByLabel(t, w, "Move Strip Up").Action()
	if !strings.Contains(w.statusLabel.Text, "No strip selected") {
		t.Errorf("no strip: %q", w.statusLabel.Text)
	}
	w.addShootDay()
	sb.selectScene(0, false)
	menuItemByLabel(t, w, "Move Strip Up").Action()
	if got := days(w)[0]; got != "kitchen" {
		t.Errorf("from the menu: %q", got)
	}
}
