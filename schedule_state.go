package main

import (
	"bytes"
	"cmp"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2/dialog"

	"github.com/LaPingvino/lexington/lex"
	lexreport "github.com/LaPingvino/lexington/report"
)

// The schedule in the window (schedule.go has the file and the anchors):
// read once per script, written at each change (Undo on the stripboard
// takes the last change back; the text's Undo does not), and measured in
// the background once per text while the stripboard or the Schedule stage
// is shown: the report's eighths and cast for each scene, its strip's
// colour, and all issues' revision marks, for the change flags.

// scheduleState is the script's schedule and what it was measured with.
type scheduleState struct {
	key  string // what the file was read for (breakdownKey)
	file scheduleFile
	err  error
	undo *scheduleFile // the file before the last change on the board

	measuredFor string // the text measured
	pendingFor  string // the text being measured
	run         int
	rep         *lexreport.Report
	toReport    []int              // the text's scenes to the report's (-1 none)
	strips      []*lexreport.Strip // by the text's scene (nil: not measured)
	// lens are the scenes' lengths as last measured, by headingKeys, for
	// the timeline and its strip (kept while typing makes the measure
	// stale); lensRun counts the measures they came from
	lens      map[string]int
	lensRun   int
	marks     []string // all issues' marks, a line each
	set       string   // the current revision's set ("" before any issue)
	breakdown lexreport.Breakdown
}

// scheduleNow is the script's schedule, read again if need be.
func (w *MainWindow) scheduleNow() *scheduleFile {
	if key := w.breakdownKey(); w.schedule.key != key {
		f, err := readSchedule(w.extrasHome())
		// measured again too (another script, or a new issue's marks); a
		// measure under way is not taken
		w.schedule = scheduleState{key: key, file: f, err: err, run: w.schedule.run + 1, lensRun: w.schedule.lensRun + 1}
	}
	return &w.schedule.file
}

// scheduleFound is the schedule and its strips as found in the text now.
func (w *MainWindow) scheduleFound() (*scheduleFile, *bdText, schedPlaced) {
	f := w.scheduleNow()
	_, bt, _ := w.breakdownNowFound()
	return f, bt, f.place(bt)
}

// changeSchedule changes the schedule (change says whether it did) and
// writes it, keeping the one before for the board's Undo.
func (w *MainWindow) changeSchedule(change func(f *scheduleFile, bt *bdText, p schedPlaced) bool) bool {
	f, bt, p := w.scheduleFound()
	if err := w.schedule.err; err != nil { // never written over what could not be read
		dialog.ShowError(fmt.Errorf("the schedule could not be read, so it is not written over (%v): "+
			"mend or move %s first", err, scheduleName), w.fyneWindow)
		w.schedule.key = ""
		return false
	}
	before := cloneSchedule(*f)
	if !change(f, bt, p) {
		return false
	}
	f.prune(bt)
	w.schedule.undo = &before
	w.writeSchedule()
	return true
}

// undoSchedule takes the board's last change back.
func (w *MainWindow) undoSchedule() {
	if w.schedule.undo == nil {
		return
	}
	w.schedule.file, w.schedule.undo = *w.schedule.undo, nil
	w.writeSchedule()
	w.setStatus("Schedule: the last change undone")
}

// writeSchedule writes the schedule where the script's extras are, and
// shows it again.
func (w *MainWindow) writeSchedule() {
	f := w.scheduleNow()
	f.Version = 1
	if f.Days == nil {
		f.Days = []schedDay{}
	}
	if err := w.writeExtra(scheduleName, f); err != nil {
		dialog.ShowError(fmt.Errorf("could not keep the schedule: %v", err), w.fyneWindow)
	}
	w.schedule.key = w.breakdownKey() // a kept copy made for it: the same schedule
	w.scheduleFollow()
}

func cloneSchedule(f scheduleFile) scheduleFile {
	out := f
	out.Days = make([]schedDay, len(f.Days))
	for i, d := range f.Days {
		d.Strips = append([]schedStrip{}, d.Strips...)
		out.Days[i] = d
	}
	return out
}

// currentSet is the revision a strip put on a day now is of: the open
// set, or the last issued ("" before any issue).
func (w *MainWindow) currentSet() string {
	if w.schedule.set != "" {
		return w.schedule.set
	}
	return w.revisions.current.Name
}

// scheduleFollow shows the schedule again where it is shown: the board
// and the pane's stage.
func (w *MainWindow) scheduleFollow() {
	if w.stripsShown() {
		w.stripboard.update()
	}
	w.productionFollow()
}

// measureSchedule measures the text in the background (once a text):
// the reports' scenes, the strips, all issues' marks; then shows the
// schedule again.
func (w *MainWindow) measureSchedule() {
	w.scheduleNow() // read again first: that lets the measures go
	text := w.textEditor.Text()
	s := &w.schedule
	if sameString(s.measuredFor, text) || sameString(s.pendingFor, text) {
		return
	}
	s.run++
	run := s.run
	s.pendingFor = text
	job := w.layoutJob()
	f, bt, ps := w.breakdownNowFound()
	file := *f
	chain, _ := w.revisionChainNow()
	lang := w.language
	type made struct {
		rep      *lexreport.Report
		toReport []int
		strips   []*lexreport.Strip
		bd       lexreport.Breakdown
		marks    []string
		set      string
	}
	inBackground(func() made {
		var m made
		r, err := scriptReport(text, job)
		if err != nil {
			r = lexreport.Report{}
		}
		m.rep = &r
		m.toReport = reportScenes(bt, &r)
		m.bd = reportBreakdownFor(&file, ps, bt, &r)
		all := r.Strips(m.bd, lexreport.StripOptions{Lang: lang})
		m.strips = make([]*lexreport.Strip, len(bt.scenes))
		for i, k := range m.toReport {
			if k >= 0 && k < len(all) {
				m.strips[i] = &all[k]
			}
		}
		if chain != nil && len(chain.issues) > 0 {
			var cur lex.RevisionSet
			m.marks, _, cur = chain.marksOf(text, true, time.Now())
			m.set = cur.Name
		}
		return m
	}, func(m made) {
		if run != s.run {
			return
		}
		s.measuredFor, s.pendingFor = text, ""
		s.rep, s.toReport, s.strips, s.breakdown, s.marks, s.set = m.rep, m.toReport, m.strips, m.bd, m.marks, m.set
		heads := make([]int, len(bt.scenes))
		for i, sc := range bt.scenes {
			heads[i] = sc.line
		}
		s.lens = map[string]int{}
		for i, key := range headingKeys(strings.Split(text, "\n"), heads) {
			if i < len(m.strips) && m.strips[i] != nil && m.strips[i].Eighths > 0 {
				s.lens[key] = int(m.strips[i].Eighths)
			}
		}
		s.lensRun++
		w.scheduleFollow()
		w.storyMeasured()
	})
}

// measured tells whether the schedule's measures are of the text now.
func (w *MainWindow) scheduleMeasured() bool {
	return w.schedule.rep != nil && sameString(w.schedule.measuredFor, w.textEditor.Text())
}

// stripOfScene is a scene's strip as measured (nil: not yet).
func (w *MainWindow) stripOfScene(sc int) *lexreport.Strip {
	if !w.scheduleMeasured() || sc < 0 || sc >= len(w.schedule.strips) {
		return nil
	}
	return w.schedule.strips[sc]
}

// sceneEighths are the scenes' lengths as measured (nil: not yet).
func (w *MainWindow) sceneEighths() []lex.Eighths {
	if !w.scheduleMeasured() {
		return nil
	}
	out := make([]lex.Eighths, len(w.schedule.strips))
	for i, s := range w.schedule.strips {
		if s != nil {
			out[i] = s.Eighths
		}
	}
	return out
}

// strip flags: a scheduled scene changed since it was put on its day.
type stripFlag struct {
	changed bool
	in      string // the revision it changed in ("" none known)
}

func (w *MainWindow) flagOf(f *scheduleFile, bt *bdText, p schedPlaced, sc int) stripFlag {
	changed, since := f.changed(bt, p, sc)
	if !changed {
		return stripFlag{}
	}
	var in string
	if w.scheduleMeasured() {
		in = sceneChangedIn(bt, sc, w.schedule.marks, since)
	}
	return stripFlag{true, in}
}

// text is the flag as the strip says it.
func (s stripFlag) text() string {
	if s.in != "" {
		return "Changed in " + s.in + ": check"
	}
	return "Changed since scheduled: check"
}

// The schedule's exports: the one-liner and the DOOD (Production >
// Reports…), the shoot days as .ics.

// scheduleTitle is the script's name without its extension.
func (w *MainWindow) scheduleTitle() string {
	return strings.TrimSuffix(w.scriptName(), filepath.Ext(w.scriptName()))
}

// lexScheduleNow is the schedule as Lexington's reports take it, with the
// report it is of (measured now if need be).
func (w *MainWindow) lexScheduleFor(rep *lexreport.Report) lexreport.Schedule {
	f, bt, p := w.scheduleFound()
	return f.lexSchedule(p, reportScenes(bt, rep))
}

// exportICS saves the shoot days as an .ics file (the browser downloads
// it).
func (w *MainWindow) exportICS() {
	if w.markdown {
		return
	}
	content, job := w.textEditor.Text(), w.layoutJob()
	w.setStatus("Making the calendar...")
	inBackground(func() *lexreport.Report {
		r, err := scriptReport(content, job)
		if err != nil {
			return nil
		}
		return &r
	}, func(r *lexreport.Report) {
		if r == nil {
			dialog.ShowError(fmt.Errorf("could not measure the script for the calendar"), w.fyneWindow)
			return
		}
		data, skipped, err := w.icsOf(r)
		if err != nil {
			dialog.ShowError(err, w.fyneWindow)
			return
		}
		dated := len(w.scheduleNow().Days) - skipped
		if dated == 0 {
			dialog.ShowInformation("No Dated Shoot Days", "Give the shoot days their dates (the stripboard's day header) "+
				"to put them in a calendar.", w.fyneWindow)
			return
		}
		name := w.scheduleTitle() + " - shoot days.ics"
		w.saveNamedFile(name, func(p string) error {
			if err := writeFileAtomic(p, data); err != nil {
				return err
			}
			msg := fmt.Sprintf("Shoot days saved as %s: %s", filepath.Base(p), plural(dated, "day"))
			if skipped > 0 {
				msg += fmt.Sprintf(" (%s without a date left out)", plural(skipped, "day"))
			}
			w.setStatus(msg)
			return nil
		})
	})
}

// icsOf is the shoot days as iCalendar, and how many were left out (no
// date).
func (w *MainWindow) icsOf(r *lexreport.Report) ([]byte, int, error) {
	var b bytes.Buffer
	n, err := lexreport.WriteICS(&b, *r, w.lexScheduleFor(r), lexreport.ICSOptions{
		Title: w.scheduleTitle(), UIDDomain: "accolade.kiefte.eu"})
	return b.Bytes(), n, err
}

// dayLabel is a day's head: "Day 2 · Tue 1 Dec · Call 07:00".
func dayLabel(i int, d schedDay) string {
	parts := []string{fmt.Sprintf("Day %d", i+1)}
	if t := d.date(); !t.IsZero() {
		parts = append(parts, t.Format("Mon 2 Jan 2006"))
	}
	if d.Call != "" {
		parts = append(parts, "Call "+d.Call)
	}
	return strings.Join(parts, " · ")
}

// lengthText is a length as the board writes it: "4 6/8 pages", "3/8
// page".
func lengthText(e lex.Eighths) string {
	if e > 0 && e <= 8 {
		return e.String() + " page"
	}
	return e.String() + " pages"
}

// scheduleCountLine is the Schedule stage's count: "18 of 87 scenes on a
// day · 3 days".
func scheduleCountLine(c schedCount, flagged, lost int) string {
	parts := []string{fmt.Sprintf("%d of %s on a day", c.scheduled, plural(c.total, "scene")), plural(c.days, "day")}
	if c.total == 0 {
		parts[0] = "No scenes to schedule yet"
	}
	if flagged > 0 {
		parts = append(parts, plural(flagged, "changed scene")+" to check")
	}
	if lost > 0 {
		parts = append(parts, plural(lost, "lost strip"))
	}
	return strings.Join(parts, " · ")
}

// sceneName is a scene as the board names it: "12 INT. KITCHEN - NIGHT".
func sceneName(bt *bdText, sc int) string {
	s := bt.scenes[sc]
	return cmp.Or(s.label, fmt.Sprint(sc+1)) + " " + s.heading
}
