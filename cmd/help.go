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

MATTERMOST DUAL-REPO COMMANDS:
    co-mm, mm <branch> [opts]    Create Mattermost dual-repo worktree
    rm-mm <branch> [opts]        Remove Mattermost dual-repo worktree
    cursor-mm <branch> [opts]    Open Mattermost worktree in Cursor

OPTIONS:
    -b, --base <branch>         Base branch for new branches (defaults to main/master)
    -f, --force                 Force removal when using 'wt rm' or 'wt rm-mm'
    --port <port>               Server port for Mattermost (auto-increments by default)
    --metrics-port <port>       Metrics port for Mattermost (auto-increments by default)
    --delete-branch             Delete branches from repos when using 'wt rm-mm'

WORKTREE STORAGE:
    Standard worktrees: ~/workspace/worktrees/<repo-name>-<branch-name>/
    Mattermost dual-repo:
        ~/workspace/worktrees/mattermost-<branch-name>/
        ├── server/      (mattermost/mattermost worktree)
        └── enterprise/  (mattermost/enterprise worktree)

EXAMPLES:
    # Standard operations
    wt co MM-12345                      # Create worktree for branch
    wt co feature/new -b develop        # Create from base branch
    wt rm MM-12345                      # Remove worktree

    # Mattermost dual-repo operations
    wt co-mm MM-12345                   # Create dual worktree (auto ports)
    wt mm MM-12345 -b master            # Create from master branch
    wt co-mm MM-12345 --port 8070       # Create with custom ports
    wt rm-mm MM-12345                   # Remove dual worktree
    wt rm-mm MM-12345 --delete-branch   # Remove and delete branches from both repos
    wt cursor-mm MM-12345               # Open in Cursor

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
