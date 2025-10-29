package cmd

import (
	"fmt"
	"os"
)

const helpText = `wt - Git Worktree Manager

USAGE:
    wt [command] [arguments]

COMMANDS:
    (no args)                    Show this help and list worktrees for current repository
    ls                           List all worktrees for current repository
    co <branch> [-b <base>]      Checkout/create worktree for branch and switch to it
    rm <branch> [-f]             Remove a worktree for branch (use -f to force)
    clean                        Remove stale worktrees (clean, >30 days old)
    cursor <branch> [-b <base>]  Open Cursor editor for branch's worktree
    install                      Install shell integration and completions
    help                         Show this help message

OPTIONS:
    -b, --base <branch>         Base branch for new branches (defaults to main/master)
    -f, --force                 Force removal when using 'wt rm'

WORKTREE STORAGE:
    Standard worktrees: ~/workspace/worktrees/<repo-name>-<branch-name>/

MATTERMOST DUAL-REPOSITORY SUPPORT:
    When working in the mattermost repository (~/workspace/mattermost), wt automatically
    creates dual-repo worktrees that include both mattermost and enterprise repositories:

        ~/workspace/worktrees/mattermost-<branch-name>/
        ├── server/      (mattermost/mattermost worktree)
        └── enterprise/  (mattermost/enterprise worktree)

    The tool automatically:
    - Detects when you're in the mattermost repository
    - Creates worktrees in both repositories for the same branch
    - Copies base configuration files (CLAUDE.md, mise.toml, etc.)
    - Updates config.json with auto-incremented ports (starting from 8065)
    - Runs 'make setup-go-work' after creation

    Requirements:
    - ~/workspace/mattermost (mattermost/mattermost repo)
    - ~/workspace/enterprise (mattermost/enterprise repo)

EXAMPLES:
    # Standard repository
    cd ~/workspace/my-project
    wt co feature-123            # Create worktree
    wt rm feature-123            # Remove worktree

    # Mattermost repository (automatic dual-repo)
    cd ~/workspace/mattermost
    wt co MM-12345               # Creates dual worktree with auto ports
    wt co MM-12345 -b master     # Create from master branch
    wt rm MM-12345               # Removes both worktrees
    wt cursor MM-12345           # Open in Cursor

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
