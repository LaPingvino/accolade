package main

import (
	"math/rand"
	"strings"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Typing in a document kept in the script (the source inside it) writes the changed span alone into the script (spanInto): after
// every kind of edit the script holds what writing the whole stretch
// would have made, the home as last seen is the home, and the script's
// cursor stays where it was.
func TestMiddleSpanWrite(t *testing.T) {
	for _, kind := range []string{"source"} {
		t.Run(kind, func(t *testing.T) {
			var w *MainWindow
			if kind == "source" {
				w, _ = sourceWindow(t, sourceScript, sourceStory)
				w.toggleSourceInside()
			} else {
				w = newTestWindow(t, "INT. BARN - DAY\n\nRain.\n\n/* Notes\nfirst notes\n\nand more\n*/\n")
				w.showPane("notes", false)
			}
			w.textEditor.SetCursorOffset(5)
			w.openMiddle(kind)
			d := w.midDoc()
			if d == nil || !d.inScript() {
				t.Fatal("not kept in the script")
			}
			whole := func() string { // the script as writing the whole stretch makes it
				s := []rune(w.textEditor.Text())
				if kind == "source" {
					start, end, block := withSourceBlock(string(s), d.Text())
					return string(s[:start]) + block + string(s[end:])
				}
				from, block := w.notesMarks().edit(string(s), d.Text())
				return string(s[:from]) + block
			}
			check := func(what string) {
				t.Helper()
				if got, want := w.textEditor.Text(), whole(); got != want {
					t.Fatalf("after %s: the script\n%q\nwant\n%q", what, got, want)
				}
				if home, err := d.home(); err != nil || home != d.homeSeen {
					t.Fatalf("after %s: the home as seen %q, is %q", what, d.homeSeen, home)
				}
				if c := w.textEditor.CursorOffset(); c != 5 {
					t.Fatalf("after %s: the script's cursor moved to %d", what, c)
				}
			}
			// a letter typed in the middle goes the short way
			d.ed.SetCursorOffset(3)
			d.ed.TypedRune('x')
			check("a letter")
			if !d.span.ok || d.span.rev != w.textEditor.Buffer().Rev() {
				t.Error("a letter typed was not written as its span")
			}
			bits := []string{"x", "é", "日本", " ", "\n", "\n\n", "*/", "* /", "/*", "-->", "word ", "two\nlines"}
			r := rand.New(rand.NewSource(7))
			for i := 0; i < 400; i++ {
				b := d.ed.Buffer()
				n := b.Len()
				what := ""
				switch op := r.Intn(10); {
				case op < 4: // a bit put in: at the start, the end, anywhere
					at := []int{0, n, r.Intn(n + 1)}[r.Intn(3)]
					s := bits[r.Intn(len(bits))]
					what = "inserting " + s
					d.ed.Edit(func(b *buffer.Buffer) { b.Replace(at, at, s) })
				case op < 6: // a stretch taken out (the text kept, for the source: its block stays)
					if n < 2 {
						continue
					}
					from := r.Intn(n)
					to := min(n, from+1+r.Intn(8))
					if kind == "source" && strings.TrimSpace(b.Slice(0, from)+b.Slice(to, n)) == "" {
						continue
					}
					what = "deleting"
					d.ed.Edit(func(b *buffer.Buffer) { b.Replace(from, to, "") })
				case op < 8: // typed at the cursor
					d.ed.SetCursorOffset(r.Intn(n + 1))
					what = "typing"
					d.ed.TypedRune([]rune("aé\n")[r.Intn(3)])
				case op == 8:
					what = "undo"
					d.ed.Undo()
				default:
					what = "redo"
					d.ed.Redo()
				}
				if kind == "source" && w.midDoc() != d { // the block went: its tab closed
					t.Fatalf("after %s: the source was put back", what)
				}
				check(what)
			}
			if kind == "notes" { // all of it gone, and typed again
				d.ed.Edit(func(b *buffer.Buffer) { b.Replace(0, b.Len(), "") })
				check("emptying")
				d.ed.TypedRune('n')
				check("typing into nothing")
			}
		})
	}
}

// Editing together while typing in the source in the middle: each key goes
// into the shared script at once (no delayed write to merge), and the
// co-writer's edits arriving between the keys (before the block, in it
// before the cursor, after it) come into the source without losing a key.
// The debounce is real: the derived views wait for a pause meanwhile.
func TestMiddleTypingTogether(t *testing.T) {
	_, _, block := withSourceBlock("INT. BARN - DAY\n\nRain.\n", "First passage here.\n\nSecond passage here.")
	text := "INT. BARN - DAY\n\nRain.\n" + block
	a, b, _ := sharedPair(t, text)
	connect(t, a, b)
	textIs(t, b, "b", text)
	a.showPane("source", false)
	a.openMiddle("source")
	realPaths(t)
	d := a.midDoc()
	if d == nil || !d.inScript() {
		t.Fatal("the source kept in the shared script is not in a's main editor")
	}
	a.fyneWindow.Canvas().Focus(d.ed)
	d.ed.SetCursorOffset(len("First passage"))
	typed := ""
	key := func(r rune) {
		d.ed.TypedRune(r)
		typed += string(r)
	}
	bEdit := func(before, s string) { // b puts s before the first `before` in its script
		bt := b.textEditor.Text()
		i := strings.Index(bt, before)
		if i < 0 {
			t.Fatalf("b has no %q: %q", before, bt)
		}
		at := len([]rune(bt[:i]))
		b.textEditor.Edit(func(e *buffer.Buffer) { e.SetCursor(at, false); e.Insert(s) })
	}
	edits := []struct{ before, s string }{
		{"Rain.", "Wet. "},            // before the block: its place moves
		{"First passage", "Very "},    // in it, before a's cursor
		{"here.\n\nSecond", "truly "}, // in it, after a's cursor
		{"Second passage", "The "},    // and once more after
		{"INT. BARN", "FADE IN:\n\n"}, // at the script's start
		{"passage here.", "long "},    // the second passage, after the cursor
		{"Very First", "A "},          // before the cursor again
		{"Wet. Rain.", "Cold. "},      // before the block again
		{"truly here.", "and "},       // after
		{"The Second long", "Then "},  // after
	}
	for i, r := range " one, two." {
		key(r)
		if i < len(edits) {
			want := typed
			waitFor(t, "b has a's key", func() bool { return strings.Contains(b.textEditor.Text(), want) })
			e := edits[i]
			bEdit(e.before, e.s)
			waitFor(t, "a's source follows b's "+e.s, func() bool { return strings.Contains(a.textEditor.Text(), e.s+e.before) })
		}
	}
	waitFor(t, "the two scripts alike", func() bool { return a.textEditor.Text() == b.textEditor.Text() })
	if !strings.Contains(d.Text(), "A Very First passage"+typed+" and truly here.") {
		t.Errorf("a's keys lost or out of place: %q", d.Text())
	}
	for _, e := range edits {
		if !strings.Contains(a.textEditor.Text(), e.s) {
			t.Errorf("b's %q lost: %q", e.s, a.textEditor.Text())
		}
	}
	if home, _ := d.home(); home != d.homeSeen || strings.TrimRight(d.Text(), "\n") != strings.TrimRight(home, "\n") {
		t.Errorf("a's source %q and its home %q (seen %q)", d.Text(), home, d.homeSeen)
	}
	if c, want := d.ed.CursorOffset(), strings.Index(d.Text(), typed)+len(typed); c != want {
		t.Errorf("a's cursor at %d, after the keys is %d", c, want)
	}
}

// The source and the notes in the middle fold their Markdown
// headings as the script editor folds its own: View > Fold or Unfold
// Here, vim's z keys, Tab at a heading's start, the gutter's marks; each
// editor its own folds. A view in the middle has none.
func TestMiddleFolds(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.openMiddle("source")
	d := w.midDoc()
	text := d.Text()
	d.ed.SetCursorOffset(strings.Index(text, "She opened"))
	cmd("view.foldHere").run(w)
	r := d.folds.ranges()
	if len(r) != 1 {
		t.Fatalf("Fold Here folded %v", r)
	}
	lines := strings.Split(text, "\n")
	if h := lines[r[0].From-1]; h != "## Chapter 1" {
		t.Errorf("folded under %q", h)
	}
	if len(w.scriptFolds.on) != 0 {
		t.Error("the script's folds changed too")
	}
	if !d.folds.headingLines()[strings.Count(text[:strings.Index(text, "# Part One")], "\n")] {
		t.Error("Part One has no fold mark")
	}
	cmd("view.unfoldAll").run(w)
	if len(d.folds.ranges()) != 0 {
		t.Fatal("Unfold All left folds")
	}
	// Tab at a heading's start
	d.ed.SetCursorOffset(strings.Index(text, "## Chapter 2"))
	d.ed.TypedKey(&fyne.KeyEvent{Name: fyne.KeyTab})
	if len(d.folds.ranges()) != 1 || d.Text() != text {
		t.Errorf("Tab at Chapter 2: folds %v, text changed %v", d.folds.ranges(), d.Text() != text)
	}
	// vim's zR
	w.settings.SetString("editor-keys", "vim")
	t.Cleanup(func() { w.settings.SetString("editor-keys", "standard") })
	d.ed.TypedKey(escape())
	d.ed.TypedRune('z')
	d.ed.TypedRune('R')
	if len(d.folds.ranges()) != 0 {
		t.Errorf("zR left %v", d.folds.ranges())
	}
	w.settings.SetString("editor-keys", "standard")
	w.closeMiddle("source")

	// a view in the middle: nothing to fold
	w.showPane("outline", false)
	w.openMiddle("outline")
	cmd("view.foldHere").run(w)
	if !strings.Contains(w.statusText(), "Folding works in an editor") {
		t.Errorf("status %q", w.statusText())
	}
}

// An entry alone: no link quotes the passage, its cut: entry does; the
// cursor leaving the passage edited in the source's tab writes it again,
// one undo step.
func TestRelinkCutOnly(t *testing.T) {
	var kept []string
	for _, l := range strings.Split(sourceScript, "\n") {
		if i := strings.Index(l, "[[from:"); i >= 0 { // no links
			l = strings.TrimSpace(l[:i])
		}
		kept = append(kept, l)
	}
	script := strings.Join(kept, "\n")
	start, end, block := withAdaptationBlock(script, "cut:\n  \"The next morning\"")
	r := []rune(script)
	script = string(r[:start]) + block + string(r[end:])
	w, _ := sourceWindow(t, script, sourceStory)
	w.openMiddle("source")
	d := w.midDoc()
	at := runeAt(t, d.Text(), "The next morning")
	before := w.textEditor.Text()
	d.ed.Edit(func(b *buffer.Buffer) { b.Replace(at+4, at+8, "following") }) // "The following morning"
	d.ed.SetCursorOffset(0)
	if !strings.Contains(w.statusText(), "1 entry of the adaptation block written again") {
		t.Errorf("status %q", w.statusText())
	}
	inside, _ := adaptationBlock(w.textEditor.Text())
	cuts := parseCuts(inside)
	nw := passagesOf(d.Text())
	if p := passageNamed(t, nw, "The following morning"); len(cuts) != 1 || resolveCut(cuts[0], nw) != (cutSpan{p, p}) {
		t.Errorf("the entry %q", inside)
	}
	w.textEditor.Undo()
	if w.textEditor.Text() != before {
		t.Error("the update is not one undo step")
	}
}
