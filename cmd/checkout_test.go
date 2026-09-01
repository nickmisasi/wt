package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nickmisasi/wt/internal"
)

func realPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", path, err)
	}
	return resolved
}

func setupRepo(t *testing.T, path string, extraBranches ...string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available on PATH")
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", path, err)
	}

	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", path}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-b", "main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(path, "README.md"), []byte("test"), 0o644); err != nil {
		t.Fatalf("WriteFile(README.md): %v", err)
	}
	run("add", ".")
	run("commit", "-m", "initial commit")
	for _, b := range extraBranches {
		if b != "main" {
			run("branch", b)
		}
	}
}

func gitRev(t *testing.T, repoPath, ref string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", ref)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-parse %s: %v\n%s", ref, err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestEnsureBranch_BaseBranchRespected(t *testing.T) {
	tmpDir := realPath(t, t.TempDir())
	repoPath := filepath.Join(tmpDir, "repo")
	worktreeBase := filepath.Join(tmpDir, "worktrees")
	setupRepo(t, repoPath, "release-4.19")

	// Diverge release-4.19 from main
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("checkout", "release-4.19")
	if err := os.WriteFile(filepath.Join(repoPath, "release.txt"), []byte("release"), 0o644); err != nil {
		t.Fatalf("WriteFile(release.txt): %v", err)
	}
	run("add", ".")
	run("commit", "-m", "release commit")
	run("checkout", "main")

	releaseHead := gitRev(t, repoPath, "release-4.19")

	cfg := &internal.Config{
		WorktreeBasePath: worktreeBase,
		RepoName:         "testrepo",
		RepoRoot:         repoPath,
	}
	repo := &internal.GitRepo{Root: repoPath, Name: "testrepo"}

	path, err := ensureBranchAndCreateWorktree(cfg, repo, "my-feature", "release-4.19")
	if err != nil {
		t.Fatalf("ensureBranchAndCreateWorktree: %v", err)
	}

	wtHead := gitRev(t, path, "HEAD")
	if wtHead != releaseHead {
		t.Errorf("expected worktree at release-4.19 (%s), got %s", releaseHead, wtHead)
	}
}

func TestEnsureBranch_BaseBranchOverridesExistingLocal(t *testing.T) {
	tmpDir := realPath(t, t.TempDir())
	repoPath := filepath.Join(tmpDir, "repo")
	worktreeBase := filepath.Join(tmpDir, "worktrees")
	setupRepo(t, repoPath, "release-4.19")

	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	// Diverge release-4.19
	run("checkout", "release-4.19")
	if err := os.WriteFile(filepath.Join(repoPath, "release.txt"), []byte("release"), 0o644); err != nil {
		t.Fatalf("WriteFile(release.txt): %v", err)
	}
	run("add", ".")
	run("commit", "-m", "release commit")
	run("checkout", "main")

	// Pre-create "my-feature" from main (simulating a previous checkout)
	run("branch", "my-feature")
	mainHead := gitRev(t, repoPath, "main")
	featureHead := gitRev(t, repoPath, "my-feature")
	if featureHead != mainHead {
		t.Fatal("setup error: my-feature should start at main")
	}

	releaseHead := gitRev(t, repoPath, "release-4.19")

	cfg := &internal.Config{
		WorktreeBasePath: worktreeBase,
		RepoName:         "testrepo",
		RepoRoot:         repoPath,
	}
	repo := &internal.GitRepo{Root: repoPath, Name: "testrepo"}

	// With -b release-4.19, it should reset my-feature to release-4.19
	path, err := ensureBranchAndCreateWorktree(cfg, repo, "my-feature", "release-4.19")
	if err != nil {
		t.Fatalf("ensureBranchAndCreateWorktree: %v", err)
	}

	wtHead := gitRev(t, path, "HEAD")
	if wtHead == mainHead {
		t.Error("branch was NOT reset — still at main HEAD (the old bug)")
	}
	if wtHead != releaseHead {
		t.Errorf("expected worktree at release-4.19 (%s), got %s", releaseHead, wtHead)
	}
}

func TestEnsureBranch_NoBaseBranch_UsesDefault(t *testing.T) {
	tmpDir := realPath(t, t.TempDir())
	repoPath := filepath.Join(tmpDir, "repo")
	worktreeBase := filepath.Join(tmpDir, "worktrees")
	setupRepo(t, repoPath)

	mainHead := gitRev(t, repoPath, "main")

	cfg := &internal.Config{
		WorktreeBasePath: worktreeBase,
		RepoName:         "testrepo",
		RepoRoot:         repoPath,
	}
	repo := &internal.GitRepo{Root: repoPath, Name: "testrepo"}

	path, err := ensureBranchAndCreateWorktree(cfg, repo, "new-branch", "")
	if err != nil {
		t.Fatalf("ensureBranchAndCreateWorktree: %v", err)
	}

	wtHead := gitRev(t, path, "HEAD")
	if wtHead != mainHead {
		t.Errorf("without -b, expected branch from main (%s), got %s", mainHead, wtHead)
	}
}
