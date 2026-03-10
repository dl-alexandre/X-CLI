package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	OAuthAuthorizeURL = "https://twitter.com/i/oauth2/authorize"
	OAuthTokenURL     = "https://api.twitter.com/2/oauth2/token"
	OAuthClientID     = "SmpOaGp1U2l0eFZFRjVja2t5Qks6MTpjaQ"
	OAuthRedirectURI  = "http://127.0.0.1:8080/callback"
	OAuthScopes       = "tweet.read tweet.write users.read offline.access"
)

type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token"`
	Scope        string    `json:"scope"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type PKCE struct {
	Verifier  string
	Challenge string
	State     string
}

type OAuthFlow struct {
	pkce       *PKCE
	callbackCh chan *OAuthToken
	errorCh    chan error
	server     *http.Server
}

func NewOAuthFlow() *OAuthFlow {
	return &OAuthFlow{
		callbackCh: make(chan *OAuthToken, 1),
		errorCh:    make(chan error, 1),
	}
}

func (f *OAuthFlow) generatePKCE() (*PKCE, error) {
	verifier, err := generateRandomString(128)
	if err != nil {
		return nil, fmt.Errorf("generate verifier: %w", err)
	}

	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	state, err := generateRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	f.pkce = &PKCE{
		Verifier:  verifier,
		Challenge: challenge,
		State:     state,
	}

	return f.pkce, nil
}

func (f *OAuthFlow) Start() (*OAuthToken, error) {
	pkce, err := f.generatePKCE()
	if err != nil {
		return nil, err
	}

	if err := f.startCallbackServer(); err != nil {
		return nil, fmt.Errorf("start callback server: %w", err)
	}

	authURL := f.buildAuthURL(pkce)

	fmt.Println("\nOpening browser for authentication...")
	fmt.Println("If the browser doesn't open automatically, visit:")
	fmt.Printf("\n%s\n\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Could not open browser automatically: %v\n", err)
		fmt.Println("Please open the URL above manually.")
	}

	select {
	case token := <-f.callbackCh:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = f.server.Shutdown(ctx)
		return token, nil
	case err := <-f.errorCh:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = f.server.Shutdown(ctx)
		return nil, err
	case <-time.After(5 * time.Minute):
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = f.server.Shutdown(ctx)
		return nil, errors.New("authentication timed out after 5 minutes")
	}
}

func (f *OAuthFlow) startCallbackServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", f.handleCallback)

	f.server = &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: mux,
	}

	go func() {
		if err := f.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			f.errorCh <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	return nil
}

func (f *OAuthFlow) handleCallback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	if query.Get("state") != f.pkce.State {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		f.errorCh <- errors.New("invalid state parameter")
		return
	}

	code := query.Get("code")
	if code == "" {
		errorDesc := query.Get("error_description")
		if errorDesc == "" {
			errorDesc = query.Get("error")
		}
		http.Error(w, "Authorization failed: "+errorDesc, http.StatusBadRequest)
		f.errorCh <- fmt.Errorf("authorization failed: %s", errorDesc)
		return
	}

	token, err := f.exchangeCodeForToken(code)
	if err != nil {
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		f.errorCh <- fmt.Errorf("token exchange: %w", err)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>Authentication Successful</title></head>
		<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #1a1a1a; color: #fff;">
			<div style="text-align: center;">
				<h1 style="color: #4ade80;">✓ Authentication Successful</h1>
				<p style="color: #888;">You can close this window and return to the terminal.</p>
			</div>
			<script>setTimeout(() => window.close(), 2000);</script>
		</body>
		</html>
	`))

	f.callbackCh <- token
}

func (f *OAuthFlow) exchangeCodeForToken(code string) (*OAuthToken, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", OAuthRedirectURI)
	data.Set("code_verifier", f.pkce.Verifier)
	data.Set("client_id", OAuthClientID)

	req, err := http.NewRequest(http.MethodPost, OAuthTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var token OAuthToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	return &token, nil
}

func (f *OAuthFlow) buildAuthURL(pkce *PKCE) string {
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", OAuthClientID)
	params.Set("redirect_uri", OAuthRedirectURI)
	params.Set("scope", OAuthScopes)
	params.Set("state", pkce.State)
	params.Set("code_challenge", pkce.Challenge)
	params.Set("code_challenge_method", "S256")

	return OAuthAuthorizeURL + "?" + params.Encode()
}

func RefreshOAuthToken(refreshToken string) (*OAuthToken, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", OAuthClientID)

	req, err := http.NewRequest(http.MethodPost, OAuthTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("token refresh failed (status %d): %s", resp.StatusCode, string(body))
	}

	var token OAuthToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	return &token, nil
}

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[int(bytes[i])%len(charset)]
	}

	return string(result), nil
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return exec.Command(cmd, args...).Start()
}

func (t *OAuthToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt.Add(-5 * time.Minute))
}

func (t *OAuthToken) NeedsRefresh() bool {
	return time.Now().After(t.ExpiresAt.Add(-10 * time.Minute))
}
