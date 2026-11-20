package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The Production pane's fourth stage, Schedule (docs/PRODUCTION.md §6 and
// §14): "18 of 87 scenes on a day · 3 days", the days (date, call, scenes,
// length; a tap opens the stripboard there), the scenes changed since they
// were scheduled ("changed in Blue: check"), and the buttons: Open
// Stripboard, Add Day, One-Liner…, Day Out of Days…, Export .ics….

// scheduleStage is the stage's buttons.
type scheduleStage struct {
	open, addDay, oneLiner, dood, ics *widget.Button
	grid                              *fyne.Container
}

func (s *scheduleStage) make(p *productionPane) {
	w := p.w
	s.open = widget.NewButtonWithIcon("Open Stripboard", theme.ListIcon(), func() { w.showStripboard(-1, -1) })
	s.open.Alignment = widget.ButtonAlignLeading
	s.open.Importance = widget.HighImportance
	s.addDay = paneButton(w, "production.addDay")
	s.oneLiner = paneButton(w, "production.oneLiner")
	s.dood = paneButton(w, "production.dood")
	s.ics = paneButton(w, "production.ics")
	s.grid = container.NewGridWithColumns(2, s.open, s.addDay, s.oneLiner, s.dood, s.ics)
}

// showScheduleStage is Production > Schedule…: the pane at its fourth
// stage.
func (w *MainWindow) showScheduleStage() {
	if w.markdown {
		return
	}
	if w.notes == nil || !w.notes.production.shown() {
		w.toggleProduction()
	}
	w.notes.production.pick(stageSchedule)
}

// showSchedule is the stage.
func (p *productionPane) showSchedule(text string) {
	w, s := p.w, &p.sched
	w.measureSchedule()
	f, bt, pl := w.scheduleFound()
	type flagged struct {
		sc, day int
		flag    stripFlag
	}
	var flags []flagged
	for d := range f.Days {
		for _, sc := range pl.scenesOf(d) {
			if fl := w.flagOf(f, bt, pl, sc); fl.changed {
				flags = append(flags, flagged{sc, d, fl})
			}
		}
	}
	p.count.SetText(scheduleCountLine(pl.count(bt), len(flags), len(pl.lost)))
	eighths := w.sceneEighths()

	var key strings.Builder
	fmt.Fprintf(&key, "schedule\x00%v\x00", eighths != nil)
	for d, day := range f.Days {
		fmt.Fprintf(&key, "%s|%s|%s|%v|", day.Date, day.Call, day.Notes, pl.scenesOf(d))
	}
	for _, x := range flags {
		fmt.Fprintf(&key, "f%d%s|", x.sc, x.flag.in)
	}
	if eighths != nil {
		fmt.Fprintf(&key, "%v", eighths)
	}
	p.setRows(key.String(), func() []fyne.CanvasObject {
		var out []fyne.CanvasObject
		if len(f.Days) == 0 {
			return []fyne.CanvasObject{note("No shoot days yet. Add Day makes one; on the stripboard each scene is a strip " +
				"in its colour (white INT. DAY, yellow EXT. DAY, blue INT. NIGHT, green EXT. NIGHT), dragged onto a day or " +
				"moved there with Move to.")}
		}
		out = append(out, caption("Shoot days"))
		for d, day := range f.Days {
			d := d
			scenes := pl.scenesOf(d)
			parts := []string{dayLabel(d, day), plural(len(scenes), "scene")}
			if eighths != nil {
				parts = append(parts, lengthText(dayLength(scenes, eighths)))
			} else {
				parts = append(parts, "measuring…")
			}
			if day.Notes != "" {
				parts = append(parts, day.Notes)
			}
			out = append(out, p.newRow(strings.Join(parts, " · "), nil, func() { w.showStripboard(d, -1) }))
		}
		if len(flags) > 0 {
			out = append(out, caption("Changed since scheduled"))
			for _, x := range flags {
				x := x
				label := fmt.Sprintf("%s (Day %d): %s", sceneName(bt, x.sc), x.day+1, strings.ToLower(x.flag.text()[:1])+x.flag.text()[1:])
				ink := revisionInk(x.flag.in)
				if x.flag.in == "" {
					ink = nil
				}
				out = append(out, p.newRow(label, ink, func() { w.showStripboard(x.day, x.sc) }))
			}
		}
		if len(pl.lost) > 0 {
			out = append(out, note(plural(len(pl.lost), "strip")+" lost: their scenes are gone (the stripboard lists them)."))
		}
		return out
	})
	if len(f.Days) == 0 {
		s.oneLiner.Disable()
		s.dood.Disable()
		s.ics.Disable()
	} else {
		s.oneLiner.Enable()
		s.dood.Enable()
		s.ics.Enable()
	}
	if p.narrow {
		p.setButtons(s.open, s.addDay, s.oneLiner, s.dood, s.ics)
	} else {
		p.setButtons(s.grid)
	}
}
