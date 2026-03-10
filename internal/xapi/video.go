package xapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/model"
)

const (
	MaxVideoSize      = 512 * 1024 * 1024
	VideoChunkSize    = 4 * 1024 * 1024
	VideoUploadURL    = "https://upload.twitter.com/1.1/media/upload.json"
	VideoStateFileDir = ".config/x/video-uploads"
)

type VideoCategory string

const (
	VideoCategoryTweet   VideoCategory = "tweet"
	VideoCategoryDM      VideoCategory = "dm"
	VideoCategoryAmplify VideoCategory = "amplify"
)

type VideoUploadState string

const (
	VideoStatePending   VideoUploadState = "pending"
	VideoStateInit      VideoUploadState = "init"
	VideoStateUploading VideoUploadState = "uploading"
	VideoStateFinalize  VideoUploadState = "finalize"
	VideoStateComplete  VideoUploadState = "complete"
	VideoStateFailed    VideoUploadState = "failed"
)

type VideoInfo struct {
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	DurationMs int    `json:"duration_millis"`
	Format     string `json:"format"`
	Codec      string `json:"codec"`
}

type VideoUploadInitResult struct {
	MediaID          int64  `json:"media_id"`
	MediaIDString    string `json:"media_id_string"`
	ExpiresAfterSecs int    `json:"expires_after_secs"`
	Video            struct {
		VideoType string `json:"video_type"`
	} `json:"video"`
}

type VideoUploadFinalizeResult struct {
	MediaID          int64  `json:"media_id"`
	MediaIDString    string `json:"media_id_string"`
	Size             int    `json:"size"`
	ExpiresAfterSecs int    `json:"expires_after_secs"`
	Video            struct {
		VideoType string `json:"video_type"`
	} `json:"video"`
	ProcessingInfo *VideoProcessingInfo `json:"processing_info,omitempty"`
}

type VideoProcessingInfo struct {
	State          string `json:"state"`
	CheckAfterSecs int    `json:"check_after_secs,omitempty"`
	Progress       int    `json:"progress_percent,omitempty"`
	Error          *struct {
		Code    int    `json:"code"`
		Name    string `json:"name"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type VideoUploadProgress struct {
	TotalBytes     int64
	UploadedBytes  int64
	CurrentChunk   int
	TotalChunks    int
	Percent        float64
	Speed          float64
	ETA            time.Duration
	Elapsed        time.Duration
	UploadState    VideoUploadState
	ProcessingInfo *VideoProcessingInfo
}

type VideoUploadStateFile struct {
	MediaID        int64            `json:"media_id"`
	MediaIDString  string           `json:"media_id_string"`
	FilePath       string           `json:"file_path"`
	FileSize       int64            `json:"file_size"`
	TotalChunks    int              `json:"total_chunks"`
	UploadedChunks int              `json:"uploaded_chunks"`
	State          VideoUploadState `json:"state"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	ExpiresAt      time.Time        `json:"expires_at"`
}

type ProgressCallback func(progress VideoUploadProgress)

type VideoUploader struct {
	client        *Client
	progressCB    ProgressCallback
	stateFile     *VideoUploadStateFile
	stateFilePath string
	stateFileMu   sync.Mutex
	startTime     time.Time
	lastUpdate    time.Time
	lastBytes     int64
}

func videoMimeTypeFromExtension(ext string) string {
	switch strings.ToLower(ext) {
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	default:
		return ""
	}
}

func isVideoFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return videoMimeTypeFromExtension(ext) != ""
}

func (c *Client) UploadVideo(filePath string, progressCB ProgressCallback) (*MediaUploadResult, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	if info.Size() > MaxVideoSize {
		return nil, fmt.Errorf("video file too large: %d bytes (max %d)", info.Size(), MaxVideoSize)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	mimeType := videoMimeTypeFromExtension(ext)
	if mimeType == "" {
		return nil, fmt.Errorf("unsupported video format: %s (allowed: .mp4, .mov)", ext)
	}

	uploader := &VideoUploader{
		client:     c,
		progressCB: progressCB,
		startTime:  time.Now(),
	}

	stateFile, err := uploader.loadOrCreateState(filePath, info.Size())
	if err != nil {
		return nil, fmt.Errorf("state initialization: %w", err)
	}
	uploader.stateFile = stateFile

	if stateFile.State == VideoStateComplete && stateFile.MediaIDString != "" {
		return &MediaUploadResult{
			MediaID:          stateFile.MediaIDString,
			MediaIDString:    stateFile.MediaIDString,
			ExpiresAfterSecs: int(time.Until(stateFile.ExpiresAt).Seconds()),
		}, nil
	}

	_, err = uploader.initUpload(filePath, info.Size(), mimeType)
	if err != nil {
		return nil, fmt.Errorf("init upload: %w", err)
	}

	if err := uploader.uploadChunks(filePath, info.Size()); err != nil {
		return nil, fmt.Errorf("upload chunks: %w", err)
	}

	finalizeResult, err := uploader.finalizeUpload()
	if err != nil {
		return nil, fmt.Errorf("finalize upload: %w", err)
	}

	if err := uploader.waitForProcessing(finalizeResult); err != nil {
		return nil, fmt.Errorf("video processing: %w", err)
	}

	uploader.stateFile.State = VideoStateComplete
	uploader.stateFile.UpdatedAt = time.Now()
	_ = uploader.saveState()

	return &MediaUploadResult{
		MediaID:          finalizeResult.MediaIDString,
		MediaIDString:    finalizeResult.MediaIDString,
		Size:             finalizeResult.Size,
		ExpiresAfterSecs: finalizeResult.ExpiresAfterSecs,
	}, nil
}

func (u *VideoUploader) loadOrCreateState(filePath string, fileSize int64) (*VideoUploadStateFile, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	stateFileName := fmt.Sprintf("%x.json", absPath)
	stateDir := filepath.Join(os.Getenv("HOME"), VideoStateFileDir)
	stateFilePath := filepath.Join(stateDir, stateFileName)

	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}

	u.stateFilePath = stateFilePath

	data, err := os.ReadFile(stateFilePath)
	if err == nil {
		var state VideoUploadStateFile
		if err := json.Unmarshal(data, &state); err == nil {
			if state.FilePath == absPath && state.FileSize == fileSize {
				if state.ExpiresAt.IsZero() || time.Now().Before(state.ExpiresAt) {
					return &state, nil
				}
			}
		}
	}

	totalChunks := int((fileSize + VideoChunkSize - 1) / VideoChunkSize)
	return &VideoUploadStateFile{
		FilePath:       absPath,
		FileSize:       fileSize,
		TotalChunks:    totalChunks,
		UploadedChunks: 0,
		State:          VideoStatePending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

func (u *VideoUploader) saveState() error {
	u.stateFileMu.Lock()
	defer u.stateFileMu.Unlock()

	u.stateFile.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(u.stateFile, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(u.stateFilePath, data, 0600)
}

func (u *VideoUploader) initUpload(filePath string, fileSize int64, mimeType string) (*VideoUploadInitResult, error) {
	if u.stateFile.State == VideoStateInit || u.stateFile.State == VideoStateUploading {
		if u.stateFile.MediaIDString != "" {
			return &VideoUploadInitResult{
				MediaID:          u.stateFile.MediaID,
				MediaIDString:    u.stateFile.MediaIDString,
				ExpiresAfterSecs: int(time.Until(u.stateFile.ExpiresAt).Seconds()),
			}, nil
		}
	}

	form := urlValues{
		"command":        "INIT",
		"media_type":     mimeType,
		"total_bytes":    fmt.Sprintf("%d", fileSize),
		"media_category": string(VideoCategoryTweet),
	}

	resp, err := u.client.doVideoUploadRequest(form, nil)
	if err != nil {
		return nil, err
	}

	var result VideoUploadInitResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse init response: %w", err)
	}

	if result.MediaIDString == "" {
		return nil, errors.New("no media_id in init response")
	}

	u.stateFile.MediaID = result.MediaID
	u.stateFile.MediaIDString = result.MediaIDString
	u.stateFile.State = VideoStateInit
	u.stateFile.ExpiresAt = time.Now().Add(time.Duration(result.ExpiresAfterSecs) * time.Second)
	_ = u.saveState()

	return &result, nil
}

func (u *VideoUploader) uploadChunks(filePath string, fileSize int64) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	startChunk := u.stateFile.UploadedChunks
	if startChunk > 0 {
		offset := int64(startChunk) * VideoChunkSize
		if _, err := file.Seek(offset, 0); err != nil {
			return fmt.Errorf("seek to resume position: %w", err)
		}
	}

	totalChunks := u.stateFile.TotalChunks
	buffer := make([]byte, VideoChunkSize)

	for chunkIndex := startChunk; chunkIndex < totalChunks; chunkIndex++ {
		n, err := io.ReadFull(file, buffer)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return fmt.Errorf("read chunk %d: %w", chunkIndex, err)
		}

		chunkData := buffer[:n]
		segmentIndex := chunkIndex

		form := urlValues{
			"command":       "APPEND",
			"media_id":      u.stateFile.MediaIDString,
			"segment_index": fmt.Sprintf("%d", segmentIndex),
		}

		if err := u.uploadChunk(form, chunkData, chunkIndex, totalChunks, fileSize); err != nil {
			return fmt.Errorf("upload chunk %d: %w", chunkIndex, err)
		}

		u.stateFile.UploadedChunks = chunkIndex + 1
		u.stateFile.State = VideoStateUploading
		_ = u.saveState()
	}

	return nil
}

func (u *VideoUploader) uploadChunk(form urlValues, data []byte, chunkIndex, totalChunks int, totalSize int64) error {
	resp, err := u.client.doVideoUploadRequest(form, data)
	if err != nil {
		return err
	}

	var result struct {
		MediaID int64 `json:"media_id"`
	}
	_ = json.Unmarshal(resp, &result)

	now := time.Now()
	elapsed := now.Sub(u.startTime)
	uploadedBytes := int64(chunkIndex+1) * VideoChunkSize
	if uploadedBytes > totalSize {
		uploadedBytes = totalSize
	}

	var speed float64
	var eta time.Duration

	if !u.lastUpdate.IsZero() && now.Sub(u.lastUpdate) > 0 {
		bytesDiff := uploadedBytes - u.lastBytes
		timeDiff := now.Sub(u.lastUpdate).Seconds()
		if timeDiff > 0 {
			speed = float64(bytesDiff) / timeDiff
		}
	}

	if speed > 0 {
		remainingBytes := totalSize - uploadedBytes
		eta = time.Duration(float64(remainingBytes)/speed) * time.Second
	}

	u.lastUpdate = now
	u.lastBytes = uploadedBytes

	if u.progressCB != nil {
		u.progressCB(VideoUploadProgress{
			TotalBytes:    totalSize,
			UploadedBytes: uploadedBytes,
			CurrentChunk:  chunkIndex + 1,
			TotalChunks:   totalChunks,
			Percent:       float64(uploadedBytes) / float64(totalSize) * 100,
			Speed:         speed,
			ETA:           eta,
			Elapsed:       elapsed,
			UploadState:   VideoStateUploading,
		})
	}

	return nil
}

func (u *VideoUploader) finalizeUpload() (*VideoUploadFinalizeResult, error) {
	form := urlValues{
		"command":  "FINALIZE",
		"media_id": u.stateFile.MediaIDString,
	}

	resp, err := u.client.doVideoUploadRequest(form, nil)
	if err != nil {
		return nil, err
	}

	var result VideoUploadFinalizeResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse finalize response: %w", err)
	}

	u.stateFile.State = VideoStateFinalize
	_ = u.saveState()

	return &result, nil
}

func (u *VideoUploader) waitForProcessing(finalizeResult *VideoUploadFinalizeResult) error {
	if finalizeResult.ProcessingInfo == nil {
		return nil
	}

	processing := finalizeResult.ProcessingInfo

	for processing.State == "pending" || processing.State == "in_progress" {
		checkAfter := processing.CheckAfterSecs
		if checkAfter == 0 {
			checkAfter = 1
		}

		time.Sleep(time.Duration(checkAfter) * time.Second)

		status, err := u.getUploadStatus()
		if err != nil {
			return fmt.Errorf("get upload status: %w", err)
		}

		processing = status

		if u.progressCB != nil {
			u.progressCB(VideoUploadProgress{
				TotalBytes:     u.stateFile.FileSize,
				UploadedBytes:  u.stateFile.FileSize,
				CurrentChunk:   u.stateFile.TotalChunks,
				TotalChunks:    u.stateFile.TotalChunks,
				Percent:        100,
				Elapsed:        time.Since(u.startTime),
				UploadState:    VideoStateFinalize,
				ProcessingInfo: processing,
			})
		}

		if processing.State == "failed" {
			msg := "video processing failed"
			if processing.Error != nil {
				msg = processing.Error.Message
			}
			return errors.New(msg)
		}
	}

	return nil
}

func (u *VideoUploader) getUploadStatus() (*VideoProcessingInfo, error) {
	form := urlValues{
		"command":  "STATUS",
		"media_id": u.stateFile.MediaIDString,
	}

	resp, err := u.client.doVideoUploadRequest(form, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		ProcessingInfo *VideoProcessingInfo `json:"processing_info"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse status response: %w", err)
	}

	return result.ProcessingInfo, nil
}

type urlValues map[string]string

func (c *Client) doVideoUploadRequest(form urlValues, mediaData []byte) ([]byte, error) {
	var body io.Reader
	var contentType string

	if mediaData != nil {
		bodyBuf := &bytes.Buffer{}
		writer := newMultipartWriter(bodyBuf)

		_ = writer.WriteField("command", form["command"])
		_ = writer.WriteField("media_id", form["media_id"])
		_ = writer.WriteField("segment_index", form["segment_index"])

		partWriter := writer.CreateFormFile("media", "video.mp4")
		_, _ = partWriter.Write(mediaData)

		_ = writer.Close()
		body = bodyBuf
		contentType = writer.FormDataContentType()
	} else {
		formData := &bytes.Buffer{}
		for key, value := range form {
			if formData.Len() > 0 {
				formData.WriteByte('&')
			}
			formData.WriteString(key + "=" + value)
		}
		body = formData
		contentType = "application/x-www-form-urlencoded"
	}

	req, err := http.NewRequest(http.MethodPost, VideoUploadURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
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

	return respBody, nil
}

type multipartWriter struct {
	*bytes.Buffer
	boundary string
}

func newMultipartWriter(buf *bytes.Buffer) *multipartWriter {
	boundary := fmt.Sprintf("----xcli%016x", time.Now().UnixNano())
	return &multipartWriter{
		Buffer:   buf,
		boundary: boundary,
	}
}

func (w *multipartWriter) FormDataContentType() string {
	return "multipart/form-data; boundary=" + w.boundary
}

func (w *multipartWriter) WriteField(name, value string) error {
	if value == "" {
		return nil
	}
	fmt.Fprintf(w, "--%s\r\n", w.boundary)
	fmt.Fprintf(w, "Content-Disposition: form-data; name=\"%s\"\r\n\r\n", name)
	fmt.Fprintf(w, "%s\r\n", value)
	return nil
}

func (w *multipartWriter) CreateFormFile(name, filename string) io.Writer {
	fmt.Fprintf(w, "--%s\r\n", w.boundary)
	fmt.Fprintf(w, "Content-Disposition: form-data; name=\"%s\"; filename=\"%s\"\r\n", name, filename)
	fmt.Fprintf(w, "Content-Type: application/octet-stream\r\n\r\n")
	return w
}

func (w *multipartWriter) Close() error {
	fmt.Fprintf(w, "--%s--\r\n", w.boundary)
	return nil
}

func FormatProgress(p VideoUploadProgress) string {
	percent := p.Percent
	if percent > 100 {
		percent = 100
	}

	barWidth := 30
	filled := int(percent / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	speedStr := ""
	if p.Speed > 0 {
		speedStr = fmt.Sprintf(" %.1f MB/s", p.Speed/1024/1024)
	}

	etaStr := ""
	if p.ETA > 0 {
		etaStr = fmt.Sprintf(" ETA: %s", formatDuration(p.ETA))
	}

	return fmt.Sprintf("[%s] %.1f%%%s%s", bar, percent, speedStr, etaStr)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

func FormatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n/div >= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func CleanupExpiredVideoStates() error {
	stateDir := filepath.Join(os.Getenv("HOME"), VideoStateFileDir)
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(stateDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var state VideoUploadStateFile
		if err := json.Unmarshal(data, &state); err != nil {
			continue
		}

		if !state.ExpiresAt.IsZero() && time.Now().After(state.ExpiresAt) {
			os.Remove(path)
		}
	}

	return nil
}

func GetVideoUploadState(filePath string) (*VideoUploadStateFile, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	stateFileName := fmt.Sprintf("%x.json", absPath)
	stateDir := filepath.Join(os.Getenv("HOME"), VideoStateFileDir)
	stateFilePath := filepath.Join(stateDir, stateFileName)

	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		return nil, err
	}

	var state VideoUploadStateFile
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func DeleteVideoUploadState(filePath string) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	stateFileName := fmt.Sprintf("%x.json", absPath)
	stateDir := filepath.Join(os.Getenv("HOME"), VideoStateFileDir)
	stateFilePath := filepath.Join(stateDir, stateFileName)

	return os.Remove(stateFilePath)
}

func (c *Client) CreatePostWithVideo(text string, mediaID string) (model.ActionResult, error) {
	return c.CreatePostWithMedia(text, mediaID)
}

type VideoMetadata struct {
	Width      int
	Height     int
	DurationMs int
	Format     string
	Codec      string
}

func ParseVideoMetadata(filePath string) (*VideoMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buf := make([]byte, 1024)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}

	data := buf[:n]

	if len(data) < 12 {
		return nil, errors.New("file too small to be a valid video")
	}

	if !bytes.Equal(data[4:8], []byte("ftyp")) && !bytes.Equal(data[4:8], []byte("moov")) && !bytes.Equal(data[4:8], []byte("mdat")) {
		if !bytes.Equal(data[:4], []byte{0x00, 0x00, 0x00, 0x00}) {
			return nil, errors.New("not a valid MP4/MOV file")
		}
	}

	return &VideoMetadata{
		Format: "mp4",
	}, nil
}

func EncodeVideoChunk(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func DecodeVideoChunk(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}
