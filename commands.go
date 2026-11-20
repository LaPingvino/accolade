package main

import (
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
)

// The command registry: every command the user can give is defined here
// once (its id, label, icon, shortcuts, when it applies and what it does),
// and where it lives too: the menu bar (menuBarLayout), the header bar's
// two rows (topRowLayout, screenplayRowLayout, markdownRowLayout) and the
// editor's context menu (contextLayout). menus.go, header_bar.go and the
// shortcuts read these; the ribbon and a command palette are to read them
// too (LOOP.md), so nothing is defined twice.
//
// Keyboard shortcuts live on the main menu: while the editor has focus,
// Fyne hands shortcuts to the focused widget and only main menu shortcuts
// get a look first. A command's first key is its menu item's; further
// keys (Alt+Enter for full screen, F6 for the panes) are caught before the
// focused widget sees them (keyPreview, window.go).

// cmdEnv is what a command's rules look at: the kind of document, where
// Accolade runs, and the script's state.
type cmdEnv struct {
	markdown bool // a Markdown document, not a screenplay
	browser  bool // the WebAssembly build: one window, no programs run
	pandoc   bool // pandoc is there to import more formats
	source   bool // the screenplay has a source (Adaptation)
	locked   bool // its scene numbers are locked (Production)
	pages    bool // its pages are locked (Production)
}

// cmdEnv is the window's.
func (w *MainWindow) cmdEnv() cmdEnv {
	e := cmdEnv{markdown: w.markdown, browser: inBrowser, pandoc: hasPandoc()}
	if w.textEditor != nil && !w.markdown {
		text := w.textEditor.Text()
		e.source = w.sourceRef().Origin != sourceNone
		e.locked = sceneLockOf(text).locked
		e.pages = pageLockOf(text) != nil
	}
	return e
}

// The rules for when a command applies at all.
func screenplayOnly(e cmdEnv) bool { return !e.markdown }
func markdownOnly(e cmdEnv) bool   { return e.markdown }
func desktopOnly(e cmdEnv) bool    { return !e.browser }
func withPandoc(e cmdEnv) bool     { return e.pandoc && !e.browser }

// The rules for when a command can be given now.
func withSource(e cmdEnv) bool    { return e.source }
func numbersLocked(e cmdEnv) bool { return e.locked }
func pagesLocked(e cmdEnv) bool   { return e.pages }

// command is one thing the user can do.
type command struct {
	id    string
	label string // in menus (and the palette)
	// mdLabel is the label in a Markdown document, if another
	mdLabel string
	// tip is the header bar's tooltip, without the shortcut (label if
	// ""); mdTip a Markdown document's; tipFor, if set, is the whole tip
	tip, mdTip string
	tipFor     func(cmdEnv) string
	// text is a text button's text (the editing row's), accessible
	// names saying more (tip)
	text string
	icon func() fyne.Resource
	// keys are its shortcuts: the first on its menu item, the rest
	// (keyPreview's) seen before the focused widget; shownKeys are keys
	// the tip shows that something else handles (Undo's Ctrl+Z)
	keys      []fyne.Shortcut
	shownKeys string
	// editing commands act on the script being written in: from a menu
	// (or its shortcut) only while the editor has the focus, from a
	// button with the focus given back to the editor
	editing bool
	run     func(w *MainWindow)
	// pick, if set, opens a picker next to anchor (nil from a menu)
	pick func(w *MainWindow, anchor fyne.CanvasObject)
	// tap, if set, is what its button does instead of run
	tap func(w *MainWindow)
	// when it applies: hidden from menus, its button disabled (nil: always)
	when func(cmdEnv) bool
	// enabled: whether it can be given now (nil: always). The commands
	// say why not themselves (a status line), so the menu bar does not
	// grey them; the palette and the ribbon look at it.
	enabled func(cmdEnv) bool
	checked func(w *MainWindow) bool
	// submenu, if set, is made when the menu is (Open Recent)
	submenu func(w *MainWindow) *fyne.Menu
}

// applies reports whether c is there in env.
func (c *command) applies(e cmdEnv) bool { return c.when == nil || c.when(e) }

// canRun reports whether c applies and can be given now.
func (c *command) canRun(e cmdEnv) bool {
	return c.applies(e) && (c.enabled == nil || c.enabled(e))
}

// labelIn is c's label in env.
func (c *command) labelIn(e cmdEnv) string {
	if e.markdown && c.mdLabel != "" {
		return c.mdLabel
	}
	return c.label
}

// tipIn is c's tooltip in env, with its shortcut.
func (c *command) tipIn(e cmdEnv) string {
	if c.tipFor != nil {
		return c.tipFor(e)
	}
	tip := c.tip
	if e.markdown && c.mdTip != "" {
		tip = c.mdTip
	}
	if tip == "" {
		tip = c.labelIn(e)
	}
	return tip + c.keysTip()
}

// keysTip is c's shortcut for a tooltip (" (Ctrl+S)"), "" if none.
func (c *command) keysTip() string {
	if c.shownKeys != "" {
		return keys(c.shownKeys)
	}
	if len(c.keys) == 0 {
		return ""
	}
	if k := keyText(c.keys[0]); k != "" {
		return keys(k)
	}
	return ""
}

// keyText is a Ctrl (Cmd) shortcut as keys takes it: "Shift+P"; "" for
// one without Ctrl.
func keyText(s fyne.Shortcut) string {
	ks, ok := s.(fyne.KeyboardShortcut)
	if !ok || ks.Mod()&fyne.KeyModifierShortcutDefault == 0 {
		return ""
	}
	var b strings.Builder
	if ks.Mod()&fyne.KeyModifierShift != 0 {
		b.WriteString("Shift+")
	}
	if ks.Mod()&fyne.KeyModifierAlt != 0 {
		b.WriteString("Alt+")
	}
	b.WriteString(string(ks.Key()))
	return b.String()
}

// do is c as a menu item or shortcut does it.
func (w *MainWindow) do(c *command) func() {
	if c.pick != nil {
		return func() { c.pick(w, nil) }
	}
	if !c.editing {
		return func() { c.run(w) }
	}
	return func() {
		if f := w.fyneWindow.Canvas().Focused(); f != nil && f != fyne.Focusable(w.textEditor) {
			return // the focus in the notes or search
		}
		c.run(w)
	}
}

// tapped is c as a button does it (anchor: the button).
func (w *MainWindow) tapped(c *command, anchor fyne.CanvasObject) func() {
	switch {
	case c.pick != nil:
		return func() { c.pick(w, anchor) }
	case c.tap != nil:
		return func() { c.tap(w) }
	case c.editing:
		return func() {
			c.run(w)
			w.fyneWindow.Canvas().Focus(w.textEditor)
		}
	}
	return func() { c.run(w) }
}

// home is a command's place in a menu or row: its id, or a separator
// (a gap on a row); label is this place's own, if not the command's;
// items, if set, are made when the menu is (none if they apply not).
type home struct {
	id    string
	label string
	sep   bool
	items func(w *MainWindow) []*fyne.MenuItem
}

func at(id string) home                              { return home{id: id} }
func as(id, label string) home                       { return home{id: id, label: label} }
func sep() home                                      { return home{sep: true} }
func made(f func(*MainWindow) []*fyne.MenuItem) home { return home{items: f} }

// menuDef is a menu of the menu bar.
type menuDef struct {
	title string
	when  func(cmdEnv) bool
	homes []home
}

// registry is the commands by id and their homes.
type registry struct {
	commands []*command // in the order defined
	byID     map[string]*command
	menuBar  []menuDef
	// the header bar's rows: the top row, and the editing row of a
	// screenplay and of a Markdown document
	topRow, screenplayRow, markdownRow []home
	context                            []home // the editor's right-click menu
	previewKeys                        []previewKey
}

var (
	regOnce sync.Once
	reg     *registry
)

// commands is the registry, made the first time it is asked for.
func commands() *registry {
	regOnce.Do(func() { reg = newRegistry() })
	return reg
}

// cmd is the command with id; it panics for one not defined (a typo in a
// layout is found by the first test that builds the menu).
func cmd(id string) *command {
	c, ok := commands().byID[id]
	if !ok {
		panic("no command " + id)
	}
	return c
}

func altKey(k fyne.KeyName) fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: k, Modifier: fyne.KeyModifierAlt}
}

func plainKey(k fyne.KeyName, mod fyne.KeyModifier) fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: k, Modifier: mod}
}

// elementCommandID is the id of the command making a line element el.
func elementCommandID(el string) string { return "format.element." + el }

func newRegistry() *registry {
	r := &registry{byID: map[string]*command{}}
	add := func(cs ...*command) {
		for _, c := range cs {
			if _, dup := r.byID[c.id]; dup {
				panic("command defined twice: " + c.id)
			}
			r.byID[c.id] = c
			r.commands = append(r.commands, c)
		}
	}

	// File
	add(
		&command{id: "file.new", label: "New", keys: []fyne.Shortcut{shortcut(fyne.KeyN, 0)}, icon: theme.DocumentCreateIcon,
			run: (*MainWindow).NewFile,
			tipFor: func(e cmdEnv) string {
				tip := "New" + keys("N") + "; with Alt, empty" + keys("Alt+N")
				if !e.browser {
					tip += "; with Shift, a new window" + keys("Shift+N")
				}
				return tip
			},
			tap: func(w *MainWindow) { // with Alt empty, with Shift in a window of its own
				mods := heldModifiers()
				empty := mods&fyne.KeyModifierAlt != 0
				switch {
				case mods&fyne.KeyModifierShift != 0 && !inBrowser:
					w.app.newScriptWindow(empty)
				case empty:
					w.NewEmptyFile()
				default:
					w.NewFile()
				}
			}},
		&command{id: "file.newEmpty", label: "New Empty", keys: []fyne.Shortcut{shortcut(fyne.KeyN, fyne.KeyModifierAlt)}, run: (*MainWindow).NewEmptyFile},
		&command{id: "file.newWindow", label: "New Window", keys: []fyne.Shortcut{shortcut(fyne.KeyN, fyne.KeyModifierShift)}, when: desktopOnly, // the browser has one window
			run: func(w *MainWindow) { w.app.newScriptWindow(false) }},
		&command{id: "file.newEmptyWindow", label: "New Empty Window", keys: []fyne.Shortcut{shortcut(fyne.KeyN, fyne.KeyModifierShift|fyne.KeyModifierAlt)}, when: desktopOnly,
			run: func(w *MainWindow) { w.app.newScriptWindow(true) }},
		&command{id: "file.newMarkdown", label: "New Markdown Document", run: (*MainWindow).NewMarkdownFile},
		&command{id: "file.open", label: openLabel, tip: "Open", keys: []fyne.Shortcut{shortcut(fyne.KeyO, 0)}, icon: theme.FolderOpenIcon, run: (*MainWindow).OpenFile},
		&command{id: "file.recent", label: "Open Recent", submenu: func(w *MainWindow) *fyne.Menu { return w.recentMenu().ChildMenu }},
		&command{id: "file.import", label: "Import as Markdown…", run: (*MainWindow).ImportDocument},
		&command{id: "file.importPandoc", label: "Import with pandoc…", when: withPandoc, run: (*MainWindow).ImportWithPandoc}, // more formats
		// a Markdown document is a source to start from
		&command{id: "file.startScreenplay", label: startScreenplayLabel, text: "Start a Screenplay", icon: theme.DocumentCreateIcon,
			when: markdownOnly, run: (*MainWindow).startScreenplayFromSource},
		&command{id: "file.save", label: saveLabel, tip: "Save", keys: []fyne.Shortcut{shortcut(fyne.KeyS, 0)}, icon: theme.DocumentSaveIcon,
			run: func(w *MainWindow) { w.reportErr(w.SaveFile)() }},
		&command{id: "file.saveAs", label: "Save As…", keys: []fyne.Shortcut{shortcut(fyne.KeyS, fyne.KeyModifierShift)},
			run: func(w *MainWindow) { w.reportErr(w.SaveFileAs)() }},
		&command{id: "file.saveDraft", label: "Save Draft…", run: (*MainWindow).showSaveDraft},
		&command{id: "file.drafts", label: "Drafts…", run: (*MainWindow).showDrafts},
		&command{id: "file.share", label: "Edit Together…", run: (*MainWindow).shareDialog},
		&command{id: "file.join", label: "Join a Shared Script…", run: (*MainWindow).joinDialog},
		&command{id: "file.titlePage", label: "Title Page…", keys: []fyne.Shortcut{shortcut(fyne.KeyT, fyne.KeyModifierShift)}, when: screenplayOnly,
			run: (*MainWindow).showTitlePageDialog},
		&command{id: "file.export", label: "Export…", tip: "Export to PDF, HTML, FDX…", mdTip: "Export to Word, ODT, HTML, PDF…",
			keys: []fyne.Shortcut{shortcut(fyne.KeyE, 0)}, icon: theme.DownloadIcon, run: (*MainWindow).showExportDialog},
		&command{id: "file.print", label: "Print…", keys: []fyne.Shortcut{shortcut(fyne.KeyP, 0)}, run: (*MainWindow).print},
		&command{id: "app.preferences", label: "Preferences…", tip: "Preferences", keys: []fyne.Shortcut{shortcut(fyne.KeyComma, 0)}, icon: theme.SettingsIcon,
			run: func(w *MainWindow) { w.app.showPreferences() }},
	)

	// Edit: Ctrl+Z / Ctrl+Y reach the focused text field directly; the
	// menu adds the Ctrl+Shift+Z redo that Fyne does not map itself
	clip := func(s func(c fyne.Clipboard) fyne.Shortcut) func(w *MainWindow) {
		return func(w *MainWindow) { w.textEditor.TypedShortcut(s(w.clipboard())) }
	}
	add(
		&command{id: "edit.undo", label: "Undo", shownKeys: "Z", icon: theme.ContentUndoIcon, run: (*MainWindow).undo},
		&command{id: "edit.redo", label: "Redo", keys: []fyne.Shortcut{shortcut(fyne.KeyZ, fyne.KeyModifierShift)}, icon: theme.ContentRedoIcon, run: (*MainWindow).redo},
		&command{id: "edit.find", label: "Find…", keys: []fyne.Shortcut{shortcut(fyne.KeyF, 0)}, run: (*MainWindow).showFind},
		&command{id: "edit.replace", label: "Replace…", keys: []fyne.Shortcut{shortcut(fyne.KeyH, 0)}, run: (*MainWindow).showFindReplace},
		// the header bar's search: the bar with its replace row
		&command{id: "edit.findReplace", label: "Find and replace", shownKeys: "F", icon: theme.SearchIcon, run: (*MainWindow).showFindReplace},
		&command{id: "edit.findNext", label: "Find Next", keys: []fyne.Shortcut{shortcut(fyne.KeyG, 0)}, run: (*MainWindow).findNext},
		&command{id: "edit.findPrevious", label: "Find Previous", keys: []fyne.Shortcut{shortcut(fyne.KeyG, fyne.KeyModifierShift)}, run: (*MainWindow).findPrevious},
		&command{id: "edit.cut", label: "Cut", run: clip(func(c fyne.Clipboard) fyne.Shortcut { return &fyne.ShortcutCut{Clipboard: c} })},
		&command{id: "edit.copy", label: "Copy", run: clip(func(c fyne.Clipboard) fyne.Shortcut { return &fyne.ShortcutCopy{Clipboard: c} })},
		&command{id: "edit.paste", label: "Paste", run: clip(func(c fyne.Clipboard) fyne.Shortcut { return &fyne.ShortcutPaste{Clipboard: c} })},
	)

	// Format: the line's element (Ctrl+1 to Ctrl+8), its direction and
	// language, emphasis, moving lines
	for i, el := range toolbarElements {
		add(&command{id: elementCommandID(el), label: el, text: elementMarks[el], keys: []fyne.Shortcut{shortcut(fyne.KeyName(rune('1'+i)), 0)},
			editing: true, when: screenplayOnly,
			run: func(w *MainWindow) { setLineElement(w.textEditor, el, w.sceneStarts()) }})
	}
	emphasis := func(marker string) func(w *MainWindow) {
		return func(w *MainWindow) { toggleEmphasis(w.textEditor, marker) }
	}
	moveLine := func(dir int) func(w *MainWindow) {
		return func(w *MainWindow) { // a strip with the focus moves instead (stripboard.go)
			w.orStrip(dir, w.do(&command{editing: true, run: func(w *MainWindow) { moveLines(w.textEditor, dir) }}))()
		}
	}
	add(
		&command{id: "format.rtl", label: "Right to Left Line", editing: true, run: func(w *MainWindow) { toggleDirection(w.textEditor) }},
		&command{id: "format.language", label: "Script Language…", when: screenplayOnly, run: (*MainWindow).pickLanguage},
		&command{id: "format.color", label: "Scene or Section Color…", text: "Color", tip: "Color of the scene or section", icon: theme.ColorPaletteIcon,
			keys: []fyne.Shortcut{shortcut(fyne.KeyL, fyne.KeyModifierShift)}, when: screenplayOnly, pick: (*MainWindow).pickColorHere},
		&command{id: "format.bold", label: "Bold", keys: []fyne.Shortcut{shortcut(fyne.KeyB, 0)}, icon: func() fyne.Resource { return boldIcon }, editing: true, run: emphasis("**")},
		&command{id: "format.italic", label: "Italic", keys: []fyne.Shortcut{shortcut(fyne.KeyI, 0)}, icon: func() fyne.Resource { return italicIcon }, editing: true, run: emphasis("*")},
		// Markdown's _ is italic, not underline
		&command{id: "format.underline", label: "Underline", keys: []fyne.Shortcut{shortcut(fyne.KeyU, 0)}, icon: func() fyne.Resource { return underlineIcon }, editing: true,
			when: screenplayOnly, run: emphasis("_")},
		&command{id: "format.lineUp", label: "Move Line Up", keys: []fyne.Shortcut{altKey(fyne.KeyUp)}, run: moveLine(-1)},
		&command{id: "format.lineDown", label: "Move Line Down", keys: []fyne.Shortcut{altKey(fyne.KeyDown)}, run: moveLine(1)},
		// the editing row's pickers and its switch
		&command{id: "write.place", label: "Inside or Outside…", text: "INT./EXT.", tip: "INT./EXT.", icon: theme.HomeIcon, when: screenplayOnly, pick: (*MainWindow).pickPlace},
		&command{id: "write.character", label: "Character Name…", text: "Character", tip: "Character", icon: theme.AccountIcon, when: screenplayOnly, pick: (*MainWindow).pickCharacter},
		&command{id: "write.elementKind", label: "Elements as a list or as buttons", icon: theme.ViewRestoreIcon, when: screenplayOnly,
			run: func(w *MainWindow) {
				w.settings.SetBoolean("element-buttons", !w.settings.GetBoolean("element-buttons"))
				_ = w.settings.Save()
				if w.headerBar != nil {
					w.headerBar.showElementKind()
				}
			}},
	)

	// Adaptation: the source a screenplay adapts (docs/ADAPTATION.md)
	add(
		&command{id: "source.choose", label: "Choose Source…", when: screenplayOnly, run: (*MainWindow).chooseSource},
		&command{id: "source.saveAs", label: "Save Source As…", when: screenplayOnly, enabled: withSource, run: (*MainWindow).saveSourceAs},
		&command{id: "source.keepInside", label: "Keep Source in the Script", when: screenplayOnly, enabled: withSource, run: (*MainWindow).toggleSourceInside,
			checked: func(w *MainWindow) bool { return w.textEditor != nil && w.sourceRef().Origin == sourceInside }},
		&command{id: "source.link", label: "Link to Source Passage", when: screenplayOnly, enabled: withSource, run: (*MainWindow).linkToSource},
		&command{id: "source.next", label: "Next Source Passage", keys: []fyne.Shortcut{shortcut(fyne.KeyJ, fyne.KeyModifierAlt)}, when: screenplayOnly, enabled: withSource,
			run: (*MainWindow).nextSourcePassage},
		&command{id: "source.cut", label: "Mark Source Passage as Cut", when: screenplayOnly, enabled: withSource, run: (*MainWindow).toggleCutHere},
		&command{id: "source.suggest", label: "Suggest Links to the Source", when: screenplayOnly, enabled: withSource, run: (*MainWindow).suggestSourceLinks},
		&command{id: "source.update", label: "Update Source Links", when: screenplayOnly, enabled: withSource, run: (*MainWindow).updateSourceLinks},
		&command{id: "source.sections", label: "Source Headings as Sections", when: screenplayOnly, enabled: withSource, run: (*MainWindow).sectionsFromSourceHere},
		&command{id: "source.learned", label: "What Accolade Learned…", when: screenplayOnly, enabled: withSource, run: (*MainWindow).showLearned},
	)

	// Production: a shooting script's (docs/PRODUCTION.md; production.go)
	prod := func(id, label string, run func(*MainWindow)) *command {
		return &command{id: id, label: label, when: screenplayOnly, run: run}
	}
	unlock := prod("production.unlockNumbers", "Unlock Scene Numbers…", (*MainWindow).unlockSceneNumbers)
	unlock.enabled = numbersLocked
	unlockPages := prod("production.unlockPages", "Unlock Pages…", (*MainWindow).unlockPages)
	unlockPages.enabled = pagesLocked
	tag := prod("production.tag", "Tag Selection…", (*MainWindow).showTagPicker)
	tag.keys = []fyne.Shortcut{shortcut(fyne.KeyB, fyne.KeyModifierShift)}
	showTags := prod("production.showTags", "Show Tagged Words", (*MainWindow).toggleBreakdownTags)
	showTags.checked = func(w *MainWindow) bool { return w.settings == nil || !w.settings.GetBoolean(breakdownHiddenPref) }
	add(
		prod("production.lockNumbers", "Lock Scene Numbers", (*MainWindow).lockSceneNumbers),
		unlock,
		prod("production.renumber", "Renumber Scenes", (*MainWindow).renumberScenes),
		prod("production.omit", "Omit Scene", (*MainWindow).omitScene),
		prod("production.restore", "Restore Scene", (*MainWindow).restoreScene),
		prod("production.issue", "Issue Draft…", (*MainWindow).showIssueDraft),
		prod("production.lockPages", "Lock Pages", (*MainWindow).lockPages),
		unlockPages,
		tag,
		prod("production.untag", "Untag", (*MainWindow).untagSelection),
		showTags,
		prod("production.breakdown", "Break Down…", (*MainWindow).showBreakdownStage),
		prod("production.stripboard", "Stripboard", (*MainWindow).toggleStripboard),
		prod("production.schedule", "Schedule…", (*MainWindow).showScheduleStage),
		prod("production.ics", "Export Shoot Days (.ics)…", (*MainWindow).exportICS),
		prod("production.reports", "Reports…", (*MainWindow).showReports),
	)

	// View
	add(
		&command{id: "view.preview", label: "Preview PDF", mdLabel: "Preview", keys: []fyne.Shortcut{shortcut(fyne.KeyP, fyne.KeyModifierShift)}, icon: theme.VisibilityIcon,
			run: (*MainWindow).preview},
		&command{id: "view.outline", label: "Outline", keys: []fyne.Shortcut{shortcut(fyne.KeyO, fyne.KeyModifierShift)}, icon: theme.ListIcon, run: (*MainWindow).toggleOutline},
		&command{id: "view.cards", label: "Cards", keys: []fyne.Shortcut{shortcut(fyne.KeyD, fyne.KeyModifierShift)}, icon: theme.GridIcon, when: screenplayOnly,
			run: (*MainWindow).toggleCards},
		&command{id: "view.foldHere", label: "Fold or Unfold Here", keys: []fyne.Shortcut{shortcut(fyne.KeyPeriod, fyne.KeyModifierShift)},
			run: func(w *MainWindow) { w.fold(false, true) }},
		&command{id: "view.foldSections", label: "Fold All (Sections Only)", run: func(w *MainWindow) { w.foldOverview(w.outlineItems(w.textEditor.Text())) }},
		&command{id: "view.headingsOnly", label: "Headings Only", run: (*MainWindow).foldHeadings},
		&command{id: "view.unfoldAll", label: "Unfold All", run: func(w *MainWindow) { w.foldAll(true) }},
		&command{id: "view.notes", label: "Notes", keys: []fyne.Shortcut{shortcut(fyne.KeyK, fyne.KeyModifierShift)}, icon: theme.FileTextIcon, run: (*MainWindow).toggleNotes},
		&command{id: "view.source", label: "Source", tip: "Source: the story the script adapts", icon: func() fyne.Resource { return sourceIcon },
			when: screenplayOnly, run: (*MainWindow).toggleSource},
		&command{id: "view.production", label: "Production", tip: "Production: lock, revise, and the stages to the shoot", icon: func() fyne.Resource { return productionIcon },
			when: screenplayOnly, run: (*MainWindow).toggleProduction},
		&command{id: "view.changes", label: "Changes", keys: []fyne.Shortcut{shortcut(fyne.KeyH, fyne.KeyModifierShift)}, run: (*MainWindow).toggleChanges},
		&command{id: "view.statistics", label: "Statistics…", when: screenplayOnly, run: (*MainWindow).showStatistics},
		// Alt+Enter too, as in many programs
		&command{id: "view.fullscreen", label: "Fullscreen", tip: "Full screen, without distractions", icon: theme.ViewFullScreenIcon,
			keys: []fyne.Shortcut{shortcut(fyne.KeyF, fyne.KeyModifierShift), plainKey(fyne.KeyReturn, fyne.KeyModifierAlt), plainKey(fyne.KeyEnter, fyne.KeyModifierAlt)},
			run:  (*MainWindow).toggleFullscreen},
		&command{id: "view.documentType", label: "Document Type", submenu: func(w *MainWindow) *fyne.Menu { return w.documentTypeMenu().ChildMenu }},
		// F6 and Shift+F6: the next or previous pane (editor, outline,
		// notes, search), whatever has the focus
		&command{id: "view.nextPane", label: "Next Pane", keys: []fyne.Shortcut{plainKey(fyne.KeyF6, 0)}, run: func(w *MainWindow) { w.cyclePane(1) }},
		&command{id: "view.previousPane", label: "Previous Pane", keys: []fyne.Shortcut{plainKey(fyne.KeyF6, fyne.KeyModifierShift)}, run: func(w *MainWindow) { w.cyclePane(-1) }},
	)

	add(&command{id: "help.about", label: "About Accolade", run: func(w *MainWindow) { w.app.showAbout() }})

	elements := make([]home, len(toolbarElements))
	for i, el := range toolbarElements {
		elements[i] = at(elementCommandID(el))
	}
	r.menuBar = []menuDef{
		{title: "File", homes: []home{
			at("file.new"), at("file.newEmpty"), at("file.newWindow"), at("file.newEmptyWindow"), at("file.newMarkdown"),
			at("file.open"), at("file.recent"), at("file.import"), at("file.importPandoc"), at("file.startScreenplay"),
			sep(), at("file.save"), at("file.saveAs"), at("file.saveDraft"), at("file.drafts"),
			sep(), at("file.share"), at("file.join"),
			sep(), at("file.titlePage"), at("file.export"), at("file.print"),
			sep(), at("app.preferences"),
			// Fyne appends Quit (Ctrl+Q) to the first menu itself.
		}},
		{title: "Edit", homes: []home{
			at("edit.undo"), at("edit.redo"),
			sep(), at("edit.find"), at("edit.replace"), at("edit.findNext"), at("edit.findPrevious"),
		}},
		{title: "Format", homes: append(elements,
			at("format.rtl"), at("format.language"), at("format.color"),
			sep(), at("format.bold"), at("format.italic"), at("format.underline"),
			sep(), at("format.lineUp"), at("format.lineDown"),
		)},
		{title: "Adaptation", when: screenplayOnly, homes: []home{
			at("source.choose"), at("source.saveAs"), at("source.keepInside"), as("view.source", "Source Pane"),
			sep(), at("source.link"), at("source.next"), at("source.cut"), at("source.suggest"), at("source.update"), at("source.sections"),
			sep(), at("source.learned"),
		}},
		{title: "Production", when: screenplayOnly, homes: []home{
			at("production.lockNumbers"), at("production.unlockNumbers"), at("production.renumber"),
			sep(), at("production.omit"), at("production.restore"),
			sep(), at("production.issue"), at("production.lockPages"), at("production.unlockPages"),
			sep(), at("production.tag"), at("production.untag"), at("production.showTags"), at("production.breakdown"),
			at("production.stripboard"), at("production.schedule"), at("production.ics"),
			sep(), at("production.reports"),
		}},
		{title: "View", homes: []home{
			at("view.preview"), at("view.outline"), at("view.cards"), at("view.foldHere"), at("view.foldSections"),
			at("view.headingsOnly"), at("view.unfoldAll"), at("view.notes"), at("view.source"), at("view.production"),
			at("view.changes"), at("view.statistics"), at("view.fullscreen"),
			sep(), at("view.documentType"),
		}},
		{title: "Help", homes: []home{at("help.about")}},
	}
	r.topRow = []home{
		at("file.new"), at("file.open"), at("file.save"), sep(),
		at("edit.undo"), at("edit.redo"),
		at("view.outline"), at("view.cards"), at("view.notes"), at("view.source"), at("view.production"), sep(),
		at("edit.findReplace"), at("view.preview"), at("file.export"), sep(),
		at("view.fullscreen"), at("app.preferences"),
	}
	// the element list (or buttons) after the switch, the format's own
	// buttons (format_buttons.go) at the end
	r.screenplayRow = []home{
		at("write.elementKind"), {id: "elements"}, at("write.place"), at("write.character"), at("format.color"),
		at("format.bold"), at("format.italic"), at("format.underline"), sep(), {id: "formatButtons"},
	}
	// a story's next step: a screenplay adapting it
	r.markdownRow = []home{at("format.bold"), at("format.italic"), sep(), at("file.startScreenplay")}
	r.context = []home{
		made((*MainWindow).tagAsLastItems), at("production.tag"), made((*MainWindow).untagHereItems),
		sep(), at("edit.cut"), at("edit.copy"), at("edit.paste"),
	}
	r.previewKeys = previewKeysOf(r)
	return r
}

// menuItems are the menu items of homes in env: separators only between
// items (none doubled, none at an end).
func (w *MainWindow) menuItems(homes []home, env cmdEnv) []*fyne.MenuItem {
	var items []*fyne.MenuItem
	pendingSep := false
	for _, h := range homes {
		var got []*fyne.MenuItem
		switch {
		case h.sep:
			pendingSep = len(items) > 0
			continue
		case h.items != nil:
			got = h.items(w)
		default:
			c := cmd(h.id)
			if !c.applies(env) {
				continue
			}
			got = []*fyne.MenuItem{w.menuItem(c, h.label, env)}
		}
		if len(got) == 0 {
			continue
		}
		if pendingSep {
			items = append(items, fyne.NewMenuItemSeparator())
			pendingSep = false
		}
		items = append(items, got...)
	}
	return items
}

// menuItem is c as a menu item (label: its home's own, if any).
func (w *MainWindow) menuItem(c *command, label string, env cmdEnv) *fyne.MenuItem {
	if label == "" {
		label = c.labelIn(env)
	}
	if c.submenu != nil {
		item := fyne.NewMenuItem(label, nil)
		item.ChildMenu = c.submenu(w)
		return item
	}
	item := fyne.NewMenuItem(label, w.do(c))
	if len(c.keys) > 0 {
		item.Shortcut = c.keys[0]
	}
	if c.checked != nil {
		item.Checked = c.checked(w)
	}
	return item
}

// menuBar is the menu bar in env.
func (w *MainWindow) menuBar(env cmdEnv) *fyne.MainMenu {
	var menus []*fyne.Menu
	for _, m := range commands().menuBar {
		if m.when != nil && !m.when(env) {
			continue
		}
		menus = append(menus, fyne.NewMenu(m.title, w.menuItems(m.homes, env)...))
	}
	return fyne.NewMainMenu(menus...)
}

// previewKey runs the command whose further key (not its menu item's) is
// name with mod, if one applies; it reports whether one did.
func (w *MainWindow) previewKey(name fyne.KeyName, mod fyne.KeyModifier) bool {
	for _, pk := range commands().previewKeys {
		if pk.key.Key() == name && pk.key.Mod() == mod && pk.c.applies(cmdEnv{markdown: w.markdown, browser: inBrowser}) {
			w.do(pk.c)()
			return true
		}
	}
	return false
}

// previewKeysOf are the keys of commands keyPreview sees: all but the
// first of those in the menu bar (the menu item has that), all of the
// others'.
func previewKeysOf(r *registry) []previewKey {
	inMenu := map[string]bool{}
	for _, m := range r.menuBar {
		for _, h := range m.homes {
			inMenu[h.id] = true
		}
	}
	var out []previewKey
	for _, c := range r.commands {
		ks := c.keys
		if inMenu[c.id] && len(ks) > 0 {
			ks = ks[1:]
		}
		for _, k := range ks {
			if kk, ok := k.(fyne.KeyboardShortcut); ok {
				out = append(out, previewKey{kk, c})
			}
		}
	}
	return out
}

// previewKey is a key keyPreview gives a command.
type previewKey struct {
	key fyne.KeyboardShortcut
	c   *command
}

// homesOf are the places c is in the menu bar, as "Format > Bold" (for
// the palette).
func homesOf(id string) []string {
	var out []string
	for _, m := range commands().menuBar {
		for _, h := range m.homes {
			if h.id == id {
				label := h.label
				if label == "" {
					label = cmd(id).label
				}
				out = append(out, m.title+" > "+label)
			}
		}
	}
	return out
}
