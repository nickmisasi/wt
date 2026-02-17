package cmd

import (
	"fmt"
	"os"
)

// RunCursor is deprecated. It prints a deprecation notice and delegates to RunEdit.
func RunCursor(config interface{}, gitRepo interface{}, branch string, baseBranch string, noClaudeDocs bool) error {
	fmt.Fprintln(os.Stderr, "WARNING: 'wt cursor' is deprecated, use 'wt edit' instead.")
	fmt.Fprintln(os.Stderr, "  Configure your editor with: wt config set editor <editor>")
	fmt.Fprintln(os.Stderr)
	return RunEdit(config, gitRepo, branch, baseBranch, noClaudeDocs)
}
