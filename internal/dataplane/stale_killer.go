package dataplane

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/zyeyefyass/CubeX/internal/clashapi"
)

// StaleFlowKiller 负责在节点熔断或切线后，精确销毁绑定在旧节点上的僵尸/阻塞连接 (Stale Flow Killer)
type StaleFlowKiller struct {
	api          *clashapi.Client
	mu           sync.Mutex
	lastKillTime map[string]int64 // 节点防抖截流 (毫秒)
}

// NewStaleFlowKiller 创建陈旧连接精准清道夫
func NewStaleFlowKiller(clashAPIBase, secret string) *StaleFlowKiller {
	httpClient := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        32,
			MaxIdleConnsPerHost: 16,
			IdleConnTimeout:     60 * time.Second,
		},
	}
	return &StaleFlowKiller{
		api:          clashapi.NewClient(clashAPIBase, secret, httpClient),
		lastKillTime: make(map[string]int64, 16),
	}
}

func (sfk *StaleFlowKiller) deleteBatch(ctx context.Context, ids []string) int {
	if len(ids) == 0 {
		return 0
	}
	// 针对批量清理设定 800ms 强制超时与 8 并发信号量，防止海量连接时协程阻塞与 CPU 尖峰
	batchCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
	defer cancel()

	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	var killMu sync.Mutex
	killed := 0
	failed := 0
	var firstErr error

loop:
	for _, id := range ids {
		select {
		case <-batchCtx.Done():
			break loop
		case sem <- struct{}{}:
			wg.Add(1)
			go func(connID string) {
				defer func() {
					<-sem
					wg.Done()
				}()
				if err := sfk.api.DeleteConnection(batchCtx, connID); err == nil {
					killMu.Lock()
					killed++
					killMu.Unlock()
				} else {
					killMu.Lock()
					failed++
					if firstErr == nil {
						firstErr = err
					}
					killMu.Unlock()
				}
			}(id)
		}
	}
	wg.Wait()
	if failed > 0 {
		log.Printf("⚠️ [StaleFlowKiller] 批量断连部分失败 (%d/%d): %v", failed, len(ids), firstErr)
	}
	return killed
}

// KillNodeConnections 精确杀死 chains 包含 targetNode 的所有活跃连接
func (sfk *StaleFlowKiller) KillNodeConnections(ctx context.Context, targetNode string) (int, error) {
	if sfk.api == nil || sfk.api.BaseURL == "" || targetNode == "" || targetNode == "DIRECT" || targetNode == "REJECT" {
		return 0, nil
	}

	// 500ms 节点级别防抖，防止短时间内重入风暴，并定期清理超过 60s 的陈旧记录
	now := time.Now().UnixMilli()
	sfk.mu.Lock()
	if last, ok := sfk.lastKillTime[targetNode]; ok && now-last < 500 {
		sfk.mu.Unlock()
		return 0, nil
	}
	for k, ts := range sfk.lastKillTime {
		if now-ts > 60000 {
			delete(sfk.lastKillTime, k)
		}
	}
	sfk.lastKillTime[targetNode] = now
	sfk.mu.Unlock()

	data, err := sfk.api.GetConnections(ctx)
	if err != nil {
		return 0, fmt.Errorf("fetch connections failed: %w", err)
	}

	matchedIDs := make([]string, 0, 32)
	for _, c := range data.Connections {
		for _, chainItem := range c.Chains {
			if chainItem == targetNode {
				matchedIDs = append(matchedIDs, c.ID)
				break
			}
		}
	}
	return sfk.deleteBatch(ctx, matchedIDs), nil
}

// KillStalledConnections 扫描并关闭长时间有上行、0 下行的僵尸黑洞连接
func (sfk *StaleFlowKiller) KillStalledConnections(ctx context.Context) (int, error) {
	if sfk.api == nil || sfk.api.BaseURL == "" {
		return 0, nil
	}

	data, err := sfk.api.GetConnections(ctx)
	if err != nil {
		return 0, err
	}

	stalledIDs := make([]string, 0, 16)
	for _, c := range data.Connections {
		if c.Upload > 1024 && c.Download == 0 && len(c.Chains) > 0 && c.Chains[0] != "DIRECT" {
			stalledIDs = append(stalledIDs, c.ID)
		}
	}
	return sfk.deleteBatch(ctx, stalledIDs), nil
}

// KillClientConnections 立即斩断指定源 IP 设备的所有活跃代理连接 (0.1秒秒切生效)
func (sfk *StaleFlowKiller) KillClientConnections(ctx context.Context, clientIP string) (int, error) {
	if sfk.api == nil || sfk.api.BaseURL == "" || clientIP == "" || clientIP == "-" {
		return 0, nil
	}

	data, err := sfk.api.GetConnections(ctx)
	if err != nil {
		return 0, fmt.Errorf("fetch connections failed: %w", err)
	}

	matchedIDs := make([]string, 0, 16)
	for _, c := range data.Connections {
		if c.Metadata.SourceIP == clientIP {
			matchedIDs = append(matchedIDs, c.ID)
		}
	}
	return sfk.deleteBatch(ctx, matchedIDs), nil
}
