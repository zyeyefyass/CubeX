package compiler_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/compiler"
	"github.com/zyeyefyass/CubeX/internal/schema"
)

func findDNSServer(t *testing.T, sbCfg map[string]any, tag string) map[string]any {
	t.Helper()
	dnsSec, _ := sbCfg["dns"].(map[string]any)
	servers, _ := dnsSec["servers"].([]any)
	for _, item := range servers {
		m, _ := item.(map[string]any)
		if m["tag"] == tag {
			return m
		}
	}
	t.Fatalf("dns server %q not found", tag)
	return nil
}

func TestSingboxDNSLocalSelfLoopImmunity(t *testing.T) {
	tmp := t.TempDir()
	localIPs := []string{"192.168.31.161"}

	cases := []struct {
		name       string
		resolvBody string
		gateway    string
		wantType   string
		wantServer string
	}{
		{
			name:       "loopback_127_falls_back_to_gateway",
			resolvBody: "nameserver 127.0.0.1\n",
			gateway:    "192.168.31.1",
			wantType:   "udp",
			wantServer: "192.168.31.1",
		},
		{
			name:       "loopback_ipv6_falls_back_to_223_without_gateway",
			resolvBody: "nameserver ::1\n",
			gateway:    "",
			wantType:   "udp",
			wantServer: "223.5.5.5",
		},
		{
			name:       "self_lan_ip_falls_back_to_gateway",
			resolvBody: "nameserver 192.168.31.161\n",
			gateway:    "192.168.31.1",
			wantType:   "udp",
			wantServer: "192.168.31.1",
		},
		{
			name:       "safe_upstream_keeps_local_type",
			resolvBody: "nameserver 223.5.5.5\n",
			gateway:    "192.168.31.1",
			wantType:   "local",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolvPath := filepath.Join(tmp, tc.name+".conf")
			if err := os.WriteFile(resolvPath, []byte(tc.resolvBody), 0644); err != nil {
				t.Fatal(err)
			}
			ir := schema.NewDefaultIR()
			data, err := compiler.BuildSingBoxWithResolvConf(ir, resolvPath, localIPs, tc.gateway)
			if err != nil {
				t.Fatalf("BuildSingBoxWithResolvConf failed: %v", err)
			}
			var sbCfg map[string]any
			if err := json.Unmarshal(data, &sbCfg); err != nil {
				t.Fatalf("invalid json: %v", err)
			}
			dnsLocal := findDNSServer(t, sbCfg, "dns-local")
			if dnsLocal["type"] != tc.wantType {
				t.Fatalf("dns-local type = %v, want %v", dnsLocal["type"], tc.wantType)
			}
			if tc.wantServer != "" {
				if dnsLocal["server"] != tc.wantServer {
					t.Fatalf("dns-local server = %v, want %v", dnsLocal["server"], tc.wantServer)
				}
				if dnsLocal["detour"] != "direct" {
					t.Fatalf("dns-local detour = %v, want direct", dnsLocal["detour"])
				}
			}
		})
	}
}

func TestSingboxDNSHostsRuleWired(t *testing.T) {
	ir := schema.NewDefaultIR()
	data, err := compiler.BuildSingBox(ir)
	if err != nil {
		t.Fatalf("BuildSingBox failed: %v", err)
	}
	var sbCfg map[string]any
	if err := json.Unmarshal(data, &sbCfg); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	dnsSec, _ := sbCfg["dns"].(map[string]any)
	rules, _ := dnsSec["rules"].([]any)

	hostsRuleIdx := -1
	localRuleIdx := -1
	fakeIPRuleIdx := -1
	matchedDomains := map[string]bool{}

	for idx, raw := range rules {
		r, _ := raw.(map[string]any)
		srv, _ := r["server"].(string)
		if srv == "dns-hosts" && hostsRuleIdx == -1 {
			hostsRuleIdx = idx
			domains, _ := r["domain"].([]any)
			for _, d := range domains {
				if ds, ok := d.(string); ok {
					matchedDomains[ds] = true
				}
			}
		}
		if srv == "dns-local" && localRuleIdx == -1 {
			localRuleIdx = idx
		}
		if srv == "dns-fakeip" && fakeIPRuleIdx == -1 {
			fakeIPRuleIdx = idx
		}
	}

	if hostsRuleIdx == -1 {
		t.Fatalf("dns-hosts server is not referenced by any dns.rules entry")
	}
	for _, required := range []string{"dns.alidns.com", "doh.pub", "dns.google", "cloudflare-dns.com"} {
		if !matchedDomains[required] {
			t.Errorf("dns-hosts rule missing required DoH bootstrap domain %q", required)
		}
	}
	if localRuleIdx != -1 && hostsRuleIdx > localRuleIdx {
		t.Errorf("dns-hosts rule (idx=%d) must appear before dns-local (idx=%d)", hostsRuleIdx, localRuleIdx)
	}
	if fakeIPRuleIdx != -1 && hostsRuleIdx > fakeIPRuleIdx {
		t.Errorf("dns-hosts rule (idx=%d) must appear before dns-fakeip (idx=%d)", hostsRuleIdx, fakeIPRuleIdx)
	}

	// Validate compiled JSON with real sing-box binary when present in ../../bin/sing-box.exe
	sbBin := filepath.Join("..", "..", "bin", "sing-box.exe")
	if _, err := os.Stat(sbBin); err == nil {
		tmp := t.TempDir()
		resolvPath := filepath.Join(tmp, "resolv.conf")
		if err := os.WriteFile(resolvPath, []byte("nameserver 192.168.31.161\n"), 0644); err != nil {
			t.Fatal(err)
		}
		loopCfg, err := compiler.BuildSingBoxWithResolvConf(schema.NewDefaultIR(), resolvPath, []string{"192.168.31.161"}, "192.168.31.1")
		if err != nil {
			t.Fatalf("BuildSingBoxWithResolvConf failed: %v", err)
		}
		cfgPath := filepath.Join(tmp, "config.json")
		if err := os.WriteFile(cfgPath, loopCfg, 0644); err != nil {
			t.Fatal(err)
		}
		etcDir := filepath.Join("..", "..", "etc")
		if out, err := exec.Command(sbBin, "check", "-c", cfgPath, "-D", etcDir).CombinedOutput(); err != nil {
			t.Fatalf("sing-box check failed: %v\noutput: %s", err, string(out))
		}
	}
}
