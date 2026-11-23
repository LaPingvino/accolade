package main

import (
	"cmp"
	"image/color"
	"reflect"
	"slices"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The keyboard pass (LOOP.md 2026-11-20, docs/KEYBOARD.md): every
// command's keys per platform, the keys a platform keeps for itself, and
// the keys vim's and helix's modes take before the menus would. The
// clash test (keyboard_test.go) and docs/KEYBOARD.md read these.

// keyPlatform is where Accolade runs, as far as keys go.
type keyPlatform int

const (
	platDesktop keyPlatform = iota // Linux and Windows
	platMac
	platBrowser // the WebAssembly build, any system (Ctrl)
)

var keyPlatforms = []keyPlatform{platDesktop, platMac, platBrowser}

func (p keyPlatform) String() string {
	return [...]string{"Linux/Windows", "macOS", "browser"}[p]
}

// keysOn are c's keys on p: its macKeys on a Mac, its webKeys in the
// browser, else its keys (as the registry was made, whatever this build
// swapped in: pcKeys keep the others').
func keysOn(c *command, p keyPlatform) []fyne.Shortcut {
	switch {
	case p == platMac && c.macKeys != nil:
		return c.macKeys
	case p == platBrowser && c.webKeys != nil:
		return c.webKeys
	}
	if c.pcKeys != nil {
		if (onMac && c.macKeys != nil) || (inBrowser && c.webKeys != nil) {
			return c.pcKeys
		}
	}
	return c.keys
}

// macKeyName is a key of a command's macKeys as a Mac names it: they
// are written in a Mac's own modifiers (Super is Cmd, Control is
// Control), whatever this build is.
func macKeyName(s fyne.Shortcut) string { return macName(s, false) }

// macName is s on a Mac; primary: s is a key for every system, its
// Primary (Ctrl here, on Linux) Cmd there.
func macName(s fyne.Shortcut, primary bool) string {
	ks, ok := s.(fyne.KeyboardShortcut)
	if !ok {
		return s.ShortcutName()
	}
	m := ks.Mod()
	cmd := m&fyne.KeyModifierSuper != 0
	ctrl := m&fyne.KeyModifierControl != 0
	if primary && m&fyne.KeyModifierShortcutDefault != 0 {
		cmd, ctrl = true, ctrl && fyne.KeyModifierShortcutDefault != fyne.KeyModifierControl
	}
	var b strings.Builder
	if cmd {
		b.WriteString("Cmd+")
	}
	if ctrl {
		b.WriteString("Control+")
	}
	if m&fyne.KeyModifierShift != 0 {
		b.WriteString("Shift+")
	}
	if m&fyne.KeyModifierAlt != 0 {
		b.WriteString("Option+")
	}
	b.WriteString(keyLabel(ks.Key()))
	return b.String()
}

// keyNamesOn are c's keys on p as p's keyboards name them.
func keyNamesOn(c *command, p keyPlatform) []string {
	var out []string
	for _, s := range keysOn(c, p) {
		switch {
		case p == platMac && c.macKeys != nil:
			out = append(out, macKeyName(s))
		case p == platMac:
			out = append(out, macName(s, true))
		default:
			out = append(out, keyName(s))
		}
	}
	return out
}

// keptKey is a key a platform keeps for itself: a command given it never
// gets it.
type keptKey struct{ key, by string }

// browserKeptKeys are the keys a browser never gives the page: Chromium's
// reserved commands (chrome/browser/ui/accelerator_table.cc,
// kReservedCommandIds: close tab and window, new window, incognito, new
// tab, restore tab, next and previous tab, exit) as its accelerator table
// maps them, read 2026-11-23 (Firefox's own list was not read).
var browserKeptKeys = []keptKey{
	{"Ctrl+N", "a new browser window"},
	{"Ctrl+Shift+N", "a new private window"},
	{"Ctrl+T", "a new tab"},
	{"Ctrl+Shift+T", "the tab closed last, again"},
	{"Ctrl+W", "closes the tab"},
	{"Ctrl+F4", "closes the tab"},
	{"Ctrl+Shift+W", "closes the browser window"},
	{"Alt+F4", "closes the browser window"},
	{"Ctrl+Tab", "the next tab"},
	{"Ctrl+Shift+Tab", "the tab before"},
	{"Ctrl+PgDn", "the next tab"},
	{"Ctrl+PgUp", "the tab before"},
}

// macKeptKeys are the keys macOS keeps for itself (or the app menu has,
// that every Mac app has): Hide, Minimize, the window and app switchers,
// Spotlight, the screenshots, the Dock, Mission Control and the Spaces.
var macKeptKeys = []keptKey{
	{"Cmd+H", "hides the app"},
	{"Cmd+Option+H", "hides the other apps"},
	{"Cmd+M", "minimizes the window"},
	{"Cmd+Tab", "the app switcher"},
	{"Cmd+`", "the app's next window"},
	{"Cmd+Space", "Spotlight"},
	{"Cmd+Shift+3", "a screenshot"},
	{"Cmd+Shift+4", "a screenshot of a part"},
	{"Cmd+Shift+5", "the screenshot tools"},
	{"Cmd+Option+D", "shows or hides the Dock"},
	{"Control+Up", "Mission Control"},
	{"Control+Down", "the app's windows"},
	{"Control+Left", "the Space before"},
	{"Control+Right", "the next Space"},
}

// desktopKeptKeys are the keys Linux desktops (GNOME, KDE, Ubuntu's) and
// Windows keep for themselves, by default.
var desktopKeptKeys = []keptKey{
	{"Ctrl+Alt+L", "locks the screen (GNOME, KDE)"},
	{"Ctrl+Alt+T", "a terminal (Ubuntu)"},
	{"Ctrl+Alt+D", "shows the desktop (Ubuntu)"},
	{"Ctrl+Alt+Delete", "the log-out or security screen"},
	{"Ctrl+Alt+Up", "the workspace above (GNOME)"},
	{"Ctrl+Alt+Down", "the workspace below (GNOME)"},
	{"Ctrl+Alt+Left", "the workspace before (GNOME, KDE)"},
	{"Ctrl+Alt+Right", "the next workspace (GNOME, KDE)"},
	{"Alt+Tab", "the window switcher"},
	{"Alt+F4", "closes the window"},
	{"Alt+F2", "the run dialog (GNOME, KDE)"},
	{"Alt+F1", "the applications menu"},
	{"Alt+Space", "the window menu"},
}

// keptKeysOn are p's kept keys.
func keptKeysOn(p keyPlatform) []keptKey {
	switch p {
	case platMac:
		return macKeptKeys
	case platBrowser:
		return browserKeptKeys
	}
	return desktopKeptKeys
}

// modeShortcuts are the keys with a modifier that vim's and helix's
// normal modes take (vim.go shortcut, helix_multi.go multiShortcut): the
// menu bar's shortcuts are looked at first (commands.go), so a command
// with one of these keys would take it from the mode.
var modeShortcuts = map[string][]string{
	"vim":   {"Ctrl+R", "Alt+J", "Alt+K", "Alt+H", "Alt+L", "Shift+Alt+J", "Shift+Alt+K", "Shift+Alt+H", "Shift+Alt+L"},
	"helix": {"Ctrl+R", "Alt+J", "Alt+K", "Alt+H", "Alt+L", "Shift+Alt+J", "Shift+Alt+K", "Shift+Alt+H", "Shift+Alt+L", "Alt+S", "Alt+,"},
}

// focusFollowers are the canvases whose Tab calls scrollToFocus (the
// main windows': keyPreview).
var focusFollowers sync.Map

// followsFocus reports whether c's Tab scrolls to the focus.
func followsFocus(c fyne.Canvas) bool { _, ok := focusFollowers.Load(c); return ok }

// tabScrolls is keyPreview's part for Tab: unless the focused widget
// keeps Tab (the editor: a cue, a parenthetical), the scrolls follow the
// focus once it has moved.
func (w *MainWindow) tabScrolls() {
	c := w.fyneWindow.Canvas()
	if t, ok := c.Focused().(interface{ AcceptsTab() bool }); ok && t.AcceptsTab() {
		return
	}
	w.after(0, func() { scrollToFocus(c) })
}

// scrollToFocus scrolls every scroll holding c's focused control so that
// the control is in sight: Tab onto a dialog's control below its fold
// shows it (Fyne's scrolls do not follow the focus). Called after Tab
// and Shift+Tab move the focus (keyPreview). It looks through containers,
// scrolls, splits, pop-ups (a dialog), cards and tabs, and any other
// widget's exported fields holding objects: a widget's renderer is not
// public.
func scrollToFocus(c fyne.Canvas) {
	f, ok := c.Focused().(fyne.CanvasObject)
	if !ok || f == nil {
		return
	}
	root := c.Content()
	if top := c.Overlays().Top(); top != nil {
		root = top
	}
	// the scrolls in sight; a control inside a widget (a radio group's
	// item) is found by where it is: in the scroll's content
	var scrolls []*container.Scroll
	seen := map[uintptr]bool{} // a widget's fields may point back up
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if o == nil || !o.Visible() {
			return
		}
		if v := reflect.ValueOf(o); v.Kind() == reflect.Pointer {
			if seen[v.Pointer()] {
				return
			}
			seen[v.Pointer()] = true
		}
		var kids []fyne.CanvasObject
		switch x := o.(type) {
		case *fyne.Container:
			kids = x.Objects
		case *container.Scroll:
			scrolls = append(scrolls, x)
			kids = []fyne.CanvasObject{x.Content}
		case *container.Split:
			kids = []fyne.CanvasObject{x.Leading, x.Trailing}
		case *widget.PopUp:
			kids = []fyne.CanvasObject{x.Content}
		case *widget.Card:
			kids = []fyne.CanvasObject{x.Content}
		case *container.AppTabs:
			if it := x.Selected(); it != nil {
				kids = []fyne.CanvasObject{it.Content}
			}
		default: // another widget: what it holds, by its exported fields (a dialog's overlay: Content)
			kids = heldObjects(o)
		}
		for _, k := range kids {
			walk(k)
		}
	}
	walk(root)
	d := fyne.CurrentApp().Driver()
	fp, fs := d.AbsolutePositionForObject(f), f.Size()
	inside := func(s *container.Scroll) bool {
		if s.Content == nil {
			return false
		}
		p, z := d.AbsolutePositionForObject(s.Content), s.Content.Size()
		return fp.X >= p.X && fp.Y >= p.Y && fp.X+fs.Width <= p.X+z.Width+1 && fp.Y+fs.Height <= p.Y+z.Height+1
	}
	var holding []*container.Scroll
	for _, s := range scrolls {
		if inside(s) {
			holding = append(holding, s)
		}
	}
	slices.SortFunc(holding, func(a, b *container.Scroll) int { // the innermost (smallest) first
		sa, sb := a.Content.Size(), b.Content.Size()
		return cmp.Compare(sa.Width*sa.Height, sb.Width*sb.Height)
	})
	for _, s := range holding {
		at := d.AbsolutePositionForObject(f).Subtract(d.AbsolutePositionForObject(s.Content))
		off, size := s.Offset, s.Size()
		switch {
		case at.Y < off.Y:
			off.Y = at.Y
		case at.Y+fs.Height > off.Y+size.Height:
			off.Y = min(at.Y, at.Y+fs.Height-size.Height)
		}
		switch {
		case at.X < off.X:
			off.X = at.X
		case at.X+fs.Width > off.X+size.Width:
			off.X = min(at.X, at.X+fs.Width-size.Width)
		}
		if off != s.Offset {
			s.ScrollToOffset(off)
		}
	}
}

// heldObjects are the objects o's exported fields hold (Content,
// Objects …): how a widget whose renderer is not public is looked into.
func heldObjects(o fyne.CanvasObject) []fyne.CanvasObject {
	v := reflect.ValueOf(o)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return nil
	}
	v = v.Elem()
	obj := reflect.TypeFor[fyne.CanvasObject]()
	var out []fyne.CanvasObject
	for i := 0; i < v.NumField(); i++ {
		f := v.Type().Field(i)
		if !f.IsExported() || f.Anonymous {
			continue
		}
		switch fv := v.Field(i); {
		case f.Type == obj:
			if c, ok := fv.Interface().(fyne.CanvasObject); ok && c != nil {
				out = append(out, c)
			}
		case f.Type.Kind() == reflect.Slice && f.Type.Elem() == obj:
			out = append(out, fv.Interface().([]fyne.CanvasObject)...)
		}
	}
	return out
}

// ringList is a list that shows it has the focus by a line round it: a
// Fyne list marks the focused row only, which a selected row hides (the
// Drafts dialog's first draft, chosen as it opens).
type ringList struct {
	widget.List
	ring *canvas.Rectangle
}

func newRingList(length func() int, create func() fyne.CanvasObject, update func(widget.ListItemID, fyne.CanvasObject)) *ringList {
	l := &ringList{ring: canvas.NewRectangle(color.Transparent)}
	l.Length, l.CreateItem, l.UpdateItem = length, create, update
	l.ring.StrokeWidth = 2
	l.ring.Hide()
	l.ExtendBaseWidget(l)
	return l
}

// view is the list with its ring, to put where the list goes.
func (l *ringList) view() fyne.CanvasObject { return container.NewStack(l, l.ring) }

func (l *ringList) FocusGained() { l.List.FocusGained(); l.showRing(true) }
func (l *ringList) FocusLost()   { l.List.FocusLost(); l.showRing(false) }

func (l *ringList) showRing(on bool) {
	l.ring.StrokeColor = theme.Color(theme.ColorNameFocus)
	l.ring.Hidden = !on
	l.ring.Refresh()
}
