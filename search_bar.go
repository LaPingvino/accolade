package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type SearchBar struct {
	window    *MainWindow
	container *fyne.Container
	
	// Search widgets
	searchEntry   *widget.Entry
	replaceEntry  *widget.Entry
	
	// Control buttons
	findNextButton     *widget.Button
	findPrevButton     *widget.Button
	replaceButton      *widget.Button
	replaceAllButton   *widget.Button
	closeButton        *widget.Button
	
	// Options
	caseSensitiveCheck *widget.Check
	wholeWordCheck     *widget.Check
	regexCheck         *widget.Check
	
	// Status
	statusLabel *widget.Label
	
	// State
	isVisible     bool
	replaceMode   bool
	currentMatch  int
	totalMatches  int
	searchText    string
}

func NewSearchBar(window *MainWindow) *SearchBar {
	sb := &SearchBar{
		window:        window,
		isVisible:     false,
		replaceMode:   false,
		currentMatch:  0,
		totalMatches:  0,
	}
	
	sb.createWidgets()
	sb.createLayout()
	sb.setupCallbacks()
	
	return sb
}

func (sb *SearchBar) createWidgets() {
	// Search entry
	sb.searchEntry = widget.NewEntry()
	sb.searchEntry.SetPlaceHolder("Find...")
	
	// Replace entry
	sb.replaceEntry = widget.NewEntry()
	sb.replaceEntry.SetPlaceHolder("Replace with...")
	
	// Control buttons
	sb.findNextButton = widget.NewButtonWithIcon("Next", theme.NavigateNextIcon(), func() {
		sb.findNext()
	})
	
	sb.findPrevButton = widget.NewButtonWithIcon("Previous", theme.NavigateBackIcon(), func() {
		sb.findPrevious()
	})
	
	sb.replaceButton = widget.NewButtonWithIcon("Replace", theme.DocumentSaveIcon(), func() {
		sb.replaceOne()
	})
	
	sb.replaceAllButton = widget.NewButtonWithIcon("Replace All", theme.ContentCopyIcon(), func() {
		sb.replaceAll()
	})
	
	sb.closeButton = widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
		sb.Hide()
	})
	
	// Options
	sb.caseSensitiveCheck = widget.NewCheck("Case sensitive", func(checked bool) {
		sb.updateSearch()
	})
	
	sb.wholeWordCheck = widget.NewCheck("Whole words", func(checked bool) {
		sb.updateSearch()
	})
	
	sb.regexCheck = widget.NewCheck("Regular expression", func(checked bool) {
		sb.updateSearch()
	})
	
	// Status
	sb.statusLabel = widget.NewLabel("No matches")
}

func (sb *SearchBar) createLayout() {
	// Search row
	searchRow := container.NewHBox(
		widget.NewLabel("Find:"),
		sb.searchEntry,
		sb.findPrevButton,
		sb.findNextButton,
		sb.closeButton,
	)
	
	// Replace row (initially hidden)
	replaceRow := container.NewHBox(
		widget.NewLabel("Replace:"),
		sb.replaceEntry,
		sb.replaceButton,
		sb.replaceAllButton,
	)
	
	// Options row
	optionsRow := container.NewHBox(
		sb.caseSensitiveCheck,
		sb.wholeWordCheck,
		sb.regexCheck,
		widget.NewLabel(""),  // Spacer
		sb.statusLabel,
	)
	
	// Main container
	sb.container = container.NewVBox(
		searchRow,
		replaceRow,
		optionsRow,
	)
	
	// Initially hide replace row
	replaceRow.Hide()
	
	// Initially hide the whole container
	sb.container.Hide()
}

func (sb *SearchBar) setupCallbacks() {
	// Search as you type
	sb.searchEntry.OnChanged = func(text string) {
		sb.searchText = text
		sb.updateSearch()
	}
	
	// Handle Enter key in search entry
	sb.searchEntry.OnSubmitted = func(text string) {
		sb.findNext()
	}
	
	// Handle Enter key in replace entry
	sb.replaceEntry.OnSubmitted = func(text string) {
		sb.replaceOne()
	}
}

func (sb *SearchBar) Show() {
	sb.container.Show()
	sb.isVisible = true
	sb.window.fyneWindow.Canvas().Focus(sb.searchEntry)
}

func (sb *SearchBar) Hide() {
	sb.container.Hide()
	sb.isVisible = false
	sb.window.fyneWindow.Canvas().Focus(sb.window.textEditor)
}

func (sb *SearchBar) SetReplaceMode(enabled bool) {
	sb.replaceMode = enabled
	
	// Get replace row (second child)
	if len(sb.container.Objects) >= 2 {
		replaceRow := sb.container.Objects[1]
		if enabled {
			replaceRow.Show()
		} else {
			replaceRow.Hide()
		}
	}
	
	sb.container.Refresh()
}

func (sb *SearchBar) IsVisible() bool {
	return sb.isVisible
}

func (sb *SearchBar) updateSearch() {
	if sb.searchText == "" {
		sb.totalMatches = 0
		sb.currentMatch = 0
		sb.updateStatus(0, 0)
		return
	}

	matches, err := sb.findAllMatches()
	if err != nil {
		sb.totalMatches = 0
		sb.currentMatch = 0
		sb.statusLabel.SetText("Invalid regular expression")
		return
	}
	sb.totalMatches = len(matches)
	sb.currentMatch = 0
	sb.updateStatus(sb.currentMatch, sb.totalMatches)
}

// findNext selects the first match starting at or after the cursor,
// wrapping around to the top of the document.
func (sb *SearchBar) findNext() {
	matches, _ := sb.findAllMatches()
	if len(matches) == 0 {
		sb.updateStatus(0, 0)
		return
	}

	cursor := sb.window.textEditor.CursorTextOffset()
	next := 0
	for i, m := range matches {
		if m.start >= cursor {
			next = i
			break
		}
	}
	sb.gotoMatch(matches, next)
}

// findPrevious selects the last match that ends before the current
// selection (or cursor), wrapping around to the bottom of the document.
func (sb *SearchBar) findPrevious() {
	matches, _ := sb.findAllMatches()
	if len(matches) == 0 {
		sb.updateStatus(0, 0)
		return
	}

	anchor := sb.selectionStart()
	prev := len(matches) - 1
	for i := len(matches) - 1; i >= 0; i-- {
		if matches[i].start < anchor {
			prev = i
			break
		}
	}
	sb.gotoMatch(matches, prev)
}

func (sb *SearchBar) gotoMatch(matches []searchMatch, i int) {
	sb.currentMatch = i + 1
	sb.totalMatches = len(matches)
	selectRange(sb.window.textEditor, matches[i].start, matches[i].end)
	sb.updateStatus(sb.currentMatch, sb.totalMatches)
}

// selectionStart is the rune offset where the current selection begins,
// or the cursor offset when nothing is selected. Selections made by
// selectRange always leave the cursor at their end.
func (sb *SearchBar) selectionStart() int {
	editor := sb.window.textEditor
	return editor.CursorTextOffset() - utf8.RuneCountInString(editor.SelectedText())
}

// replaceOne replaces the selected match (if the selection is a match)
// and moves on to the next one. Typing over the selection keeps the
// editor's undo history intact.
func (sb *SearchBar) replaceOne() {
	if sb.searchText == "" {
		return
	}

	matches, err := sb.findAllMatches()
	if err != nil {
		return
	}
	editor := sb.window.textEditor
	selected := editor.SelectedText()
	start := sb.selectionStart()
	for _, m := range matches {
		if selected != "" && m.start == start && m.end == start+utf8.RuneCountInString(selected) {
			typeOverSelection(editor, sb.expandReplacement(selected))
			sb.markChanged()
			break
		}
	}

	sb.findNext()
}

func (sb *SearchBar) replaceAll() {
	if sb.searchText == "" {
		return
	}

	text := sb.window.textEditor.Text
	newText, count, err := sb.replaceAllIn(text)
	if err != nil || count == 0 {
		sb.updateSearch()
		return
	}

	sb.window.textEditor.SetText(newText)
	sb.markChanged()
	sb.updateSearch()
	sb.statusLabel.SetText(fmt.Sprintf("Replaced %d", count))
}

// replaceAllIn returns text with every match replaced and the number of
// replacements made.
func (sb *SearchBar) replaceAllIn(text string) (string, int, error) {
	re, err := sb.pattern()
	if err != nil {
		return text, 0, err
	}

	count := 0
	var out strings.Builder
	last := 0
	for _, loc := range re.FindAllStringSubmatchIndex(text, -1) {
		if loc[0] == loc[1] || (sb.wholeWordCheck.Checked && !isWholeWord(text, loc[0], loc[1])) {
			continue
		}
		out.WriteString(text[last:loc[0]])
		if sb.regexCheck.Checked {
			out.Write(re.ExpandString(nil, sb.replaceEntry.Text, text, loc))
		} else {
			out.WriteString(sb.replaceEntry.Text)
		}
		last = loc[1]
		count++
	}
	out.WriteString(text[last:])
	return out.String(), count, nil
}

// expandReplacement resolves $1-style group references when in regex mode.
func (sb *SearchBar) expandReplacement(matched string) string {
	if !sb.regexCheck.Checked {
		return sb.replaceEntry.Text
	}
	re, err := sb.pattern()
	if err != nil {
		return sb.replaceEntry.Text
	}
	loc := re.FindStringSubmatchIndex(matched)
	if loc == nil {
		return sb.replaceEntry.Text
	}
	return string(re.ExpandString(nil, sb.replaceEntry.Text, matched, loc))
}

func (sb *SearchBar) markChanged() {
	sb.window.hasChanges = true
}

type searchMatch struct {
	start int // rune offset
	end   int // rune offset, exclusive
}

// pattern compiles the search text according to the option checkboxes.
func (sb *SearchBar) pattern() (*regexp.Regexp, error) {
	expr := sb.searchText
	if !sb.regexCheck.Checked {
		expr = regexp.QuoteMeta(expr)
	}
	if !sb.caseSensitiveCheck.Checked {
		expr = "(?i)" + expr
	}
	return regexp.Compile(expr)
}

// findAllMatches returns all non-empty matches in the editor as rune offsets,
// which is what the Fyne Entry cursor API works in.
func (sb *SearchBar) findAllMatches() ([]searchMatch, error) {
	if sb.searchText == "" {
		return nil, nil
	}
	re, err := sb.pattern()
	if err != nil {
		return nil, err
	}

	text := sb.window.textEditor.Text
	var matches []searchMatch
	runePos, bytePos := 0, 0
	for _, loc := range re.FindAllStringIndex(text, -1) {
		if loc[0] == loc[1] || (sb.wholeWordCheck.Checked && !isWholeWord(text, loc[0], loc[1])) {
			continue
		}
		runePos += utf8.RuneCountInString(text[bytePos:loc[0]])
		start := runePos
		runePos += utf8.RuneCountInString(text[loc[0]:loc[1]])
		bytePos = loc[1]
		matches = append(matches, searchMatch{start: start, end: runePos})
	}
	return matches, nil
}

// isWholeWord reports whether text[start:end] (byte offsets) is not
// directly adjoined by letters, digits or underscores.
func isWholeWord(text string, start, end int) bool {
	if start > 0 {
		r, _ := utf8.DecodeLastRuneInString(text[:start])
		if isWordRune(r) {
			return false
		}
	}
	if end < len(text) {
		r, _ := utf8.DecodeRuneInString(text[end:])
		if isWordRune(r) {
			return false
		}
	}
	return true
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func (sb *SearchBar) updateStatus(current, total int) {
	if total == 0 {
		sb.statusLabel.SetText("No matches")
	} else {
		sb.statusLabel.SetText(fmt.Sprintf("%d of %d", current, total))
	}
}

// Public API methods
func (sb *SearchBar) SetSearchText(text string) {
	sb.searchEntry.SetText(text)
	sb.searchText = text
	sb.updateSearch()
}

func (sb *SearchBar) GetSearchText() string {
	return sb.searchText
}

func (sb *SearchBar) SetReplaceText(text string) {
	sb.replaceEntry.SetText(text)
}

func (sb *SearchBar) GetReplaceText() string {
	return sb.replaceEntry.Text
}

func (sb *SearchBar) SetCaseSensitive(sensitive bool) {
	sb.caseSensitiveCheck.SetChecked(sensitive)
}

func (sb *SearchBar) SetWholeWords(wholeWords bool) {
	sb.wholeWordCheck.SetChecked(wholeWords)
}

func (sb *SearchBar) SetRegularExpression(regex bool) {
	sb.regexCheck.SetChecked(regex)
}