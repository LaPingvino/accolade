// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/lexington/lex"
)

// issueLocking issues set name with the pages locked as it prints them
// (White's offer), waiting for the pages.
func issueLocking(t *testing.T, w *MainWindow, name string, day int) {
	t.Helper()
	done := false
	w.issueDraft(lex.RevisionSet{Name: name, Date: time.Date(2026, 11, day, 0, 0, 0, 0, time.Local)}, "", true,
		func(err error) {
			if err != nil {
				t.Error(err)
			}
			done = true
		})
	waitFor(t, name+" issued", func() bool { return done })
}

// issueNames are the store's issues' names, in order.
func issueNames(t *testing.T, w *MainWindow) []string {
	t.Helper()
	var names []string
	for _, d := range draftsOf(t, w) {
		if d.Kind == kindIssue {
			names = append(names, d.Name)
		}
	}
	return names
}

// Undo right after White takes the whole issue back, by the menu, by
// Ctrl+Z in the editor, and Redo issues it again, as it was.
func TestUndoIssue(t *testing.T) {
	w, _ := scriptWindow(t, revScript)
	before := w.textEditor.Text()
	issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	issued := w.textEditor.Text()
	ds := draftsOf(t, w)
	if len(ds) != 1 || ds[0].Made == nil || !ds[0].Made.SceneLock {
		t.Fatalf("store %+v", ds)
	}
	id := ds[0].ID
	if !w.cmdEnv().issued {
		t.Error("the menus do not know the script is issued")
	}
	for _, undo := range []func(){w.undo, func() { w.textEditor.TypedShortcut(&fyne.ShortcutUndo{}) }} {
		undo()
		if got := w.textEditor.Text(); got != before {
			t.Fatalf("Undo left\n%s", got)
		}
		if got := issueNames(t, w); len(got) != 0 {
			t.Fatalf("Undo left the store with %v", got)
		}
		if _, err := w.extrasHome().readExtra(draftsDir + id + "/text.fountain"); err == nil {
			t.Error("the issue's text is still there")
		}
		if w.cmdEnv().issued || w.scriptIssued() {
			t.Error("still issued after Undo")
		}
		w.redo()
		if got := w.textEditor.Text(); got != issued {
			t.Fatalf("Redo gave\n%s", got)
		}
		ds := draftsOf(t, w)
		if len(ds) != 1 || ds[0].ID != id || ds[0].Colour != "White" || ds[0].Made == nil {
			t.Fatalf("Redo's store %+v", ds)
		}
		if !w.scriptIssued() {
			t.Error("not issued after Redo")
		}
	}
	// marks follow: an edit after White is Blue
	replaceText(t, w, "Mud everywhere.", "Mud.")
	if got := marked(w); got["Mud."] != "Blue" {
		t.Errorf("marks after Redo %v", got)
	}
}

// In an .accolade: White, an edit, Blue; Undo of Blue's issue leaves
// White, the edit and the bundle without Blue; Redo puts it back.
func TestUndoIssueInBundle(t *testing.T) {
	w := draftsWindow(t)
	p := filepath.Join(t.TempDir(), "Rain.accolade")
	w.textEditor.SetText(revScript)
	if err := w.saveToFile(p); err != nil {
		t.Fatal(err)
	}
	w.typeAfterRename("", "untitled.fountain", p, p)
	w.currentFile = p
	issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	replaceText(t, w, "Mud everywhere.", "Mud, and a dog.")
	edited := w.textEditor.Text()
	issueNow(t, w, "Blue", time.Date(2026, 11, 20, 0, 0, 0, 0, time.Local))
	if got := issueNames(t, w); len(got) != 2 {
		t.Fatalf("issues %v", got)
	}
	w.undo()
	w.flushBundleWrites()
	if got := w.textEditor.Text(); got != edited {
		t.Fatalf("Undo of Blue left\n%s", got)
	}
	if got := issueNames(t, w); len(got) != 1 || got[0] != "White" {
		t.Fatalf("store after Undo %v", got)
	}
	for _, n := range w.bundle.extras {
		if strings.Contains(n, "blue") {
			t.Errorf("the bundle still holds %s", n)
		}
	}
	if got := marked(w); got["Mud, and a dog."] != "Blue" {
		t.Errorf("the edit is not the open Blue again: %v", got)
	}
	w.redo()
	w.flushBundleWrites()
	if got := issueNames(t, w); len(got) != 2 || got[1] != "Blue" {
		t.Fatalf("store after Redo %v", got)
	}
	if currentRevisionOf(w.textEditor.Text()) != "Blue, 2026-11-20" {
		t.Errorf("revision line %q", currentRevisionOf(w.textEditor.Text()))
	}
}

// A draft saved right after an issue names its text (the same text,
// stored once): taking the issue back leaves that text for the draft.
func TestUndoIssueKeepsSharedText(t *testing.T) {
	w, _ := scriptWindow(t, revScript)
	issueNow(t, w, "White", time.Now())
	same, _ := saveDraftNow(t, w, "Same")
	w.undo()
	ds := draftsOf(t, w)
	if len(ds) != 1 || ds[0].Name != "Same" {
		t.Fatalf("store %+v", ds)
	}
	if _, err := w.extrasHome().readExtra(same.Text); err != nil {
		t.Fatalf("the draft's text went with the issue: %v", err)
	}
	w.redo()
	if got := issueNames(t, w); len(got) != 1 {
		t.Fatalf("Redo: %v", got)
	}
}

// Withdraw Last Issue…, in a window opened later (no undo history): White
// locked the numbers and the pages; Blue locked the pages again. Blue
// withdrawn: White's revision line and pages back. Then White withdrawn:
// no lock, no numbers, no line, the script as before; Undo issues it
// again, Redo withdraws it again.
func TestWithdrawLastIssue(t *testing.T) {
	holdUI(t)
	w, p := scriptWindow(t, revScript)
	before := w.textEditor.Text()
	issueLocking(t, w, "White", 1)
	white := w.textEditor.Text()
	if pageLockOf(white) == nil || !sceneLockOf(white).locked {
		t.Fatalf("White did not lock:\n%s", white)
	}
	replaceText(t, w, "Mud everywhere.", "Mud everywhere, and a dog.")
	edited := w.textEditor.Text()
	issueWait(t, w, "Blue", 20, "")
	if currentRevisionOf(w.textEditor.Text()) != "Blue, 2026-11-20" {
		t.Fatal("no Blue")
	}
	if err := w.saveToFile(p); err != nil {
		t.Fatal(err)
	}
	w2 := againWindow(t, w)
	if err := w2.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	menuItemByLabel(t, w2, "Withdraw Last Issue…")
	w2.withdrawLastIssue()
	if !dialogShows(w2, "Withdraw Blue") || !dialogShows(w2, "already gone out") {
		t.Fatal("no Withdraw Blue dialog with its warning")
	}
	wd, err := w2.planWithdrawal()
	if err != nil {
		t.Fatal(err)
	}
	w2.withdrawNow(wd)
	if got := w2.textEditor.Text(); got != edited {
		t.Fatalf("Blue withdrawn:\n%s\nwant\n%s", got, edited)
	}
	if got := issueNames(t, w2); len(got) != 1 || got[0] != "White" {
		t.Fatalf("store %v", got)
	}
	wd, err = w2.planWithdrawal()
	if err != nil {
		t.Fatal(err)
	}
	w2.withdrawNow(wd)
	got := w2.textEditor.Text()
	if pageLockOf(got) != nil || sceneLockOf(got).locked || currentRevisionOf(got) != "" || strings.Contains(got, "#1#") {
		t.Fatalf("White withdrawn:\n%s", got)
	}
	if want := strings.Replace(before, "Mud everywhere.", "Mud everywhere, and a dog.", 1); got != want {
		t.Errorf("White withdrawn: not the script as before\n%q\nwant\n%q", got, want)
	}
	if len(issueNames(t, w2)) != 0 || w2.scriptIssued() {
		t.Fatal("still issued")
	}
	w2.withdrawLastIssue() // nothing left: says so
	if !strings.Contains(w2.statusText(), "nothing has been issued") {
		t.Errorf("status %q", w2.statusText())
	}
	w2.undo() // White issued again
	if got := issueNames(t, w2); len(got) != 1 || got[0] != "White" {
		t.Fatalf("Undo of the withdrawal: %v", got)
	}
	if !sceneLockOf(w2.textEditor.Text()).locked || pageLockOf(w2.textEditor.Text()) == nil {
		t.Error("Undo did not lock again")
	}
	w2.redo()
	if len(issueNames(t, w2)) != 0 {
		t.Error("Redo of the withdrawal left White")
	}
}

// An imported FDX's issue is its own record: not withdrawn.
func TestWithdrawRefusesImported(t *testing.T) {
	w, _ := scriptWindow(t, revScript)
	w.keepImportedRevisions("X.fdx", []lex.RevisionSet{{Name: "Blue"}, {Name: "Pink"}}, nil)
	if _, err := w.planWithdrawal(); err == nil || !strings.Contains(err.Error(), "X.fdx") {
		t.Errorf("imported issue: %v", err)
	}
}

// The Withdraw dialog at a desktop's size and a phone's (SHOT_DIR).
func TestWithdrawScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(dir, 0o755)
	holdUI(t)
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"desktop", fyne.NewSize(1100, 700)}, {"phone", fyne.NewSize(380, 800)}} {
		w, _ := scriptWindow(t, revScript)
		w.fyneWindow.Resize(size.size)
		issueLocking(t, w, "White", 1)
		replaceText(t, w, "Mud everywhere.", "Mud everywhere, and a dog.")
		issueWait(t, w, "Blue", 20, "")
		wd, err := w.planWithdrawal()
		if err != nil {
			t.Fatal(err)
		}
		d := w.withdrawDialog(wd)
		d.Show()
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, "undoissue-withdraw-"+size.name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, w.fyneWindow.Canvas().Capture())
		f.Close()
		d.Hide()
	}
}

// Undo right after an issue that laid out and locked the pages (in the
// background): White's locks and Blue's lock again, each one step.
func TestUndoLockingIssue(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	before := w.textEditor.Text()
	issueLocking(t, w, "White", 1)
	white := w.textEditor.Text()
	w.undo()
	if got := w.textEditor.Text(); got != before || len(issueNames(t, w)) != 0 {
		t.Fatalf("Undo of White: %v\n%s", issueNames(t, w), got)
	}
	w.redo()
	if got := w.textEditor.Text(); got != white || len(issueNames(t, w)) != 1 {
		t.Fatalf("Redo of White: %v\n%s", issueNames(t, w), got)
	}
	replaceText(t, w, "Mud everywhere.", "Mud everywhere, and a dog.")
	edited := w.textEditor.Text()
	issueWait(t, w, "Blue", 20, "")
	blue := w.textEditor.Text()
	w.undo()
	if got := w.textEditor.Text(); got != edited {
		t.Fatalf("Undo of Blue:\n%s", got)
	}
	if got := issueNames(t, w); len(got) != 1 || got[0] != "White" {
		t.Fatalf("store %v", got)
	}
	w.redo()
	if w.textEditor.Text() != blue || len(issueNames(t, w)) != 2 {
		t.Fatal("Redo of Blue")
	}
}

// In a bundle written in the background: Undo while the issue is still
// being written keeps the step (and says so); once written it goes, and
// Redo waits for the withdrawal in turn.
func TestUndoIssueWhileWriting(t *testing.T) {
	holdUI(t)
	realPaths(t)
	w := draftsWindow(t)
	p := filepath.Join(t.TempDir(), "Rain.accolade")
	w.textEditor.SetText(revScript)
	if err := w.saveToFile(p); err != nil {
		t.Fatal(err)
	}
	w.flushBundleWrites()
	w.typeAfterRename("", "untitled.fountain", p, p)
	w.currentFile = p
	before := w.textEditor.Text()
	done := false
	w.issueDraft(lex.RevisionSet{Name: "White", Date: time.Now()}, "", false, func(err error) {
		if err != nil {
			t.Error(err)
		}
		done = true
	})
	issued := w.textEditor.Text()
	if done {
		t.Skip("the write was done at once")
	}
	w.undo()
	if w.textEditor.Text() != issued || !strings.Contains(w.statusText(), "still being written") {
		t.Fatalf("Undo while writing: status %q", w.statusText())
	}
	waitFor(t, "White written", func() bool { return done })
	w.undo()
	if w.textEditor.Text() != before {
		t.Fatal("Undo once written")
	}
	w.redo() // the withdrawal is under way
	if w.textEditor.Text() != before {
		t.Fatal("Redo while withdrawing")
	}
	waitFor(t, "White withdrawn", func() bool { return !w.bundleBusy() && len(issueNames(t, w)) == 0 })
	w.redo()
	if w.textEditor.Text() != issued {
		t.Fatal("Redo once withdrawn")
	}
	waitFor(t, "White back", func() bool { return !w.bundleBusy() && len(issueNames(t, w)) == 1 })
}

// An issue kept before entries noted what they made: Withdraw takes the
// entry and the revision line, the locks stay, and the dialog says so.
func TestWithdrawWithoutMade(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	issueLocking(t, w, "White", 1)
	home := w.extrasHome()
	data, err := home.readExtra(draftsIndexName)
	if err != nil {
		t.Fatal(err)
	}
	var idx draftsIndex
	json.Unmarshal(data, &idx)
	idx.Drafts[0].Made = nil
	data, _ = json.Marshal(idx)
	if err := home.dir.write([]bundleFile{{draftsIndexName, data}}); err != nil {
		t.Fatal(err)
	}
	wd, err := w.planWithdrawal()
	if err != nil {
		t.Fatal(err)
	}
	w.withdrawDialog(wd).Show()
	if !dialogShows(w, "before Accolade noted") {
		t.Error("the dialog does not say the locks stay")
	}
	w.withdrawNow(wd)
	got := w.textEditor.Text()
	if currentRevisionOf(got) != "" || !sceneLockOf(got).locked || pageLockOf(got) == nil {
		t.Fatalf("withdrawn without a note:\n%s", got)
	}
	if len(issueNames(t, w)) != 0 {
		t.Error("still in the store")
	}
}

// Typing while an issue lays out the pages stops it: no entry, and the
// scene lock it made stays its own undo step.
func TestIssueStopsOnTyping(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	var got error
	done := false
	w.issueDraft(lex.RevisionSet{Name: "White", Date: time.Now()}, "", true, func(err error) { got, done = err, true })
	replaceText(t, w, "Mud everywhere.", "Mud.")
	waitFor(t, "the issue ends", func() bool { return done })
	if got == nil || !strings.Contains(got.Error(), "changed while its pages") {
		t.Fatalf("error %v", got)
	}
	if len(issueNames(t, w)) != 0 || w.textEditor.UndoTag() != nil {
		t.Fatal("issued anyway")
	}
	w.undo() // the typing
	w.undo() // the lock
	if sceneLockOf(w.textEditor.Text()).locked {
		t.Error("the scene lock is not a step of its own")
	}
}
