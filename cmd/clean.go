package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nickmisasi/wt/internal"
)

const (
	DefaultStaleDays    = 30
	DefaultCleanBatch   = 10
	defaultCleanWorkers = 10
)

// RunClean removes stale worktrees (clean and older than threshold days).
// Processes in batches for responsiveness on large worktree counts.
// When confirm is false, skips interactive prompt (for programmatic use).
func RunClean(config interface{}, days int, batchSize int, includeDirty bool, confirm bool) error {
	cfg, ok := config.(*internal.Config)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	worktrees, err := internal.ListWorktreePaths(cfg)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees found for this repository.")
		return nil
	}

	fmt.Printf("Scanning %d worktrees (removing clean worktrees older than %d days)...\n\n", len(worktrees), days)

	if confirm {
		fmt.Print("Proceed? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		if r := strings.TrimSpace(strings.ToLower(response)); r != "y" && r != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
		fmt.Println()
	}

	totalRemoved := 0
	totalSkipped := 0

	for i := 0; i < len(worktrees); i += batchSize {
		end := i + batchSize
		if end > len(worktrees) {
			end = len(worktrees)
		}
		batch := worktrees[i:end]

		internal.FillWorktreeStatus(batch, defaultCleanWorkers)

		for j := range batch {
			wt := &batch[j]
			daysSince := int(time.Since(wt.LastCommit).Hours() / 24)

			if (!includeDirty && wt.IsDirty) || daysSince < days {
				totalSkipped++
				continue
			}

			fmt.Printf("  Removing %s (%d days old)...", wt.Branch, daysSince)
			if err := internal.RemoveWorktree(wt.Path); err != nil {
				internal.FixWorktreePermissions(wt.Path)
				if err := internal.RemoveWorktreeWithForce(wt.Path, true); err != nil {
					fmt.Fprintf(os.Stderr, " FAILED: %v\n", err)
					continue
				}
			}
			fmt.Println(" done")
			totalRemoved++
		}
	}

	fmt.Printf("\nDone. Removed %d, skipped %d (dirty or recent).\n", totalRemoved, totalSkipped)
	return nil
}
