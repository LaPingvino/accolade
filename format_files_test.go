package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/rules"
)

// Formats of one's own: made from another (its buttons with it), changed
// copies of built-in ones, kept or forgotten, and exported with.
func TestFormatFiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	key, err := newFormat("Strip (Franse stijl)", "bd")
	if err != nil || key != "strip-franse-stijl" {
		t.Fatalf("new: %q %v", key, err)
	}
	bd, _ := rules.GetPreset("bd")
	if f := scriptFormat(key); f.Name != "Strip (Franse stijl)" || f.Elements["speaker"] != bd.Elements["speaker"] {
		t.Errorf("made: %+v", f)
	}
	if b := formatButtons(key); len(b) == 0 || b[0].Label != formatButtons("bd")[0].Label {
		t.Errorf("buttons: %v", b)
	}
	if _, err := newFormat("strip (franse stijl)", "bd"); err == nil {
		t.Error("the same name twice")
	}

	text := formatText("comic")
	if !strings.Contains(text, `Base = "comic"`) || !strings.Contains(text, "# [Elements.") || !strings.Contains(text, "Panel") {
		t.Errorf("comic to edit:\n%s", text)
	}
	if err := saveFormat("comic", "Name = \"Comics\"\nBase = \"comic\"\n"); err != nil {
		t.Fatal(err)
	}
	if !userFormat("comic") || scriptFormat("comic").Name != "Comics" || len(formatButtons("comic")) != 0 {
		t.Errorf("changed comic: %q, %d buttons", scriptFormat("comic").Name, len(formatButtons("comic")))
	}
	if err := removeFormat("comic"); err != nil || scriptFormat("comic").Name == "Comics" || len(formatButtons("comic")) == 0 {
		t.Errorf("built-in again: %q %v", scriptFormat("comic").Name, err)
	}

	if err := saveFormat("x", "Base = \"nothing\"\n"); err == nil {
		t.Error("an unknown Base")
	}
	if err := saveFormat("x", "Name = \n"); err == nil {
		t.Error("TOML that is not")
	}

	out := filepath.Join(t.TempDir(), "s.pdf")
	if err := exportAs("INT. HOUSE - DAY\n\nRain.\n", out, "pdf", exportJob{format: key}); err != nil {
		t.Errorf("export with one's own format: %v", err)
	}
}
