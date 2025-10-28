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
			return fmt.Errorf("usage: wt co <branch>")
		}
		return cmd.RunCheckout(config, gitRepo, args[1])

	case "clean":
		return cmd.RunClean(config)

	case "cursor":
		if len(args) < 2 {
			return fmt.Errorf("usage: wt cursor <branch>")
		}
		return cmd.RunCursor(config, gitRepo, args[1])

	default:
		return fmt.Errorf("unknown command: %s\nRun 'wt help' for usage information", args[0])
	}
}

