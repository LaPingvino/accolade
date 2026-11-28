// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

// The Fix-Up tab: there after an import, in front; its PDF group only
// for a PDF; gone on Done Fixing or a save, the tab before back in front.
func TestFixUpTab(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	r.selectTab("View")
	if tabButton(r, "Fix-Up").Visible() {
		t.Fatal("a Fix-Up tab before any import")
	}
	w.showImported(filepath.Join(t.TempDir(), "old.pdf"), "INT. ROOM - DAY\n\nScene 6 - int. day -\n\nWords.\n", "a PDF")
	w.ribbonRefresh()
	if !tabButton(r, "Fix-Up").Visible() || r.selected != "Fix-Up" {
		t.Fatalf("after the import: shown %v, %s chosen", tabButton(r, "Fix-Up").Visible(), r.selected)
	}
	got := dumpRibbon(r)
	for _, want := range []string{"## Fix", "Fix Lines", "## PDF", "OCR Again", "Check Text", "## Done"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q:\n%s", want, got)
		}
	}
	w.checkGarbled() // the file is not there
	if !strings.Contains(w.statusLabel.Text, "not there") {
		t.Errorf("the check: %q", w.statusLabel.Text)
	}
	w.doneFixing()
	if tabButton(r, "Fix-Up").Visible() || r.selected != "View" {
		t.Errorf("after Done: shown %v, %s chosen", tabButton(r, "Fix-Up").Visible(), r.selected)
	}
	// Final Draft's: no PDF group; a save ends it
	w.showImported(filepath.Join(t.TempDir(), "old.fdx"), "INT. ROOM - DAY\n\nWords.\n", "Final Draft")
	w.ribbonRefresh()
	if got := dumpRibbon(r); strings.Contains(got, "## PDF") || !strings.Contains(got, "## Fix") {
		t.Errorf("Final Draft's Fix-Up:\n%s", got)
	}
	if err := w.saveToFile(filepath.Join(t.TempDir(), "old.fountain")); err != nil {
		t.Fatal(err)
	}
	if tabButton(r, "Fix-Up").Visible() {
		t.Error("still there after the save")
	}
}

func TestGarbledReport(t *testing.T) {
	if s := garbledReport("a.pdf", pdfPages{pages: 4}); !strings.Contains(s, "clean") {
		t.Error(s)
	}
	if s := garbledReport("a.pdf", pdfPages{pages: 4, scanned: 1, garbled: 2}); !strings.Contains(s, "1 scanned page") || !strings.Contains(s, "2 pages whose text looks garbled") {
		t.Error(s)
	}
}

// The Title Page tab: there while the cursor is in the title page, not
// in front by itself; a key added after the cursor's field (in the
// script's language's word), one there already found; To the Script.
func TestTitlePageTab(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, "Title: Rain\nAuthor: Ann\n\nINT. HOUSE - DAY\n\nRain.\n")
	r := ribbonOnFor(t, w)
	r.selectTab("Edit")
	w.textEditor.SetCursorOffset(3) // in Title
	w.ribbonRefresh()
	if !tabButton(r, "Title Page").Visible() || r.selected != "Edit" {
		t.Fatalf("in the title page: shown %v, %s chosen", tabButton(r, "Title Page").Visible(), r.selected)
	}
	w.addTitleKey("Draft date")
	text := w.textEditor.Text()
	if !strings.HasPrefix(text, "Title: Rain\nDraft date: \nAuthor: Ann\n") {
		t.Fatalf("the key added:\n%s", text)
	}
	if line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset()); line != 1 {
		t.Errorf("the cursor on line %d", line)
	}
	w.textEditor.SetCursorOffset(3)
	w.addTitleKey("Author") // there already
	if line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset()); line != 2 || w.textEditor.Text() != text {
		t.Errorf("Author: line %d", line)
	}
	w.addTitleKey("Contact")
	if !strings.Contains(w.textEditor.Text(), "Author: Ann\nContact:\n    \n\nINT.") {
		t.Errorf("Contact:\n%s", w.textEditor.Text())
	}
	w.ribbonRefresh()
	if !tabButton(r, "Title Page").Visible() {
		t.Error("the tab went on Contact's line")
	}
	r.selectTab("Title Page")
	w.toScript()
	w.ribbonRefresh()
	if tabButton(r, "Title Page").Visible() || r.selected != "Edit" {
		t.Errorf("in the script: shown %v, %s chosen", tabButton(r, "Title Page").Visible(), r.selected)
	}
	if !strings.HasPrefix(w.textEditor.Text()[w.textEditor.CursorOffset():], "INT. HOUSE") {
		t.Error("To the Script is not on the first scene")
	}
}

// In Arabic the keys are Lexington's Arabic ones, and the tab stays on a
// key just added (no value yet: no field to the title page's reading).
func TestTitlePageArabic(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, "العنوان: مطر\n\nداخلي. بيت - نهار\n\nمطر.\n")
	ribbonOnFor(t, w)
	w.setLanguage("ar")
	w.textEditor.SetCursorOffset(2)
	w.onCursorChanged()
	if !w.inTitlePage {
		t.Fatal("not in the title page")
	}
	w.addTitleKey("Notes")
	if !strings.Contains(w.textEditor.Text(), "ملاحظات:\n") {
		t.Fatalf("Notes in Arabic:\n%s", w.textEditor.Text())
	}
	w.onCursorChanged()
	if !w.inTitlePage {
		t.Error("the tab went on the new key")
	}
	if got := cmd(titleKeyID("Title")).ribbonLabelFor(w); got != "العنوان" {
		t.Errorf("Title's label %q", got)
	}
	_ = fyne.NewSize
}
