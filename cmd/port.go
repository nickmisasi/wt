package cmd

import (
	"fmt"

	"github.com/nickmisasi/wt/internal"
)

// RunPort displays the configured ports for the current worktree
func RunPort(config *internal.Config, gitRepo *internal.GitRepo) error {
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
