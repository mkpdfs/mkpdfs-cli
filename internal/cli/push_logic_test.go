package cli

import (
	"errors"
	"testing"

	"github.com/sim4gh/mkpdfs-cli/internal/localmap"
)

func entryMap(env, user string, e *localmap.Entry) *localmap.Map {
	m := &localmap.Map{Environment: env, UserID: user, Templates: map[string]localmap.Entry{}}
	if e != nil {
		m.Templates["invoice.hbs"] = *e
	}
	return m
}

func TestDecideCreateWhenUnknown(t *testing.T) {
	d, err := decidePush(pushInput{File: "invoice.hbs", Map: entryMap("dev", "u1", nil),
		ActiveEnv: "dev", UserID: "u1"})
	if err != nil || d.Action != pushCreate {
		t.Fatalf("want create, got %+v err=%v", d, err)
	}
}

func TestDecideUpdateWhenKnownAndClean(t *testing.T) {
	e := &localmap.Entry{TemplateID: "t1", RemoteUpdatedAt: "2026-06-11T00:00:00Z"}
	d, err := decidePush(pushInput{File: "invoice.hbs", Map: entryMap("dev", "u1", e),
		ActiveEnv: "dev", UserID: "u1", RemoteUpdatedAt: "2026-06-11T00:00:00Z"})
	if err != nil || d.Action != pushUpdate || d.TemplateID != "t1" {
		t.Fatalf("want update t1, got %+v err=%v", d, err)
	}
}

func TestEnvMismatchAborts(t *testing.T) {
	if _, err := decidePush(pushInput{File: "invoice.hbs", Map: entryMap("prod", "u1", nil),
		ActiveEnv: "dev", UserID: "u1"}); err == nil {
		t.Fatal("env mismatch must abort")
	}
}

func TestAccountMismatchAbortsWithoutForce(t *testing.T) {
	e := &localmap.Entry{TemplateID: "t1"}
	if _, err := decidePush(pushInput{File: "invoice.hbs", Map: entryMap("dev", "other", e),
		ActiveEnv: "dev", UserID: "u1"}); err == nil {
		t.Fatal("account mismatch must abort")
	}
	d, err := decidePush(pushInput{File: "invoice.hbs", Map: entryMap("dev", "other", e),
		ActiveEnv: "dev", UserID: "u1", Force: true})
	if err != nil || d.Action != pushUpdate {
		t.Fatalf("force should override, got %+v err=%v", d, err)
	}
}

func TestConflictAbortsWithoutForce(t *testing.T) {
	e := &localmap.Entry{TemplateID: "t1", RemoteUpdatedAt: "2026-06-11T00:00:00Z"}
	in := pushInput{File: "invoice.hbs", Map: entryMap("dev", "u1", e),
		ActiveEnv: "dev", UserID: "u1", RemoteUpdatedAt: "2026-06-12T09:00:00Z"}
	if _, err := decidePush(in); err == nil {
		t.Fatal("remote drift must abort")
	}
	in.Force = true
	if d, err := decidePush(in); err != nil || d.Action != pushUpdate {
		t.Fatalf("force should override, got %+v err=%v", d, err)
	}
}

func TestExplicitFlagsWin(t *testing.T) {
	if d, _ := decidePush(pushInput{File: "x.hbs", Map: entryMap("dev", "u1", nil),
		ActiveEnv: "dev", UserID: "u1", ForceID: "t9"}); d.Action != pushUpdate || d.TemplateID != "t9" {
		t.Fatal("--id must force update to that id")
	}
	e := &localmap.Entry{TemplateID: "t1"}
	if d, _ := decidePush(pushInput{File: "invoice.hbs", Map: entryMap("dev", "u1", e),
		ActiveEnv: "dev", UserID: "u1", ForceNew: true}); d.Action != pushCreate {
		t.Fatal("--new must force create")
	}
}

func TestUsageErrorsCarrySentinel(t *testing.T) {
	_, err := decidePush(pushInput{File: "invoice.hbs", Map: entryMap("prod", "u1", nil),
		ActiveEnv: "dev", UserID: "u1"})
	if !errors.Is(err, ErrUsage) {
		t.Fatal("guard errors must wrap ErrUsage (exit code 2)")
	}
}

func TestParsePushResultRejectsMissingTemplateID(t *testing.T) {
	if _, err := parsePushResult([]byte(`{"name":"demo","updatedAt":"2026-06-12T00:00:00Z"}`)); err == nil {
		t.Fatal("missing templateId must be rejected")
	}
	got, err := parsePushResult([]byte(`{"templateId":"t1","name":"demo","fileSize":58,"updatedAt":"2026-06-12T00:00:00Z"}`))
	if err != nil || got.TemplateID != "t1" || got.Name != "demo" || got.FileSize != 58 {
		t.Fatalf("valid body must parse, got %+v err=%v", got, err)
	}
	if _, err := parsePushResult([]byte(`not json`)); err == nil {
		t.Fatal("invalid JSON must be rejected")
	}
}
