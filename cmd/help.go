package cmd

import (
	"fmt"
	"os"
)

const helpText = `wt - Git Worktree Manager

USAGE:
    wt [command] [arguments]

COMMANDS:
    (no args)         Show this help and list worktrees for current repository
    ls                List all worktrees for current repository
    co <branch>       Checkout/create worktree for branch and switch to it
    clean             Remove stale worktrees (clean, >30 days old)
    cursor <branch>   Open Cursor editor for branch's worktree
    install           Install shell integration and completions
    help              Show this help message

EXAMPLES:
    # List worktrees
    wt ls

    # Switch to or create worktree for branch MM-123
    wt co MM-123

    # Open Cursor for branch feature/new-ui
    wt cursor feature/new-ui

    # Clean up old worktrees
    wt clean

    # Install shell function and completions
    wt install

WORKTREE STORAGE:
    Worktrees are stored in: ~/workspace/worktrees/
    Format: <repo-name>-<branch-name>/

INSTALLATION:
    After building, run 'wt install' to set up shell integration and completions.
    This adds a shell function to ~/.zshrc that enables automatic directory switching.
`

// RunHelp displays the help text
func RunHelp() error {
	fmt.Print(helpText)
	return nil
}

// RunDefault shows help and lists worktrees
func RunDefault(config interface{}) error {
	fmt.Print(helpText)
	fmt.Println()
	
	// Try to list worktrees if we're in a git repo
	err := RunList(config, false)
	if err != nil {
		// If we're not in a git repo, that's okay for default command
		fmt.Fprintf(os.Stderr, "\n(Run this command from inside a git repository to see worktrees)\n")
	}
	
	return nil
}

