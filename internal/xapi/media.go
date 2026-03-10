package xapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/dl-alexandre/X-CLI/internal/model"
)

const (
	MaxMediaSize   = 5 * 1024 * 1024
	MediaUploadURL = "https://upload.twitter.com/1.1/media/upload.json"
)

type MediaUploadResult struct {
	MediaID          string `json:"media_id"`
	MediaIDString    string `json:"media_id_string"`
	Size             int    `json:"size"`
	ExpiresAfterSecs int    `json:"expires_after_secs"`
	Image            struct {
		ImageType string `json:"image_type"`
		Width     int    `json:"w"`
		Height    int    `json:"h"`
	} `json:"image"`
}

func (c *Client) UploadMedia(filePath string) (*MediaUploadResult, error) {
	return c.UploadMediaWithProgress(filePath, nil)
}

func (c *Client) UploadMediaWithProgress(filePath string, progressCB ProgressCallback) (*MediaUploadResult, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(filePath))

	if isVideoFile(filePath) {
		if info.Size() > MaxVideoSize {
			return nil, fmt.Errorf("video file too large: %d bytes (max %d)", info.Size(), MaxVideoSize)
		}
		return c.UploadVideo(filePath, progressCB)
	}

	if info.Size() > MaxMediaSize {
		return nil, fmt.Errorf("file too large: %d bytes (max %d for images)", info.Size(), MaxMediaSize)
	}

	mimeType := mimeTypeFromExtension(ext)
	if mimeType == "" {
		return nil, fmt.Errorf("unsupported file type: %s (allowed: .jpg, .jpeg, .png, .gif, .mp4, .mov)", ext)
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="media"; filename="%s"`, filepath.Base(filePath)))
	h.Set("Content-Type", mimeType)

	part, err := writer.CreatePart(h)
	if err != nil {
		return nil, fmt.Errorf("create multipart part: %w", err)
	}

	if _, err := part.Write(fileData); err != nil {
		return nil, fmt.Errorf("write file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, MediaUploadURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	if c.authSession != nil && c.authSession.AuthToken != "" {
		req.Header.Set("X-Auth-Token", c.authSession.AuthToken)
		if c.authSession.CT0 != "" {
			req.Header.Set("X-Csrf-Token", c.authSession.CT0)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, truncateForError(string(respBody)))
	}

	var result MediaUploadResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if result.MediaIDString == "" {
		return nil, errors.New("no media_id in response")
	}

	return &result, nil
}

func mimeTypeFromExtension(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	default:
		return ""
	}
}

func (c *Client) CreatePostWithMedia(text string, mediaID string) (model.ActionResult, error) {
	var profileURL string
	var tweetID string

	mutation, err := c.runBrowserMutation("CreateTweet", func(ctx context.Context) error {
		if err := chromedp.Navigate("https://x.com/compose/tweet").Do(ctx); err != nil {
			return err
		}
		if err := chromedp.WaitVisible(`[data-testid="tweetTextarea_0"]`, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.AttributeValue(`[data-testid="AppTabBar_Profile_Link"]`, "href", &profileURL, nil, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}

		if text != "" {
			if err := chromedp.SendKeys(`[data-testid="tweetTextarea_0"]`, text, chromedp.ByQuery).Do(ctx); err != nil {
				return err
			}
		}

		if err := chromedp.Sleep(2 * time.Second).Do(ctx); err != nil {
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

	tweetID = stringValue(deepGet(body, "data", "create_tweet", "tweet_results", "result", "rest_id"))
	if tweetID == "" {
		tweetID = stringValue(deepGet(body, "data", "create_tweet", "tweet_results", "result", "tweet", "rest_id"))
	}
	if tweetID == "" {
		tweetID = findTweetID(body)
	}

	if tweetID != "" {
		return model.ActionResult{Action: "post", Success: true, URL: tweetURL(tweetID), Message: "post with media created"}, nil
	}

	if profileURL != "" {
		if createdURL, confirmErr := c.confirmPostedTweet("https://x.com"+profileURL, firstLine(text)); confirmErr == nil && createdURL != "" {
			return model.ActionResult{Action: "post", Success: true, URL: createdURL, Message: "post with media created"}, nil
		}
	}

	return model.ActionResult{}, errors.New("post with media was not confirmed")
}
