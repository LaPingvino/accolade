// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"github.com/LaPingvino/accolade/internal/collab"
	"github.com/LaPingvino/accolade/internal/editor"
)

// The others' places (docs/COLLAB.md, presence): each co-writer here is
// drawn in the script's editor in their colour (session_tab.go's
// peerColor), a bar in the gutter on their cursor's line and their
// selection tinted (internal/editor peers.go), and as a dot on the
// scene they are in, in the outline and on the cards. Their places come
// as anchors in the script's Doc (collab.Presence.At, From), placed in
// this side's text when their presence comes and after each of the
// peers' edits; between those, this side's own typing moves them with
// the text (buffer marks). Nothing of it is in the way of typing: the
// bar is in the gutter, a tap on it says whose it is and leaves the
// cursor where it is.

// selectionSayPause is how long a selection rests before it is said to
// the others (a drag says it once it stops, not every step).
const selectionSayPause = 150 * time.Millisecond

// hereSaid is what this side said of its place last: the line, and the
// selection's ends (equal: none).
type hereSaid struct{ line, from, to int }

// showPeers places the others in the script's editor (the outline's and
// the cards' dots follow).
func (w *MainWindow) showPeers() {
	e := w.textEditor
	if e == nil {
		return
	}
	s := w.shared
	if s == nil {
		e.SetPeers(nil)
		w.peerDotsChanged()
		return
	}
	b := e.Buffer()
	var peers []editor.Peer
	same := false
	here := s.session.Here()
	s.session.Do(func(d *collab.Doc) {
		if same = d.Len() == b.Len(); !same {
			return // the editor not caught up: kept as they are, moved with the text
		}
		for _, p := range here {
			if p.View {
				continue // a viewer follows only: no cursor, no bar (tableread_follow.go)
			}
			q := editor.Peer{Name: p.Name, Color: peerRGB(p.Replica), Cursor: -1}
			if p.At != nil {
				if at, ok := d.Place(*p.At); ok {
					q.Cursor, q.From = at, at
					if p.From != nil {
						if f, ok := d.Place(*p.From); ok {
							q.From = f
						}
					}
				}
			}
			if q.Cursor < 0 { // said as a line only, or its runes not here yet
				q.Cursor = b.LineStart(max(0, min(p.Line, b.LineCount()-1)))
				q.From = q.Cursor
			}
			peers = append(peers, q)
		}
	})
	if !same {
		return
	}
	e.SetPeers(peers)
	w.peerDotsChanged()
}

// peerRGB is replica's colour as a colour.
func peerRGB(replica string) color.Color {
	if r, g, b, ok := parseHexRGB(peerColor(replica)); ok {
		return color.NRGBA{R: r, G: g, B: b, A: 0xff}
	}
	return color.NRGBA{A: 0xff}
}

// peerTip shows whose bar the pointer is over, or that was tapped.
func (w *MainWindow) peerTip(names string, at fyne.Position, cell fyne.Size) {
	if names != "" {
		names += " is here"
		if strings.Contains(names, ", ") {
			names = strings.Replace(names, " is here", " are here", 1)
		}
	}
	if w.tooltips == nil {
		if names != "" {
			w.setStatus(names)
		}
		return
	}
	w.showEditorTip(names, at, cell)
}

// sayHereNow says this side's place: the line at once when it changed, a
// selection once it rests (always: again, as it is).
func (w *MainWindow) sayPlace(always bool) {
	s := w.shared
	if s == nil || s.paused || w.textEditor == nil {
		return
	}
	b := w.textEditor.Buffer()
	caughtUp := false
	s.session.Do(func(d *collab.Doc) { caughtUp = d.Len() == b.Len() })
	if !caughtUp {
		return // the key not in the Doc yet: said once it is (shareEdit)
	}
	cursor := b.Cursor()
	from, to := b.Selection()
	now := hereSaid{line: w.hereLine()} // the Table Read's, while it leads (tableread_follow.go)
	if from != to {                     // a cursor alone is said by its line, not at each step along it
		now.from, now.to = from, to
	}
	if !always && s.saidHere && now == s.said {
		return
	}
	if !always && s.saidHere && now.line == s.said.line {
		w.debounce(&s.sayTimer, selectionSayPause, func() {
			if w.shared == s {
				w.sayPlace(true)
			}
		})
		return
	}
	s.sayTimer.Stop()
	s.said, s.saidHere = now, true
	anchor := cursor
	if from != to {
		anchor = from + to - cursor // the selection's other end
	}
	if s.followOnly || now.line != w.scriptCursorLine() {
		cursor, anchor = -1, -1 // a viewer, or the Table Read leading: its line, no cursor to draw
	}
	s.session.SetHereAt(w.yourName(), now.line, cursor, anchor)
}

// peersInScenes are the dots of who is in which of items (the outline's
// or the cards'): the item each one's cursor is in, by its index.
func (w *MainWindow) peersInScenes(items []outlineItem) map[int][]color.Color {
	if w.shared == nil || w.textEditor == nil || len(items) == 0 {
		return nil
	}
	peers := w.textEditor.Peers()
	if len(peers) == 0 {
		return nil
	}
	b := w.textEditor.Buffer()
	out := map[int][]color.Color{}
	for _, p := range peers {
		line, _ := b.Position(p.Cursor)
		at := -1
		for i, it := range items {
			if it.Line > line {
				break
			}
			at = i
		}
		if at >= 0 {
			out[at] = append(out[at], p.Color)
		}
	}
	return out
}

// peerDotsChanged redraws the outline's and the cards' dots, if they
// changed.
func (w *MainWindow) peerDotsChanged() {
	if o := w.outline; o != nil && o.ofScript() {
		if d := w.peersInScenes(o.items); !sameDots(d, o.dots) {
			o.dots = d
			o.list.Refresh()
		}
	}
	if c := w.cardsDock; c != nil {
		if d := w.peersInScenes(c.items); !sameDots(d, c.dots) {
			c.dots = d
			c.list.Refresh()
		}
	}
	if c := w.cards; c != nil {
		c.showDots()
	}
}

func sameDots(a, b map[int][]color.Color) bool {
	if len(a) != len(b) {
		return false
	}
	for k, x := range a {
		y := b[k]
		if len(x) != len(y) {
			return false
		}
		for i := range x {
			if x[i] != y[i] {
				return false
			}
		}
	}
	return true
}

// dotSize is a co-writer's dot's diameter.
const dotSize = 9

// showRowDots shows the dots in their room (box: a centring container
// holding newPeerDots's), hidden with none.
func showRowDots(box *fyne.Container, colors []color.Color) {
	showPeerDots(box.Objects[0].(*fyne.Container), colors)
	if len(colors) == 0 {
		box.Hide()
	} else {
		box.Show()
	}
}

// newPeerDots is the room for a row's dots (showPeerDots fills it).
func newPeerDots() *fyne.Container { return container.NewHBox() }

// showPeerDots puts a dot in each colour in box (none: empty, no room).
func showPeerDots(box *fyne.Container, colors []color.Color) {
	if len(box.Objects) == len(colors) {
		same := true
		for i, c := range colors {
			if box.Objects[i].(*fyne.Container).Objects[0].(*canvas.Circle).FillColor != c {
				same = false
			}
		}
		if same {
			return
		}
	}
	box.Objects = box.Objects[:0]
	for _, c := range colors {
		box.Objects = append(box.Objects, container.NewGridWrap(fyne.NewSize(dotSize, dotSize), canvas.NewCircle(c)))
	}
	box.Refresh()
}
