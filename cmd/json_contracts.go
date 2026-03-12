package cmd

import (
	"encoding/json"
	"os"
)

// listJSONItem is the canonical machine-readable shape for `wt ls --json`.
// Keep this contract stable for CLI integrations.
type listJSONItem struct {
	Branch         string `json:"branch"`
	Path           string `json:"path"`
	IsDirty        bool   `json:"isDirty"`
	LastCommitUnix int64  `json:"lastCommitUnix"`
}

// checkoutJSONResponse is the canonical machine-readable shape for
// `wt co --json` (including `--dry-run`).
type checkoutJSONResponse struct {
	Mode              string   `json:"mode"`
	Branch            string   `json:"branch"`
	Created           bool     `json:"created"`
	Existing          bool     `json:"existing"`
	CdPath            string   `json:"cdPath"`
	WorktreePath      string   `json:"worktreePath"`
	PostSetupCommands []string `json:"postSetupCommands"`
}

// removeJSONResponse is the canonical machine-readable shape for `wt rm --json`.
type removeJSONResponse struct {
	Mode         string   `json:"mode"`
	Branch       string   `json:"branch"`
	Removed      bool     `json:"removed"`
	WorktreePath string   `json:"worktreePath"`
	RemovedPaths []string `json:"removedPaths"`
}

func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func runSilently(enabled bool, fn func() error) error {
	if !enabled {
		return fn()
	}

	originalStdout := os.Stdout
	tmpFile, err := os.CreateTemp("", "wt-quiet-*.log")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	os.Stdout = tmpFile
	defer func() {
		os.Stdout = originalStdout
	}()

	return fn()
}
