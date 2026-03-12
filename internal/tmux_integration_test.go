//go:build integration

package internal

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func requireTmux(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available, skipping integration test")
	}
}

func cleanupSession(t *testing.T, name string) {
	t.Helper()
	t.Cleanup(func() {
		_ = exec.Command("tmux", "kill-session", "-t", name).Run()
	})
}

func TestCreateSessionSetsRemainOnExit(t *testing.T) {
	requireTmux(t)

	name := "wt-test-remain-on-exit"
	cleanupSession(t, name)

	dir := t.TempDir()
	err := CreateSession(name, dir, "sleep 3600")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Verify remain-on-exit is set
	cmd := exec.Command("tmux", "show-options", "-t", name, "remain-on-exit")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("show-options failed: %v (%s)", err, string(output))
	}
	if !strings.Contains(string(output), "on") {
		t.Errorf("expected remain-on-exit to be on, got: %s", string(output))
	}
}

func TestCreateSessionSetsPaneDiedHook(t *testing.T) {
	requireTmux(t)

	name := "wt-test-pane-died-hook"
	cleanupSession(t, name)

	dir := t.TempDir()
	err := CreateSession(name, dir, "sleep 3600")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Verify pane-died hook is set
	cmd := exec.Command("tmux", "show-hooks", "-t", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("show-hooks failed: %v (%s)", err, string(output))
	}
	if !strings.Contains(string(output), "pane-died") {
		t.Errorf("expected pane-died hook to be set, got: %s", string(output))
	}
	if !strings.Contains(string(output), "respawn-pane") {
		t.Errorf("expected pane-died hook to contain respawn-pane, got: %s", string(output))
	}
}

func TestCrashRecoveryEndToEnd(t *testing.T) {
	requireTmux(t)

	name := "wt-test-crash-recovery"
	cleanupSession(t, name)

	dir := t.TempDir()

	// Use a command that writes a marker file and then exits.
	// The respawn (after sleep 2) should recreate the marker file.
	markerFile := dir + "/alive"
	command := "touch " + markerFile + " && sleep 0.1"

	err := CreateSession(name, dir, command)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Wait for the command to run and exit (it only sleeps 0.1s)
	time.Sleep(1 * time.Second)

	// The pane should be dead now (command exited) but session should still exist
	if !HasSession(name) {
		t.Fatal("expected session to still exist after command exit (remain-on-exit)")
	}

	// Remove the marker file
	os.Remove(markerFile)

	// Wait for pane-died hook to fire and respawn (sleep 2 + command time)
	time.Sleep(4 * time.Second)

	// The marker file should be recreated by the respawned command
	if _, err := os.Stat(markerFile); os.IsNotExist(err) {
		t.Error("expected marker file to be recreated by respawned pane, but it does not exist")
	}
}
