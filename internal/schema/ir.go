package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EngineType represents the active proxy core engine persona.
type EngineType string

const (
	EngineMihomo  EngineType = "mihomo"
	EngineSingbox EngineType = "singbox"
	EngineHybrid  EngineType = "hybrid"
)

// ProxyMode represents routing operational modes.
type ProxyMode string

const (
	ModeRule   ProxyMode = "rule"
	ModeGlobal ProxyMode = "global"
	ModeDirect ProxyMode = "direct"
)

// ConfigIR is the Universal Intermediate Representation for CubeX.
// Both Mihomo YAML and sing-box v1.14+ JSON are compiled from this single source of truth.
type ConfigIR struct {
	Meta              MetaConfig       `json:"meta" yaml:"meta"`
	Network           NetworkConfig    `json:"network" yaml:"network"`
	DNS               DNSConfig        `json:"dns" yaml:"dns"`
	Subscriptions     []Subscription   `json:"subscriptions" yaml:"subscriptions"`
	Proxies           []ProxyNode      `json:"proxies" yaml:"proxies"`
	Groups            []StrategyGroup  `json:"groups" yaml:"groups"`
	Rules             []Rule           `json:"rules" yaml:"rules"`
	DevicePolicies    []DevicePolicy   `json:"device_policies" yaml:"device_policies"`
	Profiles          []Profile        `json:"profiles" yaml:"profiles"`
	AdBlockWhitelist  []string         `json:"adblock_whitelist,omitempty" yaml:"adblock_whitelist,omitempty"`
	AdBlockEnabled    bool             `json:"adblock_enabled" yaml:"adblock_enabled"`
	DNSRewriteEnabled bool             `json:"dns_rewrite_enabled" yaml:"dns_rewrite_enabled"`
	DNSRewrites       []DNSRewriteRule `json:"dns_rewrites,omitempty" yaml:"dns_rewrites,omitempty"`
	UserOverrides     *UserOverrideRules `json:"user_overrides,omitempty" yaml:"user_overrides,omitempty"`
	UpdatedAt         time.Time        `json:"updated_at" yaml:"updated_at"`
}

// UserOverrideRules 定义用户最高优先级的持久化自定义分流规则（与订阅解耦，永不冲刷）
type UserOverrideRules struct {
	ForceDirectDomains []string `json:"force_direct_domains,omitempty" yaml:"force_direct_domains,omitempty"`
	ForceProxyDomains  []string `json:"force_proxy_domains,omitempty" yaml:"force_proxy_domains,omitempty"`
	ForceRejectDomains []string `json:"force_reject_domains,omitempty" yaml:"force_reject_domains,omitempty"`
}

type MetaConfig struct {
	Version       string     `json:"version" yaml:"version"`
	ActiveEngine  EngineType `json:"active_engine" yaml:"active_engine"`
	ProxyMode     ProxyMode  `json:"proxy_mode" yaml:"proxy_mode"`
	ProxyPaused   bool       `json:"proxy_paused,omitempty" yaml:"proxy_paused,omitempty"`
	PrePauseMode  ProxyMode  `json:"pre_pause_mode,omitempty" yaml:"pre_pause_mode,omitempty"`
	ActiveProfile string     `json:"active_profile" yaml:"active_profile"`
	Edition       string     `json:"edition" yaml:"edition"` // "geek" or "novice"
	LogLevel      string     `json:"log_level" yaml:"log_level"`
}

type NetworkConfig struct {
	TunInterface string `json:"tun_interface" yaml:"tun_interface"`   // e.g. cubex-tun
	TunIPv4      string `json:"tun_ipv4" yaml:"tun_ipv4"`             // e.g. 172.19.0.1/30
	TunMTU       int    `json:"tun_mtu" yaml:"tun_mtu"`               // e.g. 9000
	TunStack     string `json:"tun_stack" yaml:"tun_stack"`           // mixed / system / gvisor
	TProxyPort   int    `json:"tproxy_port" yaml:"tproxy_port"`       // 7893
	MixedPort    int    `json:"mixed_port" yaml:"mixed_port"`         // 7890
	RedirPort    int    `json:"redir_port" yaml:"redir_port"`         // 7892
	DNSPort      int    `json:"dns_port" yaml:"dns_port"`             // 53
	ClashAPIPort int    `json:"clash_api_port" yaml:"clash_api_port"` // 9090
	ClashSecret  string `json:"clash_secret" yaml:"clash_secret"`
	AllowLAN        bool   `json:"allow_lan" yaml:"allow_lan"`
	IPv6            bool   `json:"ipv6" yaml:"ipv6"`
	QUICPolicy      string `json:"quic_policy" yaml:"quic_policy"` // smart (default) / block / direct
	RouteAllTraffic bool   `json:"route_all_traffic,omitempty" yaml:"route_all_traffic,omitempty"` // false (default): Fake-IP convergence (198.18.0.0/15) - zeroes NAS STUN/PT conflicts; true: traditional 0.0.0.0/0
}

type DNSConfig struct {
	EnableFakeIP         bool      `json:"enable_fake_ip" yaml:"enable_fake_ip"`
	FakeIPRange          string    `json:"fake_ip_range" yaml:"fake_ip_range"` // 198.18.0.0/15
	FakeIPBuiltinEnabled *bool     `json:"fakeip_builtin_enabled,omitempty" yaml:"fakeip_builtin_enabled,omitempty"`
	FakeIPFilter         []string  `json:"fake_ip_filter" yaml:"fake_ip_filter"`
	DirectDNS            []string  `json:"direct_dns" yaml:"direct_dns"`
	RemoteDNS            []string  `json:"remote_dns" yaml:"remote_dns"`
	FallbackDNS          []string  `json:"fallback_dns" yaml:"fallback_dns"`
	DNSRules             []DNSRule `json:"dns_rules" yaml:"dns_rules"`
}

// IsFakeIPBuiltinEnabled reports whether the built-in high-compatibility Fake-IP bypass list is active (defaults to true).
func (d DNSConfig) IsFakeIPBuiltinEnabled() bool {
	if d.FakeIPBuiltinEnabled == nil {
		return true
	}
	return *d.FakeIPBuiltinEnabled
}

// SetFakeIPBuiltinEnabled explicitly sets whether the built-in Fake-IP bypass list is active.
func (d *DNSConfig) SetFakeIPBuiltinEnabled(enabled bool) {
	if d == nil {
		return
	}
	v := enabled
	d.FakeIPBuiltinEnabled = &v
}

// BuiltinFakeIPFilterDomains is the single-source-of-truth high-compatibility real-IP bypass list (>= 32 entries)
// covering domestic banking apps, MiHome/HomeKit smart home, NTP/STUN traversal, BT/PT trackers, and LAN connectivity.
var BuiltinFakeIPFilterDomains = []string{
	// 1. LAN & Connectivity Detection
	"*.lan",
	"*.local",
	"*.home",
	"*.internal",
	"*.home.arpa",
	"*.localdomain",
	"localhost",
	"+.cn",
	"+.msftconnecttest.com",
	"+.msftncsi.com",
	"+.connectivitycheck.gstatic.com",
	"+.cp.cloudflare.com",
	"+.test-ipv6.com",
	"+.routeros.com",
	"+.mikrotik.com",
	"+.cloud.mikrotik.com",

	// 2. Domestic Banking & Payment Apps
	"+.icbc.com.cn",
	"+.ccb.com",
	"+.abchina.com",
	"+.boc.cn",
	"+.bankcomm.com",
	"+.cmbchina.com",
	"+.cmbimg.com",
	"+.psbc.com",
	"+.spdb.com.cn",
	"+.cib.com.cn",
	"+.ecitic.com",
	"+.cebbank.com",
	"+.pingan.com",
	"+.cmbc.com.cn",
	"+.hxb.com.cn",
	"+.unionpay.com",
	"+.95516.com",
	"+.alipay.com",

	// 3. Smart Home (MiHome / HomeKit / IoT)
	"dlg.io.mi.com",
	"+.io.mi.com",
	"+.miwifi.com",
	"+.mijia.tech",
	"+.xiaomi.com",
	"+.homekit.apple.com",
	"+.push.apple.com",
	"+.ls.apple.com",
	"+.gs.apple.com",
	"+.tuya.com",
	"+.tuyacn.com",
	"+.aqara.com",
	"+.aqara.cn",
	"+.home-assistant.io",

	// 4. NTP Time Sync & STUN/TURN NAT Traversal
	"*.ntp.org",
	"+.pool.ntp.org",
	"pool.ntp.org",
	"+.time.windows.com",
	"+.time.apple.com",
	"time.asia.apple.com",
	"+.time.google.com",
	"+.time.cloudflare.com",
	"+.time.edu.cn",
	"+.ntp.aliyun.com",
	"+.ntp.tencent.com",
	"time.nist.gov",
	"+.stun.*",
	"stun.*",
	"*.stun.*",
	"+.turn.*",
	"turn.*",
	"*.turn.*",
	"stun.miwifi.com",
	"stun.qq.com",
	"stun.synology.com",
	"stun.bilibili.com",
	"stun.163.com",
	"stun.cloudflare.com",
	"stun.l.google.com",

	// 5. BT / PT Trackers & NAS Direct Resolving
	"+.tracker.*",
	"tracker.*",
	"*.tracker.*",
	"+.announce.*",
	"announce.*",
	"+.m-team.cc",
	"+.chdbits.co",
	"+.hdchina.org",
	"+.ourbits.club",
	"+.springsunday.net",
	"+.audiences.me",
	"+.pterclub.com",
	"+.keepfrds.com",
	"+.opencd.cc",
	"+.totheglory.im",
	"+.hdsky.me",
	"+.zspace.com",
	"+.zspace.cn",
	"+.zspace.com.cn",
	"+.zspace.cc",

	// 6. Core Domestic Internet Services
	"+.baidu.com",
	"+.bdstatic.com",
	"+.qq.com",
	"+.qpic.cn",
	"+.weixin.qq.com",
	"+.wechat.com",
	"+.bilibili.com",
	"+.hdslb.com",
	"+.bilivideo.com",
	"+.bilivideo.cn",
	"+.biliapi.net",
	"+.biliapi.com",
	"+.taobao.com",
	"+.alicdn.com",
	"+.jd.com",
	"+.163.com",
	"+.zhihu.com",
	"+.douyin.com",
	"+.weibo.com",
	"+.aliyun.com",
	"+.aliyuncs.com",
	"+.tencent.com",
	"+.bytedance.com",
}

// BuiltinFakeIPFilterCount returns the number of entries in the built-in Fake-IP bypass list.
func BuiltinFakeIPFilterCount() int {
	return len(BuiltinFakeIPFilterDomains)
}

// NormalizeFakeIPDomain validates and canonicalizes a single Fake-IP filter domain pattern.
func NormalizeFakeIPDomain(raw string) (string, error) {
	d := strings.ToLower(strings.TrimSpace(raw))
	d = strings.TrimSuffix(d, ".")
	if d == "" {
		return "", fmt.Errorf("domain cannot be empty")
	}
	if len(d) > 253 {
		return "", fmt.Errorf("domain %q exceeds maximum length", raw)
	}
	if strings.ContainsAny(d, " \t\r\n/\\@:?#!$%^&()=~`\"'<>;,") {
		return "", fmt.Errorf("invalid characters in domain %q", raw)
	}

	prefix := ""
	core := d
	switch {
	case strings.HasPrefix(d, "+."):
		prefix = "+."
		core = strings.TrimPrefix(d, "+.")
	case strings.HasPrefix(d, "*."):
		prefix = "*."
		core = strings.TrimPrefix(d, "*.")
	case strings.HasPrefix(d, "."):
		prefix = "+."
		core = strings.TrimPrefix(d, ".")
	}
	if core == "" || strings.Contains(core, "..") || strings.Contains(core, "+") {
		return "", fmt.Errorf("invalid domain syntax %q", raw)
	}
	if prefix == "" && core != "localhost" && !strings.Contains(core, ".") {
		return "", fmt.Errorf("domain %q must contain at least one dot", raw)
	}

	labels := strings.Split(core, ".")
	for _, lbl := range labels {
		if lbl == "" || len(lbl) > 63 {
			return "", fmt.Errorf("invalid domain label in %q", raw)
		}
		if lbl == "*" {
			continue
		}
		if strings.HasPrefix(lbl, "-") || strings.HasSuffix(lbl, "-") {
			return "", fmt.Errorf("invalid hyphen placement in %q", raw)
		}
		for i := 0; i < len(lbl); i++ {
			c := lbl[i]
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '*') {
				return "", fmt.Errorf("invalid character %q in domain %q", c, raw)
			}
		}
	}
	return prefix + core, nil
}

// NormalizeFakeIPDomains validates, normalizes, and deduplicates a slice of Fake-IP filter domains.
func NormalizeFakeIPDomains(domains []string) ([]string, error) {
	out := make([]string, 0, len(domains))
	seen := make(map[string]struct{}, len(domains))
	for _, raw := range domains {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		norm, err := NormalizeFakeIPDomain(raw)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[norm]; ok {
			continue
		}
		seen[norm] = struct{}{}
		out = append(out, norm)
	}
	return out, nil
}

type DNSRule struct {
	DomainMatches []string `json:"domain_matches" yaml:"domain_matches"`
	ServerTag     string   `json:"server_tag" yaml:"server_tag"` // direct / remote
}

type Subscription struct {
	ID         string    `json:"id" yaml:"id"`
	Name       string    `json:"name" yaml:"name"`
	URL        string    `json:"url" yaml:"url"`
	Type       string    `json:"type" yaml:"type"` // clash / singbox / v2ray
	Enabled    bool      `json:"enabled" yaml:"enabled"`
	NodeCount  int       `json:"node_count" yaml:"node_count"`
	UsedBytes  int64     `json:"used_bytes" yaml:"used_bytes"`
	TotalBytes int64     `json:"total_bytes" yaml:"total_bytes"`
	ExpireAt   int64     `json:"expire_at" yaml:"expire_at"`
	UpdatedAt  time.Time `json:"updated_at" yaml:"updated_at"`
}

// ProxyNode represents an outbound node in standard representation.
type ProxyNode struct {
	Name           string         `json:"name" yaml:"name"`
	Type           string         `json:"type" yaml:"type"` // ss, vmess, vless, trojan, hysteria2, tuic, wireguard, direct, block
	Server         string         `json:"server" yaml:"server"`
	Port           int            `json:"port" yaml:"port"`
	Ports          string         `json:"ports,omitempty" yaml:"ports,omitempty"` // Port hopping range (e.g. 20000-40000 or 20000:40000)
	UUID           string         `json:"uuid,omitempty" yaml:"uuid,omitempty"`
	Password       string         `json:"password,omitempty" yaml:"password,omitempty"`
	Cipher         string         `json:"cipher,omitempty" yaml:"cipher,omitempty"`
	TLS            bool           `json:"tls,omitempty" yaml:"tls,omitempty"`
	SNI            string         `json:"sni,omitempty" yaml:"sni,omitempty"`
	ALPN           []string       `json:"alpn,omitempty" yaml:"alpn,omitempty"`
	SkipCertVerify bool           `json:"skip_cert_verify,omitempty" yaml:"skip_cert_verify,omitempty"`
	Network        string         `json:"network,omitempty" yaml:"network,omitempty"` // tcp, ws, grpc, http
	Path           string         `json:"path,omitempty" yaml:"path,omitempty"`
	Flow           string         `json:"flow,omitempty" yaml:"flow,omitempty"`
	Congestion     string         `json:"congestion,omitempty" yaml:"congestion,omitempty"`
	Delay          int            `json:"delay" yaml:"delay"`                                   // measured latency ms
	DelayStatus    string         `json:"delay_status,omitempty" yaml:"delay_status,omitempty"` // ok, timeout, dns_failed, conn_refused, resource_limit, tls_failed, conn_reset, error
	DelayText      string         `json:"delay_text,omitempty" yaml:"delay_text,omitempty"`     // 正常, 握手超时, DNS解析失败, 连接被拒, 系统资源受限, TLS握手异常, 连接重置, 连接异常
	SubID          string         `json:"sub_id,omitempty" yaml:"sub_id,omitempty"`
	RawConfig      map[string]any `json:"raw_config,omitempty" yaml:"raw_config,omitempty"`
}

type StrategyGroup struct {
	Name           string   `json:"name" yaml:"name"`
	Type           string   `json:"type" yaml:"type"` // select, urltest, fallback, loadbalance
	Proxies        []string `json:"proxies" yaml:"proxies"`
	Now            string   `json:"now" yaml:"now"` // currently active node
	URL            string   `json:"url,omitempty" yaml:"url,omitempty"`
	Interval       int      `json:"interval,omitempty" yaml:"interval,omitempty"`
	Timeout        int      `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Strategy       string   `json:"strategy,omitempty" yaml:"strategy,omitempty"`
	Tolerance      int      `json:"tolerance,omitempty" yaml:"tolerance,omitempty"`
	Icon           string   `json:"icon,omitempty" yaml:"icon,omitempty"`                       // openai, youtube, google, github, telegram, twitter, netflix, apple, microsoft, default
	Category       string   `json:"category,omitempty" yaml:"category,omitempty"`               // business / region
	Region         string   `json:"region,omitempty" yaml:"region,omitempty"`                   // HK, TW, SG, JPKR, US, OTHERS
	Policy         string   `json:"policy,omitempty" yaml:"policy,omitempty"`                   // auto, fallback, disaster
	IsCustom       bool     `json:"is_custom,omitempty" yaml:"is_custom,omitempty"`             // whether user manually added this
	Disabled       bool     `json:"disabled,omitempty" yaml:"disabled,omitempty"`               // whether rule routing for this group is temporarily bypassed
	FilterKeywords []string `json:"filter_keywords,omitempty" yaml:"filter_keywords,omitempty"` // dynamic node filtering keywords
	FilterLogic    string   `json:"filter_logic,omitempty" yaml:"filter_logic,omitempty"`       // 'OR' (default) or 'AND'
	FilterRegex    string   `json:"filter_regex,omitempty" yaml:"filter_regex,omitempty"`       // optional regex filter
}

type Rule struct {
	Type   string `json:"type" yaml:"type"` // domain, domain_suffix, domain_keyword, geosite, geoip, ip_cidr, src_ip_cidr, port, process_name, match
	Value  string `json:"value" yaml:"value"`
	Target string `json:"target" yaml:"target"` // Proxy, Direct, Reject, or Group Name
	Invert bool   `json:"invert,omitempty" yaml:"invert,omitempty"`
}

type DevicePolicy struct {
	IP        string `json:"ip" yaml:"ip"`
	MAC       string `json:"mac,omitempty" yaml:"mac,omitempty"`
	Name      string `json:"name,omitempty" yaml:"name,omitempty"`
	Mode      string `json:"mode" yaml:"mode"` // rule / bypass / node
	BoundNode string `json:"bound_node,omitempty" yaml:"bound_node,omitempty"`
}

type Profile struct {
	ID           string `json:"id" yaml:"id"`
	Name         string `json:"name" yaml:"name"`
	Description  string `json:"description" yaml:"description"`
	DirectBypass bool   `json:"direct_bypass" yaml:"direct_bypass"`
	QuicBlock    bool   `json:"quic_block" yaml:"quic_block"`
	AdBlock      bool   `json:"ad_block" yaml:"ad_block"`
	SpeedtestURL string `json:"speedtest_url" yaml:"speedtest_url"`
}

// NormalizeDefaults fills missing required defaults when loading older or partially populated config.ir.json files.
func (ir *ConfigIR) NormalizeDefaults() {
	if ir == nil {
		return
	}
	def := NewDefaultIR()
	if strings.TrimSpace(ir.Meta.Version) == "" {
		ir.Meta.Version = def.Meta.Version
	}
	if ir.Meta.ActiveEngine != EngineMihomo && ir.Meta.ActiveEngine != EngineSingbox {
		ir.Meta.ActiveEngine = def.Meta.ActiveEngine
	}
	if ir.Meta.ProxyMode == "" {
		ir.Meta.ProxyMode = def.Meta.ProxyMode
	}
	if strings.TrimSpace(ir.Meta.ActiveProfile) == "" {
		ir.Meta.ActiveProfile = def.Meta.ActiveProfile
	}
	if strings.TrimSpace(ir.Meta.LogLevel) == "" {
		ir.Meta.LogLevel = def.Meta.LogLevel
	}
	if strings.TrimSpace(ir.Network.TunInterface) == "" {
		ir.Network.TunInterface = def.Network.TunInterface
	}
	if strings.TrimSpace(ir.Network.TunIPv4) == "" {
		ir.Network.TunIPv4 = def.Network.TunIPv4
	}
	if ir.Network.TunMTU <= 0 {
		ir.Network.TunMTU = def.Network.TunMTU
	}
	if strings.TrimSpace(ir.Network.TunStack) == "" {
		ir.Network.TunStack = def.Network.TunStack
	}
	if ir.Network.TProxyPort <= 0 {
		ir.Network.TProxyPort = def.Network.TProxyPort
	}
	if ir.Network.MixedPort <= 0 {
		ir.Network.MixedPort = def.Network.MixedPort
	}
	if ir.Network.RedirPort <= 0 {
		ir.Network.RedirPort = def.Network.RedirPort
	}
	if ir.Network.DNSPort <= 0 || ir.Network.DNSPort == 1053 {
		ir.Network.DNSPort = def.Network.DNSPort
	}
	if ir.Network.ClashAPIPort <= 0 {
		ir.Network.ClashAPIPort = def.Network.ClashAPIPort
	}
	if strings.TrimSpace(ir.Network.QUICPolicy) == "" {
		ir.Network.QUICPolicy = def.Network.QUICPolicy
	}
	if strings.TrimSpace(ir.DNS.FakeIPRange) == "" {
		ir.DNS.FakeIPRange = def.DNS.FakeIPRange
	}
	if ir.DNS.FakeIPBuiltinEnabled == nil {
		t := true
		ir.DNS.FakeIPBuiltinEnabled = &t
	}
	if ir.DNS.FakeIPFilter == nil {
		ir.DNS.FakeIPFilter = []string{}
	}
	if len(ir.DNS.DirectDNS) == 0 {
		ir.DNS.DirectDNS = append([]string(nil), def.DNS.DirectDNS...)
	}
	if len(ir.DNS.RemoteDNS) == 0 {
		ir.DNS.RemoteDNS = append([]string(nil), def.DNS.RemoteDNS...)
	}
	if ir.Subscriptions == nil {
		ir.Subscriptions = []Subscription{}
	}
	if ir.Proxies == nil {
		ir.Proxies = []ProxyNode{}
	}
	if ir.Rules == nil {
		ir.Rules = []Rule{}
	}
}

// NewDefaultIR creates an optimized default CubeX Universal IR configuration.
func NewDefaultIR() *ConfigIR {
	res := &ConfigIR{
		Meta: MetaConfig{
			Version:       "1.0.0",
			ActiveEngine:  EngineMihomo,
			ProxyMode:     ModeRule,
			ActiveProfile: "balanced",
			LogLevel:      "info",
		},
		Network: NetworkConfig{
			TunInterface: "cubex-tun",
			TunIPv4:      "172.19.0.1/30",
			TunMTU:       1500,
			TunStack:     "system",
			TProxyPort:   7893,
			MixedPort:    7890,
			RedirPort:    7892,
			DNSPort:      53,
			ClashAPIPort: 9090,
			ClashSecret:  "",
			AllowLAN:     true,
			IPv6:         true,
			QUICPolicy:   "direct",
		},
		DNS: DNSConfig{
			EnableFakeIP:         true,
			FakeIPRange:          "198.18.0.0/15",
			FakeIPBuiltinEnabled: func() *bool { t := true; return &t }(),
			FakeIPFilter:         []string{},
			DirectDNS: []string{
				"223.5.5.5",
				"119.29.29.29",
			},
			RemoteDNS: []string{
				"https://1.1.1.1/dns-query",
				"https://8.8.8.8/dns-query",
			},
			FallbackDNS: []string{
				"https://cloudflare-dns.com/dns-query",
			},
			DNSRules: []DNSRule{
				{DomainMatches: []string{"geosite:cn", "domain:cn"}, ServerTag: "direct"},
				{DomainMatches: []string{"geosite:geolocation-!cn"}, ServerTag: "remote"},
			},
		},
		Subscriptions: []Subscription{},
		Proxies: []ProxyNode{
			{Name: "DIRECT", Type: "direct"},
			{Name: "REJECT", Type: "block"},
		},
		Groups: []StrategyGroup{
			{
				Name:    "🐟 漏网之鱼",
				Type:    "select",
				Proxies: []string{"🇭🇰 香港自动", "DIRECT"},
				Now:     "🇭🇰 香港自动",
			},
			{
				Name:    "🎯 全球直连",
				Type:    "select",
				Proxies: []string{"DIRECT"},
				Now:     "DIRECT",
			},
			{
				Name:    "🛑 广告拦截",
				Type:    "select",
				Proxies: []string{"REJECT", "DIRECT"},
				Now:     "REJECT",
			},
		},
		Rules: []Rule{
			{Type: "geosite", Value: "category-ads-all", Target: "🛑 广告拦截"},
			{Type: "geosite", Value: "cn", Target: "🎯 全球直连"},
			{Type: "geoip", Value: "cn", Target: "🎯 全球直连"},
			{Type: "geoip", Value: "private", Target: "🎯 全球直连"},
			{Type: "match", Value: "", Target: "🐟 漏网之鱼"},
		},
		DevicePolicies: []DevicePolicy{},
		Profiles: []Profile{
			{
				ID:           "balanced",
				Name:         "家庭日常平衡",
				Description:  "智能规则分流，优先直连，国内低延迟，海外秒开",
				DirectBypass: true,
				QuicBlock:    true,
				AdBlock:      true,
				SpeedtestURL: "https://www.gstatic.com/generate_204",
			},
			{
				ID:           "gaming",
				Name:         "游戏低延迟专线",
				Description:  "TUN 协议栈内核优化，UDP 全链路加速，防跳出口",
				DirectBypass: true,
				QuicBlock:    false,
				AdBlock:      true,
				SpeedtestURL: "https://www.gstatic.com/generate_204",
			},
			{
				ID:           "geek",
				Name:         "极客无损调试",
				Description:  "全链路嗅探与日志追踪，无损全协议开放",
				DirectBypass: false,
				QuicBlock:    false,
				AdBlock:      false,
				SpeedtestURL: "https://www.gstatic.com/generate_204",
			},
		},
		UpdatedAt: time.Now(),
	}
	SyncAllStrategyGroups(res)
	return res
}

// WriteFileAtomicSecure writes data to a temporary file with perm 0600 (owner only)
// and atomically renames it to destPath, preventing corruption from crashes and unauthorized access.
func WriteFileAtomicSecure(destPath string, data []byte) error {
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(dir, "cubex-tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if err := tmpFile.Chmod(0600); err != nil {
		tmpFile.Close()
		return err
	}
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, destPath)
}

type DNSRewriteType string

const (
	DNSRewriteTypeCNAME DNSRewriteType = "CNAME"
	DNSRewriteTypeA     DNSRewriteType = "A"
)

type DNSRewriteRule struct {
	ID        string         `json:"id" yaml:"id"`
	Domain    string         `json:"domain" yaml:"domain"`
	Type      DNSRewriteType `json:"type" yaml:"type"`
	Target    string         `json:"target" yaml:"target"`
	Comment   string         `json:"comment" yaml:"comment"`
	Enabled   bool           `json:"enabled" yaml:"enabled"`
	CreatedAt string         `json:"created_at" yaml:"created_at"`
}

func cloneAnyValue(v any) any {
	switch tv := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(tv))
		for k, val := range tv {
			out[k] = cloneAnyValue(val)
		}
		return out
	case []any:
		out := make([]any, len(tv))
		for i, val := range tv {
			out[i] = cloneAnyValue(val)
		}
		return out
	case []string:
		return append([]string(nil), tv...)
	default:
		return v
	}
}

func cloneAnyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = cloneAnyValue(v)
	}
	return out
}

// Clone returns a deep copy of ConfigIR so concurrent compilers and handlers never share mutable slices or maps.
func (ir *ConfigIR) Clone() *ConfigIR {
	if ir == nil {
		return nil
	}
	cp := *ir

	if ir.DNS.FakeIPBuiltinEnabled != nil {
		v := *ir.DNS.FakeIPBuiltinEnabled
		cp.DNS.FakeIPBuiltinEnabled = &v
	}
	cp.DNS.FakeIPFilter = append([]string(nil), ir.DNS.FakeIPFilter...)
	cp.DNS.DirectDNS = append([]string(nil), ir.DNS.DirectDNS...)
	cp.DNS.RemoteDNS = append([]string(nil), ir.DNS.RemoteDNS...)
	cp.DNS.FallbackDNS = append([]string(nil), ir.DNS.FallbackDNS...)
	if ir.DNS.DNSRules != nil {
		cp.DNS.DNSRules = make([]DNSRule, len(ir.DNS.DNSRules))
		for i, r := range ir.DNS.DNSRules {
			cp.DNS.DNSRules[i] = DNSRule{
				DomainMatches: append([]string(nil), r.DomainMatches...),
				ServerTag:     r.ServerTag,
			}
		}
	}

	cp.Subscriptions = append([]Subscription(nil), ir.Subscriptions...)

	if ir.Proxies != nil {
		cp.Proxies = make([]ProxyNode, len(ir.Proxies))
		for i, p := range ir.Proxies {
			pCopy := p
			pCopy.ALPN = append([]string(nil), p.ALPN...)
			pCopy.RawConfig = cloneAnyMap(p.RawConfig)
			cp.Proxies[i] = pCopy
		}
	}

	if ir.Groups != nil {
		cp.Groups = make([]StrategyGroup, len(ir.Groups))
		for i, g := range ir.Groups {
			gCopy := g
			gCopy.Proxies = append([]string(nil), g.Proxies...)
			gCopy.FilterKeywords = append([]string(nil), g.FilterKeywords...)
			cp.Groups[i] = gCopy
		}
	}

	cp.Rules = append([]Rule(nil), ir.Rules...)
	cp.DevicePolicies = append([]DevicePolicy(nil), ir.DevicePolicies...)
	cp.Profiles = append([]Profile(nil), ir.Profiles...)
	cp.AdBlockWhitelist = append([]string(nil), ir.AdBlockWhitelist...)
	cp.DNSRewrites = append([]DNSRewriteRule(nil), ir.DNSRewrites...)

	if ir.UserOverrides != nil {
		cp.UserOverrides = &UserOverrideRules{
			ForceDirectDomains: append([]string(nil), ir.UserOverrides.ForceDirectDomains...),
			ForceProxyDomains:  append([]string(nil), ir.UserOverrides.ForceProxyDomains...),
			ForceRejectDomains: append([]string(nil), ir.UserOverrides.ForceRejectDomains...),
		}
	}

	return &cp
}
