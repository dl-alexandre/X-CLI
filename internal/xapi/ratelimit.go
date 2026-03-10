package xapi

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/config"
)

type RateLimitInfo struct {
	Limit     int
	Remaining int
	Reset     time.Time
	ResetAt   int64
}

type RateLimitHandler struct {
	config    *config.Config
	verbose   bool
	lastLimit *RateLimitInfo
}

func NewRateLimitHandler(cfg *config.Config, verbose bool) *RateLimitHandler {
	return &RateLimitHandler{
		config:  cfg,
		verbose: verbose,
	}
}

func (h *RateLimitHandler) ParseRateLimitHeaders(headers map[string][]string) *RateLimitInfo {
	info := &RateLimitInfo{}

	if limit := firstHeader(headers, "x-rate-limit-limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			info.Limit = val
		}
	}

	if remaining := firstHeader(headers, "x-rate-limit-remaining"); remaining != "" {
		if val, err := strconv.Atoi(remaining); err == nil {
			info.Remaining = val
		}
	}

	if reset := firstHeader(headers, "x-rate-limit-reset"); reset != "" {
		if val, err := strconv.ParseInt(reset, 10, 64); err == nil {
			info.ResetAt = val
			info.Reset = time.Unix(val, 0)
		}
	}

	if info.Limit > 0 || info.Remaining > 0 || !info.Reset.IsZero() {
		h.lastLimit = info
		return info
	}

	return nil
}

func (h *RateLimitHandler) ShouldRetry(statusCode int) bool {
	return statusCode == 429
}

func (h *RateLimitHandler) GetWaitDuration(info *RateLimitInfo) time.Duration {
	if info == nil || info.Reset.IsZero() {
		if h.config != nil {
			return h.config.RateLimit.RetryBaseDelay
		}
		return 5 * time.Second
	}

	wait := time.Until(info.Reset)
	if wait < 0 {
		wait = 1 * time.Second
	}

	return wait
}

func (h *RateLimitHandler) ShowCountdown(duration time.Duration, reason string) {
	if duration <= 0 {
		return
	}

	remaining := int(duration.Seconds())
	if remaining <= 0 {
		remaining = 1
	}

	fmt.Fprintf(os.Stderr, "\r%s Retrying in %ds...", reason, remaining)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	start := time.Now()
	for {
		select {
		case <-ticker.C:
			elapsed := time.Since(start)
			left := int((duration - elapsed).Seconds())
			if left <= 0 {
				fmt.Fprintf(os.Stderr, "\r%s Retrying now...     \n", reason)
				return
			}
			fmt.Fprintf(os.Stderr, "\r%s Retrying in %ds...    ", reason, left)
		}
	}
}

func (h *RateLimitHandler) GetMaxRetries() int {
	if h.config == nil {
		return 3
	}
	return h.config.RateLimit.MaxRetries
}

func (h *RateLimitHandler) GetRequestDelay() time.Duration {
	if h.config == nil {
		return 2500 * time.Millisecond
	}
	return h.config.RateLimit.RequestDelay
}

func (h *RateLimitHandler) AddJitter(baseDelay time.Duration) time.Duration {
	jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
	return baseDelay + jitter
}

func (h *RateLimitHandler) LastRateLimit() *RateLimitInfo {
	return h.lastLimit
}

func firstHeader(headers map[string][]string, key string) string {
	key = strings.ToLower(key)
	for k, values := range headers {
		if strings.ToLower(k) == key && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}
