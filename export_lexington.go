package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/fdx"
	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/pdf"
	"github.com/LaPingvino/lexington/pdfin"
	"github.com/LaPingvino/lexington/rules"
)

// PDF and HTML export go through the lexington library, which parses
// the script the same way the lexington command line tool does.

var englishScenes = rules.DefaultConf().Scenes("en")

func parseForExport(content string) lex.Screenplay {
	return fountain.Parse(englishScenes, strings.NewReader(content))
}

// exportJob is how to export: the script format (a Lexington preset's
// key: its rules), the paper ("letter", "a4", "a5"; "" for the format's
// suggestion or Letter), a template for the formats that take one, and
// what to do with the script.
type exportJob struct {
	record        *pdf.Drawing // a PDF's pages as drawn, filled by the export (the page view)
	format        string
	page          string
	template      string
	omitTitlePage bool           // leave the title page out
	numberScenes  bool           // number the scenes that have no number
	scenes        []string       // the script language's scene headings (nil: English)
	language      string         // the script's language: its direction for a line with none of its own ("": English)
	colors        bool           // headings and sections in their notes' colors
	colorStyle    lex.ColorStyle // how they show ("": Lexington's, a line in the margin)
	colorBlock    bool           // a marker, background or text color over the whole block, not the heading
	colorTint     bool           // a line's block lightly tinted
	colorPatterns bool           // a line's bar in its color's pattern (for color blindness)
	pagination    string         // page breaks: Lexington's -pagination ("": the format's)

	// the filters, which compose: the scenes and sections with one of the
	// tags (lex.FilterTags), then a character's sides (lex.Sides, in mode
	// sides, a cue's last cueWords words or all of it with 0)
	tags      []string
	character string
	sides     lex.SidesMode
	cueWords  int

	// revisions: the sets and each line of the content's mark (revisions.go;
	// nil: none), for PDF and FDX
	revisions *lex.Revisions
	marks     []string

	// locked pages (lockedpages.go; nil: none) and the scenes' CONTINUEDs,
	// for PDF (and the lock for FDX)
	lock       *lex.PageLock
	continueds bool
	lockFrom   string // a text whose pages are the lock (the PDF's; withIssuedPages)

	// the breakdown (breakdown_io.go; nil: none) and the tags by line,
	// and the scenes' lengths, for FDX
	breakdown *lex.TagSet
	lineTags  map[int][]lex.LineTag
	// lines: the rows carry their lines (lex.Line.Source): a preview's,
	// for the pages' lines (pagefollow.go)
	lines        bool
	sceneLengths bool

	// mergeOmitted prints a run of OMITTED headings as one, "12-14
	// OMITTED" (the production block's setting, omitted_runs.go): every
	// format but FDX and Fountain, the pages and their lock alike
	mergeOmitted bool

	// conf is the script's Lexington configuration (MainWindow.lex), its
	// formats among the element sets; unset: the machine's files
	conf lexConf
}

// options are Lexington's options for the job, writing as to to output.
func (j exportJob) options(to, output string) cli.Options {
	o := cli.Options{
		ConfigFile:   lexingtonConfig(),
		FormatDir:    formatDirs(),
		Elements:     j.format,
		Page:         j.page,
		To:           to,
		Output:       output,
		TemplatePath: j.template,
		Colors:       j.colors,
		Tags:         j.tags,
		Pagination:   j.pagination,
		Revisions:    j.revisions,
		Record:       j.record,
		Lock:         j.lock,
		Continueds:   j.continueds,
		Breakdown:    j.breakdown,
		LineTags:     j.lineTags,
		SceneLengths: j.sceneLengths,
		MergeOmitted: j.mergeOmitted,
		SceneIn:      j.language, // the HTML's direction for a line with none of its own
	}
	if j.conf.Languages != nil { // the script's own, with what it brought
		conf := j.conf.TOMLConf
		conf.Elements = maps.Clone(conf.Elements)
		if conf.Elements == nil {
			conf.Elements = map[string]rules.Set{}
		}
		for _, p := range j.conf.formats() {
			conf.Elements[p.Key] = p.Elements
		}
		o.Conf, o.FormatDir = &conf, ""
		if o.Page == "" {
			o.Page = j.conf.format(j.format).Page
		}
	}
	if j.colors { // (a style alone would switch colors on)
		o.ColorStyle = string(j.colorStyle)
		// only what goes with the style: Lexington refuses the rest
		switch j.colorStyle {
		case lex.ColorMarker, lex.ColorBackground, lex.ColorText:
			if j.colorBlock {
				o.ColorScope = "block"
			}
		case lex.ColorLine, "":
			o.ColorTint, o.ColorPatterns = j.colorTint, j.colorPatterns
		}
	}
	if j.character != "" {
		o.Character, o.Sides, o.CueWords = j.character, string(j.sides), j.cueWords
	}
	return o
}

// lang is the job's language with all its rules: the script's own (its
// window's configuration), else by its scene words.
func (j exportJob) lang() rules.Language {
	if j.language != "" && j.conf.Knows(j.language) {
		return j.conf.Language(j.language)
	}
	return langFor(j.scenes)
}

// script is the content parsed and prepared for the job.
func (j exportJob) script(content string) lex.Screenplay {
	lang := j.lang()
	var s lex.Screenplay
	if j.revisions != nil || j.lineTags != nil || j.lines {
		s = fountain.ParseLinesIn(lang, strings.NewReader(content))
		for i := range s {
			if n := s[i].Source - 1; n >= 0 && n < len(j.marks) {
				s[i].Revision = j.marks[n]
			}
		}
	} else {
		s = fountain.ParseIn(lang, strings.NewReader(content))
	}
	if j.omitTitlePage {
		s = withoutTitlePage(s)
	}
	// a locked script's numbers are always shown: a scene written last
	// may not have its number yet (production.go)
	if l := sceneLockOf(content); l.locked {
		s = lex.NumberScenes(s, lex.NumberOptions{Locked: true, Style: l.style})
	} else if j.numberScenes {
		s = lex.NumberScenes(s, lex.NumberOptions{})
	}
	return s
}

// withoutTitlePage is the screenplay without its title page (up to and
// including the page break after it).
func withoutTitlePage(s lex.Screenplay) lex.Screenplay {
	if len(s) == 0 || s[0].Type != lex.TypeTitlePage {
		return s
	}
	for i, l := range s {
		if l.Type == lex.TypeNewPage {
			return s[i+1:]
		}
	}
	return nil
}

// exportAs writes the script as Lexington's format to (its -to), with
// Lexington's command line's options: what "accolade -to ..." makes.
func exportAs(content, outputPath, to string, job exportJob) error {
	job = withLockedPages(content, to, job)
	if to == "pdf" {
		// a script the PDF writer cannot print as written (CJK; Arabic,
		// whose letters it does not join): a browser on the computer prints Lexington's
		// HTML (browser_pdf.go)
		if b := localBrowser(); b != "" && pdfWarning(content, job.lang()) != "" {
			dir, err := os.MkdirTemp("", "accolade-pdf-")
			if err != nil {
				return err
			}
			defer os.RemoveAll(dir)
			page := filepath.Join(dir, "script.html")
			if err := exportAs(content, page, "html", job); err != nil {
				return err
			}
			return browserPDF(b, page, outputPath)
		}
	}
	return writeViaTemp(outputPath, func(tmp string) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%s export failed: %v", to, r)
			}
		}()
		return cli.Write(context.Background(), job.script(content), job.options(to, tmp))
	})
}

// lexingtonConfig is Lexington's settings file in Accolade's settings
// folder: its element sets and scene headings, as for the lexington
// command (Lexington's defaults and presets without it).
func lexingtonConfig() string { return filepath.Join(getConfigDir(), "lexington.toml") }

// importFDX converts a Final Draft document to Fountain text.
func importFDX(r io.Reader) (string, error) {
	text, _, _, err := importFDXRevised(r)
	return text, err
}

// importFDXRevised is importFDX with the document's revisions: the sets
// its text uses (oldest first) and the marked lines of the text
// (revisions.go keeps them). Its locked pages go into the text's
// production block (lockedpages.go).
func importFDXRevised(r io.Reader) (text string, sets []lex.RevisionSet, marks []importedMark, err error) {
	text, sets, marks, _, err = importFDXTagged(r)
	return text, sets, marks, err
}

// importFDXTagged is importFDXRevised with the document's breakdown tags
// as Accolade's (breakdown_io.go).
func importFDXTagged(r io.Reader) (text string, sets []lex.RevisionSet, marks []importedMark, tags breakdownFile, err error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", nil, nil, tags, err
	}
	screenplay, sets, err := fdx.ParseRevisions(bytes.NewReader(data))
	if err != nil {
		return "", nil, nil, tags, err
	}
	var buf bytes.Buffer
	if err := (&fountain.FountainWriter{SceneConfig: englishScenes}).Write(&buf, screenplay); err != nil {
		return "", nil, nil, tags, err
	}
	text = buf.String()
	if len(sets) > 0 {
		marks = marksOnText(screenplay, text)
	}
	tags, _ = fdxBreakdown(data, text, englishScenes)
	if lock, err := fdx.ParseLockedPages(bytes.NewReader(data)); err == nil && lock != nil {
		start, end, block := withProductionBlock(text, withPageLock("", &pageLock{lock: lock}))
		runes := []rune(text)
		text = string(runes[:start]) + block + string(runes[end:])
	}
	return text, sets, marks, tags, nil
}

// marksOnText are a screenplay's revised rows as lines of the Fountain
// written from it: each row found in the next lines that hold its text.
func marksOnText(s lex.Screenplay, text string) []importedMark {
	lines := strings.Split(text, "\n")
	var out []importedMark
	at := 0
	for _, row := range s {
		key := strings.ToUpper(strings.TrimSpace(lex.StripNotes(row.Contents)))
		if key == "" || row.Type == lex.TypeBoneyard {
			continue
		}
		for k := at; k < len(lines) && k < at+40; k++ {
			if strings.Contains(strings.ToUpper(lines[k]), key) {
				if row.Revision != "" {
					out = append(out, importedMark{Line: k, Set: row.Revision, Text: strings.TrimSpace(lines[k])})
				}
				at = k + 1
				break
			}
		}
	}
	return out
}

// importPDF reads a screenplay from a PDF as Fountain text. Scanned
// pages are read with OCR in ocrLang (Lexington's code, "" English):
// tesseract if it is installed, else the built-in one (slower, English);
// with ocrText, the scans whose own text layer looks garbled too.
// progress (may be nil) is told the pages read; skipped are the pages that
// could not be read.
func importPDF(ctx context.Context, path string, scenes []string, ocrLang string, ocrText bool, progress func(done, pages int)) (text string, skipped []string, err error) {
	if scenes == nil {
		scenes = englishScenes
	}
	opts := pdfin.Options{Context: ctx, Progress: progress, Scenes: scenes, Skipped: func(page int, err error) {
		skipped = append(skipped, fmt.Sprintf("page %d: %v", page, err))
	}}
	if ocr, done := pdfOCR(ocrLang); ocr != nil {
		defer done()
		opts.OCR, opts.OCRText = ocr, ocrText
	}
	screenplay, err := pdfin.ReadFileWith(path, opts)
	if err != nil {
		return "", skipped, err
	}
	var buf bytes.Buffer
	if err := (&fountain.FountainWriter{SceneConfig: scenes}).Write(&buf, screenplay); err != nil {
		return "", skipped, err
	}
	return buf.String(), skipped, nil
}

// writeViaTemp lets write produce the file under a temporary name next to
// outputPath and moves it into place only if writing succeeded.
func writeViaTemp(outputPath string, write func(tmp string) error) error {
	tmp, err := os.CreateTemp(filepath.Dir(outputPath), "."+filepath.Base(outputPath)+".*.tmp")
	if err != nil {
		return err
	}
	tmp.Close()
	defer os.Remove(tmp.Name()) // no-op once renamed

	if err := write(tmp.Name()); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), outputPath)
}
