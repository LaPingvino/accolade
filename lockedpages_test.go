// These tests run the real background work; they hold the UI turn
// (holdUI, waitfor_test.go), the results coming in between their steps.

package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"

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
	holdUI(t)
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
	holdUI(t)
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
	want := pdfLabels(t, w, w.layoutJob())
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
	got := pdfLabels(t, w, w.layoutJob())
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

// issueWait issues set s (lockFrom: the offered first lock's text) and
// waits for it.
func issueWait(t *testing.T, w *MainWindow, name string, day int, lockFrom string) {
	t.Helper()
	done := false
	w.issueDraft(lex.RevisionSet{Name: name, Date: time.Date(2026, 11, day, 0, 0, 0, 0, time.Local)}, lockFrom, false,
		func(err error) {
			if err != nil {
				t.Error(err)
			}
			done = true
		})
	waitFor(t, name+" issued", func() bool { return done })
}

// pdfJob is the PDF export as Preview PDF makes it: the revisions, the
// pages the revision prints against; only: revised pages only.
func pdfJob(w *MainWindow, only bool) exportJob {
	text := w.textEditor.Text()
	return w.withIssuedPages(w.withRevisions(w.layoutJob(), text, only, false), text)
}

// revisedOnly are the page numbers revised pages only prints (page 1
// prints none: "1" for it).
func revisedOnly(t *testing.T, w *MainWindow) []string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "only.pdf")
	if err := exportAs(w.textEditor.Text(), file, "pdf", pdfJob(w, true)); err != nil {
		t.Fatal(err)
	}
	pages, err := pdfin.Pages(file)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, p := range pages {
		f := strings.Fields(p[0].Text)
		if last := f[len(f)-1]; strings.HasSuffix(last, ".") && strings.Contains(p[0].Text, "Revision") {
			out = append(out, strings.TrimSuffix(last, "."))
		} else {
			out = append(out, "1")
		}
	}
	return out
}

func lockLabelsOf(t *testing.T, w *MainWindow) []string {
	t.Helper()
	l := pageLockOf(w.textEditor.Text())
	if l == nil {
		t.Fatal("not locked")
	}
	var out []string
	for _, p := range l.lock.Pages {
		out = append(out, p.Label)
	}
	return out
}

// pageStartToken is the text in the script that the locked page's first
// row starts with: its paragraph's token (R3P2), its heading, or its
// speech's first line's token for a cue.
func pageStartToken(t *testing.T, w *MainWindow, page int) string {
	t.Helper()
	l := pageLockOf(w.textEditor.Text())
	rows := lex.Printable(w.layoutJob().script(w.textEditor.Text()), nil)
	at := lex.LocatePages(rows, l.lock)[page]
	if at == lex.Missing {
		t.Fatalf("page %d not found", page)
	}
	r := at.Row
	switch rows[r].Type {
	case lex.TypeScene:
		return rows[r].Contents
	case lex.TypeSpeaker:
		r++
	}
	f := strings.Fields(rows[r].Contents)
	if len(f) == 0 || !strings.HasPrefix(f[0], "R") {
		t.Fatalf("page %d starts in %q", page, rows[r].Contents)
	}
	return f[0]
}

// TestIssueLocksPages: the first revision's Issue locks the pages as White
// printed them, so Blue prints its changes on A-pages; the block then
// holds Blue's pages (Blue's A-page stays, what grows before it is aA);
// Pink locks again; FDX carries them, and the import keeps them.
func TestIssueLocksPages(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, pagesScript())
	issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	white := pdfLabels(t, w, w.layoutJob())
	// grow scene 3's second paragraph a great deal
	replaceText(t, w, "R3P2 ", "R3P2 "+strings.Repeat("Blue words go on and on here. ", 40))
	chain, err := w.revisionChainNow()
	if err != nil {
		t.Fatal(err)
	}
	issueWait(t, w, "Blue", 20, strings.Join(chain.issues[0].lines, "\n"))
	labels := lockLabelsOf(t, w)
	if len(labels) != len(white)+2 { // (page 1 is not in white)
		t.Fatalf("lock %v, White's pages %v", labels, white)
	}
	blue := pdfLabels(t, w, pdfJob(w, false))
	a := ""
	for _, p := range blue {
		if strings.HasSuffix(p, "A") {
			a = p
		}
	}
	if a == "" || len(blue) != len(white)+1 || !slices.Contains(labels, a) {
		t.Fatalf("Blue prints %v, White %v, locked %v", blue, white, labels)
	}
	base := strings.TrimSuffix(a, "A")
	if got := revisedOnly(t, w); !slices.Equal(got, []string{base, a}) {
		t.Fatalf("revised pages only %v, want %s and %s", got, base, a)
	}
	// Pink grows the same page before Blue's text: Blue's A-page keeps
	// its number, the new page is between
	replaceText(t, w, "R3P1 ", "R3P1 "+strings.Repeat("Pink words come first. ", 30))
	pink := pdfLabels(t, w, pdfJob(w, false))
	if !slices.Contains(pink, a) || !slices.Contains(pink, base+"aA") {
		t.Fatalf("Pink prints %v", pink)
	}
	issueWait(t, w, "Pink", 25, "")
	if got := lockLabelsOf(t, w); !slices.Contains(got, a) || !slices.Contains(got, base+"aA") {
		t.Fatalf("locked again: %v", got)
	}
	// Pink as issued prints against Blue's pages: its new page revised
	if got := revisedOnly(t, w); !slices.Contains(got, base+"aA") {
		t.Fatalf("Pink's revised pages %v", got)
	}
	// FDX: the locked pages
	fdxFile := filepath.Join(t.TempDir(), "script.fdx")
	if err := exportAs(w.textEditor.Text(), fdxFile, "fdx", w.layoutJob()); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(fdxFile)
	if !strings.Contains(string(data), `PageNumber="`+a+`"`) {
		t.Fatalf("no page %s in the FDX's LockedPages", a)
	}
	// and back: the import keeps them in its production block
	text, _, _, err := importFDXRevised(strings.NewReader(string(data)), langFor(nil))
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
	if want := lockLabelsOf(t, w); !slices.Equal(backLabels, want) {
		t.Fatalf("imported lock %v, exported %v", backLabels, want)
	}
}

// TestIssuePushedPage: locked before White, a short paragraph near a
// page's top pushes text onto an A-page with no change on it: Blue's
// revised pages are the page and its A-page; a page whose text is cut
// prints "n-m" in revised pages only.
func TestIssuePushedPage(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, pagesScript())
	lockNow(t, w)
	issueWait(t, w, "White", 1, "")
	labels := lockLabelsOf(t, w)
	// before the page's second paragraph
	text := w.textEditor.Text()
	at := strings.Index(text, pageStartToken(t, w, 2))
	next := at + strings.Index(text[at:], "\n\nR") + 2
	n := len([]rune(text[:next]))
	w.textEditor.Replace(n, n, "Blue's paragraph."+strings.Repeat(" Blue goes on.", 18)+"\n\n")
	issueWait(t, w, "Blue", 20, "")
	want := []string{labels[2], labels[2] + "A"}
	if got := revisedOnly(t, w); !slices.Equal(got, want) {
		t.Fatalf("revised pages only %v, want %v", got, want)
	}
	// Pink: a page's text cut whole (from its first paragraph to the next
	// page's)
	text = w.textEditor.Text()
	locked := lockLabelsOf(t, w)
	k := slices.Index(locked, "3")
	from := strings.Index(text, pageStartToken(t, w, k))
	to := strings.Index(text, pageStartToken(t, w, k+1))
	if from < 0 || to <= from {
		t.Fatal("pages 3 and the next not found")
	}
	cut := []rune(text[:from])
	w.textEditor.Replace(len(cut), len(cut)+len([]rune(text[from:to])), "")
	pink := pdfLabels(t, w, pdfJob(w, false))
	rng := locked[k-1] + "-3"
	if !slices.Contains(pink, rng) {
		t.Fatalf("Pink prints %v, want %s", pink, rng)
	}
	if got := revisedOnly(t, w); !slices.Contains(got, rng) {
		t.Fatalf("Pink's revised pages %v, want %s among them", got, rng)
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

// TestIssueOffersLock: the first revision's Issue dialog offers the lock;
// SHOT_DIR renders it wide and on a phone.
func TestIssueOffersLock(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, pagesScript())
	issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	replaceText(t, w, "R3P2 ", "R3P2 Blue words. ")
	menuItemByLabel(t, w, "Lock Pages")
	menuItemByLabel(t, w, "Unlock Pages…")
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"wide", fyne.NewSize(1100, 650)}, {"phone", fyne.NewSize(380, 800)}} {
		w.fyneWindow.Resize(size.size)
		w.showIssueDraft()
		if !dialogShows(w, "Lock the pages as White printed them") {
			t.Fatal("no offer to lock the pages")
		}
		if dir := os.Getenv("SHOT_DIR"); dir != "" {
			f, err := os.Create(filepath.Join(dir, "lockedpages-issue-"+size.name+".png"))
			if err != nil {
				t.Fatal(err)
			}
			png.Encode(f, w.fyneWindow.Canvas().Capture())
			f.Close()
		}
		for _, o := range w.fyneWindow.Canvas().Overlays().List() {
			w.fyneWindow.Canvas().Overlays().Remove(o)
		}
	}
}
