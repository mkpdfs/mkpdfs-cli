package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var Version = "0.1.0" // set at build time via ldflags

// usage/local-validation errors exit 2; everything else exits 1.
var ErrUsage = errors.New("usage error")

var (
	flagEnv     string
	flagJSON    bool
	flagYes     bool
	flagVerbose bool
)

var rootCmd = &cobra.Command{
	Use:           "mkp",
	Short:         "mkpdfs CLI — Handlebars templates to PDF",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Strip the ErrUsage sentinel string from the user-visible message;
		// the exit code (below) still distinguishes usage errors via errors.Is.
		msg := strings.TrimSuffix(err.Error(), ": "+ErrUsage.Error())
		fmt.Fprintln(os.Stderr, "Error:", msg)
		if errors.Is(err, ErrUsage) || isUsageError(err) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

// isUsageError catches Cobra-internal validation errors that don't flow through
// our SetFlagErrorFunc (required-flag and positional-arg validation happen
// after flag parsing and surface as plain errors). These are usage errors → 2.
func isUsageError(err error) bool {
	m := err.Error()
	return strings.HasPrefix(m, "required flag(s)") ||
		strings.Contains(m, "accepts ") && strings.Contains(m, "arg(s)") ||
		strings.HasPrefix(m, "unknown command") ||
		strings.HasPrefix(m, "unknown flag") ||
		strings.HasPrefix(m, "unknown shorthand flag") ||
		strings.HasPrefix(m, "invalid argument") ||
		strings.HasPrefix(m, "flag needs an argument")
}

// requireSubcommand makes a parent command (one with children but no action of
// its own) enforce the exit-code contract: a bare invocation prints help and
// exits 0, but an UNKNOWN subcommand exits 2 (usage error). Without this Cobra
// silently prints the parent help and returns nil (exit 0) for unknown args.
func requireSubcommand(cmd *cobra.Command) {
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("unknown command %q for %q: %w", args[0], cmd.CommandPath(), ErrUsage)
		}
		_ = cmd.Help()
		return nil
	}
}

func init() {
	rootCmd.Version = Version
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return fmt.Errorf("%v: %w", err, ErrUsage)
	})
	rootCmd.PersistentFlags().StringVar(&flagEnv, "env", "", "override environment (dev|prod)")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "machine-readable JSON output")
	rootCmd.PersistentFlags().BoolVar(&flagYes, "yes", false, "assume yes for all prompts")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("mkp version %s\n", Version)
			return nil
		},
	})

	addAuthCommands()
	addTemplatesCommands()
	addPdfCommands()
	addTokensCommands()
	addCreditsCommands()
	addUsageCommand()
	addConfigCommands()

	// Every parent command (children but no own action) gets the unknown-subcommand
	// guard so unknown args exit 2 instead of silently printing help and exiting 0.
	requireSubcommand(rootCmd)
}
