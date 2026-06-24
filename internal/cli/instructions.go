package cli

import (
	_ "embed"
	"fmt"

	"github.com/spf13/cobra"
)

// Embedded usage docs. Static markdown so `mkp instructions [--agent]` works
// offline, needs no auth, and versions in lockstep with the CLI build — the
// commands documented always match this binary.
//
//go:embed instructions_agent.md
var instructionsAgent string

//go:embed instructions_human.md
var instructionsHuman string

// newInstructionsCmd builds the command with a request-scoped --agent flag, so no
// package-level flag state leaks across invocations (mirrors newCreditsCmd).
func newInstructionsCmd() *cobra.Command {
	var agent bool
	cmd := &cobra.Command{
		Use:   "instructions",
		Short: "Print mkpdfs usage instructions (add --agent for an AI coding agent)",
		Long: "Print how to author a Handlebars template, push it, and generate a PDF.\n\n" +
			"Use --agent to emit a dense, copy-pasteable markdown doc addressed to an AI\n" +
			"coding agent (Claude Code, Cursor, …). Tell your agent, e.g.:\n" +
			"  \"create a mkpdfs love-letter template; get the format from `mkp instructions --agent`\".",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Write to OutOrStdout (no color/ANSI) so agents capturing stdout and
			// `mkp instructions --agent > mkpdfs.md` get clean content.
			doc := instructionsHuman
			if agent {
				doc = instructionsAgent
			}
			fmt.Fprint(cmd.OutOrStdout(), doc)
			return nil
		},
	}
	cmd.Flags().BoolVar(&agent, "agent", false, "emit markdown instructions for an AI coding agent")
	return cmd
}

func addInstructionsCommands() {
	rootCmd.AddCommand(newInstructionsCmd())
}
