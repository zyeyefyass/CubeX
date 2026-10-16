package compiler

import (
	"fmt"
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
		"log-level":           ir.Meta.LogLevel,
		"ipv6":                net.IPv6,
		"external-controller": fmt.Sprintf("0.0.0.0:%d", net.ClashAPIPort),
		"secret":              net.ClashSecret,
		"unified-delay":       true,
		"tcp-concurrent":      true,
		"geodata-mode":        true,
		"find-process-mode":   "strict",
	}

	// TUN Configuration
	cfg["tun"] = map[string]any{
		"enable":               true,
		"stack":                net.TunStack,
		"device":                   net.TunInterface,
		"auto-route":               true,
		"auto-detect-interface":    true,
		"strict-route":             true,
		"mtu":                      net.TunMTU,
		"endpoint-independent-nat": true,
		"dns-hijack": []string{
			"tcp://any:53",
			"udp://any:53",
		},
	}

	// Sniffer (Mihomo v1.19+ standard syntax with string port ranges)
	cfg["sniffer"] = map[string]any{
		"enable":               true,
		"force-dns-mapping":    true,
		"parse-pure-ip":        true,
		"override-destination": false,
		"sniff": map[string]any{
			"HTTP": map[string]any{"ports": []any{80, "8080-8880"}, "override-destination": true},
			"TLS":  map[string]any{"ports": []int{443, 8443}, "override-destination": true},
			"QUIC": map[string]any{"ports": []int{443, 8443}, "override-destination": true},
		},
		"skip-domain": []string{
			"Mijia Cloud",
			"dlg.io.mi.com",
			"+.apple.com",
		},
	}

	// DNS Configuration (Mihomo v1.19+ anti-leak & proxy-server bootstrap)
	remoteDNSList := make([]string, 0, len(dns.RemoteDNS))
	for _, u := range dns.RemoteDNS {
		if (strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "tls://")) && !strings.Contains(u, "#") {
			remoteDNSList = append(remoteDNSList, u+"#disable-qtype-65=true")
		} else {
			remoteDNSList = append(remoteDNSList, u)
		}
	}
	fallbackDNSList := make([]string, 0, len(dns.FallbackDNS))
	for _, u := range dns.FallbackDNS {
		if (strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "tls://")) && !strings.Contains(u, "#") {
			fallbackDNSList = append(fallbackDNSList, u+"#disable-qtype-65=true")
		} else {
			fallbackDNSList = append(fallbackDNSList, u)
		}
	}

	dnsMap := map[string]any{
		"enable":                  true,
		"listen":                  fmt.Sprintf("0.0.0.0:%d", net.DNSPort),
		"ipv6":                    net.IPv6,
		"enhanced-mode":           "fake-ip",
		"fake-ip-range":           dns.FakeIPRange,
		"fake-ip-filter":          dns.FakeIPFilter,
		"default-nameserver":      dns.DirectDNS,
		"proxy-server-nameserver": dns.DirectDNS,
		"direct-nameserver":       dns.DirectDNS,
		"nameserver":              remoteDNSList,
		"fallback":                fallbackDNSList,
		"respect-rules":           true,
		"fallback-filter": map[string]any{
			"geoip":      true,
			"geoip-code": "CN",
			"ipcidr": []string{
				"240.0.0.0/4",
			},
		},
		"nameserver-policy": map[string]any{
			"geosite:cn,private": dns.DirectDNS,
		},
	}
	cfg["dns"] = dnsMap

	// Proxies (Preserve RawConfig for VLESS Reality, AnyTLS, Hysteria 2, and TUIC)
	proxiesList := make([]map[string]any, 0, len(ir.Proxies))
	for _, p := range ir.Proxies {
		if p.Type == "direct" || p.Type == "block" || p.Type == "reject" {
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
		pMap["type"] = p.Type
		pMap["server"] = p.Server
		pMap["port"] = p.Port
		pMap["udp"] = true
		proxiesList = append(proxiesList, pMap)
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
			if g.Tolerance > 0 {
				gMap["tolerance"] = g.Tolerance
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
