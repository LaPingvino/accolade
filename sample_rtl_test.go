package main

import (
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode"

	"fyne.io/fyne/v2"
	"golang.org/x/image/font/sfnt"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
	lexfont "github.com/LaPingvino/lexington/font"
	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"
)

// What each right-to-left sample's lines read as: its headings, cues and
// transitions, the rest action, dialogue, parentheticals.
var langSampleWants = map[string]struct{ scenes, cues, transitions []string }{
	"he": {
		scenes:      []string{"1. פנים. ספרייה ישנה – לילה.", "2. חוץ. גג הבניין – יום."},
		cues:        []string{"רות", "דני (O.S.)", "דני", "רות", "דני", "דני", "רות", "דני"},
		transitions: []string{"פייד אין:", "קאט ל:", "פייד אאוט"},
	},
	"ar": {
		scenes:      []string{"خارجي - أمام صخرة كبيرة في الجبل - نهار", "مشهد 2  المغارة  نهار/داخلي"},
		cues:        []string{"@الزعيم", "الزعيم", "الزعيم", "@علي بابا", "علي بابا", "علي بابا"},
		transitions: []string{"قطع إلى:", "تلاشي إلى أسود"},
	},
}

// langSampleWindow is a window with lang's sample opened by its File
// menu item.
func langSampleWindow(t *testing.T, lang string) *MainWindow {
	t.Helper()
	w, _ := newDialogTestWindow(t)
	w.openLanguageSample(lang)
	if w.language != lang || w.textEditor.Text() != languageSample(lang) {
		t.Fatalf("%s sample: language %q, text %.40q", lang, w.language, w.textEditor.Text())
	}
	return w
}

// readBack is the script in w read in its language: its headings, cues
// and transitions as the editor reads them, each checked against
// Lexington's reading.
func readBack(t *testing.T, w *MainWindow) (scenes, cues, transitions []string) {
	t.Helper()
	lang, text := w.language, w.textEditor.Text()
	lines := strings.Split(text, "\n")
	kinds := syntax.ClassifyIn(lines, w.sceneStarts(), w.textEditor.Dialect)
	lexKinds := map[int]string{}
	for _, r := range fountain.ParseLinesIn(w.lex().Language(lang), strings.NewReader(text)) {
		if r.Type != lex.TypeEmpty { // the file's end has an empty row on its last line
			lexKinds[r.Source-1] = r.Type
		}
	}
	for i, k := range kinds {
		switch k {
		case syntax.SceneHeading:
			scenes = append(scenes, lines[i])
		case syntax.Character:
			cues = append(cues, lines[i])
		case syntax.Transition:
			transitions = append(transitions, lines[i])
		}
		lk := map[syntax.Kind]string{syntax.SceneHeading: lex.TypeScene, syntax.Character: lex.TypeSpeaker, syntax.Transition: lex.TypeTrans}[k]
		if got := lexKinds[i]; (lk != "" || got == lex.TypeScene || got == lex.TypeSpeaker || got == lex.TypeTrans) && got != lk {
			t.Errorf("%s line %d %q: editor %q, Lexington %q", lang, i+1, lines[i], lk, got)
		}
	}
	if s := fountain.ParseIn(w.lex().Language(lang), strings.NewReader(text)); len(s) == 0 || s[0].Type != lex.TypeTitlePage {
		t.Errorf("%s: no title page", lang)
	}
	return scenes, cues, transitions
}

// Each sample reads in its language as written: the editor and
// Lexington agree, and its headings are headings, its cues cues, its
// transitions transitions.
func TestLanguageSamplesRead(t *testing.T) {
	for lang, want := range langSampleWants {
		w := langSampleWindow(t, lang)
		scenes, cues, transitions := readBack(t, w)
		if !slices.Equal(scenes, want.scenes) {
			t.Errorf("%s headings: %q", lang, scenes)
		}
		if !slices.Equal(cues, want.cues) {
			t.Errorf("%s cues: %q", lang, cues)
		}
		if !slices.Equal(transitions, want.transitions) {
			t.Errorf("%s transitions: %q", lang, transitions)
		}
		// printed by Lexington's PDF writer, not a browser
		if msg := pdfWarning(w.textEditor.Text(), w.lex().Language(lang)); msg != "" {
			t.Errorf("%s PDF warning: %s", lang, msg)
		}
	}
}

// New in every language of the configuration (picked on New in:) starts
// a script in it that reads in it: its heading a heading, its cues cues,
// its transition a transition.
func TestNewInEveryLanguage(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	t.Cleanup(func() { w.settings.SetString("script-language", "en") })
	langs := w.lex().languages()
	if len(langs) < 10 {
		t.Fatalf("languages: %v", langs)
	}
	for _, lang := range langs {
		w.changeSetting("script-language", func(s *Settings) { s.SetString("script-language", lang) })
		w.newLanguage = lang
		w.NewFile()
		if w.language != lang {
			t.Errorf("New in %s: the script is in %s", lang, w.language)
			continue
		}
		scenes, cues, transitions := readBack(t, w)
		if len(scenes) < 1 || len(cues) < 1 || len(transitions) < 1 {
			t.Errorf("New in %s: headings %q, cues %q, transitions %q\n%s", lang, scenes, cues, transitions, w.textEditor.Text())
		}
	}
}

// Script Language Hebrew, New, Script Language English, New: the new
// script is English's, whatever the preferences say (Joop: "it showed
// Esperanto").
func TestNewKeepsLanguagePicked(t *testing.T) {
	for _, pref := range []string{"en", "eo"} {
		w, _ := newDialogTestWindow(t)
		w.settings.SetString("script-language", pref)
		w.changeLanguage("he", markLanguage, nil)
		w.hasChanges = false // New asks first otherwise
		w.NewFile()
		if w.language != "he" || w.textEditor.Text() != languageSample("he") {
			t.Errorf("pref %s, Hebrew, New: %s, %.30q", pref, w.language, w.textEditor.Text())
		}
		w.changeLanguage("en", markLanguage, nil)
		w.hasChanges = false
		w.NewFile()
		title := w.fyneWindow.Title()
		if w.language != "en" || w.textEditor.Text() != newScriptTemplate() ||
			strings.Contains(title, ".eo.") || strings.Contains(title, ".he.") {
			t.Errorf("pref %s, English, New: %s, title %q, %.30q", pref, w.language, title, w.textEditor.Text())
		}
		if w.languageSaid != (pref != "en") {
			t.Errorf("pref %s: English said %v, title %q", pref, w.languageSaid, title)
		}
		w.settings.SetString("script-language", "en")
	}
}

// The Hebrew sample has the points Courier Badi places by presentation
// form (shin and sin dots, holam, dagesh in a narrow letter), the Arabic
// one its vowels, sukun and shadda; Courier Badi has every character of
// both.
func TestLanguageSamplesMarks(t *testing.T) {
	he, ar := languageSample("he"), languageSample("ar")
	for _, r := range []rune{0x05C1, 0x05C2, 0x05B9, 0x05BC, 0x05BE} {
		if !strings.ContainsRune(he, r) {
			t.Errorf("Hebrew sample without %U", r)
		}
	}
	if !strings.Contains(he, "וּ") && !strings.Contains(he, "יּ") {
		t.Error("Hebrew sample without a dagesh in vav or yod")
	}
	for _, r := range []rune{0x064E, 0x064F, 0x0650, 0x0651, 0x0652} {
		if !strings.ContainsRune(ar, r) {
			t.Errorf("Arabic sample without %U", r)
		}
	}
	if !strings.Contains(ar, "اِفْتَحْ يَا سِمْسِمْ") {
		t.Error("Arabic sample without its key line vowelled")
	}
	f, err := sfnt.Parse(lexfont.CourierBadiRegular)
	if err != nil {
		t.Fatal(err)
	}
	var buf sfnt.Buffer
	for _, r := range he + ar {
		if r == '\n' || unicode.IsSpace(r) {
			continue
		}
		if g, err := f.GlyphIndex(&buf, r); err != nil || g == 0 {
			t.Errorf("Courier Badi has no %q (%U)", r, r)
		}
	}
}

// New starts from the sample in the language for new scripts.
func TestNewInLanguageSample(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	for _, lang := range []string{"he", "ar", "en"} {
		w.settings.SetString("script-language", lang)
		w.newLanguage = lang // New in:
		w.NewFile()
		want := orTemplate(languageSample(lang))
		if w.language != lang || w.textEditor.Text() != want {
			t.Errorf("New in %s: language %q, text %.40q", lang, w.language, w.textEditor.Text())
		}
	}
	w.settings.SetString("script-language", "en")
}

// orTemplate is sample, or else the new script template.
func orTemplate(sample string) string {
	if sample != "" {
		return sample
	}
	return newScriptTemplate()
}

// TestLanguageSampleScreenshots renders each sample in the editor and
// its page view, and the PDF's first page with pdftoppm (SHOT_DIR as in
// screenshot_test.go).
func TestLanguageSampleScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	shot := func(w *MainWindow, name string) {
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
	// New in Esperanto: the template in its words
	eo, _ := newDialogTestWindow(t)
	eo.fyneWindow.Resize(fyne.NewSize(1100, 750))
	eo.newLanguage = "eo"
	eo.NewFile()
	shot(eo, "new-eo-editor")
	for _, lang := range []string{"he", "ar"} {
		w := langSampleWindow(t, lang)
		w.fyneWindow.Resize(fyne.NewSize(1100, 750))
		shot(w, "sample-"+lang+"-editor")
		v := previewAndWait(t, w)
		shot(w, "sample-"+lang+"-pages-1")
		v.toAnchor(1, 0) // the script's first page
		uiSleep(100 * time.Millisecond)
		shot(w, "sample-"+lang+"-pages-2")
		pdfPath := filepath.Join(dir, "sample-"+lang+".pdf")
		job := w.layoutJob()
		if err := exportAs(w.textEditor.Text(), pdfPath, "pdf", job); err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(dir, "sample-"+lang+"-pdf")
		if b, err := exec.Command("pdftoppm", "-png", "-r", "150", pdfPath, out).CombinedOutput(); err != nil {
			t.Errorf("pdftoppm: %v %s", err, b)
		}
	}
}
