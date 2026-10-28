package compiler

import (
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

// SanitizeIR runs a fault-tolerant sandbox cleaning pipeline over the IR.
// It detects, cleans, and discards any illegal, corrupt, or broken third-party rules and proxy entries,
// logging clear warnings and guaranteeing that dirty subscription data will never block or crash
// dual-engine (Mihomo × sing-box) compilation.
func SanitizeIR(ir *schema.ConfigIR) {
	if ir == nil {
		return
	}

	// 1. Sanitize Rules
	cleanRules := make([]schema.Rule, 0, len(ir.Rules))
	for _, r := range ir.Rules {
		trimmedVal := strings.TrimSpace(r.Value)
		if trimmedVal == "" {
			log.Printf("[RuleSanitizer] Dropped empty rule for target: %s", r.Target)
			continue
		}

		rType := strings.ToLower(strings.TrimSpace(r.Type))
		isValid := true

		switch rType {
		case "ip_cidr", "ip-cidr", "src_ip_cidr", "src-ip-cidr":
			// Must be valid CIDR notation or valid single IP
			if !strings.Contains(trimmedVal, "/") {
				if net.ParseIP(trimmedVal) == nil {
					log.Printf("[RuleSanitizer] Dropped malformed IP/CIDR rule: %s (type=%s)", trimmedVal, rType)
					isValid = false
				}
			} else {
				if _, _, err := net.ParseCIDR(trimmedVal); err != nil {
					log.Printf("[RuleSanitizer] Dropped invalid CIDR rule: %s (err=%v)", trimmedVal, err)
					isValid = false
				}
			}

		case "port", "src_port", "src-port":
			if strings.Contains(trimmedVal, "-") || strings.Contains(trimmedVal, ":") {
				// Port range
				parts := strings.FieldsFunc(trimmedVal, func(c rune) bool { return c == '-' || c == ':' })
				if len(parts) != 2 {
					log.Printf("[RuleSanitizer] Dropped invalid port range: %s", trimmedVal)
					isValid = false
				} else {
					p1, e1 := strconv.Atoi(strings.TrimSpace(parts[0]))
					p2, e2 := strconv.Atoi(strings.TrimSpace(parts[1]))
					if e1 != nil || e2 != nil || p1 <= 0 || p2 <= 0 || p1 > 65535 || p2 > 65535 || p1 > p2 {
						log.Printf("[RuleSanitizer] Dropped out-of-range port range: %s", trimmedVal)
						isValid = false
					}
				}
			} else {
				p, err := strconv.Atoi(trimmedVal)
				if err != nil || p <= 0 || p > 65535 {
					log.Printf("[RuleSanitizer] Dropped invalid port number: %s", trimmedVal)
					isValid = false
				}
			}

		case "domain", "domain_suffix", "domain-suffix", "domain_keyword", "domain-keyword":
			// Domain must not contain whitespace, unprintable characters, or invalid symbols
			if strings.ContainsAny(trimmedVal, " \t\r\n/\\@!#$%^&*()=+~`\"'<>?") {
				log.Printf("[RuleSanitizer] Dropped invalid domain syntax: %s", trimmedVal)
				isValid = false
			}
		}

		if isValid {
			r.Value = trimmedVal
			cleanRules = append(cleanRules, r)
		}
	}
	ir.Rules = cleanRules

	// 2. Sanitize Proxies
	cleanProxies := make([]schema.ProxyNode, 0, len(ir.Proxies))
	nameSet := make(map[string]bool)

	for _, p := range ir.Proxies {
		p.Name = strings.TrimSpace(p.Name)
		p.Server = strings.TrimSpace(p.Server)
		if p.Name == "" || p.Server == "" || p.Port <= 0 || p.Port > 65535 {
			log.Printf("[ProxySanitizer] Dropped invalid proxy (name=%q, server=%q, port=%d)", p.Name, p.Server, p.Port)
			continue
		}

		// Server must not contain invalid URI characters or spaces
		if strings.ContainsAny(p.Server, " \t\r\n/\\") {
			log.Printf("[ProxySanitizer] Dropped proxy with malformed server host: %q (%s)", p.Name, p.Server)
			continue
		}

		// Prevent duplicate proxy names which break sing-box outbounds tag uniqueness
		originalName := p.Name
		counter := 1
		for nameSet[p.Name] {
			counter++
			p.Name = originalName + " " + strconv.Itoa(counter)
		}
		nameSet[p.Name] = true

		cleanProxies = append(cleanProxies, p)
	}
	ir.Proxies = cleanProxies

	// 3. Sanitize DNS (Anti-Recursive DNS Deadlock Guard)
	SanitizeDNS(&ir.DNS)
}

// IsLoopbackOrDeadlockDNS checks whether a DNS address is a loopback, unspecified,
// systemd-resolved, or recursive deadlock address that could cause engine query deadlocks.
func IsLoopbackOrDeadlockDNS(addr string) bool {
	raw := strings.TrimSpace(addr)
	if raw == "" {
		return true
	}
	// Strip protocol prefixes like udp://, tcp://, tls://, https://, quic://, h3://
	if idx := strings.Index(raw, "://"); idx != -1 {
		raw = raw[idx+3:]
	}
	// Strip path component for DoH/DoQ (e.g. 1.1.1.1/dns-query)
	if idx := strings.Index(raw, "/"); idx != -1 {
		raw = raw[:idx]
	}
	// Strip port if present
	host := raw
	if h, _, err := net.SplitHostPort(raw); err == nil {
		host = h
	}
	host = strings.ToLower(strings.Trim(host, "[]"))

	if host == "localhost" || host == "localhost.localdomain" || host == "0.0.0.0" || host == "::" {
		return true
	}

	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() || ip.IsUnspecified() {
			return true
		}
		// Check for 127.0.0.0/8
		if ip4 := ip.To4(); ip4 != nil {
			if ip4[0] == 127 || ip4[0] == 0 {
				return true
			}
		}
	}
	return false
}

// SanitizeDNS scrubs loopback, unspecified, and deadlocking recursive DNS servers
// from DirectDNS, RemoteDNS, and FallbackDNS, auto-recovering with resilient defaults.
func SanitizeDNS(dns *schema.DNSConfig) {
	if dns == nil {
		return
	}

	sanitizeList := func(list []string, defaultList []string, label string) []string {
		clean := make([]string, 0, len(list))
		for _, addr := range list {
			t := strings.TrimSpace(addr)
			if t == "" {
				continue
			}
			if IsLoopbackOrDeadlockDNS(t) {
				log.Printf("[DNSLoopbackGuard] Dropped loopback/deadlock DNS upstream (%s): %s", label, t)
				continue
			}
			clean = append(clean, t)
		}
		if len(clean) == 0 {
			clean = append(clean, defaultList...)
		}
		return clean
	}

	dns.DirectDNS = sanitizeList(dns.DirectDNS, []string{"223.5.5.5", "119.29.29.29"}, "DirectDNS")
	dns.RemoteDNS = sanitizeList(dns.RemoteDNS, []string{"tls://1.1.1.1", "tls://8.8.8.8"}, "RemoteDNS")
	dns.FallbackDNS = sanitizeList(dns.FallbackDNS, []string{"tls://1.1.1.1", "tls://8.8.8.8"}, "FallbackDNS")
}
