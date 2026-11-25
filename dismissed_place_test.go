package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// dismissInDocFor keeps dismissed findings in the document's notes block
// for one test (the settings outlive it).
func dismissInDocFor(t *testing.T, w *MainWindow) {
	t.Helper()
	w.settings.SetBoolean(dismissedInDocPref, true)
	t.Cleanup(func() { w.settings.SetBoolean(dismissedInDocPref, false) })
}

const dismissScript = "INT. A - DAY\n\nHi.\n\nINT. B\n\nBye.\n"

// firstFinding is the window's first finding.
func firstFinding(t *testing.T, w *MainWindow) openItem {
	t.Helper()
	for _, it := range w.openItems() {
		if it.Kind == openCheck {
			return it
		}
	}
	t.Fatalf("no finding in %q", w.textEditor.Text())
	return openItem{}
}

func findingListed(w *MainWindow, key string) bool {
	for _, it := range w.openItems() {
		if it.Kind == openCheck && it.dismissKey() == key {
			return true
		}
	}
	return false
}

// TestDismissBesideLooseFile: by default a dismissal goes in a sidecar
// beside the file, the text untouched; opening the file again keeps it
// out; switching the setting moves it into the notes block and back; a
// rename takes the sidecar along.
func TestDismissBesideLooseFile(t *testing.T) {
	w := newTestWindow(t, "")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	p := filepath.Join(t.TempDir(), "film.fountain")
	if err := os.WriteFile(p, []byte(dismissScript), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	f := firstFinding(t, w)
	key := f.dismissKey()
	w.dismissFinding(f)
	if got := w.textEditor.Text(); got != dismissScript {
		t.Errorf("the text changed: %q", got)
	}
	side, err := os.ReadFile(dismissedSidecar(p))
	if err != nil || !parseDismissed(side)[key] {
		t.Fatalf("not in the sidecar (%v): %q", err, side)
	}
	if findingListed(w, key) {
		t.Error("still listed")
	}
	again := newTestWindow(t, "")
	if err := again.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if findingListed(again, key) {
		t.Error("back after opening the file again")
	}
	if w.hasProductionData() {
		t.Error("a dismissal makes Save offer a bundle")
	}

	// in the document: moved into the notes block, the sidecar gone
	cmd := dismissedPlaceCommand()
	cmd.run(w)
	t.Cleanup(func() { w.settings.SetBoolean(dismissedInDocPref, false) })
	if !w.dismissInDocument() {
		t.Fatal("the setting did not switch")
	}
	if got := w.textEditor.Text(); !strings.Contains(got, dismissedHead+"\n- "+key) {
		t.Errorf("not moved into the notes:\n%s", got)
	}
	if _, err := os.Stat(dismissedSidecar(p)); err == nil {
		t.Error("the sidecar stayed")
	}
	if findingListed(w, key) {
		t.Error("listed after the move in")
	}
	// and back beside it, the document clean again
	cmd.run(w)
	if got := w.textEditor.Text(); got != dismissScript {
		t.Errorf("not clean after the move out: %q", got)
	}
	if side, err := os.ReadFile(dismissedSidecar(p)); err != nil || !parseDismissed(side)[key] {
		t.Errorf("not back in the sidecar (%v): %q", err, side)
	}
	if findingListed(w, key) {
		t.Error("listed after the move out")
	}

	// a rename takes it along
	if err := w.renameTo("other.fountain"); err != nil {
		t.Fatal(err)
	}
	q := filepath.Join(filepath.Dir(p), "other.fountain")
	if side, err := os.ReadFile(dismissedSidecar(q)); err != nil || !parseDismissed(side)[key] {
		t.Errorf("the sidecar did not follow the rename (%v): %q", err, side)
	}
	if findingListed(w, key) {
		t.Error("listed after the rename")
	}
}

// TestDismissBesideUnsaved: a document not saved yet keeps its
// dismissals in the extras Accolade keeps for it, its text clean.
func TestDismissBesideUnsaved(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	w := newTestWindow(t, dismissScript)
	f := firstFinding(t, w)
	w.dismissFinding(f)
	if got := w.textEditor.Text(); got != dismissScript {
		t.Errorf("the text changed: %q", got)
	}
	h := w.extrasHome()
	if data, err := h.readExtra(dismissedExtra); err != nil || !parseDismissed(data)[f.dismissKey()] {
		t.Errorf("not in the kept extras (%v): %q", err, data)
	}
	if findingListed(w, f.dismissKey()) {
		t.Error("still listed")
	}
}

// TestDismissBesideBundle: in a bundle the dismissal is its
// assets/production/dismissed.txt, there when it is opened again.
func TestDismissBesideBundle(t *testing.T) {
	holdUI(t)
	w, _ := newDialogTestWindow(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	p := filepath.Join(t.TempDir(), "Script.accolade")
	os.WriteFile(p, makeZip(t, bundleItems("Script.textbundle/")), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	realPaths(t)
	w.textEditor.SetText(dismissScript)
	if err := w.saveToFile(p); err != nil {
		t.Fatal(err)
	}
	f := firstFinding(t, w)
	w.dismissFinding(f)
	if got := w.textEditor.Text(); got != dismissScript {
		t.Errorf("the text changed: %q", got)
	}
	done := false
	w.afterBundleWrites(func() { done = true })
	waitFor(t, "the writes", func() bool { return done && !w.bundleBusy() })
	data, _ := os.ReadFile(p)
	if got, _ := zipFile(t, data, "Script.textbundle/"+dismissedExtra); !parseDismissed([]byte(got))[f.dismissKey()] {
		t.Errorf("not in the bundle: %q", got)
	}
	again, _ := newDialogTestWindow(t)
	if err := again.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if findingListed(again, f.dismissKey()) {
		t.Error("back after opening the bundle again")
	}
}

// proseChecks are the checks' kinds found in text, each with its words.
func proseChecks(text string) map[string][]string {
	out := map[string][]string{}
	for _, it := range proseItemsOf(text) {
		out[it.Check] = append(out[it.Check], it.Raw)
	}
	return out
}

// TestProseEchoContentWords: the proximity echo counts content words
// only, in Dutch, German and English; function words (je, die, dat, das,
// sie, that, would) never.
func TestProseEchoContentWords(t *testing.T) {
	for _, c := range []struct{ lang, text, want string }{
		{"nl", "Als je weet dat je het niet kunt, dan moet je dat zeggen, want die man die daar staat weet dat ook wel en hij zal het je zeggen als je hem vraagt.\n", ""},
		{"de", "Sie sagte, dass sie die Frau, die dort stand, nicht kannte, aber das Haus, das sie sah, war ihr bekannt und sie wusste nicht warum das so war.\n", ""},
		{"en", "She said that she would go, and that he would stay, because that was what they had agreed would happen when the night was over and done.\n", ""},
		{"nl", "De fiets stond daar, en de fiets was al jaren niet meer gebruikt door iemand van de familie of de buren.\n", "fiets"},
		{"de", "Der Hund lag dort, und der Hund schlief dort seit Jahren jeden Abend, wenn die Sonne unterging und es kalt wurde.\n", "hund"},
		{"en", "The bicycle stood there, and the bicycle had not been ridden by anyone in the family for many long years now.\n", "bicycle"},
	} {
		got := proseChecks(c.text)
		if c.want == "" {
			if len(got[checkRepeat]) > 0 {
				t.Errorf("%s: a function word as an echo: %v", c.lang, got[checkRepeat])
			}
			continue
		}
		if r := got[checkRepeat]; len(r) != 1 || !strings.HasPrefix(strings.ToLower(r[0]), c.want) {
			t.Errorf("%s: the echo of %q not found: %v", c.lang, c.want, got)
		}
	}
}

// TestProseDoubledWord: a word twice in a row is a typo, unless the
// language allows it (nl je je, die die, dat dat, er er; de die die, das
// das, sie sie; en had had, that that); with a comma between, never.
func TestProseDoubledWord(t *testing.T) {
	for _, c := range []struct{ lang, text, want string }{
		{"nl", "Als je je vergist, dan is dat dat niet erg, en die die het weet zegt het, want er er zijn er die het weten en ze ze ook zien.\n", ""},
		{"de", "Ich weiß, dass die die Frau kennen, und das das Haus gehört ihr, denn wenn sie sie sieht, weiß sie es und ist nicht allein.\n", ""},
		{"en", "He said that that was fine, and she had had enough of it by then, so they left the house early and walked to the station.\n", ""},
		{"nl", "Hij liep naar de de deur van het huis en keek naar buiten, maar er was niemand te zien op de straat of in de tuin.\n", "de de"},
		{"de", "Er ging zu der Tür und und sah hinaus, aber es war niemand auf der Straße zu sehen und auch nicht im Garten.\n", "und und"},
		{"en", "He walked to the the door of the house and looked out, but there was nobody to be seen in the street or the garden.\n", "the the"},
		{"en", "He walked to the door, the door of the house, and looked out at the street, where nobody was to be seen at all tonight.\n", ""},
	} {
		got := proseChecks(c.text)
		if c.want == "" {
			if len(got[checkDouble]) > 0 {
				t.Errorf("%s: a valid double flagged: %v", c.lang, got[checkDouble])
			}
			continue
		}
		if d := got[checkDouble]; len(d) != 1 || d[0] != c.want {
			t.Errorf("%s: %q not flagged: %v", c.lang, c.want, got)
		}
		if len(got[checkRepeat]) > 0 {
			t.Errorf("%s: the doubled word also an echo: %v", c.lang, got[checkRepeat])
		}
	}
}

// TestProseDoubledUnknownLanguage: a language without a list flags a
// doubled content word, not a short (function) word.
func TestProseDoubledUnknownLanguage(t *testing.T) {
	got := proseChecks("Kotka istui puun oksalla ja katseli katseli alas laaksoon, missä joki virtasi hiljaa kohti järveä.\n")
	if d := got[checkDouble]; len(d) != 1 || d[0] != "katseli katseli" {
		t.Errorf("a doubled content word: %v", got)
	}
	got = proseChecks("Kotka on on puun oksalla ja katseli alas laaksoon, missä joki virtasi hiljaa kohti järveä.\n")
	if d := got[checkDouble]; len(d) != 0 {
		t.Errorf("a short word flagged without a list: %v", d)
	}
}
