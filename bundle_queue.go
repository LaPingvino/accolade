// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"path/filepath"

	"fyne.io/fyne/v2/dialog"
)

// Writing a zipped bundle means zipping it whole again (its drafts and
// its source too, copied through), which took 94–187 ms in the browser
// for a script with a novel beside it (docs/PERF-SOURCE.md, item 7).
// So every write into the window's bundle (Save, auto-save, the source's
// Done, Save Draft) is a job, run in the background one after another:
// each job starts from the bundle as the job before it left it, never
// from one captured when it was asked for, so a Save Draft right after
// an auto-save keeps both. The window's bundle becomes the job's result
// on the UI goroutine, unless the window has since moved on to another
// document (bundleGen). Under the tests' driver (backgroundAtOnce) a job
// runs at once.

// bundleJob is one write into the window's bundle: run writes b (nil for
// a new bundle) and gives the bundle as written; done (may be nil) gets
// its error on the UI goroutine, the window's bundle already updated.
type bundleJob struct {
	run  func(b *textBundle) (*textBundle, error)
	done func(err error)
}

// bundleQueue is a window's bundle jobs.
type bundleQueue struct {
	busy bool
	cur  *runningJob // the job under way
	jobs []bundleJob
	idle []func() // run once nothing is pending
}

// runningJob is the job under way: ran is closed once its writing is
// done (before its result reaches the UI goroutine), res is what it gave.
type runningJob struct {
	job     bundleJob
	gen     int
	ran     chan struct{}
	res     bundleResult
	flushed bool // finished by flushBundleWrites: its result is in
}

// setBundle makes b the window's bundle (nil: none), as a document is
// opened, made or flattened: a job still under way for the one before
// leaves it alone.
func (w *MainWindow) setBundle(b *textBundle) {
	w.bundle = b
	w.bundleGen++
	w.srcPending = nil
}

// queueBundle adds a job; it starts at once if none is running.
func (w *MainWindow) queueBundle(job bundleJob) {
	q := &w.bundleQ
	q.jobs = append(q.jobs, job)
	if !q.busy {
		w.nextBundleJob()
	}
}

// errStillSaving: a rename waits for the save under way.
var errStillSaving = errors.New("the script is still being saved: try again in a moment")

// bundleBusy tells whether a write into the bundle is pending.
func (w *MainWindow) bundleBusy() bool { return w.bundleQ.busy }

// afterBundleWrites runs f once every pending bundle write is done (at
// once if none is): what needs the file as saved (a download, a rename,
// closing the window).
func (w *MainWindow) afterBundleWrites(f func()) {
	if !w.bundleQ.busy {
		f()
		return
	}
	w.bundleQ.idle = append(w.bundleQ.idle, f)
}

type bundleResult struct {
	b   *textBundle
	err error
}

func (w *MainWindow) nextBundleJob() {
	q := &w.bundleQ
	if len(q.jobs) == 0 {
		q.busy = false
		idle := q.idle
		q.idle = nil
		for _, f := range idle {
			f()
		}
		return
	}
	job := q.jobs[0]
	q.jobs = q.jobs[1:]
	q.busy = true
	b, gen := w.bundle, w.bundleGen
	cur := &runningJob{job: job, gen: gen, ran: make(chan struct{})}
	q.cur = cur
	inBackground(func() bundleResult {
		nb, err := job.run(b)
		cur.res = bundleResult{nb, err}
		close(cur.ran)
		return cur.res
	}, func(r bundleResult) {
		if cur.flushed {
			return // flushBundleWrites took it in
		}
		q.cur = nil
		if r.err == nil && w.bundleGen == gen {
			w.bundle = r.b
		}
		if job.done != nil {
			job.done(r.err)
		}
		w.nextBundleJob()
	})
}

// flushBundleWrites finishes every pending write now, on the UI
// goroutine: before the app quits or a window closes from a command
// (:wq), when what runs in the background would not get to finish. The
// browser's page never quits that way: there it does nothing.
func (w *MainWindow) flushBundleWrites() {
	q := &w.bundleQ
	if !q.busy || inBrowser {
		return
	}
	if cur := q.cur; cur != nil {
		<-cur.ran
		cur.flushed = true
		q.cur = nil
		if cur.res.err == nil && w.bundleGen == cur.gen {
			w.bundle = cur.res.b
		}
		if cur.job.done != nil {
			cur.job.done(cur.res.err)
		}
	}
	for len(q.jobs) > 0 {
		job := q.jobs[0]
		q.jobs = q.jobs[1:]
		nb, err := job.run(w.bundle)
		if err == nil {
			w.bundle = nb
		}
		if job.done != nil {
			job.done(err)
		}
	}
	q.busy = false
	idle := q.idle
	q.idle = nil
	for _, f := range idle {
		f()
	}
}

// saveFailed is what a background save that failed does: the script is
// unsaved again, and says why.
func (w *MainWindow) saveFailed(p string, err error) {
	w.saveFailures++
	w.hasChanges = true
	w.updateTitle()
	if w.fyneWindow != nil {
		dialog.ShowError(fmt.Errorf("could not save %s: %v", filepath.Base(p), err), w.fyneWindow)
	}
}
