package affinity

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SessionAffinityStatus 会话粘滞与防风控调度状态
type SessionAffinityStatus struct {
	Enabled             bool              `json:"enabled"`
	Strategy            string            `json:"strategy"` // consistent-hash, src-ip-dest, sticky-ttl
	SessionTTLSeconds   int               `json:"session_ttl_seconds"`
	ActiveBindingsCount int               `json:"active_bindings_count"`
	TotalRouted         uint64            `json:"total_routed"`
	FallbackMigrations  uint64            `json:"fallback_migrations"`
	LastMigrationAt     string            `json:"last_migration_at,omitempty"`
	SampleBindings      map[string]string `json:"sample_bindings,omitempty"` // host -> proxy
	UpdatedAt           string            `json:"updated_at"`
}

type bindingEntry struct {
	ProxyName string    `json:"proxy_name"`
	ExpiresAt time.Time `json:"expires_at"`
	HitCount  uint64    `json:"hit_count"`
}

// SessionAffinityManager 会话粘滞调度管理器
type SessionAffinityManager struct {
	mu                 sync.RWMutex
	enabled            bool
	strategy           string
	sessionTTL         time.Duration
	totalRouted        uint64
	fallbackMigrations uint64
	lastMigrationAt    time.Time
	bindings           map[string]*bindingEntry // key: "clientIP|destHost"
	persistPath        string
}

// NewSessionAffinityManager 创建管理器
func NewSessionAffinityManager(persistPath string) *SessionAffinityManager {
	if persistPath == "" {
		persistPath = filepath.Join("data", "session_affinity.json")
	}
	mgr := &SessionAffinityManager{
		enabled:    true,
		strategy:   "consistent-hash",
		sessionTTL: 10 * time.Minute,
		bindings:   make(map[string]*bindingEntry),
		persistPath: persistPath,
	}
	mgr.loadPersisted()
	return mgr
}

func (m *SessionAffinityManager) loadPersisted() {
	if data, err := os.ReadFile(m.persistPath); err == nil {
		var s struct {
			Enabled            bool   `json:"enabled"`
			Strategy           string `json:"strategy"`
			SessionTTLSeconds  int    `json:"session_ttl_seconds"`
			TotalRouted        uint64 `json:"total_routed"`
			FallbackMigrations uint64 `json:"fallback_migrations"`
		}
		if json.Unmarshal(data, &s) == nil {
			m.enabled = s.Enabled
			if s.Strategy != "" {
				m.strategy = s.Strategy
			}
			if s.SessionTTLSeconds > 0 {
				m.sessionTTL = time.Duration(s.SessionTTLSeconds) * time.Second
			}
			m.totalRouted = s.TotalRouted
			m.fallbackMigrations = s.FallbackMigrations
		}
	}
}

func (m *SessionAffinityManager) persist() {
	_ = os.MkdirAll(filepath.Dir(m.persistPath), 0755)
	s := struct {
		Enabled            bool   `json:"enabled"`
		Strategy           string `json:"strategy"`
		SessionTTLSeconds  int    `json:"session_ttl_seconds"`
		TotalRouted        uint64 `json:"total_routed"`
		FallbackMigrations uint64 `json:"fallback_migrations"`
	}{
		Enabled:            m.enabled,
		Strategy:           m.strategy,
		SessionTTLSeconds:  int(m.sessionTTL.Seconds()),
		TotalRouted:        m.totalRouted,
		FallbackMigrations: m.fallbackMigrations,
	}
	if b, err := json.MarshalIndent(s, "", "  "); err == nil {
		tmp := m.persistPath + ".tmp"
		if os.WriteFile(tmp, b, 0644) == nil {
			_ = os.Rename(tmp, m.persistPath)
		}
	}
}

// ConsistentHash 计算一致性哈希，将 key 映射到 proxies 列表中的确定性节点
func ConsistentHash(key string, proxies []string) string {
	if len(proxies) == 0 {
		return ""
	}
	if len(proxies) == 1 {
		return proxies[0]
	}
	h := md5.Sum([]byte(key))
	val := binary.BigEndian.Uint64(h[:8])
	idx := int(val % uint64(len(proxies)))
	return proxies[idx]
}

// ResolveProxy 为客户端及目标地址确定粘滞出口节点，确保外网出口 IP 不漂移
func (m *SessionAffinityManager) ResolveProxy(clientIP, destHost string, availableProxies []string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(availableProxies) == 0 {
		return ""
	}
	m.totalRouted++

	if !m.enabled {
		return availableProxies[0]
	}

	key := fmt.Sprintf("%s|%s", clientIP, destHost)
	now := time.Now()

	// 1. 查找是否存在尚未过期的已有粘滞绑定
	if entry, exists := m.bindings[key]; exists {
		if now.Before(entry.ExpiresAt) {
			// 验证该 proxy 是否仍在可用列表中
			for _, p := range availableProxies {
				if p == entry.ProxyName {
					entry.HitCount++
					entry.ExpiresAt = now.Add(m.sessionTTL)
					return entry.ProxyName
				}
			}
			// 原节点已下线或不可用，执行毫秒级平滑原子漂移
			m.fallbackMigrations++
			m.lastMigrationAt = now
		}
	}

	// 2. 使用一致性哈希算法选定确定性节点
	chosen := ConsistentHash(key, availableProxies)
	m.bindings[key] = &bindingEntry{
		ProxyName: chosen,
		ExpiresAt: now.Add(m.sessionTTL),
		HitCount:  1,
	}

	// 限制内存绑定条目大小，防止膨胀
	if len(m.bindings) > 5000 {
		m.cleanExpiredLocked(now)
	}

	return chosen
}

func (m *SessionAffinityManager) cleanExpiredLocked(now time.Time) {
	for k, v := range m.bindings {
		if now.After(v.ExpiresAt) {
			delete(m.bindings, k)
		}
	}
}

// TriggerNodeFailure 模拟或处理承载节点失效，将绑定到该节点的会话平滑重选
func (m *SessionAffinityManager) TriggerNodeFailure(badProxy string, healthyProxies []string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(healthyProxies) == 0 {
		return 0
	}
	count := 0
	now := time.Now()
	for k, v := range m.bindings {
		if v.ProxyName == badProxy {
			newProxy := ConsistentHash(k, healthyProxies)
			v.ProxyName = newProxy
			v.ExpiresAt = now.Add(m.sessionTTL)
			count++
		}
	}
	m.fallbackMigrations += uint64(count)
	m.lastMigrationAt = now
	m.persist()
	return count
}

// ResetBindings 清空全部会话绑定
func (m *SessionAffinityManager) ResetBindings() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bindings = make(map[string]*bindingEntry)
	m.persist()
}

// SetEnabled 设置启用/停用
func (m *SessionAffinityManager) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
	m.persist()
}

// SetStrategy 设置算法策略
func (m *SessionAffinityManager) SetStrategy(strategy string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.strategy = strategy
	m.persist()
}

// GetStatus 获取遥测状态
func (m *SessionAffinityManager) GetStatus() SessionAffinityStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	samples := make(map[string]string)
	activeCount := 0
	for k, v := range m.bindings {
		if now.Before(v.ExpiresAt) {
			activeCount++
			if len(samples) < 8 {
				samples[k] = v.ProxyName
			}
		}
	}

	lastMig := ""
	if !m.lastMigrationAt.IsZero() {
		lastMig = m.lastMigrationAt.Format("2006-01-02 15:04:05")
	}

	return SessionAffinityStatus{
		Enabled:             m.enabled,
		Strategy:            m.strategy,
		SessionTTLSeconds:   int(m.sessionTTL.Seconds()),
		ActiveBindingsCount: activeCount,
		TotalRouted:         m.totalRouted,
		FallbackMigrations:  m.fallbackMigrations,
		LastMigrationAt:     lastMig,
		SampleBindings:      samples,
		UpdatedAt:           now.Format("2006-01-02 15:04:05"),
	}
}
