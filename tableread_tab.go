package main

import (
	"fyne.io/fyne/v2/theme"
)

// The Table Read's commands (tableread_view.go): View > Table Read opens
// it; the Table Read tab is the ribbon's contextual tab while it is the
// middle tab in front (in the browser, on a phone), brought forward as it
// comes, as the Stripboard tab is (stripstab.go); a Table Read menu in the
// Menus look then. On a desktop it is a window of its own, with its own
// keys; the palette and : reach its commands there too, while it is open.

// tableReadEnv fills in what the Table Read's rules look at.
func (w *MainWindow) tableReadEnv(e *cmdEnv) {
	e.tableRead = w.tableReadInFront() && !w.markdown
	e.tableReadOpen = w.tableReadOpen() && !w.markdown
}

// The rules of the Table Read's commands.
func tableReadInFront(e cmdEnv) bool { return e.tableRead }
func tableReadShown(e cmdEnv) bool   { return e.tableReadOpen }

// onTableRead is a Table Read command's run: f with the view open.
func onTableRead(f func(v *tableReadView)) func(w *MainWindow) {
	return func(w *MainWindow) {
		if !w.tableReadOpen() {
			w.setStatus("The Table Read is not open: View > Table Read")
			return
		}
		f(w.tableRead)
	}
}

// trChecked is a check's state, the view open.
func trChecked(f func(v *tableReadView) bool) func(w *MainWindow) bool {
	return func(w *MainWindow) bool { return w.tableRead != nil && f(w.tableRead) }
}

func tableReadCommands() []*command {
	open := func(c *command) *command {
		c.when = screenplayOnly
		c.enabled = tableReadShown
		return c
	}
	return []*command{
		{id: "view.tableRead", ex: "table-read", label: "Table Read", tip: "The script read aloud: large type on dark, a scene at a time, " +
			"each actor's lines in a tint of their own (a window for a TV or projector; in the browser, a page to cast)",
			icon: theme.MediaPlayIcon, when: screenplayOnly, run: (*MainWindow).openTableRead},
		open(&command{id: "tableread.previous", ex: "table-read-previous", label: "Previous Scene Read", ribbonLabel: "Previous",
			tip: "The scene picked before this one (←, Page Up)", icon: theme.MediaSkipPreviousIcon,
			run: onTableRead((*tableReadView).previous)}),
		open(&command{id: "tableread.next", ex: "table-read-next", label: "Next Scene Read", ribbonLabel: "Next",
			tip: "The next scene picked (→, Page Down, Space)", icon: theme.MediaSkipNextIcon,
			run: onTableRead((*tableReadView).next)}),
		open(&command{id: "tableread.pick", ex: "table-read-pick", label: "Pick What Is Read", ribbonLabel: "Pick",
			tip:  "The panel: the scenes, a character's scenes, a tag's; narrator, storyboard, tints; who to follow",
			icon: theme.MenuIcon, run: onTableRead((*tableReadView).togglePanel),
			checked: trChecked(func(v *tableReadView) bool { return v.panelShown })}),
		open(&command{id: "tableread.narrator", ex: "table-read-narrator", label: "Narrator Mode", ribbonLabel: "Narrator",
			tip: "Headings and action read by one voice, marked NARRATOR", icon: theme.AccountIcon,
			run:     onTableRead(func(v *tableReadView) { v.setNarrator(!v.narrator) }),
			checked: trChecked(func(v *tableReadView) bool { return v.narrator })}),
		open(&command{id: "tableread.storyboard", ex: "table-read-storyboard", label: "Show Storyboard Frames", ribbonLabel: "Storyboard",
			tip: "The scene's storyboard frames between its lines", icon: theme.MediaPhotoIcon,
			run:     onTableRead(func(v *tableReadView) { v.setBoards(!v.boards) }),
			checked: trChecked(func(v *tableReadView) bool { return v.boards })}),
		open(&command{id: "tableread.beside", ex: "table-read-beside", label: "Storyboard Beside the Lines", ribbonLabel: "Beside",
			tip: "The storyboard's frames beside the lines they follow, each with its time (else between them)", icon: theme.ViewRestoreIcon,
			run:     onTableRead(func(v *tableReadView) { v.setBeside(!v.beside) }),
			checked: trChecked(func(v *tableReadView) bool { return v.boards && v.beside })}),
		open(&command{id: "tableread.tints", ex: "table-read-tints", label: "Actor Tints", ribbonLabel: "Tints",
			tip: "Each actor's lines in a colour of their own", icon: theme.ColorPaletteIcon,
			run:     onTableRead(func(v *tableReadView) { v.setTint(!v.tint) }),
			checked: trChecked(func(v *tableReadView) bool { return v.tint })}),
		open(&command{id: "tableread.fullscreen", ex: "table-read-fullscreen", label: "Table Read Full Screen", ribbonLabel: "Full Screen",
			tip: "The Table Read filling the screen (in the browser, the page: cast it with Chrome's Cast…)", icon: theme.ViewFullScreenIcon,
			run: onTableRead((*tableReadView).fullScreen)}),
		open(&command{id: "tableread.follow", ex: "table-read-follow", label: "Follow in the Table Read", ribbonLabel: "Follow",
			tip: "The Table Read goes where they are reading (the script shared: who is here)", icon: theme.VisibilityIcon,
			submenu: (*MainWindow).tableReadFollowMenu}),
		{id: "tableread.followLink", ex: "table-read-follow-link", label: "Copy the Follow Link", ribbonLabel: "Follow Link",
			tip:  "A link for a TV or another screen: its Table Read follows yours, and it edits nothing (the script is shared first)",
			icon: theme.ContentCopyIcon, when: screenplayOnly, run: (*MainWindow).copyFollowLink},
		open(&command{id: "tableread.close", ex: "table-read-close", label: "Close Table Read", ribbonLabel: "Close", icon: theme.CancelIcon,
			run: (*MainWindow).closeTableRead}),
	}
}

// tableReadTab is the Table Read tab's layout.
func tableReadTab() ribbonTab {
	return ribbonTab{title: "Table Read", when: tableReadInFront, contextual: true, forward: true, groups: []ribbonGroup{
		{title: "Scenes", homes: []home{big("tableread.previous"), big("tableread.next")}},
		{title: "Read", homes: []home{big("tableread.pick"), at("tableread.narrator"), at("tableread.storyboard"), at("tableread.beside"), at("tableread.tints")}},
		{title: "Show", homes: []home{big("tableread.fullscreen")}},
		{title: "Follow", homes: []home{at("tableread.follow"), at("tableread.followLink")}},
		{title: "Close", homes: []home{big("tableread.close")}},
	}}
}

// tableReadMenu is the Table Read menu (the Menus look's, while it is in front).
func tableReadMenu() []home {
	return []home{at("tableread.previous"), at("tableread.next"), sep(), at("tableread.pick"), at("tableread.narrator"),
		at("tableread.storyboard"), at("tableread.beside"), at("tableread.tints"), sep(), at("tableread.follow"), at("tableread.followLink"), sep(), at("tableread.fullscreen"), at("tableread.close")}
}
