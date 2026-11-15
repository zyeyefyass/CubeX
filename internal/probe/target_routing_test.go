package probe

import (
	"fmt"
	"testing"
)

func TestNormalizeTarget(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"rr1---sn-4g5lzney.googlevideo.com", "googlevideo.com"},
		{"api.telegram.org:443", "telegram.org"},
		{"149.154.167.99:443", "149.154.167.0/24"},
		{"news.sina.com.cn", "sina.com.cn"},
		{"www.google.co.jp", "google.co.jp"},
		{"10.0.0.1", "10.0.0.0/24"},
	}

	for _, c := range cases {
		actual := NormalizeTarget(c.input)
		if actual != c.expected {
			t.Errorf("NormalizeTarget(%q) = %q, expected %q", c.input, actual, c.expected)
		}
	}
}

func TestTargetAwareRoutingEngine(t *testing.T) {
	router := GetTargetAwareRouter()

	target := "rr2---sn-test.googlevideo.com"
	defaultNode := "HK-Default-Node"
	backupNode := "SG-Fast-Node"

	// Initial route -> should be default
	node, overridden := router.RouteTarget(target, defaultNode)
	if overridden || node != "" {
		t.Fatalf("expected no override initially, got %s", node)
	}

	// Record backupNode has fast TTFB
	router.RecordConnectionOutcome(target, defaultNode, backupNode, 110, true)

	// Simulate 2 consecutive failures on defaultNode
	router.RecordConnectionOutcome(target, defaultNode, defaultNode, 0, false)
	router.RecordConnectionOutcome(target, defaultNode, defaultNode, 0, false)

	// Now routeTarget should failover to backupNode!
	node, overridden = router.RouteTarget(target, defaultNode)
	if !overridden || node != backupNode {
		t.Fatalf("expected failover to %s, got %s (overridden=%v)", backupNode, node, overridden)
	}

	// Test 512 capacity limit
	for i := 0; i < 600; i++ {
		router.RecordConnectionOutcome(fmt.Sprintf("sub%d.example%d.com", i, i), defaultNode, defaultNode, 80, true)
	}

	router.mu.RLock()
	cnt := router.lruList.Len()
	mapLen := len(router.entries)
	router.mu.RUnlock()

	if cnt > 512 || mapLen > 512 {
		t.Fatalf("LRU capacity exceeded 512: list=%d, map=%d", cnt, mapLen)
	}
}
