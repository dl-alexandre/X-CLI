package output

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/model"
	"github.com/rodaine/table"
)

type Printer struct {
	format   string
	useColor bool
}

func NewPrinter(format string, useColor bool) *Printer {
	return &Printer{format: format, useColor: useColor}
}

func (p *Printer) PrintStatus(status model.ProjectStatus) error {
	return p.printAny(status)
}

func (p *Printer) PrintTimeline(result model.TimelineResult) error {
	switch p.format {
	case "json":
		return p.printAny(result)
	case "markdown":
		fmt.Println("# Posts")
		fmt.Println()
		for _, tweet := range result.Tweets {
			fmt.Printf("- @%s: %s\n", tweet.Author.ScreenName, tweet.Text)
		}
		return nil
	default:
		return p.printTweetsTable(result.Tweets)
	}
}

func (p *Printer) PrintTweetThread(thread model.TweetThread) error {
	switch p.format {
	case "json":
		return p.printAny(thread)
	case "markdown":
		fmt.Printf("# @%s\n\n%s\n", thread.Tweet.Author.ScreenName, thread.Tweet.Text)
		if len(thread.Replies) > 0 {
			fmt.Println()
			fmt.Println("## Replies")
			for _, reply := range thread.Replies {
				fmt.Printf("- @%s: %s\n", reply.Author.ScreenName, reply.Text)
			}
		}
		return nil
	default:
		if err := p.printTweetsTable([]model.Tweet{thread.Tweet}); err != nil {
			return err
		}
		if len(thread.Replies) > 0 {
			fmt.Println()
			fmt.Println("Replies")
			return p.printTweetsTable(thread.Replies)
		}
		return nil
	}
}

func (p *Printer) PrintUserProfile(user model.UserProfile) error {
	switch p.format {
	case "json":
		return p.printAny(user)
	case "markdown":
		fmt.Printf("# %s (@%s)\n\n", user.Name, user.ScreenName)
		if user.Bio != "" {
			fmt.Println(user.Bio)
			fmt.Println()
		}
		fmt.Printf("- Followers: %d\n", user.FollowersCount)
		fmt.Printf("- Following: %d\n", user.FollowingCount)
		fmt.Printf("- Posts: %d\n", user.TweetsCount)
		return nil
	default:
		tbl := table.New("Field", "Value").WithWriter(os.Stdout)
		p.styleHeader(tbl)
		tbl.AddRow("Name", user.Name)
		tbl.AddRow("Screen Name", "@"+user.ScreenName)
		tbl.AddRow("Followers", strconv.Itoa(user.FollowersCount))
		tbl.AddRow("Following", strconv.Itoa(user.FollowingCount))
		tbl.AddRow("Tweets", strconv.Itoa(user.TweetsCount))
		if user.Bio != "" {
			tbl.AddRow("Bio", user.Bio)
		}
		if user.Location != "" {
			tbl.AddRow("Location", user.Location)
		}
		if user.URL != "" {
			tbl.AddRow("URL", user.URL)
		}
		tbl.Print()
		return nil
	}
}

func (p *Printer) PrintUsers(users []model.UserProfile) error {
	switch p.format {
	case "json":
		return p.printAny(users)
	case "markdown":
		fmt.Println("# Users")
		fmt.Println()
		for _, user := range users {
			fmt.Printf("- @%s (%s)\n", user.ScreenName, user.Name)
		}
		return nil
	default:
		if len(users) == 0 {
			fmt.Println("No users found.")
			return nil
		}
		tbl := table.New("Screen Name", "Name", "Followers", "Following").WithWriter(os.Stdout)
		p.styleHeader(tbl)
		for _, user := range users {
			tbl.AddRow("@"+user.ScreenName, user.Name, user.FollowersCount, user.FollowingCount)
		}
		tbl.Print()
		return nil
	}
}

func (p *Printer) PrintUserSummary(user model.UserProfile, tweets []model.Tweet) error {
	summary := extractUserSummary(user, tweets)
	return p.printAny(summary)
}

type UserSummary struct {
	User         model.UserProfile `json:"user"`
	TweetCount   int               `json:"tweet_count"`
	TopHashtags  []string          `json:"top_hashtags"`
	TopMentions  []string          `json:"top_mentions"`
	AvgLikes     float64           `json:"avg_likes"`
	AvgRetweets  float64           `json:"avg_retweets"`
	AvgReplies   float64           `json:"avg_replies"`
	TopTopics    []string          `json:"top_topics"`
	RecentThemes []string          `json:"recent_themes"`
}

func extractUserSummary(user model.UserProfile, tweets []model.Tweet) UserSummary {
	summary := UserSummary{
		User:       user,
		TweetCount: len(tweets),
	}

	if len(tweets) == 0 {
		return summary
	}

	hashtagCounts := make(map[string]int)
	mentionCounts := make(map[string]int)
	wordCounts := make(map[string]int)
	totalLikes := 0
	totalRetweets := 0
	totalReplies := 0

	for _, tweet := range tweets {
		totalLikes += tweet.Metrics.Likes
		totalRetweets += tweet.Metrics.Retweets
		totalReplies += tweet.Metrics.Replies

		words := strings.Fields(strings.ToLower(tweet.Text))
		for _, word := range words {
			if strings.HasPrefix(word, "#") && len(word) > 1 {
				hashtagCounts[word]++
			} else if strings.HasPrefix(word, "@") && len(word) > 1 {
				mentionCounts[word]++
			} else if len(word) > 4 {
				wordCounts[word]++
			}
		}
	}

	summary.AvgLikes = float64(totalLikes) / float64(len(tweets))
	summary.AvgRetweets = float64(totalRetweets) / float64(len(tweets))
	summary.AvgReplies = float64(totalReplies) / float64(len(tweets))

	summary.TopHashtags = topKeys(hashtagCounts, 5)
	summary.TopMentions = topKeys(mentionCounts, 5)
	summary.TopTopics = topKeys(wordCounts, 10)

	if len(tweets) > 0 {
		summary.RecentThemes = extractThemes(tweets[:min(5, len(tweets))])
	}

	return summary
}

func extractThemes(tweets []model.Tweet) []string {
	themes := make(map[string]int)
	keywords := []string{"ai", "tech", "code", "build", "ship", "launch", "product", "startup", "data", "cloud", "security", "api", "open", "source", "learn", "write", "read", "think", "share", "grow"}

	for _, tweet := range tweets {
		text := strings.ToLower(tweet.Text)
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) {
				themes[keyword]++
			}
		}
	}

	return topKeys(themes, 5)
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
	for i := 0; i < min(n, len(pairs)); i++ {
		result = append(result, pairs[i].Key)
	}

	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (p *Printer) PrintActionResult(result model.ActionResult) error {
	switch p.format {
	case "json":
		return p.printAny(result)
	case "markdown":
		status := "failed"
		if result.Success {
			status = "ok"
		}
		fmt.Printf("- %s: %s", result.Action, status)
		if result.Target != "" {
			fmt.Printf(" (%s)", result.Target)
		}
		if result.URL != "" {
			fmt.Printf(" %s", result.URL)
		}
		if result.Message != "" {
			fmt.Printf(" - %s", result.Message)
		}
		fmt.Println()
		return nil
	default:
		status := "ok"
		if !result.Success {
			status = "failed"
		}
		tbl := table.New("Action", "Status", "Target", "Message").WithWriter(os.Stdout)
		p.styleHeader(tbl)
		tbl.AddRow(result.Action, status, result.Target, fallbackValue(result.URL, result.Message))
		tbl.Print()
		return nil
	}
}

func (p *Printer) PrintDoctor(report model.DoctorReport) error {
	switch p.format {
	case "json":
		return p.printAny(report)
	case "markdown":
		fmt.Printf("# %s\n\n", report.Name)
		for _, check := range report.Checks {
			fmt.Printf("- %s: %s", check.Name, check.Status)
			if check.Details != "" {
				fmt.Printf(" - %s", check.Details)
			}
			fmt.Println()
		}
		return nil
	default:
		tbl := table.New("Check", "Status", "Details").WithWriter(os.Stdout)
		p.styleHeader(tbl)
		for _, check := range report.Checks {
			tbl.AddRow(check.Name, check.Status, check.Details)
		}
		tbl.Print()
		return nil
	}
}

func (p *Printer) PrintSalts(samples []model.SaltSample) error {
	switch p.format {
	case "json":
		return p.printAny(samples)
	case "markdown":
		fmt.Println("# TXID Salt Samples")
		fmt.Println()
		for _, s := range samples {
			ts := time.UnixMilli(s.TimestampMS).UTC().Format("15:04:05.000")
			fmt.Printf("## %s @ %s\n\n", s.Operation, ts)
			fmt.Printf("- Salt: `%s`\n", s.Salt)
			if len(s.Salt) >= 16 {
				prefix := s.Salt[:16]
				suffix := s.Salt[len(s.Salt)-8:]
				fmt.Printf("- Pattern: `%s...%s` (%d bytes)\n", prefix, suffix, len(s.Salt)/2)
			}
			fmt.Println()
		}
		return nil
	default:
		if len(samples) == 0 {
			fmt.Println("No salt samples found.")
			return nil
		}
		tbl := table.New("Operation", "Time", "Salt (hex)", "Pattern").WithWriter(os.Stdout)
		p.styleHeader(tbl)
		for _, s := range samples {
			ts := time.UnixMilli(s.TimestampMS).UTC().Format("15:04:05")
			saltDisplay := s.Salt
			if len(saltDisplay) > 32 {
				saltDisplay = saltDisplay[:29] + "..."
			}
			pattern := ""
			if len(s.Salt) >= 16 {
				pattern = s.Salt[:8] + "..." + s.Salt[len(s.Salt)-8:]
			}
			tbl.AddRow(s.Operation, ts, saltDisplay, pattern)
		}
		tbl.Print()
		fmt.Printf("\nTotal unique operation+salt combinations: %d\n", len(samples))
		return nil
	}
}

func (p *Printer) printAny(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func (p *Printer) printTweetsTable(tweets []model.Tweet) error {
	if len(tweets) == 0 {
		fmt.Println("No posts found.")
		return nil
	}

	tbl := table.New("ID", "Author", "Text", "Likes", "RTs", "Replies").WithWriter(os.Stdout)
	p.styleHeader(tbl)

	for _, tweet := range tweets {
		tbl.AddRow(
			tweet.ID,
			"@"+tweet.Author.ScreenName,
			truncate(tweet.Text, 72),
			tweet.Metrics.Likes,
			tweet.Metrics.Retweets,
			tweet.Metrics.Replies,
		)
	}

	tbl.Print()
	return nil
}

func (p *Printer) styleHeader(tbl table.Table) {
	if !p.useColor {
		return
	}
	tbl.WithHeaderFormatter(func(format string, vals ...interface{}) string {
		return fmt.Sprintf("\033[1m%s\033[0m", fmt.Sprintf(format, vals...))
	})
}

func truncate(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}

func fallbackValue(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (p *Printer) PrintSaltComparisons(comparisons []model.SaltComparison) error {
	switch p.format {
	case "json":
		return p.printAny(comparisons)
	default:
		if len(comparisons) == 0 {
			fmt.Println("No comparisons to show.")
			return nil
		}

		fmt.Println("Salt Comparison Results (Constant Test)")
		fmt.Println("=========================================")
		fmt.Println()

		// Group by findings
		var sameSalts []model.SaltComparison
		var diffSalts []model.SaltComparison

		for _, c := range comparisons {
			if c.SaltMatch {
				sameSalts = append(sameSalts, c)
			} else {
				diffSalts = append(diffSalts, c)
			}
		}

		if len(sameSalts) > 0 {
			fmt.Printf("✓ Same Salt Pairs: %d\n", len(sameSalts))
			fmt.Println()
			for _, c := range sameSalts {
				opMarker := "↔"
				if c.SameOp {
					opMarker = "→"
				}
				fmt.Printf("  %s %s %s (gap: %ds)\n",
					c.SampleA.Operation,
					opMarker,
					c.SampleB.Operation,
					c.TimeGapSec)
			}
			fmt.Println()
		}

		if len(diffSalts) > 0 {
			fmt.Printf("✗ Different Salt Pairs: %d\n", len(diffSalts))
			fmt.Println()
			for _, c := range diffSalts {
				opMarker := "↔"
				if c.SameOp {
					opMarker = "→"
				}
				fmt.Printf("  %s %s %s (gap: %ds)\n",
					c.SampleA.Operation,
					opMarker,
					c.SampleB.Operation,
					c.TimeGapSec)
			}
			fmt.Println()
		}

		// Analysis summary
		totalPairs := len(comparisons)
		sameOpPairs := 0
		sameOpSameSalt := 0

		for _, c := range comparisons {
			if c.SameOp {
				sameOpPairs++
				if c.SaltMatch {
					sameOpSameSalt++
				}
			}
		}

		fmt.Println("Summary")
		fmt.Println("-------")
		fmt.Printf("Total pairs compared: %d\n", totalPairs)
		fmt.Printf("Same operation pairs: %d\n", sameOpPairs)

		if sameOpPairs > 0 {
			ratio := float64(sameOpSameSalt) / float64(sameOpPairs) * 100
			fmt.Printf("Same operation + same salt: %d (%.0f%%)\n", sameOpSameSalt, ratio)

			if ratio > 80 {
				fmt.Println()
				fmt.Println("📊 VERDICT: Session Constant (Scenario A)")
				fmt.Println("   The salt appears to be static across the session.")
				fmt.Println("   → Use SetStaticSalt() for native transport.")
			} else if ratio > 40 {
				fmt.Println()
				fmt.Println("📊 VERDICT: Time-Windowed (Scenario B)")
				fmt.Println("   The salt rotates periodically.")
				fmt.Println("   → Implement rotating salt generator.")
			} else {
				fmt.Println()
				fmt.Println("📊 VERDICT: Per-Request Nonce (Scenario C)")
				fmt.Println("   The salt changes with every request.")
				fmt.Println("   → Requires full JS reverse-engineering.")
			}
		}

		return nil
	}
}

func (p *Printer) PrintMentions(result any) error {
	return p.printAny(result)
}

func (p *Printer) PrintAny(value any) error {
	return p.printAny(value)
}
