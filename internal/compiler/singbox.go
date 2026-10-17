package compiler

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

// BuildSingBox transcompiles CubeX Universal IR to sing-box v1.14.3+ config.json
func BuildSingBox(ir *schema.ConfigIR) ([]byte, error) {
	if ir == nil {
		return nil, fmt.Errorf("config IR is nil")
	}

	net := ir.Network
	dns := ir.DNS

	// Root config map
	cfg := map[string]any{
		"log": map[string]any{
			"level":     ir.Meta.LogLevel,
			"timestamp": true,
		},
	}

	// 1. DNS Section (v1.14+ compliant)
	directDNSAddr := "223.5.5.5"
	if len(dns.DirectDNS) > 0 {
		directDNSAddr = dns.DirectDNS[0]
	}
	remoteDNSAddr := "https://1.1.1.1/dns-query"
	if len(dns.RemoteDNS) > 0 {
		remoteDNSAddr = dns.RemoteDNS[0]
	}

	directServer := map[string]any{
		"tag":     "dns-direct",
		"address": directDNSAddr,
		"detour":  "direct",
	}

	remoteServer := map[string]any{
		"tag":     "dns-remote",
		"address": remoteDNSAddr,
		"detour":  "direct",
	}

	cfg["dns"] = map[string]any{
		"fakeip": map[string]any{
			"enabled":     true,
			"inet4_range": "198.18.0.0/15",
		},
		"servers": []map[string]any{
			{
				"tag":     "dns-block",
				"address": "rcode://refused",
			},
			directServer,
			{
				"tag":     "dns-fakeip",
				"address": "fakeip",
			},
			remoteServer,
		},
		"rules": []map[string]any{
			{
				"query_type": []string{"HTTPS", "TYPE65", "SVCB"},
				"server":     "dns-block",
			},
			{
				"clash_mode": "direct",
				"server":     "dns-direct",
			},
			{
				"clash_mode": "global",
				"server":     "dns-fakeip",
			},
			{
				"rule_set":      []string{"geosite-cn"},
				"domain_suffix": []string{".cn", "bilibili.com", "baidu.com", "qq.com", "taobao.com", "jd.com", "alipay.com", "163.com"},
				"server":        "dns-direct",
			},
			{
				"query_type": []string{"A", "AAAA"},
				"server":     "dns-fakeip",
			},
		},
		"final":    "dns-remote",
		"strategy": "prefer_ipv4",
	}

	// 2. Inbounds Section (sing-box 1.14+ compliant)
	inbounds := []map[string]any{
		{
			"type":           "tun",
			"tag":            "tun-in",
			"interface_name": net.TunInterface,
			"address":        []string{net.TunIPv4},
			"mtu":            net.TunMTU,
			"auto_route":     true,
			"strict_route":   true,
			"stack":          net.TunStack,
		},
		{
			"type":        "tproxy",
			"tag":         "tproxy-in",
			"listen":      "0.0.0.0",
			"listen_port": net.TProxyPort,
		},
		{
			"type":        "mixed",
			"tag":         "mixed-in",
			"listen":      "0.0.0.0",
			"listen_port": net.MixedPort,
		},
	}
	cfg["inbounds"] = inbounds

	// 3. Outbounds Section
	outbounds := make([]map[string]any, 0)

	// Built-in fixed endpoints
	outbounds = append(outbounds, map[string]any{
		"type": "direct",
		"tag":  "direct",
	})
	outbounds = append(outbounds, map[string]any{
		"type": "direct",
		"tag":  "DIRECT",
	})
	outbounds = append(outbounds, map[string]any{
		"type": "block",
		"tag":  "block",
	})
	outbounds = append(outbounds, map[string]any{
		"type": "block",
		"tag":  "REJECT",
	})

	// Add user proxy nodes
	nodeTags := make([]string, 0, len(ir.Proxies))
	for _, p := range ir.Proxies {
		if p.Type == "direct" || p.Type == "block" || p.Type == "reject" {
			continue
		}
		nodeTags = append(nodeTags, p.Name)

		nodeMap := map[string]any{
			"tag":         p.Name,
			"server":      p.Server,
			"server_port": p.Port,
		}

		pType := strings.ToLower(p.Type)
		switch pType {
		case "vless":
			nodeMap["type"] = "vless"
			nodeMap["uuid"] = p.UUID
			if p.Flow != "" {
				nodeMap["flow"] = p.Flow
			}
		case "vmess":
			nodeMap["type"] = "vmess"
			nodeMap["uuid"] = p.UUID
			nodeMap["security"] = "auto"
		case "trojan":
			nodeMap["type"] = "trojan"
			nodeMap["password"] = p.Password
		case "shadowsocks", "ss":
			nodeMap["type"] = "shadowsocks"
			nodeMap["method"] = p.Cipher
			nodeMap["password"] = p.Password
		case "hysteria2", "hy2":
			nodeMap["type"] = "hysteria2"
			nodeMap["password"] = p.Password
		case "tuic":
			nodeMap["type"] = "tuic"
			nodeMap["uuid"] = p.UUID
			nodeMap["password"] = p.Password
			nodeMap["congestion_control"] = "bbr"
		case "anytls":
			nodeMap["type"] = "anytls"
			nodeMap["password"] = p.Password
		default:
			nodeMap["type"] = pType
		}

		if p.TLS {
			tlsMap := map[string]any{
				"enabled": true,
			}
			if p.SNI != "" {
				tlsMap["server_name"] = p.SNI
			}
			if p.SkipCertVerify {
				tlsMap["insecure"] = true
			}
			if len(p.ALPN) > 0 {
				tlsMap["alpn"] = p.ALPN
			}
			nodeMap["tls"] = tlsMap
		}

		if p.Network == "ws" {
			nodeMap["transport"] = map[string]any{
				"type": "ws",
				"path": p.Path,
			}
		}

		outbounds = append(outbounds, nodeMap)
	}

	// Build set of valid outbounds (built-ins + user nodes + group tags)
	validOutbounds := make(map[string]bool)
	validOutbounds["direct"] = true
	validOutbounds["DIRECT"] = true
	validOutbounds["block"] = true
	validOutbounds["REJECT"] = true
	for _, tag := range nodeTags {
		validOutbounds[tag] = true
	}
	for _, g := range ir.Groups {
		validOutbounds[g.Name] = true
	}

	// Strategy groups -> sing-box selector / urltest
	for _, g := range ir.Groups {
		proxies := make([]string, 0, len(g.Proxies))
		for _, p := range g.Proxies {
			if p != g.Name && validOutbounds[p] {
				proxies = append(proxies, p)
			}
		}
		if len(proxies) == 0 {
			proxies = append(proxies, "direct")
		}

		gType := "selector"
		if g.Type == "urltest" || g.Type == "url-test" {
			gType = "urltest"
		}

		// If g.Now is specified, move it to the front of proxies list so sing-box selects it by default
		if g.Now != "" && validOutbounds[g.Now] {
			reordered := []string{g.Now}
			for _, p := range proxies {
				if p != g.Now {
					reordered = append(reordered, p)
				}
			}
			proxies = reordered
		}

		gOutbound := map[string]any{
			"type":      gType,
			"tag":       g.Name,
			"outbounds": proxies,
		}

		if gType == "urltest" {
			url := g.URL
			if url == "" {
				url = "http://www.gstatic.com/generate_204"
			}
			interval := g.Interval
			if interval == 0 {
				interval = 300
			}
			gOutbound["url"] = url
			gOutbound["interval"] = fmt.Sprintf("%ds", interval)
			if g.Tolerance > 0 {
				gOutbound["tolerance"] = g.Tolerance
			}
		}

		outbounds = append(outbounds, gOutbound)
	}
	cfg["outbounds"] = outbounds

	// 4. Route Section with Adaptive sing-box Rule-Sets (.srs)
	ruleSets := []map[string]any{
		{
			"tag":    "217heidai-ads",
			"type":   "local",
			"format": "binary",
			"path":   "rules/adblocksingbox.srs",
		},
		{
			"tag":    "privacy-tracking",
			"type":   "local",
			"format": "binary",
			"path":   "rules/privacy.srs",
		},
		{
			"tag":    "geosite-cn",
			"type":   "local",
			"format": "binary",
			"path":   "rules/geosite-cn.srs",
		},
		{
			"tag":    "geoip-cn",
			"type":   "local",
			"format": "binary",
			"path":   "rules/geoip-cn.srs",
		},
	}

	routeRules := make([]map[string]any, 0)

	// A. Global Sniff & DNS Hijack (sing-box 1.14+ action format)
	routeRules = append(routeRules, map[string]any{
		"action": "sniff",
	})
	routeRules = append(routeRules, map[string]any{
		"protocol": "dns",
		"action":   "hijack-dns",
	})

	// B. Device Policies (Top Priority)
	for _, dev := range ir.DevicePolicies {
		cidr := dev.IP
		if !strings.Contains(cidr, "/") {
			cidr = cidr + "/32"
		}
		if dev.Mode == "bypass" {
			routeRules = append(routeRules, map[string]any{
				"source_ip_cidr": []string{cidr},
				"outbound":       "direct",
			})
		} else if dev.Mode == "node" && dev.BoundNode != "" {
			routeRules = append(routeRules, map[string]any{
				"source_ip_cidr": []string{cidr},
				"outbound":       dev.BoundNode,
			})
		}
	}

	// C. Adaptive SRS Rule-Set Interception (Adblock & Anti-Tracking native sing-box)
	routeRules = append(routeRules, map[string]any{
		"rule_set": []string{"217heidai-ads", "privacy-tracking"},
		"outbound": "block",
	})
	routeRules = append(routeRules, map[string]any{
		"rule_set": []string{"geosite-cn", "geoip-cn"},
		"outbound": "direct",
	})

	// D. Generic Rules
	for _, r := range ir.Rules {
		rType := strings.ToLower(r.Type)
		targetOutbound := r.Target
		if !validOutbounds[targetOutbound] {
			if strings.Contains(targetOutbound, "拦截") || strings.EqualFold(targetOutbound, "REJECT") || strings.EqualFold(targetOutbound, "block") {
				targetOutbound = "block"
			} else if strings.Contains(targetOutbound, "直连") || strings.EqualFold(targetOutbound, "DIRECT") || strings.EqualFold(targetOutbound, "direct") {
				targetOutbound = "direct"
			} else if validOutbounds["🐟 漏网之鱼"] {
				targetOutbound = "🐟 漏网之鱼"
			} else {
				targetOutbound = "direct"
			}
		}

		ruleMap := map[string]any{
			"outbound": targetOutbound,
		}

		switch rType {
		case "domain":
			ruleMap["domain"] = []string{r.Value}
		case "domain_suffix", "domain-suffix":
			ruleMap["domain_suffix"] = []string{r.Value}
		case "domain_keyword", "domain-keyword":
			ruleMap["domain_keyword"] = []string{r.Value}
		case "geosite":
			val := strings.ToLower(r.Value)
			if strings.Contains(val, "ad") {
				ruleMap["rule_set"] = []string{"217heidai-ads"}
			} else if val == "cn" {
				ruleMap["rule_set"] = []string{"geosite-cn"}
			}
		case "geoip":
			val := strings.ToLower(r.Value)
			if val == "cn" {
				ruleMap["rule_set"] = []string{"geoip-cn"}
			}
		case "ip_cidr", "ip-cidr":
			ruleMap["ip_cidr"] = []string{r.Value}
		case "src_ip_cidr", "src-ip-cidr":
			ruleMap["source_ip_cidr"] = []string{r.Value}
		case "rule_set", "rule-set":
			ruleMap["rule_set"] = []string{r.Value}
		case "port":
			ruleMap["port"] = []int{80, 443}
		}

		if len(ruleMap) > 1 {
			routeRules = append(routeRules, ruleMap)
		}
	}

	finalOutbound := "direct"
	if validOutbounds["🐟 漏网之鱼"] {
		finalOutbound = "🐟 漏网之鱼"
	}

	cfg["route"] = map[string]any{
		"default_domain_resolver": "dns-direct",
		"rule_set":                ruleSets,
		"rules":                   routeRules,
		"auto_detect_interface":   true,
		"final":                   finalOutbound,
	}

	// 5. Experimental Clash API (1:1 state sync)
	cfg["experimental"] = map[string]any{
		"clash_api": map[string]any{
			"external_controller": fmt.Sprintf("0.0.0.0:%d", net.ClashAPIPort),
			"secret":              net.ClashSecret,
			"default_mode":        string(ir.Meta.ProxyMode),
		},
	}

	return json.MarshalIndent(cfg, "", "  ")
}
