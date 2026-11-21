package main

import (
	"os"
	"strings"

	"fyne.io/fyne/v2"
)

// Working as a writer or for production (Preferences > General, View >
// Working As): a writer is not faced with production's settings. In
// Writer the Production menu, the Production tab of the side panel and of
// the ribbon, the header's Production button, the context menu's tagging
// and the Preferences' Production card are put away, and their keys with
// them (a command's role, commands.go, is a rule like its when). The
// script works the same: locked scene numbers stay locked, revision marks
// still print as set, tags and shoot days are kept; a script locked for
// production says so once on the status line.
//
// The role is the preferences' ("role"), and a script can keep its own
// ("script-roles": "writer:/path" or "production:/path", like the
// document types, only where it differs from the preferences'), so a
// producer can keep one script in Writer view. A script not saved yet
// keeps its own in the window until it is (unsavedRole).
//
// New users start as writers. Users from before the role (a settings file
// without one) become what their settings show they did: Production if
// they ever used a production feature (a production folder kept for a
// script, a production preference changed, a recent script locked or
// issued), else Writer.

const (
	rolePref        = "role"
	scriptRolesPref = "script-roles"
	roleWriter      = "writer"
	roleProduction  = "production"
	maxScriptRoles  = 200
)

// defaultRole is a new user's role. The tests start as Production (the
// whole app, as they were written before the role; TestMain).
var defaultRole = roleWriter

// cmdRole is who a command is for: everyone, or production.
type cmdRole int

const (
	anyRole cmdRole = iota
	productionRole
)

// in reports whether a command of role r is there in env.
func (r cmdRole) in(e cmdEnv) bool { return r != productionRole || !e.writer }

// roleLabels are the roles as Preferences and View > Working As name them.
var roleLabels = []string{"Writer", "Production"}

func roleLabel(role string) string {
	if role == roleProduction {
		return roleLabels[1]
	}
	return roleLabels[0]
}

func roleOfLabel(label string) string {
	if label == roleLabels[1] {
		return roleProduction
	}
	return roleWriter
}

// globalRole is the preferences' role.
func (w *MainWindow) globalRole() string {
	if w.settings == nil || w.settings.GetString(rolePref) != roleWriter {
		return roleProduction
	}
	return roleWriter
}

// scriptRoleEntry is how a script's own role is kept in the list.
func scriptRoleEntry(path, role string) string { return role + ":" + path }

// ownRole is the role kept for the window's script ("" if it follows the
// preferences').
func (w *MainWindow) ownRole() string {
	if w.currentFile == "" {
		return w.unsavedRole
	}
	return w.keptRole(w.currentFile)
}

// keptRole is the role kept for the file at path ("" for none).
func (w *MainWindow) keptRole(path string) string {
	if w.settings == nil || path == "" {
		return ""
	}
	path = absPath(path)
	for _, e := range w.settings.GetStringSlice(scriptRolesPref) {
		for _, r := range []string{roleWriter, roleProduction} {
			if e == scriptRoleEntry(path, r) {
				return r
			}
		}
	}
	return ""
}

// keepRole keeps role ("" none) for the file at path, the newest first.
func (w *MainWindow) keepRole(path, role string) {
	if w.settings == nil || path == "" {
		return
	}
	path = absPath(path)
	var list []string
	if role != "" {
		list = append(list, scriptRoleEntry(path, role))
	}
	old := w.settings.GetStringSlice(scriptRolesPref)
	for _, e := range old {
		if e != scriptRoleEntry(path, roleWriter) && e != scriptRoleEntry(path, roleProduction) && len(list) < maxScriptRoles {
			list = append(list, e)
		}
	}
	if len(list) == 0 && len(old) == 0 {
		return
	}
	w.settings.SetStringSlice(scriptRolesPref, list)
	_ = w.settings.Save()
}

// role is the role the window works in: its script's own, else the
// preferences'.
func (w *MainWindow) role() string {
	if r := w.ownRole(); r != "" {
		return r
	}
	return w.globalRole()
}

// writerView reports whether production is put away in the window.
func (w *MainWindow) writerView() bool { return w.role() == roleWriter }

// setOwnRole keeps role ("" none: the preferences') for this script.
func (w *MainWindow) setOwnRole(role string) {
	if w.currentFile == "" {
		w.unsavedRole = role
	} else {
		w.keepRole(w.currentFile, role)
	}
	w.roleChanged(true)
}

// workAs is View > Working As > Writer or Production: for this script if
// it keeps its own role, else the preferences' (every window follows).
func (w *MainWindow) workAs(role string) {
	if w.ownRole() != "" {
		w.setOwnRole(role)
		return
	}
	if w.settings != nil {
		w.settings.SetString(rolePref, role)
		_ = w.settings.Save()
		w.followInPreferences(rolePref)
	}
	if w.app != nil && len(w.app.windows) > 0 {
		for _, o := range w.app.windows {
			o.roleChanged(o == w)
		}
		return
	}
	w.roleChanged(true)
}

// toggleOwnRole is View > Working As > Only for This Script: the script
// keeps the role it is in now, or follows the preferences' again.
func (w *MainWindow) toggleOwnRole() {
	if w.ownRole() != "" {
		w.setOwnRole("")
		w.setStatus("This script follows the preferences again: working as " + roleLabel(w.role()))
		return
	}
	w.setOwnRole(w.role())
	w.setStatus("This script keeps working as " + roleLabel(w.role()) + ", whatever the preferences say")
}

// roleAfterRename moves the script's own role to its new name (a rename,
// Save As), or keeps an unsaved script's once it has a file.
func (w *MainWindow) roleAfterRename(oldPath, newPath string) {
	if newPath == "" {
		return
	}
	switch {
	case oldPath == "":
		if w.unsavedRole != "" {
			w.keepRole(newPath, w.unsavedRole)
			w.unsavedRole = ""
		}
	case absPath(oldPath) != absPath(newPath):
		if r := w.keptRole(oldPath); r != "" {
			w.keepRole(oldPath, "")
			w.keepRole(newPath, r)
		}
	}
}

// roleChanged shows the window as its role says: the menus, the ribbon,
// the header, the side panel; production's panes put away in Writer. said
// shows the status line for it.
func (w *MainWindow) roleChanged(said bool) {
	if w.fyneWindow == nil {
		return
	}
	writer := w.writerView()
	if writer && !w.markdown {
		if w.stripsVisible {
			w.toggleStripboard()
		}
	}
	if w.headerBar != nil {
		w.headerBar.showRole(w.cmdEnv())
	}
	w.refreshMenu()
	w.relayout()
	if w.textEditor != nil {
		w.textEditor.RedrawMarks() // the tagged words' colours (breakdown_view.go)
	}
	if said {
		if msg := w.lockedForWriter(); msg != "" {
			w.setStatus(msg)
		} else {
			w.setStatus("Working as " + roleLabel(w.role()))
		}
	}
}

// lockedForWriter is the status line of a script locked for production
// while production is put away ("" if it is not, or it is not put away).
func (w *MainWindow) lockedForWriter() string {
	if w.markdown || w.textEditor == nil || !w.writerView() {
		return ""
	}
	text := w.textEditor.Text()
	set, _, _ := strings.Cut(currentRevisionOf(text), ",")
	set = strings.TrimSpace(set)
	if set == "" && !sceneLockOf(text).locked && pageLockOf(text) == nil {
		return ""
	}
	if set != "" {
		set = " (" + set + ")"
	}
	return "This script is locked for production" + set + " — switch to Production to change that"
}

// roleMenu is View > Working As: Writer, Production, and whether this
// script keeps its own.
func (w *MainWindow) roleMenu() *fyne.MenuItem {
	role, own := w.role(), w.ownRole() != ""
	writer := fyne.NewMenuItem(roleLabels[0], func() { w.workAs(roleWriter) })
	writer.Checked = role == roleWriter
	production := fyne.NewMenuItem(roleLabels[1], func() { w.workAs(roleProduction) })
	production.Checked = role == roleProduction
	only := fyne.NewMenuItem("Only for This Script", w.toggleOwnRole)
	only.Checked = own
	item := fyne.NewMenuItem("Working As", nil)
	item.ChildMenu = fyne.NewMenu("", writer, production, fyne.NewMenuItemSeparator(), only)
	return item
}

// Users from before the role: what their settings show.

// productionDefaults are the production preferences as they start: one
// changed means production was worked on.
var productionDefaults = map[string]any{
	sceneNumberStylePref:  "10A",
	revisionMarksPref:     "last",
	revisionHeaderPref:    "{set} Revision – {date}",
	revisionDatePref:      "02/01/06",
	"export-revisions":    true,
	"export-page-colours": false,
	continuedsPref:        false,
	"report-kind":         "scenes",
	breakdownHiddenPref:   false,
}

// productionPrefs are the settings the Preferences' Production card has.
var productionPrefs = []string{sceneNumberStylePref, revisionMarksPref, revisionHeaderPref, revisionDatePref, continuedsPref}

// maxRoleScan is how many recent scripts are looked at, and how much of
// each (bytes), for a lock or an issue.
const (
	maxRoleScan     = 20
	maxRoleScanSize = 4 << 20
)

// migratedRole is the role of a user from before the role, from the
// settings as their file had them; read reads a recent script (nil: none
// looked at).
func migratedRole(data map[string]any, read func(path string) ([]byte, error)) string {
	if list, ok := data["production-files"].([]any); ok && len(list) > 0 {
		return roleProduction
	}
	for k, def := range productionDefaults {
		if v, ok := data[k]; ok && v != def {
			return roleProduction
		}
	}
	if read == nil {
		return roleWriter
	}
	recent, _ := data["recent-files"].([]any)
	for i, p := range recent {
		path, ok := p.(string)
		if !ok || i >= maxRoleScan || bundleKindOf(path) != notBundle || isMarkdownFile(path) {
			continue
		}
		b, err := read(path)
		if err != nil || len(b) > maxRoleScanSize {
			continue
		}
		text, _ := decodeScript(b)
		if _, ok := productionBlock(text); ok && (sceneLockOf(text).locked || pageLockOf(text) != nil || currentRevisionOf(text) != "") {
			return roleProduction
		}
	}
	return roleWriter
}

// readSmall reads a file, if it is not larger than maxRoleScanSize.
func readSmall(path string) ([]byte, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if st.Size() > maxRoleScanSize {
		return nil, os.ErrInvalid
	}
	return os.ReadFile(path)
}
