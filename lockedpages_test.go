package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/pdfin"
)

// pagesScript is some six pages: scenes of action and a speech, each
// paragraph with words of its own.
func pagesScript() string {
	var b strings.Builder
	b.WriteString("Title: Locked\nAuthor: Someone\n\n")
	for k := 1; k <= 10; k++ {
		fmt.Fprintf(&b, "INT. ROOM %d - DAY\n\n", k)
		for j := 1; j <= 3; j++ {
			fmt.Fprintf(&b, "R%[1]dP%[2]d Anna walks along the wall %[1]d%[2]da and counts the windows %[1]d%[2]db, "+
				"one by one, as if each %[1]d%[2]dc could tell her something new about the house %[1]d%[2]dd.\n\n", k, j)
		}
		fmt.Fprintf(&b, "ANNA\nR%[1]dD This is room %[1]d. I remember the light %[1]da. We used to run here %[1]db. "+
			"Then the rain %[1]dc started and we stayed.\n\n", k)
	}
	return b.String()
}

// lockNow locks the window's pages and waits for it.
func lockNow(t *testing.T, w *MainWindow) *pageLock {
	t.Helper()
	w.lockPages()
	waitFor(t, "the pages locked", func() bool { return pageLockOf(w.textEditor.Text()) != nil })
	return pageLockOf(w.textEditor.Text())
}

// pdfLabels are the page numbers a PDF prints (its pages' first lines
// ending in "n."), the first page after the title page's none.
func pdfLabels(t *testing.T, w *MainWindow, job exportJob) []string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "script.pdf")
	if err := exportAs(w.textEditor.Text(), file, "pdf", job); err != nil {
		t.Fatal(err)
	}
	pages, err := pdfin.Pages(file)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, p := range pages[2:] { // after the title page and page 1
		f := strings.Fields(p[0].Text)
		out = append(out, strings.TrimSuffix(f[len(f)-1], "."))
	}
	return out
}

func TestPageLockBlock(t *testing.T) {
	inside := "scene numbers: locked, 10A\nrevision: Blue, 2026-11-20"
	l := &pageLock{lock: &lex.PageLock{Paper: "a4", Pages: []lex.LockedPage{
		{Label: "1", Words: "int room one day"},
		{Label: "1A", Scene: "2", Before: "so it goes", Words: "and then"},
	}}, format: "screenplay", pagination: "keep+widows"}
	got := withPageLock(inside, l)
	if !strings.HasPrefix(got, inside+"\npages: locked, a4, format screenplay, pagination keep+widows\n  1 -:  / int room one day\n  1A 2: so it goes / and then") {
		t.Fatalf("block:\n%s", got)
	}
	text := "INT. ROOM - DAY\n\nHello.\n\n/* Production\n" + got + "\n*/\n"
	back := pageLockOf(text)
	if back == nil || back.format != "screenplay" || back.pagination != "keep+widows" || back.lock.Paper != "a4" ||
		!slices.Equal(back.lock.Pages, l.lock.Pages) {
		t.Fatalf("read back %+v", back)
	}
	// again in its place; out with the other lines kept
	if again := withPageLock(got+"\nother: kept", l); strings.Count(again, "pages:") != 1 || !strings.HasSuffix(again, "other: kept") {
		t.Fatalf("set again:\n%s", again)
	}
	if out := withPageLock(got+"\nother: kept", nil); out != inside+"\nother: kept" {
		t.Fatalf("taken out:\n%s", out)
	}
}

// TestLockPagesUndo: Lock Pages writes the lock as one undo step, Unlock
// takes it out as one; the export prints A-pages for what grew.
func TestLockPagesUndo(t *testing.T) {
	w, _ := scriptWindow(t, pagesScript())
	before := w.textEditor.Text()
	l := lockNow(t, w)
	if n := len(l.lock.Pages); n < 4 || l.lock.Paper != "letter" {
		t.Fatalf("locked %d pages on %q", n, l.lock.Paper)
	}
	if !strings.Contains(w.textEditor.Text(), "/* Production\npages: locked, letter") {
		t.Fatalf("no lock in\n%s", w.textEditor.Text()[len(before)-50:])
	}
	w.undo()
	if w.textEditor.Text() != before {
		t.Fatalf("undo left\n%s", w.textEditor.Text()[len(before)-50:])
	}
	w.redo()
	if pageLockOf(w.textEditor.Text()) == nil {
		t.Fatal("redo: no lock")
	}
	want := pdfLabels(t, w, w.savedExportJob())
	if len(want) != len(l.lock.Pages)-1 {
		t.Fatalf("printed %v, locked %d", want, len(l.lock.Pages))
	}
	// text grows on page 2: 2A, the rest as locked
	start := l.lock.Pages[2].Words
	tok := ""
	for _, f := range strings.Fields(start) {
		if strings.HasPrefix(f, "r") && strings.ContainsAny(f, "pd") {
			tok = strings.ToUpper(f)
			break
		}
	}
	text := w.textEditor.Text()
	at := strings.Index(text, "\n\n"+tok[:2]) // the scene's first paragraph on page 3, or near it
	if at < 0 {
		t.Fatalf("no %s in %q", tok, start)
	}
	more := "\n\nGROWN Something new happens and goes on for a while in the room, with the windows and the doors " +
		"and the garden. GROWN again, the rain and the wind and the cold. GROWN at last.\n\nGROWN twice. And more."
	replaceText(t, w, text[at:at+4], more+text[at:at+4])
	got := pdfLabels(t, w, w.savedExportJob())
	if !slices.Contains(got, "2A") || len(got) != len(want)+1 {
		t.Fatalf("grown: %v (locked %v)", got, want)
	}
	// unlock: one undo step
	locked := w.textEditor.Text()
	w.writePageLock(nil, nil)
	if pageLockOf(w.textEditor.Text()) != nil || strings.Contains(w.textEditor.Text(), "/* Production") {
		t.Fatalf("unlocked:\n%s", w.textEditor.Text())
	}
	w.undo()
	if w.textEditor.Text() != locked {
		t.Fatal("undo of unlock")
	}
}

// TestIssueLocksPages: the first revision's Issue locks the pages as White
// printed them (the changes are A-pages), the next locks them again with
// their A-pages; FDX carries them.
func TestIssueLocksPages(t *testing.T) {
	w, _ := scriptWindow(t, pagesScript())
	issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	white := pdfLabels(t, w, w.savedExportJob())
	// grow scene 3's second paragraph a great deal
	replaceText(t, w, "R3P2 ", "R3P2 "+strings.Repeat("Blue words go on and on here. ", 40))
	chain, err := w.revisionChainNow()
	if err != nil {
		t.Fatal(err)
	}
	done := false
	w.issueDraft(lex.RevisionSet{Name: "Blue", Date: time.Date(2026, 11, 20, 0, 0, 0, 0, time.Local)},
		strings.Join(chain.issues[0].lines, "\n"), func(err error) {
			if err != nil {
				t.Error(err)
			}
			done = true
		})
	waitFor(t, "Blue issued", func() bool { return done })
	l := pageLockOf(w.textEditor.Text())
	if l == nil || len(l.lock.Pages) != len(white)+1 {
		t.Fatalf("lock %v, White's pages %v", l, white)
	}
	blue := pdfLabels(t, w, w.savedExportJob())
	var a string
	for _, p := range blue {
		if strings.HasSuffix(p, "A") {
			a = p
		}
	}
	if a == "" || len(blue) != len(white)+1 {
		t.Fatalf("Blue prints %v, White %v", blue, white)
	}
	// revised pages only: the changed page and its A-page
	job := w.withRevisions(w.savedExportJob(), w.textEditor.Text(), true, false)
	only, n := pdfLabelsAll(t, w, job)
	base := strings.TrimSuffix(a, "A")
	if want := []string{base, a}; base == "1" && n == 2 { // (page 1 prints no number)
		want = want[1:]
		if !slices.Equal(only, want) {
			t.Fatalf("revised pages only %v, want %v", only, want)
		}
	} else if !slices.Equal(only, want) {
		t.Fatalf("revised pages only %v (%d pages), want %v", only, n, want)
	}
	// Pink: locked again, the A-page kept
	replaceText(t, w, "R8P1 ", "R8P1 Pink. ")
	done = false
	w.issueDraft(lex.RevisionSet{Name: "Pink", Date: time.Date(2026, 11, 25, 0, 0, 0, 0, time.Local)}, "", func(err error) {
		if err != nil {
			t.Error(err)
		}
		done = true
	})
	waitFor(t, "Pink issued", func() bool { return done })
	again := pageLockOf(w.textEditor.Text())
	var labels []string
	for _, p := range again.lock.Pages {
		labels = append(labels, p.Label)
	}
	if !slices.Contains(labels, a) {
		t.Fatalf("locked again: %v", labels)
	}
	// FDX: the locked pages
	fdxFile := filepath.Join(t.TempDir(), "script.fdx")
	if err := exportAs(w.textEditor.Text(), fdxFile, "fdx", w.savedExportJob()); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(fdxFile)
	if !strings.Contains(string(data), `PageNumber="`+a+`"`) {
		t.Fatalf("no page %s in the FDX's LockedPages", a)
	}
	// and back: the import keeps them in its production block
	text, _, _, err := importFDXRevised(strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	back := pageLockOf(text)
	if back == nil {
		t.Fatalf("no lock in the import:\n%s", text[max(0, len(text)-400):])
	}
	var backLabels []string
	for _, p := range back.lock.Pages {
		backLabels = append(backLabels, p.Label)
	}
	if !slices.Equal(backLabels, labels) {
		t.Fatalf("imported lock %v, exported %v", backLabels, labels)
	}
}

// pdfLabelsAll are the page numbers of every page of the PDF (those with
// one: none on the first page), and how many pages it has.
func pdfLabelsAll(t *testing.T, w *MainWindow, job exportJob) ([]string, int) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "only.pdf")
	if err := exportAs(w.textEditor.Text(), file, "pdf", job); err != nil {
		t.Fatal(err)
	}
	pages, err := pdfin.Pages(file)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, p := range pages {
		f := strings.Fields(p[0].Text)
		if last := f[len(f)-1]; strings.HasSuffix(last, ".") {
			out = append(out, strings.TrimSuffix(last, "."))
		}
	}
	return out, len(pages)
}
