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
		
		// Editor settings
		"font-family":          "monospace",
		"font-size":           12,
		"auto-indent":         true,
		"show-line-numbers":   false,
		
		// Window settings
		"preview-visible":     false,
		"toolbar-visible":     true,
		"statusbar-visible":   true,
		
		// Editor behavior
		"auto-save":           true,
		"auto-save-interval":  30, // seconds
		
		// Export settings
		"export-format":       "PDF",
		"script-format":       "default", // a Lexington preset (script_format.go)
		"script-language":     "en",      // of new scripts: their scene headings (language.go)
		"export-directory":    "",
		"include-title-page":  true,
		"page-size":          "", // the script format's (export dialog)
		
		"auto-close-brackets": true,
		
		// Fountain-specific settings
		"fountain-scene-numbers": false,
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
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	if copied, err := copyLegacySettings(configDir, legacyConfigDir()); err != nil {
		log.Printf("Could not copy settings from %s: %v", legacyConfigDir(), err)
	} else if copied {
		log.Printf("Copied settings from %s", legacyConfigDir())
	}
	return nil
}

// copyLegacySettings copies settings.json from the old directory to dir
// if dir has none yet (on macOS and Windows the settings moved out of
// ~/.config). The old file is left alone.
func copyLegacySettings(dir, legacy string) (bool, error) {
	if filepath.Clean(dir) == filepath.Clean(legacy) {
		return false, nil
	}
	target := filepath.Join(dir, "settings.json")
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		return false, err
	}
	data, err := os.ReadFile(filepath.Join(legacy, "settings.json"))
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, os.WriteFile(target, data, 0600)
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