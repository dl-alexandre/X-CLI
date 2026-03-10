package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/model"
	"github.com/dl-alexandre/X-CLI/internal/xapi"
)

type Monitor struct {
	client       *xapi.Client
	statePath    string
	state        MonitorState
	mu           sync.Mutex
	pollInterval time.Duration
	notify       bool
	filter       *MentionFilter
}

type MonitorState struct {
	LastCheckedAt time.Time `json:"last_checked_at"`
	LastTweetID   string    `json:"last_tweet_id,omitempty"`
}

type MentionFilter struct {
	Keywords []string
	Users    []string
}

type Mention struct {
	Tweet     model.Tweet `json:"tweet"`
	MatchedBy string      `json:"matched_by,omitempty"`
	IsNew     bool        `json:"is_new"`
}

type MonitorResult struct {
	Mentions    []Mention `json:"mentions"`
	TotalCount  int       `json:"total_count"`
	NewCount    int       `json:"new_count"`
	LastChecked time.Time `json:"last_checked"`
}

type Options struct {
	PollInterval time.Duration
	Notify       bool
	Filter       *MentionFilter
}

func NewMonitor(client *xapi.Client, opts Options) (*Monitor, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	statePath := filepath.Join(home, ".config", "x-cli", "monitor.json")

	m := &Monitor{
		client:       client,
		statePath:    statePath,
		pollInterval: opts.PollInterval,
		notify:       opts.Notify,
		filter:       opts.Filter,
	}

	if m.pollInterval == 0 {
		m.pollInterval = 60 * time.Second
	}

	if err := m.loadState(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load state: %w", err)
	}

	return m, nil
}

func (m *Monitor) loadState() error {
	data, err := os.ReadFile(m.statePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &m.state)
}

func (m *Monitor) saveState() error {
	if err := os.MkdirAll(filepath.Dir(m.statePath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.statePath, data, 0644)
}

func (m *Monitor) FetchMentions(since time.Duration) (MonitorResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	result, err := m.client.Search("to:me", "Latest", 50)
	if err != nil {
		return MonitorResult{}, fmt.Errorf("fetch mentions: %w", err)
	}

	var mentions []Mention
	now := time.Now()
	cutoff := now.Add(-since)

	for _, tweet := range result.Tweets {
		mention := Mention{
			Tweet: tweet,
			IsNew: m.isNewMention(tweet, cutoff),
		}

		if m.filter != nil && !m.matchesFilter(tweet) {
			continue
		}

		mentions = append(mentions, mention)
	}

	newCount := 0
	for _, m := range mentions {
		if m.IsNew {
			newCount++
		}
	}

	m.state.LastCheckedAt = now
	if len(mentions) > 0 {
		m.state.LastTweetID = mentions[0].Tweet.ID
	}
	_ = m.saveState()

	return MonitorResult{
		Mentions:    mentions,
		TotalCount:  len(mentions),
		NewCount:    newCount,
		LastChecked: now,
	}, nil
}

func (m *Monitor) isNewMention(tweet model.Tweet, cutoff time.Time) bool {
	if m.state.LastTweetID == tweet.ID {
		return false
	}

	if m.state.LastCheckedAt.IsZero() {
		return true
	}

	tweetTime, err := parseTweetTime(tweet.CreatedAt)
	if err != nil {
		return tweetTime.After(m.state.LastCheckedAt)
	}

	return tweetTime.After(cutoff)
}

func (m *Monitor) matchesFilter(tweet model.Tweet) bool {
	if m.filter == nil {
		return true
	}

	text := strings.ToLower(tweet.Text)

	for _, keyword := range m.filter.Keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}

	for _, user := range m.filter.Users {
		screenName := strings.ToLower(tweet.Author.ScreenName)
		if strings.Contains(screenName, strings.ToLower(user)) {
			return true
		}
		if strings.Contains(text, "@"+strings.ToLower(user)) {
			return true
		}
	}

	return len(m.filter.Keywords) == 0 && len(m.filter.Users) == 0
}

func (m *Monitor) Watch(ctx context.Context, onMention func(MonitorResult)) error {
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			result, err := m.FetchMentions(m.pollInterval * 2)
			if err != nil {
				continue
			}

			if len(result.Mentions) > 0 {
				if m.notify {
					m.sendNotifications(result)
				}
				onMention(result)
			}
		}
	}
}

func (m *Monitor) sendNotifications(result MonitorResult) {
	for _, mention := range result.Mentions {
		if !mention.IsNew {
			continue
		}

		title := fmt.Sprintf("@%s mentioned you", mention.Tweet.Author.ScreenName)
		body := truncateText(mention.Tweet.Text, 100)

		go m.sendDesktopNotification(title, body)
	}
}

func (m *Monitor) sendDesktopNotification(title, body string) {
	var cmd *exec.Cmd

	switch {
	case commandExists("osascript"):
		script := fmt.Sprintf(`display notification %q with title %q`, body, title)
		cmd = exec.Command("osascript", "-e", script)
	case commandExists("notify-send"):
		cmd = exec.Command("notify-send", title, body)
	case commandExists("terminal-notifier"):
		cmd = exec.Command("terminal-notifier", "-title", title, "-message", body)
	default:
		return
	}

	_ = cmd.Run()
}

func (m *Monitor) GetState() MonitorState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *Monitor) ResetState() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state = MonitorState{}
	return m.saveState()
}

func (m *Monitor) SetPollInterval(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pollInterval = d
}

func (m *Monitor) SetFilter(filter *MentionFilter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.filter = filter
}

func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	re := regexp.MustCompile(`^(\d+)([hms]?)$`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}

	value := matches[1]
	unit := matches[2]

	var multiplier time.Duration
	switch unit {
	case "h":
		multiplier = time.Hour
	case "m":
		multiplier = time.Minute
	case "s", "":
		multiplier = time.Second
	default:
		return 0, fmt.Errorf("unknown duration unit: %s", unit)
	}

	var val int
	fmt.Sscanf(value, "%d", &val)

	return time.Duration(val) * multiplier, nil
}

func parseTweetTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}

	layouts := []string{
		"Mon Jan 2 15:04:05 -0700 2006",
		time.RFC3339,
		"2006-01-02T15:04:05Z",
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse time: %s", s)
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func FormatMention(mention Mention) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("@%s: ", mention.Tweet.Author.ScreenName))
	sb.WriteString(truncateText(mention.Tweet.Text, 100))

	if mention.IsNew {
		sb.WriteString(" [NEW]")
	}

	if mention.MatchedBy != "" {
		sb.WriteString(fmt.Sprintf(" (matched: %s)", mention.MatchedBy))
	}

	return sb.String()
}

func FormatMentions(result MonitorResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Mentions: %d total, %d new\n", result.TotalCount, result.NewCount))
	sb.WriteString(fmt.Sprintf("Last checked: %s\n\n", result.LastChecked.Format("2006-01-02 15:04:05")))

	for _, mention := range result.Mentions {
		sb.WriteString(FormatMention(mention))
		sb.WriteString("\n")
	}

	return sb.String()
}
