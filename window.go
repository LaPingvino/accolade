package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"libdb.so/gotk4-sourceview/pkg/gtksource/v5"
)

type MainWindow struct {
	*adw.ApplicationWindow
	
	// Core components
	app          *Application
	textView     *TextView
	textBuffer   *TextBuffer
	previewView  *PreviewView
	headerbar    *HeaderBar
	
	// UI elements
	overlay      *gtk.Overlay
	panels       *gtk.Paned
	previewStack *gtk.Stack
	searchbar    *SearchBar
	
	// File management
	currentFile  *gio.File
	hasChanges   bool
	isFullscreen bool
	
	// Settings
	settings     *Settings
	
	// Preview components
	previewSpinner *gtk.Spinner
	securityWarning *adw.StatusPage
	
	// Status and progress
	saveProgressBar *gtk.ProgressBar
	discardInfoBar  *gtk.InfoBar
	
	// Revealer for toolbar
	toolbarRevealer *gtk.Revealer
}

func NewMainWindow(app *Application) *MainWindow {
	window := &MainWindow{
		ApplicationWindow: adw.NewApplicationWindow(&app.Application.Application),
		app:               app,
		hasChanges:        false,
		isFullscreen:      false,
	}
	
	window.settings = NewSettings()
	window.setupUI()
	window.setupActions()
	window.setupSignals()
	
	return window
}

func (w *MainWindow) setupUI() {
	// Set window properties
	w.SetTitle("Accolade")
	w.SetDefaultSize(1000, 600)
	w.AddCSSClass("accolade-window")
	
	// Create main overlay
	w.overlay = gtk.NewOverlay()
	w.overlay.SetName("FullscreenOverlay")
	w.SetContent(w.overlay)
	
	// Create panels (main content area)
	w.panels = gtk.NewPaned(gtk.OrientationHorizontal)
	w.panels.SetHExpand(true)
	w.overlay.SetChild(w.panels)
	
	// Create text view and buffer
	w.textBuffer = NewTextBuffer()
	w.textView = NewTextView(w.textBuffer)
	
	// Create preview components
	w.setupPreview()
	
	// Set up panels
	w.panels.SetStartChild(w.textView)
	w.panels.SetEndChild(w.previewStack)
	w.panels.SetResizeStartChild(true)
	w.panels.SetResizeEndChild(true)
	
	// Create header bar
	w.headerbar = NewHeaderBar(w)
	
	// Create search bar
	w.searchbar = NewSearchBar(w)
	
	// Create toolbar overlay
	w.setupToolbar()
	
	// Create info bars
	w.setupInfoBars()
	
	// Apply initial settings
	w.applySettings()
}

func (w *MainWindow) setupPreview() {
	// Create preview stack
	w.previewStack = gtk.NewStack()
	w.previewStack.SetHExpand(true)
	w.previewStack.SetTransitionType(gtk.StackTransitionTypeCrossfade)
	w.previewStack.AddCSSClass("preview-background")
	
	// Create preview spinner
	w.previewSpinner = gtk.NewSpinner()
	w.previewSpinner.SetHAlign(gtk.AlignCenter)
	w.previewSpinner.SetVAlign(gtk.AlignCenter)
	w.previewSpinner.SetHExpand(true)
	w.previewSpinner.SetVExpand(true)
	
	spinnerPage := w.previewStack.AddChild(w.previewSpinner)
	spinnerPage.SetName("spinner")
	
	// Create security warning page
	w.securityWarning = adw.NewStatusPage()
	w.securityWarning.SetIconName("dialog-warning-symbolic")
	w.securityWarning.SetTitle("This file may be insecure")
	w.securityWarning.SetDescription("Previewing files from untrusted sources can be dangerous.\nIf you're unsure about the contents of this file, open it in the Restricted Preview.")
	
	// Create buttons for security warning
	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 12)
	buttonBox.SetHAlign(gtk.AlignCenter)
	buttonBox.SetMarginBottom(12)
	
	loadButton := gtk.NewButtonWithLabel("Load Preview")
	loadButton.AddCSSClass("pill")
	loadButton.ConnectClicked(func() {
		w.loadPreview(false)
	})
	
	restrictedButton := gtk.NewButtonWithLabel("Load Restricted Preview")
	restrictedButton.AddCSSClass("pill")
	restrictedButton.AddCSSClass("suggested-action")
	restrictedButton.ConnectClicked(func() {
		w.loadPreview(true)
	})
	
	buttonBox.Append(loadButton)
	buttonBox.Append(restrictedButton)
	
	containerBox := gtk.NewBox(gtk.OrientationVertical, 0)
	containerBox.Append(buttonBox)
	w.securityWarning.SetChild(containerBox)
	
	securityPage := w.previewStack.AddChild(w.securityWarning)
	securityPage.SetName("security")
	
	// Create preview view
	w.previewView = NewPreviewView(w)
	previewPage := w.previewStack.AddChild(w.previewView)
	previewPage.SetName("preview")
	
	// Initially hide preview
	w.previewStack.SetVisible(false)
}

func (w *MainWindow) setupToolbar() {
	// Create toolbar box
	toolbarBox := gtk.NewBox(gtk.OrientationVertical, 0)
	toolbarBox.SetVAlign(gtk.AlignStart)
	
	// Create toolbar revealer
	w.toolbarRevealer = gtk.NewRevealer()
	w.toolbarRevealer.SetTransitionType(gtk.RevealerTransitionTypeCrossfade)
	w.toolbarRevealer.SetTransitionDuration(450)
	w.toolbarRevealer.SetRevealChild(true)
	
	// Create window handle
	windowHandle := gtk.NewWindowHandle()
	
	// Create header container
	headerContainer := gtk.NewBox(gtk.OrientationVertical, 0)
	headerContainer.AddCSSClass("toolbars")
	headerContainer.AddCSSClass("top")
	
	headerContainer.Append(w.headerbar)
	headerContainer.Append(w.searchbar)
	
	windowHandle.SetChild(headerContainer)
	w.toolbarRevealer.SetChild(windowHandle)
	
	// Add motion controller for auto-hide
	motionController := gtk.NewEventControllerMotion()
	motionController.ConnectEnter(func(x, y float64) {
		w.revealHeaderbar()
	})
	w.toolbarRevealer.AddController(motionController)
	
	toolbarBox.Append(w.toolbarRevealer)
	
	w.overlay.AddOverlay(toolbarBox)
}

func (w *MainWindow) setupInfoBars() {
	// Create discard info bar
	w.discardInfoBar = gtk.NewInfoBar()
	w.discardInfoBar.SetMessageType(gtk.MessageTypeWarning)
	w.discardInfoBar.SetShowCloseButton(true)
	w.discardInfoBar.SetRevealed(false)
	
	// Create info bar content
	infoContent := gtk.NewBox(gtk.OrientationVertical, 0)
	
	titleLabel := gtk.NewLabel("File Has Changed on Disk")
	titleLabel.SetHAlign(gtk.AlignStart)
	titleLabel.SetWrap(true)
	titleLabel.AddCSSClass("heading")
	
	subtitleLabel := gtk.NewLabel("The file has been changed by another program")
	subtitleLabel.SetHAlign(gtk.AlignStart)
	subtitleLabel.SetWrap(true)
	
	infoContent.Append(titleLabel)
	infoContent.Append(subtitleLabel)
	
	w.discardInfoBar.AddChild(infoContent)
	
	// Add discard button
	discardButton := gtk.NewButtonWithLabel("Discard Changes and Reload")
	discardButton.SetUseUnderline(true)
	discardButton.ConnectClicked(func() {
		w.reloadFile()
	})
	w.discardInfoBar.AddActionWidget(discardButton, gtk.ResponseClose)
	
	// Add save progress bar
	w.saveProgressBar = gtk.NewProgressBar()
	w.saveProgressBar.SetVisible(false)
	w.saveProgressBar.SetVAlign(gtk.AlignStart)
	w.saveProgressBar.AddCSSClass("osd")
	
	// Add to overlay
	infoOverlay := gtk.NewBox(gtk.OrientationVertical, 0)
	infoOverlay.SetVAlign(gtk.AlignStart)
	infoOverlay.Append(w.discardInfoBar)
	infoOverlay.Append(w.saveProgressBar)
	
	w.overlay.AddOverlay(infoOverlay)
}

func (w *MainWindow) setupActions() {
	// File actions
	openAction := gio.NewSimpleAction("open", nil)
	openAction.ConnectActivate(func() {
		w.openFile()
	})
	w.AddAction(openAction)
	
	saveAction := gio.NewSimpleAction("save", nil)
	saveAction.ConnectActivate(func() {
		w.saveFile()
	})
	w.AddAction(saveAction)
	
	saveAsAction := gio.NewSimpleAction("save_as", nil)
	saveAsAction.ConnectActivate(func() {
		w.saveFileAs()
	})
	w.AddAction(saveAsAction)
	
	// View actions
	previewAction := gio.NewSimpleActionStateful("preview", nil, glib.NewVariantBoolean(false))
	previewAction.ConnectActivate(func() {
		w.togglePreview()
	})
	w.AddAction(previewAction)
	
	focusModeAction := gio.NewSimpleActionStateful("focus_mode", nil, glib.NewVariantBoolean(false))
	focusModeAction.ConnectActivate(func() {
		w.toggleFocusMode()
	})
	w.AddAction(focusModeAction)
	
	fullscreenAction := gio.NewSimpleAction("fullscreen", nil)
	fullscreenAction.ConnectActivate(func() {
		w.toggleFullscreen()
	})
	w.AddAction(fullscreenAction)
	
	// Edit actions
	findAction := gio.NewSimpleAction("find", nil)
	findAction.ConnectActivate(func() {
		w.showSearch()
	})
	w.AddAction(findAction)
	
	findReplaceAction := gio.NewSimpleAction("find_replace", nil)
	findReplaceAction.ConnectActivate(func() {
		w.showFindReplace()
	})
	w.AddAction(findReplaceAction)
	
	// Export action
	exportAction := gio.NewSimpleAction("export", nil)
	exportAction.ConnectActivate(func() {
		w.showExportDialog()
	})
	w.AddAction(exportAction)
}

func (w *MainWindow) setupSignals() {
	// Connect text buffer change signal
	w.textBuffer.ConnectChanged(func() {
		w.onTextChanged()
	})
	
	// Connect focus events
	focusController := gtk.NewEventControllerFocus()
	focusController.ConnectLeave(func() {
		w.revealHeaderbar()
	})
	w.textView.AddController(focusController)
	
	// Connect file monitor if we have a file
	if w.currentFile != nil {
		w.setupFileMonitor()
	}
}

func (w *MainWindow) applySettings() {
	// Apply theme
	w.updateTheme()
	
	// Apply editor settings
	w.textView.applySettings(w.settings)
	
	// Apply window settings
	if w.settings.GetBoolean("preview-active") {
		w.showPreview()
	}
	
	if w.settings.GetBoolean("toolbar-active") {
		w.showToolbar()
	}
}

// File operations
func (w *MainWindow) LoadFile(file *gio.File) {
	w.currentFile = file
	
	// Read file content
	content, err := w.readFileContent(file)
	if err != nil {
		showError(w, "Error Loading File", err.Error())
		return
	}
	
	// Set content
	w.textBuffer.SetText(content)
	w.hasChanges = false
	
	// Update window title
	w.updateTitle()
	
	// Setup file monitor
	w.setupFileMonitor()
	
	// Update preview
	w.updatePreview()
}

func (w *MainWindow) readFileContent(file *gio.File) (string, error) {
	fileStream, err := file.Read(nil)
	if err != nil {
		return "", err
	}
	defer fileStream.Close()
	
	// Read all content
	content := make([]byte, 0)
	buffer := make([]byte, 4096)
	
	for {
		n, err := fileStream.Read(buffer)
		if n > 0 {
			content = append(content, buffer[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return "", err
		}
	}
	
	return string(content), nil
}

func (w *MainWindow) IsEmpty() bool {
	return w.textBuffer.GetText() == ""
}

func (w *MainWindow) HasUnsavedChanges() bool {
	return w.hasChanges
}

func (w *MainWindow) onTextChanged() {
	w.hasChanges = true
	w.updateTitle()
	w.updatePreview()
}

func (w *MainWindow) updateTitle() {
	title := "Accolade"
	
	if w.currentFile != nil {
		filename := w.currentFile.GetBasename()
		if w.hasChanges {
			title = "• " + filename
		} else {
			title = filename
		}
	} else if w.hasChanges {
		title = "• Untitled"
	}
	
	w.SetTitle(title)
}

func (w *MainWindow) updatePreview() {
	if w.previewStack.GetVisible() {
		text := w.textBuffer.GetText()
		w.previewView.UpdateContent(text)
	}
}

func (w *MainWindow) updateTheme() {
	// Apply theme based on settings
	colorScheme := w.settings.GetString("color-scheme")
	
	styleManager := adw.StyleManagerGetDefault()
	switch colorScheme {
	case "light":
		styleManager.SetColorScheme(adw.ColorSchemeForceLight)
	case "dark":
		styleManager.SetColorScheme(adw.ColorSchemeForceDark)
	case "sepia":
		// Apply sepia theme
		styleManager.SetColorScheme(adw.ColorSchemeForceLight)
		// TODO: Add sepia CSS
	default:
		styleManager.SetColorScheme(adw.ColorSchemeDefault)
	}
}

// UI actions
func (w *MainWindow) togglePreview() {
	if w.previewStack.GetVisible() {
		w.hidePreview()
	} else {
		w.showPreview()
	}
}

func (w *MainWindow) showPreview() {
	w.previewStack.SetVisible(true)
	w.updatePreview()
}

func (w *MainWindow) hidePreview() {
	w.previewStack.SetVisible(false)
}

func (w *MainWindow) loadPreview(restricted bool) {
	w.previewStack.SetVisibleChildName("preview")
	w.previewView.SetRestrictedMode(restricted)
	w.updatePreview()
}

func (w *MainWindow) toggleFocusMode() {
	// TODO: Implement focus mode
	log.Println("Toggle focus mode")
}

func (w *MainWindow) toggleFullscreen() {
	if w.isFullscreen {
		w.Unfullscreen()
		w.isFullscreen = false
	} else {
		w.Fullscreen()
		w.isFullscreen = true
	}
}

func (w *MainWindow) revealHeaderbar() {
	w.toolbarRevealer.SetRevealChild(true)
}

func (w *MainWindow) hideHeaderbar() {
	if w.settings.GetBoolean("autohide-headerbar") {
		w.toolbarRevealer.SetRevealChild(false)
	}
}

func (w *MainWindow) showSearch() {
	w.searchbar.SetSearchMode(true)
}

func (w *MainWindow) showFindReplace() {
	w.searchbar.SetSearchMode(true)
	w.searchbar.SetReplaceMode(true)
}

func (w *MainWindow) showToolbar() {
	w.toolbarRevealer.SetRevealChild(true)
}

func (w *MainWindow) openFile() {
	fileDialog := gtk.NewFileDialog()
	fileDialog.SetTitle("Open File")
	
	// Set up filter for Fountain files
	fountainFilter := gtk.NewFileFilter()
	fountainFilter.SetName("Fountain Files")
	fountainFilter.AddPattern("*.fountain")
	fountainFilter.AddPattern("*.spmd")
	
	allFilter := gtk.NewFileFilter()
	allFilter.SetName("All Files")
	allFilter.AddPattern("*")
	
	filterList := gio.NewListStore(glib.TypeFromInstance(fountainFilter))
	filterList.Append(fountainFilter)
	filterList.Append(allFilter)
	
	fileDialog.SetFilters(filterList)
	fileDialog.SetDefaultFilter(fountainFilter)
	
	fileDialog.Open(w, nil, func(result gio.AsyncResulter) {
		file, err := fileDialog.OpenFinish(result)
		if err != nil {
			if !strings.Contains(err.Error(), "dismissed") {
				showError(w, "Error", err.Error())
			}
			return
		}
		
		w.LoadFile(file)
	})
}

func (w *MainWindow) saveFile() {
	if w.currentFile == nil {
		w.saveFileAs()
		return
	}
	
	w.saveToFile(w.currentFile)
}

func (w *MainWindow) saveFileAs() {
	fileDialog := gtk.NewFileDialog()
	fileDialog.SetTitle("Save File")
	
	// Set up filter for Fountain files
	fountainFilter := gtk.NewFileFilter()
	fountainFilter.SetName("Fountain Files")
	fountainFilter.AddPattern("*.fountain")
	
	filterList := gio.NewListStore(glib.TypeFromInstance(fountainFilter))
	filterList.Append(fountainFilter)
	
	fileDialog.SetFilters(filterList)
	fileDialog.SetDefaultFilter(fountainFilter)
	
	// Set default filename
	if w.currentFile != nil {
		fileDialog.SetInitialName(w.currentFile.GetBasename())
	} else {
		fileDialog.SetInitialName("untitled.fountain")
	}
	
	fileDialog.Save(w, nil, func(result gio.AsyncResulter) {
		file, err := fileDialog.SaveFinish(result)
		if err != nil {
			if !strings.Contains(err.Error(), "dismissed") {
				showError(w, "Error", err.Error())
			}
			return
		}
		
		w.currentFile = file
		w.saveToFile(file)
	})
}

func (w *MainWindow) saveToFile(file *gio.File) {
	w.saveProgressBar.SetVisible(true)
	w.saveProgressBar.SetPulse(true)
	
	go func() {
		content := w.textBuffer.GetText()
		
		// Write to file
		err := w.writeFileContent(file, content)
		
		glib.IdleAdd(func() {
			w.saveProgressBar.SetVisible(false)
			
			if err != nil {
				showError(w, "Error Saving File", err.Error())
			} else {
				w.hasChanges = false
				w.updateTitle()
			}
		})
	}()
}

func (w *MainWindow) writeFileContent(file *gio.File, content string) error {
	fileStream, err := file.Replace(nil, false, gio.FileCreateFlagsNone, nil)
	if err != nil {
		return err
	}
	defer fileStream.Close()
	
	_, err = fileStream.WriteAll([]byte(content), nil)
	return err
}

func (w *MainWindow) reloadFile() {
	if w.currentFile != nil {
		w.LoadFile(w.currentFile)
	}
	w.discardInfoBar.SetRevealed(false)
}

func (w *MainWindow) setupFileMonitor() {
	if w.currentFile == nil {
		return
	}
	
	// TODO: Implement file monitoring
	log.Println("Setting up file monitor for:", w.currentFile.GetPath())
}

func (w *MainWindow) showExportDialog() {
	dialog := NewExportDialog(w)
	dialog.Present()
}

// Helper to get file extension
func (w *MainWindow) getFileExtension(filename string) string {
	return strings.ToLower(filepath.Ext(filename))
}