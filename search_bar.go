package main

import (
	"log"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type SearchBar struct {
	*gtk.SearchBar
	
	// Core components
	window       *MainWindow
	searchEntry  *gtk.SearchEntry
	replaceEntry *gtk.Entry
	replaceBox   *gtk.Box
	
	// Search controls
	prevButton   *gtk.Button
	nextButton   *gtk.Button
	replaceButton *gtk.Button
	replaceAllButton *gtk.Button
	closeButton  *gtk.Button
	
	// Search state
	searchMode   bool
	replaceMode  bool
	caseSensitive bool
	wholeWords   bool
	useRegex     bool
	
	// Search results
	currentMatch int
	totalMatches int
	searchText   string
	replaceText  string
}

func NewSearchBar(window *MainWindow) *SearchBar {
	sb := &SearchBar{
		SearchBar:     gtk.NewSearchBar(),
		window:        window,
		searchMode:    false,
		replaceMode:   false,
		caseSensitive: false,
		wholeWords:    false,
		useRegex:      false,
		currentMatch:  0,
		totalMatches:  0,
	}
	
	sb.setupUI()
	sb.setupSignals()
	
	return sb
}

func (sb *SearchBar) setupUI() {
	// Main container
	mainBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	mainBox.SetMarginStart(6)
	mainBox.SetMarginEnd(6)
	mainBox.SetMarginTop(6)
	mainBox.SetMarginBottom(6)
	
	// Search entry
	sb.searchEntry = gtk.NewSearchEntry()
	sb.searchEntry.SetPlaceholderText("Search...")
	sb.searchEntry.SetHExpand(true)
	mainBox.Append(sb.searchEntry)
	
	// Previous button
	sb.prevButton = gtk.NewButtonFromIconName("go-up-symbolic")
	sb.prevButton.SetTooltipText("Previous match")
	sb.prevButton.SetSensitive(false)
	mainBox.Append(sb.prevButton)
	
	// Next button
	sb.nextButton = gtk.NewButtonFromIconName("go-down-symbolic")
	sb.nextButton.SetTooltipText("Next match")
	sb.nextButton.SetSensitive(false)
	mainBox.Append(sb.nextButton)
	
	// Options button
	optionsButton := gtk.NewButtonFromIconName("preferences-system-symbolic")
	optionsButton.SetTooltipText("Search options")
	mainBox.Append(optionsButton)
	
	// Replace box (initially hidden)
	sb.replaceBox = gtk.NewBox(gtk.OrientationHorizontal, 6)
	sb.replaceBox.SetVisible(false)
	
	// Replace entry
	sb.replaceEntry = gtk.NewEntry()
	sb.replaceEntry.SetPlaceholderText("Replace with...")
	sb.replaceEntry.SetHExpand(true)
	sb.replaceBox.Append(sb.replaceEntry)
	
	// Replace button
	sb.replaceButton = gtk.NewButtonWithLabel("Replace")
	sb.replaceButton.SetSensitive(false)
	sb.replaceBox.Append(sb.replaceButton)
	
	// Replace all button
	sb.replaceAllButton = gtk.NewButtonWithLabel("Replace All")
	sb.replaceAllButton.SetSensitive(false)
	sb.replaceBox.Append(sb.replaceAllButton)
	
	// Close button
	sb.closeButton = gtk.NewButtonFromIconName("window-close-symbolic")
	sb.closeButton.SetTooltipText("Close search")
	mainBox.Append(sb.closeButton)
	
	// Container for search and replace
	containerBox := gtk.NewBox(gtk.OrientationVertical, 6)
	containerBox.Append(mainBox)
	containerBox.Append(sb.replaceBox)
	
	sb.SetChild(containerBox)
	
	// Set up search bar properties
	sb.SetSearchMode(false)
	sb.SetShowCloseButton(false)
	sb.ConnectEntry(sb.searchEntry)
}

func (sb *SearchBar) setupSignals() {
	// Search entry changed
	sb.searchEntry.ConnectSearchChanged(func() {
		sb.onSearchChanged()
	})
	
	// Search entry activated (Enter pressed)
	sb.searchEntry.ConnectActivate(func() {
		sb.findNext()
	})
	
	// Replace entry activated
	sb.replaceEntry.ConnectActivate(func() {
		sb.replaceNext()
	})
	
	// Button clicks
	sb.prevButton.ConnectClicked(func() {
		sb.findPrevious()
	})
	
	sb.nextButton.ConnectClicked(func() {
		sb.findNext()
	})
	
	sb.replaceButton.ConnectClicked(func() {
		sb.replaceNext()
	})
	
	sb.replaceAllButton.ConnectClicked(func() {
		sb.replaceAll()
	})
	
	sb.closeButton.ConnectClicked(func() {
		sb.closeSearch()
	})
	
	// Search mode changed
	sb.ConnectSearchModeChanged(func() {
		sb.onSearchModeChanged()
	})
}

func (sb *SearchBar) onSearchChanged() {
	sb.searchText = sb.searchEntry.Text()
	log.Printf("Search text changed: %s", sb.searchText)
	
	if sb.searchText == "" {
		sb.clearHighlights()
		sb.updateMatchCount(0, 0)
		return
	}
	
	// Perform search
	sb.performSearch()
}

func (sb *SearchBar) onSearchModeChanged() {
	if sb.SearchMode() {
		sb.searchEntry.GrabFocus()
	} else {
		sb.clearHighlights()
		sb.window.textView.GrabFocus()
	}
}

func (sb *SearchBar) performSearch() {
	if sb.searchText == "" {
		return
	}
	
	// TODO: Implement actual search in text buffer
	log.Printf("Performing search for: %s", sb.searchText)
	
	// Mock search results for now
	sb.totalMatches = 3
	sb.currentMatch = 1
	sb.updateMatchCount(sb.currentMatch, sb.totalMatches)
	
	// Enable/disable navigation buttons
	sb.prevButton.SetSensitive(sb.totalMatches > 0)
	sb.nextButton.SetSensitive(sb.totalMatches > 0)
	sb.replaceButton.SetSensitive(sb.totalMatches > 0 && sb.replaceMode)
	sb.replaceAllButton.SetSensitive(sb.totalMatches > 0 && sb.replaceMode)
}

func (sb *SearchBar) updateMatchCount(current, total int) {
	sb.currentMatch = current
	sb.totalMatches = total
	
	if total == 0 {
		sb.searchEntry.AddCSSClass("error")
	} else {
		sb.searchEntry.RemoveCSSClass("error")
	}
	
	// TODO: Update match count display
	log.Printf("Match count: %d/%d", current, total)
}

func (sb *SearchBar) findNext() {
	if sb.totalMatches == 0 {
		return
	}
	
	sb.currentMatch++
	if sb.currentMatch > sb.totalMatches {
		sb.currentMatch = 1
	}
	
	sb.updateMatchCount(sb.currentMatch, sb.totalMatches)
	sb.jumpToMatch(sb.currentMatch)
}

func (sb *SearchBar) findPrevious() {
	if sb.totalMatches == 0 {
		return
	}
	
	sb.currentMatch--
	if sb.currentMatch < 1 {
		sb.currentMatch = sb.totalMatches
	}
	
	sb.updateMatchCount(sb.currentMatch, sb.totalMatches)
	sb.jumpToMatch(sb.currentMatch)
}

func (sb *SearchBar) jumpToMatch(matchNum int) {
	// TODO: Implement jumping to specific match in text buffer
	log.Printf("Jumping to match %d", matchNum)
}

func (sb *SearchBar) replaceNext() {
	if sb.currentMatch == 0 {
		return
	}
	
	sb.replaceText = sb.replaceEntry.Text()
	log.Printf("Replacing match %d with: %s", sb.currentMatch, sb.replaceText)
	
	// TODO: Implement actual text replacement
	
	// Update search results
	sb.performSearch()
}

func (sb *SearchBar) replaceAll() {
	if sb.totalMatches == 0 {
		return
	}
	
	sb.replaceText = sb.replaceEntry.Text()
	log.Printf("Replacing all %d matches with: %s", sb.totalMatches, sb.replaceText)
	
	// TODO: Implement replace all functionality
	
	// Update search results
	sb.performSearch()
}

func (sb *SearchBar) clearHighlights() {
	// TODO: Clear search highlights in text buffer
	log.Println("Clearing search highlights")
}

func (sb *SearchBar) closeSearch() {
	sb.SetSearchMode(false)
	sb.SetReplaceMode(false)
}

func (sb *SearchBar) SetReplaceMode(replace bool) {
	sb.replaceMode = replace
	sb.replaceBox.SetVisible(replace)
	
	if replace {
		sb.replaceEntry.GrabFocus()
	}
}

func (sb *SearchBar) IsReplaceMode() bool {
	return sb.replaceMode
}

func (sb *SearchBar) SetCaseSensitive(caseSensitive bool) {
	sb.caseSensitive = caseSensitive
	sb.performSearch()
}

func (sb *SearchBar) IsCaseSensitive() bool {
	return sb.caseSensitive
}

func (sb *SearchBar) SetWholeWords(wholeWords bool) {
	sb.wholeWords = wholeWords
	sb.performSearch()
}

func (sb *SearchBar) IsWholeWords() bool {
	return sb.wholeWords
}

func (sb *SearchBar) SetUseRegex(useRegex bool) {
	sb.useRegex = useRegex
	sb.performSearch()
}

func (sb *SearchBar) IsUseRegex() bool {
	return sb.useRegex
}

func (sb *SearchBar) GetSearchText() string {
	return sb.searchText
}

func (sb *SearchBar) SetSearchText(text string) {
	sb.searchEntry.SetText(text)
}

func (sb *SearchBar) GetReplaceText() string {
	return sb.replaceText
}

func (sb *SearchBar) SetReplaceText(text string) {
	sb.replaceEntry.SetText(text)
}

func (sb *SearchBar) GetCurrentMatch() int {
	return sb.currentMatch
}

func (sb *SearchBar) GetTotalMatches() int {
	return sb.totalMatches
}

func (sb *SearchBar) HasMatches() bool {
	return sb.totalMatches > 0
}

func (sb *SearchBar) FindInText(text string, searchFor string) []SearchMatch {
	// TODO: Implement actual text search with various options
	matches := []SearchMatch{}
	
	if !sb.caseSensitive {
		text = strings.ToLower(text)
		searchFor = strings.ToLower(searchFor)
	}
	
	if sb.wholeWords {
		// TODO: Implement whole word search
	}
	
	if sb.useRegex {
		// TODO: Implement regex search
	} else {
		// Simple substring search
		start := 0
		for {
			pos := strings.Index(text[start:], searchFor)
			if pos == -1 {
				break
			}
			
			match := SearchMatch{
				Start:  start + pos,
				End:    start + pos + len(searchFor),
				Length: len(searchFor),
				Text:   searchFor,
			}
			matches = append(matches, match)
			
			start += pos + 1
		}
	}
	
	return matches
}

type SearchMatch struct {
	Start  int
	End    int
	Length int
	Text   string
}