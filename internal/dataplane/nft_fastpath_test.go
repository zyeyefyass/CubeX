package dataplane

import (
	"context"
	"strings"
	"testing"
)

func TestSanitizeCIDRs(t *testing.T) {
	raw := []string{
		"10.0.0.1",           // 裸 IP 应自动补 /32
		"192.168.1.0/24",     // 合法 IPv4 CIDR
		"invalid-cidr",       // 畸形字符串应过滤
		"2001:db8::1",        // IPv6 应在 wantIPv6=false 时被过滤
		"fe80::/10",          // IPv6
		"",                   // 空串应过滤
		"  172.16.0.0/12  ",  // 带前后空格应 trim
	}

	v4 := SanitizeCIDRs(raw, false)
	if len(v4) != 3 {
		t.Fatalf("expected 3 v4 CIDRs, got %d: %v", len(v4), v4)
	}
	if v4[0] != "10.0.0.1/32" || v4[1] != "192.168.1.0/24" || v4[2] != "172.16.0.0/12" {
		t.Errorf("unexpected v4 results: %v", v4)
	}

	v6 := SanitizeCIDRs(raw, true)
	if len(v6) != 2 {
		t.Fatalf("expected 2 v6 CIDRs, got %d: %v", len(v6), v6)
	}
	if v6[0] != "2001:db8::1/128" || v6[1] != "fe80::/10" {
		t.Errorf("unexpected v6 results: %v", v6)
	}
}

func TestBuildAtomicNftScriptWithIPv6Guard(t *testing.T) {
	spec := FastPathSpec{
		TunInterface:     "cubex-tun",
		DNSListenPort:    53,
		BypassMark:       0x1ed4,
		BlockIPv6AAAA:    true,
		CNIPv4CIDRs:      []string{"1.0.1.0/24", "1.0.2.0/23"},
		CustomLANSubnets: []string{"192.168.1.0/24"},
	}

	script := BuildAtomicNftScript(spec)

	// 1. 验证表名与原子重置
	if !strings.Contains(script, "table inet cubex_symbiotic") {
		t.Errorf("script missing table declaration: %s", script)
	}
	if !strings.Contains(script, "delete table inet cubex_symbiotic") {
		t.Errorf("script missing atomic delete declaration")
	}

	// 2. 验证国内网段集合与内核态 return
	if !strings.Contains(script, "1.0.1.0/24") || !strings.Contains(script, "1.0.2.0/23") {
		t.Errorf("script missing CN CIDRs")
	}
	if !strings.Contains(script, "ip daddr @cn_v4 meta mark set 0x1ed4 return") {
		t.Errorf("script missing kernel-space CN fastpath return rule")
	}

	// 3. 验证 853 DoT 逃逸拦截与 Public DoH 拦截
	if !strings.Contains(script, "meta l4proto { tcp, udp } th dport 853 reject") {
		t.Errorf("script missing DoT 853 reject rule")
	}
	if !strings.Contains(script, "ip daddr @public_doh_v4 udp dport 443 reject") {
		t.Errorf("script missing Public DoH 443 UDP reject rule")
	}

	// 4. 验证 IPv6 抢答与防漏规则
	if !strings.Contains(script, "ip6 daddr { fe80::/10, fc00::/7, ::1/128, ff00::/8 } return") {
		t.Errorf("script missing IPv6 local return rule")
	}
	if !strings.Contains(script, "meta nfproto ipv6 reject") {
		t.Errorf("script missing IPv6 53/853 reject rule")
	}
}

func TestTeardownAtomicNftRules(t *testing.T) {
	// 验证在无 nft 环境下 Teardown 不会 panic，保持幂等
	_ = TeardownAtomicNftRules(context.Background())
}
