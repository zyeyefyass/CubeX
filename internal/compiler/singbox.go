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

	cfg["dns"] = map[string]any{
		"servers": []map[string]any{
			{
				"tag":     "dns-direct",
				"address": directDNSAddr,
				"detour":  "direct",
			},
			{
				"tag":     "dns-remote",
				"address": remoteDNSAddr,
				"detour":  "🚀 节点选择",
			},
			{
				"tag":     "dns-fakeip",
				"address": "fakeip",
			},
		},
		"rules": []map[string]any{
			{
				"outbound": "any",
				"server":   "dns-direct",
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
				"geosite": []string{"cn"},
				"server":  "dns-direct",
			},
			{
				"query_type": []string{"A", "AAAA"},
				"server":     "dns-fakeip",
			},
		},
		"fakeip": map[string]any{
			"enabled":     true,
			"inet4_range": "198.18.0.0/15",
		},
		"strategy":          "prefer_ipv4",
		"independent_cache": true,
	}

	// 2. Inbounds Section
	inbounds := []map[string]any{
		{
			"type":                       "tun",
			"tag":                        "tun-in",
			"interface_name":              net.TunInterface,
			"inet4_address":              net.TunIPv4,
			"mtu":                        net.TunMTU,
			"auto_route":                 true,
			"strict_route":               true,
			"stack":                      net.TunStack,
			"sniff":                      true,
			"sniff_override_destination": true,
		},
		{
			"type":                       "tproxy",
			"tag":                        "tproxy-in",
			"listen":                     "0.0.0.0",
			"listen_port":                net.TProxyPort,
			"sniff":                      true,
			"sniff_override_destination": true,
		},
		{
			"type":        "mixed",
			"tag":         "mixed-in",
			"listen":      "0.0.0.0",
			"listen_port": net.MixedPort,
			"sniff":       true,
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
	outbounds = append(outbounds, map[string]any{
		"type": "dns",
		"tag":  "dns-out",
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

	// Strategy groups -> sing-box selector / urltest
	for _, g := range ir.Groups {
		proxies := g.Proxies
		if len(proxies) == 0 && len(nodeTags) > 0 {
			proxies = append([]string{"direct"}, nodeTags...)
		}

		gType := "selector"
		if g.Type == "urltest" || g.Type == "url-test" {
			gType = "urltest"
		}

		gOutbound := map[string]any{
			"type":      gType,
			"tag":       g.Name,
			"outbounds": proxies,
		}

		if g.Now != "" {
			gOutbound["default"] = g.Now
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

	// 4. Route Section
	routeRules := make([]map[string]any, 0)

	// A. Hijack DNS
	routeRules = append(routeRules, map[string]any{
		"protocol": "dns",
		"outbound": "dns-out",
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

	// C. Generic Rules
	for _, r := range ir.Rules {
		rType := strings.ToLower(r.Type)
		targetOutbound := r.Target
		if targetOutbound == "DIRECT" {
			targetOutbound = "direct"
		} else if targetOutbound == "REJECT" {
			targetOutbound = "block"
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
			ruleMap["geosite"] = []string{r.Value}
		case "geoip":
			ruleMap["geoip"] = []string{strings.ToLower(r.Value)}
		case "ip_cidr", "ip-cidr":
			ruleMap["ip_cidr"] = []string{r.Value}
		case "src_ip_cidr", "src-ip-cidr":
			ruleMap["source_ip_cidr"] = []string{r.Value}
		case "port":
			ruleMap["port"] = []int{80, 443}
		}

		if len(ruleMap) > 1 {
			routeRules = append(routeRules, ruleMap)
		}
	}

	cfg["route"] = map[string]any{
		"rules":                 routeRules,
		"auto_detect_interface": true,
		"final":                 "🐟 漏网之鱼",
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
