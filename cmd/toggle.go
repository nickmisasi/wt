package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nickmisasi/wt/internal"
)

// isUnderDir checks whether child is inside or equal to parent, using a
// trailing separator to avoid partial-name matches.
func isUnderDir(child, parent string) bool {
	c := filepath.Clean(child)
	p := filepath.Clean(parent)
	if c == p {
		return true
	}
	return strings.HasPrefix(c, p+string(filepath.Separator))
}

// RunToggle switches from worktree back to parent repository
func RunToggle() error {
	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	worktreesDir, err := internal.ResolveWorktreesPath()
	if err != nil {
		return fmt.Errorf("failed to resolve worktrees path: %w", err)
	}

	if !isUnderDir(cwd, worktreesDir) {
		return fmt.Errorf("not currently in a worktree directory")
	}

	// Determine which repository to return to
	var targetRepo string

	// Check if we're in a Mattermost dual-repo worktree
	if strings.Contains(cwd, "/mattermost-") {
		mattermostPath, mmErr := internal.ResolveMattermostPath()
		enterprisePath, entErr := internal.ResolveEnterprisePath()

		if strings.Contains(cwd, "/enterprise-") {
			if entErr != nil {
				return fmt.Errorf("failed to resolve enterprise path: %v", entErr)
			}
			targetRepo = enterprisePath
		} else if mmErr == nil {
			targetRepo = mattermostPath
		} else {
			return fmt.Errorf("failed to resolve mattermost paths: %v", mmErr)
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
