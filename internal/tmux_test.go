package internal

import "testing"

func TestSanitizeBranchForTmux(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"feature/auth", "wt-feature-auth"},
		{"feature-auth", "wt-feature-auth"},
		{"MM-12345", "wt-MM-12345"},
		{"release/v1.0.0", "wt-release-v100"},
		{"fix/bug:123", "wt-fix-bug123"},
		{"main", "wt-main"},
		{"feature/deep/nested/branch", "wt-feature-deep-nested-branch"},
		{"v2.0.0-rc.1", "wt-v200-rc1"},
		{"", "wt-"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeBranchForTmux(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeBranchForTmux(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeBranchForTmuxHasPrefix(t *testing.T) {
	// Every result must start with SessionPrefix
	branches := []string{"main", "feature/test", "MM-123"}
	for _, b := range branches {
		result := SanitizeBranchForTmux(b)
		if len(result) < len(SessionPrefix) || result[:len(SessionPrefix)] != SessionPrefix {
			t.Errorf("SanitizeBranchForTmux(%q) = %q, missing prefix %q", b, result, SessionPrefix)
		}
	}
}

func TestIsTmuxAvailable(t *testing.T) {
	// Just verify it doesn't panic; result depends on the test machine
	_ = IsTmuxAvailable()
}

func TestShellescape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    "hello",
			expected: "'hello'",
		},
		{
			name:     "string with spaces",
			input:    "/path/to/my directory",
			expected: "'/path/to/my directory'",
		},
		{
			name:     "string with single quote",
			input:    "it's",
			expected: "'it'\"'\"'s'",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "''",
		},
		{
			name:     "string with special chars",
			input:    "cmd --flag=value & stuff",
			expected: "'cmd --flag=value & stuff'",
		},
		{
			name:     "typical claude command",
			input:    "claude --continue --dangerously-skip-permissions",
			expected: "'claude --continue --dangerously-skip-permissions'",
		},
		{
			name:     "path with no special chars",
			input:    "/Users/nick/workspace/worktrees/repo/feature-auth",
			expected: "'/Users/nick/workspace/worktrees/repo/feature-auth'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellescape(tt.input)
			if result != tt.expected {
				t.Errorf("shellescape(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
