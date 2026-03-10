package analytics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/model"
)

func createTestCollector(t *testing.T) *AnalyticsCollector {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "analytics.db")

	return &AnalyticsCollector{
		dbPath:    dbPath,
		tweets:    make(map[string][]TweetRecord),
		snapshots: make(map[string][]UserSnapshot),
		metrics:   make(map[string][]EngagementMetrics),
	}
}

func createTestTweet(id, screenName, text string, likes, retweets, replies int) model.Tweet {
	return model.Tweet{
		ID:        id,
		Text:      text,
		CreatedAt: time.Now().Add(-1 * time.Hour).Format(time.RubyDate),
		Author: model.Author{
			ID:         "user123",
			Name:       "Test User",
			ScreenName: screenName,
		},
		Metrics: model.TweetMetrics{
			Likes:     likes,
			Retweets:  retweets,
			Replies:   replies,
			Quotes:    0,
			Views:     1000,
			Bookmarks: 0,
		},
	}
}

func createTestUserProfile(screenName string, followers int) model.UserProfile {
	return model.UserProfile{
		ID:             "user123",
		Name:           "Test User",
		ScreenName:     screenName,
		FollowersCount: followers,
		FollowingCount: 100,
		TweetsCount:    500,
	}
}

func TestRecordTweet(t *testing.T) {
	collector := createTestCollector(t)

	tweet := createTestTweet("123", "testuser", "Hello world #golang", 10, 5, 2)

	if err := collector.RecordTweet(tweet); err != nil {
		t.Fatalf("RecordTweet failed: %v", err)
	}

	collector.mu.RLock()
	records := collector.tweets["testuser"]
	collector.mu.RUnlock()

	if len(records) != 1 {
		t.Errorf("Expected 1 tweet record, got %d", len(records))
	}

	if records[0].Tweet.ID != "123" {
		t.Errorf("Expected tweet ID 123, got %s", records[0].Tweet.ID)
	}

	collector.mu.RLock()
	metrics := collector.metrics["123"]
	collector.mu.RUnlock()

	if len(metrics) != 1 {
		t.Errorf("Expected 1 metrics record, got %d", len(metrics))
	}

	if metrics[0].Likes != 10 {
		t.Errorf("Expected 10 likes, got %d", metrics[0].Likes)
	}
}

func TestRecordUserSnapshot(t *testing.T) {
	collector := createTestCollector(t)

	user := createTestUserProfile("testuser", 1000)

	if err := collector.RecordUserSnapshot(user); err != nil {
		t.Fatalf("RecordUserSnapshot failed: %v", err)
	}

	collector.mu.RLock()
	snapshots := collector.snapshots["testuser"]
	collector.mu.RUnlock()

	if len(snapshots) != 1 {
		t.Errorf("Expected 1 snapshot, got %d", len(snapshots))
	}

	if snapshots[0].FollowersCount != 1000 {
		t.Errorf("Expected 1000 followers, got %d", snapshots[0].FollowersCount)
	}
}

func TestFollowerGrowthTracking(t *testing.T) {
	collector := createTestCollector(t)

	user1 := createTestUserProfile("testuser", 1000)
	user2 := createTestUserProfile("testuser", 1050)
	user3 := createTestUserProfile("testuser", 1120)

	if err := collector.RecordUserSnapshot(user1); err != nil {
		t.Fatalf("RecordUserSnapshot 1 failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if err := collector.RecordUserSnapshot(user2); err != nil {
		t.Fatalf("RecordUserSnapshot 2 failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if err := collector.RecordUserSnapshot(user3); err != nil {
		t.Fatalf("RecordUserSnapshot 3 failed: %v", err)
	}

	growth := collector.calculateFollowerGrowth("testuser")

	if len(growth) != 3 {
		t.Errorf("Expected 3 growth records, got %d", len(growth))
	}

	if growth[1].Change != 50 {
		t.Errorf("Expected change of 50, got %d", growth[1].Change)
	}

	if growth[2].Change != 70 {
		t.Errorf("Expected change of 70, got %d", growth[2].Change)
	}
}

func TestGetTopTweets(t *testing.T) {
	collector := createTestCollector(t)

	tweets := []model.Tweet{
		createTestTweet("1", "user", "Low engagement", 5, 2, 1),
		createTestTweet("2", "user", "High engagement", 100, 50, 25),
		createTestTweet("3", "user", "Medium engagement", 50, 25, 10),
		createTestTweet("4", "user", "Another high", 90, 45, 20),
		createTestTweet("5", "user", "Another medium", 40, 20, 8),
	}

	top := collector.getTopTweets(tweets, 3)

	if len(top) != 3 {
		t.Errorf("Expected 3 top tweets, got %d", len(top))
	}

	if top[0].ID != "2" {
		t.Errorf("Expected top tweet ID 2, got %s", top[0].ID)
	}

	if top[1].ID != "4" {
		t.Errorf("Expected second top tweet ID 4, got %s", top[1].ID)
	}

	if top[2].ID != "3" {
		t.Errorf("Expected third top tweet ID 3, got %s", top[2].ID)
	}
}

func TestCalculateBestHours(t *testing.T) {
	collector := createTestCollector(t)

	now := time.Now()
	tweets := []model.Tweet{
		{
			ID:        "1",
			Text:      "Morning tweet",
			CreatedAt: time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.UTC).Format(time.RubyDate),
			Metrics:   model.TweetMetrics{Likes: 100, Retweets: 50, Replies: 25},
		},
		{
			ID:        "2",
			Text:      "Afternoon tweet",
			CreatedAt: time.Date(now.Year(), now.Month(), now.Day(), 14, 0, 0, 0, time.UTC).Format(time.RubyDate),
			Metrics:   model.TweetMetrics{Likes: 50, Retweets: 25, Replies: 10},
		},
		{
			ID:        "3",
			Text:      "Another morning",
			CreatedAt: time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, time.UTC).Format(time.RubyDate),
			Metrics:   model.TweetMetrics{Likes: 80, Retweets: 40, Replies: 20},
		},
	}

	bestHours := collector.calculateBestHours(tweets)

	if len(bestHours) == 0 {
		t.Error("Expected some best hours, got none")
	}

	if bestHours[0].Hour != 9 {
		t.Errorf("Expected best hour 9, got %d", bestHours[0].Hour)
	}

	expectedAvgLikes := (100.0 + 80.0) / 2.0
	if bestHours[0].AvgLikes != expectedAvgLikes {
		t.Errorf("Expected avg likes %.2f, got %.2f", expectedAvgLikes, bestHours[0].AvgLikes)
	}
}

func TestCalculateBestDays(t *testing.T) {
	collector := createTestCollector(t)

	tuesday := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)
	thursday := time.Date(2024, 1, 4, 12, 0, 0, 0, time.UTC)

	tweets := []model.Tweet{
		{
			ID:        "1",
			Text:      "Tuesday tweet",
			CreatedAt: tuesday.Format(time.RubyDate),
			Metrics:   model.TweetMetrics{Likes: 100, Retweets: 50, Replies: 25},
		},
		{
			ID:        "2",
			Text:      "Thursday tweet",
			CreatedAt: thursday.Format(time.RubyDate),
			Metrics:   model.TweetMetrics{Likes: 50, Retweets: 25, Replies: 10},
		},
		{
			ID:        "3",
			Text:      "Another Tuesday",
			CreatedAt: tuesday.Add(2 * time.Hour).Format(time.RubyDate),
			Metrics:   model.TweetMetrics{Likes: 80, Retweets: 40, Replies: 20},
		},
	}

	bestDays := collector.calculateBestDays(tweets)

	if len(bestDays) == 0 {
		t.Error("Expected some best days, got none")
	}

	if bestDays[0].Day != 2 {
		t.Errorf("Expected best day 2 (Tuesday), got %d", bestDays[0].Day)
	}
}

func TestGetTrendingTopics(t *testing.T) {
	collector := createTestCollector(t)

	tweets := []TweetRecord{
		{
			Tweet:      createTestTweet("1", "user", "golang is awesome for backend development", 100, 50, 25),
			RecordedAt: time.Now(),
		},
		{
			Tweet:      createTestTweet("2", "user", "golang performance is incredible", 80, 40, 20),
			RecordedAt: time.Now(),
		},
		{
			Tweet:      createTestTweet("3", "user", "backend systems need good architecture", 60, 30, 15),
			RecordedAt: time.Now(),
		},
	}

	collector.mu.Lock()
	collector.tweets["user"] = tweets
	collector.mu.Unlock()

	topics, err := collector.GetTrendingTopics("user", 7)
	if err != nil {
		t.Fatalf("GetTrendingTopics failed: %v", err)
	}

	if len(topics) == 0 {
		t.Error("Expected some trending topics, got none")
	}

	foundGolang := false
	for _, topic := range topics {
		if topic.Topic == "golang" {
			foundGolang = true
			if topic.Count < 2 {
				t.Errorf("Expected golang count >= 2, got %d", topic.Count)
			}
			break
		}
	}

	if !foundGolang {
		t.Error("Expected to find 'golang' in trending topics")
	}
}

func TestExportToJSON(t *testing.T) {
	collector := createTestCollector(t)

	tweet := createTestTweet("123", "testuser", "Test tweet", 10, 5, 2)
	if err := collector.RecordTweet(tweet); err != nil {
		t.Fatalf("RecordTweet failed: %v", err)
	}

	user := createTestUserProfile("testuser", 1000)
	if err := collector.RecordUserSnapshot(user); err != nil {
		t.Fatalf("RecordUserSnapshot failed: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "export.json")

	if err := collector.ExportToJSON("testuser", tmpFile); err != nil {
		t.Fatalf("ExportToJSON failed: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Read exported file failed: %v", err)
	}

	var exported struct {
		Tweets    []TweetRecord  `json:"tweets"`
		Snapshots []UserSnapshot `json:"snapshots"`
	}

	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatalf("Parse exported JSON failed: %v", err)
	}

	if len(exported.Tweets) != 1 {
		t.Errorf("Expected 1 tweet in export, got %d", len(exported.Tweets))
	}

	if len(exported.Snapshots) != 1 {
		t.Errorf("Expected 1 snapshot in export, got %d", len(exported.Snapshots))
	}
}

func TestExportToCSV(t *testing.T) {
	collector := createTestCollector(t)

	tweet := createTestTweet("123", "testuser", "Test tweet", 10, 5, 2)
	if err := collector.RecordTweet(tweet); err != nil {
		t.Fatalf("RecordTweet failed: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "export.csv")

	if err := collector.ExportToCSV("testuser", tmpFile); err != nil {
		t.Fatalf("ExportToCSV failed: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Read exported file failed: %v", err)
	}

	content := string(data)
	if len(content) == 0 {
		t.Error("Exported CSV is empty")
	}

	if len(content) < 50 {
		t.Errorf("Exported CSV seems too short: %d bytes", len(content))
	}
}

func TestPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "analytics.db")

	collector1 := &AnalyticsCollector{
		dbPath:    dbPath,
		tweets:    make(map[string][]TweetRecord),
		snapshots: make(map[string][]UserSnapshot),
		metrics:   make(map[string][]EngagementMetrics),
	}

	tweet := createTestTweet("123", "testuser", "Test tweet", 10, 5, 2)
	if err := collector1.RecordTweet(tweet); err != nil {
		t.Fatalf("RecordTweet failed: %v", err)
	}

	user := createTestUserProfile("testuser", 1000)
	if err := collector1.RecordUserSnapshot(user); err != nil {
		t.Fatalf("RecordUserSnapshot failed: %v", err)
	}

	collector2 := &AnalyticsCollector{
		dbPath:    dbPath,
		tweets:    make(map[string][]TweetRecord),
		snapshots: make(map[string][]UserSnapshot),
		metrics:   make(map[string][]EngagementMetrics),
	}

	if err := collector2.load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	collector2.mu.RLock()
	tweets := collector2.tweets["testuser"]
	snapshots := collector2.snapshots["testuser"]
	collector2.mu.RUnlock()

	if len(tweets) != 1 {
		t.Errorf("Expected 1 tweet after reload, got %d", len(tweets))
	}

	if len(snapshots) != 1 {
		t.Errorf("Expected 1 snapshot after reload, got %d", len(snapshots))
	}
}

func TestTopKeys(t *testing.T) {
	m := map[string]int{
		"a": 10,
		"b": 30,
		"c": 20,
		"d": 5,
		"e": 25,
	}

	result := topKeys(m, 3)

	if len(result) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(result))
	}

	if result[0] != "b" {
		t.Errorf("Expected first key 'b', got %s", result[0])
	}

	if result[1] != "e" {
		t.Errorf("Expected second key 'e', got %s", result[1])
	}

	if result[2] != "c" {
		t.Errorf("Expected third key 'c', got %s", result[2])
	}
}

func TestIsStopWord(t *testing.T) {
	if !isStopWord("the") {
		t.Error("Expected 'the' to be a stop word")
	}

	if !isStopWord("and") {
		t.Error("Expected 'and' to be a stop word")
	}

	if isStopWord("golang") {
		t.Error("Expected 'golang' not to be a stop word")
	}

	if isStopWord("programming") {
		t.Error("Expected 'programming' not to be a stop word")
	}
}

func TestEngagementMetricsRecording(t *testing.T) {
	collector := createTestCollector(t)

	tweet := createTestTweet("123", "testuser", "Test", 10, 5, 2)
	if err := collector.RecordTweet(tweet); err != nil {
		t.Fatalf("RecordTweet failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	updatedTweet := createTestTweet("123", "testuser", "Test", 15, 8, 4)
	updatedTweet.Metrics.Views = 2000

	collector.mu.Lock()
	metrics := EngagementMetrics{
		TweetID:    updatedTweet.ID,
		Likes:      updatedTweet.Metrics.Likes,
		Retweets:   updatedTweet.Metrics.Retweets,
		Replies:    updatedTweet.Metrics.Replies,
		Quotes:     updatedTweet.Metrics.Quotes,
		Views:      updatedTweet.Metrics.Views,
		Bookmarks:  updatedTweet.Metrics.Bookmarks,
		RecordedAt: time.Now(),
	}
	collector.metrics["123"] = append(collector.metrics["123"], metrics)
	collector.mu.Unlock()

	collector.mu.RLock()
	history := collector.metrics["123"]
	collector.mu.RUnlock()

	if len(history) != 2 {
		t.Errorf("Expected 2 metrics records, got %d", len(history))
	}

	if history[0].Likes != 10 {
		t.Errorf("Expected first record likes 10, got %d", history[0].Likes)
	}

	if history[1].Likes != 15 {
		t.Errorf("Expected second record likes 15, got %d", history[1].Likes)
	}
}

func TestEmptyDataHandling(t *testing.T) {
	collector := createTestCollector(t)

	topTweets := collector.getTopTweets(nil, 5)
	if len(topTweets) != 0 {
		t.Errorf("Expected 0 top tweets for nil input, got %d", len(topTweets))
	}

	topTweets = collector.getTopTweets([]model.Tweet{}, 5)
	if len(topTweets) != 0 {
		t.Errorf("Expected 0 top tweets for empty input, got %d", len(topTweets))
	}

	bestHours := collector.calculateBestHours(nil)
	if len(bestHours) != 0 {
		t.Errorf("Expected 0 best hours for nil input, got %d", len(bestHours))
	}

	bestDays := collector.calculateBestDays(nil)
	if len(bestDays) != 0 {
		t.Errorf("Expected 0 best days for nil input, got %d", len(bestDays))
	}

	growth := collector.calculateFollowerGrowth("nonexistent")
	if growth != nil {
		t.Error("Expected nil growth for nonexistent user")
	}
}

func TestMultipleUsers(t *testing.T) {
	collector := createTestCollector(t)

	tweet1 := createTestTweet("1", "user1", "User 1 tweet", 10, 5, 2)
	tweet2 := createTestTweet("2", "user2", "User 2 tweet", 20, 10, 5)

	if err := collector.RecordTweet(tweet1); err != nil {
		t.Fatalf("RecordTweet user1 failed: %v", err)
	}

	if err := collector.RecordTweet(tweet2); err != nil {
		t.Fatalf("RecordTweet user2 failed: %v", err)
	}

	collector.mu.RLock()
	user1Tweets := collector.tweets["user1"]
	user2Tweets := collector.tweets["user2"]
	collector.mu.RUnlock()

	if len(user1Tweets) != 1 {
		t.Errorf("Expected 1 tweet for user1, got %d", len(user1Tweets))
	}

	if len(user2Tweets) != 1 {
		t.Errorf("Expected 1 tweet for user2, got %d", len(user2Tweets))
	}

	if user1Tweets[0].Tweet.ID != "1" {
		t.Errorf("Expected user1 tweet ID 1, got %s", user1Tweets[0].Tweet.ID)
	}

	if user2Tweets[0].Tweet.ID != "2" {
		t.Errorf("Expected user2 tweet ID 2, got %s", user2Tweets[0].Tweet.ID)
	}
}

func TestTimeBasedFiltering(t *testing.T) {
	collector := createTestCollector(t)

	oldTweet := TweetRecord{
		Tweet:      createTestTweet("old", "user", "Old tweet", 10, 5, 2),
		RecordedAt: time.Now().AddDate(0, 0, -10),
	}

	recentTweet := TweetRecord{
		Tweet:      createTestTweet("recent", "user", "Recent tweet", 20, 10, 5),
		RecordedAt: time.Now(),
	}

	collector.mu.Lock()
	collector.tweets["user"] = []TweetRecord{oldTweet, recentTweet}
	collector.mu.Unlock()

	topics, err := collector.GetTrendingTopics("user", 7)
	if err != nil {
		t.Fatalf("GetTrendingTopics failed: %v", err)
	}

	for _, topic := range topics {
		if topic.Topic == "old" {
			t.Error("Old tweet topic should not appear in recent trending")
		}
	}
}
