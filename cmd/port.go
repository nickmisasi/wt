package cmd

import (
	"fmt"

	"github.com/nickmisasi/wt/internal"
)

// RunPort displays the configured ports for the current worktree, or for all
// worktrees when all is true
func RunPort(config *internal.Config, gitRepo *internal.GitRepo, all bool) error {
	if all {
		return runPortAll(config)
	}

	// 1. Identify if we are in a Mattermost worktree
	_, configPath, err := internal.FindMattermostConfig(gitRepo.Root)
	if err != nil {
		return err
	}

	// 2. Get the ports
	portPair := internal.ExtractPortPairFromConfig(configPath)
	if portPair.ServerPort == 0 {
		return fmt.Errorf("failed to extract server port from %s", configPath)
	}

	fmt.Printf("Server Port:  %d\n", portPair.ServerPort)
	if portPair.MetricsPort > 0 {
		fmt.Printf("Metrics Port: %d\n", portPair.MetricsPort)
	}
	fmt.Printf("Site URL:     http://localhost:%d\n", portPair.ServerPort)

	return nil
}

// runPortAll displays the configured ports for every open worktree
func runPortAll(config *internal.Config) error {
	worktrees, err := internal.ListWorktrees(config)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees found for this repository.")
		return nil
	}

	for _, wt := range worktrees {
		_, configPath, err := internal.FindMattermostConfig(wt.Path)
		if err != nil {
			continue
		}

		portPair := internal.ExtractPortPairFromConfig(configPath)
		if portPair.ServerPort == 0 {
			continue
		}

		fmt.Printf("%-30s  Server: %-6d", wt.Branch, portPair.ServerPort)
		if portPair.MetricsPort > 0 {
			fmt.Printf("  Metrics: %d", portPair.MetricsPort)
		}
		fmt.Println()
	}

	return nil
}
