package main

import (
	"slices"
	"strings"
	"testing"
)

// Lexington's settings: wrong TOML is refused with its line; an element
// set of one's own is a script format, a scene language one more.
func TestLexingtonSettings(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Cleanup(func() { lexSettings = readLexingtonSettings(); sceneConf = lexSettings.Scenes })
	if err := saveLexingtonSettings("[Elements.mine\n"); err == nil || strings.Contains(err.Error(), ".new") {
		t.Fatalf("wrong TOML: %v", err)
	}
	text := strings.Replace(defaultLexingtonSettings(), "[Elements.default", "[Elements.mine", 1)
	text = strings.Replace(text, "[Scenes]\n", "[Scenes]\n  xx = [\"INNE\", \"UTE\"]\n", 1)
	if err := saveLexingtonSettings(text); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(scriptFormatNames(), "mine") {
		t.Errorf("formats: %v", scriptFormatNames())
	}
	if scriptFormatElements("mine") == nil {
		t.Error("no elements for mine")
	}
	if !slices.Contains(scriptLanguages(), "xx") || !slices.Contains(scriptLanguages(), "en") {
		t.Errorf("languages: %v", scriptLanguages())
	}
}
