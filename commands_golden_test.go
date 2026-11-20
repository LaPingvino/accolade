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
	return onThisSystem(string(b))
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
			if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	for name, s := range got {
		want := adaptationMoves(name, readCommandsGolden(t, name))
		if s != want {
			t.Errorf("%s differs from the golden (with the Adaptation moves):\n%s", name, lineDiff(want, s))
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
// group's, left behind). The shortcuts, header bar and context menu are
// as they were.
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
	return strings.Replace(golden, "# Production\n", adaptationMenu+"# Production\n", 1)
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
