package dataplane

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// FastPathSpec 定义内核态零绕路直通与双栈防漏参数
type FastPathSpec struct {
	TunInterface     string   `json:"tun_interface"`      // 默认 "cubex-tun"
	DNSListenPort    int      `json:"dns_listen_port"`    // 默认 53
	BypassMark       uint32   `json:"bypass_mark"`        // 默认 0x1ed4
	BlockIPv6AAAA    bool     `json:"block_ipv6_leak"`    // 单栈节点下自动拦截非局域网 IPv6 逃逸
	CNIPv4CIDRs      []string `json:"cn_ipv4_cidrs"`      // 中国大陆 IPv4 聚合网段
	CustomLANSubnets []string `json:"custom_lan_subnets"` // 附加内网网段
}

// SanitizeCIDRs 严格校验并过滤非法 CIDR，防止畸形输入导致 nft 语法报错整表回滚
func SanitizeCIDRs(raw []string, wantIPv6 bool) []string {
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		s := strings.TrimSpace(item)
		if s == "" {
			continue
		}
		if !strings.Contains(s, "/") {
			if wantIPv6 {
				s += "/128"
			} else {
				s += "/32"
			}
		}
		ip, ipNet, err := net.ParseCIDR(s)
		if err != nil || ip == nil || ipNet == nil {
			continue
		}
		isV6 := ip.To4() == nil
		if isV6 == wantIPv6 {
			out = append(out, ipNet.String())
		}
	}
	return out
}

// BuildAtomicNftScript 生成单次事务原子替换的 inet 双栈 nftables 脚本
func BuildAtomicNftScript(spec FastPathSpec) string {
	if spec.DNSListenPort <= 0 {
		spec.DNSListenPort = 53
	}
	if spec.BypassMark == 0 {
		spec.BypassMark = 0x1ed4
	}

	lanV4 := SanitizeCIDRs(defaultLANSubnets(spec.CustomLANSubnets), false)
	cnV4 := SanitizeCIDRs(spec.CNIPv4CIDRs, false)

	cnElemBlock := ""
	if len(cnV4) > 0 {
		cnElemBlock = "elements = { " + strings.Join(cnV4, ", ") + " }"
	}

	ipv6GuardRule := ""
	if spec.BlockIPv6AAAA {
		// 放行 IPv6 链路本地 (fe80::/10) 与唯一本地 (fc00::/7) 流量，阻断公网 IPv6 直连绕过代理与 IPv6 DoT 逃逸
		ipv6GuardRule = `
        ip6 daddr { fe80::/10, fc00::/7, ::1/128, ff00::/8 } return
        meta l4proto { tcp, udp } th dport { 53, 853 } meta nfproto ipv6 reject`
	}

	return fmt.Sprintf(`table inet cubex_symbiotic
delete table inet cubex_symbiotic

table inet cubex_symbiotic {
    set lan_v4 {
        type ipv4_addr
        flags interval
        auto-merge
        elements = { %s }
    }

    set cn_v4 {
        type ipv4_addr
        flags interval
        auto-merge
        %s
    }

    set public_doh_v4 {
        type ipv4_addr
        elements = { 1.1.1.1, 1.0.0.1, 8.8.8.8, 8.8.4.4, 9.9.9.9, 208.67.222.222 }
    }

    chain prerouting_fastpath {
        type filter hook prerouting priority mangle - 5; policy accept;

        # 1. 已携带直通标记或内网保留地址立即内核态放行，0 用户态上下文切换
        meta mark 0x%x return
        ip daddr @lan_v4 return
%s
        # 2. 拦截硬编码 DoT (853) 与海外公共 DoQ (UDP 443) 逃逸，确保 0 明文/抢答 DNS 泄露
        meta l4proto { tcp, udp } th dport 853 reject
        ip daddr @public_doh_v4 udp dport 443 reject

        # 3. 国内目标 IP 直接在内核态打标并 return 走宿主原生硬件转发（满血直通）
        ip daddr @cn_v4 meta mark set 0x%x return
    }
}
`, strings.Join(lanV4, ", "), cnElemBlock, spec.BypassMark, ipv6GuardRule, spec.BypassMark)
}

// ApplyAtomicNftRules 通过 stdin 管道单事务原子应用规则
func ApplyAtomicNftRules(ctx context.Context, spec FastPathSpec) error {
	script := BuildAtomicNftScript(spec)
	cmd := exec.CommandContext(ctx, "nft", "-f", "-")
	cmd.Stdin = bytes.NewBufferString(script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("apply atomic nft fastpath failed: %w (out: %s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// TeardownAtomicNftRules 在容器停止或进入维护逃生模式时幂等销毁 cubex_symbiotic 表，确保宿主零残留
func TeardownAtomicNftRules(ctx context.Context) error {
	// 先 add 再 delete 保证即使表不存在也不会报错退出（100% 幂等）
	script := "table inet cubex_symbiotic\ndelete table inet cubex_symbiotic\n"
	cmd := exec.CommandContext(ctx, "nft", "-f", "-")
	cmd.Stdin = bytes.NewBufferString(script)
	_ = cmd.Run()
	return nil
}

func defaultLANSubnets(custom []string) []string {
	base := []string{
		"10.0.0.0/8",
		"100.64.0.0/10",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"224.0.0.0/4",
		"255.255.255.255/32",
	}
	return append(base, custom...)
}
