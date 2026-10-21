package compiler

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"runtime"
	"strconv"
	"strings"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

func detectDefaultInterface() string {
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
				continue
			}
			name := strings.ToLower(iface.Name)
			if strings.HasPrefix(name, "tun") || strings.HasPrefix(name, "cubex") || strings.HasPrefix(name, "wg") {
				continue
			}
			addrs, _ := iface.Addrs()
			if len(addrs) > 0 {
				return iface.Name
			}
		}
	}
	return "veth-cubex"
}

func isIPAddress(s string) bool {
	return net.ParseIP(s) != nil
}

func makeSingboxDNSServer(tag, rawAddr, detour string) map[string]any {
	serverMap := map[string]any{
		"tag": tag,
	}
	if detour != "" && detour != "direct" && detour != "DIRECT" {
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
		serverMap["server_port"] = 853
	} else if strings.HasPrefix(rawAddr, "tcp://") {
		serverMap["type"] = "tcp"
		serverMap["server"] = strings.TrimPrefix(rawAddr, "tcp://")
		serverMap["server_port"] = 53
	} else {
		serverMap["type"] = "udp"
		if detour != "" && detour != "direct" && detour != "DIRECT" {
			serverMap["type"] = "tcp"
		}
		serverMap["server"] = rawAddr
		serverMap["server_port"] = 53
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

	// Root config map with optimized log throttling
	// Default to "warn" to eliminate per-packet and per-DNS stdout flood that exhausts RouterOS container log ring buffer
	singboxLogLevel := "warn"
	if ir.Meta.LogLevel == "debug" || ir.Meta.LogLevel == "trace" {
		singboxLogLevel = ir.Meta.LogLevel
	} else if ir.Meta.LogLevel == "error" || ir.Meta.LogLevel == "fatal" || ir.Meta.LogLevel == "panic" {
		singboxLogLevel = ir.Meta.LogLevel
	}

	cfg := map[string]any{
		"log": map[string]any{
			"level":     singboxLogLevel,
			"timestamp": true,
		},
	}

	// 1. DNS Section (v1.14+ compliant)
	directDNSAddr := "223.5.5.5"
	if len(dns.DirectDNS) > 0 {
		directDNSAddr = dns.DirectDNS[0]
	}
	remoteDNSAddr := "tcp://1.1.1.1"
	if len(dns.RemoteDNS) > 0 {
		remoteDNSAddr = dns.RemoteDNS[0]
	}

	// Find foreign outbound group tag for remote DNS tunnel detour
	foreignDetour := "🌐 国外流量"
	for _, g := range ir.Groups {
		if g.Name == "🌐 国外流量" || g.Name == "🚀 国外" || g.Name == "PROXY" || g.Name == "🐟 漏网之鱼" {
			foreignDetour = g.Name
			break
		}
	}

	directServer := makeSingboxDNSServer("dns-direct", directDNSAddr, "direct")
	if directServer["type"] == "udp" && directServer["server_port"] == nil {
		directServer["server_port"] = 53
	}
	remoteServer := makeSingboxDNSServer("dns-remote", remoteDNSAddr, foreignDetour)
	if remoteServer["type"] == "https" && remoteServer["server_port"] == nil {
		remoteServer["server_port"] = 443
	}

	// Collect all proxy server domains & camouflage SNIs so they NEVER resolve to Fake-IP
	proxyDirectDomains := make([]string, 0)
	proxyDirectDomainMap := make(map[string]bool)
	for _, p := range ir.Proxies {
		if p.Server != "" && !isIPAddress(p.Server) && !proxyDirectDomainMap[p.Server] {
			proxyDirectDomainMap[p.Server] = true
			proxyDirectDomains = append(proxyDirectDomains, p.Server)
		}
		if p.SNI != "" && !isIPAddress(p.SNI) && !proxyDirectDomainMap[p.SNI] {
			proxyDirectDomainMap[p.SNI] = true
			proxyDirectDomains = append(proxyDirectDomains, p.SNI)
		}
	}

	cfg["dns"] = map[string]any{
		"servers": []map[string]any{
			{
				"tag":  "dns-local",
				"type": "local",
			},
			{
				"tag":  "dns-hosts",
				"type": "hosts",
				"predefined": map[string]any{
					"dns.alidns.com":     []string{"223.5.5.5", "223.6.6.6"},
					"doh.pub":            []string{"1.12.12.12", "120.53.53.53"},
					"dns.google":         []string{"8.8.8.8", "8.8.4.4"},
					"cloudflare-dns.com": []string{"1.1.1.1", "1.0.0.1"},
					"1.1.1.1":            []string{"1.1.1.1"},
					"8.8.8.8":            []string{"8.8.8.8"},
				},
			},
			{
				"tag":         "dns-bootstrap",
				"type":        "https",
				"server":      "dns.alidns.com",
				"server_port": 443,
				"path":        "/dns-query",
				"detour":      "direct",
			},
			directServer,
			remoteServer,
			{
				"tag":         "dns-fakeip",
				"type":        "fakeip",
				"inet4_range": "198.18.0.0/16",
			},
		},
		"rules": []map[string]any{
			{
				"domain_suffix": []string{".lan", ".local", ".home", ".internal", ".home.arpa"},
				"server":        "dns-local",
			},
			{
				"domain_regex": []string{"^[^.]+$"},
				"server":       "dns-local",
			},
			{
				"domain": proxyDirectDomains,
				"server": "dns-bootstrap",
			},
			{
				"query_type": []string{"HTTPS", "SVCB"},
				"action":     "reject",
			},
			{
				"clash_mode": "direct",
				"server":     "dns-direct",
			},
			{
				"clash_mode": "global",
				"server":     "dns-fakeip",
			},
			// 1. Google / YouTube / GStatic / Media / AI Strict Fake-IP Protection (Overrides geosite-cn to prevent DNS leak/pollution)
			{
				"domain_suffix": []string{
					"google.com",
					"googleapis.com",
					"gstatic.com",
					"googleusercontent.com",
					"googlevideo.com",
					"ggpht.com",
					"youtube.com",
					"ytimg.com",
					"youtu.be",
					"youtubei.googleapis.com",
					"cloudflare.com",
					"openai.com",
					"chatgpt.com",
					"anthropic.com",
					"claude.ai",
					"deepmind.google",
					"antigravity-unleash.goog",
					"antigravity.dev",
					"goog",
					"run.app",
					"1e100.net",
					"gvt1.com",
				},
				"server": "dns-fakeip",
			},
			{
				"domain_suffix": []string{
					".cn",
					"bilibili.com",
					"baidu.com",
					"qq.com",
					"taobao.com",
					"jd.com",
					"alipay.com",
					"163.com",
					"msftconnecttest.com",
				},
				"rule_set": []string{"geosite-cn"},
				"server":   "dns-direct",
			},
			{
				"query_type": []string{"A", "AAAA"},
				"server":     "dns-fakeip",
			},
		},
		"final":           "dns-remote",
		"reverse_mapping": true,
		"strategy":        "prefer_ipv4",
		"cache_capacity":    8192,
		"optimistic": map[string]any{
			"enabled": true,
		},
	}

	tunMTU := net.TunMTU
	if tunMTU <= 0 {
		tunMTU = 1500
	}

	tunStack := net.TunStack
	if tunStack == "" {
		tunStack = "mixed"
	}

	tproxyPort := net.TProxyPort
	if tproxyPort == 0 {
		tproxyPort = 7893
	}
	redirPort := net.RedirPort
	if redirPort == 0 {
		redirPort = 7892
	}
	dnsPort := net.DNSPort
	if dnsPort == 0 || dnsPort == 1053 {
		dnsPort = 53
	}

	inbounds := make([]map[string]any, 0)
	if runtime.GOOS != "windows" {
		inbounds = append(inbounds,
			map[string]any{
				"type":           "tun",
				"tag":            "tun-in",
				"interface_name": net.TunInterface,
				"address":        []string{net.TunIPv4},
				"mtu":            tunMTU,
				"auto_route":     true,
				"auto_redirect":  false,
				"strict_route":   false,
				"stack":                     tunStack,
				"udp_timeout":               "60s",
				"endpoint_independent_nat":  true,
				"route_exclude_address": []string{
					"10.0.0.0/8",
					"100.64.0.0/10",
					"169.254.0.0/16",
					"172.16.0.0/12",
					"192.168.0.0/16",
					"224.0.0.0/4",
				},
			},
			map[string]any{
				"type":        "tproxy",
				"tag":         "tproxy-in",
				"listen":      "0.0.0.0",
				"listen_port": tproxyPort,
				"network":     []string{"tcp", "udp"},
			},
			map[string]any{
				"type":        "redirect",
				"tag":         "redir-in",
				"listen":      "0.0.0.0",
				"listen_port": redirPort,
			},
		)
	}

	inbounds = append(inbounds,
		map[string]any{
			"type":        "mixed",
			"tag":         "mixed-in",
			"listen":      "0.0.0.0",
			"listen_port": net.MixedPort,
		},
		map[string]any{
			"type":        "direct",
			"tag":         "dns-in",
			"listen":      "0.0.0.0",
			"listen_port": dnsPort,
			"network":     []string{"tcp", "udp"},
		},
	)
	cfg["inbounds"] = inbounds

	// 3. Outbounds Section
	outbounds := make([]map[string]any, 0)

	// Built-in fixed endpoints
	outbounds = append(outbounds, map[string]any{
		"type":          "direct",
		"tag":           "direct",
		"tcp_fast_open": true,
	})
	outbounds = append(outbounds, map[string]any{
		"type":          "direct",
		"tag":           "DIRECT",
		"tcp_fast_open": true,
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
			} else {
				nodeMap["packet_encoding"] = "xudp"
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

			// Shadow-TLS plugin support in Sing-box (via detour to companion shadowtls outbound)
			if valid && p.RawConfig != nil {
				if plugin, ok := p.RawConfig["plugin"].(string); ok && plugin == "shadow-tls" {
					if pluginOpts, ok := p.RawConfig["plugin-opts"].(map[string]any); ok {
						stlsTag := p.Name + "-shadowtls"
						stlsMap := map[string]any{
							"type":        "shadowtls",
							"tag":         stlsTag,
							"server":      p.Server,
							"server_port": p.Port,
						}
						if pwd, ok := pluginOpts["password"].(string); ok {
							stlsMap["password"] = pwd
						}
						if ver, ok := pluginOpts["version"].(int); ok {
							stlsMap["version"] = ver
						} else {
							stlsMap["version"] = 3
						}
						stlsHost := p.Server
						if h, ok := pluginOpts["host"].(string); ok && h != "" {
							stlsHost = h
						}
						tlsObj := map[string]any{
							"enabled":     true,
							"server_name": stlsHost,
						}
						if fp, ok := p.RawConfig["client-fingerprint"].(string); ok && fp != "" {
							tlsObj["utls"] = map[string]any{
								"enabled":     true,
								"fingerprint": fp,
							}
						}
						stlsMap["tls"] = tlsObj
						outbounds = append(outbounds, stlsMap)

						nodeMap["detour"] = stlsTag
						delete(nodeMap, "server")
						delete(nodeMap, "server_port")
					}
				}
			}
		case "hysteria2", "hy2":
			nodeMap["type"] = "hysteria2"
			nodeMap["password"] = p.Password
			if p.RawConfig != nil {
				if up := parseBandwidthMbps(p.RawConfig["up"]); up > 0 {
					nodeMap["up_mbps"] = up
				}
				if down := parseBandwidthMbps(p.RawConfig["down"]); down > 0 {
					nodeMap["down_mbps"] = down
				}
			}
		case "tuic":
			nodeMap["type"] = "tuic"
			nodeMap["uuid"] = p.UUID
			nodeMap["password"] = p.Password
			nodeMap["congestion_control"] = "bbr"
			if p.RawConfig != nil {
				if cc, ok := p.RawConfig["congestion-controller"].(string); ok && cc != "" {
					nodeMap["congestion_control"] = cc
				}
				if urm, ok := p.RawConfig["udp-relay-mode"].(string); ok && urm != "" {
					nodeMap["udp_relay_mode"] = urm
				}
				if rtt, ok := p.RawConfig["reduce-rtt"].(bool); ok {
					nodeMap["zero_rtt_handshake"] = rtt
				}
			}
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
			// ALPN handling
			var alpnList []string
			if len(p.ALPN) > 0 {
				alpnList = p.ALPN
			} else if p.RawConfig != nil {
				if rawAlpn, ok := p.RawConfig["alpn"].([]any); ok {
					for _, a := range rawAlpn {
						if as, ok := a.(string); ok && as != "" {
							alpnList = append(alpnList, as)
						}
					}
				} else if rawAlpnStr, ok := p.RawConfig["alpn"].([]string); ok {
					alpnList = rawAlpnStr
				}
			}
			if len(alpnList) == 0 && (pType == "tuic" || pType == "hysteria2" || pType == "hy2") {
				alpnList = []string{"h3"}
			}
			if len(alpnList) > 0 {
				tlsMap["alpn"] = alpnList
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

			// 1. Client fingerprint (uTLS)
			clientFp := ""
			if val, ok := p.RawConfig["client-fingerprint"].(string); ok && val != "" {
				clientFp = val
			} else if val, ok := p.RawConfig["fingerprint"].(string); ok && val != "" {
				if !strings.Contains(val, ":") && len(val) != 64 {
					clientFp = val
				}
			}
			if clientFp != "" {
				tlsMap["utls"] = map[string]any{
					"enabled":     true,
					"fingerprint": clientFp,
				}
			}

			// 2. Certificate Fingerprint Pinning / Self-Signed Bypass
			certFp := ""
			if val, ok := p.RawConfig["fingerprint"].(string); ok && val != "" {
				if strings.Contains(val, ":") || len(val) == 64 {
					certFp = val
				}
			} else if val, ok := p.RawConfig["certificate-sha256"].(string); ok && val != "" {
				certFp = val
			}
			if certFp != "" {
				// Custom / self-signed certificate fingerprint detected in subscription.
				// In sing-box, certificate_public_key_sha256 expects SPKI hash (not x509 cert hash).
				// We set insecure=true so sing-box accepts the custom certificate without error.
				tlsMap["insecure"] = true
			}

			nodeMap["tls"] = tlsMap
		}

		if p.Network == "ws" {
			wsMap := map[string]any{
				"type": "ws",
			}
			wsPath := p.Path
			if wsOpts, ok := p.RawConfig["ws-opts"].(map[string]any); ok {
				if path, ok := wsOpts["path"].(string); ok && path != "" {
					wsPath = path
				}
				if headers, ok := wsOpts["headers"].(map[string]any); ok && len(headers) > 0 {
					wsHeaders := make(map[string]string)
					for k, v := range headers {
						wsHeaders[k] = fmt.Sprintf("%v", v)
					}
					wsMap["headers"] = wsHeaders
				}
			}
			if wsPath != "" {
				wsMap["path"] = wsPath
			}
			nodeMap["transport"] = wsMap
		} else if p.Network == "grpc" {
			grpcMap := map[string]any{
				"type": "grpc",
			}
			serviceName := p.Path
			if serviceName == "" {
				if grpcOpts, ok := p.RawConfig["grpc-opts"].(map[string]any); ok {
					if sn, ok := grpcOpts["grpc-service-name"].(string); ok {
						serviceName = sn
					}
				}
			}
			if serviceName != "" {
				grpcMap["service_name"] = serviceName
			}
			nodeMap["transport"] = grpcMap
		}

		outbounds = append(outbounds, nodeMap)
	}

	// Build set of valid outbounds (built-ins + user nodes + group tags)
	validOutbounds := make(map[string]bool)
	validOutbounds["direct"] = true
	validOutbounds["DIRECT"] = true
	validOutbounds["block"] = true
	validOutbounds["REJECT"] = true
	for _, ob := range outbounds {
		if tag, ok := ob["tag"].(string); ok && tag != "" {
			validOutbounds[tag] = true
		}
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
		"port":   []int{53, dnsPort},
		"action": "hijack-dns",
	})
	routeRules = append(routeRules, map[string]any{
		"protocol": "dns",
		"action":   "hijack-dns",
	})

	// B. QUIC & STUN Fast Reject (Immediate RST/ICMP rejection without packet drop stall)
	routeRules = append(routeRules, map[string]any{
		"port":     []int{443, 8443, 853},
		"network":  []string{"udp"},
		"action":   "reject",
		"no_drop":  true,
	})

	// C. Direct Private IP & NTP (Port 123)
	routeRules = append(routeRules, map[string]any{
		"ip_is_private": true,
		"outbound":      "direct",
	})
	routeRules = append(routeRules, map[string]any{
		"network":  []string{"udp"},
		"port":     []int{123},
		"outbound": "direct",
	})

	// High-Priority Line-Speed Benchmark / Speedtest Rule
	if foreignDetour != "" {
		routeRules = append(routeRules, map[string]any{
			"domain": []string{
				"speed.cloudflare.com",
				"challenges.cloudflare.com",
			},
			"domain_suffix": []string{
				"fast.com",
				"speedtest.net",
				"ooklaserver.net",
			},
			"outbound": foreignDetour,
		})
	}

	// C. Node Server Domains & IPs Anti-Looping Bypass
	nodeDomains := make([]string, 0)
	nodeIPs := make([]string, 0)
	for _, p := range ir.Proxies {
		if p.Server == "" {
			continue
		}
		if strings.Contains(p.Server, ":") || (strings.Count(p.Server, ".") == 3 && !strings.ContainsAny(p.Server, "abcdefghijklmnopqrstuvwxyz")) {
			cidr := p.Server
			if !strings.Contains(cidr, "/") {
				cidr = cidr + "/32"
			}
			nodeIPs = append(nodeIPs, cidr)
		} else {
			nodeDomains = append(nodeDomains, p.Server)
		}
		if p.SNI != "" && !isIPAddress(p.SNI) {
			nodeDomains = append(nodeDomains, p.SNI)
		}
	}
	if len(nodeDomains) > 0 || len(nodeIPs) > 0 {
		nodeRule := map[string]any{
			"outbound": "direct",
		}
		if len(nodeDomains) > 0 {
			nodeRule["domain"] = nodeDomains
		}
		if len(nodeIPs) > 0 {
			nodeRule["ip_cidr"] = nodeIPs
		}
		routeRules = append(routeRules, nodeRule)
	}

	// D. Device Policies (Top Priority)
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

	// E. Adaptive SRS Rule-Set Interception (Adblock & Anti-Tracking native sing-box)
	routeRules = append(routeRules, map[string]any{
		"rule_set": []string{"217heidai-ads", "privacy-tracking"},
		"action":   "reject",
	})

	// H. 11 Major Business Strategy Groups (Comprehensive Domain Matching)
	if validOutbounds["AI"] {
		routeRules = append(routeRules, map[string]any{
			"domain": []string{
				"antigravity-unleash.goog",
				"antigravity.dev",
				"generativelanguage.googleapis.com",
				"daily-cloudcode-pa.googleapis.com",
				"cloudcode-pa.googleapis.com",
				"aiplatform.googleapis.com",
				"alkalimakersuite-pa.clients6.google.com",
				"aistudio.google.com",
				"makersuite.google.com",
				"bard.google.com",
				"gemini.google.com",
				"deepmind.google",
			},
			"domain_suffix": []string{
				"antigravity.dev",
				"goog",
				"run.app",
				"openai.com",
				"chatgpt.com",
				"anthropic.com",
				"claude.ai",
				"oaistatic.com",
				"oaiusercontent.com",
				"perplexity.ai",
				"gemini.google.com",
				"deepmind.google",
				"x.ai",
				"grok.com",
			},
			"outbound": "AI",
		})
	}
	if validOutbounds["YouTube"] {
		routeRules = append(routeRules, map[string]any{
			"domain_suffix":  []string{"youtube.com", "ytimg.com", "googlevideo.com", "youtu.be", "youtubei.googleapis.com", "yt3.ggpht.com", "yt4.ggpht.com", "gvt1.com", "1e100.net", "googleusercontent.com"},
			"domain_keyword": []string{"googlevideo", "sponsorblock"},
			"outbound":       "YouTube",
		})
	}
	if validOutbounds["Google"] {
		routeRules = append(routeRules, map[string]any{
			"domain_suffix": []string{"google.com", "googleapis.com", "gstatic.com", "googleusercontent.com", "ggpht.com", "antigravity.dev", "goog", "run.app", "1e100.net", "gvt1.com"},
			"outbound":      "Google",
		})
	}
	if validOutbounds["GitHub & Docker"] {
		routeRules = append(routeRules, map[string]any{
			"domain_suffix": []string{"github.com", "githubusercontent.com", "docker.com", "docker.io", "ghcr.io"},
			"outbound":      "GitHub & Docker",
		})
	}
	if validOutbounds["Telegram"] {
		routeRules = append(routeRules, map[string]any{
			"domain_suffix": []string{"telegram.org", "t.me", "telegram.me", "telegra.ph"},
			"outbound":      "Telegram",
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
			"domain_suffix": []string{"netflix.com", "netflix.net", "nflximg.net", "nflxvideo.net", "nflxso.net", "nflxext.com", "disneyplus.com", "hbo.com"},
			"outbound":      "Netflix & 流媒体",
		})
	}
	if validOutbounds["Apple 服务"] {
		routeRules = append(routeRules, map[string]any{
			"domain_suffix": []string{"apple.com", "icloud.com", "aaplimg.com", "itunes.com"},
			"outbound":      "Apple 服务",
		})
	}
	if validOutbounds["Microsoft"] {
		routeRules = append(routeRules, map[string]any{
			"domain_suffix": []string{"microsoft.com", "windows.com", "live.com", "office.com", "azure.com", "msftconnecttest.com"},
			"outbound":      "Microsoft",
		})
	}
	if validOutbounds["Spotify"] {
		routeRules = append(routeRules, map[string]any{
			"domain_suffix": []string{"spotify.com", "scdn.co", "spoti.fi"},
			"outbound":      "Spotify",
		})
	}
	if validOutbounds["Steam 游戏"] {
		routeRules = append(routeRules, map[string]any{
			"domain_suffix": []string{"steampowered.com", "steamcommunity.com", "steamgames.com", "steamusercontent.com", "steamcontent.com"},
			"outbound":      "Steam 游戏",
		})
	}

	// F. Domestic Direct Routing (geosite-cn + Pure-IP GeoIP Isolation)
	domesticTarget := "direct"
	if validOutbounds["🎯 国内流量"] {
		domesticTarget = "🎯 国内流量"
	}
	routeRules = append(routeRules, map[string]any{
		"rule_set": []string{"geosite-cn"},
		"outbound": domesticTarget,
	})
	routeRules = append(routeRules, map[string]any{
		"type":   "logical",
		"mode":   "and",
		"invert": false,
		"rules": []map[string]any{
			{
				"domain_regex": []string{"^[^:]*[a-z][^:]*$"},
				"invert":       true,
			},
			{
				"rule_set": []string{"geoip-cn"},
			},
		},
		"outbound": domesticTarget,
	})

	// I. Custom / User Defined Generic Rules (Deduplicated)
	for _, r := range ir.Rules {
		rType := strings.ToLower(r.Type)
		targetOutbound := r.Target
		if !validOutbounds[targetOutbound] {
			if strings.Contains(targetOutbound, "拦截") || strings.EqualFold(targetOutbound, "REJECT") || strings.EqualFold(targetOutbound, "block") {
				targetOutbound = "block"
			} else if strings.Contains(targetOutbound, "直连") || strings.EqualFold(targetOutbound, "DIRECT") || strings.EqualFold(targetOutbound, "direct") {
				targetOutbound = "direct"
			} else if validOutbounds["🌐 国外流量"] {
				targetOutbound = "🌐 国外流量"
			} else if validOutbounds["🐟 漏网之鱼"] {
				targetOutbound = "🐟 漏网之鱼"
			} else {
				targetOutbound = "direct"
			}
		}

		// Deduplicate built-in rules
		val := strings.ToLower(r.Value)
		if (rType == "geosite" && (val == "cn" || strings.Contains(val, "ad"))) ||
			(rType == "geoip" && (val == "cn" || val == "private")) ||
			(rType == "port" && (val == "443" || val == "80" || val == "123")) ||
			rType == "match" || rType == "final" {
			continue
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
		case "ip_cidr", "ip-cidr":
			ruleMap["ip_cidr"] = []string{r.Value}
		case "src_ip_cidr", "src-ip-cidr":
			ruleMap["source_ip_cidr"] = []string{r.Value}
		case "rule_set", "rule-set":
			ruleMap["rule_set"] = []string{r.Value}
		}

		if len(ruleMap) > 1 {
			routeRules = append(routeRules, ruleMap)
		}
	}

	finalOutbound := "direct"
	if validOutbounds["🌐 国外流量"] {
		finalOutbound = "🌐 国外流量"
	} else if validOutbounds["PROXY"] {
		finalOutbound = "PROXY"
	} else if validOutbounds["🐟 漏网之鱼"] {
		finalOutbound = "🐟 漏网之鱼"
	}

	cfg["route"] = map[string]any{
		"auto_detect_interface":   false,
		"default_interface":       detectDefaultInterface(),
		"default_domain_resolver": "dns-direct",
		"rule_set":                ruleSets,
		"rules":                   routeRules,
		"final":                   finalOutbound,
	}

	// 5. Experimental Clash API & Cache DB (with persistent store_fakeip)
	cfg["experimental"] = map[string]any{
		"cache_file": map[string]any{
			"enabled":      true,
			"path":         "cache.db",
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


func parseBandwidthMbps(val any) int {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		v = strings.TrimSpace(strings.ToLower(v))
		v = strings.TrimSuffix(v, "mbps")
		v = strings.TrimSuffix(v, "mb/s")
		v = strings.TrimSuffix(v, "m")
		v = strings.TrimSpace(v)
		var n int
		fmt.Sscanf(v, "%d", &n)
		return n
	}
	return 0
}
