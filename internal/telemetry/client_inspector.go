package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// ClientTrafficSummary 汇总单台局域网终端设备的实时代理连接与流量占用
type ClientTrafficSummary struct {
	SourceIP        string `json:"source_ip"`
	ActiveConns     int    `json:"active_conns"`
	UploadTotal     int64  `json:"upload_total"`
	DownloadTotal   int64  `json:"download_total"`
	TopDestination  string `json:"top_destination"`
	PrimaryOutbound string `json:"primary_outbound"`
}

// DomainTrafficSummary 汇总单个目标域名/主机的高频访问统计
type DomainTrafficSummary struct {
	Host         string `json:"host"`
	ActiveConns  int    `json:"active_conns"`
	DownloadTotal int64 `json:"download_total"`
	MatchedRule  string `json:"matched_rule"`
	OutboundNode string `json:"outbound_node"`
}

// LANInspectorReport 返回给前端的极简定长透视报告（恒定 < 4KB，绝不卡死浏览器）
type LANInspectorReport struct {
	TotalActiveConns int                    `json:"total_active_conns"`
	TopClients       []ClientTrafficSummary `json:"top_clients"` // 最多返回前 10 台局域网设备
	TopDomains       []DomainTrafficSummary `json:"top_domains"` // 最多返回前 15 个目标域名
}

// ClientInspector 负责从内核 Clash API 零膨胀提取局域网终端流量画像
type ClientInspector struct {
	clashAPIBase   string
	clashAPISecret string
	httpClient     *http.Client
}

// NewClientInspector 初始化局域网终端流量画像透视器
func NewClientInspector(clashAPIBase, secret string) *ClientInspector {
	return &ClientInspector{
		clashAPIBase:   strings.TrimRight(clashAPIBase, "/"),
		clashAPISecret: secret,
		httpClient:     &http.Client{Timeout: 2 * time.Second},
	}
}

type rawClashConnSnapshot struct {
	Connections []struct {
		ID       string   `json:"id"`
		Upload   int64    `json:"upload"`
		Download int64    `json:"download"`
		Rule     string   `json:"rule"`
		Chains   []string `json:"chains"`
		Metadata struct {
			SourceIP      string `json:"sourceIP"`
			DestinationIP string `json:"destinationIP"`
			Host          string `json:"host"`
		} `json:"metadata"`
	} `json:"connections"`
}

// InspectTopLANActivity 聚合当前活跃连接，找出消耗流量最大的局域网设备与目标域名
func (ci *ClientInspector) InspectTopLANActivity(ctx context.Context) (*LANInspectorReport, error) {
	snap, err := ci.fetchRawConnections(ctx)
	if err != nil {
		return nil, err
	}

	clientMap := make(map[string]*ClientTrafficSummary, 16)
	domainMap := make(map[string]*DomainTrafficSummary, 32)

	for _, c := range snap.Connections {
		srcIP := c.Metadata.SourceIP
		if srcIP == "" {
			srcIP = "127.0.0.1"
		}
		targetHost := c.Metadata.Host
		if targetHost == "" {
			targetHost = c.Metadata.DestinationIP
		}
		outbound := "DIRECT"
		if len(c.Chains) > 0 {
			outbound = c.Chains[0]
		}

		// 1. 归并局域网源 IP 指标
		cs, ok := clientMap[srcIP]
		if !ok {
			cs = &ClientTrafficSummary{
				SourceIP:        srcIP,
				TopDestination:  targetHost,
				PrimaryOutbound: outbound,
			}
			clientMap[srcIP] = cs
		}
		cs.ActiveConns++
		cs.UploadTotal += c.Upload
		cs.DownloadTotal += c.Download

		// 2. 归并目标域名指标
		if targetHost != "" {
			ds, ok := domainMap[targetHost]
			if !ok {
				ds = &DomainTrafficSummary{
					Host:         targetHost,
					MatchedRule:  c.Rule,
					OutboundNode: outbound,
				}
				domainMap[targetHost] = ds
			}
			ds.ActiveConns++
			ds.DownloadTotal += c.Download
		}
	}

	clients := make([]ClientTrafficSummary, 0, len(clientMap))
	for _, v := range clientMap {
		clients = append(clients, *v)
	}
	sort.Slice(clients, func(i, j int) bool {
		if clients[i].DownloadTotal != clients[j].DownloadTotal {
			return clients[i].DownloadTotal > clients[j].DownloadTotal
		}
		return clients[i].ActiveConns > clients[j].ActiveConns
	})
	if len(clients) > 10 {
		clients = clients[:10]
	}

	domains := make([]DomainTrafficSummary, 0, len(domainMap))
	for _, v := range domainMap {
		domains = append(domains, *v)
	}
	sort.Slice(domains, func(i, j int) bool {
		if domains[i].DownloadTotal != domains[j].DownloadTotal {
			return domains[i].DownloadTotal > domains[j].DownloadTotal
		}
		return domains[i].ActiveConns > domains[j].ActiveConns
	})
	if len(domains) > 15 {
		domains = domains[:15]
	}

	return &LANInspectorReport{
		TotalActiveConns: len(snap.Connections),
		TopClients:       clients,
		TopDomains:       domains,
	}, nil
}

// KickClientConnections 一键精准斩断指定局域网设备 IP 的所有活跃代理连接（不影响家里其它设备）
func (ci *ClientInspector) KickClientConnections(ctx context.Context, targetSourceIP string) (int, error) {
	targetSourceIP = strings.TrimSpace(targetSourceIP)
	if targetSourceIP == "" {
		return 0, fmt.Errorf("empty source_ip")
	}

	snap, err := ci.fetchRawConnections(ctx)
	if err != nil {
		return 0, err
	}

	kicked := 0
	for _, c := range snap.Connections {
		if c.Metadata.SourceIP == targetSourceIP && c.ID != "" {
			req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
				fmt.Sprintf("%s/connections/%s", ci.clashAPIBase, c.ID), nil)
			if err != nil {
				continue
			}
			if ci.clashAPISecret != "" {
				req.Header.Set("Authorization", "Bearer "+ci.clashAPISecret)
			}
			if resp, err := ci.httpClient.Do(req); err == nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				kicked++
			}
		}
	}
	return kicked, nil
}

func (ci *ClientInspector) fetchRawConnections(ctx context.Context) (*rawClashConnSnapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ci.clashAPIBase+"/connections", nil)
	if err != nil {
		return nil, err
	}
	if ci.clashAPISecret != "" {
		req.Header.Set("Authorization", "Bearer "+ci.clashAPISecret)
	}

	resp, err := ci.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	var snap rawClashConnSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		return nil, err
	}
	return &snap, nil
}
