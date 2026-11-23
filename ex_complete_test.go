package main

import (
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// names are the choices' texts.
func names(cs []exChoice) []string {
	var out []string
	for _, c := range cs {
		out = append(out, strings.TrimSpace(c.text))
	}
	return out
}

// Every name is its command's own: none hides another or a line
// command's abbreviation, and every command of the registry is reached.
func TestExTable(t *testing.T) {
	tab := exCommands()
	ids := map[string]bool{}
	for _, c := range tab.cmds {
		if got, _ := tab.lookup(c.name); got != c {
			t.Errorf(":%s finds %v", c.name, got)
		}
		if c.short != "" {
			if got, _ := tab.lookup(c.short); got != c {
				t.Errorf(":%s (short) finds %v, not %s", c.short, got, c.name)
			}
		}
		if c.desc == "" || c.group == "" {
			t.Errorf(":%s has no line or group", c.name)
		}
		ids[c.id] = true
	}
	for _, rc := range commands().commands {
		if ids[rc.id] {
			continue
		}
		by, ok := exByCommand(rc)
		if !ok {
			t.Errorf("%s (%s) has no ex command", rc.id, rc.label)
			continue
		}
		name, _, _ := strings.Cut(by, " ")
		if c, _ := tab.lookup(name); c == nil || c.run == nil {
			t.Errorf("%s: :%s is no window command", rc.id, by)
		}
	}
	// vim's abbreviations
	for typed, want := range map[string]string{"s": "substitute", "su": "substitute", "d": "delete", "de": "delete",
		"co": "copy", "t": "t", "m": "move", "mo": "move", "u": "undo", "w": "write", "x": "xit", "exi": "exit",
		"q": "quit", "e": "edit", "h": "help", "el": "element", "sc": "scene", "g": "global", "v": "vglobal",
		"move-scene-up": "move-scene-up", "preview": "preview", "outline": "outline", "cards": "cards",
		"source": "source", "fold": "fold", "export": "export", "next-source": "next-source-passage", "pre": ""} {
		got, _ := tab.lookup(typed)
		if want == "" {
			if got != nil {
				t.Errorf(":%s finds %s, none wanted", typed, got.name)
			}
			continue
		}
		if got == nil || got.name != want {
			t.Errorf(":%s finds %v, want %s", typed, got, want)
		}
	}
}

// The line parser reads hyphened names, and m-2 stays :m with -2.
func TestExNames(t *testing.T) {
	lines := []string{"a", "b", "c", "d"}
	for cmd, want := range map[string]string{"m-2": "a|c|b|d", "2,3m0": "b|c|a|d", "m+1": "a|b|d|c", "1t.": "a|b|c|a|d", "m-9": "invalid address"} {
		var st exState
		res, err := st.run(cmd, lines, 2, [2]int{-1, -1})
		if err != nil && err.Error() == want {
			continue
		}
		if err != nil || strings.Join(res.lines, "|") != want {
			t.Errorf(":%s %q %v, want %q", cmd, strings.Join(res.lines, "|"), err, want)
		}
	}
	var st exState
	_, err := st.run("move-scene-up", lines, 0, [2]int{-1, -1})
	var nc *exNotCommand
	if !errors.As(err, &nc) || nc.name != "move-scene-up" || !errors.Is(err, errNotCommand) {
		t.Errorf("move-scene-up: %v", err)
	}
	_, err = st.run("5,6scene INT. BARN", lines, 0, [2]int{-1, -1})
	if !errors.As(err, &nc) || nc.name != "scene" || strings.TrimSpace(nc.rest) != "INT. BARN" {
		t.Errorf("scene: %v", err)
	}
}

func exTestCtx(t *testing.T) exCtx {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "barn.fountain"), nil, 0o644)
	os.WriteFile(filepath.Join(dir, "field.fountain"), nil, 0o644)
	os.Mkdir(filepath.Join(dir, "drafts"), 0o755)
	return exCtx{
		headings: outlineOf("# Act One\n\nINT. BARN - DAY\n\nRain.\n\nEXT. FIELD - NIGHT\n\nWind.\n", nil),
		dir:      dir, files: true,
		items: func(id string) []string {
			if id == "set.theme" {
				return []string{"System", "Light", "Dark"}
			}
			return nil
		},
	}
}

// What the line completes, for typical beginnings.
func TestExComplete(t *testing.T) {
	tab := exCommands()
	ctx := exTestCtx(t)
	got := func(line string) []string { return names(tab.complete(ctx, line).choices) }
	first := func(line string) string {
		if n := got(line); len(n) > 0 {
			return n[0]
		}
		return ""
	}
	if n := got(""); len(n) < 8 || n[0] != "help" || !slices.Contains(n, "scene") || !slices.Contains(n, "move-scene-up") {
		t.Errorf("an empty line: %v", n)
	}
	for line, want := range map[string]string{"s": "substitute", "d": "delete", "w": "write", "q": "quit", "pre": "preview",
		"out": "outline", "car": "cards", "sc": "scene", "exp": "export", "fo": "fold", "h": "help", "nsp": "next-source-passage",
		"'<,'>": "substitute", "%s": "substitute", "/barn/d": "delete"} {
		if f := first(line); f != want {
			t.Errorf("%q: first %q, want %q (%v)", line, f, want, got(line)[:min(5, len(got(line)))])
		}
	}
	// :mo lists every move, the prefix's first, the jumps after them
	mo := got("mo")
	for _, want := range []string{"move", "move-scene-up", "move-scene-down", "move-line-up", "move-line-down", "move-pane-top",
		"promote-section", "demote-section", "next-heading", "previous-heading", "go-parent-section", "scene"} {
		if !slices.Contains(mo, want) {
			t.Errorf(":mo has no %s: %v", want, mo)
		}
	}
	if slices.Index(mo, "move-scene-up") > slices.Index(mo, "next-heading") {
		t.Errorf(":mo: the moves after the jumps: %v", mo)
	}
	// after a range only what takes one
	for _, n := range got("'<,'>") {
		if c, _ := tab.lookup(n); c == nil || !c.ranged {
			t.Errorf("after a range: %s", n)
		}
	}
	// arguments
	for line, want := range map[string][]string{
		"scene ":        {"Act One", "1 INT. BARN - DAY", "2 EXT. FIELD - NIGHT"},
		"scene ext":     {"2 EXT. FIELD - NIGHT"},
		"scene 2":       {"2"},
		"sc fi":         {"2 EXT. FIELD - NIGHT"},
		"element ac":    {"action"},
		"el sc":         {"scene-heading"},
		"export pd":     {"pdf"},
		"w b":           {"barn.fountain"},
		"e dr":          {"drafts/"},
		"theme d":       {"Dark"},
		"help scen":     {"scene"},
		"help fount":    {"fountain"},
		"heading ":      {"0", "1", "2", "3", "4", "5", "6"},
		"s/barn/stable": nil,
	} {
		g := got(line)
		if want == nil {
			if len(g) > 0 {
				t.Errorf("%q: %v, want none", line, g)
			}
			continue
		}
		if len(g) < len(want) || !slices.Equal(g[:len(want)], want) {
			t.Errorf("%q: %v, want %v first", line, g, want)
		}
	}
	if c := tab.complete(ctx, "scene "); !c.arg || c.start != len("scene ") {
		t.Errorf("scene's argument: %+v", c)
	}
	// a command wanting its argument: Enter lists it
	if !exWantsMore("scene") || !exWantsMore("theme ") || exWantsMore("scene 2") || exWantsMore("preview") || exWantsMore("d") {
		t.Error("exWantsMore")
	}
}

// exTestWindow is a window with vim keys and a script of two scenes.
func exTestWindow(t *testing.T) (*MainWindow, fyne.Canvas) {
	w, _ := newDialogTestWindow(t)
	w.settings.SetString("editor-keys", "vim")
	t.Cleanup(func() { w.settings.SetString("editor-keys", "standard") })
	w.applyEditorSettings()
	w.textEditor.SetText("INT. BARN - DAY\n\nRain on the barn.\n\nEXT. FIELD - NIGHT\n\nWind.\n")
	c := w.fyneWindow.Canvas()
	c.Focus(w.textEditor)
	w.textEditor.TypedRune(':')
	if c.Focused() != w.exEntry {
		t.Fatalf("':' focused %T", c.Focused())
	}
	return w, c
}

func key(l *exLine, k fyne.KeyName) { l.TypedKey(&fyne.KeyEvent{Name: k}) }

// Tab completes and cycles, Shift+Tab goes back, the arrows choose
// without writing, Up on an empty line is the history; the line keeps
// the keyboard all along.
func TestExLineKeys(t *testing.T) {
	w, c := exTestWindow(t)
	l := w.exEntry
	if !l.AcceptsTab() {
		t.Fatal("the line gives Tab away")
	}
	if !w.exPop.layer.Visible() || len(l.comp.choices) == 0 || l.comp.choices[0].cmd.name != "help" {
		t.Fatal("an empty line lists no commands")
	}
	test.Type(l, "mo")
	if c.Focused() != l || !w.exPop.layer.Visible() {
		t.Fatal("typing lost the line or the list")
	}
	n := len(l.comp.choices)
	key(l, fyne.KeyTab)
	if l.Text != "move " {
		t.Errorf("Tab: %q", l.Text)
	}
	key(l, fyne.KeyTab)
	second := l.Text
	if second == "move " || !strings.HasPrefix(second, "mo") || len(l.comp.choices) != n {
		t.Errorf("Tab again: %q (%d choices, were %d)", second, len(l.comp.choices), n)
	}
	l.KeyDown(&fyne.KeyEvent{Name: "LeftShift"})
	key(l, fyne.KeyTab)
	l.KeyUp(&fyne.KeyEvent{Name: "LeftShift"})
	if l.Text != "move " {
		t.Errorf("Shift+Tab: %q", l.Text)
	}
	// the arrows choose; Enter runs the chosen
	l.set("")
	test.Type(l, "move-scene-d")
	if len(l.comp.choices) != 1 {
		t.Fatalf("move-scene-d: %v", names(l.comp.choices))
	}
	key(l, fyne.KeyDown)
	if l.Text != "move-scene-d" || l.chosen != 0 {
		t.Errorf("Down wrote %q", l.Text)
	}
	key(l, fyne.KeyReturn)
	if got := w.textEditor.Text(); got != "EXT. FIELD - NIGHT\n\nWind.\n\nINT. BARN - DAY\n\nRain on the barn.\n" {
		t.Errorf("move-scene-down: %q", got)
	}
	if c.Focused() != w.textEditor || w.exPop.layer.Visible() {
		t.Error("the line or its list stayed")
	}
	// Enter runs what is typed: :d is delete, whatever is listed
	w.textEditor.TypedRune(':')
	test.Type(l, "1d")
	key(l, fyne.KeyReturn)
	if got := w.textEditor.Text(); !strings.HasPrefix(got, "\nWind.") {
		t.Errorf(":1d: %q", got)
	}
	// Up on an empty line: the history, the list put away
	w.textEditor.TypedRune(':')
	key(l, fyne.KeyUp)
	if l.Text != "1d" || w.exPop.layer.Visible() {
		t.Errorf("Up: %q", l.Text)
	}
	key(l, fyne.KeyUp)
	if l.Text != "move-scene-down" { // what ran
		t.Errorf("Up again: %q", l.Text)
	}
	key(l, fyne.KeyDown)
	key(l, fyne.KeyDown)
	if l.Text != "" {
		t.Errorf("Down to the end: %q", l.Text)
	}
	key(l, fyne.KeyDown) // into the list
	if l.chosen != 0 || !w.exPop.layer.Visible() {
		t.Errorf("Down on an empty line: chosen %d", l.chosen)
	}
	// a recalled command: Tab and Enter leave it as it is
	key(l, fyne.KeyEscape)
	w.textEditor.TypedRune(':')
	key(l, fyne.KeyUp)
	if l.Text != "1d" {
		t.Fatalf("Up back: %q", l.Text)
	}
	key(l, fyne.KeyTab)
	key(l, fyne.KeyPageDown)
	if l.Text != "1d" || len(l.comp.choices) != 0 {
		t.Errorf("Tab on a recalled line: %q", l.Text)
	}
	key(l, fyne.KeyEscape)
	// a name that names nothing: Enter takes the list's first
	if !exUnresolved("otln") || exUnresolved("d") || exUnresolved("outline") || exUnresolved("scene 2") {
		t.Error("exUnresolved")
	}
	w.textEditor.TypedRune(':')
	test.Type(l, "otln")
	if f := l.comp.choices[0].cmd.name; f != "outline" {
		t.Fatalf("otln: first %s", f)
	}
	key(l, fyne.KeyReturn)
	if !w.paneInSight("outline") {
		t.Error(":otln did not run :outline")
	}
}

// Two scenes with one heading: each is reached from the list; a tap
// after the arrows runs the row tapped.
func TestExLineScenes(t *testing.T) {
	w, c := exTestWindow(t)
	w.textEditor.SetText("INT. KITCHEN - DAY\n\nTea.\n\nEXT. GARDEN - DAY\n\nRain.\n\nINT. KITCHEN - DAY\n\nMore tea.\n")
	c.Focus(w.textEditor)
	w.textEditor.TypedRune(':')
	l := w.exEntry
	test.Type(l, "scene kitchen")
	if got := names(l.comp.choices); !slices.Equal(got, []string{"1 INT. KITCHEN - DAY", "3 INT. KITCHEN - DAY"}) {
		t.Fatalf("scene kitchen: %v", got)
	}
	key(l, fyne.KeyDown)                        // the first chosen…
	w.exPop.shown[1].Tapped(&fyne.PointEvent{}) // …the second tapped
	if line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset()); line != 8 {
		t.Errorf("the second kitchen: line %d", line)
	}
}

// :scene with Enter alone lists the scenes; a tap on a row completes it
// and runs it, the keyboard staying in the line meanwhile.
func TestExLineArguments(t *testing.T) {
	w, c := exTestWindow(t)
	l := w.exEntry
	test.Type(l, "scene")
	key(l, fyne.KeyReturn)
	if c.Focused() != l || l.Text != "scene " || !l.comp.arg || len(l.comp.choices) != 2 {
		t.Fatalf("Enter on :scene: %q %v", l.Text, names(l.comp.choices))
	}
	w.exPop.shown[1].Tapped(&fyne.PointEvent{})
	line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
	if line != 4 || c.Focused() != w.textEditor {
		t.Errorf("the tapped scene: line %d", line)
	}
	// a tap on a command wanting more: completed, the line keeps going
	w.textEditor.TypedRune(':')
	test.Type(l, "el")
	w.exPop.shown[0].Tapped(&fyne.PointEvent{})
	if l.Text != "element " || c.Focused() != l || !w.exPop.layer.Visible() {
		t.Errorf("tapped :element: %q", l.Text)
	}
	test.Type(l, "tr")
	key(l, fyne.KeyTab)
	key(l, fyne.KeyReturn)
	if got := w.textEditor.Text(); !strings.Contains(got, "> EXT. FIELD - NIGHT") && !strings.Contains(got, "EXT. FIELD - NIGHT:") {
		t.Errorf(":element transition: %q", got)
	}
	// a registry command by its name, and :help
	w.textEditor.TypedRune(':')
	test.Type(l, "outline")
	key(l, fyne.KeyReturn)
	if !w.paneInSight("outline") {
		t.Error(":outline did not show the outline")
	}
	w.textEditor.TypedRune(':')
	test.Type(l, "h scene")
	key(l, fyne.KeyReturn)
	if !w.paneInSight("help") || w.helpPane().page.Slug != "ex" {
		t.Errorf(":h scene: %q", w.helpPane().page.Slug)
	}
	w.runEx("pr")
	if s := w.statusLabel.Text; !strings.Contains(s, "more than one") {
		t.Errorf(":pr: %q", s)
	}
	w.runEx("theme dark")
	if got := w.settings.GetString("theme"); !strings.EqualFold(got, "dark") {
		t.Errorf(":theme dark: the theme is %q", got)
	}
	// a prompt's line does not complete
	w.showPrompt("select:", func(string) {})
	test.Type(l, "s")
	if w.exPop.layer.Visible() {
		t.Error("a prompt completes")
	}
	key(l, fyne.KeyEscape)
}

// docs/help/ex.md is what the table makes, and names every command.
//
//	UPDATE_HELP=1 go test -run TestExHelp .
func TestExHelp(t *testing.T) {
	page := exHelpPage()
	for _, c := range exCommands().cmds {
		if !strings.Contains(page, "**:"+c.name+"**") && !strings.Contains(page, "["+strings.TrimPrefix(c.name, c.short)+"]**") {
			t.Errorf("the page has no :%s", c.name)
		}
	}
	path := filepath.Join("docs", "help", "ex.md")
	if os.Getenv("UPDATE_HELP") != "" {
		if err := os.WriteFile(path, []byte(page), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != page {
		t.Errorf("%s is not what the table makes: UPDATE_HELP=1 go test -run TestExHelp .", path)
	}
}

// TestExScreenshots renders the list, desktop and phone (SHOT_DIR).
func TestExScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(dir, 0o755)
	snap := func(w *MainWindow, name string) {
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		png.Encode(f, w.fyneWindow.Canvas().Capture())
	}
	w, _ := exTestWindow(t)
	snap(w, "ex-1-empty")
	test.Type(w.exEntry, "mo")
	key(w.exEntry, fyne.KeyTab)
	key(w.exEntry, fyne.KeyTab)
	snap(w, "ex-2-mo")
	w.exEntry.set("")
	test.Type(w.exEntry, "scene ")
	snap(w, "ex-3-scene")
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.exEntry.set("")
	test.Type(w.exEntry, "pre")
	snap(w, "ex-4-phone")
	w.runEx("theme light")
	w.textEditor.TypedRune(':')
	test.Type(w.exEntry, "sc")
	snap(w, "ex-5-phone-light")
}
