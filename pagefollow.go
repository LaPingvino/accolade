package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/pdf"

	"github.com/LaPingvino/accolade/internal/pace"
)

// A script's preview following its text (Follow Edits), as OpenLoco
// redraws: an edit invalidates the page it is on, which is laid out again
// at once from where it starts (as if the pages after it started where
// they did) and repainted, the pages after it dimmed — they may move; once
// the typing pauses, the pages are made again in the background from the
// page before (an edit at a page's top can take lines back to it) until a
// page starts where it did before, the rest kept as it is (pdf.Reflow):
// only the pages that look otherwise are painted again, the zoom and the
// place kept. Typing again cancels the background pass. The PDF itself is
// made again only when it is wanted (Save PDF…, Open in Viewer). Locked
// pages and revisions are drawn whole (Lexington does not start those
// from a page), as is the first page's.

// pagesFollowPause is the pause in typing after which the pages are made again.
var pagesFollowPause = 400 * time.Millisecond

// followPageHook, if set, is told as each page is made (the tests type
// meanwhile).
var followPageHook atomic.Pointer[func()]

// followEditsPref keeps Follow Edits (on by default).
const followEditsPref = "preview-follow-edits"

// followEdits reports whether the preview follows the edits.
func (w *MainWindow) followEdits() bool {
	return w.settings == nil || !w.settings.GetBoolean(followEditsPref+"-off")
}

// setFollowEdits turns Follow Edits on or off (and keeps it); turned on,
// the pages follow the text as it is now.
func (w *MainWindow) setFollowEdits(on bool) {
	if w.settings != nil {
		w.settings.SetBoolean(followEditsPref+"-off", !on)
		_ = w.settings.Save()
	}
	for _, v := range []*pageView{w.pages, w.mdPagesView()} {
		if v != nil {
			v.followC.SetChecked(on)
		}
	}
	if on {
		if v := w.previewView(); v != nil && v.stale {
			v.refresh()
		}
	}
	w.ribbonRefresh()
}

// pageFollow keeps a preview's pages following the script. Its fields are
// the UI goroutine's; the work in the background is given what it needs.
type pageFollow struct {
	w    *MainWindow
	v    *pageView
	base pdf.Drawing // the pages as made of baseText, all of them
	// baseText is the text they are of
	baseText string
	gen      int // the last edit's: work for an older one is dropped
	cancel   context.CancelFunc
	timer    *uiTimer
	busy     bool   // a page is being laid out at once
	next     func() // the one to lay out after it (the latest edit's)
	stats    followStats
}

// followStats are the measures of the last edit.
type followStats struct {
	began    time.Time
	redrawn  int           // the pages laid out at once
	reflowed int           // the pages laid out again in the background
	kept     bool          // the rest kept: a page started where it did
	shown    time.Duration // to the edited page shown
	settled  time.Duration // to the pages all made again
	painted  int           // the pages painted again for it (in sight)
}

func newPageFollow(w *MainWindow, v *pageView, text string, d pdf.Drawing) *pageFollow {
	f := &pageFollow{w: w, v: v, base: d, baseText: text}
	v.follow = f
	v.freshPDF = f.freshPDF
	return f
}

// changedLines are the lines of b that are not in a (from 1): from the
// first that differs to the last before the lines both end with.
func changedLines(a, b string) (int, int) {
	la, lb := strings.Split(a, "\n"), strings.Split(b, "\n")
	p := 0
	for p < len(la) && p < len(lb) && la[p] == lb[p] {
		p++
	}
	s := 0
	for s < len(la)-p && s < len(lb)-p && la[len(la)-1-s] == lb[len(lb)-1-s] {
		s++
	}
	return p + 1, max(p+1, len(lb)-s)
}

// drawer draws text as the job prints it, from a Resume (nil: whole);
// safe in the background (the job's and the text's only), where the text
// is read, once.
func drawer(text string, job exportJob) func(*pdf.Resume) (pdf.Drawing, error) {
	var once sync.Once
	var s lex.Screenplay
	return func(r *pdf.Resume) (d pdf.Drawing, err error) {
		defer func() { // as exportAs: a fault of Lexington's is an error, not the window gone
			if p := recover(); p != nil {
				err = fmt.Errorf("the pages could not be made: %v", p)
			}
		}()
		once.Do(func() {
			job = withLockedPages(text, "pdf", job)
			job.lines = true
			s = job.script(text)
		})
		o := job.options("pdf", "")
		o.Record = &d
		if r == nil {
			r = &pdf.Resume{}
		}
		stop := r.Stop
		r.Stop = func(st pdf.PageStart) bool {
			pace.YieldNow() // a page at a time: the browser's page takes input between
			if h := followPageHook.Load(); h != nil {
				(*h)()
			}
			return stop != nil && stop(st)
		}
		o.Resume = r
		err = cli.Write(context.Background(), s, o)
		return d, err
	}
}

// job is the preview's job for text, as Preview PDF makes it.
func (f *pageFollow) job(text string) exportJob {
	j := f.w.outputJob(f.w.pageSetup(), "pdf", text, nil)
	j.lines = true // the pages' lines
	return j
}

// edited follows an edit: the stale pages marked, the edited one laid out
// again at once, the rest after a pause.
func (f *pageFollow) edited(text string) {
	f.gen++
	gen := f.gen
	f.stop()
	f.v.setStale(text != f.baseText)
	if !f.w.followEdits() {
		return
	}
	f.stats = followStats{began: time.Now()}
	first, last := changedLines(f.baseText, text)
	draw := drawer(text, f.job(text))
	base := f.base
	f.redraw(func() {
		type got struct {
			d       pdf.Drawing
			from, n int
			err     error
		}
		inBackground(func() got {
			d, from, n, err := pdf.Redraw(base, first, last, draw)
			return got{d, from, n, err}
		}, func(g got) {
			f.done()
			if gen != f.gen || g.err != nil || f.v.follow != f {
				return
			}
			painted := f.v.painted
			f.v.setDrawing(g.d, g.from+g.n)
			f.stats.redrawn, f.stats.shown = g.n, time.Since(f.stats.began)
			f.stats.painted += f.v.painted - painted
		})
	})
	f.w.debounce(&f.timer, pagesFollowPause, func() { f.reflow(gen, text, first, draw) })
}

// redraw lays a page out at once, or after the one under way (the latest
// edit's only).
func (f *pageFollow) redraw(run func()) {
	if f.busy {
		f.next = run
		return
	}
	f.busy = true
	run()
}

// done: the page laid out at once is in; the next, if any, goes.
func (f *pageFollow) done() {
	f.busy = false
	if next := f.next; next != nil {
		f.next = nil
		f.redraw(next)
	}
}

// reflow makes the pages again in the background, from the page before
// the edit to where a page starts as before; typing again cancels it.
func (f *pageFollow) reflow(gen int, text string, first int, draw func(*pdf.Resume) (pdf.Drawing, error)) {
	if gen != f.gen {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	base := f.base
	type got struct {
		d   pdf.Drawing
		how pdf.Reflowed
		err error
	}
	inBackground(func() got {
		d, how, err := pdf.Reflow(ctx, base, first, draw)
		return got{d, how, err}
	}, func(g got) {
		cancel()
		if gen != f.gen || g.err != nil || f.v.follow != f {
			return
		}
		f.cancel = nil
		f.base, f.baseText = g.d, text
		painted := f.v.painted
		f.v.setDrawing(g.d, -1)
		f.v.setStale(false)
		f.v.pdfStale = true
		f.stats.reflowed, f.stats.kept, f.stats.settled = g.how.Drawn, g.how.Kept, time.Since(f.stats.began)
		f.stats.painted += f.v.painted - painted
	})
}

// stop cancels the work under way for an older edit.
func (f *pageFollow) stop() {
	cancel(&f.timer)
	if f.cancel != nil {
		f.cancel()
		f.cancel = nil
	}
	f.next = nil
}

// refresh makes the pages again, all of them, and the PDF (Refresh).
func (f *pageFollow) refresh() {
	f.stop()
	f.gen++
	f.w.pdfOrPrint(f.w.textEditor.Text(), f.w.makePreviewPDF)
}

// freshPDF makes the PDF of the pages shown (the text as it is), then
// runs then: Save PDF… and Open in Viewer after the pages were made again.
func (f *pageFollow) freshPDF(then func()) {
	w, v := f.w, f.v
	text := w.textEditor.Text()
	job := f.job(text)
	dir, err := previewTemp()
	if err != nil {
		w.showPDFError(err)
		return
	}
	out := filepath.Join(dir, v.name)
	w.setStatus("Preparing the PDF...")
	gen := f.gen
	type got struct {
		d   pdf.Drawing
		err error
	}
	inBackground(func() got {
		d, _, err := drawScript(text, out, job)
		return got{d, err}
	}, func(g got) {
		w.setStatus("")
		if g.err != nil {
			os.RemoveAll(dir)
			w.showPDFError(g.err)
			return
		}
		if w.previewDir != "" && w.previewDir != dir {
			os.RemoveAll(w.previewDir)
		}
		w.previewDir = dir
		v.pdfPath, v.pdfStale = out, false
		if gen == f.gen && f.v.follow == f && text == w.textEditor.Text() {
			f.base, f.baseText = g.d, text
			v.setDrawing(g.d, -1)
			v.setStale(false)
		}
		then()
	})
}

// followText follows an edit of the script in the preview's pages, if
// they follow it.
func (w *MainWindow) followText(text string) {
	if w.pagesVisible && w.pages != nil && w.pages.follow != nil {
		w.pages.follow.edited(text)
	}
}
