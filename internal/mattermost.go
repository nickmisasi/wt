package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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

	// Track what we've created for cleanup
	var serverWorktreeCreated, enterpriseWorktreeCreated bool

	cleanup := func() {
		if serverWorktreeCreated {
			removeWorktreeFromRepo(mc.MattermostPath, filepath.Join(targetDir, "server"), true)
		}
		if enterpriseWorktreeCreated {
			removeWorktreeFromRepo(mc.EnterprisePath, filepath.Join(targetDir, "enterprise"), true)
		}
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
	sanitizedBranch := SanitizeBranchName(branch)
	mattermostWorktreePath := filepath.Join(targetDir, "mattermost-"+sanitizedBranch)
	if err := createWorktreeForRepo(mattermostRepo, branch, baseBranch, mattermostWorktreePath); err != nil {
		cleanup()
		return "", fmt.Errorf("failed to create mattermost worktree: %w", err)
	}
	serverWorktreeCreated = true

	// Create enterprise worktree at enterprise-<branch>/
	fmt.Printf("Creating enterprise worktree for branch: %s\n", branch)
	enterpriseWorktreePath := filepath.Join(targetDir, "enterprise-"+sanitizedBranch)
	if err := createWorktreeForRepo(enterpriseRepo, branch, baseBranch, enterpriseWorktreePath); err != nil {
		cleanup()
		return "", fmt.Errorf("failed to create enterprise worktree: %w", err)
	}
	enterpriseWorktreeCreated = true

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

	// Update ServiceSettings
	if serviceSettings, ok := config["ServiceSettings"].(map[string]interface{}); ok {
		serviceSettings["ListenAddress"] = fmt.Sprintf(":%d", serverPort)
		serviceSettings["SiteURL"] = fmt.Sprintf("http://localhost:%d", serverPort)
	}

	// Update MetricsSettings
	if metricsSettings, ok := config["MetricsSettings"].(map[string]interface{}); ok {
		metricsSettings["ListenAddress"] = fmt.Sprintf(":%d", metricsPort)
	}

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

// GetAvailablePorts returns available ports based on existing worktrees
func GetAvailablePorts(existingWorktrees []WorktreeInfo) (serverPort, metricsPort int) {
	baseServerPort := 8066 // Start at 8066, leaving 8065 for main repo
	maxServerPort := baseServerPort

	// Find highest used port from existing worktrees
	for _, wt := range existingWorktrees {
		if IsMattermostDualWorktree(wt.Path) {
			// Find the mattermost-* directory
			entries, err := os.ReadDir(wt.Path)
			if err != nil {
				continue
			}
			
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), "mattermost-") {
					configPath := filepath.Join(wt.Path, entry.Name(), "server", "config", "config.json")
					if port := extractPortFromConfig(configPath); port > maxServerPort {
						maxServerPort = port
					}
					break
				}
			}
		}
	}

	// If we found worktrees, increment from highest; otherwise use base
	if maxServerPort > baseServerPort {
		serverPort = maxServerPort + 1
	} else {
		serverPort = baseServerPort
	}
	metricsPort = serverPort + 2 // Keep 2-port offset

	return serverPort, metricsPort
}

// extractPortFromConfig reads the server port from config.json
func extractPortFromConfig(configPath string) int {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return 0
	}

	var config MattermostServerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return 0
	}

	if listenAddr, ok := config.ServiceSettings["ListenAddress"].(string); ok {
		var port int
		if _, err := fmt.Sscanf(listenAddr, ":%d", &port); err == nil {
			return port
		}
	}

	return 0
}
