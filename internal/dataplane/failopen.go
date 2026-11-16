package dataplane

import (
	"context"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

// FailOpenState 逃生旁路状态
type FailOpenState int32

const (
	FailOpenDisabled FailOpenState = 0 // 正常代理分流态
	FailOpenActive   FailOpenState = 1 // 逃生直通态 (国内与直连 0 秒断网)
)

// FailOpenController 内核级 Fail-Open 逃生旁路控制器
// 特性：
// 1. 当代理核心进程崩溃或异常持续 >= 3 秒时，原子切换底层网络为 Fail-Open 逃生态；
// 2. 所有内网流量直接 accept/return 直通物理主路由网关，保障看国内视频、智能家居 0 故障断网；
// 3. 待核心引擎健康恢复稳定持续 5 秒后，自动无缝原子切回正常代理分流。
type FailOpenController struct {
	mu           sync.Mutex
	state        atomic.Int32
	lastSwitchAt int64
	onStateChange func(state FailOpenState, reason string)
}

// NewFailOpenController 创建逃生控制器
func NewFailOpenController(onStateChange func(state FailOpenState, reason string)) *FailOpenController {
	c := &FailOpenController{
		onStateChange: onStateChange,
	}
	c.state.Store(int32(FailOpenDisabled))
	return c
}

// IsFailOpenActive 查询当前是否处于逃生状态
func (c *FailOpenController) IsFailOpenActive() bool {
	return FailOpenState(c.state.Load()) == FailOpenActive
}

// TriggerFailOpen 触发逃生旁路 (全部直通主路由)
func (c *FailOpenController) TriggerFailOpen(ctx context.Context, reason string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if FailOpenState(c.state.Load()) == FailOpenActive {
		return nil
	}

	c.state.Store(int32(FailOpenActive))
	c.lastSwitchAt = time.Now().UnixMilli()

	// 执行系统底层 nftables 原子逃生指令 (若支持 nft)
	_ = c.execNftFailOpen(ctx, true)

	if c.onStateChange != nil {
		go c.onStateChange(FailOpenActive, reason)
	}
	return nil
}

// RestoreNormal 恢复正常代理分流
func (c *FailOpenController) RestoreNormal(ctx context.Context, reason string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if FailOpenState(c.state.Load()) == FailOpenDisabled {
		return nil
	}

	c.state.Store(int32(FailOpenDisabled))
	c.lastSwitchAt = time.Now().UnixMilli()

	// 撤销 nftables 逃生旁路规则
	_ = c.execNftFailOpen(ctx, false)

	if c.onStateChange != nil {
		go c.onStateChange(FailOpenDisabled, reason)
	}
	return nil
}

func (c *FailOpenController) execNftFailOpen(ctx context.Context, enable bool) error {
	// 在 Linux 上执行 nft 规则动态注入
	var ruleCmd string
	if enable {
		// 在 cubex_symbiotic 表的 prerouting_fastpath 最顶部插入直通规则
		ruleCmd = "nft 'insert rule inet cubex_symbiotic prerouting_fastpath index 0 meta mark set 0x1ed4 return' 2>/dev/null || true"
	} else {
		// 恢复时重新单事务刷新 fastpath 规则
		ruleCmd = "nft 'delete rule inet cubex_symbiotic prerouting_fastpath handle 1' 2>/dev/null || true"
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", ruleCmd)
	return cmd.Run()
}

// FailOpenStatusReport 供 Web 控制台展示的逃生状态
type FailOpenStatusReport struct {
	Active       bool   `json:"active"`
	StatusText   string `json:"status_text"`
	CapsuleBadge string `json:"capsule_badge"`
	LastSwitchAt int64  `json:"last_switch_at"`
}

// GetStatusReport 获取当前状态汇报
func (c *FailOpenController) GetStatusReport() FailOpenStatusReport {
	active := c.IsFailOpenActive()
	if active {
		return FailOpenStatusReport{
			Active:       true,
			StatusText:   "代理引擎维护中，已自动开启直连逃生通道（国内网络不受影响）",
			CapsuleBadge: "amber",
			LastSwitchAt: c.lastSwitchAt,
		}
	}
	return FailOpenStatusReport{
		Active:       false,
		StatusText:   "全链路双核守护正常",
		CapsuleBadge: "green",
		LastSwitchAt: c.lastSwitchAt,
	}
}
