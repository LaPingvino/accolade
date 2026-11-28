// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2/dialog"
	"github.com/LaPingvino/lexington/lex"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/images"
)

// Storyboard frames drawn in the open drawing programs. A frame is a
// plain file in the script's picture folder (the bundle's assets/, or
// Name.assets/ beside a plain file): an SVG first, else a PNG (any
// picture), Krita's .kra, Synfig's .sif/.sifz or a Lottie .json
// (Glaxnimate); its timing is in its [[board:]] note (storyboard.go).
//
//   - Edit Frame in… opens the frame under the cursor (or the one shown:
//     the frame dialog's Edit in …) in its program: Inkscape for SVG,
//     Krita for PNG and .kra, Glaxnimate for Lottie, Synfig Studio for
//     .sif — found on PATH or as a Flatpak, or as Preferences › Script ›
//     Storyboard Frames says. New Frame makes a blank SVG at the script's
//     aspect ratio ([[aspectRatio: 2.35]], else 16:9), writes its note
//     and opens it.
//   - The frames' files are watched (a stat every 0.7 s, a change acted
//     on once it has settled for a tick): a save shows at once, no
//     import step. A zipped bundle's frame is edited as a working copy
//     in the user's cache folder, written back into the bundle on save.
//   - .kra shows the picture Krita keeps in it (mergedimage.png); .sif
//     and Lottie are drawn by synfig's and glaxnimate's command lines in
//     the background when installed, else a placeholder says so.
//   - An animated frame (Glaxnimate, Synfig) takes its time from its
//     file when its note gives none: Lottie's (op − ip) / fr, Synfig's
//     canvas end-time − begin-time.
//
// The browser has no programs to open nor files to watch: the commands
// say so there.

// frameExtensions are the files a frame may be.
var frameExtensions = append(append([]string{".svg"}, pictureExtensions...), ".kra", ".sif", ".sifz", ".json")

// frameTool is a drawing program a frame's kind opens in.
type frameTool struct {
	name    string   // as shown
	pref    string   // its setting: a command line, "" the found one
	bins    []string // on PATH
	flatpak string   // its Flatpak's id
}

// frameTools by the kind of file they open.
var frameTools = map[string]frameTool{
	"svg":    {"Inkscape", "frame-editor-svg", []string{"inkscape"}, "org.inkscape.Inkscape"},
	"raster": {"Krita", "frame-editor-raster", []string{"krita"}, "org.kde.krita"},
	"lottie": {"Glaxnimate", "frame-editor-lottie", []string{"glaxnimate"}, "org.mattbas.Glaxnimate"},
	"sif":    {"Synfig Studio", "frame-editor-sif", []string{"synfigstudio", "synfig-studio"}, "org.synfig.SynfigStudio"},
}

// frameToolOrder is the tools' order in Preferences.
var frameToolOrder = []string{"svg", "raster", "lottie", "sif"}

// frameToolKind is the tool kind a frame's file opens in.
func frameToolKind(file string) string {
	switch strings.ToLower(path.Ext(file)) {
	case ".svg":
		return "svg"
	case ".json", ".lottie":
		return "lottie"
	case ".sif", ".sifz":
		return "sif"
	}
	return "raster"
}

// The test hooks: finding a program, a Flatpak, starting one.
var (
	lookPath         = exec.LookPath
	flatpakInstalled = func(id string) bool {
		if _, err := exec.LookPath("flatpak"); err != nil {
			return false
		}
		return exec.Command("flatpak", "info", id).Run() == nil
	}
	startProgram = func(argv []string) error {
		cmd := exec.Command(argv[0], argv[1:]...)
		if err := cmd.Start(); err != nil {
			return err
		}
		go cmd.Wait()
		return nil
	}
)

// foundTools are the programs found, by kind (looked for once).
var foundTools struct {
	sync.Mutex
	by map[string][]string // the command line before the file; nil: none
}

// foundTool is the command line a kind's program is started by (the file
// goes after it), and how it was found; nil: not installed.
func foundTool(kind string) (argv []string, how string) {
	foundTools.Lock()
	defer foundTools.Unlock()
	if foundTools.by == nil {
		foundTools.by = map[string][]string{}
	}
	if argv, ok := foundTools.by[kind]; ok {
		return argv, toolHow(argv)
	}
	t := frameTools[kind]
	for _, b := range t.bins {
		if p, err := lookPath(b); err == nil {
			argv = []string{p}
			break
		}
	}
	if argv == nil && t.flatpak != "" && flatpakInstalled(t.flatpak) {
		// the file handed through the document portal, so the sandbox can write it
		argv = []string{"flatpak", "run", "--file-forwarding", t.flatpak, "@@"}
	}
	foundTools.by[kind] = argv
	return argv, toolHow(argv)
}

func toolHow(argv []string) string {
	switch {
	case argv == nil:
		return ""
	case argv[0] == "flatpak":
		return "Flatpak " + argv[3]
	}
	return argv[0]
}

// forgetFoundTools looks for the programs again (a test's hooks changed).
func forgetFoundTools() {
	foundTools.Lock()
	foundTools.by = nil
	foundTools.Unlock()
}

// frameCommand is the command line that opens file in its kind's
// program: the setting's ({file} where it goes, else at the end), else
// the program found; nil: none.
func (w *MainWindow) frameCommand(file string) ([]string, frameTool) {
	kind := frameToolKind(file)
	t := frameTools[kind]
	if w.settings != nil {
		if set := splitCommand(w.settings.GetString(t.pref)); len(set) > 0 {
			out, put := make([]string, 0, len(set)+1), false
			for _, a := range set {
				if strings.Contains(a, "{file}") {
					a, put = strings.ReplaceAll(a, "{file}", file), true
				}
				out = append(out, a)
			}
			if !put {
				out = append(out, file)
			}
			return out, t
		}
	}
	argv, _ := foundTool(kind)
	if argv == nil {
		return nil, t
	}
	out := append(append([]string{}, argv...), file)
	if argv[0] == "flatpak" {
		out = append(out, "@@")
	}
	return out, t
}

// splitCommand is a command line's words: split at spaces, but not
// inside "double quotes" (a program's path with spaces: C:\Program
// Files\Inkscape\bin\inkscape.exe), the whole line when it names a file
// there is.
func splitCommand(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if fileExists(s) {
		return []string{s}
	}
	var out []string
	var cur strings.Builder
	quoted, had := false, false
	for _, r := range s {
		switch {
		case r == '"':
			quoted, had = !quoted, true
		case r == ' ' && !quoted:
			if cur.Len() > 0 || had {
				out = append(out, cur.String())
			}
			cur.Reset()
			had = false
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 || had {
		out = append(out, cur.String())
	}
	return out
}

// errFramesDesktop: the browser opens no programs.
var errFramesDesktop = errors.New("drawing a frame in Inkscape, Krita, Glaxnimate or Synfig needs Accolade on the desktop: " +
	"in the browser, draw it elsewhere and add the picture with Insert › Storyboard Frame…")

// frameOnDisk is where a frame's file (src, as the loader names it) is
// on disk for its program, and for a zipped bundle the entry the working
// copy goes back to ("": edited in place). A frame not there yet (a new
// one) is where it will be.
func (w *MainWindow) frameOnDisk(src string) (abs, entry string, err error) {
	if inBrowser {
		return "", "", errFramesDesktop
	}
	if filepath.IsAbs(filepath.FromSlash(src)) {
		return filepath.FromSlash(src), "", nil
	}
	rel := strings.TrimPrefix(path.Clean(src), "./")
	var root string
	switch b := w.bundle; {
	case b != nil && b.Zipped:
		if _, err := b.readExtra(rel); err != nil && !strings.HasPrefix(rel, "assets/") {
			if _, err2 := b.readExtra("assets/" + rel); err2 == nil {
				rel = "assets/" + rel
			}
		}
		return w.frameWorkingCopy(rel), rel, nil
	case b != nil:
		root = b.Path
	case w.currentFile != "":
		root = filepath.Dir(w.currentFile)
	default:
		return "", "", errNoPictureHome
	}
	abs = filepath.Join(root, filepath.FromSlash(rel))
	if _, err := os.Stat(abs); err != nil && !strings.HasPrefix(rel, "assets/") {
		if alt := filepath.Join(root, "assets", filepath.FromSlash(rel)); fileExists(alt) {
			abs = alt
		}
	}
	return abs, "", nil
}

// frameCacheDir is the user's cache folder for Accolade's frames.
var frameCacheDir = func() string {
	d, err := os.UserCacheDir()
	if err != nil {
		d = os.TempDir()
	}
	return filepath.Join(d, "accolade", "frames")
}

// frameWorkingCopy is the working copy of a zipped bundle's entry.
func (w *MainWindow) frameWorkingCopy(entry string) string {
	sum := sha256.Sum256([]byte(w.bundle.Path))
	return filepath.Join(frameCacheDir(), hex.EncodeToString(sum[:6]), filepath.FromSlash(entry))
}

// openFrame opens a frame's file in its program, watching it for saves.
func (w *MainWindow) openFrame(src string) {
	abs, entry, err := w.frameOnDisk(src)
	if err == nil && entry != "" { // a zipped bundle's: its working copy, as the bundle has it
		var data []byte
		if data, err = w.bundle.readExtra(entry); err == nil {
			if err = os.MkdirAll(filepath.Dir(abs), 0o755); err == nil {
				err = writeFileAtomic(abs, data)
			}
		}
	}
	if err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return
	}
	argv, t := w.frameCommand(abs)
	if argv == nil {
		dialog.ShowInformation("No Program for "+filepath.Base(abs),
			fmt.Sprintf("%s opens %s, and it was not found (on PATH, nor as the Flatpak %s). Install it, or name the program "+
				"in Preferences › Script › Storyboard Frames.", t.name, filepath.Base(abs), t.flatpak), w.fyneWindow)
		return
	}
	if err := startProgram(argv); err != nil {
		dialog.ShowError(fmt.Errorf("could not start %s: %v", t.name, err), w.fyneWindow)
		return
	}
	bundle := ""
	if entry != "" {
		bundle = w.bundle.Path
	}
	w.frameWatch.add(abs, entry, bundle, true)
	w.armFrameWatch()
	w.setStatus(fmt.Sprintf("%s opened in %s: a save shows here", filepath.Base(abs), t.name))
}

// frameUnderCursor is the frame note on the cursor's line.
func (w *MainWindow) frameUnderCursor() (frameLine, bool) {
	text := w.textEditor.Text()
	at := w.textEditor.CursorOffset()
	rs := []rune(text)
	start, end := min(at, len(rs)), min(at, len(rs))
	for start > 0 && rs[start-1] != '\n' {
		start--
	}
	for end < len(rs) && rs[end] != '\n' {
		end++
	}
	for _, f := range findFrames(text[:bodyEndByte(text)]) {
		if f.start >= start && f.end <= end {
			return f, true
		}
	}
	return frameLine{}, false
}

// editFrame is Edit Frame in…: the frame under the cursor in its program.
func (w *MainWindow) editFrame() {
	if inBrowser {
		dialog.ShowInformation("Desktop Only", errFramesDesktop.Error(), w.fyneWindow)
		return
	}
	f, ok := w.frameUnderCursor()
	if !ok {
		dialog.ShowInformation("No Frame Here", "Put the cursor on a frame's line ([[board: …]]), or tap a frame and choose Edit in….", w.fyneWindow)
		return
	}
	w.openFrame(boardSrc(f.board, boardDataPath(w.textEditor.Text())))
}

// boardAspect is the frames' shape: a [[aspectRatio: …]] note's
// (Storyboard Fountain's), else 16:9.
func boardAspect(text string) float64 {
	if i := strings.Index(text, "[[aspectRatio:"); i >= 0 {
		if j := strings.Index(text[i:], "]]"); j > 0 {
			if r, err := strconv.ParseFloat(strings.TrimSpace(text[i+len("[[aspectRatio:"):i+j]), 64); err == nil && r > 0.2 && r < 5 {
				return r
			}
		}
	}
	return 16.0 / 9
}

// blankFrame is a new frame's SVG: 1920 wide at the aspect ratio, a white
// paper layer (locked) under an empty drawing layer.
func blankFrame(aspect float64) []byte {
	h := int(1920/aspect + 0.5)
	return fmt.Appendf(nil, `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:inkscape="http://www.inkscape.org/namespaces/inkscape" xmlns:sodipodi="http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd" width="1920" height="%d" viewBox="0 0 1920 %d">
  <g inkscape:groupmode="layer" inkscape:label="Paper" sodipodi:insensitive="true" id="paper">
    <rect x="0" y="0" width="1920" height="%d" fill="#ffffff"/>
  </g>
  <g inkscape:groupmode="layer" inkscape:label="Drawing" id="drawing"/>
</svg>
`, h, h, h)
}

// newFrame is New Frame: a blank SVG in the picture folder, its note
// under the cursor's line, opened in its program.
func (w *MainWindow) newFrame() {
	if inBrowser {
		dialog.ShowInformation("Desktop Only", errFramesDesktop.Error(), w.fyneWindow)
		return
	}
	if w.bundle == nil && w.currentFile == "" {
		dialog.ShowError(errNoPictureHome, w.fyneWindow)
		return
	}
	text := w.textEditor.Text()
	dataPath := boardDataPath(text)
	name := fmt.Sprintf("frame-%02d.svg", len(findFrames(text[:bodyEndByte(text)]))+1)
	w.keepFiles(w.pictureFolder(dataPath), []bundleFile{{name, blankFrame(boardAspect(text))}}, func(rels []string, err error) {
		if err != nil {
			dialog.ShowError(fmt.Errorf("could not make the frame: %v", err), w.fyneWindow)
			return
		}
		file := rels[0]
		if dataPath != "" {
			file = strings.TrimPrefix(file, strings.Trim(path.Clean(dataPath), "/")+"/")
		}
		w.textEditor.Edit(func(b *buffer.Buffer) { insertFrame(b, lex.Board{File: file}) })
		w.openFrame(rels[0])
	})
}

// frameWatcher is a window's watch on its frames' files.
type frameWatcher struct {
	files map[string]*watchedFrame // by the path on disk
	// secs are the frames' files' animation times as read, for the
	// loader secsOf at its generation secsGen (frameFileSecs)
	secs    map[string]float64
	secsOf  *images.Loader
	secsGen int
	armed   bool
	renders int64 // renderedFrames when looked at last
}

// watchedFrame is a frame's file as last seen.
type watchedFrame struct {
	entry   string // a zipped bundle's entry its working copy goes back to
	bundle  string // and that bundle's path (another document's: not written)
	edited  bool   // opened from here: kept watched when the script no longer names it
	mod     time.Time
	size    int64
	pending bool // changed at the last look: acted on once it is the same at the next
}

// add watches a file as it is now (entry: a zipped bundle's, at
// bundle).
func (fw *frameWatcher) add(abs, entry, bundle string, edited bool) {
	if fw.files == nil {
		fw.files = map[string]*watchedFrame{}
	}
	if old := fw.files[abs]; old != nil {
		old.edited = old.edited || edited
		if entry != "" {
			old.entry, old.bundle = entry, bundle
		}
		return
	}
	f := &watchedFrame{entry: entry, bundle: bundle, edited: edited}
	if st, err := os.Stat(abs); err == nil {
		f.mod, f.size = st.ModTime(), st.Size()
	}
	fw.files[abs] = f
}

// watchScriptFrames watches the files the script's frames name (those
// on disk: not a zipped bundle's, which only change through a working
// copy) beside the ones opened from here.
func (w *MainWindow) watchScriptFrames(srcs []string) {
	if inBrowser || (w.bundle == nil && w.currentFile == "") {
		return
	}
	fw := &w.frameWatch
	want := map[string]bool{}
	if w.bundle == nil || !w.bundle.Zipped {
		for _, src := range srcs {
			if images.IsRemote(src) || strings.HasPrefix(strings.ToLower(src), "data:") {
				continue
			}
			if abs, _, err := w.frameOnDisk(src); err == nil {
				want[abs] = true
				fw.add(abs, "", "", false)
			}
		}
	}
	for abs, f := range fw.files {
		if !f.edited && !want[abs] {
			delete(fw.files, abs)
		}
	}
	if len(srcs) > 0 {
		w.armFrameWatch()
	}
}

// frameWatchTick is how often the frames' files are looked at.
const frameWatchTick = 700 * time.Millisecond

// armFrameWatch looks at the files again in a tick, while there are any
// (or frames being drawn).
func (w *MainWindow) armFrameWatch() {
	fw := &w.frameWatch
	if fw.armed || inBrowser {
		return
	}
	fw.armed = true
	w.after(frameWatchTick, func() {
		fw.armed = false
		w.pollFrames()
		if len(fw.files) > 0 || rendersBusy() {
			w.armFrameWatch()
		}
	})
}

// pollFrames looks at the watched files once: a file changed at the last
// look and the same now (its save done) is shown again; a zipped
// bundle's working copy is written back into it first. A frame drawn by
// its program since the last look shows too.
func (w *MainWindow) pollFrames() {
	fw := &w.frameWatch
	var changed []string
	var back []bundleFile
	for abs, f := range fw.files {
		if f.entry != "" && (w.bundle == nil || w.bundle.Path != f.bundle) {
			delete(fw.files, abs) // another document's working copy now: never written into this one
			continue
		}
		st, err := os.Stat(abs)
		if err != nil {
			continue // (a rename-save: there again at the next look)
		}
		if !st.ModTime().Equal(f.mod) || st.Size() != f.size {
			f.mod, f.size, f.pending = st.ModTime(), st.Size(), true
			continue
		}
		if !f.pending {
			continue
		}
		f.pending = false
		changed = append(changed, filepath.Base(abs))
		if f.entry != "" {
			if data, err := os.ReadFile(abs); err == nil {
				back = append(back, bundleFile{f.entry, data})
			}
		}
	}
	if r := renderedFrames.Load(); r != fw.renders {
		fw.renders = r
		if len(changed) == 0 {
			w.framesChanged()
		}
	}
	if len(changed) == 0 {
		return
	}
	msg := "Frame updated: " + strings.Join(changed, ", ")
	if len(back) == 0 {
		w.framesChanged()
		w.setStatus(msg)
		return
	}
	w.queueBundle(bundleJob{run: func(b *textBundle) (*textBundle, error) {
		if b == nil {
			return nil, errors.New("the script's bundle is gone")
		}
		return saveBundleFiles(b, back, false)
	}, done: func(err error) {
		if err != nil {
			w.setStatus("Could not keep the frame in the bundle: " + err.Error())
			return
		}
		w.framesChanged()
		w.setStatus(msg)
	}})
}

// framesChanged shows the frames' files read again.
func (w *MainWindow) framesChanged() {
	w.pictures().Forget()
	if w.textEditor != nil {
		w.textEditor.Refresh()
	}
}

// frameFileSecs is how long an animated frame's file says it runs (0: a
// still, or not said).
func (w *MainWindow) frameFileSecs(src string) float64 {
	fw, pics := &w.frameWatch, w.pictures()
	if fw.secsOf != pics || fw.secsGen != pics.Gen() || fw.secs == nil {
		fw.secs, fw.secsOf, fw.secsGen = map[string]float64{}, pics, pics.Gen()
	}
	if s, ok := fw.secs[src]; ok {
		return s
	}
	s := 0.0
	if p := pics.Load(src); p.Orig != nil {
		s = animationSecs(src, p.Orig)
	}
	fw.secs[src] = s
	return s
}

// animationSecs is an animation's length in seconds from its file:
// Lottie's (op − ip) / fr, Synfig's canvas end-time − begin-time.
func animationSecs(name string, data []byte) float64 {
	switch images.KindOf(name, data) {
	case images.KindLottie:
		var l struct{ Fr, Ip, Op float64 }
		if json.Unmarshal(data, &l) == nil && l.Fr > 0 && l.Op > l.Ip {
			return (l.Op - l.Ip) / l.Fr
		}
	case images.KindSynfig:
		d := xml.NewDecoder(bytes.NewReader(images.Unzipped(data)))
		for {
			tok, err := d.Token()
			if err != nil {
				return 0
			}
			if se, ok := tok.(xml.StartElement); ok {
				if se.Name.Local != "canvas" {
					return 0
				}
				fps, begin, end := 24.0, "0", "0"
				for _, a := range se.Attr {
					switch a.Name.Local {
					case "fps":
						if f, err := strconv.ParseFloat(a.Value, 64); err == nil && f > 0 {
							fps = f
						}
					case "begin-time":
						begin = a.Value
					case "end-time":
						end = a.Value
					}
				}
				return max(0, synfigTime(end, fps)-synfigTime(begin, fps))
			}
		}
	}
	return 0
}

// synfigTime is a Synfig time ("5s", "2s 12f", "1m 3s", "120" frames …)
// in seconds.
func synfigTime(s string, fps float64) float64 {
	secs := 0.0
	for _, part := range strings.Fields(s) {
		unit := part[len(part)-1]
		num := strings.TrimRight(part, "hmsf")
		v, err := strconv.ParseFloat(num, 64)
		if err != nil {
			continue
		}
		switch unit {
		case 'h':
			secs += v * 3600
		case 'm':
			secs += v * 60
		case 's':
			secs += v
		default: // f, or a bare number: frames
			secs += v / fps
		}
	}
	return secs
}

// The background drawing of Synfig and Lottie frames by their command
// lines (desktop): by the file's bytes, done once.
var (
	renders struct {
		sync.Mutex
		done map[[32]byte]renderResult
		busy map[[32]byte]bool
	}
	renderedFrames atomic.Int64 // a frame drawn: the windows show it (pollFrames)
	// runRender runs a renderer's command line (a test's stands in)
	runRender = func(argv []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
		cmd.Env = append(os.Environ(), "QT_QPA_PLATFORM=offscreen") // glaxnimate is a Qt program
		return cmd.Run()
	}
)

type renderResult struct {
	img image.Image
	err error
}

// frameRenderers are the command lines that draw a frame's first frame
// as a PNG (glaxnimate checked with 0.6's --help; synfig's flags as its
// manual has them).
var frameRenderers = map[images.Kind]struct {
	bin, ext string
	args     func(in, out string) []string
}{
	images.KindLottie: {"glaxnimate", ".json", func(in, out string) []string { return []string{in, "--render", out, "--frame", "0"} }},
	images.KindSynfig: {"synfig", ".sif", func(in, out string) []string { return []string{in, "-t", "png", "-o", out, "--time", "0"} }},
}

func init() {
	if !inBrowser {
		images.Render = renderFrame
	}
}

func rendersBusy() bool {
	renders.Lock()
	defer renders.Unlock()
	return len(renders.busy) > 0
}

// renderFrame is images.Render: a drawing done before, else
// ErrRendering while it is drawn in the background (ErrNoRenderer
// without the program).
func renderFrame(kind images.Kind, name string, data []byte) (image.Image, error) {
	r, ok := frameRenderers[kind]
	if !ok {
		return nil, images.ErrFormat
	}
	key := sha256.Sum256(data)
	renders.Lock()
	defer renders.Unlock()
	if res, ok := renders.done[key]; ok {
		return res.img, res.err
	}
	if renders.busy[key] {
		return nil, images.ErrRendering
	}
	bin, err := lookPath(r.bin)
	if err != nil {
		return nil, images.ErrNoRenderer
	}
	if renders.busy == nil {
		renders.busy, renders.done = map[[32]byte]bool{}, map[[32]byte]renderResult{}
	}
	renders.busy[key] = true
	ext := strings.ToLower(path.Ext(name))
	if ext == "" {
		ext = r.ext
	}
	go func() {
		res := renderResult{err: images.ErrFormat}
		os.MkdirAll(frameCacheDir(), 0o755)
		if dir, err := os.MkdirTemp(frameCacheDir(), ".render-"); err == nil {
			in, out := filepath.Join(dir, "in"+ext), filepath.Join(dir, "out.png")
			if os.WriteFile(in, data, 0o644) == nil && runRender(append([]string{bin}, r.args(in, out)...)) == nil {
				if f, err := os.Open(out); err == nil {
					if img, err := png.Decode(f); err == nil {
						res = renderResult{img: img}
					}
					f.Close()
				}
			}
			os.RemoveAll(dir)
		}
		renders.Lock()
		renders.done[key] = res
		delete(renders.busy, key)
		renders.Unlock()
		renderedFrames.Add(1)
	}()
	return nil, images.ErrRendering
}
