package schedule

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ScheduledTweet struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	Scheduled time.Time `json:"scheduled"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

type ScheduleStore struct {
	filePath string
	tweets   map[string]ScheduledTweet
}

var (
	ErrNotFound         = errors.New("scheduled tweet not found")
	ErrInvalidTime      = errors.New("invalid scheduled time")
	ErrTimeInPast       = errors.New("scheduled time is in the past")
	ErrEmptyText        = errors.New("tweet text cannot be empty")
	ErrAlreadyPosted    = errors.New("tweet already posted")
	ErrAlreadyCancelled = errors.New("tweet already cancelled")
)

func NewScheduleStore() (*ScheduleStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	schedulePath := filepath.Join(home, ".config", "x-cli", "scheduled.json")
	store := &ScheduleStore{
		filePath: schedulePath,
		tweets:   make(map[string]ScheduledTweet),
	}

	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return store, nil
}

func (s *ScheduleStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &s.tweets)
}

func (s *ScheduleStore) save() error {
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.tweets, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

func (s *ScheduleStore) Schedule(text string, scheduled time.Time) (*ScheduledTweet, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyText
	}

	if scheduled.Before(time.Now()) {
		return nil, ErrTimeInPast
	}

	id := generateID()
	now := time.Now()

	tweet := ScheduledTweet{
		ID:        id,
		Text:      text,
		Scheduled: scheduled,
		CreatedAt: now,
		Status:    "pending",
	}

	s.tweets[id] = tweet

	if err := s.save(); err != nil {
		return nil, err
	}

	return &tweet, nil
}

func (s *ScheduleStore) Get(id string) (*ScheduledTweet, error) {
	tweet, exists := s.tweets[id]
	if !exists {
		return nil, ErrNotFound
	}

	return &tweet, nil
}

func (s *ScheduleStore) Cancel(id string) error {
	tweet, exists := s.tweets[id]
	if !exists {
		return ErrNotFound
	}

	if tweet.Status == "posted" {
		return ErrAlreadyPosted
	}

	if tweet.Status == "cancelled" {
		return ErrAlreadyCancelled
	}

	tweet.Status = "cancelled"
	s.tweets[id] = tweet

	return s.save()
}

func (s *ScheduleStore) MarkPosted(id string) error {
	tweet, exists := s.tweets[id]
	if !exists {
		return ErrNotFound
	}

	if tweet.Status == "cancelled" {
		return ErrAlreadyCancelled
	}

	tweet.Status = "posted"
	s.tweets[id] = tweet

	return s.save()
}

func (s *ScheduleStore) List() []ScheduledTweet {
	tweets := make([]ScheduledTweet, 0, len(s.tweets))
	for _, tweet := range s.tweets {
		tweets = append(tweets, tweet)
	}

	sort.Slice(tweets, func(i, j int) bool {
		return tweets[i].Scheduled.Before(tweets[j].Scheduled)
	})

	return tweets
}

func (s *ScheduleStore) ListPending() []ScheduledTweet {
	tweets := make([]ScheduledTweet, 0)
	for _, tweet := range s.tweets {
		if tweet.Status == "pending" {
			tweets = append(tweets, tweet)
		}
	}

	sort.Slice(tweets, func(i, j int) bool {
		return tweets[i].Scheduled.Before(tweets[j].Scheduled)
	})

	return tweets
}

func (s *ScheduleStore) GetDue() []ScheduledTweet {
	now := time.Now()
	tweets := make([]ScheduledTweet, 0)

	for _, tweet := range s.tweets {
		if tweet.Status == "pending" && !tweet.Scheduled.After(now) {
			tweets = append(tweets, tweet)
		}
	}

	sort.Slice(tweets, func(i, j int) bool {
		return tweets[i].Scheduled.Before(tweets[j].Scheduled)
	})

	return tweets
}

func (s *ScheduleStore) Delete(id string) error {
	if _, exists := s.tweets[id]; !exists {
		return ErrNotFound
	}

	delete(s.tweets, id)

	return s.save()
}

func (s *ScheduleStore) Count() int {
	return len(s.tweets)
}

func (s *ScheduleStore) PendingCount() int {
	count := 0
	for _, tweet := range s.tweets {
		if tweet.Status == "pending" {
			count++
		}
	}
	return count
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func ParseTime(input string) (time.Time, error) {
	input = strings.TrimSpace(strings.ToLower(input))

	if t, err := parseAbsoluteTime(input); err == nil {
		return t, nil
	}

	if t, err := parseRelativeTime(input); err == nil {
		return t, nil
	}

	if t, err := parseNaturalTime(input); err == nil {
		return t, nil
	}

	return time.Time{}, ErrInvalidTime
}

func parseAbsoluteTime(input string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02T15:04:05",
		"01/02/2006 15:04",
		"01/02/2006 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, input); err == nil {
			return t, nil
		}
	}

	return time.Time{}, ErrInvalidTime
}

func parseRelativeTime(input string) (time.Time, error) {
	now := time.Now()

	patterns := []struct {
		regex   *regexp.Regexp
		handler func(matches []string) (time.Time, error)
	}{
		{
			regex: regexp.MustCompile(`^in\s+(\d+)\s+(second|seconds|minute|minutes|hour|hours|day|days|week|weeks)$`),
			handler: func(matches []string) (time.Time, error) {
				amount, _ := strconv.Atoi(matches[1])
				unit := matches[2]

				switch {
				case strings.HasPrefix(unit, "second"):
					return now.Add(time.Duration(amount) * time.Second), nil
				case strings.HasPrefix(unit, "minute"):
					return now.Add(time.Duration(amount) * time.Minute), nil
				case strings.HasPrefix(unit, "hour"):
					return now.Add(time.Duration(amount) * time.Hour), nil
				case strings.HasPrefix(unit, "day"):
					return now.AddDate(0, 0, amount), nil
				case strings.HasPrefix(unit, "week"):
					return now.AddDate(0, 0, amount*7), nil
				}
				return time.Time{}, ErrInvalidTime
			},
		},
		{
			regex: regexp.MustCompile(`^(\d+)\s+(second|seconds|minute|minutes|hour|hours|day|days|week|weeks)\s+from\s+now$`),
			handler: func(matches []string) (time.Time, error) {
				amount, _ := strconv.Atoi(matches[1])
				unit := matches[2]

				switch {
				case strings.HasPrefix(unit, "second"):
					return now.Add(time.Duration(amount) * time.Second), nil
				case strings.HasPrefix(unit, "minute"):
					return now.Add(time.Duration(amount) * time.Minute), nil
				case strings.HasPrefix(unit, "hour"):
					return now.Add(time.Duration(amount) * time.Hour), nil
				case strings.HasPrefix(unit, "day"):
					return now.AddDate(0, 0, amount), nil
				case strings.HasPrefix(unit, "week"):
					return now.AddDate(0, 0, amount*7), nil
				}
				return time.Time{}, ErrInvalidTime
			},
		},
	}

	for _, pattern := range patterns {
		if matches := pattern.regex.FindStringSubmatch(input); matches != nil {
			return pattern.handler(matches)
		}
	}

	return time.Time{}, ErrInvalidTime
}

func parseNaturalTime(input string) (time.Time, error) {
	now := time.Now()

	if input == "now" {
		return now, nil
	}

	if input == "tomorrow" {
		return now.AddDate(0, 0, 1), nil
	}

	tomorrowPattern := regexp.MustCompile(`^tomorrow\s+at\s+(\d{1,2})(?::(\d{2}))?\s*(am|pm)?$`)
	if matches := tomorrowPattern.FindStringSubmatch(input); matches != nil {
		hour, _ := strconv.Atoi(matches[1])
		minute := 0
		if matches[2] != "" {
			minute, _ = strconv.Atoi(matches[2])
		}
		ampm := matches[3]

		if ampm == "pm" && hour < 12 {
			hour += 12
		} else if ampm == "am" && hour == 12 {
			hour = 0
		} else if ampm == "" && hour < 8 {
			hour += 12
		}

		tomorrow := now.AddDate(0, 0, 1)
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), hour, minute, 0, 0, now.Location()), nil
	}

	todayPattern := regexp.MustCompile(`^today\s+at\s+(\d{1,2})(?::(\d{2}))?\s*(am|pm)?$`)
	if matches := todayPattern.FindStringSubmatch(input); matches != nil {
		hour, _ := strconv.Atoi(matches[1])
		minute := 0
		if matches[2] != "" {
			minute, _ = strconv.Atoi(matches[2])
		}
		ampm := matches[3]

		if ampm == "pm" && hour < 12 {
			hour += 12
		} else if ampm == "am" && hour == 12 {
			hour = 0
		} else if ampm == "" && hour < 8 {
			hour += 12
		}

		return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location()), nil
	}

	atPattern := regexp.MustCompile(`^at\s+(\d{1,2})(?::(\d{2}))?\s*(am|pm)?$`)
	if matches := atPattern.FindStringSubmatch(input); matches != nil {
		hour, _ := strconv.Atoi(matches[1])
		minute := 0
		if matches[2] != "" {
			minute, _ = strconv.Atoi(matches[2])
		}
		ampm := matches[3]

		if ampm == "pm" && hour < 12 {
			hour += 12
		} else if ampm == "am" && hour == 12 {
			hour = 0
		} else if ampm == "" && hour < 8 {
			hour += 12
		}

		result := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
		if result.Before(now) {
			result = result.AddDate(0, 0, 1)
		}
		return result, nil
	}

	nextDayPattern := regexp.MustCompile(`^next\s+(monday|tuesday|wednesday|thursday|friday|saturday|sunday)$`)
	if matches := nextDayPattern.FindStringSubmatch(input); matches != nil {
		targetDay := dayNameToWeekday(matches[1])
		daysUntil := int((targetDay - now.Weekday() + 7) % 7)
		if daysUntil == 0 {
			daysUntil = 7
		}
		return now.AddDate(0, 0, daysUntil), nil
	}

	return time.Time{}, ErrInvalidTime
}

func dayNameToWeekday(name string) time.Weekday {
	switch name {
	case "sunday":
		return time.Sunday
	case "monday":
		return time.Monday
	case "tuesday":
		return time.Tuesday
	case "wednesday":
		return time.Wednesday
	case "thursday":
		return time.Thursday
	case "friday":
		return time.Friday
	case "saturday":
		return time.Saturday
	}
	return time.Sunday
}

func FormatTimeRemaining(scheduled time.Time) string {
	now := time.Now()
	if scheduled.Before(now) {
		return "due now"
	}

	duration := scheduled.Sub(now)

	if duration < time.Minute {
		return fmt.Sprintf("in %d seconds", int(duration.Seconds()))
	}

	if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "in 1 minute"
		}
		return fmt.Sprintf("in %d minutes", minutes)
	}

	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "in 1 hour"
		}
		return fmt.Sprintf("in %d hours", hours)
	}

	days := int(duration.Hours() / 24)
	if days == 1 {
		return "in 1 day"
	}
	return fmt.Sprintf("in %d days", days)
}
