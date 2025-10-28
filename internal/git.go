package internal

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitRepo represents a git repository with operations
type GitRepo struct {
	Root string
	Name string
}

// NewGitRepo creates a new GitRepo instance for the current directory
func NewGitRepo() (*GitRepo, error) {
	// Get repository root
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("not a git repository (or any parent up to mount point)")
	}

	root := strings.TrimSpace(string(output))

	// Try to get repo name from remote URL first
	name, err := getRepoNameFromRemote()
	if err != nil || name == "" {
		// Fall back to directory name
		name = filepath.Base(root)
	}

	return &GitRepo{
		Root: root,
		Name: name,
	}, nil
}

// getRepoNameFromRemote attempts to extract the repository name from the remote URL
func getRepoNameFromRemote() (string, error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	url := strings.TrimSpace(string(output))
	if url == "" {
		return "", fmt.Errorf("no remote URL")
	}

	// Extract repo name from URL
	// Handle formats like:
	// - git@github.com:user/repo.git
	// - https://github.com/user/repo.git
	// - https://github.com/user/repo

	// Remove .git suffix if present
	url = strings.TrimSuffix(url, ".git")

	// Get the last part of the path
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		name := parts[len(parts)-1]
		// Also handle SSH format
		if strings.Contains(name, ":") {
			parts = strings.Split(name, ":")
			if len(parts) > 1 {
				return parts[len(parts)-1], nil
			}
		}
		return name, nil
	}

	return "", fmt.Errorf("could not parse repo name from URL")
}

// BranchExists checks if a branch exists locally
func (g *GitRepo) BranchExists(branch string) (bool, error) {
	cmd := exec.Command("git", "branch", "--list", branch)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(output)) != "", nil
}

// RemoteBranchExists checks if a branch exists on the remote
func (g *GitRepo) RemoteBranchExists(branch string) (bool, error) {
	cmd := exec.Command("git", "branch", "-r", "--list", "origin/"+branch)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(output)) != "", nil
}

// CreateTrackingBranch creates a local branch tracking a remote branch
func (g *GitRepo) CreateTrackingBranch(branch string) error {
	cmd := exec.Command("git", "branch", "--track", branch, "origin/"+branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create tracking branch: %s", string(output))
	}
	return nil
}

// ListBranches returns all local branches
func (g *GitRepo) ListBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	branches := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, b := range branches {
		b = strings.TrimSpace(b)
		if b != "" {
			result = append(result, b)
		}
	}
	return result, nil
}

// ListRemoteBranches returns all remote branches (without origin/ prefix)
func (g *GitRepo) ListRemoteBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "-r", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	branches := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, b := range branches {
		b = strings.TrimSpace(b)
		if b != "" && strings.HasPrefix(b, "origin/") {
			// Remove origin/ prefix and skip HEAD
			branch := strings.TrimPrefix(b, "origin/")
			if branch != "HEAD" && !strings.Contains(branch, "->") {
				result = append(result, branch)
			}
		}
	}
	return result, nil
}

// GetDefaultBranch returns the default branch (main, master, or current branch)
func (g *GitRepo) GetDefaultBranch() string {
	// Try to get the default branch from remote
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	output, err := cmd.Output()
	if err == nil {
		branch := strings.TrimSpace(string(output))
		branch = strings.TrimPrefix(branch, "refs/remotes/origin/")
		if branch != "" {
			return branch
		}
	}

	// Fall back to checking if main or master exists
	if exists, _ := g.BranchExists("main"); exists {
		return "main"
	}
	if exists, _ := g.BranchExists("master"); exists {
		return "master"
	}

	// Last resort: get current branch
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err = cmd.Output()
	if err == nil {
		branch := strings.TrimSpace(string(output))
		if branch != "" && branch != "HEAD" {
			return branch
		}
	}

	return "main" // Ultimate fallback
}

// BranchExistsAnywhere checks if a branch exists locally or remotely
func (g *GitRepo) BranchExistsAnywhere(branch string) (local bool, remote bool, err error) {
	local, err = g.BranchExists(branch)
	if err != nil {
		return false, false, err
	}

	remote, err = g.RemoteBranchExists(branch)
	if err != nil {
		return local, false, err
	}

	return local, remote, nil
}
