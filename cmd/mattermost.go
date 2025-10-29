package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nickmisasi/wt/internal"
)

// RunMattermostCheckout creates a Mattermost dual-repo worktree
func RunMattermostCheckout(branch string, baseBranch string, serverPort, metricsPort int) error {
	// Create Mattermost config
	mc, err := internal.NewMattermostConfig()
	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	// Validate setup
	if err := mc.ValidateMattermostSetup(); err != nil {
		return err
	}

	// Check if worktree already exists
	worktreePath := mc.GetMattermostWorktreePath(branch)
	if _, err := os.Stat(worktreePath); err == nil {
		// Worktree exists, just switch to it
		fmt.Printf("Switching to existing Mattermost worktree for branch: %s\n", branch)
		fmt.Printf("%s%s\n", internal.CDMarker, worktreePath)
		return nil
	}

	// Determine ports if not specified
	if serverPort == 0 || metricsPort == 0 {
		// Get existing worktrees to auto-increment ports
		config, _ := internal.NewConfig()
		if config != nil {
			worktrees, _ := internal.ListWorktrees(config)
			if worktrees != nil {
				autoServerPort, autoMetricsPort := internal.GetAvailablePorts(worktrees)
				if serverPort == 0 {
					serverPort = autoServerPort
				}
				if metricsPort == 0 {
					metricsPort = autoMetricsPort
				}
			}
		}
		
		// Fallback to defaults
		if serverPort == 0 {
			serverPort = 8065
		}
		if metricsPort == 0 {
			metricsPort = 8067
		}
	}

	mc.ServerPort = serverPort
	mc.MetricsPort = metricsPort

	// Create the dual-repo worktree
	fmt.Printf("Creating Mattermost dual-repo worktree for branch: %s\n", branch)
	createdPath, err := internal.CreateMattermostDualWorktree(mc, branch, baseBranch)
	if err != nil {
		return err
	}

	fmt.Printf("\nSuccessfully created Mattermost dual-repo worktree!\n")
	fmt.Printf("\nDirectory structure:\n")
	fmt.Printf("  %s/\n", createdPath)
	fmt.Printf("  ├── server/      (mattermost worktree)\n")
	fmt.Printf("  └── enterprise/  (enterprise worktree)\n")
	fmt.Printf("\nServer configured on:\n")
	fmt.Printf("  - Main server: http://localhost:%d\n", serverPort)
	fmt.Printf("  - Metrics:     http://localhost:%d/metrics\n", metricsPort)
	fmt.Printf("\n")

	// Output CD marker for shell integration
	fmt.Printf("%s%s\n", internal.CDMarker, createdPath)

	// Run post-setup command
	postCmd := fmt.Sprintf("cd %s/server && make setup-go-work", createdPath)
	fmt.Printf("%s%s\n", internal.CMDMarker, postCmd)

	return nil
}

// RunMattermostRemove removes a Mattermost dual-repo worktree
func RunMattermostRemove(branch string, force bool, deleteBranch bool) error {
	mc, err := internal.NewMattermostConfig()
	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	worktreePath := mc.GetMattermostWorktreePath(branch)

	// Check if it exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return fmt.Errorf("worktree not found for branch: %s", branch)
	}

	// Check if it's a Mattermost dual-repo worktree
	if !internal.IsMattermostDualWorktree(worktreePath) {
		return fmt.Errorf("not a Mattermost dual-repo worktree (use 'wt rm' for single-repo worktrees)")
	}

	// Show what will be removed
	fmt.Printf("\nThis will remove the following:\n")
	fmt.Printf("  - Mattermost worktree: %s/server/\n", worktreePath)
	fmt.Printf("  - Enterprise worktree: %s/enterprise/\n", worktreePath)
	fmt.Printf("  - Directory: %s\n", worktreePath)
	if deleteBranch {
		fmt.Printf("  - Branch '%s' from both repositories\n", branch)
	}
	fmt.Println()

	// Confirmation unless force mode
	if !force {
		fmt.Print("Are you sure you want to remove this worktree? [y/N]: ")
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
	}

	// Remove the worktree
	if err := internal.RemoveMattermostDualWorktree(mc, branch, force); err != nil {
		return err
	}

	fmt.Printf("\nSuccessfully removed Mattermost worktree for branch: %s\n", branch)

	// Delete branches if requested
	if deleteBranch {
		fmt.Println("\nDeleting branches from repositories...")
		if err := internal.DeleteBranchFromRepos(mc, branch); err != nil {
			fmt.Printf("Warning: %v\n", err)
		}
	}

	return nil
}

// RunMattermostCursor opens Cursor for a Mattermost dual-repo worktree
func RunMattermostCursor(branch string, baseBranch string, serverPort, metricsPort int) error {
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
		fmt.Printf("Worktree does not exist, creating it first...\n\n")
		if err := RunMattermostCheckout(branch, baseBranch, serverPort, metricsPort); err != nil {
			return err
		}
		// Refresh the worktree path
		worktreePath = mc.GetMattermostWorktreePath(branch)
	}

	// Open in Cursor
	fmt.Printf("Opening Cursor at: %s\n", worktreePath)
	cmd := fmt.Sprintf("cursor %s", worktreePath)
	fmt.Printf("%s%s\n", internal.CMDMarker, cmd)

	return nil
}

