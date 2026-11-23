package main

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Drafts (LOOP.md, Joop 2026-11-20): earlier versions of the script, and
// of its source, kept with it so one can go back. Collaboration > Save Draft…
// keeps the script as it is now under a name, at any time (before a lock
// too); Collaboration > Drafts… lists them, compares two (or one with the script
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
	// Colour is an issue's revision set ("Blue": lex.RevisionName), and
	// Issued the date it was issued with ("2026-11-20"), which may be
	// another than the day it was kept (revisions.go)
	Colour string `json:"colour,omitempty"`
	Issued string `json:"issued,omitempty"`
	// Text is the file holding its text, in the bundle's names; Source
	// its source's ("" none, or kept in the script itself)
	Text   string `json:"text"`
	Source string `json:"source,omitempty"`
	// Made is what an issue changed in the production block, so that
	// Withdraw Last Issue… takes back what it made and nothing else
	// (issue_undo.go); nil for a draft, an import and an issue kept
	// before Accolade noted it
	Made *issueMade `json:"made,omitempty"`
	// MergedFrom is the outside copy a merge took changes from (its
	// file's name), and Base the draft it was compared against
	// (merge.go); "" for any other draft
	MergedFrom string `json:"mergedFrom,omitempty"`
	Base       string `json:"base,omitempty"`
}

// issueMade is what an issue changed in the production block besides
// its revision line.
type issueMade struct {
	// Revision is the revision line's value before ("" none)
	Revision string `json:"revision"`
	// SceneLock: the issue locked the scene numbers (the first does, if
	// they were not)
	SceneLock bool `json:"sceneLock,omitempty"`
	// PageLock: the issue wrote the page lock (locked the pages first or
	// again); PagesBefore are the lock's lines before it ("" none)
	PageLock    bool   `json:"pageLock,omitempty"`
	PagesBefore string `json:"pagesBefore,omitempty"`
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
	return newDraftAs(r, savedDraft{Name: name, Kind: kind}, text, ext, source, now, skipSame)
}

// newDraftAs is newDraft for a draft as like is (its name, kind, an
// issue's colour and date).
func newDraftAs(r extrasReader, like savedDraft, text []byte, ext string, source []byte, now time.Time, skipSame bool) (d savedDraft, files []bundleFile, ok bool, err error) {
	name, kind := like.Name, like.Kind
	ds, err := readDrafts(r)
	if err != nil {
		return savedDraft{}, nil, false, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = fmt.Sprintf("Draft %d", len(ds)+1)
	}
	d = savedDraft{Name: name, Date: now.UTC().Truncate(time.Second), Kind: cmp.Or(kind, kindDraft),
		Colour: like.Colour, Issued: like.Issued, Made: like.Made, MergedFrom: like.MergedFrom, Base: like.Base}
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
	w.addDraftAs(name, kindDraft, "", time.Time{}, skipSame, done)
}

// addDraftAs is addDraft for a draft of a kind: an issue with its set
// (colour) and the date it is issued with (zero for a draft).
func (w *MainWindow) addDraftAs(name, kind, colour string, issued time.Time, skipSame bool, done func(d savedDraft, added bool, err error)) {
	like := savedDraft{Name: name, Kind: kind, Colour: colour}
	if !issued.IsZero() {
		like.Issued = issued.Format("2006-01-02")
	}
	w.addDraftLike(like, skipSame, done)
}

// addDraftLike is addDraft for a draft as like is (newDraftAs).
func (w *MainWindow) addDraftLike(like savedDraft, skipSame bool, done func(d savedDraft, added bool, err error)) {
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
				d0, files, ok0, err := newDraftAs(b, like, text, ext, src, now, skipSame)
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
	home, err := w.ensureExtrasHome()
	if err != nil {
		finish(savedDraft{}, false, err)
		return
	}
	d, files, ok, err := newDraftAs(home, like, text, ext, src, now, skipSame)
	if err == nil && ok {
		err = home.dir.write(files)
	}
	finish(d, ok && err == nil, err)
}

// ensureExtrasHome is a plain script's extras' home, a kept copy made
// for it if it has none yet.
func (w *MainWindow) ensureExtrasHome() (extrasHome, error) {
	home := w.extrasHome()
	if home.kind != extrasNone {
		return home, nil
	}
	dir, err := newKeptExtras(cmp.Or(w.currentFile, w.suggestedName, w.untitledName()))
	if err != nil {
		return home, err
	}
	w.rememberExtras(w.currentFile, string(dir))
	if inBrowser && w.currentFile == "" {
		w.keepDraft() // a reload keeps the extras with the script
	}
	return extrasHome{kind: extrasKept, dir: dir}, nil
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

// withdrawnDraft is a draft taken out of the store (an issue withdrawn
// or undone: issue_undo.go), its files as they were, so that it can be
// put back as it was: kept for the session only.
type withdrawnDraft struct {
	d     savedDraft
	files []bundleFile // its text and source, read before they went
}

// withoutDraft is what to write to take the draft id out of the store:
// the index without it (files) and the files only it named (gone; a
// text or source another draft names stays), and the draft as it was.
func withoutDraft(r extrasReader, id string) (files []bundleFile, gone []string, kept withdrawnDraft, err error) {
	ds, err := readDrafts(r)
	if err != nil {
		return nil, nil, kept, err
	}
	at := slices.IndexFunc(ds, func(d savedDraft) bool { return d.ID == id })
	if at < 0 {
		return nil, nil, kept, fmt.Errorf("the drafts store has no %s", id)
	}
	kept.d = ds[at]
	if kept.d.Kind == kindIssue && slices.ContainsFunc(ds[at+1:], func(d savedDraft) bool { return d.Kind == kindIssue }) {
		return nil, nil, kept, errLaterIssue
	}
	rest := slices.Delete(slices.Clone(ds), at, at+1)
	named := map[string]bool{}
	for _, d := range rest {
		named[d.Text], named[d.Source] = true, true
	}
	for _, f := range []string{kept.d.Text, kept.d.Source} {
		if f == "" {
			continue
		}
		data, err := r.readExtra(f)
		if err != nil {
			return nil, nil, kept, fmt.Errorf("could not read %s: %v", f, err)
		}
		kept.files = append(kept.files, bundleFile{f, data})
		if !named[f] {
			gone = append(gone, f)
		}
	}
	// its own folder too, once nothing in it is left
	own := draftsDir + kept.d.ID + "/"
	if slices.ContainsFunc(gone, func(f string) bool { return strings.HasPrefix(f, own) }) &&
		!slices.ContainsFunc(rest, func(d savedDraft) bool {
			return strings.HasPrefix(d.Text, own) || strings.HasPrefix(d.Source, own)
		}) {
		gone = append(gone, own)
	}
	idx, err := json.MarshalIndent(draftsIndex{Version: 1, Drafts: rest}, "", "  ")
	if err != nil {
		return nil, nil, kept, err
	}
	return []bundleFile{{draftsIndexName, append(idx, '\n')}}, gone, kept, nil
}

// draftBack is what to write to put a withdrawn draft back, last in the
// store: its files where they were if those places are free or hold the
// same, else under a folder of its own (its id taken since: a new id),
// and the index. d is the draft as it is now in the store.
func draftBack(r extrasReader, k withdrawnDraft) (d savedDraft, files []bundleFile, err error) {
	ds, err := readDrafts(r)
	if err != nil {
		return d, nil, err
	}
	d = k.d
	taken := map[string]bool{}
	for _, o := range ds {
		taken[o.ID] = true
	}
	if taken[d.ID] {
		base := fmt.Sprintf("%02d-%s", len(ds)+1, draftSlug(d.Name))
		d.ID = base
		for i := 2; taken[d.ID]; i++ {
			d.ID = fmt.Sprintf("%s-%d", base, i)
		}
	}
	for _, f := range k.files {
		got, rerr := r.readExtra(f.name)
		switch {
		case rerr == nil && bytes.Equal(got, f.data):
			continue // there as it was (another draft names it, or left)
		case errors.Is(rerr, fs.ErrNotExist) && d.ID == k.d.ID:
			files = append(files, f)
			continue
		}
		moved := draftsDir + d.ID + "/" + path.Base(f.name)
		if f.name == d.Text {
			d.Text = moved
		}
		if f.name == d.Source {
			d.Source = moved
		}
		files = append(files, bundleFile{moved, f.data})
	}
	idx, err := json.MarshalIndent(draftsIndex{Version: 1, Drafts: append(ds, d)}, "", "  ")
	if err != nil {
		return d, nil, err
	}
	return d, append(files, bundleFile{draftsIndexName, append(idx, '\n')}), nil
}
