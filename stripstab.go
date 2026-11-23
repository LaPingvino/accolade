package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The Stripboard tab: the ribbon's contextual tab while the stripboard is
// the middle tab in front (stripboard.go), brought forward when it comes
// and gone when it goes, as the Preview tab is (previewtab.go): the
// selected strip's moves (Move To, Up, Down, Unschedule, Undo), the days
// (Add Day, Edit Day, folding them), the one-off grouping, the check of a
// changed scene (Open Scene, Checked), Close. The board has no toolbar of
// its own in either look: the Menus look's Production menu has the same
// commands while the board is in front, a strip's long press or
// right-click its own, the palette and : all of them.

// stripState is what the Stripboard tab's commands look at: the board's
// selected strip (its day, -1 the unscheduled), whether it can go up or
// down, whether it is flagged as changed; the days; whether the strips
// are measured (grouping needs their cast and places); the board's undo.
type stripState struct {
	selected, up, down, flag, measured, undo bool
	day, days                                int
}

// stripsEnv fills in what the Stripboard tab's rules look at.
func (w *MainWindow) stripsEnv(e *cmdEnv) {
	e.board = stripState{day: -1}
	if e.strips {
		e.board = w.stripboard.state
	}
}

// The rules of the Stripboard tab's commands.
func stripsInFront(e cmdEnv) bool { return e.strips }
func stripPicked(e cmdEnv) bool   { return e.board.selected }
func stripCanUp(e cmdEnv) bool    { return e.board.up }
func stripCanDown(e cmdEnv) bool  { return e.board.down }
func stripOnDay(e cmdEnv) bool    { return e.board.day >= 0 }
func stripFlagged(e cmdEnv) bool  { return e.board.flag }
func stripsUndo(e cmdEnv) bool    { return e.board.undo }
func stripsGroup(e cmdEnv) bool   { return e.board.measured }
func stripsDays(e cmdEnv) bool    { return e.board.days > 0 }
func stripMovable(e cmdEnv) bool  { return e.board.selected && (e.board.day >= 0 || e.board.days > 0) }

// onBoard is a Stripboard command's run: f with the board in front.
func onBoard(f func(sb *stripboardView)) func(w *MainWindow) {
	return func(w *MainWindow) {
		if !w.stripsShown() {
			w.setStatus("The stripboard is not in front: Production > Stripboard")
			return
		}
		f(w.stripboard)
	}
}

// onStrip is a strip command's run: f with the strip selected.
func onStrip(f func(sb *stripboardView, sc int)) func(w *MainWindow) {
	return onBoard(func(sb *stripboardView) {
		if sb.selected < 0 {
			sb.w.setStatus("No strip selected: tap one (or Tab to it) first")
			return
		}
		f(sb, sb.selected)
	})
}

// groupScope is what Group sorts: the selected strip's day, else the
// unscheduled.
func (w *MainWindow) groupScope() string {
	if w.stripsShown() && w.stripboard.state.day >= 0 {
		return fmt.Sprintf("Day %d", w.stripboard.state.day+1)
	}
	return "the Unscheduled"
}

// stripsCommands are the Stripboard tab's (Add Day is Production's).
func stripsCommands() []*command {
	strip := func(c *command) *command {
		c.role = productionRole
		if c.when == nil {
			c.when = stripsInFront
		}
		return c
	}
	group := func(by, label string, icon fyne.Resource) *command {
		return strip(&command{id: "strips.by" + label, label: "Group by " + label, ribbonLabel: "By " + label, icon: res(icon),
			enabled: stripsGroup,
			nameFor: func(w *MainWindow) string { return "Group " + w.groupScope() + " by " + label },
			run:     onBoard(func(sb *stripboardView) { sb.group(by) })})
	}
	return []*command{
		strip(&command{id: "strips.moveTo", ex: "move-strip-to", label: "Move Strip To…", ribbonLabel: "Move To", tip: "Move the selected strip to a day, or back to the unscheduled",
			icon: res(theme.MailForwardIcon()), enabled: stripMovable,
			pick: func(w *MainWindow, anchor fyne.CanvasObject) {
				onStrip(func(sb *stripboardView, sc int) { sb.showMoveMenu(sc, anchor) })(w)
			}}),
		strip(&command{id: "strips.up", label: "Move Strip Up", ribbonLabel: "Up", tip: "Move the selected strip up (Alt+Up)",
			icon: res(theme.MoveUpIcon()), enabled: stripCanUp, run: onStrip(func(sb *stripboardView, _ int) { sb.shift(-1) })}),
		strip(&command{id: "strips.down", label: "Move Strip Down", ribbonLabel: "Down", tip: "Move the selected strip down (Alt+Down)",
			icon: res(theme.MoveDownIcon()), enabled: stripCanDown, run: onStrip(func(sb *stripboardView, _ int) { sb.shift(1) })}),
		strip(&command{id: "strips.unschedule", label: "Unschedule Strip", ribbonLabel: "Unschedule", tip: "Take the selected strip off its day",
			icon: res(theme.ContentRemoveIcon()), enabled: stripOnDay, run: onStrip(func(sb *stripboardView, sc int) { sb.moveScene(sc, -1, 0) })}),
		strip(&command{id: "strips.undo", ex: "undo-board-change", label: "Undo Board Change", ribbonLabel: "Undo", tip: "Take the board's last change back",
			icon: res(theme.ContentUndoIcon()), enabled: stripsUndo, run: func(w *MainWindow) { w.undoSchedule() }}),
		strip(&command{id: "strips.editDay", label: "Edit Day…", tip: "The selected strip's day: its date, call and notes",
			icon: res(theme.DocumentCreateIcon()), enabled: stripOnDay, run: onStrip(func(sb *stripboardView, _ int) { sb.editDay(sb.state.day) })}),
		strip(&command{id: "strips.fold", label: "Fold All Days", ribbonLabel: "Fold Days", tip: "Every day folded to its band: its scenes and length",
			icon: res(foldIcon), enabled: stripsDays, run: onBoard(func(sb *stripboardView) { sb.foldAll(true) })}),
		strip(&command{id: "strips.unfold", label: "Unfold All Days", ribbonLabel: "Unfold Days", tip: "Every day's strips shown",
			icon: res(unfoldIcon), enabled: stripsDays, run: onBoard(func(sb *stripboardView) { sb.foldAll(false) })}),
		group("location", "Location", theme.ListIcon()),
		group("cast", "Cast", theme.AccountIcon()),
		strip(&command{id: "strips.open", ex: "open-strip-scene", label: "Open Strip's Scene", ribbonLabel: "Open Scene", tip: "The selected strip's scene in the script (the board's tab kept)",
			icon: res(theme.DocumentIcon()), enabled: stripPicked, run: onStrip(func(sb *stripboardView, sc int) { sb.openScene(sc) })}),
		strip(&command{id: "strips.checked", ex: "strip-checked", label: "Strip Checked", ribbonLabel: "Checked", tip: "The selected strip's scene changed since it was scheduled: checked, its flag taken away",
			icon: res(theme.ConfirmIcon()), enabled: stripFlagged, run: onStrip(func(sb *stripboardView, _ int) { sb.checkSelected() })}),
		strip(&command{id: "strips.close", label: "Close Stripboard", ribbonLabel: "Close", icon: res(theme.CancelIcon()),
			run: func(w *MainWindow) { w.closeMiddle("strips") }}),
	}
}

// stripsTab is the Stripboard tab's layout.
func stripsTab() ribbonTab {
	return ribbonTab{title: "Stripboard", when: stripsInFront, role: productionRole, contextual: true, forward: true, groups: []ribbonGroup{
		{title: "Move", homes: []home{big("strips.moveTo"), at("strips.up"), at("strips.down"), at("strips.unschedule"), at("strips.undo")}},
		{title: "Day", homes: []home{big("production.addDay"), at("strips.editDay"), at("strips.fold"), at("strips.unfold")}},
		{title: "Group", homes: []home{at("strips.byLocation"), at("strips.byCast")}},
		{title: "Check", homes: []home{big("strips.open"), at("strips.checked")}},
		{title: "Close", homes: []home{big("strips.close")}},
	}}
}

// stripsMenuHomes are the Production menu's Stripboard commands (there
// while the board is in front: the Menus look's way to them).
func stripsMenuHomes() []home {
	return []home{at("strips.moveTo"), at("strips.up"), at("strips.down"), at("strips.unschedule"), at("strips.undo"),
		sep(), at("strips.editDay"), at("strips.fold"), at("strips.unfold"), at("strips.byLocation"), at("strips.byCast"),
		sep(), at("strips.open"), at("strips.checked"), at("strips.close")}
}

// showMoveMenu shows Move To's choices for scene sc's strip by anchor
// (nil: by the strip).
func (sb *stripboardView) showMoveMenu(sc int, anchor fyne.CanvasObject) {
	items := sb.moveItems(sc)
	if len(items) == 0 {
		sb.w.setStatus("No day to move it to: Add Day makes one")
		return
	}
	if anchor == nil {
		if s := sb.strips[sc]; s != nil {
			anchor = s
		} else {
			anchor = sb.scroll
		}
	}
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(anchor).AddXY(0, anchor.Size().Height)
	widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", items...), sb.w.fyneWindow.Canvas(), pos)
}

// followStrips shows the Stripboard tab's buttons as the board's
// selection has them (as a strip is picked: without the whole ribbon's
// refresh).
func (r *ribbonBar) followStrips() {
	if r.selected != "Stripboard" || !r.w.ribbonShown() {
		return
	}
	r.w.stripsEnv(&r.env)
	t, _ := tabOf("Stripboard")
	for _, g := range t.groups {
		for _, h := range g.homes {
			for _, b := range r.byID[h.id] {
				b.follow(r.env)
			}
		}
	}
}
