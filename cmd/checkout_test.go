package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/nickmisasi/wt/internal"
)

func TestRunCheckoutJSONDryRunDoesNotMutate(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	setupGitRepo(t, repoPath)
	withCwd(t, repoPath)

	cfg := &internal.Config{
		WorktreeBasePath: filepath.Join(tempDir, "worktrees"),
		RepoName:         "repo",
		RepoRoot:         repoPath,
	}
	repo := &internal.GitRepo{Root: repoPath, Name: "repo"}
	branch := "feature/dry-run"
	expectedPath := cfg.GetWorktreePath(branch)

	stdout, err := captureStdout(t, func() error {
		return RunCheckout(cfg, repo, branch, "main", false, nil, true, true)
	})
	if err != nil {
		t.Fatalf("RunCheckout returned error: %v", err)
	}

	var payload checkoutJSONResponse
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput=%q", err, stdout)
	}

	if payload.Created {
		t.Fatalf("expected Created=false for dry-run")
	}
	if payload.Existing {
		t.Fatalf("expected Existing=false for new branch dry-run")
	}
	if payload.CdPath != expectedPath {
		t.Fatalf("unexpected cdPath: %s", payload.CdPath)
	}
	if payload.WorktreePath != expectedPath {
		t.Fatalf("unexpected worktreePath: %s", payload.WorktreePath)
	}

	if _, statErr := os.Stat(expectedPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected dry-run to avoid creating path, stat err=%v", statErr)
	}

	branchRef := runGitCommand(t, repoPath, "branch", "--list", branch)
	if branchRef != "" {
		t.Fatalf("expected dry-run not to create branch, got %q", branchRef)
	}
}

func TestRunCheckoutJSONExistingWorktree(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	setupGitRepo(t, repoPath)
	withCwd(t, repoPath)

	cfg := &internal.Config{
		WorktreeBasePath: filepath.Join(tempDir, "worktrees"),
		RepoName:         "repo",
		RepoRoot:         repoPath,
	}
	repo := &internal.GitRepo{Root: repoPath, Name: "repo"}
	branch := "feature/existing"
	worktreePath, err := internal.CreateWorktree(cfg, branch, true, "main")
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	stdout, err := captureStdout(t, func() error {
		return RunCheckout(cfg, repo, branch, "main", false, nil, true, false)
	})
	if err != nil {
		t.Fatalf("RunCheckout returned error: %v", err)
	}

	var payload checkoutJSONResponse
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput=%q", err, stdout)
	}

	if payload.Created {
		t.Fatalf("expected Created=false for existing worktree")
	}
	if !payload.Existing {
		t.Fatalf("expected Existing=true for existing worktree")
	}
	if canonicalPath(payload.CdPath) != canonicalPath(worktreePath) || canonicalPath(payload.WorktreePath) != canonicalPath(worktreePath) {
		t.Fatalf("unexpected paths in payload: %+v", payload)
	}
}

func TestRunCheckoutJSONIncludesPostSetupCommands(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	setupGitRepo(t, repoPath)
	withCwd(t, repoPath)

	scriptPath := filepath.Join(repoPath, enableClaudeDocsScript)
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}
	runGitCommand(t, repoPath, "add", enableClaudeDocsScript)
	runGitCommand(t, repoPath, "commit", "-m", "add script")

	cfg := &internal.Config{
		WorktreeBasePath: filepath.Join(tempDir, "worktrees"),
		RepoName:         "repo",
		RepoRoot:         repoPath,
	}
	repo := &internal.GitRepo{Root: repoPath, Name: "repo"}
	branch := "feature/json-post-setup"

	stdout, err := captureStdout(t, func() error {
		return RunCheckout(cfg, repo, branch, "main", false, nil, true, false)
	})
	if err != nil {
		t.Fatalf("RunCheckout returned error: %v", err)
	}

	var payload checkoutJSONResponse
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput=%q", err, stdout)
	}

	if !payload.Created || payload.Existing {
		t.Fatalf("expected created=true existing=false, got %+v", payload)
	}
	if len(payload.PostSetupCommands) == 0 {
		t.Fatalf("expected postSetupCommands to include enable-claude-docs command")
	}

	expectedCommand := "cd " + payload.WorktreePath + " && ./" + enableClaudeDocsScript
	found := false
	for _, command := range payload.PostSetupCommands {
		if command == expectedCommand {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected command %q in postSetupCommands: %v", expectedCommand, payload.PostSetupCommands)
	}
}

func TestIsClaudeAvailable(t *testing.T) {
	// This is a smoke test — it verifies the function runs without panicking.
	// The actual result depends on whether claude is installed on the test machine.
	_ = isClaudeAvailable()
}

func TestShouldCreateClaudemux(t *testing.T) {
	t.Run("explicit true overrides config", func(t *testing.T) {
		v := true
		if !shouldCreateClaudemux(&v) {
			t.Fatalf("expected shouldCreateClaudemux(&true) to return true")
		}
	})

	t.Run("explicit false overrides config", func(t *testing.T) {
		v := false
		if shouldCreateClaudemux(&v) {
			t.Fatalf("expected shouldCreateClaudemux(&false) to return false")
		}
	})

	t.Run("nil falls back to config default", func(t *testing.T) {
		// With no user config file, LoadUserConfig returns defaults where
		// Claudemux.Enabled is false, so shouldCreateClaudemux(nil) should
		// return false.
		result := shouldCreateClaudemux(nil)
		if result {
			t.Fatalf("expected shouldCreateClaudemux(nil) to return false (config default)")
		}
	})
}
