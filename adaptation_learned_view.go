// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// What Accolade learned: the source pane's list of the lessons in the
// adaptation block (adaptation_learned.go), each with a button that
// takes it away, for a wrong one. Folded under one line by default; the
// line is there only when something was learned.

// learnedHeight is how tall the open list is at most, before it scrolls.
const learnedHeight = 150

// lesson is a row of the list: what it says, and how it is taken away.
type lesson struct {
	text   string
	forget func(l *learned, ln *learner)
}

// lessonsOf are the rows for what was learned.
func lessonsOf(l learned) []lesson {
	var out []lesson
	for _, a := range l.Aliases {
		for _, n := range a.Names {
			cue, name := a.Cue, n
			out = append(out, lesson{"“" + name + "” is " + cue, func(l *learned, ln *learner) {
				l.dropAlias(name)
				ln.forget(aliasLesson(cue, name))
			}})
		}
	}
	for _, n := range l.Not {
		name := n
		out = append(out, lesson{"“" + name + "” is no one", func(l *learned, ln *learner) {
			l.dropNot(name)
			ln.forget(notLesson(name))
			delete(ln.strikes, wordsKey(name))
		}})
	}
	for _, p := range l.Places {
		pl := p
		out = append(out, lesson{pl.Place + ": " + pl.Heading, func(l *learned, ln *learner) {
			l.dropPlace(pl.Place)
			ln.forget(placeLesson(pl.Place, pl.Heading))
		}})
	}
	for i, sk := range l.Skipped {
		k := i
		q := strings.Trim(shortQuote(sk.Link.Quote), `"“”„«»'‘’ `)
		out = append(out, lesson{"Not linked: " + sk.Heading + ", “" + q + "”", func(l *learned, ln *learner) {
			if k < len(l.Skipped) {
				l.Skipped = append(l.Skipped[:k], l.Skipped[k+1:]...)
			}
		}})
	}
	return out
}

// learnedView is the list in the pane.
type learnedView struct {
	toggle *widget.Button
	rows   *fyne.Container
	scroll *container.Scroll
	box    *fyne.Container
	open   bool
	shown  string // the block's inside it shows
	ok     bool
	n      int
}

func newLearnedView(s *sourcePane) *learnedView {
	v := &learnedView{}
	v.toggle = widget.NewButtonWithIcon("", theme.MenuExpandIcon(), func() {
		v.open = !v.open
		v.layout()
	})
	v.toggle.Importance = widget.LowImportance
	v.toggle.Alignment = widget.ButtonAlignLeading
	v.rows = container.NewVBox()
	v.scroll = container.NewVScroll(v.rows)
	v.scroll.Hide()
	v.box = container.NewVBox(v.toggle, v.scroll)
	v.box.Hide()
	return v
}

// update shows what the script's block holds, if it changed.
func (v *learnedView) update(s *sourcePane) {
	inside, _ := adaptationBlock(s.w.textEditor.Text())
	if v.ok && sameString(v.shown, inside) {
		return
	}
	v.shown, v.ok = inside, true
	ls := lessonsOf(learnedOf(s.w.textEditor.Text()))
	v.n = len(ls)
	v.rows.Objects = nil
	for _, ls := range ls {
		label := widget.NewLabel(ls.text)
		label.Wrapping = fyne.TextWrapWord
		label.SizeName = theme.SizeNameCaptionText
		forget := ls.forget
		text := ls.text
		del := widget.NewButtonWithIcon("", theme.ContentRemoveIcon(), func() { s.forgetLesson(text, forget) })
		del.Importance = widget.LowImportance
		fyne.SetAccessibleLabel(del, "Forget: "+text)
		v.rows.Add(container.NewBorder(nil, nil, nil, del, label))
	}
	if v.n == 0 {
		v.open = false
	}
	v.layout()
}

// layout shows the line, and the list under it when open.
func (v *learnedView) layout() {
	if v.n == 0 {
		v.box.Hide()
		return
	}
	icon := theme.MenuExpandIcon()
	if v.open {
		icon = theme.MenuDropDownIcon()
	}
	v.toggle.SetIcon(icon)
	v.toggle.SetText("What Accolade learned (" + strconv.Itoa(v.n) + ")")
	what := "Show what Accolade learned from your writing"
	if v.open {
		what = "Hide what Accolade learned"
	}
	fyne.SetAccessibleLabel(v.toggle, what)
	if v.open {
		v.rows.Refresh()
		v.scroll.SetMinSize(fyne.NewSize(0, min(v.rows.MinSize().Height, learnedHeight)))
		v.scroll.Show()
	} else {
		v.scroll.Hide()
	}
	v.box.Show()
	v.box.Refresh()
}

// forgetLesson takes a lesson out of the block (one undo step), and keeps
// it from being learned again this session.
func (s *sourcePane) forgetLesson(text string, forget func(l *learned, ln *learner)) {
	w := s.w
	l := learnedOf(w.textEditor.Text())
	forget(&l, &w.suggester().learn)
	if w.writeLearned(l, false) {
		w.setStatus("Forgotten: " + text)
	}
	s.learned.update(s)
}
