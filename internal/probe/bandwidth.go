// Package probe provides lightweight, quota-safe network probing, bandwidth sampling,
// IP intelligence, and concurrency watchdog routines for CubeX.
package probe

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var (
	ErrQuotaExceeded = errors.New("daily bandwidth test quota exceeded")
	ErrTestTimeout   = errors.New("bandwidth probe timed out")
)

// BandwidthResult represents the throughput measurement of a node.
type BandwidthResult struct {
	BytesSampled   int64     `json:"bytes_sampled"`
	DurationMs     int64     `json:"duration_ms"`
	ThroughputMbps float64   `json:"throughput_mbps"`
	SpeedGrade     string    `json:"speed_grade"` // "Ultra (4K/8K)", "Fast", "Medium", "Basic"
	TestedAt       time.Time `json:"tested_at"`
	Cached         bool      `json:"cached"`
}

// QuotaConfig defines safety boundaries to prevent depleting user subscription traffic.
type QuotaConfig struct {
	MaxChunkBytes   int64 // Max bytes to download per test (e.g., 2MB)
	DailyQuotaBytes int64 // Max total bytes per 24 hours (e.g., 50MB)
	CacheTTL        time.Duration
}

// DefaultQuotaConfig provides conservative defaults.
var DefaultQuotaConfig = QuotaConfig{
	MaxChunkBytes:   2 * 1024 * 1024,  // 2 MB chunk
	DailyQuotaBytes: 50 * 1024 * 1024, // 50 MB / day
	CacheTTL:        6 * time.Hour,    // Reuse results for 6 hours
}

// BandwidthProber safely measures downstream throughput through a proxy.
type BandwidthProber struct {
	mu         sync.RWMutex
	cfg        QuotaConfig
	dailyUsage int64
	lastReset  time.Time
	cache      map[string]BandwidthResult // key: proxy identifier
}

// NewBandwidthProber creates an initialized prober instance.
func NewBandwidthProber(cfg QuotaConfig) *BandwidthProber {
	if cfg.MaxChunkBytes <= 0 {
		cfg.MaxChunkBytes = DefaultQuotaConfig.MaxChunkBytes
	}
	if cfg.DailyQuotaBytes <= 0 {
		cfg.DailyQuotaBytes = DefaultQuotaConfig.DailyQuotaBytes
	}
	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = DefaultQuotaConfig.CacheTTL
	}
	return &BandwidthProber{
		cfg:       cfg,
		lastReset: time.Now(),
		cache:     make(map[string]BandwidthResult),
	}
}

// Probe evaluates downstream throughput through proxyURL using a range request.
func (p *BandwidthProber) Probe(ctx context.Context, proxyURL, testTargetURL string, forceRefresh bool) (*BandwidthResult, error) {
	if testTargetURL == "" {
		testTargetURL = "https://speed.cloudflare.com/__down?bytes=2097152"
	}

	p.mu.Lock()
	// Reset daily quota at 24h boundary
	if time.Since(p.lastReset) > 24*time.Hour {
		p.dailyUsage = 0
		p.lastReset = time.Now()
	}

	// Check cache
	if !forceRefresh {
		if cached, ok := p.cache[proxyURL]; ok && time.Since(cached.TestedAt) < p.cfg.CacheTTL {
			p.mu.Unlock()
			cached.Cached = true
			return &cached, nil
		}
	}

	// Check daily quota
	if p.dailyUsage+p.cfg.MaxChunkBytes > p.cfg.DailyQuotaBytes {
		p.mu.Unlock()
		return nil, ErrQuotaExceeded
	}
	p.mu.Unlock()

	// Configure HTTP client with proxy
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if proxyURL != "" {
		parsedProxy, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy url: %w", err)
		}
		transport.Proxy = http.ProxyURL(parsedProxy)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testTargetURL, nil)
	if err != nil {
		return nil, err
	}
	// Range request: 0 to MaxChunkBytes-1
	req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", p.cfg.MaxChunkBytes-1))
	req.Header.Set("User-Agent", "CubeX-Probe/5.5")

	t0 := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("probe request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("server returned unexpected status %d", resp.StatusCode)
	}

	// Read up to MaxChunkBytes
	limitReader := io.LimitReader(resp.Body, p.cfg.MaxChunkBytes)
	buf := make([]byte, 32*1024)
	var totalRead int64

	for {
		n, rErr := limitReader.Read(buf)
		if n > 0 {
			totalRead += int64(n)
		}
		if rErr != nil {
			if rErr == io.EOF {
				break
			}
			return nil, fmt.Errorf("probe read failed: %w", rErr)
		}
	}

	duration := time.Since(t0)
	if duration <= 0 {
		duration = time.Millisecond
	}

	// Record daily quota
	p.mu.Lock()
	p.dailyUsage += totalRead
	p.mu.Unlock()

	// Calculate Throughput (Mbps)
	bits := float64(totalRead) * 8.0
	seconds := duration.Seconds()
	mbps := (bits / seconds) / 1_000_000.0

	grade := "Basic"
	switch {
	case mbps >= 100.0:
		grade = "Ultra (4K/8K)"
	case mbps >= 50.0:
		grade = "Fast (4K)"
	case mbps >= 20.0:
		grade = "Medium (1080P)"
	default:
		grade = "Basic"
	}

	res := BandwidthResult{
		BytesSampled:   totalRead,
		DurationMs:     duration.Milliseconds(),
		ThroughputMbps: mbps,
		SpeedGrade:     grade,
		TestedAt:       time.Now(),
		Cached:         false,
	}

	p.mu.Lock()
	p.cache[proxyURL] = res
	p.mu.Unlock()

	return &res, nil
}

// GetDailyUsage returns current usage and remaining quota in bytes.
func (p *BandwidthProber) GetDailyUsage() (used int64, limit int64) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.dailyUsage, p.cfg.DailyQuotaBytes
}
