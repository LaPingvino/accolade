package editor

import (
	"image/color"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Co-writers (editing together): where each of the others is in the
// text, drawn so as never to be in the way of typing: a bar in their
// colour in the gutter's last column, on every row of the line their
// cursor is on (a folded line's: its heading's), and their selection's
// text faintly in their colour. The pointer over a bar, or a tap on it,
// tells OnPeer whose it is. Their places are buffer marks, so the
// editor's own edits move them with the text until they are set again.

// Peer is a co-writer: their name and colour, their cursor's offset, and
// the other end of their selection (From == Cursor: none).
type Peer struct {
	Name         string
	Color        color.Color
	Cursor, From int
}

// peerDrawing is what the rows draw of the peers: the lines with a bar
// (the first peer's on it, in order), and the selections.
type peerDrawing struct {
	bar  map[int]*widget.CustomTextGridStyle
	sels []peerSel
}

type peerSel struct {
	start, end int
	bg         color.Color
}

// SetPeers shows the co-writers (nil: none) and draws them.
func (e *ScriptEditor) SetPeers(peers []Peer) {
	if len(peers) == 0 && len(e.peers) == 0 {
		return
	}
	e.peers = append(e.peers[:0], peers...)
	marks := make([]int, 0, 2*len(peers))
	for _, p := range peers {
		marks = append(marks, p.Cursor, p.From)
	}
	e.buf.SetMarks(marks)
	e.relayout()
}

// Peers are the co-writers shown, their places moved with the text since
// SetPeers.
func (e *ScriptEditor) Peers() []Peer {
	out := append([]Peer(nil), e.peers...)
	m := e.buf.Marks()
	for i := range out {
		if 2*i+1 < len(m) {
			out[i].Cursor, out[i].From = m[2*i], m[2*i+1]
		}
	}
	return out
}

// PeerLines are the lines the co-writers' bars are drawn on, each with
// their names (for a test, or a reader of the screen).
func (e *ScriptEditor) PeerLines() map[int]string {
	e.fresh()
	out := map[int]string{}
	for _, p := range e.Peers() {
		line := e.lineShown(p.Cursor)
		if out[line] != "" {
			out[line] += ", "
		}
		out[line] += p.Name
	}
	return out
}

// lineShown is the line whose rows show offset: its own, or, folded
// away, the line before it that has rows (the fold's heading).
func (e *ScriptEditor) lineShown(off int) int {
	rows := e.layout.Rows
	i := sort.Search(len(rows), func(i int) bool { return rows[i].Start > off }) - 1
	if i < 0 {
		return 0
	}
	return rows[i].Line
}

// drawPeers is the peers' drawing as the layout is now.
func (e *ScriptEditor) drawPeers() peerDrawing {
	var d peerDrawing
	if len(e.peers) == 0 || len(e.layout.Rows) == 0 {
		return d
	}
	for _, p := range e.Peers() {
		r, g, b, _ := p.Color.RGBA()
		line := e.lineShown(p.Cursor)
		if d.bar == nil {
			d.bar = map[int]*widget.CustomTextGridStyle{}
		}
		if d.bar[line] == nil {
			d.bar[line] = &widget.CustomTextGridStyle{FGColor: p.Color,
				BGColor: color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 0x38}}
		}
		if p.From != p.Cursor {
			d.sels = append(d.sels, peerSel{min(p.From, p.Cursor), max(p.From, p.Cursor),
				color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 0x40}})
		}
	}
	return d
}

// peerSelAt is the background of a peer's selection at off (nil: none).
func (d peerDrawing) peerSelAt(off int) color.Color {
	for _, s := range d.sels {
		if off >= s.start && off < s.end {
			return s.bg
		}
	}
	return nil
}

// peerBarAt is whose bar is at p, in the editor, and its cell's place.
func (e *ScriptEditor) peerBarAt(p fyne.Position) (string, fyne.Position, bool) {
	e.fresh()
	g := e.gutter()
	if len(e.peers) == 0 || g == 0 || p.Y < 0 || len(e.layout.Rows) == 0 {
		return "", fyne.Position{}, false
	}
	row, col := int(p.Y/e.cellHeight()), int(p.X/e.cellWidth())
	if row >= len(e.layout.Rows) || col != g-1 {
		return "", fyne.Position{}, false
	}
	var names []string
	line := e.layout.Rows[row].Line
	for _, q := range e.Peers() {
		if e.lineShown(q.Cursor) == line {
			names = append(names, q.Name)
		}
	}
	if len(names) == 0 {
		return "", fyne.Position{}, false
	}
	return strings.Join(names, ", "), fyne.NewPos(float32(g-1)*e.cellWidth(), float32(row)*e.cellHeight()), true
}

// peerHover tells OnPeer of the bar under the pointer, or that it left
// one; true when the pointer is on one.
func (e *ScriptEditor) peerHover(p fyne.Position, out bool) bool {
	if e.OnPeer == nil {
		return false
	}
	names, at, ok := "", fyne.Position{}, false
	if !out {
		names, at, ok = e.peerBarAt(p)
	}
	if !ok && !e.peerShown {
		return false
	}
	e.peerShown = ok
	e.OnPeer(names, at, fyne.NewSize(e.cellWidth(), e.cellHeight()))
	return ok
}

// PeerBarAt is where the co-writers' bar on line's first row is drawn
// (false: no bar there, or the line not laid out): for a tip shown from
// elsewhere, or a test's pointer.
func (e *ScriptEditor) PeerBarAt(line int) (fyne.Position, fyne.Size, bool) {
	e.fresh()
	g := e.gutter()
	if g == 0 || e.draw.peers.bar[line] == nil {
		return fyne.Position{}, fyne.Size{}, false
	}
	for i, r := range e.layout.Rows {
		if r.Line == line {
			return fyne.NewPos(float32(g-1)*e.cellWidth(), float32(i)*e.cellHeight()), fyne.NewSize(e.cellWidth(), e.cellHeight()), true
		}
	}
	return fyne.Position{}, fyne.Size{}, false
}
