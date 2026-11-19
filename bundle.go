package main

import (
	"archive/zip"
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// The TextBundle family (docs/HIGHLAND.md): a .textbundle is a folder
// (info.json, text.fountain or text.md, assets/), a .textpack the same
// zipped, and Highland's .highland a zipped TextBundle of its own. In a
// zip the bundle's files sit at the root or inside one top-level
// Name.textbundle/ folder; a bundle is written back as it was read.
//
// Saving changes only the text file and Accolade's own key in info.json;
// every other entry (assets/, Highland's JSON, files no one here knows) is
// copied through byte for byte, in its order. The one other entry Accolade
// writes is the script's source when it is kept in the bundle (source.go's
// layers): assets/source.md, named in Accolade's key.

// accoladeInfoKey is Accolade's key in info.json.
const accoladeInfoKey = "eu.kiefte.Accolade"

// fountainUTI and markdownUTI are info.json's "type" for a new bundle.
const (
	fountainUTI = "com.quoteunquoteapps.fountain"
	markdownUTI = "net.daringfireball.markdown"
)

// bundleSourceName is where a source put in the bundle goes.
const bundleSourceName = "assets/source.md"

type bundleKind int

const (
	notBundle    bundleKind = iota
	bundleFolder            // .textbundle
	bundleZip               // .textpack, .highland
)

// bundleKindOf is the kind of bundle a name is.
func bundleKindOf(name string) bundleKind {
	switch strings.ToLower(filepath.Ext(strings.TrimRight(name, `/\`))) {
	case ".textbundle":
		return bundleFolder
	case ".textpack", ".highland":
		return bundleZip
	}
	return notBundle
}

// errNotZip: the file is no zip (a .highland that is plain text).
var errNotZip = errors.New("not a zip file")

// accoladeInfo is what Accolade keeps under its key.
type accoladeInfo struct {
	Version int `json:"version"`
	// DocumentType is "markdown" or "screenplay" when the document is not
	// what the text file's name and "type" say (View > Document Type).
	DocumentType string `json:"documentType,omitempty"`
	// Source is the bundle's entry holding the script's source.
	Source string `json:"source,omitempty"`
}

func (a accoladeInfo) empty() bool { return a.DocumentType == "" && a.Source == "" }

// textBundle is a bundle as read: where it is and what it holds besides
// the text.
type textBundle struct {
	path   string
	zipped bool
	raw    []byte // a zip as read or written last: what is copied through
	// prefix is the folder holding the bundle in a zip ("Name.textbundle/"),
	// "" for files at its root
	prefix   string
	textName string // the text file, in the bundle ("text.fountain")
	info     []byte // info.json as it is, nil for none
	natural  bool   // Markdown, as the text file's name and "type" say
	ours     accoladeInfo
	source   string // the source's entry, "" for none
	srcText  []byte // and what it holds
}

// markdown tells whether the bundle's document is Markdown.
func (b *textBundle) markdown() bool {
	switch b.ours.DocumentType {
	case "markdown":
		return true
	case "screenplay":
		return false
	}
	return b.natural
}

// bundleEntry is one file or folder in a bundle as found: name is its
// path in the zip (prefix included) or in the folder, '/' between, a
// folder's ending in '/'.
type bundleEntry struct {
	name string
	zf   *zip.File // in a zip: copied through raw
	file string    // in a folder: the file on disk
}

// bundleFile is a file Accolade writes into a bundle (name relative to
// the bundle; data nil and a name ending in '/' for a folder).
type bundleFile struct {
	name string
	data []byte
}

// junkEntry is what macOS leaves in zips; never the bundle's.
func junkEntry(name string) bool {
	return strings.HasPrefix(name, "__MACOSX/") || path.Base(name) == ".DS_Store"
}

// bundleOf is the bundle a file is the text of (X.textbundle/text.md:
// X.textbundle), as a file picked in a folder bundle opens the bundle.
func bundleOf(p string) (string, bool) {
	dir := filepath.Dir(p)
	if bundleKindOf(dir) != bundleFolder || !isTextName(filepath.Base(p)) {
		return "", false
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return "", false
	}
	return dir, true
}

func isTextName(name string) bool {
	return strings.HasPrefix(strings.ToLower(name), "text.") && len(name) > len("text.")
}

// pickText is the bundle's text among the files at its root: text.fountain,
// text.md, text.markdown, text.txt, else another text.*.
func pickText(names []string) string {
	rank := func(n string) int {
		switch strings.ToLower(path.Ext(n)) {
		case ".fountain":
			return 0
		case ".md":
			return 1
		case ".markdown":
			return 2
		case ".txt":
			return 3
		}
		return 4
	}
	best := ""
	for _, n := range names {
		if !isTextName(n) {
			continue
		}
		if best == "" || rank(n) < rank(best) || (rank(n) == rank(best) && n < best) {
			best = n
		}
	}
	return best
}

// readBundle reads a bundle: what it holds besides the text, and the text.
func readBundle(p string) (*textBundle, []byte, error) {
	st, err := os.Stat(p)
	if err != nil {
		return nil, nil, err
	}
	if st.IsDir() {
		return readFolderBundle(p)
	}
	if bundleKindOf(p) == bundleFolder {
		return nil, nil, fmt.Errorf("%s is not a folder", filepath.Base(p))
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, nil, err
	}
	return parseZipBundle(p, data)
}

// parseZipBundle reads a zipped bundle from its bytes.
func parseZipBundle(p string, data []byte) (*textBundle, []byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, nil, errNotZip
	}
	b := &textBundle{path: p, zipped: true, raw: data}
	files := map[string]*zip.File{}
	tops := map[string]bool{}
	hasRootFile := false
	for _, f := range zr.File {
		if junkEntry(f.Name) {
			continue
		}
		top, _, nested := strings.Cut(f.Name, "/")
		if nested {
			tops[top] = true
		}
		if f.FileInfo().IsDir() {
			continue
		}
		files[f.Name] = f
		hasRootFile = hasRootFile || !nested
	}
	// the layout: files at the root, else one top-level folder (the
	// *.textbundle one if there are several)
	if !hasRootFile {
		var cands []string
		for t := range tops {
			cands = append(cands, t)
		}
		if len(cands) > 1 {
			cands = slices.DeleteFunc(cands, func(t string) bool { return bundleKindOf(t) != bundleFolder })
		}
		if len(cands) == 1 {
			b.prefix = cands[0] + "/"
		}
	}
	var root []string
	for name := range files {
		if rel, ok := strings.CutPrefix(name, b.prefix); ok && !strings.Contains(rel, "/") {
			root = append(root, rel)
		}
	}
	read := func(rel string) ([]byte, error) {
		f := files[b.prefix+rel]
		if f == nil {
			return nil, fs.ErrNotExist
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer r.Close()
		return io.ReadAll(r)
	}
	return b.fill(p, root, read)
}

// readFolderBundle reads a .textbundle folder.
func readFolderBundle(dir string) (*textBundle, []byte, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	var root []string
	for _, e := range ents {
		if !e.IsDir() {
			root = append(root, e.Name())
		}
	}
	b := &textBundle{path: dir}
	return b.fill(dir, root, func(rel string) ([]byte, error) {
		return os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	})
}

// fill reads the bundle's text, info.json and source, given the files at
// its root and how to read one.
func (b *textBundle) fill(p string, root []string, read func(string) ([]byte, error)) (*textBundle, []byte, error) {
	b.textName = pickText(root)
	if b.textName == "" {
		return nil, nil, fmt.Errorf("%s holds no text file (text.fountain, text.md…)", filepath.Base(p))
	}
	text, err := read(b.textName)
	if err != nil {
		return nil, nil, err
	}
	for _, n := range root {
		if strings.EqualFold(n, "info.json") {
			if b.info, err = read(n); err != nil {
				return nil, nil, err
			}
			if b.info == nil {
				b.info = []byte{}
			}
		}
	}
	if raw, ok := infoValue(b.info, accoladeInfoKey); ok {
		_ = json.Unmarshal(raw, &b.ours)
	}
	b.natural = bundleTextMarkdown(b.textName, b.info, p)
	// the source: as Accolade's key names it, else one where Accolade or
	// a person would put it
	for _, n := range []string{b.ours.Source, bundleSourceName, "source.md", "source.markdown"} {
		if n == "" || !fs.ValidPath(n) || strings.EqualFold(n, b.textName) {
			continue
		}
		if data, err := read(n); err == nil {
			b.source, b.srcText = n, data
			break
		}
	}
	return b, text, nil
}

// bundleTextMarkdown tells whether a bundle's text is Markdown: as its
// name says (text.md, text.fountain), else as info.json's "type" does,
// else Highland's are screenplays and other TextBundles Markdown (the
// spec's default type).
func bundleTextMarkdown(textName string, info []byte, bundle string) bool {
	switch strings.ToLower(path.Ext(textName)) {
	case ".md", ".markdown":
		return true
	case ".fountain", ".spmd":
		return false
	}
	if raw, ok := infoValue(info, "type"); ok {
		var t string
		if json.Unmarshal(raw, &t) == nil {
			t = strings.ToLower(t)
			switch {
			case strings.Contains(t, "fountain"):
				return false
			case strings.Contains(t, "markdown"):
				return true
			}
		}
	}
	return !strings.EqualFold(filepath.Ext(bundle), ".highland")
}

// newBundleInfo is info.json for a bundle Accolade makes.
func newBundleInfo(markdown bool) []byte {
	t := fountainUTI
	if markdown {
		t = markdownUTI
	}
	return []byte(`{
  "version": 2,
  "type": "` + t + `",
  "transient": false,
  "creatorIdentifier": "` + accoladeInfoKey + `"
}
`)
}

// saveBundleText writes the bundle (b; nil for a new one) to dst with text
// as its text; markdown is the document's type, kept in Accolade's key if
// the bundle would say otherwise. The bundle as written is returned.
func saveBundleText(b *textBundle, dst string, text []byte, markdown bool) (*textBundle, error) {
	if b == nil {
		textName := "text.fountain"
		if markdown {
			textName = "text.md"
		}
		files := []bundleFile{{"info.json", newBundleInfo(markdown)}, {textName, text}, {"assets/", nil}}
		return writeBundle(nil, dst, files)
	}
	if sameFile(dst, b.path) {
		b = b.fresh()
	}
	ours := b.ours
	ours.DocumentType = ""
	if markdown != b.natural {
		ours.DocumentType = map[bool]string{true: "markdown", false: "screenplay"}[markdown]
	}
	files := []bundleFile{{b.textName, text}}
	if info, changed := b.infoWith(ours); changed {
		files = append(files, bundleFile{b.infoName(), info})
	}
	return writeBundle(b, dst, files)
}

// saveBundleSource writes md as the bundle's source, in place, the text
// as it is in the bundle.
func saveBundleSource(b *textBundle, md []byte) (*textBundle, error) {
	b = b.fresh()
	name := cmp.Or(b.source, bundleSourceName)
	ours := b.ours
	ours.Source = name
	files := []bundleFile{{name, md}}
	if info, changed := b.infoWith(ours); changed {
		files = append(files, bundleFile{b.infoName(), info})
	}
	return writeBundle(b, b.path, files)
}

// fresh is the bundle as it is on disk now, if it is still the same
// bundle (its layout and text file): what another app changed in it since
// is kept. Else (gone, emptied by a save dialog, made over) as read.
func (b *textBundle) fresh() *textBundle {
	nb, _, err := readBundle(b.path)
	if err != nil || nb.zipped != b.zipped || nb.prefix != b.prefix || nb.textName != b.textName {
		return b
	}
	return nb
}

// infoName is info.json's name as the bundle has it.
func (b *textBundle) infoName() string {
	if b.zipped {
		if zr, err := zip.NewReader(bytes.NewReader(b.raw), int64(len(b.raw))); err == nil {
			for _, f := range zr.File {
				if rel, ok := strings.CutPrefix(f.Name, b.prefix); ok && strings.EqualFold(rel, "info.json") {
					return rel
				}
			}
		}
	} else if ents, err := os.ReadDir(b.path); err == nil {
		for _, e := range ents {
			if strings.EqualFold(e.Name(), "info.json") {
				return e.Name()
			}
		}
	}
	return "info.json"
}

// infoWith is info.json with ours as Accolade's key, and whether that is
// a change. Nothing else in it changes; an info.json that cannot be read
// is left as it is.
func (b *textBundle) infoWith(ours accoladeInfo) ([]byte, bool) {
	norm := func(a accoladeInfo) accoladeInfo {
		if a.empty() {
			return accoladeInfo{}
		}
		a.Version = max(a.Version, 1)
		return a
	}
	ours = norm(ours)
	_, has := infoValue(b.info, accoladeInfoKey)
	if has == !ours.empty() && (!has || ours == norm(b.ours)) {
		return b.info, false // nothing new to say
	}
	var val []byte
	if !ours.empty() {
		val, _ = json.Marshal(ours)
	}
	info := b.info
	if info == nil { // a bundle without one: TextBundle's own
		info = newBundleInfo(b.natural)
	}
	info, ok := setInfoValue(info, accoladeInfoKey, val)
	if !ok || bytes.Equal(info, b.info) {
		return b.info, false
	}
	return info, true
}

// writeBundle writes a bundle to dst: b's entries (none for a new
// bundle) copied through, files replacing theirs or added after them.
func writeBundle(b *textBundle, dst string, files []bundleFile) (*textBundle, error) {
	if bundleKindOf(dst) == bundleFolder {
		if err := writeFolderBundle(b, dst, files); err != nil {
			return nil, err
		}
		nb, _, err := readFolderBundle(dst)
		return nb, err
	}
	// the layout: as read for the same file; a zip saved as another one
	// keeps its layout, its folder named for the new file; a new zip has
	// its files in Name.textbundle/, as TextBundle's own sample and
	// writer do (docs/HIGHLAND.md)
	folder := strings.TrimSuffix(filepath.Base(dst), filepath.Ext(dst)) + ".textbundle/"
	prefix := folder
	if b != nil && b.zipped {
		if sameFile(b.path, dst) || b.prefix == "" {
			prefix = b.prefix
		}
	}
	data, err := zipBundle(b, prefix, files)
	if err != nil {
		return nil, err
	}
	if err := writeFileAtomic(dst, data); err != nil {
		return nil, err
	}
	nb, _, err := parseZipBundle(dst, data)
	return nb, err
}

// entries are the bundle's files and folders, in their order.
func (b *textBundle) entries() ([]bundleEntry, error) {
	if b == nil {
		return nil, nil
	}
	if b.zipped {
		zr, err := zip.NewReader(bytes.NewReader(b.raw), int64(len(b.raw)))
		if err != nil {
			return nil, err
		}
		var es []bundleEntry
		for _, f := range zr.File {
			es = append(es, bundleEntry{name: f.Name, zf: f})
		}
		return es, nil
	}
	var es []bundleEntry
	err := filepath.WalkDir(b.path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(b.path, p)
		if rel == "." {
			return nil
		}
		name := filepath.ToSlash(rel)
		if d.IsDir() {
			name += "/"
		} else if !d.Type().IsRegular() {
			return nil
		}
		es = append(es, bundleEntry{name: name, file: p})
		return nil
	})
	return es, err
}

// zipBundle is the zip of a bundle, its files in prefix.
func zipBundle(b *textBundle, prefix string, files []bundleFile) ([]byte, error) {
	es, err := b.entries()
	if err != nil {
		return nil, err
	}
	oldPrefix := ""
	if b != nil && b.zipped {
		oldPrefix = b.prefix
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if b != nil && b.zipped {
		if zr, err := zip.NewReader(bytes.NewReader(b.raw), int64(len(b.raw))); err == nil && zr.Comment != "" {
			if err := zw.SetComment(zr.Comment); err != nil {
				return nil, err
			}
		}
	}
	pending := slices.Clone(files)
	take := func(rel string) (bundleFile, bool) {
		for i, f := range pending {
			if f.name == rel {
				pending = slices.Delete(pending, i, i+1)
				return f, true
			}
		}
		return bundleFile{}, false
	}
	now := time.Now()
	write := func(name string, data []byte, like *zip.FileHeader) error {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: now}
		if strings.HasSuffix(name, "/") {
			h.Method = zip.Store
			h.SetMode(fs.ModeDir | 0o755)
		} else {
			h.SetMode(0o644)
		}
		if like != nil {
			h.CreatorVersion, h.ExternalAttrs = like.CreatorVersion, like.ExternalAttrs
			if like.Method == zip.Store && !strings.HasSuffix(name, "/") {
				h.Method = zip.Store // stored as it was; any other method: deflated
			}
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
	if b == nil && prefix != "" {
		if err := write(prefix, nil, nil); err != nil {
			return nil, err
		}
	}
	for _, e := range es {
		name := e.name
		rel, inside := strings.CutPrefix(name, oldPrefix)
		if b != nil && !b.zipped {
			rel, inside = name, true
		}
		if inside {
			name = prefix + rel
		}
		if inside && rel != "" {
			if f, ok := take(rel); ok {
				var like *zip.FileHeader
				if e.zf != nil {
					like = &e.zf.FileHeader
				}
				if err := write(name, f.data, like); err != nil {
					return nil, err
				}
				continue
			}
		}
		switch {
		case e.zf != nil && name == e.name:
			if err := zw.Copy(e.zf); err != nil {
				return nil, err
			}
		case e.zf != nil:
			h := e.zf.FileHeader
			h.Name = name
			r, err := e.zf.OpenRaw()
			if err != nil {
				return nil, err
			}
			w, err := zw.CreateRaw(&h)
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(w, r); err != nil {
				return nil, err
			}
		case strings.HasSuffix(name, "/"):
			if err := write(name, nil, nil); err != nil {
				return nil, err
			}
		default:
			data, err := os.ReadFile(e.file)
			if err != nil {
				return nil, err
			}
			if err := write(name, data, nil); err != nil {
				return nil, err
			}
		}
	}
	for _, f := range pending {
		if err := write(prefix+f.name, f.data, nil); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeFolderBundle writes a .textbundle folder: in place, only files
// change; elsewhere a new folder with b's entries and files, put in place
// whole (replacing an empty file the save dialog left, or a bundle chosen
// to be overwritten).
func writeFolderBundle(b *textBundle, dst string, files []bundleFile) error {
	if b != nil && !b.zipped && sameFile(b.path, dst) {
		for _, f := range files {
			if err := writeBundleFile(dst, f); err != nil {
				return err
			}
		}
		return nil
	}
	tmp := dst + ".accolade-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if err := os.Mkdir(tmp, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(tmp) // nothing left once renamed
	es, err := b.entries()
	if err != nil {
		return err
	}
	oldPrefix := ""
	if b != nil && b.zipped {
		oldPrefix = b.prefix
	}
	for _, e := range es {
		rel, inside := strings.CutPrefix(e.name, oldPrefix)
		if e.zf == nil {
			rel, inside = e.name, true
		}
		if !inside || rel == "" || junkEntry(e.name) {
			continue
		}
		if slices.ContainsFunc(files, func(f bundleFile) bool { return f.name == rel }) {
			continue
		}
		var data []byte
		switch {
		case strings.HasSuffix(rel, "/"):
		case e.zf != nil:
			r, err := e.zf.Open()
			if err != nil {
				return err
			}
			data, err = io.ReadAll(r)
			r.Close()
			if err != nil {
				return err
			}
		default:
			if data, err = os.ReadFile(e.file); err != nil {
				return err
			}
		}
		if err := writeBundleFile(tmp, bundleFile{rel, data}); err != nil {
			return err
		}
	}
	for _, f := range files {
		if err := writeBundleFile(tmp, f); err != nil {
			return err
		}
	}
	if st, err := os.Lstat(dst); err == nil {
		switch {
		case st.Mode().IsRegular() && st.Size() == 0: // the save dialog's
			if err := os.Remove(dst); err != nil {
				return err
			}
		case st.IsDir():
			if _, _, err := readFolderBundle(dst); err != nil {
				return fmt.Errorf("%s is a folder but no TextBundle: not replaced", filepath.Base(dst))
			}
			old := tmp + ".old"
			if err := os.Rename(dst, old); err != nil {
				return err
			}
			if err := os.Rename(tmp, dst); err != nil {
				_ = os.Rename(old, dst)
				return err
			}
			return os.RemoveAll(old)
		default:
			return fmt.Errorf("%s exists and is no TextBundle folder", filepath.Base(dst))
		}
	}
	return os.Rename(tmp, dst)
}

// writeBundleFile writes one file (or folder) into a bundle folder.
func writeBundleFile(dir string, f bundleFile) error {
	if !fs.ValidPath(strings.TrimSuffix(f.name, "/")) {
		return fmt.Errorf("%q: no name for a file in a bundle", f.name)
	}
	p := filepath.Join(dir, filepath.FromSlash(f.name))
	if strings.HasSuffix(f.name, "/") {
		return os.MkdirAll(p, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return writeFileAtomic(p, f.data)
}

// info.json is edited as text: its members are found and only Accolade's
// value is replaced (or added, or removed), so the other keys keep their
// order and their bytes (TextBundle's own samples have a trailing comma,
// which encoding/json refuses).

type jsonMember struct {
	key                        string
	keyStart, valStart, valEnd int
}

// jsonObjectMembers are a JSON object's members, and where it closes.
func jsonObjectMembers(b []byte) (ms []jsonMember, open, close int, ok bool) {
	i := skipJSONSpace(b, 0)
	if i >= len(b) || b[i] != '{' {
		return nil, 0, 0, false
	}
	open = i
	i++
	for {
		i = skipJSONSpace(b, i)
		if i >= len(b) {
			return nil, 0, 0, false
		}
		switch b[i] {
		case '}':
			return ms, open, i, true
		case ',':
			i++
			continue
		case '"':
		default:
			return nil, 0, 0, false
		}
		ke := scanJSONString(b, i)
		if ke < 0 {
			return nil, 0, 0, false
		}
		var key string
		if json.Unmarshal(b[i:ke], &key) != nil {
			return nil, 0, 0, false
		}
		j := skipJSONSpace(b, ke)
		if j >= len(b) || b[j] != ':' {
			return nil, 0, 0, false
		}
		vs := skipJSONSpace(b, j+1)
		ve := scanJSONValue(b, vs)
		if ve < 0 {
			return nil, 0, 0, false
		}
		ms = append(ms, jsonMember{key, i, vs, ve})
		i = ve
	}
}

func skipJSONSpace(b []byte, i int) int {
	for i < len(b) && (b[i] == ' ' || b[i] == '\t' || b[i] == '\n' || b[i] == '\r') {
		i++
	}
	return i
}

// scanJSONString is the end of the string starting at i, -1 if none.
func scanJSONString(b []byte, i int) int {
	for j := i + 1; j < len(b); j++ {
		switch b[j] {
		case '\\':
			j++
		case '"':
			return j + 1
		}
	}
	return -1
}

// scanJSONValue is the end of the value starting at i, -1 if none.
func scanJSONValue(b []byte, i int) int {
	if i >= len(b) {
		return -1
	}
	switch b[i] {
	case '"':
		return scanJSONString(b, i)
	case '{', '[':
		depth := 0
		for j := i; j < len(b); j++ {
			switch b[j] {
			case '"':
				if j = scanJSONString(b, j); j < 0 {
					return -1
				}
				j--
			case '{', '[':
				depth++
			case '}', ']':
				if depth--; depth == 0 {
					return j + 1
				}
			}
		}
		return -1
	}
	j := i
	for j < len(b) && !strings.ContainsRune(",}] \t\r\n", rune(b[j])) {
		j++
	}
	if j == i {
		return -1
	}
	return j
}

// infoValue is a top-level key's value in info.json, as it is written.
func infoValue(info []byte, key string) ([]byte, bool) {
	ms, _, _, ok := jsonObjectMembers(info)
	if !ok {
		return nil, false
	}
	for _, m := range ms {
		if m.key == key {
			return info[m.valStart:m.valEnd], true
		}
	}
	return nil, false
}

// setInfoValue is info.json with key's value val (nil: the key removed),
// all else as it was; ok is false if info.json cannot be read.
func setInfoValue(info []byte, key string, val []byte) ([]byte, bool) {
	ms, open, close, ok := jsonObjectMembers(info)
	if !ok {
		return info, false
	}
	splice := func(from, to int, with string) []byte {
		return slices.Concat(info[:from], []byte(with), info[to:])
	}
	for i, m := range ms {
		if m.key != key {
			continue
		}
		if val != nil {
			return splice(m.valStart, m.valEnd, string(val)), true
		}
		if i > 0 { // from the end of the one before
			return splice(ms[i-1].valEnd, m.valEnd, ""), true
		}
		if i+1 < len(ms) {
			return splice(m.keyStart, ms[i+1].keyStart, ""), true
		}
		return splice(open+1, close, "\n"), true
	}
	if val == nil {
		return info, true
	}
	k, _ := json.Marshal(key)
	member := string(k) + ": " + string(val)
	if len(ms) == 0 {
		return splice(open+1, close, "\n  "+member+"\n"), true
	}
	// laid out as the first member is
	sep := " "
	if ws := info[open+1 : ms[0].keyStart]; bytes.ContainsRune(ws, '\n') {
		sep = "\n" + string(ws[bytes.LastIndexByte(ws, '\n')+1:])
	}
	last := ms[len(ms)-1].valEnd
	if j := skipJSONSpace(info, last); j < close && info[j] == ',' { // a trailing comma
		return splice(j+1, j+1, sep+member+","), true
	}
	return splice(last, last, ","+sep+member), true
}
