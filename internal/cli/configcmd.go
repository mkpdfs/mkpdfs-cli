package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/sim4gh/mkpdfs-cli/internal/config"
	"github.com/sim4gh/mkpdfs-cli/internal/envs"
	"github.com/spf13/cobra"
)

func addConfigCommands() {
	cfgCmd := &cobra.Command{Use: "config", Short: "View and update CLI configuration"}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List current configuration (secrets masked)",
		RunE:  runConfigList,
	}

	pathCmd := &cobra.Command{
		Use:   "path",
		Short: "Print the config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(config.Path())
			return nil
		},
	}

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value (supported: environment)",
		Args:  cobra.ExactArgs(2),
		RunE:  runConfigSet,
	}

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value (supported: environment)",
		Args:  cobra.ExactArgs(1),
		RunE:  runConfigGet,
	}

	cfgCmd.AddCommand(listCmd, pathCmd, setCmd, getCmd)
	requireSubcommand(cfgCmd)
	rootCmd.AddCommand(cfgCmd)
}

func runConfigList(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	bold := color.New(color.Bold)
	bold.Printf("Active environment: %s\n", cfg.Environment)
	fmt.Println()

	if len(cfg.Environments) == 0 {
		fmt.Println("  (no environments configured)")
		return nil
	}

	for envName, creds := range cfg.Environments {
		bold.Printf("[%s]\n", envName)
		fmt.Printf("  IDToken:      %s\n", config.MaskToken(creds.IDToken))
		fmt.Printf("  RefreshToken: %s\n", config.MaskToken(creds.RefreshToken))
		fmt.Printf("  APIKey:       %s\n", config.MaskToken(creds.APIKey))
		fmt.Printf("  LoggedInAt:   %s\n", creds.LoggedInAt)
		fmt.Println()
	}
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key, value := args[0], args[1]

	switch key {
	case "environment":
		if _, err := envs.Resolve(value); err != nil {
			return fmt.Errorf("%v: %w", err, ErrUsage)
		}
		cfg := config.Get()
		cfg.Environment = value
		if err := config.SetConfig(cfg); err != nil {
			return err
		}
		fmt.Printf("%s environment = %s\n", color.GreenString("✓"), value)
		return nil
	default:
		return fmt.Errorf("unknown config key %q (supported: environment): %w", key, ErrUsage)
	}
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	switch key {
	case "environment":
		cfg := config.Get()
		env := cfg.Environment
		if env == "" {
			env = "prod" // default
		}
		fmt.Println(env)
		return nil
	default:
		return fmt.Errorf("unknown config key %q (supported: environment): %w", key, ErrUsage)
	}
}
