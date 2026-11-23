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

	"github.com/LaPingvino/accolade/internal/editor/buffer"
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
// sample, and (shown) how to tell it is there, for those the keyboard
// pass names (nil: a dialog, an overlay or a window of its own, or
// nothing on the sample: the audit's report says which).
type dialogAudit struct {
	id    string // a registry command's, or a name of its own
	open  func(w *MainWindow)
	shown func(w *MainWindow) bool
	must  bool // it must open on the sample (LOOP.md names it)
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
	"file.mergeChanges": "Fyne's file dialog first; the merge itself is audited as \"merge view\"",
	"file.share":        "shares the script over the network as it opens",
	"file.join":         "joins over the network",
}

// mustOpen are the dialogs and views the keyboard pass names (LOOP.md):
// each must open on the sample, or the audit fails.
var mustOpen = []string{"production.renumber", "production.unlockNumbers", "production.issue", "production.reports",
	"file.drafts", "file.saveDraft", "file.titlePage", "structure.swap", "structure.move", "write.newScene", "view.stages",
	"app.preferences", "settings editor", "merge view", "collab.compare", "production.stripboard", "view.tableRead", "view.cards"}

// auditPanes are the side panes, by command and pane.
var auditPanes = map[string]string{"view.outline": "outline", "view.notes": "notes", "view.source": "source",
	"view.production": "production", "view.characters": "characters", "view.help": "help", "view.changes": "changes"}

// auditedDialogs are what the audit opens: every "…" command of
// Accolade's own, the views and the panes, the settings editor and the
// merge view.
func auditedDialogs() []dialogAudit {
	var out []dialogAudit
	run := func(c *command) func(w *MainWindow) {
		return func(w *MainWindow) { // as the palette gives it: away only when the role puts it away
			e := w.cmdEnv()
			w.runByName(c, nil, !c.applies(e) && c.appliesForRole(e))
		}
	}
	for _, c := range commands().commands {
		if !strings.HasSuffix(c.label, "…") || notOwnDialogs[c.id] != "" {
			continue
		}
		out = append(out, dialogAudit{id: c.id, open: run(c)})
	}
	views := map[string]string{"production.stripboard": "strips", "view.cards": "cards", "view.tableRead": "tableread"}
	for _, id := range []string{"production.stripboard", "view.cards", "view.tableRead"} {
		mid := views[id]
		out = append(out, dialogAudit{id: id, open: run(cmd(id)), shown: func(w *MainWindow) bool { return w.mid.shown == mid }})
	}
	for i := range out {
		if out[i].id == "collab.compare" {
			out[i].shown = func(w *MainWindow) bool { return w.mid.shown == "compare" }
		}
	}
	for _, id := range []string{"view.outline", "view.notes", "view.source", "view.production", "view.characters", "view.help", "view.changes"} {
		c, pane := cmd(id), auditPanes[id]
		out = append(out, dialogAudit{id: id, open: func(w *MainWindow) {
			if !w.paneInSight(pane) {
				run(c)(w)
			}
		}, shown: func(w *MainWindow) bool { return w.paneInSight(pane) }, must: true})
	}
	out = append(out,
		dialogAudit{id: "settings editor", open: func(w *MainWindow) {
			openTOMLFile(w.app, w.fyneWindow, lexingtonConfig(), lexingtonSettingsText())
		}},
		dialogAudit{id: "merge view", open: func(w *MainWindow) {
			p := filepath.Join(os.TempDir(), fmt.Sprintf("accolade-merge-%d.fountain", os.Getpid()))
			os.WriteFile(p, []byte(strings.Replace(w.textEditor.Text(), "Ochumyelov", "Otchumyelov", 1)), 0o644)
			defer os.Remove(p)
			w.mergeFrom(p)
			for i := 0; i < 300 && w.mid.shown != "compare"; i++ {
				uiSleep(10 * time.Millisecond)
			}
		}, shown: func(w *MainWindow) bool { return w.mid.shown == "compare" }},
	)
	for i := range out {
		out[i].must = out[i].must || slices.Contains(mustOpen, out[i].id)
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
	keepAllSettings(t, w)
	w.fyneWindow.Resize(fyne.NewSize(1200, 800))
	report := os.Getenv("TAB_REPORT") != ""
	var lines, nothing []string
	for i, a := range auditedDialogs() {
		if r := os.Getenv("AUDIT_RANGE"); r != "" { // AUDIT_RANGE=10-20: some of them (bisecting)
			var lo, hi int
			fmt.Sscanf(r, "%d-%d", &lo, &hi)
			if i < lo || i >= hi {
				continue
			}
		}
		inScene(w)
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
		opened := root != w.fyneWindow.Canvas().Content() || a.shown != nil && a.shown(w)
		if a.must && !opened {
			t.Errorf("%s opened nothing on the sample", a.id)
		}
		if !opened {
			nothing = append(nothing, a.id)
		}
		missing, chain := unreached(c, root)
		if report {
			lines = append(lines, fmt.Sprintf("%s: %d reached, %d not, opened %v, middle %q, status %q: %s", a.id, len(chain), len(missing), opened, w.mid.shown, w.statusLabel.Text, strings.Join(missing, "; ")))
		}
		if len(missing) > 0 && !report {
			t.Errorf("%s: Tab does not reach\n  %s", a.id, strings.Join(missing, "\n  "))
		}
		closeIt()
		w.mid.show("") // the script in front again, for the next
		w.focusMain()
	}
	if report {
		sort.Strings(lines)
		t.Log("\n" + strings.Join(lines, "\n"))
	}
	if os.Getenv("AUDIT_RANGE") == "" && !slices.Equal(nothing, openedNothing) {
		t.Errorf("opened nothing on the sample: %q, the audit's list (openedNothing, in docs/KEYBOARD.md) has %q", nothing, openedNothing)
	}
}

// openedNothing are the commands that open nothing over the window on
// the sample as it opens: they need a state it is not in (a character
// opened, an import, a strip chosen, a passage picked, the pages in
// front …) or show a bar or pane the window's own tests check.
var openedNothing = []string{"edit.find", "edit.replace", "source.adapt", "source.linkUp", "source.linkGuess", "source.learned",
	"production.tag", "production.breakdown", "production.schedule", "preview.pageSetup", "strips.moveTo", "strips.editDay",
	"view.topRowAdd", "fixup.applyRules", "fixup.ocrAgain", "chars.rename", "chars.merge", "chars.alias", "source.relinkFromHere"}

// openedNothingWhy says why, for docs/KEYBOARD.md.
var openedNothingWhy = map[string]string{
	"edit.find": "the search bar (TestFocusIsVisible checks it)", "edit.replace": "the search bar (TestFocusIsVisible checks it)",
	"source.adapt": "a passage picked in the Source pane", "source.linkUp": "unlinked scenes that read like the source",
	"source.linkGuess": "a guess for the cursor's scene", "source.learned": "the Source pane's list (the panes are audited)",
	"production.tag": "words selected", "production.breakdown": "the Production pane's Break Down stage (the panes are audited)",
	"production.schedule": "the Production pane's Schedule stage (the panes are audited)", "preview.pageSetup": "the pages in front (File > Page Setup… is audited)",
	"strips.moveTo": "a strip chosen on the stripboard", "strips.editDay": "a day chosen on the stripboard",
	"view.topRowAdd": "the palette's picker, a layer of the window (TestPalette)", "fixup.applyRules": "an imported script with fix-up rules",
	"fixup.ocrAgain": "a script read from a PDF", "chars.rename": "a character opened in the Characters pane",
	"chars.merge": "a character opened in the Characters pane", "chars.alias": "a character opened in the Characters pane",
	"source.relinkFromHere": "a source changed since its lock",
}

// focusDialogs are the dialogs whose focus look is checked pixel by
// pixel (slow: two captures a control): the production and structure
// ones the keyboard pass names, and those writing in the script.
var focusDialogs = []string{"production.renumber", "production.unlockNumbers", "production.issue", "production.reports",
	"file.drafts", "file.titlePage", "structure.swap", "write.newScene", "view.tableRead", "view.stages"}

// The dialogs show where the focus is on every control Tab reaches, on
// a desktop and a phone (the views and panes are TestFocusIsVisible's
// and TestDockFocusVisible's).
func TestDialogsFocusVisible(t *testing.T) {
	if raceDetector {
		t.Skip("pixel capture of focus: checked by the runs without -race (as TestFocusIsVisible)")
	}
	w := sampleWindow(t, sampleAccolade)
	keepAllSettings(t, w)
	byID := map[string]dialogAudit{}
	for _, a := range auditedDialogs() {
		byID[a.id] = a
	}
	for _, size := range []fyne.Size{{Width: 1200, Height: 800}, {Width: 380, Height: 800}} {
		w.fyneWindow.Resize(size)
		for _, id := range focusDialogs {
			if size.Width < 600 && !slices.Contains(focusShots, id) {
				continue // a phone: the shot ones (the step's time: CI's default 10 minutes)
			}
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
var focusShots = []string{"production.renumber", "file.drafts", "structure.swap", "file.titlePage", "view.tableRead"}

// focusShot shoots c with the focus on its third control (past the
// first fields, on a button or a choice), or on a list or area that
// shows its focus by a line round it.
func focusShot(t *testing.T, c fyne.Canvas, dir, name string) {
	t.Helper()
	chain := tabChain(c)
	if len(chain) == 0 {
		return
	}
	at := chain[min(2, len(chain)-1)]
	for _, f := range chain { // a line round it: the drafts list, the Table Read (keyboard.go, tableread_view.go)
		switch f.(type) {
		case *ringList, *trArea:
			at = f
		}
	}
	c.Focus(at)
	scrollToFocus(c)
	uiSleep(0)
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	png.Encode(f, c.Capture())
}

// keepAllSettings puts every setting back as it was once t ends: the
// audits give every command, and the settings are the process's (a
// report's kind chosen, the renumbering last used …).
func keepAllSettings(t *testing.T, w *MainWindow) {
	t.Helper()
	s := w.settings
	s.mu.RLock()
	was := make(map[string]any, len(s.data))
	for k, v := range s.data {
		was[k] = v
	}
	s.mu.RUnlock()
	t.Cleanup(func() {
		s.mu.Lock()
		s.data = was
		s.mu.Unlock()
	})
}

// Tab in a dialog scrolls its scroll to the control the focus went to,
// through the window's key preview (as the keyboard gives it); Tab in
// the script (a cue) walks nothing.
func TestTabScrollsToFocus(t *testing.T) {
	w := sampleWindow(t, sampleAccolade)
	keepAllSettings(t, w)
	w.fyneWindow.Resize(fyne.NewSize(1200, 800))
	w.renumberScenes()
	waitFor(t, "the dialog", func() bool { return w.renumbering != nil && w.fyneWindow.Canvas().Overlays().Top() != nil })
	c := w.fyneWindow.Canvas()
	chain := tabChain(c)
	last := chain[len(chain)-1]
	for _, f := range chain { // the last control in the scroll: a radio far down
		if strings.Contains(fmt.Sprintf("%T", f), "radioItem") {
			last = f
		}
	}
	c.Focus(last)
	o := last.(fyne.CanvasObject)
	off := func() float32 { return fyne.CurrentApp().Driver().AbsolutePositionForObject(o).Y }
	before := off()
	if before+o.Size().Height <= c.Size().Height {
		t.Skipf("the control is in sight already (%v)", before)
	}
	w.keyPreview(fyne.KeyTab, 0)
	waitFor(t, "the scroll to follow", func() bool { return off()+o.Size().Height <= c.Size().Height })
	c.Overlays().Remove(c.Overlays().Top())
	if !followsFocus(c) {
		t.Error("the window's canvas is not one whose Tab follows the focus")
	}
}

// inScene puts the script's cursor in its first scene's action (a
// command about "this scene" has one).
func inScene(w *MainWindow) {
	text := w.textEditor.Text()
	r, err := w.scriptRefs()
	if err != nil || len(r.heads) == 0 {
		return
	}
	line := r.heads[0].line + 2
	w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(min(line, b.LineCount()-1)), false) })
	_ = text
}
