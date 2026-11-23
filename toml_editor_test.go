package main

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/rules"
)

// tomlTestApp is a window with its app, the settings in a scratch folder.
func tomlTestApp(t *testing.T) *MainWindow {
	t.Helper()
	w := newTestWindow(t, "INT. KITCHEN - NIGHT\n\nRain.\n")
	scratchSettings(t)
	w.app.windows = []*MainWindow{w}
	t.Cleanup(func() {
		for _, e := range append([]*tomlEditor(nil), w.app.tomlEditors...) {
			e.doc.saved = e.doc.text
			e.close()
		}
	})
	return w
}

// find is the first object under o that ok says is it.
func find[T fyne.CanvasObject](o fyne.CanvasObject, ok func(T) bool) T {
	var zero T
	if x, is := o.(T); is && ok(x) {
		return x
	}
	var kids []fyne.CanvasObject
	switch c := o.(type) {
	case *fyne.Container:
		kids = c.Objects
	case *container.Scroll:
		kids = []fyne.CanvasObject{c.Content}
	case *widget.Card:
		kids = []fyne.CanvasObject{c.Content}
	case *flowBox:
		kids = []fyne.CanvasObject{c.box}
	case *container.AppTabs:
		if c.Selected() != nil {
			kids = []fyne.CanvasObject{c.Selected().Content}
		}
	}
	for _, k := range kids {
		if k == nil {
			continue
		}
		if x := find(k, ok); fyne.CanvasObject(x) != nil && !isNil(x) {
			return x
		}
	}
	return zero
}

func isNil[T any](x T) bool { var zero T; return any(x) == any(zero) }

// button is the editor's page's button with text.
func button(t *testing.T, e *tomlEditor, text string) *widget.Button {
	t.Helper()
	b := find(e.tabs.Selected().Content, func(b *widget.Button) bool { return b.Text == text })
	if b == nil {
		t.Fatalf("no button %q", text)
	}
	return b
}

// selectTab brings the page with title to the front.
func selectTab(t *testing.T, e *tomlEditor, title string) {
	t.Helper()
	for _, it := range e.tabs.Items {
		if it.Text == title {
			e.tabs.Select(it)
			return
		}
	}
	t.Fatalf("no tab %q", title)
}

// A .toml opened from the window goes to its editor, picked by what it
// is, and the window's script stays; one open already comes forward.
func TestOpenTOMLRoutes(t *testing.T) {
	w := tomlTestApp(t)
	dir := t.TempDir()
	write := func(name, text string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	conf := write("lexington.toml", "# mine\nStandard = \"eo\"\n")
	format := write("mine.toml", "Name = \"Mine\"\nBase = \"comic\"\n")
	b, _ := os.ReadFile("FyneApp.toml")
	other := write("FyneApp.toml", string(b))
	for _, c := range []struct {
		path  string
		kind  tomlKind
		pages []string
	}{
		{conf, tomlSettings, []string{"General", "Languages", "Element Sets", "TOML", "Help"}},
		{format, tomlFormat, []string{"Format", "Elements", "Buttons", "TOML", "Help"}},
		{other, tomlPlain, []string{"TOML"}},
	} {
		w.openPath(c.path)
		e := w.app.tomlEditorOf(c.path)
		if e == nil {
			t.Fatalf("%s: no editor", c.path)
		}
		if e.doc.kind != c.kind {
			t.Errorf("%s: %v", c.path, e.doc.kind)
		}
		var got []string
		for _, it := range e.tabs.Items {
			got = append(got, it.Text)
		}
		if strings.Join(got, ",") != strings.Join(c.pages, ",") {
			t.Errorf("%s: tabs %v", c.path, got)
		}
		if !strings.Contains(e.win.Title(), filepath.Base(c.path)) {
			t.Errorf("title %q", e.win.Title())
		}
	}
	if w.textEditor.Text() != "INT. KITCHEN - NIGHT\n\nRain.\n" || w.currentFile != "" {
		t.Errorf("the script changed: %q %q", w.currentFile, w.textEditor.Text())
	}
	n := len(w.app.tomlEditors)
	w.openPath(conf)
	if len(w.app.tomlEditors) != n {
		t.Error("opened twice")
	}
	// the command line: no script window for a .toml
	app := w.app
	app.openFiles([]string{format})
	if len(app.windows) != 1 {
		t.Errorf("%d windows", len(app.windows))
	}
	// alone: no window at all, the editor keeps Accolade up until it closes
	alone := &Application{fyneApp: app.fyneApp}
	alone.openFiles([]string{conf})
	e := alone.tomlEditorOf(conf)
	if len(alone.windows) != 0 || e == nil || !strings.Contains(e.win.Title(), "the lexington command reads it") {
		t.Fatalf("alone: %d windows, editor %v", len(alone.windows), e != nil)
	}
	e.close()
	if len(alone.tomlEditors) != 0 {
		t.Error("still open")
	}
	if !strings.Contains(app.tomlEditorOf(conf).about(), "lexington command reads this file when run in") {
		t.Errorf("about: %s", app.tomlEditorOf(conf).about())
	}
}

// The settings editor: a language's field overridden from its default,
// changed, back to the default; the older tables moved out when their
// language is changed; Save reads the settings again.
func TestSettingsEditor(t *testing.T) {
	w := tomlTestApp(t)
	text := "# my settings\n[Scenes]\n# Esperanto\neo = [\"EN\", \"EKST\"]\n\n[Languages.de]\n# German, mine\nRepeat = 3\n"
	if err := os.MkdirAll(filepath.Dir(lexingtonConfig()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lexingtonConfig(), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	reloadLexingtonSettings()
	w.app.prefs = nil
	e := openTOMLFile(w.app, w.fyneWindow, lexingtonConfig(), lexingtonSettingsText())
	selectTab(t, e, "Languages")
	e.sel["language"] = "de"
	e.redraw()
	box := find(e.tabs.Selected().Content, func(l *widget.Label) bool { return l.Text == "Transitions" })
	if box == nil {
		t.Fatal("no Transitions field")
	}
	// Transitions is the 13th field: its Override
	var overrides []*widget.Button
	collect(e.tabs.Selected().Content, func(b *widget.Button) {
		if b.Text == "Override" {
			overrides = append(overrides, b)
		}
	})
	if len(overrides) != len(rules.LanguageFields())-1 { // Repeat is set
		t.Fatalf("%d overrides", len(overrides))
	}
	test.Tap(overrides[0]) // Scenes
	if got := e.doc.conf().Languages["de"].Scenes; len(got) == 0 || got[0] != rules.DefaultLanguages()["de"].Scenes[0] {
		t.Errorf("de scenes %v", got)
	}
	if !strings.Contains(e.doc.text, "# German, mine\nRepeat = 3\nScenes = [") {
		t.Errorf("text:\n%s", e.doc.text)
	}
	// the list's first word changed
	en := find(e.tabs.Selected().Content, func(en *widget.Entry) bool { return en.Text == rules.DefaultLanguages()["de"].Scenes[0] })
	test.Type(en, "X")
	if got := e.doc.conf().Languages["de"].Scenes[0]; got != "X"+rules.DefaultLanguages()["de"].Scenes[0] {
		t.Errorf("typed: %q", got)
	}
	test.Tap(button(t, e, "Use the Default"))
	if e.doc.has("Languages", "de", "Scenes") || !e.doc.has("Languages", "de", "Repeat") {
		t.Errorf("reset:\n%s", e.doc.text)
	}
	// Esperanto, in the older [Scenes]: a change moves it out
	e.sel["language"] = "eo"
	e.redraw()
	if !e.langDo("eo", true, set("rtl", "Languages", "eo", "Direction")) {
		t.Fatal(e.status.Text)
	}
	c := rules.DefaultConf().With(e.doc.conf())
	if e.doc.has("Scenes", "eo") || c.Scenes("eo")[0] != "EN" || c.Direction("eo") != "rtl" {
		t.Errorf("moved:\n%s", e.doc.text)
	}
	for _, kept := range []string{"# my settings", "# Esperanto", "# German, mine"} {
		if !strings.Contains(e.doc.text, kept) {
			t.Errorf("lost %q:\n%s", kept, e.doc.text)
		}
	}
	// a wrong value typed in the TOML tab: the pages are off, Save refuses
	selectTab(t, e, "TOML")
	e.raw.SetText(e.doc.text + "\n[Languages.xx\n")
	if !strings.Contains(e.status.Text, "line") || !e.tabs.Items[0].Disabled() {
		t.Errorf("status %q", e.status.Text)
	}
	e.raw.SetText(strings.TrimSuffix(e.raw.Text, "\n[Languages.xx\n"))
	if e.tabs.Items[0].Disabled() {
		t.Error("still off")
	}
	e.saveNow()
	if lexSettings.Direction("eo") != "rtl" {
		t.Errorf("not read again: %q", lexSettings.Direction("eo"))
	}
	if b, _ := os.ReadFile(lexingtonConfig()); string(b) != e.doc.text || e.doc.dirty() {
		t.Errorf("saved:\n%s", b)
	}
}

// collect calls f with every T under o.
func collect[T fyne.CanvasObject](o fyne.CanvasObject, f func(T)) {
	find(o, func(x T) bool { f(x); return false })
}

// The format editor: an element changed here starts as Base's whole; a
// margin typed; the buttons moved; saved in the formats folder, a format
// to pick at once.
func TestFormatEditor(t *testing.T) {
	w := tomlTestApp(t)
	path := filepath.Join(userFormatDir(), "mine.toml")
	if err := os.MkdirAll(userFormatDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	text := "# my comic\nName = \"My Comic\"\nBase = \"comic\"\n\n# pages first\n[[Accolade.Buttons]]\nLabel = \"Page\"\nInsert = \"===\\n\\n.PAGE {n}\"\n\n[[Accolade.Buttons]]\nLabel = \"Panel\"\nInsert = \"Panel {n}. \"\n"
	e := openTOMLFile(w.app, w.fyneWindow, path, text)
	selectTab(t, e, "Elements")
	e.sel["format-element"] = "speaker"
	e.redraw()
	test.Tap(button(t, e, "Change It Here"))
	comic, _ := rules.GetPreset("comic")
	if got := e.doc.spec().Elements["speaker"]; got != comic.Elements["speaker"] {
		t.Errorf("copied %+v, want %+v", got, comic.Elements["speaker"])
	}
	left := find(e.tabs.Selected().Content, func(en *widget.Entry) bool { return en.Text == trimFloat(comic.Elements["speaker"].Left) })
	if left == nil {
		t.Fatal("no Left entry")
	}
	left.SetText("3.25")
	if got := e.doc.spec().Elements["speaker"]; got.Left != 3.25 || got.Postfix != comic.Elements["speaker"].Postfix {
		t.Errorf("left: %+v", got)
	}
	left.SetText("-1")
	if e.doc.spec().Elements["speaker"].Left != 3.25 || left.Validate() == nil {
		t.Error("a negative margin taken")
	}
	selectTab(t, e, "Buttons")
	var downs []*widget.Button
	collect(e.tabs.Selected().Content, func(b *widget.Button) {
		if b.Icon != nil && b.Icon.Name() == "arrow-down.svg" || b.Icon != nil && strings.Contains(b.Icon.Name(), "down") {
			downs = append(downs, b)
		}
	})
	if len(downs) == 0 {
		t.Fatal("no Move down")
	}
	test.Tap(downs[0])
	if b := e.doc.spec().Accolade.Buttons; len(b) != 2 || b[0].Label != "Panel" {
		t.Errorf("moved: %+v", b)
	}
	if !strings.Contains(e.doc.text, "# pages first\n[[Accolade.Buttons]]\nLabel = \"Page\"") {
		t.Errorf("comment not with its button:\n%s", e.doc.text)
	}
	e.saveNow()
	if f := scriptFormat("mine"); f.Name != "My Comic" || f.Elements["speaker"].Left != 3.25 {
		t.Errorf("not a format yet: %+v", f)
	}
	if b := formatButtons("mine"); len(b) != 2 || b[0].Label != "Panel" {
		t.Errorf("buttons: %+v", b)
	}
}

// The settings and format editors on a desktop and a phone
// (SHOT_DIR=… go test -run TestTOMLEditorScreenshots): tomleditor-*.png.
func TestTOMLEditorScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	w := tomlTestApp(t)
	conf := "# my settings\nStandard = \"en\"\n\n[Extensions]\nesp = \"eo\"\n\n[Languages.eo]\n# Esperanto, mine\nScenes = [\"EN\", \"EKST\", \"EN/EKST\"]\nDirection = \"ltr\"\n\n" +
		"[Elements.mine.scene]\nLeft = 1.5\nRight = 1.0\nStyle = \"bu\"\n\n[Elements.mine.action]\nLeft = 1.5\nRight = 1.0\n\n[Elements.mine.speaker]\nLeft = 3.7\nRight = 1.5\n\n" +
		"[Elements.mine.dialog]\nLeft = 2.5\nRight = 1.5\n\n[Elements.mine.paren]\nLeft = 3.1\nRight = 1.5\n\n[Elements.mine.trans]\nLeft = 1.5\nRight = 1.0\nAlign = \"R\"\n"
	format := "# my comic\nName = \"My Comic\"\nBase = \"comic\"\n\n[Elements.speaker]\nLeft = 3.0\nRight = 1.0\nStyle = \"b\"\nPostfix = \":\"\n\n" +
		"# pages first\n[[Accolade.Buttons]]\nLabel = \"Page\"\nInsert = \"===\\n\\n.PAGE {n}\"\n\n[[Accolade.Buttons]]\nLabel = \"Panel\"\nInsert = \"Panel {n}. \"\nRestart = \".PAGE\"\n"
	cdir := t.TempDir()
	cpath := filepath.Join(cdir, "lexington.toml")
	fpath := filepath.Join(userFormatDir(), "my-comic.toml")
	os.WriteFile(cpath, []byte(conf), 0o644)
	os.MkdirAll(userFormatDir(), 0o755)
	os.WriteFile(fpath, []byte(format), 0o644)
	for _, size := range []struct {
		name string
		w, h float32
	}{{"desktop", 980, 700}, {"phone", 380, 800}} {
		shots := []struct {
			path, tab, sel, val, name string
		}{
			{cpath, "General", "", "", "settings-0-general"},
			{cpath, "Languages", "language", "eo", "settings-1-language"},
			{cpath, "Element Sets", "set", "mine", "settings-2-set"},
			{cpath, "TOML", "", "", "settings-3-toml"},
			{fpath, "Format", "", "", "format-0-format"},
			{fpath, "Elements", "format-element", "speaker", "format-1-element"},
			{fpath, "Elements", "format-element", "dialog", "format-2-element-base"},
			{fpath, "Buttons", "", "", "format-3-buttons"},
			{fpath, "Help", "", "", "format-4-help"},
		}
		for _, s := range shots {
			w.openPath(s.path)
			e := w.app.tomlEditorOf(s.path)
			e.win.Resize(fyne.NewSize(size.w, size.h))
			if s.sel != "" {
				e.sel[s.sel] = s.val
			}
			selectTab(t, e, s.tab)
			e.redraw()
			if e.sample != nil {
				waitFor(t, "the sample", func() bool {
					var done bool
					fyne.DoAndWait(func() {
						done = strings.HasPrefix(e.sample.note.Text, "As Lexington") || strings.HasPrefix(e.sample.note.Text, "No sample")
					})
					return done
				})
			}
			e.win.Content().Refresh()
			f, err := os.Create(filepath.Join(dir, "tomleditor-"+size.name+"-"+s.name+".png"))
			if err != nil {
				t.Fatal(err)
			}
			png.Encode(f, e.win.Canvas().Capture())
			f.Close()
		}
	}
}
