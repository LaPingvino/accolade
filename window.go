package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type MainWindow struct {
	fyneWindow   fyne.Window
	app          *Application
	
	// Core components
	textEditor   *widget.Entry
	previewArea  *widget.RichText
	headerBar    *HeaderBar
	searchBar    *SearchBar
	
	// UI layout
	mainContainer *container.Split
	sidePanel     *container.Split
	toolbarContainer *fyne.Container
	statusBar     *fyne.Container
	
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
	findEntry     *widget.Entry
	replaceEntry  *widget.Entry
	findVisible   bool
	
	// Auto-save
	autoSaveEnabled bool
}

func NewMainWindow(app *Application) *MainWindow {
	window := &MainWindow{
		fyneWindow:      app.fyneApp.NewWindow("Accolade"),
		app:             app,
		hasChanges:      false,
		isFullscreen:    false,
		previewVisible:  false,
		findVisible:     false,
		autoSaveEnabled: true,
	}
	
	window.settings = NewSettings()
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
	
	// Create main text editor
	w.textEditor = widget.NewMultiLineEntry()
	w.textEditor.Wrapping = fyne.TextWrapWord
	w.textEditor.SetPlaceHolder("Start writing your screenplay here...")
	
	// Create preview area
	w.previewArea = widget.NewRichText()
	w.previewArea.Wrapping = fyne.TextWrapWord
	
	// Create header bar with toolbar
	w.headerBar = NewHeaderBar(w)
	
	// Create search bar (initially hidden)
	w.searchBar = NewSearchBar(w)
	
	// Create find/replace widgets
	w.findEntry = widget.NewEntry()
	w.findEntry.SetPlaceHolder("Find...")
	w.replaceEntry = widget.NewEntry()
	w.replaceEntry.SetPlaceHolder("Replace with...")
	
	// Create find/replace container
	findContainer := container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabel("Find:"), nil, w.findEntry),
		container.NewBorder(nil, nil, widget.NewLabel("Replace:"), nil, w.replaceEntry),
		container.NewHBox(
			widget.NewButton("Find Next", w.findNext),
			widget.NewButton("Replace", w.replaceOne),
			widget.NewButton("Replace All", w.replaceAll),
			widget.NewButton("Close", w.hideFindReplace),
		),
	)
	
	// Create status bar
	w.statusBar = container.NewHBox(
		widget.NewLabel("Ready"),
		widget.NewSeparator(),
		widget.NewLabel("Words: 0"),
		widget.NewSeparator(),
		widget.NewLabel("Characters: 0"),
	)
	
	// Create toolbar container
	w.toolbarContainer = container.NewVBox(
		w.headerBar.container,
		w.searchBar.container,
	)
	
	// Create main content area
	editorScroll := container.NewScroll(w.textEditor)
	previewScroll := container.NewScroll(w.previewArea)
	
	w.mainContainer = container.NewHSplit(
		editorScroll,
		previewScroll,
	)
	w.mainContainer.SetOffset(0.7) // Give more space to editor
	
	// Initially hide preview
	w.hidePreview()
	
	// Create main layout
	content := container.NewBorder(
		w.toolbarContainer, // top
		w.statusBar,        // bottom
		nil,                // left
		nil,                // right
		w.mainContainer,    // center
	)
	
	// Add find/replace overlay (initially hidden)
	overlay := container.NewWithoutLayout(
		content,
		findContainer,
	)
	
	// Position find/replace at top-right
	findContainer.Move(fyne.NewPos(200, 80))
	findContainer.Resize(fyne.NewSize(350, 120))
	findContainer.Hide()
	
	w.fyneWindow.SetContent(overlay)
	
	// Apply initial settings
	w.applySettings()
}

func (w *MainWindow) setupShortcuts() {
	// Text editor shortcuts
	w.textEditor.OnChanged = w.onTextChanged
	
	// TODO: Add keyboard shortcuts for Fyne
	// Fyne doesn't have as extensive shortcut support as GTK
	// Will need to implement custom key handlers
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
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}
	
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
	w.textEditor.SetText("")
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
	
	// Auto-save if enabled
	if w.autoSaveEnabled && w.currentFile != "" {
		// TODO: Implement debounced auto-save
	}
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
		// Convert Fountain text to formatted preview
		preview := w.formatFountainPreview(text)
		w.previewArea.ParseMarkdown(preview)
	}
}

func (w *MainWindow) updateStatusBar() {
	text := w.textEditor.Text
	words := countWords(text)
	chars := len(text)
	
	// Update status bar labels (this is a simplified approach)
	// In a real implementation, you'd want to access specific widgets
	log.Printf("Status: Words: %d, Characters: %d", words, chars)
}

func (w *MainWindow) formatFountainPreview(text string) string {
	// Basic Fountain to Markdown conversion
	// This is a simplified version - a full implementation would need
	// a proper Fountain parser
	
	lines := strings.Split(text, "\n")
	var preview strings.Builder
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if line == "" {
			preview.WriteString("\n\n")
			continue
		}
		
		// Character names (ALL CAPS at start of line)
		if isCharacterName(line) {
			preview.WriteString("**" + line + "**\n\n")
			continue
		}
		
		// Scene headings
		if isSceneHeading(line) {
			preview.WriteString("## " + line + "\n\n")
			continue
		}
		
		// Transitions
		if isTransition(line) {
			preview.WriteString("*" + line + "*\n\n")
			continue
		}
		
		// Action/dialogue
		preview.WriteString(line + "\n\n")
	}
	
	return preview.String()
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
	// Show preview pane - need to implement proper split container handling
	w.previewVisible = true
	w.updatePreview()
	w.settings.SetBoolean("preview-visible", true)
}

func (w *MainWindow) hidePreview() {
	// Hide preview pane - need to implement proper split container handling
	w.previewVisible = false
	w.settings.SetBoolean("preview-visible", false)
}

// Find/Replace operations
func (w *MainWindow) showFindReplace() {
	// TODO: Implement find/replace visibility
	w.findVisible = true
	// TODO: Focus on find entry
}

func (w *MainWindow) hideFindReplace() {
	// TODO: Implement find/replace hiding
	w.findVisible = false
}

func (w *MainWindow) findNext() {
	// TODO: Implement find functionality
	searchText := w.findEntry.Text
	log.Printf("Finding: %s", searchText)
}

func (w *MainWindow) replaceOne() {
	// TODO: Implement replace functionality
	searchText := w.findEntry.Text
	replaceText := w.replaceEntry.Text
	log.Printf("Replacing '%s' with '%s'", searchText, replaceText)
}

func (w *MainWindow) replaceAll() {
	// TODO: Implement replace all functionality
	searchText := w.findEntry.Text
	replaceText := w.replaceEntry.Text
	log.Printf("Replacing all '%s' with '%s'", searchText, replaceText)
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

func isSceneHeading(line string) bool {
	upper := strings.ToUpper(line)
	return strings.HasPrefix(upper, "INT.") || strings.HasPrefix(upper, "EXT.") || strings.HasPrefix(upper, "FADE")
}

func isTransition(line string) bool {
	upper := strings.ToUpper(line)
	transitions := []string{"CUT TO:", "FADE IN:", "FADE OUT:", "FADE TO BLACK:", "DISSOLVE TO:"}
	for _, trans := range transitions {
		if strings.HasSuffix(upper, trans) {
			return true
		}
	}
	return false
}