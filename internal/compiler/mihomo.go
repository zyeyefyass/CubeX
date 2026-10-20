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
	// TUN Configuration (Zero-copy kernel stack)
	cfg["tun"] = map[string]any{
		"enable":                   true,
		"stack":                    tunStack,
		"device":                   net.TunInterface,
		"auto-route":               true,
		"auto-redirect":            true,
		"auto-detect-interface":    true,
		"strict-route":             false,
		"mtu":                      tunMTU,
		"endpoint-independent-nat": true,
		"dns-hijack": []string{
			"tcp://any:53",
			"udp://any:53",
		},
	}

	// Sniffer (Optimized for zero Fake-IP corruption & preserving original destination)
	cfg["sniffer"] = map[string]any{
		"enable":               true,
		"force-dns-mapping":    true,
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

	// Profile for persistent Fake-IP & selection storage
	cfg["profile"] = map[string]any{
		"store-selected": true,
		"store-fake-ip":  true,
	}

	// Determine outbound proxy group for remote DoH queries
	foreignGroup := "🌐 国外流量"
	for _, g := range ir.Groups {
		if g.Name == "🌐 国外流量" || g.Name == "🚀 国外" || g.Name == "PROXY" || g.Name == "🐟 漏网之鱼" {
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
		fmt.Sprintf("https://dns.google/dns-query#%s&disable-qtype-65=true", foreignGroup),
		fmt.Sprintf("https://cloudflare-dns.com/dns-query#%s&disable-qtype-65=true", foreignGroup),
		fmt.Sprintf("tls://8.8.8.8#%s", foreignGroup),
	}
	fallbackDNSList := []string{
		fmt.Sprintf("https://dns.google/dns-query#%s&disable-qtype-65=true", foreignGroup),
		fmt.Sprintf("https://cloudflare-dns.com/dns-query#%s&disable-qtype-65=true", foreignGroup),
	}

	cleanFakeIPFilter := []string{
		"geosite:private",
		"rule-set:cn_domain",
		"*.lan",
		"*.local",
		"*.home",
		"*.internal",
		"localhost",
		"+.cn",
		"+.baidu.com",
		"+.qq.com",
		"+.bilibili.com",
		"+.taobao.com",
		"+.jd.com",
		"+.alipay.com",
		"+.163.com",
		"+.zhihu.com",
		"+.douyin.com",
		"+.weibo.com",
		"+.aliyun.com",
		"+.tencent.com",
		"+.bytedance.com",
		"+.msftconnecttest.com",
		"+.msftncsi.com",
		"+.routeros.com",
		"+.mikrotik.com",
	}

	dnsMap := map[string]any{
		"enable":             true,
		"cache-algorithm":    "arc",
		"cache-size":         8192,
		"listen":             fmt.Sprintf("0.0.0.0:%d", dnsPort),
		"ipv6":               net.IPv6,
		"enhanced-mode":      "fake-ip",
		"fake-ip-range":      dns.FakeIPRange,
		"fake-ip-filter":     cleanFakeIPFilter,
		"default-nameserver": []string{
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
		"respect-rules":     true,
		"use-hosts":         true,
		"use-system-hosts":  false,
		"fallback-filter": map[string]any{
			"geoip":      true,
			"geoip-code": "CN",
			"ipcidr": []string{
				"240.0.0.0/4",
			},
		},
		"nameserver-policy": map[string]any{
			"+.google.com,+.googleapis.com,+.gstatic.com,+.googlevideo.com,+.ytimg.com,+.youtube.com,+.openai.com,+.chatgpt.com,+.anthropic.com,+.claude.ai": remoteDNSList,
			"geosite:cn,private": []string{
				"https://dns.alidns.com/dns-query#disable-qtype-65=true",
				"https://doh.pub/dns-query#disable-qtype-65=true",
			},
			"rule-set:cn_domain,privacy_domain": []string{
				"https://dns.alidns.com/dns-query#disable-qtype-65=true",
				"https://doh.pub/dns-query#disable-qtype-65=true",
			},
			"geosite:geolocation-!cn": remoteDNSList,
		},
	}
	cfg["dns"] = dnsMap

	// Static DNS bootstrap hosts mapping (avoids circular resolution dependency)
	cfg["hosts"] = map[string]any{
		"dns.google":         []string{"8.8.8.8", "8.8.4.4"},
		"cloudflare-dns.com": []string{"1.1.1.1", "1.0.0.1"},
		"dns.alidns.com":     []string{"223.5.5.5", "223.6.6.6"},
		"doh.pub":            []string{"1.12.12.12", "120.53.53.53"},
	}

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
				if p.Network == "grpc" {
					pMap["grpc-opts"] = map[string]any{"grpc-service-name": p.Path}
				} else {
					pMap["ws-opts"] = map[string]any{"path": p.Path}
				}
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
	cnOutbound := "DIRECT"
	if validTargets["🎯 国内流量"] {
		cnOutbound = "🎯 国内流量"
	}
	rulesList = append(rulesList, fmt.Sprintf("RULE-SET,cn_domain,%s", cnOutbound))
	rulesList = append(rulesList, fmt.Sprintf("RULE-SET,cn_ip,%s", cnOutbound))

	// Intercept foreign UDP 443/8443 (QUIC) to prevent ISP UDP QoS throttling and force BBR TCP
	rulesList = append(rulesList, "AND,((NETWORK,udp),(OR,((DST-PORT,443),(DST-PORT,8443)))),REJECT")

	// High-Priority Line-Speed Benchmark / Speedtest Fast-Path Rule
	if validTargets[foreignGroup] {
		rulesList = append(rulesList, fmt.Sprintf("DOMAIN,speed.cloudflare.com,%s", foreignGroup))
		rulesList = append(rulesList, fmt.Sprintf("DOMAIN,challenges.cloudflare.com,%s", foreignGroup))
		rulesList = append(rulesList, fmt.Sprintf("DOMAIN-SUFFIX,fast.com,%s", foreignGroup))
		rulesList = append(rulesList, fmt.Sprintf("DOMAIN-SUFFIX,speedtest.net,%s", foreignGroup))
		rulesList = append(rulesList, fmt.Sprintf("DOMAIN-SUFFIX,ooklaserver.net,%s", foreignGroup))
	}

	// 2a. YouTube Video Streaming High-Speed Direct Rules
	if validTargets["YouTube"] {
		rulesList = append(rulesList, "DOMAIN-SUFFIX,googlevideo.com,YouTube")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,ytimg.com,YouTube")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,youtube.com,YouTube")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,youtu.be,YouTube")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,ggpht.com,YouTube")
		rulesList = append(rulesList, "DOMAIN-KEYWORD,googlevideo,YouTube")
		rulesList = append(rulesList, "DOMAIN-KEYWORD,sponsorblock,YouTube")
		rulesList = append(rulesList, "DOMAIN,youtubei.googleapis.com,YouTube")
	}

	// 2a-2. AI (OpenAI / Claude / Google Gemini / Antigravity / DeepMind)
	if validTargets["AI"] {
		rulesList = append(rulesList, "DOMAIN,generativelanguage.googleapis.com,AI")
		rulesList = append(rulesList, "DOMAIN,daily-cloudcode-pa.googleapis.com,AI")
		rulesList = append(rulesList, "DOMAIN,cloudcode-pa.googleapis.com,AI")
		rulesList = append(rulesList, "DOMAIN,aiplatform.googleapis.com,AI")
		rulesList = append(rulesList, "DOMAIN,alkalimakersuite-pa.clients6.google.com,AI")
		rulesList = append(rulesList, "DOMAIN,aistudio.google.com,AI")
		rulesList = append(rulesList, "DOMAIN,makersuite.google.com,AI")
		rulesList = append(rulesList, "DOMAIN,bard.google.com,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,deepmind.google,AI")
	}

	// 2b. High-Speed Business Service Rules (Native Domain-Suffix & GEOSITE)
	bizRules := []struct {
		group    string
		suffixes []string
		geosite  string
	}{
		{"AI", []string{"openai.com", "chatgpt.com", "anthropic.com", "claude.ai", "oaistatic.com", "oaiusercontent.com", "perplexity.ai", "gemini.google.com", "deepmind.google", "x.ai", "grok.com"}, "openai"},
		{"YouTube", []string{"youtube.com", "ytimg.com", "googlevideo.com", "youtu.be", "youtubei.googleapis.com"}, "youtube"},
		{"Google", []string{"google.com", "googleapis.com", "gstatic.com", "googleusercontent.com", "ggpht.com"}, "google"},
		{"GitHub & Docker", []string{"github.com", "githubusercontent.com", "docker.com", "docker.io", "ghcr.io"}, "github"},
		{"Telegram", []string{"telegram.org", "t.me", "telegram.me", "telegra.ph"}, "telegram"},
		{"Twitter (X)", []string{"twitter.com", "x.com", "twimg.com", "t.co"}, "twitter"},
		{"Netflix & 流媒体", []string{"netflix.com", "netflix.net", "nflximg.net", "nflxvideo.net", "nflxso.net", "nflxext.com", "disneyplus.com", "hbo.com"}, "netflix"},
		{"Spotify", []string{"spotify.com", "scdn.co", "spoti.fi"}, "spotify"},
		{"Steam 游戏", []string{"steampowered.com", "steamcommunity.com", "steamgames.com", "steamusercontent.com", "steamcontent.com"}, "steam"},
		{"Apple 服务", []string{"apple.com", "icloud.com", "aaplimg.com", "itunes.com"}, "apple"},
		{"Microsoft", []string{"microsoft.com", "windows.com", "live.com", "office.com", "azure.com", "msftconnecttest.com"}, "microsoft"},
	}

	for _, br := range bizRules {
		if validTargets[br.group] {
			for _, s := range br.suffixes {
				rulesList = append(rulesList, fmt.Sprintf("DOMAIN-SUFFIX,%s,%s", s, br.group))
			}
			if br.geosite != "" {
				rulesList = append(rulesList, fmt.Sprintf("GEOSITE,%s,%s", br.geosite, br.group))
			}
			if br.group == "Telegram" {
				rulesList = append(rulesList, "GEOIP,telegram,Telegram")
			}
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
