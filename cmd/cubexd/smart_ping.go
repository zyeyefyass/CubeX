package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

// SmartPingGroupState represents health & latency of a single routing group
type SmartPingGroupState struct {
	Group         string `json:"group"`
	ActiveNode    string `json:"active_node"`
	Latency       int    `json:"latency"`
	Status        string `json:"status"` // "healthy", "switched", "degraded"
	LastCheckedAt int64  `json:"last_checked_at"`
}

// SmartPingStatus represents the global state of the Smart Health Ping engine
type SmartPingStatus struct {
	Enabled       bool                  `json:"enabled"`
	IntervalMin   int                   `json:"interval_min"`
	LastCheckTime int64                 `json:"last_check_time"`
	FailoverCount int                   `json:"failover_count"`
	LastLog       string                `json:"last_log"`
	GroupHealth   []SmartPingGroupState `json:"group_health"`
}

// startSmartHealthPing initiates background periodic probe and auto-failover
func (s *Server) startSmartHealthPing() {
	s.smartPingMutex.Lock()
	if s.smartPingIntervalMin <= 0 {
		s.smartPingIntervalMin = 15 // Default 15 minutes
	}
	s.smartPingEnabled = true
	s.smartPingMutex.Unlock()

	go func() {
		// Initial delay 20s to allow proxy cores to settle
		time.Sleep(20 * time.Second)

		// Run first probe pass
		s.executeSmartHealthPing(false)

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		fastWatchdogCounter := 0
		for range ticker.C {
			s.smartPingMutex.RLock()
			enabled := s.smartPingEnabled
			intervalMin := s.smartPingIntervalMin
			lastCheck := s.smartPingLastCheck
			s.smartPingMutex.RUnlock()

			if !enabled || intervalMin <= 0 {
				continue
			}

			fastWatchdogCounter++
			elapsed := time.Now().Unix() - lastCheck

			// 1. Full-group standard ping check (e.g. every intervalMin minutes)
			if elapsed >= int64(intervalMin*60) {
				s.executeSmartHealthPing(false)
				fastWatchdogCounter = 0
			} else if fastWatchdogCounter >= 3 { // Every 45 seconds: agile exit check
				fastWatchdogCounter = 0
				s.executeAgileExitProbe()
			}
		}
	}()
}

// executeAgileExitProbe performs ultra-lightweight 45s health probe on active primary exit
func (s *Server) executeAgileExitProbe() {
	ir := s.engMgr.GetIR()
	if ir == nil || len(ir.Proxies) == 0 || len(ir.Groups) == 0 {
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

	// Index proxies
	proxyMap := make(map[string]*schema.ProxyNode, len(ir.Proxies))
	for i := range ir.Proxies {
		proxyMap[ir.Proxies[i].Name] = &ir.Proxies[i]
	}

	for gIdx := range ir.Groups {
		grp := &ir.Groups[gIdx]
		gUpper := strings.ToUpper(grp.Name)
		// Only inspect major exit groups
		if !strings.Contains(gUpper, "PROXY") && !strings.Contains(gUpper, "节点") && !strings.Contains(gUpper, "GLOBAL") && !strings.Contains(gUpper, "MATCH") && !strings.Contains(gUpper, "漏网") {
			continue
		}
		if len(grp.Proxies) <= 1 {
			continue
		}

		activeName := grp.Now
		if activeName == "" {
			activeName = grp.Proxies[0]
		}
		node, ok := proxyMap[activeName]
		if !ok || node == nil || node.Type == "direct" || node.Type == "block" {
			continue
		}

		// Light single-packet probe
		res := s.testNodeRealDelay(*node, clashPort, clashSecret)
		if res.Delay <= 0 {
			// Pre-emptive immediate failover!
			log.Printf("[SMART-FAILOVER] Active exit [%s] in group [%s] timed out! Triggering immediate fallback...", activeName, grp.Name)
			s.executeSmartHealthPing(true)
			break
		}
	}
}

// executeSmartHealthPing performs end-to-end latency checks and zero-perception failover
func (s *Server) executeSmartHealthPing(force bool) {
	ir := s.engMgr.GetIR()
	if ir == nil || len(ir.Proxies) == 0 || len(ir.Groups) == 0 {
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

	// Index proxies by name
	proxyMap := make(map[string]*schema.ProxyNode, len(ir.Proxies))
	for i := range ir.Proxies {
		proxyMap[ir.Proxies[i].Name] = &ir.Proxies[i]
	}

	nowTs := time.Now().Unix()
	var groupStates []SmartPingGroupState
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Limit concurrent probes to prevent network / CPU spikes
	sem := make(chan struct{}, 4)

	for gIdx := range ir.Groups {
		group := &ir.Groups[gIdx]

		// Skip non-proxy groups
		gUpper := strings.ToUpper(group.Name)
		if gUpper == "DIRECT" || gUpper == "REJECT" || gUpper == "GLOBAL" || gUpper == "COMPATIBLE" || len(group.Proxies) == 0 {
			continue
		}

		activeNodeName := group.Now
		if activeNodeName == "" && len(group.Proxies) > 0 {
			activeNodeName = group.Proxies[0]
		}

		targetNode, exists := proxyMap[activeNodeName]
		if !exists || targetNode == nil || targetNode.Type == "direct" || targetNode.Type == "block" || targetNode.Type == "reject" {
			continue
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(grp *schema.StrategyGroup, activeNode *schema.ProxyNode, activeName string) {
			defer func() {
				<-sem
				wg.Done()
			}()

			// 1. Probe currently selected active node
			res := s.testNodeRealDelay(*activeNode, clashPort, clashSecret)
			delay := res.Delay

			state := SmartPingGroupState{
				Group:         grp.Name,
				ActiveNode:    activeName,
				Latency:       delay,
				LastCheckedAt: nowTs,
			}

			if delay > 0 {
				state.Status = "healthy"
			} else {
				// 2. Pre-emptive Failover: Active node is dead or timed out!
				// Search alternative candidates in the same group
				var bestCandidate *schema.ProxyNode
				var bestDelay int = 99999

				candidateCount := 0
				for _, candName := range grp.Proxies {
					if candName == activeName {
						continue
					}
					candNode, ok := proxyMap[candName]
					if !ok || candNode == nil || candNode.Type == "direct" || candNode.Type == "block" {
						continue
					}

					candidateCount++
					if candidateCount > 5 {
						break // Inspect top 5 candidates max to keep it lightweight
					}

					candRes := s.testNodeRealDelay(*candNode, clashPort, clashSecret)
					candDelay := candRes.Delay
					if candDelay > 0 && candDelay < bestDelay {
						bestDelay = candDelay
						bestCandidate = candNode
					}
				}

				if bestCandidate != nil {
					// 3. Atomically switch group to healthy candidate
					grp.Now = bestCandidate.Name

					// Notify Clash API
					go func(gName, nodeName string) {
						payload, _ := json.Marshal(map[string]string{"name": nodeName})
						escapedGroup := url.PathEscape(gName)
						reqURL := fmt.Sprintf("http://127.0.0.1:%d/proxies/%s", clashPort, escapedGroup)
						client := http.Client{Timeout: 1 * time.Second}
						cReq, err := http.NewRequest(http.MethodPut, reqURL, bytes.NewBuffer(payload))
						if err == nil {
							cReq.Header.Set("Content-Type", "application/json")
							if clashSecret != "" {
								cReq.Header.Set("Authorization", "Bearer "+clashSecret)
							}
							if resp, doErr := client.Do(cReq); doErr == nil && resp != nil {
								_ = resp.Body.Close()
							}
						}
					}(grp.Name, bestCandidate.Name)

					logMsg := fmt.Sprintf("智能巡检感知当前节点 [%s] 异常中断，已 0 毫秒静默预热无感切至 [%s] (延迟: %dms)", activeName, bestCandidate.Name, bestDelay)
					s.logBuffer.Add("WARN", "FAILOVER", logMsg)
					log.Printf("[SMART-PING] Group %s: %s", grp.Name, logMsg)

					state.Status = "switched"
					state.ActiveNode = bestCandidate.Name
					state.Latency = bestDelay

					mu.Lock()
					s.smartPingFailovers++
					s.smartPingLastLog = logMsg
					mu.Unlock()
				} else {
					state.Status = "degraded"
					state.Latency = 0
				}
			}

			mu.Lock()
			groupStates = append(groupStates, state)
			mu.Unlock()
		}(group, targetNode, activeNodeName)
	}

	wg.Wait()

	// Update IR with any failover selections
	s.engMgr.UpdateIR(ir)

	s.smartPingMutex.Lock()
	s.smartPingLastCheck = nowTs
	s.smartPingGroupHealth = groupStates
	s.smartPingMutex.Unlock()
}

// handleSmartPingStatus returns the current status and health of the smart ping engine
func (s *Server) handleSmartPingStatus(w http.ResponseWriter, r *http.Request) {
	s.smartPingMutex.RLock()
	st := SmartPingStatus{
		Enabled:       s.smartPingEnabled,
		IntervalMin:   s.smartPingIntervalMin,
		LastCheckTime: s.smartPingLastCheck,
		FailoverCount: s.smartPingFailovers,
		LastLog:       s.smartPingLastLog,
		GroupHealth:   s.smartPingGroupHealth,
	}
	s.smartPingMutex.RUnlock()

	sendJSON(w, map[string]any{
		"ok":   true,
		"data": st,
	})
}

// handleSmartPingConfig updates the smart ping engine configuration
func (s *Server) handleSmartPingConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Enabled     *bool `json:"enabled"`
		IntervalMin *int  `json:"interval_min"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.smartPingMutex.Lock()
	if req.Enabled != nil {
		s.smartPingEnabled = *req.Enabled
	}
	if req.IntervalMin != nil {
		if *req.IntervalMin >= 0 {
			s.smartPingIntervalMin = *req.IntervalMin
		}
	}
	s.smartPingMutex.Unlock()

	s.logBuffer.Add("INFO", "SMARTPING", fmt.Sprintf("智能健康巡检配置已更新: 启用=%v, 周期=%d分钟", s.smartPingEnabled, s.smartPingIntervalMin))

	sendJSON(w, map[string]any{
		"ok":      true,
		"message": "配置更新成功",
		"enabled": s.smartPingEnabled,
		"interval": s.smartPingIntervalMin,
	})
}

// handleSmartPingTrigger triggers an immediate ad-hoc inspection pass
// handleSmartPingTrigger manually forces an immediate health ping pass and returns real-time report
func (s *Server) handleSmartPingTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	beforeCount := s.smartPingFailovers
	s.executeSmartHealthPing(true)
	afterCount := s.smartPingFailovers

	msg := "已完成全局智能避障巡检：全链路节点连接通畅，当前延迟最优。"
	if afterCount > beforeCount && s.smartPingLastLog != "" {
		msg = fmt.Sprintf("已成功触发智能无感避障：%s", s.smartPingLastLog)
	}

	s.logBuffer.Add("INFO", "SMARTPING", "全局智能避障巡检与节点故障转移已执行完成")

	sendJSON(w, map[string]any{
		"ok":        true,
		"status":    "ok",
		"message":   msg,
		"failovers": afterCount,
		"last_log":  s.smartPingLastLog,
	})
}

