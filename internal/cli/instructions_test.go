package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sim4gh/mkpdfs-cli/internal/hbs"
)

func execInstructions(args ...string) (string, error) {
	cmd := newInstructionsCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestInstructionsHumanPrintsGuide(t *testing.T) {
	out, err := execInstructions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"mkp auth login", "templates push", "pdf generate", "--agent"} {
		if !strings.Contains(out, want) {
			t.Errorf("human guide missing %q", want)
		}
	}
}

func TestInstructionsAgentPrintsFullWalkthrough(t *testing.T) {
	out, err := execInstructions("--agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Key markers across the documented workflow + the exact helper names.
	markers := []string{
		"ifEq", "gt", "formatDate", "formatCurrency", // helpers
		"@page", "size: A4", // page format
		"--env dev",      // dev-first guidance
		"auth whoami",    // the real status command
		"templates push", // push
		"pdf generate",   // generate
		"6.5 MiB",        // size limit
		"--api-key",      // headless path
	}
	for _, want := range markers {
		if !strings.Contains(out, want) {
			t.Errorf("agent doc missing %q", want)
		}
	}
	// The doc must explicitly steer agents away from the non-existent `auth
	// status` command toward `auth whoami`.
	if !strings.Contains(out, "no `auth status`") {
		t.Errorf("agent doc should explicitly note there is no `auth status` command")
	}
}

// TestEmbeddedExampleValidates guards the worked example in the agent doc against
// drifting into invalid Handlebars — it must pass the same advisory validator that
// `mkp templates push` runs.
func TestEmbeddedExampleValidates(t *testing.T) {
	const fence = "```hbs"
	i := strings.Index(instructionsAgent, fence)
	if i < 0 {
		t.Fatal("no ```hbs block in agent doc")
	}
	rest := instructionsAgent[i+len(fence):]
	j := strings.Index(rest, "```")
	if j < 0 {
		t.Fatal("unterminated ```hbs block")
	}
	if _, err := hbs.Validate(rest[:j]); err != nil {
		t.Fatalf("embedded carta.hbs fails hbs.Validate: %v", err)
	}
}

func TestInstructionsRejectsExtraArgs(t *testing.T) {
	if _, err := execInstructions("bogus"); err == nil {
		t.Fatal("want usage error for positional arg, got nil")
	}
}
