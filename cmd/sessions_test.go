package cmd

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/nickmisasi/wt/internal"
)

// tmuxAvailable returns true if tmux is installed and runnable.
func tmuxAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// --- timeAgo unit tests ---

func TestTimeAgoJustNow(t *testing.T) {
	result := timeAgo(time.Now())
	if result != "just now" {
		t.Fatalf("expected 'just now', got %q", result)
	}
}

func TestTimeAgoMinutes(t *testing.T) {
	result := timeAgo(time.Now().Add(-5 * time.Minute))
	if result != "5m ago" {
		t.Fatalf("expected '5m ago', got %q", result)
	}
}

func TestTimeAgoHours(t *testing.T) {
	result := timeAgo(time.Now().Add(-3 * time.Hour))
	if result != "3h ago" {
		t.Fatalf("expected '3h ago', got %q", result)
	}
}

func TestTimeAgoDays(t *testing.T) {
	result := timeAgo(time.Now().Add(-48 * time.Hour))
	if result != "2d ago" {
		t.Fatalf("expected '2d ago', got %q", result)
	}
}

func TestTimeAgoEdgeCases(t *testing.T) {
	// 59 seconds -> just now
	result := timeAgo(time.Now().Add(-59 * time.Second))
	if result != "just now" {
		t.Fatalf("expected 'just now' for 59s, got %q", result)
	}

	// 60 seconds -> 1m ago
	result = timeAgo(time.Now().Add(-60 * time.Second))
	if result != "1m ago" {
		t.Fatalf("expected '1m ago' for 60s, got %q", result)
	}

	// 59 minutes -> 59m ago
	result = timeAgo(time.Now().Add(-59 * time.Minute))
	if result != "59m ago" {
		t.Fatalf("expected '59m ago' for 59min, got %q", result)
	}

	// 23 hours -> 23h ago
	result = timeAgo(time.Now().Add(-23 * time.Hour))
	if result != "23h ago" {
		t.Fatalf("expected '23h ago' for 23h, got %q", result)
	}

	// 24 hours -> 1d ago
	result = timeAgo(time.Now().Add(-24 * time.Hour))
	if result != "1d ago" {
		t.Fatalf("expected '1d ago' for 24h, got %q", result)
	}
}

// --- RunSessions dispatch tests ---

func TestRunSessionsUnknownSubcommand(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &internal.Config{
		WorktreeBasePath: tempDir,
		RepoName:         "repo",
		RepoRoot:         tempDir,
	}
	err := RunSessions(cfg, []string{"bogus"})
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "unknown sessions subcommand: bogus") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// --- sessions list tests ---

func TestSessionsListNoSessions(t *testing.T) {
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}

	stdout, err := captureStdout(t, func() error {
		return runSessionsList()
	})
	if err != nil {
		t.Fatalf("runSessionsList returned error: %v", err)
	}

	// If no wt- sessions exist, we expect the "No claudemux sessions" message.
	// If sessions do exist, the output will contain the session table instead.
	if stdout != "" && !strings.Contains(stdout, "SESSION") && !strings.Contains(stdout, "No claudemux sessions") {
		t.Fatalf("expected output to contain 'No claudemux sessions' or session table header, got:\n%s", stdout)
	}
}

func TestSessionsListWithSessions(t *testing.T) {
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}

	// Create a test session with a unique name
	sessionName := "wt-test-sessions-list-" + t.Name()
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "sleep", "300")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create test tmux session: %v", err)
	}
	t.Cleanup(func() {
		exec.Command("tmux", "kill-session", "-t", sessionName).Run()
	})

	stdout, err := captureStdout(t, func() error {
		return runSessionsList()
	})
	if err != nil {
		t.Fatalf("runSessionsList returned error: %v", err)
	}

	if !strings.Contains(stdout, sessionName) {
		t.Fatalf("expected output to contain session name %q, got:\n%s", sessionName, stdout)
	}
	if !strings.Contains(stdout, "running") {
		t.Fatalf("expected output to contain 'running', got:\n%s", stdout)
	}
}

// --- sessions stop tests ---

func TestSessionsStopNoSessions(t *testing.T) {
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}

	// This test is best-effort: if no wt- sessions exist, it should print
	// "No claudemux sessions to stop." and succeed.
	// We can't guarantee a clean slate, so just verify no crash.
	_, err := captureStdout(t, func() error {
		return runSessionsStop()
	})
	if err != nil {
		t.Fatalf("runSessionsStop returned error: %v", err)
	}
}

// WARNING: This test kills ALL wt- prefixed tmux sessions. Do not run if you
// have active claudemux sessions. The runSessionsStop function stops every
// session matching the wt- prefix, not just test-created ones.
func TestSessionsStopKillsSessions(t *testing.T) {
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}

	// Create two test sessions
	session1 := "wt-test-stop-1-" + t.Name()
	session2 := "wt-test-stop-2-" + t.Name()
	for _, name := range []string{session1, session2} {
		cmd := exec.Command("tmux", "new-session", "-d", "-s", name, "sleep", "300")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create test session %s: %v", name, err)
		}
	}
	t.Cleanup(func() {
		exec.Command("tmux", "kill-session", "-t", session1).Run()
		exec.Command("tmux", "kill-session", "-t", session2).Run()
	})

	// Verify they exist
	if !internal.HasSession(session1) || !internal.HasSession(session2) {
		t.Fatal("test sessions were not created")
	}

	stdout, err := captureStdout(t, func() error {
		return runSessionsStop()
	})
	if err != nil {
		t.Fatalf("runSessionsStop returned error: %v", err)
	}

	if !strings.Contains(stdout, "Stopping") {
		t.Fatalf("expected 'Stopping' in output, got:\n%s", stdout)
	}

	// Verify sessions were killed
	if internal.HasSession(session1) {
		t.Fatalf("session %s should have been killed", session1)
	}
	if internal.HasSession(session2) {
		t.Fatalf("session %s should have been killed", session2)
	}
}

// --- sessions start tests ---

func TestSessionsStartNoWorktrees(t *testing.T) {
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}

	tempDir := t.TempDir()
	repoPath := tempDir + "/repo"
	setupGitRepo(t, repoPath)
	withCwd(t, repoPath)

	cfg := &internal.Config{
		WorktreeBasePath: tempDir + "/worktrees",
		RepoName:         "repo",
		RepoRoot:         repoPath,
	}

	stdout, err := captureStdout(t, func() error {
		return runSessionsStart(cfg)
	})
	if err != nil {
		t.Fatalf("runSessionsStart returned error: %v", err)
	}
	if !strings.Contains(stdout, "No worktrees found") {
		t.Fatalf("expected 'No worktrees found', got:\n%s", stdout)
	}
}

func TestSessionsStartCreatesSessionsForWorktrees(t *testing.T) {
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}

	tempDir := t.TempDir()
	repoPath := tempDir + "/repo"
	setupGitRepo(t, repoPath)
	withCwd(t, repoPath)

	cfg := &internal.Config{
		WorktreeBasePath: tempDir + "/worktrees",
		RepoName:         "repo",
		RepoRoot:         repoPath,
	}

	// Create a worktree
	branch := "feature/start-test"
	if _, err := internal.CreateWorktree(cfg, branch, true, "main"); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	expectedSession := internal.SanitizeBranchForTmux(branch)
	t.Cleanup(func() {
		exec.Command("tmux", "kill-session", "-t", expectedSession).Run()
	})

	stdout, err := captureStdout(t, func() error {
		return runSessionsStart(cfg)
	})
	if err != nil {
		t.Fatalf("runSessionsStart returned error: %v", err)
	}

	if !strings.Contains(stdout, "Starting") {
		t.Fatalf("expected 'Starting' in output, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Started 1 session") {
		t.Fatalf("expected 'Started 1 session' in output, got:\n%s", stdout)
	}
}

func TestSessionsStartSkipsExistingSessions(t *testing.T) {
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}

	tempDir := t.TempDir()
	repoPath := tempDir + "/repo"
	setupGitRepo(t, repoPath)
	withCwd(t, repoPath)

	cfg := &internal.Config{
		WorktreeBasePath: tempDir + "/worktrees",
		RepoName:         "repo",
		RepoRoot:         repoPath,
	}

	// Create a worktree
	branch := "feature/skip-existing"
	if _, err := internal.CreateWorktree(cfg, branch, true, "main"); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Pre-create the session so start should skip it
	sessionName := internal.SanitizeBranchForTmux(branch)
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "sleep", "300")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to pre-create session: %v", err)
	}
	t.Cleanup(func() {
		exec.Command("tmux", "kill-session", "-t", sessionName).Run()
	})

	stdout, err := captureStdout(t, func() error {
		return runSessionsStart(cfg)
	})
	if err != nil {
		t.Fatalf("runSessionsStart returned error: %v", err)
	}

	if !strings.Contains(stdout, "Started 0 session") {
		t.Fatalf("expected 'Started 0 session' (skip existing), got:\n%s", stdout)
	}
}

// --- sessions restart tests ---

func TestSessionsRestartCallsStopThenStart(t *testing.T) {
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}

	tempDir := t.TempDir()
	repoPath := tempDir + "/repo"
	setupGitRepo(t, repoPath)
	withCwd(t, repoPath)

	cfg := &internal.Config{
		WorktreeBasePath: tempDir + "/worktrees",
		RepoName:         "repo",
		RepoRoot:         repoPath,
	}

	// Create a worktree
	branch := "feature/restart-test"
	if _, err := internal.CreateWorktree(cfg, branch, true, "main"); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	sessionName := internal.SanitizeBranchForTmux(branch)
	t.Cleanup(func() {
		exec.Command("tmux", "kill-session", "-t", sessionName).Run()
	})

	// First create a session via start
	_, err := captureStdout(t, func() error {
		return runSessionsStart(cfg)
	})
	if err != nil {
		t.Fatalf("initial start failed: %v", err)
	}
	if !internal.HasSession(sessionName) {
		t.Fatal("session should exist after start")
	}

	// Now restart
	stdout, err := captureStdout(t, func() error {
		return runSessionsRestart(cfg)
	})
	if err != nil {
		t.Fatalf("runSessionsRestart returned error: %v", err)
	}

	// Output should contain both stop and start messages
	if !strings.Contains(stdout, "Stopped") {
		t.Fatalf("expected 'Stopped' in restart output, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Started") {
		t.Fatalf("expected 'Started' in restart output, got:\n%s", stdout)
	}
}

// --- sessions health tests ---

func TestSessionsHealthNoSessions(t *testing.T) {
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}

	tempDir := t.TempDir()
	cfg := &internal.Config{
		WorktreeBasePath: tempDir,
		RepoName:         "repo",
		RepoRoot:         tempDir,
	}

	stdout, err := captureStdout(t, func() error {
		return runSessionsHealth(cfg)
	})
	if err != nil {
		t.Fatalf("runSessionsHealth returned error: %v", err)
	}

	// If no wt- sessions exist, we expect the "no sessions" message.
	// If sessions do exist, the output will contain the health summary instead.
	if stdout != "" && !strings.Contains(stdout, "healthy") && !strings.Contains(stdout, "No claudemux sessions to check") {
		t.Fatalf("expected output to contain 'No claudemux sessions to check' or health summary, got:\n%s", stdout)
	}
}
