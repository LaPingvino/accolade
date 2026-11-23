package main

import (
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/LaPingvino/accolade/internal/structure"
)

// The Cards tab (LOOP.md, CONTEXTUAL RIBBON TABS (1)): the ribbon's
// contextual tab while the cards are the middle tab in front, as the
// Preview tab is the pages': Rearrange (the structural moves on the
// current card), Arrange (Grid or Timeline, the timeline's lanes and
// zoom), Group (by section, colour, character or tag: a view, the script
// as it was, unless Make It the Order), Card (New Scene, its Colour, Fold
// Synopses, Open in the Script), Close. In the Ribbon look the header's
// buttons go (the tab has them); the Menus look keeps them, and has the
// same commands in a Cards menu while the cards are in front.
//
// The current card is the script's cursor's scene (or the card whose
// synopsis is being written): the cards have no selection of their own,
// so the timeline's arrows, Colour, Swap… and the strip over the editor
// all agree, and a move carries the cursor with the scene.

// cardsLook is how a script's cards are shown, kept per script.
type cardsLook struct {
	timeline bool
	group    string // "" by section, "color", "character", "tag"
	lanes    string // the timeline's: "", "characters", "tags"
}

// cardsLookPref keeps the looks: "timeline,group=color,lanes=tags\t/path",
// only where it is not the default (the grid by section).
const cardsLookPref = "cards-looks"

func (l cardsLook) String() string {
	var parts []string
	if l.timeline {
		parts = append(parts, "timeline")
	}
	if l.group != "" {
		parts = append(parts, "group="+l.group)
	}
	if l.lanes != "" {
		parts = append(parts, "lanes="+l.lanes)
	}
	return strings.Join(parts, ",")
}

func parseCardsLook(s string) cardsLook {
	var l cardsLook
	for _, p := range strings.Split(s, ",") {
		k, v, _ := strings.Cut(p, "=")
		switch k {
		case "timeline":
			l.timeline = true
		case "group":
			l.group = v
		case "lanes":
			l.lanes = v
		}
	}
	return l
}

// cardsLook is the script's look.
func (w *MainWindow) cardsLook() cardsLook {
	if w.currentFile == "" || w.settings == nil {
		return parseCardsLook(w.cardsUnsaved)
	}
	path := absPath(w.currentFile)
	for _, e := range w.settings.GetStringSlice(cardsLookPref) {
		if how, p, ok := strings.Cut(e, "\t"); ok && p == path {
			return parseCardsLook(how)
		}
	}
	return cardsLook{}
}

// setCardsLook keeps the script's look and shows it.
func (w *MainWindow) setCardsLook(l cardsLook) {
	w.keepCardsLook(w.currentFile, l)
	if w.cards != nil {
		w.cards.relook()
	}
	w.ribbonRefresh()
	w.refreshMenu()
}

// keepCardsLook keeps the look of the script at path ("": the window's).
func (w *MainWindow) keepCardsLook(path string, l cardsLook) {
	if path == "" || w.settings == nil {
		w.cardsUnsaved = l.String()
		return
	}
	path = absPath(path)
	var list []string
	if s := l.String(); s != "" {
		list = append(list, s+"\t"+path)
	}
	old := w.settings.GetStringSlice(cardsLookPref)
	for _, e := range old {
		if _, p, _ := strings.Cut(e, "\t"); p != path && len(list) < maxDocumentTypes {
			list = append(list, e)
		}
	}
	if len(list) == 0 && len(old) == 0 {
		return
	}
	w.settings.SetStringSlice(cardsLookPref, list)
	_ = w.settings.Save()
}

// cardsLookAfterRename carries the look to the script's new name.
func (w *MainWindow) cardsLookAfterRename(from, to string) {
	if to == "" {
		return
	}
	var l cardsLook
	if from == "" {
		l = parseCardsLook(w.cardsUnsaved)
		w.cardsUnsaved = ""
	} else {
		path := absPath(from)
		for _, e := range w.settings.GetStringSlice(cardsLookPref) {
			if how, p, ok := strings.Cut(e, "\t"); ok && p == path {
				l = parseCardsLook(how)
			}
		}
	}
	if l != (cardsLook{}) {
		w.keepCardsLook(to, l)
	}
}

// The Cards tab's rules.
func cardsInFront(e cmdEnv) bool     { return e.cards }
func cardsTimeline(e cmdEnv) bool    { return e.cards && e.timeline }
func cardsGrid(e cmdEnv) bool        { return e.cards && !e.timeline }
func cardsReorderable(e cmdEnv) bool { return e.grouped }

// cardsEnv fills in what the Cards tab's rules look at.
func (w *MainWindow) cardsEnv(e *cmdEnv) {
	e.cards = !w.markdown && w.cardsShown()
	if e.cards {
		l := w.cardsLook()
		e.timeline = l.timeline
		e.grouped = !l.timeline && l.group != ""
	}
}

// cardsCurrent is the current card's item: the one whose synopsis is
// being written, else the script's cursor's (-1: none).
func (w *MainWindow) cardsCurrent(items []outlineItem) int {
	if s, ok := w.fyneWindow.Canvas().Focused().(*synopsisEntry); ok && w.cards != nil && s.i < len(w.cards.items) && s.i < len(items) {
		return s.i
	}
	return w.scriptFolds.cursorItem(items)
}

// cardsAtLine is the current card's heading line while the cards are in
// front (-1: not in front, or none), for Swap… and Move….
func (w *MainWindow) cardsAtLine() int {
	if !w.cardsShown() {
		return -1
	}
	items := w.outlineItems(w.textEditor.Text())
	if i := w.cardsCurrent(items); i >= 0 {
		return items[i].Line
	}
	return -1
}

// cardsStruct makes the move op on the current card's scene or section,
// as one undo step; the cursor goes with it, so it stays the current one.
func (w *MainWindow) cardsStruct(op structure.Op) {
	e := w.textEditor
	if e == nil || w.markdown {
		return
	}
	cv := w.fyneWindow.Canvas()
	text := e.Text()
	items := w.outlineItems(text)
	i := w.cardsCurrent(items)
	if i < 0 {
		w.setStatus("No card to move: tap one to make it the current one (its scene opens), or come back to the cards")
		return
	}
	focus := cv.Focused()
	if _, ok := focus.(*synopsisEntry); ok {
		cv.Unfocus() // the cards are made again with the move
		focus = nil
	}
	line := items[i].Line
	tr, read := w.structTree(text)
	res, err := tr.Do(op, line, line, read)
	if err != nil {
		w.setStatus(err.Error())
		return
	}
	w.applyStruct(text, res, false, line, 0)
	if focus != nil && cv.Focused() != focus {
		cv.Focus(focus) // (the timeline keeps the keys)
	}
}

// cardsMoveBy moves the current card's scene one place (the timeline's
// Alt+Left and Alt+Right; Alt+Up and Alt+Down everywhere).
func (w *MainWindow) cardsMoveBy(dir int) {
	if dir < 0 {
		w.cardsStruct(structure.MoveUp)
	} else {
		w.cardsStruct(structure.MoveDown)
	}
}

// cardsSideways is Promote or Demote with the cards in front: by its key
// in the timeline laid out left to right, a move along it; else the move.
func cardsSideways(op structure.Op, dir int) func(w *MainWindow, key bool) {
	return func(w *MainWindow, key bool) {
		if key && w.cards != nil && w.cards.timelineFocused() && !w.narrow {
			w.cardsMoveBy(dir)
			return
		}
		w.cardsStruct(op)
	}
}

// openCurrentCard is Open in the Script: the current card's scene there.
func (w *MainWindow) openCurrentCard() {
	items := w.outlineItems(w.textEditor.Text())
	if i := w.cardsCurrent(items); i >= 0 {
		w.goToScript(items[i].Line)
		return
	}
	w.showMiddle("")
}

// newSceneAfterCurrent is the Cards tab's New Scene: after the current
// card's scene (at the end without one).
func (w *MainWindow) newSceneAfterCurrent() {
	items := w.outlineItems(w.textEditor.Text())
	at := len(items)
	if i := w.cardsCurrent(items); i >= 0 {
		at = i + 1 // a scene's: in its section, after it; a section's: its first
	}
	w.askNewScene(at)
}

// cardsGroups are the Group choices: the look's group, and their labels.
var cardsGroups = [][2]string{{"", "By Section"}, {"color", "By Color"}, {"character", "By Character"}, {"tag", "By Tag"}}

// cardsCommands are the Cards tab's.
func cardsCommands() []*command {
	look := func(f func(l *cardsLook)) func(w *MainWindow) {
		return func(w *MainWindow) {
			l := w.cardsLook()
			f(&l)
			w.setCardsLook(l)
		}
	}
	cs := []*command{
		{id: "cards.grid", label: "Grid", tip: "The cards in rows, grouped", ex: "cards-grid", icon: theme.GridIcon, when: cardsInFront,
			checked: func(w *MainWindow) bool { return !w.cardsLook().timeline }, run: look(func(l *cardsLook) { l.timeline = false })},
		{id: "cards.timeline", label: "Timeline", tip: "The scenes in story order, as long as they are, under their acts", ex: "cards-timeline",
			icon: res(timelineIcon), when: cardsInFront,
			checked: func(w *MainWindow) bool { return w.cardsLook().timeline }, run: look(func(l *cardsLook) { l.timeline = true })},
		{id: "cards.lanesCharacters", label: "Lanes by Character", ribbonLabel: "Characters", tip: "A lane for each who speaks: where they are, and where they are gone",
			ex: "lanes-characters", icon: theme.AccountIcon, when: cardsTimeline,
			checked: func(w *MainWindow) bool { return w.cardsLook().lanes == "characters" },
			run:     look(func(l *cardsLook) { l.lanes = toggled(l.lanes, "characters") })},
		{id: "cards.lanesTags", label: "Lanes by Tag", ribbonLabel: "Tags", tip: "A lane for each tag on the headings ([[London]]): a storyline's scenes",
			ex: "lanes-tags", icon: res(tagIcon), when: cardsTimeline,
			checked: func(w *MainWindow) bool { return w.cardsLook().lanes == "tags" },
			run:     look(func(l *cardsLook) { l.lanes = toggled(l.lanes, "tags") })},
		{id: "cards.zoomIn", label: "Zoom In on the Timeline", ribbonLabel: "Zoom In", ex: "timeline-zoom-in", icon: theme.ZoomInIcon, when: cardsTimeline,
			run: func(w *MainWindow) { w.cardsFull().timeline().zoomBy(1) }},
		{id: "cards.zoomOut", label: "Zoom Out on the Timeline", ribbonLabel: "Zoom Out", ex: "timeline-zoom-out", icon: theme.ZoomOutIcon, when: cardsTimeline,
			run: func(w *MainWindow) { w.cardsFull().timeline().zoomBy(-1) }},
		{id: "cards.makeOrder", label: "Make It the Order", tip: "Put the scenes in the order the groups show, inside each section, as one undo step",
			ex: "make-it-the-order", icon: theme.ConfirmIcon, when: cardsGrid, enabled: cardsReorderable, run: (*MainWindow).makeCardsTheOrder},
		{id: "cards.newScene", label: "New Scene…", tip: "A new scene after the current card's", ex: "new-scene", icon: theme.ContentAddIcon,
			when: cardsInFront, run: (*MainWindow).newSceneAfterCurrent},
		{id: "cards.foldSynopses", label: "Fold Synopses", tip: "The cards as their headings and how they start, without their synopses", ex: "fold-synopses",
			icon: res(foldIcon), when: cardsGrid,
			checked: func(w *MainWindow) bool { return w.settings != nil && w.settings.GetBoolean(foldSynopsesPref) },
			run: func(w *MainWindow) {
				w.changeSetting(foldSynopsesPref, func(s *Settings) { s.SetBoolean(foldSynopsesPref, !s.GetBoolean(foldSynopsesPref)) })
				if w.cards != nil {
					w.cards.relook()
				}
			}},
		{id: "cards.open", label: "Open in the Script", tip: "The current card's scene in the script", ex: "open-card", icon: theme.DocumentIcon,
			when: cardsInFront, run: (*MainWindow).openCurrentCard},
		{id: "cards.close", label: "Back to the Script", ribbonLabel: "Back to Script", icon: theme.CancelIcon, when: cardsInFront,
			run: func(w *MainWindow) { w.showMiddle("") }},
	}
	for _, g := range cardsGroups {
		value := g[0]
		name := value
		if name == "" {
			name = "section"
		}
		id := "cards.group" + strings.ToUpper(name[:1]) + name[1:]
		cs = append(cs, &command{id: id, label: "Group " + g[1], ribbonLabel: strings.TrimPrefix(g[1], "By "),
			ex: "group-" + name, icon: groupIcon(value), when: cardsGrid,
			checked: func(w *MainWindow) bool { return w.cardsLook().group == value },
			run:     look(func(l *cardsLook) { l.group = value })})
	}
	return cs
}

// toggled is v, or "" if it is on already (a lane chosen again: none).
func toggled(was, v string) string {
	if was == v {
		return ""
	}
	return v
}

// groupIcon is a Group choice's icon.
func groupIcon(value string) func() fyne.Resource {
	switch value {
	case "color":
		return theme.ColorPaletteIcon
	case "character":
		return theme.AccountIcon
	case "tag":
		return res(tagIcon)
	}
	return func() fyne.Resource { return headingIcon }
}

// foldSynopsesPref: the grid's cards without their synopses.
const foldSynopsesPref = "cards-fold-synopses"

// cardsTab is the Cards tab's layout.
func cardsTab() ribbonTab {
	return ribbonTab{title: "Cards", when: cardsInFront, contextual: true, forward: true, groups: []ribbonGroup{
		{title: "Rearrange", homes: []home{big("structure.move"), big("structure.swap"), at("structure.up"), at("structure.down"),
			at("structure.promote"), at("structure.demote")}},
		{title: "Arrange", homes: []home{big("cards.grid"), big("cards.timeline"), at("cards.lanesCharacters"), at("cards.lanesTags"),
			at("cards.zoomIn"), at("cards.zoomOut")}},
		{title: "Group", homes: []home{at("cards.groupSection"), at("cards.groupColor"), at("cards.groupCharacter"), at("cards.groupTag"),
			big("cards.makeOrder")}},
		{title: "Card", homes: []home{big("cards.newScene"), at("format.color"), at("cards.foldSynopses"), at("cards.open")}},
		{title: "Close", homes: []home{big("cards.close")}},
	}}
}

// cardsMenu is the Cards menu, while the cards are in front.
func cardsMenu() []home {
	return []home{
		at("cards.grid"), at("cards.timeline"), at("cards.lanesCharacters"), at("cards.lanesTags"), at("cards.zoomIn"), at("cards.zoomOut"),
		sep(), at("cards.groupSection"), at("cards.groupColor"), at("cards.groupCharacter"), at("cards.groupTag"), at("cards.makeOrder"),
		sep(), at("structure.move"), at("structure.swap"), at("structure.up"), at("structure.down"), at("structure.promote"), at("structure.demote"),
		sep(), at("cards.newScene"), at("format.color"), at("cards.foldSynopses"), at("cards.open"), at("cards.close"),
	}
}

// cardGroup is a group of the grid's cards: its title, colour, items.
type cardGroup struct {
	title, color string
	items        []int
}

// cardGroups are the grid's groups for a look other than by section:
// each scene in each group it is in (a character's: each scene they speak
// in), the scenes in none last.
func cardGroups(m storyModel, group string) []cardGroup {
	var out []cardGroup
	inSome := map[int]bool{}
	add := func(title, color string, scenes []int) {
		g := cardGroup{title: title, color: color}
		for _, k := range scenes {
			g.items = append(g.items, m.scenes[k].item)
			inSome[k] = true
		}
		if len(g.items) > 0 {
			out = append(out, g)
		}
	}
	none := ""
	switch group {
	case "color":
		var colors []string
		for _, s := range m.scenes {
			if c := colorName(s.color); c != "" && !slices.Contains(colors, c) {
				colors = append(colors, c)
			}
		}
		slices.SortStableFunc(colors, func(a, b string) int { return colorRank(a) - colorRank(b) })
		for _, c := range colors {
			var ks []int
			for k, s := range m.scenes {
				if colorName(s.color) == c {
					ks = append(ks, k)
				}
			}
			add(strings.ToUpper(c[:1])+c[1:], c, ks)
		}
		none = "No color"
	case "character":
		for _, c := range m.chars {
			add(c.name, "", c.scenes)
		}
		none = "Nobody speaks"
	case "tag":
		for _, t := range m.tags {
			var ks []int
			for k, s := range m.scenes {
				if slices.ContainsFunc(s.tags, func(x string) bool { return strings.EqualFold(x, t) }) {
					ks = append(ks, k)
				}
			}
			add(t, "", ks)
		}
		none = "No tag"
	}
	var rest []int
	for k := range m.scenes {
		if !inSome[k] {
			rest = append(rest, k)
		}
	}
	add(none, "", rest)
	return out
}

// colorRank is a colour's place in the picker's order (others after).
func colorRank(c string) int {
	if i := slices.Index(beatColorNames, c); i >= 0 {
		return i
	}
	return len(beatColorNames)
}

// makeCardsTheOrder is Make It the Order: the scenes as the groups show
// them, inside each section (the acts stay as they are: a scene does not
// leave its section), as one undo step through the structure engine. A
// scene in several groups (a character's, a tag's) goes where it is
// first.
func (w *MainWindow) makeCardsTheOrder() {
	l := w.cardsLook()
	if l.timeline || l.group == "" {
		w.setStatus("The cards are in the script's order already: group them first (by color, character or tag)")
		return
	}
	e := w.textEditor
	text := e.Text()
	m := w.storyNow()
	rank := map[int]int{}
	n := 0
	for _, g := range cardGroups(m, l.group) {
		for _, i := range g.items {
			if _, ok := rank[m.items[i].Line]; !ok {
				rank[m.items[i].Line] = n
				n++
			}
		}
	}
	tr, read := w.structTree(text)
	res, err := tr.Reorder(rank, read)
	if err == structure.InPlace {
		w.setStatus("The scenes are in that order already")
		return
	}
	if err != nil {
		w.setStatus(err.Error())
		return
	}
	cur, col := -1, 0
	if c := e.CursorOffset(); c >= 0 {
		cur, col = e.Buffer().Position(c)
	}
	w.applyStruct(text, res, false, cur, col)
	if len(m.bands) > 0 {
		w.setStatus(res.Msg + ", inside each section (the sections stay where they are) · Undo takes it back")
	} else {
		w.setStatus(res.Msg + " · Undo takes it back")
	}
}
