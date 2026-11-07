package affinity

import (
	"path/filepath"
	"testing"
)

func TestSessionAffinityManager(t *testing.T) {
	tmpDir := t.TempDir()
	persistPath := filepath.Join(tmpDir, "session_affinity_test.json")

	mgr := NewSessionAffinityManager(persistPath)
	if !mgr.enabled {
		t.Errorf("expected default enabled=true")
	}

	proxies := []string{"HK-01", "SG-01", "JP-01", "US-01"}

	// 1. 测试确定性一致哈希粘滞
	p1 := mgr.ResolveProxy("192.168.1.100", "api.github.com", proxies)
	if p1 == "" {
		t.Fatalf("expected non-empty proxy")
	}

	// 相同 client + dest 应 100% 映射到相同节点
	for i := 0; i < 50; i++ {
		p2 := mgr.ResolveProxy("192.168.1.100", "api.github.com", proxies)
		if p2 != p1 {
			t.Fatalf("session affinity broken! expected %s, got %s on iteration %d", p1, p2, i)
		}
	}

	status := mgr.GetStatus()
	if status.ActiveBindingsCount != 1 {
		t.Errorf("expected 1 active binding, got %d", status.ActiveBindingsCount)
	}
	if status.TotalRouted != 51 {
		t.Errorf("expected 51 total routed, got %d", status.TotalRouted)
	}

	// 2. 模拟节点宕机，平滑故障转移
	survivingProxies := []string{"SG-01", "JP-01", "US-01"}
	if p1 == "HK-01" {
		migrated := mgr.TriggerNodeFailure("HK-01", survivingProxies)
		if migrated != 1 {
			t.Errorf("expected 1 migrated binding, got %d", migrated)
		}
		newProxy := mgr.ResolveProxy("192.168.1.100", "api.github.com", survivingProxies)
		if newProxy == "HK-01" {
			t.Errorf("expected new proxy not to be failed HK-01")
		}
	}

	// 3. 停用测试
	mgr.SetEnabled(false)
	pDisabled := mgr.ResolveProxy("10.0.0.99", "example.com", proxies)
	if pDisabled != proxies[0] {
		t.Errorf("when disabled, expected fallback to first proxy %s, got %s", proxies[0], pDisabled)
	}

	// 4. 重置测试
	mgr.ResetBindings()
	s2 := mgr.GetStatus()
	if s2.ActiveBindingsCount != 0 {
		t.Errorf("expected 0 active bindings after reset, got %d", s2.ActiveBindingsCount)
	}
}

func TestConsistentHashDeterminism(t *testing.T) {
	proxies := []string{"Node-A", "Node-B", "Node-C", "Node-D"}
	res1 := ConsistentHash("10.0.0.10|youtube.com", proxies)
	res2 := ConsistentHash("10.0.0.10|youtube.com", proxies)
	if res1 != res2 {
		t.Fatalf("ConsistentHash is non-deterministic: %s != %s", res1, res2)
	}
	resDifferent := ConsistentHash("10.0.0.11|bilibili.com", proxies)
	t.Logf("Key1 -> %s, Key2 -> %s", res1, resDifferent)
}
