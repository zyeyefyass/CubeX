package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSelectTieredBatch100Nodes(t *testing.T) {
	ewma := NewEWMAManager()
	sched := NewEWMAScheduler(SchedulerConfig{
		HotStandbyCount: 3,
		ColdBatchSize:   2,
	}, ewma, nil)

	// Build 100 nodes
	candidates := make([]string, 100)
	for i := 0; i < 100; i++ {
		candidates[i] = fmt.Sprintf("Node-%03d", i)
		// Give some nodes EWMA scores
		if i < 10 {
			ewma.RecordSample(candidates[i], 50+i*5, true)
		}
	}

	active := "Node-000"

	// Round 1
	batch1 := sched.SelectTieredBatch("ProxyGroup", active, candidates)
	// Expect active (1) + hot (top 3) + cold (2) <= 6
	if len(batch1) > 6 {
		t.Fatalf("batch1 size %d > 6", len(batch1))
	}
	if batch1[0] != active {
		t.Errorf("batch1 active sentinel mismatch: got %s, want %s", batch1[0], active)
	}

	// Verify cursor rotation over multiple rounds
	seenCold := make(map[string]bool)
	for round := 0; round < 50; round++ {
		batch := sched.SelectTieredBatch("ProxyGroup", active, candidates)
		for _, n := range batch {
			seenCold[n] = true
		}
	}

	// Over 50 rounds, 50 * 2 = 100 cold candidates should all have been touched
	if len(seenCold) < 80 {
		t.Errorf("expected at least 80 unique nodes sampled across 50 rounds, got %d", len(seenCold))
	}
}

func TestDualModeFailoverAndHysteresis(t *testing.T) {
	ewma := NewEWMAManager()
	var switchedEvents []SwitchEvent
	sched := NewEWMAScheduler(SchedulerConfig{
		SwitchMarginMs:    25,
		SwitchMarginRatio: 0.18,
		SwitchCooldown:    60 * time.Second,
	}, ewma, func(ev SwitchEvent) {
		switchedEvents = append(switchedEvents, ev)
	})

	ctx := context.Background()

	// Scenario 1: Active node is 80ms, Candidate is 72ms (only 8ms diff < 25ms margin)
	// Hysteresis must firmly maintain stability without switching
	ewma.RecordSample("Node-Active", 80, true)
	ewma.RecordSample("Node-Cand", 72, true)

	sched.EvaluateAndSwitchGroup(ctx, "ProxyGroup", "Node-Active", []string{"Node-Active", "Node-Cand"})
	if len(switchedEvents) > 0 {
		t.Fatalf("expected 0 switches for minor 8ms advantage, got %d", len(switchedEvents))
	}

	// Scenario 2: Active node suffers 2 consecutive timeouts (Emergency Fast-Failover)
	ewma.RecordSample("Node-Active", 0, false)
	ewma.RecordSample("Node-Active", 0, false)

	sched.EvaluateAndSwitchGroup(ctx, "ProxyGroup", "Node-Active", []string{"Node-Active", "Node-Cand"})
	if len(switchedEvents) != 1 {
		t.Fatalf("expected 1 emergency fast-failover, got %d", len(switchedEvents))
	}
	if !switchedEvents[0].Emergency {
		t.Errorf("expected switch event to be Emergency=true")
	}
	if switchedEvents[0].ToNode != "Node-Cand" {
		t.Errorf("expected switch to Node-Cand, got %s", switchedEvents[0].ToNode)
	}
}

func TestPenalizeAndRotateActiveDecay(t *testing.T) {
	ewma := NewEWMAManager()
	sched := NewEWMAScheduler(SchedulerConfig{}, ewma, nil)

	node := "Test-Node-HK-01"
	ewma.RecordSample(node, 50, true)
	ewma.RecordSample(node, 0, false)
	ewma.RecordSample(node, 0, false)

	snap, _ := ewma.GetNodeSnapshot(node)
	if snap.ConsecutiveFails != 2 {
		t.Fatalf("expected consecutive fails = 2, got %d", snap.ConsecutiveFails)
	}

	// 1. Manually set penalty with 300s expiration
	sched.SetNodePenalizedForTest(node, time.Now().Add(300*time.Second))
	if !sched.IsNodePenalized(node) {
		t.Errorf("expected node to be penalized within 300s window")
	}

	// 2. Set expired penalty (past timestamp), verify automatic decay and candidacy restoration
	sched.SetNodePenalizedForTest(node, time.Now().Add(-1*time.Second))
	if sched.IsNodePenalized(node) {
		t.Errorf("expected expired penalty to decay automatically")
	}

	// Verify EWMA candidacy restored: ConsecutiveFails cleared to 0
	restored, ok := ewma.GetNodeSnapshot(node)
	if !ok || restored.ConsecutiveFails != 0 {
		t.Errorf("expected EWMA ConsecutiveFails reset to 0 after decay, got %d", restored.ConsecutiveFails)
	}
}

func TestEWMAScheduler_BareHostPortAndErrorPropagation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/proxies" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"proxies": map[string]any{
					"Proxy": map[string]any{"name": "Proxy", "type": "Selector", "now": "N1", "all": []string{"N1", "N2"}},
				},
			})
			return
		}
		if r.Method == http.MethodPut && r.URL.Path == "/proxies/Proxy" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	sched := NewEWMAScheduler(SchedulerConfig{
		ClashAPIBase: strings.TrimPrefix(srv.URL, "http://"),
	}, NewEWMAManager(), nil)

	if !strings.HasPrefix(sched.cfg.ClashAPIBase, "http://") {
		t.Fatalf("expected normalized http:// ClashAPIBase, got %q", sched.cfg.ClashAPIBase)
	}
	proxies, err := sched.fetchProxiesSnapshot(context.Background())
	if err != nil || len(proxies) != 1 {
		t.Fatalf("expected fetchProxiesSnapshot success with bare host:port, got err=%v", err)
	}
	if err := sched.switchClashGroup(context.Background(), "Proxy", "N2"); err != nil {
		t.Fatalf("expected switchClashGroup success with bare host:port, got err=%v", err)
	}

	srvErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srvErr.Close()
	schedErr := NewEWMAScheduler(SchedulerConfig{ClashAPIBase: srvErr.URL}, NewEWMAManager(), nil)
	if _, err := schedErr.fetchProxiesSnapshot(context.Background()); err == nil {
		t.Fatalf("expected explicit error on HTTP 500 fetchProxiesSnapshot")
	}
	if err := schedErr.switchClashGroup(context.Background(), "Proxy", "N2"); err == nil {
		t.Fatalf("expected explicit error on HTTP 500 switchClashGroup")
	}
}

func TestEWMAScheduler_NeverTouchesDomesticOrManualSelectorOrDisasterGroups(t *testing.T) {
	var mu sync.Mutex
	putCount := make(map[string]int)
	latestSelection := map[string]string{
		"🎯 国内流量":  "DIRECT",
		"Google":   "Slow-TW",
		"🌐 国外流量":  "Slow-HK",
		"🇯🇵 日本容灾": "Slow-JP",
		"🇭🇰 香港自动": "Slow-HK",
		"🇭🇰 香港故转": "Slow-HK",
		"🇸🇬 新加坡故转": "Slow-SG",
	}

	delays := map[string]int{
		"Fast-HK": 18,
		"Slow-HK": 180,
		"Slow-TW": 210,
		"Slow-JP": 190,
		"Slow-SG": 170,
		"Fast-SG": 22,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/proxies" {
			mu.Lock()
			snap := map[string]any{
				"Fast-HK": map[string]any{"name": "Fast-HK", "type": "Shadowsocks"},
				"Slow-HK": map[string]any{"name": "Slow-HK", "type": "Shadowsocks"},
				"Slow-TW": map[string]any{"name": "Slow-TW", "type": "Shadowsocks"},
				"Slow-JP": map[string]any{"name": "Slow-JP", "type": "Shadowsocks"},
				"Slow-SG": map[string]any{"name": "Slow-SG", "type": "Shadowsocks"},
				"Fast-SG": map[string]any{"name": "Fast-SG", "type": "Shadowsocks"},
				"🎯 国内流量": map[string]any{
					"name": "🎯 国内流量", "type": "Selector",
					"now": latestSelection["🎯 国内流量"],
					"all": []string{"Fast-HK", "DIRECT"},
				},
				"Google": map[string]any{
					"name": "Google", "type": "Selector",
					"now": latestSelection["Google"],
					"all": []string{"Slow-TW", "Fast-HK", "DIRECT"},
				},
				"🌐 国外流量": map[string]any{
					"name": "🌐 国外流量", "type": "Selector",
					"now": latestSelection["🌐 国外流量"],
					"all": []string{"Slow-HK", "Fast-HK", "DIRECT"},
				},
				"🇯🇵 日本容灾": map[string]any{
					"name": "🇯🇵 日本容灾", "type": "Fallback",
					"now": latestSelection["🇯🇵 日本容灾"],
					"all": []string{"Slow-JP", "Fast-HK", "DIRECT"},
				},
				"🇭🇰 香港自动": map[string]any{
					"name": "🇭🇰 香港自动", "type": "URLTest",
					"now": latestSelection["🇭🇰 香港自动"],
					"all": []string{"Slow-HK", "Fast-HK"},
				},
				"🇭🇰 香港故转": map[string]any{
					"name": "🇭🇰 香港故转", "type": "Fallback",
					"now": latestSelection["🇭🇰 香港故转"],
					"all": []string{"Slow-HK", "Fast-HK"},
				},
				// 模拟 sing-box 将 fallback 组编译为 Selector 的双核场景
				"🇸🇬 新加坡故转": map[string]any{
					"name": "🇸🇬 新加坡故转", "type": "Selector",
					"now": latestSelection["🇸🇬 新加坡故转"],
					"all": []string{"Slow-SG", "Fast-SG"},
				},
			}
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"proxies": snap})
			return
		}
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/proxies/") && strings.HasSuffix(r.URL.Path, "/delay") {
			node := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/proxies/"), "/delay")
			if d, ok := delays[node]; ok {
				_ = json.NewEncoder(w).Encode(map[string]any{"delay": d})
				return
			}
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		if r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/proxies/") {
			grp := strings.TrimPrefix(r.URL.Path, "/proxies/")
			var body struct {
				Name string `json:"name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			mu.Lock()
			putCount[grp]++
			if body.Name != "" {
				latestSelection[grp] = body.Name
			}
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	// 子场景 1：ManagedGroups 为空（自动探测模式）
	schedAuto := NewEWMAScheduler(SchedulerConfig{
		ClashAPIBase:      srv.URL,
		SwitchMarginMs:    25,
		SwitchMarginRatio: 0.18,
		SwitchCooldown:    time.Millisecond,
	}, NewEWMAManager(), nil)

	for i := 0; i < 4; i++ {
		schedAuto.RunSingleCycle(context.Background())
	}

	// 子场景 2：显式白名单（含故意夹带 🎯 国内流量 / Google / 🇯🇵 日本容灾 的对抗用例）
	schedExplicit := NewEWMAScheduler(SchedulerConfig{
		ClashAPIBase:      srv.URL,
		SwitchMarginMs:    25,
		SwitchMarginRatio: 0.18,
		SwitchCooldown:    time.Millisecond,
		ManagedGroups: []string{
			"🎯 国内流量",
			"Google",
			"🌐 国外流量",
			"🇯🇵 日本容灾",
			"🇭🇰 香港自动",
			"🇭🇰 香港故转",
			"🇸🇬 新加坡故转",
		},
	}, NewEWMAManager(), nil)

	for i := 0; i < 4; i++ {
		schedExplicit.RunSingleCycle(context.Background())
	}

	// 子场景 3：黑洞惩罚轮换也不能触碰非托管组
	schedExplicit.PenalizeAndRotateActive("test-blackhole")

	mu.Lock()
	defer mu.Unlock()

	forbiddenGroups := []string{"🎯 国内流量", "Google", "🌐 国外流量", "🇯🇵 日本容灾"}
	for _, g := range forbiddenGroups {
		if putCount[g] != 0 {
			t.Fatalf("EWMA must NEVER switch forbidden group %q, but observed %d PUT(s)", g, putCount[g])
		}
	}
	if putCount["🇭🇰 香港自动"] == 0 {
		t.Errorf("expected URLTest group 🇭🇰 香港自动 to be optimized by EWMA, got 0 PUTs")
	}
	if putCount["🇭🇰 香港故转"] == 0 {
		t.Errorf("expected Fallback group 🇭🇰 香港故转 to be optimized by EWMA, got 0 PUTs")
	}
	if putCount["🇸🇬 新加坡故转"] == 0 {
		t.Errorf("expected sing-box Selector-backed 🇸🇬 新加坡故转 to be optimized by EWMA, got 0 PUTs")
	}
}
