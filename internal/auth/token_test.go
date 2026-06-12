package auth

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func makeJWT(t *testing.T, payload map[string]any) string {
	t.Helper()
	b, _ := json.Marshal(payload)
	return "e30." + base64.RawURLEncoding.EncodeToString(b) + ".sig"
}

func TestDecodeJWT(t *testing.T) {
	tok := makeJWT(t, map[string]any{"sub": "u1", "email": "a@b.c", "exp": 1900000000})
	p, err := DecodeJWT(tok)
	if err != nil {
		t.Fatal(err)
	}
	if p.Sub != "u1" || p.Email != "a@b.c" {
		t.Fatalf("bad payload: %+v", p)
	}
}

func TestIsTokenExpired(t *testing.T) {
	fresh := makeJWT(t, map[string]any{"exp": time.Now().Add(time.Hour).Unix()})
	stale := makeJWT(t, map[string]any{"exp": time.Now().Add(30 * time.Second).Unix()}) // inside 60s buffer
	if IsTokenExpired(fresh) {
		t.Fatal("fresh token reported expired")
	}
	if !IsTokenExpired(stale) {
		t.Fatal("token inside buffer should report expired")
	}
	if !IsTokenExpired("") || !IsTokenExpired("garbage") {
		t.Fatal("empty/garbage should be expired")
	}
}
