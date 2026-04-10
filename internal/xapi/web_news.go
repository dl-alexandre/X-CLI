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

func (c *Client) News(count int) ([]model.NewsItem, error) {
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
		Headline string `json:"headline"`
		Source   string `json:"source"`
		Meta     string `json:"meta"`
		URL      string `json:"url"`
	}

	err = chromedp.Run(session.ctx,
		network.Enable(),
		chromedp.ActionFunc(func(ctx context.Context) error { return setBrowserCookies(ctx, c.authSession.Cookies) }),
		chromedp.Navigate("https://x.com/home"),
		chromedp.Sleep(2*time.Second),
		chromedp.Navigate("https://x.com/explore/tabs/news"),
		chromedp.Sleep(4*time.Second),
		chromedp.Evaluate(`(() => {
			const clean = (value) => String(value || '').replace(/\s+/g, ' ').trim();
			const items = [];
			const seen = new Set();
			for (const link of document.querySelectorAll('a[href]')) {
				const href = clean(link.getAttribute('href'));
				if (!href || href.startsWith('/search?q=')) continue;
				const container = link.closest('article') || link.closest('[data-testid="cellInnerDiv"]') || link.closest('div[role="link"]') || link.parentElement;
				if (!container) continue;
				const pieces = Array.from(container.querySelectorAll('span, div[dir="auto"], time'))
					.map((node) => clean(node.innerText || node.getAttribute('datetime') || ''))
					.filter(Boolean);
				const headline = pieces.find((piece) => piece.length >= 24) || clean(link.innerText);
				if (!headline || seen.has(headline)) continue;
				seen.add(headline);
				const source = pieces[0] === headline ? '' : pieces[0] || '';
				const meta = pieces.filter((piece) => piece !== headline && piece !== source).slice(0, 2).join(' | ');
				items.push({
					headline,
					source,
					meta,
					url: new URL(href, location.origin).toString(),
				});
			}
			return items;
		})()`, &raw),
	)
	if err != nil {
		return nil, fmt.Errorf("browser news failed: %w", err)
	}

	items := make([]model.NewsItem, 0, len(raw))
	for _, item := range raw {
		items = append(items, model.NewsItem{
			Headline: cleanDMText(item.Headline),
			Source:   cleanDMText(item.Source),
			Meta:     cleanDMText(item.Meta),
			URL:      item.URL,
		})
	}
	if len(items) > count {
		items = items[:count]
	}
	return items, nil
}
