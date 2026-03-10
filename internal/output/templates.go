package output

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/model"
)

type TemplateEngine struct {
	template *template.Template
	raw      string
}

type TemplateData struct {
	User   model.UserProfile
	Tweet  model.Tweet
	Tweets []model.Tweet
}

var builtInFuncs = template.FuncMap{
	"upper":     strings.ToUpper,
	"lower":     strings.ToLower,
	"title":     strings.Title,
	"trim":      strings.TrimSpace,
	"truncate":  truncateStr,
	"date":      formatDate,
	"reltime":   relativeTime,
	"num":       formatNumber,
	"k":         formatK,
	"hashtag":   extractHashtags,
	"mention":   extractMentions,
	"url":       extractURLs,
	"word":      extractWords,
	"join":      strings.Join,
	"split":     strings.Split,
	"replace":   strings.ReplaceAll,
	"contains":  strings.Contains,
	"hasPrefix": strings.HasPrefix,
	"hasSuffix": strings.HasSuffix,
	"count":     length,
	"default":   defaultValue,
	"coalesce":  coalesceValues,
	"add":       add,
	"escape":    escapeCSV,
}

var builtInTemplates = map[string]string{
	"tweet.simple":   "{{.Tweet.Text}}",
	"tweet.full":     "@{{.Tweet.Author.ScreenName}}: {{.Tweet.Text}} ({{.Tweet.Metrics.Likes}} likes)",
	"tweet.metrics":  "❤️ {{.Tweet.Metrics.Likes}} 🔄 {{.Tweet.Metrics.Retweets}} 💬 {{.Tweet.Metrics.Replies}} 👁️ {{.Tweet.Metrics.Views}}",
	"tweet.line":     "[{{.Tweet.ID}}] @{{.Tweet.Author.ScreenName}}: {{.Tweet.Text | truncate 80}}",
	"user.simple":    "{{.User.Name}} (@{{.User.ScreenName}})",
	"user.bio":       "{{.User.Name}} (@{{.User.ScreenName}})\n{{.User.Bio}}",
	"user.stats":     "{{.User.Name}}: {{.User.FollowersCount | k}} followers, {{.User.FollowingCount | k}} following",
	"user.oneline":   "{{.User.Name}}|@{{.User.ScreenName}}|{{.User.FollowersCount}}|{{.User.TweetsCount}}",
	"tweets.list":    "{{range .Tweets}}- {{.Text}}\n{{end}}",
	"tweets.compact": "{{range $i, $t := .Tweets}}{{add $i 1}}. {{$t.Text | truncate 60}}\n{{end}}",
	"tweets.csv":     "id,author,text,likes\n{{range .Tweets}}{{.ID}},{{.Author.ScreenName}},{{.Text | escape}},{{.Metrics.Likes}}\n{{end}}",
}

func NewTemplateEngine(templateStr string) (*TemplateEngine, error) {
	if strings.HasPrefix(templateStr, "@") && !strings.Contains(templateStr, "{{") {
		presetName := templateStr[1:]
		if preset, ok := builtInTemplates[presetName]; ok {
			templateStr = preset
		} else {
			return nil, fmt.Errorf("unknown template preset: %s", presetName)
		}
	}

	tmpl, err := template.New("output").Funcs(builtInFuncs).Parse(templateStr)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	return &TemplateEngine{
		template: tmpl,
		raw:      templateStr,
	}, nil
}

func NewTemplateEngineFromFile(path string) (*TemplateEngine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read template file: %w", err)
	}

	return NewTemplateEngine(string(data))
}

func LoadTemplate(templateArg string) (*TemplateEngine, error) {
	if templateArg == "" {
		return nil, nil
	}

	if isFileTemplate(templateArg) {
		return NewTemplateEngineFromFile(templateArg)
	}

	return NewTemplateEngine(templateArg)
}

func isFileTemplate(s string) bool {
	if strings.Contains(s, "{{") && strings.Contains(s, "}}") {
		return false
	}

	ext := strings.ToLower(filepath.Ext(s))
	return ext == ".txt" || ext == ".tmpl" || ext == ".gotmpl" || ext == ".template"
}

func (e *TemplateEngine) Execute(data TemplateData) (string, error) {
	var buf bytes.Buffer
	if err := e.template.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}

func (e *TemplateEngine) ExecuteUser(user model.UserProfile) (string, error) {
	return e.Execute(TemplateData{User: user})
}

func (e *TemplateEngine) ExecuteTweet(tweet model.Tweet) (string, error) {
	return e.Execute(TemplateData{Tweet: tweet})
}

func (e *TemplateEngine) ExecuteTweets(tweets []model.Tweet) (string, error) {
	return e.Execute(TemplateData{Tweets: tweets})
}

func (e *TemplateEngine) Raw() string {
	return e.raw
}

func ListPresets() []string {
	presets := make([]string, 0, len(builtInTemplates))
	for name := range builtInTemplates {
		presets = append(presets, name)
	}
	return presets
}

func GetPreset(name string) (string, bool) {
	tmpl, ok := builtInTemplates[name]
	return tmpl, ok
}

func ValidateTemplate(templateStr string) error {
	_, err := NewTemplateEngine(templateStr)
	return err
}

func formatDate(layout, dateStr string) (string, error) {
	if dateStr == "" {
		return "", nil
	}

	t, err := parseTweetDate(dateStr)
	if err != nil {
		return "", err
	}

	if layout == "" {
		layout = "2006-01-02 15:04"
	}

	return t.Format(layout), nil
}

func relativeTime(dateStr string) (string, error) {
	if dateStr == "" {
		return "", nil
	}

	t, err := parseTweetDate(dateStr)
	if err != nil {
		return "", err
	}

	return formatRelativeTime(t), nil
}

func parseTweetDate(dateStr string) (time.Time, error) {
	layouts := []string{
		"Mon Jan 2 15:04:05 -0700 2006",
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse date: %s", dateStr)
}

func formatRelativeTime(t time.Time) string {
	now := time.Now()
	d := now.Sub(t)

	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1m"
		}
		return fmt.Sprintf("%dm", mins)
	}
	if d < 24*time.Hour {
		hrs := int(d.Hours())
		if hrs == 1 {
			return "1h"
		}
		return fmt.Sprintf("%dh", hrs)
	}
	if d < 7*24*time.Hour {
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d"
		}
		return fmt.Sprintf("%dd", days)
	}
	if d < 30*24*time.Hour {
		weeks := int(d.Hours() / 24 / 7)
		if weeks == 1 {
			return "1w"
		}
		return fmt.Sprintf("%dw", weeks)
	}
	if d < 365*24*time.Hour {
		months := int(d.Hours() / 24 / 30)
		if months == 1 {
			return "1mo"
		}
		return fmt.Sprintf("%dmo", months)
	}

	years := int(d.Hours() / 24 / 365)
	if years == 1 {
		return "1y"
	}
	return fmt.Sprintf("%dy", years)
}

func formatNumber(n int) string {
	return fmt.Sprintf("%d", n)
}

func formatK(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func extractHashtags(text string) []string {
	words := strings.Fields(text)
	var hashtags []string
	for _, word := range words {
		if strings.HasPrefix(word, "#") && len(word) > 1 {
			hashtags = append(hashtags, word)
		}
	}
	return hashtags
}

func extractMentions(text string) []string {
	words := strings.Fields(text)
	var mentions []string
	for _, word := range words {
		if strings.HasPrefix(word, "@") && len(word) > 1 {
			mentions = append(mentions, word)
		}
	}
	return mentions
}

func extractURLs(text string) []string {
	words := strings.Fields(text)
	var urls []string
	for _, word := range words {
		if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
			urls = append(urls, word)
		}
	}
	return urls
}

func extractWords(text string) []string {
	return strings.Fields(text)
}

func defaultValue(defaultVal, val string) string {
	if val == "" {
		return defaultVal
	}
	return val
}

func coalesceValues(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func add(a, b int) int {
	return a + b
}

func escapeCSV(s string) string {
	s = strings.ReplaceAll(s, "\"", "\"\"")
	return s
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func length(v interface{}) int {
	switch val := v.(type) {
	case string:
		return len(val)
	case []model.Tweet:
		return len(val)
	case []string:
		return len(val)
	default:
		return 0
	}
}
