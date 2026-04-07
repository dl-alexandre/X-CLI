package xapi

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/dl-alexandre/X-CLI/internal/model"
)

func (c *Client) SpaceDetails(spaceID string) (model.Space, error) {
	if c.authSession == nil || len(c.authSession.Cookies) == 0 {
		return model.Space{}, errors.New("authenticated browser session required")
	}
	spaceID = normalizeSpaceID(spaceID)
	if spaceID == "" {
		return model.Space{}, errors.New("space ID is required")
	}

	pageURL := "https://x.com/i/spaces/" + url.PathEscape(spaceID)
	session, err := c.newBrowserSession(45 * time.Second)
	if err != nil {
		return model.Space{}, err
	}
	defer session.Close()

	var raw struct {
		Title       string `json:"title"`
		Host        string `json:"host"`
		State       string `json:"state"`
		ScheduledAt string `json:"scheduled_at"`
		URL         string `json:"url"`
	}

	err = chromedp.Run(session.ctx,
		network.Enable(),
		chromedp.ActionFunc(func(ctx context.Context) error { return setBrowserCookies(ctx, c.authSession.Cookies) }),
		chromedp.Navigate("https://x.com/home"),
		chromedp.Sleep(2*time.Second),
		chromedp.Navigate(pageURL),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`(() => {
			const clean = (value) => String(value || '').replace(/\s+/g, ' ').trim();
			const textNodes = Array.from(document.querySelectorAll('main h1, main h2, main span, main div[dir="auto"], main time'))
				.map((node) => clean(node.innerText || node.getAttribute('datetime') || ''))
				.filter(Boolean);
			const title = clean(document.querySelector('main h1')?.innerText || document.querySelector('meta[property="og:title"]')?.getAttribute('content') || document.title);
			let host = '';
			let state = '';
			for (const text of textNodes) {
				if (!host && text.startsWith('@')) host = text;
				if (!state && /live|scheduled|ended|recorded/i.test(text)) state = text;
			}
			return {
				title,
				host,
				state,
				scheduled_at: clean(document.querySelector('main time')?.getAttribute('datetime') || ''),
				url: clean(location.href),
			};
		})()`, &raw),
	)
	if err != nil {
		return model.Space{}, fmt.Errorf("browser space failed: %w", err)
	}

	title := cleanSpaceTitle(raw.Title)
	if title == "" {
		title = "Space " + spaceID
	}

	return model.Space{
		ID:          spaceID,
		Title:       title,
		Host:        cleanDMText(raw.Host),
		State:       strings.ToLower(cleanDMText(raw.State)),
		ScheduledAt: raw.ScheduledAt,
		URL:         fallbackString(raw.URL, pageURL),
	}, nil
}

func normalizeSpaceID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "/spaces/") {
		if parsed, err := url.Parse(value); err == nil {
			parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
			for i := 0; i < len(parts)-1; i++ {
				if parts[i] == "spaces" {
					return parts[i+1]
				}
			}
		}
	}
	return strings.Trim(strings.TrimPrefix(value, "/spaces/"), "/")
}

func cleanSpaceTitle(value string) string {
	value = strings.TrimSpace(value)
	for _, suffix := range []string{" / X", " on X", " / Twitter"} {
		value = strings.TrimSuffix(value, suffix)
	}
	return strings.TrimSpace(value)
}
