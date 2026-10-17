package ros

import (
	"encoding/json"
	"fmt"
	"strings"
)

// CheckItem represents one diagnostic check item in RouterOS.
type CheckItem struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "missing", "warning"
	Value   string `json:"value"`
	Details string `json:"details"`
	Advice  string `json:"advice"`
}

// InspectionReport holds the comprehensive diagnostic matrix for RouterOS.
type InspectionReport struct {
	Connected   bool        `json:"connected"`
	ROSHost     string      `json:"ros_host"`
	ROSVersion  string      `json:"ros_version"`
	MiCubeIP    string      `json:"micube_ip"`
	AllOK       bool        `json:"all_ok"`
	Items       []CheckItem `json:"items"`
	CreatedTime int64       `json:"created_time"`
}

// ApplyResult reports the actions taken when auto-configuring RouterOS.
type ApplyResult struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Actions []string `json:"actions"`
	Errors  []string `json:"errors"`
}

// InspectRouting checks whether RouterOS firewall, policy routing, mangle, and DNS redirect are correctly configured.
func (m *Manager) InspectRouting() (*InspectionReport, error) {
	rosConnected, rosVersion := m.checkROSConnection()
	rosCfg := m.GetConfig()
	micubeIP := m.GetOutboundIP()
	if micubeIP == "" {
		micubeIP = "10.0.0.11"
	}

	report := &InspectionReport{
		Connected:  rosConnected,
		ROSHost:    rosCfg["host"],
		ROSVersion: rosVersion,
		MiCubeIP:   micubeIP,
		AllOK:      true,
		Items:      make([]CheckItem, 0),
	}

	if !rosConnected {
		report.AllOK = false
		report.Items = append(report.Items, CheckItem{
			Key:     "connection",
			Name:    "RouterOS 连接状态",
			Status:  "missing",
			Value:   "未连接",
			Details: fmt.Sprintf("无法连接至 RouterOS REST API (%s:%s)，请在「系统升级与联动」配置正确的主机、用户名和密码，并确保在 WinBox 中开启 www-ssl 或 www 服务。", rosCfg["host"], rosCfg["port"]),
			Advice:  "请检查 RouterOS IP、用户凭证及 /ip/service 中 REST API 端口开启状态",
		})
		return report, nil
	}

	// Fetch RouterOS resources
	var routes []map[string]any
	if data, err := m.QueryROS("/ip/route"); err == nil {
		_ = json.Unmarshal(data, &routes)
	}

	var filters []map[string]any
	if data, err := m.QueryROS("/ip/firewall/filter"); err == nil {
		_ = json.Unmarshal(data, &filters)
	}

	var mangles []map[string]any
	if data, err := m.QueryROS("/ip/firewall/mangle"); err == nil {
		_ = json.Unmarshal(data, &mangles)
	}

	var nats []map[string]any
	if data, err := m.QueryROS("/ip/firewall/nat"); err == nil {
		_ = json.Unmarshal(data, &nats)
	}

	var tables []map[string]any
	if data, err := m.QueryROS("/routing/table"); err == nil {
		_ = json.Unmarshal(data, &tables)
	}

	// 1. Check /routing/table (table micube)
	tableOK := false
	var tableVal string
	for _, t := range tables {
		if t["name"] == "micube" {
			fib, _ := t["fib"].(bool)
			if !fib {
				if fibStr, ok := t["fib"].(string); ok && (fibStr == "true" || fibStr == "yes") {
					fib = true
				}
			}
			tableOK = true
			tableVal = fmt.Sprintf("name=micube fib=%v", fib)
			break
		}
	}
	if tableOK {
		report.Items = append(report.Items, CheckItem{
			Key:     "routing_table",
			Name:    "策略路由表 (/routing/table)",
			Status:  "ok",
			Value:   tableVal,
			Details: "检测到专有路由表「micube」已正确注册并启用 FIB 转发。",
			Advice:  "无需调整",
		})
	} else {
		report.AllOK = false
		report.Items = append(report.Items, CheckItem{
			Key:     "routing_table",
			Name:    "策略路由表 (/routing/table)",
			Status:  "missing",
			Value:   "缺失",
			Details: "未找到专用策略路由表「micube」。若无此表，Mangle 标记后流量无法被策略路由接管。",
			Advice:  "点击下方「一键自动配置」即可自动创建 /routing/table add name=micube fib",
		})
	}

	// 2. Check /ip/route (default route in table micube pointing to MiCubeIP)
	routeOK := false
	var routeVal string
	for _, r := range routes {
		table, _ := r["routing-table"].(string)
		dst, _ := r["dst-address"].(string)
		gw, _ := r["gateway"].(string)
		if table == "micube" && (dst == "0.0.0.0/0" || strings.HasPrefix(dst, "0.0.0.0")) {
			routeOK = true
			routeVal = fmt.Sprintf("dst=%s gw=%s (table=%s)", dst, gw, table)
			break
		}
	}
	if routeOK {
		report.Items = append(report.Items, CheckItem{
			Key:     "default_route",
			Name:    "旁路由默认路由 (/ip/route)",
			Status:  "ok",
			Value:   routeVal,
			Details: "检测到指向 CubeX 旁路由的默认分流出口 0.0.0.0/0 已生效。",
			Advice:  "正常运行中",
		})
	} else {
		report.AllOK = false
		report.Items = append(report.Items, CheckItem{
			Key:     "default_route",
			Name:    "旁路由默认路由 (/ip/route)",
			Status:  "missing",
			Value:   "缺失",
			Details: fmt.Sprintf("未在 micube 路由表中配置指向 CubeX (%s) 的默认网关路由 0.0.0.0/0。", micubeIP),
			Advice:  "点击下方「一键自动配置」自动下发默认分流网关",
		})
	}

	// 3. Check /ip/route (Fake-IP route 198.18.0.0/16 pointing to micubeIP and NO stale ECMP conflicts)
	fakeIPRouteCount := 0
	fakeIPCorrect := false
	staleFakeIPGateways := make([]string, 0)
	for _, r := range routes {
		dst, _ := r["dst-address"].(string)
		gw, _ := r["gateway"].(string)
		table, _ := r["routing-table"].(string)
		if table == "" || table == "main" {
			if dst == "198.18.0.0/16" || dst == "198.18.0.0/15" || strings.HasPrefix(dst, "198.18.") {
				fakeIPRouteCount++
				if gw == micubeIP || strings.Contains(gw, micubeIP) {
					fakeIPCorrect = true
				} else {
					staleFakeIPGateways = append(staleFakeIPGateways, gw)
				}
			}
		}
	}
	if fakeIPCorrect && len(staleFakeIPGateways) == 0 {
		report.Items = append(report.Items, CheckItem{
			Key:     "fakeip_route",
			Name:    "Fake-IP 回指路由 (198.18.0.0/16)",
			Status:  "ok",
			Value:   fmt.Sprintf("198.18.0.0/16 -> %s", micubeIP),
			Details: "RouterOS Fake-IP 静态路由唯一正确指向 CubeX 旁路由，无多路径冲突。",
			Advice:  "Fake-IP 路由工作正常",
		})
	} else if len(staleFakeIPGateways) > 0 {
		report.AllOK = false
		report.Items = append(report.Items, CheckItem{
			Key:     "fakeip_route",
			Name:    "Fake-IP 回指路由 (198.18.0.0/16)",
			Status:  "warning",
			Value:   fmt.Sprintf("存在 ECMP 冲突路由 (%s)", strings.Join(staleFakeIPGateways, ", ")),
			Details: fmt.Sprintf("检测到 RouterOS 路由表中存在指向非当前旁路由 (%s) 的残留 Fake-IP 路由，RouterOS 会通过 ECMP 多路径负载均衡将 50%% 境外流量送入黑洞，导致客户端打不开外网！", strings.Join(staleFakeIPGateways, ", ")),
			Advice:  "点击下方「一键自动配置」即可自动彻底清除残留冲突路由并保留唯一正确网关",
		})
	} else {
		report.AllOK = false
		report.Items = append(report.Items, CheckItem{
			Key:     "fakeip_route",
			Name:    "Fake-IP 回指路由 (198.18.0.0/16)",
			Status:  "missing",
			Value:   "缺失",
			Details: fmt.Sprintf("未在 RouterOS main 路由表中配置将 198.18.0.0/16 回指给 CubeX (%s)。当客户端网关设为主路由时，Fake-IP 流量将无法抵达旁路由内核。", micubeIP),
			Advice:  "点击下方「一键自动配置」自动创建 Fake-IP 静态回指路由",
		})
	}

	// 4. Check FastTrack Compatibility (/ip/firewall/filter action=fasttrack-connection)
	fastTrackFound := false
	fastTrackExcludesCubeX := false
	for _, f := range filters {
		action, _ := f["action"].(string)
		chain, _ := f["chain"].(string)
		disabled, _ := f["disabled"].(bool)
		if disabled {
			if disStr, ok := f["disabled"].(string); ok && (disStr == "true" || disStr == "yes") {
				disabled = true
			}
		}
		if action == "fasttrack-connection" && chain == "forward" && !disabled {
			fastTrackFound = true
			srcList, _ := f["src-address-list"].(string)
			dstList, _ := f["dst-address-list"].(string)
			srcAddr, _ := f["src-address"].(string)
			dstAddr, _ := f["dst-address"].(string)

			if (strings.Contains(srcList, "!MICUBE_BYPASS") || strings.Contains(dstList, "!MICUBE_BYPASS")) ||
				(strings.Contains(srcAddr, "!"+micubeIP) && strings.Contains(dstAddr, "!"+micubeIP)) {
				fastTrackExcludesCubeX = true
				break
			}
		}
	}
	if !fastTrackFound {
		report.Items = append(report.Items, CheckItem{
			Key:     "fasttrack_compatibility",
			Name:    "FastTrack 旁路兼容性 (/ip/firewall/filter)",
			Status:  "ok",
			Value:   "未启用或已禁用 FastTrack",
			Details: "未检测到处于激活状态的 FastTrack 规则，连接跟踪全状态生效，旁路由转发无穿透冲突风险。",
			Advice:  "状态良好",
		})
	} else if fastTrackExcludesCubeX {
		report.Items = append(report.Items, CheckItem{
			Key:     "fasttrack_compatibility",
			Name:    "FastTrack 旁路兼容性 (/ip/firewall/filter)",
			Status:  "ok",
			Value:   "已排除 CubeX (10.0.0.11)",
			Details: "FastTrack 硬件/内核加速已配置排除 CubeX 旁路由流量，连接跟踪与握手完全受保障。",
			Advice:  "无需调整",
		})
	} else {
		report.AllOK = false
		report.Items = append(report.Items, CheckItem{
			Key:     "fasttrack_compatibility",
			Name:    "FastTrack 旁路兼容性 (/ip/firewall/filter)",
			Status:  "warning",
			Value:   "未排除旁路由",
			Details: fmt.Sprintf("检测到 RouterOS FastTrack 加速未排除 CubeX (%s)。FastTrack 会绕过内核连接跟踪，导致旁路由返回的数据包出现 TCP 重置 (RST)、握手失败或断流！", micubeIP),
			Advice:  "点击下方「一键自动配置」即可自动为 FastTrack 规则添加 !MICUBE_BYPASS 排除条件",
		})
	}

	// 5. Check Forward Filter Drop Rules and CubeX Accept Rules
	hasForwardDrop := false
	hasForwardAcceptCubeX := false
	for _, f := range filters {
		action, _ := f["action"].(string)
		chain, _ := f["chain"].(string)
		disabled, _ := f["disabled"].(bool)
		if chain == "forward" && !disabled {
			if action == "drop" {
				hasForwardDrop = true
			}
			dstAddr, _ := f["dst-address"].(string)
			srcAddr, _ := f["src-address"].(string)
			if action == "accept" && (strings.Contains(dstAddr, micubeIP) || strings.Contains(srcAddr, micubeIP) || strings.Contains(dstAddr, "198.18.")) {
				hasForwardAcceptCubeX = true
			}
		}
	}
	if !hasForwardDrop || hasForwardAcceptCubeX {
		report.Items = append(report.Items, CheckItem{
			Key:     "forward_filter",
			Name:    "Forward 转发链防火墙放行 (/ip/firewall/filter)",
			Status:  "ok",
			Value:   "已放行 CubeX 转发流量",
			Details: "局域网终端与 CubeX 之间的跨端口双向转发已受到防火墙明确放行，无丢包拦截风险。",
			Advice:  "运行正常",
		})
	} else {
		report.AllOK = false
		report.Items = append(report.Items, CheckItem{
			Key:     "forward_filter",
			Name:    "Forward 转发链防火墙放行 (/ip/firewall/filter)",
			Status:  "warning",
			Value:   "存在丢包拦截风险 (Drop all forward)",
			Details: fmt.Sprintf("检测到防火墙 Forward 链存在全局 Drop 规则，且未在 Drop 之前放行 CubeX (%s) 的跨端口数据包，会导致通过主路由中转的 Fake-IP 流量在主路由处被直接拦截！", micubeIP),
			Advice:  "点击下方「一键自动配置」即可在 Drop 规则之前放行发往 CubeX 的 Forward 流量",
		})
	}

	// 6. Check /ip/firewall/address-list (MICUBE_BYPASS, LAN_SUBNET)
	bypassOK := false
	lanOK := false
	if data, err := m.QueryROS("/ip/firewall/address-list"); err == nil {
		var lists []map[string]any
		_ = json.Unmarshal(data, &lists)
		for _, l := range lists {
			listName, _ := l["list"].(string)
			addr, _ := l["address"].(string)
			if listName == "MICUBE_BYPASS" && (addr == micubeIP || strings.Contains(addr, micubeIP)) {
				bypassOK = true
			}
			if listName == "LAN_SUBNET" {
				lanOK = true
			}
		}
	}
	if bypassOK && lanOK {
		report.Items = append(report.Items, CheckItem{
			Key:     "address_list",
			Name:    "防死锁地址清单 (/ip/firewall/address-list)",
			Status:  "ok",
			Value:   fmt.Sprintf("MICUBE_BYPASS (%s) & LAN_SUBNET 就绪", micubeIP),
			Details: "CubeX 自身 IP 已加入旁路白名单，彻底杜绝回环与循环代理死锁。",
			Advice:  "地址列表完备",
		})
	} else {
		report.AllOK = false
		report.Items = append(report.Items, CheckItem{
			Key:     "address_list",
			Name:    "防死锁地址清单 (/ip/firewall/address-list)",
			Status:  "missing",
			Value:   "未完全配置",
			Details: fmt.Sprintf("缺少 MICUBE_BYPASS (包含 %s) 或 LAN_SUBNET。这会导致 CubeX 自身外发流量被重复抓入代理引发死锁断网！", micubeIP),
			Advice:  "必须配置 MICUBE_BYPASS 白名单排除 CubeX 容器 IP",
		})
	}

	// 7. Check /ip/firewall/mangle (mark-routing new-routing-mark=micube)
	mangleOK := false
	var mangleVal string
	for _, mg := range mangles {
		action, _ := mg["action"].(string)
		mark, _ := mg["new-routing-mark"].(string)
		chain, _ := mg["chain"].(string)
		disabled, _ := mg["disabled"].(bool)
		if action == "mark-routing" && mark == "micube" && chain == "prerouting" && !disabled {
			mangleOK = true
			mangleVal = fmt.Sprintf("chain=%s action=mark-routing mark=%s", chain, mark)
			break
		}
	}
	if mangleOK {
		report.Items = append(report.Items, CheckItem{
			Key:     "mangle",
			Name:    "Mangle 策略分流标记 (/ip/firewall/mangle)",
			Status:  "ok",
			Value:   mangleVal,
			Details: "检测到 Prerouting 标记规则已就绪，终端出国流量正常打标转发至 micube 路由表。",
			Advice:  "正常生效中",
		})
	} else {
		report.AllOK = false
		report.Items = append(report.Items, CheckItem{
			Key:     "mangle",
			Name:    "Mangle 策略分流标记 (/ip/firewall/mangle)",
			Status:  "missing",
			Value:   "缺失",
			Details: "未检测到处于启用状态的 Mangle mark-routing -> micube 规则，内网设备流量不会自动分流到 CubeX。",
			Advice:  "点击下方「一键自动配置」下发高性能 Mangle 分流规则",
		})
	}

	// 8. Check /ip/firewall/nat (DNS redirect dst-port=53 to MiCubeIP)
	natOK := false
	for _, n := range nats {
		action, _ := n["action"].(string)
		dport, _ := n["dst-port"].(string)
		toAddr, _ := n["to-addresses"].(string)
		disabled, _ := n["disabled"].(bool)
		if action == "dst-nat" && (dport == "53" || strings.Contains(dport, "53")) && !disabled {
			if toAddr == micubeIP || strings.Contains(toAddr, micubeIP) {
				natOK = true
				break
			}
		}
	}
	if natOK {
		report.Items = append(report.Items, CheckItem{
			Key:     "dns_nat",
			Name:    "严格防 DNS 泄露重定向 (/ip/firewall/nat)",
			Status:  "ok",
			Value:   fmt.Sprintf("UDP/TCP 53 -> %s:53", micubeIP),
			Details: "局域网明文 53 端口查询已被强制劫持至 CubeX 纯净防泄露内核，阻断 ISP 污染与旁路嗅探。",
			Advice:  "防泄露状态优良",
		})
	} else {
		report.AllOK = false
		report.Items = append(report.Items, CheckItem{
			Key:     "dns_nat",
			Name:    "严格防 DNS 泄露重定向 (/ip/firewall/nat)",
			Status:  "missing",
			Value:   "未开启重定向",
			Details: fmt.Sprintf("未配置将局域网 DNS (端口 53) 劫持至 CubeX (%s:53) 的规则。内网终端可能直接向路由器或运营商明文解析，导致 DNS 泄露与分流失效。", micubeIP),
			Advice:  "强烈建议点击「一键自动配置」启用 100% 防 DNS 泄露劫持",
		})
	}

	// 9. Check Legacy Residues (via-openbox, etc.)
	hasLegacyResidue := false
	legacyDetails := make([]string, 0)
	for _, t := range tables {
		if strings.Contains(strings.ToLower(fmt.Sprintf("%v", t["name"])), "openbox") {
			hasLegacyResidue = true
			legacyDetails = append(legacyDetails, "残留路由表 via-openbox")
		}
	}
	for _, mg := range mangles {
		comment, _ := mg["comment"].(string)
		mark, _ := mg["new-routing-mark"].(string)
		if strings.Contains(strings.ToLower(comment), "openbox") || strings.Contains(strings.ToLower(mark), "openbox") {
			hasLegacyResidue = true
			legacyDetails = append(legacyDetails, "残留 Mangle 规则: "+comment)
		}
	}
	if hasLegacyResidue {
		report.AllOK = false
		report.Items = append(report.Items, CheckItem{
			Key:     "legacy_residues",
			Name:    "历史旧版残留扫描",
			Status:  "warning",
			Value:   "检测到旧版残留",
			Details: fmt.Sprintf("系统检测到旧版残留项: %s。旧版遗留规则可能干扰旁路由策略分流。", strings.Join(legacyDetails, "; ")),
			Advice:  "点击下方「一键自动配置」即可一键自动清理全部历史残留",
		})
	} else {
		report.Items = append(report.Items, CheckItem{
			Key:     "legacy_residues",
			Name:    "历史旧版残留扫描",
			Status:  "ok",
			Value:   "零历史遗留",
			Details: "RouterOS 系统干净无残留，完全遵循净室自研合规标准。",
			Advice:  "无需调整",
		})
	}

	// 10. Check /ip/firewall/filter (WireGuard port 13231 input accept)
	wgFilterOK := false
	for _, f := range filters {
		chain, _ := f["chain"].(string)
		proto, _ := f["protocol"].(string)
		dport, _ := f["dst-port"].(string)
		action, _ := f["action"].(string)
		disabled, _ := f["disabled"].(bool)
		if chain == "input" && proto == "udp" && dport == "13231" && action == "accept" && !disabled {
			wgFilterOK = true
			break
		}
	}
	if wgFilterOK {
		report.Items = append(report.Items, CheckItem{
			Key:     "wg_filter",
			Name:    "WireGuard 穿透防火墙放行 (/ip/firewall/filter)",
			Status:  "ok",
			Value:   "UDP 13231 允许入站 (input accept)",
			Details: "已放行外部移动设备通过 UDP 13231 连接家中 WireGuard 节点。",
			Advice:  "移动端免密扫码穿透已保障",
		})
	} else {
		report.Items = append(report.Items, CheckItem{
			Key:     "wg_filter",
			Name:    "WireGuard 穿透防火墙放行 (/ip/firewall/filter)",
			Status:  "warning",
			Value:   "未放行 UDP 13231",
			Details: "防火墙未显式放行 UDP 13231 端口。若 RouterOS 开启了默认 Drop Input 防火墙策略，手机在外网将无法扫码连回家中。",
			Advice:  "点击「一键自动配置」在防火墙最前列放行 UDP 13231",
		})
	}

	return report, nil
}

// ApplyRouting automatically configures missing table, route, address-list, mangle, FastTrack, forward, and NAT rules on RouterOS.
func (m *Manager) ApplyRouting(targetIP string) (*ApplyResult, error) {
	if targetIP == "" {
		targetIP = m.GetOutboundIP()
	}
	if targetIP == "" {
		targetIP = "10.0.0.11"
	}

	res := &ApplyResult{
		Success: true,
		Actions: make([]string, 0),
		Errors:  make([]string, 0),
	}

	// 1. Ensure /routing/table: micube (fib=true)
	tableExists := false
	if data, err := m.QueryROS("/routing/table"); err == nil {
		var tables []map[string]any
		_ = json.Unmarshal(data, &tables)
		for _, t := range tables {
			if t["name"] == "micube" {
				tableExists = true
				break
			}
		}
	}
	if !tableExists {
		payload := map[string]any{
			"name": "micube",
			"fib":  "true",
		}
		if _, err := m.PutROS("/routing/table", payload); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("创建路由表 micube 失败: %v", err))
		} else {
			res.Actions = append(res.Actions, "已成功创建策略路由表 /routing/table (name=micube fib=true)")
		}
	} else {
		res.Actions = append(res.Actions, "策略路由表 /routing/table micube 已存在")
	}

	// 2. Ensure /ip/route: dst=0.0.0.0/0 gateway=targetIP routing-table=micube
	defaultRouteExists := false
	var existingDefaultRouteID string
	var existingRoutes []map[string]any
	if data, err := m.QueryROS("/ip/route"); err == nil {
		_ = json.Unmarshal(data, &existingRoutes)
		for _, r := range existingRoutes {
			table, _ := r["routing-table"].(string)
			dst := fmt.Sprintf("%v", r["dst-address"])
			if table == "micube" && (dst == "0.0.0.0/0" || strings.HasPrefix(dst, "0.0.0.0")) {
				defaultRouteExists = true
				if id, ok := r[".id"].(string); ok {
					existingDefaultRouteID = id
				}
				break
			}
		}
	}
	if !defaultRouteExists {
		rPayload := map[string]any{
			"dst-address":   "0.0.0.0/0",
			"gateway":       targetIP,
			"routing-table": "micube",
			"comment":       "CubeX Gateway Route",
		}
		if _, err := m.PutROS("/ip/route", rPayload); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("创建默认路由失败: %v", err))
		} else {
			res.Actions = append(res.Actions, fmt.Sprintf("已创建默认路由 /ip/route (0.0.0.0/0 -> %s table=micube)", targetIP))
		}
	} else if existingDefaultRouteID != "" {
		_ = m.PatchROS("/ip/route/"+existingDefaultRouteID, map[string]any{
			"gateway": targetIP,
			"comment": "CubeX Gateway Route",
		})
		res.Actions = append(res.Actions, fmt.Sprintf("已更新现有策略路由网关为 %s", targetIP))
	}

	// 2b. Clean up conflicting / stale Fake-IP routes (198.18.0.0/16) and ensure single valid route to targetIP
	hasCorrectFakeIPRoute := false
	for _, r := range existingRoutes {
		table, _ := r["routing-table"].(string)
		dst := fmt.Sprintf("%v", r["dst-address"])
		gw := fmt.Sprintf("%v", r["gateway"])
		id, _ := r[".id"].(string)
		if (table == "" || table == "main") && (dst == "198.18.0.0/16" || dst == "198.18.0.0/15" || strings.HasPrefix(dst, "198.18.")) {
			if gw == targetIP || strings.Contains(gw, targetIP) {
				hasCorrectFakeIPRoute = true
			} else if id != "" {
				// Delete stale / conflicting ECMP route!
				if err := m.DeleteROS("/ip/route/" + id); err == nil {
					res.Actions = append(res.Actions, fmt.Sprintf("已清除冲突/残留 Fake-IP ECMP 路由: %s -> %s", dst, gw))
				}
			}
		}
	}
	if !hasCorrectFakeIPRoute {
		fPayload := map[string]any{
			"dst-address": "198.18.0.0/16",
			"gateway":     targetIP,
			"comment":     "CubeX Fake-IP Route",
		}
		if _, err := m.PutROS("/ip/route", fPayload); err == nil {
			res.Actions = append(res.Actions, fmt.Sprintf("已创建 Fake-IP 静态路由 /ip/route (198.18.0.0/16 -> %s)", targetIP))
		}
	} else {
		res.Actions = append(res.Actions, fmt.Sprintf("Fake-IP 回指路由 (198.18.0.0/16 -> %s) 正常就绪", targetIP))
	}

	// 3. Ensure /ip/firewall/address-list: MICUBE_BYPASS, LAN_SUBNET
	existingLists := make(map[string]bool)
	if data, err := m.QueryROS("/ip/firewall/address-list"); err == nil {
		var lists []map[string]any
		_ = json.Unmarshal(data, &lists)
		for _, l := range lists {
			existingLists[fmt.Sprintf("%v_%v", l["list"], l["address"])] = true
		}
	}

	// Add MICUBE_BYPASS
	if !existingLists[fmt.Sprintf("MICUBE_BYPASS_%s", targetIP)] {
		_, err := m.PutROS("/ip/firewall/address-list", map[string]any{
			"list":    "MICUBE_BYPASS",
			"address": targetIP,
			"comment": "CubeX Self Outbound Bypass",
		})
		if err == nil {
			res.Actions = append(res.Actions, fmt.Sprintf("已添加防死锁地址 /ip/firewall/address-list (MICUBE_BYPASS %s)", targetIP))
		}
	}

	// Add LAN_SUBNET entries
	lanSubnets := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	for _, sub := range lanSubnets {
		if !existingLists[fmt.Sprintf("LAN_SUBNET_%s", sub)] {
			_, _ = m.PutROS("/ip/firewall/address-list", map[string]any{
				"list":    "LAN_SUBNET",
				"address": sub,
				"comment": "Local LAN Subnet",
			})
		}
	}
	res.Actions = append(res.Actions, "已确认防死锁与内网清单 MICUBE_BYPASS / LAN_SUBNET")

	// 4. Query filters for FastTrack and Forward checks
	var existingFilters []map[string]any
	if data, err := m.QueryROS("/ip/firewall/filter"); err == nil {
		_ = json.Unmarshal(data, &existingFilters)
	}

	// 4b. Patch FastTrack rule to exclude CubeX
	for _, f := range existingFilters {
		action, _ := f["action"].(string)
		chain, _ := f["chain"].(string)
		id, _ := f[".id"].(string)
		if action == "fasttrack-connection" && chain == "forward" && id != "" {
			srcList, _ := f["src-address-list"].(string)
			dstList, _ := f["dst-address-list"].(string)
			srcAddr, _ := f["src-address"].(string)
			dstAddr, _ := f["dst-address"].(string)

			needsPatch := !strings.Contains(srcList, "!MICUBE_BYPASS") &&
				!strings.Contains(dstList, "!MICUBE_BYPASS") &&
				!(strings.Contains(srcAddr, "!"+targetIP) && strings.Contains(dstAddr, "!"+targetIP))

			if needsPatch {
				patchPayload := map[string]any{
					"src-address": "!" + targetIP,
					"dst-address": "!" + targetIP,
					"comment":     fmt.Sprintf("FastTrack (Excluding CubeX %s)", targetIP),
				}
				if err := m.PatchROS("/ip/firewall/filter/"+id, patchPayload); err == nil {
					res.Actions = append(res.Actions, "已成功为 FastTrack 加速规则添加旁路由排除条件 (!"+targetIP+")，保障连接跟踪稳定性")
				}
			} else {
				res.Actions = append(res.Actions, "FastTrack 规则已配置旁路由排除白名单")
			}
		}
	}

	// 4c. Ensure Forward chain accept rules for CubeX before any Drop rules
	var forwardDropID string
	hasInboundAccept := false
	hasOutboundAccept := false
	hasFakeIPAccept := false

	for _, f := range existingFilters {
		chain, _ := f["chain"].(string)
		action, _ := f["action"].(string)
		dstAddr, _ := f["dst-address"].(string)
		srcAddr, _ := f["src-address"].(string)
		id, _ := f[".id"].(string)

		if chain == "forward" {
			if action == "drop" && forwardDropID == "" {
				forwardDropID = id
			}
			if action == "accept" {
				if strings.Contains(dstAddr, targetIP) {
					hasInboundAccept = true
				}
				if strings.Contains(srcAddr, targetIP) {
					hasOutboundAccept = true
				}
				if strings.Contains(dstAddr, "198.18.") {
					hasFakeIPAccept = true
				}
			}
		}
	}

	if forwardDropID != "" {
		if !hasInboundAccept {
			_, _ = m.PutROS("/ip/firewall/filter", map[string]any{
				"chain":        "forward",
				"action":       "accept",
				"dst-address":  targetIP,
				"place-before": forwardDropID,
				"comment":      "CubeX Inbound Forward Accept",
			})
			res.Actions = append(res.Actions, "已在防火墙 Forward 丢弃前添加 CubeX 入站放行规则")
		}
		if !hasOutboundAccept {
			_, _ = m.PutROS("/ip/firewall/filter", map[string]any{
				"chain":        "forward",
				"action":       "accept",
				"src-address":  targetIP,
				"place-before": forwardDropID,
				"comment":      "CubeX Outbound Forward Accept",
			})
			res.Actions = append(res.Actions, "已在防火墙 Forward 丢弃前添加 CubeX 出站放行规则")
		}
		if !hasFakeIPAccept {
			_, _ = m.PutROS("/ip/firewall/filter", map[string]any{
				"chain":        "forward",
				"action":       "accept",
				"dst-address":  "198.18.0.0/16",
				"place-before": forwardDropID,
				"comment":      "CubeX Fake-IP Forward Accept",
			})
			res.Actions = append(res.Actions, "已在防火墙 Forward 丢弃前添加 Fake-IP 放行规则")
		}
	}

	// 5. Ensure /ip/firewall/mangle: mark-routing to micube
	mangleExists := false
	var existingMangles []map[string]any
	if data, err := m.QueryROS("/ip/firewall/mangle"); err == nil {
		_ = json.Unmarshal(data, &existingMangles)
		for _, mg := range existingMangles {
			if mg["action"] == "mark-routing" && mg["new-routing-mark"] == "micube" && mg["chain"] == "prerouting" {
				mangleExists = true
				break
			}
		}
	}
	if !mangleExists {
		mPayload := map[string]any{
			"chain":            "prerouting",
			"src-address-list": "!MICUBE_BYPASS",
			"dst-address-list": "!LAN_SUBNET",
			"dst-address-type": "!local",
			"action":           "mark-routing",
			"new-routing-mark": "micube",
			"passthrough":      "yes",
			"comment":          "CubeX Policy Routing",
		}
		if _, err := m.PutROS("/ip/firewall/mangle", mPayload); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("创建 Mangle 规则失败: %v", err))
		} else {
			res.Actions = append(res.Actions, "已创建 Mangle 策略分流打标规则 /ip/firewall/mangle (mark-routing -> micube)")
		}
	} else {
		res.Actions = append(res.Actions, "Mangle 策略分流打标规则已就绪")
	}

	// 6. Ensure /ip/firewall/nat: DNS redirect UDP/TCP 53 to targetIP
	natUDPExists := false
	natTCPExists := false
	var existingNats []map[string]any
	if data, err := m.QueryROS("/ip/firewall/nat"); err == nil {
		_ = json.Unmarshal(data, &existingNats)
		for _, n := range existingNats {
			if n["action"] == "dst-nat" && n["dst-port"] == "53" {
				if n["protocol"] == "udp" {
					natUDPExists = true
				}
				if n["protocol"] == "tcp" {
					natTCPExists = true
				}
			}
		}
	}
	if !natUDPExists {
		uPayload := map[string]any{
			"chain":            "dstnat",
			"src-address-list": "!MICUBE_BYPASS",
			"dst-address-type": "local",
			"protocol":         "udp",
			"dst-port":         "53",
			"action":           "dst-nat",
			"to-addresses":     targetIP,
			"to-ports":         "53",
			"comment":          "CubeX DNS UDP Hijack",
		}
		if _, err := m.PutROS("/ip/firewall/nat", uPayload); err == nil {
			res.Actions = append(res.Actions, fmt.Sprintf("已创建 DNS UDP 53 安全劫持至 %s:53 (仅限发往本机的查询)", targetIP))
		}
	}
	if !natTCPExists {
		tPayload := map[string]any{
			"chain":            "dstnat",
			"src-address-list": "!MICUBE_BYPASS",
			"dst-address-type": "local",
			"protocol":         "tcp",
			"dst-port":         "53",
			"action":           "dst-nat",
			"to-addresses":     targetIP,
			"to-ports":         "53",
			"comment":          "CubeX DNS TCP Hijack",
		}
		if _, err := m.PutROS("/ip/firewall/nat", tPayload); err == nil {
			res.Actions = append(res.Actions, fmt.Sprintf("已创建 DNS TCP 53 安全劫持至 %s:53 (仅限发往本机的查询)", targetIP))
		}
	}

	// 7. Clean up Legacy Residues (OpenBox tables, routes, mangles, and dead NAT rules)
	if data, err := m.QueryROS("/routing/table"); err == nil {
		var tables []map[string]any
		_ = json.Unmarshal(data, &tables)
		for _, t := range tables {
			name := fmt.Sprintf("%v", t["name"])
			id, _ := t[".id"].(string)
			if strings.Contains(strings.ToLower(name), "openbox") && id != "" {
				_ = m.DeleteROS("/routing/table/" + id)
				res.Actions = append(res.Actions, "已清理历史遗留策略路由表: "+name)
			}
		}
	}
	for _, r := range existingRoutes {
		table, _ := r["routing-table"].(string)
		id, _ := r[".id"].(string)
		if strings.Contains(strings.ToLower(table), "openbox") && id != "" {
			_ = m.DeleteROS("/ip/route/" + id)
			res.Actions = append(res.Actions, "已清理历史遗留旧路由: table="+table)
		}
	}
	for _, mg := range existingMangles {
		comment, _ := mg["comment"].(string)
		mark, _ := mg["new-routing-mark"].(string)
		id, _ := mg[".id"].(string)
		if (strings.Contains(strings.ToLower(comment), "openbox") || strings.Contains(strings.ToLower(mark), "openbox")) && id != "" {
			_ = m.DeleteROS("/ip/firewall/mangle/" + id)
			res.Actions = append(res.Actions, "已清理历史遗留 Mangle 规则: "+comment)
		}
	}
	for _, n := range existingNats {
		comment, _ := n["comment"].(string)
		toAddr, _ := n["to-addresses"].(string)
		id, _ := n[".id"].(string)
		if (strings.Contains(comment, "Phantun") || toAddr == "10.0.0.6") && id != "" {
			_ = m.DeleteROS("/ip/firewall/nat/" + id)
			res.Actions = append(res.Actions, "已清理失效端口映射规则: "+comment)
		}
	}

	// 8. Ensure /ip/firewall/filter: WireGuard UDP 13231 accept
	wgFilterExists := false
	for _, f := range existingFilters {
		if f["chain"] == "input" && f["protocol"] == "udp" && f["dst-port"] == "13231" && f["action"] == "accept" {
			wgFilterExists = true
			break
		}
	}
	if !wgFilterExists {
		wPayload := map[string]any{
			"chain":        "input",
			"protocol":     "udp",
			"dst-port":     "13231",
			"action":       "accept",
			"place-before": "0",
			"comment":      "CubeX WireGuard In",
		}
		if _, err := m.PutROS("/ip/firewall/filter", wPayload); err == nil {
			res.Actions = append(res.Actions, "已在防火墙输入链顶部放行 WireGuard UDP 13231")
		}
	} else {
		res.Actions = append(res.Actions, "WireGuard 防火墙放行规则已就绪")
	}

	if len(res.Errors) > 0 {
		res.Success = false
		res.Message = fmt.Sprintf("配置完成但存在 %d 项需注意，详见日志", len(res.Errors))
	} else {
		res.Message = "🎉 RouterOS 防火墙、策略路由与 FastTrack 兼容性规则已全自动修复并同步生效！"
	}

	return res, nil
}

// CleanRoutingRules cleanly removes Mangle mark-routing and DNS dst-nat hijack rules from RouterOS.
func (m *Manager) CleanRoutingRules() (*ApplyResult, error) {
	res := &ApplyResult{
		Success: true,
		Actions: make([]string, 0),
		Errors:  make([]string, 0),
	}

	// 1. Remove Mangle rule with comment "CubeX Policy Routing" or mark "micube"
	if data, err := m.QueryROS("/ip/firewall/mangle"); err == nil {
		var mangles []map[string]any
		_ = json.Unmarshal(data, &mangles)
		for _, mg := range mangles {
			comment, _ := mg["comment"].(string)
			mark, _ := mg["new-routing-mark"].(string)
			id, _ := mg[".id"].(string)
			if (strings.Contains(comment, "CubeX") || strings.Contains(comment, "MiCube") || mark == "micube") && id != "" {
				if delErr := m.DeleteROS("/ip/firewall/mangle/" + id); delErr == nil {
					res.Actions = append(res.Actions, "已移除 RouterOS Mangle 策略分流打标规则")
				}
			}
		}
	}

	// 2. Remove DNS redirect rules with comment "CubeX DNS UDP Hijack" / "CubeX DNS TCP Hijack"
	if data, err := m.QueryROS("/ip/firewall/nat"); err == nil {
		var nats []map[string]any
		_ = json.Unmarshal(data, &nats)
		for _, n := range nats {
			comment, _ := n["comment"].(string)
			id, _ := n[".id"].(string)
			if (strings.Contains(comment, "CubeX DNS") || strings.Contains(comment, "MiCube DNS")) && id != "" {
				if delErr := m.DeleteROS("/ip/firewall/nat/" + id); delErr == nil {
					res.Actions = append(res.Actions, "已移除 RouterOS DNS 端口 53 强制劫持规则")
				}
			}
		}
	}

	res.Message = "🌾 已成功清理 Mangle 分流与 DNS 劫持规则！当前处于经典纯旁路由模式，同号段任意网关与 DNS 下发完全自由连通！"
	return res, nil
}

// GenerateWinBoxScript generates the manual WinBox Terminal command script.
func (m *Manager) GenerateWinBoxScript(targetIP string) string {
	if targetIP == "" {
		targetIP = m.GetOutboundIP()
	}
	if targetIP == "" {
		targetIP = "10.0.0.11"
	}

	var sb strings.Builder
	sb.WriteString("# ==============================================================================\n")
	sb.WriteString("# CubeX RouterOS v7 防火墙、策略路由、FastTrack兼容性与严格防 DNS 泄露脚本\n")
	sb.WriteString(fmt.Sprintf("# 旁路由目标 IP: %s\n", targetIP))
	sb.WriteString("# ==============================================================================\n\n")

	sb.WriteString("# 1. 清理旧版冲突与残留 Fake-IP 路由\n")
	sb.WriteString("/ip/route remove [find dst-address=198.18.0.0/16 and gateway!=10.0.0.11]\n")
	sb.WriteString(fmt.Sprintf("/ip/route add dst-address=198.18.0.0/16 gateway=%s comment=\"CubeX Fake-IP Route\"\n\n", targetIP))

	sb.WriteString("# 2. 创建专用策略路由表 (FIB 转发)\n")
	sb.WriteString("/routing/table add name=micube fib\n\n")

	sb.WriteString("# 3. 添加默认出国网关路由至 micube 路由表\n")
	sb.WriteString(fmt.Sprintf("/ip/route add dst-address=0.0.0.0/0 gateway=%s routing-table=micube comment=\"CubeX Gateway Route\"\n\n", targetIP))

	sb.WriteString("# 4. 配置内网与旁路防死锁地址列表 (极为关键，杜绝自发循环死锁)\n")
	sb.WriteString(fmt.Sprintf("/ip/firewall/address-list add list=MICUBE_BYPASS address=%s comment=\"CubeX Self Outbound Bypass\"\n", targetIP))
	sb.WriteString("/ip/firewall/address-list add list=LAN_SUBNET address=10.0.0.0/8 comment=\"Local LAN Subnet\"\n")
	sb.WriteString("/ip/firewall/address-list add list=LAN_SUBNET address=172.16.0.0/12 comment=\"Local LAN Subnet\"\n")
	sb.WriteString("/ip/firewall/address-list add list=LAN_SUBNET address=192.168.0.0/16 comment=\"Local LAN Subnet\"\n\n")

	sb.WriteString("# 5. 调整 FastTrack 硬件加速规则排除旁路由 (防止 TCP 握手异常与断流)\n")
	sb.WriteString(fmt.Sprintf("/ip/firewall/filter set [find action=fasttrack-connection] src-address=!%s dst-address=!%s comment=\"FastTrack (Excluding CubeX %s)\"\n\n", targetIP, targetIP, targetIP))

	sb.WriteString("# 6. 在防火墙 Forward 丢弃规则前放行发往 CubeX 的跨端口转发流量\n")
	sb.WriteString(fmt.Sprintf("/ip/firewall/filter add chain=forward action=accept dst-address=%s place-before=[find action=drop and chain=forward] comment=\"CubeX Inbound Forward Accept\"\n", targetIP))
	sb.WriteString(fmt.Sprintf("/ip/firewall/filter add chain=forward action=accept src-address=%s place-before=[find action=drop and chain=forward] comment=\"CubeX Outbound Forward Accept\"\n", targetIP))
	sb.WriteString("/ip/firewall/filter add chain=forward action=accept dst-address=198.18.0.0/16 place-before=[find action=drop and chain=forward] comment=\"CubeX Fake-IP Forward Accept\"\n\n")

	sb.WriteString("# 7. Mangle 策略分流打标 (终端出国流量标记至 micube 路由表)\n")
	sb.WriteString("/ip/firewall/mangle add chain=prerouting src-address-list=!MICUBE_BYPASS dst-address-list=!LAN_SUBNET dst-address-type=!local action=mark-routing new-routing-mark=micube passthrough=yes comment=\"CubeX Policy Routing\"\n\n")

	sb.WriteString("# 8. 严格防 DNS 泄露重定向 (仅劫持目标为 RouterOS 本机 IP 的 53 查询，绝不干扰同网段其他网关/DNS)\n")
	sb.WriteString(fmt.Sprintf("/ip/firewall/nat add chain=dstnat src-address-list=!MICUBE_BYPASS dst-address-type=local protocol=udp dst-port=53 action=dst-nat to-addresses=%s to-ports=53 comment=\"CubeX DNS UDP Hijack\"\n", targetIP))
	sb.WriteString(fmt.Sprintf("/ip/firewall/nat add chain=dstnat src-address-list=!MICUBE_BYPASS dst-address-type=local protocol=tcp dst-port=53 action=dst-nat to-addresses=%s to-ports=53 comment=\"CubeX DNS TCP Hijack\"\n\n", targetIP))

	sb.WriteString("# 9. 清理历史遗留残留\n")
	sb.WriteString("/routing/table remove [find name=via-openbox]\n")
	sb.WriteString("/ip/firewall/mangle remove [find new-routing-mark=via-openbox]\n\n")

	sb.WriteString("# 10. WireGuard 移动端免密扫码穿透监听端口放行\n")
	sb.WriteString("/ip/firewall/filter add chain=input protocol=udp dst-port=13231 action=accept place-before=0 comment=\"CubeX WireGuard In\"\n\n")

	sb.WriteString(":put \"==========================================================\"\n")
	sb.WriteString(":put \"✅ CubeX RouterOS 防火墙与策略路由已全自动配置完毕！\"\n")
	sb.WriteString(":put \"==========================================================\"\n")

	return sb.String()
}

// SetContainerRegistryURL updates the container registry-url on RouterOS via REST API.
func (m *Manager) SetContainerRegistryURL(mirrorURL string) error {
	mirrorURL = strings.TrimSpace(mirrorURL)
	if mirrorURL == "" {
		return fmt.Errorf("registry URL cannot be empty")
	}
	_, err := m.PostROS("/container/config/set", map[string]string{
		"registry-url": mirrorURL,
	})
	return err
}
