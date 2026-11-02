package probe

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// IPQualityResult contains intelligence on the outbound node's IP address.
type IPQualityResult struct {
	IP          string   `json:"ip"`
	Country     string   `json:"country"`
	City        string   `json:"city"`
	ASN         string   `json:"asn"`
	Org         string   `json:"org"`
	IPType      string   `json:"ip_type"`     // "Residential" (原生住宅), "Datacenter" (商业机房), "Business" (企业专线)
	RiskLevel   string   `json:"risk_level"`  // "Clean" (极净), "Low", "Medium", "High"
	RiskScore   int      `json:"risk_score"`  // 0 - 100 (0=cleanest, 100=blacklisted)
	Features    []string `json:"features"`    // ["Google Gemini", "OpenAI", "GitHub Copilot", "4K Streaming"]
	TestedAt    time.Time`json:"tested_at"`
}

// IPQualityProber probes IP intelligence and capability matrix.
type IPQualityProber struct {
	client *http.Client
}

// NewIPQualityProber creates a new IP quality prober.
func NewIPQualityProber() *IPQualityProber {
	return &IPQualityProber{
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

// Inspect queries outbound IP intelligence through the specified proxy.
func (p *IPQualityProber) Inspect(ctx context.Context, proxyURL string) (*IPQualityResult, error) {
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

	customClient := &http.Client{
		Transport: transport,
		Timeout:   6 * time.Second,
	}

	// 1. Query IP intelligence
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ip.sb/geoip", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "CubeX-IPQuality/5.5")

	resp, err := customClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ip info query failed: %w", err)
	}
	defer resp.Body.Close()

	var geo struct {
		IP           string  `json:"ip"`
		Country      string  `json:"country"`
		City         string  `json:"city"`
		ASN          int     `json:"asn"`
		Org          string  `json:"organization"`
		ISP          string  `json:"isp"`
		ASNOrg       string  `json:"asn_organization"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geo); err != nil {
		return nil, fmt.Errorf("decode geo response failed: %w", err)
	}

	// 2. Classify IP type & risk based on ISP/Org heuristics
	orgLower := strings.ToLower(geo.Org + " " + geo.ISP + " " + geo.ASNOrg)
	ipType := "Residential"
	riskScore := 10
	riskLevel := "Clean"

	datacenterKeywords := []string{"amazon", "aws", "google", "cloud", "microsoft", "azure", "alibaba", "oracle", "digitalocean", "linode", "vultr", "ovh", "hetzner", "choopa", "m247", "datacenter", "hosting"}
	for _, kw := range datacenterKeywords {
		if strings.Contains(orgLower, kw) {
			ipType = "Datacenter"
			riskScore = 35
			riskLevel = "Low"
			break
		}
	}

	features := []string{"Google Gemini", "GitHub Copilot"}
	if ipType == "Residential" {
		features = append(features, "OpenAI", "4K Streaming")
		riskScore = 5
		riskLevel = "Clean"
	} else {
		features = append(features, "跨境业务协同")
	}

	return &IPQualityResult{
		IP:        geo.IP,
		Country:   geo.Country,
		City:      geo.City,
		ASN:       fmt.Sprintf("AS%d", geo.ASN),
		Org:       geo.Org,
		IPType:    ipType,
		RiskLevel: riskLevel,
		RiskScore: riskScore,
		Features:  features,
		TestedAt:  time.Now(),
	}, nil
}
