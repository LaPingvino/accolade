package main

import (
	"runtime/debug"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2/driver/desktop"
)

const (
	AppID = "eu.kiefte.Accolade"
)

type Application struct {
	fyneApp fyne.App
	windows []*MainWindow
	ctx     context.Context
}

func NewApplication() *Application {
	app := &Application{
		fyneApp: app.NewWithID(AppID),
		windows: make([]*MainWindow, 0),
		ctx:     context.Background(),
	}
	
	app.setupMetadata()
	app.setupLifecycle()
	
	return app
}

// setupMetadata used to set the name and version here; they now come from
// FyneApp.toml, which the fyne tool embeds when packaging.
func (app *Application) setupMetadata() {}

// appVersion is Accolade's version as packaged (FyneApp.toml, set from the
// release tag), or "dev" for a plain go build.
func appVersion() string {
	if a := fyne.CurrentApp(); a != nil && a.Metadata().Version != "" {
		return a.Metadata().Version
	}
	return "dev"
}

func (app *Application) setupLifecycle() {
	app.fyneApp.SetIcon(nil) // TODO: Set app icon
	
	// Setup global shortcuts
	if deskApp, ok := app.fyneApp.(desktop.App); ok {
		deskApp.SetSystemTrayMenu(app.createSystemTrayMenu())
	}
}

func (app *Application) createSystemTrayMenu() *fyne.Menu {
	newWindow := fyne.NewMenuItem("New Window", func() {
		app.newWindow()
	})
	
	preferences := fyne.NewMenuItem("Preferences", func() {
		app.showPreferences()
	})
	
	about := fyne.NewMenuItem("About", func() {
		app.showAbout()
	})
	
	quit := fyne.NewMenuItem("Quit", func() {
		app.quit()
	})
	
	return fyne.NewMenu("Accolade", newWindow, fyne.NewMenuItemSeparator(), preferences, about, fyne.NewMenuItemSeparator(), quit)
}

func (app *Application) Run(args []string) {
	log.Println("Starting Accolade application")
	
	// Handle file arguments
	if len(args) > 1 {
		app.openFiles(args[1:])
	} else {
		app.newWindow()
	}
	
	// Show and run
	app.fyneApp.Run()
}

func (app *Application) openFiles(filePaths []string) {
	// Find empty windows
	emptyWindows := make([]*MainWindow, 0)
	for _, window := range app.windows {
		if window.IsEmpty() && !window.HasUnsavedChanges() {
			emptyWindows = append(emptyWindows, window)
		}
	}
	
	// Open files
	for i, filePath := range filePaths {
		var window *MainWindow
		
		if i < len(emptyWindows) {
			window = emptyWindows[i]
		} else {
			window = app.newWindow()
		}
		
		err := window.LoadFile(filePath)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to open file %s: %v", filePath, err), window.fyneWindow)
		}
		
		window.Show()
	}
}

func (app *Application) newWindow() *MainWindow {
	log.Println("Creating new window")
	
	window := NewMainWindow(app)
	app.windows = append(app.windows, window)
	
	// Set up window close handler
	window.fyneWindow.SetOnClosed(func() {
		app.onWindowClosed(window)
	})
	
	window.Show()
	return window
}

func (app *Application) onWindowClosed(window *MainWindow) {
	window.cancelAutoSave()

	// Remove window from list
	for i, w := range app.windows {
		if w == window {
			app.windows = append(app.windows[:i], app.windows[i+1:]...)
			break
		}
	}
	
	// If no windows left, quit
	if len(app.windows) == 0 {
		app.fyneApp.Quit()
	}
}

func (app *Application) showPreferences() {
	log.Println("Showing preferences")
	
	// Get active window as parent
	var parent fyne.Window
	if len(app.windows) > 0 {
		parent = app.windows[0].fyneWindow
	}
	
	dialog := NewPreferencesDialog(app, parent)
	dialog.Show()
}

func (app *Application) showAbout() {
	log.Println("Showing about dialog")
	
	// Get active window as parent
	var parent fyne.Window
	if len(app.windows) > 0 {
		parent = app.windows[0].fyneWindow
	}
	
	content := widget.NewRichTextFromMarkdown(`
# Accolade

**Version:** ` + appVersion() + `  
**Developer:** Joop Kiefte  
**Copyright:** © 2024 Joop Kiefte  
**License:** GPL-3.0-or-later  

A distraction-free Fountain editor for screenwriters.

**Website:** https://github.com/LaPingvino/accolade  
**Issues:** https://github.com/LaPingvino/accolade/issues
`)
	
	aboutDialog := dialog.NewCustom("About Accolade", "Close", content, parent)
	aboutDialog.Resize(fyne.NewSize(500, 400))
	aboutDialog.Show()
}

// applySettingsToWindows re-applies the editor settings in every window
// after the preferences changed.
func (app *Application) applySettingsToWindows() {
	for _, w := range app.windows {
		w.applyEditorSettings()
		w.applyFontSettings()
		w.updatePreview() // the script format may have changed
	}
}

func (app *Application) setColorScheme(scheme string) {
	log.Printf("Setting color scheme to: %s", scheme)
	
	switch scheme {
	case "light":
		app.fyneApp.Settings().SetTheme(&LightTheme{})
	case "dark":
		app.fyneApp.Settings().SetTheme(&DarkTheme{})
	case "sepia":
		app.fyneApp.Settings().SetTheme(&SepiaTheme{})
	default:
		// Fyne's own theme follows the system's light/dark preference
		app.fyneApp.Settings().SetTheme(theme.DefaultTheme())
	}
}

func (app *Application) quit() {
	log.Println("Quitting application")
	
	// Check for unsaved changes
	hasUnsaved := false
	unsavedWindows := make([]*MainWindow, 0)
	
	for _, window := range app.windows {
		if window.HasUnsavedChanges() {
			hasUnsaved = true
			unsavedWindows = append(unsavedWindows, window)
		}
	}
	
	if hasUnsaved {
		// Show confirmation dialog for first window with unsaved changes
		parent := unsavedWindows[0].fyneWindow
		
		confirmDialog := dialog.NewConfirm(
			"Unsaved Changes",
			fmt.Sprintf("You have unsaved changes in %d window(s). Are you sure you want to quit?", len(unsavedWindows)),
			func(confirmed bool) {
				if confirmed {
					app.fyneApp.Quit()
				}
			},
			parent,
		)
		confirmDialog.Show()
		return
	}
	
	app.fyneApp.Quit()
}

func main() {
	// Create application
	app := NewApplication()
	
	// Run application
	app.Run(os.Args)
}

// Helper function to show error dialog
func showError(parent fyne.Window, title, message string) {
	errorDialog := dialog.NewError(fmt.Errorf("%s: %s", title, message), parent)
	errorDialog.Show()
}

// Helper function to show info dialog
func showInfo(parent fyne.Window, title, message string) {
	infoDialog := dialog.NewInformation(title, message, parent)
	infoDialog.Show()
}

// Helper function to check if executable exists
// moduleVersion is the version of a dependency this binary was built with.
func moduleVersion(path string) string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, m := range bi.Deps {
			if m.Path == path {
				if m.Replace != nil {
					return m.Version + " (replaced by " + m.Replace.Path + ")"
				}
				return m.Version
			}
		}
	}
	return "unknown"
}

// Helper function to get debug info
func getDebugInfo() string {
	info := fmt.Sprintf("Accolade %s\n", appVersion())
	info += fmt.Sprintf("Fyne: %s\n", moduleVersion("fyne.io/fyne/v2"))
	info += fmt.Sprintf("Lexington: %s (built in)\n", moduleVersion("github.com/LaPingvino/lexington"))
	
	return info
}

// Helper function to get home directory
func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp"
	}
	return home
}

// getConfigDir is where Accolade keeps its settings: accolade/ in
// $XDG_CONFIG_HOME if set, else in the system's configuration directory
// (~/.config on Linux, ~/Library/Application Support on macOS, %AppData%
// on Windows).
func getConfigDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "accolade")
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "accolade")
	}
	return legacyConfigDir()
}

// legacyConfigDir is where earlier versions kept the settings on every
// system.
func legacyConfigDir() string {
	return filepath.Join(getHomeDir(), ".config", "accolade")
}

// Helper function to ensure directory exists
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// Helper function to get file extension
func getFileExtension(filename string) string {
	return strings.ToLower(filepath.Ext(filename))
}

// Helper function to check if file is a Fountain file
func isFountainFile(filename string) bool {
	ext := getFileExtension(filename)
	return ext == ".fountain" || ext == ".spmd"
}



// Resource helpers
func getResourcePath(path string) string {
	return filepath.Join("/eu/kiefte/Accolade", path)
}