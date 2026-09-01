package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupBenchRepo(b *testing.B, numWorktrees int) (*Config, string) {
	b.Helper()
	tmpDir := b.TempDir()
	resolved, _ := filepath.EvalSymlinks(tmpDir)
	tmpDir = resolved

	repoPath := filepath.Join(tmpDir, "repo")
	worktreeBase := filepath.Join(tmpDir, "worktrees")
	os.MkdirAll(repoPath, 0755)
	os.MkdirAll(worktreeBase, 0755)

	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			b.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init", "-b", "main")
	run("config", "user.email", "bench@test.com")
	run("config", "user.name", "Bench")
	os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("bench"), 0644)
	run("add", ".")
	run("commit", "-m", "initial")

	for i := 0; i < numWorktrees; i++ {
		branch := fmt.Sprintf("bench-branch-%d", i)
		wtPath := filepath.Join(worktreeBase, fmt.Sprintf("testrepo-%s", branch))
		run("worktree", "add", "-b", branch, wtPath)
		// Add a dirty file to half of them to make git status slower
		if i%2 == 0 {
			os.WriteFile(filepath.Join(wtPath, "dirty.txt"), []byte("uncommitted"), 0644)
		}
	}

	cfg := &Config{
		WorktreeBasePath: worktreeBase,
		RepoName:         "testrepo",
		RepoRoot:         repoPath,
	}
	return cfg, repoPath
}

func BenchmarkListWorktrees_Heavy(b *testing.B) {
	for _, n := range []int{2, 5, 10, 20} {
		b.Run(fmt.Sprintf("worktrees=%d", n), func(b *testing.B) {
			cfg, _ := setupBenchRepo(b, n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ListWorktrees(cfg)
			}
		})
	}
}

func BenchmarkListWorktreePaths_Light(b *testing.B) {
	for _, n := range []int{2, 5, 10, 20} {
		b.Run(fmt.Sprintf("worktrees=%d", n), func(b *testing.B) {
			cfg, _ := setupBenchRepo(b, n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				listWorktreePaths(cfg)
			}
		})
	}
}

func BenchmarkWorktreeExists_Old(b *testing.B) {
	for _, n := range []int{2, 5, 10, 20} {
		b.Run(fmt.Sprintf("worktrees=%d", n), func(b *testing.B) {
			cfg, _ := setupBenchRepo(b, n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Simulate old WorktreeExists that called ListWorktrees
				worktreePath := cfg.GetWorktreePath("bench-branch-0")
				if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
					continue
				}
				worktrees, err := ListWorktrees(cfg)
				if err != nil {
					continue
				}
				for _, wt := range worktrees {
					if wt.Path == worktreePath {
						break
					}
				}
			}
		})
	}
}

func BenchmarkWorktreeExists_New(b *testing.B) {
	for _, n := range []int{2, 5, 10, 20} {
		b.Run(fmt.Sprintf("worktrees=%d", n), func(b *testing.B) {
			cfg, _ := setupBenchRepo(b, n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				WorktreeExists(cfg, "bench-branch-0")
			}
		})
	}
}
