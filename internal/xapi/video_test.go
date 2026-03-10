package xapi

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestVideoMimeTypeFromExtension(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".mp4", "video/mp4"},
		{".MP4", "video/mp4"},
		{".mov", "video/quicktime"},
		{".MOV", "video/quicktime"},
		{".avi", ""},
		{".mkv", ""},
		{".txt", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := videoMimeTypeFromExtension(tt.ext)
			if result != tt.expected {
				t.Errorf("videoMimeTypeFromExtension(%q) = %q, want %q", tt.ext, result, tt.expected)
			}
		})
	}
}

func TestIsVideoFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"video.mp4", true},
		{"video.MP4", true},
		{"video.mov", true},
		{"video.MOV", true},
		{"image.jpg", false},
		{"image.png", false},
		{"image.gif", false},
		{"document.txt", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isVideoFile(tt.path)
			if result != tt.expected {
				t.Errorf("isVideoFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0 * time.Second, "0s"},
		{5 * time.Second, "5s"},
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{3600 * time.Second, "1h0m"},
		{3661 * time.Second, "1h1m"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestFormatProgress(t *testing.T) {
	tests := []struct {
		name     string
		progress VideoUploadProgress
		contains []string
	}{
		{
			name: "zero progress",
			progress: VideoUploadProgress{
				TotalBytes:    100 * 1024 * 1024,
				UploadedBytes: 0,
				CurrentChunk:  0,
				TotalChunks:   25,
				Percent:       0,
			},
			contains: []string{"0.0%"},
		},
		{
			name: "half progress",
			progress: VideoUploadProgress{
				TotalBytes:    100 * 1024 * 1024,
				UploadedBytes: 50 * 1024 * 1024,
				CurrentChunk:  12,
				TotalChunks:   25,
				Percent:       50,
			},
			contains: []string{"50.0%"},
		},
		{
			name: "complete progress",
			progress: VideoUploadProgress{
				TotalBytes:    100 * 1024 * 1024,
				UploadedBytes: 100 * 1024 * 1024,
				CurrentChunk:  25,
				TotalChunks:   25,
				Percent:       100,
			},
			contains: []string{"100.0%"},
		},
		{
			name: "with speed",
			progress: VideoUploadProgress{
				TotalBytes:    100 * 1024 * 1024,
				UploadedBytes: 50 * 1024 * 1024,
				CurrentChunk:  12,
				TotalChunks:   25,
				Percent:       50,
				Speed:         5 * 1024 * 1024,
			},
			contains: []string{"50.0%", "5.0 MB/s"},
		},
		{
			name: "with eta",
			progress: VideoUploadProgress{
				TotalBytes:    100 * 1024 * 1024,
				UploadedBytes: 50 * 1024 * 1024,
				CurrentChunk:  12,
				TotalChunks:   25,
				Percent:       50,
				ETA:           30 * time.Second,
			},
			contains: []string{"50.0%", "ETA: 30s"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatProgress(tt.progress)
			for _, s := range tt.contains {
				if !bytes.Contains([]byte(result), []byte(s)) {
					t.Errorf("FormatProgress() = %q, should contain %q", result, s)
				}
			}
		})
	}
}

func TestVideoUploadStateFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "xcli-video-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stateFile := VideoUploadStateFile{
		MediaID:        123456789,
		MediaIDString:  "123456789",
		FilePath:       "/path/to/video.mp4",
		FileSize:       50 * 1024 * 1024,
		TotalChunks:    13,
		UploadedChunks: 5,
		State:          VideoStateUploading,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	data, err := json.MarshalIndent(stateFile, "", "  ")
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}

	var parsed VideoUploadStateFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}

	if parsed.MediaID != stateFile.MediaID {
		t.Errorf("MediaID = %d, want %d", parsed.MediaID, stateFile.MediaID)
	}
	if parsed.MediaIDString != stateFile.MediaIDString {
		t.Errorf("MediaIDString = %q, want %q", parsed.MediaIDString, stateFile.MediaIDString)
	}
	if parsed.State != stateFile.State {
		t.Errorf("State = %q, want %q", parsed.State, stateFile.State)
	}
	if parsed.UploadedChunks != stateFile.UploadedChunks {
		t.Errorf("UploadedChunks = %d, want %d", parsed.UploadedChunks, stateFile.UploadedChunks)
	}
}

func TestVideoUploadStateValues(t *testing.T) {
	states := []VideoUploadState{
		VideoStatePending,
		VideoStateInit,
		VideoStateUploading,
		VideoStateFinalize,
		VideoStateComplete,
		VideoStateFailed,
	}

	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			if state == "" {
				t.Error("state should not be empty")
			}
		})
	}
}

func TestVideoCategoryValues(t *testing.T) {
	categories := []VideoCategory{
		VideoCategoryTweet,
		VideoCategoryDM,
		VideoCategoryAmplify,
	}

	for _, cat := range categories {
		t.Run(string(cat), func(t *testing.T) {
			if cat == "" {
				t.Error("category should not be empty")
			}
		})
	}
}

func TestEncodeDecodeVideoChunk(t *testing.T) {
	original := []byte("test video chunk data with binary content\x00\x01\x02")

	encoded := EncodeVideoChunk(original)
	if encoded == "" {
		t.Error("encoded chunk should not be empty")
	}

	decoded, err := DecodeVideoChunk(encoded)
	if err != nil {
		t.Fatalf("decode chunk: %v", err)
	}

	if !bytes.Equal(decoded, original) {
		t.Error("decoded chunk does not match original")
	}
}

func TestVideoUploadProgressCalculations(t *testing.T) {
	progress := VideoUploadProgress{
		TotalBytes:    100 * 1024 * 1024,
		UploadedBytes: 25 * 1024 * 1024,
		CurrentChunk:  6,
		TotalChunks:   25,
		Percent:       25,
		Speed:         2 * 1024 * 1024,
		ETA:           37 * time.Second,
		Elapsed:       12 * time.Second,
		UploadState:   VideoStateUploading,
	}

	if progress.Percent < 0 || progress.Percent > 100 {
		t.Errorf("invalid percent: %f", progress.Percent)
	}

	if progress.CurrentChunk < 0 || progress.CurrentChunk > progress.TotalChunks {
		t.Errorf("invalid chunk: %d/%d", progress.CurrentChunk, progress.TotalChunks)
	}

	if progress.UploadedBytes > progress.TotalBytes {
		t.Errorf("uploaded bytes exceed total: %d > %d", progress.UploadedBytes, progress.TotalBytes)
	}
}

func TestMultipartWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := newMultipartWriter(buf)

	_ = writer.WriteField("command", "INIT")
	_ = writer.WriteField("media_id", "123456789")

	partWriter := writer.CreateFormFile("media", "video.mp4")
	_, _ = partWriter.Write([]byte("test video data"))

	_ = writer.Close()

	result := buf.String()

	if !bytes.Contains([]byte(result), []byte("--"+writer.boundary)) {
		t.Error("result should contain boundary")
	}

	if !bytes.Contains([]byte(result), []byte("command")) {
		t.Error("result should contain command field")
	}

	if !bytes.Contains([]byte(result), []byte("media_id")) {
		t.Error("result should contain media_id field")
	}

	if !bytes.Contains([]byte(result), []byte("test video data")) {
		t.Error("result should contain video data")
	}

	contentType := writer.FormDataContentType()
	if !bytes.Contains([]byte(contentType), []byte("multipart/form-data")) {
		t.Error("content type should contain multipart/form-data")
	}
}

func TestVideoUploadStateFileExpiry(t *testing.T) {
	now := time.Now()

	expiredState := VideoUploadStateFile{
		ExpiresAt: now.Add(-1 * time.Hour),
	}

	if !now.After(expiredState.ExpiresAt) {
		t.Error("state should be expired")
	}

	validState := VideoUploadStateFile{
		ExpiresAt: now.Add(24 * time.Hour),
	}

	if now.After(validState.ExpiresAt) {
		t.Error("state should not be expired")
	}
}

func TestChunkSizeCalculation(t *testing.T) {
	tests := []struct {
		fileSize int64
		expected int
	}{
		{VideoChunkSize, 1},
		{VideoChunkSize + 1, 2},
		{VideoChunkSize * 2, 2},
		{VideoChunkSize*2 + 1, 3},
		{1, 1},
		{MaxVideoSize, int((MaxVideoSize + VideoChunkSize - 1) / VideoChunkSize)},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			totalChunks := int((tt.fileSize + VideoChunkSize - 1) / VideoChunkSize)
			if totalChunks != tt.expected {
				t.Errorf("fileSize %d: totalChunks = %d, want %d", tt.fileSize, totalChunks, tt.expected)
			}
		})
	}
}

func TestCleanupExpiredVideoStates(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "xcli-video-cleanup-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	stateDir := filepath.Join(tmpDir, ".config", "x", "video-uploads")
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		t.Fatalf("create state dir: %v", err)
	}

	expiredState := VideoUploadStateFile{
		MediaID:        1,
		MediaIDString:  "1",
		FilePath:       "/old/video.mp4",
		FileSize:       1024,
		TotalChunks:    1,
		UploadedChunks: 1,
		State:          VideoStateComplete,
		ExpiresAt:      time.Now().Add(-24 * time.Hour),
	}

	validState := VideoUploadStateFile{
		MediaID:        2,
		MediaIDString:  "2",
		FilePath:       "/new/video.mp4",
		FileSize:       1024,
		TotalChunks:    1,
		UploadedChunks: 1,
		State:          VideoStateComplete,
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	expiredData, _ := json.Marshal(expiredState)
	validData, _ := json.Marshal(validState)

	if err := os.WriteFile(filepath.Join(stateDir, "expired.json"), expiredData, 0600); err != nil {
		t.Fatalf("write expired state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "valid.json"), validData, 0600); err != nil {
		t.Fatalf("write valid state: %v", err)
	}

	if err := CleanupExpiredVideoStates(); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	if _, err := os.Stat(filepath.Join(stateDir, "expired.json")); !os.IsNotExist(err) {
		t.Error("expired state file should be deleted")
	}

	if _, err := os.Stat(filepath.Join(stateDir, "valid.json")); os.IsNotExist(err) {
		t.Error("valid state file should not be deleted")
	}
}

func TestVideoProcessingInfo(t *testing.T) {
	processing := &VideoProcessingInfo{
		State:          "in_progress",
		CheckAfterSecs: 5,
		Progress:       50,
	}

	if processing.State != "in_progress" {
		t.Errorf("State = %q, want %q", processing.State, "in_progress")
	}

	if processing.Progress != 50 {
		t.Errorf("Progress = %d, want %d", processing.Progress, 50)
	}

	failedProcessing := &VideoProcessingInfo{
		State: "failed",
		Error: &struct {
			Code    int    `json:"code"`
			Name    string `json:"name"`
			Message string `json:"message"`
		}{
			Code:    1,
			Name:    "unsupported_format",
			Message: "Video format not supported",
		},
	}

	if failedProcessing.Error == nil {
		t.Error("failed processing should have error")
	}

	if failedProcessing.Error.Message != "Video format not supported" {
		t.Errorf("Error.Message = %q, want %q", failedProcessing.Error.Message, "Video format not supported")
	}
}

func TestVideoUploadInitResult(t *testing.T) {
	initResult := VideoUploadInitResult{
		MediaID:          123456789,
		MediaIDString:    "123456789",
		ExpiresAfterSecs: 86400,
	}

	if initResult.MediaIDString == "" {
		t.Error("MediaIDString should not be empty")
	}

	if initResult.ExpiresAfterSecs <= 0 {
		t.Error("ExpiresAfterSecs should be positive")
	}
}

func TestVideoUploadFinalizeResult(t *testing.T) {
	finalizeResult := VideoUploadFinalizeResult{
		MediaID:          123456789,
		MediaIDString:    "123456789",
		Size:             50 * 1024 * 1024,
		ExpiresAfterSecs: 86400,
		ProcessingInfo: &VideoProcessingInfo{
			State: "pending",
		},
	}

	if finalizeResult.MediaIDString == "" {
		t.Error("MediaIDString should not be empty")
	}

	if finalizeResult.Size <= 0 {
		t.Error("Size should be positive")
	}

	if finalizeResult.ProcessingInfo == nil {
		t.Error("ProcessingInfo should not be nil")
	}
}

func TestVideoMetadata(t *testing.T) {
	metadata := &VideoMetadata{
		Width:      1920,
		Height:     1080,
		DurationMs: 60000,
		Format:     "mp4",
		Codec:      "h264",
	}

	if metadata.Width != 1920 {
		t.Errorf("Width = %d, want %d", metadata.Width, 1920)
	}

	if metadata.Height != 1080 {
		t.Errorf("Height = %d, want %d", metadata.Height, 1080)
	}

	if metadata.DurationMs != 60000 {
		t.Errorf("DurationMs = %d, want %d", metadata.DurationMs, 60000)
	}
}

func TestMaxVideoSize(t *testing.T) {
	expectedMax := int64(512 * 1024 * 1024)
	if MaxVideoSize != expectedMax {
		t.Errorf("MaxVideoSize = %d, want %d", MaxVideoSize, expectedMax)
	}
}

func TestVideoChunkSize(t *testing.T) {
	expectedChunk := int64(4 * 1024 * 1024)
	if VideoChunkSize != expectedChunk {
		t.Errorf("VideoChunkSize = %d, want %d", VideoChunkSize, expectedChunk)
	}
}
