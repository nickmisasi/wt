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

	case "rm", "remove":
		if len(args) < 2 {
			return fmt.Errorf("usage: wt rm <branch> [-f|--force]")
		}
		branch, force := parseRemoveArgs(args[1:])
		return cmd.RunRemove(config, branch, force)

	case "clean":
		return cmd.RunClean(config)

	case "cursor":
		if len(args) < 2 {
			return fmt.Errorf("usage: wt cursor <branch> [-b|--base <base-branch>]")
		}
		branch, baseBranch := parseCheckoutArgs(args[1:])
		return cmd.RunCursor(config, gitRepo, branch, baseBranch)

	case "co-mm", "mm", "mattermost":
		if len(args) < 2 {
			return fmt.Errorf("usage: wt co-mm <branch> [-b|--base <base-branch>] [--port <port>] [--metrics-port <port>]")
		}
		branch, baseBranch, serverPort, metricsPort := parseMattermostArgs(args[1:])
		return cmd.RunMattermostCheckout(branch, baseBranch, serverPort, metricsPort)

	case "rm-mm":
		if len(args) < 2 {
			return fmt.Errorf("usage: wt rm-mm <branch> [-f|--force] [--delete-branch]")
		}
		branch, force, deleteBranch := parseMattermostRemoveArgs(args[1:])
		return cmd.RunMattermostRemove(branch, force, deleteBranch)

	case "cursor-mm":
		if len(args) < 2 {
			return fmt.Errorf("usage: wt cursor-mm <branch> [-b|--base <base-branch>] [--port <port>] [--metrics-port <port>]")
		}
		branch, baseBranch, serverPort, metricsPort := parseMattermostArgs(args[1:])
		return cmd.RunMattermostCursor(branch, baseBranch, serverPort, metricsPort)

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

// parseRemoveArgs parses branch and optional --force flag
func parseRemoveArgs(args []string) (branch string, force bool) {
	branch = ""
	force = false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-f" || a == "--force" {
			force = true
			continue
		}
		if branch == "" {
			branch = a
		}
	}
	return branch, force
}

// parseMattermostArgs parses Mattermost command arguments
func parseMattermostArgs(args []string) (branch string, baseBranch string, serverPort, metricsPort int) {
	if len(args) == 0 {
		return "", "", 0, 0
	}

	branch = args[0]
	baseBranch = ""
	serverPort = 0
	metricsPort = 0

	for i := 1; i < len(args); i++ {
		if (args[i] == "-b" || args[i] == "--base") && i+1 < len(args) {
			baseBranch = args[i+1]
			i++
		} else if args[i] == "--port" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &serverPort)
			i++
		} else if args[i] == "--metrics-port" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &metricsPort)
			i++
		}
	}

	return branch, baseBranch, serverPort, metricsPort
}

// parseMattermostRemoveArgs parses Mattermost remove command arguments
func parseMattermostRemoveArgs(args []string) (branch string, force bool, deleteBranch bool) {
	branch = ""
	force = false
	deleteBranch = false

	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-f" || a == "--force" {
			force = true
			continue
		}
		if a == "--delete-branch" || a == "--delete-branches" {
			deleteBranch = true
			continue
		}
		if branch == "" {
			branch = a
		}
	}
	return branch, force, deleteBranch
}
