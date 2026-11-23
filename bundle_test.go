package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	lb "github.com/LaPingvino/lexington/bundle"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// zipItem is one entry of a zip made for a test.
type zipItem struct {
	name   string
	data   string
	stored bool // not compressed (as images are)
}

// makeZip is a zip of items, with a comment.
func makeZip(t *testing.T, items []zipItem) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	when := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	for _, it := range items {
		h := &zip.FileHeader{Name: it.name, Method: zip.Deflate, Modified: when}
		if it.stored || strings.HasSuffix(it.name, "/") {
			h.Method = zip.Store
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		io.WriteString(w, it.data)
	}
	zw.SetComment("made for a test")
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// highlandInfo is an info.json as Highland writes one (docs/HIGHLAND.md).
const highlandInfo = `{
  "com.quoteunquoteapps.highland2" : { "lastScrollPosition": 190.5, "cursorPosition": 0, "templateName": "Screenplay" },
  "transient" : false,
  "creatorURL" : "file:\/\/\/Applications\/Highland 2.app\/",
  "type" : "com.quoteunquoteapps.fountain",
  "version" : 2
}`

const bundleScript = "INT. KITCHEN - DAY\n\nBram cooks.\n"

// bundleItems are a bundle's entries: in a Name.textbundle/ folder
// (prefix) or at the root (""), with assets, Highland's sidecars, a file
// no one knows and macOS's junk.
func bundleItems(prefix string) []zipItem {
	png := string([]byte{0x89, 'P', 'N', 'G', 0, 1, 2, 3, 0xff, 0xfe})
	items := []zipItem{}
	if prefix != "" {
		items = append(items, zipItem{name: prefix})
	}
	items = append(items,
		zipItem{name: prefix + "info.json", data: highlandInfo},
		zipItem{name: prefix + "characters.json", data: `{"BRAM":{"color":"#ff0000"}}`},
		zipItem{name: prefix + "text.fountain", data: bundleScript},
		zipItem{name: prefix + "assets/"},
		zipItem{name: prefix + "assets/cover.png", data: png, stored: true},
		zipItem{name: prefix + "unknown.bin", data: "\x00\x01 keep me"},
		zipItem{name: prefix + "sprints.json", data: "[]"},
	)
	if prefix != "" {
		items = append(items, zipItem{name: "__MACOSX/" + prefix + "._text.fountain", data: "junk"})
	}
	return items
}

// rawEntry is a zip entry as stored: its header's name, method and CRC,
// and its compressed bytes.
type rawEntry struct {
	name   string
	method uint16
	crc    uint32
	raw    []byte
}

func rawEntries(t *testing.T, data []byte) []rawEntry {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var es []rawEntry
	for _, f := range zr.File {
		r, err := f.OpenRaw()
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(r)
		es = append(es, rawEntry{f.Name, f.Method, f.CRC32, b})
	}
	return es
}

// zipFile is a file's contents in a zip.
func zipFile(t *testing.T, data []byte, name string) (string, bool) {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range zr.File {
		if f.Name == name {
			r, _ := f.Open()
			b, _ := io.ReadAll(r)
			return string(b), true
		}
	}
	return "", false
}

// sameExceptText checks that after holds before's entries in their order,
// each byte for byte as stored, but for the text (and info.json, if
// infoToo is false).
func sameExceptText(t *testing.T, before, after []byte, text string, infoChanges bool) {
	t.Helper()
	b, a := rawEntries(t, before), rawEntries(t, after)
	// an .accolade's save adds the Lexington settings it carries (its
	// language in full): those aside, nothing else may change
	a = slices.DeleteFunc(a, func(e rawEntry) bool {
		return strings.Contains(e.name, bundleLexDir) && !slices.ContainsFunc(b, func(x rawEntry) bool { return x.name == e.name })
	})
	if len(a) != len(b) {
		t.Fatalf("entries %d, were %d", len(a), len(b))
	}
	for i := range b {
		if a[i].name != b[i].name {
			t.Fatalf("entry %d is %s, was %s", i, a[i].name, b[i].name)
		}
		if a[i].name == text || (infoChanges && strings.HasSuffix(a[i].name, "info.json")) {
			continue
		}
		if a[i].method != b[i].method || a[i].crc != b[i].crc || !bytes.Equal(a[i].raw, b[i].raw) {
			t.Errorf("%s changed", a[i].name)
		}
	}
	zb, _ := zip.NewReader(bytes.NewReader(before), int64(len(before)))
	za, _ := zip.NewReader(bytes.NewReader(after), int64(len(after)))
	if za.Comment != zb.Comment {
		t.Errorf("comment %q, was %q", za.Comment, zb.Comment)
	}
}

// Opening a zipped bundle (.highland, .textpack; both layouts) gives its
// text as a screenplay; saving changes only the text, every other entry
// copied through as stored, in its order, the layout kept.
func TestBundleZipRoundTrip(t *testing.T) {
	for _, tc := range []struct{ name, prefix string }{
		{"Script.highland", "Script.textbundle/"},
		{"Root.highland", ""},
		{"Pack.textpack", "Pack.textbundle/"},
		{"Root.textpack", ""},
		{"Script.accolade", "Script.textbundle/"},
		{"Root.accolade", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, _ := newDialogTestWindow(t)
			p := filepath.Join(t.TempDir(), tc.name)
			orig := makeZip(t, bundleItems(tc.prefix))
			os.WriteFile(p, orig, 0o644)
			if err := w.LoadFile(p); err != nil {
				t.Fatal(err)
			}
			if w.textEditor.Text() != bundleScript || w.markdown || w.bundle == nil || w.bundle.Prefix != tc.prefix {
				t.Fatalf("opened %q, markdown %v, bundle %+v", w.textEditor.Text(), w.markdown, w.bundle)
			}
			if r := w.settings.GetStringSlice("recent-files"); len(r) == 0 || r[0] != p {
				t.Errorf("recent: %v", r)
			}
			w.textEditor.SetText(bundleScript + "\nBram eats.\n")
			if err := w.SaveFile(); err != nil {
				t.Fatal(err)
			}
			after, _ := os.ReadFile(p)
			sameExceptText(t, orig, after, tc.prefix+"text.fountain", false)
			if got, _ := zipFile(t, after, tc.prefix+"text.fountain"); got != bundleScript+"\nBram eats.\n" {
				t.Errorf("text saved: %q", got)
			}
			if info, _ := zipFile(t, after, tc.prefix+"info.json"); info != highlandInfo {
				t.Errorf("info.json changed:\n%s", info)
			}
			// and it opens again as saved
			w2, _ := newDialogTestWindow(t)
			if err := w2.LoadFile(p); err != nil || !strings.Contains(w2.textEditor.Text(), "Bram eats.") {
				t.Errorf("reopened: %v %q", err, w2.textEditor.Text())
			}
		})
	}
}

// Save As onto the open bundle: the save dialog has emptied the file, and
// the bundle is still written whole from what was read.
func TestBundleSaveOverEmptied(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	p := filepath.Join(t.TempDir(), "Script.highland")
	orig := makeZip(t, bundleItems("Script.textbundle/"))
	os.WriteFile(p, orig, 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(p, nil, 0o644) // as the dialog leaves it
	if err := w.saveToFile(p); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(p)
	sameExceptText(t, orig, after, "Script.textbundle/text.fountain", false)
}

// Save As another bundle carries every entry; the folder inside is named
// for the new file, files at the root stay there.
func TestBundleSaveAsAnother(t *testing.T) {
	for _, prefix := range []string{"Script.textbundle/", ""} {
		w, _ := newDialogTestWindow(t)
		dir := t.TempDir()
		p := filepath.Join(dir, "Script.highland")
		os.WriteFile(p, makeZip(t, bundleItems(prefix)), 0o644)
		if err := w.LoadFile(p); err != nil {
			t.Fatal(err)
		}
		to := filepath.Join(dir, "Copy.textpack")
		if err := w.saveToFile(to); err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(to)
		want := ""
		if prefix != "" {
			want = "Copy.textbundle/"
		}
		var names []string
		for _, e := range rawEntries(t, data) {
			names = append(names, e.name)
		}
		for _, it := range bundleItems(prefix) {
			n := it.name
			if prefix != "" && strings.HasPrefix(n, prefix) {
				n = want + strings.TrimPrefix(n, prefix)
			}
			if !slices.Contains(names, n) {
				t.Errorf("%q: no %s in %v", prefix, n, names)
			}
		}
		if png, _ := zipFile(t, data, want+"assets/cover.png"); png != bundleItems("")[4].data {
			t.Errorf("asset %q", png)
		}
		if w.bundle == nil || w.bundle.Path != to {
			t.Errorf("the window's bundle: %+v", w.bundle)
		}
	}
}

// A plain script saved as a bundle makes a minimal one: info.json
// (version 2, its type, not transient), text.fountain or text.md, in a
// Name.textbundle/ folder for a zip; a .textbundle folder on the desktop.
func TestBundleFromPlain(t *testing.T) {
	for _, tc := range []struct {
		name     string
		markdown bool
	}{
		{"New.highland", false}, {"New.accolade", false}, {"New.textpack", false}, {"Story.textpack", true}, {"New.textbundle", false}, {"Story.textbundle", true},
	} {
		w, _ := newDialogTestWindow(t)
		dir := t.TempDir()
		ext := ".fountain"
		text := bundleScript
		if tc.markdown {
			ext, text = ".md", "# Story\n\nOnce.\n"
		}
		plain := filepath.Join(dir, "plain"+ext)
		os.WriteFile(plain, []byte(text), 0o644)
		if err := w.LoadFile(plain); err != nil {
			t.Fatal(err)
		}
		to := filepath.Join(dir, tc.name)
		if tc.name == "New.textbundle" {
			os.WriteFile(to, nil, 0o644) // the save dialog's empty file
		}
		if err := w.saveToFile(to); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		textName := "text.fountain"
		wantType := lb.FountainUTI
		if tc.markdown {
			textName, wantType = "text.md", lb.MarkdownUTI
		}
		var info, got string
		if bundleKindOf(to) == bundleFolder {
			b, _ := os.ReadFile(filepath.Join(to, "info.json"))
			t2, _ := os.ReadFile(filepath.Join(to, textName))
			info, got = string(b), string(t2)
			if st, err := os.Stat(filepath.Join(to, "assets")); err != nil || !st.IsDir() {
				t.Errorf("%s: no assets/", tc.name)
			}
		} else {
			data, _ := os.ReadFile(to)
			folder := strings.TrimSuffix(tc.name, filepath.Ext(tc.name)) + ".textbundle/"
			info, _ = zipFile(t, data, folder+"info.json")
			got, _ = zipFile(t, data, folder+textName)
		}
		var meta struct {
			Version   int    `json:"version"`
			Type      string `json:"type"`
			Transient *bool  `json:"transient"`
		}
		if err := json.Unmarshal([]byte(info), &meta); err != nil || meta.Version != 2 || meta.Type != wantType || meta.Transient == nil || *meta.Transient {
			t.Errorf("%s: info.json %s (%v)", tc.name, info, err)
		}
		if got != text {
			t.Errorf("%s: text %q", tc.name, got)
		}
		w2, _ := newDialogTestWindow(t)
		if err := w2.LoadFile(to); err != nil || w2.textEditor.Text() != text || w2.markdown != tc.markdown {
			t.Errorf("%s reopened: %v, %q, markdown %v", tc.name, err, w2.textEditor.Text(), w2.markdown)
		}
	}
}

// A .textbundle folder opens by its path or by its text file's (as Open
// goes into the folder); saving writes the text, nothing else.
func TestBundleFolder(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	dir := filepath.Join(t.TempDir(), "Story.textbundle")
	os.MkdirAll(filepath.Join(dir, "assets"), 0o755)
	info := "{\n    \"version\":2,\n    \"type\":\"net.daringfireball.markdown\",\n    \"other.app\": {\"x\": 1},\n}"
	os.WriteFile(filepath.Join(dir, "info.json"), []byte(info), 0o644)
	os.WriteFile(filepath.Join(dir, "text.md"), []byte("# Story\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "assets", "pic.jpg"), []byte("jpeg"), 0o644)
	old := time.Now().Add(-time.Hour)
	os.Chtimes(filepath.Join(dir, "info.json"), old, old)
	if err := w.LoadFile(filepath.Join(dir, "text.md")); err != nil {
		t.Fatal(err)
	}
	if w.currentFile != dir || !w.markdown || w.textEditor.Text() != "# Story\n" {
		t.Fatalf("opened %s, markdown %v, %q", w.currentFile, w.markdown, w.textEditor.Text())
	}
	w.textEditor.SetText("# Story\n\nMore.\n")
	if err := w.SaveFile(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "text.md")); string(b) != "# Story\n\nMore.\n" {
		t.Errorf("text %q", b)
	}
	st, _ := os.Stat(filepath.Join(dir, "info.json"))
	if b, _ := os.ReadFile(filepath.Join(dir, "info.json")); string(b) != info || !st.ModTime().Equal(old) {
		t.Errorf("info.json touched: %s", b)
	}
	// Save As another folder carries the assets
	to := filepath.Join(filepath.Dir(dir), "Copy.textbundle")
	if err := w.saveToFile(to); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(to, "assets", "pic.jpg")); string(b) != "jpeg" {
		t.Errorf("asset %q", b)
	}
	// and into a zip
	pack := filepath.Join(filepath.Dir(dir), "Copy.textpack")
	if err := w.saveToFile(pack); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(pack)
	if s, ok := zipFile(t, data, "Copy.textbundle/assets/pic.jpg"); !ok || s != "jpeg" {
		t.Errorf("zipped asset %q", s)
	}
	if s, _ := zipFile(t, data, "Copy.textbundle/info.json"); s != info {
		t.Errorf("zipped info.json %q", s)
	}
}

// A bundle without a text file is an error, the window as it was.
func TestBundleNoText(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText("KEEP ME\n")
	p := filepath.Join(t.TempDir(), "Empty.highland")
	os.WriteFile(p, makeZip(t, []zipItem{{name: "info.json", data: highlandInfo}, {name: "assets/"}}), 0o644)
	err := w.LoadFile(p)
	if err == nil || !strings.Contains(err.Error(), "no text file") {
		t.Errorf("error %v", err)
	}
	if w.textEditor.Text() != "KEEP ME\n" || w.currentFile != "" || w.bundle != nil {
		t.Errorf("window changed: %q %q", w.textEditor.Text(), w.currentFile)
	}
	// a .textpack that is no zip is an error too
	q := filepath.Join(t.TempDir(), "plain.textpack")
	os.WriteFile(q, []byte(bundleScript), 0o644)
	if err := w.LoadFile(q); err == nil {
		t.Error("a .textpack of plain text opened")
	}
}

// A .highland that is no zip is Fountain, and saved as plain text again.
func TestHighlandPlainText(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	p := filepath.Join(t.TempDir(), "Plain.highland")
	os.WriteFile(p, []byte(bundleScript), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if w.textEditor.Text() != bundleScript || w.markdown || w.bundle != nil {
		t.Fatalf("opened %q", w.textEditor.Text())
	}
	w.textEditor.SetText(bundleScript + "More.\n")
	if err := w.SaveFile(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != bundleScript+"More.\n" {
		t.Errorf("saved %q", b)
	}
}

// A .accolade is always a zip: one of plain text does not open (only
// Highland's name is borrowed by plain Fountain files).
func TestAccoladePlainTextRefused(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	p := filepath.Join(t.TempDir(), "Plain.accolade")
	os.WriteFile(p, []byte(bundleScript), 0o644)
	if err := w.LoadFile(p); err == nil {
		t.Fatal("a .accolade of plain text opened")
	}
	if md, known := kindOfName("x.accolade"); md || !known {
		t.Error("a .accolade is not known as a screenplay")
	}
	if !slices.Contains(openExtensions(), ".accolade") {
		t.Error("Open does not offer .accolade")
	}
}

// View > Document Type on a bundle is kept in Accolade's key in info.json,
// every other key as it was, in its order; the bundle opens so again.
func TestBundleDocumentType(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	p := filepath.Join(t.TempDir(), "Script.highland")
	os.WriteFile(p, makeZip(t, bundleItems("")), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	w.setDocumentType(true)
	if err := w.SaveFile(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(p)
	info, _ := zipFile(t, data, "info.json")
	want := strings.TrimSuffix(highlandInfo, "\n}") + `,
  "eu.kiefte.Accolade": {"version":1,"documentType":"markdown"}
}`
	if info != want {
		t.Errorf("info.json:\n%s\nwant\n%s", info, want)
	}
	w.settings.SetStringSlice("document-types", nil) // only the file says it now
	w2, _ := newDialogTestWindow(t)
	if err := w2.LoadFile(p); err != nil || !w2.markdown {
		t.Errorf("reopened: %v, markdown %v", err, w2.markdown)
	}
	// back to a screenplay: Accolade's key goes, info.json is as it was
	w2.setDocumentType(false)
	w2.SaveFile()
	data, _ = os.ReadFile(p)
	if info, _ := zipFile(t, data, "info.json"); info != highlandInfo {
		t.Errorf("info.json back:\n%s", info)
	}
}

// The source kept in the bundle (layer 2, source_file.go): found there,
// written there by Choose Source and by the pane, the text in the bundle
// untouched by it; carried by Save As.
func TestBundleSource(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	dir := t.TempDir()
	p := filepath.Join(dir, "Script.highland")
	orig := makeZip(t, bundleItems("Script.textbundle/"))
	os.WriteFile(p, orig, 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if w.sourceRef().Origin != sourceNone {
		t.Fatalf("a source: %+v", w.sourceRef())
	}
	w.textEditor.SetText(bundleScript + "Not saved.\n")
	story := filepath.Join(dir, "novel.md")
	os.WriteFile(story, []byte("# Novel\n\nIt was night.\n"), 0o644)
	if err := w.useSource(story); err != nil {
		t.Fatal(err)
	}
	if r := w.sourceRef(); r.Origin != sourceBundle || r.Entry != bundleSourceName || r.describe() != "source.md · in the bundle" {
		t.Fatalf("source %+v", r)
	}
	data, _ := os.ReadFile(p)
	if got, _ := zipFile(t, data, "Script.textbundle/text.fountain"); got != bundleScript {
		t.Errorf("the text in the bundle changed: %q", got)
	}
	if got, _ := zipFile(t, data, "Script.textbundle/assets/source.md"); got != "# Novel\n\nIt was night.\n" {
		t.Errorf("source in the bundle: %q", got)
	}
	info, _ := zipFile(t, data, "Script.textbundle/info.json")
	if !strings.Contains(info, `"eu.kiefte.Accolade": {"version":1,"source":"assets/source.md"}`) {
		t.Errorf("info.json: %s", info)
	}
	// the pane edits it there
	s := w.sourcePane()
	w.togglePane("source")
	s.follow()
	s.toggleEdit()
	s.w.srcDoc.SetText("# Novel\n\nIt was day.\n")
	s.toggleEdit()
	data, _ = os.ReadFile(p)
	if got, _ := zipFile(t, data, "Script.textbundle/assets/source.md"); got != "# Novel\n\nIt was day.\n" {
		t.Errorf("edited source: %q", got)
	}
	if md, _ := w.sourceText(w.sourceRef()); md != "# Novel\n\nIt was day.\n" {
		t.Errorf("source read %q", md)
	}
	// reopened, it is found again
	w2, _ := newDialogTestWindow(t)
	w2.LoadFile(p)
	if r := w2.sourceRef(); r.Origin != sourceBundle {
		t.Errorf("reopened: %+v", r)
	}
	// saved as plain Fountain, the source is kept by Accolade
	plain := filepath.Join(dir, "Script.fountain")
	if err := w2.saveToFile(plain); err != nil {
		t.Fatal(err)
	}
	w2.currentFile = plain
	if r := w2.sourceRef(); r.Origin != sourceKept {
		t.Errorf("as plain: %+v", r)
	} else if md, _ := w2.sourceText(r); md != "# Novel\n\nIt was day.\n" {
		t.Errorf("as plain: %q", md)
	}
}

// TextBundle's own samples (github.com/shinyfrog/TextBundle, MIT; in
// testdata/textbundle with its licence): the .textpack opens and saves
// with only its text changed; the folders open, or say what is missing.
func TestTextBundleSamples(t *testing.T) {
	src := filepath.Join("testdata", "textbundle", "textpack sample.textpack")
	orig, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	w, _ := newDialogTestWindow(t)
	// named as its folder: saved, a zip's folder takes the file's name
	// (Lexington 1.46, as Highland names it)
	p := filepath.Join(t.TempDir(), "Textbundle Example.textpack")
	os.WriteFile(p, orig, 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if !w.markdown || !strings.HasPrefix(w.textEditor.Text(), "# Textbundle Example") || w.bundle.Prefix != "Textbundle Example.textbundle/" {
		t.Fatalf("opened: markdown %v, %q", w.markdown, w.textEditor.Text())
	}
	w.textEditor.SetText(w.textEditor.Text() + "\nOne more line.\n")
	if err := w.SaveFile(); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(p)
	sameExceptText(t, orig, after, "Textbundle Example.textbundle/text.markdown", false)

	for _, tc := range []struct {
		dir, text, err string
	}{
		{"only text.textbundle", "Text", ""},
		{"invalid no info.textbundle", "Text", ""},
		{"invalid no text.textbundle", "", "no text file"},
	} {
		w, _ := newDialogTestWindow(t)
		err := w.LoadFile(filepath.Join("testdata", "textbundle", tc.dir))
		switch {
		case tc.err != "" && (err == nil || !strings.Contains(err.Error(), tc.err)):
			t.Errorf("%s: error %v", tc.dir, err)
		case tc.err == "" && (err != nil || w.textEditor.Text() != tc.text || !w.markdown):
			t.Errorf("%s: %v, %q, markdown %v", tc.dir, err, w.textEditor.Text(), w.markdown)
		}
	}
}

// Real Highland files put in testdata/highland (not committed: others'
// scripts) open, and save with every entry but the text as it was.
func TestHighlandFiles(t *testing.T) {
	ents, _ := os.ReadDir(filepath.Join("testdata", "highland"))
	n := 0
	for _, e := range ents {
		if e.IsDir() || bundleKindOf(e.Name()) != bundleZip {
			continue
		}
		n++
		t.Run(e.Name(), func(t *testing.T) {
			orig, err := os.ReadFile(filepath.Join("testdata", "highland", e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			p := filepath.Join(t.TempDir(), e.Name())
			os.WriteFile(p, orig, 0o644)
			w, _ := newDialogTestWindow(t)
			if err := w.LoadFile(p); err != nil {
				t.Fatal(err)
			}
			if w.bundle == nil { // plain text named .highland
				t.Logf("plain text, %d bytes", len(orig))
				return
			}
			t.Logf("%s: layout %q, %s, markdown %v, source %q", e.Name(), w.bundle.Prefix, w.bundle.textName, w.markdown, w.bundle.source)
			w.textEditor.SetText(w.textEditor.Text() + "\n")
			if err := w.SaveFile(); err != nil {
				t.Fatal(err)
			}
			after, _ := os.ReadFile(p)
			sameExceptText(t, orig, after, w.bundle.Prefix+w.bundle.textName, false)
		})
	}
	if n == 0 {
		t.Skip("no .highland files in testdata/highland")
	}
}

// A rename keeps a bundle a bundle: to another zip's name, the file is
// renamed; to a plain name it becomes plain; a plain script given a
// bundle's name is written as one.
func TestBundleRename(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	dir := t.TempDir()
	p := filepath.Join(dir, "Script.highland")
	os.WriteFile(p, makeZip(t, bundleItems("")), 0o644)
	w.LoadFile(p)
	if err := w.renameTo("Script.textpack"); err != nil {
		t.Fatal(err)
	}
	if w.bundle.Path != filepath.Join(dir, "Script.textpack") {
		t.Errorf("bundle at %s", w.bundle.Path)
	}
	// to a plain name: converted (relocate.go), never refused
	if err := w.renameTo("Script.fountain"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "Script.fountain")); string(got) != bundleScript || w.bundle != nil {
		t.Errorf("renamed to plain: %q, bundle %v", got, w.bundle)
	}
	if _, err := os.Stat(filepath.Join(dir, "Script.textpack")); err == nil {
		t.Error("the bundle is still there")
	}
	plain := filepath.Join(dir, "Plain.fountain")
	os.WriteFile(plain, []byte(bundleScript), 0o644)
	w.LoadFile(plain)
	if err := w.renameTo("Plain.highland"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(plain); err == nil {
		t.Error("the plain file is still there")
	}
	data, _ := os.ReadFile(filepath.Join(dir, "Plain.highland"))
	if got, _ := zipFile(t, data, "Plain.textbundle/text.fountain"); got != bundleScript {
		t.Errorf("renamed into a bundle: %q", got)
	}
}

// What another app changed in the bundle since it was opened is kept
// when it is saved (the same bundle still: its layout and text file).
func TestBundleKeepsOthersChanges(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	p := filepath.Join(t.TempDir(), "Script.highland")
	os.WriteFile(p, makeZip(t, bundleItems("Script.textbundle/")), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	items := bundleItems("Script.textbundle/")
	for i := range items {
		if strings.HasSuffix(items[i].name, "characters.json") {
			items[i].data = `{"BRAM":{"color":"#00ff00"}}`
		}
	}
	changed := makeZip(t, items)
	os.WriteFile(p, changed, 0o644) // Highland saved it meanwhile
	w.textEditor.SetText(bundleScript + "Later.\n")
	if err := w.SaveFile(); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(p)
	sameExceptText(t, changed, after, "Script.textbundle/text.fountain", false)
}

// TextBundle's sample with an image (not committed: 160 KB), from a
// checkout of github.com/shinyfrog/TextBundle named by
// ACCOLADE_TEXTBUNDLE_SAMPLES (its "TextBundleTests/Sample TextBundles").
func TestTextBundleSampleWithImage(t *testing.T) {
	dir := os.Getenv("ACCOLADE_TEXTBUNDLE_SAMPLES")
	if dir == "" {
		t.Skip("ACCOLADE_TEXTBUNDLE_SAMPLES not set")
	}
	src := filepath.Join(dir, "text plus attachments.textbundle")
	jpg, err := os.ReadFile(filepath.Join(src, "assets", "oh no.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	bundle := filepath.Join(tmp, "attachments.textbundle")
	os.MkdirAll(filepath.Join(bundle, "assets"), 0o755)
	for _, n := range []string{"info.json", "text.md", "assets/oh no.jpg"} {
		b, _ := os.ReadFile(filepath.Join(src, n))
		os.WriteFile(filepath.Join(bundle, n), b, 0o644)
	}
	w, _ := newDialogTestWindow(t)
	if err := w.LoadFile(bundle); err != nil || !w.markdown {
		t.Fatalf("%v, markdown %v", err, w.markdown)
	}
	w.textEditor.SetText(w.textEditor.Text() + "\nMore.\n")
	if err := w.SaveFile(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(bundle, "assets", "oh no.jpg")); !bytes.Equal(b, jpg) {
		t.Error("the image changed")
	}
	pack := filepath.Join(tmp, "attachments.textpack")
	if err := w.saveToFile(pack); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(pack)
	if got, _ := zipFile(t, data, "attachments.textbundle/assets/oh no.jpg"); got != string(jpg) {
		t.Error("the image in the .textpack differs")
	}
	w2, _ := newDialogTestWindow(t)
	if err := w2.LoadFile(pack); err != nil || !strings.HasSuffix(w2.textEditor.Text(), "More.\n") {
		t.Errorf("the .textpack reopened: %v", err)
	}
}

// relContents are a bundle's files by their name in it, macOS's junk and
// the carried settings left out: the same for its zip and its folder.
func relContents(t *testing.T, p string) (map[string]string, bool) {
	t.Helper()
	b, err := lb.Read(p)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]string{}
	for _, n := range b.Files {
		if lb.IsJunk(n) || isCarriedEntry(n) { // the settings an .accolade carries: as the machine's are
			continue
		}
		data, _ := b.ReadFile(n)
		m[n] = string(data)
	}
	return m, b.Zipped
}

// An .accolade unzipped (the Accolade folder) and zipped again: Save As
// chooses the form, Save keeps it, Open reads the folder (itself or its
// text), the rename editor changes the form in place; every file kept.
func TestBundleFolderForms(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	dir := t.TempDir()
	p := filepath.Join(dir, "Script.accolade")
	os.WriteFile(p, makeZip(t, bundleItems("Script.textbundle/")), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	want, _ := relContents(t, p)
	folder := filepath.Join(dir, "Folder.accolade")
	w.saveAsWith(folder, &relocation{form: bundleFolder})
	got, zipped := relContents(t, folder)
	if zipped || w.bundle == nil || w.bundle.Zipped || w.currentFile != folder {
		t.Fatalf("not a folder: zipped %v, file %s", zipped, w.currentFile)
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("folder:\n%v\nwant\n%v", got, want)
	}
	if got := typeOfName(folder, false); got.id != typeAccoladeFolder.id {
		t.Errorf("type of the folder: %s", got.id)
	}
	// Save keeps the folder
	w.textEditor.SetText(bundleScript + "\nBram eats.\n")
	if err := w.SaveFile(); err != nil {
		t.Fatal(err)
	}
	got, zipped = relContents(t, folder)
	if zipped || got["text.fountain"] != bundleScript+"\nBram eats.\n" {
		t.Errorf("saved: zipped %v, %q", zipped, got["text.fountain"])
	}
	want["text.fountain"] = got["text.fountain"]
	// opened as the folder, and by its text
	for _, open := range []string{folder, filepath.Join(folder, "text.fountain")} {
		w2, _ := newDialogTestWindow(t)
		if err := w2.LoadFile(open); err != nil || w2.bundle == nil || w2.currentFile != folder || !strings.Contains(w2.textEditor.Text(), "Bram eats.") {
			t.Errorf("opened %s: %v, %s", open, err, w2.currentFile)
		}
	}
	// zipped again by Save As
	again := filepath.Join(dir, "Again.accolade")
	w.saveAsWith(again, &relocation{form: bundleZip})
	if got, zipped := relContents(t, again); !zipped || fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("zipped again (%v):\n%v", zipped, got)
	}
	// the rename editor's choice, in place: unzipped where it is
	err := w.relocate(relocation{rename: true, from: again, to: again, form: bundleFolder}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, zipped := relContents(t, again); zipped || fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("unzipped in place (%v):\n%v", zipped, got)
	}
	// Save As offers the folder on the desktop only
	ts := scriptTypes(w.lex(), false, false, "x.fountain", "")
	if !slices.ContainsFunc(ts, func(t fileType) bool { return t.id == typeAccoladeFolder.id }) {
		t.Error("no Accolade folder in Save As")
	}
	if slices.ContainsFunc(scriptTypes(w.lex(), false, true, "x.fountain", ""), func(t fileType) bool { return t.kind == bundleFolder }) {
		t.Error("a folder type in the browser")
	}
}

// A folder of any name holding a text.fountain opens as a bundle (Open
// Bundle Folder…) and saves back into itself.
func TestBundleAnyFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "My Script")
	os.Mkdir(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "text.fountain"), []byte(bundleScript), 0o644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("mine"), 0o644)
	w, _ := newDialogTestWindow(t)
	if err := w.LoadFile(dir); err != nil || w.bundle == nil || w.textEditor.Text() != bundleScript {
		t.Fatalf("opened: %v %q", err, w.textEditor.Text())
	}
	w.textEditor.SetText(bundleScript + "\nMore.\n")
	if err := w.SaveFile(); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "text.fountain")); string(data) != bundleScript+"\nMore.\n" {
		t.Errorf("saved %q", data)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "notes.txt")); string(data) != "mine" {
		t.Error("notes.txt lost")
	}
}

// A bundle keeps its form when no other type is chosen: renamed by name,
// or confirmed unchanged in the rename editor (a folder .highland, a
// zipped .textbundle).
func TestBundleKeepsItsForm(t *testing.T) {
	dir := t.TempDir()
	// an Accolade folder renamed: still a folder
	w, _ := newDialogTestWindow(t)
	a := filepath.Join(dir, "A.accolade")
	if _, err := lb.Write(nil, a, lb.Dir, lb.New([]byte(bundleScript), false, ""), nil); err != nil {
		t.Fatal(err)
	}
	if err := w.LoadFile(a); err != nil {
		t.Fatal(err)
	}
	if err := w.renameTo("B.accolade"); err != nil {
		t.Fatal(err)
	}
	if _, zipped := relContents(t, filepath.Join(dir, "B.accolade")); zipped || exists(a) {
		t.Errorf("renamed: zipped %v, the old one there %v", zipped, exists(a))
	}
	// the rename editor confirmed as it opened, in another language
	for _, tc := range []struct {
		name string
		form lb.Form
	}{{"H.highland", lb.Dir}, {"T.textbundle", lb.Zip}} {
		w, _ := newDialogTestWindow(t)
		p := filepath.Join(dir, tc.name)
		if _, err := lb.Write(nil, p, tc.form, lb.New([]byte(bundleScript), false, ""), nil); err != nil {
			t.Fatal(err)
		}
		if err := w.LoadFile(p); err != nil {
			t.Fatal(err)
		}
		r := relocation{rename: true, from: p, to: p, lang: "eo"}
		if err := w.relocate(r, nil); err != nil {
			t.Fatal(err)
		}
		if b, err := lb.Read(p); err != nil || b.Form() != tc.form {
			t.Errorf("%s: %v, %v", tc.name, err, b)
		}
	}
}
