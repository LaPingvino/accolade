package main

import (
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

const tabScript = `# ACT ONE

INT. BARN - DAY [[red]] [[London]]

ANNA
It leaks.

EXT. FIELD - DAY [[blue]]

Wind over the field, a long time, the grass bending and bending again until the light goes.

The grass bends. The grass bends again. Far off a tractor turns at the end of its row and comes back.

A crow lifts, drops, lifts. The wind has nowhere to go and goes everywhere, through the fence, the hedge, the barn's open door.

Nobody comes. The light goes down behind the hill, slowly, as if it had all day.

INT. HOUSE - NIGHT [[red]]
= Anna decides.

# ACT TWO

EXT. ROAD - DAY

BOB
Where is she?

INT. BARN - NIGHT [[blue]] [[London]]
`

// The story as the timeline reads it: lengths, the empty scene and the
// one with only its synopsis, the acts over their scenes, who speaks
// where, the tags, who has been gone.
func TestStoryOf(t *testing.T) {
	m := storyOf(tabScript, nil, nil)
	if len(m.scenes) != 5 || len(m.bands) != 2 {
		t.Fatalf("%d scenes, %d bands", len(m.scenes), len(m.bands))
	}
	if b := m.bands[0]; b.first != 0 || b.last != 2 {
		t.Errorf("act one holds %d..%d", b.first, b.last)
	}
	if b := m.bands[1]; b.first != 3 || b.last != 4 {
		t.Errorf("act two holds %d..%d", b.first, b.last)
	}
	if s := m.scenes[2]; !s.empty || !s.synopsisOnly {
		t.Errorf("the house: empty %v, synopsis only %v", s.empty, s.synopsisOnly)
	}
	if s := m.scenes[4]; !s.empty || s.synopsisOnly {
		t.Errorf("the last barn: empty %v, synopsis only %v", s.empty, s.synopsisOnly)
	}
	if m.scenes[1].eighths < m.scenes[0].eighths {
		t.Errorf("the field (%d) shorter than the barn (%d)", m.scenes[1].eighths, m.scenes[0].eighths)
	}
	if len(m.chars) != 2 || m.chars[slices.IndexFunc(m.chars, func(c storyChar) bool { return c.name == "ANNA" })].scenes[0] != 0 {
		t.Fatalf("characters: %+v", m.chars)
	}
	if !strings.EqualFold(strings.Join(m.scenes[4].tags, ","), "London") || len(m.tags) != 1 {
		t.Errorf("tags: %v / %v", m.scenes[4].tags, m.tags)
	}
	if m.current(0) != -1 || m.current(2) != 0 || m.current(1000) != 4 {
		t.Errorf("current: %d %d %d", m.current(0), m.current(2), m.current(1000))
	}
	if g := m.goneFor(4, 3); len(g) != 1 || g[0].name != "ANNA" || g[0].scenes != 4 {
		t.Errorf("gone: %+v", g)
	}
}

// cardsWindow is a window with the cards in front, in the Ribbon look.
func cardsWindow(t *testing.T) (*MainWindow, *ribbonBar) {
	t.Helper()
	w, _ := newDialogTestWindow(t)
	w.fyneWindow.Resize(fyne.NewSize(1100, 700))
	r := ribbonOnFor(t, w)
	w.textEditor.SetText(tabScript)
	w.textEditor.SetCursorOffset(strings.Index(tabScript, "EXT. FIELD"))
	w.toggleCards()
	w.ribbonRefresh()
	return w, r
}

// The Cards tab comes forward with the cards and goes with them; its
// moves work on the current card (the cursor's scene), one undo step
// each, and the header's buttons are the tab's in the Ribbon look.
func TestCardsTab(t *testing.T) {
	w, r := cardsWindow(t)
	if r.selected != "Cards" {
		t.Fatalf("the Cards tab not in front: %s", r.selected)
	}
	if w.cards.tools.Visible() {
		t.Error("the header's buttons shown in the Ribbon look")
	}
	if w.cards.lit < 0 || w.cards.items[w.cards.lit].Text != "EXT. FIELD - DAY" {
		t.Errorf("the current card is not the cursor's: %d", w.cards.lit)
	}
	buttonOf(r, "structure.down").press()
	got := w.textEditor.Text()
	if strings.Index(got, "INT. HOUSE") > strings.Index(got, "EXT. FIELD") {
		t.Fatalf("Move Down did not move the field past the house:\n%s", got)
	}
	if w.cards.items[w.cards.lit].Text != "EXT. FIELD - DAY" {
		t.Error("the moved card is not the current one")
	}
	w.undo()
	if w.textEditor.Text() != tabScript {
		t.Fatal("one undo did not take the move back")
	}
	// Alt+Right (Demote's key) in the grid demotes, here: into the next act
	w.do(cmd("structure.demote"))()
	if got := w.textEditor.Text(); strings.Index(got, "EXT. FIELD") < strings.Index(got, "# ACT TWO") {
		t.Errorf("Demote did not take the field into act two:\n%s", got)
	}
	w.undo()
	w.showMiddle("")
	w.ribbonRefresh()
	if r.selected == "Cards" {
		t.Error("the Cards tab stayed with the script in front")
	}
}

// Grouped by colour the grid is a view (dragging does not move); Make It
// the Order puts the scenes so, inside each act, as one undo step.
func TestCardsGroupAndOrder(t *testing.T) {
	w, r := cardsWindow(t)
	buttonOf(r, "cards.groupColor").press()
	if !w.cards.grouped {
		t.Fatal("not grouped")
	}
	if h := w.cards.list.Objects[0].(*fyne.Container); len(h.Objects) == 0 {
		t.Fatal("no group header")
	}
	if w.textEditor.Text() != tabScript {
		t.Fatal("grouping changed the script")
	}
	buttonOf(r, "cards.makeOrder").press()
	got := w.textEditor.Text()
	// act one: red (barn, house) before blue (field); act two: blue (barn) before none (road)
	order := []string{"INT. BARN - DAY", "INT. HOUSE - NIGHT", "EXT. FIELD - DAY", "# ACT TWO", "INT. BARN - NIGHT", "EXT. ROAD - DAY"}
	at := -1
	for _, o := range order {
		i := strings.Index(got, o)
		if i < at {
			t.Fatalf("%s out of order:\n%s", o, got)
		}
		at = i
	}
	w.undo()
	if w.textEditor.Text() != tabScript {
		t.Fatal("one undo did not take the order back")
	}
	// kept for the script
	if l := w.cardsLook(); l.group != "color" {
		t.Errorf("the look: %+v", l)
	}
}

// The timeline: a block for each scene, as wide as it is long; the arrows
// go along it with the script's cursor, Alt+Right moves the scene along,
// Enter opens it.
func TestTimeline(t *testing.T) {
	w, r := cardsWindow(t)
	buttonOf(r, "cards.timeline").press()
	tl := w.cards.tl
	if !w.cardsLook().timeline || len(w.cards.body.Objects) != 1 || w.cards.body.Objects[0] != tl {
		t.Fatal("not the timeline")
	}
	if len(tl.blocks) != 5 {
		t.Fatalf("%d blocks", len(tl.blocks))
	}
	if tl.lit != 1 {
		t.Errorf("lit %d, want the field (1)", tl.lit)
	}
	if b, f := tl.blocks[0].Size().Width, tl.blocks[1].Size().Width; f <= b {
		t.Errorf("the field's block (%v) not wider than the barn's (%v)", f, b)
	}
	if tl.blocks[0].Position().Y < tlBand { // the bands above
		t.Error("the blocks not under the bands")
	}
	w.fyneWindow.Canvas().Focus(tl)
	tl.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	if line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset()); line != tl.m.scenes[2].line || tl.lit != 2 {
		t.Errorf("Right: the cursor on line %d, lit %d", line, tl.lit)
	}
	tl.TypedKey(&fyne.KeyEvent{Name: fyne.KeyLeft})
	w.do(cmd("structure.demote"))() // Alt+Right, in the timeline: along it
	got := w.textEditor.Text()
	if strings.Index(got, "EXT. FIELD") < strings.Index(got, "INT. HOUSE") || strings.Index(got, "EXT. FIELD") > strings.Index(got, "# ACT TWO") {
		t.Fatalf("Alt+Right did not move the field one along:\n%s", got)
	}
	w.undo()
	// lanes by character: ANNA's and BOB's
	buttonOf(r, "cards.lanesCharacters").press()
	if lanes := tl.laneList(); len(lanes) != 2 || lanes[0].in[0] == lanes[1].in[0] || lanes[0].in[3] == lanes[1].in[3] {
		t.Errorf("lanes: %+v", lanes)
	}
	z := tl.scale
	tl.zoomBy(1)
	if tl.scale <= z {
		t.Error("no zoom")
	}
	tl.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if !w.scriptInMain() {
		t.Error("Enter did not open the scene")
	}
}

// The timeline strip over the editor: off at first; on, it lights the
// cursor's scene, says where that is, and a tap goes to a scene.
func TestTimelineStrip(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.fyneWindow.Resize(fyne.NewSize(1000, 650))
	w.textEditor.SetText(tabScript)
	if w.stripShown() {
		t.Fatal("the strip on at first")
	}
	w.do(cmd("view.timelineStrip"))()
	if !w.stripShown() || w.strip == nil || !w.strip.Visible() {
		t.Fatal("the strip not shown")
	}
	s := w.strip
	s.box.Refresh()
	s.drawnW = -1
	s.box.Resize(fyne.NewSize(800, 40))
	if len(s.blocks) != 5 {
		t.Fatalf("%d blocks", len(s.blocks))
	}
	w.textEditor.SetCursorOffset(strings.Index(tabScript, "Where is"))
	if s.lit != 3 || !strings.Contains(s.caption.Text, "Scene 4 of 5") || !strings.Contains(s.caption.Text, "ACT TWO") {
		t.Errorf("lit %d, caption %q", s.lit, s.caption.Text)
	}
	b := s.blocks[1]
	s.Tapped(&fyne.PointEvent{Position: s.bar.Position().Add(b.Position()).Add(fyne.NewPos(b.Size().Width/2, 2))})
	if line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset()); line != s.m.scenes[1].line {
		t.Errorf("tap: the cursor on line %d", line)
	}
	w.do(cmd("view.timelineStrip"))()
	if w.stripShown() {
		t.Error("the strip still on")
	}
}

// The Cards tab with the grid, the timeline with lanes, the strip over
// the editor, on a desktop and a phone (SHOT_DIR as TestScreenshots).
func TestCardsTabScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"desktop", fyne.NewSize(1200, 720)}, {"phone", fyne.NewSize(380, 800)}} {
		w, _ := newDialogTestWindow(t)
		ribbonOnFor(t, w)
		w.openSample()
		w.fyneWindow.Resize(size.size)
		w.onWindowWidth(size.size.Width)
		w.setCardsLook(cardsLook{})
		w.textEditor.SetCursorOffset(strings.Index(w.textEditor.Text(), "Noon heat."))
		snap := func(name string) {
			t.Helper()
			w.ribbonRefresh()
			w.fyneWindow.Content().Refresh()
			f, err := os.Create(filepath.Join(dir, "cards-"+size.name+"-"+name+".png"))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
				t.Fatal(err)
			}
		}
		w.toggleCards()
		snap("1-grid")
		l := w.cardsLook()
		l.timeline, l.lanes = true, "characters"
		w.setCardsLook(l)
		snap("2-timeline-lanes")
		w.showMiddle("")
		w.do(cmd("view.timelineStrip"))()
		w.textEditor.SetCursorOffset(strings.Index(w.textEditor.Text(), "Noon heat."))
		snap("3-strip")
		w.do(cmd("view.timelineStrip"))()
	}
}
