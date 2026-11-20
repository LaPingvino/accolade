package main

import (
	"archive/zip"
	"bytes"
	"errors"
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

// Accolade's extra files: the drafts (drafts.go) and, from production's
// later steps, the production files (docs/PRODUCTION.md §4). They are
// named as in a bundle (assets/drafts/01-first/text.fountain) and kept
// in one of three homes, the bundle's in the bundle and a plain script's
// in a folder laid out as the bundle would be:
//
//  1. In the bundle (.accolade, .highland, .textpack, .textbundle), under
//     assets/: they travel with the file, copied through on every save.
//  2. For a plain script (.fountain), a copy Accolade keeps, in its data
//     folder (production/ beside sources/; the browser's kept settings
//     folder), remembered per script in settings.json
//     ("production-files"), following a rename or Save As, as the
//     source's copy does (source_file.go).
//  3. Or, on choice when a bundle is saved as a plain file (desktop
//     only), Name.production/ beside Name.fountain: the bundle's assets
//     unzipped, to take along or put in version control.
//
// A plain script looks for its extras in that order: the kept copy
// remembered for it, else the folder beside it. Saving a plain script as
// a bundle carries its extras in; saving a bundle as a plain file asks
// where they go (flatten.go).

const (
	draftsDir     = "assets/drafts/"
	productionDir = "assets/production/"
)

// isExtra tells whether a bundle's file is one of Accolade's extras.
func isExtra(rel string) bool {
	return (strings.HasPrefix(rel, draftsDir) || strings.HasPrefix(rel, productionDir)) && !strings.HasSuffix(rel, "/")
}

// extrasReader reads an extra file by its name in the bundle.
type extrasReader interface {
	readExtra(rel string) ([]byte, error)
}

// readExtra reads one of the bundle's files, by its name in the bundle.
func (b *textBundle) readExtra(rel string) ([]byte, error) {
	if b == nil || !fs.ValidPath(rel) {
		return nil, fs.ErrNotExist
	}
	if !b.zipped {
		return os.ReadFile(filepath.Join(b.path, filepath.FromSlash(rel)))
	}
	zr, err := zip.NewReader(bytes.NewReader(b.raw), int64(len(b.raw)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if f.Name == b.prefix+rel {
			r, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer r.Close()
			return io.ReadAll(r)
		}
	}
	return nil, fs.ErrNotExist
}

// extrasFolder is a folder holding extras as a bundle holds them (a kept
// copy, or Name.production/).
type extrasFolder string

func (d extrasFolder) readExtra(rel string) ([]byte, error) {
	if d == "" || !fs.ValidPath(rel) {
		return nil, fs.ErrNotExist
	}
	return os.ReadFile(filepath.Join(string(d), filepath.FromSlash(rel)))
}

// names are the extras in the folder, sorted.
func (d extrasFolder) names() []string {
	if d == "" {
		return nil
	}
	return folderExtras(string(d))
}

// write puts files in the folder.
func (d extrasFolder) write(files []bundleFile) error {
	if err := os.MkdirAll(string(d), 0o755); err != nil {
		return err
	}
	for _, f := range files {
		if err := writeBundleFile(string(d), f); err != nil {
			return err
		}
	}
	return nil
}

// remove takes files out of the folder (a name ending in / is a folder,
// taken out if empty); one not there is no error.
func (d extrasFolder) remove(gone []string) error {
	for _, g := range gone {
		if d == "" || !fs.ValidPath(strings.TrimSuffix(g, "/")) {
			continue
		}
		err := os.Remove(filepath.Join(string(d), filepath.FromSlash(strings.TrimSuffix(g, "/"))))
		if err != nil && !errors.Is(err, fs.ErrNotExist) && !strings.HasSuffix(g, "/") {
			return err
		}
	}
	return nil
}

// folderExtras are the extras in a folder laid out as a bundle, sorted.
func folderExtras(dir string) []string {
	var names []string
	for _, sub := range []string{draftsDir, productionDir} {
		root := filepath.Join(dir, filepath.FromSlash(sub))
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || !d.Type().IsRegular() {
				return nil
			}
			rel, _ := filepath.Rel(dir, p)
			names = append(names, filepath.ToSlash(rel))
			return nil
		})
	}
	slices.Sort(names)
	return names
}

// extrasKind is where a script's extras are.
type extrasKind int

const (
	extrasNone   extrasKind = iota
	extrasBundle            // in the script's bundle
	extrasKept              // a copy Accolade keeps
	extrasBeside            // Name.production/ beside the script
)

// extrasHome is where a script's extras are kept.
type extrasHome struct {
	kind extrasKind
	b    *textBundle  // extrasBundle
	dir  extrasFolder // extrasKept, extrasBeside
}

func (h extrasHome) readExtra(rel string) ([]byte, error) {
	if h.kind == extrasBundle {
		return h.b.readExtra(rel)
	}
	return h.dir.readExtra(rel)
}

// names are the extras there, sorted.
func (h extrasHome) names() []string {
	if h.kind == extrasBundle {
		return h.b.extras
	}
	return h.dir.names()
}

// all are the extras there, read.
func (h extrasHome) all() ([]bundleFile, error) {
	var files []bundleFile
	for _, n := range h.names() {
		data, err := h.readExtra(n)
		if err != nil {
			return nil, err
		}
		files = append(files, bundleFile{n, data})
	}
	return files, nil
}

// describe is how a dialog names it.
func (h extrasHome) describe() string {
	switch h.kind {
	case extrasBundle:
		return "in " + filepath.Base(h.b.path)
	case extrasKept:
		return "kept by Accolade"
	case extrasBeside:
		return "in " + filepath.Base(string(h.dir)) + " beside the script"
	}
	return "none yet"
}

// keptExtrasDir is the folder of the extras Accolade keeps for plain
// scripts: beside the kept sources.
func keptExtrasDir() string {
	return filepath.Join(filepath.Dir(keptSourcesDir()), "production")
}

// newKeptExtras is a new folder for a script's kept extras, named after
// the script.
func newKeptExtras(script string) (extrasFolder, error) {
	name := strings.TrimSuffix(filepath.Base(script), filepath.Ext(script))
	if name == "" || name == "." {
		name = "untitled"
	}
	dir := filepath.Join(keptExtrasDir(), name+"-"+strconv.FormatInt(time.Now().UnixNano(), 36))
	return extrasFolder(dir), os.MkdirAll(dir, 0o755)
}

// productionFolderOf is the Name.production/ folder beside a script.
func productionFolderOf(script string) string {
	return strings.TrimSuffix(script, filepath.Ext(script)) + ".production"
}

// isKeptExtras tells whether dir is one Accolade keeps (not the user's).
func isKeptExtras(dir string) bool {
	return dir != "" && strings.HasPrefix(dir, keptExtrasDir()+string(filepath.Separator))
}

// chosenExtras is the kept extras folder remembered for a script.
func (w *MainWindow) chosenExtras(script string) string {
	if script == "" {
		return w.unsavedExtras
	}
	if w.settings == nil {
		return ""
	}
	script = absPath(script)
	for _, e := range w.settings.GetStringSlice("production-files") {
		if s, ok := strings.CutPrefix(e, script+"\t"); ok {
			return s
		}
	}
	return ""
}

// rememberExtras keeps the extras folder for a script ("" forgets it).
func (w *MainWindow) rememberExtras(script, dir string) {
	if script == "" {
		w.unsavedExtras = dir
		return
	}
	if w.settings == nil {
		return
	}
	script = absPath(script)
	var list []string
	if dir != "" {
		list = append(list, script+"\t"+absPath(dir))
	}
	old := w.settings.GetStringSlice("production-files")
	for _, e := range old {
		if !strings.HasPrefix(e, script+"\t") && len(list) < maxDocumentTypes {
			list = append(list, e)
		}
	}
	if len(list) == 0 && len(old) == 0 {
		return
	}
	w.settings.SetStringSlice("production-files", list)
	_ = w.settings.Save()
}

// extrasHome is where a script's extras are: the window's bundle's, else
// as a plain script finds them (the kept copy, else the folder beside).
func (w *MainWindow) extrasHome() extrasHome {
	if w.bundle != nil {
		return extrasHome{kind: extrasBundle, b: w.bundle}
	}
	return w.plainExtrasHome(w.currentFile)
}

// plainExtrasHome is where a plain script's extras are, if anywhere.
func (w *MainWindow) plainExtrasHome(script string) extrasHome {
	if d := w.chosenExtras(script); d != "" {
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			return extrasHome{kind: extrasKept, dir: extrasFolder(d)}
		}
	}
	if script != "" && !inBrowser {
		if d := productionFolderOf(script); d != script {
			if st, err := os.Stat(d); err == nil && st.IsDir() {
				return extrasHome{kind: extrasBeside, dir: extrasFolder(d)}
			}
		}
	}
	return extrasHome{}
}

// hasProductionData tells whether the script has what a plain file
// cannot hold: extras, or a production block with a lock (Save then
// offers .accolade). Asked when saving, never per key.
func (w *MainWindow) hasProductionData() bool {
	if len(w.extrasHome().names()) > 0 {
		return true
	}
	return !w.markdown && sceneLockOf(w.textEditor.Text()).locked
}

// forgetUnsavedExtras lets go of the extras of a script not saved yet, as
// another document takes the window's place.
func (w *MainWindow) forgetUnsavedExtras() {
	w.dropRenameUndo() // another document: the last rename is not taken back now
	w.unsavedRole = "" // and an unsaved script's role was its own (role.go)
	w.revisionGen++    // its issues too (revisions.go)
	if d := w.unsavedExtras; d != "" {
		w.unsavedExtras = ""
		if isKeptExtras(d) {
			os.RemoveAll(d)
		}
	}
}

// extrasAfterRename keeps a plain script's extras through a rename or
// Save As, as sourceAfterRename does its source: carried into a bundle
// (writeScript), the old kept copy then let go of if the old file is
// gone; from plain to plain the kept copy's memory moves, the folder
// beside is copied (Save As) or renamed (rename). A bundle saved as a
// plain file placed its extras itself (flatten.go).
func (w *MainWindow) extrasAfterRename(oldPath, newPath string) {
	if newPath == "" || (oldPath != "" && absPath(oldPath) == absPath(newPath)) {
		return
	}
	if oldPath != "" && bundleKindOf(oldPath) != notBundle && !w.isPlainBundle(oldPath) {
		return // from a bundle: placed as it was flattened
	}
	old := w.plainExtrasHome(oldPath)
	if old.kind == extrasNone {
		if oldPath == "" {
			w.unsavedExtras = ""
		}
		return
	}
	oldGone := w.oldGoes(oldPath)
	if bundleKindOf(newPath) != notBundle && !w.isPlainBundle(newPath) {
		// carried into the bundle: an unsaved script's copy let go of once
		// the bundle is written (bundle_queue.go)
		if old.kind == extrasKept && oldGone {
			w.rememberExtras(oldPath, "")
			dir := string(old.dir)
			w.afterBundleWrites(func() {
				if w.bundle != nil && sameFile(w.bundle.path, newPath) {
					w.discard(dir) // parked while a rename can be taken back
				}
			})
		}
		return
	}
	switch old.kind {
	case extrasKept:
		if !oldGone { // Save As: a copy of its own, the old file keeping its
			if files, err := old.all(); err == nil {
				if dir, err := newKeptExtras(newPath); err == nil {
					if dir.write(files) == nil {
						w.rememberExtras(newPath, string(dir))
						return
					}
					os.RemoveAll(string(dir))
				}
			}
		}
		w.rememberExtras(oldPath, "")
		w.rememberExtras(newPath, string(old.dir))
	case extrasBeside:
		to := productionFolderOf(newPath)
		if _, err := os.Stat(to); err == nil {
			return // one there already: the new name's own
		}
		if oldGone {
			if os.Rename(string(old.dir), to) == nil {
				return
			}
		}
		if files, err := old.all(); err == nil {
			_ = extrasFolder(to).write(files)
		}
	}
}

// extraNames are the bundle's extras, from the files at the given names
// (relative to the bundle).
func extraNames(rels []string) []string {
	var names []string
	for _, r := range rels {
		if isExtra(r) && path.Clean(r) == r {
			names = append(names, r)
		}
	}
	slices.Sort(names)
	return names
}
