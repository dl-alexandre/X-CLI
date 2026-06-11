package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/config"
)

type Session struct {
	AuthToken    string         `json:"auth_token"`
	CT0          string         `json:"ct0"`
	CookieString string         `json:"cookie_string"`
	Browser      string         `json:"browser,omitempty"`
	Cookies      []*http.Cookie `json:"-"`
}

type cacheEntry struct {
	SavedAt string  `json:"saved_at"`
	Session Session `json:"session"`
}

func Load(cfg *config.Config) (*Session, error) {
	if cfg == nil {
		return nil, nil
	}

	// Try environment variables first
	if session := loadFromEnv(cfg); session != nil {
		return session, nil
	}

	if strings.EqualFold(cfg.Auth.Source, "env") {
		return nil, errors.New("auth.source is env but X_AUTH_TOKEN/X_CT0 are missing")
	}

	// Try file-based session (no keychain access)
	if session, err := LoadSession(""); err == nil && session != nil {
		return session, nil
	}

	// Skip browser cookie extraction (avoids keychain prompts)
	// Users should use 'x login' instead

	return nil, errors.New("not authenticated - run 'x login' to authenticate")
}

func loadFromEnv(cfg *config.Config) *Session {
	authToken := strings.TrimSpace(cfg.Auth.Token)
	ct0 := strings.TrimSpace(cfg.Auth.CT0)
	if authToken == "" || ct0 == "" {
		return nil
	}

	cookies := []*http.Cookie{
		{Name: "auth_token", Value: authToken, Domain: ".x.com", Path: "/", Secure: true, HttpOnly: true},
		{Name: "ct0", Value: ct0, Domain: ".x.com", Path: "/", Secure: true},
	}

	return &Session{
		AuthToken:    authToken,
		CT0:          ct0,
		CookieString: cookieString(cookies),
		Browser:      "env",
		Cookies:      cookies,
	}
}

func cookieString(cookies []*http.Cookie) string {
	parts := make([]string, 0, len(cookies))
	seen := map[string]bool{}
	for _, cookie := range cookies {
		if cookie == nil || cookie.Name == "" || cookie.Value == "" {
			continue
		}
		if seen[cookie.Name] {
			continue
		}
		seen[cookie.Name] = true
		parts = append(parts, cookie.Name+"="+cookie.Value)
	}
	return strings.Join(parts, "; ")
}

func parseCookieString(raw string) []*http.Cookie {
	parts := strings.Split(raw, ";")
	cookies := make([]*http.Cookie, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		cookies = append(cookies, &http.Cookie{
			Name:   strings.TrimSpace(name),
			Value:  strings.TrimSpace(value),
			Domain: ".x.com",
			Path:   "/",
			Secure: true,
		})
	}
	return cookies
}

// SaveSession saves a browser session to the cache
func SaveSession(profile string, session *Session) error {
	if profile == "" {
		profile = DefaultProfile
	}

	entry := cacheEntry{
		SavedAt: time.Now().UTC().Format(time.RFC3339),
		Session: *session,
	}

	body, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	cachePath, err := sessionCacheFilePath(profile)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return err
	}

	return os.WriteFile(cachePath, body, 0600)
}

// LoadSession loads a browser session from the cache
func LoadSession(profile string) (*Session, error) {
	if profile == "" {
		profile = DefaultProfile
	}

	cachePath, err := sessionCacheFilePath(profile)
	if err != nil {
		return nil, err
	}

	body, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var entry cacheEntry
	if err := json.Unmarshal(body, &entry); err != nil {
		return nil, err
	}

	savedAt, err := time.Parse(time.RFC3339, entry.SavedAt)
	if err != nil {
		return nil, err
	}

	// Check if expired (7 days)
	if time.Since(savedAt) > 7*24*time.Hour {
		return nil, errors.New("session expired")
	}

	if entry.Session.AuthToken == "" || entry.Session.CT0 == "" {
		return nil, errors.New("session incomplete")
	}

	entry.Session.Cookies = parseCookieString(entry.Session.CookieString)
	return &entry.Session, nil
}

// DeleteSession removes a saved session
func DeleteSession(profile string) error {
	if profile == "" {
		profile = DefaultProfile
	}

	cachePath, err := sessionCacheFilePath(profile)
	if err != nil {
		return err
	}

	return os.Remove(cachePath)
}

func sessionCacheFilePath(profile string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	filename := "session.json"
	if profile != DefaultProfile {
		filename = fmt.Sprintf("session-%s.json", profile)
	}

	return filepath.Join(home, ".config", "x-cli", filename), nil
}
