package util

import "testing"

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
	}
	for _, tc := range tests {
		got := FormatBytes(tc.input)
		if got != tc.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("", 10); got != "" {
		t.Errorf("empty string: got %q", got)
	}
	if got := Truncate("short", 10); got != "short" {
		t.Errorf("no truncation needed: got %q", got)
	}
	got := Truncate("hello world", 8)
	if got != "hello..." {
		t.Errorf("truncation: got %q, want %q", got, "hello...")
	}
	// exactly at boundary — no ellipsis
	if got := Truncate("hello", 5); got != "hello" {
		t.Errorf("exact length: got %q", got)
	}
}
