package netstack

import (
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestIsIPv6GUA(t *testing.T) {
	cases := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"GUA China Telecom 240e", "240e:398:502:1111::1", true},
		{"GUA 2001::1", "2001:db8::1", true},
		{"GUA 2606::", "2606:4700:4700::1111", true},
		{"GUA 3fff::", "3fff:ffff::1", true},
		{"Link-Local fe80::", "fe80::1", false},
		{"Link-Local fe80::abc", "fe80::1234:5678:9abc:def0", false},
		{"ULA fc00::", "fc00::1", false},
		{"ULA fd12::", "fd12:3456:789a::1", false},
		{"Fake-IP fc00::/18", "fc00::10", false},
		{"IPv4 Loopback", "127.0.0.1", false},
		{"IPv4 Private", "192.168.1.1", false},
		{"IPv4 Mapped", "::ffff:192.168.1.1", false},
		{"Multicast ff02::1", "ff02::1", false},
		{"Loopback ::1", "::1", false},
		{"Unspecified ::", "::", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed := net.ParseIP(tc.ip)
			if parsed == nil {
				t.Fatalf("failed to parse IP: %s", tc.ip)
			}
			res := IsIPv6GUA(parsed)
			if res != tc.expected {
				t.Errorf("IsIPv6GUA(%s) = %v, expected %v", tc.ip, res, tc.expected)
			}
		})
	}
}

func TestIPv6PrefixWatcherLifecycle(t *testing.T) {
	stopCh := make(chan struct{})
	triggered := false

	watcher, err := NewIPv6PrefixWatcher(stopCh, func(event IPv6AddressEvent) {
		triggered = true
	})
	if err != nil {
		t.Fatalf("NewIPv6PrefixWatcher failed: %v", err)
	}

	// Close immediately and ensure no panic or hang
	close(stopCh)
	watcher.Close()

	if triggered {
		t.Logf("Event unexpectedly triggered during immediate shutdown")
	}
}

func TestIPv6PrefixTracker_DualPrefixAlternatingRTMNewAddr(t *testing.T) {
	var fired atomic.Int32
	tracker := newIPv6PrefixTracker(25*time.Millisecond, func(event IPv6AddressEvent) {
		fired.Add(1)
	})
	defer tracker.Close()

	evtA := IPv6AddressEvent{
		IP:        net.ParseIP("2409:8a62:2b8c:1b60:be24:11ff:fe45:62e0"),
		PrefixLen: 64,
		IsDeleted: false,
		Interface: "eth0",
	}
	evtB := IPv6AddressEvent{
		IP:        net.ParseIP("2409:8a62:2b8c:1b61:be24:11ff:fe45:62e0"),
		PrefixLen: 64,
		IsDeleted: false,
		Interface: "eth0",
	}

	// 首次建立双前缀集合，触发一次防抖合并通知
	if !tracker.HandleEvent(evtA) {
		t.Fatalf("expected initial prefix A event to be accepted")
	}
	if !tracker.HandleEvent(evtB) {
		t.Fatalf("expected initial prefix B event to be accepted")
	}
	time.Sleep(80 * time.Millisecond)
	if got := fired.Load(); got != 1 {
		t.Fatalf("expected 1 debounced callback for initial dual-prefix setup, got %d", got)
	}

	// 模拟双前缀周期性 RA 续租交替上报 RTM_NEWADDR，绝不应再触发路由重刷
	for round := 0; round < 10; round++ {
		if tracker.HandleEvent(evtA) {
			t.Fatalf("round %d: alternating prefix A RTM_NEWADDR must be deduplicated", round)
		}
		if tracker.HandleEvent(evtB) {
			t.Fatalf("round %d: alternating prefix B RTM_NEWADDR must be deduplicated", round)
		}
	}
	time.Sleep(60 * time.Millisecond)
	if got := fired.Load(); got != 1 {
		t.Fatalf("expected callback count to remain 1 during dual-prefix alternating renewals, got %d", got)
	}

	// 同前缀 A 新增临时隐私地址并随后删除该隐私地址（前缀 A 下仍有主地址存活），均不应触发
	evtAPrivacyAdd := IPv6AddressEvent{
		IP:        net.ParseIP("2409:8a62:2b8c:1b60::9999"),
		PrefixLen: 64,
		IsDeleted: false,
		Interface: "eth0",
	}
	evtAPrivacyDel := IPv6AddressEvent{
		IP:        net.ParseIP("2409:8a62:2b8c:1b60::9999"),
		PrefixLen: 64,
		IsDeleted: true,
		Interface: "eth0",
	}
	if tracker.HandleEvent(evtAPrivacyAdd) {
		t.Fatalf("same-prefix privacy address addition should not trigger prefix shift")
	}
	if tracker.HandleEvent(evtAPrivacyDel) {
		t.Fatalf("same-prefix privacy address deletion while primary exists should not trigger prefix shift")
	}
	time.Sleep(60 * time.Millisecond)
	if got := fired.Load(); got != 1 {
		t.Fatalf("expected callback count to remain 1 for same-prefix address churn, got %d", got)
	}

	// 真正出现全新前缀 C，应触发一次通知
	evtC := IPv6AddressEvent{
		IP:        net.ParseIP("2409:8a62:2b8c:1b62:be24:11ff:fe45:62e0"),
		PrefixLen: 64,
		IsDeleted: false,
		Interface: "eth0",
	}
	if !tracker.HandleEvent(evtC) {
		t.Fatalf("expected genuinely new prefix C to trigger notification")
	}
	time.Sleep(60 * time.Millisecond)
	if got := fired.Load(); got != 2 {
		t.Fatalf("expected callback count = 2 after genuinely new prefix C, got %d", got)
	}

	// 真正删除前缀 A 的最后一个地址，应触发一次通知；重复删除应被忽略
	evtADel := evtA
	evtADel.IsDeleted = true
	if !tracker.HandleEvent(evtADel) {
		t.Fatalf("expected deletion of last address in prefix A to trigger notification")
	}
	if tracker.HandleEvent(evtADel) {
		t.Fatalf("duplicate deletion of already removed prefix A should be ignored")
	}
	time.Sleep(60 * time.Millisecond)
	if got := fired.Load(); got != 3 {
		t.Fatalf("expected callback count = 3 after prefix A deletion, got %d", got)
	}
}
