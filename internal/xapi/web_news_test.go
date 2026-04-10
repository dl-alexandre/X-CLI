package xapi

import "testing"

func TestCleanDMTextNewsReuse(t *testing.T) {
	if got := cleanDMText("  Breaking   News \n story  "); got != "Breaking News story" {
		t.Fatalf("cleanDMText() = %q", got)
	}
}
