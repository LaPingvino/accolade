// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

// Benchmarks of the source pane on a novel (docs/PERF-SOURCE.md): a
// Markdown story of ~100k words in ~2600 passages and a screenplay of
// 200 scenes linking to it (180 links exact, 15 near, 5 missing), both
// made here. Run them with
//
//	go test -run '^$' -bench . -benchmem
//
// The window ones run on Fyne's test driver; what they time is the UI
// goroutine's share of an event (a keystroke, a cursor move, opening the
// pane), the work put off or into the background apart.

import (
	"github.com/LaPingvino/accolade/internal/storyread"

	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/LaPingvino/accolade/internal/prose"
)

var benchVocab = func() []string {
	r := rand.New(rand.NewSource(1))
	syl := []string{"ka", "lo", "mi", "ra", "ten", "sor", "el", "an", "du", "vi", "ne", "sha", "tor", "be", "lin", "ga", "ro", "fe", "is", "mon"}
	common := strings.Fields("the and a to of she he was her his in it that had with for on at said as but not they them from")
	var out []string
	for i := 0; i < 3000; i++ {
		n := 1 + r.Intn(3)
		var b strings.Builder
		for j := 0; j < n; j++ {
			b.WriteString(syl[r.Intn(len(syl))])
		}
		out = append(out, b.String())
	}
	for i := 0; i < 20; i++ {
		out = append(out, common...)
	}
	return out
}()

// novel is ~100k words of Markdown: 30 chapters of paragraphs, with
// dialogue (some repeated openings) and emphasis.
func benchNovel() (string, []string) {
	r := rand.New(rand.NewSource(2))
	var b strings.Builder
	var paras []string
	words := 0
	ch := 0
	for words < 100000 {
		if len(paras)%85 == 0 {
			ch++
			fmt.Fprintf(&b, "## Chapter %d\n\n", ch)
		}
		n := 5 + r.Intn(70)
		var p []string
		for i := 0; i < n; i++ {
			w := benchVocab[r.Intn(len(benchVocab))]
			if i == 0 {
				w = strings.ToUpper(w[:1]) + w[1:]
			}
			if r.Intn(40) == 0 {
				w = "*" + w + "*"
			}
			p = append(p, w)
		}
		s := strings.Join(p, " ") + "."
		if r.Intn(6) == 0 {
			s = `"Yes," she said. ` + s
		}
		paras = append(paras, s)
		b.WriteString(s + "\n\n")
		words += n
	}
	return b.String(), paras
}

// screenplay of 200 scenes, each linking one passage (180 exact, 15
// near: one word changed, 5 missing), ~25k words of action/dialogue.
func benchScreenplay(paras []string) string {
	r := rand.New(rand.NewSource(3))
	var b strings.Builder
	b.WriteString("Title: Bench\n\n")
	step := len(paras) / 200
	for i := 0; i < 200; i++ {
		fmt.Fprintf(&b, "INT. PLACE %d - DAY\n\n", i)
		p := strings.Fields(strings.ReplaceAll(paras[i*step], `"`, ""))
		q := p[:min(8, len(p))]
		switch {
		case i%40 == 7: // missing
			q = []string{"zzz", "qqq", "xxx", "yyy", "www", "vvv", "uuu", "ttt"}
		case i%13 == 5: // near
			q = append([]string{}, q...)
			q[len(q)/2] = "changed"
		}
		fmt.Fprintf(&b, "[[from: \"%s…\"]]\n\n", strings.Join(q, " "))
		for k := 0; k < 4; k++ {
			var a []string
			for j := 0; j < 25; j++ {
				a = append(a, benchVocab[r.Intn(len(benchVocab))])
			}
			fmt.Fprintf(&b, "%s.\n\nANNA\n%s.\n\n", strings.Join(a, " "), strings.Join(a[:8], " "))
		}
	}
	b.WriteString("/* Adaptation\ncut:\n")
	for i := 0; i < 20; i++ {
		p := strings.Fields(strings.ReplaceAll(paras[i*step+1], `"`, ""))
		fmt.Fprintf(&b, "  \"%s\"\n", strings.Join(p[:min(8, len(p))], " "))
	}
	b.WriteString("*/\n")
	return b.String()
}

var benchFixture = sync.OnceValues(func() (string, string) {
	md, paras := benchNovel()
	return md, benchScreenplay(paras)
})

// benchInside is the script with the novel kept in it.
func benchInside() string {
	md, script := benchFixture()
	return script + strings.TrimRight(sourceMarks.text(md), "\n") + "\n"
}

var benchStarts = []string{"INT.", "EXT.", "INT./EXT.", "I/E"}

func BenchmarkPassagesOf(b *testing.B) {
	md, _ := benchFixture()
	for b.Loop() {
		passagesOf(md)
	}
}

// The source model made anew, as on the first follow.
func BenchmarkSourceModelCold(b *testing.B) {
	md, script := benchFixture()
	ps := passagesOf(md)
	for b.Loop() {
		newSourceModel(ps, script, benchStarts)
	}
}

// The model after a keystroke: the passages the same, the links' matches
// kept.
func BenchmarkSourceModelKeystroke(b *testing.B) {
	md, script := benchFixture()
	ps := passagesOf(md)
	lm := newLinkMatcher(ps)
	newSourceModelWith(lm, script, benchStarts)
	edited := []string{script + "x", script + "y"}
	i := 0
	for b.Loop() {
		newSourceModelWith(lm, edited[i%2], benchStarts)
		i++
	}
}

// benchFixtureNew is the bench screenplay with its links and cuts in the
// new form (as Update Source Links writes them; the near and missing
// keep their words).
var benchFixtureNew = sync.OnceValue(func() string {
	md, script := benchFixture()
	edits, _, _ := updateLinksEdits(script, newLinkMatcher(passagesOf(md)))
	r := []rune(script)
	for _, e := range edits {
		r = slices.Concat(r[:e.Start], []rune(e.Text), r[e.End:])
	}
	return string(r)
})

// The model made anew with new-form links: the source's word index too.
func BenchmarkSourceModelColdNew(b *testing.B) {
	md, _ := benchFixture()
	script := benchFixtureNew()
	ps := passagesOf(md)
	for b.Loop() {
		newSourceModel(ps, script, benchStarts)
	}
}

// A keystroke's model with new-form links (BenchmarkSourceModelKeystroke's).
func BenchmarkSourceModelKeystrokeNew(b *testing.B) {
	md, _ := benchFixture()
	script := benchFixtureNew()
	ps := passagesOf(md)
	lm := newLinkMatcher(ps)
	if m := newSourceModelWith(lm, script, benchStarts); m.count(linkFound) < 150 || m.links[0].Ref.Old {
		b.Fatalf("new-form bench script: %d found", m.count(linkFound))
	}
	edited := []string{script + "x", script + "y"}
	i := 0
	for b.Loop() {
		newSourceModelWith(lm, edited[i%2], benchStarts)
		i++
	}
}

// A new-form link being typed: each letter a link not matched yet (the
// cache missed), against the novel: a common word (ambiguous: every
// place it is), a range whose end is nowhere, one found.
func BenchmarkMatchFragmentMiss(b *testing.B) {
	md, _ := benchFixture()
	ps := passagesOf(md)
	lm := newLinkMatcher(ps)
	lm.index()
	for name, d := range map[string]textDirective{
		"common": {Start: "the"},
		"no end": {Start: "the", End: "four words not there"},
		"near":   {Start: "zzz qqq xxx yyy www vvv uuu ttt"},
		"found":  {Start: strings.Join(strings.Fields(ps[500].Plain)[:8], " ")},
	} {
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				lm.findDir(normOf(d))
			}
		})
	}
}

// Adaptation > Update Source Links on the bench screenplay.
func BenchmarkUpdateSourceLinks(b *testing.B) {
	md, script := benchFixture()
	lm := newLinkMatcher(passagesOf(md))
	for b.Loop() {
		updateLinksEdits(script, lm)
	}
}

// A near or missing link, matched against every passage.
func BenchmarkMatchNearLink(b *testing.B) {
	md, script := benchFixture()
	ps := passagesOf(md)
	m := newSourceModel(ps, script, benchStarts)
	var words []string
	for i, l := range m.links {
		if m.matches[i].Kind == linkMissing {
			words = l.Words
		}
	}
	for b.Loop() {
		matchLink(words, ps)
	}
}

// Finding the tail blocks in a new text (the cache does not help: each
// keystroke is a new text), and again in the same one.
func BenchmarkTailBlocks(b *testing.B) {
	_, script := benchFixture()
	for _, c := range []struct{ name, text string }{{"plain", script}, {"inside", benchInside()}} {
		b.Run(c.name+"/new", func(b *testing.B) {
			for b.Loop() {
				findTails(c.text)
			}
		})
		b.Run(c.name+"/same", func(b *testing.B) {
			for b.Loop() {
				sourceBlock(c.text)
			}
		})
	}
}

// The status bar's counts: of a new text, and on a cursor move.
func BenchmarkComputeStats(b *testing.B) {
	_, script := benchFixture()
	for _, c := range []struct{ name, text string }{{"plain", script}, {"inside", benchInside()}} {
		b.Run(c.name+"/new", func(b *testing.B) {
			for b.Loop() {
				countScript(c.text, benchStarts)
			}
		})
		b.Run(c.name+"/cursor", func(b *testing.B) {
			i := 0
			for b.Loop() {
				computeStats(c.text, 1000+i%2, benchStarts)
				i++
			}
		})
	}
}

// Choose Source's conversions, the novel as each kind of file.
func BenchmarkToMarkdown(b *testing.B) {
	md, _ := benchFixture()
	doc := prose.FromMarkdown(md)
	var docx, odt bytes.Buffer
	if err := doc.WriteDOCX(&docx); err != nil {
		b.Fatal(err)
	}
	if err := doc.WriteODT(&odt); err != nil {
		b.Fatal(err)
	}
	var html strings.Builder
	html.WriteString("<html><body>")
	for _, bl := range doc.Blocks {
		tag := "p"
		if bl.Kind == prose.Heading {
			tag = "h2"
		}
		fmt.Fprintf(&html, "<%s>%s</%s>\n", tag, (&prose.Doc{Blocks: []*prose.Block{bl}}).Plain(), tag)
	}
	html.WriteString("</body></html>")
	dir := b.TempDir()
	for _, f := range []struct {
		name string
		data []byte
	}{{"novel.docx", docx.Bytes()}, {"novel.odt", odt.Bytes()}, {"novel.html", []byte(html.String())}} {
		p := filepath.Join(dir, f.name)
		os.WriteFile(p, f.data, 0o644)
		b.Run(filepath.Ext(f.name)[1:], func(b *testing.B) {
			for b.Loop() {
				if _, _, _, err := toMarkdown(p); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// benchWindow is a window with the bench script open and the novel
// beside it, the source pane shown; inside, the novel kept in the script.
func benchWindow(b *testing.B, inside bool) (*MainWindow, *sourcePane, int) {
	md, script := benchFixture()
	return benchWindowWith(b, md, script, inside)
}

// benchWindowWith is benchWindow with this script and source.
func benchWindowWith(b *testing.B, md, script string, inside bool) (*MainWindow, *sourcePane, int) {
	w, _ := newBenchWindow(b)
	dir := b.TempDir()
	p := filepath.Join(dir, "story.fountain")
	os.WriteFile(p, []byte(script), 0o644)
	os.WriteFile(filepath.Join(dir, "story.md"), []byte(md), 0o644)
	if err := w.LoadFile(p); err != nil {
		b.Fatal(err)
	}
	w.togglePane("source")
	if inside {
		if err := w.keepSourceInside(true); err != nil {
			b.Fatal(err)
		}
	}
	s := w.sourcePane()
	s.follow()
	e := w.textEditor
	line := 0
	for i, l := range strings.Split(e.Text(), "\n") {
		if strings.Contains(l, "INT. PLACE 100") {
			line = i
			break
		}
	}
	return w, s, e.Buffer().LineStart(line + 4)
}

// newBenchWindow is newDialogTestWindow for a benchmark.
func newBenchWindow(b *testing.B) (*MainWindow, *Application) {
	b.Setenv("HOME", b.TempDir())
	b.Setenv("XDG_CONFIG_HOME", b.TempDir())
	b.Setenv("XDG_DATA_HOME", b.TempDir())
	a := test.NewApp()
	b.Cleanup(a.Quit)
	app := &Application{fyneApp: a}
	w := NewMainWindow(app)
	app.windows = append(app.windows, w)
	b.Cleanup(w.stopTimers)
	w.fyneWindow.Resize(fyne.NewSize(1100, 650))
	return w, app
}

// realTimers has debounce and background work as the app has them (not
// at once, as the tests do) while a benchmark runs; what is put off is
// cancelled by the benchmark before it can run.
func realTimers(b *testing.B) {
	debounceAtOnce, backgroundAtOnce = false, false
	b.Cleanup(func() { debounceAtOnce, backgroundAtOnce = true, true })
}

// A keystroke in the script with the pane shown: what runs at once (the
// pane follows after a pause: BenchmarkPaneFollowEdit).
func BenchmarkPaneKeystroke(b *testing.B) {
	for _, inside := range []bool{false, true} {
		b.Run(map[bool]string{false: "beside", true: "inside"}[inside], func(b *testing.B) {
			w, _, off := benchWindow(b, inside)
			realTimers(b)
			e := w.textEditor
			i := 0
			for b.Loop() {
				if i%2 == 0 {
					e.Replace(off, off, "x")
				} else {
					e.Replace(off, off+1, "")
				}
				cancel(&w.followTimer)
				i++
			}
		})
	}
}

// The pane following the text after typing paused: the links read again.
func BenchmarkPaneFollowEdit(b *testing.B) {
	for _, inside := range []bool{false, true} {
		b.Run(map[bool]string{false: "beside", true: "inside"}[inside], func(b *testing.B) {
			w, s, off := benchWindow(b, inside)
			realTimers(b)
			e := w.textEditor
			i := 0
			for b.Loop() {
				b.StopTimer()
				if i%2 == 0 {
					e.Replace(off, off, "x")
				} else {
					e.Replace(off, off+1, "")
				}
				cancel(&w.followTimer)
				b.StartTimer()
				s.follow()
				i++
			}
		})
	}
}

// A cursor move with the pane shown, and hidden.
func BenchmarkPaneCursorMove(b *testing.B) {
	for _, inside := range []bool{false, true} {
		for _, hidden := range []bool{false, true} {
			name := map[bool]string{false: "beside", true: "inside"}[inside] + map[bool]string{false: "", true: "/hidden"}[hidden]
			b.Run(name, func(b *testing.B) {
				w, _, off := benchWindow(b, inside)
				if hidden {
					w.togglePane("notes")
				}
				e := w.textEditor
				i := 0
				for b.Loop() {
					e.SetCursorOffset(off + i%2)
					i++
				}
			})
		}
	}
}

// Opening the pane on the novel: all of it (at once, as in the tests),
// and the UI goroutine's share (the passages read in the background, the
// rows estimated, their measuring put off).
func BenchmarkPaneOpen(b *testing.B) {
	b.Run("all", func(b *testing.B) {
		for b.Loop() {
			b.StopTimer()
			w, _ := newBenchWindow(b)
			md, script := benchFixture()
			dir := b.TempDir()
			p := filepath.Join(dir, "story.fountain")
			os.WriteFile(p, []byte(script), 0o644)
			os.WriteFile(filepath.Join(dir, "story.md"), []byte(md), 0o644)
			w.LoadFile(p)
			b.StartTimer()
			w.togglePane("source")
		}
	})
	b.Run("ui", func(b *testing.B) {
		_, s, _ := benchWindow(b, false)
		md, _ := benchFixture()
		ps := passagesOf(md)
		realTimers(b)
		for b.Loop() {
			r := &sourceRead{ref: s.ref, modTime: s.modTime, size: s.size, src: s.src}
			s.use(r, ps)
			cancel(&s.measureT)
			s.follow()
		}
	})
}

// A width change (a split drag's step): the rows estimated again at
// once; measuring them all, which goes on a stretch at a time after.
func BenchmarkPaneWidth(b *testing.B) {
	b.Run("ui", func(b *testing.B) {
		_, s, _ := benchWindow(b, false)
		realTimers(b)
		i := 0
		for b.Loop() {
			s.measure(300 + float32(i%2)*10)
			cancel(&s.measureT)
			i++
		}
	})
	b.Run("all", func(b *testing.B) {
		_, s, _ := benchWindow(b, false)
		i := 0
		for b.Loop() {
			s.measure(300 + float32(i%2)*10)
			i++
		}
	})
}

// Edit: the novel opened as text in the pane, and a keystroke in it.
func BenchmarkSourceEditor(b *testing.B) {
	md, _ := benchFixture()
	b.Run("open", func(b *testing.B) {
		w, _ := newBenchWindow(b)
		se := newDoc(w, docSource) // the script has none: nothing is written
		w.fyneWindow.SetContent(se.view)
		w.fyneWindow.Resize(fyne.NewSize(320, 600))
		for b.Loop() {
			se.SetText(md)
		}
	})
	b.Run("key", func(b *testing.B) {
		w, _ := newBenchWindow(b)
		se := newDoc(w, docSource) // the script has none: nothing is written
		w.fyneWindow.SetContent(se.view)
		w.fyneWindow.Resize(fyne.NewSize(320, 600))
		se.SetText(md)
		w.fyneWindow.Canvas().Focus(se.ed)
		se.ed.SetCursorOffset(5000)
		for b.Loop() {
			se.ed.TypedRune('x')
		}
	})
}

// A keystroke in the source as a middle tab (middle.go):
// kept beside the script (only its own editor), and kept in the script,
// where each key is written into the script's block (document.go).
func BenchmarkSwappedSourceKey(b *testing.B) {
	for _, inside := range []bool{false, true} {
		name := "beside"
		if inside {
			name = "inside"
		}
		b.Run(name, func(b *testing.B) {
			w, _, _ := benchWindow(b, inside)
			debounceAtOnce, backgroundAtOnce = false, false // as when typing: the follow waits for a pause
			b.Cleanup(func() { debounceAtOnce, backgroundAtOnce = true, true })
			w.openMiddle("source")
			d := w.midDoc()
			w.fyneWindow.Canvas().Focus(d.ed)
			d.ed.SetCursorOffset(5000)
			for b.Loop() {
				d.ed.TypedRune('x')
			}
		})
	}
	// the notes in the middle, the novel in them (a long notes block)
	b.Run("notes", func(b *testing.B) {
		w, _, _ := benchWindow(b, true)
		md, _ := benchFixture()
		w.writeNotes(md)
		debounceAtOnce, backgroundAtOnce = false, false
		b.Cleanup(func() { debounceAtOnce, backgroundAtOnce = true, true })
		w.showPane("notes", false)
		w.openMiddle("notes")
		d := w.midDoc()
		w.fyneWindow.Canvas().Focus(d.ed)
		d.ed.SetCursorOffset(5000)
		for b.Loop() {
			d.ed.TypedRune('x')
		}
	})
	// against: a keystroke in the script itself, the source kept in it
	b.Run("script-with-source-inside", func(b *testing.B) {
		w, _, _ := benchWindow(b, true)
		debounceAtOnce, backgroundAtOnce = false, false
		b.Cleanup(func() { debounceAtOnce, backgroundAtOnce = true, true })
		w.fyneWindow.Canvas().Focus(w.textEditor)
		w.textEditor.SetCursorOffset(500)
		for b.Loop() {
			w.textEditor.TypedRune('x')
		}
	})
}

// The links guessed for the bench screenplay without its links (a
// script written before it had a source): the story read, then the
// scenes aligned to it, as the pane does it in the background.
func BenchmarkGuessScenes(b *testing.B) {
	md, script := benchFixture()
	var kept []string
	for _, l := range strings.Split(script, "\n") {
		if !strings.HasPrefix(l, "[[from:") {
			kept = append(kept, l)
		}
	}
	script = strings.Join(kept, "\n")
	ps := passagesOf(md)
	m := newSourceModel(ps, script, benchStarts)
	b.Run("read", func(b *testing.B) {
		for b.Loop() {
			storyread.Read(storyPassages(ps), "en", storyread.Hints{})
		}
	})
	b.Run("align", func(b *testing.B) {
		for b.Loop() {
			guessScenes(script, benchStarts, m)
		}
	})
}
