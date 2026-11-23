package main

import (
	"errors"
	"os"
	"path/filepath"
	"sort"

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
	return conf.With(file)
}

// reloadLexingtonSettings reads the settings file again.
func reloadLexingtonSettings() {
	lexSettings = readLexingtonSettings()
	sceneConf = lexSettings.Scenes
	startsDialects.Clear()
	lastStarts.Store(nil)
}

// customFormats are the element sets of the settings that are not
// presets, by key.
func (c lexConf) customFormats() []string {
	var keys []string
	for k := range c.Elements {
		if _, preset := rules.GetPreset(k); !preset && k != "default" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

// lexingtonSettingsText is the settings file, or what a new one starts
// with: nothing but a word on what it is (the settings editor shows the
// defaults beside it).
func lexingtonSettingsText() string {
	if b, err := os.ReadFile(lexingtonConfig()); err == nil {
		return string(b)
	}
	return newLexingtonSettings
}

// newLexingtonSettings is a new settings file.
const newLexingtonSettings = "# Lexington's settings, Accolade's and the lexington command's (lexington -config this-file).\n" +
	"# What is not here is Lexington's default. Edit in Preferences > Script > Edit Lexington Settings….\n"

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
	reloadLexingtonSettings()
	return nil
}
