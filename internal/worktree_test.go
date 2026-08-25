package internal

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func chdirTo(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
}

func evalSymlinks(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("failed to resolve symlinks for %s: %v", path, err)
	}
	return resolved
}

func TestListWorktreePaths(t *testing.T) {
	tmpDir := evalSymlinks(t, t.TempDir())
	repoPath := filepath.Join(tmpDir, "repo")
	worktreeBase := filepath.Join(tmpDir, "worktrees")

	setupTestGitRepo(t, repoPath, "branch-a", "branch-b")

	cfg := &Config{
		WorktreeBasePath: worktreeBase,
		RepoName:         "repo",
		RepoRoot:         repoPath,
	}

	os.MkdirAll(worktreeBase, 0755)
	wtPathA := filepath.Join(worktreeBase, "repo-branch-a")
	wtPathB := filepath.Join(worktreeBase, "repo-branch-b")

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	run("worktree", "add", wtPathA, "branch-a")
	run("worktree", "add", wtPathB, "branch-b")

	chdirTo(t, repoPath)

	tests := []struct {
		name          string
		config        *Config
		wantCount     int
		wantBranches  []string
		wantErrSubstr string
	}{
		{
			name:         "When worktrees exist it should return all managed paths",
			config:       cfg,
			wantCount:    2,
			wantBranches: []string{"branch-a", "branch-b"},
		},
		{
			name: "When WorktreeBasePath does not match it should return empty",
			config: &Config{
				WorktreeBasePath: "/nonexistent/path",
				RepoName:         "repo",
				RepoRoot:         repoPath,
			},
			wantCount:    0,
			wantBranches: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worktrees, err := ListWorktreePaths(tt.config)
			if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(worktrees) != tt.wantCount {
				t.Errorf("got %d worktrees, want %d", len(worktrees), tt.wantCount)
			}
			for _, wantBranch := range tt.wantBranches {
				found := false
				for _, wt := range worktrees {
					if wt.Branch == wantBranch {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected branch %q in results", wantBranch)
				}
			}
			for _, wt := range worktrees {
				if wt.IsDirty {
					t.Errorf("ListWorktreePaths should not fill IsDirty, but %q has IsDirty=true", wt.Branch)
				}
				if !wt.LastCommit.IsZero() {
					t.Errorf("ListWorktreePaths should not fill LastCommit, but %q has non-zero LastCommit", wt.Branch)
				}
			}
		})
	}
}

func TestFillWorktreeStatus(t *testing.T) {
	tmpDir := evalSymlinks(t, t.TempDir())
	repoPath := filepath.Join(tmpDir, "repo")
	worktreeBase := filepath.Join(tmpDir, "worktrees")

	setupTestGitRepo(t, repoPath, "clean-branch", "dirty-branch")

	os.MkdirAll(worktreeBase, 0755)
	cleanPath := filepath.Join(worktreeBase, "repo-clean-branch")
	dirtyPath := filepath.Join(worktreeBase, "repo-dirty-branch")

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	run("worktree", "add", cleanPath, "clean-branch")
	run("worktree", "add", dirtyPath, "dirty-branch")

	// Make dirty-branch dirty
	if err := os.WriteFile(filepath.Join(dirtyPath, "uncommitted.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatalf("failed to create dirty file: %v", err)
	}

	tests := []struct {
		name       string
		worktrees  []WorktreeInfo
		maxWorkers int
		checks     func(t *testing.T, worktrees []WorktreeInfo)
	}{
		{
			name: "When worktrees have mixed status it should detect dirty and clean correctly",
			worktrees: []WorktreeInfo{
				{Path: cleanPath, Branch: "clean-branch"},
				{Path: dirtyPath, Branch: "dirty-branch"},
			},
			maxWorkers: 2,
			checks: func(t *testing.T, worktrees []WorktreeInfo) {
				for _, wt := range worktrees {
					switch wt.Branch {
					case "clean-branch":
						if wt.IsDirty {
							t.Error("clean-branch should not be dirty")
						}
					case "dirty-branch":
						if !wt.IsDirty {
							t.Error("dirty-branch should be dirty")
						}
					}
					if wt.LastCommit.IsZero() {
						t.Errorf("%s should have a non-zero LastCommit", wt.Branch)
					}
					if time.Since(wt.LastCommit) > 5*time.Minute {
						t.Errorf("%s LastCommit too old, expected recent commit", wt.Branch)
					}
				}
			},
		},
		{
			name:       "When worktrees slice is empty it should not panic",
			worktrees:  []WorktreeInfo{},
			maxWorkers: 2,
			checks: func(t *testing.T, worktrees []WorktreeInfo) {
				if len(worktrees) != 0 {
					t.Error("expected empty slice")
				}
			},
		},
		{
			name: "When maxWorkers is 1 it should still complete all checks",
			worktrees: []WorktreeInfo{
				{Path: cleanPath, Branch: "clean-branch"},
				{Path: dirtyPath, Branch: "dirty-branch"},
			},
			maxWorkers: 1,
			checks: func(t *testing.T, worktrees []WorktreeInfo) {
				if len(worktrees) != 2 {
					t.Errorf("expected 2 worktrees, got %d", len(worktrees))
				}
				for _, wt := range worktrees {
					if wt.LastCommit.IsZero() {
						t.Errorf("%s should have non-zero LastCommit", wt.Branch)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			FillWorktreeStatus(tt.worktrees, tt.maxWorkers)
			tt.checks(t, tt.worktrees)
		})
	}
}

func TestFixWorktreePermissions(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, dir string)
		check func(t *testing.T, dir string)
	}{
		{
			name: "When files are read-only it should make them writable",
			setup: func(t *testing.T, dir string) {
				f := filepath.Join(dir, "readonly.txt")
				os.WriteFile(f, []byte("test"), 0444)
			},
			check: func(t *testing.T, dir string) {
				f := filepath.Join(dir, "readonly.txt")
				info, err := os.Stat(f)
				if err != nil {
					t.Fatalf("stat failed: %v", err)
				}
				if info.Mode()&0200 == 0 {
					t.Error("file should be writable after fix")
				}
			},
		},
		{
			name: "When directories are read-only it should make them writable",
			setup: func(t *testing.T, dir string) {
				subdir := filepath.Join(dir, "readonly-dir")
				os.MkdirAll(subdir, 0555)
				os.WriteFile(filepath.Join(subdir, "file.txt"), []byte("test"), 0444)
			},
			check: func(t *testing.T, dir string) {
				subdir := filepath.Join(dir, "readonly-dir")
				info, err := os.Stat(subdir)
				if err != nil {
					t.Fatalf("stat failed: %v", err)
				}
				if info.Mode()&0200 == 0 {
					t.Error("directory should be writable after fix")
				}
			},
		},
		{
			name: "When nested read-only files exist it should fix recursively",
			setup: func(t *testing.T, dir string) {
				deep := filepath.Join(dir, "a", "b", "c")
				os.MkdirAll(deep, 0755)
				os.WriteFile(filepath.Join(deep, "locked.bin"), []byte("binary"), 0444)
			},
			check: func(t *testing.T, dir string) {
				f := filepath.Join(dir, "a", "b", "c", "locked.bin")
				info, err := os.Stat(f)
				if err != nil {
					t.Fatalf("stat failed: %v", err)
				}
				if info.Mode()&0200 == 0 {
					t.Error("nested file should be writable after fix")
				}
			},
		},
		{
			name: "When path does not exist it should not panic",
			setup: func(t *testing.T, dir string) {
				// intentionally empty
			},
			check: func(t *testing.T, dir string) {
				FixWorktreePermissions(filepath.Join(dir, "nonexistent"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(t, dir)
			FixWorktreePermissions(dir)
			tt.check(t, dir)
		})
	}
}

func TestListWorktrees(t *testing.T) {
	tmpDir := evalSymlinks(t, t.TempDir())
	repoPath := filepath.Join(tmpDir, "repo")
	worktreeBase := filepath.Join(tmpDir, "worktrees")

	setupTestGitRepo(t, repoPath, "feat-1", "feat-2")

	os.MkdirAll(worktreeBase, 0755)
	path1 := filepath.Join(worktreeBase, "repo-feat-1")
	path2 := filepath.Join(worktreeBase, "repo-feat-2")

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	run("worktree", "add", path1, "feat-1")
	run("worktree", "add", path2, "feat-2")

	os.WriteFile(filepath.Join(path2, "dirty.txt"), []byte("x"), 0644)

	chdirTo(t, repoPath)

	cfg := &Config{
		WorktreeBasePath: worktreeBase,
		RepoName:         "repo",
		RepoRoot:         repoPath,
	}

	tests := []struct {
		name   string
		checks func(t *testing.T, worktrees []WorktreeInfo)
	}{
		{
			name: "When called it should return worktrees with status filled",
			checks: func(t *testing.T, worktrees []WorktreeInfo) {
				if len(worktrees) != 2 {
					t.Fatalf("expected 2 worktrees, got %d", len(worktrees))
				}
				for _, wt := range worktrees {
					if wt.LastCommit.IsZero() {
						t.Errorf("%s should have non-zero LastCommit", wt.Branch)
					}
				}
			},
		},
		{
			name: "When one worktree is dirty it should detect it correctly",
			checks: func(t *testing.T, worktrees []WorktreeInfo) {
				for _, wt := range worktrees {
					switch wt.Branch {
					case "feat-1":
						if wt.IsDirty {
							t.Error("feat-1 should be clean")
						}
					case "feat-2":
						if !wt.IsDirty {
							t.Error("feat-2 should be dirty")
						}
					}
				}
			},
		},
	}

	worktrees, err := ListWorktrees(cfg)
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.checks(t, worktrees)
		})
	}
}
