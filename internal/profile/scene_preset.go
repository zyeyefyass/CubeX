package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SceneMode 场景模式枚举
type SceneMode string

const (
	SceneBalanced SceneMode = "balanced" // 经典均衡模式
	SceneGaming   SceneMode = "gaming"   // 极限电竞低延迟模式
	SceneMedia    SceneMode = "media"    // 4K/8K 极致影音模式
	SceneSecurity SceneMode = "security" // 全家安全静默模式
)

// SceneParameters 单场景参数集合
type SceneParameters struct {
	Name                 string  `json:"name"`
	Description          string  `json:"description"`
	Icon                 string  `json:"icon"`
	DSCPValue            int     `json:"dscp_value"`             // DSCP 报文标记 (例如 46 为 EF 极速加速)
	CoDelTargetMs        float64 `json:"codel_target_ms"`        // CoDel 拥塞控制目标延迟
	SocketBufferMB       int     `json:"socket_buffer_mb"`       // TCP 读写缓冲区软上限 (MB)
	StrictDNSLeakProof   bool    `json:"strict_dns_leak_proof"`  // 严格防 DNS 侧漏 (强制 SERVFAIL)
	AdBlockStrict        bool    `json:"ad_block_strict"`        // 217heidai 权威规则全开
	UDPDirectFastPath    bool    `json:"udp_direct_fast_path"`   // UDP 极速直连直出通道
	HappyEyeballsEnabled bool    `json:"happy_eyeballs_enabled"` // 双栈极速并发竞速
}

// SceneStatus 当前生效的场景模式快照
type SceneStatus struct {
	ActiveScene    SceneMode                  `json:"active_scene"`
	ActiveParams   SceneParameters            `json:"active_params"`
	AvailableModes map[SceneMode]SceneParameters `json:"available_modes"`
	SwitchedAt     time.Time                  `json:"switched_at"`
}

// ScenePresetManager 场景预设中枢管理器
type ScenePresetManager struct {
	mu          sync.RWMutex
	filePath    string
	activeScene SceneMode
	switchedAt  time.Time
	definitions map[SceneMode]SceneParameters
}

// NewScenePresetManager 实例化场景预设管理器
func NewScenePresetManager(dataDir string) *ScenePresetManager {
	filePath := filepath.Join(dataDir, "scene_preset.json")

	defs := map[SceneMode]SceneParameters{
		SceneBalanced: {
			Name:                 "经典均衡",
			Description:          "多设备日常办公、网页浏览与流媒体对等调度，功耗与延迟最优兼顾",
			Icon:                 "⚖️",
			DSCPValue:            0,
			CoDelTargetMs:        2.5,
			SocketBufferMB:       16,
			StrictDNSLeakProof:   true,
			AdBlockStrict:        true,
			UDPDirectFastPath:    false,
			HappyEyeballsEnabled: true,
		},
		SceneGaming: {
			Name:                 "极限电竞低延",
			Description:          "启用 DSCP EF 极速报文排队，UDP 快转直通，缓冲区延迟压缩至 1.0ms，彻底杜绝跳 ping",
			Icon:                 "🎮",
			DSCPValue:            46, // EF (Expedited Forwarding)
			CoDelTargetMs:        1.0,
			SocketBufferMB:       8,
			StrictDNSLeakProof:   true,
			AdBlockStrict:        false, // 电竞时减少规则比对时延
			UDPDirectFastPath:    true,
			HappyEyeballsEnabled: true,
		},
		SceneMedia: {
			Name:                 "4K/8K 极致影音",
			Description:          "TCP 32MB 满血读写缓冲区全开，Happy Eyeballs 双栈并发拉取极速边缘 CDN，首包 0 缓冲秒开",
			Icon:                 "🎬",
			DSCPValue:            0,
			CoDelTargetMs:        5.0,
			SocketBufferMB:       32,
			StrictDNSLeakProof:   true,
			AdBlockStrict:        true,
			UDPDirectFastPath:    false,
			HappyEyeballsEnabled: true,
		},
		SceneSecurity: {
			Name:                 "全家安全静默",
			Description:          "217heidai 深度广告拦截，未决解析绝对阻断防侧漏，陌生设备自动入沙盒隔离",
			Icon:                 "🛡️",
			DSCPValue:            0,
			CoDelTargetMs:        3.0,
			SocketBufferMB:       16,
			StrictDNSLeakProof:   true,
			AdBlockStrict:        true,
			UDPDirectFastPath:    false,
			HappyEyeballsEnabled: false,
		},
	}

	mgr := &ScenePresetManager{
		filePath:    filePath,
		activeScene: SceneBalanced,
		switchedAt:  time.Now(),
		definitions: defs,
	}

	mgr.load()
	return mgr
}

// GetStatus 获取当前场景快照
func (m *ScenePresetManager) GetStatus() SceneStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	params, exists := m.definitions[m.activeScene]
	if !exists {
		params = m.definitions[SceneBalanced]
	}

	return SceneStatus{
		ActiveScene:    m.activeScene,
		ActiveParams:   params,
		AvailableModes: m.definitions,
		SwitchedAt:     m.switchedAt,
	}
}

// SwitchScene 切换场景模式
func (m *ScenePresetManager) SwitchScene(mode SceneMode) (SceneParameters, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	params, exists := m.definitions[mode]
	if !exists {
		return SceneParameters{}, fmt.Errorf("unknown scene mode: %s", mode)
	}

	m.activeScene = mode
	m.switchedAt = time.Now()
	m.saveLocked()
	return params, nil
}

type persistedScene struct {
	ActiveScene SceneMode `json:"active_scene"`
	SwitchedAt  string    `json:"switched_at"`
}

func (m *ScenePresetManager) load() {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return
	}

	var p persistedScene
	if err := json.Unmarshal(data, &p); err == nil && p.ActiveScene != "" {
		if _, ok := m.definitions[p.ActiveScene]; ok {
			m.activeScene = p.ActiveScene
			if t, err := time.Parse(time.RFC3339, p.SwitchedAt); err == nil {
				m.switchedAt = t
			}
		}
	}
}

func (m *ScenePresetManager) saveLocked() {
	if m.filePath == "" {
		return
	}

	dir := filepath.Dir(m.filePath)
	_ = os.MkdirAll(dir, 0755)

	p := persistedScene{
		ActiveScene: m.activeScene,
		SwitchedAt:  m.switchedAt.Format(time.RFC3339),
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
