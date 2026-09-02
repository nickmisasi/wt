package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const zshShellFunctionTemplate = `
# wt-shell-integration
wt() {
    local output
    output=$(%s "$@")
    local exit_code=$?
    
    if echo "$output" | grep -q "^__WT_CD__:"; then
        local new_dir=$(echo "$output" | grep "^__WT_CD__:" | cut -d':' -f2-)
        builtin cd "$new_dir" || return 1
        
        # Check if there's a post-setup command to run
        if echo "$output" | grep -q "^__WT_CMD__:"; then
            local cmd=$(echo "$output" | grep "^__WT_CMD__:" | cut -d':' -f2-)
            echo "Running setup: $cmd"
            eval "$cmd"
        fi
        
        # Show output without markers
        echo "$output" | grep -v "^__WT_CD__:" | grep -v "^__WT_CMD__:"
    else
        echo "$output"
    fi
    
    return $exit_code
}

# Smart cd for worktrees - makes "cd .." from worktree root go to workspace
cd() {
    if [[ "$1" == ".." ]]; then
        local parent_dir="${PWD%%/*}"
        if [[ "$parent_dir" == "%s" ]]; then
            builtin cd "%s"
            return
        fi
    fi
    builtin cd "$@"
}
# end wt-shell-integration
`

func installZsh(ctx *installContext) error {
	zshrcPath := filepath.Join(ctx.homeDir, ".zshrc")

	content, err := os.ReadFile(zshrcPath)
	alreadyInstalled := false
	if err == nil {
		alreadyInstalled = strings.Contains(string(content), shellFunctionMarker)
	}

	if !alreadyInstalled {
		f, err := os.OpenFile(zshrcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open .zshrc: %w", err)
		}
		defer f.Close()

		functionCode := fmt.Sprintf(zshShellFunctionTemplate, ctx.wtPath, ctx.worktreesPath, ctx.workspaceRoot)
		if _, err := f.WriteString("\n" + functionCode); err != nil {
			return fmt.Errorf("failed to write to .zshrc: %w", err)
		}

		fmt.Println("✓ Added shell function to ~/.zshrc")
	} else {
		fmt.Println("✓ Shell function already installed in ~/.zshrc")
	}

	completionInstalled, err := installZshCompletion(ctx.homeDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to install completions: %v\n", err)
	} else if completionInstalled {
		fmt.Println("✓ Installed zsh completions")
	}

	printZshInstallComplete()
	return nil
}

func installZshCompletion(homeDir string) (bool, error) {
	completionDirs := []string{
		"/usr/local/share/zsh/site-functions",
		filepath.Join(homeDir, ".zsh", "completion"),
		filepath.Join(homeDir, ".oh-my-zsh", "completions"),
	}

	var targetDir string
	for _, dir := range completionDirs {
		if err := os.MkdirAll(dir, 0755); err == nil {
			targetDir = dir
			break
		}
	}

	if targetDir == "" {
		return false, fmt.Errorf("no suitable completion directory found")
	}

	completionFile := filepath.Join(targetDir, "_wt")

	if _, err := os.Stat(completionFile); err == nil {
		content, err := os.ReadFile(completionFile)
		if err == nil && strings.Contains(string(content), "#compdef wt") {
			return false, nil
		}
	}

	err := os.WriteFile(completionFile, []byte(zshCompletionScript), 0644)
	if err != nil {
		return false, fmt.Errorf("failed to write completion file: %w", err)
	}

	return true, nil
}

func printZshInstallComplete() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Installation complete!")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\nTo start using wt, either:")
	fmt.Println("  1. Restart your terminal, or")
	fmt.Println("  2. Run: source ~/.zshrc")
	fmt.Println("\nThen try: wt help")
	fmt.Println("\nIf TAB completion doesn't appear, verify your zsh is set up for completions:")
	fmt.Println("  - Initialize compinit (in ~/.zshrc):")
	fmt.Println("      autoload -Uz compinit && compinit -i")
	fmt.Println("  - Ensure the user completion directory is on $fpath (before compinit):")
	fmt.Println("      fpath=(\"$HOME/.zsh/completion\" $fpath)")
	fmt.Println("      typeset -U fpath")
	fmt.Println("  - After changing ~/.zshrc: source ~/.zshrc or open a new terminal")
	fmt.Println()
}
