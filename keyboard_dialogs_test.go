package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// The dialogs' Tab order (docs/KEYBOARD.md): every control of every
// dialog, pane and view of Accolade's own is reached by Tab, and every
// one shows that it has the focus (the …FocusVisible test).

// focusablesIn are the controls under root that Tab should reach: shown
// (all the way up), not disabled.
func focusablesIn(root fyne.CanvasObject) []fyne.Focusable {
	var out []fyne.Focusable
	var visit func(o fyne.CanvasObject)
	visit = func(o fyne.CanvasObject) {
		if o == nil || !o.Visible() {
			return
		}
		if f, ok := o.(fyne.Focusable); ok {
			// a row of tabs is one stop: the tab in front, the arrow
			// keys along the row (dockTab, Fyne's tabButton)
			skip, _ := o.(interface{ SkipsFocusChain() bool })
			dis := skip != nil && skip.SkipsFocusChain()
			if d, ok := o.(fyne.Disableable); ok {
				dis = dis || d.Disabled()
			}
			if !dis {
				out = append(out, f)
			}
		}
		switch c := o.(type) {
		case *fyne.Container:
			for _, k := range c.Objects {
				visit(k)
			}
		case fyne.Widget:
			for _, k := range test.WidgetRenderer(c).Objects() {
				visit(k)
			}
		}
	}
	visit(root)
	return out
}

// tabChain is what Tab reaches on c, in order, from nothing focused.
func tabChain(c fyne.Canvas) []fyne.Focusable {
	c.Unfocus()
	var out []fyne.Focusable
	seen := map[fyne.Focusable]bool{}
	for i := 0; i < 300; i++ {
		c.FocusNext()
		f := c.Focused()
		if f == nil || seen[f] {
			break
		}
		seen[f] = true
		out = append(out, f)
	}
	c.Unfocus()
	return out
}

// focusName is a control as a report names it.
func focusName(f fyne.Focusable) string {
	o, _ := f.(fyne.CanvasObject)
	name := ""
	if o != nil {
		name = fyne.AccessibleLabel(o)
	}
	return fmt.Sprintf("%T %q", f, name)
}

// unreached are root's controls that Tab on c does not reach.
func unreached(c fyne.Canvas, root fyne.CanvasObject) (missing []string, chain []string) {
	got := tabChain(c)
	for _, f := range got {
		chain = append(chain, focusName(f))
	}
	for _, f := range focusablesIn(root) {
		if !slices.Contains(got, f) {
			missing = append(missing, focusName(f))
		}
	}
	return missing, chain
}

// dialogAudit is a dialog, pane or view to audit: how it opens on the
// sample, and what it is.
type dialogAudit struct {
	id   string // a registry command's, or a name of its own
	open func(w *MainWindow)
}

// notOwnDialogs are the commands with "…" whose dialog is not Accolade's
// own (the system's or Fyne's file dialog, the browser), or that open no
// dialog on the sample.
var notOwnDialogs = map[string]string{
	"file.open":         "Fyne's file dialog",
	"file.openFolder":   "Fyne's folder dialog",
	"file.import":       "Fyne's file dialog",
	"file.importPandoc": "Fyne's file dialog (pandoc)",
	"file.print":        "the system's print dialog",
	"help.feedback":     "the web browser",
	"preview.savePDF":   "Fyne's file dialog",
	"production.ics":    "Fyne's file dialog",
	"source.saveAs":     "Fyne's file dialog",
	"file.exportClean":  "Fyne's file dialog",
	"file.share":        "shares the script over the network as it opens",
	"file.join":         "joins over the network",
}

// auditedDialogs are what the audit opens: every "…" command of
// Accolade's own, and the views and panes beside them.
func auditedDialogs() []dialogAudit {
	var out []dialogAudit
	for _, c := range commands().commands {
		if !strings.HasSuffix(c.label, "…") || notOwnDialogs[c.id] != "" {
			continue
		}
		c := c
		out = append(out, dialogAudit{c.id, func(w *MainWindow) { w.runByName(c, nil, !c.applies(w.cmdEnv())) }})
	}
	for _, id := range []string{"production.stripboard", "view.cards", "view.tableRead", "view.changes", "view.outline", "view.notes", "view.source", "view.production", "view.characters", "view.help"} {
		if c := commands().byID[id]; c != nil {
			c := c
			out = append(out, dialogAudit{c.id, func(w *MainWindow) { w.runByName(c, nil, !c.applies(w.cmdEnv())) }})
		}
	}
	return out
}

// openedBy runs a.open on w and gives the canvas and root it opened: a
// window of its own, an overlay on w's (a dialog, a pop-up), or w's own
// content (a pane, a view); close puts it away again.
func openedBy(w *MainWindow, a dialogAudit) (c fyne.Canvas, root fyne.CanvasObject, close func()) {
	before := fyne.CurrentApp().Driver().AllWindows()
	wc := w.fyneWindow.Canvas()
	overlays := len(wc.Overlays().List())
	a.open(w)
	uiSleep(0)
	for _, x := range fyne.CurrentApp().Driver().AllWindows() {
		if !slices.Contains(before, x) {
			return x.Canvas(), x.Canvas().Content(), x.Close
		}
	}
	if l := wc.Overlays().List(); len(l) > overlays {
		top := l[len(l)-1]
		return wc, top, func() {
			for len(wc.Overlays().List()) > overlays {
				wc.Overlays().Remove(wc.Overlays().Top())
			}
		}
	}
	return wc, wc.Content(), func() {}
}

// Every dialog of Accolade's own, and its views and panes, on the sample:
// Tab reaches every control.
func TestDialogTabOrder(t *testing.T) {
	w := sampleWindow(t, sampleAccolade)
	w.fyneWindow.Resize(fyne.NewSize(1200, 800))
	report := os.Getenv("TAB_REPORT") != ""
	var lines []string
	for _, a := range auditedDialogs() {
		var c fyne.Canvas
		var root fyne.CanvasObject
		var closeIt func()
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s: panics: %v", a.id, r)
					c = nil
				}
			}()
			c, root, closeIt = openedBy(w, a)
		}()
		if c == nil {
			continue
		}
		missing, chain := unreached(c, root)
		if report {
			lines = append(lines, fmt.Sprintf("%s: %d reached, %d not: %s", a.id, len(chain), len(missing), strings.Join(missing, "; ")))
		}
		if len(missing) > 0 && !report {
			t.Errorf("%s: Tab does not reach\n  %s", a.id, strings.Join(missing, "\n  "))
		}
		closeIt()
		w.focusMain()
	}
	if report {
		sort.Strings(lines)
		t.Log("\n" + strings.Join(lines, "\n"))
	}
}

// focusDialogs are the dialogs whose focus look is checked pixel by
// pixel (slow: two captures a control): the production and structure
// ones the keyboard pass names, and those writing in the script.
var focusDialogs = []string{"production.renumber", "production.unlockNumbers", "production.issue", "production.reports",
	"file.drafts", "file.saveDraft", "file.titlePage", "file.addSettings", "structure.swap", "structure.move",
	"write.newScene", "view.tableRead", "view.stages"}

// The dialogs show where the focus is on every control Tab reaches, on
// a desktop and a phone (the views and panes are TestFocusIsVisible's
// and TestDockFocusVisible's).
func TestDialogsFocusVisible(t *testing.T) {
	if raceDetector {
		t.Skip("pixel capture of focus: checked by the runs without -race (as TestFocusIsVisible)")
	}
	w := sampleWindow(t, sampleAccolade)
	byID := map[string]dialogAudit{}
	for _, a := range auditedDialogs() {
		byID[a.id] = a
	}
	for _, size := range []fyne.Size{{Width: 1200, Height: 800}, {Width: 380, Height: 800}} {
		w.fyneWindow.Resize(size)
		for _, id := range focusDialogs {
			if only := os.Getenv("FOCUS_ONLY"); only != "" && !strings.Contains(","+only+",", ","+id+",") {
				continue // FOCUS_ONLY=file.drafts,…: a quick look at some
			}
			a, ok := byID[id]
			if !ok {
				t.Fatalf("%s is not audited", id)
			}
			start := time.Now()
			c, root, closeIt := openedBy(w, a)
			if testing.Verbose() {
				defer func(id string) { t.Logf("%s %v: %v", id, size, time.Since(start)) }(id)
			}
			if root == w.fyneWindow.Canvas().Content() && id != "view.tableRead" {
				t.Errorf("%s opened no dialog", id)
			}
			if u := invisibleFocus(c); len(u) > 0 {
				t.Errorf("%s at %v: the focus is not seen on\n  %s", a.id, size, strings.Join(u, "\n  "))
			}
			if dir := os.Getenv("SHOT_DIR"); dir != "" && slices.Contains(focusShots, a.id) {
				focusShot(t, c, dir, fmt.Sprintf("focus-%s-%.0f.png", a.id, size.Width))
			}
			closeIt()
			w.focusMain()
		}
	}
}

// focusShots are the dialogs shot with the focus on a control
// (SHOT_DIR=… go test -run TestDialogsFocusVisible).
var focusShots = []string{"production.renumber", "file.drafts", "structure.swap", "file.titlePage"}

// focusShot shoots c with the focus on its third control (past the
// first fields, on a button or a choice).
func focusShot(t *testing.T, c fyne.Canvas, dir, name string) {
	t.Helper()
	chain := tabChain(c)
	if len(chain) == 0 {
		return
	}
	c.Focus(chain[min(2, len(chain)-1)])
	uiSleep(0)
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	png.Encode(f, c.Capture())
}
