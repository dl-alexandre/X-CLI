package auth

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/chromedp/chromedp"
)

// BrowserLogin launches Chrome and waits for user to login, then saves cookies.
func BrowserLogin() (*Session, error) {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║  X Login - Browser Authentication                            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Step 1: Chrome will open")
	fmt.Println("Step 2: Log in to X (if not already logged in)")
	fmt.Println("Step 3: Return here and press ENTER")
	fmt.Println()

	// Create Chrome allocator with visible window
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.WindowSize(1200, 800),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)
	defer cancelBrowser()

	// Navigate to X home page
	fmt.Println("Opening Chrome...")
	if err := chromedp.Run(browserCtx, chromedp.Navigate("https://x.com/home")); err != nil {
		return nil, fmt.Errorf("open Chrome: %w", err)
	}

	fmt.Println()
	fmt.Println("✓ Chrome is now open")
	fmt.Println()
	fmt.Println("Please:")
	fmt.Println("  1. Log in to X if you see a login page")
	fmt.Println("  2. Wait until you see your home timeline")
	fmt.Println("  3. Come back to this terminal")
	fmt.Println()
	fmt.Print("Press ENTER when you're logged in to X... ")

	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')

	fmt.Println()
	fmt.Println("Reading cookies from Chrome...")

	// Get cookies
	var cookieStr string
	if err := chromedp.Run(browserCtx, chromedp.Evaluate(`document.cookie`, &cookieStr)); err != nil {
		cancelBrowser()
		cancelAlloc()
		return nil, fmt.Errorf("read cookies: %w", err)
	}

	// Close Chrome
	cancelBrowser()
	cancelAlloc()

	if cookieStr == "" {
		return nil, fmt.Errorf("no cookies found - are you logged in?")
	}

	// Parse cookies
	var cookies []*http.Cookie
	var authToken, ct0 string

	parts := strings.Split(cookieStr, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		eqIdx := strings.Index(part, "=")
		if eqIdx < 0 {
			continue
		}

		name := part[:eqIdx]
		value := part[eqIdx+1:]

		cookies = append(cookies, &http.Cookie{
			Name:   name,
			Value:  value,
			Domain: ".x.com",
			Path:   "/",
			Secure: true,
		})

		switch name {
		case "auth_token":
			authToken = value
		case "ct0":
			ct0 = value
		}
	}

	if authToken == "" || ct0 == "" {
		return nil, fmt.Errorf("login incomplete - auth_token or ct0 not found in cookies\n\nFound cookies: %d\nCookie names: %v",
			len(cookies),
			func() []string {
				names := []string{}
				for _, c := range cookies {
					names = append(names, c.Name)
				}
				return names
			}())
	}

	return &Session{
		AuthToken:    authToken,
		CT0:          ct0,
		CookieString: cookieString(cookies),
		Browser:      "chrome",
		Cookies:      cookies,
	}, nil
}
