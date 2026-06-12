package localmap

import (
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	m := &Map{Environment: "dev", UserID: "u1", Templates: map[string]Entry{
		"invoice.hbs": {TemplateID: "t1", Name: "Invoice", RemoteUpdatedAt: "2026-06-11T00:00:00Z"},
	}}
	if err := Save(dir, m); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Templates["invoice.hbs"].TemplateID != "t1" {
		t.Fatalf("round trip failed: %+v", loaded)
	}
}

func TestLoadMissingReturnsEmpty(t *testing.T) {
	m, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if m.Templates == nil || len(m.Templates) != 0 {
		t.Fatal("expected empty initialized map")
	}
}

func TestEntryKeyIsBasename(t *testing.T) {
	if Key(filepath.Join("some", "dir", "invoice.hbs")) != "invoice.hbs" {
		t.Fatal("key should be the basename")
	}
}
