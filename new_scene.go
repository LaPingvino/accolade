package main

import (
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/rules"
)

// Insert › New Scene… (Format menu, Write › Scene, :new-scene, and from
// Inside or Outside…; Joop 2026-11-23: "an 'insert new scene' option
// that formats this properly with fill in fields"): a small form with a
// field for each slot of the form the script's language writes a new
// heading by (rules.Language.HeadingForm: its NewHeading, its first
// heading pattern, or Fountain's INT. PLACE - DAY), in that form's
// order: Inside/Outside in the language's words, the Place with the
// script's places to pick from, the Time in its time words, the Number,
// free text. The heading it makes shows as it is typed, with whether it
// reads as one; Insert puts it after the cursor's scene (at the cursor
// between scenes), one undo step, the cursor on the scene's first action
// line. In a locked script the number is the one the lock gives a scene
// there (12A), written with the heading.

// sceneDraft is what the form holds: the slots' values typed and what
// they make; inWords, the script has its numbers in its headings
// (scene_numbers.go), read and written by words.
type sceneDraft struct {
	l         rules.Language
	form      string
	values    map[string]string
	suggested string // the number a scene there gets: the lock's, or by counting
	locked    bool
	inWords   bool
	words     *fountain.Dialect
	hash      bool // a locked script whose headings have #12#: this one too
}

// heading is the heading the draft makes: the form filled in. With the
// numbers in the headings, the number (typed, else the suggested one)
// is in its words: its {n}, or, for a form without one, where the
// language's pattern with one has it (hi-Deva's "3. रसोई/दिन/अंदर"), else
// as Fountain's #12#; in a locked script whose headings have #12#, as
// that too, the two equal. Without, a number typed or a locked script's
// is Fountain's #12#.
func (d sceneDraft) heading() string {
	v := map[string]string{}
	for k, x := range d.values {
		v[k] = strings.TrimSpace(x)
	}
	num := v[rules.SlotNumber]
	if num == "" && (d.locked || d.inWords) { // a locked script's scenes all have one; so do numbered headings
		num = d.suggested
	}
	v[rules.SlotNumber] = ""
	hasN := slices.Contains(rules.PatternSlots(d.form), rules.SlotNumber)
	if d.inWords && hasN {
		v[rules.SlotNumber] = num
	}
	h := rules.FillPattern(d.form, v)
	switch {
	case num == "":
		return h
	case v[rules.SlotNumber] != "":
	case d.inWords && d.words != nil:
		if out, ok := d.words.WithHeadingNumber(h, num); ok {
			h = out
			break
		}
		fallthrough
	default:
		return lex.WithSceneNumber(h, num)
	}
	if d.hash && d.locked {
		h = lex.WithSceneNumber(h, num)
	}
	return h
}

// reads tells whether the heading reads as one in the language: a slot
// left empty is left out with its own text (rules.FillPattern), and what
// is left may still be one (INT. KITCHEN, 场景2 厨房 内, ○台所).
func (d sceneDraft) reads() bool {
	h := d.heading()
	return strings.TrimSpace(h) != "" && readsAsHeading(d.l, h)
}

// newSceneAt is where a new scene goes, in runes: after the scene the
// cursor is in (before the next heading or section); between scenes at
// the cursor's line if it is blank, else after its paragraph; after the
// title page on it; before the blocks at the end after them.
func newSceneAt(text string, cursor int, starts []string) int {
	runes := []rune(text)
	body := bodyEnd(text)
	cursor = min(max(cursor, 0), len(runes))
	if cursor >= body {
		return body
	}
	lines := strings.Split(text, "\n")
	lineStart := func(i int) int { // in runes
		n := 0
		for _, l := range lines[:i] {
			n += len([]rune(l)) + 1
		}
		return min(n, len(runes))
	}
	line := strings.Count(string(runes[:cursor]), "\n")
	if tp := titlePageEnd(lines); tp > 0 && line < tp {
		return min(lineStart(tp), body)
	}
	in, next := false, -1
	for _, it := range outlineOf(text, starts) {
		if lineStart(it.Line) >= body {
			break
		}
		if it.Line <= line {
			in = it.Scene
			continue
		}
		next = it.Line
		break
	}
	if in {
		if next >= 0 {
			return lineStart(next)
		}
		return body
	}
	if strings.TrimSpace(lines[line]) == "" {
		return lineStart(line)
	}
	for i := line + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			return min(lineStart(i), body)
		}
	}
	return body
}

// newSceneText is what goes in at at for a scene with heading: the blank
// lines it needs before and after, and where the cursor goes (the first
// action line), in runes from at.
func newSceneText(text string, at int, heading string) (insert string, cursor int) {
	runes := []rune(text)
	before, after := string(runes[:at]), string(runes[at:])
	pre := ""
	switch {
	case before == "" || strings.HasSuffix(before, "\n\n"):
	case strings.HasSuffix(before, "\n"):
		pre = "\n"
	default:
		pre = "\n\n"
	}
	insert = pre + heading + "\n\n"
	cursor = len([]rune(insert))
	switch {
	case strings.TrimSpace(after) == "":
	case strings.HasPrefix(after, "\n"):
		insert += "\n"
	default:
		insert += "\n\n"
	}
	return insert, cursor
}

// suggestedNumber is the number a scene put in at at gets: in a locked
// script the lock's (12A), else by counting.
func suggestedNumber(text string, starts []string, at int, lock sceneLock) string {
	heads := sceneHeads(text[:bodyEndByte(text)], starts)
	line := strings.Count(string([]rune(text)[:at]), "\n")
	k := 0
	for k < len(heads) && heads[k].line < line {
		k++
	}
	heads = slices.Insert(heads, k, sceneHead{line: line})
	o := lex.NumberOptions{}
	if lock.locked {
		o = lex.NumberOptions{Locked: true, Style: lock.style}
	}
	nums := sceneNumbersFor(heads, o)
	if k < len(nums) {
		return nums[k]
	}
	return ""
}

// newSceneDraft is the draft for the window's script at its cursor.
func (w *MainWindow) newSceneDraft() (sceneDraft, int) {
	text := w.textEditor.Text()
	starts := w.sceneStarts()
	l := w.lex().Language(w.language)
	at := newSceneAt(text, w.textEditor.CursorOffset(), starts)
	lock := sceneLockOf(text)
	d := sceneDraft{l: l, form: l.HeadingForm(), values: map[string]string{}, locked: lock.locked,
		suggested: suggestedNumber(text, starts, at, lock), inWords: w.numbersInHeadings(text), words: wordsDialect(starts)}
	for _, h := range sceneHeads(text[:bodyEndByte(text)], starts) {
		if _, n := lex.SceneNumber(h.text); n != "" {
			d.hash = true
			break
		}
	}
	if io := l.IOWords(d.form); len(io) > 0 {
		d.values[rules.SlotIO] = io[0]
	}
	if len(l.Times) > 0 {
		d.values[rules.SlotTime] = l.Times[0]
	}
	return d, at
}

// showNewScene shows the form.
func (w *MainWindow) showNewScene() {
	if w.markdown || w.textEditor == nil || w.scriptAway("New Scene") {
		return
	}
	var dlg dialog.Dialog
	content, first := w.newSceneContent(func() { dlg.Hide() })
	dlg = dialog.NewCustomWithoutButtons("New Scene", content, w.fyneWindow)
	dlg.Resize(fitDialog(w.fyneWindow, 520, 640))
	dlg.Show()
	if first != nil {
		w.fyneWindow.Canvas().Focus(first)
	}
}

// newSceneContent is the form, and its first field to type in; hide
// closes it. Insert puts the scene in after the cursor's.
func (w *MainWindow) newSceneContent(hide func()) (fyne.CanvasObject, fyne.Focusable) {
	return w.newSceneContentWith(hide, nil)
}

// newSceneContentWith is the form; with put, it has a Synopsis too and
// Insert gives put the heading and the synopsis (the cards' New Scene,
// cards.go).
func (w *MainWindow) newSceneContentWith(hide func(), put func(heading, synopsis string)) (fyne.CanvasObject, fyne.Focusable) {
	d, at := w.newSceneDraft()
	text := w.textEditor.Text()
	src := slotSourceOf(text, d.l, w.sceneStarts()) // what the slots take (heading_slots.go)
	places := src.places
	preview := widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Monospace: true})
	preview.Wrapping = fyne.TextWrapWord
	note := widget.NewLabel("")
	note.Wrapping = fyne.TextWrapWord
	insert := widget.NewButtonWithIcon("Insert", theme.ContentAddIcon(), nil)
	insert.Importance = widget.HighImportance
	update := func() {
		preview.SetText(d.heading())
		if d.reads() {
			note.Importance = widget.MediumImportance
			note.SetText("Reads as a scene heading in " + languageLabel(w.language) + ".")
			insert.Enable()
		} else {
			note.Importance = widget.WarningImportance
			note.SetText("Not a scene heading in " + languageLabel(w.language) + " yet: fill in more of it.")
			insert.Disable()
		}
		note.Refresh()
	}
	fields := container.NewVBox()
	var first fyne.Focusable
	label := func(s string) fyne.CanvasObject {
		return widget.NewLabelWithStyle(s, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}
	seen := map[string]bool{}
	for _, slot := range rules.PatternSlots(d.form) {
		if seen[slot] {
			continue
		}
		seen[slot] = true
		switch slot {
		case rules.SlotIO:
			words := src.choices(rules.SlotIO, d.form)
			if !strings.HasPrefix(d.form, "{"+rules.SlotIO+"}") { // not the word that makes it a heading: may be left out
				words = append(words, noneChoice)
			}
			sel := widget.NewSelect(words, func(v string) { d.values[rules.SlotIO] = noneIs(v); update() })
			if len(words) > 0 {
				sel.SetSelected(words[0])
			}
			fyne.SetAccessibleLabel(sel, "Inside or outside")
			fields.Add(container.NewVBox(label("Inside/Outside"), sel))
		case rules.SlotPlace:
			en := widget.NewEntry()
			en.SetPlaceHolder(samplePlaceFor(w.language))
			picks := container.New(flowLayout{})
			flow := newFlowBox()
			flow.box = picks
			fill := func(typed string) {
				picks.Objects = nil
				q := strings.ToUpper(strings.TrimSpace(typed))
				n := 0
				for _, p := range places {
					if n == 6 || p == q {
						continue
					}
					if q == "" || strings.Contains(strings.ToUpper(p), q) {
						p := p
						b := widget.NewButton(p, func() { en.SetText(p) })
						b.Importance = widget.LowImportance
						picks.Objects = append(picks.Objects, b)
						n++
					}
				}
				picks.Refresh()
				flow.Refresh()
			}
			en.OnChanged = func(v string) {
				d.values[rules.SlotPlace] = v
				fill(v)
				update()
			}
			en.OnSubmitted = func(string) {
				if !insert.Disabled() {
					insert.OnTapped()
				}
			}
			fyne.SetAccessibleLabel(en, "Place")
			fill("")
			fields.Add(container.NewVBox(label("Place"), en, flow))
			first = en
		case rules.SlotTime:
			var c fyne.CanvasObject
			if len(d.l.Times) > 0 {
				sel := widget.NewSelect(append(slices.Clone(d.l.Times), noneChoice), func(v string) { d.values[rules.SlotTime] = noneIs(v); update() })
				sel.SetSelected(d.values[rules.SlotTime])
				fyne.SetAccessibleLabel(sel, "Time")
				c = sel
			} else { // free: the script's times to pick from
				opts := src.timeWords()
				sel := widget.NewSelectEntry(opts)
				sel.OnChanged = func(v string) { d.values[rules.SlotTime] = v; update() }
				if len(opts) > 0 {
					sel.SetText(opts[0])
				}
				fyne.SetAccessibleLabel(sel, "Time")
				c = sel
			}
			fields.Add(container.NewVBox(label("Time"), c))
		case rules.SlotNumber:
			fields.Add(numberField(&d, label, update))
		case rules.SlotAny:
			en := widget.NewEntry()
			en.SetPlaceHolder(samplePlaceFor(w.language))
			en.OnChanged = func(v string) { d.values[rules.SlotAny] = v; update() }
			fyne.SetAccessibleLabel(en, "Free text")
			fields.Add(container.NewVBox(label("Free text"), en))
			if first == nil {
				first = en
			}
		}
	}
	if !seen[rules.SlotNumber] { // Fountain's #12#, optional unless locked
		fields.Add(numberField(&d, label, update))
	}
	var syn *widget.Entry
	if put != nil {
		syn = widget.NewMultiLineEntry()
		syn.Wrapping = fyne.TextWrapWord
		syn.SetPlaceHolder("What happens")
		fields.Add(container.NewVBox(label("Synopsis"), syn))
	}
	form := pageNote("Written as " + d.form + " (" + languageLabel(w.language) + "; Lexington Settings > Languages > NewHeading changes it).")
	insert.OnTapped = func() {
		if !d.reads() {
			return
		}
		hide()
		if put != nil {
			put(d.heading(), syn.Text)
			return
		}
		w.insertScene(at, d.heading())
	}
	cancel := widget.NewButton("Cancel", hide)
	body := container.NewVBox(fields, form)
	result := container.NewVBox(widget.NewSeparator(), label("Heading"), preview, note)
	update()
	return container.NewBorder(nil, container.NewVBox(result, w.dialogButtons(cancel, insert)), nil, nil, container.NewVScroll(body)), first
}

// noneChoice is a word slot's choice of nothing: the heading without it.
const noneChoice = "(none)"

// noneIs is a choice's value: "" for noneChoice.
func noneIs(v string) string {
	if v == noneChoice {
		return ""
	}
	return v
}

// numberField is the Number: the suggested one shown until another is
// typed.
func numberField(d *sceneDraft, label func(string) fyne.CanvasObject, update func()) fyne.CanvasObject {
	if d.inWords && !d.locked { // the scene's place gives it: none to type
		return container.NewVBox(label("Number"), pageNote(d.suggested+": kept in step with the scenes' order (Format › Number Scenes in Headings)."))
	}
	en := widget.NewEntry()
	hint := "Optional"
	switch {
	case d.locked:
		hint = d.suggested + " (the locked numbers' next)"
	case d.suggested != "":
		hint = "Optional: " + d.suggested + " is next"
	}
	en.SetPlaceHolder(hint)
	en.OnChanged = func(v string) { d.values[rules.SlotNumber] = v; update() }
	fyne.SetAccessibleLabel(en, "Scene number")
	return container.NewVBox(label("Number"), en)
}

// samplePlaceFor is a place in lang, as a hint.
func samplePlaceFor(lang string) string {
	if p := samplePlaces[lang]; p != "" {
		return p
	}
	return "KITCHEN"
}

// insertScene puts a scene with heading in at at: one undo step, the
// cursor on its first action line.
func (w *MainWindow) insertScene(at int, heading string) {
	text := w.textEditor.Text()
	at = min(at, len([]rune(text)))
	ins, cur := newSceneText(text, at, heading)
	w.holdingStep(func() {
		w.textEditor.Edit(func(b *buffer.Buffer) {
			b.Group(func() {
				b.Replace(at, at, ins)
				b.SetCursor(at+cur, false)
			})
		})
	})
	w.fyneWindow.Canvas().Focus(w.textEditor)
	w.setStatus("New scene: " + heading)
	w.keepInStepSaying("New scene: " + heading) // the scenes after it, in the same undo step
}
