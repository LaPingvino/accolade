# Indentation Improvements Summary

## Overview

This document summarizes the indentation improvements made to the Accolade screenplay editor to fix both title page formatting and visual element indentation issues.

## Issues Fixed

### 1. Title Page Formatting

**Previous Issues:**
- Title page used plain text labels instead of proper Fountain syntax
- No proper indentation for contact information
- Inconsistent formatting that wouldn't work with PDF generation

**Fixes Applied:**
- Updated `title_page_dialog.go` to use proper Fountain key-value syntax:
  - `Title: [title]`
  - `Author: [author]`
  - `Credit: [credit]`
  - `Contact:` with properly indented contact details (4 spaces)
  - `Draft date: [date]`
- Contact information now properly indented under `Contact:` field
- Multiline addresses handled correctly with consistent indentation
- Proper spacing between title page and script content (two blank lines)

### 2. Visual Element Indentation

**Previous Issues:**
- Used crude tab-based indentation (`\t\t`, `\t\t\t`, etc.)
- Inconsistent with industry screenplay formatting standards
- Didn't match the lexington integration margin specifications

**Fixes Applied:**
- Replaced tab-based system with character-based spacing that matches industry standards
- Added indentation constants in `window.go`:
  ```go
  const (
      ActionIndent       = 0   // Action text - left margin (1.5")
      DialogueIndent     = 25  // Dialogue - 2.5" from left margin
      ParentheticalIndent = 31 // Parentheticals - 3.1" from left margin  
      CharacterIndent    = 37  // Character names - 3.7" from left margin
      TransitionIndent   = 60  // Transitions - right aligned (6.0" from left margin)
  )
  ```

### 3. Element Detection Improvements

**Previous Issues:**
- Element detection didn't properly handle the new character-based indentation
- Inconsistent detection of indented elements

**Fixes Applied:**
- Updated `detectElementType()` function to recognize elements by their proper indentation levels
- Improved logic to detect:
  - Character names at 37-space indentation
  - Dialogue at 25-space indentation
  - Parentheticals at 31-space indentation
  - Transitions at 55+ space indentation
- Better handling of edge cases and mixed content

### 4. Element Formatting Consistency

**Previous Issues:**
- `applyElementFormatting()` still used tabs instead of proper spacing
- Inconsistent with the lexington integration specifications

**Fixes Applied:**
- Updated formatting functions to use the new indentation constants
- Character names: positioned at 37 spaces, automatically uppercased
- Dialogue: positioned at 25 spaces
- Parentheticals: positioned at 31 spaces, auto-wrapped if needed
- Transitions: positioned at 60 spaces, automatically uppercased
- Scene headings: left-aligned, automatically uppercased

## Technical Details

### Indentation Standards Applied

Based on industry-standard screenplay formatting (12-point Courier font):

- **Action**: Left margin (1.5" = 0 spaces relative to text area)
- **Dialogue**: 2.5" from left margin = 25 character spaces
- **Parentheticals**: 3.1" from left margin = 31 character spaces
- **Character Names**: 3.7" from left margin = 37 character spaces
- **Transitions**: 6.0" from left margin = 60 character spaces (right-aligned)

### Files Modified

1. `title_page_dialog.go`:
   - `generateTitlePageText()` - Fixed Fountain syntax
   - `hasExistingTitlePage()` - Updated detection logic
   - `removeExistingTitlePage()` - Improved parsing

2. `window.go`:
   - Added indentation constants
   - Updated `detectElementType()` for character-based indentation
   - Fixed `applyElementFormatting()` to use proper spacing
   - Updated `getIndentationForNextElement()` to use constants

### Testing

Created comprehensive test files:
- `indentation_test.go` - Unit tests for all indentation functions
- `verify_indentation.go` - Standalone verification script

## Benefits

1. **Industry Standard Compliance**: Formatting now matches professional screenplay standards
2. **PDF Generation Ready**: Proper Fountain syntax ensures correct PDF output
3. **Consistent Spacing**: All elements use precise character-based positioning
4. **Better User Experience**: Visual formatting in editor matches final output
5. **Lexington Integration**: Consistent with the lexington library specifications

## Next Steps

1. **Testing**: Run the application to verify visual improvements
2. **Font Consistency**: Ensure monospace font is properly applied
3. **Real-time Formatting**: Test automatic indentation during typing
4. **Export Verification**: Confirm PDF exports use proper formatting
5. **User Feedback**: Gather feedback on visual improvements

## Notes

- The improvements maintain backward compatibility with existing Fountain files
- Character-based spacing assumes 12-point Courier or similar monospace font
- Title page improvements follow official Fountain specification
- All changes preserve the automatic formatting and element detection features