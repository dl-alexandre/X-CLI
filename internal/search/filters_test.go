package search

import (
	"testing"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/model"
)

func TestFilter_HasFilters(t *testing.T) {
	tests := []struct {
		name   string
		filter Filter
		want   bool
	}{
		{"empty filter", Filter{}, false},
		{"from user", Filter{FromUser: "test"}, true},
		{"min likes", Filter{MinLikes: 10}, true},
		{"min retweets", Filter{MinRetweets: 5}, true},
		{"has media", Filter{HasMedia: true}, true},
		{"is reply", Filter{IsReply: true}, true},
		{"is retweet", Filter{IsRetweet: true}, true},
		{"date start", Filter{DateStart: time.Now()}, true},
		{"date end", Filter{DateEnd: time.Now()}, true},
		{"zero values", Filter{MinLikes: 0, MinRetweets: 0}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.HasFilters(); got != tt.want {
				t.Errorf("Filter.HasFilters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilter_Validate(t *testing.T) {
	tests := []struct {
		name    string
		filter  Filter
		wantErr bool
	}{
		{"valid empty", Filter{}, false},
		{"valid with user", Filter{FromUser: "test"}, false},
		{"valid min likes", Filter{MinLikes: 10}, false},
		{"negative min likes", Filter{MinLikes: -1}, true},
		{"negative min retweets", Filter{MinRetweets: -1}, true},
		{"valid date range", Filter{DateStart: time.Now().Add(-24 * time.Hour), DateEnd: time.Now()}, false},
		{"invalid date range", Filter{DateStart: time.Now(), DateEnd: time.Now().Add(-24 * time.Hour)}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Filter.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFilter_Matches(t *testing.T) {
	now := time.Now()
	testTweet := model.Tweet{
		ID:        "123",
		Text:      "Test tweet",
		CreatedAt: now.Format(time.RubyDate),
		Author:    model.Author{ScreenName: "testuser"},
		Metrics:   model.TweetMetrics{Likes: 100, Retweets: 50, Replies: 10},
		URLs:      []string{"https://example.com/image.jpg"},
	}

	tests := []struct {
		name   string
		filter Filter
		tweet  model.Tweet
		want   bool
	}{
		{
			name:   "no filter matches all",
			filter: Filter{},
			tweet:  testTweet,
			want:   true,
		},
		{
			name:   "from user match",
			filter: Filter{FromUser: "testuser"},
			tweet:  testTweet,
			want:   true,
		},
		{
			name:   "from user no match",
			filter: Filter{FromUser: "otheruser"},
			tweet:  testTweet,
			want:   false,
		},
		{
			name:   "min likes match",
			filter: Filter{MinLikes: 50},
			tweet:  testTweet,
			want:   true,
		},
		{
			name:   "min likes no match",
			filter: Filter{MinLikes: 200},
			tweet:  testTweet,
			want:   false,
		},
		{
			name:   "min retweets match",
			filter: Filter{MinRetweets: 25},
			tweet:  testTweet,
			want:   true,
		},
		{
			name:   "min retweets no match",
			filter: Filter{MinRetweets: 100},
			tweet:  testTweet,
			want:   false,
		},
		{
			name:   "has media match",
			filter: Filter{HasMedia: true},
			tweet:  testTweet,
			want:   true,
		},
		{
			name:   "has media no match",
			filter: Filter{HasMedia: true},
			tweet:  model.Tweet{ID: "124", Author: model.Author{ScreenName: "test"}, Metrics: model.TweetMetrics{}},
			want:   false,
		},
		{
			name:   "is reply match",
			filter: Filter{IsReply: true},
			tweet:  testTweet,
			want:   true,
		},
		{
			name:   "is reply no match",
			filter: Filter{IsReply: true},
			tweet:  model.Tweet{ID: "124", Author: model.Author{ScreenName: "test"}, Metrics: model.TweetMetrics{Replies: 0}},
			want:   false,
		},
		{
			name:   "is retweet match",
			filter: Filter{IsRetweet: true},
			tweet:  model.Tweet{ID: "124", Author: model.Author{ScreenName: "test"}, IsRetweet: true},
			want:   true,
		},
		{
			name:   "is retweet no match",
			filter: Filter{IsRetweet: true},
			tweet:  testTweet,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.Matches(tt.tweet); got != tt.want {
				t.Errorf("Filter.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilter_Matches_DateRange(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)

	tweetTime := yesterday.Format(time.RubyDate)
	testTweet := model.Tweet{
		ID:        "123",
		Text:      "Test tweet",
		CreatedAt: tweetTime,
		Author:    model.Author{ScreenName: "testuser"},
		Metrics:   model.TweetMetrics{},
	}

	tests := []struct {
		name   string
		filter Filter
		want   bool
	}{
		{
			name:   "date range contains tweet",
			filter: Filter{DateStart: twoDaysAgo, DateEnd: now},
			want:   true,
		},
		{
			name:   "date range before tweet",
			filter: Filter{DateStart: twoDaysAgo.Add(-48 * time.Hour), DateEnd: twoDaysAgo},
			want:   false,
		},
		{
			name:   "date range after tweet",
			filter: Filter{DateStart: now, DateEnd: now.Add(24 * time.Hour)},
			want:   false,
		},
		{
			name:   "only start date",
			filter: Filter{DateStart: twoDaysAgo},
			want:   true,
		},
		{
			name:   "only end date",
			filter: Filter{DateEnd: now},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.Matches(testTweet); got != tt.want {
				t.Errorf("Filter.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApply(t *testing.T) {
	tweets := []model.Tweet{
		{ID: "1", Author: model.Author{ScreenName: "alice"}, Metrics: model.TweetMetrics{Likes: 100}},
		{ID: "2", Author: model.Author{ScreenName: "bob"}, Metrics: model.TweetMetrics{Likes: 50}},
		{ID: "3", Author: model.Author{ScreenName: "alice"}, Metrics: model.TweetMetrics{Likes: 200}},
	}

	tests := []struct {
		name      string
		filter    *Filter
		wantCount int
		wantIDs   []string
	}{
		{
			name:      "nil filter returns all",
			filter:    nil,
			wantCount: 3,
		},
		{
			name:      "empty filter returns all",
			filter:    &Filter{},
			wantCount: 3,
		},
		{
			name:      "filter by user",
			filter:    &Filter{FromUser: "alice"},
			wantCount: 2,
			wantIDs:   []string{"1", "3"},
		},
		{
			name:      "filter by min likes",
			filter:    &Filter{MinLikes: 75},
			wantCount: 2,
			wantIDs:   []string{"1", "3"},
		},
		{
			name:      "combined filters",
			filter:    &Filter{FromUser: "alice", MinLikes: 150},
			wantCount: 1,
			wantIDs:   []string{"3"},
		},
		{
			name:      "no matches",
			filter:    &Filter{MinLikes: 1000},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Apply(tweets, tt.filter)
			if len(result) != tt.wantCount {
				t.Errorf("Apply() returned %d tweets, want %d", len(result), tt.wantCount)
			}
			if tt.wantIDs != nil {
				for i, wantID := range tt.wantIDs {
					if i >= len(result) || result[i].ID != wantID {
						t.Errorf("Apply()[%d].ID = %v, want %v", i, result[i].ID, wantID)
					}
				}
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{"empty string", "", time.Time{}, false},
		{"YYYY-MM-DD", "2024-01-15", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), false},
		{"YYYY/MM/DD", "2024/01/15", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), false},
		{"Jan 2, 2006", "Jan 15, 2024", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), false},
		{"January 2, 2006", "January 15, 2024", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), false},
		{"RFC3339", "2024-01-15T10:30:00Z", time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC), false},
		{"invalid format", "not-a-date", time.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.IsZero() && !tt.want.IsZero() {
				if !got.Equal(tt.want) {
					t.Errorf("ParseDate() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestParseDateRange(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantStart bool
		wantEnd   bool
		wantErr   bool
	}{
		{"empty string", "", false, false, false},
		{"valid range", "2024-01-01,2024-01-31", true, true, false},
		{"start only", "2024-01-01,", true, false, false},
		{"end only", ",2024-01-31", false, true, false},
		{"invalid format", "2024-01-01", false, false, true},
		{"invalid start date", "invalid,2024-01-31", false, false, true},
		{"invalid end date", "2024-01-01,invalid", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := ParseDateRange(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDateRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tt.wantStart && start.IsZero() {
					t.Error("ParseDateRange() start is zero, expected non-zero")
				}
				if !tt.wantStart && !start.IsZero() {
					t.Error("ParseDateRange() start is non-zero, expected zero")
				}
				if tt.wantEnd && end.IsZero() {
					t.Error("ParseDateRange() end is zero, expected non-zero")
				}
				if !tt.wantEnd && !end.IsZero() {
					t.Error("ParseDateRange() end is non-zero, expected zero")
				}
			}
		})
	}
}

func TestParseTweetDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty string", "", true},
		{"RubyDate format", "Mon Jan 15 10:30:00 +0000 2024", false},
		{"RFC3339", "2024-01-15T10:30:00Z", false},
		{"invalid format", "not-a-date", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTweetDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTweetDate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeUsername(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"with @ prefix", "@TestUser", "testuser"},
		{"without @ prefix", "TestUser", "testuser"},
		{"with spaces", "  @TestUser  ", "testuser"},
		{"already lowercase", "testuser", "testuser"},
		{"mixed case", "TeStUsEr", "testuser"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeUsername(tt.input); got != tt.want {
				t.Errorf("NormalizeUsername() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildSearchQuery(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		filter   *Filter
		contains []string
	}{
		{
			name:     "nil filter",
			base:     "golang",
			filter:   nil,
			contains: []string{"golang"},
		},
		{
			name:     "empty filter",
			base:     "golang",
			filter:   &Filter{},
			contains: []string{"golang"},
		},
		{
			name:     "from user",
			base:     "golang",
			filter:   &Filter{FromUser: "golang"},
			contains: []string{"golang", "from:golang"},
		},
		{
			name:     "min likes",
			base:     "golang",
			filter:   &Filter{MinLikes: 100},
			contains: []string{"golang", "min_faves:100"},
		},
		{
			name:     "min retweets",
			base:     "golang",
			filter:   &Filter{MinRetweets: 50},
			contains: []string{"golang", "min_retweets:50"},
		},
		{
			name:     "has media",
			base:     "golang",
			filter:   &Filter{HasMedia: true},
			contains: []string{"golang", "filter:media"},
		},
		{
			name:     "is reply",
			base:     "golang",
			filter:   &Filter{IsReply: true},
			contains: []string{"golang", "filter:replies"},
		},
		{
			name:     "is retweet",
			base:     "golang",
			filter:   &Filter{IsRetweet: true},
			contains: []string{"golang", "filter:retweets"},
		},
		{
			name:     "date range",
			base:     "golang",
			filter:   &Filter{DateStart: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), DateEnd: time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)},
			contains: []string{"golang", "since:2024-01-01", "until:2024-01-31"},
		},
		{
			name: "combined filters",
			base: "golang",
			filter: &Filter{
				FromUser: "golang",
				MinLikes: 100,
				HasMedia: true,
			},
			contains: []string{"golang", "from:golang", "min_faves:100", "filter:media"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildSearchQuery(tt.base, tt.filter)
			for _, want := range tt.contains {
				if !containsString(result, want) {
					t.Errorf("BuildSearchQuery() = %v, should contain %v", result, want)
				}
			}
		})
	}
}

func TestHasMediaURL(t *testing.T) {
	tests := []struct {
		name  string
		tweet model.Tweet
		want  bool
	}{
		{
			name:  "no URLs",
			tweet: model.Tweet{ID: "1"},
			want:  false,
		},
		{
			name:  "image URL",
			tweet: model.Tweet{ID: "1", URLs: []string{"https://example.com/image.jpg"}},
			want:  true,
		},
		{
			name:  "video URL",
			tweet: model.Tweet{ID: "1", URLs: []string{"https://example.com/video.mp4"}},
			want:  true,
		},
		{
			name:  "non-media URL",
			tweet: model.Tweet{ID: "1", URLs: []string{"https://example.com/page"}},
			want:  false,
		},
		{
			name:  "mixed URLs",
			tweet: model.Tweet{ID: "1", URLs: []string{"https://example.com/page", "https://example.com/image.png"}},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasMediaURL(tt.tweet); got != tt.want {
				t.Errorf("HasMediaURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
