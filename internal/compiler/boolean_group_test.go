package compiler

import (
	"testing"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

func TestFilterNodesByBooleanKeywords(t *testing.T) {
	nodes := []schema.ProxyNode{
		{Name: "🇭🇰 香港 01 [自建] 专线"},
		{Name: "🇭🇰 香港 02 普通 BGP"},
		{Name: "🇯🇵 日本 01 [自建] 软银"},
		{Name: "🇺🇸 美国 01 普通 10G"},
		{Name: "DIRECT"},
	}

	// 1. Mode ALL
	all := FilterNodesByBooleanKeywords(nodes, "all", nil)
	if len(all) != 4 {
		t.Fatalf("expected 4 nodes for ALL, got %d", len(all))
	}

	// 2. Mode OR: "香港", "日本"
	orNodes := FilterNodesByBooleanKeywords(nodes, "or", []string{"香港", "日本"})
	if len(orNodes) != 3 {
		t.Fatalf("expected 3 nodes for OR, got %d: %v", len(orNodes), orNodes)
	}

	// 3. Mode AND: "自建", "香港"
	andNodes := FilterNodesByBooleanKeywords(nodes, "and", []string{"自建", "香港"})
	if len(andNodes) != 1 || andNodes[0] != "🇭🇰 香港 01 [自建] 专线" {
		t.Fatalf("expected 1 node for AND, got %d: %v", len(andNodes), andNodes)
	}

	// 4. Preview
	preview := PreviewBooleanFilter(nodes, "and", []string{"自建", "香港"})
	if preview.MatchedCount != 1 || len(preview.ExcludedNodes) != 3 {
		t.Fatalf("preview mismatch: matched=%d, excluded=%d", preview.MatchedCount, len(preview.ExcludedNodes))
	}
}
