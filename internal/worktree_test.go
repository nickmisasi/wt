package internal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGit(t *testing.T, cwd string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
	return strings.TrimSpace(string(output))
}

func setupRepo(t *testing.T, repoPath string) {
	t.Helper()
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		t.Fatalf("failed to create repo directory: %v", err)
	}

	runGit(t, repoPath, "init", "-b", "main")
	runGit(t, repoPath, "config", "user.email", "test@example.com")
	runGit(t, repoPath, "config", "user.name", "Test User")

	readmePath := filepath.Join(repoPath, "README.md")
	if err := os.WriteFile(readmePath, []byte("test\n"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	runGit(t, repoPath, "add", "README.md")
	runGit(t, repoPath, "commit", "-m", "initial")
}

func withWorkingDirectory(t *testing.T, dir string) {
	t.Helper()
	current, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change cwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(current)
	})
}

func TestWorktreeExistsIgnoresSanitizedPathCollision(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	setupRepo(t, repoPath)
	withWorkingDirectory(t, repoPath)

	cfg := &Config{
		WorktreeBasePath: filepath.Join(tempDir, "worktrees"),
		RepoName:         "repo",
		RepoRoot:         repoPath,
	}

	if _, err := CreateWorktree(cfg, "feature/foo", true, "main"); err != nil {
		t.Fatalf("failed to create first worktree: %v", err)
	}

	exists, _ := WorktreeExists(cfg, "feature-foo")
	if exists {
		t.Fatalf("expected WorktreeExists(feature-foo) to be false due to branch mismatch")
	}

	collision, collisionPath, err := FindCollidingWorktree(cfg, "feature-foo")
	if err != nil {
		t.Fatalf("FindCollidingWorktree returned error: %v", err)
	}
	if collision == nil {
		t.Fatalf("expected a collision for branch feature-foo")
	}
	if collision.Branch != "feature/foo" {
		t.Fatalf("expected colliding branch feature/foo, got %s", collision.Branch)
	}
	if collisionPath != cfg.GetWorktreePath("feature-foo") {
		t.Fatalf("unexpected collision path: %s", collisionPath)
	}
}

func TestCreateWorktreeRejectsSanitizedPathCollision(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	setupRepo(t, repoPath)
	withWorkingDirectory(t, repoPath)

	cfg := &Config{
		WorktreeBasePath: filepath.Join(tempDir, "worktrees"),
		RepoName:         "repo",
		RepoRoot:         repoPath,
	}

	if _, err := CreateWorktree(cfg, "feature/foo", true, "main"); err != nil {
		t.Fatalf("failed to create first worktree: %v", err)
	}

	_, err := CreateWorktree(cfg, "feature-foo", true, "main")
	if err == nil {
		t.Fatalf("expected collision error when creating feature-foo")
	}
	if !strings.Contains(err.Error(), "worktree path collision") {
		t.Fatalf("expected collision error, got: %v", err)
	}
}

func TestCreateWorktreeSetsPushAutoSetupRemote(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	setupRepo(t, repoPath)
	withWorkingDirectory(t, repoPath)

	cfg := &Config{
		WorktreeBasePath: filepath.Join(tempDir, "worktrees"),
		RepoName:         "repo",
		RepoRoot:         repoPath,
	}

	worktreePath, err := CreateWorktree(cfg, "feature/set-auto-remote", true, "main")
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	value := runGit(t, repoPath, "-C", worktreePath, "config", "--local", "--get", "push.autoSetupRemote")
	if value != "true" {
		t.Fatalf("expected push.autoSetupRemote=true, got %q", value)
	}
}

func TestBranchFromWorktreePath(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	setupRepo(t, repoPath)
	withWorkingDirectory(t, repoPath)

	cfg := &Config{
		WorktreeBasePath: filepath.Join(tempDir, "worktrees"),
		RepoName:         "repo",
		RepoRoot:         repoPath,
	}

	worktreePath, err := CreateWorktree(cfg, "feature/cleanup-test", true, "main")
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	branch := branchFromWorktreePath(worktreePath)
	if branch != "feature/cleanup-test" {
		t.Errorf("expected 'feature/cleanup-test', got %q", branch)
	}
}

func TestBranchFromWorktreePathNonExistent(t *testing.T) {
	branch := branchFromWorktreePath("/nonexistent/path")
	if branch != "" {
		t.Errorf("expected empty string for non-existent path, got %q", branch)
	}
}
