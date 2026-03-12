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
    co <branch> [options]        Checkout/create worktree for branch and switch to it
    rm <branch> [-f]             Remove a worktree for branch (use -f to force)
    clean                        Remove stale worktrees (clean, >30 days old)
    edit [<branch>] [options]    Open configured editor (current worktree if no branch)
    cursor                       (deprecated) Alias for 'edit'
    port                         Show current worktree's mapped ports
    t, toggle                    Return to parent repository from worktree
    sessions [subcommand]        Manage claudemux persistent Claude sessions
    config                       Manage configuration (get/set/show)
    install                      Install shell integration and completions
    help                         Show this help message

OPTIONS:
    -b, --base <branch>         Base branch for new branches (defaults to main/master)
    -f, --force                 Force removal when using 'wt rm'
    -n, --no-claude-docs        Skip running enable-claude-docs.sh after worktree creation
    --claudemux                 Create a claudemux session for this worktree (overrides config)
    --no-claudemux              Skip claudemux session creation (overrides config)

WORKTREE STORAGE:
    Standard worktrees: <worktrees.path>/<repo-name>-<branch-name>/
    worktrees.path defaults to <workspace.root>/worktrees (configurable via 'wt config')

CLAUDEMUX (PERSISTENT CLAUDE SESSIONS):
    Claudemux creates detached tmux sessions running Claude Code alongside your worktrees.
    Sessions persist across terminal restarts and are accessible from claude.ai/code and
    the Claude mobile app via remote control. Claude's --continue flag restores full
    conversation history if the process restarts.

    Quick start:
        wt config set claudemux.enabled true    Enable claudemux globally
        wt co my-branch                         Creates worktree + Claude session
        wt sessions                             List active sessions

    Per-checkout override:
        wt co my-branch --claudemux             Force session creation (even if disabled)
        wt co my-branch --no-claudemux          Skip session (even if enabled)

    Session management:
        wt sessions                  List all managed tmux sessions with status
        wt sessions health           Check sessions and auto-respawn any dead ones
        wt sessions start            Create sessions for all worktrees missing one
        wt sessions stop             Kill all managed tmux sessions
        wt sessions restart          Stop all sessions, then start fresh

    Sessions are named wt-<sanitized-branch> (e.g., wt-feature-auth for feature/auth).
    They auto-cleanup when you run 'wt rm' or 'wt clean'.

    Crash recovery:
        Sessions use tmux remain-on-exit + pane-died hooks to automatically respawn
        if Claude crashes. You can also manually check with 'wt sessions health'.

    Session cap:
        A maximum number of concurrent sessions is enforced (default: 10). When the cap
        is reached, the oldest session is evicted. Configure with:
            wt config set claudemux.max_sessions 15

    Requirements:
        - tmux must be installed and on PATH
        - claude CLI must be installed and on PATH

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

    # With claudemux
    wt config set claudemux.enabled true
    wt co feature-123            # Creates worktree + Claude session
    wt sessions                  # Check session status
    wt rm feature-123            # Removes worktree + kills session

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
        worktrees.path              Worktrees directory (default: <workspace.root>/worktrees)
        mattermost.path             Mattermost repo (default: <workspace.root>/mattermost)
        mattermost.enterprise_path  Enterprise repo (default: <workspace.root>/enterprise)
        claudemux.enabled           Enable claudemux sessions on checkout (default: false)
        claudemux.command           Claude command to run (default: claude --continue --dangerously-skip-permissions)
        claudemux.max_sessions      Maximum concurrent sessions (default: 10)

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
	err := RunList(config, false, false)
	if err != nil {
		// If we're not in a git repo, that's okay for default command
		fmt.Fprintf(os.Stderr, "\n(Run this command from inside a git repository to see worktrees)\n")
	}

	return nil
}
