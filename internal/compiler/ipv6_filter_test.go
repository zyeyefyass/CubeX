package compiler_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/compiler"
	"github.com/zyeyefyass/CubeX/internal/schema"
	"gopkg.in/yaml.v3"
)

func TestIsIPv6LiteralHost(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{"2403:ad80:89:4c00:15:0:2274:689b", true},
		{"[2001:db8::1]", true},
		{"fe80::1%eth0", true},
		{"[fe80::1%eth0]", true},
		{"203.0.113.10", false},
		{"::ffff:203.0.113.10", false},
		{"jp.example.com", false},
		{"", false},
	}

	for _, tc := range cases {
		if got := compiler.IsIPv6LiteralHost(tc.host); got != tc.want {
			t.Errorf("IsIPv6LiteralHost(%q) = %v, want %v", tc.host, got, tc.want)
		}
	}
}

func TestDualCore_IPv6DisabledFilterAndEmptyGroupGuard(t *testing.T) {
	buildIR := func(ipv6Enabled bool) *schema.ConfigIR {
		ir := schema.NewDefaultIR()
		ir.Network.IPv6 = ipv6Enabled
		ir.Proxies = []schema.ProxyNode{
			{
				Name:    "SelfHosted-V6-Reality",
				Type:    "vless",
				Server:  "2403:ad80:89:4c00:15:0:2274:689b",
				Port:    443,
				UUID:    "11111111-2222-3333-4444-555555555555",
				TLS:     true,
				SNI:     "www.apple.com",
				Network: "tcp",
				Flow:    "xtls-rprx-vision",
				RawConfig: map[string]any{
					"reality-opts": map[string]any{"public-key": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "short-id": "ab12cd"},
					"fingerprint":  "chrome",
				},
			},
			{
				Name:     "SelfHosted-V6-Hy2",
				Type:     "hysteria2",
				Server:   "[2001:db8::1]",
				Port:     8443,
				Password: "hy2-v6-password",
				TLS:      true,
				SNI:      "bing.com",
			},
			{
				Name:     "SelfHosted-V4-Hy2",
				Type:     "hysteria2",
				Server:   "203.0.113.10",
				Port:     8443,
				Password: "hy2-v4-password",
				TLS:      true,
				SNI:      "bing.com",
			},
			{
				Name:     "Domain-Trojan",
				Type:     "trojan",
				Server:   "jp.example.com",
				Port:     443,
				Password: "trojan-password",
				TLS:      true,
				SNI:      "jp.example.com",
			},
		}
		ir.Groups = []schema.StrategyGroup{
			{
				Name:    "Mixed-Group",
				Type:    "select",
				Proxies: []string{"SelfHosted-V6-Reality", "SelfHosted-V4-Hy2", "Domain-Trojan"},
				Now:     "SelfHosted-V6-Reality",
			},
			{
				Name:    "OnlyV6-Select",
				Type:    "select",
				Proxies: []string{"SelfHosted-V6-Reality", "SelfHosted-V6-Hy2"},
				Now:     "SelfHosted-V6-Reality",
			},
			{
				Name:     "OnlyV6-URLTest",
				Type:     "urltest",
				Proxies:  []string{"SelfHosted-V6-Reality"},
				Now:      "SelfHosted-V6-Reality",
				URL:      "https://www.gstatic.com/generate_204",
				Interval: 300,
			},
			{
				Name:     "OnlyV6-Fallback",
				Type:     "fallback",
				Proxies:  []string{"SelfHosted-V6-Hy2"},
				Now:      "SelfHosted-V6-Hy2",
				URL:      "https://www.gstatic.com/generate_204",
				Interval: 300,
			},
		}
		return ir
	}

	// 1. ipv6: false -> 过滤纯 IPv6 字面量节点 + 空组自动兜底为 DIRECT + 不修改调用方原始 IR
	irOff := buildIR(false)
	origProxyCount := len(irOff.Proxies)

	mihomoBytes, err := compiler.BuildMihomo(irOff)
	if err != nil {
		t.Fatalf("BuildMihomo(ipv6=false) failed: %v", err)
	}
	if len(irOff.Proxies) != origProxyCount {
		t.Fatalf("BuildMihomo mutated caller IR proxies: got %d, want %d", len(irOff.Proxies), origProxyCount)
	}

	var mhCfg struct {
		Proxies []struct {
			Name   string `yaml:"name"`
			Server string `yaml:"server"`
		} `yaml:"proxies"`
		ProxyGroups []struct {
			Name    string   `yaml:"name"`
			Type    string   `yaml:"type"`
			Proxies []string `yaml:"proxies"`
		} `yaml:"proxy-groups"`
	}
	if err := yaml.Unmarshal(mihomoBytes, &mhCfg); err != nil {
		t.Fatalf("failed to unmarshal Mihomo YAML: %v", err)
	}
	mhNames := make(map[string]bool)
	for _, p := range mhCfg.Proxies {
		mhNames[p.Name] = true
		if compiler.IsIPv6LiteralHost(p.Server) {
			t.Errorf("Mihomo proxies still contains IPv6 literal node %q (%s)", p.Name, p.Server)
		}
	}
	if mhNames["SelfHosted-V6-Reality"] || mhNames["SelfHosted-V6-Hy2"] {
		t.Fatalf("Mihomo must exclude pure IPv6 literal nodes when ipv6=false")
	}
	if !mhNames["SelfHosted-V4-Hy2"] || !mhNames["Domain-Trojan"] {
		t.Fatalf("Mihomo must preserve IPv4 and domain nodes when ipv6=false, got %v", mhNames)
	}
	for _, g := range mhCfg.ProxyGroups {
		if len(g.Proxies) == 0 {
			t.Fatalf("Mihomo group %q has empty proxies slice", g.Name)
		}
		for _, member := range g.Proxies {
			if member == "SelfHosted-V6-Reality" || member == "SelfHosted-V6-Hy2" {
				t.Errorf("Mihomo group %q still references filtered IPv6 node %q", g.Name, member)
			}
		}
		if strings.HasPrefix(g.Name, "OnlyV6-") {
			if len(g.Proxies) != 1 || g.Proxies[0] != "DIRECT" {
				t.Errorf("Mihomo group %q expected fallback [DIRECT], got %v", g.Name, g.Proxies)
			}
		}
	}

	sbBytes, err := compiler.BuildSingBox(irOff)
	if err != nil {
		t.Fatalf("BuildSingBox(ipv6=false) failed: %v", err)
	}
	var sbCfg struct {
		Outbounds []struct {
			Tag       string   `json:"tag"`
			Type      string   `json:"type"`
			Server    string   `json:"server"`
			Outbounds []string `json:"outbounds"`
		} `json:"outbounds"`
	}
	if err := json.Unmarshal(sbBytes, &sbCfg); err != nil {
		t.Fatalf("failed to unmarshal sing-box JSON: %v", err)
	}
	sbTags := make(map[string]bool)
	for _, ob := range sbCfg.Outbounds {
		sbTags[ob.Tag] = true
		if ob.Server != "" && compiler.IsIPv6LiteralHost(ob.Server) {
			t.Errorf("sing-box outbounds still contains IPv6 literal node %q (%s)", ob.Tag, ob.Server)
		}
		if ob.Type == "selector" || ob.Type == "urltest" {
			if len(ob.Outbounds) == 0 {
				t.Fatalf("sing-box group %q has empty outbounds slice", ob.Tag)
			}
			for _, member := range ob.Outbounds {
				if member == "SelfHosted-V6-Reality" || member == "SelfHosted-V6-Hy2" {
					t.Errorf("sing-box group %q still references filtered IPv6 node %q", ob.Tag, member)
				}
			}
			if strings.HasPrefix(ob.Tag, "OnlyV6-") {
				if len(ob.Outbounds) != 1 || ob.Outbounds[0] != "DIRECT" {
					t.Errorf("sing-box group %q expected fallback [DIRECT], got %v", ob.Tag, ob.Outbounds)
				}
			}
		}
	}
	if sbTags["SelfHosted-V6-Reality"] || sbTags["SelfHosted-V6-Hy2"] {
		t.Fatalf("sing-box must exclude pure IPv6 literal nodes when ipv6=false")
	}
	if !sbTags["SelfHosted-V4-Hy2"] || !sbTags["Domain-Trojan"] {
		t.Fatalf("sing-box must preserve IPv4 and domain nodes when ipv6=false")
	}

	// 若本地存在 sing-box.exe，直接执行真实内核语法校验（验证纯 IPv6 空组兜底 100% 合法不炸核）
	sbBin := filepath.Join("..", "..", "bin", "sing-box.exe")
	etcDir := filepath.Join("..", "..", "etc")
	if _, statErr := os.Stat(sbBin); statErr == nil {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.json")
		if wErr := os.WriteFile(cfgPath, sbBytes, 0644); wErr != nil {
			t.Fatalf("write config.json failed: %v", wErr)
		}
		cmd := exec.Command(sbBin, "check", "-c", cfgPath, "-D", etcDir)
		if out, chkErr := cmd.CombinedOutput(); chkErr != nil {
			t.Fatalf("sing-box check failed on ipv6=false filtered config: %v\n%s", chkErr, string(out))
		}
	}

	// 2. ipv6: true -> 完整保留纯 IPv6 字面量节点
	irOn := buildIR(true)
	mhOnBytes, err := compiler.BuildMihomo(irOn)
	if err != nil {
		t.Fatalf("BuildMihomo(ipv6=true) failed: %v", err)
	}
	if !strings.Contains(string(mhOnBytes), "SelfHosted-V6-Reality") || !strings.Contains(string(mhOnBytes), "SelfHosted-V6-Hy2") {
		t.Fatalf("BuildMihomo(ipv6=true) should retain pure IPv6 literal nodes")
	}
	sbOnBytes, err := compiler.BuildSingBox(irOn)
	if err != nil {
		t.Fatalf("BuildSingBox(ipv6=true) failed: %v", err)
	}
	if !strings.Contains(string(sbOnBytes), "SelfHosted-V6-Reality") || !strings.Contains(string(sbOnBytes), "SelfHosted-V6-Hy2") {
		t.Fatalf("BuildSingBox(ipv6=true) should retain pure IPv6 literal nodes")
	}
}
