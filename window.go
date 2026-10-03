package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// Screenplay indentation constants (in character spaces)
// Based on industry standard margins for 12-point Courier font
const (
	ActionIndent       = 0   // Action text - left margin (1.5")
	DialogueIndent     = 25  // Dialogue - 2.5" from left margin
	ParentheticalIndent = 31 // Parentheticals - 3.1" from left margin  
	CharacterIndent    = 37  // Character names - 3.7" from left margin
	TransitionIndent   = 60  // Transitions - right aligned (6.0" from left margin)
)

type MainWindow struct {
	fyneWindow   fyne.Window
	app          *Application
	
	// Core components
	textEditor   *screenplayEntry
	previewArea  *widget.RichText
	headerBar    *HeaderBar
	searchBar    *SearchBar
	
	// UI layout
	mainContainer *container.Split
	sidePanel     *container.Split
	toolbarContainer *fyne.Container
	// Status bar
	statusBar     *fyne.Container
	elementLabel  *widget.Label
	
	// File management
	currentFile   string
	hasChanges    bool
	isFullscreen  bool
	
	// Settings
	settings      *Settings
	
	// Preview state
	previewVisible bool
	previewRestricted bool
	
	// Find/Replace
	findVisible   bool
	
	// Auto-save (autosave.go)
	autoSaveTimer *time.Timer
	autoSaveDelay time.Duration // overrides the setting; for tests

	statusLabel *widget.Label
	
	// Lexington integration
	lexParser     *LexingtonParser
	
	// Current cursor position for element detection
	currentElement string
}

func NewMainWindow(app *Application) *MainWindow {
	window := &MainWindow{
		fyneWindow:      app.fyneApp.NewWindow("Accolade"),
		app:             app,
		hasChanges:      false,
		isFullscreen:    false,
		previewVisible:  false,
		findVisible:     false,
		lexParser:       NewLexingtonParser(),
	}
	
	window.settings = GetSettings() // shared with the preferences dialog
	window.setupUI()
	window.setupShortcuts()
	window.setupCallbacks()
	
	return window
}

func (w *MainWindow) setupUI() {
	// Set window properties
	w.fyneWindow.SetTitle("Accolade")
	w.fyneWindow.Resize(fyne.NewSize(1000, 600))
	w.fyneWindow.CenterOnScreen()
	
	// Create main text editor with Courier Prime font
	w.textEditor = newScreenplayEntry(w)
	w.textEditor.Wrapping = fyne.TextWrapWord
	w.textEditor.SetPlaceHolder("Start writing your screenplay here...")
	
	// Add some initial content - start with a title page
	w.textEditor.SetText("Title: UNTITLED SCREENPLAY\n\nCredit: Written by\n\nAuthor: Your Name\n\nDraft date: " + time.Now().Format("January 2, 2006") + "\n\nContact:\nYour Name\nYour Address\nYour Phone\nYour Email\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\nFADE IN:\n\nINT. LIVING ROOM - DAY\n\nA simple room with basic furniture. Light streams through the windows.\n\nJOHN sits at a desk, typing on a laptop. He looks frustrated.\n\n\t\t\tJOHN\n\t\t(sighing)\n\tThis screenplay isn't writing itself.\n\nHe takes a sip of coffee and continues typing.\n\nFADE OUT.")
	
	// Apply Courier Prime font for screenplay formatting
	w.applyScreenplayFont()
	
	// Create preview area
	w.previewArea = widget.NewRichText()
	w.previewArea.Wrapping = fyne.TextWrapWord
	
	// Create header bar with toolbar
	w.headerBar = NewHeaderBar(w)
	
	// Create search bar (initially hidden)
	w.searchBar = NewSearchBar(w)
	
	// Create toolbar container
	w.toolbarContainer = container.NewVBox(
		w.headerBar.container,
		w.searchBar.container,
	)
	
	// Create status bar with element indicator
	w.elementLabel = widget.NewLabel("Element: Action")
	w.statusLabel = widget.NewLabel("Ready")
	w.statusBar = container.NewHBox(
		w.statusLabel,
		widget.NewSeparator(),
		widget.NewLabel("Words: 0"),
		widget.NewSeparator(),
		widget.NewLabel("Characters: 0"),
		widget.NewSeparator(),
		w.elementLabel,
	)
	
	// Create main content area - just the editor for now
	editorScroll := container.NewScroll(w.textEditor)
	
	// Create main layout
	content := container.NewBorder(
		w.toolbarContainer, // top
		w.statusBar,        // bottom
		nil,                // left
		nil,                // right
		editorScroll,       // center - just the editor
	)
	
	w.fyneWindow.SetContent(content)
	
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
	themeName := w.settings.GetString("theme")
	_ = GetThemeByName(themeName)
	w.fyneWindow.SetContent(w.fyneWindow.Content()) // Refresh with new theme
	
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
	// Enable/disable word wrap
	if w.settings.GetBoolean("word-wrap") {
		w.textEditor.Wrapping = fyne.TextWrapWord
	} else {
		w.textEditor.Wrapping = fyne.TextWrapOff
	}
	
	// TODO: Apply other editor settings like auto-indent, spell check, etc.
}

func (w *MainWindow) applyFontSettings() {
	// TODO: Implement font settings
	// Fyne has limited font customization compared to GTK
}

// File operations
func (w *MainWindow) LoadFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}
	
	w.textEditor.SetText(string(content))
	w.currentFile = filePath
	w.hasChanges = false
	w.updateTitle()
	w.updatePreview()
	w.updateStatusBar()
	
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
		
		content := w.textEditor.Text
		_, err = io.WriteString(closer, content)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to save file: %v", err), w.fyneWindow)
			return
		}
		
		w.currentFile = filePath
		w.hasChanges = false
		w.updateTitle()
		
		dialog.ShowInformation("File Saved", fmt.Sprintf("File saved to %s", filepath.Base(filePath)), w.fyneWindow)
	}, w.fyneWindow)
	
	// Set default filename
	if w.currentFile != "" {
		fileDialog.SetFileName(filepath.Base(w.currentFile))
	} else {
		fileDialog.SetFileName("untitled.fountain")
	}
	
	// TODO: Set file filter when Fyne supports it
	// fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".fountain", ".spmd"}))
	
	fileDialog.Show()
	return nil
}

func (w *MainWindow) saveToFile(filePath string) error {
	content := w.textEditor.Text
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
	return strings.TrimSpace(w.textEditor.Text) == ""
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
	} else if w.hasChanges {
		title = "• Untitled - Accolade"
	}
	
	w.fyneWindow.SetTitle(title)
}

func (w *MainWindow) updatePreview() {
	if w.previewVisible {
		text := w.textEditor.Text
		// Parse with Lexington and generate HTML preview
		elements, err := w.lexParser.ParseText(text)
		if err != nil {
			log.Printf("Error parsing Fountain text: %v", err)
			return
		}
		
		htmlPreview := w.lexParser.FormatForPreview(elements)
		w.previewArea.ParseMarkdown(htmlPreview)
	}
}

func (w *MainWindow) setStatus(text string) {
	if w.statusLabel != nil {
		w.statusLabel.SetText(text)
	}
}

func (w *MainWindow) updateStatusBar() {
	text := w.textEditor.Text
	words := countWords(text)
	chars := len(text)
	
	// Parse with Lexington for more detailed stats
	elements, err := w.lexParser.ParseText(text)
	if err == nil {
		pages := w.estimatePageCount(elements)
		log.Printf("Status: Words: %d, Characters: %d, Pages: %d", words, chars, pages)
	} else {
		log.Printf("Status: Words: %d, Characters: %d", words, chars)
	}
}

func (w *MainWindow) onCursorChanged() {
	w.updateCurrentElement()
}

func (w *MainWindow) updateCurrentElement() {
	text := w.textEditor.Text
	if text == "" {
		w.elementLabel.SetText("Element: Empty")
		w.currentElement = "empty"
		return
	}
	
	// Get cursor position - use CursorRow for line detection
	cursorRow := w.textEditor.CursorRow
	
	// Find the current line
	lines := strings.Split(text, "\n")
	
	if cursorRow < len(lines) {
		currentLine := strings.TrimSpace(lines[cursorRow])
		elementType := w.detectElementType(currentLine)
		w.elementLabel.SetText("Element: " + elementType)
		w.currentElement = elementType
	}
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

func (w *MainWindow) applyScreenplayFont() {
	// Apply monospace font for proper screenplay formatting
	// Fyne doesn't have direct font setting for widgets, but we can use text styling
	// This sets the text to use a monospace font family
	w.textEditor.TextStyle = fyne.TextStyle{
		Monospace: true,
	}
	log.Println("Applied monospace font for screenplay formatting")
}

func (w *MainWindow) estimatePageCount(elements []ParsedElement) int {
	// Rough estimate: 250 words per page for screenplays
	// This is a simplified calculation
	wordCount := 0
	for _, element := range elements {
		wordCount += len(strings.Fields(element.Text))
	}
	pages := wordCount / 250
	if pages == 0 {
		pages = 1
	}
	return pages
}

// Preview operations
func (w *MainWindow) togglePreview() {
	if w.previewVisible {
		w.hidePreview()
	} else {
		w.showPreview()
	}
}

func (w *MainWindow) showPreview() {
	// Show preview pane by creating a split container
	w.previewVisible = true
	w.updatePreview()
	w.settings.SetBoolean("preview-visible", true)
	
	// Create the split container with both editor and preview
	editorScroll := container.NewScroll(w.textEditor)
	previewScroll := container.NewScroll(w.previewArea)
	
	splitContainer := container.NewHSplit(
		editorScroll,
		previewScroll,
	)
	splitContainer.SetOffset(0.7) // Give more space to editor
	
	// Update the main layout
	content := container.NewBorder(
		w.toolbarContainer, // top
		w.statusBar,        // bottom
		nil,                // left
		nil,                // right
		splitContainer,     // center - split container
	)
	
	w.fyneWindow.SetContent(content)
}

func (w *MainWindow) hidePreview() {
	// Hide preview pane by replacing the split container with just the editor
	w.previewVisible = false
	w.settings.SetBoolean("preview-visible", false)
	
	// Replace the split container with just the editor scroll
	editorScroll := container.NewScroll(w.textEditor)
	
	// Update the main layout to show only the editor
	content := container.NewBorder(
		w.toolbarContainer, // top
		w.statusBar,        // bottom
		nil,                // left
		nil,                // right
		editorScroll,       // center - just the editor
	)
	
	w.fyneWindow.SetContent(content)
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
