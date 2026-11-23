package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Edit > Fix Lines Like This… (fixup.go): the pattern fixer's dialog. The
// line the cursor is on is the example; the pattern it fits is proposed in
// plain words (the regular expression under it, to write by hand), every
// line it matches is counted, lit in the script and listed with what it
// becomes; Apply makes the change in one undo step and keeps the rule in
// the script's /* Fix-Up */ block. Its saved rules can be applied again
// (after a re-import) from here too.

// fixPreviewMax is how many changes the preview lists.
const fixPreviewMax = 40

// fixRememberPref remembers the "keep the rule" box.
const fixRememberPref = "fixup-remember"

// fixupDlg is the dialog and its controls.
type fixupDlg struct {
	w        *MainWindow
	d        dialog.Dialog
	example  *widget.Entry
	pattern  *widget.Entry
	regex    *widget.Entry
	action   *widget.Select
	template *widget.Entry
	tmplRow  *fyne.Container
	count    *widget.Label
	preview  *fyne.Container
	remember *widget.Check
	apply    *widget.Button
	saved    *fyne.Container
	regexBox *widget.Accordion
	rule     fixRule
	changes  []fixChange
	filling  bool // the controls are being set from a rule
}

// showPatternFixer opens the pattern fixer on the cursor's line.
func (w *MainWindow) showPatternFixer() {
	if w.scriptAway("Fix Lines Like This") || w.markdown || w.textEditor == nil {
		return
	}
	f := w.newPatternFixer()
	f.d.Show()
	w.fyneWindow.Canvas().Focus(f.pattern)
}

// cursorLine is the script's line the cursor is on (the first line with
// text from there on, if it is blank).
func (w *MainWindow) cursorLine() string {
	b := w.textEditor.Buffer()
	line, _ := b.Position(b.Cursor())
	for i := line; i < b.LineCount() && i < line+20; i++ {
		if l := b.Line(i); strings.TrimSpace(l) != "" {
			return l
		}
	}
	return b.Line(line)
}

// newPatternFixer is the dialog for the cursor's line (not shown).
func (w *MainWindow) newPatternFixer() *fixupDlg {
	f := &fixupDlg{w: w}
	w.fixAsked = f
	bold := func(s string) fyne.CanvasObject {
		return widget.NewLabelWithStyle(s, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}
	low := func(s string) *widget.Label {
		l := widget.NewLabel(s)
		l.Wrapping = fyne.TextWrapWord
		l.Importance = widget.LowImportance
		return l
	}
	intro := widget.NewLabel("Every line like this one, set right at once: nothing changes until Apply, and one Undo takes it back.")
	intro.Wrapping = fyne.TextWrapWord

	f.example = widget.NewEntry()
	f.example.SetText(w.cursorLine())
	fyne.SetAccessibleLabel(f.example, "Line like this")
	f.example.OnChanged = func(string) { f.propose() }

	f.pattern = widget.NewEntry()
	fyne.SetAccessibleLabel(f.pattern, "Pattern")
	f.pattern.OnChanged = func(s string) {
		if f.filling {
			return
		}
		f.rule.Pattern, f.rule.Regex = s, ""
		f.showRegex()
		f.update()
	}
	f.regex = widget.NewEntry()
	fyne.SetAccessibleLabel(f.regex, "Regular expression")
	f.regex.OnChanged = func(s string) {
		if f.filling {
			return
		}
		if gen, _ := patternRegex(f.rule.Pattern); s == gen {
			f.rule.Regex = ""
		} else {
			f.rule.Regex = s
		}
		f.update()
	}
	regexItem := widget.NewAccordionItem("Pattern parts, regular expression", container.NewVBox(
		low("The parts a pattern can have: "+fixPlaceholders()+". The rest is matched as written, in any case, any dash for a dash."),
		f.regex,
		low("The pattern as a regular expression, matched against each line without its marks (! . @ >). Written by hand, it is used instead; its groups go in the result as <1>, <2>… or by their names.")))
	regexBox := widget.NewAccordion(regexItem)
	f.regexBox = regexBox

	var labels []string
	for _, a := range fixActions {
		labels = append(labels, a.label)
	}
	f.action = widget.NewSelect(labels, func(label string) {
		if f.filling {
			return
		}
		for _, a := range fixActions {
			if a.label == label {
				f.chooseAction(a.action)
			}
		}
	})
	fyne.SetAccessibleLabel(f.action, "Make them")

	f.template = widget.NewEntry()
	fyne.SetAccessibleLabel(f.template, "Result")
	f.template.OnChanged = func(s string) {
		if f.filling {
			return
		}
		f.rule.Template = s
		f.update()
	}
	f.tmplRow = container.NewVBox(bold("Result"), f.template,
		low("The pattern's parts (a second <text> is <text 2>); <line> is the whole line."))

	f.count = widget.NewLabel("")
	f.count.Wrapping = fyne.TextWrapWord
	f.preview = container.NewVBox()
	f.remember = widget.NewCheck("Keep the rule with the script", func(on bool) {
		w.settings.SetBoolean(fixRememberPref, !on)
	})
	f.remember.Checked = !w.settings.GetBoolean(fixRememberPref) // on unless switched off
	f.saved = container.NewVBox()

	parts := []fyne.CanvasObject{intro,
		bold("Line like this"), f.example,
		bold("Pattern"), f.pattern, f.count,
		bold("Make them"), f.action, f.tmplRow,
		bold("Preview"), f.preview,
		regexBox,
		f.remember, low("Kept in its Fix-Up block, to apply again after a re-import."),
		f.saved,
	}
	cancel := widget.NewButton("Cancel", func() { f.d.Hide() })
	f.apply = widget.NewButton("Apply", f.applyNow)
	f.apply.Importance = widget.HighImportance
	body := container.NewVScroll(container.NewVBox(parts...))
	f.d = dialog.NewCustomWithoutButtons("Fix Lines Like This", container.NewBorder(nil, w.dialogButtons(cancel, f.apply), nil, nil, body), w.fyneWindow)
	f.d.SetOnClosed(func() {
		w.textEditor.SetHighlights(nil)
		if w.fixAsked == f {
			w.fixAsked = nil
		}
	})
	f.d.Resize(fitDialog(w.fyneWindow, 640, 760))
	f.showSaved()
	f.propose()
	return f
}

// env is what the actions need of the script.
func (f *fixupDlg) env() fixEnv { return fixEnv{scenes: f.w.sceneHeaders()} }

// propose proposes the rule for the example line and shows it.
func (f *fixupDlg) propose() {
	f.setRule(proposeRule(f.example.Text, splitForFix(f.w.textEditor.Text())))
}

// setRule shows a rule in the controls.
func (f *fixupDlg) setRule(r fixRule) {
	f.rule = r
	f.filling = true
	f.pattern.SetText(r.Pattern)
	if r.Regex != "" {
		f.regex.SetText(r.Regex)
	}
	f.action.SetSelected(r.Action.label())
	f.template.SetText(r.Template)
	f.filling = false
	if r.Regex == "" {
		f.showRegex()
	}
	f.update()
}

// showRegex shows the pattern's regular expression.
func (f *fixupDlg) showRegex() {
	gen, _ := patternRegex(f.rule.Pattern)
	f.filling = true
	f.regex.SetText(gen)
	f.filling = false
}

// chooseAction makes the rule's action a, the pattern and the result
// starting over where the action needs its own.
func (f *fixupDlg) chooseAction(a fixAction) {
	r := f.rule
	r.Action = a
	if p := defaultPattern(a); p != "" && (r.Action != f.rule.Action) {
		// join, hyphens, spaces: what the lines are like, not what they say
		if prop := proposeRule(f.example.Text, splitForFix(f.w.textEditor.Text())); prop.Action != a {
			r.Pattern, r.Regex = p, ""
		}
	}
	if a.templated() && r.Template == "" {
		r.Template = "<line>"
		if a == fixNarration {
			r.Template = "NARRATOR (V.O.)"
		}
	}
	f.setRule(r)
}

// update counts the matches, lights them in the script and lists what
// they become.
func (f *fixupDlg) update() {
	if f.count == nil {
		return
	}
	if f.rule.Action.templated() {
		f.tmplRow.Show()
	} else {
		f.tmplRow.Hide()
	}
	text := f.w.textEditor.Text()
	lines := splitForFix(text)
	ms, err := lines.matches(f.rule)
	f.preview.RemoveAll()
	f.changes = nil
	if err != nil {
		f.count.Importance = widget.DangerImportance
		f.count.SetText("The regular expression does not read: " + err.Error())
		f.w.textEditor.SetHighlights(nil)
		f.apply.Disable()
		return
	}
	cs, _ := lines.changes(f.rule, f.env())
	f.changes = cs
	var at []int
	for _, m := range ms {
		at = append(at, m.line)
	}
	f.w.textEditor.SetHighlights(lineRanges(lines.lines, at))
	f.count.Importance = widget.MediumImportance
	switch {
	case len(ms) == 0:
		f.count.SetText("No line matches the pattern.")
	case len(cs) == len(ms):
		f.count.SetText(fmt.Sprintf("%s, lit in the script.", matchCount(len(ms))))
	default:
		f.count.SetText(fmt.Sprintf("%s, lit in the script; %d of them change.", matchCount(len(ms)), len(cs)))
	}
	f.count.Refresh()
	for k, c := range cs {
		if k == fixPreviewMax {
			f.preview.Add(widget.NewLabel(fmt.Sprintf("… and %d more", len(cs)-fixPreviewMax)))
			break
		}
		f.preview.Add(fixChangeView(c.before(lines), c.after))
	}
	if len(cs) == 0 {
		f.apply.SetText("Apply")
		f.apply.Disable()
	} else {
		f.apply.SetText(fmt.Sprintf("Apply to %d", len(cs)))
		f.apply.Enable()
	}
}

// matchCount is "1 line matches", "3 lines match".
func matchCount(n int) string {
	if n == 1 {
		return "1 line matches"
	}
	return fmt.Sprintf("%d lines match", n)
}

// fixChangeView is one change in the preview: the lines as they are,
// quiet, and what they become.
func fixChangeView(before, after []string) fyne.CanvasObject {
	show := func(ls []string) string {
		var p []string
		for _, l := range ls {
			if strings.TrimSpace(l) != "" {
				p = append(p, strings.TrimSpace(l))
			}
		}
		if len(p) == 0 {
			return "(dropped)"
		}
		s := strings.Join(p, " ⏎ ")
		if utf8.RuneCountInString(s) > 160 {
			s = string([]rune(s)[:160]) + "…"
		}
		return s
	}
	was := widget.NewLabel(show(before))
	was.Wrapping = fyne.TextWrapWord
	was.Importance = widget.LowImportance
	now := widget.NewLabelWithStyle("→ "+show(after), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	now.Wrapping = fyne.TextWrapWord
	return container.NewVBox(was, now, widget.NewSeparator())
}

// applyNow makes the change, in one undo step, and keeps the rule.
func (f *fixupDlg) applyNow() {
	r := f.rule
	n, err := f.w.applyFixRules([]fixRule{r}, f.remember.Checked)
	if err != nil {
		f.count.SetText(err.Error())
		return
	}
	f.d.Hide()
	what := strings.ToLower(r.Action.label())
	kept := ""
	if f.remember.Checked {
		kept = "; the rule is kept in the script's Fix-Up block"
	}
	f.w.setStatus(fmt.Sprintf("Fix Lines: %s (%s), one undo step%s", plural(n, "change"), what, kept))
}

// applyFixRules applies rules to the script, one after the other, in one
// undo step, and keeps them in its Fix-Up block if keep; it is how many
// places changed.
func (w *MainWindow) applyFixRules(rules []fixRule, keep bool) (int, error) {
	text := w.textEditor.Text()
	out, counts := applyRules(text, rules, fixEnv{scenes: w.sceneHeaders()})
	n := 0
	for _, c := range counts {
		n += c
	}
	if n == 0 && !keep {
		return 0, nil
	}
	start, end, mid := changedSpan(text, out)
	w.textEditor.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			if n > 0 {
				b.Replace(start, end, mid)
			}
			if keep { // the block at the end, of the text as it is now
				for _, r := range rules {
					bs, be, block := withFixRule(b.Text(), r)
					b.Replace(bs, be, block)
				}
			}
			b.SetCursor(min(start, b.Len()), false)
		})
	})
	return n, nil
}

// showSaved lists the script's saved rules, to pick one or apply them all.
func (f *fixupDlg) showSaved() {
	f.saved.RemoveAll()
	rules := fixRulesOf(f.w.textEditor.Text())
	if len(rules) == 0 {
		return
	}
	bold := widget.NewLabelWithStyle(fmt.Sprintf("This script's rules (%d)", len(rules)), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	f.saved.Add(bold)
	for _, r := range rules {
		r := r
		b := widget.NewButton(r.String(), func() { f.setRule(r) })
		b.Alignment = widget.ButtonAlignLeading
		f.saved.Add(b)
	}
	all := widget.NewButton("Apply Them All Again…", func() { f.w.askApplySaved(rules, f.d) })
	f.saved.Add(all)
}

// askApplySaved asks before applying the saved rules again: how many
// places each changes, then one undo step for all.
func (w *MainWindow) askApplySaved(rules []fixRule, parent dialog.Dialog) {
	_, counts := applyRules(w.textEditor.Text(), rules, fixEnv{scenes: w.sceneHeaders()})
	var lines []string
	total := 0
	for k, r := range rules {
		lines = append(lines, fmt.Sprintf("%s: %s", r.Action.label(), plural(counts[k], "change")))
		total += counts[k]
	}
	if total == 0 {
		w.setStatus("The script's fix-up rules change nothing: it is fixed up already")
		return
	}
	msg := widget.NewLabel(strings.Join(lines, "\n") + "\n\nOne undo step takes it all back.")
	msg.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustomConfirm("Apply the Script's Rules Again", "Apply", "Cancel", msg, func(ok bool) {
		if !ok {
			return
		}
		if parent != nil {
			parent.Hide()
		}
		n, _ := w.applyFixRules(rules, false)
		w.setStatus(fmt.Sprintf("Fix-up rules applied again: %s, one undo step", plural(n, "change")))
	}, w.fyneWindow)
	d.Resize(fitDialog(w.fyneWindow, 460, 320))
	d.Show()
}

// carryFixRules is the imported text with the Fix-Up block of the script
// it replaces, if that was imported from a file of the same name (a
// re-import), and the rules carried; the window then offers to apply
// them (offerFixRules), never applying them itself.
func (w *MainWindow) carryFixRules(filePath, text string) (string, []fixRule) {
	if w.textEditor == nil || w.markdown {
		return text, nil
	}
	old := w.textEditor.Text()
	rules := fixRulesOf(old)
	if len(rules) == 0 {
		return text, nil
	}
	stem := func(p string) string {
		b := filepath.Base(p)
		for filepath.Ext(b) != "" && len(b) > len(filepath.Ext(b)) {
			b = strings.TrimSuffix(b, filepath.Ext(b))
		}
		return strings.ToLower(b)
	}
	was := w.currentFile
	if was == "" {
		was = w.suggestedName
	}
	if was == "" || stem(was) != stem(filePath) {
		return text, nil
	}
	for _, r := range rules {
		start, end, block := withFixRule(text, r)
		rs := []rune(text)
		text = string(rs[:start]) + block + string(rs[end:])
	}
	return text, rules
}

// offerFixRules asks whether to apply the rules a re-import carried.
func (w *MainWindow) offerFixRules(rules []fixRule) {
	if len(rules) == 0 {
		return
	}
	w.askApplySaved(rules, nil)
}
