package main

import (
	"errors"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The stages (docs/LAYERS.md; LOOP.md, Joop 2026-11-20: "lock the
// writer's source, start adding on top"). Each stage locks what came
// before and adds layers on top, never editing it:
//
//   - the source, locked when adaptation starts: kept in the drafts store
//     as a draft of the kind "source" (the script as it was then, and the
//     source); the source as it is now is measured against it, passage by
//     passage, in the source pane ("changed since the script was
//     linked", Relink From Here);
//   - the writer's draft, locked at Production's Lock: the White issue
//     (revisions.go);
//   - the shooting script: the writer's draft and production's layers
//     (scene numbers, OMITTED, the later issues, the sidecars).
//
// The window's title names the stage; View > Stages… lists the locked
// ones, each with Compare; Export has the writer's version and the
// shooting version (stages_export.go).

// kindSource is a source lock's kind in the drafts store.
const kindSource = "source"

// storyStage is where a script is.
type storyStage int

const (
	stageNone     storyStage = iota // a Markdown document
	stageAtSource                   // a source, nothing adapted yet
	stageWriting                    // the writer's draft, open
	stageLocked                     // the writer's draft locked (White), unchanged since
	stageShooting                   // revisions after the lock
)

func (s storyStage) String() string {
	return [...]string{"", "Source", "Writing", "Locked", "Shooting"}[s]
}

// about is what the stage means, for View > Stages.
func (s storyStage) about() string {
	switch s {
	case stageAtSource:
		return "The source, nothing adapted yet: linking the first passage locks it."
	case stageWriting:
		return "The writer's draft, open: Production's Lock (Issue White) locks it."
	case stageLocked:
		return "The writer's draft is locked as White: changes from now on are revisions."
	case stageShooting:
		return "The shooting script: the writer's draft and production's revisions and layers on top."
	}
	return ""
}

// stageFacts are what the stage is told by.
type stageFacts struct {
	markdown     bool // a Markdown document: no stage
	hasSource    bool
	linked       bool // the script links its source
	sourceLocked bool
	issues       int
	// changed: the text differs from the last issue
	changed bool
}

// stageOf is the stage the facts tell.
func stageOf(f stageFacts) storyStage {
	switch {
	case f.markdown:
		return stageNone
	case f.issues == 0 && f.hasSource && !f.linked && !f.sourceLocked:
		return stageAtSource
	case f.issues == 0:
		return stageWriting
	case f.issues == 1 && !f.changed:
		return stageLocked
	}
	return stageShooting
}

// stagesState is the store as the stages last read it, for the script
// (key), and the lock Unlock Source took out.
type stagesState struct {
	key     string
	gen     int // a lock written or taken out: read again
	locks   []savedDraft
	lockSrc string // the newest lock's source
	err     error
	// unlocked is the lock Unlock Source took out (for unlockedKey's
	// script), which Lock Source puts back as it was
	unlocked    *issueRecord
	unlockedKey string
	shown       storyStage // in the title
}

// stagesKey is what the stages are read again for.
func (w *MainWindow) stagesKey() string {
	return fmt.Sprintf("%s|%d", w.revisionKey(), w.stages.gen)
}

// storeOf names the script's drafts store: its extras' home, the script.
func (w *MainWindow) storeOf() string {
	k := w.revisionKey()
	return k[:strings.LastIndexByte(k, '|')]
}

// readSourceLock is the source draft d holds: its source.md, or the
// Source block of its text (a source kept in the script).
func readSourceLock(r extrasReader, d savedDraft) (string, error) {
	if d.Source != "" {
		data, err := r.readExtra(d.Source)
		if err != nil {
			return "", err
		}
		md, _ := decodeScript(data)
		return md, nil
	}
	data, err := r.readExtra(d.Text)
	if err != nil {
		return "", err
	}
	text, _ := decodeScript(data)
	_, _, md, _ := sourceBlock(text)
	return md, nil
}

// sourceLocks are the script's source locks, oldest first, and the
// newest one's source; read again when the store changed.
func (w *MainWindow) sourceLocks() ([]savedDraft, string) {
	if w.markdown {
		return nil, ""
	}
	key := w.stagesKey()
	st := &w.stages
	if st.key == key {
		return st.locks, st.lockSrc
	}
	st.key, st.locks, st.lockSrc, st.err = key, nil, "", nil
	h := w.extrasHome()
	ds, err := readDrafts(h)
	if err != nil {
		st.err = err
		return nil, ""
	}
	for _, d := range ds {
		if d.Kind == kindSource {
			st.locks = append(st.locks, d)
		}
	}
	if n := len(st.locks); n > 0 {
		st.lockSrc, st.err = readSourceLock(h, st.locks[n-1])
	}
	return st.locks, st.lockSrc
}

// sourceLock is the newest source lock (nil none) and its source.
func (w *MainWindow) sourceLock() (*savedDraft, string) {
	locks, src := w.sourceLocks()
	if len(locks) == 0 {
		return nil, ""
	}
	return &locks[len(locks)-1], src
}

// stageFactsNow are the window's script's.
func (w *MainWindow) stageFactsNow() stageFacts {
	f := stageFacts{markdown: w.markdown || w.textEditor == nil}
	if f.markdown {
		return f
	}
	text := w.textEditor.Text()
	f.hasSource = w.sourceRef().Origin != sourceNone
	f.linked = strings.Contains(text[:bodyEndByte(text)], "[[from:")
	lock, _ := w.sourceLock()
	f.sourceLocked = lock != nil
	if c, err := w.revisionChainNow(); err == nil && c != nil && len(c.issues) > 0 {
		f.issues = len(c.issues)
		f.changed = !sameLines(bodyLines(text), c.issues[len(c.issues)-1].lines)
	}
	return f
}

// stageNow is the window's script's stage.
func (w *MainWindow) stageNow() storyStage { return stageOf(w.stageFactsNow()) }

// stageShown is the stage the title shows: none for a script with no
// source and nothing issued (writing, and nothing else to tell).
func (w *MainWindow) stageShown() storyStage {
	f := w.stageFactsNow()
	if !f.hasSource && !f.sourceLocked && f.issues == 0 {
		return stageNone
	}
	return stageOf(f)
}

// refreshStage shows the stage in the title if it changed (an issue, a
// lock, a link, typing after an issue: the revision marks' pause calls
// it, as the script is opened too). The title reads what it found.
func (w *MainWindow) refreshStage() {
	if w.fyneWindow == nil {
		return
	}
	if st := w.stageShown(); st != w.stages.shown {
		w.stages.shown = st
		w.updateTitle()
	}
}

// stageAfterStore follows a write to the drafts store: the stages read
// again, the title, the menus, the source pane's measure.
func (w *MainWindow) stageAfterStore() {
	w.stages.gen++
	w.refreshStage()
	w.refreshMenu()
	if s := w.source; s != nil && s.shown() { // measured from the lock now (not inside a follow that locked)
		w.debounce(&w.followTimer, followPause, s.follow)
	}
}

// canLockSource tells why the script's source cannot be locked ("" it can).
func (w *MainWindow) canLockSource() string {
	switch {
	case w.markdown:
		return "A Markdown document is a source itself"
	case w.sourceRef().Origin == sourceNone:
		return "No source to lock: Adaptation > Choose Source… first"
	case w.shared != nil && (w.shared.followOnly || !w.shared.own):
		return "The source is locked by the writer who shared the script"
	}
	return ""
}

// lockSource keeps the source as it is now as a locked source draft:
// the next "Source n" (the one Unlock Source took out, put back, if the
// source still reads as it did). auto: the first link of the session
// asked for it (the status line says so; a .highland not warned about
// Accolade's files yet is left, with a word on how). done (may be nil)
// gets whether a lock was written.
func (w *MainWindow) lockSource(auto bool, done func(bool)) {
	finish := func(ok bool) {
		if done != nil {
			done(ok)
		}
	}
	if why := w.canLockSource(); why != "" {
		if !auto {
			w.setStatus(why)
		}
		finish(false)
		return
	}
	src, err := w.sourceText(w.sourceRef())
	if err != nil {
		w.setStatus("The source could not be read: " + err.Error())
		finish(false)
		return
	}
	locks, lockSrc := w.sourceLocks()
	if auto && len(locks) > 0 { // locked when adapting started already
		finish(false)
		return
	}
	if len(locks) > 0 && sameString(lockSrc, src) {
		if !auto {
			w.setStatus("The source is locked as it is already: " + locks[len(locks)-1].Name)
		}
		finish(false)
		return
	}
	if auto && w.bundle != nil && w.warnsHighland(w.bundle.Path, []bundleFile{{draftsIndexName, nil}}) {
		w.setStatus("The source is not locked: Highland may drop Accolade's files from a .highland. " +
			"Adaptation > Lock Source keeps it (or save as .accolade first)")
		finish(false)
		return
	}
	st := &w.stages
	if rec := st.unlocked; rec != nil && st.unlockedKey == w.storeOf() && rec.kept != nil {
		if was, err := readSourceLock(keptFiles(rec.kept.files), rec.kept.d); err == nil && sameString(was, src) {
			w.putBackInStore(rec, func(err error) {
				if err != nil {
					dialog.ShowError(fmt.Errorf("could not lock the source: %v", err), w.fyneWindow)
					finish(false)
					return
				}
				st.unlocked = nil
				w.stageAfterStore()
				w.setStatus("The source is locked again: " + rec.name + " (Adaptation > Unlock Source takes it back)")
				finish(true)
			})
			return
		}
	}
	name := fmt.Sprintf("Source %d", len(locks)+1)
	w.addDraftLike(savedDraft{Name: name, Kind: kindSource}, false, func(d savedDraft, ok bool, err error) {
		if err != nil {
			dialog.ShowError(fmt.Errorf("could not lock the source: %v", err), w.fyneWindow)
			finish(false)
			return
		}
		st.unlocked = nil
		w.stageAfterStore()
		say := "The source is locked as " + d.Name
		if auto {
			say = "Adapting: the source is locked as adapted (" + d.Name + ")"
		}
		w.setStatus(say + ": editing it makes the next source draft; Adaptation > Unlock Source takes it back")
		finish(ok)
	})
}

// keptFiles reads a withdrawn draft's files.
type keptFiles []bundleFile

func (k keptFiles) readExtra(rel string) ([]byte, error) {
	for _, f := range k {
		if f.name == rel {
			return f.data, nil
		}
	}
	return nil, errors.New(rel + " is not in the draft")
}

// unlockSource takes the newest source lock out of the store, kept for
// the session: Lock Source puts it back as it was.
func (w *MainWindow) unlockSource() {
	lock, _ := w.sourceLock()
	if lock == nil {
		w.setStatus("The source is not locked")
		return
	}
	rec := &issueRecord{name: lock.Name, id: lock.ID, inStore: true}
	key := w.storeOf()
	w.withdrawFromStore(rec, func(err error) {
		if err != nil {
			dialog.ShowError(fmt.Errorf("could not unlock the source: %v", err), w.fyneWindow)
			return
		}
		w.stages.unlocked, w.stages.unlockedKey = rec, key
		w.stageAfterStore()
		w.setStatus(rec.name + " unlocked: no longer in the drafts store (Adaptation > Lock Source puts it back)")
	})
}

// relinkFromHere is Relink From Here: the links looked at from the
// locked source to the source as it is now (written where certain), and
// the source now locked as the next source draft, after asking.
func (w *MainWindow) relinkFromHere() {
	s := w.source
	if s == nil || w.markdown {
		return
	}
	s.load()
	n := len(s.changed)
	lock, lockSrc := w.sourceLock()
	if lock == nil { // measured from the session's first reading: locking is all
		w.lockSource(false, nil)
		return
	}
	if sameString(lockSrc, s.src) {
		w.setStatus("The source is as " + lock.Name + " has it: nothing to relink")
		return
	}
	msg := fmt.Sprintf("The source changed since the script was linked (%s): %s changed. "+
		"Relink From Here looks at the links again from %s to the source as it is now (written only where that is certain) "+
		"and locks the source as it is now as Source %d. %s stays in the drafts store, to compare.",
		lock.Name, plural(n, "linked passage"), lock.Name, len(w.stages.locks)+1, lock.Name)
	label := widget.NewLabel(msg)
	label.Wrapping = fyne.TextWrapWord
	compare := widget.NewButtonWithIcon("Compare the Source", res(compareIcon)(), func() { w.compareStage(*lock) })
	content := container.NewVBox(label, container.NewHBox(compare))
	d := dialog.NewCustomConfirm("Relink From Here", "Relink From Here", "Cancel", content, func(ok bool) {
		if !ok {
			return
		}
		w.relinkNow(lockSrc, nil)
	}, w.fyneWindow)
	d.Resize(fitDialog(w.fyneWindow, 480, 260))
	d.Show()
}

// relinkNow is Relink From Here, asked: the links looked at from the
// locked source (lockSrc) to the source now, then the source locked;
// done (may be nil) gets whether it was.
func (w *MainWindow) relinkNow(lockSrc string, done func(bool)) {
	s := w.source
	w.relinkInBackground(s.ref, lockSrc, s.src, s.passages, false, func(bool) {
		w.lockSource(false, done)
	})
}

// compareStage compares a locked stage with the script now, in the
// Compare middle tab: a source lock's source.
func (w *MainWindow) compareStage(d savedDraft) {
	w.afterBundleWrites(func() {
		ds, err := readDrafts(w.extrasHome())
		if err != nil {
			dialog.ShowError(err, w.fyneWindow)
			return
		}
		cv := showCompare(w, ds, d.ID, nowID)
		if d.Kind == kindSource && cv != nil && cv.what != nil {
			cv.what.SetSelected("Source")
		}
	})
}

// stageLabel is how View > Stages names a locked stage.
func stageLabel(d savedDraft, first bool) string {
	switch {
	case d.Kind == kindSource:
		return "Source locked: " + d.Name
	case first:
		return "Writer's draft locked: " + d.Name
	}
	return "Revision issued: " + d.Name
}

// lockedStages are the drafts that are locked stages, oldest first: the
// source locks and the issues.
func lockedStages(ds []savedDraft) []savedDraft {
	var out []savedDraft
	for _, d := range ds {
		if d.Kind == kindSource || d.Kind == kindIssue {
			out = append(out, d)
		}
	}
	return out
}

// showStages is View > Stages…: the stage, the locked stages with
// Compare, and what can be done next.
func (w *MainWindow) showStages() {
	if w.markdown {
		w.setStatus("A Markdown document has no stages: it is a source")
		return
	}
	w.afterBundleWrites(func() {
		ds, err := readDrafts(w.extrasHome())
		if err != nil {
			dialog.ShowError(err, w.fyneWindow)
			return
		}
		w.stagesDialog(ds).Show()
	})
}

// stagesDialog is View > Stages' dialog for the store's drafts ds.
func (w *MainWindow) stagesDialog(ds []savedDraft) dialog.Dialog {
	st := w.stageNow()
	head := widget.NewLabelWithStyle("Stage: "+st.String(), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	about := widget.NewLabel(st.about())
	about.Wrapping = fyne.TextWrapWord
	rows := container.NewVBox()
	var d dialog.Dialog
	issued := false
	stages := lockedStages(ds)
	for _, s := range stages {
		s := s
		first := s.Kind == kindIssue && !issued
		issued = issued || s.Kind == kindIssue
		name := widget.NewLabel(stageLabel(s, first))
		name.Wrapping = fyne.TextWrapWord
		when := widget.NewLabel(s.Date.Local().Format("2 Jan 2006 15:04"))
		when.SizeName = theme.SizeNameCaptionText
		when.Importance = widget.LowImportance
		cmp := widget.NewButtonWithIcon("Compare", res(compareIcon)(), func() {
			d.Hide()
			w.compareStage(s)
		})
		fyne.SetAccessibleLabel(cmp, "Compare "+s.Name+" with the script now")
		rows.Add(container.NewBorder(nil, nil, nil, cmp, container.NewVBox(name, when)))
	}
	if len(stages) == 0 {
		none := widget.NewLabel("Nothing is locked yet.")
		none.Importance = widget.LowImportance
		rows.Add(none)
	}
	act := func(id string) fyne.CanvasObject {
		b := paneButton(w, id)
		c := cmd(id)
		inner := b.OnTapped
		b.OnTapped = func() { d.Hide(); inner() }
		if c.enabled != nil && !c.enabled(w.cmdEnv()) {
			b.Disable()
		}
		return b
	}
	acts := container.NewVBox()
	if lock, _ := w.sourceLock(); lock != nil {
		acts.Add(act("source.unlock"))
	} else if w.sourceRef().Origin != sourceNone {
		acts.Add(act("source.lock"))
	}
	if !issued && !w.writerView() {
		b := act("production.issue").(*widget.Button)
		b.SetText("Lock the Writer's Draft (Issue White)…")
		acts.Add(b)
	}
	acts.Add(act("file.exportWriter"))
	acts.Add(act("file.exportShooting"))
	body := container.NewVBox(head, about, widget.NewSeparator(),
		caption("Locked, oldest first"), rows, widget.NewSeparator(), caption("Next"), acts)
	scroll := container.NewVScroll(body)
	d = dialog.NewCustom("Stages", "Close", container.NewBorder(nil, nil, nil, nil, scroll), w.fyneWindow)
	d.Resize(fitDialog(w.fyneWindow, 520, 520))
	return d
}

// The stages' commands.
func sourceLockable(e cmdEnv) bool   { return !e.markdown && e.source && !e.sourceLocked }
func sourceUnlockable(e cmdEnv) bool { return !e.markdown && e.sourceLocked }
func sourceRelinkable(e cmdEnv) bool { return !e.markdown && e.source && e.sourceChanged }

func (w *MainWindow) stagesEnv(e *cmdEnv) {
	if e.markdown || w.textEditor == nil {
		return
	}
	lock, _ := w.sourceLock()
	e.sourceLocked = lock != nil
	e.sourceChanged = w.source != nil && len(w.source.changed) > 0
}

func stageCommands() []*command {
	return []*command{
		{id: "view.stages", label: "Stages…", tip: "The script's locked stages: the source, the writer's draft, the issues", ex: "stages",
			icon: theme.HistoryIcon, when: screenplayOnly, run: (*MainWindow).showStages},
		{id: "source.lock", label: "Lock Source", tip: "Keep the source as it is now as a locked source draft", ex: "lock-source",
			icon: res(lockIcon), when: screenplayOnly, enabled: sourceLockable, run: func(w *MainWindow) { w.lockSource(false, nil) }},
		{id: "source.unlock", label: "Unlock Source", tip: "Take the newest source lock out of the drafts store", ex: "unlock-source",
			icon: res(unlockIcon), when: screenplayOnly, enabled: sourceUnlockable, run: (*MainWindow).unlockSource},
		{id: "source.relinkFromHere", label: "Relink From Here…", tip: "Link the script to the source as it is now, and lock that", ex: "relink-from-here",
			icon: theme.ViewRefreshIcon, when: screenplayOnly, enabled: sourceRelinkable, run: (*MainWindow).relinkFromHere},
		{id: "file.exportWriter", label: "Export Writer's Version…", tip: "The writer's draft as locked, without production's layers", ex: "export-writer",
			icon: theme.DownloadIcon, when: screenplayOnly, run: func(w *MainWindow) { w.showVersionExport(versionWriter) }},
		{id: "file.exportShooting", label: "Export Shooting Version…", tip: "The shooting script: scene numbers, OMITTED, revisions", ex: "export-shooting",
			icon: theme.DownloadIcon, when: screenplayOnly, run: func(w *MainWindow) { w.showVersionExport(versionShooting) }},
	}
}
