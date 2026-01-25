package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/nickmisasi/wt/internal"
)

// RunCursor opens Cursor editor for the given branch's worktree
func RunCursor(config interface{}, gitRepo interface{}, branch string, baseBranch string, noClaudeDocs bool) error {
	cfg, ok := config.(*internal.Config)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	repo, ok := gitRepo.(*internal.GitRepo)
	if !ok {
		return fmt.Errorf("invalid git repo type")
	}

	// Check if Cursor CLI is available
	if _, err := exec.LookPath("cursor"); err != nil {
		return fmt.Errorf("cursor command not found. Please install Cursor CLI first")
	}

	// Check if this is the mattermost repository
	if internal.IsMattermostRepo(repo) {
		return runMattermostCursor(repo, branch, baseBranch, noClaudeDocs)
	}

	// Standard worktree cursor workflow
	return runStandardCursor(cfg, repo, branch, baseBranch, noClaudeDocs)
}

// runStandardCursor handles standard single-repo cursor opening
func runStandardCursor(cfg *internal.Config, repo *internal.GitRepo, branch string, baseBranch string, noClaudeDocs bool) error {
	// Check if worktree already exists
	exists, path := internal.WorktreeExists(cfg, branch)
	worktreeCreated := false

	if !exists {
		// Create the worktree first
		fmt.Printf("Worktree doesn't exist for branch '%s'. Creating it...\n", branch)

		// Check if branch exists, create tracking branch if needed
		branchExists, err := repo.BranchExists(branch)
		if err != nil {
			return fmt.Errorf("failed to check if branch exists: %w", err)
		}

		createNewBranch := false
		if !branchExists {
			remoteBranchExists, err := repo.RemoteBranchExists(branch)
			if err != nil {
				return fmt.Errorf("failed to check remote branches: %w", err)
			}

			if remoteBranchExists {
				fmt.Printf("Creating local branch '%s' tracking 'origin/%s'...\n", branch, branch)
				err = repo.CreateTrackingBranch(branch)
				if err != nil {
					return fmt.Errorf("failed to create tracking branch: %w", err)
				}
			} else {
				// If no base branch specified, use the default branch
				if baseBranch == "" {
					baseBranch = repo.GetDefaultBranch()
				}
				fmt.Printf("Creating new branch '%s' from '%s'\n", branch, baseBranch)
				createNewBranch = true
			}
		}

		// Create the worktree
		path, err = internal.CreateWorktree(cfg, branch, createNewBranch, baseBranch)
		if err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
		fmt.Printf("Worktree created at: %s\n", path)
		worktreeCreated = true
	}

	// Open Cursor
	fmt.Printf("Opening Cursor for branch: %s\n", branch)
	cmd := exec.Command("cursor", path)
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to open Cursor: %w", err)
	}

	// Optionally also switch directory
	fmt.Printf("%s%s\n", internal.CDMarker, path)

	// If we created a new worktree, check if there's a post-setup command
	if worktreeCreated {
		if postCmd := cfg.GetPostSetupCommand(path); postCmd != "" {
			fmt.Printf("%s%s\n", internal.CMDMarker, postCmd)
		}

		// Run enable-claude-docs.sh if it exists and not disabled
		if !noClaudeDocs {
			emitEnableClaudeDocsCommand(path)
		}
	}

	return nil
}

// runMattermostCursor handles Mattermost dual-repo cursor opening
func runMattermostCursor(repo *internal.GitRepo, branch string, baseBranch string, noClaudeDocs bool) error {
	mc, err := internal.NewMattermostConfig()
	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	// Validate setup
	if err := mc.ValidateMattermostSetup(); err != nil {
		return err
	}

	worktreePath := mc.GetMattermostWorktreePath(branch)

	// Check if worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		// Create it first
		fmt.Printf("Worktree doesn't exist for branch '%s'. Creating it...\n\n", branch)
		if err := runMattermostCheckout(repo, branch, baseBranch, 0, 0, noClaudeDocs); err != nil {
			return err
		}
		// Refresh the worktree path
		worktreePath = mc.GetMattermostWorktreePath(branch)
	}

	// Open in Cursor
	fmt.Printf("Opening Cursor for branch: %s\n", branch)

	cmd := exec.Command("cursor", worktreePath)
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to open Cursor: %w", err)
	}

	// Switch directory
	fmt.Printf("%s%s\n", internal.CDMarker, worktreePath)

	return nil
}
