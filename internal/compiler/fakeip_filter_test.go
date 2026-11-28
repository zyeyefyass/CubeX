package compiler_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/compiler"
	"github.com/zyeyefyass/CubeX/internal/schema"
	"gopkg.in/yaml.v3"
)

func TestSchemaBuiltinFakeIPListAndNormalization(t *testing.T) {
	if cnt := schema.BuiltinFakeIPFilterCount(); cnt < 32 {
		t.Fatalf("expected >= 32 builtin fake-ip filter entries, got %d", cnt)
	}

	norm, err := schema.NormalizeFakeIPDomains([]string{" Example.COM ", "+.Bank.CN.", "*.PT.Tracker.org", "example.com"})
	if err != nil {
		t.Fatalf("unexpected error normalizing valid domains: %v", err)
	}
	if len(norm) != 3 {
		t.Fatalf("expected 3 deduplicated domains, got %v", norm)
	}
	if norm[0] != "example.com" || norm[1] != "+.bank.cn" || norm[2] != "*.pt.tracker.org" {
		t.Fatalf("unexpected normalized domains: %v", norm)
	}

	for _, bad := range []string{"bad domain.com", "https://evil.com/path", "foo..bar", "singlelabel"} {
		if _, err := schema.NormalizeFakeIPDomain(bad); err == nil {
			t.Errorf("expected error for invalid domain %q, got nil", bad)
		}
	}
}

func TestDualCoreFakeIPBuiltinToggleAndCustomMerge(t *testing.T) {
	checkDomainPresence := func(builtinEnabled bool) {
		ir := schema.NewDefaultIR()
		ir.DNS.SetFakeIPBuiltinEnabled(builtinEnabled)
		ir.DNS.FakeIPFilter = []string{"custom-nas.home.arpa", "+.my-pt-site.org"}

		mhBytes, err := compiler.BuildMihomo(ir)
		if err != nil {
			t.Fatalf("BuildMihomo(builtin=%v) failed: %v", builtinEnabled, err)
		}
		var mhCfg struct {
			DNS struct {
				FakeIPFilter []string `yaml:"fake-ip-filter"`
			} `yaml:"dns"`
		}
		if err := yaml.Unmarshal(mhBytes, &mhCfg); err != nil {
			t.Fatalf("unmarshal mihomo yaml failed: %v", err)
		}

		mhSet := make(map[string]bool, len(mhCfg.DNS.FakeIPFilter))
		for _, f := range mhCfg.DNS.FakeIPFilter {
			mhSet[f] = true
		}
		if !mhSet["custom-nas.home.arpa"] || !mhSet["+.my-pt-site.org"] {
			t.Fatalf("Mihomo missing custom domains when builtin=%v: %v", builtinEnabled, mhCfg.DNS.FakeIPFilter)
		}
		if mhSet["+.icbc.com.cn"] != builtinEnabled {
			t.Fatalf("Mihomo +.icbc.com.cn presence = %v, want %v", mhSet["+.icbc.com.cn"], builtinEnabled)
		}
		if mhSet["dlg.io.mi.com"] != builtinEnabled {
			t.Fatalf("Mihomo dlg.io.mi.com presence = %v, want %v", mhSet["dlg.io.mi.com"], builtinEnabled)
		}

		sbBytes, err := compiler.BuildSingBox(ir)
		if err != nil {
			t.Fatalf("BuildSingBox(builtin=%v) failed: %v", builtinEnabled, err)
		}
		var sbCfg struct {
			DNS struct {
				Rules []map[string]any `json:"rules"`
			} `json:"dns"`
		}
		if err := json.Unmarshal(sbBytes, &sbCfg); err != nil {
			t.Fatalf("unmarshal singbox json failed: %v", err)
		}

		sbSuffixSet := make(map[string]bool)
		for _, rule := range sbCfg.DNS.Rules {
			if rule["server"] != "dns-direct" {
				continue
			}
			if arr, ok := rule["domain_suffix"].([]any); ok {
				for _, item := range arr {
					if s, ok := item.(string); ok {
						sbSuffixSet[s] = true
					}
				}
			}
		}
		if !sbSuffixSet["custom-nas.home.arpa"] || !sbSuffixSet["my-pt-site.org"] {
			t.Fatalf("sing-box missing custom domains when builtin=%v", builtinEnabled)
		}
		if sbSuffixSet["icbc.com.cn"] != builtinEnabled {
			t.Fatalf("sing-box icbc.com.cn presence = %v, want %v", sbSuffixSet["icbc.com.cn"], builtinEnabled)
		}

		sbBin := filepath.Join("..", "..", "bin", "sing-box.exe")
		etcDir := filepath.Join("..", "..", "etc")
		if _, statErr := os.Stat(sbBin); statErr == nil {
			tmpDir := t.TempDir()
			cfgPath := filepath.Join(tmpDir, "config.json")
			if err := os.WriteFile(cfgPath, sbBytes, 0644); err != nil {
				t.Fatal(err)
			}
			if out, chkErr := exec.Command(sbBin, "check", "-c", cfgPath, "-D", etcDir).CombinedOutput(); chkErr != nil {
				t.Fatalf("sing-box check failed (builtin=%v): %v\n%s", builtinEnabled, chkErr, string(out))
			}
		}
	}

	checkDomainPresence(true)
	checkDomainPresence(false)
}
