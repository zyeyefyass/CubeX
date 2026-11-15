package probe

import (
	"container/list"
	"net"
	"strings"
	"sync"
	"time"
)

// TargetRoutingEntry stores per-target failover & latency telemetry in an LRU slot.
type TargetRoutingEntry struct {
	TargetKey        string    `json:"target_key"`        // Normalized eTLD+1 or /24 IP
	DefaultNode      string    `json:"default_node"`      // Currently configured active group node
	PreferredNode    string    `json:"preferred_node"`    // Dynamically chosen fastest node
	BestTTFBMs       uint16    `json:"best_ttfb_ms"`      // Best time-to-first-byte
	ConsecutiveFails uint8     `json:"consecutive_fails"` // Failures observed on DefaultNode
	SuccessCount     uint32    `json:"success_count"`
	TotalConnections uint32    `json:"total_connections"`
	LastSwitchedAt   int64     `json:"last_switched_at"`
	ExpiresAt        int64     `json:"expires_at"` // Unix sec when preferred override expires
}

// TargetAwareRoutingEngine implements a fixed 512-capacity LRU target-aware smart failover router.
// Total heap memory footprint is strictly capped < 256KB.
type TargetAwareRoutingEngine struct {
	mu          sync.RWMutex
	capacity    int
	lruList     *list.List
	entries     map[string]*list.Element
	defaultTTL  time.Duration // Duration before evaluating default node again (default: 10m)
	ttfbCutoff  uint16        // TTFB threshold in ms triggering failover (default: 350ms)
	failCutoff  uint8         // Consecutive failures triggering failover (default: 2)
}

var (
	globalTargetRouterOnce sync.Once
	globalTargetRouter     *TargetAwareRoutingEngine
)

// NewTargetAwareRoutingEngine creates a new instance with the specified LRU capacity.
func NewTargetAwareRoutingEngine(capacity int) *TargetAwareRoutingEngine {
	if capacity <= 0 {
		capacity = 512
	}
	return &TargetAwareRoutingEngine{
		capacity:   capacity,
		lruList:    list.New(),
		entries:    make(map[string]*list.Element, capacity),
		defaultTTL: 10 * time.Minute,
		ttfbCutoff: 350,
		failCutoff: 2,
	}
}

// GetTargetAwareRouter returns the singleton instance of TargetAwareRoutingEngine.
func GetTargetAwareRouter() *TargetAwareRoutingEngine {
	globalTargetRouterOnce.Do(func() {
		globalTargetRouter = &TargetAwareRoutingEngine{
			capacity:   512,
			lruList:    list.New(),
			entries:    make(map[string]*list.Element, 512),
			defaultTTL: 10 * time.Minute,
			ttfbCutoff: 350,
			failCutoff: 2,
		}
	})
	return globalTargetRouter
}

// NormalizeTarget extracts the apex/eTLD+1 domain or /24 IPv4 block from a raw host string.
// Examples:
// - "rr1---sn-4g5lzney.googlevideo.com" -> "googlevideo.com"
// - "api.telegram.org" -> "telegram.org"
// - "149.154.167.99" -> "149.154.167.0/24"
// - "149.154.167.99:443" -> "149.154.167.0/24"
func NormalizeTarget(rawHost string) string {
	rawHost = strings.TrimSpace(rawHost)
	if rawHost == "" {
		return ""
	}

	// Strip port if present
	host := rawHost
	if h, _, err := net.SplitHostPort(rawHost); err == nil {
		host = h
	}

	// Check if host is IPv4
	ip := net.ParseIP(host)
	if ip != nil {
		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4.Mask(net.CIDRMask(24, 32)).String() + "/24"
		}
		// IPv6 /48
		return ip.Mask(net.CIDRMask(48, 128)).String() + "/48"
	}

	// Host is domain: normalize to lowercase
	host = strings.ToLower(host)
	parts := strings.Split(host, ".")
	if len(parts) <= 2 {
		return host
	}

	// Heuristic eTLD+1 extraction
	last2 := parts[len(parts)-2] + "." + parts[len(parts)-1]
	// Handle common two-part TLDs: .com.cn, .org.cn, .co.uk, .com.hk
	twoPartTLDs := map[string]bool{
		"com.cn": true, "net.cn": true, "org.cn": true, "gov.cn": true,
		"co.uk": true, "org.uk": true, "com.hk": true, "com.tw": true,
		"co.jp": true, "com.sg": true,
	}
	if twoPartTLDs[last2] && len(parts) >= 3 {
		return parts[len(parts)-3] + "." + last2
	}

	// Standard eTLD+1
	return parts[len(parts)-2] + "." + parts[len(parts)-1]
}

// RecordConnectionOutcome records the latency (TTFB) or failure of a connection to target.
func (r *TargetAwareRoutingEngine) RecordConnectionOutcome(rawHost, defaultNode, activeNode string, ttfbMs uint16, success bool) {
	key := NormalizeTarget(rawHost)
	if key == "" || defaultNode == "" {
		return
	}

	nowSec := time.Now().Unix()

	r.mu.Lock()
	defer r.mu.Unlock()

	elem, exists := r.entries[key]
	var entry *TargetRoutingEntry

	if exists {
		r.lruList.MoveToFront(elem)
		entry = elem.Value.(*TargetRoutingEntry)
	} else {
		// Evict oldest if capacity reached
		if r.lruList.Len() >= r.capacity {
			oldest := r.lruList.Back()
			if oldest != nil {
				r.lruList.Remove(oldest)
				oldEntry := oldest.Value.(*TargetRoutingEntry)
				delete(r.entries, oldEntry.TargetKey)
			}
		}
		entry = &TargetRoutingEntry{
			TargetKey:   key,
			DefaultNode: defaultNode,
		}
		elem = r.lruList.PushFront(entry)
		r.entries[key] = elem
	}

	entry.TotalConnections++
	entry.DefaultNode = defaultNode

	if !success {
		if activeNode == defaultNode {
			entry.ConsecutiveFails++
		}
	} else {
		entry.SuccessCount++
		// If connected successfully via an alternative node or default node with fast TTFB
		if entry.ConsecutiveFails > 0 && activeNode == defaultNode && ttfbMs < r.ttfbCutoff {
			entry.ConsecutiveFails = 0
		}
		if entry.BestTTFBMs == 0 || ttfbMs < entry.BestTTFBMs {
			entry.BestTTFBMs = ttfbMs
			entry.PreferredNode = activeNode
		}
	}

	// Trigger smart failover if default node failed >= cutoff times or TTFB exceeded threshold
	if (entry.ConsecutiveFails >= r.failCutoff || (activeNode == defaultNode && ttfbMs > r.ttfbCutoff)) && entry.PreferredNode != "" && entry.PreferredNode != defaultNode {
		entry.ExpiresAt = nowSec + int64(r.defaultTTL.Seconds())
		entry.LastSwitchedAt = nowSec
	}
}

// RouteTarget returns the preferred proxy node for the given target, or "" if default should be used.
func (r *TargetAwareRoutingEngine) RouteTarget(rawHost, defaultNode string) (preferredNode string, isOverridden bool) {
	key := NormalizeTarget(rawHost)
	if key == "" {
		return "", false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	elem, exists := r.entries[key]
	if !exists {
		return "", false
	}

	entry := elem.Value.(*TargetRoutingEntry)
	nowSec := time.Now().Unix()

	// Check if override is active and not expired
	if entry.PreferredNode != "" && entry.PreferredNode != defaultNode && entry.ExpiresAt > nowSec {
		return entry.PreferredNode, true
	}

	return "", false
}

// GetSnapshot returns active target routing entries for UI inspection.
func (r *TargetAwareRoutingEngine) GetSnapshot() []TargetRoutingEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nowSec := time.Now().Unix()
	res := make([]TargetRoutingEntry, 0, len(r.entries))
	for _, elem := range r.entries {
		entry := elem.Value.(*TargetRoutingEntry)
		cp := *entry
		// Mark if currently active
		if cp.ExpiresAt > nowSec {
			res = append(res, cp)
		}
	}
	return res
}

// Clear completely empties the LRU cache.
func (r *TargetAwareRoutingEngine) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = make(map[string]*list.Element, r.capacity)
	r.lruList.Init()
}

// ClearTarget manually resets any smart routing overrides for a target.
func (r *TargetAwareRoutingEngine) ClearTarget(rawHost string) {
	key := NormalizeTarget(rawHost)
	if key == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if elem, ok := r.entries[key]; ok {
		r.lruList.Remove(elem)
		delete(r.entries, key)
	}
}
