package main

import (
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/docs/help"
)

// The shortcuts page and sheet list every key every command has, in each
// mode, on the desktop and in the browser; docs/help's copies (the web
// pages') are what the registry makes:
//
//	UPDATE_HELP=1 go test -run TestShortcutsHelp .
func TestShortcutsHelp(t *testing.T) {
	for _, env := range []cmdEnv{{}, {markdown: true}, {browser: true}, {markdown: true, browser: true}} {
		page, sheet := shortcutsPage(env, false), shortcutsSheet(env, false)
		n := 0
		for _, c := range commands().commands {
			if !c.applies(env) {
				continue
			}
			label := strings.TrimSuffix(c.labelIn(env), "…")
			pageKeys, sheetKeys := keysOnLine(page, "- **"+label+"** — "), keysOnSheet(sheet, label)
			for _, k := range commandKeys(c) {
				n++
				if !slices.Contains(pageKeys, k) {
					t.Errorf("%+v: the page's %q line has no %s (%s)", env, label, k, c.id)
				}
				if !slices.Contains(sheetKeys, k) {
					t.Errorf("%+v: the sheet's %q line has no %s (%s)", env, label, k, c.id)
				}
			}
		}
		if n < 25 {
			t.Errorf("%+v: only %d keys", env, n)
		}
	}
	for file, want := range map[string]string{
		"shortcuts.md":       shortcutsPage(cmdEnv{}, true),
		"cheat-shortcuts.md": shortcutsSheet(cmdEnv{}, true),
	} {
		path := filepath.Join("docs", "help", file)
		if os.Getenv("UPDATE_HELP") != "" {
			if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Errorf("%s is not what the registry makes: UPDATE_HELP=1 go test -run TestShortcutsHelp .", path)
		}
	}
}

// keysOnLine are the keys on page's line starting with prefix: "a, b"
// up to a note in brackets.
func keysOnLine(page, prefix string) []string {
	for _, l := range strings.Split(page, "\n") {
		if rest, ok := strings.CutPrefix(l, prefix); ok {
			rest, _, _ = strings.Cut(rest, " (")
			return strings.Split(rest, ", ")
		}
	}
	return nil
}

// keysOnSheet are the keys of the sheet's line for label: "- **a, b** label".
func keysOnSheet(sheet, label string) []string {
	for _, l := range strings.Split(sheet, "\n") {
		if strings.HasSuffix(l, "** "+label) && strings.HasPrefix(l, "- **") {
			return strings.Split(strings.TrimSuffix(strings.TrimPrefix(l, "- **"), "** "+label), ", ")
		}
	}
	return nil
}

func TestKeyName(t *testing.T) {
	for want, s := range map[string]fyne.Shortcut{
		"Ctrl+Shift+P": shortcut(fyne.KeyP, fyne.KeyModifierShift),
		"Alt+Up":       altKey(fyne.KeyUp),
		"F6":           plainKey(fyne.KeyF6, 0),
		"Alt+Enter":    plainKey(fyne.KeyReturn, fyne.KeyModifierAlt),
		"Ctrl+,":       shortcut(fyne.KeyComma, 0),
	} {
		if got := keyName(s); got != want {
			t.Errorf("keyName = %q, want %q", got, want)
		}
	}
}

// helpWindow is a window with page slug open in the Help tab.
func helpWindow(t *testing.T, slug string) *MainWindow {
	t.Helper()
	w := newTestWindow(t, "INT. HOUSE - DAY\n\nRain.\n")
	w.fyneWindow.Resize(fyne.NewSize(1100, 700))
	w.showHelp(slug, "")
	return w
}

// The Help menu opens each page in the Help tab, beside Notes, Source and
// Production; links go to their page and heading; Back comes back.
func TestHelpTab(t *testing.T) {
	w := helpWindow(t, "index")
	if !w.paneInSight("help") || w.help.page.Slug != "index" {
		t.Fatalf("Help not shown: in sight %v, page %q", w.paneInSight("help"), w.help.page.Slug)
	}
	for _, id := range []string{"notes", "source", "production"} {
		w.showPane(id, false)
	}
	w.showHelp("index", "")
	if tabs := dockTitles(w, dockBottom); tabs != "Notes Source Production Help" || !w.paneInSight("help") {
		t.Errorf("tabs %q, help in sight %v", tabs, w.paneInSight("help"))
	}
	if c := w.fyneWindow.Canvas().Focused(); c != w.help.search {
		t.Errorf("focus on %T", c)
	}
	if u := unnamed(w.fyneWindow.Canvas()); len(u) > 0 {
		t.Errorf("no name for %q", u)
	}
	for _, p := range help.Pages() {
		menuRun(t, w, "Help", map[string]string{"index": "Help Contents", "production": "Production Help"}[p.Slug], p.Title)
		if w.help.page.Slug != p.Slug || len(w.help.content.Objects) == 0 {
			t.Errorf("Help > %s showed %q", p.Title, w.help.page.Slug)
		}
	}
	// a link to another page's heading
	w.showHelp("index", "")
	w.help.follow("final-draft.md")
	if w.help.page.Slug != "final-draft" {
		t.Fatalf("link went to %q", w.help.page.Slug)
	}
	w.help.goBack()
	if w.help.page.Slug != "index" {
		t.Errorf("Back went to %q", w.help.page.Slug)
	}
	// a heading on the page: scrolled to it
	w.showHelp("shortcuts", "")
	secs := help.Sections(w.help.page.Markdown)
	anchor := secs[3].Anchor
	w.help.follow("#" + anchor)
	if w.help.scroll.Offset.Y <= 0 || w.help.scroll.Offset.Y != w.help.anchors[anchor].Position().Y {
		t.Errorf("not scrolled to #%s: %v", anchor, w.help.scroll.Offset)
	}
	// web addresses go to the browser
	var opened []string
	oldOpen := openWebPage
	openWebPage = func(u string) { opened = append(opened, u) }
	t.Cleanup(func() { openWebPage = oldOpen })
	w.help.follow("https://example.org/x")
	if len(opened) != 1 || opened[0] != "https://example.org/x" {
		t.Errorf("opened %q", opened)
	}
}

// menuRun runs the menu item label (or alt) of the menu bar's menu.
func menuRun(t *testing.T, w *MainWindow, menu string, labels ...string) {
	t.Helper()
	for _, m := range w.menuBar(w.cmdEnv()).Items {
		if m.Label != menu {
			continue
		}
		for _, it := range m.Items {
			for _, l := range labels {
				if l != "" && it.Label == l && it.Action != nil {
					it.Action()
					return
				}
			}
		}
	}
	t.Fatalf("no %s > %q", menu, labels)
}

// The links in the app's pages each go to a page or heading the app has.
func TestHelpLinksInApp(t *testing.T) {
	w := helpWindow(t, "index")
	h := w.helpPane()
	for _, p := range w.helpPages() {
		h.open(p.Slug, "")
		var walk func([]widget.RichTextSegment)
		links := 0
		walk = func(segs []widget.RichTextSegment) {
			for _, s := range segs {
				switch x := s.(type) {
				case *widget.HyperlinkSegment:
					links++
					if x.OnTapped == nil {
						t.Errorf("%s: link %v opens as a web address", p.Slug, x.URL)
					}
				case *widget.ParagraphSegment:
					walk(x.Texts)
				case *widget.ListSegment:
					walk(x.Items)
				}
			}
		}
		for _, o := range h.content.Objects {
			walk(o.(*widget.RichText).Segments)
		}
		if links != len(help.Links(p.Markdown)) {
			t.Errorf("%s: %d links shown, %d written", p.Slug, links, len(help.Links(p.Markdown)))
		}
	}
}

// Search lists the places with the words; a hit opens its page there.
func TestHelpSearch(t *testing.T) {
	w := helpWindow(t, "index")
	h := w.helpPane()
	test.Type(h.search, "parenthetical")
	if len(h.content.Objects) < 2 {
		t.Fatalf("no hits: %d", len(h.content.Objects))
	}
	hit := h.content.Objects[1].(*fyne.Container).Objects[0].(*widget.Button)
	test.Tap(hit)
	if h.page.Slug == "index" || h.search.Text != "" {
		t.Errorf("the hit opened %q (search %q)", h.page.Slug, h.search.Text)
	}
}

// The picker has the pages and the cheat sheets, these in the mode's
// order; the shortcuts are the mode's.
func TestHelpPickerFollowsMode(t *testing.T) {
	w := helpWindow(t, "cheat-shortcuts")
	h := w.helpPane()
	firstCheat := func() string {
		for _, o := range h.picker.Options {
			if strings.HasPrefix(o, "Cheat sheet: ") {
				return o
			}
		}
		return ""
	}
	if got := firstCheat(); got != "Cheat sheet: Keyboard Shortcuts" {
		t.Errorf("screenplay's first: %q", got)
	}
	if !strings.Contains(h.page.Markdown, "Tab** on an empty line: a cue") {
		t.Errorf("the screenplay's sheet has no Tab flow")
	}
	w.setMarkdown(true)
	w.showHelp("cheat-shortcuts", "")
	if got := firstCheat(); got != "Cheat sheet: Markdown Syntax" {
		t.Errorf("Markdown's first: %q", got)
	}
	w.showPane("notes", false)
	w.showPane("source", false) // a screenplay's: not in a Markdown document's dock
	if tabs := dockTitles(w, dockBottom); tabs != "Notes Help" {
		t.Errorf("Markdown's tabs %q", tabs)
	}
	if strings.Contains(h.page.Markdown, "Tab** on an empty line: a cue") || strings.Contains(h.page.Markdown, "Scene Heading") {
		t.Errorf("a Markdown document's sheet has the screenplay's keys")
	}
	// the picker opens what is picked
	h.picker.SetSelected("Cheat sheet: Fountain Syntax")
	if h.page.Slug != "cheat-fountain" {
		t.Errorf("picked %q", h.page.Slug)
	}
}

// Online Help opens the page shown at its address with the version;
// Send Feedback a new issue with the version, nothing of the script.
func TestHelpOnlineAndFeedback(t *testing.T) {
	var opened []string
	oldOpen := openWebPage
	openWebPage = func(u string) { opened = append(opened, u) }
	t.Cleanup(func() { openWebPage = oldOpen })
	w := helpWindow(t, "production")
	menuRun(t, w, "Help", "Online Help")
	if len(opened) != 1 || opened[0] != help.URL("production", appVersion(), "") {
		t.Errorf("Online Help opened %q", opened)
	}
	test.Tap(w.helpPane().online)
	if len(opened) != 2 || opened[1] != opened[0] {
		t.Errorf("the globe opened %q", opened)
	}
	menuRun(t, w, "Help", "Send Feedback…")
	if len(opened) != 3 || !strings.HasPrefix(opened[2], help.Issues+"?") || strings.Contains(opened[2], "Rain") ||
		!strings.Contains(opened[2], strings.ReplaceAll(appVersion(), " ", "+")) {
		t.Errorf("Send Feedback opened %q", opened[len(opened)-1])
	}
}

// Print makes the page a web page to print: a cheat sheet a card.
func TestHelpPrint(t *testing.T) {
	var printed string
	old := printFile
	printFile = func(p string) { printed = p }
	t.Cleanup(func() { printFile = old })
	w := helpWindow(t, "cheat-dood")
	test.Tap(w.helpPane().print)
	b, err := os.ReadFile(printed)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `class="card"`) || !strings.Contains(string(b), "SWF") {
		t.Errorf("not the card: %.200s", b)
	}
}

// On a phone (380 wide) the Help tab fits under the script.
func TestHelpPhone(t *testing.T) {
	w := helpWindow(t, "fountain")
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.fyneWindow.Content().Refresh()
	if mw := w.helpPane().box.MinSize().Width; mw > 380 {
		t.Errorf("the Help tab is %v wide", mw)
	}
	if dir := os.Getenv("SHOT_DIR"); dir != "" {
		snapHelp(t, w, dir, "phone-help-fountain")
		w.showHelp("cheat-fountain", "")
		snapHelp(t, w, dir, "phone-help-cheat-fountain")
	}
}

// TestHelpScreenshots renders the Help tab with the Fountain reference
// and a cheat sheet (SHOT_DIR as in screenshot_test.go).
func TestHelpScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	w := helpWindow(t, "fountain")
	w.fyneWindow.Resize(fyne.NewSize(1200, 760))
	if w.sideSplit != nil {
		w.sideSplit.SetOffset(0.42)
	}
	snapHelp(t, w, dir, "help-fountain")
	w.showHelp("cheat-shortcuts", "")
	snapHelp(t, w, dir, "help-cheat-shortcuts")
	w.showHelp("cheat-breakdown", "")
	snapHelp(t, w, dir, "help-cheat-breakdown")
}

func snapHelp(t *testing.T, w *MainWindow, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
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
