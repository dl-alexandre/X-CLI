package xapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type WebCaptureOptions struct {
	URL           string
	Duration      time.Duration
	Filter        string
	IncludeBodies bool
}

type CapturedWebOperation struct {
	Method       string `json:"method"`
	URL          string `json:"url"`
	Path         string `json:"path,omitempty"`
	Operation    string `json:"operation,omitempty"`
	QueryID      string `json:"query_id,omitempty"`
	Status       int    `json:"status,omitempty"`
	ContentType  string `json:"content_type,omitempty"`
	RequestBody  string `json:"request_body,omitempty"`
	ResponseBody string `json:"response_body,omitempty"`
	TXID         string `json:"txid,omitempty"`
	TimestampMS  int64  `json:"timestamp_ms"`
}

type WebCaptureResult struct {
	URL        string                 `json:"url"`
	DurationMS int64                  `json:"duration_ms"`
	Filter     string                 `json:"filter,omitempty"`
	CapturedAt string                 `json:"captured_at"`
	Operations []CapturedWebOperation `json:"operations"`
}

const webOperationCaptureScript = `(function() {
  if (window.__xcliWebCaptureInstalled) return;
  window.__xcliWebCaptureInstalled = true;
  window.__xcliWebOps = window.__xcliWebOps || [];

  const shouldCapture = (url) => String(url || '').includes('/i/api/');

  const bodyToString = (value) => {
    try {
      if (value == null) return '';
      if (typeof value === 'string') return value;
      if (typeof URLSearchParams !== 'undefined' && value instanceof URLSearchParams) return value.toString();
      if (typeof FormData !== 'undefined' && value instanceof FormData) {
        const out = [];
        value.forEach((v, k) => out.push(String(k) + '=' + String(v)));
        return out.join('&');
      }
      if (typeof value === 'object') return JSON.stringify(value);
      return String(value);
    } catch (_) {
      return '';
    }
  };

  const toHeaderMap = (value) => {
    const out = {};
    if (!value) return out;
    try {
      if (Array.isArray(value)) {
        for (const pair of value) {
          if (Array.isArray(pair) && pair.length >= 2) out[String(pair[0])] = String(pair[1]);
        }
        return out;
      }
      if (typeof Headers !== 'undefined' && value instanceof Headers) {
        value.forEach((v, k) => { out[String(k)] = String(v); });
        return out;
      }
      if (typeof value === 'object') {
        for (const [k, v] of Object.entries(value)) out[String(k)] = String(v);
      }
    } catch (_) {}
    return out;
  };

  const pushRecord = (record) => {
    try {
      if (!shouldCapture(record && record.url)) return;
      window.__xcliWebOps.push({
        method: String((record && record.method) || 'GET'),
        url: String((record && record.url) || ''),
        status: Number((record && record.status) || 0),
        contentType: String((record && record.contentType) || ''),
        requestBody: bodyToString(record && record.requestBody),
        responseBody: bodyToString(record && record.responseBody),
        txid: String((record && record.txid) || ''),
        ts: Number((record && record.ts) || Date.now()),
      });
    } catch (_) {}
  };

  const originalFetch = window.fetch;
  window.fetch = async function(input, init) {
    const url = typeof input === 'string' ? input : (input && input.url) || '';
    const method = (init && init.method) || (input && input.method) || 'GET';
    const headers = toHeaderMap((init && init.headers) || (input && input.headers));
    const txid = headers['x-client-transaction-id'] || headers['X-Client-Transaction-Id'] || '';
    const requestBody = (init && init.body) || (input && input.body) || '';
    const response = await originalFetch.apply(this, arguments);
    try {
      const clone = response.clone();
      const responseBody = await clone.text().catch(() => '');
      const contentType = clone.headers.get('content-type') || '';
      pushRecord({ url, method, status: clone.status, contentType, requestBody, responseBody, txid, ts: Date.now() });
    } catch (_) {}
    return response;
  };

  const originalOpen = XMLHttpRequest.prototype.open;
  const originalSend = XMLHttpRequest.prototype.send;
  const originalSetRequestHeader = XMLHttpRequest.prototype.setRequestHeader;

  XMLHttpRequest.prototype.open = function(method, url) {
    this.__xcliMethod = method || 'GET';
    this.__xcliURL = url || '';
    this.__xcliHeaders = this.__xcliHeaders || {};
    return originalOpen.apply(this, arguments);
  };

  XMLHttpRequest.prototype.setRequestHeader = function(name, value) {
    try {
      this.__xcliHeaders = this.__xcliHeaders || {};
      this.__xcliHeaders[String(name)] = String(value);
    } catch (_) {}
    return originalSetRequestHeader.apply(this, arguments);
  };

  XMLHttpRequest.prototype.send = function(body) {
    this.__xcliRequestBody = bodyToString(body);
    this.addEventListener('loadend', function() {
      try {
        const headers = this.__xcliHeaders || {};
        const txid = headers['x-client-transaction-id'] || headers['X-Client-Transaction-Id'] || '';
        pushRecord({
          url: this.__xcliURL || this.responseURL || '',
          method: this.__xcliMethod || 'GET',
          status: this.status || 0,
          contentType: this.getResponseHeader('content-type') || '',
          requestBody: this.__xcliRequestBody || '',
          responseBody: this.responseText || '',
          txid,
          ts: Date.now(),
        });
      } catch (_) {}
    });
    return originalSend.apply(this, arguments);
  };
})();`

func (c *Client) CaptureWebOperations(opts WebCaptureOptions) (WebCaptureResult, error) {
	if c.authSession == nil || len(c.authSession.Cookies) == 0 {
		return WebCaptureResult{}, errors.New("authenticated browser session required")
	}

	if opts.Duration <= 0 {
		opts.Duration = 20 * time.Second
	}
	if strings.TrimSpace(opts.URL) == "" {
		opts.URL = "https://x.com/home"
	}

	session, err := c.newBrowserSession(opts.Duration + 30*time.Second)
	if err != nil {
		return WebCaptureResult{}, err
	}
	defer session.Close()

	err = chromedp.Run(session.ctx,
		network.Enable(),
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(webOperationCaptureScript).Do(ctx)
			return err
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return setBrowserCookies(ctx, c.authSession.Cookies)
		}),
		chromedp.Navigate("https://x.com/home"),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(webOperationCaptureScript, nil),
		chromedp.Navigate(opts.URL),
		chromedp.Sleep(2*time.Second),
		chromedp.ActionFunc(func(context.Context) error {
			time.Sleep(opts.Duration)
			return nil
		}),
	)
	if err != nil {
		return WebCaptureResult{}, err
	}

	var rawRecords []map[string]any
	if err := chromedp.Run(session.ctx,
		chromedp.Evaluate(`(() => window.__xcliWebOps || [])()`, &rawRecords),
	); err != nil {
		return WebCaptureResult{}, err
	}

	operations := make([]CapturedWebOperation, 0, len(rawRecords))
	for _, record := range rawRecords {
		op := capturedWebOperationFromRecord(record, opts.IncludeBodies)
		if !matchesCapturedWebFilter(op, opts.Filter) {
			continue
		}
		operations = append(operations, op)
	}

	sort.Slice(operations, func(i, j int) bool {
		return operations[i].TimestampMS < operations[j].TimestampMS
	})

	return WebCaptureResult{
		URL:        opts.URL,
		DurationMS: opts.Duration.Milliseconds(),
		Filter:     strings.TrimSpace(opts.Filter),
		CapturedAt: time.Now().UTC().Format(time.RFC3339),
		Operations: operations,
	}, nil
}

func capturedWebOperationFromRecord(record map[string]any, includeBodies bool) CapturedWebOperation {
	urlString := stringValue(record["url"])
	queryID, operation, path := parseCapturedWebURL(urlString)

	op := CapturedWebOperation{
		Method:      strings.ToUpper(fallbackString(stringValue(record["method"]), http.MethodGet)),
		URL:         urlString,
		Path:        path,
		Operation:   operation,
		QueryID:     queryID,
		Status:      intValue(record["status"]),
		ContentType: stringValue(record["contentType"]),
		TXID:        stringValue(record["txid"]),
		TimestampMS: int64Value(record["ts"]),
	}
	if includeBodies {
		op.RequestBody = stringValue(record["requestBody"])
		op.ResponseBody = stringValue(record["responseBody"])
	}
	return op
}

func parseCapturedWebURL(raw string) (queryID string, operation string, path string) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", "", ""
	}

	path = parsed.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i := 0; i+3 < len(parts); i++ {
		if parts[i] == "i" && parts[i+1] == "api" && parts[i+2] == "graphql" {
			return parts[i+3], strings.Join(parts[i+4:], "/"), path
		}
	}

	return "", "", path
}

func matchesCapturedWebFilter(op CapturedWebOperation, filter string) bool {
	filter = strings.TrimSpace(strings.ToLower(filter))
	if filter == "" {
		return true
	}
	return strings.Contains(strings.ToLower(op.URL), filter) ||
		strings.Contains(strings.ToLower(op.Path), filter) ||
		strings.Contains(strings.ToLower(op.Operation), filter)
}
