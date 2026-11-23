package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/LaPingvino/accolade/internal/tooltip"
)

// endIDs are the ids on the ribbon's tab row's end, in order.
func endIDs(r *ribbonBar) []string {
	var out []string
	for _, it := range r.end.items {
		for _, b := range append([]*ribbonButton{r.palette, r.order, r.menus}, r.pinned...) {
			if it.obj == fyne.CanvasObject(b) || b == r.palette && r.search != nil && it.obj == fyne.CanvasObject(r.search.Container) {
				out = append(out, b.c.id)
			}
		}
	}
	return out
}

// Add to the Top Row from a ribbon button's menu, Remove from a top row
// button's; the desktop's and the phone's kept apart; Reset; ☰ stays.
func TestTopRowPins(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	w.fyneWindow.Resize(fyne.NewSize(1200, 700))
	w.fyneWindow.Canvas().Capture()
	if got := endIDs(r); !slices.Equal(got, []string{"app.palette", "view.ribbonLock", "view.menu"}) {
		t.Fatalf("the standard end: %v", got)
	}
	r.selectTab("Edit")
	m := r.buttonMenu(buttonOf(r, "edit.find"))
	var add *fyne.MenuItem
	for _, it := range m.Items {
		if it.Label == "Add to the Top Row" {
			add = it
		}
	}
	if add == nil {
		t.Fatalf("no Add to the Top Row in %v", m.Items)
	}
	add.Action()
	if got := endIDs(r); !slices.Equal(got, []string{"edit.find", "app.palette", "view.ribbonLock", "view.menu"}) {
		t.Fatalf("after the add: %v", got)
	}
	if b := w.headerBar.button("edit.find"); b == nil {
		t.Error("the Menus look's toolbar has no Find")
	}
	// the pinned one's own menu takes it off
	if it := r.buttonMenu(r.pinned[0]); it == nil || it.Items[0].Label != "Remove from the Top Row" {
		t.Fatalf("the pinned one's menu: %v", it)
	}
	// taking the lock off; ☰ has no such item
	if r.buttonMenu(r.menus) != nil {
		t.Error("☰ can be taken off")
	}
	r.buttonMenu(r.order).Items[0].Action()
	if got := endIDs(r); slices.Contains(got, "view.ribbonLock") {
		t.Errorf("the lock still there: %v", got)
	}
	if !w.cmdEnv().topEdited {
		t.Error("not edited")
	}
	// a phone has its own
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.fyneWindow.Canvas().Capture()
	if got := endIDs(r); !slices.Equal(got, []string{"app.palette", "view.ribbonLock", "view.menu"}) {
		t.Errorf("the phone's end: %v", got)
	}
	w.fyneWindow.Resize(fyne.NewSize(1200, 700))
	w.fyneWindow.Canvas().Capture()
	w.resetTopRow()
	if got := endIDs(r); !slices.Equal(got, []string{"app.palette", "view.ribbonLock", "view.menu"}) {
		t.Errorf("after Reset: %v", got)
	}
	// the toolbar's button: a right-click there takes it off (Menus look)
	w.addToTopRow("edit.find")
	b := w.headerBar.button("edit.find")
	if b == nil || b.Secondary == nil {
		t.Fatal("no toolbar Find with a menu")
	}
	w.removeFromTopRow("edit.find")
	if w.headerBar.button("edit.find") != nil || w.onTopRow("edit.find") {
		t.Error("Find still on the top row")
	}
	// the toolbar's standard ones can go too, and come back
	w.settings.SetString(lookPref, "menus")
	w.applyLook()
	w.removeFromTopRow("file.export")
	if w.headerBar.button("file.export") != nil {
		t.Error("Export still on the toolbar")
	}
	w.addToTopRow("file.export")
	if w.headerBar.button("file.export") == nil || w.topRowEdited() {
		t.Error("Export not back in its place")
	}
}

// The palette: a right-click on a line adds its command; Add to the Top
// Row… picks one; the ex name is matched too.
func TestTopRowPalette(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	ribbonOnFor(t, w)
	w.pickForTopRow()
	p := w.palette
	p.entry.SetText("next source")
	if len(p.shown) == 0 || p.shown[0].c.id != "source.next" {
		t.Fatalf("the pick: %v", p.shown)
	}
	p.run()
	if !w.onTopRow("source.next") {
		t.Error("Next Source Passage not added")
	}
	w.showPalette()
	w.palette.entry.SetText(":fix-lines")
	if len(w.palette.shown) == 0 || w.palette.shown[0].c.id != "edit.fixLines" {
		t.Errorf("the ex name: %v", w.palette.shown)
	}
	w.palette.close()
}

// The search field: the first letter opens the palette, the rest typed
// goes on there; on a phone it is the palette's button.
func TestTopRowSearch(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	w.fyneWindow.Resize(fyne.NewSize(1200, 700))
	w.fyneWindow.Canvas().Capture()
	s := r.search
	if s == nil || s.compact || !s.entry.Visible() {
		t.Fatal("no search field on the desktop")
	}
	c := w.fyneWindow.Canvas()
	c.Focus(s.entry)
	for _, ch := range "sav" {
		c.Focused().TypedRune(ch)
	}
	if w.palette == nil || !w.palette.pop.Visible() || w.palette.entry.Text != "sav" || s.entry.Text != "" {
		t.Fatalf("typed: palette %q, field %q", w.palette.entry.Text, s.entry.Text)
	}
	if w.palette.shown[0].c.id != "file.save" {
		t.Errorf("first: %s", w.palette.shown[0].c.id)
	}
	w.palette.close()
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.fyneWindow.Canvas().Capture()
	if !r.search.compact || r.search.entry.Visible() {
		t.Error("a phone's search is not the button")
	}
}

// On a phone the name keeps its place before the icons, which go into
// the … menu; narrower still, the name goes too, its editor in the menu
// and on the Script middle tab.
func TestTopRowPhoneName(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.fyneWindow.Canvas().Capture()
	r.tabRow.Refresh()
	r.end.Layout(r.end.Objects, r.end.Size())
	if !r.title.Visible() || r.title.Size().Width < 80 {
		t.Fatalf("the phone's name: shown %v, %v wide (end %v)", r.title.Visible(), r.title.Size().Width, r.end.Size())
	}
	if !r.end.more.Visible() {
		t.Error("no … for the icons")
	}
	labels := func() []string {
		var out []string
		for _, it := range r.end.menuItems(nil) {
			out = append(out, it.Label)
		}
		return out
	}
	if got := labels(); len(got) == 0 || slices.ContainsFunc(got, func(s string) bool { return strings.HasPrefix(s, "Rename") }) {
		t.Errorf("the … menu: %v", got)
	}
	// narrower: the name goes, its editor in the menu
	r.end.Resize(fyne.NewSize(100, r.end.Size().Height))
	r.end.Layout(r.end.Objects, r.end.Size())
	if r.title.Visible() {
		t.Error("the name still there in 100 px")
	}
	if got := labels(); len(got) == 0 || !strings.HasPrefix(got[0], "Rename story.fountain") {
		t.Errorf("no Rename in the … menu: %v", got)
	}
}

// A long press on a toolbar button is its menu, not a tap.
func TestTopRowLongPress(t *testing.T) {
	test.NewTempApp(t)
	tapped, menu := 0, 0
	b := tooltip.NewButton(nil, "x", func() { tapped++ }, nil)
	b.Secondary = func(fyne.Position) { menu++ }
	test.TapSecondary(b)
	if menu != 1 || tapped != 0 {
		t.Errorf("right-click: menu %d, tapped %d", menu, tapped)
	}
}

// TestTopRowScreenshots renders the top row and the contextual tabs
// (SHOT_DIR as in screenshot_test.go): the desktop with a command added
// and the search field, in both looks; a phone with the name kept, and
// narrower; the Fix-Up and Title Page tabs.
func TestTopRowScreenshots(t *testing.T) {
	if os.Getenv("SHOT_DIR") == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	w.fyneWindow.Resize(fyne.NewSize(1200, 700))
	w.fyneWindow.Canvas().Capture()
	w.addToTopRow("edit.find")
	w.addToTopRow(elementCommandID(elementCharacter))
	w.textEditor.SetCursorOffset(len("Title: Rain\n\nINT"))
	w.onCursorChanged()
	r.selectTab("Write")
	shotTo(t, w, "toprow-desktop")
	w.settings.SetString(lookPref, "menus")
	w.applyLook()
	shotTo(t, w, "toprow-desktop-menus")
	w.settings.SetString(lookPref, "ribbon")
	w.applyLook()
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.fyneWindow.Canvas().Capture()
	w.addToTopRow("edit.find")
	shotTo(t, w, "toprow-phone")
	w.fyneWindow.Resize(fyne.NewSize(300, 700))
	w.fyneWindow.Canvas().Capture()
	shotTo(t, w, "toprow-phone-300")
	r.end.more.OnTapped()
	shotTo(t, w, "toprow-phone-300-more")
	for c := w.fyneWindow.Canvas(); c.Overlays().Top() != nil; {
		c.Overlays().Remove(c.Overlays().Top())
	}
	w.fyneWindow.Resize(fyne.NewSize(1200, 700))
	w.fyneWindow.Canvas().Capture()
	w.textEditor.SetCursorOffset(3)
	w.onCursorChanged()
	r.selectTab("Title Page")
	shotTo(t, w, "tab-title-page")
	w.showImported(filepath.Join(t.TempDir(), "old.pdf"), "Scene 1 - int. kitchen - day -\n\nAnna cooks.\n\nANNA(quietly)\nSoup again.\n", "a PDF")
	w.ribbonRefresh()
	shotTo(t, w, "tab-fix-up")
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.fyneWindow.Canvas().Capture()
	shotTo(t, w, "tab-fix-up-phone")
}
