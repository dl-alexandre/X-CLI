package text

import (
	"strings"
	"testing"
)

func TestSplitIntoChunks_ShortText(t *testing.T) {
	text := "This is a short tweet"
	chunks := SplitIntoChunks(text)

	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}

	if chunks[0] != text {
		t.Errorf("Expected %q, got %q", text, chunks[0])
	}
}

func TestSplitIntoChunks_LongText(t *testing.T) {
	words := make([]string, 100)
	for i := range words {
		words[i] = "word"
	}
	text := strings.Join(words, " ")

	chunks := SplitIntoChunks(text)

	if len(chunks) < 2 {
		t.Errorf("Expected multiple chunks for long text, got %d", len(chunks))
	}

	for i, chunk := range chunks {
		if len(chunk) > MaxChunkLength {
			t.Errorf("Chunk %d exceeds max length: %d > %d", i, len(chunk), MaxChunkLength)
		}
	}
}

func TestSplitIntoChunks_RespectsWordBoundaries(t *testing.T) {
	text := strings.Repeat("testing ", 50)
	chunks := SplitIntoChunks(text)

	for i, chunk := range chunks {
		if len(chunk) > MaxChunkLength {
			t.Errorf("Chunk %d exceeds max length: %d > %d", i, len(chunk), MaxChunkLength)
		}

		if !strings.HasSuffix(chunk, "testing") && !strings.HasSuffix(chunk, "testin") {
			t.Errorf("Chunk %d doesn't end at word boundary: %q", i, chunk[len(chunk)-20:])
		}
	}
}

func TestPrepareThread_SingleTweet(t *testing.T) {
	text := "Single tweet"
	chunks := PrepareThread(text)

	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}

	if chunks[0].Total != 1 {
		t.Errorf("Expected total 1, got %d", chunks[0].Total)
	}

	if chunks[0].Number != 1 {
		t.Errorf("Expected number 1, got %d", chunks[0].Number)
	}
}

func TestPrepareThread_MultipleTweets(t *testing.T) {
	words := make([]string, 100)
	for i := range words {
		words[i] = "testing"
	}
	text := strings.Join(words, " ")

	chunks := PrepareThread(text)

	if len(chunks) < 2 {
		t.Errorf("Expected multiple chunks, got %d", len(chunks))
	}

	for i, chunk := range chunks {
		if chunk.Number != i+1 {
			t.Errorf("Expected number %d, got %d", i+1, chunk.Number)
		}

		if chunk.Total != len(chunks) {
			t.Errorf("Expected total %d, got %d", len(chunks), chunk.Total)
		}

		if len(chunks) > 1 {
			expectedSuffix := " (" + itoa(i+1) + "/" + itoa(len(chunks)) + ")"
			if !strings.HasSuffix(chunk.Text, expectedSuffix) {
				t.Errorf("Chunk %d missing thread suffix: %q", i, chunk.Text[len(chunk.Text)-20:])
			}
		}
	}
}

func TestSplitIntoChunks_EmptyText(t *testing.T) {
	chunks := SplitIntoChunks("")
	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty text, got %d", len(chunks))
	}
}

func TestSplitIntoChunks_WhitespaceOnly(t *testing.T) {
	chunks := SplitIntoChunks("   \t\n  ")
	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for whitespace-only text, got %d", len(chunks))
	}
}
