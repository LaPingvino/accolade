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
	// a Mac's own keys where they are others (macKeys): Replace's Cmd+H
	// hides the app there
	s = strings.ReplaceAll(s, "Replace… (Ctrl+H)", "Replace… (Ctrl+Alt+F)")
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
		want := notesMoves(base, templateMoves(base, keyboardMoves(base, highlandMoves(base, numberScenesMoves(base, bundleFolderMoves(base, newSceneMoves(base, referencesMoves(base, stagesMoves(base, mergeMoves(base, tableReadMoves(base, frameMoves(base, topRowMoves(base, stripMoves(base, structureMoves(base, fixLinesMoves(base, linkUpMoves(base, passageMoves(base, sampleMoves(base, toolbarMoves(base, middleMoves(base, dockMoves(base, modeLabelMoves(base, roleMoves(base, regroupMoves(base, helpMoves(base, adaptationMoves(base, readCommandsGolden(t, base)))))))))))))))))))))))))))) // the moves write Linux keys too
		if writer {
			want = writerMoves(base, want)
		}
		want = onThisSystem(want)
		if s != want {
			t.Errorf("%s differs from the golden (with the Adaptation, Help and regrouping moves):\n%s", name, lineDiff(want, s))
		}
	}
}

// keyboardMoves: the keyboard pass (keyboard.go, docs/KEYBOARD.md): on a
// Mac, Replace… is Cmd+Option+F (Cmd+H hides the app there); the
// markers have Highland's keys.
func keyboardMoves(name, golden string) string {
	mac := runtime.GOOS == "darwin"
	// the markers' keys, Highland's (⌃M, ⌃⌥M on a Mac)
	markers := [][2]string{{"Primary+M", "Marker…"}, {"Primary+Alt+M", "Next Marker"}, {"Primary+Shift+Alt+M", "Previous Marker"}}
	if mac {
		markers = [][2]string{{"Control+M", "Marker…"}, {"Control+Alt+M", "Next Marker"}, {"Control+Shift+Alt+M", "Previous Marker"}}
	}
	screenplay := strings.Contains(name, "screenplay")
	if screenplay && strings.HasPrefix(name, "menu-") {
		for _, m := range markers {
			golden = strings.Replace(golden, "  "+m[1]+"\n", "  "+m[1]+" ["+m[0]+"]\n", 1)
		}
	}
	if mac {
		golden = strings.Replace(golden, "  Replace… [Primary+H]\n", "  Replace… [Primary+Alt+F]\n", 1)
	}
	if !strings.HasPrefix(name, "shortcuts-") && !strings.HasPrefix(name, "fullscreen-") {
		return golden
	}
	lines := strings.Split(strings.TrimSuffix(golden, "\n"), "\n")
	for i, l := range lines {
		switch {
		case mac && l == "Primary+H = Replace…":
			lines[i] = "Primary+Alt+F = Replace…"
		case mac && l == "Primary+H":
			lines[i] = "Primary+Alt+F"
		}
	}
	if screenplay {
		for _, m := range markers {
			if strings.HasPrefix(name, "shortcuts-") {
				lines = append(lines, m[0]+" = "+m[1])
			} else {
				lines = append(lines, m[0])
			}
		}
	}
	slices.Sort(lines)
	return strings.Join(lines, "\n") + "\n"
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
// structureMoves: the structural moves (structure_window.go): Edit's
// Move Line Up/Down are View > Structure's Move Up/Down, with the rest of
// the moves (a screenplay's) beside them, and their keys.
// stripMoves: View > Timeline Strip after the Structure submenu (a
// screenplay's: timeline_strip.go).
func stripMoves(base, golden string) string {
	if !strings.HasPrefix(base, "menu-screenplay") {
		return golden
	}
	return strings.Replace(golden, "    Swap…\n    Move…\n", "    Swap…\n    Move…\n  Timeline Strip\n", 1)
}

// topRowMoves applies the top row's own (toprow.go): the palette's
// button inside the search field, View's Add to the Top Row… and Reset.
func topRowMoves(base, golden string) string {
	switch {
	case strings.HasPrefix(base, "header-"):
		const b = "button foreground_palette.svg tip=Command palette: find any command (Ctrl+K)"
		return strings.Replace(golden, "  "+b+"\n", "  box[*widget.Entry; "+b+"]\n", 1)
	case strings.HasPrefix(base, "menu-"):
		return strings.Replace(golden, "  Toolbar (checked)\n", "  Toolbar (checked)\n  Add to the Top Row…\n  Reset the Top Row\n", 1)
	}
	return golden
}

func structureMoves(base, golden string) string {
	md := strings.Contains(base, "markdown")
	type key struct{ keys, label string }
	keys := []key{{"Alt+Up", "Move Up"}, {"Alt+Down", "Move Down"}}
	if !md {
		keys = append(keys, key{"Alt+Left", "Promote"}, key{"Alt+Right", "Demote"},
			key{"Shift+Alt+Up", "Into the One Before"}, key{"Shift+Alt+Down", "Into the Next One"},
			key{"Shift+Alt+Left", "Pass the Last Speech On"}, key{"Shift+Alt+Right", "Take In the Next Speech"},
			key{"Alt+T", "Swap Speakers' Lines"}, key{"Alt+O", "Expand Selection"}, key{"Alt+I", "Shrink Selection"})
	}
	if runtime.GOOS == "darwin" { // Option+Left/Right move by word there
		for i, k := range keys {
			if strings.HasSuffix(k.keys, "Left") || strings.HasSuffix(k.keys, "Right") {
				keys[i].keys = "Control+" + k.keys
			}
		}
	}
	switch {
	case strings.HasPrefix(base, "menu-"):
		golden = strings.Replace(golden, "  ---\n  Move Line Up [Alt+Up]\n  Move Line Down [Alt+Down]\n", "", 1)
		sub := []string{"  Structure"}
		item := func(i int) string { return "    " + keys[i].label + " [" + keys[i].keys + "]" }
		if md {
			sub = append(sub, item(0), item(1))
		} else {
			sub = append(sub, item(0), item(1), item(2), item(3), "    ---", item(4), item(5), item(6), item(7),
				"    ---", item(8), "    Line to Action", "    Action into the Speech", "    ---", item(9), item(10),
				"    ---", "    Swap…", "    Move…")
		}
		golden = strings.Replace(golden, "  Unfold All\n", "  Unfold All\n"+strings.Join(sub, "\n")+"\n", 1)
	case strings.HasPrefix(base, "shortcuts-"), strings.HasPrefix(base, "fullscreen-"):
		var lines []string
		for _, l := range strings.Split(strings.TrimSuffix(golden, "\n"), "\n") {
			if !strings.Contains(l, "Alt+Up") && !strings.Contains(l, "Alt+Down") || strings.Contains(l, "Primary") {
				lines = append(lines, l)
			}
		}
		for _, k := range keys {
			if strings.HasPrefix(base, "shortcuts-") {
				lines = append(lines, k.keys+" = "+k.label)
			} else {
				lines = append(lines, k.keys)
			}
		}
		slices.Sort(lines)
		golden = strings.Join(lines, "\n") + "\n"
	}
	return golden
}

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
// Italic (the notes' commands after Footnote: markdown_notes.go).
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
  Go to Note or Reference
  Renumber Notes
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
	// File: Page Setup… before Print…, for both (pagesetup.go)
	golden = strings.Replace(golden, "  Print… [Primary+P]\n", "  Page Setup…\n  Print… [Primary+P]\n", 1)
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
	for _, l := range []string{"Outline", "Notes", "Cards", "Source", "Production", "Help", "Changes"} { // Cards a pane too (docs/MIDDLE.md)
		if it := takeItem(ms, "View", l); it != "" {
			panes = append(panes, it)
		}
		if l == "Notes" {
			panes = append(panes, "  Characters") // a pane of its own since (characters_pane.go)
		}
	}
	if strings.HasPrefix(name, "menu-screenplay") {
		panes = append(panes, "  Stripboard") // a middle tab beside the panes (docs/MIDDLE.md), Production's
	}
	panes = append(panes, "  Side Panes", "  Move Pane to Top", "  Move Pane to Bottom", "  Open Pane in the Middle", // docs/MIDDLE.md
		"  Middle Tab Back to the Side", "  Next Middle Tab [Primary+Next]", "  Previous Middle Tab [Primary+Prior]")
	for i := range ms {
		if ms[i].title == "Adaptation" { // its common case of Open in the Middle (docs/MIDDLE.md)
			for j, it := range ms[i].items {
				if ms[i].labelOf(it) == "Source Pane" {
					ms[i].items = slices.Insert(ms[i].items, j+1, "  Source in the Middle")
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

// middleMoves: the middle tabs (middle.go) add Next and Previous Middle
// Tab, Ctrl+PgDn and Ctrl+PgUp, to the shortcuts (and full screen's).
// Their menu items are dockMoves'.
func middleMoves(name, golden string) string {
	var add []string
	switch {
	case strings.HasPrefix(name, "shortcuts-"):
		add = []string{"Primary+Next = Next Middle Tab", "Primary+Prior = Previous Middle Tab"}
	case strings.HasPrefix(name, "fullscreen-"):
		add = []string{"Primary+Next", "Primary+Prior"}
	default:
		return golden
	}
	lines := append(strings.Split(strings.TrimSuffix(golden, "\n"), "\n"), add...)
	slices.Sort(lines)
	return strings.Join(lines, "\n") + "\n"
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

// passageMoves: Next and Previous only move, Adapt… writes, Go to the
// Passage's Scene (adapt_dialog.go): the Adaptation menu's passage group
// before its links, and the new keys.
func passageMoves(name, golden string) string {
	switch {
	case strings.HasPrefix(name, "menu-screenplay"):
		return strings.Replace(golden, `  Link to Source Passage
  Next Source Passage [Primary+Alt+J]
  Mark Source Passage as Cut
`, `  Next Source Passage [Primary+Alt+J]
  Previous Source Passage [Primary+Alt+K]
  Adapt Source Passage… [Primary+Alt+A]
  Mark Source Passage as Cut
  Go to the Passage's Scene
  ---
  Link to Source Passage
`, 1)
	case strings.HasPrefix(name, "shortcuts-screenplay"):
		golden = strings.Replace(golden, "Primary+Alt+J = Next Source Passage\n", "Primary+Alt+A = Adapt Source Passage…\nPrimary+Alt+J = Next Source Passage\n", 1)
		return strings.Replace(golden, "Primary+Alt+J = Next Source Passage\n", "Primary+Alt+J = Next Source Passage\nPrimary+Alt+K = Previous Source Passage\n", 1)
	case strings.HasPrefix(name, "fullscreen-screenplay"):
		return strings.Replace(golden, "Primary+Alt+J\n", "Primary+Alt+A\nPrimary+Alt+J\nPrimary+Alt+K\n", 1)
	}
	return golden
}

// fixLinesMoves: Fix Lines Like This… (fixup_dialog.go) in Edit after the
// finding, at the Adaptation menu's end and the editor's context menu's.
func fixLinesMoves(name, golden string) string {
	const item = "Fix Lines Like This…\n"
	switch {
	case strings.HasPrefix(name, "menu-screenplay"):
		golden = strings.Replace(golden, "  Find Previous [Primary+Shift+G]\n", "  Find Previous [Primary+Shift+G]\n  "+item, 1)
		return strings.Replace(golden, "  What Accolade Learned…\n", "  What Accolade Learned…\n  "+item, 1)
	case strings.HasPrefix(name, "context-screenplay"):
		return strings.TrimSuffix(golden, "\n") + "\n---\n" + item
	}
	return golden
}

// notesMoves: the notepad gone for the Notes pane (notes_pane.go): its
// toggle without a key and with a tip of its own; Ctrl+Shift+K is Add
// Note now, in Format with New Unsorted Note.
func notesMoves(name, golden string) string {
	const add = "  Add Note [Primary+Shift+K]\n  New Unsorted Note\n"
	switch {
	case strings.HasPrefix(name, "menu-"):
		golden = strings.Replace(golden, "  Notes [Primary+Shift+K]\n", "  Notes\n", 1)
		if strings.HasPrefix(name, "menu-screenplay") {
			return strings.Replace(golden, "# Adaptation\n", add+"# Adaptation\n", 1)
		}
		return strings.Replace(golden, "  Link… [Primary+L]\n", add+"  Link… [Primary+L]\n", 1)
	case strings.HasPrefix(name, "shortcuts-"):
		return strings.Replace(golden, "Primary+Shift+K = Notes\n", "Primary+Shift+K = Add Note\n", 1)
	case strings.HasPrefix(name, "header-"):
		tip := "Notes: the script's notes, cuts, to-dos and beats in its order, and its unsorted notes"
		golden = strings.Replace(golden, "tip=Notes (Ctrl+Shift+K)\n", "tip="+tip+"\n", 1)
		return strings.Replace(golden, "    menu:Notes icon=foreground_file-text.svg\n", "    menu:"+tip+" icon=foreground_file-text.svg\n", 1)
	}
	return golden
}

// frameMoves: Format › Storyboard Frame… at a screenplay's Format menu's
// end (storyboard.go).
func frameMoves(name, golden string) string {
	if strings.HasPrefix(name, "menu-screenplay") {
		return strings.Replace(golden, "# Adaptation\n", "  ---\n  Storyboard Frame…\n# Adaptation\n", 1)
	}
	return golden
}

// highlandMoves: Format's Include File…, Marker…, Next Marker and
// Previous Marker after Storyboard Frame… (Highland's directives,
// highland_directives.go), in a screenplay's menu bar.
func highlandMoves(name, golden string) string {
	if strings.HasPrefix(name, "menu-screenplay") {
		return strings.Replace(golden, "  Storyboard Frame…\n", "  Storyboard Frame…\n  Include File…\n  Marker…\n  Next Marker\n  Previous Marker\n", 1)
	}
	return golden
}

// tableReadMoves: View > Table Read after Preview PDF, Copy the Follow
// Link after Join a Shared Script… (tableread_tab.go).
func tableReadMoves(name, golden string) string {
	if strings.HasPrefix(name, "menu-screenplay") {
		golden = strings.Replace(golden, "  Join a Shared Script…\n", "  Join a Shared Script…\n  Copy the Follow Link\n", 1)
		golden = strings.Replace(golden, "  Preview PDF [Primary+Shift+P]\n", "  Preview PDF [Primary+Shift+P]\n  Table Read\n", 1)
	}
	return golden
}

// stagesMoves: File's Export Writer's Version… and Export Shooting
// Version…, Adaptation's Lock Source, Unlock Source and Relink From
// Here…, View's Stages… (stages.go), in a screenplay's menu bar.
func stagesMoves(name, golden string) string {
	if !strings.HasPrefix(name, "menu-screenplay") {
		return golden
	}
	golden = strings.Replace(golden, "  Export… [Primary+E]\n", "  Export… [Primary+E]\n  Export Writer's Version…\n  Export Shooting Version…\n", 1)
	golden = strings.Replace(golden, "  ---\n  What Accolade Learned…\n", "  ---\n  Lock Source\n  Unlock Source\n  Relink From Here…\n  ---\n  What Accolade Learned…\n", 1)
	return strings.Replace(golden, "  Statistics…\n", "  Statistics…\n  Stages…\n", 1)
}

// linkUpMoves: Suggest Links to the Source became Link Guess for This
// Scene… (Ctrl+Alt+G: Ctrl+Alt+L locks many Linux desktops) and Link Up the Script… (linkup.go).
func linkUpMoves(name, golden string) string {
	switch {
	case strings.HasPrefix(name, "menu-screenplay"):
		return strings.Replace(golden, "  Suggest Links to the Source\n", "  Link Guess for This Scene… [Primary+Alt+G]\n  Link Up the Script…\n", 1)
	case strings.HasPrefix(name, "shortcuts-screenplay"):
		return strings.Replace(golden, "Primary+Alt+J = Next Source Passage\n", "Primary+Alt+G = Link Guess for This Scene…\nPrimary+Alt+J = Next Source Passage\n", 1)
	case strings.HasPrefix(name, "fullscreen-screenplay"):
		return strings.Replace(golden, "Primary+Alt+J\n", "Primary+Alt+G\nPrimary+Alt+J\n", 1)
	}
	return golden
}

// bundleFolderMoves: File > Open Bundle Folder… after Open… (a bundle in
// its folder form, Name.accolade/: window.go).
func bundleFolderMoves(name, golden string) string {
	if strings.HasPrefix(name, "menu-") {
		return strings.Replace(golden, "  Open… [Primary+O]\n", "  Open… [Primary+O]\n  Open Bundle Folder…\n", 1)
	}
	return golden
}

// referencesMoves: View > References, a Markdown document's pane, before
// Help (references_pane.go).
func referencesMoves(name, golden string) string {
	if strings.HasPrefix(name, "menu-markdown") {
		return strings.Replace(golden, "  Help\n  Changes [Primary+Shift+H]\n", "  References\n  Help\n  Changes [Primary+Shift+H]\n", 1)
	}
	return golden
}

// newSceneMoves: Format › New Scene… before Storyboard Frame…
// (new_scene.go).
func newSceneMoves(name, golden string) string {
	if strings.HasPrefix(name, "menu-screenplay") {
		return strings.Replace(golden, "  ---\n  Storyboard Frame…\n", "  ---\n  New Scene…\n  Storyboard Frame…\n", 1)
	}
	return golden
}

// numberScenesMoves: Format > Number Scenes in Headings after New Scene…
// (scene_numbers.go).
func numberScenesMoves(name, golden string) string {
	if strings.HasPrefix(name, "menu-screenplay") {
		return strings.Replace(golden, "  New Scene…\n", "  New Scene…\n  Number Scenes in Headings\n", 1)
	}
	return golden
}

// mergeMoves: File > Merge Changes… before Import as Markdown… (merge.go).
func mergeMoves(name, golden string) string {
	if strings.HasPrefix(name, "menu-") {
		return strings.Replace(golden, "  Import as Markdown…\n", "  Merge Changes…\n  Import as Markdown…\n", 1)
	}
	return golden
}

// templateMoves: File > New from Structure before Merge Changes…, and a
// screenplay's Format > Insert Structure after New Scene…, each with the
// structures (structure_templates.go).
func templateMoves(name, golden string) string {
	if !strings.HasPrefix(name, "menu-") {
		return golden
	}
	sub := func(label string) string {
		s := "  " + label + "\n"
		for _, t := range structureTemplates {
			s += "    " + t.name + "\n"
		}
		return s
	}
	golden = strings.Replace(golden, "  Merge Changes…\n", sub("New from Structure")+"  Merge Changes…\n", 1)
	if strings.HasPrefix(name, "menu-screenplay") {
		golden = strings.Replace(golden, "  New Scene…\n", "  New Scene…\n"+sub("Insert Structure"), 1)
		// the structure out and in (outline_io.go, treatment.go)
		golden = strings.Replace(golden, "  Export Clean Fountain…\n", "  Export Clean Fountain…\n  Export Outline (OPML) — for Workflowy, OmniOutliner…\n  Export Treatment (PDF, Word)…\n", 1)
		golden = strings.Replace(golden, "  Paste\n  Select All\n", "  Paste\n  Paste as Structure\n  Select All\n", 1)
	}
	return golden
}

// sampleMoves: File > Open Sample after Open Recent (sample.go).
func sampleMoves(name, golden string) string {
	if !strings.HasPrefix(name, "menu-") {
		return golden
	}
	golden = strings.Replace(golden, "  Open Recent\n    No recent scripts (disabled) (no action)\n", "  Open Recent\n    No recent scripts (disabled) (no action)\n  Open Sample\n", 1)
	// its submenu: the sample, then each language's (sample.go)
	sub := "  Open Sample\n    A Chameleon (English)\n    ---\n"
	for _, lang := range sampleLanguages() {
		sub += "    " + languageName(lang) + ": " + langSamples()[lang].title + "\n"
	}
	return strings.Replace(golden, "  Open Sample\n", sub, 1)
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
		takeItem(ms, "View", "Stripboard") // Production's middle tab
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
