package internal

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	WorktreeBaseDir = "worktrees"
	CDMarker        = "__WT_CD__:"
)

// Config holds the configuration for the worktree manager
type Config struct {
	WorktreeBasePath string
	RepoName         string
	RepoRoot         string
}

// NewConfig creates a new configuration instance
func NewConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return &Config{
		WorktreeBasePath: filepath.Join(homeDir, "workspace", WorktreeBaseDir),
	}, nil
}

// GetWorktreePath returns the full path for a worktree given a branch name
func (c *Config) GetWorktreePath(branch string) string {
	sanitized := sanitizeBranchName(branch)
	worktreeName := c.RepoName + "-" + sanitized
	return filepath.Join(c.WorktreeBasePath, worktreeName)
}

// sanitizeBranchName removes or replaces characters that are problematic in filesystem paths
func sanitizeBranchName(branch string) string {
	// Replace common problematic characters
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
	)
	return replacer.Replace(branch)
}

// StripRepoPrefix removes the repo name prefix from a worktree directory name
func (c *Config) StripRepoPrefix(worktreeName string) string {
	prefix := c.RepoName + "-"
	if strings.HasPrefix(worktreeName, prefix) {
		return strings.TrimPrefix(worktreeName, prefix)
	}
	return worktreeName
}

