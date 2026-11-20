package main

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/accolade/internal/prose"
)

// Starting a screenplay from a source (docs/ADAPTATION.md): a story open
// as Markdown (opened, imported, or written here) becomes the source of a
// new script, which starts with a title page naming the story, the
// story's chapters as its sections, and the first scene for its first
// passage, ready to write (Next, as the source pane does it). A script
// with a source but nothing written yet is offered the same in the pane.

// startScreenplayLabel is the File menu's item, and the Markdown
// toolbar's button says the same.
const startScreenplayLabel = "Start a Screenplay from This Source"

// sourceTitle is a story's title: its front matter's title:, else its
// first top heading, else its file's name.
func sourceTitle(md, name string) string {
	lines := strings.Split(md, "\n")
	kinds := syntax.ClassifyMarkdown(lines)
	for i := 0; i < len(kinds) && kinds[i] == syntax.MdFrontMatter; i++ {
		if k, v, ok := strings.Cut(lines[i], ":"); ok && strings.EqualFold(strings.TrimSpace(k), "title") {
			if v = strings.Trim(strings.TrimSpace(v), `"'`); v != "" {
				return v
			}
		}
	}
	for _, p := range passagesOf(md) {
		if p.Kind == prose.Heading && p.Level == 1 && p.Plain != "" {
			return p.Plain
		}
	}
	return strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
}

// scriptFromSource is what a script started from a source holds: a title
// page naming the story, then the story's headings as sections (its title
// too, if it is a heading: what comes under it before the first chapter
// has its place there).
func scriptFromSource(title string, ps []passage, now time.Time) string {
	lines := []string{
		"Title: " + title,
		"Credit: Written by",
		"Author: Your Name",
		"Source: Based on “" + title + "”",
		"Draft date: " + now.Format("January 2, 2006"),
		"",
	}
	for _, h := range sourceHeadings(ps) {
		lines = append(lines, strings.Repeat("#", h.Level)+" "+h.Text, "")
	}
	return strings.Join(lines, "\n")
}

// unwritten reports whether a script has nothing written in it yet: no
// scene, no section, no link (a title page may be there).
func unwritten(text string, starts []string) bool {
	body := text[:bodyEndByte(text)]
	return len(outlineOf(body, starts)) == 0 && len(parseLinks(body)) == 0
}

// startScreenplayFromSource is File > Start a Screenplay from This
// Source, on a Markdown document: a new script with it as its source.
// On the desktop the document stays open, in its own window, and a saved
// one is the source itself, so that it can go on being written; one not
// saved (an import) is kept by Accolade, as the browser keeps any.
func (w *MainWindow) startScreenplayFromSource() {
	if !w.markdown {
		return
	}
	md := w.textEditor.Text()
	name := cmp.Or(w.currentFile, w.suggestedName, "story.md")
	file := ""
	if w.currentFile != "" && w.bundle == nil && !w.hasChanges && !inBrowser {
		file = w.currentFile
	}
	t := w
	if replaceable := w.currentFile == "" && (!w.hasChanges || md == w.filledWith); !replaceable && !inBrowser && w.app != nil {
		t = w.app.newWindow()
	}
	t.newScriptFromSource(md, name, file)
}

// newScriptFromSource makes the window's document a new script with the
// source md (named name; file, if it is to be the source itself).
func (w *MainWindow) newScriptFromSource(md, name, file string) {
	w.cancelAutoSave()
	w.createNewFile(true)
	langNote := ""
	if lang, sure := readingLanguage(md, ""); sure && sceneConf[lang] != nil && lang != w.language {
		w.setLanguage(lang) // the story's language: its scene headings
		langNote = "; the script's language is the story's, " + lang + " (Edit > Script Language…)"
	}
	ps := passagesOf(md)
	title := sourceTitle(md, name)
	w.textEditor.SetText(scriptFromSource(title, ps, time.Now()))
	w.textEditor.SetCursorOffset(len([]rune(w.textEditor.Text())))
	w.suggestedName = strings.TrimSuffix(filepath.Base(name), filepath.Ext(name)) + ".fountain"
	w.hasChanges = true
	if file != "" {
		w.rememberSource("", file)
	} else if err := w.useSourceText(name, md); err != nil {
		w.setStatus(err.Error())
	}
	w.updateTitle()
	w.updateStatusBar()
	w.showSource()
	w.sourcePane().firstScene()
	w.setStatus("A screenplay from “" + title + "”: its chapters as sections, the first scene started; write its heading" + langNote)
}

// firstScene starts the script's first scene, for the source's first
// passage (Next), or offers it, as the preference says.
func (s *sourcePane) firstScene() {
	s.load()
	if !slices.ContainsFunc(s.passages, passage.counted) {
		return
	}
	s.nextPassage()
}

// seedScript puts the source's headings in an unwritten script as its
// sections, after its title page, and starts the first scene: one undo
// step with the sections; the scene its own.
func (s *sourcePane) seedScript() {
	e := s.w.textEditor
	text := e.Text()
	if !unwritten(text, s.w.sceneStarts()) {
		return
	}
	var b strings.Builder
	for _, h := range sourceHeadings(s.passages) {
		b.WriteString(strings.Repeat("#", h.Level) + " " + h.Text + "\n\n")
	}
	if b.Len() > 0 {
		end := len([]rune(strings.TrimRight(string([]rune(text)[:bodyEnd(text)]), "\n")))
		lead := ""
		if end > 0 {
			lead = "\n\n"
		}
		e.Edit(func(buf *buffer.Buffer) {
			buf.Group(func() {
				buf.Replace(end, end, lead+strings.TrimRight(b.String(), "\n"))
				buf.SetCursor(end+len([]rune(lead+strings.TrimRight(b.String(), "\n"))), false)
			})
		})
	}
	s.firstScene()
}
