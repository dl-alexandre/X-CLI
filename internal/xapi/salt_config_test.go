package xapi

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

func TestNativeTransactionProvider_StaticSaltConfig(t *testing.T) {
	provider := NewNativeTransactionProvider()

	// Initially should not have static salt
	if provider.HasStaticSalt() {
		t.Error("Expected no static salt initially")
	}

	// Set valid salt (58 hex chars = 29 bytes)
	validSalt := "105016980dc6fe71dae0c322ce5d4f09a0f3c205ea421e2161eb83ac4e"
	if len(validSalt) != 58 {
		t.Fatalf("Test salt wrong length: %d, expected 58", len(validSalt))
	}

	err := provider.SetStaticSalt(validSalt)
	if err != nil {
		t.Errorf("Failed to set static salt: %v", err)
	}

	// Now should have static salt
	if !provider.HasStaticSalt() {
		t.Error("Expected HasStaticSalt to return true after setting")
	}

	// Generate a txid and verify it uses the static salt
	ctx := t.Context()
	txid, err := provider.Generate(ctx, "POST", "https://x.com/i/api/graphql/abc/CreateTweet")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if txid == "" {
		t.Error("Expected non-empty txid")
	}

	t.Logf("Generated txid: %s", txid)

	// Verify the salt is in the generated txid
	decoded, err := decodeTxIDForTest(txid)
	if err != nil {
		t.Fatalf("Failed to decode txid: %v", err)
	}

	extractedSalt := fmt.Sprintf("%x", decoded[41:70])
	if extractedSalt != validSalt {
		t.Errorf("Salt mismatch: expected %s, got %s", validSalt, extractedSalt)
	}
}

func decodeTxIDForTest(txid string) ([]byte, error) {
	if rem := len(txid) % 4; rem != 0 {
		txid += strings.Repeat("=", 4-rem)
	}
	return base64.StdEncoding.DecodeString(txid)
}
