// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Screenshots (SHOT_DIR=… go test -run TestNewSceneScreenshots): New
// Scene in English, Chinese and Hindi, the heading pattern builder, the
// completion list in a heading and a cue; desktop and phone.
func TestNewSceneScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(dir, 0o755)
	save := func(name string, c fyne.Canvas) {
		t.Helper()
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		png.Encode(f, c.Capture())
	}
	scripts := map[string]string{
		"en":      "INT. KITCHEN - DAY\n\nRain on the window.\n\nANNA\nWhere were you?\n\nEXT. GARDEN - NIGHT\n\nTom waits.\n",
		"zh":      "场景1 客厅 日 内\n\n下雨了。\n\n场景2 小区楼下 夜 外\n\n他在等。\n",
		"hi-Deva": "रसोई/दिन/अंदर\n\nबारिश हो रही है।\n\nबगीचा/रात/बाहर\n\nवह इंतज़ार करता है।\n",
	}
	for _, size := range []struct {
		name string
		w, h float32
	}{{"desktop", 980, 700}, {"phone", 380, 800}} {
		for _, lang := range []string{"en", "zh", "hi-Deva"} {
			w := newTestWindow(t, "")
			w.setLanguage(lang)
			w.textEditor.SetText(scripts[lang])
			w.fyneWindow.Resize(fyne.NewSize(size.w, size.h))
			w.textEditor.SetCursorOffset(5)
			w.showNewScene()
			save("newscene-"+size.name+"-"+lang, w.fyneWindow.Canvas())
		}
		// the completion list: a heading's place, a cue
		for _, c := range []struct{ name, typed string }{{"heading", "\nEXT. "}, {"cue", "\nAN"}} {
			w := newTestWindow(t, scripts["en"]+"\nANDREW\nHi.\n")
			w.fyneWindow.Resize(fyne.NewSize(size.w, size.h))
			w.textEditor.SetCursorOffset(len([]rune(w.textEditor.Text())))
			typeKeys(t, w, c.typed)
			w.fyneWindow.Content().Refresh()
			save("complete-"+size.name+"-"+c.name, w.fyneWindow.Canvas())
		}
		// the builder: Chinese's patterns
		w := tomlTestApp(t)
		conf := "[Languages.zh]\n# my patterns\nHeadings = [\"场景{n} {place} {time} {io}\", \"{n}. {place} {time} {io}\"]\n"
		cpath := filepath.Join(t.TempDir(), "lexington.toml")
		os.WriteFile(cpath, []byte(conf), 0o644)
		w.openPath(cpath)
		e := w.app.tomlEditorOf(cpath)
		e.win.Resize(fyne.NewSize(size.w, size.h))
		e.sel["language"] = "zh"
		selectTab(t, e, "Languages")
		e.redraw()
		e.win.Content().Refresh()
		// scrolled to Headings
		if sc := find(e.tabs.Selected().Content, func(*container.Scroll) bool { return true }); sc != nil {
			sc.ScrollToOffset(fyne.NewPos(0, headingsOffset(sc)))
		}
		e.win.Content().Refresh()
		e.win.Content().Refresh()
		save("builder-"+size.name, e.win.Canvas())
		e.doc.saved = e.doc.text
		e.close()
	}
}

// headingsOffset is how far down the language page its Headings field is.
func headingsOffset(s *container.Scroll) float32 {
	var walk func(o fyne.CanvasObject, off float32) (float32, bool)
	walk = func(o fyne.CanvasObject, off float32) (float32, bool) {
		if l, ok := o.(*widget.Label); ok && l.Text == "Headings" {
			return off + o.Position().Y, true
		}
		if c, ok := o.(*fyne.Container); ok {
			for _, k := range c.Objects {
				if y, ok := walk(k, off+o.Position().Y); ok {
					return y, true
				}
			}
		}
		return 0, false
	}
	y, _ := walk(s.Content, 0)
	return y
}

// Screenshots (SHOT_DIR=… go test -run TestSceneNumberShots): a Chinese
// script with its numbers in its headings before and after its last
// scene moves to the top, and Number Scenes in Headings turned off;
// desktop and phone.
func TestSceneNumberShots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(dir, 0o755)
	save := func(name string, c fyne.Canvas) {
		t.Helper()
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		png.Encode(f, c.Capture())
	}
	for _, size := range []struct {
		name string
		w, h float32
	}{{"desktop", 980, 700}, {"phone", 380, 800}} {
		w := numberedWindow(t, "zh")
		w.fyneWindow.Resize(fyne.NewSize(size.w, size.h))
		w.fyneWindow.Content().Refresh()
		save("numbers-"+size.name+"-before", w.fyneWindow.Canvas())
		w.moveOutlineItem(2, 0)
		w.fyneWindow.Content().Refresh()
		save("numbers-"+size.name+"-moved", w.fyneWindow.Canvas())
		w.toggleNumbersInHeadings()
		w.fyneWindow.Content().Refresh()
		save("numbers-"+size.name+"-off", w.fyneWindow.Canvas())
		w.toggleNumbersInHeadings()
		w.fyneWindow.Content().Refresh()
		save("numbers-"+size.name+"-on", w.fyneWindow.Canvas())
	}
}
