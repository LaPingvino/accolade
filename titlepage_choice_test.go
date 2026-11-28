// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/rules"
)

// A title page template of one's own, as one machine has it.
const loudSettings = "[TitlePages.loud]\nName = \"Loud\"\n\n[[TitlePages.loud.Keys]]\nKey = \"Logline\"\nPlace = \"top\"\n\n" +
	"[[TitlePages.loud.Keys]]\nKey = \"Title\"\nPlace = \"centre\"\nSize = 30\n"

const loudScript = "Title: Rain\nAuthor: Ann\nLogline: A storm comes.\n\nINT. HOUSE - DAY\n\nRain.\n"

// withLoud is a settings folder whose lexington.toml has the template
// loud, the default template put back after the test.
func withLoud(t *testing.T) {
	t.Helper()
	prev := GetSettings().GetString(titleTemplatePref)
	t.Cleanup(func() { GetSettings().SetString(titleTemplatePref, prev) })
	conf := scratchSettings(t)
	if err := os.MkdirAll(conf, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(conf, "lexington.toml"), []byte(loudSettings), 0o644); err != nil {
		t.Fatal(err)
	}
	reloadLexingtonSettings()
	GetSettings().SetString(titleTemplatePref, "")
}

func scriptFile(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(loudScript), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// Each script keeps its own template: one chosen for one script leaves
// another, saved with its own, as it was; the last chosen is where new
// scripts start; a script without one prints with the default and is
// given it when saved.
func TestTitleTemplatePerScript(t *testing.T) {
	w := draftsWindow(t)
	withLoud(t)
	dir := t.TempDir()
	a, b := scriptFile(t, dir, "a.fountain"), scriptFile(t, dir, "b.fountain")
	open := func(p string) {
		t.Helper()
		if err := w.LoadFile(p); err != nil {
			t.Fatal(err)
		}
	}

	open(a)
	if w.titleTemplate() != "" || w.titleChoice != "" {
		t.Fatalf("a new file: %q (own %q)", w.titleTemplate(), w.titleChoice)
	}
	if err := w.SaveFile(); err != nil {
		t.Fatal(err)
	}
	if got := w.titleChoiceOf(a); got != rules.StandardTitlePage {
		t.Fatalf("saved, a keeps %q", got)
	}

	open(b)
	w.setTitleTemplate("poster")
	if w.titleTemplate() != "poster" || w.titleChoiceOf(b) != "poster" || w.defaultTitleTemplate() != "poster" {
		t.Fatalf("chosen for b: %q, kept %q, default %q", w.titleTemplate(), w.titleChoiceOf(b), w.defaultTitleTemplate())
	}
	if j := w.layoutJob(); j.titleTemplate != "poster" || j.options("pdf", "x.pdf").TitlePage != "poster" {
		t.Errorf("b's export: %q", j.titleTemplate)
	}

	open(a)
	if w.titleTemplate() != "" || w.pageSetup().titleTemplate != "" {
		t.Errorf("a took b's template: %q", w.titleTemplate())
	}
	open(b)
	if w.titleTemplate() != "poster" {
		t.Errorf("b reopened: %q", w.titleTemplate())
	}

	w.NewFile()
	if w.titleTemplate() != "poster" {
		t.Errorf("a new script: %q, want the last chosen", w.titleTemplate())
	}

	// Page Setup's choice is the script's too
	open(a)
	p := w.pageSetup()
	p.titleTemplate = "loud"
	w.setPageSetup(p)
	if w.titleChoiceOf(a) != "loud" || w.defaultTitleTemplate() != "loud" {
		t.Errorf("Page Setup: kept %q, default %q", w.titleChoiceOf(a), w.defaultTitleTemplate())
	}
	open(b)
	if w.titleTemplate() != "poster" {
		t.Errorf("b after a's Page Setup: %q", w.titleTemplate())
	}

	// a rename takes the choice along
	if err := w.renameTo("c.fountain"); err != nil {
		t.Fatal(err)
	}
	if got := w.titleChoiceOf(filepath.Join(dir, "c.fountain")); got != "poster" {
		t.Errorf("renamed: %q", got)
	}
}

// An .accolade names its template in info.json and carries one of one's
// own: on a machine without it, Accolade and Lexington's command line
// print with it, and the machine's default stays as it was.
func TestTitleTemplateTravels(t *testing.T) {
	w := draftsWindow(t)
	withLoud(t)
	dir := t.TempDir()
	if err := w.LoadFile(scriptFile(t, dir, "film.fountain")); err != nil {
		t.Fatal(err)
	}
	w.setTitleTemplate("loud")
	if err := w.renameTo("film.accolade"); err != nil {
		t.Fatal(err)
	}
	w.flushBundleWrites()
	p := filepath.Join(dir, "film.accolade")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	const pre = "film.textbundle/"
	info, _ := zipFile(t, data, pre+"info.json")
	if a := oursIn(t, info); a.TitlePage != "loud" {
		t.Errorf("info.json: %+v", a)
	}
	if lexText, ok := zipFile(t, data, pre+bundleLexFile); !ok || !strings.Contains(lexText, "[TitlePages.loud]") {
		t.Fatalf("lexington.toml carried: %v\n%s", ok, lexText)
	}

	// another machine, without loud
	w2 := againWindow(t, w)
	scratchSettings(t)
	GetSettings().SetString(titleTemplatePref, "bold")
	if err := w2.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if w2.titleTemplate() != "loud" || w2.lex().TitlePageName("loud") != "Loud" {
		t.Errorf("opened elsewhere: %q", w2.titleTemplate())
	}
	if s := w2.statusText(); !strings.Contains(s, "title page template “Loud”") {
		t.Errorf("status: %q", s)
	}
	if _, ok := lexSettings.TitlePage("loud"); ok || w2.defaultTitleTemplate() != "bold" {
		t.Error("opening changed the machine's settings")
	}
	if o := w2.layoutJob().options("pdf", "x.pdf"); o.TitlePage != "loud" || o.Conf == nil {
		t.Errorf("export options: %q", o.TitlePage)
	}

	// Lexington's command line, with the machine's settings only
	out := filepath.Join(dir, "film.html")
	o := cli.Options{Input: p, Output: out, ConfigFile: filepath.Join(dir, "none.toml")}
	if err := cli.Convert(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	html, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if i, j := strings.Index(string(html), "A storm comes."), strings.LastIndex(string(html), "Rain</"); i < 0 || j < 0 || i > j {
		t.Errorf("the command line's title page (Logline above the title):\n%s", html)
	}
}

// TestTitleChoiceScreenshots renders, with SHOT_DIR set: script a with
// Poster chosen on its Title Page tab, its pages previewed; script b,
// opened after it, still on its own Standard; and an .accolade whose
// template of its own (Loud) came with it, on a machine without it,
// with the status line saying so.
func TestTitleChoiceScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	holdUI(t)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	w := draftsWindow(t)
	withLoud(t)
	r := ribbonOnFor(t, w)
	w.fyneWindow.Resize(fyne.NewSize(1200, 760))
	shot := func(name string) {
		t.Helper()
		w.textEditor.SetCursorOffset(3)
		w.ribbonRefresh()
		r.selectTab("Title Page")
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
	}
	scripts := t.TempDir()
	a, b := scriptFile(t, scripts, "a.fountain"), scriptFile(t, scripts, "b.fountain")
	if err := w.LoadFile(b); err != nil {
		t.Fatal(err)
	}
	if err := w.SaveFile(); err != nil { // b: Standard, its own
		t.Fatal(err)
	}
	if err := w.LoadFile(a); err != nil {
		t.Fatal(err)
	}
	w.setTitleTemplate("poster")
	painted(t, previewAndWait(t, w), 0)
	shot("tplchoice-1-a-poster.png")
	if err := w.LoadFile(b); err != nil {
		t.Fatal(err)
	}
	painted(t, previewAndWait(t, w), 0)
	shot("tplchoice-2-b-standard.png")
	if r.titleTpl.Selected != "Standard" {
		t.Errorf("b's tab shows %q", r.titleTpl.Selected)
	}

	// an .accolade with its own template, on a machine without it
	w.setTitleTemplate("loud")
	if err := w.renameTo("film.accolade"); err != nil {
		t.Fatal(err)
	}
	w.flushBundleWrites()
	scratchSettings(t)
	if err := w.LoadFile(filepath.Join(scripts, "film.accolade")); err != nil {
		t.Fatal(err)
	}
	painted(t, previewAndWait(t, w), 0)
	shot("tplchoice-3-carried-loud.png")
	if r.titleTpl.Selected != "Loud" {
		t.Errorf("the carried template's tab shows %q", r.titleTpl.Selected)
	}
	w.addToMySettings() // the template offered to keep
	w.fyneWindow.Content().Refresh()
	f, err := os.Create(filepath.Join(dir, "tplchoice-4-add-to-my-settings.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
		t.Fatal(err)
	}
}

// A template of one's own that came with an .accolade is added by Add to
// My Settings: then the machine's, the script's choice as it was.
func TestAddTitleTemplateToMySettings(t *testing.T) {
	w := draftsWindow(t)
	withLoud(t)
	dir := t.TempDir()
	if err := w.LoadFile(scriptFile(t, dir, "film.fountain")); err != nil {
		t.Fatal(err)
	}
	w.setTitleTemplate("loud")
	if err := w.renameTo("film.accolade"); err != nil {
		t.Fatal(err)
	}
	w.flushBundleWrites()

	conf := scratchSettings(t)
	if err := w.LoadFile(filepath.Join(dir, "film.accolade")); err != nil {
		t.Fatal(err)
	}
	w.addToMySettings()
	a := w.adoptAsked
	if a == nil || len(a.adds) != 1 || a.adds[rules.ItemTitlePage+":loud"] == nil {
		t.Fatalf("listed: %+v", a)
	}
	a.apply()
	if _, ok := lexSettings.TitlePage("loud"); !ok {
		t.Fatal("not added")
	}
	if b, _ := os.ReadFile(filepath.Join(conf, "lexington.toml")); !strings.Contains(string(b), "[TitlePages.loud]") {
		t.Errorf("lexington.toml:\n%s", b)
	}
	if w.titleTemplate() != "loud" || len(w.carried.toAdd()) != 0 {
		t.Errorf("after: %q, left %+v", w.titleTemplate(), w.carried.toAdd())
	}
}

// The template an .accolade names and carries is the one its script
// prints with, however it got it: chosen in an earlier session (kept
// for the plain file), or the default it never changed.
func TestTitleTemplateTravelsLater(t *testing.T) {
	w := draftsWindow(t)
	withLoud(t)
	dir := t.TempDir()
	carried := func(name string) {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(dir, name+".accolade"))
		if err != nil {
			t.Fatal(err)
		}
		pre := name + ".textbundle/"
		info, _ := zipFile(t, data, pre+"info.json")
		lexText, _ := zipFile(t, data, pre+bundleLexFile)
		if a := oursIn(t, info); a.TitlePage != "loud" || !strings.Contains(lexText, "[TitlePages.loud]") {
			t.Errorf("%s: info.json %+v, lexington.toml:\n%s", name, a, lexText)
		}
	}
	// chosen, then the file opened again in another window
	if err := w.LoadFile(scriptFile(t, dir, "one.fountain")); err != nil {
		t.Fatal(err)
	}
	w.setTitleTemplate("loud")
	w2 := againWindow(t, w)
	if err := w2.LoadFile(filepath.Join(dir, "one.fountain")); err != nil {
		t.Fatal(err)
	}
	if err := w2.renameTo("one.accolade"); err != nil {
		t.Fatal(err)
	}
	w2.flushBundleWrites()
	carried("one")

	// never chosen: the default is loud
	w3 := againWindow(t, w)
	GetSettings().SetString(titleTemplatePref, "loud")
	if err := w3.LoadFile(scriptFile(t, dir, "two.fountain")); err != nil {
		t.Fatal(err)
	}
	if err := w3.renameTo("two.accolade"); err != nil {
		t.Fatal(err)
	}
	w3.flushBundleWrites()
	carried("two")
}
