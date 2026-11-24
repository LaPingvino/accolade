package main

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// TestMain: the tests start as Production, the whole app, as they were
// written before the role (role.go); the role's own tests choose theirs.
func TestMain(m *testing.M) {
	defaultRole = roleProduction
	defaultLook = "menus"
	// what Accolade keeps in its data folder (kept sources, the sources
	// links were looked at against: relink.go) never in the real one
	data, err := os.MkdirTemp("", "accolade-data-")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_DATA_HOME", data)
	// nor the settings: a test that saves them (a ribbon toggle, Preferences)
	// with no folder of its own writes here, never the user's settings.json
	realHome, _ = os.UserHomeDir()
	home, err := os.MkdirTemp("", "accolade-home-")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	code := m.Run()
	os.RemoveAll(data)
	os.RemoveAll(home)
	os.Exit(code)
}

// realHome is the user's home before TestMain moved it.
var realHome string

// lockedScript is a script locked for production and issued Blue.
const lockedScript = "INT. BARN - DAY #1#\n\nRain.\n\n/* Production\nscene numbers: locked, 10A\nrevision: Blue, 2026-11-20\n*/\n"

// Writer puts production away everywhere the registry reaches: the menu
// bar, the side panel, the header, the context menu, the keys, the
// ribbon; the script stays as it is (still locked).
func TestRoleWriterPutsProductionAway(t *testing.T) {
	w := newTestWindow(t, lockedScript)
	workAsFor(t, w, roleWriter)
	if !w.writerView() || !w.cmdEnv().writer {
		t.Fatal("not working as a writer")
	}
	m := w.fyneWindow.MainMenu()
	if strings.Contains(strings.Join(menuTitles(m), " "), "Production") {
		t.Errorf("Writer has a Production menu: %v", menuTitles(m))
	}
	for _, l := range labelsOf(m, "View") {
		if l == "Production" {
			t.Error("View > Production (the pane) in Writer")
		}
	}
	w.togglePane("notes")
	w.showPane("production", false)
	if tabs := dockTitles(w, dockBottom); tabs != "Notes" {
		t.Errorf("the side panel's tabs in Writer: %q", tabs)
	}
	for _, it := range w.headerBar.top.items {
		if it.obj == w.headerBar.button("view.production") && !it.off {
			t.Error("the header's Production button in Writer")
		}
	}
	if w.previewKey(fyne.KeyB, fyne.KeyModifierShortcutDefault|fyne.KeyModifierShift) {
		t.Error("a production key ran in Writer")
	}
	if !sceneLockOf(w.textEditor.Text()).locked {
		t.Error("the script's lock went")
	}
	r := ribbonOnFor(t, w)
	for _, tb := range r.shownTabs() {
		if tb.title == "Production" {
			t.Error("the ribbon's Production tab in Writer")
		}
	}
	r.selectTab("View")
	if buttonOf(r, "view.production") != nil {
		t.Error("the ribbon's Production Pane button in Writer")
	}
	ribbonGolden(t, "screenplay-writer-view", dumpRibbon(r))

	// and back (in Menus mode: the menu bar there)
	w.settings.SetString(lookPref, "menus")
	w.applyEditorSettings()
	w.workAs(roleProduction)
	if !strings.Contains(strings.Join(menuTitles(w.fyneWindow.MainMenu()), " "), "Production") {
		t.Error("Production: no Production menu")
	}
	w.showPane("production", false)
	if tabs := dockTitles(w, dockBottom); tabs != "Notes Production" {
		t.Errorf("Production: the side panel's tabs %q", tabs)
	}
}

// Switching to Writer with the Production pane open puts it away: no view
// left without its toggle.
func TestRoleWriterClosesProductionPane(t *testing.T) {
	w := newTestWindow(t, lockedScript)
	w.togglePane("production")
	if !w.productionPane().shown() {
		t.Fatal("the pane did not open")
	}
	workAsFor(t, w, roleWriter)
	if w.productionPane().shown() || w.paneShown("production") {
		t.Error("the Production pane stays open in Writer")
	}
}

// A locked script says so, quietly, once production is put away; an
// unlocked one says nothing of it.
func TestRoleLockedStatus(t *testing.T) {
	w := newTestWindow(t, lockedScript)
	workAsFor(t, w, roleWriter)
	want := "This script is locked for production (Blue) — switch to Production to change that"
	if got := w.lockedForWriter(); got != want {
		t.Errorf("status %q, want %q", got, want)
	}
	if w.statusLabel.Text != want {
		t.Errorf("the status line says %q", w.statusLabel.Text)
	}
	w.textEditor.SetText("INT. BARN - DAY #1#\n\n/* Production\nscene numbers: locked, 10A\n*/\n")
	if got := w.lockedForWriter(); got != "This script is locked for production — switch to Production to change that" {
		t.Errorf("locked, never issued: %q", got)
	}
	w.textEditor.SetText("INT. BARN - DAY\n\nRain.\n")
	if got := w.lockedForWriter(); got != "" {
		t.Errorf("unlocked: %q", got)
	}
	w.workAs(roleProduction)
	w.textEditor.SetText(lockedScript)
	if got := w.lockedForWriter(); got != "" {
		t.Errorf("Production: %q", got)
	}
}

// Opening a locked script as a writer says so on the status line.
func TestRoleLockedStatusOnOpen(t *testing.T) {
	w := newTestWindow(t, "")
	workAsFor(t, w, roleWriter)
	path := filepath.Join(t.TempDir(), "locked.fountain")
	if err := os.WriteFile(path, []byte(lockedScript), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := w.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(w.statusLabel.Text, "This script is locked for production (Blue)") {
		t.Errorf("status on open: %q", w.statusLabel.Text)
	}
}

// A script can keep its own role: the preferences' change does not touch
// it, it follows a rename, and an unsaved one's is kept when it is saved.
func TestRoleOwnScript(t *testing.T) {
	w := newTestWindow(t, "")
	workAsFor(t, w, roleProduction)
	dir := t.TempDir()
	path := filepath.Join(dir, "story.fountain")
	os.WriteFile(path, []byte(lockedScript), 0o644)
	if err := w.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	w.workAs(roleWriter) // follows the preferences: they change
	if w.settings.GetString(rolePref) != roleWriter || !w.writerView() {
		t.Fatal("View > Working As did not set the preferences' role")
	}
	w.workAs(roleProduction)
	w.toggleOwnRole() // this script: Production, whatever they say
	w.settings.SetString(rolePref, roleWriter)
	w.roleChanged(false)
	if w.writerView() {
		t.Error("the script's own role did not win over the preferences'")
	}
	w.workAs(roleWriter) // with its own role: only it changes
	if w.settings.GetString(rolePref) != roleWriter || w.ownRole() != roleWriter {
		t.Errorf("own role %q, preferences %q", w.ownRole(), w.settings.GetString(rolePref))
	}
	w.workAs(roleProduction)
	newPath := filepath.Join(dir, "film.fountain")
	os.Rename(path, newPath)
	w.typeAfterRename(path, "story.fountain", newPath, "film.fountain")
	w.currentFile = newPath
	if w.keptRole(newPath) != roleProduction || w.keptRole(path) != "" {
		t.Errorf("after a rename: new %q, old %q", w.keptRole(newPath), w.keptRole(path))
	}
	w.toggleOwnRole() // follows the preferences again
	if w.ownRole() != "" || !w.writerView() {
		t.Error("Only for This Script off: still its own")
	}

	// an unsaved script's, kept when it is saved
	w.createNewFile(true)
	w.setOwnRole(roleProduction)
	saved := filepath.Join(dir, "new.fountain")
	if err := w.saveToFile(saved); err != nil {
		t.Fatal(err)
	}
	w.typeAfterRename("", "untitled.fountain", saved, "new.fountain")
	if w.keptRole(saved) != roleProduction {
		t.Errorf("the unsaved script's role was not kept: %q", w.keptRole(saved))
	}
	w.createNewFile(true)
	if w.ownRole() != "" {
		t.Error("a new script took the last one's role")
	}
}

// The palette: Production's commands found by name in Writer, marked, and
// run they switch the script to Production; not listed before typing.
func TestRolePalette(t *testing.T) {
	w := newTestWindow(t, "INT. BARN - DAY\n\nRain.\n")
	workAsFor(t, w, roleWriter)
	entries := w.paletteEntries(w.cmdEnv())
	for _, e := range paletteMatches(entries, "") {
		if e.away {
			t.Errorf("listed before typing: %s", e.label)
		}
	}
	ms := paletteMatches(entries, "lock scene numbers")
	if len(ms) == 0 || ms[0].label != "Lock Scene Numbers (Production)" || !ms[0].away {
		t.Fatalf("Lock Scene Numbers not found as Production's: %+v", ms)
	}
	w.showPalette()
	w.palette.shown, w.palette.chosen = ms, 0
	w.palette.run()
	if w.writerView() || w.ownRole() != roleProduction {
		t.Error("running it did not switch this script to Production")
	}
	if w.settings.GetString(rolePref) != roleWriter {
		t.Error("the preferences' role changed too")
	}
	if !sceneLockOf(w.textEditor.Text()).locked {
		t.Error("Lock Scene Numbers did not run")
	}
	if !strings.Contains(w.statusLabel.Text, "Production") {
		t.Errorf("the switch not said: %q", w.statusLabel.Text)
	}
}

// Users from before the role become what their settings show.
func TestMigratedRole(t *testing.T) {
	none := func(string) ([]byte, error) { return nil, os.ErrNotExist }
	cases := []struct {
		name string
		data map[string]any
		read func(string) ([]byte, error)
		want string
	}{
		{"defaults only", map[string]any{"scene-number-style": "10A", "revision-marks": "last", "theme": "dark"}, none, roleWriter},
		{"a production folder", map[string]any{"production-files": []any{"/a.fountain:/a.production"}}, none, roleProduction},
		{"lettering changed", map[string]any{"scene-number-style": "A11"}, none, roleProduction},
		{"continueds on", map[string]any{"export-continueds": true}, none, roleProduction},
		{"a recent script locked", map[string]any{"recent-files": []any{"/x.fountain"}},
			func(string) ([]byte, error) { return []byte(lockedScript), nil }, roleProduction},
		{"recent scripts unlocked", map[string]any{"recent-files": []any{"/x.fountain"}},
			func(string) ([]byte, error) { return []byte("INT. A - DAY\n"), nil }, roleWriter},
	}
	for _, c := range cases {
		if got := migratedRole(c.data, c.read); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}

// A new user is a writer; a settings file from before the role gets one
// from what it holds, kept.
func TestRoleOfNewAndOldUsers(t *testing.T) {
	old := defaultRole
	defaultRole = roleWriter
	t.Cleanup(func() { defaultRole = old })
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if s := NewSettings(); s.GetString(rolePref) != roleWriter {
		t.Errorf("a new user: %q", s.GetString(rolePref))
	}
	os.MkdirAll(filepath.Join(dir, "accolade"), 0o755)
	os.WriteFile(filepath.Join(dir, "accolade", "settings.json"), []byte(`{"scene-number-style": "A11"}`), 0o644)
	s := NewSettings()
	if s.GetString(rolePref) != roleProduction || !s.dirty {
		t.Errorf("a user from before the role, lettering changed: %q (kept: %v)", s.GetString(rolePref), s.dirty)
	}
	os.WriteFile(filepath.Join(dir, "accolade", "settings.json"), []byte(`{"role": "writer", "scene-number-style": "A11"}`), 0o644)
	if s := NewSettings(); s.GetString(rolePref) != roleWriter {
		t.Error("a chosen role was not kept")
	}
}

// Preferences: the Production card only for Production; Reset to Defaults
// as a writer keeps production's settings (out of sight) as they are.
func TestRolePreferences(t *testing.T) {
	w, app := newDialogTestWindow(t)
	workAsFor(t, w, roleWriter)
	w.settings.SetString(sceneNumberStylePref, "A11")
	t.Cleanup(func() { w.settings.SetString(sceneNumberStylePref, "10A") })
	pd := NewPreferencesDialog(app, w.fyneWindow)
	if pd.roleSelect.Selected != "Writer" || pd.productionCard.Visible() {
		t.Errorf("Writer: role %q, Production card shown %v", pd.roleSelect.Selected, pd.productionCard.Visible())
	}
	pd.roleSelect.SetSelected("Production")
	if !pd.productionCard.Visible() {
		t.Error("Production: no Production card")
	}
	pd.roleSelect.SetSelected("Writer")
	defaults := &Settings{data: map[string]interface{}{}}
	defaults.loadDefaults()
	for _, k := range productionPrefs {
		defaults.data[k] = pd.settings.get(k)
	}
	pd.loadFrom(defaults) // what the confirmed reset does as a writer
	pd.saveSettings()
	if w.settings.GetString(sceneNumberStylePref) != "A11" {
		t.Error("Reset as a writer changed production's lettering")
	}
	pd.roleSelect.SetSelected("Production")
	pd.saveSettings()
	if w.settings.GetString(rolePref) != roleProduction {
		t.Error("the role chosen was not saved")
	}
	pd.close()
}

// The ribbon replaces the menu bar (no double bar); its ☰ has the same
// menus, as the role has them.
func TestRibbonMenusButton(t *testing.T) {
	w := newTestWindow(t, "INT. BARN - DAY\n\nRain.\n")
	r := ribbonOnFor(t, w)
	if !nativeMenus && w.fyneWindow.MainMenu() != nil {
		t.Error("Ribbon mode keeps the menu bar: a double bar")
	}
	if r.menus == nil || r.menus.AccessibilityLabel() == "" {
		t.Fatal("no ☰ button, or no name for it")
	}
	same := func() {
		t.Helper()
		var got, want strings.Builder
		for _, it := range w.menusMenu().Items {
			got.WriteString("# " + it.Label + "\n")
			dumpMenuItems(&got, it.ChildMenu.Items, 1)
		}
		bar := w.menuBar(w.cmdEnv())
		bar.Items[0].Items = append(bar.Items[0].Items, fyne.NewMenuItemSeparator(), fyne.NewMenuItem("Quit", func() {}))
		for _, m := range bar.Items {
			want.WriteString("# " + m.Label + "\n")
			dumpMenuItems(&want, m.Items, 1)
		}
		if got.String() != want.String() {
			t.Errorf("☰ differs from the menu bar:\n%s", lineDiff(want.String(), got.String()))
		}
	}
	same()
	workAsFor(t, w, roleWriter)
	same()
	for _, it := range w.menusMenu().Items {
		if it.Label == "Production" {
			t.Error("☰ has Production's menu in Writer")
		}
	}
	// the ☰ shows a menu by it, and the keyboard reaches it: a key tip
	w.showMenusAt(r.menus)
	if w.fyneWindow.Canvas().Overlays().Top() == nil {
		t.Error("☰ opened no menu")
	}
	w.fyneWindow.Canvas().Overlays().Remove(w.fyneWindow.Canvas().Overlays().Top())
	w.keyPreview(fyne.KeyF10, 0)
	if r.menus.tip == "" {
		t.Error("☰ has no key tip")
	}
	r.leaveKeyTips(true)
	// Menus mode: the menu bar is back
	w.settings.SetString(lookPref, "menus")
	w.applyEditorSettings()
	if w.fyneWindow.MainMenu() == nil {
		t.Error("Menus mode: no menu bar")
	}
}

// TestRoleScreenshots renders Writer and Production side by side: the
// View menu with Working As, the ribbon's View tab with ☰ open, the side
// panel, the Preferences' General and Script tabs, desktop and phone
// (SHOT_DIR=… go test -run TestRoleScreenshots). The test window draws
// no menu bar, so menus show as pop-ups where they would open.
func TestRoleScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(dir, 0o755)
	w, app := newDialogTestWindow(t)
	p := filepath.Join(t.TempDir(), "locked.fountain")
	os.WriteFile(p, []byte(suggestScript+"\n/* Production\nscene numbers: locked, 10A\nrevision: Blue, 2026-11-20\n*/\n"), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	c := w.fyneWindow.Canvas()
	snap := func(name string) {
		t.Helper()
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, "role-"+name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		png.Encode(f, c.Capture())
	}
	clear := func() {
		for c.Overlays().Top() != nil {
			c.Overlays().Remove(c.Overlays().Top())
		}
	}
	menu := func(title string, x float32) *widget.PopUpMenu {
		for _, m := range w.fyneWindow.MainMenu().Items {
			if m.Label == title {
				pm := widget.NewPopUpMenu(m, c)
				pm.ShowAtPosition(fyne.NewPos(x, 4))
				return pm
			}
		}
		return nil
	}
	for _, role := range []string{roleWriter, roleProduction} {
		workAsFor(t, w, role)
		for _, size := range []struct {
			name string
			size fyne.Size
		}{{"desktop", fyne.NewSize(1100, 680)}, {"phone", fyne.NewSize(380, 800)}} {
			w.fyneWindow.Resize(size.size)
			w.settings.SetString(lookPref, "menus")
			w.applyEditorSettings()
			if !w.paneShown("notes") {
				w.togglePane("notes")
			}
			clear()
			// the menu bar's menus, in a menu (the test window draws none)
			widget.NewPopUpMenu(w.menusMenu(), c).ShowAtPosition(fyne.NewPos(4, 4))
			snap(role + "-menus-" + size.name)
			clear()
			menu("View", 4)
			snap(role + "-view-menu-" + size.name)
			clear()
			r := ribbonOnFor(t, w)
			r.selectTab("View")
			w.showMenusAt(r.menus)
			snap(role + "-ribbon-" + size.name)
			clear()
			r.selectTab("Write")
			snap(role + "-ribbon-write-" + size.name)
		}
		w.settings.SetString(lookPref, "menus")
		w.applyEditorSettings()
		w.fyneWindow.Resize(fyne.NewSize(1100, 680))
		pd := NewPreferencesDialog(app, w.fyneWindow)
		pd.Show()
		pd.tabs.SelectIndex(0)
		pw := pd.win.Canvas()
		pd.win.Resize(fyne.NewSize(640, 720))
		for i, tab := range []string{"general", "script"} {
			pd.tabs.SelectIndex([]int{0, 2}[i])
			f, _ := os.Create(filepath.Join(dir, "role-"+role+"-preferences-"+tab+".png"))
			png.Encode(f, pw.Capture())
			f.Close()
		}
		pd.close()
	}
}
