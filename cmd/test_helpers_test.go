package cmd

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGitCommand(t *testing.T, cwd string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
	return strings.TrimSpace(string(output))
}

func setupGitRepo(t *testing.T, repoPath string) {
	t.Helper()
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		t.Fatalf("failed to create repo directory: %v", err)
	}

	runGitCommand(t, repoPath, "init", "-b", "main")
	runGitCommand(t, repoPath, "config", "user.email", "test@example.com")
	runGitCommand(t, repoPath, "config", "user.name", "Test User")

	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("test\n"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}
	runGitCommand(t, repoPath, "add", "README.md")
	runGitCommand(t, repoPath, "commit", "-m", "initial")
}

func withCwd(t *testing.T, dir string) {
	t.Helper()
	current, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(current)
	})
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	defer reader.Close()

	os.Stdout = writer
	callErr := fn()
	_ = writer.Close()
	os.Stdout = originalStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatalf("failed to read captured stdout: %v", err)
	}

	return strings.TrimSpace(buf.String()), callErr
}

func canonicalPath(path string) string {
	cleaned := filepath.Clean(path)
	resolved, err := filepath.EvalSymlinks(cleaned)
	if err == nil {
		return resolved
	}

	parent := filepath.Dir(cleaned)
	resolvedParent, parentErr := filepath.EvalSymlinks(parent)
	if parentErr == nil {
		return filepath.Join(resolvedParent, filepath.Base(cleaned))
	}

	return cleaned
}
