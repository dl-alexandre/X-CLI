package xapi

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestCalculateE(t *testing.T) {
	e := calculateE()
	now := time.Now().Unix()
	expected := (now*1000 - txidEpoch*1000) / 1000

	diff := e - expected
	if diff < -1 || diff > 1 {
		t.Errorf("calculateE() = %d, expected around %d, diff = %d", e, expected, diff)
	}
}

func TestNativeTransactionProvider_Generate(t *testing.T) {
	provider := NewNativeTransactionProvider()

	if !provider.Ready() {
		t.Error("Expected provider to be ready")
	}

	if provider.Name() != "native-transaction-provider" {
		t.Errorf("Expected name 'native-transaction-provider', got '%s'", provider.Name())
	}

	ctx := context.Background()
	txid, err := provider.Generate(ctx, "POST", "https://x.com/i/api/graphql/abc123/CreateTweet")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if txid == "" {
		t.Error("Expected non-empty txid")
	}

	// Decode and verify length
	decoded, err := base64.StdEncoding.DecodeString(txid)
	if err != nil {
		// Try with padding
		padded := txid + strings.Repeat("=", (4-len(txid)%4)%4)
		decoded, err = base64.StdEncoding.DecodeString(padded)
		if err != nil {
			t.Fatalf("Failed to decode txid: %v", err)
		}
	}

	if len(decoded) != 70 {
		t.Errorf("Expected decoded length 70, got %d", len(decoded))
	}
}

func TestNativeTransactionProvider_GenerateDifferentOperations(t *testing.T) {
	provider := NewNativeTransactionProvider()
	ctx := context.Background()

	txid1, err := provider.Generate(ctx, "POST", "https://x.com/i/api/graphql/abc/CreateTweet")
	if err != nil {
		t.Fatalf("Generate for CreateTweet failed: %v", err)
	}

	txid2, err := provider.Generate(ctx, "POST", "https://x.com/i/api/graphql/def/FavoriteTweet")
	if err != nil {
		t.Fatalf("Generate for FavoriteTweet failed: %v", err)
	}

	// Different operations should produce different txids
	if txid1 == txid2 {
		t.Error("Different operations should produce different txids")
	}
}
