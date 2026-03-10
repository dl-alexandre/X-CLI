package output

import (
	"testing"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/model"
)

func TestNewTemplateEngine(t *testing.T) {
	tests := []struct {
		name      string
		template  string
		wantError bool
	}{
		{
			name:      "simple text template",
			template:  "{{.User.Name}}",
			wantError: false,
		},
		{
			name:      "template with function",
			template:  "{{.User.Name | upper}}",
			wantError: false,
		},
		{
			name:      "template with multiple fields",
			template:  "{{.User.Name}} (@{{.User.ScreenName}}): {{.User.Bio}}",
			wantError: false,
		},
		{
			name:      "template with conditionals",
			template:  "{{if .User.Verified}}✓{{end}}{{.User.Name}}",
			wantError: false,
		},
		{
			name:      "template with range",
			template:  "{{range .Tweets}}{{.Text}}{{end}}",
			wantError: false,
		},
		{
			name:      "invalid template syntax",
			template:  "{{.User.Name",
			wantError: true,
		},
		{
			name:      "invalid field reference",
			template:  "{{.User.NonExistentField}}",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTemplateEngine(tt.template)
			if (err != nil) != tt.wantError {
				t.Errorf("NewTemplateEngine() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestNewTemplateEnginePreset(t *testing.T) {
	tests := []struct {
		name      string
		template  string
		wantError bool
	}{
		{
			name:      "valid preset tweet.simple",
			template:  "@tweet.simple",
			wantError: false,
		},
		{
			name:      "valid preset user.bio",
			template:  "@user.bio",
			wantError: false,
		},
		{
			name:      "invalid preset",
			template:  "@nonexistent.preset",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTemplateEngine(tt.template)
			if (err != nil) != tt.wantError {
				t.Errorf("NewTemplateEngine() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestTemplateEngine_ExecuteUser(t *testing.T) {
	user := model.UserProfile{
		ID:             "123",
		Name:           "John Doe",
		ScreenName:     "johndoe",
		Bio:            "Software developer",
		Location:       "San Francisco",
		FollowersCount: 10000,
		FollowingCount: 500,
		TweetsCount:    1234,
		Verified:       true,
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "simple name",
			template: "{{.User.Name}}",
			want:     "John Doe",
		},
		{
			name:     "name and screen name",
			template: "{{.User.Name}} (@{{.User.ScreenName}})",
			want:     "John Doe (@johndoe)",
		},
		{
			name:     "with upper function",
			template: "{{.User.Name | upper}}",
			want:     "JOHN DOE",
		},
		{
			name:     "with lower function",
			template: "{{.User.ScreenName | lower}}",
			want:     "johndoe",
		},
		{
			name:     "with k formatting",
			template: "{{.User.FollowersCount | k}}",
			want:     "10.0K",
		},
		{
			name:     "with number formatting",
			template: "{{.User.FollowersCount | num}}",
			want:     "10000",
		},
		{
			name:     "with default value",
			template: "{{.User.Location | default \"Unknown\"}}",
			want:     "San Francisco",
		},
		{
			name:     "with coalesce",
			template: "{{coalesce .User.Name .User.Bio}}",
			want:     "John Doe",
		},
		{
			name:     "conditional verified",
			template: "{{if .User.Verified}}✓ Verified{{else}}Not verified{{end}}",
			want:     "✓ Verified",
		},
		{
			name:     "multiple fields",
			template: "Name: {{.User.Name}}, Followers: {{.User.FollowersCount}}",
			want:     "Name: John Doe, Followers: 10000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewTemplateEngine(tt.template)
			if err != nil {
				t.Fatalf("NewTemplateEngine() error = %v", err)
			}

			got, err := engine.ExecuteUser(user)
			if err != nil {
				t.Fatalf("ExecuteUser() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("ExecuteUser() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTemplateEngine_ExecuteTweet(t *testing.T) {
	tweet := model.Tweet{
		ID:        "123456789",
		Text:      "Hello #golang @twitter https://example.com",
		CreatedAt: "Mon Jan 15 10:30:00 +0000 2024",
		Author: model.Author{
			ID:         "100",
			Name:       "Jane Doe",
			ScreenName: "janedoe",
			Verified:   true,
		},
		Metrics: model.TweetMetrics{
			Likes:    1500,
			Retweets: 250,
			Replies:  50,
			Views:    10000,
		},
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "simple text",
			template: "{{.Tweet.Text}}",
			want:     "Hello #golang @twitter https://example.com",
		},
		{
			name:     "author and text",
			template: "@{{.Tweet.Author.ScreenName}}: {{.Tweet.Text}}",
			want:     "@janedoe: Hello #golang @twitter https://example.com",
		},
		{
			name:     "metrics with k formatting",
			template: "{{.Tweet.Metrics.Likes | k}} likes",
			want:     "1.5K likes",
		},
		{
			name:     "truncate text",
			template: "{{truncate .Tweet.Text 20}}",
			want:     "Hello #golang @tw...",
		},
		{
			name:     "extract hashtags",
			template: "{{range .Tweet.Text | hashtag}}{{.}} {{end}}",
			want:     "#golang ",
		},
		{
			name:     "extract mentions",
			template: "{{range .Tweet.Text | mention}}{{.}} {{end}}",
			want:     "@twitter ",
		},
		{
			name:     "extract urls",
			template: "{{range .Tweet.Text | url}}{{.}} {{end}}",
			want:     "https://example.com ",
		},
		{
			name:     "full tweet format",
			template: "[{{.Tweet.ID}}] @{{.Tweet.Author.ScreenName}}: {{.Tweet.Text}} ({{.Tweet.Metrics.Likes}} likes)",
			want:     "[123456789] @janedoe: Hello #golang @twitter https://example.com (1500 likes)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewTemplateEngine(tt.template)
			if err != nil {
				t.Fatalf("NewTemplateEngine() error = %v", err)
			}

			got, err := engine.ExecuteTweet(tweet)
			if err != nil {
				t.Fatalf("ExecuteTweet() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("ExecuteTweet() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTemplateEngine_ExecuteTweets(t *testing.T) {
	tweets := []model.Tweet{
		{
			ID:   "1",
			Text: "First tweet",
			Author: model.Author{
				ScreenName: "user1",
			},
			Metrics: model.TweetMetrics{Likes: 100},
		},
		{
			ID:   "2",
			Text: "Second tweet",
			Author: model.Author{
				ScreenName: "user2",
			},
			Metrics: model.TweetMetrics{Likes: 200},
		},
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "list tweets",
			template: "{{range .Tweets}}- {{.Text}}\n{{end}}",
			want:     "- First tweet\n- Second tweet\n",
		},
		{
			name:     "numbered list",
			template: "{{range $i, $t := .Tweets}}{{add $i 1}}. {{$t.Text}}\n{{end}}",
			want:     "1. First tweet\n2. Second tweet\n",
		},
		{
			name:     "count tweets",
			template: "Total: {{count .Tweets}} tweets",
			want:     "Total: 2 tweets",
		},
		{
			name:     "first tweet only",
			template: "{{(index .Tweets 0).Text}}",
			want:     "First tweet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewTemplateEngine(tt.template)
			if err != nil {
				t.Fatalf("NewTemplateEngine() error = %v", err)
			}

			got, err := engine.ExecuteTweets(tweets)
			if err != nil {
				t.Fatalf("ExecuteTweets() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("ExecuteTweets() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuiltInPresets(t *testing.T) {
	user := model.UserProfile{
		Name:           "Test User",
		ScreenName:     "testuser",
		Bio:            "Test bio",
		FollowersCount: 1500,
		FollowingCount: 300,
	}

	tweet := model.Tweet{
		ID:   "123",
		Text: "Test tweet content",
		Author: model.Author{
			ScreenName: "testuser",
		},
		Metrics: model.TweetMetrics{
			Likes:    100,
			Retweets: 50,
			Replies:  10,
			Views:    1000,
		},
	}

	tests := []struct {
		name         string
		preset       string
		data         TemplateData
		wantContains string
	}{
		{
			name:         "tweet.simple",
			preset:       "@tweet.simple",
			data:         TemplateData{Tweet: tweet},
			wantContains: "Test tweet content",
		},
		{
			name:         "tweet.full",
			preset:       "@tweet.full",
			data:         TemplateData{Tweet: tweet},
			wantContains: "100 likes",
		},
		{
			name:         "user.simple",
			preset:       "@user.simple",
			data:         TemplateData{User: user},
			wantContains: "Test User",
		},
		{
			name:         "user.stats",
			preset:       "@user.stats",
			data:         TemplateData{User: user},
			wantContains: "1.5K followers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewTemplateEngine(tt.preset)
			if err != nil {
				t.Fatalf("NewTemplateEngine() error = %v", err)
			}

			got, err := engine.Execute(tt.data)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if !contains(got, tt.wantContains) {
				t.Errorf("Execute() = %q, should contain %q", got, tt.wantContains)
			}
		})
	}
}

func TestFormatK(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{n: 500, want: "500"},
		{n: 999, want: "999"},
		{n: 1000, want: "1.0K"},
		{n: 1500, want: "1.5K"},
		{n: 10000, want: "10.0K"},
		{n: 100000, want: "100.0K"},
		{n: 1000000, want: "1.0M"},
		{n: 2500000, want: "2.5M"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatK(tt.n)
			if got != tt.want {
				t.Errorf("formatK(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		date time.Time
		want string
	}{
		{"just now", now.Add(-30 * time.Second), "just now"},
		{"1 minute", now.Add(-1 * time.Minute), "1m"},
		{"5 minutes", now.Add(-5 * time.Minute), "5m"},
		{"1 hour", now.Add(-1 * time.Hour), "1h"},
		{"3 hours", now.Add(-3 * time.Hour), "3h"},
		{"1 day", now.Add(-24 * time.Hour), "1d"},
		{"3 days", now.Add(-72 * time.Hour), "3d"},
		{"1 week", now.Add(-7 * 24 * time.Hour), "1w"},
		{"2 weeks", now.Add(-14 * 24 * time.Hour), "2w"},
		{"1 month", now.Add(-30 * 24 * time.Hour), "1mo"},
		{"6 months", now.Add(-180 * 24 * time.Hour), "6mo"},
		{"1 year", now.Add(-365 * 24 * time.Hour), "1y"},
		{"2 years", now.Add(-730 * 24 * time.Hour), "2y"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRelativeTime(tt.date)
			if got != tt.want {
				t.Errorf("formatRelativeTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractFunctions(t *testing.T) {
	text := "Hello #golang #programming @user1 @user2 https://example.com http://test.org"

	t.Run("hashtags", func(t *testing.T) {
		hashtags := extractHashtags(text)
		if len(hashtags) != 2 {
			t.Errorf("expected 2 hashtags, got %d", len(hashtags))
		}
	})

	t.Run("mentions", func(t *testing.T) {
		mentions := extractMentions(text)
		if len(mentions) != 2 {
			t.Errorf("expected 2 mentions, got %d", len(mentions))
		}
	})

	t.Run("urls", func(t *testing.T) {
		urls := extractURLs(text)
		if len(urls) != 2 {
			t.Errorf("expected 2 urls, got %d", len(urls))
		}
	})
}

func TestValidateTemplate(t *testing.T) {
	tests := []struct {
		name      string
		template  string
		wantError bool
	}{
		{"valid", "{{.User.Name}}", false},
		{"invalid syntax", "{{.User.Name", true},
		{"valid with function", "{{.User.Name | upper}}", false},
		{"valid with range", "{{range .Tweets}}{{.}}{{end}}", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemplate(tt.template)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateTemplate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestListPresets(t *testing.T) {
	presets := ListPresets()
	if len(presets) == 0 {
		t.Error("ListPresets() returned empty list")
	}

	for _, preset := range presets {
		if _, ok := GetPreset(preset); !ok {
			t.Errorf("GetPreset(%q) returned false for listed preset", preset)
		}
	}
}

func TestIsFileTemplate(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"template.txt", true},
		{"template.tmpl", true},
		{"template.gotmpl", true},
		{"template.template", true},
		{"{{.User.Name}}", false},
		{"{{.Text}}", false},
		{"path/to/file.txt", true},
		{"user.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isFileTemplate(tt.input)
			if got != tt.want {
				t.Errorf("isFileTemplate(%q) = %v, want %v", tt.input, got, tt.want)
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
