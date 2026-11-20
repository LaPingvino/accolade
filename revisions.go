package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io/fs"
	"strings"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2/dialog"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/scriptdiff"
	"github.com/LaPingvino/lexington/lex"
)

// Production, step 4 (docs/PRODUCTION.md §10): issues and revision sets.
//
// Production > Issue Draft… keeps the script as it is now in the drafts
// store (drafts.go) as a draft of the kind "issue", in the next colour of
// the WGA's order (lex.RevisionName: White, Blue, Pink … Cherry, Second
// Blue …), with its date; the first, White, is the lock: it locks the
// scene numbers first. The production block records it ("revision: Blue,
// 2026-11-20").
//
// A revision is never marked in the text. A line's colour is found by
// diffing the issued drafts in turn (scriptdiff.Lines, line by line, the
// blank lines left out, the blocks at the script's end too): a line the
// same as in the issue before keeps the colour it had there, a new or
// changed one gets the issue's colour, and a deletion marks the line
// after it (so its page counts as changed). The script as it is now,
// when it differs from the last issue, is the open set: its changes are
// in the next colour (dated today). The marks are those of the current
// revision only (the open set, or the last issue against the one before
// it), or of all issues, each in its own colour (Preferences > Script >
// Production, "revision-marks").
//
// The editor shows them as a bar in the margin (the fold marks' column);
// the PDF as Lexington prints them (an asterisk at the right margin, the
// changed pages' header, revised pages only, coloured paper); FDX as its
// <Revisions> and RevisionIDs.
//
// An FDX read with revisions (Final Draft's marks) has no earlier drafts
// to diff: its text is kept as an issue of its latest set, and its marks,
// by line of that issue, in assets/production/revisions.json. That issue
// starts the chain with those marks; from the next Issue on, the
// snapshots take over (the imported marks still show, in all issues'
// marks, on the lines no later issue changed).
//
// The store is the record: Undo after Issue takes the production block's
// line away, not the issue (Drafts… still lists it).

// revisionsName is the imported revisions' file among the extras.
const revisionsName = productionDir + "revisions.json"

// revisionLineKey is the production block's line of the current revision.
const revisionLineKey = "revision"

// revisionMarksPref is the setting: "last" (the current revision's marks)
// or "all" (each issue's, in its colour).
const revisionMarksPref = "revision-marks"

// revisionHeaderPref and revisionDatePref are a changed page's header
// (lex.Revisions.Header: {set}, {date}) and its date's layout.
const (
	revisionHeaderPref = "revision-header"
	revisionDatePref   = "revision-date"
)

// revisionMarkChoices are the setting's choices, "last" and "all".
var revisionMarkChoices = []string{"The last issue (the current revision)", "All issues, each in its colour"}

// revisionDates are the date layouts offered, by an example of each.
var revisionDates = []struct{ label, layout string }{
	{"20/11/26", "02/01/06"},
	{"11/20/26", "01/02/06"},
	{"2026-11-20", "2006-01-02"},
	{"20.11.2026", "02.01.2006"},
}

// importedRevisions is revisions.json: the marks an FDX came with.
type importedRevisions struct {
	Version  int              `json:"version"`
	Imported *importedMarkSet `json:"imported,omitempty"`
}

// importedMarkSet are the marks of an FDX, on the issue its text was
// kept as.
type importedMarkSet struct {
	From  string         `json:"from"`  // the file's name
	Draft string         `json:"draft"` // the issue's id in drafts.json
	Sets  []importedSet  `json:"sets"`  // the sets its text uses, oldest first
	Marks []importedMark `json:"marks"` // by line of the issue's text
}

type importedSet struct {
	Name string    `json:"name"`
	Date time.Time `json:"date,omitzero"`
}

type importedMark struct {
	Line int    `json:"line"` // from 0
	Set  string `json:"set"`
	Text string `json:"text"` // the line's text, to check
}

// issue is an issued draft as the marks need it: its set, its body's
// lines and the marks it came with (imported; nil none).
type issue struct {
	set      lex.RevisionSet
	lines    []string
	imported []string
	sets     []lex.RevisionSet // an imported issue's earlier sets
}

// revisionChain are the script's issues, oldest first.
type revisionChain struct {
	issues []issue
	// lastMarks are the last issue's marks over all issues (computed once)
	lastMarks []string
}

// bodyLines are the lines of a script's body: up to the blocks at its end.
func bodyLines(text string) []string {
	return strings.Split(text[:bodyEndByte(text)], "\n")
}

// carry are the marks of the new lines: a line the same as an old one
// has the old one's mark, a new or changed line set, and the line after
// a deletion (the last line for one at the end) set too. Blank lines are
// left out of the comparison and never marked.
func carry(old, oldMarks, new []string, set string) []string {
	var ok, nk []string
	var oi, ni []int
	for i, l := range old {
		if t := strings.TrimSpace(l); t != "" {
			ok, oi = append(ok, t), append(oi, i)
		}
	}
	for i, l := range new {
		if t := strings.TrimSpace(l); t != "" {
			nk, ni = append(nk, t), append(ni, i)
		}
	}
	marks := make([]string, len(new))
	i, j, deleted := 0, 0, false
	for _, op := range scriptdiff.Lines(ok, nk) {
		switch op {
		case scriptdiff.Same:
			if oi[i] < len(oldMarks) {
				marks[ni[j]] = oldMarks[oi[i]]
			}
			if deleted {
				marks[ni[j]] = set
			}
			i, j, deleted = i+1, j+1, false
		case scriptdiff.Removed:
			i, deleted = i+1, true
		case scriptdiff.Added:
			marks[ni[j]] = set
			j, deleted = j+1, false
		}
	}
	if deleted && len(ni) > 0 {
		marks[ni[len(ni)-1]] = set
	}
	return marks
}

// nextSet is the set after the last issue (White if none): the next in
// the order after the last's colour.
func (c *revisionChain) nextSet(now time.Time) lex.RevisionSet {
	if c == nil || len(c.issues) == 0 {
		return lex.RevisionSet{Name: lex.RevisionName(0), Date: now}
	}
	last := c.issues[len(c.issues)-1].set.Name
	n := lex.RevisionIndex(last)
	if n < 0 {
		n = lex.RevisionIndex(lex.RevisionColor(last))
	}
	if n < 0 {
		n = len(c.issues) - 1
	}
	return lex.RevisionSet{Name: lex.RevisionName(n + 1), Date: now}
}

// allMarks are the last issue's marks over all issues.
func (c *revisionChain) allMarks() []string {
	if c.lastMarks != nil || len(c.issues) == 0 {
		return c.lastMarks
	}
	m := c.issues[0].imported
	if m == nil {
		m = make([]string, len(c.issues[0].lines))
	}
	for k := 1; k < len(c.issues); k++ {
		m = carry(c.issues[k-1].lines, m, c.issues[k].lines, c.issues[k].set.Name)
	}
	c.lastMarks = m
	return m
}

// revisionMarks are the marks of the script's text, one per line ("" none)
// with the sets they are of (oldest first, the open one last if the text
// differs from the last issue), and the current revision's set.
func (c *revisionChain) marksOf(text string, all bool, now time.Time) (marks []string, sets []lex.RevisionSet, current lex.RevisionSet) {
	n := strings.Count(text, "\n") + 1
	marks = make([]string, n)
	if c == nil || len(c.issues) == 0 {
		return marks, nil, lex.RevisionSet{}
	}
	for _, is := range c.issues {
		sets = append(sets, is.sets...)
		sets = append(sets, is.set)
	}
	body := bodyLines(text)
	last := c.issues[len(c.issues)-1]
	var m []string
	switch {
	case !sameLines(body, last.lines): // the open set
		open := c.nextSet(now)
		sets = append(sets, open)
		current = open
		base := []string(nil)
		if all {
			base = c.allMarks()
		}
		m = carry(last.lines, base, body, open.Name)
	case all:
		current = last.set
		m = c.allMarks()
	case len(c.issues) >= 2:
		current = last.set
		m = carry(c.issues[len(c.issues)-2].lines, nil, last.lines, last.set.Name)
	default:
		current = last.set
		m = last.imported
	}
	copy(marks, m)
	return marks, sets, current
}

// sameLines tells whether two bodies are the same but for blank lines
// and the spaces at the ends of lines.
func sameLines(a, b []string) bool {
	i, j := 0, 0
	for {
		for i < len(a) && strings.TrimSpace(a[i]) == "" {
			i++
		}
		for j < len(b) && strings.TrimSpace(b[j]) == "" {
			j++
		}
		if i == len(a) || j == len(b) {
			return i == len(a) && j == len(b)
		}
		if strings.TrimSpace(a[i]) != strings.TrimSpace(b[j]) {
			return false
		}
		i, j = i+1, j+1
	}
}

// issueDate is an issue's date: the one it was issued with, else when it
// was kept.
func issueDate(d savedDraft) time.Time {
	if t, err := time.ParseInLocation("2006-01-02", d.Issued, time.Local); err == nil {
		return t
	}
	return d.Date.Local()
}

// readRevisionChain reads the issues from the extras.
func readRevisionChain(r extrasReader) (*revisionChain, error) {
	ds, err := readDrafts(r)
	if err != nil {
		return nil, err
	}
	var imp importedRevisions
	if data, err := r.readExtra(revisionsName); err == nil {
		if err := json.Unmarshal(data, &imp); err != nil {
			return nil, fmt.Errorf("the revisions' file cannot be read: %v", err)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	c := &revisionChain{}
	for _, d := range ds {
		if d.Kind != kindIssue {
			continue
		}
		data, err := r.readExtra(d.Text)
		if err != nil {
			return nil, fmt.Errorf("could not read the issue %s: %v", d.Name, err)
		}
		text, _ := decodeScript(data)
		is := issue{set: lex.RevisionSet{Name: d.Colour, Date: issueDate(d)}, lines: bodyLines(text)}
		if is.set.Name == "" {
			is.set.Name = lex.RevisionName(len(c.issues))
		}
		if imp.Imported != nil && imp.Imported.Draft == d.ID && len(c.issues) == 0 {
			is.imported = make([]string, len(is.lines))
			for _, m := range imp.Imported.Marks {
				if m.Line >= 0 && m.Line < len(is.lines) && strings.TrimSpace(is.lines[m.Line]) == strings.TrimSpace(m.Text) {
					is.imported[m.Line] = m.Set
				}
			}
			for _, s := range imp.Imported.Sets {
				if s.Name != is.set.Name {
					is.sets = append(is.sets, lex.RevisionSet{Name: s.Name, Date: s.Date})
				}
			}
		}
		c.issues = append(c.issues, is)
	}
	c.allMarks() // now, not in the background runs side by side
	return c, nil
}

// withBlockLine is the production block's inside with the line of key set
// to "key: value" (in its place, else at the end), the others kept.
func withBlockLine(inside, key, value string) string {
	var lines []string
	set := false
	line := key + ": " + value
	if inside != "" {
		for _, l := range strings.Split(inside, "\n") {
			if k, _, ok := strings.Cut(l, ":"); ok && strings.EqualFold(strings.TrimSpace(k), key) {
				if !set {
					lines = append(lines, line)
					set = true
				}
				continue
			}
			lines = append(lines, l)
		}
	}
	if !set {
		lines = append(lines, line)
	}
	return strings.Trim(strings.Join(lines, "\n"), "\n")
}

// revisionLine is the production block's revision line for a set.
func revisionLine(s lex.RevisionSet, note string) string {
	v := s.Name
	if !s.Date.IsZero() {
		v += ", " + s.Date.Format("2006-01-02")
	}
	if note != "" {
		v += " (" + note + ")"
	}
	return v
}

// currentRevisionOf is the production block's revision line's value ("").
func currentRevisionOf(text string) string {
	inside, ok := productionBlock(text)
	if !ok {
		return ""
	}
	for _, l := range strings.Split(inside, "\n") {
		if k, v, ok := strings.Cut(l, ":"); ok && strings.EqualFold(strings.TrimSpace(k), revisionLineKey) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// revisionInk is the colour a set's mark has in the editor.
func revisionInk(set string) color.Color {
	if c, ok := lex.RevisionInk(set); ok {
		return color.NRGBA{R: c[0], G: c[1], B: c[2], A: 0xff}
	}
	return color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff}
}

// The window's side.

// revisionState is what the window knows of the script's revisions.
type revisionState struct {
	key   string // what it was read for (revisionKey)
	chain *revisionChain
	err   error
	// the marks shown, by line, for the text they were computed for
	text  string
	marks []string
	ink   map[int]color.Color
}

// revisionKey is what the issues are read again for: another script, or
// a new issue (revisionGen).
func (w *MainWindow) revisionKey() string {
	p := w.currentFile
	if w.bundle != nil {
		p = w.bundle.path
	}
	return fmt.Sprintf("%s|%d", p, w.revisionGen)
}

// revisionChainNow are the script's issues, read again if need be.
func (w *MainWindow) revisionChainNow() (*revisionChain, error) {
	key := w.revisionKey()
	if w.revisions.key != key {
		c, err := readRevisionChain(w.extrasHome())
		w.revisions = revisionState{key: key, chain: c, err: err}
	}
	return w.revisions.chain, w.revisions.err
}

// marksAll tells whether the marks are all issues' (the setting).
func (w *MainWindow) marksAll() bool {
	return w.settings != nil && w.settings.GetString(revisionMarksPref) == "all"
}

// revisionPause is how long typing pauses before the marks are updated.
const revisionPause = 400 * time.Millisecond

// scheduleRevisionMarks updates the margin's marks once typing pauses.
func (w *MainWindow) scheduleRevisionMarks() {
	if w.markdown || w.textEditor == nil {
		return
	}
	w.debounce(&w.revisionTimer, revisionPause, w.updateRevisionMarks)
}

// updateRevisionMarks computes the marks of the text (in the background)
// and shows them in the editor's margin.
func (w *MainWindow) updateRevisionMarks() {
	if w.markdown || w.textEditor == nil {
		return
	}
	chain, err := w.revisionChainNow()
	if err != nil || chain == nil || len(chain.issues) == 0 {
		if w.revisions.ink != nil {
			w.revisions.ink, w.revisions.marks = nil, nil
			w.textEditor.RedrawMarks()
		}
		return
	}
	text, all, key := w.textEditor.Text(), w.marksAll(), w.revisions.key
	w.revisionRun++
	run := w.revisionRun
	type shown struct {
		marks []string
		ink   map[int]color.Color
	}
	inBackground(func() shown {
		marks, _, _ := chain.marksOf(text, all, time.Now())
		ink := map[int]color.Color{}
		for i, m := range marks {
			if m != "" {
				ink[i] = revisionInk(m)
			}
		}
		return shown{marks, ink}
	}, func(r shown) {
		if run != w.revisionRun || key != w.revisions.key {
			return // a later text's are coming
		}
		w.revisions.text, w.revisions.marks, w.revisions.ink = text, r.marks, r.ink
		w.textEditor.RedrawMarks()
	})
}

// revisionMarksShown are the margin's marks (the editor's Marks).
func (w *MainWindow) revisionMarksShown() map[int]color.Color { return w.revisions.ink }

// exportRevisions are the revisions of the text for an export: the sets
// and each line's mark, as the setting has them (nil when the script has
// no issue).
func (w *MainWindow) exportRevisions(text string) (*lex.Revisions, []string) {
	if w.markdown {
		return nil, nil
	}
	chain, err := w.revisionChainNow()
	if err != nil || chain == nil || len(chain.issues) == 0 {
		return nil, nil
	}
	marks, sets, _ := chain.marksOf(text, w.marksAll(), time.Now())
	r := &lex.Revisions{Sets: sets}
	if w.settings != nil {
		r.Header = w.settings.GetString(revisionHeaderPref)
		r.DateLayout = w.settings.GetString(revisionDatePref)
	}
	return r, marks
}

// keepImportedRevisions keeps an imported script's revisions: its text
// as an issue of its latest set, the marks on it in revisions.json, the
// production block's line.
func (w *MainWindow) keepImportedRevisions(from string, sets []lex.RevisionSet, marks []importedMark) {
	last := sets[len(sets)-1]
	e := w.textEditor
	text := e.Text()
	inside, _ := productionBlock(text)
	start, end, block := withProductionBlock(text, withBlockLine(inside, revisionLineKey, revisionLine(last, "imported from "+from)))
	e.Edit(func(b *buffer.Buffer) { b.Replace(start, end, block) })
	w.addDraftAs(last.Name+" (imported)", kindIssue, last.Name, last.Date, false, func(d savedDraft, ok bool, err error) {
		if err == nil && ok {
			imp := importedRevisions{Version: 1, Imported: &importedMarkSet{From: from, Draft: d.ID, Marks: marks}}
			for _, s := range sets {
				imp.Imported.Sets = append(imp.Imported.Sets, importedSet{Name: s.Name, Date: s.Date})
			}
			err = w.writeExtra(revisionsName, imp)
		}
		if err != nil {
			dialog.ShowError(fmt.Errorf("could not keep the script's revisions: %v", err), w.fyneWindow)
		}
		w.revisionGen++
		w.updateRevisionMarks()
		w.setStatus("Imported from Final Draft with its revisions (" + plural(len(sets), "set") + ", up to " + last.Name + "): changes from now on are " + (&revisionChain{issues: []issue{{set: last}}}).nextSet(time.Now()).Name)
	})
}

// writeExtra writes one of the extras as JSON where the script's are.
func (w *MainWindow) writeExtra(rel string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	files := []bundleFile{{rel, append(data, '\n')}}
	if w.bundle != nil {
		w.queueBundle(bundleJob{run: func(b *textBundle) (*textBundle, error) {
			if b == nil {
				return nil, errors.New("the script's bundle is gone")
			}
			return saveBundleFiles(b, files, false)
		}})
		return nil
	}
	home := w.extrasHome()
	if home.kind == extrasNone {
		return errors.New("the script has no place for its production files yet")
	}
	return home.dir.write(files)
}

// scriptIssued tells whether the script has an issue (its revisions can
// be exported).
func (w *MainWindow) scriptIssued() bool {
	if w.markdown {
		return false
	}
	c, err := w.revisionChainNow()
	return err == nil && c != nil && len(c.issues) > 0
}

// withRevisions is the export job with the text's revisions (none if the
// script has no issue): the marks, revised pages only, coloured paper.
func (w *MainWindow) withRevisions(j exportJob, text string, pagesOnly, colours bool) exportJob {
	r, marks := w.exportRevisions(text)
	if r == nil {
		return j
	}
	r.PagesOnly, r.PageColors = pagesOnly, colours
	j.revisions, j.marks = r, marks
	return j
}

// issueDraft issues the script as it is now as set s, dated date: the
// scene numbers locked first if they are not, the production block's
// revision line written, the text kept as an issue. done (may be nil)
// gets the error.
func (w *MainWindow) issueDraft(s lex.RevisionSet, done func(error)) {
	finish := func(err error) {
		if done != nil {
			done(err)
		}
	}
	if !sceneLockOf(w.textEditor.Text()).locked {
		w.lockSceneNumbers()
		if !sceneLockOf(w.textEditor.Text()).locked {
			finish(errors.New("the scene numbers could not be locked: the script needs scene headings"))
			return
		}
	}
	e := w.textEditor
	text := e.Text()
	inside, _ := productionBlock(text)
	start, end, block := withProductionBlock(text, withBlockLine(inside, revisionLineKey, revisionLine(s, "")))
	e.Edit(func(b *buffer.Buffer) {
		c := b.Cursor()
		b.Group(func() {
			b.Replace(start, end, block)
			b.SetCursor(min(c, utf8.RuneCountInString(b.Text())), false)
		})
	})
	w.addDraftAs(s.Name, kindIssue, s.Name, s.Date, false, func(_ savedDraft, _ bool, err error) {
		w.revisionGen++
		w.updateRevisionMarks()
		finish(err)
	})
}
