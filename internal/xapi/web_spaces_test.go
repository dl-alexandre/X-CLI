package xapi

import "testing"

func TestNormalizeSpaceID(t *testing.T) {
	tests := map[string]string{
		"1yoKMxyz":                          "1yoKMxyz",
		"/spaces/1yoKMxyz":                  "1yoKMxyz",
		"https://x.com/i/spaces/1yoKMxyz":   "1yoKMxyz",
		"https://x.com/spaces/1yoKMxyz?x=1": "1yoKMxyz",
	}
	for input, want := range tests {
		if got := normalizeSpaceID(input); got != want {
			t.Fatalf("normalizeSpaceID(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCleanSpaceTitle(t *testing.T) {
	if got := cleanSpaceTitle("Weekly Build Chat / X"); got != "Weekly Build Chat" {
		t.Fatalf("cleanSpaceTitle() = %q", got)
	}
}
