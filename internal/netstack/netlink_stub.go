//go:build !linux

package netstack

import (
	"net"
)

// IPv6AddressEvent represents an IPv6 GUA address event.
type IPv6AddressEvent struct {
	IP        net.IP
	PrefixLen int
	IsDeleted bool
	Interface string
}

// IPv6PrefixWatcher is a no-op stub for non-Linux platforms (Windows/Darwin).
type IPv6PrefixWatcher struct {
}

// NewIPv6PrefixWatcher returns a no-op watcher stub on non-Linux platforms.
func NewIPv6PrefixWatcher(stopCh <-chan struct{}, onChanged func(event IPv6AddressEvent)) (*IPv6PrefixWatcher, error) {
	return &IPv6PrefixWatcher{}, nil
}

// Close is a no-op on non-Linux platforms.
func (w *IPv6PrefixWatcher) Close() {
}

// IsIPv6GUA checks whether an IPv6 address belongs to the 2000::/3 Global Unicast Address range.
func IsIPv6GUA(ip net.IP) bool {
	if ip == nil {
		return false
	}
	ip16 := ip.To16()
	if ip16 == nil || ip.To4() != nil {
		return false
	}
	return (ip16[0] & 0xe0) == 0x20
}
