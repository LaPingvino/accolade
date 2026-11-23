package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

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

// midTitles are the middle's tab titles as shown, in order, the one in
// front in [brackets] ("" with no row: the script alone).
func midTitles(w *MainWindow) string {
	var out []string
	for _, t := range w.mid.tabsIn(nil) {
		if t.selected {
			out = append(out, "["+t.title+"]")
		} else {
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

// The source as a middle tab: a Markdown document there, its commands,
// its own undo, find and replace in it, the title naming it; File > Save
// writes it beside the script; the script's tab in front again brings the
// screenplay's commands back; closing the tab writes the source and puts
// the pane back in reach.
func TestMiddleSource(t *testing.T) {
	w, p := sourceWindow(t, sourceScript, sourceStory)
	script := w.textEditor.Text()
	w.openMiddle("source")
	d := w.midDoc()
	if w.mid.shown != "source" || d == nil || d.kind != docSource || w.mainEditor() != d.ed || d.Text() != sourceStory {
		t.Fatalf("in front %q, main %v", w.mid.shown, w.mainEditor() == w.textEditor)
	}
	if w.mid.body != d.view {
		t.Error("the source is not in the middle")
	}
	if got := midTitles(w); got != "Script [Source]" {
		t.Errorf("middle tabs %q", got)
	}
	if got := tabTitles(w); strings.Contains(got, "Source") || strings.Contains(got, "Script") {
		t.Errorf("dock tabs %q: the source is in the middle, not also in the dock, and the script not there", got)
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
	if strings.Contains(d.Text(), "garden") || withoutLinks(w.textEditor.Text()) != withoutLinks(script) { // (its links follow the source: relink.go)
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
	w.showMiddle("") // the script's tab: the source stays open beside it
	if !w.scriptInMain() || w.mainEditor() != w.textEditor || w.srcDoc != d || midTitles(w) != "[Script] Source" {
		t.Fatalf("the script's tab: %q", midTitles(w))
	}
	if !strings.HasPrefix(w.fyneWindow.Title(), "•") {
		t.Errorf("the source behind the script's tab, not written: title %q", w.fyneWindow.Title())
	}
	if env := w.cmdEnv(); !cmd(elementCommandID("Action")).applies(env) || cmd("md.heading2").applies(env) {
		t.Error("the screenplay's commands are not back")
	}
	w.showMiddle("source")
	if w.mainEditor() != d.ed || !w.cmdEnv().midDoc {
		t.Fatal("back to the source's tab")
	}
	w.backToSide("source")
	if !w.scriptInMain() || w.mid.has("source") || w.srcDoc != nil || midTitles(w) != "" {
		t.Fatalf("not back to the side: %q", midTitles(w))
	}
	data, _ = os.ReadFile(filepath.Join(filepath.Dir(p), "story.md"))
	if !strings.HasSuffix(string(data), "The end.\n") {
		t.Errorf("not written on the way back: %q", data)
	}
	if got := tabTitles(w); !strings.Contains(got, "Source") {
		t.Errorf("dock tabs %q after", got)
	}
	if s := w.sourcePane(); !strings.Contains(s.src, "The end.") {
		t.Error("the pane does not show the source as written")
	}
}

// A source kept in the script: each change in the main editor goes into
// the script's block at once, all of them one undo step there; a change
// made to the block in the script (Undo, a co-writer) comes into the
// source's editor outside its own history.
func TestMiddleSourceInsideScript(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.toggleSourceInside()
	if w.sourceRef().Origin != sourceInside {
		t.Fatal("not inside")
	}
	w.openMiddle("source")
	d := w.midDoc()
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
	w.closeMiddle("source")
	before := w.textEditor.Text()
	w.fyneWindow.Canvas().Focus(w.textEditor)
	if !strings.Contains(before, "Then She opened") || strings.Contains(before, "[[from: \"She opened the door") {
		t.Fatalf("the link to the passage edited not written again: %q", before)
	}
	w.undo() // the links written again for the source's new words: a step of their own
	if !strings.Contains(w.textEditor.Text(), "[[from: \"She opened the door") || !strings.Contains(w.textEditor.Text(), "One.") {
		t.Fatalf("the first Undo: %q", w.textEditor.Text())
	}
	w.undo()
	if w.textEditor.Text() == before || strings.Contains(w.textEditor.Text(), "One.") {
		t.Errorf("the script's Undo did not take the source's writing back as one: %q", w.textEditor.Text())
	}
}

// The notes in the middle: written into the script's notes block, the
// Notes pane following when back; in a Markdown document as its HTML
// comment.
func TestMiddleNotes(t *testing.T) {
	w := newTestWindow(t, "INT. BARN - DAY\n\nRain.\n\n/* Notes\nfirst\n*/\n")
	w.showPane("notes", false)
	w.openMiddle("notes")
	d := w.midDoc()
	if d == nil || d.kind != docNotes || d.Text() != "first" || midTitles(w) != "Script [Notes]" || w.paneShown("notes") {
		t.Fatalf("notes %v, tabs %q", d, midTitles(w))
	}
	d.ed.Edit(func(b *buffer.Buffer) { b.SetCursor(b.Len(), false); b.Insert(" and more") })
	if notesOf(w.textEditor.Text()) != "first and more" {
		t.Errorf("the script's notes %q", notesOf(w.textEditor.Text()))
	}
	w.backToSide("notes")
	if w.notesPane().entry.Text != "first and more" {
		t.Errorf("the Notes pane %q", w.notesPane().entry.Text)
	}

	m := mdWindow(t, "# Story\n\nText.\n\n<!-- Notes\nmine\n-->\n", 0, 0)
	m.showPane("notes", false)
	m.openMiddle("notes")
	if m.midDoc() == nil || m.midDoc().Text() != "mine" {
		t.Fatal("the Markdown document's notes")
	}
	m.midDoc().ed.Edit(func(b *buffer.Buffer) { b.SetCursor(b.Len(), false); b.Insert("!") })
	if !strings.HasSuffix(m.textEditor.Text(), "<!-- Notes\nmine!\n-->\n") {
		t.Errorf("Markdown's notes block %q", m.textEditor.Text())
	}
	if m.middleBlocked("source") == "" || m.middleBlocked("cards") == "" {
		t.Error("a Markdown document has no source or cards to put in the middle")
	}
}

// A pane is in the dock or in the middle, never both: what brings it into
// the dock (Help's pages, the source's commands, Side Panes) brings its
// middle tab to the front instead.
func TestMiddleOnePlace(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.openMiddle("help")
	w.showMiddle("")
	w.showPane("help", true)
	if w.paneShown("help") || w.mid.shown != "help" {
		t.Errorf("Help: in the dock %v, in front %q", w.paneShown("help"), w.mid.shown)
	}
	w.dock.toggleAll() // the source out, stashed
	cmd("source.middle").run(w)
	w.dock.toggleAll() // back: not the source, a middle tab now
	if w.paneShown("source") || !w.mid.has("source") || strings.Contains(tabTitles(w), "Source") {
		t.Errorf("Side Panes brought the source into the dock too: %q", tabTitles(w))
	}
}

// Open in the Middle is refused, with a reason, where it cannot be; the
// tab's menu item is greyed.
func TestMiddleBlocked(t *testing.T) {
	w := newTestWindow(t, "INT. BARN - DAY\n")
	w.showPane("source", false)
	w.openMiddle("source")
	if w.mid.has("source") || !strings.Contains(w.statusLabel.Text, "No source") || !w.paneShown("source") {
		t.Errorf("in the middle %v, status %q", w.mid.tabs, w.statusLabel.Text)
	}
	if m := w.dock.menu("source"); !m.Items[1].Disabled {
		t.Error("Open in the Middle not greyed")
	}
}

// Another script in the window: the source written first and its tab
// closed, the script in front, a view's tab (the outline) kept; the same
// for the document type, which closes the cards' tab too.
func TestMiddleLeftForAnotherDocument(t *testing.T) {
	w, p := sourceWindow(t, sourceScript, sourceStory)
	w.openMiddle("outline")
	w.openMiddle("source")
	w.midDoc().ed.Edit(func(b *buffer.Buffer) { b.SetCursor(0, false); b.Insert("New first line.\n\n") })
	other := filepath.Join(t.TempDir(), "other.fountain")
	os.WriteFile(other, []byte("INT. ELSEWHERE - DAY\n"), 0o644)
	if err := w.LoadFile(other); err != nil {
		t.Fatal(err)
	}
	if !w.scriptInMain() || w.srcDoc != nil || w.mainEditor() != w.textEditor || midTitles(w) != "[Script] Outline" {
		t.Errorf("after Open: %q", midTitles(w))
	}
	if data, _ := os.ReadFile(filepath.Join(filepath.Dir(p), "story.md")); !strings.HasPrefix(string(data), "New first line.") {
		t.Errorf("the source not written: %q", data)
	}

	n := newTestWindow(t, "INT. BARN - DAY\n\n/* Notes\nkept\n*/\n")
	n.showPane("notes", false)
	n.openMiddle("production")
	n.setOwnRole(roleWriter)
	if n.mid.has("production") {
		t.Error("a Writer kept Production's middle tab")
	}
	n.openMiddle("cards")
	n.openMiddle("notes")
	n.setDocumentType(true)
	if !n.scriptInMain() || len(n.mid.tabs) != 0 || !strings.HasSuffix(n.textEditor.Text(), "<!-- Notes\nkept\n-->\n") {
		t.Errorf("middle %v, %q", n.mid.tabs, n.textEditor.Text())
	}
}

// Auto-save writes the source in the middle with the script.
func TestMiddleAutoSave(t *testing.T) {
	w, p := sourceWindow(t, sourceScript, sourceStory)
	w.settings.SetBoolean("auto-save", true)
	t.Cleanup(func() { w.settings.SetBoolean("auto-save", false) })
	w.openMiddle("source")
	w.showMiddle("") // behind the script's tab: written all the same
	w.srcDoc.ed.Edit(func(b *buffer.Buffer) { b.SetCursor(b.Len(), false); b.Insert("Saved by itself.\n") })
	w.autoSaveNow()
	if data, _ := os.ReadFile(filepath.Join(filepath.Dir(p), "story.md")); !strings.HasSuffix(string(data), "Saved by itself.\n") {
		t.Errorf("not auto-saved: %q", data)
	}
}

// A view in the middle (the outline): its box there, nothing to write in
// (the editing commands say so), the script's cursor commands say why
// not; a scene tapped puts the script's tab in front at it; the View
// command shows the tab, and in front closes it.
func TestMiddleView(t *testing.T) {
	w := newTestWindow(t, "INT. BARN - DAY\n\nRain.\n\nEXT. FIELD - NIGHT\n\nWind.\n")
	w.showPane("outline", false)
	w.openMiddle("outline")
	if w.mid.body != w.outlinePane().box || w.mainEditor() != nil || !w.paneInSight("outline") || w.paneShown("outline") {
		t.Fatal("the outline is not in the middle")
	}
	if cmd(elementCommandID("Action")).applies(w.cmdEnv()) || cmd("md.heading1").applies(w.cmdEnv()) {
		t.Error("no editor: no element commands")
	}
	before := w.textEditor.Text()
	w.tapped(cmd("format.bold"), nil)()
	if w.textEditor.Text() != before || !strings.Contains(w.statusLabel.Text, "Nothing to write in") {
		t.Errorf("bold with no editor: status %q", w.statusLabel.Text)
	}
	// the outline in the middle follows the script's changes (Undo, a peer)
	w.textEditor.Edit(func(b *buffer.Buffer) { b.SetCursor(b.Len(), false); b.Insert("\nINT. HALL - DAY\n\nDust.\n") })
	if items := w.outlinePane().all; len(items) != 3 {
		t.Errorf("the outline in the middle did not follow the script: %d items", len(items))
	}
	w.textEditor.Undo()
	w.omitScene()
	if w.textEditor.Text() != before || !strings.Contains(w.statusLabel.Text, "its middle tab first") {
		t.Errorf("omit with the script away: %q", w.statusLabel.Text)
	}
	w.outlinePane().goTo(4)
	if line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset()); !w.scriptInMain() || line != 4 ||
		w.fyneWindow.Canvas().Focused() != w.textEditor {
		t.Errorf("a scene tapped: script in front %v, line %d", w.scriptInMain(), line)
	}
	w.togglePane("outline") // its View command: its tab to the front
	if w.mid.shown != "outline" {
		t.Fatal("View > Outline did not show its tab")
	}
	w.togglePane("outline") // in front: closed
	if w.mid.has("outline") || !w.scriptInMain() || w.paneShown("outline") {
		t.Error("View > Outline in front did not close its tab")
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

// typeShortcut is s typed with the keyboard where it is: the focused
// widget's first (the editors hand the menu's keys on), else the menu's,
// as the driver does.
func typeShortcut(w *MainWindow, s fyne.Shortcut) {
	if w.fyneWindow.MainMenu() != nil && runMenuShortcut(w.fyneWindow.MainMenu(), s) {
		return // the driver tries the menu bar's first
	}
	if f, ok := w.fyneWindow.Canvas().Focused().(fyne.Shortcutable); ok {
		if _, entry := f.(*widget.Entry); !entry {
			f.TypedShortcut(s)
			return
		}
	}
	if w.menuShortcut(s) {
		return
	}
}

// runMenuShortcut runs the menu bar's item with shortcut s, as the driver
// does; false if none has it.
func runMenuShortcut(mm *fyne.MainMenu, s fyne.Shortcut) bool {
	var find func([]*fyne.MenuItem) bool
	find = func(items []*fyne.MenuItem) bool {
		for _, it := range items {
			if it.Shortcut != nil && it.Shortcut.ShortcutName() == s.ShortcutName() && it.Action != nil {
				it.Action()
				return true
			}
			if it.ChildMenu != nil && find(it.ChildMenu.Items) {
				return true
			}
		}
		return false
	}
	for _, m := range mm.Items {
		if find(m.Items) {
			return true
		}
	}
	return false
}

// Editing together: the notes and a source kept in the script are the
// script's text, so they go both ways while in the middle; a source kept
// elsewhere is this writer's alone, as the status says.
func TestMiddleTogether(t *testing.T) {
	_, _, block := withSourceBlock("INT. BARN - DAY\n\nRain.\n", "First passage here.\n\nSecond passage here.")
	text := "INT. BARN - DAY\n\nRain.\n" + block
	a, b, _ := sharedPair(t, text)
	connect(t, a, b)
	textIs(t, b, "b", text)
	a.showPane("source", false)
	a.openMiddle("source")
	d := a.midDoc()
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

// On a phone the middle's tabs are one row: those that do not fit in the
// More menu, the one in front always on the row; the tab's menu has Back
// to the Side and Close.
func TestMiddlePhone(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.onWindowWidth(380)
	if !w.narrow {
		t.Fatal("the window is not narrow here")
	}
	for _, id := range []string{"cards", "outline", "notes", "help", "characters", "source"} {
		w.openMiddle(id)
	}
	w.showPane("changes", false) // the sheet under the middle: half the height

	if w.mid.shown != "source" || w.mid.row == nil || w.paneSplit == nil {
		t.Fatalf("tabs %q", midTitles(w))
	}
	w.fyneWindow.Content().Refresh()
	r := w.mid.row
	r.Layout(r.Objects, r.Size())
	src, hidden := -1, 0
	for i, it := range r.items {
		if it.obj == fyne.CanvasObject(w.mid.tabW["source"]) {
			src = i
		}
		if r.hidden(i) {
			hidden++
		}
	}
	if src < 0 || r.hidden(src) || hidden == 0 || !r.more.Visible() {
		t.Errorf("the row: the source's tab hidden %v, %d in the menu", src < 0 || r.hidden(src), hidden)
	}
	if rs := r.Size(); rs.Width > 380 {
		t.Errorf("the row is %v wide on a phone", rs.Width)
	}
	// the main editor keeps its cursor in view at a phone's height
	d := w.midDoc()
	d.ed.SetCursorOffset(len([]rune(d.Text())))
	if sc, ok := d.view.Content.(*container.Scroll); !ok || sc.Offset.Y <= 0 {
		t.Errorf("the source's editor did not scroll to its cursor (%v)", ok)
	}
	m := w.mid.menu("source")
	if m.Items[0].Label != "Back to the Side" || m.Items[1].Label != "Close Source" ||
		len(w.mid.menu("").Items) != 1 || !strings.HasPrefix(w.mid.menu("").Items[0].Label, "Rename ") { // the script's: its name's editor (toprow.go)
		t.Errorf("menu %q, %q", m.Items[0].Label, m.Items[1].Label)
	}
	m.Items[0].Action()
	if w.mid.has("source") || !w.paneInSight("source") || w.mid.shown != "characters" {
		t.Errorf("not back to the side, or not the tab before in front: %q", midTitles(w))
	}
}

// Adaptation > Source in the Middle: the source's tab (opened if it is
// not), and again the script's; View > Open Pane in the Middle for the
// pane picked last, Middle Tab Back to the Side, and Ctrl+PgDn/PgUp
// round the tabs.
func TestMiddleCommands(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.togglePane("source") // out of the dock
	c := cmd("source.middle")
	if !c.canRun(w.cmdEnv()) {
		t.Fatal("not there with a source")
	}
	c.run(w)
	if w.mid.shown != "source" || !c.checked(w) || w.fyneWindow.Canvas().Focused() != w.midDoc().ed {
		t.Fatalf("in front %q", w.mid.shown)
	}
	c.run(w)
	if !w.scriptInMain() || !w.mid.has("source") || c.checked(w) {
		t.Error("not the script's tab, the source's kept")
	}
	w.showPane("notes", false)
	cmd("view.paneMiddle").run(w)
	if w.mid.shown != "notes" || midTitles(w) != "Script Source [Notes]" {
		t.Errorf("Open Pane in the Middle: %q", midTitles(w))
	}
	next, prev := cmd("view.midNext"), cmd("view.midPrev")
	if !next.canRun(w.cmdEnv()) || len(next.keys) == 0 {
		t.Fatal("Next Middle Tab not there")
	}
	next.run(w)
	if !w.scriptInMain() {
		t.Errorf("Next from the last: %q", midTitles(w))
	}
	prev.run(w)
	prev.run(w)
	if w.mid.shown != "source" {
		t.Errorf("Previous twice from the script: %q", midTitles(w))
	}
	cmd("view.paneSide").run(w)
	if w.mid.has("source") || !w.paneInSight("source") {
		t.Errorf("Back to the Side: %q", midTitles(w))
	}
	// the keys, with the keyboard in the editor of the tab in front
	w.showMiddle("")
	w.fyneWindow.Canvas().Focus(w.textEditor)
	typeShortcut(w, next.keys[0])
	if w.mid.shown != "notes" {
		t.Errorf("Ctrl+PgDn from the script: %q", midTitles(w))
	}
	typeShortcut(w, prev.keys[0])
	if !w.scriptInMain() {
		t.Errorf("Ctrl+PgUp from the notes: %q", midTitles(w))
	}
	// the strip's tabs: the arrows go along, Down into the tab in front
	t0 := w.mid.tabW[""]
	t0.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	if w.mid.shown != "notes" || w.fyneWindow.Canvas().Focused() != w.mid.tabW["notes"] {
		t.Errorf("Right on the Script tab: %q", midTitles(w))
	}
	w.mid.tabW["notes"].TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	if w.fyneWindow.Canvas().Focused() != w.midDoc().ed {
		t.Error("Down on the Notes tab: not in its editor")
	}
}

// The Cards pane: one card a scene or section, its synopsis written into
// the script, a drag moving the scene, the cursor's scene lit; with the
// source in the middle, the passage's scenes lit. View > Cards opens the
// full view as a middle tab, never in the dock first; a card opened from
// it puts the script's tab in front, the cards' kept.
func TestCardsPane(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.togglePane("cards") // View > Cards: the middle
	if !w.cardsShown() || w.paneShown("cards") || midTitles(w) != "Script [Cards]" {
		t.Fatalf("View > Cards: tabs %q, in the dock %v", midTitles(w), w.paneShown("cards"))
	}
	w.backToSide("cards")
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
	w.openMiddle("source")
	d := w.midDoc()
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
	w.closeMiddle("source")
	// the pane's full-view button: the middle tab
	w.openMiddle("cards")
	if !w.cardsShown() || w.mid.body != w.cards.box || w.paneShown("cards") || midTitles(w) != "Script [Cards]" {
		t.Fatalf("the full Cards view: tabs %q", midTitles(w))
	}
	w.cards.open(0)
	if !w.scriptInMain() || !w.mid.has("cards") {
		t.Error("a card opened: the script not in front, or the cards' tab gone")
	}
	w.togglePane("cards") // the cards' tab to the front
	w.togglePane("cards") // in front: closed
	if w.mid.has("cards") || w.paneShown("cards") {
		t.Error("View > Cards in front did not close the tab")
	}
}

// Leaving the source's tab (it stays open) after editing a passage writes
// the links quoting it again, one undo step.
func TestRelinkOnLeavingTab(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.openMiddle("source")
	d := w.midDoc()
	at := runeAt(t, d.Text(), "She opened")
	script := w.textEditor.Text()
	d.ed.Edit(func(b *buffer.Buffer) { b.Replace(at+4, at+10, "pushed") })
	w.showMiddle("")
	if strings.Contains(w.textEditor.Text(), "She opened the door") || !w.mid.has("source") {
		t.Errorf("leaving the source's tab did not write the link again:\n%s", w.textEditor.Text())
	}
	w.textEditor.Undo()
	if w.textEditor.Text() != script {
		t.Error("not one undo step")
	}
}

// TestMiddleScreenshots renders the middle's tabs, on the desktop and at
// a phone's size (SHOT_DIR as in screenshot_test.go): the script alone,
// Cards from View > Cards (a middle tab, nothing in the sidebar), then
// Script + Cards + Source with each in front, its links written again,
// the notes, the outline in the middle, the Cards pane beside the script.
func TestMiddleScreenshots(t *testing.T) {
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
		w.hidePane("source")
		w.fyneWindow.Resize(fyne.NewSize(size.w, size.h))
		w.onWindowWidth(size.w)
		snap := func(name string) {
			t.Helper()
			w.fyneWindow.Content().Refresh()
			snapWindow(t, w, filepath.Join(dir, "middle-"+size.name+"-"+name+".png"))
		}
		w.textEditor.SetCursorOffset(runeAt(t, w.textEditor.Text(), "EXT. GARDEN - DAY"))
		snap("0-script")
		w.cardsPane().update(w.textEditor.Text())
		w.cardsPane().setSynopsis(0, "Anna hears someone at the door")
		cmd("view.cards").run(w)
		snap("1-cards")
		w.openMiddle("source")
		d := w.midDoc()
		d.ed.SetCursorOffset(runeAt(t, d.Text(), "The next morning"))
		snap("2-source")
		w.showMiddle("")
		snap("3-script-of-three")
		w.showMiddle("cards")
		snap("4-cards-of-three")
		w.showMiddle("source")
		at := runeAt(t, d.Text(), "She opened")
		d.ed.Edit(func(b *buffer.Buffer) { b.Replace(at+4, at+10, "pushed") })
		d.ed.SetCursorOffset(runeAt(t, d.Text(), "The next morning"))
		snap("5-relinked")
		w.openMiddle("notes")
		w.midDoc().SetText("Anna's door: does it open inward?\n\n- the frost scene moves to the morning\n")
		snap("6-notes")
		w.openMiddle("outline")
		snap("7-outline")
		w.closeMiddle("outline")
		w.closeMiddle("notes")
		w.closeMiddle("source")
		w.backToSide("cards")
		w.textEditor.SetCursorOffset(runeAt(t, w.textEditor.Text(), "EXT. GARDEN - NIGHT"))
		snap("8-cards-pane")
		w.toggleStripboard() // a middle tab, the side panes beside it
		snap("9-stripboard")
		w.showMiddle("")
		shown := make(chan *pageView, 4)
		onPagesShown(t, func(v *pageView) { shown <- v })
		w.preview() // a phone's: the Pages tab next to Script; a desktop's: beside the script
		awaitPages(t, shown)
		snap("10-pages")
		w.showMiddle("")
		snap("11-script-and-pages")
	}
}

// Keep Source in the Script, both ways, with the source open: in the
// middle, and in the pane's Edit; the document stays open, kept where the source
// is now, what was being written kept.
func TestKeepSourceInsideWhileOpen(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.openMiddle("source")
	d := w.midDoc()
	d.ed.Edit(func(b *buffer.Buffer) { b.SetCursor(b.Len(), false); b.Insert("Written first.\n") })
	w.toggleSourceInside()
	if w.mid.shown != "source" || w.midDoc() != d || w.srcDoc != d || !d.inScript() {
		t.Fatalf("in: in front %q, doc kept %v", w.mid.shown, w.srcDoc == d)
	}
	if _, _, md, _ := sourceBlock(w.textEditor.Text()); !strings.Contains(md, "Written first.") {
		t.Error("the block has not what was written")
	}
	w.toggleSourceInside()
	if w.mid.shown != "source" || w.srcDoc != d || d.inScript() || d.ref.Origin != sourceKept {
		t.Fatalf("out: in front %q, ref %+v", w.mid.shown, d.ref)
	}
	if strings.Contains(w.textEditor.Text(), "/* Source") {
		t.Error("the block stayed")
	}
	data, _ := os.ReadFile(d.ref.Path)
	if !strings.Contains(string(data), "Written first.") {
		t.Errorf("the kept copy %q", data)
	}
	w.backToSide("source")

	s := w.sourcePane()
	s.toggleEdit()
	pd := w.srcDoc
	w.toggleSourceInside()
	if !s.editing || w.srcDoc != pd || !pd.inScript() {
		t.Error("the pane's Edit ended, or another document")
	}
	s.toggleEdit()
}

// Rename and a join while the source is in the middle: written first,
// its tab closed, the script in front.
func TestMiddleRenameAndJoin(t *testing.T) {
	w, p := relocWindow(t)
	w.showPane("source", false)
	w.openMiddle("source")
	if w.mid.shown != "source" {
		t.Fatal("not in the middle")
	}
	w.midDoc().ed.Edit(func(b *buffer.Buffer) { b.SetCursor(0, false); b.Insert("Before the rename.\n\n") })
	dir := filepath.Dir(p)
	if err := w.relocate(relocation{rename: true, from: p, to: filepath.Join(dir, "film.fountain"), keepBesideName: true}, nil); err != nil {
		t.Fatal(err)
	}
	if !w.scriptInMain() || w.mid.has("source") || w.srcDoc != nil {
		t.Error("the source's tab still open after the rename")
	}
	md, _ := w.sourceText(w.sourceRef())
	if !strings.HasPrefix(md, "Before the rename.") {
		t.Errorf("the source after the rename %q", md)
	}

	a, b, key := sharedPair(t, "INT. BARN - DAY\n\n/* Notes\nmine\n*/\n")
	_ = a
	b.stopSharing()
	b.showPane("notes", false)
	b.openMiddle("notes")
	b.startSharing(key, true)
	if !b.scriptInMain() || b.mid.has("notes") || b.notesDoc != nil {
		t.Error("a join left the notes in the middle")
	}
}

// withoutLinks is a script without its source links (what the source's
// own edits leave as it is).
func withoutLinks(text string) string {
	for {
		i := strings.Index(text, "[[from:")
		if i < 0 {
			return text
		}
		j := strings.Index(text[i:], "]]")
		if j < 0 {
			return text
		}
		text = text[:i] + text[i+j+2:]
	}
}

// ex's :s and vim's keys act on the source in the middle, not the
// script; Live Preview shows the source, and the script's tab in front
// takes it away (a screenplay's preview is its pages).
func TestMiddleExVimPreview(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	script := w.textEditor.Text()
	w.openMiddle("source")
	d := w.midDoc()
	w.runEx("%s/garden/orchard/g")
	if strings.Contains(d.Text(), "garden") || w.textEditor.Text() != script {
		t.Errorf(":s went elsewhere: %q", d.Text())
	}
	w.settings.SetString("editor-keys", "vim")
	t.Cleanup(func() { w.settings.SetString("editor-keys", "standard") })
	w.applyEditorSettings()
	if !d.vim.active() {
		t.Fatal("the source's vim is not on")
	}
	d.ed.SetCursorOffset(0)
	d.vim.mode = vimNormal
	d.ed.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	for _, r := range "dd" {
		d.ed.TypedRune(r)
	}
	if strings.HasPrefix(d.Text(), "---\n") || withoutLinks(w.textEditor.Text()) != withoutLinks(script) { // (its links follow the source: relink.go)
		t.Errorf("dd: %q", strings.SplitN(d.Text(), "\n", 2)[0])
	}
	w.settings.SetString("editor-keys", "standard")
	w.applyEditorSettings()
	w.preview()
	if !w.previewVisible || !strings.Contains(richText(w.previewPane.text), "orchard") {
		t.Error("Live Preview does not show the source")
	}
	// its page setup line is the document's, not the script's PDF
	if l := w.previewPane.line; l == nil || l.summary.Text != w.markdownPageSummary() {
		t.Errorf("Live Preview's page setup line: %v", l)
	}
	w.closeMiddle("source")
	if w.previewVisible {
		t.Error("a screenplay's preview is a PDF: the Live Preview hidden again")
	}
}

func richText(r interface{ String() string }) string { return r.String() }

// The notes in a shared session go both ways while in the middle.
func TestMiddleNotesTogether(t *testing.T) {
	a, b, _ := sharedPair(t, "INT. BARN - DAY\n\n/* Notes\nshared\n*/\n")
	connect(t, a, b)
	textIs(t, b, "b", "INT. BARN - DAY\n\n/* Notes\nshared\n*/\n")
	a.showPane("notes", false)
	a.openMiddle("notes")
	d := a.midDoc()
	d.ed.Edit(func(e *buffer.Buffer) { e.SetCursor(e.Len(), false); e.Insert(" by a") })
	waitFor(t, "b has a's notes", func() bool { return notesOf(b.textEditor.Text()) == "shared by a" })
	b.writeNotes("b first; shared by a")
	waitFor(t, "a's notes follow b", func() bool { return d.Text() == "b first; shared by a" })
}

// The stripboard is a middle tab (docs/MIDDLE.md): opened by its command,
// one tab among the others, the side panes beside it, the ribbon's
// Stripboard tab while it is in front; the Script tab and a scene opened
// keep its tab; the command in front closes
// it; it has no place in the side panes; the Writer role closes it.
func TestMiddleStripboard(t *testing.T) {
	w := newTestWindow(t, schedScript)
	w.fyneWindow.Resize(fyne.NewSize(1100, 700))
	r := ribbonOnFor(t, w)
	w.togglePane("outline")
	w.openMiddle("cards")
	w.showMiddle("")
	r.selected = "Write"
	w.ribbonRefresh()

	w.toggleStripboard()
	if !w.stripsShown() || midTitles(w) != "Script Cards [Stripboard]" {
		t.Fatalf("tabs %q", midTitles(w))
	}
	if w.mid.body != fyne.CanvasObject(w.stripboard.box) || !w.paneInSight("outline") || w.sideSplit == nil {
		t.Error("the board is not in the middle with the side panes beside it")
	}
	if r.selected != "Stripboard" {
		t.Errorf("the ribbon: %q", r.selected)
	}
	if e := w.cmdEnv(); !e.away || e.midSide {
		t.Error("the board can go back to the side")
	}
	if m := w.mid.menu("strips"); len(m.Items) != 1 || m.Items[0].Label != "Close Stripboard" {
		t.Errorf("its tab's menu %v", m.Items)
	}
	w.backToSide("strips")
	if !w.mid.has("strips") {
		t.Error("Back to the Side closed it")
	}
	w.showMiddle("")
	if !w.scriptInMain() || !w.mid.has("strips") || r.selected != "Write" {
		t.Errorf("the Script tab: tabs %q, the ribbon's %q", midTitles(w), r.selected)
	}
	w.mid.step(-1) // round to the last: the board
	if !w.stripsShown() {
		t.Fatalf("Previous Middle Tab: %q", midTitles(w))
	}
	w.stripboard.openScene(1)
	if !w.scriptInMain() || !w.mid.has("strips") {
		t.Errorf("a scene opened: %q", midTitles(w))
	}
	w.showStripboard(-1, 1) // the Schedule stage's row: the board to the front, its scene selected
	if !w.stripsShown() || w.stripboard.selected != 1 {
		t.Errorf("showStripboard: %q, selected %d", midTitles(w), w.stripboard.selected)
	}
	w.toggleStripboard()
	if w.mid.has("strips") || w.mid.shown != "cards" {
		t.Errorf("the command in front did not close it: %q", midTitles(w))
	}
	w.toggleStripboard()
	w.setOwnRole(roleWriter)
	if w.mid.has("strips") {
		t.Error("the Writer role kept the board")
	}
	w.toggleStripboard()
	if w.mid.has("strips") {
		t.Error("the board opened for a writer")
	}
}

// On a phone the page view is a middle tab, Pages, next to the script's:
// Preview opens it in front, a tap on Script and back keeps the pages (the
// ribbon's Preview tab with them in front only), its × closes them, and a
// desktop's width puts them beside the script again.
func TestMiddlePhonePages(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	holdUI(t)
	atWidth(w, 380)
	w.openMiddle("cards")
	w.showMiddle("")
	shown := make(chan *pageView, 4)
	onPagesShown(t, func(v *pageView) { shown <- v })
	w.preview()
	awaitPages(t, shown)
	if midTitles(w) != "Script [Pages] Cards" || w.mid.body != fyne.CanvasObject(w.pages) {
		t.Fatalf("tabs %q", midTitles(w))
	}
	if !w.cmdEnv().preview {
		t.Error("the Pages tab in front, no Preview tab")
	}
	gen := w.previewGen
	w.showMiddle("")
	if !w.pagesVisible || midTitles(w) != "[Script] Pages Cards" || w.previewGen != gen || w.mid.body != w.editorView {
		t.Errorf("a tap on Script: tabs %q, pages %v", midTitles(w), w.pagesVisible)
	}
	if w.cmdEnv().preview {
		t.Error("the script in front, the Preview tab still there")
	}
	w.showMiddle("cards")
	w.showMiddle("pages")
	if !w.pagesVisible || w.mid.shown != "pages" {
		t.Errorf("through the cards: tabs %q, pages %v", midTitles(w), w.pagesVisible)
	}
	if m := w.mid.menu("pages"); len(m.Items) != 1 || m.Items[0].Label != "Close Pages" {
		t.Errorf("its tab's menu %v", m.Items)
	}
	test.Tap(w.mid.shut)
	if w.pagesVisible || w.mid.has("pages") || !w.scriptInMain() {
		t.Errorf("the ×: tabs %q, pages %v", midTitles(w), w.pagesVisible)
	}
	w.preview()
	awaitPages(t, shown)
	atWidth(w, 1100)
	if !w.pagesVisible || w.mid.has("pages") || !w.scriptInMain() {
		t.Errorf("a desktop's width: tabs %q, pages %v", midTitles(w), w.pagesVisible)
	}
	atWidth(w, 380)
	if w.mid.shown != "pages" {
		t.Errorf("a phone's width again: tabs %q", midTitles(w))
	}
	w.hidePages() // Close Preview, Escape on the pages
	if w.mid.has("pages") || !w.scriptInMain() {
		t.Errorf("Close Preview: tabs %q", midTitles(w))
	}
}
