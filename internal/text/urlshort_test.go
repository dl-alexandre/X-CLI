package text

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDetectURLs_NoURLs(t *testing.T) {
	shortener := NewURLShortener(DefaultURLShortenerConfig())
	text := "This is a text without any URLs"

	urls := shortener.DetectURLs(text)

	if len(urls) != 0 {
		t.Errorf("Expected 0 URLs, got %d", len(urls))
	}
}

func TestDetectURLs_SingleURL(t *testing.T) {
	shortener := NewURLShortener(DefaultURLShortenerConfig())
	text := "Check out https://example.com for more info"

	urls := shortener.DetectURLs(text)

	if len(urls) != 1 {
		t.Errorf("Expected 1 URL, got %d", len(urls))
		return
	}

	if urls[0].Original != "https://example.com" {
		t.Errorf("Expected URL %q, got %q", "https://example.com", urls[0].Original)
	}
}

func TestDetectURLs_MultipleURLs(t *testing.T) {
	shortener := NewURLShortener(DefaultURLShortenerConfig())
	text := "Visit https://example.com and http://test.org for details"

	urls := shortener.DetectURLs(text)

	if len(urls) != 2 {
		t.Errorf("Expected 2 URLs, got %d", len(urls))
		return
	}

	if urls[0].Original != "https://example.com" {
		t.Errorf("Expected first URL %q, got %q", "https://example.com", urls[0].Original)
	}

	if urls[1].Original != "http://test.org" {
		t.Errorf("Expected second URL %q, got %q", "http://test.org", urls[1].Original)
	}
}

func TestDetectURLs_URLWithQuery(t *testing.T) {
	shortener := NewURLShortener(DefaultURLShortenerConfig())
	text := "Link: https://example.com/path?query=value&other=123"

	urls := shortener.DetectURLs(text)

	if len(urls) != 1 {
		t.Errorf("Expected 1 URL, got %d", len(urls))
		return
	}

	expected := "https://example.com/path?query=value&other=123"
	if urls[0].Original != expected {
		t.Errorf("Expected URL %q, got %q", expected, urls[0].Original)
	}
}

func TestDetectURLs_URLWithFragment(t *testing.T) {
	shortener := NewURLShortener(DefaultURLShortenerConfig())
	text := "See https://example.com/page#section"

	urls := shortener.DetectURLs(text)

	if len(urls) != 1 {
		t.Errorf("Expected 1 URL, got %d", len(urls))
		return
	}

	if urls[0].Original != "https://example.com/page#section" {
		t.Errorf("Expected URL with fragment, got %q", urls[0].Original)
	}
}

func TestShortenURL_Disabled(t *testing.T) {
	config := DefaultURLShortenerConfig()
	config.Enabled = false
	shortener := NewURLShortener(config)

	ctx := context.Background()
	result, err := shortener.ShortenURL(ctx, "https://example.com")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != "https://example.com" {
		t.Errorf("Expected original URL when disabled, got %q", result)
	}
}

func TestShortenURL_TinyURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "api-create.php") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("https://tinyurl.com/abc123"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config := URLShortenerConfig{
		Enabled: true,
		Service: ShortenerTinyURL,
		Timeout: 5 * time.Second,
	}
	shortener := NewURLShortener(config)
	shortener.httpClient = server.Client()

	ctx := context.Background()
	result, err := shortener.ShortenURL(ctx, "https://example.com/very/long/url/path")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result == "https://example.com/very/long/url/path" {
		t.Error("URL should have been shortened")
	}
}

func TestShortenURL_IsGd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "create.php") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("https://is.gd/abc123"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config := URLShortenerConfig{
		Enabled: true,
		Service: ShortenerIsGd,
		Timeout: 5 * time.Second,
	}
	shortener := NewURLShortener(config)
	shortener.httpClient = server.Client()

	ctx := context.Background()
	result, err := shortener.ShortenURL(ctx, "https://example.com/long")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result == "https://example.com/long" {
		t.Error("URL should have been shortened")
	}
}

func TestShortenURL_VGd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "create.php") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("https://v.gd/abc123"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config := URLShortenerConfig{
		Enabled: true,
		Service: ShortenerVGd,
		Timeout: 5 * time.Second,
	}
	shortener := NewURLShortener(config)
	shortener.httpClient = server.Client()

	ctx := context.Background()
	result, err := shortener.ShortenURL(ctx, "https://example.com/long")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result == "https://example.com/long" {
		t.Error("URL should have been shortened")
	}
}

func TestShortenURL_Custom(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"short_url": "https://short.link/abc"}`))
	}))
	defer server.Close()

	config := URLShortenerConfig{
		Enabled:      true,
		Service:      ShortenerCustom,
		CustomAPIURL: server.URL,
		Timeout:      5 * time.Second,
	}
	shortener := NewURLShortener(config)
	shortener.httpClient = server.Client()

	ctx := context.Background()
	result, err := shortener.ShortenURL(ctx, "https://example.com/long")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != "https://short.link/abc" {
		t.Errorf("Expected shortened URL, got %q", result)
	}
}

func TestShortenURL_CustomPlainText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("https://short.link/xyz"))
	}))
	defer server.Close()

	config := URLShortenerConfig{
		Enabled:      true,
		Service:      ShortenerCustom,
		CustomAPIURL: server.URL,
		Timeout:      5 * time.Second,
	}
	shortener := NewURLShortener(config)
	shortener.httpClient = server.Client()

	ctx := context.Background()
	result, err := shortener.ShortenURL(ctx, "https://example.com/long")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != "https://short.link/xyz" {
		t.Errorf("Expected shortened URL, got %q", result)
	}
}

func TestShortenAllURLs_NoURLs(t *testing.T) {
	shortener := NewURLShortener(DefaultURLShortenerConfig())
	ctx := context.Background()

	text := "No URLs here"
	result, urls, err := shortener.ShortenAllURLs(ctx, text)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != text {
		t.Errorf("Text should not change without URLs")
	}

	if len(urls) != 0 {
		t.Errorf("Expected 0 URLs, got %d", len(urls))
	}
}

func TestShortenAllURLs_MultipleURLs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("https://short.link/abc"))
	}))
	defer server.Close()

	config := URLShortenerConfig{
		Enabled: true,
		Service: ShortenerTinyURL,
		Timeout: 5 * time.Second,
	}
	shortener := NewURLShortener(config)
	shortener.httpClient = server.Client()

	ctx := context.Background()
	text := "Check https://example.com/very/long/one and https://example.com/very/long/two"

	result, urls, err := shortener.ShortenAllURLs(ctx, text)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(urls) != 2 {
		t.Errorf("Expected 2 URLs, got %d", len(urls))
	}

	for _, u := range urls {
		if u.IsShortened {
			if !strings.Contains(result, u.Shortened) {
				t.Errorf("Result should contain shortened URL %q", u.Shortened)
			}
		}
	}
}

func TestCountCharacters_NoURLs(t *testing.T) {
	text := "Simple text without URLs"
	count := CountCharacters(text, nil)

	expected := len(text)
	if count != expected {
		t.Errorf("Expected %d characters, got %d", expected, count)
	}
}

func TestCountCharacters_WithShortenedURLs(t *testing.T) {
	originalText := "Check https://example.com/very/long/url/path/that/is/long for info"
	urls := []URLInfo{
		{
			Original:    "https://example.com/very/long/url/path/that/is/long",
			Shortened:   "https://short.link/abc",
			IsShortened: true,
		},
	}

	count := CountCharacters(originalText, urls)

	originalLen := utf8RuneCountInString("https://example.com/very/long/url/path/that/is/long")
	shortenedLen := utf8RuneCountInString("https://short.link/abc")
	expected := utf8RuneCountInString(originalText) - originalLen + shortenedLen

	if count != expected {
		t.Errorf("Expected %d characters (with shortened URL), got %d", expected, count)
	}
}

func TestCountCharacters_WithUnshortenedURLs(t *testing.T) {
	text := "Check https://example.com/very/long for info"
	urls := []URLInfo{
		{
			Original:    "https://example.com/very/long",
			Shortened:   "https://example.com/very/long",
			IsShortened: false,
		},
	}

	count := CountCharacters(text, urls)
	expected := utf8RuneCountInString(text)

	if count != expected {
		t.Errorf("Expected %d characters, got %d", expected, count)
	}
}

func TestExtractURLs(t *testing.T) {
	text := "Visit https://example.com and http://test.org"
	urls := ExtractURLs(text)

	if len(urls) != 2 {
		t.Errorf("Expected 2 URLs, got %d", len(urls))
	}

	if urls[0] != "https://example.com" {
		t.Errorf("Expected %q, got %q", "https://example.com", urls[0])
	}

	if urls[1] != "http://test.org" {
		t.Errorf("Expected %q, got %q", "http://test.org", urls[1])
	}
}

func TestContainsURL_True(t *testing.T) {
	text := "Check https://example.com"

	if !ContainsURL(text) {
		t.Error("Expected ContainsURL to return true")
	}
}

func TestContainsURL_False(t *testing.T) {
	text := "No URLs here"

	if ContainsURL(text) {
		t.Error("Expected ContainsURL to return false")
	}
}

func TestReplaceURLs(t *testing.T) {
	text := "Visit https://example.com and https://test.org"
	replacements := map[string]string{
		"https://example.com": "https://short.link/a",
		"https://test.org":    "https://short.link/b",
	}

	result := ReplaceURLs(text, replacements)

	expected := "Visit https://short.link/a and https://short.link/b"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestSplitIntoChunks_WithURLs(t *testing.T) {
	longURL := "https://example.com/very/long/path/that/takes/many/characters/to/write/out/fully"
	words := make([]string, 50)
	for i := range words {
		words[i] = "word"
	}
	words[25] = longURL
	text := strings.Join(words, " ")

	chunks := SplitIntoChunks(text)

	if len(chunks) < 2 {
		t.Errorf("Expected multiple chunks for long text with URL, got %d", len(chunks))
	}

	for i, chunk := range chunks {
		if len(chunk) > MaxChunkLength {
			t.Errorf("Chunk %d exceeds max length: %d > %d", i, len(chunk), MaxChunkLength)
		}
	}
}

func TestSplitIntoChunks_URLAtBoundary(t *testing.T) {
	url := "https://example.com/path"
	prefix := strings.Repeat("x ", 130)
	text := prefix + url

	chunks := SplitIntoChunks(text)

	urlFound := false
	for _, chunk := range chunks {
		if strings.Contains(chunk, url) {
			urlFound = true
			break
		}
	}

	if !urlFound {
		t.Error("URL should be present in at least one chunk")
	}
}

func TestShortenURL_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := URLShortenerConfig{
		Enabled:      true,
		Service:      ShortenerCustom,
		CustomAPIURL: server.URL,
		Timeout:      100 * time.Millisecond,
	}
	shortener := NewURLShortener(config)

	ctx := context.Background()
	_, err := shortener.ShortenURL(ctx, "https://example.com")

	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestShortenURL_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := URLShortenerConfig{
		Enabled:      true,
		Service:      ShortenerCustom,
		CustomAPIURL: server.URL,
		Timeout:      5 * time.Second,
	}
	shortener := NewURLShortener(config)

	ctx := context.Background()
	_, err := shortener.ShortenURL(ctx, "https://example.com")

	if err == nil {
		t.Error("Expected error for server error response")
	}
}

func TestDefaultURLShortenerConfig(t *testing.T) {
	config := DefaultURLShortenerConfig()

	if config.Enabled {
		t.Error("Default config should have URL shortening disabled")
	}

	if config.Service != ShortenerTinyURL {
		t.Errorf("Expected default service to be tinyurl, got %s", config.Service)
	}

	if config.Timeout != 10*time.Second {
		t.Errorf("Expected default timeout of 10s, got %v", config.Timeout)
	}
}

func utf8RuneCountInString(s string) int {
	return len([]rune(s))
}
