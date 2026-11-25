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
