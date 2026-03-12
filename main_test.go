package main

import "testing"

func TestParseCheckoutCommandArgs(t *testing.T) {
	t.Run("parses json and dry-run flags", func(t *testing.T) {
		parsed, err := parseCheckoutCommandArgs([]string{"feature/test", "--json", "--dry-run", "-b", "main"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if parsed.branch != "feature/test" {
			t.Fatalf("unexpected branch: %s", parsed.branch)
		}
		if !parsed.jsonOutput || !parsed.dryRun {
			t.Fatalf("expected jsonOutput and dryRun to be true")
		}
		if parsed.baseBranch != "main" {
			t.Fatalf("unexpected base branch: %s", parsed.baseBranch)
		}
	})

	t.Run("fails when branch is missing", func(t *testing.T) {
		if _, err := parseCheckoutCommandArgs([]string{"--json"}); err == nil {
			t.Fatalf("expected error when branch is missing")
		}
	})

	t.Run("fails for unknown flags", func(t *testing.T) {
		if _, err := parseCheckoutCommandArgs([]string{"feature/test", "--unknown"}); err == nil {
			t.Fatalf("expected error for unknown option")
		}
	})

	t.Run("parses --claudemux flag", func(t *testing.T) {
		parsed, err := parseCheckoutCommandArgs([]string{"feature/test", "--claudemux"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if parsed.claudemux == nil || *parsed.claudemux != true {
			t.Fatalf("expected claudemux to be *bool(true), got %v", parsed.claudemux)
		}
	})

	t.Run("parses --no-claudemux flag", func(t *testing.T) {
		parsed, err := parseCheckoutCommandArgs([]string{"feature/test", "--no-claudemux"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if parsed.claudemux == nil || *parsed.claudemux != false {
			t.Fatalf("expected claudemux to be *bool(false), got %v", parsed.claudemux)
		}
	})

	t.Run("claudemux is nil when not specified", func(t *testing.T) {
		parsed, err := parseCheckoutCommandArgs([]string{"feature/test"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if parsed.claudemux != nil {
			t.Fatalf("expected claudemux to be nil, got %v", parsed.claudemux)
		}
	})

	t.Run("parses --claudemux with -b and branch", func(t *testing.T) {
		parsed, err := parseCheckoutCommandArgs([]string{"--claudemux", "-b", "main", "feature/x"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if parsed.branch != "feature/x" {
			t.Fatalf("unexpected branch: %s", parsed.branch)
		}
		if parsed.baseBranch != "main" {
			t.Fatalf("unexpected base branch: %s", parsed.baseBranch)
		}
		if parsed.claudemux == nil || *parsed.claudemux != true {
			t.Fatalf("expected claudemux to be *bool(true)")
		}
	})
}

func TestParseListArgs(t *testing.T) {
	jsonOutput, err := parseListArgs([]string{"--json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !jsonOutput {
		t.Fatalf("expected jsonOutput=true")
	}

	if _, err := parseListArgs([]string{"--unknown"}); err == nil {
		t.Fatalf("expected unknown option to fail")
	}
}

func TestParseRemoveArgs(t *testing.T) {
	branch, force, jsonOutput, err := parseRemoveArgs([]string{"feature/test", "-f", "--json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "feature/test" || !force || !jsonOutput {
		t.Fatalf("unexpected parse result: branch=%s force=%v json=%v", branch, force, jsonOutput)
	}

	if _, _, _, err := parseRemoveArgs([]string{"--json"}); err == nil {
		t.Fatalf("expected missing branch to fail")
	}
	if _, _, _, err := parseRemoveArgs([]string{"feature/test", "--unknown"}); err == nil {
		t.Fatalf("expected unknown option to fail")
	}
}
