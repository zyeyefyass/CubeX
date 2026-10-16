package sub

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

// Fetcher downloads and synchronizes subscription nodes.
type Fetcher struct {
	client *http.Client
}

// NewFetcher creates a subscription fetcher with high-speed timeout and TLS configuration.
func NewFetcher() *Fetcher {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &Fetcher{
		client: &http.Client{Transport: tr, Timeout: 30 * time.Second},
	}
}

// FetchResult contains extracted subscription nodes and quota telemetry.
type FetchResult struct {
	Nodes      []schema.ProxyNode `json:"nodes"`
	Upload     int64              `json:"upload"`
	Download   int64              `json:"download"`
	Total      int64              `json:"total"`
	Expire     int64              `json:"expire"`
	NodeCount  int                `json:"node_count"`
	DurationMs int64              `json:"duration_ms"`
}

// Fetch downloads subscription content from a URL and extracts standard ProxyNodes.
func (f *Fetcher) Fetch(urlStr string) (*FetchResult, error) {
	startTime := time.Now()
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid subscription url: %w", err)
	}

	req.Header.Set("User-Agent", "Clash.Meta; sing-box/1.14.3; CubeX-Universal/1.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subscription server returned HTTP status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	res := &FetchResult{
		DurationMs: time.Since(startTime).Milliseconds(),
	}

	// Parse subscription userinfo header (e.g. upload=1234; download=5678; total=99999; expire=1700000000)
	userinfo := resp.Header.Get("Subscription-Userinfo")
	if userinfo != "" {
		parts := strings.Split(userinfo, ";")
		for _, part := range parts {
			kv := strings.Split(strings.TrimSpace(part), "=")
			if len(kv) == 2 {
				k := strings.ToLower(kv[0])
				v, _ := strconv.ParseInt(kv[1], 10, 64)
				switch k {
				case "upload":
					res.Upload = v
				case "download":
					res.Download = v
				case "total":
					res.Total = v
				case "expire":
					res.Expire = v
				}
			}
		}
	}

	nodes, err := ParseSubscription(body)
	if err != nil {
		return nil, err
	}

	res.Nodes = nodes
	res.NodeCount = len(nodes)
	return res, nil
}
