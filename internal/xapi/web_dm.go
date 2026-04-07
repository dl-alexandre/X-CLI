package xapi

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/dl-alexandre/X-CLI/internal/model"
)

func (c *Client) DMInbox(count int) (model.DMInbox, error) {
	if c.authSession == nil || len(c.authSession.Cookies) == 0 {
		return model.DMInbox{}, errors.New("authenticated browser session required")
	}
	if count <= 0 {
		count = 20
	}

	session, err := c.newBrowserSession(45 * time.Second)
	if err != nil {
		return model.DMInbox{}, err
	}
	defer session.Close()

	var raw []struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Preview     string `json:"preview"`
		URL         string `json:"url"`
		UpdatedAt   string `json:"updated_at"`
		Unread      bool   `json:"unread"`
		Participant string `json:"participant"`
	}

	err = chromedp.Run(session.ctx,
		network.Enable(),
		chromedp.ActionFunc(func(ctx context.Context) error { return setBrowserCookies(ctx, c.authSession.Cookies) }),
		chromedp.Navigate("https://x.com/home"),
		chromedp.Sleep(2*time.Second),
		chromedp.Navigate("https://x.com/messages"),
		chromedp.Sleep(4*time.Second),
		chromedp.Evaluate(`(() => {
			const clean = (value) => String(value || '').replace(/\s+/g, ' ').trim();
			const ignored = new Set(['Messages', 'New message', 'Primary', 'Requests', 'Search Direct Messages', 'Search messages']);
			const items = [];
			const seen = new Set();
			for (const link of document.querySelectorAll('a[href*="/messages/"]')) {
				const href = clean(link.getAttribute('href'));
				const match = href.match(/\/messages\/([^\/?#]+)/);
				if (!match) continue;
				const id = match[1];
				if (!id || id === 'compose' || seen.has(id)) continue;
				const container = link.closest('[data-testid="cellInnerDiv"]') || link.closest('a') || link.parentElement;
				const pieces = Array.from((container || link).querySelectorAll('span, div[dir="auto"], time'))
					.map((node) => clean(node.innerText || node.getAttribute('datetime') || ''))
					.filter(Boolean)
					.filter((text) => !ignored.has(text));
				const unique = [];
				for (const piece of pieces) {
					if (!unique.includes(piece)) unique.push(piece);
				}
				const title = clean(unique[0] || link.innerText || id);
				const preview = clean(unique.slice(1).join(' ').slice(0, 280));
				const updatedAt = clean((container || link).querySelector('time')?.getAttribute('datetime') || '');
				const unread = /unread|new message/i.test(clean((container || link).innerText || ''));
				let participant = '';
				for (const candidate of unique) {
					if (candidate && candidate !== title && !/^\d+[smhdw]$/.test(candidate)) {
						participant = candidate;
						break;
					}
				}
				seen.add(id);
				items.push({
					id,
					title,
					preview,
					url: new URL(href, location.origin).toString(),
					updated_at: updatedAt,
					unread: unread,
					participant,
				});
			}
			return items;
		})()`, &raw),
	)
	if err != nil {
		return model.DMInbox{}, fmt.Errorf("browser dm inbox failed: %w", err)
	}

	conversations := make([]model.DMConversation, 0, len(raw))
	for _, item := range raw {
		conversations = append(conversations, model.DMConversation{
			ID:          item.ID,
			Title:       cleanDMText(item.Title),
			Preview:     cleanDMText(item.Preview),
			URL:         item.URL,
			UpdatedAt:   item.UpdatedAt,
			Unread:      item.Unread,
			Participant: cleanDMText(item.Participant),
		})
	}

	if len(conversations) > count {
		conversations = conversations[:count]
	}

	return model.DMInbox{Conversations: conversations}, nil
}

func (c *Client) DMThread(conversationID string, count int) (model.DMThread, error) {
	if c.authSession == nil || len(c.authSession.Cookies) == 0 {
		return model.DMThread{}, errors.New("authenticated browser session required")
	}
	conversationID = normalizeConversationID(conversationID)
	if conversationID == "" {
		return model.DMThread{}, errors.New("conversation ID is required")
	}
	if count <= 0 {
		count = 50
	}

	pageURL := "https://x.com/messages/" + url.PathEscape(conversationID)
	session, err := c.newBrowserSession(60 * time.Second)
	if err != nil {
		return model.DMThread{}, err
	}
	defer session.Close()

	var raw struct {
		Title    string `json:"title"`
		Messages []struct {
			Text      string `json:"text"`
			CreatedAt string `json:"created_at"`
			Sender    string `json:"sender"`
			Outgoing  bool   `json:"outgoing"`
		} `json:"messages"`
	}

	err = chromedp.Run(session.ctx,
		network.Enable(),
		chromedp.ActionFunc(func(ctx context.Context) error { return setBrowserCookies(ctx, c.authSession.Cookies) }),
		chromedp.Navigate("https://x.com/home"),
		chromedp.Sleep(2*time.Second),
		chromedp.Navigate(pageURL),
		chromedp.Sleep(5*time.Second),
		chromedp.Evaluate(`(() => {
			const clean = (value) => String(value || '').replace(/\s+/g, ' ').trim();
			const ignored = new Set(['Messages', 'Message info', 'Start a new message', 'Write a message']);
			const title = clean(document.querySelector('main h2')?.innerText || document.querySelector('main h1')?.innerText || '');
			const items = [];
			const seen = new Set();
			const roots = document.querySelectorAll('main [data-testid="cellInnerDiv"], main article, main section > div > div');
			for (const root of roots) {
				const textNodes = Array.from(root.querySelectorAll('div[dir="auto"], span, p'))
					.map((node) => clean(node.innerText))
					.filter(Boolean)
					.filter((text) => !ignored.has(text));
				const unique = [];
				for (const piece of textNodes) {
					if (!unique.includes(piece)) unique.push(piece);
				}
				if (unique.length === 0) continue;
				const createdAt = clean(root.querySelector('time')?.getAttribute('datetime') || '');
				let sender = '';
				for (const piece of unique) {
					if (piece.startsWith('@')) { sender = piece; break; }
				}
				const text = clean(unique.filter((piece) => piece !== sender && piece !== createdAt).join(' '));
				if (!text) continue;
				const key = text + '|' + createdAt;
				if (seen.has(key)) continue;
				seen.add(key);
				const style = window.getComputedStyle(root);
				const outgoing = style.justifyContent === 'flex-end' || style.alignItems === 'flex-end' || /You sent/i.test(clean(root.innerText));
				items.push({ text, created_at: createdAt, sender, outgoing });
			}
			return { title, messages: items };
		})()`, &raw),
	)
	if err != nil {
		return model.DMThread{}, fmt.Errorf("browser dm thread failed: %w", err)
	}

	messages := make([]model.DirectMessage, 0, len(raw.Messages))
	for _, item := range raw.Messages {
		messages = append(messages, model.DirectMessage{
			Text:      cleanDMText(item.Text),
			CreatedAt: item.CreatedAt,
			Sender:    cleanDMText(item.Sender),
			Outgoing:  item.Outgoing,
		})
	}

	sort.SliceStable(messages, func(i, j int) bool {
		return messageSortKey(messages[i]) < messageSortKey(messages[j])
	})
	if len(messages) > count {
		messages = messages[len(messages)-count:]
	}

	return model.DMThread{
		Conversation: model.DMConversation{
			ID:    conversationID,
			Title: cleanDMText(raw.Title),
			URL:   pageURL,
		},
		Messages: messages,
	}, nil
}

func normalizeConversationID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "/messages/") {
		if parsed, err := url.Parse(value); err == nil {
			parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
			for i := 0; i < len(parts)-1; i++ {
				if parts[i] == "messages" {
					return parts[i+1]
				}
			}
		}
	}
	return strings.Trim(strings.TrimPrefix(value, "/messages/"), "/")
}

func cleanDMText(value string) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	return strings.Trim(value, "| ")
}

func messageSortKey(msg model.DirectMessage) string {
	if msg.CreatedAt != "" {
		return msg.CreatedAt + "|" + msg.Text
	}
	return "zzzz|" + msg.Text
}
