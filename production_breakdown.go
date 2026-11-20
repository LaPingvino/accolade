package main

import (
	"fmt"
	"image/color"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/lex"
	lexreport "github.com/LaPingvino/lexington/report"
)

// The Production pane's third stage, Break down (docs/PRODUCTION.md §6 and
// §13): "62% broken down · 33 scenes to go", the scene at the cursor with
// its eighths, its cast (who speaks, from the cues, and who is tagged)
// and its location (from the heading), its elements by category, the
// elements tagged in earlier scenes whose words it has (a tap tags them
// here), and the tags lost anywhere (their words gone: never dropped by
// themselves). Next Scene goes to the first scene not marked done; Mark
// Scene Done marks it. The eighths and cast come from the reports, made
// in the background once per text while the stage is shown.

// breakdownStage is the stage's buttons and what it last showed.
type breakdownStage struct {
	next, done, tag, sheets *widget.Button
	grid                    *fyne.Container // the buttons in two columns, beside the script

	scene      int // the scene shown (-1: none)
	report     *lexreport.Report
	reportFor  string // the text the report is of
	pendingFor string // the text a report is being made of
	reportRun  int
}

func (b *breakdownStage) make(p *productionPane) {
	w := p.w
	b.next = paneButton(w, "production.nextScene")
	b.next.Importance = widget.HighImportance
	b.done = paneButton(w, "production.sceneDone")
	b.tag = paneButton(w, "production.tag")
	b.sheets = paneButton(w, "production.sheets")
	b.grid = container.NewGridWithColumns(2, b.next, b.done, b.tag, b.sheets)
	b.scene = -1
}

// productionShowsBreakdown tells whether the pane shows Break down.
func (w *MainWindow) productionShowsBreakdown() bool {
	return w.notes != nil && w.notes.production.shown() && w.notes.production.view() == stageBreakdown
}

// showBreakdownStage is Production > Break Down…: the pane at its third
// stage.
func (w *MainWindow) showBreakdownStage() {
	if w.markdown {
		return
	}
	if w.notes == nil || !w.notes.production.shown() {
		w.toggleProduction()
	}
	w.notes.production.pick(stageBreakdown)
}

// breakdownFollowCursor shows the scene the cursor went to, when the
// stage is shown and the tags are found for the text as it is (typing:
// the pause's refresh brings it).
func (w *MainWindow) breakdownFollowCursor() {
	if !w.productionShowsBreakdown() || w.breakdown.bt == nil || !sameString(w.breakdown.text, w.textEditor.Text()) {
		return
	}
	if w.breakdown.bt.sceneAt(w.textEditor.CursorOffset()) != w.notes.production.bd.scene {
		w.notes.production.refresh()
	}
}

// showBreakdown is the stage.
func (p *productionPane) showBreakdown(text string) {
	w, b := p.w, &p.bd
	f, bt, ps := w.breakdownNowFound()
	prog := f.progress(bt)
	lost := 0
	for _, x := range ps {
		if x.lost {
			lost++
		}
	}
	var parts []string
	switch {
	case prog.total == 0:
		parts = append(parts, "No scenes to break down yet")
	case prog.done == prog.total:
		parts = append(parts, "100% broken down", plural(prog.total, "scene"))
	default:
		parts = append(parts, fmt.Sprintf("%d%% broken down", prog.percent()), plural(prog.total-prog.done, "scene")+" to go")
	}
	if lost > 0 {
		parts = append(parts, plural(lost, "lost tag"))
	}
	p.count.SetText(strings.Join(parts, " · "))

	if (b.report == nil || !sameString(b.reportFor, text)) && !sameString(b.pendingFor, text) {
		p.makeReport(text) // once a text
	}
	sc := bt.sceneAt(w.textEditor.CursorOffset())
	if sc < 0 && len(bt.scenes) > 0 {
		sc = 0
	}
	b.scene = sc

	var rs *lexreport.Scene
	numbers := map[string]int{}
	if b.report != nil && sameString(b.reportFor, text) && sc >= 0 {
		if k := reportScenes(bt, b.report)[sc]; k >= 0 {
			rs = &b.report.Scenes[k]
		}
		numbers = b.report.CastNumbers(reportBreakdownFor(f, ps, bt, b.report))
	}
	var key strings.Builder
	fmt.Fprintf(&key, "breakdown\x00%d\x00%v\x00%d\x00", sc, rs != nil, len(numbers))
	if sc >= 0 {
		s := bt.scenes[sc]
		fmt.Fprintf(&key, "%s\x00%s\x00%v\x00", s.label, s.heading, f.isDone(s))
	}
	if rs != nil {
		fmt.Fprintf(&key, "%s\x00%s\x00", rs.Eighths, strings.Join(rs.Characters, ","))
	}
	for _, x := range ps {
		fmt.Fprintf(&key, "%s\x00%d\x00%d\x00%v\x00", x.tag.Element, x.start, x.scene, x.lost)
	}
	p.setRows(key.String(), func() []fyne.CanvasObject { return p.breakdownRows(f, bt, ps, sc, rs, numbers) })

	if sc >= 0 && f.isDone(bt.scenes[sc]) {
		b.done.SetText("Mark Scene Not Done")
		b.done.SetIcon(theme.CancelIcon())
	} else {
		b.done.SetText("Mark Scene Done")
		b.done.SetIcon(theme.ConfirmIcon())
	}
	if sc < 0 || bt.scenes[sc].omitted {
		b.done.Disable()
	} else {
		b.done.Enable()
	}
	if prog.next < 0 {
		b.next.Disable()
	} else {
		b.next.Enable()
	}
	if p.narrow { // a phone: one under the other, at the list's end
		p.setButtons(b.next, b.done, b.tag, b.sheets)
	} else {
		p.setButtons(b.grid)
	}
}

// breakdownRows are the stage's list: the scene, its cast and location,
// its elements by category, the suggestions, the lost tags.
func (p *productionPane) breakdownRows(f *breakdownFile, bt *bdText, ps []placed, sc int, rs *lexreport.Scene, numbers map[string]int) []fyne.CanvasObject {
	w := p.w
	if sc < 0 {
		return []fyne.CanvasObject{note("No scenes yet: a breakdown lists each scene's cast and elements.")}
	}
	s := bt.scenes[sc]
	title := "Scene " + s.label
	if f.isDone(s) {
		title += " · done"
	}
	out := []fyne.CanvasObject{caption(title)}
	head := s.heading
	if rs != nil {
		head += " · " + rs.Eighths.String()
		if rs.Eighths.Pages() == 0 {
			head += " page"
		} else {
			head += " pages"
		}
	} else {
		head += " · measuring…"
	}
	out = append(out, p.newRow(head, nil, func() { w.goToLine(s.line) }))
	if s.omitted {
		return append(out, note("OMITTED: nothing to break down."))
	}
	inkOf := func(c lex.TagCategory) color.Color {
		if r, g, bl, ok := parseHexRGB(c.Color); ok {
			return color.NRGBA{R: r, G: g, B: bl, A: 0xff}
		}
		return nil
	}
	if rs != nil && rs.Location != "" {
		loc, _ := lex.TagCategoryOf("Location")
		where := rs.Location
		if more := strings.Join(slices.DeleteFunc([]string{rs.Place, rs.Time}, func(x string) bool { return x == "" }), ", "); more != "" {
			where += " (" + more + ")"
		}
		out = append(out, p.newRow("Location: "+where, inkOf(loc),
			func() { w.goToLine(s.line) }))
	}

	// the cast: who speaks (the cues), then who is tagged
	cast, _ := lex.TagCategoryOf(lex.CastCategory)
	var castRows []fyne.CanvasObject
	seen := map[string]bool{}
	if rs != nil {
		for _, name := range rs.Characters {
			seen[name] = true
			label := name + " (speaks)"
			if n := numbers[name]; n > 0 {
				label = strconv.Itoa(n) + " " + label
			}
			castRows = append(castRows, p.newRow(label, inkOf(cast), func() { w.goToLine(s.line) }))
		}
	}
	type group struct {
		cat  lex.TagCategory
		rows []fyne.CanvasObject
	}
	groups := map[string]*group{}
	var order []string
	tagged := map[string]bool{} // the elements tagged in the scene
	for k := range ps {
		x := ps[k]
		if x.lost || x.scene != sc {
			continue
		}
		tagged[x.element.ID] = true
		c, _ := f.category(x.element.Category)
		if c.Name == lex.CastCategory {
			name := strings.ToUpper(x.element.Name)
			if seen[name] {
				continue
			}
			seen[name] = true
			label := name
			if n := numbers[name]; n > 0 {
				label = strconv.Itoa(n) + " " + name
			}
			castRows = append(castRows, p.newRow(label, inkOf(cast), func() { w.selectRange(x.start, x.end) }))
			continue
		}
		g := groups[c.Name]
		if g == nil {
			g = &group{cat: c}
			groups[c.Name] = g
			order = append(order, c.Name)
		}
		g.rows = append(g.rows, p.newRow(x.element.Name, inkOf(c), func() { w.selectRange(x.start, x.end) }))
	}
	out = append(out, caption("Cast Members"))
	if len(castRows) == 0 {
		out = append(out, note("Nobody speaks here: tag a silent cast member with Tag Selection…, Cast Members."))
	}
	out = append(out, castRows...)
	rank := map[string]int{}
	for i, c := range f.categories() {
		rank[c.Name] = i
	}
	slices.SortStableFunc(order, func(a, b string) int { return rank[a] - rank[b] })
	for _, name := range order {
		out = append(out, caption(name))
		out = append(out, groups[name].rows...)
	}
	if len(order) == 0 {
		out = append(out, note("No elements tagged here yet: select words (a prop, a vehicle, wardrobe) and Tag Selection…."))
	}

	// suggestions: elements tagged elsewhere whose words this scene has
	var sugg []fyne.CanvasObject
	lo, hi := bt.sceneWords(sc)
	for _, e := range f.Elements {
		if tagged[e.ID] {
			continue
		}
		c, _ := f.category(e.Category)
		if c.Name == lex.CastCategory {
			continue
		}
		want := normWords(e.Name)
		for i := lo; len(want) > 0 && i+len(want) <= hi; i++ {
			if bt.wordsAt(i, want) {
				i, j := i, i+len(want)
				e := e
				sugg = append(sugg, p.newRow("Tag “"+e.Name+"” here ("+c.Name+")", inkOf(c), func() { w.tagWordsAs(i, j, e.Category) }))
				break
			}
		}
	}
	if len(sugg) > 0 {
		out = append(out, caption("Tagged in other scenes, found here"))
		out = append(out, sugg...)
	}

	// lost tags, anywhere
	var lostRows []fyne.CanvasObject
	for _, x := range ps {
		if !x.lost {
			continue
		}
		x := x
		where := x.tag.Scene
		if where == "" {
			where = strings.ToUpper(x.tag.Heading)
		}
		lostRows = append(lostRows, p.newRow(fmt.Sprintf("Lost: “%s” (%s), scene %s: its words are gone", x.words, x.element.Category, where),
			nil, func() { w.removeLostTag(x.tag) }))
	}
	if len(lostRows) > 0 {
		out = append(out, caption("Lost tags (a tap removes one)"))
		out = append(out, lostRows...)
	}
	return out
}

// makeReport makes the reports of text in the background (the scenes'
// eighths and cast), then shows the stage again.
func (p *productionPane) makeReport(text string) {
	b := &p.bd
	b.reportRun++
	run := b.reportRun
	b.pendingFor = text
	job := p.w.savedExportJob()
	type made struct {
		r   lexreport.Report
		err error
	}
	inBackground(func() made {
		r, err := scriptReport(text, job)
		return made{r, err}
	}, func(m made) {
		if run != b.reportRun {
			return
		}
		b.report, b.reportFor, b.pendingFor = &m.r, text, ""
		if m.err != nil {
			b.report = &lexreport.Report{}
		}
		p.refresh()
	})
}

// selectRange selects runes [start, end) of the script and puts the focus
// there.
func (w *MainWindow) selectRange(start, end int) {
	w.textEditor.Select(start, end)
	w.fyneWindow.Canvas().Focus(w.textEditor)
}

// tagWordsAs tags words [i, j) of the text as found now as category.
func (w *MainWindow) tagWordsAs(i, j int, category string) {
	_, bt, _ := w.breakdownNowFound()
	if j > len(bt.words) {
		return
	}
	w.textEditor.Select(bt.words[i].start, bt.words[j-1].end)
	w.tagSelectionAs(category)
}

// nextBreakdownScene is Next Scene: the cursor to the first scene not
// marked done, its cast and location shown.
func (w *MainWindow) nextBreakdownScene() {
	f, bt, _ := w.breakdownNowFound()
	prog := f.progress(bt)
	if prog.next < 0 {
		w.setStatus("Every scene is broken down")
		return
	}
	s := bt.scenes[prog.next]
	w.goToLine(s.line)
	w.setStatus(fmt.Sprintf("Scene %s: %d%% broken down, %s to go", s.label, prog.percent(), plural(prog.total-prog.done, "scene")))
	w.productionFollow()
}

// markSceneDone marks a scene done (or, if it is, not done).
func (w *MainWindow) markSceneDone(sc int) {
	f, bt, _ := w.breakdownNowFound()
	if sc < 0 || sc >= len(bt.scenes) {
		return
	}
	s := bt.scenes[sc]
	done := !f.isDone(s)
	f.setDone(s, done)
	w.saveBreakdown()
	if done {
		w.setStatus("Scene " + s.label + " broken down")
	} else {
		w.setStatus("Scene " + s.label + " not done")
	}
}

// markSceneDoneHere is Mark Scene Done: the scene the Break down stage
// shows, else the cursor's.
func (w *MainWindow) markSceneDoneHere() {
	if w.markdown || w.textEditor == nil {
		return
	}
	if w.productionShowsBreakdown() {
		w.markSceneDone(w.notes.production.bd.scene)
		return
	}
	_, bt, _ := w.breakdownNowFound()
	if bt == nil {
		return
	}
	w.markSceneDone(bt.sceneAt(w.textEditor.CursorOffset()))
}

// removeLostTag asks, then takes a lost tag away.
func (w *MainWindow) removeLostTag(t bdTag) {
	dialog.ShowConfirm("Remove the Lost Tag?", "Its words are no longer in the script. Remove the tag? (Undo does not bring it back.)",
		func(ok bool) {
			if !ok {
				return
			}
			f := w.breakdownNow()
			for k, x := range f.Tags {
				if x == t {
					f.Tags = slices.Delete(f.Tags, k, k+1)
					break
				}
			}
			f.pruneElements()
			w.saveBreakdown()
		}, w.fyneWindow)
}
