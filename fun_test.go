// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every command has a line on Fun Things to Try, or is one of the plain
// ones (funQuiet): a new feature gets its line here. docs/help/fun.md
// (the web's) is what the registry makes:
//
//	UPDATE_HELP=1 go test -run TestFunPage .
func TestFunPage(t *testing.T) {
	for _, c := range funUnlined() {
		t.Errorf("%s (%s) has no line on Fun Things to Try: give it one in funLines (fun.go), or put it in funQuiet", c.id, c.label)
	}
	seen := map[string]bool{}
	for _, m := range funMoments {
		if len(funLines[m]) == 0 {
			t.Errorf("no lines for %s", m)
		}
		for _, l := range funLines[m] {
			if _, ok := commands().byID[l.run]; !ok {
				t.Errorf("%s: no command %s", m, l.run)
			}
			if seen[l.run] {
				t.Errorf("%s has two lines", l.run)
			}
			seen[l.run] = true
			for _, a := range l.also {
				if strings.HasSuffix(a, ".") {
					continue
				}
				if _, ok := commands().byID[a]; !ok {
					t.Errorf("%s's line: no command %s", l.run, a)
				}
			}
		}
	}
	if len(funLines) != len(funMoments) {
		t.Errorf("funLines has %d moments, funMoments %d", len(funLines), len(funMoments))
	}
	if funLines[funMoments[0]][0].run != funSamplePDF {
		t.Error("the PDF import is not the first fun thing")
	}
	page := funPage(false)
	first := strings.Index(page, "](cmd:")
	if first < 0 || !strings.Contains(page[:first], "Open any screenplay PDF and keep writing") {
		t.Error("the page does not start with the PDF line")
	}
	if strings.Contains(funPage(true), "cmd:") {
		t.Error("the web page has the app's links")
	}
	path := filepath.Join("docs", "help", "fun.md")
	want := funPage(true)
	if os.Getenv("UPDATE_HELP") != "" {
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != want {
		t.Errorf("%s is not what the registry makes: UPDATE_HELP=1 go test -run TestFunPage .", path)
	}
}

// A line's link does its command: the outline's shows the outline.
func TestFunRuns(t *testing.T) {
	w := newTestWindow(t, "INT. ROOM - DAY\n\nA line.\n")
	w.tryFun("view.outline")
	if !w.paneInSight("outline") {
		t.Error("the outline's line did not show the outline")
	}
}

// The first line's sample PDF: printed by Lexington, it reads back in
// as the sample's scenes.
func TestFunSamplePDF(t *testing.T) {
	w := sampleWindow(t, sampleAccolade)
	text, _ := w.writerVersion()
	out := filepath.Join(t.TempDir(), "sample.pdf")
	if err := exportAs(text, out, "pdf", w.writerJob(w.pageSetup(), "pdf", text, nil)); err != nil {
		t.Fatal(err)
	}
	back, _, err := importPDF(context.Background(), out, nil, "", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range strings.Split(text, "\n") {
		if h := strings.TrimSpace(l); strings.HasPrefix(h, "EXT.") || strings.HasPrefix(h, "INT.") {
			if !strings.Contains(back, strings.Fields(h)[1]) {
				t.Errorf("the PDF read back has no %q", h)
			}
		}
	}
}

// A new window says a PDF can be dropped on it, until it is written in.
func TestFunDropLine(t *testing.T) {
	w := newTestWindow(t, "INT. ROOM - DAY\n\nA line.\n")
	w.filledWith = w.textEditor.Text()
	w.hasChanges = false
	w.updateTitle()
	if !w.dropLine.Visible() {
		t.Error("no drop line in a window holding nothing of its own")
	}
	w.textEditor.SetText(w.textEditor.Text() + "More.\n")
	if w.dropLine.Visible() {
		t.Error("the drop line stays once written in")
	}
}
