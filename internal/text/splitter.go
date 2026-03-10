package text

import (
	"context"
	"strings"
	"unicode/utf8"
)

const (
	MaxTweetLength    = 280
	ThreadSuffixSpace = 8
	MaxChunkLength    = MaxTweetLength - ThreadSuffixSpace
)

type ThreadChunk struct {
	Text    string
	Number  int
	Total   int
	ReplyTo string
}

type SplitOptions struct {
	ShortenURLs  bool
	URLShortener *URLShortener
	Context      context.Context
}

func SplitIntoChunks(text string) []string {
	return SplitIntoChunksWithOptions(text, SplitOptions{})
}

func SplitIntoChunksWithOptions(text string, opts SplitOptions) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	if opts.ShortenURLs && opts.URLShortener != nil && opts.Context != nil {
		var err error
		var urls []URLInfo
		text, urls, err = opts.URLShortener.ShortenAllURLs(opts.Context, text)
		if err == nil && len(urls) > 0 {
			_ = urls
		}
	}

	if utf8.RuneCountInString(text) <= MaxTweetLength {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var chunks []string
	currentChunk := ""
	currentLength := 0

	for _, word := range words {
		wordLength := utf8.RuneCountInString(word)
		spaceLength := 0
		if currentLength > 0 {
			spaceLength = 1
		}

		if currentLength+spaceLength+wordLength <= MaxChunkLength {
			if currentLength > 0 {
				currentChunk += " "
				currentLength++
			}
			currentChunk += word
			currentLength += wordLength
		} else {
			if currentChunk != "" {
				chunks = append(chunks, currentChunk)
			}

			if wordLength > MaxChunkLength {
				if isURL(word) {
					chunks = append(chunks, word)
					currentChunk = ""
					currentLength = 0
				} else {
					chunks = append(chunks, splitLongWord(word, MaxChunkLength)...)
					currentChunk = ""
					currentLength = 0
				}
			} else {
				currentChunk = word
				currentLength = wordLength
			}
		}
	}

	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}

func isURL(word string) bool {
	return strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://")
}

func splitLongWord(word string, maxLen int) []string {
	var chunks []string
	runes := []rune(word)

	for i := 0; i < len(runes); i += maxLen {
		end := i + maxLen
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}

	return chunks
}

func FormatThreadChunks(chunks []string) []ThreadChunk {
	if len(chunks) == 0 {
		return nil
	}

	total := len(chunks)
	result := make([]ThreadChunk, total)

	for i, chunk := range chunks {
		var formatted string
		if total > 1 {
			formatted = chunk + " (" + strings.TrimPrefix(itoa(i+1), "") + "/" + itoa(total) + ")"
		} else {
			formatted = chunk
		}

		result[i] = ThreadChunk{
			Text:   formatted,
			Number: i + 1,
			Total:  total,
		}
	}

	return result
}

func PrepareThread(text string) []ThreadChunk {
	chunks := SplitIntoChunks(text)
	return FormatThreadChunks(chunks)
}

func PrepareThreadWithOptions(text string, opts SplitOptions) []ThreadChunk {
	chunks := SplitIntoChunksWithOptions(text, opts)
	return FormatThreadChunks(chunks)
}

func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return itoa(i/10) + string(rune('0'+i%10))
}
