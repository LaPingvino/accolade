package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

const (
	eoScript   = "EN. DOMO - TAGO\n\nBram kuiras.\n\nINT. is English for inside.\n"
	eoFountain = ".EN. DOMO - TAGO\n\nBram kuiras.\n\n!INT. is English for inside.\n"
)

// ours is Accolade's key in a bundle's info.json.
func oursIn(t *testing.T, info string) accoladeInfo {
	t.Helper()
	raw, ok := infoValue([]byte(info), accoladeInfoKey)
	var a accoladeInfo
	if ok {
		if err := json.Unmarshal(raw, &a); err != nil {
			t.Fatal(err)
		}
	}
	return a
}

// withEntry is a zip with one entry's data changed (another app's save).
func withEntry(t *testing.T, data []byte, name, text string) []byte {
	t.Helper()
	var items []zipItem
	for _, e := range rawEntries(t, data) {
		got, _ := zipFile(t, data, e.name)
		if e.name == name {
			got = text
		}
		items = append(items, zipItem{name: e.name, data: got, stored: e.method == 0})
	}
	return makeZip(t, items)
}

// A screenplay in Esperanto in a Highland bundle: text.eo.fountain as
// written, text.fountain beside it as standard Fountain, the language and
// text.fountain's hash in Accolade's key; the name stays plain, the other
// files as they were. It opens again in Esperanto from its own text.
func TestBundleLanguage(t *testing.T) {
	for _, tc := range []struct{ name, prefix string }{
		{"Script.highland", "Script.textbundle/"}, {"Root.textpack", ""}, {"Own.accolade", "Own.textbundle/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := draftsWindow(t)
			p := filepath.Join(t.TempDir(), tc.name)
			orig := makeZip(t, bundleItems(tc.prefix))
			os.WriteFile(p, orig, 0o644)
			if err := w.LoadFile(p); err != nil {
				t.Fatal(err)
			}
			if w.language != "en" {
				t.Fatalf("a bundle without a language is English, not %s", w.language)
			}
			w.setLanguage("eo")
			w.textEditor.SetText(eoScript)
			if err := w.SaveFile(); err != nil {
				t.Fatal(err)
			}
			w.flushBundleWrites()
			data, _ := os.ReadFile(p)
			if got, _ := zipFile(t, data, tc.prefix+"text.fountain"); got != eoFountain {
				t.Errorf("text.fountain:\n%s", got)
			}
			if got, _ := zipFile(t, data, tc.prefix+"text.eo.fountain"); got != eoScript {
				t.Errorf("text.eo.fountain:\n%s", got)
			}
			info, _ := zipFile(t, data, tc.prefix+"info.json")
			ours := oursIn(t, info)
			if ours.Language != "eo" || ours.FountainHash != fountainHash([]byte(eoFountain)) {
				t.Errorf("Accolade's key: %+v", ours)
			}
			if !strings.Contains(info, `"creatorURL" : "file:\/\/\/Applications\/Highland 2.app\/"`) {
				t.Errorf("info.json's own text changed:\n%s", info)
			}
			for _, keep := range []string{"characters.json", "unknown.bin", "sprints.json", "assets/cover.png"} {
				was, _ := zipFile(t, orig, tc.prefix+keep)
				if got, ok := zipFile(t, data, tc.prefix+keep); !ok || got != was {
					t.Errorf("%s not kept", keep)
				}
			}
			w2 := draftsWindow(t)
			if err := w2.LoadFile(p); err != nil {
				t.Fatal(err)
			}
			if w2.textEditor.Text() != eoScript || w2.language != "eo" || w2.hasChanges {
				t.Errorf("reopened in %s, changed %v:\n%s", w2.language, w2.hasChanges, w2.textEditor.Text())
			}
			// another round: the same bytes
			if err := w2.SaveFile(); err != nil {
				t.Fatal(err)
			}
			w2.flushBundleWrites()
			again, _ := os.ReadFile(p)
			for _, n := range []string{"text.fountain", "text.eo.fountain", "info.json"} {
				a, _ := zipFile(t, again, tc.prefix+n)
				b, _ := zipFile(t, data, tc.prefix+n)
				if a != b {
					t.Errorf("%s changed on a second save:\n%s\n---\n%s", n, b, a)
				}
			}
			// back to English: one text again, no language
			w2.setLanguage("en")
			w2.hasChanges = true
			if err := w2.SaveFile(); err != nil {
				t.Fatal(err)
			}
			w2.flushBundleWrites()
			en, _ := os.ReadFile(p)
			if _, ok := zipFile(t, en, tc.prefix+"text.eo.fountain"); ok {
				t.Error("text.eo.fountain left in an English bundle")
			}
			if got, _ := zipFile(t, en, tc.prefix+"text.fountain"); got != eoScript {
				t.Errorf("English text.fountain:\n%s", got)
			}
			info, _ = zipFile(t, en, tc.prefix+"info.json")
			if o := oursIn(t, info); o.Language != "" || o.FountainHash != "" {
				t.Errorf("English key: %+v", o)
			}
		})
	}
}

// Another app (Highland) changed text.fountain: on opening, its change
// is the script, its force marks taken back; the script before it is
// kept as a draft; saving makes both texts the same again.
func TestBundleLanguageOtherAppWins(t *testing.T) {
	w := draftsWindow(t)
	p := filepath.Join(t.TempDir(), "Script.highland")
	os.WriteFile(p, makeZip(t, bundleItems("Script.textbundle/")), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	w.setLanguage("eo")
	w.textEditor.SetText(eoScript)
	w.SaveFile()
	w.flushBundleWrites()
	data, _ := os.ReadFile(p)
	theirs := eoFountain + "\n.EKST. GARDENO - NOKTO\n\nHighland wrote this.\n"
	os.WriteFile(p, withEntry(t, data, "Script.textbundle/text.fountain", theirs), 0o644)

	w2 := draftsWindow(t)
	if err := w2.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	w2.flushBundleWrites()
	want := eoScript + "\nEKST. GARDENO - NOKTO\n\nHighland wrote this.\n"
	if w2.textEditor.Text() != want || w2.language != "eo" || !w2.hasChanges {
		t.Fatalf("opened (changed %v):\n%s", w2.hasChanges, w2.textEditor.Text())
	}
	ds := draftsOf(t, w2)
	if len(ds) != 1 {
		t.Fatalf("drafts: %+v", ds)
	}
	if got, _ := w2.extrasHome().readExtra(ds[0].Text); string(got) != eoScript {
		t.Errorf("the draft holds:\n%s", got)
	}
	if !strings.Contains(w2.statusText(), "another app") {
		t.Errorf("status: %q", w2.statusText())
	}
	w2.SaveFile()
	w2.flushBundleWrites()
	data, _ = os.ReadFile(p)
	native, _ := zipFile(t, data, "Script.textbundle/text.eo.fountain")
	plain, _ := zipFile(t, data, "Script.textbundle/text.fountain")
	if native != want || plain != englishFountain(want, "eo") {
		t.Errorf("after saving:\n%s\n---\n%s", native, plain)
	}
	if _, ok := zipFile(t, data, "Script.textbundle/unknown.bin"); !ok {
		t.Error("unknown.bin lost")
	}
	// opened once more: nothing changed elsewhere, no new draft
	w3 := draftsWindow(t)
	w3.LoadFile(p)
	w3.flushBundleWrites()
	if w3.hasChanges || len(draftsOf(t, w3)) != 1 {
		t.Errorf("opened again: changed %v, %d drafts", w3.hasChanges, len(draftsOf(t, w3)))
	}
}

// An .accolade may keep only the language's text: no text.fountain;
// turned on again, it is written again.
func TestAccoladeWithoutFountainCopy(t *testing.T) {
	w := draftsWindow(t)
	dir := t.TempDir()
	plain := filepath.Join(dir, "film.eo.fountain")
	os.WriteFile(plain, []byte(eoScript), 0o644)
	if err := w.LoadFile(plain); err != nil || w.language != "eo" {
		t.Fatal(err, w.language)
	}
	p := filepath.Join(dir, "film.accolade")
	w.noFountainCopy = true
	if err := w.saveToFile(p); err != nil {
		t.Fatal(err)
	}
	w.flushBundleWrites()
	data, _ := os.ReadFile(p)
	if _, ok := zipFile(t, data, "film.textbundle/text.fountain"); ok {
		t.Error("text.fountain written though left out")
	}
	if got, _ := zipFile(t, data, "film.textbundle/text.eo.fountain"); got != eoScript {
		t.Errorf("text.eo.fountain:\n%s", got)
	}
	info, _ := zipFile(t, data, "film.textbundle/info.json")
	if o := oursIn(t, info); o.Language != "eo" || !o.NoFountainCopy || o.FountainHash != "" {
		t.Errorf("key: %+v", o)
	}
	w2 := draftsWindow(t)
	if err := w2.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if w2.textEditor.Text() != eoScript || w2.language != "eo" || !w2.noFountainCopy || w2.hasChanges {
		t.Fatalf("reopened: %s %v %v\n%s", w2.language, w2.noFountainCopy, w2.hasChanges, w2.textEditor.Text())
	}
	w2.noFountainCopy = false
	w2.SaveFile()
	w2.flushBundleWrites()
	data, _ = os.ReadFile(p)
	if got, _ := zipFile(t, data, "film.textbundle/text.fountain"); got != eoFountain {
		t.Errorf("the copy again:\n%s", got)
	}
	// a Highland bundle always keeps it
	hl := filepath.Join(dir, "film.highland")
	w2.noFountainCopy = true
	w2.saveToFile(hl)
	w2.flushBundleWrites()
	data, _ = os.ReadFile(hl)
	if _, ok := zipFile(t, data, "film.textbundle/text.fountain"); !ok {
		t.Error("a .highland without text.fountain")
	}
}

// A bundle from before: its name says the language, its text.fountain
// is in it; it opens in that language.
func TestBundleLanguageFromOldName(t *testing.T) {
	w := draftsWindow(t)
	p := filepath.Join(t.TempDir(), "Old.eo.highland")
	os.WriteFile(p, makeZip(t, []zipItem{{name: "info.json", data: highlandInfo}, {name: "text.fountain", data: eoScript}}), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if w.language != "eo" || w.textEditor.Text() != eoScript {
		t.Errorf("opened in %s:\n%s", w.language, w.textEditor.Text())
	}
}

// Plain files: film.eo.fountain as written; film.fountain standard
// Fountain with its language in a last boneyard line, both taken back
// on opening; an English one as it is.
func TestPlainFountainLanguage(t *testing.T) {
	dir := t.TempDir()
	w := draftsWindow(t)
	dialectPath := filepath.Join(dir, "film.eo.fountain")
	os.WriteFile(dialectPath, []byte(eoScript), 0o644)
	if err := w.LoadFile(dialectPath); err != nil {
		t.Fatal(err)
	}
	plain := filepath.Join(dir, "film.fountain")
	if err := w.saveToFile(plain); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(plain)
	if string(got) != eoFountain+"\n/* Language: eo */\n" {
		t.Errorf("film.fountain:\n%s", got)
	}
	w.saveToFile(dialectPath)
	if got, _ := os.ReadFile(dialectPath); string(got) != eoScript {
		t.Errorf("film.eo.fountain:\n%s", got)
	}
	w2 := draftsWindow(t)
	if err := w2.LoadFile(plain); err != nil {
		t.Fatal(err)
	}
	if w2.language != "eo" || w2.textEditor.Text() != eoScript || w2.hasChanges {
		t.Errorf("film.fountain opened in %s:\n%s", w2.language, w2.textEditor.Text())
	}
	// English: as it is, whatever the preference
	was := w2.settings.GetString("script-language")
	w2.settings.SetString("script-language", "eo")
	t.Cleanup(func() { w2.settings.SetString("script-language", was) })
	en := filepath.Join(dir, "en.fountain")
	os.WriteFile(en, []byte(bundleScript), 0o644)
	w3 := draftsWindow(t)
	w3.LoadFile(en)
	w3.saveToFile(en)
	if got, _ := os.ReadFile(en); w3.language != "en" || string(got) != bundleScript {
		t.Errorf("English file in %s:\n%s", w3.language, got)
	}
}

// eoAccolade is a window with film.accolade in Esperanto, saved.
func eoAccolade(t *testing.T) (*MainWindow, string) {
	t.Helper()
	w := draftsWindow(t)
	dir := t.TempDir()
	plain := filepath.Join(dir, "film.eo.fountain")
	os.WriteFile(plain, []byte(eoScript), 0o644)
	if err := w.LoadFile(plain); err != nil {
		t.Fatal(err)
	}
	if err := w.renameTo("film.accolade"); err != nil {
		t.Fatal(err)
	}
	w.flushBundleWrites()
	p := filepath.Join(dir, "film.accolade")
	if w.currentFile != p || w.language != "eo" {
		t.Fatalf("made %s in %s", w.currentFile, w.language)
	}
	return w, p
}

// The rename editor on a bundle: the Language picker is there; another
// language with the same name saves it inside, the name as it is;
// another name too renames it, the old language's text gone.
func TestRenameEditorBundleLanguage(t *testing.T) {
	w, p := eoAccolade(t)
	data, _ := os.ReadFile(p)
	if got, _ := zipFile(t, data, "film.textbundle/text.eo.fountain"); got != eoScript {
		t.Fatalf("converted: text.eo.fountain %q", got)
	}
	w.renameScript()
	a := w.renameAsked
	if a.lang.Selected != languageLabel("eo") || a.base.Text != "film" || a.kind.Selected != "Accolade (.accolade)" {
		t.Fatalf("opened as %q %q %q", a.lang.Selected, a.base.Text, a.kind.Selected)
	}
	if a.result.Text != "Becomes: film.accolade  · Esperanto, in info.json, its text as text.eo.fountain" {
		t.Errorf("result %q", a.result.Text)
	}
	if !strings.Contains(planText(a.plan), "text.eo.fountain, text.fountain (standard Fountain)") {
		t.Errorf("plan:\n%s", planText(a.plan))
	}
	a.lang.SetSelected(languageLabel("nl"))
	a.confirm()
	w.flushBundleWrites()
	data, _ = os.ReadFile(p)
	info, _ := zipFile(t, data, "film.textbundle/info.json")
	if _, ok := zipFile(t, data, "film.textbundle/text.eo.fountain"); ok || oursIn(t, info).Language != "nl" || w.language != "nl" {
		t.Errorf("in Dutch: %s, %s", w.language, info)
	}
	if got, _ := zipFile(t, data, "film.textbundle/text.nl.fountain"); got != eoScript {
		t.Errorf("text.nl.fountain %q", got)
	}
	// renamed and in Esperanto again, without the copy
	w.renameScript()
	a = w.renameAsked
	a.base.SetText("story")
	a.lang.SetSelected(languageLabel("eo"))
	a.plan.copy.SetChecked(false)
	if strings.Contains(planText(a.plan), "text.fountain (standard") {
		t.Errorf("plan still has the copy:\n%s", planText(a.plan))
	}
	a.confirm()
	w.flushBundleWrites()
	story := filepath.Join(filepath.Dir(p), "story.accolade")
	data, _ = os.ReadFile(story)
	if exists(p) || w.currentFile != story {
		t.Fatalf("renamed to %s, old there %v", w.currentFile, exists(p))
	}
	for name, want := range map[string]bool{"text.eo.fountain": true, "text.nl.fountain": false, "text.fountain": false} {
		if _, ok := zipFile(t, data, "story.textbundle/"+name); ok != want {
			t.Errorf("%s there: %v", name, ok)
		}
	}
	// Undo Rename: film.accolade in Dutch again
	w.undoRename()
	w.flushBundleWrites()
	if w.currentFile != p || w.language != "nl" || exists(story) {
		t.Errorf("after Undo Rename: %s in %s", w.currentFile, w.language)
	}
}

// Edit > Script Language… on a bundle keeps its name.
func TestScriptLanguageBundle(t *testing.T) {
	w, p := eoAccolade(t)
	w.changeLanguage("nl", markLanguage, nil)
	w.flushBundleWrites()
	data, _ := os.ReadFile(p)
	info, _ := zipFile(t, data, "film.textbundle/info.json")
	if w.currentFile != p || oursIn(t, info).Language != "nl" {
		t.Errorf("%s: %s", w.currentFile, info)
	}
}

// Screenshots of the rename editor for an Esperanto .accolade, on a
// desktop and a phone (SHOT_DIR=… go test -run TestBundleLangScreenshots):
// bundlelang-*.png.
func TestBundleLangScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(dir, 0o755)
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"desktop", fyne.NewSize(1100, 760)}, {"phone", fyne.NewSize(380, 800)}} {
		w, _ := eoAccolade(t)
		w.fyneWindow.Resize(size.size)
		w.renameScript()
		capture(t, w.fyneWindow.Canvas(), "bundlelang-rename-"+size.name+".png")
		w.renameAsked.scroll.ScrollToBottom()
		capture(t, w.fyneWindow.Canvas(), "bundlelang-rename-files-"+size.name+".png")
		w.renameAsked.kind.SetSelected("Esperanto Fountain (.eo.fountain)")
		w.renameAsked.scroll.ScrollToTop()
		capture(t, w.fyneWindow.Canvas(), "bundlelang-rename-dialect-"+size.name+".png")
	}
}
