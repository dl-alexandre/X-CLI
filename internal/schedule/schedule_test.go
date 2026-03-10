package schedule

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSchedule(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "schedule-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := &ScheduleStore{
		filePath: filepath.Join(tmpDir, "scheduled.json"),
		tweets:   make(map[string]ScheduledTweet),
	}

	future := time.Now().Add(1 * time.Hour)
	tweet, err := store.Schedule("Test tweet", future)
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	if tweet.ID == "" {
		t.Error("Expected non-empty ID")
	}

	if tweet.Text != "Test tweet" {
		t.Errorf("Expected text 'Test tweet', got '%s'", tweet.Text)
	}

	if tweet.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", tweet.Status)
	}

	if !tweet.Scheduled.Equal(future) {
		t.Errorf("Expected scheduled time %v, got %v", future, tweet.Scheduled)
	}
}

func TestScheduleEmptyText(t *testing.T) {
	store := &ScheduleStore{
		filePath: filepath.Join(os.TempDir(), "test-scheduled.json"),
		tweets:   make(map[string]ScheduledTweet),
	}

	future := time.Now().Add(1 * time.Hour)
	_, err := store.Schedule("", future)
	if err != ErrEmptyText {
		t.Errorf("Expected ErrEmptyText, got %v", err)
	}

	_, err = store.Schedule("   ", future)
	if err != ErrEmptyText {
		t.Errorf("Expected ErrEmptyText for whitespace, got %v", err)
	}
}

func TestScheduleTimeInPast(t *testing.T) {
	store := &ScheduleStore{
		filePath: filepath.Join(os.TempDir(), "test-scheduled.json"),
		tweets:   make(map[string]ScheduledTweet),
	}

	past := time.Now().Add(-1 * time.Hour)
	_, err := store.Schedule("Test tweet", past)
	if err != ErrTimeInPast {
		t.Errorf("Expected ErrTimeInPast, got %v", err)
	}
}

func TestList(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "schedule-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := &ScheduleStore{
		filePath: filepath.Join(tmpDir, "scheduled.json"),
		tweets:   make(map[string]ScheduledTweet),
	}

	now := time.Now()
	store.tweets["1"] = ScheduledTweet{ID: "1", Text: "First", Scheduled: now.Add(2 * time.Hour), Status: "pending"}
	store.tweets["2"] = ScheduledTweet{ID: "2", Text: "Second", Scheduled: now.Add(1 * time.Hour), Status: "pending"}
	store.tweets["3"] = ScheduledTweet{ID: "3", Text: "Third", Scheduled: now.Add(3 * time.Hour), Status: "cancelled"}

	list := store.List()
	if len(list) != 3 {
		t.Errorf("Expected 3 tweets, got %d", len(list))
	}

	if list[0].ID != "2" {
		t.Errorf("Expected first tweet to be '2' (earliest), got '%s'", list[0].ID)
	}
}

func TestListPending(t *testing.T) {
	store := &ScheduleStore{
		filePath: filepath.Join(os.TempDir(), "test-scheduled.json"),
		tweets:   make(map[string]ScheduledTweet),
	}

	now := time.Now()
	store.tweets["1"] = ScheduledTweet{ID: "1", Text: "First", Scheduled: now.Add(1 * time.Hour), Status: "pending"}
	store.tweets["2"] = ScheduledTweet{ID: "2", Text: "Second", Scheduled: now.Add(2 * time.Hour), Status: "posted"}
	store.tweets["3"] = ScheduledTweet{ID: "3", Text: "Third", Scheduled: now.Add(3 * time.Hour), Status: "cancelled"}
	store.tweets["4"] = ScheduledTweet{ID: "4", Text: "Fourth", Scheduled: now.Add(4 * time.Hour), Status: "pending"}

	pending := store.ListPending()
	if len(pending) != 2 {
		t.Errorf("Expected 2 pending tweets, got %d", len(pending))
	}

	for _, tweet := range pending {
		if tweet.Status != "pending" {
			t.Errorf("Expected pending status, got '%s'", tweet.Status)
		}
	}
}

func TestCancel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "schedule-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := &ScheduleStore{
		filePath: filepath.Join(tmpDir, "scheduled.json"),
		tweets:   make(map[string]ScheduledTweet),
	}

	now := time.Now()
	store.tweets["1"] = ScheduledTweet{ID: "1", Text: "Test", Scheduled: now.Add(1 * time.Hour), Status: "pending"}

	err = store.Cancel("1")
	if err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}

	tweet, _ := store.Get("1")
	if tweet.Status != "cancelled" {
		t.Errorf("Expected status 'cancelled', got '%s'", tweet.Status)
	}
}

func TestCancelNotFound(t *testing.T) {
	store := &ScheduleStore{
		filePath: filepath.Join(os.TempDir(), "test-scheduled.json"),
		tweets:   make(map[string]ScheduledTweet),
	}

	err := store.Cancel("nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestCancelAlreadyPosted(t *testing.T) {
	store := &ScheduleStore{
		filePath: filepath.Join(os.TempDir(), "test-scheduled.json"),
		tweets:   make(map[string]ScheduledTweet),
	}

	now := time.Now()
	store.tweets["1"] = ScheduledTweet{ID: "1", Text: "Test", Scheduled: now.Add(1 * time.Hour), Status: "posted"}

	err := store.Cancel("1")
	if err != ErrAlreadyPosted {
		t.Errorf("Expected ErrAlreadyPosted, got %v", err)
	}
}

func TestCancelAlreadyCancelled(t *testing.T) {
	store := &ScheduleStore{
		filePath: filepath.Join(os.TempDir(), "test-scheduled.json"),
		tweets:   make(map[string]ScheduledTweet),
	}

	now := time.Now()
	store.tweets["1"] = ScheduledTweet{ID: "1", Text: "Test", Scheduled: now.Add(1 * time.Hour), Status: "cancelled"}

	err := store.Cancel("1")
	if err != ErrAlreadyCancelled {
		t.Errorf("Expected ErrAlreadyCancelled, got %v", err)
	}
}

func TestMarkPosted(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "schedule-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := &ScheduleStore{
		filePath: filepath.Join(tmpDir, "scheduled.json"),
		tweets:   make(map[string]ScheduledTweet),
	}

	now := time.Now()
	store.tweets["1"] = ScheduledTweet{ID: "1", Text: "Test", Scheduled: now.Add(1 * time.Hour), Status: "pending"}

	err = store.MarkPosted("1")
	if err != nil {
		t.Fatalf("MarkPosted failed: %v", err)
	}

	tweet, _ := store.Get("1")
	if tweet.Status != "posted" {
		t.Errorf("Expected status 'posted', got '%s'", tweet.Status)
	}
}

func TestGetDue(t *testing.T) {
	store := &ScheduleStore{
		filePath: filepath.Join(os.TempDir(), "test-scheduled.json"),
		tweets:   make(map[string]ScheduledTweet),
	}

	now := time.Now()
	store.tweets["1"] = ScheduledTweet{ID: "1", Text: "Past", Scheduled: now.Add(-1 * time.Hour), Status: "pending"}
	store.tweets["2"] = ScheduledTweet{ID: "2", Text: "Future", Scheduled: now.Add(1 * time.Hour), Status: "pending"}
	store.tweets["3"] = ScheduledTweet{ID: "3", Text: "Past Cancelled", Scheduled: now.Add(-1 * time.Hour), Status: "cancelled"}
	store.tweets["4"] = ScheduledTweet{ID: "4", Text: "Past Posted", Scheduled: now.Add(-1 * time.Hour), Status: "posted"}

	due := store.GetDue()
	if len(due) != 1 {
		t.Errorf("Expected 1 due tweet, got %d", len(due))
	}

	if len(due) > 0 && due[0].ID != "1" {
		t.Errorf("Expected due tweet '1', got '%s'", due[0].ID)
	}
}

func TestDelete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "schedule-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := &ScheduleStore{
		filePath: filepath.Join(tmpDir, "scheduled.json"),
		tweets:   make(map[string]ScheduledTweet),
	}

	now := time.Now()
	store.tweets["1"] = ScheduledTweet{ID: "1", Text: "Test", Scheduled: now.Add(1 * time.Hour), Status: "pending"}

	err = store.Delete("1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = store.Get("1")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got %v", err)
	}
}

func TestDeleteNotFound(t *testing.T) {
	store := &ScheduleStore{
		filePath: filepath.Join(os.TempDir(), "test-scheduled.json"),
		tweets:   make(map[string]ScheduledTweet),
	}

	err := store.Delete("nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestParseTimeAbsolute(t *testing.T) {
	tests := []struct {
		input    string
		hasError bool
	}{
		{"2024-01-15 09:00", false},
		{"2024-01-15 09:00:00", false},
		{"01/15/2024 09:00", false},
		{"2024-01-15", false},
		{"invalid", true},
	}

	for _, test := range tests {
		_, err := ParseTime(test.input)
		if test.hasError && err == nil {
			t.Errorf("Expected error for input '%s', got nil", test.input)
		}
		if !test.hasError && err != nil {
			t.Errorf("Expected no error for input '%s', got %v", test.input, err)
		}
	}
}

func TestParseTimeRelative(t *testing.T) {
	now := time.Now()

	tests := []struct {
		input      string
		checkAfter func(time.Time) bool
	}{
		{"in 1 hour", func(t time.Time) bool { return t.After(now.Add(59*time.Minute)) && t.Before(now.Add(61*time.Minute)) }},
		{"in 2 hours", func(t time.Time) bool { return t.After(now.Add(119*time.Minute)) && t.Before(now.Add(121*time.Minute)) }},
		{"in 30 minutes", func(t time.Time) bool { return t.After(now.Add(29*time.Minute)) && t.Before(now.Add(31*time.Minute)) }},
		{"in 1 day", func(t time.Time) bool { return t.After(now.Add(23*time.Hour)) && t.Before(now.Add(25*time.Hour)) }},
		{"in 2 days", func(t time.Time) bool { return t.After(now.Add(47*time.Hour)) && t.Before(now.Add(49*time.Hour)) }},
		{"in 1 week", func(t time.Time) bool { return t.After(now.Add(6*24*time.Hour)) && t.Before(now.Add(8*24*time.Hour)) }},
		{"1 hour from now", func(t time.Time) bool { return t.After(now.Add(59*time.Minute)) && t.Before(now.Add(61*time.Minute)) }},
		{"2 days from now", func(t time.Time) bool { return t.After(now.Add(47*time.Hour)) && t.Before(now.Add(49*time.Hour)) }},
	}

	for _, test := range tests {
		result, err := ParseTime(test.input)
		if err != nil {
			t.Errorf("ParseTime('%s') failed: %v", test.input, err)
			continue
		}

		if !test.checkAfter(result) {
			t.Errorf("ParseTime('%s') = %v, time check failed", test.input, result)
		}
	}
}

func TestParseTimeNatural(t *testing.T) {
	now := time.Now()

	tests := []struct {
		input      string
		checkAfter func(time.Time) bool
	}{
		{"now", func(t time.Time) bool { return t.Sub(now) < time.Second }},
		{"tomorrow", func(t time.Time) bool {
			expected := now.AddDate(0, 0, 1)
			return t.Day() == expected.Day()
		}},
		{"tomorrow at 9am", func(t time.Time) bool {
			return t.Hour() == 9 && t.Minute() == 0
		}},
		{"tomorrow at 2pm", func(t time.Time) bool {
			return t.Hour() == 14 && t.Minute() == 0
		}},
		{"tomorrow at 9:30am", func(t time.Time) bool {
			return t.Hour() == 9 && t.Minute() == 30
		}},
		{"today at 3pm", func(t time.Time) bool {
			return t.Hour() == 15 && t.Minute() == 0
		}},
		{"at 5pm", func(t time.Time) bool {
			return t.Hour() == 17 && t.Minute() == 0
		}},
		{"next monday", func(t time.Time) bool {
			return t.Weekday() == time.Monday && t.After(now)
		}},
		{"next friday", func(t time.Time) bool {
			return t.Weekday() == time.Friday && t.After(now)
		}},
	}

	for _, test := range tests {
		result, err := ParseTime(test.input)
		if err != nil {
			t.Errorf("ParseTime('%s') failed: %v", test.input, err)
			continue
		}

		if !test.checkAfter(result) {
			t.Errorf("ParseTime('%s') = %v, time check failed", test.input, result)
		}
	}
}

func TestParseTimeInvalid(t *testing.T) {
	invalidInputs := []string{
		"",
		"invalid",
		"at some point",
		"in",
		"tomorrow at",
	}

	for _, input := range invalidInputs {
		_, err := ParseTime(input)
		if err == nil {
			t.Errorf("Expected error for input '%s', got nil", input)
		}
	}
}

func TestFormatTimeRemaining(t *testing.T) {
	now := time.Now()

	tests := []struct {
		scheduled   time.Time
		checkFormat func(string) bool
	}{
		{now.Add(-1 * time.Minute), func(s string) bool { return s == "due now" }},
		{now.Add(30 * time.Second), func(s string) bool { return strings.Contains(s, "second") }},
		{now.Add(90 * time.Second), func(s string) bool { return strings.Contains(s, "minute") }},
		{now.Add(5 * time.Minute), func(s string) bool { return strings.Contains(s, "minute") }},
		{now.Add(90 * time.Minute), func(s string) bool { return strings.Contains(s, "hour") }},
		{now.Add(3 * time.Hour), func(s string) bool { return strings.Contains(s, "hour") }},
		{now.Add(30 * time.Hour), func(s string) bool { return strings.Contains(s, "day") || strings.Contains(s, "hour") }},
		{now.Add(72 * time.Hour), func(s string) bool { return strings.Contains(s, "day") }},
	}

	for _, test := range tests {
		result := FormatTimeRemaining(test.scheduled)
		if !test.checkFormat(result) {
			t.Errorf("FormatTimeRemaining(%v) = '%s', format check failed", test.scheduled, result)
		}
	}
}

func TestPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "schedule-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "scheduled.json")

	store1 := &ScheduleStore{
		filePath: filePath,
		tweets:   make(map[string]ScheduledTweet),
	}

	future := time.Now().Add(1 * time.Hour)
	tweet, err := store1.Schedule("Persistent tweet", future)
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	store2 := &ScheduleStore{
		filePath: filePath,
		tweets:   make(map[string]ScheduledTweet),
	}

	if err := store2.load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	loaded, err := store2.Get(tweet.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if loaded.Text != "Persistent tweet" {
		t.Errorf("Expected text 'Persistent tweet', got '%s'", loaded.Text)
	}

	if loaded.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", loaded.Status)
	}
}

func TestCount(t *testing.T) {
	store := &ScheduleStore{
		filePath: filepath.Join(os.TempDir(), "test-scheduled.json"),
		tweets:   make(map[string]ScheduledTweet),
	}

	now := time.Now()
	store.tweets["1"] = ScheduledTweet{ID: "1", Text: "First", Scheduled: now.Add(1 * time.Hour), Status: "pending"}
	store.tweets["2"] = ScheduledTweet{ID: "2", Text: "Second", Scheduled: now.Add(2 * time.Hour), Status: "posted"}
	store.tweets["3"] = ScheduledTweet{ID: "3", Text: "Third", Scheduled: now.Add(3 * time.Hour), Status: "cancelled"}

	if store.Count() != 3 {
		t.Errorf("Expected count 3, got %d", store.Count())
	}

	if store.PendingCount() != 1 {
		t.Errorf("Expected pending count 1, got %d", store.PendingCount())
	}
}
