package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const shellFunctionMarker = "# wt-shell-integration"

const shellFunction = `
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

# Smart cd for worktrees - makes "cd .." from worktree root go to ~/workspace
cd() {
    # Only intercept "cd .." from worktree root
    if [[ "$1" == ".." ]]; then
        local parent_dir="${PWD%%/*}"  # Get parent directory
        # Check if parent is ~/workspace/worktrees
        if [[ "$parent_dir" == "$HOME/workspace/worktrees" ]]; then
            builtin cd "$HOME/workspace"
            return
        fi
    fi
    builtin cd "$@"
}
# end wt-shell-integration
`

const completionScript = `#compdef wt

_wt() {
    local curcontext="$curcontext" state line
    typeset -A opt_args

    _arguments -C \
        '1: :->command' \
        '*::arg:->args'

    case $state in
        command)
            _values 'wt command' \
                'ls[List worktrees]' \
                'co[Checkout/create worktree]' \
                'rm[Remove a worktree]' \
                'clean[Remove stale worktrees]' \
                'cursor[Open Cursor editor]' \
                'install[Install shell integration]' \
                'help[Show help]'
            ;;
        args)
            case $line[1] in
                co|cursor)
                    _arguments \
                        '1:branch:_wt_complete_branches' \
                        '-b[Base branch]:base branch:_wt_complete_branches' \
                        '--base[Base branch]:base branch:_wt_complete_branches' \
                        '-n[Skip running enable-claude-docs.sh]' \
                        '--no-claude-docs[Skip running enable-claude-docs.sh]'
                    ;;
                rm)
                    _arguments \
                        '1:branch:_wt_complete_branches' \
                        '-f[Force removal]' \
                        '--force[Force removal]'
                    ;;
            esac
            ;;
    esac
}

_wt_complete_branches() {
    local -a branches
    branches=()

    branches+=(${(f)"$(
        {
          git branch --format='%(refname:short)';
          git branch -r --format='%(refname:short)';
        } 2>/dev/null |
        sed -e 's|^origin/||' -e 's|^remotes/origin/||' -e 's|^remotes/||' |
        grep -v '^HEAD$' |
        sort -u
    )"})

    _describe -t branches 'branch' branches
}
`

// RunInstall installs the shell integration and completions
func RunInstall() error {
	// Get the path to the wt binary
	wtPath, err := exec.LookPath("wt")
	if err != nil {
		// If not in PATH, try to get current executable path
		wtPath, err = os.Executable()
		if err != nil {
			return fmt.Errorf("failed to determine wt executable path: %w", err)
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	zshrcPath := filepath.Join(homeDir, ".zshrc")

	// Check if shell function already exists
	content, err := os.ReadFile(zshrcPath)
	alreadyInstalled := false
	if err == nil {
		alreadyInstalled = strings.Contains(string(content), shellFunctionMarker)
	}

	if !alreadyInstalled {
		// Add shell function to .zshrc
		f, err := os.OpenFile(zshrcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open .zshrc: %w", err)
		}
		defer f.Close()

		functionCode := fmt.Sprintf(shellFunction, wtPath)
		if _, err := f.WriteString("\n" + functionCode); err != nil {
			return fmt.Errorf("failed to write to .zshrc: %w", err)
		}

		fmt.Println("✓ Added shell function to ~/.zshrc")
	} else {
		fmt.Println("✓ Shell function already installed in ~/.zshrc")
	}

	// Install completion script
	completionInstalled, err := installCompletion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to install completions: %v\n", err)
	} else if completionInstalled {
		fmt.Println("✓ Installed zsh completions")
	}

	// Print next steps
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

	return nil
}

// installCompletion installs the zsh completion script
func installCompletion() (bool, error) {
	// Try common completion directories
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}

	// Possible completion directories in order of preference
	completionDirs := []string{
		"/usr/local/share/zsh/site-functions",
		filepath.Join(homeDir, ".zsh", "completion"),
		filepath.Join(homeDir, ".oh-my-zsh", "completions"),
	}

	var targetDir string
	for _, dir := range completionDirs {
		// Check if directory exists or can be created
		if err := os.MkdirAll(dir, 0755); err == nil {
			targetDir = dir
			break
		}
	}

	if targetDir == "" {
		return false, fmt.Errorf("no suitable completion directory found")
	}

	completionFile := filepath.Join(targetDir, "_wt")

	// Check if completion already exists
	if _, err := os.Stat(completionFile); err == nil {
		// File exists, check if it's ours
		content, err := os.ReadFile(completionFile)
		if err == nil && strings.Contains(string(content), "#compdef wt") {
			return false, nil // Already installed
		}
	}

	// Write completion file
	err = os.WriteFile(completionFile, []byte(completionScript), 0644)
	if err != nil {
		return false, fmt.Errorf("failed to write completion file: %w", err)
	}

	return true, nil
}
