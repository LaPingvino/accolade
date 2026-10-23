package main

import (
	"errors"
	"os"
	"path/filepath"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/rules"
)

// Lexington's settings (lexington.toml in the settings folder) add
// element sets, which are script formats of one's own, and scene heading
// languages; the exports read the same file.

// lexSettings are the settings: the defaults with the file's on top.
var lexSettings = readLexingtonSettings()

func readLexingtonSettings() rules.TOMLConf {
	conf := rules.DefaultConf()
	if _, err := os.Stat(lexingtonConfig()); err != nil {
		return conf
	}
	file, err := rules.ReadFile(lexingtonConfig())
	if err != nil {
		return conf
	}
	for k, v := range file.Elements {
		conf.Elements[k] = v
	}
	for k, v := range file.Scenes {
		conf.Scenes[k] = v
	}
	return conf
}

// customFormats are the element sets of the settings that are not
// presets, by key.
func customFormats() []string {
	var keys []string
	for k := range lexSettings.Elements {
		if _, preset := rules.GetPreset(k); !preset && k != "default" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

// lexingtonSettingsText is the settings file, or the defaults to start
// one from.
func lexingtonSettingsText() string {
	if b, err := os.ReadFile(lexingtonConfig()); err == nil {
		return string(b)
	}
	return defaultLexingtonSettings()
}

func defaultLexingtonSettings() string {
	dir, err := os.MkdirTemp("", "accolade-settings-")
	if err != nil {
		return ""
	}
	defer os.RemoveAll(dir)
	p := filepath.Join(dir, "lexington.toml")
	if rules.Dump(p) != nil {
		return ""
	}
	b, _ := os.ReadFile(p)
	return string(b)
}

// saveLexingtonSettings checks text as Lexington's settings and keeps it.
func saveLexingtonSettings(text string) error {
	if err := os.MkdirAll(filepath.Dir(lexingtonConfig()), 0o755); err != nil {
		return err
	}
	tmp := lexingtonConfig() + ".new"
	if err := os.WriteFile(tmp, []byte(text), 0o644); err != nil {
		return err
	}
	if _, err := rules.ReadFile(tmp); err != nil {
		_ = os.Remove(tmp)
		if e := errors.Unwrap(err); e != nil {
			return e // the TOML's own message, with its line (not the file's name)
		}
		return err
	}
	if err := os.Rename(tmp, lexingtonConfig()); err != nil {
		return err
	}
	lexSettings = readLexingtonSettings()
	sceneConf = lexSettings.Scenes
	accoladeConf = readAccoladeSettings()
	return nil
}

// editLexingtonSettings edits the settings file; saved is called after a
// save.
func editLexingtonSettings(parent fyne.Window, saved func()) {
	entry := widget.NewMultiLineEntry()
	entry.TextStyle = fyne.TextStyle{Monospace: true}
	entry.SetText(lexingtonSettingsText())
	help := widget.NewLabel("Element sets here are script formats of your own (Preferences > Script format, the toolbar);\nScenes are what scene headings start with, by language. The exports use them too.")
	var d dialog.Dialog
	save := widget.NewButton("Save", func() {
		if err := saveLexingtonSettings(entry.Text); err != nil {
			dialog.ShowError(err, parent)
			return
		}
		d.Hide()
		if saved != nil {
			saved()
		}
	})
	save.Importance = widget.HighImportance
	defaults := widget.NewButton("Start from the Defaults", func() { entry.SetText(defaultLexingtonSettings()) })
	cancel := widget.NewButton("Cancel", func() { d.Hide() })
	buttons := container.NewHBox(defaults, layout.NewSpacer(), cancel, save)
	d = dialog.NewCustomWithoutButtons("Lexington Settings", container.NewBorder(help, buttons, nil, nil, entry), parent)
	d.Resize(fyne.NewSize(720, 560))
	d.Show()
}
