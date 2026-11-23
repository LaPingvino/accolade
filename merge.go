package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"fyne.io/fyne/v2/dialog"

	"github.com/LaPingvino/accolade/internal/scriptdiff"
	"github.com/LaPingvino/accolade/internal/structure"
)

// File > Merge Changes… (LOOP.md, Joop 2026-11-20): someone edits a copy
// outside Accolade — a co-writer in Highland or Final Draft, production's
// FDX, the flattened .fountain — and it comes back. The drafts store
// (drafts.go) holds the version it was made from, so the changes are
// told apart three ways (internal/scriptdiff's merge): theirs are offered
// in the Compare middle tab (drafts_view.go, merge_view.go), each to Take
// or Leave; ours stay; where both changed the same element both are
// shown, to pick or edit. Nothing is written until one takes a change
// and merges: then only what was taken goes in, as one undo step and a
// new draft, the script as it was kept as a draft first. Stored drafts
// and issues are never written into (the locked stages of LOOP.md's
// layers stay as they are).
//
// What an outside copy cannot carry is not counted as their change: an
// FDX has no sections, synopses or boneyard and moves a note into the
// next cue; a Clean Fountain has no source links; either may lose the
// scene numbers. Those are left out of the comparison, and a change taken
// keeps ours' (links, notes, a locked number on its heading). Accolade's
// blocks at the end (Source, Adaptation, Production, Characters, Notes:
// the cut marks among them) are ours and stay; the tags and strips follow
// their words and scenes, as after a structural move.

// outsideCopy is the copy from outside, read.
type outsideCopy struct {
	path string
	name string // its file's name, as the compare says it
	text string // in the editor's form
	// lossy: a format without Fountain's outline, boneyard and inline
	// notes (FDX)
	lossy bool
}

// outsideID stands for the outside copy among the versions to compare.
const outsideID = "\x00outside"

// mergeExtensions are the files Merge Changes… reads: the scripts
// Accolade opens, without PDF (its text read back is for later).
func mergeExtensions() []string {
	return slices.DeleteFunc(openExtensions(), func(e string) bool { return strings.EqualFold(e, ".pdf") })
}

// readOutside reads a copy of the script from outside as the editor has
// a script's text: a bundle's text (Highland's change if it made one), a
// Final Draft file converted, a plain file as opening it reads it.
func (w *MainWindow) readOutside(path string) (*outsideCopy, error) {
	oc := &outsideCopy{path: path, name: filepath.Base(path)}
	ext := strings.ToLower(filepath.Ext(path))
	if b, ok := bundleOf(path); ok {
		path = b
	}
	var text string
	switch {
	case ext == ".pdf":
		return nil, errors.New("a PDF cannot be merged yet: open it as an import and compare with that")
	case ext == ".fdx":
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if text, err = importFDX(f); err != nil {
			return nil, fmt.Errorf("could not read %s: %v", oc.name, err)
		}
		oc.lossy = true
	case isBundlePath(path):
		b, content, err := readBundle(path)
		if errors.Is(err, errNotZip) && ext == ".highland" { // Fountain with Highland's name
			content, err = os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			text, _ = decodeScript(content)
			text, _ = plainFileText(path, text)
			break
		}
		if err != nil {
			return nil, err
		}
		text, _ = decodeScript(content)
		if !w.markdown {
			text, _ = bundleScriptText(b, content, text)
		}
	default:
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		text, _ = decodeScript(data)
		if !w.markdown {
			text, _ = plainFileText(path, text)
		}
	}
	if !w.markdown {
		text = withoutIndents(text, w.sceneStarts())
	}
	oc.text = text
	return oc, nil
}

// mergeCarried is what the outside copy carries of what Accolade writes:
// what it has none of is left out of the comparison.
type mergeCarried struct {
	links, notes, numbers, outline bool
}

var (
	mergeLinkRe   = regexp.MustCompile(`(?s)\s?\[\[from:.*?\]\]`)
	mergeNoteRe   = regexp.MustCompile(`(?s)\s?\[\[.*?\]\]`)
	mergeNumberRe = regexp.MustCompile(`\s+#[\p{L}\p{N}.\-]+#\s*$`)
)

// carriedBy is what text carries; a lossy copy (FDX) has no outline,
// boneyard or notes of its own: what it brings is not Fountain's.
func carriedBy(text string, lossy bool, isHeading func(string) bool) mergeCarried {
	c := mergeCarried{links: strings.Contains(text, "[[from:"), notes: !lossy && strings.Contains(text, "[[")}
	for _, e := range scriptdiff.Split(text, isHeading) {
		first, _, _ := strings.Cut(e.Text, "\n")
		if e.Heading && mergeNumberRe.MatchString(first) {
			c.numbers = true
		}
		if !lossy && outlineElement(e.Text) {
			c.outline = true
		}
	}
	return c
}

// outlineElement: a section, a synopsis or a boneyard.
func outlineElement(s string) bool {
	return strings.HasPrefix(s, "#") || strings.HasPrefix(s, "=") || strings.HasPrefix(s, "/*")
}

// mergeOptions are how the script now, its drafts and theirs are
// compared for a merge.
func (w *MainWindow) mergeOptions(theirs string, lossy bool) scriptdiff.MergeOptions {
	var heading func(string) bool
	if w.markdown {
		heading = func(l string) bool { return strings.HasPrefix(l, "#") }
	} else {
		starts := w.sceneStarts()
		heading = func(l string) bool { return isSceneHeadingIn(l, starts) || strings.HasPrefix(l, "# ") }
	}
	c := carriedBy(theirs, lossy, heading)
	o := scriptdiff.MergeOptions{IsHeading: heading, Key: mergeKey(c, w.markdown), Carry: mergeCarry(c)}
	if !w.markdown {
		o.Body = bodyEndByte
	}
	return o
}

// mergeKey is an element as compared: without what the copy does not
// carry, its spaces as one; "" for what it cannot carry at all.
func mergeKey(c mergeCarried, markdown bool) func(string) string {
	return func(s string) string {
		if !markdown && !c.outline && outlineElement(s) {
			return ""
		}
		if !c.notes {
			s = mergeNoteRe.ReplaceAllString(s, "")
		} else if !c.links {
			s = mergeLinkRe.ReplaceAllString(s, "")
		}
		lines := strings.Split(s, "\n")
		out := lines[:0]
		for i, l := range lines {
			if i == 0 && !c.numbers && !markdown {
				l = mergeNumberRe.ReplaceAllString(l, "")
			}
			l = strings.Join(strings.Fields(l), " ")
			if strings.HasPrefix(l, ">") && strings.HasSuffix(l, "<") && len(l) > 1 { // centred, however spaced
				l = ">" + strings.TrimSpace(l[1:len(l)-1]) + "<"
			}
			if l != "" {
				out = append(out, l)
			}
		}
		return strings.Join(out, "\n")
	}
}

// mergeCarry is theirs' text for ours' element: with ours' links (or
// notes) where the copy has none, and ours' heading's number where
// theirs' heading has none.
func mergeCarry(c mergeCarried) func(o, t string) string {
	return func(o, t string) string {
		re := mergeLinkRe
		if !c.notes {
			re = mergeNoteRe
		}
		if !c.links || !c.notes {
			var add []string
			for _, n := range re.FindAllString(o, -1) {
				if n = strings.TrimSpace(n); !strings.Contains(t, n) {
					add = append(add, n)
				}
			}
			if len(add) > 0 {
				t = strings.TrimRight(t, " ") + " " + strings.Join(add, " ")
			}
		}
		{ // a locked number stays, whether the copy has numbers or not
			of, _, _ := strings.Cut(o, "\n")
			tf, rest, more := strings.Cut(t, "\n")
			if n := mergeNumberRe.FindString(of); n != "" && !mergeNumberRe.MatchString(tf) {
				tf = strings.TrimRight(tf, " ") + " " + strings.TrimSpace(n)
				if more {
					tf += "\n" + rest
				}
				t = tf
			}
		}
		return t
	}
}

// mergeBase is the version theirs was most likely made from: of the
// drafts (newest first) and the script now, the one with most of
// theirs' elements as they are, then the fewest of its own theirs has
// not, then the newest (a draft before the script now).
func (w *MainWindow) mergeBase(drafts []savedDraft, theirs string, o scriptdiff.MergeOptions) string {
	best, bestSame, bestOther := nowID, -1, 0
	seen := map[string]bool{}
	try := func(id, text string) {
		same, other := scriptdiff.Resemblance(text, theirs, o)
		if same > bestSame || (same == bestSame && other < bestOther) {
			best, bestSame, bestOther = id, same, other
		}
	}
	for _, d := range drafts { // newest first
		if seen[d.Text] {
			continue
		}
		seen[d.Text] = true
		data, err := w.extrasHome().readExtra(d.Text)
		if err != nil {
			continue
		}
		text, _ := decodeScript(data)
		try(d.ID, text)
	}
	try(nowID, w.textEditor.Text())
	return best
}

// mergeChanges is File > Merge Changes…: a file to merge, chosen.
func (w *MainWindow) mergeChanges() {
	w.chooseFile(mergeExtensions(), w.mergeFrom)
}

// mergeFrom merges the copy at path: read, its base found, compared.
func (w *MainWindow) mergeFrom(path string) {
	oc, err := w.readOutside(path)
	if err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return
	}
	w.afterBundleWrites(func() {
		ds, err := readDrafts(w.extrasHome())
		if err != nil {
			dialog.ShowError(err, w.fyneWindow)
			return
		}
		slices.Reverse(ds) // newest first
		base := w.mergeBase(ds, oc.text, w.mergeOptions(oc.text, oc.lossy))
		showMerge(w, ds, base, oc)
	})
}

// applyMerge puts what was taken into the script: the script as it is
// kept as a draft first (unless one holds it), the text changed in one
// undo step with the extras following, and kept as a new draft. done
// (may be nil) gets the error, on the UI goroutine.
func (w *MainWindow) applyMerge(ms *mergeState, done func(error)) {
	finish := func(err error) {
		if done != nil {
			done(err)
		}
	}
	text := w.textEditor.Text()
	if ms.m == nil || text != ms.ours {
		finish(errors.New("the script changed since it was compared: compare again"))
		return
	}
	after, lineMap, pos := ms.m.Apply(ms.take, ms.edits)
	if after == text {
		finish(errors.New("nothing taken: Take the changes to merge first"))
		return
	}
	n := scriptdiff.Taken(ms.take, ms.edits)
	name := ms.from
	w.addDraft("Before merging "+name, true, func(_ savedDraft, _ bool, err error) {
		if err != nil {
			finish(err)
			return
		}
		if w.textEditor.Text() != text {
			finish(errors.New("the script changed since it was compared: compare again"))
			return
		}
		first := 0
		for first < len(lineMap) && lineMap[first] == first {
			first++
		}
		lines := strings.Split(after, "\n")
		first = min(first, len(lines)-1)
		res := structure.Result{Lines: lines, Map: lineMap, Pos: pos, First: first, Last: first,
			Msg: "Merged " + plural(n, "change") + " from " + name + ", kept as a new draft"}
		w.applyStruct(text, res, false, -1, 0)
		like := savedDraft{Name: "Merged from " + name, Kind: kindDraft, MergedFrom: name, Base: ms.baseID}
		w.addDraftLike(like, false, func(_ savedDraft, _ bool, err error) { finish(err) })
	})
}
