package sub

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/zyeyefyass/CubeX/internal/config"
	"github.com/zyeyefyass/CubeX/internal/netutil"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

const maxSubResponseBytes = 15 * 1024 * 1024

// FetchError 为订阅拉取提供结构化失败归因，严格只保留域名（脱敏隐藏路径与 token）。
type FetchError struct {
	Category  string `json:"category"`   // dns_failed | connect_timeout | tls_failed | http_status | body_too_large | parse_failed | invalid_url | network_error
	Domain    string `json:"domain"`     // desensitized host only
	UsedProxy bool   `json:"used_proxy"` // whether local mixed-port fallback was attempted
	Message   string `json:"message"`
	Err       error  `json:"-"`
}

func (e *FetchError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *FetchError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// ClassifyFetchError 提取订阅拉取错误的归因分类、域名与代理回退标记。
func ClassifyFetchError(err error) (category, domain string, usedProxy bool) {
	if err == nil {
		return "", "", false
	}
	var fe *FetchError
	if errors.As(err, &fe) {
		return fe.Category, fe.Domain, fe.UsedProxy
	}
	return classifyRawErrorString(err), "", false
}

func sanitizeNetworkErrDetail(err error, rawURL, domain string) string {
	if err == nil {
		return ""
	}
	var ue *url.Error
	msg := err.Error()
	if errors.As(err, &ue) && ue.Err != nil {
		msg = ue.Err.Error()
	}
	if rawURL != "" && strings.Contains(msg, rawURL) {
		if domain == "" {
			domain = "<redacted>"
		}
		msg = strings.ReplaceAll(msg, rawURL, domain)
	}
	return msg
}

func extractDesensitizedDomain(rawURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func classifyRawErrorString(err error) string {
	if err == nil {
		return ""
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "dns_failed"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "connect_timeout"
	}
	low := strings.ToLower(err.Error())
	switch {
	case strings.Contains(low, "no such host") || strings.Contains(low, "lookup "):
		return "dns_failed"
	case strings.Contains(low, "timeout") || strings.Contains(low, "deadline exceeded"):
		return "connect_timeout"
	case strings.Contains(low, "tls") || strings.Contains(low, "x509") || strings.Contains(low, "certificate"):
		return "tls_failed"
	case strings.Contains(low, "http status"):
		return "http_status"
	case strings.Contains(low, "exceeds 15mb") || strings.Contains(low, "body too large"):
		return "body_too_large"
	case strings.Contains(low, "parse") || strings.Contains(low, "no valid") || strings.Contains(low, "unmarshal") || strings.Contains(low, "empty"):
		return "parse_failed"
	default:
		return "network_error"
	}
}

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
	domain := extractDesensitizedDomain(urlStr)
	res, err := f.doFetch(urlStr, f.client)
	if err == nil {
		return res, nil
	}

	// Parse/HTTP/size errors should surface directly without futile proxy retries, unless network failed
	var directFE *FetchError
	if errors.As(err, &directFE) && (directFE.Category == "parse_failed" || directFE.Category == "body_too_large" || directFE.Category == "invalid_url") {
		return nil, directFE
	}

	// Automatic fallback to local core mixed-port if direct fetching times out or is blocked
	mixedAddr := config.GlobalRuntimeSpec.MixedProxyAddr
	if !strings.Contains(mixedAddr, "://") {
		mixedAddr = "http://" + mixedAddr
	}
	proxyURL, pErr := url.Parse(mixedAddr)
	if pErr == nil {
		proxyTr := &http.Transport{
			Proxy:           http.ProxyURL(proxyURL),
			DialContext:     CreateFastDualStackDialer(),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		defer proxyTr.CloseIdleConnections()
		proxyClient := &http.Client{
			Transport: proxyTr,
			Timeout:   20 * time.Second,
		}
		resProxy, errProxy := f.doFetch(urlStr, proxyClient)
		if errProxy == nil {
			return resProxy, nil
		}
		var proxyFE *FetchError
		if errors.As(errProxy, &proxyFE) {
			proxyFE.UsedProxy = true
			if proxyFE.Domain == "" {
				proxyFE.Domain = domain
			}
			// Keep direct error category if proxy itself refused connection because core wasn't up yet
			if directFE != nil && strings.Contains(strings.ToLower(proxyFE.Message), "connection refused") {
				directFE.UsedProxy = true
				return nil, directFE
			}
			return nil, proxyFE
		}
	}

	if directFE != nil {
		return nil, directFE
	}
	return nil, err
}

func (f *Fetcher) doFetch(urlStr string, client *http.Client) (*FetchResult, error) {
	startTime := time.Now()
	domain := extractDesensitizedDomain(urlStr)
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, &FetchError{
			Category: "invalid_url",
			Domain:   domain,
			Message:  fmt.Sprintf("invalid subscription url: %v", err),
			Err:      err,
		}
	}

	singboxVer, _ := config.MetaCoreVersions()
	req.Header.Set("User-Agent", fmt.Sprintf("Clash.Meta; sing-box/%s; CubeX-Universal/1.0", strings.TrimPrefix(singboxVer, "v")))

	resp, err := client.Do(req)
	if err != nil {
		cat := classifyRawErrorString(err)
		return nil, &FetchError{
			Category: cat,
			Domain:   domain,
			Message:  fmt.Sprintf("network request failed (%s) for host %s: %s", cat, domain, sanitizeNetworkErrDetail(err, urlStr, domain)),
			Err:      err,
		}
	}
	defer netutil.DrainAndCloseBody(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, &FetchError{
			Category: "http_status",
			Domain:   domain,
			Message:  fmt.Sprintf("subscription server returned HTTP status %d", resp.StatusCode),
		}
	}

	// Guard against memory explosion with 15MB limit reader (+1 byte overflow sentinel)
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSubResponseBytes+1))
	if err != nil {
		return nil, &FetchError{
			Category: classifyRawErrorString(err),
			Domain:   domain,
			Message:  fmt.Sprintf("failed to read response body for host %s: %s", domain, sanitizeNetworkErrDetail(err, urlStr, domain)),
			Err:      err,
		}
	}
	if len(body) > maxSubResponseBytes {
		return nil, &FetchError{
			Category: "body_too_large",
			Domain:   domain,
			Message:  "subscription response exceeds 15MB limit (body_too_large)",
		}
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
		return nil, &FetchError{
			Category: "parse_failed",
			Domain:   domain,
			Message:  fmt.Sprintf("subscription parse failed: %v", err),
			Err:      err,
		}
	}

	res.Nodes = nodes
	res.NodeCount = len(nodes)
	return res, nil
}
