package compiler

import (
	"fmt"
	"log"
	"strings"

	"github.com/zyeyefyass/CubeX/internal/schema"
	"gopkg.in/yaml.v3"
)

// BuildMihomo transcompiles CubeX Universal IR to production Mihomo config.yaml
func BuildMihomo(ir *schema.ConfigIR) ([]byte, error) {
	if ir == nil {
		return nil, fmt.Errorf("config IR is nil")
	}

	net := ir.Network
	dns := ir.DNS

	cfg := map[string]any{
		"mixed-port":          net.MixedPort,
		"redir-port":          net.RedirPort,
		"tproxy-port":         net.TProxyPort,
		"allow-lan":           net.AllowLAN,
		"mode":                string(ir.Meta.ProxyMode),
		"log-level":           "warning",
		"ipv6":                net.IPv6,
		"external-controller": fmt.Sprintf("0.0.0.0:%d", net.ClashAPIPort),
		"secret":              net.ClashSecret,
		"unified-delay":       true,
		"tcp-concurrent":      true,
		"geodata-mode":        true,
		"find-process-mode":   "strict",
		"keep-alive-idle":     600,
		"keep-alive-interval": 15,
	}

	tunMTU := net.TunMTU
	if tunMTU <= 0 || tunMTU > 1500 {
		tunMTU = 1500
	}

	tunStack := net.TunStack
	if tunStack == "" || tunStack == "mixed" {
		tunStack = "system"
	}

	// TUN Configuration (Zero-copy kernel stack)
	cfg["tun"] = map[string]any{
		"enable":                   true,
		"stack":                    tunStack,
		"device":                   net.TunInterface,
		"auto-route":               true,
		"auto-detect-interface":    true,
		"strict-route":             false,
		"mtu":                      tunMTU,
		"endpoint-independent-nat": true,
		"dns-hijack": []string{
			"tcp://any:53",
			"udp://any:53",
		},
	}

	// Sniffer (Disabled by default to eliminate CPU overhead and TLS handshake latency)
	cfg["sniffer"] = map[string]any{
		"enable":               false,
		"force-dns-mapping":    false,
		"parse-pure-ip":        true,
		"override-destination": false,
		"sniff": map[string]any{
			"HTTP": map[string]any{"ports": []any{80, "8080-8880"}},
			"TLS":  map[string]any{"ports": []int{443, 8443}},
			"QUIC": map[string]any{"ports": []int{443, 8443}},
		},
		"skip-domain": []string{
			"Mijia Cloud",
			"dlg.io.mi.com",
			"+.apple.com",
			"+.push.apple.com",
		},
	}

	// Determine outbound proxy group for remote DoH queries
	foreignGroup := "🐟 漏网之鱼"
	for _, g := range ir.Groups {
		if g.Name == "🚀 国外" || g.Name == "🐟 漏网之鱼" || g.Name == "PROXY" {
			foreignGroup = g.Name
			break
		}
	}

	// DNS Configuration (Mihomo v1.19+ anti-leak & proxy-server bootstrap)
	dnsPort := net.DNSPort
	if dnsPort == 0 || dnsPort == 1053 {
		dnsPort = 53
	}

	remoteDNSList := []string{
		fmt.Sprintf("https://8.8.8.8/dns-query#%s&ecs=223.5.5.0/24&disable-qtype-65=true", foreignGroup),
	}
	fallbackDNSList := []string{
		fmt.Sprintf("https://8.8.8.8/dns-query#%s&disable-qtype-65=true", foreignGroup),
	}

	dnsMap := map[string]any{
		"enable":             true,
		"cache-algorithm":    "arc",
		"cache-size":         4096,
		"listen":             fmt.Sprintf("0.0.0.0:%d", dnsPort),
		"ipv6":               net.IPv6,
		"enhanced-mode":      "fake-ip",
		"fake-ip-range":      dns.FakeIPRange,
		"fake-ip-filter":     dns.FakeIPFilter,
		"default-nameserver": []string{
			"https://223.5.5.5/dns-query",
			"223.5.5.5",
			"119.29.29.29",
		},
		"proxy-server-nameserver": []string{
			"https://dns.alidns.com/dns-query",
			"https://doh.pub/dns-query",
			"223.5.5.5",
		},
		"direct-nameserver": []string{
			"https://dns.alidns.com/dns-query",
			"https://doh.pub/dns-query",
		},
		"nameserver":    remoteDNSList,
		"fallback":      fallbackDNSList,
		"respect-rules": true,
		"fallback-filter": map[string]any{
			"geoip":      true,
			"geoip-code": "CN",
			"ipcidr": []string{
				"240.0.0.0/4",
			},
		},
		"nameserver-policy": map[string]any{
			"geosite:cn,private": []string{
				"https://dns.alidns.com/dns-query#disable-qtype-65=true",
				"https://doh.pub/dns-query#disable-qtype-65=true",
			},
		},
	}
	cfg["dns"] = dnsMap

	// Build map of disabled or exhausted subscriptions
	disabledSubIDs := make(map[string]bool)
	for _, sub := range ir.Subscriptions {
		if !sub.Enabled || (sub.TotalBytes > 0 && sub.UsedBytes >= sub.TotalBytes) {
			disabledSubIDs[sub.ID] = true
		}
	}

	// Proxies (Preserve RawConfig for VLESS Reality, AnyTLS, Hysteria 2, and TUIC)
	proxiesList := make([]map[string]any, 0, len(ir.Proxies))
	for _, p := range ir.Proxies {
		if p.SubID != "" && disabledSubIDs[p.SubID] {
			continue
		}
		ptype := strings.ToLower(p.Type)
		if ptype == "direct" || ptype == "block" || ptype == "reject" || ptype == "" {
			continue
		}
		if p.Server == "" || p.Port <= 0 || p.Name == "" {
			continue
		}

		pMap := make(map[string]any)
		if len(p.RawConfig) > 0 {
			for k, v := range p.RawConfig {
				pMap[k] = v
			}
		} else {
			pMap["name"] = p.Name
			pMap["type"] = p.Type
			pMap["server"] = p.Server
			pMap["port"] = p.Port
			if p.UUID != "" {
				pMap["uuid"] = p.UUID
			}
			if p.Password != "" {
				pMap["password"] = p.Password
			}
			if p.Cipher != "" {
				pMap["cipher"] = p.Cipher
			}
			if p.TLS {
				pMap["tls"] = true
				if p.SNI != "" {
					pMap["servername"] = p.SNI
				}
				if p.SkipCertVerify {
					pMap["skip-cert-verify"] = true
				}
				if len(p.ALPN) > 0 {
					pMap["alpn"] = p.ALPN
				}
			}
			if p.Network != "" {
				pMap["network"] = p.Network
			}
			if p.Path != "" {
				pMap["ws-opts"] = map[string]any{"path": p.Path}
			}
			if p.Flow != "" {
				pMap["flow"] = p.Flow
			}
		}
		pMap["name"] = p.Name
		pMap["server"] = p.Server
		pMap["port"] = p.Port
		pMap["udp"] = true

		// Normalize Type for Mihomo (Clash Meta)
		if ptype == "shadowsocks" {
			ptype = "ss"
		} else if ptype == "shadowsocksr" {
			ptype = "ssr"
		}
		pMap["type"] = ptype

		// Mihomo strict validation fixes
		if ptype == "vmess" {
			if p.UUID == "" {
				continue
			}
			if _, ok := pMap["cipher"]; !ok || pMap["cipher"] == "" {
				pMap["cipher"] = "auto"
			}
			if _, ok := pMap["alterId"]; !ok {
				pMap["alterId"] = 0
			}
		} else if ptype == "vless" {
			if p.UUID == "" {
				continue
			}
		} else if ptype == "ss" {
			cipher := strings.ToLower(p.Cipher)
			if !ValidSSCiphers[cipher] {
				log.Printf("[Compiler] Skipping invalid SS proxy %s: unsupported cipher %s", p.Name, p.Cipher)
				continue
			}
			if strings.HasPrefix(cipher, "2022-") {
				cleanPwd, ok := CleanSS2022Password(cipher, p.Password)
				if !ok {
					log.Printf("[Compiler] Skipping SS-2022 proxy %s: invalid or multi-user password %s", p.Name, p.Password)
					continue
				}
				pMap["password"] = cleanPwd
			}
		}

		// Filter out protocols that Mihomo cannot parse to avoid daemon abort
		switch ptype {
		case "ss", "ssr", "vmess", "vless", "trojan", "socks5", "http", "snell", "wireguard", "tuic", "hysteria", "hysteria2":
			proxiesList = append(proxiesList, pMap)
		default:
			// Unsupported protocol in Mihomo (e.g. anytls) - safely skip
			continue
		}
	}
	cfg["proxies"] = proxiesList

	// Build set of all known valid target names (proxies + group names)
	validTargets := make(map[string]bool)
	validTargets["DIRECT"] = true
	validTargets["REJECT"] = true
	validTargets["COMPATIBLE"] = true
	validTargets["PASS"] = true
	for _, p := range proxiesList {
		if name, ok := p["name"].(string); ok {
			validTargets[name] = true
		}
	}
	for _, g := range ir.Groups {
		validTargets[g.Name] = true
	}

	// Proxy Groups
	groupsList := make([]map[string]any, 0, len(ir.Groups))
	for _, g := range ir.Groups {
		gType := g.Type
		if gType == "select" {
			gType = "select"
		} else if gType == "urltest" {
			gType = "url-test"
		} else if gType == "loadbalance" {
			gType = "load-balance"
		}

		proxies := make([]string, 0, len(g.Proxies))
		for _, p := range g.Proxies {
			if p != g.Name && validTargets[p] {
				proxies = append(proxies, p)
			}
		}
		if len(proxies) == 0 {
			proxies = append(proxies, "DIRECT")
		}

		// If g.Now is specified, move it to the front of proxies list so Mihomo selects it by default
		if g.Now != "" && validTargets[g.Now] {
			reordered := []string{g.Now}
			for _, p := range proxies {
				if p != g.Now {
					reordered = append(reordered, p)
				}
			}
			proxies = reordered
		}

		gMap := map[string]any{
			"name":    g.Name,
			"type":    gType,
			"proxies": proxies,
		}
		if gType == "url-test" || gType == "fallback" || gType == "load-balance" {
			gMap["url"] = g.URL
			if gMap["url"] == nil || gMap["url"] == "" {
				gMap["url"] = "http://www.gstatic.com/generate_204"
			}
			gMap["interval"] = g.Interval
			if g.Interval == 0 {
				gMap["interval"] = 300
			}
			gMap["lazy"] = true
			if g.Tolerance > 0 {
				gMap["tolerance"] = g.Tolerance
			} else if gType == "url-test" {
				gMap["tolerance"] = 50
			}
		}
		groupsList = append(groupsList, gMap)
	}
	cfg["proxy-groups"] = groupsList

	// 217heidai MRS Adaptive Rule Providers for Mihomo
	ruleProviders := map[string]any{
		"217heidai_ads": map[string]any{
			"type":     "file",
			"behavior": "domain",
			"format":   "mrs",
			"path":     "./rules/ads_domain.mrs",
		},
		"privacy_domain": map[string]any{
			"type":     "file",
			"behavior": "domain",
			"format":   "mrs",
			"path":     "./rules/privacy_domain.mrs",
		},
		"cn_domain": map[string]any{
			"type":     "file",
			"behavior": "domain",
			"format":   "mrs",
			"path":     "./rules/cn_domain.mrs",
		},
		"cn_ip": map[string]any{
			"type":     "file",
			"behavior": "ipcidr",
			"format":   "mrs",
			"path":     "./rules/cn_ip.mrs",
		},
	}

	bizMRS := []struct {
		id       string
		group    string
		file     string
		behavior string
	}{
		{"google_domain", "Google", "./rules/google.mrs", "domain"},
		{"youtube_domain", "YouTube", "./rules/youtube.mrs", "domain"},
		{"openai_domain", "AI", "./rules/openai.mrs", "domain"},
		{"github_domain", "GitHub & Docker", "./rules/github.mrs", "domain"},
		{"telegram_domain", "Telegram", "./rules/telegram.mrs", "domain"},
		{"telegram_ip", "Telegram", "./rules/telegram_ip.mrs", "ipcidr"},
		{"twitter_domain", "Twitter (X)", "./rules/twitter.mrs", "domain"},
		{"netflix_domain", "Netflix & 流媒体", "./rules/netflix.mrs", "domain"},
		{"spotify_domain", "Spotify", "./rules/spotify.mrs", "domain"},
		{"steam_domain", "Steam 游戏", "./rules/steam.mrs", "domain"},
		{"apple_domain", "Apple 服务", "./rules/apple.mrs", "domain"},
		{"microsoft_domain", "Microsoft", "./rules/microsoft.mrs", "domain"},
	}

	for _, bm := range bizMRS {
		if validTargets[bm.group] {
			ruleProviders[bm.id] = map[string]any{
				"type":     "file",
				"behavior": bm.behavior,
				"format":   "mrs",
				"path":     bm.file,
			}
		}
	}
	cfg["rule-providers"] = ruleProviders

	// Rules compilation with device policies top-priority prepending
	rulesList := make([]string, 0)

	// 1. Device Policies (Top Priority)
	for _, dev := range ir.DevicePolicies {
		cidr := dev.IP
		if !strings.Contains(cidr, "/") {
			cidr = cidr + "/32"
		}
		if dev.Mode == "bypass" {
			rulesList = append(rulesList, fmt.Sprintf("SRC-IP-CIDR,%s,DIRECT", cidr))
		} else if dev.Mode == "node" && dev.BoundNode != "" {
			rulesList = append(rulesList, fmt.Sprintf("SRC-IP-CIDR,%s,%s", cidr, dev.BoundNode))
		}
	}

	// 2. 217heidai MRS High-Performance Interception Rules (Mihomo Adaptive)
	rulesList = append(rulesList, "RULE-SET,217heidai_ads,REJECT")
	rulesList = append(rulesList, "RULE-SET,privacy_domain,REJECT")
	rulesList = append(rulesList, "RULE-SET,cn_domain,DIRECT")
	rulesList = append(rulesList, "RULE-SET,cn_ip,DIRECT")

	// Intercept foreign UDP 443/8443 (QUIC) to prevent ISP UDP QoS throttling and force BBR TCP
	rulesList = append(rulesList, "AND,((NETWORK,udp),(OR,((DST-PORT,443),(DST-PORT,8443)))),REJECT")

	// 2a. YouTube Video Streaming High-Speed Direct Rules
	if validTargets["YouTube"] {
		rulesList = append(rulesList, "DOMAIN-SUFFIX,googlevideo.com,YouTube")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,ytimg.com,YouTube")
		rulesList = append(rulesList, "DOMAIN-KEYWORD,googlevideo,YouTube")
		rulesList = append(rulesList, "DOMAIN-KEYWORD,sponsorblock,YouTube")
		rulesList = append(rulesList, "DOMAIN,youtubei.googleapis.com,YouTube")
	}

	// 2b. Business Service Rule-Sets
	for _, bm := range bizMRS {
		if validTargets[bm.group] {
			rulesList = append(rulesList, fmt.Sprintf("RULE-SET,%s,%s", bm.id, bm.group))
		}
	}

	// 3. Generic Rules
	for _, r := range ir.Rules {
		rType := strings.ToUpper(r.Type)
		target := r.Target
		if !validTargets[target] {
			if strings.Contains(target, "拦截") || strings.Contains(target, "REJECT") || strings.Contains(target, "BLOCK") {
				target = "REJECT"
			} else if strings.Contains(target, "直连") || strings.Contains(target, "DIRECT") {
				target = "DIRECT"
			} else if validTargets["🐟 漏网之鱼"] {
				target = "🐟 漏网之鱼"
			} else {
				target = "DIRECT"
			}
		}

		switch rType {
		case "DOMAIN":
			rulesList = append(rulesList, fmt.Sprintf("DOMAIN,%s,%s", r.Value, target))
		case "DOMAIN_SUFFIX", "DOMAIN-SUFFIX":
			rulesList = append(rulesList, fmt.Sprintf("DOMAIN-SUFFIX,%s,%s", r.Value, target))
		case "DOMAIN_KEYWORD", "DOMAIN-KEYWORD":
			rulesList = append(rulesList, fmt.Sprintf("DOMAIN-KEYWORD,%s,%s", r.Value, target))
		case "GEOSITE":
			rulesList = append(rulesList, fmt.Sprintf("GEOSITE,%s,%s", r.Value, target))
		case "GEOIP":
			rulesList = append(rulesList, fmt.Sprintf("GEOIP,%s,%s", strings.ToUpper(r.Value), target))
		case "IP_CIDR", "IP-CIDR":
			rulesList = append(rulesList, fmt.Sprintf("IP-CIDR,%s,%s", r.Value, target))
		case "SRC_IP_CIDR", "SRC-IP-CIDR":
			rulesList = append(rulesList, fmt.Sprintf("SRC-IP-CIDR,%s,%s", r.Value, target))
		case "RULE_SET", "RULE-SET":
			rulesList = append(rulesList, fmt.Sprintf("RULE-SET,%s,%s", r.Value, target))
		case "MATCH", "FINAL":
			rulesList = append(rulesList, fmt.Sprintf("MATCH,%s", target))
		}
	}

	// Fallback MATCH if not present
	hasMatch := false
	for _, r := range rulesList {
		if strings.HasPrefix(r, "MATCH,") {
			hasMatch = true
			break
		}
	}
	if !hasMatch {
		if validTargets["🐟 漏网之鱼"] {
			rulesList = append(rulesList, "MATCH,🐟 漏网之鱼")
		} else {
			rulesList = append(rulesList, "MATCH,DIRECT")
		}
	}

	cfg["rules"] = rulesList

	return yaml.Marshal(cfg)
}
