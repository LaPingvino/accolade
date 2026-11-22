package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/rules"
)

const chineseScript = `Title: 咖啡馆

场景1 咖啡馆 日 内

张三坐在窗边。

李四
（笑）
你来了。

张三（画外音）
我一直在这里。

李四
真的吗？

淡出

/* Characters
张三: 主角
*/
`

// The editor reads a script's lines as Lexington's parser does, by the
// language's rules: its headings (patterns too), cues, transitions.
func TestClassifyAgreesWithLexington(t *testing.T) {
	type sample struct {
		lang, text string
		min        int
	}
	samples := []sample{{"zh", chineseScript, 5}}
	files, _ := filepath.Glob("../lexington-langrules/testdata/hekaya/*.hekaya")
	for _, f := range files {
		b, _ := os.ReadFile(f)
		least := 10 // Hekaya's samples; its templates are short
		if strings.HasPrefix(filepath.Base(f), "template-") {
			least = 0
		}
		samples = append(samples, sample{rules.Hekaya, string(b), least})
	}
	if len(samples) < 4 {
		t.Fatalf("samples: %d", len(samples))
	}
	for _, s := range samples {
		lang := lexSettings.Language(s.lang)
		lines := strings.Split(s.text, "\n")
		kinds := syntax.ClassifyIn(lines, sceneStarts(s.lang), fountain.NewDialect(lang))
		want := map[int]string{}
		for _, r := range fountain.ParseLinesIn(lang, strings.NewReader(s.text)) {
			switch r.Type {
			case lex.TypeSpeaker, lex.TypeScene, lex.TypeTrans:
				want[r.Source-1] = r.Type
			}
		}
		of := map[syntax.Kind]string{syntax.Character: lex.TypeSpeaker, syntax.SceneHeading: lex.TypeScene, syntax.Transition: lex.TypeTrans}
		n := 0
		for i, k := range kinds {
			got := of[k]
			if got != want[i] {
				t.Errorf("%s line %d %q: editor %q, Lexington %q", s.lang, i+1, lines[i], got, want[i])
			}
			if got != "" {
				n++
			}
		}
		if n < s.min {
			t.Errorf("%s: only %d headings, cues and transitions", s.lang, n)
		}
	}
}

// Marking a line Character makes it one as the language reads it.
func TestMarkCharacterByRules(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.setLanguage("zh")
	w.textEditor.SetText("场景1 咖啡馆 日 内\n\n王五\n你好。\n")
	w.textEditor.SetCursorOffset(len([]rune("场景1 咖啡馆 日 内\n\n王五")))
	before := w.textEditor.Text()
	w.markLineElement(elementCharacter)
	text := w.textEditor.Text()
	if !strings.Contains(text, "\n王五\n你好。") || !isListed(text, "王五") {
		t.Fatalf("not listed:\n%s", text)
	}
	if k := syntax.ClassifyIn(strings.Split(text, "\n"), w.sceneStarts(), w.dialect())[2]; k != syntax.Character {
		t.Errorf("reads as %v", k)
	}
	w.textEditor.Undo()
	if w.textEditor.Text() != before {
		t.Errorf("one undo step:\n%s", w.textEditor.Text())
	}
	// a name the rules read already: nothing added
	w.textEditor.SetText("场景1 咖啡馆 日 内\n\n王五（画外音）\n你好。\n")
	w.textEditor.SetCursorOffset(len([]rune("场景1 咖啡馆 日 内\n\n王五")))
	w.markLineElement(elementCharacter)
	if strings.Contains(w.textEditor.Text(), "/* Characters") {
		t.Errorf("listed though read:\n%s", w.textEditor.Text())
	}
	// Hekaya: its registry is the script's @ uses
	w.setLanguage(rules.Hekaya)
	w.textEditor.SetText("داخلي - قهوة - نهار\n\nسمير\nأهلا.\n")
	w.textEditor.SetCursorOffset(len([]rune("داخلي - قهوة - نهار\n\nسمير")))
	w.markLineElement(elementCharacter)
	if !strings.Contains(w.textEditor.Text(), "\n@سمير\n") {
		t.Errorf("hekaya:\n%s", w.textEditor.Text())
	}
	// English: capitals, as before
	w.setLanguage("en")
	w.textEditor.SetText("INT. CAFE - DAY\n\nanna\nHi.\n")
	w.textEditor.SetCursorOffset(len([]rune("INT. CAFE - DAY\n\nanna")))
	w.markLineElement(elementCharacter)
	if !strings.Contains(w.textEditor.Text(), "\nANNA\n") {
		t.Errorf("en:\n%s", w.textEditor.Text())
	}
}

// A .hekaya file is Hekaya's: its own type, read with Hekaya's rules, its
// Arabic printed by Lexington's PDF (no browser).
func TestHekayaFile(t *testing.T) {
	if l := languageOf("film.hekaya"); l != rules.Hekaya {
		t.Errorf("language %q", l)
	}
	if ty := typeOfName("film.hekaya", false); ty.id != "hekaya" || ty.label+" ("+ty.shownExt()+")" != "Hekaya (.hekaya)" {
		t.Errorf("type %+v", ty)
	}
	found := false
	for _, ty := range scriptTypes(machineLex(), false, false, "film.ar.fountain", "ar") {
		found = found || ty.id == "hekaya"
	}
	if !found {
		t.Error("Save As offers no Hekaya for an Arabic script")
	}
	b, err := os.ReadFile("../lexington-langrules/testdata/hekaya/آخر-أيام-الصيف.hekaya")
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "film.hekaya")
	os.WriteFile(p, b, 0o644)
	w, _ := newDialogTestWindow(t)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if w.language != rules.Hekaya || !w.textEditor.LanguageRTL {
		t.Errorf("language %q, rtl %v", w.language, w.textEditor.LanguageRTL)
	}
	if msg := pdfWarning(w.textEditor.Text(), w.sceneHeaders()); msg != "" {
		t.Errorf("PDF warning for Arabic: %s", msg)
	}
	if items := outlineOf(w.textEditor.Text(), w.sceneStarts()); len(items) < 2 {
		t.Errorf("outline %+v", items)
	}
}
