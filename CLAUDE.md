# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`wt` is a Go CLI tool for managing Git worktrees. It has zero external dependencies (standard library only). It includes special dual-repo worktree support for Mattermost's mattermost/enterprise workflow.

## Build & Test Commands

```bash
make build          # Build binary → ./wt
make install        # Build and install to /usr/local/bin or ~/bin
make test           # Run all tests (go test ./...)
make clean          # Remove built binary

# Run a single test
go test ./internal -run TestFunctionName -v

# Run all tests in a package
go test ./internal -v
```

## Architecture

**Entry point**: `main.go` — parses CLI args and routes to command handlers. Commands that don't need a git repo (help, install, config) are handled before git repo initialization.

**`cmd/`** — Command implementations. Each file is a subcommand:
- `checkout.go` — worktree creation/switching (the core workflow)
- `edit.go` — open worktree in configured editor; `cursor.go` is a deprecated alias
- `rm.go` / `clean.go` — worktree removal (individual and batch stale cleanup)
- `list.go` / `toggle.go` / `port.go` / `help.go` / `install.go` / `config.go`

**`internal/`** — Core business logic:
- `git.go` — Git operations wrapper (`GitRepo` struct)
- `config.go` — Runtime config (`Config` struct: repo name, root, worktree base path)
- `userconfig.go` — Persistent user settings stored as JSON (`~/.config/wt/config.json`)
- `worktree.go` — Worktree management (create, list, remove, path resolution)
- `mattermost.go` — Mattermost dual-repo logic: creates synchronized worktrees in both mattermost and enterprise repos, auto-assigns unique ports (8065-8999 range), copies config files, runs `make setup-go-work`

## Key Conventions

- Shell integration uses special output markers: `__WT_CD__:<path>` signals the shell wrapper to `cd`, `__WT_CMD__:<cmd>` signals a post-setup command to run.
- Worktrees are stored under `~/workspace/worktrees/<repo-name>-<branch>/`.
- The `Config` struct is initialized in `main.go` and threaded through command functions. `GitRepo` is only created when the command requires being inside a git repository.
- User config (`userconfig.go`) uses dot-notation keys normalized to lowercase (e.g., `editor.command`).
