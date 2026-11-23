package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"image/color"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	"github.com/LaPingvino/accolade/internal/images"
	"github.com/LaPingvino/accolade/internal/prose"
)

// The document's pictures (internal/images): a Markdown document's
// ![alt](path), a screenplay's storyboard frames, the characters' cast
// photos, all read through one Loader per document — the files beside
// it, or in its bundle (assets/…). The Live Preview, the page view, the
// PDF, Word and ODT, the frame chips and the Characters tab share it.
// Nothing on the web is fetched.

// picturesOf is what the window's loader reads: the bundle's files, or
// those beside the plain file (none for one not saved, nor in the
// browser, where a plain file has no folder).
func (w *MainWindow) picturesFiles() images.Files {
	if b := w.bundle; b != nil {
		return images.FilesFunc(func(rel string) ([]byte, error) { return b.readExtra(strings.TrimPrefix(rel, "./")) })
	}
	if w.currentFile != "" && !inBrowser {
		return images.Dir(filepath.Dir(w.currentFile))
	}
	if w.currentFile == "" && w.unsavedDir != "" && !inBrowser {
		return images.Dir(w.unsavedDir)
	}
	return nil
}

// pictures is the window's picture loader, for the document as it is
// now: made again when the document (or its bundle, written) changes.
func (w *MainWindow) pictures() *images.Loader {
	key := picturesKey{w.bundle, w.currentFile, w.unsavedDir}
	if w.pics == nil || w.picsKey != key {
		w.pics, w.picsKey = images.New(w.picturesFiles()), key
	}
	return w.pics
}

// picturesKey is what the loader reads from.
type picturesKey struct {
	b         *textBundle
	file, dir string
}

// mdImage is a picture as the Markdown names it.
type mdImage struct{ src, alt, title string }

// markdownImages are the pictures in Markdown, in the order Fyne's rich
// text has them (the document's).
func markdownImages(md string) []mdImage {
	if !strings.Contains(md, "![") {
		return nil
	}
	src := []byte(md)
	root := prose.Markdown.Parser().Parse(text.NewReader(src))
	var out []mdImage
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if img, ok := n.(*ast.Image); ok && entering {
			var alt strings.Builder
			for c := img.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					alt.Write(t.Segment.Value(src))
				}
			}
			out = append(out, mdImage{string(img.Destination), alt.String(), string(img.Title)})
		}
		return ast.WalkContinue, nil
	})
	return out
}

// withPictures replaces Fyne's image segments (which would read a path
// from the working folder, or fetch an address) by the document's
// pictures, read by l (nil: none read, each a placeholder).
func withPictures(segs []widget.RichTextSegment, imgs []mdImage, l *images.Loader) []widget.RichTextSegment {
	n := 0
	var walk func([]widget.RichTextSegment) []widget.RichTextSegment
	walk = func(segs []widget.RichTextSegment) []widget.RichTextSegment {
		out := segs[:0:0]
		for _, seg := range segs {
			switch s := seg.(type) {
			case *widget.ImageSegment:
				img := mdImage{src: s.Source.String(), title: s.Title}
				if n < len(imgs) {
					img = imgs[n]
				}
				n++
				out = append(out, &pictureSegment{img: img, pic: l.Load(img.src)})
				continue
			case *widget.ParagraphSegment:
				p := *s
				p.Texts = walk(s.Texts)
				seg = &p
			case *widget.ListSegment:
				ls := *s
				ls.Items = walk(s.Items)
				seg = &ls
			}
			out = append(out, seg)
		}
		return out
	}
	return walk(segs)
}

// previewPictureWidth is the most a picture in the preview takes of its
// width at first (its pane may be narrow).
const previewPictureWidth = 300

// pictureSegment is a picture in the Live Preview: its image at its
// size (a smaller one at most previewPictureWidth wide), its caption
// under it; a quiet placeholder with the alt text and where it is when
// it cannot be shown.
type pictureSegment struct {
	img mdImage
	pic *images.Picture
}

func (p *pictureSegment) Inline() bool               { return false }
func (p *pictureSegment) Textual() string            { return p.img.alt }
func (p *pictureSegment) Select(_, _ fyne.Position)  {}
func (p *pictureSegment) SelectedText() string       { return "" }
func (p *pictureSegment) Unselect()                  {}
func (p *pictureSegment) Update(o fyne.CanvasObject) {}

func (p *pictureSegment) Visual() fyne.CanvasObject {
	if !p.pic.OK() {
		return picturePlaceholder(p.img.alt, p.img.src, p.pic.Err)
	}
	pw, ph := p.pic.Size()
	w := min(float32(pw), previewPictureWidth)
	img := canvas.NewImageFromImage(p.pic.Image)
	img.FillMode = canvas.ImageFillContain
	img.ScaleMode = canvas.ImageScaleSmooth
	img.SetMinSize(fyne.NewSize(w, w*float32(ph)/float32(pw)))
	fyne.SetAccessibleLabel(img, p.img.alt)
	objs := []fyne.CanvasObject{container.NewCenter(img)}
	if c := p.img.title; c != "" {
		cap := widget.NewLabelWithStyle(c, fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
		cap.Wrapping = fyne.TextWrapWord
		cap.SizeName = theme.SizeNameCaptionText
		objs = append(objs, cap)
	}
	return container.NewVBox(objs...)
}

// picturePlaceholder is a picture that cannot be shown: its alt text,
// where it is and why, quiet.
func picturePlaceholder(alt, src string, err error) fyne.CanvasObject {
	where := src
	if strings.HasPrefix(strings.ToLower(where), "data:") {
		where = "an embedded picture"
	} else if len([]rune(where)) > 80 {
		where = string([]rune(where)[:77]) + "…"
	}
	label := widget.NewLabel(strings.TrimSpace("🖼 " + strings.TrimSpace(alt) + "\n" + where + " (" + images.Why(err) + ")"))
	label.Wrapping = fyne.TextWrapWord
	label.Importance = widget.LowImportance
	label.SizeName = theme.SizeNameCaptionText
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeColor = theme.Color(theme.ColorNameSeparator)
	border.StrokeWidth = 1
	border.CornerRadius = 4
	return container.NewStack(border, label)
}

// imgTag is an <img> as goldmark writes it.
var imgTag = regexp.MustCompile(`<img src="([^"]*)" alt="([^"]*)"((?: title="[^"]*")?)\s*/?>`)

// inlinePictures makes a web page of the document stand alone, for a
// browser to print or make a PDF of from a temporary folder: each local
// picture in a data: URI, one on the web its alt text and link (never
// fetched), one not found its alt text and where it was.
func inlinePictures(page []byte, l *images.Loader) []byte {
	return imgTag.ReplaceAllFunc(page, func(tag []byte) []byte {
		m := imgTag.FindSubmatch(tag)
		src, alt := html.UnescapeString(string(m[1])), html.UnescapeString(string(m[2]))
		p := l.Load(src)
		switch {
		case p.OK():
			uri := "data:" + mimeOf(p.Data) + ";base64," + base64.StdEncoding.EncodeToString(p.Data)
			return []byte(fmt.Sprintf(`<img src="%s" alt="%s"%s style="max-width:100%%">`, uri, m[2], m[3]))
		case images.IsRemote(src):
			return []byte(fmt.Sprintf(`<a href="%s">%s</a>`, m[1], html.EscapeString(cmpOr(alt, src))))
		}
		return []byte(fmt.Sprintf(`<span style="color:#777;font-style:italic">[%s — %s, %s]</span>`,
			html.EscapeString(cmpOr(alt, "picture")), html.EscapeString(src), images.Why(p.Err)))
	})
}

// mimeOf is a picture's media type by its bytes.
func mimeOf(data []byte) string {
	switch {
	case bytes.HasPrefix(data, []byte("\xff\xd8")):
		return "image/jpeg"
	case bytes.HasPrefix(data, []byte("GIF8")):
		return "image/gif"
	case len(data) > 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "image/webp"
	}
	return "image/png"
}

// pictureFolder is where pictures added to the document go, as the
// document names them: dataPath's folder when a screenplay names one,
// else a bundle's assets/, else Name.assets/ beside the plain file.
func (w *MainWindow) pictureFolder(dataPath string) string {
	switch {
	case dataPath != "":
		return strings.Trim(path.Clean(dataPath), "/")
	case w.bundle != nil:
		return "assets"
	}
	return strings.TrimSuffix(filepath.Base(w.currentFile), filepath.Ext(w.currentFile)) + ".assets"
}

// errNoPictureHome: an unsaved script (or a plain file in the browser)
// has nowhere to keep a picture.
var errNoPictureHome = errors.New("save the script first — as an .accolade (File › Save As) to keep its pictures inside it, " +
	"or, on the desktop, as a plain file to keep them in a folder beside it")

// keepPicture copies the picture file at src into the document's picture
// folder (folder), under its own name (numbered if another picture has
// it), and tells done the name the document links it by (relative to
// the document). Read from the folder at once (the loader made again).
func (w *MainWindow) keepPicture(src, folder string, done func(rel string, err error)) {
	data, err := readPickedFile(src)
	if err == nil {
		_, err = images.Decode(data, src)
		if errors.Is(err, images.ErrSVG) {
			err = nil // kept: shown as its alt text
		}
	}
	if err != nil {
		done("", fmt.Errorf("%s: %v", filepath.Base(src), err))
		return
	}
	name := pictureName(filepath.Base(src))
	if b := w.bundle; b != nil {
		for i := 2; ; i++ { // a name not taken by another picture
			old, err := b.readExtra(folder + "/" + name)
			if err != nil || bytes.Equal(old, data) {
				break
			}
			ext := path.Ext(name)
			name = fmt.Sprintf("%s-%d%s", strings.TrimSuffix(pictureName(filepath.Base(src)), ext), i, ext)
		}
		rel := folder + "/" + name
		files := []bundleFile{{rel, data}}
		w.queueBundle(bundleJob{run: func(b *textBundle) (*textBundle, error) {
			if b == nil {
				return nil, errors.New("the script's bundle is gone")
			}
			return saveBundleFiles(b, files, false)
		}, done: func(err error) { done(rel, err) }})
		return
	}
	if w.currentFile == "" || inBrowser {
		done("", errNoPictureHome)
		return
	}
	dir := filepath.Join(filepath.Dir(w.currentFile), filepath.FromSlash(folder))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		done("", err)
		return
	}
	name = uniqueName(dir, name, data)
	if err := writeFileAtomic(filepath.Join(dir, name), data); err != nil {
		done("", err)
		return
	}
	w.pictures().Forget()
	done(folder+"/"+name, nil)
}

// pictureName is a file's name as a picture's in a document: no spaces
// (a Markdown link and a board note read it as it is).
func pictureName(name string) string {
	name = strings.Map(func(r rune) rune {
		if r == ' ' || r == ',' || r == '(' || r == ')' || r == '[' || r == ']' || r == '<' || r == '>' {
			return '-'
		}
		return r
	}, name)
	return strings.TrimLeft(name, ".-")
}

// readPickedFile reads a file picked in the file dialog.
var readPickedFile = os.ReadFile

// pictureExtensions are the pictures a file dialog offers.
var pictureExtensions = []string{".png", ".jpg", ".jpeg", ".gif", ".webp"}
