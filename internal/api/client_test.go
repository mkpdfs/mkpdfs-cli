package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseAPIError(t *testing.T) {
	// structured body wins over status text
	err := parseAPIError(429, []byte(`{"message":"Rate limit exceeded. Please try again later."}`))
	if err.Error() != "Rate limit exceeded. Please try again later." {
		t.Fatalf("got %q", err.Error())
	}
	// error field fallback
	err = parseAPIError(400, []byte(`{"error":"slow_down"}`))
	if err.Error() != "slow_down" {
		t.Fatalf("got %q", err.Error())
	}
	// fallback when body is not JSON
	err = parseAPIError(502, []byte("Bad Gateway"))
	if err.Error() != "API error (status 502)" {
		t.Fatalf("got %q", err.Error())
	}
}

func TestRequestAttachesBearer(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, token: "jwt123"}
	if _, err := c.do("GET", "/x", nil, nil); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer jwt123" {
		t.Fatalf("got %q", gotAuth)
	}
}

func TestRequestAttachesAPIKey(t *testing.T) {
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-api-key")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, apiKey: "tlfy_x"}
	if _, err := c.do("POST", "/x", map[string]string{"a": "b"}, map[string]string{"x-api-key": "tlfy_x"}); err != nil {
		t.Fatal(err)
	}
	if gotKey != "tlfy_x" {
		t.Fatalf("got %q", gotKey)
	}
}

func TestErrorResponseReturnsParsedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`{"message":"Template limit reached"}`))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL}
	resp, err := c.do("GET", "/x", nil, nil)
	if err == nil || err.Error() != "Template limit reached" {
		t.Fatalf("want parsed message, got %v", err)
	}
	if resp == nil || resp.StatusCode != 403 {
		t.Fatalf("response should still carry status: %+v", resp)
	}
}
