# wt-shell-integration

function __wt_branches
    {
        git branch --format='%(refname:short)' 2>/dev/null
        git branch -r --format='%(refname:short)' 2>/dev/null
    } | sed -e 's|^origin/||' -e 's|^remotes/origin/||' -e 's|^remotes/||' |
      grep -v '^HEAD$' | sort -u
end

complete -c wt -f
complete -c wt -n '__fish_use_subcommand' -a ls -d 'List worktrees'
complete -c wt -n '__fish_use_subcommand' -a co -d 'Checkout/create worktree'
complete -c wt -n '__fish_use_subcommand' -a rm -d 'Remove a worktree'
complete -c wt -n '__fish_use_subcommand' -a clean -d 'Remove stale worktrees'
complete -c wt -n '__fish_use_subcommand' -a cursor -d 'Open Cursor editor'
complete -c wt -n '__fish_use_subcommand' -a edit -d 'Open configured editor'
complete -c wt -n '__fish_use_subcommand' -a config -d 'Manage configuration'
complete -c wt -n '__fish_use_subcommand' -a install -d 'Install shell integration'
complete -c wt -n '__fish_use_subcommand' -a help -d 'Show help'

complete -c wt -n '__fish_seen_subcommand_from co cursor edit rm' -a '(__wt_branches)' -d 'branch'
complete -c wt -n '__fish_seen_subcommand_from co cursor edit' -s b -l base -a '(__wt_branches)' -d 'Base branch'
complete -c wt -n '__fish_seen_subcommand_from co cursor edit' -s n -l no-claude-docs -d 'Skip running enable-claude-docs.sh'
complete -c wt -n '__fish_seen_subcommand_from rm' -s f -l force -d 'Force removal'
complete -c wt -n '__fish_seen_subcommand_from config' -a 'get set show' -d 'subcommand'
