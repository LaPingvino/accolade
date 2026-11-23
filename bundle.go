package main

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/LaPingvino/accolade/internal/pace"
	lb "github.com/LaPingvino/lexington/bundle"
	"github.com/LaPingvino/lexington/rules"
)

// The TextBundle family (docs/HIGHLAND.md, docs/FORMAT.md): a script's
// text file with info.json and assets/, read and written by Lexington's
// bundle package, as a zip (.textpack, .highland, .accolade) or as a
// plain folder (.textbundle, and any of the others unzipped:
// Name.accolade/ is a folder). Accolade's .accolade is a .highland with
// Accolade's drafts and production files added (extras.go). A bundle is
// written back in the form and layout it was read in; Save As and Rename
// can choose the other form.
//
// Saving changes only the text file and Accolade's own key in info.json;
// every other entry (assets/, Highland's JSON, files no one here knows) is
// copied through byte for byte, in its order. The one other entry Accolade
// writes is the script's source when it is kept in the bundle (source.go's
// layers): assets/source.md, named in Accolade's key.

func init() { lb.Yield = pace.Yield } // the browser's page draws while a zip is written (bundle_queue.go)

// accoladeInfoKey is Accolade's key in info.json.
const accoladeInfoKey = "eu.kiefte.Accolade"

// bundleSourceName is where a source put in the bundle goes.
const bundleSourceName = "assets/source.md"

type bundleKind int

const (
	notBundle    bundleKind = iota
	bundleFolder            // a folder: .textbundle, or Name.accolade/ unzipped
	bundleZip               // a zip: .textpack, .highland, .accolade
)

// bundleKindOf is the kind of bundle a name is: none for a name no bundle
// has; else a folder or a zip as it is on disk, and for a name not there
// yet as its type is (a .textbundle a folder, the others zips).
func bundleKindOf(name string) bundleKind {
	if !lb.Is(name) {
		return notBundle
	}
	return kindOfForm(lb.FormOf(name))
}

func kindOfForm(f lb.Form) bundleKind {
	if f == lb.Dir {
		return bundleFolder
	}
	return bundleZip
}

// form is the bundle form a kind asks for (notBundle: as it is, Auto).
func (k bundleKind) form() lb.Form {
	switch k {
	case bundleFolder:
		return lb.Dir
	case bundleZip:
		return lb.Zip
	}
	return lb.Auto
}

// isBundlePath tells whether a path is opened as a bundle: a bundle's
// name, or any folder (Open Bundle Folder…).
func isBundlePath(p string) bool {
	if lb.Is(p) {
		return true
	}
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// errNotZip: the file is no zip (a .highland that is plain text).
var errNotZip = lb.ErrNotZip

// accoladeInfo is what Accolade keeps under its key.
type accoladeInfo struct {
	Version int `json:"version"`
	// DocumentType is "markdown" or "screenplay" when the document is not
	// what the text file's name and "type" say (View > Document Type).
	DocumentType string `json:"documentType,omitempty"`
	// Source is the bundle's entry holding the script's source.
	Source string `json:"source,omitempty"`
	// Drafts is the drafts' index (drafts.go), when there are drafts.
	Drafts string `json:"drafts,omitempty"`
	// HighlandWarned: Save as .highland has said once that Highland may
	// drop Accolade's files (flatten.go).
	HighlandWarned bool `json:"highlandWarned,omitempty"`
	// Language is a screenplay's language, other than the standard one: its text
	// is text.<language>.fountain, as Accolade writes it, and text.fountain
	// the same as standard Fountain (fountain_dialect.go). A bundle's name
	// never says it.
	Language string `json:"language,omitempty"`
	// FountainHash is the hash of the text.fountain Accolade wrote beside
	// the language's text: another, and another app changed it.
	FountainHash string `json:"fountainHash,omitempty"`
	// NoFountainCopy: an .accolade keeps only the language's text, no
	// text.fountain (the script's choice).
	NoFountainCopy bool `json:"noFountainCopy,omitempty"`
	// Format is an .accolade's script format, when not the screenplay:
	// the script's own (MainWindow.scriptFormatKey), its file carried in
	// assets/lexington/formats/ if it is one of one's own
	// (lexington_carried.go).
	Format string `json:"format,omitempty"`
}

func (a accoladeInfo) empty() bool {
	return a.DocumentType == "" && a.Source == "" && a.Drafts == "" && !a.HighlandWarned &&
		a.Language == "" && a.FountainHash == "" && !a.NoFountainCopy && a.Format == ""
}

// textBundle is a bundle as read (Lexington's: where it is, its form and
// layout, its files and info.json) and what Accolade reads in it.
type textBundle struct {
	*lb.Bundle
	textName string // the text file, in the bundle ("text.fountain")
	// noText: there is no textName in the bundle, only the language's
	// text (an .accolade without its Fountain copy)
	noText bool
	// lang is the screenplay's language, "" for the standard one: Accolade's key's,
	// else (a bundle from before) as the bundle's name says
	lang string
	// native is the language's text's entry (text.eo.fountain), "" none,
	// and nativeText what it holds
	native     string
	nativeText []byte
	natural    bool // Markdown, as the text file's name and "type" say
	ours       accoladeInfo
	source     string   // the source's entry, "" for none
	srcText    []byte   // and what it holds
	extras     []string // Accolade's extras in it (extras.go), sorted
	// carried is the Lexington configuration an .accolade brought
	// (lexington_carried.go), nil for none; carriedNames its entries
	carried      *carriedConf
	carriedNames []string
}

// lex is the configuration the bundle's script reads: the machine's, with
// what it brought over it.
func (b *textBundle) lex() lexConf { return layered(b.carried) }

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

// bundleFile is a file Accolade writes into a bundle (name relative to
// the bundle; data nil and a name ending in '/' for a folder).
type bundleFile struct {
	name string
	data []byte
}

func lbFiles(files []bundleFile) []lb.File {
	out := make([]lb.File, len(files))
	for i, f := range files {
		out[i] = lb.File{Name: f.name, Data: f.data}
	}
	return out
}

// bundleOf is the bundle a file is the text of (X.textbundle/text.md:
// X.textbundle; X.accolade/text.fountain: the folder X.accolade), as a
// file picked in a folder bundle opens the bundle.
func bundleOf(p string) (string, bool) {
	dir := filepath.Dir(p)
	if !lb.Is(dir) || !lb.IsTextName(filepath.Base(p)) {
		return "", false
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return "", false
	}
	return dir, true
}

// nativeTextName is the entry of a screenplay's text in language lang.
func nativeTextName(lang string) string { return "text." + lang + ".fountain" }

// nativeLanguage is the language a text entry is in (text.eo.fountain:
// "eo"), "" for none.
func nativeLanguage(name string) string { return machineLex().nativeLanguage(name) }

func (c lexConf) nativeLanguage(name string) string {
	mid, ok := strings.CutPrefix(strings.ToLower(name), "text.")
	if !ok {
		return ""
	}
	if l, ok := strings.CutSuffix(mid, ".fountain"); ok && c.dialect(l) {
		return l
	}
	return ""
}

// fountainHash is the hash Accolade's key keeps of the text.fountain it
// wrote.
func fountainHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:12])
}

// pickText is the bundle's text among the files at its root, as
// Lexington picks it (text.fountain, text.md…), a language's text
// (text.eo.fountain) never: that is no other app's.
func pickText(names []string, conf lexConf) string {
	return lb.PickText(names, func(n string) bool { return conf.nativeLanguage(n) != "" })
}

// readBundle reads a bundle, a zip or a folder: what it holds besides the
// text, and the text.
func readBundle(p string) (*textBundle, []byte, error) {
	b, err := lb.Read(p)
	if err != nil {
		return nil, nil, err
	}
	return fillBundle(b)
}

// parseZipBundle reads a zipped bundle from its bytes.
func parseZipBundle(p string, data []byte) (*textBundle, []byte, error) {
	b, err := lb.ReadZip(p, data)
	if err != nil {
		return nil, nil, err
	}
	return fillBundle(b)
}

// fillBundle reads the bundle's text, info.json and source, its extras
// and what it carries.
func fillBundle(lbb *lb.Bundle) (*textBundle, []byte, error) {
	b := &textBundle{Bundle: lbb, extras: extraNames(lbb.Files)}
	p := lbb.Path
	for _, n := range lbb.Files {
		if isCarriedEntry(n) {
			b.carriedNames = append(b.carriedNames, n)
		}
	}
	if carriesConf(p) {
		b.carried = readCarried(b.carriedNames, b.ReadFile)
	}
	conf := b.lex() // what it brought, before its language is known
	root := lbb.Root
	b.textName = pickText(root, conf)
	var natives []string
	for _, n := range root {
		if conf.nativeLanguage(n) != "" {
			natives = append(natives, n)
		}
	}
	if b.textName == "" && len(natives) == 0 {
		return nil, nil, fmt.Errorf("%s holds no text file (text.fountain, text.md…)", filepath.Base(p))
	}
	var text []byte
	var err error
	if b.textName == "" { // only the language's text
		b.textName, b.noText = "text.fountain", true
	} else if text, err = b.ReadFile(b.textName); err != nil {
		return nil, nil, err
	}
	if raw, ok := lb.InfoValue(b.Info, accoladeInfoKey); ok {
		_ = json.Unmarshal(raw, &b.ours)
	}
	b.natural = lb.TextMarkdown(b.textName, b.Info, p)
	// the language: Accolade's key's, else the name's (a bundle from
	// before the key had it); its text, else the one there is
	if !b.markdown() {
		b.lang = b.ours.Language
		if !conf.dialect(b.lang) {
			b.lang = languageOf(p)
		}
		pick := ""
		for _, n := range natives {
			if strings.EqualFold(n, nativeTextName(b.lang)) {
				pick = n
			}
		}
		if pick == "" && len(natives) > 0 && (b.lang == "" || b.noText) {
			slices.Sort(natives)
			pick = natives[0]
		}
		if pick != "" {
			if b.nativeText, err = b.ReadFile(pick); err != nil {
				return nil, nil, err
			}
			b.native, b.lang = pick, conf.nativeLanguage(pick)
		}
	}
	// the source: as Accolade's key names it, else one where Accolade or
	// a person would put it
	for _, n := range []string{b.ours.Source, bundleSourceName, "source.md", "source.markdown"} {
		if n == "" || !fs.ValidPath(n) || strings.EqualFold(n, b.textName) {
			continue
		}
		if data, err := b.ReadFile(n); err == nil {
			b.source, b.srcText = n, data
			break
		}
	}
	return b, text, nil
}

// screenplayBundle tells whether a bundle's name says it holds a
// screenplay: Highland's .highland and Accolade's own .accolade (the same
// zipped TextBundle, with Accolade's drafts and production files added:
// extras.go).
func screenplayBundle(name string) bool {
	t, _ := lb.TypeOf(name)
	return t.Screenplay
}

// newBundleInfo is info.json for a bundle Accolade makes.
func newBundleInfo(markdown bool) []byte { return lb.NewInfo(markdown, accoladeInfoKey) }

// bundleSave is what saveBundleText writes.
type bundleSave struct {
	text     []byte
	markdown bool // the document's type
	// extra are Accolade's files carried in (a plain script's drafts,
	// saved as a bundle), added to or replacing the bundle's
	extra []bundleFile
	// source is a plain script's source carried in (assets/source.md)
	source []byte
	// highlandWarned marks the Highland warning given (flatten.go)
	highlandWarned bool
	// native is a screenplay's text in its language, lang (text is then
	// the same as standard Fountain), nil for the standard language; noCopy leaves
	// text.fountain out (an .accolade's choice)
	native []byte
	lang   string
	noCopy bool
	// conf is the script's configuration (MainWindow.lex; unset: the
	// machine's); an .accolade carries carry (lexington_carried.go) and
	// names format (the screenplay's: "")
	conf   lexConf
	carry  *carriedConf
	format string
	// form is the form asked for (Save As's Accolade folder; notBundle:
	// the bundle's own for its file, else as the name has it)
	form bundleKind
}

// lex is the configuration the script is saved with.
func (s bundleSave) lex() lexConf {
	if s.conf.Languages == nil {
		return machineLex()
	}
	return s.conf
}

// carried are the configuration's files a save to dst writes, and the
// old ones gone (all of them for another kind of bundle); ours names the
// format.
func (s bundleSave) carried(ours *accoladeInfo, dst string, old []string) (files []bundleFile, gone []string) {
	ours.Format = ""
	if !carriesConf(dst) || s.markdown {
		return nil, old
	}
	if s.format != rules.ScreenplayPreset {
		ours.Format = s.format
	}
	files = s.carry.files()
	for _, n := range old {
		if !slices.ContainsFunc(files, func(f bundleFile) bool { return f.name == n }) {
			gone = append(gone, n)
		}
	}
	return files, gone
}

// language is what Accolade's key says of the language, and the text
// entries written and gone.
func (s bundleSave) language(ours *accoladeInfo, textName, oldNative string, oldNoText bool) (files []bundleFile, gone []string) {
	ours.Language, ours.FountainHash, ours.NoFountainCopy = "", "", false
	native := ""
	if s.native != nil && s.lex().dialect(s.lang) {
		native = nativeTextName(s.lang)
		ours.Language = s.lang
		files = append(files, bundleFile{native, s.native})
	}
	if oldNative != "" && oldNative != native {
		gone = append(gone, oldNative)
	}
	switch {
	case native == "" || !s.noCopy:
		files = append(files, bundleFile{textName, s.text})
		if native != "" {
			ours.FountainHash = fountainHash(s.text)
		}
	case !oldNoText:
		gone = append(gone, textName)
		ours.NoFountainCopy = true
	default:
		ours.NoFountainCopy = true
	}
	return files, gone
}

// saveBundleText writes the bundle (b; nil for a new one) to dst with s's
// text as its text; the document's type is kept in Accolade's key if the
// bundle would say otherwise. The bundle as written is returned.
func saveBundleText(b *textBundle, dst string, s bundleSave) (*textBundle, error) {
	if b == nil {
		textName := "text.fountain"
		if s.markdown {
			textName = "text.md"
		}
		info := newBundleInfo(s.markdown)
		ours := accoladeInfo{Version: 1, HighlandWarned: s.highlandWarned, Drafts: draftsIndexIn(s.extra)}
		if s.source != nil {
			ours.Source = bundleSourceName
		}
		texts, _ := s.language(&ours, textName, "", true)
		carried, _ := s.carried(&ours, dst, nil)
		if !ours.empty() {
			val, _ := json.Marshal(ours)
			info, _ = lb.SetInfoValue(info, accoladeInfoKey, val)
		}
		files := append(append([]bundleFile{{"info.json", info}}, texts...), bundleFile{"assets/", nil})
		files = append(append(files, carried...), s.extra...)
		if s.source != nil {
			files = append(files, bundleFile{bundleSourceName, s.source})
		}
		return writeBundleAs(nil, dst, s.form, files, nil)
	}
	if sameFile(dst, b.Path) {
		b = b.fresh()
	}
	ours := b.ours
	ours.DocumentType = ""
	if s.markdown != b.natural {
		ours.DocumentType = map[bool]string{true: "markdown", false: "screenplay"}[s.markdown]
	}
	ours.HighlandWarned = ours.HighlandWarned || s.highlandWarned
	ours.Drafts = cmp.Or(draftsIndexIn(s.extra), ours.Drafts)
	files, gone := s.language(&ours, b.textName, b.native, b.noText)
	carried, old := s.carried(&ours, dst, b.carriedNames)
	files, gone = append(append(files, carried...), s.extra...), append(gone, old...)
	if info, changed := b.infoWith(ours); changed {
		files = append(files, bundleFile{b.infoName(), info})
	}
	return writeBundleAs(b, dst, s.form, files, gone)
}

// saveBundleFiles writes files into the bundle, in place, the text as it
// is in the bundle (Save Draft); highlandWarned as bundleSave's.
func saveBundleFiles(b *textBundle, files []bundleFile, highlandWarned bool) (*textBundle, error) {
	return saveBundleFilesWithout(b, files, nil, highlandWarned)
}

// saveBundleFilesWithout is saveBundleFiles with the files gone (names in
// the bundle, as files name them) taken out.
func saveBundleFilesWithout(b *textBundle, files []bundleFile, gone []string, highlandWarned bool) (*textBundle, error) {
	b = b.fresh()
	ours := b.ours
	ours.Drafts = cmp.Or(draftsIndexIn(files), ours.Drafts)
	ours.HighlandWarned = ours.HighlandWarned || highlandWarned
	if info, changed := b.infoWith(ours); changed {
		files = append(slices.Clone(files), bundleFile{b.infoName(), info})
	}
	return writeBundleWithout(b, b.Path, files, gone)
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
	return writeBundle(b, b.Path, files)
}

// fresh is the bundle as it is on disk now, if it is still the same
// bundle (its form, layout and text file): what another app changed in
// it since is kept. Else (gone, emptied by a save dialog, made over) as
// read.
func (b *textBundle) fresh() *textBundle {
	if !b.Changed() { // unchanged since read or written here: as it is
		return b
	}
	nb, _, err := readBundle(b.Path)
	if err != nil || nb.Zipped != b.Zipped || nb.Prefix != b.Prefix || nb.textName != b.textName {
		return b
	}
	return nb
}

// infoName is info.json's name as the bundle has it.
func (b *textBundle) infoName() string { return cmp.Or(b.InfoName, "info.json") }

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
	_, has := lb.InfoValue(b.Info, accoladeInfoKey)
	if has == !ours.empty() && (!has || ours == norm(b.ours)) {
		return b.Info, false // nothing new to say
	}
	var val []byte
	if !ours.empty() {
		val, _ = json.Marshal(ours)
	}
	info := b.Info
	if info == nil { // a bundle without one: TextBundle's own
		info = newBundleInfo(b.natural)
	}
	info, ok := lb.SetInfoValue(info, accoladeInfoKey, val)
	if !ok || slices.Equal(info, b.Info) {
		return b.Info, false
	}
	return info, true
}

// writeBundle writes a bundle to dst: b's entries (none for a new
// bundle) copied through, files replacing theirs or added after them.
func writeBundle(b *textBundle, dst string, files []bundleFile) (*textBundle, error) {
	return writeBundleWithout(b, dst, files, nil)
}

// writeBundleWithout is writeBundle with the entries gone (names in the
// bundle) left out.
func writeBundleWithout(b *textBundle, dst string, files []bundleFile, gone []string) (*textBundle, error) {
	return writeBundleAs(b, dst, notBundle, files, gone)
}

// writeBundleAs is writeBundleWithout in a form asked for (notBundle: the
// bundle's own for its file, else what is at dst or its name's), the
// layout as Lexington's bundle.Write keeps it.
func writeBundleAs(b *textBundle, dst string, form bundleKind, files []bundleFile, gone []string) (*textBundle, error) {
	var base *lb.Bundle
	if b != nil {
		base = b.Bundle
	}
	nb, err := lb.Write(base, dst, form.form(), lbFiles(files), gone)
	if err != nil {
		return nil, err
	}
	tb, _, err := fillBundle(nb)
	return tb, err
}
