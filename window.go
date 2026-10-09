package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/tooltip"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Screenplay indentation constants (in character spaces)
// Based on industry standard margins for 12-point Courier font
const (
	ActionIndent        = 0  // Action text - left margin (1.5")
	DialogueIndent      = 25 // Dialogue - 2.5" from left margin
	ParentheticalIndent = 31 // Parentheticals - 3.1" from left margin
	CharacterIndent     = 37 // Character names - 3.7" from left margin
	TransitionIndent    = 60 // Transitions - right aligned (6.0" from left margin)

	// pageColumns is the width of the editor's centred text column: room
	// for a transition after TransitionIndent
	pageColumns = 76
)

type MainWindow struct {
	fyneWindow fyne.Window
	app        *Application

	// Core components
	textEditor   *editor.ScriptEditor
	editorView   *container.ThemeOverride // the editor with its font settings
	previewArea  *previewPane
	previewGen   int         // which rebuild of the preview is current
	previewTimer *time.Timer // the rebuild waiting for typing to pause
	headerBar    *HeaderBar
	tooltips     *tooltip.Layer
	searchBar    *SearchBar

	// UI layout
	mainContainer    *container.Split
	sidePanel        *container.Split
	toolbarContainer *fyne.Container
	// Status bar
	statusBar *fyne.Container
	// headerArea is the header bar (the toolbar) with its separator
	headerArea *fyne.Container
	// the sidebar beside the editor: the outline (View > Outline) or the
	// notes (View > Notes)
	// vim is the editor's vim mode (vim.go), on with the setting
	vim *vim
	// language is the script's language ("en", "eo", ...): its scene
	// headings (language.go)
	language string
	// format is how the script's file stores it (encoding.go)
	format fileFormat
	// imported is closed when the PDF import under way is in
	imported       chan struct{}
	outline        *outlinePanel
	outlineVisible bool
	notes          *notesPanel
	notesVisible   bool

	// File management
	currentFile  string
	hasChanges   bool
	isFullscreen bool

	// Settings
	settings *Settings

	// Preview state
	previewVisible    bool
	previewRestricted bool

	// Find/Replace
	findVisible bool

	// Name offered by Save As for an imported (converted) document
	suggestedName string

	// Auto-save (autosave.go)
	autoSaveTimer *time.Timer
	autoSaveDelay time.Duration // overrides the setting; for tests

	statusLabel *widget.Label
	statsLabel  *widget.Label

	// Lexington integration
	lexParser *LexingtonParser

	// Current cursor position for element detection
	currentElement string
}

func NewMainWindow(app *Application) *MainWindow {
	window := &MainWindow{
		fyneWindow:     app.fyneApp.NewWindow("Accolade"),
		app:            app,
		hasChanges:     false,
		isFullscreen:   false,
		previewVisible: false,
		findVisible:    false,
		lexParser:      NewLexingtonParser(),
	}

	window.settings = GetSettings() // shared with the preferences dialog
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

	// Add some initial content - start with a title page
	w.textEditor.SetText(defaultScript(time.Now()))

	// Apply Courier Prime font for screenplay formatting
	// the editor scrolls to keep the cursor in view; the theme override
	// carries the font settings
	w.editorView = container.NewThemeOverride(editor.NewPageScroll(w.textEditor, pageColumns), newEditorTheme("", 0))

	// Create preview area
	w.previewArea = newPreviewPane()

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
	w.statsLabel = small("")
	w.statusBar = container.NewVBox(
		widget.NewSeparator(),
		container.NewBorder(nil, nil, w.statusLabel, w.statsLabel),
	)

	w.relayout()

	// Initially hide preview
	w.previewVisible = false

	// Apply initial settings
	w.applySettings()
}

func (w *MainWindow) setupShortcuts() {
	// Text editor shortcuts
	w.textEditor.OnChanged = w.onTextChanged
	w.setLanguage("")                      // the preferences' until a file says otherwise
	w.fyneWindow.SetOnDropped(w.onDropped) // drop a script or a PDF to open it

	// Lines are formatted when completed with Enter (editor.go)

	// Keyboard shortcuts are attached to the main menu items (see menus.go)
	w.fyneWindow.SetMainMenu(w.buildMainMenu())
}

func (w *MainWindow) setupCallbacks() {
	// Window close callback
	w.fyneWindow.SetCloseIntercept(func() {
		if w.hasChanges {
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
	})
}

// TODO: Implement file drop handling when needed
// Fyne's drag and drop API may differ from GTK

func (w *MainWindow) applySettings() {
	// Apply theme
	w.app.setColorScheme(w.settings.GetString("theme"))

	// Apply editor settings
	w.applyEditorSettings()

	// Apply window settings
	if w.settings.GetBoolean("preview-visible") {
		w.showPreview()
	}

	// Apply font settings
	w.applyFontSettings()
}

func (w *MainWindow) applyEditorSettings() {
	// the script editor always wraps at the window width (screenplays are
	// laid out in columns)
	w.textEditor.SetLineNumbers(w.settings.GetBoolean("show-line-numbers"))
	setVisible(w.headerArea, w.settings.GetBoolean("toolbar-visible"))
	// vim mode switched on starts in normal mode, switched off types again
	if w.vim != nil {
		if on := w.editorKeys() != "standard"; on && w.vim.mode == vimInsert {
			w.vim.setMode(vimNormal)
		} else if !on && w.vim.mode != vimInsert {
			w.vim.mode = vimInsert
		}
	}
	setVisible(w.statusBar, w.settings.GetBoolean("statusbar-visible"))
	// auto-indent is read when Enter is pressed (editor.go)
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
}

// File operations
func (w *MainWindow) LoadFile(filePath string) error {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".fdx", ".pdf":
		return w.importFile(filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// as stored: Windows-1252, a byte order mark, CR LF are kept for saving
	text, format := decodeScript(content)
	w.format = format
	w.textEditor.SetText(text)
	w.currentFile = filePath
	w.suggestedName = ""
	w.rememberRecent(filePath)
	w.setLanguage(languageOf(filePath))
	w.hasChanges = false
	w.updateTitle()
	w.updatePreview()
	w.updateStatusBar()

	return nil
}

// importFile opens a Final Draft document or a PDF of a screenplay
// converted to Fountain. It has no file of its own until saved, so Save
// asks where to put the .fountain file instead of overwriting the
// original.
func (w *MainWindow) importFile(filePath string) error {
	var text, from string
	var err error
	if strings.EqualFold(filepath.Ext(filePath), ".pdf") {
		w.importPDFAsync(filePath)
		return nil
	} else {
		var f *os.File
		if f, err = os.Open(filePath); err != nil {
			return fmt.Errorf("failed to read file: %v", err)
		}
		defer f.Close()
		text, err = importFDX(f)
		from = "Final Draft"
	}
	if err != nil {
		return fmt.Errorf("could not import %s: %v", filepath.Base(filePath), err)
	}

	w.showImported(filePath, text, from)
	return nil
}

// showImported puts an imported script in the window, unsaved.
func (w *MainWindow) showImported(filePath, text, from string) {
	w.cancelAutoSave()
	w.format = fileFormat{}
	w.textEditor.SetText(text)
	w.currentFile = ""
	w.suggestedName = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)) + ".fountain"
	w.hasChanges = true
	w.updateTitle()
	w.updatePreview()
	w.updateStatusBar()
	w.setStatus("Imported from " + from)
}

// importPDFAsync reads a PDF in the background (a scanned one needs OCR,
// seconds a page), showing how far it is.
func (w *MainWindow) importPDFAsync(filePath string) {
	bar := widget.NewProgressBar()
	label := widget.NewLabel("Reading " + filepath.Base(filePath) + "...")
	// a long scanned script takes minutes to read: Cancel stops it
	ctx, cancel := context.WithCancel(context.Background())
	cancelBtn := widget.NewButton("Cancel", cancel)
	d := dialog.NewCustomWithoutButtons("Importing PDF", container.NewVBox(label, bar, container.NewCenter(cancelBtn)), w.fyneWindow)
	d.Show()
	done := make(chan struct{})
	w.imported = done
	go func() {
		defer close(done)
		defer cancel()
		text, skipped, err := importPDF(ctx, filePath, func(done, pages int) {
			fyne.Do(func() {
				bar.SetValue(float64(done) / float64(max(pages, 1)))
				label.SetText(fmt.Sprintf("Read %d of %d pages", done, pages))
				if done == pages {
					label.SetText("Read all pages; putting the screenplay together...")
				}
			})
		})
		fyne.Do(func() {
			d.Hide()
			if ctx.Err() != nil && errors.Is(err, context.Canceled) {
				return // cancelled: nothing to show
			}
			if err != nil {
				dialog.ShowError(fmt.Errorf("could not import %s: %v", filepath.Base(filePath), err), w.fyneWindow)
				return
			}
			w.showImported(filePath, text, "a PDF: check the result, a PDF only shows how the script looked")
			if len(skipped) > 0 {
				dialog.ShowInformation("Some pages were left out",
					"These pages could not be read:\n"+strings.Join(skipped, "\n"), w.fyneWindow)
			}
		})
	}()
}

func (w *MainWindow) SaveFile() error {
	if w.currentFile == "" {
		return w.SaveFileAs()
	}

	return w.saveToFile(w.currentFile)
}

func (w *MainWindow) SaveFileAs() error {
	fileDialog := dialog.NewFileSave(func(closer fyne.URIWriteCloser, err error) {
		if err != nil || closer == nil {
			return
		}

		filePath := closer.URI().Path()
		// written as Save does, safely: not through the dialog's file,
		// which truncates first
		closer.Close()
		if err := w.saveToFile(filePath); err != nil {
			dialog.ShowError(fmt.Errorf("failed to save file: %v", err), w.fyneWindow)
			return
		}

		w.currentFile = filePath
		w.suggestedName = ""
		w.rememberRecent(filePath)
		if l := languageOf(filePath); l != "" {
			w.setLanguage(l)
		}
		w.hasChanges = false
		w.updateTitle()

		dialog.ShowInformation("File Saved", fmt.Sprintf("File saved to %s", filepath.Base(filePath)), w.fyneWindow)
	}, w.fyneWindow)

	// Set default filename
	if w.currentFile != "" {
		fileDialog.SetFileName(filepath.Base(w.currentFile))
	} else if w.suggestedName != "" {
		fileDialog.SetFileName(w.suggestedName)
	} else {
		fileDialog.SetFileName("untitled.fountain")
	}

	// TODO: Set file filter when Fyne supports it
	// fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".fountain", ".spmd"}))

	fileDialog.Show()
	return nil
}

func (w *MainWindow) saveToFile(filePath string) error {
	data, utf8Instead := encodeScript(w.textEditor.Text(), w.format)
	if err := writeFileAtomic(filePath, data); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}
	if utf8Instead {
		w.format.Windows1252 = false
		w.setStatus("Saved as UTF-8: Windows-1252 cannot hold every character now in the script")
	}

	w.cancelAutoSave()
	w.hasChanges = false
	w.updateTitle()
	return nil
}

func (w *MainWindow) OpenFile() {
	fileDialog := dialog.NewFileOpen(func(closer fyne.URIReadCloser, err error) {
		if err != nil || closer == nil {
			return
		}

		filePath := closer.URI().Path()
		defer closer.Close()
		w.openPath(filePath)
	}, w.fyneWindow)

	// TODO: Set file filter when Fyne supports it
	// fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".fountain", ".spmd", ".txt"}))

	fileDialog.Show()
}

// openPath opens a file (a script, a Final Draft document, a PDF): in
// this window if it is empty and unchanged, else in a new one, so that
// nothing unsaved is replaced.
func (w *MainWindow) openPath(path string) {
	target := w
	if (w.hasChanges || w.currentFile != "") && w.app != nil {
		target = w.app.newWindow()
	}
	if err := target.LoadFile(path); err != nil {
		dialog.ShowError(fmt.Errorf("failed to open %s: %v", filepath.Base(path), err), target.fyneWindow)
	}
}

// onDropped opens the files dropped on the window, as Highland does.
func (w *MainWindow) onDropped(_ fyne.Position, uris []fyne.URI) {
	for _, u := range uris {
		if u.Scheme() == "file" {
			w.openPath(u.Path())
		}
	}
}

func (w *MainWindow) NewFile() {
	if w.hasChanges {
		dialog.ShowConfirm(
			"Unsaved Changes",
			"You have unsaved changes. Create a new file anyway?",
			func(confirmed bool) {
				if confirmed {
					w.createNewFile()
				}
			},
			w.fyneWindow,
		)
	} else {
		w.createNewFile()
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
// preferences): its scene headings in the editor, preview and exports.
func (w *MainWindow) setLanguage(lang string) {
	if lang == "" && w.settings != nil {
		lang = w.settings.GetString("script-language")
	}
	if sceneConf[lang] == nil {
		lang = "en"
	}
	w.language = lang
	if w.textEditor != nil {
		w.textEditor.SceneStarts = sceneStarts(lang)
		w.textEditor.Refresh()
	}
}

// sceneStarts are what scene headings start with in the script's
// language (English without a window).
func (w *MainWindow) sceneStarts() []string {
	if w == nil {
		return nil
	}
	return sceneStarts(w.language)
}

// newScriptTemplate is what a new script starts with: a title page and
// a scene to write over, with the indents Accolade gives each element.
func newScriptTemplate() string {
	ind := func(n int, s string) string { return strings.Repeat(" ", n) + s }
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
		ind(CharacterIndent, "JOHN"),
		ind(ParentheticalIndent, "(sighing)"),
		ind(DialogueIndent, "This screenplay isn't writing itself."),
		"",
		"He takes a sip of coffee and continues typing.",
		"",
		ind(TransitionIndent, "> FADE OUT."),
		"",
	}, "\n")
}

func (w *MainWindow) createNewFile() {
	w.format = fileFormat{} // a new script is UTF-8
	w.textEditor.SetText(newScriptTemplate())
	w.currentFile = ""
	w.hasChanges = false
	w.updateTitle()
	w.updatePreview()
	w.updateStatusBar()
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

func (w *MainWindow) onTextChanged(text string) {
	w.hasChanges = true
	w.updateTitle()
	w.schedulePreview()
	w.updateStatusBar()
	w.updateCurrentElement()
	if w.outlineVisible && w.outline != nil {
		w.outline.update(text)
	}
	if w.notesVisible && w.notes != nil {
		w.notes.update(text)
	}
	if w.searchBar != nil && w.searchBar.IsVisible() {
		w.searchBar.updateSearch() // the matches move with the text
	}

	w.scheduleAutoSave()
}

func (w *MainWindow) updateTitle() {
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

	w.fyneWindow.SetTitle(title)
	if w.headerBar != nil {
		name := strings.TrimSuffix(title, " - Accolade")
		if name == "Accolade" {
			name = "Untitled"
		}
		w.headerBar.SetTitle(name)
	}
}

func (w *MainWindow) updatePreview() {
	if w.previewVisible {
		// the screenplay as Lexington prints it (its layout package)
		w.previewGen++ // a rebuild still on its way is out of date
		w.previewArea.SetSegments(previewSegments(w.textEditor.Text(), sceneHeaders(w.language), currentScriptFormat()))
	}
}

// previewAsync is false in the tests: Fyne's test driver runs fyne.Do
// on the calling goroutine, not the UI thread.
var previewAsync = true

// schedulePreview rebuilds the preview after typing pauses, off the UI
// thread: on a long script it takes a tenth of a second or more, which
// every keystroke used to wait for.
func (w *MainWindow) schedulePreview() {
	if !w.previewVisible {
		return
	}
	if !previewAsync {
		w.updatePreview()
		return
	}
	w.previewGen++
	gen := w.previewGen
	text, headers, format := w.textEditor.Text(), sceneHeaders(w.language), currentScriptFormat()
	if w.previewTimer != nil {
		w.previewTimer.Stop()
	}
	w.previewTimer = time.AfterFunc(250*time.Millisecond, func() {
		segs := previewSegments(text, headers, format)
		fyne.Do(func() {
			if gen == w.previewGen && w.previewVisible {
				w.previewArea.SetSegments(segs)
			}
		})
	})
}

func (w *MainWindow) setStatus(text string) {
	if w.statusLabel != nil {
		w.statusLabel.SetText(text)
	}
}

func (w *MainWindow) updateStatusBar() {
	if w.statsLabel == nil {
		return // called while the window is being built
	}
	text := computeStats(w.textEditor.Text(), w.textEditor.CursorOffset(), w.sceneStarts()).String()
	if w.currentElement != "" && w.currentElement != "empty" {
		text = w.currentElement + " · " + text
	}
	w.statsLabel.SetText(text)
}

func (w *MainWindow) onCursorChanged() {
	w.updateCurrentElement()
	w.updateStatusBar() // the scene under the cursor
	// what Tab would complete, or nothing once it no longer applies
	if h := completionHint(w.textEditor.Text(), w.textEditor.CursorOffset(), w.sceneStarts()); h != "" {
		w.setStatus(h)
	} else if w.statusLabel != nil && strings.HasPrefix(w.statusLabel.Text, "Tab: ") {
		w.setStatus("")
	}
}

func (w *MainWindow) updateCurrentElement() {
	text := w.textEditor.Text()
	if text == "" {
		w.currentElement = "empty"
		return
	}

	// The logical line under the cursor (CursorRow counts wrapped rows),
	// with its indentation, which tells dialogue from action
	runes := []rune(text)
	start, end := lineBounds(runes, min(w.textEditor.CursorOffset(), len(runes)))
	elementType := w.detectElementType(string(runes[start:end]))
	w.currentElement = elementType
}

func (w *MainWindow) detectElementType(line string) string {
	if line == "" {
		return "Empty"
	}

	// Simple pattern matching for better performance
	upper := strings.ToUpper(strings.TrimSpace(line))

	// Scene headings
	if strings.HasPrefix(upper, "INT.") || strings.HasPrefix(upper, "EXT.") || strings.HasPrefix(upper, "EST.") {
		return "Scene Heading"
	}

	// Transitions (FADE IN: opens a script and stays at the left margin)
	if isTransition(upper) {
		return "Transition"
	}

	// Parentheticals
	if strings.HasPrefix(line, "(") && strings.HasSuffix(line, ")") {
		return "Parenthetical"
	}

	// Character names - all caps, not a "SOMETHING:" cue like FADE IN:
	if upper == line && len(line) > 1 && !strings.Contains(line, ".") && !strings.HasSuffix(line, ":") && hasLetter(line) {
		return "Character"
	}

	// Centered text
	if strings.HasPrefix(line, ">") && strings.HasSuffix(line, "<") {
		return "Centered"
	}

	// Check for character names first (indented at proper character position and uppercase)
	if strings.HasPrefix(line, strings.Repeat(" ", CharacterIndent)) && upper == strings.TrimSpace(line) {
		return "Character"
	}

	// Check for parentheticals (indented at proper position and wrapped in parentheses)
	if strings.HasPrefix(line, strings.Repeat(" ", ParentheticalIndent)) &&
		strings.HasPrefix(strings.TrimSpace(line), "(") &&
		strings.HasSuffix(strings.TrimSpace(line), ")") {
		return "Parenthetical"
	}

	// Check for dialogue (indented at dialogue position)
	if strings.HasPrefix(line, strings.Repeat(" ", DialogueIndent)) && !strings.HasPrefix(line, strings.Repeat(" ", ParentheticalIndent)) {
		return "Dialogue"
	}

	// Check for transitions (heavily indented, usually at transition position)
	if strings.HasPrefix(line, strings.Repeat(" ", TransitionIndent-5)) {
		return "Transition"
	}

	// Default to action
	return "Action"
}

func (w *MainWindow) getIndentationForNextElement(currentElement string) string {
	switch currentElement {
	case "Character":
		// After character name, expect dialogue
		return strings.Repeat(" ", DialogueIndent)
	case "Scene Heading":
		// After scene heading, expect action (no indent)
		return ""
	case "Dialogue":
		// After dialogue, could be parenthetical or more dialogue
		return ""
	case "Parenthetical":
		// After parenthetical, expect dialogue
		return strings.Repeat(" ", DialogueIndent)
	case "Transition":
		// After transition, expect scene heading
		return ""
	case "Action":
		// After action, could be anything
		return ""
	default:
		return ""
	}
}

// Preview operations
func (w *MainWindow) togglePreview() {
	if w.previewVisible {
		w.hidePreview()
	} else {
		w.showPreview()
	}
}

// setContent shows content in the window, with the tooltip layer over it.
func (w *MainWindow) setContent(content fyne.CanvasObject) {
	w.fyneWindow.SetContent(container.NewStack(content, w.tooltips))
}

func (w *MainWindow) showPreview() {
	w.previewVisible = true
	w.updatePreview()
	w.settings.SetBoolean("preview-visible", true)
	w.relayout()
}

func (w *MainWindow) hidePreview() {
	w.previewVisible = false
	w.settings.SetBoolean("preview-visible", false)
	w.relayout()
}

// toggleOutline shows or hides the outline beside the editor.
func (w *MainWindow) toggleOutline() {
	if w.outline == nil {
		w.outline = newOutlinePanel(w)
	}
	w.outlineVisible, w.notesVisible = !w.outlineVisible, false
	if w.outlineVisible {
		w.outline.update(w.textEditor.Text())
	}
	w.relayout()
}

// toggleNotes shows or hides the script's notes beside the editor.
func (w *MainWindow) toggleNotes() {
	if w.notes == nil {
		w.notes = newNotesPanel(w)
	}
	w.notesVisible, w.outlineVisible = !w.notesVisible, false
	if w.notesVisible {
		w.notes.update(w.textEditor.Text())
		w.relayout()
		w.fyneWindow.Canvas().Focus(w.notes.entry)
		return
	}
	w.relayout()
}

// relayout puts the window together: the toolbar and the status bar,
// the outline on the left if shown, the editor with the preview beside
// it if shown.
func (w *MainWindow) relayout() {
	var center fyne.CanvasObject = w.editorView
	if w.previewVisible {
		split := container.NewHSplit(w.editorView, w.previewArea.content)
		split.SetOffset(0.55) // the preview shows a whole page width
		center = split
	}
	var left fyne.CanvasObject
	switch {
	case w.outlineVisible && w.outline != nil:
		left = container.NewHBox(w.outline.box, widget.NewSeparator())
	case w.notesVisible && w.notes != nil:
		left = container.NewHBox(w.notes.box, widget.NewSeparator())
	}
	w.setContent(container.NewBorder(w.toolbarContainer, w.statusBar, left, nil, center))
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

// Focus mode and fullscreen
func (w *MainWindow) toggleFocusMode() {
	// TODO: Implement focus mode (hide toolbars, etc.)
	log.Println("Toggle focus mode")
}

func (w *MainWindow) toggleFullscreen() {
	if w.isFullscreen {
		w.fyneWindow.SetFullScreen(false)
		w.isFullscreen = false
	} else {
		w.fyneWindow.SetFullScreen(true)
		w.isFullscreen = true
	}
}

// Export functionality
func (w *MainWindow) showExportDialog() {
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
		strings.Repeat(" ", CharacterIndent) + "JOHN",
		strings.Repeat(" ", ParentheticalIndent) + "(sighing)",
		strings.Repeat(" ", DialogueIndent) + "This screenplay isn't writing itself.",
		"",
		"He takes a sip of coffee and continues typing.",
		"",
		strings.Repeat(" ", TransitionIndent) + "> FADE OUT.",
	}, "\n")
}
