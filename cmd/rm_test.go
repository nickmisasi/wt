package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/nickmisasi/wt/internal"
)

func TestRunRemoveJSONRemovesWorktree(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	setupGitRepo(t, repoPath)
	withCwd(t, repoPath)

	cfg := &internal.Config{
		WorktreeBasePath: filepath.Join(tempDir, "worktrees"),
		RepoName:         "repo",
		RepoRoot:         repoPath,
	}

	branch := "feature/remove-json"
	worktreePath, err := internal.CreateWorktree(cfg, branch, true, "main")
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	stdout, err := captureStdout(t, func() error {
		return RunRemove(cfg, branch, true, true)
	})
	if err != nil {
		t.Fatalf("RunRemove returned error: %v", err)
	}

	var payload removeJSONResponse
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput=%q", err, stdout)
	}

	if payload.Mode != "standard" {
		t.Fatalf("expected mode=standard, got %s", payload.Mode)
	}
	if !payload.Removed {
		t.Fatalf("expected removed=true, got false")
	}
	if canonicalPath(payload.WorktreePath) != canonicalPath(worktreePath) {
		t.Fatalf("unexpected worktree path: %s", payload.WorktreePath)
	}
	if len(payload.RemovedPaths) != 1 || canonicalPath(payload.RemovedPaths[0]) != canonicalPath(worktreePath) {
		t.Fatalf("unexpected removedPaths: %v", payload.RemovedPaths)
	}

	if _, statErr := os.Stat(worktreePath); !os.IsNotExist(statErr) {
		t.Fatalf("expected worktree path to be removed, stat err=%v", statErr)
	}
}

func TestRunRemoveJSONMissingWorktreeIsIdempotent(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	setupGitRepo(t, repoPath)
	withCwd(t, repoPath)

	cfg := &internal.Config{
		WorktreeBasePath: filepath.Join(tempDir, "worktrees"),
		RepoName:         "repo",
		RepoRoot:         repoPath,
	}
	branch := "feature/missing"

	stdout, err := captureStdout(t, func() error {
		return RunRemove(cfg, branch, true, true)
	})
	if err != nil {
		t.Fatalf("RunRemove returned error: %v", err)
	}

	var payload removeJSONResponse
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput=%q", err, stdout)
	}

	if payload.Removed {
		t.Fatalf("expected removed=false for missing worktree")
	}
	if payload.Branch != branch {
		t.Fatalf("expected branch %s, got %s", branch, payload.Branch)
	}
	if payload.WorktreePath != cfg.GetWorktreePath(branch) {
		t.Fatalf("unexpected worktreePath: %s", payload.WorktreePath)
	}
}
