// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fyne.io/fyne/v2/theme"

	"github.com/LaPingvino/accolade/internal/scriptdiff"
)

// The Compare tab (LOOP.md, CONTEXTUAL RIBBON TABS (6)): the ribbon's
// contextual tab while Compare is the middle tab in front (Collaboration >
// Compare…, Drafts' Compare, Production's Compare with the Last Issue),
// brought forward as it comes: Previous and Next Change (the changed
// scenes, round), Side by Side, Close; merging (File > Merge Changes…,
// merge_view.go) Take, Leave and Take All From This Scene on the change
// in view, and Merge Taken. Restoring a whole draft is in Drafts….

// The rules of the Compare tab's commands.
func comparing(e cmdEnv) bool         { return e.compare }
func compareChanges(e cmdEnv) bool    { return e.compare && e.compareChanged }
func compareSideBySide(e cmdEnv) bool { return e.compare && !e.phone }
func merging(e cmdEnv) bool          { return e.compare && e.merging }
func mergeChanges(e cmdEnv) bool     { return merging(e) && e.compareChanged }
func mergeTaken(e cmdEnv) bool       { return merging(e) && e.mergeTaken }

func (w *MainWindow) compareEnv(e *cmdEnv) {
	e.compare = w.compare != nil && w.mid.shown == "compare"
	e.compareChanged = e.compare && len(w.compare.changes) > 0
	e.merging = e.compare && w.compare.merge != nil
	e.mergeTaken = e.merging && w.compare.merge.m != nil && scriptdiff.Taken(w.compare.merge.take, w.compare.merge.edits) > 0
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
		{id: "compare.take", label: "Take", tip: "Take this change into the script, then the next", ex: "take",
			icon: theme.ConfirmIcon, when: merging, enabled: mergeChanges, run: on((*compareView).takeCurrent)},
		{id: "compare.leave", label: "Leave", tip: "Leave the script as it is here, then the next change", ex: "leave",
			icon: theme.ContentClearIcon, when: merging, enabled: mergeChanges, run: on((*compareView).leaveCurrent)},
		{id: "compare.takeScene", label: "Take All From This Scene", ribbonLabel: "Take Scene", tip: "Take every change in this change's scene", ex: "take-scene",
			icon: theme.ContentAddIcon, when: merging, enabled: mergeChanges, run: on((*compareView).takeCurrentScene)},
		{id: "compare.merge", label: "Merge Taken Changes", ribbonLabel: "Merge", tip: "Write what is taken into the script, as a new draft", ex: "merge-taken",
			icon: res(draftIcon), when: merging, enabled: mergeTaken, run: on((*compareView).doMerge)},
		{id: "compare.close", label: "Close Compare", ribbonLabel: "Close", icon: theme.CancelIcon, when: comparing,
			run: (*MainWindow).closeCompare},
	}
}

// compareTab is the Compare tab's layout.
func compareTab() ribbonTab {
	return ribbonTab{title: "Compare", when: comparing, contextual: true, forward: true, groups: []ribbonGroup{
		{title: "Changes", homes: []home{big("compare.next"), at("compare.previous")}},
		{title: "Merge", when: merging, homes: []home{big("compare.take"), at("compare.leave"), at("compare.takeScene"), big("compare.merge")}},
		{title: "Show", when: compareSideBySide, homes: []home{big("set.draftsSide")}},
		{title: "Close", homes: []home{big("compare.close")}},
	}}
}

// compareMenu is the Compare menu, while Compare is in front (the Menus look).
func compareMenu() []home {
	return []home{at("compare.previous"), at("compare.next"),
		sep(), at("compare.take"), at("compare.leave"), at("compare.takeScene"), at("compare.merge"),
		sep(), at("compare.close")} // Side by side: the view's own check there
}
