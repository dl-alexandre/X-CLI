package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTokenStorage_ProfileKeyringKey(t *testing.T) {
	tests := []struct {
		profile  string
		expected string
	}{
		{DefaultProfile, KeyringUser},
		{"work", KeyringUser + "-work"},
		{"personal", KeyringUser + "-personal"},
	}

	for _, tt := range tests {
		t.Run(tt.profile, func(t *testing.T) {
			result := profileKeyringKey(tt.profile)
			if result != tt.expected {
				t.Errorf("profileKeyringKey(%q) = %q, expected %q", tt.profile, result, tt.expected)
			}
		})
	}
}

func TestTokenFilePath(t *testing.T) {
	tests := []struct {
		name     string
		profile  string
		contains string
	}{
		{"default profile", DefaultProfile, "tokens.json.enc"},
		{"work profile", "work", "tokens-work.json.enc"},
		{"personal profile", "personal", "tokens-personal.json.enc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tokenFilePath(tt.profile)
			if err != nil {
				t.Fatalf("tokenFilePath() error = %v", err)
			}

			if !contains(result, tt.contains) {
				t.Errorf("tokenFilePath() = %q, expected to contain %q", result, tt.contains)
			}

			if !contains(result, ".config") {
				t.Errorf("tokenFilePath() = %q, expected to contain .config", result)
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty data", []byte{}},
		{"short data", []byte("hello")},
		{"medium data", []byte("this is a longer test string for encryption")},
		{"json data", []byte(`{"access_token":"test123","refresh_token":"refresh456"}`)},
		{"binary data", []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := encrypt(tt.data)
			if err != nil {
				t.Fatalf("encrypt() error = %v", err)
			}

			if len(encrypted) <= len(tt.data) {
				t.Error("encrypt() should add overhead (nonce + tag)")
			}

			decrypted, err := decrypt(encrypted)
			if err != nil {
				t.Fatalf("decrypt() error = %v", err)
			}

			if string(decrypted) != string(tt.data) {
				t.Errorf("decrypt() = %q, expected %q", decrypted, tt.data)
			}
		})
	}
}

func TestEncryptProducesDifferentCiphertext(t *testing.T) {
	data := []byte("test data for encryption")

	encrypted1, err := encrypt(data)
	if err != nil {
		t.Fatalf("encrypt() error = %v", err)
	}

	encrypted2, err := encrypt(data)
	if err != nil {
		t.Fatalf("encrypt() error = %v", err)
	}

	if string(encrypted1) == string(encrypted2) {
		t.Error("encrypt() should produce different ciphertext due to random nonce")
	}

	decrypted1, err := decrypt(encrypted1)
	if err != nil {
		t.Fatalf("decrypt() error = %v", err)
	}

	decrypted2, err := decrypt(encrypted2)
	if err != nil {
		t.Fatalf("decrypt() error = %v", err)
	}

	if string(decrypted1) != string(decrypted2) {
		t.Error("Both decryptions should produce the same plaintext")
	}
}

func TestDecryptInvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty data", []byte{}},
		{"too short", []byte{0x00, 0x01, 0x02}},
		{"invalid nonce", []byte("this is not valid encrypted data")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decrypt(tt.data)
			if err == nil {
				t.Error("decrypt() should fail with invalid data")
			}
		})
	}
}

func TestDeriveKey(t *testing.T) {
	key1 := deriveKey()
	key2 := deriveKey()

	if len(key1) != 32 {
		t.Errorf("deriveKey() length = %d, expected 32", len(key1))
	}

	for i := range key1 {
		if key1[i] != key2[i] {
			t.Error("deriveKey() should produce consistent keys on same machine")
			break
		}
	}
}

func TestNewTokenStorageWithProfile(t *testing.T) {
	tests := []struct {
		profile string
	}{
		{DefaultProfile},
		{"work"},
		{"personal"},
		{""},
	}

	for _, tt := range tests {
		t.Run(tt.profile, func(t *testing.T) {
			storage := NewTokenStorageWithProfile(tt.profile)

			if storage == nil {
				t.Fatal("NewTokenStorageWithProfile() returned nil")
			}

			expectedProfile := tt.profile
			if expectedProfile == "" {
				expectedProfile = DefaultProfile
			}

			if storage.GetProfile() != expectedProfile {
				t.Errorf("GetProfile() = %q, expected %q", storage.GetProfile(), expectedProfile)
			}
		})
	}
}

func TestTokenStorage_SaveAndLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "x-cli-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	storage := NewTokenStorageWithProfile("test-profile")
	storage.useKeyring = false

	token := &OAuthToken{
		AccessToken:  "test_access_token",
		TokenType:    "Bearer",
		ExpiresIn:    7200,
		RefreshToken: "test_refresh_token",
		Scope:        "tweet.read tweet.write",
		ExpiresAt:    time.Now().Add(2 * time.Hour),
	}

	err = storage.Save(token)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := storage.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded == nil {
		t.Fatal("Load() returned nil")
	}

	if loaded.AccessToken != token.AccessToken {
		t.Errorf("AccessToken = %q, expected %q", loaded.AccessToken, token.AccessToken)
	}

	if loaded.RefreshToken != token.RefreshToken {
		t.Errorf("RefreshToken = %q, expected %q", loaded.RefreshToken, token.RefreshToken)
	}
}

func TestTokenStorage_Delete(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "x-cli-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	storage := NewTokenStorageWithProfile("test-delete-profile")
	storage.useKeyring = false

	token := &OAuthToken{
		AccessToken:  "test_access_token",
		RefreshToken: "test_refresh_token",
		ExpiresAt:    time.Now().Add(2 * time.Hour),
	}

	err = storage.Save(token)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	err = storage.Delete()
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	loaded, err := storage.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded != nil {
		t.Error("Load() should return nil after Delete()")
	}
}

func TestTokenStorage_LoadNonExistent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "x-cli-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	storage := NewTokenStorageWithProfile("nonexistent-profile")
	storage.useKeyring = false

	loaded, err := storage.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded != nil {
		t.Error("Load() should return nil for non-existent token")
	}
}

func TestListProfiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "x-cli-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	configDir := filepath.Join(tempDir, ".config", "x-cli")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	profiles := ListProfiles()

	if len(profiles) == 0 {
		t.Error("ListProfiles() should at least return default profile")
	}

	hasDefault := false
	for _, p := range profiles {
		if p == DefaultProfile {
			hasDefault = true
			break
		}
	}

	if !hasDefault {
		t.Error("ListProfiles() should include default profile")
	}
}
