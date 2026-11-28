// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/rules"
)

// Lexington's settings: wrong TOML is refused with its line; an element
// set of one's own is a script format, a scene language one more.
func TestLexingtonSettings(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Cleanup(func() { lexSettings = readLexingtonSettings(); sceneConf = lexSettings.Scenes })
	if err := saveLexingtonSettings("[Elements.mine\n"); err == nil || strings.Contains(err.Error(), ".new") {
		t.Fatalf("wrong TOML: %v", err)
	}
	text := strings.Replace(defaultSettingsText(t), "[Elements.default", "[Elements.mine", 1)
	text += "\n[Languages.xx]\nScenes = [\"INNE\", \"UTE\"]\n"
	if err := saveLexingtonSettings(text); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(scriptFormatNames(), "mine") {
		t.Errorf("formats: %v", scriptFormatNames())
	}
	if machineLex().formatElements("mine") == nil {
		t.Error("no elements for mine")
	}
	if !slices.Contains(scriptLanguages(), "xx") || !slices.Contains(scriptLanguages(), "en") {
		t.Errorf("languages: %v", scriptLanguages())
	}
}

// defaultSettingsText is the default settings written out in full.
func defaultSettingsText(t *testing.T) string {
	t.Helper()
	b, err := rules.DefaultConf().Encode()
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
