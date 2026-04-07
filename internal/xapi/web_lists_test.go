package xapi

import "testing"

func TestCleanListTitle(t *testing.T) {
	got := cleanListTitle("Builders / X")
	if got != "Builders" {
		t.Fatalf("cleanListTitle() = %q, want Builders", got)
	}
}

func TestExtractListCount(t *testing.T) {
	body := "3,210 Members 98 Followers"
	if got := extractListCount(body, "member"); got != 3210 {
		t.Fatalf("member count = %d, want 3210", got)
	}
	if got := extractListCount(body, "follower"); got != 98 {
		t.Fatalf("follower count = %d, want 98", got)
	}
}

func TestParseHumanCount(t *testing.T) {
	if got := parseHumanCount("12,345"); got != 12345 {
		t.Fatalf("parseHumanCount() = %d, want 12345", got)
	}
	if got := parseHumanCount("invalid"); got != 0 {
		t.Fatalf("parseHumanCount() = %d, want 0", got)
	}
}
