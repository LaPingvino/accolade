package main

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/scriptdiff"
)

const draftScriptOld = `INT. KITCHEN - DAY

Bram cooks.

BRAM
Dinner's ready.

EXT. GARDEN - NIGHT

Rain falls on the roses.

INT. SHED - DAY

Tom waits by the door.
`

const draftScriptNew = `INT. KITCHEN - DAY

Bram cooks a stew, slowly.

BRAM
Dinner's ready.

INT. SHED - DAY

Tom waits by the door.

TOM
Who's there?
`

// draftsShotWindow is a window with Script.accolade open, two drafts in
// it, and the script changed since.
func draftsShotWindow(t *testing.T, size fyne.Size) *MainWindow {
	t.Helper()
	w := draftsWindow(t)
	p := filepath.Join(t.TempDir(), "Script.accolade")
	os.WriteFile(p, makeZip(t, bundleItems("Script.textbundle/")), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	w.fyneWindow.Resize(size)
	w.textEditor.SetText(draftScriptOld)
	saveDraftNow(t, w, "First draft")
	w.textEditor.SetText(draftScriptOld + "\nTOM\nHello?\n")
	saveDraftNow(t, w, "For Anna")
	w.textEditor.SetText(draftScriptNew)
	return w
}

func TestDraftsDialog(t *testing.T) {
	w := draftsShotWindow(t, fyne.NewSize(1100, 700))
	dd, err := newDraftsDialog(w)
	if err != nil {
		t.Fatal(err)
	}
	dd.dialog.Show()
	if len(dd.drafts) != 2 || dd.drafts[0].Name != "For Anna" || dd.selected != 0 {
		t.Fatalf("drafts %+v, selected %d", dd.drafts, dd.selected)
	}
	if dd.compare.Disabled() || dd.restore.Disabled() {
		t.Error("buttons off with a draft selected")
	}
	// Compare the newest with the script now
	cv := showCompare(w, dd.drafts, dd.drafts[0].ID, nowID)
	removed, added, changed := scriptdiff.Count(cv.rows)
	if removed != 3 || added != 1 || changed != 1 {
		t.Errorf("cut %d added %d changed %d", removed, added, changed)
	}
	if !strings.Contains(cv.summary.Text, "3 scenes changed") {
		t.Errorf("summary %q", cv.summary.Text)
	}
	// two drafts
	cv.from.SetSelected(cv.labelOf(dd.drafts[1].ID))
	cv.to.SetSelected(cv.labelOf(dd.drafts[0].ID))
	if removed, added, changed := scriptdiff.Count(cv.rows); removed != 0 || added != 1 || changed != 0 {
		t.Errorf("drafts: cut %d added %d changed %d", removed, added, changed)
	}
	cv.to.SetSelected(cv.labelOf(dd.drafts[1].ID))
	if cv.summary.Text != "The same, word for word." {
		t.Errorf("the same: %q", cv.summary.Text)
	}
	// in the menu
	found := 0
	for _, it := range w.fyneWindow.MainMenu().Items {
		for _, m := range it.Items {
			if it.Label == "Collaboration" && (m.Label == "Save Draft…" || m.Label == "Drafts…") {
				found++
			}
		}
	}
	if found != 2 {
		t.Error("no Collaboration > Save Draft… and Drafts…")
	}
}

// On a phone the dialogs fit, the compare is one under the other.
func TestDraftsDialogPhone(t *testing.T) {
	w := draftsShotWindow(t, fyne.NewSize(380, 800))
	dd, _ := newDraftsDialog(w)
	dd.dialog.Show()
	if top := w.fyneWindow.Canvas().Overlays().Top(); top == nil || top.MinSize().Width > 380 {
		t.Errorf("drafts dialog %v wide", top.MinSize().Width)
	}
	dd.dialog.Hide()
	cv := showCompare(w, dd.drafts, dd.drafts[0].ID, nowID)
	if !cv.narrow {
		t.Error("not narrow")
	}
	if top := w.fyneWindow.Canvas().Overlays().Top(); top == nil || top.MinSize().Width > 380 {
		t.Errorf("compare %v wide", top.MinSize().Width)
	}
	for _, o := range cv.body.Objects {
		if _, ok := o.(*widget.RichText); ok {
			return // one under the other
		}
	}
	t.Error("no row one under the other on a phone")
}

// The Save Draft dialog saves under the name typed.
func TestSaveDraftDialog(t *testing.T) {
	w := draftsShotWindow(t, fyne.NewSize(1100, 700))
	w.showSaveDraft()
	top := w.fyneWindow.Canvas().Overlays().Top()
	if top == nil {
		t.Fatal("no dialog")
	}
	entry := findEntry(top)
	if entry == nil {
		t.Fatal("no name field")
	}
	entry.SetText("Third")
	entry.OnSubmitted("Third")
	if ds := draftsOf(t, w); len(ds) != 3 || ds[2].Name != "Third" {
		t.Errorf("drafts %+v", ds)
	}
}

func findEntry(o fyne.CanvasObject) *widget.Entry {
	switch x := o.(type) {
	case *widget.Entry:
		return x
	case *fyne.Container:
		for _, c := range x.Objects {
			if e := findEntry(c); e != nil {
				return e
			}
		}
	case fyne.Widget:
		for _, c := range test.WidgetRenderer(x).Objects() {
			if e := findEntry(c); e != nil {
				return e
			}
		}
	}
	return nil
}

// TestDraftsScreenshots renders Drafts… and Compare at a desktop's size
// and a phone's (SHOT_DIR as in screenshot_test.go).
func TestDraftsScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(dir, 0o755)
	shot := func(w *MainWindow, name string) {
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, w.fyneWindow.Canvas().Capture())
		f.Close()
	}
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"wide", fyne.NewSize(1100, 700)}, {"phone", fyne.NewSize(380, 800)}} {
		w := draftsShotWindow(t, size.size)
		dd, _ := newDraftsDialog(w)
		dd.dialog.Show()
		shot(w, "drafts-"+size.name+".png")
		dd.dialog.Hide()
		showCompare(w, dd.drafts, dd.drafts[0].ID, nowID)
		shot(w, "compare-"+size.name+".png")
	}
}
