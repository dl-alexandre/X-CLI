package cli

import (
	"fmt"
	"strings"

	"github.com/dl-alexandre/X-CLI/internal/analytics"
	"github.com/dl-alexandre/X-CLI/internal/xapi"
)

type AnalyticsCmd struct {
	User      string `help:"User screen name for analytics" name:"user"`
	Tweet     string `help:"Post ID for analytics" name:"tweet"`
	Trending  bool   `help:"Show trending topics" name:"trending"`
	BestTimes bool   `help:"Show best posting times" name:"best-times"`
	Days      int    `help:"Number of days to analyze" default:"30" name:"days"`
	Export    string `help:"Export to file (json or csv)" name:"export"`
}

func (c *AnalyticsCmd) Run(globals *Globals) error {
	collector, err := analytics.NewAnalyticsCollector(globals.Client)
	if err != nil {
		return fmt.Errorf("create analytics collector: %w", err)
	}

	switch {
	case c.User != "":
		return c.runUserAnalytics(globals, collector)
	case c.Tweet != "":
		return c.runTweetAnalytics(globals, collector)
	case c.Trending:
		return c.runTrendingAnalytics(globals, collector)
	case c.BestTimes:
		return c.runBestTimesAnalytics(globals, collector)
	default:
		return fmt.Errorf("specify one of: --user, --post, --trending, or --best-times")
	}
}

func (c *AnalyticsCmd) runUserAnalytics(globals *Globals, collector *analytics.AnalyticsCollector) error {
	report, err := collector.CollectUserAnalytics(xapi.NormalizeScreenName(c.User), c.Days)
	if err != nil {
		return fmt.Errorf("collect user analytics: %w", err)
	}

	if c.Export != "" {
		if strings.HasSuffix(c.Export, ".json") {
			return collector.ExportToJSON(c.User, c.Export)
		} else if strings.HasSuffix(c.Export, ".csv") {
			return collector.ExportToCSV(c.User, c.Export)
		}
		return fmt.Errorf("export format must be .json or .csv")
	}

	return globals.Printer("").PrintAny(report)
}

func (c *AnalyticsCmd) runTweetAnalytics(globals *Globals, collector *analytics.AnalyticsCollector) error {
	report, err := collector.CollectTweetAnalytics(xapi.NormalizeTweetID(c.Tweet))
	if err != nil {
		return fmt.Errorf("collect post analytics: %w", err)
	}

	return globals.Printer("").PrintAny(report)
}

func (c *AnalyticsCmd) runTrendingAnalytics(globals *Globals, collector *analytics.AnalyticsCollector) error {
	if c.User == "" {
		return fmt.Errorf("--user is required for trending topics")
	}

	topics, err := collector.GetTrendingTopics(xapi.NormalizeScreenName(c.User), c.Days)
	if err != nil {
		return fmt.Errorf("get trending topics: %w", err)
	}

	return globals.Printer("").PrintAny(topics)
}

func (c *AnalyticsCmd) runBestTimesAnalytics(globals *Globals, collector *analytics.AnalyticsCollector) error {
	if c.User == "" {
		return fmt.Errorf("--user is required for best posting times")
	}

	times, err := collector.GetBestPostingTimes(xapi.NormalizeScreenName(c.User))
	if err != nil {
		return fmt.Errorf("get best posting times: %w", err)
	}

	return globals.Printer("").PrintAny(times)
}
