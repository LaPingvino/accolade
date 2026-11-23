package images

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pngOf(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// Names resolve beside the document, in its assets/, percent-escaped,
// absolute; data: URIs are read; the web is never asked, an SVG is not
// drawn, a missing file says so.
func TestLoad(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "assets"), 0o755)
	os.WriteFile(filepath.Join(dir, "beside.png"), pngOf(t, 4, 2), 0o644)
	os.WriteFile(filepath.Join(dir, "assets", "in assets.png"), pngOf(t, 3, 3), 0o644)
	os.WriteFile(filepath.Join(dir, "x.svg"), []byte("<svg xmlns='http://www.w3.org/2000/svg'/>"), 0o644)
	os.WriteFile(filepath.Join(dir, "text.png"), []byte("not a picture"), 0o644)
	asked := 0
	files := FilesFunc(func(rel string) ([]byte, error) {
		asked++
		return Dir(dir).ReadFile(rel)
	})
	l := New(files)
	data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngOf(t, 5, 1))
	for _, c := range []struct {
		src  string
		w    int
		want error
	}{
		{"beside.png", 4, nil},
		{"./beside.png", 4, nil},
		{"in assets.png", 3, nil},
		{"in%20assets.png", 3, nil},
		{"assets/in%20assets.png", 3, nil},
		{filepath.Join(dir, "beside.png"), 4, nil},
		{"file:///" + strings.TrimPrefix(filepath.ToSlash(filepath.Join(dir, "beside.png")), "/"), 4, nil},
		{data, 5, nil},
		{"https://example.com/a.png", 0, ErrRemote},
		{"//example.com/a.png", 0, ErrRemote},
		{`\\server\share\a.png`, 0, ErrRemote},
		{"x.svg", 0, ErrSVG},
		{"text.png", 0, ErrFormat},
		{"gone.png", 0, fs.ErrNotExist},
		{"", 0, ErrNone},
	} {
		p := l.Load(c.src)
		if c.want != nil {
			if !errors.Is(p.Err, c.want) {
				t.Errorf("%s: %v, want %v", c.src, p.Err, c.want)
			}
			continue
		}
		if w, _ := p.Size(); p.Err != nil || w != c.w {
			t.Errorf("%s: %v, %d wide", c.src, p.Err, w)
		}
	}
	before := asked
	l.Load("beside.png")
	if asked != before {
		t.Error("read again, not kept")
	}
	l.Forget()
	l.Load("beside.png")
	if asked == before {
		t.Error("kept after Forget")
	}
	// a missing picture, there later: shown
	if l.Load("later.png").OK() {
		t.Fatal("there before it is")
	}
	os.WriteFile(filepath.Join(dir, "later.png"), pngOf(t, 2, 2), 0o644)
	if !l.Load("later.png").OK() {
		t.Error("a picture put there later not read")
	}
	if th := l.Thumbnail("beside.png", 2, 2); th == nil || th.Bounds().Dx() != 2 || th.Bounds().Dy() != 1 {
		t.Errorf("thumbnail %v", th)
	}
}
