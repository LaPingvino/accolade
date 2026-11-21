package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/docs/help"
	"github.com/LaPingvino/accolade/internal/tooltip"
)

// The commands' golden files (testdata/commands/) hold the menu bar, the
// shortcuts, the header bar's two rows and the editor's context menu as
// they were before the command registry (commands.go): written once from
// main's code with UPDATE_COMMANDS_GOLDEN=1, then kept. The registry must
// give the same, except for the moves the Adaptation menu made, which
// the test applies to the golden itself (adaptationMoves).

// shortcutText is a shortcut the same on every system: Primary for
// Ctrl (Cmd on macOS).
func shortcutText(s fyne.Shortcut) string {
	ks, ok := s.(fyne.KeyboardShortcut)
	if !ok {
		return s.ShortcutName()
	}
	var mods []string
	m := ks.Mod()
	if m&fyne.KeyModifierShortcutDefault != 0 {
		mods = append(mods, "Primary")
		m &^= fyne.KeyModifierShortcutDefault
	}
	for _, x := range []struct {
		m    fyne.KeyModifier
		name string
	}{{fyne.KeyModifierControl, "Control"}, {fyne.KeyModifierSuper, "Super"}, {fyne.KeyModifierShift, "Shift"}, {fyne.KeyModifierAlt, "Alt"}} {
		if m&x.m != 0 {
			mods = append(mods, x.name)
		}
	}
	return strings.Join(append(mods, string(ks.Key())), "+")
}

// dumpMenuItems writes items one a line, indented by depth.
func dumpMenuItems(b *strings.Builder, items []*fyne.MenuItem, depth int) {
	for _, it := range items {
		b.WriteString(strings.Repeat("  ", depth))
		if it.IsSeparator {
			b.WriteString("---\n")
			continue
		}
		b.WriteString(it.Label)
		if it.Shortcut != nil {
			b.WriteString(" [" + shortcutText(it.Shortcut) + "]")
		}
		if it.Icon != nil {
			b.WriteString(" icon=" + it.Icon.Name())
		}
		if it.Checked {
			b.WriteString(" (checked)")
		}
		if it.Disabled {
			b.WriteString(" (disabled)")
		}
		if it.Action == nil && it.ChildMenu == nil {
			b.WriteString(" (no action)")
		}
		b.WriteString("\n")
		if it.ChildMenu != nil {
			dumpMenuItems(b, it.ChildMenu.Items, depth+1)
		}
	}
}

// dumpMainMenu is the menu bar, menu after menu.
func dumpMainMenu(m *fyne.MainMenu) string {
	var b strings.Builder
	for _, menu := range m.Items {
		b.WriteString("# " + menu.Label + "\n")
		dumpMenuItems(&b, menu.Items, 1)
	}
	return b.String()
}

// dumpShortcuts is the shortcut table: each shortcut and the label of the
// item it runs, sorted by shortcut.
func dumpShortcuts(m *fyne.MainMenu) string {
	var lines []string
	var walk func(items []*fyne.MenuItem)
	walk = func(items []*fyne.MenuItem) {
		for _, it := range items {
			if it.ChildMenu != nil {
				walk(it.ChildMenu.Items)
			}
			if it.Shortcut != nil {
				lines = append(lines, shortcutText(it.Shortcut)+" = "+it.Label)
			}
		}
	}
	for _, menu := range m.Items {
		walk(menu.Items)
	}
	slices.Sort(lines)
	return strings.Join(lines, "\n") + "\n"
}

// dumpFullscreenShortcuts are the shortcuts full screen adds to the
// canvas (the menu bar gone), sorted.
func dumpFullscreenShortcuts(w *MainWindow) string {
	var lines []string
	for _, sc := range w.menuShortcuts {
		lines = append(lines, shortcutText(sc.shortcut))
	}
	slices.Sort(lines)
	return strings.Join(lines, "\n") + "\n"
}

// dumpRowObject is one thing on a header row.
func dumpRowObject(o fyne.CanvasObject) string {
	switch x := o.(type) {
	case *tooltip.Button:
		s := "button " + resourceName(x.Icon) + " tip=" + x.Tip
		if x.Disabled() {
			s += " (disabled)"
		}
		return s
	case *widget.Button:
		s := fmt.Sprintf("button %s text=%q label=%q", resourceName(x.Icon), x.Text, fyne.AccessibleLabel(x))
		if x.Disabled() {
			s += " (disabled)"
		}
		return s
	case *widget.Select:
		return "select label=" + fyne.AccessibleLabel(x)
	case *fyne.Container:
		if len(x.Objects) == 0 {
			return "gap"
		}
		var parts []string
		for _, k := range x.Objects {
			parts = append(parts, dumpRowObject(k))
		}
		return "box[" + strings.Join(parts, "; ") + "]"
	}
	return reflect.TypeOf(o).String()
}

func resourceName(r fyne.Resource) string {
	if r == nil {
		return "-"
	}
	return r.Name()
}

// dumpOverflowRow is a header row's items, with what each is in the
// row's "more" menu.
func dumpOverflowRow(b *strings.Builder, r *overflowRow) {
	for _, it := range r.items {
		if it.off {
			continue // put away (a role's)
		}
		if c, ok := it.obj.(*fyne.Container); ok && len(c.Objects) == 0 {
			b.WriteString("  gap\n")
			continue
		}
		line := "  " + dumpRowObject(it.obj)
		if it.keep {
			line += " (kept)"
		}
		b.WriteString(line + "\n")
		if it.menu != nil {
			var mb strings.Builder
			dumpMenuItems(&mb, it.menu(nil), 2)
			for _, l := range strings.Split(strings.TrimRight(mb.String(), "\n"), "\n") {
				b.WriteString("    menu:" + strings.TrimLeft(l, " ") + "\n")
			}
		}
	}
}

// findOverflowRows are the overflow rows in o, in order.
func findOverflowRows(o fyne.CanvasObject) []*overflowRow {
	var out []*overflowRow
	if c, ok := o.(*fyne.Container); ok {
		if r, ok := c.Layout.(*overflowRow); ok {
			return []*overflowRow{r}
		}
		for _, k := range c.Objects {
			out = append(out, findOverflowRows(k)...)
		}
	}
	return out
}

// dumpHeader is the header bar's rows: the top one and the editing row.
func dumpHeader(hb *HeaderBar) string {
	var b strings.Builder
	for i, o := range hb.container.Objects {
		for _, r := range findOverflowRows(o) {
			fmt.Fprintf(&b, "# row %d\n", i)
			dumpOverflowRow(&b, r)
		}
	}
	return b.String()
}

// dumpContextMenu is the editor's right-click menu, as it pops up.
func dumpContextMenu(w *MainWindow) string {
	c := w.fyneWindow.Canvas()
	for c.Overlays().Top() != nil {
		c.Overlays().Remove(c.Overlays().Top())
	}
	w.textEditor.OnSecondaryTapped(fyne.NewPos(10, 10))
	top := c.Overlays().Top()
	if top == nil {
		return "(none)\n"
	}
	defer c.Overlays().Remove(top)
	var items []fyne.CanvasObject
	switch x := top.(type) {
	case *widget.PopUpMenu:
		items = x.Items
	default: // an overlay around the menu
		v := reflect.ValueOf(top)
		if v.Kind() == reflect.Pointer {
			v = v.Elem()
		}
		if f := v.FieldByName("Content"); f.IsValid() {
			switch m := f.Interface().(type) {
			case *widget.Menu:
				items = m.Items
			case *widget.PopUpMenu:
				items = m.Items
			default:
				return "(not a menu: " + reflect.TypeOf(f.Interface()).String() + ")\n"
			}
		}
	}
	var b strings.Builder
	for _, o := range items {
		label := fyne.AccessibleLabel(o)
		if label == "" {
			label = "---"
		}
		b.WriteString(label + "\n")
	}
	return b.String()
}

// commandsGolden is everything the golden files hold, by file name.
func commandsGolden(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, pandoc := range []bool{false, true} {
		old := pandocPath
		if pandoc {
			pandocPath = func() string { return "/usr/bin/pandoc" }
		} else {
			pandocPath = func() string { return "" }
		}
		for _, md := range []bool{false, true} {
			w := newTestWindow(t, "INT. BARN - DAY\n\nRain.\n")
			mode := "screenplay"
			if md {
				w.setDocumentType(true)
				mode = "markdown"
			}
			suffix := ""
			if pandoc {
				suffix = "-pandoc"
			}
			m := w.fyneWindow.MainMenu()
			out["menu-"+mode+suffix+".golden"] = dumpMainMenu(m)
			out["shortcuts-"+mode+suffix+".golden"] = dumpShortcuts(m)
			if !pandoc {
				out["header-"+mode+".golden"] = dumpHeader(w.headerBar)
				out["context-"+mode+".golden"] = dumpContextMenu(w)
				if !md {
					w.lastTagCategory = "Props"
					out["context-screenplay-tagged.golden"] = dumpContextMenu(w)
				}
				w.toggleFullscreen()
				out["fullscreen-"+mode+".golden"] = dumpFullscreenShortcuts(w)
				w.toggleFullscreen()
			}
			if !md { // and as a writer sees it (role.go): writerMoves
				w.lastTagCategory = ""
				workAsFor(t, w, roleWriter)
				m := w.fyneWindow.MainMenu()
				out["menu-"+mode+suffix+"-writer.golden"] = dumpMainMenu(m)
				out["shortcuts-"+mode+suffix+"-writer.golden"] = dumpShortcuts(m)
				if !pandoc {
					out["header-"+mode+"-writer.golden"] = dumpHeader(w.headerBar)
					out["context-"+mode+"-writer.golden"] = dumpContextMenu(w)
					w.lastTagCategory = "Props"
					out["context-screenplay-tagged-writer.golden"] = dumpContextMenu(w)
					w.toggleFullscreen()
					out["fullscreen-"+mode+"-writer.golden"] = dumpFullscreenShortcuts(w)
					w.toggleFullscreen()
				}
				workAsFor(t, w, roleProduction)
			}
		}
		pandocPath = old
	}
	return out
}

// keysRe is a tip's shortcut as the golden has it (Linux's).
var keysRe = regexp.MustCompile(` \(Ctrl\+([^)]*)\)`)

// onThisSystem is golden text with tips' shortcuts as keys shows them here.
func onThisSystem(s string) string {
	if runtime.GOOS != "darwin" {
		return s
	}
	return keysRe.ReplaceAllStringFunc(s, func(m string) string {
		return keys(keysRe.FindStringSubmatch(m)[1])
	})
}

func readCommandsGolden(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "commands", name))
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n") // a Windows checkout may give CRLF
}

// TestCommandsGolden: the menus, shortcuts, header bar and context menu
// as main's code made them, with the Adaptation menu's moves.
func TestCommandsGolden(t *testing.T) {
	got := commandsGolden(t)
	dir := filepath.Join("testdata", "commands")
	if os.Getenv("UPDATE_COMMANDS_GOLDEN") != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for name, s := range got {
			if strings.HasSuffix(name, "-writer.golden") {
				continue // the moves make them (writerMoves)
			}
			if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	for name, s := range got {
		base, writer := strings.CutSuffix(name, "-writer.golden")
		if writer {
			base += ".golden"
		}
		want := toolbarMoves(base, dockMoves(base, modeLabelMoves(base, roleMoves(base, regroupMoves(base, helpMoves(base, adaptationMoves(base, readCommandsGolden(t, base)))))))) // the moves write Linux keys too
		if writer {
			want = writerMoves(base, want)
		}
		want = onThisSystem(want)
		if s != want {
			t.Errorf("%s differs from the golden (with the Adaptation, Help and regrouping moves):\n%s", name, lineDiff(want, s))
		}
	}
}

// adaptationMenu is the Adaptation menu as the golden files would have
// it: the source's items out of File and Format, in a menu of their own.
const adaptationMenu = `# Adaptation
  Choose Source…
  Save Source As…
  Keep Source in the Script
  Source Pane
  ---
  Link to Source Passage
  Next Source Passage [Primary+Alt+J]
  Mark Source Passage as Cut
  Suggest Links to the Source
  Update Source Links
  Source Headings as Sections
  ---
  What Accolade Learned…
`

// adaptationMoves is golden text with the moves the Adaptation menu made,
// and only those: in a screenplay's menu bar, Choose Source… and Save
// Source As… leave File, the source's group leaves Format (its separator
// too), and the Adaptation menu comes before Production; in a Markdown
// document's, Format loses the separator it ended with (the source's
// group's, left behind). And Renumber Scenes became Renumber Scenes… (a
// dialog of strategies since). The shortcuts, header bar and context menu
// are as they were.
func adaptationMoves(name, golden string) string {
	if !strings.HasPrefix(name, "menu-") {
		return golden
	}
	if strings.HasPrefix(name, "menu-markdown") {
		return strings.Replace(golden, "  ---\n# View\n", "# View\n", 1)
	}
	golden = strings.Replace(golden, "  Choose Source…\n", "", 1)
	golden = strings.Replace(golden, "  Save Source As…\n", "", 1)
	golden = strings.Replace(golden, `  ---
  Link to Source Passage
  Next Source Passage [Primary+Alt+J]
  Mark Source Passage as Cut
  Source Headings as Sections
  Suggest Links to the Source
  Update Source Links
  Keep Source in the Script
`, "", 1)
	golden = strings.Replace(golden, "  Renumber Scenes\n", "  Renumber Scenes…\n", 1) // a dialog since its strategies
	return strings.Replace(golden, "# Production\n", adaptationMenu+"# Production\n", 1)
}

// helpMoves applies the Help menu's pages, cheat sheets, Online Help and
// Send Feedback (help_pane.go), and View > Help, to a menu golden.
func helpMoves(name, golden string) string {
	if !strings.HasPrefix(name, "menu-") {
		return golden
	}
	var b strings.Builder
	b.WriteString("# Help\n")
	for _, p := range help.Pages() {
		label := p.Title
		switch p.Slug {
		case "index":
			label = "Help Contents"
		case "production":
			label = "Production Help"
		}
		b.WriteString("  " + label + "\n")
	}
	b.WriteString("  Cheat Sheets\n")
	for _, p := range help.CheatsFor(strings.HasPrefix(name, "menu-markdown")) {
		b.WriteString("    " + p.Title + "\n")
	}
	b.WriteString("  ---\n  Online Help\n  Send Feedback…\n  ---\n  About Accolade\n")
	golden = strings.Replace(golden, "# Help\n  About Accolade\n", b.String(), 1)
	return strings.Replace(golden, "  Changes [Primary+Shift+H]\n", "  Help\n  Changes [Primary+Shift+H]\n", 1)
}

// lineDiff shows the lines only in want (-) and only in got (+), in order.
func lineDiff(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	var b strings.Builder
	i, j := 0, 0
	for i < len(w) || j < len(g) {
		switch {
		case i < len(w) && j < len(g) && w[i] == g[j]:
			i++
			j++
		case j < len(g) && !slices.Contains(w[i:], g[j]):
			b.WriteString("+ " + g[j] + "\n")
			j++
		case i < len(w):
			b.WriteString("- " + w[i] + "\n")
			i++
		default:
			b.WriteString("+ " + g[j] + "\n")
			j++
		}
	}
	return b.String()
}

// The regrouping of the menus (Joop, 2026-11-20), the palette and the
// Markdown commands, as explicit moves on the golden text, so nothing
// else changes unnoticed: File keeps new, open, import, save and the
// ways out; Edit gets Cut, Copy, Paste, Select All, Move Line Up/Down
// (from Format), Title Page… (from File), Script Language… (from Format:
// the script's own, like its title page) and Preferences… (from File);
// the Collaboration menu, before View, has Edit Together…, Join a Shared
// Script… (from File) · Save Draft…, Drafts… (from File) and Compare…;
// Help starts with Command Palette… (Ctrl+K); a Markdown document's
// Format has its own commands; the top row the palette's button, the
// Markdown row the common Markdown.

// goldenMenu is a menu of a golden file: its title and its items, each
// with its submenu's lines.
type goldenMenu struct {
	title string
	items []string // an item's line and its submenu's, joined
}

func parseGoldenMenus(golden string) []goldenMenu {
	var ms []goldenMenu
	for _, line := range strings.Split(strings.TrimRight(golden, "\n"), "\n") {
		switch {
		case strings.HasPrefix(line, "# "):
			ms = append(ms, goldenMenu{title: line[2:]})
		case strings.HasPrefix(line, "    "):
			m := &ms[len(ms)-1]
			m.items[len(m.items)-1] += "\n" + line
		default:
			m := &ms[len(ms)-1]
			m.items = append(m.items, line)
		}
	}
	return ms
}

func (m goldenMenu) labelOf(item string) string {
	l := strings.TrimSpace(strings.SplitN(item, "\n", 2)[0])
	if i := strings.Index(l, " ["); i >= 0 {
		l = l[:i]
	}
	return l
}

// take takes the item labelled label out of menu title ("" if none).
func takeItem(ms []goldenMenu, title, label string) string {
	for i := range ms {
		if ms[i].title != title {
			continue
		}
		for j, it := range ms[i].items {
			if ms[i].labelOf(it) == label {
				ms[i].items = slices.Delete(ms[i].items, j, j+1)
				return it
			}
		}
	}
	return ""
}

// tidy drops doubled separators and those at a menu's ends.
func tidy(items []string) []string {
	var out []string
	for _, it := range items {
		if it == "  ---" && (len(out) == 0 || out[len(out)-1] == "  ---") {
			continue
		}
		if it != "" {
			out = append(out, it)
		}
	}
	for len(out) > 0 && out[len(out)-1] == "  ---" {
		out = out[:len(out)-1]
	}
	return out
}

func joinMenus(ms []goldenMenu) string {
	var b strings.Builder
	for _, m := range ms {
		b.WriteString("# " + m.title + "\n")
		for _, it := range tidy(m.items) {
			b.WriteString(it + "\n")
		}
	}
	return b.String()
}

// markdownFormat is what a Markdown document's Format menu gained after
// Italic.
const markdownFormat = `  Strikethrough
  Inline Code
  ---
  Heading 1 [Primary+1]
  Heading 2 [Primary+2]
  Heading 3 [Primary+3]
  Heading 4 [Primary+4]
  Heading 5 [Primary+5]
  Heading 6 [Primary+6]
  Normal Text [Primary+0]
  ---
  Link… [Primary+L]
  Image…
  Footnote
  ---
  Bulleted List
  Numbered List
  Task List
  Quote
  ---
  Code Block
  Horizontal Rule
  Table…`

// markdownKeys are the Markdown commands' keys.
var markdownKeys = map[string]string{"Primary+0": "Normal Text", "Primary+1": "Heading 1", "Primary+2": "Heading 2", "Primary+3": "Heading 3",
	"Primary+4": "Heading 4", "Primary+5": "Heading 5", "Primary+6": "Heading 6", "Primary+L": "Link…"}

// markdownRowGolden is the Markdown editing row.
const markdownRowGolden = `# row 1
  button foreground_heading.svg text="Heading" label="Heading level, or normal text"
    menu:Heading icon=foreground_heading.svg
  button foreground_bold.svg tip=Bold (Ctrl+B)
    menu:Bold icon=foreground_bold.svg
  button foreground_italic.svg tip=Italic (Ctrl+I)
    menu:Italic icon=foreground_italic.svg
  button foreground_strikethrough.svg tip=Strikethrough
    menu:Strikethrough icon=foreground_strikethrough.svg
  button foreground_code.svg tip=Inline Code
    menu:Inline Code icon=foreground_code.svg
  button foreground_link.svg tip=Link… (Ctrl+L)
    menu:Link… icon=foreground_link.svg
  gap
  button foreground_bullets.svg tip=Bulleted List
    menu:Bulleted List icon=foreground_bullets.svg
  button foreground_numbers.svg tip=Numbered List
    menu:Numbered List icon=foreground_numbers.svg
  button foreground_tasks.svg tip=Task List
    menu:Task List icon=foreground_tasks.svg
  button foreground_quote.svg tip=Quote
    menu:Quote icon=foreground_quote.svg
  gap
  button foreground_document-create.svg text="Start a Screenplay" label="Start a Screenplay from This Source"
    menu:Start a Screenplay from This Source icon=foreground_document-create.svg
`

func regroupMoves(name, golden string) string {
	md := strings.Contains(name, "markdown")
	switch {
	case strings.HasPrefix(name, "shortcuts-"), strings.HasPrefix(name, "fullscreen-"):
		lines := strings.Split(strings.TrimRight(golden, "\n"), "\n")
		add := func(key, label string) {
			if strings.HasPrefix(name, "shortcuts-") {
				lines = append(lines, key+" = "+label)
			} else {
				lines = append(lines, key)
			}
		}
		add("Primary+K", "Command Palette…")
		if md {
			for k, l := range markdownKeys {
				add(k, l)
			}
		}
		slices.Sort(lines)
		return strings.Join(lines, "\n") + "\n"
	case strings.HasPrefix(name, "header-"):
		golden = strings.Replace(golden, "    menu:Find and replace icon=foreground_search.svg\n",
			"    menu:Find and replace icon=foreground_search.svg\n"+
				"  button foreground_palette.svg tip=Command palette: find any command (Ctrl+K)\n"+
				"    menu:Command palette: find any command icon=foreground_palette.svg\n", 1)
		if md {
			golden = golden[:strings.Index(golden, "# row 1\n")] + markdownRowGolden
		}
		return golden
	case !strings.HasPrefix(name, "menu-"):
		return golden
	}
	// Production: Withdraw Last Issue… after Issue Draft… (issue_undo.go)
	golden = strings.Replace(golden, "  Issue Draft…\n", "  Issue Draft…\n  Withdraw Last Issue…\n", 1)
	// File: Export Clean Fountain… after Export…, a screenplay's (clean_fountain.go)
	if !md {
		golden = strings.Replace(golden, "  Export… [Primary+E]\n", "  Export… [Primary+E]\n  Export Clean Fountain…\n", 1)
	}
	// File: Rename… and Undo Rename after Save As (relocate.go)
	golden = strings.Replace(golden, "  Save As… [Primary+Shift+S]\n", "  Save As… [Primary+Shift+S]\n  Rename…\n  Undo Rename\n", 1)
	ms := parseGoldenMenus(golden)
	together, join := takeItem(ms, "File", "Edit Together…"), takeItem(ms, "File", "Join a Shared Script…")
	saveDraft, drafts := takeItem(ms, "File", "Save Draft…"), takeItem(ms, "File", "Drafts…")
	titlePage, prefs := takeItem(ms, "File", "Title Page…"), takeItem(ms, "File", "Preferences…")
	up, down := takeItem(ms, "Format", "Move Line Up"), takeItem(ms, "Format", "Move Line Down")
	language := takeItem(ms, "Format", "Script Language…")
	var out []goldenMenu
	for _, m := range ms {
		switch m.title {
		case "Edit":
			// Undo, Redo · the clipboard · Find … · the lines · the script's · Preferences
			items := append([]string{}, m.items[:2]...)
			items = append(items, "  ---", "  Cut", "  Copy", "  Paste", "  Select All")
			items = append(items, m.items[2:]...)
			items = append(items, "  ---", up, down, "  ---", titlePage, language, "  ---", prefs)
			m.items = items
		case "Format":
			if md {
				for i, it := range m.items {
					if m.labelOf(it) == "Italic" {
						m.items = slices.Insert(m.items, i+1, strings.Split(markdownFormat, "\n")...)
						break
					}
				}
			}
		case "View":
			out = append(out, goldenMenu{title: "Collaboration", items: []string{together, join, "  ---", saveDraft, drafts, "  Compare…"}})
		case "Help":
			m.items = append([]string{"  Command Palette… [Primary+K]", "  ---"}, m.items...)
		}
		out = append(out, m)
	}
	return joinMenus(out)
}

// workAsFor makes the window work as role (the preferences'), put back
// to Production after the test.
func workAsFor(t *testing.T, w *MainWindow, role string) {
	t.Helper()
	t.Cleanup(func() {
		w.settings.SetString(rolePref, roleProduction)
		w.settings.SetStringSlice(scriptRolesPref, nil)
	})
	w.workAs(role)
}

// roleMoves: View > Working As (role.go) after Document Type, in a
// screenplay's menu bar, working as Production.
func roleMoves(name, golden string) string {
	if !strings.HasPrefix(name, "menu-screenplay") {
		return golden
	}
	return strings.Replace(golden, "    Markdown\n# Help\n",
		"    Markdown\n  Working As\n    Writer\n    Production (checked)\n    ---\n    Only for This Script\n# Help\n", 1)
}

// modeLabelMoves: a Markdown document's labels say what the command
// does there (ribbon polish): New and New Empty make a screenplay, and
// the preview is the live one beside the editor, not the PDF.
func modeLabelMoves(name, golden string) string {
	if !strings.Contains(name, "-markdown") {
		return golden
	}
	for _, r := range [][2]string{
		{"  New [", "  New Screenplay ["}, {"  New Empty [", "  New Empty Screenplay ["},
		{"= New\n", "= New Screenplay\n"}, {"= New Empty\n", "= New Empty Screenplay\n"},
		{"  Preview [", "  Live Preview ["}, {"= Preview\n", "= Live Preview\n"},
		{"tip=New (", "tip=New screenplay ("}, {"menu:New icon=", "menu:New screenplay icon="},
		{"tip=Preview (", "tip=Live Preview ("}, {"menu:Preview icon=", "menu:Live Preview icon="},
	} {
		golden = strings.ReplaceAll(golden, r[0], r[1])
	}
	return golden
}

// dockMoves: the side dock (dock.go) gathers its panes in the View menu,
// after the folding, between separators, in the dock's order (Outline
// moves down from beside Preview to join them), and adds its own: Side
// Panes (all in and out), Move Pane to Top and Move Pane to Bottom;
// Characters, a pane since, after Notes. Nothing else: the keys, header, context menu and full screen are as
// they were.
func dockMoves(name, golden string) string {
	if !strings.HasPrefix(name, "menu-") {
		return golden
	}
	ms := parseGoldenMenus(golden)
	var panes []string
	for _, l := range []string{"Outline", "Notes", "Cards", "Source", "Production", "Help", "Changes"} { // Cards a pane too (docs/SWAP.md)
		if it := takeItem(ms, "View", l); it != "" {
			panes = append(panes, it)
		}
		if l == "Notes" {
			panes = append(panes, "  Characters") // a pane of its own since (characters_pane.go)
		}
	}
	panes = append(panes, "  Side Panes", "  Move Pane to Top", "  Move Pane to Bottom", "  Swap Pane with Main") // docs/SWAP.md
	for i := range ms {
		if ms[i].title == "Adaptation" { // its common case of the swap (docs/SWAP.md)
			for j, it := range ms[i].items {
				if ms[i].labelOf(it) == "Source Pane" {
					ms[i].items = slices.Insert(ms[i].items, j+1, "  Swap Source and Script")
					break
				}
			}
		}
		if ms[i].title != "View" {
			continue
		}
		for j, it := range ms[i].items {
			if ms[i].labelOf(it) == "Unfold All" {
				in := append(append([]string{"  ---"}, panes...), "  ---")
				ms[i].items = slices.Insert(ms[i].items, j+1, in...)
				break
			}
		}
	}
	return joinMenus(ms)
}

// toolbarMoves: View > Toolbar (ribbon_settings.go) shows and hides the
// toolbar, after Fullscreen; in the ☰ menu too, which is the menu bar.
// Nothing else.
func toolbarMoves(name, golden string) string {
	if !strings.HasPrefix(name, "menu-") {
		return golden
	}
	ms := parseGoldenMenus(golden)
	for i := range ms {
		if ms[i].title != "View" {
			continue
		}
		for j, it := range ms[i].items {
			if ms[i].labelOf(it) == "Fullscreen" {
				ms[i].items = slices.Insert(ms[i].items, j+1, "  Toolbar (checked)") // shown
				break
			}
		}
	}
	return joinMenus(ms)
}

// writerMoves is a Production golden as a writer sees it (role.go): the
// Production menu gone, and View's Production pane; Working As on
// Writer; Tag Selection's key with its menu; the header's Production
// button; the context menu's tagging. Nothing else.
func writerMoves(name, golden string) string {
	switch {
	case strings.HasPrefix(name, "menu-"):
		ms := parseGoldenMenus(golden)
		takeItem(ms, "View", "Production")
		var out []goldenMenu
		for _, m := range ms {
			if m.title == "Production" {
				continue
			}
			for i, it := range m.items {
				if m.title == "View" && m.labelOf(it) == "Working As" {
					it = strings.Replace(it, "    Writer\n", "    Writer (checked)\n", 1)
					m.items[i] = strings.Replace(it, "    Production (checked)", "    Production", 1)
				}
			}
			out = append(out, m)
		}
		return joinMenus(out)
	case strings.HasPrefix(name, "shortcuts-"):
		return strings.Replace(golden, "Primary+Shift+B = Tag Selection…\n", "", 1)
	case strings.HasPrefix(name, "fullscreen-"):
		return strings.Replace(golden, "Primary+Shift+B\n", "", 1)
	case strings.HasPrefix(name, "header-"):
		return strings.Replace(golden, "  button foreground_production.svg tip=Production: lock, revise, and the stages to the shoot\n"+
			"    menu:Production: lock, revise, and the stages to the shoot icon=foreground_production.svg\n", "", 1)
	case strings.HasPrefix(name, "context-"):
		golden = strings.Replace(golden, "Tag as Props\n", "", 1)
		return strings.Replace(golden, "Tag Selection…\n---\n", "", 1)
	}
	return golden
}
