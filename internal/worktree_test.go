package internal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func realPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", path, err)
	}
	return resolved
}

func testConfig(t *testing.T, repoPath, basePath string) *Config {
	t.Helper()
	return &Config{
		WorktreeBasePath: basePath,
		RepoName:         "testrepo",
		RepoRoot:         repoPath,
	}
}

func TestListWorktreePaths(t *testing.T) {
	tmpDir := realPath(t, t.TempDir())
	repoPath := filepath.Join(tmpDir, "repo")
	worktreeBase := filepath.Join(tmpDir, "worktrees")
	setupTestGitRepo(t, repoPath, "feature-a", "feature-b")

	cfg := testConfig(t, repoPath, worktreeBase)
	if err := os.MkdirAll(worktreeBase, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", worktreeBase, err)
	}

	// Create two worktrees inside the managed base path
	wtA := filepath.Join(worktreeBase, "testrepo-feature-a")
	wtB := filepath.Join(worktreeBase, "testrepo-feature-b")
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("worktree", "add", wtA, "feature-a")
	run("worktree", "add", wtB, "feature-b")

	paths, err := listWorktreePaths(cfg)
	if err != nil {
		t.Fatalf("listWorktreePaths: %v", err)
	}

	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}

	found := map[string]bool{}
	for _, p := range paths {
		found[p] = true
	}
	if !found[wtA] {
		t.Errorf("missing path %s", wtA)
	}
	if !found[wtB] {
		t.Errorf("missing path %s", wtB)
	}
}

func TestListWorktreePaths_FiltersUnmanaged(t *testing.T) {
	tmpDir := realPath(t, t.TempDir())
	repoPath := filepath.Join(tmpDir, "repo")
	worktreeBase := filepath.Join(tmpDir, "managed")
	unmanagedDir := filepath.Join(tmpDir, "unmanaged")
	setupTestGitRepo(t, repoPath, "branch-a")

	cfg := testConfig(t, repoPath, worktreeBase)
	if err := os.MkdirAll(worktreeBase, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", worktreeBase, err)
	}
	if err := os.MkdirAll(unmanagedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", unmanagedDir, err)
	}

	// Create one worktree inside managed, one outside
	wtManaged := filepath.Join(worktreeBase, "testrepo-branch-a")
	wtUnmanaged := filepath.Join(unmanagedDir, "other-wt")

	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("worktree", "add", wtManaged, "branch-a")
	run("worktree", "add", "-b", "unmanaged-branch", wtUnmanaged)

	paths, err := listWorktreePaths(cfg)
	if err != nil {
		t.Fatalf("listWorktreePaths: %v", err)
	}

	if len(paths) != 1 {
		t.Fatalf("expected 1 managed path, got %d: %v", len(paths), paths)
	}
	if paths[0] != wtManaged {
		t.Errorf("expected %s, got %s", wtManaged, paths[0])
	}
}

func TestWorktreeExists_Found(t *testing.T) {
	tmpDir := realPath(t, t.TempDir())
	repoPath := filepath.Join(tmpDir, "repo")
	worktreeBase := filepath.Join(tmpDir, "worktrees")
	setupTestGitRepo(t, repoPath, "my-branch")

	cfg := testConfig(t, repoPath, worktreeBase)
	if err := os.MkdirAll(worktreeBase, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", worktreeBase, err)
	}

	wtPath := cfg.GetWorktreePath("my-branch")
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("worktree", "add", wtPath, "my-branch")

	exists, path := WorktreeExists(cfg, "my-branch")
	if !exists {
		t.Fatal("expected worktree to exist")
	}
	if path != wtPath {
		t.Errorf("expected path %s, got %s", wtPath, path)
	}
}

func TestWorktreeExists_NotFound(t *testing.T) {
	tmpDir := realPath(t, t.TempDir())
	repoPath := filepath.Join(tmpDir, "repo")
	worktreeBase := filepath.Join(tmpDir, "worktrees")
	setupTestGitRepo(t, repoPath)

	cfg := testConfig(t, repoPath, worktreeBase)

	exists, _ := WorktreeExists(cfg, "nonexistent")
	if exists {
		t.Fatal("expected worktree to not exist")
	}
}

func TestWorktreeExists_DirExistsButNotWorktree(t *testing.T) {
	tmpDir := realPath(t, t.TempDir())
	repoPath := filepath.Join(tmpDir, "repo")
	worktreeBase := filepath.Join(tmpDir, "worktrees")
	setupTestGitRepo(t, repoPath)

	cfg := testConfig(t, repoPath, worktreeBase)

	// Create the directory manually (not a real worktree)
	fakePath := cfg.GetWorktreePath("fake-branch")
	if err := os.MkdirAll(fakePath, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", fakePath, err)
	}

	exists, _ := WorktreeExists(cfg, "fake-branch")
	if exists {
		t.Fatal("directory exists but is not a worktree — should return false")
	}
}

func TestCreateWorktree_NewBranchFromBase(t *testing.T) {
	tmpDir := realPath(t, t.TempDir())
	repoPath := filepath.Join(tmpDir, "repo")
	worktreeBase := filepath.Join(tmpDir, "worktrees")
	setupTestGitRepo(t, repoPath, "release-4.19")

	// Add a commit on release-4.19 so it differs from main
	run := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("checkout", "release-4.19")
	if err := os.WriteFile(filepath.Join(repoPath, "release.txt"), []byte("release"), 0o644); err != nil {
		t.Fatalf("WriteFile(release.txt): %v", err)
	}
	run("add", ".")
	run("commit", "-m", "release commit")
	releaseHead := run("rev-parse", "HEAD")
	run("checkout", "main")

	cfg := testConfig(t, repoPath, worktreeBase)

	path, err := CreateWorktree(cfg, "new-feature", true, "release-4.19", false)
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	// Verify the worktree HEAD matches release-4.19
	wtHead := run("-C", path, "rev-parse", "HEAD")

	// Get what release-4.19 resolves to from the main repo
	if wtHead != releaseHead {
		t.Errorf("worktree HEAD %s != release-4.19 HEAD %s", wtHead, releaseHead)
	}
}

func TestCreateWorktree_ForceCreateResetsBranch(t *testing.T) {
	tmpDir := realPath(t, t.TempDir())
	repoPath := filepath.Join(tmpDir, "repo")
	worktreeBase := filepath.Join(tmpDir, "worktrees")
	setupTestGitRepo(t, repoPath, "release-4.19")

	run := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}

	// Add a commit on release-4.19
	run("checkout", "release-4.19")
	if err := os.WriteFile(filepath.Join(repoPath, "release.txt"), []byte("release"), 0o644); err != nil {
		t.Fatalf("WriteFile(release.txt): %v", err)
	}
	run("add", ".")
	run("commit", "-m", "release commit")
	releaseHead := run("rev-parse", "HEAD")
	run("checkout", "main")

	mainHead := run("rev-parse", "HEAD")

	// Create branch "my-feature" from main first
	run("branch", "my-feature")

	cfg := testConfig(t, repoPath, worktreeBase)

	// Now force-create worktree with -B, resetting my-feature to release-4.19
	path, err := CreateWorktree(cfg, "my-feature", true, "release-4.19", true)
	if err != nil {
		t.Fatalf("CreateWorktree with forceCreate: %v", err)
	}

	wtHead := run("-C", path, "rev-parse", "HEAD")
	if wtHead != releaseHead {
		t.Errorf("forceCreate should reset branch to release-4.19 (%s), got %s", releaseHead, wtHead)
	}
	if wtHead == mainHead {
		t.Error("forceCreate did not reset the branch — still pointing at main")
	}
}

func TestCreateWorktree_ExistingBranch(t *testing.T) {
	tmpDir := realPath(t, t.TempDir())
	repoPath := filepath.Join(tmpDir, "repo")
	worktreeBase := filepath.Join(tmpDir, "worktrees")
	setupTestGitRepo(t, repoPath, "existing-branch")

	cfg := testConfig(t, repoPath, worktreeBase)

	path, err := CreateWorktree(cfg, "existing-branch", false, "", false)
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("worktree directory not created at %s", path)
	}
}
