package host

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MultiWANPolicyMode 多宽带分流策略模式
type MultiWANPolicyMode string

const (
	PolicyLatency   MultiWANPolicyMode = "latency"   // 最低时延优先 (跨境协同业务)
	PolicyBandwidth MultiWANPolicyMode = "bandwidth" // 加权负载聚合 (大文件与视频)
	PolicyFailover  MultiWANPolicyMode = "failover"  // 毫秒级主备容灾 (主线断开无感切 5G CPE)
)

// WANPathMetrics 单条 WAN 物理出口链路实时状态
type WANPathMetrics struct {
	Interface string  `json:"interface"`
	Gateway   string  `json:"gateway"`
	Carrier   string  `json:"carrier"` // 电信, 联通, 移动, 5G CPE
	IsActive  bool    `json:"is_active"`
	LatencyMs float64 `json:"latency_ms"`
	JitterMs  float64 `json:"jitter_ms"`
	LossRate  float64 `json:"loss_rate"`
	Weight    int     `json:"weight"`
}

// MultiWANBindingStatus 多出口绑定与容灾状态汇总
type MultiWANBindingStatus struct {
	ActiveMode     MultiWANPolicyMode `json:"active_mode"`
	PrimaryWAN     string             `json:"primary_wan"`
	BackupWAN      string             `json:"backup_wan"`
	Paths          []WANPathMetrics   `json:"paths"`
	FailoverCount  int64              `json:"failover_count"`
	LastFailoverAt time.Time          `json:"last_failover_at"`
	Summary        string             `json:"summary"`
}

// MultiWANBindingManager 多出口智能绑定与容灾中枢管理器
type MultiWANBindingManager struct {
	mu             sync.RWMutex
	filePath       string
	activeMode     MultiWANPolicyMode
	primaryWAN     string
	backupWAN      string
	paths          map[string]*WANPathMetrics
	failoverCount  int64
	lastFailoverAt time.Time
}

var (
	bindingOnce sync.Once
	bindingInst *MultiWANBindingManager
)

// GetMultiWANBindingManager 单例获取绑定管理器
func GetMultiWANBindingManager(dataDir string) *MultiWANBindingManager {
	bindingOnce.Do(func() {
		bindingInst = newMultiWANBindingManager(dataDir)
	})
	return bindingInst
}

func newMultiWANBindingManager(dataDir string) *MultiWANBindingManager {
	filePath := filepath.Join(dataDir, "multiwan_binding.json")

	defaultPaths := map[string]*WANPathMetrics{
		"pppoe-out1": {
			Interface: "pppoe-out1",
			Gateway:   "100.64.123.1",
			Carrier:   "主线 · 极速光纤",
			IsActive:  true,
			LatencyMs: 9.8,
			JitterMs:  1.2,
			LossRate:  0.0,
			Weight:    70,
		},
		"wan2": {
			Interface: "wan2",
			Gateway:   "192.168.8.1",
			Carrier:   "副线 · 5G CPE 热备",
			IsActive:  true,
			LatencyMs: 22.4,
			JitterMs:  3.5,
			LossRate:  0.0,
			Weight:    30,
		},
	}

	mgr := &MultiWANBindingManager{
		filePath:       filePath,
		activeMode:     PolicyLatency,
		primaryWAN:     "pppoe-out1",
		backupWAN:      "wan2",
		paths:          defaultPaths,
		lastFailoverAt: time.Now(),
	}

	mgr.load()
	return mgr
}

// GetStatus 获取当前状态快照
func (m *MultiWANBindingManager) GetStatus() MultiWANBindingStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pathsList := make([]WANPathMetrics, 0, len(m.paths))
	for _, p := range m.paths {
		pathsList = append(pathsList, *p)
	}

	summary := "双宽带对等协同运行中，毫秒级主备热备已就绪"
	if m.activeMode == PolicyBandwidth {
		summary = "大带宽加权聚合模式已启用，跨物理接口并发加速中"
	} else if m.activeMode == PolicyLatency {
		summary = "最低时延优先模式已启用，自动锁定最优出站路径"
	}

	return MultiWANBindingStatus{
		ActiveMode:     m.activeMode,
		PrimaryWAN:     m.primaryWAN,
		BackupWAN:      m.backupWAN,
		Paths:          pathsList,
		FailoverCount:  m.failoverCount,
		LastFailoverAt: m.lastFailoverAt,
		Summary:        summary,
	}
}

// SetPolicyMode 设置分流策略模式
func (m *MultiWANBindingManager) SetPolicyMode(mode MultiWANPolicyMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if mode != PolicyLatency && mode != PolicyBandwidth && mode != PolicyFailover {
		return fmt.Errorf("invalid policy mode: %s", mode)
	}

	m.activeMode = mode
	m.saveLocked()
	return nil
}

// RecordPathProbe 记录接口的探测数据
func (m *MultiWANBindingManager) RecordPathProbe(iface string, latency, jitter, loss float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p, ok := m.paths[iface]; ok {
		p.LatencyMs = latency
		p.JitterMs = jitter
		p.LossRate = loss
		p.IsActive = loss < 100.0
	}
}

// TriggerFailover 模拟或执行主备切换
func (m *MultiWANBindingManager) TriggerFailover(fromIface, toIface string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.primaryWAN = toIface
	m.backupWAN = fromIface
	m.failoverCount++
	m.lastFailoverAt = time.Now()
	m.saveLocked()
}

type persistedBinding struct {
	ActiveMode    MultiWANPolicyMode `json:"active_mode"`
	PrimaryWAN    string             `json:"primary_wan"`
	BackupWAN     string             `json:"backup_wan"`
	FailoverCount int64              `json:"failover_count"`
}

func (m *MultiWANBindingManager) load() {
	if m.filePath == "" {
		return
	}
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return
	}
	var p persistedBinding
	if err := json.Unmarshal(data, &p); err == nil && p.ActiveMode != "" {
		m.activeMode = p.ActiveMode
		if p.PrimaryWAN != "" {
			m.primaryWAN = p.PrimaryWAN
		}
		if p.BackupWAN != "" {
			m.backupWAN = p.BackupWAN
		}
		m.failoverCount = p.FailoverCount
	}
}

func (m *MultiWANBindingManager) saveLocked() {
	if m.filePath == "" {
		return
	}
	dir := filepath.Dir(m.filePath)
	_ = os.MkdirAll(dir, 0755)

	p := persistedBinding{
		ActiveMode:    m.activeMode,
		PrimaryWAN:    m.primaryWAN,
		BackupWAN:     m.backupWAN,
		FailoverCount: m.failoverCount,
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return
	}

	tmp := m.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err == nil {
		_ = os.Rename(tmp, m.filePath)
	}
}
