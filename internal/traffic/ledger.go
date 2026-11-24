package traffic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

// MonthlyRecord 持久化存储在 data/traffic_monthly.json 中的当月流量数据
type MonthlyRecord struct {
	YearMonth string `json:"year_month"` // 如 "2026-11"
	Uplink    uint64 `json:"uplink"`     // 经过代理出站的上行字节数
	Downlink  uint64 `json:"downlink"`   // 经过代理出站的下行字节数
	Total     uint64 `json:"total"`      // 经过代理出站的总字节数
	LastReset int64  `json:"last_reset"` // 上次月结重置时间戳 (Unix)
	UpdatedAt int64  `json:"updated_at"` // 上次刷盘更新时间戳 (Unix)
}

// MonthlySummary 对外 HTTP 响应结构
type MonthlySummary struct {
	YearMonth     string `json:"year_month"`
	Uplink        uint64 `json:"uplink"`
	Downlink      uint64 `json:"downlink"`
	Total         uint64 `json:"total"`
	UplinkHuman   string `json:"uplink_human"`
	DownlinkHuman string `json:"downlink_human"`
	TotalHuman    string `json:"total_human"`
	UpdatedAt     int64  `json:"updated_at"`
}

// ConnectionMeta 简化版的连接元信息，便于从 Clash API /connections 解码与提取
type ConnectionMeta struct {
	ID       string   `json:"id"`
	Upload   int64    `json:"upload"`
	Download int64    `json:"download"`
	Chains   []string `json:"chains"`
	Rule     string   `json:"rule"`
}

type connDelta struct {
	lastUp   int64
	lastDown int64
}

// MonthlyLedger 纯 Go 轻量、极简、无 CGO 依赖的当月代理出站累计流量账本
type MonthlyLedger struct {
	workDir string

	uplink   atomic.Uint64
	downlink atomic.Uint64

	monthMu   sync.RWMutex
	currMonth string
	lastReset int64

	connMu    sync.Mutex
	connBytes map[string]connDelta

	stopCh chan struct{}
}

// NewMonthlyLedger 创建并初始化当月流量账本
func NewMonthlyLedger(workDir string) *MonthlyLedger {
	now := time.Now()
	currM := now.Format("2006-01")

	l := &MonthlyLedger{
		workDir:   workDir,
		currMonth: currM,
		lastReset: now.Unix(),
		connBytes: make(map[string]connDelta, 256),
		stopCh:    make(chan struct{}),
	}

	l.loadFromDisk()

	// 启动后台 5 分钟低频落盘与跨月检查协程
	go l.backgroundWorker()

	return l
}

// loadFromDisk 从 data/traffic_monthly.json 加载持久化数据
func (l *MonthlyLedger) loadFromDisk() {
	if l.workDir == "" {
		return
	}
	filePath := filepath.Join(l.workDir, "data", "traffic_monthly.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return
	}

	var rec MonthlyRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return
	}

	nowMonth := time.Now().Format("2006-01")
	l.monthMu.Lock()
	defer l.monthMu.Unlock()

	if rec.YearMonth == nowMonth {
		// 同一个月，无缝继承历史累计
		l.currMonth = rec.YearMonth
		l.lastReset = rec.LastReset
		l.uplink.Store(rec.Uplink)
		l.downlink.Store(rec.Downlink)
	} else {
		// 跨月，自动归档旧月并以当前月份重置
		l.currMonth = nowMonth
		l.lastReset = time.Now().Unix()
		l.uplink.Store(0)
		l.downlink.Store(0)
		// 立即将新月份初始状态写盘
		_ = l.saveToDiskLocked()
	}
}

// isProxyChain 严格检查连接出站链，剥离 DIRECT / REJECT 纯直连或拦截
func isProxyChain(chains []string) bool {
	if len(chains) == 0 {
		return false
	}
	// 遍历 chains 列表
	for _, c := range chains {
		clean := strings.ToUpper(strings.TrimSpace(c))
		if clean == "DIRECT" || clean == "REJECT" || clean == "GLOBAL" || clean == "COMPATIBLE" {
			continue
		}
		// 如果包含有效的代理节点或分流代理链
		return true
	}
	return false
}

// ProcessConnections 从内核 /connections 获取的连接列表中，精确差值计量真实代理流量
func (l *MonthlyLedger) ProcessConnections(conns []ConnectionMeta) {
	if len(conns) == 0 {
		return
	}

	// 跨月平滑重置检查
	nowMonth := time.Now().Format("2006-01")
	l.monthMu.RLock()
	curM := l.currMonth
	l.monthMu.RUnlock()

	if curM != nowMonth {
		l.monthMu.Lock()
		if l.currMonth != nowMonth {
			l.currMonth = nowMonth
			l.lastReset = time.Now().Unix()
			l.uplink.Store(0)
			l.downlink.Store(0)
		}
		l.monthMu.Unlock()
	}

	var batchUp uint64
	var batchDown uint64

	l.connMu.Lock()
	activeSet := make(map[string]struct{}, len(conns))

	for _, c := range conns {
		if c.ID == "" {
			continue
		}
		activeSet[c.ID] = struct{}{}

		// 剥离 DIRECT 和 REJECT 出站
		if !isProxyChain(c.Chains) {
			continue
		}

		prev, exists := l.connBytes[c.ID]
		if !exists {
			// 新连接：首次计入当前的 upload 与 download
			if c.Upload > 0 {
				batchUp += uint64(c.Upload)
			}
			if c.Download > 0 {
				batchDown += uint64(c.Download)
			}
			l.connBytes[c.ID] = connDelta{lastUp: c.Upload, lastDown: c.Download}
		} else {
			// 活跃连接：计算递增 delta
			if c.Upload > prev.lastUp {
				batchUp += uint64(c.Upload - prev.lastUp)
			}
			if c.Download > prev.lastDown {
				batchDown += uint64(c.Download - prev.lastDown)
			}
			l.connBytes[c.ID] = connDelta{lastUp: c.Upload, lastDown: c.Download}
		}
	}

	// 清理已关闭连接，防止活跃映射泄漏
	for id := range l.connBytes {
		if _, stillAlive := activeSet[id]; !stillAlive {
			delete(l.connBytes, id)
		}
	}
	l.connMu.Unlock()

	// 原子无锁累加当月上行与下行流量
	if batchUp > 0 {
		l.uplink.Add(batchUp)
	}
	if batchDown > 0 {
		l.downlink.Add(batchDown)
	}
}

// GetMonthlySummary 获取格式化后的当月代理出站流量概览
func (l *MonthlyLedger) GetMonthlySummary() MonthlySummary {
	l.monthMu.RLock()
	m := l.currMonth
	l.monthMu.RUnlock()

	up := l.uplink.Load()
	down := l.downlink.Load()
	total := up + down

	return MonthlySummary{
		YearMonth:     m,
		Uplink:        up,
		Downlink:      down,
		Total:         total,
		UplinkHuman:   formatBytesHuman(up),
		DownlinkHuman: formatBytesHuman(down),
		TotalHuman:    formatBytesHuman(total),
		UpdatedAt:     time.Now().Unix(),
	}
}

// Flush 强制落盘（服务优雅退出或切换引擎时调用）
func (l *MonthlyLedger) Flush() error {
	l.monthMu.Lock()
	defer l.monthMu.Unlock()
	return l.saveToDiskLocked()
}

// saveToDiskLocked 内部保存方法（调用方持有 monthMu）
func (l *MonthlyLedger) saveToDiskLocked() error {
	if l.workDir == "" {
		return nil
	}
	up := l.uplink.Load()
	down := l.downlink.Load()

	rec := MonthlyRecord{
		YearMonth: l.currMonth,
		Uplink:    up,
		Downlink:  down,
		Total:     up + down,
		LastReset: l.lastReset,
		UpdatedAt: time.Now().Unix(),
	}

	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Join(l.workDir, "data")
	_ = os.MkdirAll(dir, 0700)
	return schema.WriteFileAtomicSecure(filepath.Join(dir, "traffic_monthly.json"), data)
}

// backgroundWorker 后台 5 分钟定时落盘及跨月巡检
func (l *MonthlyLedger) backgroundWorker() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = l.Flush()
		case <-l.stopCh:
			_ = l.Flush()
			return
		}
	}
}

// Close 停止后台定时器并执行最终落盘
func (l *MonthlyLedger) Close() {
	select {
	case <-l.stopCh:
	default:
		close(l.stopCh)
	}
}

func formatBytesHuman(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
