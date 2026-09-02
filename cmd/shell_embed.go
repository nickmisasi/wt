package cmd

import _ "embed"

//go:embed shell/zsh/_wt
var zshCompletionScript string

//go:embed shell/fish/wt.fish
var fishCompletionScript string
