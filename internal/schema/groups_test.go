package schema

import (
	"testing"
)

func TestRegionAndBusinessGroupsSync(t *testing.T) {
	ir := NewDefaultIR()

	// Inject sample proxy nodes representing different regions
	ir.Proxies = []ProxyNode{
		{Name: "DIRECT", Type: "direct"},
		{Name: "REJECT", Type: "block"},
		{Name: "🇭🇰 香港 01 [BGP]", Type: "ss", Server: "1.1.1.1", Port: 8388},
		{Name: "🇭🇰 香港 02 [CN2 GIA]", Type: "ss", Server: "1.1.1.2", Port: 8388},
		{Name: "🇹🇼 台湾 01 [Hinet]", Type: "ss", Server: "1.1.1.3", Port: 8388},
		{Name: "🇸🇬 新加坡 01 [AWS]", Type: "ss", Server: "1.1.1.4", Port: 8388},
		{Name: "[HHost-JP] 🇯🇵 cloud xtls-reality", Type: "vless", Server: "1.1.1.5", Port: 8388},
		{Name: "[Mtice] 🇯🇵 日本东京 01", Type: "ss", Server: "1.1.1.6", Port: 8388},
		{Name: "🇰🇷 韩国首尔 01", Type: "ss", Server: "1.1.1.7", Port: 8388},
		{Name: "🇺🇸 美国洛杉矶 01", Type: "ss", Server: "1.1.1.8", Port: 8388},
		{Name: "🇬🇧 英国伦敦 01", Type: "ss", Server: "1.1.1.9", Port: 8388},
		{Name: "🇩🇪 德国法兰克福 01", Type: "ss", Server: "1.1.1.10", Port: 8388},
	}

	SyncAllStrategyGroups(ir)

	// Check that 24 regional groups (8 regions * 3 policies) + standard business groups are created
	if len(ir.Groups) < 36 {
		t.Fatalf("expected at least 36 groups (24 regional + 12 business), got %d", len(ir.Groups))
	}
	disasterCount := 0
	for _, g := range ir.Groups {
		if g.Policy == "disaster" {
			disasterCount++
		}
	}
	if disasterCount != 8 {
		t.Fatalf("expected 8 regional disaster groups, got %d", disasterCount)
	}

	groupMap := make(map[string]StrategyGroup)
	for _, g := range ir.Groups {
		groupMap[g.Name] = g
	}

	// Verify Hong Kong region: manual deleted, auto -> fall -> disaster added
	if _, ok := groupMap["🇭🇰 香港手动"]; ok {
		t.Fatalf("unexpected '🇭🇰 香港手动' group, should be removed")
	}

	hkAuto, ok := groupMap["🇭🇰 香港自动"]
	if !ok {
		t.Fatalf("missing '🇭🇰 香港自动' group")
	}
	if len(hkAuto.Proxies) != 2 {
		t.Errorf("expected 2 HK proxies in '🇭🇰 香港自动', got %d", len(hkAuto.Proxies))
	}

	hkFall, ok := groupMap["🇭🇰 香港故转"]
	if !ok {
		t.Fatalf("missing '🇭🇰 香港故转' group")
	}
	if len(hkFall.Proxies) != 2 {
		t.Errorf("expected 2 HK proxies in '🇭🇰 香港故转', got %d", len(hkFall.Proxies))
	}

	hkDisaster, ok := groupMap["🇭🇰 香港容灾"]
	if !ok {
		t.Fatalf("missing '🇭🇰 香港容灾' group")
	}
	// HK Disaster contains nodes + DIRECT
	if len(hkDisaster.Proxies) < 2 {
		t.Errorf("expected at least 2 HK proxies in '🇭🇰 香港容灾', got %d", len(hkDisaster.Proxies))
	}

	// Verify Taiwan region groups
	twAuto, ok := groupMap["🇹🇼 台湾自动"]
	if !ok {
		t.Fatalf("missing '🇹🇼 台湾自动' group")
	}
	if len(twAuto.Proxies) != 1 {
		t.Errorf("expected 1 TW proxy, got %d", len(twAuto.Proxies))
	}

	// Verify Japan region groups
	jpAuto, ok := groupMap["🇯🇵 日本自动"]
	if !ok {
		t.Fatalf("missing '🇯🇵 日本自动' group")
	}
	if len(jpAuto.Proxies) != 2 {
		t.Errorf("expected 2 JP proxies in '🇯🇵 日本自动', got %d: %v", len(jpAuto.Proxies), jpAuto.Proxies)
	}

	// Verify Korea region groups
	krAuto, ok := groupMap["🇰🇷 韩国自动"]
	if !ok {
		t.Fatalf("missing '🇰🇷 韩国自动' group")
	}
	if len(krAuto.Proxies) != 1 {
		t.Errorf("expected 1 KR proxy in '🇰🇷 韩国自动', got %d: %v", len(krAuto.Proxies), krAuto.Proxies)
	}

	// Verify Singapore region groups
	sgAuto, ok := groupMap["🇸🇬 新加坡自动"]
	if !ok {
		t.Fatalf("missing '🇸🇬 新加坡自动' group")
	}
	if len(sgAuto.Proxies) != 1 {
		t.Errorf("expected 1 SG proxy, got %d", len(sgAuto.Proxies))
	}

	// Verify US region groups
	usAuto, ok := groupMap["🇺🇸 美国自动"]
	if !ok {
		t.Fatalf("missing '🇺🇸 美国自动' group")
	}
	if len(usAuto.Proxies) != 1 {
		t.Errorf("expected 1 US proxy, got %d", len(usAuto.Proxies))
	}

	// Verify Others region groups (UK, Germany)
	otherAuto, ok := groupMap["🌐 其它自动"]
	if !ok {
		t.Fatalf("missing '🌐 其它自动' group")
	}
	if len(otherAuto.Proxies) != 2 {
		t.Errorf("expected 2 Others proxies, got %d", len(otherAuto.Proxies))
	}

	// Verify Business Groups exist
	expectedBiz := []string{
		"🎯 国内流量",
		"🌐 国外流量",
		"AI",
		"Gemini & Antigravity",
		"YouTube",
		"Google",
		"GitHub & Docker",
		"Telegram",
		"Twitter (X)",
		"Netflix & 流媒体",
		"Apple 服务",
		"Microsoft",
		"Spotify",
		"Steam 游戏",
		"🐟 漏网之鱼",
	}

	for _, bName := range expectedBiz {
		if _, exists := groupMap[bName]; !exists {
			t.Errorf("expected business group %s to exist", bName)
		}
	}

	// Verify Region Statistics
	stats := ExtractRegionStats(ir)
	if len(stats) != len(StandardRegions) {
		t.Fatalf("expected %d region stats, got %d", len(StandardRegions), len(stats))
	}
}
