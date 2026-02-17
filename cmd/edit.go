package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nickmisasi/wt/internal"
)

// parseEditor splits an editor config string into the program name and any extra arguments.
func parseEditor(editor string) (program string, args []string) {
	parts := strings.Fields(editor)
	return parts[0], parts[1:]
}

// RunEditHere opens the configured editor on the current worktree (no branch argument needed)
func RunEditHere() error {
	// Load user config to get editor
	userCfg, err := internal.LoadUserConfig()
	if err != nil {
		return fmt.Errorf("failed to load user config: %w", err)
	}

	editor := userCfg.Editor.Command
	if editor == "" {
		return fmt.Errorf("no editor configured. Set one with: wt config set editor.command <editor>")
	}

	editorProgram, editorArgs := parseEditor(editor)

	if _, err := exec.LookPath(editorProgram); err != nil {
		return fmt.Errorf("editor %q not found in PATH", editorProgram)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	cfg, err := internal.NewConfig()
	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	if !strings.HasPrefix(cwd, cfg.WorktreeBasePath) {
		return fmt.Errorf("not in a worktree directory. Usage: wt edit <branch>")
	}

	// Extract worktree root (first path component under WorktreeBasePath)
	relPath, err := filepath.Rel(cfg.WorktreeBasePath, cwd)
	if err != nil {
		return fmt.Errorf("failed to determine relative path: %w", err)
	}

	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) == 0 || parts[0] == "." {
		return fmt.Errorf("not in a worktree directory. Usage: wt edit <branch>")
	}

	worktreeRoot := filepath.Join(cfg.WorktreeBasePath, parts[0])

	fmt.Printf("Opening %s in %s\n", editorProgram, worktreeRoot)
	cmd := exec.Command(editorProgram, append(editorArgs, worktreeRoot)...)
	return cmd.Start()
}

// RunEdit opens the user-configured editor for the given branch's worktree
func RunEdit(cfg *internal.Config, repo *internal.GitRepo, branch string, baseBranch string, noClaudeDocs bool) error {
	// Load user config to get editor
	userCfg, err := internal.LoadUserConfig()
	if err != nil {
		return fmt.Errorf("failed to load user config: %w", err)
	}

	editor := userCfg.Editor.Command
	if editor == "" {
		return fmt.Errorf("no editor configured. Set one with: wt config set editor.command <editor>")
	}

	// Check if editor program is available
	editorProgram, _ := parseEditor(editor)
	if _, err := exec.LookPath(editorProgram); err != nil {
		return fmt.Errorf("editor %q not found in PATH", editorProgram)
	}

	// Check if this is the mattermost repository
	if internal.IsMattermostRepo(repo) {
		return runMattermostEdit(repo, branch, baseBranch, noClaudeDocs, editor)
	}

	// Standard worktree edit workflow
	return runStandardEdit(cfg, repo, branch, baseBranch, noClaudeDocs, editor)
}

// runStandardEdit handles standard single-repo editor opening
func runStandardEdit(cfg *internal.Config, repo *internal.GitRepo, branch string, baseBranch string, noClaudeDocs bool, editor string) error {
	// Check if worktree already exists
	exists, path := internal.WorktreeExists(cfg, branch)
	worktreeCreated := false

	if !exists {
		fmt.Printf("Worktree doesn't exist for branch '%s'. Creating it...\n", branch)

		var err error
		path, err = ensureBranchAndCreateWorktree(cfg, repo, branch, baseBranch)
		if err != nil {
			return err
		}
		fmt.Printf("Worktree created at: %s\n", path)
		worktreeCreated = true
	}

	// Open editor
	editorProgram, editorArgs := parseEditor(editor)
	fmt.Printf("Opening %s for branch: %s\n", editorProgram, branch)
	cmd := exec.Command(editorProgram, append(editorArgs, path)...)
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", editorProgram, err)
	}

	// Optionally also switch directory
	fmt.Printf("%s%s\n", internal.CDMarker, path)

	// If we created a new worktree, check if there's a post-setup command
	if worktreeCreated {
		if postCmd := cfg.GetPostSetupCommand(path); postCmd != "" {
			fmt.Printf("%s%s\n", internal.CMDMarker, postCmd)
		}

		// Run enable-claude-docs.sh if it exists and not disabled
		if !noClaudeDocs {
			emitEnableClaudeDocsCommand(path)
		}
	}

	return nil
}

// runMattermostEdit handles Mattermost dual-repo editor opening
func runMattermostEdit(repo *internal.GitRepo, branch string, baseBranch string, noClaudeDocs bool, editor string) error {
	mc, err := internal.NewMattermostConfig()
	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	// Validate setup
	if err := mc.ValidateMattermostSetup(); err != nil {
		return err
	}

	worktreePath := mc.GetMattermostWorktreePath(branch)

	// Check if worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		// Create it first
		fmt.Printf("Worktree doesn't exist for branch '%s'. Creating it...\n\n", branch)
		if err := runMattermostCheckout(repo, branch, baseBranch, 0, 0, noClaudeDocs); err != nil {
			return err
		}
		// Refresh the worktree path
		worktreePath = mc.GetMattermostWorktreePath(branch)
	}

	// Open in editor
	editorProgram, editorArgs := parseEditor(editor)
	fmt.Printf("Opening %s for branch: %s\n", editorProgram, branch)

	cmd := exec.Command(editorProgram, append(editorArgs, worktreePath)...)
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", editorProgram, err)
	}

	// Switch directory
	fmt.Printf("%s%s\n", internal.CDMarker, worktreePath)

	return nil
}
