package compiler_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/compiler"
	"github.com/zyeyefyass/CubeX/internal/config"
	"github.com/zyeyefyass/CubeX/internal/schema"
)

func buildRichSampleNodes() []schema.ProxyNode {
	return []schema.ProxyNode{
		{
			Name:    "HK-Reality-01",
			Type:    "vless",
			Server:  "203.0.113.10",
			Port:    443,
			UUID:    "11111111-2222-3333-4444-555555555555",
			TLS:     true,
			SNI:     "www.apple.com",
			Network: "tcp",
			Flow:    "xtls-rprx-vision",
			RawConfig: map[string]any{
				"reality-opts": map[string]any{"public-key": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "short-id": "ab12cd"},
				"fingerprint":  "chrome",
				"spider_x":     "/legacy-spider",
			},
		},
		{
			Name:     "US-Hy2-Hop-02",
			Type:     "hysteria2",
			Server:   "203.0.113.20",
			Port:     443,
			Password: "hy2-secret-password",
			TLS:      true,
			SNI:      "bing.com",
			RawConfig: map[string]any{
				"ports":    "20000-30000",
				"up":       "50 Mbps",
				"down":     "200 Mbps",
				"obfs":     "salamander",
				"obfs-pwd": "obfs-pass",
			},
		},
		{
			Name:     "JP-SS2022-03",
			Type:     "ss",
			Server:   "203.0.113.30",
			Port:     8388,
			Cipher:   "2022-blake3-aes-128-gcm",
			Password: "dGVzdC1rZXktMTIzNDU2Nw==",
		},
		{
			Name:     "SG-Trojan-WS-04",
			Type:     "trojan",
			Server:   "sg.example.com",
			Port:     443,
			Password: "trojan-password",
			TLS:      true,
			SNI:      "sg.example.com",
			Network:  "ws",
			Path:     "/ws-path",
		},
		{
			Name:     "TW-TUIC-05",
			Type:     "tuic",
			Server:   "tw.example.com",
			Port:     8443,
			UUID:     "22222222-3333-4444-5555-666666666666",
			Password: "tuic-password",
			TLS:      true,
			SNI:      "tw.example.com",
		},
		{
			Name:     "KR-Snell-v4-06",
			Type:     "snell",
			Server:   "203.0.113.40",
			Port:     6160,
			Password: "snell-psk-secret",
			RawConfig: map[string]any{
				"version": 4,
			},
		},
	}
}

func TestSingboxV1146_TwelveScenarioCheckAndBackCompat(t *testing.T) {
	if got, _ := config.MetaCoreVersions(filepath.Join("..", "..", "meta.json")); got != "v1.14.6" {
		t.Fatalf("MetaCoreVersions().sing_box = %q, want v1.14.6", got)
	}
	if config.DefaultSingBoxVersion != "v1.14.6" {
		t.Fatalf("DefaultSingBoxVersion = %q, want v1.14.6", config.DefaultSingBoxVersion)
	}

	sbBin := filepath.Join("..", "..", "bin", "sing-box.exe")
	if _, err := os.Stat(sbBin); err != nil {
		t.Skipf("sing-box.exe not found at %s: %v", sbBin, err)
	}
	etcDir := filepath.Join("..", "..", "etc")

	scenarios := []struct {
		name  string
		setup func() *schema.ConfigIR
	}{
		{"01_default_clean_ir", func() *schema.ConfigIR { return schema.NewDefaultIR() }},
		{"02_rule_mode_with_nodes", func() *schema.ConfigIR {
			ir := schema.NewDefaultIR()
			ir.Proxies = buildRichSampleNodes()
			schema.SyncAllStrategyGroups(ir)
			return ir
		}},
		{"03_global_mode_with_nodes", func() *schema.ConfigIR {
			ir := schema.NewDefaultIR()
			ir.Meta.ProxyMode = schema.ModeGlobal
			ir.Proxies = buildRichSampleNodes()
			schema.SyncAllStrategyGroups(ir)
			return ir
		}},
		{"04_direct_mode_with_nodes", func() *schema.ConfigIR {
			ir := schema.NewDefaultIR()
			ir.Meta.ProxyMode = schema.ModeDirect
			ir.Proxies = buildRichSampleNodes()
			schema.SyncAllStrategyGroups(ir)
			return ir
		}},
		{"05_proxy_paused_passthrough", func() *schema.ConfigIR {
			ir := schema.NewDefaultIR()
			ir.Meta.ProxyPaused = true
			ir.Meta.ProxyMode = schema.ModeDirect
			ir.Proxies = buildRichSampleNodes()
			schema.SyncAllStrategyGroups(ir)
			return ir
		}},
		{"06_fakeip_disabled_realip", func() *schema.ConfigIR {
			ir := schema.NewDefaultIR()
			ir.DNS.EnableFakeIP = false
			ir.Proxies = buildRichSampleNodes()
			schema.SyncAllStrategyGroups(ir)
			return ir
		}},
		{"07_ipv6_disabled", func() *schema.ConfigIR {
			ir := schema.NewDefaultIR()
			ir.Network.IPv6 = false
			ir.Proxies = buildRichSampleNodes()
			schema.SyncAllStrategyGroups(ir)
			return ir
		}},
		{"08_quic_block_and_device_policies", func() *schema.ConfigIR {
			ir := schema.NewDefaultIR()
			ir.Network.QUICPolicy = "block"
			ir.Proxies = buildRichSampleNodes()
			ir.DevicePolicies = []schema.DevicePolicy{
				{Name: "NAS", IP: "192.168.50.10", Mode: "bypass"},
				{Name: "Media", IP: "192.168.50.11", Mode: "media"},
				{Name: "AppleTV", IP: "192.168.50.12", Mode: "node", BoundNode: "HK-Reality-01"},
			}
			schema.SyncAllStrategyGroups(ir)
			return ir
		}},
		{"09_quic_smart_and_route_all_traffic", func() *schema.ConfigIR {
			ir := schema.NewDefaultIR()
			ir.Network.QUICPolicy = "smart"
			ir.Network.RouteAllTraffic = true
			ir.Proxies = buildRichSampleNodes()
			schema.SyncAllStrategyGroups(ir)
			return ir
		}},
		{"10_dns_rewrite_and_user_overrides", func() *schema.ConfigIR {
			ir := schema.NewDefaultIR()
			ir.DNSRewriteEnabled = true
			ir.DNSRewrites = []schema.DNSRewriteRule{
				{ID: "r1", Domain: "nas.home.arpa", Target: "192.168.50.10", Type: schema.DNSRewriteTypeA, Enabled: true},
				{ID: "r2", Domain: "alias.home.arpa", Target: "nas.home.arpa", Type: schema.DNSRewriteTypeCNAME, Enabled: true},
			}
			ir.UserOverrides = &schema.UserOverrideRules{
				ForceDirectDomains: []string{"internal.corp"},
				ForceProxyDomains:  []string{"api.custom-ai.example"},
				ForceRejectDomains: []string{"telemetry.bad.example"},
			}
			ir.Proxies = buildRichSampleNodes()
			schema.SyncAllStrategyGroups(ir)
			return ir
		}},
		{"11_gvisor_stack_adblock_enabled", func() *schema.ConfigIR {
			ir := schema.NewDefaultIR()
			ir.Network.TunStack = "gvisor"
			ir.AdBlockEnabled = true
			ir.Proxies = buildRichSampleNodes()
			schema.SyncAllStrategyGroups(ir)
			return ir
		}},
		{"12_loopback_resolv_and_legacy_kernel", func() *schema.ConfigIR {
			t.Setenv("CUBEX_KERNEL_RELEASE", "4.4.60+")
			ir := schema.NewDefaultIR()
			ir.Proxies = buildRichSampleNodes()
			schema.SyncAllStrategyGroups(ir)
			return ir
		}},
	}

	tmp := t.TempDir()
	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			ir := sc.setup()
			var data []byte
			var err error
			if sc.name == "12_loopback_resolv_and_legacy_kernel" {
				resolv := filepath.Join(tmp, "resolv.conf")
				_ = os.WriteFile(resolv, []byte("nameserver 127.0.0.1\n"), 0644)
				data, err = compiler.BuildSingBoxWithResolvConf(ir, resolv, []string{"192.168.50.2"}, "192.168.50.1")
			} else {
				data, err = compiler.BuildSingBox(ir)
			}
			if err != nil {
				t.Fatalf("BuildSingBox failed: %v", err)
			}

			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("invalid json: %v", err)
			}

			// Guard backward compatibility with older running v1.14.4 / v1.14.5 cores.
			rawStr := string(data)
			for _, forbiddenKey := range []string{"max_header_size", "\"spider_x\""} {
				if strings.Contains(rawStr, forbiddenKey) {
					t.Errorf("generated sing-box config contains forbidden key %s", forbiddenKey)
				}
			}

			cfgPath := filepath.Join(tmp, sc.name+".json")
			if err := os.WriteFile(cfgPath, data, 0644); err != nil {
				t.Fatal(err)
			}
			out, err := exec.Command(sbBin, "check", "-c", cfgPath, "-D", etcDir).CombinedOutput()
			outStr := string(out)
			if err != nil {
				t.Fatalf("sing-box v1.14.6 check failed: %v\noutput: %s", err, outStr)
			}
			low := strings.ToLower(outStr)
			for _, bad := range []string{"warn", "error", "deprecated"} {
				if strings.Contains(low, bad) {
					t.Errorf("sing-box v1.14.6 check emitted %q in output:\n%s", bad, outStr)
				}
			}
		})
	}
}
