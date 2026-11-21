package main

import (
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"

	lexreport "github.com/LaPingvino/lexington/report"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
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
	renamed  bool // the last rename can be taken back (relocate.go)
	brought  bool // the script brought Lexington settings of its own (lexington_carried.go)
	issued   bool // the script has an issue (revisions.go)
	// writer: production is put away (role.go); the zero value shows
	// everything, so an env made for a document's kind alone hides nothing
	writer bool
	// fontSize: the editor's text size (Larger and Smaller Text)
	fontSize int
	// away: the script is not in the main editor, a pane is swapped in
	// (swap.go); swappedDoc: what is there is a Markdown document of its
	// own (the source, the notes). markdown stays the file's kind: the
	// menus that act on the document (Adaptation, Production, Export)
	// are the script's while it is away
	away, swappedDoc bool
}

// cmdEnv is the window's.
func (w *MainWindow) cmdEnv() cmdEnv {
	e := cmdEnv{markdown: w.markdown, browser: inBrowser, pandoc: hasPandoc()}
	if w.textEditor != nil && !w.markdown {
		text := w.textEditor.Text()
		e.source = w.sourceRef().Origin != sourceNone
		e.locked = sceneLockOf(text).locked
		e.pages = pageLockOf(text) != nil
		e.issued = w.scriptIssued()
	}
	e.writer = w.writerView()
	e.away = w.swapped != ""
	e.swappedDoc = e.away && w.mainDoc != nil
	e.renamed = w.canUndoRename()
	e.brought = !w.carried.empty()
	if w.settings != nil {
		e.fontSize = w.settings.GetInt("font-size")
	}
	return e
}

// The rules for when a command applies at all.
func screenplayOnly(e cmdEnv) bool { return !e.markdown }
func markdownOnly(e cmdEnv) bool   { return e.markdown }

// The rules for the commands that write in the main editor: a
// screenplay's elements in the script, Markdown's in a Markdown document
// (the file, or the source or notes swapped in).
func scriptEditing(e cmdEnv) bool   { return !e.markdown && !e.away }
func markdownEditing(e cmdEnv) bool { return e.markdown || e.swappedDoc }
func desktopOnly(e cmdEnv) bool     { return !e.browser }
func withPandoc(e cmdEnv) bool      { return e.pandoc && !e.browser }

// The rules for when a command can be given now.
func withSource(e cmdEnv) bool      { return e.source }
func numbersLocked(e cmdEnv) bool   { return e.locked }
func pagesLocked(e cmdEnv) bool     { return e.pages }
func renameToUndo(e cmdEnv) bool    { return e.renamed }
func broughtSettings(e cmdEnv) bool { return e.brought }
func hasIssue(e cmdEnv) bool        { return e.issued }

// command is one thing the user can do.
type command struct {
	id    string
	label string // in menus (and the palette)
	// mdLabel is the label in a Markdown document, if another
	mdLabel string
	// byEditor: mdLabel and mdTip follow what the main editor holds (the
	// source swapped in is Markdown), not the file's kind
	byEditor bool
	// ribbonLabel is its ribbon button's short label, if another: its
	// group says the rest ("Next" in Adaptation > Links); the full label
	// stays its tooltip and screen-reader name, menus and the palette
	// keep the full one
	ribbonLabel string
	// ribbonLabelFor, if set, is its ribbon label as the window has it
	// now ("New in: English": a setting's value)
	ribbonLabelFor func(w *MainWindow) string
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
	// role: who it is for (role.go); production's are put away in Writer,
	// from menus, the ribbon, the header and their keys alike (the
	// palette still finds them: palette.go)
	role cmdRole
}

// applies reports whether c is there in env: its rule, and its role.
func (c *command) applies(e cmdEnv) bool {
	return (c.when == nil || c.when(e)) && c.role.in(e)
}

// appliesForRole reports whether c is there in env, whatever the role
// (the palette's: Production's commands found in Writer too).
func (c *command) appliesForRole(e cmdEnv) bool {
	e.writer = false
	return c.applies(e)
}

// canRun reports whether c applies and can be given now.
func (c *command) canRun(e cmdEnv) bool {
	return c.applies(e) && (c.enabled == nil || c.enabled(e))
}

// labelIn is c's label in env.
func (c *command) labelIn(e cmdEnv) string {
	if c.mdIn(e) && c.mdLabel != "" {
		return c.mdLabel
	}
	return c.label
}

// mdIn reports whether c is labelled as for Markdown in env.
func (c *command) mdIn(e cmdEnv) bool {
	if c.byEditor {
		return markdownEditing(e)
	}
	return e.markdown
}

// ribbonLabelIn is c's label on a ribbon button in env: its short one,
// else its text button's, else its label (without the "…": the tooltip
// says it asks).
func (c *command) ribbonLabelIn(e cmdEnv) string {
	l := c.ribbonLabel
	if l == "" {
		l = c.text
	}
	if l == "" {
		l = c.labelIn(e)
	}
	return strings.TrimSuffix(l, "…")
}

// tipIn is c's tooltip in env, with its shortcut.
func (c *command) tipIn(e cmdEnv) string {
	if c.tipFor != nil {
		return c.tipFor(e)
	}
	tip := c.tip
	if c.mdIn(e) && c.mdTip != "" {
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
		e := w.mainEditor()
		if e == nil {
			w.setStatus("Nothing to write in here: Swap with Main puts the script back in the main area")
			return
		}
		if f := w.fyneWindow.Canvas().Focused(); f != nil && f != fyne.Focusable(e) {
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
			if w.mainEditor() == nil {
				w.setStatus("Nothing to write in here: Swap with Main puts the script back in the main area")
				return
			}
			c.run(w)
			w.focusMain()
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
	large bool // on the ribbon: a large button (a common action)
	// compact: on the ribbon an icon alone, beside the other compact ones
	// in one row (B I U: their tooltips and names say the rest)
	compact bool
	// keys: on the ribbon its key shows after its label (a heading's
	// Ctrl+1: given by its key all the time); not on a phone, which has
	// phoneOnly's instead (no keyboard: one picker, not a row of seven)
	keys, phoneOnly bool
	// short: its ribbon label here, if shorter than its ribbon label
	// elsewhere (the big Source's "Choose" beside it, "Choose Source" on
	// the File tab); its name and tooltip stay the full label
	short string
	role  cmdRole // made items' role (a command's is its own)
}

func at(id string) home                              { return home{id: id} }
func as(id, label string) home                       { return home{id: id, label: label} }
func sep() home                                      { return home{sep: true} }
func made(f func(*MainWindow) []*fyne.MenuItem) home { return home{items: f} }
func big(id string) home                             { return home{id: id, large: true} }
func tool(id string) home                            { return home{id: id, compact: true} }
func madeFor(r cmdRole, f func(*MainWindow) []*fyne.MenuItem) home {
	return home{items: f, role: r}
}
func bigAs(id, label string) home   { return home{id: id, label: label, large: true} }
func keyed(id string) home          { return home{id: id, keys: true} }
func atShort(id, short string) home { return home{id: id, short: short} }

// menuDef is a menu of the menu bar.
type menuDef struct {
	title string
	when  func(cmdEnv) bool
	role  cmdRole
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
	ribbon                             []ribbonTab
	ribbonEnd                          []home // the tab row's end: the palette, ☰
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

// res is a resource as a command's icon.
func res(r fyne.Resource) func() fyne.Resource { return func() fyne.Resource { return r } }

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
		&command{id: "file.new", label: "New", mdLabel: "New Screenplay", keys: []fyne.Shortcut{shortcut(fyne.KeyN, 0)}, icon: theme.DocumentCreateIcon,
			run: (*MainWindow).NewFile,
			tipFor: func(e cmdEnv) string {
				tip := "New" + keys("N") + "; with Alt, empty" + keys("Alt+N")
				if e.markdown { // a screenplay, not another Markdown document
					tip = "New screenplay" + keys("N") + "; with Alt, empty" + keys("Alt+N")
				}
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
		&command{id: "file.newEmpty", label: "New Empty", mdLabel: "New Empty Screenplay", icon: theme.FileIcon, keys: []fyne.Shortcut{shortcut(fyne.KeyN, fyne.KeyModifierAlt)}, run: (*MainWindow).NewEmptyFile},
		&command{id: "file.newWindow", label: "New Window", icon: theme.ComputerIcon, keys: []fyne.Shortcut{shortcut(fyne.KeyN, fyne.KeyModifierShift)}, when: desktopOnly, // the browser has one window
			run: func(w *MainWindow) { w.app.newScriptWindow(false) }},
		&command{id: "file.newEmptyWindow", label: "New Empty Window", icon: theme.DesktopIcon, keys: []fyne.Shortcut{shortcut(fyne.KeyN, fyne.KeyModifierShift|fyne.KeyModifierAlt)}, when: desktopOnly,
			run: func(w *MainWindow) { w.app.newScriptWindow(true) }},
		&command{id: "file.newMarkdown", label: "New Markdown Document", icon: theme.FileTextIcon, run: (*MainWindow).NewMarkdownFile},
		&command{id: "file.open", label: openLabel, tip: "Open", keys: []fyne.Shortcut{shortcut(fyne.KeyO, 0)}, icon: theme.FolderOpenIcon, run: (*MainWindow).OpenFile},
		&command{id: "file.recent", label: "Open Recent", icon: theme.HistoryIcon, submenu: func(w *MainWindow) *fyne.Menu { return w.recentMenu().ChildMenu }},
		// a script to try everything on (sample.go)
		&command{id: "file.openSample", label: "Open Sample", icon: theme.DocumentIcon, run: (*MainWindow).openSample},
		&command{id: "file.import", label: "Import as Markdown…", icon: theme.UploadIcon, run: (*MainWindow).ImportDocument},
		&command{id: "file.importPandoc", label: "Import with pandoc…", icon: theme.UploadIcon, when: withPandoc, run: (*MainWindow).ImportWithPandoc}, // more formats
		// a Markdown document is a source to start from
		&command{id: "file.startScreenplay", label: startScreenplayLabel, text: "Start a Screenplay", icon: theme.DocumentCreateIcon,
			when: markdownOnly, run: (*MainWindow).startScreenplayFromSource},
		&command{id: "file.save", label: saveLabel, tip: "Save", keys: []fyne.Shortcut{shortcut(fyne.KeyS, 0)}, icon: theme.DocumentSaveIcon,
			run: func(w *MainWindow) { w.reportErr(w.SaveFile)() }},
		&command{id: "file.saveAs", label: "Save As…", icon: theme.DocumentSaveIcon, keys: []fyne.Shortcut{shortcut(fyne.KeyS, fyne.KeyModifierShift)},
			run: func(w *MainWindow) { w.reportErr(w.SaveFileAs)() }},
		&command{id: "file.rename", label: "Rename…", icon: theme.DocumentCreateIcon, run: (*MainWindow).renameScript},
		&command{id: "file.undoRename", label: "Undo Rename", icon: theme.ContentUndoIcon, enabled: renameToUndo, run: (*MainWindow).undoRename},
		&command{id: "file.addSettings", label: "Add to My Settings…", icon: theme.SettingsIcon, when: broughtSettings,
			run: (*MainWindow).addToMySettings},
		&command{id: "file.saveDraft", label: "Save Draft…", icon: res(draftIcon), run: (*MainWindow).showSaveDraft},
		&command{id: "file.drafts", label: "Drafts…", icon: theme.HistoryIcon, run: (*MainWindow).showDrafts},
		&command{id: "file.share", label: "Edit Together…", icon: res(togetherIcon), run: (*MainWindow).shareDialog},
		&command{id: "file.join", label: "Join a Shared Script…", icon: theme.LoginIcon, run: (*MainWindow).joinDialog},
		&command{id: "file.titlePage", label: "Title Page…", icon: theme.DocumentIcon, keys: []fyne.Shortcut{shortcut(fyne.KeyT, fyne.KeyModifierShift)}, when: screenplayOnly,
			run: (*MainWindow).showTitlePageDialog},
		&command{id: "file.export", label: "Export…", tip: "Export to PDF, HTML, FDX…", mdTip: "Export to Word, ODT, HTML, PDF…",
			keys: []fyne.Shortcut{shortcut(fyne.KeyE, 0)}, icon: theme.DownloadIcon, run: (*MainWindow).showExportDialog},
		// a copy without Accolade's data, for other apps (clean_fountain.go)
		&command{id: "file.exportClean", label: "Export Clean Fountain…", tip: "A .fountain without Accolade's data, for other apps",
			icon: theme.DocumentIcon, when: screenplayOnly, run: (*MainWindow).showCleanFountain},
		pageSetupCommand(), // how it prints (pagesetup.go)
		&command{id: "file.print", label: "Print…", icon: theme.DocumentPrintIcon, keys: []fyne.Shortcut{shortcut(fyne.KeyP, 0)}, run: (*MainWindow).print},
		&command{id: "app.preferences", label: "Preferences…", tip: "Preferences", keys: []fyne.Shortcut{shortcut(fyne.KeyComma, 0)}, icon: theme.SettingsIcon,
			run: func(w *MainWindow) { w.app.showPreferences() }},
	)

	// Edit: Ctrl+Z / Ctrl+Y reach the focused text field directly; the
	// menu adds the Ctrl+Shift+Z redo that Fyne does not map itself
	// Cut, Copy, Paste and Select All act on the text field with the focus
	// (the editor, a search field), the editor if none: their keys (shown,
	// not on the menu items) stay the field's own, so a search field's
	// Ctrl+C copies its text. In the browser the clipboard is the page's
	// (navigator.clipboard, glfw-js): Paste from a menu or button asks the
	// browser to read it, which it may ask the user about or refuse (then
	// nothing is pasted); Ctrl+V is the sure way there.
	clip := func(s func(c fyne.Clipboard) fyne.Shortcut) func(w *MainWindow) {
		return func(w *MainWindow) { w.editShortcut(s(w.clipboard())) }
	}
	add(
		&command{id: "edit.undo", label: "Undo", shownKeys: "Z", icon: theme.ContentUndoIcon, run: (*MainWindow).undo},
		&command{id: "edit.redo", label: "Redo", keys: []fyne.Shortcut{shortcut(fyne.KeyZ, fyne.KeyModifierShift)}, icon: theme.ContentRedoIcon, run: (*MainWindow).redo},
		&command{id: "edit.find", label: "Find…", keys: []fyne.Shortcut{shortcut(fyne.KeyF, 0)}, icon: theme.SearchIcon, run: (*MainWindow).showFind},
		&command{id: "edit.replace", label: "Replace…", keys: []fyne.Shortcut{shortcut(fyne.KeyH, 0)}, icon: theme.SearchReplaceIcon, run: (*MainWindow).showFindReplace},
		// the header bar's search: the bar with its replace row
		&command{id: "edit.findReplace", label: "Find and replace", shownKeys: "F", icon: theme.SearchIcon, run: (*MainWindow).showFindReplace},
		&command{id: "edit.findNext", label: "Find Next", keys: []fyne.Shortcut{shortcut(fyne.KeyG, 0)}, icon: theme.NavigateNextIcon, run: (*MainWindow).findNext},
		&command{id: "edit.findPrevious", label: "Find Previous", keys: []fyne.Shortcut{shortcut(fyne.KeyG, fyne.KeyModifierShift)}, icon: theme.NavigateBackIcon, run: (*MainWindow).findPrevious},
		&command{id: "edit.cut", label: "Cut", shownKeys: "X", icon: theme.ContentCutIcon, run: clip(func(c fyne.Clipboard) fyne.Shortcut { return &fyne.ShortcutCut{Clipboard: c} })},
		&command{id: "edit.copy", label: "Copy", shownKeys: "C", icon: theme.ContentCopyIcon, run: clip(func(c fyne.Clipboard) fyne.Shortcut { return &fyne.ShortcutCopy{Clipboard: c} })},
		&command{id: "edit.paste", label: "Paste", shownKeys: "V", icon: theme.ContentPasteIcon, run: clip(func(c fyne.Clipboard) fyne.Shortcut { return &fyne.ShortcutPaste{Clipboard: c} })},
		&command{id: "edit.selectAll", label: "Select All", shownKeys: "A", icon: func() fyne.Resource { return selectAllIcon },
			run: func(w *MainWindow) { w.editShortcut(&fyne.ShortcutSelectAll{}) }},
	)

	// Format: the line's element (Ctrl+1 to Ctrl+8), its direction and
	// language, emphasis; Edit's moving lines
	for i, el := range toolbarElements {
		add(&command{id: elementCommandID(el), label: el, text: elementMarks[el], keys: []fyne.Shortcut{shortcut(fyne.KeyName(rune('1'+i)), 0)},
			editing: true, when: scriptEditing,
			run: func(w *MainWindow) { setLineElement(w.textEditor, el, w.sceneStarts()) }})
	}
	emphasis := func(marker string) func(w *MainWindow) {
		return func(w *MainWindow) { toggleEmphasis(w.writeEditor(), marker) }
	}
	moveLine := func(dir int) func(w *MainWindow) {
		return func(w *MainWindow) { // a strip with the focus moves instead (stripboard.go)
			w.orStrip(dir, w.do(&command{editing: true, run: func(w *MainWindow) { moveLines(w.writeEditor(), dir) }}))()
		}
	}
	add(
		&command{id: "format.rtl", label: "Right to Left Line", icon: func() fyne.Resource { return rtlIcon }, editing: true, run: func(w *MainWindow) { toggleDirection(w.writeEditor()) }},
		&command{id: "format.language", label: "Script Language…", icon: func() fyne.Resource { return languageIcon }, when: screenplayOnly, run: (*MainWindow).pickLanguage},
		&command{id: "format.color", label: "Scene or Section Color…", text: "Color", tip: "Color of the scene or section", icon: theme.ColorPaletteIcon,
			keys: []fyne.Shortcut{shortcut(fyne.KeyL, fyne.KeyModifierShift)}, when: screenplayOnly, pick: (*MainWindow).pickColorHere},
		&command{id: "format.bold", label: "Bold", keys: []fyne.Shortcut{shortcut(fyne.KeyB, 0)}, icon: func() fyne.Resource { return boldIcon }, editing: true, run: emphasis("**")},
		&command{id: "format.italic", label: "Italic", keys: []fyne.Shortcut{shortcut(fyne.KeyI, 0)}, icon: func() fyne.Resource { return italicIcon }, editing: true, run: emphasis("*")},
		// Markdown's _ is italic, not underline
		&command{id: "format.underline", label: "Underline", keys: []fyne.Shortcut{shortcut(fyne.KeyU, 0)}, icon: func() fyne.Resource { return underlineIcon }, editing: true,
			when: scriptEditing, run: emphasis("_")},
		&command{id: "format.lineUp", label: "Move Line Up", keys: []fyne.Shortcut{altKey(fyne.KeyUp)}, icon: theme.MoveUpIcon, run: moveLine(-1)},
		&command{id: "format.lineDown", label: "Move Line Down", keys: []fyne.Shortcut{altKey(fyne.KeyDown)}, icon: theme.MoveDownIcon, run: moveLine(1)},
		// the editing row's pickers and its switch
		&command{id: "write.place", label: "Inside or Outside…", text: "INT./EXT.", tip: "INT./EXT.", icon: theme.HomeIcon, when: scriptEditing, pick: (*MainWindow).pickPlace},
		&command{id: "write.character", label: "Character Name…", text: "Character", tip: "Character", icon: theme.AccountIcon, when: scriptEditing, pick: (*MainWindow).pickCharacter},
		&command{id: "write.elementKind", label: "Elements as a list or as buttons", icon: theme.ViewRestoreIcon, when: scriptEditing,
			run: func(w *MainWindow) {
				w.settings.SetBoolean("element-buttons", !w.settings.GetBoolean("element-buttons"))
				_ = w.settings.Save()
				if w.headerBar != nil {
					w.headerBar.showElementKind()
				}
			}},
	)

	// Markdown's Format menu (markdown_format.go): plain Markdown on the
	// lines or the selection, Ctrl+1 to Ctrl+6 the headings (a
	// screenplay's elements have those keys), Ctrl+0 normal text
	md := func(id, label string, ic fyne.Resource, run func(b *buffer.Buffer)) *command {
		return &command{id: id, label: label, icon: func() fyne.Resource { return ic }, editing: true, when: markdownEditing,
			run: func(w *MainWindow) { w.mdEdit(run) }}
	}
	for level := 0; level <= 6; level++ {
		label, text := "Normal Text", "Normal"
		if level > 0 {
			label, text = "Heading "+string(rune('0'+level)), "H"+string(rune('0'+level))
		}
		c := md("md.heading"+string(rune('0'+level)), label, headingIcon, func(b *buffer.Buffer) { setMarkdownHeading(b, level) })
		c.text = text
		c.keys = []fyne.Shortcut{shortcut(fyne.KeyName(rune('0'+level)), 0)}
		add(c)
	}
	strike := md("md.strike", "Strikethrough", strikeIcon, nil)
	strike.run = emphasis("~~")
	code := md("md.code", "Inline Code", codeIcon, nil)
	code.run = emphasis("`")
	link := md("md.link", "Link…", linkIcon, nil)
	link.run, link.keys = func(w *MainWindow) { w.askLink(false) }, []fyne.Shortcut{shortcut(fyne.KeyL, 0)}
	image := md("md.image", "Image…", imageIcon, nil)
	image.run = func(w *MainWindow) { w.askLink(true) }
	table := md("md.table", "Table…", tableIcon, nil)
	table.run = (*MainWindow).askTable
	add(
		// a picker of the headings, for the Markdown row and the ribbon
		&command{id: "md.headings", label: "Heading…", text: "Heading", tip: "Heading level, or normal text", icon: func() fyne.Resource { return headingIcon },
			when: markdownEditing, pick: func(w *MainWindow, anchor fyne.CanvasObject) { w.pickHeading(anchor) }},
		strike, code,
		md("md.codeBlock", "Code Block", codeBlockIcon, toggleCodeBlock),
		link, image,
		md("md.bullets", "Bulleted List", bulletsIcon, func(b *buffer.Buffer) { toggleMarkdownBlock(b, mdBullets) }),
		md("md.numbers", "Numbered List", numbersIcon, func(b *buffer.Buffer) { toggleMarkdownBlock(b, mdNumbers) }),
		md("md.tasks", "Task List", tasksIcon, func(b *buffer.Buffer) { toggleMarkdownBlock(b, mdTasks) }),
		md("md.quote", "Quote", quoteIcon, func(b *buffer.Buffer) { toggleMarkdownBlock(b, mdQuote) }),
		md("md.rule", "Horizontal Rule", ruleIcon, insertRule),
		table,
		md("md.footnote", "Footnote", footnoteIcon, insertFootnote),
	)

	// Adaptation: the source a screenplay adapts (docs/ADAPTATION.md)
	add(
		&command{id: "source.choose", label: "Choose Source…", icon: theme.FolderOpenIcon, when: screenplayOnly, run: (*MainWindow).chooseSource},
		&command{id: "source.saveAs", label: "Save Source As…", icon: theme.DocumentSaveIcon, when: screenplayOnly, enabled: withSource, run: (*MainWindow).saveSourceAs},
		&command{id: "source.keepInside", label: "Keep Source in the Script", icon: res(keepInsideIcon), when: screenplayOnly, enabled: withSource, run: (*MainWindow).toggleSourceInside,
			checked: func(w *MainWindow) bool { return w.textEditor != nil && w.sourceRef().Origin == sourceInside }},
		&command{id: "source.link", label: "Link to Source Passage", icon: func() fyne.Resource { return linkIcon }, when: screenplayOnly, enabled: withSource, run: (*MainWindow).linkToSource},
		&command{id: "source.next", label: "Next Source Passage", keys: []fyne.Shortcut{shortcut(fyne.KeyJ, fyne.KeyModifierAlt)}, icon: theme.NavigateNextIcon, when: screenplayOnly, enabled: withSource,
			run: (*MainWindow).nextSourcePassage},
		&command{id: "source.cut", label: "Mark Source Passage as Cut", icon: theme.ContentCutIcon, when: screenplayOnly, enabled: withSource, run: (*MainWindow).toggleCutHere},
		&command{id: "source.suggest", label: "Suggest Links to the Source", icon: func() fyne.Resource { return suggestIcon }, when: screenplayOnly, enabled: withSource, run: (*MainWindow).suggestSourceLinks},
		&command{id: "source.update", label: "Update Source Links", icon: theme.ViewRefreshIcon, when: screenplayOnly, enabled: withSource, run: (*MainWindow).updateSourceLinks},
		&command{id: "source.sections", label: "Source Headings as Sections", icon: res(tocIcon), when: screenplayOnly, enabled: withSource, run: (*MainWindow).sectionsFromSourceHere},
		// the common case of View > Swap Pane with Main (swap.go)
		&command{id: "source.swap", label: "Swap Source and Script", ribbonLabel: "Swap", tip: "The source in the main editor, the script in its place in the side panes; again, back",
			icon: res(swapIcon), when: screenplayOnly, enabled: withSource, run: (*MainWindow).swapSourceAndScript,
			checked: func(w *MainWindow) bool { return w.swapped == "source" }},
		&command{id: "source.learned", label: "What Accolade Learned…", icon: res(learnedIcon), when: screenplayOnly, enabled: withSource, run: (*MainWindow).showLearned},
	)

	// Production: a shooting script's (docs/PRODUCTION.md; production.go);
	// text is the Production pane's button's, if shorter
	prod := func(id, label string, ic func() fyne.Resource, run func(*MainWindow)) *command {
		return &command{id: id, label: label, icon: ic, when: screenplayOnly, run: run, role: productionRole}
	}
	unlock := prod("production.unlockNumbers", "Unlock Scene Numbers…", res(unlockIcon), (*MainWindow).unlockSceneNumbers)
	unlock.enabled, unlock.text = numbersLocked, "Unlock…"
	renumber := prod("production.renumber", "Renumber Scenes…", res(renumberIcon), (*MainWindow).renumberScenes)
	renumber.text = "Renumber"
	unlockPages := prod("production.unlockPages", "Unlock Pages…", res(unlockPagesIcon), (*MainWindow).unlockPages)
	unlockPages.enabled = pagesLocked
	withdraw := prod("production.withdraw", "Withdraw Last Issue…", theme.ContentUndoIcon, (*MainWindow).withdrawLastIssue)
	withdraw.enabled, withdraw.text = hasIssue, "Withdraw…"
	tag := prod("production.tag", "Tag Selection…", res(tagIcon), (*MainWindow).showTagPicker)
	tag.keys = []fyne.Shortcut{shortcut(fyne.KeyB, fyne.KeyModifierShift)}
	showTags := prod("production.showTags", "Show Tagged Words", theme.VisibilityIcon, (*MainWindow).toggleBreakdownTags)
	showTags.checked = func(w *MainWindow) bool { return w.settings == nil || !w.settings.GetBoolean(breakdownHiddenPref) }
	nextScene := prod("production.nextScene", "Next Scene to Break Down", theme.NavigateNextIcon, (*MainWindow).nextBreakdownScene)
	nextScene.text = "Next Scene"
	add(
		prod("production.lockNumbers", "Lock Scene Numbers", res(lockIcon), (*MainWindow).lockSceneNumbers),
		unlock, renumber,
		prod("production.omit", "Omit Scene", res(omitIcon), (*MainWindow).omitScene),
		prod("production.restore", "Restore Scene", res(restoreIcon), (*MainWindow).restoreScene),
		prod("production.issue", "Issue Draft…", res(issueIcon), (*MainWindow).showIssueDraft),
		withdraw,
		prod("production.lockPages", "Lock Pages", res(lockPagesIcon), (*MainWindow).lockPages),
		unlockPages,
		// the Revise stage's
		prod("production.revisedPages", "Revised Pages PDF…", res(pdfIcon), (*MainWindow).revisedPagesNow),
		prod("production.compare", "Compare with the Last Issue", res(compareIcon), (*MainWindow).compareWithLastIssue),
		tag,
		prod("production.untag", "Untag", theme.ContentClearIcon, (*MainWindow).untagSelection),
		showTags,
		prod("production.breakdown", "Break Down…", theme.ListIcon, (*MainWindow).showBreakdownStage),
		// the Break down stage's
		nextScene,
		prod("production.sceneDone", "Mark Scene Done", res(doneIcon), (*MainWindow).markSceneDoneHere),
		prod("production.sheets", "Breakdown Sheets…", res(sheetsIcon), func(w *MainWindow) { w.showReportsOf(lexreport.KindSheets) }),
		prod("production.stripboard", "Stripboard", func() fyne.Resource { return stripboardIcon }, (*MainWindow).toggleStripboard),
		prod("production.schedule", "Schedule…", theme.CalendarIcon, (*MainWindow).showScheduleStage),
		// the Schedule stage's
		prod("production.addDay", "Add Day", res(addDayIcon), (*MainWindow).addShootDay),
		prod("production.oneLiner", "One-Liner…", res(oneLinerIcon), func(w *MainWindow) { w.showReportsOf(lexreport.KindOneLiner) }),
		prod("production.dood", "Day Out of Days…", theme.GridIcon, func(w *MainWindow) { w.showReportsOf(lexreport.KindDOOD) }),
		prod("production.ics", "Export Shoot Days (.ics)…", theme.DownloadIcon, (*MainWindow).exportICS),
		prod("production.reports", "Reports…", res(reportsIcon), (*MainWindow).showReports),
	)
	r.byID["production.ics"].text = "Export .ics…"

	// Collaboration: writing together and the drafts
	add(&command{id: "collab.compare", label: "Compare…", icon: res(compareIcon), run: (*MainWindow).compareDrafts})

	// View
	add(
		// a screenplay's: the PDF as exported; a Markdown document's: the
		// live preview beside the editor, shown or hidden
		&command{id: "view.preview", label: "Preview PDF", mdLabel: "Live Preview", keys: []fyne.Shortcut{shortcut(fyne.KeyP, fyne.KeyModifierShift)}, icon: theme.VisibilityIcon,
			byEditor: true, run: (*MainWindow).preview, checked: func(w *MainWindow) bool { return w.editorMarkdown() && w.previewVisible }},
		&command{id: "view.foldHere", label: "Fold or Unfold Here", keys: []fyne.Shortcut{shortcut(fyne.KeyPeriod, fyne.KeyModifierShift)}, icon: res(foldIcon),
			run: func(w *MainWindow) { w.fold(false, true) }},
		&command{id: "view.foldSections", label: "Fold All (Sections Only)", icon: res(foldSectionsIcon), run: (*MainWindow).foldSections},
		&command{id: "view.headingsOnly", label: "Headings Only", icon: func() fyne.Resource { return headingIcon }, run: (*MainWindow).foldHeadings},
		&command{id: "view.unfoldAll", label: "Unfold All", icon: res(unfoldIcon), run: func(w *MainWindow) { w.foldAll(true) }},
		&command{id: "view.statistics", label: "Statistics…", icon: res(statisticsIcon), when: screenplayOnly, run: (*MainWindow).showStatistics},
		// Alt+Enter too, as in many programs
		&command{id: "view.fullscreen", label: "Fullscreen", tip: "Full screen, without distractions", icon: theme.ViewFullScreenIcon,
			keys: []fyne.Shortcut{shortcut(fyne.KeyF, fyne.KeyModifierShift), plainKey(fyne.KeyReturn, fyne.KeyModifierAlt), plainKey(fyne.KeyEnter, fyne.KeyModifierAlt)},
			run:  (*MainWindow).toggleFullscreen},
		&command{id: "view.documentType", label: "Document Type", icon: theme.FileIcon, submenu: func(w *MainWindow) *fyne.Menu { return w.documentTypeMenu().ChildMenu }},
		// Writer or Production (role.go), as Preferences > General has it
		&command{id: "view.role", label: "Working As", icon: theme.AccountIcon, when: screenplayOnly, submenu: func(w *MainWindow) *fyne.Menu { return w.roleMenu().ChildMenu }},
		// the ribbon's ☰: the menus as the menu bar has them (ribbon.go)
		&command{id: "view.menu", label: "Menus…", tip: "All the menus", icon: theme.MenuIcon, pick: (*MainWindow).showMenusAt},
		// F6 and Shift+F6: the next or previous pane (editor, outline,
		// notes, search), whatever has the focus
		&command{id: "view.nextPane", label: "Next Pane", keys: []fyne.Shortcut{plainKey(fyne.KeyF6, 0)}, run: func(w *MainWindow) { w.cyclePane(1) }},
		&command{id: "view.previousPane", label: "Previous Pane", keys: []fyne.Shortcut{plainKey(fyne.KeyF6, fyne.KeyModifierShift)}, run: func(w *MainWindow) { w.cyclePane(-1) }},
		// F10 (and Alt alone, in Ribbon mode) the ribbon's key tips (ribbon.go)
		&command{id: "view.keyTips", label: "Ribbon Key Tips", keys: []fyne.Shortcut{plainKey(fyne.KeyF10, 0)},
			run: func(w *MainWindow) {
				if w.ribbon != nil && w.ribbonOn() {
					w.ribbon.showKeyTips()
				}
			}},
	)

	// the command palette (palette.go): Ctrl+K, F1 (Ctrl+Shift+P is Preview)
	add(&command{id: "app.palette", label: "Command Palette…", tip: "Command palette: find any command", icon: func() fyne.Resource { return paletteIcon },
		keys: []fyne.Shortcut{shortcut(fyne.KeyK, 0), plainKey(fyne.KeyF1, 0)}, run: (*MainWindow).showPalette})
	add(&command{id: "help.about", label: "About Accolade", icon: theme.InfoIcon, run: func(w *MainWindow) { w.app.showAbout() }})
	add(dockCommands()...)    // the side dock's panes and its own (dock.go)
	add(helpCommands()...)    // help_pane.go
	add(settingCommands()...) // ribbon_settings.go

	elements := make([]home, len(toolbarElements))
	for i, el := range toolbarElements {
		elements[i] = at(elementCommandID(el))
	}
	headings := make([]home, 0, 7)
	for _, id := range mdHeadingIDs()[1:] {
		headings = append(headings, at(id))
	}
	headings = append(headings, at("md.heading0"))
	r.menuBar = []menuDef{
		{title: "File", homes: []home{
			at("file.new"), at("file.newEmpty"), at("file.newWindow"), at("file.newEmptyWindow"), at("file.newMarkdown"),
			at("file.open"), at("file.recent"), at("file.openSample"), at("file.import"), at("file.importPandoc"), at("file.startScreenplay"),
			sep(), at("file.save"), at("file.saveAs"), at("file.rename"), at("file.undoRename"), at("file.addSettings"),
			sep(), at("file.export"), at("file.exportClean"), at("file.pageSetup"), at("file.print"),
			// Fyne appends Quit (Ctrl+Q) to the first menu itself.
		}},
		{title: "Edit", homes: []home{
			at("edit.undo"), at("edit.redo"),
			sep(), at("edit.cut"), at("edit.copy"), at("edit.paste"), at("edit.selectAll"),
			sep(), at("edit.find"), at("edit.replace"), at("edit.findNext"), at("edit.findPrevious"),
			sep(), at("format.lineUp"), at("format.lineDown"),
			// the script's own: its title page and its language
			sep(), at("file.titlePage"), at("format.language"),
			sep(), at("app.preferences"),
		}},
		{title: "Format", homes: append(append(elements,
			at("format.rtl"), at("format.color"),
			sep(), at("format.bold"), at("format.italic"), at("format.underline"), at("md.strike"), at("md.code"),
			sep()), append(headings,
			sep(), at("md.link"), at("md.image"), at("md.footnote"),
			sep(), at("md.bullets"), at("md.numbers"), at("md.tasks"), at("md.quote"),
			sep(), at("md.codeBlock"), at("md.rule"), at("md.table"),
		)...)},
		{title: "Adaptation", when: screenplayOnly, homes: []home{
			at("source.choose"), at("source.saveAs"), at("source.keepInside"), as("view.source", "Source Pane"), at("source.swap"),
			sep(), at("source.link"), at("source.next"), at("source.cut"), at("source.suggest"), at("source.update"), at("source.sections"),
			sep(), at("source.learned"),
		}},
		{title: "Production", when: screenplayOnly, role: productionRole, homes: []home{
			at("production.lockNumbers"), at("production.unlockNumbers"), at("production.renumber"),
			sep(), at("production.omit"), at("production.restore"),
			sep(), at("production.issue"), at("production.withdraw"), at("production.lockPages"), at("production.unlockPages"),
			sep(), at("production.tag"), at("production.untag"), at("production.showTags"), at("production.breakdown"),
			at("production.stripboard"), at("production.schedule"), at("production.ics"),
			sep(), at("production.reports"),
		}},
		{title: "Collaboration", homes: []home{
			at("file.share"), at("file.join"),
			sep(), at("file.saveDraft"), at("file.drafts"), at("collab.compare"),
		}},
		{title: "View", homes: append(append([]home{
			at("view.preview"), at("view.foldHere"), at("view.foldSections"),
			at("view.headingsOnly"), at("view.unfoldAll"), sep()},
			dockHomes()...), // the side dock's panes together, and its own (dock.go)
			sep(), at("view.statistics"), at("view.fullscreen"), at("set.toolbar"),
			sep(), at("view.documentType"), at("view.role"),
		)},
		{title: "Help", homes: helpMenu()},
	}
	r.topRow = []home{
		at("file.new"), at("file.open"), at("file.save"), sep(),
		at("edit.undo"), at("edit.redo"),
		at("view.outline"), at("view.cards"), at("view.notes"), at("view.source"), at("view.production"), sep(),
		at("edit.findReplace"), at("app.palette"), at("view.preview"), at("file.export"), sep(),
		at("view.fullscreen"), at("app.preferences"),
	}
	// the element list (or buttons) after the switch, the format's own
	// buttons (format_buttons.go) at the end
	r.screenplayRow = []home{
		at("write.elementKind"), {id: "elements"}, at("write.place"), at("write.character"), at("format.color"),
		at("format.bold"), at("format.italic"), at("format.underline"), sep(), {id: "formatButtons"},
	}
	// a story's: the common Markdown, then its next step, a screenplay
	// adapting it
	r.markdownRow = []home{
		at("md.headings"), at("format.bold"), at("format.italic"), at("md.strike"), at("md.code"), at("md.link"),
		sep(), at("md.bullets"), at("md.numbers"), at("md.tasks"), at("md.quote"),
		sep(), at("file.startScreenplay"),
	}
	r.context = []home{
		madeFor(productionRole, (*MainWindow).tagAsLastItems), at("production.tag"), madeFor(productionRole, (*MainWindow).untagHereItems),
		sep(), at("edit.cut"), at("edit.copy"), at("edit.paste"),
	}
	for id, short := range ribbonLabels {
		cmd := r.byID[id]
		if cmd == nil {
			panic("ribbonLabels: no command " + id)
		}
		cmd.ribbonLabel = short
	}
	r.ribbon = ribbonLayout()
	r.ribbonEnd = []home{at("app.palette"), at("view.menu")}
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
			if !h.role.in(env) {
				continue
			}
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
		if m.when != nil && !m.when(env) || !m.role.in(env) {
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
		if pk.key.Key() == name && pk.key.Mod() == mod && pk.c.applies(cmdEnv{markdown: w.markdown, browser: inBrowser, writer: w.writerView()}) {
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

// homesOf are the places c is in the menu bar, as "Format > Bold", and,
// if it is in none, on the ribbon, as "Production > Lock > Lock Pages"
// (for the palette).
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
	if len(out) > 0 {
		return out
	}
	for _, t := range commands().ribbon {
		for _, g := range t.groups {
			for _, h := range g.homes {
				if h.id == id {
					return []string{t.title + " > " + g.title + " > " + cmd(id).label}
				}
			}
		}
	}
	return nil
}

// ribbonTab is a tab of the ribbon (ribbon.go): its groups of commands,
// shown when it applies (when; nil: always).
type ribbonTab struct {
	title  string
	when   func(cmdEnv) bool
	role   cmdRole
	groups []ribbonGroup
}

// ribbonGroup is a group of a ribbon tab: its title under its buttons;
// when, if set, says when it is there; stage, if not 0, is the Production
// stage (1 + stageLock …) whose actions it is: the group is there only
// while the script is at that stage.
type ribbonGroup struct {
	title   string
	mdTitle string // a Markdown document's title, if another
	when    func(cmdEnv) bool
	stage   int
	homes   []home
}

// ribbonLabels are the ribbon's short labels, where the group around a
// button says the rest: "Next" in Adaptation > Links is the next source
// passage. The full label stays the button's tooltip and screen-reader
// name; the menus and the palette keep it. (A label's "…" never shows on
// the ribbon: ribbonLabelIn.)
var ribbonLabels = map[string]string{
	// File (the browser's Open and Save say more in their menu items)
	"file.open": "Open", "file.save": "Save",
	"file.newEmpty": "Empty", "file.newMarkdown": "Markdown", "file.newWindow": "Window", "file.newEmptyWindow": "Empty Window",
	"file.recent": "Recent", "file.openSample": "Sample", "file.import": "Import", "file.importPandoc": "With pandoc",
	"file.exportClean": "Clean Fountain", "file.pageSetup": "Page Setup",
	"source.choose": "Choose Source", "file.startScreenplay": "Start Screenplay",
	// Write
	"format.rtl": "Right to Left", "md.rule": "Rule", "md.bullets": "Bullets", "md.numbers": "Numbered", "md.tasks": "Tasks",
	"md.code": "Code", "md.strike": "Strike",
	// Edit
	"edit.findNext": "Next", "edit.findPrevious": "Previous", "format.lineUp": "Move Up", "format.lineDown": "Move Down",
	"format.language": "Language",
	// Adaptation
	"source.saveAs": "Save As", "source.keepInside": "Keep in Script", "source.next": "Next", "source.link": "Link",
	"source.cut": "Mark as Cut", "source.suggest": "Suggest", "source.update": "Update",
	"source.sections": "Headings as Sections", "source.learned": "Learned Notes",
	// Production
	"production.lockNumbers": "Lock Numbers", "production.unlockNumbers": "Unlock", "production.issue": "Issue",
	"production.revisedPages": "Revised PDF", "production.compare": "Compare", "production.withdraw": "Withdraw",
	"production.sceneDone": "Scene Done", "production.tag": "Tag", "production.showTags": "Show Tags",
	"production.sheets": "Sheets", "production.ics": "Export .ics", "production.omit": "Omit", "production.restore": "Restore",
	// Collaboration
	"file.join": "Join",
	// View
	"view.preview": "Preview", "view.foldHere": "Fold Here", "view.foldSections": "Fold Sections",
	"app.palette": "Commands",
	// the settings (ribbon_settings.go)
	"set.suggest": "Suggest While Writing", "set.nextInserts": "Next Starts a Scene",
	"set.revisionMarks": "Revision Marks", "set.lettering": "Scene Lettering", "set.continueds": "CONTINUEDs",
	"set.sceneNumbers": "Scene Numbers", "set.draftsSide": "Side by Side",
}

// ribbonLayout is the ribbon: the tabs follow the menus (File, Write for
// Format, Edit, Adaptation, Production, Collaboration, View); "elements",
// "formatButtons" and "scriptFormat" are the Write tab's element buttons,
// the script format's own buttons and its list.
func ribbonLayout() []ribbonTab {
	noSource := func(e cmdEnv) bool { return !e.markdown && !e.source }
	noMarkdown := func(e cmdEnv) bool { return !e.markdown }
	markdownEnv := func(e cmdEnv) bool { return e.markdown }
	return []ribbonTab{
		{title: "File", groups: []ribbonGroup{
			{title: "New", homes: []home{big("file.new"), at("file.newEmpty"), at("file.newMarkdown"), at("file.newWindow"), at("file.newEmptyWindow"),
				at("set.newLanguage"), at("file.openSample")}}, // "New in: English": the language they are in; the sample to try things on
			{title: "Open", homes: []home{big("file.open"), at("file.recent"), at("file.import"), at("file.importPandoc")}},
			{title: "Save", homes: []home{big("file.save"), at("file.saveAs"), at("file.rename"), at("file.undoRename")}},
			{title: "Output", when: noMarkdown, homes: []home{big("file.export"), big("file.print"), at("file.exportClean"), at("file.pageSetup")}},
			// a document's, without Clean Fountain: Print small over Page Setup, so the tab fits
			{title: "Output", when: markdownEnv, homes: []home{big("file.export"), at("file.print"), at("file.pageSetup")}},
			// a source to adapt: the Adaptation tab once there is one
			{title: "Adapt", when: noSource, homes: []home{big("source.choose")}},
			{title: "Adapt", homes: []home{big("file.startScreenplay")}},
		}},
		{title: "Write", groups: []ribbonGroup{
			{title: "Element", homes: []home{{id: "elements"}}},
			{title: "Scene", homes: []home{at("write.place"), at("write.character"), at("format.color")}},
			// a Markdown document's elements: its headings, given by their keys
			// as a screenplay's elements are (Ctrl+1…6, Ctrl+0)
			{title: "Heading", homes: []home{keyed("md.heading1"), keyed("md.heading2"), keyed("md.heading3"),
				keyed("md.heading4"), keyed("md.heading5"), keyed("md.heading6"), keyed("md.heading0"),
				{id: "md.headings", large: true, phoneOnly: true}}},
			{title: "Emphasis", homes: []home{tool("format.bold"), tool("format.italic"), tool("format.underline"), tool("md.strike"), tool("md.code"),
				sep(), tool("format.rtl")}}, // a row of its own, under them
			{title: "Insert", homes: []home{big("md.link"), at("md.image"), at("md.footnote"), at("md.table"), at("md.rule"), at("md.codeBlock")}},
			{title: "Lists", homes: []home{at("md.bullets"), at("md.numbers"), at("md.tasks"), at("md.quote")}},
			// the script's format, its title page, the format's own blocks
			{title: "Script", homes: []home{{id: "scriptFormat"}, at("file.titlePage"), {id: "formatButtons"}}},
		}},
		{title: "Edit", groups: []ribbonGroup{
			{title: "Undo", homes: []home{at("edit.undo"), at("edit.redo")}},
			{title: "Clipboard", homes: []home{at("edit.cut"), at("edit.copy"), at("edit.paste"), at("edit.selectAll")}},
			// verb + variants (as File's New, Open, Save): the big verb, the
			// small ones named by what differs
			{title: "Find", homes: []home{big("edit.find"), at("edit.findNext"), at("edit.findPrevious"), at("edit.replace")}},
			{title: "Lines", homes: []home{at("format.lineUp"), at("format.lineDown")}},
			{title: "Script", mdTitle: "Document", homes: []home{at("file.titlePage"), at("format.language")}},
			// the editor's keys (standard, vim, helix), beside the rest of the preferences
			{title: "Settings", homes: []home{at("set.editorKeys"), at("app.preferences")}},
		}},
		// a source's, or a story's next step
		{title: "Adaptation", when: func(e cmdEnv) bool { return e.source || e.markdown }, groups: []ribbonGroup{
			{title: "Source", homes: []home{{id: "view.source", label: "Source Pane", short: "Source", large: true},
				atShort("source.choose", "Choose"), at("source.saveAs"), at("source.keepInside"), at("source.swap")}},
			// Next is the adapting's step, Link its verb; Suggest and Update
			// are linking's variants
			{title: "Links", homes: []home{big("source.next"), big("source.link"), at("source.suggest"), at("source.update"), at("source.cut"), at("source.sections")}},
			{title: "Learned", homes: []home{big("source.learned")}},
			{title: "Options", homes: []home{at("set.suggest"), at("set.nextInserts")}},
			{title: "Start", homes: []home{big("file.startScreenplay")}},
		}},
		// the stage's actions first, as the Production pane has them
		{title: "Production", when: screenplayOnly, role: productionRole, groups: []ribbonGroup{
			{title: "Lock", stage: 1 + stageLock, homes: []home{big("production.lockNumbers"), at("production.unlockNumbers"), at("production.renumber"),
				big("production.lockPages"), big("production.issue")}},
			{title: "Revise", stage: 1 + stageRevise, homes: []home{big("production.issue"), at("production.withdraw"), at("production.compare"), at("production.revisedPages")}},
			{title: "Break Down", stage: 1 + stageBreakdown, homes: []home{big("production.nextScene"), big("production.tag"),
				at("production.sceneDone"), at("production.untag"), at("production.showTags"), at("production.sheets")}},
			{title: "Schedule", stage: 1 + stageSchedule, homes: []home{big("production.stripboard"), at("production.addDay"), at("production.oneLiner"),
				at("production.dood"), at("production.ics")}},
			{title: "Stages", homes: []home{bigAs("view.production", "Production Pane"), at("production.breakdown"), at("production.schedule")}},
			{title: "Scenes", homes: []home{at("production.omit"), at("production.restore"), at("production.unlockPages")}},
			{title: "Reports", homes: []home{big("production.reports")}},
			{title: "Options", homes: []home{at("set.revisionMarks"), at("set.lettering"), at("set.continueds"), at("set.sceneNumbers"), at("file.pageSetup")}},
		}},
		{title: "Collaboration", groups: []ribbonGroup{
			{title: "Together", homes: []home{big("file.share"), at("file.join"), at("view.changes")}},
			{title: "Drafts", homes: []home{big("file.saveDraft"), at("file.drafts"), at("collab.compare"), at("set.draftsSide")}},
		}},
		{title: "View", groups: []ribbonGroup{
			// the side dock's panes (dock.go)
			{title: "Panes", homes: dockRibbonHomes()},
			// what the window shows; its look (the preferences changed while working: ribbon_settings.go)
			{title: "Show", homes: []home{big("view.preview"), at("view.statistics"), at("set.lineNumbers"), at("set.statusBar")}},
			{title: "Folding", homes: []home{at("view.foldHere"), at("view.foldSections"), at("view.headingsOnly"), at("view.unfoldAll")}},
			{title: "Window", homes: []home{at("view.fullscreen"), at("view.documentType"), at("view.role"), at("set.ribbon"), at("set.theme"),
				tool("set.textSmaller"), tool("set.textLarger"), tool("set.font")}},
		}},
	}
}
