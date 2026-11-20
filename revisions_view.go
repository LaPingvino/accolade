package main

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/lex"
)

// Production > Issue Draft… (revisions.go has the marks).

// showIssueDraft asks for the issue's date and issues the script as the
// next set.
func (w *MainWindow) showIssueDraft() {
	if w.markdown || w.textEditor == nil {
		return
	}
	w.afterBundleWrites(func() {
		chain, err := w.revisionChainNow()
		if err != nil {
			dialog.ShowError(err, w.fyneWindow)
			return
		}
		set := chain.nextSet(time.Now())
		first := chain == nil || len(chain.issues) == 0
		date := widget.NewEntry()
		date.SetText(time.Now().Format("2006-01-02"))
		fyne.SetAccessibleLabel(date, "Date")
		var what string
		switch {
		case first:
			what = "Issue the script as it is now as the White draft: the production draft, kept with the script. " +
				"From now on each change is marked, in the next colour (" + lex.RevisionName(1) + "), until it is issued too."
		default:
			what = "Issue the script as it is now as the " + set.Name + " revision: the lines changed since " +
				chain.issues[len(chain.issues)-1].set.Name + " are its marks. The changes after it are " +
				chain.nextSetAfter(set).Name + "."
		}
		if first && !sceneLockOf(w.textEditor.Text()).locked {
			what += "\n\nIssuing White locks the scene numbers first (Production > Lock Scene Numbers)."
		}
		locked := pageLockOf(w.textEditor.Text()) != nil
		if locked {
			what += "\n\nThe locked pages are locked again with their A-pages, as " + set.Name + " prints them."
		}
		// the first revision: offer to lock the pages as the issue before
		// printed them, so that this one's changed pages are A-pages
		var lockCheck *widget.Check
		if !first && !locked {
			last := chain.issues[len(chain.issues)-1]
			lockCheck = widget.NewCheck("Lock the pages as "+last.set.Name+" printed them (pages that grew get A-pages: 12A)", nil)
			lockCheck.SetChecked(len(chain.issues) == 1)
		}
		text := widget.NewLabel(what)
		text.Wrapping = fyne.TextWrapWord
		where := widget.NewLabel(draftsWhere(w))
		where.Wrapping = fyne.TextWrapWord
		where.Importance = widget.LowImportance
		items := []*widget.FormItem{
			widget.NewFormItem("", text),
			widget.NewFormItem("Date", date),
		}
		if lockCheck != nil {
			items = append(items, widget.NewFormItem("", lockCheck))
		}
		items = append(items, widget.NewFormItem("", where))
		var d *dialog.FormDialog
		d = dialog.NewForm("Issue "+set.Name, "Issue", "Cancel", items, func(ok bool) {
			if !ok {
				return
			}
			day, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(date.Text), time.Local)
			if err != nil {
				dialog.ShowError(fmt.Errorf("the date is written as 2026-11-20"), w.fyneWindow)
				return
			}
			set.Date = day
			lockFrom := ""
			if lockCheck != nil && lockCheck.Checked {
				lockFrom = strings.Join(chain.issues[len(chain.issues)-1].lines, "\n")
			}
			w.issueDraft(set, lockFrom, func(err error) {
				if err != nil {
					dialog.ShowError(fmt.Errorf("could not issue %s: %v", set.Name, err), w.fyneWindow)
					return
				}
				w.setStatus(set.Name + " issued (" + day.Format("2 Jan 2006") + "): changes from now on are marked " +
					chain.nextSetAfter(set).Name)
			})
		}, w.fyneWindow)
		date.Validator = func(s string) error {
			_, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(s), time.Local)
			return err
		}
		d.Resize(fitDialog(w.fyneWindow, 480, 300))
		d.Show()
	})
}

// nextSetAfter is the set after s.
func (c *revisionChain) nextSetAfter(s lex.RevisionSet) lex.RevisionSet {
	n := lex.RevisionIndex(s.Name)
	if n < 0 {
		n = lex.RevisionIndex(lex.RevisionColor(s.Name))
	}
	return lex.RevisionSet{Name: lex.RevisionName(n + 1)}
}
