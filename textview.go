package main

import (
	"log"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"libdb.so/gotk4-sourceview/pkg/gtksource/v5"
	"github.com/diamondburned/gotk4/pkg/pango"
)

type TextView struct {
	*gtk.ScrolledWindow
	
	// Core components
	sourceView   *gtksource.View
	textBuffer   *TextBuffer
	window       *MainWindow
	
	// Fountain-specific features
	formatter    *FountainFormatter
	spellChecker *SpellChecker
	
	// Settings
	settings     *Settings
	
	// State
	focusMode    bool
	hemingwayMode bool
}

func NewTextView(buffer *TextBuffer) *TextView {
	tv := &TextView{
		ScrolledWindow: gtk.NewScrolledWindow(),
		textBuffer:     buffer,
		focusMode:      false,
		hemingwayMode:  false,
	}
	
	tv.setupUI()
	tv.setupSourceView()
	tv.setupFormatting()
	tv.setupSpellCheck()
	tv.setupSignals()
	
	return tv
}

func (tv *TextView) setupUI() {
	// Configure scrolled window
	tv.SetHExpand(true)
	tv.SetVExpand(true)
	tv.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	tv.SetHasFrame(false)
	
	// Add CSS classes
	tv.AddCSSClass("editor")
	tv.AddCSSClass("view")
}

func (tv *TextView) setupSourceView() {
	// Create source view
	tv.sourceView = gtksource.NewViewWithBuffer(tv.textBuffer.Buffer)
	
	// Basic settings
	tv.sourceView.SetShowLineNumbers(false)
	tv.sourceView.SetShowRightMargin(false)
	tv.sourceView.SetHighlightCurrentLine(false)
	tv.sourceView.SetWrapMode(gtk.WrapWord)
	tv.sourceView.SetLeftMargin(20)
	tv.sourceView.SetRightMargin(20)
	tv.sourceView.SetTopMargin(20)
	tv.sourceView.SetBottomMargin(20)
	tv.sourceView.SetIndentOnTab(true)
	tv.sourceView.SetTabWidth(4)
	tv.sourceView.SetInsertSpacesInsteadOfTabs(true)
	tv.sourceView.SetAutoIndent(true)
	tv.sourceView.SetSmartBackspace(true)
	tv.sourceView.SetSmartHomeEnd(gtksource.SmartHomeEndAfter)
	
	// Set up font
	tv.setupFont()
	
	// Set up syntax highlighting for Fountain
	tv.setupSyntaxHighlighting()
	
	// Add to scrolled window
	tv.SetChild(tv.sourceView)
}

func (tv *TextView) setupFont() {
	// Use monospace font for screenplay formatting
	fontDesc := pango.NewFontDescription()
	fontDesc.SetFamily("monospace")
	fontDesc.SetSize(11 * pango.SCALE)
	tv.sourceView.OverrideFont(fontDesc)
}

func (tv *TextView) setupSyntaxHighlighting() {
	// Get language manager
	langManager := gtksource.LanguageManagerGetDefault()
	
	// Try to get Fountain language definition
	// If not available, we'll use plain text
	lang := langManager.GetLanguage("fountain")
	if lang == nil {
		// Create a basic language definition for Fountain
		lang = tv.createFountainLanguage()
	}
	
	if lang != nil {
		tv.textBuffer.SetLanguage(lang)
	}
	
	// Set up style scheme
	tv.setupStyleScheme()
}

func (tv *TextView) createFountainLanguage() *gtksource.Language {
	// TODO: Create a proper Fountain language definition
	// For now, return nil and use plain text
	return nil
}

func (tv *TextView) setupStyleScheme() {
	schemeManager := gtksource.StyleSchemeManagerGetDefault()
	
	// Default to a readable scheme
	scheme := schemeManager.GetScheme("Adwaita")
	if scheme == nil {
		scheme = schemeManager.GetScheme("classic")
	}
	
	if scheme != nil {
		tv.textBuffer.SetStyleScheme(scheme)
	}
}

func (tv *TextView) setupFormatting() {
	tv.formatter = NewFountainFormatter(tv)
}

func (tv *TextView) setupSpellCheck() {
	tv.spellChecker = NewSpellChecker(tv)
}

func (tv *TextView) setupSignals() {
	// Connect key press events for Fountain-specific behavior
	keyController := gtk.NewEventControllerKey()
	keyController.ConnectKeyPressed(func(keyval uint, keycode uint, state gdk.ModifierType) bool {
		return tv.onKeyPressed(keyval, keycode, state)
	})
	tv.sourceView.AddController(keyController)
	
	// Connect cursor position changes
	tv.textBuffer.ConnectCursorMoved(func() {
		tv.onCursorMoved()
	})
	
	// Connect text changes
	tv.textBuffer.ConnectChanged(func() {
		tv.onTextChanged()
	})
}

func (tv *TextView) onKeyPressed(keyval uint, keycode uint, state gdk.ModifierType) bool {
	// Handle Fountain-specific key bindings
	switch keyval {
	case gdk.KEY_Return:
		return tv.handleEnterKey(state)
	case gdk.KEY_Tab:
		return tv.handleTabKey(state)
	case gdk.KEY_colon:
		return tv.handleColonKey(state)
	case gdk.KEY_parenleft:
		return tv.handleParenKey(state)
	case gdk.KEY_period:
		return tv.handlePeriodKey(state)
	}
	
	return false // Let other handlers process
}

func (tv *TextView) handleEnterKey(state gdk.ModifierType) bool {
	// Get current line
	buffer := tv.textBuffer.Buffer
	cursor := buffer.GetInsert()
	iter := buffer.GetIterAtMark(cursor)
	
	lineStart := iter
	lineStart.SetLineOffset(0)
	
	lineEnd := iter
	lineEnd.ForwardToLineEnd()
	
	currentLine := buffer.GetText(&lineStart, &lineEnd, false)
	currentLine = strings.TrimSpace(currentLine)
	
	// Handle different Fountain elements
	if tv.isCharacterLine(currentLine) {
		// After character name, start dialogue
		tv.insertNewlineAndIndent("")
		return true
	} else if tv.isSceneHeading(currentLine) {
		// After scene heading, add blank line
		tv.insertNewlineAndIndent("")
		tv.insertNewlineAndIndent("")
		return true
	} else if tv.isTransition(currentLine) {
		// After transition, add blank line
		tv.insertNewlineAndIndent("")
		tv.insertNewlineAndIndent("")
		return true
	}
	
	return false
}

func (tv *TextView) handleTabKey(state gdk.ModifierType) bool {
	// In Fountain, Tab can be used for formatting
	if state&gdk.ModifierTypeShiftMask != 0 {
		// Shift+Tab - move to previous element type
		return tv.cyclePreviousElementType()
	} else {
		// Tab - move to next element type
		return tv.cycleNextElementType()
	}
}

func (tv *TextView) handleColonKey(state gdk.ModifierType) bool {
	// Auto-format transitions ending with ":"
	return tv.checkForTransition()
}

func (tv *TextView) handleParenKey(state gdk.ModifierType) bool {
	// Auto-format parentheticals
	return tv.formatParenthetical()
}

func (tv *TextView) handlePeriodKey(state gdk.ModifierType) bool {
	// Check for forced scene headings starting with "."
	return tv.checkForForcedSceneHeading()
}

func (tv *TextView) insertNewlineAndIndent(indent string) {
	buffer := tv.textBuffer.Buffer
	cursor := buffer.GetInsert()
	iter := buffer.GetIterAtMark(cursor)
	
	buffer.Insert(&iter, "\n"+indent)
}

func (tv *TextView) isCharacterLine(line string) bool {
	// Character names are typically ALL CAPS
	if len(line) == 0 {
		return false
	}
	
	// Remove any character extensions like (V.O.) or (O.S.)
	if strings.Contains(line, "(") {
		parts := strings.Split(line, "(")
		line = strings.TrimSpace(parts[0])
	}
	
	// Check if it's all uppercase letters, spaces, periods, and apostrophes
	for _, r := range line {
		if !((r >= 'A' && r <= 'Z') || r == ' ' || r == '.' || r == '\'' || r == '_' || r == '-') {
			return false
		}
	}
	
	return len(line) > 0
}

func (tv *TextView) isSceneHeading(line string) bool {
	line = strings.ToUpper(strings.TrimSpace(line))
	
	// Standard scene headings
	prefixes := []string{"INT.", "EXT.", "EST.", "INT ", "EXT ", "EST "}
	for _, prefix := range prefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	
	// Forced scene headings start with "."
	if strings.HasPrefix(line, ".") {
		return true
	}
	
	return false
}

func (tv *TextView) isTransition(line string) bool {
	line = strings.ToUpper(strings.TrimSpace(line))
	
	// Common transitions
	transitions := []string{
		"FADE IN:",
		"FADE OUT.",
		"CUT TO:",
		"DISSOLVE TO:",
		"SMASH CUT TO:",
		"MATCH CUT TO:",
		"JUMP CUT TO:",
		"IRIS IN:",
		"IRIS OUT:",
	}
	
	for _, transition := range transitions {
		if line == transition {
			return true
		}
	}
	
	// General pattern: ends with "TO:"
	if strings.HasSuffix(line, "TO:") {
		return true
	}
	
	return false
}

func (tv *TextView) cycleNextElementType() bool {
	// TODO: Implement element type cycling
	log.Println("Cycling to next element type")
	return false
}

func (tv *TextView) cyclePreviousElementType() bool {
	// TODO: Implement element type cycling
	log.Println("Cycling to previous element type")
	return false
}

func (tv *TextView) checkForTransition() bool {
	// TODO: Implement transition auto-formatting
	return false
}

func (tv *TextView) formatParenthetical() bool {
	// TODO: Implement parenthetical formatting
	return false
}

func (tv *TextView) checkForForcedSceneHeading() bool {
	// TODO: Implement forced scene heading detection
	return false
}

func (tv *TextView) onCursorMoved() {
	// Update current element type indication
	tv.updateElementType()
}

func (tv *TextView) onTextChanged() {
	// Apply real-time formatting
	tv.applyFormatting()
}

func (tv *TextView) updateElementType() {
	// Get current line and determine element type
	buffer := tv.textBuffer.Buffer
	cursor := buffer.GetInsert()
	iter := buffer.GetIterAtMark(cursor)
	
	lineStart := iter
	lineStart.SetLineOffset(0)
	
	lineEnd := iter
	lineEnd.ForwardToLineEnd()
	
	currentLine := buffer.GetText(&lineStart, &lineEnd, false)
	currentLine = strings.TrimSpace(currentLine)
	
	// Determine element type
	elementType := tv.getElementType(currentLine)
	
	// Update UI to show current element type
	tv.showElementType(elementType)
}

func (tv *TextView) getElementType(line string) string {
	if tv.isSceneHeading(line) {
		return "Scene Heading"
	} else if tv.isCharacterLine(line) {
		return "Character"
	} else if tv.isTransition(line) {
		return "Transition"
	} else if strings.HasPrefix(line, "(") && strings.HasSuffix(line, ")") {
		return "Parenthetical"
	} else if strings.HasPrefix(line, "=") {
		return "Synopsis"
	} else if strings.HasPrefix(line, "[[") && strings.HasSuffix(line, "]]") {
		return "Note"
	} else {
		return "Action"
	}
}

func (tv *TextView) showElementType(elementType string) {
	// TODO: Show element type in status bar or similar
	log.Printf("Current element type: %s", elementType)
}

func (tv *TextView) applyFormatting() {
	if tv.formatter != nil {
		tv.formatter.FormatText()
	}
}

func (tv *TextView) applySettings(settings *Settings) {
	tv.settings = settings
	
	// Apply font settings
	if settings.GetBoolean("bigger-text") {
		tv.setBiggerText(true)
	}
	
	// Apply spell check settings
	if settings.GetBoolean("spellcheck") {
		tv.enableSpellCheck()
	} else {
		tv.disableSpellCheck()
	}
	
	// Apply line length settings
	lineLength := settings.GetInt("characters-per-line")
	tv.setLineLength(lineLength)
	
	// Apply focus mode
	if settings.GetBoolean("focus-mode") {
		tv.enableFocusMode()
	}
	
	// Apply hemingway mode
	if settings.GetBoolean("hemingway-mode") {
		tv.enableHemingwayMode()
	}
}

func (tv *TextView) setBiggerText(bigger bool) {
	fontDesc := pango.NewFontDescription()
	fontDesc.SetFamily("monospace")
	
	if bigger {
		fontDesc.SetSize(13 * pango.SCALE)
	} else {
		fontDesc.SetSize(11 * pango.SCALE)
	}
	
	tv.sourceView.OverrideFont(fontDesc)
}

func (tv *TextView) enableSpellCheck() {
	if tv.spellChecker != nil {
		tv.spellChecker.Enable()
	}
}

func (tv *TextView) disableSpellCheck() {
	if tv.spellChecker != nil {
		tv.spellChecker.Disable()
	}
}

func (tv *TextView) setLineLength(length int) {
	tv.sourceView.SetRightMarginPosition(uint(length))
	tv.sourceView.SetShowRightMargin(true)
}

func (tv *TextView) enableFocusMode() {
	tv.focusMode = true
	tv.AddCSSClass("focus-mode")
}

func (tv *TextView) disableFocusMode() {
	tv.focusMode = false
	tv.RemoveCSSClass("focus-mode")
}

func (tv *TextView) enableHemingwayMode() {
	tv.hemingwayMode = true
	tv.sourceView.SetEditable(false)
	tv.AddCSSClass("hemingway-mode")
}

func (tv *TextView) disableHemingwayMode() {
	tv.hemingwayMode = false
	tv.sourceView.SetEditable(true)
	tv.RemoveCSSClass("hemingway-mode")
}

func (tv *TextView) GetText() string {
	return tv.textBuffer.GetText()
}

func (tv *TextView) SetText(text string) {
	tv.textBuffer.SetText(text)
}

func (tv *TextView) GrabFocus() {
	tv.sourceView.GrabFocus()
}

func (tv *TextView) GetBuffer() *TextBuffer {
	return tv.textBuffer
}

func (tv *TextView) GetSourceView() *gtksource.View {
	return tv.sourceView
}

// Insert text at current cursor position
func (tv *TextView) InsertAtCursor(text string) {
	buffer := tv.textBuffer.Buffer
	buffer.InsertAtCursor(text)
}

// Get current cursor position
func (tv *TextView) GetCursorPosition() int {
	buffer := tv.textBuffer.Buffer
	cursor := buffer.GetInsert()
	iter := buffer.GetIterAtMark(cursor)
	return iter.GetOffset()
}

// Set cursor position
func (tv *TextView) SetCursorPosition(pos int) {
	buffer := tv.textBuffer.Buffer
	iter := buffer.GetIterAtOffset(pos)
	buffer.PlaceCursor(&iter)
}

// Get selected text
func (tv *TextView) GetSelectedText() string {
	buffer := tv.textBuffer.Buffer
	
	var start, end gtk.TextIter
	if buffer.GetSelectionBounds(&start, &end) {
		return buffer.GetText(&start, &end, false)
	}
	
	return ""
}

// Replace selected text
func (tv *TextView) ReplaceSelectedText(text string) {
	buffer := tv.textBuffer.Buffer
	
	var start, end gtk.TextIter
	if buffer.GetSelectionBounds(&start, &end) {
		buffer.Delete(&start, &end)
		buffer.Insert(&start, text)
	}
}

// Find text in buffer
func (tv *TextView) FindText(text string, caseSensitive bool) bool {
	// TODO: Implement text search
	return false
}

// Replace text in buffer
func (tv *TextView) ReplaceText(findText, replaceText string, caseSensitive bool) bool {
	// TODO: Implement text replacement
	return false
}