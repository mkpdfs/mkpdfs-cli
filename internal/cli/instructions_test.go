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

func mustContain(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestInstructionsHumanPrintsGuideAndMenu(t *testing.T) {
	out, err := execInstructions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mustContain(t, out,
		"mkp auth login", "templates push", "pdf generate",
		"--format", "--auth", "--environments", "--plans", "--agent",
	)
}

func TestInstructionsTopicFlags(t *testing.T) {
	cases := []struct {
		flag  string
		wants []string
	}{
		{"--format", []string{"@page", "size: A4", "ifEq", "formatDate", "6.5 MiB"}},
		{"--auth", []string{"auth whoami", "--api-key", "no `auth status`"}},
		{"--environments", []string{"dev vs prod", "--env dev", "PROD", ".mkpdfs.json"}},
		{"--plans", []string{"1,000 credits", "welcome credits", "enterprise", "INSUFFICIENT_CREDITS"}},
	}
	for _, c := range cases {
		out, err := execInstructions(c.flag)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.flag, err)
		}
		mustContain(t, out, c.wants...)
	}
}

func TestInstructionsCombinesTopics(t *testing.T) {
	out, err := execInstructions("--format", "--plans")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Both sections present, joined by a horizontal rule.
	mustContain(t, out, "Template format", "Plans, credits & limits", "\n---\n")
}

func TestInstructionsAgentPrintsFullWalkthrough(t *testing.T) {
	out, err := execInstructions("--agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Markers spanning every composed section of the agent doc.
	mustContain(t, out,
		"AI coding agent",                            // intro framing
		"dev vs prod", "--env dev",                   // environments
		"auth whoami", "--api-key",                   // auth
		"ifEq", "gt", "formatDate", "formatCurrency", // helpers
		"@page", "size: A4", "6.5 MiB",               // format
		"templates push", "pdf generate",             // example workflow
		"1,000 credits", "enterprise",                // plans
	)
	// Must explicitly steer agents away from the non-existent `auth status`.
	if !strings.Contains(out, "no `auth status`") {
		t.Errorf("agent doc should note there is no `auth status` command")
	}
}

// TestEmbeddedExampleValidates guards the worked example against drifting into
// invalid Handlebars — it must pass the same advisory validator that
// `mkp templates push` runs.
func TestEmbeddedExampleValidates(t *testing.T) {
	const fence = "```hbs"
	i := strings.Index(instrExample, fence)
	if i < 0 {
		t.Fatal("no ```hbs block in example doc")
	}
	rest := instrExample[i+len(fence):]
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
