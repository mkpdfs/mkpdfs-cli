//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// binaryPath holds the absolute path to the compiled CLI binary.
// Resolved once at init time so os.Chdir in tests does not affect it.
var binaryPath string

func init() {
	// The test source lives at test/integration/; the binary is at the repo root.
	// __file__ is not available in Go, but we can resolve via os.Executable or
	// walk up from the working directory set by `go test`.
	abs, err := filepath.Abs("../../mkp-cli")
	if err != nil {
		panic("cannot resolve binary path: " + err.Error())
	}
	binaryPath = abs
}

func run(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command(binaryPath, append([]string{"--env", "dev"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mkp %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}

func TestWhoami(t *testing.T) {
	if !strings.Contains(run(t, "auth", "whoami"), "@") {
		t.Fatal("whoami should print an email")
	}
}

func TestTemplatesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to tempdir: %v", err)
	}
	defer os.Chdir(old) //nolint:errcheck

	if err := os.WriteFile("it.hbs", []byte("<p>{{msg}}</p>"), 0644); err != nil {
		t.Fatalf("write it.hbs: %v", err)
	}

	out := run(t, "templates", "push", "it.hbs", "--yes")
	if !strings.Contains(out, "Created") {
		t.Fatalf("expected 'Created' in push output: %s", out)
	}

	out = run(t, "templates", "push", "it.hbs", "--yes")
	if !strings.Contains(out, "Updated") {
		t.Fatalf("expected 'Updated' in push output: %s", out)
	}

	listOut := run(t, "templates", "list", "--json")
	if !strings.Contains(listOut, "it") {
		t.Fatalf("template missing from list: %s", listOut)
	}

	// Read the templateId recorded in .mkpdfs.json
	data, err := os.ReadFile(".mkpdfs.json")
	if err != nil {
		t.Fatalf("reading .mkpdfs.json: %v", err)
	}
	idx := strings.Index(string(data), `"templateId": "`)
	if idx < 0 {
		t.Fatal("no templateId recorded in .mkpdfs.json")
	}
	rest := string(data)[idx+len(`"templateId": "`):]
	id := rest[:strings.Index(rest, `"`)]
	if id == "" {
		t.Fatal("empty templateId in .mkpdfs.json")
	}

	run(t, "templates", "delete", id, "--force")
}
