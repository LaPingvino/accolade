package main

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2/dialog"
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

// The samples in a language written right to left (Joop, 2026-11-23: to
// try Hebrew's points and Arabic's joining in the editor, the page view
// and the PDF): a short scene each, in the language's own scene words,
// cues and transitions (Lexington's rules for he and ar), with a
// passage fully pointed or vowelled, plain lines, a parenthetical,
// numbers and a name in Latin letters. New starts from the one in the
// language for new scripts; File > Open Hebrew Sample and Open Arabic
// Sample open them any time. The Hebrew passage is Genesis 11:1–4 (the
// tower of Babel), as pointed by Mechon Mamre, without cantillation; the
// Arabic is from Ali Baba and the Forty Thieves.
var (
	//go:embed examples/sample.he.fountain
	sampleHebrew []byte
	//go:embed examples/sample.ar.fountain
	sampleArabic []byte
)

// langSample is a language's sample, its name and its text.
type langSample struct {
	lang, name string
	text       []byte
}

// langSamples are the samples by language.
var langSamples = map[string]langSample{
	"he": {"he", "Hebrew sample", sampleHebrew},
	"ar": {"ar", "Arabic sample", sampleArabic},
}

// languageSample is the text New starts a script in lang with: its
// sample, or "" for none.
func languageSample(lang string) string {
	return string(langSamples[lang].text)
}

// openLanguageSample is File > Open Hebrew (Arabic) Sample: a copy as
// for Open Sample, named for its language (sample.he.fountain), so it
// opens in it whatever the language for new scripts is.
func (w *MainWindow) openLanguageSample(lang string) {
	s := langSamples[lang]
	p, err := copySample(samplesDir(), s.name, "."+s.lang+typeFountain.ext, s.text)
	if err != nil {
		dialog.ShowError(fmt.Errorf("could not open the sample: %v", err), w.fyneWindow)
		return
	}
	w.openPath(p)
}
