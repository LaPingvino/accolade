// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// Taking an issue back (LOOP.md, Joop 2026-11-20; PRODUCTION.md §10).
//
// An issue is one undo step (revisions.go's issueDraft joins the scene
// lock, the page lock and the revision line into one), tagged with an
// issueStep. Undo right after it takes the step's text back and the
// issue's entry out of the drafts store; Redo writes the text again and
// puts the entry back as it was. Production > Withdraw Last Issue… does
// the same later, to the newest issue only (a later issue is built on
// it): what the issue made, as its entry noted it (issueMade), is taken
// out of the production block where the block still holds it, in one
// undo step tagged the other way round: Undo issues it again.
//
// An entry taken out keeps its files in memory (withdrawnDraft), in the
// step's record, so Redo or Undo can put it back for as long as the undo
// history lasts: this session, this document (opening another forgets
// the history, and the records with it).

// issueRecord is one issue as the undo steps that issue or withdraw it
// know it.
type issueRecord struct {
	name    string // its set, for the status lines
	id      string // its entry in the drafts store ("" not written)
	inStore bool   // the entry is in the store now
	pending bool   // a store write for it is under way
	kept    *withdrawnDraft
}

// issueStep tags an undo step: one that issued (Undo withdraws), or one
// that withdrew (withdraw: Undo issues again).
type issueStep struct {
	rec      *issueRecord
	withdraw bool
}

// newIssueRecord is the record of an issue being written.
func (w *MainWindow) newIssueRecord(name string) *issueRecord {
	return &issueRecord{name: name, pending: true}
}

// errLaterIssue: an issue was made after the one to take back.
var errLaterIssue = errors.New("a later issue is built on it")

// onTaggedStep is the editor's OnTaggedStep: the drafts store follows an
// issue's step being undone or redone. Only a store write still under
// way keeps the step (try again in a moment); otherwise the text always
// goes, and the status line says when the store could not follow.
func (w *MainWindow) onTaggedStep(tag any, redo bool) bool {
	st, ok := tag.(*issueStep)
	if !ok {
		return true
	}
	rec := st.rec
	if rec.pending {
		w.setStatus(rec.name + " is still being written to the drafts store: try again in a moment")
		return false
	}
	out := redo == st.withdraw // the store loses it: undo an issue, redo a withdraw
	switch {
	case out && rec.inStore:
		if err := w.checkNewestIssue(rec.id); err != nil {
			w.setStatus(rec.name + " stays in the drafts store (" + err.Error() + "): only its text is taken back")
			return true
		}
		w.withdrawFromStore(rec, func(err error) {
			if err != nil {
				dialog.ShowError(fmt.Errorf("could not take %s out of the drafts store: %v", rec.name, err), w.fyneWindow)
				return
			}
			w.setStatus(rec.name + " taken back: no longer issued (Redo issues it again)")
		})
	case !out && !rec.inStore && rec.kept != nil:
		w.putBackInStore(rec, func(err error) {
			if err != nil {
				dialog.ShowError(fmt.Errorf("could not put %s back in the drafts store: %v", rec.name, err), w.fyneWindow)
				return
			}
			w.setStatus(rec.name + " issued again, as it was")
		})
	}
	return true
}

// checkNewestIssue tells why the store's issue id is not the newest one
// (nil if it is).
func (w *MainWindow) checkNewestIssue(id string) error {
	ds, err := readDrafts(w.extrasHome())
	if err != nil {
		return err
	}
	for i := len(ds) - 1; i >= 0; i-- {
		if ds[i].Kind != kindIssue {
			continue
		}
		if ds[i].ID == id {
			return nil
		}
		return fmt.Errorf("%s was issued after it", ds[i].Name)
	}
	return errors.New("it is not in the drafts store")
}

// withdrawFromStore takes rec's entry out of the drafts store (a bundle
// write in the background), keeping it in rec; done gets the error.
func (w *MainWindow) withdrawFromStore(rec *issueRecord, done func(error)) {
	finish := func(kept withdrawnDraft, err error) {
		rec.pending = false
		if err == nil {
			rec.kept, rec.inStore = &kept, false
		}
		w.revisionGen++
		w.updateRevisionMarks()
		w.refreshMenu()
		done(err)
	}
	rec.pending = true
	if w.bundle != nil {
		var kept withdrawnDraft
		w.queueBundle(bundleJob{
			run: func(b *textBundle) (*textBundle, error) {
				if b == nil {
					return nil, errors.New("the script's bundle is gone")
				}
				files, gone, k, err := withoutDraft(b, rec.id)
				if err != nil {
					return b, err
				}
				kept = k
				return saveBundleFilesWithout(b, files, gone, false)
			},
			done: func(err error) { finish(kept, err) },
		})
		return
	}
	home := w.extrasHome()
	if home.kind == extrasNone || home.kind == extrasBundle {
		finish(withdrawnDraft{}, errors.New("the script has no drafts store"))
		return
	}
	files, gone, kept, err := withoutDraft(home.dir, rec.id)
	if err == nil {
		err = home.dir.write(files)
	}
	if err == nil {
		err = home.dir.remove(gone)
	}
	finish(kept, err)
}

// putBackInStore puts rec's kept entry back in the drafts store, last;
// done gets the error.
func (w *MainWindow) putBackInStore(rec *issueRecord, done func(error)) {
	finish := func(d savedDraft, err error) {
		rec.pending = false
		if err == nil {
			rec.id, rec.inStore, rec.kept = d.ID, true, nil
		}
		w.revisionGen++
		w.updateRevisionMarks()
		w.refreshMenu()
		done(err)
	}
	k := *rec.kept
	rec.pending = true
	if w.bundle != nil {
		var d savedDraft
		w.queueBundle(bundleJob{
			run: func(b *textBundle) (*textBundle, error) {
				if b == nil {
					return nil, errors.New("the script's bundle is gone")
				}
				d0, files, err := draftBack(b, k)
				if err != nil {
					return b, err
				}
				d = d0
				return saveBundleFiles(b, files, false)
			},
			done: func(err error) { finish(d, err) },
		})
		return
	}
	home, err := w.ensureExtrasHome()
	if err != nil {
		finish(savedDraft{}, err)
		return
	}
	d, files, err := draftBack(home.dir, k)
	if err == nil {
		err = home.dir.write(files)
	}
	finish(d, err)
}

// withdrawal is what Withdraw Last Issue… would do: the issue, the
// production block's new inside and the scenes' numbers (one undo step,
// applyNumberPlan), and what it says.
type withdrawal struct {
	d        savedDraft
	heads    []sceneHead
	plan     numberPlan
	inside   string
	takes    []string // what goes, for the dialog
	stays    []string // what stays, and why
	previous string   // the issue before it ("" none)
}

// planWithdrawal is the withdrawal of the newest issue from the script as
// it is now, or why there is none.
func (w *MainWindow) planWithdrawal() (*withdrawal, error) {
	home := w.extrasHome()
	ds, err := readDrafts(home)
	if err != nil {
		return nil, err
	}
	var issues []savedDraft
	for _, d := range ds {
		if d.Kind == kindIssue {
			issues = append(issues, d)
		}
	}
	if len(issues) == 0 {
		return nil, errors.New("nothing has been issued")
	}
	d := issues[len(issues)-1]
	var imp importedRevisions
	if data, err := home.readExtra(revisionsName); err == nil && json.Unmarshal(data, &imp) == nil &&
		imp.Imported != nil && imp.Imported.Draft == d.ID {
		return nil, fmt.Errorf("%s came in from %s with its revisions: it is the imported script's own record, not an issue made here", d.Name, imp.Imported.From)
	}
	data, err := home.readExtra(d.Text)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %v", d.Name, err)
	}
	issued, _ := decodeScript(data)
	text := w.textEditor.Text()
	inside, _ := productionBlock(text)
	issuedInside, _ := productionBlock(issued)
	wd := &withdrawal{d: d}
	if len(issues) > 1 {
		wd.previous = issues[len(issues)-2].Name
	}

	// the revision line: back to the one before, if it is still the issue's
	if cur := currentRevisionOf(text); cur != "" && cur == currentRevisionOf(issued) {
		before := ""
		if d.Made != nil {
			before = d.Made.Revision
		} else if len(issues) > 1 {
			if data, err := home.readExtra(issues[len(issues)-2].Text); err == nil {
				prev, _ := decodeScript(data)
				before = currentRevisionOf(prev)
			}
		}
		if before == "" {
			inside = withoutBlockLine(inside, revisionLineKey)
			wd.takes = append(wd.takes, "the revision line ("+cur+") goes")
		} else {
			inside = withBlockLine(inside, revisionLineKey, before)
			wd.takes = append(wd.takes, "the revision line goes back to "+before)
		}
	} else if cur != "" {
		wd.stays = append(wd.stays, "the revision line ("+cur+") was changed since: it stays")
	}

	// the pages and the scene numbers: what the issue made, as it noted it
	curLock, issLock := pageLockIn(inside), pageLockIn(issuedInside)
	pagesBack := false
	switch {
	case d.Made == nil:
		wd.stays = append(wd.stays, d.Name+" was issued before Accolade noted what an issue locks: the scene numbers and the pages stay as they are (Unlock Scene Numbers… and Unlock Pages… take them back)")
	case d.Made.PageLock && curLock != nil && issLock != nil && curLock.text() == issLock.text():
		pagesBack = true
		if d.Made.PagesBefore == "" {
			wd.takes = append(wd.takes, "the page lock "+d.Name+" made is taken out")
		} else {
			wd.takes = append(wd.takes, "the pages are locked again as they were before "+d.Name)
		}
	case d.Made.PageLock && curLock != nil:
		wd.stays = append(wd.stays, "the locked pages changed since "+d.Name+": they stay (Unlock Pages… takes them out)")
	}
	wd.heads = sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())
	wd.plan = numberPlan{nums: make([]string, len(wd.heads)), drop: make([]bool, len(wd.heads))}
	for i, h := range wd.heads {
		wd.plan.nums[i] = h.num
	}
	if d.Made != nil && d.Made.SceneLock && sceneLockOf(text).locked {
		keep, _ := handNumbersOf(text)
		wd.plan = removePlan(wd.heads, keep)
		inside = pageLockFollowing(text, withoutBlockLine(withSceneLock(inside, nil), handNumbersKey), wd.heads, wd.plan)
		gone := 0
		for i, h := range wd.heads {
			if h.num != "" && wd.plan.nums[i] == "" {
				gone++
			}
		}
		wd.takes = append(wd.takes, "the scene numbers are unlocked, and the "+plural(gone, "number")+" the lock gave go")
	}
	if pagesBack {
		inside = withPageLock(inside, pageLockIn(d.Made.PagesBefore))
	}
	wd.inside = inside
	return wd, nil
}

// withdrawLastIssue is Production > Withdraw Last Issue…: asks, then
// takes the newest issue back.
func (w *MainWindow) withdrawLastIssue() {
	if w.markdown || w.textEditor == nil {
		return
	}
	w.afterBundleWrites(func() {
		wd, err := w.planWithdrawal()
		if err != nil {
			w.setStatus("No issue to withdraw: " + err.Error())
			return
		}
		w.withdrawDialog(wd).Show()
	})
}

// withdrawDialog is the Withdraw dialog (not shown).
func (w *MainWindow) withdrawDialog(wd *withdrawal) dialog.Dialog {
	d := wd.d
	when := d.Issued
	if t := issueDate(d); !t.IsZero() {
		when = t.Format("2 Jan 2006")
	}
	var msg strings.Builder
	fmt.Fprintf(&msg, "%s, issued %s, is taken back as if it had not been issued: it leaves the drafts store", d.Name, when)
	for _, t := range wd.takes {
		msg.WriteString("; " + t)
	}
	msg.WriteString(".")
	if wd.previous != "" {
		fmt.Fprintf(&msg, " The changes since %s are the open set again, marked %s; the next issue is %s.", wd.previous, d.Name, d.Name)
	} else {
		msg.WriteString(" Nothing is issued then, and the script has no revision marks.")
	}
	for _, s := range wd.stays {
		msg.WriteString("\n\n" + strings.ToUpper(s[:1]) + s[1:] + ".")
	}
	fmt.Fprintf(&msg, "\n\nIf %s's PDF or FDX has already gone out, withdrawing it here does not call those back: the pages and files people have still say %s, and the next issue will be %s too, with other changes.", d.Name, d.Name, d.Name)
	msg.WriteString("\n\nUndo puts it back.")
	label := widget.NewLabel(msg.String())
	label.Wrapping = fyne.TextWrapWord
	var dl dialog.Dialog
	withdraw := widget.NewButton("Withdraw "+d.Name, func() { dl.Hide(); w.withdrawNow(wd) })
	withdraw.Importance = widget.DangerImportance
	cancel := widget.NewButton("Cancel", func() { dl.Hide() })
	body := container.NewVScroll(label)
	dl = dialog.NewCustomWithoutButtons("Withdraw "+d.Name, container.NewBorder(nil, w.dialogButtons(cancel, withdraw), nil, nil, body), w.fyneWindow)
	dl.Resize(fitDialog(w.fyneWindow, 540, 500))
	return dl
}

// withdrawNow takes the issue back: the production block and the numbers
// in one undo step, tagged, and its entry out of the drafts store.
func (w *MainWindow) withdrawNow(wd *withdrawal) {
	e := w.textEditor
	mark := e.UndoSteps()
	w.applyNumberPlan(wd.heads, wd.plan, wd.inside)
	rec := &issueRecord{name: wd.d.Name, id: wd.d.ID, inStore: true}
	if e.UndoSteps() > mark {
		e.JoinStepsSince(mark)
		e.TagLastStep(&issueStep{rec: rec, withdraw: true})
	}
	w.withdrawFromStore(rec, func(err error) {
		if err != nil {
			dialog.ShowError(fmt.Errorf("could not withdraw %s: %v", wd.d.Name, err), w.fyneWindow)
			return
		}
		w.setStatus(wd.d.Name + " withdrawn: no longer issued (Undo puts it back)")
	})
}
