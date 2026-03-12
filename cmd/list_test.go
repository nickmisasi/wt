package cmd

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/nickmisasi/wt/internal"
)

func TestRunListJSONEmpty(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	setupGitRepo(t, repoPath)
	withCwd(t, repoPath)

	cfg := &internal.Config{
		WorktreeBasePath: filepath.Join(tempDir, "worktrees"),
		RepoName:         "repo",
		RepoRoot:         repoPath,
	}

	stdout, err := captureStdout(t, func() error {
		return RunList(cfg, false, true)
	})
	if err != nil {
		t.Fatalf("RunList returned error: %v", err)
	}

	var items []listJSONItem
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput=%q", err, stdout)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty array, got %d items", len(items))
	}
}

func TestRunListJSONShape(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	setupGitRepo(t, repoPath)
	withCwd(t, repoPath)

	cfg := &internal.Config{
		WorktreeBasePath: filepath.Join(tempDir, "worktrees"),
		RepoName:         "repo",
		RepoRoot:         repoPath,
	}

	worktreePath, err := internal.CreateWorktree(cfg, "feature/json-list", true, "main")
	if err != nil {
		t.Fatalf("failed to create test worktree: %v", err)
	}

	stdout, err := captureStdout(t, func() error {
		return RunList(cfg, false, true)
	})
	if err != nil {
		t.Fatalf("RunList returned error: %v", err)
	}

	var items []listJSONItem
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput=%q", err, stdout)
	}
	if len(items) != 1 {
		t.Fatalf("expected one item, got %d", len(items))
	}

	item := items[0]
	if item.Branch != "feature/json-list" {
		t.Fatalf("unexpected branch: %s", item.Branch)
	}
	if canonicalPath(item.Path) != canonicalPath(worktreePath) {
		t.Fatalf("unexpected path: %s", item.Path)
	}
	if item.LastCommitUnix <= 0 {
		t.Fatalf("expected lastCommitUnix > 0, got %d", item.LastCommitUnix)
	}
}
