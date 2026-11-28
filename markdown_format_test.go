// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// mdWindow is a Markdown document with text, the selection [from, to)
// ("|" in text marks the cursor when from is -1).
func mdWindow(t *testing.T, text string, from, to int) *MainWindow {
	t.Helper()
	if from < 0 {
		from = strings.Index(text, "|")
		text = strings.Replace(text, "|", "", 1)
		to = from
	}
	w := newTestWindow(t, "")
	w.setDocumentType(true)
	w.textEditor.SetText(text)
	w.textEditor.Navigate(func(b *buffer.Buffer) { b.Select(from, to) })
	w.fyneWindow.Canvas().Focus(w.textEditor)
	return w
}

// runMD runs a Markdown command as its menu item does.
func runMD(w *MainWindow, id string) { w.do(cmd(id))() }

// Each Markdown command: on, off again (where it toggles), the selection
// or the cursor's line, one undo step back to the text before.
func TestMarkdownCommands(t *testing.T) {
	for _, tc := range []struct {
		name, id, text string
		from, to       int
		want, again    string // again: run twice ("" if not checked)
	}{
		{"heading on the line", "md.heading2", "Intro|duction\n\nText.", -1, 0, "## Introduction\n\nText.", "Introduction\n\nText."},
		{"heading of another level", "md.heading1", "### Intro|\n", -1, 0, "# Intro\n", "Intro\n"},
		{"normal text", "md.heading0", "## Intro|\n", -1, 0, "Intro\n", "Intro\n"},
		{"headings on the lines", "md.heading3", "one\ntwo\n", 0, 7, "### one\n### two\n", "one\ntwo\n"},
		{"heading 6", "md.heading6", "deep|", -1, 0, "###### deep", "deep"},
		{"bold", "format.bold", "a word here", 2, 6, "a **word** here", "a word here"},
		{"italic", "format.italic", "a word here", 2, 6, "a *word* here", "a word here"},
		{"strikethrough", "md.strike", "a word here", 2, 6, "a ~~word~~ here", "a word here"},
		{"inline code", "md.code", "call f() now", 5, 8, "call `f()` now", "call f() now"},
		{"code block on lines", "md.codeBlock", "x := 1\ny := 2", 0, 13, "```\nx := 1\ny := 2\n```", "x := 1\ny := 2"},
		{"code block on the cursor's line", "md.codeBlock", "before\n\nx|\n\nafter", -1, 0, "before\n\n```\nx\n```\n\nafter", "before\n\nx\n\nafter"},
		{"empty code block", "md.codeBlock", "|", -1, 0, "```\n\n```", ""},
		{"bullets", "md.bullets", "one\ntwo", 0, 7, "- one\n- two", "one\ntwo"},
		{"bullets over numbers", "md.bullets", "1. one\n2. two", 0, 13, "- one\n- two", "one\ntwo"},
		{"numbers", "md.numbers", "one\ntwo\nthree", 0, 13, "1. one\n2. two\n3. three", "one\ntwo\nthree"},
		{"tasks", "md.tasks", "buy| milk", -1, 0, "- [ ] buy milk", "buy milk"},
		{"tasks over bullets", "md.tasks", "- buy| milk", -1, 0, "- [ ] buy milk", "buy milk"},
		{"quote", "md.quote", "one\n\ntwo", 0, 8, "> one\n\n> two", "one\n\ntwo"},
		{"rule", "md.rule", "Text.|", -1, 0, "Text.\n\n---\n", ""},
		{"footnote", "md.footnote", "A claim.| More.", -1, 0, "A claim.[^1] More.\n\n[^1]: ", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := mdWindow(t, tc.text, tc.from, tc.to)
			before := w.textEditor.Text()
			runMD(w, tc.id)
			if got := w.textEditor.Text(); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			if tc.again != "" {
				runMD(w, tc.id)
				if got := w.textEditor.Text(); got != tc.again {
					t.Errorf("again: got %q, want %q", got, tc.again)
				}
				w.undo()
			}
			w.undo()
			if got := w.textEditor.Text(); got != before {
				t.Errorf("undone: %q, want %q", got, before)
			}
		})
	}
}

// A second footnote comes after the first, under it.
func TestMarkdownSecondFootnote(t *testing.T) {
	w := mdWindow(t, "One.[^1] Two.|\n\n[^1]: First.\n", -1, 0)
	runMD(w, "md.footnote")
	if got, want := w.textEditor.Text(), "One.[^1] Two.[^2]\n\n[^1]: First.\n[^2]: "; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Link… and Image… ask the address; the selection is the text; a link
// selected goes back to its text. Table… asks rows and columns.
func TestMarkdownLinkImageTable(t *testing.T) {
	b := buffer.New("see the docs")
	b.Select(8, 12)
	insertMarkdownLink(b, "https://example.org", false)
	if got := b.Text(); got != "see the [docs](https://example.org)" {
		t.Fatalf("link: %q", got)
	}
	b.Select(8, len([]rune(b.Text())))
	insertMarkdownLink(b, "x", false)
	if got := b.Text(); got != "see the docs" {
		t.Errorf("link taken away: %q", got)
	}
	b = buffer.New("")
	insertMarkdownLink(b, "cat.png", true)
	if got, sel := b.Text(), b.SelectedText(); got != "![description](cat.png)" || sel != "description" {
		t.Errorf("image: %q, selected %q", got, sel)
	}
	if got := tableSkeleton(2, 2); got != "| Column 1 | Column 2 |\n| --- | --- |\n|   |   |\n|   |   |" {
		t.Errorf("table: %q", got)
	}

	// through the dialog: the address typed, Insert
	w := mdWindow(t, "see the docs", 8, 12)
	runMD(w, "md.link")
	form := topForm(t, w)
	test.Type(form.entries[0], "https://example.org")
	form.confirm.OnTapped()
	if got := w.textEditor.Text(); got != "see the [docs](https://example.org)" {
		t.Errorf("Link… wrote %q", got)
	}
	w.undo()
	if got := w.textEditor.Text(); got != "see the docs" {
		t.Errorf("Link… undone: %q", got)
	}
	w = mdWindow(t, "Text.|", -1, 0)
	runMD(w, "md.table")
	form = topForm(t, w)
	form.entries[0].SetText("1")
	form.entries[1].SetText("2")
	form.confirm.OnTapped()
	if got := w.textEditor.Text(); got != "Text.\n\n| Column 1 | Column 2 |\n| --- | --- |\n|   |   |\n" {
		t.Errorf("Table… wrote %q", got)
	}
}

// Ctrl+1 to Ctrl+6 are the headings in Markdown, the elements in a
// screenplay; Ctrl+L Link…; the Markdown commands are Markdown's alone.
func TestMarkdownKeys(t *testing.T) {
	w := mdWindow(t, "Title|\n", -1, 0)
	m := w.fyneWindow.MainMenu()
	keys := map[string]string{}
	var walk func(items []*fyne.MenuItem)
	walk = func(items []*fyne.MenuItem) {
		for _, it := range items {
			if it.Shortcut != nil {
				keys[shortcutText(it.Shortcut)] = it.Label
			}
		}
	}
	for _, menu := range m.Items {
		walk(menu.Items)
	}
	for k, want := range map[string]string{"Primary+1": "Heading 1", "Primary+6": "Heading 6", "Primary+0": "Normal Text", "Primary+L": "Link…", "Primary+K": "Command Palette…"} {
		if keys[k] != want {
			t.Errorf("Markdown %s = %q, want %q", k, keys[k], want)
		}
	}
	w.setDocumentType(false)
	keys = map[string]string{}
	for _, menu := range w.fyneWindow.MainMenu().Items {
		walk(menu.Items)
	}
	if keys["Primary+1"] != elementScene || keys["Primary+L"] != "" {
		t.Errorf("screenplay Ctrl+1 = %q, Ctrl+L = %q", keys["Primary+1"], keys["Primary+L"])
	}
	for _, c := range commands().commands {
		if strings.HasPrefix(c.id, "md.") && c.applies(cmdEnv{}) {
			t.Errorf("%s applies to a screenplay", c.id)
		}
	}
}

// formParts are a form dialog's entries and its confirm button.
type formParts struct {
	entries []*widget.Entry
	confirm *widget.Button
}

// topForm is the form dialog over the window.
func topForm(t *testing.T, w *MainWindow) formParts {
	t.Helper()
	top := w.fyneWindow.Canvas().Overlays().Top()
	if top == nil {
		t.Fatal("no dialog")
	}
	var f formParts
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		switch x := o.(type) {
		case *widget.Entry:
			f.entries = append(f.entries, x)
		case *widget.Button:
			if x.Text == "Insert" {
				f.confirm = x
			}
		case *widget.Form:
			for _, it := range x.Items {
				walk(it.Widget)
			}
		case *fyne.Container:
			for _, k := range x.Objects {
				walk(k)
			}
		case fyne.Widget:
			r := test.TempWidgetRenderer(t, x)
			for _, k := range r.Objects() {
				walk(k)
			}
		}
	}
	walk(top)
	if len(f.entries) == 0 || f.confirm == nil {
		t.Fatalf("not a form: %d entries", len(f.entries))
	}
	return f
}
