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
    co <branch> [-b <base>] [-n] Checkout/create worktree for branch and switch to it
    rm <branch> [-f]             Remove a worktree for branch (use -f to force)
    clean                        Remove stale worktrees (clean, >30 days old)
    edit [<branch>] [-b <base>] [-n] Open configured editor (current worktree if no branch)
    cursor                           (deprecated) Alias for 'edit'
    port                         Show current worktree's mapped ports
    t, toggle                    Return to parent repository from worktree
    config                       Manage configuration (get/set/show)
    install                      Install shell integration and completions
    help                         Show this help message

OPTIONS:
    -b, --base <branch>         Base branch for new branches (defaults to main/master)
    -f, --force                 Force removal when using 'wt rm'
    -n, --no-claude-docs        Skip running enable-claude-docs.sh after worktree creation

WORKTREE STORAGE:
    Standard worktrees: <worktrees.path>/<repo-name>-<branch-name>/
    worktrees.path defaults to <workspace.root>/worktrees (configurable via 'wt config')

MATTERMOST DUAL-REPOSITORY SUPPORT:
    When working in the mattermost repository, wt automatically creates dual-repo
    worktrees that include both mattermost and enterprise repositories:

        <worktrees.path>/mattermost-<branch-name>/
        ├── mattermost-<branch-name>/  (mattermost/mattermost worktree)
        ├── enterprise-<branch-name>/  (mattermost/enterprise worktree)
        ├── mattermost -> mattermost-<branch-name>/  (symlink for scripts)
        └── enterprise -> enterprise-<branch-name>/  (symlink for scripts)

    The tool automatically:
    - Detects when you're in the mattermost repository
    - Creates worktrees in both repositories for the same branch
    - Copies base configuration files (CLAUDE.md, mise.toml, etc.)
    - Updates config.json with auto-incremented ports (starting from 8065)
    - Runs 'make setup-go-work' after creation

    Requirements (paths configurable via 'wt config'):
    - mattermost/mattermost repo  (default: <workspace.root>/mattermost)
    - mattermost/enterprise repo  (default: <workspace.root>/enterprise)

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
    wt edit MM-12345             # Open in configured editor
    wt port                      # Show server ports

    # Navigation
    wt t                         # Return to parent repository from worktree

CONFIGURATION:
    wt config show              Show all configuration values (JSON)
    wt config get <key>         Get a configuration value
    wt config set <key> <value> Set a configuration value

    Available keys:
        editor.command              Editor command (default: cursor)
        workspace.root              Workspace root (default: ~/workspace)
        worktrees.path              Worktrees directory (default: <workspace>/worktrees)
        mattermost.path             Mattermost repo (default: <workspace>/mattermost)
        mattermost.enterprise_path  Enterprise repo (default: <workspace>/enterprise)

    Relative paths resolve from $HOME; absolute paths are used as-is.
    Re-run 'wt install' after changing paths to update shell integration.

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
