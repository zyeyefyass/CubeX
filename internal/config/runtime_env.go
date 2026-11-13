package config

import (
	"os"
	"strconv"
	"time"
)

// RuntimeSpec 统一收口全项目所有网络端点、探针目标与资源配额，彻底消灭硬编码
type RuntimeSpec struct {
	// 1. 网络与端口绑定（取代散落的 "10.0.0.7", 2026, 7890, 9090, 53, "cubex-tun"）
	CubeXHostIP       string        `json:"cubex_host_ip"`       // 默认自动探测本机主网卡 IP，兜底 "10.0.0.7"
	ControlListenAddr string        `json:"control_listen_addr"` // 默认 ":2026"
	ClashAPIAddr      string        `json:"clash_api_addr"`      // 默认 "127.0.0.1:9090"
	MixedProxyAddr    string        `json:"mixed_proxy_addr"`    // 默认 "127.0.0.1:7890"
	DNSListenPort     int           `json:"dns_listen_port"`     // 默认 53
	TunInterfaceName  string        `json:"tun_interface_name"`  // 默认 "cubex-tun"

	// 2. 逃生 DNS 与 EWMA 探针目标（取代硬编码的 "223.5.5.5" 与 "gstatic.com"）
	FallbackDNS  string        `json:"fallback_dns"`  // 默认 "223.5.5.5,119.29.29.29"
	ProbeURLs    []string      `json:"probe_urls"`    // 支持主备探针 URL 列表，防止单一 gstatic 被阻断
	ProbeTimeout time.Duration `json:"probe_timeout"` // 默认 2500ms

	// 3. 硬件自适应内存配额档位（取代写死的 GOMEMLIMIT=28MiB）
	MemoryProfile   string `json:"memory_profile"`   // "ultra-light" (35MB 容器) | "performance" (物理机/高配 CT)
	ChildGoMemLimit string `json:"child_gomemlimit"` // 自动根据档位计算："28MiB" 或 "128MiB"
	ChildGoGC       string `json:"child_gogc"`       // 自动根据档位计算："40" 或 "80"
}

// GlobalRuntimeSpec holds the singleton spec
var GlobalRuntimeSpec = LoadRuntimeSpec()

// LoadRuntimeSpec 从环境变量与配置文件加载运行参数，提供零硬编码默认值
func LoadRuntimeSpec() *RuntimeSpec {
	profile := envOrDefault("CUBEX_MEMORY_PROFILE", "ultra-light")
	memLimit := "28MiB"
	goGC := "40"
	if profile == "performance" {
		memLimit = "128MiB"
		goGC = "80"
	}

	return &RuntimeSpec{
		CubeXHostIP:       envOrDefault("CUBEX_HOST_IP", "10.0.0.7"),
		ControlListenAddr: envOrDefault("CUBEX_CONTROL_ADDR", ":2026"),
		ClashAPIAddr:      envOrDefault("CUBEX_CLASH_API", "127.0.0.1:9090"),
		MixedProxyAddr:    envOrDefault("CUBEX_MIXED_ADDR", "127.0.0.1:7890"),
		DNSListenPort:     envIntOrDefault("CUBEX_DNS_PORT", 53),
		TunInterfaceName:  envOrDefault("CUBEX_TUN_NAME", "cubex-tun"),
		FallbackDNS:       envOrDefault("CUBEX_FALLBACK_DNS", "223.5.5.5,119.29.29.29"),
		ProbeURLs: []string{
			envOrDefault("CUBEX_PROBE_URL", "https://www.gstatic.com/generate_204"),
			"https://cp.cloudflare.com/generate_204",
		},
		ProbeTimeout:    2500 * time.Millisecond,
		MemoryProfile:   profile,
		ChildGoMemLimit: envOrDefault("CUBEX_CHILD_GOMEMLIMIT", memLimit),
		ChildGoGC:       envOrDefault("CUBEX_CHILD_GOGC", goGC),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOrDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}
