package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// UserOverrideSpec 定义用户自定义的最高优先级分流规则（独立持久化，订阅更新永不丢失）
type UserOverrideSpec struct {
	// ForceDirectDomains 强制走主路由物理直连的域名后缀（自动同步加入 fake-ip-filter，直接返回真 IP）
	ForceDirectDomains []string `json:"force_direct_domains"`
	// ForceProxyDomains 强制走代理隧道的域名后缀（自动注入路由规则最顶层）
	ForceProxyDomains []string `json:"force_proxy_domains"`
	// ForceRejectDomains 强制拦截屏蔽的广告/追踪域名后缀
	ForceRejectDomains []string `json:"force_reject_domains"`
}

// CompiledOverridePatch 封装编译期注入 Mihomo 与 sing-box 的增量补丁
type CompiledOverridePatch struct {
	ExtraFakeIPBypass   []string           // 需追加进 fake-ip-filter 的直连域名（保证真 IP 走主路由直出）
	MihomoPrependRules  []string           // 需置顶插入 Mihomo rules 的规则行
	SingBoxPrependRules []map[string]any   // 需置顶插入 sing-box route.rules 的规则对象
}

// UserOverrideManager 管控用户自定义分流规则的持久化与跨引擎编译
type UserOverrideManager struct {
	mu       sync.RWMutex
	filePath string
	current  UserOverrideSpec
}

// NewUserOverrideManager 初始化用户自定义分流规则管理器
func NewUserOverrideManager(workDir string) *UserOverrideManager {
	dataDir := filepath.Join(workDir, "data")
	m := &UserOverrideManager{
		filePath: filepath.Join(dataDir, "user_overrides.json"),
	}
	_ = m.loadFromDisk()
	return m
}

// GetSpec 获取当前用户自定义分流规则副本
func (m *UserOverrideManager) GetSpec() UserOverrideSpec {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return UserOverrideSpec{
		ForceDirectDomains: append([]string(nil), m.current.ForceDirectDomains...),
		ForceProxyDomains:  append([]string(nil), m.current.ForceProxyDomains...),
		ForceRejectDomains: append([]string(nil), m.current.ForceRejectDomains...),
	}
}

// UpdateSpec 净化并原子保存用户自定义规则
func (m *UserOverrideManager) UpdateSpec(spec UserOverrideSpec) error {
	cleaned := UserOverrideSpec{
		ForceDirectDomains: normalizeDomainList(spec.ForceDirectDomains),
		ForceProxyDomains:  normalizeDomainList(spec.ForceProxyDomains),
		ForceRejectDomains: normalizeDomainList(spec.ForceRejectDomains),
	}

	m.mu.Lock()
	m.current = cleaned
	err := m.saveToDiskLocked()
	m.mu.Unlock()
	return err
}

// CompilePatch 生成双擎置顶规则与 Fake-IP 免劫持联动列表
// primaryProxyGroup 为默认代理策略组名称（如 "Proxy" 或 "节点选择"）
func (m *UserOverrideManager) CompilePatch(primaryProxyGroup string) *CompiledOverridePatch {
	if primaryProxyGroup == "" {
		primaryProxyGroup = "Proxy"
	}
	spec := m.GetSpec()
	patch := &CompiledOverridePatch{
		ExtraFakeIPBypass:   make([]string, 0, len(spec.ForceDirectDomains)),
		MihomoPrependRules:  make([]string, 0, len(spec.ForceDirectDomains)+len(spec.ForceProxyDomains)+len(spec.ForceRejectDomains)),
		SingBoxPrependRules: make([]map[string]any, 0, 3),
	}

	// 1. 最高优：屏蔽规则 (REJECT / block)
	if len(spec.ForceRejectDomains) > 0 {
		for _, d := range spec.ForceRejectDomains {
			patch.MihomoPrependRules = append(patch.MihomoPrependRules, "DOMAIN-SUFFIX,"+d+",REJECT")
		}
		patch.SingBoxPrependRules = append(patch.SingBoxPrependRules, map[string]any{
			"domain_suffix": spec.ForceRejectDomains,
			"outbound":      "block",
		})
	}

	// 2. 次高优：强制直连规则 (DIRECT) + 同步注入 Fake-IP 免劫持白名单！
	if len(spec.ForceDirectDomains) > 0 {
		for _, d := range spec.ForceDirectDomains {
			patch.ExtraFakeIPBypass = append(patch.ExtraFakeIPBypass, "+."+d)
			patch.MihomoPrependRules = append(patch.MihomoPrependRules, "DOMAIN-SUFFIX,"+d+",DIRECT")
		}
		patch.SingBoxPrependRules = append(patch.SingBoxPrependRules, map[string]any{
			"domain_suffix": spec.ForceDirectDomains,
			"outbound":      "direct",
		})
	}

	// 3. 第三优：强制代理规则 (PROXY)
	if len(spec.ForceProxyDomains) > 0 {
		for _, d := range spec.ForceProxyDomains {
			patch.MihomoPrependRules = append(patch.MihomoPrependRules, "DOMAIN-SUFFIX,"+d+","+primaryProxyGroup)
		}
		patch.SingBoxPrependRules = append(patch.SingBoxPrependRules, map[string]any{
			"domain_suffix": spec.ForceProxyDomains,
			"outbound":      primaryProxyGroup,
		})
	}

	return patch
}

// normalizeDomainList 清洗用户输入的 URL/通配符（如将 "https://*.Example.com/path" 规范化为 "example.com"）
func normalizeDomainList(raw []string) []string {
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		s := strings.ToLower(strings.TrimSpace(item))
		s = strings.TrimPrefix(s, "http://")
		s = strings.TrimPrefix(s, "https://")
		if slashIdx := strings.IndexByte(s, '/'); slashIdx != -1 {
			s = s[:slashIdx]
		}
		if colonIdx := strings.IndexByte(s, ':'); colonIdx != -1 {
			s = s[:colonIdx]
		}
		s = strings.TrimPrefix(s, "+.")
		s = strings.TrimPrefix(s, "*.")
		s = strings.TrimPrefix(s, ".")
		if s == "" || !strings.Contains(s, ".") {
			continue
		}
		if _, exists := seen[s]; !exists {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

func (m *UserOverrideManager) loadFromDisk() error {
	raw, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}
	var spec UserOverrideSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return err
	}
	m.mu.Lock()
	m.current = spec
	m.mu.Unlock()
	return nil
}

func (m *UserOverrideManager) saveToDiskLocked() error {
	if err := os.MkdirAll(filepath.Dir(m.filePath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m.current, "", "  ")
	if err != nil {
		return err
	}
	tmp := m.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, m.filePath)
}
