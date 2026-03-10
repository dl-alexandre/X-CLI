package text

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type ShortenerService string

const (
	ShortenerNone    ShortenerService = "none"
	ShortenerTinyURL ShortenerService = "tinyurl"
	ShortenerIsGd    ShortenerService = "isgd"
	ShortenerVGd     ShortenerService = "vgd"
	ShortenerCustom  ShortenerService = "custom"
)

type URLShortenerConfig struct {
	Enabled      bool             `mapstructure:"enabled"`
	Service      ShortenerService `mapstructure:"service"`
	CustomAPIURL string           `mapstructure:"custom_api_url"`
	Timeout      time.Duration    `mapstructure:"timeout"`
}

type URLInfo struct {
	Original    string
	Shortened   string
	StartPos    int
	EndPos      int
	IsShortened bool
}

type URLShortener struct {
	config     URLShortenerConfig
	httpClient *http.Client
}

var urlRegex = regexp.MustCompile(`https?://[^\s<>"{}|\\^[\]]+`)

func DefaultURLShortenerConfig() URLShortenerConfig {
	return URLShortenerConfig{
		Enabled:      false,
		Service:      ShortenerTinyURL,
		CustomAPIURL: "",
		Timeout:      10 * time.Second,
	}
}

func NewURLShortener(config URLShortenerConfig) *URLShortener {
	return &URLShortener{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

func (s *URLShortener) DetectURLs(text string) []URLInfo {
	matches := urlRegex.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return nil
	}

	urls := make([]URLInfo, 0, len(matches))
	for _, match := range matches {
		originalURL := text[match[0]:match[1]]
		urls = append(urls, URLInfo{
			Original:  originalURL,
			StartPos:  match[0],
			EndPos:    match[1],
			Shortened: originalURL,
		})
	}

	return urls
}

func (s *URLShortener) ShortenURL(ctx context.Context, originalURL string) (string, error) {
	if !s.config.Enabled || s.config.Service == ShortenerNone {
		return originalURL, nil
	}

	switch s.config.Service {
	case ShortenerTinyURL:
		return s.shortenWithTinyURL(ctx, originalURL)
	case ShortenerIsGd:
		return s.shortenWithIsGd(ctx, originalURL)
	case ShortenerVGd:
		return s.shortenWithVGd(ctx, originalURL)
	case ShortenerCustom:
		return s.shortenWithCustom(ctx, originalURL)
	default:
		return originalURL, nil
	}
}

func (s *URLShortener) shortenWithTinyURL(ctx context.Context, originalURL string) (string, error) {
	apiURL := fmt.Sprintf("https://tinyurl.com/api-create.php?url=%s", url.QueryEscape(originalURL))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("tinyurl request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tinyurl returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	shortened := strings.TrimSpace(string(body))
	if shortened == "" {
		return "", fmt.Errorf("empty response from tinyurl")
	}

	return shortened, nil
}

func (s *URLShortener) shortenWithIsGd(ctx context.Context, originalURL string) (string, error) {
	apiURL := fmt.Sprintf("https://is.gd/create.php?format=simple&url=%s", url.QueryEscape(originalURL))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("is.gd request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("is.gd returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	shortened := strings.TrimSpace(string(body))
	if shortened == "" {
		return "", fmt.Errorf("empty response from is.gd")
	}

	return shortened, nil
}

func (s *URLShortener) shortenWithVGd(ctx context.Context, originalURL string) (string, error) {
	apiURL := fmt.Sprintf("https://v.gd/create.php?format=simple&url=%s", url.QueryEscape(originalURL))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("v.gd request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("v.gd returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	shortened := strings.TrimSpace(string(body))
	if shortened == "" {
		return "", fmt.Errorf("empty response from v.gd")
	}

	return shortened, nil
}

func (s *URLShortener) shortenWithCustom(ctx context.Context, originalURL string) (string, error) {
	if s.config.CustomAPIURL == "" {
		return "", fmt.Errorf("custom API URL not configured")
	}

	apiURL := fmt.Sprintf("%s?url=%s", s.config.CustomAPIURL, url.QueryEscape(originalURL))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("custom shortener request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("custom shortener returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var result struct {
		ShortURL  string `json:"short_url"`
		Shortened string `json:"shortened"`
		URL       string `json:"url"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		shortened := strings.TrimSpace(string(body))
		if shortened != "" {
			return shortened, nil
		}
		return "", fmt.Errorf("parse response: %w", err)
	}

	if result.ShortURL != "" {
		return result.ShortURL, nil
	}
	if result.Shortened != "" {
		return result.Shortened, nil
	}
	if result.URL != "" {
		return result.URL, nil
	}

	return "", fmt.Errorf("no shortened URL in response")
}

func (s *URLShortener) ShortenAllURLs(ctx context.Context, text string) (string, []URLInfo, error) {
	urls := s.DetectURLs(text)
	if len(urls) == 0 {
		return text, nil, nil
	}

	result := text
	offset := 0

	for i := range urls {
		originalURL := urls[i].Original
		shortened, err := s.ShortenURL(ctx, originalURL)
		if err != nil {
			shortened = originalURL
		}

		urls[i].Shortened = shortened
		urls[i].IsShortened = shortened != originalURL

		startPos := urls[i].StartPos + offset
		endPos := urls[i].EndPos + offset

		result = result[:startPos] + shortened + result[endPos:]

		lengthDiff := utf8.RuneCountInString(shortened) - utf8.RuneCountInString(originalURL)
		offset += lengthDiff
	}

	return result, urls, nil
}

func CountCharacters(text string, urls []URLInfo) int {
	if len(urls) == 0 {
		return utf8.RuneCountInString(text)
	}

	count := utf8.RuneCountInString(text)

	for _, urlInfo := range urls {
		originalLen := utf8.RuneCountInString(urlInfo.Original)
		shortenedLen := utf8.RuneCountInString(urlInfo.Shortened)

		if urlInfo.IsShortened {
			count = count - originalLen + shortenedLen
		}
	}

	return count
}

func ExtractURLs(text string) []string {
	return urlRegex.FindAllString(text, -1)
}

func ContainsURL(text string) bool {
	return urlRegex.MatchString(text)
}

func ReplaceURLs(text string, replacements map[string]string) string {
	urls := ExtractURLs(text)
	result := text

	for _, originalURL := range urls {
		if replacement, ok := replacements[originalURL]; ok {
			result = strings.Replace(result, originalURL, replacement, 1)
		}
	}

	return result
}
