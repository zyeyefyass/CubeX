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
		"device":               net.TunInterface,
		"auto-route":           true,
		"auto-detect-interface": true,
		"strict-route":         true,
		"mtu":                  net.TunMTU,
		"dns-hijack": []string{
			"tcp://any:53",
			"udp://any:53",
		},
	}

	// Sniffer
	cfg["sniffer"] = map[string]any{
		"enable": true,
		"sniff": map[string]any{
			"HTTP": map[string]any{"ports": []int{80, 8080-8880}, "override-destination": true},
			"TLS":  map[string]any{"ports": []int{443, 8443}, "override-destination": true},
			"QUIC": map[string]any{"ports": []int{443, 8443}, "override-destination": true},
		},
		"skip-domain": []string{
			"Mijia Cloud",
			"dlg.io.mi.com",
			"+.apple.com",
		},
	}

	// DNS Configuration
	dnsMap := map[string]any{
		"enable":             true,
		"listen":             fmt.Sprintf("0.0.0.0:%d", net.DNSPort),
		"ipv6":               net.IPv6,
		"enhanced-mode":      "fake-ip",
		"fake-ip-range":      dns.FakeIPRange,
		"fake-ip-filter":     dns.FakeIPFilter,
		"default-nameserver": dns.DirectDNS,
		"nameserver":         dns.RemoteDNS,
		"fallback":           dns.FallbackDNS,
		"fallback-filter": map[string]any{
			"geoip":     true,
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

	// Proxies
	proxiesList := make([]map[string]any, 0, len(ir.Proxies))
	for _, p := range ir.Proxies {
		if p.Type == "direct" || p.Type == "block" || p.Type == "reject" {
			continue
		}
		pMap := map[string]any{
			"name":   p.Name,
			"type":   p.Type,
			"server": p.Server,
			"port":   p.Port,
		}
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
		proxiesList = append(proxiesList, pMap)
	}
	cfg["proxies"] = proxiesList

	// Proxy Groups
	groupsList := make([]map[string]any, 0, len(ir.Groups))
	for _, g := range ir.Groups {
		gType := g.Type
		if gType == "select" {
			gType = "select"
		} else if gType == "urltest" {
			gType = "url-test"
		}

		proxies := g.Proxies
		if len(proxies) == 0 && len(proxiesList) > 0 {
			// Auto populate proxies if empty
			for _, p := range proxiesList {
				proxies = append(proxies, p["name"].(string))
			}
		}

		gMap := map[string]any{
			"name":    g.Name,
			"type":    gType,
			"proxies": proxies,
		}
		if gType == "url-test" || gType == "fallback" {
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

	// 2. Generic Rules
	for _, r := range ir.Rules {
		rType := strings.ToUpper(r.Type)
		switch rType {
		case "DOMAIN":
			rulesList = append(rulesList, fmt.Sprintf("DOMAIN,%s,%s", r.Value, r.Target))
		case "DOMAIN_SUFFIX", "DOMAIN-SUFFIX":
			rulesList = append(rulesList, fmt.Sprintf("DOMAIN-SUFFIX,%s,%s", r.Value, r.Target))
		case "DOMAIN_KEYWORD", "DOMAIN-KEYWORD":
			rulesList = append(rulesList, fmt.Sprintf("DOMAIN-KEYWORD,%s,%s", r.Value, r.Target))
		case "GEOSITE":
			rulesList = append(rulesList, fmt.Sprintf("GEOSITE,%s,%s", r.Value, r.Target))
		case "GEOIP":
			rulesList = append(rulesList, fmt.Sprintf("GEOIP,%s,%s", strings.ToUpper(r.Value), r.Target))
		case "IP_CIDR", "IP-CIDR":
			rulesList = append(rulesList, fmt.Sprintf("IP-CIDR,%s,%s", r.Value, r.Target))
		case "SRC_IP_CIDR", "SRC-IP-CIDR":
			rulesList = append(rulesList, fmt.Sprintf("SRC-IP-CIDR,%s,%s", r.Value, r.Target))
		case "MATCH", "FINAL":
			rulesList = append(rulesList, fmt.Sprintf("MATCH,%s", r.Target))
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
		rulesList = append(rulesList, "MATCH,🐟 漏网之鱼")
	}

	cfg["rules"] = rulesList

	return yaml.Marshal(cfg)
}
