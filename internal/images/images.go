// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

// Package images is the one place Accolade reads pictures from: a
// Markdown document's ![alt](path), a screenplay's storyboard frames
// ([[board: …]]) and the characters' cast photos all go through a
// Loader, which the Live Preview, the page view, the PDF, Word and ODT,
// the script's frame chips and the Characters tab share.
//
// A picture is named as the document names it: a path relative to the
// document (in a bundle, to the bundle's root, so assets/shot.png; a
// name not found there is looked for in assets/ too), an absolute path
// or file: URL on the desktop, or a data: URI. Remote addresses
// (http, https, anything with a host) are never fetched — not in the
// preview, not in an export: privacy, working offline, and the browser's
// CORS. They come back as ErrRemote, and the caller shows the alt text
// and the link. PNG, JPEG, GIF and WebP are read, and SVG, Krita's .kra,
// Synfig's and Lottie's files as formats.go draws them.
package images

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	_ "image/gif" // the formats read
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// The reasons a picture is not shown.
var (
	ErrRemote = errors.New("a picture on the web: not fetched")
	ErrFormat = errors.New("not a picture Accolade reads (PNG, JPEG, GIF, WebP)")
	ErrNone   = errors.New("no picture named")
)

// Files reads a document's files by their path relative to it (slashes;
// an absolute path as it is). A bundle's are read from the bundle.
type Files interface {
	ReadFile(rel string) ([]byte, error)
}

// Dir is the files of a document in a folder on disk ("" none).
type Dir string

func (d Dir) ReadFile(rel string) ([]byte, error) {
	if filepath.IsAbs(rel) || filepath.IsAbs(filepath.FromSlash(rel)) {
		return os.ReadFile(filepath.FromSlash(rel))
	}
	if d == "" {
		return nil, fs.ErrNotExist
	}
	return os.ReadFile(filepath.Join(string(d), filepath.FromSlash(rel)))
}

// FilesFunc is a function as Files.
type FilesFunc func(rel string) ([]byte, error)

func (f FilesFunc) ReadFile(rel string) ([]byte, error) { return f(rel) }

// Picture is a picture as read: its file's bytes and the image, or why it
// cannot be shown (Err).
type Picture struct {
	Src  string // as the document names it
	Data []byte // a picture's bytes: a drawing program's file as its PNG
	// Orig is the file's bytes as they are (Data, but for a drawing
	// program's file)
	Orig  []byte
	Image image.Image
	Err   error
}

// OK tells whether the picture can be shown.
func (p *Picture) OK() bool { return p != nil && p.Err == nil && p.Image != nil }

// Size is the picture's size in pixels (0, 0 if it cannot be shown).
func (p *Picture) Size() (w, h int) {
	if !p.OK() {
		return 0, 0
	}
	b := p.Image.Bounds()
	return b.Dx(), b.Dy()
}

// IsRemote tells whether src names a picture elsewhere: an address with
// a scheme other than file: and data:, or a host.
func IsRemote(src string) bool {
	src = strings.TrimSpace(src)
	if strings.HasPrefix(src, "//") || strings.HasPrefix(src, `\\`) { // (a UNC path: the network)
		return true
	}
	u, err := url.Parse(src)
	if err != nil || u.Scheme == "" || len(u.Scheme) == 1 { // (C:\…)
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "file":
		return u.Host != "" && u.Host != "localhost"
	case "data":
		return false
	}
	return true
}

// Loader reads pictures and keeps what it read, by name. It is safe for
// use by several goroutines.
type Loader struct {
	mu    sync.Mutex
	files Files
	cache map[string]*Picture
	thumb map[thumbKey]image.Image
	gen   int
}

type thumbKey struct {
	src  string
	w, h int
}

// New is a Loader of the document's files (nil: none, data: URIs only).
func New(files Files) *Loader {
	return &Loader{files: files, cache: map[string]*Picture{}, thumb: map[thumbKey]image.Image{}}
}

// SetFiles makes the loader read another document's files (or the same
// as they are now): what it kept is forgotten.
func (l *Loader) SetFiles(files Files) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.files = files
	l.forget()
}

// Reading makes the loader read from files from now on, keeping what it
// read (the same document's files, written again: a bundle saved).
func (l *Loader) Reading(files Files) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.files = files
}

// Forget forgets what was read: the files are read again when asked.
func (l *Loader) Forget() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.forget()
}

func (l *Loader) forget() {
	l.cache, l.thumb = map[string]*Picture{}, map[thumbKey]image.Image{}
	l.gen++
}

// Gen changes each time what was read is forgotten (a view's cache of
// what it drew can follow it).
func (l *Loader) Gen() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.gen
}

// Load is the picture src names, read (or as read before).
func (l *Loader) Load(src string) *Picture {
	if l == nil {
		return &Picture{Src: src, Err: fs.ErrNotExist}
	}
	l.mu.Lock()
	if p, ok := l.cache[src]; ok {
		l.mu.Unlock()
		return p
	}
	files := l.files
	l.mu.Unlock()
	p := read(files, src)
	if !errors.Is(p.Err, fs.ErrNotExist) { // (one not there yet is looked for again)
		l.mu.Lock()
		l.cache[src] = p
		l.mu.Unlock()
	}
	return p
}

// Thumbnail is the picture made to fit in w × h pixels (its shape kept;
// never larger than it is), nil if it cannot be shown.
func (l *Loader) Thumbnail(src string, w, h int) image.Image {
	p := l.Load(src)
	if !p.OK() || w <= 0 || h <= 0 {
		return nil
	}
	k := thumbKey{src, w, h}
	l.mu.Lock()
	if t, ok := l.thumb[k]; ok {
		l.mu.Unlock()
		return t
	}
	l.mu.Unlock()
	t := Fit(p.Image, w, h)
	l.mu.Lock()
	l.thumb[k] = t
	l.mu.Unlock()
	return t
}

// Fit is img made to fit in w × h pixels, its shape kept, never larger.
func Fit(img image.Image, w, h int) image.Image {
	b := img.Bounds()
	if b.Dx() <= w && b.Dy() <= h {
		return img
	}
	s := min(float64(w)/float64(b.Dx()), float64(h)/float64(b.Dy()))
	tw, th := max(1, int(float64(b.Dx())*s+0.5)), max(1, int(float64(b.Dy())*s+0.5))
	out := image.NewNRGBA(image.Rect(0, 0, tw, th))
	xdraw.CatmullRom.Scale(out, out.Bounds(), img, b, xdraw.Src, nil)
	return out
}

// read reads a picture.
func read(files Files, src string) *Picture {
	p := &Picture{Src: src}
	s := strings.TrimSpace(src)
	switch {
	case s == "":
		p.Err = ErrNone
		return p
	case strings.HasPrefix(strings.ToLower(s), "data:"):
		p.Data, p.Err = dataURI(s)
	case IsRemote(s):
		p.Err = ErrRemote
		return p
	default:
		p.Data, p.Err = readLocal(files, s)
	}
	if p.Err != nil {
		return p
	}
	p.Orig = p.Data
	p.Image, p.Err = Decode(p.Data, s)
	if p.Err == nil && KindOf(s, p.Data) != "" {
		p.Data = encodePNG(p.Image)
	}
	return p
}

// Decode reads a picture's bytes (name tells an SVG, a .kra… by its
// extension).
func Decode(data []byte, name string) (image.Image, error) {
	if k := KindOf(name, data); k != "" {
		return decodeKind(k, data, name)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrFormat
	}
	return img, nil
}

func isSVG(data []byte, name string) bool {
	if strings.EqualFold(path.Ext(strings.SplitN(name, "?", 2)[0]), ".svg") {
		return true
	}
	head := bytes.TrimSpace(data[:min(len(data), 512)])
	return bytes.HasPrefix(head, []byte("<svg")) || (bytes.HasPrefix(head, []byte("<?xml")) && bytes.Contains(head, []byte("<svg")))
}

// readLocal reads a path relative to the document (or absolute, or a
// file: URL), percent-escapes undone if the name as written is not
// there, then in assets/.
func readLocal(files Files, s string) ([]byte, error) {
	if strings.HasPrefix(strings.ToLower(s), "file:") {
		u, err := url.Parse(s)
		if err != nil {
			return nil, err
		}
		s = u.Path
		if len(s) > 2 && s[0] == '/' && s[2] == ':' { // file:///C:/…
			s = s[1:]
		}
	}
	if files == nil {
		return nil, fs.ErrNotExist
	}
	names := []string{s}
	if u, err := url.PathUnescape(s); err == nil && u != s {
		names = append(names, u)
	}
	for _, n := range names {
		if !filepath.IsAbs(filepath.FromSlash(n)) && !strings.HasPrefix(path.Clean(n), "assets/") {
			names = append(names, "assets/"+strings.TrimPrefix(n, "./"))
		}
	}
	var first error
	for _, n := range names {
		data, err := files.ReadFile(n)
		if err == nil {
			return data, nil
		}
		if first == nil {
			first = err
		}
	}
	return nil, first
}

// dataURI is a data: URI's bytes.
func dataURI(s string) ([]byte, error) {
	head, body, ok := strings.Cut(s[len("data:"):], ",")
	if !ok {
		return nil, ErrFormat
	}
	if strings.HasSuffix(strings.ToLower(head), ";base64") {
		body = strings.Map(func(r rune) rune {
			if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
				return -1
			}
			return r
		}, body)
		data, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			data, err = base64.RawStdEncoding.DecodeString(strings.TrimRight(body, "="))
		}
		if err != nil {
			return nil, ErrFormat
		}
		return data, nil
	}
	u, err := url.PathUnescape(body)
	if err != nil {
		return nil, ErrFormat
	}
	return []byte(u), nil
}

// Why is a short reason a picture is not shown, for a placeholder.
func Why(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrRemote):
		return "on the web, not fetched"
	case errors.Is(err, ErrNoRenderer):
		return "its program (Synfig, Glaxnimate) draws it: not installed"
	case errors.Is(err, ErrRendering):
		return "being drawn…"
	case errors.Is(err, fs.ErrNotExist), errors.Is(err, ErrNone):
		return "not found"
	}
	return "not a picture"
}
