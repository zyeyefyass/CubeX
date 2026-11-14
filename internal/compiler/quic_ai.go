package compiler

import (
	"strings"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

// InjectQUICDegradationRule 在双擎路由表最前端插入海外 UDP 443 拒绝规则，触发浏览器 0ms 回退 TCP HTTP/2
func InjectQUICDegradationRule(
	disableQUIC bool,
	mihomoRules []string,
	singBoxRules []map[string]any,
) ([]string, []map[string]any) {
	if !disableQUIC {
		return mihomoRules, singBoxRules
	}

	// Mihomo: 拦截非直连的 UDP 443 流量并发送 ICMP 拒绝
	mihomoQUICRule := "AND,((NETWORK,UDP),(DST-PORT,443)),REJECT"
	mihomoOut := append([]string{mihomoQUICRule}, mihomoRules...)

	// sing-box v1.14+: 使用 method "default" 发送 ICMP Port Unreachable，严禁使用 drop 以免浏览器干等超时
	sbQUICRule := map[string]any{
		"network": []string{"udp"},
		"port":    []int{443},
		"action":  "reject",
		"method":  "default",
	}
	sbOut := append([]map[string]any{sbQUICRule}, singBoxRules...)

	return mihomoOut, sbOut
}

// SelectAICompliantNodes 提取美、日、新、台、韩、英等合规节点，并严格隔离国内/香港/澳门
func SelectAICompliantNodes(validNodes []string) []string {
	var out []string
	for _, n := range validNodes {
		lower := strings.ToLower(n)
		if strings.Contains(n, "香港") || strings.Contains(lower, "hk") ||
			strings.Contains(n, "澳门") || strings.Contains(lower, "macau") ||
			strings.Contains(n, "国内") || strings.Contains(lower, "china") || strings.Contains(n, "直连") {
			continue
		}
		if strings.Contains(n, "美") || strings.Contains(lower, "us") ||
			strings.Contains(n, "日") || strings.Contains(lower, "jp") ||
			strings.Contains(n, "新") || strings.Contains(lower, "sg") ||
			strings.Contains(n, "台") || strings.Contains(lower, "tw") ||
			strings.Contains(n, "韩") || strings.Contains(lower, "kr") ||
			strings.Contains(n, "英") || strings.Contains(lower, "uk") || strings.Contains(lower, "gb") {
			out = append(out, n)
		}
	}
	return out
}

// UpsertAICompliantGroup 自动提炼合规节点，幂等注入或更新「AI 专线」策略组
func UpsertAICompliantGroup(groups []schema.StrategyGroup, validNodes []string) []schema.StrategyGroup {
	aiNodes := SelectAICompliantNodes(validNodes)
	if len(aiNodes) == 0 {
		return groups
	}
	const aiGroupName = "AI 专线"

	for i := range groups {
		if groups[i].Name == aiGroupName || groups[i].Name == "OpenAI" || groups[i].Name == "AI" {
			groups[i].Proxies = aiNodes
			return groups
		}
	}

	aiGroup := schema.StrategyGroup{
		Name:    aiGroupName,
		Type:    "urltest",
		Proxies: aiNodes,
	}
	return append([]schema.StrategyGroup{aiGroup}, groups...)
}

// MatchAIGroupName 探查目标策略组集合中存在的 AI 出站组名称，支持 "AI 专线"、"AI"、"OpenAI"
func MatchAIGroupName(exists func(string) bool) string {
	for _, cand := range []string{"AI 专线", "AI", "OpenAI"} {
		if exists(cand) {
			return cand
		}
	}
	return ""
}

