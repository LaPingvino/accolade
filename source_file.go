package main

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Where a script's source is kept, in layers, looked for in this order:
//
//  1. Inside the script: a boneyard block at its end, before the notes,
//     "/* Source … */" (Format > Keep Source in the Script, off by
//     default). Other Fountain apps ignore a boneyard, so one .fountain
//     file carries the script and its source anywhere.
//  2. A copy Accolade keeps: File > Choose Source… reads a Markdown file
//     (or a Word, OpenDocument, web page or text file, through
//     internal/prose) and keeps it as Markdown in Accolade's own folder,
//     remembered per script in settings.json ("source-files", as
//     doctype.go keeps document types). On the desktop the folder is the
//     app's data folder; in the browser, which has no folder of the
//     user's (an opened file is an upload), the settings folder, which
//     the page keeps (web/index.html's memfsPersist). File > Save Source
//     As… gets it out again.
//  3. Beside the script, on the desktop: story.md (or .markdown) next to
//     story.fountain.
//
// The pane says which one is in use.

// sourceOrigin is where a source is kept.
type sourceOrigin int

const (
	sourceNone   sourceOrigin = iota
	sourceInside              // in the script
	sourceKept                // a copy Accolade keeps (or a file chosen for the script)
	sourceBeside              // the file beside the script
)

// sourceRef is a script's source: where it is, and its file (for
// sourceKept and sourceBeside).
type sourceRef struct {
	Origin sourceOrigin
	Path   string
}

// describe is how the pane names it.
func (r sourceRef) describe() string {
	switch r.Origin {
	case sourceInside:
		return "Kept in the script"
	case sourceKept:
		if strings.HasPrefix(r.Path, keptSourcesDir()+string(filepath.Separator)) {
			return filepath.Base(r.Path) + " · kept by Accolade"
		}
		return filepath.Base(r.Path) + " · chosen"
	case sourceBeside:
		return filepath.Base(r.Path) + " · beside the script"
	}
	return "Source"
}

// sourceMarks are the source block's, as the notes block's (notes.go).
var sourceMarks = newNotesMarks("/* Source", "*/", "* /")

// sourceBlock is the source kept in the script: the block closing just
// before the notes (or at the end); where it is, in runes, from the
// blank lines before it to the notes, and the Markdown.
func sourceBlock(text string) (start, end int, md string, ok bool) {
	end = notesStart(text)
	before := string([]rune(text)[:end])
	i, inside := sourceMarks.at(before)
	if i < 0 {
		return 0, 0, "", false
	}
	md = strings.ReplaceAll(inside, sourceMarks.escaped, sourceMarks.close)
	if md != "" && !strings.HasSuffix(md, "\n") { // the block's last line ends as a file's
		md += "\n"
	}
	return len([]rune(before[:i])), end, md, true
}

// withSourceBlock is the edit that puts md in the script as its source
// block (replacing one there): where (runes) and what.
func withSourceBlock(text, md string) (start, end int, block string) {
	if s, e, _, ok := sourceBlock(text); ok {
		start, end = s, e
	} else {
		end = notesStart(text)
		start = len([]rune(strings.TrimRight(string([]rune(text)[:end]), "\n")))
	}
	block = strings.TrimRight(sourceMarks.text(md), "\n")
	if end == len([]rune(text)) { // no notes after it: the script ends with a new line
		block += "\n"
	}
	return start, end, block
}

// keptSourcesDir is the folder of the sources Accolade keeps: the app's
// data folder on the desktop ($XDG_DATA_HOME/accolade, or
// ~/.local/share/accolade on Linux; beside the settings elsewhere), the
// kept settings folder in the browser.
func keptSourcesDir() string {
	if !inBrowser {
		if d := os.Getenv("XDG_DATA_HOME"); d != "" {
			return filepath.Join(d, "accolade", "sources")
		}
		if runtime.GOOS == "linux" {
			if home, err := os.UserHomeDir(); err == nil {
				return filepath.Join(home, ".local", "share", "accolade", "sources")
			}
		}
	}
	return filepath.Join(getConfigDir(), "sources")
}

// keepSourceCopy keeps md as a source in Accolade's folder, as name
// (its extension made .md), in a folder of its own so the name stays
// what it was (Save Source As offers it).
func keepSourceCopy(name, md string) (string, error) {
	name = strings.TrimSuffix(filepath.Base(name), filepath.Ext(name)) + ".md"
	dir := filepath.Join(keptSourcesDir(), strconv.FormatInt(time.Now().UnixNano(), 36))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	p := filepath.Join(dir, name)
	return p, writeFileAtomic(p, []byte(md))
}

// sourceEntry is how a script's source is remembered in the settings.
func sourceEntry(script, source string) string { return script + "\t" + source }

// companionOf is the Markdown file with a script's name beside it
// (story.fountain: story.md, else story.markdown), if there is one.
func companionOf(script string) string {
	if script == "" {
		return ""
	}
	base := strings.TrimSuffix(script, filepath.Ext(script))
	for _, ext := range []string{".md", ".markdown", ".MD"} {
		if p := base + ext; p != script {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p
			}
		}
	}
	return ""
}

// chosenSource is the source file remembered for a script, if any.
func (w *MainWindow) chosenSource(script string) string {
	if script == "" {
		return w.unsavedSource
	}
	if w.settings == nil {
		return ""
	}
	script = absPath(script)
	for _, e := range w.settings.GetStringSlice("source-files") {
		if s, ok := strings.CutPrefix(e, script+"\t"); ok {
			return s
		}
	}
	return ""
}

// sourceRef is the window's script's source, looked for in the layers'
// order; none for a Markdown document, which is a source itself.
func (w *MainWindow) sourceRef() sourceRef {
	if w.markdown {
		return sourceRef{}
	}
	if _, _, _, ok := sourceBlock(w.textEditor.Text()); ok {
		return sourceRef{Origin: sourceInside}
	}
	if s := w.chosenSource(w.currentFile); s != "" {
		if _, err := os.Stat(s); err == nil {
			return sourceRef{Origin: sourceKept, Path: s}
		}
	}
	if c := companionOf(w.currentFile); c != "" {
		return sourceRef{Origin: sourceBeside, Path: c}
	}
	return sourceRef{}
}

// rememberSource keeps the source file for a script ("" forgets it).
func (w *MainWindow) rememberSource(script, source string) {
	if script == "" {
		w.unsavedSource = source
		return
	}
	if w.settings == nil {
		return
	}
	script = absPath(script)
	if source != "" {
		source = absPath(source)
	}
	var list []string
	if source != "" {
		list = append(list, sourceEntry(script, source))
	}
	old := w.settings.GetStringSlice("source-files")
	for _, e := range old {
		if !strings.HasPrefix(e, script+"\t") && len(list) < maxDocumentTypes {
			list = append(list, e)
		}
	}
	if len(list) == 0 && len(old) == 0 {
		return
	}
	w.settings.SetStringSlice("source-files", list)
	_ = w.settings.Save()
}

// sourceAfterRename keeps a script's source through a rename or Save As:
// the old name's remembered source, or the file beside the old name,
// becomes the new name's, unless the new name has one beside it.
func (w *MainWindow) sourceAfterRename(oldPath, newPath string) {
	if newPath == "" || (oldPath != "" && absPath(oldPath) == absPath(newPath)) {
		return
	}
	source := w.chosenSource(oldPath)
	if source == "" {
		source = companionOf(oldPath)
	}
	if oldPath != "" {
		w.rememberSource(oldPath, "")
	} else {
		w.unsavedSource = ""
	}
	if source != "" && companionOf(newPath) == "" {
		w.rememberSource(newPath, source)
	}
}

// useSource makes a file the script's source: read as Markdown
// (toMarkdown: Markdown and text as they are, Word, OpenDocument and web
// pages converted), kept in Accolade's folder and remembered.
func (w *MainWindow) useSource(path string) error {
	md, _, _, err := toMarkdown(path)
	if err != nil {
		return err
	}
	kept, err := keepSourceCopy(filepath.Base(path), md)
	if err != nil {
		return fmt.Errorf("could not keep the source: %v", err)
	}
	w.rememberSource(w.currentFile, kept)
	return nil
}

// sourceText is the source's Markdown as it is now.
func (w *MainWindow) sourceText(r sourceRef) (string, error) {
	switch r.Origin {
	case sourceInside:
		_, _, md, _ := sourceBlock(w.textEditor.Text())
		return md, nil
	case sourceKept, sourceBeside:
		data, err := os.ReadFile(r.Path)
		text, _ := decodeScript(data)
		return text, err
	}
	return "", nil
}

// sourceName is the name Save Source As offers.
func (w *MainWindow) sourceName(r sourceRef) string {
	if r.Path != "" {
		return filepath.Base(r.Path)
	}
	name := cmp.Or(w.currentFile, w.suggestedName, w.untitledName())
	return strings.TrimSuffix(filepath.Base(name), filepath.Ext(name)) + ".md"
}

// keepSourceInside moves the source into the script (on) or out of it
// into a copy Accolade keeps (off); each one edit, undone as one.
func (w *MainWindow) keepSourceInside(on bool) error {
	r := w.sourceRef()
	e := w.textEditor
	if on {
		if r.Origin == sourceInside {
			return nil
		}
		if r.Origin == sourceNone {
			return fmt.Errorf("the script has no source yet: File > Choose Source… first")
		}
		md, err := w.sourceText(r)
		if err != nil {
			return err
		}
		start, end, block := withSourceBlock(e.Text(), md)
		e.Replace(start, end, block)
		w.refreshMenu() // Keep Source in the Script, checked
		return nil
	}
	if r.Origin != sourceInside {
		return nil
	}
	md, _ := w.sourceText(r)
	kept := w.chosenSource(w.currentFile)
	if !strings.HasPrefix(kept, keptSourcesDir()+string(filepath.Separator)) { // a copy of its own
		var err error
		if kept, err = keepSourceCopy(w.sourceName(r), md); err != nil {
			return fmt.Errorf("could not keep the source: %v", err)
		}
	} else if err := writeFileAtomic(kept, []byte(md)); err != nil {
		return err
	}
	w.rememberSource(w.currentFile, kept)
	start, end, _, _ := sourceBlock(e.Text())
	tail := ""
	if end == len([]rune(e.Text())) {
		tail = "\n"
	}
	e.Replace(start, end, tail)
	w.refreshMenu()
	return nil
}
