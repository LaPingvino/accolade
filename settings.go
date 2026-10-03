package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"

	"fyne.io/fyne/v2"
)

type Settings struct {
	mu     sync.RWMutex
	data   map[string]interface{}
	dirty  bool
}

func NewSettings() *Settings {
	s := &Settings{
		data: make(map[string]interface{}),
		dirty: false,
	}
	
	s.loadDefaults()
	s.load()
	
	return s
}

func (s *Settings) loadDefaults() {
	defaults := map[string]interface{}{
		// Theme settings
		"theme":                "system",
		"color-scheme":         "system",
		"use-sepia-theme":      false,
		
		// Editor settings
		"font-family":          "monospace",
		"font-size":           12,
		"auto-indent":         true,
		"show-line-numbers":   false,
		"highlight-current-line": true,
		"tab-width":           4,
		"use-spaces":          true,
		
		// Window settings
		"window-width":        1000,
		"window-height":       600,
		"window-maximized":    false,
		"preview-visible":     false,
		"toolbar-visible":     true,
		"statusbar-visible":   true,
		"fullscreen-mode":     false,
		
		// Editor behavior
		"auto-save":           true,
		"auto-save-interval":  30, // seconds
		"backup-files":        true,
		"spell-check":         true,
		"spell-check-language": "en_US",
		"autocomplete":        true,
		"smart-quotes":        false,
		
		// Writing settings
		"focus-mode":          false,
		"typewriter-mode":     false,
		"hemingway-mode":      false,
		"word-count-visible":  true,
		"character-count-visible": true,
		
		// Export settings
		"export-format":       "pdf",
		"export-directory":    "",
		"include-title-page":  true,
		"page-size":          "letter",
		"font-name":          "Courier",
		"font-size-export":   12,
		
		// Search settings
		"search-case-sensitive": false,
		"search-whole-words":   false,
		"search-regex":        false,
		"search-wrap-around":  true,
		
		// Recent files
		"recent-files":        []string{},
		"max-recent-files":    10,
		
		// Advanced settings
		"autohide-headerbar":  false,
		"smooth-scrolling":    true,
		"show-whitespace":     false,
		"highlight-matching-brackets": true,
		"auto-close-brackets": true,
		
		// Fountain-specific settings
		"fountain-auto-format": true,
		"fountain-scene-numbers": false,
		"fountain-dual-dialogue": true,
		"fountain-title-page":   true,
		
		// Backup and recovery
		"auto-backup":         true,
		"backup-directory":    "",
		"recovery-enabled":    true,
		"session-restore":     true,
	}
	
	for key, value := range defaults {
		s.data[key] = value
	}
}

func (s *Settings) load() {
	configFile := filepath.Join(getConfigDir(), "settings.json")
	
	file, err := os.Open(configFile)
	if err != nil {
		// Settings file doesn't exist, use defaults
		log.Printf("Settings file not found, using defaults: %v", err)
		return
	}
	defer file.Close()
	
	var loadedData map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&loadedData); err != nil {
		log.Printf("Error decoding settings: %v", err)
		return
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Merge loaded data with defaults
	for key, value := range loadedData {
		s.data[key] = value
	}
	
	log.Println("Settings loaded successfully")
}

func (s *Settings) save() error {
	if !s.dirty {
		return nil
	}
	
	configFile := filepath.Join(getConfigDir(), "settings.json")
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
		return err
	}
	
	file, err := os.Create(configFile)
	if err != nil {
		return err
	}
	defer file.Close()
	
	s.mu.RLock()
	data := make(map[string]interface{})
	for k, v := range s.data {
		data[k] = v
	}
	s.mu.RUnlock()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return err
	}
	
	s.dirty = false
	log.Println("Settings saved successfully")
	return nil
}

func (s *Settings) Save() error {
	return s.save()
}

// String settings
func (s *Settings) GetString(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if value, exists := s.data[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func (s *Settings) SetString(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.data[key] = value
	s.dirty = true
}

// Boolean settings
func (s *Settings) GetBoolean(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if value, exists := s.data[key]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

func (s *Settings) SetBoolean(key string, value bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.data[key] = value
	s.dirty = true
}

// Integer settings
func (s *Settings) GetInt(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if value, exists := s.data[key]; exists {
		// Handle both int and float64 (JSON unmarshaling)
		switch v := value.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return 0
}

func (s *Settings) SetInt(key string, value int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.data[key] = value
	s.dirty = true
}

// Float settings
func (s *Settings) GetFloat(key string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if value, exists := s.data[key]; exists {
		if f, ok := value.(float64); ok {
			return f
		}
		if i, ok := value.(int); ok {
			return float64(i)
		}
	}
	return 0.0
}

func (s *Settings) SetFloat(key string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.data[key] = value
	s.dirty = true
}

// String slice settings
func (s *Settings) GetStringSlice(key string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if value, exists := s.data[key]; exists {
		if slice, ok := value.([]interface{}); ok {
			result := make([]string, len(slice))
			for i, v := range slice {
				if str, ok := v.(string); ok {
					result[i] = str
				}
			}
			return result
		}
		if slice, ok := value.([]string); ok {
			return slice
		}
	}
	return []string{}
}

func (s *Settings) SetStringSlice(key string, value []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.data[key] = value
	s.dirty = true
}

// Recent files management
func (s *Settings) AddRecentFile(filePath string) {
	recentFiles := s.GetStringSlice("recent-files")
	maxRecent := s.GetInt("max-recent-files")
	
	// Remove if already exists
	for i, file := range recentFiles {
		if file == filePath {
			recentFiles = append(recentFiles[:i], recentFiles[i+1:]...)
			break
		}
	}
	
	// Add to beginning
	recentFiles = append([]string{filePath}, recentFiles...)
	
	// Limit to max items
	if len(recentFiles) > maxRecent {
		recentFiles = recentFiles[:maxRecent]
	}
	
	s.SetStringSlice("recent-files", recentFiles)
}

func (s *Settings) GetRecentFiles() []string {
	files := s.GetStringSlice("recent-files")
	
	// Filter out files that no longer exist
	validFiles := make([]string, 0, len(files))
	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			validFiles = append(validFiles, file)
		}
	}
	
	// Update the list if it changed
	if len(validFiles) != len(files) {
		s.SetStringSlice("recent-files", validFiles)
	}
	
	return validFiles
}

func (s *Settings) ClearRecentFiles() {
	s.SetStringSlice("recent-files", []string{})
}

// Window state management
func (s *Settings) SaveWindowState(size fyne.Size, position fyne.Position, maximized bool) {
	s.SetInt("window-width", int(size.Width))
	s.SetInt("window-height", int(size.Height))
	s.SetBoolean("window-maximized", maximized)
}

func (s *Settings) GetWindowState() (fyne.Size, fyne.Position, bool) {
	width := s.GetInt("window-width")
	height := s.GetInt("window-height")
	maximized := s.GetBoolean("window-maximized")
	
	if width <= 0 {
		width = 1000
	}
	if height <= 0 {
		height = 600
	}
	
	size := fyne.NewSize(float32(width), float32(height))
	position := fyne.NewPos(0, 0) // Let the OS decide
	
	return size, position, maximized
}

// Reset settings to defaults
func (s *Settings) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.data = make(map[string]interface{})
	s.loadDefaults()
	s.dirty = true
}

func (s *Settings) ResetKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Create a temporary settings object to get the default value
	temp := &Settings{data: make(map[string]interface{})}
	temp.loadDefaults()
	
	if defaultValue, exists := temp.data[key]; exists {
		s.data[key] = defaultValue
		s.dirty = true
	}
}

// Check if key exists
func (s *Settings) HasKey(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	_, exists := s.data[key]
	return exists
}

// Get all keys
func (s *Settings) GetKeys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	keys := make([]string, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}
	return keys
}

// Export settings to file
func (s *Settings) ExportToFile(filePath string) error {
	s.mu.RLock()
	data := make(map[string]interface{})
	for k, v := range s.data {
		data[k] = v
	}
	s.mu.RUnlock()
	
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// Import settings from file
func (s *Settings) ImportFromFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	var importedData map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&importedData); err != nil {
		return err
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for key, value := range importedData {
		s.data[key] = value
	}
	s.dirty = true
	
	return nil
}

// Auto-save periodically (call this in a goroutine)
func (s *Settings) AutoSave() {
	if s.dirty {
		if err := s.save(); err != nil {
			log.Printf("Error auto-saving settings: %v", err)
		}
	}
}

// Helper function to ensure config directory exists
func ensureConfigDir() error {
	configDir := getConfigDir()
	return os.MkdirAll(configDir, 0755)
}

// Helper function to get config file path
func getConfigFilePath() string {
	return filepath.Join(getConfigDir(), "settings.json")
}

// Global settings instance
var globalSettings *Settings
var settingsOnce sync.Once

func GetSettings() *Settings {
	settingsOnce.Do(func() {
		if err := ensureConfigDir(); err != nil {
			log.Printf("Warning: Could not create config directory: %v", err)
		}
		globalSettings = NewSettings()
	})
	return globalSettings
}

// Save settings on application exit
func SaveSettingsOnExit() {
	if globalSettings != nil {
		if err := globalSettings.Save(); err != nil {
			log.Printf("Error saving settings on exit: %v", err)
		}
	}
}