package cmd

import (
	"strings"
	"testing"
)

func TestHelpTextContainsSections(t *testing.T) {
	sections := []string{
		"USAGE:",
		"COMMANDS:",
		"OPTIONS:",
		"WORKTREE STORAGE:",
		"CLAUDEMUX (PERSISTENT CLAUDE SESSIONS):",
		"MATTERMOST DUAL-REPOSITORY SUPPORT:",
		"EXAMPLES:",
		"CONFIGURATION:",
		"INSTALLATION:",
	}

	for _, section := range sections {
		if !strings.Contains(helpText, section) {
			t.Errorf("helpText missing section: %s", section)
		}
	}
}

func TestHelpTextContainsClaudemuxCommands(t *testing.T) {
	commands := []string{
		"sessions",
		"--claudemux",
		"--no-claudemux",
		"claudemux.enabled",
		"claudemux.command",
		"claudemux.max_sessions",
		"wt sessions health",
		"wt sessions start",
		"wt sessions stop",
		"wt sessions restart",
	}

	for _, cmd := range commands {
		if !strings.Contains(helpText, cmd) {
			t.Errorf("helpText missing claudemux reference: %s", cmd)
		}
	}
}

func TestHelpTextContainsRequirements(t *testing.T) {
	requirements := []string{
		"tmux must be installed",
		"claude CLI must be installed",
	}

	for _, req := range requirements {
		if !strings.Contains(helpText, req) {
			t.Errorf("helpText missing requirement: %s", req)
		}
	}
}
