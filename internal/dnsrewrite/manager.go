package dnsrewrite

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

// Config represents the persisted DNS rewrite configuration.
type Config struct {
	Enabled bool                    `json:"enabled"`
	Rules   []schema.DNSRewriteRule `json:"rules"`
}

// Manager manages domain alias (CNAME) and static IP (A) rewrites.
type Manager struct {
	mu       sync.RWMutex
	filePath string
	cfg      Config
}

// DefaultConfig returns the production defaults (Google Play fix included).
func DefaultConfig() Config {
	return Config{
		Enabled: true,
		Rules: []schema.DNSRewriteRule{
			{
				ID:        "rw_google_play_services",
				Domain:    "services.googleapis.cn",
				Type:      schema.DNSRewriteTypeCNAME,
				Target:    "services.googleapis.com",
				Comment:   "解决 Google Play 无法下载 App 问题 (0% 卡顿)",
				Enabled:   true,
				CreatedAt: "2026-10-22T00:00:00Z",
			},
			{
				ID:        "rw_google_developers",
				Domain:    "developers.google.cn",
				Type:      schema.DNSRewriteTypeCNAME,
				Target:    "developers.google.com",
				Comment:   "重定向至 Google 开发者海外官方全功能域",
				Enabled:   true,
				CreatedAt: "2026-10-22T00:00:00Z",
			},
			{
				ID:        "rw_example_a_record",
				Domain:    "nas.lan",
				Type:      schema.DNSRewriteTypeA,
				Target:    "10.0.0.100",
				Comment:   "局域网 A 记录静态映射示例 (如将自定义域名直连私有 NAS 或服务器)",
				Enabled:   false,
				CreatedAt: "2026-10-25T00:00:00Z",
			},
		},
	}
}

// NewManager creates a new DNS Rewrite manager with persistent disk storage.
func NewManager(workDir string) *Manager {
	dir := filepath.Join(workDir, "data")
	_ = os.MkdirAll(dir, 0755)
	p := filepath.Join(dir, "dns_rewrite.json")

	m := &Manager{
		filePath: p,
		cfg:      DefaultConfig(),
	}

	m.loadLocked()
	return m
}

func (m *Manager) loadLocked() {
	if _, err := os.Stat(m.filePath); os.IsNotExist(err) {
		m.saveLocked()
		return
	}
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return
	}
	var loaded Config
	if err := json.Unmarshal(data, &loaded); err == nil {
		if len(loaded.Rules) == 0 {
			loaded = DefaultConfig()
		}
		m.cfg = loaded
	}
}

func (m *Manager) saveLocked() error {
	data, err := json.MarshalIndent(m.cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.filePath, data, 0644)
}

// GetConfig returns a deep copy of the DNS rewrite configuration.
func (m *Manager) GetConfig() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	copied := Config{
		Enabled: m.cfg.Enabled,
		Rules:   make([]schema.DNSRewriteRule, len(m.cfg.Rules)),
	}
	copy(copied.Rules, m.cfg.Rules)
	return copied
}

// SaveConfig updates and saves the DNS rewrite configuration.
func (m *Manager) SaveConfig(cfg Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg = cfg
	return m.saveLocked()
}

// IsEnabled returns true if DNS rewriting is globally active.
func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.Enabled
}

// SetEnabled toggles the global DNS rewrite engine.
func (m *Manager) SetEnabled(enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg.Enabled = enabled
	return m.saveLocked()
}

// GetAllRules returns all configured rewrite rules.
func (m *Manager) GetAllRules() []schema.DNSRewriteRule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]schema.DNSRewriteRule, len(m.cfg.Rules))
	copy(res, m.cfg.Rules)
	return res
}

// GetActiveRules returns only enabled rewrite rules when global switch is ON.
func (m *Manager) GetActiveRules() []schema.DNSRewriteRule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.cfg.Enabled {
		return nil
	}
	var res []schema.DNSRewriteRule
	for _, r := range m.cfg.Rules {
		if r.Enabled && r.Domain != "" && r.Target != "" {
			res = append(res, r)
		}
	}
	return res
}

// AddRule creates a new rewrite rule.
func (m *Manager) AddRule(domain string, rType schema.DNSRewriteType, target, comment string) (*schema.DNSRewriteRule, error) {
	domain = strings.TrimSpace(strings.ToLower(domain))
	target = strings.TrimSpace(target)
	comment = strings.TrimSpace(comment)

	if domain == "" || target == "" {
		return nil, fmt.Errorf("domain and target cannot be empty")
	}

	// Auto detect type if not explicitly set
	if rType == "" {
		if net.ParseIP(target) != nil {
			rType = schema.DNSRewriteTypeA
		} else {
			rType = schema.DNSRewriteTypeCNAME
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check duplicates
	for _, r := range m.cfg.Rules {
		if strings.EqualFold(r.Domain, domain) {
			return nil, fmt.Errorf("rewrite rule for domain %s already exists", domain)
		}
	}

	newRule := schema.DNSRewriteRule{
		ID:        fmt.Sprintf("rw_%d", time.Now().UnixNano()),
		Domain:    domain,
		Type:      rType,
		Target:    target,
		Comment:   comment,
		Enabled:   true,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	m.cfg.Rules = append(m.cfg.Rules, newRule)
	if err := m.saveLocked(); err != nil {
		return nil, err
	}
	return &newRule, nil
}

// UpdateRule updates an existing rewrite rule.
func (m *Manager) UpdateRule(id, domain string, rType schema.DNSRewriteType, target, comment string, enabled bool) error {
	domain = strings.TrimSpace(strings.ToLower(domain))
	target = strings.TrimSpace(target)
	comment = strings.TrimSpace(comment)

	if domain == "" || target == "" {
		return fmt.Errorf("domain and target cannot be empty")
	}

	if rType == "" {
		if net.ParseIP(target) != nil {
			rType = schema.DNSRewriteTypeA
		} else {
			rType = schema.DNSRewriteTypeCNAME
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	found := false
	for i := range m.cfg.Rules {
		if m.cfg.Rules[i].ID == id {
			// Check domain collision with other rules
			for j := range m.cfg.Rules {
				if j != i && strings.EqualFold(m.cfg.Rules[j].Domain, domain) {
					return fmt.Errorf("another rule for domain %s already exists", domain)
				}
			}
			m.cfg.Rules[i].Domain = domain
			m.cfg.Rules[i].Type = rType
			m.cfg.Rules[i].Target = target
			m.cfg.Rules[i].Comment = comment
			m.cfg.Rules[i].Enabled = enabled
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("rule %s not found", id)
	}

	return m.saveLocked()
}

// DeleteRule removes a rewrite rule by ID.
func (m *Manager) DeleteRule(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := -1
	for i, r := range m.cfg.Rules {
		if r.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("rule %s not found", id)
	}

	m.cfg.Rules = append(m.cfg.Rules[:idx], m.cfg.Rules[idx+1:]...)
	return m.saveLocked()
}

// ToggleRule enables or disables a single rewrite rule.
func (m *Manager) ToggleRule(id string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	found := false
	for i := range m.cfg.Rules {
		if m.cfg.Rules[i].ID == id {
			m.cfg.Rules[i].Enabled = enabled
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("rule %s not found", id)
	}
	return m.saveLocked()
}

// ResetDefaults restores the default Google Play rules.
func (m *Manager) ResetDefaults() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cfg = DefaultConfig()
	return m.saveLocked()
}
