package internal

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SessionPrefix is prepended to all wt-managed tmux session names.
const SessionPrefix = "wt-"

// SessionInfo holds metadata about a tmux session.
type SessionInfo struct {
	Name    string
	Created time.Time
	Dead    bool
}

// SanitizeBranchForTmux converts a branch name to a valid tmux session name.
// tmux session names cannot contain "." or ":". Slashes are replaced with "-".
func SanitizeBranchForTmux(branch string) string {
	name := SessionPrefix + branch
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, ".", "")
	name = strings.ReplaceAll(name, ":", "")
	return name
}

// IsTmuxAvailable checks if tmux is installed and on PATH.
func IsTmuxAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// shellescape wraps a string in single quotes for safe embedding in tmux commands.
// Single quotes within the string are escaped as '"'"' (end quote, literal quote, start quote).
func shellescape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// HasSession checks if a tmux session with the given name exists.
func HasSession(name string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", name)
	return cmd.Run() == nil
}

// CreateSession creates a new detached tmux session running the given command.
// It enables remain-on-exit and sets a pane-died hook for automatic crash recovery.
func CreateSession(name, workDir, command string) error {
	if HasSession(name) {
		return fmt.Errorf("tmux session %q already exists", name)
	}
	// tmux new-session treats the trailing argument as a shell command string
	// (executed via $SHELL -c), so passing a multi-word string like
	// "claude --continue --dangerously-skip-permissions" as a single argument
	// is correct — tmux will invoke it through the shell.
	cmd := exec.Command("tmux", "new-session", "-d",
		"-s", name,
		"-c", workDir,
		"-x", "200", "-y", "50",
		command,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create tmux session: %s", string(output))
	}

	// Enable remain-on-exit so panes stay alive after process death.
	// This is required for the pane-died hook to fire.
	setCmd := exec.Command("tmux", "set-option", "-t", name, "remain-on-exit", "on")
	_ = setCmd.Run() // best-effort: session still works without this

	// Set pane-died hook for automatic respawn.
	// The "sleep 2" prevents tight crash loops from spinning.
	shellCmd := "sleep 2 && " + command
	respawnCmd := fmt.Sprintf("respawn-pane -c %s %s", shellescape(workDir), shellescape(shellCmd))
	hookCmd := exec.Command("tmux", "set-hook", "-t", name, "pane-died", respawnCmd)
	_ = hookCmd.Run() // best-effort: manual recovery via `wt sessions health` still works

	return nil
}

// KillSession kills a tmux session by name.
func KillSession(name string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to kill tmux session: %s", string(output))
	}
	return nil
}

// ListSessions returns all tmux sessions whose names start with the given prefix.
func ListSessions(prefix string) ([]SessionInfo, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F",
		"#{session_name}\t#{session_created}\t#{pane_dead}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "no server running") ||
			strings.Contains(string(output), "no sessions") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list tmux sessions: %s", string(output))
	}

	var sessions []SessionInfo
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		name := parts[0]
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		created := time.Time{}
		if ts, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
			created = time.Unix(ts, 0)
		}
		dead := parts[2] == "1"
		sessions = append(sessions, SessionInfo{Name: name, Created: created, Dead: dead})
	}
	return sessions, nil
}

// RespawnPane respawns a dead pane in an existing tmux session.
func RespawnPane(name, workDir, command string) error {
	cmd := exec.Command("tmux", "respawn-pane", "-t", name, "-c", workDir, command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to respawn pane: %s", string(output))
	}
	return nil
}

// IsSessionHealthy returns true if the session exists and its pane is alive.
func IsSessionHealthy(name string) bool {
	cmd := exec.Command("tmux", "display-message", "-t", name, "-p", "#{pane_dead}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "0"
}

// EvictOldestSession kills the oldest wt-prefixed session.
// Returns the name of the evicted session, or empty string if nothing to evict.
func EvictOldestSession() (string, error) {
	sessions, err := ListSessions(SessionPrefix)
	if err != nil {
		return "", err
	}
	if len(sessions) == 0 {
		return "", nil
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Created.Before(sessions[j].Created)
	})
	oldest := sessions[0].Name
	return oldest, KillSession(oldest)
}
