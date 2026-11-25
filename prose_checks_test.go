package main

import (
	"os"
	"strings"
	"testing"
)

// TestProseFindingsReal counts the prose checks' findings on a real
// story (Chekhov's, in Garnett's English): a page gets few, never a
// flood.
func TestProseFindingsReal(t *testing.T) {
	b, err := os.ReadFile("examples/a-chameleon.md")
	if err != nil {
		t.Skip(err)
	}
	items := proseItemsOf(string(b))
	for _, it := range items {
		t.Logf("%d %s: %s [%s]", it.Line, it.Check, it.Text, firstLine(it.Raw))
	}
	if words := len(strings.Fields(string(b))); len(items) == 0 || len(items)*100 > words {
		t.Errorf("%d findings in %d words: want a few", len(items), words)
	}
}

// TestProseKeepAndDismiss: Keep writes the finding under its paragraph
// and the note then stands for it; a dismissed one is left out by the
// text alone (as after opening the file again).
func TestProseKeepAndDismiss(t *testing.T) {
	text := "# One\n\nThe door was open and the door was\nshut by the wind.\n\nNext paragraph here.\n"
	items := proseItemsOf(text)
	if len(items) != 1 || items[0].Check != checkRepeat || items[0].Raw != "door was open and the door" {
		t.Fatalf("findings: %+v", items)
	}
	it := items[0]
	if !it.still(text) || !it.keepable() {
		t.Fatalf("not still or not keepable: %+v", it)
	}
	kept := it.keepEdit().apply(text)
	if !strings.Contains(kept, "wind.\n<!-- revision: “door” twice in 6 words: another word, or one less? -->\n\nNext") {
		t.Fatalf("kept:\n%s", kept)
	}
	if f := proseItemsOf(kept); len(f) != 0 {
		t.Errorf("the kept note does not stand for its finding: %+v", f)
	}
	var rev []openItem
	for _, o := range openItemsOf(kept, nil, true) {
		if o.Kind == openRevision {
			rev = append(rev, o)
		}
	}
	if len(rev) != 1 || !strings.HasPrefix(rev[0].Text, "“door” twice") {
		t.Errorf("revision note not listed: %+v", rev)
	}
	// dismissed: a line in the notes block
	m := markdownNotes
	from, block := m.edit(text, withDismissed("an unsorted note", []string{it.dismissKey()}))
	dis := textEdit{from, len([]rune(text)), block}.apply(text)
	gone, rest := dismissedIn(strings.ReplaceAll(m.of(dis), m.escaped, m.close))
	if !gone[it.dismissKey()] || rest != "an unsorted note" {
		t.Errorf("dismissed %v, rest %q, in:\n%s", gone, rest, dis)
	}
	if !strings.Contains(dis, "<!-- Notes\nan unsorted note\n\n"+dismissedHead) {
		t.Errorf("notes block:\n%s", dis)
	}
}

// TestProseNotEnglish: filter words and the passive are English's.
func TestProseNotEnglish(t *testing.T) {
	text := "La hundo estis vidita kaj estis prenita kaj estis batita, kaj li sentis, vidis kaj aŭdis ĉion en la ĝardeno de la domo hieraŭ vespere kun la amikoj.\n"
	for _, it := range proseItemsOf(text) {
		if it.Check == checkFilter || it.Check == checkPassive {
			t.Errorf("an English check on Esperanto: %+v", it)
		}
	}
}

// TestProseRepeatsOtherLanguage: in Esperanto prose its common words
// (estis, sed, kiam…) are not repetitions.
func TestProseRepeatsOtherLanguage(t *testing.T) {
	para := "Kiam la vespero venis, la virino estis ankoraŭ en la ĝardeno, kaj ŝi estis laca sed kontenta. " +
		"La infanoj estis en la domo, kaj kiam ili vokis ŝin, ŝi ne respondis, ĉar ŝi estis aŭskultanta la birdojn. " +
		"Poste ŝia frato venis el la urbo kun leteroj, sed neniu el ili estis por ŝi, kaj tio ŝin ne surprizis.\n\n"
	text := strings.Repeat(para, 6)
	for _, it := range proseItemsOf(text) {
		t.Errorf("a finding in plain Esperanto: %s [%s]", it.Text, it.Raw)
	}
	// a Hindi paragraph's dandas end its sentences: none of them long
	hi := strings.Repeat("वह बगीचे में बैठी रही और शाम को देखती रही। ", 12) + "\n"
	for _, it := range proseItemsOf(hi) {
		if it.Check == checkSentence {
			t.Errorf("one long sentence across dandas: %s", it.Text)
		}
	}
}

// TestRevisionSourceInMiddle: the source in front in the middle gets
// the prose checks, stepped through in its own editor; Dismiss keeps the
// finding in the script's notes; a Dismiss in the script, the source not
// open, lets none of the source's go.
func TestRevisionSourceInMiddle(t *testing.T) {
	story := sourceStory + "\nThe rain fell on the roof, the rain on the road.\n"
	w, _ := sourceWindow(t, sourceScript, story)
	dismissInDocFor(t, w) // in the script's notes block
	w.openMiddle("source")
	d := w.midDoc()
	if ed, src := w.revEditor(); !src || ed != d.ed {
		t.Fatal("the Revision tab is not in the source")
	}
	d.ed.SetCursorOffset(0)
	w.stepOpen(1)
	it, ok := w.currentItem("test")
	if !ok || it.Check != checkRepeat || d.ed.CursorOffset() != it.Start {
		t.Fatalf("not on the source's repeat: %+v; items %+v; status %q; text %q", it, w.revItems(), w.statusText(), d.Text())
	}
	script := w.textEditor.Text()
	w.dismissCurrent()
	if got := w.textEditor.Text(); !strings.Contains(got, "- "+it.dismissKey()) || d.Text() != story {
		t.Fatalf("dismissed into the script's notes:\n%s", got)
	}
	for _, f := range w.revItems() {
		if f.Kind == openCheck && f.Check == checkRepeat {
			t.Errorf("still listed: %+v", f)
		}
	}
	// the source closed, a script finding dismissed: the source's stays
	w.textEditor.SetText(script + "\nINT. SHED\n\nDark.\n")
	w.textEditor.SetText(w.textEditor.Text() + "\n/* Notes\n" + dismissedHead + "\n- " + it.dismissKey() + "\n*/\n")
	w.srcDoc = nil
	w.mid.shown = ""
	for _, f := range w.openItems() {
		if f.Kind == openCheck {
			w.dismissFinding(f)
			break
		}
	}
	if !w.dismissedOf()[it.dismissKey()] {
		t.Errorf("the source's dismissal let go:\n%s", w.textEditor.Text())
	}
}

// TestDismissUnclosedNote: a finding about an unclosed [[ dismissed
// leaves no finding behind in the notes block.
func TestDismissUnclosedNote(t *testing.T) {
	w := newTestWindow(t, "INT. A - DAY\n\nHi. [[ oops\n\nINT. B - DAY\n\nBye.\n")
	var f openItem
	for _, it := range w.openItems() {
		if it.Kind == openCheck {
			f = it
		}
	}
	if f.Kind != openCheck {
		t.Fatal("no finding")
	}
	w.dismissFinding(f)
	for _, it := range w.openItems() {
		if it.Kind == openCheck {
			t.Errorf("after Dismiss: %+v in\n%s", it, w.textEditor.Text())
		}
	}
}
