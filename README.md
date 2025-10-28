# wt - Git Worktree Manager

A powerful CLI tool to manage Git worktrees across multiple repositories, designed for developers who work on multiple branches simultaneously using tools like Cursor.

## Features

- 🚀 **Seamless Directory Switching**: Automatically switches to worktree directories
- 📋 **List Worktrees**: View all worktrees with status and last commit info
- 🔄 **Smart Branch Handling**: Auto-creates tracking branches from remotes
- 🧹 **Automatic Cleanup**: Remove stale worktrees older than 30 days
- 💻 **Cursor Integration**: Open Cursor editor directly in worktree
- ⚡ **Shell Completions**: Zsh auto-completions for commands and branches

## Installation

### Build from Source

```bash
cd /Users/nickmisasi/workspace/wt
go build -o wt
sudo mv wt /usr/local/bin/
```

### Install Shell Integration

After building and moving the binary to your PATH:

```bash
wt install
```

This will:
- Add a shell function to `~/.zshrc` for directory switching
- Install zsh completions
- Provide instructions for activation

Then restart your terminal or run:

```bash
source ~/.zshrc
```

## Usage

### List Worktrees

```bash
wt ls
```

Shows all worktrees for the current repository with their status and last commit date.

### Checkout/Create Worktree

```bash
wt co <branch>
```

- If the worktree exists, switches to it
- If not, creates the worktree and switches to it
- If branch doesn't exist locally but exists on remote, creates a tracking branch
- If branch doesn't exist anywhere, creates a new branch

Example:
```bash
wt co MM-123
# Creates worktree at ~/workspace/worktrees/mattermost-plugin-ai-MM-123/
# Automatically switches to that directory
```

### Clean Stale Worktrees

```bash
wt clean
```

Removes worktrees that:
- Have no uncommitted changes (clean)
- Haven't been updated in 30+ days

Shows a confirmation prompt before removing.

### Open in Cursor

```bash
wt cursor <branch>
```

Opens Cursor editor for the branch's worktree. Creates the worktree if it doesn't exist.

### Show Help

```bash
wt help
# or just
wt
```

## How It Works

### Worktree Storage

Worktrees are stored in: `~/workspace/worktrees/`

Format: `<repo-name>-<branch-name>/`

Example:
- Repository: `mattermost-plugin-ai`
- Branch: `MM-123`
- Worktree path: `~/workspace/worktrees/mattermost-plugin-ai-MM-123/`

### Directory Switching

The tool uses a shell function wrapper that:
1. Captures output from the `wt` binary
2. Detects special `__WT_CD__:<path>` marker
3. Executes `cd <path>` in your current shell
4. Shows remaining output

This provides seamless directory switching without subshell limitations.

## Workflow Example

```bash
# Start in your main repository
cd ~/workspace/mattermost-plugin-ai

# Create worktree for ticket MM-123
wt co MM-123
# Now in ~/workspace/worktrees/mattermost-plugin-ai-MM-123/

# Open another Cursor window for a different ticket
wt cursor MM-456

# List all worktrees
wt ls
# Output:
#   MM-123                          [dirty]  (last commit: today)
#   MM-456                          [clean]  (last commit: 2 days ago)
#   feature/old-branch              [clean]  (last commit: 45 days ago)

# Clean up old worktrees
wt clean
# Removes feature/old-branch (>30 days old and clean)
```

## Requirements

- Go 1.16+ (for building)
- Git 2.5+ (for worktree support)
- Zsh (for shell integration)
- Cursor CLI (optional, for `wt cursor` command)

## Contributing

Feel free to submit issues and pull requests!

## License

MIT

