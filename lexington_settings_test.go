package main

import (
	"reflect"
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

// An alias is a format under a name of its own on an element set, with
// buttons of its own; TV series is one of Accolade's.
func TestFormatAliases(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Cleanup(func() {
		lexSettings, accoladeConf = readLexingtonSettings(), readAccoladeSettings()
		sceneConf = lexSettings.Scenes
	})
	text := defaultLexingtonSettings() + `
[Accolade.Formats.strip]
Name = "Strip (Franse stijl)"
Elements = "bd"

[[Accolade.Buttons.strip]]
Label = "Plaat"
Insert = "===\n\n.PLAAT {n}"
`
	if err := saveLexingtonSettings(text); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(scriptFormatNames(), "Strip (Franse stijl)") || !slices.Contains(scriptFormatNames(), "TV series") {
		t.Errorf("formats: %v", scriptFormatNames())
	}
	if elementsKey("strip") != "bd" || elementsKey("series") != "default" || elementsKey("comic") != "comic" {
		t.Errorf("elements: %s %s", elementsKey("strip"), elementsKey("series"))
	}
	if b := formatButtons("strip"); len(b) != 1 || b[0].Label != "Plaat" {
		t.Errorf("strip buttons: %v", b)
	}
	if b := formatButtons("series"); len(b) == 0 || b[0].Label != "Cold Open" {
		t.Errorf("series buttons: %v", b)
	}
}

// The defaults to start from have Accolade's part, and read back as they
// are built in.
func TestAccoladeDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Cleanup(func() {
		lexSettings, accoladeConf = readLexingtonSettings(), readAccoladeSettings()
		sceneConf = lexSettings.Scenes
	})
	if err := saveLexingtonSettings(defaultLexingtonSettings()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(accoladeConf.Buttons, defaultFormatButtons) || !reflect.DeepEqual(accoladeConf.Formats, defaultAliases) {
		t.Errorf("read back: %v", accoladeConf)
	}
}
