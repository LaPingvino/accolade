package main

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2/dialog"
	"github.com/LaPingvino/lexington/lex"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/images"
)

// Import Storyboarder Boards…: one way, Storyboarder's boards (its
// .storyboarder JSON, storyboarder.go) made Accolade's own — each board
// a frame file in the script's picture folder, its layers flattened to
// one PNG on white paper (each at its opacity; the poster frame when no
// layer's file is there), and a [[board:]] note with its time (its own,
// else the project's default) and "continued" for a board that is not
// a new shot. The boards of a scene in storyboards/ go under its
// heading (found by its #n-XXXXX# id); a .storyboarder chosen goes under
// the cursor's line. Storyboarder's files stay as they are: reading
// them goes on as before, nothing is written back.
//
// Layered SVG or .kra is not made: oksvg draws no embedded pictures (a
// layered SVG would show blank here), and a .kra's layers are Krita's
// own tiles, not PNG.

// sbImportBoard is a board as the importer reads it: storyboarder.go's
// fields and its layers.
type sbImportBoard struct {
	sbBoard
	NewShot bool `json:"newShot"`
	Layers  map[string]struct {
		URL     string   `json:"url"`
		Opacity *float64 `json:"opacity"`
	} `json:"layers"`
}

// sbLayerOrder is Storyboarder's layers, bottom first (its layer
// names; the board's url is its main layer).
var sbLayerOrder = []string{"reference", "main", "fill", "tone", "pencil", "ink", "notes"}

// sbImported is a scene's (or a project's) boards made frames.
type sbImported struct {
	sceneID string // its heading's id; "": under the cursor
	files   []bundleFile
	boards  []lex.Board
}

// readStoryboarderProject reads a .storyboarder file's boards as frames
// (named after the file); sceneID is the heading they go under.
func readStoryboarderProject(file, sceneID string) (sbImported, error) {
	out := sbImported{sceneID: sceneID}
	data, err := os.ReadFile(file)
	if err != nil {
		return out, err
	}
	var f struct {
		DefaultBoardTiming float64         `json:"defaultBoardTiming"`
		Boards             []sbImportBoard `json:"boards"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return out, fmt.Errorf("%s: not a Storyboarder project (%v)", filepath.Base(file), err)
	}
	if f.DefaultBoardTiming <= 0 {
		f.DefaultBoardTiming = 2000
	}
	dir := filepath.Join(filepath.Dir(file), "images")
	stem := pictureName(strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)))
	for i, b := range f.Boards {
		img := flattenBoard(dir, b)
		if img == nil {
			continue
		}
		name := fmt.Sprintf("%s-%s.png", stem, pictureName(cmp.Or(b.Shot, fmt.Sprint(b.Number), fmt.Sprint(i+1))))
		ms := b.Duration
		if ms <= 0 {
			ms = f.DefaultBoardTiming
		}
		var png bytes.Buffer
		if err := pngEncode(&png, img); err != nil {
			return out, err
		}
		out.files = append(out.files, bundleFile{name, png.Bytes()})
		out.boards = append(out.boards, lex.Board{Duration: float64(int(ms/10+0.5)) / 100, Continued: i > 0 && !b.NewShot})
	}
	return out, nil
}

var pngEncode = png.Encode

// flattenBoard is a board's layers drawn on white paper, bottom first;
// its poster frame (or thumbnail) when none is there; nil: no picture.
func flattenBoard(dir string, b sbImportBoard) image.Image {
	type layer struct {
		file    string
		opacity float64
	}
	var layers []layer
	names := map[string]bool{}
	for n := range b.Layers {
		names[n] = true
	}
	order := slices.Clone(sbLayerOrder)
	var rest []string
	for n := range names {
		if !slices.Contains(order, n) {
			rest = append(rest, n)
		}
	}
	slices.Sort(rest)
	for _, n := range append(order, rest...) {
		if l, ok := b.Layers[n]; ok && l.URL != "" {
			op := 1.0
			if l.Opacity != nil {
				op = *l.Opacity
			}
			layers = append(layers, layer{l.URL, op})
		} else if n == "fill" && !names["main"] && b.URL != "" {
			layers = append(layers, layer{b.URL, 1}) // the board's url: its main layer
		}
	}
	var pics []image.Image
	var ops []float64
	for _, l := range layers {
		if data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(l.file))); err == nil {
			if img, err := images.Decode(data, l.file); err == nil {
				pics, ops = append(pics, img), append(ops, l.opacity)
			}
		}
	}
	if len(pics) == 0 {
		base := strings.TrimSuffix(b.URL, ".png")
		for _, n := range []string{base + "-posterframe.jpg", base + "-thumbnail.png"} {
			if data, err := os.ReadFile(filepath.Join(dir, n)); err == nil {
				if img, err := images.Decode(data, n); err == nil {
					return img
				}
			}
		}
		return nil
	}
	bounds := pics[0].Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(out, out.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	for i, p := range pics {
		mask := image.NewUniform(color.Alpha{A: uint8(max(0, min(1, ops[i]))*255 + 0.5)})
		draw.DrawMask(out, out.Bounds(), p, p.Bounds().Min, mask, image.Point{}, draw.Over)
	}
	return out
}

// importStoryboarder is Import Storyboarder Boards…: the scenes' boards
// in storyboards/ under their headings, else a .storyboarder chosen.
func (w *MainWindow) importStoryboarder() {
	if inBrowser {
		dialog.ShowInformation("Desktop Only", "Importing Storyboarder's boards reads its project folder, so it needs Accolade on the desktop.", w.fyneWindow)
		return
	}
	if w.bundle == nil && w.currentFile == "" {
		dialog.ShowError(errNoPictureHome, w.fyneWindow)
		return
	}
	if sb := w.storyboarder(); sb != nil {
		var all []sbImported
		for _, line := range strings.Split(w.textEditor.Text(), "\n") {
			if m := sbSceneID.FindStringSubmatch(line); m != nil {
				if s := sb.scenes[m[2]]; s != nil {
					folder := filepath.Join(filepath.Dir(w.currentFile), filepath.FromSlash(s.folder))
					imp, err := readStoryboarderProject(filepath.Join(folder, filepath.Base(folder)+".storyboarder"), m[2])
					if err == nil && len(imp.boards) > 0 {
						all = append(all, imp)
					}
				}
			}
		}
		if len(all) > 0 {
			w.adoptBoards(all)
			return
		}
	}
	chooseFileIn(w.fyneWindow, []string{".storyboarder"}, func(picked string) {
		imp, err := readStoryboarderProject(picked, "")
		if err != nil {
			dialog.ShowError(err, w.fyneWindow)
			return
		}
		if len(imp.boards) == 0 {
			dialog.ShowInformation("No Boards", filepath.Base(picked)+" has no boards with a picture.", w.fyneWindow)
			return
		}
		w.adoptBoards([]sbImported{imp})
	})
}

// adoptBoards writes the boards' frame files (one write) and their notes
// (one undo step): a scene's under its heading, the others under the
// cursor's line.
func (w *MainWindow) adoptBoards(all []sbImported) {
	var files []bundleFile
	for _, imp := range all {
		files = append(files, imp.files...)
	}
	dataPath := boardDataPath(w.textEditor.Text())
	w.keepFiles(w.pictureFolder(dataPath), files, func(rels []string, err error) {
		if err != nil {
			dialog.ShowError(fmt.Errorf("could not import the boards: %v", err), w.fyneWindow)
			return
		}
		k := 0
		for i := range all {
			for j := range all[i].boards {
				file := rels[k]
				if dataPath != "" {
					file = strings.TrimPrefix(file, strings.Trim(path.Clean(dataPath), "/")+"/")
				}
				all[i].boards[j].File = file
				k++
			}
		}
		w.textEditor.Edit(func(b *buffer.Buffer) { insertBoards(b, all) })
		w.setStatus(fmt.Sprintf("Storyboarder's boards imported: %s, as frames", plural(len(files), "board")))
		w.focusMain()
	})
}

// insertBoards writes the boards' notes, in one undo step: a scene's on
// the lines under its heading, the rest under the cursor's line.
func insertBoards(b *buffer.Buffer, all []sbImported) {
	notes := func(bs []lex.Board) string {
		var s strings.Builder
		for _, bd := range bs {
			s.WriteString("\n[[" + bd.String() + "]]")
		}
		return s.String()
	}
	text := b.Text()
	type put struct {
		at   int
		text string
	}
	var puts []put
	heads := map[string]int{} // a scene id's heading line's end, in runes
	off := 0
	for _, line := range strings.Split(text, "\n") {
		off += utf8.RuneCountInString(line)
		if m := sbSceneID.FindStringSubmatch(line); m != nil {
			if _, ok := heads[m[2]]; !ok {
				heads[m[2]] = off
			}
		}
		off++
	}
	rs := []rune(text)
	cur := b.Cursor()
	for cur < len(rs) && rs[cur] != '\n' {
		cur++
	}
	for _, imp := range all {
		at, ok := heads[imp.sceneID]
		if !ok {
			at = cur
		}
		puts = append(puts, put{at, notes(imp.boards)})
	}
	slices.SortStableFunc(puts, func(a, b put) int { return b.at - a.at }) // from the end: the offsets before stay
	b.Group(func() {
		for _, p := range puts {
			b.SetCursor(p.at, false)
			b.Insert(p.text)
		}
	})
}
