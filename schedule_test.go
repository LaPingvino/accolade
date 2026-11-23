package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	lexreport "github.com/LaPingvino/lexington/report"
)

const schedScript = `Title: The Shed

INT. KITCHEN - DAY

Anna makes tea. Rain on the window.

ANNA
Is it still raining?

TOM
Like the end of the world.

EXT. GARDEN - NIGHT

Tom runs to the shed in the dark.

TOM
Wait for me!

INT. SHED - NIGHT

Tom lights a lamp. Spiders everywhere, and a bicycle.

EXT. GARDEN - DAY

The garden in the sun. Anna hangs the washing.

ANNA
Dry at last.
`

// board is the window's stripboard, shown and measured.
func board(t *testing.T, w *MainWindow) *stripboardView {
	t.Helper()
	if !w.stripsShown() {
		w.toggleStripboard()
	}
	w.measureSchedule()
	w.stripboard.update()
	if !w.scheduleMeasured() {
		t.Fatal("not measured")
	}
	return w.stripboard
}

// days are the schedule's days as the scenes' first words: "kitchen|garden".
func days(w *MainWindow) []string {
	_, bt, p := w.scheduleFound()
	var out []string
	for d := range p.days {
		var names []string
		for _, sc := range p.scenesOf(d) {
			names = append(names, strings.ToLower(strings.Fields(bt.scenes[sc].heading)[1]))
		}
		out = append(out, strings.Join(names, ","))
	}
	return out
}

// sceneWith is the scene whose heading is h.
func sceneWith(t *testing.T, w *MainWindow, h string) int {
	t.Helper()
	_, bt, _ := w.scheduleFound()
	for i, s := range bt.scenes {
		if s.heading == h {
			return i
		}
	}
	t.Fatalf("no scene %q", h)
	return -1
}

// Strips are coloured by INT/EXT and day or night: white, green, blue,
// yellow here; the board's strips say so.
func TestStripColours(t *testing.T) {
	w, _ := scriptWindow(t, schedScript)
	sb := board(t, w)
	want := []string{"white", "green", "blue", "yellow"}
	for i, c := range want {
		st := w.stripOfScene(i)
		if st == nil {
			t.Fatalf("strip %d not measured", i)
		}
		if got, _ := st.Colour(); got != c {
			t.Errorf("scene %d: %s, want %s", i, got, c)
		}
		if s := sb.strips[i]; s == nil || !strings.Contains(s.label, c+" strip") {
			t.Errorf("scene %d's strip: %v", i, s)
		}
	}
	if st := w.stripOfScene(0); strings.Join(st.Cast, ",") != "1 TOM,2 ANNA" || st.Eighths <= 0 {
		t.Errorf("scene 1: %v %v", st.Cast, st.Eighths)
	}
}

// Days and order: strips moved onto days, up and down over the day
// breaks and back to the pool; the day totals add up.
func TestScheduleOrder(t *testing.T) {
	w, _ := scriptWindow(t, schedScript)
	sb := board(t, w)
	if c := sb.count.Text; c != "0 of 4 scenes on a day · 0 days" {
		t.Errorf("count %q", c)
	}
	w.addShootDay()
	w.addShootDay()
	sb.selectScene(3, false)
	sb.moveSelected(0, 0) // garden day on day 1
	sb.selectScene(0, false)
	sb.moveSelected(0, 0) // kitchen before it
	sb.selectScene(2, false)
	sb.moveSelected(1, 0) // the shed on day 2
	if got := strings.Join(days(w), "|"); got != "kitchen,garden|shed" {
		t.Fatalf("days %q", got)
	}
	if c := sb.count.Text; c != "3 of 4 scenes on a day · 2 days" {
		t.Errorf("count %q", c)
	}
	// down over the break: garden to day 2's start; up again
	sb.selectScene(3, false)
	sb.shift(1)
	if got := strings.Join(days(w), "|"); got != "kitchen|garden,shed" {
		t.Errorf("after down %q", got)
	}
	sb.shift(-1)
	if got := strings.Join(days(w), "|"); got != "kitchen,garden|shed" {
		t.Errorf("after up %q", got)
	}
	// the pool's top up: onto the last day's end
	if sb.pool[0] != 1 {
		t.Fatalf("pool %v", sb.pool)
	}
	sb.selectScene(1, false)
	sb.shift(-1)
	if got := strings.Join(days(w), "|"); got != "kitchen,garden|shed,garden" {
		t.Errorf("from the pool %q", got)
	}
	// the last day's last down: back to the pool; Undo puts it back
	sb.shift(1)
	if len(sb.pool) != 1 {
		t.Errorf("pool %v", sb.pool)
	}
	w.undoSchedule()
	if got := strings.Join(days(w), "|"); got != "kitchen,garden|shed,garden" {
		t.Errorf("undone %q", got)
	}
	// the totals
	e := w.sceneEighths()
	_, _, p := w.scheduleFound()
	want := e[0] + e[3]
	if dayLength(p.scenesOf(0), e) != want {
		t.Errorf("day 1: %v", dayLength(p.scenesOf(0), e))
	}
	found := false
	for _, o := range sb.list.Objects {
		found = found || strings.Contains(textOf(o), "End of Day 1 — "+lengthText(want))
	}
	if !found {
		t.Errorf("no day break with %s", lengthText(want))
	}
	// Move To: the choice (the tab's menu, and the strip's)
	sb.selectScene(0, false)
	menuItemNamed(t, sb.moveItems(0), "Day 2").Action()
	if got := strings.Join(days(w), "|"); got != "garden|shed,garden,kitchen" {
		t.Errorf("moved to day 2: %q", got)
	}
	// removing a day: its strips to the pool
	sb.setDay(0, "", "", "", true)
	if got := strings.Join(days(w), "|"); got != "shed,garden,kitchen" || len(sb.pool) != 1 {
		t.Errorf("day removed: %q, pool %v", got, sb.pool)
	}
}

// menuItemNamed is the item labelled label among items.
func menuItemNamed(t *testing.T, items []*fyne.MenuItem, label string) *fyne.MenuItem {
	t.Helper()
	for _, it := range items {
		if it.Label == label {
			return it
		}
	}
	t.Fatalf("no %q in the menu", label)
	return nil
}

// textOf is the texts in an object, its children's too.
func textOf(o fyne.CanvasObject) string {
	var b strings.Builder
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		switch x := o.(type) {
		case *fyne.Container:
			for _, k := range x.Objects {
				walk(k)
			}
		case *widget.Label:
			b.WriteString(x.Text)
		case *canvas.Text:
			b.WriteString(x.Text)
		}
	}
	walk(o)
	return b.String()
}

// Group by location and by cast sort one day, once; the pool's sort is
// kept as its own.
func TestScheduleGroup(t *testing.T) {
	w, _ := scriptWindow(t, schedScript)
	sb := board(t, w)
	w.addShootDay()
	for _, sc := range []int{0, 1, 2, 3} {
		sb.selectScene(sc, false)
		sb.moveSelected(0, 9)
	}
	sb.selectScene(0, false)
	sb.group("location")
	if got := days(w)[0]; got != "kitchen,garden,garden,shed" {
		t.Errorf("by location %q", got)
	}
	sb.group("cast") // Tom's (1) alone first, then Tom and Anna's, Anna's, nobody's
	if got := days(w)[0]; got != "garden,kitchen,garden,shed" {
		t.Errorf("by cast %q", got)
	}
	w.undoSchedule()
	if got := days(w)[0]; got != "kitchen,garden,garden,shed" {
		t.Errorf("undone %q", got)
	}
}

// Strips keep their scenes through edits: a scene put before, the
// numbers locked, a heading changed; a scene cut leaves its strip lost.
func TestScheduleAnchors(t *testing.T) {
	w, _ := scriptWindow(t, schedScript)
	sb := board(t, w)
	w.addShootDay()
	sb.selectScene(sceneWith(t, w, "EXT. GARDEN - DAY"), false)
	sb.moveSelected(0, 0)
	sb.selectScene(sceneWith(t, w, "INT. SHED - NIGHT"), false)
	sb.moveSelected(0, 1)
	check := func(what string) {
		t.Helper()
		_, bt, p := w.scheduleFound()
		var got []string
		for _, sc := range p.scenesOf(0) {
			got = append(got, bt.sceneStart(sc))
		}
		if strings.Join(got, "|") != "the garden in the sun anna hangs the|tom lights a lamp spiders everywhere and a" {
			t.Errorf("%s: %q", what, got)
		}
	}
	// a garden scene in the day put before
	replaceText(t, w, "INT. KITCHEN - DAY", "EXT. GARDEN - DAY\n\nMorning dew.\n\nINT. KITCHEN - DAY")
	check("a scene put before")
	w.lockSceneNumbers()
	check("numbers locked")
	if !strings.Contains(w.textEditor.Text(), "#5#") {
		t.Fatal("not numbered")
	}
	// renumbered by the writer: the heading and words still tell it
	replaceText(t, w, "INT. SHED - NIGHT #4#", "INT. SHED - NIGHT #40#")
	check("renumbered")
	replaceText(t, w, "INT. SHED - NIGHT", "INT. OLD SHED - NIGHT")
	check("heading changed")
	// stored after the lock: the number, then the heading too changed
	_, _, p := w.scheduleFound()
	sb.selectScene(p.scenesOf(0)[0], false)
	sb.shift(1)
	sb.shift(-1)
	replaceText(t, w, "EXT. GARDEN - DAY #5#", "EXT. BACK GARDEN - DAY #5#")
	check("numbered, heading changed")
	// the shed cut: lost, kept in the file
	text := w.textEditor.Text()
	i, j := strings.Index(text, "INT. OLD SHED"), strings.Index(text, "EXT. BACK GARDEN")
	w.textEditor.SetText(text[:i] + text[j:])
	sb.update()
	_, _, p = w.scheduleFound()
	if len(p.lost) != 1 || len(p.scenesOf(0)) != 1 || !strings.Contains(sb.count.Text, "1 lost strip") {
		t.Errorf("lost %v, day %v, %q", p.lost, p.scenesOf(0), sb.count.Text)
	}
	if f := w.scheduleNow(); len(f.Days[0].Strips) != 2 {
		t.Errorf("the lost strip not kept: %+v", f.Days[0].Strips)
	}
	// a move on the day keeps it
	sb.selectScene(p.scenesOf(0)[0], false)
	sb.moveSelected(0, 0)
	if f := w.scheduleNow(); len(f.Days[0].Strips) != 2 {
		t.Errorf("the lost strip dropped by a move: %+v", f.Days[0].Strips)
	}
}

// Two strips of one scene: the second is left out.
func TestScheduleTwice(t *testing.T) {
	w, _ := scriptWindow(t, schedScript)
	board(t, w)
	_, bt, _ := w.scheduleFound()
	f := w.scheduleNow()
	f.Days = []schedDay{{ID: "a", Strips: []schedStrip{bt.stripOf(0, "")}}, {ID: "b", Strips: []schedStrip{bt.stripOf(0, ""), bt.stripOf(1, "")}}}
	p := f.place(bt)
	if p.dups != 1 || p.dayOf[0] != 0 || p.dayOf[1] != 1 || len(p.lost) != 0 {
		t.Errorf("%+v", p)
	}
	f.prune(bt)
	if len(f.Days[1].Strips) != 1 {
		t.Errorf("not pruned: %+v", f.Days[1].Strips)
	}
}

// A scene changed after it was scheduled is flagged, with the revision
// it changed in, whatever the margin shows; Checked takes the flag away.
func TestScheduleRevisionFlag(t *testing.T) {
	w, _ := scriptWindow(t, schedScript)
	issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	replaceText(t, w, "Spiders everywhere", "Spiders everywhere, webs") // changed before scheduling
	sb := board(t, w)
	w.addShootDay()
	shed := sceneWith(t, w, "INT. SHED - NIGHT")
	sb.selectScene(shed, false)
	sb.moveSelected(0, 0)
	sb.selectScene(0, false)
	sb.moveSelected(0, 1)
	f, bt, p := w.scheduleFound()
	if w.flagOf(f, bt, p, shed).changed {
		t.Error("flagged for a change before it was scheduled")
	}
	issueNow(t, w, "Blue", time.Date(2026, 11, 5, 0, 0, 0, 0, time.Local))
	replaceText(t, w, "Anna makes tea.", "Anna makes coffee.")
	issueNow(t, w, "Pink", time.Date(2026, 11, 9, 0, 0, 0, 0, time.Local))
	w.updateRevisionMarks()
	w.schedule.measuredFor = "" // measured again
	sb.update()
	f, bt, p = w.scheduleFound()
	fl := w.flagOf(f, bt, p, 0)
	if !fl.changed || fl.in != "Pink" || fl.text() != "Changed in Pink: check" {
		t.Errorf("kitchen: %+v", fl)
	}
	if !strings.Contains(sb.strips[0].label, "Changed in Pink") || !strings.Contains(sb.count.Text, "1 changed scene to check") {
		t.Errorf("strip %q, count %q", sb.strips[0].label, sb.count.Text)
	}
	sb.selectScene(0, false)
	if !stripFlagged(w.cmdEnv()) {
		t.Error("Checked is greyed")
	}
	cmd("strips.checked").run(w)
	f, bt, p = w.scheduleFound()
	if w.flagOf(f, bt, p, 0).changed {
		t.Error("still flagged after Checked")
	}
	if w.flagOf(f, bt, p, shed).changed {
		t.Error("the shed flagged")
	}
	// a later change, before any issue of it: the open set's name
	replaceText(t, w, "Tom lights a lamp.", "Tom lights two lamps.")
	w.schedule.measuredFor = ""
	sb.update()
	f, bt, p = w.scheduleFound()
	if fl := w.flagOf(f, bt, p, shed); fl.in != "Yellow" {
		t.Errorf("shed: %+v", fl)
	}
}

// The file: schedule.json in the kept copy, read again; never written
// over when it cannot be read.
func TestScheduleStored(t *testing.T) {
	w, p := scriptWindow(t, schedScript)
	sb := board(t, w)
	w.addShootDay()
	sb.selectScene(1, false)
	sb.moveSelected(0, 0)
	sb.setDay(0, "2026-12-01", "7:00", "Back garden, Ljubljana", false)
	h := w.extrasHome()
	data, err := h.readExtra(scheduleName)
	if err != nil {
		t.Fatal(err)
	}
	var f scheduleFile
	if err := json.Unmarshal(data, &f); err != nil || f.Version != 1 || len(f.Days) != 1 || f.Days[0].Call != "07:00" ||
		f.Days[0].Date != "2026-12-01" || len(f.Days[0].Strips) != 1 || f.Days[0].Strips[0].Heading != "ext garden night" ||
		f.Days[0].Strips[0].Sum == "" || f.Days[0].Strips[0].Start == "" {
		t.Fatalf("%v %s", err, data)
	}
	w.NewFile()
	if len(w.scheduleNow().Days) != 0 {
		t.Error("a new script has the last one's days")
	}
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if got := days(w); len(got) != 1 || got[0] != "garden" {
		t.Errorf("read again: %v", got)
	}
	if next := w.scheduleNow(); next.newDay() != 1 || next.Days[1].Date != "2026-12-02" {
		t.Errorf("the next day's date %q", next.Days[1].Date)
	}
	w.schedule.key = "" // that day was not written: read again
	h.dir.write([]bundleFile{{scheduleName, []byte("{not json")}})
	w.breakdownGen++
	w.addShootDay()
	if data, _ := h.readExtra(scheduleName); string(data) != "{not json" {
		t.Errorf("written over: %s", data)
	}
}

// One-liner and DOOD as the reports show them, against golden CSVs; the
// .ics of the days with dates.
func TestScheduleReports(t *testing.T) {
	w, _ := scriptWindow(t, schedScript)
	sb := board(t, w)
	w.addShootDay()
	w.addShootDay()
	for _, m := range []struct{ sc, d int }{{2, 0}, {1, 0}, {3, 1}} {
		sb.selectScene(m.sc, false)
		sb.moveSelected(m.d, 9)
	}
	sb.setDay(0, "2026-12-01", "07:00", "The shed; the garden", false)
	sb.setDay(1, "2026-12-02", "", "", false)
	t.Cleanup(func() { w.settings.SetString("report-kind", lexreport.KindScenes) })
	for _, kind := range []string{lexreport.KindOneLiner, lexreport.KindDOOD} {
		w.lastReports = nil
		w.showReportsOf(kind)
		waitFor(t, "the reports", func() bool { return w.lastReports != nil })
		rd := w.lastReports
		if rd.kindOf() != kind {
			t.Fatalf("kind %s", rd.kindOf())
		}
		var b bytes.Buffer
		if err := rd.table.WriteCSV(&b); err != nil {
			t.Fatal(err)
		}
		golden := filepath.Join("testdata", "schedule", kind+".csv")
		if os.Getenv("UPDATE_GOLDEN") != "" {
			os.MkdirAll(filepath.Dir(golden), 0o755)
			os.WriteFile(golden, b.Bytes(), 0o644)
		} else if want, err := os.ReadFile(golden); err != nil || !bytes.Equal(b.Bytes(), want) {
			t.Errorf("%s:\n%s\nwant\n%s (%v)", kind, b.String(), want, err)
		}
		if !strings.Contains(rd.summary.Text, "2 shoot days, 3 scenes on them.") {
			t.Errorf("summary %q", rd.summary.Text)
		}
		rd.dialog.Hide()
	}
	ics, skipped, err := w.icsOf(w.schedule.rep)
	if err != nil || skipped != 0 {
		t.Fatalf("%v %d", err, skipped)
	}
	s := string(ics)
	for _, want := range []string{"BEGIN:VCALENDAR\r\n", "DTSTART:20261201T070000\r\n", "DTSTART;VALUE=DATE:20261202\r\n",
		"SUMMARY:story: shoot day 2 of 2\r\n", "LOCATION:The shed\\; the garden\r\n", "@accolade.kiefte.eu\r\n"} {
		if !strings.Contains(s, want) {
			t.Errorf("no %q in\n%s", want, s)
		}
	}
}

// The Schedule stage: its count, its days, its buttons; it becomes the
// script's stage with the first day.
func TestScheduleStage(t *testing.T) {
	w, _ := scriptWindow(t, schedScript)
	p := prodPane(t, w)
	test.Tap(p.stages[stageSchedule])
	if p.view() != stageSchedule {
		t.Fatal("not shown")
	}
	if got := p.count.Text; got != "0 of 4 scenes on a day · 0 days" {
		t.Errorf("count %q", got)
	}
	if !p.sched.ics.Disabled() || len(p.buttons.Objects) != 1 {
		t.Error("the stage's buttons")
	}
	test.Tap(p.sched.addDay)
	if p.stage != stageSchedule {
		t.Errorf("stage %d", p.stage)
	}
	sb := board(t, w)
	sb.selectScene(0, false)
	sb.moveSelected(0, 0)
	w.toggleStripboard()
	p.refresh()
	if got := p.count.Text; got != "1 of 4 scenes on a day · 1 day" {
		t.Errorf("count %q", got)
	}
	rows := rowLabels(p)
	if len(rows) != 1 || !strings.HasPrefix(rows[0].Text, "Day 1 · Sat 21 Nov 2026") && !strings.HasPrefix(rows[0].Text, "Day 1 · 1 scene") {
		var ts []string
		for _, r := range rows {
			ts = append(ts, r.Text)
		}
		t.Errorf("rows %q", ts)
	}
	rows[0].onTap()
	if !w.stripsShown() {
		t.Error("a day's row does not open the stripboard")
	}
	w.toggleStripboard()
	test.Tap(p.sched.open)
	if !w.stripsShown() {
		t.Error("Open Stripboard")
	}
	w.toggleStripboard()
	if p.sched.ics.Disabled() || p.sched.dood.Disabled() {
		t.Error("the reports greyed with a day")
	}
	menuItemByLabel(t, w, "Stripboard")
	menuItemByLabel(t, w, "Export Shoot Days (.ics)…")
}

// The keyboard: Up and Down select, Alt+Up and Alt+Down move.
func TestStripboardKeys(t *testing.T) {
	w, _ := scriptWindow(t, schedScript)
	sb := board(t, w)
	w.addShootDay()
	sb.selectScene(0, false)
	sb.moveSelected(0, 0)
	sb.selectScene(1, false)
	sb.moveSelected(0, 1)
	c := w.fyneWindow.Canvas()
	c.Focus(sb.strips[0])
	sb.strips[0].TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	if sb.selected != 1 || c.Focused() != fyne.Focusable(sb.strips[1]) {
		t.Errorf("Down: selected %d", sb.selected)
	}
	sb.strips[1].TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyUp, Modifier: fyne.KeyModifierAlt})
	if got := days(w)[0]; got != "garden,kitchen" {
		t.Errorf("Alt+Up: %q", got)
	}
	if sb.selected != 1 || c.Focused() != fyne.Focusable(sb.strips[1]) {
		t.Error("the moved strip lost the focus")
	}
}

// A drag drops a strip where its middle is: on a strip's lower half,
// after it; on a day's break, at its end.
func TestStripboardDrag(t *testing.T) {
	w, _ := scriptWindow(t, schedScript)
	w.fyneWindow.Resize(fyne.NewSize(1100, 650))
	sb := board(t, w)
	w.addShootDay()
	sb.selectScene(0, false)
	sb.moveSelected(0, 0)
	middle := func(o fyne.CanvasObject, lower bool) fyne.Position {
		at := fyne.CurrentApp().Driver().AbsolutePositionForObject(o)
		y := o.Size().Height / 4
		if lower {
			y *= 3
		}
		return at.Add(fyne.NewPos(o.Size().Width/2, y))
	}
	// the shed (in the pool) onto the kitchen's lower half: after it
	shed := sb.strips[2]
	sb.dropped(2, shed, middle(sb.strips[0], true))
	if got := days(w)[0]; got != "kitchen,shed" {
		t.Errorf("dropped after: %q", got)
	}
	// the garden onto the kitchen's upper half: before it
	sb.dropped(1, sb.strips[1], middle(sb.strips[0], false))
	if got := days(w)[0]; got != "garden,kitchen,shed" {
		t.Errorf("dropped before: %q", got)
	}
	// the garden onto the shed's lower half: to the end
	sb.dropped(1, sb.strips[1], middle(sb.strips[2], true))
	if got := days(w)[0]; got != "kitchen,shed,garden" {
		t.Errorf("dropped down: %q", got)
	}
	// onto the pool's head: unscheduled
	var poolHead fyne.CanvasObject
	for _, tg := range sb.targets {
		if tg.day < 0 && !tg.halves {
			poolHead = tg.obj
		}
	}
	sb.dropped(0, sb.strips[0], middle(poolHead, false))
	if got := days(w)[0]; got != "shed,garden" {
		t.Errorf("unscheduled: %q", got)
	}
}

// On a phone the board is a list of two-line strips, no wider than the
// window, no toolbar over them (the ribbon's tab has its commands); a
// drag scrolls it, the strip's grip moves it, a long press shows its
// actions; the side panes' sheet is folded down under it.
func TestStripboardNarrow(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, schedScript)
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	sb := board(t, w)
	w.addShootDay()
	sb.selectScene(0, false)
	sb.moveSelected(0, 0)
	if mw := sb.box.MinSize().Width; mw > 380 {
		t.Errorf("the board is at least %v wide", mw)
	}
	if !sb.narrow || sb.box.Objects[0] != fyne.CanvasObject(sb.scroll) || len(sb.box.Objects) != 1 {
		t.Error("something over the strips")
	}
	if got := w.fyneWindow.Canvas().Capture().Bounds().Dx(); got != 380 {
		t.Errorf("the window is %d wide", got)
	}
	// a drag scrolls rather than moves
	sb.strips[0].Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DY: -40}})
	sb.strips[0].DragEnd()
	if got := days(w)[0]; got != "kitchen" {
		t.Errorf("moved by a drag on a phone: %q", got)
	}
	// the grip moves it: the garden (unscheduled) dragged by it onto the
	// kitchen's lower half, after it
	garden := sb.strips[1]
	if garden.grip == nil || sb.strips[0].grip == nil {
		t.Fatal("no grip on a phone's strip")
	}
	d := fyne.CurrentApp().Driver()
	from, to := d.AbsolutePositionForObject(garden), d.AbsolutePositionForObject(sb.strips[0])
	dy := to.Y + sb.strips[0].Size().Height*3/4 - (from.Y + garden.Size().Height/2)
	garden.grip.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DY: dy}})
	garden.grip.DragEnd()
	if got := days(w)[0]; got != "kitchen,garden" {
		t.Errorf("dragged by its grip: %q", got)
	}
	// a long press: the strip's actions; the tap after it is no tap
	var fire func()
	oldAfter := longPressAfter
	longPressAfter = func(_ time.Duration, f func()) *time.Timer {
		fire = f
		return time.NewTimer(time.Hour)
	}
	t.Cleanup(func() { longPressAfter = oldAfter })
	c := w.fyneWindow.Canvas()
	s := sb.strips[0]
	s.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary})
	fire()
	if c.Overlays().Top() == nil || sb.selected != 0 {
		t.Fatalf("no menu after a long press (selected %d)", sb.selected)
	}
	for c.Overlays().Top() != nil {
		c.Overlays().Remove(c.Overlays().Top())
	}
	s.Tapped(&fyne.PointEvent{})
	if s.pressed {
		t.Error("the long press's tap not swallowed")
	}
	m := sb.stripMenu(0)
	if l := m.Items[0].Label; l != "Move Up" || !m.Items[0].Disabled {
		t.Errorf("the strip's menu: %q (disabled %v): the day's first", l, m.Items[0].Disabled)
	}
	menuItemNamed(t, m.Items, "Unschedule").Action()
	if got := days(w)[0]; got != "garden" {
		t.Errorf("unscheduled from its menu: %q", got)
	}
	// a day folds to its band: a strip dropped on it goes to its end
	sb.fold(0, true)
	if sb.strips[1] != nil || !strings.Contains(textOf(sb.list), "1 scene · ") {
		t.Errorf("folded: its strips shown, or no band: %q", textOf(sb.list))
	}
	var band fyne.CanvasObject
	for _, tg := range sb.targets {
		if tg.day == 0 {
			band = tg.obj
		}
	}
	at := d.AbsolutePositionForObject(band)
	sb.dropped(0, sb.strips[0], at.AddXY(10, band.Size().Height/2))
	if got := days(w)[0]; got != "garden,kitchen" || sb.strips[0] == nil {
		t.Errorf("dropped on the folded day: %q (unfolded to show it: %v)", got, sb.strips[0] != nil)
	}
	// the side panes' sheet folded down under the board; a pane shown
	// brings it up
	if !w.sheetDown {
		t.Error("the sheet is up under the board")
	}
	p := prodPane(t, w)
	if w.sheetDown || !p.shown() {
		t.Error("the sheet stayed down for a pane shown")
	}
	p.pick(stageSchedule)
	if p.box.MinSize().Width > sourceWidth {
		t.Errorf("the pane is at least %v wide", p.box.MinSize().Width)
	}
	if len(p.buttons.Objects) != 5 {
		t.Errorf("%d buttons on a phone", len(p.buttons.Objects))
	}
}

// Everything Tab reaches on the board shows its focus.
func TestStripboardFocusVisible(t *testing.T) {
	holdUI(t)
	if raceDetector {
		t.Skip("pixel capture of focus: checked by the runs without -race")
	}
	w, _ := scriptWindow(t, schedScript)
	w.fyneWindow.Resize(fyne.NewSize(1000, 900))
	sb := board(t, w)
	w.addShootDay()
	sb.selectScene(0, false)
	sb.moveSelected(0, 0)
	sb.selectScene(-1, false)
	if u := invisibleFocus(w.fyneWindow.Canvas()); len(u) > 0 {
		t.Errorf("the focus is not seen on\n  %s", strings.Join(u, "\n  "))
	}
	if u := unnamed(w.fyneWindow.Canvas()); len(u) > 0 {
		t.Errorf("no name for\n  %s", strings.Join(u, "\n  "))
	}
}

// SHOT_DIR renders the stripboard and the Schedule stage, wide and on a
// phone, and the DOOD's PDF.
func TestScheduleScreenshots(t *testing.T) {
	holdUI(t)
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	w, _ := scriptWindow(t, schedScript)
	issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	sb := board(t, w)
	w.addShootDay()
	w.addShootDay()
	for _, m := range []struct{ sc, d int }{{2, 0}, {1, 0}, {0, 1}} {
		sb.selectScene(m.sc, false)
		sb.moveSelected(m.d, 9)
	}
	sb.setDay(0, "2026-12-01", "18:00", "The shed and the garden, Kovač house", false)
	sb.setDay(1, "2026-12-02", "07:00", "", false)
	replaceText(t, w, "Anna makes tea.", "Anna makes coffee.")
	w.updateRevisionMarks()
	sb.update()
	sb.selectScene(1, true)
	comboShot(t, w, "stripboard")
	w.toggleStripboard()
	p := prodPane(t, w)
	p.pick(stageSchedule)
	p.refresh()
	comboShot(t, w, "schedule-stage")
	var b bytes.Buffer
	rep := *w.schedule.rep
	if err := lexreport.WriteDOODPDF(&b, "The Shed", rep, w.schedule.breakdown, w.lexScheduleFor(&rep), ""); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "schedule-dood.pdf"), b.Bytes(), 0o644)
	b.Reset()
	lexreport.WriteOneLinerPDF(&b, "The Shed", rep, w.schedule.breakdown, w.lexScheduleFor(&rep), lexreport.StripOptions{}, "")
	os.WriteFile(filepath.Join(dir, "schedule-oneliner.pdf"), b.Bytes(), 0o644)
}

// SHOT_DIR renders the stripboard in the Ribbon look, wide and on a
// phone, the outline in the side panes (a phone's sheet folded down under
// the board); then with its second day folded.
func TestStripboardRibbonScreenshots(t *testing.T) {
	holdUI(t)
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	w, _ := scriptWindow(t, schedScript)
	ribbonOnFor(t, w)
	w.togglePane("outline")
	sb := board(t, w)
	w.addShootDay()
	w.addShootDay()
	for _, m := range []struct{ sc, d int }{{2, 0}, {1, 0}, {0, 1}} {
		sb.selectScene(m.sc, false)
		sb.moveSelected(m.d, 9)
	}
	sb.setDay(0, "2026-12-01", "18:00", "The shed and the garden, Kovač house", false)
	sb.setDay(1, "2026-12-02", "07:00", "", false)
	sb.update()
	sb.selectScene(1, true)
	comboShot(t, w, "stripboard-ribbon")
	sb.fold(1, true)
	comboShot(t, w, "stripboard-folded")
}

// In an .accolade the schedule is in the bundle, and read again after an
// issue (which reads the extras again).
func TestScheduleInBundle(t *testing.T) {
	w := draftsWindow(t)
	p := filepath.Join(t.TempDir(), "Script.accolade")
	os.WriteFile(p, makeZip(t, bundleItems("Script.textbundle/")), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	sb := board(t, w)
	w.addShootDay()
	sb.selectScene(0, false)
	sb.moveSelected(0, 0)
	b, _, err := readBundle(p)
	if err != nil {
		t.Fatal(err)
	}
	if data, err := b.readExtra(scheduleName); err != nil || !strings.Contains(string(data), `"heading": "int kitchen day"`) {
		t.Errorf("in the bundle: %v %s", err, data)
	}
	w.toggleStripboard()
	issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	if got := days(w); len(got) != 1 || got[0] != "kitchen" {
		t.Errorf("after the issue: %v", got)
	}
}

// Grouping the unscheduled is undone by a tap on their head; a silent
// cast member tagged shows on the strip; the menu's Alt+Up/Down keys
// move a strip with the focus.
func TestStripboardPoolCastKeys(t *testing.T) {
	w, _ := scriptWindow(t, schedScript)
	sb := board(t, w)
	sb.group("location")
	if w.scheduleNow().Pool != "location" || sb.pool[0] != 0 || sb.pool[1] != 3 || sb.pool[2] != 1 { // the gardens together, day first
		t.Errorf("pool %v", sb.pool)
	}
	var head *tapLabel
	for _, o := range sb.list.Objects {
		if l, ok := o.(*tapLabel); ok && strings.Contains(l.Text, "tap for script order") {
			head = l
		}
	}
	if head == nil {
		t.Fatal("no head to tap")
	}
	head.onTap()
	if w.scheduleNow().Pool != "" || sb.pool[1] != 1 || sb.pool[2] != 2 {
		t.Errorf("script order: %v", sb.pool)
	}
	w.toggleStripboard()
	tagNow(t, w, "Spiders", 0, "Cast Members")
	sb = board(t, w)
	if st := w.stripOfScene(2); st == nil || len(st.Cast) != 1 || !strings.HasSuffix(st.Cast[0], "SPIDERS") {
		t.Errorf("the shed's cast %v", st)
	}
	w.addShootDay()
	sb.selectScene(0, false)
	sb.moveSelected(0, 0)
	sb.selectScene(1, false)
	sb.moveSelected(0, 1)
	w.fyneWindow.Canvas().Focus(sb.strips[1])
	menuItemByLabel(t, w, "Move Up").Action()
	if got := days(w)[0]; got != "garden,kitchen" {
		t.Errorf("Alt+Up through the menu: %q", got)
	}
}

// A scheduled scene's words are the page's: a source link or a note put
// in or edited is no change, nor a strip summed when the words had them
// (before they were left out); a word of the script is.
func TestScheduleSumSkipsNotes(t *testing.T) {
	text := "INT. KITCHEN - NIGHT\n\n[[from: \"Anna opened the door.\"]]\n\nAnna opens the door. [[slower]]\n"
	bt := readBdText(text, nil)
	f := scheduleFile{Days: []schedDay{{Strips: []schedStrip{bt.stripOf(0, "")}}}}
	old := scheduleFile{Days: []schedDay{{Strips: []schedStrip{bt.stripOf(0, "")}}}}
	old.Days[0].Strips[0].Sum = wordsSum(bt.scenes[0].key.Heading, wordsIn(bt.runes, 0, len(bt.runes)))
	if bt.sceneStart(0) != "anna opens the door" {
		t.Errorf("start %q", bt.sceneStart(0))
	}
	changed := func(f scheduleFile, text string) bool {
		bt := readBdText(text, nil)
		c, _ := f.changed(bt, f.place(bt), 0)
		return c
	}
	if changed(old, text) {
		t.Error("summed with the notes, as before: changed")
	}
	for _, edit := range []string{text, strings.Replace(text, "Anna opened", "She opened", 1), strings.Replace(text, "slower", "faster", 1)} {
		if changed(f, edit) {
			t.Errorf("changed: %q", edit)
		}
	}
	if edit := strings.Replace(text, "opens the door", "opens the window", 1); !changed(f, edit) {
		t.Error("a word of the script: not changed")
	}
}
