package main

import (
	"log"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type PreferencesDialog struct {
	*adw.PreferencesWindow
	settings *Settings
}

func NewPreferencesDialog() *PreferencesDialog {
	return &PreferencesDialog{
		PreferencesWindow: adw.NewPreferencesWindow(),
		settings:          NewSettings(),
	}
}

func (pd *PreferencesDialog) Present() {
	log.Println("Preferences dialog presented")
	pd.PreferencesWindow.Present()
}

func (pd *PreferencesDialog) SetTransientFor(parent gtk.Windower) {
	pd.PreferencesWindow.SetTransientFor(parent)
}

func (pd *PreferencesDialog) setupUI() {
	// TODO: Set up preferences UI
	log.Println("Setting up preferences UI")
}

func (pd *PreferencesDialog) loadSettings() {
	// TODO: Load current settings
	log.Println("Loading settings")
}

func (pd *PreferencesDialog) saveSettings() {
	// TODO: Save settings
	log.Println("Saving settings")
}