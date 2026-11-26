package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/LaPingvino/accolade/internal/images"
)

const lottieJSON = `{"v":"5.7.0","fr":24,"ip":0,"op":60,"w":160,"h":90,"layers":[]}`

// fakeTools stands in for the programs: found are the binaries on PATH,
// flatpaks the Flatpaks installed; started gets each command line run.
func fakeTools(t *testing.T, found, flatpaks []string) *[][]string {
	t.Helper()
	oldLook, oldFlat, oldStart := lookPath, flatpakInstalled, startProgram
	t.Cleanup(func() { lookPath, flatpakInstalled, startProgram = oldLook, oldFlat, oldStart; forgetFoundTools() })
	var started [][]string
	lookPath = func(b string) (string, error) {
		if slices.Contains(found, b) {
			return "/usr/bin/" + b, nil
		}
		return "", errors.New("not found")
	}
	flatpakInstalled = func(id string) bool { return slices.Contains(flatpaks, id) }
	startProgram = func(argv []string) error { started = append(started, argv); return nil }
	forgetFoundTools()
	return &started
}

// krita is a .kra holding pic as its mergedimage.png.
func krita(t *testing.T, pic []byte) []byte {
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	f, _ := z.Create("mimetype")
	f.Write([]byte("application/x-krita"))
	f, _ = z.Create("mergedimage.png")
	f.Write(pic)
	z.Close()
	return b.Bytes()
}

// The drawing programs' files show as pictures: SVG drawn, .kra by its
// merged image, Lottie by its renderer (here a stand-in), Synfig without
// one says so; an animation's time is read from its file.
func TestFrameFileKinds(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.svg"), blankFrame(2.35), 0o644)
	os.WriteFile(filepath.Join(dir, "b.kra"), krita(t, testPicture(t, 64, 36)), 0o644)
	os.WriteFile(filepath.Join(dir, "c.json"), []byte(lottieJSON), 0o644)
	os.WriteFile(filepath.Join(dir, "d.sif"), []byte(`<?xml version="1.0"?><canvas version="1.2" width="480" height="270" fps="24" begin-time="0f" end-time="2s 12f"></canvas>`), 0o644)
	old := images.Render
	t.Cleanup(func() { images.Render = old })
	images.Render = func(k images.Kind, _ string, _ []byte) (image.Image, error) {
		if k == images.KindLottie {
			return image.NewNRGBA(image.Rect(0, 0, 16, 9)), nil
		}
		return nil, images.ErrNoRenderer
	}
	l := images.New(images.Dir(dir))
	for src, w := range map[string]int{"a.svg": 1920, "b.kra": 64, "c.json": 16} {
		p := l.Load(src)
		if gw, _ := p.Size(); !p.OK() || gw != w {
			t.Errorf("%s: %v, %d wide", src, p.Err, gw)
		}
		if _, err := images.Decode(p.Data, "x.png"); err != nil {
			t.Errorf("%s: its Data is not a PNG", src)
		}
	}
	if gw, gh := l.Load("a.svg").Size(); gw != 1920 || gh != 817 {
		t.Errorf("the blank frame at 2.35:1 is %d×%d", gw, gh)
	}
	if p := l.Load("d.sif"); !errors.Is(p.Err, images.ErrNoRenderer) {
		t.Errorf("a .sif without synfig: %v", p.Err)
	}
	for name, want := range map[string]float64{"c.json": 2.5, "d.sif": 2.5, "a.svg": 0} {
		if got := animationSecs(name, l.Load(name).Orig); math.Abs(got-want) > 1e-9 {
			t.Errorf("%s runs %v s, want %v", name, got, want)
		}
	}
	if synfigTime("1m 3s", 24) != 63 || synfigTime("48", 24) != 2 {
		t.Error("synfig times")
	}
}

// Edit Frame in… opens the frame on the cursor's line in its kind's
// program: found on PATH, else the Flatpak (its file forwarded), else
// as the setting says ({file} placed); none found says so.
func TestEditFrameInProgram(t *testing.T) {
	started := fakeTools(t, []string{"inkscape"}, []string{"org.kde.krita"})
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.svg"), blankFrame(16.0/9), 0o644)
	os.WriteFile(filepath.Join(dir, "b.png"), testPicture(t, 32, 18), 0o644)
	script := filepath.Join(dir, "s.fountain")
	os.WriteFile(script, []byte("INT. A - DAY\n\nShe runs.\n[[board: a.svg]]\n\nShe stops.\n[[board: b.png]]\n"), 0o644)
	w := newTestWindow(t, "")
	if err := w.LoadFile(script); err != nil {
		t.Fatal(err)
	}
	text := w.textEditor.Text()
	w.textEditor.SetCursorOffset(len([]rune(text[:strings.Index(text, "[[board: a")])) + 3)
	w.editFrame()
	w.textEditor.SetCursorOffset(len([]rune(text[:strings.Index(text, "[[board: b")])))
	w.editFrame()
	w.settings.SetString("frame-editor-svg", "mypaint --open={file} --new-window")
	t.Cleanup(func() { w.settings.SetString("frame-editor-svg", "") })
	w.textEditor.SetCursorOffset(len([]rune(text[:strings.Index(text, "[[board: a")])))
	w.editFrame()
	want := [][]string{
		{"/usr/bin/inkscape", filepath.Join(dir, "a.svg")},
		{"flatpak", "run", "--file-forwarding", "org.kde.krita", "@@", filepath.Join(dir, "b.png"), "@@"},
		{"mypaint", "--open=" + filepath.Join(dir, "a.svg"), "--new-window"},
	}
	if !slices.EqualFunc(*started, want, slices.Equal) {
		t.Errorf("started %q,\nwant %q", *started, want)
	}
	if len(w.frameWatch.files) != 2 {
		t.Errorf("watching %d files", len(w.frameWatch.files))
	}
	if argv, tool := w.frameCommand("x.sif"); argv != nil || tool.name != "Synfig Studio" {
		t.Errorf("a .sif with no Synfig: %q", argv)
	}
}

// New Frame writes a blank SVG at the script's shape into its picture
// folder, its note under the line, and opens it.
func TestNewFrame(t *testing.T) {
	started := fakeTools(t, []string{"inkscape"}, nil)
	dir := t.TempDir()
	script := filepath.Join(dir, "s.fountain")
	os.WriteFile(script, []byte("[[aspectRatio: 2.35]]\n\nINT. A - DAY\n\nShe runs.\n"), 0o644)
	w := newTestWindow(t, "")
	if err := w.LoadFile(script); err != nil {
		t.Fatal(err)
	}
	w.textEditor.SetCursorOffset(len([]rune(w.textEditor.Text())) - 3)
	w.newFrame()
	file := filepath.Join(dir, "s.assets", "frame-01.svg")
	data, err := os.ReadFile(file)
	if err != nil || !strings.Contains(string(data), `viewBox="0 0 1920 817"`) {
		t.Fatalf("the frame: %v %s", err, data)
	}
	if !strings.Contains(w.textEditor.Text(), "She runs.\n[[board: s.assets/frame-01.svg]]") {
		t.Errorf("the note: %q", w.textEditor.Text())
	}
	if len(*started) != 1 || (*started)[0][1] != file {
		t.Errorf("started %q", *started)
	}
}

// A frame's file saved by its program shows once its save has settled
// (debounced: the look after the change), with no import step.
func TestFrameWatchRefreshes(t *testing.T) {
	dir := t.TempDir()
	pic := filepath.Join(dir, "a.png")
	os.WriteFile(pic, testPicture(t, 32, 18), 0o644)
	script := filepath.Join(dir, "s.fountain")
	os.WriteFile(script, []byte("INT. A - DAY\n\nShe runs.\n[[board: a.png]]\n"), 0o644)
	w := newTestWindow(t, "")
	if err := w.LoadFile(script); err != nil {
		t.Fatal(err)
	}
	w.editorChips(w.textEditor.Text())
	if _, ok := w.frameWatch.files[pic]; !ok {
		t.Fatalf("not watched: %v", w.frameWatch.files)
	}
	gen := w.pictures().Gen()
	os.WriteFile(pic, testPicture(t, 64, 36), 0o644)
	os.Chtimes(pic, time.Now().Add(time.Second), time.Now().Add(time.Second))
	w.pollFrames()
	if w.pictures().Gen() != gen {
		t.Error("shown while the save may still be going on")
	}
	w.pollFrames()
	if w.pictures().Gen() == gen {
		t.Fatal("not shown after the save settled")
	}
	if pw, _ := w.pictures().Load("a.png").Size(); pw != 64 {
		t.Errorf("the picture as before: %d wide", pw)
	}
}

// A zipped bundle's frame is edited as a working copy, written back into
// the bundle when saved.
func TestFrameWorkingCopyInZip(t *testing.T) {
	started := fakeTools(t, []string{"krita"}, nil)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	p := filepath.Join(t.TempDir(), "Script.accolade")
	items := append(bundleItems(""), zipItem{name: "assets/a.png", data: string(testPicture(t, 32, 18)), stored: true})
	for i := range items {
		if items[i].name == "text.fountain" {
			items[i].data = "INT. A - DAY\n\nShe runs.\n[[board: assets/a.png]]\n"
		}
	}
	os.WriteFile(p, makeZip(t, items), 0o644)
	w := newTestWindow(t, "")
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if w.bundle == nil || !w.bundle.Zipped {
		t.Fatal("not opened as a zipped bundle")
	}
	w.openFrame("assets/a.png")
	if len(*started) != 1 {
		t.Fatalf("started %q", *started)
	}
	copyPath := (*started)[0][1]
	if !strings.HasPrefix(copyPath, frameCacheDir()) {
		t.Errorf("edited in place: %s", copyPath)
	}
	os.WriteFile(copyPath, testPicture(t, 64, 36), 0o644)
	os.Chtimes(copyPath, time.Now().Add(time.Second), time.Now().Add(time.Second))
	w.pollFrames()
	w.pollFrames()
	waitFor(t, "the bundle written", func() bool { return !w.bundleBusy() })
	data, _ := w.bundle.readExtra("assets/a.png")
	if img, err := images.Decode(data, "a.png"); err != nil || img.Bounds().Dx() != 64 {
		t.Errorf("the bundle's frame: %v", err)
	}
}

// The board timeline exports as OTIO: a clip per frame with its picture
// and time (the note's, the file's, else 2 s), a marker per scene.
func TestBoardTimelineOTIO(t *testing.T) {
	text := "INT. HARBOUR - DAWN\n\nThe boats come in.\n[[board: a.png, duration: 1.50]]\n\nANNA\nThere he is.\n[[board: c.json]]\n\nEXT. SEA - DAY\n\nWaves.\n[[board: b.svg]]\n"
	s := readTableWith(text, nil, nil, nil, func(src string) float64 {
		if src == "c.json" {
			return 2.5
		}
		return 0
	})
	data, n := boardTimeline(s, "Harbour", func(src string) string { return "file:///x/" + src })
	if n != 3 {
		t.Fatalf("%d clips", n)
	}
	var tl struct {
		Schema string `json:"OTIO_SCHEMA"`
		Tracks struct {
			Children []struct {
				Kind     string `json:"kind"`
				Children []struct {
					Schema string `json:"OTIO_SCHEMA"`
					Range  struct {
						Duration struct{ Value, Rate float64 } `json:"duration"`
					} `json:"source_range"`
					Refs map[string]struct {
						URL string `json:"target_url"`
					} `json:"media_references"`
					Meta struct {
						Accolade map[string]any `json:"accolade"`
					} `json:"metadata"`
				} `json:"children"`
				Markers []struct {
					Name  string `json:"name"`
					Range struct {
						Start struct{ Value float64 } `json:"start_time"`
					} `json:"marked_range"`
				} `json:"markers"`
			} `json:"children"`
		} `json:"tracks"`
	}
	if err := json.Unmarshal(data, &tl); err != nil {
		t.Fatal(err)
	}
	tr := tl.Tracks.Children[0]
	if tl.Schema != "Timeline.1" || tr.Kind != "Video" || len(tr.Children) != 3 {
		t.Fatalf("%s", data)
	}
	var durs []float64
	for _, c := range tr.Children {
		durs = append(durs, c.Range.Duration.Value)
	}
	if !slices.Equal(durs, []float64{36, 60, 48}) || tr.Children[0].Refs["DEFAULT_MEDIA"].URL != "file:///x/a.png" || tr.Children[0].Schema != "Clip.2" {
		t.Errorf("clips: %v, %+v", durs, tr.Children[0])
	}
	if !strings.Contains(tr.Children[1].Meta.Accolade["text"].(string), "There he is.") {
		t.Errorf("the second clip's lines: %v", tr.Children[1].Meta.Accolade)
	}
	if len(tr.Markers) != 2 || tr.Markers[1].Range.Start.Value != 96 || !strings.Contains(tr.Markers[1].Name, "EXT. SEA - DAY") {
		t.Errorf("markers: %+v", tr.Markers)
	}
}

// Storyboarder's boards become frames: layers flattened on white (the
// main layer by the board's url), timings and new shots in the notes,
// under the scene's heading.
func TestImportStoryboarder(t *testing.T) {
	dir := t.TempDir()
	scene := "Scene-1-HARBOUR-DAWN-x2k9q"
	imgs := filepath.Join(dir, "storyboards", scene, "images")
	os.MkdirAll(imgs, 0o755)
	half := 0.5
	sb, _ := json.Marshal(map[string]any{"version": "2.0", "aspectRatio": 1.7777, "defaultBoardTiming": 2000,
		"boards": []map[string]any{
			{"uid": "AAAAA", "url": "board-1-AAAAA.png", "number": 1, "shot": "1A", "newShot": true,
				"layers": map[string]any{"reference": map[string]any{"url": "board-1-AAAAA-reference.png", "opacity": half}}},
			{"uid": "BBBBB", "url": "board-2-BBBBB.png", "number": 2, "shot": "1B", "duration": 1500},
		}})
	os.WriteFile(filepath.Join(dir, "storyboards", scene, scene+".storyboarder"), sb, 0o644)
	clear := image.NewNRGBA(image.Rect(0, 0, 40, 20)) // a transparent main layer
	var b bytes.Buffer
	pngEncode(&b, clear)
	os.WriteFile(filepath.Join(imgs, "board-1-AAAAA.png"), b.Bytes(), 0o644)
	os.WriteFile(filepath.Join(imgs, "board-1-AAAAA-reference.png"), testPicture(t, 40, 20), 0o644)
	os.WriteFile(filepath.Join(imgs, "board-2-BBBBB-posterframe.jpg"), testPicture(t, 40, 20), 0o644) // (a PNG, read by its bytes)
	script := filepath.Join(dir, "harbour.fountain")
	os.WriteFile(script, []byte("INT. HARBOUR - DAWN #1-x2k9q#\n\nThe boats come in.\n"), 0o644)
	w := newTestWindow(t, "")
	if err := w.LoadFile(script); err != nil {
		t.Fatal(err)
	}
	w.importStoryboarder()
	want := "INT. HARBOUR - DAWN #1-x2k9q#\n[[board: harbour.assets/" + scene + "-1A.png, duration: 2.00]]\n[[board: harbour.assets/" + scene + "-1B.png, duration: 1.50, continued: true]]\n\nThe boats come in.\n"
	if got := w.textEditor.Text(); got != want {
		t.Errorf("the script:\n%q\nwant\n%q", got, want)
	}
	data, err := os.ReadFile(filepath.Join(dir, "harbour.assets", scene+"-1A.png"))
	if err != nil {
		t.Fatal(err)
	}
	img, _ := images.Decode(data, "x.png")
	ref, _ := images.Decode(testPicture(t, 40, 20), "x.png")
	r, g, bl, _ := img.At(3, 3).RGBA()
	rr, rg, rb, _ := ref.At(3, 3).RGBA()
	mid := func(a, b uint32) bool { return math.Abs(float64(a)-float64(b+0xffff)/2) < 0x300 }
	if !mid(r, rr) || !mid(g, rg) || !mid(bl, rb) {
		t.Errorf("the reference layer at half on white: %v %v %v (layer %v %v %v)", r, g, bl, rr, rg, rb)
	}
}
