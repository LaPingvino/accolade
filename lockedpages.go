package main

import (
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/lex"
)

// Production, step 5 (docs/PRODUCTION.md §11): locked pages.
//
// Production > Lock Pages keeps the pages as the PDF prints them now (the
// export's settings): each page's number and where it starts, anchored by
// words (Lexington's lex.PageLock: the scene it starts in, its first
// words, the words before it), so an edit elsewhere, in any program, does
// not move it. The lock is in the production block, a line per page under
// its "pages:" line, which also holds the layout the pages were locked in
// (paper, CONTINUEDs, script format, pagination):
//
//	/* Production
//	scene numbers: locked, 10A
//	pages: locked, letter, format screenplay, pagination all
//	  1 -:  / int kitchen day anna stands at the window
//	  2 3: and the rain goes on / anna turns
//	*/
//
// The block is the lock's one home, for a bundle too: Undo takes a lock
// back like any edit, the lock survives a flattening to plain Fountain and
// an edit in another program, and the words it holds are a few kilobytes
// for a feature. (PRODUCTION.md §4 had planned pages.json in the bundle.)
//
// A locked script exports and previews in its lock's layout (whatever the
// Export dialog says of paper, format and pagination): text that grew goes
// onto A-pages (12A), a page whose text is gone gives its number to the
// one before (12-13), a page that shrank stays short. Revised pages only
// prints the changed pages with their A-pages. Lock Pages on a locked
// script locks again with the A-pages (Final Draft's Lock A Pages), as
// every Issue does; the first revision's Issue offers to lock the pages
// as the issue before it printed them. Unlock Pages… takes the lock out.
// Print (through HTML) does not keep locked pages; filters (tags, sides)
// leave them out too.

// continuedsPref is the setting of the scenes' CONTINUEDs in the PDF.
const continuedsPref = "export-continueds"

// pageLockOf is the script's page lock, nil for none.
func pageLockOf(text string) *pageLock {
	inside, ok := productionBlock(text)
	if !ok {
		return nil
	}
	l := lex.ParsePageLock(inside)
	if l == nil {
		return nil
	}
	pl := &pageLock{lock: l}
	if from, _, ok := lex.PageLockLines(inside); ok {
		_, value, _ := strings.Cut(strings.Split(inside, "\n")[from], ":")
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if f, ok := strings.CutPrefix(part, "format "); ok {
				pl.format = strings.TrimSpace(f)
			}
			if p, ok := strings.CutPrefix(part, "pagination "); ok {
				pl.pagination = strings.TrimSpace(p)
			}
		}
	}
	return pl
}

// pageLock is a script's locked pages, with the format and pagination
// they were locked in (Accolade's; Lexington's lock has paper and
// CONTINUEDs).
type pageLock struct {
	lock               *lex.PageLock
	format, pagination string
}

// text is the lock as the block's lines.
func (l pageLock) text() string {
	head, rest, _ := strings.Cut(l.lock.Text(), "\n")
	if l.format != "" {
		head += ", format " + l.format
	}
	if l.pagination != "" {
		head += ", pagination " + l.pagination
	}
	if rest == "" {
		return head
	}
	return head + "\n" + rest
}

// withPageLock is the block's inside with the lock's lines set (nil:
// taken out), in their place, else at the end; the other lines kept.
func withPageLock(inside string, l *pageLock) string {
	var lines []string
	if inside != "" {
		lines = strings.Split(inside, "\n")
	}
	var add []string
	if l != nil {
		add = strings.Split(l.text(), "\n")
	}
	if from, to, ok := lex.PageLockLines(inside); ok {
		lines = append(append(append([]string(nil), lines[:from]...), add...), lines[to:]...)
	} else {
		lines = append(lines, add...)
	}
	return strings.Trim(strings.Join(lines, "\n"), "\n")
}

// applyTo is an export job in the lock's layout, keeping its pages.
func (l pageLock) applyTo(j exportJob) exportJob {
	j.lock = l.lock
	j.continueds = l.lock.Continueds
	if l.format != "" {
		j.format = l.format
	}
	j.pagination = l.pagination
	return j
}

// withLockedPages is the job as it exports content: in its page lock's
// layout, for a PDF or an FDX of the whole script.
func withLockedPages(content, to string, j exportJob) exportJob {
	if j.lock != nil || (to != "pdf" && to != "fdx") || len(j.tags) > 0 || j.character != "" {
		return j
	}
	if l := pageLockOf(content); l != nil {
		return l.applyTo(j)
	}
	return j
}

// lockJob is the export the pages are locked by: the export's settings,
// a lock's own layout when there is one.
func (w *MainWindow) lockJob(text string) exportJob {
	j := w.savedExportJob()
	if l := pageLockOf(text); l != nil {
		j = l.applyTo(j)
	}
	return j
}

// lockedPagesOf locks the pages of body as job prints them: in the
// background, the lock given to done on the UI thread.
func lockedPagesOf(body string, job exportJob, done func(*pageLock, error)) {
	go func() {
		l, err := cli.LockPages(job.script(body), job.options("pdf", ""))
		fyne.Do(func() {
			if err != nil {
				done(nil, err)
				return
			}
			done(&pageLock{lock: &l, format: job.format, pagination: job.pagination}, nil)
		})
	}()
}

// writePageLock puts a lock into the production block (nil takes it
// out), as one undo step with extra (other lines of the block's inside:
// may be nil), the cursor kept.
func (w *MainWindow) writePageLock(l *pageLock, extra func(inside string) string) {
	e := w.textEditor
	text := e.Text()
	inside, _ := productionBlock(text)
	inside = withPageLock(inside, l)
	if extra != nil {
		inside = extra(inside)
	}
	start, end, block := withProductionBlock(text, inside)
	if strings.TrimSpace(inside) == "" {
		block = ""
		if end == utf8.RuneCountInString(text) {
			block = "\n"
		}
	}
	e.Edit(func(b *buffer.Buffer) {
		c := b.Cursor()
		b.Group(func() {
			b.Replace(start, end, block)
			b.SetCursor(min(c, utf8.RuneCountInString(b.Text())), false)
		})
	})
}

// lockPages locks the pages as the script prints now (again, with its
// A-pages, when locked).
func (w *MainWindow) lockPages() {
	if w.markdown || w.textEditor == nil {
		return
	}
	text := w.textEditor.Text()
	again := pageLockOf(text) != nil
	job := w.lockJob(text)
	w.setStatus("Locking the pages...")
	lockedPagesOf(text, job, func(l *pageLock, err error) {
		if err != nil {
			w.setStatus("")
			dialog.ShowError(err, w.fyneWindow)
			return
		}
		if w.textEditor.Text() != text {
			w.setStatus("The script changed while its pages were laid out: lock them again")
			return
		}
		w.writePageLock(l, nil)
		w.setStatus(lockedStatus(l, again))
	})
}

// lockedStatus says what a lock did.
func lockedStatus(l *pageLock, again bool) string {
	n := len(l.lock.Pages)
	paper := strings.ToUpper(l.lock.Paper[:1]) + l.lock.Paper[1:]
	if l.lock.Paper != "letter" {
		paper = strings.ToUpper(l.lock.Paper)
	}
	if again {
		return "Pages locked again: " + plural(n, "page") + " (their A-pages kept), on " + paper
	}
	return "Pages locked: " + plural(n, "page") + " on " + paper + ". Text that grows goes onto A-pages (12A); the export keeps this layout"
}

// unlockPages asks, then takes the page lock out.
func (w *MainWindow) unlockPages() {
	if w.markdown || w.textEditor == nil {
		return
	}
	if pageLockOf(w.textEditor.Text()) == nil {
		w.setStatus("The pages are not locked")
		return
	}
	dialog.ShowConfirm("Unlock Pages",
		"The pages are numbered as they fall again: A-pages go, and a page's text moves to where it fits.\nUnlock?",
		func(ok bool) {
			if ok {
				w.writePageLock(nil, nil)
				w.setStatus("Pages unlocked; they are numbered as they fall")
			}
		}, w.fyneWindow)
}
