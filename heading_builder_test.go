package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/rules"
)

// The builder's model gives every default pattern back byte for byte;
// its edits keep the text around the slots.
func TestPatternModel(t *testing.T) {
	for lang, l := range rules.DefaultLanguages() {
		for _, p := range append(l.Headings, rules.SceneForm, l.NewHeading) {
			if got := modelOf(p).String(); got != p {
				t.Errorf("%s: %q gives %q", lang, p, got)
			}
		}
	}
	m := modelOf("场景{n} {place} {time} {io}")
	for _, c := range []struct {
		got, want string
	}{
		{m.remove(0).String(), "场景{place} {time} {io}"},
		{m.remove(1).String(), "场景{n} {time} {io}"},
		{m.remove(3).String(), "场景{n} {place} {time}"},
		{m.swap(2, 3).String(), "场景{n} {place} {io} {time}"},
		{m.add(rules.SlotAny).String(), "场景{n} {place} {time} {io} {any}"},
		{modelOf("○{place}（{time}）").remove(1).String(), "○{place}"},
		{modelOf("({n})").remove(0).String(), "()"},
	} {
		if c.got != c.want {
			t.Errorf("%q, want %q", c.got, c.want)
		}
	}
	if m.String() != "场景{n} {place} {time} {io}" {
		t.Error("an edit changed the model it came from")
	}
}

// The settings editor's Headings: the builder edits the list in the
// file, comments kept; opening the page and changing language changes
// nothing; Write New Scenes So sets NewHeading.
func TestHeadingBuilderEdits(t *testing.T) {
	w := tomlTestApp(t)
	text := "# mine\n[Languages.zh]\n# Chinese, my patterns\nHeadings = [\n  \"场景{n} {place} {time} {io}\", # the XISU form\n  \"{n}. {place} {time} {io}\",\n]\nRepeat = 3\n"
	os.MkdirAll(filepath.Dir(lexingtonConfig()), 0o755)
	os.WriteFile(lexingtonConfig(), []byte(text), 0o644)
	reloadLexingtonSettings()
	w.app.prefs = nil
	e := openTOMLFile(w.app, w.fyneWindow, lexingtonConfig(), lexingtonSettingsText())
	selectTab(t, e, "Languages")
	for _, lang := range []string{"zh", "en", "hi-Deva", "zh"} {
		e.sel["language"] = lang
		e.redraw()
	}
	if e.doc.text != text {
		t.Fatalf("looking changed the text:\n%s", e.doc.text)
	}
	page := e.tabs.Selected().Content
	ex := find(page, func(l *widget.Label) bool { return strings.HasPrefix(l.Text, "Example: 场景12 客厅 日 内") })
	if ex == nil || !strings.Contains(ex.Text, "✓") {
		t.Fatalf("no example: %v", ex)
	}
	// Fountain's own first, then the two patterns
	if find(page, func(c *widget.Card) bool { return strings.HasPrefix(c.Subtitle, "Fountain's own") }) == nil {
		t.Error("no Fountain's own form")
	}
	// the text between slots typed: the example follows, the comments stay
	sep := find(page, func(en *widget.Entry) bool { return en.Text == "场景" })
	if sep == nil {
		t.Fatal("no lead entry")
	}
	sep.SetText("场 ")
	if got := e.doc.conf().Languages["zh"].Headings[0]; got != "场 {n} {place} {time} {io}" {
		t.Errorf("typed: %q", got)
	}
	if !strings.Contains(ex.Text, "场 12 客厅") {
		t.Errorf("example %q", ex.Text)
	}
	for _, kept := range []string{"# mine", "# Chinese, my patterns", "Repeat = 3"} {
		if !strings.Contains(e.doc.text, kept) {
			t.Errorf("lost %q:\n%s", kept, e.doc.text)
		}
	}
	// the second pattern the one new scenes are written by
	var writeSo []*widget.Button
	collect(e.tabs.Selected().Content, func(b *widget.Button) {
		if b.Text == "Write New Scenes So" {
			writeSo = append(writeSo, b)
		}
	})
	if len(writeSo) != 2 { // Fountain's and the second: the first is it now
		t.Fatalf("%d Write New Scenes So", len(writeSo))
	}
	test.Tap(writeSo[1])
	if got := e.doc.conf().Language("zh").HeadingForm(); got != "{n}. {place} {time} {io}" {
		t.Errorf("form %q:\n%s", got, e.doc.text)
	}
	// Add Pattern adds one that reads
	test.Tap(button(t, e, "Add Pattern"))
	hs := e.doc.conf().Languages["zh"].Headings
	if len(hs) != 3 {
		t.Fatalf("headings %v", hs)
	}
	if h, ok := exampleOf("zh", e.doc.conf().Language("zh"), hs[2]); !ok {
		t.Errorf("the added pattern's %q does not read", h)
	}
}
