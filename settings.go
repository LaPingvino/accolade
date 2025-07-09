package main

import (
	"log"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

type Settings struct {
	*gio.Settings
	
	// Schema information
	schemaID   string
	schemaPath string
	
	// Cache for frequently accessed values
	cache      map[string]interface{}
	
	// Change handlers
	changeHandlers []func(string)
}

func NewSettings() *Settings {
	schemaID := "org.codeberg.lapingvino.Accolade"
	
	settings := &Settings{
		Settings:   gio.NewSettings(schemaID),
		schemaID:   schemaID,
		schemaPath: "/org/codeberg/lapingvino/Accolade/",
		cache:      make(map[string]interface{}),
		changeHandlers: make([]func(string), 0),
	}
	
	settings.setupSignals()
	
	return settings
}

func (s *Settings) setupSignals() {
	// Connect to change signals
	s.Settings.ConnectChanged(func(key string) {
		s.onSettingChanged(key)
	})
}

func (s *Settings) onSettingChanged(key string) {
	log.Printf("Setting changed: %s", key)
	
	// Clear cache for this key
	delete(s.cache, key)
	
	// Notify handlers
	for _, handler := range s.changeHandlers {
		handler(key)
	}
}

func (s *Settings) ConnectChanged(handler func(string)) {
	s.changeHandlers = append(s.changeHandlers, handler)
}

// String settings
func (s *Settings) GetString(key string) string {
	if cached, exists := s.cache[key]; exists {
		return cached.(string)
	}
	
	value := s.Settings.String(key)
	s.cache[key] = value
	return value
}

func (s *Settings) SetString(key, value string) {
	s.Settings.SetString(key, value)
	s.cache[key] = value
}

// Boolean settings
func (s *Settings) GetBoolean(key string) bool {
	if cached, exists := s.cache[key]; exists {
		return cached.(bool)
	}
	
	value := s.Settings.Boolean(key)
	s.cache[key] = value
	return value
}

func (s *Settings) SetBoolean(key string, value bool) {
	s.Settings.SetBoolean(key, value)
	s.cache[key] = value
}

// Integer settings
func (s *Settings) GetInt(key string) int {
	if cached, exists := s.cache[key]; exists {
		return cached.(int)
	}
	
	value := s.Settings.Int(key)
	s.cache[key] = value
	return value
}

func (s *Settings) SetInt(key string, value int) {
	s.Settings.SetInt(key, value)
	s.cache[key] = value
}

// Double settings
func (s *Settings) GetDouble(key string) float64 {
	if cached, exists := s.cache[key]; exists {
		return cached.(float64)
	}
	
	value := s.Settings.Double(key)
	s.cache[key] = value
	return value
}

func (s *Settings) SetDouble(key string, value float64) {
	s.Settings.SetDouble(key, value)
	s.cache[key] = value
}

// Enum settings
func (s *Settings) GetEnum(key string) int {
	if cached, exists := s.cache[key]; exists {
		return cached.(int)
	}
	
	value := s.Settings.Enum(key)
	s.cache[key] = value
	return value
}

func (s *Settings) SetEnum(key string, value int) {
	s.Settings.SetEnum(key, value)
	s.cache[key] = value
}

// Convenience methods for common settings

func (s *Settings) GetColorScheme() string {
	return s.GetString("color-scheme")
}

func (s *Settings) SetColorScheme(scheme string) {
	s.SetString("color-scheme", scheme)
}

func (s *Settings) GetInputFormat() string {
	return s.GetString("input-format")
}

func (s *Settings) SetInputFormat(format string) {
	s.SetString("input-format", format)
}

func (s *Settings) IsSpellCheckEnabled() bool {
	return s.GetBoolean("spellcheck")
}

func (s *Settings) SetSpellCheck(enabled bool) {
	s.SetBoolean("spellcheck", enabled)
}

func (s *Settings) IsSyncScrollEnabled() bool {
	return s.GetBoolean("sync-scroll")
}

func (s *Settings) SetSyncScroll(enabled bool) {
	s.SetBoolean("sync-scroll", enabled)
}

func (s *Settings) IsAutohideHeaderbar() bool {
	return s.GetBoolean("autohide-headerbar")
}

func (s *Settings) SetAutohideHeaderbar(enabled bool) {
	s.SetBoolean("autohide-headerbar", enabled)
}

func (s *Settings) GetOpenFilePath() string {
	return s.GetString("open-file-path")
}

func (s *Settings) SetOpenFilePath(path string) {
	s.SetString("open-file-path", path)
}

func (s *Settings) GetDefaultStat() string {
	return s.GetString("stat-default")
}

func (s *Settings) SetDefaultStat(stat string) {
	s.SetString("stat-default", stat)
}

func (s *Settings) GetCharactersPerLine() int {
	return s.GetInt("characters-per-line")
}

func (s *Settings) SetCharactersPerLine(count int) {
	s.SetInt("characters-per-line", count)
}

func (s *Settings) IsHemingwayMode() bool {
	return s.GetBoolean("hemingway-mode")
}

func (s *Settings) SetHemingwayMode(enabled bool) {
	s.SetBoolean("hemingway-mode", enabled)
}

func (s *Settings) GetHemingwayToastCount() int {
	return s.GetInt("hemingway-toast-count")
}

func (s *Settings) SetHemingwayToastCount(count int) {
	s.SetInt("hemingway-toast-count", count)
}

func (s *Settings) GetPreviewMode() int {
	return s.GetEnum("preview-mode")
}

func (s *Settings) SetPreviewMode(mode int) {
	s.SetEnum("preview-mode", mode)
}

func (s *Settings) GetPreviewSecurity() int {
	return s.GetEnum("preview-security")
}

func (s *Settings) SetPreviewSecurity(security int) {
	s.SetEnum("preview-security", security)
}

func (s *Settings) IsPreviewActive() bool {
	return s.GetBoolean("preview-active")
}

func (s *Settings) SetPreviewActive(active bool) {
	s.SetBoolean("preview-active", active)
}

func (s *Settings) IsBiggerText() bool {
	return s.GetBoolean("bigger-text")
}

func (s *Settings) SetBiggerText(bigger bool) {
	s.SetBoolean("bigger-text", bigger)
}

func (s *Settings) IsToolbarActive() bool {
	return s.GetBoolean("toolbar-active")
}

func (s *Settings) SetToolbarActive(active bool) {
	s.SetBoolean("toolbar-active", active)
}

// Utility methods

func (s *Settings) Reset() {
	// Reset all settings to default
	schema := s.Settings.Schema()
	for _, key := range schema.ListKeys() {
		s.Settings.Reset(key)
	}
	
	// Clear cache
	s.cache = make(map[string]interface{})
}

func (s *Settings) ResetKey(key string) {
	s.Settings.Reset(key)
	delete(s.cache, key)
}

func (s *Settings) HasKey(key string) bool {
	schema := s.Settings.Schema()
	keys := schema.ListKeys()
	
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	
	return false
}

func (s *Settings) GetSchemaID() string {
	return s.schemaID
}

func (s *Settings) GetSchemaPath() string {
	return s.schemaPath
}

func (s *Settings) Sync() {
	// Force synchronization of settings
	s.Settings.Sync()
}

func (s *Settings) GetDefaultValue(key string) *glib.Variant {
	schema := s.Settings.Schema()
	return schema.GetDefaultValue(key)
}

func (s *Settings) IsWritable(key string) bool {
	return s.Settings.IsWritable(key)
}

func (s *Settings) GetRange(key string) *glib.Variant {
	return s.Settings.GetRange(key)
}

func (s *Settings) RangeCheck(key string, value *glib.Variant) bool {
	return s.Settings.RangeCheck(key, value)
}

func (s *Settings) Delay() {
	s.Settings.Delay()
}

func (s *Settings) Apply() {
	s.Settings.Apply()
}

func (s *Settings) Revert() {
	s.Settings.Revert()
}

func (s *Settings) GetHasUnapplied() bool {
	return s.Settings.GetHasUnapplied()
}

// Settings validation
func (s *Settings) ValidateSettings() []string {
	var errors []string
	
	// Validate color scheme
	colorScheme := s.GetColorScheme()
	validSchemes := []string{"system", "light", "dark", "sepia"}
	valid := false
	for _, scheme := range validSchemes {
		if colorScheme == scheme {
			valid = true
			break
		}
	}
	if !valid {
		errors = append(errors, "Invalid color scheme: "+colorScheme)
	}
	
	// Validate characters per line
	charsPerLine := s.GetCharactersPerLine()
	if charsPerLine < 40 || charsPerLine > 200 {
		errors = append(errors, "Characters per line must be between 40 and 200")
	}
	
	// Validate input format
	inputFormat := s.GetInputFormat()
	if inputFormat != "fountain" {
		errors = append(errors, "Input format must be 'fountain'")
	}
	
	return errors
}

// Export/Import settings
func (s *Settings) ExportSettings() map[string]interface{} {
	exported := make(map[string]interface{})
	
	schema := s.Settings.Schema()
	keys := schema.ListKeys()
	
	for _, key := range keys {
		value := s.Settings.GetValue(key)
		exported[key] = value.String()
	}
	
	return exported
}

func (s *Settings) ImportSettings(settings map[string]interface{}) {
	for key, value := range settings {
		if s.HasKey(key) {
			switch v := value.(type) {
			case string:
				s.SetString(key, v)
			case bool:
				s.SetBoolean(key, v)
			case int:
				s.SetInt(key, v)
			case float64:
				s.SetDouble(key, v)
			default:
				log.Printf("Unsupported setting type for key %s: %T", key, v)
			}
		}
	}
}

// Backup and restore
func (s *Settings) BackupSettings() map[string]interface{} {
	return s.ExportSettings()
}

func (s *Settings) RestoreSettings(backup map[string]interface{}) {
	s.ImportSettings(backup)
}