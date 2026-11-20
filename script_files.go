package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// The script's files: the types it can be saved as, its name's parts, and
// every file that belongs to it with where each goes when the script is
// renamed or saved as another type. The rename editor (rename.go) and Save
// As (saveas.go) show the same plan (planRows); relocate.go carries it out.

// fileType is a kind of file a script can be kept in.
type fileType struct {
	id, label, ext string
	kind           bundleKind
	desktopOnly    bool
}

var (
	typeFountain   = fileType{id: "fountain", label: "Fountain", ext: ".fountain"}
	typeMarkdown   = fileType{id: "markdown", label: "Markdown", ext: ".md"}
	typeAccolade   = fileType{id: "accolade", label: "Accolade", ext: ".accolade", kind: bundleZip}
	typeHighland   = fileType{id: "highland", label: "Highland", ext: ".highland", kind: bundleZip}
	typeTextPack   = fileType{id: "textpack", label: "TextPack", ext: ".textpack", kind: bundleZip}
	typeTextBundle = fileType{id: "textbundle", label: "TextBundle", ext: ".textbundle", kind: bundleFolder, desktopOnly: true}
)

// asItIs is the type of a plain file named otherwise (.txt, .spmd,
// .markdown): its extension kept.
func asItIs(ext string) fileType {
	return fileType{id: "asis", label: "As it is", ext: ext}
}

// knownTypes are the types by extension.
var knownTypes = []fileType{typeFountain, typeMarkdown, typeAccolade, typeHighland, typeTextPack, typeTextBundle}

// plainExts are the plain extensions a script or document can have.
var plainExts = []string{".fountain", ".spmd", ".txt", ".md", ".markdown"}

// typeOfName is the type a file's name says; a plain one with another
// extension is "as it is", no extension Fountain (or Markdown).
func typeOfName(name string, markdown bool) fileType {
	ext := strings.ToLower(filepath.Ext(strings.TrimRight(name, `/\`)))
	for _, t := range knownTypes {
		if t.ext == ext {
			return t
		}
	}
	if slices.Contains(plainExts, ext) {
		return asItIs(filepath.Ext(name))
	}
	if markdown {
		return typeMarkdown
	}
	return typeFountain
}

// hasTypeExt tells whether a typed name ends in an extension a script's
// file can have: then it is taken as it is.
func hasTypeExt(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext != "" && (slices.ContainsFunc(knownTypes, func(t fileType) bool { return t.ext == ext }) || slices.Contains(plainExts, ext))
}

// scriptTypes are the types offered, in order: the document's own plain
// type first, the bundles, the other kind's plain type last (it switches
// the document); a current name with another plain extension adds "as it
// is" first. The browser has no folders.
func scriptTypes(markdown, browser bool, current string) []fileType {
	own, other := typeFountain, typeMarkdown
	if markdown {
		own, other = typeMarkdown, typeFountain
	}
	var ts []fileType
	if t := typeOfName(current, markdown); t.id == "asis" {
		ts = append(ts, t)
	}
	ts = append(ts, own, typeAccolade, typeHighland, typeTextPack)
	if !browser {
		ts = append(ts, typeTextBundle)
	}
	return append(ts, other)
}

// keeps is the one line saying what a type keeps.
func (t fileType) keeps(markdown, browser bool) string {
	doc := "the script"
	if markdown {
		doc = "the document"
	}
	extras := "drafts and production files kept by Accolade, or in a Name.production folder beside it"
	if browser {
		extras = "drafts and production files kept by Accolade"
	}
	switch t.id {
	case "fountain":
		if markdown {
			return "Makes the document a screenplay, as plain Fountain."
		}
		return "Plain Fountain, for any Fountain app: the script with Accolade's blocks inside; " + extras + "."
	case "markdown":
		if !markdown {
			return "Makes the script a Markdown document, as plain Markdown."
		}
		return "Plain Markdown, for any editor: the document; " + extras + "."
	case "accolade":
		return "Accolade's own: everything in one file — " + doc + ", its source, drafts and production files."
	case "highland":
		return "For Highland 2: one file Highland opens. Highland may drop Accolade's drafts and source when it saves."
	case "textpack":
		return "A zipped TextBundle, for other TextBundle apps: " + doc + ", with Accolade's files under assets/."
	case "textbundle":
		return "A TextBundle folder (desktop only): the same as a TextPack, unzipped, for version control."
	case "asis":
		return "Plain text under its own name (" + t.ext + "), as it is now."
	}
	return ""
}

// recommendedType is the type a script is best kept in: Markdown for a
// document, .accolade once a screenplay has production data (saveAsName's
// rule), else Fountain.
func recommendedType(markdown, production bool) fileType {
	switch {
	case markdown:
		return typeMarkdown
	case production:
		return typeAccolade
	}
	return typeFountain
}

// nameParts are a script's file name's parts: "film.eo.fountain" is the
// base "film", the language "eo" and the extension ".fountain".
type nameParts struct {
	base, lang, ext string
}

// splitScriptName splits a name into its parts; a language is only a
// script language's code, an extension any.
func splitScriptName(name string) nameParts {
	ext := filepath.Ext(name)
	rest := strings.TrimSuffix(name, ext)
	if ext != "" && !hasTypeExt(name) { // "Mr. Smith": no extension
		ext, rest = "", name
	}
	p := nameParts{base: rest, ext: ext}
	if l := filepath.Ext(rest); len(l) > 1 && sceneConf[strings.ToLower(l[1:])] != nil {
		p.lang = strings.ToLower(l[1:])
		p.base = strings.TrimSuffix(rest, l)
	}
	return p
}

// String is the name the parts make.
func (p nameParts) String() string {
	s := p.base
	if p.lang != "" {
		s += "." + p.lang
	}
	return s + p.ext
}

// composedName is the name the rename editor makes: a typed name with an
// extension as it is (for those who know), else the base with the
// language and type picked.
func composedName(typed, lang string, t fileType) string {
	typed = strings.TrimSpace(typed)
	if hasTypeExt(typed) {
		return typed
	}
	return nameParts{base: typed, lang: lang, ext: t.ext}.String()
}

// relocation is a rename or a Save As: the script from its file (""
// unsaved) to another, and the choices on the way.
type relocation struct {
	rename   bool   // the old file goes (a rename), or stays (Save As)
	from, to string // the script's file now and after
	// how a bundle's drafts and production files leave it for a plain file
	how flattenChoice
	// keepBesideName: the source beside the script (story.md) keeps its
	// name, instead of following the script's (film.md)
	keepBesideName bool
}

// kinds are what the script is now and will be: a plain .highland stays
// plain when renamed to another .highland.
func (w *MainWindow) kinds(r relocation) (from, to bundleKind) {
	if r.from != "" && w.bundle != nil {
		from = bundleKindOf(r.from)
	}
	to = bundleKindOf(r.to)
	if r.rename && from == notBundle && w.isPlainBundle(r.from) && strings.EqualFold(filepath.Ext(r.to), ".highland") {
		to = notBundle
	}
	return from, to
}

// sameKind tells whether the script stays the kind of file it is (a
// rename then only renames): plain, a zip, a folder.
func sameKind(from, to bundleKind) bool { return from == to }

// besideTarget is where the source beside the script goes for the
// script's new name (story.md beside film.fountain: film.md), "" if it
// keeps its name or one is there already.
func (w *MainWindow) besideTarget(r relocation, ref sourceRef) string {
	if ref.Origin != sourceBeside || r.keepBesideName || r.to == "" {
		return ""
	}
	t := strings.TrimSuffix(r.to, filepath.Ext(r.to)) + filepath.Ext(ref.Path)
	if absPath(t) == absPath(ref.Path) {
		return ""
	}
	if _, err := os.Stat(t); err == nil {
		return ""
	}
	return t
}

// sourceOutBeside is where a bundle's source goes leaving it for a plain
// file with its drafts in the folder beside: Name.md beside it, if free;
// "" for a kept copy.
func sourceOutBeside(r relocation) string {
	if r.how != flattenFolder || inBrowser {
		return ""
	}
	t := strings.TrimSuffix(r.to, filepath.Ext(r.to)) + ".md"
	if _, err := os.Stat(t); err == nil || absPath(t) == absPath(r.to) {
		return ""
	}
	return t
}

// planRow is one file of the script in the plan: what it is, where it is
// now and where it will be. fixed: its name is the format's, not the
// writer's; warn: it is left out.
type planRow struct {
	what, now, after string
	fixed, warn      bool
}

// extrasGroups are the extras by what they are: the drafts together, the
// production files each.
func extrasGroups(names []string) (drafts int, files []string) {
	seen := map[string]bool{}
	for _, n := range names {
		if rest, ok := strings.CutPrefix(n, draftsDir); ok {
			if dir, _, sub := strings.Cut(rest, "/"); sub && !seen[dir] { // one folder a draft
				seen[dir] = true
				drafts++
			}
			continue
		}
		files = append(files, strings.TrimPrefix(n, productionDir))
	}
	return drafts, files
}

// otherFiles are the bundle's files that are neither its text, info.json,
// its source nor Accolade's extras: another app's, copied through.
func (b *textBundle) otherFiles() []string {
	if b == nil {
		return nil
	}
	es, _ := b.entries()
	var out []string
	for _, e := range es {
		rel := e.name
		if b.zipped {
			var ok bool
			if rel, ok = strings.CutPrefix(e.name, b.prefix); !ok {
				continue
			}
		}
		if rel == "" || strings.HasSuffix(rel, "/") || junkEntry(e.name) || rel == b.textName ||
			strings.EqualFold(rel, "info.json") || rel == b.source || isExtra(rel) {
			continue
		}
		out = append(out, rel)
	}
	return out
}

// planRows are the script's files and where each goes in r.
func (w *MainWindow) planRows(r relocation) []planRow {
	fromKind, toKind := w.kinds(r)
	old := "not saved yet"
	if r.from != "" {
		old = filepath.Base(r.from)
	}
	newName := filepath.Base(r.to)
	what := "Script"
	if w.markdown || isMarkdownFile(r.to) {
		what = "Document"
	}
	verb := ""
	if !r.rename && r.from != "" {
		verb = "a copy: "
	}
	rows := []planRow{{what: what, now: old, after: newName}}
	if r.rename && r.from != "" && exists(r.to) && absPath(r.from) != absPath(r.to) && !sameFile(r.from, r.to) {
		rows[0].after, rows[0].warn = newName+": a file by that name is there already", true
	}
	inside := func(name string) string { return "inside " + name }
	beside := func(name string) string { return name + " beside it" }

	// the text inside a bundle: a name the format fixes
	if toKind != notBundle {
		text := "text.fountain"
		if w.markdown {
			text = "text.md"
		}
		if fromKind != notBundle && w.bundle != nil {
			text = w.bundle.textName
		}
		now := "the file itself"
		if fromKind != notBundle && w.bundle != nil {
			now = inside(old)
		}
		rows = append(rows, planRow{what: text + ", info.json", now: now, after: inside(newName), fixed: true})
	}

	// the source
	if !w.markdown {
		ref := w.sourceRef()
		switch ref.Origin {
		case sourceInside:
			rows = append(rows, planRow{what: "Source", now: "in the script (/* Source */)", after: "in the script"})
		case sourceBundle:
			row := planRow{what: "Source", now: path.Base(ref.Entry) + " " + inside(old)}
			switch {
			case toKind != notBundle:
				row.what, row.after, row.fixed = ref.Entry, inside(newName), true
			case sourceOutBeside(r) != "":
				row.after = beside(filepath.Base(sourceOutBeside(r)))
			default:
				row.after = "kept by Accolade"
			}
			rows = append(rows, row)
		case sourceKept:
			carried := carries(fromKind, toKind, ref)
			row := planRow{what: "Source", now: filepath.Base(ref.Path) + ", kept by Accolade"}
			if !strings.HasPrefix(ref.Path, keptSourcesDir()+string(filepath.Separator)) {
				row.now = filepath.Base(ref.Path) + ", chosen"
			}
			switch {
			case carried:
				row.what, row.after, row.fixed = bundleSourceName, inside(newName), true
			case r.rename:
				row.after = "kept by Accolade, as now"
			default:
				row.after = verb + "kept by Accolade"
			}
			rows = append(rows, row)
		case sourceBeside:
			row := planRow{what: "Source", now: beside(filepath.Base(ref.Path))}
			t := w.besideTarget(r, ref)
			theirs := strings.TrimSuffix(r.to, filepath.Ext(r.to)) + filepath.Ext(ref.Path)
			switch {
			case carries(fromKind, toKind, ref):
				row.what, row.after, row.fixed = bundleSourceName, inside(newName), true
			case t != "":
				row.after = verb + beside(filepath.Base(t))
			case r.keepBesideName || absPath(theirs) == absPath(ref.Path):
				row.after = beside(filepath.Base(ref.Path)) + ", as now"
			default:
				row.after = filepath.Base(theirs) + " is there already: that one is used"
				row.warn = true
			}
			rows = append(rows, row)
		}
	}

	// drafts and production files
	home := w.extrasHome()
	names := home.names()
	if len(names) > 0 {
		drafts, files := extrasGroups(names)
		type group struct{ what, dir string }
		var groups []group
		if drafts > 0 {
			label := fmt.Sprintf("%d drafts", drafts)
			if drafts == 1 {
				label = "1 draft"
			}
			groups = append(groups, group{label, draftsDir})
		}
		for _, f := range files {
			groups = append(groups, group{f, productionDir})
		}
		for _, g := range groups {
			row := planRow{what: g.what, fixed: true}
			switch home.kind {
			case extrasBundle:
				row.now = g.dir + " " + inside(old)
			case extrasKept:
				row.now = "kept by Accolade"
			case extrasBeside:
				row.now = filepath.Base(string(home.dir)) + "/ beside it"
			}
			switch {
			case toKind != notBundle:
				row.after = g.dir + " " + inside(newName)
			case fromKind != notBundle:
				switch r.how {
				case flattenFolder:
					row.after = filepath.Base(productionFolderOf(r.to)) + "/ beside it"
				case flattenDrop:
					row.after, row.warn = "left out ("+old+" keeps them)", true
				default:
					row.after = "kept by Accolade"
				}
			case home.kind == extrasKept:
				row.after = map[bool]string{true: "kept by Accolade, as now", false: verb + "kept by Accolade"}[r.rename]
			case home.kind == extrasBeside:
				row.after = verb + filepath.Base(productionFolderOf(r.to)) + "/ beside it"
			}
			rows = append(rows, row)
		}
	}

	// another app's files in the bundle
	if fromKind != notBundle && w.bundle != nil {
		if others := w.bundle.otherFiles(); len(others) > 0 {
			row := planRow{what: fmt.Sprintf("%d other files (%s)", len(others), strings.Join(firstN(others, 3), ", ")), now: inside(old)}
			if len(others) == 1 {
				row.what = "1 other file (" + others[0] + ")"
			}
			switch {
			case toKind != notBundle:
				row.after, row.fixed = inside(newName), true
			case r.rename:
				row.after, row.warn = "left out: a plain file cannot hold them", true
			default:
				row.after, row.warn = "left out ("+old+" keeps them)", true
			}
			rows = append(rows, row)
		}
	}
	return rows
}

// firstN are the first n of ss, "…" after them if there are more.
func firstN(ss []string, n int) []string {
	if len(ss) <= n {
		return ss
	}
	return append(slices.Clone(ss[:n]), "…")
}
