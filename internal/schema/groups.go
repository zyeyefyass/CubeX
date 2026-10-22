package schema

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// RegionDef defines standard geographic routing clusters.
type RegionDef struct {
	ID           string
	Name         string
	Flag         string
	Regex        *regexp.Regexp
	AutoName     string
	FallName     string
	DisasterName string
}

// StandardRegions defines standard geographic and custom node routing clusters.
var StandardRegions = []RegionDef{
	{
		ID:           "HK",
		Name:         "香港",
		Flag:         "🇭🇰",
		Regex:        regexp.MustCompile(`(?i)(港|hk|hong kong)`),
		AutoName:     "🇭🇰 香港自动",
		FallName:     "🇭🇰 香港故转",
		DisasterName: "🇭🇰 香港容灾",
	},
	{
		ID:           "TW",
		Name:         "台湾",
		Flag:         "🇹🇼",
		Regex:        regexp.MustCompile(`(?i)(台|tw|taiwan)`),
		AutoName:     "🇹🇼 台湾自动",
		FallName:     "🇹🇼 台湾故转",
		DisasterName: "🇹🇼 台湾容灾",
	},
	{
		ID:           "JP",
		Name:         "日本",
		Flag:         "🇯🇵",
		Regex:        regexp.MustCompile(`(?i)(日|jp|japan|东京|大阪)`),
		AutoName:     "🇯🇵 日本自动",
		FallName:     "🇯🇵 日本故转",
		DisasterName: "🇯🇵 日本容灾",
	},
	{
		ID:           "KR",
		Name:         "韩国",
		Flag:         "🇰🇷",
		Regex:        regexp.MustCompile(`(?i)(韩|kr|korea|首尔)`),
		AutoName:     "🇰🇷 韩国自动",
		FallName:     "🇰🇷 韩国故转",
		DisasterName: "🇰🇷 韩国容灾",
	},
	{
		ID:           "SG",
		Name:         "新加坡",
		Flag:         "🇸🇬",
		Regex:        regexp.MustCompile(`(?i)(新|sg|singapore|狮城)`),
		AutoName:     "🇸🇬 新加坡自动",
		FallName:     "🇸🇬 新加坡故转",
		DisasterName: "🇸🇬 新加坡容灾",
	},
	{
		ID:           "US",
		Name:         "美国",
		Flag:         "🇺🇸",
		Regex:        regexp.MustCompile(`(?i)(美|us|united states|america|洛杉矶|硅谷|西雅图|达拉斯)`),
		AutoName:     "🇺🇸 美国自动",
		FallName:     "🇺🇸 美国故转",
		DisasterName: "🇺🇸 美国容灾",
	},
	{
		ID:           "CUSTOM",
		Name:         "自建",
		Flag:         "🏠",
		Regex:        nil, // Handled specifically for manually added nodes
		AutoName:     "🏠 自建自动",
		FallName:     "🏠 自建故转",
		DisasterName: "🏠 自建容灾",
	},
	{
		ID:           "OTHERS",
		Name:         "其它",
		Flag:         "🌐",
		Regex:        nil, // Catch-all for non-HK/TW/JP/KR/SG/US airport nodes
		AutoName:     "🌐 其它自动",
		FallName:     "🌐 其它故转",
		DisasterName: "🌐 其它容灾",
	},
}

// RegionalFailoverOrder defines the cross-region multi-active disaster failover priority
var RegionalFailoverOrder = map[string][]string{
	"HK":     {"TW", "SG", "JP", "KR", "US"},
	"TW":     {"HK", "JP", "SG", "KR", "US"},
	"JP":     {"KR", "SG", "HK", "TW", "US"},
	"KR":     {"JP", "HK", "SG", "TW", "US"},
	"SG":     {"HK", "JP", "TW", "KR", "US"},
	"US":     {"TW", "JP", "KR"},
	"CUSTOM": {"HK", "JP", "SG", "TW", "KR", "US"},
	"OTHERS": {"HK", "JP", "SG", "TW", "KR", "US"},
}

// StandardBusinessGroups defines standard application routing groups matching user's config.yaml.
var StandardBusinessGroups = []struct {
	Name string
	Icon string
}{
	{Name: "🎯 国内流量", Icon: "direct"},
	{Name: "🌐 国外流量", Icon: "globe"},
	{Name: "AI", Icon: "openai"},
	{Name: "Gemini & Antigravity", Icon: "google"},
	{Name: "YouTube", Icon: "youtube"},
	{Name: "Google", Icon: "google"},
	{Name: "GitHub & Docker", Icon: "github"},
	{Name: "Telegram", Icon: "telegram"},
	{Name: "Twitter (X)", Icon: "twitter"},
	{Name: "Netflix & 流媒体", Icon: "netflix"},
	{Name: "Apple 服务", Icon: "apple"},
	{Name: "Microsoft", Icon: "microsoft"},
	{Name: "Spotify", Icon: "spotify"},
	{Name: "Steam 游戏", Icon: "steam"},
	{Name: "🐟 漏网之鱼", Icon: "fish"},
}

// AlignNodeAirportPrefixes prepends subscription/airport name to proxy node names if not already present.
func AlignNodeAirportPrefixes(ir *ConfigIR) {
	if ir == nil || len(ir.Proxies) == 0 {
		return
	}

	subNameMap := make(map[string]string)
	for _, sub := range ir.Subscriptions {
		if sub.ID != "" && sub.Name != "" {
			subNameMap[sub.ID] = sub.Name
		}
	}

	renameMap := make(map[string]string)

	for i := range ir.Proxies {
		p := &ir.Proxies[i]
		if p.Type == "direct" || p.Type == "block" || p.Type == "reject" {
			continue
		}

		if p.SubID == "custom" || strings.HasPrefix(p.Name, "[自建]") || (p.SubID == "" && strings.Contains(strings.ToLower(p.Name), "自建")) {
			if !strings.HasPrefix(p.Name, "[") {
				oldName := p.Name
				newName := fmt.Sprintf("[自建] %s", oldName)
				p.Name = newName
				renameMap[oldName] = newName
			}
			continue
		}

		airportName := ""
		if p.SubID != "" && subNameMap[p.SubID] != "" {
			airportName = subNameMap[p.SubID]
		}

		if airportName != "" {
			prefix := fmt.Sprintf("[%s]", airportName)
			if !strings.HasPrefix(p.Name, "[") {
				oldName := p.Name
				newName := fmt.Sprintf("%s %s", prefix, oldName)
				p.Name = newName
				renameMap[oldName] = newName
			}
		}
	}

	if len(renameMap) == 0 {
		return
	}

	// Also update groups referencing old node names
	for gi := range ir.Groups {
		g := &ir.Groups[gi]
		if newNow, exists := renameMap[g.Now]; exists {
			g.Now = newNow
		}
		for pi := range g.Proxies {
			if newP, exists := renameMap[g.Proxies[pi]]; exists {
				g.Proxies[pi] = newP
			}
		}
	}
}

// SyncAllStrategyGroups dynamically aligns region clusters (24 groups) and business routing groups.
func SyncAllStrategyGroups(ir *ConfigIR) {
	if ir == nil {
		return
	}

	// 0. Ensure standard DNS port is strictly 53 (never leak or divert to 1053)
	if ir.Network.DNSPort == 0 || ir.Network.DNSPort == 1053 {
		ir.Network.DNSPort = 53
	}

	// 1. Ensure all nodes have airport prefixes
	AlignNodeAirportPrefixes(ir)

	// 1. Classify all proxies into regional buckets
	regionNodes := make(map[string][]string)
	for _, reg := range StandardRegions {
		regionNodes[reg.ID] = make([]string, 0)
	}

	for _, p := range ir.Proxies {
		if p.Type == "direct" || p.Type == "block" || p.Type == "reject" {
			continue
		}
		name := p.Name
		if isInformationalOrInvalidNode(name) {
			continue
		}

		// Check if it's a manually added custom node
		if p.SubID == "custom" || strings.HasPrefix(name, "[自建]") || (p.SubID == "" && strings.Contains(strings.ToLower(name), "自建")) {
			regionNodes["CUSTOM"] = append(regionNodes["CUSTOM"], name)
			continue
		}

		matched := false
		for _, reg := range StandardRegions {
			if reg.ID == "CUSTOM" || reg.ID == "OTHERS" {
				continue
			}
			// Strict isolation: HHost-JP node is strictly excluded from Taiwan groups
			if (reg.ID == "TW" || strings.Contains(reg.Name, "台湾")) && strings.Contains(strings.ToLower(name), "hhost") && strings.Contains(strings.ToLower(name), "jp") {
				continue
			}
			if reg.Regex != nil && reg.Regex.MatchString(name) {
				regionNodes[reg.ID] = append(regionNodes[reg.ID], name)
				matched = true
				break
			}
		}
		if !matched {
			if p.SubID == "custom" || strings.HasPrefix(name, "[自建]") || strings.Contains(name, "自建") {
				regionNodes["CUSTOM"] = append(regionNodes["CUSTOM"], name)
			} else {
				regionNodes["OTHERS"] = append(regionNodes["OTHERS"], name)
			}
		}
	}

	// Universal Node Quality Sorting: place dedicated lines (MPLS / IPLC / BGP / 优化) at the top
	for regID, nList := range regionNodes {
		sortNodesByQuality(nList)
		regionNodes[regID] = nList
	}

	// Existing groups map for preserving selections and custom additions
	existingMap := make(map[string]StrategyGroup)
	for _, g := range ir.Groups {
		existingMap[g.Name] = g
	}

	newGroups := make([]StrategyGroup, 0)

	// 3. Ensure all 21 Regional Strategy Groups exist (Sorted: 地区自动 -> 地区故转 -> 地区容灾)
	for _, reg := range StandardRegions {
		nodes := regionNodes[reg.ID]
		if len(nodes) == 0 {
			nodes = []string{"DIRECT"}
		}

		// (1) 地区自动 (url-test 最优延迟)
		autoGrp := StrategyGroup{
			Name:      reg.AutoName,
			Type:      "urltest",
			Proxies:   nodes,
			Now:       nodes[0],
			URL:       "http://www.gstatic.com/generate_204",
			Interval:  300,
			Tolerance: 50,
			Category:  "region",
			Region:    reg.ID,
			Policy:    "auto",
			Icon:      reg.Flag,
		}
		if eg, exists := existingMap[reg.AutoName]; exists && eg.Now != "" && (reg.ID == "CUSTOM" || eg.Now != "DIRECT") && containsString(nodes, eg.Now) {
			autoGrp.Now = eg.Now
		}
		newGroups = append(newGroups, autoGrp)

		// (2) 地区故转 (fallback 顺位主备)
		fallGrp := StrategyGroup{
			Name:      reg.FallName,
			Type:      "fallback",
			Proxies:   nodes,
			Now:       nodes[0],
			URL:       "http://www.gstatic.com/generate_204",
			Interval:  300,
			Category:  "region",
			Region:    reg.ID,
			Policy:    "fallback",
			Icon:      reg.Flag,
		}
		if eg, exists := existingMap[reg.FallName]; exists && eg.Now != "" && (reg.ID == "CUSTOM" || eg.Now != "DIRECT") && containsString(nodes, eg.Now) {
			fallGrp.Now = eg.Now
		}
		newGroups = append(newGroups, fallGrp)

		// (3) 地区容灾 (disaster 跨区多活容灾兜底: 首选本地地区节点 -> 灾备邻近地区节点 -> 全球节点)
		disasterNodes := append([]string{}, nodes...)
		if failoverRegions, ok := RegionalFailoverOrder[reg.ID]; ok {
			for _, fRegID := range failoverRegions {
				if fNodes, fOk := regionNodes[fRegID]; fOk && len(fNodes) > 0 {
					for _, fn := range fNodes {
						if fn != "DIRECT" && !containsString(disasterNodes, fn) {
							disasterNodes = append(disasterNodes, fn)
						}
					}
				}
			}
		}
		if !containsString(disasterNodes, "DIRECT") {
			disasterNodes = append(disasterNodes, "DIRECT")
		}
		disasterGrp := StrategyGroup{
			Name:      reg.DisasterName,
			Type:      "fallback",
			Proxies:   disasterNodes,
			Now:       disasterNodes[0],
			URL:       "http://www.gstatic.com/generate_204",
			Interval:  180,
			Category:  "region",
			Region:    reg.ID,
			Policy:    "disaster",
			Icon:      reg.Flag,
		}
		if eg, exists := existingMap[reg.DisasterName]; exists && eg.Now != "" && containsString(disasterNodes, eg.Now) {
			disasterGrp.Now = eg.Now
		}
		newGroups = append(newGroups, disasterGrp)
	}

	// 4. Ensure Business Routing Groups exist (Order: 地区自动, 地区故转, 地区容灾)
	allRegionalCandidateProxies := make([]string, 0)
	for _, reg := range StandardRegions {
		allRegionalCandidateProxies = append(allRegionalCandidateProxies, reg.AutoName, reg.FallName, reg.DisasterName)
	}
	// Append concrete proxy nodes as well for granular manual override
	for _, p := range ir.Proxies {
		if p.Type != "direct" && p.Type != "block" && p.Type != "reject" {
			allRegionalCandidateProxies = append(allRegionalCandidateProxies, p.Name)
		}
	}
	allRegionalCandidateProxies = append(allRegionalCandidateProxies, "DIRECT")

	for _, bg := range StandardBusinessGroups {
		candidates := append([]string{}, allRegionalCandidateProxies...)

		defaultNow := candidates[0]
		if bg.Name == "Apple 服务" || bg.Name == "Microsoft" || bg.Name == "🎯 国内流量" {
			defaultNow = "DIRECT"
		} else if bg.Name == "AI" || bg.Name == "Gemini & Antigravity" {
			for _, pref := range []string{"🇺🇸 美国自动", "🇯🇵 日本自动", "🇹🇼 台湾自动", "🇸🇬 新加坡自动"} {
				if containsString(candidates, pref) {
					defaultNow = pref
					break
				}
			}
		} else if bg.Name == "YouTube" || bg.Name == "Netflix & 流媒体" || bg.Name == "Google" {
			for _, pref := range []string{"🇯🇵 日本自动", "🇭🇰 香港自动", "🇸🇬 新加坡自动", "🇹🇼 台湾自动"} {
				if containsString(candidates, pref) {
					defaultNow = pref
					break
				}
			}
		} else {
			// All other foreign groups (🌐 国外流量, 🐟 漏网之鱼, Telegram, Twitter, GitHub, etc.)
			for _, pref := range []string{"🇭🇰 香港自动", "🇯🇵 日本自动", "🇸🇬 新加坡自动", "🇹🇼 台湾自动", "🇺🇸 美国自动"} {
				if containsString(candidates, pref) {
					defaultNow = pref
					break
				}
			}
		}

		bizGrp := StrategyGroup{
			Name:     bg.Name,
			Type:     "select",
			Proxies:  candidates,
			Now:      defaultNow,
			Category: "business",
			Icon:     bg.Icon,
			IsCustom: false,
		}

		if eg, exists := existingMap[bg.Name]; exists {
			if eg.Now != "" && containsString(candidates, eg.Now) {
				if eg.Now == "DIRECT" && bg.Name != "Apple 服务" && bg.Name != "Microsoft" && bg.Name != "🎯 国内流量" {
					bizGrp.Now = defaultNow
				} else if (bg.Name == "AI" || bg.Name == "Gemini & Antigravity") && (eg.Now == "🇭🇰 香港自动" || eg.Now == "🇭🇰 香港故转" || eg.Now == "🇭🇰 香港容灾") {
					bizGrp.Now = defaultNow

				} else {
					bizGrp.Now = eg.Now
				}
			}
			if len(eg.Proxies) > 0 {
				bizGrp.Proxies = mergeProxiesUnique(candidates, eg.Proxies)
			}
		} else if bg.Name == "AI" || bg.Name == "Gemini & Antigravity" {
			for _, oldName := range []string{"OpenAI & AI", "OpenAI / Claude", "OpenAI", "ChatGPT"} {
				if eg, ok := existingMap[oldName]; ok {
					if eg.Now != "" && containsString(candidates, eg.Now) {
						if eg.Now == "DIRECT" {
							bizGrp.Now = defaultNow
						} else {
							bizGrp.Now = eg.Now
						}
					}
					if len(eg.Proxies) > 0 {
						bizGrp.Proxies = mergeProxiesUnique(candidates, eg.Proxies)
					}
					break
				}
			}
		}
		newGroups = append(newGroups, bizGrp)
	}

	// 5. Preserve user custom-added business groups (filter out corrupt or placeholder names)
	for _, g := range ir.Groups {
		if g.IsCustom {
			// Skip corrupted names like '?? Spotify??' or empty
			if strings.Contains(g.Name, "??") || strings.Contains(g.Name, "\ufffd") || strings.TrimSpace(g.Name) == "" {
				continue
			}
			// Strict isolation: if custom group name contains "台湾", exclude HHost-JP
			if strings.Contains(g.Name, "台湾") {
				filteredProxies := make([]string, 0, len(g.Proxies))
				for _, px := range g.Proxies {
					pxLower := strings.ToLower(px)
					if strings.Contains(pxLower, "hhost") && strings.Contains(pxLower, "jp") {
						continue
					}
					filteredProxies = append(filteredProxies, px)
				}
				g.Proxies = filteredProxies
				if strings.Contains(strings.ToLower(g.Now), "hhost") && strings.Contains(strings.ToLower(g.Now), "jp") {
					if len(g.Proxies) > 0 {
						g.Now = g.Proxies[0]
					}
				}
			}
			// Skip if already in newGroups
			found := false
			for _, ng := range newGroups {
				if ng.Name == g.Name {
					found = true
					break
				}
			}
			if !found {
				newGroups = append(newGroups, g)
			}
		}
	}

	ir.Groups = newGroups
}

// GetRegionSummary returns statistics for the 7 geographic regions.
type RegionStat struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Flag          string   `json:"flag"`
	TotalNodes    int      `json:"total_nodes"`
	ActiveNode    string   `json:"active_node"`
	LatencyMs     int      `json:"latency_ms"`
	AutoGroup     string   `json:"auto_group"`
	FallGroup     string   `json:"fall_group"`
	DisasterGroup string   `json:"disaster_group"`
	NodeNames     []string `json:"node_names"`
}

func ExtractRegionStats(ir *ConfigIR) []RegionStat {
	if ir == nil {
		return nil
	}

	SyncAllStrategyGroups(ir)

	stats := make([]RegionStat, 0, len(StandardRegions))
	groupMap := make(map[string]StrategyGroup)
	for _, g := range ir.Groups {
		groupMap[g.Name] = g
	}

	for _, reg := range StandardRegions {
		autoGrp := groupMap[reg.AutoName]
		nodes := make([]string, 0)
		for _, p := range autoGrp.Proxies {
			if p != "DIRECT" && p != "REJECT" {
				nodes = append(nodes, p)
			}
		}

		stat := RegionStat{
			ID:            reg.ID,
			Name:          reg.Name,
			Flag:          reg.Flag,
			TotalNodes:    len(nodes),
			ActiveNode:    autoGrp.Now,
			LatencyMs:     32, // calculated or default
			AutoGroup:     reg.AutoName,
			FallGroup:     reg.FallName,
			DisasterGroup: reg.DisasterName,
			NodeNames:     nodes,
		}

		if stat.ActiveNode == "" && len(nodes) > 0 {
			stat.ActiveNode = nodes[0]
		}
		stats = append(stats, stat)
	}

	return stats
}

func mergeProxiesUnique(base, extra []string) []string {
	seen := make(map[string]bool)
	res := make([]string, 0, len(base)+len(extra))
	for _, item := range base {
		if strings.HasSuffix(item, "容灾") {
			continue
		}
		if !seen[item] {
			seen[item] = true
			res = append(res, item)
		}
	}
	for _, item := range extra {
		if strings.HasSuffix(item, "容灾") {
			continue
		}
		if !seen[item] {
			seen[item] = true
			res = append(res, item)
		}
	}
	return res
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

// isInformationalOrInvalidNode filters out airport notices, traffic balance notices, and expired tags
func isInformationalOrInvalidNode(name string) bool {
	lower := strings.ToLower(name)
	keywords := []string{
		"剩余流量", "套餐到期", "官网", "官方", "下次重置", "到期时间",
		"订阅地址失效", "请重新复制", "如果只显示此节点", "客户端太旧",
		"公告", "通知", "更新客户端", "防失联", "群组", "频道",
		"重置剩余", "请到网站", "过期时间", "额度", "返利",
		"expire date", "expiredate", "traffic", "balance",
		"客服", "网址", "网站", "续费", "教程", "备用",
	}
	for _, kw := range keywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// sortNodesByQuality prioritizes high-speed, dedicated, low-latency nodes
func sortNodesByQuality(nodes []string) {
	sort.SliceStable(nodes, func(i, j int) bool {
		return getNodeQualityScore(nodes[i]) < getNodeQualityScore(nodes[j])
	})
}

func getNodeQualityScore(name string) int {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "mpls") || strings.Contains(lower, "iplc") || strings.Contains(lower, "专线") {
		return 0
	}
	if strings.Contains(lower, "优化") || strings.Contains(lower, "bgp") || strings.Contains(lower, "cn2") || strings.Contains(lower, "gia") || strings.Contains(lower, "9929") {
		return 1
	}
	if strings.Contains(lower, "hy2") || strings.Contains(lower, "hysteria") || strings.Contains(lower, "tuic") {
		return 2
	}
	if strings.Contains(lower, "原生") || strings.Contains(lower, "直连") {
		return 3
	}
	return 5
}


