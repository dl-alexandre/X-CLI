package xapi

import (
	"testing"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/config"
)

func TestRateLimitHandler_ParseRateLimitHeaders(t *testing.T) {
	cfg := &config.Config{}
	handler := NewRateLimitHandler(cfg, false)

	tests := []struct {
		name     string
		headers  map[string][]string
		expected *RateLimitInfo
	}{
		{
			name: "valid headers",
			headers: map[string][]string{
				"X-Rate-Limit-Limit":     {"100"},
				"X-Rate-Limit-Remaining": {"50"},
				"X-Rate-Limit-Reset":     {"1609459200"},
			},
			expected: &RateLimitInfo{
				Limit:     100,
				Remaining: 50,
				ResetAt:   1609459200,
			},
		},
		{
			name: "case insensitive headers",
			headers: map[string][]string{
				"x-rate-limit-limit":     {"200"},
				"x-rate-limit-remaining": {"150"},
				"x-rate-limit-reset":     {"1609459200"},
			},
			expected: &RateLimitInfo{
				Limit:     200,
				Remaining: 150,
				ResetAt:   1609459200,
			},
		},
		{
			name:     "empty headers",
			headers:  map[string][]string{},
			expected: nil,
		},
		{
			name: "partial headers",
			headers: map[string][]string{
				"X-Rate-Limit-Limit": {"100"},
			},
			expected: &RateLimitInfo{
				Limit: 100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.ParseRateLimitHeaders(tt.headers)

			if tt.expected == nil {
				if result != nil {
					t.Error("ParseRateLimitHeaders() should return nil for empty headers")
				}
				return
			}

			if result == nil {
				t.Fatal("ParseRateLimitHeaders() returned nil")
			}

			if result.Limit != tt.expected.Limit {
				t.Errorf("Limit = %d, expected %d", result.Limit, tt.expected.Limit)
			}

			if result.Remaining != tt.expected.Remaining {
				t.Errorf("Remaining = %d, expected %d", result.Remaining, tt.expected.Remaining)
			}

			if result.ResetAt != tt.expected.ResetAt {
				t.Errorf("ResetAt = %d, expected %d", result.ResetAt, tt.expected.ResetAt)
			}
		})
	}
}

func TestRateLimitHandler_ShouldRetry(t *testing.T) {
	cfg := &config.Config{}
	handler := NewRateLimitHandler(cfg, false)

	tests := []struct {
		statusCode int
		expected   bool
	}{
		{429, true},
		{200, false},
		{401, false},
		{500, false},
		{403, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := handler.ShouldRetry(tt.statusCode)
			if result != tt.expected {
				t.Errorf("ShouldRetry(%d) = %v, expected %v", tt.statusCode, result, tt.expected)
			}
		})
	}
}

func TestRateLimitHandler_GetWaitDuration(t *testing.T) {
	cfg := &config.Config{
		RateLimit: config.RateLimitConfig{
			RetryBaseDelay: 5 * time.Second,
		},
	}
	handler := NewRateLimitHandler(cfg, false)

	t.Run("with rate limit info", func(t *testing.T) {
		futureTime := time.Now().Add(30 * time.Second)
		info := &RateLimitInfo{
			Reset: futureTime,
		}

		duration := handler.GetWaitDuration(info)

		if duration < 29*time.Second || duration > 31*time.Second {
			t.Errorf("GetWaitDuration() = %v, expected ~30s", duration)
		}
	})

	t.Run("without rate limit info", func(t *testing.T) {
		duration := handler.GetWaitDuration(nil)

		if duration != 5*time.Second {
			t.Errorf("GetWaitDuration(nil) = %v, expected 5s", duration)
		}
	})

	t.Run("past reset time", func(t *testing.T) {
		pastTime := time.Now().Add(-10 * time.Second)
		info := &RateLimitInfo{
			Reset: pastTime,
		}

		duration := handler.GetWaitDuration(info)

		if duration != 1*time.Second {
			t.Errorf("GetWaitDuration() for past time = %v, expected 1s", duration)
		}
	})
}

func TestRateLimitHandler_GetMaxRetries(t *testing.T) {
	t.Run("with config", func(t *testing.T) {
		cfg := &config.Config{
			RateLimit: config.RateLimitConfig{
				MaxRetries: 5,
			},
		}
		handler := NewRateLimitHandler(cfg, false)

		result := handler.GetMaxRetries()
		if result != 5 {
			t.Errorf("GetMaxRetries() = %d, expected 5", result)
		}
	})

	t.Run("without config", func(t *testing.T) {
		handler := NewRateLimitHandler(nil, false)

		result := handler.GetMaxRetries()
		if result != 3 {
			t.Errorf("GetMaxRetries() = %d, expected 3 (default)", result)
		}
	})
}

func TestRateLimitHandler_GetRequestDelay(t *testing.T) {
	t.Run("with config", func(t *testing.T) {
		cfg := &config.Config{
			RateLimit: config.RateLimitConfig{
				RequestDelay: 2 * time.Second,
			},
		}
		handler := NewRateLimitHandler(cfg, false)

		result := handler.GetRequestDelay()
		if result != 2*time.Second {
			t.Errorf("GetRequestDelay() = %v, expected 2s", result)
		}
	})

	t.Run("without config", func(t *testing.T) {
		handler := NewRateLimitHandler(nil, false)

		result := handler.GetRequestDelay()
		if result != 2500*time.Millisecond {
			t.Errorf("GetRequestDelay() = %v, expected 2500ms", result)
		}
	})
}

func TestRateLimitHandler_AddJitter(t *testing.T) {
	cfg := &config.Config{}
	handler := NewRateLimitHandler(cfg, false)

	baseDelay := 2 * time.Second

	for i := 0; i < 100; i++ {
		result := handler.AddJitter(baseDelay)

		if result < baseDelay {
			t.Errorf("AddJitter() = %v, should be >= base delay %v", result, baseDelay)
		}

		if result > baseDelay+1*time.Second {
			t.Errorf("AddJitter() = %v, should be <= base delay + 1s", result)
		}
	}
}

func TestRateLimitHandler_LastRateLimit(t *testing.T) {
	cfg := &config.Config{}
	handler := NewRateLimitHandler(cfg, false)

	if handler.LastRateLimit() != nil {
		t.Error("LastRateLimit() should return nil initially")
	}

	headers := map[string][]string{
		"X-Rate-Limit-Limit":     {"100"},
		"X-Rate-Limit-Remaining": {"50"},
		"X-Rate-Limit-Reset":     {"1609459200"},
	}

	handler.ParseRateLimitHeaders(headers)

	last := handler.LastRateLimit()
	if last == nil {
		t.Fatal("LastRateLimit() should return info after parsing")
	}

	if last.Limit != 100 {
		t.Errorf("LastRateLimit().Limit = %d, expected 100", last.Limit)
	}
}

func TestFirstHeader(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string][]string
		key      string
		expected string
	}{
		{
			name: "single value",
			headers: map[string][]string{
				"X-Test": {"value1"},
			},
			key:      "X-Test",
			expected: "value1",
		},
		{
			name: "multiple values",
			headers: map[string][]string{
				"X-Test": {"value1", "value2"},
			},
			key:      "X-Test",
			expected: "value1",
		},
		{
			name: "case insensitive",
			headers: map[string][]string{
				"X-TEST": {"value1"},
			},
			key:      "x-test",
			expected: "value1",
		},
		{
			name:     "missing key",
			headers:  map[string][]string{},
			key:      "X-Test",
			expected: "",
		},
		{
			name: "empty values",
			headers: map[string][]string{
				"X-Test": {},
			},
			key:      "X-Test",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := firstHeader(tt.headers, tt.key)
			if result != tt.expected {
				t.Errorf("firstHeader() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
