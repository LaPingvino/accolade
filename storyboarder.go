package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/images"
)

// Storyboarder's boards (wonderunit/storyboarder; not OSI: read here,
// never written). Opening a Fountain script it numbers each heading
// #<n>-<id># (a five-character id; valid Fountain, kept as it is,
// docs/PRODUCTION.md) and keeps a scene's boards beside the script in
// storyboards/Scene-<n>-<slug>-<id>/Scene-….storyboarder (JSON: its
// boards, each with its number, shot, duration in ms, dialogue, action,
// notes and url board-<n>-<uid>.png), their pictures in images/ —
// <url>-posterframe.jpg the composed board, -thumbnail.png a small one.
// A scene's folder is found by the id, its folder name's last part, as
// Storyboarder finds it. Read from the source (src/js/window/
// main-window.js loadScene, src/js/models/board.js), without a sample
// project to try it on.
//
// A heading with boards shows them as a chip on its scene number: the
// first board's thumbnail and how many; a tap shows them all.

// sbBoard is a board as Storyboarder keeps it.
type sbBoard struct {
	UID      string  `json:"uid"`
	URL      string  `json:"url"`
	Number   int     `json:"number"`
	Shot     string  `json:"shot"`
	Duration float64 `json:"duration"` // ms; 0: the scene's default
	Dialogue string  `json:"dialogue"`
	Action   string  `json:"action"`
	Notes    string  `json:"notes"`
}

// sbScene is a scene's boards: the folder (relative to the script's)
// and its file's boards.
type sbScene struct {
	folder  string
	timing  float64 // the default board's time, ms
	boards  []sbBoard
	picture []string // each board's picture, relative to the script's folder
}

// sbProject is the boards found beside a script, by scene id.
type sbProject struct {
	dir    string
	stamp  time.Time
	read   time.Time
	scenes map[string]*sbScene
}

// sbSceneID is a Storyboarder scene number in a heading: #<n>-<id>#.
var sbSceneID = regexp.MustCompile(`#(\d+)-([0-9A-Za-z]{5})#\s*$`)

// storyboarder is the boards beside the window's plain script (nil: none,
// or not a plain file on the desktop), read again when the folder changed
// (looked at every few seconds at most).
func (w *MainWindow) storyboarder() *sbProject {
	if w.bundle != nil || w.currentFile == "" || inBrowser {
		return nil
	}
	dir := filepath.Join(filepath.Dir(w.currentFile), "storyboards")
	p := w.sbBoards
	if p != nil && p.dir == dir && time.Since(p.read) < 3*time.Second {
		return p.orNil()
	}
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		w.sbBoards = &sbProject{dir: dir, read: time.Now()}
		return nil
	}
	if p != nil && p.dir == dir && p.stamp.Equal(newestIn(dir)) {
		p.read = time.Now()
		return p.orNil()
	}
	w.sbBoards = readStoryboarder(dir)
	return w.sbBoards.orNil()
}

func (p *sbProject) orNil() *sbProject {
	if p == nil || len(p.scenes) == 0 {
		return nil
	}
	return p
}

// newestIn is the latest change in dir and its scene folders.
func newestIn(dir string) time.Time {
	var t time.Time
	if st, err := os.Stat(dir); err == nil {
		t = st.ModTime()
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if info, err := e.Info(); err == nil && info.ModTime().After(t) {
			t = info.ModTime()
		}
		if e.IsDir() {
			if st, err := os.Stat(filepath.Join(dir, e.Name(), e.Name()+".storyboarder")); err == nil && st.ModTime().After(t) {
				t = st.ModTime()
			}
		}
	}
	return t
}

// readStoryboarder reads the scenes' boards in a storyboards/ folder.
func readStoryboarder(dir string) *sbProject {
	p := &sbProject{dir: dir, stamp: newestIn(dir), read: time.Now(), scenes: map[string]*sbScene{}}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "Scene-") {
			continue
		}
		parts := strings.Split(e.Name(), "-")
		id := parts[len(parts)-1]
		data, err := os.ReadFile(filepath.Join(dir, e.Name(), e.Name()+".storyboarder"))
		if err != nil {
			continue
		}
		var f struct {
			DefaultBoardTiming float64   `json:"defaultBoardTiming"`
			Boards             []sbBoard `json:"boards"`
		}
		if json.Unmarshal(data, &f) != nil {
			continue
		}
		s := &sbScene{folder: "storyboards/" + e.Name(), timing: f.DefaultBoardTiming, boards: f.Boards}
		for _, b := range f.Boards {
			s.picture = append(s.picture, sbPicture(filepath.Join(dir, e.Name(), "images"), s.folder+"/images", b.URL))
		}
		p.scenes[id] = s
	}
	return p
}

// sbPicture is a board's picture: the composed board (its poster frame),
// else its thumbnail, else the file its url names.
func sbPicture(dir, rel, url string) string {
	base := strings.TrimSuffix(url, ".png")
	for _, name := range []string{base + "-posterframe.jpg", base + "-thumbnail.png", url} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return path.Join(rel, name)
		}
	}
	return path.Join(rel, base+"-posterframe.jpg")
}

// storyboarderChips are the chips of the headings with Storyboarder's
// boards: on the scene number, the first board's thumbnail and the count.
func storyboarderChips(body string, p *sbProject, pics *images.Loader) []editor.Chip {
	if p == nil || !strings.Contains(body, "#") {
		return nil
	}
	var out []editor.Chip
	off := 0
	for len(body) > 0 {
		line := body
		if i := strings.IndexByte(body, '\n'); i >= 0 {
			line, body = body[:i], body[i+1:]
		} else {
			body = ""
		}
		if m := sbSceneID.FindStringSubmatchIndex(line); m != nil {
			id := line[m[4]:m[5]]
			if s := p.scenes[id]; s != nil && len(s.boards) > 0 {
				start := off + utf8.RuneCountInString(line[:m[0]])
				end := start + utf8.RuneCountInString(strings.TrimRight(line[m[0]:], " \t"))
				label := fmt.Sprintf("#%s · %s", line[m[2]:m[3]], plural(len(s.boards), "board"))
				c := editor.Chip{Start: start, End: end, Label: label,
					Tip: fmt.Sprintf("Storyboarder: %s in %s (tap to see them; Storyboarder's, read only)", plural(len(s.boards), "board"), s.folder)}
				if thumb := pics.Thumbnail(s.picture[0], 160, 90); thumb != nil {
					c.Image, c.ImageCols, c.Label = thumb, frameThumbCols, strings.Repeat(" ", frameThumbCols)+" "+label
				}
				out = append(out, c)
			}
		}
		off += utf8.RuneCountInString(line) + 1
	}
	return out
}

// showStoryboarderScene shows a scene's boards, in order, each with its
// shot, time and text.
func (w *MainWindow) showStoryboarderScene(s *sbScene) {
	var cells []fyne.CanvasObject
	for i, b := range s.boards {
		var pic fyne.CanvasObject
		if p := w.pictures().Load(s.picture[i]); p.OK() {
			img := canvas.NewImageFromImage(images.Fit(p.Image, 480, 270))
			img.FillMode = canvas.ImageFillContain
			img.SetMinSize(fyne.NewSize(240, 135))
			pic = img
		} else {
			pic = picturePlaceholder("Board "+b.Shot, s.picture[i], p.Err)
		}
		d := b.Duration
		if d == 0 {
			d = s.timing
		}
		about := fmt.Sprintf("%s · %.1f s", cmpOr(b.Shot, fmt.Sprint(b.Number)), d/1000)
		for _, t := range []string{b.Action, b.Dialogue, b.Notes} {
			if t = strings.TrimSpace(t); t != "" {
				about += "\n" + t
			}
		}
		l := widget.NewLabel(about)
		l.Wrapping = fyne.TextWrapWord
		cells = append(cells, container.NewBorder(nil, l, nil, nil, pic))
	}
	grid := container.New(layout.NewGridWrapLayout(fyne.NewSize(250, 230)), cells...)
	d := dialog.NewCustom("Storyboarder: "+s.folder, "Close", container.NewVScroll(grid), w.fyneWindow)
	d.Resize(fitDialog(w.fyneWindow, 800, 600))
	d.Show()
}
