# wt - Git Worktree Manager

A powerful CLI tool to manage Git worktrees across multiple repositories, designed for developers who work on multiple branches simultaneously using tools like Cursor.

## Features

- 🚀 **Seamless Directory Switching**: Automatically switches to worktree directories
- 📋 **List Worktrees**: View all worktrees with status and last commit info
- 🔄 **Smart Branch Handling**: Auto-creates tracking branches from remotes
- 🧹 **Automatic Cleanup**: Remove stale worktrees older than 30 days
- 💻 **Cursor Integration**: Open Cursor editor directly in worktree
- ⚡ **Smart Completions**: Zsh auto-completions that prioritize existing worktrees
- 🧭 **Smart Navigation**: `cd ..` from worktree root takes you to ~/workspace
- 🔗 **Mattermost Dual-Repo**: Special support for Mattermost's dual-repository workflow

## Installation

### Quick Install

```bash
# Clone the repository
git clone <repo-url>
cd wt

# Build and install (easiest method)
make install

# Or manually:
# Build the binary
go build -o wt

# Move to PATH (choose one):
# Option 1: System-wide (requires sudo)
sudo mv wt /usr/local/bin/

# Option 2: User-local (no sudo needed)
mkdir -p ~/bin
mv wt ~/bin/
export PATH="$HOME/bin:$PATH"  # Add this to your .zshrc

# Install shell integration
wt install
```

### What `wt install` Does

The `install` command will:
- Add a shell function to `~/.zshrc` for seamless directory switching
- Install zsh completions for commands and branch names
- Provide instructions for activation

After installation, restart your terminal or run:

```bash
source ~/.zshrc
```

If TAB completion doesn't appear, verify your zsh completion is initialized and the user completion directory is on `$fpath`:

```zsh
# In your ~/.zshrc (before compinit)
fpath=("$HOME/.zsh/completion" $fpath)
typeset -U fpath

# Initialize zsh completions
autoload -Uz compinit && compinit -i
```

Then reload your shell:

```bash
source ~/.zshrc
```

### Manual Installation (Alternative)

If you prefer to manually add the shell function, add this to your `~/.zshrc`:

```zsh
# wt-shell-integration
wt() {
    local output
    output=$(command wt "$@")
    local exit_code=$?
    
    if echo "$output" | grep -q "^__WT_CD__:"; then
        local new_dir=$(echo "$output" | grep "^__WT_CD__:" | cut -d':' -f2-)
        builtin cd "$new_dir" || return 1
        
        # Check if there's a post-setup command to run
        if echo "$output" | grep -q "^__WT_CMD__:"; then
            local cmd=$(echo "$output" | grep "^__WT_CMD__:" | cut -d':' -f2-)
            echo "Running setup: $cmd"
            eval "$cmd"
        fi
        
        # Show output without markers
        echo "$output" | grep -v "^__WT_CD__:" | grep -v "^__WT_CMD__:"
    else
        echo "$output"
    fi
    
    return $exit_code
}

# Smart cd for worktrees - makes "cd .." from worktree root go to ~/workspace
cd() {
    # Only intercept "cd .." from worktree root
    if [[ "$1" == ".." ]]; then
        local parent_dir="${PWD%/*}"  # Get parent directory
        # Check if parent is ~/workspace/worktrees
        if [[ "$parent_dir" == "$HOME/workspace/worktrees" ]]; then
            builtin cd "$HOME/workspace"
            return
        fi
    fi
    builtin cd "$@"
}
# end wt-shell-integration
```

## Usage

### List Worktrees

```bash
wt ls
```

Shows all worktrees for the current repository with their status and last commit date.

### Checkout/Create Worktree

```bash
wt co <branch> [-b <base-branch>]
```

- If the worktree exists, switches to it
- If not, creates the worktree and switches to it
- If branch doesn't exist locally but exists on remote, creates a tracking branch
- If branch doesn't exist anywhere, creates a new branch from base branch (defaults to main/master)

Examples:
```bash
# Create worktree from default branch
wt co MM-123
# Creates worktree at ~/workspace/worktrees/mattermost-plugin-ai-MM-123/
# Automatically switches to that directory

# Create worktree from specific base branch
wt co feature/new-ui -b develop
# Creates new branch 'feature/new-ui' based on 'develop' branch

# Create worktree from release branch
wt co hotfix/urgent-fix --base release-1.0
```

### Clean Stale Worktrees

```bash
wt clean
```

Removes worktrees that:
- Have no uncommitted changes (clean)
- Haven't been updated in 30+ days

Shows a confirmation prompt before removing.

### Remove a Worktree

```bash
wt rm <branch> [-f|--force]
```

- Removes the git worktree and deletes the associated directory
- Use `-f` if the worktree has uncommitted changes

Example:
```bash
wt rm ai-prom-metrics
wt rm MM-123 -f
```

### Open in Cursor

```bash
wt cursor <branch> [-b <base-branch>]
```

Opens Cursor editor for the branch's worktree. Creates the worktree if it doesn't exist.

Examples:
```bash
# Open existing worktree
wt cursor MM-123

# Create new worktree from develop and open in Cursor
wt cursor feature/experiment -b develop
```

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

### Repository-Specific Setup

Some repositories require additional setup after creating a worktree. The tool automatically handles this:

**Mattermost Repository (`mattermost/mattermost`):**
- After creating a worktree, automatically runs `make setup-go-work` from the `server/` directory
- This ensures Go workspace files are properly configured for the new worktree
- The command runs automatically when switching to a newly created worktree

### Directory Switching

The tool uses a shell function wrapper that:
1. Captures output from the `wt` binary
2. Detects special `__WT_CD__:<path>` marker
3. Executes `cd <path>` in your current shell
4. Runs any post-setup commands if needed (via `__WT_CMD__` marker)
5. Shows remaining output

This provides seamless directory switching without subshell limitations, and automatically handles repository-specific setup commands.

### Smart `cd` Navigation

The installation includes a smart `cd` wrapper that makes navigation more intuitive:

**When you're at the root of a worktree** (e.g., `~/workspace/worktrees/mattermost-MM-123/`):
- `cd ..` → Takes you to `~/workspace` (your main workspace)
- This treats worktrees as siblings to your main repositories

**When you're in a subdirectory** (e.g., `~/workspace/worktrees/mattermost-MM-123/server/`):
- `cd ..` → Works normally, goes to parent directory

**All other `cd` commands** work exactly as expected.

This makes worktrees feel naturally integrated into your workspace hierarchy without needing to navigate through the `worktrees/` directory.

### Smart Tab Completions

The zsh completions intelligently prioritize what you're most likely to want:

**When you press TAB after `wt co` or `wt cursor`:**
1. **Existing worktrees** are shown first with "(existing worktree)" label
2. **Local branches** come next with "(local branch)" label  
3. **Remote branches** appear last with "(remote branch)" label

**Example:**
```bash
wt co agents-<TAB>
# Shows:
#   agents-prom-grafana  -- existing worktree
#   agents-dev          -- local branch
#   agents-staging      -- remote branch

wt cursor ai-<TAB>
# Completes to existing worktree: wt cursor ai-prom-metrics
```

This makes it fast to switch between your active worktrees without typing full branch names.

## Mattermost Dual-Repository Workflow

For developers working on Mattermost, `wt` provides special commands to manage the dual-repository setup (`mattermost/mattermost` and `mattermost/enterprise`).

### Setup Requirements

Ensure you have both repositories cloned:
```bash
~/workspace/mattermost/    # mattermost/mattermost monorepo
~/workspace/enterprise/    # mattermost/enterprise
```

### Creating a Mattermost Dual-Repo Worktree

```bash
# Create dual-repo worktree (auto-increments ports from 8065)
wt co-mm MM-12345

# Or use the short alias
wt mm MM-12345

# Create from a specific base branch
wt co-mm MM-12345 -b master

# Specify custom ports
wt co-mm MM-12345 --port 8070 --metrics-port 8072
```

This creates a unified worktree structure:
```
~/workspace/worktrees/mattermost-MM-12345/
├── [base config files]     # CLAUDE.md, mise.toml, etc.
├── server/                 # Worktree from mattermost repo
│   ├── server/
│   ├── webapp/
│   └── ...
└── enterprise/             # Worktree from enterprise repo
    └── ...
```

**What it does:**
1. Creates worktrees for both `mattermost` and `enterprise` repositories
2. Copies base configuration files from your main mattermost repo
3. Copies `go.work*` files and other development configurations
4. Updates `config.json` with unique ports for the server and metrics
5. Automatically runs `make setup-go-work` in the server directory
6. Switches your shell to the new worktree directory

### Removing Mattermost Dual-Repo Worktrees

```bash
# Standard removal
wt rm-mm MM-12345

# Force removal (for dirty worktrees)
wt rm-mm MM-12345 -f

# Remove and delete branches from both repos
wt rm-mm MM-12345 --delete-branch
```

### Opening in Cursor

```bash
# Open existing or create new Mattermost worktree in Cursor
wt cursor-mm MM-12345
```

## Workflow Examples

### Standard Repository Workflow

```bash
# Start in your main repository
cd ~/workspace/my-project

# Create worktree for ticket
wt co feature-123
# Now in ~/workspace/worktrees/my-project-feature-123/

# Open another Cursor window for a different branch
wt cursor feature-456

# List all worktrees
wt ls

# Clean up old worktrees
wt clean
```

### Mattermost Dual-Repo Workflow

```bash
# Start in mattermost repo
cd ~/workspace/mattermost

# Create dual-repo worktree for ticket MM-12345
wt co-mm MM-12345
# Now in ~/workspace/worktrees/mattermost-MM-12345/

# The server runs on auto-assigned port (e.g., 8066)
# Access at http://localhost:8066

# Work on another ticket in parallel
wt cursor-mm MM-12346
# Server runs on different port (e.g., 8067)

# Remove when done
wt rm-mm MM-12345
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

