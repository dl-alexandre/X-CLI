package xapi

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
)

// generateValidSalt creates a 58-character hex string (29 bytes * 2 = 58 chars)
func generateValidSalt() string {
	// 29 bytes worth of hex characters
	return "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5"
}

func TestNativeTransactionProvider_SetStaticSalt(t *testing.T) {
	provider := NewNativeTransactionProvider()

	// Test valid 58-char hex salt
	validSalt := generateValidSalt()
	err := provider.SetStaticSalt(validSalt)
	if err != nil {
		t.Errorf("SetStaticSalt with valid salt failed: %v", err)
	}

	// Generate a txid and verify salt is used
	ctx := context.Background()
	txid, err := provider.Generate(ctx, "POST", "https://x.com/i/api/graphql/abc/CreateTweet")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Decode and check salt bytes
	decoded, err := base64.StdEncoding.DecodeString(padBase64(txid))
	if err != nil {
		padded := txid + strings.Repeat("=", (4-len(txid)%4)%4)
		decoded, err = base64.StdEncoding.DecodeString(padded)
		if err != nil {
			t.Fatalf("Failed to decode txid: %v", err)
		}
	}

	if len(decoded) != 70 {
		t.Fatalf("Expected 70 bytes, got %d", len(decoded))
	}

	// Check salt bytes (41-70) match our input
	extractedSalt := decoded[41:70]
	expectedBytes, _ := decodeHexSalt(validSalt)

	if string(extractedSalt) != string(expectedBytes) {
		t.Errorf("Salt mismatch: expected %x, got %x", expectedBytes, extractedSalt)
	}
}

func TestNativeTransactionProvider_SetStaticSalt_Invalid(t *testing.T) {
	provider := NewNativeTransactionProvider()

	// Test invalid hex (too short)
	shortSalt := "a1b2c3"
	err := provider.SetStaticSalt(shortSalt)
	if err == nil {
		t.Error("Expected error for short salt, got nil")
	}

	// Test invalid hex (invalid characters)
	invalidSalt := "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
	err = provider.SetStaticSalt(invalidSalt)
	if err == nil {
		t.Error("Expected error for invalid hex chars, got nil")
	}
}

func TestDecodeHexSalt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantLen int
	}{
		{
			name:    "valid lowercase",
			input:   generateValidSalt(),
			wantErr: false,
			wantLen: 29,
		},
		{
			name:    "valid uppercase",
			input:   "A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5",
			wantErr: false,
			wantLen: 29,
		},
		{
			name:    "too short",
			input:   "a1b2c3",
			wantErr: true,
		},
		{
			name:    "too long",
			input:   "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			wantErr: true,
		},
		{
			name:    "invalid chars",
			input:   "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeHexSalt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeHexSalt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("decodeHexSalt() returned %d bytes, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestHexCharValue(t *testing.T) {
	tests := []struct {
		input byte
		want  int
	}{
		{'0', 0},
		{'9', 9},
		{'a', 10},
		{'f', 15},
		{'A', 10},
		{'F', 15},
		{'g', -1},
		{'z', -1},
		{' ', -1},
	}

	for _, tt := range tests {
		got := hexCharValue(tt.input)
		if got != tt.want {
			t.Errorf("hexCharValue(%c) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestDerivePlaceholderSalt(t *testing.T) {
	salt := derivePlaceholderSalt()
	if len(salt) != 29 {
		t.Errorf("Expected 29 bytes, got %d", len(salt))
	}

	// Check it's deterministic
	salt2 := derivePlaceholderSalt()
	if string(salt) != string(salt2) {
		t.Error("Placeholder salt should be deterministic")
	}
}
