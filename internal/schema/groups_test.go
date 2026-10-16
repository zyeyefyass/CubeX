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
		{Name: "🇯🇵 日本东京 01", Type: "ss", Server: "1.1.1.5", Port: 8388},
		{Name: "🇰🇷 韩国首尔 01", Type: "ss", Server: "1.1.1.6", Port: 8388},
		{Name: "🇺🇸 美国洛杉矶 01", Type: "ss", Server: "1.1.1.7", Port: 8388},
		{Name: "🇬🇧 英国伦敦 01", Type: "ss", Server: "1.1.1.8", Port: 8388},
		{Name: "🇩🇪 德国法兰克福 01", Type: "ss", Server: "1.1.1.9", Port: 8388},
	}

	SyncAllStrategyGroups(ir)

	// Check that 18 regional groups + standard business groups are created
	if len(ir.Groups) < 29 {
		t.Fatalf("expected at least 29 groups (18 regional + 11 business), got %d", len(ir.Groups))
	}

	groupMap := make(map[string]StrategyGroup)
	for _, g := range ir.Groups {
		groupMap[g.Name] = g
	}

	// Verify Hong Kong region groups
	hkAuto, ok := groupMap["🇭🇰 香港自动"]
	if !ok {
		t.Fatalf("missing '🇭🇰 香港自动' group")
	}
	if len(hkAuto.Proxies) != 2 {
		t.Errorf("expected 2 HK proxies in '🇭🇰 香港自动', got %d", len(hkAuto.Proxies))
	}

	// Verify Taiwan region groups
	twAuto, ok := groupMap["🇹🇼 台湾自动"]
	if !ok {
		t.Fatalf("missing '🇹🇼 台湾自动' group")
	}
	if len(twAuto.Proxies) != 1 {
		t.Errorf("expected 1 TW proxy, got %d", len(twAuto.Proxies))
	}

	// Verify Singapore region groups
	sgAuto, ok := groupMap["🇸🇬 新加坡自动"]
	if !ok {
		t.Fatalf("missing '🇸🇬 新加坡自动' group")
	}
	if len(sgAuto.Proxies) != 1 {
		t.Errorf("expected 1 SG proxy, got %d", len(sgAuto.Proxies))
	}

	// Verify Japan & Korea region groups
	jpkrAuto, ok := groupMap["🇯🇵🇰🇷 日韩自动"]
	if !ok {
		t.Fatalf("missing '🇯🇵🇰🇷 日韩自动' group")
	}
	if len(jpkrAuto.Proxies) != 2 {
		t.Errorf("expected 2 JPKR proxies, got %d", len(jpkrAuto.Proxies))
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
		"🤖 OpenAI&AI",
		"📹 YouTube",
		"🍀 Google",
		"👨‍💻 GitHub&docker",
		"📲 Telegram",
		"🐦 Twitter(X)",
		"🎬 Netflix&流媒体",
		"🍏 Apple服务",
		"🪟 Microsoft",
		"🚀 国外",
		"🐟 漏网之鱼",
		"🎯 全球直连",
	}

	for _, bName := range expectedBiz {
		if _, exists := groupMap[bName]; !exists {
			t.Errorf("expected business group %s to exist", bName)
		}
	}

	// Verify Region Statistics
	stats := ExtractRegionStats(ir)
	if len(stats) != 6 {
		t.Fatalf("expected 6 region stats, got %d", len(stats))
	}
}
