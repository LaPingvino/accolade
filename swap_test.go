package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// tabTitles are the dock's tab titles as shown, in order.
func tabTitles(w *MainWindow) string {
	var out []string
	for _, p := range w.dock.visible(dockSheet) {
		if t := w.dock.tabs[p.id]; t != nil {
			out = append(out, t.title)
		}
	}
	return strings.Join(out, " ")
}

// runeAt is the rune offset of s in text.
func runeAt(t *testing.T, text, s string) int {
	t.Helper()
	i := strings.Index(text, s)
	if i < 0 {
		t.Fatalf("no %q in %q", s, text)
	}
	return len([]rune(text[:i]))
}

// The source swapped into the main editor: a Markdown document there,
// its commands, its own undo, find and replace in it, the title naming
// it; File > Save writes it beside the script; swapping back puts the
// script back, the source written.
func TestSwapSourceIntoMain(t *testing.T) {
	w, p := sourceWindow(t, sourceScript, sourceStory)
	script := w.textEditor.Text()
	w.swapWithMain("source")
	d := w.mainDoc
	if w.swapped != "source" || d == nil || d.kind != docSource || w.mainEditor() != d.ed || d.Text() != sourceStory {
		t.Fatalf("swapped %q, main %v", w.swapped, w.mainEditor() == w.textEditor)
	}
	if w.swappedCenter() != d.view {
		t.Error("the source is not in the main area")
	}
	if got := tabTitles(w); !strings.Contains(got, "Script") || strings.Contains(got, "Source") {
		t.Errorf("tabs %q: the Script tab in the source's place", got)
	}
	env := w.cmdEnv()
	if !cmd("md.heading2").applies(env) || cmd(elementCommandID("Action")).applies(env) || cmd("format.underline").applies(env) {
		t.Error("the main editor holds Markdown: its commands, not the screenplay's")
	}
	if !cmd("source.choose").applies(env) || env.markdown {
		t.Error("the file is still a screenplay: Adaptation stays")
	}
	if !w.editorMarkdown() || !strings.Contains(w.statsLabel.Text, "Source") {
		t.Errorf("status %q", w.statsLabel.Text)
	}
	// a heading, through the command as the menu gives it
	w.fyneWindow.Canvas().Focus(d.ed)
	d.ed.SetCursorOffset(runeAt(t, d.Text(), "The next morning"))
	w.do(cmd("md.heading2"))()
	if !strings.Contains(d.Text(), "## The next morning") || w.textEditor.Text() != script {
		t.Fatalf("the heading went into the script, or nowhere: %q", d.Text())
	}
	if !strings.HasPrefix(w.fyneWindow.Title(), "• story.md — ") {
		t.Errorf("title %q", w.fyneWindow.Title())
	}
	// Undo is the source's own
	w.undo()
	if d.Text() != sourceStory || w.textEditor.Text() != script {
		t.Error("Undo did not undo in the source")
	}
	// Find and Replace in the source
	w.showFindReplace()
	search(w, "garden")
	w.searchBar.replaceEntry.SetText("orchard")
	w.replaceAll()
	if strings.Contains(d.Text(), "garden") || !strings.Contains(w.textEditor.Text(), "garden") {
		t.Errorf("replaced in the wrong text: %q", d.Text())
	}
	w.hideFindReplace()
	if data, _ := os.ReadFile(filepath.Join(filepath.Dir(p), "story.md")); string(data) != sourceStory {
		t.Error("written before Save")
	}
	if !w.unsavedDocs() {
		t.Error("the close question would not ask")
	}
	if err := w.SaveFile(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(filepath.Dir(p), "story.md"))
	if string(data) != d.Text() || w.unsavedDocs() || strings.HasPrefix(w.fyneWindow.Title(), "•") {
		t.Errorf("after Save: file %q, title %q", data, w.fyneWindow.Title())
	}
	d.ed.Edit(func(b *buffer.Buffer) { b.SetCursor(b.Len(), false); b.Insert("\nThe end.\n") })
	w.swapWithMain("source") // back
	if w.swapped != "" || w.mainEditor() != w.textEditor || w.srcDoc != nil || w.swappedCenter() == d.view {
		t.Fatal("not back")
	}
	data, _ = os.ReadFile(filepath.Join(filepath.Dir(p), "story.md"))
	if !strings.HasSuffix(string(data), "The end.\n") {
		t.Errorf("not written on the way back: %q", data)
	}
	if got := tabTitles(w); !strings.Contains(got, "Source") || strings.Contains(got, "Script") {
		t.Errorf("tabs %q after", got)
	}
	if env := w.cmdEnv(); !cmd(elementCommandID("Action")).applies(env) || cmd("md.heading2").applies(env) {
		t.Error("the screenplay's commands are not back")
	}
	if s := w.sourcePane(); !strings.Contains(s.src, "The end.") {
		t.Error("the pane does not show the source as written")
	}
}

// A source kept in the script: each change in the main editor goes into
// the script's block at once, all of them one undo step there; a change
// made to the block in the script (Undo, a co-writer) comes into the
// source's editor outside its own history.
func TestSwapSourceInsideScript(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.toggleSourceInside()
	if w.sourceRef().Origin != sourceInside {
		t.Fatal("not inside")
	}
	w.swapWithMain("source")
	d := w.mainDoc
	if !d.inScript() || d.sharedAs() != "script" {
		t.Fatal("kept in the script: shared with it")
	}
	steps := w.textEditor.UndoSteps()
	for _, s := range []string{"One.", " Two.", " Three."} {
		d.ed.Edit(func(b *buffer.Buffer) { b.SetCursor(b.Len(), false); b.Insert(s) })
	}
	if _, _, md, _ := sourceBlock(w.textEditor.Text()); !strings.HasSuffix(strings.TrimSpace(md), "One. Two. Three.") {
		t.Fatalf("the block %q", md)
	}
	if got := w.textEditor.UndoSteps(); got != steps+1 {
		t.Errorf("%d script steps for three edits, want one", got-steps)
	}
	// the block changed in the script: here too, the source's own Undo kept
	text := w.textEditor.Text()
	i := strings.Index(text, "/* Source")
	i += strings.Index(text[i:], "She opened") // in the block, not the script's link
	w.textEditor.Edit(func(b *buffer.Buffer) { b.Remote(len([]rune(text[:i])), len([]rune(text[:i])), "Then ") })
	if !strings.Contains(d.Text(), "Then She opened") {
		t.Fatalf("the source did not follow the script: %q", d.Text())
	}
	d.ed.Undo()
	if strings.Contains(d.Text(), "Three.") || !strings.Contains(d.Text(), "Then She opened") {
		t.Errorf("the source's Undo: %q", d.Text())
	}
	w.swapBack()
	before := w.textEditor.Text()
	w.fyneWindow.Canvas().Focus(w.textEditor)
	w.undo()
	if w.textEditor.Text() == before || strings.Contains(w.textEditor.Text(), "One.") {
		t.Errorf("the script's Undo did not take the source's writing back as one: %q", w.textEditor.Text())
	}
}

// The notes in the main editor: written into the script's notes block,
// the Notes pane following; in a Markdown document as its HTML comment.
func TestSwapNotes(t *testing.T) {
	w := newTestWindow(t, "INT. BARN - DAY\n\nRain.\n\n/* Notes\nfirst\n*/\n")
	w.showPane("notes", false)
	w.swapWithMain("notes")
	d := w.mainDoc
	if d == nil || d.kind != docNotes || d.Text() != "first" || !strings.Contains(tabTitles(w), "Script") {
		t.Fatalf("notes %v", d)
	}
	d.ed.Edit(func(b *buffer.Buffer) { b.SetCursor(b.Len(), false); b.Insert(" and more") })
	if notesOf(w.textEditor.Text()) != "first and more" {
		t.Errorf("the script's notes %q", notesOf(w.textEditor.Text()))
	}
	w.swapBack()
	if w.notesPane().entry.Text != "first and more" {
		t.Errorf("the Notes pane %q", w.notesPane().entry.Text)
	}

	m := mdWindow(t, "# Story\n\nText.\n\n<!-- Notes\nmine\n-->\n", 0, 0)
	m.showPane("notes", false)
	m.swapWithMain("notes")
	if m.mainDoc == nil || m.mainDoc.Text() != "mine" {
		t.Fatal("the Markdown document's notes")
	}
	m.mainDoc.ed.Edit(func(b *buffer.Buffer) { b.SetCursor(b.Len(), false); b.Insert("!") })
	if !strings.HasSuffix(m.textEditor.Text(), "<!-- Notes\nmine!\n-->\n") {
		t.Errorf("Markdown's notes block %q", m.textEditor.Text())
	}
	if m.swapBlocked("source") == "" || m.swapBlocked("cards") == "" {
		t.Error("a Markdown document has no source or cards to swap")
	}
}

// Swapping is refused, with a reason, where it cannot be.
func TestSwapBlocked(t *testing.T) {
	w := newTestWindow(t, "INT. BARN - DAY\n")
	w.showPane("source", false)
	w.swapWithMain("source")
	if w.swapped != "" || !strings.Contains(w.statusLabel.Text, "No source") {
		t.Errorf("swapped %q, status %q", w.swapped, w.statusLabel.Text)
	}
}

// Another script in the window: the source written first, the script
// back in the main editor; the same for the document type.
func TestSwapLeftForAnotherDocument(t *testing.T) {
	w, p := sourceWindow(t, sourceScript, sourceStory)
	w.swapWithMain("source")
	w.mainDoc.ed.Edit(func(b *buffer.Buffer) { b.SetCursor(0, false); b.Insert("New first line.\n\n") })
	other := filepath.Join(t.TempDir(), "other.fountain")
	os.WriteFile(other, []byte("INT. ELSEWHERE - DAY\n"), 0o644)
	if err := w.LoadFile(other); err != nil {
		t.Fatal(err)
	}
	if w.swapped != "" || w.srcDoc != nil || w.mainEditor() != w.textEditor {
		t.Error("still swapped")
	}
	if data, _ := os.ReadFile(filepath.Join(filepath.Dir(p), "story.md")); !strings.HasPrefix(string(data), "New first line.") {
		t.Errorf("the source not written: %q", data)
	}

	n := newTestWindow(t, "INT. BARN - DAY\n\n/* Notes\nkept\n*/\n")
	n.showPane("notes", false)
	n.swapWithMain("notes")
	n.setDocumentType(true)
	if n.swapped != "" || !strings.HasSuffix(n.textEditor.Text(), "<!-- Notes\nkept\n-->\n") {
		t.Errorf("swapped %q, %q", n.swapped, n.textEditor.Text())
	}
}

// Auto-save writes the source in the main editor with the script.
func TestSwapAutoSave(t *testing.T) {
	w, p := sourceWindow(t, sourceScript, sourceStory)
	w.settings.SetBoolean("auto-save", true)
	t.Cleanup(func() { w.settings.SetBoolean("auto-save", false) })
	w.swapWithMain("source")
	w.mainDoc.ed.Edit(func(b *buffer.Buffer) { b.SetCursor(b.Len(), false); b.Insert("Saved by itself.\n") })
	w.autoSaveNow()
	if data, _ := os.ReadFile(filepath.Join(filepath.Dir(p), "story.md")); !strings.HasSuffix(string(data), "Saved by itself.\n") {
		t.Errorf("not auto-saved: %q", data)
	}
}

// A view swapped in (the outline): its box in the main area, nothing to
// write in (the editing commands say so), the Script tab lighting the
// scene the outline goes to; the script's cursor commands say why not.
func TestSwapView(t *testing.T) {
	w := newTestWindow(t, "INT. BARN - DAY\n\nRain.\n\nEXT. FIELD - NIGHT\n\nWind.\n")
	w.showPane("outline", false)
	w.swapWithMain("outline")
	if w.swappedCenter() != w.outlinePane().box || w.mainEditor() != nil || !w.paneInSight("outline") {
		t.Fatal("the outline is not in the main area")
	}
	if cmd(elementCommandID("Action")).applies(w.cmdEnv()) || cmd("md.heading1").applies(w.cmdEnv()) {
		t.Error("no editor: no element commands")
	}
	before := w.textEditor.Text()
	w.tapped(cmd("format.bold"), nil)()
	if w.textEditor.Text() != before || !strings.Contains(w.statusLabel.Text, "Nothing to write in") {
		t.Errorf("bold with no editor: status %q", w.statusLabel.Text)
	}
	w.outlinePane().goTo(4)
	sp := w.scriptPane()
	if !sp.lit[1] || sp.lit[0] {
		t.Errorf("the Script tab lights %v", sp.lit)
	}
	w.omitScene()
	if w.textEditor.Text() != before || !strings.Contains(w.statusLabel.Text, "Swap with Main") {
		t.Errorf("omit with the script away: %q", w.statusLabel.Text)
	}
	w.togglePane("outline") // its View command: back in its place
	if w.swapped != "" || w.swappedCenter() == w.outlinePane().box {
		t.Error("View > Outline did not put it back")
	}
}

// The Script tab follows the source's cursor: the scenes linking to its
// passage lit, the passage's status said; a tap goes to the scene's
// passage, and the script's cursor to the scene.
func TestScriptTabFollowsSource(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.swapWithMain("source")
	d := w.mainDoc
	sp := w.scriptPane()
	d.ed.SetCursorOffset(runeAt(t, d.Text(), "The next morning"))
	var lit []string
	for i := range sp.items {
		if sp.lit[i] {
			lit = append(lit, sp.items[i].Text)
		}
	}
	if strings.Join(lit, ",") != "EXT. GARDEN - DAY" || !strings.Contains(sp.info.Text, "1 scene") {
		t.Errorf("lit %v, info %q", lit, sp.info.Text)
	}
	d.ed.SetCursorOffset(runeAt(t, d.Text(), "a list item"))
	if !strings.Contains(sp.info.Text, "not adapted yet") {
		t.Errorf("info %q", sp.info.Text)
	}
	for i, it := range sp.items {
		if it.Text == "INT. HOUSE - NIGHT" {
			sp.tap(i)
		}
	}
	line, _ := d.ed.Buffer().Position(d.ed.CursorOffset())
	if got := strings.Split(d.Text(), "\n")[line]; !strings.HasPrefix(got, "She opened the door") {
		t.Errorf("the source's cursor on %q", got)
	}
	sl, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
	if got := strings.Split(w.textEditor.Text(), "\n")[sl]; got != "INT. HOUSE - NIGHT" {
		t.Errorf("the script's cursor on %q", got)
	}
}

// The passages' lines are found by their opening words: headings, list
// items, a quote, after the front matter.
func TestPassageLines(t *testing.T) {
	ps := passagesOf(sourceStory)
	lines := passageLines(sourceStory, ps)
	src := strings.Split(sourceStory, "\n")
	for i, p := range ps {
		if lines[i] < 0 {
			t.Errorf("passage %d %q not found", i, p.Plain)
			continue
		}
		if !strings.Contains(normLine(src[lines[i]]), strings.Join(p.Words[:min(3, len(p.Words))], " ")) {
			t.Errorf("passage %d %q on line %q", i, p.Plain, src[lines[i]])
		}
	}
}

func normLine(s string) string { return strings.Join(normWords(s), " ") }

// Editing together: the notes and a source kept in the script are the
// script's text, so they go both ways while swapped in; a source kept
// elsewhere is this writer's alone, as the status says.
func TestSwapTogether(t *testing.T) {
	_, _, block := withSourceBlock("INT. BARN - DAY\n\nRain.\n", "First passage here.\n\nSecond passage here.")
	text := "INT. BARN - DAY\n\nRain.\n" + block
	a, b, _ := sharedPair(t, text)
	connect(t, a, b)
	textIs(t, b, "b", text)
	a.showPane("source", false)
	a.swapWithMain("source")
	d := a.mainDoc
	if d == nil || !d.inScript() {
		t.Fatal("the source kept in the shared script is not in a's main editor")
	}
	d.ed.Edit(func(b *buffer.Buffer) { b.SetCursor(b.Len(), false); b.Insert(" Typed by a.") })
	waitFor(t, "b has a's typing in the source", func() bool { return strings.Contains(b.textEditor.Text(), "Typed by a.") })
	bt := b.textEditor.Text()
	i := strings.Index(bt, "Second")
	b.textEditor.Edit(func(e *buffer.Buffer) { e.SetCursor(len([]rune(bt[:i])), false); e.Insert("Really ") })
	waitFor(t, "a's source follows b", func() bool { return strings.Contains(d.Text(), "Really Second passage") })
	if !strings.Contains(d.Text(), "Typed by a.") {
		t.Errorf("a's typing lost: %q", d.Text())
	}
}

// On a phone the Script tab is in the sheet, and the swap goes as wide.
func TestSwapPhone(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.onWindowWidth(380)
	if !w.narrow {
		t.Skip("the window is not narrow here")
	}
	w.swapWithMain("source")
	if !strings.Contains(tabTitles(w), "Script") || w.paneSplit == nil {
		t.Errorf("tabs %q", tabTitles(w))
	}
	m := w.dock.menu("source")
	if m.Items[1].Label != "Swap with Main (put back)" {
		t.Errorf("menu %q", m.Items[1].Label)
	}
	m.Items[1].Action()
	if w.swapped != "" {
		t.Error("not back")
	}
}

// Adaptation > Swap Source and Script: the Source pane brought in when it
// is not there, swapped, and back; View > Swap Pane with Main the same
// for the pane picked last.
func TestSwapSourceAndScriptCommand(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.togglePane("source") // out of the dock
	c := cmd("source.swap")
	if !c.canRun(w.cmdEnv()) {
		t.Fatal("not there with a source")
	}
	c.run(w)
	if w.swapped != "source" || !c.checked(w) || w.fyneWindow.Canvas().Focused() != w.mainDoc.ed {
		t.Fatalf("swapped %q", w.swapped)
	}
	c.run(w)
	if w.swapped != "" || !w.paneInSight("source") {
		t.Error("not back, the pane in its place")
	}
	w.showPane("notes", false)
	cmd("view.paneSwap").run(w)
	if w.swapped != "notes" {
		t.Errorf("Swap Pane with Main swapped %q", w.swapped)
	}
	cmd("view.paneSwap").run(w)
	if w.swapped != "" {
		t.Error("Swap Pane with Main did not put it back")
	}
}

// The Cards pane: one card a scene or section, its synopsis written into
// the script, a drag moving the scene, the cursor's scene lit; with the
// source in the main editor, the passage's scenes lit. Swapped with the
// main area it is the full Cards view.
func TestCardsPane(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.togglePane("cards")
	c := w.cardsPane()
	if !c.shown() || len(c.items) != 3 || c.items[0].Text != "INT. HOUSE - NIGHT" {
		t.Fatalf("cards %v", c.items)
	}
	// the cursor's scene
	w.textEditor.SetCursorOffset(runeAt(t, w.textEditor.Text(), "ANNA"))
	if !c.lit[1] || c.lit[0] || c.lit[2] {
		t.Errorf("lit %v for the second scene", c.lit)
	}
	// a synopsis typed on a card
	c.setSynopsis(0, "Anna hears something")
	if !strings.Contains(w.textEditor.Text(), "INT. HOUSE - NIGHT\n= Anna hears something") || c.items[0].Synopsis != "Anna hears something" {
		t.Errorf("synopsis: %q", w.textEditor.Text())
	}
	// a card dragged below the last moves its scene to the end
	c.moveRow(0, dropIndex(c, 0, 3*c.row))
	if c.items[2].Text != "INT. HOUSE - NIGHT" {
		t.Errorf("after the drag: %v", c.items)
	}
	// the source's passage lights its scenes' cards
	w.swapWithMain("source")
	d := w.mainDoc
	d.ed.SetCursorOffset(runeAt(t, d.Text(), "The next morning"))
	var lit []string
	for i := range c.items {
		if c.lit[i] {
			lit = append(lit, c.items[i].Text)
		}
	}
	if strings.Join(lit, ",") != "EXT. GARDEN - DAY" {
		t.Errorf("lit for the passage: %v", lit)
	}
	w.swapBack()
	// swapped: the full view
	w.toggleCards()
	if w.swapped != "cards" || !w.cardsVisible || w.swappedCenter() != w.cards.box || !strings.Contains(tabTitles(w), "Script") {
		t.Fatalf("the full Cards view: swapped %q", w.swapped)
	}
	w.cards.open(0)
	if w.swapped != "" || w.cardsVisible {
		t.Error("a card opened: the script not back")
	}
}

// A passage's opening words edited: the links that found it by the old
// words are written again for the new ones, the others left as they are.
func TestLinkFixes(t *testing.T) {
	old := passagesOf(sourceStory)
	story := strings.Replace(sourceStory, "She opened the door to the garden", "She pushed open the door of the garden", 1)
	nw := passagesOf(story)
	edits, n := linkFixes(sourceScript, old, nw)
	if n != 1 || len(edits) != 1 {
		t.Fatalf("%d links, %d edits: %v", n, len(edits), edits)
	}
	script := []rune(sourceScript)
	e := edits[0]
	got := string(script[:e.Start]) + e.Text + string(script[e.End:])
	if strings.Contains(got, "She opened the door") || !strings.Contains(got, "pushed") {
		t.Fatalf("the link %q", e.Text)
	}
	m := newSourceModel(nw, got, nil)
	at := passageNamed(t, nw, "She pushed")
	if lines := m.linked(at); len(lines) != 1 {
		t.Errorf("the new link finds %v", lines)
	}
	if _, n := linkFixes(sourceScript, old, old); n != 0 {
		t.Errorf("nothing changed, yet %d links to update", n)
	}
	// a passage split in two: a link to it goes to its last part too
	if to := alignPassages(old, passagesOf(strings.Replace(sourceStory, "and stepped out", "and stopped.\n\nShe stepped out", 1))); to[2] != 3 {
		t.Errorf("the split passage maps to %d", to[2])
	}
}

// In the main editor: the cursor leaving a passage whose opening was
// edited, the notice asks; Update Links is one undo step of the script's;
// Not Now leaves the links. The Source pane's Done asks as well.
func TestLinkOffer(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.swapWithMain("source")
	d := w.mainDoc
	at := runeAt(t, d.Text(), "She opened")
	d.ed.Edit(func(b *buffer.Buffer) { b.Replace(at+4, at+10, "pushed") }) // "She pushed the door…"
	if w.linkOffer.box.Visible() {
		t.Fatal("asked while still in the passage")
	}
	d.ed.SetCursorOffset(runeAt(t, d.Text(), "The next morning"))
	if !w.linkOffer.box.Visible() || !strings.Contains(w.linkOffer.text.Text, "1 link") {
		t.Fatalf("no offer: %q", w.linkOffer.text.Text)
	}
	script := w.textEditor.Text()
	w.updateOfferedLinks()
	if w.linkOffer.box.Visible() || strings.Contains(w.textEditor.Text(), "She opened the door") {
		t.Fatalf("not updated: %q", w.textEditor.Text())
	}
	w.textEditor.Undo()
	if w.textEditor.Text() != script {
		t.Error("the update is not one undo step")
	}
	// Not Now: as they are
	at = runeAt(t, d.Text(), "The next morning")
	d.ed.Edit(func(b *buffer.Buffer) { b.Replace(at+4, at+11, "following") })
	d.ed.SetCursorOffset(0)
	if !w.linkOffer.box.Visible() {
		t.Fatal("no offer for the second passage")
	}
	w.hideLinkOffer()
	if w.textEditor.Text() != script {
		t.Error("Not Now changed the script")
	}
	w.swapBack()

	// the pane's Edit and Done
	w2, _ := sourceWindow(t, sourceScript, sourceStory)
	s := w2.sourcePane()
	s.toggleEdit()
	s.w.srcDoc.SetText(strings.Replace(sourceStory, "The next morning the garden", "Next day the garden", 1))
	s.toggleEdit()
	if !w2.linkOffer.box.Visible() {
		t.Error("Done did not offer to update the link")
	}
}

// TestSwapScreenshots renders the swapped states, on the desktop and at
// a phone's size (SHOT_DIR as in screenshot_test.go): the source in the
// main editor with the Script tab, the link offer, the notes, the Cards
// pane, the full Cards view, the outline in the main area.
func TestSwapScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, size := range []struct {
		name string
		w, h float32
	}{{"desktop", 1100, 680}, {"phone", 380, 800}} {
		w, _ := sourceWindow(t, sourceScript, sourceStory)
		w.fyneWindow.Resize(fyne.NewSize(size.w, size.h))
		w.onWindowWidth(size.w)
		snap := func(name string) {
			t.Helper()
			w.fyneWindow.Content().Refresh()
			snapWindow(t, w, filepath.Join(dir, "swap-"+size.name+"-"+name+".png"))
		}
		w.textEditor.SetCursorOffset(runeAt(t, w.textEditor.Text(), "EXT. GARDEN - DAY"))
		snap("0-before")
		w.swapWithMain("source")
		d := w.mainDoc
		d.ed.SetCursorOffset(runeAt(t, d.Text(), "The next morning"))
		snap("1-source")
		at := runeAt(t, d.Text(), "She opened")
		d.ed.Edit(func(b *buffer.Buffer) { b.Replace(at+4, at+10, "pushed") })
		d.ed.SetCursorOffset(runeAt(t, d.Text(), "The next morning"))
		snap("2-link-offer")
		w.hideLinkOffer()
		w.swapBack()
		w.showPane("notes", false)
		w.swapWithMain("notes")
		w.mainDoc.SetText("Anna's door: does it open inward?\n\n- the frost scene moves to the morning\n")
		snap("3-notes")
		w.swapBack()
		w.hidePane("notes")
		w.hidePane("source")
		w.showPane("cards", false)
		w.cardsPane().setSynopsis(0, "Anna hears someone at the door")
		w.textEditor.SetCursorOffset(runeAt(t, w.textEditor.Text(), "EXT. GARDEN - NIGHT"))
		snap("4-cards-pane")
		w.toggleCards()
		snap("5-cards-full")
		w.toggleCards()
		w.showPane("outline", false)
		w.swapWithMain("outline")
		snap("6-outline-main")
		w.swapBack()
	}
}
