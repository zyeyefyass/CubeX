package config

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/zyeyefyass/CubeX/internal/netenv"
)

// 全局系统与内核版本默认常量（单一真源兜底）
const (
	DefaultPort          = 2026
	DefaultPortStr       = "2026"
	DefaultFakeIPRange   = "198.18.0.0/15"
	DefaultFakeIPv6Range = "fc00::/18"

	DefaultSingBoxVersion = "v1.14.5"
	DefaultMihomoVersion  = "v1.19.34"
)

// MetaCoreVersions 从 meta.json 单一真源读取当前随包对齐的双核版本，缺失时安全回退默认版本。
func MetaCoreVersions(metaPaths ...string) (singboxVer, mihomoVer string) {
	singboxVer = DefaultSingBoxVersion
	mihomoVer = DefaultMihomoVersion

	candidates := append([]string{}, metaPaths...)
	candidates = append(candidates, "meta.json", "/app/meta.json", "/opt/cubex/meta.json", "/opt/CubeX/meta.json")
	for _, p := range candidates {
		if strings.TrimSpace(p) == "" {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var meta struct {
			CoreVersions struct {
				SingBox string `json:"sing_box"`
				Mihomo  string `json:"mihomo"`
			} `json:"core_versions"`
		}
		if json.Unmarshal(data, &meta) == nil {
			if v := strings.TrimSpace(meta.CoreVersions.SingBox); v != "" {
				singboxVer = v
			}
			if v := strings.TrimSpace(meta.CoreVersions.Mihomo); v != "" {
				mihomoVer = v
			}
			return singboxVer, mihomoVer
		}
	}
	return singboxVer, mihomoVer
}

// RuntimeSpec 统一收口全项目所有网络端点、探针目标与资源配额，彻底消灭硬编码
type RuntimeSpec struct {
	// 1. 网络与端口绑定（取代写死 IP 假兜底, 2026, 7890, 9090, 53, "cubex-tun"）
	CubeXHostIP       string        `json:"cubex_host_ip"`       // 动态嗅探本机主网卡 IP，零假兜底
	ControlListenAddr string        `json:"control_listen_addr"` // 默认 ":2026"
	ClashAPIAddr      string        `json:"clash_api_addr"`      // 仅环境变量 CUBEX_CLASH_API 显式覆盖时非空（已规范化为 http://host:port）；为空时由 ResolveClashAPIBase 按 IR 端口推导
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

	hostIP := envOrDefault("CUBEX_HOST_IP", "")
	if hostIP == "" {
		topo := netenv.DetectHostTopology("", "")
		hostIP = topo.LocalIP
	}

	return &RuntimeSpec{
		CubeXHostIP:       hostIP,
		ControlListenAddr: envOrDefault("CUBEX_CONTROL_ADDR", ":2026"),
		ClashAPIAddr:      NormalizeBaseURL(envOrDefault("CUBEX_CLASH_API", "")),
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

// ResolveDefaultWorkDir 统一探测 CubeX 默认工作目录（兼容 /app 容器、/opt/cubex 与历史 /opt/CubeX 安装路径）。
func ResolveDefaultWorkDir() string {
	for _, dir := range []string{"/app", "/opt/cubex", "/opt/CubeX"} {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return "/opt/cubex"
}

// NormalizeBaseURL 将 "127.0.0.1:9090" / "http://127.0.0.1:9090/" 等形式统一规范为 "http://127.0.0.1:9090"。
// 空输入返回空串。缺少 scheme 时自动补全 http://，避免 net/http 构造请求时报
// "first path segment in URL cannot contain colon" 而静默失效。
func NormalizeBaseURL(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	if !strings.Contains(addr, "://") {
		addr = "http://" + addr
	}
	return strings.TrimRight(addr, "/")
}

// ResolveClashAPIBase 返回内核 Clash API 的规范 Base URL：
// 优先使用 CUBEX_CLASH_API 显式覆盖，否则按 IR 中配置的 ClashAPIPort（缺省 9090）推导本地回环地址。
func ResolveClashAPIBase(irPort int) string {
	if GlobalRuntimeSpec != nil && GlobalRuntimeSpec.ClashAPIAddr != "" {
		return GlobalRuntimeSpec.ClashAPIAddr
	}
	if irPort <= 0 {
		irPort = 9090
	}
	return "http://127.0.0.1:" + strconv.Itoa(irPort)
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
