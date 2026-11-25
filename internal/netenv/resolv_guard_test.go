package netenv_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/netenv"
)

func TestAuditResolvConfContent_DetectsLoopbackAndSelfIP(t *testing.T) {
	localIPs := []string{"192.168.31.161", "fe80::1"}

	tests := []struct {
		name         string
		content      string
		gateway      string
		wantLoop     bool
		wantLooped   string
		wantFallback string
	}{
		{
			name:         "loopback_ipv4",
			content:      "nameserver 127.0.0.1\nnameserver 223.5.5.5\n",
			gateway:      "192.168.31.1",
			wantLoop:     true,
			wantLooped:   "127.0.0.1",
			wantFallback: "192.168.31.1",
		},
		{
			name:         "loopback_ipv6",
			content:      "# comment\nnameserver ::1\n",
			gateway:      "",
			wantLoop:     true,
			wantLooped:   "::1",
			wantFallback: "223.5.5.5",
		},
		{
			name:         "dhcp_renew_self_lan_ip",
			content:      "domain lan\nnameserver 192.168.31.161\n",
			gateway:      "192.168.31.1",
			wantLoop:     true,
			wantLooped:   "192.168.31.161",
			wantFallback: "192.168.31.1",
		},
		{
			name:         "gateway_equals_self_falls_back_to_223",
			content:      "nameserver 192.168.31.161\n",
			gateway:      "192.168.31.161",
			wantLoop:     true,
			wantLooped:   "192.168.31.161",
			wantFallback: "223.5.5.5",
		},
		{
			name:         "clean_external_dns",
			content:      "nameserver 223.5.5.5\nnameserver 192.168.31.1\n",
			gateway:      "192.168.31.1",
			wantLoop:     false,
			wantFallback: "192.168.31.1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			audit := netenv.AuditResolvConfContent("/etc/resolv.conf", tc.content, localIPs, tc.gateway)
			if audit.HasSelfLoop != tc.wantLoop {
				t.Fatalf("HasSelfLoop = %v, want %v (looped=%v)", audit.HasSelfLoop, tc.wantLoop, audit.LoopedAddrs)
			}
			if audit.FallbackDNS != tc.wantFallback {
				t.Fatalf("FallbackDNS = %q, want %q", audit.FallbackDNS, tc.wantFallback)
			}
			if tc.wantLoop {
				if len(audit.LoopedAddrs) == 0 || audit.LoopedAddrs[0] != tc.wantLooped {
					t.Fatalf("LoopedAddrs = %v, want %q", audit.LoopedAddrs, tc.wantLooped)
				}
				if !strings.Contains(audit.Remediation, "chattr +i") {
					t.Fatalf("Remediation should warn against chattr +i, got: %s", audit.Remediation)
				}
			}
		})
	}
}

func TestAuditResolvConf_FileRead(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "resolv.conf")
	if err := os.WriteFile(p, []byte("nameserver 192.168.31.161\n"), 0644); err != nil {
		t.Fatal(err)
	}
	audit := netenv.AuditResolvConf(p, []string{"192.168.31.161"}, "192.168.31.1")
	if !audit.Readable || !audit.HasSelfLoop || audit.FallbackDNS != "192.168.31.1" {
		t.Fatalf("unexpected file audit: %+v", audit)
	}

	missing := netenv.AuditResolvConf(filepath.Join(tmp, "nonexistent"), []string{"192.168.31.161"}, "192.168.31.1")
	if missing.Readable || missing.HasSelfLoop {
		t.Fatalf("missing file should not trigger false self-loop: %+v", missing)
	}
}
