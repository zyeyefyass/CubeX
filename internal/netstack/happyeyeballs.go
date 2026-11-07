package netstack

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// HappyEyeballsTelemetry tracks telemetry of IPv6 vs IPv4 dual-stack connections.
type HappyEyeballsTelemetry struct {
	TotalAttempts   uint64 `json:"total_attempts"`
	IPv6Successes   uint64 `json:"ipv6_successes"`
	IPv4Fallbacks   uint64 `json:"ipv4_fallbacks"`
	AvgResolutionMs float64 `json:"avg_resolution_ms"`
	AvgHandshakeMs  float64 `json:"avg_handshake_ms"`
	LastDialedHost  string  `json:"last_dialed_host"`
	LastWinnerProto string  `json:"last_winner_proto"` // "IPv6" or "IPv4"
}

// HappyEyeballsDialer implements RFC 8305 dual-stack parallel dialer.
type HappyEyeballsDialer struct {
	FallbackDelay time.Duration // RFC 8305 recommends 200ms
	Timeout       time.Duration

	mu              sync.RWMutex
	totalAttempts   uint64
	ipv6Successes   uint64
	ipv4Fallbacks   uint64
	lastDialedHost  string
	lastWinnerProto string
}

var (
	globalHEDialer *HappyEyeballsDialer
	heOnce         sync.Once
)

// GetHappyEyeballsDialer returns the global dialer instance.
func GetHappyEyeballsDialer() *HappyEyeballsDialer {
	heOnce.Do(func() {
		globalHEDialer = &HappyEyeballsDialer{
			FallbackDelay: 200 * time.Millisecond,
			Timeout:       5 * time.Second,
		}
	})
	return globalHEDialer
}

// DialContext connects to host:port using Happy Eyeballs v2 algorithm.
func (d *HappyEyeballsDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	atomic.AddUint64(&d.totalAttempts, 1)

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		host = address
		port = "80"
	}

	d.mu.Lock()
	d.lastDialedHost = host
	d.mu.Unlock()

	// Direct IP without DNS lookup
	if ip := net.ParseIP(host); ip != nil {
		var dialer net.Dialer
		dialer.Timeout = d.Timeout
		return dialer.DialContext(ctx, network, address)
	}

	// 1. Resolve host addresses (IPv4 & IPv6)
	var resolver net.Resolver
	ips, err := resolver.LookupIPAddr(ctx, host)
	if err != nil || len(ips) == 0 {
		var dialer net.Dialer
		dialer.Timeout = d.Timeout
		return dialer.DialContext(ctx, network, address)
	}

	var ipv6List []net.IPAddr
	var ipv4List []net.IPAddr
	for _, ip := range ips {
		if ip.IP.To4() != nil {
			ipv4List = append(ipv4List, ip)
		} else if ip.IP.To16() != nil {
			ipv6List = append(ipv6List, ip)
		}
	}

	// If single stack, dial directly
	if len(ipv6List) == 0 || len(ipv4List) == 0 {
		var dialer net.Dialer
		dialer.Timeout = d.Timeout
		conn, dialErr := dialer.DialContext(ctx, network, address)
		if dialErr == nil {
			proto := "IPv4"
			if len(ipv6List) > 0 {
				proto = "IPv6"
				atomic.AddUint64(&d.ipv6Successes, 1)
			} else {
				atomic.AddUint64(&d.ipv4Fallbacks, 1)
			}
			d.mu.Lock()
			d.lastWinnerProto = proto
			d.mu.Unlock()
		}
		return conn, dialErr
	}

	// Dual-stack Happy Eyeballs v2 race:
	// Start IPv6 first, if not connected within FallbackDelay (200ms), start IPv4 concurrently.
	type dialResult struct {
		conn  net.Conn
		err   error
		proto string
	}

	resultCh := make(chan dialResult, 2)
	raceCtx, cancelRace := context.WithCancel(ctx)
	defer cancelRace()

	// Launch IPv6 attempt
	go func() {
		target := net.JoinHostPort(ipv6List[0].String(), port)
		var dialer net.Dialer
		dialer.Timeout = d.Timeout
		conn, err := dialer.DialContext(raceCtx, "tcp6", target)
		resultCh <- dialResult{conn: conn, err: err, proto: "IPv6"}
	}()

	// Wait up to FallbackDelay for IPv6
	fallbackTimer := time.NewTimer(d.FallbackDelay)
	defer fallbackTimer.Stop()

	var firstRes *dialResult
	select {
	case res := <-resultCh:
		if res.err == nil {
			// IPv6 succeeded before delay!
			atomic.AddUint64(&d.ipv6Successes, 1)
			d.mu.Lock()
			d.lastWinnerProto = "IPv6"
			d.mu.Unlock()
			return res.conn, nil
		}
		// IPv6 failed early, immediately proceed to IPv4
		firstRes = &res
	case <-fallbackTimer.C:
		// 200ms elapsed, launch IPv4 in parallel
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Launch IPv4 attempt
	go func() {
		target := net.JoinHostPort(ipv4List[0].String(), port)
		var dialer net.Dialer
		dialer.Timeout = d.Timeout
		conn, err := dialer.DialContext(raceCtx, "tcp4", target)
		resultCh <- dialResult{conn: conn, err: err, proto: "IPv4"}
	}()

	// Wait for whichever connects first
	needed := 1
	if firstRes != nil {
		// IPv6 already finished with error, only 1 candidate remains
		needed = 1
	} else {
		needed = 2
	}

	var lastErr error
	for i := 0; i < needed; i++ {
		select {
		case res := <-resultCh:
			if res.err == nil {
				cancelRace() // Cancel the losing attempt
				if res.proto == "IPv6" {
					atomic.AddUint64(&d.ipv6Successes, 1)
				} else {
					atomic.AddUint64(&d.ipv4Fallbacks, 1)
				}
				d.mu.Lock()
				d.lastWinnerProto = res.proto
				d.mu.Unlock()
				return res.conn, nil
			}
			lastErr = res.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("happy eyeballs dial failed: all candidates timed out")
}

// GetTelemetry returns live telemetry.
func (d *HappyEyeballsDialer) GetTelemetry() HappyEyeballsTelemetry {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return HappyEyeballsTelemetry{
		TotalAttempts:   atomic.LoadUint64(&d.totalAttempts),
		IPv6Successes:   atomic.LoadUint64(&d.ipv6Successes),
		IPv4Fallbacks:   atomic.LoadUint64(&d.ipv4Fallbacks),
		LastDialedHost:  d.lastDialedHost,
		LastWinnerProto: d.lastWinnerProto,
	}
}
