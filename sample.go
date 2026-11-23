package main

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"github.com/LaPingvino/lexington/fountain"
)

// File > Open Sample (LOOP.md, Joop 2026-11-20): a script to try
// everything on, Chekhov's "A Chameleon" (Constance Garnett's
// translation, public domain in the USA) partly adapted. It is made, not
// written by hand: sample_test.go's TestMakeSample builds
// examples/sample.accolade through Accolade's own commands
// (UPDATE_SAMPLE=1), and TestSample opens it as File > Open Sample does
// and checks that every feature it is meant to show is in it.
// examples/README.md says what it shows and what not.

//go:embed examples/sample.accolade
var sampleAccolade []byte

// sampleName is the sample's file name, without the extension.
const sampleName = "A Chameleon (sample)"

// samplesDir is where the sample is opened from: a real file, so its
// drafts, breakdown and schedule are written back into it as in any
// .accolade. The app's data folder on the desktop; in the browser the
// folder an opened file goes to (browser_js.go's pickFile).
func samplesDir() string {
	if inBrowser {
		return "/files"
	}
	return filepath.Join(filepath.Dir(keptSourcesDir()), "samples")
}

// sampleCopy is a copy of the sample in dir as it came with the app:
// one left as it was is opened again; one changed since is kept, and a
// fresh copy goes beside it ("A Chameleon (sample) 2.accolade").
func sampleCopy(dir string) (string, error) {
	return copySample(dir, sampleName, typeAccolade.ext, sampleAccolade)
}

// copySample is sampleCopy for any sample: data as name+ext in dir.
func copySample(dir, sample, ext string, data []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	for n := 1; n < 1000; n++ {
		name := sample + ext
		if n > 1 {
			name = fmt.Sprintf("%s %d%s", sample, n, ext)
		}
		p := filepath.Join(dir, name)
		old, err := os.ReadFile(p)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return p, os.WriteFile(p, data, 0o644)
		case err == nil && bytes.Equal(old, data):
			return p, nil
		}
	}
	return "", fmt.Errorf("no free name for the sample in %s", dir)
}

// openSample is File > Open Sample: as Open opens a file (in this
// window if it holds nothing of its own, else in another).
func (w *MainWindow) openSample() {
	p, err := sampleCopy(samplesDir())
	if err != nil {
		dialog.ShowError(fmt.Errorf("could not open the sample: %v", err), w.fyneWindow)
		return
	}
	w.openPath(p)
}

// The samples in other languages (Joop, 2026-11-23): a short scene
// each, written in the language, in its own scene words, cues,
// extensions, transitions and title page keys (Lexington's rules for
// it), a story or setting of its culture: examples/sample.<lang>.fountain
// (examples/README.md says which story each is, and that each is
// machine-written, to be checked by a native reader). New starts from
// the one in the language for new scripts; File > Open Sample lists them
// all. The Hebrew one has a passage fully pointed (Genesis 11:1–4, the
// tower of Babel, as pointed by Mechon Mamre, without cantillation), the
// Arabic one vowelled (Ali Baba and the Forty Thieves), to try Hebrew's
// points and Arabic's joining in the editor, the page view and the PDF.
// Hindi has three (the monkey and the crocodile): hi English Fountain's
// words with the Hindi in Latin letters, hi-Latn and hi-Deva each its own.
//
//go:embed examples/sample.*.fountain
var langSampleFiles embed.FS

// langSample is a language's sample: its language, its title (the title
// page's) and its text.
type langSample struct {
	lang, title string
	text        []byte
}

// langSamples are the samples, by language, read once.
var langSamples = sync.OnceValue(func() map[string]langSample {
	out := map[string]langSample{}
	files, _ := fs.Glob(langSampleFiles, "examples/sample.*.fountain")
	for _, f := range files {
		lang := strings.TrimSuffix(strings.TrimPrefix(path.Base(f), "sample."), typeFountain.ext)
		text, err := langSampleFiles.ReadFile(f)
		if err != nil {
			continue
		}
		title := lang
		for _, r := range fountain.ParseIn(machineLex().Language(lang), bytes.NewReader(text)) {
			if r.Type == "Title" {
				title = strings.TrimSpace(r.Contents)
				break
			}
		}
		out[lang] = langSample{lang, title, text}
	}
	return out
})

// sampleLanguages are the languages with a sample, by their names.
func sampleLanguages() []string {
	langs := slices.Collect(maps.Keys(langSamples()))
	slices.SortFunc(langs, func(a, b string) int { return strings.Compare(languageName(a), languageName(b)) })
	return langs
}

// languageSample is the text New starts a script in lang with: its
// sample, or "" for none.
func languageSample(lang string) string {
	return string(langSamples()[lang].text)
}

// sampleMenu is File > Open Sample: the sample to try everything on,
// then each language's ("Esperanto: La unua kongreso").
func (w *MainWindow) sampleMenu() *fyne.Menu {
	items := []*fyne.MenuItem{fyne.NewMenuItem("A Chameleon (English)", w.openSample), fyne.NewMenuItemSeparator()}
	for _, lang := range sampleLanguages() {
		items = append(items, fyne.NewMenuItem(languageName(lang)+": "+langSamples()[lang].title, func() { w.openLanguageSample(lang) }))
	}
	return fyne.NewMenu("Open Sample", items...)
}

// openLanguageSample opens lang's sample: a copy as for Open Sample,
// named for its title and language ("La unua kongreso (sample).eo.fountain"),
// so it opens in it whatever the language for new scripts is.
func (w *MainWindow) openLanguageSample(lang string) {
	s := langSamples()[lang]
	name := strings.NewReplacer("/", "-", "\\", "-", ":", "-").Replace(s.title) + " (sample)"
	p, err := copySample(samplesDir(), name, "."+s.lang+typeFountain.ext, s.text)
	if err != nil {
		dialog.ShowError(fmt.Errorf("could not open the sample: %v", err), w.fyneWindow)
		return
	}
	w.openPath(p)
}
