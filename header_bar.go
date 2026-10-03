package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type HeaderBar struct {
	window    *MainWindow
	container *fyne.Container
	
	// File buttons
	newButton    *widget.Button
	openButton   *widget.Button
	saveButton   *widget.Button
	saveAsButton *widget.Button
	
	// Edit buttons
	undoButton   *widget.Button
	redoButton   *widget.Button
	findButton   *widget.Button
	
	// View buttons
	previewButton    *widget.Button
	fullscreenButton *widget.Button
	focusModeButton  *widget.Button
	
	// Tools
	exportButton *widget.Button
	titlePageButton *widget.Button
	
	// Settings
	preferencesButton *widget.Button
	
	// Title area
	titleLabel *widget.Label
}

func NewHeaderBar(window *MainWindow) *HeaderBar {
	hb := &HeaderBar{
		window: window,
	}
	
	hb.createButtons()
	hb.createLayout()
	
	return hb
}

func (hb *HeaderBar) createButtons() {
	// File operations
	hb.newButton = widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
		hb.window.NewFile()
	})
	hb.newButton.SetText("New")
	
	hb.openButton = widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		hb.window.OpenFile()
	})
	hb.openButton.SetText("Open")
	
	hb.saveButton = widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), hb.window.reportErr(hb.window.SaveFile))
	hb.saveButton.SetText("Save")
	
	hb.saveAsButton = widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), hb.window.reportErr(hb.window.SaveFileAs))
	hb.saveAsButton.SetText("Save As")
	
	// Edit operations
	hb.undoButton = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), hb.window.undo)
	hb.undoButton.SetText("Undo")
	
	hb.redoButton = widget.NewButtonWithIcon("", theme.NavigateNextIcon(), hb.window.redo)
	hb.redoButton.SetText("Redo")
	
	hb.findButton = widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		hb.window.showFindReplace()
	})
	hb.findButton.SetText("Find")
	
	// View operations
	hb.previewButton = widget.NewButtonWithIcon("", theme.VisibilityIcon(), func() {
		hb.window.togglePreview()
	})
	hb.previewButton.SetText("Preview")
	
	hb.fullscreenButton = widget.NewButtonWithIcon("", theme.ZoomFitIcon(), func() {
		hb.window.toggleFullscreen()
	})
	hb.fullscreenButton.SetText("Fullscreen")
	
	hb.focusModeButton = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		hb.window.toggleFocusMode()
	})
	hb.focusModeButton.SetText("Focus")
	
	// Tools
	hb.exportButton = widget.NewButtonWithIcon("", theme.DocumentIcon(), func() {
		hb.window.showExportDialog()
	})
	hb.exportButton.SetText("Export")
	
	hb.titlePageButton = widget.NewButtonWithIcon("", theme.InfoIcon(), func() {
		hb.window.showTitlePageDialog()
	})
	hb.titlePageButton.SetText("Title Page")
	
	// Settings
	hb.preferencesButton = widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		hb.window.app.showPreferences()
	})
	hb.preferencesButton.SetText("Preferences")
	
	// Title
	hb.titleLabel = widget.NewLabel("Accolade")
	hb.titleLabel.TextStyle = fyne.TextStyle{Bold: true}
}

func (hb *HeaderBar) createLayout() {
	// Create button groups
	fileGroup := container.NewHBox(
		hb.newButton,
		hb.openButton,
		hb.saveButton,
		hb.saveAsButton,
		widget.NewSeparator(),
	)
	
	editGroup := container.NewHBox(
		hb.undoButton,
		hb.redoButton,
		hb.findButton,
		widget.NewSeparator(),
	)
	
	viewGroup := container.NewHBox(
		hb.previewButton,
		hb.fullscreenButton,
		hb.focusModeButton,
		widget.NewSeparator(),
	)
	
	toolsGroup := container.NewHBox(
		hb.exportButton,
		hb.titlePageButton,
		widget.NewSeparator(),
	)
	
	settingsGroup := container.NewHBox(
		hb.preferencesButton,
	)
	
	// Create spacer to push title to center and settings to right
	spacer1 := widget.NewLabel("")
	spacer2 := widget.NewLabel("")
	
	// Create main container
	hb.container = container.NewHBox(
		fileGroup,
		editGroup,
		viewGroup,
		toolsGroup,
		spacer1,
		hb.titleLabel,
		spacer2,
		settingsGroup,
	)
}

func (hb *HeaderBar) SetTitle(title string) {
	hb.titleLabel.SetText(title)
}

func (hb *HeaderBar) UpdateButtons() {
	// Update button states based on window state
	hasChanges := hb.window.hasChanges
	
	// Save button should be enabled if there are changes
	if hasChanges {
		hb.saveButton.Enable()
	} else {
		hb.saveButton.Disable()
	}
	
	// Save As button should always be enabled if there's content
	if hb.window.textEditor.Text() != "" {
		hb.saveAsButton.Enable()
	} else {
		hb.saveAsButton.Disable()
	}
	
	// Export button should be enabled if there's content
	if hb.window.textEditor.Text() != "" {
		hb.exportButton.Enable()
	} else {
		hb.exportButton.Disable()
	}
	
	// Update preview button based on preview state
	if hb.window.previewVisible {
		hb.previewButton.SetIcon(theme.VisibilityOffIcon())
		hb.previewButton.SetText("Hide Preview")
	} else {
		hb.previewButton.SetIcon(theme.VisibilityIcon())
		hb.previewButton.SetText("Show Preview")
	}
	
	// Update fullscreen button based on state
	if hb.window.isFullscreen {
		hb.fullscreenButton.SetIcon(theme.ZoomOutIcon())
		hb.fullscreenButton.SetText("Exit Fullscreen")
	} else {
		hb.fullscreenButton.SetIcon(theme.ZoomFitIcon())
		hb.fullscreenButton.SetText("Fullscreen")
	}
}

func (hb *HeaderBar) SetSaveEnabled(enabled bool) {
	if enabled {
		hb.saveButton.Enable()
	} else {
		hb.saveButton.Disable()
	}
}

func (hb *HeaderBar) SetUndoEnabled(enabled bool) {
	if enabled {
		hb.undoButton.Enable()
	} else {
		hb.undoButton.Disable()
	}
}

func (hb *HeaderBar) SetRedoEnabled(enabled bool) {
	if enabled {
		hb.redoButton.Enable()
	} else {
		hb.redoButton.Disable()
	}
}

// Helper method to create toolbar-style buttons
func (hb *HeaderBar) createToolbarButton(icon fyne.Resource, text string, callback func()) *widget.Button {
	btn := widget.NewButtonWithIcon(text, icon, callback)
	// Make buttons smaller and more compact for toolbar
	btn.Resize(fyne.NewSize(80, 32))
	return btn
}

// Method to hide/show the header bar (for focus mode)
func (hb *HeaderBar) SetVisible(visible bool) {
	if visible {
		hb.container.Show()
	} else {
		hb.container.Hide()
	}
}

// Method to get the container for embedding in window
func (hb *HeaderBar) GetContainer() *fyne.Container {
	return hb.container
}