package dataplane

import (
	"os"
	"path/filepath"
	"strings"
)

// KernelTuneReport 记录内核性能调优与 Fake-IP 持久化状态
type KernelTuneReport struct {
	UDPBuffersOptimized bool   `json:"udp_buffers_optimized"`
	ConntrackOptimized  bool   `json:"conntrack_optimized"`
	BBRDetected         bool   `json:"bbr_detected"`
	FakeIPCachePath     string `json:"fakeip_cache_path"`
}

// OptimizeKernelForHighThroughput 自动调优 Linux 容器/宿主网络栈参数：
// 1. 放大 UDP 缓冲区至 8MB，消除 Hysteria2 / TUIC / QUIC 在千兆宽带下的丢包限速
// 2. 缩短 nf_conntrack 老化等待时间，防止局域网高并发连接或 PT 撑爆 conntrack 表
// 3. 检测宿主内核 BBR 拥塞控制
func OptimizeKernelForHighThroughput(workDir string) *KernelTuneReport {
	report := &KernelTuneReport{
		FakeIPCachePath: filepath.Join(workDir, "cache_fakeip.db"),
	}

	// 1. 调优 UDP Socket 缓冲区上限（适配千兆高速传输）
	errR := writeSysctlSafe("/proc/sys/net/core/rmem_max", "8388608")
	errW := writeSysctlSafe("/proc/sys/net/core/wmem_max", "8388608")
	if errR == nil && errW == nil {
		report.UDPBuffersOptimized = true
	}

	// 2. 调优 Conntrack 超时水位，加速回收半关闭与僵尸 UDP 会话
	_ = writeSysctlSafe("/proc/sys/net/netfilter/nf_conntrack_udp_timeout", "30")
	_ = writeSysctlSafe("/proc/sys/net/netfilter/nf_conntrack_udp_timeout_stream", "120")
	if err := writeSysctlSafe("/proc/sys/net/netfilter/nf_conntrack_tcp_timeout_time_wait", "30"); err == nil {
		report.ConntrackOptimized = true
	}

	// 3. 检测宿主内核是否已启用 BBR 拥塞控制
	cc := readSysctlSafe("/proc/sys/net/ipv4/tcp_congestion_control")
	report.BBRDetected = strings.Contains(strings.ToLower(cc), "bbr")

	return report
}

// BuildMihomoCacheConfig 生成 Mihomo 专属的 Fake-IP 与策略组选择持久化配置段
func BuildMihomoCacheConfig(workDir string) map[string]any {
	return map[string]any{
		"store-selected": true, // 记住用户在面板切换的策略组节点
		"store-fake-ip":  true, // 持久化 Fake-IP 映射，重启或切核不断连
	}
}

// BuildSingBoxCacheConfig 生成 sing-box v1.14+ 专属的实验性 cache_file 配置段
func BuildSingBoxCacheConfig(workDir string) map[string]any {
	cachePath := "singbox_cache.db"
	if workDir != "" {
		cachePath = filepath.Join(workDir, "singbox_cache.db")
	}
	return map[string]any{
		"enabled":      true,
		"path":         cachePath,
		"store_fakeip": true,
		"store_rdrc":   true,
	}
}

func writeSysctlSafe(path, val string) error {
	return os.WriteFile(path, []byte(val+"\n"), 0644)
}

func readSysctlSafe(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
