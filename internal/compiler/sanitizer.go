package compiler

import (
	"fmt"
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
	ir.Proxies = SanitizeAndDeduplicateNodes(ir.Proxies)

	// 2b. Auto-extract AI compliant nodes and inject/update "AI 专线" group
	var validProxyNames []string
	for _, p := range ir.Proxies {
		if p.Name != "" && !schema.IsInformationalOrInvalidNode(p.Name) {
			validProxyNames = append(validProxyNames, p.Name)
		}
	}
	ir.Groups = UpsertAICompliantGroup(ir.Groups, validProxyNames)

	// 3. Reconcile Policy Groups (Clean dangling references & guard against empty outbounds)
	ir.Groups = ReconcilePolicyGroups(ir.Groups, ir.Proxies)

	// 4. Sanitize DNS (Anti-Recursive DNS Deadlock Guard)
	SanitizeDNS(&ir.DNS)
}

// ReconcilePolicyGroups 联动清洗所有策略组中的节点引用：
// 1. 移除已被 SanitizeAndDeduplicateNodes 剔除的脏节点/伪节点名称悬空引用
// 2. 剔除单个策略组内部的重复节点引用
// 3. 若某策略组清洗后无任何有效节点，自动注入 "DIRECT" 兜底，严防 sing-box / Mihomo 启动崩溃
func ReconcilePolicyGroups(groups []schema.StrategyGroup, validNodes []schema.ProxyNode) []schema.StrategyGroup {
	validSet := make(map[string]struct{}, len(validNodes)+len(groups)+6)
	for _, n := range validNodes {
		validSet[n.Name] = struct{}{}
	}
	// 允许策略组互相嵌套引用及内置关键字
	for _, g := range groups {
		validSet[g.Name] = struct{}{}
	}
	for _, builtin := range []string{"DIRECT", "REJECT", "REJECT-DROP", "PASS", "COMPATIBLE"} {
		validSet[builtin] = struct{}{}
	}

	cleanedGroups := make([]schema.StrategyGroup, 0, len(groups))
	for _, g := range groups {
		seenInGroup := make(map[string]struct{}, len(g.Proxies))
		kept := make([]string, 0, len(g.Proxies))

		for _, member := range g.Proxies {
			if _, ok := validSet[member]; !ok {
				continue // 剔除已被净化的脏节点悬空引用
			}
			if _, dup := seenInGroup[member]; dup {
				continue
			}
			seenInGroup[member] = struct{}{}
			kept = append(kept, member)
		}

		// 空池防崩兜底：若该组所有节点均被净化剔除，强制挂入 DIRECT 保证内核 100% 能启动
		if len(kept) == 0 {
			kept = []string{"DIRECT"}
		}
		g.Proxies = kept

		// 如果当前选中的 Now 节点已被剔除，自愈回退到第一个有效节点
		if g.Now != "" {
			if _, ok := validSet[g.Now]; !ok {
				g.Now = kept[0]
			}
		}

		cleanedGroups = append(cleanedGroups, g)
	}

	return cleanedGroups
}

// SanitizeAndDeduplicateNodes filters out airport pseudo-nodes (remaining traffic, expiration, website URLs),
// drops invalid servers, and deduplicates node names.
func SanitizeAndDeduplicateNodes(nodes []schema.ProxyNode) []schema.ProxyNode {
	cleanProxies := make([]schema.ProxyNode, 0, len(nodes))
	nameSet := make(map[string]bool)

	for _, p := range nodes {
		p.Name = strings.TrimSpace(p.Name)
		p.Server = strings.TrimSpace(p.Server)

		// 1. Filter out informational or invalid notices (e.g. 剩余流量, 到期时间, 官网)
		if schema.IsInformationalOrInvalidNode(p.Name) {
			log.Printf("[NodeSanitizer] Dropped pseudo/notice node: %q", p.Name)
			continue
		}

		// 🛡️ Guard against stringified array in Server (e.g. "[1.2.3.4 5.6.7.8]")
		if strings.HasPrefix(p.Server, "[") && strings.HasSuffix(p.Server, "]") {
			// Check if it's a valid IPv6 literal like "[2001:db8::1]"
			rawHost := strings.Trim(p.Server, "[]")
			if net.ParseIP(rawHost) == nil {
				// Not a valid IPv6, it's a stringified slice from bad subscription!
				fields := strings.Fields(rawHost)
				if len(fields) > 0 {
					p.Server = strings.TrimSpace(fields[0])
					log.Printf("[ProxySanitizer] Recovered array-polluted server host for %q -> %s", p.Name, p.Server)
				}
			}
		}

		// Sanitize RawConfig if present
		if len(p.RawConfig) > 0 {
			p.RawConfig = SanitizeNodeRawConfig(p.RawConfig)
		}

		if p.Name == "" || p.Server == "" || p.Port <= 0 || p.Port > 65535 {
			log.Printf("[ProxySanitizer] Dropped invalid proxy (name=%q, server=%q, port=%d)", p.Name, p.Server, p.Port)
			continue
		}

		// Server must not contain invalid URI characters or spaces
		if strings.ContainsAny(p.Server, " \t\r\n/\\") {
			log.Printf("[ProxySanitizer] Dropped proxy with malformed server host: %q (%s)", p.Name, p.Server)
			continue
		}

		// Loopback or dummy servers disguised as proxies
		if p.Server == "127.0.0.1" || p.Server == "0.0.0.0" || strings.ToLower(p.Server) == "localhost" {
			log.Printf("[ProxySanitizer] Dropped loopback dummy server: %q (%s)", p.Name, p.Server)
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
	return cleanProxies
}

// SanitizeNodeRawConfig cleans and normalizes raw map keys from third-party airport subscriptions,
// strictly guaranteeing that scalar fields (server, port, password, uuid, type) are never arrays.
func SanitizeNodeRawConfig(raw map[string]any) map[string]any {
	cleaned := make(map[string]any, len(raw))
	for k, v := range raw {
		if v == nil {
			continue
		}
		// String scalar fields: if slice, unwrap first element
		switch k {
		case "server", "name", "tag", "type", "password", "uuid", "cipher", "method", "network", "flow", "sni", "servername":
			switch val := v.(type) {
			case string:
				cleaned[k] = strings.TrimSpace(val)
			case []any:
				if len(val) > 0 && val[0] != nil {
					cleaned[k] = strings.TrimSpace(fmt.Sprintf("%v", val[0]))
				}
			case []string:
				if len(val) > 0 {
					cleaned[k] = strings.TrimSpace(val[0])
				}
			default:
				cleaned[k] = strings.TrimSpace(fmt.Sprintf("%v", val))
			}
		case "port", "server_port":
			switch val := v.(type) {
			case int:
				cleaned[k] = val
			case float64:
				cleaned[k] = int(val)
			case string:
				if p, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
					cleaned[k] = p
				}
			case []any:
				if len(val) > 0 && val[0] != nil {
					if p, err := strconv.Atoi(fmt.Sprintf("%v", val[0])); err == nil {
						cleaned[k] = p
					}
				}
			case []int:
				if len(val) > 0 {
					cleaned[k] = val[0]
				}
			default:
				cleaned[k] = v
			}
		default:
			cleaned[k] = v
		}
	}
	return cleaned
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
