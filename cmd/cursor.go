package cmd

import (
	"fmt"
	"os"

	"github.com/nickmisasi/wt/internal"
)

// RunCursor is deprecated. It prints a deprecation notice and delegates to RunEdit.
func RunCursor(cfg *internal.Config, repo *internal.GitRepo, branch string, baseBranch string, noClaudeDocs bool) error {
	fmt.Fprintln(os.Stderr, "WARNING: 'wt cursor' is deprecated, use 'wt edit' instead.")
	fmt.Fprintln(os.Stderr, "  Configure your editor with: wt config set editor.command <editor>")
	fmt.Fprintln(os.Stderr)
	return RunEdit(cfg, repo, branch, baseBranch, noClaudeDocs)
}
