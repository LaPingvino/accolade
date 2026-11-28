// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

// hebrewNeutral is a Hebrew script with lines of no direction of their
// own ("…", "2.") between Hebrew lines, and one in Latin letters.
const hebrewNeutral = `פנים. בית - יום

דני נכנס לחדר.

…

רות מחכה ליד החלון.

2.

Latin letters keep their own direction.

דני
שלום.
`

// TestDirectionFallbackScreenshot renders a Hebrew script whose neutral
// lines take the language's direction (SHOT_DIR as in screenshot_test.go;
// SHOT_NAME names the file, "rtlfallback-after" by default).
func TestDirectionFallbackScreenshot(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	name := os.Getenv("SHOT_NAME")
	if name == "" {
		name = "rtlfallback-after"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	w, _ := newDialogTestWindow(t)
	w.fyneWindow.Resize(fyne.NewSize(1000, 650))
	p := filepath.Join(t.TempDir(), "script.he.fountain")
	if err := os.WriteFile(p, []byte(hebrewNeutral), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	w.fyneWindow.Content().Refresh()
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
		t.Fatal(err)
	}
}

// A script's language sets the direction of its lines with none of their
// own: the editor's, Right to Left Line's, the HTML export's. Every other
// line keeps its own.
func TestDirectionFallback(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	dir := t.TempDir()
	p := filepath.Join(dir, "script.he.fountain")
	os.WriteFile(p, []byte(hebrewNeutral), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	e := w.textEditor
	if !e.LanguageRTL || !e.LineRTL("…") || e.LineRTL("Latin letters keep their own direction.") {
		t.Fatalf("Hebrew: language rtl %v", e.LanguageRTL)
	}
	// Right to Left Line on "…" (right to left already): a mark turns it
	at := strings.Index(hebrewNeutral, "…")
	e.Buffer().SetCursor(len([]rune(hebrewNeutral[:at])), false)
	toggleDirection(e)
	line := func() string { return strings.Split(e.Text(), "\n")[4] }
	if line() != "\u200E…" {
		t.Errorf("to left to right: %q", line())
	}
	toggleDirection(e)
	if line() != "…" {
		t.Errorf("back: %q", line())
	}
	// the HTML export
	job := w.layoutJob()
	job.format = "default"
	out := filepath.Join(dir, "s.html")
	if err := exportAs(e.Text(), out, "html", job); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(out)
	if html := string(b); !strings.Contains(html, `dir="rtl">…<`) || !strings.Contains(html, `dir="auto">Latin letters`) {
		t.Errorf("HTML:\n%s", html)
	}

	// English: left to right
	en := filepath.Join(dir, "script.fountain")
	os.WriteFile(en, []byte("INT. HOUSE - DAY\n\n…\n"), 0o644)
	if err := w.LoadFile(en); err != nil {
		t.Fatal(err)
	}
	if e.LanguageRTL || e.LineRTL("…") {
		t.Error("English: right to left")
	}
}

// A language of one's own that runs right to left travels with an
// .accolade: opened where it is unknown it reads right to left, and Add
// to My Settings writes its direction too.
func TestDirectionCarried(t *testing.T) {
	w := draftsWindow(t)
	conf := scratchSettings(t)
	os.MkdirAll(conf, 0o755)
	os.WriteFile(filepath.Join(conf, "lexington.toml"),
		[]byte("[Scenes]\nxr = [\"ENDE\", \"AUSE\"]\n\n[Directions]\nxr = \"rtl\"\n"), 0o644)
	reloadLexingtonSettings()
	dir := t.TempDir()
	plain := filepath.Join(dir, "film.xr.fountain")
	os.WriteFile(plain, []byte("ENDE. DOMO - TAGO\n\n…\n"), 0o644)
	if err := w.LoadFile(plain); err != nil {
		t.Fatal(err)
	}
	if w.language != "xr" || !w.textEditor.LanguageRTL {
		t.Fatalf("xr here: %q, rtl %v", w.language, w.textEditor.LanguageRTL)
	}
	if err := w.renameTo("film.accolade"); err != nil {
		t.Fatal(err)
	}
	w.flushBundleWrites()
	p := filepath.Join(dir, "film.accolade")

	// another machine, an empty settings folder
	w2 := draftsWindow(t)
	conf2 := scratchSettings(t)
	if err := w2.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if w2.language != "xr" || !w2.textEditor.LanguageRTL || lexSettings.RTL("xr") {
		t.Fatalf("opened elsewhere: %q, rtl %v", w2.language, w2.textEditor.LanguageRTL)
	}
	w2.addToMySettings()
	w2.adoptAsked.apply()
	b, _ := os.ReadFile(filepath.Join(conf2, "lexington.toml"))
	if !lexSettings.RTL("xr") || !strings.Contains(string(b), "[Languages.xr]") || !strings.Contains(string(b), `Direction = "rtl"`) {
		t.Errorf("added:\n%s", b)
	}
	// Hebrew is still right to left there (a language not named keeps
	// the default's)
	if !lexSettings.RTL("he") {
		t.Error("Hebrew turned around")
	}
}
