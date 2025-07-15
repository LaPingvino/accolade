package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"
)

// LexingtonParser wraps the Lexington library for Fountain parsing
type LexingtonParser struct {
	sceneHeaders []string
}

// ParsedElement represents a parsed Fountain element with styling info
type ParsedElement struct {
	Type     string
	Text     string
	Raw      string
	Line     int
	IsBold   bool
	IsItalic bool
	IsUnder  bool
}

// NewLexingtonParser creates a new parser with default configuration
func NewLexingtonParser() *LexingtonParser {
	// Default English scene headers
	sceneHeaders := []string{"INT", "EXT", "EST", "INT./EXT", "INT/EXT", "EXT/INT", "EXT./INT", "I/E"}
	return &LexingtonParser{
		sceneHeaders: sceneHeaders,
	}
}

// ParseText parses Fountain text and returns structured elements
func (lp *LexingtonParser) ParseText(text string) ([]ParsedElement, error) {
	// Parse the fountain text using Lexington
	reader := strings.NewReader(text)
	screenplay := fountain.Parse(lp.sceneHeaders, reader)

	var elements []ParsedElement
	lineNum := 1

	for _, line := range screenplay {
		parsedElement := ParsedElement{
			Type: string(line.Type),
			Text: string(line.Contents),
			Raw:  string(line.Contents),
			Line: lineNum,
		}

		// Check for formatting (bold, italic, underline)
		content := string(line.Contents)
		parsedElement.IsBold = strings.Contains(content, "**") || 
			(line.Type == lex.TypeScene) ||
			(line.Type == lex.TypeSpeaker)
		parsedElement.IsItalic = strings.Contains(content, "*") && !parsedElement.IsBold
		parsedElement.IsUnder = strings.Contains(content, "_")

		elements = append(elements, parsedElement)
		lineNum++
	}

	return elements, nil
}

// GetElementStyle returns formatting information for display
func (pe *ParsedElement) GetElementStyle() ElementStyle {
	style := ElementStyle{
		FontFamily: "Courier Prime", // Default screenplay font
		FontSize:   12,
		IsBold:     pe.IsBold,
		IsItalic:   pe.IsItalic,
		IsUnder:    pe.IsUnder,
	}

	switch pe.Type {
	case lex.TypeScene:
		style.FontSize = 12
		style.IsBold = true
		style.LeftMargin = 1.5
		style.RightMargin = 1.0
		style.Alignment = "left"
		
	case lex.TypeSpeaker:
		style.FontSize = 12
		style.LeftMargin = 3.7
		style.RightMargin = 1.5
		style.Alignment = "left"
		
	case lex.TypeDialog:
		style.FontSize = 12
		style.LeftMargin = 2.5
		style.RightMargin = 1.5
		style.Alignment = "left"
		
	case lex.TypeParen:
		style.FontSize = 12
		style.LeftMargin = 3.1
		style.RightMargin = 1.5
		style.Alignment = "left"
		
	case lex.TypeAction:
		style.FontSize = 12
		style.LeftMargin = 1.5
		style.RightMargin = 1.0
		style.Alignment = "left"
		
	case lex.TypeTrans:
		style.FontSize = 12
		style.LeftMargin = 1.5
		style.RightMargin = 1.0
		style.Alignment = "right"
		
	case lex.TypeCenter:
		style.FontSize = 12
		style.LeftMargin = 1.5
		style.RightMargin = 1.0
		style.Alignment = "center"
		
	case lex.TypeTitle:
		style.FontSize = 12
		style.LeftMargin = 1.5
		style.RightMargin = 1.0
		style.Alignment = "center"
		
	default:
		style.LeftMargin = 1.5
		style.RightMargin = 1.0
		style.Alignment = "left"
	}

	return style
}

// ElementStyle contains display formatting information
type ElementStyle struct {
	FontFamily  string
	FontSize    float32
	IsBold      bool
	IsItalic    bool
	IsUnder     bool
	LeftMargin  float64
	RightMargin float64
	Alignment   string // "left", "center", "right"
}

// FormatForPreview converts parsed elements to HTML for preview
func (lp *LexingtonParser) FormatForPreview(elements []ParsedElement) string {
	var html strings.Builder
	
	html.WriteString(`<div style="font-family: 'Courier Prime', 'Courier New', monospace; font-size: 12pt; line-height: 1.2; white-space: pre-wrap;">`)
	
	for _, element := range elements {
		style := element.GetElementStyle()
		
		// Build CSS style
		cssStyle := "margin-left: " + formatMargin(style.LeftMargin) + "; " +
			"margin-right: " + formatMargin(style.RightMargin) + "; " +
			"text-align: " + style.Alignment + "; "
		
		if style.IsBold {
			cssStyle += "font-weight: bold; "
		}
		if style.IsItalic {
			cssStyle += "font-style: italic; "
		}
		if style.IsUnder {
			cssStyle += "text-decoration: underline; "
		}
		
		// Add spacing for certain elements
		switch element.Type {
		case lex.TypeScene:
			cssStyle += "margin-top: 24pt; margin-bottom: 12pt; "
		case lex.TypeSpeaker:
			cssStyle += "margin-top: 12pt; margin-bottom: 0pt; "
		case lex.TypeTrans:
			cssStyle += "margin-top: 12pt; margin-bottom: 12pt; "
		case lex.TypeEmpty:
			html.WriteString("<br/>")
			continue
		}
		
		html.WriteString(`<div style="` + cssStyle + `">`)
		html.WriteString(strings.ReplaceAll(element.Text, "\n", "<br/>"))
		html.WriteString("</div>")
	}
	
	html.WriteString("</div>")
	return html.String()
}

// formatMargin converts inches to CSS units
func formatMargin(inches float64) string {
	return fmt.Sprintf("%.1fin", inches)
}

// GetAvailableFonts returns a list of recommended screenplay fonts
func GetAvailableFonts() []string {
	return []string{
		"Courier Prime",
		"Courier New", 
		"Courier",
		"Monaco",
		"Menlo",
		"Liberation Mono",
		"DejaVu Sans Mono",
	}
}

// IsScriptFont checks if a font is suitable for screenwriting
func IsScriptFont(fontName string) bool {
	scriptFonts := GetAvailableFonts()
	for _, font := range scriptFonts {
		if strings.EqualFold(fontName, font) {
			return true
		}
	}
	return false
}

// ParseLexFile parses a .lex file directly
func (lp *LexingtonParser) ParseLexFile(reader io.Reader) ([]ParsedElement, error) {
	screenplay := lex.Parse(reader)
	
	var elements []ParsedElement
	lineNum := 1

	for _, line := range screenplay {
		parsedElement := ParsedElement{
			Type: string(line.Type),
			Text: string(line.Contents),
			Raw:  string(line.Contents),
			Line: lineNum,
		}

		// Check for formatting
		content := string(line.Contents)
		parsedElement.IsBold = strings.Contains(content, "**") || 
			(line.Type == lex.TypeScene) ||
			(line.Type == lex.TypeSpeaker)
		parsedElement.IsItalic = strings.Contains(content, "*") && !parsedElement.IsBold
		parsedElement.IsUnder = strings.Contains(content, "_")

		elements = append(elements, parsedElement)
		lineNum++
	}

	return elements, nil
}