package schema

import (
	"os"
	"path/filepath"
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
	Meta           MetaConfig       `json:"meta" yaml:"meta"`
	Network        NetworkConfig    `json:"network" yaml:"network"`
	DNS            DNSConfig        `json:"dns" yaml:"dns"`
	Subscriptions  []Subscription   `json:"subscriptions" yaml:"subscriptions"`
	Proxies        []ProxyNode      `json:"proxies" yaml:"proxies"`
	Groups         []StrategyGroup  `json:"groups" yaml:"groups"`
	Rules          []Rule           `json:"rules" yaml:"rules"`
	DevicePolicies []DevicePolicy   `json:"device_policies" yaml:"device_policies"`
	Profiles       []Profile        `json:"profiles" yaml:"profiles"`
	UpdatedAt      time.Time        `json:"updated_at" yaml:"updated_at"`
}

type MetaConfig struct {
	Version       string     `json:"version" yaml:"version"`
	ActiveEngine  EngineType `json:"active_engine" yaml:"active_engine"`
	ProxyMode     ProxyMode  `json:"proxy_mode" yaml:"proxy_mode"`
	ActiveProfile string     `json:"active_profile" yaml:"active_profile"`
	Edition       string     `json:"edition" yaml:"edition"` // "geek" or "novice"
	LogLevel      string     `json:"log_level" yaml:"log_level"`
}

type NetworkConfig struct {
	TunInterface string `json:"tun_interface" yaml:"tun_interface"` // e.g. cubex-tun
	TunIPv4      string `json:"tun_ipv4" yaml:"tun_ipv4"`           // e.g. 172.19.0.1/30
	TunMTU       int    `json:"tun_mtu" yaml:"tun_mtu"`             // e.g. 9000
	TunStack     string `json:"tun_stack" yaml:"tun_stack"`         // mixed / system / gvisor
	TProxyPort   int    `json:"tproxy_port" yaml:"tproxy_port"`     // 7893
	MixedPort    int    `json:"mixed_port" yaml:"mixed_port"`       // 7890
	RedirPort    int    `json:"redir_port" yaml:"redir_port"`       // 7892
	DNSPort      int    `json:"dns_port" yaml:"dns_port"`           // 53
	ClashAPIPort int    `json:"clash_api_port" yaml:"clash_api_port"` // 9090
	ClashSecret  string `json:"clash_secret" yaml:"clash_secret"`
	AllowLAN     bool   `json:"allow_lan" yaml:"allow_lan"`
	IPv6         bool   `json:"ipv6" yaml:"ipv6"`
}

type DNSConfig struct {
	EnableFakeIP bool     `json:"enable_fake_ip" yaml:"enable_fake_ip"`
	FakeIPRange  string   `json:"fake_ip_range" yaml:"fake_ip_range"` // 198.18.0.1/16
	FakeIPFilter []string `json:"fake_ip_filter" yaml:"fake_ip_filter"`
	DirectDNS    []string `json:"direct_dns" yaml:"direct_dns"`
	RemoteDNS    []string `json:"remote_dns" yaml:"remote_dns"`
	FallbackDNS  []string `json:"fallback_dns" yaml:"fallback_dns"`
	DNSRules     []DNSRule `json:"dns_rules" yaml:"dns_rules"`
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
	Name           string            `json:"name" yaml:"name"`
	Type           string            `json:"type" yaml:"type"` // ss, vmess, vless, trojan, hysteria2, tuic, wireguard, direct, block
	Server         string            `json:"server" yaml:"server"`
	Port           int               `json:"port" yaml:"port"`
	UUID           string            `json:"uuid,omitempty" yaml:"uuid,omitempty"`
	Password       string            `json:"password,omitempty" yaml:"password,omitempty"`
	Cipher         string            `json:"cipher,omitempty" yaml:"cipher,omitempty"`
	TLS            bool              `json:"tls,omitempty" yaml:"tls,omitempty"`
	SNI            string            `json:"sni,omitempty" yaml:"sni,omitempty"`
	ALPN           []string          `json:"alpn,omitempty" yaml:"alpn,omitempty"`
	SkipCertVerify bool              `json:"skip_cert_verify,omitempty" yaml:"skip_cert_verify,omitempty"`
	Network        string            `json:"network,omitempty" yaml:"network,omitempty"` // tcp, ws, grpc, http
	Path           string            `json:"path,omitempty" yaml:"path,omitempty"`
	Flow           string            `json:"flow,omitempty" yaml:"flow,omitempty"`
	Congestion     string            `json:"congestion,omitempty" yaml:"congestion,omitempty"`
	Delay          int               `json:"delay" yaml:"delay"` // measured latency ms
	SubID          string            `json:"sub_id,omitempty" yaml:"sub_id,omitempty"`
	RawConfig      map[string]any    `json:"raw_config,omitempty" yaml:"raw_config,omitempty"`
}

type StrategyGroup struct {
	Name      string   `json:"name" yaml:"name"`
	Type      string   `json:"type" yaml:"type"` // select, urltest, fallback, loadbalance
	Proxies   []string `json:"proxies" yaml:"proxies"`
	Now       string   `json:"now" yaml:"now"` // currently active node
	URL       string   `json:"url,omitempty" yaml:"url,omitempty"`
	Interval  int      `json:"interval,omitempty" yaml:"interval,omitempty"`
	Tolerance int      `json:"tolerance,omitempty" yaml:"tolerance,omitempty"`
	Icon      string   `json:"icon,omitempty" yaml:"icon,omitempty"`           // openai, youtube, google, github, telegram, twitter, netflix, apple, microsoft, default
	Category  string   `json:"category,omitempty" yaml:"category,omitempty"`   // business / region
	Region    string   `json:"region,omitempty" yaml:"region,omitempty"`       // HK, TW, SG, JPKR, US, OTHERS
	Policy    string   `json:"policy,omitempty" yaml:"policy,omitempty"`       // auto, fallback, disaster
	IsCustom  bool     `json:"is_custom,omitempty" yaml:"is_custom,omitempty"` // whether user manually added this
}

type Rule struct {
	Type   string `json:"type" yaml:"type"`     // domain, domain_suffix, domain_keyword, geosite, geoip, ip_cidr, src_ip_cidr, port, process_name, match
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
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	DirectBypass bool  `json:"direct_bypass" yaml:"direct_bypass"`
	QuicBlock   bool   `json:"quic_block" yaml:"quic_block"`
	AdBlock     bool   `json:"ad_block" yaml:"ad_block"`
	SpeedtestURL string `json:"speedtest_url" yaml:"speedtest_url"`
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
			IPv6:         false,
		},
		DNS: DNSConfig{
			EnableFakeIP: true,
			FakeIPRange:  "198.18.0.1/16",
			FakeIPFilter: []string{
				"geosite:cn",
				"+.cn",
				"*.lan", "*.local", "localhost",
				"+.msftconnecttest.com", "+.msftncsi.com",
				"+.routeros.com", "+.mikrotik.com",
				"+.bilibili.com", "+.baidu.com", "+.qq.com",
				"+.taobao.com", "+.jd.com", "+.alipay.com", "+.163.com",
			},
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
		DevicePolicies: []DevicePolicy{
			{Name: "家庭极空间 NAS (数据中心)", IP: "10.0.0.88", Mode: "bypass"},
			{Name: "客厅 Apple TV 4K", IP: "10.0.0.120", Mode: "rule"},
			{Name: "主卧智能电视", IP: "10.0.0.122", Mode: "rule"},
			{Name: "书房主力 PC (开发工作站)", IP: "10.0.0.101", Mode: "rule"},
			{Name: "索尼 PS5 游戏主机", IP: "10.0.0.199", Mode: "rule"},
		},
		Profiles: []Profile{
			{
				ID:           "balanced",
				Name:         "家庭日常平衡",
				Description:  "智能规则分流，优先直连，国内低延迟，海外秒开",
				DirectBypass: true,
				QuicBlock:    true,
				AdBlock:      true,
				SpeedtestURL: "http://www.gstatic.com/generate_204",
			},
			{
				ID:           "gaming",
				Name:         "游戏低延迟专线",
				Description:  "TUN 协议栈内核优化，UDP 全链路加速，防跳出口",
				DirectBypass: true,
				QuicBlock:    false,
				AdBlock:      true,
				SpeedtestURL: "http://www.gstatic.com/generate_204",
			},
			{
				ID:           "geek",
				Name:         "极客无损调试",
				Description:  "全链路嗅探与日志追踪，无损全协议开放",
				DirectBypass: false,
				QuicBlock:    false,
				AdBlock:      false,
				SpeedtestURL: "http://www.gstatic.com/generate_204",
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

