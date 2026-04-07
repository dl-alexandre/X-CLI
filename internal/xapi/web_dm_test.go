package xapi

import (
	"testing"

	"github.com/dl-alexandre/X-CLI/internal/model"
)

func TestNormalizeConversationID(t *testing.T) {
	tests := map[string]string{
		"12345":                                "12345",
		"/messages/12345":                      "12345",
		"https://x.com/messages/12345":         "12345",
		"https://x.com/i/messages/12345":       "12345",
		"https://x.com/messages/12345?foo=bar": "12345",
	}
	for input, want := range tests {
		if got := normalizeConversationID(input); got != want {
			t.Fatalf("normalizeConversationID(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCleanDMText(t *testing.T) {
	if got := cleanDMText(" hello   there \n friend "); got != "hello there friend" {
		t.Fatalf("cleanDMText() = %q", got)
	}
}

func TestMessageSortKey(t *testing.T) {
	withTime := model.DirectMessage{Text: "a", CreatedAt: "2026-01-01T00:00:00Z"}
	withoutTime := model.DirectMessage{Text: "b"}
	if !(messageSortKey(withTime) < messageSortKey(withoutTime)) {
		t.Fatal("expected timestamped message to sort before missing timestamp")
	}
}
