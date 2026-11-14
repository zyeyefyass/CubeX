package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/zyeyefyass/CubeX/internal/adblock"
	"github.com/zyeyefyass/CubeX/internal/compiler"
	"github.com/zyeyefyass/CubeX/internal/dnsrewrite"
	"github.com/zyeyefyass/CubeX/internal/schema"
	"github.com/zyeyefyass/CubeX/internal/sub"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

func resolveProxyFromIR(ir *schema.ConfigIR, name string, depth int) *schema.ProxyNode {

	if ir == nil || name == "" || depth > 8 {

		return nil

	}

	trimmed := strings.TrimSpace(name)

	for i := range ir.Proxies {

		if ir.Proxies[i].Name == trimmed && ir.Proxies[i].Type != "direct" && ir.Proxies[i].Type != "block" && ir.Proxies[i].Type != "reject" {

			return &ir.Proxies[i]

		}

	}

	for _, g := range ir.Groups {

		if g.Name == trimmed || strings.EqualFold(g.Name, trimmed) {

			nxt := g.Now

			if nxt == "" && len(g.Proxies) > 0 {

				nxt = g.Proxies[0]

			}

			if nxt != "" && nxt != trimmed {

				if found := resolveProxyFromIR(ir, nxt, depth+1); found != nil {

					return found

				}

			}

		}

	}

	for i := range ir.Proxies {

		if strings.Contains(strings.ToLower(ir.Proxies[i].Name), strings.ToLower(trimmed)) && ir.Proxies[i].Type != "direct" {

			return &ir.Proxies[i]

		}

	}

	return nil

}





func (s *Server) handleRulesDeltaStatus(w http.ResponseWriter, r *http.Request) {
	if s.deltaRulesMgr == nil {
		sendJSON(w, map[string]any{"status": "error", "message": "delta manager not initialized"})
		return
	}
	sendJSON(w, map[string]any{
		"status": "ok",
		"delta":  s.deltaRulesMgr.GetStatus(),
	})
}

// --- 维度三：物理双 WAN 动态感知与对等运营商分流 ---





func (s *Server) handleDNSPrefetchStatus(w http.ResponseWriter, r *http.Request) {
	st := s.dnsPrefetchMgr.GetStatus()
	sendJSON(w, map[string]any{"status": "ok", "prefetch": st})
}





func (s *Server) handleDNSPrefetchToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	st := s.dnsPrefetchMgr.Toggle(req.Enabled)
	action := "启用"
	if !req.Enabled {
		action = "暂停"
	}
	s.logBuffer.Add("INFO", "DNS", fmt.Sprintf("DNS 常访域名主动预取与负缓存免疫巡航已%s", action))
	sendJSON(w, map[string]any{"status": "ok", "prefetch": st})
}





func (s *Server) handleDNSPrefetchFlush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st := s.dnsPrefetchMgr.Flush()
	s.logBuffer.Add("INFO", "DNS", "已刷新常访域名内存热池并重置负缓存免疫计数")
	sendJSON(w, map[string]any{"status": "ok", "prefetch": st})
}

// ── 宿主底座策略路由影子巡检与防篡改自愈器 API ──





func (s *Server) handleSubAdd(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		Name string `json:"name"`

		URL string `json:"url"`

		Type string `json:"type"`

		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)

		return

	}

	if req.URL == "" && req.Content == "" {

		http.Error(w, "url or content required", http.StatusBadRequest)

		return

	}

	var res *sub.FetchResult

	if req.Content != "" {

		nodes, err := sub.UniversalSmartParse(req.Content)

		if err != nil {

			sendJSON(w, map[string]any{"status": "error", "error": err.Error()})

			return

		}

		res = &sub.FetchResult{

			Nodes: nodes,

			NodeCount: len(nodes),

			DurationMs: 1,
		}

	} else if strings.HasPrefix(req.URL, "http://") || strings.HasPrefix(req.URL, "https://") {

		fetcher := sub.NewFetcher()

		var err error

		res, err = fetcher.Fetch(req.URL)

		if err != nil {

			sendJSON(w, map[string]any{"status": "error", "error": err.Error()})

			return

		}

	} else {

		filePath := strings.TrimPrefix(req.URL, "file://")

		data, err := os.ReadFile(filePath)

		if err != nil {

			sendJSON(w, map[string]any{"status": "error", "error": err.Error()})

			return

		}

		nodes, err := sub.ParseSubscription(data)

		if err != nil {

			sendJSON(w, map[string]any{"status": "error", "error": err.Error()})

			return

		}

		res = &sub.FetchResult{

			Nodes: nodes,

			NodeCount: len(nodes),

			DurationMs: 1,
		}

	}

	ir := s.engMgr.GetIR()

	subID := fmt.Sprintf("sub_%d", time.Now().Unix())

	if req.Name == "" {

		req.Name = fmt.Sprintf("订阅 %d", len(ir.Subscriptions)+1)

	}

	newSub := schema.Subscription{

		ID: subID,

		Name: req.Name,

		URL: req.URL,

		Type: req.Type,

		Enabled: true,

		NodeCount: res.NodeCount,

		UsedBytes: res.Upload + res.Download,

		TotalBytes: res.Total,

		ExpireAt: res.Expire,

		UpdatedAt: time.Now(),
	}

	ir.Subscriptions = append(ir.Subscriptions, newSub)

	// Merge new nodes into Proxies with airport prefix

	nodeNames := make([]string, 0, len(res.Nodes))

	for _, n := range res.Nodes {

		n.SubID = subID

		prefix := fmt.Sprintf("[%s]", req.Name)

		if !strings.HasPrefix(n.Name, prefix) {

			n.Name = fmt.Sprintf("%s %s", prefix, n.Name)

		}

		ir.Proxies = append(ir.Proxies, n)

		nodeNames = append(nodeNames, n.Name)

	}

	schema.SyncAllStrategyGroups(ir)

	s.engMgr.UpdateIR(ir)

	// Re-trigger compilation for active engine

	_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())

	sendJSON(w, map[string]any{

		"ok": true,

		"status": "added",

		"sub_id": subID,

		"node_count": res.NodeCount,

		"duration_ms": res.DurationMs,
	})

}





func (s *Server) handleSubBatchAdd(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		Items []struct {
			Name string `json:"name"`

			URL string `json:"url"`
		} `json:"items"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})

		return

	}

	if len(req.Items) == 0 {

		sendJSON(w, map[string]any{"ok": false, "error": "请至少提供一条有效的网络订阅"})

		return

	}

	ir := s.engMgr.GetIR()

	fetcher := sub.NewFetcher()

	addedCount := 0

	totalNodes := 0

	for i, item := range req.Items {

		url := strings.TrimSpace(item.URL)

		if url == "" {

			continue

		}

		name := strings.TrimSpace(item.Name)

		if name == "" {

			name = fmt.Sprintf("网络订阅 %d", len(ir.Subscriptions)+1)

		}

		subID := fmt.Sprintf("sub_%d_%d", time.Now().Unix(), i)

		res, err := fetcher.Fetch(url)

		if err != nil {

			res = &sub.FetchResult{

				Nodes: nil,

				NodeCount: 0,
			}

		}

		newSub := schema.Subscription{

			ID: subID,

			Name: name,

			URL: url,

			Enabled: true,

			NodeCount: res.NodeCount,

			UsedBytes: res.Upload + res.Download,

			TotalBytes: res.Total,

			ExpireAt: res.Expire,

			UpdatedAt: time.Now(),
		}

		ir.Subscriptions = append(ir.Subscriptions, newSub)

		for _, n := range res.Nodes {

			n.SubID = subID

			prefix := fmt.Sprintf("[%s]", name)

			if !strings.HasPrefix(n.Name, prefix) {

				n.Name = fmt.Sprintf("%s %s", prefix, n.Name)

			}

			ir.Proxies = append(ir.Proxies, n)

			totalNodes++

		}

		addedCount++

	}

	if addedCount > 0 {

		schema.SyncAllStrategyGroups(ir)

		s.engMgr.UpdateIR(ir)

		_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())

	}

	sendJSON(w, map[string]any{

		"ok": true,

		"added_subs": addedCount,

		"total_nodes": totalNodes,
	})

}





func (s *Server) handleSubList(w http.ResponseWriter, r *http.Request) {

	ir := s.engMgr.GetIR()

	sendJSON(w, map[string]any{

		"ok": true,

		"subscriptions": ir.Subscriptions,

		"proxies": ir.Proxies,
	})

}





func (s *Server) handleSubDelete(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		ID string `json:"id"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.ID == "" {

		http.Error(w, "id required", http.StatusBadRequest)

		return

	}

	ir := s.engMgr.GetIR()

	newSubs := make([]schema.Subscription, 0)

	for _, sub := range ir.Subscriptions {

		if sub.ID != req.ID {

			newSubs = append(newSubs, sub)

		}

	}

	ir.Subscriptions = newSubs

	// Remove proxies belonging to this sub

	newProxies := make([]schema.ProxyNode, 0)

	removedNames := make(map[string]bool)

	for _, p := range ir.Proxies {

		if p.SubID == req.ID {

			removedNames[p.Name] = true

		} else {

			newProxies = append(newProxies, p)

		}

	}

	ir.Proxies = newProxies

	// Clean up groups

	for i := range ir.Groups {

		newGrpProxies := make([]string, 0)

		for _, name := range ir.Groups[i].Proxies {

			if !removedNames[name] {

				newGrpProxies = append(newGrpProxies, name)

			}

		}

		ir.Groups[i].Proxies = newGrpProxies

		if removedNames[ir.Groups[i].Now] && len(newGrpProxies) > 0 {

			ir.Groups[i].Now = newGrpProxies[0]

		}

	}

	s.engMgr.UpdateIR(ir)

	_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())

	sendJSON(w, map[string]any{"ok": true, "deleted_id": req.ID})

}





func (s *Server) handleSubToggle(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		ID string `json:"id"`

		Enabled bool `json:"enabled"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)

	ir := s.engMgr.GetIR()

	for i := range ir.Subscriptions {

		if ir.Subscriptions[i].ID == req.ID {

			ir.Subscriptions[i].Enabled = req.Enabled

			break

		}

	}

	schema.SyncAllStrategyGroups(ir)

	s.engMgr.UpdateIR(ir)

	_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())

	sendJSON(w, map[string]any{"ok": true})

}





func (s *Server) handleSubRename(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		ID string `json:"id"`

		Name string `json:"name"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)

	req.Name = strings.TrimSpace(req.Name)

	if req.ID == "" || req.Name == "" {

		http.Error(w, "id and name required", http.StatusBadRequest)

		return

	}

	ir := s.engMgr.GetIR()

	var oldName string

	for i := range ir.Subscriptions {

		if ir.Subscriptions[i].ID == req.ID {

			oldName = ir.Subscriptions[i].Name

			ir.Subscriptions[i].Name = req.Name

			break

		}

	}

	if oldName != "" && oldName != req.Name {

		oldPrefix := fmt.Sprintf("[%s]", oldName)

		newPrefix := fmt.Sprintf("[%s]", req.Name)

		for j := range ir.Proxies {

			if ir.Proxies[j].SubID == req.ID {

				if strings.HasPrefix(ir.Proxies[j].Name, oldPrefix) {

					ir.Proxies[j].Name = newPrefix + strings.TrimPrefix(ir.Proxies[j].Name, oldPrefix)

				}

			}

		}

		schema.SyncAllStrategyGroups(ir)

		s.engMgr.UpdateIR(ir)

		_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())

	} else {

		s.engMgr.UpdateIR(ir)

	}

	sendJSON(w, map[string]any{"ok": true, "name": req.Name})

}





func (s *Server) handleNodeAdd(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		Name string `json:"name"`

		Link string `json:"link"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)

		return

	}

	req.Link = strings.TrimSpace(req.Link)

	if req.Link == "" {

		sendJSON(w, map[string]any{"ok": false, "error": "节点链接不能为空"})

		return

	}

	var node schema.ProxyNode

	var err error

	if strings.Contains(req.Link, "://") {

		node, err = sub.ParseURILink(req.Link)

	} else {

		nodes, pErr := sub.UniversalSmartParse(req.Link)

		if pErr == nil && len(nodes) > 0 {

			node = nodes[0]

		} else {

			err = fmt.Errorf("无法解析节点链接")

		}

	}

	if err != nil || node.Server == "" {

		sendJSON(w, map[string]any{"ok": false, "error": "节点格式无法识别，请检查 ss://, vmess://, vless://, trojan://, hysteria2:// 链接"})

		return

	}

	if req.Name != "" {

		node.Name = req.Name

	}

	if node.Name == "" {

		node.Name = fmt.Sprintf("节点-%s:%d", node.Server, node.Port)

	}

	if !strings.HasPrefix(node.Name, "[自建]") {

		node.Name = fmt.Sprintf("[自建] %s", node.Name)

	}

	node.SubID = "custom"

	ir := s.engMgr.GetIR()

	// Ensure unique node name

	exists := false

	for _, p := range ir.Proxies {

		if p.Name == node.Name {

			exists = true

			break

		}

	}

	if exists {

		node.Name = fmt.Sprintf("%s (%d)", node.Name, time.Now().Unix()%1000)

	}

	ir.Proxies = append(ir.Proxies, node)

	schema.SyncAllStrategyGroups(ir)

	s.engMgr.UpdateIR(ir)

	_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())

	sendJSON(w, map[string]any{

		"ok": true,

		"node": node,
	})

}





func (s *Server) handleNodeBatchAdd(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		Items []struct {
			Name string `json:"name"`

			Link string `json:"link"`
		} `json:"items"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})

		return

	}

	if len(req.Items) == 0 {

		sendJSON(w, map[string]any{"ok": false, "error": "请至少提供一个有效的节点链接"})

		return

	}

	ir := s.engMgr.GetIR()

	addedCount := 0

	for i, item := range req.Items {

		link := strings.TrimSpace(item.Link)

		if link == "" {

			continue

		}

		name := strings.TrimSpace(item.Name)

		var node schema.ProxyNode

		var err error

		if strings.Contains(link, "://") {

			node, err = sub.ParseURILink(link)

		} else {

			nodes, pErr := sub.UniversalSmartParse(link)

			if pErr == nil && len(nodes) > 0 {

				node = nodes[0]

			} else {

				err = fmt.Errorf("无法解析节点")

			}

		}

		if err != nil || node.Server == "" {

			continue

		}

		if name != "" {

			node.Name = name

		}

		if node.Name == "" {

			node.Name = fmt.Sprintf("节点-%s:%d", node.Server, node.Port)

		}

		if !strings.HasPrefix(node.Name, "[自建]") {

			node.Name = fmt.Sprintf("[自建] %s", node.Name)

		}

		node.SubID = "custom"

		for _, p := range ir.Proxies {

			if p.Name == node.Name {

				node.Name = fmt.Sprintf("%s (%d)", node.Name, (time.Now().Unix()%1000)+int64(i))

				break

			}

		}

		ir.Proxies = append(ir.Proxies, node)

		addedCount++

	}

	if addedCount > 0 {

		schema.SyncAllStrategyGroups(ir)

		s.engMgr.UpdateIR(ir)

		_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())

	}

	sendJSON(w, map[string]any{

		"ok": true,

		"added_nodes": addedCount,
	})

}





func (s *Server) handleNodeDelete(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		Name string `json:"name"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.Name == "" {

		http.Error(w, "name required", http.StatusBadRequest)

		return

	}

	ir := s.engMgr.GetIR()

	newProxies := make([]schema.ProxyNode, 0)

	for _, p := range ir.Proxies {

		if p.Name != req.Name {

			newProxies = append(newProxies, p)

		}

	}

	ir.Proxies = newProxies

	for i := range ir.Groups {

		newGrpProxies := make([]string, 0)

		for _, name := range ir.Groups[i].Proxies {

			if name != req.Name {

				newGrpProxies = append(newGrpProxies, name)

			}

		}

		ir.Groups[i].Proxies = newGrpProxies

		if ir.Groups[i].Now == req.Name && len(newGrpProxies) > 0 {

			ir.Groups[i].Now = newGrpProxies[0]

		}

	}

	s.engMgr.UpdateIR(ir)

	sendJSON(w, map[string]any{"ok": true, "deleted_name": req.Name})

}





func (s *Server) handleGroupSelect(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		Group string `json:"group"`

		Node string `json:"node"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)

		return

	}

	if req.Group == "" || req.Node == "" {

		http.Error(w, "group and node required", http.StatusBadRequest)

		return

	}

	ir := s.engMgr.GetIR()

	updated := false

	for i := range ir.Groups {

		if ir.Groups[i].Name == req.Group {

			ir.Groups[i].Now = req.Node

			updated = true

			break

		}

	}

	if updated {

		s.engMgr.UpdateIR(ir)

		// If Clash API is available, notify Mihomo / sing-box and purge stale connections
		SafeGo("GroupSelectPurge", func() {

			payload, _ := json.Marshal(map[string]string{"name": req.Node})

			escapedGroup := url.PathEscape(req.Group)

			reqURL := fmt.Sprintf("http://127.0.0.1:%d/proxies/%s", ir.Network.ClashAPIPort, escapedGroup)

			client := http.Client{Timeout: 1 * time.Second}

			cReq, err := http.NewRequest(http.MethodPut, reqURL, bytes.NewBuffer(payload))

			if err == nil {

				cReq.Header.Set("Content-Type", "application/json")

				if resp, err := client.Do(cReq); err == nil && resp != nil {

					_ = resp.Body.Close()

				}

			}

			// Terminate active HTTP/2 and keep-alive sockets so new traffic immediately routes through the new node

			delReq, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://127.0.0.1:%d/connections", ir.Network.ClashAPIPort), nil)

			if err == nil {

				if resp, err := client.Do(delReq); err == nil && resp != nil {

					_ = resp.Body.Close()

				}

			}

		})

	}

	sendJSON(w, map[string]any{

		"ok": true,

		"group": req.Group,

		"now": req.Node,
	})

}





func (s *Server) handleRoutingRegions(w http.ResponseWriter, r *http.Request) {

	ir := s.engMgr.GetIR()

	stats := schema.ExtractRegionStats(ir)

	sendJSON(w, map[string]any{

		"ok": true,

		"regions": stats,
	})

}





func (s *Server) handleCustomGroupAdd(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		Name string `json:"name"`

		Icon string `json:"icon"`

		Type string `json:"type"`

		Proxies []string `json:"proxies"`

		Now string `json:"now"`

		RuleType string `json:"rule_type"`

		RuleValue string `json:"rule_value"`

		FilterKeywords []string `json:"filter_keywords"`

		FilterLogic string `json:"filter_logic"`

		FilterRegex string `json:"filter_regex"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)

		return

	}

	req.Name = strings.TrimSpace(req.Name)

	if req.Name == "" {

		sendJSON(w, map[string]any{"ok": false, "error": "业务分流名称不能为空"})

		return

	}

	ir := s.engMgr.GetIR()

	for _, g := range ir.Groups {

		if g.Name == req.Name {

			sendJSON(w, map[string]any{"ok": false, "error": "该策略组名称已存在"})

			return

		}

	}

	if req.Type == "" {

		req.Type = "select"

	}

	if req.Icon == "" {

		req.Icon = "default"

	}

	if len(req.FilterKeywords) > 0 || req.FilterRegex != "" {

		matched := schema.FilterNodesByKeywords(ir.Proxies, req.FilterKeywords, req.FilterLogic, req.FilterRegex)

		if len(matched) > 0 {

			req.Proxies = matched

		} else {

			req.Proxies = []string{"DIRECT"}

		}

	}

	if len(req.Proxies) == 0 {

		req.Proxies = []string{"🇭🇰 香港自动", "🇹🇼 台湾自动", "🇸🇬🇯🇵🇰🇷 新日韩自动", "🇺🇸 美国自动", "🏠 自建自动", "🌐 其它自动", "DIRECT"}

	}

	if req.Now == "" || req.Now == "🇸🇬 新加坡自动" || req.Now == "🇯🇵 日韩自动" {

		req.Now = req.Proxies[0]

	}

	newGroup := schema.StrategyGroup{

		Name: req.Name,

		Type: req.Type,

		Proxies: req.Proxies,

		Now: req.Now,

		Icon: req.Icon,

		Category: "business",

		IsCustom: true,

		FilterKeywords: req.FilterKeywords,

		FilterLogic: req.FilterLogic,

		FilterRegex: req.FilterRegex,
	}

	ir.Groups = append(ir.Groups, newGroup)

	if req.RuleValue != "" {

		rType := req.RuleType

		if rType == "" {

			rType = "geosite"

		}

		ir.Rules = append([]schema.Rule{

			{

				Type: rType,

				Value: req.RuleValue,

				Target: req.Name,
			},
		}, ir.Rules...)

	}

	s.engMgr.UpdateIR(ir)

	s.ir = ir

	_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())

	sendJSON(w, map[string]any{"ok": true, "group": newGroup})

}





func (s *Server) handleCustomGroupUpdate(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		Name string `json:"name"`

		NewName string `json:"new_name"`

		Icon string `json:"icon"`

		Type string `json:"type"`

		Proxies []string `json:"proxies"`

		Now string `json:"now"`

		FilterKeywords []string `json:"filter_keywords"`

		FilterLogic string `json:"filter_logic"`

		FilterRegex string `json:"filter_regex"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)

		return

	}

	req.Name = strings.TrimSpace(req.Name)

	if req.Name == "" {

		sendJSON(w, map[string]any{"ok": false, "error": "策略组名称不能为空"})

		return

	}

	ir := s.engMgr.GetIR()

	found := false

	for i := range ir.Groups {

		if ir.Groups[i].Name == req.Name {

			found = true

			if req.NewName != "" && req.NewName != req.Name {

				ir.Groups[i].Name = strings.TrimSpace(req.NewName)

				for j := range ir.Rules {

					if ir.Rules[j].Target == req.Name {

						ir.Rules[j].Target = ir.Groups[i].Name

					}

				}

			}

			if req.Icon != "" {

				ir.Groups[i].Icon = req.Icon

			}

			if req.Type != "" {

				ir.Groups[i].Type = req.Type

			}

			ir.Groups[i].FilterKeywords = req.FilterKeywords

			ir.Groups[i].FilterLogic = req.FilterLogic

			ir.Groups[i].FilterRegex = req.FilterRegex

			if len(req.FilterKeywords) > 0 || req.FilterRegex != "" {

				matched := schema.FilterNodesByKeywords(ir.Proxies, req.FilterKeywords, req.FilterLogic, req.FilterRegex)

				if len(matched) > 0 {

					ir.Groups[i].Proxies = matched

				} else {

					ir.Groups[i].Proxies = []string{"DIRECT"}

				}

			} else if len(req.Proxies) > 0 {

				ir.Groups[i].Proxies = req.Proxies

			}

			if req.Now != "" {

				ir.Groups[i].Now = req.Now

			} else if len(ir.Groups[i].Proxies) > 0 {

				ir.Groups[i].Now = ir.Groups[i].Proxies[0]

			}

			break

		}

	}

	if !found {

		sendJSON(w, map[string]any{"ok": false, "error": "未找到指定策略组"})

		return

	}

	schema.SyncAllStrategyGroups(ir)

	s.engMgr.UpdateIR(ir)

	s.ir = ir

	_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())

	sendJSON(w, map[string]any{"ok": true})

}





func (s *Server) handleSubEdit(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		ID string `json:"id"`

		Name string `json:"name"`

		URL string `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})

		return

	}

	req.ID = strings.TrimSpace(req.ID)

	req.URL = strings.TrimSpace(req.URL)

	req.Name = strings.TrimSpace(req.Name)

	if req.ID == "" {

		sendJSON(w, map[string]any{"ok": false, "error": "订阅 ID 不能为空"})

		return

	}

	ir := s.engMgr.GetIR()

	subIdx := -1

	for i := range ir.Subscriptions {

		if ir.Subscriptions[i].ID == req.ID {

			subIdx = i

			break

		}

	}

	if subIdx == -1 {

		sendJSON(w, map[string]any{"ok": false, "error": "未找到指定订阅"})

		return

	}

	if req.URL == "" {

		req.URL = ir.Subscriptions[subIdx].URL

	}

	// 1. 若 URL 未发生变化（仅修改名称），秒级重命名并更新节点前缀，无需发起耗时网络重拉

	if req.URL == ir.Subscriptions[subIdx].URL {

		if req.Name != "" && req.Name != ir.Subscriptions[subIdx].Name {

			oldName := ir.Subscriptions[subIdx].Name

			ir.Subscriptions[subIdx].Name = req.Name

			oldPrefix := fmt.Sprintf("[%s]", oldName)

			newPrefix := fmt.Sprintf("[%s]", req.Name)

			for j := range ir.Proxies {

				if ir.Proxies[j].SubID == req.ID {

					if strings.HasPrefix(ir.Proxies[j].Name, oldPrefix) {

						ir.Proxies[j].Name = newPrefix + strings.TrimPrefix(ir.Proxies[j].Name, oldPrefix)

					}

				}

			}

			schema.SyncAllStrategyGroups(ir)

			s.engMgr.UpdateIR(ir)

			s.ir = ir

			_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())

		}

		sendJSON(w, map[string]any{

			"ok": true,

			"node_count": ir.Subscriptions[subIdx].NodeCount,

			"name": ir.Subscriptions[subIdx].Name,

			"url": ir.Subscriptions[subIdx].URL,
		})

		return

	}

	// 2. 若 URL 发生变化，则向新地址拉取节点并校验解析

	fetcher := sub.NewFetcher()

	res, err := fetcher.Fetch(req.URL)

	if err != nil || res == nil || res.NodeCount == 0 {

		errMsg := "新订阅链接验证失败（未能获取到有效节点），已自动保留现有节点与配置！"

		if err != nil {

			errMsg = fmt.Sprintf("新订阅解析失败: %v，已安全保留原配置！", err)

		}

		sendJSON(w, map[string]any{"ok": false, "error": errMsg})

		return

	}

	// Fetch succeeded: atomically replace nodes for this subscription

	if req.Name != "" {

		ir.Subscriptions[subIdx].Name = req.Name

	}

	ir.Subscriptions[subIdx].URL = req.URL

	ir.Subscriptions[subIdx].NodeCount = res.NodeCount

	ir.Subscriptions[subIdx].UsedBytes = res.Upload + res.Download

	ir.Subscriptions[subIdx].TotalBytes = res.Total

	ir.Subscriptions[subIdx].ExpireAt = res.Expire

	ir.Subscriptions[subIdx].UpdatedAt = time.Now()

	newProxies := make([]schema.ProxyNode, 0, len(ir.Proxies))

	for _, p := range ir.Proxies {

		if p.SubID != req.ID {

			newProxies = append(newProxies, p)

		}

	}

	for _, n := range res.Nodes {

		n.SubID = req.ID

		prefix := fmt.Sprintf("[%s]", ir.Subscriptions[subIdx].Name)

		if !strings.HasPrefix(n.Name, prefix) {

			n.Name = fmt.Sprintf("%s %s", prefix, n.Name)

		}

		newProxies = append(newProxies, n)

	}

	ir.Proxies = newProxies

	schema.SyncAllStrategyGroups(ir)

	s.engMgr.UpdateIR(ir)

	s.ir = ir

	_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())

	sendJSON(w, map[string]any{

		"ok": true,

		"node_count": res.NodeCount,

		"name": ir.Subscriptions[subIdx].Name,

		"url": ir.Subscriptions[subIdx].URL,
	})

}





func (s *Server) handleCustomGroupDelete(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)

		return

	}

	ir := s.engMgr.GetIR()

	newGroups := make([]schema.StrategyGroup, 0)

	for _, g := range ir.Groups {

		if g.Name != req.Name {

			newGroups = append(newGroups, g)

		}

	}

	ir.Groups = newGroups

	newRules := make([]schema.Rule, 0)

	for _, r := range ir.Rules {

		if r.Target != req.Name {

			newRules = append(newRules, r)

		}

	}

	ir.Rules = newRules

	s.engMgr.UpdateIR(ir)

	s.ir = ir

	_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())

	sendJSON(w, map[string]any{"ok": true, "deleted_group": req.Name})

}





func (s *Server) handleCustomGroupToggle(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)

		return

	}

	ir := s.engMgr.GetIR()

	found := false

	var newState bool

	for i := range ir.Groups {

		if ir.Groups[i].Name == req.Name {

			ir.Groups[i].Disabled = !ir.Groups[i].Disabled

			newState = ir.Groups[i].Disabled

			found = true

			break

		}

	}

	if !found {

		sendJSON(w, map[string]any{"ok": false, "error": "未找到指定策略组"})

		return

	}

	s.engMgr.UpdateIR(ir)

	s.ir = ir

	_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())

	sendJSON(w, map[string]any{"ok": true, "name": req.Name, "disabled": newState})

}





func (s *Server) handleSubUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID    string `json:"id"`
		Force bool   `json:"force"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.ID == "" {
		subsUpdated, totalNodes, err := s.UpdateAllSubscriptions()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sendJSON(w, map[string]any{
			"ok":           true,
			"status":       "updated",
			"subs_updated": subsUpdated,
			"total_nodes":  totalNodes,
		})
		return
	}

	ir := s.engMgr.GetIR()
	if ir == nil {
		http.Error(w, "IR not initialized", http.StatusInternalServerError)
		return
	}

	fetcher := sub.NewFetcher()
	totalNodes := 0
	var updatedProxies []schema.ProxyNode

	for _, p := range ir.Proxies {
		if p.Type == "direct" || p.Type == "block" || p.Type == "reject" || p.SubID == "custom" {
			updatedProxies = append(updatedProxies, p)
		} else if p.SubID != req.ID {
			updatedProxies = append(updatedProxies, p)
		}
	}

	var targetQuota sub.QuotaTelemetry
	subFound := false

	for i := range ir.Subscriptions {
		if ir.Subscriptions[i].ID != req.ID {
			continue
		}
		subFound = true
		if !ir.Subscriptions[i].Enabled || ir.Subscriptions[i].URL == "" {
			continue
		}

		res, err := fetcher.Fetch(ir.Subscriptions[i].URL)
		if err == nil {
			ir.Subscriptions[i].NodeCount = res.NodeCount
			ir.Subscriptions[i].UsedBytes = res.Upload + res.Download
			ir.Subscriptions[i].TotalBytes = res.Total
			ir.Subscriptions[i].ExpireAt = res.Expire
			ir.Subscriptions[i].UpdatedAt = time.Now()

			targetQuota = sub.QuotaTelemetry{
				Upload:   res.Upload,
				Download: res.Download,
				Total:    res.Total,
				Expire:   res.Expire,
			}

			for _, n := range res.Nodes {
				n.SubID = ir.Subscriptions[i].ID
				prefix := fmt.Sprintf("[%s]", ir.Subscriptions[i].Name)
				if !strings.HasPrefix(n.Name, prefix) {
					n.Name = fmt.Sprintf("%s %s", prefix, n.Name)
				}
				updatedProxies = append(updatedProxies, n)
			}
			totalNodes += res.NodeCount
		} else {
			log.Printf("⚠️ [Sub] Failed to fetch sub '%s' (%s): %v. Preserving existing nodes.", ir.Subscriptions[i].Name, ir.Subscriptions[i].URL, err)
			for _, p := range ir.Proxies {
				if p.SubID == ir.Subscriptions[i].ID {
					updatedProxies = append(updatedProxies, p)
					totalNodes++
				}
			}
		}
		break
	}

	if !subFound {
		http.Error(w, "subscription not found", http.StatusNotFound)
		return
	}

	outcomeStatus := "updated"
	msg := fmt.Sprintf("订阅更新完成，共加载 %d 个节点", totalNodes)

	if s.autoSyncGuard != nil {
		report, guardErr := s.autoSyncGuard.EvaluateAndSync(
			r.Context(),
			updatedProxies,
			targetQuota,
			req.Force,
			func(ctx context.Context, validNodes []schema.ProxyNode) error {
				ir.Proxies = validNodes
				schema.SyncAllStrategyGroups(ir)
				s.engMgr.UpdateIR(ir)
				_, err := s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
				return err
			},
		)
		if guardErr != nil {
			http.Error(w, guardErr.Error(), http.StatusInternalServerError)
			return
		}
		if report != nil {
			msg = report.Message
			switch report.Outcome {
			case sub.OutcomeCircuitBroken:
				outcomeStatus = "circuit_broken"
				s.logBuffer.Add("WARN", "SUB_GUARD", fmt.Sprintf("单订阅熔断保护: %s", report.Message))
			case sub.OutcomeSkippedUnchanged:
				outcomeStatus = "skipped_unchanged"
				s.logBuffer.Add("INFO", "SUB_GUARD", fmt.Sprintf("单订阅防抖静默跳过: %s", report.Message))
				s.engMgr.UpdateIR(ir)
			case sub.OutcomeAppliedAndReload:
				outcomeStatus = "updated"
				s.logBuffer.Add("INFO", "SUB_GUARD", fmt.Sprintf("单订阅更新已平滑生效: %s", report.Message))
			}
		}
	} else {
		ir.Proxies = updatedProxies
		schema.SyncAllStrategyGroups(ir)
		s.engMgr.UpdateIR(ir)
		_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
	}

	runtime.GC()
	debug.FreeOSMemory()

	sendJSON(w, map[string]any{
		"ok":          true,
		"status":      outcomeStatus,
		"message":     msg,
		"total_nodes": totalNodes,
	})
}

// UpdateAllSubscriptions fetches all active enabled subscriptions, updates node counts and metadata,
// re-syncs strategy groups, saves the IR to disk, and reloads the active proxy engine with AutoSyncGuard.




func (s *Server) UpdateAllSubscriptions() (int, int, error) {
	ir := s.engMgr.GetIR()
	if ir == nil {
		return 0, 0, fmt.Errorf("IR not initialized")
	}

	fetcher := sub.NewFetcher()
	totalNodes := 0
	subsUpdated := 0
	var updatedProxies []schema.ProxyNode

	// Preserve direct, reject, and custom manual nodes
	for _, p := range ir.Proxies {
		if p.Type == "direct" || p.Type == "block" || p.Type == "reject" || p.SubID == "custom" {
			updatedProxies = append(updatedProxies, p)
		}
	}

	var totalQuota sub.QuotaTelemetry

	for i := range ir.Subscriptions {
		if !ir.Subscriptions[i].Enabled || ir.Subscriptions[i].URL == "" {
			continue
		}

		res, err := fetcher.Fetch(ir.Subscriptions[i].URL)
		if err == nil {
			ir.Subscriptions[i].NodeCount = res.NodeCount
			ir.Subscriptions[i].UsedBytes = res.Upload + res.Download
			ir.Subscriptions[i].TotalBytes = res.Total
			ir.Subscriptions[i].ExpireAt = res.Expire
			ir.Subscriptions[i].UpdatedAt = time.Now()

			totalQuota.Upload += res.Upload
			totalQuota.Download += res.Download
			totalQuota.Total += res.Total
			if res.Expire > totalQuota.Expire {
				totalQuota.Expire = res.Expire
			}

			for _, n := range res.Nodes {
				n.SubID = ir.Subscriptions[i].ID
				prefix := fmt.Sprintf("[%s]", ir.Subscriptions[i].Name)
				if !strings.HasPrefix(n.Name, prefix) {
					n.Name = fmt.Sprintf("%s %s", prefix, n.Name)
				}
				updatedProxies = append(updatedProxies, n)
			}
			totalNodes += res.NodeCount
			subsUpdated++
		} else {
			log.Printf("⚠️ [Sub] Failed to fetch sub '%s' (%s): %v. Preserving existing nodes.", ir.Subscriptions[i].Name, ir.Subscriptions[i].URL, err)
			for _, p := range ir.Proxies {
				if p.SubID == ir.Subscriptions[i].ID {
					updatedProxies = append(updatedProxies, p)
					totalNodes++
				}
			}
		}
	}

	// 接入 AutoSyncGuard 防抖比对与熔断审查
	if s.autoSyncGuard != nil {
		report, guardErr := s.autoSyncGuard.EvaluateAndSync(
			context.Background(),
			updatedProxies,
			totalQuota,
			false,
			func(ctx context.Context, validNodes []schema.ProxyNode) error {
				ir.Proxies = validNodes
				schema.SyncAllStrategyGroups(ir)
				s.engMgr.UpdateIR(ir)
				_, err := s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
				return err
			},
		)
		if guardErr != nil {
			return subsUpdated, totalNodes, guardErr
		}
		if report != nil {
			switch report.Outcome {
			case sub.OutcomeCircuitBroken:
				s.logBuffer.Add("WARN", "SUB_GUARD", fmt.Sprintf("定时订阅触发熔断拦截: %s", report.Message))
				log.Printf("🛡️ [SubGuard] %s", report.Message)
				return subsUpdated, len(ir.Proxies), nil
			case sub.OutcomeSkippedUnchanged:
				s.logBuffer.Add("INFO", "SUB_GUARD", fmt.Sprintf("定时订阅无变更防抖静默跳过: %s", report.Message))
				log.Printf("🛡️ [SubGuard] %s", report.Message)
				s.engMgr.UpdateIR(ir)
				return subsUpdated, len(ir.Proxies), nil
			case sub.OutcomeAppliedAndReload:
				s.logBuffer.Add("INFO", "SUB_GUARD", fmt.Sprintf("定时订阅更新平滑生效: %s", report.Message))
				log.Printf("✅ [SubGuard] %s", report.Message)
			}
		}
	} else {
		ir.Proxies = updatedProxies
		schema.SyncAllStrategyGroups(ir)
		s.engMgr.UpdateIR(ir)
		_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
	}

	runtime.GC()
	debug.FreeOSMemory()
	return subsUpdated, totalNodes, nil
}

// startDailySubscriptionAutoUpdate runs a continuous background cron loop that triggers

// at exactly 00:00 (midnight local time) every day to keep all airport subscriptions fresh.





func (s *Server) startDailySubscriptionAutoUpdate() {
	SafeGo("DailySubCron", func() {

		for {

			now := time.Now()

			// Calculate next 12-hour sync time: either today 12:00:05 or tomorrow 00:00:05
			var nextSync time.Time
			todayNoon := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 5, 0, now.Location())
			tomorrowMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 5, 0, now.Location())

			if now.Before(todayNoon) {
				nextSync = todayNoon
			} else {
				nextSync = tomorrowMidnight
			}

			durationUntil := nextSync.Sub(now)

			log.Printf("⏰ [Sub-Cron] 12小时高可用订阅自动同步就绪 (将在 %v 后触发，下次: %s)",

				durationUntil.Round(time.Second), nextSync.Format("2006-01-02 15:04:05"))

			s.logBuffer.Add("INFO", "SUB", fmt.Sprintf("定时任务: 12小时双周期订阅自动同步就绪 (下次触发: %s)", nextSync.Format("2006-01-02 15:04:05")))

			select {

			case <-time.After(durationUntil):

				log.Println("⏰ [Sub-Cron] 订阅同步周期已到达：正在全自动拉取更新所有订阅...")

				subsCount, nodesCount, err := s.UpdateAllSubscriptions()

				if err != nil {

					log.Printf("⚠️ [Sub-Cron] 每日 00:00 自动同步异常: %v", err)

					s.logBuffer.Add("WARN", "SUB", fmt.Sprintf("每日 00:00 订阅自动同步异常: %v", err))

				} else {

					msg := fmt.Sprintf("每日 00:00 定时同步完成: 成功同步 %d 个订阅，装载 %d 个节点，引擎热重载完毕", subsCount, nodesCount)

					log.Println("✅ [Sub-Cron]", msg)

					s.logBuffer.Add("INFO", "SUB", msg)

				}

			}

		}

	})

}





func (s *Server) handleSubCronStatus(w http.ResponseWriter, r *http.Request) {

	now := time.Now()

	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 5, 0, now.Location())

	sendJSON(w, map[string]any{

		"enabled": true,

		"schedule": "daily_00_00",

		"description": "每天 00:00 自动拉取更新所有已启用的网络订阅",

		"next_run": nextMidnight.Format("2006-01-02 15:04:05"),

		"seconds_to_next": int(nextMidnight.Sub(now).Seconds()),
	})

}





type NodeDelayResult struct {
	Delay      int    `json:"delay"`
	Status     string `json:"status"`      // ok, timeout, dns_failed, conn_refused, resource_limit, tls_failed, conn_reset, error
	StatusText string `json:"status_text"` // 正常, 握手超时, DNS解析失败, 连接被拒, 系统资源受限, TLS握手异常, 连接重置, 连接异常
}





func classifyDelayError(errStr string, httpCode int) (string, string) {
	lower := strings.ToLower(errStr)
	if httpCode == http.StatusNotFound || strings.Contains(lower, "not found") || strings.Contains(lower, "not exist") {
		return "not_found", "未加载"
	}
	if strings.Contains(lower, "lookup") || strings.Contains(lower, "no such host") || strings.Contains(lower, "dns") || strings.Contains(lower, "name resolution") {
		return "dns_failed", "DNS解析失败"
	}
	if strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded") || httpCode == http.StatusGatewayTimeout {
		return "timeout", "握手超时"
	}
	if strings.Contains(lower, "connection refused") || strings.Contains(lower, "refused") {
		return "conn_refused", "连接被拒"
	}
	if strings.Contains(lower, "too many open files") || strings.Contains(lower, "resource temporarily unavailable") || strings.Contains(lower, "cannot assign requested address") || strings.Contains(lower, "socket: buffer") {
		return "resource_limit", "系统资源受限"
	}
	if strings.Contains(lower, "tls:") || strings.Contains(lower, "handshake") || strings.Contains(lower, "cert") || strings.Contains(lower, "bad record mac") || strings.Contains(lower, "first record does not look like a tls handshake") {
		return "tls_failed", "TLS握手异常"
	}
	if strings.Contains(lower, "reset by peer") || strings.Contains(lower, "broken pipe") || strings.Contains(lower, "eof") {
		return "conn_reset", "连接重置"
	}
	if lower != "" {
		return "error", "连接异常"
	}
	return "timeout", "握手超时"
}





func (s *Server) testNodeRealDelay(node schema.ProxyNode, clashPort int, clashSecret string) NodeDelayResult {
	if clashPort <= 0 || node.Name == "" {
		return NodeDelayResult{Delay: 0, Status: "error", StatusText: "节点未就绪"}
	}
	if node.Type == "direct" {
		return NodeDelayResult{Delay: 1, Status: "ok", StatusText: "直连"}
	}
	if node.Type == "block" || node.Type == "reject" {
		return NodeDelayResult{Delay: 0, Status: "block", StatusText: "阻断"}
	}

	// 1. 查询多策略组共享测速缓存 (20秒内有效)
	s.delayCacheMu.RLock()
	if c, ok := s.delayCache[node.Name]; ok && time.Since(c.testedAt) < 20*time.Second {
		s.delayCacheMu.RUnlock()
		return c.result
	}
	s.delayCacheMu.RUnlock()

	// 2. In-Flight Singleflight 并发去重 (同节点多策略组共享同一物理探针)
	s.delaySFMu.Lock()
	if call, ok := s.delaySFGroup[node.Name]; ok {
		s.delaySFMu.Unlock()
		call.wg.Wait()
		return call.val
	}
	call := new(delayFlightCall)
	call.wg.Add(1)
	s.delaySFGroup[node.Name] = call
	s.delaySFMu.Unlock()

	defer func() {
		s.delaySFMu.Lock()
		delete(s.delaySFGroup, node.Name)
		s.delaySFMu.Unlock()
		call.wg.Done()
	}()

	res := s.executeNodeRealDelayProbe(node, clashPort, clashSecret)

	// 写入 EWMA 滑动窗口防抖评分引擎
	if s.ewmaMgr != nil {
		s.ewmaMgr.RecordSample(node.Name, res.Delay, res.Delay > 0)
	}

	// 写入多策略组共享缓存
	s.delayCacheMu.Lock()
	s.delayCache[node.Name] = delayCacheEntry{result: res, testedAt: time.Now()}
	go s.savePersistentDelayCache()
	s.delayCacheMu.Unlock()

	call.val = res
	return res
}





func (s *Server) executeNodeRealDelayProbe(node schema.ProxyNode, clashPort int, clashSecret string) NodeDelayResult {
	escaped := url.PathEscape(node.Name)
	escaped = strings.ReplaceAll(escaped, "[", "%5B")
	escaped = strings.ReplaceAll(escaped, "]", "%5D")

	probeURLs := []string{
		"https://cp.cloudflare.com/generate_204",
		"https://www.gstatic.com/generate_204",
	}

	tr := &http.Transport{
		DisableKeepAlives: true,
		MaxIdleConns:      0,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   2800 * time.Millisecond,
	}

	lastStatus := "timeout"
	lastStatusText := "握手超时"

	for _, probe := range probeURLs {
		testURL := fmt.Sprintf("http://127.0.0.1:%d/proxies/%s/delay?url=%s&timeout=2500",
			clashPort, escaped, url.QueryEscape(probe))
		req, err := http.NewRequest(http.MethodGet, testURL, nil)
		if err != nil {
			lastStatus, lastStatusText = classifyDelayError(err.Error(), 0)
			continue
		}
		if clashSecret != "" {
			req.Header.Set("Authorization", "Bearer "+clashSecret)
		}
		resp, err := client.Do(req)
		if err != nil {
			lastStatus, lastStatusText = classifyDelayError(err.Error(), 0)
			continue
		}

		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		resp.Body.Close()

		var result struct {
			Delay   int    `json:"delay"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(bodyBytes, &result)

		if resp.StatusCode == http.StatusOK && result.Delay > 0 {
			return NodeDelayResult{
				Delay:      result.Delay,
				Status:     "ok",
				StatusText: "正常",
			}
		}

		errMsg := result.Message
		if errMsg == "" {
			errMsg = string(bodyBytes)
		}
		lastStatus, lastStatusText = classifyDelayError(errMsg, resp.StatusCode)
	}

	return NodeDelayResult{
		Delay:      0,
		Status:     lastStatus,
		StatusText: lastStatusText,
	}
}






func (s *Server) handleProxyDelay(w http.ResponseWriter, r *http.Request) {

	name := strings.TrimSpace(r.URL.Query().Get("name"))

	if name == "" {

		http.Error(w, "missing name parameter", http.StatusBadRequest)

		return

	}

	ir := s.engMgr.GetIR()

	var targetNode *schema.ProxyNode

	var targetIdx int = -1

	for i, p := range ir.Proxies {

		if p.Name == name {

			targetNode = &ir.Proxies[i]

			targetIdx = i

			break

		}

	}

	if targetNode == nil {
		sendJSON(w, map[string]any{
			"ok":          false,
			"name":        name,
			"delay":       0,
			"status":      "not_found",
			"status_text": "节点未找到",
			"error":       "node not found",
		})
		return
	}

	clashPort := 9090

	clashSecret := ""

	if ir.Network.ClashAPIPort > 0 {

		clashPort = ir.Network.ClashAPIPort

	}

	if ir.Network.ClashSecret != "" {

		clashSecret = ir.Network.ClashSecret

	}

	res := s.testNodeRealDelay(*targetNode, clashPort, clashSecret)

	if targetIdx >= 0 {

		ir.Proxies[targetIdx].Delay = res.Delay
		ir.Proxies[targetIdx].DelayStatus = res.Status
		ir.Proxies[targetIdx].DelayText = res.StatusText

		s.engMgr.UpdateIR(ir)

	}

	sendJSON(w, map[string]any{

		"ok": true,

		"name": name,

		"delay": res.Delay,

		"status": res.Status,

		"status_text": res.StatusText,
	})

}





func (s *Server) handleSubSpeedTest(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		ID string `json:"id"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)

	ir := s.engMgr.GetIR()

	type testItem struct {
		index int

		node schema.ProxyNode
	}

	var items []testItem

	for i, p := range ir.Proxies {

		if req.ID != "" && p.SubID != req.ID {

			continue

		}

		if p.Type == "direct" || p.Type == "block" || p.Type == "reject" || p.Server == "" {

			continue

		}

		lower := strings.ToLower(p.Name)

		if strings.Contains(lower, "剩余流量") || strings.Contains(lower, "套餐到期") || strings.Contains(lower, "距离下次重置") || strings.Contains(lower, "官网") || strings.Contains(lower, "续费") {

			continue

		}

		items = append(items, testItem{index: i, node: p})

	}

	if len(items) == 0 {

		sendJSON(w, map[string]any{

			"ok": true,

			"sub_id": req.ID,

			"tested_count": 0,

			"online_count": 0,

			"avg_delay": 0,
		})

		return

	}

	clashPort := 9090

	clashSecret := ""

	if ir.Network.ClashAPIPort > 0 {

		clashPort = ir.Network.ClashAPIPort

	}

	if ir.Network.ClashSecret != "" {

		clashSecret = ir.Network.ClashSecret

	}

	var wg sync.WaitGroup

	// Throttle to 8 concurrent tests to protect router memory

	semaphore := make(chan struct{}, 8)

	var mu sync.Mutex

	onlineCount := 0

	totalDelay := int64(0)

	for _, it := range items {

		wg.Add(1)

		semaphore <- struct{}{}
		item := it
		SafeGo("BatchNodeDelay", func() {
			defer wg.Done()
			defer func() { <-semaphore }()

			res := s.testNodeRealDelay(item.node, clashPort, clashSecret)

			mu.Lock()
			if item.index >= 0 && item.index < len(ir.Proxies) {
				ir.Proxies[item.index].Delay = res.Delay
				ir.Proxies[item.index].DelayStatus = res.Status
				ir.Proxies[item.index].DelayText = res.StatusText
				if res.Delay > 0 {
					onlineCount++
					totalDelay += int64(res.Delay)
				}
			}
			mu.Unlock()
		})

	}

	wg.Wait()

	// Immediate memory reclamation after sub test

	runtime.GC()

	debug.FreeOSMemory()

	avgDelay := 0

	if onlineCount > 0 {

		avgDelay = int(totalDelay / int64(onlineCount))

	}

	s.engMgr.UpdateIR(ir)

	sendJSON(w, map[string]any{

		"ok": true,

		"sub_id": req.ID,

		"tested_count": len(items),

		"online_count": onlineCount,

		"avg_delay": avgDelay,
	})

}





func (s *Server) handleProxiesSpeedTest(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		Names []string `json:"names"`

		Group string `json:"group"`

		SubID string `json:"sub_id"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)

	ir := s.engMgr.GetIR()

	type testItem struct {
		index int

		node schema.ProxyNode
	}

	var items []testItem

	nameSet := make(map[string]bool)

	for _, n := range req.Names {

		nameSet[n] = true

	}

	for i, p := range ir.Proxies {

		if len(req.Names) > 0 && !nameSet[p.Name] {

			continue

		}

		if req.SubID != "" && p.SubID != req.SubID {

			continue

		}

		if p.Type == "direct" || p.Type == "block" || p.Type == "reject" || p.Server == "" {

			continue

		}

		lower := strings.ToLower(p.Name)

		if strings.Contains(lower, "剩余流量") || strings.Contains(lower, "套餐到期") || strings.Contains(lower, "官网") {

			continue

		}

		items = append(items, testItem{index: i, node: p})

	}

	if len(items) == 0 {

		sendJSON(w, map[string]any{

			"ok": true,

			"tested_count": 0,

			"online_count": 0,

			"avg_delay": 0,

			"min_delay": 0,

			"results": map[string]int{},
		})

		return

	}

	clashPort := 9090

	clashSecret := ""

	if ir.Network.ClashAPIPort > 0 {

		clashPort = ir.Network.ClashAPIPort

	}

	if ir.Network.ClashSecret != "" {

		clashSecret = ir.Network.ClashSecret

	}

	var wg sync.WaitGroup

	// Throttle concurrency to 8 to avoid memory explosion and TLS socket exhaustion on routers

	semaphore := make(chan struct{}, 8)

	var mu sync.Mutex

	onlineCount := 0

	totalDelay := int64(0)

	minDelay := 9999

	results := make(map[string]int)
	diagnostics := make(map[string]NodeDelayResult)

	for _, it := range items {

		wg.Add(1)

		semaphore <- struct{}{}
		item := it
		SafeGo("SingleNodeDelay", func() {
			defer wg.Done()
			defer func() { <-semaphore }()

			res := s.testNodeRealDelay(item.node, clashPort, clashSecret)

			mu.Lock()
			if item.index >= 0 && item.index < len(ir.Proxies) {
				ir.Proxies[item.index].Delay = res.Delay
				ir.Proxies[item.index].DelayStatus = res.Status
				ir.Proxies[item.index].DelayText = res.StatusText
			}
			results[item.node.Name] = res.Delay
			diagnostics[item.node.Name] = res
			if res.Delay > 0 {
				onlineCount++
				totalDelay += int64(res.Delay)
				if res.Delay < minDelay {
					minDelay = res.Delay
				}
			}
			mu.Unlock()
		})

	}

	wg.Wait()

	// Immediate memory reclamation after batch node test

	runtime.GC()

	debug.FreeOSMemory()

	avgDelay := 0

	if onlineCount > 0 {

		avgDelay = int(totalDelay / int64(onlineCount))

	} else {

		minDelay = 0

	}

	s.engMgr.UpdateIR(ir)

	if req.Group != "" && ir.Network.ClashAPIPort > 0 {
		grp, port := req.Group, ir.Network.ClashAPIPort
		SafeGo("GroupDelayProbe", func() {
			escaped := url.PathEscape(grp)
			u := fmt.Sprintf("http://127.0.0.1:%d/group/%s/delay?url=https://cp.cloudflare.com/generate_204&timeout=2500", port, escaped)
			client := http.Client{Timeout: 3 * time.Second}
			if resp, err := client.Get(u); err == nil && resp != nil {
				_ = resp.Body.Close()
			}
		})
	}

	sendJSON(w, map[string]any{

		"ok": true,

		"tested_count": len(items),

		"online_count": onlineCount,

		"avg_delay": avgDelay,

		"min_delay": minDelay,

		"results":     results,
		"diagnostics": diagnostics,
	})

}





func (s *Server) handleAdblockStats(w http.ResponseWriter, r *http.Request) {

	stats := s.adblockMgr.GetStats(true, 0)

	sendJSON(w, stats)

}





func (s *Server) handleAdblockConfig(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodGet {

		sendJSON(w, s.adblockMgr.GetFilterConfig())

		return

	}

	if r.Method == http.MethodPost {

		var cfg adblock.FilterConfig

		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {

			http.Error(w, err.Error(), http.StatusBadRequest)

			return

		}

		if err := s.adblockMgr.SaveFilterConfig(cfg); err != nil {

			sendJSON(w, map[string]any{"ok": false, "error": err.Error()})

			return

		}

		s.syncAdblockToEngines()

		sendJSON(w, map[string]any{"ok": true, "config": s.adblockMgr.GetFilterConfig()})

		return

	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

}





func (s *Server) handleAdblockToggle(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		Enabled bool `json:"enabled"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)

	_ = s.adblockMgr.SetFilterEnabled(req.Enabled)

	s.syncAdblockToEngines()

	s.logBuffer.Add("INFO", "ADBLOCK", fmt.Sprintf("域名过滤总开关更新: %v", req.Enabled))

	sendJSON(w, map[string]any{"ok": true, "enabled": req.Enabled})

}





func (s *Server) handleAdblockSourceAdd(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var src adblock.RuleSource

	if err := json.NewDecoder(r.Body).Decode(&src); err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)

		return

	}

	if err := s.adblockMgr.AddRuleSource(src); err != nil {

		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})

		return

	}

	s.logBuffer.Add("INFO", "ADBLOCK", fmt.Sprintf("新增域名过滤规则源: %s (%s)", src.Name, src.URL))

	sendJSON(w, map[string]any{"ok": true, "config": s.adblockMgr.GetFilterConfig()})

}





func (s *Server) handleAdblockSourceUpdate(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var src adblock.RuleSource

	if err := json.NewDecoder(r.Body).Decode(&src); err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)

		return

	}

	if err := s.adblockMgr.UpdateRuleSource(src); err != nil {

		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})

		return

	}

	sendJSON(w, map[string]any{"ok": true, "config": s.adblockMgr.GetFilterConfig()})

}





func (s *Server) handleAdblockSourceDelete(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		ID string `json:"id"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)

	_ = s.adblockMgr.DeleteRuleSource(req.ID)

	sendJSON(w, map[string]any{"ok": true, "config": s.adblockMgr.GetFilterConfig()})

}





func (s *Server) handleAdblockSourceToggle(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		ID string `json:"id"`

		Enabled bool `json:"enabled"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)

	_ = s.adblockMgr.ToggleRuleSource(req.ID, req.Enabled)

	s.syncAdblockToEngines()

	sendJSON(w, map[string]any{"ok": true, "config": s.adblockMgr.GetFilterConfig()})

}





func (s *Server) handleAdblockSourceRefresh(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		ID string `json:"id"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)

	count, err := s.adblockMgr.RefreshRuleSource(req.ID)

	if err != nil {

		sendJSON(w, map[string]any{"ok": false, "error": err.Error(), "rules_count": count})

		return

	}

	s.logBuffer.Add("INFO", "ADBLOCK", fmt.Sprintf("规则源 [%s] 更新完成，当前规则数: %d 条", req.ID, count))

	s.syncAdblockToEngines()

	sendJSON(w, map[string]any{"ok": true, "rules_count": count, "config": s.adblockMgr.GetFilterConfig()})

}





func (s *Server) handleAdblockWhitelistAdd(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		Domain string `json:"domain"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := s.adblockMgr.AddWhitelistDomain(req.Domain); err != nil {

		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})

		return

	}

	s.syncAdblockToEngines()

	s.logBuffer.Add("INFO", "ADBLOCK", fmt.Sprintf("域名过滤新增放行白名单: %s", req.Domain))

	sendJSON(w, map[string]any{"ok": true, "whitelist": s.adblockMgr.GetWhitelistDomains()})

}





func (s *Server) handleAdblockWhitelistDelete(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		Domain string `json:"domain"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)

	_ = s.adblockMgr.DeleteWhitelistDomain(req.Domain)

	s.syncAdblockToEngines()

	s.logBuffer.Add("INFO", "ADBLOCK", fmt.Sprintf("域名过滤移除放行白名单: %s", req.Domain))

	sendJSON(w, map[string]any{"ok": true, "whitelist": s.adblockMgr.GetWhitelistDomains()})

}





func (s *Server) handleDNSRewriteConfig(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodGet {

		sendJSON(w, s.dnsRewriteMgr.GetConfig())

		return

	}

	if r.Method == http.MethodPost {

		var cfg dnsrewrite.Config

		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {

			http.Error(w, err.Error(), http.StatusBadRequest)

			return

		}

		if err := s.dnsRewriteMgr.SaveConfig(cfg); err != nil {

			sendJSON(w, map[string]any{"ok": false, "error": err.Error()})

			return

		}

		s.syncDNSRewriteToEngines()

		sendJSON(w, map[string]any{"ok": true, "config": s.dnsRewriteMgr.GetConfig()})

		return

	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

}





func (s *Server) handleDNSRewriteToggle(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)

		return

	}

	_ = s.dnsRewriteMgr.SetEnabled(req.Enabled)

	s.syncDNSRewriteToEngines()

	stateStr := "已启用"

	if !req.Enabled {

		stateStr = "已关闭"

	}

	s.logBuffer.Add("INFO", "DNS_REWRITE", fmt.Sprintf("DNS重写引擎全局开关: %s", stateStr))

	sendJSON(w, map[string]any{"ok": true, "enabled": req.Enabled, "config": s.dnsRewriteMgr.GetConfig()})

}





func (s *Server) handleDNSRewriteRuleAdd(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req schema.DNSRewriteRule

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)

		return

	}

	rule, err := s.dnsRewriteMgr.AddRule(req.Domain, req.Type, req.Target, req.Comment)

	if err != nil {

		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})

		return

	}

	s.syncDNSRewriteToEngines()

	s.logBuffer.Add("INFO", "DNS_REWRITE", fmt.Sprintf("新增DNS重写规则: %s -> %s (%s)", req.Domain, req.Target, req.Type))

	sendJSON(w, map[string]any{"ok": true, "rule": rule, "config": s.dnsRewriteMgr.GetConfig()})

}





func (s *Server) handleDNSRewriteRuleUpdate(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req schema.DNSRewriteRule

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)

		return

	}

	if err := s.dnsRewriteMgr.UpdateRule(req.ID, req.Domain, req.Type, req.Target, req.Comment, req.Enabled); err != nil {

		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})

		return

	}

	s.syncDNSRewriteToEngines()

	s.logBuffer.Add("INFO", "DNS_REWRITE", fmt.Sprintf("更新DNS重写规则: %s -> %s (%s)", req.Domain, req.Target, req.Type))

	sendJSON(w, map[string]any{"ok": true, "config": s.dnsRewriteMgr.GetConfig()})

}





func (s *Server) handleDNSRewriteRuleDelete(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)

		return

	}

	if err := s.dnsRewriteMgr.DeleteRule(req.ID); err != nil {

		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})

		return

	}

	s.syncDNSRewriteToEngines()

	s.logBuffer.Add("INFO", "DNS_REWRITE", fmt.Sprintf("删除DNS重写规则: ID=%s", req.ID))

	sendJSON(w, map[string]any{"ok": true, "config": s.dnsRewriteMgr.GetConfig()})

}





func (s *Server) handleDNSRewriteRuleToggle(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	var req struct {
		ID string `json:"id"`

		Enabled bool `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)

		return

	}

	if err := s.dnsRewriteMgr.ToggleRule(req.ID, req.Enabled); err != nil {

		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})

		return

	}

	s.syncDNSRewriteToEngines()

	sendJSON(w, map[string]any{"ok": true, "config": s.dnsRewriteMgr.GetConfig()})

}





func (s *Server) handleDNSRewriteReset(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	if err := s.dnsRewriteMgr.ResetDefaults(); err != nil {

		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})

		return

	}

	s.syncDNSRewriteToEngines()

	s.logBuffer.Add("INFO", "DNS_REWRITE", "DNS重写规则已恢复默认设置 (Google Play重写生效)")

	sendJSON(w, map[string]any{"ok": true, "config": s.dnsRewriteMgr.GetConfig()})

}





func (s *Server) handleFakeIPFilter(w http.ResponseWriter, r *http.Request) {
	ir := s.engMgr.GetIR()
	if ir == nil {
		sendJSON(w, map[string]any{"status": "error", "error": "IR not initialized"})
		return
	}

	defaults := []string{
		"+.zspace.com", "+.zspace.cn", "+.zspace.com.cn", "+.zspace.cc",
		"geosite:cn", "+.cn", "*.lan", "*.local", "localhost",
		"+.msftconnecttest.com", "+.msftncsi.com", "+.routeros.com", "+.mikrotik.com",
	}

	if r.Method == http.MethodGet {
		sendJSON(w, map[string]any{
			"status":   "ok",
			"filters":  ir.DNS.FakeIPFilter,
			"defaults": defaults,
		})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Filters []string `json:"filters"`
		Add     string   `json:"add"`
		Remove  string   `json:"remove"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, map[string]any{"status": "error", "error": "invalid json: " + err.Error()})
		return
	}

	currentMap := make(map[string]bool)
	var newFilters []string

	if len(req.Filters) > 0 {
		for _, f := range req.Filters {
			t := strings.TrimSpace(f)
			if t != "" && !currentMap[t] {
				currentMap[t] = true
				newFilters = append(newFilters, t)
			}
		}
	} else {
		for _, f := range ir.DNS.FakeIPFilter {
			t := strings.TrimSpace(f)
			if t != "" && !currentMap[t] {
				currentMap[t] = true
				newFilters = append(newFilters, t)
			}
		}
		if req.Add != "" {
			t := strings.TrimSpace(req.Add)
			if t != "" && !currentMap[t] {
				currentMap[t] = true
				newFilters = append(newFilters, t)
			}
		}
		if req.Remove != "" {
			t := strings.TrimSpace(req.Remove)
			filtered := make([]string, 0, len(newFilters))
			for _, f := range newFilters {
				if f != t {
					filtered = append(filtered, f)
				}
			}
			newFilters = filtered
		}
	}

	ir.DNS.FakeIPFilter = newFilters
	s.engMgr.UpdateIR(ir)
	_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
	s.logBuffer.Add("INFO", "DNS", fmt.Sprintf("Fake-IP 白名单已更新并持久化，当前自定义项: %d", len(newFilters)))

	sendJSON(w, map[string]any{
		"status":   "ok",
		"filters":  ir.DNS.FakeIPFilter,
		"defaults": defaults,
	})
}





func (s *Server) handleRulesTrace(w http.ResponseWriter, r *http.Request) {

	target := strings.TrimSpace(r.URL.Query().Get("target"))

	if target == "" && r.Method == http.MethodPost {

		var req struct {
			Target string `json:"target"`
		}

		_ = json.NewDecoder(r.Body).Decode(&req)

		target = strings.TrimSpace(req.Target)

	}

	if target == "" {

		target = "youku"

	}

	// Clean target (strip protocol, slashes, port)

	cleaned := strings.ToLower(target)

	cleaned = strings.TrimPrefix(cleaned, "http://")

	cleaned = strings.TrimPrefix(cleaned, "https://")

	if idx := strings.Index(cleaned, "/"); idx != -1 {

		cleaned = cleaned[:idx]

	}

	if idx := strings.Index(cleaned, ":"); idx != -1 {

		cleaned = cleaned[:idx]

	}

	ir := s.engMgr.GetIR()

	// Helper to find group's active node

	getActiveNode := func(grpName string) string {

		for _, g := range ir.Groups {

			if strings.EqualFold(g.Name, grpName) {

				if g.Now != "" && !strings.Contains(g.Now, "节点选择") && !strings.Contains(g.Now, "自动优选") {

					return g.Now

				}

				for _, p := range g.Proxies {

					if !strings.Contains(p, "节点选择") && !strings.Contains(p, "自动优选") {

						return p

					}

				}

			}

		}

		return "DIRECT"

	}

	// Domestic keywords for GeoSite:cn

	cnKeywords := []string{

		"youku", "bilibili", "baidu", "taobao", "tmall", "alipay", "jd", "qq", "wechat", "weixin",

		"tencent", "iqiyi", "mgtv", "weibo", "zhihu", "douyin", "bytedance", "kuaishou", "163", "126",

		"netease", "sina", "sohu", "pinduoduo", "xiaomi", "huawei", "oppo", "vivo", "meituan", "eleme",

		"amap", "autonavi", "ctrip", "xunlei", "douyu", "huya", "tieba", "cctv", "smzdm", "csdn",

		"gitee", "aliyun", "bce", "qcloud", "volces", "v2ex", "dianping",
	}

	isCN := false

	if strings.HasSuffix(cleaned, ".cn") || strings.Contains(cleaned, ".cn.") {

		isCN = true

	} else if strings.HasPrefix(cleaned, "192.168.") || strings.HasPrefix(cleaned, "10.") || strings.HasPrefix(cleaned, "172.") || cleaned == "localhost" || strings.HasSuffix(cleaned, ".lan") || strings.HasSuffix(cleaned, ".local") {

		isCN = true

	} else {

		for _, kw := range cnKeywords {

			if strings.Contains(cleaned, kw) {

				isCN = true

				break

			}

		}

	}

	if isCN {

		sendJSON(w, map[string]any{

			"target": target,

			"rule": "GEOSITE, cn / GEOIP, CN, DIRECT (国内直连)",

			"group": "🎯 全球直连",

			"active_node": "🎯 本地直连出站 (DIRECT - 真实公网IP直出)",
		})

		return

	}

	// Foreign business routing matching

	type bizMatch struct {
		Keywords []string

		RuleName string

		GrpName string
	}

	bizRules := []bizMatch{

		{Keywords: []string{"gemini", "antigravity", "aistudio", "deepmind", "notebooklm", "makersuite", "bard", "generativelanguage"}, RuleName: "DOMAIN-SUFFIX, gemini.google.com", GrpName: "Gemini & Antigravity"},

		{Keywords: []string{"openai", "chatgpt", "claude", "anthropic", "copilot"}, RuleName: "GEOSITE, openai", GrpName: "AI"},

		{Keywords: []string{"youtube", "ytimg", "googlevideo"}, RuleName: "GEOSITE, youtube", GrpName: "YouTube"},

		{Keywords: []string{"google", "gmail", "gstatic", "android"}, RuleName: "GEOSITE, google", GrpName: "Google"},

		{Keywords: []string{"github", "ghcr", "docker"}, RuleName: "GEOSITE, github", GrpName: "GitHub & Docker"},

		{Keywords: []string{"telegram", "t.me", "telegra.ph"}, RuleName: "GEOSITE, telegram", GrpName: "Telegram"},

		{Keywords: []string{"twitter", "x.com", "twimg"}, RuleName: "GEOSITE, twitter", GrpName: "Twitter (X)"},

		{Keywords: []string{"netflix", "nflxvideo", "disney", "hbo", "max", "primevideo"}, RuleName: "GEOSITE, netflix", GrpName: "Netflix & 流媒体"},

		{Keywords: []string{"apple", "icloud", "itunes", "aaplimg"}, RuleName: "GEOSITE, apple", GrpName: "Apple 服务"},

		{Keywords: []string{"microsoft", "windows", "office", "azure", "live.com", "msn"}, RuleName: "GEOSITE, microsoft", GrpName: "Microsoft"},

		{Keywords: []string{"spotify"}, RuleName: "GEOSITE, spotify", GrpName: "Spotify"},

		{Keywords: []string{"steam", "steampowered", "steamcommunity"}, RuleName: "GEOSITE, steam", GrpName: "Steam 游戏"},
	}

	for _, bm := range bizRules {

		matched := false

		for _, kw := range bm.Keywords {

			if strings.Contains(kw, ".") {

				if cleaned == kw || strings.HasSuffix(cleaned, "."+kw) {

					matched = true

					break

				}

			} else {

				if strings.Contains(cleaned, kw) {

					matched = true

					break

				}

			}

		}

		if matched {

			sendJSON(w, map[string]any{

				"target": target,

				"rule": fmt.Sprintf("%s, %s", bm.RuleName, bm.GrpName),

				"group": bm.GrpName,

				"active_node": getActiveNode(bm.GrpName),
			})

			return

		}

	}

	// Fallback to 🐟 漏网之鱼

	fishNode := getActiveNode("🐟 漏网之鱼")

	if fishNode == "" {

		fishNode = "DIRECT"

	}

	sendJSON(w, map[string]any{

		"target": target,

		"rule": "MATCH, 🐟 漏网之鱼",

		"group": "🐟 漏网之鱼",

		"active_node": fishNode,
	})

}

// ──────────────────────────────────────────────

// Ring Log Buffer & Fully Desensitized Diagnostic Export

// ──────────────────────────────────────────────





func (s *Server) handleRuleOverrides(w http.ResponseWriter, r *http.Request) {
	if s.userOverrideMgr == nil {
		sendJSON(w, map[string]any{"ok": false, "error": "user override manager not initialized"})
		return
	}
	if r.Method == http.MethodGet {
		spec := s.userOverrideMgr.GetSpec()
		sendJSON(w, map[string]any{
			"ok":   true,
			"data": spec,
		})
		return
	}
	if r.Method == http.MethodPost {
		var spec compiler.UserOverrideSpec
		if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.userOverrideMgr.UpdateSpec(spec); err != nil {
			sendJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		cleanSpec := s.userOverrideMgr.GetSpec()
		if s.ir != nil {
			s.ir.UserOverrides = &schema.UserOverrideRules{
				ForceDirectDomains: cleanSpec.ForceDirectDomains,
				ForceProxyDomains:  cleanSpec.ForceProxyDomains,
				ForceRejectDomains: cleanSpec.ForceRejectDomains,
			}
		}
		SafeGo("UserOverridesSyncReload", func() {
			if s.engMgr != nil {
				_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
			}
		})
		s.logBuffer.Add("INFO", "USER_OVERRIDES", fmt.Sprintf("已原子更新自定义分流覆写规则 (直连:%d, 代理:%d, 屏蔽:%d)",
			len(cleanSpec.ForceDirectDomains), len(cleanSpec.ForceProxyDomains), len(cleanSpec.ForceRejectDomains)))
		sendJSON(w, map[string]any{
			"ok":   true,
			"data": cleanSpec,
		})
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

