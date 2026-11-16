package dataplane

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zyeyefyass/CubeX/internal/netutil"
)

// StaleFlowKiller 负责在节点熔断或切线后，精确销毁绑定在旧节点上的僵尸/阻塞连接 (Stale Flow Killer)
// 特性：
// 1. 精确打击：仅检索并关闭 chains 中包含失效节点的连接，国内直连 (DIRECT) 与其它节点连接 100% 绝对零误伤；
// 2. 毫秒级闭环：调用 DELETE /connections/{id} 瞬时斩断死链，逼迫微信、视频流、HTTP/2、QUIC (UDP 443) 终端在 100ms 内向新节点重连；
// 3. 彻底根治“切了节点依然刷不出来、必须手动开关飞行模式”的世纪痛点。
type StaleFlowKiller struct {
	clashAPIBase   string
	clashAPISecret string
	httpClient     *http.Client
	mu             sync.Mutex
	lastKillTime   map[string]int64 // 节点防抖截流 (毫秒)
}

// NewStaleFlowKiller 创建陈旧连接精准清道夫
func NewStaleFlowKiller(clashAPIBase, secret string) *StaleFlowKiller {
	return &StaleFlowKiller{
		clashAPIBase:   strings.TrimRight(clashAPIBase, "/"),
		clashAPISecret: secret,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        32,
				MaxIdleConnsPerHost: 16,
				IdleConnTimeout:     60 * time.Second,
			},
		},
		lastKillTime: make(map[string]int64, 16),
	}
}

type clashConnectionsResponse struct {
	Connections []struct {
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
	} `json:"connections"`
}

// KillNodeConnections 精确杀死 chains 包含 targetNode 的所有活跃连接
func (sfk *StaleFlowKiller) KillNodeConnections(ctx context.Context, targetNode string) (int, error) {
	if sfk.clashAPIBase == "" || targetNode == "" || targetNode == "DIRECT" || targetNode == "REJECT" {
		return 0, nil
	}

	// 500ms 节点级别防抖，防止短时间内重入风暴
	now := time.Now().UnixMilli()
	sfk.mu.Lock()
	if last, ok := sfk.lastKillTime[targetNode]; ok && now-last < 500 {
		sfk.mu.Unlock()
		return 0, nil
	}
	sfk.lastKillTime[targetNode] = now
	sfk.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sfk.clashAPIBase+"/connections", nil)
	if err != nil {
		return 0, err
	}
	if sfk.clashAPISecret != "" {
		req.Header.Set("Authorization", "Bearer "+sfk.clashAPISecret)
	}

	resp, err := sfk.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fetch connections failed: %w", err)
	}
	defer netutil.DrainAndCloseBody(resp)

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status %d fetching connections", resp.StatusCode)
	}

	var data clashConnectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("decode connections failed: %w", err)
	}

	// 针对批量清理设定 800ms 强制超时与 8 并发信号量，防止海量连接时协程阻塞与 CPU 尖峰
	batchCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
	defer cancel()

	matchedIDs := make([]string, 0, 32)
	for _, c := range data.Connections {
		for _, chainItem := range c.Chains {
			if chainItem == targetNode {
				matchedIDs = append(matchedIDs, c.ID)
				break
			}
		}
	}

	if len(matchedIDs) == 0 {
		return 0, nil
	}

	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	var killMu sync.Mutex
	killed := 0

	for _, id := range matchedIDs {
		select {
		case <-batchCtx.Done():
			break
		case sem <- struct{}{}:
			wg.Add(1)
			go func(connID string) {
				defer func() {
					<-sem
					wg.Done()
				}()
				if err := sfk.deleteConnection(batchCtx, connID); err == nil {
					killMu.Lock()
					killed++
					killMu.Unlock()
				}
			}(id)
		}
	}
	wg.Wait()
	return killed, nil
}

// KillStalledConnections 扫描并关闭长时间有上行、0 下行的僵尸黑洞连接
func (sfk *StaleFlowKiller) KillStalledConnections(ctx context.Context) (int, error) {
	if sfk.clashAPIBase == "" {
		return 0, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sfk.clashAPIBase+"/connections", nil)
	if err != nil {
		return 0, err
	}
	if sfk.clashAPISecret != "" {
		req.Header.Set("Authorization", "Bearer "+sfk.clashAPISecret)
	}

	resp, err := sfk.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer netutil.DrainAndCloseBody(resp)

	var data clashConnectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}

	batchCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
	defer cancel()

	stalledIDs := make([]string, 0, 16)
	for _, c := range data.Connections {
		if c.Upload > 1024 && c.Download == 0 && len(c.Chains) > 0 && c.Chains[0] != "DIRECT" {
			stalledIDs = append(stalledIDs, c.ID)
		}
	}

	if len(stalledIDs) == 0 {
		return 0, nil
	}

	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	var killMu sync.Mutex
	killed := 0

	for _, id := range stalledIDs {
		select {
		case <-batchCtx.Done():
			break
		case sem <- struct{}{}:
			wg.Add(1)
			go func(connID string) {
				defer func() {
					<-sem
					wg.Done()
				}()
				if err := sfk.deleteConnection(batchCtx, connID); err == nil {
					killMu.Lock()
					killed++
					killMu.Unlock()
				}
			}(id)
		}
	}
	wg.Wait()
	return killed, nil
}

func (sfk *StaleFlowKiller) deleteConnection(ctx context.Context, connID string) error {
	delURL := fmt.Sprintf("%s/connections/%s", sfk.clashAPIBase, connID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, delURL, nil)
	if err != nil {
		return err
	}
	if sfk.clashAPISecret != "" {
		req.Header.Set("Authorization", "Bearer "+sfk.clashAPISecret)
	}

	resp, err := sfk.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer netutil.DrainAndCloseBody(resp)
	return nil
}
