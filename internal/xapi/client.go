package xapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/dl-alexandre/X-CLI/internal/auth"
	"github.com/dl-alexandre/X-CLI/internal/config"
	"github.com/dl-alexandre/X-CLI/internal/model"
)

const (
	bearerToken = "AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA"
	defaultUA   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
)

var fallbackQueryIDs = map[string]string{
	"UserByScreenName": "pLsOiyHJ1eFwPJlNmLp4Bg",
	"UserTweets":       "tBNuKtAJqe33sRX5V6Vlbg",
	"SearchTimeline":   "qUm8YPFHJWjQ56E_dP4MDg",
	"TweetDetail":      "vsCTCQrF8oqASUb-x2SBcg",
}

var defaultFeatures = map[string]bool{
	"creator_subscriptions_tweet_preview_api_enabled":                         true,
	"communities_web_enable_tweet_community_results_fetch":                    true,
	"c9s_tweet_anatomy_moderator_badge_enabled":                               true,
	"articles_preview_enabled":                                                true,
	"responsive_web_edit_tweet_api_enabled":                                   true,
	"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
	"view_counts_everywhere_api_enabled":                                      true,
	"longform_notetweets_consumption_enabled":                                 true,
	"responsive_web_twitter_article_tweet_consumption_enabled":                true,
	"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
	"longform_notetweets_rich_text_read_enabled":                              true,
	"freedom_of_speech_not_reach_fetch_enabled":                               true,
	"standardized_nudges_misinfo":                                             true,
	"responsive_web_graphql_timeline_navigation_enabled":                      true,
	"responsive_web_enhance_cards_enabled":                                    false,
}

type Client struct {
	config       *config.Config
	httpClient   *http.Client
	standardHTTP *http.Client
	verbose      bool
	debug        bool
	transport    *http.Transport
	guestToken   string
	guestTokenMu sync.Mutex
	queryIDs     map[string]string
	queryMu      sync.Mutex
	authSession  *auth.Session
	txProvider   TransactionIDProvider
	traceWriter  *txIDTraceWriter
}

type Options struct {
	Config  *config.Config
	Verbose bool
	Debug   bool
}

type NotImplementedError struct {
	Operation string
}

func (e NotImplementedError) Error() string {
	return fmt.Sprintf("%s is not implemented yet", e.Operation)
}

func NewClient(opts Options) *Client {
	proxyValue := ""
	if opts.Config != nil {
		proxyValue = opts.Config.HTTP.Proxy
	}
	transport := newNativeHTTPTransport(proxyValue)

	session, _ := auth.Load(opts.Config)
	txProvider := NewNativeTransactionProvider()
	if opts.Config != nil && opts.Config.Browser.StaticSalt != "" {
		if err := txProvider.SetStaticSalt(opts.Config.Browser.StaticSalt); err != nil {
			// Log error but continue with placeholder
		}
	}
	traceFile := ""
	traceMode := "writes"
	traceOps := ""
	if opts.Config != nil {
		traceFile = opts.Config.Browser.TraceTxIDFile
		traceMode = opts.Config.Browser.TraceTxIDMode
		traceOps = opts.Config.Browser.TraceTxIDOps
	}

	return &Client{
		config: opts.Config,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		standardHTTP: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		},
		verbose:     opts.Verbose,
		debug:       opts.Debug,
		transport:   transport,
		queryIDs:    map[string]string{},
		authSession: session,
		txProvider:  txProvider,
		traceWriter: newTxIDTraceWriter(traceFile, traceMode, traceOps),
	}
}

func (c *Client) Status(version string) model.ProjectStatus {
	implemented := []string{
		"config loading",
		"output formatting",
		"go-native browser/env auth loading",
		"guest token transport",
		"feed",
		"bookmarks",
		"list",
		"user lookup",
		"user-posts",
		"likes",
		"followers/following",
		"tweet fallback via syndication",
		"browser-backed search",
		"verified browser-backed write actions",
		"remote-debug browser attach",
	}
	planned := []string{
		"native authenticated API transport",
		"streamlined remote-debug UX",
		"browser-backed integration tests",
	}
	capabilities := []string{
		"status",
		"feed",
		"favorites",
		"list",
		"user",
		"user-posts",
		"likes",
		"followers",
		"following",
		"tweet",
		"search",
		"like/unlike",
		"bookmark/unbookmark",
		"retweet/unretweet",
		"post/delete",
	}

	return model.ProjectStatus{
		Name:         "X-CLI",
		Binary:       "x",
		Version:      version,
		Module:       "github.com/dl-alexandre/X-CLI",
		Implemented:  implemented,
		Planned:      planned,
		Capabilities: capabilities,
	}
}

func (c *Client) Doctor() model.DoctorReport {
	checks := []model.DoctorCheck{}

	authStatus := "missing"
	authDetails := "no auth session loaded"
	if c.authSession != nil {
		authStatus = "ok"
		authDetails = "browser=" + fallbackString(c.authSession.Browser, "unknown")
	}
	checks = append(checks, model.DoctorCheck{Name: "auth", Status: authStatus, Details: authDetails})

	debugURL, err := c.remoteDebugWebSocketURL()
	if err == nil && debugURL != "" {
		checks = append(checks, model.DoctorCheck{Name: "remote-debug", Status: "ok", Details: debugURL})
	} else {
		checks = append(checks, model.DoctorCheck{Name: "remote-debug", Status: "warn", Details: "no live DevTools endpoint found"})
	}

	tlsDetails := "uTLS Chrome impersonation enabled"
	if c.transport == nil || c.transport.DialTLSContext == nil {
		tlsDetails = "native TLS impersonation not configured"
	}
	checks = append(checks, model.DoctorCheck{Name: "tls-transport", Status: ternaryStatus(c.transport != nil && c.transport.DialTLSContext != nil), Details: tlsDetails})

	txStatus := "warn"
	txDetails := c.txProvider.Name() + ": pending native implementation"
	if c.txProvider.Ready() {
		txStatus = "ok"
		txDetails = c.txProvider.Name()
		// Check if we have static salt configured
		if nativeProvider, ok := c.txProvider.(*NativeTransactionProvider); ok {
			if nativeProvider.HasStaticSalt() {
				txDetails += " + static-salt"
			}
		}
	}
	checks = append(checks, model.DoctorCheck{Name: "transaction-id", Status: txStatus, Details: txDetails})

	traceStatus := "warn"
	traceDetails := "disabled"
	if c.traceWriter != nil && c.traceWriter.Enabled() {
		traceStatus = "ok"
		traceDetails = c.traceWriter.path + " (mode=" + c.traceWriter.mode + ")"
		if len(c.traceWriter.ops) > 0 {
			targets := make([]string, 0, len(c.traceWriter.ops))
			for op := range c.traceWriter.ops {
				targets = append(targets, op)
			}
			sort.Strings(targets)
			traceDetails += " ops=" + strings.Join(targets, ",")
		}
	}
	checks = append(checks, model.DoctorCheck{Name: "txid-trace", Status: traceStatus, Details: traceDetails})

	writeStatus := "warn"
	writeDetails := "remote-debug recommended for post/delete"
	if err == nil && debugURL != "" {
		writeStatus = "ok"
		writeDetails = "remote-debug write path available"
	}
	checks = append(checks, model.DoctorCheck{Name: "writes", Status: writeStatus, Details: writeDetails})

	return model.DoctorReport{
		Name:      "X-CLI Doctor",
		Checks:    checks,
		Transport: "native uTLS + browser fallback",
		Auth:      authStatus,
	}
}

func (c *Client) Feed(feedType string, count int) (model.TimelineResult, error) {
	if !strings.EqualFold(feedType, "following") {
		return c.browserTimeline("https://x.com/home", count, nil, "feed")
	}

	return c.browserTimeline("https://x.com/home", count, func(ctx context.Context) error {
		if err := chromedp.WaitVisible(`article`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Click(`//span[text()="Following"]`, chromedp.BySearch).Do(ctx); err != nil {
			return err
		}
		return chromedp.Sleep(2 * time.Second).Do(ctx)
	}, "feed")
}

func (c *Client) Favorites(count int) (model.TimelineResult, error) {
	return c.browserTimeline("https://x.com/i/bookmarks", count, nil, "favorites")
}

func (c *Client) Search(query string, product string, count int) (model.TimelineResult, error) {
	if c.authSession != nil {
		return c.browserSearch(query, product, count)
	}

	if count <= 0 {
		count = 20
	}

	variables := map[string]any{
		"rawQuery":    query,
		"count":       count,
		"querySource": "typed_query",
		"product":     product,
	}

	payload, err := c.graphQLGet("SearchTimeline", variables, defaultFeatures, nil, "https://x.com/search")
	if err != nil {
		return model.TimelineResult{}, err
	}

	root, ok := payload.(map[string]any)
	if !ok {
		return model.TimelineResult{}, errors.New("unexpected search response")
	}

	instructions := deepGet(root, "data", "search_by_raw_query", "search_timeline", "timeline", "instructions")
	tweets := parseTimelineTweets(instructions, count)
	return model.TimelineResult{Tweets: tweets, Source: "search"}, nil
}

func (c *Client) Tweet(id string, count int) (model.TweetThread, error) {
	if count <= 0 {
		count = 20
	}

	variables := map[string]any{
		"focalTweetId":                           id,
		"referrer":                               "tweet",
		"with_rux_injections":                    false,
		"includePromotedContent":                 true,
		"rankingMode":                            "Relevance",
		"withCommunity":                          true,
		"withQuickPromoteEligibilityTweetFields": true,
		"withBirdwatchNotes":                     true,
		"withVoice":                              true,
		"count":                                  count,
	}
	fieldToggles := map[string]any{
		"withArticleRichContentState": true,
		"withArticlePlainText":        false,
		"withGrokAnalyze":             false,
		"withDisallowedReplyControls": false,
	}

	payload, err := c.graphQLGet("TweetDetail", variables, defaultFeatures, fieldToggles, "https://x.com/i/status/"+id)
	if err != nil {
		fallbackTweet, fallbackErr := c.fetchSyndicationTweet(id)
		if fallbackErr != nil {
			return model.TweetThread{}, err
		}
		return model.TweetThread{Tweet: fallbackTweet}, nil
	}

	root, ok := payload.(map[string]any)
	if !ok {
		return model.TweetThread{}, errors.New("unexpected tweet response")
	}

	instructions := deepGet(root, "data", "threaded_conversation_with_injections_v2", "instructions")
	if instructions == nil {
		instructions = deepGet(root, "data", "tweetResult", "result", "timeline", "instructions")
	}
	tweets := parseTimelineTweets(instructions, count+1)
	if len(tweets) == 0 {
		return model.TweetThread{}, errors.New("tweet not found")
	}

	return model.TweetThread{Tweet: tweets[0], Replies: tweets[1:]}, nil
}

func (c *Client) List(listID string, count int) (model.TimelineResult, error) {
	return c.browserTimeline("https://x.com/i/lists/"+url.PathEscape(listID), count, nil, "list")
}

func (c *Client) User(screenName string) (model.UserProfile, error) {
	variables := map[string]any{
		"screen_name":              screenName,
		"withSafetyModeUserFields": true,
	}
	features := map[string]bool{
		"hidden_profile_subscriptions_enabled":                              true,
		"rweb_tipjar_consumption_enabled":                                   true,
		"responsive_web_graphql_exclude_directive_enabled":                  true,
		"verified_phone_label_enabled":                                      false,
		"subscriptions_verification_info_is_identity_verified_enabled":      true,
		"subscriptions_verification_info_verified_since_enabled":            true,
		"highlights_tweets_tab_ui_enabled":                                  true,
		"responsive_web_twitter_article_notes_tab_enabled":                  true,
		"subscriptions_feature_can_gift_premium":                            true,
		"creator_subscriptions_tweet_preview_api_enabled":                   true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled": false,
		"responsive_web_graphql_timeline_navigation_enabled":                true,
	}

	payload, err := c.graphQLGet("UserByScreenName", variables, features, nil, "https://x.com/"+screenName)
	if err != nil {
		return model.UserProfile{}, err
	}

	root, ok := payload.(map[string]any)
	if !ok {
		return model.UserProfile{}, errors.New("unexpected user response")
	}

	result, ok := deepGet(root, "data", "user", "result").(map[string]any)
	if !ok || len(result) == 0 {
		return model.UserProfile{}, fmt.Errorf("user @%s not found", screenName)
	}

	return parseUserProfile(result), nil
}

func (c *Client) UserPosts(screenName string, count int) (model.TimelineResult, error) {
	if count <= 0 {
		count = 20
	}

	user, err := c.User(screenName)
	if err != nil {
		return model.TimelineResult{}, err
	}

	variables := map[string]any{
		"userId":                                 user.ID,
		"count":                                  count,
		"includePromotedContent":                 false,
		"latestControlAvailable":                 true,
		"requestContext":                         "launch",
		"withQuickPromoteEligibilityTweetFields": true,
		"withVoice":                              true,
		"withV2Timeline":                         true,
	}

	payload, err := c.graphQLGet("UserTweets", variables, defaultFeatures, nil, "https://x.com/"+screenName)
	if err != nil {
		return model.TimelineResult{}, err
	}

	root, ok := payload.(map[string]any)
	if !ok {
		return model.TimelineResult{}, errors.New("unexpected user-posts response")
	}

	instructions := deepGet(root, "data", "user", "result", "timeline_v2", "timeline", "instructions")
	if instructions == nil {
		instructions = deepGet(root, "data", "user", "result", "timeline", "instructions")
	}
	if instructions == nil {
		instructions = deepGet(root, "data", "user", "result", "timeline", "timeline", "instructions")
	}
	tweets := parseTimelineTweets(instructions, count)
	return model.TimelineResult{Tweets: tweets, Source: "user-posts"}, nil
}

func (c *Client) Likes(screenName string, count int) (model.TimelineResult, error) {
	return c.browserTimeline("https://x.com/"+url.PathEscape(screenName)+"/likes", count, nil, "likes")
}

func (c *Client) Followers(screenName string, count int) ([]model.UserProfile, error) {
	return c.browserUsers("https://x.com/"+url.PathEscape(screenName)+"/followers", count)
}

func (c *Client) Following(screenName string, count int) ([]model.UserProfile, error) {
	return c.browserUsers("https://x.com/"+url.PathEscape(screenName)+"/following", count)
}

func (c *Client) CreatePost(text string) (model.ActionResult, error) {
	if strings.TrimSpace(text) == "" {
		return model.ActionResult{}, errors.New("post text cannot be empty")
	}

	var profileURL string
	mutation, err := c.runBrowserMutation("CreateTweet", func(ctx context.Context) error {
		if err := chromedp.Navigate("https://x.com/home").Do(ctx); err != nil {
			return err
		}
		if err := chromedp.WaitVisible(`[data-testid="tweetTextarea_0"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.AttributeValue(`[data-testid="AppTabBar_Profile_Link"]`, "href", &profileURL, nil, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.SendKeys(`[data-testid="tweetTextarea_0"]`, text, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Sleep(1200 * time.Millisecond).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Click(`[data-testid="tweetButtonInline"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		return chromedp.Sleep(3 * time.Second).Do(ctx)
	})
	if err != nil {
		return model.ActionResult{}, err
	}

	body, ok := mutation.Payload.(map[string]any)
	if !ok {
		return model.ActionResult{}, errors.New("create tweet returned an unexpected response")
	}
	tweetID := stringValue(deepGet(body, "data", "create_tweet", "tweet_results", "result", "rest_id"))
	if tweetID == "" {
		tweetID = stringValue(deepGet(body, "data", "create_tweet", "tweet_results", "result", "tweet", "rest_id"))
	}
	if tweetID == "" {
		tweetID = stringValue(deepGet(body, "data", "create_tweet", "tweet_results", "result", "legacy", "id_str"))
	}
	if tweetID == "" {
		tweetID = stringValue(deepGet(body, "data", "create_tweet", "tweet_results", "result", "tweet", "legacy", "id_str"))
	}
	if tweetID == "" {
		tweetID = stringValue(deepGet(body, "data", "create_tweet", "tweet_result", "result", "rest_id"))
	}
	if tweetID == "" {
		tweetID = findTweetID(body)
	}
	if tweetID != "" {
		return model.ActionResult{Action: "post", Success: true, URL: tweetURL(tweetID), Message: "post created"}, nil
	}

	if profileURL != "" {
		if createdURL, confirmErr := c.confirmPostedTweet("https://x.com"+profileURL, firstLine(text)); confirmErr == nil && createdURL != "" {
			return model.ActionResult{Action: "post", Success: true, URL: createdURL, Message: "post created"}, nil
		}
	}

	payloadBytes, _ := json.Marshal(mutation.Payload)
	if mutation.Status == 200 {
		return model.ActionResult{}, fmt.Errorf("post submitted but could not confirm tweet id from payload: %s", truncateForError(string(payloadBytes)))
	}

	return model.ActionResult{}, fmt.Errorf("post was not confirmed by X (status=%d payload=%s)", mutation.Status, truncateForError(string(payloadBytes)))
}

func (c *Client) DeletePost(id string) (model.ActionResult, error) {
	mutation, err := c.runBrowserMutation("DeleteTweet", func(ctx context.Context) error {
		if err := chromedp.Navigate(tweetURL(id)).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.WaitVisible(`[data-testid="caret"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Click(`[data-testid="caret"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.WaitVisible(`//span[text()="Delete"]`, chromedp.BySearch).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Click(`//span[text()="Delete"]`, chromedp.BySearch).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.WaitVisible(`[data-testid="confirmationSheetConfirm"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Click(`[data-testid="confirmationSheetConfirm"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		return chromedp.Sleep(2 * time.Second).Do(ctx)
	})
	if err != nil {
		return model.ActionResult{}, err
	}
	if mutation.Status != 200 {
		return model.ActionResult{}, errors.New("delete was not confirmed by X")
	}
	return model.ActionResult{Action: "delete", Target: id, Success: true, Message: "post deleted"}, nil
}

func (c *Client) LikePost(id string) (model.ActionResult, error) {
	return c.browserTweetAction(id, `[data-testid="like"]`, `[data-testid="unlike"]`, "like", "post liked")
}

func (c *Client) UnlikePost(id string) (model.ActionResult, error) {
	return c.browserTweetAction(id, `[data-testid="unlike"]`, `[data-testid="like"]`, "unlike", "post unliked")
}

func (c *Client) RetweetPost(id string) (model.ActionResult, error) {
	err := c.runBrowserFlow(func(ctx context.Context) error {
		if err := chromedp.Navigate(tweetURL(id)).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.WaitVisible(`[data-testid="retweet"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Click(`[data-testid="retweet"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.WaitVisible(`[data-testid="retweetConfirm"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Click(`[data-testid="retweetConfirm"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Sleep(1500 * time.Millisecond).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Navigate(tweetURL(id)).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Sleep(1500 * time.Millisecond).Do(ctx); err != nil {
			return err
		}
		return chromedp.WaitVisible(`[data-testid="unretweet"]`, chromedp.ByQuery).Do(ctx)
	})
	if err != nil {
		return model.ActionResult{}, err
	}
	return model.ActionResult{Action: "retweet", Target: id, Success: true, Message: "post retweeted"}, nil
}

func (c *Client) UnretweetPost(id string) (model.ActionResult, error) {
	err := c.runBrowserFlow(func(ctx context.Context) error {
		if err := chromedp.Navigate(tweetURL(id)).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.WaitVisible(`[data-testid="unretweet"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Click(`[data-testid="unretweet"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.WaitVisible(`[data-testid="unretweetConfirm"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Click(`[data-testid="unretweetConfirm"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Sleep(1500 * time.Millisecond).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Navigate(tweetURL(id)).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Sleep(1500 * time.Millisecond).Do(ctx); err != nil {
			return err
		}
		return chromedp.WaitVisible(`[data-testid="retweet"]`, chromedp.ByQuery).Do(ctx)
	})
	if err != nil {
		return model.ActionResult{}, err
	}
	return model.ActionResult{Action: "unretweet", Target: id, Success: true, Message: "retweet removed"}, nil
}

func (c *Client) BookmarkPost(id string) (model.ActionResult, error) {
	err := c.runBrowserFlow(func(ctx context.Context) error {
		if err := chromedp.Navigate(tweetURL(id)).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.WaitVisible(`[data-testid="bookmark"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Click(`[data-testid="bookmark"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Sleep(1500 * time.Millisecond).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Navigate("https://x.com/i/bookmarks").Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Sleep(2500 * time.Millisecond).Do(ctx); err != nil {
			return err
		}
		var exists bool
		if err := chromedp.Evaluate(fmt.Sprintf(`document.querySelector('a[href*="/status/%s"]') !== null`, id), &exists).Do(ctx); err != nil {
			return err
		}
		if !exists {
			return errors.New("bookmark was not confirmed by X")
		}
		return nil
	})
	if err != nil {
		return model.ActionResult{}, err
	}
	return model.ActionResult{Action: "bookmark", Target: id, Success: true, Message: "post bookmarked"}, nil
}

func (c *Client) UnbookmarkPost(id string) (model.ActionResult, error) {
	err := c.runBrowserFlow(func(ctx context.Context) error {
		if err := chromedp.Navigate(tweetURL(id)).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.WaitVisible(`[data-testid="removeBookmark"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Click(`[data-testid="removeBookmark"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Sleep(1500 * time.Millisecond).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Navigate("https://x.com/i/bookmarks").Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Sleep(2500 * time.Millisecond).Do(ctx); err != nil {
			return err
		}
		var exists bool
		if err := chromedp.Evaluate(fmt.Sprintf(`document.querySelector('a[href*="/status/%s"]') !== null`, id), &exists).Do(ctx); err != nil {
			return err
		}
		if exists {
			return errors.New("bookmark removal was not confirmed by X")
		}
		return nil
	})
	if err != nil {
		return model.ActionResult{}, err
	}
	return model.ActionResult{Action: "unbookmark", Target: id, Success: true, Message: "bookmark removed"}, nil
}

func (c *Client) graphQLGet(operation string, variables map[string]any, features map[string]bool, fieldToggles map[string]any, referer string) (any, error) {
	queryID, err := c.resolveQueryID(operation)
	if err != nil {
		return nil, err
	}

	urlString, err := buildGraphQLURL(queryID, operation, variables, features, fieldToggles)
	if err != nil {
		return nil, err
	}

	payload, status, err := c.getJSON(urlString, referer)
	if err == nil {
		return payload, nil
	}

	if status == http.StatusNotFound {
		if refreshErr := c.scanQueryIDs(); refreshErr == nil {
			freshID, freshErr := c.resolveQueryID(operation)
			if freshErr == nil && freshID != queryID {
				urlString, freshErr = buildGraphQLURL(freshID, operation, variables, features, fieldToggles)
				if freshErr == nil {
					freshPayload, _, freshGetErr := c.getJSON(urlString, referer)
					return freshPayload, freshGetErr
				}
			}
		}
	}

	return nil, err
}

func (c *Client) getJSON(urlString string, referer string) (any, int, error) {
	body, status, err := c.doRequest(http.MethodGet, urlString, nil, referer)
	if err != nil {
		return nil, status, err
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, status, fmt.Errorf("decode response: %w", err)
	}

	if err := parseAPIError(payload); err != nil {
		return nil, status, err
	}

	return payload, status, nil
}

func (c *Client) doRequest(method string, urlString string, body []byte, referer string) ([]byte, int, error) {
	guestToken, err := c.ensureGuestToken()
	if err != nil {
		return nil, 0, err
	}

	headers := map[string]string{
		"Authorization":             "Bearer " + bearerToken,
		"X-Guest-Token":             guestToken,
		"User-Agent":                c.userAgent(),
		"Accept":                    "*/*",
		"Accept-Language":           "en-US,en;q=0.9",
		"Origin":                    "https://x.com",
		"Referer":                   referer,
		"X-Twitter-Active-User":     "yes",
		"X-Twitter-Client-Language": "en",
	}

	return c.rawRequest(method, urlString, headers, body)
}

func (c *Client) ensureGuestToken() (string, error) {
	c.guestTokenMu.Lock()
	defer c.guestTokenMu.Unlock()

	if c.guestToken != "" {
		return c.guestToken, nil
	}

	body, status, err := c.rawRequest(http.MethodPost, "https://api.twitter.com/1.1/guest/activate.json", map[string]string{
		"Authorization": "Bearer " + bearerToken,
		"User-Agent":    c.userAgent(),
		"Accept":        "*/*",
	}, []byte{})
	if err != nil {
		return "", fmt.Errorf("guest activate failed %d: %w", status, err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}

	token, _ := payload["guest_token"].(string)
	if token == "" {
		return "", errors.New("guest token missing from response")
	}

	c.guestToken = token
	return token, nil
}

func (c *Client) resolveQueryID(operation string) (string, error) {
	c.queryMu.Lock()
	defer c.queryMu.Unlock()

	if queryID := c.queryIDs[operation]; queryID != "" {
		return queryID, nil
	}
	if queryID := fallbackQueryIDs[operation]; queryID != "" {
		return queryID, nil
	}
	return "", fmt.Errorf("no query id available for %s", operation)
}

func (c *Client) scanQueryIDs() error {
	homeHTML, _, err := c.rawRequest(http.MethodGet, "https://x.com", map[string]string{
		"User-Agent":      c.userAgent(),
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Accept-Language": "en-US,en;q=0.9",
	}, nil)
	if err != nil {
		return err
	}

	scriptPattern := regexp.MustCompile(`https://abs\.twimg\.com/responsive-web/client-web[^"']+\.js`)
	scriptURLs := scriptPattern.FindAllString(string(homeHTML), -1)
	if len(scriptURLs) == 0 {
		return errors.New("no x bundles found")
	}

	queryPattern := regexp.MustCompile(`queryId:\s*"([A-Za-z0-9_-]+)"[^}]{0,200}operationName:\s*"([^"]+)"`)
	updated := false

	for _, scriptURL := range scriptURLs {
		body, status, reqErr := c.rawRequest(http.MethodGet, scriptURL, map[string]string{
			"User-Agent": c.userAgent(),
			"Referer":    "https://x.com/",
			"Accept":     "*/*",
		}, nil)
		if reqErr != nil || status >= 400 {
			continue
		}

		matches := queryPattern.FindAllStringSubmatch(string(body), -1)
		if len(matches) == 0 {
			continue
		}

		c.queryMu.Lock()
		for _, match := range matches {
			if len(match) == 3 {
				c.queryIDs[match[2]] = match[1]
				updated = true
			}
		}
		c.queryMu.Unlock()
	}

	if !updated {
		return errors.New("no query ids extracted from x bundles")
	}

	return nil
}

func (c *Client) userAgent() string {
	if c.config != nil && c.config.HTTP.UserAgent != "" && c.config.HTTP.UserAgent != "x/dev" {
		return c.config.HTTP.UserAgent
	}
	return defaultUA
}

func buildGraphQLURL(queryID string, operation string, variables map[string]any, features map[string]bool, fieldToggles map[string]any) (string, error) {
	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return "", err
	}

	compactFeatures := map[string]bool{}
	for key, value := range features {
		if value {
			compactFeatures[key] = true
		}
	}
	featuresJSON, err := json.Marshal(compactFeatures)
	if err != nil {
		return "", err
	}

	urlString := fmt.Sprintf(
		"https://x.com/i/api/graphql/%s/%s?variables=%s&features=%s",
		queryID,
		operation,
		url.QueryEscape(string(variablesJSON)),
		url.QueryEscape(string(featuresJSON)),
	)

	if len(fieldToggles) > 0 {
		fieldJSON, marshalErr := json.Marshal(fieldToggles)
		if marshalErr != nil {
			return "", marshalErr
		}
		urlString += "&fieldToggles=" + url.QueryEscape(string(fieldJSON))
	}

	return urlString, nil
}

func parseUserProfile(result map[string]any) model.UserProfile {
	legacy, _ := result["legacy"].(map[string]any)
	core, _ := result["core"].(map[string]any)

	return model.UserProfile{
		ID:             stringValue(result["rest_id"]),
		Name:           fallbackString(stringValue(core["name"]), stringValue(legacy["name"])),
		ScreenName:     fallbackString(stringValue(core["screen_name"]), stringValue(legacy["screen_name"])),
		Bio:            stringValue(legacy["description"]),
		Location:       fallbackString(stringValue(result["location"]), stringValue(legacy["location"])),
		URL:            stringValue(deepGet(legacy, "entities", "url", "urls", 0, "expanded_url")),
		FollowersCount: intValue(legacy["followers_count"]),
		FollowingCount: intValue(legacy["friends_count"]),
		TweetsCount:    intValue(legacy["statuses_count"]),
		LikesCount:     intValue(legacy["favourites_count"]),
		Verified:       boolValue(result["is_blue_verified"]) || boolValue(legacy["verified"]),
	}
}

func parseTimelineTweets(raw any, limit int) []model.Tweet {
	instructions, ok := raw.([]any)
	if !ok {
		return nil
	}

	seen := map[string]bool{}
	tweets := make([]model.Tweet, 0, limit)

	for _, instruction := range instructions {
		instructionMap, ok := instruction.(map[string]any)
		if !ok {
			continue
		}

		entries, _ := instructionMap["entries"].([]any)
		for _, entry := range entries {
			entryMap, ok := entry.(map[string]any)
			if !ok {
				continue
			}

			content, _ := entryMap["content"].(map[string]any)
			collectTweets(content, seen, &tweets, limit)
			if len(tweets) >= limit {
				return tweets[:limit]
			}
		}

		entry, _ := instructionMap["entry"].(map[string]any)
		if len(entry) > 0 {
			content, _ := entry["content"].(map[string]any)
			collectTweets(content, seen, &tweets, limit)
			if len(tweets) >= limit {
				return tweets[:limit]
			}
		}
	}

	return tweets
}

func collectTweets(content map[string]any, seen map[string]bool, tweets *[]model.Tweet, limit int) {
	if content == nil {
		return
	}

	if result, ok := deepGet(content, "itemContent", "tweet_results", "result").(map[string]any); ok {
		appendTweet(result, seen, tweets, limit)
	}

	if items, ok := content["items"].([]any); ok {
		for _, item := range items {
			itemMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if result, ok := deepGet(itemMap, "item", "itemContent", "tweet_results", "result").(map[string]any); ok {
				appendTweet(result, seen, tweets, limit)
			}
		}
	}
}

func appendTweet(result map[string]any, seen map[string]bool, tweets *[]model.Tweet, limit int) {
	if len(*tweets) >= limit {
		return
	}

	tweet, ok := parseTweet(result)
	if !ok || tweet.ID == "" || seen[tweet.ID] {
		return
	}

	seen[tweet.ID] = true
	*tweets = append(*tweets, tweet)
}

func parseTweet(result map[string]any) (model.Tweet, bool) {
	tweetData := result
	if typename, _ := result["__typename"].(string); typename == "TweetWithVisibilityResults" {
		if nested, ok := result["tweet"].(map[string]any); ok {
			tweetData = nested
		}
	}

	legacy, ok := tweetData["legacy"].(map[string]any)
	if !ok {
		return model.Tweet{}, false
	}
	core, ok := tweetData["core"].(map[string]any)
	if !ok {
		return model.Tweet{}, false
	}

	userResult, _ := deepGet(core, "user_results", "result").(map[string]any)
	userLegacy, _ := userResult["legacy"].(map[string]any)

	isRetweet := deepGet(legacy, "retweeted_status_result", "result") != nil
	actualData := tweetData
	actualLegacy := legacy
	actualUser := userResult
	actualUserLegacy := userLegacy
	retweetedBy := ""

	if isRetweet {
		retweetedBy = stringValue(userLegacy["screen_name"])
		if retweetResult, ok := deepGet(legacy, "retweeted_status_result", "result").(map[string]any); ok {
			if typename, _ := retweetResult["__typename"].(string); typename == "TweetWithVisibilityResults" {
				if nested, nestedOK := retweetResult["tweet"].(map[string]any); nestedOK {
					retweetResult = nested
				}
			}
			if retweetLegacy, legacyOK := retweetResult["legacy"].(map[string]any); legacyOK {
				if retweetCore, coreOK := retweetResult["core"].(map[string]any); coreOK {
					actualData = retweetResult
					actualLegacy = retweetLegacy
					if nestedUser, userOK := deepGet(retweetCore, "user_results", "result").(map[string]any); userOK {
						actualUser = nestedUser
						if nestedUserLegacy, legacyOK := nestedUser["legacy"].(map[string]any); legacyOK {
							actualUserLegacy = nestedUserLegacy
						}
					}
				}
			}
		}
	}

	metrics := model.TweetMetrics{
		Likes:     intValue(actualLegacy["favorite_count"]),
		Retweets:  intValue(actualLegacy["retweet_count"]),
		Replies:   intValue(actualLegacy["reply_count"]),
		Quotes:    intValue(actualLegacy["quote_count"]),
		Views:     intValue(deepGet(actualData, "views", "count")),
		Bookmarks: intValue(actualLegacy["bookmark_count"]),
	}

	urls := []string{}
	if entities, ok := deepGet(actualLegacy, "entities", "urls").([]any); ok {
		for _, item := range entities {
			itemMap, mapOK := item.(map[string]any)
			if !mapOK {
				continue
			}
			if expanded := stringValue(itemMap["expanded_url"]); expanded != "" {
				urls = append(urls, expanded)
			}
		}
	}

	return model.Tweet{
		ID:        stringValue(actualData["rest_id"]),
		Text:      stringValue(actualLegacy["full_text"]),
		CreatedAt: stringValue(actualLegacy["created_at"]),
		Lang:      stringValue(actualLegacy["lang"]),
		URLs:      urls,
		Author: model.Author{
			ID:              stringValue(actualUser["rest_id"]),
			Name:            fallbackString(stringValue(deepGet(actualUser, "core", "name")), stringValue(actualUserLegacy["name"])),
			ScreenName:      fallbackString(stringValue(deepGet(actualUser, "core", "screen_name")), stringValue(actualUserLegacy["screen_name"])),
			ProfileImageURL: fallbackString(stringValue(deepGet(actualUser, "avatar", "image_url")), stringValue(actualUserLegacy["profile_image_url_https"])),
			Verified:        boolValue(actualUser["is_blue_verified"]) || boolValue(actualUserLegacy["verified"]),
		},
		Metrics:     metrics,
		IsRetweet:   isRetweet,
		RetweetedBy: retweetedBy,
	}, true
}

func parseAPIError(payload any) error {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil
	}
	if errorsList, ok := root["errors"].([]any); ok && len(errorsList) > 0 {
		first, _ := errorsList[0].(map[string]any)
		message := stringValue(first["message"])
		if message == "" {
			message = "unknown x api error"
		}
		return errors.New(message)
	}
	return nil
}

func deepGet(value any, path ...any) any {
	current := value
	for _, part := range path {
		switch key := part.(type) {
		case string:
			currentMap, ok := current.(map[string]any)
			if !ok {
				return nil
			}
			current = currentMap[key]
		case int:
			currentList, ok := current.([]any)
			if !ok || key < 0 || key >= len(currentList) {
				return nil
			}
			current = currentList[key]
		default:
			return nil
		}
	}
	return current
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

func fallbackString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstLine(value string) string {
	parts := strings.Split(strings.TrimSpace(value), "\n")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func ternaryStatus(ok bool) string {
	if ok {
		return "ok"
	}
	return "warn"
}

func findTweetID(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range []string{"rest_id", "id_str", "tweet_id"} {
			candidate := stringValue(typed[key])
			if looksLikeTweetID(candidate) {
				return candidate
			}
		}
		for _, nested := range typed {
			if candidate := findTweetID(nested); candidate != "" {
				return candidate
			}
		}
	case []any:
		for _, nested := range typed {
			if candidate := findTweetID(nested); candidate != "" {
				return candidate
			}
		}
	}
	return ""
}

func looksLikeTweetID(value string) bool {
	if len(value) < 8 {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func boolValue(value any) bool {
	typed, ok := value.(bool)
	return ok && typed
}

func intValue(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(math.Round(typed))
	case int:
		return typed
	case int64:
		return int(typed)
	case json.Number:
		i, _ := typed.Int64()
		return int(i)
	case string:
		var asNumber json.Number = json.Number(strings.TrimSpace(typed))
		if i, err := asNumber.Int64(); err == nil {
			return int(i)
		}
	}
	return 0
}

func int64Value(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(math.Round(typed))
	case int:
		return int64(typed)
	case int64:
		return typed
	case json.Number:
		i, _ := typed.Int64()
		return i
	case string:
		var asNumber json.Number = json.Number(strings.TrimSpace(typed))
		if i, err := asNumber.Int64(); err == nil {
			return i
		}
	}
	return 0
}

func truncateForError(text string) string {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) <= 300 {
		return trimmed
	}
	return trimmed[:300] + "..."
}

func (c *Client) fetchSyndicationTweet(id string) (model.Tweet, error) {
	urlString := "https://cdn.syndication.twimg.com/tweet-result?token=x&id=" + url.QueryEscape(id)
	body, _, err := c.rawRequest(http.MethodGet, urlString, map[string]string{
		"User-Agent": c.userAgent(),
		"Referer":    "https://x.com/",
		"Accept":     "application/json,text/plain,*/*",
	}, nil)
	if err != nil {
		return model.Tweet{}, err
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return model.Tweet{}, err
	}

	if len(payload) == 0 {
		return model.Tweet{}, errors.New("empty syndication payload")
	}

	user, _ := payload["user"].(map[string]any)
	return model.Tweet{
		ID:        stringValue(payload["id_str"]),
		Text:      stringValue(payload["text"]),
		CreatedAt: stringValue(payload["created_at"]),
		Lang:      stringValue(payload["lang"]),
		Author: model.Author{
			ID:              stringValue(user["id_str"]),
			Name:            stringValue(user["name"]),
			ScreenName:      stringValue(user["screen_name"]),
			ProfileImageURL: stringValue(user["profile_image_url_https"]),
			Verified:        boolValue(user["is_blue_verified"]) || boolValue(user["verified"]),
		},
		Metrics: model.TweetMetrics{
			Likes:   intValue(payload["favorite_count"]),
			Replies: intValue(payload["conversation_count"]),
		},
	}, nil
}

func (c *Client) browserSearch(query string, product string, count int) (model.TimelineResult, error) {
	return c.browserTimeline(buildSearchURL(query, product), count, nil, "search")
}

type browserMutationResult struct {
	URL     string
	Status  int
	Payload any
}

type browserSession struct {
	ctx     context.Context
	cancel  context.CancelFunc
	cleanup func()
}

const mutationInterceptorScript = `(function() {
  if (window.__xcliMutationHookInstalled) return;
  window.__xcliMutationHookInstalled = true;
  window.__xcliMutations = window.__xcliMutations || [];

  const pushRecord = (url, status, body) => {
    try {
      window.__xcliMutations.push({
        url: String(url || ''),
        status: Number(status || 0),
        body: body == null ? '' : String(body),
        ts: Date.now(),
      });
    } catch (_) {}
  };

  const originalFetch = window.fetch;
  window.fetch = async function(...args) {
    const response = await originalFetch.apply(this, args);
    try {
      const clone = response.clone();
      clone.text().then((text) => pushRecord(clone.url, clone.status, text)).catch(() => {});
    } catch (_) {}
    return response;
  };

  const originalOpen = XMLHttpRequest.prototype.open;
  const originalSend = XMLHttpRequest.prototype.send;
  XMLHttpRequest.prototype.open = function(method, url) {
    this.__xcliURL = url;
    return originalOpen.apply(this, arguments);
  };
  XMLHttpRequest.prototype.send = function() {
    this.addEventListener('loadend', function() {
      try {
        pushRecord(this.__xcliURL || this.responseURL, this.status, this.responseText || '');
      } catch (_) {}
    });
    return originalSend.apply(this, arguments);
  };
})();`

const txidInterceptorScript = `(function() {
  if (window.__xcliTxidHookInstalled) return;
  window.__xcliTxidHookInstalled = true;
  window.__xcliTxids = window.__xcliTxids || [];

  const pushRecord = (url, headers) => {
    try {
      const txid = headers['x-client-transaction-id'] || headers['X-Client-Transaction-Id'] || '';
      if (!txid) return;
      window.__xcliTxids.push({
        url: String(url || ''),
        txid: String(txid),
        method: String((headers[':method'] || headers['method'] || headers['__method'] || '') || ''),
        csrf: String((headers['x-csrf-token'] || headers['X-Csrf-Token'] || '') || ''),
        authorization: String((headers['authorization'] || headers['Authorization'] || '') || ''),
        ts: Date.now(),
      });
    } catch (_) {}
  };

  const toHeaderMap = (value) => {
    const out = {};
    if (!value) return out;
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
    return out;
  };

  const originalFetch = window.fetch;
  window.fetch = function(input, init) {
    try {
      const url = typeof input === 'string' ? input : (input && input.url) || '';
      const headers = toHeaderMap((init && init.headers) || (input && input.headers));
      headers['__method'] = (init && init.method) || (input && input.method) || 'GET';
      pushRecord(url, headers);
    } catch (_) {}
    return originalFetch.apply(this, arguments);
  };

  const originalOpen = XMLHttpRequest.prototype.open;
  const originalSetRequestHeader = XMLHttpRequest.prototype.setRequestHeader;
  XMLHttpRequest.prototype.open = function(method, url) {
    this.__xcliURL = url;
    this.__xcliHeaders = {};
    this.__xcliHeaders['__method'] = method || 'GET';
    return originalOpen.apply(this, arguments);
  };
  XMLHttpRequest.prototype.setRequestHeader = function(name, value) {
    try {
      this.__xcliHeaders = this.__xcliHeaders || {};
      this.__xcliHeaders[name] = value;
      pushRecord(this.__xcliURL || '', this.__xcliHeaders);
    } catch (_) {}
    return originalSetRequestHeader.apply(this, arguments);
  };
})();`

func (c *Client) runBrowserMutation(operation string, flow func(context.Context) error) (browserMutationResult, error) {
	session, err := c.newBrowserSession(75 * time.Second)
	if err != nil {
		return browserMutationResult{}, err
	}
	defer session.Close()

	err = chromedp.Run(session.ctx,
		network.Enable(),
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(mutationInterceptorScript).Do(ctx)
			return err
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(txidInterceptorScript).Do(ctx)
			return err
		}),
		chromedp.Navigate("https://x.com/home"),
		chromedp.Sleep(4*time.Second),
		chromedp.Evaluate(mutationInterceptorScript, nil),
		chromedp.Evaluate(txidInterceptorScript, nil),
		chromedp.ActionFunc(flow),
	)
	if err != nil {
		return browserMutationResult{}, err
	}
	_ = c.flushTXIDTrace(session.ctx, "browser-mutation")

	var rawRecords []map[string]any
	for i := 0; i < 16; i++ {
		err = chromedp.Run(session.ctx,
			chromedp.Evaluate(fmt.Sprintf(`(() => (window.__xcliMutations || []).filter((m) => (m.url || '').includes('/%s')) )()`, operation), &rawRecords),
		)
		if err != nil {
			return browserMutationResult{}, err
		}
		if len(rawRecords) > 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if len(rawRecords) == 0 {
		return browserMutationResult{}, fmt.Errorf("%s did not return a mutation response", operation)
	}

	last := rawRecords[len(rawRecords)-1]
	result := browserMutationResult{
		URL:    stringValue(last["url"]),
		Status: intValue(last["status"]),
	}
	bodyText := stringValue(last["body"])
	if bodyText != "" {
		var payload any
		if err := json.Unmarshal([]byte(bodyText), &payload); err == nil {
			result.Payload = payload
		} else {
			result.Payload = map[string]any{"raw": bodyText}
		}
	}
	if result.Status >= 400 {
		return browserMutationResult{}, fmt.Errorf("%s failed with status %d", operation, result.Status)
	}
	return result, nil
}

func (c *Client) runBrowserFlow(flow func(context.Context) error) error {
	session, err := c.newBrowserSession(60 * time.Second)
	if err != nil {
		return err
	}
	defer session.Close()

	err = chromedp.Run(session.ctx,
		network.Enable(),
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(txidInterceptorScript).Do(ctx)
			return err
		}),
		chromedp.Navigate("https://x.com/home"),
		chromedp.Sleep(4*time.Second),
		chromedp.Evaluate(txidInterceptorScript, nil),
		chromedp.ActionFunc(flow),
	)
	if err != nil {
		return err
	}
	return c.flushTXIDTrace(session.ctx, "browser-flow")
}

func (c *Client) newBrowserSession(timeout time.Duration) (*browserSession, error) {
	if wsURL, err := c.remoteDebugWebSocketURL(); err == nil && wsURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		allocCtx, allocCancel := chromedp.NewRemoteAllocator(ctx, wsURL)
		browserCtx, browserCancel := chromedp.NewContext(allocCtx)
		c.attachTraceListener(browserCtx, "remote-debug")
		return &browserSession{
			ctx:    browserCtx,
			cancel: cancel,
			cleanup: func() {
				browserCancel()
				allocCancel()
			},
		}, nil
	}

	profileRoot, err := cloneChromeProfile()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.UserDataDir(profileRoot),
		chromedp.Flag("profile-directory", "Default"),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
	)...)
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	c.attachTraceListener(browserCtx, "cloned-profile")

	return &browserSession{
		ctx:    browserCtx,
		cancel: cancel,
		cleanup: func() {
			browserCancel()
			allocCancel()
			_ = os.RemoveAll(profileRoot)
		},
	}, nil
}

func (c *Client) attachTraceListener(browserCtx context.Context, source string) {
	if c.traceWriter == nil || !c.traceWriter.Enabled() {
		return
	}
	requestMeta := map[network.RequestID]txIDTraceMeta{}
	chromedp.ListenTarget(browserCtx, func(ev any) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			if e.Request == nil {
				return
			}
			requestMeta[e.RequestID] = txIDTraceMeta{
				URL:         e.Request.URL,
				Method:      e.Request.Method,
				TimestampMS: time.Now().UnixMilli(),
				WallTime:    fmt.Sprint(e.WallTime),
			}
		case *network.EventRequestWillBeSentExtraInfo:
			meta := requestMeta[e.RequestID]
			_ = c.traceWriter.Record(meta, e.Headers, source)
		}
	})
}

func (c *Client) remoteDebugWebSocketURL() (string, error) {
	candidates := []string{}
	if c.config != nil && strings.TrimSpace(c.config.Browser.RemoteDebugURL) != "" {
		candidates = append(candidates, strings.TrimSpace(c.config.Browser.RemoteDebugURL))
	}
	candidates = append(candidates,
		"http://127.0.0.1:9222/json/version",
		"http://localhost:9222/json/version",
	)

	client := &http.Client{Timeout: 2 * time.Second}
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, "ws://") || strings.HasPrefix(candidate, "wss://") {
			return candidate, nil
		}
		resp, err := client.Get(candidate)
		if err != nil {
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil || resp.StatusCode >= 400 {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			continue
		}
		if wsURL := stringValue(payload["webSocketDebuggerUrl"]); wsURL != "" {
			return wsURL, nil
		}
	}

	return "", errors.New("no remote debug websocket available")
}

func (s *browserSession) Close() {
	if s == nil {
		return
	}
	if s.cleanup != nil {
		s.cleanup()
	}
	if s.cancel != nil {
		s.cancel()
	}
}

func (c *Client) flushTXIDTrace(browserCtx context.Context, source string) error {
	if c.traceWriter == nil || !c.traceWriter.Enabled() {
		return nil
	}
	var records []map[string]any
	if err := chromedp.Run(browserCtx,
		chromedp.Evaluate(`(() => window.__xcliTxids || [])()`, &records),
	); err != nil {
		return err
	}
	for _, record := range records {
		headers := network.Headers{}
		headers["x-client-transaction-id"] = stringValue(record["txid"])
		headers["x-csrf-token"] = stringValue(record["csrf"])
		headers["authorization"] = stringValue(record["authorization"])
		meta := txIDTraceMeta{
			URL:         stringValue(record["url"]),
			Method:      stringValue(record["method"]),
			TimestampMS: int64Value(record["ts"]),
		}
		if err := c.traceWriter.Record(meta, headers, source); err != nil {
			return err
		}
	}
	return chromedp.Run(browserCtx, chromedp.Evaluate(`window.__xcliTxids = []`, nil))
}

func (c *Client) confirmPostedTweet(profileURL string, needle string) (string, error) {
	var createdURL string
	err := c.runBrowserFlow(func(ctx context.Context) error {
		for i := 0; i < 6; i++ {
			if err := chromedp.Navigate(profileURL).Do(ctx); err != nil {
				return err
			}
			if err := chromedp.Sleep(2500 * time.Millisecond).Do(ctx); err != nil {
				return err
			}
			script := fmt.Sprintf(`(() => {
				const needle = %q;
				for (const article of document.querySelectorAll('article')) {
					const text = Array.from(article.querySelectorAll('div[lang]')).map(n => (n.innerText || '').trim()).filter(Boolean).join('\n');
					if (!text || !text.includes(needle)) continue;
					const link = article.querySelector('a[href*="/status/"]');
					if (link) return link.href;
				}
				return '';
			})()`, needle)
			if err := chromedp.Evaluate(script, &createdURL).Do(ctx); err != nil {
				return err
			}
			if createdURL != "" {
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return createdURL, nil
}

func (c *Client) browserTweetAction(id string, selector string, expectedAfter string, action string, message string) (model.ActionResult, error) {
	err := c.runBrowserFlow(func(ctx context.Context) error {
		if err := chromedp.Navigate(tweetURL(id)).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.WaitVisible(selector, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Click(selector, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Sleep(1500 * time.Millisecond).Do(ctx); err != nil {
			return err
		}
		return chromedp.WaitVisible(expectedAfter, chromedp.ByQuery).Do(ctx)
	})
	if err != nil {
		return model.ActionResult{}, err
	}
	return model.ActionResult{Action: action, Target: id, Success: true, Message: message}, nil
}

func (c *Client) browserTimeline(pageURL string, count int, afterNavigate func(context.Context) error, source string) (model.TimelineResult, error) {
	if c.authSession == nil || len(c.authSession.Cookies) == 0 {
		return model.TimelineResult{}, errors.New("authenticated browser session required")
	}
	if count <= 0 {
		count = 20
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
	)...)
	defer cancelAlloc()

	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)
	defer cancelBrowser()

	var rawTweets []struct {
		ID         string `json:"id"`
		ScreenName string `json:"screen_name"`
		Name       string `json:"name"`
		Text       string `json:"text"`
	}
	actions := []chromedp.Action{
		network.Enable(),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return setBrowserCookies(ctx, c.authSession.Cookies)
		}),
		chromedp.Navigate("https://x.com/home"),
		chromedp.Sleep(2 * time.Second),
		chromedp.Navigate(pageURL),
		chromedp.WaitVisible(`article`, chromedp.ByQuery),
	}
	if afterNavigate != nil {
		actions = append(actions, chromedp.ActionFunc(afterNavigate))
	}
	actions = append(actions,
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`(() => {
			const seen = new Set();
			const items = [];
			for (const article of document.querySelectorAll('article')) {
				let id = '';
				let screen = '';
				for (const link of article.querySelectorAll('a[href*="/status/"]')) {
					const href = link.getAttribute('href') || '';
					const match = href.match(/^\/([^\/]+)\/status\/(\d+)/);
					if (match) {
						screen = match[1];
						id = match[2];
						break;
					}
				}
				const nameNode = article.querySelector('[data-testid="User-Name"] span');
				const name = (nameNode?.innerText || '').trim();
				const text = Array.from(article.querySelectorAll('div[lang]')).map((n) => (n.innerText || '').trim()).filter(Boolean).join('\n');
				if (!id || !text || seen.has(id)) continue;
				seen.add(id);
				items.push({ id, screen_name: screen, name, text });
			}
			return items;
		})()`, &rawTweets),
	)

	err := chromedp.Run(browserCtx, actions...)
	if err != nil {
		return model.TimelineResult{}, fmt.Errorf("browser timeline failed: %w", err)
	}

	sort.Slice(rawTweets, func(i int, j int) bool { return rawTweets[i].ID > rawTweets[j].ID })
	if len(rawTweets) > count {
		rawTweets = rawTweets[:count]
	}

	tweets := make([]model.Tweet, 0, len(rawTweets))
	for _, raw := range rawTweets {
		tweets = append(tweets, model.Tweet{
			ID:   raw.ID,
			Text: raw.Text,
			Author: model.Author{
				ScreenName: raw.ScreenName,
				Name:       fallbackString(raw.Name, raw.ScreenName),
			},
		})
	}

	return model.TimelineResult{Tweets: tweets, Source: source}, nil
}

func (c *Client) browserUsers(pageURL string, count int) ([]model.UserProfile, error) {
	if c.authSession == nil || len(c.authSession.Cookies) == 0 {
		return nil, errors.New("authenticated browser session required")
	}
	if count <= 0 {
		count = 20
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
	)...)
	defer cancelAlloc()
	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)
	defer cancelBrowser()

	var rawUsers []struct {
		ScreenName string `json:"screen_name"`
		Name       string `json:"name"`
		Bio        string `json:"bio"`
	}

	err := chromedp.Run(browserCtx,
		network.Enable(),
		chromedp.ActionFunc(func(ctx context.Context) error { return setBrowserCookies(ctx, c.authSession.Cookies) }),
		chromedp.Navigate("https://x.com/home"),
		chromedp.Sleep(2*time.Second),
		chromedp.Navigate(pageURL),
		chromedp.WaitVisible(`[data-testid="UserCell"]`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`(() => {
			const seen = new Set();
			const items = [];
			for (const cell of document.querySelectorAll('[data-testid="UserCell"]')) {
				let screen = '';
				for (const link of cell.querySelectorAll('a[href^="/"]')) {
					const href = link.getAttribute('href') || '';
					const match = href.match(/^\/([^\/\?]+)$/);
					if (match && !['i','home','explore','search','messages','compose'].includes(match[1])) {
						screen = match[1];
						break;
					}
				}
				if (!screen || seen.has(screen)) continue;
				seen.add(screen);
				const spans = Array.from(cell.querySelectorAll('span')).map((n) => (n.innerText || '').trim()).filter(Boolean);
				const name = spans.find((s) => !s.startsWith('@')) || screen;
				const bio = Array.from(cell.querySelectorAll('div[dir="auto"]')).map((n) => (n.innerText || '').trim()).filter(Boolean).join('\n');
				items.push({ screen_name: screen, name, bio });
			}
			return items;
		})()`, &rawUsers),
	)
	if err != nil {
		return nil, fmt.Errorf("browser users failed: %w", err)
	}

	if len(rawUsers) > count {
		rawUsers = rawUsers[:count]
	}
	users := make([]model.UserProfile, 0, len(rawUsers))
	for _, raw := range rawUsers {
		users = append(users, model.UserProfile{ScreenName: raw.ScreenName, Name: raw.Name, Bio: raw.Bio})
	}
	return users, nil
}

func setBrowserCookies(ctx context.Context, cookies []*http.Cookie) error {
	seen := map[string]bool{}
	for _, cookie := range cookies {
		if cookie == nil || cookie.Name == "" || cookie.Value == "" {
			continue
		}
		domain := cookie.Domain
		if domain == "" {
			domain = ".x.com"
		}
		if !strings.Contains(domain, "x.com") {
			continue
		}
		key := domain + "|" + cookie.Name
		if seen[key] {
			continue
		}
		seen[key] = true

		err := network.SetCookie(cookie.Name, cookie.Value).
			WithDomain(domain).
			WithPath("/").
			WithSecure(true).
			Do(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func buildSearchURL(query string, product string) string {
	values := url.Values{}
	values.Set("q", query)
	values.Set("src", "typed_query")
	if filter := mapSearchFilter(product); filter != "" {
		values.Set("f", filter)
	}
	return "https://x.com/search?" + values.Encode()
}

func tweetURL(id string) string {
	return "https://x.com/i/status/" + url.PathEscape(id)
}

func cloneChromeProfile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	sourceRoot := filepath.Join(home, "Library", "Application Support", "Google", "Chrome")
	if _, err := os.Stat(sourceRoot); err != nil {
		return "", fmt.Errorf("chrome profile not found: %w", err)
	}

	destRoot, err := os.MkdirTemp("", "x-cli-chrome-")
	if err != nil {
		return "", err
	}

	for _, relative := range []string{"Local State", "First Run"} {
		src := filepath.Join(sourceRoot, relative)
		if _, err := os.Stat(src); err == nil {
			if err := copyFile(src, filepath.Join(destRoot, relative)); err != nil {
				return "", err
			}
		}
	}

	if err := copyDir(filepath.Join(sourceRoot, "Default"), filepath.Join(destRoot, "Default")); err != nil {
		return "", err
	}

	return destRoot, nil
}

func CloneProfileForDebug() (string, error) {
	return cloneChromeProfile()
}

func copyDir(src string, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0755)
		}

		if shouldSkipProfilePath(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}

		if d.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFileWithMode(path, target, info.Mode())
	})
}

func shouldSkipProfilePath(rel string) bool {
	for _, prefix := range []string{
		"Cache",
		"Code Cache",
		"Crashpad",
		"DawnGraphiteCache",
		"DawnWebGPUCache",
		"GPUCache",
		"GrShaderCache",
		"GraphiteDawnCache",
		"ShaderCache",
		"Service Worker/CacheStorage",
		"blob_storage",
	} {
		if rel == prefix || strings.HasPrefix(rel, prefix+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func copyFile(src string, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return copyFileWithMode(src, dst, info.Mode())
}

func copyFileWithMode(src string, dst string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func mapSearchFilter(product string) string {
	switch strings.ToLower(product) {
	case "latest":
		return "live"
	case "photos":
		return "image"
	case "videos":
		return "video"
	default:
		return ""
	}
}

func (c *Client) rawRequest(method string, urlString string, headers map[string]string, body []byte) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, urlString, reqBody)
	if err != nil {
		return nil, 0, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if c.txProvider != nil && c.txProvider.Ready() && req.Header.Get("x-client-transaction-id") == "" {
		if txID, err := c.txProvider.Generate(req.Context(), method, urlString); err == nil && txID != "" {
			req.Header.Set("x-client-transaction-id", txID)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil && c.standardHTTP != nil {
		var fallbackBody io.Reader
		if body != nil {
			fallbackBody = bytes.NewReader(body)
		}
		fallbackReq, cloneErr := http.NewRequest(method, urlString, fallbackBody)
		if cloneErr == nil {
			for key, value := range headers {
				fallbackReq.Header.Set(key, value)
			}
			if c.txProvider != nil && c.txProvider.Ready() && fallbackReq.Header.Get("x-client-transaction-id") == "" {
				if txID, txErr := c.txProvider.Generate(fallbackReq.Context(), method, urlString); txErr == nil && txID != "" {
					fallbackReq.Header.Set("x-client-transaction-id", txID)
				}
			}
			resp, err = c.standardHTTP.Do(fallbackReq)
		}
	}
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return responseBody, resp.StatusCode, fmt.Errorf("x api error %d: %s", resp.StatusCode, truncateForError(string(responseBody)))
	}
	return responseBody, resp.StatusCode, nil
}

func NormalizeScreenName(screenName string) string {
	return strings.TrimPrefix(strings.TrimSpace(screenName), "@")
}

func NormalizeTweetID(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimSuffix(trimmed, "/")
	parts := strings.Split(trimmed, "/")
	return parts[len(parts)-1]
}
