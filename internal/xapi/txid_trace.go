package xapi

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
)

type txIDTraceWriter struct {
	path string
	mode string
	ops  map[string]bool
	mu   sync.Mutex
	seen map[string]bool
}

type txIDTraceRecord struct {
	CapturedAt        string `json:"captured_at"`
	TimestampMS       int64  `json:"timestamp_ms"`
	Method            string `json:"method,omitempty"`
	Operation         string `json:"operation"`
	Path              string `json:"path"`
	TxID              string `json:"txid"`
	TxIDSalt          string `json:"txid_salt,omitempty"`
	TxIDVersion       byte   `json:"txid_version,omitempty"`
	CT0Hash           string `json:"ct0_hash,omitempty"`
	AuthorizationHash string `json:"authorization_hash,omitempty"`
	Source            string `json:"source,omitempty"`
	WallTime          string `json:"wall_time,omitempty"`
}

type txIDTraceMeta struct {
	URL         string
	Method      string
	TimestampMS int64
	WallTime    string
}

func newTxIDTraceWriter(path string, mode string, opsCSV string) *txIDTraceWriter {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		mode = "writes"
	}
	ops := map[string]bool{}
	for _, item := range strings.Split(opsCSV, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			ops[item] = true
		}
	}
	return &txIDTraceWriter{path: path, mode: mode, ops: ops, seen: map[string]bool{}}
}

func (w *txIDTraceWriter) Enabled() bool {
	return w != nil && w.path != ""
}

func (w *txIDTraceWriter) Record(meta txIDTraceMeta, headers network.Headers, source string) error {
	if !w.Enabled() {
		return nil
	}
	txid := headerValue(headers, "x-client-transaction-id")
	if txid == "" {
		return nil
	}
	path := meta.URL
	if path == "" {
		path = associatedPath(headers)
	}
	if path == "" {
		return nil
	}
	if !w.shouldCapture(meta, path) {
		return nil
	}

	operation := operationFromPath(path)
	if len(w.ops) > 0 && !w.ops[operation] {
		return nil
	}
	key := operation + "|" + txid

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.seen[key] {
		return nil
	}
	w.seen[key] = true

	record := txIDTraceRecord{
		CapturedAt:        time.Now().UTC().Format(time.RFC3339Nano),
		TimestampMS:       fallbackTimestamp(meta.TimestampMS),
		Method:            meta.Method,
		Operation:         operation,
		Path:              path,
		TxID:              txid,
		CT0Hash:           digestValue(headerValue(headers, "x-csrf-token")),
		AuthorizationHash: digestValue(headerValue(headers, "authorization")),
		Source:            source,
		WallTime:          meta.WallTime,
	}

	// Extract salt and version from decoded txid
	if decoded, err := base64.StdEncoding.DecodeString(padBase64(txid)); err == nil && len(decoded) >= 70 {
		record.TxIDVersion = decoded[0]
		record.TxIDSalt = fmt.Sprintf("%x", decoded[41:70])
	}

	line, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(w.path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

func (w *txIDTraceWriter) shouldCapture(meta txIDTraceMeta, path string) bool {
	if w == nil {
		return false
	}
	operation := operationFromPath(path)
	if len(w.ops) > 0 && !w.ops[operation] {
		return false
	}
	switch w.mode {
	case "all":
		return true
	case "writes", "mutations":
		method := strings.ToUpper(strings.TrimSpace(meta.Method))
		return method == http.MethodPost && strings.Contains(path, "/graphql/")
	default:
		return strings.Contains(path, "/graphql/")
	}
}

func fallbackTimestamp(v int64) int64 {
	if v > 0 {
		return v
	}
	return time.Now().UnixMilli()
}

func digestValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", sum[:8])
}

func headerValue(headers network.Headers, key string) string {
	for k, v := range headers {
		if strings.EqualFold(k, key) {
			return fmt.Sprint(v)
		}
	}
	return ""
}

func associatedPath(headers network.Headers) string {
	for _, key := range []string{":path", "x-request-path", "referer"} {
		if value := headerValue(headers, key); value != "" {
			return value
		}
	}
	return ""
}

func operationFromPath(path string) string {
	parts := strings.Split(strings.TrimSpace(path), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part != "" {
			return strings.Split(part, "?")[0]
		}
	}
	return path
}
