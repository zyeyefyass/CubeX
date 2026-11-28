package netstack

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// IPv6AddressEvent represents an IPv6 GUA address event.
type IPv6AddressEvent struct {
	IP        net.IP
	PrefixLen int
	IsDeleted bool
	Interface string
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
	// 2000::/3 -> first 3 bits are 001 -> (ip[0] & 0xe0) == 0x20
	return (ip16[0] & 0xe0) == 0x20
}

type ipv6PrefixTracker struct {
	mu             sync.Mutex
	debounceDelay  time.Duration
	debounceTimer  *time.Timer
	pendingEvent   IPv6AddressEvent
	activePrefixes map[string]map[string]struct{}
	isClosed       atomic.Bool
	onChanged      func(event IPv6AddressEvent)
}

func newIPv6PrefixTracker(debounceDelay time.Duration, onChanged func(event IPv6AddressEvent)) *ipv6PrefixTracker {
	if debounceDelay <= 0 {
		debounceDelay = 2500 * time.Millisecond
	}
	return &ipv6PrefixTracker{
		debounceDelay:  debounceDelay,
		activePrefixes: make(map[string]map[string]struct{}),
		onChanged:      onChanged,
	}
}

func ipv6PrefixKey(iface string, ip16 net.IP, prefixLen int) string {
	if prefixLen <= 0 || prefixLen > 128 {
		prefixLen = 128
	}
	mask := net.CIDRMask(prefixLen, 128)
	masked := ip16.Mask(mask)
	return fmt.Sprintf("%s|%s/%d", iface, masked.String(), prefixLen)
}

// HandleEvent tracks active IPv6 GUA prefix sets per interface and schedules a debounced
// notification only when the set of active prefixes genuinely changes.
func (t *ipv6PrefixTracker) HandleEvent(event IPv6AddressEvent) bool {
	if t == nil || t.isClosed.Load() || !IsIPv6GUA(event.IP) {
		return false
	}
	ip16 := event.IP.To16()
	if ip16 == nil {
		return false
	}
	ipStr := ip16.String()
	prefixKey := ipv6PrefixKey(event.Interface, ip16, event.PrefixLen)

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.isClosed.Load() {
		return false
	}
	if t.activePrefixes == nil {
		t.activePrefixes = make(map[string]map[string]struct{})
	}

	if !event.IsDeleted {
		if addrs, exists := t.activePrefixes[prefixKey]; exists && len(addrs) > 0 {
			// 已有活跃前缀下发生周期性 RA 续租或同前缀地址变更，更新集合但不重复触发路由刷新
			addrs[ipStr] = struct{}{}
			return false
		}
		t.activePrefixes[prefixKey] = map[string]struct{}{ipStr: {}}
	} else {
		addrs, exists := t.activePrefixes[prefixKey]
		if !exists {
			return false
		}
		delete(addrs, ipStr)
		if len(addrs) > 0 {
			return false
		}
		delete(t.activePrefixes, prefixKey)
	}

	t.pendingEvent = event
	t.scheduleLocked()
	return true
}

func (t *ipv6PrefixTracker) scheduleLocked() {
	if t.debounceTimer == nil {
		t.debounceTimer = time.AfterFunc(t.debounceDelay, t.fireDebouncedEvent)
		return
	}
	if !t.debounceTimer.Stop() {
		select {
		case <-t.debounceTimer.C:
		default:
		}
	}
	t.debounceTimer.Reset(t.debounceDelay)
}

func (t *ipv6PrefixTracker) fireDebouncedEvent() {
	t.mu.Lock()
	if t.isClosed.Load() {
		t.mu.Unlock()
		return
	}
	event := t.pendingEvent
	t.mu.Unlock()

	if t.onChanged != nil {
		t.onChanged(event)
	}
}

func (t *ipv6PrefixTracker) Close() {
	if t == nil {
		return
	}
	t.isClosed.Store(true)
	t.mu.Lock()
	if t.debounceTimer != nil {
		t.debounceTimer.Stop()
		t.debounceTimer = nil
	}
	t.mu.Unlock()
}
