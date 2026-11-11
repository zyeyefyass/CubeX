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
				if node.Name != "" && node.Server != "" && !schema.IsInformationalOrInvalidNode(node.Name) {
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
				if node.Name != "" && node.Server != "" && !schema.IsInformationalOrInvalidNode(node.Name) {
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
		if err == nil && node.Name != "" && node.Server != "" && !schema.IsInformationalOrInvalidNode(node.Name) {
			nodes = append(nodes, node)
		}
	}

	if len(nodes) > 0 {
		return nodes, nil
	}

	return nil, fmt.Errorf("unrecognized subscription format or no valid nodes found")
}

// UniversalSmartParse strips markdown formatting, HTML entities, and carriage returns,
// then automatically identifies and parses single/multi-line proxy links, base64 blobs, Clash YAML, or sing-box JSON.
func UniversalSmartParse(rawInput string) ([]schema.ProxyNode, error) {
	trimmed := strings.TrimSpace(rawInput)
	if trimmed == "" {
		return nil, fmt.Errorf("empty input")
	}

	// 1. Strip markdown fences (```yaml ... ``` or ```json ... ``` or ``` ...)
	if strings.HasPrefix(trimmed, "```") {
		lines := strings.Split(trimmed, "\n")
		if len(lines) >= 2 {
			lines = lines[1:]
			if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
				lines = lines[:len(lines)-1]
			}
			trimmed = strings.TrimSpace(strings.Join(lines, "\n"))
		}
	}

	// 2. Unescape common HTML encoded entities
	trimmed = strings.ReplaceAll(trimmed, "&amp;", "&")
	trimmed = strings.ReplaceAll(trimmed, "&lt;", "<")
	trimmed = strings.ReplaceAll(trimmed, "&gt;", ">")
	trimmed = strings.ReplaceAll(trimmed, "&quot;", "\"")
	trimmed = strings.ReplaceAll(trimmed, "&#39;", "'")

	return ParseSubscription([]byte(trimmed))
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

	cipher := getString("cipher")
	if cipher == "" {
		cipher = getString("method")
	}
	ports := getString("ports")
	if ports == "" {
		ports = getString("mport")
	}
	if ports == "" {
		ports = getString("server_ports")
	}

	node := schema.ProxyNode{
		Name:           getString("name"),
		Type:           strings.ToLower(getString("type")),
		Server:         getString("server"),
		Port:           getInt("port"),
		Ports:          ports,
		UUID:           getString("uuid"),
		Password:       getString("password"),
		Cipher:         cipher,
		TLS:            getBool("tls"),
		SNI:            getString("servername"),
		SkipCertVerify: getBool("skip-cert-verify"),
		Network:        getString("network"),
		Flow:           getString("flow"),
	}

	if node.Password == "" {
		node.Password = getString("psk")
	}
	if node.SNI == "" {
		node.SNI = getString("sni")
	}

	if rawAlpn, ok := p["alpn"].([]any); ok {
		for _, a := range rawAlpn {
			if as, ok := a.(string); ok && as != "" {
				node.ALPN = append(node.ALPN, as)
			}
		}
	} else if rawAlpnStr, ok := p["alpn"].([]string); ok {
		node.ALPN = rawAlpnStr
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
	} else if huOpts, ok := p["httpupgrade-opts"].(map[string]any); ok {
		if path, ok := huOpts["path"].(string); ok {
			node.Path = path
		}
	} else if xhOpts, ok := p["xhttp-opts"].(map[string]any); ok {
		if path, ok := xhOpts["path"].(string); ok {
			node.Path = path
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

	cipher := getString("method")
	if cipher == "" {
		cipher = getString("cipher")
	}
	node := schema.ProxyNode{
		Name:     getString("tag"),
		Type:     pType,
		Server:   getString("server"),
		Port:     getInt("server_port"),
		UUID:     getString("uuid"),
		Password: getString("password"),
		Cipher:   cipher,
		Flow:     getString("flow"),
	}

	raw := make(map[string]any)
	for k, v := range ob {
		raw[k] = v
	}
	if cipher != "" {
		raw["cipher"] = cipher
		raw["method"] = cipher
	}

	if tlsMap, ok := ob["tls"].(map[string]any); ok {
		node.TLS = true
		if sni, ok := tlsMap["server_name"].(string); ok {
			node.SNI = sni
		}
		if insecure, ok := tlsMap["insecure"].(bool); ok {
			node.SkipCertVerify = insecure
		}
		if rawAlpn, ok := tlsMap["alpn"].([]any); ok {
			for _, a := range rawAlpn {
				if as, ok := a.(string); ok && as != "" {
					node.ALPN = append(node.ALPN, as)
				}
			}
		} else if rawAlpnStr, ok := tlsMap["alpn"].([]string); ok {
			node.ALPN = rawAlpnStr
		}
		if realityMap, ok := tlsMap["reality"].(map[string]any); ok {
			realityOpts := make(map[string]any)
			if pk, ok := realityMap["public_key"].(string); ok {
				realityOpts["public-key"] = pk
			}
			if sid, ok := realityMap["short_id"].(string); ok {
				realityOpts["short-id"] = sid
			}
			if spx, ok := realityMap["spider_x"].(string); ok && spx != "" {
				realityOpts["spider-x"] = spx
			} else if spx, ok := realityMap["spider-x"].(string); ok && spx != "" {
				realityOpts["spider-x"] = spx
			}
			raw["reality-opts"] = realityOpts
		}
		if utlsMap, ok := tlsMap["utls"].(map[string]any); ok {
			if fp, ok := utlsMap["fingerprint"].(string); ok {
				raw["client-fingerprint"] = fp
			}
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

	node.RawConfig = raw
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
		raw := map[string]any{
			"name":    node.Name,
			"type":    "vmess",
			"server":  node.Server,
			"port":    node.Port,
			"uuid":    node.UUID,
			"alterId": 0,
			"cipher":  "auto",
			"udp":     true,
		}
		if node.TLS {
			raw["tls"] = true
			if node.SNI != "" {
				raw["servername"] = node.SNI
			}
		}
		applyURITransport(&node, raw, node.Network, node.Path, v.Host, node.Path, "", "")
		node.RawConfig = raw
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
	if node.Server == "" && u.Host != "" {
		hostStr := u.Host
		if idx := strings.LastIndex(hostStr, ":"); idx != -1 {
			node.Server = hostStr[:idx]
			portPart := hostStr[idx+1:]
			if p, err := strconv.Atoi(portPart); err == nil {
				node.Port = p
			} else if strings.Contains(portPart, "-") || strings.Contains(portPart, ":") {
				node.Ports = portPart
			}
		} else {
			node.Server = hostStr
		}
	}

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
			if spx := q.Get("spx"); spx != "" {
				realityOpts["spider-x"] = spx
			} else if spiderX := q.Get("spider_x"); spiderX != "" {
				realityOpts["spider-x"] = spiderX
			}
			raw["reality-opts"] = realityOpts
		}
		applyURITransport(&node, raw, node.Network, node.Path, q.Get("host"), q.Get("serviceName"), q.Get("mode"), q.Get("extra"))
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
		q := u.Query()
		node.Type = "trojan"
		node.Password = u.User.Username()
		node.SNI = q.Get("sni")
		node.TLS = true
		if q.Get("insecure") == "1" || q.Get("insecure") == "true" {
			node.SkipCertVerify = true
		}
		if netType := q.Get("type"); netType != "" {
			node.Network = netType
		}
		if p := q.Get("path"); p != "" {
			node.Path = p
		}
		raw := map[string]any{
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
		applyURITransport(&node, raw, node.Network, node.Path, q.Get("host"), q.Get("serviceName"), q.Get("mode"), q.Get("extra"))
		node.RawConfig = raw
	case "hysteria2", "hy2":
		node.Type = "hysteria2"
		node.Password = u.User.Username()
		node.SNI = u.Query().Get("sni")
		node.TLS = true
		if u.Query().Get("insecure") == "1" || u.Query().Get("insecure") == "true" {
			node.SkipCertVerify = true
		}
		if node.Ports == "" {
			ports := u.Query().Get("mport")
			if ports == "" {
				ports = u.Query().Get("ports")
			}
			node.Ports = ports
		}
		raw := map[string]any{
			"name":             node.Name,
			"type":             "hysteria2",
			"server":           node.Server,
			"port":             node.Port,
			"password":         node.Password,
			"sni":              node.SNI,
			"skip-cert-verify": node.SkipCertVerify,
			"udp":              true,
		}
		if node.Ports != "" {
			raw["ports"] = node.Ports
			raw["mport"] = node.Ports
		}
		node.RawConfig = raw
	case "tuic":
		node.Type = "tuic"
		node.UUID = u.User.Username()
		if pass, ok := u.User.Password(); ok {
			node.Password = pass
		}
		node.SNI = u.Query().Get("sni")
		node.TLS = true
		raw := map[string]any{
			"name":                  node.Name,
			"type":                  "tuic",
			"server":                node.Server,
			"port":                  node.Port,
			"uuid":                  node.UUID,
			"password":              node.Password,
			"sni":                   node.SNI,
			"tls":                   true,
			"udp":                   true,
			"congestion-controller": "bbr",
			"reduce-rtt":            true,
		}
		if cc := u.Query().Get("congestion_control"); cc != "" {
			raw["congestion-controller"] = cc
		}
		if urm := u.Query().Get("udp_relay_mode"); urm != "" {
			raw["udp-relay-mode"] = urm
		}
		if alpn := u.Query().Get("alpn"); alpn != "" {
			node.ALPN = strings.Split(alpn, ",")
			raw["alpn"] = node.ALPN
		}
		node.RawConfig = raw
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
	case "snell":
		q := u.Query()
		node.Type = "snell"
		if u.User != nil {
			node.Password = u.User.Username()
		}
		if node.Password == "" {
			node.Password = q.Get("psk")
		}
		ver := 6
		if vStr := q.Get("version"); vStr != "" {
			if v, err := strconv.Atoi(vStr); err == nil && v > 0 {
				ver = v
			}
		} else if vStr := q.Get("ver"); vStr != "" {
			if v, err := strconv.Atoi(vStr); err == nil && v > 0 {
				ver = v
			}
		}
		raw := map[string]any{
			"name":    node.Name,
			"type":    "snell",
			"server":  node.Server,
			"port":    node.Port,
			"psk":     node.Password,
			"version": ver,
			"udp":     true,
		}
		if obfs := q.Get("obfs"); obfs != "" {
			obfsOpts := map[string]any{"mode": obfs}
			if h := q.Get("obfs-host"); h != "" {
				obfsOpts["host"] = h
			} else if h := q.Get("host"); h != "" {
				obfsOpts["host"] = h
			}
			raw["obfs-opts"] = obfsOpts
		}
		node.RawConfig = raw
	}

	return node, nil
}

// applyURITransport normalizes ws, grpc, httpupgrade, and xhttp transports into IR + Mihomo RawConfig.
func applyURITransport(node *schema.ProxyNode, raw map[string]any, netType, path, host, serviceName, mode, extra string) {
	norm := strings.ToLower(strings.TrimSpace(netType))
	if norm == "splithttp" {
		norm = "xhttp"
	}
	node.Network = norm
	switch norm {
	case "grpc":
		sn := serviceName
		if sn == "" {
			sn = path
		}
		node.Path = sn
		raw["network"] = "grpc"
		raw["grpc-opts"] = map[string]any{
			"grpc-service-name": sn,
		}
	case "ws":
		node.Path = path
		raw["network"] = "ws"
		wsOpts := map[string]any{
			"path": path,
		}
		if host != "" {
			wsOpts["headers"] = map[string]any{"Host": host}
		}
		raw["ws-opts"] = wsOpts
	case "httpupgrade":
		node.Path = path
		raw["network"] = "httpupgrade"
		huOpts := map[string]any{}
		if path != "" {
			huOpts["path"] = path
		}
		if host != "" {
			huOpts["host"] = host
		}
		raw["httpupgrade-opts"] = huOpts
	case "xhttp":
		node.Path = path
		raw["network"] = "xhttp"
		xhOpts := map[string]any{}
		if path != "" {
			xhOpts["path"] = path
		}
		if host != "" {
			xhOpts["host"] = host
		}
		if mode != "" {
			xhOpts["mode"] = mode
		}
		if extra != "" {
			xhOpts["extra"] = extra
		}
		raw["xhttp-opts"] = xhOpts
	}
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
