package sub

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/zyeyefyass/CubeX/internal/schema"
	"gopkg.in/yaml.v3"
)

// ParseSubscription automatically identifies and extracts ProxyNodes from Clash YAML, sing-box JSON, or base64 links.
func ParseSubscription(raw []byte) ([]schema.ProxyNode, error) {
	trimmed := strings.TrimSpace(string(raw))
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty subscription data")
	}

	// 1. Try Clash YAML (look for 'proxies:')
	if strings.Contains(trimmed, "proxies:") {
		var clashCfg struct {
			Proxies []map[string]any `yaml:"proxies"`
		}
		if err := yaml.Unmarshal(raw, &clashCfg); err == nil && len(clashCfg.Proxies) > 0 {
			nodes := make([]schema.ProxyNode, 0, len(clashCfg.Proxies))
			for _, p := range clashCfg.Proxies {
				node := parseClashNode(p)
				if node.Name != "" && node.Server != "" {
					nodes = append(nodes, node)
				}
			}
			if len(nodes) > 0 {
				return nodes, nil
			}
		}
	}

	// 2. Try sing-box JSON (look for 'outbounds')
	if strings.HasPrefix(trimmed, "{") && strings.Contains(trimmed, "outbounds") {
		var sbCfg struct {
			Outbounds []map[string]any `json:"outbounds"`
		}
		if err := json.Unmarshal(raw, &sbCfg); err == nil && len(sbCfg.Outbounds) > 0 {
			nodes := make([]schema.ProxyNode, 0, len(sbCfg.Outbounds))
			for _, ob := range sbCfg.Outbounds {
				node := parseSingboxNode(ob)
				if node.Name != "" && node.Server != "" {
					nodes = append(nodes, node)
				}
			}
			if len(nodes) > 0 {
				return nodes, nil
			}
		}
	}

	// 3. Try base64 decoded plain-text links
	var textToParse = trimmed
	if !strings.Contains(trimmed, "://") {
		// Attempt base64 decode
		cleanB64 := strings.ReplaceAll(trimmed, "\r\n", "")
		cleanB64 = strings.ReplaceAll(cleanB64, "\n", "")
		decoded, err := base64.StdEncoding.DecodeString(cleanB64)
		if err == nil {
			textToParse = string(decoded)
		} else {
			// Try URL-safe base64 decode
			decoded, err = base64.RawURLEncoding.DecodeString(cleanB64)
			if err == nil {
				textToParse = string(decoded)
			}
		}
	}

	// 4. Parse URI lines (ss://, vmess://, vless://, trojan://, hysteria2://, tuic://)
	lines := strings.Split(textToParse, "\n")
	nodes := make([]schema.ProxyNode, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		node, err := ParseURILink(line)
		if err == nil && node.Name != "" && node.Server != "" {
			nodes = append(nodes, node)
		}
	}

	if len(nodes) > 0 {
		return nodes, nil
	}

	return nil, fmt.Errorf("unrecognized subscription format or no valid nodes found")
}

func parseClashNode(p map[string]any) schema.ProxyNode {
	getString := func(key string) string {
		if v, ok := p[key]; ok && v != nil {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}
	getInt := func(key string) int {
		if v, ok := p[key]; ok && v != nil {
			switch n := v.(type) {
			case int:
				return n
			case float64:
				return int(n)
			case string:
				val, _ := strconv.Atoi(n)
				return val
			}
		}
		return 0
	}
	getBool := func(key string) bool {
		if v, ok := p[key]; ok && v != nil {
			b, _ := strconv.ParseBool(fmt.Sprintf("%v", v))
			return b
		}
		return false
	}

	node := schema.ProxyNode{
		Name:           getString("name"),
		Type:           strings.ToLower(getString("type")),
		Server:         getString("server"),
		Port:           getInt("port"),
		UUID:           getString("uuid"),
		Password:       getString("password"),
		Cipher:         getString("cipher"),
		TLS:            getBool("tls"),
		SNI:            getString("servername"),
		SkipCertVerify: getBool("skip-cert-verify"),
		Network:        getString("network"),
		Flow:           getString("flow"),
	}

	if node.SNI == "" {
		node.SNI = getString("sni")
	}

	// Auto-enable TLS for protocols that inherently require TLS
	if node.Type == "hysteria2" || node.Type == "hy2" || node.Type == "tuic" || node.Type == "trojan" || node.Type == "anytls" {
		node.TLS = true
	}
	if node.Flow != "" {
		node.TLS = true
	}

	if wsOpts, ok := p["ws-opts"].(map[string]any); ok {
		if path, ok := wsOpts["path"].(string); ok {
			node.Path = path
		}
	} else if grpcOpts, ok := p["grpc-opts"].(map[string]any); ok {
		if sn, ok := grpcOpts["grpc-service-name"].(string); ok {
			node.Path = sn
		}
	}

	node.RawConfig = p

	return node
}

func parseSingboxNode(ob map[string]any) schema.ProxyNode {
	getString := func(key string) string {
		if v, ok := ob[key]; ok && v != nil {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}
	getInt := func(key string) int {
		if v, ok := ob[key]; ok && v != nil {
			switch n := v.(type) {
			case int:
				return n
			case float64:
				return int(n)
			case string:
				val, _ := strconv.Atoi(n)
				return val
			}
		}
		return 0
	}

	pType := strings.ToLower(getString("type"))
	if pType == "direct" || pType == "block" || pType == "dns" || pType == "selector" || pType == "urltest" {
		return schema.ProxyNode{}
	}

	node := schema.ProxyNode{
		Name:     getString("tag"),
		Type:     pType,
		Server:   getString("server"),
		Port:     getInt("server_port"),
		UUID:     getString("uuid"),
		Password: getString("password"),
		Cipher:   getString("method"),
		Flow:     getString("flow"),
	}

	if tlsMap, ok := ob["tls"].(map[string]any); ok {
		node.TLS = true
		if sni, ok := tlsMap["server_name"].(string); ok {
			node.SNI = sni
		}
		if insecure, ok := tlsMap["insecure"].(bool); ok {
			node.SkipCertVerify = insecure
		}
	}

	if transMap, ok := ob["transport"].(map[string]any); ok {
		node.Network = fmt.Sprintf("%v", transMap["type"])
		if path, ok := transMap["path"].(string); ok {
			node.Path = path
		} else if sn, ok := transMap["service_name"].(string); ok {
			node.Path = sn
		}
	}

	return node
}

// ParseURILink parses a single node URI (ss, vmess, vless, trojan, hysteria2, tuic).
func ParseURILink(link string) (schema.ProxyNode, error) {
	node := schema.ProxyNode{}

	if strings.HasPrefix(link, "vmess://") {
		// vmess is base64 encoded JSON
		b64 := strings.TrimPrefix(link, "vmess://")
		decoded, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			decoded, err = base64.RawURLEncoding.DecodeString(b64)
			if err != nil {
				return node, err
			}
		}
		var v struct {
			V    any    `json:"v"`
			PS   string `json:"ps"`
			Add  string `json:"add"`
			Port any    `json:"port"`
			ID   string `json:"id"`
			Net  string `json:"net"`
			TLS  string `json:"tls"`
			Path string `json:"path"`
			Host string `json:"host"`
			SNI  string `json:"sni"`
		}
		if err := json.Unmarshal(decoded, &v); err != nil {
			return node, err
		}
		node.Name = v.PS
		node.Type = "vmess"
		node.Server = v.Add
		node.Port = toInt(v.Port)
		node.UUID = v.ID
		node.Network = v.Net
		node.Path = v.Path
		node.TLS = v.TLS == "tls"
		node.SNI = v.SNI
		if node.SNI == "" {
			node.SNI = v.Host
		}
		return node, nil
	}

	u, err := url.Parse(link)
	if err != nil {
		return node, err
	}

	if u.Fragment != "" {
		if unescaped, err := url.QueryUnescape(u.Fragment); err == nil && unescaped != "" {
			node.Name = unescaped
		} else {
			node.Name = u.Fragment
		}
	}
	if node.Name == "" {
		node.Name = u.Host
	}

	node.Server = u.Hostname()
	node.Port, _ = strconv.Atoi(u.Port())

	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "vless":
		node.Type = "vless"
		node.UUID = u.User.Username()
		q := u.Query()
		node.Flow = q.Get("flow")
		node.Network = q.Get("type")
		node.Path = q.Get("path")
		node.SNI = q.Get("sni")
		if node.SNI == "" {
			node.SNI = q.Get("servername")
		}
		security := strings.ToLower(q.Get("security"))
		if security == "tls" || security == "reality" || node.Flow != "" {
			node.TLS = true
		}
		if q.Get("insecure") == "1" || q.Get("insecure") == "true" {
			node.SkipCertVerify = true
		}

		raw := map[string]any{
			"name":   node.Name,
			"type":   "vless",
			"server": node.Server,
			"port":   node.Port,
			"uuid":   node.UUID,
			"udp":    true,
		}
		if node.Flow != "" {
			raw["flow"] = node.Flow
		}
		if node.TLS {
			raw["tls"] = true
			if node.SNI != "" {
				raw["servername"] = node.SNI
			}
			if node.SkipCertVerify {
				raw["skip-cert-verify"] = true
			}
		}
		if fp := q.Get("fp"); fp != "" {
			raw["client-fingerprint"] = fp
		}
		if security == "reality" || q.Get("pbk") != "" {
			realityOpts := map[string]any{
				"public-key": q.Get("pbk"),
			}
			if sid := q.Get("sid"); sid != "" {
				realityOpts["short-id"] = sid
			}
			raw["reality-opts"] = realityOpts
		}
		if node.Network == "grpc" {
			serviceName := q.Get("serviceName")
			if serviceName == "" {
				serviceName = q.Get("path")
			}
			node.Path = serviceName
			raw["network"] = "grpc"
			raw["grpc-opts"] = map[string]any{
				"grpc-service-name": serviceName,
			}
		} else if node.Network == "ws" {
			raw["network"] = "ws"
			raw["ws-opts"] = map[string]any{
				"path": node.Path,
			}
		}
		node.RawConfig = raw
	case "anytls":
		node.Type = "anytls"
		node.Password = u.User.Username()
		node.SNI = u.Query().Get("sni")
		node.SkipCertVerify = u.Query().Get("insecure") == "1" || u.Query().Get("insecure") == "true"
		node.TLS = true
		node.RawConfig = map[string]any{
			"name":             node.Name,
			"type":             "anytls",
			"server":           node.Server,
			"port":             node.Port,
			"password":         node.Password,
			"sni":              node.SNI,
			"skip-cert-verify": node.SkipCertVerify,
			"tls":              true,
		}
	case "trojan":
		node.Type = "trojan"
		node.Password = u.User.Username()
		node.SNI = u.Query().Get("sni")
		node.TLS = true
		if u.Query().Get("insecure") == "1" || u.Query().Get("insecure") == "true" {
			node.SkipCertVerify = true
		}
		node.RawConfig = map[string]any{
			"name":             node.Name,
			"type":             "trojan",
			"server":           node.Server,
			"port":             node.Port,
			"password":         node.Password,
			"servername":       node.SNI,
			"tls":              true,
			"skip-cert-verify": node.SkipCertVerify,
			"udp":              true,
		}
	case "hysteria2", "hy2":
		node.Type = "hysteria2"
		node.Password = u.User.Username()
		node.SNI = u.Query().Get("sni")
		node.TLS = true
		if u.Query().Get("insecure") == "1" || u.Query().Get("insecure") == "true" {
			node.SkipCertVerify = true
		}
		node.RawConfig = map[string]any{
			"name":             node.Name,
			"type":             "hysteria2",
			"server":           node.Server,
			"port":             node.Port,
			"password":         node.Password,
			"sni":              node.SNI,
			"skip-cert-verify": node.SkipCertVerify,
			"udp":              true,
		}
	case "tuic":
		node.Type = "tuic"
		node.UUID = u.User.Username()
		if pass, ok := u.User.Password(); ok {
			node.Password = pass
		}
		node.SNI = u.Query().Get("sni")
		node.TLS = true
	case "ss":
		node.Type = "shadowsocks"
		// ss://base64(method:password)@server:port
		if u.User != nil {
			rawUser := u.User.Username()
			if decoded, err := base64.URLEncoding.DecodeString(rawUser); err == nil {
				parts := strings.SplitN(string(decoded), ":", 2)
				if len(parts) == 2 {
					node.Cipher = parts[0]
					node.Password = parts[1]
				}
			} else if decoded, err := base64.StdEncoding.DecodeString(rawUser); err == nil {
				parts := strings.SplitN(string(decoded), ":", 2)
				if len(parts) == 2 {
					node.Cipher = parts[0]
					node.Password = parts[1]
				}
			} else if pass, ok := u.User.Password(); ok {
				node.Cipher = rawUser
				node.Password = pass
			}
		}
	}

	return node, nil
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case string:
		val, _ := strconv.Atoi(n)
		return val
	}
	return 0
}
