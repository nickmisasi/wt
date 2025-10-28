package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nickmisasi/wt/internal"
)

const staleDays = 30

// RunClean removes stale worktrees (clean and older than 30 days)
func RunClean(config interface{}) error {
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

	// Find worktrees that qualify for removal
	var staleWorktrees []internal.WorktreeInfo
	for _, wt := range worktrees {
		// Skip if it has uncommitted changes
		if wt.IsDirty {
			continue
		}

		// Check if last commit is older than staleDays
		daysSince := int(time.Since(wt.LastCommit).Hours() / 24)
		if daysSince >= staleDays {
			staleWorktrees = append(staleWorktrees, wt)
		}
	}

	if len(staleWorktrees) == 0 {
		fmt.Println("No stale worktrees found (clean and >30 days old).")
		return nil
	}

	// Display worktrees that will be removed
	fmt.Printf("Found %d stale worktree(s) to remove:\n\n", len(staleWorktrees))
	for _, wt := range staleWorktrees {
		daysSince := int(time.Since(wt.LastCommit).Hours() / 24)
		fmt.Printf("  • %s (last commit: %d days ago)\n", wt.Branch, daysSince)
	}

	// Ask for confirmation
	fmt.Print("\nDo you want to remove these worktrees? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Aborted.")
		return nil
	}

	// Remove the worktrees
	fmt.Println()
	removed := 0
	for _, wt := range staleWorktrees {
		fmt.Printf("Removing worktree: %s...\n", wt.Branch)
		err := internal.RemoveWorktree(wt.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ Failed to remove %s: %v\n", wt.Branch, err)
		} else {
			fmt.Printf("  ✓ Removed %s\n", wt.Branch)
			removed++
		}
	}

	fmt.Printf("\nRemoved %d worktree(s).\n", removed)
	return nil
}

