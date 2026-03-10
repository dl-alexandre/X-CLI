package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/monitor"
)

type MentionsCmd struct {
	Watch    bool   `help:"Start continuous monitoring mode"`
	Since    string `help:"Show mentions from the last duration (e.g., '1h', '30m', '2d')"`
	Filter   string `help:"Filter mentions by keyword (comma-separated)"`
	FromUser string `help:"Filter mentions from specific users (comma-separated)" name:"from-user"`
	Notify   bool   `help:"Enable desktop notifications for new mentions"`
	Interval string `help:"Polling interval for watch mode (default: 60s)" default:"60s"`
	Reset    bool   `help:"Reset the monitor state"`
	Max      int    `help:"Maximum mentions to fetch" default:"50"`
}

func (c *MentionsCmd) Run(globals *Globals) error {
	if c.Reset {
		return c.resetMonitor()
	}

	opts := monitor.Options{
		Notify: c.Notify,
	}

	if c.Interval != "" {
		interval, err := monitor.ParseDuration(c.Interval)
		if err != nil {
			return fmt.Errorf("invalid interval: %w", err)
		}
		opts.PollInterval = interval
	}

	if c.Filter != "" || c.FromUser != "" {
		filter := &monitor.MentionFilter{}
		if c.Filter != "" {
			filter.Keywords = strings.Split(c.Filter, ",")
			for i, k := range filter.Keywords {
				filter.Keywords[i] = strings.TrimSpace(k)
			}
		}
		if c.FromUser != "" {
			filter.Users = strings.Split(c.FromUser, ",")
			for i, u := range filter.Users {
				filter.Users[i] = strings.TrimSpace(u)
			}
		}
		opts.Filter = filter
	}

	m, err := monitor.NewMonitor(globals.Client, opts)
	if err != nil {
		return fmt.Errorf("create monitor: %w", err)
	}

	if c.Watch {
		return c.watchMode(globals, m)
	}

	return c.fetchOnce(globals, m)
}

func (c *MentionsCmd) fetchOnce(globals *Globals, m *monitor.Monitor) error {
	var since time.Duration
	if c.Since != "" {
		var err error
		since, err = monitor.ParseDuration(c.Since)
		if err != nil {
			return fmt.Errorf("invalid since duration: %w", err)
		}
	} else {
		since = 24 * time.Hour
	}

	result, err := m.FetchMentions(since)
	if err != nil {
		return err
	}

	return globals.Printer("").PrintMentions(result)
}

func (c *MentionsCmd) watchMode(globals *Globals, m *monitor.Monitor) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nStopping monitor...")
		cancel()
	}()

	fmt.Printf("Starting mention monitor (interval: %s)...\n", c.Interval)
	fmt.Println("Press Ctrl+C to stop.")
	fmt.Println()

	return m.Watch(ctx, func(result monitor.MonitorResult) {
		if globals.Format == "json" {
			_ = globals.Printer("").PrintMentions(result)
		} else {
			for _, mention := range result.Mentions {
				if mention.IsNew {
					fmt.Printf("[NEW] @%s: %s\n", mention.Tweet.Author.ScreenName, truncateText(mention.Tweet.Text, 80))
				}
			}
			fmt.Printf("\n[%s] %d mentions (%d new)\n", result.LastChecked.Format("15:04:05"), result.TotalCount, result.NewCount)
		}
	})
}

func (c *MentionsCmd) resetMonitor() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	statePath := home + "/.config/x-cli/monitor.json"
	if err := os.Remove(statePath); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No monitor state to reset.")
			return nil
		}
		return fmt.Errorf("remove state file: %w", err)
	}

	fmt.Println("Monitor state reset.")
	return nil
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}
