package main

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2/dialog"
)

// Renaming a script, or saving it as another file, with every file that
// belongs to it (script_files.go's plan), as one action: all or nothing.
// What it does to the files is journaled — files made, files renamed,
// files it would delete parked under a hidden name beside them — so a
// failure on the way, or File > Undo Rename after, puts everything back:
// the parked files return, the made ones go, the settings that remember
// a script's source, drafts and type are as they were, and so is the
// window. Parked files are deleted once nothing can take the rename back
// (another rename, another document, the window closed).

// fileJournal is what a rename or Save As did to the files.
type fileJournal struct {
	renamed  [][2]string // from, to
	created  []string    // made; removed again
	parked   [][2]string // a file it replaced or let go of, and where it is parked
	settings map[string][]string
	state    windowState
	from, to string // the script's file before and after
	text     string // the script's text then
	wrote    bool   // the script's text was written to the new file
	rename   bool   // a rename: the old file goes
	carried  bool   // the source goes into the new bundle
	print    string // the extras and source as they were after it (fingerprint)
	done     bool   // committed: what it parked is the undo's
}

// journalKeys are the settings a rename changes.
var journalKeys = []string{"source-files", "production-files", "document-types", "recent-files", scriptRolesPref, looseReadingPref}

// windowState is the window's part in a rename.
type windowState struct {
	currentFile, suggestedName, plainBundle, language  string
	unsavedSource, unsavedExtras                       string
	bundle                                             *textBundle
	markdown, hasChanges, noFountainCopy, languageSaid bool
}

// beginJournal starts a journal of the window as it is.
func (w *MainWindow) beginJournal(from, to string) *fileJournal {
	j := &fileJournal{settings: map[string][]string{}, from: from, to: to, text: w.textEditor.Text()}
	if w.settings != nil {
		for _, k := range journalKeys {
			j.settings[k] = w.settings.GetStringSlice(k)
		}
	}
	j.state = windowState{
		currentFile: w.currentFile, suggestedName: w.suggestedName, plainBundle: w.plainBundle, language: w.language,
		unsavedSource: w.unsavedSource, unsavedExtras: w.unsavedExtras,
		bundle: w.bundle, markdown: w.markdown, hasChanges: w.hasChanges, noFountainCopy: w.noFountainCopy,
		languageSaid: w.languageSaid,
	}
	return j
}

// move renames from to to (copying across file systems), journaled.
func (j *fileJournal) move(from, to string) error {
	if err := movePath(from, to); err != nil {
		return err
	}
	j.renamed = append(j.renamed, [2]string{from, to})
	return nil
}

// copy copies a file or folder from to to, journaled.
func (j *fileJournal) copy(from, to string) error {
	if err := copyPath(from, to); err != nil {
		os.RemoveAll(to)
		return err
	}
	j.created = append(j.created, to)
	return nil
}

// park moves p out of the way under a hidden name beside it, journaled;
// nothing there is no error.
func (j *fileJournal) park(p string) error {
	if _, err := os.Lstat(p); err != nil {
		return nil
	}
	to := filepath.Join(filepath.Dir(p), "."+filepath.Base(p)+".accolade-undo-"+strconv.FormatInt(time.Now().UnixNano(), 36))
	if err := movePath(p, to); err != nil {
		return err
	}
	j.parked = append(j.parked, [2]string{p, to})
	return nil
}

// made notes a file that was made, if it is there and not one renamed
// there.
func (j *fileJournal) made(p string) {
	if p == "" || !exists(p) {
		return
	}
	for _, m := range j.renamed {
		if absPath(m[1]) == absPath(p) {
			return
		}
	}
	j.created = append(j.created, p)
}

// undoFiles puts the files back: the made ones removed, the renamed ones
// back, the parked ones returned (last first).
func (j *fileJournal) undoFiles() error {
	var errs []string
	for i := len(j.created) - 1; i >= 0; i-- {
		if err := os.RemoveAll(j.created[i]); err != nil {
			errs = append(errs, err.Error())
		}
	}
	for i := len(j.renamed) - 1; i >= 0; i-- {
		if err := movePath(j.renamed[i][1], j.renamed[i][0]); err != nil {
			errs = append(errs, err.Error())
		}
	}
	for i := len(j.parked) - 1; i >= 0; i-- {
		os.RemoveAll(j.parked[i][0]) // what took its place, if anything
		if err := movePath(j.parked[i][1], j.parked[i][0]); err != nil {
			errs = append(errs, err.Error())
		}
	}
	j.created, j.renamed, j.parked = nil, nil, nil
	if len(errs) > 0 {
		return fmt.Errorf("not everything could be put back: %s", strings.Join(errs, "; "))
	}
	return nil
}

// forget deletes what the journal parked: nothing takes it back now.
func (j *fileJournal) forget() {
	if j == nil {
		return
	}
	for _, p := range j.parked {
		os.RemoveAll(p[1])
	}
	j.parked = nil
}

// rollback puts the files, the settings and the window back as the
// journal found them.
func (w *MainWindow) rollback(j *fileJournal) error {
	err := j.undoFiles()
	if w.settings != nil {
		// only the entries of the script's two names: what other windows
		// remembered since stays
		mine := map[string]bool{}
		for _, p := range []string{j.from, j.to} {
			if p != "" {
				mine[absPath(p)] = true
			}
		}
		for _, k := range journalKeys {
			now := w.settings.GetStringSlice(k)
			var kept, back []string
			for _, e := range j.settings[k] {
				if mine[entryScript(e)] {
					back = append(back, e)
				}
			}
			for _, e := range now {
				if !mine[entryScript(e)] {
					kept = append(kept, e)
				}
			}
			if list := append(back, kept...); !slicesEqual(list, now) {
				w.settings.SetStringSlice(k, list)
			}
		}
		_ = w.settings.Save()
	}
	s := j.state
	w.currentFile, w.suggestedName, w.plainBundle = s.currentFile, s.suggestedName, s.plainBundle
	w.unsavedSource, w.unsavedExtras = s.unsavedSource, s.unsavedExtras
	b := s.bundle
	if b != nil { // as it is on disk again (a rename moved the one in hand)
		if nb, _, rerr := readBundle(s.currentFile); rerr == nil {
			b = nb
		} else {
			b.path = s.currentFile
		}
	}
	w.setBundle(b)
	if w.markdown != s.markdown {
		w.setMarkdown(s.markdown)
	}
	if w.language != s.language {
		w.setLanguage(s.language)
	}
	w.noFountainCopy, w.languageSaid = s.noFountainCopy, s.languageSaid
	w.hasChanges = s.hasChanges || w.textEditor.Text() != j.text
	w.updateTitle()
	w.ribbonRefresh()
	return err
}

// entryScript is the script a remembered entry is about: "script\tsource",
// "markdown:script", or the path itself (recent files).
func entryScript(e string) string {
	if s, _, ok := strings.Cut(e, "\t"); ok {
		return absPath(s)
	}
	for _, p := range []string{"markdown:", "screenplay:"} {
		if s, ok := strings.CutPrefix(e, p); ok {
			return absPath(s)
		}
	}
	return absPath(e)
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// discard lets go of a file a rename leaves behind: parked while a rename
// runs (so it can be taken back), else deleted.
func (w *MainWindow) discard(p string) {
	if j := w.relocating; j != nil {
		if err := j.park(p); err == nil {
			return
		}
	}
	os.RemoveAll(p)
}

// movePath renames from to to, copying and then deleting where a rename
// cannot (another file system).
func movePath(from, to string) error {
	if err := os.Rename(from, to); err == nil {
		return nil
	} else if _, serr := os.Lstat(from); serr != nil {
		return err
	}
	if err := copyPath(from, to); err != nil {
		os.RemoveAll(to)
		return err
	}
	return os.RemoveAll(from)
}

// copyPath copies a file, or a folder with what is in it.
func copyPath(from, to string) error {
	st, err := os.Stat(from)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return copyFile(from, to, st.Mode().Perm())
	}
	return filepath.WalkDir(from, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, p)
		dst := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		return copyFile(p, dst, 0o644)
	})
}

func copyFile(from, to string, perm fs.FileMode) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// relocate carries out r: the script and its files renamed, or saved as
// another file, once the writes under way are done. done (may be nil)
// gets the outcome once it is known: the bundle written, or nothing done.
func (w *MainWindow) relocate(r relocation, done func(error)) error {
	if done == nil {
		done = func(error) {}
	}
	if w.bundleBusy() { // a save under way finishes first
		gen, file := w.bundleGen, w.currentFile
		w.afterBundleWrites(func() {
			if w.bundleGen != gen || w.currentFile != file {
				done(fmt.Errorf("another document was opened meanwhile: nothing was renamed or saved"))
				return
			}
			if err := w.relocateNow(r, done); err != nil {
				done(err)
			}
		})
		return nil
	}
	return w.relocateNow(r, done)
}

// relocateNow is relocate with no bundle write pending. Its error is
// what failed at once (then nothing changed); a bundle written in the
// background that fails puts everything back and says so (done).
func (w *MainWindow) relocateNow(r relocation, done func(error)) error {
	oldName := cmp.Or(r.from, w.suggestedName, w.untitledName())
	if r.rename && r.how == flattenDrop { // the bundle goes: nothing would keep them
		r.how = flattenKept
	}
	lang, noCopy := w.langAfter(r)
	said := w.languageSaid || r.lang != ""
	if r.rename && r.from == "" { // not saved yet: only the name it will be saved by
		w.suggestedName = filepath.Base(r.to)
		w.setLanguage(lang)
		w.noFountainCopy, w.languageSaid = noCopy, said
		w.afterNewName(oldName, "", filepath.Base(r.to))
		done(nil)
		return nil
	}
	same := r.from != "" && (absPath(r.from) == absPath(r.to) || sameFile(r.from, r.to))
	if same && r.rename && filepath.Base(r.from) == filepath.Base(r.to) {
		if !w.reforms(r) && lang == w.language && noCopy == w.noFountainCopy && said == w.languageSaid {
			done(nil)
			return nil // nothing to do
		}
		// the same file, in another language: saved so (Edit > Script
		// Language… changes it back)
		w.setLanguage(lang)
		w.noFountainCopy, w.languageSaid = noCopy, said
		w.hasChanges = true
		if err := w.saveToFile(r.from); err != nil {
			return err
		}
		w.afterBundleWrites(func() { done(nil) })
		return nil
	}
	if _, err := os.Lstat(r.to); err == nil && r.rename && !same {
		return fmt.Errorf("%s already exists", filepath.Base(r.to))
	}
	fromKind, toKind := w.kinds(r)
	reform := w.reforms(r)
	ref := sourceRef{}
	if !w.markdown {
		ref = w.sourceRef()
	}
	home := w.extrasHome()
	j := w.beginJournal(r.from, r.to)
	j.rename = r.rename
	j.carried = carries(fromKind, toKind, ref)
	w.relocating = j
	// the language and the Fountain copy as chosen (rollback puts them back)
	w.setLanguage(lang)
	w.noFountainCopy, w.languageSaid = noCopy, said
	// what may be made for the new name, to know it was
	keptSource, keptExtras := w.chosenSource(r.to), w.chosenExtras(r.to)
	oldSource, oldExtras := w.chosenSource(r.from), w.chosenExtras(r.from) // moved, not made
	folder, besideMd := productionFolderOf(r.to), strings.TrimSuffix(r.to, filepath.Ext(r.to))+".md"
	folderThere, besideMdThere := exists(folder), exists(besideMd)
	// what was made for the new name: a kept copy, the folder or the
	// source beside it
	noteMade := func() {
		if k := w.chosenSource(r.to); k != keptSource && k != oldSource {
			if strings.HasPrefix(k, keptSourcesDir()+string(filepath.Separator)) {
				j.made(filepath.Dir(k))
			}
		}
		if k := w.chosenExtras(r.to); k != keptExtras && k != oldExtras && isKeptExtras(k) {
			j.made(k)
		}
		if !folderThere {
			j.made(folder)
		}
		if !besideMdThere {
			j.made(besideMd)
		}
	}
	fail := func(err error) error {
		noteMade()
		w.flattenTo = flattenKept
		w.carrySource, w.sourceBeside = nil, ""
		w.relocating = nil
		w.rollback(j)
		return err
	}

	// the source beside the script follows its name (story.md: film.md)
	if t := w.besideTarget(r, ref); t != "" && !j.carried {
		var err error
		if r.rename {
			err = j.move(ref.Path, t)
		} else {
			err = j.copy(ref.Path, t)
		}
		if err != nil {
			return fail(fmt.Errorf("could not take %s along: %v", filepath.Base(ref.Path), err))
		}
	}
	var after []func() // once the script is written: what the rename leaves behind
	failures := w.saveFailures
	if r.rename && sameKind(fromKind, toKind) && !reform {
		// the same kind of file: renamed, the folder beside it too
		wasPlain := w.isPlainBundle(r.from)
		if err := j.move(r.from, r.to); err != nil {
			return fail(fmt.Errorf("could not rename the file: %v", err))
		}
		if w.bundle != nil {
			w.bundle.path = r.to
		}
		if wasPlain {
			w.plainBundle = ""
			if strings.EqualFold(filepath.Ext(r.to), ".highland") {
				w.plainBundle = r.to
			}
		}
		if home.kind == extrasBeside {
			if to := productionFolderOf(r.to); !exists(to) {
				if err := j.move(string(home.dir), to); err != nil {
					return fail(fmt.Errorf("could not rename %s: %v", filepath.Base(string(home.dir)), err))
				}
			}
		}
	} else {
		// written anew: a bundle made, left or changed for another kind
		if !same {
			if err := j.park(r.to); err != nil { // replaced (Save As over a file): back if it fails
				return fail(err)
			}
		}
		w.flattenTo = r.how
		w.carrySource = nil
		if j.carried {
			md, err := w.sourceText(ref)
			if err != nil {
				return fail(fmt.Errorf("could not read the source to carry in: %v", err))
			}
			w.carrySource = []byte(md)
		}
		w.sourceBeside = ""
		if fromKind != notBundle && toKind == notBundle {
			w.sourceBeside = sourceOutBeside(r)
		}
		err := w.saveToFile(r.to)
		w.carrySource, w.sourceBeside = nil, ""
		if !same { // a bundle may be written yet (in the background): its name noted now
			j.created = append(j.created, r.to)
		}
		if err != nil {
			return fail(err)
		}
		j.wrote = true
		if j.carried {
			if r.rename && ref.Origin == sourceBeside {
				p := ref.Path
				after = append(after, func() { w.discard(p) })
			}
		}
		if r.rename {
			from := r.from
			after = append(after, func() { w.discard(from) })
			if toKind != notBundle && fromKind == notBundle && home.kind == extrasBeside {
				dir := string(home.dir)
				after = append(after, func() { w.discard(dir) })
			}
		}
	}
	// the window follows: its file, its source's and drafts' memories, its
	// type and language
	w.currentFile = r.to
	w.suggestedName = ""
	w.rememberRecent(r.to)
	w.afterNewName(oldName, r.from, r.to)
	noteMade()
	if j.carried {
		// carried in: the bundle has it now
		kept := ref.Path
		w.rememberSource(r.to, "")
		if r.rename && ref.Origin == sourceKept {
			w.rememberSource(r.from, "")
			after = append(after, func() {
				if strings.HasPrefix(kept, keptSourcesDir()+string(filepath.Separator)) {
					if dir := filepath.Dir(kept); dir != keptSourcesDir() {
						w.discard(dir)
					} else {
						w.discard(kept)
					}
				}
			})
		}
	}
	w.updateTitle()
	gen, file := w.bundleGen, w.currentFile
	w.afterBundleWrites(func() {
		moved := w.bundleGen != gen || w.currentFile != file // another document since
		if w.saveFailures != failures || !exists(r.to) {     // the bundle could not be written: all back
			w.relocating = nil
			var err error
			if moved { // the files only: the window is another's now
				err = j.undoFiles()
			} else {
				w.saveFailures = failures
				err = w.rollback(j)
			}
			done(cmp.Or(err, fmt.Errorf("%s could not be written: nothing was changed", filepath.Base(r.to))))
			return
		}
		for _, f := range after {
			f()
		}
		w.relocating = nil
		j.done = true
		w.dropRenameUndo() // only the last one, and not after a Save As
		if r.rename && !moved {
			if j.wrote {
				j.print = w.extrasPrint()
			}
			w.renameUndo = j
		} else {
			j.forget() // the file it replaced is gone for good
		}
		w.ribbonRefresh()
		done(nil)
	})
	return nil
}

// carries tells whether a script's source goes into the bundle it becomes:
// a plain script's kept copy or file beside (the bundle's own then).
func carries(fromKind, toKind bundleKind, ref sourceRef) bool {
	return fromKind == notBundle && toKind != notBundle && (ref.Origin == sourceKept || ref.Origin == sourceBeside)
}

// afterNewName is what a new name means for the window (a rename, Save
// As): the type and language it says, the source's and drafts' memories
// following (typeAfterRename).
func (w *MainWindow) afterNewName(oldName, oldPath, newName string) {
	newPath := ""
	if w.currentFile != "" {
		newPath = w.currentFile
	}
	if l := languageOf(newName); l != "" && bundleKindOf(newName) == notBundle {
		w.setLanguage(l)
		w.languageSaid = true
	}
	w.typeAfterRename(oldPath, oldName, newPath, newName)
	w.looseAfterRename(oldPath, newPath)
	w.updateTitle()
}

// exists tells whether something is at p.
func exists(p string) bool {
	_, err := os.Lstat(p)
	return p != "" && err == nil
}

// undoRename takes the last rename back (File > Undo Rename), once the
// writes under way are done.
func (w *MainWindow) undoRename() {
	j := w.renameUndo
	if j == nil || !w.canUndoRename() {
		w.setStatus("No rename to take back")
		return
	}
	w.afterBundleWrites(func() {
		if w.renameUndo != j {
			return
		}
		if j.wrote && j.print != w.extrasPrint() {
			dialog.ShowInformation("Undo Rename",
				"Drafts, production files or the source changed since the rename: taking it back would lose that. "+
					"Rename it back instead (File > Rename…).", w.fyneWindow)
			return
		}
		w.renameUndo = nil
		name := filepath.Base(j.to)
		if err := w.rollback(j); err != nil {
			dialog.ShowError(err, w.fyneWindow)
			return
		}
		w.setStatus("Rename taken back: " + name + " is " + filepath.Base(cmp.Or(j.from, w.scriptName(), w.untitledName())) + " again")
	})
}

// canUndoRename tells whether Undo Rename can be given: the script is
// still the file the last rename made.
func (w *MainWindow) canUndoRename() bool {
	j := w.renameUndo
	return j != nil && w.currentFile != "" && absPath(w.currentFile) == absPath(j.to)
}

// extrasPrint is a fingerprint of the script's drafts, production files
// and source as they are now: Undo Rename after a conversion would lose
// what changed since.
func (w *MainWindow) extrasPrint() string {
	h := sha256.New()
	files, _ := w.extrasHome().all()
	for _, f := range files {
		fmt.Fprintf(h, "%s %d\n", f.name, len(f.data))
		h.Write(f.data)
	}
	if !w.markdown {
		if md, err := w.sourceText(w.sourceRef()); err == nil {
			fmt.Fprintf(h, "source %d\n%s", len(md), md)
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// dropRenameUndo lets the last rename's undo go (another document, the
// window closed): what it parked is deleted.
func (w *MainWindow) dropRenameUndo() {
	w.renameUndo.forget()
	w.renameUndo = nil
}
