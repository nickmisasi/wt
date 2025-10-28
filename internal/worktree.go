package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// WorktreeInfo contains information about a worktree
type WorktreeInfo struct {
	Path       string
	Branch     string
	IsDirty    bool
	LastCommit time.Time
}

// ListWorktrees returns all worktrees for the current repository
func ListWorktrees(config *Config) ([]WorktreeInfo, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []WorktreeInfo
	lines := strings.Split(string(output), "\n")

	var currentWorktree WorktreeInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if currentWorktree.Path != "" {
				// Check if this worktree is in our managed directory
				if strings.HasPrefix(currentWorktree.Path, config.WorktreeBasePath) {
					worktrees = append(worktrees, currentWorktree)
				}
				currentWorktree = WorktreeInfo{}
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			currentWorktree.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "branch ") {
			branch := strings.TrimPrefix(line, "branch ")
			// Remove refs/heads/ prefix
			branch = strings.TrimPrefix(branch, "refs/heads/")
			currentWorktree.Branch = branch
		}
	}

	// Don't forget the last one
	if currentWorktree.Path != "" && strings.HasPrefix(currentWorktree.Path, config.WorktreeBasePath) {
		worktrees = append(worktrees, currentWorktree)
	}

	// Check dirty status and last commit for each worktree
	for i := range worktrees {
		worktrees[i].IsDirty = isWorktreeDirty(worktrees[i].Path)
		worktrees[i].LastCommit = getLastCommitTime(worktrees[i].Path)
	}

	return worktrees, nil
}

// isWorktreeDirty checks if a worktree has uncommitted changes
func isWorktreeDirty(path string) bool {
	cmd := exec.Command("git", "-C", path, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

// getLastCommitTime returns the timestamp of the last commit in a worktree
func getLastCommitTime(path string) time.Time {
	cmd := exec.Command("git", "-C", path, "log", "-1", "--format=%ct")
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}
	}

	timestamp := strings.TrimSpace(string(output))
	var unixTime int64
	fmt.Sscanf(timestamp, "%d", &unixTime)
	return time.Unix(unixTime, 0)
}

// CreateWorktree creates a new worktree for the given branch
func CreateWorktree(config *Config, branch string, createBranch bool, baseBranch string) (string, error) {
	worktreePath := config.GetWorktreePath(branch)

	// Ensure the base directory exists
	if err := os.MkdirAll(config.WorktreeBasePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create worktree base directory: %w", err)
	}

	// Create the worktree
	var cmd *exec.Cmd
	if createBranch {
		// Create new branch from base branch
		if baseBranch != "" {
			cmd = exec.Command("git", "worktree", "add", "-b", branch, worktreePath, baseBranch)
		} else {
			cmd = exec.Command("git", "worktree", "add", "-b", branch, worktreePath)
		}
	} else {
		// Use existing branch
		cmd = exec.Command("git", "worktree", "add", worktreePath, branch)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create worktree: %s", string(output))
	}

	return worktreePath, nil
}

// WorktreeExists checks if a worktree already exists for the given branch
func WorktreeExists(config *Config, branch string) (bool, string) {
	worktreePath := config.GetWorktreePath(branch)

	// Check if directory exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return false, ""
	}

	// Verify it's actually a worktree by checking git worktree list
	worktrees, err := ListWorktrees(config)
	if err != nil {
		return false, ""
	}

	for _, wt := range worktrees {
		if wt.Path == worktreePath {
			return true, worktreePath
		}
	}

	return false, ""
}

// RemoveWorktree removes a worktree
func RemoveWorktree(path string) error {
	return RemoveWorktreeWithForce(path, false)
}

// RemoveWorktreeWithForce removes a worktree; when force is true it passes -f to git
func RemoveWorktreeWithForce(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, path)
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %s", string(output))
	}
	return nil
}

// GetWorktreeByBranch finds a worktree by branch name
func GetWorktreeByBranch(config *Config, branch string) (*WorktreeInfo, error) {
	worktrees, err := ListWorktrees(config)
	if err != nil {
		return nil, err
	}

	for _, wt := range worktrees {
		if wt.Branch == branch {
			return &wt, nil
		}
	}

	return nil, fmt.Errorf("worktree not found for branch: %s", branch)
}

// GetBranchNameFromWorktreePath extracts the branch name from a worktree path
func GetBranchNameFromWorktreePath(config *Config, path string) string {
	// Get the directory name
	dirName := filepath.Base(path)

	// Strip the repo prefix
	return config.StripRepoPrefix(dirName)
}
