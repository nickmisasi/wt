package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nickmisasi/wt/internal"
)

const configUsage = `Usage: wt config <subcommand> [arguments]

Subcommands:
    show              Show all configuration values (JSON)
    get <key>         Get a configuration value
    set <key> <value> Set a configuration value

Available keys:
    editor.command    Editor command to use (default: cursor)
`

// RunConfig routes config subcommands.
func RunConfig(args []string) error {
	if len(args) == 0 {
		fmt.Print(configUsage)
		return nil
	}

	switch args[0] {
	case "show":
		return runConfigShow()
	case "get":
		return runConfigGet(args[1:])
	case "set":
		return runConfigSet(args[1:])
	default:
		return fmt.Errorf("unknown config subcommand: %s\n\n%s", args[0], configUsage)
	}
}

func runConfigShow() error {
	cfg, err := internal.LoadUserConfig()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func runConfigGet(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: wt config get <key>\n\nAvailable keys: %s",
			strings.Join(internal.ValidKeyNames(), ", "))
	}

	key := args[0]
	if !internal.IsValidKey(key) {
		return fmt.Errorf("unknown config key: %s (valid keys: %s)",
			key, strings.Join(internal.ValidKeyNames(), ", "))
	}

	cfg, err := internal.LoadUserConfig()
	if err != nil {
		return err
	}

	val, err := cfg.GetConfigValue(key)
	if err != nil {
		return err
	}

	fmt.Println(val)
	return nil
}

func runConfigSet(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: wt config set <key> <value>\n\nAvailable keys: %s",
			strings.Join(internal.ValidKeyNames(), ", "))
	}

	key := args[0]
	value := strings.Join(args[1:], " ")

	if !internal.IsValidKey(key) {
		return fmt.Errorf("unknown config key: %s (valid keys: %s)",
			key, strings.Join(internal.ValidKeyNames(), ", "))
	}

	cfg, err := internal.LoadUserConfig()
	if err != nil {
		return err
	}

	if err := cfg.SetConfigValue(key, value); err != nil {
		return err
	}

	if err := internal.SaveUserConfig(cfg); err != nil {
		return err
	}

	fmt.Printf("%s = %s\n", internal.NormalizeKey(key), value)
	return nil
}
