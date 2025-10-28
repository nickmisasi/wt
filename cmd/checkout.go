package cmd

import (
	"fmt"

	"github.com/nickmisasi/wt/internal"
)

// RunCheckout checks out or creates a worktree for the given branch
func RunCheckout(config interface{}, gitRepo interface{}, branch string, baseBranch string) error {
	cfg, ok := config.(*internal.Config)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	repo, ok := gitRepo.(*internal.GitRepo)
	if !ok {
		return fmt.Errorf("invalid git repo type")
	}

	// Check if worktree already exists
	exists, path := internal.WorktreeExists(cfg, branch)
	if exists {
		fmt.Printf("Switching to existing worktree for branch: %s\n", branch)
		fmt.Printf("%s%s\n", internal.CDMarker, path)
		return nil
	}

	// Worktree doesn't exist, check if branch exists
	branchExists, err := repo.BranchExists(branch)
	if err != nil {
		return fmt.Errorf("failed to check if branch exists: %w", err)
	}

	createNewBranch := false
	if !branchExists {
		// Check if branch exists on remote
		remoteBranchExists, err := repo.RemoteBranchExists(branch)
		if err != nil {
			return fmt.Errorf("failed to check remote branches: %w", err)
		}

		if remoteBranchExists {
			// Create tracking branch
			fmt.Printf("Creating local branch '%s' tracking 'origin/%s'...\n", branch, branch)
			err = repo.CreateTrackingBranch(branch)
			if err != nil {
				return fmt.Errorf("failed to create tracking branch: %w", err)
			}
		} else {
			// Branch doesn't exist anywhere, create it
			// If no base branch specified, use the default branch
			if baseBranch == "" {
				baseBranch = repo.GetDefaultBranch()
			}
			fmt.Printf("Creating new branch '%s' from '%s'\n", branch, baseBranch)
			createNewBranch = true
		}
	}

	// Create the worktree
	fmt.Printf("Creating worktree for branch: %s\n", branch)
	worktreePath, err := internal.CreateWorktree(cfg, branch, createNewBranch, baseBranch)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	fmt.Printf("Worktree created at: %s\n", worktreePath)
	fmt.Printf("%s%s\n", internal.CDMarker, worktreePath)

	return nil
}
