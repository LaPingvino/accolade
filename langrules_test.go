package main

import (
	"slices"
	"time"

	"fyne.io/fyne/v2"
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
	if ty := typeOfName("film.hekaya", false); ty.id != "langext" || ty.label+" ("+ty.shownExt()+")" != "Hekaya (.hekaya)" {
		t.Errorf("type %+v", ty)
	}
	found := false
	for _, ty := range scriptTypes(machineLex(), false, false, "film.ar.fountain", "ar") {
		found = found || ty.id == "langext" && ty.lang == rules.Hekaya
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
	if msg := pdfWarning(w.textEditor.Text(), w.lex().Language(w.language)); msg != "" {
		t.Errorf("PDF warning for Arabic: %s", msg)
	}
	if items := outlineOf(w.textEditor.Text(), w.sceneStarts()); len(items) < 2 {
		t.Errorf("outline %+v", items)
	}
}

// Screenshots (SHOT_DIR=… go test -run TestLangRulesScreenshots): an
// Arabic (Hekaya) and a Chinese script in the editor and the page view,
// on a desktop and a phone: langrules-{ar,zh}-{editor,page}-{desktop,phone}.png.
func TestLangRulesScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("SHOT_DIR not set")
	}
	ar, err := os.ReadFile("../lexington-langrules/testdata/hekaya/آخر-أيام-الصيف.hekaya")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		lang, name, text string
	}{{rules.Hekaya, "ar", string(ar)}, {"zh", "zh", chineseScript}} {
		for _, size := range []struct {
			name string
			size fyne.Size
		}{{"desktop", fyne.NewSize(1200, 800)}, {"phone", fyne.NewSize(380, 800)}} {
			w, _ := newDialogTestWindow(t)
			w.setLanguage(c.lang)
			w.textEditor.SetText(c.text)
			w.fyneWindow.Resize(size.size)
			time.Sleep(100 * time.Millisecond)
			fyne.DoAndWait(func() {})
			capture(t, w.fyneWindow.Canvas(), "langrules-"+c.name+"-editor-"+size.name+".png")
			if c.name == "zh" {
				shown := make(chan *pageView, 4)
				onPagesShown(t, func(pv *pageView) { shown <- pv })
				w.previewPDF()
				select {
				case <-shown:
				case <-time.After(20 * time.Second):
					var status string
					fyne.DoAndWait(func() { status = w.statusText() })
					t.Logf("zh: no pages; status %q", status)
				}
				time.Sleep(300 * time.Millisecond)
				fyne.DoAndWait(func() {})
				capture(t, w.fyneWindow.Canvas(), "langrules-zh-page-"+size.name+".png")
				continue
			}
			v := previewAndWait(t, w)
			painted(t, v, 0)
			if len(v.d.Pages) > 1 {
				v.area.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
				painted(t, v, 1)
			}
			time.Sleep(100 * time.Millisecond)
			fyne.DoAndWait(func() {})
			capture(t, w.fyneWindow.Canvas(), "langrules-"+c.name+"-page-"+size.name+".png")
		}
	}
}

// A language's own extension: names, a user-defined one, a save and a
// reopen as written.
func TestLanguageExtensions(t *testing.T) {
	for _, c := range [][3]string{
		{"film.ar.fountain", rules.Hekaya, "film.hekaya"},
		{"film.hekaya", "ar", "film.ar.fountain"},
		{"film.hekaya", rules.Hekaya, "film.hekaya"},
		{"film.fountain", rules.Hekaya, "film.hekaya"},
	} {
		if got := withLanguageName(c[0], c[1]); got != c[2] {
			t.Errorf("%s as %s: %s, want %s", c[0], c[1], got, c[2])
		}
	}
	old := lexSettings
	t.Cleanup(func() { lexSettings = old; sceneConf = lexSettings.Scenes; startsDialects.Clear() })
	lexSettings.Extensions = map[string]string{"kissa": "zh"}
	if l := languageOf("x.kissa"); l != "zh" {
		t.Errorf("own extension: %q", l)
	}
	if !slices.Contains(openExtensions(), ".kissa") || !slices.Contains(openExtensions(), ".hekaya") {
		t.Errorf("open: %v", openExtensions())
	}
	p := filepath.Join(t.TempDir(), "film.kissa")
	os.WriteFile(p, []byte(chineseScript), 0o644)
	w, _ := newDialogTestWindow(t)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if w.language != "zh" {
		t.Fatalf("language %q", w.language)
	}
	if err := w.SaveFile(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != chineseScript {
		t.Errorf("saved otherwise:\n%s", b)
	}
	w2, _ := newDialogTestWindow(t)
	if err := w2.LoadFile(p); err != nil || w2.language != "zh" || w2.textEditor.Text() != strings.TrimSuffix(chineseScript, "") && !strings.HasPrefix(w2.textEditor.Text(), "Title: 咖啡馆") {
		t.Errorf("reopened: %v %q", err, w2.language)
	}
}

// Action and Transition are marked as the language reads them.
func TestMarkActionTransitionByRules(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	set := func(lang, text, line string) { // the cursor at the end of the first line of line in text
		w.setLanguage(lang)
		w.textEditor.SetText(text)
		i := strings.Index(text, line)
		first, _, _ := strings.Cut(strings.TrimPrefix(line, "\n"), "\n")
		w.textEditor.SetCursorOffset(len([]rune(text[:i+len(line)-len(strings.TrimPrefix(line, "\n"))+len(first)])))
	}
	set(rules.Hekaya, "داخلي - قهوة - نهار\n\n@سمير\nأهلا.\n\nقطع\n\nسمير\nكلام.\n", "قطع")
	w.markLineElement(elementTransition)
	if !strings.Contains(w.textEditor.Text(), "\n\nقطع\n\n") {
		t.Errorf("a transition by its word got a mark:\n%s", w.textEditor.Text())
	}
	set(rules.Hekaya, "داخلي - قهوة - نهار\n\n@سمير\nأهلا.\n\nسمير\nكلام.\n", "\nسمير\nكلام")
	w.markLineElement(elementAction)
	if !strings.Contains(w.textEditor.Text(), "\n!سمير\nكلام") {
		t.Errorf("a registered name marked Action got no !:\n%q", w.textEditor.Text())
	}
	set("zh", "场景1 咖啡馆 日 内\n\n他走了。\n", "他走了。")
	w.markLineElement(elementAction)
	if !strings.Contains(w.textEditor.Text(), "\n他走了。\n") {
		t.Errorf("action marked:\n%s", w.textEditor.Text())
	}
}
