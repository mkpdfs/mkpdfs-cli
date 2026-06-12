package cli

import (
	"errors"
	"fmt"
	"os"

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
		fmt.Fprintln(os.Stderr, "Error:", err)
		if errors.Is(err, ErrUsage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = Version
	rootCmd.PersistentFlags().StringVar(&flagEnv, "env", "", "override environment (dev|prod)")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "machine-readable JSON output")
	rootCmd.PersistentFlags().BoolVar(&flagYes, "yes", false, "assume yes for all prompts")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "verbose output")

	addAuthCommands()
	addTemplatesCommands()
	addPdfCommands()
	addTokensCommands()
	addUsageCommand()
	addConfigCommands()
}
