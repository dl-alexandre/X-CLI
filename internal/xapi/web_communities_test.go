package xapi

import "testing"

func TestCleanCommunityTitle(t *testing.T) {
	got := cleanCommunityTitle("Go Builders Community / X")
	if got != "Go Builders" {
		t.Fatalf("cleanCommunityTitle() = %q, want Go Builders", got)
	}
}
