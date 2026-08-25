package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/nickmisasi/wt/internal"
)

func evalSymlinks(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("failed to resolve symlinks for %s: %v", path, err)
	}
	return resolved
}

func setupCleanTestRepo(t *testing.T, tmpDir string, branches ...string) (repoPath, worktreeBase string) {
	t.Helper()
	repoPath = filepath.Join(tmpDir, "repo")
	worktreeBase = filepath.Join(tmpDir, "worktrees")

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available on PATH")
	}

	os.MkdirAll(repoPath, 0755)
	os.MkdirAll(worktreeBase, 0755)

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	run("init", "-b", "main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")

	os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("test"), 0644)
	run("add", ".")
	run("commit", "-m", "initial commit")

	for _, branch := range branches {
		if branch != "main" {
			run("branch", branch)
		}
	}

	return repoPath, worktreeBase
}

func createWorktree(t *testing.T, repoPath, worktreeBase, repoName, branch string) string {
	t.Helper()
	wtPath := filepath.Join(worktreeBase, repoName+"-"+branch)
	cmd := exec.Command("git", "-C", repoPath, "worktree", "add", wtPath, branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create worktree %s: %v\n%s", branch, err, out)
	}
	return wtPath
}

func makeWorktreeDirty(t *testing.T, wtPath string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(wtPath, "uncommitted.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatalf("failed to make worktree dirty: %v", err)
	}
}

func makeWorktreeOld(t *testing.T, repoPath, wtPath, branch string, daysAgo int) {
	t.Helper()
	past := time.Now().AddDate(0, 0, -daysAgo)
	dateStr := past.Format("Mon Jan 2 15:04:05 2006 -0700")
	cmd := exec.Command("git", "-C", wtPath, "commit", "--allow-empty",
		"-m", "old commit",
		"--date", dateStr)
	cmd.Env = append(os.Environ(),
		"GIT_COMMITTER_DATE="+dateStr,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to backdate worktree %s: %v\n%s", branch, err, out)
	}
}

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

func TestRunClean(t *testing.T) {
	tests := []struct {
		name         string
		branches     []string
		setup        func(t *testing.T, repoPath, worktreeBase string, wtPaths map[string]string)
		days         int
		batchSize    int
		includeDirty bool
		wantRemoved  []string
		wantKept     []string
	}{
		{
			name:     "When all worktrees are recent it should remove none",
			branches: []string{"feat-a", "feat-b"},
			setup:    func(t *testing.T, repoPath, worktreeBase string, wtPaths map[string]string) {},
			days:     30, batchSize: 10, includeDirty: false,
			wantRemoved: nil,
			wantKept:    []string{"feat-a", "feat-b"},
		},
		{
			name:     "When worktrees are old and clean it should remove them",
			branches: []string{"old-a", "old-b"},
			setup: func(t *testing.T, repoPath, worktreeBase string, wtPaths map[string]string) {
				makeWorktreeOld(t, repoPath, wtPaths["old-a"], "old-a", 60)
				makeWorktreeOld(t, repoPath, wtPaths["old-b"], "old-b", 45)
			},
			days: 30, batchSize: 10, includeDirty: false,
			wantRemoved: []string{"old-a", "old-b"},
			wantKept:    nil,
		},
		{
			name:     "When worktree is old but dirty it should skip without --dirty",
			branches: []string{"dirty-old"},
			setup: func(t *testing.T, repoPath, worktreeBase string, wtPaths map[string]string) {
				makeWorktreeOld(t, repoPath, wtPaths["dirty-old"], "dirty-old", 60)
				makeWorktreeDirty(t, wtPaths["dirty-old"])
			},
			days: 30, batchSize: 10, includeDirty: false,
			wantRemoved: nil,
			wantKept:    []string{"dirty-old"},
		},
		{
			name:     "When worktree is old and dirty with --dirty it should remove it",
			branches: []string{"dirty-old"},
			setup: func(t *testing.T, repoPath, worktreeBase string, wtPaths map[string]string) {
				makeWorktreeOld(t, repoPath, wtPaths["dirty-old"], "dirty-old", 60)
				makeWorktreeDirty(t, wtPaths["dirty-old"])
			},
			days: 30, batchSize: 10, includeDirty: true,
			wantRemoved: []string{"dirty-old"},
			wantKept:    nil,
		},
		{
			name:     "When days threshold is high it should keep more worktrees",
			branches: []string{"medium-age"},
			setup: func(t *testing.T, repoPath, worktreeBase string, wtPaths map[string]string) {
				makeWorktreeOld(t, repoPath, wtPaths["medium-age"], "medium-age", 45)
			},
			days: 90, batchSize: 10, includeDirty: false,
			wantRemoved: nil,
			wantKept:    []string{"medium-age"},
		},
		{
			name:     "When days threshold is low it should remove more worktrees",
			branches: []string{"medium-age"},
			setup: func(t *testing.T, repoPath, worktreeBase string, wtPaths map[string]string) {
				makeWorktreeOld(t, repoPath, wtPaths["medium-age"], "medium-age", 45)
			},
			days: 7, batchSize: 10, includeDirty: false,
			wantRemoved: []string{"medium-age"},
			wantKept:    nil,
		},
		{
			name:     "When mixed old/recent/dirty it should only remove old clean ones",
			branches: []string{"old-clean", "recent-clean", "old-dirty"},
			setup: func(t *testing.T, repoPath, worktreeBase string, wtPaths map[string]string) {
				makeWorktreeOld(t, repoPath, wtPaths["old-clean"], "old-clean", 60)
				makeWorktreeOld(t, repoPath, wtPaths["old-dirty"], "old-dirty", 60)
				makeWorktreeDirty(t, wtPaths["old-dirty"])
			},
			days: 30, batchSize: 10, includeDirty: false,
			wantRemoved: []string{"old-clean"},
			wantKept:    []string{"recent-clean", "old-dirty"},
		},
		{
			name:     "When batch size is 1 it should still process all worktrees",
			branches: []string{"old-1", "old-2", "old-3"},
			setup: func(t *testing.T, repoPath, worktreeBase string, wtPaths map[string]string) {
				makeWorktreeOld(t, repoPath, wtPaths["old-1"], "old-1", 60)
				makeWorktreeOld(t, repoPath, wtPaths["old-2"], "old-2", 60)
				makeWorktreeOld(t, repoPath, wtPaths["old-3"], "old-3", 60)
			},
			days: 30, batchSize: 1, includeDirty: false,
			wantRemoved: []string{"old-1", "old-2", "old-3"},
			wantKept:    nil,
		},
		{
			name:     "When no worktrees exist it should succeed with no errors",
			branches: []string{},
			setup:    func(t *testing.T, repoPath, worktreeBase string, wtPaths map[string]string) {},
			days:     30, batchSize: 10, includeDirty: false,
			wantRemoved: nil,
			wantKept:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := evalSymlinks(t, t.TempDir())
			repoPath, worktreeBase := setupCleanTestRepo(t, tmpDir, tt.branches...)

			cfg := &internal.Config{
				WorktreeBasePath: worktreeBase,
				RepoName:         "repo",
				RepoRoot:         repoPath,
			}

			wtPaths := make(map[string]string)
			for _, branch := range tt.branches {
				wtPaths[branch] = createWorktree(t, repoPath, worktreeBase, "repo", branch)
			}

			tt.setup(t, repoPath, worktreeBase, wtPaths)

			chdirTo(t, repoPath)

			err := RunClean(cfg, tt.days, tt.batchSize, tt.includeDirty, false)
			if err != nil {
				t.Fatalf("RunClean returned error: %v", err)
			}

			for _, branch := range tt.wantRemoved {
				path := filepath.Join(worktreeBase, "repo-"+branch)
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Errorf("worktree %s should have been removed but still exists", branch)
				}
			}

			for _, branch := range tt.wantKept {
				path := filepath.Join(worktreeBase, "repo-"+branch)
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Errorf("worktree %s should have been kept but was removed", branch)
				}
			}
		})
	}
}

func TestRunCleanWithReadOnlyFiles(t *testing.T) {
	t.Run("When worktree contains read-only files it should still remove it", func(t *testing.T) {
		tmpDir := evalSymlinks(t, t.TempDir())
		repoPath, worktreeBase := setupCleanTestRepo(t, tmpDir, "locked-branch")

		cfg := &internal.Config{
			WorktreeBasePath: worktreeBase,
			RepoName:         "repo",
			RepoRoot:         repoPath,
		}

		wtPath := createWorktree(t, repoPath, worktreeBase, "repo", "locked-branch")
		makeWorktreeOld(t, repoPath, wtPath, "locked-branch", 60)

		chdirTo(t, repoPath)

		// Create read-only files like envtest binaries
		lockedDir := filepath.Join(wtPath, "tools", "bin")
		os.MkdirAll(lockedDir, 0755)
		os.WriteFile(filepath.Join(lockedDir, "etcd"), []byte("binary"), 0444)
		os.Chmod(lockedDir, 0555)

		// includeDirty=true because the read-only files are untracked (makes worktree dirty)
		err := RunClean(cfg, 30, 10, true, false)
		if err != nil {
			t.Fatalf("RunClean returned error: %v", err)
		}

		if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
			t.Error("worktree with read-only files should have been removed")
		}
	})
}
