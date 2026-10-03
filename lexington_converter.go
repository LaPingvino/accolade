package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type LexingtonConverter struct {
	// Configuration
	configPath   string
	tempDir      string
	
	// Conversion settings
	theme        string
	margins      map[string]float64
	fontSize     int
	fontFamily   string
	
	// Cache
	lastContent  string
	lastHTML     string
	lastPDF      string
	cacheTime    time.Time
	cacheTimeout time.Duration
}

func NewLexingtonConverter() *LexingtonConverter {
	tempDir, err := os.MkdirTemp("", "accolade-preview-")
	if err != nil {
		log.Printf("Failed to create temp directory: %v", err)
		tempDir = "/tmp"
	}
	
	return &LexingtonConverter{
		tempDir:      tempDir,
		theme:        "default",
		margins:      make(map[string]float64),
		fontSize:     12,
		fontFamily:   "Courier Prime",
		cacheTimeout: 1 * time.Second,
	}
}

func (lc *LexingtonConverter) ConvertToHTML(fountainText string, restrictedMode bool) (string, error) {
	// Check cache first
	if lc.lastContent == fountainText && 
	   lc.lastHTML != "" && 
	   time.Since(lc.cacheTime) < lc.cacheTimeout {
		return lc.lastHTML, nil
	}
	
	// Create temporary input file
	inputFile := filepath.Join(lc.tempDir, "input.fountain")
	err := os.WriteFile(inputFile, []byte(fountainText), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write input file: %v", err)
	}
	defer os.Remove(inputFile)
	
	// Build lexington command
	cmd := exec.Command("lexington", "-i", inputFile, "-to", "html")
	
	// Add configuration if available
	if lc.configPath != "" {
		cmd.Args = append(cmd.Args, "-config", lc.configPath)
	}
	
	// Execute lexington
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("lexington conversion failed: %v", err)
	}
	
	htmlContent := string(output)
	
	// Apply restricted mode if needed
	if restrictedMode {
		htmlContent = lc.sanitizeHTML(htmlContent)
	}
	
	// Update cache
	lc.lastContent = fountainText
	lc.lastHTML = htmlContent
	lc.cacheTime = time.Now()
	
	return htmlContent, nil
}

func (lc *LexingtonConverter) ConvertToPDF(fountainText string, outputPath string) error {
	// Create temporary input file
	inputFile := filepath.Join(lc.tempDir, "input.fountain")
	err := os.WriteFile(inputFile, []byte(fountainText), 0644)
	if err != nil {
		return fmt.Errorf("failed to write input file: %v", err)
	}
	defer os.Remove(inputFile)
	
	// Build lexington command
	cmd := exec.Command("lexington", "-i", inputFile, "-o", outputPath, "-to", "pdf")
	
	// Add configuration if available
	if lc.configPath != "" {
		cmd.Args = append(cmd.Args, "-config", lc.configPath)
	}
	
	// Execute lexington
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("lexington PDF conversion failed: %v", err)
	}
	
	return nil
}

func (lc *LexingtonConverter) ConvertToFDX(fountainText string, outputPath string) error {
	// Create temporary input file
	inputFile := filepath.Join(lc.tempDir, "input.fountain")
	err := os.WriteFile(inputFile, []byte(fountainText), 0644)
	if err != nil {
		return fmt.Errorf("failed to write input file: %v", err)
	}
	defer os.Remove(inputFile)
	
	// Build lexington command
	cmd := exec.Command("lexington", "-i", inputFile, "-o", outputPath, "-to", "fdx")
	
	// Execute lexington
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("lexington FDX conversion failed: %v", err)
	}
	
	return nil
}

func (lc *LexingtonConverter) sanitizeHTML(html string) string {
	// Remove JavaScript and other potentially unsafe content
	html = strings.ReplaceAll(html, "<script", "<!-- script")
	html = strings.ReplaceAll(html, "</script>", "script -->")
	html = strings.ReplaceAll(html, "javascript:", "")
	html = strings.ReplaceAll(html, "onclick=", "data-onclick=")
	html = strings.ReplaceAll(html, "onload=", "data-onload=")
	html = strings.ReplaceAll(html, "onerror=", "data-onerror=")
	
	return html
}

func (lc *LexingtonConverter) SetTheme(theme string) {
	lc.theme = theme
	lc.invalidateCache()
}

func (lc *LexingtonConverter) SetMargins(margins map[string]float64) {
	lc.margins = margins
	lc.invalidateCache()
}

func (lc *LexingtonConverter) SetFont(family string, size int) {
	lc.fontFamily = family
	lc.fontSize = size
	lc.invalidateCache()
}

func (lc *LexingtonConverter) SetConfigPath(path string) {
	lc.configPath = path
	lc.invalidateCache()
}

func (lc *LexingtonConverter) invalidateCache() {
	lc.lastContent = ""
	lc.lastHTML = ""
	lc.lastPDF = ""
}

func (lc *LexingtonConverter) GetSupportedFormats() []string {
	return []string{
		"html",
		"pdf",
		"fdx",
		"latex",
		"epub",
		"markdown",
	}
}

func (lc *LexingtonConverter) IsLexingtonAvailable() bool {
	_, err := exec.LookPath("lexington")
	return err == nil
}

func (lc *LexingtonConverter) GetLexingtonVersion() (string, error) {
	cmd := exec.Command("lexington", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	return strings.TrimSpace(string(output)), nil
}

// ValidationError is a problem found in a Fountain script.
type ValidationError struct {
	Line    int
	Message string
}

func (lc *LexingtonConverter) ValidateFountain(fountainText string) []ValidationError {
	// TODO: Use lexington linter to validate Fountain syntax
	errors := []ValidationError{}
	
	// Create temporary input file
	inputFile := filepath.Join(lc.tempDir, "input.fountain")
	err := os.WriteFile(inputFile, []byte(fountainText), 0644)
	if err != nil {
		errors = append(errors, ValidationError{
			Line:    0,
			Message: "Failed to write temporary file: " + err.Error(),
		})
		return errors
	}
	defer os.Remove(inputFile)
	
	// Run lexington linter if available
	cmd := exec.Command("lexington", "-i", inputFile, "-lint")
	output, err := cmd.Output()
	if err != nil {
		// Linter might not be available, that's okay
		return errors
	}
	
	// Parse linter output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		// Parse error format: "line:column: message"
		parts := strings.SplitN(line, ":", 3)
		if len(parts) >= 3 {
			// TODO: Parse line number from parts[0]
			errors = append(errors, ValidationError{
				Line:    0, // TODO: convert parts[0] to int
				Message: strings.TrimSpace(parts[2]),
			})
		}
	}
	
	return errors
}

func (lc *LexingtonConverter) GetElementAtPosition(fountainText string, line, column int) (string, error) {
	// TODO: Use lexington to parse structure and determine element type at position
	return "action", nil
}

func (lc *LexingtonConverter) GetStructure(fountainText string) (*FountainStructure, error) {
	// TODO: Parse Fountain structure using lexington
	return &FountainStructure{
		Scenes:     []SceneInfo{},
		Characters: []string{},
		Locations:  []string{},
	}, nil
}

func (lc *LexingtonConverter) Cleanup() {
	if lc.tempDir != "" && lc.tempDir != "/tmp" {
		os.RemoveAll(lc.tempDir)
	}
}

// Supporting types

type FountainStructure struct {
	Scenes     []SceneInfo
	Characters []string
	Locations  []string
}

type SceneInfo struct {
	Number      int
	Heading     string
	Location    string
	TimeOfDay   string
	PageNumber  int
	Characters  []string
}

