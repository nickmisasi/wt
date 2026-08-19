package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nickmisasi/wt/internal"
)

const shellFunctionMarker = "# wt-shell-integration"

type installContext struct {
	wtPath        string
	worktreesPath string
	workspaceRoot string
	homeDir       string
}

// RunInstall installs the shell integration and completions
func RunInstall(args []string) error {
	shell, err := parseInstallShell(args)
	if err != nil {
		return err
	}

	ctx, err := resolveInstallContext()
	if err != nil {
		return err
	}

	switch shell {
	case "fish":
		return installFish(ctx)
	case "zsh":
		return installZsh(ctx)
	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}
}

func parseInstallShell(args []string) (string, error) {
	shell := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--shell" && i+1 < len(args) {
			shell = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--shell=") {
			shell = strings.TrimPrefix(arg, "--shell=")
			continue
		}
		return "", fmt.Errorf("unknown install argument: %s\nusage: wt install [--shell fish|zsh]", arg)
	}

	if shell != "" {
		if shell != "fish" && shell != "zsh" {
			return "", fmt.Errorf("unsupported shell: %s (supported: fish, zsh)", shell)
		}
		return shell, nil
	}

	return detectShell(), nil
}

func detectShell() string {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		fmt.Fprintln(os.Stderr, "Note: $SHELL is not set, defaulting to zsh")
		return "zsh"
	}

	switch filepath.Base(shellPath) {
	case "fish":
		return "fish"
	case "zsh":
		return "zsh"
	default:
		fmt.Fprintf(os.Stderr, "Note: unrecognized shell %q, defaulting to zsh\n", filepath.Base(shellPath))
		return "zsh"
	}
}

func resolveInstallContext() (*installContext, error) {
	wtPath, err := exec.LookPath("wt")
	if err != nil {
		wtPath, err = os.Executable()
		if err != nil {
			return nil, fmt.Errorf("failed to determine wt executable path: %w", err)
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	worktreesPath, err := internal.ResolveWorktreesPath()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve worktrees path: %w", err)
	}

	workspaceRoot, err := internal.ResolveWorkspaceRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve workspace root: %w", err)
	}

	return &installContext{
		wtPath:        wtPath,
		worktreesPath: worktreesPath,
		workspaceRoot: workspaceRoot,
		homeDir:       homeDir,
	}, nil
}
