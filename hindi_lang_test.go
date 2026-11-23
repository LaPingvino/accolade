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

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"
)

// Hindi per script (Lexington's hi-Latn, hi-Deva, hi): named so in the
// pickers, found in a file's name in any case and by its fallback.
func TestHindiNames(t *testing.T) {
	for code, want := range map[string]string{"hi-Latn": "Hindi (Latin)", "hi-Deva": "Hindi (Devanagari)", "hi": "Hindi"} {
		if got := languageName(code); got != want {
			t.Errorf("languageName(%s) = %q, want %q", code, got, want)
		}
		label := strings.TrimSuffix(want, ")") + ", " + code + ")"
		if code == "hi" {
			label = "Hindi (hi)"
		}
		if got, want := languageLabel(code), label; got != want {
			t.Errorf("languageLabel(%s) = %q, want %q", code, got, want)
		}
		if !slices.Contains(scriptLanguages(), code) {
			t.Errorf("%s is not among the languages", code)
		}
	}
	for path, want := range map[string]string{
		"/s/film.hi-Latn.fountain": "hi-Latn", "/s/film.hi-latn.fountain": "hi-Latn", "/s/film.hi_Deva.fountain": "hi-Deva",
		"/s/film.hi-Deva-IN.fountain": "hi-Deva", "/s/film.HI.fountain": "hi", "/s/film.hi-Arab.fountain": "hi",
		"/s/film.de-CH.fountain": "de", "/s/film.xx-Yyyy.fountain": "", "/s/my.script.fountain": "",
	} {
		if got := languageOf(path); got != want {
			t.Errorf("languageOf(%s) = %q, want %q", path, got, want)
		}
	}
	if p := splitScriptName("film.hi-latn.fountain"); p.lang != "hi-Latn" || p.String() != "film.hi-Latn.fountain" {
		t.Errorf("splitScriptName: %+v %s", p, p)
	}
	if p := splitScriptName("film.EO.fountain"); p.lang != "eo" {
		t.Errorf("splitScriptName EO: %+v", p)
	}
	w := newTestWindow(t, "")
	if w.setLanguage("hi-deva"); w.language != "hi-Deva" {
		t.Errorf("setLanguage(hi-deva): %q", w.language)
	}
	if got := withLanguageName("/s/film.hi-Deva.fountain", "hi-Latn"); got != "/s/film.hi-Latn.fountain" {
		t.Errorf("withLanguageName: %s", got)
	}
	if _, lang, ok := cutLanguageLine(withLanguageLine("INT. ROOM - DAY\n", "hi-deva")); !ok || lang != "hi-Deva" {
		t.Errorf("language line: %q %v", lang, ok)
	}
}

// TestHindiNewTemplate: New in each Hindi opens its sample, and the template a Hindi without one would get is in its
// words that reads back as a screenplay in it: the title keys, the
// heading a heading, the cues cues (the one with Hindi's extension too),
// the closing transition a transition; a window in hi-Latn reads with
// hi-Latn's own rules, not English's with the same scene words.
func TestHindiNewTemplate(t *testing.T) {
	for lang, c := range map[string]struct {
		words     []string
		ext       string
		head, end string
	}{
		"hi-Latn": {[]string{"Sheershak: UNTITLED SCREENPLAY", "Shrey: Written by", "Masaude ki tithi: "}, "V.O.", "INT. LIVING ROOM - DIN", "FADE OUT."},
		"hi-Deva": {[]string{"शीर्षक: UNTITLED SCREENPLAY", "श्रेय: Written by", "लेखक: Your Name", "मसौदे की तिथि: ", "संपर्क:"}, "वॉयस ओवर", "अंदर. LIVING ROOM - दिन", "फ़ेड आउट"},
		"hi":      {[]string{"Title: UNTITLED SCREENPLAY", "Credit: Written by", "Author: Your Name", "Draft date: ", "Contact:"}, "V.O.", "INT. LIVING ROOM - DAY", "FADE OUT."},
	} {
		w := newTestWindow(t, "")
		w.setLanguage(lang)
		if w.language != lang {
			t.Fatalf("%s: the window's language is %q", lang, w.language)
		}
		if got, want := w.newScriptText(lang), languageSample(lang); want == "" || got != want {
			t.Errorf("%s: New is not its sample: %.40q", lang, got)
		}
		text := templateIn(w.lex().Language(lang))
		cue := "JOHN (" + c.ext + ")"
		for _, word := range append(c.words, c.head, cue, c.end) {
			if !strings.Contains(text, word) {
				t.Errorf("%s: no %q in\n%s", lang, word, text)
			}
		}
		types := map[string]string{}
		for _, l := range fountain.ParseIn(w.dialect().Lang, strings.NewReader(text)) {
			types[strings.TrimSpace(l.Contents)] = l.Type
		}
		for row, want := range map[string]string{c.head: lex.TypeScene, "JOHN": lex.TypeSpeaker, cue: lex.TypeSpeaker,
			"(sighing)": lex.TypeParen, "UNTITLED SCREENPLAY": "Title", "Written by": "Credit",
			"This screenplay isn't writing itself.": lex.TypeDialog, c.end: lex.TypeTrans} {
			if got := types[row]; got != want {
				t.Errorf("%s: %q read as %q, want %q", lang, row, got, want)
			}
		}
		if !isSceneHeadingIn(c.head, sceneStarts(lang)) {
			t.Errorf("%s: the editor does not see %q as a heading", lang, c.head)
		}
		if lang == "hi-Latn" && !slices.Contains(w.dialect().Lang.Times, "DIN") {
			t.Errorf("hi-Latn's window reads with %v's times", w.dialect().Lang.Times)
		}
	}
}

// devanagariScene is a short Hindi scene in Devanagari with conjuncts
// (क्षत्रिय, श्रद्धा, ज्ञान), matras before and after their consonant
// (कि, की), reph (धर्म) and eyelash ra (र्‍), for the screenshots.
const devanagariScene = "शीर्षक: क्षत्रिय\n\nशिक्षा अधिकारी का दफ़्तर (बाहरी कक्ष)/दिन/अंदर\n\n" +
	"रजनी कमरे में आती है। कि की कु कू के कै को कौ कं कः कँ\n\n" +
	"रजनी\n(धीरे से)\nक्षत्रिय का धर्म और कर्म, श्रद्धा और ज्ञान। र्‍य द्ध ट्र प्र\n\n" +
	"ANNA\nनमस्ते, हिन्दी।\n\nकट टू\n"

// TestDevanagariScreenshots renders a Devanagari scene in the editor, its
// page view and the PDF's page (SHOT_DIR as in screenshot_test.go).
func TestDevanagariScreenshots(t *testing.T) {
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
	w, _ := newDialogTestWindow(t)
	w.fyneWindow.Resize(fyne.NewSize(1100, 750))
	w.setLanguage("hi-Deva")
	w.textEditor.SetText(devanagariScene)
	uiSleep(200 * time.Millisecond)
	shot(w, "hi-deva-editor")
	pdfPath := filepath.Join(dir, "hi-deva.pdf")
	if err := exportAs(w.textEditor.Text(), pdfPath, "pdf", w.layoutJob()); err != nil {
		t.Fatal(err)
	}
	if b, err := exec.Command("pdftoppm", "-png", "-r", "150", pdfPath, filepath.Join(dir, "hi-deva-pdf")).CombinedOutput(); err != nil {
		t.Errorf("pdftoppm: %v %s", err, b)
	}
	t.Logf("PDF warning: %q", pdfWarning(w.textEditor.Text(), w.lex().Language("hi-Deva")))
	v := previewAndWait(t, w)
	v.toAnchor(1, 0) // the script's first page
	uiSleep(100 * time.Millisecond)
	shot(w, "hi-deva-pages")
}

// The pickers show Hindi, its variants (hi-Latn, hi-Deva) behind More
// variants…, a variant picked shown too; every other language as before.
func TestLanguageVariantsHidden(t *testing.T) {
	all := scriptLanguages()
	shown, hidden := shownLanguages(all, "")
	if !slices.Contains(shown, "hi") || slices.Contains(shown, "hi-Latn") || slices.Contains(shown, "hi-Deva") ||
		!slices.Equal(hidden, []string{"hi-Deva", "hi-Latn"}) && !slices.Equal(hidden, []string{"hi-Latn", "hi-Deva"}) ||
		len(shown)+len(hidden) != len(all) {
		t.Errorf("shown %v, hidden %v", shown, hidden)
	}
	if shown, _ := shownLanguages(all, "hi-Deva"); !slices.Contains(shown, "hi-Deva") || slices.Contains(shown, "hi-Latn") {
		t.Errorf("hi-Deva picked: shown %v", shown)
	}
	w := newTestWindow(t, "")
	t.Cleanup(func() { w.settings.SetString("script-language", "en") })
	menu := newLanguageCommand().submenu(w)
	var top []string
	var more *fyne.Menu
	for _, it := range menu.Items {
		top = append(top, it.Label)
		if it.Label == moreVariants {
			more = it.ChildMenu
		}
	}
	if !slices.Contains(top, "Hindi (hi)") || slices.Contains(top, languageLabel("hi-Deva")) || more == nil || len(more.Items) != 2 {
		t.Fatalf("New in: %q, more %v", top, more)
	}
	pickNewIn(t, w, "hi-Deva")
	if w.newFileLanguage() != "hi-Deva" {
		t.Errorf("More variants > Hindi (Devanagari): new in %s", w.newFileLanguage())
	}
	// a Select: More variants… shows them all, the others pick as before
	got := ""
	sel := widget.NewSelect(nil, func(v string) { got = v })
	offerVariants(sel, shown, all)
	if slices.Contains(sel.Options, "hi-Latn") || sel.Options[len(sel.Options)-1] != moreVariants {
		t.Fatalf("options %v", sel.Options)
	}
	sel.SetSelected(moreVariants)
	if got != "" || !slices.Equal(sel.Options, all) || sel.Selected != "" {
		t.Errorf("More variants…: got %q, options %v, selected %q", got, sel.Options, sel.Selected)
	}
	sel.SetSelected("hi-Latn")
	if got != "hi-Latn" {
		t.Errorf("picked %q", got)
	}
	sel2 := widget.NewSelect(nil, nil)
	offerVariants(sel2, shown, all)
	if selectOption(sel2, "hi-Deva", all); sel2.Selected != "hi-Deva" {
		t.Errorf("selectOption: %q in %v", sel2.Selected, sel2.Options)
	}
}
