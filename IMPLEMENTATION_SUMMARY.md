# Mattermost Dual-Repo Worktree Implementation Summary

## ✅ Implementation Complete

Successfully implemented Mattermost dual-repository worktree support for the `wt` utility.

## 🎯 What Was Implemented

### New Commands

1. **`wt co-mm <branch>`** (aliases: `wt mm`, `wt mattermost`)
   - Creates a unified worktree with both mattermost and enterprise repositories
   - Auto-increments ports from 8065 for server and 8067 for metrics
   - Supports `-b/--base` for base branch selection
   - Supports `--port` and `--metrics-port` for custom ports

2. **`wt rm-mm <branch>`**
   - Removes Mattermost dual-repo worktrees
   - Removes worktrees from both repositories
   - Supports `-f/--force` for dirty worktrees
   - Supports `--delete-branch` to delete branches from both repos

3. **`wt cursor-mm <branch>`**
   - Opens Cursor editor for Mattermost worktree
   - Creates worktree if it doesn't exist

### Core Features

✅ **Dual-Repository Management**
- Automatically creates worktrees from both `~/workspace/mattermost` and `~/workspace/enterprise`
- Maintains proper git worktree references in both repositories

✅ **File Copying**
- Copies base configuration files (CLAUDE.md, mise.toml, etc.) from mattermost repo
- Copies `go.work*` files from both repositories
- Copies development configurations (config.json, docker-compose overrides, etc.)
- Uses configurable file mappings for extensibility

✅ **Port Management**
- Auto-increments ports based on existing worktrees
- Starts from 8065 for server, 8067 for metrics
- Updates config.json with unique ports for each worktree
- Prevents port conflicts when running multiple instances

✅ **Branch Handling**
- Detects branches in both repositories (local and remote)
- Creates tracking branches from remotes
- Creates new branches from specified base branch
- Uses same base branch for both repos when creating new branches

✅ **Error Handling & Cleanup**
- Validates repository existence before starting
- Comprehensive error messages
- Automatic cleanup on failures (removes partial worktrees)
- Rollback mechanism to maintain clean state

✅ **Shell Integration**
- Uses existing `__WT_CD__` marker for directory switching
- Uses `__WT_CMD__` marker to run `make setup-go-work` automatically
- Works seamlessly with existing shell integration

## 📁 Files Created

### `internal/mattermost.go` (576 lines)
Core Mattermost functionality:
- `MattermostConfig`: Configuration structure
- `NewMattermostConfig()`: Initializes Mattermost config
- `ValidateMattermostSetup()`: Validates repository setup
- `CreateMattermostDualWorktree()`: Main worktree creation logic
- `RemoveMattermostDualWorktree()`: Removal logic
- `DeleteBranchFromRepos()`: Branch deletion from both repos
- `IsMattermostDualWorktree()`: Detection function
- `GetAvailablePorts()`: Port auto-increment logic
- File copying utilities with glob pattern support
- JSON config.json manipulation for port updates

### `cmd/mattermost.go` (157 lines)
Command handlers:
- `RunMattermostCheckout()`: Handles co-mm command
- `RunMattermostRemove()`: Handles rm-mm command
- `RunMattermostCursor()`: Handles cursor-mm command

### Modified Files

**`main.go`**
- Added routing for new commands (co-mm, mm, mattermost, rm-mm, cursor-mm)
- Added argument parsing functions for Mattermost commands

**`cmd/help.go`**
- Updated help text with Mattermost commands section
- Added examples and port configuration documentation

**`README.md`**
- Added Mattermost dual-repository workflow section
- Comprehensive setup requirements and examples
- Workflow examples comparing standard and Mattermost usage

## 🏗️ Worktree Structure

When you run `wt co-mm MM-12345`, it creates:

```
~/workspace/worktrees/mattermost-MM-12345/
├── CLAUDE.md                      # Copied from ~/workspace/mattermost
├── CLAUDE.local.md               # Copied from ~/workspace/mattermost
├── mise.toml                     # Copied from ~/workspace/mattermost
├── docker-compose.yml            # Copied from ~/workspace/mattermost
├── [other base files]            # Copied from ~/workspace/mattermost
├── server/                       # Git worktree from ~/workspace/mattermost
│   ├── server/
│   │   ├── go.work              # Copied from ~/workspace/mattermost/server/
│   │   └── go.work.sum          # Copied from ~/workspace/mattermost/server/
│   ├── webapp/
│   │   └── .dir-locals.el       # Copied from ~/workspace/mattermost/webapp/
│   ├── config/
│   │   └── config.json          # Copied and ports updated
│   └── ...
└── enterprise/                   # Git worktree from ~/workspace/enterprise
    ├── go.work                  # Copied from ~/workspace/enterprise/
    ├── go.work.sum              # Copied from ~/workspace/enterprise/
    └── ...
```

## 🧪 Testing Checklist

To test the implementation:

- [ ] **Setup**: Ensure you have both repos at `~/workspace/mattermost` and `~/workspace/enterprise`
- [ ] **Create worktree**: `wt co-mm test-branch`
- [ ] **Verify structure**: Check that both server/ and enterprise/ exist
- [ ] **Check ports**: Verify config.json has unique ports
- [ ] **Create another**: `wt co-mm test-branch-2` and verify port increment
- [ ] **Remove**: `wt rm-mm test-branch`
- [ ] **Force remove**: Make changes and `wt rm-mm test-branch-2 -f`
- [ ] **Branch deletion**: `wt rm-mm test-branch-3 --delete-branch`
- [ ] **Custom ports**: `wt co-mm test-branch-4 --port 8080`
- [ ] **Base branch**: `wt co-mm test-branch-5 -b develop`
- [ ] **Cursor integration**: `wt cursor-mm test-branch`
- [ ] **Help text**: `wt help` shows Mattermost commands
- [ ] **Error handling**: Try without repos and verify error messages

## 🚀 Usage Examples

### Basic Creation
```bash
cd ~/workspace/mattermost
wt co-mm MM-12345
# Creates dual worktree with auto ports (8065, 8067)
# Automatically switches to ~/workspace/worktrees/mattermost-MM-12345/
```

### With Custom Ports
```bash
wt co-mm MM-12346 --port 8070 --metrics-port 8072
# Server on http://localhost:8070
# Metrics on http://localhost:8072/metrics
```

### From Specific Base Branch
```bash
wt co-mm feature/new-login -b develop
# Creates new branch from develop in both repos
```

### Removal
```bash
# Standard removal
wt rm-mm MM-12345

# Force removal (dirty worktree)
wt rm-mm MM-12345 -f

# Remove and delete branches from both repos
wt rm-mm MM-12345 --delete-branch
```

### Open in Cursor
```bash
wt cursor-mm MM-12345
# Opens Cursor at worktree root
```

## 🎨 Design Decisions

### Why Separate Files?
- **Isolation**: Mattermost logic is completely separate from core worktree logic
- **Maintainability**: Easy to update Mattermost-specific features without affecting other repos
- **Clarity**: Clear separation of concerns
- **Optional**: Mattermost commands don't affect standard worktree operations

### Why Auto-Increment Ports?
- **No User Interruption**: Automatic port selection maintains smooth workflow
- **Conflict Prevention**: Prevents port conflicts when running multiple instances
- **Predictable**: Consistent port assignment based on existing worktrees
- **Override Available**: Users can still specify custom ports via flags

### Why Copy Files Instead of Symlinks?
- **Independence**: Each worktree has its own config that can be modified
- **Port Configuration**: Allows unique port settings per worktree
- **Isolation**: Changes in one worktree don't affect others
- **Safety**: Prevents accidental modification of base repository files

## 📊 Statistics

- **Lines Added**: 959 lines
- **Files Created**: 2 new files
- **Files Modified**: 3 existing files
- **New Commands**: 3 main commands (with 2 aliases)
- **Build Status**: ✅ Compiles successfully
- **Linter Status**: ✅ No linting errors

## 🔄 Next Steps

### Potential Future Enhancements

1. **Enhanced List Command**
   - Add dual-repo indicator to `wt ls` output
   - Show port assignments in list view

2. **Configuration File**
   - Allow users to customize file mappings
   - Support for other dual-repo setups

3. **Port Conflict Detection**
   - Check if ports are in use before assignment
   - Suggest alternative ports if conflicts exist

4. **Completion Support**
   - Add zsh completions for Mattermost commands
   - Include branch suggestions from both repos

5. **Status Indicators**
   - Show dirty status for both repos in dual worktrees
   - Display last commit for both server and enterprise

## 📝 Notes

- All changes are on the `feature/mattermost-dual-repo` branch
- Ready for testing and review
- Standard worktree operations (`wt co`, `wt rm`, etc.) remain unchanged
- Mattermost functionality only activates when using Mattermost-specific commands
- Maintains backward compatibility with existing workflows

## ✨ Success Criteria Met

✅ Creates unified worktree with both repositories
✅ Copies necessary configuration files
✅ Updates config.json with unique ports
✅ Auto-increments ports from existing worktrees
✅ Handles branch creation in both repos
✅ Provides comprehensive error handling
✅ Supports all required flags and options
✅ Includes proper documentation
✅ Maintains clean code structure
✅ No impact on existing functionality

