package main

import (
	"os"
)

// Saving a bundle that has extras (drafts, production files) as a plain
// .fountain: the text keeps what is in it (scene numbers, OMITTED, the
// blocks); the extras go where the writer chose (docs/PRODUCTION.md §4):
// a copy Accolade keeps (the default), Name.production/ beside the file
// (desktop only), or nowhere, after asking. Reopening the plain file
// finds them again (extras.go: the kept copy by its path, else the
// folder beside).

// flattenChoice is where a bundle's extras go as it is saved plain.
type flattenChoice int

const (
	flattenKept   flattenChoice = iota // a copy Accolade keeps
	flattenFolder                      // Name.production/ beside the file
	flattenDrop                        // left out
)

// placeFlattened puts bundle b's extras where how says, for the plain
// script at p.
func (w *MainWindow) placeFlattened(b *textBundle, p string, how flattenChoice) error {
	if len(b.extras) == 0 {
		return nil
	}
	old := w.chosenExtras(p) // a kept copy from an earlier save under this name: replaced
	if how != flattenKept && old != "" {
		w.rememberExtras(p, "")
		if isKeptExtras(old) {
			os.RemoveAll(old)
		}
	}
	if how == flattenDrop {
		return nil
	}
	files, err := (extrasHome{kind: extrasBundle, b: b}).all()
	if err != nil {
		return err
	}
	if how == flattenFolder && !inBrowser {
		return extrasFolder(productionFolderOf(p)).write(files)
	}
	dir, err := newKeptExtras(p)
	if err != nil {
		return err
	}
	if err := dir.write(files); err != nil {
		os.RemoveAll(string(dir))
		return err
	}
	w.rememberExtras(p, string(dir))
	if isKeptExtras(old) && old != string(dir) {
		os.RemoveAll(old)
	}
	return nil
}
