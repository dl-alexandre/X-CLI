package xapi

import (
	"testing"
)

func TestWhitening(t *testing.T) {
	provider := NewNativeTransactionProvider()

	// Create a test buffer (70 bytes)
	buffer := make([]byte, 70)
	for i := range buffer {
		buffer[i] = byte(i)
	}

	// Create a test salt (29 bytes)
	salt := make([]byte, 29)
	for i := range salt {
		salt[i] = byte(0x42 + i)
	}

	// Apply whitening
	whitened := provider.whiten(buffer, salt)

	// Verify version byte (index 0) is unchanged
	if whitened[0] != buffer[0] {
		t.Error("Version byte should not be whitened")
	}

	// Verify bytes 1-69 are XORed
	for i := 1; i < 70; i++ {
		expected := buffer[i] ^ salt[(i-1)%29]
		if whitened[i] != expected {
			t.Errorf("Byte %d: expected %x, got %x", i, expected, whitened[i])
		}
	}

	// Verify double whitening restores original (XOR is reversible)
	restored := provider.whiten(whitened, salt)
	for i := 0; i < 70; i++ {
		if restored[i] != buffer[i] {
			t.Errorf("Byte %d not restored: expected %x, got %x", i, buffer[i], restored[i])
		}
	}
}

func TestWhiteningWithStaticSalt(t *testing.T) {
	provider := NewNativeTransactionProvider()

	// Set a static salt
	validSalt := "105016980dc6fe71dae0c322ce5d4f09a0f3c205ea421e2161eb83ac4e"
	err := provider.SetStaticSalt(validSalt)
	if err != nil {
		t.Fatalf("Failed to set salt: %v", err)
	}

	// Generate with and without whitening
	ctx := t.Context()

	// Without whitening (default)
	txid1, err := provider.Generate(ctx, "POST", "https://x.com/i/api/graphql/abc/CreateTweet")
	if err != nil {
		t.Fatalf("Generate without whitening failed: %v", err)
	}

	// Enable whitening
	provider.EnableWhitening()
	txid2, err := provider.Generate(ctx, "POST", "https://x.com/i/api/graphql/abc/CreateTweet")
	if err != nil {
		t.Fatalf("Generate with whitening failed: %v", err)
	}

	// They should be different
	if txid1 == txid2 {
		t.Error("Whitened and unwhitened txids should be different")
	}

	t.Logf("Without whitening: %s", txid1)
	t.Logf("With whitening: %s", txid2)
}
