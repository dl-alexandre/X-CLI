package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dl-alexandre/X-CLI/internal/auth"
)

func TestExportProfile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "x-cli-export-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	storage := auth.NewTokenStorageWithProfile("test-export")
	storage.Save(&auth.OAuthToken{
		AccessToken:  "test_access_token",
		TokenType:    "Bearer",
		ExpiresIn:    7200,
		RefreshToken: "test_refresh_token",
		Scope:        "tweet.read tweet.write",
		ExpiresAt:    time.Now().Add(2 * time.Hour),
	})

	outputPath := filepath.Join(tempDir, "export.json")
	exported, err := ExportProfile("test-export", ExportOptions{OutputPath: outputPath})
	if err != nil {
		t.Fatalf("ExportProfile() error = %v", err)
	}

	if exported == nil {
		t.Fatal("ExportProfile() returned nil")
	}

	if exported.Version != ExportVersion {
		t.Errorf("Version = %q, expected %q", exported.Version, ExportVersion)
	}

	if exported.Format != ExportFormat {
		t.Errorf("Format = %q, expected %q", exported.Format, ExportFormat)
	}

	if exported.ProfileName != "test-export" {
		t.Errorf("ProfileName = %q, expected %q", exported.ProfileName, "test-export")
	}

	if exported.Token == nil {
		t.Fatal("Token should not be nil")
	}

	if exported.Token.EncryptedData == "" {
		t.Error("EncryptedData should not be empty")
	}

	if exported.Token.Nonce == "" {
		t.Error("Nonce should not be empty")
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Export file was not created")
	}
}

func TestExportProfile_NonExistent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "x-cli-export-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	_, err = ExportProfile("nonexistent", ExportOptions{})
	if err == nil {
		t.Error("ExportProfile() should fail for non-existent profile")
	}
}

func TestImportProfile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "x-cli-import-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	originalToken := &auth.OAuthToken{
		AccessToken:  "import_test_access_token",
		TokenType:    "Bearer",
		ExpiresIn:    7200,
		RefreshToken: "import_test_refresh_token",
		Scope:        "tweet.read tweet.write",
		ExpiresAt:    time.Now().Add(2 * time.Hour),
	}

	encryptedToken, err := encryptTokenForExport(originalToken)
	if err != nil {
		t.Fatalf("encryptTokenForExport() error = %v", err)
	}

	exported := &ExportedProfile{
		Version:     ExportVersion,
		Format:      ExportFormat,
		ExportedAt:  time.Now().UTC().Format(time.RFC3339),
		ProfileName: "import-test",
		Token:       encryptedToken,
		Metadata: ProfileMetadata{
			Name:   "import-test",
			Source: "test",
		},
	}

	exportPath := filepath.Join(tempDir, "import.json")
	data, _ := json.MarshalIndent(exported, "", "  ")
	os.WriteFile(exportPath, data, 0600)

	token, err := ImportProfile(ImportOptions{
		InputPath:          exportPath,
		ConflictResolution: ConflictSkip,
	})
	if err != nil {
		t.Fatalf("ImportProfile() error = %v", err)
	}

	if token == nil {
		t.Fatal("ImportProfile() returned nil token")
	}

	if token.AccessToken != originalToken.AccessToken {
		t.Errorf("AccessToken = %q, expected %q", token.AccessToken, originalToken.AccessToken)
	}

	if token.RefreshToken != originalToken.RefreshToken {
		t.Errorf("RefreshToken = %q, expected %q", token.RefreshToken, originalToken.RefreshToken)
	}
}

func TestImportProfile_ConflictResolution(t *testing.T) {
	tests := []struct {
		name       string
		resolution ConflictResolution
		wantError  bool
	}{
		{"skip on conflict", ConflictSkip, true},
		{"overwrite on conflict", ConflictOverwrite, false},
		{"rename on conflict", ConflictRename, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "x-cli-conflict-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			originalHome := os.Getenv("HOME")
			os.Setenv("HOME", tempDir)
			defer os.Setenv("HOME", originalHome)

			storage := auth.NewTokenStorageWithProfile("conflict-test")
			storage.Save(&auth.OAuthToken{
				AccessToken:  "existing_token",
				RefreshToken: "existing_refresh",
				ExpiresAt:    time.Now().Add(2 * time.Hour),
			})

			exported := &ExportedProfile{
				Version:     ExportVersion,
				Format:      ExportFormat,
				ExportedAt:  time.Now().UTC().Format(time.RFC3339),
				ProfileName: "conflict-test",
				Token: mustEncryptToken(t, &auth.OAuthToken{
					AccessToken:  "new_token",
					RefreshToken: "new_refresh",
					ExpiresAt:    time.Now().Add(2 * time.Hour),
				}),
				Metadata: ProfileMetadata{Name: "conflict-test"},
			}

			exportPath := filepath.Join(tempDir, "conflict.json")
			data, _ := json.MarshalIndent(exported, "", "  ")
			os.WriteFile(exportPath, data, 0600)

			_, err = ImportProfile(ImportOptions{
				InputPath:          exportPath,
				ConflictResolution: tt.resolution,
			})

			if tt.wantError && err == nil {
				t.Error("ImportProfile() should return error for skip resolution")
			}
			if !tt.wantError && err != nil {
				t.Errorf("ImportProfile() error = %v", err)
			}
		})
	}
}

func TestImportProfile_WithNewName(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "x-cli-rename-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	exported := &ExportedProfile{
		Version:     ExportVersion,
		Format:      ExportFormat,
		ExportedAt:  time.Now().UTC().Format(time.RFC3339),
		ProfileName: "original-name",
		Token:       mustEncryptToken(t, &auth.OAuthToken{AccessToken: "test", ExpiresAt: time.Now().Add(2 * time.Hour)}),
		Metadata:    ProfileMetadata{Name: "original-name"},
	}

	exportPath := filepath.Join(tempDir, "rename.json")
	data, _ := json.MarshalIndent(exported, "", "  ")
	os.WriteFile(exportPath, data, 0600)

	_, err = ImportProfile(ImportOptions{
		InputPath: exportPath,
		NewName:   "renamed-profile",
	})
	if err != nil {
		t.Fatalf("ImportProfile() error = %v", err)
	}

	storage := auth.NewTokenStorageWithProfile("renamed-profile")
	token, err := storage.Load()
	if err != nil {
		t.Fatalf("Load renamed profile: %v", err)
	}
	if token == nil {
		t.Error("Renamed profile should exist")
	}
}

func TestValidateExport(t *testing.T) {
	tests := []struct {
		name      string
		exported  *ExportedProfile
		wantValid bool
		wantErrs  int
	}{
		{
			name: "valid export",
			exported: &ExportedProfile{
				Version:     ExportVersion,
				Format:      ExportFormat,
				ProfileName: "test",
				Token:       &EncryptedToken{EncryptedData: "data", Nonce: "nonce"},
			},
			wantValid: true,
			wantErrs:  0,
		},
		{
			name: "missing version",
			exported: &ExportedProfile{
				Format:      ExportFormat,
				ProfileName: "test",
				Token:       &EncryptedToken{EncryptedData: "data", Nonce: "nonce"},
			},
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "invalid format",
			exported: &ExportedProfile{
				Version:     ExportVersion,
				Format:      "invalid",
				ProfileName: "test",
				Token:       &EncryptedToken{EncryptedData: "data", Nonce: "nonce"},
			},
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "missing profile name",
			exported: &ExportedProfile{
				Version: ExportVersion,
				Format:  ExportFormat,
				Token:   &EncryptedToken{EncryptedData: "data", Nonce: "nonce"},
			},
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "missing token",
			exported: &ExportedProfile{
				Version:     ExportVersion,
				Format:      ExportFormat,
				ProfileName: "test",
			},
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "missing encrypted data",
			exported: &ExportedProfile{
				Version:     ExportVersion,
				Format:      ExportFormat,
				ProfileName: "test",
				Token:       &EncryptedToken{Nonce: "nonce"},
			},
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "missing nonce",
			exported: &ExportedProfile{
				Version:     ExportVersion,
				Format:      ExportFormat,
				ProfileName: "test",
				Token:       &EncryptedToken{EncryptedData: "data"},
			},
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "missing export timestamp warning",
			exported: &ExportedProfile{
				Version:     ExportVersion,
				Format:      ExportFormat,
				ProfileName: "test",
				Token:       &EncryptedToken{EncryptedData: "data", Nonce: "nonce"},
			},
			wantValid: true,
			wantErrs:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateExport(tt.exported)

			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, expected %v", result.Valid, tt.wantValid)
			}

			if len(result.Errors) != tt.wantErrs {
				t.Errorf("Errors count = %d, expected %d", len(result.Errors), tt.wantErrs)
			}
		})
	}
}

func TestValidateBackup(t *testing.T) {
	tests := []struct {
		name      string
		backup    *BackupFile
		wantValid bool
	}{
		{
			name: "valid backup",
			backup: &BackupFile{
				Version: ExportVersion,
				Format:  ExportFormat + "-backup",
				Count:   1,
				Profiles: []ExportedProfile{
					{
						Version:     ExportVersion,
						Format:      ExportFormat,
						ProfileName: "test",
						Token:       &EncryptedToken{EncryptedData: "data", Nonce: "nonce"},
					},
				},
			},
			wantValid: true,
		},
		{
			name: "count mismatch warning",
			backup: &BackupFile{
				Version: ExportVersion,
				Format:  ExportFormat + "-backup",
				Count:   2,
				Profiles: []ExportedProfile{
					{
						Version:     ExportVersion,
						Format:      ExportFormat,
						ProfileName: "test",
						Token:       &EncryptedToken{EncryptedData: "data", Nonce: "nonce"},
					},
				},
			},
			wantValid: true,
		},
		{
			name: "missing version",
			backup: &BackupFile{
				Format: ExportFormat + "-backup",
				Count:  0,
			},
			wantValid: false,
		},
		{
			name: "invalid profile in backup",
			backup: &BackupFile{
				Version: ExportVersion,
				Format:  ExportFormat + "-backup",
				Count:   1,
				Profiles: []ExportedProfile{
					{
						Format:      ExportFormat,
						ProfileName: "test",
					},
				},
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateBackup(tt.backup)

			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, expected %v", result.Valid, tt.wantValid)
			}
		})
	}
}

func TestEncryptDecryptToken(t *testing.T) {
	token := &auth.OAuthToken{
		AccessToken:  "test_access_token_12345",
		TokenType:    "Bearer",
		ExpiresIn:    7200,
		RefreshToken: "test_refresh_token_67890",
		Scope:        "tweet.read tweet.write users.read offline.access",
		ExpiresAt:    time.Now().Add(2 * time.Hour),
	}

	encrypted, err := encryptTokenForExport(token)
	if err != nil {
		t.Fatalf("encryptTokenForExport() error = %v", err)
	}

	if encrypted.EncryptedData == "" {
		t.Error("EncryptedData should not be empty")
	}

	if encrypted.Nonce == "" {
		t.Error("Nonce should not be empty")
	}

	if encrypted.KeyHint == "" {
		t.Error("KeyHint should not be empty")
	}

	decrypted, err := decryptTokenFromExport(encrypted)
	if err != nil {
		t.Fatalf("decryptTokenFromExport() error = %v", err)
	}

	if decrypted.AccessToken != token.AccessToken {
		t.Errorf("AccessToken = %q, expected %q", decrypted.AccessToken, token.AccessToken)
	}

	if decrypted.RefreshToken != token.RefreshToken {
		t.Errorf("RefreshToken = %q, expected %q", decrypted.RefreshToken, token.RefreshToken)
	}

	if decrypted.Scope != token.Scope {
		t.Errorf("Scope = %q, expected %q", decrypted.Scope, token.Scope)
	}
}

func TestEncryptProducesDifferentCiphertext(t *testing.T) {
	token := &auth.OAuthToken{
		AccessToken:  "same_token",
		RefreshToken: "same_refresh",
		ExpiresAt:    time.Now().Add(2 * time.Hour),
	}

	encrypted1, err := encryptTokenForExport(token)
	if err != nil {
		t.Fatalf("encryptTokenForExport() error = %v", err)
	}

	encrypted2, err := encryptTokenForExport(token)
	if err != nil {
		t.Fatalf("encryptTokenForExport() error = %v", err)
	}

	if encrypted1.EncryptedData == encrypted2.EncryptedData {
		t.Error("Encryption should produce different ciphertext due to random nonce")
	}

	decrypted1, err := decryptTokenFromExport(encrypted1)
	if err != nil {
		t.Fatalf("decryptTokenFromExport() error = %v", err)
	}

	decrypted2, err := decryptTokenFromExport(encrypted2)
	if err != nil {
		t.Fatalf("decryptTokenFromExport() error = %v", err)
	}

	if decrypted1.AccessToken != decrypted2.AccessToken {
		t.Error("Both decryptions should produce the same token")
	}
}

func TestDecryptInvalidData(t *testing.T) {
	tests := []struct {
		name      string
		encrypted *EncryptedToken
		wantError bool
	}{
		{
			name:      "empty encrypted data",
			encrypted: &EncryptedToken{EncryptedData: "", Nonce: "nonce"},
			wantError: true,
		},
		{
			name:      "empty nonce",
			encrypted: &EncryptedToken{EncryptedData: "data", Nonce: ""},
			wantError: true,
		},
		{
			name:      "invalid encrypted data",
			encrypted: &EncryptedToken{EncryptedData: "invalid!!!", Nonce: "nonce"},
			wantError: true,
		},
		{
			name:      "invalid nonce size",
			encrypted: &EncryptedToken{EncryptedData: "data", Nonce: "short"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decryptTokenFromExport(tt.encrypted)
			if tt.wantError && err == nil {
				t.Error("decryptTokenFromExport() should return error")
			}
		})
	}
}

func TestBackupAllProfiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "x-cli-backup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	storage1 := auth.NewTokenStorageWithProfile("backup1")
	storage1.Save(&auth.OAuthToken{
		AccessToken:  "token1",
		RefreshToken: "refresh1",
		ExpiresAt:    time.Now().Add(2 * time.Hour),
	})

	storage2 := auth.NewTokenStorageWithProfile("backup2")
	storage2.Save(&auth.OAuthToken{
		AccessToken:  "token2",
		RefreshToken: "refresh2",
		ExpiresAt:    time.Now().Add(2 * time.Hour),
	})

	outputPath := filepath.Join(tempDir, "backup.json")
	backup, err := BackupAllProfiles(outputPath)
	if err != nil {
		t.Fatalf("BackupAllProfiles() error = %v", err)
	}

	if backup == nil {
		t.Fatal("BackupAllProfiles() returned nil")
	}

	if backup.Count < 2 {
		t.Errorf("Count = %d, expected at least 2", backup.Count)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Backup file was not created")
	}
}

func TestRestoreBackup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "x-cli-restore-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	backup := &BackupFile{
		Version:    ExportVersion,
		Format:     ExportFormat + "-backup",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Count:      2,
		Profiles: []ExportedProfile{
			{
				Version:     ExportVersion,
				Format:      ExportFormat,
				ExportedAt:  time.Now().UTC().Format(time.RFC3339),
				ProfileName: "restore1",
				Token:       mustEncryptToken(t, &auth.OAuthToken{AccessToken: "token1", ExpiresAt: time.Now().Add(2 * time.Hour)}),
				Metadata:    ProfileMetadata{Name: "restore1"},
			},
			{
				Version:     ExportVersion,
				Format:      ExportFormat,
				ExportedAt:  time.Now().UTC().Format(time.RFC3339),
				ProfileName: "restore2",
				Token:       mustEncryptToken(t, &auth.OAuthToken{AccessToken: "token2", ExpiresAt: time.Now().Add(2 * time.Hour)}),
				Metadata:    ProfileMetadata{Name: "restore2"},
			},
		},
	}

	backupPath := filepath.Join(tempDir, "restore.json")
	data, _ := json.MarshalIndent(backup, "", "  ")
	os.WriteFile(backupPath, data, 0600)

	imported, err := RestoreBackup(backupPath, ConflictSkip)
	if err != nil {
		t.Fatalf("RestoreBackup() error = %v", err)
	}

	if len(imported) != 2 {
		t.Errorf("Imported count = %d, expected 2", len(imported))
	}
}

func TestGenerateUniqueProfileName(t *testing.T) {
	name1 := generateUniqueProfileName("test")
	name2 := generateUniqueProfileName("test")

	if name1 == name2 {
		t.Error("generateUniqueProfileName() should produce unique names")
	}

	if name1 == "test" {
		t.Error("generateUniqueProfileName() should modify the base name")
	}
}

func TestGenerateExportKey(t *testing.T) {
	key1 := generateExportKey()
	key2 := generateExportKey()

	if len(key1) != 32 {
		t.Errorf("Key length = %d, expected 32", len(key1))
	}

	for i := range key1 {
		if key1[i] != key2[i] {
			t.Error("generateExportKey() should produce consistent keys on same machine")
			break
		}
	}
}

func TestGenerateKeyHint(t *testing.T) {
	hint := generateKeyHint()

	if hint == "" {
		t.Error("generateKeyHint() should not return empty string")
	}
}

func mustEncryptToken(t *testing.T, token *auth.OAuthToken) *EncryptedToken {
	t.Helper()
	encrypted, err := encryptTokenForExport(token)
	if err != nil {
		t.Fatalf("encryptTokenForExport() error = %v", err)
	}
	return encrypted
}
