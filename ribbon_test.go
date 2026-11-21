package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/tooltip"
)

// ribbonWindow is w in Ribbon mode.
func ribbonOnFor(t *testing.T, w *MainWindow) *ribbonBar {
	t.Helper()
	old := pandocPath // the goldens have pandoc's items (CI's runners have no pandoc)
	pandocPath = func() string { return "/usr/bin/pandoc" }
	t.Cleanup(func() { pandocPath = old })
	w.settings.SetString(lookPref, "ribbon")
	t.Cleanup(func() { w.settings.SetString(lookPref, "menus") })
	w.applyEditorSettings()
	if !w.ribbonShown() {
		t.Fatal("the ribbon does not show")
	}
	return w.ribbon
}

// dumpRibbon is the ribbon as it shows: the tabs, the one chosen, and
// its groups' buttons (L: large, T: an icon alone), greyed or checked.
func dumpRibbon(r *ribbonBar) string {
	var b strings.Builder
	var tabs []string
	for _, t := range r.shownTabs() {
		name := t.title
		if t.selected {
			name = "[" + name + "]"
		}
		tabs = append(tabs, name)
	}
	b.WriteString("tabs: " + strings.Join(tabs, " ") + "\n")
	for _, o := range r.groupBox.Objects {
		c, ok := o.(*fyne.Container)
		if !ok || len(c.Objects) < 2 {
			continue
		}
		title := c.Objects[1].(*widget.Label).Text
		b.WriteString("## " + title + "\n")
		var walk func(o fyne.CanvasObject)
		walk = func(o fyne.CanvasObject) {
			switch x := o.(type) {
			case *ribbonButton:
				size := "S"
				if x.large {
					size = "L"
				}
				if x.compact {
					size = "T" // an icon alone in a row of them
				}
				line := fmt.Sprintf("  %s %s", size, x.label)
				if x.mark != "" {
					line += " mark=" + x.mark
				} else {
					line += " icon=" + resourceName(x.icon)
				}
				if x.disabled {
					line += " (disabled)"
				}
				if x.checked {
					line += " (checked)"
				}
				line += " name=" + x.AccessibilityLabel()
				b.WriteString(line + "\n")
			case *widget.Select:
				b.WriteString("  select " + fyne.AccessibleLabel(x) + "\n")
			case *fyne.Container:
				for _, k := range x.Objects {
					walk(k)
				}
			}
		}
		walk(c.Objects[0])
	}
	return b.String()
}

// ribbonGolden compares (or, with UPDATE_RIBBON_GOLDEN=1, writes) a
// golden file of testdata/ribbon.
func ribbonGolden(t *testing.T, name, got string) {
	t.Helper()
	p := filepath.Join("testdata", "ribbon", name+".golden")
	if os.Getenv("UPDATE_RIBBON_GOLDEN") != "" {
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	w := onThisSystem(strings.ReplaceAll(string(want), "\r\n", "\n")) // the golden has Linux's keys
	if w != got {
		t.Errorf("%s differs from the golden:\n%s", name, lineDiff(w, got))
	}
}

// The ribbon's tabs and groups, from the registry: a screenplay's, one
// with a source (the Adaptation tab), a Markdown document's.
func TestRibbonGolden(t *testing.T) {
	w := newTestWindow(t, "INT. BARN - DAY\n\nRain.\n")
	r := ribbonOnFor(t, w)
	for _, tab := range []string{"File", "Write", "Edit", "Production", "Collaboration", "View"} {
		r.selectTab(tab)
		ribbonGolden(t, "screenplay-"+strings.ToLower(tab), dumpRibbon(r))
	}
	r.selectTab("Adaptation") // not there without a source: Write
	if r.selected != "Write" {
		t.Errorf("the Adaptation tab chosen without a source: %s", r.selected)
	}

	ws := suggestWindow(t, suggestScript)
	rs := ribbonOnFor(t, ws)
	rs.selectTab("Adaptation")
	ribbonGolden(t, "screenplay-source-adaptation", dumpRibbon(rs))

	w.setDocumentType(true)
	for _, tab := range []string{"File", "Write", "Edit", "Adaptation", "Collaboration", "View"} {
		r.selectTab(tab)
		ribbonGolden(t, "markdown-"+strings.ToLower(tab), dumpRibbon(r))
	}
	r.selectTab("Production")
	if r.selected != "Write" {
		t.Errorf("Markdown has a Production tab: %s", r.selected)
	}
}

// buttonOf is the ribbon's button of command id (nil if none shows).
func buttonOf(r *ribbonBar, id string) *ribbonButton {
	if bs := r.byID[id]; len(bs) > 0 {
		return bs[0]
	}
	return nil
}

// The buttons follow the script's state: a lock enables Unlock (and an
// undo of it disables it again, once typing pauses), the document's kind
// the tabs.
func TestRibbonEnablement(t *testing.T) {
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	r.selectTab("Production")
	unlock := buttonOf(r, "production.unlockNumbers")
	if unlock == nil || !unlock.Disabled() {
		t.Fatal("Unlock Scene Numbers not there, or enabled before a lock")
	}
	buttonOf(r, "production.lockNumbers").press()
	unlock = buttonOf(r, "production.unlockNumbers")
	if unlock.Disabled() {
		t.Error("Unlock disabled after the lock")
	}
	w.undo() // the text's change: the ribbon follows when typing pauses
	if !buttonOf(r, "production.unlockNumbers").Disabled() {
		t.Error("Unlock enabled after the lock was undone")
	}
	w.setDocumentType(true)
	var tabs []string
	for _, tb := range r.shownTabs() {
		tabs = append(tabs, tb.title)
	}
	if !slices.Equal(tabs, []string{"File", "Write", "Edit", "Adaptation", "Collaboration", "View"}) {
		t.Errorf("Markdown's tabs %q", tabs)
	}
	w.setDocumentType(false)
	if r.tabs[3].Visible() {
		t.Error("the Adaptation tab without a source")
	}
}

// The Adaptation tab comes when a source does (the ribbon on first),
// its commands enabled.
func TestRibbonSourceArrives(t *testing.T) {
	w, _ := scriptWindow(t, suggestScript)
	r := ribbonOnFor(t, w)
	if r.tabs[3].Visible() {
		t.Fatal("an Adaptation tab without a source")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "story.md")
	os.WriteFile(p, []byte(suggestStory), 0o644)
	if err := w.useSource(p); err != nil {
		t.Fatal(err)
	}
	if !r.tabs[3].Visible() {
		t.Error("no Adaptation tab once the source came")
	}
}

func TestRibbonSourceTab(t *testing.T) {
	w := suggestWindow(t, suggestScript)
	r := ribbonOnFor(t, w)
	r.selectTab("Adaptation")
	if r.selected != "Adaptation" {
		t.Fatal("no Adaptation tab with a source")
	}
	if b := buttonOf(r, "source.next"); b == nil || b.Disabled() || !b.large {
		t.Error("Next Source Passage: not a large enabled button")
	}
}

// The Production tab's first group is the stage's actions: Lock, then
// Revise once issued, Break Down once tagged, Schedule with a day.
func TestRibbonProductionStages(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	r.selectTab("Production")
	first := func() string {
		return r.groupBox.Objects[1].(*fyne.Container).Objects[1].(*widget.Label).Text // after the strut
	}
	if first() != "Lock" {
		t.Fatalf("first group %q", first())
	}
	issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	if first() != "Revise" || buttonOf(r, "production.revisedPages") == nil {
		t.Fatalf("after White: %q", first())
	}
	ribbonGolden(t, "screenplay-production-revise", dumpRibbon(r))
	tagNow(t, w, "Mud", 0, "Props")
	if first() != "Break Down" {
		t.Fatalf("after a tag: %q", first())
	}
	ribbonGolden(t, "screenplay-production-breakdown", dumpRibbon(r))
	w.addShootDay()
	if first() != "Schedule" {
		t.Fatalf("after a day: %q", first())
	}
	ribbonGolden(t, "screenplay-production-schedule", dumpRibbon(r))
	// the stage looked at in the pane
	p := prodPane(t, w)
	p.pick(stageLock)
	if first() != "Lock" {
		t.Errorf("the pane at Lock, the ribbon at %q", first())
	}
}

// At a phone's width: one row of tabs, one of buttons that scrolls
// sideways, every button 44 px high at least, nothing wider than the
// phone.
func TestRibbonPhone(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.fyneWindow.Canvas().Capture() // laid out at that width
	if !w.narrow || !r.phone {
		t.Fatal("not narrow")
	}
	for _, tab := range []string{"Write", "Production", "View"} {
		r.selectTab(tab)
		if got := r.box.MinSize().Width; got > 380 {
			t.Errorf("%s: the ribbon %v wide", tab, got)
		}
		for _, b := range r.buttons {
			if s := b.MinSize(); s.Height < 44 || s.Width < 44 {
				t.Errorf("%s: %s is %v", tab, b.label, s)
			}
		}
		for _, tb := range r.shownTabs() {
			if tb.MinSize().Height < 44 {
				t.Errorf("tab %s is %v", tb.title, tb.MinSize())
			}
		}
	}
	if _, ok := interface{}(r.groups).(*container.Scroll); !ok || r.groups.Direction != container.ScrollHorizontalOnly {
		t.Error("the groups do not scroll sideways")
	}
	if r.title.Visible() && r.title.Size().Width < 80 {
		t.Error("a title squeezed into the phone's tab row")
	}
}

// Key tips: unique, none the start of another, for every tab, stage,
// format and mode; F10 shows them, a letter picks a tab, then a button,
// which runs, nothing typed into the script; Esc goes back, then out.
func TestRibbonKeyTips(t *testing.T) {
	if got := keyTipsFor([]string{"Bold", "Bullets", "Back", "B", "Beta"}); !slices.Equal(got, []string{"B", "U", "A", "1", "E"}) {
		t.Errorf("tips %q", got)
	}
	many := make([]string, 70)
	for i := range many {
		many[i] = "x"
	}
	checkTips(t, "seventy", keyTipsFor(many))

	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	for _, md := range []bool{false, true} {
		w.setDocumentType(md)
		for _, f := range scriptFormatKeys() {
			w.settings.SetString("script-format", f)
			r.refresh()
			for _, tab := range r.shownTabs() {
				r.selectTab(tab.title)
				r.tipsOn = true
				r.assignTips(2)
				var tips []string
				for _, b := range r.buttons {
					if !b.Disabled() {
						tips = append(tips, b.tip)
					}
				}
				checkTips(t, fmt.Sprintf("%s md=%v %s", tab.title, md, f), tips)
				r.leaveKeyTips(false)
			}
		}
	}
	w.settings.SetString("script-format", "default")
	w.setDocumentType(false)
	r.selectTab("Write")

	text := w.textEditor.Text()
	c := w.fyneWindow.Canvas()
	c.Focus(w.textEditor)
	if !w.keyPreview(fyne.KeyF10, 0) || !r.tipsOn {
		t.Fatal("F10: no key tips")
	}
	var tabTips []string
	for _, tb := range r.shownTabs() {
		tabTips = append(tabTips, tb.tip)
	}
	checkTips(t, "tabs", append(tabTips, r.palette.tip))
	if _, ok := c.Focused().(*ribbonTabButton); !ok {
		t.Fatalf("the focus is not on the tabs: %T", c.Focused())
	}
	view := ""
	for _, tb := range r.shownTabs() {
		if tb.title == "View" {
			view = tb.tip
		}
	}
	c.Focused().TypedRune([]rune(strings.ToLower(view))[0])
	if r.selected != "View" || r.tipLevel != 2 {
		t.Fatalf("the View tab's tip %q: %s, level %d", view, r.selected, r.tipLevel)
	}
	outline := buttonOf(r, "view.outline")
	if outline.tip == "" {
		t.Fatal("Outline has no tip")
	}
	c.Focused().(fyne.Focusable).TypedRune([]rune(outline.tip)[0])
	if !w.outlineVisible || r.tipsOn {
		t.Errorf("Outline's tip: outline %v, tips %v", w.outlineVisible, r.tipsOn)
	}
	if w.textEditor.Text() != text {
		t.Error("a key tip was typed into the script")
	}
	switch c.Focused().(type) { // back to the editor, or where Outline put it
	case *ribbonTabButton, *ribbonButton:
		t.Errorf("the focus stayed on the ribbon: %T", c.Focused())
	}
	// Esc: back to the tabs' tips, then out
	w.keyPreview(fyne.KeyF10, 0)
	c.Focused().TypedRune([]rune(strings.ToLower(view))[0])
	c.Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if !r.tipsOn || r.tipLevel != 1 {
		t.Error("Esc did not go back to the tabs")
	}
	c.Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if r.tipsOn || c.Focused() != fyne.Focusable(w.textEditor) {
		t.Error("Esc did not leave the key tips")
	}
	// Alt alone: pressed and let go with no key between, the key tips
	// (the menu bar is gone in Ribbon mode, nothing else takes Alt); Alt
	// with a key shows none, nor Alt+Tab (the window loses the keyboard
	// before Alt's let-go arrives)
	alt := func() { w.keyPreview(desktop.KeyAltLeft, 0) }
	altUp := func() { w.textEditor.KeyUp(&fyne.KeyEvent{Name: desktop.KeyAltLeft}) }
	if nativeMenus {
		// a Mac: the menu bar is the system's and stays; Alt alone does
		// nothing there (F10 shows the key tips, checked above)
		if w.fyneWindow.MainMenu() == nil {
			t.Error("Ribbon mode on a Mac: the system menu bar went")
		}
	} else {
		c.Focus(w.textEditor)
		if w.fyneWindow.MainMenu() != nil {
			t.Error("Ribbon mode: the menu bar is still there (a double bar)")
		}
		alt()
		if r.tipsOn {
			t.Error("key tips before Alt was let go")
		}
		altUp()
		if !r.tipsOn {
			t.Error("Alt alone: no key tips")
		}
		c.Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if r.tipsOn || c.Focused() != fyne.Focusable(w.textEditor) {
			t.Error("Esc after Alt: tips still on, or the focus not back")
		}
		alt()
		w.keyPreview(fyne.KeyJ, fyne.KeyModifierAlt)
		altUp()
		if r.tipsOn {
			t.Error("Alt with a key showed the key tips")
		}
		alt()
		w.textEditor.FocusLost() // Alt+Tab: the window loses the keyboard
		altUp()
		if r.tipsOn {
			t.Error("Alt+Tab showed the key tips")
		}
		c.Focus(w.textEditor)
		alt() // Alt alone again, with the tips on: they go
		altUp()
		if !r.tipsOn {
			t.Fatal("Alt alone (again): no key tips")
		}
		alt()
		c.Focused().(desktop.Keyable).KeyUp(&fyne.KeyEvent{Name: desktop.KeyAltLeft}) // on the ribbon's tab
		if r.tipsOn {
			t.Error("Alt alone on the ribbon did not take the key tips away")
		}
	}
	// a click in the script while they show takes them away
	w.keyPreview(fyne.KeyF10, 0)
	c.Focus(w.textEditor)
	waitFor(t, "the key tips gone", func() bool { return !r.tipsOn })
	// in Menus mode, Alt is the menu bar's
	w.settings.SetString(lookPref, "menus")
	w.applyEditorSettings()
	if w.fyneWindow.MainMenu() == nil {
		t.Error("Menus mode: the menu bar did not come back")
	}
	c.Focus(w.textEditor)
	alt()
	altUp()
	if r.tipsOn {
		t.Error("key tips in Menus mode")
	}
}

// checkTips fails for an empty, doubled or prefix tip.
func checkTips(t *testing.T, what string, tips []string) {
	t.Helper()
	for i, a := range tips {
		if a == "" {
			t.Errorf("%s: a button without a tip", what)
		}
		for j, b := range tips {
			if i != j && strings.HasPrefix(b, a) {
				t.Errorf("%s: tips %q and %q", what, a, b)
			}
		}
	}
}

// scriptFormatKeys are the built-in formats' keys.
func scriptFormatKeys() []string {
	files, _ := builtinFormats.ReadDir("formats")
	out := []string{"default"}
	for _, f := range files {
		out = append(out, strings.TrimSuffix(f.Name(), ".toml"))
	}
	return out
}

// Tab and the arrows move between tabs and buttons; Enter presses one.
func TestRibbonKeyboard(t *testing.T) {
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	r.selectTab("View")
	c := w.fyneWindow.Canvas()
	tabs := r.shownTabs()
	c.Focus(tabs[0])
	tabs[0].TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	if c.Focused() != fyne.Focusable(tabs[1]) {
		t.Fatalf("Right: %T", c.Focused())
	}
	r.up()
	c.Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	b, ok := c.Focused().(*ribbonButton)
	if !ok || b != r.buttons[0] {
		t.Fatalf("Down: %T", c.Focused())
	}
	b.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	if c.Focused() != fyne.Focusable(r.buttons[1]) {
		t.Error("Right in the groups")
	}
	outline := buttonOf(r, "view.outline")
	c.Focus(outline)
	outline.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if !w.outlineVisible {
		t.Error("Enter did not press Outline")
	}
	if outline.AccessibilityRole() != fyne.AccessibleRoleButton || !strings.HasPrefix(outline.AccessibilityLabel(), "Outline"+keys("Shift+O")) {
		t.Errorf("Outline's name %q", outline.AccessibilityLabel())
	}
	if tabs[0].AccessibilityRole() != fyne.AccessibleRoleTab {
		t.Error("a tab's role")
	}
}

// Every tab and button shows when it has the focus.
func TestRibbonFocusVisible(t *testing.T) {
	w, _ := scriptWindow(t, revScript)
	ribbonOnFor(t, w)
	w.fyneWindow.Resize(fyne.NewSize(1200, 700))
	for _, name := range invisibleFocus(w.fyneWindow.Canvas()) {
		if strings.Contains(name, "tab") || strings.Contains(name, "ribbon") {
			t.Errorf("focus not visible: %s", name)
		}
	}
}

// Everything on the ribbon has a name for a screen reader.
func TestRibbonNames(t *testing.T) {
	w := suggestWindow(t, suggestScript)
	r := ribbonOnFor(t, w)
	for _, tab := range r.shownTabs() {
		r.selectTab(tab.title)
		if u := unnamed(w.fyneWindow.Canvas()); len(u) > 0 {
			t.Errorf("%s: no name for\n  %s", tab.title, strings.Join(u, "\n  "))
		}
	}
}

// Menus mode is as it was: the header's rows, no ribbon.
func TestMenusModeUnchanged(t *testing.T) {
	w := newTestWindow(t, "")
	if w.ribbon != nil || w.headerArea.Objects[0] != w.headerBar.container {
		t.Fatal("Menus mode is not the header's rows")
	}
	ribbonOnFor(t, w)
	w.settings.SetString(lookPref, "menus")
	w.applyEditorSettings()
	if w.headerArea.Objects[0] != w.headerBar.container {
		t.Error("back to Menus: not the header's rows")
	}
}

// TestRibbonScreenshots renders the ribbon on each tab, at a phone's
// width and with key tips (SHOT_DIR=… go test -run TestRibbonScreenshots).
func TestRibbonScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(dir, 0o755)
	w := suggestWindow(t, suggestScript)
	r := ribbonOnFor(t, w)
	w.fyneWindow.Resize(fyne.NewSize(1000, 600))
	snap := func(name string) {
		t.Helper()
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		png.Encode(f, w.fyneWindow.Canvas().Capture())
	}
	for _, tab := range []string{"File", "Write", "Edit", "Adaptation", "Production", "Collaboration", "View"} {
		r.selectTab(tab)
		snap("ribbon-" + strings.ToLower(tab))
	}
	r.selectTab("Write")
	w.keyPreview(fyne.KeyF10, 0)
	snap("ribbon-keytips-tabs")
	w.fyneWindow.Canvas().Focused().TypedRune('w')
	snap("ribbon-keytips-write")
	r.leaveKeyTips(true)
	w.setDocumentType(true)
	r.selectTab("Write")
	snap("ribbon-markdown-write")
	w.setDocumentType(false)
	w.showPalette()
	test.Type(w.palette.entry, "src")
	snap("palette-desktop")
	w.palette.close()
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	r.selectTab("Write")
	snap("ribbon-phone-write")
	r.selectTab("Production")
	snap("ribbon-phone-production")
	w.showPalette()
	snap("palette-phone")
	w.palette.close()
}

// TestRibbonShotMatrix renders every tab as it can show (SHOT_DIR, names
// SHOT_PREFIX…): both roles, a screenplay with a source and a Markdown
// document, each Production stage, at 1000 px and a phone's 380 — for a
// before/after look at the ribbon.
func TestRibbonShotMatrix(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	prefix := os.Getenv("SHOT_PREFIX")
	if prefix == "" {
		prefix = "ribbon-"
	}
	holdUI(t)
	os.MkdirAll(dir, 0o755)
	w := suggestWindow(t, suggestScript)
	r := ribbonOnFor(t, w)
	snap := func(name string) {
		t.Helper()
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, prefix+name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		png.Encode(f, w.fyneWindow.Canvas().Capture())
	}
	stages := []string{"lock", "revise", "breakdown", "schedule"}
	for _, width := range []float32{1000, 380} {
		size := "desktop"
		if width < 600 {
			size = "phone"
		}
		w.fyneWindow.Resize(fyne.NewSize(width, map[bool]float32{true: 800, false: 600}[width < 600]))
		w.fyneWindow.Canvas().Capture()
		for _, role := range []string{roleProduction, roleWriter} {
			w.workAs(role)
			for _, md := range []bool{false, true} {
				w.setDocumentType(md)
				doc := "screenplay"
				if md {
					doc = "markdown"
				}
				for _, tab := range commands().ribbon {
					if !tabShown(tab, w.cmdEnv()) {
						continue
					}
					r.selectTab(tab.title)
					name := strings.Join([]string{size, role, doc, strings.ToLower(tab.title)}, "-")
					if tab.title != "Production" {
						snap(name)
						continue
					}
					p := w.notes.production
					if !p.shown() {
						w.toggleProduction()
					}
					for i, s := range stages {
						p.pick(stageLock + i)
						r.refresh()
						snap(name + "-" + s)
					}
					w.toggleProduction()
				}
			}
		}
	}
	w.workAs(roleProduction)
	w.setDocumentType(false)
	w.settings.SetBoolean("notes-visible", false)
	// the widest Write tabs, and the key tips over the short labels
	w.fyneWindow.Resize(fyne.NewSize(1000, 600))
	for _, f := range []string{"series", "radio"} {
		w.settings.SetString("script-format", f)
		r.selectTab("Write")
		snap("desktop-production-screenplay-write-" + f)
	}
	w.settings.SetString("script-format", "default")
	for _, tab := range []string{"Write", "View"} {
		r.selectTab(tab)
		w.keyPreview(fyne.KeyF10, 0)
		w.fyneWindow.Canvas().Focused().TypedRune([]rune(strings.ToLower(tab))[0])
		snap("desktop-production-screenplay-keytips-" + strings.ToLower(tab))
		r.leaveKeyTips(true)
	}
}

// Short labels: the group says the rest, no "…" on the ribbon, the full
// label (and keys) the button's name and its tooltip; a Markdown
// document's labels say what its commands do there.
func TestRibbonShortLabels(t *testing.T) {
	w := suggestWindow(t, suggestScript)
	r := ribbonOnFor(t, w)
	for _, md := range []bool{false, true} {
		w.setDocumentType(md)
		for _, tab := range r.shownTabs() {
			r.selectTab(tab.title)
			for _, b := range r.buttons {
				if strings.HasSuffix(b.label, "…") {
					t.Errorf("%s: %q shows its …", tab.title, b.label)
				}
				if b.c != nil && b.homeLabel == "" && b.mark == "" && !strings.HasPrefix(b.AccessibilityLabel(), b.c.labelIn(r.env)) {
					t.Errorf("%s: %q is named %q, not its full label %q", tab.title, b.label, b.AccessibilityLabel(), b.c.labelIn(r.env))
				}
			}
		}
	}
	r.selectTab("View")
	if p := buttonOf(r, "view.preview"); p == nil || p.label != "Preview" || !strings.HasPrefix(p.AccessibilityLabel(), "Live Preview") {
		t.Errorf("Markdown's preview button: %+v", p)
	}
	old := tooltip.Delay
	tooltip.Delay = time.Millisecond
	t.Cleanup(func() { tooltip.Delay = old })
	w.setDocumentType(false)
	r.selectTab("Adaptation")
	next := buttonOf(r, "source.next")
	if next.label != "Next" {
		t.Fatalf("Next Source Passage shows %q", next.label)
	}
	next.MouseIn(nil)
	waitFor(t, "the tooltip", func() bool { return w.tooltips.Text() == next.AccessibilityLabel() })
	if !strings.HasPrefix(w.tooltips.Text(), "Next Source Passage") {
		t.Errorf("tooltip %q", w.tooltips.Text())
	}
	next.MouseOut()
	if w.tooltips.Text() != "" {
		t.Error("the tooltip stayed")
	}
}

// At 1000 px every tab fits without scrolling: both roles, a screenplay
// in every built-in format and a Markdown document, each Production
// stage.
func TestRibbonFitsDesktop(t *testing.T) {
	holdUI(t)
	w := suggestWindow(t, suggestScript)
	r := ribbonOnFor(t, w)
	w.fyneWindow.Resize(fyne.NewSize(1000, 700))
	w.fyneWindow.Canvas().Capture()
	check := func(what string) {
		t.Helper()
		w.fyneWindow.Canvas().Capture() // laid out
		room := r.groups.Size().Width   // what the groups have at 1000 px
		got := r.groupBox.MinSize().Width
		if testing.Verbose() {
			t.Logf("%-40s %4.0f of %4.0f", what, got, room)
		}
		if got > room {
			t.Errorf("%s: %v wide, more than %v", what, got, room)
		}
	}
	defer w.settings.SetString("script-format", "default")
	for _, role := range []string{roleProduction, roleWriter} {
		w.workAs(role)
		for _, md := range []bool{false, true} {
			w.setDocumentType(md)
			for _, tab := range r.shownTabs() {
				r.selectTab(tab.title)
				what := fmt.Sprintf("%s md=%v %s", role, md, tab.title)
				switch {
				case tab.title == "Write" && !md:
					for _, f := range scriptFormatKeys() {
						w.settings.SetString("script-format", f)
						r.refresh()
						check(what + " " + f)
					}
					w.settings.SetString("script-format", "default")
					r.refresh()
				case tab.title == "Production":
					if !w.notes.production.shown() {
						w.toggleProduction()
					}
					for s := stageLock; s <= stageSchedule; s++ {
						w.notes.production.pick(s)
						r.refresh()
						check(fmt.Sprintf("%s stage %d", what, s))
					}
					w.toggleProduction()
				default:
					check(what)
				}
			}
		}
	}
	w.workAs(roleProduction)
	w.setDocumentType(false)
}
