package sub

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

// SyncOutcome 定义单次订阅同步的执行裁决结果
type SyncOutcome string

const (
	OutcomeSkippedUnchanged SyncOutcome = "SKIPPED_UNCHANGED" // 节点指纹未变，仅刷新流量配额，0 内核重载
	OutcomeAppliedAndReload SyncOutcome = "APPLIED_RELOADED"  // 节点发生实质变更，已触发平滑重载
	OutcomeCircuitBroken    SyncOutcome = "CIRCUIT_BROKEN"    // 触发空节点或暴跌熔断保护，保留旧配置
)

// QuotaTelemetry 记录订阅配额信息
type QuotaTelemetry struct {
	Upload   int64 `json:"upload"`
	Download int64 `json:"download"`
	Total    int64 `json:"total"`
	Expire   int64 `json:"expire"`
}

// SyncExecutionReport 记录一次订阅同步的完整审计快照
type SyncExecutionReport struct {
	Outcome       SyncOutcome    `json:"outcome"`
	PrevNodeCount int            `json:"prev_node_count"`
	NewNodeCount  int            `json:"new_node_count"`
	NodesDigest   string         `json:"nodes_digest"`
	Quota         QuotaTelemetry `json:"quota"`
	CheckedAt     time.Time      `json:"checked_at"`
	Message       string         `json:"message"`
}

// AutoSyncGuard 管控订阅的指纹比对、跌落熔断与按需重载
type AutoSyncGuard struct {
	mu             sync.Mutex
	lastDigest     string
	lastValidCount int
	minRetainRatio float64 // 默认 0.30（新节点数低于原节点数的 30% 且原节点数>=10时触发熔断保护）
}

// NewAutoSyncGuard 创建订阅防抖与熔断守卫
func NewAutoSyncGuard() *AutoSyncGuard {
	return &AutoSyncGuard{
		minRetainRatio: 0.30,
	}
}

// SetInitialState 初始化现有已加载节点的指纹与数量
func (g *AutoSyncGuard) SetInitialState(nodes []schema.ProxyNode) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.lastValidCount = len(nodes)
	g.lastDigest = computeValidNodesDigest(nodes)
}

// EvaluateAndSync 对净化后的节点列表执行熔断预检与增量指纹比对
func (g *AutoSyncGuard) EvaluateAndSync(
	ctx context.Context,
	validNodes []schema.ProxyNode,
	quota QuotaTelemetry,
	forceReload bool,
	reloadEngineFn func(ctx context.Context, nodes []schema.ProxyNode) error,
) (*SyncExecutionReport, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	newCount := len(validNodes)
	prevCount := g.lastValidCount

	// [防坑门禁 1：零节点硬熔断] 严禁将空节点池写入活跃配置导致全屋断网
	if newCount == 0 {
		return &SyncExecutionReport{
			Outcome:       OutcomeCircuitBroken,
			PrevNodeCount: prevCount,
			NewNodeCount:  0,
			Quota:         quota,
			CheckedAt:     time.Now(),
			Message:       "远端订阅未返回任何有效节点（已触发熔断保护，继续使用本地缓存节点）",
		}, nil
	}

	// [防坑门禁 2：节点暴跌熔断] 若原先节点 >= 10 个，而本次拉取暴跌超过 70%（且非手动强制刷新），触发熔断
	if !forceReload && prevCount >= 10 && float64(newCount) < float64(prevCount)*g.minRetainRatio {
		return &SyncExecutionReport{
			Outcome:       OutcomeCircuitBroken,
			PrevNodeCount: prevCount,
			NewNodeCount:  newCount,
			Quota:         quota,
			CheckedAt:     time.Now(),
			Message: fmt.Sprintf("订阅节点数异常暴跌 (%d -> %d)，疑似上游接口故障，已自动拦截并保留原配置",
				prevCount, newCount),
		}, nil
	}

	// [防坑门禁 3：结构化指纹增量比对] 若节点列表分毫不差，仅更新流量提示，跳过 IR 编译与内核重启
	newDigest := computeValidNodesDigest(validNodes)
	if !forceReload && g.lastDigest != "" && g.lastDigest == newDigest {
		return &SyncExecutionReport{
			Outcome:       OutcomeSkippedUnchanged,
			PrevNodeCount: prevCount,
			NewNodeCount:  newCount,
			NodesDigest:   newDigest,
			Quota:         quota,
			CheckedAt:     time.Now(),
			Message:       "节点配置未发生变更，已静默刷新流量配额（0 内核重启、0 连接打断）",
		}, nil
	}

	// 节点发生实质变更，触发 IR 编译与双擎重载
	if reloadEngineFn != nil {
		if err := reloadEngineFn(ctx, validNodes); err != nil {
			return nil, fmt.Errorf("apply updated subscription nodes failed: %w", err)
		}
	}

	g.lastDigest = newDigest
	g.lastValidCount = newCount

	return &SyncExecutionReport{
		Outcome:       OutcomeAppliedAndReload,
		PrevNodeCount: prevCount,
		NewNodeCount:  newCount,
		NodesDigest:   newDigest,
		Quota:         quota,
		CheckedAt:     time.Now(),
		Message:       fmt.Sprintf("订阅同步成功，已平滑加载 %d 个有效节点", newCount),
	}, nil
}

func computeValidNodesDigest(nodes []schema.ProxyNode) string {
	if len(nodes) == 0 {
		return ""
	}
	raw, err := json.Marshal(nodes)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:16])
}
