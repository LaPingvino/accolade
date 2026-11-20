package main

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Drafts (LOOP.md, Joop 2026-11-20): earlier versions of the script, and
// of its source, kept with it so one can go back. File > Save Draft…
// keeps the script as it is now under a name, at any time (before a lock
// too); File > Drafts… lists them, compares two (or one with the script
// now: drafts_view.go) and restores one as a new draft, never
// overwriting. Production's issues (White, Blue …, PRODUCTION step 4)
// will be drafts of the kind "issue", in the same store.
//
// The store is among the script's extras (extras.go):
//
//	assets/drafts/drafts.json            the index
//	assets/drafts/01-first-draft/text.fountain
//	assets/drafts/02-for-anna/text.fountain
//	assets/drafts/02-for-anna/source.md  the source, when it changed
//
// The index names each draft's text and source by their files, so a text
// (or a source) the same as one stored already is not stored again: the
// new draft names the one there. A draft whose text and source are both
// stored already has no folder of its own.

const draftsIndexName = draftsDir + "drafts.json"

// draftKind is a draft's kind: one saved while writing, or an issued
// revision (step 4).
const (
	kindDraft = "draft"
	kindIssue = "issue"
)

// savedDraft is one draft in the index.
type savedDraft struct {
	ID   string    `json:"id"` // its folder's name: 02-for-anna
	Name string    `json:"name"`
	Date time.Time `json:"date"`
	Kind string    `json:"kind"`
	// Colour is an issue's revision colour (step 4)
	Colour string `json:"colour,omitempty"`
	// Text is the file holding its text, in the bundle's names; Source
	// its source's ("" none, or kept in the script itself)
	Text   string `json:"text"`
	Source string `json:"source,omitempty"`
}

// draftsIndex is drafts.json.
type draftsIndex struct {
	Version int          `json:"version"`
	Drafts  []savedDraft `json:"drafts"`
}

// draftsIndexIn is the drafts' index if files hold it, else "".
func draftsIndexIn(files []bundleFile) string {
	for _, f := range files {
		if f.name == draftsIndexName {
			return draftsIndexName
		}
	}
	return ""
}

// readDrafts are the drafts in the store, oldest first (none if there is
// no index).
func readDrafts(r extrasReader) ([]savedDraft, error) {
	data, err := r.readExtra(draftsIndexName)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var idx draftsIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("the drafts' index cannot be read: %v", err)
	}
	return idx.Drafts, nil
}

// draftSlug is a draft's name as a folder's: letters and digits in lower
// case, accents taken off, the rest a dash; at most 40 long.
func draftSlug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range norm.NFD.String(name) {
		switch {
		case unicode.Is(unicode.Mn, r):
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if r < 128 {
				b.WriteRune(unicode.ToLower(r))
				dash = false
			} else if !dash && b.Len() > 0 {
				b.WriteByte('-')
				dash = true
			}
		default:
			if !dash && b.Len() > 0 {
				b.WriteByte('-')
				dash = true
			}
		}
		if b.Len() >= 40 {
			break
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return "draft"
	}
	return s
}

// newDraft is what to add to the store for a draft: its entry and the
// files to write (the index among them). text is the script; ext its
// text file's extension (.fountain, .md); source is the source's
// Markdown, nil for none. A text (and a source) the same as one stored is
// not stored again. With skipSame, nothing is added if the text and the
// source are both the same as a stored draft's (no files, ok false).
func newDraft(r extrasReader, name, kind string, text []byte, ext string, source []byte, now time.Time, skipSame bool) (d savedDraft, files []bundleFile, ok bool, err error) {
	ds, err := readDrafts(r)
	if err != nil {
		return savedDraft{}, nil, false, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = fmt.Sprintf("Draft %d", len(ds)+1)
	}
	d = savedDraft{Name: name, Date: now.UTC().Truncate(time.Second), Kind: cmp.Or(kind, kindDraft)}
	taken := map[string]bool{}
	for _, o := range ds {
		taken[o.ID] = true
	}
	base := fmt.Sprintf("%02d-%s", len(ds)+1, draftSlug(name))
	d.ID = base
	for i := 2; taken[d.ID]; i++ {
		d.ID = fmt.Sprintf("%s-%d", base, i)
	}
	// stored already? the latest first
	same := func(ref string, data []byte) bool {
		got, err := r.readExtra(ref)
		return err == nil && bytes.Equal(got, data)
	}
	for i := len(ds) - 1; i >= 0 && d.Text == ""; i-- {
		if ds[i].Text != "" && same(ds[i].Text, text) {
			d.Text = ds[i].Text
		}
	}
	for i := len(ds) - 1; i >= 0 && source != nil && d.Source == ""; i-- {
		if ds[i].Source != "" && same(ds[i].Source, source) {
			d.Source = ds[i].Source
		}
	}
	if skipSame && d.Text != "" && (source == nil || d.Source != "") {
		return savedDraft{}, nil, false, nil
	}
	if d.Text == "" {
		d.Text = draftsDir + d.ID + "/text" + ext
		files = append(files, bundleFile{d.Text, text})
	}
	if source != nil && d.Source == "" {
		d.Source = draftsDir + d.ID + "/source.md"
		files = append(files, bundleFile{d.Source, source})
	}
	idx, err := json.MarshalIndent(draftsIndex{Version: 1, Drafts: append(ds, d)}, "", "  ")
	if err != nil {
		return savedDraft{}, nil, false, err
	}
	files = append(files, bundleFile{draftsIndexName, append(idx, '\n')})
	return d, files, true, nil
}

// draftExt is the extension of a draft's text file.
func draftExt(markdown bool) string {
	if markdown {
		return ".md"
	}
	return ".fountain"
}

// draftLabel is how a list names a draft: its name and date.
func draftLabel(d savedDraft) string {
	return d.Name + " · " + d.Date.Local().Format("2 Jan 2006 15:04")
}

// draftSource is the source to keep with a draft: its Markdown, nil when
// there is none or it is kept in the script (the text holds it then).
func (w *MainWindow) draftSource() []byte {
	r := w.sourceRef()
	if r.Origin == sourceNone || r.Origin == sourceInside {
		return nil
	}
	md, err := w.sourceText(r)
	if err != nil {
		return nil
	}
	return []byte(md)
}

// addDraft keeps the script as it is now as a draft named name (with
// skipSame, not if a draft holds it already); done (may be nil) gets the
// draft and whether one was added, on the UI goroutine. In a bundle it is
// written in the background (bundle_queue.go); a plain script's go to its
// kept copy or its folder (extras.go), a kept copy made if it has none.
func (w *MainWindow) addDraft(name string, skipSame bool, done func(d savedDraft, added bool, err error)) {
	text, ext, src, now := []byte(w.textEditor.Text()), draftExt(w.markdown), w.draftSource(), time.Now()
	finish := func(d savedDraft, ok bool, err error) {
		if done != nil {
			done(d, ok, err)
		}
	}
	if w.bundle != nil {
		var d savedDraft
		var ok bool
		p := w.bundle.path
		warn := w.warnsHighland(p, []bundleFile{{draftsIndexName, nil}})
		w.queueBundle(bundleJob{
			run: func(b *textBundle) (*textBundle, error) {
				if b == nil {
					return nil, errors.New("the script's bundle is gone")
				}
				d0, files, ok0, err := newDraft(b, name, kindDraft, text, ext, src, now, skipSame)
				if err != nil || !ok0 {
					return b, err
				}
				nb, err := saveBundleFiles(b, files, warn)
				if err == nil {
					d, ok = d0, true
				}
				return nb, err
			},
			done: func(err error) {
				if err == nil && ok && warn {
					w.highlandWarning(p)
				}
				finish(d, ok, err)
			},
		})
		return
	}
	home := w.extrasHome()
	if home.kind == extrasNone {
		dir, err := newKeptExtras(cmp.Or(w.currentFile, w.suggestedName, w.untitledName()))
		if err != nil {
			finish(savedDraft{}, false, err)
			return
		}
		w.rememberExtras(w.currentFile, string(dir))
		home = extrasHome{kind: extrasKept, dir: dir}
		if inBrowser && w.currentFile == "" {
			keepDraft(w.draft()) // a reload keeps the drafts with the script
		}
	}
	d, files, ok, err := newDraft(home, name, kindDraft, text, ext, src, now, skipSame)
	if err == nil && ok {
		err = home.dir.write(files)
	}
	finish(d, ok && err == nil, err)
}

// restoreDraft makes a draft's text the script's again, as a new draft:
// the script as it is now is kept as a draft first (unless a draft holds
// it already), the text replaced in one undo step, and kept as "Name
// (restored)". The source stays as it is (a draft's source is there to
// compare).
func (w *MainWindow) restoreDraft(d savedDraft, done func(error)) {
	finish := func(err error) {
		if done != nil {
			done(err)
		}
	}
	data, err := w.extrasHome().readExtra(d.Text)
	if err != nil {
		finish(fmt.Errorf("could not read the draft %s: %v", d.Name, err))
		return
	}
	text, _ := decodeScript(data)
	w.addDraft("Before restoring "+d.Name, true, func(_ savedDraft, _ bool, err error) {
		if err != nil {
			finish(err)
			return
		}
		e := w.textEditor
		e.Replace(0, utf8.RuneCountInString(e.Text()), text)
		w.addDraft(d.Name+" (restored)", false, func(_ savedDraft, _ bool, err error) { finish(err) })
	})
}
