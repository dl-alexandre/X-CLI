package xapi

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/dl-alexandre/X-CLI/internal/model"
)

func (c *Client) List(listID string, count int) (model.TimelineResult, error) {
	return c.browserTimeline("https://x.com/i/lists/"+url.PathEscape(listID), count, nil, "list")
}

func (c *Client) ListDetails(listID string) (model.List, error) {
	pageURL := "https://x.com/i/lists/" + url.PathEscape(listID)
	return c.browserList(pageURL, listID)
}

func (c *Client) ListMembers(listID string, count int) ([]model.UserProfile, error) {
	return c.browserUsers("https://x.com/i/lists/"+url.PathEscape(listID)+"/members", count)
}

func (c *Client) ListFollowers(listID string, count int) ([]model.UserProfile, error) {
	return c.browserUsers("https://x.com/i/lists/"+url.PathEscape(listID)+"/followers", count)
}

func (c *Client) browserList(pageURL string, listID string) (model.List, error) {
	if c.authSession == nil || len(c.authSession.Cookies) == 0 {
		return model.List{}, fmt.Errorf("authenticated browser session required")
	}

	session, err := c.newBrowserSession(45 * time.Second)
	if err != nil {
		return model.List{}, err
	}
	defer session.Close()

	var raw struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		BodyText    string `json:"body_text"`
		URL         string `json:"url"`
	}

	err = chromedp.Run(session.ctx,
		network.Enable(),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return setBrowserCookies(ctx, c.authSession.Cookies)
		}),
		chromedp.Navigate("https://x.com/home"),
		chromedp.Sleep(2*time.Second),
		chromedp.Navigate(pageURL),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`(() => {
			const text = (value) => String(value || '').trim();
			const meta = (name, attr = 'content') => {
				const node = document.querySelector(name);
				return text(node ? node.getAttribute(attr) : '');
			};
			const title = text(document.querySelector('main h1')?.innerText) || meta('meta[property="og:title"]') || text(document.title);
			const description = meta('meta[name="description"]') || meta('meta[property="og:description"]');
			return {
				title,
				description,
				body_text: text(document.body?.innerText),
				url: text(location.href),
			};
		})()`, &raw),
	)
	if err != nil {
		return model.List{}, fmt.Errorf("browser list failed: %w", err)
	}

	name := cleanListTitle(raw.Title)
	if name == "" {
		name = "List " + listID
	}

	description := strings.TrimSpace(raw.Description)
	if strings.EqualFold(description, raw.Title) {
		description = ""
	}

	return model.List{
		ID:             listID,
		Name:           name,
		Description:    description,
		URL:            fallbackString(raw.URL, pageURL),
		MembersCount:   extractListCount(raw.BodyText, "member"),
		FollowersCount: extractListCount(raw.BodyText, "follower"),
	}, nil
}

func cleanListTitle(value string) string {
	value = strings.TrimSpace(value)
	for _, suffix := range []string{" / X", " / Twitter", " on X"} {
		value = strings.TrimSuffix(value, suffix)
	}
	return strings.TrimSpace(value)
}

func extractListCount(body string, label string) int {
	body = strings.TrimSpace(body)
	if body == "" || label == "" {
		return 0
	}
	re := regexp.MustCompile(`(?i)([0-9][0-9,\.]*)\s+` + regexp.QuoteMeta(label) + `s?`)
	match := re.FindStringSubmatch(body)
	if len(match) < 2 {
		return 0
	}
	return parseHumanCount(match[1])
}

func parseHumanCount(value string) int {
	cleaned := strings.NewReplacer(",", "", ".", "").Replace(strings.TrimSpace(value))
	count, err := strconv.Atoi(cleaned)
	if err != nil {
		return 0
	}
	return count
}
