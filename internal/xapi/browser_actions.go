package xapi

import (
	"context"
	"math"
	"net/http"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// browserStep is a chromedp action that returns no value.
type browserStep = chromedp.Action[chromedp.Void]

// enableNetwork sends Network.enable to the tab.
func enableNetwork() browserStep {
	return chromedp.Func(func(ctx context.Context, t *chromedp.Target) error {
		_, err := cdp.Call(ctx, t, network.Enable, network.EnableParams{})
		return err
	})
}

// addScriptOnNewDocument registers source to run in every new document of the
// tab before the page's own scripts.
func addScriptOnNewDocument(source string) browserStep {
	return chromedp.Func(func(ctx context.Context, t *chromedp.Target) error {
		_, err := cdp.Call(ctx, t, page.AddScriptToEvaluateOnNewDocument,
			page.AddScriptToEvaluateOnNewDocumentParams{Source: source})
		return err
	})
}

// setCookies installs the X session cookies in the browser.
func setCookies(cookies []*http.Cookie) browserStep {
	return chromedp.Func(func(ctx context.Context, t *chromedp.Target) error {
		return setBrowserCookies(ctx, t, cookies)
	})
}

// runScript evaluates expression and ignores its result.
func runScript(expression string) browserStep {
	return chromedp.Evaluate[chromedp.Void](expression)
}

// evaluateInto evaluates expression and stores the decoded result in dst.
func evaluateInto[T any](expression string, dst *T) browserStep {
	return chromedp.Func(func(ctx context.Context, t *chromedp.Target) error {
		v, err := chromedp.Evaluate[T](expression)(ctx, t)
		if err != nil {
			return err
		}
		*dst = v
		return nil
	})
}

// profileHrefInto stores the href of the profile link of the logged-in
// account in dst. dst stays unchanged when the link has no href.
func profileHrefInto(dst *string) browserStep {
	return chromedp.Func(func(ctx context.Context, t *chromedp.Target) error {
		res, err := chromedp.AttributeValue(chromedp.CSS(`[data-testid="AppTabBar_Profile_Link"]`), "href")(ctx, t)
		if err != nil {
			return err
		}
		if res.Exists {
			*dst = res.Value
		}
		return nil
	})
}

// formatWallTime formats a CDP wall time (seconds since the epoch) as an
// RFC 3339 UTC timestamp. It returns "" for an unset time.
func formatWallTime(t cdp.TimeSinceEpoch) string {
	secs := float64(t)
	if secs <= 0 || math.IsNaN(secs) || math.IsInf(secs, 0) {
		return ""
	}
	whole, frac := math.Modf(secs)
	return time.Unix(int64(whole), int64(frac*1e9)).UTC().Format(time.RFC3339Nano)
}
