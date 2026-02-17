package internal

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestDefaultUserConfig(t *testing.T) {
	cfg := DefaultUserConfig()
	if cfg.Editor.Command != "cursor" {
		t.Errorf("expected default editor command to be 'cursor', got %q", cfg.Editor.Command)
	}
}

func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"editor.command", "editor.command"},
		{".editor.command", "editor.command"},
		{"..editor", ".editor"}, // only one leading dot stripped
		{"", ""},
	}

	for _, tt := range tests {
		got := NormalizeKey(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsValidKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"editor.command", true},
		{".editor.command", true},
		{"editor", false},
		{"bogus", false},
		{"", false},
	}

	for _, tt := range tests {
		got := IsValidKey(tt.key)
		if got != tt.want {
			t.Errorf("IsValidKey(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestGetConfigValue(t *testing.T) {
	cfg := DefaultUserConfig()

	val, err := cfg.GetConfigValue("editor.command")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "cursor" {
		t.Errorf("expected 'cursor', got %q", val)
	}

	// With leading dot
	val, err = cfg.GetConfigValue(".editor.command")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "cursor" {
		t.Errorf("expected 'cursor', got %q", val)
	}

	// Invalid key
	_, err = cfg.GetConfigValue("bogus")
	if err == nil {
		t.Error("expected error for invalid key")
	}
}

func TestSetConfigValue(t *testing.T) {
	cfg := DefaultUserConfig()

	if err := cfg.SetConfigValue("editor.command", "vim"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Editor.Command != "vim" {
		t.Errorf("expected 'vim', got %q", cfg.Editor.Command)
	}

	// With leading dot
	if err := cfg.SetConfigValue(".editor.command", "code"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Editor.Command != "code" {
		t.Errorf("expected 'code', got %q", cfg.Editor.Command)
	}

	// Invalid key
	if err := cfg.SetConfigValue("bogus", "val"); err == nil {
		t.Error("expected error for invalid key")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	// Use a temp directory as the config home
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "wt", "config.json")

	// Patch UserConfigPath by saving directly to the temp path
	cfg := DefaultUserConfig()
	cfg.Editor.Command = "neovim"

	// Create dir and write
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	// Save using our helper (manually, since we can't override UserConfigPath easily)
	data, err := marshalConfig(&cfg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Read back
	loaded, err := loadConfigFromPath(configPath)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if loaded.Editor.Command != "neovim" {
		t.Errorf("round-trip: expected editor command 'neovim', got %q", loaded.Editor.Command)
	}
}

func TestLoadConfigFromMissingFile(t *testing.T) {
	cfg, err := loadConfigFromPath("/nonexistent/path/config.json")
	if err != nil {
		t.Fatalf("unexpected error for missing file: %v", err)
	}
	if cfg.Editor.Command != "cursor" {
		t.Errorf("expected default editor command 'cursor', got %q", cfg.Editor.Command)
	}
}

func TestValidKeyNames(t *testing.T) {
	names := ValidKeyNames()
	if len(names) == 0 {
		t.Error("expected at least one valid key name")
	}

	found := false
	for _, n := range names {
		if n == "editor.command" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'editor.command' in valid key names")
	}
}

func TestValidKeyNamesIsSorted(t *testing.T) {
	names := ValidKeyNames()
	if !sort.StringsAreSorted(names) {
		t.Errorf("ValidKeyNames() returned unsorted slice: %v", names)
	}
}

func TestSetConfigValueWithSpaces(t *testing.T) {
	cfg := DefaultUserConfig()

	// Simulate what happens when the CLI joins multi-word args with spaces
	if err := cfg.SetConfigValue("editor.command", "/usr/local/bin/my editor"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Editor.Command != "/usr/local/bin/my editor" {
		t.Errorf("expected '/usr/local/bin/my editor', got %q", cfg.Editor.Command)
	}

	// Verify round-trip preserves spaces
	val, err := cfg.GetConfigValue("editor.command")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "/usr/local/bin/my editor" {
		t.Errorf("round-trip: expected '/usr/local/bin/my editor', got %q", val)
	}
}

func TestSaveAndLoadRoundTripWithSpaces(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "wt", "config.json")

	cfg := DefaultUserConfig()
	cfg.Editor.Command = "code --wait"

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	data, err := marshalConfig(&cfg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	loaded, err := loadConfigFromPath(configPath)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if loaded.Editor.Command != "code --wait" {
		t.Errorf("round-trip: expected 'code --wait', got %q", loaded.Editor.Command)
	}
}
