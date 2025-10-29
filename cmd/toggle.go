package cmd

import (
	"fmt"
	"os"
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
		// Standard worktree - extract repo name from worktree directory
		// Pattern: ~/workspace/worktrees/<repo-name>-<branch-name>/
		relPath, err := filepath.Rel(worktreesDir, cwd)
		if err != nil {
			return fmt.Errorf("failed to determine relative path: %w", err)
		}

		// Get the worktree root directory name
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) == 0 {
			return fmt.Errorf("could not determine worktree directory")
		}

		worktreeDirName := parts[0]

		// Extract repo name from pattern: <repo-name>-<branch-name>
		// Find the last dash to separate repo from branch
		lastDash := strings.LastIndex(worktreeDirName, "-")
		if lastDash == -1 {
			return fmt.Errorf("could not determine repository name from worktree directory")
		}

		repoName := worktreeDirName[:lastDash]
		targetRepo = filepath.Join(workspaceRoot, repoName)
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
