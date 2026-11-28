package clashapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zyeyefyass/CubeX/internal/config"
	"github.com/zyeyefyass/CubeX/internal/netutil"
)

// ProxyEntry 描述 Clash REST API `/proxies` 返回的单个节点或策略组条目。
type ProxyEntry struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Now     string   `json:"now"`
	All     []string `json:"all"`
	History []struct {
		Delay int `json:"delay"`
	} `json:"history"`
}

// ConnectionEntry 描述 Clash REST API `/connections` 中的单条活跃连接。
type ConnectionEntry struct {
	ID       string   `json:"id"`
	Upload   int64    `json:"upload"`
	Download int64    `json:"download"`
	Chains   []string `json:"chains"`
	Rule     string   `json:"rule"`
	Metadata struct {
		Network       string `json:"network"`
		SourceIP      string `json:"sourceIP"`
		DestinationIP string `json:"destinationIP"`
		Host          string `json:"host"`
	} `json:"metadata"`
}

// ConnectionsResponse 为 `/connections` 响应体。
type ConnectionsResponse struct {
	Connections []ConnectionEntry `json:"connections"`
}

// ProxiesResponse 为 `/proxies` 响应体。
type ProxiesResponse struct {
	Proxies map[string]ProxyEntry `json:"proxies"`
}

// Client 统一封装 Mihomo / sing-box Clash REST API 访问，强制规范化 BaseURL 并统一超时与鉴权。
type Client struct {
	BaseURL    string
	Secret     string
	HTTPClient *http.Client
}

// NewClient 创建规范化的 Clash API 客户端（自动补全缺少的 http:// scheme 并截断尾部斜杠）。
func NewClient(baseURL, secret string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = netutil.NewBoundedHTTPClient(3 * time.Second)
	}
	return &Client{
		BaseURL:    config.NormalizeBaseURL(baseURL),
		Secret:     secret,
		HTTPClient: httpClient,
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string, body []byte) (*http.Request, error) {
	if c == nil || c.BaseURL == "" {
		return nil, fmt.Errorf("clash api base url is empty")
	}
	var bodyReader *bytes.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	var req *http.Request
	var err error
	if bodyReader != nil {
		req, err = http.NewRequestWithContext(ctx, method, c.BaseURL+path, bodyReader)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, c.BaseURL+path, nil)
	}
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.Secret)
	}
	return req, nil
}

// GetConnections 获取当前内核全部活跃连接。
func (c *Client) GetConnections(ctx context.Context) (*ConnectionsResponse, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/connections", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer netutil.DrainAndCloseBody(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d fetching connections", resp.StatusCode)
	}
	var out ConnectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteConnection 关闭指定 ID 的活跃连接。
func (c *Client) DeleteConnection(ctx context.Context, connID string) error {
	if strings.TrimSpace(connID) == "" {
		return nil
	}
	req, err := c.newRequest(ctx, http.MethodDelete, "/connections/"+url.PathEscape(connID), nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer netutil.DrainAndCloseBody(resp)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status %d deleting connection", resp.StatusCode)
	}
	return nil
}

// DeleteAllConnections 冲刷全部活跃连接（DELETE /connections）。
func (c *Client) DeleteAllConnections(ctx context.Context) error {
	req, err := c.newRequest(ctx, http.MethodDelete, "/connections", nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer netutil.DrainAndCloseBody(resp)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status %d flushing connections", resp.StatusCode)
	}
	return nil
}

// GetProxies 拉取全部策略组与节点快照。
func (c *Client) GetProxies(ctx context.Context) (map[string]ProxyEntry, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/proxies", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer netutil.DrainAndCloseBody(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d fetching proxies", resp.StatusCode)
	}
	var parsed ProxiesResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	return parsed.Proxies, nil
}

// SwitchProxy 切换指定策略组的当前出站节点。
func (c *Client) SwitchProxy(ctx context.Context, groupName, targetNode string) error {
	payload, err := json.Marshal(map[string]string{"name": targetNode})
	if err != nil {
		return err
	}
	req, err := c.newRequest(ctx, http.MethodPut, "/proxies/"+url.PathEscape(groupName), payload)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer netutil.DrainAndCloseBody(resp)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("clash api switch group %s returned status %d", groupName, resp.StatusCode)
	}
	return nil
}

// ProbeNodeDelay 通过 Clash API 对单个节点发起延迟测速。
func (c *Client) ProbeNodeDelay(ctx context.Context, nodeName, probeURL string, timeout time.Duration) (int, bool) {
	if c == nil || c.BaseURL == "" {
		return 0, false
	}
	u := fmt.Sprintf("/proxies/%s/delay?url=%s&timeout=%d",
		url.PathEscape(nodeName),
		url.QueryEscape(probeURL),
		timeout.Milliseconds(),
	)
	req, err := c.newRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, false
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, false
	}
	defer netutil.DrainAndCloseBody(resp)
	if resp.StatusCode != http.StatusOK {
		return 0, false
	}
	var data struct {
		Delay int `json:"delay"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil || data.Delay <= 0 {
		return 0, false
	}
	return data.Delay, true
}
