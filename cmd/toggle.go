package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nickmisasi/wt/internal"
)

// RunToggle switches from worktree back to parent repository
func RunToggle() error {
	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	workspaceRoot := filepath.Join(homeDir, "workspace")
	worktreesDir := filepath.Join(workspaceRoot, "worktrees")

	// Check if we're in a worktree directory
	if !strings.HasPrefix(cwd, worktreesDir) {
		return fmt.Errorf("not currently in a worktree directory")
	}

	// Determine which repository to return to
	var targetRepo string

	// Check if we're in a Mattermost dual-repo worktree
	if strings.Contains(cwd, "/mattermost-") {
		// Could be in either mattermost/ or enterprise/ subdirectory
		if strings.Contains(cwd, "/mattermost/") {
			targetRepo = filepath.Join(workspaceRoot, "mattermost")
		} else if strings.Contains(cwd, "/enterprise/") {
			targetRepo = filepath.Join(workspaceRoot, "enterprise")
		} else {
			// At the root of a mattermost worktree, default to mattermost repo
			targetRepo = filepath.Join(workspaceRoot, "mattermost")
		}
	} else {
		// Use git worktree list to find the parent repository
		// The first entry in the list is always the main repository
		targetRepo, err = getParentRepositoryPath()
		if err != nil {
			return fmt.Errorf("failed to determine parent repository: %w", err)
		}
	}

	// Verify target repository exists
	if _, err := os.Stat(targetRepo); os.IsNotExist(err) {
		return fmt.Errorf("parent repository not found: %s", targetRepo)
	}

	// Output CD marker for shell integration
	fmt.Printf("Returning to parent repository: %s\n", targetRepo)
	fmt.Printf("%s%s\n", internal.CDMarker, targetRepo)

	return nil
}

// getParentRepositoryPath uses git worktree list to find the parent repository path
func getParentRepositoryPath() (string, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Parse the output - the first "worktree" entry is the main repository
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			return path, nil
		}
	}

	return "", fmt.Errorf("no parent repository found in git worktree list")
}
