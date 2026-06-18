package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseAPIError(t *testing.T) {
	// 429 + "limit" → keep backend message, append usage guidance
	err := parseAPIError(429, []byte(`{"message":"Rate limit exceeded. Please try again later."}`))
	if err.Error() != "Rate limit exceeded. Please try again later. — check: mkp usage" {
		t.Fatalf("got %q", err.Error())
	}
	// error field fallback (no mapping applies)
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

func TestParseAPIErrorMapping(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{
			name:   "401 unauthorized → login guidance",
			status: 401,
			body:   `{"message":"Unauthorized"}`,
			want:   "session expired or invalid. Run: mkp auth login",
		},
		{
			name:   "403 invalid token → login guidance",
			status: 403,
			body:   `{"message":"Invalid token"}`,
			want:   "session expired or invalid. Run: mkp auth login",
		},
		{
			name:   "401 Authentication failed → login guidance",
			status: 401,
			body:   `{"message":"Authentication failed"}`,
			want:   "session expired or invalid. Run: mkp auth login",
		},
		{
			name:   "402 → buy-credits guidance appended",
			status: 402,
			body:   `{"message":"You have no PDF credits."}`,
			want:   "You have no PDF credits. — buy credits: mkp credits buy",
		},
		{
			name:   "subscription wording is no longer special-cased → raw message",
			status: 403,
			body:   `{"message":"Your subscription is not active"}`,
			want:   "Your subscription is not active",
		},
		{
			name:   "403 limit → usage guidance appended",
			status: 403,
			body:   `{"message":"Template limit reached"}`,
			want:   "Template limit reached — check: mkp usage",
		},
		{
			name:   "429 limit → usage guidance appended",
			status: 429,
			body:   `{"message":"Monthly page limit exceeded"}`,
			want:   "Monthly page limit exceeded — check: mkp usage",
		},
		{
			name:   "unmapped 400 → raw message",
			status: 400,
			body:   `{"message":"Bad request: missing field"}`,
			want:   "Bad request: missing field",
		},
		{
			name:   "404 token wording but not auth status → raw (no limit/subscription)",
			status: 404,
			body:   `{"message":"Template not found"}`,
			want:   "Template not found",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := parseAPIError(tc.status, []byte(tc.body))
			if err.Error() != tc.want {
				t.Fatalf("status=%d body=%s\n got:  %q\n want: %q", tc.status, tc.body, err.Error(), tc.want)
			}
		})
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

func TestRequestAttachesApiKey(t *testing.T) {
	var gotKey, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-api-key")
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, apiKey: "tlfy_abc"}
	if _, err := c.do("GET", "/x", nil, nil); err != nil {
		t.Fatal(err)
	}
	if gotKey != "tlfy_abc" {
		t.Fatalf("x-api-key = %q, want tlfy_abc", gotKey)
	}
	if gotAuth != "" {
		t.Fatalf("Authorization should be empty in api-key mode, got %q", gotAuth)
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
	if err == nil || err.Error() != "Template limit reached — check: mkp usage" {
		t.Fatalf("want parsed message, got %v", err)
	}
	if resp == nil || resp.StatusCode != 403 {
		t.Fatalf("response should still carry status: %+v", resp)
	}
}
