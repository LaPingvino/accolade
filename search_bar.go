package main

import (
	"fmt"
	"strings"

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
	lastSearchPos int
}

func NewSearchBar(window *MainWindow) *SearchBar {
	sb := &SearchBar{
		window:        window,
		isVisible:     false,
		replaceMode:   false,
		currentMatch:  0,
		totalMatches:  0,
		lastSearchPos: 0,
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
	sb.searchEntry.FocusGained()
}

func (sb *SearchBar) Hide() {
	sb.container.Hide()
	sb.isVisible = false
	sb.clearHighlights()
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
		sb.clearHighlights()
		sb.updateStatus(0, 0)
		return
	}
	
	matches := sb.findAllMatches()
	sb.totalMatches = len(matches)
	
	if sb.totalMatches > 0 {
		sb.currentMatch = 1
		sb.highlightMatches(matches)
		sb.scrollToMatch(matches[0])
	} else {
		sb.currentMatch = 0
		sb.clearHighlights()
	}
	
	sb.updateStatus(sb.currentMatch, sb.totalMatches)
}

func (sb *SearchBar) findNext() {
	if sb.searchText == "" {
		return
	}
	
	matches := sb.findAllMatches()
	if len(matches) == 0 {
		return
	}
	
	// Find next match after current cursor position
	cursorPos := sb.getCurrentCursorPosition()
	nextMatch := -1
	
	for i, match := range matches {
		if match.start > cursorPos {
			nextMatch = i
			break
		}
	}
	
	// If no match found after cursor, wrap to beginning
	if nextMatch == -1 {
		nextMatch = 0
	}
	
	sb.currentMatch = nextMatch + 1
	sb.scrollToMatch(matches[nextMatch])
	sb.selectMatch(matches[nextMatch])
	sb.updateStatus(sb.currentMatch, len(matches))
}

func (sb *SearchBar) findPrevious() {
	if sb.searchText == "" {
		return
	}
	
	matches := sb.findAllMatches()
	if len(matches) == 0 {
		return
	}
	
	// Find previous match before current cursor position
	cursorPos := sb.getCurrentCursorPosition()
	prevMatch := -1
	
	for i := len(matches) - 1; i >= 0; i-- {
		if matches[i].start < cursorPos {
			prevMatch = i
			break
		}
	}
	
	// If no match found before cursor, wrap to end
	if prevMatch == -1 {
		prevMatch = len(matches) - 1
	}
	
	sb.currentMatch = prevMatch + 1
	sb.scrollToMatch(matches[prevMatch])
	sb.selectMatch(matches[prevMatch])
	sb.updateStatus(sb.currentMatch, len(matches))
}

func (sb *SearchBar) replaceOne() {
	if sb.searchText == "" || sb.replaceEntry.Text == "" {
		return
	}
	
	// Get current selection or find next match
	selectedText := sb.getSelectedText()
	if selectedText == sb.searchText || (sb.caseSensitiveCheck.Checked && selectedText == sb.searchText) {
		// Replace current selection
		sb.replaceSelectedText(sb.replaceEntry.Text)
	}
	
	// Find next occurrence
	sb.findNext()
}

func (sb *SearchBar) replaceAll() {
	if sb.searchText == "" {
		return
	}
	
	text := sb.window.textEditor.Text
	replacement := sb.replaceEntry.Text
	
	var newText string
	if sb.caseSensitiveCheck.Checked {
		newText = strings.ReplaceAll(text, sb.searchText, replacement)
	} else {
		// Case-insensitive replace
		newText = sb.replaceAllCaseInsensitive(text, sb.searchText, replacement)
	}
	
	sb.window.textEditor.SetText(newText)
	sb.updateSearch() // Refresh search results
}

func (sb *SearchBar) replaceAllCaseInsensitive(text, search, replace string) string {
	if search == "" {
		return text
	}
	
	lowerText := strings.ToLower(text)
	lowerSearch := strings.ToLower(search)
	
	var result strings.Builder
	start := 0
	
	for {
		index := strings.Index(lowerText[start:], lowerSearch)
		if index == -1 {
			result.WriteString(text[start:])
			break
		}
		
		actualIndex := start + index
		result.WriteString(text[start:actualIndex])
		result.WriteString(replace)
		start = actualIndex + len(search)
	}
	
	return result.String()
}

type searchMatch struct {
	start int
	end   int
	text  string
}

func (sb *SearchBar) findAllMatches() []searchMatch {
	if sb.searchText == "" {
		return nil
	}
	
	text := sb.window.textEditor.Text
	searchText := sb.searchText
	
	var matches []searchMatch
	
	if !sb.caseSensitiveCheck.Checked {
		text = strings.ToLower(text)
		searchText = strings.ToLower(searchText)
	}
	
	start := 0
	for {
		index := strings.Index(text[start:], searchText)
		if index == -1 {
			break
		}
		
		actualIndex := start + index
		match := searchMatch{
			start: actualIndex,
			end:   actualIndex + len(sb.searchText),
			text:  sb.searchText,
		}
		
		// Check whole word option
		if sb.wholeWordCheck.Checked {
			if sb.isWholeWord(sb.window.textEditor.Text, match.start, match.end) {
				matches = append(matches, match)
			}
		} else {
			matches = append(matches, match)
		}
		
		start = actualIndex + 1
	}
	
	return matches
}

func (sb *SearchBar) isWholeWord(text string, start, end int) bool {
	// Check if character before start is word boundary
	if start > 0 {
		prevChar := text[start-1]
		if sb.isWordChar(prevChar) {
			return false
		}
	}
	
	// Check if character after end is word boundary
	if end < len(text) {
		nextChar := text[end]
		if sb.isWordChar(nextChar) {
			return false
		}
	}
	
	return true
}

func (sb *SearchBar) isWordChar(char byte) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '_'
}

func (sb *SearchBar) highlightMatches(matches []searchMatch) {
	// TODO: Implement text highlighting in Fyne
	// Fyne's Entry widget has limited text formatting capabilities
	// This would need a custom widget or RichText widget
}

func (sb *SearchBar) clearHighlights() {
	// TODO: Clear text highlights
}

func (sb *SearchBar) scrollToMatch(match searchMatch) {
	// TODO: Scroll to match position
	// This is limited in Fyne's Entry widget
}

func (sb *SearchBar) selectMatch(match searchMatch) {
	// TODO: Select text at match position
	// Limited in Fyne's Entry widget
}

func (sb *SearchBar) getCurrentCursorPosition() int {
	// TODO: Get cursor position from text editor
	// This is not directly available in Fyne's Entry widget
	return sb.lastSearchPos
}

func (sb *SearchBar) getSelectedText() string {
	// TODO: Get currently selected text
	// Limited in Fyne's Entry widget
	return ""
}

func (sb *SearchBar) replaceSelectedText(replacement string) {
	// TODO: Replace currently selected text
	// Limited in Fyne's Entry widget
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