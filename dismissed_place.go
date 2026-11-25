package main

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strings"

	"fyne.io/fyne/v2/theme"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Where dismissed findings are kept (LOOP.md, REVISION FOLLOW-UPS; Joop
// 2026-11-25: "support both options and enable keeping it clean"), a
// setting:
//
//   - beside the document (the default): in a bundle its
//     assets/production/dismissed.txt; beside a loose file on the desktop
//     Name.fountain.dismissed; for a document with no place beside it (not
//     saved yet, or in the browser) the extras Accolade keeps for it
//     (extras.go), which follow it when it is saved;
//   - in the document: the paragraph in its notes block (dismissedIn), as
//     before.
//
// Both say it the same way (dismissedHead, a line a finding), so deleting
// a line brings a finding back either way. Both are read, whatever the
// setting; a dismissal writes the chosen one and empties the other, and
// switching moves the open documents' dismissals at once.

// dismissedInDocPref: dismissed findings in the document's notes block
// (off: beside it).
const dismissedInDocPref = "dismissed-in-document"

// dismissedExtra is the extras' file of dismissed findings.
const dismissedExtra = productionDir + "dismissed.txt"

// dismissedSidecar is the file beside a loose document that keeps its
// dismissed findings.
func dismissedSidecar(file string) string { return file + ".dismissed" }

// dismissCache is what was read beside a document (key: where).
type dismissCache struct {
	key string
	set map[string]bool
}

// dismissInDocument tells whether dismissals go in the document.
func (w *MainWindow) dismissInDocument() bool {
	return w.settings != nil && w.settings.GetBoolean(dismissedInDocPref)
}

// looseSidecar is the sidecar of the window's document, "" when it has
// none (a bundle, not saved, the browser).
func (w *MainWindow) looseSidecar() string {
	if w.bundle != nil || w.currentFile == "" || inBrowser {
		return ""
	}
	return dismissedSidecar(absPath(w.currentFile))
}

// besideKey says where the window's document's dismissals beside it are.
func (w *MainWindow) besideKey() string {
	if w.bundle != nil {
		return "bundle\t" + w.bundle.Path
	}
	k := "file\t" + w.looseSidecar()
	if h := w.plainExtrasHome(w.currentFile); h.kind != extrasNone {
		k += "\t" + string(h.dir)
	}
	return k
}

// parseDismissed reads a file of dismissed findings.
func parseDismissed(data []byte) map[string]bool {
	d, _ := dismissedIn(string(data))
	return d
}

// besideDismissed are the findings dismissed beside the window's
// document (read once per place).
func (w *MainWindow) besideDismissed() map[string]bool {
	key := w.besideKey()
	if c := w.dismissed; c != nil && c.key == key {
		return c.set
	}
	set := map[string]bool{}
	if w.bundle != nil {
		if data, err := w.bundle.readExtra(dismissedExtra); err == nil {
			maps.Copy(set, parseDismissed(data))
		}
	} else {
		if s := w.looseSidecar(); s != "" {
			if data, err := os.ReadFile(s); err == nil {
				maps.Copy(set, parseDismissed(data))
			}
		}
		if h := w.plainExtrasHome(w.currentFile); h.kind != extrasNone {
			if data, err := h.readExtra(dismissedExtra); err == nil {
				maps.Copy(set, parseDismissed(data))
			}
		}
	}
	w.dismissed = &dismissCache{key, set}
	return set
}

// inDocDismissed are the findings the window's document's notes block
// keeps dismissed, and its notes without them.
func (w *MainWindow) inDocDismissed() (map[string]bool, string) {
	if w.textEditor == nil {
		return nil, ""
	}
	m := w.notesMarks()
	return dismissedIn(strings.ReplaceAll(m.of(w.textEditor.Text()), m.escaped, m.close))
}

// allDismissed are the findings dismissed in either place.
func (w *MainWindow) allDismissed() map[string]bool {
	in, _ := w.inDocDismissed()
	beside := w.besideDismissed()
	if len(in) == 0 {
		return beside
	}
	out := maps.Clone(in)
	maps.Copy(out, beside)
	return out
}

// storeDismissed keeps keys as the document's dismissed findings, where
// the setting says, the other place emptied (the notes block's change one
// undo step).
func (w *MainWindow) storeDismissed(keys []string) error {
	slices.Sort(keys)
	in := w.dismissInDocument()
	besideKeys := keys
	if in {
		besideKeys = nil
	}
	err := w.writeBeside(besideKeys)
	if err != nil && !in {
		return err // the notes block keeps them: nothing lost
	}
	old, rest := w.inDocDismissed()
	if !in {
		keys = nil
	}
	if len(old) == 0 && len(keys) == 0 {
		return err
	}
	m := w.notesMarks()
	from, block := m.edit(w.textEditor.Text(), withDismissed(rest, keys))
	w.textEditor.Edit(func(b *buffer.Buffer) {
		cursor := b.Cursor()
		b.Group(func() {
			replaceLeast(b, from, b.Len(), block)
			b.SetCursor(min(cursor, from), false)
		})
	})
	return err
}

// writeBeside writes the dismissed findings beside the document (none:
// the file taken away).
func (w *MainWindow) writeBeside(keys []string) error {
	set := map[string]bool{}
	for _, k := range keys {
		set[k] = true
	}
	had := w.besideDismissed()
	if len(had) == 0 && len(keys) == 0 {
		return nil
	}
	var data []byte
	if len(keys) > 0 {
		data = []byte(withDismissed("", slices.Clone(keys)) + "\n")
	}
	key := w.besideKey()
	defer func() { w.dismissed = &dismissCache{key, set} }()
	if w.bundle != nil {
		var files []bundleFile
		var gone []string
		if data != nil {
			files = []bundleFile{{dismissedExtra, data}}
		} else {
			gone = []string{dismissedExtra}
		}
		w.queueBundle(bundleJob{run: func(b *textBundle) (*textBundle, error) {
			if b == nil {
				return nil, errors.New("the script's bundle is gone")
			}
			return saveBundleFilesWithout(b, files, gone, false)
		}})
		return nil
	}
	h := w.plainExtrasHome(w.currentFile)
	if s := w.looseSidecar(); s != "" {
		if h.kind != extrasNone { // kept there before it was saved: here now
			if err := h.dir.remove([]string{dismissedExtra}); err != nil {
				return err
			}
		}
		if data == nil {
			if err := os.Remove(s); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			return nil
		}
		return os.WriteFile(s, data, 0o644)
	}
	if data == nil {
		if h.kind == extrasNone {
			return nil
		}
		return h.dir.remove([]string{dismissedExtra})
	}
	h, err := w.ensureExtrasHome()
	if err != nil {
		return err
	}
	return h.dir.write([]bundleFile{{dismissedExtra, data}})
}

// moveDismissed puts the document's dismissed findings where the setting
// says now.
func (w *MainWindow) moveDismissed() error {
	if w.textEditor == nil {
		return nil
	}
	all := w.allDismissed()
	if len(all) == 0 {
		return nil
	}
	err := w.storeDismissed(slices.Collect(maps.Keys(all)))
	w.openChanged()
	return err
}

// dismissedAfterRename follows a loose document's sidecar to its new
// name (a rename takes it along, Save As copies it); into a bundle it
// goes once the bundle is written.
func (w *MainWindow) dismissedAfterRename(oldPath, newPath string) {
	if oldPath == "" || newPath == "" || inBrowser || sameFile(oldPath, newPath) {
		return
	}
	from := dismissedSidecar(absPath(oldPath))
	data, err := os.ReadFile(from)
	if err != nil {
		return
	}
	gone := w.oldGoes(oldPath)
	if bundleKindOf(newPath) == notBundle || w.isPlainBundle(newPath) {
		to := dismissedSidecar(absPath(newPath))
		if _, err := os.Stat(to); err == nil {
			return // the new name's own
		}
		if gone && os.Rename(from, to) == nil {
			return
		}
		_ = os.WriteFile(to, data, 0o644)
		return
	}
	w.afterBundleWrites(func() {
		if w.bundle == nil || !sameFile(w.bundle.Path, newPath) {
			return
		}
		keys := slices.Collect(maps.Keys(parseDismissed(data)))
		if w.writeBeside(keys) == nil && gone {
			_ = os.Remove(from)
		}
	})
}

// dismissedPlaceCommand is the setting, on the Revision tab.
func dismissedPlaceCommand() *command {
	return &command{id: "revision.dismissInDoc", label: "Keep Dismissed Findings in the Document", ribbonLabel: "Dismissed in Document",
		ex: "dismissed-in-document", icon: theme.VisibilityOffIcon,
		tip:     "On: dismissed findings are a paragraph in the document's notes block. Off (the default): beside it (in a bundle, or a .dismissed file next to the file), the document kept clean. Switching moves them",
		checked: (*MainWindow).dismissInDocument,
		run: func(w *MainWindow) {
			on := !w.dismissInDocument()
			w.changeSetting(dismissedInDocPref, func(s *Settings) { s.SetBoolean(dismissedInDocPref, on) })
			var failed error
			ws := w.windows()
			if !slices.Contains(ws, w) {
				ws = append(ws, w)
			}
			for _, o := range ws {
				if err := o.moveDismissed(); err != nil {
					failed = err
				}
			}
			switch {
			case failed != nil:
				w.setStatus("Could not move the dismissed findings: " + failed.Error())
			case on:
				w.setStatus("Dismissed findings are kept in the document's notes block now")
			default:
				w.setStatus("Dismissed findings are kept beside the document now, the document clean")
			}
		}}
}
