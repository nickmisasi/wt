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

	// Resolve the base path for symlink-safe comparison (macOS /var -> /private/var)
	canonicalBase := config.WorktreeBasePath
	if resolved, err := filepath.EvalSymlinks(canonicalBase); err == nil {
		canonicalBase = resolved
	}

	isManaged := func(path string) bool {
		if strings.HasPrefix(path, config.WorktreeBasePath) {
			return true
		}
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			return strings.HasPrefix(resolved, canonicalBase)
		}
		return false
	}

	var worktrees []WorktreeInfo
	lines := strings.Split(string(output), "\n")

	var currentWorktree WorktreeInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if currentWorktree.Path != "" {
				// Check if this worktree is in our managed directory
				if isManaged(currentWorktree.Path) {
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
	if currentWorktree.Path != "" && isManaged(currentWorktree.Path) {
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

	// Check for sanitized path collisions with a different branch
	collision, collisionPath, err := FindCollidingWorktree(config, branch)
	if err != nil {
		return "", fmt.Errorf("failed to check for worktree collisions: %w", err)
	}
	if collision != nil {
		return "", fmt.Errorf("worktree path collision: branch %q already occupies %s (requested branch %q sanitizes to the same path)", collision.Branch, collisionPath, branch)
	}

	// Ensure the base directory exists
	if err := os.MkdirAll(config.WorktreeBasePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create worktree base directory: %w", err)
	}

	// Create the worktree
	var gitCmd *exec.Cmd
	if createBranch {
		// Create new branch from base branch
		if baseBranch != "" {
			gitCmd = exec.Command("git", "worktree", "add", "-b", branch, worktreePath, baseBranch)
		} else {
			gitCmd = exec.Command("git", "worktree", "add", "-b", branch, worktreePath)
		}
	} else {
		// Use existing branch
		gitCmd = exec.Command("git", "worktree", "add", worktreePath, branch)
	}

	output, err := gitCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create worktree: %s", string(output))
	}

	// Set push.autoSetupRemote for the new worktree
	autoSetupCmd := exec.Command("git", "-C", worktreePath, "config", "--local", "push.autoSetupRemote", "true")
	_ = autoSetupCmd.Run() // best-effort

	return worktreePath, nil
}

// WorktreeExists checks if a worktree already exists for the given branch.
// It verifies both the path existence and that the branch name matches,
// to avoid false positives from sanitized path collisions (e.g., feature/foo
// and feature-foo both map to the same directory).
func WorktreeExists(config *Config, branch string) (bool, string) {
	worktreePath := config.GetWorktreePath(branch)

	// Check if directory exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return false, ""
	}

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	canonicalTarget, _ := filepath.EvalSymlinks(worktreePath)
	if canonicalTarget == "" {
		canonicalTarget = worktreePath
	}

	// Verify it's actually a worktree for this branch by checking git worktree list
	worktrees, err := ListWorktrees(config)
	if err != nil {
		return false, ""
	}

	for _, wt := range worktrees {
		// Must match both path AND branch name to avoid sanitized-name collisions
		if wt.Branch != branch {
			continue
		}
		canonicalWt, _ := filepath.EvalSymlinks(wt.Path)
		if canonicalWt == "" {
			canonicalWt = wt.Path
		}
		if wt.Path == worktreePath || canonicalWt == canonicalTarget {
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
	// Kill associated claudemux session if it exists (best-effort)
	if branch := branchFromWorktreePath(path); branch != "" {
		sessionName := SanitizeBranchForTmux(branch)
		if HasSession(sessionName) {
			_ = KillSession(sessionName)
		}
	}

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

// branchFromWorktreePath uses git worktree list to find the branch for a worktree path.
func branchFromWorktreePath(path string) string {
	// Run from the target path itself so git can find the repo
	cmd := exec.Command("git", "-C", path, "worktree", "list", "--porcelain")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	canonicalPath := path
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		canonicalPath = resolved
	}

	var currentPath string
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")
		}
		if strings.HasPrefix(line, "branch ") {
			match := currentPath == path
			if !match {
				if resolved, err := filepath.EvalSymlinks(currentPath); err == nil {
					match = resolved == canonicalPath
				}
			}
			if match {
				ref := strings.TrimPrefix(line, "branch ")
				return strings.TrimPrefix(ref, "refs/heads/")
			}
		}
	}
	return ""
}

// FindCollidingWorktree checks if a different branch already occupies the same
// sanitized worktree path. Returns the existing worktree info, the colliding
// path, and any error.
func FindCollidingWorktree(config *Config, branch string) (*WorktreeInfo, string, error) {
	targetPath := config.GetWorktreePath(branch)

	// Check if directory exists
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return nil, "", nil
	}

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	canonicalTarget := targetPath
	if resolved, err := filepath.EvalSymlinks(targetPath); err == nil {
		canonicalTarget = resolved
	}

	pathsMatch := func(a string) bool {
		if a == targetPath {
			return true
		}
		if resolved, err := filepath.EvalSymlinks(a); err == nil {
			return resolved == canonicalTarget
		}
		return false
	}

	// The directory exists — find which branch actually occupies it using porcelain output
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, "", fmt.Errorf("failed to list worktrees: %w", err)
	}

	var currentPath, currentBranch string
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")
			currentBranch = ""
		} else if strings.HasPrefix(line, "branch ") {
			ref := strings.TrimPrefix(line, "branch ")
			currentBranch = strings.TrimPrefix(ref, "refs/heads/")
		}
		if line == "" && pathsMatch(currentPath) && currentBranch != "" && currentBranch != branch {
			return &WorktreeInfo{
				Path:   currentPath,
				Branch: currentBranch,
			}, targetPath, nil
		}
	}
	// Check the last entry (porcelain output may not end with blank line)
	if pathsMatch(currentPath) && currentBranch != "" && currentBranch != branch {
		return &WorktreeInfo{
			Path:   currentPath,
			Branch: currentBranch,
		}, targetPath, nil
	}

	return nil, "", nil
}
