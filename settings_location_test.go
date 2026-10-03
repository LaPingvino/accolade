package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDirHonoursXDG(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if got := getConfigDir(); got != filepath.Join(dir, "accolade") {
		t.Errorf("getConfigDir() = %q", got)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	want, _ := os.UserConfigDir()
	if got := getConfigDir(); got != filepath.Join(want, "accolade") {
		t.Errorf("without XDG_CONFIG_HOME: %q, want the system's %q", got, want)
	}
}

func TestCopyLegacySettings(t *testing.T) {
	legacy, dir := t.TempDir(), t.TempDir()
	if copied, err := copyLegacySettings(dir, legacy); copied || err != nil {
		t.Fatalf("nothing to copy: copied=%v err=%v", copied, err)
	}
	os.WriteFile(filepath.Join(legacy, "settings.json"), []byte(`{"theme":"sepia"}`), 0o600)
	if copied, err := copyLegacySettings(dir, legacy); !copied || err != nil {
		t.Fatalf("first run: copied=%v err=%v", copied, err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "settings.json")); string(got) != `{"theme":"sepia"}` {
		t.Errorf("copied %q", got)
	}
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"theme":"dark"}`), 0o600)
	if copied, _ := copyLegacySettings(dir, legacy); copied {
		t.Error("existing settings overwritten")
	}
	if copied, _ := copyLegacySettings(legacy, legacy); copied {
		t.Error("copied a directory onto itself")
	}
}
