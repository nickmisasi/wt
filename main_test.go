package main

import (
	"testing"

	"github.com/nickmisasi/wt/cmd"
)

func TestParseCleanArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantDays  int
		wantBatch int
		wantDirty bool
		wantYes   bool
	}{
		{
			name:      "When no args are given it should return defaults",
			args:      []string{},
			wantDays:  cmd.DefaultStaleDays,
			wantBatch: cmd.DefaultCleanBatch,
			wantDirty: false,
			wantYes:   false,
		},
		{
			name:      "When --days is set it should override default days",
			args:      []string{"--days", "90"},
			wantDays:  90,
			wantBatch: cmd.DefaultCleanBatch,
			wantDirty: false,
			wantYes:   false,
		},
		{
			name:      "When -d shorthand is used it should set days",
			args:      []string{"-d", "7"},
			wantDays:  7,
			wantBatch: cmd.DefaultCleanBatch,
			wantDirty: false,
			wantYes:   false,
		},
		{
			name:      "When --batch is set it should override default batch size",
			args:      []string{"--batch", "50"},
			wantDays:  cmd.DefaultStaleDays,
			wantBatch: 50,
			wantDirty: false,
			wantYes:   false,
		},
		{
			name:      "When --dirty is set it should enable dirty removal",
			args:      []string{"--dirty"},
			wantDays:  cmd.DefaultStaleDays,
			wantBatch: cmd.DefaultCleanBatch,
			wantDirty: true,
			wantYes:   false,
		},
		{
			name:      "When --yes is set it should skip confirmation",
			args:      []string{"--yes"},
			wantDays:  cmd.DefaultStaleDays,
			wantBatch: cmd.DefaultCleanBatch,
			wantDirty: false,
			wantYes:   true,
		},
		{
			name:      "When -y shorthand is used it should skip confirmation",
			args:      []string{"-y"},
			wantDays:  cmd.DefaultStaleDays,
			wantBatch: cmd.DefaultCleanBatch,
			wantDirty: false,
			wantYes:   true,
		},
		{
			name:      "When all flags are combined it should parse all correctly",
			args:      []string{"-d", "14", "--batch", "20", "--dirty", "-y"},
			wantDays:  14,
			wantBatch: 20,
			wantDirty: true,
			wantYes:   true,
		},
		{
			name:      "When days is zero it should fall back to default",
			args:      []string{"-d", "0"},
			wantDays:  cmd.DefaultStaleDays,
			wantBatch: cmd.DefaultCleanBatch,
			wantDirty: false,
			wantYes:   false,
		},
		{
			name:      "When days is negative it should fall back to default",
			args:      []string{"-d", "-5"},
			wantDays:  cmd.DefaultStaleDays,
			wantBatch: cmd.DefaultCleanBatch,
			wantDirty: false,
			wantYes:   false,
		},
		{
			name:      "When batch is zero it should fall back to default",
			args:      []string{"--batch", "0"},
			wantDays:  cmd.DefaultStaleDays,
			wantBatch: cmd.DefaultCleanBatch,
			wantDirty: false,
			wantYes:   false,
		},
		{
			name:      "When --days has no value it should keep default",
			args:      []string{"--days"},
			wantDays:  cmd.DefaultStaleDays,
			wantBatch: cmd.DefaultCleanBatch,
			wantDirty: false,
			wantYes:   false,
		},
		{
			name:      "When flags are in reverse order it should parse all correctly",
			args:      []string{"-y", "--dirty", "--batch", "5", "-d", "60"},
			wantDays:  60,
			wantBatch: 5,
			wantDirty: true,
			wantYes:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			days, batch, dirty, yes := parseCleanArgs(tt.args)
			if days != tt.wantDays {
				t.Errorf("days = %d, want %d", days, tt.wantDays)
			}
			if batch != tt.wantBatch {
				t.Errorf("batch = %d, want %d", batch, tt.wantBatch)
			}
			if dirty != tt.wantDirty {
				t.Errorf("dirty = %v, want %v", dirty, tt.wantDirty)
			}
			if yes != tt.wantYes {
				t.Errorf("yes = %v, want %v", yes, tt.wantYes)
			}
		})
	}
}

func TestParseRemoveArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantBranch string
		wantForce  bool
	}{
		{
			name:       "When only branch is given it should return branch without force",
			args:       []string{"my-branch"},
			wantBranch: "my-branch",
			wantForce:  false,
		},
		{
			name:       "When -f is given it should enable force",
			args:       []string{"my-branch", "-f"},
			wantBranch: "my-branch",
			wantForce:  true,
		},
		{
			name:       "When --force is given it should enable force",
			args:       []string{"my-branch", "--force"},
			wantBranch: "my-branch",
			wantForce:  true,
		},
		{
			name:       "When -f comes before branch it should parse both",
			args:       []string{"-f", "my-branch"},
			wantBranch: "my-branch",
			wantForce:  true,
		},
		{
			name:       "When no args are given it should return empty branch",
			args:       []string{},
			wantBranch: "",
			wantForce:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branch, force := parseRemoveArgs(tt.args)
			if branch != tt.wantBranch {
				t.Errorf("branch = %q, want %q", branch, tt.wantBranch)
			}
			if force != tt.wantForce {
				t.Errorf("force = %v, want %v", force, tt.wantForce)
			}
		})
	}
}
