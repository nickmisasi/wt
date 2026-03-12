package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nickmisasi/wt/internal"
)

// RunSessions dispatches to the appropriate sessions subcommand.
// Called from main.go with args being everything after "sessions".
func RunSessions(config *internal.Config, args []string) error {
	if !internal.IsTmuxAvailable() {
		return fmt.Errorf("tmux is not installed or not on PATH. Install it with: brew install tmux")
	}

	if len(args) == 0 {
		return runSessionsList()
	}

	switch args[0] {
	case "list", "ls":
		return runSessionsList()
	case "health":
		return runSessionsHealth(config)
	case "start":
		return runSessionsStart(config)
	case "stop":
		return runSessionsStop()
	case "restart":
		return runSessionsRestart(config)
	default:
		return fmt.Errorf("unknown sessions subcommand: %s\nUsage: wt sessions [list|health|start|stop|restart]", args[0])
	}
}

// runSessionsList lists all wt-managed tmux sessions with their status.
func runSessionsList() error {
	sessions, err := internal.ListSessions(internal.SessionPrefix)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		fmt.Println("No claudemux sessions running.")
		return nil
	}

	fmt.Printf("%-30s %-10s %s\n", "SESSION", "STATUS", "CREATED")
	for _, s := range sessions {
		status := "running"
		if s.Dead {
			status = "dead"
		}
		age := timeAgo(s.Created)
		fmt.Printf("%-30s %-10s %s\n", s.Name, status, age)
	}
	return nil
}

// runSessionsHealth checks all sessions and respawns dead ones.
func runSessionsHealth(config *internal.Config) error {
	sessions, err := internal.ListSessions(internal.SessionPrefix)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		fmt.Println("No claudemux sessions to check.")
		return nil
	}

	cfg, err := internal.LoadUserConfig()
	if err != nil {
		return err
	}
	command := cfg.Claudemux.Command
	if command == "" {
		command = "claude --continue --dangerously-skip-permissions"
	}

	healthy, respawned, failed := 0, 0, 0
	for _, s := range sessions {
		if internal.IsSessionHealthy(s.Name) {
			healthy++
			continue
		}
		fmt.Printf("Respawning dead session: %s\n", s.Name)
		// Derive worktree path: strip the session prefix to get the sanitized branch,
		// then look up the worktree path via config.
		branch := strings.TrimPrefix(s.Name, internal.SessionPrefix)
		worktreePath := config.GetWorktreePath(branch)
		if err := internal.RespawnPane(s.Name, worktreePath, command); err != nil {
			fmt.Fprintf(os.Stderr, "  Failed to respawn %s: %v\n", s.Name, err)
			failed++
		} else {
			fmt.Printf("  Respawned %s\n", s.Name)
			respawned++
		}
	}

	if failed > 0 {
		fmt.Printf("\n%d healthy, %d respawned, %d failed\n", healthy, respawned, failed)
	} else {
		fmt.Printf("\n%d healthy, %d respawned\n", healthy, respawned)
	}
	return nil
}

// runSessionsStop kills all wt-managed tmux sessions.
func runSessionsStop() error {
	sessions, err := internal.ListSessions(internal.SessionPrefix)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		fmt.Println("No claudemux sessions to stop.")
		return nil
	}

	stopped := 0
	for _, s := range sessions {
		fmt.Printf("Stopping %s...\n", s.Name)
		if err := internal.KillSession(s.Name); err != nil {
			fmt.Fprintf(os.Stderr, "  Failed: %v\n", err)
		} else {
			stopped++
		}
	}
	fmt.Printf("Stopped %d session(s).\n", stopped)
	return nil
}

// runSessionsStart creates sessions for all worktrees that don't already have one.
// Respects the max_sessions cap from user config.
func runSessionsStart(config *internal.Config) error {
	worktrees, err := internal.ListWorktrees(config)
	if err != nil {
		return err
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees found.")
		return nil
	}

	cfg, err := internal.LoadUserConfig()
	if err != nil {
		return err
	}

	if !isClaudeAvailable() {
		fmt.Fprintf(os.Stderr, "Warning: 'claude' binary not found on PATH. Sessions may fail to start.\n")
	}

	maxSessions := cfg.Claudemux.MaxSessions
	if maxSessions <= 0 {
		maxSessions = 10
	}

	command := cfg.Claudemux.Command
	if command == "" {
		command = "claude --continue --dangerously-skip-permissions"
	}

	created := 0
	for _, wt := range worktrees {
		sessionName := internal.SanitizeBranchForTmux(wt.Branch)
		if internal.HasSession(sessionName) {
			continue
		}

		// Enforce cap: check current session count before each creation
		sessions, _ := internal.ListSessions(internal.SessionPrefix)
		if len(sessions) >= maxSessions {
			fmt.Printf("At session cap (%d). Stopping.\n", maxSessions)
			break
		}

		fmt.Printf("Starting %s...\n", sessionName)
		if err := internal.CreateSession(sessionName, wt.Path, command); err != nil {
			fmt.Fprintf(os.Stderr, "  Failed: %v\n", err)
		} else {
			created++
		}
	}
	fmt.Printf("Started %d session(s).\n", created)
	return nil
}

// runSessionsRestart stops all sessions, then starts them fresh.
// Because claude --continue is used, conversation history is preserved.
func runSessionsRestart(config *internal.Config) error {
	if err := runSessionsStop(); err != nil {
		return err
	}
	fmt.Println()
	return runSessionsStart(config)
}

// timeAgo returns a human-readable relative time string.
func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
