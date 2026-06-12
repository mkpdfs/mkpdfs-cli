package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	overridePathForTest(filepath.Join(dir, "config.json"))

	cfg := &Config{Environment: "dev"}
	cfg.Creds("dev").IDToken = "tok123"
	cfg.Creds("dev").APIKey = "tlfy_abc"
	if err := SetConfig(cfg); err != nil {
		t.Fatal(err)
	}

	resetForTest()
	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Environment != "dev" || loaded.Creds("dev").IDToken != "tok123" {
		t.Fatalf("round trip failed: %+v", loaded)
	}
}

func TestFilePermissions(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	overridePathForTest(p)
	if err := SetConfig(&Config{Environment: "prod"}); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(p)
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected 0600, got %v", info.Mode().Perm())
	}
}

func TestCredsCreatesEnvLazily(t *testing.T) {
	cfg := &Config{}
	cfg.Creds("prod").APIKey = "x"
	if cfg.Environments["prod"].APIKey != "x" {
		t.Fatal("lazy env creation failed")
	}
}

func TestLoadErrorDoesNotCorruptSingleton(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	overridePathForTest(p)
	if err := os.WriteFile(p, []byte("{not valid json"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("expected error loading invalid JSON, got nil")
	}
	// A second Load must surface the same error again — not a silently-empty config.
	if _, err := Load(); err == nil {
		t.Fatal("expected error on second Load, got nil (singleton was corrupted)")
	}
}

func TestMaskToken(t *testing.T) {
	if got := MaskToken("tlfy_1234567890abcdef"); got != "tlfy...cdef" {
		t.Fatalf("got %q", got)
	}
	if MaskToken("") != "(not set)" {
		t.Fatal("empty should be (not set)")
	}
}
