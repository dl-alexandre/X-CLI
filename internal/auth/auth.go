package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/browserutils/kooky"
	_ "github.com/browserutils/kooky/browser/all"
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

const cacheTTL = 24 * time.Hour

func Load(cfg *config.Config) (*Session, error) {
	if cfg == nil {
		return nil, nil
	}

	if session := loadFromEnv(cfg); session != nil {
		return session, nil
	}

	if strings.EqualFold(cfg.Auth.Source, "env") {
		return nil, errors.New("auth.source is env but X_AUTH_TOKEN/X_CT0 are missing")
	}

	session, err := extractFromBrowser()
	if err == nil && session != nil {
		_ = saveToCache(*session)
		return session, nil
	}

	if session, cacheErr := loadFromCache(); cacheErr == nil && session != nil {
		return session, nil
	}

	if err != nil {
		return nil, err
	}
	return nil, errors.New("no browser or cached session available")
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

func extractFromBrowser() (*Session, error) {
	ctx := context.Background()
	stores := kooky.FindAllCookieStores(ctx)

	var bestCookies []*http.Cookie
	bestBrowser := ""
	bestScore := 0

	for _, store := range stores {
		if strings.Contains(store.FilePath(), "/Network/") {
			_ = store.Close()
			continue
		}

		cookies := store.TraverseCookies().Collect(ctx)
		_ = store.Close()

		relevant := filterRelevantCookies(cookies)
		score := scoreCookies(relevant)
		if score > bestScore {
			bestScore = score
			bestCookies = relevant
			bestBrowser = store.Browser()
		}
	}

	if bestScore == 0 || len(bestCookies) == 0 {
		return nil, errors.New("no usable X browser cookies found")
	}

	authToken := firstCookieValue(bestCookies, "auth_token")
	ct0 := firstCookieValue(bestCookies, "ct0")
	if authToken == "" || ct0 == "" {
		return nil, errors.New("browser cookies missing auth_token or ct0")
	}

	return &Session{
		AuthToken:    authToken,
		CT0:          ct0,
		CookieString: cookieString(bestCookies),
		Browser:      bestBrowser,
		Cookies:      bestCookies,
	}, nil
}

func filterRelevantCookies(cookies kooky.Cookies) []*http.Cookie {
	filtered := make([]*http.Cookie, 0, len(cookies))
	seen := map[string]bool{}

	for _, cookie := range cookies {
		domain := strings.TrimSpace(cookie.Domain)
		if !isXDomain(domain) {
			continue
		}

		copied := cookie.Cookie
		copied.Value = strings.Trim(copied.Value, "\"")
		if copied.Value == "" {
			continue
		}
		if copied.Domain == "" {
			copied.Domain = ".x.com"
		}
		if copied.Path == "" {
			copied.Path = "/"
		}

		key := copied.Domain + "|" + copied.Name
		if seen[key] {
			continue
		}
		seen[key] = true
		filtered = append(filtered, &copied)
	}

	sort.Slice(filtered, func(i int, j int) bool {
		if filtered[i].Domain == filtered[j].Domain {
			return filtered[i].Name < filtered[j].Name
		}
		return filtered[i].Domain < filtered[j].Domain
	})

	return filtered
}

func isXDomain(domain string) bool {
	domain = strings.TrimSpace(domain)
	return domain == "x.com" || domain == ".x.com" || domain == "twitter.com" || domain == ".twitter.com" || strings.HasSuffix(domain, ".x.com") || strings.HasSuffix(domain, ".twitter.com")
}

func scoreCookies(cookies []*http.Cookie) int {
	score := 0
	for _, cookie := range cookies {
		switch cookie.Name {
		case "auth_token", "ct0":
			score += 100
		case "att", "gt", "kdt", "twid":
			score += 10
		default:
			score++
		}
	}
	return score
}

func firstCookieValue(cookies []*http.Cookie, name string) string {
	for _, cookie := range cookies {
		if cookie.Name == name && cookie.Value != "" {
			return cookie.Value
		}
	}
	return ""
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

func cacheFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "x", "auth-cache.json"), nil
}

func loadFromCache() (*Session, error) {
	cachePath, err := cacheFilePath()
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
	if time.Since(savedAt) > cacheTTL {
		return nil, errors.New("cached auth expired")
	}
	if entry.Session.AuthToken == "" || entry.Session.CT0 == "" {
		return nil, errors.New("cached auth incomplete")
	}
	entry.Session.Cookies = parseCookieString(entry.Session.CookieString)
	return &entry.Session, nil
}

func saveToCache(session Session) error {
	cachePath, err := cacheFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return err
	}

	entry := cacheEntry{SavedAt: time.Now().UTC().Format(time.RFC3339), Session: session}
	body, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if err := os.WriteFile(cachePath, body, 0600); err != nil {
		return err
	}
	return nil
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
