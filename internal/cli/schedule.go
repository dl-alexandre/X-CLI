package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/schedule"
)

type ScheduleCmd struct {
	Save   ScheduleSaveCmd   `cmd:"" help:"Schedule a post for later posting"`
	List   ScheduleListCmd   `cmd:"" help:"List all scheduled posts"`
	Cancel ScheduleCancelCmd `cmd:"" help:"Cancel a scheduled post"`
	Post   SchedulePostCmd   `cmd:"" help:"Post a scheduled post immediately"`
}

type ScheduleSaveCmd struct {
	Text string `arg:"" help:"Post text to schedule"`
	Time string `help:"Scheduled time (e.g., '2024-01-15 09:00', 'tomorrow at 9am', 'in 2 hours')"`
}

func (c *ScheduleSaveCmd) Run(globals *Globals) error {
	store, err := schedule.NewScheduleStore()
	if err != nil {
		return fmt.Errorf("schedule store: %w", err)
	}

	scheduledTime, err := schedule.ParseTime(c.Time)
	if err != nil {
		return fmt.Errorf("parse time: %w", err)
	}

	tweet, err := store.Schedule(c.Text, scheduledTime)
	if err != nil {
		return fmt.Errorf("schedule post: %w", err)
	}

	fmt.Printf("Tweet scheduled successfully!\n")
	fmt.Printf("  ID: %s\n", tweet.ID)
	fmt.Printf("  Scheduled for: %s\n", tweet.Scheduled.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Time until posting: %s\n", schedule.FormatTimeRemaining(tweet.Scheduled))
	return nil
}

type ScheduleListCmd struct {
	All bool `help:"Show all scheduled posts including posted and cancelled"`
}

func (c *ScheduleListCmd) Run(globals *Globals) error {
	store, err := schedule.NewScheduleStore()
	if err != nil {
		return fmt.Errorf("schedule store: %w", err)
	}

	var tweets []schedule.ScheduledTweet
	if c.All {
		tweets = store.List()
	} else {
		tweets = store.ListPending()
	}

	if len(tweets) == 0 {
		fmt.Println("No scheduled posts.")
		return nil
	}

	fmt.Printf("Scheduled posts (%d):\n\n", len(tweets))
	for i, tweet := range tweets {
		preview := tweet.Text
		if len(preview) > 50 {
			preview = preview[:47] + "..."
		}

		statusIcon := "⏳"
		switch tweet.Status {
		case "posted":
			statusIcon = "✓"
		case "cancelled":
			statusIcon = "✗"
		}

		fmt.Printf("%d. [%s] %s %s\n", i+1, tweet.ID, statusIcon, preview)
		fmt.Printf("   Scheduled: %s", tweet.Scheduled.Format("2006-01-02 15:04"))
		if tweet.Status == "pending" {
			fmt.Printf(" (%s)", schedule.FormatTimeRemaining(tweet.Scheduled))
		} else {
			fmt.Printf(" [%s]", tweet.Status)
		}
		fmt.Println()
		fmt.Printf("   Created: %s\n\n", tweet.CreatedAt.Format("2006-01-02 15:04"))
	}

	return nil
}

type ScheduleCancelCmd struct {
	ID string `arg:"" help:"ID of the scheduled post to cancel"`
}

func (c *ScheduleCancelCmd) Run(globals *Globals) error {
	store, err := schedule.NewScheduleStore()
	if err != nil {
		return fmt.Errorf("schedule store: %w", err)
	}

	tweet, err := store.Get(c.ID)
	if err != nil {
		return fmt.Errorf("get scheduled post: %w", err)
	}

	if err := store.Cancel(c.ID); err != nil {
		return fmt.Errorf("cancel scheduled post: %w", err)
	}

	fmt.Printf("Scheduled post cancelled: %s\n", c.ID)
	fmt.Printf("  Text: %s\n", truncateScheduleText(tweet.Text, 60))
	fmt.Printf("  Was scheduled for: %s\n", tweet.Scheduled.Format("2006-01-02 15:04"))
	return nil
}

type SchedulePostCmd struct {
	ID string `arg:"" help:"ID of the scheduled post to post now"`
}

func (c *SchedulePostCmd) Run(globals *Globals) error {
	store, err := schedule.NewScheduleStore()
	if err != nil {
		return fmt.Errorf("schedule store: %w", err)
	}

	tweet, err := store.Get(c.ID)
	if err != nil {
		return fmt.Errorf("get scheduled post: %w", err)
	}

	if tweet.Status != "pending" {
		return fmt.Errorf("cannot post with status '%s'", tweet.Status)
	}

	result, err := globals.Client.CreatePost(tweet.Text)
	if err != nil {
		return fmt.Errorf("post scheduled: %w", err)
	}

	if err := store.MarkPosted(c.ID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to mark post as posted: %v\n", err)
	}

	fmt.Printf("Scheduled tweet posted successfully!\n")
	fmt.Printf("  ID: %s\n", c.ID)
	return globals.Printer("").PrintActionResult(result)
}

func truncateScheduleText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}

func RunScheduledTweets(client interface {
	CreatePost(text string) (interface{}, error)
}) error {
	store, err := schedule.NewScheduleStore()
	if err != nil {
		return err
	}

	due := store.GetDue()
	if len(due) == 0 {
		return nil
	}

	for _, tweet := range due {
		_, err := client.CreatePost(tweet.Text)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to post scheduled post %s: %v\n", tweet.ID, err)
			continue
		}

		if err := store.MarkPosted(tweet.ID); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to mark post %s as posted: %v\n", tweet.ID, err)
		}

		time.Sleep(2 * time.Second)
	}

	return nil
}
