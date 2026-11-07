package netstack

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// QUICSessionRecord 记录活跃的 QUIC 连接迁移状态
type QUICSessionRecord struct {
	CID            string    `json:"cid"`
	ClientIP       string    `json:"client_ip"`
	LastSeenPort   int       `json:"last_seen_port"`
	InitialIP      string    `json:"initial_ip"`
	MigrationCount int       `json:"migration_count"`
	LastMigratedAt time.Time `json:"last_migrated_at"`
	LastActiveAt   time.Time `json:"last_active_at"`
	RTTMs          float64   `json:"rtt_ms"`
}

// ZeroRTTSessionTicket 存储 TLS 1.3 / QUIC 0-RTT 会话票据
type ZeroRTTSessionTicket struct {
	Domain     string    `json:"domain"`
	TicketHash string    `json:"ticket_hash"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	HitCount   int64     `json:"hit_count"`
	Data       []byte    `json:"-"`
}

// QUICMigrationStatus 遥测状态汇总
type QUICMigrationStatus struct {
	Enabled             bool                `json:"enabled"`
	ZeroRTTEnabled      bool                `json:"zero_rtt_enabled"`
	ActiveSessions      int                 `json:"active_sessions"`
	TotalMigrations     int64               `json:"total_migrations"`
	ZeroRTTHits         int64               `json:"zero_rtt_hits"`
	FECRecoveredPackets int64               `json:"fec_recovered_packets"`
	RecentSessions      []QUICSessionRecord `json:"recent_sessions"`
}

// QUICMigrationManager 负责 RFC 9000 QUIC 连接迁移与 0-RTT 加速
type QUICMigrationManager struct {
	mu                  sync.RWMutex
	enabled             bool
	zeroRTTEnabled      bool
	sessions            map[string]*QUICSessionRecord
	zeroRTTStore        map[string]*ZeroRTTSessionTicket
	totalMigrations     int64
	zeroRTTHits         int64
	fecRecoveredPackets int64
}

// NewQUICMigrationManager 实例化连接迁移管理器
func NewQUICMigrationManager() *QUICMigrationManager {
	mgr := &QUICMigrationManager{
		enabled:        true,
		zeroRTTEnabled: true,
		sessions:       make(map[string]*QUICSessionRecord),
		zeroRTTStore:   make(map[string]*ZeroRTTSessionTicket),
	}
	go mgr.backgroundPruneLoop()
	return mgr
}

// RecordPacket 处理接收到的 QUIC 数据包并侦测连接迁移
func (m *QUICMigrationManager) RecordPacket(cid string, clientIP string, port int, rttMs float64) bool {
	if !m.enabled || cid == "" {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	rec, exists := m.sessions[cid]
	if !exists {
		m.sessions[cid] = &QUICSessionRecord{
			CID:            cid,
			ClientIP:       clientIP,
			LastSeenPort:   port,
			InitialIP:      clientIP,
			MigrationCount: 0,
			LastActiveAt:   now,
			RTTMs:          rttMs,
		}
		return false
	}

	rec.LastActiveAt = now
	rec.RTTMs = rttMs

	// 侦测到客户端 IP 变动（如不同 AP / Wi-Fi 漫游）
	if rec.ClientIP != clientIP || rec.LastSeenPort != port {
		rec.ClientIP = clientIP
		rec.LastSeenPort = port
		rec.MigrationCount++
		rec.LastMigratedAt = now
		m.totalMigrations++
		return true
	}

	return false
}

// CacheZeroRTTTicket 存储域名的 0-RTT 会话票据
func (m *QUICMigrationManager) CacheZeroRTTTicket(domain string, ticketData []byte, ttl time.Duration) {
	if !m.zeroRTTEnabled || domain == "" || len(ticketData) == 0 {
		return
	}

	h := sha256.Sum256(ticketData)
	hashStr := hex.EncodeToString(h[:8])

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	m.zeroRTTStore[domain] = &ZeroRTTSessionTicket{
		Domain:     domain,
		TicketHash: hashStr,
		CreatedAt:  now,
		ExpiresAt:  now.Add(ttl),
		HitCount:   0,
		Data:       ticketData,
	}
}

// GetZeroRTTTicket 获取域名的 0-RTT 会话票据以实现 0-RTT 极速握手
func (m *QUICMigrationManager) GetZeroRTTTicket(domain string) ([]byte, bool) {
	if !m.zeroRTTEnabled || domain == "" {
		return nil, false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	t, exists := m.zeroRTTStore[domain]
	if !exists {
		return nil, false
	}

	if time.Now().After(t.ExpiresAt) {
		delete(m.zeroRTTStore, domain)
		return nil, false
	}

	t.HitCount++
	m.zeroRTTHits++
	return t.Data, true
}

// RecordFECRecovery 记录一次前向纠错抗丢包成功修复
func (m *QUICMigrationManager) RecordFECRecovery(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fecRecoveredPackets += count
}

// GetStatus 获取遥测快照
func (m *QUICMigrationManager) GetStatus() QUICMigrationStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	recent := make([]QUICSessionRecord, 0, 10)
	for _, s := range m.sessions {
		recent = append(recent, *s)
		if len(recent) >= 10 {
			break
		}
	}

	return QUICMigrationStatus{
		Enabled:             m.enabled,
		ZeroRTTEnabled:      m.zeroRTTEnabled,
		ActiveSessions:      len(m.sessions),
		TotalMigrations:     m.totalMigrations,
		ZeroRTTHits:         m.zeroRTTHits,
		FECRecoveredPackets: m.fecRecoveredPackets,
		RecentSessions:      recent,
	}
}

// Toggle 设置开关状态
func (m *QUICMigrationManager) Toggle(enabled bool, zeroRTT bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
	m.zeroRTTEnabled = zeroRTT
}

// backgroundPruneLoop 周期清理僵尸会话与过期票据
func (m *QUICMigrationManager) backgroundPruneLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()

		for cid, s := range m.sessions {
			if now.Sub(s.LastActiveAt) > 10*time.Minute {
				delete(m.sessions, cid)
			}
		}

		for domain, t := range m.zeroRTTStore {
			if now.After(t.ExpiresAt) {
				delete(m.zeroRTTStore, domain)
			}
		}
		m.mu.Unlock()
	}
}
