package cmd

import (
	"fmt"
	"strings"

	"github.com/nickmisasi/wt/internal"
)

// RunRemove removes a worktree for the given branch. When force is true, uses git -f
func RunRemove(config interface{}, branch string, force bool) error {
	cfg, ok := config.(*internal.Config)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	if strings.TrimSpace(branch) == "" {
		return fmt.Errorf("usage: wt rm <branch> [-f|--force]")
	}

	// Check if this is a Mattermost dual-repo worktree
	mc, err := internal.NewMattermostConfig()
	if err == nil {
		worktreePath := mc.GetMattermostWorktreePath(branch)
		if internal.IsMattermostDualWorktree(worktreePath) {
			return runMattermostRemove(mc, branch, force)
		}
	}

	// Standard worktree removal
	return runStandardRemove(cfg, branch, force)
}

// runStandardRemove handles standard single-repo worktree removal
func runStandardRemove(cfg *internal.Config, branch string, force bool) error {
	wt, err := internal.GetWorktreeByBranch(cfg, branch)
	if err != nil {
		return fmt.Errorf("worktree not found for branch: %s", branch)
	}

	fmt.Printf("Removing worktree for branch '%s' at %s\n", wt.Branch, wt.Path)
	if force {
		fmt.Println("Using --force (-f)")
	}

	if err := internal.RemoveWorktreeWithForce(wt.Path, force); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	fmt.Println("✓ Worktree removed")
	return nil
}

// runMattermostRemove handles Mattermost dual-repo worktree removal
func runMattermostRemove(mc *internal.MattermostConfig, branch string, force bool) error {
	worktreePath := mc.GetMattermostWorktreePath(branch)
	sanitizedBranch := internal.SanitizeBranchName(branch)

	// Show what will be removed
	fmt.Printf("\nRemoving Mattermost dual-repo worktree:\n")
	fmt.Printf("  - Mattermost worktree: %s/mattermost-%s/\n", worktreePath, sanitizedBranch)
	fmt.Printf("  - Enterprise worktree: %s/enterprise-%s/\n", worktreePath, sanitizedBranch)
	fmt.Printf("  - Directory: %s\n", worktreePath)
	if force {
		fmt.Println("Using --force (-f)")
	}
	fmt.Println()

	// Remove the worktree
	if err := internal.RemoveMattermostDualWorktree(mc, branch, force); err != nil {
		return err
	}

	fmt.Println("✓ Mattermost worktree removed")
	return nil
}
