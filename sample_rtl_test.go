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

// What each language's sample reads as: its headings, cues and
// transitions, the rest action, dialogue, parentheticals. Every sample
// (examples/sample.<lang>.fountain) has its line here.
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
	"eo": {
		scenes:      []string{"EKST. BULONJO-ĈE-MARO, ANTAŬ LA TEATRO - TAGO", "EN. MALANTAŬ LA SCENEJO - VESPERO"},
		cues:        []string{"MISTER BROWN", "MARIE", "MISTER BROWN", "PJOTR", "KLARA", "ZAMENHOF", "KLARA", "VOĈO (O.S.)", "ZAMENHOF"},
		transitions: []string{"> TRANĈO.", "> MALLUMIĜO."},
	},
	"de": {
		scenes:      []string{"1. LANDSTRASSE NACH BREMEN AUSSEN/TAG", "2. VOR DEM RÄUBERHAUS IM WALD AUSSEN/NACHT"},
		cues:        []string{"ESEL", "HUND", "ESEL", "HUND", "ESEL", "KATZE (OFF)", "HAHN", "ESEL"},
		transitions: []string{"SCHNITT AUF:", "> ABBLENDE."},
	},
	"fr": {
		scenes:      []string{"1 - FORÊT, AU PIED D'UN VIEUX CHÊNE - EXT/JOUR"},
		cues:        []string{"RENARD", "RENARD (SUITE)", "RENARD", "UNE PIE (OFF)", "RENARD", "CORBEAU"},
		transitions: []string{"FONDU AU NOIR"},
	},
	"it": {
		scenes:      []string{"1. INT. BOTTEGA DI GEPPETTO - GIORNO", "2. EST. STRADA DEL PAESE - GIORNO"},
		cues:        []string{"GEPPETTO", "GEPPETTO (CONTINUA)", "PINOCCHIO", "GEPPETTO", "PINOCCHIO", "MASTRO CILIEGIA (VOCE FUORI CAMPO)", "GEPPETTO", "GEPPETTO (CONTINUA)"},
		transitions: []string{"STACCO SU:"},
	},
	"nl": {
		scenes:      []string{"BIN. ATELIER VAN REMBRANDT, SINT ANTHONIESBREESTRAAT - DAG"},
		cues:        []string{"BANNINCK COCQ", "REMBRANDT", "SCHUTTER (OFF)", "REMBRANDT", "BANNINCK COCQ", "REMBRANDT", "REMBRANDT (VERVOLGT)"},
		transitions: []string{"> LANGZAAM NAAR ZWART."},
	},
	"ru": {
		scenes:      []string{"1. НАТ. ЛЕСНАЯ ПОЛЯНА ПОД ЛИПАМИ. ДЕНЬ"},
		cues:        []string{"МАРТЫШКА", "ОСЁЛ", "КОЗЁЛ", "МИШКА", "СОЛОВЕЙ (ЗА КАДРОМ)", "МАРТЫШКА"},
		transitions: []string{"> ЗАТЕМНЕНИЕ."},
	},
	"tok": {
		scenes:      []string{"KON. NASIN LON POKA PI MA PAN - TENPO SELI", "INSA. TOMO PI PIPI PALI - TENPO LETE"},
		cues:        []string{"PIPI KALAMA", "PIPI PALI", "PIPI KALAMA", "PIPI KALAMA (O.S.)", "PIPI PALI"},
		transitions: []string{"> TENPO LI TAWA.", "> PINI."},
	},
	"zh": {
		scenes:      []string{"场景1 宋国田边 日 外", "场景2 宋国田边 昏 外"},
		cues:        []string{"农夫", "邻居（画外音）", "农夫", "邻居", "农夫", "邻居"},
		transitions: []string{"淡入：", "淡出"},
	},
	"hi-Deva": {
		scenes: []string{"1. नदी के बीच मगरमच्छ का घर/शाम/बाहर", "2. नदी के बीच/दिन/बाहर", "3. नदी किनारे जामुन का पेड़/दिन/बाहर"},
		cues: []string{"पत्नी (ऑफ़ स्क्रीन)", "मगरमच्छ", "पत्नी", "मगरमच्छ", "पत्नी", "मगरमच्छ",
			"बंदर", "बंदर", "बंदर (जारी)", "कथावाचक (वॉयस ओवर)"},
		transitions: []string{"फ़ेड इन", "कट टू", "कट टू", "फ़ेड आउट"},
	},
	"hi": {
		scenes: []string{"EXT. MIDDLE OF THE RIVER, THE CROCODILE'S HOME - EVENING", "EXT. MIDDLE OF THE RIVER - DAY", "EXT. RIVERBANK, JAMUN TREE - DAY"},
		cues: []string{"PATNI (O.S.)", "MAGARMACHH", "PATNI", "MAGARMACHH", "PATNI", "MAGARMACHH",
			"BANDAR", "BANDAR", "BANDAR (CONT'D)", "KATHAVAACHAK (V.O.)"},
		transitions: []string{"CUT TO:", "CUT TO:", "> FADE OUT."},
	},
	"hi-Latn": {
		scenes: []string{"EXT. NADI KE BEECH, MAGARMACHH KA GHAR - SHAAM", "EXT. NADI KE BEECH - DIN", "EXT. NADI KINAARA, JAAMUN KA PED - DIN"},
		cues: []string{"PATNI (O.S.)", "MAGARMACHH", "PATNI", "MAGARMACHH", "PATNI", "MAGARMACHH",
			"BANDAR", "BANDAR", "BANDAR (CONT'D)", "KATHAVAACHAK (V.O.)"},
		transitions: []string{"CUT TO:", "CUT TO:", "> FADE OUT."},
	},
	"ja": {
		scenes:      []string{"○川辺（朝）", "○おじいさんとおばあさんの家・囲炉裏端（夕）"},
		cues:        []string{"おばあさん", "おじいさん", "おばあさん", "おじいさん", "おばあさん", "語り手（N）"},
		transitions: []string{"F・I", "F・O"},
	},
}

// noPDFFont are the samples whose letters Lexington's PDF font (Courier
// Badi) lacks: the PDF says so, and HTML prints them.
var noPDFFont = map[string]bool{"zh": true, "ja": true}

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

// titlePage is a script's title page as Lexington reads it in its
// language: each standard key's value.
func titlePage(w *MainWindow) map[string]string {
	got := map[string]string{}
	for _, r := range fountain.ParseIn(w.lex().Language(w.language), strings.NewReader(w.textEditor.Text())) {
		if r.Type == lex.TypeScene {
			break
		}
		if r.Contents != "" && got[r.Type] == "" {
			got[r.Type] = strings.TrimSpace(r.Contents)
		}
	}
	return got
}

// Each sample reads in its language as written: the editor and
// Lexington agree, and its headings are headings, its cues cues, its
// transitions transitions.
func TestLanguageSamplesRead(t *testing.T) {
	langs := sampleLanguages()
	if len(langs) != len(langSampleWants) {
		t.Errorf("samples %v, wants for %d", langs, len(langSampleWants))
	}
	for _, lang := range langs {
		want, ok := langSampleWants[lang]
		if !ok {
			t.Errorf("%s: a sample with no wants", lang)
			continue
		}
		w := langSampleWindow(t, lang)
		// written in the language's own title page keys, read as the
		// standard ones
		tp := titlePage(w)
		if tp["Title"] == "" || tp["Title"] != langSamples()[lang].title || tp["Author"] != "Accolade" ||
			tp["Draft date"] == "" && tp["Date"] == "" {
			t.Errorf("%s title page: %q", lang, tp)
		}
		first, _, _ := strings.Cut(w.textEditor.Text(), ":")
		if lexKey := w.lex().Language(lang).TitleKeys[first]; lexKey != "Title" {
			t.Errorf("%s: the title's key %q is not the language's own", lang, first)
		}
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
		if msg := pdfWarning(w.textEditor.Text(), w.lex().Language(lang)); (msg != "") != noPDFFont[lang] {
			t.Errorf("%s PDF warning: %q", lang, msg)
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
		pickNewIn(t, w, lang)
		w.NewFile()
		if w.language != lang {
			t.Errorf("New in %s: the script is in %s", lang, w.language)
			continue
		}
		scenes, cues, transitions := readBack(t, w)
		if s, ok := langSampleWants[lang]; ok {
			if !slices.Equal(scenes, s.scenes) || !slices.Equal(cues, s.cues) || !slices.Equal(transitions, s.transitions) {
				t.Errorf("New in %s: not its sample", lang)
			}
			continue
		}
		// every line written as a heading, cue or transition reads as one
		lines := strings.Split(w.textEditor.Text(), "\n")
		var wantCues []string
		for i, l := range lines {
			if (l == "JOHN" || strings.HasPrefix(l, "JOHN (")) && i > 0 && lines[i-1] == "" {
				wantCues = append(wantCues, l)
			}
		}
		last := lines[len(lines)-2]
		if len(scenes) != 1 || !strings.Contains(scenes[0], "LIVING ROOM") || !slices.Equal(cues, wantCues) || len(wantCues) < 1 ||
			!slices.Equal(transitions, []string{last}) {
			t.Errorf("New in %s: headings %q, cues %q (want %q), transitions %q (want %q)\n%s", lang, scenes, cues, wantCues, transitions, last, w.textEditor.Text())
		}
	}
}

// pickNewIn picks lang on File > New in:, as the menu does.
func pickNewIn(t *testing.T, w *MainWindow, lang string) {
	t.Helper()
	for _, it := range flatItems(newLanguageCommand().submenu(w).Items) { // a variant's under More variants…
		if it.Label == languageLabel(lang) {
			it.Action()
			return
		}
	}
	t.Fatalf("New in: has no %s", lang)
}

// newInShown checks that everything showing New's or the script's
// language shows lang: the script, the ribbon's New in:, its check, the
// place button, the title.
func newInShown(t *testing.T, w *MainWindow, lang, what string) {
	t.Helper()
	label := newLanguageCommand().ribbonLabelFor(w)
	checked := ""
	for _, it := range flatItems(newLanguageCommand().submenu(w).Items) {
		if it.Checked {
			checked += it.Label
		}
	}
	_, _, place, _ := placeWords(w.lex().headers(lang))
	title := w.fyneWindow.Title()
	if w.language != lang || !strings.HasSuffix(label, languageName(lang)) || checked != languageLabel(lang) ||
		w.headerBar.place.Text != place {
		t.Errorf("%s: script %s, ribbon %q, checked %q, place %q (want %s, %q)", what, w.language, label, checked, w.headerBar.place.Text, lang, place)
	}
	for _, other := range []string{"eo", "he", "ar"} {
		if other != lang && strings.Contains(title, "."+other+".") {
			t.Errorf("%s: title %q", what, title)
		}
	}
}

// New in: Hebrew, New, New in: English, New, with the preferences in
// Esperanto too (Joop: "it showed Esperanto"): English everywhere.
func TestNewInPickedShown(t *testing.T) {
	for _, pref := range []string{"en", "eo"} {
		w, _ := newDialogTestWindow(t)
		w.settings.SetString("script-language", pref)
		w.setLanguage("")
		pickNewIn(t, w, "he")
		w.NewFile()
		newInShown(t, w, "he", pref+": New in Hebrew")
		if w.textEditor.Text() != languageSample("he") {
			t.Errorf("%s: New in Hebrew is not its sample", pref)
		}
		pickNewIn(t, w, "en")
		w.hasChanges = false
		w.NewFile()
		newInShown(t, w, "en", pref+": New in English")
		if w.textEditor.Text() != newScriptTemplate() {
			t.Errorf("%s: New in English is not the template", pref)
		}
		w.settings.SetString("script-language", "en")
	}
}

// A script New made in Hebrew, saved, opens in Hebrew again whatever the
// language for new scripts is by then.
func TestNewLanguageKeptOnSave(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	t.Cleanup(func() { w.settings.SetString("script-language", "en") })
	pickNewIn(t, w, "he")
	w.NewFile()
	p := filepath.Join(t.TempDir(), "script.fountain")
	if err := w.saveToFile(p); err != nil {
		t.Fatal(err)
	}
	pickNewIn(t, w, "ar")
	w.hasChanges = false
	w.NewFile()
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if w.language != "he" {
		t.Errorf("reopened in %s", w.language)
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
		newInShown(t, w, "en", pref+": Script Language English, New")
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
	for _, lang := range []string{"he", "ar", "hi-Deva", "hi-Latn", "hi", "en"} {
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
	// the others in the editor
	for _, lang := range []string{"eo", "zh", "de", "ru", "ja", "hi-Latn"} {
		w := langSampleWindow(t, lang)
		w.fyneWindow.Resize(fyne.NewSize(1100, 750))
		shot(w, "sample-"+lang+"-editor")
	}
	// and with their page views and PDFs: the right-to-left ones, and
	// Devanagari in Lekhan monospace (a conjunct two cells, a simple
	// cluster one)
	for _, lang := range []string{"he", "ar", "hi-Deva"} {
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
