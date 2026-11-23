package netstack

import (
	"net"
	"testing"
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
