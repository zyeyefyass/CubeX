package dnsradar

import (
	"container/list"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// StaleDNSEntry 缓存条目
type StaleDNSEntry struct {
	Domain      string
	IPs         []string
	ExpiresAt   int64 // 绝对到期毫秒时间戳
	LastUpdated int64 // 更新时间戳
}

// ServeStaleResolver 实现具有乐观缓存 (serve-stale) 与 80ms 阶梯并发竞速的 DNS 解析器
// 特性：
// 1. 固定 4096 容量 LRU 环形防溢出缓存，内存恒定常数级；
// 2. 乐观过期回包：记录 TTL 过期时，若上游在 150ms 内未响应，立即先返回旧 IP 保证页面秒开，后台异步静默刷新；
// 3. 80ms 阶梯并发 (Staggered Hedged Requests)：首选主上游 80ms 迟延触发副上游竞速，消灭弱网与节点抽风断网。
type ServeStaleResolver struct {
	mu                sync.RWMutex
	capacity          int
	lruList           *list.List
	cache             map[string]*list.Element
	primaryUpstream   string // 主上游 (如 223.5.5.5:53)
	secondaryUpstream string // 副上游 (如 119.29.29.29:53)
	hedgeDelay        time.Duration
	staleTimeout      time.Duration
	dialer            *net.Dialer
}

// NewServeStaleResolver 构造乐观缓存解析器
func NewServeStaleResolver(primary, secondary string, capacity int) *ServeStaleResolver {
	if primary == "" {
		primary = "223.5.5.5:53"
	}
	if secondary == "" {
		secondary = "119.29.29.29:53"
	}
	if !strings.Contains(primary, ":") {
		primary = primary + ":53"
	}
	if !strings.Contains(secondary, ":") {
		secondary = secondary + ":53"
	}
	if capacity <= 0 {
		capacity = 4096
	}

	return &ServeStaleResolver{
		capacity:          capacity,
		lruList:           list.New(),
		cache:             make(map[string]*list.Element, capacity),
		primaryUpstream:   primary,
		secondaryUpstream: secondary,
		hedgeDelay:        80 * time.Millisecond,
		staleTimeout:      150 * time.Millisecond,
		dialer:            &net.Dialer{Timeout: 2 * time.Second},
	}
}

// Resolve 解析域名，返回 (IP 列表, 是否为陈旧乐观回包, 错误)
func (r *ServeStaleResolver) Resolve(ctx context.Context, domain string) ([]string, bool, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return nil, false, fmt.Errorf("empty domain")
	}

	// 1. 查询缓存
	now := time.Now().UnixMilli()
	r.mu.Lock()
	if elem, ok := r.cache[domain]; ok {
		entry := elem.Value.(*StaleDNSEntry)
		r.lruList.MoveToFront(elem)

		// 命中新鲜缓存，直接返回 (0 额外耗时)
		if now < entry.ExpiresAt {
			ips := append([]string(nil), entry.IPs...)
			r.mu.Unlock()
			return ips, false, nil
		}

		// 缓存已过期，但具备陈旧数据 (Stale Entry 可用)
		staleIPs := append([]string(nil), entry.IPs...)
		r.mu.Unlock()

		// 启动异步上游阶梯竞速更新
		type fetchResult struct {
			ips []string
			err error
		}
		resCh := make(chan fetchResult, 1)

		go func() {
			ips, err := r.staggeredFetch(context.Background(), domain)
			resCh <- fetchResult{ips: ips, err: err}
			if err == nil && len(ips) > 0 {
				r.updateCache(domain, ips, 60*time.Second)
			}
		}()

		// 乐观等待 150ms：若 150ms 内上游快速返回最新 IP，则优先使用最新 IP；超时则立即先行返回旧 IP
		select {
		case <-ctx.Done():
			return staleIPs, true, nil
		case res := <-resCh:
			if res.err == nil && len(res.ips) > 0 {
				return res.ips, false, nil
			}
			return staleIPs, true, nil
		case <-time.After(r.staleTimeout):
			// 150ms 超时，立即返回乐观过期旧 IP，后台协程静默更新
			return staleIPs, true, nil
		}
	}
	r.mu.Unlock()

	// 2. 无任何历史缓存，执行 80ms 阶梯并发竞速
	ips, err := r.staggeredFetch(ctx, domain)
	if err != nil {
		return nil, false, err
	}

	r.updateCache(domain, ips, 60*time.Second)
	return ips, false, nil
}

// staggeredFetch 80ms 阶梯式并发竞速拉取
func (r *ServeStaleResolver) staggeredFetch(ctx context.Context, domain string) ([]string, error) {
	type queryOutcome struct {
		ips []string
		err error
	}

	outCh := make(chan queryOutcome, 2)
	fetchCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// 启动主上游查询
	go func() {
		ips, err := r.queryUpstream(fetchCtx, r.primaryUpstream, domain)
		outCh <- queryOutcome{ips: ips, err: err}
	}()

	// 启动 80ms 阶梯定时器：仅当 80ms 内主上游未响应时，才唤醒副上游竞速
	hedgeTimer := time.NewTimer(r.hedgeDelay)
	defer hedgeTimer.Stop()

	var secondaryStarted bool

	select {
	case out := <-outCh:
		if out.err == nil && len(out.ips) > 0 {
			return out.ips, nil
		}
		// 主上游报错，立即唤醒副上游
		secondaryStarted = true
		go func() {
			ips, err := r.queryUpstream(fetchCtx, r.secondaryUpstream, domain)
			outCh <- queryOutcome{ips: ips, err: err}
		}()
	case <-hedgeTimer.C:
		// 80ms 已过，主上游仍未响应，唤醒副上游并发竞速
		secondaryStarted = true
		go func() {
			ips, err := r.queryUpstream(fetchCtx, r.secondaryUpstream, domain)
			outCh <- queryOutcome{ips: ips, err: err}
		}()
	case <-fetchCtx.Done():
		return nil, fetchCtx.Err()
	}

	// 等待最终获胜者
	if secondaryStarted {
		select {
		case out := <-outCh:
			if out.err == nil && len(out.ips) > 0 {
				return out.ips, nil
			}
			// 尝试等待另一个结果
			select {
			case out2 := <-outCh:
				return out2.ips, out2.err
			case <-fetchCtx.Done():
				return nil, fetchCtx.Err()
			}
		case <-fetchCtx.Done():
			return nil, fetchCtx.Err()
		}
	}

	return nil, fmt.Errorf("all dns upstreams failed for %s", domain)
}

func (r *ServeStaleResolver) queryUpstream(ctx context.Context, upstreamAddr, domain string) ([]string, error) {
	res := net.Resolver{
		PreferGo: true,
		Dial: func(dialCtx context.Context, network, address string) (net.Conn, error) {
			return r.dialer.DialContext(dialCtx, "udp", upstreamAddr)
		},
	}
	return res.LookupHost(ctx, domain)
}

func (r *ServeStaleResolver) updateCache(domain string, ips []string, ttl time.Duration) {
	if len(ips) == 0 {
		return
	}
	now := time.Now().UnixMilli()
	exp := now + ttl.Milliseconds()

	r.mu.Lock()
	defer r.mu.Unlock()

	if elem, ok := r.cache[domain]; ok {
		entry := elem.Value.(*StaleDNSEntry)
		entry.IPs = ips
		entry.ExpiresAt = exp
		entry.LastUpdated = now
		r.lruList.MoveToFront(elem)
		return
	}

	if r.lruList.Len() >= r.capacity {
		oldest := r.lruList.Back()
		if oldest != nil {
			r.lruList.Remove(oldest)
			delete(r.cache, oldest.Value.(*StaleDNSEntry).Domain)
		}
	}

	entry := &StaleDNSEntry{
		Domain:      domain,
		IPs:         ips,
		ExpiresAt:   exp,
		LastUpdated: now,
	}
	elem := r.lruList.PushFront(entry)
	r.cache[domain] = elem
}

// SetUpstreams 动态更新主备上游
func (r *ServeStaleResolver) SetUpstreams(primary, secondary string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if primary != "" {
		if !strings.Contains(primary, ":") {
			primary += ":53"
		}
		r.primaryUpstream = primary
	}
	if secondary != "" {
		if !strings.Contains(secondary, ":") {
			secondary += ":53"
		}
		r.secondaryUpstream = secondary
	}
}
