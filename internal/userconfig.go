package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// EditorConfig holds editor-related settings.
type EditorConfig struct {
	Command string `json:"command"`
}

// UserConfig holds user-facing persistent settings (distinct from the runtime Config).
type UserConfig struct {
	Editor EditorConfig `json:"editor"`
}

// DefaultUserConfig returns a UserConfig populated with default values.
func DefaultUserConfig() UserConfig {
	return UserConfig{
		Editor: EditorConfig{
			Command: "cursor",
		},
	}
}

// validKeys returns the set of recognised configuration key names.
func validKeys() map[string]bool {
	return map[string]bool{
		"editor.command": true,
	}
}

// UserConfigPath returns the path to the config file:
// <os.UserConfigDir>/wt/config.json
func UserConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine config directory: %w", err)
	}
	return filepath.Join(dir, "wt", "config.json"), nil
}

// LoadUserConfig reads the config file from disk. If the file does not exist
// the returned config contains default values and no error is returned.
func LoadUserConfig() (*UserConfig, error) {
	cfg := DefaultUserConfig()

	path, err := UserConfigPath()
	if err != nil {
		return &cfg, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}
		return &cfg, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return &cfg, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// SaveUserConfig writes the config to disk, creating the parent directory if
// needed.
func SaveUserConfig(cfg *UserConfig) error {
	path, err := UserConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// NormalizeKey strips a leading dot from a config key for user convenience.
func NormalizeKey(key string) string {
	return strings.TrimPrefix(key, ".")
}

// IsValidKey reports whether key (after normalisation) is a recognised config key.
func IsValidKey(key string) bool {
	return validKeys()[NormalizeKey(key)]
}

// ValidKeyNames returns a sorted slice of valid key names (for error messages).
func ValidKeyNames() []string {
	vk := validKeys()
	keys := make([]string, 0, len(vk))
	for k := range vk {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// GetConfigValue returns the string value of the given config key.
func (c *UserConfig) GetConfigValue(key string) (string, error) {
	switch NormalizeKey(key) {
	case "editor.command":
		return c.Editor.Command, nil
	default:
		return "", fmt.Errorf("unknown config key: %s (valid keys: %s)", key, strings.Join(ValidKeyNames(), ", "))
	}
}

// SetConfigValue sets the value of the given config key.
func (c *UserConfig) SetConfigValue(key, value string) error {
	switch NormalizeKey(key) {
	case "editor.command":
		c.Editor.Command = value
		return nil
	default:
		return fmt.Errorf("unknown config key: %s (valid keys: %s)", key, strings.Join(ValidKeyNames(), ", "))
	}
}

// marshalConfig serialises a UserConfig to indented JSON with a trailing newline.
func marshalConfig(cfg *UserConfig) ([]byte, error) {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// loadConfigFromPath reads a UserConfig from a specific file path, returning
// defaults when the file does not exist.
func loadConfigFromPath(path string) (*UserConfig, error) {
	cfg := DefaultUserConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}
		return &cfg, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return &cfg, err
	}

	return &cfg, nil
}
