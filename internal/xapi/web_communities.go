package xapi

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/dl-alexandre/X-CLI/internal/model"
)

func (c *Client) CommunityPosts(communityID string, count int) (model.TimelineResult, error) {
	return c.browserTimeline("https://x.com/i/communities/"+url.PathEscape(communityID), count, nil, "community")
}

func (c *Client) CommunityDetails(communityID string) (model.Community, error) {
	pageURL := "https://x.com/i/communities/" + url.PathEscape(communityID)
	return c.browserCommunity(pageURL, communityID)
}

func (c *Client) browserCommunity(pageURL string, communityID string) (model.Community, error) {
	if c.authSession == nil || len(c.authSession.Cookies) == 0 {
		return model.Community{}, fmt.Errorf("authenticated browser session required")
	}

	session, err := c.newBrowserSession(45 * time.Second)
	if err != nil {
		return model.Community{}, err
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
		chromedp.ActionFunc(func(ctx context.Context) error { return setBrowserCookies(ctx, c.authSession.Cookies) }),
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
		return model.Community{}, fmt.Errorf("browser community failed: %w", err)
	}

	name := cleanCommunityTitle(raw.Title)
	if name == "" {
		name = "Community " + communityID
	}

	description := strings.TrimSpace(raw.Description)
	if strings.EqualFold(description, raw.Title) {
		description = ""
	}

	return model.Community{
		ID:           communityID,
		Name:         name,
		Description:  description,
		URL:          fallbackString(raw.URL, pageURL),
		MembersCount: extractListCount(raw.BodyText, "member"),
		RulesCount:   extractListCount(raw.BodyText, "rule"),
	}, nil
}

func cleanCommunityTitle(value string) string {
	value = strings.TrimSpace(value)
	for _, suffix := range []string{" / X", " / Twitter", " on X", " Community"} {
		value = strings.TrimSuffix(value, suffix)
	}
	return strings.TrimSpace(value)
}
