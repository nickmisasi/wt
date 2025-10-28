package main

import (
	"fmt"
	"os"

	"github.com/nickmisasi/wt/cmd"
	"github.com/nickmisasi/wt/internal"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]

	// Handle commands that don't require git repo
	if len(args) == 0 {
		return cmd.RunDefault(nil)
	}

	if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		return cmd.RunHelp()
	}

	if args[0] == "install" {
		return cmd.RunInstall()
	}

	// For all other commands, we need to be in a git repo
	gitRepo, err := internal.NewGitRepo()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	config, err := internal.NewConfig()
	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}
	config.RepoName = gitRepo.Name
	config.RepoRoot = gitRepo.Root

	// Route commands
	switch args[0] {
	case "ls", "list":
		return cmd.RunList(config, true)

	case "co", "checkout":
		if len(args) < 2 {
			return fmt.Errorf("usage: wt co <branch> [-b|--base <base-branch>]")
		}
		branch, baseBranch := parseCheckoutArgs(args[1:])
		return cmd.RunCheckout(config, gitRepo, branch, baseBranch)

	case "clean":
		return cmd.RunClean(config)

	case "cursor":
		if len(args) < 2 {
			return fmt.Errorf("usage: wt cursor <branch> [-b|--base <base-branch>]")
		}
		branch, baseBranch := parseCheckoutArgs(args[1:])
		return cmd.RunCursor(config, gitRepo, branch, baseBranch)

	default:
		return fmt.Errorf("unknown command: %s\nRun 'wt help' for usage information", args[0])
	}
}

// parseCheckoutArgs parses branch and optional base branch from command arguments
func parseCheckoutArgs(args []string) (branch string, baseBranch string) {
	if len(args) == 0 {
		return "", ""
	}

	branch = args[0]
	baseBranch = ""

	// Look for -b or --base flag
	for i := 1; i < len(args); i++ {
		if (args[i] == "-b" || args[i] == "--base") && i+1 < len(args) {
			baseBranch = args[i+1]
			break
		}
	}

	return branch, baseBranch
}
