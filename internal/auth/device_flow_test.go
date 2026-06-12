package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestPollForTokenHandlesPendingAndSlowDown(t *testing.T) {
	// Override slowDownStep so the slow_down path doesn't sleep 5s in tests.
	old := slowDownStep
	slowDownStep = 0
	defer func() { slowDownStep = old }()

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		switch n {
		case 1:
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
		case 2:
			w.WriteHeader(429)
			json.NewEncoder(w).Encode(map[string]string{"error": "slow_down"})
		default:
			json.NewEncoder(w).Encode(map[string]string{
				"idToken": "id1", "accessToken": "ac1", "refreshToken": "rf1",
			})
		}
	}))
	defer srv.Close()

	tok, err := PollForToken(srv.URL, "dev123", 0) // 0 interval → fast test
	if err != nil {
		t.Fatal(err)
	}
	if tok.IDToken != "id1" || tok.RefreshToken != "rf1" {
		t.Fatalf("bad tokens: %+v", tok)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("expected 3 polls, got %d", calls)
	}
}

func TestPollForTokenStopsOnDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": "access_denied"})
	}))
	defer srv.Close()
	if _, err := PollForToken(srv.URL, "dev123", 0); err == nil {
		t.Fatal("expected error on access_denied")
	}
}

func TestPollForTokenStopsOnExpired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": "expired_token"})
	}))
	defer srv.Close()
	if _, err := PollForToken(srv.URL, "dev123", 0); err == nil {
		t.Fatal("expected error on expired_token")
	}
}
