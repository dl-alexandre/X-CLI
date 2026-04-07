package xapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/dl-alexandre/X-CLI/internal/model"
)

func (c *Client) Trends(count int) ([]model.Trend, error) {
	if c.authSession == nil || len(c.authSession.Cookies) == 0 {
		return nil, errors.New("authenticated browser session required")
	}
	if count <= 0 {
		count = 20
	}

	session, err := c.newBrowserSession(45 * time.Second)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	var raw []struct {
		Name     string `json:"name"`
		Category string `json:"category"`
		Posts    string `json:"posts"`
		URL      string `json:"url"`
	}

	err = chromedp.Run(session.ctx,
		network.Enable(),
		chromedp.ActionFunc(func(ctx context.Context) error { return setBrowserCookies(ctx, c.authSession.Cookies) }),
		chromedp.Navigate("https://x.com/home"),
		chromedp.Sleep(2*time.Second),
		chromedp.Navigate("https://x.com/explore/tabs/trending"),
		chromedp.Sleep(4*time.Second),
		chromedp.Evaluate(`(() => {
			const clean = (value) => String(value || '').replace(/\s+/g, ' ').trim();
			const items = [];
			const seen = new Set();
			for (const link of document.querySelectorAll('a[href^="/search?q="]')) {
				const href = clean(link.getAttribute('href'));
				const container = link.closest('[data-testid="trend"]') || link.closest('div[role="link"]') || link.parentElement;
				const pieces = Array.from((container || link).querySelectorAll('span, div[dir="auto"]'))
					.map((node) => clean(node.innerText))
					.filter(Boolean);
				const filtered = pieces.filter((piece) => !/^Show more$/i.test(piece));
				const name = filtered.find((piece) => piece.startsWith('#') || piece.length > 1) || clean(link.innerText);
				if (!name || seen.has(name)) continue;
				seen.add(name);
				const category = filtered[0] === name ? '' : filtered[0] || '';
				const posts = filtered.find((piece) => /posts?$|k posts|m posts/i.test(piece)) || '';
				items.push({
					name,
					category,
					posts,
					url: new URL(href, location.origin).toString(),
				});
			}
			return items;
		})()`, &raw),
	)
	if err != nil {
		return nil, fmt.Errorf("browser trends failed: %w", err)
	}

	trends := make([]model.Trend, 0, len(raw))
	for _, item := range raw {
		trends = append(trends, model.Trend{
			Name:     cleanDMText(item.Name),
			Category: cleanDMText(item.Category),
			Posts:    cleanDMText(item.Posts),
			URL:      item.URL,
		})
	}
	if len(trends) > count {
		trends = trends[:count]
	}
	return trends, nil
}
