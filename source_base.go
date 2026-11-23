package main

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// What "changed since the script was linked" is measured from
// (docs/LAYERS.md): the source as the script was linked to it, the newest
// source lock in the drafts store (stages.go), else the source as the
// pane first read it this session. The source's passages now are aligned
// with the base's (relink.go's alignPassagesSure, in the background for a
// long one); a linked passage whose words differ from the base's passage
// in its place is changed: its status bar in the warning colour, a notice
// over its text, Relink From Here on the picked one.

// sourceBase is the pane's base and its diff with the source now.
type sourceBase struct {
	session sessionRead // the source as the pane first read it, for a script
	// the diff as made last: of base with src, the source now's passages
	// whose words differ (by index)
	diffBase, diffSrc string
	diff              map[int]bool
	running           int // the diff being made (a later one drops it)
}

// sessionRead is the source as first read this session, for the script
// and the source it was read for.
type sessionRead struct {
	script string
	ref    sourceRef
	src    string
	set    bool
}

// baseText is the base: the newest lock's source (locked), else the
// session's first reading of the source now.
func (s *sourcePane) baseText() (src string, locked bool) {
	if lock, lockSrc := s.w.sourceLock(); lock != nil {
		return lockSrc, true
	}
	r := &s.base.session
	if !r.set || r.ref != s.ref || r.script != s.w.currentFile {
		*r = sessionRead{script: s.w.currentFile, ref: s.ref, src: s.src, set: true}
	}
	return r.src, false
}

// changedPassages are the passages of nw whose words differ from those of
// the passage of old aligned with them (a passage added has none).
func changedPassages(old, nw []passage) map[int]bool {
	to, _ := alignPassagesSure(old, nw)
	out := map[int]bool{}
	for k, i := range to {
		if i >= 0 && i < len(nw) && !slices.Equal(old[k].Words, nw[i].Words) {
			out[i] = true
		}
	}
	return out
}

// baseDiff is the diff of the source now with the base, if made; one is
// started if not (the pane follows when it is in).
func (s *sourcePane) baseDiff() map[int]bool {
	base, _ := s.baseText()
	b := &s.base
	if sameString(base, s.src) {
		b.diffBase, b.diffSrc, b.diff = base, s.src, nil
		return nil
	}
	if sameString(b.diffBase, base) && sameString(b.diffSrc, s.src) {
		return b.diff
	}
	b.running++
	run, src, ps := b.running, s.src, s.passages
	sync := true
	inBackground(func() map[int]bool { return changedPassages(passagesOf(base), ps) }, func(d map[int]bool) {
		if run != b.running || !sameString(src, s.src) {
			return
		}
		b.diffBase, b.diffSrc, b.diff = base, src, d
		if !sync && s.shown() {
			s.follow()
		}
	})
	sync = false
	if sameString(b.diffBase, base) && sameString(b.diffSrc, s.src) {
		return b.diff // made at once (tests)
	}
	return nil
}

// noticed tells whether row id shows the notice: a linked passage changed
// since.
func (s *sourcePane) noticed(id int) bool { return s.changed[id] }

// changedNotice is the notice over a changed passage's text.
func changedNotice() widget.RichTextSegment {
	return &widget.TextSegment{Text: "Changed since the script was linked", Style: widget.RichTextStyle{
		ColorName: theme.ColorNameWarning, SizeName: theme.SizeNameCaptionText, TextStyle: fyne.TextStyle{Italic: true},
	}}
}

// linksSeen follows whether the script links its source: the first link
// of a session with no source locked locks it (stages.go), as adapting
// starts. found: the script's links that find a passage now.
func (s *sourcePane) linksSeen(found bool) {
	starts := s.linksOf == s.of && !s.hadLinks && found
	if s.linksOf != s.of || s.hadLinks != found {
		s.linksOf, s.hadLinks = s.of, found
		s.w.refreshStage()
	}
	if starts {
		if lock, _ := s.w.sourceLock(); lock == nil {
			s.w.lockSource(true, nil)
		}
	}
}
