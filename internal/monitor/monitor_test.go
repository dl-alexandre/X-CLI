package monitor

import (
	"testing"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/model"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"1h", time.Hour, false},
		{"30m", 30 * time.Minute, false},
		{"45s", 45 * time.Second, false},
		{"60", 60 * time.Second, false},
		{"2h", 2 * time.Hour, false},
		{"", 0, true},
		{"invalid", 0, true},
		{"1x", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseDuration(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for input %q, got none", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
				return
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestMentionFilter_Keywords(t *testing.T) {
	filter := &MentionFilter{
		Keywords: []string{"golang", "rust"},
	}

	tests := []struct {
		text     string
		expected bool
	}{
		{"I love golang programming", true},
		{"Rust is great", true},
		{"Python is my favorite", false},
		{"GOLANG in uppercase", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			tweet := model.Tweet{Text: tt.text}
			result := (&Monitor{filter: filter}).matchesFilter(tweet)
			if result != tt.expected {
				t.Errorf("expected %v for text %q, got %v", tt.expected, tt.text, result)
			}
		})
	}
}

func TestMentionFilter_Users(t *testing.T) {
	filter := &MentionFilter{
		Users: []string{"alice", "bob"},
	}

	tests := []struct {
		screenName string
		text       string
		expected   bool
	}{
		{"alice", "hello world", true},
		{"bob", "test tweet", true},
		{"charlie", "hey @alice check this", true},
		{"charlie", "hey @bob look", true},
		{"charlie", "no match here", false},
		{"ALICE", "uppercase screen name", true},
	}

	for _, tt := range tests {
		t.Run(tt.screenName+"_"+tt.text, func(t *testing.T) {
			tweet := model.Tweet{
				Text:   tt.text,
				Author: model.Author{ScreenName: tt.screenName},
			}
			result := (&Monitor{filter: filter}).matchesFilter(tweet)
			if result != tt.expected {
				t.Errorf("expected %v for screenName=%q text=%q, got %v", tt.expected, tt.screenName, tt.text, result)
			}
		})
	}
}

func TestMentionFilter_Empty(t *testing.T) {
	filter := &MentionFilter{}
	m := &Monitor{filter: filter}

	tweet := model.Tweet{Text: "any text"}
	if !m.matchesFilter(tweet) {
		t.Error("empty filter should match all tweets")
	}
}

func TestMentionFilter_Nil(t *testing.T) {
	m := &Monitor{filter: nil}

	tweet := model.Tweet{Text: "any text"}
	if !m.matchesFilter(tweet) {
		t.Error("nil filter should match all tweets")
	}
}

func TestIsNewMention(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		state       MonitorState
		tweetID     string
		tweetTime   string
		cutoff      time.Time
		expectedNew bool
	}{
		{
			name:        "first check - always new",
			state:       MonitorState{LastCheckedAt: time.Time{}},
			tweetID:     "123",
			tweetTime:   "Mon Jan 2 15:04:05 +0000 2024",
			cutoff:      now.Add(-time.Hour),
			expectedNew: true,
		},
		{
			name: "same tweet id - not new",
			state: MonitorState{
				LastCheckedAt: now.Add(-time.Hour),
				LastTweetID:   "123",
			},
			tweetID:     "123",
			tweetTime:   "Mon Jan 2 15:04:05 +0000 2024",
			cutoff:      now.Add(-time.Hour),
			expectedNew: false,
		},
		{
			name: "different tweet - after last checked",
			state: MonitorState{
				LastCheckedAt: now.Add(-30 * time.Minute),
				LastTweetID:   "122",
			},
			tweetID:     "124",
			tweetTime:   now.Add(-15 * time.Minute).Format("Mon Jan 2 15:04:05 -0700 2006"),
			cutoff:      now.Add(-time.Hour),
			expectedNew: true,
		},
		{
			name: "different tweet - before cutoff",
			state: MonitorState{
				LastCheckedAt: now.Add(-30 * time.Minute),
				LastTweetID:   "122",
			},
			tweetID:     "124",
			tweetTime:   now.Add(-2 * time.Hour).Format("Mon Jan 2 15:04:05 -0700 2006"),
			cutoff:      now.Add(-time.Hour),
			expectedNew: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Monitor{state: tt.state}
			tweet := model.Tweet{
				ID:        tt.tweetID,
				CreatedAt: tt.tweetTime,
			}
			result := m.isNewMention(tweet, tt.cutoff)
			if result != tt.expectedNew {
				t.Errorf("expected %v, got %v", tt.expectedNew, result)
			}
		})
	}
}

func TestParseTweetTime(t *testing.T) {
	tests := []struct {
		input    string
		hasError bool
	}{
		{"Mon Jan 2 15:04:05 +0000 2006", false},
		{"2024-01-02T15:04:05Z", false},
		{"2024-01-02T15:04:05+00:00", false},
		{"", true},
		{"invalid time string", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := parseTweetTime(tt.input)
			if tt.hasError && err == nil {
				t.Errorf("expected error for input %q, got none", tt.input)
			}
			if !tt.hasError && err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
			}
		})
	}
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly 10", 10, "exactly 10"},
		{"this is a longer text that needs truncation", 20, "this is a longer ..."},
		{"", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncateText(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatMention(t *testing.T) {
	tests := []struct {
		name     string
		mention  Mention
		contains []string
	}{
		{
			name: "basic mention",
			mention: Mention{
				Tweet: model.Tweet{
					Author: model.Author{ScreenName: "alice"},
					Text:   "Hello world",
				},
				IsNew: false,
			},
			contains: []string{"@alice:", "Hello world"},
		},
		{
			name: "new mention",
			mention: Mention{
				Tweet: model.Tweet{
					Author: model.Author{ScreenName: "bob"},
					Text:   "New tweet",
				},
				IsNew: true,
			},
			contains: []string{"@bob:", "New tweet", "[NEW]"},
		},
		{
			name: "long text truncation",
			mention: Mention{
				Tweet: model.Tweet{
					Author: model.Author{ScreenName: "charlie"},
					Text:   "This is a very long tweet that should be truncated because it exceeds the maximum length for display and more",
				},
				IsNew: false,
			},
			contains: []string{"@charlie:", "..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMention(tt.mention)
			for _, s := range tt.contains {
				if !contains(result, s) {
					t.Errorf("expected result to contain %q, got %q", s, result)
				}
			}
		})
	}
}

func TestFormatMentions(t *testing.T) {
	result := MonitorResult{
		Mentions: []Mention{
			{
				Tweet: model.Tweet{
					Author: model.Author{ScreenName: "alice"},
					Text:   "First mention",
				},
				IsNew: true,
			},
			{
				Tweet: model.Tweet{
					Author: model.Author{ScreenName: "bob"},
					Text:   "Second mention",
				},
				IsNew: false,
			},
		},
		TotalCount:  2,
		NewCount:    1,
		LastChecked: time.Now(),
	}

	output := FormatMentions(result)

	if !contains(output, "2 total") {
		t.Error("expected output to contain total count")
	}
	if !contains(output, "1 new") {
		t.Error("expected output to contain new count")
	}
	if !contains(output, "@alice") {
		t.Error("expected output to contain alice")
	}
	if !contains(output, "@bob") {
		t.Error("expected output to contain bob")
	}
}

func TestMonitorState_Persistence(t *testing.T) {
	state := MonitorState{
		LastCheckedAt: time.Now(),
		LastTweetID:   "123456789",
	}

	if state.LastTweetID != "123456789" {
		t.Errorf("expected LastTweetID to be 123456789, got %s", state.LastTweetID)
	}

	if state.LastCheckedAt.IsZero() {
		t.Error("expected LastCheckedAt to be set")
	}
}

func TestMonitorResult_Counts(t *testing.T) {
	result := MonitorResult{
		Mentions: []Mention{
			{IsNew: true},
			{IsNew: true},
			{IsNew: false},
			{IsNew: false},
			{IsNew: false},
		},
		TotalCount: 5,
		NewCount:   2,
	}

	if result.TotalCount != 5 {
		t.Errorf("expected TotalCount 5, got %d", result.TotalCount)
	}
	if result.NewCount != 2 {
		t.Errorf("expected NewCount 2, got %d", result.NewCount)
	}
}

func TestMentionFilter_Combined(t *testing.T) {
	filter := &MentionFilter{
		Keywords: []string{"urgent"},
		Users:    []string{"admin"},
	}

	tests := []struct {
		screenName string
		text       string
		expected   bool
	}{
		{"user1", "urgent: server down", true},
		{"admin", "regular update", true},
		{"user2", "nothing special", false},
		{"ADMIN", "URGENT issue", true},
	}

	for _, tt := range tests {
		t.Run(tt.screenName+"_"+tt.text, func(t *testing.T) {
			tweet := model.Tweet{
				Text:   tt.text,
				Author: model.Author{ScreenName: tt.screenName},
			}
			m := &Monitor{filter: filter}
			result := m.matchesFilter(tweet)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
