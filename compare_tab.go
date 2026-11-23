package main

import (
	"fyne.io/fyne/v2/theme"
)

// The Compare tab (LOOP.md, CONTEXTUAL RIBBON TABS (6)): the ribbon's
// contextual tab while Compare is the middle tab in front (Collaboration >
// Compare…, Drafts' Compare, Production's Compare with the Last Issue),
// brought forward as it comes: Previous and Next Change (the changed
// scenes, round), Side by Side, Close. Compare is read only: taking a
// change over or keeping the old one (Accept/Keep) is for when it can
// edit; restoring a whole draft is in Drafts….

// The rules of the Compare tab's commands.
func comparing(e cmdEnv) bool         { return e.compare }
func compareChanges(e cmdEnv) bool    { return e.compare && e.compareChanged }
func compareSideBySide(e cmdEnv) bool { return e.compare && !e.phone }

func (w *MainWindow) compareEnv(e *cmdEnv) {
	e.compare = w.compare != nil && w.mid.shown == "compare"
	e.compareChanged = e.compare && len(w.compare.changes) > 0
}

// compareCommands are the Compare tab's.
func compareCommands() []*command {
	on := func(f func(cv *compareView)) func(w *MainWindow) {
		return func(w *MainWindow) {
			if w.compare != nil {
				f(w.compare)
			}
		}
	}
	return []*command{
		{id: "compare.previous", label: "Previous Change", ribbonLabel: "Previous", tip: "The changed scene before", ex: "previous-change",
			icon: theme.NavigateBackIcon, when: comparing, enabled: compareChanges, run: on(func(cv *compareView) { cv.step(-1) })},
		{id: "compare.next", label: "Next Change", ribbonLabel: "Next", tip: "The next changed scene", ex: "next-change",
			icon: theme.NavigateNextIcon, when: comparing, enabled: compareChanges, run: on(func(cv *compareView) { cv.step(1) })},
		{id: "compare.close", label: "Close Compare", ribbonLabel: "Close", icon: theme.CancelIcon, when: comparing,
			run: (*MainWindow).closeCompare},
	}
}

// compareTab is the Compare tab's layout.
func compareTab() ribbonTab {
	return ribbonTab{title: "Compare", when: comparing, contextual: true, forward: true, groups: []ribbonGroup{
		{title: "Changes", homes: []home{big("compare.next"), at("compare.previous")}},
		{title: "Show", when: compareSideBySide, homes: []home{big("set.draftsSide")}},
		{title: "Close", homes: []home{big("compare.close")}},
	}}
}

// compareMenu is the Compare menu, while Compare is in front (the Menus look).
func compareMenu() []home {
	return []home{at("compare.previous"), at("compare.next"), sep(), at("compare.close")} // Side by side: the view's own check there
}
