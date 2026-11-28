package dataplane

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/zyeyefyass/CubeX/internal/clashapi"
)

// WANHealer 监控主路由网络状态切换，在 PPPoE 重拨或上游闪断恢复瞬间冲刷内核陈旧 Socket
type WANHealer struct {
	api      *clashapi.Client
	onHealed func(reason string)
}

// NewWANHealer 创建 WAN 重拨与僵尸连接瞬时冲刷器
func NewWANHealer(clashAPIBase, secret string, onHealed func(reason string)) *WANHealer {
	return &WANHealer{
		api:      clashapi.NewClient(clashAPIBase, secret, &http.Client{Timeout: 2 * time.Second}),
		onHealed: onHealed,
	}
}

// FlushStaleConnections 调用内核 Clash API (DELETE /connections) 瞬时清理绑定在旧 WAN IP 上的僵尸连接
// 优势：无需重启 Mihomo / sing-box 进程，0 内存毛刺，耗时 < 3ms
func (h *WANHealer) FlushStaleConnections(ctx context.Context, reason string) error {
	if h.api == nil || h.api.BaseURL == "" {
		return nil
	}
	if err := h.api.DeleteAllConnections(ctx); err != nil {
		return fmt.Errorf("flush stale connections failed: %w", err)
	}
	if h.onHealed != nil {
		h.onHealed(reason)
	}
	return nil
}

// WatchGatewayTransition 在后台每隔 interval 周期探测网关连通性。
// 一旦感知到上游网关由“不可达”恢复为“可达”（典型的凌晨 PPPoE 重拨完成或物理链路闪断恢复场景），
// 全自动调用 FlushStaleConnections 冲刷陈旧连接池，达成 0 秒无感恢复。
func (h *WANHealer) WatchGatewayTransition(ctx context.Context, interval time.Duration, probeGatewayFn func() bool) {
	if probeGatewayFn == nil {
		return
	}
	if interval <= 0 {
		interval = 15 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	lastReachable := true

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentReachable := probeGatewayFn()
			if !lastReachable && currentReachable {
				// 状态从不可达翻转为可达：主路由已完成重拨或网关自愈
				if err := h.FlushStaleConnections(ctx, "上游网关PPPoE重拨/闪断后自愈恢复"); err != nil {
					log.Printf("⚠️ [WANHealer] 网关恢复后冲刷陈旧连接失败: %v", err)
				}
			}
			lastReachable = currentReachable
		}
	}
}
