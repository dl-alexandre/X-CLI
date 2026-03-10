package xapi

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	utls "github.com/refraction-networking/utls"
)

const txidEpoch = 1682924400

type TransactionIDProvider interface {
	Name() string
	Ready() bool
	Generate(ctx context.Context, method string, requestURL string) (string, error)
}

type SaltMode int

const (
	SaltModePlaceholder SaltMode = iota
	SaltModeStatic
	SaltModeRotating
	SaltModeSniffer
)

type NativeTransactionProvider struct {
	active           bool
	saltMode         SaltMode
	staticSalt       []byte
	snifferPath      string
	saltMu           sync.RWMutex
	lastSalt         []byte
	saltExpires      time.Time
	whiteningEnabled bool
}

func NewNativeTransactionProvider() *NativeTransactionProvider {
	return &NativeTransactionProvider{
		active:   true,
		saltMode: SaltModePlaceholder,
	}
}

func (p *NativeTransactionProvider) Name() string { return "native-transaction-provider" }
func (p *NativeTransactionProvider) Ready() bool  { return p.active }

func (p *NativeTransactionProvider) HasStaticSalt() bool {
	p.saltMu.RLock()
	defer p.saltMu.RUnlock()
	return p.saltMode == SaltModeStatic && len(p.staticSalt) == 29
}

func (p *NativeTransactionProvider) SetStaticSalt(hexSalt string) error {
	if hexSalt == "" {
		return nil
	}
	salt, err := decodeHexSalt(hexSalt)
	if err != nil {
		return fmt.Errorf("invalid hex salt: %w", err)
	}
	p.saltMu.Lock()
	p.staticSalt = salt
	p.saltMode = SaltModeStatic
	p.saltMu.Unlock()
	return nil
}

func (p *NativeTransactionProvider) SetSnifferMode(traceFile string) {
	p.saltMu.Lock()
	p.snifferPath = traceFile
	p.saltMode = SaltModeSniffer
	p.saltMu.Unlock()
}

func (p *NativeTransactionProvider) Generate(ctx context.Context, method string, requestURL string) (string, error) {
	u, err := url.Parse(requestURL)
	if err != nil {
		return "", fmt.Errorf("parse request url: %w", err)
	}

	e := calculateE()
	path := u.Path
	if u.RawQuery != "" {
		path = path + "?" + u.RawQuery
	}

	hashInput := fmt.Sprintf("%s!%s!%d", strings.ToUpper(method), path, e)
	digest := sha256.Sum256([]byte(hashInput))

	buffer := make([]byte, 70)
	buffer[0] = 0x01
	copy(buffer[1:33], digest[:])
	binary.BigEndian.PutUint64(buffer[33:41], uint64(e))

	salt := p.getSalt()
	copy(buffer[41:70], salt)

	// Apply whitening if enabled (for testing)
	if p.whiteningEnabled {
		buffer = p.whiten(buffer, salt)
	}

	encoded := base64.StdEncoding.EncodeToString(buffer)
	return encoded, nil
}

func (p *NativeTransactionProvider) getSalt() []byte {
	p.saltMu.RLock()
	mode := p.saltMode
	static := p.staticSalt
	snifferPath := p.snifferPath
	p.saltMu.RUnlock()

	switch mode {
	case SaltModeStatic:
		if len(static) == 29 {
			return static
		}
	case SaltModeSniffer:
		if salt := p.sniffSalt(snifferPath); len(salt) == 29 {
			return salt
		}
	case SaltModeRotating:
		if salt := p.getRotatingSalt(); len(salt) == 29 {
			return salt
		}
	}

	// Fallback to placeholder
	return derivePlaceholderSalt()
}

func (p *NativeTransactionProvider) sniffSalt(traceFile string) []byte {
	if traceFile == "" {
		return nil
	}

	// Check if file exists and is recent (< 5 minutes old)
	info, err := os.Stat(traceFile)
	if err != nil {
		return nil
	}
	if time.Since(info.ModTime()) > 5*time.Minute {
		return nil
	}

	samples, err := ExtractSalts(traceFile)
	if err != nil || len(samples) == 0 {
		return nil
	}

	// Use the most recent salt
	latest := samples[len(samples)-1]
	salt, _ := decodeHexSalt(latest.Salt)
	return salt
}

func (p *NativeTransactionProvider) getRotatingSalt() []byte {
	p.saltMu.RLock()
	lastSalt := p.lastSalt
	expires := p.saltExpires
	p.saltMu.RUnlock()

	// If we have a valid cached salt, use it
	if len(lastSalt) == 29 && time.Now().Before(expires) {
		return lastSalt
	}

	// Future: TOTP-style rotation based on time window
	// Currently falls back to sniffer or placeholder
	return nil
}

func (p *NativeTransactionProvider) UpdateSaltFromTrace(traceFile string) bool {
	if salt := p.sniffSalt(traceFile); len(salt) == 29 {
		p.saltMu.Lock()
		p.lastSalt = salt
		p.saltExpires = time.Now().Add(60 * time.Second)
		p.saltMu.Unlock()
		return true
	}
	return false
}

// whiten applies XOR-based whitening to the buffer
// Skip version byte (index 0), XOR bytes 1-69 with salt in circular fashion
func (p *NativeTransactionProvider) whiten(buffer []byte, salt []byte) []byte {
	if len(buffer) != 70 || len(salt) != 29 {
		return buffer
	}

	result := make([]byte, 70)
	copy(result, buffer)

	// XOR bytes 1-69 with salt
	for i := 1; i < 70; i++ {
		result[i] ^= salt[(i-1)%29]
	}

	return result
}

// EnableWhitening turns on XOR whitening for testing
func (p *NativeTransactionProvider) EnableWhitening() {
	p.whiteningEnabled = true
}

// DisableWhitening turns off XOR whitening (default)
func (p *NativeTransactionProvider) DisableWhitening() {
	p.whiteningEnabled = false
}

func calculateE() int64 {
	now := time.Now().UnixMilli()
	return (now - txidEpoch*1000) / 1000
}

func derivePlaceholderSalt() []byte {
	salt := make([]byte, 29)
	for i := range salt {
		salt[i] = byte(i*7 + 0x42)
	}
	return salt
}

func decodeHexSalt(hex string) ([]byte, error) {
	if len(hex) != 58 { // 29 bytes = 58 hex chars
		return nil, fmt.Errorf("expected 58 hex chars, got %d", len(hex))
	}

	salt := make([]byte, 29)
	for i := 0; i < 29; i++ {
		b1 := hexCharValue(hex[i*2])
		b2 := hexCharValue(hex[i*2+1])
		if b1 < 0 || b2 < 0 {
			return nil, fmt.Errorf("invalid hex char at position %d", i*2)
		}
		salt[i] = byte(b1<<4 | b2)
	}
	return salt, nil
}

func hexCharValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c - 'a' + 10)
	case c >= 'A' && c <= 'F':
		return int(c - 'A' + 10)
	default:
		return -1
	}
}

func newNativeHTTPTransport(proxyValue string) *http.Transport {
	transport := &http.Transport{
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   20 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	if strings.TrimSpace(proxyValue) != "" {
		if proxyURL, err := url.Parse(proxyValue); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	} else {
		transport.Proxy = http.ProxyFromEnvironment
	}

	transport.DialTLSContext = func(ctx context.Context, networkName string, addr string) (net.Conn, error) {
		dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
		rawConn, err := dialer.DialContext(ctx, networkName, addr)
		if err != nil {
			return nil, err
		}

		host := addr
		if h, _, err := net.SplitHostPort(addr); err == nil {
			host = h
		}

		config := &utls.Config{ServerName: host, NextProtos: []string{"http/1.1"}}
		conn := utls.UClient(rawConn, config, utls.HelloChrome_Auto)
		if err := conn.HandshakeContext(ctx); err != nil {
			_ = rawConn.Close()
			return nil, err
		}
		state := conn.ConnectionState()
		if !state.HandshakeComplete {
			_ = rawConn.Close()
			return nil, fmt.Errorf("uTLS handshake incomplete")
		}
		if len(config.ServerName) > 0 && !strings.EqualFold(state.ServerName, config.ServerName) && state.ServerName != "" {
			_ = rawConn.Close()
			return nil, fmt.Errorf("unexpected tls server name %q", state.ServerName)
		}
		return conn, nil
	}

	return transport
}
