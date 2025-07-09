package main

import (
	"log"
	"strings"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"libdb.so/gotk4-sourceview/pkg/gtksource/v5"
)

type TextBuffer struct {
	*gtksource.Buffer
	
	// Associated view
	view *TextView
	
	// Undo/Redo management
	undoManager *gtksource.UndoManager
	
	// Change tracking
	hasChanges       bool
	lastChangeTime   time.Time
	changeCallbacks  []func()
	
	// Auto-save
	autoSaveTimer    *glib.Source
	autoSaveInterval time.Duration
	
	// Text tags for Fountain elements
	characterTag      *gtk.TextTag
	sceneHeadingTag   *gtk.TextTag
	actionTag         *gtk.TextTag
	dialogueTag       *gtk.TextTag
	parentheticalTag  *gtk.TextTag
	transitionTag     *gtk.TextTag
	noteTag           *gtk.TextTag
	boneyardTag       *gtk.TextTag
	emphasisTag       *gtk.TextTag
	boldTag           *gtk.TextTag
	underlineTag      *gtk.TextTag
	centeredTag       *gtk.TextTag
	pageBreakTag      *gtk.TextTag
	sectionTag        *gtk.TextTag
	synopsisTag       *gtk.TextTag
	
	// Formatting state
	suppressFormatting bool
	formatTimer        *glib.Source
}

func NewTextBuffer() *TextBuffer {
	sourceBuffer := gtksource.NewBuffer(nil)
	
	buffer := &TextBuffer{
		Buffer:           sourceBuffer,
		hasChanges:       false,
		changeCallbacks:  make([]func(), 0),
		autoSaveInterval: 30 * time.Second,
		suppressFormatting: false,
	}
	
	// Set up undo manager
	buffer.undoManager = sourceBuffer.UndoManager()
	buffer.undoManager.SetMaxUndoLevels(100)
	
	// Create text tags
	buffer.createTextTags()
	
	// Connect signals
	buffer.connectSignals()
	
	return buffer
}

func (tb *TextBuffer) createTextTags() {
	tagTable := tb.Buffer.TagTable()
	
	// Character name tag (ALL CAPS, usually centered)
	tb.characterTag = gtk.NewTextTag("character")
	tb.characterTag.SetProperty("weight", 700) // Bold
	tb.characterTag.SetProperty("justification", gtk.JustifyCenter)
	tb.characterTag.SetProperty("pixels-above-lines", 12)
	tb.characterTag.SetProperty("pixels-below-lines", 6)
	tagTable.Add(tb.characterTag)
	
	// Scene heading tag (ALL CAPS, bold)
	tb.sceneHeadingTag = gtk.NewTextTag("scene-heading")
	tb.sceneHeadingTag.SetProperty("weight", 700) // Bold
	tb.sceneHeadingTag.SetProperty("pixels-above-lines", 12)
	tb.sceneHeadingTag.SetProperty("pixels-below-lines", 6)
	tagTable.Add(tb.sceneHeadingTag)
	
	// Action tag (normal text)
	tb.actionTag = gtk.NewTextTag("action")
	tb.actionTag.SetProperty("pixels-above-lines", 0)
	tb.actionTag.SetProperty("pixels-below-lines", 6)
	tagTable.Add(tb.actionTag)
	
	// Dialogue tag (indented)
	tb.dialogueTag = gtk.NewTextTag("dialogue")
	tb.dialogueTag.SetProperty("left-margin", 60)
	tb.dialogueTag.SetProperty("right-margin", 60)
	tb.dialogueTag.SetProperty("pixels-above-lines", 0)
	tb.dialogueTag.SetProperty("pixels-below-lines", 6)
	tagTable.Add(tb.dialogueTag)
	
	// Parenthetical tag (indented, italicized)
	tb.parentheticalTag = gtk.NewTextTag("parenthetical")
	tb.parentheticalTag.SetProperty("style", 2) // Italic
	tb.parentheticalTag.SetProperty("left-margin", 80)
	tb.parentheticalTag.SetProperty("right-margin", 100)
	tb.parentheticalTag.SetProperty("pixels-above-lines", 0)
	tb.parentheticalTag.SetProperty("pixels-below-lines", 6)
	tagTable.Add(tb.parentheticalTag)
	
	// Transition tag (right-aligned, ALL CAPS)
	tb.transitionTag = gtk.NewTextTag("transition")
	tb.transitionTag.SetProperty("weight", 700) // Bold
	tb.transitionTag.SetProperty("justification", gtk.JustifyRight)
	tb.transitionTag.SetProperty("pixels-above-lines", 12)
	tb.transitionTag.SetProperty("pixels-below-lines", 6)
	tagTable.Add(tb.transitionTag)
	
	// Note tag (comments, not printed)
	tb.noteTag = gtk.NewTextTag("note")
	tb.noteTag.SetProperty("foreground", "#666666")
	tb.noteTag.SetProperty("style", 2) // Italic
	tb.noteTag.SetProperty("pixels-above-lines", 0)
	tb.noteTag.SetProperty("pixels-below-lines", 6)
	tagTable.Add(tb.noteTag)
	
	// Boneyard tag (omitted text)
	tb.boneyardTag = gtk.NewTextTag("boneyard")
	tb.boneyardTag.SetProperty("foreground", "#999999")
	tb.boneyardTag.SetProperty("strikethrough", true)
	tb.boneyardTag.SetProperty("pixels-above-lines", 0)
	tb.boneyardTag.SetProperty("pixels-below-lines", 6)
	tagTable.Add(tb.boneyardTag)
	
	// Emphasis tag (italic)
	tb.emphasisTag = gtk.NewTextTag("emphasis")
	tb.emphasisTag.SetProperty("style", 2) // Italic
	tagTable.Add(tb.emphasisTag)
	
	// Bold tag
	tb.boldTag = gtk.NewTextTag("bold")
	tb.boldTag.SetProperty("weight", 700) // Bold
	tagTable.Add(tb.boldTag)
	
	// Underline tag
	tb.underlineTag = gtk.NewTextTag("underline")
	tb.underlineTag.SetProperty("underline", 1) // Single underline
	tagTable.Add(tb.underlineTag)
	
	// Centered tag
	tb.centeredTag = gtk.NewTextTag("centered")
	tb.centeredTag.SetProperty("justification", gtk.JustifyCenter)
	tb.centeredTag.SetProperty("pixels-above-lines", 12)
	tb.centeredTag.SetProperty("pixels-below-lines", 6)
	tagTable.Add(tb.centeredTag)
	
	// Page break tag
	tb.pageBreakTag = gtk.NewTextTag("page-break")
	tb.pageBreakTag.SetProperty("justification", gtk.JustifyCenter)
	tb.pageBreakTag.SetProperty("weight", 700) // Bold
	tb.pageBreakTag.SetProperty("pixels-above-lines", 24)
	tb.pageBreakTag.SetProperty("pixels-below-lines", 24)
	tagTable.Add(tb.pageBreakTag)
	
	// Section tag (outline)
	tb.sectionTag = gtk.NewTextTag("section")
	tb.sectionTag.SetProperty("weight", 700) // Bold
	tb.sectionTag.SetProperty("foreground", "#0066CC")
	tb.sectionTag.SetProperty("pixels-above-lines", 12)
	tb.sectionTag.SetProperty("pixels-below-lines", 6)
	tagTable.Add(tb.sectionTag)
	
	// Synopsis tag
	tb.synopsisTag = gtk.NewTextTag("synopsis")
	tb.synopsisTag.SetProperty("style", 2) // Italic
	tb.synopsisTag.SetProperty("foreground", "#666666")
	tb.synopsisTag.SetProperty("pixels-above-lines", 0)
	tb.synopsisTag.SetProperty("pixels-below-lines", 6)
	tagTable.Add(tb.synopsisTag)
}

func (tb *TextBuffer) connectSignals() {
	// Connect change signal
	tb.Buffer.ConnectChanged(func() {
		tb.onTextChanged()
	})
	
	// Connect cursor position changed
	tb.Buffer.ConnectCursorMoved(func() {
		tb.onCursorMoved()
	})
	
	// Connect undo/redo signals
	tb.undoManager.ConnectCanUndoChanged(func() {
		tb.onUndoRedoChanged()
	})
	
	tb.undoManager.ConnectCanRedoChanged(func() {
		tb.onUndoRedoChanged()
	})
}

func (tb *TextBuffer) onTextChanged() {
	tb.hasChanges = true
	tb.lastChangeTime = time.Now()
	
	// Schedule formatting update
	tb.scheduleFormatting()
	
	// Schedule auto-save
	tb.scheduleAutoSave()
	
	// Notify callbacks
	for _, callback := range tb.changeCallbacks {
		callback()
	}
}

func (tb *TextBuffer) onCursorMoved() {
	// Update current element context
	if tb.view != nil {
		tb.view.onCursorMoved()
	}
}

func (tb *TextBuffer) onUndoRedoChanged() {
	// Update UI state for undo/redo actions
	log.Printf("Undo available: %v, Redo available: %v", 
		tb.undoManager.CanUndo(), tb.undoManager.CanRedo())
}

func (tb *TextBuffer) scheduleFormatting() {
	if tb.suppressFormatting {
		return
	}
	
	// Cancel existing timer
	if tb.formatTimer != nil {
		tb.formatTimer.Destroy()
	}
	
	// Schedule new formatting update
	tb.formatTimer = glib.TimeoutAdd(100, func() bool {
		tb.updateFormatting()
		tb.formatTimer = nil
		return false
	})
}

func (tb *TextBuffer) scheduleAutoSave() {
	// Cancel existing timer
	if tb.autoSaveTimer != nil {
		tb.autoSaveTimer.Destroy()
	}
	
	// Schedule new auto-save
	tb.autoSaveTimer = glib.TimeoutAdd(uint(tb.autoSaveInterval.Milliseconds()), func() bool {
		tb.autoSave()
		tb.autoSaveTimer = nil
		return false
	})
}

func (tb *TextBuffer) updateFormatting() {
	if tb.suppressFormatting {
		return
	}
	
	// Clear existing formatting
	tb.clearFormatting()
	
	// Apply Fountain formatting
	tb.applyFountainFormatting()
}

func (tb *TextBuffer) clearFormatting() {
	start := tb.Buffer.GetStartIter()
	end := tb.Buffer.GetEndIter()
	
	// Remove all tags
	tb.Buffer.RemoveAllTags(&start, &end)
}

func (tb *TextBuffer) applyFountainFormatting() {
	text := tb.GetText()
	lines := strings.Split(text, "\n")
	
	tb.suppressFormatting = true
	defer func() {
		tb.suppressFormatting = false
	}()
	
	lineStart := 0
	for lineNum, line := range lines {
		lineEnd := lineStart + len(line)
		
		// Apply formatting based on line type
		tb.formatLine(lineNum, lineStart, lineEnd, line)
		
		// Move to next line (add 1 for newline character)
		lineStart = lineEnd + 1
	}
}

func (tb *TextBuffer) formatLine(lineNum, start, end int, line string) {
	trimmedLine := strings.TrimSpace(line)
	
	if len(trimmedLine) == 0 {
		return
	}
	
	startIter := tb.Buffer.GetIterAtOffset(start)
	endIter := tb.Buffer.GetIterAtOffset(end)
	
	// Determine line type and apply appropriate formatting
	if tb.isSceneHeading(trimmedLine) {
		tb.Buffer.ApplyTag(tb.sceneHeadingTag, &startIter, &endIter)
	} else if tb.isCharacterName(trimmedLine) {
		tb.Buffer.ApplyTag(tb.characterTag, &startIter, &endIter)
	} else if tb.isTransition(trimmedLine) {
		tb.Buffer.ApplyTag(tb.transitionTag, &startIter, &endIter)
	} else if tb.isParenthetical(trimmedLine) {
		tb.Buffer.ApplyTag(tb.parentheticalTag, &startIter, &endIter)
	} else if tb.isNote(trimmedLine) {
		tb.Buffer.ApplyTag(tb.noteTag, &startIter, &endIter)
	} else if tb.isBoneyard(trimmedLine) {
		tb.Buffer.ApplyTag(tb.boneyardTag, &startIter, &endIter)
	} else if tb.isCentered(trimmedLine) {
		tb.Buffer.ApplyTag(tb.centeredTag, &startIter, &endIter)
	} else if tb.isPageBreak(trimmedLine) {
		tb.Buffer.ApplyTag(tb.pageBreakTag, &startIter, &endIter)
	} else if tb.isSection(trimmedLine) {
		tb.Buffer.ApplyTag(tb.sectionTag, &startIter, &endIter)
	} else if tb.isSynopsis(trimmedLine) {
		tb.Buffer.ApplyTag(tb.synopsisTag, &startIter, &endIter)
	} else if tb.isDialogue(lineNum, trimmedLine) {
		tb.Buffer.ApplyTag(tb.dialogueTag, &startIter, &endIter)
	} else {
		tb.Buffer.ApplyTag(tb.actionTag, &startIter, &endIter)
	}
	
	// Apply inline formatting (emphasis, bold, underline)
	tb.applyInlineFormatting(start, end, line)
}

func (tb *TextBuffer) applyInlineFormatting(start, end int, line string) {
	// TODO: Implement inline formatting for *emphasis*, **bold**, _underline_
	// This would involve parsing the line for inline markup and applying tags
}

func (tb *TextBuffer) isSceneHeading(line string) bool {
	upper := strings.ToUpper(line)
	
	// Standard scene headings
	prefixes := []string{"INT.", "EXT.", "EST.", "INT ", "EXT ", "EST "}
	for _, prefix := range prefixes {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}
	
	// Forced scene headings
	if strings.HasPrefix(line, ".") {
		return true
	}
	
	return false
}

func (tb *TextBuffer) isCharacterName(line string) bool {
	// Character names are ALL CAPS
	if len(line) == 0 {
		return false
	}
	
	// Remove character extensions
	if strings.Contains(line, "(") {
		parts := strings.Split(line, "(")
		line = strings.TrimSpace(parts[0])
	}
	
	// Check if all uppercase
	for _, r := range line {
		if !((r >= 'A' && r <= 'Z') || r == ' ' || r == '.' || r == '\'' || r == '_' || r == '-') {
			return false
		}
	}
	
	return len(line) > 0
}

func (tb *TextBuffer) isTransition(line string) bool {
	upper := strings.ToUpper(line)
	
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
		if upper == transition {
			return true
		}
	}
	
	return strings.HasSuffix(upper, "TO:")
}

func (tb *TextBuffer) isParenthetical(line string) bool {
	return strings.HasPrefix(line, "(") && strings.HasSuffix(line, ")")
}

func (tb *TextBuffer) isNote(line string) bool {
	return strings.HasPrefix(line, "[[") && strings.HasSuffix(line, "]]")
}

func (tb *TextBuffer) isBoneyard(line string) bool {
	return strings.HasPrefix(line, "/*") && strings.HasSuffix(line, "*/")
}

func (tb *TextBuffer) isCentered(line string) bool {
	return strings.HasPrefix(line, ">") && strings.HasSuffix(line, "<")
}

func (tb *TextBuffer) isPageBreak(line string) bool {
	return strings.HasPrefix(line, "===") || strings.HasPrefix(line, "***")
}

func (tb *TextBuffer) isSection(line string) bool {
	return strings.HasPrefix(line, "#")
}

func (tb *TextBuffer) isSynopsis(line string) bool {
	return strings.HasPrefix(line, "=")
}

func (tb *TextBuffer) isDialogue(lineNum int, line string) bool {
	// Dialogue follows character names
	if lineNum == 0 {
		return false
	}
	
	// Get text up to this line
	text := tb.GetText()
	lines := strings.Split(text, "\n")
	
	// Look backwards for character name
	for i := lineNum - 1; i >= 0; i-- {
		prevLine := strings.TrimSpace(lines[i])
		if len(prevLine) == 0 {
			continue
		}
		
		if tb.isCharacterName(prevLine) {
			return true
		}
		
		// If we hit anything else, it's not dialogue
		break
	}
	
	return false
}

func (tb *TextBuffer) autoSave() {
	if !tb.hasChanges {
		return
	}
	
	// TODO: Implement auto-save functionality
	log.Println("Auto-save triggered")
}

func (tb *TextBuffer) GetText() string {
	start := tb.Buffer.GetStartIter()
	end := tb.Buffer.GetEndIter()
	return tb.Buffer.GetText(&start, &end, false)
}

func (tb *TextBuffer) SetText(text string) {
	tb.suppressFormatting = true
	tb.Buffer.SetText(text)
	tb.suppressFormatting = false
	tb.hasChanges = false
	tb.updateFormatting()
}

func (tb *TextBuffer) InsertAtCursor(text string) {
	tb.Buffer.InsertAtCursor(text)
}

func (tb *TextBuffer) HasChanges() bool {
	return tb.hasChanges
}

func (tb *TextBuffer) MarkSaved() {
	tb.hasChanges = false
}

func (tb *TextBuffer) ConnectChanged(callback func()) {
	tb.changeCallbacks = append(tb.changeCallbacks, callback)
}

func (tb *TextBuffer) Undo() {
	if tb.undoManager.CanUndo() {
		tb.undoManager.Undo()
	}
}

func (tb *TextBuffer) Redo() {
	if tb.undoManager.CanRedo() {
		tb.undoManager.Redo()
	}
}

func (tb *TextBuffer) CanUndo() bool {
	return tb.undoManager.CanUndo()
}

func (tb *TextBuffer) CanRedo() bool {
	return tb.undoManager.CanRedo()
}

func (tb *TextBuffer) GetCursorPosition() int {
	cursor := tb.Buffer.GetInsert()
	iter := tb.Buffer.GetIterAtMark(cursor)
	return iter.GetOffset()
}

func (tb *TextBuffer) SetCursorPosition(pos int) {
	iter := tb.Buffer.GetIterAtOffset(pos)
	tb.Buffer.PlaceCursor(&iter)
}

func (tb *TextBuffer) GetSelectedText() string {
	var start, end gtk.TextIter
	if tb.Buffer.GetSelectionBounds(&start, &end) {
		return tb.Buffer.GetText(&start, &end, false)
	}
	return ""
}

func (tb *TextBuffer) ReplaceSelectedText(text string) {
	var start, end gtk.TextIter
	if tb.Buffer.GetSelectionBounds(&start, &end) {
		tb.Buffer.Delete(&start, &end)
		tb.Buffer.Insert(&start, text)
	}
}

func (tb *TextBuffer) SelectAll() {
	start := tb.Buffer.GetStartIter()
	end := tb.Buffer.GetEndIter()
	tb.Buffer.SelectRange(&start, &end)
}

func (tb *TextBuffer) Cut() {
	clipboard := tb.Buffer.GetClipboard()
	tb.Buffer.CutClipboard(clipboard, true)
}

func (tb *TextBuffer) Copy() {
	clipboard := tb.Buffer.GetClipboard()
	tb.Buffer.CopyClipboard(clipboard)
}

func (tb *TextBuffer) Paste() {
	clipboard := tb.Buffer.GetClipboard()
	tb.Buffer.PasteClipboard(clipboard, nil, true)
}

func (tb *TextBuffer) GetStats() map[string]int {
	text := tb.GetText()
	
	// Count characters
	characters := len(text)
	
	// Count words
	words := len(strings.Fields(text))
	
	// Count lines
	lines := strings.Count(text, "\n") + 1
	
	// Count paragraphs (separated by blank lines)
	paragraphs := len(strings.Split(text, "\n\n"))
	
	// Count pages (rough estimate: 250 words per page)
	pages := (words + 249) / 250
	
	return map[string]int{
		"characters": characters,
		"words":      words,
		"lines":      lines,
		"paragraphs": paragraphs,
		"pages":      pages,
	}
}

func (tb *TextBuffer) GetElementAtCursor() string {
	cursor := tb.Buffer.GetInsert()
	iter := tb.Buffer.GetIterAtMark(cursor)
	
	// Get current line
	lineStart := iter
	lineStart.SetLineOffset(0)
	
	lineEnd := iter
	lineEnd.ForwardToLineEnd()
	
	line := tb.Buffer.GetText(&lineStart, &lineEnd, false)
	trimmedLine := strings.TrimSpace(line)
	
	// Determine element type
	if tb.isSceneHeading(trimmedLine) {
		return "Scene Heading"
	} else if tb.isCharacterName(trimmedLine) {
		return "Character"
	} else if tb.isTransition(trimmedLine) {
		return "Transition"
	} else if tb.isParenthetical(trimmedLine) {
		return "Parenthetical"
	} else if tb.isNote(trimmedLine) {
		return "Note"
	} else if tb.isBoneyard(trimmedLine) {
		return "Boneyard"
	} else if tb.isCentered(trimmedLine) {
		return "Centered"
	} else if tb.isPageBreak(trimmedLine) {
		return "Page Break"
	} else if tb.isSection(trimmedLine) {
		return "Section"
	} else if tb.isSynopsis(trimmedLine) {
		return "Synopsis"
	} else {
		return "Action"
	}
}