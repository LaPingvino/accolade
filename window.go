package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/accolade/internal/pace"
	"github.com/LaPingvino/accolade/internal/tooltip"
	"github.com/LaPingvino/lexington/fountain"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/pdfin"
)

type MainWindow struct {
	fyneWindow fyne.Window
	app        *Application

	// Core components
	textEditor *editor.ScriptEditor
	editorView *container.ThemeOverride // the editor with its font settings
	headerBar  *HeaderBar
	tooltips   *tooltip.Layer
	// noteTipShown is the note's text shown over its mark in the editor
	// ("" none: markdown_notes.go)
	noteTipShown string
	chips        chipCache // the script's links as chips, as found last (link_chips.go)
	searchBar    *SearchBar

	// UI layout
	mainContainer    *container.Split
	sidePanel        *container.Split
	toolbarContainer *fyne.Container
	// frame is the window's content around the script: the toolbar (or
	// ribbon) on top, the status bar at the foot (relayout)
	frame *fyne.Container
	// Status bar
	statusBar *fyne.Container
	// bottom is the status bar with the notice above it that a script
	// relied on Lexington's old reading (loose_reading.go)
	bottom       *fyne.Container
	loose        *looseNotice
	looseUnsaved string // the answer for a script not saved yet
	// headerArea is the header bar (the toolbar) with its separator
	headerArea *fyne.Container
	// the sidebar beside the editor: the outline (View > Outline) or the
	// notes (View > Notes)
	// vim is the editor's vim mode (vim.go), on with the setting
	vim *vim
	// keysApplied: the editor keys the vim state was last reset for
	// (applyEditorSettings), "" before the first
	keysApplied string
	// the ex command line (ex_window.go)
	exState  exState
	exBar    *fyne.Container
	exEntry  *exLine
	exLabel  *widget.Label
	exPrompt func(string) // the line is a prompt's (helix s), not a command
	exVisual [2]int
	exPop    *exPop // the ':' line's completion list (ex_window.go)
	// the middle tabs: the script, the cards' full view, the panes put
	// there (middle.go)
	mid middle
	// epoch counts the script's changes (scriptChange); changing is the
	// depth of the transitions under way (state.go)
	epoch    uint64
	changing int
	// what the page view, the status bar, the title and the commands
	// were last made for (state.go)
	pagesWant, statusOf, titleOf, commandsOf docKey
	// pagesComing is the Preview being made whose pages will be shown
	// (its previewGen; 0 none)
	pagesComing int
	// pagesPreview: the page view shows the script's Preview (not
	// Export's preview or a report's PDF, which are not followed)
	pagesPreview bool
	// the window's documents (document.go): the script as one, the
	// source and the notes once opened
	script, srcDoc, notesDoc *document
	// passageSeen are the source's passages' lines as found last, for the
	// main cursor (source_passage_lines.go)
	passageSeen linesSeen
	cardsDock   *cardsPane // the Cards pane (cards_pane.go)
	// the Characters pane, and the last rename for Undo (characters_pane.go, characters_rename.go)
	charsPane   *charactersPane
	charRenamed *renamed
	// relinking is how many looks at the links are being worked out (relink.go)
	relinking int
	// linkGuesses are the links the relinking did not write, by their
	// rawKey: a guess to check, or a warning (relink.go)
	linkGuesses map[string]linkGuess
	// the others' changes to a shared script (changes.go)
	changes      []change
	changesPanel *changesPanel
	cards        *cardsView
	scriptFolds  *folds // the script editor's (editor_folds.go)
	// mdCounted is the Markdown text the status bar counted last (markdown.go)
	mdCounted struct {
		text, out string
		ok        bool
	}
	// language is the script's language ("en", "eo", ...): its scene
	// headings (language.go)
	language string
	// noFountainCopy: the script's .accolade keeps no text.fountain beside
	// its language's text (script_text.go)
	noFountainCopy bool
	// carried is the Lexington configuration the script brought (an
	// .accolade's, lexington_carried.go), laid over the machine's for it
	// only; nil for none
	carried *carriedConf
	// ownFormat is the script's format when its .accolade names one
	// (else the settings' "script-format"), scriptFormatKey
	ownFormat string
	// adoptAsked is the Add to My Settings dialog shown last (for tests)
	adoptAsked *adoptDialog
	// languageSaid: the script's language was said (by its file, or chosen
	// since), not only assumed from the preferences (script_text.go)
	languageSaid bool
	// format is how the script's file stores it (encoding.go)
	format fileFormat
	// bundle is the TextBundle (.textbundle, .textpack, .highland, .accolade) the
	// script is the text of, nil for a plain file (bundle.go)
	bundle *textBundle
	// plainBundle is a file with a bundle's name holding plain text (a
	// .highland that is no zip), saved as plain text again
	plainBundle string
	// bundleQ are the writes into the bundle, in the background
	// (bundle_queue.go); bundleGen counts the bundles the window had
	bundleQ      bundleQueue
	bundleGen    int
	saveFailures int            // background saves that failed (bundle_queue.go)
	srcPending   *pendingSource // a source being written into the bundle (source_file.go)
	// imported is closed when the PDF import under way is in
	imported chan struct{}
	// the side dock and its panes (dock.go), each made at its first need
	dock          *dock
	outline       *outlinePanel
	notes         *notesPanel
	source        *sourcePane
	production    *productionPane
	help          *helpPane
	narrow        bool             // a phone's width: the panes under the script (relayout)
	paneSplit     *container.Split // the script over the panes, while narrow
	sideSplit     *container.Split // the panes beside the script, while wide
	paneTaller    *tooltip.Button  // makes them taller, or half again
	unsavedSource string           // the source chosen for a script not saved yet (source.go)
	unsavedExtras string           // the kept extras of a script not saved yet (extras.go)
	// flattenTo is where a bundle's extras go when it is saved as a plain
	// file, as asked (flatten.go); then back to the kept copy
	flattenTo    flattenChoice
	flattenAsked *flattenAsk // the question last asked (saveas.go)
	// a rename or Save As under way, and the last rename, to take back
	// (relocate.go); the source carried into a new bundle, or put beside
	// a plain file leaving one, as it is written
	relocating, renameUndo *fileJournal
	carrySource            []byte
	renameAsked            *renameAsk        // the rename editor last opened (rename.go)
	cleanAsked             *cleanFountainDlg // Export Clean Fountain… last opened (clean_fountain.go)
	saveAsAsked            *saveAsAsk        // the Save As chooser last opened (saveas_dialog.go)
	sourceBeside           string
	highlandWarned         map[string]bool // the files the Highland warning was given for
	// markdown: the document is Markdown, not a screenplay (markdown.go)
	markdown       bool
	previewPane    *previewPanel // a Markdown document's preview
	previewSetup   *previewSetup // Preview PDF's page setup (pagesetup.go)
	previewVisible bool
	pages          *pageView // the page view: a PDF's pages (pageview.go)
	pagesVisible   bool
	previewDir     string // the folder of the PDF the page view shows
	previewGen     int    // Preview PDF's latest making: an older one's pages are dropped

	// File management
	currentFile string
	hasChanges  bool
	// filledWith is the text the window was filled with and has no file
	// for: the sample script, or an import; while the text is that,
	// opening a file replaces it (openPath)
	filledWith string
	// shared: the script edited together with peers (collab_window.go)
	shared       *shared
	isFullscreen bool
	// menuShortcuts are the menu's shortcuts in full screen, without the
	// menu bar: on the canvas, and for the editor (menuShortcut)
	menuShortcuts map[string]menuShortcut

	// the ribbon, once the look is Ribbon (ribbon.go), its refresh when
	// typing pauses, and the command palette while open (palette.go)
	ribbon      *ribbonBar
	ribbonTimer *uiTimer
	// ribbonOrder is the ribbon's order in this window, as its lock set it:
	// "smart", "fixed", or "" (Smart on a phone, Fixed on the desktop: quick.go)
	ribbonOrder string
	quickTimer  *uiTimer    // the usage counts' save
	ownQuick    *quickStore // the counts of a window with no app (a test's)
	palette     *palette

	// Settings
	settings *Settings

	// Find/Replace
	findVisible bool

	// Name offered by Save As for an imported (converted) document
	suggestedName string

	// Auto-save (autosave.go)
	autoSaveTimer *uiTimer
	timers        uiTimers   // the window's (background.go)
	followTimer   *uiTimer   // the source pane's follow, after typing
	flashTimer    *uiTimer   // a scene's flash taken away (flashScene)
	suggest       *suggester // suggestions from the source while writing (suggest.go)
	complete      *completer // Tab's completions of names and headings (complete.go)
	seenRev       uint64     // the text's revision the cursor was last seen in
	importGen     int        // the import under way (a later one wins)
	bundleSrc     struct {   // the bundle's source, decoded (sourceText)
		raw  []byte
		text string
		ok   bool
	}
	draftTimer       *uiTimer       // the browser keeping the script, after typing
	sceneNumberTimer *uiTimer       // a locked script's new scenes numbered, after typing (production.go)
	revisionTimer    *uiTimer       // the revision marks updated, after typing (revisions.go)
	revisions        revisionState  // the script's issues and the marks shown
	unsavedRole      string         // an unsaved script's own role, "" none (role.go)
	menuGone         bool           // the ribbon replaces the in-window menu bar (ribbon.go)
	revisionGen      int            // a new issue (the issues read again)
	revisionRun      int            // the marks being computed
	breakdownTimer   *uiTimer       // the tags found again, after typing (breakdown_view.go)
	breakdown        breakdownState // the script's breakdown and its tags as found
	breakdownGen     int            // the breakdown read again (an import)
	numbers          numbersSeen    // the scene numbers the extras follow (renumber_follow.go)
	renumbering      *renumberDlg   // Renumber Scenes… as last opened (renumber_dialog.go)
	breakdownRun     int            // the tags being found
	schedule         scheduleState  // the shooting schedule (schedule_state.go)
	stripsVisible    bool           // the stripboard in the editor's place (stripboard.go)
	stripboard       *stripboardView
	lastTagCategory  string         // the category last tagged in (offered first)
	lastReports      *reportsDialog // the reports last shown (tests)
	lastPicker       *tagPicker     // the tag picker last shown (tests)
	sourceGen        int            // the Choose Source under way
	autoSaveDelay    time.Duration  // overrides the setting; for tests

	statusLabel *widget.Label
	statusHint  string   // the "Tab: …" hint shown, "" none (status_hint.go)
	tabChip     *tabChip // before it: a tap is Tab
	statsLabel  *widget.Label

	// Lexington integration
	lexParser *LexingtonParser

	// Current cursor position for element detection
	currentElement string
	flow           keyFlow // the Final Draft keyboard flow (keyflow.go)
}

func NewMainWindow(app *Application) *MainWindow {
	window := &MainWindow{
		fyneWindow:   app.fyneApp.NewWindow("Accolade"),
		app:          app,
		hasChanges:   false,
		isFullscreen: false,
		findVisible:  false,
		lexParser:    NewLexingtonParser(),
	}

	window.settings = GetSettings() // shared with the preferences dialog
	// the theme before the window is built: setting it later themes the
	// whole of every window again
	app.setColorScheme(window.settings.GetString("theme"))
	window.dock = newDock(window)
	window.mid.w = window
	window.setupUI()
	window.setupShortcuts()
	window.setupCallbacks()
	window.updateTitle()
	window.updateCurrentElement()
	window.updateStatusBar()

	return window
}

func (w *MainWindow) setupUI() {
	// Set window properties
	w.fyneWindow.SetTitle("Accolade")
	w.fyneWindow.Resize(fyne.NewSize(1000, 600))
	w.fyneWindow.CenterOnScreen()

	// Create main text editor with Courier Prime font
	w.textEditor = newScriptEditor(w)
	w.textEditor.OnCursorChanged = w.onCursorChanged
	w.textEditor.OnHover = w.noteHover

	// Add some initial content - start with a title page
	w.textEditor.SetText(defaultScript(time.Now()))
	w.filledWith = w.textEditor.Text()

	// Apply Courier Prime font for screenplay formatting
	// the editor scrolls to keep the cursor in view; the theme override
	// carries the font settings
	w.editorView = container.NewThemeOverride(editor.NewPageScroll(w.textEditor, 0), newEditorTheme("", 0))

	// Create header bar with toolbar
	w.tooltips = tooltip.NewLayer()
	w.headerBar = NewHeaderBar(w)

	// Create search bar (initially hidden)
	w.searchBar = NewSearchBar(w)

	// Create toolbar container
	// the toolbar (the header bar) can be hidden, the search bar stays
	w.headerArea = container.NewVBox(w.headerBar.container, widget.NewSeparator())
	w.toolbarContainer = container.NewVBox(
		w.headerArea,
		w.searchBar.container,
	)

	// Create status bar with element indicator
	// Status bar: messages on the left (saved, auto-saved), the element
	// under the cursor and the script's statistics on the right
	small := func(text string) *widget.Label {
		l := widget.NewLabel(text)
		l.SizeName = theme.SizeNameCaptionText
		l.Importance = widget.LowImportance
		return l
	}
	w.statusLabel = small("")
	w.statusLabel.Wrapping = fyne.TextWrapWord // a phone's width: on more lines, not wider than the screen
	w.tabChip = newTabChip(w.takeHint)
	w.tabChip.Hide()
	w.statsLabel = small("")
	w.statsLabel.Alignment = fyne.TextAlignTrailing
	w.statsLabel.Truncation = fyne.TextTruncateEllipsis // a narrow window shows what fits
	w.statusBar = container.NewVBox(
		widget.NewSeparator(),
		container.New(&statusRow{}, w.tabChip, w.statusLabel, w.statsLabel),
	)
	w.bottom = container.NewVBox(w.newLooseNotice(), w.statusBar)

	w.relayout()

	// Apply initial settings
	w.applySettings()
}

func (w *MainWindow) setupShortcuts() {
	// Text editor shortcuts
	w.textEditor.OnChanged = w.onTextChanged
	w.setLanguage("")                      // the preferences' until a file says otherwise
	w.fyneWindow.SetOnDropped(w.onDropped) // drop a script or a PDF to open it

	// Lines are formatted when completed with Enter (editor.go)

	// Keyboard shortcuts are attached to the main menu items (see menus.go);
	// in Ribbon mode, with no menu bar, to the window (refreshMenu)
	w.refreshMenu()
	// the keys not on a menu item (keyPreview)
	if c, ok := w.fyneWindow.Canvas().(desktop.KeyPreviewCanvas); ok {
		c.SetOnKeyPreview(w.keyPreview)
	}
	// Alt let go alone: the ribbon's key tips (ribbon.go), seen where the
	// keyboard is: an editor (attachVim), the ribbon, or nothing focused
	if c, ok := w.fyneWindow.Canvas().(desktop.Canvas); ok {
		c.SetOnKeyUp(func(k *fyne.KeyEvent) { w.altUp(k.Name) })
	}
}

// keyPreview sees each key before the focused widget: the commands'
// keys that are not on a menu item (Alt+Enter for full screen, F6 and
// Shift+F6 for the panes; commands.go); it reports whether it took the key.
func (w *MainWindow) keyPreview(key fyne.KeyName, mod fyne.KeyModifier) bool {
	keyArrived()
	if t, ok := w.fyneWindow.Canvas().Focused().(*dockTab); ok && key == fyne.KeyF10 && mod == fyne.KeyModifierShift {
		t.showMenu(fyne.Position{}) // a dock tab's menu (dock.go)
		return true
	}
	w.altKey(key) // Alt alone shows the ribbon's key tips
	return w.previewKey(key, mod)
}

func (w *MainWindow) setupCallbacks() {
	// Window close callback
	var askClose func()
	askClose = func() {
		w.dock.keepSizes()  // the dock's width and split, for next time
		if w.bundleBusy() { // a save under way finishes first, then asks as ever
			w.setStatus("Saving...")
			w.afterBundleWrites(askClose)
			return
		}
		if w.hasChanges || w.unsavedDocs() {
			dialog.ShowConfirm(
				"Unsaved Changes",
				"You have unsaved changes. Are you sure you want to close?",
				func(confirmed bool) {
					if confirmed {
						w.fyneWindow.Close()
					}
				},
				w.fyneWindow,
			)
		} else {
			w.fyneWindow.Close()
		}
	}
	w.fyneWindow.SetCloseIntercept(askClose)
}

// TODO: Implement file drop handling when needed
// Fyne's drag and drop API may differ from GTK

func (w *MainWindow) applySettings() {
	// (the theme: NewMainWindow, before the window is built)

	// Apply editor settings
	w.applyEditorSettings()

	// Apply window settings

	// Apply font settings
	w.applyFontSettings()

	w.updateRevisionMarks() // since the last issue or all (revisions.go)
}

func (w *MainWindow) applyEditorSettings() {
	// the script editor always wraps at the window width (screenplays are
	// laid out in columns)
	w.textEditor.SetLineNumbers(w.settings.GetBoolean("show-line-numbers"))
	setVisible(w.headerArea, w.settings.GetBoolean("toolbar-visible"))
	w.applyLook() // the header's rows or the ribbon
	// other editor keys (Preferences, the ribbon, the palette): the
	// keys' state starts again, in normal mode for vim and helix, and the
	// status line says which keys now
	if keys := w.editorKeys(); w.vim != nil && keys != w.keysApplied {
		first := w.keysApplied == ""
		w.keysApplied = keys
		w.vim.reset(keys != "standard")
		for _, d := range w.openDocs() { // the documents' editors too (document.go)
			d.vim.reset(keys != "standard")
		}
		if !first {
			w.setStatus(editorKeysStatus(keys))
		}
	}
	setVisible(w.statusBar, w.settings.GetBoolean("statusbar-visible"))
	// auto-indent (formatting the line on Enter) is read when Enter is pressed (editor.go)
	w.applyScriptFormat()
}

// setCarried makes c what the script brought and format its own format
// ("" the settings'), as a document is opened or made.
func (w *MainWindow) setCarried(c *carriedConf, format string) {
	if c == w.carried && format == w.ownFormat {
		return
	}
	w.carried, w.ownFormat = c, format
	if w.textEditor == nil {
		return
	}
	w.applyScriptFormat()
	if w.headerBar != nil && w.headerBar.format != nil {
		w.headerBar.showFormats()
	}
	w.ribbonRefresh()
}

// applyScriptFormat lays the editor out in the script format's columns.
func (w *MainWindow) applyScriptFormat() {
	w.textEditor.Format, w.textEditor.PageColumns = editorFormat(w.currentScriptFormat())
	if w.markdown { // prose: a column of text, no screenplay layout
		w.textEditor.Format, w.textEditor.PageColumns = nil, markdownColumns
	}
	w.textEditor.Refresh()
	w.shareMeta("format", w.scriptFormatKey())
}

func setVisible(o fyne.CanvasObject, visible bool) {
	if o == nil {
		return
	}
	if visible {
		o.Show()
	} else {
		o.Hide()
	}
}

func (w *MainWindow) applyFontSettings() {
	w.editorView.Theme = newEditorTheme(w.settings.GetString("font-family"), w.settings.GetInt("font-size"))
	w.editorView.Refresh()
	w.applyDocFonts(newEditorTheme(w.settings.GetString("font-family"), w.settings.GetInt("font-size")))
}

// File operations
func (w *MainWindow) LoadFile(filePath string) error {
	// the views follow at the end (state.go)
	defer w.scriptChange()()
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".fdx", ".pdf":
		return w.importFile(filePath)
	}

	// the text of a .textbundle folder opens the bundle
	if b, ok := bundleOf(filePath); ok {
		filePath = b
	}
	// all read before anything changes: a file that cannot be read
	// leaves the window as it was
	var bundle *textBundle
	plain := ""
	var content []byte
	var err error
	if bundleKindOf(filePath) != notBundle {
		bundle, content, err = readBundle(filePath)
		if errors.Is(err, errNotZip) && strings.EqualFold(filepath.Ext(filePath), ".highland") {
			bundle, plain = nil, filePath // Fountain with Highland's name
			content, err = os.ReadFile(filePath)
		}
	} else {
		content, err = os.ReadFile(filePath)
	}
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}
	w.newDocument() // read: a shared script stays with the peers
	w.plainBundle = plain
	w.setBundle(bundle)
	w.forgetUnsavedExtras()
	// what an .accolade brought, for this script only (lexington_carried.go)
	if bundle != nil && carriesConf(filePath) {
		w.setCarried(bundle.carried, bundle.ours.Format)
	} else {
		w.setCarried(nil, "")
	}

	// as stored: Windows-1252, a byte order mark, CR LF are kept for saving
	text, format := decodeScript(content)
	if bundle != nil && bundle.noText {
		_, format = decodeScript(bundle.nativeText)
	}
	w.format = format
	// as chosen for it (View > Document Type), else as its extension says
	w.setMarkdown(w.typeOf(filePath))
	// the language, and the text in it (script_text.go)
	lang, before := "", ""
	w.noFountainCopy = bundle != nil && bundle.ours.NoFountainCopy
	switch {
	case w.markdown:
		lang = languageOf(filePath)
	case bundle != nil:
		text, before = bundleScriptText(bundle, content, text)
		lang = bundle.lang
	default:
		text, lang = plainFileText(filePath, text)
	}
	if bundle != nil && !w.markdown { // a bundle saying none is standard Fountain
		lang = cmp.Or(lang, w.lex().standard())
	}
	w.setLanguage(lang) // none: the preferences', assumed
	w.languageSaid = lang != ""
	if !w.markdown { // Markdown's indents are its own (lists, code)
		text = withoutIndents(text, w.sceneStarts())
		before = withoutIndents(before, w.sceneStarts())
	}
	if before != "" { // another app changed text.fountain: kept as a draft first
		w.textEditor.SetText(before)
		w.addDraft("Before the change to text.fountain made in another app", true, nil)
	}
	w.textEditor.SetText(text)
	w.currentFile = filePath
	w.suggestedName = ""
	w.rememberRecent(filePath)
	w.hasChanges = before != "" // both texts the same again on the next save
	w.readBeatTags()
	w.roleChanged(false) // its own role, if it keeps one (role.go)
	if before != "" {
		w.setStatus("text.fountain was changed in another app: that change is the script now; the script before it " +
			"is kept as a draft (Collaboration > Compare… shows the change)")
	} else if msg := w.carried.broughtStatus(); msg != "" {
		w.setStatus(msg)
	} else if f := w.ownFormat; f != "" && !w.lex().hasFormat(f) {
		w.setStatus("This script's format, " + f + ", is not here: it is shown as a screenplay")
	} else if msg := w.lockedForWriter(); msg != "" {
		w.setStatus(msg) // quietly, last: production put away (role.go)
	}
	w.checkLooseReading() // read otherwise than Lexington before 1.19 did? (loose_reading.go)

	return nil
}

// bundleScriptText is a screenplay bundle's text for the editor, given
// its text.fountain as stored (content) and read (text): the language's
// text, unless another app changed text.fountain since Accolade wrote it
// — then that, its force marks taken back, and before is the language's
// text it replaces.
func bundleScriptText(b *textBundle, content []byte, text string) (now, before string) {
	conf := b.lex()
	if b.native == "" {
		return conf.dialectFountain(text, b.lang), ""
	}
	native, _ := decodeScript(b.nativeText)
	if b.noText {
		return native, ""
	}
	theirs := conf.dialectFountain(text, b.lang)
	changed := theirs != native
	if h := b.ours.FountainHash; h != "" {
		changed = h != fountainHash(content) && changed
	}
	if !changed {
		return native, ""
	}
	return theirs, native
}

// importFile opens a Final Draft document or a PDF of a screenplay
// converted to Fountain. It has no file of its own until saved, so Save
// asks where to put the .fountain file instead of overwriting the
// original.
func (w *MainWindow) importFile(filePath string) error {
	var text, from string
	var err error
	var sets []lex.RevisionSet
	var marks []importedMark
	var tags breakdownFile
	if strings.EqualFold(filepath.Ext(filePath), ".pdf") {
		w.importPDFAsync(filePath)
		return nil
	} else {
		var f *os.File
		if f, err = os.Open(filePath); err != nil {
			return fmt.Errorf("failed to read file: %v", err)
		}
		defer f.Close()
		text, sets, marks, tags, err = importFDXTagged(f)
		from = "Final Draft"
	}
	if err != nil {
		return fmt.Errorf("could not import %s: %v", filepath.Base(filePath), err)
	}

	w.showImported(filePath, text, from)
	if len(sets) > 0 {
		w.keepImportedRevisions(filepath.Base(filePath), sets, marks)
	}
	if len(tags.Tags) > 0 {
		if n := w.keepImportedBreakdown(tags); n > 0 {
			w.setStatus("Imported from " + from + " with its breakdown: " + plural(n, "tag") + ", kept by their words")
		}
	}
	return nil
}

// showImported puts an imported script in the window, unsaved.
func (w *MainWindow) showImported(filePath, text, from string) {
	// the views follow at the end (state.go)
	defer w.scriptChange()()
	w.newDocument()
	w.cancelAutoSave()
	w.forgetUnsavedSource()
	w.format = fileFormat{}
	w.setMarkdown(false)
	w.textEditor.SetText(text)
	w.looseUnsaved = ""
	w.hideLooseNotice() // written by Lexington: read as written
	w.filledWith = w.textEditor.Text()
	w.currentFile, w.plainBundle = "", ""
	w.setBundle(nil)
	w.setCarried(nil, "")
	w.suggestedName = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)) + ".fountain"
	w.hasChanges = true
	if pageLockOf(text) != nil {
		w.setStatus("Imported from " + from + ", its locked pages kept: where Accolade prints a page longer, it goes on to an A-page")
	} else {
		w.setStatus("Imported from " + from)
	}
}

// importPDFAsync reads a PDF in the background (a scanned one needs OCR,
// seconds a page), showing how far it is. Before scanned pages are read,
// it asks what language they are in, showing the one it guesses.
func (w *MainWindow) importPDFAsync(filePath string) {
	// the scene headings' language: the file name's (script.ru.pdf), else
	// the one its headings are in (read with every language's, not the
	// replaced script's)
	lang := languageOf(filePath)
	if scanned, pages, err := pdfin.ScannedPages(filePath); err == nil && scanned > 0 {
		if langs, tesseract := ocrLanguages(); len(langs) > 0 {
			w.askOCRLanguage(filePath, scanned, pages, w.guessOCRLanguage(lang, langs), langs, tesseract, func(chosen string) {
				w.readPDF(filePath, chosen, chosen)
			})
			return
		}
	}
	w.readPDF(filePath, lang, "")
}

// readPDF reads a PDF in the background, its scene headings in lang ("":
// whichever they are in), scanned pages read in ocrLang.
func (w *MainWindow) readPDF(filePath, lang, ocrLang string) {
	scenes := sceneHeaders(lang)
	if lang == "" {
		scenes = allSceneHeaders()
	}
	read := func(ctx context.Context, progress func(done, pages int)) (string, []string, error) {
		return importPDF(ctx, filePath, scenes, ocrLang, progress)
	}
	w.readPDFInBackground(filePath, "the screenplay", read, func(text string) {
		named := lang != "" // the file name says it (script.ru.pdf): that first
		if !named {
			lang = headingsLanguage(text, w.settings.GetString("script-language"))
		}
		w.setLanguage(lang)
		w.showImported(filePath, text, "a PDF: check the result, a PDF only shows how the script looked")
		if !named && lang != "en" { // saved, it keeps the language its headings are in
			w.suggestedName = strings.TrimSuffix(w.suggestedName, ".fountain") + "." + lang + ".fountain"
			w.updateTitle()
		}
	})
}

// readPDFInBackground reads a PDF with read in the background, showing
// how far it is with a Cancel button (a long scanned one takes minutes),
// then gives done what it read and tells which pages were left out.
// what is what is put together at the end ("the screenplay").
func (w *MainWindow) readPDFInBackground(filePath, what string,
	read func(ctx context.Context, progress func(done, pages int)) (string, []string, error), done func(text string)) {
	bar := widget.NewProgressBar()
	label := widget.NewLabel("Reading " + filepath.Base(filePath) + "...")
	ctx, cancel := context.WithCancel(context.Background())
	cancelBtn := widget.NewButton("Cancel", cancel)
	d := dialog.NewCustomWithoutButtons("Importing PDF", container.NewVBox(label, bar, container.NewCenter(cancelBtn)), w.fyneWindow)
	d.SetOnEscape(cancel)
	d.Show()
	finished := make(chan struct{})
	w.imported = finished
	go func() {
		defer close(finished)
		defer cancel()
		text, skipped, err := read(ctx, func(n, pages int) {
			pace.YieldNow() // the browser's page draws and takes input meanwhile (internal/pace)
			uiDo(func() {
				bar.SetValue(float64(n) / float64(max(pages, 1)))
				label.SetText(fmt.Sprintf("Read %d of %d pages", n, pages))
				if n == pages {
					label.SetText("Read all pages; putting " + what + " together...")
				}
			})
		})
		uiDo(func() {
			d.Hide()
			if ctx.Err() != nil && errors.Is(err, context.Canceled) {
				return // cancelled: nothing to show
			}
			if err != nil {
				dialog.ShowError(fmt.Errorf("could not import %s: %v", filepath.Base(filePath), err), w.fyneWindow)
				return
			}
			done(text)
			if len(skipped) > 0 {
				dialog.ShowInformation("Some pages were left out",
					"These pages could not be read:\n"+strings.Join(skipped, "\n"), w.fyneWindow)
			}
		})
	}()
}

func (w *MainWindow) SaveFile() error {
	w.relinkAfterEdit(w.srcDoc)           // the source edited: its links looked at again (relink.go)
	if err := w.storeDocs(); err != nil { // the source in the main editor, with the script (document.go)
		return err
	}
	if w.currentFile == "" {
		return w.SaveFileAs()
	}
	if err := w.saveToFile(w.currentFile); err != nil {
		return err
	}
	if inBrowser { // kept in the browser, and a copy to download
		p := w.currentFile
		w.afterBundleWrites(func() { offerDownload(p) })
	}
	return nil
}

func (w *MainWindow) SaveFileAs() error {
	// the type first, then where (saveas_dialog.go); the browser asks
	// the name there, for the download
	w.showSaveAs()
	return nil
}

func (w *MainWindow) saveToFile(filePath string) error {
	main, native, hasNative := w.scriptTexts(filePath, w.textEditor.Text())
	data, utf8Instead := encodeScript(main, w.format)
	var nativeData []byte
	if hasNative {
		var also bool
		nativeData, also = encodeScript(native, w.format)
		utf8Instead = utf8Instead || also
	}
	if err := w.writeScript(filePath, data, nativeData); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}
	w.keepLinkBase(filePath) // the source its links were looked at against, for its next opening (relink.go)
	if utf8Instead {
		w.format.Windows1252 = false
		w.setStatus("Saved as UTF-8: Windows-1252 cannot hold every character now in the script")
	}

	w.cancelAutoSave()
	w.hasChanges = false
	w.updateTitle()
	return nil
}

// writeScript writes the script's text to filePath: into the bundle it
// came from or a new one for a bundle's name (bundle.go), else as a plain
// file (a bundle saved under a plain name leaves the bundle, its source
// kept by Accolade).
func (w *MainWindow) writeScript(filePath string, data, native []byte) error {
	if w.writesBundle(filePath) {
		if inBrowser && bundleKindOf(filePath) == bundleFolder {
			return fmt.Errorf("a .textbundle is a folder, which the browser cannot save: use .textpack")
		}
		markdown, gen, lang, noCopy := w.markdown, w.bundleGen, w.language, !w.keepsFountainCopy(filePath)
		conf, format := w.lex(), w.scriptFormatKey()
		carry := carryFor(conf, lang, format)
		var extra []bundleFile
		if w.bundle == nil { // a plain script made a bundle: its extras carried in
			var err error
			if extra, err = w.extrasHome().all(); err != nil {
				return fmt.Errorf("could not read the drafts to carry in: %v", err)
			}
		}
		var source []byte
		if w.bundle == nil { // and its source (relocate.go)
			source = w.carrySource
		}
		warn := w.warnsHighland(filePath, extra) || (source != nil && w.warnsHighland(filePath, []bundleFile{{bundleSourceName, source}}))
		w.plainBundle = ""
		// in the background (bundle_queue.go): an error makes the script
		// unsaved again
		w.queueBundle(bundleJob{
			run: func(b *textBundle) (*textBundle, error) {
				return saveBundleText(b, filePath, bundleSave{text: data, markdown: markdown, extra: extra, source: source,
					highlandWarned: warn, native: native, lang: lang, noCopy: noCopy, conf: conf, carry: carry, format: format})
			},
			done: func(err error) {
				if err == nil && warn {
					w.highlandWarning(filePath)
				}
				if err != nil && w.bundleGen == gen {
					w.saveFailed(filePath, err)
				} else if err != nil {
					w.setStatus("Could not save " + filepath.Base(filePath) + ": " + err.Error())
				}
			},
		})
		return nil
	}
	if err := writeFileAtomic(filePath, data); err != nil {
		return err
	}
	if b := w.bundle; b != nil {
		entry, src := w.bundleSource()
		how := w.flattenTo
		w.flattenTo = flattenKept
		w.setBundle(nil)
		switch {
		case entry == "":
		case w.sourceBeside != "": // beside it, with the drafts' folder (relocate.go)
			if err := writeFileAtomic(w.sourceBeside, src); err != nil {
				return fmt.Errorf("the script is saved, but its source could not be put beside it: %v", err)
			}
		case w.chosenSource(filePath) == "":
			if kept, err := keepSourceCopy(w.sourceName(sourceRef{}), string(src)); err == nil {
				w.rememberSource(filePath, kept)
			}
		}
		if err := w.placeFlattened(b, filePath, how); err != nil {
			return fmt.Errorf("the script is saved, but its drafts could not be kept: %v", err)
		}
	}
	if bundleKindOf(filePath) != notBundle {
		w.plainBundle = filePath
	}
	return nil
}

func (w *MainWindow) OpenFile() {
	if inBrowser {
		pickFile(openExtensions(), w.openPath)
		return
	}
	fileDialog := dialog.NewFileOpen(func(closer fyne.URIReadCloser, err error) {
		if err != nil || closer == nil {
			return
		}

		filePath := closer.URI().Path()
		defer closer.Close()
		w.openPath(filePath)
	}, w.fyneWindow)

	// what Accolade opens: scripts, Markdown, Final Draft, PDF
	fileDialog.SetFilter(storage.NewExtensionFileFilter(openExtensions()))

	fileDialog.Show()
}

// openPath opens a file (a script, a Final Draft document, a PDF): in
// this window if it holds only what it was filled with (the sample
// script, an import not edited since), else in a new one, so that
// nothing unsaved is replaced.
func (w *MainWindow) openPath(path string) {
	if isTOMLFile(path) { // its editor (toml_open.go); the script stays
		w.openTOMLFrom(path)
		return
	}
	w.withTarget(func(target *MainWindow) {
		if err := target.LoadFile(path); err != nil {
			dialog.ShowError(fmt.Errorf("failed to open %s: %v", filepath.Base(path), err), target.fyneWindow)
		}
	})
}

// withTarget gives the window something new goes in: this one if it
// holds nothing of its own (no file, no edits beyond what it was filled
// with), else a new window; in the browser, which has one window, this
// one after asking.
func (w *MainWindow) withTarget(then func(*MainWindow)) {
	// a shared script is never replaceable: it is the peers' too
	replaceable := w.currentFile == "" && (!w.hasChanges || w.textEditor.Text() == w.filledWith) && w.shared == nil
	title, msg := w.replaceQuestion()
	switch {
	case replaceable || w.app == nil:
		then(w)
	case inBrowser:
		dialog.ShowConfirm(title, msg,
			func(ok bool) {
				if ok {
					then(w)
				}
			}, w.fyneWindow)
	default:
		then(w.app.newWindow())
	}
}

// replaceQuestion is what the browser asks before something else takes
// the window's script's place.
func (w *MainWindow) replaceQuestion() (title, msg string) {
	if w.shared != nil {
		return "Leave the Shared Script?", "You edit this script together with others. Open the other one here instead? " +
			"The shared script stays with them, as it is; its link brings you back to it."
	}
	return "Replace the Script?", "The script here has changes not saved. Replace it? (Save first to keep it.)"
}

// onDropped opens the files dropped on the window, as Highland does.
func (w *MainWindow) onDropped(_ fyne.Position, uris []fyne.URI) {
	for _, u := range uris {
		if u.Scheme() == "file" {
			w.openPath(u.Path())
		}
	}
}

func (w *MainWindow) NewFile() { w.newFile(false) }

// NewEmptyFile starts a script with nothing in it, not even the sample
// (Ctrl+Alt+N, or Alt and the New button).
func (w *MainWindow) NewEmptyFile() { w.newFile(true) }

// newFile starts a new script, the sample one or an empty one, asking
// first if the script has unsaved changes.
func (w *MainWindow) newFile(empty bool) {
	if w.hasChanges {
		dialog.ShowConfirm(
			"Unsaved Changes",
			"You have unsaved changes. Create a new file anyway?",
			func(confirmed bool) {
				if confirmed {
					w.createNewFile(empty)
				}
			},
			w.fyneWindow,
		)
	} else {
		w.createNewFile(empty)
	}
}

// editorKeys is how the editor's keys work: "standard", "vim" or
// "helix" (an older "vim-mode" setting is vim).
func (w *MainWindow) editorKeys() string {
	if w == nil || w.settings == nil {
		return "standard"
	}
	switch k := w.settings.GetString("editor-keys"); k {
	case "vim", "helix":
		return k
	}
	if w.settings.GetBoolean("vim-mode") {
		return "vim"
	}
	return "standard"
}

// setLanguage makes the script's language lang ("" for the one in the
// preferences): its scene headings in the editor and exports.
func (w *MainWindow) setLanguage(lang string) {
	if lang == "" && w.settings != nil {
		lang = w.settings.GetString("script-language")
	}
	conf := w.lex()
	if !conf.knows(lang) {
		lang = "en"
	}
	w.language = lang
	if w.headerBar != nil && w.headerBar.place != nil {
		// the place button in the language: BIN./BUI. in Dutch
		if _, _, both, ok := placeWords(conf.headers(lang)); ok {
			w.headerBar.place.SetText(both)
		}
	}
	w.ribbonRefresh() // its place button too
	if w.textEditor != nil {
		w.textEditor.SceneStarts = conf.starts(lang)
		w.textEditor.Dialect = fountain.NewDialect(conf.Language(lang))
		w.textEditor.LanguageRTL = conf.RTL(lang) // for a line with no direction of its own
		w.textEditor.Refresh()
	}
	w.shareMeta("language", lang)
}

// sceneStarts are what scene headings start with in the script's
// language (English without a window).
func (w *MainWindow) sceneStarts() []string {
	if w == nil {
		return nil
	}
	return w.lex().starts(w.language)
}

// newScriptTemplate is what a new script starts with: a title page and
// a scene to write over.
func newScriptTemplate() string {
	return strings.Join([]string{
		"Title: UNTITLED SCREENPLAY",
		"Credit: Written by",
		"Author: Your Name",
		"Draft date: " + time.Now().Format("January 2, 2006"),
		"Contact:",
		"    Your Name",
		"    Your Address",
		"    Your Phone",
		"    Your Email",
		"",
		"FADE IN:",
		"",
		"INT. LIVING ROOM - DAY",
		"",
		"A simple room with basic furniture. Light streams through the windows.",
		"",
		"JOHN sits at a desk, typing on a laptop. He looks frustrated.",
		"",
		"JOHN",
		"(sighing)",
		"This screenplay isn't writing itself.",
		"",
		"He takes a sip of coffee and continues typing.",
		"",
		"> FADE OUT.",
		"",
	}, "\n")
}

func (w *MainWindow) createNewFile(empty bool) {
	// the views follow at the end (state.go)
	defer w.scriptChange()()
	w.newDocument()
	w.forgetUnsavedSource()
	w.setMarkdown(false)
	w.format = fileFormat{} // a new script is UTF-8
	text := newScriptTemplate()
	if empty {
		text = ""
	}
	w.textEditor.SetText(text)
	w.currentFile, w.plainBundle = "", ""
	w.looseUnsaved = ""
	w.hideLooseNotice()
	w.setBundle(nil)
	w.setCarried(nil, "")
	w.hasChanges = false
	w.roleChanged(false) // the preferences' role again (role.go)
}

// statusRow lays out the status bar: the Tab chip of a hint, if shown;
// the message as wide as it is, or wrapped if it is wider than the bar
// (a phone's); the statistics in what is left, cut to fit. A long
// message never makes the window wider.
type statusRow struct{ width float32 } // the width laid out last

// statsMin is how wide the statistics are at least to be shown.
const statsMin = 60

func (*statusRow) natural(l *widget.Label) float32 {
	if l.Text == "" {
		return 0
	}
	return fyne.MeasureText(l.Text, theme.CaptionTextSize(), l.TextStyle).Width + 2*theme.InnerPadding() + 1
}

// fit gives the message its width in a bar so wide, and its height for it.
func (r *statusRow) fit(msg *widget.Label, width float32) float32 {
	mw := min(r.natural(msg), width)
	msg.Resize(fyne.NewSize(mw, msg.Size().Height))
	msg.Resize(fyne.NewSize(mw, msg.MinSize().Height))
	return mw
}

// chipWidth is how wide the chip is, 0 hidden.
func chipWidth(chip fyne.CanvasObject) float32 {
	if !chip.Visible() {
		return 0
	}
	return chip.MinSize().Width
}

func (r *statusRow) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	chip, msg, stats := objs[0], objs[1].(*widget.Label), objs[2]
	r.width = size.Width
	cw := chipWidth(chip)
	chip.Move(fyne.NewPos(0, 0))
	chip.Resize(chip.MinSize())
	mw := r.fit(msg, max(size.Width-cw, 0))
	msg.Move(fyne.NewPos(cw, 0))
	mw += cw
	if left := size.Width - mw; left >= statsMin || msg.Text == "" {
		stats.Show()
		stats.Move(fyne.NewPos(mw, 0))
		stats.Resize(fyne.NewSize(left, stats.MinSize().Height))
	} else {
		stats.Hide() // the message first
	}
}

func (r *statusRow) MinSize(objs []fyne.CanvasObject) fyne.Size {
	chip, msg, stats := objs[0], objs[1].(*widget.Label), objs[2]
	h := stats.MinSize().Height
	if cw := chipWidth(chip); cw > 0 {
		h = max(h, chip.MinSize().Height)
	}
	if msg.Text != "" {
		if r.width > 0 {
			r.fit(msg, max(r.width-chipWidth(chip), 0))
		}
		h = max(h, msg.Size().Height)
	}
	return fyne.NewSize(statsMin, h)
}

// UI operations
func (w *MainWindow) Show() {
	w.fyneWindow.Show()
}

func (w *MainWindow) IsEmpty() bool {
	return strings.TrimSpace(w.textEditor.Text()) == ""
}

func (w *MainWindow) HasUnsavedChanges() bool {
	return w.hasChanges
}

// kept is the script in a window as the web version keeps it, for after
// a reload (browser_js.go).
type kept struct {
	Text     string `json:"text"`
	Language string `json:"language,omitempty"`
	File     string `json:"file,omitempty"` // in the browser's memory (kept too)
	Name     string `json:"name,omitempty"` // an import's name to save it as
	Markdown bool   `json:"markdown,omitempty"`
	Source   string `json:"source,omitempty"` // the source of a script not saved yet (its kept copy)
	Extras   string `json:"extras,omitempty"` // its drafts (their kept copy: extras.go)
}

// keepDraft keeps the window's script in the browser, if it is the
// writer's own (keepsOwnDraft): a joined one is kept as the session's
// state only (keepSharedState), so joining never overwrites the script
// kept before.
func (w *MainWindow) keepDraft() {
	if keepsOwnDraft(w.shared) {
		keepDraft(w.draft())
	}
}

// draft is the window's script as the web version keeps it.
func (w *MainWindow) draft() kept {
	d := kept{Text: w.textEditor.Text(), Language: w.language, File: w.currentFile, Name: w.suggestedName, Markdown: w.markdown}
	if w.currentFile == "" {
		d.Source, d.Extras = w.unsavedSource, w.unsavedExtras
	}
	return d
}

// restore puts a kept script back: its file again, if it still has it,
// with the changes not saved.
func (w *MainWindow) restore(d kept) {
	// the views follow at the end (state.go)
	defer w.scriptChange()()
	w.newDocument()
	if d.File != "" && w.LoadFile(d.File) == nil {
		if d.Text != w.textEditor.Text() {
			w.textEditor.SetText(d.Text)
			w.checkLooseReading()
		}
		w.setStatus("Your script, as this browser kept it")
		return
	}
	w.setLanguage(d.Language)
	w.setMarkdown(d.Markdown)
	w.textEditor.SetText(d.Text)
	w.checkLooseReading() // (asked again after a reload: a script not saved keeps no answer)
	w.suggestedName = d.Name
	w.unsavedSource = ""
	if _, err := os.Stat(d.Source); d.Source != "" && err == nil {
		w.unsavedSource = d.Source
	}
	w.unsavedExtras = ""
	if _, err := os.Stat(d.Extras); d.Extras != "" && err == nil {
		w.unsavedExtras = d.Extras
	}
	w.updateTitle()
	w.setStatus("Your script not saved yet, as this browser kept it")
}

// draftPause is how long typing pauses before the browser keeps the
// script (browser_js.go's keepDraft).
const draftPause = time.Second

func (w *MainWindow) onTextChanged(text string) {
	w.hasChanges = true
	if inBrowser && keepsOwnDraft(w.shared) { // kept when typing pauses: the whole script is written each time
		w.debounce(&w.draftTimer, draftPause, func() {
			w.keepDraft()
		})
	}
	w.shareEdit(text)
	w.updateTitle()
	own := w.docWriting() // a document in the main editor wrote this: it says its own status (mainChanged)
	if !own {
		w.updateStatusBar()
		w.updateCurrentElement()
	}
	w.scheduleSceneNumbers()
	w.scheduleRevisionMarks()
	w.scheduleBreakdown()
	w.dock.edited(text) // the panes in the dock (dock.go)
	if w.cardsShown() {
		w.cards.update(text)
	}
	if w.stripsVisible && w.stripboard != nil {
		w.stripboard.update()
	}
	w.followSource()
	w.docsFollowScript() // the notes or source kept in it, open as documents (document.go)
	if w.scriptInMain() {
		if w.previewVisible && w.previewPane != nil {
			w.updateLivePreview()
		}
		if w.searchBar != nil && w.searchBar.IsVisible() {
			w.searchBar.updateSearch() // the matches move with the text
		}
	}

	w.followText(text) // the preview's pages (pagefollow.go)
	w.ribbonFollow()   // a lock or a source block changes what it can give
	w.scheduleAutoSave()
}

func (w *MainWindow) updateTitle() {
	w.titleOf = w.frontKey()
	title := "Accolade"

	if w.currentFile != "" {
		filename := filepath.Base(w.currentFile)
		if w.hasChanges {
			title = "• " + filename + " - Accolade"
		} else {
			title = filename + " - Accolade"
		}
	} else if w.suggestedName != "" {
		title = "• " + w.suggestedName + " (not saved yet) - Accolade"
	} else if w.hasChanges {
		title = "• Untitled - Accolade"
	}
	if d := w.midDoc(); d != nil { // the document in the main editor first (middle.go)
		mark := ""
		if d.dirty() {
			mark = "• "
		}
		title = mark + d.title() + " — " + title
	}

	if w.midDoc() == nil && w.unsavedDocs() && !strings.HasPrefix(title, "• ") { // a document behind another tab, not written yet
		title = "• " + title
	}
	w.fyneWindow.SetTitle(title)
	if w.headerBar != nil {
		name := strings.TrimSuffix(title, " - Accolade")
		if name == "Accolade" {
			name = "Untitled"
		}
		w.headerBar.SetTitle(name)
		if w.ribbon != nil {
			w.ribbon.title.SetText(name)
		}
	}
}

func (w *MainWindow) updateStatusBar() {
	if w.statsLabel == nil {
		return // called while the window is being built
	}
	w.statusOf = w.frontKey()
	var text string
	if d := w.main(); d != nil && d.kind != docScript { // the document in the main editor (middle.go)
		text = w.markdownStats(d.Text()) + " · " + d.kind.String() + ", " + d.where()
		if w.shared != nil && d.sharedAs() == "" {
			text += " · yours only, not shared"
		}
	} else if w.markdown {
		text = w.markdownStats(w.textEditor.Text())
	} else {
		text = computeStats(w.textEditor.Text(), w.textEditor.CursorOffset(), w.sceneStarts()).String()
	}
	if w.currentElement != "" && w.currentElement != "empty" {
		text = w.currentElement + " · " + text
	}
	if s := w.sharingStatus(); s != "" {
		text = s + " · " + text
	}
	w.statsLabel.SetText(text)
}

func (w *MainWindow) onCursorChanged() {
	if w.docWriting() { // a document kept in the script wrote there, the cursor kept (document.go)
		w.followSource()
		return
	}
	w.flowCursorMoved()
	if w.noteTipShown != "" && !w.markdown { // a chip's tip (a long press's) goes with a move
		w.showEditorTip("", fyne.Position{}, fyne.Size{})
	}
	w.updateCurrentElement()
	w.breakdownFollowCursor()
	w.scheduleSceneNumbers() // the heading just left
	w.updateStatusBar()      // the scene under the cursor
	w.followSource()         // the scene's passages
	if w.cardsDock != nil && w.paneInSight("cards") {
		w.cardsDock.follow() // the scene's card
	}
	// what Tab would complete or take from the source, or nothing once
	// it no longer applies
	w.showSuggestionHint()
}

func (w *MainWindow) updateCurrentElement() {
	e := w.mainEditor()
	if e == nil { // a view in the middle (middle.go)
		w.currentElement = ""
		return
	}
	text := e.Text()
	if text == "" {
		w.currentElement = "empty"
		return
	}

	line, _ := e.Buffer().Position(e.CursorOffset())
	if w.editorMarkdown() {
		kinds, ok := e.MarkdownKinds() // as the editor laid them out
		if !ok {
			kinds = syntax.ClassifyMarkdown(strings.Split(text, "\n"))
		}
		w.currentElement = markdownNames[kinds[min(line, len(kinds)-1)]]
		return
	}
	kinds, ok := w.textEditor.TypingKinds() // as the editor laid them out
	if !ok || !slices.Equal(w.textEditor.SceneStarts, w.sceneStarts()) || w.textEditor.Dialect != w.dialect() {
		kinds = syntax.ClassifyTypingIn(strings.Split(text, "\n"), w.sceneStarts(), w.dialect(), line)
	}
	w.currentElement = elementNames[kinds[min(line, len(kinds)-1)]]
	if w.cueHere() { // Tab made it a cue (keyflow.go)
		w.currentElement = elementNames[syntax.Character]
	}
	if w.headerBar != nil {
		w.headerBar.showElement(w.currentElement)
		if w.ribbon != nil {
			w.ribbon.showElement(w.currentElement)
		}
	}
}

// elementNames are the elements' names in the status bar.
var elementNames = map[syntax.Kind]string{
	syntax.Action: "Action", syntax.Empty: "Empty", syntax.TitlePage: "Title Page",
	syntax.SceneHeading: "Scene Heading", syntax.Character: "Character",
	syntax.Parenthetical: "Parenthetical", syntax.Dialogue: "Dialogue",
	syntax.Transition: "Transition", syntax.Centered: "Centered", syntax.Section: "Section",
	syntax.Synopsis: "Synopsis", syntax.Note: "Note", syntax.PageBreak: "Page Break",
	syntax.Lyrics: "Lyrics",
}

// previewPDF prints the script as the PDF export does, with the page
// setup, and shows its pages in the page view, with the page setup
// they are made with: a change in it makes them again.
func (w *MainWindow) previewPDF() {
	w.pdfOrPrint(w.textEditor.Text(), w.makePreviewPDF)
}

func (w *MainWindow) makePreviewPDF() {
	dir, err := previewTemp()
	if err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return
	}
	name := w.pdfName()
	out := filepath.Join(dir, name)
	content := w.textEditor.Text()
	job := w.outputJob(w.pageSetup(), "pdf", content, nil)
	job.lines = true // the pages' lines, to follow the edits
	line := w.previewLine()
	w.setStatus("Preparing the PDF...")
	w.previewGen++
	gen := w.previewGen
	w.pagesWant, w.pagesComing = w.scriptKey(), gen // the pages describe the script (state.go)
	previewsMaking.Add(1)
	go func() {
		defer previewsMaking.Add(-1)
		d, note, err := drawScript(content, out, job)
		uiDo(func() { // the UI's turn: in the tests this runs beside them
			if gen != w.previewGen { // made again since
				os.RemoveAll(dir)
				return
			}
			w.pagesComing = 0
			if err != nil {
				os.RemoveAll(dir)
				w.setStatus("")
				w.showPDFError(err)
				return
			}
			w.setStatus("PDF preview shown")
			v := w.showPages(d, "Preview: "+name, out, name, line.box)
			w.pagesPreview = true
			v.remake = func() {
				line.refresh()
				w.pdfOrPrint(w.textEditor.Text(), w.makePreviewPDF)
			}
			v.setNote(note)
			v.setupLine = line
			v.followLook()
			newPageFollow(w, v, content, d)
			if text := w.textEditor.Text(); text != content { // typed meanwhile
				v.follow.edited(text)
			}
			w.ribbonRefresh() // its Setup, Refresh
			pagesShown(v)
		})
	}()
}

// printFile prints an HTML file: in the browser's print dialog (the web
// version), or opened in the system's browser to print from there.
// previewsMaking counts Preview PDFs being made in the background; tests
// wait for none before the next test starts.
var previewsMaking atomic.Int32

var printFile = func(path string) { openFile(path) }

// printScript prints the script as Lexington's HTML lays it out, through
// the browser: its fonts have every script, and its print dialog saves a
// PDF too.
func (w *MainWindow) printScript() {
	dir, err := os.MkdirTemp("", "accolade-print-")
	if err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return
	}
	out := filepath.Join(dir, "script.html")
	content := w.textEditor.Text()
	job := w.outputJob(w.pageSetup(), "html", content, nil)
	w.setStatus("Preparing to print...")
	inBackground(func() error { return exportAs(content, out, "html", job) }, func(err error) {
		if err != nil {
			dialog.ShowError(fmt.Errorf("could not prepare the script to print: %v", err), w.fyneWindow)
			return
		}
		w.setStatus("Print: choose a printer, or Save as PDF")
		printFile(out)
	})
}

// openFile opens a file in the system's viewer for that kind of file.
var openFile = func(path string) { systemOpen(path) }

// systemOpen opens a file in the system's viewer for its kind.
func systemOpen(path string) {
	if u, err := url.Parse("file://" + filepath.ToSlash(path)); err == nil {
		fyne.CurrentApp().OpenURL(u)
	}
}

// setContent shows content in the window, with the tooltip layer over it.
func (w *MainWindow) setContent(content fyne.CanvasObject) {
	w.fyneWindow.SetContent(container.New(&windowLayout{w: w}, content, w.exPopup().layer, w.tooltips))
}

// relayout puts the window together: the toolbar and the status bar,
// the outline and notes on the left if shown, the editor.
func (w *MainWindow) relayout() {
	for _, v := range []*pageView{w.pages, w.mdPagesView()} { // their bars: the Menus look's
		if v != nil {
			v.followLook()
		}
	}
	var center fyne.CanvasObject = w.editorView
	if w.stripsVisible && w.stripboard != nil {
		center = w.stripboard.box
	} else if !w.scriptInMain() {
		// another middle tab in front: w.mid.center below puts it there
	} else if w.pagesVisible && w.pages != nil && w.narrow { // a phone: in the script's place
		center = w.pages
	} else if w.pagesVisible && w.pages != nil {
		split := container.NewHSplit(w.editorView, w.pages)
		split.Offset = 0.5
		center = split
	} else if w.previewVisible && w.previewPane != nil {
		split := container.NewHSplit(w.editorView, w.previewPane.box)
		split.Offset = 0.55
		center = split
	}
	if !w.stripsVisible || w.stripboard == nil {
		center = w.mid.center(center) // the tab in front, under the middle's tabs
	}
	side := w.dock.object() // dock.go
	var left fyne.CanvasObject
	if side != nil && w.narrow {
		// a phone: the panes under the script, both in sight, the split
		// where it was (a pane beside the script left it no room); the
		// sheet's row has a button making it taller (dock.go)
		offset := paneHalf
		if w.paneSplit != nil {
			offset = w.paneSplit.Offset
		}
		w.paneSplit = container.NewVSplit(center, narrowFit(side))
		w.paneSplit.Offset = offset
		w.showPaneSize()
		center = w.paneSplit
	} else if side != nil {
		// beside the script, in a split the writer can drag (Joop: the
		// side panes could not change size); the width stays across
		// relayouts and sessions (dock.keepSizes)
		offset := w.settings.GetFloat(dockWidthPref)
		if offset <= 0.05 || offset >= 0.9 {
			offset = sidePaneDefault
		}
		w.sideSplit = container.NewHSplit(side, center)
		w.sideSplit.Offset = offset
		center = w.sideSplit
	} else {
		w.sideSplit = nil
	}
	// the objects in the order they are seen, which is the order Tab and
	// screen readers go through them (NewBorder puts the centre first:
	// Tab went from the editor to the toolbar, the side panel came last)
	objects := []fyne.CanvasObject{w.toolbarContainer}
	if left != nil {
		objects = append(objects, left)
	}
	objects = append(objects, center, w.bottom)
	w.frame = container.New(layout.NewBorderLayout(w.toolbarContainer, w.bottom, left, nil), objects...)
	var content fyne.CanvasObject = w.frame
	if w.isFullscreen {
		// the way back, in sight (not only Ctrl+Shift+F)
		leave := widget.NewButtonWithIcon("Leave full screen", theme.ViewRestoreIcon(), w.toggleFullscreen)
		leave.Importance = widget.LowImportance
		// a row of its own above the content, not floating over it: floating
		// it sat on the top-right buttons of the views (Cards' and the
		// Stripboard's "Back to the Script")
		content = container.NewBorder(container.NewHBox(layout.NewSpacer(), leave), nil, nil, nil, content)
	}
	w.setContent(content)
}

// relayoutTop lays the window out again round the top (the ribbon grew
// or shrank a row: the phone's Quick tab), the script below getting the
// height left.
func (w *MainWindow) relayoutTop() {
	if w.frame != nil {
		w.toolbarContainer.Refresh()
		w.frame.Refresh()
	}
}

// sidePaneDefault is the side dock's share of a wide window at first.
const sidePaneDefault = 0.28

// The script's share of a narrow window over the panes: half, or a
// little with the panes made taller.
const (
	paneHalf = 0.5
	paneTall = 0.15
)

// showPaneSize shows what the panes' size button does next.
func (w *MainWindow) showPaneSize() {
	if w.paneSplit.Offset > paneTall+0.05 {
		w.paneTaller.Icon, w.paneTaller.Tip = theme.MoveUpIcon(), "Make the panes taller"
	} else {
		w.paneTaller.Icon, w.paneTaller.Tip = theme.MoveDownIcon(), "Make the panes half the height again"
	}
	w.paneTaller.Refresh()
}

// narrowWidth is the window's width under which the panes go under the
// script rather than beside it.
const narrowWidth = 600

// onWindowWidth lays the window out again when it turns narrow or wide.
func (w *MainWindow) onWindowWidth(width float32) {
	if narrow := width < narrowWidth; narrow != w.narrow {
		w.narrow = narrow
		if r := w.ribbon; narrow && r != nil && w.ribbonOrder == "" && !r.picked && !r.previewing {
			r.selected = "" // a phone's ribbon: the order's first tab in front (Quick in Smart: quick.go)
		}
		c := w.fyneWindow.Canvas()
		focused := c.Focused() // a new content drops the focus
		w.relayout()
		w.productionFollow() // its buttons go where the room is
		w.ribbonRefresh()    // one row of tabs, one of buttons, on a phone
		if w.stripsVisible && w.stripboard != nil {
			w.stripboard.update() // strips of two lines on a phone
		}
		if focused != nil {
			c.Focus(focused)
		}
	}
}

// windowLayout stacks the window's content and the tooltips, and tells
// onWindowWidth the width.
type windowLayout struct {
	w     *MainWindow
	width float32
}

func (l *windowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return layout.NewStackLayout().MinSize(objects)
}

func (l *windowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	layout.NewStackLayout().Layout(objects, size)
	if size.Width != l.width {
		l.width = size.Width
		l.w.onWindowWidth(size.Width)
	}
}

// narrowFit lays a pane out to the width it is given, below its least
// (the panes' widths are for beside the script; under it, the window's
// width is all there is).
func narrowFit(o fyne.CanvasObject) fyne.CanvasObject {
	return container.New(fitWidth{}, o)
}

type fitWidth struct{}

func (fitWidth) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(0, objects[0].MinSize().Height)
}

func (fitWidth) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	objects[0].Move(fyne.Position{})
	objects[0].Resize(size)
}

// Find/Replace operations
func (w *MainWindow) showFindReplace() {
	w.findVisible = true
	w.searchBar.SetReplaceMode(true)
	w.searchBar.Show()
}

func (w *MainWindow) showFind() {
	w.findVisible = true
	w.searchBar.SetReplaceMode(false)
	w.searchBar.Show()
}

func (w *MainWindow) hideFindReplace() {
	w.findVisible = false
	w.searchBar.Hide()
}

func (w *MainWindow) findNext() {
	w.searchBar.findNext()
}

func (w *MainWindow) findPrevious() {
	w.searchBar.findPrevious()
}

func (w *MainWindow) replaceOne() {
	w.searchBar.replaceOne()
}

func (w *MainWindow) replaceAll() {
	w.searchBar.replaceAll()
}

// toggleFullscreen switches the distraction-free full screen: the script
// (and the outline and notes, if open), without the menu bar, toolbar
// and status bar; a button in the corner leaves it, as Ctrl+Shift+F
// does. The menu's shortcuts keep working.
func (w *MainWindow) toggleFullscreen() {
	w.isFullscreen = !w.isFullscreen
	w.fyneWindow.SetFullScreen(w.isFullscreen)
	browserFullscreen(w.isFullscreen, func() { // the browser left it (Escape)
		if w.isFullscreen {
			w.toggleFullscreen()
		}
	})
	if w.isFullscreen {
		setVisible(w.headerArea, false)
		setVisible(w.statusBar, false)
		w.fyneWindow.SetMainMenu(nil)
	} else {
		w.applyEditorSettings() // the toolbar and status bar as set
	}
	w.refreshMenu()
	w.relayout()
	w.focusMain()
}

// refreshMenu builds the menu again (the recent files changed): in the
// menu bar, or in full screen and where the ribbon replaces the menu bar
// (menuless) only its shortcuts.
func (w *MainWindow) refreshMenu() {
	c := w.fyneWindow.Canvas()
	for _, sc := range w.menuShortcuts {
		c.RemoveShortcut(sc.shortcut)
	}
	w.menuShortcuts = nil
	w.ribbonRefresh() // what the commands can give changed too
	w.menuGone = w.menuless()
	if !w.isFullscreen && !w.menuGone {
		w.fyneWindow.SetMainMenu(w.buildMainMenu())
		return
	}
	w.fyneWindow.SetMainMenu(nil) // the ribbon's place (or full screen): its keys only
	w.menuShortcuts = map[string]menuShortcut{}
	var add func(items []*fyne.MenuItem)
	add = func(items []*fyne.MenuItem) {
		for _, item := range items {
			if item.ChildMenu != nil {
				add(item.ChildMenu.Items)
			}
			if item.Shortcut == nil || item.Action == nil {
				continue
			}
			action := item.Action
			c.AddShortcut(item.Shortcut, func(fyne.Shortcut) { action() })
			w.menuShortcuts[item.Shortcut.ShortcutName()] = menuShortcut{item.Shortcut, action}
		}
	}
	for _, m := range w.buildMainMenu().Items {
		add(m.Items)
	}
}

// menuShortcut is a menu item's shortcut and action.
type menuShortcut struct {
	shortcut fyne.Shortcut
	action   func()
}

// menuShortcut runs the menu item with shortcut s in full screen (the
// editor gets shortcuts before the canvas does); it reports whether it did.
func (w *MainWindow) menuShortcut(s fyne.Shortcut) bool {
	if sc, ok := w.menuShortcuts[s.ShortcutName()]; ok {
		sc.action()
		return true
	}
	return false
}

// Export functionality
func (w *MainWindow) showExportDialog() {
	if w.markdown { // a document's formats, not a screenplay's
		newMarkdownExportDialog(w).Show()
		return
	}
	dialog := NewExportDialog(w)
	dialog.Show()
}

// Title page functionality
func (w *MainWindow) showTitlePageDialog() {
	dialog := NewTitlePageDialog(w)
	dialog.Show()
}

// Helper functions
func countWords(text string) int {
	if text == "" {
		return 0
	}
	words := strings.Fields(text)
	return len(words)
}

func isCharacterName(line string) bool {
	// Character names are typically ALL CAPS and don't start with common scene indicators
	if line == strings.ToUpper(line) && !strings.HasPrefix(line, "INT.") && !strings.HasPrefix(line, "EXT.") {
		// Additional checks to avoid false positives
		if !strings.Contains(line, " TO ") && !strings.HasSuffix(line, ":") {
			return len(strings.TrimSpace(line)) > 0 && len(strings.TrimSpace(line)) < 50
		}
	}
	return false
}

// defaultScript is the text a new window starts with: a title page to fill
// in and the first lines of a script.
func defaultScript(now time.Time) string {
	return strings.Join([]string{
		"Title: UNTITLED SCREENPLAY",
		"Credit: Written by",
		"Author: Your Name",
		"Draft date: " + now.Format("January 2, 2006"),
		"Contact:",
		"    Your Name",
		"    Your Address",
		"    Your Phone",
		"    Your Email",
		"",
		"FADE IN:",
		"",
		"INT. LIVING ROOM - DAY",
		"",
		"A simple room with basic furniture. Light streams through the windows.",
		"",
		"JOHN sits at a desk, typing on a laptop. He looks frustrated.",
		"",
		"JOHN",
		"(sighing)",
		"This screenplay isn't writing itself.",
		"",
		"He takes a sip of coffee and continues typing.",
		"",
		"> FADE OUT.",
	}, "\n")
}

// panes are the parts of the window that take the keyboard, in order:
// the editor, each dock slot's pane in front, and the search bar when shown.
func (w *MainWindow) panes() []fyne.Focusable {
	panes := []fyne.Focusable{w.mainFocus()} // the middle's editor, view or tab (middle.go)
	// each slot's pane in front, the top's first (dock.go)
	panes = append(panes, w.dock.focusTargets()...)
	if w.searchBar != nil && w.searchBar.IsVisible() {
		panes = append(panes, w.searchBar.searchEntry)
	}
	return panes
}

// cyclePane moves the focus to the pane step panes on from the one that
// has it (from anywhere else: the editor's neighbour).
func (w *MainWindow) cyclePane(step int) {
	panes := w.panes()
	at := 0
	for i, p := range panes {
		if w.fyneWindow.Canvas().Focused() == p {
			at = i
		}
	}
	n := len(panes)
	w.fyneWindow.Canvas().Focus(panes[((at+step)%n+n)%n])
}

// isPlainBundle tells whether p is the bundle-named file of plain text
// the window has (a .highland that is no zip).
func (w *MainWindow) isPlainBundle(p string) bool {
	return w.plainBundle != "" && sameFile(p, w.plainBundle)
}

// dialogMargin is the room a dialog leaves around it on a phone, and
// takes for its own padding inside.
const dialogMargin = 32

// fitDialog is a dialog's size: as wanted, but no bigger than the window
// (a phone's), less a margin.
func fitDialog(win fyne.Window, width, height float32) fyne.Size {
	if win != nil {
		if c := win.Canvas().Size(); c.Width > 0 && c.Height > 0 {
			width = min(width, c.Width-dialogMargin)
			height = min(height, c.Height-dialogMargin)
		}
	}
	return fyne.NewSize(max(width, 100), max(height, 100))
}
