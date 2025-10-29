# Worktree Operations Analysis

This document provides a comprehensive analysis of all operations, file manipulations, and configurations that will need to be performed by the wt command line project.

## Overview

The `wt` utility manages git worktrees with special handling for the Mattermost dual-repository setup:
- **Mattermost Repository**: `~/workspace/mattermost` (mattermost/mattermost monorepo with server and webapp)
- **Enterprise Repository**: `~/workspace/enterprise` (mattermost/enterprise)

## Your Workspace Structure

```
~/workspace/
├── mattermost/              # mattermost/mattermost monorepo
│   ├── server/
│   ├── webapp/
│   └── ... (other files)
├── enterprise/              # mattermost/enterprise repo
│   └── ...
├── wt/                      # this project
├── <other-repos>/           # your other git repositories
└── worktrees/               # all worktrees created by wt
    ├── mattermost-MM-123/   # worktree for mattermost repo, branch MM-123
    ├── mattermost-MM-456/   # another mattermost worktree
    └── enterprise-MM-123/   # worktree for enterprise repo, branch MM-123
```

### Current `wt` Behavior

**Worktree Naming Convention:**
- Format: `<repo-name>-<branch-name>`
- Example: Branch `MM-123` in `mattermost` → `~/workspace/worktrees/mattermost-MM-123/`
- Example: Branch `MM-123` in `enterprise` → `~/workspace/worktrees/enterprise-MM-123/`

**What `wt` Currently Does:**
1. Creates single worktrees for individual repositories
2. Automatically switches to worktree directory
3. Runs `make setup-go-work` for mattermost repo after creation
4. Lists, cleans, and removes worktrees
5. Opens worktrees in Cursor

**What Needs to Be Added (from bash script analysis):**
The bash script creates a unified workspace that links both mattermost and enterprise repos together with shared configuration. This needs to be implemented as Mattermost-specific functionality.

## Mattermost Dual-Repo Worktree Structure

When working on Mattermost, developers often need to work on both the `mattermost` and `enterprise` repositories simultaneously for the same branch. The bash script creates a unified worktree structure:

```
~/workspace/worktrees/mattermost-MM-123/
├── [base configuration files]
├── server/          # git worktree from ~/workspace/mattermost
│   ├── server/
│   ├── webapp/
│   └── ...
└── enterprise/      # git worktree from ~/workspace/enterprise
    └── ...
```

**Key Concept:** Instead of separate `mattermost-MM-123` and `enterprise-MM-123` directories, create ONE directory that contains worktrees from BOTH repos, mimicking the standard Mattermost development setup.

### Implementation Strategy for `wt`

**New Command Needed:** `wt co-mattermost <branch>` (or similar)

This command should:
1. Detect that you're in the `mattermost` repository
2. Create a unified worktree directory at `~/workspace/worktrees/mattermost-<branch>/`
3. Create worktree for `mattermost` repo at `<dir>/server/`
4. Create worktree for `enterprise` repo at `<dir>/enterprise/`
5. Copy necessary configuration files
6. Update configuration files with branch-specific settings

## File Copy Operations

### 1. Base Configuration Files

**Source:** `~/workspace/mattermost/` (the main repository)

**What Gets Copied to Worktree Root:**
All files and directories from `~/workspace/mattermost/` **EXCEPT**:
- `server/` directory (this becomes a worktree)
- `webapp/` directory (included in the server worktree)
- `.git/` directory
- Any other git-ignored files

**Implementation:**
```go
// Copy all items from source except exclusions
exclusions := []string{"server", "webapp", ".git"}
copyFilesExcept(sourceDir, targetDir, exclusions)
```

**Typical Files That Get Copied:**
- `CLAUDE.md`
- `CLAUDE.local.md`
- `mise.toml`
- `docker-compose.yml`
- `.gitignore`
- Any other configuration files in the base directory

### 2. Mattermost Repository Files to Copy

After creating the `server/` worktree, certain files need to be copied from `~/workspace/mattermost/` into the worktree:

| Source Path | Destination Path | Purpose |
|-------------|------------------|---------|
| `~/workspace/mattermost/server/go.work*` | `<worktree>/server/server/go.work*` | Go workspace configuration |
| `~/workspace/mattermost/webapp/.dir-locals.el` | `<worktree>/server/webapp/.dir-locals.el` | Emacs editor config |
| `~/workspace/mattermost/config/config.json` | `<worktree>/server/config/config.json` | Mattermost server config |
| `~/workspace/mattermost/docker-compose.override.yaml` | `<worktree>/server/docker-compose.override.yaml` | Docker overrides |
| `~/workspace/mattermost/server/config.override.mk` | `<worktree>/server/server/config.override.mk` | Make overrides |

**Implementation Notes:**
1. These files are copied AFTER the git worktree is created
2. Check if source file exists before copying (some may not exist in all setups)
3. Create parent directories as needed
4. The `go.work*` glob should match `go.work`, `go.work.sum`, etc.

**Go Configuration:**
```go
type FileMapping struct {
    SourceGlob string  // Pattern to match (e.g., "server/go.work*")
    DestPath   string  // Destination path relative to worktree
}

mattermostFileMappings := []FileMapping{
    {"server/go.work*", "server/server/"},
    {"webapp/.dir-locals.el", "server/webapp/.dir-locals.el"},
    {"config/config.json", "server/config/config.json"},
    {"docker-compose.override.yaml", "server/docker-compose.override.yaml"},
    {"server/config.override.mk", "server/server/config.override.mk"},
}
```

### 3. Enterprise Repository Files to Copy

After creating the `enterprise/` worktree, copy Go workspace files:

| Source Path | Destination Path | Purpose |
|-------------|------------------|---------|
| `~/workspace/enterprise/go.work*` | `<worktree>/enterprise/go.work*` | Go workspace configuration |

**Note:** The bash script has an empty array for additional enterprise files, suggesting this is configurable but typically only `go.work*` files are needed.

## Configuration Modifications

### Port Configuration for config.json

**Purpose:** Allow multiple Mattermost instances (worktrees) to run simultaneously on different ports.

**File Location:** `~/workspace/worktrees/mattermost-<branch>/server/config/config.json`

**Default Ports:**
- Server Port: `8065`
- Metrics Port: `8067`

**Implementation Approach:**

Rather than prompting the user for ports (interrupts workflow), consider:
1. **Auto-increment ports** based on existing worktrees
2. **Use branch name hash** to generate consistent but unique ports
3. **Allow configuration** via flags: `wt co-mattermost MM-123 --port 8066 --metrics-port 8068`

### JSON Modifications Required

The `config.json` file needs three fields updated:

#### 1. ServiceSettings.ListenAddress
```json
{
  "ServiceSettings": {
    "ListenAddress": ":8065"  // Change to unique port
  }
}
```

#### 2. ServiceSettings.SiteURL
```json
{
  "ServiceSettings": {
    "SiteURL": "http://localhost:8065"  // Update to match port
  }
}
```

#### 3. MetricsSettings.ListenAddress
```json
{
  "MetricsSettings": {
    "ListenAddress": ":8067"  // Change to unique metrics port
  }
}
```

### Implementation in Go

**Option 1: Using encoding/json (recommended)**

```go
import (
    "encoding/json"
    "os"
)

type Config struct {
    ServiceSettings struct {
        ListenAddress string `json:"ListenAddress"`
        SiteURL       string `json:"SiteURL"`
    } `json:"ServiceSettings"`
    MetricsSettings struct {
        ListenAddress string `json:"ListenAddress"`
    } `json:"MetricsSettings"`
}

func updateConfigPorts(configPath string, port, metricsPort int) error {
    // Read file
    data, err := os.ReadFile(configPath)
    if err != nil {
        return err
    }
    
    // Parse JSON
    var config Config
    if err := json.Unmarshal(data, &config); err != nil {
        return err
    }
    
    // Update ports
    config.ServiceSettings.ListenAddress = fmt.Sprintf(":%d", port)
    config.ServiceSettings.SiteURL = fmt.Sprintf("http://localhost:%d", port)
    config.MetricsSettings.ListenAddress = fmt.Sprintf(":%d", metricsPort)
    
    // Write back with indentation
    updatedData, err := json.MarshalIndent(config, "", "  ")
    if err != nil {
        return err
    }
    
    return os.WriteFile(configPath, updatedData, 0644)
}
```

**Option 2: Using regex/text replacement (preserves formatting better)**

```go
import (
    "os"
    "regexp"
)

func updateConfigPorts(configPath string, port, metricsPort int) error {
    data, err := os.ReadFile(configPath)
    if err != nil {
        return err
    }
    
    content := string(data)
    
    // Update ServiceSettings.ListenAddress (first occurrence)
    re1 := regexp.MustCompile(`("ListenAddress":\s*)":[0-9]+"`)
    content = re1.ReplaceAllString(content, fmt.Sprintf(`$1":%d"`, port))
    
    // Update SiteURL
    re2 := regexp.MustCompile(`("SiteURL":\s*)"http://localhost:[0-9]+"`)
    content = re2.ReplaceAllString(content, fmt.Sprintf(`$1"http://localhost:%d"`, port))
    
    // Update MetricsSettings.ListenAddress
    // More complex: need to match within MetricsSettings block
    re3 := regexp.MustCompile(`("MetricsSettings":[^}]*"ListenAddress":\s*)":[0-9]+"`)
    content = re3.ReplaceAllString(content, fmt.Sprintf(`$1":%d"`, metricsPort))
    
    return os.WriteFile(configPath, []byte(content), 0644)
}
```

**Recommendation:** Use Option 1 (JSON parsing) if the full config structure is known, or Option 2 (regex) to preserve exact formatting and comments.

## Git Worktree Operations for Mattermost

### Dual-Repository Workflow

When creating a Mattermost worktree, you need to handle TWO git repositories:
1. `~/workspace/mattermost` → creates worktree at `<target>/server/`
2. `~/workspace/enterprise` → creates worktree at `<target>/enterprise/`

### Branch Detection (Per Repository)

The `wt` utility already implements branch detection. The logic needs to be applied to BOTH repositories:

#### Current Implementation (from internal/git.go)
```go
// 1. Check if branch exists locally
branchExists, err := repo.BranchExists(branch)

// 2. If not local, check remote
if !branchExists {
    remoteBranchExists, err := repo.RemoteBranchExists(branch)
    if remoteBranchExists {
        // Create tracking branch
        repo.CreateTrackingBranch(branch)
    }
}

// 3. If doesn't exist anywhere, create new branch
if !branchExists && !remoteBranchExists {
    // Use base branch (default or specified)
    createNewBranch = true
}
```

### Mattermost Dual-Worktree Creation Process

**Step-by-step for `wt co-mattermost MM-123`:**

1. **Verify repositories exist**
   ```go
   mattermostPath := filepath.Join(homeDir, "workspace", "mattermost")
   enterprisePath := filepath.Join(homeDir, "workspace", "enterprise")
   // Check both exist and are git repos
   ```

2. **Create target directory**
   ```go
   targetDir := filepath.Join(homeDir, "workspace", "worktrees", "mattermost-MM-123")
   os.MkdirAll(targetDir, 0755)
   ```

3. **Copy base configuration files**
   ```go
   copyFilesExcept(mattermostPath, targetDir, []string{"server", "webapp", ".git"})
   ```

4. **Create mattermost worktree at `server/`**
   ```go
   serverWorktreePath := filepath.Join(targetDir, "server")
   // From ~/workspace/mattermost, create worktree for branch MM-123
   // Use existing worktree creation logic
   ```

5. **Create enterprise worktree at `enterprise/`**
   ```go
   enterpriseWorktreePath := filepath.Join(targetDir, "enterprise")
   // From ~/workspace/enterprise, create worktree for branch MM-123
   // Use same base branch if specified
   ```

6. **Copy additional files**
   - Copy go.work* files
   - Copy config files (config.json, etc.)

7. **Update config.json ports**
   ```go
   configPath := filepath.Join(targetDir, "server", "config", "config.json")
   updateConfigPorts(configPath, serverPort, metricsPort)
   ```

8. **Run post-setup**
   ```go
   // Run make setup-go-work in server/server/
   ```

### Base Branch Handling

**For Mattermost dual-repo setup:**

If branch doesn't exist in either repository, use the same base branch for both:
- User specifies: `wt co-mattermost MM-123 -b develop`
- Or detect default branch: typically `master` for both repos

**Implementation:**
```go
baseBranch := specifiedBaseBranch
if baseBranch == "" {
    // Detect from mattermost repo
    baseBranch = detectDefaultBranch(mattermostPath)
}

// Use this base branch for both mattermost and enterprise worktrees
```

### Error Handling and Rollback

If any step fails during dual-worktree creation:

```go
func createMattermostWorktree(branch string) error {
    // Create target directory
    targetDir := getTargetDir(branch)
    
    // Cleanup function
    cleanup := func() {
        // Remove target directory
        os.RemoveAll(targetDir)
        
        // Prune worktrees from both repos
        exec.Command("git", "-C", mattermostPath, "worktree", "prune").Run()
        exec.Command("git", "-C", enterprisePath, "worktree", "prune").Run()
    }
    
    // Try creating worktrees
    if err := createMattermostWorktree(); err != nil {
        cleanup()
        return err
    }
    
    if err := createEnterpriseWorktree(); err != nil {
        cleanup()
        return err
    }
    
    // Copy files, update config, etc.
    // ...
    
    return nil
}
```

## Worktree Removal Operations

### Current `wt rm` Behavior

The `wt` utility already implements worktree removal via `wt rm <branch>`. For Mattermost dual-repo worktrees, this needs enhancement.

### Mattermost Removal Requirements

For dual-repository Mattermost worktrees, need to remove:
1. The `server/` worktree (from mattermost repo)
2. The `enterprise/` worktree (from enterprise repo)
3. The entire directory structure

### Detection Strategy

**Identify if a worktree is a Mattermost dual-repo setup:**

```go
func isMattermostDualWorktree(worktreePath string) bool {
    // Check if both server/ and enterprise/ subdirectories exist
    // and if they are git worktrees
    serverPath := filepath.Join(worktreePath, "server")
    enterprisePath := filepath.Join(worktreePath, "enterprise")
    
    serverIsWorktree := isGitWorktree(serverPath)
    enterpriseIsWorktree := isGitWorktree(enterprisePath)
    
    return serverIsWorktree && enterpriseIsWorktree
}
```

### Enhanced Removal Process

```go
func removeMattermostWorktree(branch string, force bool) error {
    worktreePath := filepath.Join(worktreeBasePath, "mattermost-" + branch)
    
    if !isMattermostDualWorktree(worktreePath) {
        // Standard removal
        return removeStandardWorktree(worktreePath, force)
    }
    
    // Dual-repo removal
    serverPath := filepath.Join(worktreePath, "server")
    enterprisePath := filepath.Join(worktreePath, "enterprise")
    
    // Remove server worktree (from mattermost repo)
    mattermostRepo := filepath.Join(homeDir, "workspace", "mattermost")
    removeWorktreeFromRepo(mattermostRepo, serverPath, force)
    
    // Remove enterprise worktree (from enterprise repo)
    enterpriseRepo := filepath.Join(homeDir, "workspace", "enterprise")
    removeWorktreeFromRepo(enterpriseRepo, enterprisePath, force)
    
    // Remove directory structure
    return os.RemoveAll(worktreePath)
}

func removeWorktreeFromRepo(repoPath, worktreePath string, force bool) error {
    args := []string{"-C", repoPath, "worktree", "remove"}
    if force {
        args = append(args, "-f")
    }
    args = append(args, worktreePath)
    
    cmd := exec.Command("git", args...)
    return cmd.Run()
}
```

### Optional Branch Deletion

The bash script prompts to delete branches after worktree removal. Consider adding this as a flag:

```bash
wt rm MM-123 --delete-branch    # Remove worktree AND delete branch
wt rm MM-123 -f --delete-branch # Force remove and delete branch
```

```go
func deleteBranchFromRepos(branch string) error {
    // Delete from mattermost repo
    mattermostRepo := filepath.Join(homeDir, "workspace", "mattermost")
    exec.Command("git", "-C", mattermostRepo, "branch", "-D", branch).Run()
    
    // Delete from enterprise repo
    enterpriseRepo := filepath.Join(homeDir, "workspace", "enterprise")
    exec.Command("git", "-C", enterpriseRepo, "branch", "-D", branch).Run()
    
    return nil
}
```

## List Operations

### Current `wt ls` Behavior

The `wt` utility already lists worktrees using `git worktree list --porcelain` and filters by worktree base path.

### Enhancement for Mattermost Dual-Repo

When listing worktrees, detect and display Mattermost dual-repo worktrees differently:

```go
type WorktreeInfo struct {
    Path       string
    Branch     string
    IsDirty    bool
    LastCommit time.Time
    IsDualRepo bool  // NEW: flag for Mattermost dual-repo setup
}

func listWorktrees() ([]WorktreeInfo, error) {
    // Get worktrees from git
    worktrees := getGitWorktrees()
    
    // Enhance with dual-repo detection
    for i := range worktrees {
        worktrees[i].IsDualRepo = isMattermostDualWorktree(worktrees[i].Path)
    }
    
    return worktrees, nil
}
```

**Display Format:**
```
Worktrees for mattermost:
  MM-123    [dual: ✓] [clean]  (last commit: 2 days ago)
  MM-456    [dual: ✓] [dirty]  (last commit: today)
  MM-789    [single]  [clean]  (last commit: 5 days ago)
```

## Validation and Error Handling

### Pre-flight Checks for Mattermost Dual-Repo

Before creating a Mattermost dual-repository worktree:

```go
func validateMattermostSetup() error {
    homeDir, _ := os.UserHomeDir()
    
    // Check mattermost repository exists
    mattermostPath := filepath.Join(homeDir, "workspace", "mattermost")
    if !isGitRepo(mattermostPath) {
        return fmt.Errorf("mattermost repository not found at %s", mattermostPath)
    }
    
    // Check enterprise repository exists
    enterprisePath := filepath.Join(homeDir, "workspace", "enterprise")
    if !isGitRepo(enterprisePath) {
        return fmt.Errorf("enterprise repository not found at %s", enterprisePath)
    }
    
    // Check worktrees directory is accessible
    worktreesPath := filepath.Join(homeDir, "workspace", "worktrees")
    if err := os.MkdirAll(worktreesPath, 0755); err != nil {
        return fmt.Errorf("cannot create worktrees directory: %w", err)
    }
    
    return nil
}

func isGitRepo(path string) bool {
    gitDir := filepath.Join(path, ".git")
    info, err := os.Stat(gitDir)
    return err == nil && (info.IsDir() || info.Mode().IsRegular())
}
```

### Port Validation

```go
func validatePort(port int) error {
    if port < 1 || port > 65535 {
        return fmt.Errorf("port must be between 1 and 65535, got %d", port)
    }
    return nil
}

func isPortInUse(port int) bool {
    // Optional: check if port is already in use
    conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), time.Second)
    if err != nil {
        return false
    }
    conn.Close()
    return true
}
```

### Branch Name Validation

```go
func validateBranchName(branch string) error {
    // Git branch names have specific rules
    if strings.Contains(branch, "..") {
        return fmt.Errorf("branch name cannot contain '..'")
    }
    if strings.HasPrefix(branch, "/") || strings.HasSuffix(branch, "/") {
        return fmt.Errorf("branch name cannot start or end with '/'")
    }
    if strings.Contains(branch, "//") {
        return fmt.Errorf("branch name cannot contain '//'")
    }
    return nil
}
```

### Error Cleanup

Defer cleanup to ensure proper rollback on errors:

```go
func createMattermostWorktree(branch string, baseBranch string) error {
    targetDir := getTargetDir(branch)
    
    // Track what we've created for cleanup
    var serverWorktreeCreated, enterpriseWorktreeCreated, targetDirCreated bool
    
    // Defer cleanup on error
    defer func() {
        if err := recover(); err != nil {
            cleanup(targetDir, serverWorktreeCreated, enterpriseWorktreeCreated)
            panic(err)
        }
    }()
    
    // Create target directory
    if err := os.MkdirAll(targetDir, 0755); err != nil {
        return fmt.Errorf("failed to create target directory: %w", err)
    }
    targetDirCreated = true
    
    // Create server worktree
    if err := createServerWorktree(branch, baseBranch); err != nil {
        cleanup(targetDir, false, false)
        return fmt.Errorf("failed to create server worktree: %w", err)
    }
    serverWorktreeCreated = true
    
    // Create enterprise worktree
    if err := createEnterpriseWorktree(branch, baseBranch); err != nil {
        cleanup(targetDir, serverWorktreeCreated, false)
        return fmt.Errorf("failed to create enterprise worktree: %w", err)
    }
    enterpriseWorktreeCreated = true
    
    // Copy files and configure
    if err := copyAndConfigure(targetDir); err != nil {
        cleanup(targetDir, serverWorktreeCreated, enterpriseWorktreeCreated)
        return fmt.Errorf("failed to configure worktree: %w", err)
    }
    
    return nil
}

func cleanup(targetDir string, serverCreated, enterpriseCreated bool) {
    homeDir, _ := os.UserHomeDir()
    
    if serverCreated {
        mattermostPath := filepath.Join(homeDir, "workspace", "mattermost")
        exec.Command("git", "-C", mattermostPath, "worktree", "prune").Run()
    }
    
    if enterpriseCreated {
        enterprisePath := filepath.Join(homeDir, "workspace", "enterprise")
        exec.Command("git", "-C", enterprisePath, "worktree", "prune").Run()
    }
    
    if targetDir != "" {
        os.RemoveAll(targetDir)
    }
}
```

## Implementation Checklist for `wt` Utility

### What's Already Implemented ✓

- ✓ **Basic worktree operations** (create, list, remove, clean)
- ✓ **Branch detection** (local and remote)
- ✓ **Directory switching** (shell integration with `__WT_CD__`)
- ✓ **Post-setup commands** (via `__WT_CMD__` marker)
- ✓ **Cursor integration** (`wt cursor` command)
- ✓ **Zsh completions**
- ✓ **Git repository detection**
- ✓ **Worktree status checking** (dirty/clean, last commit)

### What Needs to Be Added for Mattermost

#### 1. New Command: Mattermost Dual-Repo Worktree

- [ ] **Command:** `wt co-mm <branch>` or `wt checkout-mattermost <branch>`
  - [ ] Alias options: `wt mm <branch>`, `wt com <branch>`
  - [ ] Flag support: `-b/--base <branch>`, `--port <port>`, `--metrics-port <port>`

#### 2. Directory Management

- [ ] **Create unified worktree directory**
  - [ ] Pattern: `~/workspace/worktrees/mattermost-<branch>/`
  - [ ] Create `server/` subdirectory for mattermost worktree
  - [ ] Create `enterprise/` subdirectory for enterprise worktree

- [ ] **Copy base files**
  - [ ] Copy all files from `~/workspace/mattermost/` except `server/`, `webapp/`, `.git/`
  - [ ] Implement `copyFilesExcept()` function with exclusion list

#### 3. Dual-Worktree Creation

- [ ] **Mattermost worktree**
  - [ ] Create from `~/workspace/mattermost` repo
  - [ ] Target: `<worktree-dir>/server/`
  - [ ] Apply existing branch detection logic

- [ ] **Enterprise worktree**
  - [ ] Create from `~/workspace/enterprise` repo
  - [ ] Target: `<worktree-dir>/enterprise/`
  - [ ] Use same base branch as mattermost if creating new

#### 4. File Copy Operations

- [ ] **Post-worktree file copying**
  - [ ] Copy `go.work*` from `~/workspace/mattermost/server/` → `<worktree>/server/server/`
  - [ ] Copy `go.work*` from `~/workspace/enterprise/` → `<worktree>/enterprise/`
  - [ ] Copy other configured files (config.json, docker-compose overrides, etc.)
  - [ ] Implement glob pattern matching for `go.work*`

- [ ] **Make it configurable**
  - [ ] Configuration file or struct for file mappings
  - [ ] Allow users to add custom file mappings

#### 5. Configuration File Editing

- [ ] **config.json port updates**
  - [ ] Parse JSON file
  - [ ] Update `ServiceSettings.ListenAddress`
  - [ ] Update `ServiceSettings.SiteURL`
  - [ ] Update `MetricsSettings.ListenAddress`
  - [ ] Write back (preserve formatting or use MarshalIndent)

- [ ] **Port selection strategy**
  - [ ] Auto-increment from 8065 based on existing worktrees
  - [ ] OR use hash of branch name for consistent ports
  - [ ] OR allow explicit port specification via flags
  - [ ] Validate port ranges (1-65535)
  - [ ] Optional: check if port is in use

#### 6. Enhanced Removal

- [ ] **Detect dual-repo worktrees**
  - [ ] Function: `isMattermostDualWorktree(path string) bool`
  - [ ] Check for both `server/` and `enterprise/` worktrees

- [ ] **Remove from both repos**
  - [ ] Remove worktree from `~/workspace/mattermost`
  - [ ] Remove worktree from `~/workspace/enterprise`
  - [ ] Remove directory structure

- [ ] **Optional branch deletion**
  - [ ] Flag: `--delete-branch` or `--delete-branches`
  - [ ] Delete from both mattermost and enterprise repos
  - [ ] Confirmation prompt (unless `--force`)

#### 7. Enhanced List Command

- [ ] **Detect and display dual-repo worktrees**
  - [ ] Add `IsDualRepo` field to `WorktreeInfo`
  - [ ] Display indicator: `[dual: ✓]` or `[single]`
  - [ ] Show both branch names if they differ

#### 8. Validation

- [ ] **Pre-flight checks**
  - [ ] Verify `~/workspace/mattermost` exists and is git repo
  - [ ] Verify `~/workspace/enterprise` exists and is git repo
  - [ ] Verify `~/workspace/worktrees` is accessible

- [ ] **Branch name validation**
  - [ ] Check for invalid git branch characters
  - [ ] Sanitize for filesystem paths (already done in config.go)

#### 9. Error Handling

- [ ] **Rollback on failure**
  - [ ] Track what's been created
  - [ ] Remove server worktree if enterprise fails
  - [ ] Clean up target directory
  - [ ] Prune worktrees from both repos

- [ ] **User-friendly errors**
  - [ ] Clear messages for missing repos
  - [ ] Guidance when repos not found
  - [ ] Show which step failed in dual-repo creation

#### 10. Post-Setup Commands

- [ ] **Extend existing post-setup**
  - [ ] Already runs `make setup-go-work` for mattermost
  - [ ] May need adjustments for dual-repo structure
  - [ ] Verify it runs in correct directory (`server/server/`)

#### 11. Clean Command Enhancement

- [ ] **Handle dual-repo worktrees**
  - [ ] Properly detect stale dual-repo worktrees
  - [ ] Remove from both repos
  - [ ] Check dirty status in both server/ and enterprise/

#### 12. Documentation

- [ ] **Update README.md**
  - [ ] Document new `wt co-mm` command
  - [ ] Explain dual-repo structure
  - [ ] Show examples with flags

- [ ] **Update help text**
  - [ ] Add mattermost commands to help output
  - [ ] Document port configuration options

#### 13. Testing Considerations

- [ ] Test with existing branch (both repos)
- [ ] Test with branch only in one repo
- [ ] Test with branch in neither repo (create new)
- [ ] Test removal of dual-repo worktrees
- [ ] Test error handling and rollback
- [ ] Test port configuration
- [ ] Test file copying

## Key Edge Cases

### Mattermost-Specific

1. **Branch exists in mattermost but not enterprise** → Create new in enterprise from base
2. **Branch exists in enterprise but not mattermost** → Create new in mattermost from base
3. **Branch exists in both repos** → Use both existing branches
4. **Branch doesn't exist in either** → Create new in both from same base branch
5. **Directory already exists** → Fail fast with clear error
6. **Only one repo found** → Error with guidance on setup
7. **config.json doesn't exist after copying** → Warn but continue
8. **Ports already in use** → Optionally detect and suggest alternatives

### File Operations

9. **go.work* glob matches no files** → Skip silently (optional file)
10. **Source file doesn't exist** → Skip with warning
11. **Destination directory needs creation** → Create parent directories
12. **Permission errors** → Fail with clear error message

### Git Operations

13. **Worktree already exists for branch** → Detect and just switch to it
14. **Base branch doesn't exist** → Error with list of available branches
15. **Git command fails** → Rollback and show git error output
16. **Detached HEAD state** → Handle gracefully

### Removal Edge Cases

17. **Removing currently active worktree** → Warn user
18. **Dirty worktree without --force** → Require --force flag
19. **Worktree has unpushed commits** → Warn before deletion
20. **Partial cleanup after failed creation** → Ensure no orphaned state

## Configuration Approach

### File Mappings

**Struct Definition:**
```go
type FileCopyConfig struct {
    SourceGlob      string
    DestinationPath string
    Required        bool  // If true, fail if source doesn't exist
}

var mattermostServerFiles = []FileCopyConfig{
    {"server/go.work*", "server/server/", false},
    {"webapp/.dir-locals.el", "server/webapp/.dir-locals.el", false},
    {"config/config.json", "server/config/config.json", true},
    {"docker-compose.override.yaml", "server/docker-compose.override.yaml", false},
    {"server/config.override.mk", "server/server/config.override.mk", false},
}

var enterpriseFiles = []FileCopyConfig{
    {"go.work*", "enterprise/", false},
}
```

### Port Strategy

**Recommended Approach:**
```go
func getAvailablePorts(existingWorktrees []WorktreeInfo) (serverPort, metricsPort int) {
    baseServerPort := 8065
    baseMetricsPort := 8067
    
    // Find highest used port
    maxServerPort := baseServerPort
    for _, wt := range existingWorktrees {
        if port := extractPortFromConfig(wt.Path); port > maxServerPort {
            maxServerPort = port
        }
    }
    
    // Increment from highest
    serverPort = maxServerPort + 1
    metricsPort = serverPort + 2  // Keep 2-port offset
    
    return serverPort, metricsPort
}
```

**Alternative (Consistent Hashing):**
```go
func getPortsFromBranchName(branch string) (serverPort, metricsPort int) {
    hash := fnv.New32a()
    hash.Write([]byte(branch))
    
    // Map to port range 8065-8164 (100 ports)
    portOffset := int(hash.Sum32() % 100)
    serverPort = 8065 + portOffset
    metricsPort = serverPort + 2
    
    return serverPort, metricsPort
}
```

## Summary of State Changes

### For Mattermost Dual-Repo Worktree Creation

**Created Artifacts:**
- New unified directory: `~/workspace/worktrees/mattermost-<branch>/`
- Server worktree: `~/workspace/worktrees/mattermost-<branch>/server/`
  - Git worktree from `~/workspace/mattermost`
- Enterprise worktree: `~/workspace/worktrees/mattermost-<branch>/enterprise/`
  - Git worktree from `~/workspace/enterprise`
- Copied base files (CLAUDE.md, mise.toml, etc.)
- Copied go.work* files
- Copied and modified `config.json` with unique ports

**Git Repository State Changes:**

In `~/workspace/mattermost/.git/`:
- New worktree entry in `.git/worktrees/`
- Reference to `~/workspace/worktrees/mattermost-<branch>/server/`

In `~/workspace/enterprise/.git/`:
- New worktree entry in `.git/worktrees/`
- Reference to `~/workspace/worktrees/mattermost-<branch>/enterprise/`

**Potentially New Branches:**
- If branch didn't exist, created in both repos from base branch

### Cleanup Requirements on Removal

1. Remove worktree from `~/workspace/mattermost` git
2. Remove worktree from `~/workspace/enterprise` git
3. Delete entire `~/workspace/worktrees/mattermost-<branch>/` directory
4. Prune orphaned worktree entries from both repos
5. Optionally delete branches from both repos (if `--delete-branch` specified)

## Command Examples

### Creating Mattermost Dual-Repo Worktree

```bash
# Using existing branch
wt co-mm MM-12345

# Creating new branch from master
wt co-mm MM-12345 -b master

# With custom ports
wt co-mm MM-12345 --port 8070 --metrics-port 8072

# Combined
wt co-mm feature/new-login -b develop --port 8080
```

### Removing Mattermost Dual-Repo Worktree

```bash
# Standard removal (with confirmation)
wt rm MM-12345

# Force removal (dirty worktree)
wt rm MM-12345 -f

# Remove and delete branches from both repos
wt rm MM-12345 --delete-branch

# Force remove and delete branches
wt rm MM-12345 -f --delete-branch
```

### Listing Worktrees

```bash
# List all worktrees for current repo
wt ls

# Output shows dual-repo indicator:
# Worktrees for mattermost:
#   MM-123    [dual: ✓] [clean]  (last commit: 2 days ago)
#   MM-456    [dual: ✓] [dirty]  (last commit: today)
```

## Integration with Existing `wt` Features

### Directory Switching
- Already works with `__WT_CD__` marker
- Should output path to unified worktree root: `~/workspace/worktrees/mattermost-MM-123/`

### Cursor Integration
- Add `wt cursor-mm <branch>` or make `wt cursor` detect dual-repo worktrees
- Open Cursor at worktree root to see both server/ and enterprise/

### Completions
- Add completions for `wt co-mm` command
- Show existing dual-repo worktrees first
- Then show branches from mattermost repo

### Smart cd
- Already implemented, works with worktree directories
- `cd ..` from `~/workspace/worktrees/mattermost-MM-123/` → `~/workspace`

