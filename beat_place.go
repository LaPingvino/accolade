package main

import (
	"fmt"
	"image/color"
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/pdf"
)

// Where the beats land (LOOP.md, (A); Joop 2026-11-24: "We need
// statistics that come with a call to action"): each section's page and
// its place in the script (% of the pages), read from the page layout
// the PDF and its page numbers are made of (drawer, the Preview's), so
// it is the page the script will print it on; nothing kept but the last
// layout made, for the text it was made of. A script whose sections are
// a structure's (structure_templates.go, matched by their names) gets
// that structure's typical ranges beside them; the Revision tab's Beats
// group shows the cursor's section, lists them all (each a link there)
// and steps to the ones outside their range. Any other structure, the
// writer's own, gets the pages and the percentages, without a range.

// beatPlace is a section and where it lands.
type beatPlace struct {
	line, level int
	name        string
	page, pages int          // its page (the script's pages counted from 1, the title page not), of how many
	pct         float64      // where in the pages it starts, 0–100
	lo, hi      float64      // its typical range, if has
	film        *filmMeasure // where films put it (beat_films.go), nil: the range is the convention
	has         bool
}

// rng is the place's range, measured or the convention.
func (b beatPlace) rng() beatRange { return beatRange{lo: b.lo, hi: b.hi, film: b.film} }

// off tells whether the section lands outside its typical range, give or
// take a page (its place is counted by the page it is on, so a page's
// share either side is the layout's, not the writing's).
func (b beatPlace) off() bool {
	grace := 0.0
	if b.pages > 0 {
		grace = 100 / float64(b.pages)
	}
	return b.has && (b.pct < b.lo-grace || b.pct > b.hi+grace)
}

// usually is its range as the marks say it: "usually 19–31%, measured".
func (b beatPlace) usually() string {
	return fmt.Sprintf("usually %.0f–%.0f%%, %s", b.lo, b.hi, b.rng().kind())
}

// String is how the readout and the list say it.
func (b beatPlace) String() string {
	s := fmt.Sprintf("%s · p. %d · %.0f%%", b.name, b.page, b.pct)
	if b.has {
		s += " (" + b.usually() + ")"
	}
	return s
}

// advice is the call to action for a section off its range.
func (b beatPlace) advice() string {
	switch {
	case !b.off():
		return b.String()
	case b.pct < b.lo:
		return fmt.Sprintf("%s comes early: page %d of %d (%.0f%%), usually %.0f–%.0f%% (%s): what before it could grow?", b.name, b.page, b.pages, b.pct, b.lo, b.hi, b.rng().source())
	}
	return fmt.Sprintf("%s comes late: page %d of %d (%.0f%%), usually %.0f–%.0f%% (%s): what before it could go or be shorter?", b.name, b.page, b.pages, b.pct, b.lo, b.hi, b.rng().source())
}

// sectionKey is a section's name as names are matched: letters and
// digits in lower case, one space between.
func sectionKey(s string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }), " ")
}

// templateSections are a template's section names, by their key.
func templateSections(t structureTemplate) map[string]bool {
	out := map[string]bool{}
	for _, l := range strings.Split(t.body, "\n") {
		if strings.HasPrefix(l, "#") {
			out[sectionKey(strings.TrimLeft(l, "#"))] = true
		}
	}
	return out
}

// structureOf is the structure whose sections the script's are: the one
// whose names most of them have (two at least), else none.
func structureOf(names []string) (structureTemplate, bool) {
	best, n := -1, 1
	for i, t := range structureTemplates {
		ts := templateSections(t)
		k := 0
		for _, s := range names {
			if ts[sectionKey(s)] {
				k++
			}
		}
		if k > n || k == n && best >= 0 && len(t.ranges) > 0 && len(structureTemplates[best].ranges) == 0 {
			best, n = i, k
		}
	}
	if best < 0 {
		return structureTemplate{}, false
	}
	return structureTemplates[best], true
}

// placesOf are the script's sections and where they land in d, the
// pages made of text (with lines: drawer).
func placesOf(text string, starts []string, d pdf.Drawing) (places []beatPlace, structure string) {
	var body []int // the pages' indexes that print the script (not its title page)
	for i, p := range d.Pages {
		if p.First > 0 {
			body = append(body, i)
		}
	}
	if len(body) == 0 {
		return nil, ""
	}
	var names []string
	outline := outlineOf(text[:fountainBodyEnd(text)], starts)
	for _, o := range outline {
		if !o.Scene {
			names = append(names, o.Text)
		}
	}
	t, matched := structureOf(names)
	for _, o := range outline {
		if o.Scene {
			continue
		}
		// a section prints nothing: its page is the one its next printed
		// line is on (Source counts from 1)
		k := len(body) - 1
		for j, i := range body {
			if d.Pages[i].Last >= o.Line+1 {
				k = j
				break
			}
		}
		b := beatPlace{line: o.Line, level: o.Level, name: o.Text, page: k + 1, pages: len(body), pct: 100 * float64(k) / float64(len(body))}
		if matched {
			if r, ok := t.ranges[sectionKey(o.Text)]; ok {
				b.lo, b.hi, b.film, b.has = r.lo, r.hi, r.film, true
			}
		}
		places = append(places, b)
	}
	if matched {
		structure = t.name
	}
	return places, structure
}

// beatLayout is the last layout the placement read: the text it is of,
// the places, and whether one is being made.
type beatLayout struct {
	text      string
	places    []beatPlace
	structure string
	ok, busy  bool
}

// beatPlaces are the script's places as last laid out (stale while the
// layout of the text now is made in the background, which then shows on
// the ribbon); ok false if none was made yet.
func (w *MainWindow) beatPlaces() ([]beatPlace, string, bool) {
	if w.textEditor == nil || w.markdown {
		return nil, "", false
	}
	text := w.textEditor.Text()
	l := &w.beats
	if !strings.Contains(text, "#") { // no sections: nothing to lay out for
		return nil, "", true
	}
	if (!l.ok || !sameString(l.text, text)) && !l.busy {
		starts := w.sceneStarts()
		if v := w.previewView(); v != nil && v.follow != nil && sameString(v.follow.baseText, text) { // the Preview's pages are this text's
			l.places, l.structure = placesOf(text, starts, v.follow.base)
			l.text, l.ok = text, true
		} else {
			l.busy = true
			job := w.outputJob(w.pageSetup(), "pdf", text, nil)
			draw := drawer(text, job)
			type got struct {
				places    []beatPlace
				structure string
			}
			inBackground(func() got {
				d, err := draw(nil)
				if err != nil {
					return got{}
				}
				p, s := placesOf(text, starts, d)
				return got{p, s}
			}, func(g got) {
				l.busy = false
				l.text, l.places, l.structure, l.ok = text, g.places, g.structure, true
				w.beatsLanded()
			})
		}
	}
	return l.places, l.structure, l.ok
}

// placeAtCursor is the section the cursor is in, -1 none.
func (w *MainWindow) placeAtCursor(places []beatPlace) int {
	line := w.scriptCursorLine()
	k := -1
	for i, p := range places {
		if p.line <= line {
			k = i
		}
	}
	return k
}

// beatsReadout is the Beats group's readout: the cursor's section and
// where it lands, else how many are off their range.
func (w *MainWindow) beatsReadout() string {
	places, _, ok := w.beatPlaces()
	switch {
	case !ok:
		return "Laying Out…"
	case len(places) == 0:
		return "No Sections"
	}
	if k := w.placeAtCursor(places); k >= 0 {
		p := places[k]
		name := p.name
		if r := []rune(name); len(r) > 18 {
			name = string(r[:17]) + "…"
		}
		s := fmt.Sprintf("%s · p. %d · %.0f%%", name, p.page, p.pct)
		if p.off() {
			s += " ⚑"
		}
		return s
	}
	return plural(len(places), "Section")
}

// beatsNamed is the readout's tooltip: the section, its place, its range.
func (w *MainWindow) beatsNamed() string {
	places, structure, ok := w.beatPlaces()
	name := "Where the Beats Land"
	if !ok || len(places) == 0 {
		return name + ": each section's page and place in the script (# sections)"
	}
	if k := w.placeAtCursor(places); k >= 0 {
		name += ": " + places[k].String()
	}
	if structure != "" {
		name += " — ranges of " + structure + ": " + rangesNote(structure)
	}
	return name
}

// offPlaces are the places off their range.
func offPlaces(places []beatPlace) []beatPlace {
	var out []beatPlace
	for _, p := range places {
		if p.off() {
			out = append(out, p)
		}
	}
	return out
}

// beatsMenu lists the sections and where they land, each going there.
func (w *MainWindow) beatsMenu() *fyne.Menu {
	places, structure, ok := w.beatPlaces()
	var items []*fyne.MenuItem
	for _, p := range places {
		label := strings.Repeat("   ", max(0, p.level-1)) + p.String()
		if p.off() {
			label = "⚑ " + label
		}
		items = append(items, fyne.NewMenuItem(label, func() { w.goToScript(p.line); w.setStatus(p.advice()) }))
	}
	if len(items) == 0 {
		say := "No # sections to place yet"
		if !ok {
			say = "Laying the pages out…"
		}
		none := fyne.NewMenuItem(say, nil)
		none.Disabled = true
		items = append(items, none)
	} else if structure != "" {
		note := fyne.NewMenuItem("Ranges: "+structure+"'s, "+rangesNote(structure), nil)
		note.Disabled = true
		items = append(items, fyne.NewMenuItemSeparator(), note)
	}
	return fyne.NewMenu("Where the Beats Land", items...)
}

// stepOff goes to the next (1) or previous (-1) section off its range,
// round at the ends, and says what to do about it.
func (w *MainWindow) stepOff(dir int) {
	if w.textEditor == nil || w.scriptAway("Next Section Off Its Range") {
		return
	}
	places, _, ok := w.beatPlaces()
	off := offPlaces(places)
	if len(off) == 0 {
		switch {
		case !ok:
			w.setStatus("The pages are being laid out: a moment")
		case len(places) > 0 && !places[0].has && !anyHas(places):
			w.setStatus("These sections are not a structure Accolade knows (File > New from Structure): no ranges to compare")
		default:
			w.setStatus("Every section lands within its usual range")
		}
		return
	}
	line := w.scriptCursorLine()
	k := -1
	if dir > 0 {
		for i, p := range off {
			if p.line > line {
				k = i
				break
			}
		}
		if k < 0 {
			k = 0
		}
	} else {
		for i := len(off) - 1; i >= 0; i-- {
			if off[i].line < line {
				k = i
				break
			}
		}
		if k < 0 {
			k = len(off) - 1
		}
	}
	w.goToScript(off[k].line)
	w.setStatus(fmt.Sprintf("%d of %d off its range · %s", k+1, len(off), off[k].advice()))
	w.ribbonRefresh()
}

// anyHas tells whether any place has a range.
func anyHas(places []beatPlace) bool {
	for _, p := range places {
		if p.has {
			return true
		}
	}
	return false
}

// offLabel is Next Off's ribbon label: with how many there are.
func (w *MainWindow) offLabel() string {
	places, _, _ := w.beatPlaces()
	if n := len(offPlaces(places)); n > 0 {
		return fmt.Sprintf("Next Off (%d)", n)
	}
	return "Next Off"
}

// beatCommands are the Beats group's.
func beatCommands() []*command {
	script := func(e cmdEnv) bool { return !e.away && !e.markdown }
	return []*command{
		{id: "revision.beats", label: "Where the Beats Land", ex: "where-beats-land", icon: theme.ListIcon,
			tip:            "Each section's page and place in the script, beside its structure's usual range; each goes there",
			ribbonLabelFor: (*MainWindow).beatsReadout, nameFor: (*MainWindow).beatsNamed,
			when: script, submenu: (*MainWindow).beatsMenu},
		{id: "revision.nextOff", label: "Next Section Off Its Range", ribbonLabel: "Next Off", ex: "next-off-range", icon: theme.NavigateNextIcon,
			tip:            "The next section landing outside its structure's usual range, and what to do about it",
			ribbonLabelFor: (*MainWindow).offLabel,
			when:           script, run: func(w *MainWindow) { w.stepOff(1) }},
		{id: "revision.previousOff", label: "Previous Section Off Its Range", ribbonLabel: "Previous Off", ex: "previous-off-range", icon: theme.NavigateBackIcon,
			tip:  "The section before the cursor's landing outside its usual range",
			when: script, run: func(w *MainWindow) { w.stepOff(-1) }},
	}
}

// Where the beats land in the views that show the sections (Joop
// 2026-11-24: "integrated with the cards timeline view etc."): a section
// with a range gets a quiet mark on its Cards header, its timeline band
// and its outline row (on target, early, late; the strip over the script
// flags an off one's tick), a tap going there with
// what to do about it. The places are beatPlaces' (the layout the
// Revision ribbon reads); while a new one is made the last stays, each
// matched to its section by order and name, so typing does not blink.

// landings are the places of items' sections with a range, by the
// section's line (none in Markdown, or before a layout). lay asks for a
// layout of the text now if the last is stale (the Cards and the
// timeline, which follow the typing after a pause); the outline, which
// follows each key, takes the last as it is and is shown the next.
func (w *MainWindow) landings(items []outlineItem, lay bool) map[int]beatPlace {
	if w.textEditor == nil || w.markdown {
		return nil
	}
	places := w.beats.places
	if lay {
		places, _, _ = w.beatPlaces()
	}
	if !anyHas(places) {
		return nil
	}
	out := map[int]beatPlace{}
	k := 0
	for _, it := range items {
		if it.Scene {
			continue
		}
		if k < len(places) && sectionKey(places[k].name) == sectionKey(it.Text) && places[k].has {
			out[it.Line] = places[k]
		}
		k++
	}
	return out
}

// landWord is a place's state in a word: on target, early, late.
func (b beatPlace) landWord() string {
	switch {
	case !b.off():
		return "On target"
	case b.pct < b.lo:
		return "Early"
	}
	return "Late"
}

// landMark is a place's mark: "⚑ Late · p. 60 · usually 49–53%,
// measured", or, on target, "✓ p. 31 · measured" ("convention" for a
// range no study gives).
func (b beatPlace) landMark() string {
	if !b.off() {
		return fmt.Sprintf("✓ p. %d · %s", b.page, b.rng().kind())
	}
	return fmt.Sprintf("⚑ %s · p. %d · %s", b.landWord(), b.page, b.usually())
}

// goToLanding goes to the section and says where it lands, with what to
// do about it if it is off.
func (w *MainWindow) goToLanding(b beatPlace) {
	w.goToScript(b.line)
	if b.off() {
		w.setStatus(b.advice())
		return
	}
	w.setStatus(fmt.Sprintf("%s lands on target: page %d of %d (%.0f%%), usually %.0f–%.0f%% (%s)", b.name, b.page, b.pages, b.pct, b.lo, b.hi, b.rng().source()))
}

// landLink is a Cards header's mark of where its section lands.
func landLink(w *MainWindow, b beatPlace) *linkLabel {
	l := newLinkLabel(func() { w.goToLanding(b) })
	l.SetText(b.landMark())
	l.Importance = widget.LowImportance
	if b.off() {
		l.Importance = widget.WarningImportance
	}
	return l
}

// beatsLanded shows a layout made in the background where the sections
// are shown: the cards (and timeline) in the middle and in the dock, the
// outline, the strip over the script (an off section's tick flagged).
func (w *MainWindow) beatsLanded() {
	w.ribbonRefresh()
	if w.textEditor == nil {
		return
	}
	if w.cards != nil && w.cardsShown() {
		w.cards.update(w.textEditor.Text())
	}
	if w.cardsDock != nil && w.paneInSight("cards") {
		w.cardsDock.update(w.textEditor.Text())
	}
	if w.outline != nil && w.outlineInSight() {
		w.outline.refresh()
	}
	if w.strip != nil && w.stripShown() {
		w.strip.update()
	}
}

// tapZone is a timeline band's tap: a transparent area over it.
type tapZone struct {
	widget.BaseWidget
	onTap func()
}

func newTapZone(onTap func()) *tapZone {
	z := &tapZone{onTap: onTap}
	z.ExtendBaseWidget(z)
	return z
}

func (z *tapZone) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

func (z *tapZone) Tapped(*fyne.PointEvent) { z.onTap() }

func (z *tapZone) Cursor() desktop.Cursor { return desktop.PointerCursor }
