package xapi

import (
	"testing"
)

func TestMimeTypeFromExtension(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".png", "image/png"},
		{".gif", "image/gif"},
		{".bmp", ""},
		{".txt", ""},
		{".pdf", ""},
		{".mp4", ""},
		{"", ""},
		{"jpg", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := mimeTypeFromExtension(tt.ext)
			if result != tt.expected {
				t.Errorf("mimeTypeFromExtension(%q) = %q, expected %q", tt.ext, result, tt.expected)
			}
		})
	}
}

func TestMaxMediaSize(t *testing.T) {
	if MaxMediaSize != 5*1024*1024 {
		t.Errorf("MaxMediaSize = %d, expected 5MB (5242880)", MaxMediaSize)
	}
}

func TestMediaUploadURL(t *testing.T) {
	expected := "https://upload.twitter.com/1.1/media/upload.json"
	if MediaUploadURL != expected {
		t.Errorf("MediaUploadURL = %q, expected %q", MediaUploadURL, expected)
	}
}
