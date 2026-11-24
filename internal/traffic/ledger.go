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
	Metadata struct {
		Network         string `json:"network"`
		Type            string `json:"type"`
		SourceIP        string `json:"sourceIP"`
		DestinationIP   string `json:"destinationIP"`
		SourcePort      string `json:"sourcePort"`
		DestinationPort string `json:"destinationPort"`
		Host            string `json:"host"`
	} `json:"metadata"`
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

	timeNow func() time.Time
	stopCh  chan struct{}
}

func (l *MonthlyLedger) now() time.Time {
	if l.timeNow != nil {
		return l.timeNow()
	}
	return time.Now()
}

// NewMonthlyLedger 创建并初始化当月流量账本
func NewMonthlyLedger(workDir string) *MonthlyLedger {
	l := &MonthlyLedger{
		workDir:   workDir,
		timeNow:   time.Now,
		connBytes: make(map[string]connDelta, 256),
		stopCh:    make(chan struct{}),
	}

	n := l.now()
	l.currMonth = fmt.Sprintf("%04d-%02d", n.Year(), int(n.Month()))
	l.lastReset = n.Unix()

	l.loadFromDisk()

	// 启动后台 1 分钟低频跨月巡检与 5 分钟定时刷盘协程
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

	now := l.now()
	l.monthMu.Lock()
	defer l.monthMu.Unlock()

	l.currMonth = rec.YearMonth
	l.lastReset = rec.LastReset
	l.uplink.Store(rec.Uplink)
	l.downlink.Store(rec.Downlink)

	// 动态检测是否停机跨月（跨入每月 1 日 00:00:00），若跨月自动清零并归档
	l.checkMonthRolloverLocked(now)
}

// checkMonthRolloverLocked 核心自然月轮转检查 (调用方须持有 monthMu 锁)
// 依据当前系统时区与自然公历 (每月 1 日 00:00:00 起)，动态感知月份变更并无损自愈重置，零硬编码
func (l *MonthlyLedger) checkMonthRolloverLocked(now time.Time) bool {
	// 动态提取当前自然年月 (如 2026-11 -> 2026-12 -> 2027-01)，每月 1 日 00:00:00 准时变更
	currentMonthStr := fmt.Sprintf("%04d-%02d", now.Year(), int(now.Month()))
	if l.currMonth == currentMonthStr {
		return false
	}

	// 发生自然月轮转 (每月 1 日 00:00:00 起新账期)
	oldMonth := l.currMonth
	oldUp := l.uplink.Load()
	oldDown := l.downlink.Load()

	// 1. 自动归档旧月历史记录至 data/traffic_history_YYYY-MM.json，保障账目可追溯
	if l.workDir != "" && oldMonth != "" && (oldUp > 0 || oldDown > 0) {
		historyRec := MonthlyRecord{
			YearMonth: oldMonth,
			Uplink:    oldUp,
			Downlink:  oldDown,
			Total:     oldUp + oldDown,
			LastReset: l.lastReset,
			UpdatedAt: now.Unix(),
		}
		if histData, err := json.MarshalIndent(historyRec, "", "  "); err == nil {
			histFile := filepath.Join(l.workDir, "data", fmt.Sprintf("traffic_history_%s.json", oldMonth))
			_ = schema.WriteFileAtomicSecure(histFile, histData)
		}
	}

	// 2. 原生重置当月账本计数器为 0
	l.currMonth = currentMonthStr
	l.lastReset = now.Unix()
	l.uplink.Store(0)
	l.downlink.Store(0)

	// 3. 重置连接差值映射，防止老月份连接的差值跨月误加
	l.connMu.Lock()
	l.connBytes = make(map[string]connDelta, 256)
	l.connMu.Unlock()

	// 4. 立即将新月份初态 (0 流量) 原子持久化到 traffic_monthly.json
	_ = l.saveToDiskLocked()
	return true
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

	// 跨月平滑重置检查 (每月 1 日 00:00:00 自动清零)
	now := l.now()
	l.monthMu.Lock()
	l.checkMonthRolloverLocked(now)
	l.monthMu.Unlock()

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
	now := l.now()
	l.monthMu.Lock()
	l.checkMonthRolloverLocked(now)
	m := l.currMonth
	up := l.uplink.Load()
	down := l.downlink.Load()
	l.monthMu.Unlock()

	total := up + down

	return MonthlySummary{
		YearMonth:     m,
		Uplink:        up,
		Downlink:      down,
		Total:         total,
		UplinkHuman:   formatBytesHuman(up),
		DownlinkHuman: formatBytesHuman(down),
		TotalHuman:    formatBytesHuman(total),
		UpdatedAt:     l.now().Unix(),
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
		UpdatedAt: l.now().Unix(),
	}

	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Join(l.workDir, "data")
	_ = os.MkdirAll(dir, 0700)
	return schema.WriteFileAtomicSecure(filepath.Join(dir, "traffic_monthly.json"), data)
}

// backgroundWorker 后台 1 分钟跨月巡检及 5 分钟定时刷盘
func (l *MonthlyLedger) backgroundWorker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	flushTicks := 0

	for {
		select {
		case <-ticker.C:
			now := l.now()
			l.monthMu.Lock()
			rolled := l.checkMonthRolloverLocked(now)
			flushTicks++
			// 若发生每月 1 日跨月重置，或每 5 分钟 (5 个 tick) 周期到达，执行刷盘
			if rolled || flushTicks >= 5 {
				_ = l.saveToDiskLocked()
				flushTicks = 0
			}
			l.monthMu.Unlock()
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
