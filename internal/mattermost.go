package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Port selection constants for Mattermost worktrees
const (
	// PortRangeStart is the beginning of the port range for worktree allocation
	PortRangeStart = 8100

	// PortRangeEnd is the end of the port range for worktree allocation (inclusive)
	PortRangeEnd = 8999

	// MainRepoPort is the port used by the main mattermost repository (excluded from allocation)
	MainRepoPort = 8065

	// MetricsPortOffset is added to the server port to get the metrics port
	// This is 2 to match the main Mattermost repo convention (8065 server → 8067 metrics)
	MetricsPortOffset = 2

	// PortRandomRetries is the number of random attempts before falling back to sequential scan
	PortRandomRetries = 50
)

// ExcludedPorts contains ports that should never be allocated to worktrees
var ExcludedPorts = map[int]bool{
	MainRepoPort:     true, // Main repo server port
	MainRepoPort + 2: true, // Main repo metrics port (8067)
}

// MattermostConfig holds configuration for Mattermost dual-repo worktrees
type MattermostConfig struct {
	WorkspaceRoot    string // e.g., ~/workspace
	MattermostPath   string // e.g., ~/workspace/mattermost
	EnterprisePath   string // e.g., ~/workspace/enterprise
	WorktreeBasePath string // e.g., ~/workspace/worktrees
	ServerPort       int
	MetricsPort      int
}

// FileCopyConfig defines files to copy with glob support
type FileCopyConfig struct {
	SourceGlob      string
	DestinationPath string
	Required        bool
}

// Mattermost file mappings (paths will be prefixed with branch-specific directory names)
var mattermostServerFiles = []FileCopyConfig{
	{"server/go.work*", "server/", false},
	{"webapp/.dir-locals.el", "webapp/.dir-locals.el", false},
	{"server/config/config.json", "server/config/config.json", true},
	{"docker-compose.override.yaml", "docker-compose.override.yaml", false},
	{"server/config.override.mk", "server/config.override.mk", false},
}

var enterpriseFiles = []FileCopyConfig{
	{"go.work*", "", false},
}

// IsMattermostRepo checks if the given repo is the mattermost repository
func IsMattermostRepo(repo *GitRepo) bool {
	// Check if repo name is "mattermost"
	if repo.Name != "mattermost" {
		return false
	}

	// Additional validation: check if enterprise repo exists alongside it
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	workspaceRoot := filepath.Join(homeDir, "workspace")
	enterprisePath := filepath.Join(workspaceRoot, "enterprise")

	// If enterprise repo exists, this is definitely the mattermost setup
	return isGitRepo(enterprisePath)
}

// NewMattermostConfig creates a new Mattermost configuration
func NewMattermostConfig() (*MattermostConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	workspaceRoot := filepath.Join(homeDir, "workspace")

	return &MattermostConfig{
		WorkspaceRoot:    workspaceRoot,
		MattermostPath:   filepath.Join(workspaceRoot, "mattermost"),
		EnterprisePath:   filepath.Join(workspaceRoot, "enterprise"),
		WorktreeBasePath: filepath.Join(workspaceRoot, "worktrees"),
		ServerPort:       8065,
		MetricsPort:      8067,
	}, nil
}

// ValidateMattermostSetup checks if the required repositories exist
func (mc *MattermostConfig) ValidateMattermostSetup() error {
	if !isGitRepo(mc.MattermostPath) {
		return fmt.Errorf("mattermost repository not found at %s\n\nPlease ensure you have cloned mattermost/mattermost to ~/workspace/mattermost", mc.MattermostPath)
	}

	if !isGitRepo(mc.EnterprisePath) {
		return fmt.Errorf("enterprise repository not found at %s\n\nPlease ensure you have cloned mattermost/enterprise to ~/workspace/enterprise", mc.EnterprisePath)
	}

	// Ensure worktrees directory exists
	if err := os.MkdirAll(mc.WorktreeBasePath, 0755); err != nil {
		return fmt.Errorf("cannot create worktrees directory: %w", err)
	}

	return nil
}

// isGitRepo checks if a path is a git repository
func isGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && (info.IsDir() || info.Mode().IsRegular())
}

// GetMattermostWorktreePath returns the path for a Mattermost dual-repo worktree
func (mc *MattermostConfig) GetMattermostWorktreePath(branch string) string {
	sanitized := SanitizeBranchName(branch)
	worktreeName := "mattermost-" + sanitized
	return filepath.Join(mc.WorktreeBasePath, worktreeName)
}

// IsMattermostDualWorktree checks if a path is a Mattermost dual-repo worktree
func IsMattermostDualWorktree(worktreePath string) bool {
	// Check for directories matching pattern mattermost-* and enterprise-*
	entries, err := os.ReadDir(worktreePath)
	if err != nil {
		return false
	}

	hasMattermost := false
	hasEnterprise := false

	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			if strings.HasPrefix(name, "mattermost-") {
				path := filepath.Join(worktreePath, name)
				if isGitWorktree(path) {
					hasMattermost = true
				}
			} else if strings.HasPrefix(name, "enterprise-") {
				path := filepath.Join(worktreePath, name)
				if isGitWorktree(path) {
					hasEnterprise = true
				}
			}
		}
	}

	return hasMattermost && hasEnterprise
}

// isGitWorktree checks if a directory is a git worktree
func isGitWorktree(path string) bool {
	gitFile := filepath.Join(path, ".git")
	info, err := os.Stat(gitFile)
	if err != nil {
		return false
	}

	// Worktrees have a .git file (not directory) that points to the main repo
	if info.Mode().IsRegular() {
		data, err := os.ReadFile(gitFile)
		if err == nil && strings.HasPrefix(string(data), "gitdir:") {
			return true
		}
	}

	// Could also be a directory for the main repo
	return info.IsDir()
}

// CreateMattermostDualWorktree creates a unified worktree with both repos
func CreateMattermostDualWorktree(mc *MattermostConfig, branch string, baseBranch string) (string, error) {
	targetDir := mc.GetMattermostWorktreePath(branch)

	// Check if worktree already exists
	if _, err := os.Stat(targetDir); err == nil {
		return targetDir, fmt.Errorf("worktree directory already exists: %s", targetDir)
	}

	// Calculate paths upfront
	sanitizedBranch := SanitizeBranchName(branch)
	mattermostWorktreePath := filepath.Join(targetDir, "mattermost-"+sanitizedBranch)
	enterpriseWorktreePath := filepath.Join(targetDir, "enterprise-"+sanitizedBranch)

	// Prune any orphaned worktree references before starting
	// This handles the case where a previous creation failed
	exec.Command("git", "-C", mc.MattermostPath, "worktree", "prune").Run()
	exec.Command("git", "-C", mc.EnterprisePath, "worktree", "prune").Run()

	// Track what we've created for cleanup
	var serverWorktreeCreated, enterpriseWorktreeCreated bool

	cleanup := func() {
		// Remove worktrees from git
		if serverWorktreeCreated {
			removeWorktreeFromRepo(mc.MattermostPath, mattermostWorktreePath, true)
		}
		if enterpriseWorktreeCreated {
			removeWorktreeFromRepo(mc.EnterprisePath, enterpriseWorktreePath, true)
		}
		// Always prune to clean up git's internal state
		exec.Command("git", "-C", mc.MattermostPath, "worktree", "prune").Run()
		exec.Command("git", "-C", mc.EnterprisePath, "worktree", "prune").Run()
		// Remove directory
		if targetDir != "" {
			os.RemoveAll(targetDir)
		}
	}

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create target directory: %w", err)
	}

	// Copy base files from mattermost repo
	fmt.Println("Copying base configuration files...")
	if err := copyFilesExcept(mc.MattermostPath, targetDir, []string{"server", "webapp", ".git"}); err != nil {
		cleanup()
		return "", fmt.Errorf("failed to copy base files: %w", err)
	}

	// Create GitRepo instances
	mattermostRepo := &GitRepo{Root: mc.MattermostPath, Name: "mattermost"}
	enterpriseRepo := &GitRepo{Root: mc.EnterprisePath, Name: "enterprise"}

	// Determine base branch if not specified
	if baseBranch == "" {
		baseBranch = mattermostRepo.GetDefaultBranch()
	}

	// Create mattermost worktree at mattermost-<branch>/
	fmt.Printf("Creating mattermost worktree for branch: %s\n", branch)
	if err := createWorktreeForRepo(mattermostRepo, branch, baseBranch, mattermostWorktreePath); err != nil {
		cleanup()
		return "", fmt.Errorf("failed to create mattermost worktree: %w", err)
	}
	serverWorktreeCreated = true

	// Create enterprise worktree at enterprise-<branch>/
	fmt.Printf("Creating enterprise worktree for branch: %s\n", branch)
	if err := createWorktreeForRepo(enterpriseRepo, branch, baseBranch, enterpriseWorktreePath); err != nil {
		cleanup()
		// Check if this is an "already used by worktree" error
		if strings.Contains(err.Error(), "already used by worktree") {
			return "", fmt.Errorf("failed to create enterprise worktree: %w\n\nTo fix this, run these commands:\n  cd ~/workspace/enterprise\n  git worktree prune\n\nThen try again", err)
		}
		return "", fmt.Errorf("failed to create enterprise worktree: %w", err)
	}
	enterpriseWorktreeCreated = true

	// Create symlinks for compatibility with make and other scripts
	// These allow scripts that reference ../../enterprise to still work
	fmt.Println("Creating compatibility symlinks...")
	mattermostSymlink := filepath.Join(targetDir, "mattermost")
	enterpriseSymlink := filepath.Join(targetDir, "enterprise")
	
	if err := os.Symlink("mattermost-"+sanitizedBranch, mattermostSymlink); err != nil {
		cleanup()
		return "", fmt.Errorf("failed to create mattermost symlink: %w", err)
	}
	
	if err := os.Symlink("enterprise-"+sanitizedBranch, enterpriseSymlink); err != nil {
		cleanup()
		return "", fmt.Errorf("failed to create enterprise symlink: %w", err)
	}

	// Copy additional files
	fmt.Println("Copying additional configuration files...")
	if err := copyMattermostFiles(mc, targetDir, sanitizedBranch); err != nil {
		cleanup()
		return "", fmt.Errorf("failed to copy additional files: %w", err)
	}

	// Update config.json with unique ports
	configPath := filepath.Join(targetDir, "mattermost-"+sanitizedBranch, "server", "config", "config.json")
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Configuring server ports (server: %d, metrics: %d)...\n", mc.ServerPort, mc.MetricsPort)
		if err := updateConfigPorts(configPath, mc.ServerPort, mc.MetricsPort); err != nil {
			// Non-fatal error
			fmt.Printf("Warning: failed to update ports in config.json: %v\n", err)
		}
	} else {
		fmt.Println("Note: config.json not found, skipping port configuration")
	}

	return targetDir, nil
}

// createWorktreeForRepo creates a worktree from a repository
func createWorktreeForRepo(repo *GitRepo, branch, baseBranch, worktreePath string) error {
	// Check if branch exists in this specific repository using -C flag
	localExists := checkBranchExists(repo.Root, branch)
	remoteExists := checkRemoteBranchExists(repo.Root, branch)

	var cmd *exec.Cmd

	if localExists {
		// Branch exists locally and is verified
		fmt.Printf("  → Using existing local branch in %s\n", repo.Name)
		cmd = exec.Command("git", "-C", repo.Root, "worktree", "add", worktreePath, branch)
	} else if remoteExists {
		// Branch exists on remote - create tracking branch
		fmt.Printf("  → Branch exists on remote, creating tracking branch in %s\n", repo.Name)
		cmd = exec.Command("git", "-C", repo.Root, "worktree", "add", "--track", "-b", branch, worktreePath, "origin/"+branch)
	} else {
		// Branch doesn't exist - create new branch from base
		// Verify base branch exists
		verifyBaseCmd := exec.Command("git", "-C", repo.Root, "rev-parse", "--verify", baseBranch)
		if err := verifyBaseCmd.Run(); err != nil {
			// Base branch doesn't exist locally, try origin/baseBranch
			verifyOriginBaseCmd := exec.Command("git", "-C", repo.Root, "rev-parse", "--verify", "origin/"+baseBranch)
			if err := verifyOriginBaseCmd.Run(); err != nil {
				return fmt.Errorf("base branch '%s' not found in %s (tried local and origin/%s)", baseBranch, repo.Name, baseBranch)
			}
			baseBranch = "origin/" + baseBranch
		}

		fmt.Printf("  → Creating new branch from %s in %s\n", baseBranch, repo.Name)
		cmd = exec.Command("git", "-C", repo.Root, "worktree", "add", "-b", branch, worktreePath, baseBranch)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree add failed: %s", string(output))
	}

	return nil
}

// checkBranchExists checks if a branch exists locally in a specific repository
func checkBranchExists(repoPath, branch string) bool {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", "--quiet", branch)
	return cmd.Run() == nil
}

// checkRemoteBranchExists checks if a branch exists on remote in a specific repository
func checkRemoteBranchExists(repoPath, branch string) bool {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", "--quiet", "origin/"+branch)
	return cmd.Run() == nil
}

// copyFilesExcept copies all files from src to dst except those in the exclusion list
func copyFilesExcept(src, dst string, exclusions []string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip exclusions
		skip := false
		for _, excl := range exclusions {
			if name == excl {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Skip hidden files except .gitignore
		if strings.HasPrefix(name, ".") && name != ".gitignore" {
			continue
		}

		srcPath := filepath.Join(src, name)
		dstPath := filepath.Join(dst, name)

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Copy permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// copyMattermostFiles copies additional files based on file mappings
func copyMattermostFiles(mc *MattermostConfig, targetDir string, sanitizedBranch string) error {
	mattermostDirName := "mattermost-" + sanitizedBranch
	enterpriseDirName := "enterprise-" + sanitizedBranch

	// Copy mattermost server files
	for _, mapping := range mattermostServerFiles {
		srcPattern := filepath.Join(mc.MattermostPath, mapping.SourceGlob)
		matches, err := filepath.Glob(srcPattern)
		if err != nil {
			return fmt.Errorf("glob pattern error: %w", err)
		}

		if len(matches) == 0 {
			if mapping.Required {
				return fmt.Errorf("required file not found: %s", mapping.SourceGlob)
			}
			continue
		}

		for _, srcPath := range matches {
			// Determine destination with branch-specific directory
			var dstPath string
			if strings.HasSuffix(mapping.DestinationPath, "/") {
				// Destination is a directory
				dstPath = filepath.Join(targetDir, mattermostDirName, mapping.DestinationPath, filepath.Base(srcPath))
			} else {
				// Destination is a file
				dstPath = filepath.Join(targetDir, mattermostDirName, mapping.DestinationPath)
			}

			if err := copyFile(srcPath, dstPath); err != nil {
				if mapping.Required {
					return fmt.Errorf("failed to copy required file %s: %w", srcPath, err)
				}
				fmt.Printf("  Warning: failed to copy %s: %v\n", srcPath, err)
			}
		}
	}

	// Copy enterprise files
	for _, mapping := range enterpriseFiles {
		srcPattern := filepath.Join(mc.EnterprisePath, mapping.SourceGlob)
		matches, err := filepath.Glob(srcPattern)
		if err != nil {
			return fmt.Errorf("glob pattern error: %w", err)
		}

		if len(matches) == 0 {
			if mapping.Required {
				return fmt.Errorf("required file not found: %s", mapping.SourceGlob)
			}
			continue
		}

		for _, srcPath := range matches {
			var dstPath string
			if mapping.DestinationPath == "" {
				// Copy to enterprise directory root
				dstPath = filepath.Join(targetDir, enterpriseDirName, filepath.Base(srcPath))
			} else if strings.HasSuffix(mapping.DestinationPath, "/") {
				dstPath = filepath.Join(targetDir, enterpriseDirName, mapping.DestinationPath, filepath.Base(srcPath))
			} else {
				dstPath = filepath.Join(targetDir, enterpriseDirName, mapping.DestinationPath)
			}

			if err := copyFile(srcPath, dstPath); err != nil {
				if mapping.Required {
					return fmt.Errorf("failed to copy required file %s: %w", srcPath, err)
				}
				fmt.Printf("  Warning: failed to copy %s: %v\n", srcPath, err)
			}
		}
	}

	return nil
}

// MattermostServerConfig represents the structure of Mattermost's config.json
type MattermostServerConfig struct {
	ServiceSettings map[string]interface{} `json:"ServiceSettings"`
	MetricsSettings map[string]interface{} `json:"MetricsSettings"`
}

// updateConfigPorts updates the ports in config.json
func updateConfigPorts(configPath string, serverPort, metricsPort int) error {
	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	// Parse as generic JSON to preserve all fields
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// Get or create ServiceSettings
	serviceSettings, ok := config["ServiceSettings"].(map[string]interface{})
	if !ok {
		// ServiceSettings doesn't exist or is not a map - create it
		serviceSettings = make(map[string]interface{})
		config["ServiceSettings"] = serviceSettings
	}
	serviceSettings["ListenAddress"] = fmt.Sprintf(":%d", serverPort)
	serviceSettings["SiteURL"] = fmt.Sprintf("http://localhost:%d", serverPort)

	// Get or create MetricsSettings
	metricsSettings, ok := config["MetricsSettings"].(map[string]interface{})
	if !ok {
		// MetricsSettings doesn't exist or is not a map - create it
		metricsSettings = make(map[string]interface{})
		config["MetricsSettings"] = metricsSettings
	}
	metricsSettings["ListenAddress"] = fmt.Sprintf(":%d", metricsPort)

	// Write back with indentation
	updatedData, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, updatedData, 0644)
}

// RemoveMattermostDualWorktree removes a Mattermost dual-repo worktree
func RemoveMattermostDualWorktree(mc *MattermostConfig, branch string, force bool) error {
	worktreePath := mc.GetMattermostWorktreePath(branch)

	// Check if it exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return fmt.Errorf("worktree not found: %s", worktreePath)
	}

	// Check if it's a dual-repo worktree
	if !IsMattermostDualWorktree(worktreePath) {
		return fmt.Errorf("not a Mattermost dual-repo worktree: %s", worktreePath)
	}

	// Find the actual directory names (they include the branch name)
	entries, err := os.ReadDir(worktreePath)
	if err != nil {
		return fmt.Errorf("failed to read worktree directory: %w", err)
	}

	var mattermostPath, enterprisePath string
	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			if strings.HasPrefix(name, "mattermost-") {
				mattermostPath = filepath.Join(worktreePath, name)
			} else if strings.HasPrefix(name, "enterprise-") {
				enterprisePath = filepath.Join(worktreePath, name)
			}
		}
	}

	// Remove mattermost worktree
	if mattermostPath != "" {
		fmt.Println("Removing mattermost worktree...")
		if err := removeWorktreeFromRepo(mc.MattermostPath, mattermostPath, force); err != nil {
			return fmt.Errorf("failed to remove mattermost worktree: %w", err)
		}
	}

	// Remove enterprise worktree
	if enterprisePath != "" {
		fmt.Println("Removing enterprise worktree...")
		if err := removeWorktreeFromRepo(mc.EnterprisePath, enterprisePath, force); err != nil {
			return fmt.Errorf("failed to remove enterprise worktree: %w", err)
		}
	}

	// Remove directory structure
	fmt.Printf("Removing directory: %s\n", worktreePath)
	return os.RemoveAll(worktreePath)
}

// removeWorktreeFromRepo removes a worktree from a repository
func removeWorktreeFromRepo(repoPath, worktreePath string, force bool) error {
	args := []string{"-C", repoPath, "worktree", "remove"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, worktreePath)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove failed: %s", string(output))
	}

	return nil
}

// DeleteBranchFromRepos deletes a branch from both mattermost and enterprise repos
func DeleteBranchFromRepos(mc *MattermostConfig, branch string) error {
	errors := []string{}

	// Delete from mattermost repo
	cmd := exec.Command("git", "-C", mc.MattermostPath, "branch", "-D", branch)
	if output, err := cmd.CombinedOutput(); err != nil {
		errors = append(errors, fmt.Sprintf("mattermost: %s", string(output)))
	} else {
		fmt.Printf("Deleted branch '%s' from mattermost repository\n", branch)
	}

	// Delete from enterprise repo
	cmd = exec.Command("git", "-C", mc.EnterprisePath, "branch", "-D", branch)
	if output, err := cmd.CombinedOutput(); err != nil {
		errors = append(errors, fmt.Sprintf("enterprise: %s", string(output)))
	} else {
		fmt.Printf("Deleted branch '%s' from enterprise repository\n", branch)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to delete branches:\n  %s", strings.Join(errors, "\n  "))
	}

	return nil
}

// IsPortAvailable checks if a port is available for use on localhost.
// It attempts to listen on the port and returns true if successful (port is free),
// or false if the port is already in use.
func IsPortAvailable(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// PortPair represents a server port and its associated metrics port
type PortPair struct {
	ServerPort  int
	MetricsPort int
}

// GetReservedPorts extracts all ports currently used by existing Mattermost worktrees.
// It returns a map of port numbers that are reserved (both server and metrics ports).
// Missing or invalid config files are tolerated (logged but don't cause errors).
func GetReservedPorts(existingWorktrees []WorktreeInfo) map[int]bool {
	reserved := make(map[int]bool)

	// Copy excluded ports into the reserved set
	for port := range ExcludedPorts {
		reserved[port] = true
	}

	for _, wt := range existingWorktrees {
		if !IsMattermostDualWorktree(wt.Path) {
			continue
		}

		// Find the mattermost-* directory
		entries, err := os.ReadDir(wt.Path)
		if err != nil {
			// Tolerate missing directories
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), "mattermost-") {
				configPath := filepath.Join(wt.Path, entry.Name(), "server", "config", "config.json")
				portPair := ExtractPortPairFromConfig(configPath)
				if portPair.ServerPort > 0 {
					reserved[portPair.ServerPort] = true
				}
				if portPair.MetricsPort > 0 {
					reserved[portPair.MetricsPort] = true
				}
				break
			}
		}
	}

	return reserved
}

// FindMattermostConfig finds the path to config.json in a worktree or repo
func FindMattermostConfig(root string) (string, string, error) {
	// 1. Check if we are in a Mattermost dual worktree
	isDual := IsMattermostDualWorktree(root)

	if isDual {
		// Find the mattermost-* directory
		entries, err := os.ReadDir(root)
		if err != nil {
			return "", "", fmt.Errorf("failed to read worktree directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), "mattermost-") {
				serverDir := filepath.Join(root, entry.Name(), "server")
				configPath := filepath.Join(serverDir, "config", "config.json")
				return serverDir, configPath, nil
			}
		}

		return "", "", fmt.Errorf("could not find mattermost server directory in dual worktree")
	}

	// 2. Check if we are in a standard Mattermost repo or a single worktree
	// We expect a server/config/config.json relative to root
	candidateServerDir := filepath.Join(root, "server")
	candidateConfig := filepath.Join(candidateServerDir, "config", "config.json")

	if _, err := os.Stat(candidateConfig); err == nil {
		return candidateServerDir, candidateConfig, nil
	}

	return "", "", fmt.Errorf("not a recognized Mattermost worktree (config.json not found)")
}

// ExtractPortPairFromConfig reads both server and metrics ports from config.json
func ExtractPortPairFromConfig(configPath string) PortPair {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return PortPair{}
	}

	var config MattermostServerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return PortPair{}
	}

	var pair PortPair

	// Extract server port from ServiceSettings.ListenAddress
	if listenAddr, ok := config.ServiceSettings["ListenAddress"].(string); ok {
		fmt.Sscanf(listenAddr, ":%d", &pair.ServerPort)
	}

	// Extract metrics port from MetricsSettings.ListenAddress
	if config.MetricsSettings != nil {
		if listenAddr, ok := config.MetricsSettings["ListenAddress"].(string); ok {
			fmt.Sscanf(listenAddr, ":%d", &pair.MetricsPort)
		}
	}

	return pair
}

// isPortPairAvailable checks if both the server port and metrics port are available.
// A port pair is available if:
// 1. Neither port is in the reserved set
// 2. Neither port is currently in use on localhost
func isPortPairAvailable(serverPort int, reserved map[int]bool) bool {
	metricsPort := serverPort + MetricsPortOffset

	// Check if ports are reserved by existing worktrees
	if reserved[serverPort] || reserved[metricsPort] {
		return false
	}

	// Check if ports are currently in use on localhost
	if !IsPortAvailable(serverPort) || !IsPortAvailable(metricsPort) {
		return false
	}

	return true
}

// GetAvailablePorts returns available ports for a new Mattermost worktree.
// It uses a randomized search within the port range, validating that both
// server and metrics ports are free. Falls back to sequential scan if
// random attempts are exhausted.
func GetAvailablePorts(existingWorktrees []WorktreeInfo) (serverPort, metricsPort int) {
	return GetAvailablePortsWithRand(existingWorktrees, nil)
}

// GetAvailablePortsWithRand is like GetAvailablePorts but accepts a custom random
// source for deterministic testing. If rng is nil, a new random source is used.
func GetAvailablePortsWithRand(existingWorktrees []WorktreeInfo, rng *rand.Rand) (serverPort, metricsPort int) {
	reserved := GetReservedPorts(existingWorktrees)

	// Use provided RNG or create a new one
	if rng == nil {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}

	// Calculate the valid port range (accounting for metrics port offset)
	// Server port can be from PortRangeStart to (PortRangeEnd - MetricsPortOffset)
	// so that metrics port doesn't exceed PortRangeEnd
	maxServerPort := PortRangeEnd - MetricsPortOffset
	portRangeSize := maxServerPort - PortRangeStart + 1

	// Phase 1: Random selection attempts
	for attempt := 0; attempt < PortRandomRetries; attempt++ {
		candidatePort := PortRangeStart + rng.Intn(portRangeSize)
		if isPortPairAvailable(candidatePort, reserved) {
			return candidatePort, candidatePort + MetricsPortOffset
		}
	}

	// Phase 2: Sequential fallback scan
	// Start from a random position to avoid always returning the same port
	// when random attempts fail due to many reserved ports
	startOffset := rng.Intn(portRangeSize)
	for i := 0; i < portRangeSize; i++ {
		candidatePort := PortRangeStart + ((startOffset + i) % portRangeSize)
		if isPortPairAvailable(candidatePort, reserved) {
			return candidatePort, candidatePort + MetricsPortOffset
		}
	}

	// If all ports are exhausted, return a fallback (this should be rare)
	// Return 0, 0 to indicate no ports available
	return 0, 0
}

