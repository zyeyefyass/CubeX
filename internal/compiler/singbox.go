package compiler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

func makeSingboxDNSServer(tag, rawAddr, detour string) map[string]any {
	serverMap := map[string]any{
		"tag": tag,
	}
	if detour != "" {
		serverMap["detour"] = detour
	}
	if strings.HasPrefix(rawAddr, "https://") {
		serverMap["type"] = "https"
		u, err := url.Parse(rawAddr)
		if err == nil {
			serverMap["server"] = u.Hostname()
			if u.Port() != "" {
				p, _ := strconv.Atoi(u.Port())
				serverMap["server_port"] = p
			}
			if u.Path != "" {
				serverMap["path"] = u.Path
			}
		} else {
			serverMap["server"] = strings.TrimPrefix(rawAddr, "https://")
		}
	} else if strings.HasPrefix(rawAddr, "tls://") {
		serverMap["type"] = "tls"
		serverMap["server"] = strings.TrimPrefix(rawAddr, "tls://")
	} else if strings.HasPrefix(rawAddr, "tcp://") {
		serverMap["type"] = "tcp"
		serverMap["server"] = strings.TrimPrefix(rawAddr, "tcp://")
	} else {
		serverMap["type"] = "udp"
		serverMap["server"] = rawAddr
	}
	return serverMap
}

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

	// Find foreign outbound group tag for remote DNS tunnel detour
	foreignDetour := "🐟 漏网之鱼"
	for _, g := range ir.Groups {
		if g.Name == "🚀 国外" || g.Name == "🐟 漏网之鱼" || g.Name == "PROXY" {
			foreignDetour = g.Name
			break
		}
	}

	directServer := makeSingboxDNSServer("dns-direct", directDNSAddr, "")
	remoteServer := makeSingboxDNSServer("dns-remote", remoteDNSAddr, foreignDetour)

	cfg["dns"] = map[string]any{
		"servers": []map[string]any{
			directServer,
			{
				"type":        "fakeip",
				"tag":         "dns-fakeip",
				"inet4_range": "198.18.0.0/15",
			},
			remoteServer,
		},
		"rules": []map[string]any{
			{
				"action":     "predefined",
				"query_type": []string{"HTTPS", "SVCB"},
				"rcode":      "NOERROR",
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
		"final":           "dns-remote",
		"reverse_mapping": true,
		"strategy":        "prefer_ipv4",
	}

	dnsPort := net.DNSPort
	if dnsPort == 0 {
		dnsPort = 53
	}

	tunMTU := net.TunMTU
	if tunMTU <= 0 || tunMTU > 1500 {
		tunMTU = 1500
	}

	tunStack := net.TunStack
	if tunStack == "" || tunStack == "mixed" {
		tunStack = "system"
	}

	inbounds := []map[string]any{
		{
			"type":           "tun",
			"tag":            "tun-in",
			"interface_name": net.TunInterface,
			"address":        []string{net.TunIPv4},
			"mtu":            tunMTU,
			"auto_route":     true,
			"strict_route":   false,
			"dns_mode":       "hijack",
			"udp_timeout":    "60s",
			"stack":          tunStack,
		},
		{
			"type":        "direct",
			"tag":         "dns-in",
			"listen":      "0.0.0.0",
			"listen_port": dnsPort,
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

	// Build map of disabled or exhausted subscriptions
	disabledSubIDs := make(map[string]bool)
	for _, sub := range ir.Subscriptions {
		if !sub.Enabled || (sub.TotalBytes > 0 && sub.UsedBytes >= sub.TotalBytes) {
			disabledSubIDs[sub.ID] = true
		}
	}

	// Add user proxy nodes
	nodeTags := make([]string, 0, len(ir.Proxies))
	for _, p := range ir.Proxies {
		if p.SubID != "" && disabledSubIDs[p.SubID] {
			continue
		}
		pType := strings.ToLower(p.Type)
		if pType == "direct" || pType == "block" || pType == "reject" || pType == "" {
			continue
		}
		if p.Server == "" || p.Port <= 0 || p.Name == "" {
			continue
		}

		nodeMap := map[string]any{
			"tag":         p.Name,
			"server":      p.Server,
			"server_port": p.Port,
		}

		valid := true
		switch pType {
		case "vless":
			if p.UUID == "" {
				continue
			}
			nodeMap["type"] = "vless"
			nodeMap["uuid"] = p.UUID
			if p.Flow != "" {
				nodeMap["flow"] = p.Flow
			}
		case "vmess":
			if p.UUID == "" {
				continue
			}
			nodeMap["type"] = "vmess"
			nodeMap["uuid"] = p.UUID
			nodeMap["security"] = "auto"
		case "trojan":
			nodeMap["type"] = "trojan"
			nodeMap["password"] = p.Password
		case "shadowsocks", "ss":
			cipher := strings.ToLower(p.Cipher)
			if !ValidSSCiphers[cipher] {
				valid = false
			} else if strings.HasPrefix(cipher, "2022-") {
				cleanPwd, ok := CleanSS2022Password(cipher, p.Password)
				if !ok {
					valid = false
				} else {
					nodeMap["type"] = "shadowsocks"
					nodeMap["method"] = p.Cipher
					nodeMap["password"] = cleanPwd
				}
			} else {
				nodeMap["type"] = "shadowsocks"
				nodeMap["method"] = p.Cipher
				nodeMap["password"] = p.Password
			}
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
			valid = false
		}

		if !valid || nodeMap["type"] == "" {
			log.Printf("[Singbox] Skipping invalid or unsupported proxy: %s (%s)", p.Name, pType)
			continue
		}
		nodeTags = append(nodeTags, p.Name)

		// Mandatory TLS protocols in Sing-box
		isTLSRequired := (pType == "hysteria2" || pType == "hy2" || pType == "tuic" || pType == "trojan" || pType == "anytls")
		if pType == "vless" && p.Flow != "" {
			isTLSRequired = true
		}

		if p.TLS || isTLSRequired {
			tlsMap := map[string]any{
				"enabled": true,
			}
			sni := p.SNI
			if sni == "" {
				sni = p.Server
			}
			tlsMap["server_name"] = sni

			if p.SkipCertVerify {
				tlsMap["insecure"] = true
			}
			if len(p.ALPN) > 0 {
				tlsMap["alpn"] = p.ALPN
			}

			// Reality opts
			if realityOpts, ok := p.RawConfig["reality-opts"].(map[string]any); ok {
				realityMap := map[string]any{
					"enabled": true,
				}
				if pk, ok := realityOpts["public-key"].(string); ok && pk != "" {
					realityMap["public_key"] = pk
				}
				if sid, ok := realityOpts["short-id"].(string); ok && sid != "" {
					realityMap["short_id"] = sid
				}
				tlsMap["reality"] = realityMap
			}

			// Client fingerprint (uTLS)
			if fp, ok := p.RawConfig["client-fingerprint"].(string); ok && fp != "" {
				tlsMap["utls"] = map[string]any{
					"enabled":     true,
					"fingerprint": fp,
				}
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
			} else {
				gOutbound["tolerance"] = 50
			}
			gOutbound["idle_timeout"] = "12h"
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
		"inbound": []string{"dns-in"},
		"action":  "hijack-dns",
	})
	routeRules = append(routeRules, map[string]any{
		"protocol": "dns",
		"action":   "hijack-dns",
	})
	routeRules = append(routeRules, map[string]any{
		"ip_is_private": true,
		"outbound":      "direct",
	})
	routeRules = append(routeRules, map[string]any{
		"network":  []string{"udp"},
		"port":     []int{123},
		"outbound": "direct",
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

	// C1.5. Intercept foreign UDP 443/8443 (QUIC) to prevent ISP UDP QoS throttling and force BBR TCP
	routeRules = append(routeRules, map[string]any{
		"network":  []string{"udp"},
		"port":     []int{443, 8443},
		"outbound": "block",
	})

	// C2. Application / Business routing groups for Sing-box
	if validOutbounds["Google"] {
		routeRules = append(routeRules, map[string]any{
			"domain_suffix": []string{"google.com", "googleapis.com", "gstatic.com", "googleusercontent.com", "googlevideo.com"},
			"outbound":      "Google",
		})
	}
	if validOutbounds["YouTube"] {
		routeRules = append(routeRules, map[string]any{
			"domain_suffix": []string{"youtube.com", "ytimg.com", "googlevideo.com", "youtu.be"},
			"outbound":      "YouTube",
		})
	}
	if validOutbounds["AI"] {
		routeRules = append(routeRules, map[string]any{
			"domain_suffix": []string{"openai.com", "chatgpt.com", "anthropic.com", "claude.ai", "oaistatic.com", "oaiusercontent.com"},
			"outbound":      "AI",
		})
	}
	if validOutbounds["Telegram"] {
		routeRules = append(routeRules, map[string]any{
			"domain_suffix": []string{"telegram.org", "t.me", "telegram.me", "telegra.ph"},
			"outbound":      "Telegram",
		})
	}
	if validOutbounds["GitHub & Docker"] {
		routeRules = append(routeRules, map[string]any{
			"domain_suffix": []string{"github.com", "githubusercontent.com", "docker.com", "docker.io"},
			"outbound":      "GitHub & Docker",
		})
	}
	if validOutbounds["Twitter (X)"] {
		routeRules = append(routeRules, map[string]any{
			"domain_suffix": []string{"twitter.com", "x.com", "twimg.com", "t.co"},
			"outbound":      "Twitter (X)",
		})
	}
	if validOutbounds["Netflix & 流媒体"] {
		routeRules = append(routeRules, map[string]any{
			"domain_suffix": []string{"netflix.com", "netflix.net", "nflximg.net", "nflxvideo.net", "nflxso.net", "nflxext.com"},
			"outbound":      "Netflix & 流媒体",
		})
	}
	if validOutbounds["Apple 服务"] {
		routeRules = append(routeRules, map[string]any{
			"domain_suffix": []string{"apple.com", "icloud.com", "aaplimg.com"},
			"outbound":      "Apple 服务",
		})
	}
	if validOutbounds["Microsoft"] {
		routeRules = append(routeRules, map[string]any{
			"domain_suffix": []string{"microsoft.com", "windows.com", "live.com", "office.com", "azure.com"},
			"outbound":      "Microsoft",
		})
	}

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

	// 5. Experimental Clash API & Cache DB (with persistent store_fakeip)
	cfg["experimental"] = map[string]any{
		"cache_file": map[string]any{
			"enabled":      true,
			"path":         "/app/etc/cache.db",
			"store_fakeip": true,
		},
		"clash_api": map[string]any{
			"external_controller": fmt.Sprintf("0.0.0.0:%d", net.ClashAPIPort),
			"secret":              net.ClashSecret,
			"default_mode":        string(ir.Meta.ProxyMode),
		},
	}

	return json.MarshalIndent(cfg, "", "  ")
}
