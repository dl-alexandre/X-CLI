package xapi

import "testing"

func TestParseCapturedWebURLGraphQL(t *testing.T) {
	queryID, operation, path := parseCapturedWebURL("https://x.com/i/api/graphql/abc123/CreateTweet?variables=%7B%7D")
	if queryID != "abc123" {
		t.Fatalf("queryID = %q, want abc123", queryID)
	}
	if operation != "CreateTweet" {
		t.Fatalf("operation = %q, want CreateTweet", operation)
	}
	if path != "/i/api/graphql/abc123/CreateTweet" {
		t.Fatalf("path = %q", path)
	}
}

func TestParseCapturedWebURLREST(t *testing.T) {
	queryID, operation, path := parseCapturedWebURL("https://x.com/i/api/1.1/dm/inbox_initial_state.json")
	if queryID != "" {
		t.Fatalf("queryID = %q, want empty", queryID)
	}
	if operation != "" {
		t.Fatalf("operation = %q, want empty", operation)
	}
	if path != "/i/api/1.1/dm/inbox_initial_state.json" {
		t.Fatalf("path = %q", path)
	}
}

func TestMatchesCapturedWebFilter(t *testing.T) {
	op := CapturedWebOperation{
		URL:       "https://x.com/i/api/graphql/abc123/CreateTweet",
		Path:      "/i/api/graphql/abc123/CreateTweet",
		Operation: "CreateTweet",
	}

	if !matchesCapturedWebFilter(op, "tweet") {
		t.Fatal("expected filter to match operation")
	}
	if matchesCapturedWebFilter(op, "messages") {
		t.Fatal("expected filter not to match")
	}
}
