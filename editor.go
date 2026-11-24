package main

import (
	"github.com/LaPingvino/accolade/internal/structure"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/lexington/rules"
	"strings"
	"unicode"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// newScriptEditor creates the screenplay editor (internal/editor) with
// Accolade's behaviour: Fountain colouring, and on Enter the completed line
// is formatted and the next one indented (dialogue after a character
// name). Formatting only touches the line just completed and is one undo
// step together with the line break.
func newScriptEditor(w *MainWindow) *editor.ScriptEditor {
	e := editor.New("")
	e.Syntax = true
	e.OnEnter = func() bool { return completeLine(w, e) }
	if w != nil {
		e.OnTaggedStep = w.onTaggedStep // an issue's step: the drafts store follows (issue_undo.go)
	}
	tab := &completer{starts: w.sceneStarts}
	if w != nil {
		tab.first = func(cursor int) ([]string, []string) { return w.suggester().firsts(cursor) }
		tab.lang = func() rules.Language { return w.lex().Language(w.language) }
		tab.chosen = func(*completion) int { return w.compl.chosenNow() }
		w.complete = tab
	}
	e.OnTab = func() bool {
		if w != nil && w.scriptFolds.cycleHere() {
			return true
		}
		if w != nil && w.compl.tab() { // the completion list's chosen one, while it is open (complete_list.go)
			return true
		}
		if w != nil && w.compl.open() && tab.tab(e) { // going through the completions shown
			return true
		}
		if w != nil && !w.markdown && w.suggester().tab() { // from the source (suggest.go)
			return true
		}
		if (w == nil || !w.markdown) && tab.tab(e) { // names and headings: a screenplay's
			return true
		}
		return w.flowTab() // the Final Draft flow (keyflow.go)
	}
	vi := attachVim(w, e, true)
	if w != nil {
		w.scriptFolds = newFolds(w, e, w.outlineItems)
		w.scriptFolds.blocks = true // Accolade's own blocks fold away (block_folds.go)
		e.Chips = w.editorChips     // and its links and storyboard frames are chips (link_chips.go, storyboard.go)
		e.OnChipPressed = w.chipPressed
		e.OnChipTapped = w.chipTapped
		e.Tags = w.guessTags // a scene's guessed link beside its heading (linkup.go)
		e.OnTagTapped = w.guessTagTapped
		e.Marks = w.marksShown // the revisions' colours, the notes' marks (notes_marks.go)
		e.OnPeer = w.peerTip   // whose bar in the gutter (collab_cursors.go)
		e.Inks = w.breakdownInksShown
		e.OnSecondaryTapped = w.showTagMenu
		w.vim = vi
	}
	return e
}

// attachVim gives an editor the window's keys: vim or helix as the
// preferences say (vim.go), the menu's shortcuts in full screen, and
// brackets closed as they are typed. script: the screenplay editor, whose
// vim moves through the outline (a document's does not); each folds its
// own headings.
func attachVim(w *MainWindow, e *editor.ScriptEditor, script bool) *vim {
	vi := newVim(e)
	vi.on = func() bool { return w != nil && w.editorKeys() != "standard" }
	vi.helix = func() bool { return w != nil && w.editorKeys() == "helix" }
	vi.status = func(s string) {
		if w != nil {
			w.setStatus(s)
		}
	}
	vi.search = func() {
		if w != nil {
			w.showFind()
		}
	}
	vi.ex = func(visual [2]int) {
		if w != nil {
			w.showEx(visual)
		}
	}
	vi.org = func(cmd string) bool { return script && w != nil && w.org(cmd) }
	vi.tree = func() *structure.Tree { // the script as the window reads it (its language's headings)
		if !script || w == nil || w.markdown {
			return nil
		}
		tr, _ := w.structTree(e.Text())
		return tr
	}
	vi.exWith = func(visual [2]int, text string) {
		if w != nil {
			w.showExWith(visual, text)
		}
	}
	vi.fold = func(cmd rune) {
		if w == nil {
			return
		}
		f := w.foldsOf(e)
		if f == nil {
			return
		}
		switch cmd {
		case 'a':
			f.fold(false, true)
		case 'c':
			f.fold(false, false)
		case 'o':
			f.fold(true, false)
		case 'R':
			f.foldAll(true)
		case 'M':
			f.foldAll(false)
		}
	}
	vi.prompt = func(label string, done func(string)) {
		if w != nil {
			w.showPrompt(label, done)
		}
	}
	e.OnKey = func(k *fyne.KeyEvent) bool {
		if script && w != nil && w.compl.key(k) { // the completion list's ↑↓, Enter, Esc (complete_list.go)
			return true
		}
		// Esc puts a suggestion away; vim's insert mode ends too
		if script && k.Name == fyne.KeyEscape && w != nil && w.suggest != nil && w.suggest.dismiss() && !vi.active() {
			return true
		}
		if script && w != nil && w.citeKey(k) { // the @ list's keys while it is open (cite_complete.go)
			return true
		}
		return vi.key(k)
	}
	e.OnShortcut = func(s fyne.Shortcut) bool {
		if w != nil && w.menuShortcut(s) {
			return true
		}
		return vi.shortcut(s)
	}
	e.OnRune = func(r rune) bool {
		if vi.rune(r) {
			return true
		}
		if script && unicode.IsLower(r) && w.cueHere() { // a cue Tab started: in capitals
			e.Edit(func(b *buffer.Buffer) { b.Insert(string(unicode.ToUpper(r))) })
			return true
		}
		if w == nil || w.settings == nil || !w.settings.GetBoolean("auto-close-brackets") {
			return false
		}
		return closeBracket(e, r)
	}
	if w != nil { // Alt let go alone: the ribbon's key tips (ribbon.go)
		e.OnKeyUp = func(k *fyne.KeyEvent) { w.altUp(k.Name) }
		e.OnFocusLost = w.altCancel
	}
	return vi
}

// brackets are the pairs closed as they are typed: a parenthetical, a
// note ([[...]]).
var brackets = map[rune]rune{'(': ')', '[': ']'}

// closeBracket types an opening bracket with its closing one after the
// cursor (or around the selection), and steps over a closing bracket
// that is already there. It reports false for other characters.
func closeBracket(e *editor.ScriptEditor, r rune) bool {
	if closing, ok := brackets[r]; ok {
		e.Edit(func(b *buffer.Buffer) {
			b.Group(func() {
				start, end := b.Selection()
				sel := b.Slice(start, end)
				b.Replace(start, end, string(r)+sel+string(closing))
				if sel == "" {
					b.SetCursor(start+1, false)
				} else {
					b.Select(start+1, start+1+len([]rune(sel)))
				}
			})
		})
		return true
	}
	for _, closing := range brackets {
		if r != closing {
			continue
		}
		b := e.Buffer()
		if c := b.Cursor(); !b.HasSelection() && c < b.Len() && b.Slice(c, c+1) == string(r) {
			e.Navigate(func(b *buffer.Buffer) { b.SetCursor(c+1, false) })
			return true
		}
	}
	return false
}

// completeLine handles Enter: format the current line if the cursor is at
// its end, then break the line. It reports false
// to leave Enter to the editor (auto-indent off, or a selection).
func completeLine(w *MainWindow, e *editor.ScriptEditor) bool {
	if w != nil && w.settings != nil && !w.settings.GetBoolean("auto-indent") {
		return false
	}
	if w != nil && w.markdown { // no screenplay elements: lists go on
		return continueList(e)
	}
	if e.SelectedText() != "" {
		return false
	}
	if w.flowEnter() { // the Final Draft flow (keyflow.go)
		return true
	}

	text := []rune(e.Text())
	cursor := e.CursorOffset()
	start, end := lineBounds(text, cursor)
	line := string(text[start:end])

	formatted, kind := line, ""
	if cursor == end {
		prev := ""
		if start > 0 {
			ps, _ := lineBounds(text, start-1)
			prev = string(text[ps : start-1])
		}
		formatted, kind = formatCompletedLine(prev, line, w.sceneStarts())
	}
	// after a scene heading, the blank line Fountain wants and the action
	breaks := "\n"
	if kind == "Scene Heading" && w.flowOn() {
		breaks = "\n\n"
	}

	e.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			if formatted != line {
				b.Replace(start, end, formatted)
			}
			b.Insert(breaks)
		})
	})
	if len(breaks) == 2 {
		w.flowHeadingDone()
	}
	return true
}

// lineBounds returns the rune range [start, end) of the line containing pos.
func lineBounds(text []rune, pos int) (start, end int) {
	start, end = pos, pos
	for start > 0 && text[start-1] != '\n' {
		start--
	}
	for end < len(text) && text[end] != '\n' {
		end++
	}
	return start, end
}

// formatCompletedLine returns how a just-completed line should look, given
// the line before it, and which element it is. Lines that are not clearly
// a screenplay element are returned unchanged.
func formatCompletedLine(prev, line string, starts []string) (string, string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return line, "Empty"
	}
	upper := strings.ToUpper(trimmed)
	afterBlank := strings.TrimSpace(prev) == ""

	switch {
	case isSceneHeadingIn(upper, starts) && afterBlank:
		return upper, "Scene Heading"
	case isTransition(upper) && afterBlank:
		// Only "...TO:" is a transition by itself in Fountain; others such
		// as FADE OUT. need the ">" marker for other Fountain apps (and
		// the PDF export) to read them as one.
		if !strings.HasSuffix(upper, "TO:") && !strings.HasPrefix(upper, ">") {
			upper = "> " + upper
		}
		return upper, "Transition"
	case strings.HasPrefix(trimmed, "(") && strings.HasSuffix(trimmed, ")") && !afterBlank:
		return trimmed, "Parenthetical"
	case isCharacterCue(trimmed) && afterBlank:
		return trimmed, "Character"
	}
	return line, "Action"
}

// isSceneHeading reports whether a line starts like an English scene
// heading.
func isSceneHeading(line string) bool { return isSceneHeadingIn(line, nil) }

// isTransition follows Fountain: an uppercase line ending in TO:, or one
// forced with ">", plus the usual endings like FADE OUT. that a writer types
// without the marker. FADE IN: is deliberately not one.
func isTransition(line string) bool {
	upper := strings.ToUpper(strings.TrimSpace(line))
	if strings.HasSuffix(upper, " TO:") || upper == "CUT TO:" {
		return true
	}
	if strings.HasPrefix(upper, ">") && !strings.HasSuffix(upper, "<") {
		return true
	}
	switch upper {
	case "FADE OUT.", "FADE OUT:", "FADE TO BLACK.", "THE END":
		return true
	}
	return false
}

// isCharacterCue reports whether a trimmed line reads as a character name:
// entirely uppercase with at least one letter, optionally followed by an
// extension like (V.O.) or (CONT'D).
func isCharacterCue(trimmed string) bool {
	name := trimmed
	if i := strings.Index(trimmed, "("); i > 0 && strings.HasSuffix(trimmed, ")") {
		name = strings.TrimSpace(trimmed[:i])
	}
	if len(name) < 2 || strings.HasSuffix(name, ":") || strings.ContainsAny(name, ".!?") {
		return false
	}
	// capitals only, and at least one (Hebrew, Arabic: no cue without @)
	return name == strings.ToUpper(name) && strings.IndexFunc(name, unicode.IsUpper) >= 0
}

func hasLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}
