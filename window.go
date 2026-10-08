package main

import (
	"fmt"
	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/tooltip"
	"io"
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
	textEditor  *editor.ScriptEditor
	editorView  *container.ThemeOverride // the editor with its font settings
	previewArea *widget.RichText
	headerBar   *HeaderBar
	tooltips    *tooltip.Layer
	searchBar   *SearchBar

	// UI layout
	mainContainer    *container.Split
	sidePanel        *container.Split
	toolbarContainer *fyne.Container
	// Status bar
	statusBar *fyne.Container

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
	w.previewArea = widget.NewRichText()
	w.previewArea.Wrapping = fyne.TextWrapOff // lines are wrapped to the screenplay's columns already

	// Create header bar with toolbar
	w.tooltips = tooltip.NewLayer()
	w.headerBar = NewHeaderBar(w)

	// Create search bar (initially hidden)
	w.searchBar = NewSearchBar(w)

	// Create toolbar container
	w.toolbarContainer = container.NewVBox(
		w.headerBar.container,
		widget.NewSeparator(),
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

	// Create main content area - just the editor for now
	editorScroll := w.editorView

	// Create main layout
	content := container.NewBorder(
		w.toolbarContainer, // top
		w.statusBar,        // bottom
		nil,                // left
		nil,                // right
		editorScroll,       // center - just the editor
	)

	w.setContent(content)

	// Initially hide preview
	w.previewVisible = false

	// Apply initial settings
	w.applySettings()
}

func (w *MainWindow) setupShortcuts() {
	// Text editor shortcuts
	w.textEditor.OnChanged = w.onTextChanged

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
	// auto-indent is read when Enter is pressed (editor.go)
}

func (w *MainWindow) applyFontSettings() {
	w.editorView.Theme = newEditorTheme(w.settings.GetString("font-family"), w.settings.GetInt("font-size"))
	w.editorView.Refresh()
}

// File operations
func (w *MainWindow) LoadFile(filePath string) error {
	if strings.EqualFold(filepath.Ext(filePath), ".fdx") {
		return w.importFile(filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	w.textEditor.SetText(string(content))
	w.currentFile = filePath
	w.suggestedName = ""
	w.hasChanges = false
	w.updateTitle()
	w.updatePreview()
	w.updateStatusBar()

	return nil
}

// importFile opens a Final Draft document converted to Fountain. It has no
// file of its own until saved, so Save asks where to put the .fountain
// file instead of overwriting the original.
func (w *MainWindow) importFile(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}
	defer f.Close()

	text, err := importFDX(f)
	if err != nil {
		return fmt.Errorf("could not import %s: %v", filepath.Base(filePath), err)
	}

	w.cancelAutoSave()
	w.textEditor.SetText(text)
	w.currentFile = ""
	w.suggestedName = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)) + ".fountain"
	w.hasChanges = true
	w.updateTitle()
	w.updatePreview()
	w.updateStatusBar()
	w.setStatus("Imported from Final Draft")
	return nil
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
		defer closer.Close()

		content := w.textEditor.Text()
		_, err = io.WriteString(closer, content)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to save file: %v", err), w.fyneWindow)
			return
		}

		w.currentFile = filePath
		w.suggestedName = ""
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
	content := w.textEditor.Text()
	err := writeFileAtomic(filePath, []byte(content))
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
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

		err = w.LoadFile(filePath)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to open file: %v", err), w.fyneWindow)
		}
	}, w.fyneWindow)

	// TODO: Set file filter when Fyne supports it
	// fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".fountain", ".spmd", ".txt"}))

	fileDialog.Show()
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

func (w *MainWindow) createNewFile() {
	w.textEditor.SetText("Title: UNTITLED SCREENPLAY\n\nCredit: Written by\n\nAuthor: Your Name\n\nDraft date: " + time.Now().Format("January 2, 2006") + "\n\nContact:\nYour Name\nYour Address\nYour Phone\nYour Email\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\nFADE IN:\n\nINT. LIVING ROOM - DAY\n\nA simple room with basic furniture. Light streams through the windows.\n\nJOHN sits at a desk, typing on a laptop. He looks frustrated.\n\n\t\t\tJOHN\n\t\t(sighing)\n\tThis screenplay isn't writing itself.\n\nHe takes a sip of coffee and continues typing.\n\nFADE OUT.")
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
	w.updatePreview()
	w.updateStatusBar()
	w.updateCurrentElement()

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
		w.previewArea.Segments = previewSegments(w.textEditor.Text(), w.lexParser.sceneHeaders, currentScriptFormat())
		w.previewArea.Refresh()
	}
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
	text := computeStats(w.textEditor.Text(), w.textEditor.CursorOffset()).String()
	if w.currentElement != "" && w.currentElement != "empty" {
		text = w.currentElement + " · " + text
	}
	w.statsLabel.SetText(text)
}

func (w *MainWindow) onCursorChanged() {
	w.updateCurrentElement()
	w.updateStatusBar() // the scene under the cursor
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
	// Show preview pane by creating a split container
	w.previewVisible = true
	w.updatePreview()
	w.settings.SetBoolean("preview-visible", true)

	// Create the split container with both editor and preview
	editorScroll := w.editorView
	previewScroll := container.NewScroll(w.previewArea)

	splitContainer := container.NewHSplit(
		editorScroll,
		previewScroll,
	)
	splitContainer.SetOffset(0.55) // the preview shows a whole page width

	// Update the main layout
	content := container.NewBorder(
		w.toolbarContainer, // top
		w.statusBar,        // bottom
		nil,                // left
		nil,                // right
		splitContainer,     // center - split container
	)

	w.setContent(content)
}

func (w *MainWindow) hidePreview() {
	// Hide preview pane by replacing the split container with just the editor
	w.previewVisible = false
	w.settings.SetBoolean("preview-visible", false)

	// Replace the split container with just the editor scroll
	editorScroll := w.editorView

	// Update the main layout to show only the editor
	content := container.NewBorder(
		w.toolbarContainer, // top
		w.statusBar,        // bottom
		nil,                // left
		nil,                // right
		editorScroll,       // center - just the editor
	)

	w.setContent(content)
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
