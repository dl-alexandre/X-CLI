package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	KeyringService = "x-cli"
	KeyringUser    = "oauth-tokens"
	DefaultProfile = "default"
)

type TokenStorage struct {
	useKeyring bool
	mu         sync.Mutex
	profile    string
}

func NewTokenStorage() *TokenStorage {
	return NewTokenStorageWithProfile(DefaultProfile)
}

func NewTokenStorageWithProfile(profile string) *TokenStorage {
	if profile == "" {
		profile = DefaultProfile
	}

	key := profileKeyringKey(profile)
	_, err := keyring.Get(KeyringService, key)
	useKeyring := err == nil || err != keyring.ErrNotFound

	return &TokenStorage{
		useKeyring: useKeyring,
		profile:    profile,
	}
}

func profileKeyringKey(profile string) string {
	if profile == DefaultProfile {
		return KeyringUser
	}
	return KeyringUser + "-" + profile
}

func (s *TokenStorage) Save(token *OAuthToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}

	key := profileKeyringKey(s.profile)

	if s.useKeyring {
		if err := keyring.Set(KeyringService, key, string(data)); err != nil {
			s.useKeyring = false
			return s.saveToFile(token)
		}
		return nil
	}

	return s.saveToFile(token)
}

func (s *TokenStorage) Load() (*OAuthToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := profileKeyringKey(s.profile)

	if s.useKeyring {
		token, err := s.loadFromKeyring(key)
		if err != nil {
			if errors.Is(err, keyring.ErrNotFound) {
				return nil, nil
			}
			s.useKeyring = false
			return s.loadFromFile()
		}
		return token, nil
	}

	return s.loadFromFile()
}

func (s *TokenStorage) Delete() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := profileKeyringKey(s.profile)

	if s.useKeyring {
		if err := keyring.Delete(KeyringService, key); err != nil && !errors.Is(err, keyring.ErrNotFound) {
			s.useKeyring = false
		}
	}

	return s.deleteFile()
}

func (s *TokenStorage) loadFromKeyring(key string) (*OAuthToken, error) {
	data, err := keyring.Get(KeyringService, key)
	if err != nil {
		return nil, err
	}

	var token OAuthToken
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	return &token, nil
}

func (s *TokenStorage) saveToFile(token *OAuthToken) error {
	path, err := tokenFilePath(s.profile)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}

	encrypted, err := encrypt(data)
	if err != nil {
		return fmt.Errorf("encrypt token: %w", err)
	}

	if err := os.WriteFile(path, encrypted, 0600); err != nil {
		return fmt.Errorf("write token file: %w", err)
	}

	return nil
}

func (s *TokenStorage) loadFromFile() (*OAuthToken, error) {
	path, err := tokenFilePath(s.profile)
	if err != nil {
		return nil, err
	}

	encrypted, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read token file: %w", err)
	}

	data, err := decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt token: %w", err)
	}

	var token OAuthToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	return &token, nil
}

func (s *TokenStorage) deleteFile() error {
	path, err := tokenFilePath(s.profile)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete token file: %w", err)
	}

	return nil
}

func tokenFilePath(profile string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}

	filename := "tokens.json.enc"
	if profile != DefaultProfile && profile != "" {
		filename = "tokens-" + profile + ".json.enc"
	}

	return filepath.Join(home, ".config", "x-cli", filename), nil
}

func encrypt(plaintext []byte) ([]byte, error) {
	key := deriveKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func decrypt(ciphertext []byte) ([]byte, error) {
	key := deriveKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func deriveKey() []byte {
	hostname, _ := os.Hostname()
	home, _ := os.UserHomeDir()
	seed := hostname + home + "x-cli-oauth-key-2024"

	key := make([]byte, 32)
	for i := range key {
		if i < len(seed) {
			key[i] = seed[i]
		} else {
			key[i] = byte(i)
		}
	}

	return key
}

func (s *TokenStorage) IsKeyringAvailable() bool {
	return s.useKeyring
}

func (s *TokenStorage) GetProfile() string {
	return s.profile
}

func ListProfiles() []string {
	profiles := []string{DefaultProfile}

	home, err := os.UserHomeDir()
	if err != nil {
		return profiles
	}

	configDir := filepath.Join(home, ".config", "x-cli")
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return profiles
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "tokens-") && strings.HasSuffix(name, ".json.enc") {
			profile := strings.TrimPrefix(name, "tokens-")
			profile = strings.TrimSuffix(profile, ".json.enc")
			if profile != "" && profile != DefaultProfile {
				profiles = append(profiles, profile)
			}
		}
	}

	return profiles
}

func GetTokenStatus(token *OAuthToken) string {
	if token == nil {
		return "not authenticated"
	}

	if token.IsExpired() {
		return "expired"
	}

	if token.NeedsRefresh() {
		return "needs refresh"
	}

	remaining := time.Until(token.ExpiresAt)
	if remaining < 0 {
		return "expired"
	}

	return fmt.Sprintf("valid for %v", remaining.Round(time.Minute))
}
