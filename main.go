package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
)

const (
	AppID = "org.codeberg.lapingvino.Accolade"
)

type Application struct {
	*adw.Application
	windows []*MainWindow
}

func NewApplication() *Application {
	app := &Application{
		Application: adw.NewApplication(AppID, gio.ApplicationFlagsHandlesOpen|gio.ApplicationFlagsNonUnique),
		windows:     make([]*MainWindow, 0),
	}
	
	app.ConnectActivate(app.onActivate)
	app.ConnectOpen(app.onOpen)
	app.ConnectStartup(app.onStartup)
	
	return app
}

func (app *Application) onStartup() {
	log.Println("Application starting up...")
	
	// Set up CSS providers
	app.setupStyles()
	
	// Set up actions
	app.setupActions()
	
	// Set up keyboard shortcuts
	app.setupShortcuts()
}

func (app *Application) setupStyles() {
	// Load CSS for sepia theme
	cssProvider := gtk.NewCSSProvider()
	cssProvider.LoadFromResource("/org/codeberg/lapingvino/Accolade/style-sepia.css")
	
	// Add resource paths
	iconTheme := gtk.IconThemeGetForDisplay(gtk.DisplayGetDefault())
	iconTheme.AddResourcePath("/org/codeberg/lapingvino/Accolade/icons")
}

func (app *Application) setupActions() {
	// New window action
	newWindowAction := gio.NewSimpleAction("new_window", nil)
	newWindowAction.ConnectActivate(func() {
		app.newWindow()
	})
	app.AddAction(newWindowAction)
	
	// Preferences action
	preferencesAction := gio.NewSimpleAction("preferences", nil)
	preferencesAction.ConnectActivate(func() {
		app.showPreferences()
	})
	app.AddAction(preferencesAction)
	
	// About action
	aboutAction := gio.NewSimpleAction("about", nil)
	aboutAction.ConnectActivate(func() {
		app.showAbout()
	})
	app.AddAction(aboutAction)
	
	// Quit action
	quitAction := gio.NewSimpleAction("quit", nil)
	quitAction.ConnectActivate(func() {
		app.quit()
	})
	app.AddAction(quitAction)
	
	// Color scheme action
	colorSchemeAction := gio.NewSimpleActionStateful("color_scheme", 
		glib.NewVariantType("s"), glib.NewVariantString("system"))
	colorSchemeAction.ConnectActivate(func(action *gio.SimpleAction, parameter *glib.Variant) {
		if parameter != nil {
			action.SetState(parameter)
			app.setColorScheme(parameter.String())
		}
	})
	app.AddAction(colorSchemeAction)
}

func (app *Application) setupShortcuts() {
	// Window shortcuts
	app.SetAccelsForAction("win.focus_mode", []string{"<Ctrl>d"})
	app.SetAccelsForAction("win.hemingway_mode", []string{"<Ctrl>t"})
	app.SetAccelsForAction("win.preview", []string{"<Ctrl>p"})
	app.SetAccelsForAction("win.fullscreen", []string{"F11"})
	app.SetAccelsForAction("win.find", []string{"<Ctrl>f"})
	app.SetAccelsForAction("win.find_replace", []string{"<Ctrl>h"})
	
	// Application shortcuts
	app.SetAccelsForAction("app.new_window", []string{"<Ctrl>n"})
	app.SetAccelsForAction("app.preferences", []string{"<Ctrl>comma"})
	app.SetAccelsForAction("app.quit", []string{"<Ctrl>q"})
	
	// File shortcuts
	app.SetAccelsForAction("win.open", []string{"<Ctrl>o"})
	app.SetAccelsForAction("win.save", []string{"<Ctrl>s"})
	app.SetAccelsForAction("win.save_as", []string{"<Ctrl><Shift>s"})
	app.SetAccelsForAction("win.close", []string{"<Ctrl>w"})
	
	// Spell check
	app.SetAccelsForAction("app.spellcheck", []string{"F7"})
}

func (app *Application) onActivate() {
	log.Println("Application activated")
	
	if len(app.windows) == 0 {
		app.newWindow()
	}
	
	// Present the last window
	if len(app.windows) > 0 {
		app.windows[len(app.windows)-1].Present()
	}
}

func (app *Application) onOpen(files []*gio.File, hint string) {
	log.Printf("Opening %d files", len(files))
	
	app.Activate()
	
	// Find empty windows
	emptyWindows := make([]*MainWindow, 0)
	for _, window := range app.windows {
		if window.IsEmpty() && !window.HasUnsavedChanges() {
			emptyWindows = append(emptyWindows, window)
		}
	}
	
	// Open files
	for i, file := range files {
		var window *MainWindow
		
		if i < len(emptyWindows) {
			window = emptyWindows[i]
		} else {
			window = app.newWindow()
		}
		
		window.LoadFile(file)
		window.Present()
	}
}

func (app *Application) newWindow() *MainWindow {
	log.Println("Creating new window")
	
	window := NewMainWindow(app)
	app.windows = append(app.windows, window)
	
	// Set up window close handler
	window.ConnectCloseRequest(func() bool {
		return app.onWindowCloseRequest(window)
	})
	
	window.Show()
	return window
}

func (app *Application) onWindowCloseRequest(window *MainWindow) bool {
	// Remove window from list
	for i, w := range app.windows {
		if w == window {
			app.windows = append(app.windows[:i], app.windows[i+1:]...)
			break
		}
	}
	
	// If no windows left, quit
	if len(app.windows) == 0 {
		app.Quit()
	}
	
	return false // Allow close
}

func (app *Application) showPreferences() {
	log.Println("Showing preferences")
	
	// Get active window
	var activeWindow *MainWindow
	if len(app.windows) > 0 {
		activeWindow = app.windows[0] // Use first window as parent
	}
	
	dialog := NewPreferencesDialog()
	if activeWindow != nil {
		dialog.SetTransientFor(&activeWindow.Window)
	}
	dialog.Present()
}

func (app *Application) showAbout() {
	log.Println("Showing about dialog")
	
	// Get active window
	var activeWindow *MainWindow
	if len(app.windows) > 0 {
		activeWindow = app.windows[0]
	}
	
	aboutDialog := adw.NewAboutWindow()
	aboutDialog.SetApplicationName("Accolade")
	aboutDialog.SetApplicationIcon(AppID)
	aboutDialog.SetVersion("0.1.0")
	aboutDialog.SetDeveloperName("Joop Kiefte")
	aboutDialog.SetCopyright("© 2024 Joop Kiefte")
	aboutDialog.SetLicense("GPL-3.0-or-later")
	aboutDialog.SetWebsite("https://codeberg.org/lapingvino/accolade")
	aboutDialog.SetIssueURL("https://codeberg.org/lapingvino/accolade/issues")
	aboutDialog.SetComments("A distraction-free Fountain editor for screenwriters")
	
	if activeWindow != nil {
		aboutDialog.SetTransientFor(&activeWindow.Window)
	}
	
	aboutDialog.Present()
}

func (app *Application) setColorScheme(scheme string) {
	log.Printf("Setting color scheme to: %s", scheme)
	
	styleManager := adw.StyleManagerGetDefault()
	switch scheme {
	case "light":
		styleManager.SetColorScheme(adw.ColorSchemeForceLight)
	case "dark":
		styleManager.SetColorScheme(adw.ColorSchemeForceDark)
	default:
		styleManager.SetColorScheme(adw.ColorSchemeDefault)
	}
}

func (app *Application) quit() {
	log.Println("Quitting application")
	
	// Check for unsaved changes
	hasUnsaved := false
	for _, window := range app.windows {
		if window.HasUnsavedChanges() {
			hasUnsaved = true
			break
		}
	}
	
	if hasUnsaved {
		// TODO: Show confirmation dialog
		log.Println("Warning: There are unsaved changes")
	}
	
	app.Quit()
}

func main() {
	// Initialize GTK
	gtk.Init()
	adw.Init()
	
	// Create application
	app := NewApplication()
	
	// Set resource base path
	app.SetResourceBasePath("/org/codeberg/lapingvino/Accolade")
	
	// Run application
	ctx := context.Background()
	if code := app.RunWithContext(ctx, os.Args); code > 0 {
		os.Exit(code)
	}
}

// Helper function to get resource path
func getResourcePath(path string) string {
	return filepath.Join("/org/codeberg/lapingvino/Accolade", path)
}

// Helper function to show error dialog
func showError(parent gtk.Windower, title, message string) {
	dialog := adw.NewAlertDialog(title, message)
	dialog.AddResponse("close", "Close")
	dialog.SetDefaultResponse("close")
	dialog.SetCloseResponse("close")
	
	if parent != nil {
		dialog.SetTransientFor(parent)
	}
	
	dialog.Present()
}

// Helper function to check if executable exists
func executableExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

// Helper function to get debug info
func getDebugInfo() string {
	info := fmt.Sprintf("Accolade %s\n", "0.1.0")
	info += fmt.Sprintf("GTK: %d.%d.%d\n", gtk.GetMajorVersion(), gtk.GetMinorVersion(), gtk.GetMicroVersion())
	info += fmt.Sprintf("GLib: %d.%d.%d\n", glib.GetMajorVersion(), glib.GetMinorVersion(), glib.GetMicroVersion())
	info += fmt.Sprintf("Libadwaita: %d.%d.%d\n", adw.GetMajorVersion(), adw.GetMinorVersion(), adw.GetMicroVersion())
	
	// Check for lexington
	if executableExists("lexington") {
		info += "Lexington: Available\n"
	} else {
		info += "Lexington: Not found\n"
	}
	
	return info
}