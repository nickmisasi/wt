package cmd

import (
	"fmt"
	"time"

	"github.com/nickmisasi/wt/internal"
)

// RunList lists all worktrees for the current repository
func RunList(config interface{}, showHeader bool) error {
	cfg, ok := config.(*internal.Config)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	worktrees, err := internal.ListWorktrees(cfg)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees found for this repository.")
		return nil
	}

	if showHeader {
		fmt.Printf("\nWorktrees for %s:\n", cfg.RepoName)
		fmt.Println("=" + repeat("=", len(cfg.RepoName)+15))
	}

	for _, wt := range worktrees {
		branch := wt.Branch
		status := "clean"
		if wt.IsDirty {
			status = "dirty"
		}

		// Calculate days since last commit
		daysSince := int(time.Since(wt.LastCommit).Hours() / 24)
		lastCommitStr := fmt.Sprintf("%d days ago", daysSince)
		if daysSince == 0 {
			lastCommitStr = "today"
		} else if daysSince == 1 {
			lastCommitStr = "yesterday"
		}

		fmt.Printf("  %-30s  [%s]  (last commit: %s)\n", branch, status, lastCommitStr)
	}

	return nil
}

// repeat returns a string with character c repeated n times
func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

