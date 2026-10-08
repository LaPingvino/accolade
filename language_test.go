package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const esperanto = "Title: LA BISKVITO\n\nEN. KUIREJO - TAGO\n\nAnna rigardas.\n\nANNA\nSaluton.\n\nEKST. ĜARDENO - NOKTO\n"

func TestLanguageOf(t *testing.T) {
	for path, want := range map[string]string{
		"scene.eo.fountain": "eo", "/a/b/film.NL.fountain": "nl", "scene.fountain": "",
		"scene.xx.fountain": "", "eo.fountain": "",
	} {
		if got := languageOf(path); got != want {
			t.Errorf("languageOf(%q) = %q, want %q", path, got, want)
		}
	}
	if !isSceneHeadingIn("en. kuirejo - tago", sceneStarts("eo")) || isSceneHeadingIn("INT. HOUSE", sceneStarts("eo")) {
		t.Error("Esperanto scene headings")
	}
}

// An Esperanto script (scene.eo.fountain) has its scene headings
// everywhere: the editor's formatting, the outline, completion, the
// statistics, the preview and the exports.
func TestEsperantoScript(t *testing.T) {
	path := filepath.Join(t.TempDir(), "biskvito.eo.fountain")
	os.WriteFile(path, []byte(esperanto), 0o644)
	w, _ := newDialogTestWindow(t)
	if err := w.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	if w.language != "eo" {
		t.Fatalf("language %q", w.language)
	}
	if got, kind := formatCompletedLine("", "ekst. strato - tago", w.sceneStarts()); got != "EKST. STRATO - TAGO" || kind != "Scene Heading" {
		t.Errorf("Enter on a heading: %q %s", got, kind)
	}
	if items := outlineOf(w.textEditor.Text(), w.sceneStarts()); len(items) != 2 || items[1].Text != "EKST. ĜARDENO - NOKTO" {
		t.Errorf("outline %+v", items)
	}
	if st := computeStats(w.textEditor.Text(), 0, w.sceneStarts()); st.Scenes != 2 {
		t.Errorf("%d scenes", st.Scenes)
	}
	if c := completeAt(esperanto+"\nEN", len([]rune(esperanto))+3, w.sceneStarts()); c == nil || c.candidates[0] != "EN. KUIREJO - TAGO" {
		t.Errorf("completion %+v", c)
	}
	s := exportJob{scenes: sceneHeaders(w.language)}.script(w.textEditor.Text())
	scenes := 0
	for _, l := range s {
		if l.Type == "scene" {
			scenes++
		}
	}
	if scenes != 2 {
		t.Errorf("exported scenes: %d", scenes)
	}
	w.showPreview()
	if p := previewText(w.previewArea.Segments); !strings.Contains(p, "EN. KUIREJO - TAGO") {
		t.Errorf("preview:\n%s", p)
	}
	// a new script has the language of the preferences
	w.settings.SetString("script-language", "nl")
	t.Cleanup(func() { w.settings.SetString("script-language", "en") })
	w2, _ := newDialogTestWindow(t)
	if w2.language != "nl" {
		t.Errorf("new script: %q", w2.language)
	}
}

func TestNewScriptTemplate(t *testing.T) {
	tpl := newScriptTemplate()
	if strings.Contains(tpl, "\n\n\n") || strings.Contains(tpl, "\t") {
		t.Errorf("template has runs of blank lines or tabs:\n%s", tpl)
	}
	if st := computeStats(tpl, 0, nil); st.Scenes != 1 {
		t.Errorf("template scenes: %d", st.Scenes)
	}
}
