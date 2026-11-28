package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The palette finds commands by a fuzzy match, the label first: their
// letters in order, word starts better; its lines have the keys and the
// home path.
func TestPaletteSearch(t *testing.T) {
	w := suggestWindow(t, suggestScript)
	entries := w.paletteEntries(w.cmdEnv())
	first := func(q string) paletteEntry {
		t.Helper()
		m := paletteMatches(entries, q)
		if len(m) == 0 {
			t.Fatalf("%q: nothing", q)
		}
		return m[0]
	}
	for q, want := range map[string]string{
		"next source": "Next Source Passage",
		"nsp":         "Next Source Passage",
		"lock pages":  "Lock Pages",
		"bold":        "Bold",
		"title pa":    "Title Page…",
		"prefs":       "Preferences…",
		"fullscr":     "Fullscreen",
		"src":         "Source",
	} {
		if got := first(q).label; got != want {
			t.Errorf("%q: %q first, want %q", q, got, want)
		}
	}
	e := first("next source")
	if e.home != "Adaptation > Next Source Passage" || e.keys != "Ctrl+Alt+J" {
		t.Errorf("Next Source Passage: home %q, keys %q", e.home, e.keys)
	}
	if e := first("revised pages"); e.home != "Production > Revise > Revised Pages PDF…" {
		t.Errorf("a ribbon-only command's home: %q", e.home)
	}
	if got := paletteMatches(entries, "zzqx"); len(got) != 0 {
		t.Errorf("zzqx found %d", len(got))
	}
	for _, e := range entries {
		if e.c.id == "app.palette" {
			t.Error("the palette lists itself")
		}
	}
	// Open Recent and the like by their items
	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.label, "Document Type > ") {
			found = true
		}
	}
	if !found {
		t.Error("Document Type's items are not in the palette")
	}
	// a Markdown document's: its commands, not a screenplay's
	w.setDocumentType(true)
	md := w.paletteEntries(w.cmdEnv())
	has := func(es []paletteEntry, label string) bool {
		for _, e := range es {
			if e.label == label {
				return true
			}
		}
		return false
	}
	if has(md, "Lock Scene Numbers") || !has(md, "Heading 2") || !has(md, "Bulleted List") {
		t.Error("Markdown's palette has a screenplay's commands, or not its own")
	}
}

// Ctrl+K and F1 open it (once: again gives it the keyboard); Enter runs
// the chosen command, an editing one on the script; Esc closes it.
func TestPaletteRuns(t *testing.T) {
	w := newTestWindow(t, "INT. BARN - DAY\n\nRain falls.\n")
	c := w.fyneWindow.Canvas()
	c.Focus(w.textEditor)
	w.textEditor.Navigate(func(b *buffer.Buffer) { b.Select(17, 21) }) // "Rain"
	var item *fyne.MenuItem
	for _, m := range w.fyneWindow.MainMenu().Items {
		for _, it := range m.Items {
			if it.Shortcut != nil && shortcutText(it.Shortcut) == "Primary+K" {
				item = it
			}
		}
	}
	if item == nil || item.Label != "Command Palette…" {
		t.Fatal("no Ctrl+K menu item")
	}
	item.Action()
	if w.palette == nil || !w.palette.pop.Visible() || c.Focused() != fyne.Focusable(w.palette.entry) {
		t.Fatal("Ctrl+K: no palette with the keyboard")
	}
	overlays := len(c.Overlays().List())
	if !w.keyPreview(fyne.KeyF1, 0) || len(c.Overlays().List()) != overlays {
		t.Error("F1 with the palette open: not taken, or a second palette")
	}
	test.Type(w.palette.entry, "bold")
	if got := w.palette.shown[w.palette.chosen].label; got != "Bold" {
		t.Fatalf("chosen %q", got)
	}
	w.palette.entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if got := w.textEditor.Text(); got != "INT. BARN - DAY\n\n**Rain** falls.\n" {
		t.Errorf("Bold from the palette: %q", got)
	}
	if w.palette.pop.Visible() || c.Focused() != fyne.Focusable(w.textEditor) {
		t.Error("the palette stayed, or the editor lost the focus")
	}
	// a picker: Scene or Section Color… opens its menu
	w.keyPreview(fyne.KeyF1, 0)
	test.Type(w.palette.entry, "section color")
	w.palette.entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if _, ok := c.Overlays().Top().(*widget.PopUpMenu); !ok && c.Overlays().Top() == nil {
		t.Error("Scene or Section Color… opened nothing")
	}
	for c.Overlays().Top() != nil {
		c.Overlays().Remove(c.Overlays().Top())
	}
	// Down chooses the next; Esc closes, the focus back
	c.Focus(w.textEditor)
	w.showPalette()
	test.Type(w.palette.entry, "fold")
	w.palette.entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	if w.palette.chosen != 1 {
		t.Errorf("Down: chosen %d", w.palette.chosen)
	}
	w.palette.entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if w.palette.pop.Visible() || c.Focused() != fyne.Focusable(w.textEditor) {
		t.Error("Esc: the palette stayed, or the focus did not go back")
	}
	if strings.Contains(w.textEditor.Text(), "fold") {
		t.Error("the search was typed into the script")
	}
}

// The palette's own button: on the header's top row and the ribbon.
func TestPaletteButtons(t *testing.T) {
	w := newTestWindow(t, "")
	if w.headerBar.button("app.palette") == nil {
		t.Error("no palette button on the toolbar")
	}
	r := ribbonOnFor(t, w)
	r.palette.press()
	if w.palette == nil || !w.palette.pop.Visible() {
		t.Error("the ribbon's palette button opened nothing")
	}
}

// "beats" and "beat sheet" find the structures (Joop looked for the
// beat templates by that word): the commands with it in their labels
// first, then every structure to insert or start from, by the keywords.
func TestPaletteFindsBeatTemplates(t *testing.T) {
	w := suggestWindow(t, suggestScript)
	entries := w.paletteEntries(w.cmdEnv())
	for _, q := range []string{"beats", "beat sheet"} {
		labels := map[string]bool{}
		for _, e := range paletteMatches(entries, q) {
			labels[e.label] = true
		}
		for _, want := range []string{"Insert Beats / Structure > Three Acts", "New from Structure > Freytag's Pyramid", "Insert Beats / Structure > Six Stages"} {
			if !labels[want] {
				t.Errorf("%q: no %q", q, want)
			}
		}
	}
	// a label with the word comes before a keyword's match
	m := paletteMatches(entries, "beats")
	at := map[string]int{}
	for i, e := range m {
		at[e.label] = i
	}
	if m[0].label != "Show Beats" || at["Insert Beats / Structure > Three Acts"] > at["New from Structure > Three Acts"] {
		t.Errorf("beats: %q first, Insert Beats / Structure at %d, New from Structure at %d", m[0].label,
			at["Insert Beats / Structure > Three Acts"], at["New from Structure > Three Acts"])
	}
}
