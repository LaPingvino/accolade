package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/lexington/lex"
)

// The sample (sample.go): examples/a-chameleon.md is the story, as taken
// from Project Gutenberg's eBook #13417 (its licence and header left
// out), with front matter; examples/a-chameleon.fountain the adaptation
// as typed, with {{n}} where a link to passage n goes and {{i-j}} where
// one spans passages i to j. TestMakeSample (UPDATE_SAMPLE=1) builds
// examples/sample.accolade from them through Accolade's own commands, as
// a writer would: Save As .accolade, Choose Source, Link to Source
// Passage, Mark as Cut, the Characters pane, Lock Scene Numbers, Omit
// Scene, Save Draft, Lock Pages, Issue, Tag Selection, the stripboard.
// TestSampleFresh builds it again and checks it, so the generator keeps
// working as the formats change; TestSample checks the one the app
// carries, opened as File > Open Sample opens it.

// The sample's dates: fixed, so it says the same each time it is made.
var (
	sampleWhite = time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	sampleBlue  = time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)
)

// sampleNotes is the script's Notes.
const sampleNotes = `This is Accolade's sample script (File > Open Sample), made by its own commands to try every feature on: see examples/README.md in Accolade's source.

The story: "A Chameleon" by Anton Chekhov, in Constance Garnett's English translation (The Cook's Wedding and Other Stories, The Tales of Chekhov vol. 12). Text from Project Gutenberg eBook #13417, whose catalogue record says: "Public domain in the USA." Outside the USA, check your country's law before using it.

Partly adapted on purpose: the last two passages are still to adapt (Adaptation > Next Source Passage), and two are marked as cut.`

// sampleMarker is where a link goes in the adaptation as typed.
var sampleMarker = regexp.MustCompile(` ?\{\{(\d+)(?:-(\d+))?\}\}`)

// makeSample builds the sample and gives its bytes.
func makeSample(t *testing.T) []byte {
	t.Helper()
	holdUI(t)
	script, err := readLF(filepath.Join("examples", "a-chameleon.fountain"))
	if err != nil {
		t.Fatal(err)
	}
	story, err := readLF(filepath.Join("examples", "a-chameleon.md"))
	if err != nil {
		t.Fatal(err)
	}
	w := draftsWindow(t)
	e := w.textEditor
	e.SetText(string(script))

	// File > Save As… an .accolade
	p := filepath.Join(t.TempDir(), sampleName+".accolade")
	w.saveAs(p)
	if w.bundle == nil {
		t.Fatal("not saved as a bundle")
	}
	// Adaptation > Choose Source…: in the bundle (assets/source.md)
	if err := w.useSourceText("a-chameleon.md", string(story)); err != nil {
		t.Fatal(err)
	}
	if r := w.sourceRef(); r.Origin != sourceBundle {
		t.Fatalf("the source is not in the bundle: %+v", r)
	}

	// Link to Source Passage at each marker, the passage picked in the pane
	w.showPane("source", false)
	s := w.sourcePane()
	s.load()
	for {
		text := e.Text()
		m := sampleMarker.FindStringSubmatchIndex(text)
		if m == nil {
			break
		}
		from, to := len([]rune(text[:m[0]])), len([]rune(text[:m[1]]))
		i, _ := strconv.Atoi(text[m[2]:m[3]])
		e.Replace(from, to, "")
		if m[4] < 0 { // one passage: the command itself
			s.list.Select(i)
			e.SetCursorOffset(from)
			w.linkToSource()
			if !strings.HasPrefix(w.statusText(), "Linked to") {
				t.Fatalf("passage %d: %q", i, w.statusText())
			}
			continue
		}
		// a span of passages: written as the command writes one, over them all
		j, _ := strconv.Atoi(text[m[4]:m[5]])
		d, unique := fragmentFor(matcherFor(s.passages), i, j)
		if !unique {
			t.Fatalf("passages %d-%d: no link tells them apart", i, j)
		}
		at, insert, _ := insertLink(e.Text(), from, fragmentNote("", []textDirective{d}), w.sceneStarts())
		e.Replace(at, at, insert)
	}
	// Mark Source Passage as Cut: "Do you know that for a fact?" and its answer
	s.load()
	for _, i := range []int{16, 17} {
		s.list.Select(i)
		s.toggleCut()
	}

	// the Characters pane: notes and fields, an alias, a name listed who
	// does not speak (yet)
	w.showPane("characters", false)
	cp := w.charactersPane()
	cp.update(e.Text())
	entry := func(key, note string, fields map[int]string) {
		cp.update(e.Text())
		if _, ok := cp.charOf(key); !ok {
			t.Fatalf("%s not in the pane", key)
		}
		cp.openChar(key)
		cp.detail.note.SetText(note)
		for f := range fieldCount {
			if v := fields[f]; v != "" {
				cp.detail.fields[f].SetText(v)
			}
		}
		cp.closeDetail()
	}
	entry("OTCHUMYELOV", "The police superintendent. Changes his mind with every guess at whose dog it is.",
		map[int]string{fieldAge: "50s", fieldLook: "a new overcoat, on and off", fieldArc: "stern, then fawning, then stern again"})
	entry("YELDYRIN", "The policeman. Thinks aloud at the worst moments.", map[int]string{fieldLook: "red-haired, a sieve of gooseberries"})
	entry("HRYUKIN", "The goldsmith, half drunk, holding up a bitten finger.", map[int]string{fieldAge: "40s"})
	entry("PROHOR", "The General's cook.", nil)
	cp.addAlias("OTCHUMYELOV", "the police superintendent, the superintendent")
	cp.addAlias("YELDYRIN", "the policeman")
	if start, end, block, ok := addedName(e.Text(), "General Zhigalov"); ok {
		e.Edit(func(b *buffer.Buffer) { replaceLeast(b, start, end, block) })
	} else {
		t.Fatal("General Zhigalov not added to the list")
	}
	entry("GENERAL ZHIGALOV", "Never seen. Whose dog it is decides everything.", nil)

	// what Accolade learned: names that are no one, a place
	inside, _ := adaptationBlock(e.Text())
	l := parseLearned(inside)
	l.Not = append(l.Not, "Petersburg", "Moscow")
	l.Places = append(l.Places, placeEntry{Place: "the timber-yard", Heading: "EXT. PITCHUGIN'S TIMBER-YARD"})
	w.writeLearned(l, false)

	// Notes
	w.writeNotes(sampleNotes)

	// File > Save Draft…: the writer's draft
	saveDraftNow(t, w, "First draft")

	// Production > Lock Scene Numbers, Omit Scene (the tavern), and two
	// scenes written after the lock: A-numbers
	w.lockSceneNumbers()
	e.SetCursorOffset(runeIndex(t, e.Text(), "Sleepy drinkers"))
	w.omitScene()
	// before the timber-yard's first scene: the dog gets out
	at := runeIndex(t, e.Text(), "EXT. PITCHUGIN'S TIMBER-YARD - DAY")
	w.fyneWindow.Canvas().Focus(e)
	e.SetCursorOffset(at)
	test.Type(e, "EXT. GENERAL ZHIGALOV'S YARD - DAY\n\nA white borzoy puppy noses the gate open and slips out into the lane.\n\n")
	e.SetCursorOffset(0) // the cursor leaves the heading: numbered
	if h := "EXT. GENERAL ZHIGALOV'S YARD - DAY #1A#"; !strings.Contains(e.Text(), h) {
		t.Fatalf("no %q:\n%s", h, strings.Join(headsOf(w), "\n"))
	}
	// the heading teaches nothing: its scene links no passage, and none
	// reads as a place it renames ("God's world" once did)
	if pl := learnedOf(e.Text()).Places; len(pl) != 1 {
		t.Fatalf("places learned: %v", pl)
	}

	// Production > Lock Pages
	lockNow(t, w)

	// Tag Selection… in several categories; Prohor's basket goes in Blue: lost
	tag := func(words, category string) { // in the script's words, not a link's
		text := e.Text()
		i := runeIndex(t, text, words)
		for _, l := range parseLinks(text) {
			if i >= l.Start && i < l.End {
				j := strings.Index(string([]rune(text)[l.End:]), words)
				if j < 0 {
					t.Fatalf("%q only in a link", words)
				}
				i = l.End + len([]rune(string([]rune(text)[l.End:])[:j]))
			}
		}
		e.Select(i, i+len([]rune(words)))
		w.tagSelectionAs(category)
	}
	tag("new overcoat", "Wardrobe")
	tag("parcel", "Props")
	tag("sieve full of confiscated gooseberries", "Props")
	tag("white borzoy puppy", "Animals")
	tag("bleeding finger", "Makeup/Hair")
	tag("crowd has sprung out of the earth", "Background Actors")
	tag("basket", "Props")

	// the stripboard: two shoot days
	sb := board(t, w)
	w.addShootDay()
	w.addShootDay()
	_, bt, _ := w.scheduleFound()
	day := map[int][]int{} // the scenes by the day they are shot
	for i, sc := range bt.scenes {
		switch h := strings.ToLower(sc.heading); {
		case strings.Contains(h, "omitted"):
		case strings.Contains(h, "zhigalov"):
			day[1] = append(day[1], i)
		default:
			day[0] = append(day[0], i)
		}
	}
	for d := range 2 {
		for k, sc := range day[d] {
			sb.selectScene(sc, false)
			sb.moveSelected(d, k)
		}
	}
	sb.setDay(0, "2026-12-07", "07:00", "The market square and the timber-yard", false)
	sb.setDay(1, "2026-12-08", "08:00", "General Zhigalov's yard: the puppy at the gate", false)
	w.toggleStripboard()

	// Production > Issue Draft…: White; then Blue's changes, and Blue
	sampleIssue(t, w, "White", sampleWhite)
	replaceText(t, w, " Not even a beggar near them.", "")
	replaceText(t, w, "passes with a basket.", "passes by.")
	replaceText(t, w, "OTCHUMYELOV\nDon't argue!", "OTCHUMYELOV\n(to Hryukin)\nDon't argue! Not another word!")
	replaceText(t, w, "Yeldyrin helps him back into the overcoat.", "Yeldyrin helps him back into the overcoat. "+
		"The crowd presses in to see the finger, the dog, the superintendent's face, which is the real show. "+
		"A boy climbs the timber-yard gate for a better view. A woman with eggs in her apron asks who has been killed. "+
		"Nobody answers her; everyone is waiting for the next turn. "+
		"Hryukin keeps his finger in the air like a candle in church, and the puppy, forgotten for a moment, licks her paw.")
	sampleIssue(t, w, "Blue", sampleBlue)
	if l := pageLockOf(e.Text()); l == nil || !slices.ContainsFunc(l.lock.Pages, func(p lex.LockedPage) bool { return strings.HasSuffix(p.Label, "A") }) {
		t.Fatalf("no A-page in the lock: %v", lockLabelsOf(t, w))
	}

	// File > Save
	w.textEditor.SetCursorOffset(0)
	if err := w.SaveFile(); err != nil {
		t.Fatal(err)
	}
	w.flushBundleWrites()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// sampleIssue issues the draft as name on day, and waits for it.
func sampleIssue(t *testing.T, w *MainWindow, name string, day time.Time) {
	t.Helper()
	done := false
	w.issueDraft(lex.RevisionSet{Name: name, Date: day}, "", false, func(err error) {
		if err != nil {
			t.Error(err)
		}
		done = true
	})
	waitFor(t, name+" issued", func() bool { return done })
}

// readLF reads a file with its line ends as LF (a Windows checkout may
// give CRLF).
func readLF(p string) ([]byte, error) {
	b, err := os.ReadFile(p)
	return []byte(strings.ReplaceAll(string(b), "\r\n", "\n")), err
}

// runeIndex is where words first are in text, in runes.
func runeIndex(t *testing.T, text, words string) int {
	t.Helper()
	i := strings.Index(text, words)
	if i < 0 {
		t.Fatalf("no %q in the script", words)
	}
	return len([]rune(text[:i]))
}

// TestMakeSample writes examples/sample.accolade, with UPDATE_SAMPLE=1.
func TestMakeSample(t *testing.T) {
	if os.Getenv("UPDATE_SAMPLE") == "" {
		t.Skip("UPDATE_SAMPLE=1 writes examples/sample.accolade")
	}
	data := makeSample(t)
	if err := os.WriteFile(filepath.Join("examples", "sample.accolade"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestSample: the sample the app carries, opened as File > Open Sample
// opens it, has every feature it is meant to show.
func TestSample(t *testing.T) {
	checkSample(t, sampleAccolade)
}

// TestSampleFresh: the generator still makes a sample with every
// feature (the formats changed: make it again with UPDATE_SAMPLE=1).
func TestSampleFresh(t *testing.T) {
	checkSample(t, makeSample(t))
}

// sampleWindow is the sample opened in a window of its own, nothing of
// the generator's around: settings, data and kept copies all new.
func sampleWindow(t *testing.T, data []byte) *MainWindow {
	t.Helper()
	holdUI(t)
	w := draftsWindow(t)
	old := sampleAccolade
	sampleAccolade = data
	t.Cleanup(func() { sampleAccolade = old })
	w.openSample()
	if w.bundle == nil || !strings.HasPrefix(filepath.Base(w.currentFile), sampleName) {
		t.Fatalf("the sample is not open: %q", w.currentFile)
	}
	return w
}

// sampleFeatures are what the sample shows, one check each: what
// examples/README.md lists.
var sampleFeatures = []struct {
	name  string
	check func(w *MainWindow) error
}{
	{"title page", func(w *MainWindow) error {
		return want(strings.HasPrefix(w.textEditor.Text(), "Title: A Chameleon\n"), "no title page")
	}},
	{"source kept in the bundle (assets/source.md)", func(w *MainWindow) error {
		r := w.sourceRef()
		return want(r.Origin == sourceBundle && r.Entry == bundleSourceName, "the source is %+v", r)
	}},
	{"the source's front matter", func(w *MainWindow) error {
		_, src := w.bundleSource()
		md := string(src)
		return want(frontMatterValue(md, "title") == "A Chameleon" && frontMatterValue(md, "translator") == "Constance Garnett" &&
			strings.Contains(frontMatterValue(md, "rights"), "Public domain in the USA") && sourceTitle(md, "x.md") == "A Chameleon",
			"front matter: title %q", frontMatterValue(md, "title"))
	}},
	{"Source pane: passages, links found", func(w *MainWindow) error {
		s := sampleSource(w)
		m := s.current()
		return want(len(s.passages) == 31 && m.count(linkFound) == len(m.matches) && len(m.matches) >= 25,
			"%d passages, %d of %d links found", len(s.passages), m.count(linkFound), len(m.matches))
	}},
	{"a link in Text Fragment form", func(w *MainWindow) error {
		return want(slices.ContainsFunc(parseLinks(w.textEditor.Text()), func(l sourceLink) bool { return !l.Ref.Old && l.Ref.Dir.Start != "" }), "none")
	}},
	{"a link with a range (textStart,textEnd)", func(w *MainWindow) error {
		return want(slices.ContainsFunc(parseLinks(w.textEditor.Text()), func(l sourceLink) bool { return l.Ref.Dir.End != "" }), "none")
	}},
	{"a link spanning two passages", func(w *MainWindow) error {
		s := sampleSource(w)
		for _, mt := range s.current().matches {
			if mt.Kind == linkFound && mt.last() > mt.Passages[0] {
				return nil
			}
		}
		return want(false, "none")
	}},
	{"a twin told apart by a prefix", func(w *MainWindow) error {
		return want(slices.ContainsFunc(parseLinks(w.textEditor.Text()), func(l sourceLink) bool {
			return l.Ref.Dir.Prefix != "" && l.Ref.Dir.Start == `"Yes."`
		}), "no prefix- link to the second \"Yes.\"")
	}},
	{"passages cut", func(w *MainWindow) error {
		s := sampleSource(w)
		return want(s.statusOf(16) == statusCut && s.statusOf(17) == statusCut, "passages 16, 17 are %v %v", s.statusOf(16), s.statusOf(17))
	}},
	{"passages still to adapt", func(w *MainWindow) error {
		s := sampleSource(w)
		return want(s.statusOf(29) == statusToDo && s.statusOf(30) == statusToDo, "the last two are %v %v", s.statusOf(29), s.statusOf(30))
	}},
	{"learned: an alias", func(w *MainWindow) error {
		l := sampleLearned(w)
		return want(slices.ContainsFunc(l.Aliases, func(a aliasEntry) bool { return a.Cue == "YELDYRIN" && slices.Contains(a.Names, "the policeman") }), "%+v", l.Aliases)
	}},
	{"learned: not characters", func(w *MainWindow) error {
		return want(slices.Contains(sampleLearned(w).Not, "Petersburg"), "%v", sampleLearned(w).Not)
	}},
	{"learned: a place", func(w *MainWindow) error {
		return want(slices.Contains(sampleLearned(w).Places, placeEntry{"the timber-yard", "EXT. PITCHUGIN'S TIMBER-YARD"}), "%v", sampleLearned(w).Places)
	}},
	{"Characters block: notes and fields", func(w *MainWindow) error {
		l := charListOf(w.textEditor.Text())
		i := l.find("OTCHUMYELOV")
		return want(i >= 0 && l.entries[i].Note != "" && l.entries[i].Fields[fieldAge] == "50s" && l.entries[i].Fields[fieldArc] != "", "%+v", l.entries)
	}},
	{"Characters pane: an alias", func(w *MainWindow) error {
		c, ok := sampleChars(w).charOf("OTCHUMYELOV")
		return want(ok && slices.Contains(c.Aliases, "the superintendent"), "%+v", c)
	}},
	{"Characters: a listed name who doesn't speak", func(w *MainWindow) error {
		c, ok := sampleChars(w).charOf("GENERAL ZHIGALOV")
		return want(ok && c.Entry >= 0 && len(c.Cues) == 0, "%+v", c)
	}},
	{"Notes", func(w *MainWindow) error {
		return want(strings.Contains(notesOf(w.textEditor.Text()), "Public domain in the USA"), "no notes")
	}},
	{"scene numbers locked", func(w *MainWindow) error {
		if !sceneLockOf(w.textEditor.Text()).locked {
			return fmt.Errorf("not locked")
		}
		// the Characters pane says the locked number: HRYUKIN first speaks
		// in scene 2, the third heading counting the A-scene before it
		p := sampleChars(w)
		c, _ := p.charOf("HRYUKIN")
		return want(strings.HasSuffix(p.charStats(c), "from scene 2"), "%q", p.charStats(c))
	}},
	{"an OMITTED scene", func(w *MainWindow) error {
		return want(strings.Contains(w.textEditor.Text(), "\n.OMITTED #3#\n"), "none")
	}},
	{"an A-scene", func(w *MainWindow) error {
		t := w.textEditor.Text()
		return want(strings.Contains(t, "EXT. GENERAL ZHIGALOV'S YARD - DAY #1A#"), "no 1A")
	}},
	{"drafts: a writer's, White and Blue issued", func(w *MainWindow) error {
		ds, err := readDrafts(w.extrasHome())
		if err != nil {
			return err
		}
		var got []string
		for _, d := range ds {
			got = append(got, d.Kind+":"+cmpOr(d.Colour, d.Name))
		}
		return want(slices.Equal(got, []string{"draft:First draft", "issue:White", "issue:Blue"}), "%v", got)
	}},
	{"Blue's revision marks", func(w *MainWindow) error {
		w.updateRevisionMarks()
		return want(slices.Contains(w.revisions.marks, "Blue"), "%v", w.revisions.marks)
	}},
	{"pages locked, with an A-page", func(w *MainWindow) error {
		l := pageLockOf(w.textEditor.Text())
		return want(l != nil && slices.ContainsFunc(l.lock.Pages, func(p lex.LockedPage) bool { return strings.HasSuffix(p.Label, "A") }), "%v", l)
	}},
	{"Page Setup in the file: the format the pages were locked in", func(w *MainWindow) error {
		l := pageLockOf(w.textEditor.Text())
		return want(l != nil && l.format != "" && strings.Contains(l.text(), "format "+l.format), "%+v", l)
	}},
	{"breakdown tags in several categories", func(w *MainWindow) error {
		f := w.breakdownNow()
		cats := map[string]bool{}
		for _, el := range f.Elements {
			cats[el.Category] = true
		}
		return want(len(cats) >= 5, "%v", cats)
	}},
	{"a lost breakdown tag", func(w *MainWindow) error {
		w.updateBreakdown()
		_, _, ps := w.breakdownNowFound()
		if !slices.ContainsFunc(ps, func(p placed) bool { return p.lost && p.element.Name == "basket" }) {
			return fmt.Errorf("none lost")
		}
		// on the script's words, never in a link's quote
		links := parseLinks(w.textEditor.Text())
		for _, p := range ps {
			if !p.lost && slices.ContainsFunc(links, func(l sourceLink) bool { return p.start < l.End && l.Start < p.end }) {
				return fmt.Errorf("%q found in a link", p.element.Name)
			}
		}
		return nil
	}},
	{"a schedule over two days", func(w *MainWindow) error {
		w.measureSchedule()
		f, _, p := w.scheduleFound()
		return want(len(f.Days) == 2 && len(p.days) == 2 && len(p.days[0]) > 0 && len(p.days[1]) > 0, "%d days", len(f.Days))
	}},
	{"Swap with Main: the source in the main editor", func(w *MainWindow) error {
		w.swapSourceAndScript()
		defer w.swapBack()
		return want(w.swapped == "source" && w.mainDoc != nil && strings.Contains(w.mainDoc.Text(), "Otchumyelov"), "swapped %q", w.swapped)
	}},
}

// want is nil if ok, else the error said so.
func want(ok bool, format string, args ...any) error {
	if ok {
		return nil
	}
	return fmt.Errorf(format, args...)
}

func sampleSource(w *MainWindow) *sourcePane {
	w.showPane("source", false)
	s := w.sourcePane()
	s.load()
	return s
}

func sampleLearned(w *MainWindow) learned {
	inside, _ := adaptationBlock(w.textEditor.Text())
	return parseLearned(inside)
}

func sampleChars(w *MainWindow) *charactersPane {
	w.showPane("characters", false)
	p := w.charactersPane()
	p.update(w.textEditor.Text())
	return p
}

// checkSample opens data as the sample and checks each feature.
func checkSample(t *testing.T, data []byte) {
	t.Helper()
	w := sampleWindow(t, data)
	for _, f := range sampleFeatures {
		if err := f.check(w); err != nil {
			t.Errorf("%s: %v", f.name, err)
		}
	}
	// looked at, not changed: Open Sample opens the same copy again
	w.flushBundleWrites()
	if b, err := os.ReadFile(w.currentFile); err != nil || string(b) != string(data) {
		t.Errorf("opening the sample changed its file (%v): Open Sample would make a new copy each time", err)
	}
}

// TestSampleScreenshots renders the sample open, with SHOT_DIR set: the
// script with the Source pane, the Characters pane, the Production pane,
// the stripboard, the source swapped in; desktop and phone.
func TestSampleScreenshots(t *testing.T) {
	if os.Getenv("SHOT_DIR") == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	w := sampleWindow(t, sampleAccolade)
	w.fyneWindow.Resize(fyne.NewSize(1100, 650))
	sampleSource(w)
	comboShot(t, w, "sample-source")
	sampleChars(w)
	comboShot(t, w, "sample-characters")
	p := prodPane(t, w)
	p.pick(stageBreakdown)
	p.refresh()
	comboShot(t, w, "sample-breakdown")
	board(t, w)
	comboShot(t, w, "sample-stripboard")
	w.toggleStripboard()
	w.swapSourceAndScript()
	comboShot(t, w, "sample-swapped")
	w.swapBack()
}

// Open Sample opens the copy left as it was again; a changed one is kept
// and a fresh copy goes beside it.
func TestSampleCopy(t *testing.T) {
	dir := t.TempDir()
	p, err := sampleCopy(dir)
	if err != nil || filepath.Base(p) != sampleName+".accolade" {
		t.Fatalf("%q %v", p, err)
	}
	if again, _ := sampleCopy(dir); again != p {
		t.Errorf("not opened again: %q", again)
	}
	os.WriteFile(p, []byte("changed"), 0o644)
	p2, err := sampleCopy(dir)
	if err != nil || filepath.Base(p2) != sampleName+" 2.accolade" {
		t.Errorf("beside the changed one: %q %v", p2, err)
	}
	if b, _ := os.ReadFile(p); string(b) != "changed" {
		t.Error("the changed copy was written over")
	}
}
