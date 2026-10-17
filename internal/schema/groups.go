package schema

import (
	"fmt"
	"regexp"
	"strings"
)

// RegionDef defines standard geographic routing clusters.
type RegionDef struct {
	ID         string
	Name       string
	Flag       string
	Regex      *regexp.Regexp
	ManualName string
	AutoName   string
	FallName   string
	DisaName   string
}

// StandardRegions defines standard geographic and custom node routing clusters.
var StandardRegions = []RegionDef{
	{
		ID:         "HK",
		Name:       "香港",
		Flag:       "🇭🇰",
		Regex:      regexp.MustCompile(`(?i)(港|hk|hong kong)`),
		ManualName: "🇭🇰 香港手动",
		AutoName:   "🇭🇰 香港自动",
		FallName:   "🇭🇰 香港故转",
		DisaName:   "🇭🇰 香港容灾",
	},
	{
		ID:         "TW",
		Name:       "台湾",
		Flag:       "🇹🇼",
		Regex:      regexp.MustCompile(`(?i)(台|tw|taiwan)`),
		ManualName: "🇹🇼 台湾手动",
		AutoName:   "🇹🇼 台湾自动",
		FallName:   "🇹🇼 台湾故转",
		DisaName:   "🇹🇼 台湾容灾",
	},
	{
		ID:         "JP",
		Name:       "日本",
		Flag:       "🇯🇵",
		Regex:      regexp.MustCompile(`(?i)(hhost[-_ ]?jp|日|jp|japan|东京|大阪|tokyo|osaka|🇯🇵)`),
		ManualName: "🇯🇵 日本手动",
		AutoName:   "🇯🇵 日本自动",
		FallName:   "🇯🇵 日本故转",
		DisaName:   "🇯🇵 日本容灾",
	},
	{
		ID:         "SG",
		Name:       "新加坡",
		Flag:       "🇸🇬",
		Regex:      regexp.MustCompile(`(?i)(新|sg|singapore|狮城|🇸🇬)`),
		ManualName: "🇸🇬 新加坡手动",
		AutoName:   "🇸🇬 新加坡自动",
		FallName:   "🇸🇬 新加坡故转",
		DisaName:   "🇸🇬 新加坡容灾",
	},
	{
		ID:         "KR",
		Name:       "韩国",
		Flag:       "🇰🇷",
		Regex:      regexp.MustCompile(`(?i)(韩|kr|korea|首尔|🇰🇷)`),
		ManualName: "🇰🇷 韩国手动",
		AutoName:   "🇰🇷 韩国自动",
		FallName:   "🇰🇷 韩国故转",
		DisaName:   "🇰🇷 韩国容灾",
	},
	{
		ID:         "US",
		Name:       "美国",
		Flag:       "🇺🇸",
		Regex:      regexp.MustCompile(`(?i)(美|us|united states|america|洛杉矶|硅谷|西雅图|达拉斯)`),
		ManualName: "🇺🇸 美国手动",
		AutoName:   "🇺🇸 美国自动",
		FallName:   "🇺🇸 美国故转",
		DisaName:   "🇺🇸 美国容灾",
	},
	{
		ID:         "CUSTOM",
		Name:       "自建",
		Flag:       "🏠",
		Regex:      nil, // Handled specifically for manually added nodes
		ManualName: "🏠 自建手动",
		AutoName:   "🏠 自建自动",
		FallName:   "🏠 自建故转",
		DisaName:   "🏠 自建容灾",
	},
	{
		ID:         "OTHERS",
		Name:       "其它",
		Flag:       "🌐",
		Regex:      nil, // Catch-all for non-HK/TW/SG_JPKR/US airport nodes
		ManualName: "🌐 其它手动",
		AutoName:   "🌐 其它自动",
		FallName:   "🌐 其它故转",
		DisaName:   "🌐 其它容灾",
	},
}

// StandardBusinessGroups defines standard application routing groups matching user's config.yaml.
var StandardBusinessGroups = []struct {
	Name string
	Icon string
}{
	{Name: "AI", Icon: "openai"},
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

		// If sub_id missing, heuristically map based on node signature
		if airportName == "" {
			nameLower := strings.ToLower(p.Name)
			if strings.Contains(nameLower, "cloud") || strings.Contains(nameLower, "xtls") || strings.Contains(nameLower, "reality") || strings.Contains(nameLower, "tuic") {
				airportName = "Hhost-JP"
			} else if strings.Contains(nameLower, "专线") || strings.Contains(nameLower, "家庭") || strings.Contains(nameLower, "dmit") {
				airportName = "Mtice"
			} else if strings.Contains(nameLower, "hk-") || strings.Contains(nameLower, "tw-") || strings.Contains(nameLower, "sg-") || strings.Contains(nameLower, "jp-") || strings.Contains(nameLower, "us-") || strings.Contains(nameLower, "kr-") {
				airportName = "Jscloud"
			} else if strings.Contains(p.Name, "MPLS") {
				airportName = "Niubi"
			} else if strings.Contains(p.Name, "x15") || strings.Contains(p.Name, "Forest") || strings.Contains(p.Name, "国际优化") {
				airportName = "Forest"
			} else {
				airportName = "Kuailei"
			}
		}

		prefix := fmt.Sprintf("[%s]", airportName)
		if !strings.HasPrefix(p.Name, "[") {
			oldName := p.Name
			newName := fmt.Sprintf("%s %s", prefix, oldName)
			p.Name = newName
			renameMap[oldName] = newName
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

	// 0. Ensure all nodes have airport prefixes
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
			if reg.Regex != nil && reg.Regex.MatchString(name) {
				regionNodes[reg.ID] = append(regionNodes[reg.ID], name)
				matched = true
				break
			}
		}
		if !matched {
			// If not matched to HK/TW/SG_JPKR/US:
			// If it has airport prefix or sub_id, put into OTHERS;
			// If standalone without airport signature, put into CUSTOM
			if p.SubID == "" && !strings.Contains(name, "Hhost") && !strings.Contains(name, "Mtice") && !strings.Contains(name, "Jscloud") && !strings.Contains(name, "Niubi") && !strings.Contains(name, "Forest") && !strings.Contains(name, "Kuailei") {
				regionNodes["CUSTOM"] = append(regionNodes["CUSTOM"], name)
			} else {
				regionNodes["OTHERS"] = append(regionNodes["OTHERS"], name)
			}
		}
	}

	// Existing groups map for preserving selections and custom additions
	existingMap := make(map[string]StrategyGroup)
	for _, g := range ir.Groups {
		existingMap[g.Name] = g
	}

	newGroups := make([]StrategyGroup, 0)

	// 3. Ensure all 18 Regional Strategy Groups exist
	for _, reg := range StandardRegions {
		nodes := regionNodes[reg.ID]
		if len(nodes) == 0 {
			nodes = []string{"DIRECT"}
		}

		// (0) 手动 (select)
		manualGrp := StrategyGroup{
			Name:     reg.ManualName,
			Type:     "select",
			Proxies:  nodes,
			Now:      nodes[0],
			Category: "region",
			Region:   reg.ID,
			Policy:   "manual",
			Icon:     reg.Flag,
		}
		if eg, exists := existingMap[reg.ManualName]; exists && eg.Now != "" {
			manualGrp.Now = eg.Now
		}
		newGroups = append(newGroups, manualGrp)

		// (1) 自动 (url-test)
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
		if eg, exists := existingMap[reg.AutoName]; exists && eg.Now != "" {
			autoGrp.Now = eg.Now
		}
		newGroups = append(newGroups, autoGrp)

		// (2) 故转 (fallback)
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
		if eg, exists := existingMap[reg.FallName]; exists && eg.Now != "" {
			fallGrp.Now = eg.Now
		}
		newGroups = append(newGroups, fallGrp)

		// (3) 容灾 (loadbalance / select)
		disaGrp := StrategyGroup{
			Name:      reg.DisaName,
			Type:      "loadbalance",
			Proxies:   nodes,
			Now:       nodes[0],
			URL:       "http://www.gstatic.com/generate_204",
			Interval:  300,
			Category:  "region",
			Region:    reg.ID,
			Policy:    "disaster",
			Icon:      reg.Flag,
		}
		if eg, exists := existingMap[reg.DisaName]; exists && eg.Now != "" {
			disaGrp.Now = eg.Now
		}
		newGroups = append(newGroups, disaGrp)
	}

	// 4. Ensure Business Routing Groups exist
	allRegionalCandidateProxies := make([]string, 0)
	for _, reg := range StandardRegions {
		allRegionalCandidateProxies = append(allRegionalCandidateProxies, reg.ManualName, reg.AutoName, reg.FallName, reg.DisaName)
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

		bizGrp := StrategyGroup{
			Name:     bg.Name,
			Type:     "select",
			Proxies:  candidates,
			Now:      candidates[0],
			Category: "business",
			Icon:     bg.Icon,
			IsCustom: false,
		}

		if eg, exists := existingMap[bg.Name]; exists {
			if eg.Now != "" {
				bizGrp.Now = eg.Now
			}
			if len(eg.Proxies) > 0 {
				bizGrp.Proxies = mergeProxiesUnique(candidates, eg.Proxies)
			}
		} else if bg.Name == "AI" {
			for _, oldName := range []string{"OpenAI & AI", "OpenAI / Claude", "OpenAI", "ChatGPT"} {
				if eg, ok := existingMap[oldName]; ok {
					if eg.Now != "" {
						bizGrp.Now = eg.Now
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

// GetRegionSummary returns statistics for the 6 geographic regions.
type RegionStat struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Flag        string   `json:"flag"`
	TotalNodes  int      `json:"total_nodes"`
	ActiveNode  string   `json:"active_node"`
	LatencyMs   int      `json:"latency_ms"`
	ManualGroup string   `json:"manual_group"`
	AutoGroup   string   `json:"auto_group"`
	FallGroup   string   `json:"fall_group"`
	DisaGroup   string   `json:"disa_group"`
	NodeNames   []string `json:"node_names"`
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
			ID:          reg.ID,
			Name:        reg.Name,
			Flag:        reg.Flag,
			TotalNodes:  len(nodes),
			ActiveNode:  autoGrp.Now,
			LatencyMs:   32, // calculated or default
			ManualGroup: reg.ManualName,
			AutoGroup:   reg.AutoName,
			FallGroup:   reg.FallName,
			DisaGroup:   reg.DisaName,
			NodeNames:   nodes,
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
		if !seen[item] {
			seen[item] = true
			res = append(res, item)
		}
	}
	for _, item := range extra {
		if !seen[item] {
			seen[item] = true
			res = append(res, item)
		}
	}
	return res
}
