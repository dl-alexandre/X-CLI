package search

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/model"
)

type Filter struct {
	FromUser    string
	MinLikes    int
	MinRetweets int
	DateStart   time.Time
	DateEnd     time.Time
	HasMedia    bool
	IsReply     bool
	IsRetweet   bool
}

func (f *Filter) HasDateRange() bool {
	return !f.DateStart.IsZero() || !f.DateEnd.IsZero()
}

func (f *Filter) HasFilters() bool {
	return f.FromUser != "" ||
		f.MinLikes > 0 ||
		f.MinRetweets > 0 ||
		f.HasDateRange() ||
		f.HasMedia ||
		f.IsReply ||
		f.IsRetweet
}

func (f *Filter) Validate() error {
	if f.MinLikes < 0 {
		return errors.New("min-likes cannot be negative")
	}
	if f.MinRetweets < 0 {
		return errors.New("min-retweets cannot be negative")
	}
	if !f.DateStart.IsZero() && !f.DateEnd.IsZero() && f.DateStart.After(f.DateEnd) {
		return errors.New("date range start cannot be after end")
	}
	return nil
}

func (f *Filter) Matches(tweet model.Tweet) bool {
	if f.FromUser != "" {
		if !strings.EqualFold(tweet.Author.ScreenName, f.FromUser) {
			return false
		}
	}

	if f.MinLikes > 0 && tweet.Metrics.Likes < f.MinLikes {
		return false
	}

	if f.MinRetweets > 0 && tweet.Metrics.Retweets < f.MinRetweets {
		return false
	}

	if f.HasDateRange() {
		tweetTime, err := ParseTweetDate(tweet.CreatedAt)
		if err != nil {
			return false
		}
		if !f.DateStart.IsZero() && tweetTime.Before(f.DateStart) {
			return false
		}
		if !f.DateEnd.IsZero() && tweetTime.After(f.DateEnd) {
			return false
		}
	}

	if f.HasMedia && len(tweet.URLs) == 0 {
		return false
	}

	if f.IsReply && tweet.Metrics.Replies == 0 {
		return false
	}

	if f.IsRetweet && !tweet.IsRetweet {
		return false
	}

	return true
}

func Apply(tweets []model.Tweet, filter *Filter) []model.Tweet {
	if filter == nil || !filter.HasFilters() {
		return tweets
	}

	result := make([]model.Tweet, 0, len(tweets))
	for _, tweet := range tweets {
		if filter.Matches(tweet) {
			result = append(result, tweet)
		}
	}
	return result
}

func ParseDateRange(dateRange string) (start, end time.Time, err error) {
	if dateRange == "" {
		return time.Time{}, time.Time{}, nil
	}

	parts := strings.Split(dateRange, ",")
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid date range format: %q (expected 'start,end')", dateRange)
	}

	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	if startStr != "" {
		start, err = ParseDate(startStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start date: %w", err)
		}
	}

	if endStr != "" {
		end, err = ParseDate(endStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end date: %w", err)
		}
		end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	return start, end, nil
}

func ParseDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, nil
	}

	layouts := []string{
		"2006-01-02",
		"2006/01/02",
		"Jan 2, 2006",
		"January 2, 2006",
		"02-Jan-2006",
		"02 Jan 2006",
		"2006-01-02T15:04:05",
		time.RFC3339,
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, dateStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %q (supported formats: YYYY-MM-DD, YYYY/MM/DD, Jan 2, 2006, etc.)", dateStr)
}

func ParseTweetDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, errors.New("empty date string")
	}

	t, err := time.Parse(time.RubyDate, dateStr)
	if err == nil {
		return t, nil
	}

	t, err = time.Parse("Mon Jan 2 15:04:05 -0700 2006", dateStr)
	if err == nil {
		return t, nil
	}

	t, err = time.Parse(time.RFC3339, dateStr)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unable to parse post date: %q", dateStr)
}

func NormalizeUsername(username string) string {
	username = strings.TrimSpace(username)
	username = strings.TrimPrefix(username, "@")
	return strings.ToLower(username)
}

func BuildSearchQuery(baseQuery string, filter *Filter) string {
	if filter == nil {
		return baseQuery
	}

	var modifiers []string

	if filter.FromUser != "" {
		modifiers = append(modifiers, fmt.Sprintf("from:%s", filter.FromUser))
	}

	if filter.MinLikes > 0 {
		modifiers = append(modifiers, fmt.Sprintf("min_faves:%d", filter.MinLikes))
	}

	if filter.MinRetweets > 0 {
		modifiers = append(modifiers, fmt.Sprintf("min_retweets:%d", filter.MinRetweets))
	}

	if filter.HasDateRange() {
		if !filter.DateStart.IsZero() {
			modifiers = append(modifiers, fmt.Sprintf("since:%s", filter.DateStart.Format("2006-01-02")))
		}
		if !filter.DateEnd.IsZero() {
			modifiers = append(modifiers, fmt.Sprintf("until:%s", filter.DateEnd.Format("2006-01-02")))
		}
	}

	if filter.HasMedia {
		modifiers = append(modifiers, "filter:media")
	}

	if filter.IsReply {
		modifiers = append(modifiers, "filter:replies")
	}

	if filter.IsRetweet {
		modifiers = append(modifiers, "filter:retweets")
	}

	if len(modifiers) == 0 {
		return baseQuery
	}

	return baseQuery + " " + strings.Join(modifiers, " ")
}

var mediaURLPattern = regexp.MustCompile(`(?i)\.(jpg|jpeg|png|gif|mp4|webm|mov)$`)

func HasMediaURL(tweet model.Tweet) bool {
	for _, url := range tweet.URLs {
		if mediaURLPattern.MatchString(url) {
			return true
		}
	}
	return false
}
