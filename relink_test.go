package main

import (
	"maps"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// withEdits is text with edits made (the last first, as relinkFixes
// gives them).
func withEdits(text string, edits []linkEdit) string {
	r := []rune(text)
	for _, e := range edits {
		r = append(append(append([]rune(nil), r[:e.Start]...), []rune(e.Text)...), r[e.End:]...)
	}
	return string(r)
}

// A passage's words edited: the links that found it are written again
// for it (re-anchored), the adaptation block's entries with them; the
// others are left as they are, and nothing changed writes nothing.
func TestRelinkReanchors(t *testing.T) {
	old := passagesOf(sourceStory)
	start, end, block := withAdaptationBlock(sourceScript, "cut:\n  \"She opened the door\"\n  \"The next morning\"")
	script := []rune(sourceScript)
	text := string(script[:start]) + block + string(script[end:])
	nw := passagesOf(strings.Replace(sourceStory, "She opened the door to the garden", "She pushed open the door of the garden", 1))
	edits, c, moved := relinkFixes(text, old, nw)
	if c.reanchored != 1 || c.entries != 1 || c.lengthened != 0 || c.warned != 0 || len(moved) != 1 {
		t.Fatalf("%+v, moved %v: %v", c, moved, edits)
	}
	got := withEdits(text, edits)
	if strings.Contains(got, "She opened the door to") {
		t.Fatalf("the link is as it was:\n%s", got)
	}
	m := newSourceModel(nw, got, nil)
	at := passageNamed(t, nw, "She pushed")
	if lines := m.linked(at); len(lines) != 1 {
		t.Errorf("the new link finds %v", lines)
	}
	inside, _ := adaptationBlock(got)
	if cuts := parseCuts(inside); len(cuts) != 2 || resolveCut(cuts[0], nw) != (cutSpan{at, at}) || cuts[1].String() != `"The next morning"` {
		t.Errorf("the entries: %q", inside)
	}
	if !strings.Contains(c.String(), "1 link re-anchored") {
		t.Errorf("said %q", c.String())
	}
	if edits, c, _ := relinkFixes(text, old, old); len(edits) != 0 {
		t.Errorf("nothing changed, yet %+v", c)
	}
	// the change not known (a script opened): a near link alone
	near := strings.Replace(sourceScript, "She opened the door to the garden and stepped…", "She opened a door to the garden and stepped…", 1)
	edits, c, _ = relinkFixes(near, nil, old)
	if c.reanchored != 1 || len(edits) != 1 || !strings.Contains(edits[0].Text, "#:~:text=") {
		t.Errorf("the near link: %+v %v", c, edits)
	}
}

// A passage added elsewhere that the short link fits as well: the link
// grows until it finds its own passage alone again.
func TestRelinkLengthens(t *testing.T) {
	old := passagesOf(sourceStory)
	at := passageNamed(t, old, "She opened")
	d, _ := fragmentFor(newLinkMatcher(old), at, at)
	script := "INT. HOUSE - NIGHT\n\n" + fragmentNote("", []textDirective{d}) + "\n"
	story := strings.Replace(sourceStory, "## Chapter 1\n", "## Chapter 1\n\nShe waited for the rain.\n", 1)
	nw := passagesOf(story)
	if mt := newLinkMatcher(nw).matchRef(linkRef{Dir: d}); mt.Kind != linkAmbiguous {
		t.Fatalf("the test's link %+v is %+v in the new story", d, mt)
	}
	edits, c, _ := relinkFixes(script, old, nw)
	if c.lengthened != 1 || len(edits) != 1 {
		t.Fatalf("%+v: %v", c, edits)
	}
	got := withEdits(script, edits)
	m := newSourceModel(nw, got, nil)
	if want := passageNamed(t, nw, "She opened"); m.matches[0].Kind != linkFound || m.matches[0].Passages[0] != want {
		t.Errorf("%q finds %+v", got, m.matches[0])
	}
	if len(got) <= len(script) {
		t.Errorf("not longer: %q", got)
	}
}

// When nothing can tell the passage apart (the story and its copy), the
// link warns: its chip says "↗ 2 places", the pane why. When its passage
// is gone but its words fit another, it is written in the passage's
// whole words: not found, "↗ source?".
func TestRelinkImpossible(t *testing.T) {
	story := "Morning came.\n\nShe opened the door to the rain.\n\nHe slept.\n"
	old := passagesOf(story)
	d, unique := fragmentFor(newLinkMatcher(old), 1, 1)
	if !unique {
		t.Fatal("not unique to start with")
	}
	script := "INT. HOUSE - NIGHT\n\n" + fragmentNote("", []textDirective{d}) + "\n"
	twice := passagesOf(story + "\n" + story)
	edits, c, _ := relinkFixes(script, old, twice)
	if len(edits) != 1 || c.warned != 1 {
		t.Fatalf("alike: %+v %v", c, edits)
	}
	alike := withEdits(script, edits) // the words that fit both, a few more: no later edit makes it find one of them by chance
	if !strings.Contains(alike, "#:~:text=Morning came.-,She opened the door to the rain.,-He slept.]]") {
		t.Errorf("alike: %q", alike)
	}
	if again, _, _ := relinkFixes(alike, twice, twice); len(again) != 0 {
		t.Errorf("written again and again: %v", again)
	}
	lm := newLinkMatcher(twice)
	chips := findLinkChips(alike, lm)
	if len(chips) != 1 || chips[0].Label != "↗ 2 places" || !strings.Contains(chips[0].Tip, "fits 2 places in the source alike") {
		t.Errorf("the chip: %+v", chips)
	}
	// gone, and its short words found in a passage added later
	gone := passagesOf("Morning came.\n\nHe slept.\n\nShe danced in the rain.\n")
	if mt := newLinkMatcher(gone).matchRef(linkRef{Dir: d}); !mt.found() {
		t.Fatalf("the test's link %+v is %+v without its passage", d, mt)
	}
	edits, c, _ = relinkFixes(script, old, gone)
	if c.warned != 1 || len(edits) != 1 {
		t.Fatalf("gone: %+v %v", c, edits)
	}
	got := withEdits(script, edits)
	if !strings.Contains(got, "#:~:text=Morning came.-,She opened the door to the rain.]]") {
		t.Errorf("gone: %q", got)
	}
	chips = findLinkChips(got, newLinkMatcher(gone))
	if len(chips) != 1 || chips[0].Label != "↗ source?" || !strings.Contains(chips[0].Tip, "taken out") {
		t.Errorf("the chip: %+v", chips)
	}
	// and it comes back: found again
	if mt := newLinkMatcher(old).matchRef(parseLinks(got)[0].Ref); mt.Kind != linkFound || mt.Passages[0] != 1 {
		t.Errorf("the passage back: %+v", mt)
	}
}

// In the window: the cursor leaving a passage of the source edited in
// the main editor writes the links again at once, one undo step of the
// script's, the status line saying so; the Source pane's Done, the
// source's file read again, and a script opened with a near link as well.
func TestRelinkInWindow(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.openMiddle("source")
	d := w.midDoc()
	at := runeAt(t, d.Text(), "She opened")
	script := w.textEditor.Text()
	d.ed.Edit(func(b *buffer.Buffer) { b.Replace(at+4, at+10, "pushed") }) // "She pushed the door…"
	if w.textEditor.Text() != script {
		t.Fatal("written again while still in the passage")
	}
	d.ed.SetCursorOffset(runeAt(t, d.Text(), "The next morning"))
	if strings.Contains(w.textEditor.Text(), "She opened the door") || !strings.Contains(w.statusText(), "1 link re-anchored") {
		t.Fatalf("not written again (%q):\n%s", w.statusText(), w.textEditor.Text())
	}
	w.textEditor.Undo()
	if w.textEditor.Text() != script {
		t.Error("not one undo step")
	}
	w.closeMiddle("source")

	// the pane's Edit and Done
	w2, _ := sourceWindow(t, sourceScript, sourceStory)
	s := w2.sourcePane()
	s.toggleEdit()
	s.w.srcDoc.SetText(strings.Replace(sourceStory, "The next morning the garden", "Next day the garden", 1))
	s.toggleEdit()
	if strings.Contains(w2.textEditor.Text(), "The next morning the garden") {
		t.Errorf("Done did not write the link again:\n%s", w2.textEditor.Text())
	}

	// the file changed on disk, read again
	w3, p := sourceWindow(t, sourceScript, sourceStory)
	story := filepath.Join(filepath.Dir(p), "story.md")
	os.WriteFile(story, []byte(strings.Replace(sourceStory, "The next morning the garden", "Next day the garden", 1)), 0o644)
	later := time.Now().Add(time.Minute)
	os.Chtimes(story, later, later)
	w3.sourcePane().follow()
	if strings.Contains(w3.textEditor.Text(), "The next morning the garden") || !strings.Contains(w3.statusText(), "re-anchored") {
		t.Errorf("the file read again: the link not written again (%q):\n%s", w3.statusText(), w3.textEditor.Text())
	}

	// opened again after its source changed while it was closed: from the
	// source as it was when the script was saved
	w4, p4 := sourceWindow(t, sourceScript, sourceStory)
	if err := w4.SaveFile(); err != nil {
		t.Fatal(err)
	}
	story4 := filepath.Join(filepath.Dir(p4), "story.md")
	os.WriteFile(story4, []byte(strings.Replace(sourceStory, "The next morning the garden", "Next day the garden", 1)), 0o644)
	w5, _ := newDialogTestWindow(t)
	if err := w5.LoadFile(p4); err != nil {
		t.Fatal(err)
	}
	w5.togglePane("source")
	if strings.Contains(w5.textEditor.Text(), "The next morning the garden") || !strings.Contains(w5.statusText(), "re-anchored") {
		t.Errorf("opened: the link not written again (%q):\n%s", w5.statusText(), w5.textEditor.Text())
	}
	// not saved: opened again, written again; saved, as it is
	w6, _ := newDialogTestWindow(t)
	if err := w6.LoadFile(p4); err != nil {
		t.Fatal(err)
	}
	w6.togglePane("source")
	if w6.textEditor.Text() != w5.textEditor.Text() {
		t.Errorf("opened again, not saved: %s", w6.textEditor.Text())
	}
	if err := w6.SaveFile(); err != nil {
		t.Fatal(err)
	}
	saved := w6.textEditor.Text()
	w6, _ = newDialogTestWindow(t)
	if err := w6.LoadFile(p4); err != nil {
		t.Fatal(err)
	}
	w6.togglePane("source")
	if w6.textEditor.Text() != saved || w6.hasChanges {
		t.Errorf("opened, saved after the change: written again:\n%s", w6.textEditor.Text())
	}
}

// A passage split in two: a link to it goes to the part most like it;
// a passage gone goes nowhere, though another took its place.
func TestAlignPassages(t *testing.T) {
	old := passagesOf(sourceStory)
	at := passageNamed(t, old, "She opened")
	split := passagesOf(strings.Replace(sourceStory, "and stepped out", "and stopped.\n\nShe stepped out", 1))
	if to := alignPassages(old, split); to[at] != at && to[at] != at+1 {
		t.Errorf("the split passage maps to %d", to[at])
	}
	// one moved from the start to the end: the others stay where they are
	moved := passagesOf(strings.Replace(sourceStory, "She opened the door to the garden and stepped out into the *rain*.\n\n", "", 1) +
		"\nShe opened the door to the garden and stepped out into the *rain*.\n")
	if to := alignPassages(old, moved); to[at] != len(moved)-1 || to[at+1] != at {
		t.Errorf("moved: %v", to)
	}
	other := passagesOf(strings.Replace(sourceStory, "She opened the door to the garden and stepped out into the *rain*.", "A wholly different paragraph now.", 1))
	if to := alignPassages(old, other); to[at] != -1 {
		t.Errorf("a passage written over maps to %d", to[at])
	}
}

// nearly reports whether a passage's words are b's, or nearly (as a
// near link finds them: a quarter of them edited).
func nearly(a, b string) bool {
	wa, wb := normWords(a), normWords(b)
	limit := max(1, len(wa)/4)
	return prefixDistance(wa, wb, limit) <= limit || prefixDistance(wb, wa, limit) <= limit // either way, as a near link's words
}

// Random edits of a source, the links looked at again after each, as
// the window does: every link finds its own passage (or, it gone, one
// with nearly its words: a copy; or one word for word the same: it is
// the link's then; or one it was ambiguous with, once the others
// changed; or its twin, word for word, before the step), or warns
// (ambiguous, or not found). Never another passage. A small vocabulary, so short links collide.
func TestRelinkProperty(t *testing.T) {
	vocab := strings.Fields("rain door sea wind she he opened walked the a garden light night stone river boat " +
		"kettle stairs lamp bread salt road hill field bell cold warm late early old new red grey")
	seeds := 30
	if testing.Short() {
		seeds = 5
	}
	fixed, warned, checked := 0, 0, 0
	for seed := int64(1); seed <= int64(seeds); seed++ {
		rng := rand.New(rand.NewSource(seed))
		para := func() string {
			n := 3 + rng.Intn(6)
			ws := make([]string, n)
			for k := range ws {
				ws[k] = vocab[rng.Intn(len(vocab))]
			}
			return strings.Join(ws, " ") + "."
		}
		type par struct {
			id   int
			text string
		}
		var paras []par
		next := 0
		for range 12 {
			paras = append(paras, par{next, para()})
			next++
		}
		lastText := map[int]string{}
		source := func() string {
			var b strings.Builder
			for _, p := range paras {
				b.WriteString(p.text + "\n\n")
				lastText[p.id] = p.text
			}
			return b.String()
		}
		src := source()
		ps := passagesOf(src)
		lm := newLinkMatcher(ps)
		var script strings.Builder
		var linkIDs []int
		for i := range ps {
			if d, unique := fragmentFor(lm, i, i); unique {
				script.WriteString(fragmentNote("", []textDirective{d}) + "\n\n")
				linkIDs = append(linkIDs, paras[i].id)
			}
		}
		text := script.String()
		alikeIDs := map[int]map[int]bool{} // a link's passages alike when it was ambiguous, by the link
		for step := range 40 {
			for range 1 + rng.Intn(2) {
				switch op := rng.Intn(5); {
				case op == 0 || len(paras) < 4: // a passage added
					k := rng.Intn(len(paras) + 1)
					paras = append(paras[:k], append([]par{{next, para()}}, paras[k:]...)...)
					next++
				case op == 1: // one taken out
					k := rng.Intn(len(paras))
					paras = append(paras[:k], paras[k+1:]...)
				case op == 2: // a word changed
					k := rng.Intn(len(paras))
					ws := strings.Fields(strings.TrimSuffix(paras[k].text, "."))
					ws[rng.Intn(len(ws))] = vocab[rng.Intn(len(vocab))]
					paras[k].text = strings.Join(ws, " ") + "."
				case op == 4: // one moved
					k, to := rng.Intn(len(paras)), rng.Intn(len(paras))
					m := paras[k]
					paras = append(paras[:k], paras[k+1:]...)
					paras = append(paras[:to], append([]par{m}, paras[to:]...)...)
				default: // one copied elsewhere
					k, to := rng.Intn(len(paras)), rng.Intn(len(paras)+1)
					c := par{next, paras[k].text}
					next++
					paras = append(paras[:to], append([]par{c}, paras[to:]...)...)
				}
			}
			old := ps
			before := maps.Clone(lastText)
			src = source()
			ps = passagesOf(src)
			if len(ps) != len(paras) {
				t.Fatalf("seed %d step %d: %d passages of %d paragraphs", seed, step, len(ps), len(paras))
			}
			edits, c, _ := relinkFixes(text, old, ps)
			fixed += c.lengthened + c.reanchored
			text = withEdits(text, edits)
			lm := newLinkMatcher(ps)
			links := parseLinks(text)
			if len(links) != len(linkIDs) {
				t.Fatalf("seed %d step %d: %d links, %d made", seed, step, len(links), len(linkIDs))
			}
			current := func(id int) string { // the passage's text now, "" if gone
				for _, pp := range paras {
					if pp.id == id {
						return pp.text
					}
				}
				return ""
			}
			for k, l := range links {
				checked++
				mt := lm.matchRef(l.Ref)
				if !mt.found() {
					warned++
					if mt.Kind == linkAmbiguous {
						if alikeIDs[k] == nil {
							alikeIDs[k] = map[int]bool{}
						}
						for _, q := range mt.Passages {
							alikeIDs[k][paras[q].id] = true
						}
					}
					continue
				}
				p := mt.Passages[0]
				switch {
				case mt.last() == p && paras[p].id == linkIDs[k]:
				case mt.last() == p && (paras[p].text == current(linkIDs[k]) || paras[p].text == before[linkIDs[k]]):
					linkIDs[k] = paras[p].id // a copy, word for word (of its words now, or before the step's edit): told apart by nothing
				case mt.last() == p && current(linkIDs[k]) == "" && (nearly(paras[p].text, lastText[linkIDs[k]]) || nearly(paras[p].text, before[linkIDs[k]])):
					linkIDs[k] = paras[p].id // its own gone, a copy of it (nearly) is the link's from now on
				case mt.last() == p && before[paras[p].id] == before[linkIDs[k]] && before[linkIDs[k]] != "":
					linkIDs[k] = paras[p].id // word for word its twin before the step: which is which, no words tell
				case mt.last() == p && alikeIDs[k][paras[p].id]:
					linkIDs[k] = paras[p].id // one it could not be told from, the others changed since
				default:
					t.Errorf("seed %d step %d: the link %q to %d (%q) finds %d..%d (%d, %q)", seed, step, l.Ref.Dir.String(false),
						linkIDs[k], lastText[linkIDs[k]], p, mt.last(), paras[p].id, paras[p].text)
				}
				delete(alikeIDs, k) // found: one passage again
			}
			if t.Failed() {
				return
			}
		}
	}
	t.Logf("%d links looked at, %d written again, %d warning", checked, fixed, warned)
	if fixed == 0 {
		t.Error("nothing was written again: the edits do not test it")
	}
}

// Shots of the warning chips (SHOT_DIR as in screenshot_test.go): a
// link that fits two places alike, one whose passage was taken out, the
// Source pane saying why.
func TestRelinkScreenshots(t *testing.T) {
	if os.Getenv("SHOT_DIR") == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	story := "Morning came.\n\nShe opened the door to the rain.\n\nHe slept.\n"
	script := "INT. HOUSE - NIGHT\n\nAnna at the door. [[from: #:~:text=Morning came.-,She opened the door to the rain.,-He slept.]]\n\n" +
		"EXT. GARDEN - NIGHT\n\nRain. [[from: #:~:text=Morning came.-,The kettle sang on the stove.]]\n\nINT. HOUSE - DAY\n\n[[from: #:~:text=He,slept.]]\n"
	w, _ := sourceWindow(t, script, story+"\n"+story)
	w.fyneWindow.Resize(fyne.NewSize(1100, 650))
	for _, at := range []struct{ name, line string }{{"places", "Anna at the door."}, {"lost", "Rain."}} {
		w.textEditor.SetCursorOffset(runeAt(t, w.textEditor.Text(), at.line))
		w.sourcePane().follow()
		w.fyneWindow.Content().Refresh()
		shotTo(t, w, "relink-chip-"+at.name)
	}
}

// A look at the links dropped (the script changed while it was worked
// out, on the desktop in the background): the pane's next follow looks
// again, from the source the links were last looked at against.
func TestRelinkDroppedLookAgain(t *testing.T) {
	changed := strings.Replace(sourceStory, "The next morning the garden", "Next day the garden", 1)
	w, _ := sourceWindow(t, sourceScript, changed)
	s := w.sourcePane()
	if !strings.Contains(w.textEditor.Text(), "The next morning the garden") {
		t.Fatal("written again on opening the first time")
	}
	// as a dropped look leaves it: the links last looked at against the story before the change
	s.linkedSrc = sourceStory
	if w.relink(w.textEditor.Text()+"x", []linkEdit{{0, 0, "y"}}, relinkCount{}, nil) {
		t.Fatal("a look for another script applied")
	}
	s.follow()
	if strings.Contains(w.textEditor.Text(), "The next morning the garden") || !sameString(s.linkedSrc, s.src) {
		t.Errorf("not looked at again:\n%s", w.textEditor.Text())
	}
	before := w.textEditor.Text()
	s.follow()
	if w.textEditor.Text() != before {
		t.Error("looked at again and again")
	}
}
