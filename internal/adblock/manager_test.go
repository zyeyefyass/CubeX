package adblock

import (
	"fmt"
	"sync"
	"testing"
)

func TestClassifyRule(t *testing.T) {
	tests := []struct {
		rule     string
		expected InterceptCategory
	}{
		{"217heidai (去广告&隐私追踪)", CategoryPrivacy}, // contains 隐私
		{"ads_domain (217heidai)", CategoryAds},
		{"anti-ad_filter", CategoryAds},
		{"google_analytics_tracking", CategoryPrivacy},
		{"telemetry_beacon", CategoryPrivacy},
		{"phishing_and_malware_hijack", CategoryMalicious},
		{"malicious_trojan", CategoryMalicious},
		{"dns_poisoning_hijack", CategoryMalicious},
		{"unknown_rule_default", CategoryAds},
	}

	for _, tc := range tests {
		res := ClassifyRule(tc.rule)
		if res != tc.expected {
			t.Errorf("ClassifyRule(%q) = %v, expected %v", tc.rule, res, tc.expected)
		}
	}
}

func TestCategoryStatsAtomicConsistency(t *testing.T) {
	mgr := NewManager("http://127.0.0.1:9090")

	// Concurrent increments across categories
	var wg sync.WaitGroup
	workers := 20
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				switch (workerID + j) % 3 {
				case 0:
					mgr.RecordBlocked(fmt.Sprintf("ad-%d-%d.com", workerID, j), "10.0.0.50", "ads_filter")
				case 1:
					mgr.RecordBlocked(fmt.Sprintf("track-%d-%d.com", workerID, j), "10.0.0.50", "tracking_telemetry")
				case 2:
					mgr.RecordBlocked(fmt.Sprintf("malware-%d-%d.com", workerID, j), "10.0.0.50", "malicious_phishing")
				}
			}
		}(i)
	}

	wg.Wait()

	stats := mgr.GetStats(true, 0)
	cat := stats.CategoryStats

	expectedTotal := uint64(workers * iterations)
	sumSubCategories := cat.Ads + cat.Privacy + cat.Malicious + cat.Other

	if sumSubCategories != expectedTotal {
		t.Fatalf("Sum of sub-categories %d != expected %d (Ads=%d, Privacy=%d, Malicious=%d, Other=%d)",
			sumSubCategories, expectedTotal, cat.Ads, cat.Privacy, cat.Malicious, cat.Other)
	}

	// 🛡️ 师父防雷补丁 5: 验证 Total 绝对数学自洽
	if cat.Total != sumSubCategories {
		t.Fatalf("Mathematical inconsistency: cat.Total (%d) != sum (%d)", cat.Total, sumSubCategories)
	}

	if stats.BlockedCount != int(cat.Total) {
		t.Fatalf("stats.BlockedCount (%d) != cat.Total (%d)", stats.BlockedCount, cat.Total)
	}
}
