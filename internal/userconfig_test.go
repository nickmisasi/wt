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
	if cfg.Workspace.Root != "workspace" {
		t.Errorf("expected default workspace root to be 'workspace', got %q", cfg.Workspace.Root)
	}
	if cfg.Worktrees.Path != "" {
		t.Errorf("expected default worktrees path to be empty, got %q", cfg.Worktrees.Path)
	}
	if cfg.Mattermost.Path != "" {
		t.Errorf("expected default mattermost path to be empty, got %q", cfg.Mattermost.Path)
	}
	if cfg.Mattermost.EnterprisePath != "" {
		t.Errorf("expected default enterprise path to be empty, got %q", cfg.Mattermost.EnterprisePath)
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
		{"workspace.root", true},
		{".workspace.root", true},
		{"worktrees.path", true},
		{"mattermost.path", true},
		{"mattermost.enterprise_path", true},
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

	// Workspace root
	val, err = cfg.GetConfigValue("workspace.root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "workspace" {
		t.Errorf("expected 'workspace', got %q", val)
	}

	// Worktrees path (empty default)
	val, err = cfg.GetConfigValue("worktrees.path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty string, got %q", val)
	}

	// Mattermost path (empty default)
	val, err = cfg.GetConfigValue("mattermost.path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty string, got %q", val)
	}

	// Enterprise path (empty default)
	val, err = cfg.GetConfigValue("mattermost.enterprise_path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty string, got %q", val)
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

	// Workspace root
	if err := cfg.SetConfigValue("workspace.root", "mm"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Workspace.Root != "mm" {
		t.Errorf("expected 'mm', got %q", cfg.Workspace.Root)
	}

	// Absolute path
	if err := cfg.SetConfigValue("workspace.root", "/opt/repos"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Workspace.Root != "/opt/repos" {
		t.Errorf("expected '/opt/repos', got %q", cfg.Workspace.Root)
	}

	// Worktrees path
	if err := cfg.SetConfigValue("worktrees.path", "/tmp/wt"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Worktrees.Path != "/tmp/wt" {
		t.Errorf("expected '/tmp/wt', got %q", cfg.Worktrees.Path)
	}

	// Mattermost path
	if err := cfg.SetConfigValue("mattermost.path", "mm/server"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mattermost.Path != "mm/server" {
		t.Errorf("expected 'mm/server', got %q", cfg.Mattermost.Path)
	}

	// Enterprise path
	if err := cfg.SetConfigValue("mattermost.enterprise_path", "mm/enterprise"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mattermost.EnterprisePath != "mm/enterprise" {
		t.Errorf("expected 'mm/enterprise', got %q", cfg.Mattermost.EnterprisePath)
	}

	// Invalid key
	if err := cfg.SetConfigValue("bogus", "val"); err == nil {
		t.Error("expected error for invalid key")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "wt", "config.json")

	cfg := DefaultUserConfig()
	cfg.Editor.Command = "neovim"
	cfg.Workspace.Root = "mm"
	cfg.Worktrees.Path = "/opt/worktrees"
	cfg.Mattermost.Path = "mm/mattermost"
	cfg.Mattermost.EnterprisePath = "mm/enterprise"

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

	if loaded.Editor.Command != "neovim" {
		t.Errorf("round-trip: expected editor command 'neovim', got %q", loaded.Editor.Command)
	}
	if loaded.Workspace.Root != "mm" {
		t.Errorf("round-trip: expected workspace root 'mm', got %q", loaded.Workspace.Root)
	}
	if loaded.Worktrees.Path != "/opt/worktrees" {
		t.Errorf("round-trip: expected worktrees path '/opt/worktrees', got %q", loaded.Worktrees.Path)
	}
	if loaded.Mattermost.Path != "mm/mattermost" {
		t.Errorf("round-trip: expected mattermost path 'mm/mattermost', got %q", loaded.Mattermost.Path)
	}
	if loaded.Mattermost.EnterprisePath != "mm/enterprise" {
		t.Errorf("round-trip: expected enterprise path 'mm/enterprise', got %q", loaded.Mattermost.EnterprisePath)
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

func TestResolvePath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	t.Run("empty value uses workspace root + fallback", func(t *testing.T) {
		got, err := resolvePath("", "/home/user/ws", "worktrees")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "/home/user/ws/worktrees" {
			t.Errorf("expected '/home/user/ws/worktrees', got %q", got)
		}
	})

	t.Run("absolute value used as-is", func(t *testing.T) {
		got, err := resolvePath("/opt/my-worktrees", "/home/user/ws", "worktrees")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "/opt/my-worktrees" {
			t.Errorf("expected '/opt/my-worktrees', got %q", got)
		}
	})

	t.Run("relative value resolved from HOME", func(t *testing.T) {
		got, err := resolvePath("mm/worktrees", "/home/user/ws", "worktrees")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join(homeDir, "mm/worktrees")
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})
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
