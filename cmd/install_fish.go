package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const fishFunctionTemplate = `# wt-shell-integration
function wt --description 'Git worktree manager with directory switching'
    set -l output (command %s $argv)
    set -l exit_code $status
    set -l saw_cd 0

    for line in $output
        if string match -q '__WT_CD__:*' -- $line
            set -l new_dir (string replace '__WT_CD__:' '' -- $line)
            cd $new_dir; or return $exit_code
            set saw_cd 1
        else if string match -q '__WT_CMD__:*' -- $line
            if test $saw_cd -eq 1
                set -l cmd (string replace '__WT_CMD__:' '' -- $line)
                echo "Running setup: $cmd"
                eval $cmd
            end
        else
            echo $line
        end
    end

    return $exit_code
end
`

const fishIntegrationTemplate = `# wt-shell-integration
function cd --wraps cd --description 'Smart cd for worktrees'
    if test (count $argv) -eq 1; and test "$argv[1]" = ..
        set -l parent_dir (dirname $PWD)
        if test "$parent_dir" = "%s"
            builtin cd "%s"
            return
        end
    end
    builtin cd $argv
end
# end wt-shell-integration
`

func installFish(ctx *installContext) error {
	fishConfigDir := filepath.Join(ctx.homeDir, ".config", "fish")
	functionsDir := filepath.Join(fishConfigDir, "functions")
	confDir := filepath.Join(fishConfigDir, "conf.d")
	completionsDir := filepath.Join(fishConfigDir, "completions")

	integrationPath := filepath.Join(confDir, "wt-integration.fish")
	alreadyInstalled := fileContainsMarker(integrationPath)

	if !alreadyInstalled {
		if err := os.MkdirAll(functionsDir, 0755); err != nil {
			return fmt.Errorf("failed to create fish functions directory: %w", err)
		}
		if err := os.MkdirAll(confDir, 0755); err != nil {
			return fmt.Errorf("failed to create fish conf.d directory: %w", err)
		}

		functionPath := filepath.Join(functionsDir, "wt.fish")
		functionCode := fmt.Sprintf(fishFunctionTemplate, ctx.wtPath)
		if err := os.WriteFile(functionPath, []byte(functionCode), 0644); err != nil {
			return fmt.Errorf("failed to write fish function: %w", err)
		}

		integrationCode := fmt.Sprintf(fishIntegrationTemplate, ctx.worktreesPath, ctx.workspaceRoot)
		if err := os.WriteFile(integrationPath, []byte(integrationCode), 0644); err != nil {
			return fmt.Errorf("failed to write fish integration config: %w", err)
		}

		fmt.Println("✓ Added shell function to \"$HOME/.config/fish/functions/wt.fish\"")
		fmt.Println("✓ Added smart cd wrapper to \"$HOME/.config/fish/conf.d/wt-integration.fish\"")
	} else {
		fmt.Println("✓ Shell integration already installed in \"$HOME/.config/fish/conf.d/wt-integration.fish\"")
	}

	completionInstalled, err := installFishCompletion(completionsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to install completions: %v\n", err)
	} else if completionInstalled {
		fmt.Println("✓ Installed fish completions")
	}

	printFishInstallComplete()
	return nil
}

func installFishCompletion(completionsDir string) (bool, error) {
	if err := os.MkdirAll(completionsDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create fish completions directory: %w", err)
	}

	completionFile := filepath.Join(completionsDir, "wt.fish")
	if fileContainsMarker(completionFile) {
		return false, nil
	}

	if err := os.WriteFile(completionFile, []byte(fishCompletionScript), 0644); err != nil {
		return false, fmt.Errorf("failed to write completion file: %w", err)
	}

	return true, nil
}

func printFishInstallComplete() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Installation complete!")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\nTo start using wt, open a new fish terminal.")
	fmt.Println("\nThen try: wt help")
	fmt.Println("\nFish loads functions and completions automatically from \"$HOME/.config/fish/\"")
	fmt.Println()
}

func fileContainsMarker(path string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), shellFunctionMarker)
}
