package auth

import (
	"testing"
	"time"
)

func TestOAuthToken_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{
			name:      "expired token",
			expiresAt: time.Now().Add(-1 * time.Hour),
			expected:  true,
		},
		{
			name:      "token expiring soon (within 5 min buffer)",
			expiresAt: time.Now().Add(3 * time.Minute),
			expected:  true,
		},
		{
			name:      "valid token",
			expiresAt: time.Now().Add(2 * time.Hour),
			expected:  false,
		},
		{
			name:      "token just past boundary",
			expiresAt: time.Now().Add(6 * time.Minute),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &OAuthToken{
				ExpiresAt: tt.expiresAt,
			}

			result := token.IsExpired()
			if result != tt.expected {
				t.Errorf("IsExpired() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestOAuthToken_NeedsRefresh(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{
			name:      "expired token",
			expiresAt: time.Now().Add(-1 * time.Hour),
			expected:  true,
		},
		{
			name:      "token expiring soon (within 10 min buffer)",
			expiresAt: time.Now().Add(8 * time.Minute),
			expected:  true,
		},
		{
			name:      "valid token",
			expiresAt: time.Now().Add(2 * time.Hour),
			expected:  false,
		},
		{
			name:      "token just past boundary",
			expiresAt: time.Now().Add(11 * time.Minute),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &OAuthToken{
				ExpiresAt: tt.expiresAt,
			}

			result := token.NeedsRefresh()
			if result != tt.expected {
				t.Errorf("NeedsRefresh() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"short string", 10},
		{"medium string", 32},
		{"long string", 128},
		{"very long string", 256},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generateRandomString(tt.length)
			if err != nil {
				t.Fatalf("generateRandomString() error = %v", err)
			}

			if len(result) != tt.length {
				t.Errorf("generateRandomString() length = %d, expected %d", len(result), tt.length)
			}

			result2, err := generateRandomString(tt.length)
			if err != nil {
				t.Fatalf("generateRandomString() error = %v", err)
			}

			if result == result2 {
				t.Error("generateRandomString() should produce different values")
			}
		})
	}
}

func TestGenerateRandomString_Uniqueness(t *testing.T) {
	const iterations = 100
	const length = 32

	seen := make(map[string]bool)
	for i := 0; i < iterations; i++ {
		result, err := generateRandomString(length)
		if err != nil {
			t.Fatalf("generateRandomString() error = %v", err)
		}

		if seen[result] {
			t.Errorf("generateRandomString() produced duplicate: %s", result)
		}
		seen[result] = true
	}
}

func TestPKCEGeneration(t *testing.T) {
	flow := NewOAuthFlow()
	pkce, err := flow.generatePKCE()
	if err != nil {
		t.Fatalf("generatePKCE() error = %v", err)
	}

	if len(pkce.Verifier) != 128 {
		t.Errorf("Verifier length = %d, expected 128", len(pkce.Verifier))
	}

	if pkce.Challenge == "" {
		t.Error("Challenge should not be empty")
	}

	if pkce.State == "" {
		t.Error("State should not be empty")
	}

	if len(pkce.State) != 32 {
		t.Errorf("State length = %d, expected 32", len(pkce.State))
	}
}

func TestBuildAuthURL(t *testing.T) {
	flow := NewOAuthFlow()
	pkce, err := flow.generatePKCE()
	if err != nil {
		t.Fatalf("generatePKCE() error = %v", err)
	}

	url := flow.buildAuthURL(pkce)

	if url == "" {
		t.Error("buildAuthURL() should not return empty string")
	}

	expectedParts := []string{
		"response_type=code",
		"client_id=",
		"redirect_uri=",
		"scope=",
		"state=" + pkce.State,
		"code_challenge=",
		"code_challenge_method=S256",
	}

	for _, part := range expectedParts {
		if !contains(url, part) {
			t.Errorf("buildAuthURL() missing expected part: %s", part)
		}
	}
}

func TestGetTokenStatus(t *testing.T) {
	tests := []struct {
		name     string
		token    *OAuthToken
		expected string
	}{
		{
			name:     "nil token",
			token:    nil,
			expected: "not authenticated",
		},
		{
			name: "expired token",
			token: &OAuthToken{
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			expected: "expired",
		},
		{
			name: "needs refresh",
			token: &OAuthToken{
				ExpiresAt: time.Now().Add(8 * time.Minute),
			},
			expected: "needs refresh",
		},
		{
			name: "valid token",
			token: &OAuthToken{
				ExpiresAt: time.Now().Add(2 * time.Hour),
			},
			expected: "valid for",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTokenStatus(tt.token)
			if !contains(result, tt.expected) {
				t.Errorf("GetTokenStatus() = %q, expected to contain %q", result, tt.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
