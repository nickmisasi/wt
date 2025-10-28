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
