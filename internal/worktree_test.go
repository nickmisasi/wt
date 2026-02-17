package internal

import (
	"os"
	"path/filepath"
	"testing"
)

// --- CopyClaudeConfig tests ---

func TestCopyClaudeConfig(t *testing.T) {
	t.Run("copies .claude directory when it exists", func(t *testing.T) {
		srcRoot := t.TempDir()
		dstRoot := t.TempDir()

		claudeDir := filepath.Join(srcRoot, ".claude")
		os.MkdirAll(claudeDir, 0755)
		os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(`{"key":"value"}`), 0644)

		CopyClaudeConfig(srcRoot, dstRoot)

		data, err := os.ReadFile(filepath.Join(dstRoot, ".claude", "settings.local.json"))
		if err != nil {
			t.Fatalf("expected settings.local.json to be copied: %v", err)
		}
		if string(data) != `{"key":"value"}` {
			t.Errorf("expected copied content to match, got %q", string(data))
		}
	})

	t.Run("copies nested subdirectories", func(t *testing.T) {
		srcRoot := t.TempDir()
		dstRoot := t.TempDir()

		subDir := filepath.Join(srcRoot, ".claude", "subdir")
		os.MkdirAll(subDir, 0755)
		os.WriteFile(filepath.Join(subDir, "nested.json"), []byte("nested"), 0644)

		CopyClaudeConfig(srcRoot, dstRoot)

		data, err := os.ReadFile(filepath.Join(dstRoot, ".claude", "subdir", "nested.json"))
		if err != nil {
			t.Fatalf("expected nested file to be copied: %v", err)
		}
		if string(data) != "nested" {
			t.Errorf("expected nested content to match, got %q", string(data))
		}
	})

	t.Run("no-op when .claude directory does not exist", func(t *testing.T) {
		srcRoot := t.TempDir()
		dstRoot := t.TempDir()

		CopyClaudeConfig(srcRoot, dstRoot)

		if _, err := os.Stat(filepath.Join(dstRoot, ".claude")); !os.IsNotExist(err) {
			t.Error("expected .claude directory to not be created when source is absent")
		}
	})

	t.Run("no-op when source root does not exist", func(t *testing.T) {
		dstRoot := t.TempDir()

		// srcRoot path that doesn't exist at all
		CopyClaudeConfig("/nonexistent/path/to/repo", dstRoot)

		if _, err := os.Stat(filepath.Join(dstRoot, ".claude")); !os.IsNotExist(err) {
			t.Error("expected .claude directory to not be created when source root is absent")
		}
	})

	t.Run("preserves file permissions", func(t *testing.T) {
		srcRoot := t.TempDir()
		dstRoot := t.TempDir()

		claudeDir := filepath.Join(srcRoot, ".claude")
		os.MkdirAll(claudeDir, 0755)
		os.WriteFile(filepath.Join(claudeDir, "script.sh"), []byte("#!/bin/sh"), 0755)

		CopyClaudeConfig(srcRoot, dstRoot)

		info, err := os.Stat(filepath.Join(dstRoot, ".claude", "script.sh"))
		if err != nil {
			t.Fatalf("expected script.sh to be copied: %v", err)
		}
		if info.Mode().Perm() != 0755 {
			t.Errorf("expected permissions 0755, got %04o", info.Mode().Perm())
		}
	})

	t.Run("handles symlinks inside .claude directory", func(t *testing.T) {
		srcRoot := t.TempDir()
		dstRoot := t.TempDir()

		claudeDir := filepath.Join(srcRoot, ".claude")
		os.MkdirAll(claudeDir, 0755)

		os.WriteFile(filepath.Join(claudeDir, "real.json"), []byte("real"), 0644)
		os.Symlink("real.json", filepath.Join(claudeDir, "link.json"))

		CopyClaudeConfig(srcRoot, dstRoot)

		target, err := os.Readlink(filepath.Join(dstRoot, ".claude", "link.json"))
		if err != nil {
			t.Fatalf("expected link.json to be a symlink: %v", err)
		}
		if target != "real.json" {
			t.Errorf("expected symlink target 'real.json', got %q", target)
		}
	})

	t.Run("does not fail worktree creation on copy error", func(t *testing.T) {
		srcRoot := t.TempDir()

		claudeDir := filepath.Join(srcRoot, ".claude")
		os.MkdirAll(claudeDir, 0755)
		os.WriteFile(filepath.Join(claudeDir, "test.json"), []byte("test"), 0644)

		dstRoot := t.TempDir()
		os.MkdirAll(filepath.Join(dstRoot, ".claude"), 0000)
		defer os.Chmod(filepath.Join(dstRoot, ".claude"), 0755)

		// Should not panic — errors are printed as warnings
		CopyClaudeConfig(srcRoot, dstRoot)
	})

	t.Run("copies multiple files", func(t *testing.T) {
		srcRoot := t.TempDir()
		dstRoot := t.TempDir()

		claudeDir := filepath.Join(srcRoot, ".claude")
		os.MkdirAll(claudeDir, 0755)
		os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte("local"), 0644)
		os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("global"), 0644)
		os.WriteFile(filepath.Join(claudeDir, "commands.md"), []byte("commands"), 0644)

		CopyClaudeConfig(srcRoot, dstRoot)

		for _, name := range []string{"settings.local.json", "settings.json", "commands.md"} {
			if _, err := os.Stat(filepath.Join(dstRoot, ".claude", name)); err != nil {
				t.Errorf("expected %s to be copied: %v", name, err)
			}
		}
	})

	t.Run("warns on stat error other than not-exist", func(t *testing.T) {
		srcRoot := t.TempDir()
		dstRoot := t.TempDir()

		// Create .claude as a directory but make the parent unreadable
		// so Stat on .claude fails with permission denied
		claudeDir := filepath.Join(srcRoot, ".claude")
		os.MkdirAll(claudeDir, 0755)
		os.WriteFile(filepath.Join(claudeDir, "test.json"), []byte("test"), 0644)
		os.Chmod(srcRoot, 0000)
		defer os.Chmod(srcRoot, 0755)

		// Should not panic — prints warning and returns
		CopyClaudeConfig(srcRoot, dstRoot)

		// Destination .claude should not exist since source couldn't be accessed
		if _, err := os.Stat(filepath.Join(dstRoot, ".claude")); !os.IsNotExist(err) {
			t.Error("expected .claude directory to not be created when source is inaccessible")
		}
	})
}

// --- copyDir tests ---

func TestCopyDir(t *testing.T) {
	t.Run("copies empty directory", func(t *testing.T) {
		src := t.TempDir()
		dst := filepath.Join(t.TempDir(), "dest")

		emptyDir := filepath.Join(src, "empty")
		os.MkdirAll(emptyDir, 0755)

		if err := copyDir(emptyDir, dst); err != nil {
			t.Fatalf("copyDir failed: %v", err)
		}

		info, err := os.Stat(dst)
		if err != nil {
			t.Fatalf("expected destination to exist: %v", err)
		}
		if !info.IsDir() {
			t.Error("expected destination to be a directory")
		}

		entries, _ := os.ReadDir(dst)
		if len(entries) != 0 {
			t.Errorf("expected empty directory, got %d entries", len(entries))
		}
	})

	t.Run("copies flat directory with files", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "src")
		dst := filepath.Join(t.TempDir(), "dst")
		os.MkdirAll(src, 0755)

		os.WriteFile(filepath.Join(src, "a.txt"), []byte("aaa"), 0644)
		os.WriteFile(filepath.Join(src, "b.txt"), []byte("bbb"), 0600)

		if err := copyDir(src, dst); err != nil {
			t.Fatalf("copyDir failed: %v", err)
		}

		data, err := os.ReadFile(filepath.Join(dst, "a.txt"))
		if err != nil {
			t.Fatalf("expected a.txt to exist: %v", err)
		}
		if string(data) != "aaa" {
			t.Errorf("expected 'aaa', got %q", string(data))
		}

		data, err = os.ReadFile(filepath.Join(dst, "b.txt"))
		if err != nil {
			t.Fatalf("expected b.txt to exist: %v", err)
		}
		if string(data) != "bbb" {
			t.Errorf("expected 'bbb', got %q", string(data))
		}
	})

	t.Run("copies deeply nested structure", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "src")
		dst := filepath.Join(t.TempDir(), "dst")

		deep := filepath.Join(src, "a", "b", "c")
		os.MkdirAll(deep, 0755)
		os.WriteFile(filepath.Join(deep, "deep.txt"), []byte("deep"), 0644)
		os.WriteFile(filepath.Join(src, "a", "top.txt"), []byte("top"), 0644)

		if err := copyDir(src, dst); err != nil {
			t.Fatalf("copyDir failed: %v", err)
		}

		data, err := os.ReadFile(filepath.Join(dst, "a", "b", "c", "deep.txt"))
		if err != nil {
			t.Fatalf("expected deep.txt to exist: %v", err)
		}
		if string(data) != "deep" {
			t.Errorf("expected 'deep', got %q", string(data))
		}

		data, err = os.ReadFile(filepath.Join(dst, "a", "top.txt"))
		if err != nil {
			t.Fatalf("expected top.txt to exist: %v", err)
		}
		if string(data) != "top" {
			t.Errorf("expected 'top', got %q", string(data))
		}
	})

	t.Run("preserves file permissions", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "src")
		dst := filepath.Join(t.TempDir(), "dst")
		os.MkdirAll(src, 0755)

		os.WriteFile(filepath.Join(src, "exec.sh"), []byte("#!/bin/sh"), 0755)
		os.WriteFile(filepath.Join(src, "readonly.txt"), []byte("ro"), 0444)

		if err := copyDir(src, dst); err != nil {
			t.Fatalf("copyDir failed: %v", err)
		}

		info, _ := os.Stat(filepath.Join(dst, "exec.sh"))
		if info.Mode().Perm() != 0755 {
			t.Errorf("expected exec.sh permissions 0755, got %04o", info.Mode().Perm())
		}

		info, _ = os.Stat(filepath.Join(dst, "readonly.txt"))
		if info.Mode().Perm() != 0444 {
			t.Errorf("expected readonly.txt permissions 0444, got %04o", info.Mode().Perm())
		}
	})

	t.Run("recreates symlinks to files", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "src")
		dst := filepath.Join(t.TempDir(), "dst")
		os.MkdirAll(src, 0755)

		os.WriteFile(filepath.Join(src, "target.txt"), []byte("target"), 0644)
		os.Symlink("target.txt", filepath.Join(src, "link.txt"))

		if err := copyDir(src, dst); err != nil {
			t.Fatalf("copyDir failed: %v", err)
		}

		// Verify it's a symlink, not a regular file
		linfo, err := os.Lstat(filepath.Join(dst, "link.txt"))
		if err != nil {
			t.Fatalf("expected link.txt to exist: %v", err)
		}
		if linfo.Mode()&os.ModeSymlink == 0 {
			t.Error("expected link.txt to be a symlink")
		}

		target, err := os.Readlink(filepath.Join(dst, "link.txt"))
		if err != nil {
			t.Fatalf("expected to read symlink: %v", err)
		}
		if target != "target.txt" {
			t.Errorf("expected symlink target 'target.txt', got %q", target)
		}

		// Verify the target file was also copied
		data, _ := os.ReadFile(filepath.Join(dst, "target.txt"))
		if string(data) != "target" {
			t.Errorf("expected target content 'target', got %q", string(data))
		}
	})

	t.Run("recreates symlinks to directories", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "src")
		dst := filepath.Join(t.TempDir(), "dst")

		subDir := filepath.Join(src, "realdir")
		os.MkdirAll(subDir, 0755)
		os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("inside"), 0644)
		os.Symlink("realdir", filepath.Join(src, "linkdir"))

		if err := copyDir(src, dst); err != nil {
			t.Fatalf("copyDir failed: %v", err)
		}

		// linkdir should be a symlink pointing to "realdir"
		linfo, err := os.Lstat(filepath.Join(dst, "linkdir"))
		if err != nil {
			t.Fatalf("expected linkdir to exist: %v", err)
		}
		if linfo.Mode()&os.ModeSymlink == 0 {
			t.Error("expected linkdir to be a symlink")
		}

		target, _ := os.Readlink(filepath.Join(dst, "linkdir"))
		if target != "realdir" {
			t.Errorf("expected symlink target 'realdir', got %q", target)
		}
	})

	t.Run("preserves relative symlink targets", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "src")
		dst := filepath.Join(t.TempDir(), "dst")

		os.MkdirAll(filepath.Join(src, "sub"), 0755)
		os.WriteFile(filepath.Join(src, "sub", "real.txt"), []byte("data"), 0644)
		os.Symlink("sub/real.txt", filepath.Join(src, "link.txt"))

		if err := copyDir(src, dst); err != nil {
			t.Fatalf("copyDir failed: %v", err)
		}

		target, _ := os.Readlink(filepath.Join(dst, "link.txt"))
		if target != "sub/real.txt" {
			t.Errorf("expected relative symlink target 'sub/real.txt', got %q", target)
		}
	})

	t.Run("returns error for nonexistent source", func(t *testing.T) {
		dst := filepath.Join(t.TempDir(), "dst")

		err := copyDir("/nonexistent/source", dst)
		if err == nil {
			t.Error("expected error for nonexistent source directory")
		}
	})

	t.Run("returns error for unwritable destination parent", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "src")
		os.MkdirAll(src, 0755)
		os.WriteFile(filepath.Join(src, "f.txt"), []byte("f"), 0644)

		parent := filepath.Join(t.TempDir(), "locked")
		os.MkdirAll(parent, 0000)
		defer os.Chmod(parent, 0755)

		dst := filepath.Join(parent, "dst")
		err := copyDir(src, dst)
		if err == nil {
			t.Error("expected error when destination parent is unwritable")
		}
	})
}

// --- copyFile tests ---

func TestCopyFile(t *testing.T) {
	t.Run("copies file content", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "src.txt")
		dst := filepath.Join(t.TempDir(), "dst.txt")

		os.WriteFile(src, []byte("hello world"), 0644)

		if err := copyFile(src, dst); err != nil {
			t.Fatalf("copyFile failed: %v", err)
		}

		data, _ := os.ReadFile(dst)
		if string(data) != "hello world" {
			t.Errorf("expected 'hello world', got %q", string(data))
		}
	})

	t.Run("copies empty file", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "empty.txt")
		dst := filepath.Join(t.TempDir(), "dst.txt")

		os.WriteFile(src, []byte{}, 0644)

		if err := copyFile(src, dst); err != nil {
			t.Fatalf("copyFile failed: %v", err)
		}

		info, _ := os.Stat(dst)
		if info.Size() != 0 {
			t.Errorf("expected empty file, got size %d", info.Size())
		}
	})

	t.Run("preserves executable permission", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "script.sh")
		dst := filepath.Join(t.TempDir(), "script-copy.sh")

		os.WriteFile(src, []byte("#!/bin/sh\necho hi"), 0755)

		if err := copyFile(src, dst); err != nil {
			t.Fatalf("copyFile failed: %v", err)
		}

		info, _ := os.Stat(dst)
		if info.Mode().Perm() != 0755 {
			t.Errorf("expected 0755, got %04o", info.Mode().Perm())
		}
	})

	t.Run("preserves read-only permission", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "ro.txt")
		dst := filepath.Join(t.TempDir(), "ro-copy.txt")

		os.WriteFile(src, []byte("readonly"), 0444)

		if err := copyFile(src, dst); err != nil {
			t.Fatalf("copyFile failed: %v", err)
		}

		info, _ := os.Stat(dst)
		if info.Mode().Perm() != 0444 {
			t.Errorf("expected 0444, got %04o", info.Mode().Perm())
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "src.txt")
		dstDir := t.TempDir()
		dst := filepath.Join(dstDir, "a", "b", "c", "dst.txt")

		os.WriteFile(src, []byte("deep"), 0644)

		if err := copyFile(src, dst); err != nil {
			t.Fatalf("copyFile failed: %v", err)
		}

		data, _ := os.ReadFile(dst)
		if string(data) != "deep" {
			t.Errorf("expected 'deep', got %q", string(data))
		}
	})

	t.Run("returns error for nonexistent source", func(t *testing.T) {
		dst := filepath.Join(t.TempDir(), "dst.txt")

		err := copyFile("/nonexistent/file.txt", dst)
		if err == nil {
			t.Error("expected error for nonexistent source file")
		}
	})

	t.Run("overwrites existing destination", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "src.txt")
		dst := filepath.Join(t.TempDir(), "dst.txt")

		os.WriteFile(src, []byte("new content"), 0644)
		os.WriteFile(dst, []byte("old content"), 0644)

		if err := copyFile(src, dst); err != nil {
			t.Fatalf("copyFile failed: %v", err)
		}

		data, _ := os.ReadFile(dst)
		if string(data) != "new content" {
			t.Errorf("expected 'new content', got %q", string(data))
		}
	})
}
