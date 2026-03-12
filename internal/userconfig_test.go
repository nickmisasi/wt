package internal

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
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

func TestClaudemuxConfigDefaults(t *testing.T) {
	cfg := DefaultUserConfig()
	if cfg.Claudemux.Enabled {
		t.Error("expected claudemux.enabled to default to false")
	}
	if cfg.Claudemux.Command == "" {
		t.Error("expected claudemux.command to have a default value")
	}
	if cfg.Claudemux.MaxSessions != 10 {
		t.Errorf("expected claudemux.max_sessions to default to 10, got %d", cfg.Claudemux.MaxSessions)
	}
}

func TestGetSetClaudemuxEnabled(t *testing.T) {
	cfg := DefaultUserConfig()

	if err := cfg.SetConfigValue("claudemux.enabled", "true"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val, err := cfg.GetConfigValue("claudemux.enabled")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "true" {
		t.Errorf("expected 'true', got %q", val)
	}

	// Also test "1" as truthy
	if err := cfg.SetConfigValue("claudemux.enabled", "1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Claudemux.Enabled {
		t.Error("expected '1' to set enabled to true")
	}

	// Test false
	if err := cfg.SetConfigValue("claudemux.enabled", "false"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Claudemux.Enabled {
		t.Error("expected 'false' to set enabled to false")
	}
}

func TestGetSetClaudemuxCommand(t *testing.T) {
	cfg := DefaultUserConfig()

	if err := cfg.SetConfigValue("claudemux.command", "claude --resume"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val, err := cfg.GetConfigValue("claudemux.command")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "claude --resume" {
		t.Errorf("expected 'claude --resume', got %q", val)
	}
}

func TestGetSetClaudemuxMaxSessions(t *testing.T) {
	cfg := DefaultUserConfig()

	if err := cfg.SetConfigValue("claudemux.max_sessions", "5"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val, err := cfg.GetConfigValue("claudemux.max_sessions")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "5" {
		t.Errorf("expected '5', got %q", val)
	}

	// Non-integer should fail
	if err := cfg.SetConfigValue("claudemux.max_sessions", "abc"); err == nil {
		t.Error("expected error for non-integer max_sessions")
	}

	// Zero/negative should fail
	if err := cfg.SetConfigValue("claudemux.max_sessions", "0"); err == nil {
		t.Error("expected error for zero max_sessions")
	}
	if err := cfg.SetConfigValue("claudemux.max_sessions", "-1"); err == nil {
		t.Error("expected error for negative max_sessions")
	}
}

func TestSetConfigValue_MaxSessionsValidation(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "positive integer succeeds",
			value:   "5",
			wantErr: false,
		},
		{
			name:    "one succeeds",
			value:   "1",
			wantErr: false,
		},
		{
			name:    "zero rejected",
			value:   "0",
			wantErr: true,
			errMsg:  "positive integer",
		},
		{
			name:    "negative rejected",
			value:   "-3",
			wantErr: true,
			errMsg:  "positive integer",
		},
		{
			name:    "non-integer rejected",
			value:   "abc",
			wantErr: true,
			errMsg:  "positive integer",
		},
		{
			name:    "float rejected",
			value:   "3.5",
			wantErr: true,
			errMsg:  "positive integer",
		},
		{
			name:    "large value succeeds",
			value:   "100",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultUserConfig()
			err := cfg.SetConfigValue("claudemux.max_sessions", tt.value)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for value %q, got nil", tt.value)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for value %q: %v", tt.value, err)
				}
			}
		})
	}
}

func TestClaudemuxKeysAreValid(t *testing.T) {
	keys := []string{"claudemux.enabled", "claudemux.command", "claudemux.max_sessions"}
	for _, key := range keys {
		if !IsValidKey(key) {
			t.Errorf("expected %q to be a valid key", key)
		}
	}
}

func TestClaudemuxConfigRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "wt", "config.json")

	cfg := DefaultUserConfig()
	cfg.Claudemux.Enabled = true
	cfg.Claudemux.Command = "claude --resume"
	cfg.Claudemux.MaxSessions = 5

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

	if !loaded.Claudemux.Enabled {
		t.Error("round-trip: expected claudemux.enabled to be true")
	}
	if loaded.Claudemux.Command != "claude --resume" {
		t.Errorf("round-trip: expected command 'claude --resume', got %q", loaded.Claudemux.Command)
	}
	if loaded.Claudemux.MaxSessions != 5 {
		t.Errorf("round-trip: expected max_sessions 5, got %d", loaded.Claudemux.MaxSessions)
	}
}
