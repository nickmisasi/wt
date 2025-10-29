package cmd

import (
	"fmt"
	"path/filepath"

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

	// Check if this is the mattermost repository
	if internal.IsMattermostRepo(repo) {
		// Use Mattermost dual-repo workflow
		return runMattermostCheckout(repo, branch, baseBranch, 0, 0)
	}

	// Standard worktree workflow
	return runStandardCheckout(cfg, repo, branch, baseBranch)
}

// runStandardCheckout handles standard single-repo worktree creation
func runStandardCheckout(cfg *internal.Config, repo *internal.GitRepo, branch string, baseBranch string) error {
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

	// Check if there's a post-setup command for this repo
	if postCmd := cfg.GetPostSetupCommand(worktreePath); postCmd != "" {
		fmt.Printf("%s%s\n", internal.CMDMarker, postCmd)
	}

	return nil
}

// runMattermostCheckout handles Mattermost dual-repo worktree creation
func runMattermostCheckout(repo *internal.GitRepo, branch string, baseBranch string, serverPort, metricsPort int) error {
	// Create Mattermost config
	mc, err := internal.NewMattermostConfig()
	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	// Validate setup
	if err := mc.ValidateMattermostSetup(); err != nil {
		return err
	}

	// Determine target directory based on current repo
	// If in mattermost repo, go to mattermost-<branch>/ subdirectory
	// If in enterprise repo, go to enterprise-<branch>/ subdirectory
	worktreePath := mc.GetMattermostWorktreePath(branch)
	sanitizedBranch := internal.SanitizeBranchName(branch)
	targetPath := worktreePath
	
	if repo.Root == mc.MattermostPath {
		targetPath = filepath.Join(worktreePath, "mattermost-"+sanitizedBranch)
	} else if repo.Root == mc.EnterprisePath {
		targetPath = filepath.Join(worktreePath, "enterprise-"+sanitizedBranch)
	}

	// Check if worktree already exists
	if internal.IsMattermostDualWorktree(worktreePath) {
		// Worktree exists and is valid, just switch to it
		fmt.Printf("Switching to existing Mattermost worktree for branch: %s\n", branch)
		fmt.Printf("%s%s\n", internal.CDMarker, targetPath)
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

		// Fallback to defaults (start at 8066, reserving 8065 for main repo)
		if serverPort == 0 {
			serverPort = 8066
		}
		if metricsPort == 0 {
			metricsPort = 8068
		}
	}

	mc.ServerPort = serverPort
	mc.MetricsPort = metricsPort

	// Create the dual-repo worktree
	fmt.Printf("Creating Mattermost dual-repo worktree for branch: %s\n", branch)
	fmt.Println("(Detected mattermost repository - creating unified worktree with enterprise)")
	createdPath, err := internal.CreateMattermostDualWorktree(mc, branch, baseBranch)
	if err != nil {
		return err
	}

	fmt.Printf("\nSuccessfully created Mattermost dual-repo worktree!\n")
	fmt.Printf("\nDirectory structure:\n")
	fmt.Printf("  %s/\n", createdPath)
	fmt.Printf("  ├── mattermost-%s/  (mattermost worktree)\n", sanitizedBranch)
	fmt.Printf("  └── enterprise-%s/  (enterprise worktree)\n", sanitizedBranch)
	fmt.Printf("\nServer configured on:\n")
	fmt.Printf("  - Main server: http://localhost:%d\n", serverPort)
	fmt.Printf("  - Metrics:     http://localhost:%d/metrics\n", metricsPort)
	fmt.Printf("\n")

	// Output CD marker for shell integration (use intelligent target path)
	fmt.Printf("%s%s\n", internal.CDMarker, targetPath)

	// Run post-setup command
	postCmd := fmt.Sprintf("cd %s/mattermost-%s/server && make setup-go-work", createdPath, sanitizedBranch)
	fmt.Printf("%s%s\n", internal.CMDMarker, postCmd)

	return nil
}
