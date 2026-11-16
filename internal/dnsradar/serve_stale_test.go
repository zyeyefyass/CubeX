package dnsradar

import (
	"context"
	"testing"
	"time"
)

func TestServeStaleResolver_CacheHitAndStaleFallback(t *testing.T) {
	resolver := NewServeStaleResolver("223.5.5.5:53", "119.29.29.29:53", 100)

	// 手动注入一条已过期的缓存记录
	domain := "test.baidu.com"
	now := time.Now().UnixMilli()
	resolver.mu.Lock()
	entry := &StaleDNSEntry{
		Domain:      domain,
		IPs:         []string{"180.101.50.242"},
		ExpiresAt:   now - 1000, // 已过期 1 秒
		LastUpdated: now - 5000,
	}
	elem := resolver.lruList.PushFront(entry)
	resolver.cache[domain] = elem
	resolver.mu.Unlock()

	// 解析请求：上游可能慢，但有旧缓存，绝不能报错
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	ips, isStale, err := resolver.Resolve(ctx, domain)
	if err != nil {
		t.Fatalf("expected stale resolve to succeed, got error: %v", err)
	}

	if len(ips) == 0 {
		t.Fatal("expected at least 1 IP returned")
	}

	// 验证成功返回了旧 IP 或新 IP
	t.Logf("Resolved %s -> %v (isStale=%v)", domain, ips, isStale)
}

func TestServeStaleResolver_CapacityCap(t *testing.T) {
	cap := 5
	resolver := NewServeStaleResolver("223.5.5.5:53", "119.29.29.29:53", cap)

	for i := 0; i < 10; i++ {
		resolver.updateCache(string(rune('a'+i))+".example.com", []string{"1.2.3.4"}, time.Minute)
	}

	resolver.mu.RLock()
	defer resolver.mu.RUnlock()

	if len(resolver.cache) > cap {
		t.Fatalf("cache size %d exceeded capacity %d", len(resolver.cache), cap)
	}
	if resolver.lruList.Len() > cap {
		t.Fatalf("lru list len %d exceeded capacity %d", resolver.lruList.Len(), cap)
	}
}
