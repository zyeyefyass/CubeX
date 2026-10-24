package sub

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

// CreateFastDualStackDialer returns a dial function that prioritizes IPv4 addresses,
// preventing hanging on unreachable IPv6 routes when DNS returns AAAA records (Happy Eyeballs fast fallback).
func CreateFastDualStackDialer() func(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout:       8 * time.Second,
		KeepAlive:     30 * time.Second,
		FallbackDelay: 150 * time.Millisecond,
	}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return dialer.DialContext(ctx, network, addr)
		}
		// Resolve IPs and prioritize IPv4 to eliminate timeout delays on unreachable IPv6 routes
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err == nil && len(ips) > 0 {
			var v4s []net.IP
			var v6s []net.IP
			for _, ip := range ips {
				if ip.To4() != nil {
					v4s = append(v4s, ip)
				} else {
					v6s = append(v6s, ip)
				}
			}
			sortedIPs := append(v4s, v6s...)
			var lastErr error
			for _, ip := range sortedIPs {
				targetAddr := net.JoinHostPort(ip.String(), port)
				conn, err := dialer.DialContext(ctx, network, targetAddr)
				if err == nil {
					return conn, nil
				}
				lastErr = err
			}
			if lastErr != nil {
				return nil, lastErr
			}
		}
		return dialer.DialContext(ctx, network, addr)
	}
}

// Fetcher downloads and synchronizes subscription nodes.
type Fetcher struct {
	client *http.Client
}

// NewFetcher creates a subscription fetcher with high-speed timeout, IPv4-first dialing and TLS configuration.
func NewFetcher() *Fetcher {
	tr := &http.Transport{
		DialContext:     CreateFastDualStackDialer(),
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
	res, err := f.doFetch(urlStr, f.client)
	if err == nil {
		return res, nil
	}

	// Automatic fallback to local core mixed-port (7890) if direct fetching times out or is blocked by GFW
	proxyURL, pErr := url.Parse("http://127.0.0.1:7890")
	if pErr == nil {
		proxyClient := &http.Client{
			Transport: &http.Transport{
				Proxy:           http.ProxyURL(proxyURL),
				DialContext:     CreateFastDualStackDialer(),
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: 20 * time.Second,
		}
		if resProxy, errProxy := f.doFetch(urlStr, proxyClient); errProxy == nil {
			return resProxy, nil
		}
	}

	return nil, err
}

func (f *Fetcher) doFetch(urlStr string, client *http.Client) (*FetchResult, error) {
	startTime := time.Now()
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid subscription url: %w", err)
	}

	req.Header.Set("User-Agent", "Clash.Meta; sing-box/1.14.4; CubeX-Universal/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subscription server returned HTTP status %d", resp.StatusCode)
	}

	// Guard against memory explosion with 15MB limit reader
	body, err := io.ReadAll(io.LimitReader(resp.Body, 15*1024*1024))
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
