package main

import (
	"fmt"
	"os"
	"strings"

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

	if args[0] == "config" {
		return cmd.RunConfig(args[1:])
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
		jsonOutput, err := parseListArgs(args[1:])
		if err != nil {
			return err
		}
		return cmd.RunList(config, true, jsonOutput)

	case "co", "checkout":
		parsed, err := parseCheckoutCommandArgs(args[1:])
		if err != nil {
			return err
		}
		return cmd.RunCheckout(config, gitRepo, parsed.branch, parsed.baseBranch, parsed.noClaudeDocs, parsed.claudemux, parsed.jsonOutput, parsed.dryRun)

	case "rm", "remove":
		branch, force, jsonOutput, err := parseRemoveArgs(args[1:])
		if err != nil {
			return err
		}
		return cmd.RunRemove(config, branch, force, jsonOutput)

	case "clean":
		return cmd.RunClean(config)

	case "cursor":
		parsed, err := parseCheckoutCommandArgs(args[1:])
		if err != nil {
			return err
		}
		return cmd.RunCursor(config, gitRepo, parsed.branch, parsed.baseBranch, parsed.noClaudeDocs)

	case "edit":
		if len(args) < 2 {
			return cmd.RunEditHere()
		}
		parsed, err := parseCheckoutCommandArgs(args[1:])
		if err != nil {
			return err
		}
		return cmd.RunEdit(config, gitRepo, parsed.branch, parsed.baseBranch, parsed.noClaudeDocs)

	case "t", "toggle":
		return cmd.RunToggle()

	case "port":
		return cmd.RunPort(config, gitRepo)

	case "sessions":
		return cmd.RunSessions(config, args[1:])

	default:
		return fmt.Errorf("unknown command: %s\nRun 'wt help' for usage information", args[0])
	}
}

type checkoutCommandArgs struct {
	branch       string
	baseBranch   string
	noClaudeDocs bool
	claudemux    *bool
	jsonOutput   bool
	dryRun       bool
}

// parseCheckoutCommandArgs parses checkout command arguments into a struct.
func parseCheckoutCommandArgs(args []string) (checkoutCommandArgs, error) {
	var parsed checkoutCommandArgs

	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			switch a {
			case "-b", "--base":
				if i+1 >= len(args) {
					return parsed, fmt.Errorf("option %s requires a value", a)
				}
				parsed.baseBranch = args[i+1]
				i++
			case "-n", "--no-claude-docs":
				parsed.noClaudeDocs = true
			case "--claudemux":
				t := true
				parsed.claudemux = &t
			case "--no-claudemux":
				f := false
				parsed.claudemux = &f
			case "--json":
				parsed.jsonOutput = true
			case "--dry-run":
				parsed.dryRun = true
			default:
				return parsed, fmt.Errorf("unknown option: %s", a)
			}
		} else {
			if parsed.branch == "" {
				parsed.branch = a
			} else {
				return parsed, fmt.Errorf("unexpected argument: %s", a)
			}
		}
	}

	if parsed.branch == "" {
		return parsed, fmt.Errorf("usage: wt co <branch> [-b|--base <base-branch>] [-n|--no-claude-docs] [--claudemux|--no-claudemux] [--json] [--dry-run]")
	}

	return parsed, nil
}

// parseListArgs parses list command arguments.
func parseListArgs(args []string) (jsonOutput bool, err error) {
	for _, a := range args {
		switch a {
		case "--json":
			jsonOutput = true
		default:
			return false, fmt.Errorf("unknown option: %s", a)
		}
	}
	return jsonOutput, nil
}

// parseRemoveArgs parses branch, optional --force flag, and --json flag.
func parseRemoveArgs(args []string) (branch string, force bool, jsonOutput bool, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			switch a {
			case "-f", "--force":
				force = true
			case "--json":
				jsonOutput = true
			default:
				return "", false, false, fmt.Errorf("unknown option: %s", a)
			}
		} else {
			if branch == "" {
				branch = a
			} else {
				return "", false, false, fmt.Errorf("unexpected argument: %s", a)
			}
		}
	}
	if branch == "" {
		return "", false, false, fmt.Errorf("usage: wt rm <branch> [-f|--force] [--json]")
	}
	return branch, force, jsonOutput, nil
}
