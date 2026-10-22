package compiler

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/zyeyefyass/CubeX/internal/schema"
	"gopkg.in/yaml.v3"
)

// isMRSAvailable checks whether a compiled binary Mihomo rule set exists on disk.
func isMRSAvailable(ruleFilename string) bool {
	clean := strings.TrimPrefix(ruleFilename, "./")
	clean = strings.TrimPrefix(clean, "/")
	candidates := []string{
		filepath.Join("/cubex/app/etc", clean),
		filepath.Join("etc", clean),
		clean,
		ruleFilename,
		filepath.Join("..", "etc", clean),
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.Size() > 50 {
			return true
		}
	}
	return false
}

// BuildMihomo transcompiles CubeX Universal IR to production Mihomo config.yaml
func BuildMihomo(ir *schema.ConfigIR) ([]byte, error) {
	if ir == nil {
		return nil, fmt.Errorf("config IR is nil")
	}
	SanitizeIR(ir)

	net := ir.Network
	dns := ir.DNS

	logLevel := "info"


	if ir.Meta.LogLevel != "" {


		logLevel = ir.Meta.LogLevel


	}



	cfg := map[string]any{


		"mixed-port":          net.MixedPort,


		"redir-port":          net.RedirPort,


		"tproxy-port":         net.TProxyPort,


		"allow-lan":           net.AllowLAN,


		"mode":                string(ir.Meta.ProxyMode),


		"log-level":           logLevel,
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
	if tunMTU <= 0 || tunMTU > 1400 {
		tunMTU = 1400
	}

	tunStack := net.TunStack
	if tunStack == "" {
		tunStack = "mixed"
	}

	// TUN Configuration (Zero-copy gvisor/mixed stack with local subnet bypass)
	cfg["tun"] = map[string]any{
		"enable":                   true,
		"stack":                    tunStack,
		"device":                   net.TunInterface,
		"auto-route":               true,
		"auto-redirect":            false,
		"auto-detect-interface":    true,
		"strict-route":             false,
		"dns-mode":                 "hijack",
		"mtu":                      tunMTU,
		"gso":                      true,
		"gso-max-size":             65536,
		"endpoint-independent-nat": true,
		"route-exclude-address": []string{
			"10.0.0.0/8",
			"100.64.0.0/10",
			"127.0.0.0/8",
			"169.254.0.0/16",
			"172.16.0.0/12",
			"192.168.0.0/16",
			"224.0.0.0/4",
		},
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
		"override-destination": true,
		"sniff": map[string]any{
			"HTTP": map[string]any{"ports": []any{80, "8080-8880"}},
			"TLS":  map[string]any{"ports": []int{443, 8443}},
			"QUIC": map[string]any{"ports": []int{443, 8443}},
		},
		"skip-domain": []string{
			"dlg.io.mi.com",
			"+.mijia.tech",
			"+.io.mi.com",
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

	fallbackDNSList := []string{
		"223.5.5.5",
		"119.29.29.29",
	}

	cleanFakeIPFilter := []string{
		"geosite:private",
		"geosite:cn",
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
			"223.5.5.5",
			"119.29.29.29",
		},
		"direct-nameserver": []string{
			"223.5.5.5",
			"119.29.29.29",
		},
		"nameserver": []string{
			"223.5.5.5",
			"119.29.29.29",
		},
		"fallback":          fallbackDNSList,
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
			"geosite:cn,private": []string{
				"223.5.5.5",
				"119.29.29.29",
			},
			"geosite:cn": []string{
				"223.5.5.5",
				"119.29.29.29",
			},
			"geosite:private": []string{
				"223.5.5.5",
				"119.29.29.29",
			},
		},
	}
	cfg["dns"] = dnsMap

	// Static DNS bootstrap hosts mapping & DNS Rewrite rules
	hostsMap := map[string]any{
		"dns.google":         []string{"8.8.8.8", "8.8.4.4"},
		"cloudflare-dns.com": []string{"1.1.1.1", "1.0.0.1"},
		"dns.alidns.com":     []string{"223.5.5.5", "223.6.6.6"},
		"doh.pub":            []string{"1.12.12.12", "120.53.53.53"},
	}
	if ir.DNSRewriteEnabled {
		for _, rw := range ir.DNSRewrites {
			if rw.Enabled && rw.Domain != "" && rw.Target != "" {
				hostsMap[rw.Domain] = rw.Target
			}
		}
	}
	cfg["hosts"] = hostsMap

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
			if p.Network == "grpc" {
				grpcOpts, ok := pMap["grpc-opts"].(map[string]any)
				if !ok {
					grpcOpts = make(map[string]any)
				}
				if p.Path != "" && grpcOpts["grpc-service-name"] == nil {
					grpcOpts["grpc-service-name"] = p.Path
				}
				pMap["grpc-opts"] = grpcOpts
			} else if p.Network == "ws" {
				wsOpts, ok := pMap["ws-opts"].(map[string]any)
				if !ok {
					wsOpts = make(map[string]any)
				}
				if p.Path != "" && wsOpts["path"] == nil {
					wsOpts["path"] = p.Path
				}
				pMap["ws-opts"] = wsOpts
			}
			if p.Flow != "" {
				pMap["flow"] = p.Flow
			}
		}
		pMap["name"] = p.Name
		pMap["server"] = p.Server
		pMap["port"] = p.Port
		pMap["udp"] = true
		ptypeLower := strings.ToLower(p.Type)
		if ptypeLower == "vless" || ptypeLower == "vmess" || ptypeLower == "trojan" || ptypeLower == "shadowsocks" || ptypeLower == "ss" {
			pMap["tfo"] = true
		}

		// Sanitize sing-box / raw config artifacts for Mihomo
		if tlsMap, ok := pMap["tls"].(map[string]any); ok {
			pMap["tls"] = true
			if en, exists := tlsMap["enabled"].(bool); exists && !en {
				delete(pMap, "tls")
			}
			if sni, exists := tlsMap["server_name"].(string); exists && sni != "" {
				pMap["servername"] = sni
			}
			if insec, exists := tlsMap["insecure"].(bool); exists && insec {
				pMap["skip-cert-verify"] = true
			}
			if alpn, exists := tlsMap["alpn"].([]any); exists && len(alpn) > 0 {
				pMap["alpn"] = alpn
			}
			if reality, exists := tlsMap["reality"].(map[string]any); exists {
				if rEn, ok := reality["enabled"].(bool); ok && rEn {
					realityOpts := make(map[string]any)
					if pk, ok := reality["public_key"].(string); ok && pk != "" {
						realityOpts["public-key"] = pk
					}
					if sid, ok := reality["short_id"].(string); ok && sid != "" {
						realityOpts["short-id"] = sid
					}
					pMap["reality-opts"] = realityOpts
				}
			}
			if utls, exists := tlsMap["utls"].(map[string]any); exists {
				if fp, ok := utls["fingerprint"].(string); ok && fp != "" {
					pMap["client-fingerprint"] = fp
				}
			}
		} else if _, ok := pMap["tls"].(bool); !ok {
			if p.TLS {
				pMap["tls"] = true
			} else {
				delete(pMap, "tls")
			}
		}

		if transport, ok := pMap["transport"].(map[string]any); ok {
			if tType, ok := transport["type"].(string); ok && tType != "" {
				pMap["network"] = tType
				if tType == "ws" {
					wsOpts := make(map[string]any)
					if path, ok := transport["path"].(string); ok && path != "" {
						wsOpts["path"] = path
					}
					if headers, ok := transport["headers"].(map[string]any); ok {
						wsOpts["headers"] = headers
					}
					pMap["ws-opts"] = wsOpts
				}
			}
			delete(pMap, "transport")
		}

		// Clean sing-box specific outbounds keys that crash Mihomo
		delete(pMap, "tag")
		delete(pMap, "server_port")
		delete(pMap, "utls")

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
				gMap["url"] = "https://cp.cloudflare.com/generate_204"
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

	// 217heidai MRS Adaptive Rule Providers for Mihomo (Full Dual-Engine Coverage)
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

	domesticGroup := "DIRECT"
	if validTargets["🎯 国内流量"] {
		domesticGroup = "🎯 国内流量"
	} else if validTargets["🎯 全球直连"] {
		domesticGroup = "🎯 全球直连"
	}

	bizMRSMappings := []struct {
		tag      string
		filename string
		behavior string
		outbound string
	}{
		{"openai_mrs", "./rules/openai.mrs", "domain", "AI"},
		{"google_gemini_mrs", "./rules/google-gemini.mrs", "domain", "Gemini & Antigravity"},
		{"google_deepmind_mrs", "./rules/google-deepmind.mrs", "domain", "Gemini & Antigravity"},
		{"youtube_mrs", "./rules/youtube.mrs", "domain", "YouTube"},
		{"google_mrs", "./rules/google.mrs", "domain", "Google"},
		{"github_mrs", "./rules/github.mrs", "domain", "GitHub & Docker"},
		{"telegram_mrs", "./rules/telegram.mrs", "domain", "Telegram"},
		{"telegram_ip_mrs", "./rules/telegram_ip.mrs", "ipcidr", "Telegram"},
		{"twitter_mrs", "./rules/twitter.mrs", "domain", "Twitter (X)"},
		{"netflix_mrs", "./rules/netflix.mrs", "domain", "Netflix & 流媒体"},
		{"disney_mrs", "./rules/disney.mrs", "domain", "Netflix & 流媒体"},
		{"apple_mrs", "./rules/apple.mrs", "domain", "Apple 服务"},
		{"microsoft_mrs", "./rules/microsoft.mrs", "domain", "Microsoft"},
		{"onedrive_mrs", "./rules/onedrive.mrs", "domain", "Microsoft"},
		{"spotify_mrs", "./rules/spotify.mrs", "domain", "Spotify"},
		{"steam_mrs", "./rules/steam.mrs", "domain", "Steam 游戏"},
		{"epicgames_mrs", "./rules/epicgames.mrs", "domain", "Steam 游戏"},
		{"gfw_mrs", "./rules/gfw_domain.mrs", "domain", foreignGroup},
		{"cn_not_domain_mrs", "./rules/cn_not_domain.mrs", "domain", foreignGroup},
		// Extended common ecosystems
		{"tencent_mrs", "./rules/tencent.mrs", "domain", domesticGroup},
		{"alibaba_mrs", "./rules/alibaba.mrs", "domain", domesticGroup},
		{"baidu_mrs", "./rules/baidu.mrs", "domain", domesticGroup},
		{"bytedance_mrs", "./rules/bytedance.mrs", "domain", domesticGroup},
		{"netease_mrs", "./rules/netease.mrs", "domain", domesticGroup},
		{"bilibili_mrs", "./rules/bilibili.mrs", "domain", domesticGroup},
		{"discord_mrs", "./rules/discord.mrs", "domain", foreignGroup},
		{"whatsapp_mrs", "./rules/whatsapp.mrs", "domain", "Telegram"},
		{"line_mrs", "./rules/line.mrs", "domain", "Telegram"},
		{"signal_mrs", "./rules/signal.mrs", "domain", "Telegram"},
		{"primevideo_mrs", "./rules/primevideo.mrs", "domain", "Netflix & 流媒体"},
		{"hbo_mrs", "./rules/hbo.mrs", "domain", "Netflix & 流媒体"},
		{"twitch_mrs", "./rules/twitch.mrs", "domain", "Netflix & 流媒体"},
		{"bahamut_mrs", "./rules/bahamut.mrs", "domain", "Netflix & 流媒体"},
		{"playstation_mrs", "./rules/playstation.mrs", "domain", "Steam 游戏"},
		{"nintendo_mrs", "./rules/nintendo.mrs", "domain", "Steam 游戏"},
		{"ea_mrs", "./rules/ea.mrs", "domain", "Steam 游戏"},
		{"blizzard_mrs", "./rules/blizzard.mrs", "domain", "Steam 游戏"},
		{"binance_mrs", "./rules/binance.mrs", "domain", foreignGroup},
		{"okx_mrs", "./rules/okx.mrs", "domain", foreignGroup},
		{"paypal_mrs", "./rules/paypal.mrs", "domain", foreignGroup},
		{"gitlab_mrs", "./rules/gitlab.mrs", "domain", "GitHub & Docker"},
		{"notion_mrs", "./rules/notion.mrs", "domain", "Microsoft"},
		{"zoom_mrs", "./rules/zoom.mrs", "domain", "Microsoft"},
		{"amazon_mrs", "./rules/amazon.mrs", "domain", foreignGroup},
		{"speedtest_mrs", "./rules/speedtest.mrs", "domain", foreignGroup},
		{"tiktok_mrs", "./rules/tiktok.mrs", "domain", foreignGroup},
		{"reddit_mrs", "./rules/reddit.mrs", "domain", foreignGroup},
	}

	for _, m := range bizMRSMappings {
		if isMRSAvailable(m.filename) {
			ruleProviders[m.tag] = map[string]any{
				"type":     "file",
				"behavior": m.behavior,
				"format":   "mrs",
				"path":     m.filename,
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

	// 1b. Domain Filter Whitelist (放行域名 / 优先直连放行，彻底消除误杀)
	for _, d := range ir.AdBlockWhitelist {
		d = strings.TrimSpace(d)
		if d != "" {
			rulesList = append(rulesList, fmt.Sprintf("DOMAIN-SUFFIX,%s,DIRECT", d))
		}
	}

	// 2. 217heidai MRS High-Performance Interception Rules (Mihomo Adaptive)
	if ir.AdBlockEnabled {
		rulesList = append(rulesList, "RULE-SET,217heidai_ads,REJECT")
		rulesList = append(rulesList, "RULE-SET,privacy_domain,REJECT")
	}

	// Reject foreign UDP 443/8443 (QUIC) BEFORE business rules so YouTube / Google automatically use line-speed TCP (xtls-reality/BBR)
	// Eliminates the ~5000 Kbps ISP UDP throttling and restores 100+ Mbps 4K 60fps streaming
	rulesList = append(rulesList, "AND,((NETWORK,UDP),(OR,((DST-PORT,443),(DST-PORT,8443))),(GEOIP,CN)),DIRECT")
	rulesList = append(rulesList, "AND,((NETWORK,UDP),(OR,((DST-PORT,443),(DST-PORT,8443)))),REJECT")

	// High-Performance Dynamic MRS Business Rules (Dual-Engine Symmetrical Acceleration)
	for _, m := range bizMRSMappings {
		if _, ok := ruleProviders[m.tag]; ok {
			targetOut := m.outbound
			if !validTargets[targetOut] {
				if targetOut == "Telegram" && validTargets[foreignGroup] {
					targetOut = foreignGroup
				} else if validTargets[foreignGroup] {
					targetOut = foreignGroup
				} else {
					targetOut = "DIRECT"
				}
			}
			rulesList = append(rulesList, fmt.Sprintf("RULE-SET,%s,%s", m.tag, targetOut))
		}
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

	// High-Priority Line-Speed Benchmark / Speedtest Fast-Path Rule
	if validTargets[foreignGroup] {
		rulesList = append(rulesList, fmt.Sprintf("DOMAIN,speed.cloudflare.com,%s", foreignGroup))
		rulesList = append(rulesList, fmt.Sprintf("DOMAIN,challenges.cloudflare.com,%s", foreignGroup))
		rulesList = append(rulesList, fmt.Sprintf("DOMAIN-SUFFIX,fast.com,%s", foreignGroup))
		rulesList = append(rulesList, fmt.Sprintf("DOMAIN-SUFFIX,speedtest.net,%s", foreignGroup))
		rulesList = append(rulesList, fmt.Sprintf("DOMAIN-SUFFIX,ooklaserver.net,%s", foreignGroup))
	}


	// 2a-2. AI (OpenAI / Claude / Perplexity / Copilot / Grok / LLMs)
	if validTargets["AI"] {
		rulesList = append(rulesList, "DOMAIN-SUFFIX,openai.com,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,chatgpt.com,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,oaistatic.com,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,oaiusercontent.com,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,sora.com,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,claude.ai,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,claude.com,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,anthropic.com,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,claudecdn.com,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,claudecontent.com,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,perplexity.ai,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,x.ai,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,grok.com,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,copilot.microsoft.com,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,sydney.bing.com,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,mistral.ai,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,poe.com,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,midjourney.com,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,huggingface.co,AI")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,civitai.com,AI")
	}

	// 2a-3. Gemini & Antigravity (Google Gemini / Antigravity / AI Studio / DeepMind)
	if validTargets["Gemini & Antigravity"] {
		rulesList = append(rulesList, "DOMAIN,antigravity-unleash.goog,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN,antigravity.dev,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,antigravity.dev,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,antigravity.google,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,antigravity-unleash.goog,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN,generativelanguage.googleapis.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,generativelanguage.googleapis.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN,daily-cloudcode-pa.googleapis.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,daily-cloudcode-pa.googleapis.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN,cloudcode-pa.googleapis.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,cloudcode-pa.googleapis.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN,aiplatform.googleapis.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,aiplatform.googleapis.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN,alkalimakersuite-pa.clients6.google.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,alkalimakersuite-pa.clients6.google.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN,aistudio.google.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,aistudio.google.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN,makersuite.google.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,makersuite.google.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN,bard.google.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,bard.google.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN,gemini.google.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,gemini.google.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN,gemini.google,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,gemini.google,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,ai.studio,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,deepmind.google,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,deepmind.com,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,notebooklm.google,Gemini & Antigravity")
		rulesList = append(rulesList, "DOMAIN-SUFFIX,notebooklm.google.com,Gemini & Antigravity")
	}

	// 2b. High-Speed Business Service Rules (Native Domain-Suffix & GEOSITE)
	bizRules := []struct {
		group    string
		suffixes []string
		geosite  string
	}{
		{"AI", []string{"openai.com", "chatgpt.com", "anthropic.com", "claude.ai", "claude.com", "claudecdn.com", "claudecontent.com", "oaistatic.com", "oaiusercontent.com", "sora.com", "perplexity.ai", "x.ai", "grok.com", "copilot.microsoft.com", "sydney.bing.com", "mistral.ai", "poe.com", "midjourney.com", "huggingface.co", "civitai.com"}, "openai"},
		{"Gemini & Antigravity", []string{"gemini.google.com", "gemini.google", "aistudio.google.com", "ai.studio", "deepmind.google", "deepmind.com", "antigravity.dev", "antigravity.google", "notebooklm.google"}, "google-gemini"},
		{"YouTube", []string{"youtube.com", "ytimg.com", "googlevideo.com", "youtu.be", "youtubei.googleapis.com", "yt3.ggpht.com", "yt4.ggpht.com", "gvt1.com", "1e100.net", "googleusercontent.com"}, "youtube"},
		{"Google", []string{"google.com", "googleapis.com", "gstatic.com", "googleusercontent.com", "ggpht.com", "goog", "run.app", "1e100.net", "gvt1.com"}, "google"},
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

	// 2c. Domestic Direct Routing (China Domain & IP - Prioritized after business strategies)
	cnOutbound := "DIRECT"
	if validTargets["🎯 国内流量"] {
		cnOutbound = "🎯 国内流量"
	}
	rulesList = append(rulesList, fmt.Sprintf("RULE-SET,cn_domain,%s", cnOutbound))
	rulesList = append(rulesList, fmt.Sprintf("RULE-SET,cn_ip,%s", cnOutbound))
	rulesList = append(rulesList, fmt.Sprintf("GEOIP,CN,%s", cnOutbound))

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
