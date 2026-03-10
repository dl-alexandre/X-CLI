package profile

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/auth"
)

const (
	ExportVersion = "1.0"
	ExportFormat  = "x-cli-profile"
)

type ConflictResolution string

const (
	ConflictSkip      ConflictResolution = "skip"
	ConflictOverwrite ConflictResolution = "overwrite"
	ConflictRename    ConflictResolution = "rename"
)

type ExportedProfile struct {
	Version     string          `json:"version"`
	Format      string          `json:"format"`
	ExportedAt  string          `json:"exported_at"`
	ProfileName string          `json:"profile_name"`
	Token       *EncryptedToken `json:"token"`
	Metadata    ProfileMetadata `json:"metadata"`
}

type EncryptedToken struct {
	EncryptedData string `json:"encrypted_data"`
	Nonce         string `json:"nonce"`
	KeyHint       string `json:"key_hint"`
}

type ProfileMetadata struct {
	Name        string `json:"name"`
	CreatedAt   string `json:"created_at"`
	LastUsed    string `json:"last_used"`
	TokenStatus string `json:"token_status"`
	Source      string `json:"source"`
}

type BackupFile struct {
	Version    string            `json:"version"`
	Format     string            `json:"format"`
	ExportedAt string            `json:"exported_at"`
	Profiles   []ExportedProfile `json:"profiles"`
	Count      int               `json:"count"`
}

type ExportOptions struct {
	OutputPath string
}

type ImportOptions struct {
	InputPath          string
	ConflictResolution ConflictResolution
	NewName            string
}

type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

func ExportProfile(profileName string, opts ExportOptions) (*ExportedProfile, error) {
	if profileName == "" {
		profileName = auth.DefaultProfile
	}

	storage := auth.NewTokenStorageWithProfile(profileName)
	token, err := storage.Load()
	if err != nil {
		return nil, fmt.Errorf("load profile %q: %w", profileName, err)
	}

	if token == nil {
		return nil, fmt.Errorf("profile %q not found or not authenticated", profileName)
	}

	encryptedToken, err := encryptTokenForExport(token)
	if err != nil {
		return nil, fmt.Errorf("encrypt token: %w", err)
	}

	exported := &ExportedProfile{
		Version:     ExportVersion,
		Format:      ExportFormat,
		ExportedAt:  time.Now().UTC().Format(time.RFC3339),
		ProfileName: profileName,
		Token:       encryptedToken,
		Metadata: ProfileMetadata{
			Name:        profileName,
			CreatedAt:   getProfileCreatedTime(profileName),
			LastUsed:    time.Now().UTC().Format(time.RFC3339),
			TokenStatus: auth.GetTokenStatus(token),
			Source:      getExportSource(),
		},
	}

	if opts.OutputPath != "" {
		if err := writeExportFile(exported, opts.OutputPath); err != nil {
			return nil, fmt.Errorf("write export file: %w", err)
		}
	}

	return exported, nil
}

func ImportProfile(opts ImportOptions) (*auth.OAuthToken, error) {
	exported, err := readExportFile(opts.InputPath)
	if err != nil {
		return nil, fmt.Errorf("read export file: %w", err)
	}

	validation := ValidateExport(exported)
	if !validation.Valid {
		return nil, fmt.Errorf("invalid export file: %s", strings.Join(validation.Errors, ", "))
	}

	profileName := exported.ProfileName
	if opts.NewName != "" {
		profileName = opts.NewName
	}

	storage := auth.NewTokenStorageWithProfile(profileName)
	existingToken, err := storage.Load()
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("check existing profile: %w", err)
	}

	if existingToken != nil {
		switch opts.ConflictResolution {
		case ConflictSkip:
			return nil, fmt.Errorf("profile %q already exists (use --overwrite or --rename)", profileName)
		case ConflictOverwrite:
		case ConflictRename:
			profileName = generateUniqueProfileName(profileName)
			storage = auth.NewTokenStorageWithProfile(profileName)
		default:
			return nil, fmt.Errorf("profile %q already exists", profileName)
		}
	}

	token, err := decryptTokenFromExport(exported.Token)
	if err != nil {
		return nil, fmt.Errorf("decrypt token: %w", err)
	}

	if err := storage.Save(token); err != nil {
		return nil, fmt.Errorf("save imported profile: %w", err)
	}

	return token, nil
}

func BackupAllProfiles(outputPath string) (*BackupFile, error) {
	profiles := auth.ListProfiles()

	backup := &BackupFile{
		Version:    ExportVersion,
		Format:     ExportFormat + "-backup",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Profiles:   make([]ExportedProfile, 0),
	}

	for _, profileName := range profiles {
		exported, err := ExportProfile(profileName, ExportOptions{})
		if err != nil {
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not authenticated") {
				continue
			}
			return nil, fmt.Errorf("export profile %q: %w", profileName, err)
		}
		backup.Profiles = append(backup.Profiles, *exported)
	}

	backup.Count = len(backup.Profiles)

	if outputPath != "" {
		if err := writeBackupFile(backup, outputPath); err != nil {
			return nil, fmt.Errorf("write backup file: %w", err)
		}
	}

	return backup, nil
}

func RestoreBackup(inputPath string, resolution ConflictResolution) ([]string, error) {
	backup, err := readBackupFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("read backup file: %w", err)
	}

	imported := make([]string, 0, len(backup.Profiles))

	for _, exported := range backup.Profiles {
		validation := ValidateExport(&exported)
		if !validation.Valid {
			continue
		}

		profileName := exported.ProfileName
		storage := auth.NewTokenStorageWithProfile(profileName)
		existingToken, err := storage.Load()
		if err != nil && !os.IsNotExist(err) {
			continue
		}

		if existingToken != nil {
			switch resolution {
			case ConflictSkip:
				continue
			case ConflictRename:
				profileName = generateUniqueProfileName(profileName)
				storage = auth.NewTokenStorageWithProfile(profileName)
			case ConflictOverwrite:
			default:
				continue
			}
		}

		token, err := decryptTokenFromExport(exported.Token)
		if err != nil {
			continue
		}

		if err := storage.Save(token); err != nil {
			continue
		}

		imported = append(imported, profileName)
	}

	return imported, nil
}

func ValidateExport(exported *ExportedProfile) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	if exported.Version == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "missing version field")
	}

	if exported.Format != ExportFormat {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("invalid format: expected %q, got %q", ExportFormat, exported.Format))
	}

	if exported.ProfileName == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "missing profile name")
	}

	if exported.Token == nil {
		result.Valid = false
		result.Errors = append(result.Errors, "missing token data")
	} else {
		if exported.Token.EncryptedData == "" {
			result.Valid = false
			result.Errors = append(result.Errors, "missing encrypted token data")
		}
		if exported.Token.Nonce == "" {
			result.Valid = false
			result.Errors = append(result.Errors, "missing encryption nonce")
		}
	}

	if exported.ExportedAt == "" {
		result.Warnings = append(result.Warnings, "missing export timestamp")
	}

	return result
}

func ValidateBackup(backup *BackupFile) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	if backup.Version == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "missing version field")
	}

	if backup.Count != len(backup.Profiles) {
		result.Warnings = append(result.Warnings, "profile count mismatch")
	}

	for i, profile := range backup.Profiles {
		profileValidation := ValidateExport(&profile)
		if !profileValidation.Valid {
			result.Valid = false
			for _, err := range profileValidation.Errors {
				result.Errors = append(result.Errors, fmt.Sprintf("profile[%d]: %s", i, err))
			}
		}
	}

	return result
}

func encryptTokenForExport(token *auth.OAuthToken) (*EncryptedToken, error) {
	data, err := json.Marshal(token)
	if err != nil {
		return nil, fmt.Errorf("marshal token: %w", err)
	}

	key := generateExportKey()
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

	ciphertext := gcm.Seal(nil, nonce, data, nil)

	return &EncryptedToken{
		EncryptedData: encodeBase64(ciphertext),
		Nonce:         encodeBase64(nonce),
		KeyHint:       generateKeyHint(),
	}, nil
}

func decryptTokenFromExport(encrypted *EncryptedToken) (*auth.OAuthToken, error) {
	ciphertext, err := decodeBase64(encrypted.EncryptedData)
	if err != nil {
		return nil, fmt.Errorf("decode encrypted data: %w", err)
	}

	nonce, err := decodeBase64(encrypted.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decode nonce: %w", err)
	}

	key := generateExportKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(nonce) != gcm.NonceSize() {
		return nil, errors.New("invalid nonce size")
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt token: %w", err)
	}

	var token auth.OAuthToken
	if err := json.Unmarshal(plaintext, &token); err != nil {
		return nil, fmt.Errorf("unmarshal token: %w", err)
	}

	return &token, nil
}

func generateExportKey() []byte {
	hostname, _ := os.Hostname()
	home, _ := os.UserHomeDir()
	seed := hostname + home + "x-cli-export-key-2024"

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

func generateKeyHint() string {
	hostname, _ := os.Hostname()
	if len(hostname) > 8 {
		return hostname[:8]
	}
	return hostname
}

func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func getProfileCreatedTime(profileName string) string {
	path, err := getProfilePath(profileName)
	if err != nil {
		return ""
	}

	info, err := os.Stat(path)
	if err != nil {
		return ""
	}

	return info.ModTime().UTC().Format(time.RFC3339)
}

func getProfilePath(profileName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	filename := "tokens.json.enc"
	if profileName != auth.DefaultProfile && profileName != "" {
		filename = "tokens-" + profileName + ".json.enc"
	}

	return filepath.Join(home, ".config", "x-cli", filename), nil
}

func getExportSource() string {
	hostname, _ := os.Hostname()
	return hostname
}

func generateUniqueProfileName(base string) string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s-%d", base, timestamp)
}

func writeExportFile(exported *ExportedProfile, path string) error {
	data, err := json.MarshalIndent(exported, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func writeBackupFile(backup *BackupFile, path string) error {
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func readExportFile(path string) (*ExportedProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var exported ExportedProfile
	if err := json.Unmarshal(data, &exported); err != nil {
		return nil, err
	}

	return &exported, nil
}

func readBackupFile(path string) (*BackupFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var backup BackupFile
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, err
	}

	return &backup, nil
}
