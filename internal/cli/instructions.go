package cli

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Embedded usage docs, split into self-contained topic sections. Static markdown
// so `mkp instructions [flags]` works offline, needs no auth, and versions in
// lockstep with the CLI build — the commands documented always match this binary.
//
//go:embed instructions_human.md
var instrHuman string

//go:embed instr_agent_intro.md
var instrAgentIntro string

//go:embed instr_environments.md
var instrEnvironments string

//go:embed instr_auth.md
var instrAuth string

//go:embed instr_format.md
var instrFormat string

//go:embed instr_example.md
var instrExample string

//go:embed instr_plans.md
var instrPlans string

// joinSections concatenates topic sections with a markdown horizontal rule
// between them and a trailing newline.
func joinSections(sections ...string) string {
	parts := make([]string, 0, len(sections))
	for _, s := range sections {
		parts = append(parts, strings.TrimRight(s, "\n"))
	}
	return strings.Join(parts, "\n\n---\n\n") + "\n"
}

// newInstructionsCmd builds the command with request-scoped topic flags, so no
// package-level flag state leaks across invocations (mirrors newCreditsCmd).
func newInstructionsCmd() *cobra.Command {
	var agent, format, auth, environments, plans bool

	cmd := &cobra.Command{
		Use:   "instructions",
		Short: "Print mkpdfs usage instructions (topic flags, or --agent for everything)",
		Long: "Print how to author a Handlebars template, push it, and generate a PDF.\n\n" +
			"With no flag it prints a short human guide. Pick a topic with a flag, or\n" +
			"combine several:\n" +
			"  --format        template format: HTML/CSS, @page, variables, helpers\n" +
			"  --auth          authentication: login, whoami, --api-key\n" +
			"  --environments  dev vs prod and how to switch\n" +
			"  --plans         plans, credits and limits\n" +
			"  --agent         the whole thing, framed for an AI coding agent\n\n" +
			"Tell your agent, e.g.: \"create a mkpdfs love-letter template; get the\n" +
			"format from `mkp instructions --agent`\".",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Write to OutOrStdout (no color/ANSI) so agents capturing stdout and
			// `mkp instructions --agent > mkpdfs.md` get clean content.
			out := cmd.OutOrStdout()

			// --agent: the full agent-framed walkthrough (all topics, in order).
			if agent {
				fmt.Fprint(out, joinSections(
					instrAgentIntro, instrEnvironments, instrAuth, instrFormat, instrExample, instrPlans,
				))
				return nil
			}

			// Topic flags: print the selected sections in canonical order.
			var sections []string
			if environments {
				sections = append(sections, instrEnvironments)
			}
			if auth {
				sections = append(sections, instrAuth)
			}
			if format {
				sections = append(sections, instrFormat)
			}
			if plans {
				sections = append(sections, instrPlans)
			}

			if len(sections) == 0 {
				fmt.Fprint(out, instrHuman) // no topic flag → human overview + menu
				return nil
			}
			fmt.Fprint(out, joinSections(sections...))
			return nil
		},
	}

	cmd.Flags().BoolVar(&agent, "agent", false, "full walkthrough for an AI coding agent (all topics)")
	cmd.Flags().BoolVar(&format, "format", false, "template format: HTML/CSS, @page, variables, helpers")
	cmd.Flags().BoolVar(&auth, "auth", false, "authentication: login, whoami, --api-key")
	cmd.Flags().BoolVar(&environments, "environments", false, "dev vs prod and how to switch")
	cmd.Flags().BoolVar(&plans, "plans", false, "plans, credits and limits")
	return cmd
}

func addInstructionsCommands() {
	rootCmd.AddCommand(newInstructionsCmd())
}
