// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/LaPingvino/lexington/lex"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/images"
)

// Storyboard frames in a screenplay, in Storyboard Fountain's notation
// (setpixel/storyboard-fountain, MIT; lex.Board): a note alone on its
// line under the line it illustrates,
//
//	[[board: shot-01.png, duration: 2.50, continued: true]]
//
// the picture's file in the folder a [[dataPath: …]] note names (beside
// the script), else beside the script or in its bundle (assets/). Plain
// Fountain notes: other programs keep them. In the editor a frame is a
// chip (as the source links are, link_chips.go): its thumbnail and name,
// the cursor passing into it to show the note, a tap showing it large.
// Insert › Storyboard Frame… adds one, its picture copied into the
// bundle's assets/ (or a folder beside the script) and the note written
// the same way. Page Setup's "Print storyboard frames" (off) prints them
// under their lines (Lexington's PDFWriter.Pictures).

// frameThumbCols is how many cells a frame chip's thumbnail takes.
const frameThumbCols = 4

// boardDataPath is the folder a script's [[dataPath: …]] note names
// ("" none), slashes as written.
func boardDataPath(text string) string {
	at := 0
	for {
		i := strings.Index(text[at:], "[[")
		if i < 0 {
			return ""
		}
		i += at
		j := strings.Index(text[i:], "]]")
		if j < 0 {
			return ""
		}
		note := strings.TrimSpace(text[i+2 : i+j])
		if len(note) > 9 && strings.EqualFold(note[:9], "dataPath:") {
			return strings.TrimSpace(note[9:])
		}
		at = i + j + 2
	}
}

// boardSrc is where a frame's picture is, as the loader names it: in the
// data path's folder when the script has one.
func boardSrc(b lex.Board, dataPath string) string {
	f := b.File
	if dataPath == "" || images.IsRemote(f) || path.IsAbs(f) || filepath.IsAbs(f) {
		return f
	}
	return path.Join(dataPath, f)
}

// frameLine is a storyboard frame's note: the runes of its line it takes,
// [start, end), counted from the text's start.
type frameLine struct {
	start, end int
	board      lex.Board
}

// findFrames are the frames in a script's body: board notes alone on
// their lines (outside a boneyard).
func findFrames(body string) []frameLine {
	if !strings.Contains(body, "[[") {
		return nil
	}
	var out []frameLine
	off, inBoneyard := 0, false
	for len(body) > 0 {
		line := body
		if i := strings.IndexByte(body, '\n'); i >= 0 {
			line, body = body[:i], body[i+1:]
		} else {
			body = ""
		}
		t := strings.TrimSpace(line)
		switch {
		case inBoneyard || strings.HasPrefix(t, "/*"):
			inBoneyard = !strings.Contains(strings.TrimPrefix(t, "/*"), "*/")
		case strings.HasPrefix(t, "[[") && strings.HasSuffix(t, "]]") && strings.Count(t, "[[") == 1:
			if b, ok := lex.ParseBoard(t[2 : len(t)-2]); ok {
				lead := utf8.RuneCountInString(line[:strings.Index(line, "[[")])
				out = append(out, frameLine{start: off + lead, end: off + lead + utf8.RuneCountInString(t), board: b})
			}
		}
		off += utf8.RuneCountInString(line) + 1
	}
	return out
}

// frameCache is the script's frame chips as made last.
type frameCache struct {
	text   string
	pics   *images.Loader
	gen    int // the loader's, when made
	sb     *sbProject
	chips  []editor.Chip
	merged []editor.Chip // with the links' (linkChips)
	links  []editor.Chip // the links' they were merged with
	of     []editor.Chip // and the frames'
}

// frameChips are the script's storyboard frames as chips: a thumbnail
// and the picture's name (a missing picture's name in the warning
// colour), the note as their tip.
func (w *MainWindow) frameChips(text string) []editor.Chip {
	if w.markdown {
		return nil
	}
	c := &w.frames
	pics, sb := w.pictures(), w.storyboarder()
	if c.pics == pics && c.gen == pics.Gen() && c.sb == sb && sameString(c.text, text) {
		return c.chips
	}
	c.text, c.pics, c.gen, c.sb, c.chips = text, pics, pics.Gen(), sb, nil
	body := text[:bodyEndByte(text)]
	dataPath := boardDataPath(text)
	var srcs []string
	for _, f := range findFrames(body) {
		src := boardSrc(f.board, dataPath)
		srcs = append(srcs, src)
		c.chips = append(c.chips, frameChip(f, dataPath, pics, w.frameFileSecs(src)))
	}
	w.watchScriptFrames(srcs) // a save in its program shows (frames.go)
	if more := storyboarderChips(body, sb, pics); len(more) > 0 {
		c.chips = append(c.chips, more...)
		slices.SortFunc(c.chips, func(a, b editor.Chip) int { return a.Start - b.Start })
	}
	return c.chips
}

// frameChip is a frame's chip; fileSecs is how long its file runs, an
// animation's (0: a still).
func frameChip(f frameLine, dataPath string, pics *images.Loader, fileSecs float64) editor.Chip {
	src := boardSrc(f.board, dataPath)
	name := strings.TrimSuffix(path.Base(f.board.File), path.Ext(f.board.File))
	if r := []rune(name); len(r) > 14 {
		name = string(r[:13]) + "…"
	}
	label := name
	if f.board.Duration > 0 {
		label += " · " + strconv.FormatFloat(f.board.Duration, 'f', -1, 64) + "s"
	} else if fileSecs > 0 {
		label += " ▶ " + strconv.FormatFloat(fileSecs, 'f', 1, 64) + "s"
	}
	if f.board.Continued {
		label += " ↻"
	}
	tip := "Storyboard frame: " + f.board.File
	if f.board.Duration > 0 {
		tip += fmt.Sprintf(", %.2f s", f.board.Duration)
	} else if fileSecs > 0 {
		tip += fmt.Sprintf(", animated: %.2f s, as its file says", fileSecs)
	}
	if f.board.Continued {
		tip += ", carries on from the frame before"
	}
	c := editor.Chip{Start: f.start, End: f.end, Tip: tip + " (tap to see it)"}
	if thumb := pics.Thumbnail(src, 160, 90); thumb != nil {
		c.Image, c.ImageCols = thumb, frameThumbCols
		c.Label = strings.Repeat(" ", frameThumbCols) + " " + label
		return c
	}
	c.Label, c.Warn = "▨ "+label, true
	c.Tip = tip + " — " + images.Why(pics.Load(src).Err)
	return c
}

// editorChips are the script editor's chips: its source links and its
// storyboard frames, in order (the same slice for the same text).
func (w *MainWindow) editorChips(text string) []editor.Chip {
	links := w.linkChips(text)
	if inc := w.highlandChips(text); len(inc) > 0 { // Highland's includes and markers (highland_directives.go)
		c := &w.hlChips
		if c.merged == nil || !sameChips(c.links, links) || !sameChips(c.of, inc) {
			c.merged, c.links, c.of = mergeChips(links, inc), links, inc
		}
		links = c.merged
	}
	frames := w.frameChips(text)
	if len(frames) == 0 {
		return links
	}
	c := &w.frames
	if c.merged != nil && sameChips(c.links, links) && sameChips(c.of, frames) {
		return c.merged
	}
	merged := append(slices.Clone(links), frames...)
	slices.SortFunc(merged, func(a, b editor.Chip) int { return a.Start - b.Start })
	c.merged, c.links, c.of = merged, links, frames
	return merged
}

// sameChips: the same slice.
func sameChips(a, b []editor.Chip) bool {
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0])
}

// frameTapped shows a frame's picture large.
func (w *MainWindow) frameTapped(c editor.Chip, _ fyne.Position, _ fyne.Size) {
	text := w.textEditor.Text()
	body := text[:bodyEndByte(text)]
	for _, f := range findFrames(body) {
		if f.start == c.Start {
			w.showFrame(f.board, boardSrc(f.board, boardDataPath(text)), f.start)
			return
		}
	}
	if sb := w.storyboarder(); sb != nil { // Storyboarder's, on a heading
		lines := strings.Split(body, "\n")
		off := 0
		for _, line := range lines {
			if m := sbSceneID.FindStringSubmatchIndex(line); m != nil && off+utf8.RuneCountInString(line[:m[0]]) == c.Start {
				if s := sb.scenes[line[m[4]:m[5]]]; s != nil {
					w.showStoryboarderScene(s)
				}
				return
			}
			off += utf8.RuneCountInString(line) + 1
		}
	}
}

// showFrame shows a frame's picture as large as the window has room for,
// with its note's details; Edit Note puts the cursor in the note (at
// at: a phone's way in, where a tap shows the picture).
func (w *MainWindow) showFrame(b lex.Board, src string, at int) {
	p := w.pictures().Load(src)
	var body fyne.CanvasObject
	if p.OK() {
		img := canvas.NewImageFromImage(p.Image)
		img.FillMode = canvas.ImageFillContain
		img.ScaleMode = canvas.ImageScaleSmooth
		pw, ph := p.Size()
		size := w.fyneWindow.Canvas().Size()
		mw, mh := max(size.Width*0.85-40, 200), max(size.Height*0.7-120, 150)
		s := min(mw/float32(pw), mh/float32(ph), 2)
		img.SetMinSize(fyne.NewSize(float32(pw)*s, float32(ph)*s))
		fyne.SetAccessibleLabel(img, "Storyboard frame "+b.File)
		body = img
	} else {
		body = picturePlaceholder(b.File, src, p.Err)
	}
	about := b.File
	if b.Duration > 0 {
		about += fmt.Sprintf(" · %.2f s", b.Duration)
	}
	if b.Continued {
		about += " · carries on from the frame before"
	}
	info := widget.NewLabel(about)
	info.Wrapping = fyne.TextWrapWord
	var d *dialog.CustomDialog
	edit := widget.NewButton("Edit Note", func() {
		d.Hide()
		w.textEditor.SetCursorOffset(at + 2)
		w.focusMain()
	})
	closeB := widget.NewButton("Close", func() { d.Hide() })
	closeB.Importance = widget.HighImportance
	_, tool := w.frameCommand(b.File)
	draw := widget.NewButton("Edit in "+tool.name, func() { // its program (frames.go)
		d.Hide()
		w.openFrame(src)
	})
	if inBrowser {
		draw.Disable()
		info.SetText(about + "\n" + errFramesDesktop.Error())
	}
	d = dialog.NewCustomWithoutButtons("Storyboard Frame", container.NewBorder(nil, container.NewVBox(info, w.dialogButtons(edit, draw, closeB)), nil, nil, body), w.fyneWindow)
	d.SetOnEscape(d.Hide)
	d.Show()
}

// framePictures are the script's frames' pictures, by their notes'
// names, for a PDF that prints them: those that can be read (a missing
// one prints as a box with its name); never nil.
func (w *MainWindow) framePictures(text string) map[string][]byte {
	out := map[string][]byte{}
	dataPath := boardDataPath(text)
	for _, f := range findFrames(text[:bodyEndByte(text)]) {
		if _, ok := out[f.board.File]; ok {
			continue
		}
		if p := w.pictures().Load(boardSrc(f.board, dataPath)); p.OK() {
			out[f.board.File] = p.Data
		}
	}
	return out
}

// insertFrame writes a frame's note on a line of its own under the line
// the cursor is on (after the frames under it already), in one undo step.
func insertFrame(b *buffer.Buffer, board lex.Board) {
	b.Group(func() {
		text := []rune(b.Text())
		at := b.Cursor()
		for at < len(text) && text[at] != '\n' {
			at++
		}
		for at < len(text) { // after the frames there
			end := at + 1
			for end < len(text) && text[end] != '\n' {
				end++
			}
			t := strings.TrimSpace(string(text[at+1 : end]))
			if _, ok := lex.ParseBoard(strings.TrimSuffix(strings.TrimPrefix(t, "[["), "]]")); !ok || !strings.HasPrefix(t, "[[") {
				break
			}
			at = end
		}
		b.SetCursor(at, false)
		b.Insert("\n[[" + board.String() + "]]")
	})
}

// askFrame is Insert › Storyboard Frame…: a picture chosen, copied into
// the script's picture folder, its note written under the cursor's line.
func (w *MainWindow) askFrame() {
	if w.bundle == nil && (w.currentFile == "" || inBrowser) {
		dialog.ShowError(errNoPictureHome, w.fyneWindow)
		return
	}
	chooseFileIn(w.fyneWindow, frameExtensions, func(picked string) {
		dataPath := boardDataPath(w.textEditor.Text())
		w.keepPicture(picked, w.pictureFolder(dataPath), func(rel string, err error) {
			if err != nil {
				dialog.ShowError(fmt.Errorf("could not add the frame: %v", err), w.fyneWindow)
				return
			}
			file := rel
			if dataPath != "" {
				file = strings.TrimPrefix(rel, strings.Trim(path.Clean(dataPath), "/")+"/")
			}
			w.textEditor.Edit(func(b *buffer.Buffer) { insertFrame(b, lex.Board{File: file}) })
			w.setStatus("Storyboard frame added: " + file + " (Page Setup prints the frames, if asked)")
			w.focusMain()
		})
	})
}
