package analytics

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/model"
	"github.com/dl-alexandre/X-CLI/internal/xapi"
)

type EngagementMetrics struct {
	TweetID    string    `json:"tweet_id"`
	Likes      int       `json:"likes"`
	Retweets   int       `json:"retweets"`
	Replies    int       `json:"replies"`
	Quotes     int       `json:"quotes"`
	Views      int       `json:"views"`
	Bookmarks  int       `json:"bookmarks"`
	RecordedAt time.Time `json:"recorded_at"`
}

type UserSnapshot struct {
	ScreenName     string    `json:"screen_name"`
	FollowersCount int       `json:"followers_count"`
	FollowingCount int       `json:"following_count"`
	TweetsCount    int       `json:"tweets_count"`
	RecordedAt     time.Time `json:"recorded_at"`
}

type TweetRecord struct {
	Tweet      model.Tweet `json:"tweet"`
	RecordedAt time.Time   `json:"recorded_at"`
}

type HourlyEngagement struct {
	Hour        int     `json:"hour"`
	AvgLikes    float64 `json:"avg_likes"`
	AvgRetweets float64 `json:"avg_retweets"`
	AvgReplies  float64 `json:"avg_replies"`
	TweetCount  int     `json:"tweet_count"`
}

type DayOfWeekEngagement struct {
	Day         int     `json:"day"`
	AvgLikes    float64 `json:"avg_likes"`
	AvgRetweets float64 `json:"avg_retweets"`
	AvgReplies  float64 `json:"avg_replies"`
	TweetCount  int     `json:"tweet_count"`
}

type TrendingTopic struct {
	Topic      string `json:"topic"`
	Count      int    `json:"count"`
	Engagement int    `json:"engagement"`
}

type UserAnalyticsReport struct {
	ScreenName          string                `json:"screen_name"`
	Period              string                `json:"period"`
	TotalTweets         int                   `json:"total_tweets"`
	TotalLikes          int                   `json:"total_likes"`
	TotalRetweets       int                   `json:"total_retweets"`
	TotalReplies        int                   `json:"total_replies"`
	AvgLikesPerTweet    float64               `json:"avg_likes_per_tweet"`
	AvgRetweetsPerTweet float64               `json:"avg_retweets_per_tweet"`
	AvgRepliesPerTweet  float64               `json:"avg_replies_per_tweet"`
	EngagementRate      float64               `json:"engagement_rate"`
	TopTweets           []model.Tweet         `json:"top_tweets"`
	BestHours           []HourlyEngagement    `json:"best_hours"`
	BestDays            []DayOfWeekEngagement `json:"best_days"`
	FollowerGrowth      []FollowerGrowth      `json:"follower_growth"`
	TopHashtags         []string              `json:"top_hashtags"`
	TopMentions         []string              `json:"top_mentions"`
}

type TweetAnalyticsReport struct {
	TweetID        string              `json:"tweet_id"`
	Text           string              `json:"text"`
	Author         string              `json:"author"`
	CreatedAt      string              `json:"created_at"`
	CurrentMetrics EngagementMetrics   `json:"current_metrics"`
	History        []EngagementMetrics `json:"history"`
	GrowthRate     EngagementGrowth    `json:"growth_rate"`
}

type EngagementGrowth struct {
	LikesPerHour    float64 `json:"likes_per_hour"`
	RetweetsPerHour float64 `json:"retweets_per_hour"`
	RepliesPerHour  float64 `json:"replies_per_hour"`
}

type FollowerGrowth struct {
	Date       string  `json:"date"`
	Count      int     `json:"count"`
	Change     int     `json:"change"`
	ChangeRate float64 `json:"change_rate"`
}

type BestPostingTimes struct {
	BestHours []HourlyEngagement    `json:"best_hours"`
	BestDays  []DayOfWeekEngagement `json:"best_days"`
}

type AnalyticsCollector struct {
	client    *xapi.Client
	dbPath    string
	mu        sync.RWMutex
	tweets    map[string][]TweetRecord
	snapshots map[string][]UserSnapshot
	metrics   map[string][]EngagementMetrics
}

func NewAnalyticsCollector(client *xapi.Client) (*AnalyticsCollector, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home directory: %w", err)
	}

	dbPath := filepath.Join(homeDir, ".config", "x-cli", "analytics.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create config directory: %w", err)
	}

	collector := &AnalyticsCollector{
		client:    client,
		dbPath:    dbPath,
		tweets:    make(map[string][]TweetRecord),
		snapshots: make(map[string][]UserSnapshot),
		metrics:   make(map[string][]EngagementMetrics),
	}

	if err := collector.load(); err != nil {
		return collector, nil
	}

	return collector, nil
}

func (c *AnalyticsCollector) load() error {
	data, err := os.ReadFile(c.dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read analytics db: %w", err)
	}

	var stored struct {
		Tweets    map[string][]TweetRecord       `json:"tweets"`
		Snapshots map[string][]UserSnapshot      `json:"snapshots"`
		Metrics   map[string][]EngagementMetrics `json:"metrics"`
	}

	if err := json.Unmarshal(data, &stored); err != nil {
		return fmt.Errorf("parse analytics db: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if stored.Tweets != nil {
		c.tweets = stored.Tweets
	}
	if stored.Snapshots != nil {
		c.snapshots = stored.Snapshots
	}
	if stored.Metrics != nil {
		c.metrics = stored.Metrics
	}

	return nil
}

func (c *AnalyticsCollector) save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.saveLocked()
}

func (c *AnalyticsCollector) saveLocked() error {
	data, err := json.MarshalIndent(struct {
		Tweets    map[string][]TweetRecord       `json:"tweets"`
		Snapshots map[string][]UserSnapshot      `json:"snapshots"`
		Metrics   map[string][]EngagementMetrics `json:"metrics"`
	}{
		Tweets:    c.tweets,
		Snapshots: c.snapshots,
		Metrics:   c.metrics,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal analytics db: %w", err)
	}

	if err := os.WriteFile(c.dbPath, data, 0600); err != nil {
		return fmt.Errorf("write analytics db: %w", err)
	}

	return nil
}

func (c *AnalyticsCollector) RecordTweet(tweet model.Tweet) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	screenName := tweet.Author.ScreenName
	record := TweetRecord{
		Tweet:      tweet,
		RecordedAt: time.Now(),
	}

	c.tweets[screenName] = append(c.tweets[screenName], record)

	metrics := EngagementMetrics{
		TweetID:    tweet.ID,
		Likes:      tweet.Metrics.Likes,
		Retweets:   tweet.Metrics.Retweets,
		Replies:    tweet.Metrics.Replies,
		Quotes:     tweet.Metrics.Quotes,
		Views:      tweet.Metrics.Views,
		Bookmarks:  tweet.Metrics.Bookmarks,
		RecordedAt: time.Now(),
	}

	c.metrics[tweet.ID] = append(c.metrics[tweet.ID], metrics)

	return c.saveLocked()
}

func (c *AnalyticsCollector) RecordUserSnapshot(user model.UserProfile) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	snapshot := UserSnapshot{
		ScreenName:     user.ScreenName,
		FollowersCount: user.FollowersCount,
		FollowingCount: user.FollowingCount,
		TweetsCount:    user.TweetsCount,
		RecordedAt:     time.Now(),
	}

	c.snapshots[user.ScreenName] = append(c.snapshots[user.ScreenName], snapshot)

	return c.saveLocked()
}

func (c *AnalyticsCollector) UpdateTweetMetrics(tweetID string) error {
	thread, err := c.client.Tweet(tweetID, 0)
	if err != nil {
		return fmt.Errorf("fetch post: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	metrics := EngagementMetrics{
		TweetID:    thread.Tweet.ID,
		Likes:      thread.Tweet.Metrics.Likes,
		Retweets:   thread.Tweet.Metrics.Retweets,
		Replies:    thread.Tweet.Metrics.Replies,
		Quotes:     thread.Tweet.Metrics.Quotes,
		Views:      thread.Tweet.Metrics.Views,
		Bookmarks:  thread.Tweet.Metrics.Bookmarks,
		RecordedAt: time.Now(),
	}

	c.metrics[tweetID] = append(c.metrics[tweetID], metrics)

	return c.saveLocked()
}

func (c *AnalyticsCollector) CollectUserAnalytics(screenName string, days int) (*UserAnalyticsReport, error) {
	user, err := c.client.User(screenName)
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", err)
	}

	if err := c.RecordUserSnapshot(user); err != nil {
		return nil, fmt.Errorf("record snapshot: %w", err)
	}

	result, err := c.client.UserPosts(screenName, 100)
	if err != nil {
		return nil, fmt.Errorf("fetch user posts: %w", err)
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	var recentTweets []model.Tweet

	for _, tweet := range result.Tweets {
		if err := c.RecordTweet(tweet); err != nil {
			continue
		}

		createdAt, parseErr := time.Parse(time.RubyDate, tweet.CreatedAt)
		if parseErr == nil && createdAt.After(cutoff) {
			recentTweets = append(recentTweets, tweet)
		}
	}

	report := &UserAnalyticsReport{
		ScreenName:  screenName,
		Period:      fmt.Sprintf("%d days", days),
		TotalTweets: len(recentTweets),
	}

	if len(recentTweets) == 0 {
		return report, nil
	}

	totalLikes := 0
	totalRetweets := 0
	totalReplies := 0
	hashtagCounts := make(map[string]int)
	mentionCounts := make(map[string]int)

	for _, tweet := range recentTweets {
		totalLikes += tweet.Metrics.Likes
		totalRetweets += tweet.Metrics.Retweets
		totalReplies += tweet.Metrics.Replies

		words := strings.Fields(strings.ToLower(tweet.Text))
		for _, word := range words {
			if strings.HasPrefix(word, "#") && len(word) > 1 {
				hashtagCounts[word]++
			} else if strings.HasPrefix(word, "@") && len(word) > 1 {
				mentionCounts[word]++
			}
		}
	}

	report.TotalLikes = totalLikes
	report.TotalRetweets = totalRetweets
	report.TotalReplies = totalReplies
	report.AvgLikesPerTweet = float64(totalLikes) / float64(len(recentTweets))
	report.AvgRetweetsPerTweet = float64(totalRetweets) / float64(len(recentTweets))
	report.AvgRepliesPerTweet = float64(totalReplies) / float64(len(recentTweets))

	if user.FollowersCount > 0 {
		report.EngagementRate = (float64(totalLikes+totalRetweets+totalReplies) / float64(user.FollowersCount) / float64(len(recentTweets))) * 100
	}

	report.TopTweets = c.getTopTweets(recentTweets, 5)
	report.BestHours = c.calculateBestHours(recentTweets)
	report.BestDays = c.calculateBestDays(recentTweets)
	report.FollowerGrowth = c.calculateFollowerGrowth(screenName)
	report.TopHashtags = topKeys(hashtagCounts, 10)
	report.TopMentions = topKeys(mentionCounts, 10)

	return report, nil
}

func (c *AnalyticsCollector) CollectTweetAnalytics(tweetID string) (*TweetAnalyticsReport, error) {
	thread, err := c.client.Tweet(tweetID, 0)
	if err != nil {
		return nil, fmt.Errorf("fetch post: %w", err)
	}

	tweet := thread.Tweet

	if err := c.RecordTweet(tweet); err != nil {
		return nil, fmt.Errorf("record post: %w", err)
	}

	report := &TweetAnalyticsReport{
		TweetID:   tweet.ID,
		Text:      tweet.Text,
		Author:    tweet.Author.ScreenName,
		CreatedAt: tweet.CreatedAt,
		CurrentMetrics: EngagementMetrics{
			TweetID:    tweet.ID,
			Likes:      tweet.Metrics.Likes,
			Retweets:   tweet.Metrics.Retweets,
			Replies:    tweet.Metrics.Replies,
			Quotes:     tweet.Metrics.Quotes,
			Views:      tweet.Metrics.Views,
			Bookmarks:  tweet.Metrics.Bookmarks,
			RecordedAt: time.Now(),
		},
	}

	c.mu.RLock()
	history := c.metrics[tweetID]
	c.mu.RUnlock()

	report.History = history

	if len(history) >= 2 {
		oldest := history[0]
		newest := history[len(history)-1]
		timeDiff := newest.RecordedAt.Sub(oldest.RecordedAt).Hours()

		if timeDiff > 0 {
			report.GrowthRate = EngagementGrowth{
				LikesPerHour:    float64(newest.Likes-oldest.Likes) / timeDiff,
				RetweetsPerHour: float64(newest.Retweets-oldest.Retweets) / timeDiff,
				RepliesPerHour:  float64(newest.Replies-oldest.Replies) / timeDiff,
			}
		}
	}

	return report, nil
}

func (c *AnalyticsCollector) GetTrendingTopics(screenName string, days int) ([]TrendingTopic, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tweetRecords := c.tweets[screenName]
	if len(tweetRecords) == 0 {
		return nil, nil
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	topicCounts := make(map[string]int)
	topicEngagement := make(map[string]int)

	for _, record := range tweetRecords {
		if record.RecordedAt.Before(cutoff) {
			continue
		}

		words := strings.Fields(strings.ToLower(record.Tweet.Text))
		engagement := record.Tweet.Metrics.Likes + record.Tweet.Metrics.Retweets + record.Tweet.Metrics.Replies

		for _, word := range words {
			if len(word) > 4 && !isStopWord(word) {
				topicCounts[word]++
				topicEngagement[word] += engagement
			}
		}
	}

	var topics []TrendingTopic
	for topic, count := range topicCounts {
		if count >= 2 {
			topics = append(topics, TrendingTopic{
				Topic:      topic,
				Count:      count,
				Engagement: topicEngagement[topic],
			})
		}
	}

	sort.Slice(topics, func(i, j int) bool {
		return topics[i].Engagement > topics[j].Engagement
	})

	if len(topics) > 20 {
		topics = topics[:20]
	}

	return topics, nil
}

func (c *AnalyticsCollector) GetBestPostingTimes(screenName string) (*BestPostingTimes, error) {
	c.mu.RLock()
	tweetRecords := c.tweets[screenName]
	c.mu.RUnlock()

	if len(tweetRecords) == 0 {
		return &BestPostingTimes{}, nil
	}

	var tweets []model.Tweet
	for _, record := range tweetRecords {
		tweets = append(tweets, record.Tweet)
	}

	return &BestPostingTimes{
		BestHours: c.calculateBestHours(tweets),
		BestDays:  c.calculateBestDays(tweets),
	}, nil
}

func (c *AnalyticsCollector) getTopTweets(tweets []model.Tweet, n int) []model.Tweet {
	sorted := make([]model.Tweet, len(tweets))
	copy(sorted, tweets)

	sort.Slice(sorted, func(i, j int) bool {
		engagementI := sorted[i].Metrics.Likes + sorted[i].Metrics.Retweets + sorted[i].Metrics.Replies
		engagementJ := sorted[j].Metrics.Likes + sorted[j].Metrics.Retweets + sorted[j].Metrics.Replies
		return engagementI > engagementJ
	})

	if len(sorted) > n {
		sorted = sorted[:n]
	}

	return sorted
}

func (c *AnalyticsCollector) calculateBestHours(tweets []model.Tweet) []HourlyEngagement {
	hourlyData := make(map[int]struct {
		totalLikes, totalRetweets, totalReplies, count int
	})

	for _, tweet := range tweets {
		createdAt, err := time.Parse(time.RubyDate, tweet.CreatedAt)
		if err != nil {
			continue
		}

		hour := createdAt.Hour()
		data := hourlyData[hour]
		data.totalLikes += tweet.Metrics.Likes
		data.totalRetweets += tweet.Metrics.Retweets
		data.totalReplies += tweet.Metrics.Replies
		data.count++
		hourlyData[hour] = data
	}

	var results []HourlyEngagement
	for hour, data := range hourlyData {
		if data.count > 0 {
			results = append(results, HourlyEngagement{
				Hour:        hour,
				AvgLikes:    float64(data.totalLikes) / float64(data.count),
				AvgRetweets: float64(data.totalRetweets) / float64(data.count),
				AvgReplies:  float64(data.totalReplies) / float64(data.count),
				TweetCount:  data.count,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		avgI := results[i].AvgLikes + results[i].AvgRetweets + results[i].AvgReplies
		avgJ := results[j].AvgLikes + results[j].AvgRetweets + results[j].AvgReplies
		return avgI > avgJ
	})

	if len(results) > 5 {
		results = results[:5]
	}

	return results
}

func (c *AnalyticsCollector) calculateBestDays(tweets []model.Tweet) []DayOfWeekEngagement {
	dayData := make(map[int]struct {
		totalLikes, totalRetweets, totalReplies, count int
	})

	for _, tweet := range tweets {
		createdAt, err := time.Parse(time.RubyDate, tweet.CreatedAt)
		if err != nil {
			continue
		}

		day := int(createdAt.Weekday())
		data := dayData[day]
		data.totalLikes += tweet.Metrics.Likes
		data.totalRetweets += tweet.Metrics.Retweets
		data.totalReplies += tweet.Metrics.Replies
		data.count++
		dayData[day] = data
	}

	var results []DayOfWeekEngagement
	for day, data := range dayData {
		if data.count > 0 {
			results = append(results, DayOfWeekEngagement{
				Day:         day,
				AvgLikes:    float64(data.totalLikes) / float64(data.count),
				AvgRetweets: float64(data.totalRetweets) / float64(data.count),
				AvgReplies:  float64(data.totalReplies) / float64(data.count),
				TweetCount:  data.count,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		avgI := results[i].AvgLikes + results[i].AvgRetweets + results[i].AvgReplies
		avgJ := results[j].AvgLikes + results[j].AvgRetweets + results[j].AvgReplies
		return avgI > avgJ
	})

	return results
}

func (c *AnalyticsCollector) calculateFollowerGrowth(screenName string) []FollowerGrowth {
	c.mu.RLock()
	snapshots := c.snapshots[screenName]
	c.mu.RUnlock()

	if len(snapshots) < 2 {
		return nil
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].RecordedAt.Before(snapshots[j].RecordedAt)
	})

	var growth []FollowerGrowth
	for i, snapshot := range snapshots {
		change := 0
		changeRate := 0.0

		if i > 0 {
			prev := snapshots[i-1]
			change = snapshot.FollowersCount - prev.FollowersCount
			if prev.FollowersCount > 0 {
				changeRate = float64(change) / float64(prev.FollowersCount) * 100
			}
		}

		growth = append(growth, FollowerGrowth{
			Date:       snapshot.RecordedAt.Format("2006-01-02"),
			Count:      snapshot.FollowersCount,
			Change:     change,
			ChangeRate: changeRate,
		})
	}

	return growth
}

func (c *AnalyticsCollector) ExportToJSON(screenName string, outputPath string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data := struct {
		Tweets    []TweetRecord                  `json:"tweets"`
		Snapshots []UserSnapshot                 `json:"snapshots"`
		Metrics   map[string][]EngagementMetrics `json:"metrics"`
	}{
		Tweets:    c.tweets[screenName],
		Snapshots: c.snapshots[screenName],
		Metrics:   c.metrics,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("write json: %w", err)
	}

	return nil
}

func (c *AnalyticsCollector) ExportToCSV(screenName string, outputPath string) error {
	c.mu.RLock()
	tweetRecords := c.tweets[screenName]
	c.mu.RUnlock()

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create csv: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"post_id", "text", "created_at", "likes", "reposts", "replies", "quotes", "views", "bookmarks", "recorded_at"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for _, record := range tweetRecords {
		row := []string{
			record.Tweet.ID,
			record.Tweet.Text,
			record.Tweet.CreatedAt,
			fmt.Sprintf("%d", record.Tweet.Metrics.Likes),
			fmt.Sprintf("%d", record.Tweet.Metrics.Retweets),
			fmt.Sprintf("%d", record.Tweet.Metrics.Replies),
			fmt.Sprintf("%d", record.Tweet.Metrics.Quotes),
			fmt.Sprintf("%d", record.Tweet.Metrics.Views),
			fmt.Sprintf("%d", record.Tweet.Metrics.Bookmarks),
			record.RecordedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}

	return nil
}

func topKeys(m map[string]int, n int) []string {
	type kv struct {
		Key   string
		Value int
	}

	var pairs []kv
	for k, v := range m {
		pairs = append(pairs, kv{k, v})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value
	})

	result := make([]string, 0, n)
	for i := 0; i < len(pairs) && i < n; i++ {
		result = append(result, pairs[i].Key)
	}

	return result
}

func isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "and": true, "for": true, "are": true, "but": true,
		"not": true, "you": true, "all": true, "can": true, "had": true,
		"her": true, "was": true, "one": true, "our": true, "out": true,
		"has": true, "have": true, "been": true, "will": true, "would": true,
		"there": true, "their": true, "what": true, "about": true, "which": true,
		"when": true, "this": true, "that": true, "with": true, "from": true,
	}
	return stopWords[word]
}
